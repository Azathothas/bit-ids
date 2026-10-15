// trwindows - what the Windows half of the `transmission` adapter decides, as
// functions with no Windows in them, so they can be asked on any host.
//
// CI-11, CLIENT-06. The adapter itself is adapter_transmission_windows.go and
// builds for Windows alone; everything here is the part of it that a Linux gate
// can prove: the settings document it writes, how it reads the build's answer
// and winget's, how it finds the two executables in the vendor's MSI once it is
// laid out, the one command line that lays it out, the service state it reads
// back, and how it takes the RPC verifier back out of the settings the daemon
// rewrites on exit. `bit-check transmission-windows --selftest` asks each of
// them.
//
// ⚠ NOT A SECOND ADAPTER. This file registers nothing, as qbtwindows.go
// registers nothing; the Linux half is adapter_transmission.go.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The two executables the capture runs, by the names the MSI's File table gives
// them. ⭐ Measured on 2026-10-15 by reading `transmission-4.1.3-x64.msi`'s File
// table with msitools: `transmission-daemon.exe` (component
// `exe.transmission.daemon`, feature `Daemon`) and `transmission-remote.exe`
// (`exe.transmission.remote`, feature `CmdTools`), both under
// `INSTALLDIR = ProgramFiles64Folder\Transmission`, beside `libcurl.dll`,
// `libcrypto-3-x64.dll`, `libssl-3-x64.dll` and `zlib.dll` (feature
// `CommonLibs`).
const (
	trWinDaemonExe = "transmission-daemon.exe"
	trWinRemoteExe = "transmission-remote.exe"
)

// trWinWingetID is the package route's identifier in winget's index.
//
// ⭐ MEASURED, NOT GUESSED. `microsoft/winget-pkgs` carries
// `manifests/t/Transmission/Transmission/4.1.3/`, whose x64 installer is
// `https://github.com/transmission/transmission/releases/download/4.1.3/transmission-4.1.3-x64.msi`
// with InstallerSha256 `C8EA492D...21741F8`: the digest GitHub's listing states
// for the release route's asset. InstallerType `wix`, Scope `machine`,
// DefaultInstallLocation `%ProgramFiles%/Transmission`. Read through
// `AGENTS.md` rule 8's route on 2026-10-15.
const trWinWingetID = "Transmission.Transmission"

// trWinMSIFeatures is what the package route asks the MSI to install.
//
// ⛔ A DEFAULT INSTALL CARRIES NEITHER EXECUTABLE THE CAPTURE RUNS. The MSI's
// Feature table, read on 2026-10-15: `CommonLibs` and `QtClient` are level 1,
// and `Daemon`, `CmdTools` and `WebUi` are level 2, while the Property table
// sets no `INSTALLLEVEL`, so Windows Installer's default of 1 applies and a
// silent install places the Qt client alone. So the route names its features,
// and names `CommonLibs` too, because the daemon links the DLLs it carries.
// ⚠ A misspelt feature is a loud failure: msiexec refuses the install.
const trWinMSIFeatures = "CommonLibs,Daemon,CmdTools"

// trWinServiceName is the service the MSI installs with the daemon.
//
// ⛔ AND STARTS. The ServiceInstall table registers `Transmission`, own
// process, start type 2 (automatic), as `NT AUTHORITY\LocalService`, in the
// daemon's own component; the ServiceControl table's event 163 is start on
// install, stop on install, stop and delete on uninstall. So the package route
// leaves a second daemon running with the vendor's defaults, which is the
// distribution's service on Linux in another form, and the route stops and
// disables it for the reason the Linux half gives.
const trWinServiceName = "Transmission"

