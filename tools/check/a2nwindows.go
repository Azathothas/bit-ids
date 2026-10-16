// a2nwindows - what the Windows half of the `aria2-next` adapter decides, as
// functions with no Windows in them, so they can be asked on any host; and the
// option read-back both halves share.
//
// CI-11, CLIENT-14. The adapter itself is adapter_aria2-next_windows.go and
// builds for Windows alone; everything here is the part of it a Linux gate can
// prove: the command line `start` passes, how it reads the build's answer, how
// it reads Scoop's manifest, where it finds the build Scoop placed, and the
// JSON-RPC bodies and answers. `bit-check aria2-next-windows --selftest` asks
// each of them.
//
// ⛔ THE READ-BACK IS ONE FUNCTION FOR BOTH HALVES. aria2NextOptionsRefusal is
// what `start` on Linux and on Windows asks of the build's own options dump, so
// the switches it names, the families it sweeps and the options it has judged
// harmless when `true` are one list, not two that drift. It moved here from
// adapter_aria2-next.go unchanged, message for message, because that file
// builds off Windows only.
//
// ⚠ NOT A SECOND ADAPTER. This file registers nothing; the Linux half is
// adapter_aria2-next.go.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// -- shared with the Linux half -----------------------------------------------

// aria2NextSwitches is every discovery switch `start` passes, each as
// `--<switch>=false`, and requires the build to report as `false`.
//
// ⛔ FIVE, NOT THE THREE THE CONTRACT NAMES: DHT, local discovery and peer
// exchange, then `bt-port-mapping` (UPnP and NAT-PMP, measured bound to UDP 1900
// on 2026-09-09) and `ed2k-port-mapping` (3.0.0's UPnP for the ED2K ports, on in
// 3.0.2). adapter_aria2-next.go gives the measurements with its flags.
var aria2NextSwitches = []string{"enable-dht", "bt-enable-lpd", "enable-peer-exchange", "bt-port-mapping", "ed2k-port-mapping"}

// The option families a discovery switch belongs to, and the options this file
// has judged harmless when the build reports them `true`: thirteen, each read
// off 3.0.2's own dump, and the same thirteen 3.0.3's Linux build reports on
// 2026-10-15.
var (
	aria2NextKeyRe     = regexp.MustCompile(`"[^"]+":`)
	aria2NextFamilyRe  = regexp.MustCompile(`^(enable-dht[a-z0-9-]*|[a-z0-9-]*port-mapping|[a-z0-9-]*-lpd|[a-z0-9-]*-lsd|[a-z0-9-]*peer-exchange)$`)
	aria2NextTrueRe    = regexp.MustCompile(`"[^"]+": *"?true`)
	aria2NextTrueKeyRe = regexp.MustCompile(`^"([^"]+)".*`)
	aria2NextHarmless  = " allow-overwrite auto-file-renaming check-certificate enable-color enable-http-keep-alive enable-rpc follow-metalink follow-torrent http-accept-gzip keep-unfinished-download-result rpc-save-upload-metadata save-not-found show-console-readout "
)

// lines is a file's lines, as `grep` reads them.
func lines(b []byte) []string {
	var out []string
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Buffer(make([]byte, 0, 64*1024), 1<<30)
	for s.Scan() {
		out = append(out, s.Text())
	}
	return out
}

