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
	"debug/pe"
	"encoding/binary"
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
// ⛔ windows-curses IS UPSTREAM'S OWN WINDOWS REQUIREMENT, WITH UPSTREAM'S
// MARKER. capture-client run 39588980862's Windows source lane installed the
// tag and its import check stopped on `import curses`: CPython on Windows has
// no `_curses`, and the console's `widgets/sidebar.py` imports it. The tag's
// `setup.py` does not declare it; its `requirements.txt` carries
// `windows-curses; sys_platform == 'win32'`, so the requirement is passed in
// that form and is nothing off Windows. Unpinned, as upstream leaves it:
// 2.4.2 is the newest, with a cp312 win_amd64 wheel.
// ⚠ The same file caps `twisted[tls]<23` on win32, and that cap is NOT taken:
// run 39616084687's Windows source lane captured on Twisted 26, and Twisted is
// not on the wire.
// ⛔ AND libtorrent IS 1.2.19 ON WINDOWS, THE VENDOR INSTALLER'S OWN. The first
// green Windows pair, run 39616084687, assembled `divergent`: the installer's
// libtorrent 1.2.19 and the source route's 2.0.15 differ in the peer wire's
// reserved bytes and in the tracker user agent, so neither route could
// publish. The two routes on one host have to run one engine, and the
// vendor's build fixes it at 1.2.19, whose Windows wheels stop at cp310; the
// Windows source route runs on Python 3.10 for it, see delugePickPython.
// Linux keeps 2.0.15, and a field that differs by platform is
// platform-scoped under the operator's settled rule.
var delugePins = []string{`libtorrent==2.0.15; sys_platform != "win32"`, `libtorrent==1.2.19; sys_platform == "win32"`,
	"setuptools<81", "pyOpenSSL<26.3",
	`windows-curses; sys_platform == "win32"`}

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

// delugeDelivery is what the Windows package route records as its delivery:
// the vendor's installer, fetched over https by `install-rootless --fetch`
// and run by this adapter, rather than by winget.
const delugeDelivery = "https-vendor-mirror-nsis"

// delugeWingetField is the one value `winget show` prints under `Installer:`
// for key, such as `Installer Url` or `Installer SHA256`. ⛔ Two different
// ones is a refusal, as wingetInstallerURL refuses two URLs.
func delugeWingetField(show, key string) (string, error) {
	found := ""
	for _, l := range strings.Split(show, "\n") {
		l = strings.TrimSpace(strings.TrimRight(l, "\r"))
		v, ok := strings.CutPrefix(l, key+":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if found != "" && found != v {
			return "", fmt.Errorf("winget's index named two values for %s, %s and %s", key, found, v)
		}
		found = v
	}
	if found == "" {
		return "", fmt.Errorf("winget's index named no %s", key)
	}
	return found, nil
}

var delugeSHA256Re = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// delugeIndexSHA256 is the InstallerSha256 winget's index pins, in lower case.
func delugeIndexSHA256(show string) (string, error) {
	v, err := delugeWingetField(show, "Installer SHA256")
	if err != nil {
		return "", err
	}
	if !delugeSHA256Re.MatchString(v) {
		return "", fmt.Errorf("winget's index named [%s] as the installer's SHA256", v)
	}
	return strings.ToLower(v), nil
}

var delugeIndexURLRe = regexp.MustCompile(`^https?://download\.deluge-torrent\.org/windows/(deluge-[0-9][0-9A-Za-z.]*-win64-setup\.exe)$`)

