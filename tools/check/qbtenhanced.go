// qbtenhanced - what both halves of the `qbittorrent-enhanced` adapter decide,
// as functions with no process and no platform in them, so any host can ask
// them through `bit-check qbittorrent-enhanced --selftest`.
//
// CLIENT-02. The adapters are adapter_qbittorrent-enhanced.go (Linux) and
// adapter_qbittorrent-enhanced_windows.go; this file registers nothing.
//
// ⛔ AN INDEPENDENT TARGET, NOT A FLAVOUR OF `qbittorrent`. Enhanced Edition is
// a fork, and its observable identity is what this project has to measure,
// never what its parent's profile says. So the scheme, the assets, the routes,
// the profile and the version reading here are this target's own declarations,
// each measured against Enhanced's own releases and source on 2026-10-21. ⚠
// What is REUSED from the qBittorrent Windows half is mechanics only: running a
// tool into a log, launching and waiting on a process, asking the Web UI on
// loopback, the throwaway password hash and its scrub, and finding the one
// `qbittorrent.exe` inside an extracted installer. None of those carries a
// value a record publishes.
//
// -- ⭐ WHAT ENHANCED PUBLISHES, read on 2026-10-21 -----------------------------
//
// `release-5.2.4.10`, 2026-10-03, the newest of 100 release objects and not a
// prerelease, carries 13 assets and no checksums document or signature. The
// listing states a sha256 for every asset. For x86-64 there are three builds:
//
//	qBittorrent-Enhanced-Edition-x86_64.AppImage       the GUI, Qt 6.12.0, libtorrent 1.2.20
//	qbittorrent-enhanced-nox_x86_64-linux-musl_static.zip  one static `qbittorrent-nox`, libtorrent 1.2.20
//	qbittorrent_enhanced_5.2.4.10_x64_setup.exe        NSIS, one static `qbittorrent.exe`, libtorrent 1.2.20
//
// Each was read without being run, by the digest the listing states, its
// strings and its archive layout. All three embed `-LT12K0-`, libtorrent
// 1.2.20's own fingerprint; the vendor's build scripts take libtorrent from the
// moving `RC_1_2` branch, so `1.2.20` is the version that branch reported.
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// The release scheme, one declaration for both halves.
//
// ⛔ FOUR COMPONENTS AND NOTHING ELSE, WITH A FLOOR OF 4. Enhanced tags its
// own releases `release-<qBittorrent version>.<build>`, `release-5.2.4.10`, and
// its repository also carries the 167 three-component tags it inherited from
// upstream, `release-0.x` to `release-3.3.11`. Measured over the 273 live refs
// on 2026-10-21: every inherited tag leads with 3 or less, so a floor of 4
// releases them as `below_scheme_floor` and selects `release-5.2.4.10`. ⚠ And a
// three-component tag that leads with 4 or more - upstream's `release-5.3.0`
// pushed into the fork - is `unorderable` and BLOCKS rather than being read as
// Enhanced's `5.3.0.0`, which would be selecting the parent's version by name.
const (
	qbeTarget        = "qbittorrent-enhanced"
	qbeRepo          = "c0re100/qBittorrent-Enhanced-Edition"
	qbeTagPrefix     = "release-"
	qbeMinComponents = 4
	qbeMaxComponents = 4
	qbeFloorMajor    = 4
	// ⚠ The vendor names the Linux builds with no version, so the pattern is a
	// literal and the selected release object scopes it. Exactly one asset on
	// `release-5.2.4.10` matches.
	qbeLinuxAsset   = "qbittorrent-enhanced-nox_x86_64-linux-musl_static.zip"
	qbeWindowsAsset = "qbittorrent_enhanced_{version}_x64_setup.exe"
	// qbeDigestUnpublished is the measured reason install-rootless is given.
	qbeDigestUnpublished = "this release publishes no checksums document and no signatures; the listing states a digest for every asset"
	// qbeNoxExe is what the Linux release asset carries and both Linux routes
	// install: the headless build, which needs no display server.
	qbeNoxExe = "qbittorrent-nox"
	// The Windows package route's index: Scoop's `extras` bucket.
	qbeScoopBucket = "extras"
	qbeScoopApp    = "qbittorrent-enhanced"
)

