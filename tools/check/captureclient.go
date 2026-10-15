// capture-client - a capture of an INSTALLED BUILD, on a host that has already
// been claimed and whose capture is already contained.
//
// CI-11. It was scripts/capture/capture-client.sh, and the reasons it gave are
// kept here beside the rules they explain.
//
// ⛔ THE DIFFERENCE FROM capture-run IS THE DRIVER, AND IT IS THE WHOLE
// PRODUCT. capture-run drives this project's observer with `curl` and writes
// `kind=fixture`. Here an ADAPTER stands for one target: it answers what the
// installed build's exact version is, and it starts that build on the torrent
// the observer generated. ⛔ It installs nothing and builds nothing; that
// happened in an earlier step, through `install-client`, while the network was
// in reach.
//
// -- ⛔ WHAT MAKES THIS A MEASUREMENT OF A BUILD -----------------------------
//
//  1. The adapter answers `version` from the INSTALLED EXECUTABLE, which is the
//     rule `E-ACQ-10` states for a record.
//  2. An announce arrived carrying a peer ID THIS OBSERVER DID NOT GENERATE, and
//     the raw transcript on disk carries those exact bytes. Everything else is
//     satisfied by empty artifacts verifying against their own empty digests.
//  3. A SECOND CONNECTOR READ THE SAME BYTES AND SAID WHAT IT SAW. Absolute 2
//     asks for one, and `E-CAP-01` refuses a record declaring none as an
//     INVALID DOCUMENT: that is what refused run 14's four artifacts.
//
// ⚠ The adapter DECLARES what it is and the attestation copies that: a stub
// adapter yields `stock_client=false`.
//
// -- ⚠ WHERE THIS DIFFERS FROM THE SHELL HALF -------------------------------
//
//   - The adapter is still `sh <adapter> <subcommand>`, bounded as `timeout -k`
//     bounded it, until the adapters port. The connector is still Python, which
//     is the point of it: its independence from this project's Rust.
//   - Digests are Go's. The bundle is still verified by a host tool, which is
//     `sha256sum -c` or, on Windows, `certutil`, and `evidence_verified_by`
//     names which one did.
//
// Usage:
//
//	bit-check capture-client --run-id <id> --out <dir> --observer <path>
//	                         --adapter <path> --connector <path>
//	                         [--seconds <n>] [--sessions <n>] [--peer-port <n>]
//	                         [--route-table <path>]
//
// Exit codes: 0 the capture ran and its evidence verifies, 1 a guard refuses,
// 2 could not run.
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const captureClientUsage = `usage: bit-check capture-client --run-id <id> --out <dir> --observer <path>
                               --adapter <path> --connector <path>
                               [--seconds <n>] [--sessions <n>] [--peer-port <n>]
                               [--route-table <path>]

Exit codes: 0 the capture ran and its evidence verifies, 1 a guard refuses,
2 could not run.
`

// boundedCall is boundedRun with each output either created or APPENDED to,
// as `>` and `>>` chose in the shell half, and a name of "-" for the null
// device.
func boundedCall(stdout string, appendOut bool, stderr string, appendErr bool,
	bound, grace int, prog string, args ...string) int {
	open := func(name string, add bool) (*os.File, error) {
		if name == "-" {
			return os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		}
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if add {
			flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		}
		return os.OpenFile(name, flags, 0o644)
	}
	out, err := open(stdout, appendOut)
	if err != nil {
		return 125
	}
	defer out.Close()
	var errf *os.File
	if stderr == stdout && appendErr == appendOut {
		errf = out
	} else {
		if errf, err = open(stderr, appendErr); err != nil {
			return 125
		}
		defer errf.Close()
	}
	return runBoundedFiles(out, errf, bound, grace, prog, args...)
}

// runBoundedFiles is the bound itself, over files already open.
func runBoundedFiles(out, errf *os.File, bound, grace int, prog string, args ...string) int {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(bound)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, prog, args...)
	null, _ := os.Open(os.DevNull)
	if null != nil {
		defer null.Close()
		cmd.Stdin = null
	}
	cmd.Stdout, cmd.Stderr = out, errf
	cmd.Cancel = func() error { return terminate(cmd.Process) }
	cmd.WaitDelay = time.Duration(grace) * time.Second
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 124
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if c := ee.ExitCode(); c >= 0 {
				return c
			}
			return signalCode(ee)
		}
		return 126
	}
	return 0
}