// delugeMirrorURL is the https URL the Windows package route fetches, for the
// installer URL winget's index pins, and the installer's file name.
//
// ⛔ THE INDEX PINS AN http URL AND NO RECORD MAY NAME ONE. `DelugeTeam.Deluge`
// 2.2.0 names `http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe`,
// and so does Scoop's extras bucket, and `Url::parse` refuses a plain http
// retrieval because it cannot support a claim about which bytes arrived:
// assemble-bundle run 39617383153 refused the pair for it. So the route fetches
// the same file from the vendor's own mirror, `ftp.osuosl.org/pub/deluge/windows/`,
// over https, and records that URL, which is what it really fetched.
// ⚠ THE NAME IS KEPT AND THE BYTES ARE CHECKED TWICE: against the index's
// InstallerSha256 and against the vendor's `.sha256` beside the file, measured
// equal on 2026-10-16 (`b2e414dd...8f4e889`). A URL of any other shape is
// refused rather than mapped.
func delugeMirrorURL(indexURL string) (string, string, error) {
	m := delugeIndexURLRe.FindStringSubmatch(indexURL)
	if m == nil {
		return "", "", fmt.Errorf("winget's index pins [%s], which is not the vendor's win64 installer on download.deluge-torrent.org", indexURL)
	}
	return "https://ftp.osuosl.org/pub/deluge/windows/" + m[1], m[1], nil
}

