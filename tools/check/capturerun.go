// capture-run - the fixture capture, on a host that has already been claimed
// and already had its route off the machine deleted.
//
// CI-11. It was scripts/capture/capture-run.sh and is here whole: every guard,
// in the order the shell half ran them, and every message. capture-run.ps1
// stays the Windows half until the Windows lane is built on this one.
//
// ⛔ IT BUILDS NOTHING. By the time this runs there is no network, so the
// observer is a path to an ALREADY BUILT binary, and a missing one is `could
// not run` rather than something to repair in place.
//
// ⛔ IT IS A FIXTURE CAPTURE. Nothing here installs a client: `capture.kind`
// in the attestation says `fixture`, so a bundle cannot be mistaken for a stock
// build's. What is measured is the containment and the evidence path, which is
// `CI-03`'s Prove and not `CLIENT-01`'s.
//
// ⭐ THE DRIVER IS A CLIENT THIS PROJECT DID NOT WRITE. `curl` puts real bytes
// through the tracker observer, and it sends a token it knows, `key=<run-id>`,
// which the transcript must hold: without that, an observer that wrote an empty
// artifact and a bundle that verified against its own empty digest would pass
// every other check here.
//
// ⛔ EVERY REFUSAL SAYS SOMETHING DIFFERENT, and `check-capture.sh` asserts
// the message on every case, not the code alone.
//
// Usage:
//
//	bit-check capture-run --run-id <id> --out <dir> --observer <path>
//	                      [--seconds <n>] [--route-table <path>]
//
// Exit codes: 0 the capture ran and its evidence verifies, 1 a guard refuses,
// 2 could not run.
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const captureRunUsage = `usage: bit-check capture-run --run-id <id> --out <dir> --observer <path>
                             [--seconds <n>] [--route-table <path>]

Exit codes: 0 the capture ran and its evidence verifies, 1 a guard refuses,
2 could not run.
`

func captureRun(args []string) int {
	return resolveMain("capture-run", captureRunUsage, args, runCaptureRun)
}

