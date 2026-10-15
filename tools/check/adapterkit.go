//go:build !windows

// adapterkit - what the four Linux capture adapters share, and nothing they
// decide.
//
// CI-11. The adapters were POSIX `sh` files under scripts/capture/adapters/,
// and each carried its own copy of `refuse`, `cannot`, `rootless`, `prefix`
// and a `kill` loop. ⭐ Those are mechanics, so they live here once; every
// sentence about a PRODUCT - its routes, its switches, how it spells its
// version - stays in that product's own adapter_<name>.go, which is the file
// the contract in scripts/capture/adapters/README.md says knows it.
//
// ⛔ A SHELL'S SEMANTICS ARE KEPT ON PURPOSE, because the port is held to the
// shell half's behaviour and not to a cleaner idea of it:
//
//   - a command with no redirection inherits the adapter's own stream, as a
//     shell command does. `dpkg-deb -x` printed to the caller's stdout in the
//     shell half and it still does.
//   - a command started in the background ignores SIGINT and SIGQUIT and reads
//     the null device, which is what a non-interactive shell gives `&`.
//   - a background child is reaped as soon as it exits, which a shell does at
//     its next foreground command. ⚠ Without that a dead daemon is a zombie,
//     `kill -0` keeps answering for it, and a start loop waits out its whole
//     bound over a build that refused its options at once.
//   - `$(...)` drops every trailing newline, and only those.
//
// ⚠ WHAT IS DELIBERATELY NOT KEPT: the shell half found `bit-check` through
// BIT_CHECK and was could-not-run without it. A Go adapter IS `bit-check`, so
// the prefix is rootlessPrefix() in process - the one derivation - and the
// installer is this same executable in a child process.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// adapterExit is `exit N` from anywhere inside an adapter.
type adapterExit int

// adapterKit is one adapter call's mechanics: its name, which every message
// leads with, as `$ME` did.
type adapterKit struct{ me string }

// shAdapter wraps an adapter body so `refuse`, `cannot` and `exit` end it with
// their code, wherever they are called from.
func shAdapter(me string, body func(k *adapterKit, sub string, args []string)) goAdapter {
	return func(sub string, args []string) (code int) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(adapterExit); ok {
					code = int(e)
					return
				}
				panic(r)
			}
		}()
		body(&adapterKit{me: me}, sub, args)
		return 0
	}
}

func (k *adapterKit) say(msg string) { fmt.Fprintf(os.Stderr, "%s: %s\n", k.me, msg) }

// refuse is exit 1 with the adapter's own words.
func (k *adapterKit) refuse(msg string) { k.say(msg); panic(adapterExit(1)) }

// cannot is exit 2 with the adapter's own words.
func (k *adapterKit) cannot(msg string) { k.say(msg); panic(adapterExit(2)) }

func (k *adapterKit) exit(code int) { panic(adapterExit(code)) }

// -- the prefix and the installer -----------------------------------------
//
// ⛔ THE PREFIX IS ASKED FOR, NEVER COMPOSED. rootlessPrefix() is the one
// derivation in this tree, so no adapter can disagree with the installer about
// where a build went. ⚠ The shell half asked `bit-check install-rootless
// --prefix`, whose refusal reached stderr in that command's words; they are
// printed here in the same words.
func (k *adapterKit) prefix(quiet bool) (string, bool) {
	p, err := rootlessPrefix()
	if err != nil {
		if !quiet {
			msg := err.Error()
			fmt.Fprintf(os.Stderr, "install-rootless: %s\n", msg)
		}
		return "", false
	}
	return p, true
}

// self is this executable, which is both the installer and the adapter.
func (k *adapterKit) self() string {
	p, err := os.Executable()
	if err != nil {
		k.cannot("cannot name this executable: " + err.Error())
	}
	return p
}

// rootless is `bit-check install-rootless <args> </dev/null >>log 2>&1`.
// ⛔ A CHILD PROCESS, NOT A CALL, because the installer prints to its own
// stdout and stderr and the shell half sent both to the log.
func (k *adapterKit) rootless(log string, args ...string) int {
	c := exec.Command(k.self(), append([]string{"install-rootless"}, args...)...)
	return run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"})
}

