// install-client - install one target through one route, on a host that has
// already been claimed and still has its way out.
//
// CI-11. It was scripts/acquisition/install-client.sh, and the reasons it gave
// are kept here with the rules they explain.
//
// ⛔ THIS IS THE ONE STEP THAT RUNS WITH THE NETWORK STILL IN REACH, AND THE
// ORDER IS THE CONTAINMENT. `capture-client` refuses to install anything, so
// everything a run needs is fetched here and nothing is fetched after. ⚠ It
// deliberately does not check the egress guard: being able to reach the network
// is this step's precondition. The CLAIM is checked: a host that already ran a
// capture must not have a second product installed on it.
//
// ⛔ TWO ROUTES ARE TWO INVOCATIONS. One route per call, so "two independent
// routes" stays a property of the workflow that proves it, and `E-ACQ-07` and
// `E-ACQ-08` refuse a pair sharing a resolver or a delivery mechanism.
//
// ⭐ IT PRINTS WHAT THE BUILD SAYS AFTERWARDS, which is the only version this
// project believes, and it asks the host what it already had BEFORE the route
// runs, so the record says whether this route acquired anything. `aria2` ships
// on `ubuntu-24.04`, so `apt-get install aria2` there installs nothing and
// exits 0; client capture runs 3 and 4 recorded `route=package` over exactly
// that, and two routes that each acquired nothing hand `ACQ-03` one binary to
// compare against itself, which reads as `byte_identical`.
//
// -- ⛔ EVERY ADAPTER CALL IS BOUNDED, AND HANDED FILES -----------------------
//
// An adapter shells out to a product this project did not write, and a product
// that waits on a prompt waits forever: measured on 2026-09-08, two of three
// jobs of the first client dispatch sat in this step for half an hour and
// reported nothing. ⛔ And nothing the adapter spawns may hold a pipe this
// process reads: a command substitution ended when its pipe did, so measured on
// 2026-09-16 an adapter whose `version` left a `sleep` behind still blocked at
// 25 seconds under a 4-second bound. `adapterRun` is the only way to an adapter
// from here; it bounds the call, gives the adapter stdin from /dev/null so a
// prompt gets end of file, and writes its output to FILES.
//
// ⛔ A TIMEOUT IS 124, coreutils' verdict for "it never answered", and a
// different fact from a route that refused: TERM at the bound, KILL after
// BIT_IDS_KILL_AFTER, which `-k` was there for because a child that ignores
// TERM is otherwise never killed at all.
//
// -- ⚠ WHERE THIS DIFFERS FROM THE SHELL HALF -------------------------------
//
//   - The adapter is still `sh <adapter> <subcommand>` until the adapters port,
//     and the claim marker is still asked of the host guard; on Windows that is
//     `assert-disposable.ps1 -Marker`.
//   - Digests and sizes are Go's; `platform=` is still `uname -srm` where a
//     `uname` exists, and GOOS and GOARCH where none does.
//
// Usage:
//
//	bit-check install-client --adapter <path> --route <name> --workdir <dir> [--record <file>]
//
// Exit codes: 0 installed and the build reported a version, 1 a guard or the
// route refuses, 2 could not run.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const installClientUsage = `usage: bit-check install-client --adapter <path> --route <package|release|source>
                               --workdir <dir> [--record <file>]

