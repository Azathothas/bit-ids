// install-step - what *Install the client* does, in one process, so the step's
// own process can be bounded from outside it.
//
// CI-11. It was scripts/acquisition/install-step.sh, the most platform-bound
// file on the capture path; what is the same on every platform is here, and
// what a session or a job object is lives in installstep_unix.go and
// installstep_windows.go.
//
// ⛔ EVERY BOUND TRIED BEFORE THIS ONE FAILED, AND EACH FOR ITS OWN REASON.
// `timeout` inside `install-client` ends its own child and not a shell blocked
// around it; `timeout-minutes` was measured on run 6 not to end the step; a
// watchdog loop in the step's own shell was measured on run 9 not to fire. ⭐
// So the workflow bounds THIS process from outside, and nothing inside it has
// to be reachable for that to work. ⚠ The point is that the step ENDS: a job
// whose runner is killed leaves no log and no artifact, measured on runs 7, 8
// and 9.
//
// ⛔ AND NOTHING IT SPAWNS INHERITS THE STEP'S OUTPUT. A runner ends a step
// when the command has exited AND the step's output pipe has reached end of
// file, so the install's output goes to `step.log` and is printed afterwards.
//
// ⛔ THE INSTALL RUNS IN A CONTAINER OF ITS OWN: a session on Unix, a job
// object on Windows. Nested bounds each start a process group of their own, so
// a bound reaches the group it started and not the one beneath: measured on
// 2026-09-16, a `sleep 600` outlived every bound with PPID 1. Everything the
// install starts stays in the container, so after the wait this lists it into
// `orphans.log` and ends it. `CI-08`.
//
// ⛔ AND THIS NEVER WAITS ON A PROCESS THAT MAY NOT DIE. The loop has a
// deadline of its own: an install still running at it is reported as 124, the
// number coreutils uses for "it never answered", from this side of the bound.
// ⚠ TEN bounds were measured not to end this step on `capture-client` runs 3
// to 10 and 21 to 25, because a bound only ever acts on the command and the
// step also waits on its pipe.
//
// Usage:
//
//	ADAPTER=<path> ROUTE=<name> RUNNER_TEMP=<dir> bit-check install-step
//
// Exit codes: 0 installed, 1 the route or a guard refused, 2 could not run,
// 124 the install never answered.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func installStep(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "install-step: it takes no arguments; ADAPTER, ROUTE and RUNNER_TEMP say what to do")
		return 2
	}
	say := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "install-step: "+f+"\n", a...) }
	env := map[string]string{}
	for _, k := range []string{"ADAPTER", "ROUTE", "RUNNER_TEMP"} {
		env[k] = os.Getenv(k)
		if env[k] == "" {
			say("%s is required", k)
			return 2
		}
	}
	adapter, route, temp := env["ADAPTER"], env["ROUTE"], env["RUNNER_TEMP"]
	workdir := filepath.Join(temp, "install-"+route)
	record := filepath.Join(temp, "install-"+route+".txt")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		say("cannot create %s", workdir)
		return 2
	}

	// ⚠ THE URL TRAVELS THROUGH A FILE RATHER THAN THROUGH GITHUB_ENV, so the
	// value the install used is on disk beside the record that names it. ⛔ And
	// the whole digest disposition travels as the resolution itself: four
	// adapters choosing between four dispositions would be four copies of one
	// decision. `ACQ-06`.
	if route == "release" {
		urlFile := filepath.Join(temp, "release", "url.txt")
		if !isRegular(urlFile) {
			say("the release lane has no resolved artifact")
			return 2
		}
		url := readTrimmed(urlFile)
		os.Setenv("BIT_IDS_RELEASE_URL", url)
		fmt.Printf("the release route will fetch %s\n", url)
		res := filepath.Join(temp, "release", "resolution.txt")
		if !isRegular(res) {
			say("the release lane has no resolution to take a digest from")
			return 2
		}
		os.Setenv("BIT_IDS_RELEASE_RESOLUTION", res)
		body, _ := os.ReadFile(res)
		fmt.Printf("the release route takes its digest disposition from %s (%s)\n", res, recordField(body, "digest_source"))
	}
	// ⛔ THE SOURCE ROUTE TAKES ITS TAG FROM ITS OWN RESOLUTION. Two routes
	// sharing a resolver are one route, which `E-ACQ-07` refuses; two
	// resolutions landing on two versions is a vendor that moved, and catching
	// it is the correct outcome.
	if route == "source" {
		res := filepath.Join(temp, "source", "resolution.txt")
		if !isRegular(res) {
			say("the source lane has no resolution of its own to take a tag from")
			return 2
		}
		body, _ := os.ReadFile(res)
		tag := recordField(body, "selected_tag")
		if tag == "" {
			say("the source resolution record names no selected_tag")
			return 2
		}
		os.Setenv("BIT_IDS_SOURCE_TAG", tag)
		fmt.Printf("the source route will build %s, resolved from this repository refs\n", tag)
	}

	// ⚠ THE RELATION IS WHAT MATTERS, not the number: LARGER than
	// install-client's inner bounds, so a slow route is refused there with a
	// message, and SMALLER than the workflow step's own bound, so this is
	// reached first. The workflow states both.
	stepSeconds := envSeconds("BIT_IDS_STEP_TIMEOUT", 1020)
	killAfter := envSeconds("BIT_IDS_KILL_AFTER", 20)
	interval := envSeconds("BIT_IDS_INSTALL_INTERVAL", 5)
	psSeconds := envSeconds("BIT_IDS_PS_TIMEOUT", 20)

	self, err := os.Executable()
	if err != nil {
		say("cannot find this binary to run install-client with")
		return 2
	}
	stepLog := filepath.Join(workdir, "step.log")
	logf, err := os.Create(stepLog)
	if err != nil {
		say("cannot write %s", stepLog)
		return 2
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		say("cannot open %s", os.DevNull)
		return 2
	}
	cmd := exec.Command(self, "install-client", "--adapter", adapter, "--route", route,
		"--workdir", workdir, "--record", record)
	// ⛔ THE REDIRECTION IS THIS PROCESS'S: the descriptor the install inherits
	// is that FILE and never this step's output pipe.
	cmd.Stdout, cmd.Stderr, cmd.Stdin = logf, logf, null
	box, err := startContained(cmd)
	logf.Close()
	null.Close()
	if err != nil {
		say("the install could not be started: %v", err)
		return 2
	}

	done := make(chan int, 1)
	go func() { done <- waitCode(cmd) }()

	// ⭐ THE LOOP IS THE EVIDENCE, NOT THE BOUND: a process table beside the
	// clock is what separates a command that is slow from one that is stopped.
	// ⛔ `ps` IS BOUNDED, because a read of `/proc` for a task wedged in the
	// kernel can block in exactly the way this loop exists to observe; a sample
	// headed and empty says the host stopped answering.
	watchdog := filepath.Join(workdir, "watchdog.log")
	rc, finished, waited := 0, false, 0
	deadline := time.NewTimer(time.Duration(stepSeconds) * time.Second)
	defer deadline.Stop()