// aria2NextOptionsRefusal reads the build's `aria2.getGlobalOption` answer and
// answers the refusal it earns, or "" when every check holds.
//
// ⭐ A COMMAND LINE REPORTS WHAT WAS PASSED; THIS REPORTS WHAT TOOK EFFECT. The
// build accepts an option it does not have and skips it, which is the measured
// behaviour of `--enable-dht6`.
//
//   - ⛔ every switch in aria2NextSwitches is reported, and `false`. A switch
//     the build stopped reporting is one nothing here can show took effect;
//   - ⛔ every KEY of those families the build reports, named or not, reads
//     `false` too. `ed2k-port-mapping` was on in 3.0.2 and named nowhere. The
//     keys are taken off the dump and each is asked for the exact pair, so a
//     value this parse cannot read refuses rather than passing unread;
//   - ⛔ every option reported `true` is in aria2NextHarmless, so a new default
//     reported as `true` refuses whatever it is called. ⚠ A surface turned on
//     with another value, a port or `auto`, is not caught here;
//   - ⛔ and the peer port the observer watches is the one the build reports,
//     unless it was left to the build with `0`.
func aria2NextOptionsRefusal(options []byte, port string) string {
	has := func(key, value string) bool {
		return bytes.Contains(options, []byte(`"`+key+`":"`+value+`"`))
	}
	for _, sw := range aria2NextSwitches {
		if !has(sw, "false") {
			return "aria2-next reports " + sw + " is not false; a discovery surface is live"
		}
	}
	for _, line := range lines(options) {
		for _, m := range aria2NextKeyRe.FindAllString(line, -1) {
			key := strings.NewReplacer(`"`, "", ":", "").Replace(m)
			if key == "" || !aria2NextFamilyRe.MatchString(key) {
				continue
			}
			if !has(key, "false") {
				return "aria2-next reports " + key + " and not as false; a discovery surface is live"
			}
		}
	}
	for _, line := range lines(options) {
		for _, m := range aria2NextTrueRe.FindAllString(line, -1) {
			key := strings.Trim(aria2NextTrueKeyRe.ReplaceAllString(m, "$1"), " \t")
			if key == "" || strings.Contains(aria2NextHarmless, " "+key+" ") {
				continue
			}
			return "aria2-next reports " + key + " as true, and this file has not judged it; a discovery surface may be live"
		}
	}
	if port != "0" && !has("listen-port", port) {
		return "aria2-next reports a listen port other than " + port
	}
	return ""
}

// -- the Windows half's decisions ---------------------------------------------

// a2nWinExe is the build's executable on Windows, as both routes place it:
// install-rootless `--as` names it on the release route, and Scoop's manifest
// renames the asset to it with `#/aria2-next.exe` on the package route.
const a2nWinExe = "aria2-next.exe"

// The package route's index: Scoop's `main` bucket, app `aria2-next`.
//
// ⭐ MEASURED, NOT GUESSED, through `AGENTS.md` rule 8's route on 2026-10-15.
// `ScoopInstaller/Main` carries `bucket/aria2-next.json`. It named 3.0.3 from
// commit 1359d92 (2026-10-14T08:38Z) to commit 2384d7c (2026-10-15T08:38Z), with
// the vendor's own asset URL and hash `b2f2f7a9...d2be9465`, the digest GitHub's
// listing and the vendor's checksums document both state. Since 2384d7c it names
// 3.0.4, the vendor's newest stable. ⛔ winget's `AnInsomniacy.Aria2Next` stops
// at 2.8.3: its 3.0.3 pull request, #452639, was closed unmerged. Chocolatey's
// community feed answers no `aria2-next` at all.
const (
	a2nWinScoopBucket = "main"
	a2nWinScoopApp    = "aria2-next"
)

// a2nWinScoopInstaller is Scoop's own installer script, which is what
// `https://get.scoop.sh` answers with a 301 to; both served the same bytes on
// 2026-10-15, sha256 `94f983b1...f2512`. ⚠ It is the installer of the package
// manager and not of the build: the build's bytes come through Scoop's own
// download, checked against the manifest's hash.
const a2nWinScoopInstaller = "https://raw.githubusercontent.com/ScoopInstaller/Install/master/install.ps1"

// a2nWinScoopInstallerUnpublished is the measured reason install-rootless is
// given for fetching that script with no digest to check it against.
const a2nWinScoopInstallerUnpublished = "Scoop's installer is served from its repository's master branch and publishes no digest of itself"

// a2nWinPackageFormat is how each route delivers the build, as `Build.package`
// spells it.
//
// ⚠ ONE VALUE FOR BOTH ROUTES, because both deliver one form: the vendor's bare
// PE executable, placed as a file, with no installer and no archive between it
// and the host. The Linux half calls its two routes' bare executables
// `elf-binary` for the same reason.
func a2nWinPackageFormat(route string) (string, bool) {
	switch route {
	case "release", "package":
		return "pe-binary", true
	}
	return "", false
}

