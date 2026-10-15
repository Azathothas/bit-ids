//go:build windows

// adapter_transmission_windows - the CLIENT-06 capture adapter, Windows half.
//
// CI-11. The contract is scripts/capture/adapters/README.md and the Linux half
// is adapter_transmission.go; this file is the same five subcommands for the
// vendor's Windows build, run as `bit-check adapter transmission <subcommand>`
// because a Windows runner has no `sh`. What it decides without Windows in it
// is in trwindows.go, which any host can ask through `bit-check
// transmission-windows --selftest`.
//
// ⚠ THE MECHANICS ARE qBittorrent's WINDOWS HALF'S, REUSED AND NOT COPIED:
// qbtWinRun, qbtWinLaunch, qbtWinWaiter, qbtWinFreePort and the exit type. They
// carry no decision of that product; every decision here is this one's.
//
// -- ⭐ WHAT IS THE SAME AS LINUX ------------------------------------------------
//
// The daemon plus `transmission-remote`, with the whole settings document
// written; `--foreground`, which on Windows is not a preference but the only
// way the daemon runs outside the service control manager (`daemon-win32.cc`:
// without it `spawn` calls `StartServiceCtrlDispatcher`, which fails from a
// console); `version` as `transmission-daemon --version` and nothing else, the
// second field taken; the RPC verifier taken out of `settings.json` once the
// daemon has gone; and the package route stopping and disabling the service its
// installer starts.
//
// -- ⛔ WHAT DIFFERS FROM LINUX, AND WHY EACH IS A FACT OF THE BUILD ---------
//
//   - `stop` asks the daemon to exit with `transmission-remote --exit`, which is
//     the RPC `session-close`: `daemon.cc`'s RPC callback calls `stop()` on it,
//     the same exit SIGTERM reaches on Linux, which saves the settings and closes
//     the session so the build announces `stopped`. Windows has no signal to send
//     a console process in another process group. A daemon that has not exited
//     after the grace is killed, and `stop.txt` says which of the two ended it.
//   - The RPC port is a free loopback port drawn per session and written to
//     `rpc-port`, not 9091, because the package route's own service is installed
//     to listen on 9091.
//   - The scrub is Go, not `jq`, and takes every member naming a password.
//   - `helpers=` names `transmission-remote.exe`, which `start` and `stop` run,
//     so the firewall blocks it beside the daemon.
//
// -- the routes -----------------------------------------------------------------
//
// ⛔ `release`: the vendor's `transmission-<version>-x64.msi`, fetched by
// `bit-check install-rootless --fetch` under the resolution's digest
// disposition, then laid out as an ADMINISTRATIVE IMAGE, `msiexec /a /qn
// TARGETDIR=`, into the rootless prefix. No install runs: the MSI's
// AdminExecuteSequence, read on 2026-10-15, is CostInitialize, FileCost,
// CostFinalize, InstallValidate, InstallInitialize, InstallAdminPackage,
// InstallFiles and InstallFinalize - no custom action, no service, no
// registry, no launch condition. ⚠ Chosen over 7-Zip, which the qBittorrent half
// uses for an NSIS installer: 7-Zip opens an MSI as a compound file and its
// cabinet's members are named by File-table keys in one flat directory, so the
// layout would be this project's own second reading of the Directory and File
// tables. Windows Installer is the reader the vendor's package was written for.
// ⛔ `package`: winget's `Transmission.Transmission`, which runs the same MSI
// into Program Files with its Daemon and CmdTools features named, because a
// default install carries neither.
//
// ⚠ THE PAIR SHARES ITS ORIGIN. winget's x64 installer URL is the release
// route's own GitHub asset URL, digest for digest. The resolvers differ
// (winget's index against the release listing) and so does the delivery
// (winget against `install-rootless`), which is what `E-ACQ-07` and `E-ACQ-08`
// compare; the host serving the bytes does not, which is weaker than
// qBittorrent's pair, whose winget route fetches from SourceForge.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter("transmission", trWinAdapter) }

const trWinMe = "transmission"

// trWinDigestUnpublished is the Linux half's declaration, because it is the
// same release: one listing, no checksums asset, a digest on every asset.
const trWinDigestUnpublished = "this release publishes installers and a source tarball and no checksums asset"

func trWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = trWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = trWinInstall(args[0], args[1])
		}
	case "version":
		err = trWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = trWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = trWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", trWinMe, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", trWinMe, err)
	return 2
}

// -- where the build is ---------------------------------------------------------

// trWinReleaseDir is where the release route places the administrative image.
// ⛔ THE PREFIX IS ASKED FOR, NEVER COMPOSED, as the qBittorrent half asks it.
func trWinReleaseDir() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "opt", "transmission"), nil
}

// trWinPackageDir is where the package route's installer places the build: the
// MSI's `INSTALLDIR`, `ProgramFiles64Folder\Transmission`, which is the winget
// manifest's `DefaultInstallLocation`.
func trWinPackageDir() string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return ""
	}
	return filepath.Join(pf, "Transmission")
}

// trWinExes answers the installed daemon and its remote, or "" for each that
// is not there.
//
// ⭐ THE ROUTE'S OWN PREFIX FIRST, for the reason the Linux half gives. ⚠ No
// PATH lookup: neither route puts the build on PATH. ⚠ The overrides are the
// Linux half's, for a standalone drive.
func trWinExes() (string, string) {
	daemon, remote := os.Getenv("BIT_IDS_TRANSMISSION_DAEMON"), os.Getenv("BIT_IDS_TRANSMISSION_REMOTE")
	if daemon != "" {
		if remote == "" {
			remote = filepath.Join(filepath.Dir(daemon), trWinRemoteExe)
		}
		return daemon, remote
	}
	if dir, err := trWinReleaseDir(); err == nil && isDir(dir) {
		if d, r, err := trWinFindPayload(dir); err == nil {
			return d, r
		}
	}
	if dir := trWinPackageDir(); dir != "" {
		d := filepath.Join(dir, trWinDaemonExe)
		r := filepath.Join(dir, trWinRemoteExe)
		if isRegular(d) {
			if !isRegular(r) {
				r = ""
			}
			return d, r
		}
	}
	return "", ""
}

// -- describe -------------------------------------------------------------------

func trWinDescribe() error {
	var b strings.Builder
	b.WriteString("target=transmission\n")
	b.WriteString("kind=stock\n")
	daemon, remote := trWinExes()
	if daemon != "" {
		fmt.Fprintf(&b, "binary=%s\n", daemon)
	}
	// ⛔ THE LINUX HALF'S RELEASE SCHEME, because it is the same repository's
	// releases; only the asset differs.
	b.WriteString("release_repo=transmission/transmission\n")
	b.WriteString("release_tag_prefix=-\n")
	b.WriteString("release_min_components=3\n")
	b.WriteString("release_max_components=3\n")
	b.WriteString("release_floor_major=4\n")
	// ⛔ THE x64 MSI WITHOUT `qt5`. Measured through rule 8's route on
	// 2026-10-15: 4.1.3 carries four MSIs, `-x64`, `-x86`, `-qt5-x64` and
	// `-qt5-x86`, and this literal matches the first alone. ⭐ It is the one
	// winget's x64 installer names, sha256 `c8ea492d...21741f8`, so the two
	// routes deliver one artifact.
	b.WriteString("release_asset=transmission-{version}-x64.msi\n")
	b.WriteString("release_digest_unpublished=" + trWinDigestUnpublished + "\n")
	// ⚠ Neither route compiles, so there is no `compiles` key.
	b.WriteString("routes=package,release\n")
	// ⛔ WHAT `start` AND `stop` RUN BESIDE THE BUILD, so the firewall blocks it.
	// ⚠ Only when it is there: a helper the list named and no file backed would
	// be a rule over nothing.
	if remote != "" && isRegular(remote) {
		fmt.Fprintf(&b, "helpers=%s\n", remote)
	}
	fmt.Print(b.String())
	return nil
}

// trWinPackageFormat is how each route delivers the build, as `Build.package`
// spells it. ⚠ Two values for one MSI's bytes, as the qBittorrent half has two
// for one installer's: an image laid out with no install run, and what the
// install itself placed.
func trWinPackageFormat(route string) (string, bool) {
	switch route {
	case "release":
		return "msi-payload", true
	case "package":
		return "msi-installer", true
	}
	return "", false
}

