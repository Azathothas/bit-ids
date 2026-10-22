//go:build !windows

// adapter_deluge - the CLIENT-07 capture adapter on Linux, `bit-check adapter
// deluge`.
//
// The contract is scripts/capture/adapters/README.md. ⚠ The Windows half
// registers the same name from adapter_deluge_windows.go; one platform builds
// one of them. What either half decides without a process in it is deluge.go,
// which `bit-check deluge --selftest` asks on any host.
//
// ⭐ DRIVEN THE WAY TRANSMISSION IS: the daemon, `deluged`, started in the
// foreground with its settings written, and the torrent handed over by the
// product's own client, `deluge-console`, over the daemon's RPC on loopback.
// The console finds the daemon's `localclient` credential in the config
// directory the two share, so nothing here ever holds it, and `stop` scrubs it
// out of the workdir before the workdir ships.
//
// -- the routes, measured on 2026-10-16 ----------------------------------------
//
// ⛔ `package`: PyPI's `deluge`, with no version, so PyPI's index is the
// route's resolver, installed as the vendor's wheel (`--only-binary :all:`) into
// a venv at `<prefix>/opt/deluge` beside libtorrent 2.0.15 from PyPI. Ubuntu
// 24.04's own `deluged` is 2.1.2~dev0, which is not the newest stable.
// ⛔ `source`: the `deluge-2.2.0` tag, resolved from the repository's own refs,
// cloned and pip-installed into the same venv path with the same libtorrent.
// ⚠ Both routes take Deluge's dependencies, Twisted and the rest, from PyPI,
// so the pair is independent in how DELUGE arrived and not in how its
// dependencies did. `pip-freeze.txt` beside each install says what each got.
//
// ⚠ THE EXECUTABLE `describe` NAMES IS pip's LAUNCHER, `<venv>/bin/deluged`, a
// few lines of Python whose bytes are fixed by the venv's path and the entry
// point, so both routes' launchers digest alike whatever Deluge they start.
// The digest of what each route installed is `installed-files.txt`, every file
// of the installed distribution with its sha256.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter("deluge", shAdapter("deluge", delugeAdapter)) }

// delugeVenv is where both routes build the venv. ⛔ THE PREFIX IS ASKED FOR,
// NEVER COMPOSED.
func delugeVenv(k *adapterKit, quiet bool) (string, bool) {
	p, ok := k.prefix(quiet)
	if !ok {
		return "", false
	}
	return filepath.Join(p, "opt", "deluge"), true
}

// delugeExes answers the daemon and its console, or "" for each that is not
// there: an override, then the venv, then PATH, as every Linux adapter looks.
func delugeExes(k *adapterKit) (string, string) {
	if d := os.Getenv("BIT_IDS_DELUGE_DAEMON"); d != "" {
		c := os.Getenv("BIT_IDS_DELUGE_CONSOLE")
		if c == "" {
			c = filepath.Join(filepath.Dir(d), "deluge-console")
		}
		return d, c
	}
	if venv, ok := delugeVenv(k, true); ok && isExec(filepath.Join(venv, "bin", "deluged")) {
		return filepath.Join(venv, "bin", "deluged"), filepath.Join(venv, "bin", "deluge-console")
	}
	d, _ := commandV("deluged")
	c, _ := commandV("deluge-console")
	return d, c
}

// delugePackageFormat is what each route delivers, as `Build.package` spells
// it: the vendor's published wheel, or a wheel this host built from the tag.
func delugePackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "python-wheel", true
	case "source":
		return "python-source", true
	}
	return "", false
}

func delugeAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Print("target=deluge\n")
		fmt.Print("kind=stock\n")
		if d, _ := delugeExes(k); d != "" {
			fmt.Printf("binary=%s\n", d)
		}
		// ⛔ WHAT THE SOURCE ROUTE RESOLVES THROUGH, AND NO RELEASE ROUTE. The
		// repository publishes tags and no release objects, so there is no
		// `release_asset` to declare and `resolve-release` says so. `deluge-` tags
		// of three components from 1.0.0; the four-component 0.5 series predates
		// the floor, and 2.0's PEP 440 betas are prereleases. `CLIENT-07`.
		fmt.Print("release_repo=deluge-torrent/deluge\n")
		fmt.Print("release_tag_prefix=deluge-\n")
		fmt.Print("release_min_components=3\n")
		fmt.Print("release_max_components=3\n")
		fmt.Print("release_floor_major=2\n")
		fmt.Print("release_series=-\n")
		fmt.Print("routes=package,source\n")

	case "install":
		route, workdir, format := k.installArgs(args, delugePackageFormat)
		switch route {
		case "package":
			delugePackageRoute(k, workdir)
		case "source":
			delugeSourceRoute(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		d, _ := delugeExes(k)
		if d == "" {
			k.cannot("deluged is not installed")
		}
		output, version, _ := delugeAsk(k, d)
		k.versionReport(output, version)

	case "start":
		delugeStart(k, args)

	case "stop":
		delugeStop(k, args)
	}
}

