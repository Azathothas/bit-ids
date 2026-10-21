// deluge - what the `deluge` adapter decides, on either platform, as functions
// with no process in them, so any host can ask them.
//
// CLIENT-07. The adapter is two files that build for one platform each,
// adapter_deluge.go off Windows and adapter_deluge_windows.go on it; everything
// here is the part of them a Linux gate can prove: the settings document the
// daemon reads, how the build's answer is read, the commands its own console is
// handed, what pip and winget said they installed, where each route leaves the
// build, and the scrub of the credential the daemon writes. `bit-check deluge
// --selftest` asks each of them.
//
// -- the routes, measured on 2026-10-16 ----------------------------------------
//
// Deluge's newest stable is 2.2.0, published on 2025-04-28. Its GitHub
// repository carries tags and no release objects, so no `release` route can be
// resolved through `resolve-release`; the vendor's own artifacts are a source
// tarball and four Windows installers on download.deluge-torrent.org
// (ftp.osuosl.org), each with a `.sha256`, and a wheel and an sdist on PyPI.
//
//	platform  route    what it installs
//	Linux     package  PyPI's `deluge`, unpinned, as a wheel, into a venv
//	Linux     source   the `deluge-2.2.0` tag, cloned and pip-installed into a venv
//	Windows   package  winget's `DelugeTeam.Deluge`, the vendor's win64 installer
//	Windows   source   the same tag, cloned and pip-installed on the runner's Python
//
// ⛔ Ubuntu 24.04's `deluged` is 2.1.2~dev0 and Chocolatey's `deluge` is
// 2.1.1, so neither is a route at the newest stable. Scoop's extras bucket
// carries 2.2.0 and points at the same installer winget does, so it would be a
// second package route on a host that runs one package route.
//
// Exit codes: the selftest's, 0 every case held, 1 one did not, 2 could not run.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	delugePyPIName = "deluge"
	delugeWingetID = "DelugeTeam.Deluge"
)

// delugePins are what every pip route installs beside Deluge, and the only
// versions any route names.
//
// ⚠ libtorrent IS NAMED BECAUSE DELUGE DECLARES NO DEPENDENCY ON IT, so a route
// that installs it has to; left unpinned it would be PyPI's newest, `2.1.1`
// since 2026-08-10, a series newer than Deluge 2.2.0. 2.0.15 is the newest of
// the 2.0 series and the libtorrent the published qBittorrent pair measured.
// ⛔ setuptools IS PINNED BECAUSE THE UNPINNED INSTALL STARTS NO DAEMON.
// Measured on 2026-10-16: PyPI's newest setuptools, 84.0.0, carries no
// `pkg_resources`, which 2.2.0's `pluginmanagerbase.py` imports, so `deluged`
// died with `ModuleNotFoundError` at start while `--version` still answered.
// 81.0.0 carries it, 82.0.0 does not, and setuptools' own warning says to pin
// below 81. The vendor's Windows installer bundles 68.2.2.
// ⛔ pyOpenSSL IS PINNED FOR THE SAME REASON, found the same day by the next
// start: 2.2.0's `rpcserver.py` makes its certificate with
// `OpenSSL.crypto.X509Req`, which 26.2.0 carries and 26.3.0 does not, and the
// daemon logged *Unable to start deluged* with its RPC already bound. The
// vendor's Windows installer bundles 25.0.0.
var delugePins = []string{"libtorrent==2.0.15", "setuptools<81", "pyOpenSSL<26.3"}

// delugeImportCheck is what a pip route asks the venv's interpreter after the
// install: the engine, the daemon's modules and the console's, each of which
// `--version` answers without loading, and the one name the RPC server takes
// from pyOpenSSL at start, which an import of its module does not reach.
const delugeImportCheck = "import libtorrent, deluge.core.daemon, deluge.ui.console.main; from OpenSSL.crypto import X509Req"

// delugeReadOnlyKeys are passed to `--read-only-config-keys`, so nothing that
// reaches the daemon over RPC can turn a surface back on.
var delugeReadOnlyKeys = []string{
	"dht", "lsd", "upnp", "natpmp", "utpex",
	"enc_in_policy", "enc_out_policy", "enc_level",
	"new_release_check", "send_info", "allow_remote",
}