// -- install --------------------------------------------------------------------

func trWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := trWinPackageFormat(route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	switch route {
	case "release":
		if err := trWinInstallRelease(workdir, log); err != nil {
			return err
		}
	case "package":
		if err := trWinInstallPackage(workdir, log); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

// trWinSystemTool is a tool from System32, by absolute path, so the route runs
// the host's own and not whatever PATH finds first.
func trWinSystemTool(name string) string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		return ""
	}
	p := filepath.Join(root, "System32", name)
	if !isRegular(p) {
		return ""
	}
	return p
}

func trWinInstallRelease(workdir, log string) error {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	msiexec := trWinSystemTool("msiexec.exe")
	if msiexec == "" {
		return qbtCannot("the release route lays the MSI out with msiexec and %%SystemRoot%%\\System32\\msiexec.exe is not on this host")
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	// ⛔ THE FETCH AND THE DIGEST ARE ONE CALL, AND IT IS NOT THIS FILE'S.
	// ⚠ `--fetch`, not `--install`: the artifact is a package, not the build.
	disposition := []string{}
	if res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION"); res != "" {
		disposition = append(disposition, "--from-resolution", res)
	} else {
		disposition = append(disposition, "--digest-unpublished", trWinDigestUnpublished)
	}
	msi := filepath.Join(workdir, "transmission-release.msi")
	args := append([]string{"install-rootless", "--fetch", "--url", url, "--into", msi}, disposition...)
	args = append(args, "--report", filepath.Join(workdir, "rootless-install.txt"))
	switch rc := qbtWinRun(log, self, args...); rc {
	case 0:
	case 1:
		return qbtRefuse("the release route could not fetch the vendor's MSI")
	default:
		return qbtCannot("the rootless installer could not run for the release route (exit %d)", rc)
	}

	// ⛔ LAID OUT, NEVER INSTALLED, into a staging directory renamed into place
	// only once the image holds exactly one daemon with its remote.
	dir, err := trWinReleaseDir()
	if err != nil {
		return qbtCannot("the rootless prefix could not be derived")
	}
	staged := dir + ".new"
	if err := os.RemoveAll(staged); err != nil {
		return qbtRefuse("cannot clear %s", staged)
	}
	if err := os.MkdirAll(staged, 0o755); err != nil {
		return qbtRefuse("this user cannot create %s; a rootless install writes nowhere else", staged)
	}
	msiLog := filepath.Join(workdir, "msiexec.log")
	cmdLine, err := trWinMsiexecCmdLine(msiexec, msi, staged, msiLog)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if rc := trWinRunCmdLine(log, msiexec, cmdLine); rc != 0 {
		return qbtRefuse("msiexec could not lay the MSI out as an administrative image (exit %d); msiexec.log has why", rc)
	}
	if _, _, err := trWinFindPayload(staged); err != nil {
		return qbtRefuse("the administrative image carries no single build: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return qbtRefuse("cannot clear %s", dir)
	}
	if err := os.Rename(staged, dir); err != nil {
		return qbtRefuse("cannot place the image at %s", dir)
	}
	daemon, remote, _ := trWinFindPayload(dir)
	// ⭐ WHAT LAID IT OUT IS RECORDED BESIDE WHAT WAS LAID OUT, and so is the
	// runtime it will load. ⚠ The MSI carries no Visual C++ runtime: the daemon
	// imports MSVCP140, VCRUNTIME140 and VCRUNTIME140_1, and an administrative
	// image skips the launch condition that requires them, so the release route's
	// build runs on the host's copies, named here by digest.
	sum, _ := fileSHA256(msiexec)
	var b strings.Builder
	fmt.Fprintf(&b, "extractor=%s\nextractor_sha256=%s\nextractor_mode=administrative-image\n", msiexec, sum)
	fmt.Fprintf(&b, "payload=%s\nexecutable=%s\nremote=%s\n", dir, daemon, remote)
	for _, dll := range []string{"msvcp140.dll", "vcruntime140.dll", "vcruntime140_1.dll"} {
		p := trWinSystemTool(dll)
		d := ""
		if p != "" {
			d, _ = fileSHA256(p)
		}
		fmt.Fprintf(&b, "runtime=%s %s\n", dll, d)
	}
	if err := os.WriteFile(filepath.Join(workdir, "extract.txt"), []byte(b.String()), 0o644); err != nil {
		return qbtRefuse("the route could not record how it laid the MSI out")
	}
	return nil
}

// trWinRunCmdLine is qbtWinRun for a program whose command line is written
// whole, which msiexec's has to be.
func trWinRunCmdLine(log, prog, cmdLine string) int {
	f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 125
	}
	defer f.Close()
	fmt.Fprintf(f, "## %s\n", cmdLine)
	null, _ := os.Open(os.DevNull)
	if null != nil {
		defer null.Close()
	}
	cmd := exec.Command(prog)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdLine}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, f, f
	return exitOf(cmd.Run())
}

