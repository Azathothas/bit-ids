//go:build windows

// adapter_biglybt_windows - the CLIENT-09 capture adapter, Windows half.
//
// The contract is scripts/capture/adapters/README.md and the Linux half is
// adapter_biglybt.go; this is the same five subcommands for Windows, run as
// `bit-check adapter biglybt <subcommand>`. What it decides without Windows in
// it is in biglybt.go, which `bit-check biglybt --selftest` asks on any host.
//
// ⚠ THE MECHANICS ARE THE OTHER WINDOWS HALVES', REUSED AND NOT COPIED:
// qbtWinRun, qbtWinWaiter, the exit type, trWinSystemTool and
// trWinWingetVersion. They carry no decision of those products.
//
// -- ⭐ WHAT IS THE SAME AS LINUX ------------------------------------------------
//
// The vendor's console UI started headless with `biglybt.config` written whole
// per session, the torrent handed to the running instance by a second launch,
// `version` as the probe class run on the route's own JVM, and the scrub of
// the config directory on every path out of `stop`.
//
// -- ⛔ WHAT DIFFERS, AND WHY EACH IS A FACT OF THE BUILD ------------------------
//
//   - The JVM is the route's, not the host's. The vendor's installer bundles a
//     JRE in `<install>\jre`; the NoJRE installer winget runs records the JVM
//     it chose in `.install4j\inst_jre.cfg`, which the vendor's launcher reads.
//     bbWinJava answers in that order and `jvm.txt` says which.
//   - The vendor's launcher, `BiglyBT.exe`, is a windowed install4j launcher
//     with no console, so the JVM is run as `java.exe` with the vendor's
//     `BiglyBT.vmoptions`, the launch bbLaunchArgs composes.
//   - `stop` asks the running instance to close down with the vendor's own
//     `--closedown`, sent over the instance port by a second launch, because
//     Windows has no SIGTERM for a console process in another process group;
//     the core gets twenty seconds and is then ended with `taskkill /T /F`.
//     `stop.txt` says which.
//   - `helpers=` names the route's `java.exe` and `taskkill.exe`. ⛔ The JVM
//     is the process that opens every socket, so the firewall has to name it;
//     `binary=` is the jar, the bytes the route installed.
//
// -- the routes -----------------------------------------------------------------
//
// ⛔ `release`: the vendor's `GitHub_BiglyBT_Windows_Installer.exe`, install4j
// with a bundled JRE, fetched under the vendor's `GitHub_sha256sum.txt` and
// the listing's digest, run `-q -dir %ProgramFiles%\BiglyBT`.
// ⛔ `package`: winget's `BiglySoftware.BiglyBT`, no version. Its 4.1 manifest
// installs the vendor's `GitHub_BiglyBT_Windows_NoJRE.exe` from the same
// release, https, and declares `EclipseAdoptium.Temurin.25.JRE` as its
// dependency. ⚠ The index spells the version `4.1`. Scoop's `nonportable`
// bucket stops at 4.0.0.0 and Chocolatey at 3.5.0.0.
// ⚠ So the two Windows routes install one jar on two JVMs, and the record
// carries both JVMs' own answers. `CLIENT-09`.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"context"
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

func init() { registerAdapter(bbTarget, bbWinAdapter) }

func bbWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = bbWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = bbWinInstall(args[0], args[1])
		}
	case "version":
		err = bbWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = bbWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = bbWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", bbTarget, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", bbTarget, err)
	return 2
}

// bbWinDir is where both routes have the vendor's installer place the build:
// install4j's default for this product, named to it with `-dir` on the
// release route.
func bbWinDir() string {
	if v := os.Getenv("BIT_IDS_BIGLYBT_HOME"); v != "" {
		return v
	}
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return ""
	}
	return filepath.Join(pf, "BiglyBT")
}

// bbWinJar is the installed jar, or "".
func bbWinJar() string {
	dir := bbWinDir()
	if dir == "" {
		return ""
	}
	if jar := filepath.Join(dir, bbJar); isRegular(jar) {
		return jar
	}
	return ""
}

