//go:build windows

// adapter_aria2-next_windows - the CLIENT-14 capture adapter, Windows half.
//
// CI-11. The contract is scripts/capture/adapters/README.md and the Linux half
// is adapter_aria2-next.go; this file is the same five subcommands for the
// vendor's Windows build, run as `bit-check adapter aria2-next <subcommand>`
// because a Windows runner has no `sh`. What it decides without Windows in it
// is in a2nwindows.go, which any host can ask through `bit-check
// aria2-next-windows --selftest`.
//
// ⚠ THE MECHANICS ARE qBittorrent's WINDOWS HALF'S, REUSED AND NOT COPIED:
// qbtWinRun, qbtWinLaunch, qbtWinWaiter, qbtWinClient and the exit type. They
// carry no decision of that product; every decision here is this one's.
//
// -- ⭐ WHAT IS THE SAME AS LINUX ------------------------------------------------
//
// `version` is `aria2-next --version` and nothing else, anchored on the
// `version` token. `start` passes the Linux half's flags in its order, with the
// same five discovery switches off and a per-run RPC secret, waits for
// `aria2.getVersion`, reads `aria2.getGlobalOption` back through the one
// function both halves share, aria2NextOptionsRefusal, and hands the torrent
// over as base64 with `aria2.addTorrent`. `stop` asks `aria2.shutdown`, gives it
// the measured ten seconds, and deletes the token before anything packs the
// workdir. The RPC port is 6800 unless BIT_IDS_ARIA2_NEXT_RPC_PORT says
// otherwise, and the stock identity, `-qB5230-`, is not overridden.
//
// -- ⛔ WHAT DIFFERS FROM LINUX, AND WHY EACH IS A FACT OF THE HOST -------------
//
//   - RPC is Go's own HTTP client to loopback with no proxy, not `curl`, so
//     `start` and `stop` run no program beside the build and `bit-check`, and
//     `describe` names no `helpers=`.
//   - A build that has not exited after `aria2.shutdown` and the grace is
//     killed: Windows has no SIGTERM to send a console process in another
//     process group. `stop.txt` says which of the two ended it.
//   - No PATH lookup. The package route's Scoop puts a SHIM named
//     `aria2-next.exe` on PATH, which is Scoop's executable and not the build.
//
// -- the routes -----------------------------------------------------------------
//
// ⛔ `release`: the vendor's `aria2-next-<version>-windows-x86_64.exe`, a bare
// console PE, placed by `bit-check install-rootless --install` at
// `<prefix>\bin\aria2-next.exe` under the resolution's digest disposition. The
// vendor publishes a checksums document and the listing a digest, so it reaches
// `both`, as the Linux route does.
// ⛔ `package`: Scoop's `main/aria2-next`, the one package manager that carries
// this product at the vendor's version. Scoop is installed by its own installer
// script into `%USERPROFILE%\scoop`, its `main` bucket is cloned at whatever
// commit is current, and `scoop install main/aria2-next` fetches and checks the
// bytes against the manifest's hash. No version is passed: the bucket is this
// route's resolver, as winget's index is the other Windows halves'.
//
// ⚠ THE PAIR SHARES ITS ORIGIN, AS TRANSMISSION'S DOES. Scoop's x64 URL is the
// release route's own GitHub asset URL, and its hash is the digest both the
// listing and the vendor's document state. The resolvers differ (a commit of
// Scoop's bucket against the release listing) and so does the delivery and its
// check (Scoop's downloader and hash against `install-rootless`), which is what
// `E-ACQ-07` and `E-ACQ-08` compare; the host serving the bytes does not.
// ⚠ And Scoop's bucket is kept current by Scoop's own bot reading the vendor's
// releases, so its index follows the vendor rather than judging it.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() { registerAdapter("aria2-next", a2nWinAdapter) }

const a2nWinMe = "aria2-next"

