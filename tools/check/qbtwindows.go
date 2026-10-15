// qbtwindows - what the Windows half of the `qbittorrent` adapter decides, as
// functions with no Windows in them, so they can be asked on any host.
//
// CI-11, CLIENT-01. The adapter itself is adapter_qbittorrent_windows.go and
// builds for Windows alone; everything here is the part of it that a Linux
// gate can prove: the profile it writes, how it reads the build's answer, how
// it finds the build inside the vendor's installer, and how it takes the one
// credential the profile needs back out before the workdir ships.
// `bit-check qbittorrent-windows --selftest` asks each of them.
//
// ⚠ NOT A SECOND ADAPTER. This file registers nothing, and the Linux adapter's
// Go port is a different file; a name both halves need would live in neither.
package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// qbtWinExe is the GUI build's executable, which is the only one either 5.2.4
// Windows installer carries. ⚠ The vendor ships no `qbittorrent-nox` for
// Windows at all, so the two routes run one program, as on Linux.
const qbtWinExe = "qbittorrent.exe"

// qbtWinWingetID is the package route's identifier in winget's index.
//
// ⛔ THE `lt2` IDENTIFIER, MEASURED RATHER THAN GUESSED. Read through
// `AGENTS.md` rule 8's route on 2026-10-15: `microsoft/winget-pkgs` carries
// `manifests/q/qBittorrent/qBittorrent/lt2/5.2.4/`, whose installer manifest
// names `qbittorrent_5.2.4_lt20_x64_setup.exe` with InstallerSha256
// `28888FE5...928CFA` - the digest GitHub's listing states for the release
// route's asset. Chocolatey's `qbittorrent` 5.2.4 embeds only the default
// installer, and Scoop's extras bucket points at the default installer too, so
// winget is the one manager that installs the `_lt20` build at this version.
const qbtWinWingetID = "qBittorrent.qBittorrent.lt2"

