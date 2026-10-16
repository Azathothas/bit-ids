// flight-recorder - push the state of this runner OFF the runner, every few
// seconds, for as long as the job lasts.
//
// CI-11. It was scripts/ci/flight-recorder.sh and is here whole: the same
// reads under the same bounds, the same check-run requests through `curl`, the
// same stop and done files, and the same lines on its output. A dry run prints
// one snapshot and posts nothing, as before.
//
// ⛔ WHY IT EXISTS. `capture-client` runs 3 to 34 wedged in *Install the client*
// and left NOTHING: no artifact, and no job log at all - the log endpoint answers
// `BlobNotFound` even for the steps that finished. Every instrument tried so far
// wrote to the runner's own disk and waited for an upload step that never ran, so
// every one of them measured nothing. This one does not wait: each snapshot leaves
// the host the moment it is taken, as the output of a check run on the commit.
//
// ⭐ WHAT ITS SILENCE MEANS IS PART OF THE MEASUREMENT. A runner that keeps
// posting is alive and the snapshot says what is stuck; a runner whose posts STOP
// lost contact at that instant, and the last snapshot is the state it lost it in.
//
// ⛔ AND A JOB THAT ENDED IN CONTACT SAYS SO, WHICH IT DID NOT UNTIL 2026-10-14.
// The loop ran to a 3000-second limit no job reaches, so the runner killed the
// recorder at the end of every job and every check run it had opened stayed
// `in_progress` forever: a host that lost contact and one that finished cleanly
// looked the same. ⭐ The job's last step touches `<log-dir>/flight-recorder.stop`;
// the recorder completes its check run in the request that posts its final
// snapshot, reads the answer back, and writes `<log-dir>/flight-recorder.done`.
// A check run still `in_progress` after its job ended is now a host that never
// reached that step, and one that closed still shows the state the job ended in.
//
// ⚠ IT READS NOTHING THAT CAN BLOCK IT. `ps` is asked for `stat` and `wchan`,
// which come from `/proc/<pid>/stat` and `wchan`; never for `args`, because
// reading a process's command line takes its memory lock and a reader can hang
// on exactly the process it is trying to describe. Every read is under
// `timeout`, and a read that times out is reported as such.
//
// ⚠ IT IS A DIAGNOSTIC. It never refuses anything and it never changes the host.
// The kernel's ring buffer and the stacks of blocked tasks need root, so they are
// asked through `sudo -n`, which answers nothing rather than prompting where it
// is not available. ⚠ It reads `/proc`, so it is not run on Windows, whose lane
// never cuts its route and so needs nothing seen through a cut.
//
// ⚠ THE REQUESTS GO THROUGH `curl`, AS THE SHELL HALF'S DID, so the API sees the
// same client and `check-step-bodies` drives this against the same stub `curl`.
// The request bodies are written by this binary rather than by `jq`.
//
// Usage (backgrounded by the first step of a job):
//
//	GH_TOKEN=... bit-check flight-recorder <label> <log-dir>
//
// Environment: GH_TOKEN, GITHUB_API_URL, GITHUB_REPOSITORY, GITHUB_SHA, which a
// hosted runner provides except the token, which the step passes.
// BIT_IDS_RECORDER_INTERVAL (seconds, default 15) and BIT_IDS_RECORDER_LIMIT
// (seconds, default 3000) bound how often and how long it posts.
// BIT_IDS_RECORDER_DRYRUN=1 prints one snapshot and posts nothing.
//
// Exit codes: 0 it ran until the job's last step or its limit, 2 it could not
// start.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type recorder struct {
	label, logDir, api, token, apiURL string
	scratch                           string
}