func trWinInstallPackage(workdir, log string) error {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return qbtCannot("the package route installs through winget and it is not on this host")
	}
	// ⭐ WHAT THE INDEX SAYS, KEPT BEFORE IT IS ACTED ON.
	show := filepath.Join(workdir, "winget-show.txt")
	if rc := qbtWinRun(show, winget, "show", "--id", trWinWingetID, "--exact", "--source", "winget",
		"--accept-source-agreements", "--disable-interactivity"); rc != 0 {
		return qbtRefuse("winget's index could not describe %s (exit %d)", trWinWingetID, rc)
	}
	shown, _ := os.ReadFile(show)
	indexVersion, err := trWinWingetVersion(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	// ⛔ NO `--version`, for the reason the qBittorrent half gives: the index is
	// this route's resolver. ⛔ AND THE FEATURES NAMED, because a default install
	// places the Qt client alone; `--custom` appends to winget's own silent
	// switches for a `wix` installer rather than replacing them.
	rc := qbtWinRun(log, winget, "install", "--id", trWinWingetID, "--exact", "--source", "winget",
		"--scope", "machine", "--silent", "--disable-interactivity",
		"--accept-package-agreements", "--accept-source-agreements",
		"--custom", "ADDLOCAL="+trWinMSIFeatures,
		"--log", filepath.Join(workdir, "winget.log"))
	if rc != 0 {
		return qbtRefuse("the package route did not install %s (winget exit %d)", trWinWingetID, rc)
	}
	dir := trWinPackageDir()
	daemon, remote := filepath.Join(dir, trWinDaemonExe), filepath.Join(dir, trWinRemoteExe)
	if dir == "" || !isRegular(daemon) || !isRegular(remote) {
		return qbtRefuse("winget reported success and left no %s with %s beside it in %s", trWinDaemonExe, trWinRemoteExe, dir)
	}
	// ⛔ THE INSTALLER'S OWN DAEMON IS STOPPED AND DISABLED, as on Linux: a
	// capture must own the only running copy, and this one runs with the
	// vendor's defaults and holds the default ports.
	if err := trWinStopService(workdir); err != nil {
		return err
	}
	// ⛔ AND THE BUILD IS ASKED WHETHER IT IS WHAT THE INDEX NAMED. An index that
	// installed another version is a defect of this route, refused here rather
	// than handed to `ACQ-03` as a disagreement between routes. ⚠ The record's
	// version is still `version`'s, asked again by `install-client`.
	answer, err := trWinAsk(daemon)
	if err != nil {
		return qbtRefuse("the build winget installed could not be asked its version: %v", err)
	}
	built, err := trWinParseVersion(answer)
	if err != nil {
		return qbtRefuse("%v", err)
	}
	note := fmt.Sprintf("index=%s\nindex_version=%s\nreported_version=%s\nfeatures=%s\n", trWinWingetID, indexVersion, built, trWinMSIFeatures)
	if err := os.WriteFile(filepath.Join(workdir, "winget-version.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	origin, err := wingetInstallerURL(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if err := writePackageIndex(workdir, "winget", trWinWingetID, indexVersion, origin); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	if built != indexVersion {
		return qbtRefuse("winget's index named %s and the build it installed reports %s", indexVersion, built)
	}
	return nil
}

// trWinStopService stops and disables the service the MSI installs and starts,
// and writes what it found and did to `service.txt`.
//
// ⚠ A SERVICE THAT IS NOT THERE IS RECORDED, NOT REFUSED: the capture's need is
// that nothing else runs, and nothing does. ⛔ One still running after the bound
// is refused, because the capture would then measure whichever daemon the
// torrent reached.
func trWinStopService(workdir string) error {
	sc := trWinSystemTool("sc.exe")
	if sc == "" {
		return qbtCannot("the package route stops the installer's service with sc.exe and it is not on this host")
	}
	record := filepath.Join(workdir, "service.txt")
	log := filepath.Join(workdir, "service.log")
	query := func() string {
		out := filepath.Join(workdir, "service-query.txt")
		os.Remove(out)
		qbtWinRun(out, sc, "query", trWinServiceName)
		body, _ := os.ReadFile(out)
		return trWinServiceState(string(body))
	}
	before := query()
	if before == "" {
		appendFile(record, fmt.Sprintf("%s service %s: not installed\n", utcNow(), trWinServiceName))
		return nil
	}
	qbtWinRun(log, sc, "stop", trWinServiceName)
	qbtWinRun(log, sc, "config", trWinServiceName, "start=", "disabled")
	state := query()
	for i := 0; i < 30 && state != "STOPPED"; i++ {
		time.Sleep(time.Second)
		state = query()
	}
	appendFile(record, fmt.Sprintf("%s service %s: was %s, now %s, start=disabled requested\n", utcNow(), trWinServiceName, before, state))
	if state != "STOPPED" {
		return qbtRefuse("the installer's %s service is still %s after it was asked to stop", trWinServiceName, state)
	}
	return nil
}

// trWinAsk runs `<daemon> --version` with stdin empty and both streams merged,
// as the Linux half merges them, and answers what it printed. ⚠ No bound of
// its own: every caller bounds the subcommand from outside.
func trWinAsk(daemon string) (string, error) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		return "", err
	}
	defer null.Close()
	cmd := exec.Command(daemon, "--version")
	cmd.Stdin = null
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s --version exited %d", trWinDaemonExe, exitOf(err))
	}
	return string(out), nil
}

