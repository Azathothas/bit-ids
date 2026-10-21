//go:build windows

// adapter_deluge_windows - the CLIENT-07 capture adapter, Windows half.
//
// The contract is scripts/capture/adapters/README.md and the Linux half is
// adapter_deluge.go; this is the same five subcommands for Windows, run as
// `bit-check adapter deluge <subcommand>`. What it decides without Windows in
// it is in deluge.go, which `bit-check deluge --selftest` asks on any host.
//
// ⚠ THE MECHANICS ARE qBittorrent's AND Transmission's WINDOWS HALVES',
// REUSED AND NOT COPIED: qbtWinRun, qbtWinLaunch, qbtWinWaiter, the exit type,
// trWinSystemTool and trWinWingetVersion. They carry no decision of those
// products.
//
// -- ⭐ WHAT IS THE SAME AS LINUX ------------------------------------------------
//
// The daemon with `core.conf` written whole per session, its RPC on a free
// loopback port, the torrent added by `deluge-console` over that RPC with the
// credential read from the shared config directory, `version` as the build's
// own `--version`, and the scrub of the credential and the TLS key on every
// path out of `stop`.
//
// -- ⛔ WHAT DIFFERS, AND WHY EACH IS A FACT OF THE BUILD ------------------------
//
//   - The executable is `deluged-debug.exe`, not `deluged.exe`. The vendor's
//     `deluged.exe` is a windowed PyInstaller build with no console, so
//     `--version` would print nowhere; `deluged-debug.exe` is the same frozen
//     program with a console, 7792803 bytes beside 7788185, read from the
//     installer on 2026-10-16. pip makes both on the source route too, and the
//     adapter runs the same name there.
//   - No `-d`: 2.2.0 adds it off Windows only, and the daemon never forks here.
//   - `stop` sends `halt` through `deluge-console`, the RPC `daemon.shutdown`,
//     because Windows has no SIGTERM for a console process in another process
//     group; the daemon gets ten seconds, as the other Windows halves give,
//     and is then ended with `taskkill /T`, because the source route's
//     launcher runs its interpreter as a child. `stop.txt` says which.
//   - `helpers=` names the console, `taskkill.exe` and, on the source route,
//     the venv's interpreter and the one it starts, which is the process that
//     opens the sockets.
//
// -- the routes ---------------------------------------------------------------
//
// ⛔ `package`: winget's `DelugeTeam.Deluge`, the vendor's
// `deluge-2.2.0-win64-setup.exe` (libtorrent 1.2.19, Python 3.9), installed
// with `--location` so the build lands where this looks. Scoop's extras bucket
// names the same installer, Chocolatey stops at 2.1.1.
// ⛔ `source`: the `deluge-2.2.0` tag, cloned behind Git for Windows' own
// `timeout` and pip-installed into a venv on the runner's Python with the Linux
// half's pins, libtorrent 2.0.15 among them. ⚠ So the two Windows routes
// install one Deluge version on two engines; `CLIENT-07` records it.
// ⚠ No `release` route: the vendor's GitHub repository has no release objects.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() { registerAdapter("deluge", dlWinAdapter) }

const (
	dlWinMe      = "deluge"
	dlWinDaemon  = "deluged-debug.exe"
	dlWinConsole = "deluge-console.exe"
)

func dlWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = dlWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = dlWinInstall(args[0], args[1])
		}
	case "version":
		err = dlWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = dlWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = dlWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", dlWinMe, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", dlWinMe, err)
	return 2
}

// -- where the build is ---------------------------------------------------------

// dlWinVenv is the source route's venv. ⛔ THE PREFIX IS ASKED FOR.
func dlWinVenv() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "opt", "deluge"), nil
}

// dlWinPackageDir is where the package route has winget's installer place the
// build, named to it with `--location`.
func dlWinPackageDir() string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return ""
	}
	return filepath.Join(pf, "Deluge")
}

// dlWinExes answers the daemon, the console and the route that placed them:
// an override, then the source route's venv, then the package route's
// directory. ⚠ No PATH lookup: neither route puts the build on PATH.
func dlWinExes() (daemon, console, route string) {
	if d := os.Getenv("BIT_IDS_DELUGE_DAEMON"); d != "" {
		c := os.Getenv("BIT_IDS_DELUGE_CONSOLE")
		if c == "" {
			c = filepath.Join(filepath.Dir(d), dlWinConsole)
		}
		return d, c, "override"
	}
	if venv, err := dlWinVenv(); err == nil {
		d := filepath.Join(venv, "Scripts", dlWinDaemon)
		if isRegular(d) {
			return d, filepath.Join(venv, "Scripts", dlWinConsole), "source"
		}
	}
	if dir := dlWinPackageDir(); dir != "" {
		d := filepath.Join(dir, dlWinDaemon)
		if isRegular(d) {
			return d, filepath.Join(dir, dlWinConsole), "package"
		}
	}
	return "", "", ""
}