// The observer's log, read the way the shell half's `awk` programs read it.
type observerLog struct{ lines [][]string }

func readObserverLog(path string) observerLog {
	var l observerLog
	b, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(b), "\n") {
		l.lines = append(l.lines, strings.Fields(line))
	}
	return l
}

// first is `awk '$1 == KEY { print $N; exit }'`.
func (l observerLog) first(key string, n int) string {
	for _, f := range l.lines {
		if len(f) > 0 && f[0] == key {
			if n < len(f) {
				return f[n]
			}
			return ""
		}
	}
	return ""
}

// rest is `awk '$1 == KEY { $1 = ""; sub(/^ /, ""); print; exit }'`: every
// field after the first, joined by one space as awk rebuilds a record.
func (l observerLog) rest(key string) string {
	for _, f := range l.lines {
		if len(f) > 0 && f[0] == key {
			return strings.Join(f[1:], " ")
		}
	}
	return ""
}

func wholeNumber(s string) bool { return s != "" && strings.Trim(s, "0123456789") == "" }

// pythonInterpreter is the connector's interpreter. ⚠ `python3` where it is,
// and `python` on a Windows image, which names the same interpreter so.
func pythonInterpreter() string {
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("python"); err == nil {
			return p
		}
	}
	return ""
}

// verifyBundleSums runs the host's verifier over OUT/SHA256SUMS, writing what
// it said to the log, and answers whether it agreed and which reader it was.
func verifyBundleSums(out, log string) (bool, string) {
	reader := sumsReader()
	switch reader {
	case "sha256sum":
		cmd := exec.Command("sha256sum", "-c", "SHA256SUMS")
		cmd.Dir = out
		b, err := cmd.CombinedOutput()
		os.WriteFile(log, b, 0o644)
		return err == nil, reader
	case "certutil":
		ok := verifySums(out)
		if ok {
			os.WriteFile(log, []byte("every row verified by certutil\n"), 0o644)
		} else {
			os.WriteFile(log, []byte("certutil disagreed with a row of SHA256SUMS\n"), 0o644)
		}
		return ok, reader
	}
	os.WriteFile(log, []byte("no verifier on this host\n"), 0o644)
	return false, ""
}

