//go:build windows

// adapter_qbittorrent_windows - the CLIENT-01 capture adapter, Windows half.
//
// CI-11. The contract is scripts/capture/adapters/README.md and the Linux half
// is scripts/capture/adapters/qbittorrent.sh (and its Go port); this file is
// the same five subcommands for the vendor's Windows build, run as `bit-check
// adapter qbittorrent <subcommand>` because a Windows runner has no `sh`. What
// this decides without Windows in it is in qbtwindows.go, which any host can
// ask through `bit-check qbittorrent-windows --selftest`.
//
// -- ⛔ WHAT DIFFERS FROM LINUX, AND WHY EACH IS A FACT OF THE BUILD ---------
//
//   - `version` asks the running build over its Web UI, because the Windows GUI
//     build has no `--version`: `cmdoptions.cpp` compiles the option out on
//     Windows, so the flag is an unknown parameter and the build opens a modal
//     *Bad command line* box that nothing answers. Read from `release-5.2.4`'s
//     source on 2026-10-15. ⚠ So `version` starts the build, in a throwaway
//     profile it deletes, with every surface off - which is the one place this
//     adapter is not the pure read the contract describes, and it says so.
//   - `stop` asks the build to shut down over the same Web UI, which is the
//     graceful exit Linux gets from SIGTERM; Windows has no signal to send a
//     GUI process, and `TerminateProcess` would cut it off before it announces
//     `stopped`. A build that does not exit in time is then killed, and the
//     workdir's `stop.txt` says which of the two ended it.
//   - No display server: Windows gives a GUI process its desktop, so there is
//     no Xvfb. ⚠ NOT MEASURED on a hosted runner: whether the runner's session
//     lets the build create its window is what the first dispatch answers.
//
// -- the routes -----------------------------------------------------------------
//
// ⛔ `release`: the vendor's `qbittorrent_<version>_lt20_x64_setup.exe`,
// fetched by `bit-check install-rootless --fetch` under the resolution's digest
// disposition, then its NSIS payload EXTRACTED by the runner's 7-Zip into the
// rootless prefix. No installer code runs. Settled by the operator on
// 2026-10-15.
// ⛔ `package`: winget's `qBittorrent.qBittorrent.lt2`, which runs the vendor's
// installer into Program Files. Its resolver is winget's own index and its
// delivery is SourceForge, so it shares neither with the release route; its
// installer's digest equals the release asset's, so the two routes deliver one
// build two ways, which is the strongest pair `ACQ-03` can be handed.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter("qbittorrent", qbtWinAdapter) }

const qbtWinMe = "qbittorrent"

// qbtWinExit carries a subcommand's exit code and its one line on stderr.
type qbtWinExit struct {
	code int
	msg  string
}

func (e qbtWinExit) Error() string { return e.msg }

func qbtRefuse(f string, a ...any) error { return qbtWinExit{1, fmt.Sprintf(f, a...)} }
func qbtCannot(f string, a ...any) error { return qbtWinExit{2, fmt.Sprintf(f, a...)} }

func qbtWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = qbtWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = qbtWinInstall(args[0], args[1])
		}
	case "version":
		err = qbtWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = qbtWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = qbtWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", qbtWinMe, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", qbtWinMe, err)
	return 2
}

// -- where the build is ---------------------------------------------------------

// qbtWinReleaseDir is where the release route places the extracted payload.
// ⛔ THE PREFIX IS ASKED FOR, NEVER COMPOSED: `install-rootless --prefix` is the
// one derivation, and rootlessPrefix is it, in this process.
func qbtWinReleaseDir() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "opt", "qbittorrent"), nil
}

// qbtWinPackageExe is where the package route's installer places the build:
// the winget manifest's `DefaultInstallLocation`, `%ProgramFiles%\qBittorrent`.
func qbtWinPackageExe() string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return ""
	}
	return filepath.Join(pf, "qBittorrent", qbtWinExe)
}