// describeField is `sh "$0" describe | awk -F= '$1 == KEY { sub(/^[^=]*=/, ""); print; exit }'`.
// ⛔ ASKED OF THIS ADAPTER'S OWN `describe` RATHER THAN SPELLED AGAIN, in a
// child process as the shell half asked it, so a describe that writes to
// stderr still does.
func (k *adapterKit) describeField(key string) string {
	c := exec.Command(k.self(), "adapter", k.me, "describe")
	out, _ := capture(c, stdio{})
	for _, line := range strings.Split(out, "\n") {
		name, value, _ := strings.Cut(line, "=")
		if name == key {
			return value
		}
	}
	return ""
}

// releaseDisposition is the shell half's two-branch choice of the digest
// disposition: the resolution's record, or this adapter's own declaration.
//
// ⛔ THE DIGEST DISPOSITION IS THE RESOLUTION'S AND THE INSTALLER READS IT.
// `ACQ-06`. Four adapters choosing between four dispositions would be four
// copies of one decision, which is the one-gated-door shape
// `docs/methodology/reviews.md` names the most recurring hole there is.
// ⚠ The second branch is the standalone drive - an adapter run with a URL and
// no resolution - and it carries this adapter's own declaration, asked of
// `describe` rather than spelled a second time.
func (k *adapterKit) releaseDisposition() []string {
	if res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION"); res != "" {
		return []string{"--from-resolution", res}
	}
	why := k.describeField("release_digest_unpublished")
	if why == "" {
		k.cannot("this adapter declares no digest disposition for its release route")
	}
	return []string{"--digest-unpublished", why}
}

// rootlessRelease runs one release route's installer call and answers its
// refusal the way every shell half did: the log's tail, then 2 for an installer
// that could not run and 1 for one that refused.
func (k *adapterKit) rootlessRelease(workdir, refusal string, args ...string) {
	log := filepath.Join(workdir, "install.log")
	rc := k.rootless(log, args...)
	if rc == 0 {
		return
	}
	c := exec.Command("tail", "-20", log)
	run(c, stdio{out: "&2"})
	if rc != 1 {
		k.cannot(fmt.Sprintf("the rootless installer could not run for the release route (exit %d)", rc))
	}
	k.refuse(refusal)
}

// -- finding a build --------------------------------------------------------

// lookup is the shell halves' `binary()`: an override, then the prefix, then
// PATH. ⛔ THE ROUTE'S OWN PREFIX, BEFORE THE PATH. Without it the two routes
// would race on PATH order, which is a property of the host rather than of the
// acquisition. ⚠ quiet is `prefix 2>/dev/null`, and a prefix that cannot be
// derived then falls through to PATH rather than failing the lookup.
func (k *adapterKit) lookup(override, exe string, quiet bool) (string, bool) {
	if override != "" {
		return override, true
	}
	p, ok := k.prefix(quiet)
	if !ok && !quiet {
		return "", false
	}
	if ok && isExec(filepath.Join(p, "bin", exe)) {
		return filepath.Join(p, "bin", exe), true
	}
	return commandV(exe)
}

// commandV is `command -v NAME`.
func commandV(name string) (string, bool) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return p, true
}

// need is the shell halves' `command -v X >/dev/null 2>&1 || cannot ...`.
func (k *adapterKit) need(msg func(string) string, names ...string) {
	for _, n := range names {
		if _, ok := commandV(n); !ok {
			k.cannot(msg(n))
		}
	}
}

// isExec is `[ -x PATH ]`.
func isExec(path string) bool { return syscall.Access(path, 1) == nil }

// isFile is `[ -f PATH ]`.
func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// exists is `[ -e PATH ]`.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isNonEmpty is `[ -s PATH ]`.
func isNonEmpty(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}

// isSocket is `[ -S PATH ]`.
func isSocket(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode()&os.ModeSocket != 0
}

// -- running a command, as a shell line would --------------------------------

// stdio is one command's redirections. An empty path inherits the adapter's
// own stream, as a shell command with no redirection does; "&1" is `2>&1` and
// "&2" is `>&2`.
type stdio struct {
	in        string
	out       string
	outAppend bool
	err       string
	errAppend bool
	env       []string // VAR=value pairs in front of the command
	dir       string   // `(cd DIR && ...)`
}