// delugeNSISCmdLine is the command line that runs the vendor's installer
// silently into dir, written whole.
//
// ⛔ NSIS READS `/D=` UNQUOTED AND LAST, so it is never left to an argument
// quoter, which would quote a path holding a space. ⚠ This is the line winget
// runs for a `nullsoft` installer with `--silent --location`: `/S` and `/D=`.
// A path holding a double quote or a newline is refused.
func delugeNSISCmdLine(setup, dir string) (string, error) {
	for _, p := range []string{setup, dir} {
		if p == "" || strings.ContainsAny(p, "\"\r\n") {
			return "", fmt.Errorf("path [%s] cannot be written on the installer's command line", p)
		}
	}
	return fmt.Sprintf(`"%s" /S /D=%s`, setup, dir), nil
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
	check("Linux pins libtorrent 2.0.15", len(delugePins) == 5 && delugePins[0] == `libtorrent==2.0.15; sys_platform != "win32"`)
	check("Windows pins the vendor installer's libtorrent, 1.2.19", len(delugePins) == 5 &&
		delugePins[1] == `libtorrent==1.2.19; sys_platform == "win32"`)
	check("the routes pin pyOpenSSL below 26.3, which still carries X509Req", len(delugePins) == 5 && delugePins[3] == "pyOpenSSL<26.3")
	check("Windows gets windows-curses, upstream's marker and all", len(delugePins) == 5 &&
		delugePins[4] == `windows-curses; sys_platform == "win32"`)
	tc := []string{`C:\hostedtoolcache\windows\Python\3.12.10\x64\python.exe`, `C:\hostedtoolcache\windows\Python\3.10.9\x64\python.exe`,
		`C:\hostedtoolcache\windows\Python\3.10.11\x64\python.exe`, `C:\hostedtoolcache\windows\Python\3.9.13\x64\python.exe`,
		`C:\hostedtoolcache\windows\Python\3.10.12\x86\python.exe`}
	py, err := delugePickPython(tc)
	check("the Windows source route takes the newest x64 3.10, not the runner's 3.12", err == nil && py == tc[2])
	py, err = delugePickPython([]string{tc[0], tc[3]})
	check("and 3.9 when there is no 3.10", err == nil && py == tc[3])
	_, err = delugePickPython([]string{tc[0], `C:\t\Python\3.10.x\x64\python.exe`})
	check("a tool cache with no 3.10 or 3.9 is refused", err != nil)
	for in, want := range map[string]string{"libcrypto-1_1-x64.dll": "_internal/libcrypto-1_1-x64.dll",
		"LIBSSL-1_1-X64.DLL": "_internal/libssl-1_1-x64.dll"} {
		got, ok := delugeVendorDLL(in)
		check("the wheel's "+in+" comes from the vendor's installer as "+want, ok && got == want)
	}
	for _, other := range []string{"MSVCP140.dll", "libcrypto-1_1.dll", "python310.dll"} {
		_, taken := delugeVendorDLL(other)
		check(other+" is not taken from the installer: a missing one is refused by name", !taken)
	}
	vu, vf, err := delugeVendorInstallerURL("2.2.0")
	check("the vendor's installer for the tag's version is fetched over https from its mirror", err == nil &&
		vu == "https://ftp.osuosl.org/pub/deluge/windows/deluge-2.2.0-win64-setup.exe" && vf == "deluge-2.2.0-win64-setup.exe")
	_, _, err = delugeVendorInstallerURL("2.2.0/../x")
	check("a version that is not one names no installer", err != nil)
	check("an API set is the loader's", delugeAPISet("api-ms-win-crt-runtime-l1-1-0.dll") && !delugeAPISet("KERNEL32.dll"))
	fixture := filepath.Join(os.TempDir(), fmt.Sprintf("deluge-pe-%d.pyd", os.Getpid()))
	err = os.WriteFile(fixture, delugePEFixture("libssl-1_1-x64.dll", "WSOCK32.dll"), 0o644)
	imps, ierr := delugePEImports(fixture)
	check("a PE import directory is read whole, in order", err == nil && ierr == nil &&
		strings.Join(imps, ",") == "libssl-1_1-x64.dll,WSOCK32.dll")
	os.Remove(fixture)
	notPE := filepath.Join(os.TempDir(), fmt.Sprintf("deluge-not-pe-%d", os.Getpid()))
	_ = os.WriteFile(notPE, []byte("MZ but nothing more"), 0o644)
	_, err = delugePEImports(notPE)
	check("a file that is not a PE image has no import table to read", err != nil)
	os.Remove(notPE)
	check("the import check reaches the console, which imports curses", strings.Contains(delugeImportCheck, "deluge.ui.console.main"))
	check("the routes pin setuptools below 81, which still carries pkg_resources", len(delugePins) == 5 && delugePins[2] == "setuptools<81")
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
		"  Installer Url: http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe\r\n" +
		"  Installer SHA256: " + strings.Repeat("B2", 32) + "\r\n"
	_, err = wingetInstallerURL(show)
	check("winget's http installer URL is refused as an origin, as Url::parse refuses it", err != nil)
	idx, err := delugeWingetField(show, "Installer Url")
	mirror, file, merr := delugeMirrorURL(idx)
	check("the index's http URL maps to the vendor's https mirror", err == nil && merr == nil && file == "deluge-2.2.0-win64-setup.exe" &&
		mirror == "https://ftp.osuosl.org/pub/deluge/windows/deluge-2.2.0-win64-setup.exe")
	check("the route records an https origin", strings.HasPrefix(mirror, "https://"))
	for _, bad := range []string{"http://download.deluge-torrent.org/windows/deluge-2.2.0-win32-setup.exe",
		"http://evil.example/windows/deluge-2.2.0-win64-setup.exe", "http://download.deluge-torrent.org/windows/../x-win64-setup.exe",
		"http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe?x"} {
		_, _, err := delugeMirrorURL(bad)
		check("the index URL ["+bad+"] is refused rather than mapped", err != nil)
	}
	sha, err := delugeIndexSHA256(show)
	check("the index's InstallerSha256 is read in lower case", err == nil && sha == strings.Repeat("b2", 32))
	_, err = delugeIndexSHA256(strings.Replace(show, strings.Repeat("B2", 32), "b2e4", 1))
	check("a short InstallerSha256 is refused", err != nil)
	_, err = delugeIndexSHA256(show + "  Installer SHA256: " + strings.Repeat("c3", 32) + "\n")
	check("two InstallerSha256 values are refused", err != nil)
	cl, err := delugeNSISCmdLine(`C:\w\deluge-2.2.0-win64-setup.exe`, `C:\Program Files\Deluge`)
	check("the installer runs silently into the location, /D= unquoted and last",
		err == nil && cl == `"C:\w\deluge-2.2.0-win64-setup.exe" /S /D=C:\Program Files\Deluge`)
	_, err = delugeNSISCmdLine(`C:\w\s.exe`, `C:\a"b`)
	check("a location holding a double quote is refused", err != nil)
	check("the delivery is named, and is a slug", delugeDelivery == "https-vendor-mirror-nsis")
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

// delugePEImports is the DLLs a Windows executable or extension imports, read
// from its import directory, in the directory's order.
//
// ⚠ READ HERE RATHER THAN THROUGH debug/pe's ImportedLibraries, which returns
// nothing for a PE file, and ImportedSymbols, which leaves out a DLL imported
// by ordinal alone: libtorrent 1.2.19's `.pyd` imports `WSOCK32.dll` that way.
func delugePEImports(path string) ([]string, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var dir pe.DataDirectory
	switch h := f.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		if len(h.DataDirectory) > pe.IMAGE_DIRECTORY_ENTRY_IMPORT {
			dir = h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_IMPORT]
		}
	case *pe.OptionalHeader32:
		if len(h.DataDirectory) > pe.IMAGE_DIRECTORY_ENTRY_IMPORT {
			dir = h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_IMPORT]
		}
	default:
		return nil, fmt.Errorf("%s has no optional header", path)
	}
	read := func(rva uint32) ([]byte, error) {
		for _, s := range f.Sections {
			if rva >= s.VirtualAddress && rva < s.VirtualAddress+s.VirtualSize {
				b, err := s.Data()
				if err != nil {
					return nil, err
				}
				if off := rva - s.VirtualAddress; int(off) < len(b) {
					return b[off:], nil
				}
			}
		}
		return nil, fmt.Errorf("RVA %#x is in no section of %s", rva, path)
	}
	if dir.VirtualAddress == 0 {
		return nil, nil
	}
	table, err := read(dir.VirtualAddress)
	if err != nil {
		return nil, err
	}
	var out []string
	for i := 0; i+20 <= len(table); i += 20 {
		nameRVA := uint32(table[i+12]) | uint32(table[i+13])<<8 | uint32(table[i+14])<<16 | uint32(table[i+15])<<24
		if nameRVA == 0 {
			return out, nil
		}
		name, err := read(nameRVA)
		if err != nil {
			return nil, err
		}
		if end := bytes.IndexByte(name, 0); end >= 0 {
			name = name[:end]
		}
		out = append(out, string(name))
	}
	return nil, fmt.Errorf("the import directory of %s has no terminator", path)
}