// qbtWinBinary is the installed executable, or "" when there is none.
//
// ⭐ THE ROUTE'S OWN PREFIX FIRST, for the reason the Linux half gives: without
// it two routes would race on search order, which is a property of the host
// rather than of the acquisition. ⚠ No PATH lookup: neither route puts the
// build on PATH, and a `qbittorrent.exe` found there would be a build neither
// route acquired.
func qbtWinBinary() string {
	if v := os.Getenv("BIT_IDS_QBITTORRENT"); v != "" {
		return v
	}
	if dir, err := qbtWinReleaseDir(); err == nil {
		if exe, err := qbtWinFindPayload(dir); err == nil {
			return exe
		}
	}
	if exe := qbtWinPackageExe(); exe != "" && isRegular(exe) {
		return exe
	}
	return ""
}

// -- describe -------------------------------------------------------------------

func qbtWinDescribe() error {
	var b strings.Builder
	b.WriteString("target=qbittorrent\n")
	b.WriteString("kind=stock\n")
	// ⚠ OMITTED WHEN NOTHING IS INSTALLED, which is an answer: `binary=` with no
	// value could not be told from an adapter that declines to say.
	if exe := qbtWinBinary(); exe != "" {
		fmt.Fprintf(&b, "binary=%s\n", exe)
	}
	// ⛔ THE SAME RELEASE SCHEME AS LINUX, because it is the same repository's
	// releases; only the asset differs.
	b.WriteString("release_repo=qbittorrent/qBittorrent\n")
	b.WriteString("release_tag_prefix=release-\n")
	b.WriteString("release_min_components=3\n")
	b.WriteString("release_max_components=4\n")
	b.WriteString("release_floor_major=-\n")
	// ⛔ THE `_lt20` INSTALLER, settled by the operator on 2026-10-15. ⭐
	// `{version}` pins it: release-5.2.4 carries `qbittorrent_5.2.4_x64_setup.exe`
	// beside it, and a pattern with a run where the version goes would match both
	// and be refused. Measured through rule 8's route on 2026-10-15.
	b.WriteString("release_asset=qbittorrent_{version}_lt20_x64_setup.exe\n")
	// ⚠ THE VENDOR'S CHECKSUMS DOCUMENT, NOT THE DIGEST: release-5.2.4 has ten
	// assets, five of them detached `.asc` signatures, and no checksums file, and
	// the listing states a digest for every asset, so the route reaches
	// `source-listing`, as the Linux route does.
	b.WriteString("release_digest_unpublished=this release signs each asset with a detached .asc and publishes no checksums document\n")
	// ⚠ THE ROUTES THIS HALF HAS, and none of them compiles, so there is no
	// `compiles` key: the workflow bounds both lanes as downloads.
	b.WriteString("routes=package,release\n")
	fmt.Print(b.String())
	return nil
}

// qbtWinPackageFormat is how each route delivers the build, as
// `Build.package` spells it. ⚠ Two values for one installer's bytes, because
// they are two forms: one is the payload taken out of the installer with no
// installer code run, the other is what the installer itself placed.
func qbtWinPackageFormat(route string) (string, bool) {
	switch route {
	case "release":
		return "nsis-payload", true
	case "package":
		return "nsis-installer", true
	}
	return "", false
}

// -- install --------------------------------------------------------------------

// qbtWinRun runs a tool with stdin empty and both streams appended to log, and
// answers its exit code. ⚠ No bound of its own: `install-client` bounds the
// whole `install` call and `install-step`'s job object holds every process it
// starts.
func qbtWinRun(log string, prog string, args ...string) int {
	f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 125
	}
	defer f.Close()
	fmt.Fprintf(f, "## %s %s\n", prog, strings.Join(args, " "))
	null, _ := os.Open(os.DevNull)
	if null != nil {
		defer null.Close()
	}
	cmd := exec.Command(prog, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, f, f
	return exitOf(cmd.Run())
}

func qbtWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := qbtWinPackageFormat(route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	switch route {
	case "release":
		if err := qbtWinInstallRelease(workdir, log); err != nil {
			return err
		}
	case "package":
		if err := qbtWinInstallPackage(workdir, log); err != nil {
			return err
		}
	}
	// ⛔ WHAT THE ROUTE DELIVERED, WRITTEN WHERE `install-client` READS IT.
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

// qbtWin7z is the runner's 7-Zip.
func qbtWin7z() string {
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		if p := filepath.Join(pf, "7-Zip", "7z.exe"); isRegular(p) {
			return p
		}
	}
	if p, err := exec.LookPath("7z"); err == nil {
		return p
	}
	return ""
}