// a2nWinPortOK says whether s is a decimal TCP port, as the build is to be told.
func a2nWinPortOK(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 65535 && strconv.Itoa(n) == s
}

var a2nWinTokenRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// a2nWinFlags is the whole command line `start` gives the build.
//
// ⛔ THE LINUX HALF'S FLAGS, IN ITS ORDER, because the Windows build reads the
// same options: all five discovery switches off, `--enable-dht6` absent (the
// build skips it), `--bt-require-crypto` absent (it set nothing), RPC on
// loopback with a per-run secret, and the peer port only when one was asked for.
// ⚠ Read rather than run on 2026-10-15: every option key the Linux 3.0.3 build
// reports over RPC is a string in the Windows 3.0.3 executable, and its
// discovery-family strings are the same seven as Linux's.
//
// ⚠ A peer port that is not a decimal number, an RPC port that is not one, and
// a token that is not 32 hex digits are refused rather than passed.
func a2nWinFlags(downloads, peerPort, rpcPort, token string) ([]string, error) {
	if peerPort != "0" && !a2nWinPortOK(peerPort) {
		return nil, fmt.Errorf("peer port [%s] is not a port", peerPort)
	}
	if !a2nWinPortOK(rpcPort) {
		return nil, fmt.Errorf("RPC port [%s] is not a port", rpcPort)
	}
	if !a2nWinTokenRe.MatchString(token) {
		return nil, fmt.Errorf("the RPC token is not 32 hex digits")
	}
	if downloads == "" {
		return nil, fmt.Errorf("no download directory")
	}
	flags := []string{
		"--enable-rpc=true",
		"--rpc-listen-port=" + rpcPort,
		"--rpc-listen-all=false",
		"--rpc-secret=" + token,
	}
	for _, sw := range aria2NextSwitches {
		flags = append(flags, "--"+sw+"=false")
	}
	flags = append(flags,
		"--dir="+downloads,
		"--seed-time=0",
		"--allow-overwrite=true",
		"--console-log-level=info",
		"--summary-interval=0",
	)
	if peerPort != "0" {
		flags = append(flags, "--listen-port="+peerPort)
	}
	return flags, nil
}

var a2nWinVersionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}[A-Za-z0-9.+-]*$`)

// a2nWinParseVersion reads `aria2-next --version`.
//
// ⛔ ANCHORED ON THE `version` TOKEN, as the Linux half is: 2.7.4 printed `Aria2
// Next version 2.7.4` and 3.0.3 prints `aria2-next version 3.0.3`, so the
// product name is prose whose word count is the vendor's to change.
// ⚠ STRICTER THAN THE LINUX HALF, which takes whatever follows the token: here
// the first line must name aria2 and the field must be shaped as a version, so a
// usage text or an error message is refused rather than read.
func a2nWinParseVersion(output string) (string, error) {
	line, _, _ := strings.Cut(strings.ReplaceAll(output, "\r", ""), "\n")
	line = strings.TrimSpace(line)
	f := strings.Fields(line)
	if !strings.Contains(strings.ToLower(line), "aria2") {
		return "", fmt.Errorf("aria2-next reported no parseable version: [%s]", line)
	}
	for i := 0; i+1 < len(f); i++ {
		if f[i] == "version" && a2nWinVersionRe.MatchString(f[i+1]) {
			return f[i+1], nil
		}
	}
	return "", fmt.Errorf("aria2-next reported no parseable version: [%s]", line)
}

// a2nWinScoopEntry is what Scoop's manifest says it will install.
type a2nWinScoopEntry struct {
	version, url, hash string
}

