//go:build !windows

// adapter_libtorrent - the ENGINE-01 capture adapter on Linux, `bit-check
// adapter libtorrent`.
//
// The contract is scripts/capture/adapters/README.md. What the adapter decides
// without a process in it is libtorrent.go, which `bit-check libtorrent
// --selftest` asks on any host.
//
// ⛔ THE PRODUCT IS THE HARNESS IN tools/harness/libtorrent/, run by the venv's
// interpreter against the libtorrent binding each route installed into that
// venv. ⚠ NO WINDOWS HALF: `TODO/engines.md` `ENGINE-01` records what Windows
// lacks.
//
// ⭐ `binary=` IS THE BINDING'S EXTENSION MODULE, the library as each route
// built it. The interpreter is the host's on both routes and would digest
// alike whatever libtorrent it loads, as Deluge's launcher does.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func init() {
	registerAdapter(libtorrentTarget, shAdapter(libtorrentTarget, libtorrentAdapter))
}

// libtorrentVenv is where both routes build the venv, under the prefix.
func libtorrentVenv(k *adapterKit, quiet bool) (string, bool) {
	p, ok := k.prefix(quiet)
	if !ok {
		return "", false
	}
	return filepath.Join(p, "opt", "libtorrent"), true
}

// libtorrentInstalled answers the venv's interpreter, the harness the install
// placed beside it and the binding's module, or "" for each that is not there.
// ⛔ Never PATH: a libtorrent the host carries is not this route's.
func libtorrentInstalled(k *adapterKit) (python, harness, module string) {
	venv, ok := libtorrentVenv(k, true)
	if !ok {
		return "", "", ""
	}
	if p := filepath.Join(venv, "bin", "python"); isExec(p) {
		python = p
	}
	if h := filepath.Join(venv, libtorrentHarnessFile); isFile(h) {
		harness = h
	}
	mods, _ := filepath.Glob(filepath.Join(venv, "lib", "python*", "site-packages", "libtorrent", "__init__*.so"))
	if len(mods) == 1 {
		module = mods[0]
	}
	return python, harness, module
}

func libtorrentAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Printf("target=%s\n", libtorrentTarget)
		fmt.Print("kind=stock\n")
		if _, _, module := libtorrentInstalled(k); module != "" {
			fmt.Printf("binary=%s\n", module)
		}
		fmt.Print(libtorrentDescribe())

	case "install":
		route, workdir, format := k.installArgs(args, libtorrentPackageFormat)
		_ = os.Remove(filepath.Join(workdir, "build-features.txt"))
		switch route {
		case "package":
			libtorrentPackageRoute(k, workdir)
		case "source":
			libtorrentSourceRoute(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		libtorrentFinish(k, workdir, route)
		k.writePackage(workdir, format)

	case "version":
		python, harness, _ := libtorrentInstalled(k)
		if python == "" || harness == "" {
			k.cannot(libtorrentTarget + " is not installed")
		}
		answer, m, err := libtorrentAsk(python, harness)
		if err != nil {
			k.cannot(err.Error())
		}
		v, err := libtorrentRecordVersion(m["library_version"])
		if err != nil {
			k.cannot(err.Error())
		}
		k.versionReport(answer, v)

	case "start":
		libtorrentStart(k, args)

	case "stop":
		if len(args) < 1 {
			k.cannot("stop takes <workdir>")
		}
		libtorrentStop(k, args[0])

	default:
		k.cannot("unknown subcommand: " + sub)
	}
}

// libtorrentAsk runs the harness's `--version`, a pure read.
func libtorrentAsk(python, harness string) (string, map[string]string, error) {
	c := exec.Command(python, "-I", harness, "--version")
	c.Env = libtorrentChildEnv(os.Environ())
	out, rc := capture(c, stdio{in: os.DevNull, err: "&1"})
	answer := strings.TrimRight(out, "\n")
	if rc != 0 {
		return answer, nil, fmt.Errorf("the harness's --version exited %d: %s", rc, firstLine(answer))
	}
	m, err := libtorrentFields(answer, "bit-ids/libtorrent-harness-version/1")
	return answer, m, err
}