func qbtWinInstallRelease(workdir, log string) error {
	// ⛔ RESOLVED BEFORE THE ROUTE RUNS AND HANDED IN; resolving here would be a
	// second resolver inside one route.
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	sevenZip := qbtWin7z()
	if sevenZip == "" {
		return qbtCannot("the release route extracts with 7-Zip and neither %%ProgramFiles%%\\7-Zip\\7z.exe nor 7z is on this host")
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	// ⛔ THE FETCH AND THE DIGEST ARE ONE CALL, AND IT IS NOT THIS FILE'S.
	// `install-rootless --fetch` bounds the transfer and settles the digest
	// before anything is unpacked. ⚠ `--fetch`, not `--install`: the artifact is
	// an installer, and placing it as the executable would be the one mistake
	// this route exists not to make.
	disposition := []string{}
	if res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION"); res != "" {
		disposition = append(disposition, "--from-resolution", res)
	} else {
		// ⚠ The standalone drive, an adapter run with a URL and no resolution,
		// carries this adapter's own declaration rather than a second spelling.
		disposition = append(disposition, "--digest-unpublished",
			"this release signs each asset with a detached .asc and publishes no checksums document")
	}
	setup := filepath.Join(workdir, "qbittorrent-setup.exe")
	args := append([]string{"install-rootless", "--fetch", "--url", url, "--into", setup}, disposition...)
	args = append(args, "--report", filepath.Join(workdir, "rootless-install.txt"))
	switch rc := qbtWinRun(log, self, args...); rc {
	case 0:
	case 1:
		return qbtRefuse("the release route could not fetch the vendor's installer")
	default:
		return qbtCannot("the rootless installer could not run for the release route (exit %d)", rc)
	}

	// ⛔ EXTRACTED, NEVER RUN. 7-Zip reads an NSIS installer as an archive, so
	// the payload lands in a staging directory beside the prefix's own, and is
	// renamed into place only once exactly one executable is found in it.
	dir, err := qbtWinReleaseDir()
	if err != nil {
		return qbtCannot("the rootless prefix could not be derived")
	}
	staged := dir + ".new"
	if err := os.RemoveAll(staged); err != nil {
		return qbtRefuse("cannot clear %s", staged)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return qbtRefuse("this user cannot create %s; a rootless install writes nowhere else", filepath.Dir(dir))
	}
	if rc := qbtWinRun(log, sevenZip, "x", "-y", "-bd", "-o"+staged, setup); rc != 0 {
		return qbtRefuse("7-Zip could not extract the installer (exit %d)", rc)
	}
	if _, err := qbtWinFindPayload(staged); err != nil {
		return qbtRefuse("the extracted installer carries no single build: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return qbtRefuse("cannot clear %s", dir)
	}
	if err := os.Rename(staged, dir); err != nil {
		return qbtRefuse("cannot place the payload at %s", dir)
	}
	// ⭐ WHAT EXTRACTED IT IS RECORDED BESIDE WHAT WAS EXTRACTED: the tool's path
	// and digest, so the record says which code turned the installer into files.
	sum, _ := fileSHA256(sevenZip)
	exe, _ := qbtWinFindPayload(dir)
	note := fmt.Sprintf("extractor=%s\nextractor_sha256=%s\npayload=%s\nexecutable=%s\n", sevenZip, sum, dir, exe)
	if err := os.WriteFile(filepath.Join(workdir, "extract.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record how it extracted the installer")
	}
	return nil
}

func qbtWinInstallPackage(workdir, log string) error {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return qbtCannot("the package route installs through winget and it is not on this host")
	}
	// ⭐ WHAT THE INDEX SAYS, KEPT BEFORE IT IS ACTED ON: the version it will
	// install, the installer URL and the digest it pins. ⚠ It is the index's
	// statement and is never the version this route reports; `version` asks the
	// build.
	show := filepath.Join(workdir, "winget-show.txt")
	if rc := qbtWinRun(show, winget, "show", "--id", qbtWinWingetID, "--exact", "--source", "winget",
		"--accept-source-agreements", "--disable-interactivity"); rc != 0 {
		return qbtRefuse("winget's index could not describe %s (exit %d)", qbtWinWingetID, rc)
	}
	// ⭐ WHAT THE INDEX NAMED IS RECORDED AS THE ROUTE'S SOURCE IDENTITY.
	shown, _ := os.ReadFile(show)
	indexVersion, err := trWinWingetVersion(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	origin, err := wingetInstallerURL(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if err := writePackageIndex(workdir, "winget", qbtWinWingetID, indexVersion, origin); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	// ⛔ NO `--version`. The index is this route's resolver, as apt's is on
	// Linux: pinning the version the release route chose would make the two
	// routes share a resolver, which `E-ACQ-07` refuses. Absolute 4 is checked
	// after the install, by asking both builds.
	// ⚠ `--silent` passes the installer `/S`, and stdin is the null device, so a
	// prompt gets end of file rather than a wait.
	rc := qbtWinRun(log, winget, "install", "--id", qbtWinWingetID, "--exact", "--source", "winget",
		"--scope", "machine", "--silent", "--disable-interactivity",
		"--accept-package-agreements", "--accept-source-agreements",
		"--log", filepath.Join(workdir, "winget.log"))
	if rc != 0 {
		return qbtRefuse("the package route did not install %s (winget exit %d)", qbtWinWingetID, rc)
	}
	if exe := qbtWinPackageExe(); exe == "" || !isRegular(exe) {
		return qbtRefuse("winget reported success and left no %s", qbtWinPackageExe())
	}
	return nil
}

// -- asking and stopping the running build ---------------------------------------

// qbtWinFreePort asks the kernel for a loopback port nothing holds.
func qbtWinFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// qbtWinClient never reaches a proxy: every request it makes is to loopback.
var qbtWinClient = &http.Client{
	Timeout:   5 * time.Second,
	Transport: &http.Transport{Proxy: nil},
}

func qbtWinAPI(port int, method, path string) (int, []byte, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	if err != nil {
		return 0, nil, err
	}
	resp, err := qbtWinClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, body, nil
}

// qbtWinLaunch starts the build with stdin empty and both streams appended to
// out, and returns without waiting for it.
func qbtWinLaunch(out string, exe string, args ...string) (*exec.Cmd, error) {
	f, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	null, err := os.Open(os.DevNull)
	if err != nil {
		return nil, err
	}
	defer null.Close()
	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, f, f
	// ⚠ Its own process group, so a console event aimed at the caller's group
	// does not reach a build that has to outlive the call that started it.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// qbtWinWaiter starts the one wait a process gets and answers a function that
// says whether it has ended within a further d.
//
// ⚠ ONE WAIT PER PROCESS. On Windows a second `Wait` on a handle the first one
// released answers at once with an error, which a caller reading "it returned"
// would take for an exit; so every question goes to the same waiter.
func qbtWinWaiter(p *os.Process) func(time.Duration) bool {
	done := make(chan struct{})
	go func() { p.Wait(); close(done) }()
	return func(d time.Duration) bool {
		select {
		case <-done:
			return true
		case <-time.After(d):
			return false
		}
	}
}

// qbtWinShutdown asks the build to exit over its Web UI and, when it does not
// within the grace, kills it. It answers which of the two ended it.
func qbtWinShutdown(p *os.Process, exited func(time.Duration) bool, port int, grace time.Duration) string {
	if exited(0) {
		return "already exited"
	}
	how := "killed"
	if port > 0 {
		if code, _, err := qbtWinAPI(port, http.MethodPost, "/api/v2/app/shutdown"); err == nil && code == http.StatusOK {
			if exited(grace) {
				return "webui-shutdown"
			}
			how = "killed after a shutdown it did not finish"
		}
	}
	p.Kill()
	exited(5 * time.Second)
	return how
}

// -- version --------------------------------------------------------------------

// qbtWinVersion asks the installed build what it is, over its Web UI.
//
// ⛔ THE BUILD SPEAKING, AND THE ONE WAY THIS BUILD CAN. `/api/v2/app/version`
// is `QBT_VERSION` compiled into the executable, answered by the executable
// running; nothing here reads a filename, a version resource or the route that
// installed it. ⚠ And `/api/v2/app/buildInfo` is kept with it on stderr: Qt,
// libtorrent, Boost, OpenSSL and zlib, which is what tells a `_lt20` build from
// the default one when both answer 5.2.4.
//
// ⚠ IT STARTS THE BUILD, which the contract's other adapters do not: in a
// profile made for this call and deleted after it, with the Web UI on a free
// loopback port and DHT, peer exchange, local discovery, UPnP, the update check
// and the GeoIP fetch all off, so on a host that still has its route it asks
// nothing of the network. It adds no torrent. ⛔ It is still bounded by its
// caller, and it ends what it started on every path out.
func qbtWinVersion() error {
	exe := qbtWinBinary()
	if exe == "" {
		return qbtCannot("%s is not installed", qbtWinExe)
	}
	tmp, err := os.MkdirTemp("", "qbt-version-")
	if err != nil {
		return qbtCannot("cannot make a temporary profile")
	}
	defer os.RemoveAll(tmp)
	port, err := qbtWinFreePort()
	if err != nil {
		return qbtCannot("no loopback port is free to ask the build on")
	}
	hash, err := qbtWinRandomHash()
	if err != nil {
		return qbtCannot("cannot draw the Web UI's throwaway credential")
	}
	profile := filepath.Join(tmp, "profile")
	config := filepath.Join(profile, "qBittorrent", "config")
	if err := os.MkdirAll(config, 0o755); err != nil {
		return qbtCannot("cannot create the temporary profile")
	}
	if err := os.WriteFile(filepath.Join(config, "qBittorrent.ini"),
		[]byte(qbtWinProfile(filepath.Join(tmp, "downloads"), "0", port, hash)), 0o644); err != nil {
		return qbtCannot("the temporary profile could not be written")
	}
	out := filepath.Join(tmp, "client.out")
	cmd, err := qbtWinLaunch(out, exe, "--profile="+profile, "--no-splash")
	if err != nil {
		return qbtCannot("%s could not be started: %v", exe, err)
	}
	exited := qbtWinWaiter(cmd.Process)

	// ⚠ THE WAIT IS SHORTER THAN EVERY CALLER'S BOUND, so a build that never
	// answers is stopped HERE, with a message, rather than killed from outside
	// with none. BIT_IDS_QBT_ANSWER_TIMEOUT moves it.
	wait := time.Duration(envSeconds("BIT_IDS_QBT_ANSWER_TIMEOUT", 30)) * time.Second
	deadline := time.Now().Add(wait)
	var answer, info []byte
	for time.Now().Before(deadline) {
		if code, body, err := qbtWinAPI(port, http.MethodGet, "/api/v2/app/version"); err == nil && code == http.StatusOK {
			answer = body
			break
		}
		if exited(500 * time.Millisecond) {
			break
		}
	}
	if answer != nil {
		if code, body, err := qbtWinAPI(port, http.MethodGet, "/api/v2/app/buildInfo"); err == nil && code == http.StatusOK {
			info = body
		}
	}
	how := qbtWinShutdown(cmd.Process, exited, port, 10*time.Second)
	said, _ := os.ReadFile(out)
	if answer == nil {
		if len(bytes.TrimSpace(said)) > 0 {
			fmt.Fprintf(os.Stderr, "%s: the build printed:\n%s\n", qbtWinMe, bytes.TrimSpace(said))
		}
		return qbtCannot("%s did not answer on its Web UI within %s (%s)", exe, wait, how)
	}
	version, err := qbtWinParseVersion(string(answer))
	if err != nil {
		return qbtCannot("%v", err)
	}
	// ⭐ THE WHOLE ANSWER ON STDERR, the parsed field alone on stdout.
	fmt.Fprintf(os.Stderr, "%s reported, asked over /api/v2/app/version on 127.0.0.1:%d (%s):\n%s\n",
		qbtWinMe, port, how, strings.TrimSpace(string(answer)))
	if info != nil {
		var pretty bytes.Buffer
		if json.Indent(&pretty, info, "", "  ") == nil {
			info = pretty.Bytes()
		}
		fmt.Fprintf(os.Stderr, "/api/v2/app/buildInfo:\n%s\n", info)
	} else {
		fmt.Fprintln(os.Stderr, "/api/v2/app/buildInfo: no answer")
	}
	fmt.Println(version)
	return nil
}

// -- start and stop ---------------------------------------------------------------

func qbtWinStart(torrent, workdir, peerPort string) error {
	exe := qbtWinBinary()
	if exe == "" {
		return qbtCannot("%s is not installed", qbtWinExe)
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	profile := filepath.Join(workdir, "profile")
	config := filepath.Join(profile, "qBittorrent", "config")
	downloads := filepath.Join(workdir, "downloads")
	if err := os.MkdirAll(config, 0o755); err != nil {
		return qbtCannot("cannot create the profile under %s", workdir)
	}
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return qbtCannot("cannot create %s", downloads)
	}
	port, err := qbtWinFreePort()
	if err != nil {
		return qbtCannot("no loopback port is free for the Web UI that stops the build")
	}
	hash, err := qbtWinRandomHash()
	if err != nil {
		return qbtCannot("cannot draw the Web UI's throwaway credential")
	}
	// ⛔ THE WHOLE PROFILE IS WRITTEN, NOT PATCHED, on every session, so a
	// session's settings are never the previous session's. ⚠ The file is
	// `qBittorrent.ini`: on Windows the build names its configuration with
	// `.ini` where Linux uses `.conf`.
	if err := os.WriteFile(filepath.Join(config, "qBittorrent.ini"),
		[]byte(qbtWinProfile(downloads, peerPort, port, hash)), 0o644); err != nil {
		return qbtCannot("the profile could not be written")
	}
	if err := os.WriteFile(filepath.Join(workdir, "webui-port"), []byte(strconv.Itoa(port)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the Web UI port stop will use")
	}
	// ⛔ THE TORRENT IS A POSITIONAL ARGUMENT, as on Linux: the one control the
	// product already has that needs no second process. The Web UI is there to
	// stop the build, not to drive it.
	cmd, err := qbtWinLaunch(filepath.Join(workdir, "client.out"), exe,
		"--profile="+profile, "--relative-fastresume", "--skip-dialog=true", "--no-splash", torrent)
	if err != nil {
		return qbtCannot("%s could not be started: %v", exe, err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the build's process id")
	}
	// ⚠ RELEASED, NOT WAITED ON: the build keeps running after this call ends.
	cmd.Process.Release()
	fmt.Printf("started %s pid %d on port %s, Web UI 127.0.0.1:%d\n", qbtWinExe, pid, peerPort, port)
	return nil
}

func qbtWinStop(workdir string) error {
	// ⛔ THE CREDENTIAL LEAVES THE PROFILE ON EVERY PATH OUT OF HERE, after the
	// build has exited and rewritten its own configuration, and before anything
	// packs the workdir. `capture-client` scans the bundle for it too.
	defer func() {
		if n, err := qbtWinScrub(filepath.Join(workdir, "profile")); err != nil {
			fmt.Fprintf(os.Stderr, "%s: the profile could not be scrubbed: %v\n", qbtWinMe, err)
		} else if n > 0 {
			fmt.Fprintf(os.Stderr, "%s: removed %d credential line(s) from the profile\n", qbtWinMe, n)
		}
	}()
	pidText := readTrimmed(filepath.Join(workdir, "pid"))
	pid, err := strconv.Atoi(strings.TrimSpace(pidText))
	if err != nil || pid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	port, _ := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "webui-port"))))
	// ⚠ The grace fits inside the caller's bound: install-client and
	// capture-client bound this whole call from outside.
	how := qbtWinShutdown(p, qbtWinWaiter(p), port, 10*time.Second)
	appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: %s\n", utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", qbtWinExe, pid, how)
	return nil
}