// qbtWinProfile is the whole INI the build reads from `--profile`, written
// rather than patched, for the reason the Linux adapter gives: a capture that
// edited an existing configuration would measure the build under whatever else
// was in it.
//
// ⛔ THE SAME SWITCHES AS LINUX, AND FIVE MORE THAT ONLY WINDOWS NEEDS. Each was
// read from `release-5.2.4`'s own source on 2026-10-15:
//
//   - `General\SystrayEnabled` and `General\CloseToTray` default to TRUE in
//     `preferences.cpp`, so a closed window hides rather than quits;
//   - `General\ExitConfirm` defaults to true and is a modal dialog nobody
//     answers;
//   - `Advanced\updateCheck` defaults to true and is a Windows-only request to
//     the vendor's site at start;
//   - `Connection\ResolvePeerCountries` defaults to true and fetches a GeoIP
//     database at start;
//   - `Application\FileLogger\Enabled` defaults to true, and is turned off so
//     the build writes nothing into the profile that the workdir would ship.
//
// ⛔ AND THE WEB UI, BECAUSE IT IS THE ONLY WAY THIS BUILD ANSWERS. The Windows
// GUI build compiles `--version` OUT: `cmdoptions.cpp` declares it under
// `#if !defined(Q_OS_WIN) || defined(DISABLE_GUI)`, so on Windows the flag is an
// unknown parameter and `main.cpp` shows a modal *Bad command line* box. So the
// build is asked over `/api/v2/app/version`, on loopback, and stopped over
// `/api/v2/app/shutdown`. ⚠ `LocalHostAuth=false` lets a loopback client in
// without a session; the GUI build still refuses to start its Web UI with no
// password hash at all (`webui.cpp`: *Credentials are not set*), so one is
// written - of a random password that is discarded - and `stop` takes it back
// out of the profile before anything packs the workdir.
//
// ⚠ Paths are written with forward slashes: QSettings reads a backslash in an
// INI value as an escape, and Qt takes `C:/...` as a Windows path.
func qbtWinProfile(downloads string, peerPort string, webuiPort int, passwordHash string) string {
	// ⚠ NOT filepath.ToSlash, which rewrites only THIS host's separator and so
	// changes nothing on the Linux gate that proves this; the selftest caught it.
	dl := strings.ReplaceAll(downloads, `\`, "/")
	var b strings.Builder
	line := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	line("[Application]")
	line(`FileLogger\Enabled=false`)
	line("")
	line("[BitTorrent]")
	line(`Session\DHTEnabled=false`)
	line(`Session\PeXEnabled=false`)
	line(`Session\LSDEnabled=false`)
	line(`Session\AnonymousModeEnabled=false`)
	line(`Session\Encryption=0`)
	line(`Session\DefaultSavePath=%s`, dl)
	line(`Session\TempPathEnabled=false`)
	line(`Session\AddTorrentStopped=false`)
	line(`Session\GlobalMaxSeedingMinutes=1`)
	if peerPort != "" && peerPort != "0" {
		line(`Session\Port=%s`, peerPort)
	}
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
	line(`WebUI\Address=127.0.0.1`)
	line(`WebUI\Enabled=true`)
	line(`WebUI\LocalHostAuth=false`)
	line(`WebUI\Password_PBKDF2="%s"`, passwordHash)
	line(`WebUI\Port=%d`, webuiPort)
	line(`WebUI\UseUPnP=false`)
	line("")
	line("[LegalNotice]")
	line("Accepted=true")
	line("")
	line("[AddNewTorrentDialog]")
	line("Enabled=false")
	return b.String()
}

// qbtWinPasswordHash is the value `WebUI\Password_PBKDF2` holds, in the
// build's own form: `@ByteArray(<salt>:<key>)`, base64, PBKDF2-SHA512 over
// 100000 rounds as `base/utils/password.cpp` derives it.
//
// ⛔ OF A PASSWORD NOBODY KEEPS. Thirty-two random bytes are drawn, hashed and
// dropped; loopback needs no session because `LocalHostAuth=false`, so nothing
// ever has to present it. ⚠ The hash is still credential-shaped, which is why
// `qbtWinScrub` removes it.
func qbtWinPasswordHash(r io.Reader) (string, error) {
	secret := make([]byte, 32)
	salt := make([]byte, 16)
	if _, err := io.ReadFull(r, secret); err != nil {
		return "", err
	}
	if _, err := io.ReadFull(r, salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha512.New, string(secret), salt, 100000, 64)
	if err != nil {
		return "", err
	}
	return "@ByteArray(" + base64.StdEncoding.EncodeToString(salt) + ":" +
		base64.StdEncoding.EncodeToString(key) + ")", nil
}

// qbtWinRandomHash is qbtWinPasswordHash over the system's random source.
func qbtWinRandomHash() (string, error) { return qbtWinPasswordHash(rand.Reader) }

// qbtWinScrub removes every line naming a password from each INI the profile
// holds, and answers how many it removed.
//
// ⛔ ON EVERY CONFIGURATION FILE, NOT THE ONE THIS WROTE. The build rewrites its
// own INI at exit from the settings it holds in memory, so the line this
// adapter wrote comes back in a file the build chose; a scrub of only the
// written path would miss the copy that ships. ⚠ Matched by key, case
// insensitively, rather than by the one spelling written, for the same reason.
func qbtWinScrub(profile string) (int, error) {
	removed := 0
	err := filepath.WalkDir(profile, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ini" && ext != ".conf" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var kept []string
		n := 0
		for _, l := range strings.SplitAfter(string(body), "\n") {
			key, _, ok := strings.Cut(l, "=")
			if ok && strings.Contains(strings.ToLower(key), "password") {
				n++
				continue
			}
			kept = append(kept, l)
		}
		if n == 0 {
			return nil
		}
		removed += n
		return os.WriteFile(path, []byte(strings.Join(kept, "")), 0o644)
	})
	return removed, err
}

// qbtWinVersionRe is what `/api/v2/app/version` answers: QBT_VERSION, which
// carries a leading `v`, and nothing else.
var qbtWinVersionRe = regexp.MustCompile(`^v?([0-9]+(\.[0-9]+){1,3}([A-Za-z0-9.+-]*))$`)

// qbtWinParseVersion reads the build's answer, stripping the `v` for the
// reason the Linux adapter strips it: a label carrying a sigil is a second
// spelling of one value. An answer that is not a version is refused rather than
// trimmed into one.
func qbtWinParseVersion(body string) (string, error) {
	s := strings.TrimSpace(body)
	m := qbtWinVersionRe.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("the build answered [%s], which is not a version", s)
	}
	return m[1], nil
}

// qbtWinFindPayload finds the one executable an extracted installer carries.
//
// ⛔ EXACTLY ONE, AND NSIS'S OWN DIRECTORIES ARE NOT THE PAYLOAD. 7-Zip unpacks
// an NSIS installer's plugin directory as `$PLUGINSDIR` beside the files the
// installer would have placed; a directory whose name starts `$` is the
// installer's and is skipped. Two `qbittorrent.exe` is a refusal rather than a
// first match, for the reason `select-asset` refuses two assets: choosing by
// the order a walk found them is choosing by a property of the walk.
// ⚠ Four levels deep at most, so a payload the vendor nests is still found and
// a tree that is not a payload is not walked whole.
func qbtWinFindPayload(root string) (string, error) {
	var found []string
	base := strings.Count(filepath.Clean(root), string(filepath.Separator))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), "$") {
				return filepath.SkipDir
			}
			if strings.Count(filepath.Clean(path), string(filepath.Separator))-base >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), qbtWinExe) {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("no %s under %s", qbtWinExe, root)
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("%d copies of %s under %s: %s", len(found), qbtWinExe, root, strings.Join(found, ", "))
}

// -- the selftest -------------------------------------------------------------

// qbittorrentWindows is `bit-check qbittorrent-windows --selftest`: every
// function above asked over a case whose answer is known, on any host.
func qbittorrentWindows(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "qbittorrent-windows: usage: bit-check qbittorrent-windows --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "qbittorrent-windows: FAIL %s\n", name)
		}
	}

	// The profile: every surface off, the Web UI on loopback only.
	p := qbtWinProfile(`C:\work\downloads`, "51413", 40123, "@ByteArray(s:k)")
	for _, want := range []string{
		`Session\DHTEnabled=false`, `Session\PeXEnabled=false`, `Session\LSDEnabled=false`,
		`Connection\UPnP=false`, `Session\Port=51413`, `WebUI\Address=127.0.0.1`,
		`WebUI\Port=40123`, `WebUI\LocalHostAuth=false`, `Advanced\updateCheck=false`,
		`Connection\ResolvePeerCountries=false`, `General\SystrayEnabled=false`,
		`General\CloseToTray=false`, `General\ExitConfirm=false`, `FileLogger\Enabled=false`,
		"Accepted=true", `Downloads\SavePath=C:/work/downloads`,
	} {
		check("the profile carries "+want, strings.Contains(p, want+"\n"))
	}
	check("a path is written without a backslash", !strings.Contains(p, `C:\work`))
	check("peer port 0 leaves the port to the build", !strings.Contains(qbtWinProfile("d", "0", 1, "h"), `Session\Port=`))

	// The hash: the build's form, and a fresh one each time.
	h1, err1 := qbtWinRandomHash()
	h2, err2 := qbtWinRandomHash()
	hashRe := regexp.MustCompile(`^@ByteArray\([A-Za-z0-9+/]{22}==:[A-Za-z0-9+/]{86}==\)$`)
	check("the hash is @ByteArray(salt:key)", err1 == nil && hashRe.MatchString(h1))
	check("two hashes differ", err2 == nil && h1 != h2)

	// The answer.
	for in, want := range map[string]string{"v5.2.4": "5.2.4", "5.2.4\n": "5.2.4", "v5.3.0beta1": "5.3.0beta1"} {
		got, err := qbtWinParseVersion(in)
		check("the answer ["+strings.TrimSpace(in)+"] reads as "+want, err == nil && got == want)
	}
	for _, in := range []string{"", "Forbidden", "v", "<html>5.2.4</html>", "qBittorrent v5.2.4"} {
		_, err := qbtWinParseVersion(in)
		check("the answer ["+in+"] is refused", err != nil)
	}

	// The payload, and the scrub, over a scratch tree.
	work, err := os.MkdirTemp("", "qbtwin-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "qbittorrent-windows: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	mk := func(rel, body string) {
		p := filepath.Join(work, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	mk("one/qbittorrent.exe", "x")
	mk("one/$PLUGINSDIR/qbittorrent.exe", "nsis")
	got, err := qbtWinFindPayload(filepath.Join(work, "one"))
	check("the payload is found and $PLUGINSDIR is not", err == nil && got == filepath.Join(work, "one", "qbittorrent.exe"))
	mk("two/a/qbittorrent.exe", "x")
	mk("two/b/QBITTORRENT.EXE", "x")
	_, err = qbtWinFindPayload(filepath.Join(work, "two"))
	check("two payloads are refused, not chosen between", err != nil)
	mk("none/readme.txt", "x")
	_, err = qbtWinFindPayload(filepath.Join(work, "none"))
	check("no payload is refused", err != nil)
	mk("deep/a/b/c/d/qbittorrent.exe", "x")
	_, err = qbtWinFindPayload(filepath.Join(work, "deep"))
	check("a payload deeper than four levels is not walked to", err != nil)

	mk("profile/qBittorrent/config/qBittorrent.ini",
		"[Preferences]\nWebUI\\Port=1\nWebUI\\Password_PBKDF2=\"@ByteArray(a:b)\"\nWebUI\\password_ha1=x\n")
	mk("profile/qBittorrent/config/other.conf", "a=b\n")
	n, err := qbtWinScrub(filepath.Join(work, "profile"))
	left, _ := os.ReadFile(filepath.Join(work, "profile", "qBittorrent", "config", "qBittorrent.ini"))
	check("the scrub removes both password keys", err == nil && n == 2)
	check("the scrub keeps everything else", string(left) == "[Preferences]\nWebUI\\Port=1\n")
	_, err = qbtWinScrub(filepath.Join(work, "no-such-profile"))
	check("a profile that was never written scrubs clean", err == nil)

	fmt.Printf(`{"schema":"qbittorrent-windows-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