// delugePEFixture is a minimal PE32+ image importing the named DLLs, for the
// selftest: one section holding the import directory and the names.
func delugePEFixture(dlls ...string) []byte {
	const fileAlign, rva = 0x200, 0x1000
	var sec bytes.Buffer
	names := 20 * (len(dlls) + 1)
	var desc, strs bytes.Buffer
	for _, d := range dlls {
		var e [20]byte
		at := uint32(rva + names + strs.Len())
		e[12], e[13], e[14], e[15] = byte(at), byte(at>>8), byte(at>>16), byte(at>>24)
		desc.Write(e[:])
		strs.WriteString(d)
		strs.WriteByte(0)
	}
	desc.Write(make([]byte, 20))
	sec.Write(desc.Bytes())
	sec.Write(strs.Bytes())
	sec.Write(make([]byte, fileAlign-sec.Len()%fileAlign))
	var b bytes.Buffer
	dos := make([]byte, 0x40)
	dos[0], dos[1], dos[0x3c] = 'M', 'Z', 0x40
	b.Write(dos)
	b.WriteString("PE\x00\x00")
	oh := pe.OptionalHeader64{Magic: 0x20b, SectionAlignment: rva, FileAlignment: fileAlign,
		SizeOfImage: rva + uint32(sec.Len()), SizeOfHeaders: fileAlign, NumberOfRvaAndSizes: 16}
	oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_IMPORT] = pe.DataDirectory{VirtualAddress: rva, Size: uint32(desc.Len())}
	fh := pe.FileHeader{Machine: pe.IMAGE_FILE_MACHINE_AMD64, NumberOfSections: 1,
		SizeOfOptionalHeader: uint16(binary.Size(oh)), Characteristics: 0x2022}
	sh := pe.SectionHeader32{VirtualSize: uint32(sec.Len()), VirtualAddress: rva, SizeOfRawData: uint32(sec.Len()),
		PointerToRawData: fileAlign, Characteristics: 0x40000040}
	copy(sh.Name[:], ".idata")
	binary.Write(&b, binary.LittleEndian, fh)
	binary.Write(&b, binary.LittleEndian, oh)
	binary.Write(&b, binary.LittleEndian, sh)
	b.Write(make([]byte, fileAlign-b.Len()))
	b.Write(sec.Bytes())
	return b.Bytes()
}