// trWinSettings is the whole settings document the daemon reads from
// `--config-dir`, written rather than patched.
//
// ⛔ THE LINUX HALF'S DOCUMENT, KEY FOR KEY AND IN ITS ORDER. DHT, peer
// exchange, local peer discovery, uTP and port forwarding off, no encryption
// preference, RPC on loopback alone with authentication off, and the peer port
// the capture asked for. ⚠ Two things differ, both because this is Windows:
//
//   - every value is JSON-encoded. The Linux half interpolates its download
//     directory raw, which is valid JSON for a POSIX path and is NOT for
//     `C:\work\...`, whose backslashes are escapes; the selftest plants exactly
//     that;
//   - the RPC port is the caller's, a free loopback port drawn per session,
//     rather than 9091, because the package route's own service is installed to
//     listen there.
//
// ⚠ A peer port that is not a decimal number is refused rather than written,
// where the Linux half would write it into the document as it came.
func trWinSettings(downloads, peerPort string, rpcPort int) (string, error) {
	if peerPort != "" && peerPort != "0" {
		if n, err := strconv.Atoi(peerPort); err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != peerPort {
			return "", fmt.Errorf("peer port [%s] is not a port", peerPort)
		}
	}
	if rpcPort < 1 || rpcPort > 65535 {
		return "", fmt.Errorf("RPC port %d is not a port", rpcPort)
	}
	str := func(s string) string {
		b, _ := json.Marshal(s)
		return string(b)
	}
	type kv struct{ k, v string }
	doc := []kv{
		{"dht-enabled", "false"},
		{"pex-enabled", "false"},
		{"lpd-enabled", "false"},
		{"utp-enabled", "false"},
		{"port-forwarding-enabled", "false"},
		{"encryption", "0"},
		{"download-dir", str(downloads)},
		{"incomplete-dir-enabled", "false"},
		{"rpc-enabled", "true"},
		{"rpc-bind-address", str("127.0.0.1")},
		{"rpc-port", strconv.Itoa(rpcPort)},
		{"rpc-authentication-required", "false"},
		{"rpc-whitelist", str("127.0.0.1")},
		{"rpc-host-whitelist-enabled", "false"},
		{"ratio-limit-enabled", "false"},
	}
	if peerPort != "" && peerPort != "0" {
		doc = append(doc, kv{"peer-port", peerPort})
	}
	doc = append(doc, kv{"peer-port-random-on-start", "false"})
	var b strings.Builder
	b.WriteString("{\n")
	for i, e := range doc {
		sep := ","
		if i == len(doc)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "  %s: %s%s\n", str(e.k), e.v, sep)
	}
	b.WriteString("}\n")
	return b.String(), nil
}

// trWinVersionRe is a version as the build prints one.
var trWinVersionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}[A-Za-z0-9.+-]*$`)

// trWinParseVersion reads `transmission-daemon --version`.
//
// ⛔ THE BUILD SPEAKING, AS ON LINUX. `daemon.cc` prints `"%s %s\n", MyName,
// LONG_VERSION_STRING` to stderr, which for this MSI's build is
// `transmission-daemon 4.1.3 (838877323f)`: the ProductName in the MSI's
// Property table carries the same `(838877323f)`. The second field is taken, and
// the build identifier beside it stays on stderr, because a version and a build
// are two facts.
// ⚠ STRICTER THAN THE LINUX HALF, which takes the second field of any first
// line: here the first field must name the daemon and the second must be shaped
// as a version, so a usage text or an error message is refused rather than read.
func trWinParseVersion(output string) (string, error) {
	line, _, _ := strings.Cut(strings.ReplaceAll(output, "\r", ""), "\n")
	line = strings.TrimSpace(line)
	f := strings.Fields(line)
	if len(f) < 2 || f[0] != "transmission-daemon" || !trWinVersionRe.MatchString(f[1]) {
		return "", fmt.Errorf("transmission-daemon reported no parseable version: [%s]", line)
	}
	return f[1], nil
}

// trWinWingetVersion reads the version winget's index states, out of what
// `winget show` printed.
//
// ⚠ IT IS THE INDEX'S STATEMENT AND NEVER THE VERSION THIS ROUTE REPORTS. The
// route compares it with what the installed build answers, so an index that
// installed something other than what it named is a refusal rather than a
// record. Only a top-level `Version:` line counts: `winget show` indents the
// installer's own keys, and a progress spinner leaves carriage returns that end
// with the real text.
func trWinWingetVersion(show string) (string, error) {
	found := ""
	for _, l := range strings.Split(show, "\n") {
		l = strings.TrimRight(l, "\r")
		if i := strings.LastIndex(l, "\r"); i >= 0 {
			l = l[i+1:]
		}
		if l == "" || l[0] == ' ' || l[0] == '\t' {
			continue
		}
		key, val, ok := strings.Cut(l, ":")
		if !ok || key != "Version" {
			continue
		}
		val = strings.TrimSpace(val)
		if found != "" && found != val {
			return "", fmt.Errorf("winget's index named two versions, %s and %s", found, val)
		}
		found = val
	}
	if found == "" || !trWinVersionRe.MatchString(found) {
		return "", fmt.Errorf("winget's index named no version: [%s]", found)
	}
	return found, nil
}