// delugeAsk runs `deluged --version` and answers the whole answer, the version
// and the engine. ⛔ THE BUILD SPEAKING, AND A PURE READ: argparse answers the
// flag while it parses, before the daemon makes its config directory.
func delugeAsk(k *adapterKit, daemon string) (string, string, string) {
	c := exec.Command(daemon, "--version")
	output, rc := capture(c, stdio{in: os.DevNull, err: "&1"})
	if rc != 0 {
		k.cannot(fmt.Sprintf("deluged --version exited %d: %s", rc, firstLine(output)))
	}
	version, engine, err := delugeParseVersion(output)
	if err != nil {
		k.cannot(err.Error())
	}
	return output, version, engine
}

// delugePython is the interpreter the venv is made from: the host's own
// `python3`, or BIT_IDS_DELUGE_PYTHON for a standalone drive.
func delugePython(k *adapterKit) string {
	if p := os.Getenv("BIT_IDS_DELUGE_PYTHON"); p != "" {
		return p
	}
	p, ok := commandV("python3")
	if !ok {
		k.cannot("the routes make a venv and python3 is not on this host")
	}
	return p
}

// delugeInstallVenv makes the venv afresh and pip-installs into it, recording
// what it ran on and what it installed. On a refusal the half-made venv is
// removed, so no lane measures it.
func delugeInstallVenv(k *adapterKit, workdir string, binaryOnly bool, specs ...string) {
	venv, ok := delugeVenv(k, false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	python := delugePython(k)
	log := filepath.Join(workdir, "install.log")
	if os.RemoveAll(venv) != nil || !mkdirP(filepath.Dir(venv)) {
		k.refuse("this user cannot clear " + venv + "; a rootless install writes nowhere else")
	}
	fail := func(msg string) {
		_ = os.RemoveAll(venv)
		k.refuse(msg)
	}
	c := exec.Command("timeout", "120", python, "-m", "venv", venv)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		fail("python3 could not make a venv at " + venv)
	}
	pip := []string{"600", filepath.Join(venv, "bin", "python"), "-m", "pip", "install", "--no-cache-dir",
		"--disable-pip-version-check", "--no-input", "--timeout", "60",
		"--report", filepath.Join(workdir, "pip-report.json")}
	if binaryOnly {
		pip = append(pip, "--only-binary", ":all:")
	}
	pip = append(append(pip, specs...), delugePins...)
	c = exec.Command("timeout", pip...)
	if rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}); rc != 0 {
		if rc == 124 {
			fail("pip did not finish within 600s")
		}
		fail(fmt.Sprintf("pip could not install %s (exit %d)", strings.Join(specs, " "), rc))
	}
	c = exec.Command(filepath.Join(venv, "bin", "python"), "-m", "pip", "freeze", "--all")
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "pip-freeze.txt"), err: log, errAppend: true}) != 0 {
		fail("pip could not say what it installed")
	}
	pyVersion, _ := capture(exec.Command(python, "--version"), stdio{in: os.DevNull, err: "&1"})
	sum, _ := fileSHA256(python)
	_ = writeText(filepath.Join(workdir, "build-deps.txt"),
		fmt.Sprintf("python=%s\npython_version=%s\npython_sha256=%s\npins=%s\n", python, pyVersion, sum, strings.Join(delugePins, " ")))
	if err := delugeRecordFiles(venv, filepath.Join(workdir, "installed-files.txt")); err != nil {
		fail("the installed distribution could not be digested: " + err.Error())
	}
}

// delugeRecordFiles digests the one `deluge` package the venv holds. See
// delugeRecordTree.
func delugeRecordFiles(venv, out string) error {
	pkgs, _ := filepath.Glob(filepath.Join(venv, "lib", "python*", "site-packages", "deluge"))
	if len(pkgs) != 1 {
		return fmt.Errorf("%d deluge packages in %s", len(pkgs), venv)
	}
	return delugeRecordTree(pkgs[0], "deluge", out)
}