func a2nWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = a2nWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = a2nWinInstall(args[0], args[1])
		}
	case "version":
		err = a2nWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = a2nWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = a2nWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", a2nWinMe, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", a2nWinMe, err)
	return 2
}

// -- where the build is ---------------------------------------------------------

// a2nWinReleaseExe is where the release route places the build. ⛔ THE PREFIX
// IS ASKED FOR, NEVER COMPOSED, as the other Windows halves ask it.
func a2nWinReleaseExe() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "bin", a2nWinExe), nil
}

// a2nWinScoopDir is the root the package route installs Scoop into: Scoop's
// own default, `%USERPROFILE%\scoop`, passed to its installer by name so the
// route does not depend on a `SCOOP` the host happens to carry.
func a2nWinScoopDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !filepath.IsAbs(home) {
		return ""
	}
	return filepath.Join(home, "scoop")
}

// a2nWinScoopAppDir is where Scoop places this app's version directories.
func a2nWinScoopAppDir(root string) string {
	return filepath.Join(root, "apps", a2nWinScoopApp)
}

// a2nWinBinary is the installed executable, or "" when there is none.
//
// ⭐ THE ROUTE'S OWN PREFIX FIRST, for the reason the Linux half gives, then the
// version directory Scoop placed. ⚠ No PATH lookup: see the header.
func a2nWinBinary() string {
	if v := os.Getenv("BIT_IDS_ARIA2_NEXT"); v != "" {
		return v
	}
	if exe, err := a2nWinReleaseExe(); err == nil && isRegular(exe) {
		return exe
	}
	if root := a2nWinScoopDir(); root != "" {
		if _, exe, err := a2nWinScoopBuild(a2nWinScoopAppDir(root)); err == nil {
			return exe
		}
	}
	return ""
}

// -- describe -------------------------------------------------------------------

func a2nWinDescribe() error {
	var b strings.Builder
	b.WriteString("target=aria2-next\n")
	b.WriteString("kind=stock\n")
	if exe := a2nWinBinary(); exe != "" {
		fmt.Fprintf(&b, "binary=%s\n", exe)
	}
	// ⛔ THE LINUX HALF'S RELEASE SCHEME, because it is the same repository's
	// releases; only the asset differs.
	b.WriteString("release_repo=AnInsomniacy/aria2-next\n")
	b.WriteString("release_tag_prefix=v\n")
	b.WriteString("release_min_components=3\n")
	b.WriteString("release_max_components=3\n")
	b.WriteString("release_floor_major=-\n")
	b.WriteString("release_series=-\n")
	// ⭐ ONE MATCH. Measured through rule 8's route on 2026-10-15: v3.0.3 and
	// v3.0.4 each carry eight assets, two of them Windows builds,
	// `-windows-x86_64.exe` and `-windows-arm64.exe`, and this literal matches
	// the first alone. ⭐ It is the file Scoop's x64 URL names, sha256
	// `b2f2f7a9...d2be9465` at 3.0.3, so the two routes deliver one artifact.
	b.WriteString("release_asset=aria2-next-{version}-windows-x86_64.exe\n")
	// ⭐ THE VENDOR'S DOCUMENT, as on Linux: one checksums file covers all seven
	// builds, and its row for the Windows x64 build matches the listing's digest.
	b.WriteString("release_digest_asset=aria2-next-{version}-checksums.sha256\n")
	// ⚠ Neither route compiles, so there is no `compiles` key; and RPC is this
	// binary's own HTTP client, so there is no `helpers` key.
	b.WriteString("routes=package,release\n")
	fmt.Print(b.String())
	return nil
}

// -- install --------------------------------------------------------------------