// dlWinHelpers is every executable other than the build that `start` and
// `stop` run, or that the build starts, for the firewall.
func dlWinHelpers(console, route string) []string {
	var out []string
	if console != "" && isRegular(console) {
		out = append(out, console)
	}
	if tk := trWinSystemTool("taskkill.exe"); tk != "" {
		out = append(out, tk)
	}
	if route == "source" {
		if venv, err := dlWinVenv(); err == nil {
			if py := filepath.Join(venv, "Scripts", "python.exe"); isRegular(py) {
				out = append(out, py)
			}
			cfg, _ := os.ReadFile(filepath.Join(venv, "pyvenv.cfg"))
			if home := delugeVenvHome(string(cfg)); home != "" {
				if py := filepath.Join(home, "python.exe"); isRegular(py) {
					out = append(out, py)
				}
			}
		}
	}
	return out
}

// -- describe -------------------------------------------------------------------

func dlWinDescribe() error {
	var b strings.Builder
	b.WriteString("target=deluge\n")
	b.WriteString("kind=stock\n")
	daemon, console, route := dlWinExes()
	if daemon != "" {
		fmt.Fprintf(&b, "binary=%s\n", daemon)
	}
	// ⛔ THE LINUX HALF'S SCHEME, because it is the same repository's tags.
	b.WriteString("release_repo=deluge-torrent/deluge\n")
	b.WriteString("release_tag_prefix=deluge-\n")
	b.WriteString("release_min_components=3\n")
	b.WriteString("release_max_components=3\n")
	b.WriteString("release_floor_major=2\n")
	b.WriteString("routes=package,source\n")
	if h := dlWinHelpers(console, route); len(h) > 0 {
		fmt.Fprintf(&b, "helpers=%s\n", strings.Join(h, "|"))
	}
	fmt.Print(b.String())
	return nil
}

// dlWinPackageFormat is how each route delivers the build: what the vendor's
// NSIS installer placed, or a wheel this host built from the tag.
func dlWinPackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "nsis-installer", true
	case "source":
		return "python-source", true
	}
	return "", false
}

// -- install --------------------------------------------------------------------

func dlWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := dlWinPackageFormat(route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	var err error
	switch route {
	case "package":
		err = dlWinInstallPackage(workdir, log)
	case "source":
		err = dlWinInstallSource(workdir, log)
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

func dlWinInstallPackage(workdir, log string) error {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return qbtCannot("the package route installs through winget and it is not on this host")
	}
	show := filepath.Join(workdir, "winget-show.txt")
	if rc := qbtWinRun(show, winget, "show", "--id", delugeWingetID, "--exact", "--source", "winget",
		"--accept-source-agreements", "--disable-interactivity"); rc != 0 {
		return qbtRefuse("winget's index could not describe %s (exit %d)", delugeWingetID, rc)
	}
	shown, _ := os.ReadFile(show)
	indexVersion, err := trWinWingetVersion(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	origin, err := delugeWingetOrigin(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if err := writePackageIndex(workdir, "winget", delugeWingetID, indexVersion, origin); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	dir := dlWinPackageDir()
	if dir == "" {
		return qbtCannot("%%ProgramFiles%% is not set, so there is no install location to name")
	}
	// ⛔ NO `--version`: the index is this route's resolver. ⚠ `--location`
	// is the NSIS installer's `/D=`, so the build lands where `dlWinExes` looks.
	rc := qbtWinRun(log, winget, "install", "--id", delugeWingetID, "--exact", "--source", "winget",
		"--scope", "machine", "--silent", "--disable-interactivity", "--location", dir,
		"--accept-package-agreements", "--accept-source-agreements",
		"--log", filepath.Join(workdir, "winget.log"))
	if rc != 0 {
		return qbtRefuse("the package route did not install %s (winget exit %d)", delugeWingetID, rc)
	}
	daemon, console := filepath.Join(dir, dlWinDaemon), filepath.Join(dir, dlWinConsole)
	if !isRegular(daemon) || !isRegular(console) {
		return qbtRefuse("winget reported success and left no %s with %s beside it in %s", dlWinDaemon, dlWinConsole, dir)
	}
	answer, err := dlWinAsk(daemon)
	if err != nil {
		return qbtRefuse("the build winget installed could not be asked its version: %v", err)
	}
	built, engine, err := delugeParseVersion(answer)
	if err != nil {
		return qbtRefuse("%v", err)
	}
	note := fmt.Sprintf("index=%s\nindex_version=%s\nreported_version=%s\nlibtorrent=%s\n", delugeWingetID, indexVersion, built, engine)
	if err := os.WriteFile(filepath.Join(workdir, "winget-version.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	if err := delugeRecordTree(dir, "Deluge", filepath.Join(workdir, "installed-files.txt")); err != nil {
		return qbtRefuse("the installed tree could not be digested: %v", err)
	}
	if built != indexVersion {
		return qbtRefuse("winget's index named %s and the build it installed reports %s", indexVersion, built)
	}
	return nil
}

func dlWinInstallSource(workdir, log string) error {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		return qbtCannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	want := delugeTagVersion(tag)
	if want == "" {
		return qbtCannot("the source tag [%s] names no deluge version", tag)
	}
	gitExe, err := exec.LookPath("git")
	if err != nil {
		return qbtCannot("the source route clones and git is not on this host")
	}
	gnuTimeout := delugeGNUTimeout(gitExe)
	if !isRegular(gnuTimeout) {
		return qbtCannot("the clone is bounded by Git for Windows' own timeout and %s is not there", gnuTimeout)
	}
	python := os.Getenv("BIT_IDS_DELUGE_PYTHON")
	if python == "" {
		if python, err = exec.LookPath("python"); err != nil {
			return qbtCannot("the source route makes a venv and python is not on this host")
		}
	}
	src := filepath.Join(workdir, "src")
	// ⛔ BOUNDED BY COREUTILS' `timeout`, which Git for Windows carries: the
	// call names it, as every adapter's clone does, and runs Git's copy rather
	// than System32's pause. ⚠ `c.Err` is cleared because the name was looked
	// up on PATH before the path was set.
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/deluge-torrent/deluge.git", src)
	c.Path, c.Err = gnuTimeout, nil
	if rc := dlWinRunCmd(log, c); rc != 0 {
		if rc == 124 {
			return qbtRefuse("the source route's clone of %s did not finish within 600s", tag)
		}
		return qbtRefuse("the source route could not clone %s (exit %d)", tag, rc)
	}
	commit := filepath.Join(workdir, "source-commit.out")
	if rc := qbtWinRun(commit, gitExe, "-C", src, "rev-parse", "HEAD"); rc != 0 {
		return qbtRefuse("the source route cloned %s and could not name its commit", tag)
	}
	if err := os.WriteFile(filepath.Join(workdir, "source-commit"), []byte(dlWinLastLine(commit)+"\n"), 0o644); err != nil {
		return qbtRefuse("the source route could not record its commit")
	}

	venv, err := dlWinVenv()
	if err != nil {
		return qbtCannot("the rootless prefix could not be derived")
	}
	if err := os.RemoveAll(venv); err != nil {
		return qbtRefuse("cannot clear %s", venv)
	}
	if err := os.MkdirAll(filepath.Dir(venv), 0o755); err != nil {
		return qbtRefuse("this user cannot create %s; a rootless install writes nowhere else", filepath.Dir(venv))
	}
	fail := func(f string, a ...any) error {
		os.RemoveAll(venv)
		return qbtRefuse(f, a...)
	}
	if rc := qbtWinRun(log, python, "-m", "venv", venv); rc != 0 {
		return fail("python could not make a venv at %s (exit %d)", venv, rc)
	}
	py := filepath.Join(venv, "Scripts", "python.exe")
	pip := append([]string{"-m", "pip", "install", "--no-cache-dir", "--disable-pip-version-check", "--no-input",
		"--timeout", "60", "--report", filepath.Join(workdir, "pip-report.json"), src}, delugePins...)
	if rc := qbtWinRun(log, py, pip...); rc != 0 {
		return fail("pip could not install %s (exit %d)", tag, rc)
	}
	if rc := qbtWinRun(filepath.Join(workdir, "pip-freeze.txt"), py, "-m", "pip", "freeze", "--all"); rc != 0 {
		return fail("pip could not say what it installed")
	}
	pyVersion := filepath.Join(workdir, "python-version.out")
	qbtWinRun(pyVersion, python, "--version")
	sum, _ := fileSHA256(python)
	deps := fmt.Sprintf("python=%s\npython_version=%s\npython_sha256=%s\npins=%s\n", python, dlWinLastLine(pyVersion), sum, strings.Join(delugePins, " "))
	if err := os.WriteFile(filepath.Join(workdir, "build-deps.txt"), []byte(deps), 0o644); err != nil {
		return fail("the route could not record what it built on")
	}
	if err := delugeRecordTree(filepath.Join(venv, "Lib", "site-packages", "deluge"), "deluge", filepath.Join(workdir, "installed-files.txt")); err != nil {
		return fail("the installed distribution could not be digested: %v", err)
	}
	daemon := filepath.Join(venv, "Scripts", dlWinDaemon)
	if !isRegular(daemon) || !isRegular(filepath.Join(venv, "Scripts", dlWinConsole)) {
		return fail("pip installed and left no %s with %s beside it in %s", dlWinDaemon, dlWinConsole, venv)
	}
	answer, err := dlWinAsk(daemon)
	if err != nil {
		return fail("the build could not be asked its version: %v", err)
	}
	built, _, err := delugeParseVersion(answer)
	if err != nil {
		return fail("%v", err)
	}
	if built != want {
		return fail("the tag %s named %s and the build it installed reports %s", tag, want, built)
	}
	// ⛔ AND THE DAEMON'S MODULES LOAD, for the reason the Linux half gives.
	if rc := qbtWinRun(log, py, "-I", "-c", delugeImportCheck); rc != 0 {
		return fail("the build answers --version and its daemon's modules do not load; install.log has why")
	}
	return nil
}

// dlWinRunCmd runs a prepared command with stdin empty and both streams
// appended to log, as qbtWinRun runs one it builds itself.
func dlWinRunCmd(log string, c *exec.Cmd) int {
	f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 125
	}
	defer f.Close()
	fmt.Fprintf(f, "## %s %s\n", c.Path, strings.Join(c.Args[1:], " "))
	null, _ := os.Open(os.DevNull)
	if null != nil {
		defer null.Close()
	}
	c.Stdin, c.Stdout, c.Stderr = null, f, f
	return exitOf(c.Run())
}

// dlWinLastLine is the last non-empty line of a qbtWinRun log, which is the
// command's own answer under the `##` line that names it.
func dlWinLastLine(path string) string {
	b, _ := os.ReadFile(path)
	last := ""
	for _, l := range strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "## ") {
			last = l
		}
	}
	return last
}

// dlWinAsk runs `<daemon> --version` with stdin empty and both streams merged.
func dlWinAsk(daemon string) (string, error) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		return "", err
	}
	defer null.Close()
	cmd := exec.Command(daemon, "--version")
	cmd.Stdin = null
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s --version exited %d", dlWinDaemon, exitOf(err))
	}
	return string(out), nil
}