// libtorrentPython is the interpreter the venv is made from: the host's
// `python3`, or BIT_IDS_LIBTORRENT_PYTHON for a standalone drive.
func libtorrentPython(k *adapterKit) string {
	if p := os.Getenv("BIT_IDS_LIBTORRENT_PYTHON"); p != "" {
		return p
	}
	p, ok := commandV("python3")
	if !ok {
		k.cannot("the routes make a venv and python3 is not on this host")
	}
	return p
}

// libtorrentMakeVenv makes the venv afresh. On a refusal later in the route
// the half-made venv is removed by libtorrentFail.
func libtorrentMakeVenv(k *adapterKit, workdir string) string {
	venv, ok := libtorrentVenv(k, false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	if os.RemoveAll(venv) != nil || !mkdirP(filepath.Dir(venv)) {
		k.refuse("this user cannot clear " + venv + "; a rootless install writes nowhere else")
	}
	c := exec.Command("timeout", "120", libtorrentPython(k), "-m", "venv", venv)
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), outAppend: true, err: "&1"}) != 0 {
		libtorrentFail(k, "python3 could not make a venv at "+venv)
	}
	return venv
}

// libtorrentFail removes the half-made venv and refuses.
func libtorrentFail(k *adapterKit, msg string) {
	if venv, ok := libtorrentVenv(k, true); ok {
		_ = os.RemoveAll(venv)
	}
	k.refuse(msg)
}

// libtorrentPip runs one pip command in the venv, bounded.
func libtorrentPip(venv, workdir string, seconds int, env []string, args ...string) int {
	full := append([]string{fmt.Sprint(seconds), filepath.Join(venv, "bin", "python"), "-m", "pip"}, args...)
	c := exec.Command("timeout", full...)
	c.Env = env
	return run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), outAppend: true, err: "&1"})
}

// libtorrentPipEnv is the environment pip runs with: the host's, every `PIP_*`
// and `PYTHON*` dropped, so no configuration names another index.
func libtorrentPipEnv() []string {
	var out []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "PIP_") || strings.HasPrefix(upper, "PYTHON") {
			continue
		}
		out = append(out, kv)
	}
	// ⚠ An isolated pip: no user or site configuration file either.
	return append(out, "PIP_CONFIG_FILE="+os.DevNull)
}

// libtorrentPackageRoute is PyPI. ⛔ THE SERIES, NEVER THE VERSION: PyPI's index
// chooses among the 2.0 wheels, and pip's report says which it chose.
func libtorrentPackageRoute(k *adapterKit, workdir string) {
	venv := libtorrentMakeVenv(k, workdir)
	report := filepath.Join(workdir, "pip-report.json")
	rc := libtorrentPip(venv, workdir, 600, libtorrentPipEnv(), "install", "--no-cache-dir", "--disable-pip-version-check",
		"--no-input", "--timeout", "60", "--only-binary", ":all:", "--report", report, libtorrentPipSpec)
	if rc != 0 {
		libtorrentFail(k, fmt.Sprintf("pip could not install %s (exit %d)", libtorrentPipSpec, rc))
	}
	body, _ := os.ReadFile(report)
	version, url, sha, err := delugePipReport(body, libtorrentPyPIName)
	if err != nil {
		libtorrentFail(k, err.Error())
	}
	if msg := libtorrentPinRefusal("PyPI's index", version); msg != "" {
		libtorrentFail(k, msg)
	}
	if !strings.HasPrefix(url, "https://") {
		libtorrentFail(k, "pip's report names the origin ["+url+"], which is not https")
	}
	_ = appendText(filepath.Join(workdir, "build-features.txt"), fmt.Sprintf("wheel=%s\nwheel_sha256=%s\n", url, sha))
	if err := writePackageIndex(workdir, "pypi", libtorrentPyPIName, version, url); err != nil {
		libtorrentFail(k, "the route could not record what the index named")
	}
}