func flightRecorder(args []string) int {
	say := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "flight-recorder: "+f+"\n", a...) }
	if len(args) != 2 {
		say("takes <label> <log-dir>")
		return 2
	}
	if runtime.GOOS == "windows" {
		say("reads /proc, so it does not run on Windows")
		return 2
	}
	for _, v := range []string{"GH_TOKEN", "GITHUB_API_URL", "GITHUB_REPOSITORY", "GITHUB_SHA"} {
		if os.Getenv(v) == "" {
			say("%s is not set", v)
			return 2
		}
	}
	for _, tool := range []string{"curl", "timeout", "ps"} {
		if _, err := exec.LookPath(tool); err != nil {
			say("%s is not on this host", tool)
			return 2
		}
	}
	interval, ok := recorderSeconds("BIT_IDS_RECORDER_INTERVAL", 15)
	if !ok {
		say("BIT_IDS_RECORDER_INTERVAL is not a whole number of seconds")
		return 2
	}
	limit, ok := recorderSeconds("BIT_IDS_RECORDER_LIMIT", 3000)
	if !ok {
		say("BIT_IDS_RECORDER_LIMIT is not a whole number of seconds")
		return 2
	}
	scratch, err := os.MkdirTemp("", "flight-recorder.")
	if err != nil {
		return 2
	}
	r := &recorder{
		label:   args[0],
		logDir:  args[1],
		apiURL:  os.Getenv("GITHUB_API_URL"),
		token:   os.Getenv("GH_TOKEN"),
		scratch: scratch,
	}
	r.api = r.apiURL + "/repos/" + os.Getenv("GITHUB_REPOSITORY") + "/check-runs"

	// ⚠ A DRY RUN PRINTS ONE SNAPSHOT AND POSTS NOTHING, which is how this is
	// driven anywhere but a runner.
	if os.Getenv("BIT_IDS_RECORDER_DRYRUN") == "1" {
		os.Stdout.Write(r.snapshot(0))
		return 0
	}

	// ⚠ The first post creates the check run; every later one updates it. A
	// token that cannot create one is reported once and the recorder stops,
	// because a recorder that cannot post records nothing.
	create := r.file("create.json")
	writeJSON(create, recorderCreate{
		Name:    "flight-recorder (" + r.label + ")",
		HeadSHA: os.Getenv("GITHUB_SHA"),
		Status:  "in_progress",
		Output:  recorderOutput{Title: "starting", Summary: "starting"},
	})
	id := jsonField(r.call("POST", r.api, create), "id")
	if id == "" {
		curlErr, _ := os.ReadFile(r.file("curl.err"))
		say("could not create a check run: %s", strings.TrimRight(string(curlErr), "\n"))
		return 2
	}
	fmt.Printf("flight-recorder: posting to check run %s every %ds\n", id, interval)

	stopFile := filepath.Join(r.logDir, "flight-recorder.stop")
	doneFile := filepath.Join(r.logDir, "flight-recorder.done")
	stopped := func() bool { _, err := os.Lstat(stopFile); return err == nil }
	run := r.api + "/" + id

	start := time.Now().Unix()
	n := 0
	ending := fmt.Sprintf("the limit of %ds was reached", limit)
	for {
		elapsed := time.Now().Unix() - start
		if elapsed >= int64(limit) {
			break
		}
		if stopped() {
			ending = "the job reached its last step"
			break
		}
		n++
		r.update(fmt.Sprintf("snapshot %d", n), elapsed, false)
		r.call("PATCH", run, r.file("update.json"))
		// ⚠ ONE SECOND AT A TIME, so the stop is seen within a second rather
		// than an interval: the job's last step waits on this, and a host torn
		// down while it waited would leave the check run open over a job that
		// ended in contact.
		for slept := 0; slept < interval && !stopped(); slept++ {
			time.Sleep(time.Second)
		}
	}

	// ⭐ THE LAST SNAPSHOT AND THE COMPLETION, AND THE COMPLETION IS READ BACK.
	// A PATCH that was refused leaves the run `in_progress`, which is the
	// silence this file says means lost contact; so the answer is what the done
	// file reports, and a run the API did not confirm says that rather than
	// claiming it closed.
	//
	// ⛔ ONE REQUEST, BECAUSE TWO ERASED THE SNAPSHOT. Until 2026-10-14 the last
	// snapshot went up first and the completion after it, carrying an `output`
	// with a title and no `text`; measured on `capture-client` run 35, both
	// check runs that closed cleanly showed that title over an empty text. A
	// check run's output is replaced by the update that carries one.
	r.update(ending, time.Now().Unix()-start, true)
	answerBody := r.call("PATCH", run, r.file("update.json"))
	os.WriteFile(r.file("answer.json"), answerBody, 0o644)
	answer := jsonField(answerBody, "status")
	// ⚠ AND A SNAPSHOT THE API REFUSES MUST NOT KEEP THE RUN OPEN. One request
	// carries both, so a text the API would not take would refuse the
	// completion with it. An answer that is not `completed` is asked once more
	// with no `output`, which leaves the last snapshot the run accepted. ⚠ Only
	// when the API answered: a request that timed out says nothing about its
	// text, and a second one would spend the wait the job's last step gives this.
	if answer != "completed" && len(answerBody) > 0 {
		done := r.file("done.json")
		writeJSON(done, recorderClose{Status: "completed", Conclusion: "neutral"})
		answer = jsonField(r.call("PATCH", run, done), "status")
	}
	line := "completed: " + ending + "\n"
	if answer != "completed" {
		line = "not confirmed: the check run answered [" + answer + "]\n"
	}
	os.WriteFile(doneFile, []byte(line), 0o644)
	if b, err := os.ReadFile(doneFile); err == nil {
		os.Stdout.Write(b)
	}
	return 0
}