// -- version --------------------------------------------------------------------

func dlWinVersion() error {
	daemon, _, _ := dlWinExes()
	if daemon == "" {
		return qbtCannot("%s is not installed", dlWinDaemon)
	}
	answer, err := dlWinAsk(daemon)
	if err != nil {
		return qbtCannot("%v", err)
	}
	version, _, err := delugeParseVersion(answer)
	if err != nil {
		return qbtCannot("%v", err)
	}
	fmt.Fprint(os.Stderr, answer)
	if !strings.HasSuffix(answer, "\n") {
		fmt.Fprintln(os.Stderr)
	}
	fmt.Println(version)
	return nil
}

// -- start and stop ---------------------------------------------------------------

// dlWinConsoleRun runs one console step and answers what it printed.
func dlWinConsoleRun(workdir, console, config string, rpc int, step, path string) (string, error) {
	args, err := delugeConsoleArgs(config, rpc, step, path)
	if err != nil {
		return "", err
	}
	out := filepath.Join(workdir, "console-"+step+".out")
	os.Remove(out)
	qbtWinRun(out, console, args...)
	said, _ := os.ReadFile(out)
	appendFile(filepath.Join(workdir, "client.out"), string(said))
	return string(said), nil
}

func dlWinStart(torrent, workdir, peerPort string) error {
	daemon, console, _ := dlWinExes()
	if daemon == "" {
		return qbtCannot("%s is not installed", dlWinDaemon)
	}
	if !isRegular(console) {
		return qbtCannot("%s is not installed", dlWinConsole)
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	config := filepath.Join(workdir, "config")
	downloads := filepath.Join(workdir, "downloads")
	// ⛔ A FRESH CONFIG DIRECTORY PER SESSION, for the Linux half's reason.
	if err := os.RemoveAll(config); err != nil {
		return qbtCannot("cannot clear %s", config)
	}
	for _, d := range []string{config, downloads} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return qbtCannot("cannot create %s", d)
		}
	}
	conf, err := delugeCoreConf(downloads, peerPort)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if err := os.WriteFile(filepath.Join(config, "core.conf"), []byte(conf), 0o644); err != nil {
		return qbtCannot("the settings could not be written")
	}
	rpc := 0
	if p := os.Getenv("BIT_IDS_DELUGE_RPC"); p != "" {
		rpc, _ = strconv.Atoi(p)
	} else if p, err := delugeFreePort(); err == nil {
		rpc = p
	}
	if rpc <= 0 {
		return qbtCannot("no loopback port is free for the daemon's RPC")
	}
	if err := os.WriteFile(filepath.Join(workdir, "rpc-port"), []byte(strconv.Itoa(rpc)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the RPC port stop will use")
	}
	logfile := ""
	for n := 1; ; n++ {
		logfile = filepath.Join(workdir, fmt.Sprintf("deluged-%d.log", n))
		if !isRegular(logfile) {
			break
		}
	}
	cmd, err := qbtWinLaunch(filepath.Join(workdir, "client.out"), daemon, delugeDaemonArgs(config, rpc, logfile, false)...)
	if err != nil {
		return qbtCannot("%s could not be started: %v", daemon, err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the daemon's process id")
	}
	exited := qbtWinWaiter(cmd.Process)
	at := fmt.Sprintf("127.0.0.1:%d", rpc)
	answered := false
	for i := 0; i < 30; i++ {
		if conn, err := net.DialTimeout("tcp", at, time.Second); err == nil {
			conn.Close()
			answered = true
			break
		}
		if exited(time.Second) {
			return qbtRefuse("the daemon exited before it answered")
		}
	}
	if !answered {
		return qbtRefuse("the daemon never answered on %s", at)
	}
	said, err := dlWinConsoleRun(workdir, console, config, rpc, "add", torrent)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if err := delugeAdded(said); err != nil {
		return qbtRefuse("%v", err)
	}
	fmt.Printf("started %s pid %d on port %s, RPC %s\n", dlWinDaemon, pid, peerPort, at)
	return nil
}

func dlWinStop(workdir string) error {
	// ⛔ THE CREDENTIAL AND THE TLS KEY LEAVE THE WORKDIR ON EVERY PATH OUT.
	defer func() {
		gone, err := delugeScrub(workdir, filepath.Join(workdir, "config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: the workdir could not be scrubbed: %v\n", dlWinMe, err)
		} else if len(gone) > 0 {
			fmt.Fprintf(os.Stderr, "%s: removed %d file(s) holding the daemon's credential: %s\n", dlWinMe, len(gone), strings.Join(gone, ", "))
		}
	}()
	pid, err := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "pid"))))
	if err != nil || pid <= 1 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		appendFile(filepath.Join(workdir, "stop.txt"), delugeStopRecord(utcNow(), pid, "already exited"))
		return nil
	}
	exited := qbtWinWaiter(p)
	how := "already exited"
	if !exited(200 * time.Millisecond) {
		how = "killed"
		rpc, _ := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "rpc-port"))))
		_, console, _ := dlWinExes()
		if rpc > 0 && isRegular(console) {
			if _, err := dlWinConsoleRun(workdir, console, filepath.Join(workdir, "config"), rpc, "halt", ""); err == nil {
				if exited(10 * time.Second) {
					how = "halt"
				} else {
					how = "killed after a halt it did not finish"
				}
			}
		}
		if how != "halt" {
			// ⚠ The whole tree: the source route's launcher runs the interpreter
			// that is the daemon as its child.
			if tk := trWinSystemTool("taskkill.exe"); tk != "" {
				qbtWinRun(filepath.Join(workdir, "stop.log"), tk, "/PID", strconv.Itoa(pid), "/T", "/F")
			}
			p.Kill()
			exited(5 * time.Second)
		}
	}
	appendFile(filepath.Join(workdir, "stop.txt"), delugeStopRecord(utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", dlWinDaemon, pid, how)
	return nil
}