// libtorrentSourceRoute is the tag, cloned and built into a wheel by its own
// setup.py against the Boost the vendor's wheels use.
func libtorrentSourceRoute(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	if tag != libtorrentPinTag {
		k.refuse(libtorrentPinRefusal("the tag "+tag, strings.TrimPrefix(tag, "v")))
	}
	k.need(func(n string) string { return "the source route needs " + n + " and it is not on this host" },
		"git", "curl", "tar", "g++")
	log := filepath.Join(workdir, "install.log")
	src := filepath.Join(workdir, "src")
	boost := filepath.Join(workdir, "boost")
	defer func() {
		// ⚠ Boost alone is 919 megabytes extracted; what the build says of
		// itself is in the workdir's own files.
		_ = os.RemoveAll(boost)
		_ = os.RemoveAll(src)
		_ = os.RemoveAll(filepath.Join(workdir, "wheel"))
	}()
	_ = os.RemoveAll(src)
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/"+libtorrentRepo+".git", src)
	if rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}); rc != 0 {
		k.refuse(fmt.Sprintf("the source route could not clone %s (exit %d)", tag, rc))
	}
	k.cloneCommit(workdir, tag)
	if got := readTrimmed(filepath.Join(workdir, "source-commit")); got != libtorrentPinnedCommit {
		k.refuse("the tag " + tag + " names commit " + got + ", not the pinned " + libtorrentPinnedCommit)
	}
	// ⚠ try_signal is the one submodule the library's build compiles.
	c = exec.Command("timeout", "300", "git", "-C", src, "submodule", "update", "--init", "--depth", "1", "deps/try_signal")
	if rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}); rc != 0 {
		k.refuse(fmt.Sprintf("the source route could not fetch deps/try_signal (exit %d)", rc))
	}

	// ⛔ BOOST FROM WHERE THE VENDOR'S setup_boost.sh FETCHES IT, held to the
	// archive's own digest.
	archive := filepath.Join(workdir, "boost.tar.gz")
	c = exec.Command("curl", "-fsSL", "--proto", "=https", "--proto-redir", "=https", "--max-time", "600",
		"-o", archive, libtorrentBoostURL)
	if rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}); rc != 0 {
		k.refuse(fmt.Sprintf("Boost %s could not be fetched from %s (exit %d)", libtorrentBoostVersion, libtorrentBoostURL, rc))
	}
	sum, _ := fileSHA256(archive)
	if sum != libtorrentBoostPinnedSha256 {
		_ = os.Remove(archive)
		k.refuse("Boost's archive digests as [" + sum + "], not the archive's stated " + libtorrentBoostPinnedSha256)
	}
	if !mkdirP(boost) {
		k.cannot("cannot create " + boost)
	}
	c = exec.Command("tar", "-xzf", archive, "-C", boost, "--strip-components", "1")
	rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"})
	_ = os.Remove(archive)
	if rc != 0 {
		k.refuse("Boost's archive could not be extracted")
	}
	c = exec.Command("timeout", "300", "./bootstrap.sh")
	if rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: boost}); rc != 0 {
		k.refuse(fmt.Sprintf("Boost's bootstrap.sh exited %d; install.log says why", rc))
	}

	venv := libtorrentMakeVenv(k, workdir)
	started := time.Now()
	wheels := filepath.Join(workdir, "wheel")
	// ⛔ `pip wheel` OF THE TAG WITH NO OPTION, which is what cibuildwheel runs:
	// setup.py then adds the vendor's own b2 features. `-v`, so the b2 line is
	// in the log and can be held.
	rc = libtorrentPip(venv, workdir, 1300, libtorrentBuildEnv(libtorrentPipEnv(), boost), "wheel", "-v", "--no-deps",
		"--no-cache-dir", "--disable-pip-version-check", "--no-input", "-w", wheels, src)
	took := int(time.Since(started) / time.Second)
	switch rc {
	case 0:
	case 124:
		libtorrentFail(k, "pip wheel did not finish within 1300s")
	default:
		libtorrentFail(k, fmt.Sprintf("pip wheel exited %d; install.log says why", rc))
	}
	logBody, _ := os.ReadFile(log)
	line, err := libtorrentB2Line(string(logBody))
	if err != nil {
		libtorrentFail(k, err.Error())
	}
	if msg := libtorrentB2Refusal(line); msg != "" {
		libtorrentFail(k, msg)
	}
	built, _ := filepath.Glob(filepath.Join(wheels, "libtorrent-"+libtorrentPin+"-*.whl"))
	if len(built) != 1 {
		libtorrentFail(k, fmt.Sprintf("the build left %d libtorrent %s wheels, not one", len(built), libtorrentPin))
	}
	wsum, _ := fileSHA256(built[0])
	if rc := libtorrentPip(venv, workdir, 120, libtorrentPipEnv(), "install", "--no-index", "--no-deps",
		"--disable-pip-version-check", built[0]); rc != 0 {
		libtorrentFail(k, fmt.Sprintf("pip could not install the built wheel (exit %d)", rc))
	}
	ssl, _ := capture(exec.Command("dpkg-query", "-W", "-f", "${Package} ${Version}\n", "libssl-dev"),
		stdio{in: os.DevNull, err: os.DevNull})
	_ = appendText(filepath.Join(workdir, "build-features.txt"), fmt.Sprintf(
		"b2=%s\nboost=%s\nboost_url=%s\nboost_sha256=%s\nlibssl_dev=%s\nwheel=%s\nwheel_sha256=%s\nbuild_seconds=%d\n",
		line, libtorrentBoostVersion, libtorrentBoostURL, libtorrentBoostPinnedSha256, strings.TrimSpace(ssl),
		filepath.Base(built[0]), wsum, took))
}