func openRedirect(path string, appendMode bool) (*os.File, error) {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if appendMode {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}
	return os.OpenFile(path, flag, 0o666)
}

// wire applies s to c and answers the files to close after it ran. ⚠ A
// redirection that cannot be opened is a failed command, as it is in a shell,
// which answers 2 for it.
func wire(c *exec.Cmd, s stdio) (closers []*os.File, ok bool) {
	open := func(path string, appendMode bool) *os.File {
		f, err := openRedirect(path, appendMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot create %s: %v\n", path, err)
			ok = false
			return nil
		}
		closers = append(closers, f)
		return f
	}
	ok = true
	c.Stdin = os.Stdin
	if s.in != "" {
		f, err := os.Open(s.in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open %s: %v\n", s.in, err)
			return closers, false
		}
		closers = append(closers, f)
		c.Stdin = f
	}
	if c.Stdout == nil {
		switch s.out {
		case "":
			c.Stdout = os.Stdout
		case "&2":
			c.Stdout = os.Stderr
		default:
			if f := open(s.out, s.outAppend); f != nil {
				c.Stdout = f
			}
		}
	}
	switch s.err {
	case "":
		c.Stderr = os.Stderr
	case "&1":
		c.Stderr = c.Stdout
	default:
		if f := open(s.err, s.errAppend); f != nil {
			c.Stderr = f
		}
	}
	if len(s.env) > 0 {
		c.Env = append(os.Environ(), s.env...)
	}
	c.Dir = s.dir
	return closers, ok
}

func closeAll(fs []*os.File) {
	for _, f := range fs {
		_ = f.Close()
	}
}

// run is one shell command line, answering `$?`.
func run(c *exec.Cmd, s stdio) int {
	fs, ok := wire(c, s)
	defer closeAll(fs)
	if !ok {
		return 2
	}
	err := c.Run()
	if err != nil {
		var x *exec.ExitError
		if !errors.As(err, &x) {
			// ⚠ What a shell prints for a program it could not start, onto
			// the command's own stderr, and the 127 it answers.
			fmt.Fprintf(c.Stderr, "%s: %v\n", c.Path, err)
		}
	}
	return exitOf(err)
}

// capture is `$(command)`: its stdout with every trailing newline dropped, and
// its `$?`.
func capture(c *exec.Cmd, s stdio) (string, int) {
	var b bytes.Buffer
	c.Stdout = &b
	rc := run(c, s)
	return strings.TrimRight(b.String(), "\n"), rc
}

// startBackground is `command ... &`, answering `$!`.
//
// ⛔ THE CHILD IS REAPED WHEN IT EXITS, as a shell reaps a background job at
// its next foreground command; see the header.
func startBackground(c *exec.Cmd, s stdio) (int, error) {
	if s.in == "" {
		s.in = os.DevNull
	}
	fs, ok := wire(c, s)
	defer closeAll(fs)
	if !ok {
		return 0, errors.New("a redirection could not be opened")
	}
	// ⚠ What `&` in a non-interactive shell gives the job: SIGINT and SIGQUIT
	// ignored. An ignored disposition survives exec, so the child keeps it.
	signal.Ignore(syscall.SIGINT, syscall.SIGQUIT)
	defer signal.Reset(syscall.SIGINT, syscall.SIGQUIT)
	if err := c.Start(); err != nil {
		return 0, err
	}
	go func() { _ = c.Wait() }()
	return c.Process.Pid, nil
}

// -- the pid a `start` left, and ending it ---------------------------------

// readPID is `PID=$(cat "$WORKDIR/pid")` and its `case` guard: the answer is
// empty when the file is absent, empty or not all digits, and `stop` is then
// done rather than refused.
func readPID(file string) string {
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	pid := strings.TrimRight(string(b), "\n")
	if pid == "" || strings.Trim(pid, "0123456789") != "" {
		return ""
	}
	return pid
}