// delugeVendorDLL is the member of the vendor's Windows installer that
// carries a DLL libtorrent 1.2.19's Windows wheel imports and does not
// bundle: OpenSSL 1.1's two, as the vendor ships them beside the same wheel.
// Nothing else is taken from the installer. See dlWinEngineDeps.
func delugeVendorDLL(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "libcrypto-1_1-x64.dll":
		return "_internal/libcrypto-1_1-x64.dll", true
	case "libssl-1_1-x64.dll":
		return "_internal/libssl-1_1-x64.dll", true
	}
	return "", false
}

// delugeVendorInstallerURL is the vendor's win64 installer for a version on
// its https mirror, and the file's name: the file the Windows package route
// fetches too, here asked by the version the source tag names.
func delugeVendorInstallerURL(version string) (string, string, error) {
	if !delugeVersionRe.MatchString(version) {
		return "", "", fmt.Errorf("[%s] is not a version", version)
	}
	file := "deluge-" + version + "-win64-setup.exe"
	return "https://ftp.osuosl.org/pub/deluge/windows/" + file, file, nil
}

// delugeAPISet says whether an import is a Windows API set, which the loader
// resolves itself and no directory carries.
func delugeAPISet(name string) bool {
	n := strings.ToLower(name)
	return strings.HasPrefix(n, "api-ms-win-") || strings.HasPrefix(n, "ext-ms-win-")
}

// delugeWinPythonMinors are the Python versions the Windows source route may
// run on, in preference order: those libtorrent 1.2.19 publishes win_amd64
// wheels for, newest first. See delugePins.
var delugeWinPythonMinors = []string{"3.10", "3.9"}

// delugePickPython chooses the interpreter the Windows source route makes its
// venv from, out of the runner tool cache's `Python\<version>\x64\python.exe`
// paths: the newest patch of the first minor in delugeWinPythonMinors that
// any path carries.
//
// ⛔ NOT THE RUNNER'S DEFAULT `python`, which run 39616084687 measured at
// 3.12.10 and for which libtorrent 1.2.19 publishes no wheel. ⚠ A path whose
// version cannot be read is passed over, and finding none is a refusal that
// names what was wanted.
func delugePickPython(paths []string) (string, error) {
	for _, minor := range delugeWinPythonMinors {
		best, bestPatch := "", -1
		for _, p := range paths {
			parts := strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' })
			if len(parts) < 3 || !strings.EqualFold(parts[len(parts)-1], "python.exe") || parts[len(parts)-2] != "x64" {
				continue
			}
			patch, ok := strings.CutPrefix(parts[len(parts)-3], minor+".")
			n, err := strconv.Atoi(patch)
			if !ok || err != nil || strconv.Itoa(n) != patch {
				continue
			}
			if n > bestPatch {
				best, bestPatch = p, n
			}
		}
		if best != "" {
			return best, nil
		}
	}
	return "", fmt.Errorf("the tool cache carries no x64 Python %s, the versions libtorrent 1.2.19 has Windows wheels for",
		strings.Join(delugeWinPythonMinors, " or "))
}