// trWinFindPayload finds the two executables an administrative image of the
// MSI carries, and answers the daemon and the remote.
//
// ⛔ EXACTLY ONE DAEMON, AND ITS REMOTE BESIDE IT. An administrative install
// lays the image out by the MSI's Directory table, as `PFiles\Transmission\`
// under TARGETDIR, and writes a copy of the package at TARGETDIR's root. Two
// daemons is a refusal rather than a first match, for the reason `select-asset`
// refuses two assets; a remote anywhere but beside the daemon is refused because
// the two are one build and the MSI places them together.
// ⚠ Four levels deep at most, as qbtWinFindPayload walks.
func trWinFindPayload(root string) (string, string, error) {
	var found []string
	base := strings.Count(filepath.Clean(root), string(filepath.Separator))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.Count(filepath.Clean(path), string(filepath.Separator))-base >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), trWinDaemonExe) {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	switch len(found) {
	case 0:
		return "", "", fmt.Errorf("no %s under %s", trWinDaemonExe, root)
	case 1:
	default:
		return "", "", fmt.Errorf("%d copies of %s under %s: %s", len(found), trWinDaemonExe, root, strings.Join(found, ", "))
	}
	remote := filepath.Join(filepath.Dir(found[0]), trWinRemoteExe)
	if !isRegular(remote) {
		return "", "", fmt.Errorf("no %s beside %s", trWinRemoteExe, found[0])
	}
	return found[0], remote, nil
}

// trWinMsiexecCmdLine is the one command line the release route lays the MSI
// out with: an administrative installation, quiet, into target, logged.
//
// ⛔ WRITTEN WHOLE, NOT LEFT TO AN ARGUMENT QUOTER. msiexec parses its own
// command line and wants `PROPERTY="value"`, which Go's quoting of an argument
// holding a space would write as `"PROPERTY=value"`. ⚠ A path holding a double
// quote cannot be written in that form at all and is refused.
func trWinMsiexecCmdLine(msiexec, msi, target, log string) (string, error) {
	for _, p := range []string{msiexec, msi, target, log} {
		if p == "" || strings.ContainsAny(p, "\"\r\n") {
			return "", fmt.Errorf("path [%s] cannot be written on msiexec's command line", p)
		}
	}
	return fmt.Sprintf(`"%s" /a "%s" /qn TARGETDIR="%s" /L*v "%s"`, msiexec, msi, target, log), nil
}

// trWinServiceState reads the STATE line of `sc.exe query <name>`, which is
// `STATE : 4  RUNNING` in the tool's own layout, and answers the word. A service
// that does not exist answers "", and the caller says so.
func trWinServiceState(query string) string {
	for _, l := range strings.Split(query, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(strings.TrimRight(l, "\r")), ":")
		if !ok || strings.TrimSpace(key) != "STATE" {
			continue
		}
		f := strings.Fields(val)
		if len(f) >= 2 {
			return f[1]
		}
	}
	return ""
}

// trWinScrubSettings takes every member naming a password out of the settings
// document the daemon rewrote on exit, and answers how many it took. ⚠ It
// answers deleted=true when the document was not a JSON object it could read,
// because then the whole file is removed.
//
// ⛔ THE LINUX HALF'S RULE-12 SCRUB, WITHOUT `jq`, WHICH A WINDOWS RUNNER DOES
// NOT CARRY. Measured on `capture-client` run 38600252161: 4.1.3 saves
// `"rpc-password": "{<salted hash>"` at exit even with authentication off. The
// members are kept in the order the build wrote them and the file stays valid
// JSON; one the scrub cannot parse is deleted, as the Linux half deletes it when
// `jq` is absent, rather than edited into something that might not parse.
// ⚠ BROADER THAN THE LINUX HALF, which deletes `rpc-password` alone: any member
// whose name holds `password` goes, as qbtWinScrub matches by key.
func trWinScrubSettings(path string) (removed int, deleted bool, err error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	type member struct {
		key string
		val json.RawMessage
	}
	var members []member
	parse := func() bool {
		dec := json.NewDecoder(bytes.NewReader(body))
		if t, err := dec.Token(); err != nil || t != json.Delim('{') {
			return false
		}
		for dec.More() {
			t, err := dec.Token()
			if err != nil {
				return false
			}
			k, ok := t.(string)
			if !ok {
				return false
			}
			var v json.RawMessage
			if dec.Decode(&v) != nil {
				return false
			}
			members = append(members, member{k, v})
		}
		if t, err := dec.Token(); err != nil || t != json.Delim('}') {
			return false
		}
		_, err := dec.Token()
		return err != nil
	}
	if !parse() {
		if err := os.Remove(path); err != nil {
			return 0, false, err
		}
		return 0, true, nil
	}
	var b bytes.Buffer
	b.WriteString("{\n")
	kept := 0
	for _, m := range members {
		if strings.Contains(strings.ToLower(m.key), "password") {
			removed++
			continue
		}
		if kept > 0 {
			b.WriteString(",\n")
		}
		k, _ := json.Marshal(m.key)
		var v bytes.Buffer
		if json.Compact(&v, m.val) != nil {
			v.Reset()
			v.Write(m.val)
		}
		fmt.Fprintf(&b, "    %s: %s", k, v.Bytes())
		kept++
	}
	b.WriteString("\n}\n")
	if removed == 0 {
		return 0, false, nil
	}
	if err := os.WriteFile(path+".new", b.Bytes(), 0o644); err != nil {
		return 0, false, err
	}
	if err := os.Rename(path+".new", path); err != nil {
		os.Remove(path + ".new")
		return 0, false, err
	}
	return removed, false, nil
}