// delugeCoreConf is `core.conf` as the daemon reads it from `-c`: Deluge's two
// JSON objects, the format header and the settings, written whole on every
// session rather than patched.
//
// ⛔ EVERY SURFACE ITS BUILD HAS, NAMED. DHT, local peer discovery, UPnP,
// NAT-PMP and peer exchange (`utpex`) default to true in 2.2.0's
// `preferencesmanager.py`, read from the tarball on 2026-10-16, and so does
// `new_release_check`, a request to deluge-torrent.org at start. `send_info`
// and `allow_remote` are already false and are written anyway, because a
// default is a property of a version this file does not pin.
//
// ⚠ ENCRYPTION IS ALLOWED, NOT FORCED, which is qBittorrent's choice for the
// same engine: `Session\Encryption=0` is libtorrent's `pe_enabled` both ways,
// and so are Deluge's `enc_in_policy` and `enc_out_policy` of 1, with
// `enc_level` 2 for either handshake. Transmission's `"encryption": 0` is its
// own spelling of the same tolerance.
// ⚠ NO uTP SWITCH: 2.2.0's settings carry none, so uTP stays libtorrent's
// default, as it does for qBittorrent.
//
// ⚠ Peer port "0" leaves the port to the build, as the other adapters do:
// neither `listen_ports` nor `random_port` is written, and Deluge's own default
// draws one.
func delugeCoreConf(downloads, peerPort string) (string, error) {
	if peerPort != "" && peerPort != "0" {
		if n, err := strconv.Atoi(peerPort); err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != peerPort {
			return "", fmt.Errorf("peer port [%s] is not a port", peerPort)
		}
	}
	if downloads == "" {
		return "", fmt.Errorf("the download directory is empty")
	}
	str := func(s string) string {
		b, _ := json.Marshal(s)
		return string(b)
	}
	type kv struct{ k, v string }
	doc := []kv{
		{"dht", "false"},
		{"lsd", "false"},
		{"upnp", "false"},
		{"natpmp", "false"},
		{"utpex", "false"},
		{"enc_in_policy", "1"},
		{"enc_out_policy", "1"},
		{"enc_level", "2"},
		{"new_release_check", "false"},
		{"send_info", "false"},
		{"allow_remote", "false"},
		{"enabled_plugins", "[]"},
		{"download_location", str(downloads)},
		{"torrentfiles_location", str(downloads)},
		{"move_completed", "false"},
		{"copy_torrent_file", "false"},
		{"pre_allocate_storage", "false"},
		{"add_paused", "false"},
		{"stop_seed_at_ratio", "false"},
		{"remove_seed_at_ratio", "false"},
	}
	if peerPort != "" && peerPort != "0" {
		doc = append(doc, kv{"random_port", "false"}, kv{"listen_ports", "[" + peerPort + ", " + peerPort + "]"})
	}
	var b strings.Builder
	b.WriteString("{\n    \"file\": 1,\n    \"format\": 1\n}{\n")
	for i, e := range doc {
		sep := ","
		if i == len(doc)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "    %s: %s%s\n", str(e.k), e.v, sep)
	}
	b.WriteString("}")
	return b.String(), nil
}

// delugeDaemonArgs is the daemon's command line, after its executable.
//
// ⛔ THE RPC ON LOOPBACK, ON A PORT DRAWN PER SESSION, AND THE LOG IN THE
// WORKDIR. `-u 127.0.0.1` is the interface the console reaches it on; the
// torrent listener is left on every interface, as the other adapters leave it.
// ⚠ `-d` keeps the daemon the process `start` launched. It exists off Windows
// only: 2.2.0's `argparserbase.py` adds it under `if not windows_check()`, and
// on Windows the daemon never forks.
func delugeDaemonArgs(config string, rpcPort int, logfile string, foreground bool) []string {
	args := []string{}
	if foreground {
		args = append(args, "-d")
	}
	return append(args, "-c", config, "-p", strconv.Itoa(rpcPort), "-u", "127.0.0.1",
		"-L", "info", "-l", logfile, "--read-only-config-keys", strings.Join(delugeReadOnlyKeys, ","))
}