func a2nWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := a2nWinPackageFormat(route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	switch route {
	case "release":
		if err := a2nWinInstallRelease(workdir, log); err != nil {
			return err
		}
	case "package":
		if err := a2nWinInstallPackage(workdir, log); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

func a2nWinInstallRelease(workdir, log string) error {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	// ⛔ THE DISPOSITION IS THE RESOLUTION'S, and without one this route cannot
	// run: the adapter declares the vendor's document, not a reason it has none,
	// which is the Linux half's standalone behaviour too.
	res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION")
	if res == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_RESOLUTION; this adapter declares a vendor checksums document and no reason to install without it")
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	// ⛔ THE FETCH, THE DIGEST AND THE PLACING ARE ONE CALL, AND IT IS NOT THIS
	// FILE'S. ⚠ `--install`, not `--fetch`: the artifact IS the build.
	args := []string{"install-rootless", "--install", "--url", url,
		"--into", filepath.Join(workdir, "aria2-next-release.exe"), "--as", a2nWinExe,
		"--from-resolution", res, "--report", filepath.Join(workdir, "rootless-install.txt")}
	switch rc := qbtWinRun(log, self, args...); rc {
	case 0:
	case 1:
		return qbtRefuse("the release route could not install the vendor's build")
	default:
		return qbtCannot("the rootless installer could not run for the release route (exit %d)", rc)
	}
	exe, err := a2nWinReleaseExe()
	if err != nil || !isRegular(exe) {
		return qbtRefuse("the release route reported an install and left no %s in the prefix", a2nWinExe)
	}
	return nil
}

// a2nWinPwsh runs a PowerShell script file with its arguments, stdin empty and
// both streams appended to log, and answers its exit code. ⚠ No bound of its
// own: `install-client` bounds the whole `install` call and `install-step`'s
// job object holds every process it starts, Scoop's downloads among them.
func a2nWinPwsh(log, pwsh, script string, args ...string) int {
	return qbtWinRun(log, pwsh, append([]string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", script}, args...)...)
}

func a2nWinInstallPackage(workdir, log string) error {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		return qbtCannot("the package route runs Scoop under pwsh and it is not on this host")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return qbtCannot("the package route names Scoop's bucket by its git commit and git is not on this host")
	}
	root := a2nWinScoopDir()
	if root == "" {
		return qbtCannot("there is no absolute home directory to install Scoop under")
	}
	// ⛔ A SCOOP ALREADY HERE IS REFUSED. Its bucket would be at a state this
	// route did not read, and an app it already holds would be a build this route
	// did not acquire. Scoop's installer refuses a non-empty root too; this says
	// why in this adapter's words.
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		return qbtRefuse("a Scoop is already at %s, so its index state is not this route's", root)
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}

	// ⭐ SCOOP'S OWN INSTALLER, FETCHED THROUGH THE ONE FETCH, and its digest
	// recorded in `scoop-installer.txt`. ⚠ Identified, not verified: nobody
	// publishes a digest of it.
	installer := filepath.Join(workdir, "scoop-install.ps1")
	switch rc := qbtWinRun(log, self, "install-rootless", "--fetch", "--url", a2nWinScoopInstaller,
		"--into", installer, "--digest-unpublished", a2nWinScoopInstallerUnpublished,
		"--report", filepath.Join(workdir, "scoop-installer.txt")); rc {
	case 0:
	case 1:
		return qbtRefuse("the package route could not fetch Scoop's installer")
	default:
		return qbtCannot("the rootless installer could not run for Scoop's installer (exit %d)", rc)
	}
	// ⚠ `-RunAsAdmin` because a hosted runner's user is an administrator, which
	// Scoop's installer refuses by default; it changes nothing else. `SCOOP` is
	// set for every later call, so `scoop.ps1` reads the root this route chose.
	os.Setenv("SCOOP", root)
	if rc := a2nWinPwsh(log, pwsh, installer, "-ScoopDir", root, "-RunAsAdmin"); rc != 0 {
		return qbtRefuse("Scoop's installer did not install Scoop into %s (exit %d)", root, rc)
	}

	// ⭐ WHAT THE INDEX SAYS, KEPT BEFORE IT IS ACTED ON: the manifest the bucket
	// holds and the bucket's commit, which together name the index's state.
	bucket := filepath.Join(root, "buckets", a2nWinScoopBucket)
	manifestPath := filepath.Join(bucket, "bucket", a2nWinScoopApp+".json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return qbtRefuse("Scoop's %s bucket carries no %s.json", a2nWinScoopBucket, a2nWinScoopApp)
	}
	_ = os.WriteFile(filepath.Join(workdir, "scoop-manifest.json"), manifest, 0o644)
	entry, err := a2nWinScoopManifest(manifest)
	if err != nil {
		return qbtRefuse("%v", err)
	}
	commit := ""
	{
		c := exec.Command(git, "-C", bucket, "rev-parse", "HEAD")
		out, err := c.Output()
		commit = strings.TrimSpace(string(out))
		if err != nil || !lowerHex40(commit) {
			return qbtRefuse("Scoop's %s bucket at %s has no git commit to name its state by", a2nWinScoopBucket, bucket)
		}
	}
	note := fmt.Sprintf("index=scoop\nbucket=%s\nbucket_commit=%s\napp=%s\nindex_version=%s\nindex_url=%s\nindex_sha256=%s\n",
		a2nWinScoopBucket, commit, a2nWinScoopApp, entry.version, entry.url, entry.hash)
	if err := os.WriteFile(filepath.Join(workdir, "scoop-index.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}

	// ⛔ NO VERSION. The bucket is this route's resolver; `--no-update-scoop`
	// keeps the bucket at the commit just read, so what installs is what was
	// recorded.
	scoop := filepath.Join(root, "apps", "scoop", "current", "bin", "scoop.ps1")
	if !isRegular(scoop) {
		return qbtRefuse("Scoop's installer left no %s", scoop)
	}
	if rc := a2nWinPwsh(log, pwsh, scoop, "install", a2nWinScoopBucket+"/"+a2nWinScoopApp, "--no-update-scoop"); rc != 0 {
		return qbtRefuse("the package route did not install %s/%s (scoop exit %d)", a2nWinScoopBucket, a2nWinScoopApp, rc)
	}
	placed, exe, err := a2nWinScoopBuild(a2nWinScoopAppDir(root))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if placed != entry.version {
		return qbtRefuse("Scoop's manifest named %s and Scoop placed %s", entry.version, placed)
	}
	// ⛔ THE FILE THAT WILL RUN IS THE FILE THE INDEX PINNED. Scoop checked the
	// download; this checks what it left, after the rename, by digest.
	sum, err := fileSHA256(exe)
	if err != nil || sum != entry.hash {
		return qbtRefuse("the build Scoop placed digests as %s and its manifest pins %s", sum, entry.hash)
	}
	// ⛔ AND THE BUILD IS ASKED WHETHER IT IS WHAT THE INDEX NAMED, as the
	// Transmission half asks winget's. ⚠ The record's version is still
	// `version`'s, asked again by `install-client`.
	answer, _, err := a2nWinAsk(exe)
	if err != nil {
		return qbtRefuse("the build Scoop installed could not be asked its version: %v", err)
	}
	built, err := a2nWinParseVersion(answer)
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if err := writePackageIndex(workdir, "scoop", a2nWinScoopBucket+"/"+a2nWinScoopApp, entry.version, entry.url); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	appendFile(filepath.Join(workdir, "scoop-index.txt"), fmt.Sprintf("placed=%s\nplaced_sha256=%s\nreported_version=%s\n", exe, sum, built))
	if built != entry.version {
		return qbtRefuse("Scoop's manifest named %s and the build it installed reports %s", entry.version, built)
	}
	return nil
}

// lowerHex40 says whether s is a full git object name.
func lowerHex40(s string) bool {
	return len(s) == 40 && strings.Trim(s, "0123456789abcdef") == ""
}

// a2nWinAsk runs `<exe> --version` with stdin empty and answers its stdout and
// its stderr apart, as the Linux half reads stdout alone. ⚠ No bound of its
// own: every caller bounds the subcommand from outside.
func a2nWinAsk(exe string) (string, string, error) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		return "", "", err
	}
	defer null.Close()
	var out, errb bytes.Buffer
	cmd := exec.Command(exe, "--version")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, &out, &errb
	if err := cmd.Run(); err != nil {
		return out.String(), errb.String(), fmt.Errorf("%s --version exited %d", a2nWinExe, exitOf(err))
	}
	return out.String(), errb.String(), nil
}

// -- version --------------------------------------------------------------------

// a2nWinVersion asks the installed build what it is. ⛔ THE BUILD SPEAKING, a
// pure read, as on Linux: it prints and exits before anything is started.
func a2nWinVersion() error {
	exe := a2nWinBinary()
	if exe == "" {
		return qbtCannot("aria2-next is not installed")
	}
	answer, said, err := a2nWinAsk(exe)
	if err != nil {
		return qbtCannot("%v", err)
	}
	version, err := a2nWinParseVersion(answer)
	if err != nil {
		return qbtCannot("%v", err)
	}
	// ⭐ THE WHOLE ANSWER ON STDERR, the parsed field alone on stdout. ⭐ It names
	// the build's features and libraries, which no version number carries.
	fmt.Fprintf(os.Stderr, "%s reported:\n%s\n", a2nWinMe, strings.TrimRight(strings.ReplaceAll(answer, "\r", ""), "\n"))
	if s := strings.TrimSpace(said); s != "" {
		fmt.Fprintf(os.Stderr, "%s also printed on stderr:\n%s\n", a2nWinMe, s)
	}
	fmt.Println(version)
	return nil
}

// -- start and stop ---------------------------------------------------------------

// a2nWinRPCPort is the JSON-RPC port, the Linux half's default and override.
func a2nWinRPCPort() string {
	if p := os.Getenv("BIT_IDS_ARIA2_NEXT_RPC_PORT"); p != "" {
		return p
	}
	return "6800"
}

// a2nWinRPC posts one JSON-RPC body to the build on loopback and writes the
// answer to out. ⛔ qbtWinClient has no proxy: this is a capture's own control
// channel, and it must not leave the host.
func a2nWinRPC(port, body, out string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:"+port+"/jsonrpc", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := qbtWinClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if out != "" {
		_ = os.WriteFile(out, answer, 0o644)
	}
	return answer, nil
}

func a2nWinStart(torrent, workdir, peerPort string) error {
	exe := a2nWinBinary()
	if exe == "" {
		return qbtCannot("aria2-next is not installed")
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	downloads := filepath.Join(workdir, "downloads")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return qbtCannot("cannot create %s", downloads)
	}
	rpcPort := a2nWinRPCPort()
	// ⛔ A PER-RUN TOKEN, and `stop` deletes it, for the reasons the Linux half
	// gives: the workdir is the evidence bundle.
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return qbtCannot("cannot generate an RPC token")
	}
	token := hex.EncodeToString(raw)
	flags, err := a2nWinFlags(downloads, peerPort, rpcPort, token)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "rpc-token"), []byte(token+"\n"), 0o600); err != nil {
		return qbtCannot("cannot record the RPC token")
	}
	if err := os.WriteFile(filepath.Join(workdir, "rpc-port"), []byte(rpcPort+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the RPC port")
	}
	cmd, err := qbtWinLaunch(filepath.Join(workdir, "client.out"), exe, flags...)
	if err != nil {
		return qbtCannot("aria2-next could not be started: %v", err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the pid of aria2-next")
	}
	exited := qbtWinWaiter(cmd.Process)
	rpc := func(method, out string, params ...string) ([]byte, error) {
		return a2nWinRPC(rpcPort, a2nWinRPCBody(method, token, params...), filepath.Join(workdir, out))
	}

	// ⚠ A BOUNDED WAIT ON THE WORK, NOT ON THE CLOCK, and a dead build is
	// reported as one rather than waited out.
	for waited := 0; ; waited++ {
		if body, err := rpc("aria2.getVersion", "rpc-version.json"); err == nil {
			if _, err := a2nWinRPCAnswer(body); err == nil {
				break
			}
		}
		if exited(time.Second) {
			return qbtRefuse("aria2-next exited before its RPC server answered; %s says why", filepath.Join(workdir, "client.out"))
		}
		if waited >= 30 {
			return qbtRefuse("aria2-next did not answer RPC on port %s within 30 seconds", rpcPort)
		}
	}

	// ⭐ THE CONTAINMENT SWITCHES ARE READ BACK FROM THE BUILD, through the one
	// function both halves ask.
	options, err := rpc("aria2.getGlobalOption", "rpc-options.json")
	if err != nil {
		return qbtRefuse("aria2-next would not report its global options")
	}
	if _, err := a2nWinRPCAnswer(options); err != nil {
		return qbtRefuse("aria2-next refused to report its global options: %v", err)
	}
	if msg := aria2NextOptionsRefusal(options, peerPort); msg != "" {
		return qbtRefuse("%s", msg)
	}

	// ⭐ THE TORRENT GOES IN OVER RPC, base64-ENCODED: the build is handed the
	// object the observer wrote, not a path it re-reads.
	body, err := os.ReadFile(torrent)
	if err != nil || len(body) == 0 {
		return qbtCannot("the torrent at %s could not be encoded", torrent)
	}
	added, err := rpc("aria2.addTorrent", "rpc-add.json", base64.StdEncoding.EncodeToString(body))
	if err != nil {
		return qbtRefuse("aria2-next would not accept the torrent over RPC")
	}
	gid, err := a2nWinRPCResult(added)
	if err != nil {
		return qbtRefuse("aria2-next refused the torrent: %v", err)
	}
	_ = os.WriteFile(filepath.Join(workdir, "gid"), []byte(gid+"\n"), 0o644)
	// ⚠ NOT WAITED ON: the build keeps running after this call ends.
	fmt.Printf("started aria2-next pid %d on port %s, download %s, RPC %s\n", pid, peerPort, gid, rpcPort)
	return nil
}

func a2nWinStop(workdir string) error {
	// ⛔ THE TOKEN LEAVES THE WORKDIR ON EVERY PATH OUT OF HERE, read first and
	// deleted when this returns, before `capture-client` packs the directory.
	tokenFile := filepath.Join(workdir, "rpc-token")
	token := strings.TrimSpace(readTrimmed(tokenFile))
	defer os.Remove(tokenFile)

	pid, err := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "pid"))))
	// ⛔ A pid at or below 1 is never acted on, as signalPID refuses it on Linux.
	if err != nil || pid <= 1 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: already exited\n", utcNow(), pid))
		return nil
	}
	exited := qbtWinWaiter(p)
	how := "already exited"
	if !exited(200 * time.Millisecond) {
		how = "killed"
		port := strings.TrimSpace(readTrimmed(filepath.Join(workdir, "rpc-port")))
		if port == "" {
			port = a2nWinRPCPort()
		}
		if token != "" {
			// ⭐ THE PRODUCT'S OWN SHUTDOWN FIRST, and the Linux half's measured ten
			// seconds: `aria2.shutdown` answered `OK` and the build was still alive
			// five seconds later on 2026-09-09.
			answer, err := a2nWinRPC(port, a2nWinRPCBody("aria2.shutdown", token), filepath.Join(workdir, "rpc-shutdown.json"))
			if err == nil {
				if ok, err := a2nWinRPCResult(answer); err == nil && ok == "OK" {
					if exited(10 * time.Second) {
						how = "rpc-shutdown"
					} else {
						how = "killed after a shutdown it did not finish"
					}
				}
			}
		}
		if how != "rpc-shutdown" {
			p.Kill()
			exited(5 * time.Second)
		}
	}
	appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: %s\n", utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", a2nWinExe, pid, how)
	return nil
}