// delugeCheckBuilt asks the build the venv now holds and refuses a version that
// is not the one the route was handed.
func delugeCheckBuilt(k *adapterKit, workdir, want, from string) {
	d, _ := delugeExes(k)
	venv, _ := delugeVenv(k, true)
	if d != filepath.Join(venv, "bin", "deluged") {
		k.refuse("the route installed and left no deluged in " + venv)
	}
	// ⛔ THE DAEMON'S OWN MODULES LOAD, ASKED BEFORE `--version`, with the
	// traceback in its own log and its tail in the refusal. `--version` answers
	// before the daemon's modules are imported, so a build that cannot start
	// passes it: measured on 2026-10-16, when setuptools 84 had dropped
	// `pkg_resources`. And Deluge swallows a failed `import libtorrent` there,
	// so on run 39623700677's Windows lane `--version` answered with no engine
	// and the import's own error was written nowhere. See delugePins.
	importLog := filepath.Join(workdir, "import-check.log")
	c := exec.Command(filepath.Join(venv, "bin", "python"), "-I", "-c", delugeImportCheck)
	if run(c, stdio{in: os.DevNull, out: importLog, err: "&1"}) != 0 {
		run(exec.Command("tail", "-25", importLog), stdio{out: "&2"})
		k.refuse("the build's daemon, console or engine does not load; import-check.log has the traceback")
	}
	// ⚠ And the answer is kept before it is read, so a refusal over it leaves
	// the build's own words in the workdir the run uploads.
	answer, rc := capture(exec.Command(d, "--version"), stdio{in: os.DevNull, err: "&1"})
	_ = writeText(filepath.Join(workdir, "install-version.log"), answer+"\n")
	if rc != 0 {
		k.refuse(fmt.Sprintf("deluged --version exited %d; install-version.log has its answer", rc))
	}
	version, _, err := delugeParseVersion(answer)
	if err != nil {
		k.refuse(err.Error() + "; install-version.log has the whole answer")
	}
	if version != want {
		k.refuse(fmt.Sprintf("%s named %s and the build it installed reports %s", from, want, version))
	}
}

// delugePackageRoute is PyPI. ⛔ NO VERSION, because the index is this route's
// resolver; the version it chose is read back out of pip's own report.
func delugePackageRoute(k *adapterKit, workdir string) {
	delugeInstallVenv(k, workdir, true, delugePyPIName)
	body, _ := os.ReadFile(filepath.Join(workdir, "pip-report.json"))
	version, url, _, err := delugePipReport(body, delugePyPIName)
	if err != nil {
		k.refuse(err.Error())
	}
	if err := writePackageIndex(workdir, "pypi", delugePyPIName, version, url); err != nil {
		k.refuse("the route could not record what the index named")
	}
	delugeCheckBuilt(k, workdir, version, "PyPI's index")
}

// delugeSourceRoute is the tag, cloned and built. ⛔ The commit is the one
// `E-ACQ-06` asks for, and the version the tag names is the one the build must
// report.
func delugeSourceRoute(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	want := delugeTagVersion(tag)
	if want == "" {
		k.cannot("the source tag [" + tag + "] names no deluge version")
	}
	k.need(func(string) string { return "the source route clones and git is not on this host" }, "git")
	repo := k.describeField("release_repo")
	if repo == "" {
		k.cannot("this adapter declares no release_repo to clone from")
	}
	// ⚠ Shallow, and the tag is all the build's version needs: 2.2.0's
	// `version.py` runs `git describe --tags`, which a clone at the tag answers.
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/"+repo+".git", filepath.Join(workdir, "src"))
	if rc := run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	k.cloneCommit(workdir, tag)
	delugeInstallVenv(k, workdir, false, filepath.Join(workdir, "src"))
	delugeCheckBuilt(k, workdir, want, "the tag "+tag)
}

// delugeNextLog is the next `deluged-<n>.log` in the workdir, so each session's
// log is its own and none is overwritten.
func delugeNextLog(workdir string) string {
	for n := 1; ; n++ {
		p := filepath.Join(workdir, fmt.Sprintf("deluged-%d.log", n))
		if !exists(p) {
			return p
		}
	}
}

func delugeStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	daemon, console := delugeExes(k)
	if daemon == "" {
		k.cannot("deluged is not installed")
	}
	if console == "" || !isExec(console) {
		k.cannot("deluge-console is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	config := filepath.Join(workdir, "config")
	downloads := filepath.Join(workdir, "downloads")
	// ⛔ A FRESH CONFIG DIRECTORY PER SESSION: the previous session's state would
	// resume the torrent before the console adds it, and its credential was
	// scrubbed by its `stop`.
	if os.RemoveAll(config) != nil || !mkdirP(config, downloads) {
		k.cannot("cannot create the config under " + workdir)
	}
	conf, err := delugeCoreConf(downloads, port)
	if err != nil {
		k.cannot(err.Error())
	}
	if writeText(filepath.Join(config, "core.conf"), conf) != nil {
		k.cannot("the settings could not be written")
	}
	rpc := 0
	if p := os.Getenv("BIT_IDS_DELUGE_RPC"); p != "" {
		rpc, _ = strconv.Atoi(p)
	} else if p, err := delugeFreePort(); err == nil {
		rpc = p
	}
	if rpc <= 0 {
		k.cannot("no loopback port is free for the daemon's RPC")
	}
	if writeText(filepath.Join(workdir, "rpc-port"), strconv.Itoa(rpc)+"\n") != nil {
		k.cannot("cannot record the RPC port stop will use")
	}
	out := filepath.Join(workdir, "client.out")
	c := exec.Command(daemon, delugeDaemonArgs(config, rpc, delugeNextLog(workdir), true)...)
	n, err := startBackground(c, stdio{out: out, outAppend: true, err: "&1"})
	if err != nil {
		k.cannot("deluged could not be started: " + err.Error())
	}
	pid := fmt.Sprint(n)
	if writeText(filepath.Join(workdir, "pid"), pid+"\n") != nil {
		k.cannot("cannot record the pid of deluged")
	}
	// ⚠ The RPC is asked to accept before the torrent is handed over, for the
	// reason Transmission's adapter gives: a console that cannot connect reads
	// as a build that declined the torrent.
	at := "127.0.0.1:" + strconv.Itoa(rpc)
	answered := false
	for waited := 0; waited < 30 && !answered; waited++ {
		if conn, err := net.DialTimeout("tcp", at, time.Second); err == nil {
			conn.Close()
			answered = true
			break
		}
		if !alive(pid) {
			k.refuse("the daemon exited before it answered")
		}
		time.Sleep(time.Second)
	}
	if !answered {
		k.refuse("the daemon never answered on " + at)
	}
	cargs, err := delugeConsoleArgs(config, rpc, "add", torrent)
	if err != nil {
		k.cannot(err.Error())
	}
	said, _ := capture(exec.Command(console, cargs...), stdio{in: os.DevNull, err: "&1"})
	_ = appendText(out, said+"\n")
	if err := delugeAdded(said); err != nil {
		k.refuse(err.Error())
	}
	fmt.Printf("started deluged pid %s on port %s, RPC %s\n", pid, port, at)
}

func delugeStop(k *adapterKit, args []string) {
	if len(args) < 1 {
		k.cannot("stop takes <workdir>")
	}
	workdir := args[0]
	// ⛔ THE CREDENTIAL LEAVES THE WORKDIR ON EVERY PATH OUT OF HERE, once the
	// daemon has gone and can write nothing more. See delugeScrub.
	defer func() {
		gone, err := delugeScrub(workdir, filepath.Join(workdir, "config"))
		if err != nil {
			k.say("the workdir could not be scrubbed: " + err.Error())
			return
		}
		if len(gone) > 0 {
			k.say(fmt.Sprintf("removed %d file(s) holding the daemon's credential: %s", len(gone), strings.Join(gone, ", ")))
		}
	}()
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	n, _ := strconv.Atoi(pid)
	how := "already exited"
	// ⚠ SIGTERM IS THE DAEMON'S GRACEFUL EXIT: Twisted catches it and 2.2.0's
	// `daemon.py` shuts its components down before the reactor stops, which
	// closes the session so the build announces `stopped`. Eight seconds, as the
	// other Linux adapters give, then KILL.
	if signalPID(pid, syscall.SIGTERM) {
		how = "sigterm"
		if !waitGone(pid, 8) {
			signalPID(pid, syscall.SIGKILL)
			how = "killed after a sigterm it did not finish"
			time.Sleep(time.Second)
		}
	}
	_ = appendText(filepath.Join(workdir, "stop.txt"), delugeStopRecord(utcNow(), n, how))
}