// -- version --------------------------------------------------------------------

// trWinVersion asks the installed daemon what it is. ⛔ THE BUILD SPEAKING, a
// pure read, as on Linux: unlike the qBittorrent half, this build has a
// `--version` on Windows, and it prints and exits before anything is started.
func trWinVersion() error {
	daemon, _ := trWinExes()
	if daemon == "" {
		return qbtCannot("%s is not installed", trWinDaemonExe)
	}
	answer, err := trWinAsk(daemon)
	if err != nil {
		return qbtCannot("%v", err)
	}
	version, err := trWinParseVersion(answer)
	if err != nil {
		return qbtCannot("%v", err)
	}
	// ⭐ THE WHOLE ANSWER ON STDERR, the parsed field alone on stdout.
	fmt.Fprint(os.Stderr, answer)
	if !strings.HasSuffix(answer, "\n") {
		fmt.Fprintln(os.Stderr)
	}
	fmt.Println(version)
	return nil
}

// -- start and stop ---------------------------------------------------------------

// trWinRemote runs transmission-remote against the session on loopback with
// its output appended to out, and answers its exit code.
func trWinRemote(out, remote string, port int, args ...string) int {
	return qbtWinRun(out, remote, append([]string{fmt.Sprintf("127.0.0.1:%d", port)}, args...)...)
}