// a2nWinScoopManifest reads `bucket/aria2-next.json` for the x64 build.
//
// ⛔ THE INDEX'S STATEMENT, AND IT HAS TO BE ONE STATEMENT. The version, the one
// x64 URL and the one hash; the URL's `#/aria2-next.exe` is Scoop's rename and
// is dropped, so what is recorded is where the bytes come from. ⚠ Refused:
//
//   - a URL that is not https, or a list of URLs, because the route would not
//     know which one Scoop fetched;
//   - a URL whose file is not `aria2-next-<version>-windows-x86_64.exe` for the
//     version the manifest names, because an index whose two fields disagree
//     says two things about one build;
//   - a hash that is not SHA-256, with or without Scoop's `sha256:` prefix.
func a2nWinScoopManifest(body []byte) (a2nWinScoopEntry, error) {
	var m struct {
		Version      string `json:"version"`
		Architecture map[string]struct {
			URL  json.RawMessage `json:"url"`
			Hash json.RawMessage `json:"hash"`
		} `json:"architecture"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest is not JSON: %v", err)
	}
	if !a2nWinVersionRe.MatchString(m.Version) {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names no version: [%s]", m.Version)
	}
	x64, ok := m.Architecture["64bit"]
	if !ok {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names no 64bit build")
	}
	var url, hash string
	if json.Unmarshal(x64.URL, &url) != nil {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names its 64bit url as something other than one string")
	}
	if json.Unmarshal(x64.Hash, &hash) != nil {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names its 64bit hash as something other than one string")
	}
	url, _, _ = strings.Cut(url, "#")
	if !strings.HasPrefix(url, "https://") {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names no https url: [%s]", url)
	}
	if want := "aria2-next-" + m.Version + "-windows-x86_64.exe"; path.Base(url) != want {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names version %s and the url %s, which is not %s", m.Version, url, want)
	}
	hash = strings.ToLower(strings.TrimPrefix(strings.ToLower(hash), "sha256:"))
	if !lowerHex64.MatchString(hash) {
		return a2nWinScoopEntry{}, fmt.Errorf("Scoop's manifest names no SHA-256 hash: [%s]", hash)
	}
	return a2nWinScoopEntry{version: m.Version, url: url, hash: hash}, nil
}

// a2nWinScoopBuild finds the build Scoop placed under `<scoop>\apps\aria2-next`
// and answers its version directory's name and the executable in it.
//
// ⛔ THE VERSION DIRECTORY, NEVER `current` AND NEVER THE SHIM. Scoop places an
// app in a directory named by its version and points `current` at it with a
// junction, and puts a shim of the same name on PATH. A shim is Scoop's own
// executable, so a lookup through PATH would hand the firewall and `version` a
// program that is not the build; and a rule names the path it was given.
// ⚠ Exactly one version directory: two is a refusal rather than a choice,
// for the reason `select-asset` refuses two assets.
func a2nWinScoopBuild(appDir string) (string, string, error) {
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return "", "", fmt.Errorf("Scoop placed no %s: %v", appDir, err)
	}
	var found []string
	for _, e := range entries {
		if e.Name() == "current" || !e.IsDir() {
			continue
		}
		found = append(found, e.Name())
	}
	switch len(found) {
	case 0:
		return "", "", fmt.Errorf("no version directory under %s", appDir)
	case 1:
	default:
		return "", "", fmt.Errorf("%d version directories under %s: %s", len(found), appDir, strings.Join(found, ", "))
	}
	exe := filepath.Join(appDir, found[0], a2nWinExe)
	if !isRegular(exe) {
		return "", "", fmt.Errorf("no %s in %s", a2nWinExe, filepath.Join(appDir, found[0]))
	}
	return found[0], exe, nil
}

// a2nWinRPCBody is one JSON-RPC request, the Linux half's shape. ⚠ Every
// parameter is JSON-encoded, the token among them.
func a2nWinRPCBody(method, token string, params ...string) string {
	all := append([]string{"token:" + token}, params...)
	p, _ := json.Marshal(all)
	m, _ := json.Marshal(method)
	return `{"jsonrpc":"2.0","id":"bit-ids","method":` + string(m) + `,"params":` + string(p) + `}`
}

// a2nWinRPCAnswer reads a JSON-RPC answer's `result`, whatever its shape.
//
// ⚠ A JSON-RPC ERROR ARRIVES WITH HTTP 200 AND AN `error` MEMBER, so the status
// is never the answer: an `error` member is a refusal, and so is an answer with
// no result at all.
func a2nWinRPCAnswer(body []byte) (json.RawMessage, error) {
	var r struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("the answer is not JSON-RPC: %v", err)
	}
	if len(r.Error) > 0 && string(r.Error) != "null" {
		return nil, fmt.Errorf("the build refused: %s", r.Error)
	}
	if len(r.Result) == 0 || string(r.Result) == "null" {
		return nil, fmt.Errorf("the answer carries no result")
	}
	return r.Result, nil
}

// a2nWinRPCResult reads a JSON-RPC answer whose `result` is one string, which
// is what `aria2.addTorrent` (the gid) and `aria2.shutdown` (`OK`) answer.
func a2nWinRPCResult(body []byte) (string, error) {
	raw, err := a2nWinRPCAnswer(body)
	if err != nil {
		return "", err
	}
	var s string
	if json.Unmarshal(raw, &s) != nil || s == "" {
		return "", fmt.Errorf("the answer carries no result string")
	}
	return s, nil
}

// -- the selftest -------------------------------------------------------------

// aria2NextWindows is `bit-check aria2-next-windows --selftest`: every function
// above asked over a case whose answer is known, on any host.
func aria2NextWindows(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "aria2-next-windows: usage: bit-check aria2-next-windows --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "aria2-next-windows: FAIL %s\n", name)
		}
	}
	// ⚠ Built rather than written: a long hex literal is what check-no-secrets
	// refuses, and these are shapes, not values.
	tok := strings.Repeat("0123456789abcdef", 2)
	sum := strings.Repeat("b2f2", 16)

	// The command line: the Linux half's flags, in its order.
	flags, err := a2nWinFlags(`C:\a b\client\downloads`, "51413", "6800", tok)
	check("the flags are built", err == nil)
	want := []string{"--enable-rpc=true", "--rpc-listen-port=6800", "--rpc-listen-all=false", "--rpc-secret=" + tok,
		"--enable-dht=false", "--bt-enable-lpd=false", "--enable-peer-exchange=false", "--bt-port-mapping=false",
		"--ed2k-port-mapping=false", `--dir=C:\a b\client\downloads`, "--seed-time=0", "--allow-overwrite=true",
		"--console-log-level=info", "--summary-interval=0", "--listen-port=51413"}
	check("the flags are the Linux half's, in its order", strings.Join(flags, "\n") == strings.Join(want, "\n"))
	for _, sw := range aria2NextSwitches {
		n := 0
		for _, f := range flags {
			if f == "--"+sw+"=false" {
				n++
			}
		}
		check("the flags turn "+sw+" off once", n == 1)
	}
	for _, f := range flags {
		check("the flags carry no "+f+" that sets nothing",
			!strings.HasPrefix(f, "--enable-dht6") && !strings.HasPrefix(f, "--bt-require-crypto"))
	}
	f0, err := a2nWinFlags("d", "0", "6800", tok)
	check("peer port 0 leaves the port to the build", err == nil && !strings.Contains(strings.Join(f0, " "), "--listen-port"))
	for _, bad := range []string{"", "x", "70000", "-1", "051413", "51413 "} {
		_, err := a2nWinFlags("d", bad, "6800", tok)
		check("the peer port ["+bad+"] is refused", err != nil)
	}
	_, err = a2nWinFlags("d", "1", "0", tok)
	check("an RPC port of 0 is refused", err != nil)
	_, err = a2nWinFlags("d", "1", "6800", "short")
	check("a token that is not 32 hex digits is refused", err != nil)

	// The read-back, over a dump in the build's own compact form. ⭐ Planted
	// defects: each surface left on, a new switch of a family, a new `true`.
	clean := `{"id":"bit-ids","jsonrpc":"2.0","result":{"allow-overwrite":"true","bt-dht-bootstrap-nodes":"a:1","bt-enable-lpd":"false",` +
		`"bt-port-mapping":"false","ed2k-port-mapping":"false","enable-dht":"false","enable-peer-exchange":"false",` +
		`"enable-rpc":"true","follow-torrent":"true","listen-port":"51413","rpc-listen-all":"false"}}`
	check("a clean dump is held", aria2NextOptionsRefusal([]byte(clean), "51413") == "")
	check("a clean dump with the port left to the build is held", aria2NextOptionsRefusal([]byte(clean), "0") == "")
	for _, sw := range aria2NextSwitches {
		on := strings.Replace(clean, `"`+sw+`":"false"`, `"`+sw+`":"true"`, 1)
		check("the planted "+sw+"=true is refused", strings.Contains(aria2NextOptionsRefusal([]byte(on), "51413"), sw))
		gone := strings.Replace(clean, `"`+sw+`":"false",`, "", 1)
		check("the planted dump without "+sw+" is refused", strings.Contains(aria2NextOptionsRefusal([]byte(gone), "51413"), sw))
	}
	for _, k := range []string{"enable-dht6", "kad-port-mapping", "bt-enable-lsd", "ed2k-peer-exchange"} {
		on := strings.Replace(clean, `"enable-rpc"`, `"`+k+`":"auto","enable-rpc"`, 1)
		check("the planted family key "+k+" not false is refused", strings.Contains(aria2NextOptionsRefusal([]byte(on), "51413"), k))
	}
	newTrue := strings.Replace(clean, `"enable-rpc"`, `"enable-kad":"true","enable-rpc"`, 1)
	check("the planted unjudged true is refused", strings.Contains(aria2NextOptionsRefusal([]byte(newTrue), "51413"), "enable-kad as true"))
	check("a listen port other than the one asked for is refused",
		aria2NextOptionsRefusal([]byte(clean), "6881") == "aria2-next reports a listen port other than 6881")

	// The build's answer.
	for in, want := range map[string]string{
		"aria2-next version 3.0.3\r\nCopyright (C) 2006, 2019 Tatsuhiro Tsujikawa\r\n": "3.0.3",
		"Aria2 Next version 2.7.4\n":   "2.7.4",
		"aria2-next version 3.1.0-rc1": "3.1.0-rc1",
	} {
		v, err := a2nWinParseVersion(in)
		check("the answer ["+strings.TrimSpace(firstLineOf(in))+"] reads as "+want, err == nil && v == want)
	}
	for _, in := range []string{"", "aria2-next version", "aria2-next version version", "Usage: aria2-next [OPTIONS]",
		"transmission-daemon version 4.1.3", "aria2-next 3.0.3"} {
		_, err := a2nWinParseVersion(in)
		check("the answer ["+in+"] is refused", err != nil)
	}

	// Scoop's manifest, in `bucket/aria2-next.json`'s shape at 1359d92.
	man := `{"version":"3.0.3","architecture":{"64bit":{"url":"https://github.com/AnInsomniacy/aria2-next/releases/download/v3.0.3/aria2-next-3.0.3-windows-x86_64.exe#/aria2-next.exe",` +
		`"hash":"` + sum + `"},"arm64":{"url":"https://x/arm64.exe","hash":"00"}}}`
	e, err := a2nWinScoopManifest([]byte(man))
	check("Scoop's manifest is read", err == nil && e.version == "3.0.3" &&
		e.url == "https://github.com/AnInsomniacy/aria2-next/releases/download/v3.0.3/aria2-next-3.0.3-windows-x86_64.exe" &&
		e.hash == sum)
	e, err = a2nWinScoopManifest([]byte(strings.Replace(man, `"hash":"`+sum, `"hash":"SHA256:`+strings.ToUpper(sum), 1)))
	check("Scoop's sha256: prefix and upper case are read", err == nil && e.hash == sum)
	for name, bad := range map[string]string{
		"a version the url does not name": strings.Replace(man, `"version":"3.0.3"`, `"version":"3.0.4"`, 1),
		"a url that is not https":         strings.Replace(man, "https://github.com", "http://github.com", 1),
		"a list of urls": strings.Replace(man, `"url":"https://github.com/AnInsomniacy/aria2-next/releases/download/v3.0.3/aria2-next-3.0.3-windows-x86_64.exe#/aria2-next.exe"`,
			`"url":["https://a/aria2-next-3.0.3-windows-x86_64.exe","https://b/aria2-next-3.0.3-windows-x86_64.exe"]`, 1),
		"a short hash":           strings.Replace(man, `"hash":"`+sum, `"hash":"`+sum[:40], 1),
		"no 64bit build":         strings.Replace(man, `"64bit"`, `"32bit"`, 1),
		"no version":             strings.Replace(man, `"version":"3.0.3",`, "", 1),
		"the arm64 build's file": strings.Replace(man, "windows-x86_64.exe#", "windows-arm64.exe#", 1),
		"not JSON":               man[:40],
	} {
		_, err := a2nWinScoopManifest([]byte(bad))
		check("Scoop's manifest with "+name+" is refused", err != nil)
	}

	// The RPC bodies and answers.
	body := a2nWinRPCBody("aria2.addTorrent", tok, "QUJD")
	var parsed map[string]any
	check("an RPC body is JSON", json.Unmarshal([]byte(body), &parsed) == nil)
	check("an RPC body carries the token first", strings.Contains(body, `"params":["token:`+tok+`","QUJD"]`))
	g, err := a2nWinRPCResult([]byte(`{"id":"bit-ids","jsonrpc":"2.0","result":"2089b05ecca3d829"}`))
	check("a result is read", err == nil && g == "2089b05ecca3d829")
	_, err = a2nWinRPCResult([]byte(`{"id":"bit-ids","jsonrpc":"2.0","error":{"code":1,"message":"Unauthorized"}}`))
	check("an error member is a refusal", err != nil)
	_, err = a2nWinRPCResult([]byte(`{"id":"bit-ids","jsonrpc":"2.0","result":{"a":1}}`))
	check("a result that is not a string is refused where a string is asked for", err != nil)
	_, err = a2nWinRPCAnswer([]byte(`{"id":"bit-ids","jsonrpc":"2.0","result":{"version":"3.0.3"}}`))
	check("an object result is an answer", err == nil)
	_, err = a2nWinRPCAnswer([]byte(`{"id":"bit-ids","jsonrpc":"2.0","result":null}`))
	check("a null result is not an answer", err != nil)
	_, err = a2nWinRPCAnswer([]byte(`<html>502</html>`))
	check("an answer that is not JSON is refused", err != nil)

	// The packaging slug.
	for _, r := range []string{"release", "package"} {
		f, ok := a2nWinPackageFormat(r)
		check("the "+r+" route delivers pe-binary", ok && f == "pe-binary")
	}
	_, ok := a2nWinPackageFormat("source")
	check("the source route is not a Windows route", !ok)

	// Where Scoop placed the build, over a scratch tree.
	work, err := os.MkdirTemp("", "a2nwin-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "aria2-next-windows: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	mk := func(rel string) {
		p := filepath.Join(work, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
	}
	mk("one/3.0.3/aria2-next.exe")
	mk("one/current/aria2-next.exe")
	v, exe, err := a2nWinScoopBuild(filepath.Join(work, "one"))
	check("the version directory is found and current is not", err == nil && v == "3.0.3" &&
		exe == filepath.Join(work, "one", "3.0.3", a2nWinExe))
	mk("two/3.0.3/aria2-next.exe")
	mk("two/3.0.4/aria2-next.exe")
	_, _, err = a2nWinScoopBuild(filepath.Join(work, "two"))
	check("two version directories are refused, not chosen between", err != nil)
	mk("only-current/current/aria2-next.exe")
	_, _, err = a2nWinScoopBuild(filepath.Join(work, "only-current"))
	check("current alone is refused", err != nil)
	mk("empty/3.0.3/readme.txt")
	_, _, err = a2nWinScoopBuild(filepath.Join(work, "empty"))
	check("a version directory with no build is refused", err != nil)
	_, _, err = a2nWinScoopBuild(filepath.Join(work, "absent"))
	check("no app directory is refused", err != nil)

	fmt.Printf(`{"schema":"aria2-next-windows-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}

// firstLineOf is a case name's first line.
func firstLineOf(s string) string {
	l, _, _ := strings.Cut(strings.ReplaceAll(s, "\r", ""), "\n")
	return l
}