// delugeConsoleArgs is `deluge-console`'s command line for one step on the
// daemon: connect, then the step.
//
// ⛔ CONNECTED BY ADDRESS WITH NO USER, so the console reads the daemon's own
// `localclient` credential out of the config directory it shares: 2.2.0's
// `client.py` does that only for a host spelled `127.0.0.1` or `localhost`.
// The credential is never on a command line, in a log or in this process.
// ⚠ The console splits on `;` before it reads quotes, and quotes with POSIX
// `shlex`, so a path holding either a `;` or a `'` is refused rather than
// written; inside single quotes a Windows backslash is literal.
func delugeConsoleArgs(config string, rpcPort int, step string, path string) ([]string, error) {
	cmd := fmt.Sprintf("connect 127.0.0.1:%d; %s", rpcPort, step)
	if path != "" {
		if strings.ContainsAny(path, ";'\r\n") {
			return nil, fmt.Errorf("path [%s] cannot be handed to deluge-console", path)
		}
		cmd += " '" + path + "'"
	}
	return []string{"-c", config, "-L", "error", cmd}, nil
}

// delugeAdded reads what `deluge-console` printed for `add`.
//
// ⚠ THE CONSOLE ANSWERS 0 WHATEVER HAPPENED, so the words are what is read:
// 2.2.0's `add.py` prints `Torrent added!` for each torrent the daemon took,
// and `Could not connect to daemon` from `main.py` is the other answer.
func delugeAdded(output string) error {
	if strings.Contains(output, "Could not connect") {
		return fmt.Errorf("deluge-console could not connect to the daemon")
	}
	if !strings.Contains(output, "Torrent added!") {
		return fmt.Errorf("deluge-console did not say the torrent was added")
	}
	return nil
}

var delugeVersionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}([A-Za-z0-9.+-]*)$`)
var delugeProgRe = regexp.MustCompile(`(?i)^deluged(-debug)?(\.exe)?$`)

// delugeParseVersion reads `deluged --version` and answers the version and
// the libtorrent it loaded.
//
// ⛔ THE BUILD SPEAKING. 2.2.0's `argparserbase.py` prints `<prog> <version>`,
// `libtorrent: <version>`, `Python: ...` and `OS: ...`, where the version is
// the installed distribution's metadata, read by the running build. The
// program's name is argparse's `prog`, so it is `deluged` on Linux and may be
// `deluged-debug.exe` on Windows.
// ⚠ A BUILD THAT DID NOT LOAD LIBTORRENT IS REFUSED: the line is printed only
// when the import worked, and a daemon without its engine cannot download.
func delugeParseVersion(output string) (version, engine string, err error) {
	var first string
	for _, l := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if first == "" {
			first = l
			continue
		}
		if v, ok := strings.CutPrefix(l, "libtorrent:"); ok && engine == "" {
			engine = strings.TrimSpace(v)
		}
	}
	f := strings.Fields(first)
	if len(f) != 2 || !delugeProgRe.MatchString(f[0]) || !delugeVersionRe.MatchString(f[1]) {
		return "", "", fmt.Errorf("deluged reported no parseable version: [%s]", first)
	}
	if !delugeVersionRe.MatchString(engine) {
		return "", "", fmt.Errorf("deluged %s reported no libtorrent: [%s]", f[1], engine)
	}
	return f[1], engine, nil
}

// delugeTagVersion is the version a `deluge-` tag names, or "".
func delugeTagVersion(tag string) string {
	v, ok := strings.CutPrefix(tag, "deluge-")
	if !ok || !delugeVersionRe.MatchString(v) {
		return ""
	}
	return v
}

// delugePipReport reads what `pip install --report` said it installed for one
// package: the version, where the bytes came from, and their sha256 when the
// index stated one.
//
// ⛔ IT IS PIP'S STATEMENT, NEVER THE VERSION THE ROUTE REPORTS. The route
// compares it with the build's own answer and refuses a difference, as the
// winget routes refuse one. ⚠ Exactly one entry for the package, or none is
// read: two would be a report this route did not ask for.
func delugePipReport(body []byte, pkg string) (version, url, sha string, err error) {
	var report struct {
		Install []struct {
			DownloadInfo struct {
				URL         string `json:"url"`
				ArchiveInfo *struct {
					Hashes map[string]string `json:"hashes"`
				} `json:"archive_info"`
			} `json:"download_info"`
			Metadata struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"metadata"`
		} `json:"install"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return "", "", "", fmt.Errorf("pip's report is not JSON: %v", err)
	}
	found := 0
	for _, e := range report.Install {
		if !strings.EqualFold(e.Metadata.Name, pkg) {
			continue
		}
		found++
		version, url = e.Metadata.Version, e.DownloadInfo.URL
		if e.DownloadInfo.ArchiveInfo != nil {
			sha = e.DownloadInfo.ArchiveInfo.Hashes["sha256"]
		}
	}
	switch {
	case found == 0:
		return "", "", "", fmt.Errorf("pip's report names no %s", pkg)
	case found > 1:
		return "", "", "", fmt.Errorf("pip's report names %s %d times", pkg, found)
	case !delugeVersionRe.MatchString(version):
		return "", "", "", fmt.Errorf("pip's report names %s at [%s], which is not a version", pkg, version)
	case url == "":
		return "", "", "", fmt.Errorf("pip's report names no origin for %s", pkg)
	}
	return version, url, sha, nil
}

// delugeWingetOrigin is the installer URL winget's index pins for the
// package route.
//
// ⚠ AN http URL IS ACCEPTED HERE, AND ONLY HERE. `DelugeTeam.Deluge` 2.2.0's
// manifest names `http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe`,
// read on 2026-10-16, and winget checks the download against the manifest's
// InstallerSha256 `B2E414DD...8F4E889`, which is the vendor's own `.sha256`.
// The record carries the URL as the index named it.
func delugeWingetOrigin(show string) (string, error) {
	return wingetInstallerURLWith(show, "http://", "https://")
}

// delugeScrub takes the daemon's credential out of a workdir before it ships,
// and answers what it removed, relative to the workdir.
//
// ⛔ RULE 12, AND THE BUNDLE SCAN CANNOT SEE THIS ONE. The daemon writes
// `auth` into its config directory on first start, `localclient:<40 hex>:10`,
// and keeps `auth.bak` beside it: a name `scan-bundle` does not call a
// credential and a line that is not a JSON member. So every file named
// `auth` or `auth.<anything>` under the config directory goes, and so does
// every file anywhere in the workdir that holds one of their passwords, which
// is where a log or a `hostlist.conf` would carry it. ⚠ `hostlist.conf` goes
// whether or not it matches, because 2.2.0's `hostlist.py` writes the
// localhost password into its default entry.
// ⛔ AND THE RPC SERVER'S TLS KEY. Driven on 2026-10-16: the daemon makes
// `ssl/daemon.pkey`, a PEM private key, and `ssl/daemon.cert` on first start,
// and `.pkey` is not a name the bundle scan refuses. Everything under `ssl/`
// goes.
// ⚠ A password shorter than eight characters is not searched for, as the
// bundle scan's value rule stops at eight: Deluge writes forty.
func delugeScrub(workdir, config string) ([]string, error) {
	var passwords [][]byte
	var removed []string
	var credentialFiles []string
	err := filepath.WalkDir(config, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		inSSL := filepath.Base(filepath.Dir(p)) == "ssl" && filepath.Dir(filepath.Dir(p)) == filepath.Clean(config)
		if inSSL || name == "auth" || strings.HasPrefix(name, "auth.") || name == "hostlist.conf" || strings.HasPrefix(name, "hostlist.conf.") {
			credentialFiles = append(credentialFiles, p)
		}
		if name == "auth" || strings.HasPrefix(name, "auth.") {
			body, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for _, line := range strings.Split(string(body), "\n") {
				f := strings.Split(strings.TrimSpace(line), ":")
				if len(f) >= 2 && len(f[1]) >= 8 {
					passwords = append(passwords, []byte(f[1]))
				}
			}
		}
		return nil
	})
	if err != nil {
		return removed, err
	}
	rel := func(p string) string {
		if r, err := filepath.Rel(workdir, p); err == nil {
			return filepath.ToSlash(r)
		}
		return p
	}
	for _, p := range credentialFiles {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return removed, err
		}
		removed = append(removed, rel(p))
	}
	if len(passwords) == 0 {
		sort.Strings(removed)
		return removed, nil
	}
	err = filepath.WalkDir(workdir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, pw := range passwords {
			if bytes.Contains(body, pw) {
				if err := os.Remove(p); err != nil {
					return err
				}
				removed = append(removed, rel(p))
				break
			}
		}
		return nil
	})
	sort.Strings(removed)
	return removed, err
}

// delugeFreePort asks the kernel for a loopback port nothing holds, for the
// daemon's RPC. ⚠ Per session and recorded in `rpc-port`, because the
// daemon's default, 58846, is one a host may already hold.
func delugeFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// delugeVenvHome reads the `home` a venv's `pyvenv.cfg` names: the directory
// of the interpreter the venv's own launcher starts. ⚠ On Windows that
// interpreter is a second process beside the launcher, and the one that opens
// the sockets, so the firewall has to name it.
func delugeVenvHome(cfg string) string {
	for _, l := range strings.Split(strings.ReplaceAll(cfg, "\r", ""), "\n") {
		k, v, ok := strings.Cut(l, "=")
		if ok && strings.TrimSpace(k) == "home" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// delugeStopRecord is the line `stop` appends to `stop.txt`.
func delugeStopRecord(at string, pid int, how string) string {
	return fmt.Sprintf("%s pid %d: %s\n", at, pid, how)
}

// -- the selftest -------------------------------------------------------------

// delugeSelftest is `bit-check deluge --selftest`: every function above asked
// over a case whose answer is known, on any host.
func delugeSelftest(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "deluge: usage: bit-check deluge --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "deluge: FAIL %s\n", name)
		}
	}

	// The settings: Deluge's two objects, every surface off, a Windows path that
	// survives being read back.
	doc, err := delugeCoreConf(`C:\work\client\downloads`, "51413")
	var head, body map[string]any
	parsed := false
	if err == nil {
		dec := json.NewDecoder(strings.NewReader(doc))
		parsed = dec.Decode(&head) == nil && dec.Decode(&body) == nil && !dec.More()
	}
	check("core.conf is two JSON objects", parsed)
	check("core.conf's header is file 1, format 1", parsed && head["file"] == float64(1) && head["format"] == float64(1))
	for k, want := range map[string]any{
		"dht": false, "lsd": false, "upnp": false, "natpmp": false, "utpex": false,
		"enc_in_policy": float64(1), "enc_out_policy": float64(1), "enc_level": float64(2),
		"new_release_check": false, "send_info": false, "allow_remote": false,
		"random_port": false, "download_location": `C:\work\client\downloads`, "add_paused": false,
	} {
		check(fmt.Sprintf("core.conf holds %s=%v", k, want), parsed && body[k] == want)
	}
	ports, _ := body["listen_ports"].([]any)
	check("core.conf listens on the peer port alone", len(ports) == 2 && ports[0] == float64(51413) && ports[1] == float64(51413))
	d0, err0 := delugeCoreConf("/d", "0")
	check("peer port 0 leaves the port to the build", err0 == nil && !strings.Contains(d0, "listen_ports") && !strings.Contains(d0, "random_port"))
	for _, bad := range []string{"51413 ", "x", "70000", "-1", "0051413"} {
		_, err := delugeCoreConf("/d", bad)
		check("the peer port ["+bad+"] is refused", err != nil)
	}
	_, err = delugeCoreConf("", "1")
	check("an empty download directory is refused", err != nil)
	for _, k := range []string{"dht", "lsd", "upnp", "natpmp", "utpex"} {
		in := false
		for _, r := range delugeReadOnlyKeys {
			in = in || r == k
		}
		check("the surface "+k+" is read-only to RPC", in)
	}

	// The pins.
	check("the routes pin libtorrent 2.0.15", len(delugePins) == 3 && delugePins[0] == "libtorrent==2.0.15")
	check("the routes pin pyOpenSSL below 26.3, which still carries X509Req", len(delugePins) == 3 && delugePins[2] == "pyOpenSSL<26.3")
	check("the routes pin setuptools below 81, which still carries pkg_resources", delugePins[1] == "setuptools<81")
	check("the import check loads the daemon and the console", strings.Contains(delugeImportCheck, "deluge.core.daemon") &&
		strings.Contains(delugeImportCheck, "deluge.ui.console.main") && strings.Contains(delugeImportCheck, "libtorrent"))

	// The command lines.
	a := delugeDaemonArgs("/w/config", 40123, "/w/deluged-1.log", true)
	check("the daemon stays in the foreground off Windows", len(a) > 0 && a[0] == "-d")
	check("the daemon's RPC is on loopback, on the drawn port",
		strings.Contains(strings.Join(a, " "), "-p 40123 -u 127.0.0.1"))
	check("the daemon is handed its read-only keys",
		strings.Contains(strings.Join(a, " "), "--read-only-config-keys dht,lsd,upnp,natpmp,utpex,"))
	w := delugeDaemonArgs(`C:\w\config`, 1, `C:\w\l.log`, false)
	check("the daemon is handed no -d on Windows", len(w) > 0 && w[0] == "-c")
	c, err := delugeConsoleArgs("/w/config", 40123, "add", "/w/t.torrent")
	check("the console connects by address and adds by path",
		err == nil && c[len(c)-1] == "connect 127.0.0.1:40123; add '/w/t.torrent'" && c[0] == "-c" && c[1] == "/w/config")
	c, err = delugeConsoleArgs(`C:\w\config`, 1, "add", `C:\w\t.torrent`)
	check("a Windows path is quoted with its backslashes", err == nil && c[len(c)-1] == `connect 127.0.0.1:1; add 'C:\w\t.torrent'`)
	c, err = delugeConsoleArgs("/c", 2, "halt", "")
	check("halt is connect then halt", err == nil && c[len(c)-1] == "connect 127.0.0.1:2; halt")
	for _, bad := range []string{"/w/a;b.torrent", "/w/it's.torrent"} {
		_, err := delugeConsoleArgs("/c", 1, "add", bad)
		check("the path ["+bad+"] is refused", err != nil)
	}
	check("an added torrent reads as added", delugeAdded("Attempting to add torrent: /w/t.torrent\nTorrent added!\n") == nil)
	check("a refused connection is refused", delugeAdded("Could not connect to daemon: 127.0.0.1:1\n") != nil)
	check("silence is not an add", delugeAdded("") != nil)
	check("a torrent the daemon declined is refused", delugeAdded("Attempting to add torrent: t\nTorrent was not added: Torrent already in session\n") != nil)

	// The build's answer.
	for in, want := range map[string][2]string{
		"deluged 2.2.0\nlibtorrent: 2.0.15.0\nPython: 3.12.3\nOS: Linux Ubuntu 24.04 noble\n":     {"2.2.0", "2.0.15.0"},
		"deluged-debug.exe 2.2.0\r\nlibtorrent: 1.2.19.0\r\nPython: 3.9.13\r\nOS: Windows 10\r\n": {"2.2.0", "1.2.19.0"},
		"deluged-debug 2.2.0\nlibtorrent: 2.0.15.0\n":                                             {"2.2.0", "2.0.15.0"},
	} {
		v, e, err := delugeParseVersion(in)
		check("the answer ["+strings.TrimSpace(strings.SplitN(in, "\n", 2)[0])+"] reads as "+want[0]+" on "+want[1], err == nil && v == want[0] && e == want[1])
	}
	for _, in := range []string{"", "deluged", "deluge 2.2.0\nlibtorrent: 2.0.15.0\n", "deluged 2.2.0\nPython: 3.12.3\n",
		"deluged: error: unrecognized arguments\nlibtorrent: 2.0.15.0\n", "deluged 2.2.0 extra\nlibtorrent: 2.0.15.0\n"} {
		_, _, err := delugeParseVersion(in)
		check("the answer ["+strings.ReplaceAll(in, "\n", "|")+"] is refused", err != nil)
	}
	check("the tag deluge-2.2.0 names 2.2.0", delugeTagVersion("deluge-2.2.0") == "2.2.0")
	check("a foreign tag names nothing", delugeTagVersion("archive/1.3-stable") == "" && delugeTagVersion("2.2.0") == "")

	// What pip and winget said.
	rep := `{"version":"1","install":[{"download_info":{"url":"https://files.pythonhosted.org/packages/cc/c7/x/deluge-2.2.0-py3-none-any.whl","archive_info":{"hashes":{"sha256":"5ac4ce73"}}},"metadata":{"name":"deluge","version":"2.2.0"}},` +
		`{"download_info":{"url":"https://files.pythonhosted.org/l/libtorrent-2.0.15.whl","archive_info":{"hashes":{"sha256":"aa"}}},"metadata":{"name":"libtorrent","version":"2.0.15"}}]}`
	v, u, s, err := delugePipReport([]byte(rep), "deluge")
	check("pip's report names deluge 2.2.0, its wheel and its digest", err == nil && v == "2.2.0" && strings.HasSuffix(u, "deluge-2.2.0-py3-none-any.whl") && s == "5ac4ce73")
	v, _, _, err = delugePipReport([]byte(rep), "libtorrent")
	check("and the engine beside it", err == nil && v == "2.0.15")
	src := `{"install":[{"download_info":{"url":"file:///w/src","dir_info":{}},"metadata":{"name":"deluge","version":"2.2.0"}}]}`
	v, u, s, err = delugePipReport([]byte(src), "deluge")
	check("a local build names its directory and no digest", err == nil && v == "2.2.0" && u == "file:///w/src" && s == "")
	_, _, _, err = delugePipReport([]byte(`{"install":[]}`), "deluge")
	check("a report that installed no deluge is refused", err != nil)
	_, _, _, err = delugePipReport([]byte(strings.Replace(rep, `"libtorrent"`, `"Deluge"`, 1)), "deluge")
	check("a report naming deluge twice is refused", err != nil)
	_, _, _, err = delugePipReport([]byte("not json"), "deluge")
	check("a report that is not JSON is refused", err != nil)
	show := "Found Deluge BitTorrent Client [DelugeTeam.Deluge]\r\nVersion: 2.2.0\r\nInstaller:\r\n  Installer Type: nullsoft\r\n" +
		"  Installer Url: http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe\r\n"
	o, err := delugeWingetOrigin(show)
	check("winget's http installer URL is the origin", err == nil && o == "http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe")
	_, err = wingetInstallerURL(show)
	check("and every other winget route still refuses http", err != nil)
	_, err = delugeWingetOrigin("Installer Url: ftp://x/y.exe\n")
	check("an installer URL that is neither http nor https is refused", err != nil)
	wv, err := trWinWingetVersion(show)
	check("winget's index version is read", err == nil && wv == "2.2.0")

	// The venv's interpreter.
	check("pyvenv.cfg names its home", delugeVenvHome("home = C:\\hostedtoolcache\\windows\\Python\\3.12.10\\x64\r\ninclude-system-site-packages = false\r\n") ==
		`C:\hostedtoolcache\windows\Python\3.12.10\x64`)
	check("a pyvenv.cfg with no home names none", delugeVenvHome("version = 3.12.3\n") == "")
	check("stop's record says how", delugeStopRecord("2026-10-16T00:00:00Z", 7, "halt") == "2026-10-16T00:00:00Z pid 7: halt\n")

	// The scrub, over a scratch workdir.
	work, err := os.MkdirTemp("", "deluge-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "deluge: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	mk := func(rel, body string) {
		p := filepath.Join(work, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	// ⚠ Forty characters, as Deluge writes, and not hex, which the public
	// secret rule rightly refuses in the tree.
	pw := strings.Repeat("pw", 20)
	mk("config/auth", "localclient:"+pw+":10\n")
	mk("config/auth.bak", "localclient:"+pw+":10\n")
	mk("config/hostlist.conf", "{\"file\": 3}{\"hosts\": []}")
	mk("config/ssl/daemon.pkey", "the key, by its place and not its bytes\n")
	mk("config/ssl/daemon.cert", "the certificate\n")
	mk("config/core.conf", "{\"file\": 1}{\"dht\": false}")
	mk("config/core.conf.bak", "{\"file\": 1}{\"dht\": false}")
	mk("deluged-1.log", "[INFO] authorized localclient with "+pw+"\n")
	mk("client.out", "started deluged\n")
	mk("downloads/payload.bin", "\x00\x01")
	gone, err := delugeScrub(work, filepath.Join(work, "config"))
	check("the scrub removes the credential, its backup, the hostlist, the TLS key and the log that held it", err == nil &&
		strings.Join(gone, ",") == "config/auth,config/auth.bak,config/hostlist.conf,config/ssl/daemon.cert,config/ssl/daemon.pkey,deluged-1.log")
	check("the scrub keeps the settings, their backup and the rest", isRegular(filepath.Join(work, "config", "core.conf")) &&
		isRegular(filepath.Join(work, "config", "core.conf.bak")) && isRegular(filepath.Join(work, "client.out")) &&
		isRegular(filepath.Join(work, "downloads", "payload.bin")))
	left := false
	filepath.WalkDir(work, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if b, _ := os.ReadFile(p); bytes.Contains(b, []byte(pw)) {
				left = true
			}
		}
		return nil
	})
	check("no file in the workdir holds the password", !left)
	gone, err = delugeScrub(work, filepath.Join(work, "config"))
	check("a second scrub finds nothing", err == nil && len(gone) == 0)
	gone, err = delugeScrub(work, filepath.Join(work, "no-such-config"))
	check("a workdir whose daemon never started scrubs clean", err == nil && len(gone) == 0)
	mk("short/config/auth", "localclient:short:10\n")
	mk("short/keep.txt", "short\n")
	gone, err = delugeScrub(filepath.Join(work, "short"), filepath.Join(work, "short", "config"))
	check("a short password is not searched for, and its file still goes", err == nil && strings.Join(gone, ",") == "config/auth" &&
		isRegular(filepath.Join(work, "short", "keep.txt")))

	// The installed tree's digest.
	mk("tree/deluge/__init__.py", "a")
	mk("tree/deluge/core/core.py", "b")
	mk("tree/deluge/core/__pycache__/core.cpython-312.pyc", "c")
	listing := filepath.Join(work, "installed-files.txt")
	err = delugeRecordTree(filepath.Join(work, "tree", "deluge"), "deluge", listing)
	got, _ := os.ReadFile(listing)
	check("the tree is digested by path, bytecode left out", err == nil && string(got) ==
		fmt.Sprintf("%x  deluge/__init__.py\n%x  deluge/core/core.py\n", sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))))
	check("Git for Windows' timeout is beside its cmd directory",
		delugeGNUTimeout("/c/Program Files/Git/cmd/git.exe") == "/c/Program Files/Git/usr/bin/timeout.exe")
	check("no git names no timeout", delugeGNUTimeout("") == "")

	port, err := delugeFreePort()
	check("a free loopback port is drawn", err == nil && port > 0 && port < 65536)

	fmt.Printf(`{"schema":"deluge-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}

// delugeRecordTree writes every file under root with its sha256, in
// `sha256sum`'s form, each path led by label and the whole sorted by path,
// which is the digest of what a route installed. ⚠ Bytecode is left out: the
// interpreter writes it after the install.
func delugeRecordTree(root, label, out string) error {
	digests := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() || strings.HasSuffix(p, ".pyc") {
			return err
		}
		sum, err := fileSHA256(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		digests[label+"/"+filepath.ToSlash(rel)] = sum
		return nil
	})
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(digests))
	for p := range digests {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&b, "%s  %s\n", digests[p], p)
	}
	return os.WriteFile(out, []byte(b.String()), 0o644)
}

// delugeGNUTimeout is where Git for Windows keeps coreutils' `timeout`, given
// the `git.exe` PATH found: `<Git>\usr\bin\timeout.exe` beside `<Git>\cmd` or
// `<Git>\bin`. ⛔ The Windows source route bounds its clone with it, because
// System32's `timeout.exe` is a pause and not a bound, and `check-adapters`
// requires a clone to run behind a `timeout`.
func delugeGNUTimeout(git string) string {
	if git == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(filepath.Dir(git)), "usr", "bin", "timeout.exe")
}