Exit codes: 0 installed and the build reported a version, 1 a guard or the
route refuses, 2 could not run.
`

// boundedRun runs prog with stdin from the null device and stdout and stderr
// in the files given (inheriting this process's own where a name is empty),
// TERM at bound seconds and KILL grace seconds later. It answers the exit code
// `timeout -k grace bound` would have: the command's own, 124 when the bound
// fired and TERM ended it, and 137 when KILL had to.
func boundedRun(stdout, stderr string, bound, grace int, prog string, args ...string) int {
	open := func(name string, std *os.File) (*os.File, func(), error) {
		if name == "" {
			return std, func() {}, nil
		}
		f, err := os.Create(name)
		if err != nil {
			return nil, nil, err
		}
		return f, func() { f.Close() }, nil
	}
	out, closeOut, err := open(stdout, os.Stdout)
	if err != nil {
		return 125
	}
	defer closeOut()
	errf, closeErr, err := open(stderr, os.Stderr)
	if err != nil {
		return 125
	}
	defer closeErr()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(bound)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Stdout, cmd.Stderr = out, errf
	cmd.Cancel = func() error { return terminate(cmd.Process) }
	cmd.WaitDelay = time.Duration(grace) * time.Second
	err = cmd.Run()
	timedOut := ctx.Err() == context.DeadlineExceeded
	if err == nil {
		if timedOut {
			return 124
		}
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			if timedOut {
				if ws.Signal() == syscall.SIGKILL {
					return 137
				}
				return 124
			}
			return 128 + int(ws.Signal())
		}
		if timedOut {
			return 124
		}
		return ee.ExitCode()
	}
	if timedOut {
		return 124
	}
	return 126
}

// readTrimmed is `$(cat FILE)`: the content with every trailing newline
// removed, as a command substitution strips them.
func readTrimmed(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(b), "\n")
}

// stripAll is `tr -d ' \n\r'`.
func stripAll(s string) string {
	return strings.NewReplacer(" ", "", "\n", "", "\r", "").Replace(s)
}

var packageSlug = regexp.MustCompile(`^[a-z0-9-]+$`)

func installClient(args []string) int {
	err := runInstallClient(args)
	if err == nil {
		return 0
	}
	if e, ok := err.(resolveExit); ok {
		if e.code == -1 {
			fmt.Print(installClientUsage)
			return 0
		}
		fmt.Fprintf(os.Stderr, "install-client: %s\n", e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "install-client: %v\n", err)
	return 2
}

// hostMarker asks the host guard where its claim marker lives. ⛔ ASKED, NEVER
// COMPOSED: a caller that spelled the state directory itself would go on
// reading the old place the day it moved, and report an unclaimed host fresh.
func hostMarker(root string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		guard := filepath.Join(root, "scripts", "acquisition", "assert-disposable.ps1")
		if !isRegular(guard) {
			return "", resolveExit{2, guard + " is not present"}
		}
		cmd = exec.Command("pwsh", "-NoProfile", "-File", guard, "-Marker")
	} else {
		// ⭐ In process: the guard is this binary, and `claimMarker` is the
		// derivation its `--marker` prints.
		_, m, err := claimMarker()
		if err != nil {
			return "", resolveExit{2, "the guard could not report its marker path"}
		}
		return m, nil
	}
	out, err := cmd.Output()
	if err != nil {
		return "", resolveExit{2, "the guard could not report its marker path"}
	}
	m := strings.TrimRight(string(out), "\r\n")
	if m == "" {
		return "", resolveExit{2, "the guard reported an empty marker path"}
	}
	return m, nil
}

func hostPlatform() string {
	if out, err := exec.Command("uname", "-srm").Output(); err == nil {
		return strings.TrimRight(string(out), "\n")
	}
	return runtime.GOOS + " " + runtime.GOARCH
}

func runInstallClient(args []string) error {
	var adapter, route, workdir, record string
	help, err := parseResolveArgs("install-client", args, map[string]*string{
		"--adapter": &adapter, "--route": &route, "--workdir": &workdir, "--record": &record,
	})
	if help {
		return resolveExit{-1, ""}
	}
	if err != nil {
		return err
	}
	cannot := func(f string, x ...any) error { return resolveExit{2, fmt.Sprintf(f, x...)} }
	refuse := func(f string, x ...any) error { return resolveExit{1, fmt.Sprintf(f, x...)} }
	switch {
	case adapter == "":
		return cannot("--adapter is required")
	case route == "":
		return cannot("--route is required")
	case workdir == "":
		return cannot("--workdir is required")
	case !isRegular(adapter):
		return cannot("the adapter %s is not present", adapter)
	}

	installSeconds := envSeconds("BIT_IDS_INSTALL_TIMEOUT", 420)
	versionSeconds := envSeconds("BIT_IDS_VERSION_TIMEOUT", 60)
	describeSeconds := envSeconds("BIT_IDS_DESCRIBE_TIMEOUT", 30)
	killAfter := envSeconds("BIT_IDS_KILL_AFTER", 20)

	// ⛔ THE ROUTE VOCABULARY IS CLOSED: a spelling that is not `RouteKind`'s
	// would install through something no record can describe.
	switch route {
	case "package", "release", "source":
	default:
		return cannot("unknown route: %s (package, release or source)", route)
	}

	root, err := captureRoot(filepath.Dir(adapter))
	if err != nil {
		return err
	}
	marker, err := hostMarker(root)
	if err != nil {
		return err
	}
	if !isRegular(marker) {
		return refuse("the host was never claimed; %s does not exist", marker)
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return cannot("cannot create %s", workdir)
	}
	if workdir, err = filepath.Abs(workdir); err != nil {
		return cannot("cannot create %s", workdir)
	}
	wd := func(name string) string { return filepath.Join(workdir, name) }

	adapterRun := func(bound int, out, errf string, sub ...string) int {
		return boundedRun(out, errf, bound, killAfter, "sh", append([]string{adapter}, sub...)...)
	}

	if rc := adapterRun(describeSeconds, wd("describe.log"), wd("describe.err"), "describe"); rc != 0 {
		return cannot("the adapter would not describe itself (exit %d)", rc)
	}
	desc, _ := os.ReadFile(wd("describe.log"))
	target := keyField(desc, "target")
	if target == "" {
		return cannot("the adapter named no target")
	}
	started := utcNow()

	// ⛔ WHAT THE HOST ALREADY HAD, ASKED BEFORE THE ROUTE RUNS. ⚠ Its stderr
	// is kept and never printed as a diagnosis: a target not yet installed is the
	// ORDINARY case here.
	preRC := adapterRun(versionSeconds, wd("version-before.log"), wd("version-before.err"), "version")
	preexisting := readTrimmed(wd("version-before.log"))
	if preRC != 0 {
		preexisting = ""
	}

	// ⛔ AND THE EXECUTABLE ITSELF, BECAUSE A VERSION IS NOT AN IDENTITY: the
	// release route installs a different aria2 1.37.0 than the image's, and a
	// comparison of version strings alone calls that no acquisition. ⚠ A
	// `describe` that fails here is not fatal: the digest refines the version
	// comparison, and an adapter that names no executable leaves it empty.
	adapterBinary := func(tag string) string {
		if adapterRun(describeSeconds, wd("describe-"+tag+".log"), wd("describe-"+tag+".err"), "describe") != 0 {
			return ""
		}
		b, _ := os.ReadFile(wd("describe-" + tag + ".log"))
		return keyField(b, "binary")
	}
	digestOf := func(path string) string {
		if path == "" || !isRegular(path) {
			return ""
		}
		d, err := fileSHA256(path)
		if err != nil {
			return ""
		}
		return d
	}
	preBinary := adapterBinary("before")
	preDigest := digestOf(preBinary)

	// ⚠ The install's own output is this process's, which `install-step` sends
	// to a file one level up.
	installRC := boundedRun("", "", installSeconds, killAfter, "sh", adapter, "install", route, workdir)

	// ⚠ THE LOGS THE ROUTE WROTE ARE PRINTED ON EVERY FAILURE, INDENTED, or the
	// job log carries the refusal and not its cause.
	showRouteLogs := func() {
		for _, log := range []string{wd("update.log"), wd("install.log")} {
			if !isRegular(log) {
				continue
			}
			fmt.Fprintf(os.Stderr, "          -- %s (last 20 lines)\n", log)
			b, _ := os.ReadFile(log)
			lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
			if len(b) == 0 {
				lines = nil
			}
			if len(lines) > 20 {
				lines = lines[len(lines)-20:]
			}
			for _, l := range lines {
				fmt.Fprintf(os.Stderr, "          %s\n", l)
			}
		}
	}
	switch installRC {
	case 0:
	case 124:
		showRouteLogs()
		return refuse("the %s route did not answer within %ds; it is hung, not slow", route, installSeconds)
	case 1:
		showRouteLogs()
		return refuse("the %s route did not install %s", route, target)
	default:
		showRouteLogs()
		return cannot("the %s route could not run (exit %d)", route, installRC)
	}

	// ⛔ AND THE BUILD IS ASKED, HERE, while there is still something to do
	// about a failure. ⛔ The adapter's own message is kept: silencing the
	// diagnosis and then reporting its absence is one defect, not two.
	versionRC := adapterRun(versionSeconds, wd("version.log"), wd("version.err"), "version")
	version := readTrimmed(wd("version.log"))
	showAdapterError := func() {
		if fi, err := os.Stat(wd("version.err")); err != nil || fi.Size() == 0 {
			return
		}
		fmt.Fprintln(os.Stderr, "          -- the adapter said")
		indentFile(wd("version.err"))
	}
	switch versionRC {
	case 0:
	case 124:
		showAdapterError()
		return refuse("%s installed through the %s route and did not answer --version within %ds", target, route, versionSeconds)
	default:
		showAdapterError()
		return refuse("%s installed through the %s route but would not report a version (adapter exit %d)", target, route, versionRC)
	}
	if version == "" {
		return refuse("%s installed through the %s route and reported an empty version", target, route)
	}

	postBinary := adapterBinary("after")
	postDigest := digestOf(postBinary)
	acquired := acquiredVerdict(preexisting, version, preDigest, postDigest)
	if acquired != "yes" {
		fmt.Fprintf(os.Stderr, "install-client: %s %s was ALREADY on this host and the %s route installed nothing; this run acquired no build\n",
			target, version, route)
	}
	finished := utcNow()

	sourceCommit := ""
	if b, err := os.ReadFile(wd("source-commit")); err == nil && isRegular(wd("source-commit")) {
		sourceCommit = stripAll(string(b))
	}
	// ⛔ HOW THE ROUTE PACKAGED WHAT IT INSTALLED: `Build.package` is part of the
	// identity tuple a store path is derived from, and a guess files two
	// packagings at one name. A slug, checked here so the message names the
	// adapter rather than a schema a dispatch later.
	pkg := ""
	if b, err := os.ReadFile(wd("package")); err == nil && isRegular(wd("package")) {
		pkg = stripAll(string(b))
	}
	switch {
	case pkg == "":
		return refuse("the %s route installed and recorded no package format", route)
	case !packageSlug.MatchString(pkg) || strings.HasPrefix(pkg, "-") || strings.HasSuffix(pkg, "-"):
		return refuse("the %s route recorded package [%s], which is not a slug", route, pkg)
	}
	if route == "source" {
		// ⛔ Forty lowercase hex digits, checked as a shape: `E-ACQ-06` exists
		// because an abbreviated commit is not an immutable reference.
		if sourceCommit == "" || strings.Trim(sourceCommit, "0123456789abcdef") != "" {
			return refuse("the source route recorded no full commit object name")
		}
		if len(sourceCommit) != 40 {
			return refuse("the source route recorded [%s], which is not a full object name", sourceCommit)
		}
	}

	// -- rule 12, on the OTHER path that ships an artifact ------------------------
	//
	// ⛔ The install workdir's top level is uploaded as a second artifact. The
	// scan is this binary's own, in this process.
	switch rc := scanBundle([]string{"--top-level", workdir}); rc {
	case 0:
	case 1:
		return refuse("the install would ship what rule 12 forbids, named above")
	default:
		return cannot("the rule-12 scan could not run (exit %d)", rc)
	}

	if record != "" {
		size := ""
		if fi, err := os.Stat(postBinary); err == nil && postBinary != "" && fi.Mode().IsRegular() {
			size = fmt.Sprint(fi.Size())
		}
		asum, _ := fileSHA256(adapter)
		var b strings.Builder
		b.WriteString("bit-ids/install-record/1\n")
		for _, kv := range [][2]string{
			{"target", target}, {"route", route}, {"reported_version", version},
			{"preexisting_version", preexisting}, {"preexisting_binary", preBinary},
			{"preexisting_binary_sha256", preDigest}, {"installed_binary", postBinary},
			{"installed_binary_sha256", postDigest}, {"installed_binary_bytes", size},
			{"retrieved_at", finished}, {"signature", "not_checked"}, {"acquired", acquired},
			{"source_commit", sourceCommit}, {"package", pkg}, {"adapter", adapter},
			{"adapter_sha256", asum}, {"workdir", workdir}, {"started_at", started},
			{"finished_at", finished}, {"platform", hostPlatform()},
		} {
			fmt.Fprintf(&b, "%s=%s\n", kv[0], kv[1])
		}
		if err := os.WriteFile(record, []byte(b.String()), 0o644); err != nil {
			return cannot("the install record could not be written to %s", record)
		}
		if err := recordHasAs(record, "the install record was written without ", "target", "route",
			"reported_version", "preexisting_version", "preexisting_binary", "preexisting_binary_sha256",
			"installed_binary", "installed_binary_sha256", "acquired", "source_commit", "package"); err != nil {
			return err
		}
		// ⛔ AND THE RECORD IS CHECKED AGAINST ITSELF RATHER THAN RESTATED: the
		// verdict is re-derived from the fields read back off disk, compared
		// with `==`, never as a pattern, because every `.` in `1.37.0` matches
		// any byte.
		body, _ := os.ReadFile(record)
		pre := recordField(body, "preexisting_version")
		rep := recordField(body, "reported_version")
		acq := recordField(body, "acquired")
		if rep != version {
			return refuse("the install record says reported_version=[%s] over a build that said [%s]", rep, version)
		}
		if want := acquiredVerdict(pre, rep, recordField(body, "preexisting_binary_sha256"),
			recordField(body, "installed_binary_sha256")); acq != want {
			return refuse("the install record says acquired=[%s] over preexisting [%s] and reported [%s]", acq, pre, rep)
		}
	}

	fmt.Println(version)
	fmt.Fprintf(os.Stderr, "install-client: %s %s through the %s route (acquired=%s)\n", target, version, route, acquired)
	return nil
}

// acquiredVerdict is whether a route acquired anything. Absent before and
// answering now is an install; a different version is an upgrade; the same
// version from a different executable is the branch a version comparison
// cannot reach, and the one the aria2 pair needs. Anything else acquired
// nothing, whatever the route reported.
func acquiredVerdict(pre, now, preDigest, postDigest string) string {
	switch {
	case pre == "":
		return "yes"
	case pre != now:
		return "yes"
	case preDigest != "" && postDigest != "" && preDigest != postDigest:
		return "yes"
	}
	return "no"
}

// recordHasAs is recordHas with the caller's own words for a missing field.
func recordHasAs(path, prefix string, keys ...string) error {
	if err := recordHas(path, keys...); err != nil {
		msg := err.Error()
		return resolveExit{2, strings.Replace(msg, "the resolution record was written without ", prefix, 1)}
	}
	return nil
}