// -- the selftest -------------------------------------------------------------

// transmissionWindows is `bit-check transmission-windows --selftest`: every
// function above asked over a case whose answer is known, on any host.
func transmissionWindows(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "transmission-windows: usage: bit-check transmission-windows --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "transmission-windows: FAIL %s\n", name)
		}
	}

	// The settings: every surface off, RPC on loopback alone, and a Windows
	// path that survives being read back as JSON.
	doc, err := trWinSettings(`C:\work\client\downloads`, "51413", 40123)
	var got map[string]any
	parsed := err == nil && json.Unmarshal([]byte(doc), &got) == nil
	check("the settings are a JSON document", parsed)
	for k, want := range map[string]any{
		"dht-enabled": false, "pex-enabled": false, "lpd-enabled": false, "utp-enabled": false,
		"port-forwarding-enabled": false, "rpc-enabled": true, "rpc-bind-address": "127.0.0.1",
		"rpc-whitelist": "127.0.0.1", "rpc-authentication-required": false,
		"rpc-port": float64(40123), "peer-port": float64(51413), "peer-port-random-on-start": false,
		"download-dir": `C:\work\client\downloads`, "encryption": float64(0),
	} {
		check(fmt.Sprintf("the settings hold %s=%v", k, want), parsed && got[k] == want)
	}
	check("the settings keep the Linux half's order", strings.Index(doc, `"dht-enabled"`) < strings.Index(doc, `"pex-enabled"`) &&
		strings.Index(doc, `"rpc-port"`) < strings.Index(doc, `"peer-port"`))
	d0, err0 := trWinSettings("d", "0", 1)
	check("peer port 0 leaves the port to the build", err0 == nil && !strings.Contains(d0, `"peer-port":`))
	for _, bad := range []string{"51413 ", "x", "70000", "-1", "0051413"} {
		_, err := trWinSettings("d", bad, 1)
		check("the peer port ["+bad+"] is refused", err != nil)
	}
	_, err = trWinSettings("d", "1", 0)
	check("an RPC port of 0 is refused", err != nil)

	// The build's answer.
	for in, want := range map[string]string{
		"transmission-daemon 4.1.3 (838877323f)\r\n": "4.1.3",
		"transmission-daemon 4.1.3 (838877323f)":     "4.1.3",
		"transmission-daemon 4.2.0-beta.1 (abc)\n":   "4.2.0-beta.1",
	} {
		v, err := trWinParseVersion(in)
		check("the answer ["+strings.TrimSpace(in)+"] reads as "+want, err == nil && v == want)
	}
	for _, in := range []string{"", "transmission-daemon", "Transmission 4.1.3 (838877323f)",
		"transmission-daemon: unrecognized option", "transmission-daemon (838877323f) 4.1.3"} {
		_, err := trWinParseVersion(in)
		check("the answer ["+in+"] is refused", err != nil)
	}

	// winget's statement.
	show := "## winget show --id Transmission.Transmission\r\n   - \r   \\ \rFound Transmission [Transmission.Transmission]\r\n" +
		"Version: 4.1.3\r\nPublisher: Transmission Project\r\nInstaller:\r\n  Installer Type: wix\r\n  Version: 9.9.9\r\n"
	v, err := trWinWingetVersion(show)
	check("winget's index version is read, its installer's indented keys are not", err == nil && v == "4.1.3")
	_, err = trWinWingetVersion("Found Transmission\nPublisher: x\n")
	check("an index that names no version is refused", err != nil)
	_, err = trWinWingetVersion("Version: 4.1.3\nVersion: 4.1.2\n")
	check("an index that names two versions is refused", err != nil)

	// The command line.
	cl, err := trWinMsiexecCmdLine(`C:\Windows\System32\msiexec.exe`, `C:\w\t.msi`, `C:\a b\opt`, `C:\w\msi.log`)
	check("msiexec's command line quotes the value, not the property",
		err == nil && cl == `"C:\Windows\System32\msiexec.exe" /a "C:\w\t.msi" /qn TARGETDIR="C:\a b\opt" /L*v "C:\w\msi.log"`)
	_, err = trWinMsiexecCmdLine("m", `C:\w\t".msi`, "t", "l")
	check("a path holding a double quote is refused", err != nil)

	// The service.
	check("a running service reads as RUNNING", trWinServiceState(
		"\r\nSERVICE_NAME: Transmission\r\n        TYPE               : 10  WIN32_OWN_PROCESS\r\n        STATE              : 4  RUNNING\r\n") == "RUNNING")
	check("a stopped service reads as STOPPED", trWinServiceState("        STATE              : 1  STOPPED\n") == "STOPPED")
	check("a service that does not exist reads as nothing",
		trWinServiceState("[SC] EnumQueryServicesStatus:OpenService FAILED 1060:\n") == "")

	// The payload, and the scrub, over a scratch tree.
	work, err := os.MkdirTemp("", "trwin-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "transmission-windows: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	mk := func(rel, body string) {
		p := filepath.Join(work, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	mk("one/t.msi", "x")
	mk("one/PFiles/Transmission/transmission-daemon.exe", "x")
	mk("one/PFiles/Transmission/transmission-remote.exe", "x")
	dmn, rmt, err := trWinFindPayload(filepath.Join(work, "one"))
	check("the daemon and its remote are found in the image",
		err == nil && dmn == filepath.Join(work, "one", "PFiles", "Transmission", trWinDaemonExe) &&
			rmt == filepath.Join(work, "one", "PFiles", "Transmission", trWinRemoteExe))
	mk("two/a/transmission-daemon.exe", "x")
	mk("two/a/transmission-remote.exe", "x")
	mk("two/b/TRANSMISSION-DAEMON.EXE", "x")
	_, _, err = trWinFindPayload(filepath.Join(work, "two"))
	check("two daemons are refused, not chosen between", err != nil)
	mk("apart/a/transmission-daemon.exe", "x")
	mk("apart/b/transmission-remote.exe", "x")
	_, _, err = trWinFindPayload(filepath.Join(work, "apart"))
	check("a remote not beside the daemon is refused", err != nil)
	mk("none/readme.txt", "x")
	_, _, err = trWinFindPayload(filepath.Join(work, "none"))
	check("no daemon is refused", err != nil)
	mk("deep/a/b/c/d/transmission-daemon.exe", "x")
	mk("deep/a/b/c/d/transmission-remote.exe", "x")
	_, _, err = trWinFindPayload(filepath.Join(work, "deep"))
	check("a daemon deeper than four levels is not walked to", err != nil)

	settings := filepath.Join(work, "settings.json")
	mk("settings.json", "{\n    \"dht-enabled\": false,\n    \"rpc-password\": \"{0123456789abcdef0123456789abcdef01234567Ab3dE6f9\",\n"+
		"    \"rpc-port\": 40123,\n    \"rpc-whitelist\": \"127.0.0.1\",\n    \"proxy-Password\": \"x\",\n    \"speed-limit\": {\"a\": [1, 2]}\n}\n")
	n, del, err := trWinScrubSettings(settings)
	left, _ := os.ReadFile(settings)
	var back map[string]any
	check("the scrub removes both password members", err == nil && !del && n == 2)
	check("the scrubbed settings are still JSON", json.Unmarshal(left, &back) == nil)
	check("the scrub keeps everything else in order", string(left) ==
		"{\n    \"dht-enabled\": false,\n    \"rpc-port\": 40123,\n    \"rpc-whitelist\": \"127.0.0.1\",\n    \"speed-limit\": {\"a\":[1,2]}\n}\n")
	check("no verifier is left", !strings.Contains(string(left), "rpc-password"))
	mk("broken.json", "{\n    \"rpc-password\": \"{abc\",\n")
	_, del, err = trWinScrubSettings(filepath.Join(work, "broken.json"))
	check("settings the scrub cannot parse are deleted, not edited", err == nil && del && !isRegular(filepath.Join(work, "broken.json")))
	mk("clean.json", "{\"a\": 1}\n")
	n, del, err = trWinScrubSettings(filepath.Join(work, "clean.json"))
	clean, _ := os.ReadFile(filepath.Join(work, "clean.json"))
	check("settings with no password are left as written", err == nil && n == 0 && !del && string(clean) == "{\"a\": 1}\n")
	_, _, err = trWinScrubSettings(filepath.Join(work, "no-such-settings.json"))
	check("settings that were never written scrub clean", err == nil)

	fmt.Printf(`{"schema":"transmission-windows-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