// libtorrentFinish is what both routes end with: the harness placed beside
// the venv, the build asked, and what it said held and kept.
func libtorrentFinish(k *adapterKit, workdir, route string) {
	venv, _ := libtorrentVenv(k, true)
	root, err := captureRoot()
	if err != nil {
		libtorrentFail(k, err.Error())
	}
	harnessSrc := filepath.Join(root, filepath.FromSlash(libtorrentHarnessDir), libtorrentHarnessFile)
	if d := os.Getenv("BIT_IDS_LIBTORRENT_HARNESS"); d != "" {
		harnessSrc = d
	}
	body, err := os.ReadFile(harnessSrc)
	if err != nil {
		libtorrentFail(k, "the harness is not at "+harnessSrc)
	}
	harness := filepath.Join(venv, libtorrentHarnessFile)
	if err := os.WriteFile(harness, body, 0o644); err != nil {
		libtorrentFail(k, "the harness could not be placed at "+harness)
	}
	python := filepath.Join(venv, "bin", "python")
	c := exec.Command(python, "-m", "pip", "freeze", "--all")
	c.Env = libtorrentPipEnv()
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "pip-freeze.txt"), err: filepath.Join(workdir, "install.log"), errAppend: true}) != 0 {
		libtorrentFail(k, "pip could not say what it installed")
	}
	answer, m, err := libtorrentAsk(python, harness)
	_ = writeText(filepath.Join(workdir, "install-version.log"), answer+"\n")
	if err != nil {
		libtorrentFail(k, err.Error()+"; install-version.log has the whole answer")
	}
	if msg := libtorrentAnswerRefusal(m); msg != "" {
		libtorrentFail(k, msg+"; install-version.log has the whole answer")
	}
	c = exec.Command(python, "-I", harness, "--defaults")
	c.Env = libtorrentChildEnv(os.Environ())
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "default-settings.txt"), err: filepath.Join(workdir, "install.log"), errAppend: true}) != 0 {
		libtorrentFail(k, "the harness could not list the build's default settings")
	}
	hsum, _ := fileSHA256(harness)
	pyVersion, _ := capture(exec.Command(libtorrentPython(k), "--version"), stdio{in: os.DevNull, err: "&1"})
	var note strings.Builder
	fmt.Fprintf(&note, "route=%s\nlibrary=%s\nversion=%s\nharness=%s/%s\nharness_sha256=%s\npython=%s\n",
		route, libtorrentRepo, libtorrentPin, libtorrentHarnessDir, libtorrentHarnessFile, hsum, strings.TrimSpace(pyVersion))
	if c, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
		fmt.Fprintf(&note, "tree_commit=%s\n", strings.TrimSpace(string(c)))
	}
	note.WriteString(answer + "\n")
	_ = appendText(filepath.Join(workdir, "build-features.txt"), note.String())
}

func libtorrentStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	python, harness, _ := libtorrentInstalled(k)
	if python == "" || harness == "" {
		k.cannot(libtorrentTarget + " is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	// ⛔ A HOME AND A DATA DIRECTORY OF ITS OWN PER SESSION.
	home := filepath.Join(workdir, "home")
	downloads := filepath.Join(workdir, "downloads")
	for _, f := range []string{home, downloads, filepath.Join(workdir, "ready.txt"), filepath.Join(workdir, "stop-request")} {
		_ = os.RemoveAll(f)
	}
	if !mkdirP(home, downloads) {
		k.cannot("cannot create the harness's directories under " + workdir)
	}
	alerts := ""
	for n := 1; ; n++ {
		alerts = filepath.Join(workdir, fmt.Sprintf("alerts-%d.log", n))
		if !exists(alerts) {
			break
		}
	}
	flags, err := libtorrentStartArgs(harness, torrent, workdir, alerts, port)
	if err != nil {
		k.cannot(err.Error())
	}
	if msg := libtorrentArgsRefusal(flags); msg != "" {
		k.refuse("the command line would start the harness with " + msg)
	}
	_ = writeText(filepath.Join(workdir, "command-line.txt"), strings.Join(append([]string{python}, flags...), " ")+"\n")
	c := exec.Command(python, flags...)
	c.Env = libtorrentChildEnv(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+home+"/.config",
		"XDG_DATA_HOME="+home+"/.local/share", "XDG_CACHE_HOME="+home+"/.cache")
	pid, err := startBackground(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "client.out"), outAppend: true, err: "&1"})
	if err != nil {
		k.cannot(libtorrentTarget + " could not be started: " + err.Error())
	}
	ps := fmt.Sprint(pid)
	_ = writeText(filepath.Join(workdir, "pid"), ps+"\n")
	listen, err := libtorrentDrive(workdir, port, func(d time.Duration) bool {
		time.Sleep(d)
		return !alive(ps)
	})
	if err != nil {
		k.refuse(err.Error())
	}
	// ⭐ THE SOCKETS THE HARNESS HOLDS, read rather than assumed.
	if _, ok := commandV("ss"); ok {
		out, _ := capture(exec.Command("ss", "-tuanp"), stdio{in: os.DevNull, err: os.DevNull})
		var mine []string
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "pid="+ps+",") {
				mine = append(mine, l)
			}
		}
		_ = writeText(filepath.Join(workdir, "sockets.txt"), strings.Join(mine, "\n")+"\n")
	}
	fmt.Printf("started %s pid %s on %s\n", libtorrentTarget, ps, listen)
}

// libtorrentStop asks the library to shut down and scrubs what the session
// left. ⭐ THE SHUTDOWN IS THE LIBRARY'S OWN: the stop file, which the harness
// polls, and the session's destructor after it. KILL after the grace.
func libtorrentStop(k *adapterKit, workdir string) {
	home := filepath.Join(workdir, "home")
	downloads := filepath.Join(workdir, "downloads")
	defer func() {
		for _, d := range []string{home, downloads} {
			if listing := rqbitStateListing(d); listing != "" {
				appendText(filepath.Join(workdir, "state.txt"), utcNow()+" the session left under "+filepath.Base(d)+":\n"+listing)
			}
			_ = os.RemoveAll(d)
		}
		_ = os.Remove(filepath.Join(workdir, "stop-request"))
	}()
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	how := "already exited"
	if alive(pid) {
		_ = writeText(filepath.Join(workdir, "stop-request"), utcNow()+"\n")
		how = "closed on the stop file"
		if !waitGone(pid, libtorrentGraceSeconds) {
			signalPID(pid, syscall.SIGKILL)
			waitGone(pid, 5)
			how = fmt.Sprintf("killed after the stop file went unanswered for %ds", libtorrentGraceSeconds)
		}
	}
	appendText(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %s: %s\n", utcNow(), pid, how))
}