func runCaptureRun(args []string) error {
	var runID, out, observer, routeTable string
	seconds := "15"
	help, err := parseResolveArgs("capture-run", args, map[string]*string{
		"--run-id": &runID, "--out": &out, "--observer": &observer,
		"--seconds": &seconds, "--route-table": &routeTable,
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
	}
	// ⚠ The guard's slug rule, restated: this value becomes a token on the wire
	// and a key in a document before the guard is consulted.
	if strings.Trim(runID, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		return cannot("run id must be lowercase a-z0-9-: %s", runID)
	}
	if !wholeNumber(seconds) {
		return cannot("--seconds must be a whole number: %s", seconds)
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return cannot("curl not found")
	}
	if sumsReader() == "" {
		return cannot("sha256sum not found")
	}

	// ⛔ THE BUILD RAN BEFORE THE NETWORK WAS CUT, OR IT DID NOT RUN: the one
	// refusal that reports a workflow whose step order has drifted.
	if fi, err := os.Stat(observer); err != nil || fi.IsDir() || (runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0) {
		return cannot("the observer %s is not executable; it is built before the route is deleted, never here", observer)
	}
	// ⛔ A SECOND CAPTURE MUST NOT WRITE OVER THE FIRST ONE'S EVIDENCE.
	if _, err := os.Lstat(out); err == nil {
		return refuse("%s already exists; a capture never writes into another run's evidence", out)
	}

	// -- the host, as this run finds it -------------------------------------
	//
	// ⛔ THE MARKER'S PATH IS ASKED FOR, NEVER COMPOSED, and the guard is this
	// binary. ⛔ The claim is checked here as well as run earlier: a step order
	// that dropped the claim step leaves this capturing on a host nothing
	// claimed, and a guard on one of two paths in is the one-gated-door defect.
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

	// ⛔ AND THE ROUTE IS READ AGAIN AT THE MOMENT OF CAPTURE, and the table it
	// was read from is recorded, so a run over a fixture table is visible.
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
	bundle := at("bundle")
	logPath := at("observer.log")
	attest := at("attestation.txt")
	started := utcNow()

	// -- the run --------------------------------------------------------------
	//
	// ⚠ The observer prints its endpoints and then serves until its own
	// deadline; waiting on the line rather than a fixed delay is what makes this
	// work on a slow runner.
	obsLog, err := os.Create(logPath)
	if err != nil {
		return cannot("cannot write %s", logPath)
	}
	obs := exec.Command(observer, bundle, seconds)
	obs.Stdout, obs.Stderr = obsLog, obsLog
	if err := obs.Start(); err != nil {
		obsLog.Close()
		return cannot("the observer could not be started: %v", err)
	}
	obsLog.Close()
	obsDone := make(chan int, 1)
	go func() { obsDone <- waitCode(obs) }()
	exited := func() bool {
		select {
		case rc := <-obsDone:
			obsDone <- rc
			return true
		default:
			return false
		}
	}
	for waited := 0; waited < 30; waited++ {
		if lb, _ := os.ReadFile(logPath); bytes.Contains(append([]byte("\n"), lb...), []byte("\nserving for ")) {
			break
		}
		if exited() {
			break
		}
		time.Sleep(time.Second)
	}
	tracker := ""
	for _, f := range readObserverLog(logPath).lines {
		if len(f) > 2 && f[0] == "endpoint" && f[1] == "tracker-http" {
			tracker = f[2]
			break
		}
	}
	if tracker == "" {
		<-obsDone
		indentFile(logPath)
		return refuse("the observer never reported a tracker-http endpoint")
	}

	// ⭐ A TOKEN THE DRIVER KNOWS IT SENT. BEP 3's `key` is an opaque client
	// value, so a tracker observer keeps it verbatim.
	driverVersion := ""
	if vb, err := exec.Command("curl", "--version").Output(); err == nil {
		driverVersion, _, _ = strings.Cut(string(vb), "\n")
	}
	infoHash := strings.Repeat("%5a", 20)
	announce := "http://" + tracker + "/announce?info_hash=" + infoHash + "&peer_id=bit-ids-fixture-0001" +
		"&port=6881&uploaded=0&downloaded=0&left=0&compact=1" +
		"&event=started&key=" + runID

	// ⚠ --noproxy is not tidiness: a proxy variable would send the announce
	// off this host, the one thing the containment exists to stop.
	driverErr := at("driver.err")
	ef, err := os.Create(driverErr)
	if err != nil {
		return cannot("cannot write %s", driverErr)
	}
	var status bytes.Buffer
	drv := exec.Command("curl", "-sS", "--noproxy", "*", "--max-time", "20", "-o", os.DevNull, "-w", "%{http_code}", announce)
	drv.Stdout, drv.Stderr = &status, ef
	driverRC := 0
	if err := drv.Start(); err != nil {
		driverRC = 127
	} else {
		driverRC = waitCode(drv)
	}
	ef.Close()
	observerRC := <-obsDone

	if driverRC != 0 {
		indentFile(driverErr)
		return refuse("the driver could not reach the observer (curl exit %d)", driverRC)
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

	// -- the evidence, checked by something else ------------------------------
	//
	// ⭐ THE ROWS ARE THE OBSERVER'S OWN CLAIMS AND THE HOST'S VERIFIER IS WHAT
	// TESTS THEM: the writer checking itself is not a check.
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
	os.WriteFile(at("SHA256SUMS"), []byte(rows.String()), 0o644)
	if evidence == 0 {
		return refuse("the observer described no evidence")
	}
	ok, verifiedBy := verifyBundleSums(out, at("verify.log"))
	if !ok {
		return refuse("the evidence does not verify against the digests the observer declared")
	}

	// ⛔ AND THE TRANSCRIPT MUST CARRY WHAT THE DRIVER SENT: the only check that
	// says a client's bytes reached the record.
	tokenHex := hex.EncodeToString([]byte("key=" + runID))
	if tb, err := os.ReadFile(filepath.Join(bundle, "tracker-http.transcript.json")); err != nil || !bytes.Contains(tb, []byte(tokenHex)) {
		return refuse("the transcript does not carry the bytes the driver sent")
	}

	finished := utcNow()
	obsSum, _ := fileSHA256(observer)

	// -- the attestation ------------------------------------------------------
	//
	// ⚠ Key=value, one per line, the shape the claim marker uses.
	var a strings.Builder
	a.WriteString("bit-ids/capture-attestation/1\n")
	for _, kv := range [][2]string{
		{"run", runID}, {"kind", "fixture"}, {"measured_build", "none"}, {"stock_client", "false"},
		{"platform", hostPlatform()}, {"started_at", started}, {"finished_at", finished},
		{"host_fingerprint", fingerprint}, {"claim_marker", marker}, {"claimed_run", claimedRun},
		{"egress", "closed"}, {"egress_source", egressSource}, {"observer", observer},
		{"observer_sha256", obsSum}, {"driver", "curl"}, {"driver_version", driverVersion},
		{"driver_status", status.String()}, {"segments", segments},
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
	fmt.Printf("capture-run: %s segment(s), %d artifact(s), verified by %s\n", segments, evidence, verifiedBy)
	fmt.Printf("capture-run: attestation %s\n", attest)
	return nil
}
