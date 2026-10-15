// harness.go - what a ported mutation harness needs, in one place.
//
// ⭐ THIS IS scripts/corpus/store-lib.sh AND ITS POWERSHELL TWIN, ported only as
// far as a ported harness calls it. CI-10. Every harness that library serves
// builds an example, makes a scratch directory, plants a literal exactly once,
// counts a row and prints one verdict; the first two harnesses to leave the twin
// layer are `check-cache` and `check-catalogue`, and they call exactly this.
//
// ⛔ A FUNCTION NOTHING CALLS IS A FUNCTION NOBODY KNOWS WORKS. The shell
// library's `place`, `tree_digest` and `tree_files` are deliberately absent, for
// the reason its PowerShell twin gave for leaving them out: they land with the
// first ported harness that exercises them.
//
// -- THE REPORT IS THE CONTRACT -----------------------------------------------
//
// A ported harness answers exactly what its shell half answered: the same
// `{"schema":...,"total":...,"passed":...,"failed":...}` line under `--json`,
// and the same verdict - a run that passed nothing is red whatever else it says.
// `check-bitcheck.sh --compare` held the two to that, byte for byte, before
// either half was deleted.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// harness accumulates rows the way store-lib's `pass` and `fail` do.
type harness struct {
	me   string
	rows []string
	pass int
	fail int
}

func (h *harness) ok(label string) {
	h.rows = append(h.rows, "✅ "+label)
	h.pass++
}

func (h *harness) bad(label string) {
	h.rows = append(h.rows, "❌ "+label)
	h.fail++
}

// report is store_report: the JSON line alone under --json, or every row, the
// failing rows again, and a summary.
//
// ⛔ THE FAILING ROWS ARE REPRINTED AT THE END, for the reason the shell half
// gives: the gate shows the TAIL of a failed check's log, and a harness prints
// dozens of passing rows before the one that failed.
func (h *harness) report(schema, noun string) verdict {
	total := h.pass + h.fail
	code := 0
	if h.pass == 0 || h.fail > 0 {
		code = 1
	}
	v := verdict{
		code: code,
		json: fmt.Sprintf(`{"schema":"%s","total":%d,"passed":%d,"failed":%d}`, schema, total, h.pass, h.fail),
	}

	// ⚠ The shell half compares its row list against its counters, because there
	// a caller's own variable once overwrote the accumulator. Here both move in
	// the same two methods, so the comparison cannot fire; it is kept so the two
	// reports still describe themselves the same way.
	if len(h.rows) != total {
		v.code = 1
		v.stderr = fmt.Sprintf("%s: %d rows recorded, %d counted; the report does not describe itself\n",
			h.me, len(h.rows), total)
		return v
	}

	var b strings.Builder
	b.WriteString("\n")
	for _, r := range h.rows {
		fmt.Fprintf(&b, "  %s\n", r)
	}
	b.WriteString("\n")
	if h.fail > 0 {
		b.WriteString("the case(s) that failed:\n")
		for _, r := range h.rows {
			if strings.Contains(r, "❌") {
				fmt.Fprintf(&b, "  %s\n", r)
			}
		}
	}
	fmt.Fprintf(&b, "%d %s: %d passed, %d failed\n", total, noun, h.pass, h.fail)
	switch {
	case h.pass == 0:
		b.WriteString("❌ NOTHING RAN. Zero cases passed, so this is red whatever else it says.\n")
	case h.fail > 0:
		b.WriteString("❌ a guard did not refuse its defect.\n")
	default:
		b.WriteString("✅ every planted defect was refused, and the clean tree was not.\n")
	}
	v.text = b.String()
	return v
}

// needTools is store_require: every tool named when absent, and COULD NOT RUN
// rather than a refusal, because a machine without a tool has not failed a rule.
func needTools(tools ...string) error {
	for _, t := range tools {
		if _, err := exec.LookPath(t); err != nil {
			return errCannotRun(t + " not found")
		}
	}
	return nil
}

// cargoArtifact is the part of one line of `cargo build --message-format=json`
// this reads.
type cargoArtifact struct {
	Reason string `json:"reason"`
	Target struct {
		Name string   `json:"name"`
		Kind []string `json:"kind"`
	} `json:"target"`
	Executable *string `json:"executable"`
}