func captureClient(args []string) int {
	err := runCaptureClient(args)
	if err == nil {
		return 0
	}
	if e, ok := err.(resolveExit); ok {
		if e.code == -1 {
			fmt.Print(captureClientUsage)
			return 0
		}
		fmt.Fprintf(os.Stderr, "capture-client: %s\n", e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "capture-client: %v\n", err)
	return 2
}

func runCaptureClient(args []string) error {
	var runID, out, observer, adapter, connector, peerPort, routeTable string
	seconds, sessions := "25", "3"
	help, err := parseResolveArgs("capture-client", args, map[string]*string{
		"--run-id": &runID, "--out": &out, "--observer": &observer, "--adapter": &adapter,
		"--connector": &connector, "--seconds": &seconds, "--sessions": &sessions,
		"--peer-port": &peerPort, "--route-table": &routeTable,
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
	case runID == "":
		return cannot("--run-id is required")
	case out == "":
		return cannot("--out is required")
	case observer == "":
		return cannot("--observer is required")
	case adapter == "":
		return cannot("--adapter is required")
	case connector == "":
		// ⛔ REQUIRED, WITH NO DEFAULT AND NO FALLBACK: a capture that ran with
		// one connector is evidence that cannot become a record.
		return cannot("--connector is required; a one-connector capture cannot become a record")
	}
	if strings.Trim(runID, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		return cannot("run id must be lowercase a-z0-9-: %s", runID)
	}
	if !wholeNumber(seconds) {
		return cannot("--seconds must be a whole number: %s", seconds)
	}
	if !wholeNumber(sessions) {
		return cannot("--sessions must be a whole number: %s", sessions)
	}
	secondsN, _ := strconv.Atoi(seconds)
	sessionsN, _ := strconv.Atoi(sessions)
	// ⚠ At least one: zero sessions is a capture that starts nothing.
	if sessionsN < 1 {
		return cannot("--sessions must be at least 1: %s", sessions)
	}
	if peerPort != "" && !wholeNumber(peerPort) {
		return cannot("--peer-port must be a whole number: %s", peerPort)
	}
	python := pythonInterpreter()
	if python == "" {
		return cannot("python3 not found; the second connector is deliberately not this project's Rust")
	}

	// ⛔ EVERY ADAPTER CALL IS BOUNDED, and stdin is the null device on every
	// one. ⚠ Under containment a hang is worse than during an install: the
	// observer's own deadline expires meanwhile and the run reports "the build
	// announced nothing" over a build that was never asked anything.
	adapterSeconds := envSeconds("BIT_IDS_ADAPTER_TIMEOUT", 120)
	killAfter := envSeconds("BIT_IDS_KILL_AFTER", 20)

	if fi, err := os.Stat(observer); err != nil || fi.IsDir() || (runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0) {
		return cannot("the observer %s is not executable; it is built before the route is deleted, never here", observer)
	}
	if !isRegular(adapter) {
		return cannot("the adapter %s is not present", adapter)
	}
	if !isRegular(connector) {
		return cannot("the connector %s is not present", connector)
	}
	// ⛔ A SECOND CAPTURE MUST NOT WRITE OVER THE FIRST ONE'S EVIDENCE.
	if _, err := os.Stat(out); err == nil {
		return refuse("%s already exists; a capture never writes into another run's evidence", out)
	}

	// -- the host, as this run finds it -------------------------------------
	//
	// ⛔ THE MARKER'S PATH IS ASKED FOR, NEVER COMPOSED, and the guard is this
	// binary, so it is asked in process.
	_, marker, err := claimMarker()
	if err != nil || marker == "" {
		return cannot("the guard could not report its marker path")
	}
	if !isRegular(marker) {
		return refuse("the host was never claimed; %s does not exist", marker)
	}
	mb, _ := os.ReadFile(marker)
	claimedRun := ""
	for _, line := range strings.Split(string(mb), "\n") {
		parts := strings.Split(line, "=")
		if len(parts) >= 2 && parts[0] == "run" {
			claimedRun = parts[1]
			break
		}
	}
	if claimedRun != runID {
		return refuse("the host is claimed by run [%s], not [%s]", claimedRun, runID)
	}

	// ⛔ AND THE ROUTE IS READ AGAIN AT THE MOMENT OF CAPTURE: nothing between
	// the workflow's own step and this one put it back.
	egressArgs := []string{"--egress"}
	egressSource := "/proc/net/route"
	if routeTable != "" {
		egressArgs = append(egressArgs, routeTable)
		egressSource = routeTable
	}
	switch rc := assertDisposable(egressArgs); rc {
	case 0:
	case 1:
		return refuse("egress is open; the default route is still there and a capture would leak")
	default:
		return cannot("egress could not be established from %s", egressSource)
	}
	fingerprint := hostFingerprint()
	if fingerprint == "" {
		return cannot("the host fingerprint could not be read")
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		return cannot("cannot create %s", out)
	}
	if out, err = filepath.Abs(out); err != nil {
		return cannot("cannot create %s", out)
	}
	at := func(name string) string { return filepath.Join(out, name) }
	// ⭐ THE CLAIM GOES WITH THE EVIDENCE; `assemble-capture` reads the host
	// facts a run manifest needs from it. `CI-09`.
	if err := os.WriteFile(at("host-claim.txt"), mb, 0o644); err != nil {
		return cannot("cannot keep the host claim beside the attestation")
	}
	bundle := at("bundle")
	logPath := at("observer.log")
	sums := at("SHA256SUMS")
	attest := at("attestation.txt")
	workdir := at("client")
	torrent := at("run.torrent")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return cannot("cannot create %s", workdir)
	}

	adapterCall := func(stdout string, appendOut bool, stderr string, appendErr bool, sub ...string) int {
		return boundedCall(stdout, appendOut, stderr, appendErr, adapterSeconds, killAfter,
			"sh", append([]string{adapter}, sub...)...)
	}

	// -- what the adapter says it is ---------------------------------------
	//
	// ⛔ ASKED BEFORE ANYTHING IS STARTED, so a run whose adapter cannot
	// describe itself never launches a process at all.
	if adapterCall(at("adapter.describe"), false, at("adapter.err"), false, "describe") != 0 {
		return cannot("the adapter could not describe itself")
	}
	desc, _ := os.ReadFile(at("adapter.describe"))
	target := keyField(desc, "target")
	kind := keyField(desc, "kind")
	if target == "" {
		return cannot("the adapter named no target")
	}
	switch kind {
	case "stock", "stub":
	case "":
		return cannot("the adapter declared no kind")
	default:
		return cannot("the adapter declared an unknown kind: %s", kind)
	}

	// ⛔ THE VERSION COMES OUT OF THE INSTALLED EXECUTABLE, into a FILE, never
	// through a substitution that waits on a pipe the product may hold.
	versionRC := adapterCall(at("adapter.version"), false, at("adapter.err"), true, "version")
	measured := readTrimmed(at("adapter.version"))
	if versionRC == 124 {
		return cannot("the installed build did not answer --version within %ds", adapterSeconds)
	}
	if versionRC != 0 {
		return cannot("the adapter could not ask the installed build its version")
	}
	if measured == "" {
		return cannot("the adapter reported an empty version for the installed build")
	}
	for _, r := range measured {
		if r < ' ' || r > '~' {
			return cannot("the reported version is not printable text: %s", measured)
		}
	}
	started := utcNow()

	// -- the run --------------------------------------------------------------
	//
	// ⚠ The observer prints its endpoints, writes the torrent, and then serves
	// until its own deadline. Waiting on its line rather than on a delay is
	// what makes this work on a slow runner.
	obsArgs := []string{bundle, seconds, torrent}
	if peerPort != "" {
		obsArgs = append(obsArgs, peerPort)
	}
	obsLog, err := os.Create(logPath)
	if err != nil {
		return cannot("cannot write %s", logPath)
	}
	obs := exec.Command(observer, obsArgs...)
	obs.Stdout, obs.Stderr = obsLog, obsLog
	if err := obs.Start(); err != nil {
		obsLog.Close()
		return cannot("the observer could not be started: %v", err)
	}
	obsLog.Close()
	obsDone := make(chan int, 1)
	go func() { obsDone <- waitCode(obs) }()
	// ⚠ Read once, here: every deadline below is measured from the moment the
	// observer began its own.
	epoch := time.Now()
	waitObserver := func() int {
		rc := <-obsDone
		obsDone <- rc
		return rc
	}
	exited := func() bool {
		select {
		case rc := <-obsDone:
			obsDone <- rc
			return true
		default:
			return false
		}
	}
	for waited := 0; waited < 60; waited++ {
		if lb, _ := os.ReadFile(logPath); bytes.Contains(append([]byte("\n"), lb...), []byte("\nserving for ")) {
			break
		}
		if exited() {
			break
		}
		time.Sleep(time.Second)
	}
	announceURL := readObserverLog(logPath).first("announce-url", 1)
	if announceURL == "" {
		waitObserver()
		indentFile(logPath)
		return refuse("the observer never reported an announce URL")
	}
	// ⛔ THE TORRENT MUST EXIST BEFORE THE CLIENT IS STARTED.
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		waitObserver()
		return refuse("the observer wrote no torrent at %s", torrent)
	}

	// -- ⛔ THE SESSIONS, THE ONLY DIMENSION THAT SEPARATES TWO LIFETIMES -------
	//
	// A value the build stores and one it regenerates read identically in one
	// session; run 20's `aria2-next` announced twice with one peer ID. ⛔ The
	// stop is inside the observer's window, so a `stopped` announce reaches a
	// tracker that is still listening. ⚠ The margin is a fifth of the run,
	// capped at five seconds and floored at one.
	stopMargin := secondsN / 5
	if stopMargin > 5 {
		stopMargin = 5
	}
	if stopMargin < 1 {
		stopMargin = 1
	}
	window := secondsN - stopMargin
	clientLog := at("client.log")
	port := peerPort
	if port == "" {
		port = "0"
	}
	started_, stopRC, stoppedAfter := 0, 0, 0
	for session := 1; session <= sessionsN; session++ {
		startRC := adapterCall(clientLog, true, clientLog, true, "start", torrent, workdir, port)
		if startRC != 0 {
			// ⛔ THE FIRST SESSION REFUSES AND A LATER ONE ENDS THE LOOP: a build
			// measured once is weaker than was asked and better than nothing.
			if started_ > 0 {
				fmt.Fprintf(os.Stderr, "capture-client: session %d would not start (exit %d); %d of %d ran\n",
					session, startRC, started_, sessionsN)
				break
			}
			adapterCall("-", false, "-", false, "stop", workdir)
			waitObserver()
			indentFile(clientLog)
			if startRC == 124 {
				return refuse("the adapter did not return from start within %ds; it launches and returns", adapterSeconds)
			}
			return refuse("the adapter could not start the build (exit %d)", startRC)
		}
		started_++
		// ⛔ EACH SESSION'S BOUNDARY IS MEASURED FROM THE OBSERVER'S OWN START,
		// never from here, or a slow start drifts past an expired deadline.
		endsAt := window * session / sessionsN
		elapsed := int(time.Since(epoch) / time.Second)
		if stopIn := endsAt - elapsed; stopIn > 0 {
			time.Sleep(time.Duration(stopIn) * time.Second)
		}
		// ⛔ READ BEFORE THE STOP IS CALLED, NEVER AFTER IT RETURNS: the question
		// is whether the observer was still recording when the build was ASKED.
		stoppedAfter = int(time.Since(epoch) / time.Second)
		stopRC = adapterCall(clientLog, true, clientLog, true, "stop", workdir)
	}
	stoppedWithin := "no"
	if stoppedAfter < secondsN {
		stoppedWithin = "yes"
	}
	observerRC := waitObserver()
	// ⚠ STOPPED WHATEVER HAPPENED ABOVE: a second stop is how one interrupted
	// by its own time limit still leaves nothing running.
	if stopRC != 0 {
		adapterCall(clientLog, true, clientLog, true, "stop", workdir)
	}
	if observerRC != 0 {
		indentFile(logPath)
		return refuse("the observer exited %d", observerRC)
	}

	log := readObserverLog(logPath)
	segments := log.first("segments:", 1)
	if segments == "" {
		return refuse("the observer reported no segment count")
	}
	if n, err := strconv.Atoi(segments); err != nil || n <= 0 {
		return refuse("the run recorded no bytes at all")
	}
	announces := log.first("announces", 1)
	if announces == "" {
		return refuse("the observer reported no announce count")
	}
	if n, err := strconv.Atoi(announces); err != nil || n <= 0 {
		return refuse("the build announced nothing; there is no measurement here")
	}
	// ⛔ AND THE INTERVAL MUST NOT OUTLIVE THE RUN: run 18 answered 60 under a
	// 45-second deadline, so no re-announce was ever due. Both values are the
	// observer's own, never recomputed here.
	servingFor := ""
	for _, f := range log.lines {
		if len(f) >= 3 && f[0] == "serving" && f[1] == "for" {
			servingFor = strings.TrimSuffix(f[2], "s")
			break
		}
	}
	if !wholeNumber(servingFor) {
		return cannot("the observer did not report how long it served")
	}
	if servingFor != seconds {
		return cannot("the observer served for %ss, not the %ss it was given", servingFor, seconds)
	}
	offered := log.first("offered-interval", 1)
	if !wholeNumber(offered) {
		return cannot("the observer reported no announce interval")
	}
	if o, _ := strconv.Atoi(offered); o > secondsN {
		return cannot("the tracker answered interval %ss under a %ss deadline; no re-announce can fall due", offered, servingFor)
	}

	// -- the evidence, checked by something else -----------------------------
	var rows strings.Builder
	evidence := 0
	for _, f := range log.lines {
		if len(f) > 0 && f[0] == "evidence" {
			digest, rel := "", ""
			if len(f) > 4 {
				digest = strings.TrimPrefix(f[4], "sha256:")
			}
			if len(f) > 2 {
				rel = f[2]
			}
			fmt.Fprintf(&rows, "%s  bundle/%s\n", digest, rel)
			evidence++
		}
	}
	os.WriteFile(sums, []byte(rows.String()), 0o644)
	if evidence == 0 {
		return refuse("the observer described no evidence")
	}
	ok, verifiedBy := verifyBundleSums(out, at("verify.log"))
	if !ok {
		return refuse("the evidence does not verify against the digests the observer declared")
	}

	// -- and the transcript must carry a peer ID this observer did not generate
	observedOnly := true
	for _, f := range log.lines {
		if len(f) >= 4 && f[0] == "announce" && f[2] == "is-observer-peer-id" && f[3] == "false" {
			observedOnly = false
		}
	}
	if observedOnly {
		return refuse("every announce carried this observer's own peer ID; no build was measured")
	}
	// ⛔ THE HEX IS REQUIRED TO BE HEX, taken from the first announce that was
	// not this observer's own; `absent` is not a peer ID.
	peerHex, pending, indexOf := "", "", ""
	for _, f := range log.lines {
		if len(f) >= 4 && f[0] == "announce" && f[2] == "peer-id" {
			pending, indexOf = f[3], f[1]
		}
		if peerHex == "" && len(f) >= 4 && f[0] == "announce" && f[2] == "is-observer-peer-id" &&
			f[1] == indexOf && f[3] == "false" && pending != "" && strings.Trim(pending, "0123456789abcdef") == "" {
			peerHex = pending
		}
	}
	if peerHex == "" {
		return refuse("no announce carried a readable peer ID; no build was identified")
	}
	if tb, err := os.ReadFile(filepath.Join(bundle, "tracker-http.transcript.json")); err != nil || !bytes.Contains(tb, []byte(peerHex)) {
		return refuse("the transcript does not carry the peer ID the observer reported")
	}

	// -- and a SECOND reading of those same bytes -------------------------------
	//
	// ⛔ IT RUNS AFTER THE VERIFIER, DELIBERATELY: the connector reads the same
	// transcripts the observer wrote, and reading them before they were
	// verified would corroborate bytes nothing had established.
	describeConnector := func(appendErr bool) []byte {
		tmp := at(".connector.describe")
		boundedCall(tmp, false, at("connector.err"), appendErr, adapterSeconds, killAfter,
			python, connector, "--describe")
		b, _ := os.ReadFile(tmp)
		os.Remove(tmp)
		return b
	}
	connectorID := keyField(describeConnector(false), "id")
	if connectorID == "" {
		return cannot("the connector did not describe its own identifier")
	}
	connectorVersion := keyField(describeConnector(true), "version")
	if connectorVersion == "" {
		return cannot("the connector did not describe its own version")
	}
	if strings.HasPrefix(connectorID, "-") || strings.HasSuffix(connectorID, "-") ||
		strings.Contains(connectorID, "--") || strings.Trim(connectorID, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		return cannot("the connector named itself [%s], which is not a slug", connectorID)
	}
	// ⛔ THE REPORT'S PATH IS THE CONNECTOR'S OWN DEFAULT, NEVER COMPOSED HERE.
	switch rc := boundedCall(at("connector.log"), true, at("connector.err"), true, adapterSeconds, killAfter,
		python, connector, "--bundle", bundle); rc {
	case 0:
	case 1:
		indentFile(at("connector.err"))
		return refuse("the second connector could not read what the observer recorded")
	case 124:
		return cannot("the connector did not return within %ds", adapterSeconds)
	default:
		indentFile(at("connector.err"))
		return cannot("the connector could not run (exit %d)", rc)
	}
	report := filepath.Join(bundle, "connector", connectorID+".txt")
	if fi, err := os.Stat(report); err != nil || fi.Size() == 0 {
		return refuse("the connector exited 0 and wrote no report at %s", report)
	}
	// ⚠ A DISAGREEMENT IS RECORDED, NOT REFUSED; silence is what is refused.
	rb, _ := os.ReadFile(report)
	connectorPeer := keyField(rb, "tracker_http/peer_id")
	if connectorPeer == "" {
		return refuse("the connector's report says nothing about tracker_http/peer_id")
	}
	// ⛔ THE REPORT JOINS THE DOCUMENT THAT COVERS THE BUNDLE, and the whole
	// document is re-verified, which says the append disturbed nothing.
	rsum, _ := fileSHA256(report)
	appendFile(sums, fmt.Sprintf("%s  bundle/connector/%s.txt\n", rsum, connectorID))
	if ok, _ := verifyBundleSums(out, at("verify.log")); !ok {
		return refuse("the bundle no longer verifies once the connector's report is covered")
	}
	evidence++

	// -- rule 12, at the choke point rather than in each adapter ----------------
	switch rc := scanBundle([]string{out}); rc {
	case 0:
	case 1:
		return refuse("the capture would ship what rule 12 forbids, named above")
	default:
		return cannot("the rule-12 scan could not run (exit %d)", rc)
	}

	finished := utcNow()
	obsSum, _ := fileSHA256(observer)
	conSum, _ := fileSHA256(connector)
	adSum, _ := fileSHA256(adapter)
	peerSessions := log.first("peer-sessions", 1)
	peerConnections := log.first("peer-connections", 1)
	// ⛔ HOW MANY SESSIONS THE PEER SURFACE WAS DIALLED IN, beside how many
	// started; fewer is a weaker run rather than a refusal, so it is said.
	if peerPort != "" {
		n, _ := strconv.Atoi(orZero(peerSessions))
		if n < started_ {
			fmt.Fprintf(os.Stderr, "capture-client: the peer surface was dialled in %s of %d session(s)\n", orZero(peerSessions), started_)
		}
	}
	uptime := ""
	if b, err := os.ReadFile("/proc/uptime"); err == nil {
		uptime, _, _ = strings.Cut(strings.TrimSpace(string(b)), " ")
	}
	stock := "false"
	if kind == "stock" {
		stock = "true"
	}
	orNone := func(s string) string {
		if s == "" {
			return "none"
		}
		return s
	}

	// -- the attestation -----------------------------------------------------------
	var a strings.Builder
	a.WriteString("bit-ids/capture-attestation/1\n")
	for _, kv := range [][2]string{
		{"run", runID}, {"kind", "client"}, {"target", target}, {"measured_build", measured},
		{"stock_client", stock}, {"adapter", adapter}, {"adapter_kind", kind},
		{"adapter_sha256", adSum}, {"adapter_stop_status", fmt.Sprint(stopRC)},
		{"stopped_within_window", stoppedWithin}, {"stopped_after", fmt.Sprintf("%ds", stoppedAfter)},
		{"stop_margin", fmt.Sprintf("%ds", stopMargin)}, {"sessions", sessions},
		{"sessions_started", fmt.Sprint(started_)}, {"platform", hostPlatform()},
		{"started_at", started}, {"finished_at", finished}, {"host_fingerprint", fingerprint},
		{"claim_marker", marker}, {"uptime_at_finish", uptime}, {"claimed_run", claimedRun},
		{"egress", "closed"}, {"egress_source", egressSource}, {"observer", observer},
		{"observer_sha256", obsSum}, {"observer_version", log.first("observer-version", 1)},
		{"connectors", connectorID}, {"connector", connector}, {"connector_sha256", conSum},
		{"connector_version", connectorVersion}, {"connector_peer_id", connectorPeer},
		{"announce_url", announceURL}, {"info_hash", log.first("info-hash", 1)},
		{"fixture", log.first("fixture-sha256", 1)}, {"announces", announces},
		{"measured_peer_id", peerHex}, {"peer_streams", orZero(log.first("peer-streams", 1))},
		{"peer_dialled", orNone(log.rest("peer-dialled"))}, {"peer_sessions", orZero(peerSessions)},
		{"peer_connections", orZero(peerConnections)}, {"segments", segments},
		{"evidence", fmt.Sprint(evidence)}, {"evidence_verified_by", verifiedBy},
	} {
		fmt.Fprintf(&a, "%s=%s\n", kv[0], kv[1])
	}
	if err := os.WriteFile(attest, []byte(a.String()), 0o644); err != nil {
		return refuse("the attestation was not written")
	}
	// ⛔ READ BACK, NOT ASSUMED.
	if !fileHasLine(attest, "run="+runID) {
		return refuse("the attestation was not written")
	}

	fmt.Println(fingerprint)
	fmt.Fprintf(os.Stdout, "capture-client: %s %s, %s announce(s), peer id %s\n", target, measured, announces, peerHex)
	fmt.Fprintf(os.Stdout, "capture-client: %s segment(s), %d artifact(s), verified by %s\n", segments, evidence, verifiedBy)
	fmt.Fprintf(os.Stdout, "capture-client: attestation %s\n", attest)
	return nil
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func fileHasLine(path, want string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if sc.Text() == want {
			return true
		}
	}
	return false
}