// bbWinJava is the JVM the vendor's launcher would run, and where it was
// named: the bundled JRE, then install4j's preference file, then the JVM the
// installer recorded.
func bbWinJava(dir string) (string, string) {
	if v := os.Getenv("BIT_IDS_BIGLYBT_JAVA"); v != "" {
		return v, "BIT_IDS_BIGLYBT_JAVA"
	}
	if j := filepath.Join(dir, "jre", "bin", "java.exe"); isRegular(j) {
		return j, "bundled jre"
	}
	for _, cfg := range []string{"pref_jre.cfg", "inst_jre.cfg"} {
		b, err := os.ReadFile(filepath.Join(dir, ".install4j", cfg))
		if err != nil {
			continue
		}
		if home := bbJRECfg(string(b)); home != "" {
			if j := filepath.Join(home, "bin", "java.exe"); isRegular(j) {
				return j, ".install4j\\" + cfg
			}
		}
	}
	return "", ""
}

func bbWinDescribe() error {
	var b strings.Builder
	fmt.Fprintf(&b, "target=%s\nkind=stock\n", bbTarget)
	jar := bbWinJar()
	if jar != "" {
		fmt.Fprintf(&b, "binary=%s\n", jar)
	}
	b.WriteString(bbDescribeScheme(bbWindowsAsset))
	b.WriteString("routes=release,package\n")
	var helpers []string
	if jar != "" {
		if j, _ := bbWinJava(filepath.Dir(jar)); j != "" {
			helpers = append(helpers, j)
		}
	}
	if tk := trWinSystemTool("taskkill.exe"); tk != "" {
		helpers = append(helpers, tk)
	}
	if len(helpers) > 0 {
		fmt.Fprintf(&b, "helpers=%s\n", strings.Join(helpers, "|"))
	}
	fmt.Print(b.String())
	return nil
}

// -- install --------------------------------------------------------------------

func bbWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := bbPackageFormat(true, route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	var err error
	switch route {
	case "release":
		err = bbWinInstallRelease(workdir, log)
	case "package":
		err = bbWinInstallPackage(workdir, log)
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

// bbWinAfterInstall ends any BiglyBT the installer started, records the
// install, digests it and asks the build. ⚠ install4j's finish screen can
// launch the product; an unattended install should not, and one that did
// would be a session on the vendor's defaults before the capture, so it is
// ended and `launched.txt` says so.
func bbWinAfterInstall(workdir, dir, want, from string) error {
	if tk := trWinSystemTool("taskkill.exe"); tk != "" {
		rc := qbtWinRun(filepath.Join(workdir, "launched.log"), tk, "/IM", "BiglyBT.exe", "/T", "/F")
		appendFile(filepath.Join(workdir, "launched.txt"), fmt.Sprintf("%s taskkill /IM BiglyBT.exe exited %d (128 is none running)\n", utcNow(), rc))
	}
	jar := filepath.Join(dir, bbJar)
	if !isRegular(jar) || !isRegular(filepath.Join(dir, bbCLIJar)) {
		listing, _ := os.ReadDir(dir)
		var names []string
		for _, e := range listing {
			names = append(names, e.Name())
		}
		return qbtRefuse("the installer reported success and left no %s with %s beside it in %s (it holds: %s)", bbJar, bbCLIJar, dir, strings.Join(names, ", "))
	}
	if err := bbRecordTreeWin(dir, filepath.Join(workdir, "installed-files.txt")); err != nil {
		return qbtRefuse("the installed tree could not be digested: %v", err)
	}
	output, a, err := bbWinAsk(jar)
	_ = os.WriteFile(filepath.Join(workdir, "install-version.log"), []byte(output+"\n"), 0o644)
	if err != nil {
		return qbtRefuse("the build could not be asked its version: %v; install-version.log has the answer", err)
	}
	if want != "" && !bbSameVersion(want, a.version) {
		return qbtRefuse("%s named %s and the build it installed reports %s", from, want, a.version)
	}
	return nil
}

func bbWinInstallRelease(workdir, log string) error {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION")
	if res == "" {
		return qbtCannot("the release route takes its digest disposition from BIT_IDS_RELEASE_RESOLUTION")
	}
	body, _ := os.ReadFile(res)
	want := recordField(body, "selected_version")
	dir := bbWinDir()
	if dir == "" {
		return qbtCannot("%%ProgramFiles%% is not set, so there is no install location to name")
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	setup := filepath.Join(workdir, "BiglyBT_Windows_Installer.exe")
	switch rc := qbtWinRun(log, self, "install-rootless", "--fetch", "--url", url, "--into", setup,
		"--from-resolution", res, "--report", filepath.Join(workdir, "rootless-install.txt")); rc {
	case 0:
	case 1:
		return qbtRefuse("the release route could not fetch the vendor's installer")
	default:
		return qbtCannot("the rootless installer could not run for the release route (exit %d)", rc)
	}
	args, err := bbInstall4jArgs(dir, filepath.Join(workdir, "install4j.log"))
	if err != nil {
		return qbtCannot("%v", err)
	}
	rc := qbtWinRun(log, setup, args...)
	sum, _ := fileSHA256(setup)
	note := fmt.Sprintf("installer=%s\ninstaller_sha256=%s\nargs=%s\nexit=%d\n", filepath.Base(setup), sum, strings.Join(args, " "), rc)
	if err := os.WriteFile(filepath.Join(workdir, "install4j-install.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record how it ran the installer")
	}
	if rc != 0 {
		return qbtRefuse("the vendor's installer exited %d; install4j.log has its words", rc)
	}
	return bbWinAfterInstall(workdir, dir, want, "the release resolution")
}

func bbWinInstallPackage(workdir, log string) error {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return qbtCannot("the package route installs through winget and it is not on this host")
	}
	show := filepath.Join(workdir, "winget-show.txt")
	if rc := qbtWinRun(show, winget, "show", "--id", bbWingetID, "--exact", "--source", "winget",
		"--architecture", "x64", "--accept-source-agreements", "--disable-interactivity"); rc != 0 {
		return qbtRefuse("winget's index could not describe %s (exit %d)", bbWingetID, rc)
	}
	shown, _ := os.ReadFile(show)
	indexVersion, err := trWinWingetVersion(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	origin, err := wingetInstallerURL(string(shown))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if !strings.HasPrefix(origin, "https://") {
		return qbtRefuse("winget's index pins %s, and a record may not name a retrieval that is not https", origin)
	}
	// ⛔ NO `--version`: the index is this route's resolver. ⚠ `--scope
	// machine` and `--architecture x64` because the manifest declares the one
	// and offers two of the other. ⛔ AND NO `--location`: 4.1's manifest
	// declares no InstallLocation switch, so the installer lays the build out at
	// its own default, which bbWinDir names.
	rc := qbtWinRun(log, winget, "install", "--id", bbWingetID, "--exact", "--source", "winget",
		"--scope", "machine", "--architecture", "x64", "--silent", "--disable-interactivity",
		"--accept-package-agreements", "--accept-source-agreements",
		"--log", filepath.Join(workdir, "winget.log"))
	if rc != 0 {
		return qbtRefuse("the package route did not install %s (winget exit %d)", bbWingetID, rc)
	}
	if err := writePackageIndex(workdir, "winget", bbWingetID, indexVersion, origin); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	dir := bbWinDir()
	if dir == "" {
		return qbtCannot("%%ProgramFiles%% is not set, so there is no install location to look in")
	}
	if err := bbWinAfterInstall(workdir, dir, indexVersion, "winget's index"); err != nil {
		return err
	}
	note := fmt.Sprintf("index=%s\nindex_version=%s\norigin=%s\n", bbWingetID, indexVersion, origin)
	if err := os.WriteFile(filepath.Join(workdir, "winget-version.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	return nil
}

// bbRecordTreeWin is bbRecordTree's Windows twin: every file under root with its
// sha256, sorted by path.
func bbRecordTreeWin(root, out string) error {
	var lines []string
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || !fi.Mode().IsRegular() {
			return err
		}
		sum, err := fileSHA256(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		lines = append(lines, sum+"  "+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	return os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// -- asking the build -----------------------------------------------------------

// bbWinRunEnv runs a second launch with stdin empty, an environment of its own
// and both streams appended to log, and kills it after 60 seconds, answering
// 124. ⚠ A second launch that cannot reach the instance port goes on to start
// a core of its own, so it is never left to run.
func bbWinRunEnv(log string, env []string, prog string, args ...string) int {
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, f, f
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 124
	}
	return exitOf(err)
}

// bbWinAsk runs the probe class on the route's JVM against the jar.
func bbWinAsk(jar string) (string, bbAnswer, error) {
	java, _ := bbWinJava(filepath.Dir(jar))
	if java == "" {
		return "", bbAnswer{}, fmt.Errorf("no JVM: no bundled jre and no .install4j JRE file in %s", filepath.Dir(jar))
	}
	tmp, err := os.MkdirTemp("", "biglybt-version-")
	if err != nil {
		return "", bbAnswer{}, fmt.Errorf("cannot make a temporary directory")
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, bbProbeClass+".class"), bbVersionClass(), 0o644); err != nil {
		return "", bbAnswer{}, fmt.Errorf("cannot write the probe")
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		return "", bbAnswer{}, err
	}
	defer null.Close()
	cmd := exec.Command(java, "-cp", tmp+";"+jar, bbProbeClass)
	cmd.Env, _ = bbJavaEnv(os.Environ())
	cmd.Stdin = null
	out, err := cmd.CombinedOutput()
	output := strings.TrimRight(string(out), "\r\n")
	if err != nil {
		return output, bbAnswer{}, fmt.Errorf("the probe exited %d", exitOf(err))
	}
	a, err := bbParseAnswer(output)
	return output, a, err
}

func bbWinVersion() error {
	jar := bbWinJar()
	if jar == "" {
		return qbtCannot("%s is not installed", bbJar)
	}
	output, a, err := bbWinAsk(jar)
	if err != nil {
		return qbtCannot("%v", err)
	}
	fmt.Fprintf(os.Stderr, "%s reported:\n%s\n", bbTarget, output)
	fmt.Println(a.version)
	return nil
}

// -- start and stop ---------------------------------------------------------------

// bbWinLaunch starts the JVM with stdin empty, an environment of its own and
// both streams appended to out, in a process group of its own, and returns.
func bbWinLaunch(out string, env []string, dir, exe string, args ...string) (*exec.Cmd, error) {
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
	cmd.Env, cmd.Dir = env, dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, f, f
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// bbWinSession is a session's launch: the JVM, its environment and the
// vendor's options, read once.
type bbWinSession struct {
	java, from, dir, config string
	env, vmopts             []string
	instance, major         int
}

func bbWinSessionOf(workdir string) (bbWinSession, error) {
	s := bbWinSession{config: filepath.Join(workdir, "config")}
	jar := bbWinJar()
	if jar == "" {
		return s, qbtCannot("%s is not installed", bbJar)
	}
	s.dir = filepath.Dir(jar)
	s.java, s.from = bbWinJava(s.dir)
	if s.java == "" {
		return s, qbtCannot("no JVM: no bundled jre and no .install4j JRE file in %s", s.dir)
	}
	if b, err := os.ReadFile(filepath.Join(s.dir, "BiglyBT.vmoptions")); err == nil {
		s.vmopts = bbVMOptions(string(b))
	}
	s.env, _ = bbJavaEnv(os.Environ())
	return s, nil
}

func (s bbWinSession) args(extra ...string) []string {
	return bbLaunchArgs(s.dir, ";", s.config, s.instance, s.major, s.vmopts, extra...)
}

func bbWinStart(torrent, workdir, peerPort string) error {
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	s, err := bbWinSessionOf(workdir)
	if err != nil {
		return err
	}
	output, a, err := bbWinAsk(filepath.Join(s.dir, bbJar))
	if err != nil {
		return qbtCannot("%v", err)
	}
	s.major = bbJavaMajor(a.java)
	appendFile(filepath.Join(workdir, "jvm.txt"), fmt.Sprintf("## %s java=%s from=%s vmoptions=%s\n%s\n", utcNow(), s.java, s.from, strings.Join(s.vmopts, " "), output))
	downloads := filepath.Join(workdir, "downloads")
	if err := os.RemoveAll(s.config); err != nil {
		return qbtCannot("cannot clear %s", s.config)
	}
	for _, d := range []string{s.config, downloads} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return qbtCannot("cannot create %s", d)
		}
	}
	doc, err := bbConfig(downloads, peerPort)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if msg := bbConfigRefusal(doc); msg != "" {
		return qbtRefuse("the settings would start the build with %s", msg)
	}
	if err := os.WriteFile(filepath.Join(s.config, "biglybt.config"), doc, 0o600); err != nil {
		return qbtCannot("the settings could not be written")
	}
	if err := os.WriteFile(filepath.Join(workdir, "settings.txt"), []byte(bbSettingsText(doc)), 0o644); err != nil {
		return qbtCannot("the settings could not be recorded")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return qbtCannot("no loopback port is free for the client's instance port")
	}
	s.instance = l.Addr().(*net.TCPAddr).Port
	l.Close()
	if err := os.WriteFile(filepath.Join(workdir, "instance-port"), []byte(strconv.Itoa(s.instance)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the instance port")
	}
	launch := s.args("--ui=console")
	appendFile(filepath.Join(workdir, "launch.txt"), fmt.Sprintf("## %s\n%s %s\n", utcNow(), s.java, strings.Join(launch, " ")))
	out := filepath.Join(workdir, "client.out")
	var offset int64
	if fi, err := os.Stat(out); err == nil {
		offset = fi.Size()
	}
	session := func() string {
		b, _ := os.ReadFile(out)
		if int64(len(b)) < offset {
			return ""
		}
		return string(b[offset:])
	}
	cmd, err := bbWinLaunch(out, s.env, workdir, s.java, launch...)
	if err != nil {
		return qbtCannot("the JVM could not be started: %v", err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the JVM's process id")
	}
	exited := qbtWinWaiter(cmd.Process)
	// ⛔ THE CORE FIRST, THEN THE TORRENT, for the Linux half's measured reason.
	started := false
	for i := 0; i < 90 && !started; i++ {
		switch bbCoreState(session()) {
		case "started":
			started = true
			continue
		case "failed":
			return qbtRefuse("the core did not start; client.out has its words")
		}
		if exited(time.Second) {
			return qbtRefuse("the JVM exited before the core started; client.out has its words")
		}
	}
	if !started {
		return qbtRefuse("the core did not start within 90s")
	}
	if rc := bbWinRunEnv(filepath.Join(workdir, "handover.out"), s.env, s.java, s.args("--ui=console", torrent)...); rc != 0 {
		return qbtRefuse("the torrent could not be handed to the running instance (exit %d); handover.out has its words", rc)
	}
	for i := 0; i < 30; i++ {
		if bbOpened(session(), torrent) {
			fmt.Printf("started BiglyBT %s pid %d on port %s, Java %s\n", a.version, pid, peerPort, a.java)
			return nil
		}
		if exited(time.Second) {
			return qbtRefuse("the JVM exited after the torrent was handed over; client.out has its words")
		}
	}
	return qbtRefuse("the running instance never opened the torrent")
}

func bbWinStop(workdir string) error {
	// ⛔ THE KEYS LEAVE THE WORKDIR ON EVERY PATH OUT. See bbScrub.
	defer func() {
		gone, err := bbScrub(workdir, filepath.Join(workdir, "config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: the workdir could not be scrubbed: %v\n", bbTarget, err)
		} else if len(gone) > 0 {
			fmt.Fprintf(os.Stderr, "%s: removed %d file(s) of the client's config and keys\n", bbTarget, len(gone))
		}
	}()
	pid, err := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "pid"))))
	if err != nil || pid <= 1 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		appendFile(filepath.Join(workdir, "stop.txt"), bbStopRecord(utcNow(), pid, "already exited"))
		return nil
	}
	exited := qbtWinWaiter(p)
	how := "already exited"
	if !exited(200 * time.Millisecond) {
		how = "killed"
		s, err := bbWinSessionOf(workdir)
		instance, _ := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "instance-port"))))
		if err == nil && instance > 0 {
			s.instance = instance
			if _, a, err := bbWinAsk(filepath.Join(s.dir, bbJar)); err == nil {
				s.major = bbJavaMajor(a.java)
			}
			// ⭐ THE VENDOR'S OWN CLOSEDOWN, over the instance port.
			bbWinRunEnv(filepath.Join(workdir, "stop.log"), s.env, s.java, s.args("--closedown")...)
			if exited(20 * time.Second) {
				how = "closedown"
			} else {
				how = "killed after a closedown it did not finish"
			}
		}
		if how != "closedown" {
			if tk := trWinSystemTool("taskkill.exe"); tk != "" {
				qbtWinRun(filepath.Join(workdir, "stop.log"), tk, "/PID", strconv.Itoa(pid), "/T", "/F")
			}
			p.Kill()
			exited(5 * time.Second)
		}
	}
	appendFile(filepath.Join(workdir, "stop.txt"), bbStopRecord(utcNow(), pid, how))
	fmt.Printf("stopped BiglyBT pid %d: %s\n", pid, how)
	return nil
}