// signalPID is `kill [-SIG] PID`, answering whether it succeeded.
//
// ⛔ A PID AT OR BELOW 1 IS NEVER SIGNALLED. `kill 0` signals the caller's whole
// process group and `kill 1` signals init, and a pid file holding `0` passes
// the all-digits guard the shell halves used. Reported by the Go port on
// 2026-10-15; no `start` writes either value, so it is not a running build.
func signalPID(pid string, sig syscall.Signal) bool {
	n, err := strconv.Atoi(pid)
	if err != nil || n <= 1 {
		return false
	}
	return syscall.Kill(n, sig) == nil
}

func alive(pid string) bool { return signalPID(pid, 0) }

// waitGone is `while [ W -lt N ]; do kill -0 PID || <gone>; sleep 1; W++; done`,
// answering true as soon as the process is gone.
func waitGone(pid string, seconds int) bool {
	for i := 0; i < seconds; i++ {
		if !alive(pid) {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}

// termThenKill is the shell halves' common ending: TERM, a grace of `grace`
// seconds, then KILL. It answers once the process is gone or has been sent
// KILL; a TERM that could not be sent is a process already gone.
func termThenKill(pid string, grace int) {
	if !signalPID(pid, syscall.SIGTERM) {
		return
	}
	if waitGone(pid, grace) {
		return
	}
	signalPID(pid, syscall.SIGKILL)
}

// -- text, as the shell halves read it -------------------------------------

// shFields is awk's default field split: runs of blanks and newlines.
func shFields(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
}

// firstLine is `printf '%s\n' "$X" | head -1`.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// hostFlag is `${NAME-<unset>}`: an empty value is printed empty and an unset
// one says so.
func hostFlag(name string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return "<unset>"
}

// writeText is `{ printf ...; } >FILE`.
func writeText(path, text string) error {
	return os.WriteFile(path, []byte(text), 0o666)
}

// appendText is `printf ... >>FILE`.
func appendText(path, text string) error {
	f, err := openRedirect(path, true)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(text)
	return errors.Join(werr, f.Close())
}

// shGlob is a shell's pathname expansion of one pattern: the sorted matches,
// or the pattern itself when nothing matches. ⚠ A component whose pattern
// starts with a wildcard does not match a name starting with a dot, which is
// the shell's rule and not filepath.Glob's.
func shGlob(pattern string) []string {
	matches, _ := filepath.Glob(pattern)
	pat := strings.Split(pattern, "/")
	var out []string
	for _, m := range matches {
		parts := strings.Split(m, "/")
		hidden := false
		for i := range parts {
			if i < len(pat) && strings.HasPrefix(parts[i], ".") && pat[i] != "" &&
				strings.ContainsAny(pat[i][:1], "*?[") {
				hidden = true
			}
		}
		if !hidden {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return []string{pattern}
	}
	return out
}

// mkdirP is `mkdir -p DIR...`, run as the tool so its own diagnosis still
// reaches stderr.
func mkdirP(dirs ...string) bool {
	c := exec.Command("mkdir", append([]string{"-p"}, dirs...)...)
	return run(c, stdio{}) == 0
}

// cpFile is `cp FROM TO`, and mvFile `mv FROM TO`, run as the tools for the
// same reason and for the modes they give.
func cpFile(args ...string) bool {
	c := exec.Command("cp", args...)
	return run(c, stdio{}) == 0
}

func mvFile(from, to string) bool {
	c := exec.Command("mv", from, to)
	return run(c, stdio{}) == 0
}

// aptPrivate is the `-o` set that keeps `apt-get`'s index and cache in a
// workdir, so it runs as an unprivileged user and leaves the host's own alone.
func aptPrivate(dir string) []string {
	return []string{"-o", "Dir::State::Lists=" + dir + "/lists", "-o", "Dir::Cache=" + dir + "/cache",
		"-o", "Debug::NoLocking=1"}
}

// aptPackageRoute is the package route three adapters share: refresh the
// host's index, install, and say which refusal fired.
//
// ⚠ THE INDEX IS REFRESHED FIRST, and this is the one network call the
// package route makes that is not the install. A runner image ships a package
// list that was current when the image was built, and an install against a
// stale one fails with a 404 on a version that has moved rather than with
// anything naming the cause.
// ⛔ NEEDRESTART_MODE=l IS `list`, AND THE LETTER IS THE WHOLE POINT. Ubuntu
// 24.04 ships needrestart, which after a package install opens an interactive
// dialog listing the services to restart; DEBIAN_FRONTEND does not suppress it,
// and a dialog on a runner is a step that never returns. ⚠ `a` stops the
// dialog by RESTARTING those services instead, which on a runner means
// restarting daemons the job is standing on: measured on 2026-09-08, an install
// under `a` finished and the step after it then hung. `l` reports and touches
// nothing.
// ⚠ Every apt call reads /dev/null, so anything that still asks gets
// end-of-file rather than a wait.
func (k *adapterKit) aptPackageRoute(workdir, refusal string, packages ...string) {
	k.need(func(string) string { return "apt-get is not on this host" }, "apt-get")
	env := []string{"DEBIAN_FRONTEND=noninteractive", "NEEDRESTART_MODE=l"}
	c := exec.Command("apt-get", "update")
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "update.log"), err: "&1", env: env}) != 0 {
		k.refuse("the package index could not be refreshed")
	}
	c = exec.Command("apt-get", append([]string{"install", "-y", "--no-install-recommends"}, packages...)...)
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), err: "&1", env: env}) != 0 {
		k.refuse(refusal)
	}
}