// recorderSeconds reads a bound from the environment. ⚠ A value that is not
// a whole number is refused at the start; the shell half's arithmetic would
// have failed in the loop instead.
func recorderSeconds(name string, def int) (int, bool) {
	v := os.Getenv(name)
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func (r *recorder) file(name string) string { return filepath.Join(r.scratch, name) }

type recorderOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text,omitempty"`
}

type recorderCreate struct {
	Name    string         `json:"name"`
	HeadSHA string         `json:"head_sha"`
	Status  string         `json:"status"`
	Output  recorderOutput `json:"output"`
}

type recorderUpdate struct {
	Output     recorderOutput `json:"output"`
	Status     string         `json:"status,omitempty"`
	Conclusion string         `json:"conclusion,omitempty"`
}

type recorderClose struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

func writeJSON(path string, v any) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	os.WriteFile(path, b.Bytes(), 0o644)
}

// jsonField is `jq -r '.<name> // empty'`: the field as text, or "" when the
// answer is not a JSON object or the field is absent, null or false.
func jsonField(body []byte, name string) string {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return ""
	}
	switch v := m[name].(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
	}
	return ""
}

// call is one request to the checks API, bounded as the shell half bounded
// it, with curl's stderr appended to curl.err. It answers the response body.
func (r *recorder) call(method, url, bodyFile string) []byte {
	cmd := exec.Command("timeout", "20", "curl", "-sS", "--max-time", "15", "-X", method, url,
		"-H", "Authorization: Bearer "+r.token,
		"-H", "Accept: application/vnd.github+json",
		"--data-binary", "@"+bodyFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	if f, err := os.OpenFile(r.file("curl.err"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		defer f.Close()
		cmd.Stderr = f
	}
	cmd.Run()
	return out.Bytes()
}

// update writes one snapshot as the body of a check run update. `closing`
// closes the run in the same request.
func (r *recorder) update(prefix string, elapsed int64, closing bool) {
	snap := r.snapshot(elapsed)
	os.WriteFile(r.file("snap.txt"), snap, 0o644)
	// ⚠ The output text is capped well under the API's 65535 characters,
	// keeping the TAIL, because the newest lines are the ones a hang is about.
	if len(snap) > 60000 {
		snap = snap[len(snap)-60000:]
	}
	os.WriteFile(r.file("text.txt"), snap, 0o644)
	title := fmt.Sprintf("%s at +%ds", prefix, elapsed)
	u := recorderUpdate{Output: recorderOutput{Title: title, Summary: title, Text: string(snap)}}
	if closing {
		u.Status, u.Conclusion = "completed", "neutral"
	}
	writeJSON(r.file("update.json"), u)
}

// section is one read, bounded, labelled, never fatal: its title, the last 60
// lines of what it printed, and its exit code when that was not 0.
func (r *recorder) section(w io.Writer, title string, seconds int, argv ...string) {
	fmt.Fprintf(w, "\n### %s\n", title)
	// ⛔ INTO A FILE, THEN THE TAIL: a pipe's status would be the tail's, and a
	// read that timed out would report as one that answered.
	out, rc := r.bounded(seconds, "section.out", argv...)
	w.Write(lastLines(out, 60))
	if rc != 0 {
		timedOut := ""
		if rc == 124 {
			timedOut = ", TIMED OUT"
		}
		fmt.Fprintf(w, "(exit %d%s)\n", rc, timedOut)
	}
}

// bounded runs `timeout -k 2 <seconds> <argv...>` with both streams in one
// scratch file, and answers what it wrote and its exit code.
func (r *recorder) bounded(seconds int, name string, argv ...string) ([]byte, int) {
	path := r.file(name)
	f, err := os.Create(path)
	if err != nil {
		return nil, 2
	}
	cmd := exec.Command("timeout", append([]string{"-k", "2", strconv.Itoa(seconds)}, argv...)...)
	cmd.Stdout, cmd.Stderr = f, f
	rc := 0
	if err := cmd.Run(); err != nil {
		rc = 127
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		}
	}
	f.Close()
	b, _ := os.ReadFile(path)
	return b, rc
}

// lastLines is `tail -n <n>`.
func lastLines(b []byte, n int) []byte {
	i := len(b) - 1
	if i >= 0 && b[i] == '\n' {
		i--
	}
	for ; i >= 0; i-- {
		if b[i] == '\n' {
			n--
			if n == 0 {
				return b[i+1:]
			}
		}
	}
	return b
}

// textLines splits what a command printed into the lines awk would read.
func textLines(b []byte) []string {
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func (r *recorder) snapshot(elapsed int64) []byte {
	var w bytes.Buffer
	fmt.Fprintf(&w, "elapsed %ds, %s\n", elapsed, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	r.section(&w, "uptime and load", 5, "cat", "/proc/loadavg")
	r.section(&w, "memory", 5, "free", "-m")
	r.section(&w, "disk", 5, "df", "-h", "/", "/tmp", r.logDir)
	// ⚠ D is uninterruptible sleep and Z a zombie: the two states a signal
	// cannot end, so the two a bound cannot end either.
	fmt.Fprintf(&w, "\n### blocked or zombie tasks\n")
	ps, rc := r.bounded(5, "ps.out", "ps", "-eo", "pid,ppid,pgid,stat,wchan:32,etimes,comm")
	if rc != 0 {
		fmt.Fprintf(&w, "(ps exit %d)\n", rc)
	}
	var blocked []string
	for i, line := range textLines(ps) {
		f := strings.Fields(line)
		state := ""
		if len(f) >= 4 {
			state = f[3]
		}
		if i == 0 || strings.HasPrefix(state, "D") || strings.HasPrefix(state, "Z") {
			fmt.Fprintln(&w, line)
		}
		if i > 0 && strings.HasPrefix(state, "D") {
			blocked = append(blocked, f[0])
		}
	}
	// ⭐ WHERE IN THE KERNEL EACH BLOCKED TASK IS WAITING, which is the one fact
	// a process table cannot give and the one that names a cause.
	for _, pid := range blocked {
		r.section(&w, "kernel stack of "+pid, 5, "sudo", "-n", "cat", "/proc/"+pid+"/stack")
	}
	r.section(&w, "the newest 40 tasks", 5, "sh", "-c", "ps -eo pid,ppid,pgid,stat,wchan:32,etimes,comm --sort=etimes | head -41")
	r.section(&w, "kernel ring buffer", 5, "sudo", "-n", "dmesg", "--ctime")
	for _, pattern := range []string{"install-step.log", "install-*/step.log", "install-*/watchdog.log", "install-*/install.log"} {
		matches, _ := filepath.Glob(filepath.Join(r.logDir, pattern))
		for _, f := range matches {
			if st, err := os.Stat(f); err != nil || !st.Mode().IsRegular() {
				continue
			}
			r.section(&w, "tail of "+f, 5, "tail", "-n", "30", f)
		}
	}
	// ⭐ THE NEWEST FILES ANYWHERE UNDER THE LOG DIRECTORY, which is where a
	// traced install writes as it goes: the last line that leaves the host is
	// the last thing the install did before it stopped being able to say so.
	r.section(&w, "newest files", 5, "sh", "-c",
		`find "$1" -type f -newermt "-120 seconds" -printf "%TT %s %p\n" 2>/dev/null | sort | tail -n 15`, "sh", r.logDir)
	for _, f := range r.newestLogs() {
		r.section(&w, "tail of "+f, 5, "tail", "-n", "25", f)
	}
	r.section(&w, "route to the API", 10, "curl", "-sS", "-o", "/dev/null", "-w", `%{http_code} in %{time_total}s\n`,
		"--max-time", "8", r.apiURL+"/zen")
	return w.Bytes()
}

// newestLogs is the four most recently written logs under the log directory
// in the last minute, oldest first: `find -printf '%T@ %p\n' | sort -n | tail
// -n 4 | cut -d' ' -f2-`.
func (r *recorder) newestLogs() []string {
	cmd := exec.Command("find", r.logDir, "-type", "f", "-newermt", "-60 seconds",
		"(", "-name", "*.log", "-o", "-name", "*.err", "-o", "-name", "*.txt", ")", "-printf", `%T@ %p\n`)
	out, _ := cmd.Output()
	type entry struct {
		at   float64
		line string
	}
	var es []entry
	for _, line := range textLines(out) {
		at, _ := strconv.ParseFloat(strings.SplitN(line, " ", 2)[0], 64)
		es = append(es, entry{at, line})
	}
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].at != es[j].at {
			return es[i].at < es[j].at
		}
		return es[i].line < es[j].line
	})
	if len(es) > 4 {
		es = es[len(es)-4:]
	}
	var paths []string
	for _, e := range es {
		p := e.line
		if _, rest, found := strings.Cut(e.line, " "); found {
			p = rest
		}
		paths = append(paths, p)
	}
	return paths
}