// qbeDescribeScheme is the scheme lines both halves' `describe` print.
func qbeDescribeScheme(asset string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "release_repo=%s\n", qbeRepo)
	fmt.Fprintf(&b, "release_tag_prefix=%s\n", qbeTagPrefix)
	fmt.Fprintf(&b, "release_min_components=%d\n", qbeMinComponents)
	fmt.Fprintf(&b, "release_max_components=%d\n", qbeMaxComponents)
	fmt.Fprintf(&b, "release_floor_major=%d\n", qbeFloorMajor)
	b.WriteString("release_series=-\n")
	fmt.Fprintf(&b, "release_asset=%s\n", asset)
	fmt.Fprintf(&b, "release_digest_unpublished=%s\n", qbeDigestUnpublished)
	return b.String()
}

// qbePackageFormat is how each route delivers the build, as `Build.package`
// spells it, per platform.
//
// ⚠ Linux's two forms differ: the vendor's musl static executable taken out
// of its zip, and a build linked here against Qt's shared libraries. Windows'
// two are one form, the NSIS payload extracted by 7-Zip with no installer code
// run, whichever route did the extracting; `aria2-next` calls its pair one
// form for the same reason.
func qbePackageFormat(windows bool, route string) (string, bool) {
	switch {
	case !windows && route == "release":
		return "elf-binary-static", true
	case !windows && route == "source":
		return "elf-binary", true
	case windows && (route == "release" || route == "package"):
		return "nsis-payload", true
	}
	return "", false
}