// writePackage is the end every `install` shares.
//
// ⛔ WHAT THE ROUTE DELIVERED, WRITTEN WHERE `install-client` READS IT.
// `Build.package` is part of the identity tuple a store path is derived from,
// so a record filed without it - or with one this project guessed - files two
// packagings of one version at one name. Measured on 2026-09-09 by assembling
// capture-client run 14: nothing any adapter wrote said how the build arrived,
// and `assemble-capture` refused both lanes for it.
// ⚠ THE KEY IS `package` BECAUSE THAT IS THE RECORD'S OWN VOCABULARY, and the
// ROUTE also called `package` is the host's package-manager route. They are
// different things and an adapter is the one place both appear.
func (k *adapterKit) writePackage(workdir, format string) {
	if writeText(filepath.Join(workdir, "package"), format+"\n") != nil {
		k.refuse("the route installed and could not record how it packaged the build")
	}
}

// versionReport is the end every `version` shares.
//
// ⛔ WHAT THE BUILD SAID ABOUT ITSELF IS KEPT, NOT ONLY THE FIELD PARSED OUT OF
// IT. Two routes can install builds that report ONE version and are not one
// build: measured on 2026-09-08, Ubuntu's aria2 1.37.0 and a 1.37.0 built from
// the vendor's own release tarball answer the same version and enable
// different features. ⚠ Features are what a build DOES on the wire, so a record
// holding only the version has dropped the evidence that would have
// distinguished them.
// ⭐ stderr, because both callers already keep it: install-client writes it to
// `version.err` beside the install record and capture-client appends it to
// `adapter.err` inside the evidence bundle.
func (k *adapterKit) versionReport(output, version string) {
	fmt.Fprintf(os.Stderr, "%s reported:\n%s\n", k.me, output)
	fmt.Printf("%s\n", version)
}

// installArgs is the `install <route> <workdir>` preamble every adapter
// shares, answering the route, the workdir and the route's package slug.
func (k *adapterKit) installArgs(args []string, format func(string) (string, bool)) (string, string, string) {
	if len(args) < 2 {
		k.cannot("install takes <route> <workdir>")
	}
	route, workdir := args[0], args[1]
	if !mkdirP(workdir) {
		k.cannot("cannot create " + workdir)
	}
	f, ok := format(route)
	if !ok {
		k.cannot("this adapter does not say how the " + route + " route packages a build")
	}
	return route, workdir, f
}

// cloneCommit is the source routes' commit read. ⛔ THE COMMIT `E-ACQ-06` ASKS
// FOR, NOT THE TAG, which is a name somebody can move.
func (k *adapterKit) cloneCommit(workdir, tag string) {
	c := exec.Command("git", "-C", filepath.Join(workdir, "src"), "rev-parse", "HEAD")
	s := stdio{out: filepath.Join(workdir, "source-commit"), err: filepath.Join(workdir, "install.log"), errAppend: true}
	if run(c, s) != 0 {
		k.refuse("the source route cloned " + tag + " and could not name its commit")
	}
}