// exampleBinary builds one example and answers the path CARGO reports for it.
//
// ⛔ THE PATH IS ASKED FOR, NEVER COMPOSED. The shell half composed it out of
// CARGO_TARGET_DIR or the tree's own `target`, and that composition was a real
// hole once: with the variable set, the binary landed elsewhere and five provers
// exited 2 together. Cargo already names every artifact it builds, including an
// up-to-date one, so reading that answer honours the variable, `--target-dir`,
// a configured target directory and Windows' `.exe` without a second rule for
// any of them.
//
// ⛔ And it checks the binary is there afterwards. A build that exits 0 having
// produced nothing is the step that succeeds without doing what it was asked.
func exampleBinary(root, example, pkg string) (string, error) {
	if err := needTools("cargo"); err != nil {
		return "", err
	}
	cmd := exec.Command("cargo", "build", "--manifest-path", filepath.Join(root, "Cargo.toml"),
		"-p", pkg, "--locked", "--example", example, "--message-format=json")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", errCannotRun(fmt.Sprintf("cannot build the %s example", example))
	}

	bin := ""
	sc := bufio.NewScanner(&out)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var a cargoArtifact
		if json.Unmarshal(sc.Bytes(), &a) != nil || a.Reason != "compiler-artifact" {
			continue
		}
		if a.Target.Name != example || a.Executable == nil {
			continue
		}
		for _, k := range a.Target.Kind {
			if k == "example" {
				bin = *a.Executable
			}
		}
	}
	if bin == "" {
		return "", errCannotRun(fmt.Sprintf("cargo built the %s example and named no executable", example))
	}
	st, err := os.Stat(bin)
	if err != nil || st.IsDir() || (runtime.GOOS != "windows" && st.Mode()&0o111 == 0) {
		return "", errCannotRun(fmt.Sprintf("%s is not executable after a successful build", bin))
	}
	return bin, nil
}

// scratch is store_workdir: a directory of this run's own, which the caller
// removes.
func scratch(tag string) (string, error) {
	d, err := os.MkdirTemp("", "."+tag+".")
	if err != nil {
		return "", errCannotRun("cannot make a scratch directory")
	}
	return d, nil
}

// runTo runs a program with its stdout and stderr each sent to a FILE, and
// answers the exit code of that process and nothing else.
//
// ⛔ A FILE AND NOT A PIPE. A command substitution ends on its pipe's end of
// file rather than on its child's exit, so one process a subject leaves behind
// holds a pipe open forever; a file has an end. docs/conventions/shell.md 9.
//
// ⚠ An exit code of -1 means the program did not start, which the shell half
// cannot tell apart from a refusal either: both halves report the code they got.
func runTo(outPath, errPath, prog string, args ...string) int {
	o, err := os.Create(outPath)
	if err != nil {
		return -1
	}
	defer o.Close()
	e, err := os.Create(errPath)
	if err != nil {
		return -1
	}
	defer e.Close()
	cmd := exec.Command(prog, args...)
	cmd.Stdout, cmd.Stderr = o, e
	if err := cmd.Run(); err != nil {
		if x, ok := err.(*exec.ExitError); ok {
			return x.ExitCode()
		}
		return -1
	}
	return 0
}

// fileLines reads a file into lines the way `grep` and `head` see it; a missing
// file reads as empty, which is what those tools print for one.
func fileLines(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range splitLines(b) {
		out = append(out, string(l))
	}
	return out
}

// headJoin is `head -n N FILE | tr '\n' ' '`: the first lines, each followed by
// a space.
func headJoin(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[:n]
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteString(" ")
	}
	return b.String()
}

// tailJoin is `tail -n N FILE | tr '\n' ' '`.
func tailJoin(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return headJoin(lines, n)
}

// anyContains is `grep -q -F -e NEEDLE`: a fixed string, anywhere in any line.
func anyContains(lines []string, needle string) bool {
	for _, l := range lines {
		if strings.Contains(l, needle) {
			return true
		}
	}
	return false
}

// diffCount is the number of `<` and `>` lines `diff A B` prints: the lines only
// A has plus the lines only B has, over a longest common subsequence.
//
// ⚠ THE SHELL HALF ASKS `diff` AND ITS TWIN ASKS `Compare-Object -SyncWindow 0`,
// and those two are different questions that happen to agree on the one shape a
// harness here compares - two runs differing on one line. A positional
// comparison counts a shifted tail line by line where `diff` counts one
// insertion; this answers `diff`'s question, because that half is the one whose
// argument each case carries.
func diffCount(a, b []string) int {
	n, m := len(a), len(b)
	prev := make([]int, m+1)
	cur := make([]int, m+1)
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			switch {
			case a[i-1] == b[j-1]:
				cur[j] = prev[j-1] + 1
			case prev[j] >= cur[j-1]:
				cur[j] = prev[j]
			default:
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
	}
	lcs := prev[m]
	return (n - lcs) + (m - lcs)
}