// qbeProfile is the whole configuration file the build reads from `--profile`,
// written rather than patched, for every session.
//
// ⛔ ENHANCED'S OWN SWITCHES ARE WRITTEN AT ITS OWN DEFAULTS, AND NAMED. Read
// from `release-5.2.4.10`'s `sessionimpl.cpp` and `preferences.cpp` on
// 2026-10-21, against upstream `release-5.2.4`; these are the session keys the
// fork adds that change what reaches the wire:
//
//   - `Session\AutoBanUnknownPeer`, default false: drops a peer whose client is
//     unknown and whose address GeoIP places in CN, and a set of "offline
//     downloader" peer IDs;
//   - `Session\AutoBanBTPlayerPeer`, default false: drops media-player peers;
//   - `Session\ShadowBan`, default false;
//   - `Session\AddTrackersFromURLEnabled` and the older
//     `Session\AutoUpdateTrackersEnabled`, default false: when on, the build
//     fetches a public tracker list and adds those trackers to every torrent,
//     which would be announces to hosts that are not the lab.
//
// ⚠ AND ONE FILTER THAT NO SETTING TURNS OFF, recorded rather than changed:
// `create_drop_bad_peers_plugin` is added unconditionally and drops a peer whose
// first eight peer-ID bytes match `-(XL|SD|XF|QD|BN|DL|TS|DT|HP)\d+-` or whose
// client name is four dot-separated numbers. The observer presents
// `bit-ids-fixture-NNNN`, which matches neither. Its country branch needs GeoIP,
// and `Connection\ResolvePeerCountries=false` keeps that database unloaded, so
// the branch cannot fire on a lab address either way.
//
// ⛔ THE SURFACES ARE THE CONTRACT'S, as every adapter writes them: DHT, peer
// exchange and local discovery off, UPnP off, encryption at the build's
// either-way default. ⚠ The peer ID and user agent are not configurable in this
// fork; nothing here sets them.
//
// ⭐ webuiPort 0 is the Linux nox build: its Web UI is OFF, so it generates and
// prints no temporary password. Anything else is the Windows GUI build, whose
// Web UI on loopback is the only way it answers its version and the graceful
// way it stops, with the throwaway hash qbtWinPasswordHash draws.
func qbeProfile(downloads, peerPort string, webuiPort int, passwordHash string) string {
	dl := strings.ReplaceAll(downloads, `\`, "/")
	var b strings.Builder
	line := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	line("[Application]")
	line(`FileLogger\Enabled=false`)
	line("")
	line("[BitTorrent]")
	line(`Session\AddTrackersEnabled=false`)
	line(`Session\AddTrackersFromURLEnabled=false`)
	line(`Session\AddTorrentStopped=false`)
	line(`Session\AnonymousModeEnabled=false`)
	line(`Session\AutoBanBTPlayerPeer=false`)
	line(`Session\AutoBanUnknownPeer=false`)
	line(`Session\AutoUpdateTrackersEnabled=false`)
	line(`Session\DHTEnabled=false`)
	line(`Session\DefaultSavePath=%s`, dl)
	line(`Session\Encryption=0`)
	line(`Session\GlobalMaxSeedingMinutes=1`)
	line(`Session\LSDEnabled=false`)
	line(`Session\PeXEnabled=false`)
	if peerPort != "" && peerPort != "0" {
		line(`Session\Port=%s`, peerPort)
	}
	line(`Session\ShadowBan=false`)
	line(`Session\TempPathEnabled=false`)
	line("")
	line("[Preferences]")
	line(`Advanced\updateCheck=false`)
	line(`Connection\ResolvePeerCountries=false`)
	line(`Connection\UPnP=false`)
	line(`Downloads\SavePath=%s`, dl)
	line(`General\CloseToTray=false`)
	line(`General\ExitConfirm=false`)
	line(`General\Locale=en`)
	line(`General\MinimizeToTray=false`)
	line(`General\SystrayEnabled=false`)
	if webuiPort == 0 {
		line(`WebUI\Enabled=false`)
	} else {
		line(`WebUI\Address=127.0.0.1`)
		line(`WebUI\Enabled=true`)
		line(`WebUI\LocalHostAuth=false`)
		line(`WebUI\Password_PBKDF2="%s"`, passwordHash)
		line(`WebUI\Port=%d`, webuiPort)
		line(`WebUI\UseUPnP=false`)
	}
	line("")
	line("[LegalNotice]")
	line("Accepted=true")
	line("")
	line("[AddNewTorrentDialog]")
	line("Enabled=false")
	return b.String()
}

// qbeEnhancedOff is every Enhanced-only switch the profile must carry as false.
var qbeEnhancedOff = []string{
	`Session\AutoBanUnknownPeer`, `Session\AutoBanBTPlayerPeer`, `Session\ShadowBan`,
	`Session\AddTrackersFromURLEnabled`, `Session\AutoUpdateTrackersEnabled`,
}

// qbeProfileRefusal answers what is wrong with a profile, or "". ⛔ The adapter
// asks it of the text it is about to write, so a profile that left a surface or
// an Enhanced switch on never reaches the build.
func qbeProfileRefusal(profile string) string {
	has := func(l string) bool { return strings.Contains("\n"+profile, "\n"+l+"\n") }
	for _, k := range append([]string{`Session\DHTEnabled`, `Session\PeXEnabled`, `Session\LSDEnabled`,
		`Connection\UPnP`, `Connection\ResolvePeerCountries`}, qbeEnhancedOff...) {
		if !has(k + "=false") {
			return k + " is not written as false"
		}
		if strings.Count(profile, "\n"+k+"=") > 1 || strings.HasPrefix(profile, k+"=") {
			return k + " is written more than once"
		}
	}
	if !has("Accepted=true") {
		return "the legal notice is not accepted in the profile"
	}
	return ""
}

var qbeNoxVersionRe = regexp.MustCompile(`^qBittorrent v([0-9]+(\.[0-9]+){2,3}[A-Za-z0-9.+-]*)$`)

// qbeParseNoxVersion reads `qbittorrent-nox --version`, which `cmdoptions.cpp`
// prints as `<application name> <QBT_VERSION>`: `qBittorrent v5.2.4.10`.
// ⚠ The fork keeps the application name `qBittorrent`, so the name is no
// evidence of which product answered; the four-component version is. A line
// that is not that shape is refused rather than trimmed into a version.
func qbeParseNoxVersion(output string) (string, error) {
	for _, l := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		if m := qbeNoxVersionRe.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("%s reported no parseable version: [%s]", qbeNoxExe, strings.TrimSpace(firstLineOf(output)))
}

// qbeUnzipNox takes the one executable out of the vendor's zip and writes it
// to dest at mode 0755, answering the number of bytes written.
//
// ⛔ EXACTLY ONE ENTRY, NAMED `qbittorrent-nox`, A REGULAR FILE. Measured on
// 2026-10-21: the x86-64 zip holds that one file, 35877040 bytes. A second
// entry, a directory, a link, or a name with a path in it is a refusal: the
// route would otherwise be choosing which of the vendor's files is the build, or
// writing where the archive says rather than where this route does.
func qbeUnzipNox(zipPath, dest string) (int64, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, fmt.Errorf("the release asset is not a zip: %v", err)
	}
	defer r.Close()
	if len(r.File) != 1 {
		var names []string
		for _, f := range r.File {
			names = append(names, f.Name)
		}
		return 0, fmt.Errorf("the zip holds %d entries, not one %s: %s", len(r.File), qbeNoxExe, strings.Join(names, ", "))
	}
	f := r.File[0]
	if f.Name != qbeNoxExe {
		return 0, fmt.Errorf("the zip's one entry is %q, not %s", f.Name, qbeNoxExe)
	}
	if !f.Mode().IsRegular() {
		return 0, fmt.Errorf("the zip's %s is not a regular file (%v)", qbeNoxExe, f.Mode())
	}
	in, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return n, err
	}
	if uint64(n) != f.UncompressedSize64 {
		return n, fmt.Errorf("the zip says %d bytes and %d were written", f.UncompressedSize64, n)
	}
	return n, os.Chmod(dest, 0o755)
}

// qbeScoopEntry is what Scoop's manifest says it will install.
type qbeScoopEntry struct {
	version, url, hash string
}

var qbeVersionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+){3}$`)

// qbeScoopManifest reads `bucket/qbittorrent-enhanced.json` for the x64 build.
//
// ⭐ MEASURED, through `AGENTS.md` rule 8's route on 2026-10-21: Scoop's
// `extras` bucket names 5.2.4.10 since commit `c4049da8` (2026-10-03T21:21Z),
// the vendor's own GitHub asset URL with Scoop's `#/dl.7z` rename, and hash
// `d76fe0df...0d27dc2`, the digest the release listing states. ⛔ winget's
// `c0re100.qBittorrent-Enhanced-Edition` stops at 5.2.3.10 (its 5.2.4.10 pull
// request #449896 was closed unmerged), and Chocolatey's `qbittorrent-enhanced`
// stops at 5.2.3.10 too, so Scoop is the one manager at the newest stable.
//
// ⚠ Refused: a URL that is not https or not one string; a URL whose file is
// not the installer for the version the manifest names; a hash that is not
// SHA-256; and a version that is not Enhanced's four components.
func qbeScoopManifest(body []byte) (qbeScoopEntry, error) {
	var m struct {
		Version      string `json:"version"`
		Architecture map[string]struct {
			URL  json.RawMessage `json:"url"`
			Hash json.RawMessage `json:"hash"`
		} `json:"architecture"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest is not JSON: %v", err)
	}
	if !qbeVersionRe.MatchString(m.Version) {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names no four-component version: [%s]", m.Version)
	}
	x64, ok := m.Architecture["64bit"]
	if !ok {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names no 64bit build")
	}
	var url, hash string
	if json.Unmarshal(x64.URL, &url) != nil {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names its 64bit url as something other than one string")
	}
	if json.Unmarshal(x64.Hash, &hash) != nil {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names its 64bit hash as something other than one string")
	}
	url, _, _ = strings.Cut(url, "#")
	if !strings.HasPrefix(url, "https://") {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names no https url: [%s]", url)
	}
	if want := strings.ReplaceAll(qbeWindowsAsset, "{version}", m.Version); path.Base(url) != want {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names version %s and the url %s, which is not %s", m.Version, url, want)
	}
	hash = strings.TrimPrefix(strings.ToLower(hash), "sha256:")
	if !lowerHex64.MatchString(hash) {
		return qbeScoopEntry{}, fmt.Errorf("Scoop's manifest names no SHA-256 hash: [%s]", hash)
	}
	return qbeScoopEntry{version: m.Version, url: url, hash: hash}, nil
}

// qbeScoopBuild finds the build Scoop placed under
// `<scoop>\apps\qbittorrent-enhanced` and answers its version directory's name
// and the executable in it. ⛔ The version directory, never `current` (a
// junction) and never the shim on PATH, which is Scoop's own executable; and
// exactly one, as `select-asset` refuses two assets.
func qbeScoopBuild(appDir string) (string, string, error) {
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
	if len(found) != 1 {
		return "", "", fmt.Errorf("%d version directories under %s: %s", len(found), appDir, strings.Join(found, ", "))
	}
	exe := filepath.Join(appDir, found[0], qbtWinExe)
	if !isRegular(exe) {
		return "", "", fmt.Errorf("no %s in %s", qbtWinExe, filepath.Join(appDir, found[0]))
	}
	return found[0], exe, nil
}

// -- the selftest -------------------------------------------------------------

// qbittorrentEnhancedSelftest is `bit-check qbittorrent-enhanced --selftest`:
// every decision above asked over a case whose answer is known, with the
// defects planted, on any host.
func qbittorrentEnhancedSelftest(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "qbittorrent-enhanced: usage: bit-check qbittorrent-enhanced --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "qbittorrent-enhanced: FAILED %s\n", name)
		}
	}

	// The scheme: four components, a floor of 4, this repository and no other.
	d := qbeDescribeScheme(qbeLinuxAsset)
	for _, want := range []string{"release_repo=c0re100/qBittorrent-Enhanced-Edition", "release_tag_prefix=release-",
		"release_min_components=4", "release_max_components=4", "release_floor_major=4", "release_series=-",
		"release_asset=qbittorrent-enhanced-nox_x86_64-linux-musl_static.zip"} {
		check("describe carries "+want, strings.Contains(d, want+"\n"))
	}
	check("describe never names the parent's repository", !strings.Contains(d, "qbittorrent/qBittorrent"))
	check("the Windows asset is pinned by {version}",
		strings.Contains(qbeDescribeScheme(qbeWindowsAsset), "release_asset=qbittorrent_enhanced_{version}_x64_setup.exe\n"))

	// The profile: every surface and every Enhanced switch off, written once.
	lin := qbeProfile("/w/downloads", "51413", 0, "")
	win := qbeProfile(`C:\w\downloads`, "51413", 40123, "@ByteArray(s:k)")
	check("the Linux profile is held", qbeProfileRefusal(lin) == "")
	check("the Windows profile is held", qbeProfileRefusal(win) == "")
	check("the Linux profile turns the Web UI off", strings.Contains(lin, "\nWebUI\\Enabled=false\n"))
	check("the Linux profile carries no password", !strings.Contains(strings.ToLower(lin), "password"))
	check("the Windows profile serves the Web UI on loopback only",
		strings.Contains(win, "\nWebUI\\Address=127.0.0.1\n") && strings.Contains(win, "\nWebUI\\Port=40123\n"))
	check("a path is written without a backslash", !strings.Contains(win, `C:\w`))
	check("the peer port is written", strings.Contains(lin, "\nSession\\Port=51413\n"))
	check("peer port 0 leaves the port to the build", !strings.Contains(qbeProfile("d", "0", 0, ""), `Session\Port=`))
	for _, k := range append([]string{`Session\DHTEnabled`, `Session\PeXEnabled`, `Session\LSDEnabled`,
		`Connection\UPnP`, `Connection\ResolvePeerCountries`}, qbeEnhancedOff...) {
		on := strings.Replace(lin, k+"=false", k+"=true", 1)
		check("the planted "+k+"=true is refused", strings.Contains(qbeProfileRefusal(on), k))
		gone := strings.Replace(lin, k+"=false\n", "", 1)
		check("the planted profile without "+k+" is refused", strings.Contains(qbeProfileRefusal(gone), k))
		twice := strings.Replace(lin, k+"=false\n", k+"=false\n"+k+"=true\n", 1)
		check("the planted second "+k+" is refused", strings.Contains(qbeProfileRefusal(twice), k))
	}
	check("a profile with no legal notice accepted is refused",
		qbeProfileRefusal(strings.Replace(lin, "Accepted=true", "Accepted=false", 1)) != "")

	// The build's answer.
	for in, want := range map[string]string{"qBittorrent v5.2.4.10\n": "5.2.4.10", "qBittorrent v5.2.4.10": "5.2.4.10",
		"\nqBittorrent v5.3.0.10beta1\n": "5.3.0.10beta1"} {
		v, err := qbeParseNoxVersion(in)
		check("the answer ["+strings.TrimSpace(in)+"] reads as "+want, err == nil && v == want)
	}
	for _, in := range []string{"", "qBittorrent", "qBittorrent v", "qBittorrent 5.2.4.10", "v5.2.4.10",
		"Bad command line: --version", "qBittorrent v5", "qBittorrent v5.2.4.10 extra"} {
		_, err := qbeParseNoxVersion(in)
		check("the answer ["+in+"] is refused", err != nil)
	}

	// The packaging slugs.
	for _, c := range []struct {
		win         bool
		route, want string
		ok          bool
	}{{false, "release", "elf-binary-static", true}, {false, "source", "elf-binary", true},
		{false, "package", "", false}, {true, "release", "nsis-payload", true}, {true, "package", "nsis-payload", true},
		{true, "source", "", false}} {
		f, ok := qbePackageFormat(c.win, c.route)
		check(fmt.Sprintf("windows=%v route %s packages as [%s]", c.win, c.route, c.want), ok == c.ok && f == c.want)
	}

	// Scoop's manifest, in `bucket/qbittorrent-enhanced.json`'s shape at c4049da8.
	sum := strings.Repeat("d76f", 16)
	u := "https://github.com/c0re100/qBittorrent-Enhanced-Edition/releases/download/release-5.2.4.10/qbittorrent_enhanced_5.2.4.10_x64_setup.exe"
	man := `{"version":"5.2.4.10","architecture":{"64bit":{"url":"` + u + `#/dl.7z","hash":"` + sum + `"}},"bin":"qbittorrent.exe"}`
	e, err := qbeScoopManifest([]byte(man))
	check("Scoop's manifest is read", err == nil && e.version == "5.2.4.10" && e.url == u && e.hash == sum)
	e, err = qbeScoopManifest([]byte(strings.Replace(man, `"hash":"`+sum, `"hash":"SHA256:`+strings.ToUpper(sum), 1)))
	check("Scoop's sha256: prefix and upper case are read", err == nil && e.hash == sum)
	for name, bad := range map[string]string{
		"a version the url does not name": strings.Replace(man, `"version":"5.2.4.10"`, `"version":"5.2.4.11"`, 1),
		"a three-component version":       strings.Replace(man, `"version":"5.2.4.10"`, `"version":"5.2.4"`, 1),
		"a url that is not https":         strings.Replace(man, "https://github.com", "http://github.com", 1),
		"a list of urls":                  strings.Replace(man, `"url":"`+u+`#/dl.7z"`, `"url":["`+u+`","`+u+`"]`, 1),
		"the parent's installer":          strings.Replace(man, "qbittorrent_enhanced_5.2.4.10_x64", "qbittorrent_5.2.4.10_x64", 1),
		"a short hash":                    strings.Replace(man, `"hash":"`+sum, `"hash":"`+sum[:40], 1),
		"no 64bit build":                  strings.Replace(man, `"64bit"`, `"32bit"`, 1),
		"not JSON":                        man[:40],
	} {
		_, err := qbeScoopManifest([]byte(bad))
		check("Scoop's manifest with "+name+" is refused", err != nil)
	}

	// The zip and Scoop's tree, over scratch files.
	work, err := os.MkdirTemp("", "qbe-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "qbittorrent-enhanced: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	mkzip := func(name string, entries ...[2]string) string {
		p := filepath.Join(work, name)
		f, _ := os.Create(p)
		w := zip.NewWriter(f)
		for _, en := range entries {
			if strings.HasSuffix(en[0], "/") {
				w.Create(en[0])
				continue
			}
			h := &zip.FileHeader{Name: en[0], Method: zip.Deflate}
			if en[1] == "@link" {
				h.SetMode(os.ModeSymlink | 0o777)
			} else {
				h.SetMode(0o755)
			}
			x, _ := w.CreateHeader(h)
			x.Write([]byte(en[1]))
		}
		w.Close()
		f.Close()
		return p
	}
	dest := filepath.Join(work, "out")
	n, err := qbeUnzipNox(mkzip("good.zip", [2]string{"qbittorrent-nox", "ELF"}), dest)
	body, _ := os.ReadFile(dest)
	fi, _ := os.Stat(dest)
	check("the one qbittorrent-nox is taken out whole", err == nil && n == 3 && string(body) == "ELF")
	check("and placed executable", fi != nil && fi.Mode().Perm() == 0o755)
	for name, z := range map[string]string{
		"two entries":        mkzip("two.zip", [2]string{"qbittorrent-nox", "a"}, [2]string{"README", "b"}),
		"another name":       mkzip("name.zip", [2]string{"qbittorrent", "a"}),
		"a path in the name": mkzip("path.zip", [2]string{"../qbittorrent-nox", "a"}),
		"a nested name":      mkzip("nest.zip", [2]string{"bin/qbittorrent-nox", "a"}),
		"a link":             mkzip("link.zip", [2]string{"qbittorrent-nox", "@link"}),
		"a directory":        mkzip("dir.zip", [2]string{"qbittorrent-nox/", ""}),
		"no entries":         mkzip("empty.zip"),
	} {
		_, err := qbeUnzipNox(z, filepath.Join(work, "bad-out"))
		check("a zip with "+name+" is refused", err != nil)
	}
	os.WriteFile(filepath.Join(work, "not.zip"), []byte("not a zip"), 0o644)
	_, err = qbeUnzipNox(filepath.Join(work, "not.zip"), filepath.Join(work, "bad-out"))
	check("a file that is not a zip is refused", err != nil)

	mk := func(rel string) {
		p := filepath.Join(work, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
	}
	mk("one/5.2.4.10/qbittorrent.exe")
	mk("one/current/qbittorrent.exe")
	v, exe, err := qbeScoopBuild(filepath.Join(work, "one"))
	check("the version directory is found and current is not", err == nil && v == "5.2.4.10" &&
		exe == filepath.Join(work, "one", "5.2.4.10", qbtWinExe))
	mk("two/5.2.3.10/qbittorrent.exe")
	mk("two/5.2.4.10/qbittorrent.exe")
	_, _, err = qbeScoopBuild(filepath.Join(work, "two"))
	check("two version directories are refused, not chosen between", err != nil)
	mk("only-current/current/qbittorrent.exe")
	_, _, err = qbeScoopBuild(filepath.Join(work, "only-current"))
	check("current alone is refused", err != nil)
	mk("empty/5.2.4.10/readme.txt")
	_, _, err = qbeScoopBuild(filepath.Join(work, "empty"))
	check("a version directory with no build is refused", err != nil)

	fmt.Printf(`{"schema":"qbittorrent-enhanced-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