loop:
	for {
		select {
		case rc = <-done:
			finished = true
			break loop
		case <-deadline.C:
			break loop
		case <-time.After(time.Duration(interval) * time.Second):
			waited += interval
			sample := fmt.Sprintf("## +%ds %s\n", waited, utcNow())
			out, ok := processTable(psSeconds)
			if ok {
				sample += out
			} else {
				sample += out + fmt.Sprintf("## ps did not answer within %ds\n", psSeconds)
			}
			appendFile(watchdog, sample)
		}
	}
	if !finished {
		// ⚠ The install's own bound, `timeout -k` in the shell half, fires
		// here: TERM now and KILL after the grace, without waiting on either.
		box.terminate(killAfter)
		rc = 124
		say("the install was still running after %ds; it is hung, not slow", stepSeconds)
	}

	// ⛔ AND WHAT THE INSTALL LEFT IN ITS CONTAINER IS NAMED AND ENDED.
	swept, listing := box.sweep(psSeconds)
	os.WriteFile(filepath.Join(workdir, "orphans.log"), []byte(listing), 0o644)
	if swept > 0 {
		say("ended %d process(es) the install left in its session; orphans.log names them", swept)
	}
	if b, err := os.ReadFile(stepLog); err == nil {
		os.Stdout.Write(b)
	}

	// ⭐ AND WHATEVER STILL HOLDS THE LOG, OR THIS STEP'S OWN OUTPUT, IS NAMED.
	// The file that keeps a STEP open is the step's own output, so that is
	// asked about too. ⚠ Bounded, because it walks every process's
	// descriptors, and a diagnostic that held the step open would be the
	// failure it exists to explain.
	held := reportHolders(adapter, stepLog, filepath.Join(workdir, "holders.log"))
	fmt.Printf("the holder report exited %d\n", held)

	if rc != 0 {
		return rc
	}
	if b, err := os.ReadFile(record); err == nil {
		os.Stdout.Write(b)
	}
	return 0
}

func appendFile(path, text string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	f.WriteString(text)
	f.Close()
}

// waitCode waits for a started command and answers its exit code, 128 plus the
// signal for one a signal ended, as a shell's `$?` would.
func waitCode(cmd *exec.Cmd) int {
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if code := ee.ExitCode(); code >= 0 {
			return code
		}
		return signalCode(ee)
	}
	return 126
}

// boundedOutput runs a diagnostic with a bound and answers what it printed and
// whether it answered in time.
func boundedOutput(seconds int, prog string, args ...string) (string, bool) {
	cmd := exec.Command(prog, args...)
	var out strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		return err.Error() + "\n", true
	}
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	select {
	case <-done:
		return out.String(), true
	case <-time.After(time.Duration(seconds) * time.Second):
		cmd.Process.Kill()
		<-done
		return "", false
	}
}