func trWinStart(torrent, workdir, peerPort string) error {
	daemon, remote := trWinExes()
	if daemon == "" {
		return qbtCannot("%s is not installed", trWinDaemonExe)
	}
	if remote == "" || !isRegular(remote) {
		return qbtCannot("%s is not installed", trWinRemoteExe)
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	config := filepath.Join(workdir, "config")
	downloads := filepath.Join(workdir, "downloads")
	if err := os.MkdirAll(config, 0o755); err != nil {
		return qbtCannot("cannot create the config under %s", workdir)
	}
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return qbtCannot("cannot create %s", downloads)
	}
	port := 0
	if p := os.Getenv("BIT_IDS_TRANSMISSION_RPC"); p != "" {
		port, _ = strconv.Atoi(p)
	} else if p, err := qbtWinFreePort(); err == nil {
		port = p
	}
	if port <= 0 {
		return qbtCannot("no loopback port is free for the RPC that drives and stops the daemon")
	}
	settings, err := trWinSettings(downloads, peerPort, port)
	if err != nil {
		return qbtCannot("%v", err)
	}
	// ⛔ THE WHOLE SETTINGS FILE IS WRITTEN, on every session.
	if err := os.WriteFile(filepath.Join(config, "settings.json"), []byte(settings), 0o644); err != nil {
		return qbtCannot("the settings could not be written")
	}
	if err := os.WriteFile(filepath.Join(workdir, "rpc-port"), []byte(strconv.Itoa(port)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the RPC port stop will use")
	}
	out := filepath.Join(workdir, "client.out")
	// ⚠ `--log-level=error` is what the Linux half's `--log-error` is deprecated
	// for in 4.1.3's `daemon.cc`, the same level without the warning.
	cmd, err := qbtWinLaunch(out, daemon, "--foreground", "--config-dir", config, "--log-level=error")
	if err != nil {
		return qbtCannot("%s could not be started: %v", daemon, err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the daemon's process id")
	}
	exited := qbtWinWaiter(cmd.Process)

	// ⚠ THE SESSION IS ASKED FOR BEFORE A TORRENT IS HANDED OVER, for the reason
	// the Linux half gives.
	answered := false
	for i := 0; i < 20; i++ {
		if trWinRemote(filepath.Join(workdir, "remote.log"), remote, port, "--session-info") == 0 {
			answered = true
			break
		}
		if exited(time.Second) {
			return qbtRefuse("the daemon exited before it answered")
		}
	}
	if !answered {
		return qbtRefuse("the daemon never answered on 127.0.0.1:%d", port)
	}
	if trWinRemote(out, remote, port, "--add", torrent) != 0 {
		return qbtRefuse("transmission-remote would not add the torrent")
	}
	if trWinRemote(out, remote, port, "--torrent", "all", "--start") != 0 {
		return qbtRefuse("transmission-remote would not start the torrent")
	}
	// ⚠ NOT WAITED ON: the daemon keeps running after this call ends.
	fmt.Printf("started %s pid %d on port %s, RPC 127.0.0.1:%d\n", trWinDaemonExe, pid, peerPort, port)
	return nil
}

func trWinStop(workdir string) error {
	// ⛔ THE VERIFIER LEAVES THE SETTINGS ON EVERY PATH OUT OF HERE, after the
	// daemon has exited and rewritten them, and before anything packs the
	// workdir. `capture-client` scans the bundle for it too.
	defer func() {
		n, deleted, err := trWinScrubSettings(filepath.Join(workdir, "config", "settings.json"))
		switch {
		case err != nil:
			fmt.Fprintf(os.Stderr, "%s: the settings could not be scrubbed: %v\n", trWinMe, err)
		case deleted:
			fmt.Fprintf(os.Stderr, "%s: the settings could not be read as JSON and were deleted\n", trWinMe)
		case n > 0:
			fmt.Fprintf(os.Stderr, "%s: removed %d password member(s) from the settings\n", trWinMe, n)
		}
	}()
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
	// ⚠ A moment rather than none, so a daemon that has already gone is read as
	// gone and not raced against the waiter starting.
	if !exited(200 * time.Millisecond) {
		how = "killed"
		port, _ := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "rpc-port"))))
		_, remote := trWinExes()
		if port > 0 && remote != "" {
			// ⚠ The grace fits inside the caller's bound, as the qBittorrent
			// half's does; the Linux half gives SIGTERM eight seconds.
			if trWinRemote(filepath.Join(workdir, "remote.log"), remote, port, "--exit") == 0 {
				if exited(10 * time.Second) {
					how = "session-close"
				} else {
					how = "killed after a session-close it did not finish"
				}
			}
		}
		if how != "session-close" {
			p.Kill()
			exited(5 * time.Second)
		}
	}
	appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: %s\n", utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", trWinDaemonExe, pid, how)
	return nil
}