// replaceOnce is store-lib's `replace_once`: a LITERAL that occurs exactly once
// is replaced, the file is rewritten through a temporary file beside it, and the
// edit is refused unless the bytes moved.
//
// ⛔ EXACTLY ONCE, OR NOT AT ALL. A literal that matches twice edits something
// other than what the case names, and one that matches nothing edits nothing
// while the case still reports a guard that failed to fire.
//
// ⚠ SINGLE-LINE LITERALS ONLY, refused rather than counted, for the shell half's
// reason: its `grep -F` split a multi-line pattern into alternatives and
// miscounted three plants. Nothing here would miscount one, and the refusal is
// kept anyway so the two halves plant the same set of literals.
func replaceOnce(path, old, repl string) bool {
	if strings.Contains(old, "\n") || old == "" {
		return false
	}
	before, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if strings.Count(string(before), old) != 1 {
		return false
	}
	after := []byte(strings.Replace(string(before), old, repl, 1))
	tmp := path + ".replace-once"
	if os.WriteFile(tmp, after, 0o644) != nil {
		return false
	}
	if os.Rename(tmp, path) != nil {
		_ = os.Remove(tmp)
		return false
	}
	return sha256.Sum256(before) != sha256.Sum256(after)
}

// probeGuards is `store_probe_guards`: the six self-guards every harness that
// plants defects owes, over a file the caller hands it and two scratch files of
// its own.
//
// ⭐ A probe's guard is a guard like any other, and this project has been burned
// by an unverified plant three times.
func (h *harness) probeGuards(path, present, ambiguous string) {
	if replaceOnce(path, "a literal this file does not carry", "x") {
		h.bad("probe    an absent literal was reported as planted")
	} else {
		h.ok("probe    an absent literal is refused")
	}
	if replaceOnce(path, ambiguous, "x") {
		h.bad("probe    an ambiguous literal was reported as planted")
	} else {
		h.ok("probe    an ambiguous literal is refused")
	}
	if replaceOnce(path, present, present) {
		h.bad("probe    a no-op edit was reported as planted")
	} else {
		h.ok("probe    a no-op edit is refused")
	}
	if replaceOnce(path, present+"\n", "x") {
		h.bad("probe    a multi-line literal was reported as planted")
	} else {
		h.ok("probe    a multi-line literal is refused")
	}

	// ⛔ IN A SCRATCH DIRECTORY OF ITS OWN, NEVER BESIDE THE CALLER'S FILE, which
	// may be a tracked path in the working tree.
	dir, err := os.MkdirTemp("", ".storeprobe.")
	if err != nil {
		h.bad("probe    a scratch directory for the plant probes could not be made")
		return
	}
	defer os.RemoveAll(dir)

	// The literal `a.b` occurs once in `axb then a.b`, and a REGEX would match
	// `axb` first. It asserts WHERE the edit landed, not that one happened.
	meta := filepath.Join(dir, "metachar")
	_ = os.WriteFile(meta, []byte("axb then a.b\n"), 0o644)
	if replaceOnce(meta, "a.b", "PLANTED") && strings.TrimRight(readString(meta), "\n") == "axb then PLANTED" {
		h.ok("probe    a literal carrying a regex metacharacter plants where it occurs")
	} else {
		h.bad("probe    a metacharacter literal planted [" + strings.TrimRight(readString(meta), "\n") + "]")
	}

	// A `/` ended the shell half's old `sed` expression, so every literal naming a
	// path was a case that quietly never ran.
	slash := filepath.Join(dir, "slash")
	_ = os.WriteFile(slash, []byte("keep a/b here\n"), 0o644)
	if replaceOnce(slash, "a/b", "PLANTED") && strings.TrimRight(readString(slash), "\n") == "keep PLANTED here" {
		h.ok("probe    a literal carrying a path separator can be planted")
	} else {
		h.bad("probe    a literal carrying a slash planted [" + strings.TrimRight(readString(slash), "\n") + "]")
	}
}

func readString(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}
