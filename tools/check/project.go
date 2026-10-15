// project.go - this repository's own invariants, as one check.
//
// ⛔ THE BIG ONE, AND IT IS THIRTY REFUSALS RATHER THAN ONE.
// `check-project` is 997 lines of `sh` with a hand-written PowerShell twin
// beside it, and the twin is where `CI-10`'s remaining wall clock sits. Every
// rule below is the rule its shell half stated, over the same inputs, with the
// same message: a port that changed one verdict while getting faster would be
// trading the thing being measured for the measurement.
//
// ⚠ THE COUNT IS THE ONE THAT CAN BE GREPPED. "Rules" is a judgement - the
// TODO bookkeeping is one subject and eight separate refusals - and a count in
// prose is a value in two places with nothing comparing them. `f.say(` here,
// `say_fail "` in the shell half and `failures.Add(` in the twin each appeared
// twenty-nine times when both halves were deleted. The thirtieth, the effort
// inventory, was written here on 2026-10-11 and has no half to compare against.
//
// ⚠ THE PARSERS ARE TRANSCRIBED, NOT REDESIGNED. Several of them look wrong at
// first reading - a prefix stripped by its literal rather than its length, an
// indent measured in SPACES only, a `grep` that loses its path prefix over a
// single file - and each of those shapes is a defect this repository already
// paid for once. `docs/conventions/shell.md` and the comments in the shell half
// carry the reasons; changing one here would be re-introducing it in a language
// where nobody would look.
//
// ⛔ A PAIR LEAVES `check-twins` ONE WAY ONLY:
// `check-bitcheck.sh --compare` ran every planted case against this
// implementation AND both halves, and any difference in the exit code or in the
// `--json` line was a failure. It ran before each half was deleted; no half is
// left, and `bit-check check-bitcheck` keeps the plants.
//
// Ported from scripts/common/check-project.sh and its PowerShell twin.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// -- the failure ledger --------------------------------------------------------

// projectFails is `say_fail`: it counts and it records, and the count is what
// the JSON line carries.
type projectFails struct {
	n    int
	body strings.Builder
}

func (f *projectFails) say(msg string) {
	f.n++
	fmt.Fprintf(&f.body, "FAIL: %s\n", msg)
}

// -- shelling out to git -------------------------------------------------------

// gitLines runs one `git ls-files` and returns its lines.
//
// ⛔ ASKED OF git, NEVER RE-DERIVED, for `repo.go`'s reason: a second answer to
// what is in the tree drifts, and it drifts towards staying green.
func gitLines(args ...string) []string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// gitScope is `{ git ls-files P; git ls-files --others --exclude-standard P; } |
// sort -u`, which is what every rule below that reads a language's files uses.
//
// ⛔ TRACKED ALONE IS NOT ENOUGH: a file that has never been staged is exactly
// when a new one is likeliest to carry a defect.
func gitScope(pathspecs ...string) []string {
	seen := map[string]bool{}
	for _, args := range [][]string{
		append([]string{"ls-files"}, pathspecs...),
		append([]string{"ls-files", "--others", "--exclude-standard"}, pathspecs...),
	} {
		for _, f := range gitLines(args...) {
			seen[f] = true
		}
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

func readLines(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := splitLines(raw)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = string(l)
	}
	return out
}

// trimSpace is awk's `gsub(/^ +| +$/, "")`: SPACES, not every blank.
//
// ⚠ A tab in a markdown table cell would survive this in awk and must survive it
// here, or a cell nobody can see changes would compare differently in one
// implementation.
func trimSpaces(s string) string { return strings.Trim(s, " ") }

// blankSet is awk's `[[:space:]]` in the C locale, which is WIDER than a space
// and a tab.
//
// ⚠ A `.ps1` in this tree keeps CRLF, so a rule that trimmed only ` \t` would
// read a blank line as non-blank wherever a carriage return survives.
const blankSet = " \t\n\v\f\r"

func trimBlankLeft(s string) string { return strings.TrimLeft(s, blankSet) }

// -- the check -----------------------------------------------------------------

var (
	projEntryRe     = regexp.MustCompile(`^\| (FOUND|SCHEMA|OBS|ACQ|CLIENT|ENGINE|CORPUS|LIB|PUB|CI|DOC)-[0-9][0-9] `)
	projBodyIDRe    = regexp.MustCompile(`^## (FOUND|SCHEMA|OBS|ACQ|CLIENT|ENGINE|CORPUS|LIB|PUB|CI|DOC)-[0-9][0-9]:`)
	projStatusRe    = regexp.MustCompile(`^(OPEN|IN_PROGRESS|BLOCKED|DONE)$`)
	projMatrixRowRe = regexp.MustCompile("^\\| `[a-z0-9-]+` \\|")
)

func checkProject(r *repo) (verdict, error) {
	f := &projectFails{}

	projectFiles(f)
	projectTargets(f)
	rows, open, inProgress, blocked, done := projectTodo(f)
	projectPython(f)
	projectPins(f)
	projectArtifacts(f)
	projectLineEndings(f)
	projectShfmt(f)
	projectSetU(f)
	projectTargetDir(f)
	projectLockfile(f)
	projectAcceptance(f)
	projectGitDependency(f)
	projectPowerShell(f)
	projectWorkflowPwsh(f)

	v := verdict{
		json: fmt.Sprintf(
			`{"schema":"check-project/2","failures":%d,"todo_entries":%d,"open":%d,"in_progress":%d,"blocked":%d,"done":%d}`,
			f.n, rows, open, inProgress, blocked, done),
	}
	if f.n > 0 {
		v.code = 1
		v.text = f.body.String()
		return v, nil
	}
	v.text = fmt.Sprintf(
		"bit-ids project invariants pass (%d entries; %d open; %d in progress; %d blocked; %d done)\n",
		rows, open, inProgress, blocked, done)
	return v, nil
}

// -- 1. the files a reader is promised -----------------------------------------

func projectFiles(f *projectFails) {
	for _, path := range []string{
		"README.md", "LICENSE", "Cargo.toml", "Cargo.lock", "catalogue/clients.toml",
		"TODO/INDEX.md", "TODO/PROGRESS.md", "TODO/SUMMARY.md", "docs/AGENTS.md",
		"docs/reference-sweeps/bit-cli.md", ".github/workflows/ci.yml",
	} {
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			f.say("missing " + path)
		}
	}
}

// -- 2. the target set, derived in both directions -----------------------------
//
// ⛔ A RESULT TOO SMALL TO BE REAL IS REFUSED. Two empty sets agree perfectly,
// so a parser that stopped matching would report a pinned matrix over nothing.

func projectTargets(f *projectFails) {
	catalogue := map[string]bool{}
	for _, line := range readLines("catalogue/clients.toml") {
		if !strings.HasPrefix(line, `id = "`) {
			continue
		}
		// awk -F'"': the id is the second field, whatever follows it.
		parts := strings.Split(line, `"`)
		if len(parts) >= 2 {
			catalogue[parts[1]] = true
		}
	}
	matrix := map[string]bool{}
	for _, line := range readLines("docs/client-matrix.md") {
		if !projMatrixRowRe.MatchString(line) {
			continue
		}
		parts := strings.Split(line, "`")
		if len(parts) >= 2 {
			matrix[parts[1]] = true
		}
	}
	if len(catalogue) < 10 || len(matrix) < 10 {
		f.say(fmt.Sprintf("target sets too small to be real: catalogue %d, matrix %d",
			len(catalogue), len(matrix)))
		return
	}
	for _, id := range sortedKeys(catalogue) {
		if !matrix[id] {
			f.say("client matrix lacks target " + id + ", which the catalogue carries")
		}
	}
	for _, id := range sortedKeys(matrix) {
		if !catalogue[id] {
			f.say("the client matrix names " + id + ", which the catalogue does not carry")
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// -- 3-8. the TODO bookkeeping -------------------------------------------------

var (
	projPriorityLineRe = regexp.MustCompile(`^Priority: `)
	projStripTailRe    = regexp.MustCompile(` \|.*$`)
	projAfterEffortRe  = regexp.MustCompile(`^.*Effort: `)
	projAfterStatusRe  = regexp.MustCompile(`^.*Status: `)
)

func projectTodo(f *projectFails) (rows, open, inProgress, blocked, done int) {
	// ⛔ EVERY FIELD THE TWO PLACES BOTH CARRY IS COMPARED, not the status alone:
	// a priority that disagreed with its own entry lived here from the commit
	// that created both.
	var index []string
	counts := map[string]int{}
	byPrefix := map[string]map[string]int{}
	totalByPrefix := map[string]int{}
	byPriority := map[string]map[string]int{}
	totalByPriority := map[string]int{}
	byEffort := map[string]int{}
	var invalid []string
	ids := map[string]int{}

	for _, line := range readLines("TODO/INDEX.md") {
		if !projEntryRe.MatchString(line) {
			continue
		}
		cell := strings.Split(line, "|")
		if len(cell) < 5 {
			continue
		}
		id := trimSpaces(cell[1])
		priority := trimSpaces(cell[2])
		effort := trimSpaces(cell[3])
		status := trimSpaces(cell[4])
		index = append(index, id+"|"+priority+"|"+effort+"|"+status)
		counts[status]++
		ids[id]++
		if !projStatusRe.MatchString(status) {
			invalid = append(invalid, id)
		}
		prefix := id
		if at := strings.LastIndex(prefix, "-"); at > 0 {
			prefix = prefix[:at]
		}
		if byPrefix[prefix] == nil {
			byPrefix[prefix] = map[string]int{}
		}
		byPrefix[prefix][status]++
		totalByPrefix[prefix]++
		if byPriority[priority] == nil {
			byPriority[priority] = map[string]int{}
		}
		byPriority[priority][status]++
		totalByPriority[priority]++
		byEffort[effort]++
	}

	// ⛔ THE GLOB, NOT `git ls-files`. Both halves list the DIRECTORY, so a new
	// entry file that has never been staged is read by them - and it is exactly
	// when an entry is likeliest to disagree with its index row.
	//
	// ⚠ AND THE TWO HALVES DIFFER HERE, which `check-twins` could never see: awk
	// carries `id` across files and the twin resets it per file, so an entry
	// heading left unterminated at the end of one file would pair with the next
	// file's first `Priority:` line in one implementation and not the other. This
	// resets, with the twin, because pairing across files is an answer nobody
	// wrote down. No file in this tree ends that way, which is why the comparison
	// is blind to it.
	var bodies []string
	for _, file := range projTodoFiles() {
		id := ""
		for _, line := range readLines(file) {
			if projBodyIDRe.MatchString(line) {
				// awk's $2 over the default field separator, with the colon cut.
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					id = strings.TrimSuffix(fields[1], ":")
				}
				continue
			}
			if id == "" || !projPriorityLineRe.MatchString(line) {
				continue
			}
			priority := projStripTailRe.ReplaceAllString(strings.TrimPrefix(line, "Priority: "), "")
			effort := projStripTailRe.ReplaceAllString(projAfterEffortRe.ReplaceAllString(line, ""), "")
			status := projAfterStatusRe.ReplaceAllString(line, "")
			bodies = append(bodies, id+"|"+priority+"|"+effort+"|"+status)
			id = ""
		}
	}

	sort.Strings(index)
	sort.Strings(bodies)
	rows = len(index)
	open = counts["OPEN"]
	inProgress = counts["IN_PROGRESS"]
	blocked = counts["BLOCKED"]
	done = counts["DONE"]

	if len(invalid) > 0 {
		sort.Strings(invalid)
		f.say("TODO index has invalid statuses: " + strings.Join(invalid, "\n"))
	}
	if len(bodies) != rows {
		f.say("TODO body count does not match index count")
	}
	if strings.Join(index, "\n") != strings.Join(bodies, "\n") {
		f.say("TODO IDs, priorities, efforts or statuses disagree between index and category bodies")
	}
	for _, n := range ids {
		if n > 1 {
			f.say("TODO index contains duplicate IDs")
			break
		}
	}

	for _, file := range []string{"TODO/INDEX.md", "TODO/PROGRESS.md"} {
		for _, pair := range []struct {
			key    string
			actual int
		}{
			{"Total", rows}, {"Open", open}, {"In progress", inProgress},
			{"Blocked", blocked}, {"Done", done},
		} {
			declared := projDeclared(file, pair.key)
			if declared != strconv.Itoa(pair.actual) {
				f.say(fmt.Sprintf("%s declares %s=%s, computed %d", file, pair.key, declared, pair.actual))
			}
		}
	}

	projectSummaryTotal(f, open, inProgress, blocked, done, rows)
	projectSummaryRows(f, byPrefix, totalByPrefix)
	projectPriorityTable(f, byPriority, totalByPriority)
	projectEffortInventory(f, byEffort)
	return rows, open, inProgress, blocked, done
}

// projectEffortInventory compares `TODO/SUMMARY.md`'s effort sentence against
// the index.
//
// ⛔ IT DRIFTED WHILE IT WAS ONLY A SENTENCE. It said 49 L and 0 XL for as long
// as the index carried 51 and 1, found by a reading on 2026-10-11 rather than by
// any check. ⚠ The four sizes are named because the SENTENCE is the thing being
// checked: a size it omits is the defect, so the expected set cannot be derived
// from it. A size the index carries that is not one of the four is refused too,
// or a fifth size would be counted nowhere and the sentence would still agree.
var projEffortLineRe = regexp.MustCompile(`^Effort inventory: ([0-9]+) S, ([0-9]+) M, ([0-9]+) L, ([0-9]+) XL\.`)

func projectEffortInventory(f *projectFails, byEffort map[string]int) {
	var seen []string
	for _, line := range readLines("TODO/SUMMARY.md") {
		if m := projEffortLineRe.FindStringSubmatch(line); m != nil {
			seen = append(seen, strings.Join(m[1:], " "))
		}
	}
	sizes := []string{"S", "M", "L", "XL"}
	want := make([]string, len(sizes))
	known := 0
	for i, size := range sizes {
		want[i] = strconv.Itoa(byEffort[size])
		known += byEffort[size]
	}
	var bad []string
	if got := strings.Join(seen, "\n"); got != strings.Join(want, " ") {
		bad = append(bad, fmt.Sprintf("declares %q, computed %q", got, strings.Join(want, " ")))
	}
	total := 0
	for _, n := range byEffort {
		total += n
	}
	if known != total {
		bad = append(bad, fmt.Sprintf("%d index row(s) carry an effort outside S, M, L and XL", total-known))
	}
	if len(bad) > 0 {
		f.say("TODO/SUMMARY.md effort inventory disagrees with the index: " + strings.Join(bad, "; "))
	}
}

// projDeclared is `awk -F ': ' '$1 == key { print $2; exit }'`.
//
// ⚠ THE SEPARATOR IS THE TWO-CHARACTER STRING, so `In progress: 1` has the key
// a reader sees and a second `: ` later in a line would start a third field
// rather than extend the second.
func projDeclared(file, key string) string {
	for _, line := range readLines(file) {
		parts := strings.Split(line, ": ")
		if len(parts) >= 2 && parts[0] == key {
			return parts[1]
		}
	}
	return ""
}

func projectSummaryTotal(f *projectFails, open, inProgress, blocked, done, rows int) {
	// ⚠ EVERY MATCH, JOINED, because the half this replaces captures awk's whole
	// output: a file with two Total rows compares as two lines and is refused,
	// where keeping only the last would accept it if the last one happened to be
	// right.
	var seen []string
	for _, line := range readLines("TODO/SUMMARY.md") {
		if !strings.HasPrefix(line, "| Total |") {
			continue
		}
		cell := strings.Split(line, "|")
		if len(cell) < 9 {
			continue
		}
		seen = append(seen, strings.Join([]string{
			trimSpaces(cell[3]), trimSpaces(cell[4]), trimSpaces(cell[5]),
			trimSpaces(cell[6]), trimSpaces(cell[7]),
		}, "|"))
	}
	got := strings.Join(seen, "\n")
	want := fmt.Sprintf("%d|%d|%d|%d|%d", open, inProgress, blocked, done, rows)
	if got != want {
		f.say(fmt.Sprintf("TODO/SUMMARY.md total is %s, computed %s", got, want))
	}
}

// projectSummaryRows compares every CATEGORY row, in both directions.
//
// ⛔ THE TOTAL ROW WAS THE ONLY ROW CHECKED AND IT IS ONE OF TWELVE. The mapping
// from a category to the identifiers it counts is declared in `TODO/SUMMARY.md`
// itself; a table of names here would be a second copy.
//
// ⚠ A DATA ROW IS RECOGNISED BY ITS SHAPE, NOT BY ITS CASE. The first version
// matched an upper-case first letter and the PowerShell twin let the header
// through, because its match is case-insensitive and awk's is not.
var projNumberRe = regexp.MustCompile(`^[0-9]+$`)

func projectSummaryRows(f *projectFails, byPrefix map[string]map[string]int, totalByPrefix map[string]int) {
	var bad []string
	declared := map[string]bool{}
	seen := map[string]bool{}
	for _, line := range readLines("TODO/SUMMARY.md") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cell := strings.Split(line, "|")
		if len(cell) < 9 {
			continue
		}
		prefix := strings.ReplaceAll(trimSpaces(cell[2]), "`", "")
		if prefix == "" {
			continue
		}
		numeric := true
		for i := 3; i <= 7; i++ {
			if !projNumberRe.MatchString(trimSpaces(cell[i])) {
				numeric = false
				break
			}
		}
		if !numeric {
			continue
		}
		if seen[prefix] {
			bad = append(bad, "duplicate row for "+prefix)
			continue
		}
		seen[prefix] = true
		declared[prefix] = true
		counts := byPrefix[prefix]
		if atoi(trimSpaces(cell[3])) != counts["OPEN"] ||
			atoi(trimSpaces(cell[4])) != counts["IN_PROGRESS"] ||
			atoi(trimSpaces(cell[5])) != counts["BLOCKED"] ||
			atoi(trimSpaces(cell[6])) != counts["DONE"] ||
			atoi(trimSpaces(cell[7])) != totalByPrefix[prefix] {
			bad = append(bad, prefix)
		}
	}
	for _, p := range sortedKeysOfCounts(byPrefix) {
		if !declared[p] {
			bad = append(bad, p+" has no row")
		}
	}
	for _, p := range sortedKeys(declared) {
		if byPrefix[p] == nil {
			bad = append(bad, p+" names nothing in the index")
		}
	}
	if len(bad) == 0 {
		return
	}
	sort.Strings(bad)
	f.say("TODO/SUMMARY.md category rows disagree: " + strings.Join(bad, " "))
}

func sortedKeysOfCounts(m map[string]map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// projectPriorityTable holds the index's own P0/P1/P2 table against its rows.
//
// ⚠ The three priorities are named here because the TABLE is the thing being
// checked: a priority the table omits entirely is the defect, so the expected
// set cannot be derived from the table.
var projPriorityRowRe = regexp.MustCompile(`^\| P[0-2] \|`)

func projectPriorityTable(f *projectFails, byPriority map[string]map[string]int, totalByPriority map[string]int) {
	type row struct {
		open, inProgress, blocked, done, total int
	}
	declared := map[string]row{}
	for _, line := range readLines("TODO/INDEX.md") {
		if !projPriorityRowRe.MatchString(line) {
			continue
		}
		cell := strings.Split(line, "|")
		if len(cell) < 8 {
			continue
		}
		declared[trimSpaces(cell[1])] = row{
			open:       atoi(trimSpaces(cell[2])),
			inProgress: atoi(trimSpaces(cell[3])),
			blocked:    atoi(trimSpaces(cell[4])),
			done:       atoi(trimSpaces(cell[5])),
			total:      atoi(trimSpaces(cell[6])),
		}
	}
	var bad []string
	for _, p := range []string{"P0", "P1", "P2"} {
		d, ok := declared[p]
		counts := byPriority[p]
		if !ok ||
			d.open != counts["OPEN"] ||
			d.inProgress != counts["IN_PROGRESS"] ||
			d.blocked != counts["BLOCKED"] ||
			d.done != counts["DONE"] ||
			d.total != totalByPriority[p] {
			bad = append(bad, p)
		}
	}
	if len(bad) > 0 {
		f.say("TODO priority table disagrees for: " + strings.Join(bad, "\n"))
	}
}

// -- 9. Python exists only with an approved exception --------------------------
//
// ⛔ THE RULE ASKS FOR THE DOCUMENTATION, WHICH IS THE PART THAT MATTERS. Three
// conditions: the file declares an entry, the entry is one `TODO/INDEX.md`
// really carries, and THAT ENTRY'S OWN SECTION mentions the file's path - so the
// argument lives in the record rather than only in the file that benefits.

var projPyMarkerRe = regexp.MustCompile(`bit-ids:python-exception=[A-Z]+-[0-9]+`)

func projectPython(f *projectFails) {
	var out strings.Builder
	for _, py := range gitScope("*.py") {
		marker := ""
		for _, line := range readLines(py) {
			if m := projPyMarkerRe.FindString(line); m != "" {
				marker = strings.TrimPrefix(m, "bit-ids:python-exception=")
				break
			}
		}
		owner := ""
		for _, file := range projTodoFiles() {
			if containsLinePrefix(file, "## "+marker+":") {
				owner = file
				break
			}
		}
		switch {
		case marker == "":
			fmt.Fprintf(&out, " %s declares no bit-ids:python-exception=<ENTRY>;", py)
		case !containsLinePrefix("TODO/INDEX.md", "| "+marker+" |"):
			fmt.Fprintf(&out, " %s declares python-exception=%s, which is not an entry in TODO/INDEX.md;", py, marker)
		case owner == "":
			fmt.Fprintf(&out, " %s declares python-exception=%s, which no file under TODO/ carries an entry for;", py, marker)
		case !sectionMentions(owner, "## "+marker+":", py):
			fmt.Fprintf(&out, " %s is approved by %s and that entry never mentions it;", py, marker)
		}
	}
	if out.Len() > 0 {
		f.say("Python exists without an approved exception:" + out.String())
	}
}

// projTodoFiles is the `TODO/*.md` shell glob: on disk, in lexicographic order.
//
// ⚠ A GLOB RATHER THAN `git ls-files`, matching the half this replaces, so an
// entry file that is new and unstaged still owns its marker.
func projTodoFiles() []string {
	entries, err := os.ReadDir("TODO")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, "TODO/"+e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func containsLinePrefix(file, prefix string) bool {
	if prefix == "## :" || prefix == "| |" {
		return false
	}
	for _, line := range readLines(file) {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// sectionMentions reads ONE entry's section, which is the smallest unit that can
// be said to have made an argument.
//
// ⛔ NOT THE WHOLE FILE. Asking only that the file mention the path let a plant
// naming a real but unrelated entry survive twice.
func sectionMentions(file, heading, needle string) bool {
	inside := false
	for _, line := range readLines(file) {
		if strings.HasPrefix(line, heading) {
			inside = true
			continue
		}
		if strings.HasPrefix(line, "## ") {
			inside = false
		}
		if inside && strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

// -- 10. every action is pinned to an immutable form ---------------------------
//
// ⛔ AN ALLOWLIST OF IMMUTABLE FORMS, NOT A DENYLIST OF FLOATING ONES, so a form
// nobody thought of fails closed. ⚠ The version comment is part of the pin:
// `check-remote-items` resolves it against the tag it names, and a pin without
// one is a pin that check never examines.

var (
	projUsesRe    = regexp.MustCompile(`^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]`)
	projUsesCutRe = regexp.MustCompile(`^[^:]*uses:[[:space:]]*`)
	projCommentRe = regexp.MustCompile(`[[:space:]]+#`)
	projHasWordRe = regexp.MustCompile(`#[[:space:]]*[^[:space:]]`)
	projHexRe     = regexp.MustCompile(`^[0-9a-f]*$`)
)

// projWorkflowFiles is the scope rules 10, 11 and 22-24 share, deliberately and
// identically: a composite action carries its own steps and runs with the same
// permissions, and a workflow may be `.yaml`.
func projWorkflowFiles() []string {
	var out []string
	for _, dir := range []string{".github/workflows"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
				names = append(names, dir+"/"+e.Name())
			}
		}
		sort.Strings(names)
		out = append(out, names...)
	}
	entries, err := os.ReadDir(".github/actions")
	if err != nil {
		return out
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		for _, leaf := range []string{"action.yml", "action.yaml"} {
			p := ".github/actions/" + e.Name() + "/" + leaf
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				names = append(names, p)
			}
		}
	}
	sort.Strings(names)
	return append(out, names...)
}

func projectPins(f *projectFails) {
	var out strings.Builder
	for _, wf := range projWorkflowFiles() {
		var per strings.Builder
		for i, line := range readLines(wf) {
			nr := i + 1
			if !projUsesRe.MatchString(line) {
				continue
			}
			ref := projUsesCutRe.ReplaceAllString(line, "")
			comment := ""
			if loc := projCommentRe.FindStringIndex(ref); loc != nil {
				comment = ref[loc[0]:]
				ref = ref[:loc[0]]
			}
			ref = strings.TrimRight(ref, " \t")
			ref = trimOneQuote(ref)
			// A local action is this repository, reviewed with everything else.
			if strings.HasPrefix(ref, "./") {
				continue
			}
			at := strings.LastIndex(ref, "@")
			if at < 0 {
				fmt.Fprintf(&per, "%s:%d carries no ref at all: %s\n", wf, nr, ref)
				continue
			}
			pinned := ref[at+1:]
			if strings.HasPrefix(ref, "docker://") {
				if !strings.HasPrefix(pinned, "sha256:") {
					fmt.Fprintf(&per, "%s:%d container is not pinned to a digest: %s\n", wf, nr, ref)
					continue
				}
				digest := pinned[len("sha256:"):]
				if len(digest) != 64 || !projHexRe.MatchString(digest) {
					fmt.Fprintf(&per, "%s:%d container digest is not a sha256: %s\n", wf, nr, ref)
				}
				continue
			}
			if len(pinned) != 40 || !projHexRe.MatchString(pinned) {
				fmt.Fprintf(&per, "%s:%d not pinned to a 40-character commit: %s\n", wf, nr, ref)
				continue
			}
			if !projHasWordRe.MatchString(comment) {
				fmt.Fprintf(&per, "%s:%d pin carries no version comment, so nothing can check it: %s\n", wf, nr, ref)
			}
		}
		// ⚠ A COMMAND SUBSTITUTION STRIPS TRAILING NEWLINES, so the halves
		// concatenate one file's findings onto the previous file's last line.
		// Reproduced rather than tidied: the count is the verdict, and the text
		// is what a reader of either implementation already sees.
		out.WriteString(strings.TrimRight(per.String(), "\n"))
	}
	if out.Len() > 0 {
		f.say("workflow action pin: " + out.String())
	}
}

// trimOneQuote is awk's `gsub(/^["']|["']$/, "")`: one at each end, not a pair.
func trimOneQuote(s string) string {
	if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
		s = s[1:]
	}
	if len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
		s = s[:len(s)-1]
	}
	return s
}

// -- 11. an artifact a workflow downloads is one some workflow uploads ---------
//
// ⛔ Two workflows joined by a name nobody compares are not joined at all, and
// the failure is a dispatch that dies on its first step.
//
// ⭐ NAMES ARE TEMPLATED, so the comparison is over a PATTERN: every `${{ ... }}`
// becomes `*` on both sides. ⚠ `pattern:` counts as well as `name:`, because
// `download-artifact` accepts either.

type projArtifact struct {
	kind    string
	where   string
	name    string
	pattern string
	marker  string
}

var (
	projNoProducerRe = regexp.MustCompile(`bit-ids:no-producer=[A-Z]+-[0-9]+`)
	projExprRe       = regexp.MustCompile(`\$\{\{[^}]*\}\}`)
	projArtUsesRe    = regexp.MustCompile(`uses:[ \t]*actions/(upload|download)-artifact@`)
	projWithRe       = regexp.MustCompile(`^(-[ \t]+)?with:[ \t]*$`)
	projDashLeadRe   = regexp.MustCompile(`^[ \t]*-[ \t]+`)
	projNameKeyRe    = regexp.MustCompile(`^(name|pattern):[ \t]`)
	projNameCutRe    = regexp.MustCompile(`^(name|pattern):[ \t]*`)
)

func indentOf(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}

func projectArtifactRows() []projArtifact {
	var rows []projArtifact
	for _, wf := range projWorkflowFiles() {
		kind, val, marker := "", "", "-"
		at := 0
		stepind, withind := -1, -1
		flush := func() {
			if kind != "" && val != "" {
				rows = append(rows, projArtifact{
					kind:    kind,
					where:   fmt.Sprintf("%s:%d", wf, at),
					name:    val,
					pattern: projExprRe.ReplaceAllString(val, "*"),
					marker:  marker,
				})
			}
			kind, val, marker, at, withind = "", "", "-", 0, -1
		}
		for i, raw := range readLines(wf) {
			nr := i + 1
			ind := indentOf(raw)
			line := raw[ind:]
			if m := projNoProducerRe.FindString(line); m != "" {
				marker = strings.TrimPrefix(m, "bit-ids:no-producer=")
				continue
			}
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if line == "-" || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "-\t") {
				if stepind < 0 || ind <= stepind {
					flush()
					stepind = ind
				}
			} else if stepind >= 0 && ind <= stepind {
				flush()
				stepind = -1
			}
			if projArtUsesRe.MatchString(line) {
				if strings.Contains(line, "upload-artifact") {
					kind = "upload"
				} else {
					kind = "download"
				}
				at = nr
			}
			if withind >= 0 && ind <= withind {
				withind = -1
			}
			if projWithRe.MatchString(line) {
				withind = ind
				// ⛔ THE COLUMN THAT MATTERS IS THE `with:` KEY, NOT THE LINE. In
				// `- with:` the dash is part of the indent, so the step's other
				// keys sit at the KEY's column rather than inside it.
				if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "-\t") {
					withind = len(projDashLeadRe.FindString(raw))
				}
				continue
			}
			if withind >= 0 && ind > withind && projNameKeyRe.MatchString(line) {
				v := projNameCutRe.ReplaceAllString(line, "")
				v = strings.TrimRight(v, " \t")
				val = trimOneQuote(v)
				if at == 0 {
					at = nr
				}
			}
		}
		flush()
	}
	return rows
}

func projectArtifacts(f *projectFails) {
	rows := projectArtifactRows()
	var uploads []string
	for _, row := range rows {
		if row.kind == "upload" {
			uploads = append(uploads, row.pattern)
		}
	}
	var out strings.Builder
	for _, row := range rows {
		if row.kind != "download" {
			continue
		}
		found := false
		for _, up := range uploads {
			if shellMatch(up, row.pattern) {
				found = true
				break
			}
		}
		switch {
		case found:
			if row.marker != "-" {
				fmt.Fprintf(&out, " %s declares no-producer=%s over [%s], which an upload-artifact in this tree now produces;",
					row.where, row.marker, row.name)
			}
		case row.marker == "-":
			fmt.Fprintf(&out, " %s downloads [%s], which no upload-artifact in this tree produces;",
				row.where, row.name)
		case !containsLinePrefix("TODO/INDEX.md", "| "+row.marker+" |"):
			fmt.Fprintf(&out, " %s declares no-producer=%s, which is not an entry in TODO/INDEX.md;",
				row.where, row.marker)
		}
	}
	if out.Len() > 0 {
		f.say("workflow artifact name:" + out.String())
	}
}

// shellMatch is `case "$subject" in $pattern)`: sh's own globbing, where `*`
// crosses every character including a slash.
//
// ⛔ `path.Match` IS NOT THE SAME RULE - its `*` stops at a separator - and a
// port that used it would answer differently for the first artifact name that
// carries one. The class is implemented rather than the instances this tree has.
func shellMatch(pattern, subject string) bool {
	return shellMatchAt(pattern, subject)
}

func shellMatchAt(p, s string) bool {
	for {
		if p == "" {
			return s == ""
		}
		switch p[0] {
		case '*':
			p = p[1:]
			if p == "" {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if shellMatchAt(p, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if s == "" {
				return false
			}
			p, s = p[1:], s[1:]
		case '[':
			if s == "" {
				return false
			}
			ok, rest, matched := shellMatchClass(p, s[0])
			if !ok {
				// An unterminated class is a literal bracket, as sh reads it.
				if s[0] != '[' {
					return false
				}
				p, s = p[1:], s[1:]
				continue
			}
			if !matched {
				return false
			}
			p, s = rest, s[1:]
		case '\\':
			if len(p) < 2 || s == "" || s[0] != p[1] {
				return false
			}
			p, s = p[2:], s[1:]
		default:
			if s == "" || s[0] != p[0] {
				return false
			}
			p, s = p[1:], s[1:]
		}
	}
}

// shellMatchClass reads one `[...]`, reporting whether it is terminated at all,
// what is left of the pattern, and whether the character is in it.
func shellMatchClass(p string, c byte) (ok bool, rest string, matched bool) {
	i := 1
	negate := false
	if i < len(p) && (p[i] == '!' || p[i] == '^') {
		negate = true
		i++
	}
	// A `]` first in the class is the character itself.
	first := true
	hit := false
	for i < len(p) {
		if p[i] == ']' && !first {
			if negate {
				return true, p[i+1:], !hit
			}
			return true, p[i+1:], hit
		}
		first = false
		if i+2 < len(p) && p[i+1] == '-' && p[i+2] != ']' {
			if c >= p[i] && c <= p[i+2] {
				hit = true
			}
			i += 3
			continue
		}
		if p[i] == c {
			hit = true
		}
		i++
	}
	return false, "", false
}

// -- 12. the working tree agrees with .gitattributes ---------------------------
//
// ⭐ GIT'S OWN ANSWER, NOT A SECOND TABLE: `git ls-files --eol` resolves the
// attribute per path the way a checkout will. ⚠ `w/none` is not a disagreement -
// a file with no line ending carries no evidence either way.

var projEolAttrRe = regexp.MustCompile(`eol=[a-z]+`)

func projectLineEndings(f *projectFails) {
	out, err := exec.Command("git", "ls-files", "--eol").Output()
	if err != nil {
		return
	}
	var found strings.Builder
	for _, line := range strings.Split(string(out), "\n") {
		tab := strings.Index(line, "\t")
		if tab < 0 {
			continue
		}
		head, path := line[:tab], line[tab+1:]
		attr := projEolAttrRe.FindString(head)
		if attr == "" {
			continue
		}
		attr = strings.TrimPrefix(attr, "eol=")
		// awk's split on / +/ collapses runs of spaces; field 2 is the working
		// tree's ending.
		fields := projSpaceRunSplitRe.Split(head, -1)
		if len(fields) < 2 {
			continue
		}
		w := strings.TrimPrefix(fields[1], "w/")
		if w == "none" {
			continue
		}
		if w != attr {
			fmt.Fprintf(&found, "%s is w/%s under eol=%s; ", path, w, attr)
		}
	}
	if found.Len() > 0 {
		f.say("line endings disagree with .gitattributes: " + found.String())
	}
}

// projSpaceRunSplitRe is awk's `split(s, a, / +/)`: runs of spaces separate, and
// a leading run produces an empty first field exactly as awk does.
var projSpaceRunSplitRe = regexp.MustCompile(` +`)

// -- 13. the shfmt pin is one version in every running place -------------------

var (
	projShfmtRefRe = regexp.MustCompile(`mvdan\.cc/sh/v3/cmd/shfmt@v[0-9]+\.[0-9]+\.[0-9]+`)
	projShfmtVarRe = regexp.MustCompile(`^SHFMT_VERSION=`)
)

func projectShfmt(f *projectFails) {
	pin := ""
	for _, line := range readLines("scripts/doctor/provision.sh") {
		if !projShfmtVarRe.MatchString(line) {
			continue
		}
		// awk -F=: the value is the field after the FIRST `=`, and a second one
		// would start a third field rather than extend the second.
		parts := strings.Split(line, "=")
		if len(parts) >= 2 {
			pin = parts[1]
		}
		break
	}
	if pin == "" {
		f.say("scripts/doctor/provision.sh declares no SHFMT_VERSION to compare against")
		return
	}
	var out strings.Builder
	for _, file := range gitLines("ls-files", ".github/workflows/*", ".github/actions/*", "scripts/*") {
		for i, line := range readLines(file) {
			for _, m := range projShfmtRefRe.FindAllString(line, -1) {
				at := strings.Index(m, "@v")
				version := m[at+2:]
				if version != pin {
					fmt.Fprintf(&out, "%s:%d:%s names v%s, not v%s; ", file, i+1, m[:at], version, pin)
				}
			}
		}
	}
	if out.Len() > 0 {
		f.say("shfmt version disagrees with the pin: " + out.String())
	}
}

// -- 14. a shell option is stated, not inherited -------------------------------
//
// ⭐ THE FIRST CODE LINE, NOT ANYWHERE IN THE FILE: a loose rule is spoofable,
// and this tree contains a heredoc whose body begins `set -u`.
//
// ⚠ `store-lib.sh` is exempt by name: it is SOURCED, so an option set there
// changes the caller's shell and stays changed.

func projectSetU(f *projectFails) {
	var out strings.Builder
	for _, sh := range gitLines("ls-files", "*.sh") {
		if sh == "scripts/corpus/store-lib.sh" {
			continue
		}
		first := ""
		for i, line := range readLines(sh) {
			if i == 0 && strings.HasPrefix(line, "#!") {
				continue
			}
			if strings.HasPrefix(trimBlankLeft(line), "#") {
				continue
			}
			if trimBlankLeft(line) == "" {
				continue
			}
			first = line
			break
		}
		if first != "set -u" {
			fmt.Fprintf(&out, " %s begins [%s];", sh, first)
		}
	}
	if out.Len() > 0 {
		f.say("a script does not state set -u as its first line:" + out.String())
	}
}

// -- 15. the cargo output directory is asked for, never composed ---------------

var projTargetPathRe = regexp.MustCompile(`/target[/}]|(debug|release)/examples`)

func projectTargetDir(f *projectFails) {
	var hits []string
	for _, file := range gitLines("ls-files", "*.sh", "*.ps1") {
		for i, line := range readLines(file) {
			if !projTargetPathRe.MatchString(line) {
				continue
			}
			row := fmt.Sprintf("%s:%d:%s", file, i+1, line)
			if strings.Contains(row, "CARGO_TARGET_DIR") {
				continue
			}
			hits = append(hits, row)
		}
	}
	if len(hits) > 0 {
		f.say("a cargo output path is composed without CARGO_TARGET_DIR: " + strings.Join(hits, " "))
	}
}

// -- 16. every dependency comes from the registry, with a checksum -------------

const projRegistry = "registry+https://github.com/rust-lang/crates.io-index"

func projectLockfile(f *projectFails) {
	var out strings.Builder
	name, source, checksum := "", "", false
	flush := func() {
		if name != "" && source != "" {
			if source != projRegistry {
				fmt.Fprintf(&out, "  %s is not from the crates.io registry: %s\n", name, source)
			} else if !checksum {
				fmt.Fprintf(&out, "  %s has no checksum\n", name)
			}
		}
		name, source, checksum = "", "", false
	}
	for _, line := range readLines("Cargo.lock") {
		switch {
		case strings.HasPrefix(line, "[[package]]"):
			flush()
		case strings.HasPrefix(line, `name = "`):
			name = strings.TrimSuffix(strings.TrimPrefix(line, `name = "`), `"`)
		case strings.HasPrefix(line, `source = "`):
			source = strings.TrimSuffix(strings.TrimPrefix(line, `source = "`), `"`)
		case strings.HasPrefix(line, `checksum = "`):
			checksum = true
		}
	}
	flush()
	if out.Len() > 0 {
		f.say("unreviewed dependency source:\n" + out.String())
	}
}

// -- 17. an acceptance command must not be able to pass over nothing -----------
//
// ⛔ `cargo test` with a bare word selects by test NAME, and a filter matching
// none prints `running 0 tests` for every binary and exits 0.
//
// ⛔ TWO SOURCES, ONE TOKENISER: an entry's `Prove` and a workflow's `run:` are
// two doors into the same mistake, so the extractors differ and the judgement
// does not.

var (
	projCargoTestRe   = regexp.MustCompile(`^cargo[ \t]+test([ \t]|$)`)
	projCargoAnywhere = regexp.MustCompile(`cargo[ \t]+test`)
	projCargoCutRe    = regexp.MustCompile(`^.*cargo[ \t]+test`)
	projTrailSlashRe  = regexp.MustCompile(`[ \t]*\\[ \t]*$`)
	projValueFlagRe   = regexp.MustCompile(`^(-p|-j|-F|--package|--exclude|--test|--bin|--example|--bench|--features|--target|--target-dir|--manifest-path|--profile|--jobs|--message-format|--color|--config|--test-threads|--skip)$`)
	projSpaceRunRe    = regexp.MustCompile(`[ \t]+`)
)

type projCommand struct {
	file    string
	line    int
	command string
}

func projectAcceptance(f *projectFails) {
	var found []projCommand

	// ⚠ SCOPED TO `Prove:` PARAGRAPHS, and that is the whole rule rather than an
	// exclusion list: a `Closure evidence` paragraph records what was run on a
	// past tree, and rewriting it would falsify the record.
	for _, file := range gitLines("ls-files", "TODO/*.md") {
		inProve := false
		buf := ""
		startLine := 0
		emit := func() {
			if inProve && buf != "" {
				// ⚠ A code span wraps across lines, so the paragraph is joined
				// before the spans are found.
				parts := strings.Split(buf, "`")
				for k := 1; k < len(parts); k += 2 {
					if projCargoTestRe.MatchString(parts[k]) {
						found = append(found, projCommand{file, startLine, parts[k]})
					}
				}
			}
			inProve = false
			buf = ""
		}
		for i, line := range readLines(file) {
			nr := i + 1
			switch {
			case trimBlankLeft(line) == "":
				emit()
			case strings.HasPrefix(line, "Prove:"):
				emit()
				inProve = true
				startLine = nr
				buf = line
			default:
				if inProve {
					buf = buf + " " + line
				}
			}
		}
		emit()
	}

	for _, file := range gitLines("ls-files", ".github/workflows/*.yml") {
		for i, line := range readLines(file) {
			// A commented-out command is not one every push runs.
			if strings.HasPrefix(trimBlankLeft(line), "#") {
				continue
			}
			if !projCargoAnywhere.MatchString(line) {
				continue
			}
			cmd := projCargoCutRe.ReplaceAllString(line, "cargo test")
			cmd = projTrailSlashRe.ReplaceAllString(cmd, "")
			found = append(found, projCommand{file, i + 1, cmd})
		}
	}

	var out strings.Builder
	for _, c := range found {
		tok := projSpaceRunRe.Split(c.command, -1)
		expect := false
		for i := 2; i < len(tok); i++ {
			t := tok[i]
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "-") {
				// A flag that takes a value consumes the next bare word, which is
				// then a target or a package rather than a name filter.
				expect = !strings.Contains(t, "=") && projValueFlagRe.MatchString(t)
				continue
			}
			if expect {
				expect = false
				continue
			}
			fmt.Fprintf(&out, "  %s:%d selects tests by name, so it exits 0 over nothing: %s\n",
				c.file, c.line, c.command)
			break
		}
	}
	if out.Len() > 0 {
		f.say("an acceptance that can pass over nothing:\n" + out.String())
	}
}

// -- 18. a git dependency in a manifest ----------------------------------------
//
// ⚠ The lockfile rule above is the authority; this one fires earlier and names
// the manifest line, so the report points at the file somebody edited.

var projGitDepRe = regexp.MustCompile(`(^|[{,][[:space:]]*)git[[:space:]]*=`)

func projectGitDependency(f *projectFails) {
	files := gitScope("*Cargo.toml")
	var hits []string
	for _, file := range files {
		for i, line := range readLines(file) {
			if !projGitDepRe.MatchString(line) {
				continue
			}
			// ⚠ `grep -n` WITHOUT `-H` PRINTS NO PATH OVER A SINGLE FILE, and the
			// half this replaces pipes exactly that through `xargs`. Reproduced:
			// a port that always prefixed would differ from both halves on a tree
			// with one manifest.
			if len(files) == 1 {
				hits = append(hits, fmt.Sprintf("%d:%s", i+1, line))
			} else {
				hits = append(hits, fmt.Sprintf("%s:%d:%s", file, i+1, line))
			}
		}
	}
	if len(hits) > 0 {
		f.say("git dependency in a manifest: " + strings.Join(hits, "\n"))
	}
}

// -- 19-21. the three .ps1 rules -----------------------------------------------

var projWriteErrorRe = regexp.MustCompile(`(^|[|;{])[[:space:]]*Write-Error([[:space:]]|$)`)
var projCommentAfterRe = regexp.MustCompile(`:[[:space:]]*#`)

func projectPowerShell(f *projectFails) {
	var bomless, native, writeError strings.Builder
	for _, ps1 := range gitScope("*.ps1") {
		raw, err := os.ReadFile(ps1)
		if err != nil {
			continue
		}

		// ⛔ THE TEST IS ON THE BYTES, not on a list of files, and TAB, CR and LF
		// are stripped first: a `.ps1` keeps CRLF here, so a class that did not
		// exclude the carriage return demanded a BOM for a file that is pure
		// ASCII.
		stripped := make([]byte, 0, len(raw))
		for _, b := range raw {
			if b == '\t' || b == '\n' || b == '\r' {
				continue
			}
			stripped = append(stripped, b)
		}
		nonASCII := false
		for _, b := range stripped {
			if b < ' ' || b > '~' {
				nonASCII = true
				break
			}
		}
		if nonASCII && !(len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF) {
			fmt.Fprintf(&bomless, " %s", ps1)
		}

		text := string(raw)
		if strings.Contains(text, "ErrorActionPreference = 'Stop'") &&
			!strings.Contains(text, "PSNativeCommandUseErrorActionPreference") {
			fmt.Fprintf(&native, " %s", ps1)
		}

		// ⚠ THE NEEDLE IS AN INVOCATION, NOT THE WORD. Its first run fired on this
		// rule's own twin, because the failure message it raises contains the
		// name, so a match is the name in COMMAND POSITION and a comment is
		// skipped.
		for i, lineB := range splitLines(raw) {
			line := string(lineB)
			if !projWriteErrorRe.MatchString(line) {
				continue
			}
			if projCommentAfterRe.MatchString(fmt.Sprintf("%d:%s", i+1, line)) {
				continue
			}
			fmt.Fprintf(&writeError, " %s", ps1)
			break
		}
	}
	if bomless.Len() > 0 {
		f.say("a .ps1 with non-ASCII and no UTF-8 BOM is mis-decoded by PowerShell 5.1:" + bomless.String())
	}
	if native.Len() > 0 {
		f.say("a .ps1 stops on errors without saying what a native exit code means:" + native.String())
	}
	if writeError.Len() > 0 {
		f.say("a .ps1 reports through Write-Error, whose rendering wraps by host width:" + writeError.String())
	}
}

// -- 22-24. the same language behind a second door -----------------------------
//
// ⛔ The three rules above iterate `*.ps1`, so none of them reaches a `pwsh` block
// inside a workflow - the same language with the same two hazards, plus a third
// that only exists here: GitHub reads the block's residual `$LASTEXITCODE` as the
// step's verdict, so an INVERTED guard fails the step by succeeding at its job.

type projPwshBlock struct {
	id   string
	body []string
}

func projectPwshBlocks() []projPwshBlock {
	var out []projPwshBlock
	shellRe := regexp.MustCompile(`^ *shell:[ \t]*(pwsh|powershell)[ \t]*$`)
	nameRe := regexp.MustCompile(`^ *- name:`)
	nameCutRe := regexp.MustCompile(`^ *- name:[ \t]*`)
	runRe := regexp.MustCompile(`^ *run:`)
	runCutRe := regexp.MustCompile(`^ *run:[ \t]*`)

	for _, wf := range projWorkflowFiles() {
		var body []string
		ispwsh := false
		stepname := ""
		inrun := false
		runind := -1
		keyind := -1
		flush := func() {
			if stepname != "" && ispwsh && len(body) > 0 {
				out = append(out, projPwshBlock{id: wf + ":" + stepname, body: body})
			}
			body, ispwsh, stepname, inrun, runind = nil, false, "", false, -1
		}
		for _, raw := range readLines(wf) {
			line := strings.TrimSuffix(raw, "\r")
			// ⚠ THE INDENT IS SPACES ONLY, as awk's `match(s, /[^ ]/)` counts it:
			// a tab-led line reads as column 0 in both halves.
			ind := -1
			for i := 0; i < len(line); i++ {
				if line[i] != ' ' {
					ind = i
					break
				}
			}
			if inrun {
				if ind < 0 {
					body = append(body, "")
					continue
				}
				if ind >= runind {
					body = append(body, line[runind:])
					continue
				}
				inrun = false
			}
			if ind < 0 {
				continue
			}
			if nameRe.MatchString(line) {
				flush()
				stepname = trimOneQuote(nameCutRe.ReplaceAllString(line, ""))
				keyind = ind + 2
				continue
			}
			if stepname == "" {
				continue
			}
			if ind < keyind {
				flush()
				continue
			}
			if ind != keyind {
				continue
			}
			if shellRe.MatchString(line) {
				ispwsh = true
				continue
			}
			if runRe.MatchString(line) {
				v := runCutRe.ReplaceAllString(line, "")
				if v == "|" || v == ">" || v == "|-" || v == ">-" {
					inrun = true
					runind = keyind + 2
					continue
				}
				body = append(body, v)
			}
		}
		flush()
	}
	return out
}

func projectWorkflowPwsh(f *projectFails) {
	var wfNative, wfWriteErr, wfResidue strings.Builder
	for _, block := range projectPwshBlocks() {
		hasStop, hasNative, hasCode := false, false, false
		last := ""
		reported := false
		for _, body := range block.body {
			bare := trimBlankLeft(body)
			if strings.HasPrefix(bare, "#") {
				continue
			}
			if bare != "" {
				last = bare
			}
			if strings.Contains(body, "ErrorActionPreference = 'Stop'") {
				hasStop = true
			}
			if strings.Contains(body, "PSNativeCommandUseErrorActionPreference") {
				hasNative = true
			}
			if strings.Contains(body, "$LASTEXITCODE") {
				hasCode = true
			}
			if !reported && projWriteErrorRe.MatchString(bare) {
				fmt.Fprintf(&wfWriteErr, " [%s]", block.id)
				reported = true
			}
		}
		if hasStop && !hasNative {
			fmt.Fprintf(&wfNative, " [%s]", block.id)
		}
		// ⭐ A BLOCK THAT READS $LASTEXITCODE ENDS IN AN EXPLICIT `exit`, so the
		// step's status is a decision rather than a residue.
		if hasCode && last != "exit" && !strings.HasPrefix(last, "exit ") {
			fmt.Fprintf(&wfResidue, " [%s]", block.id)
		}
	}
	if wfNative.Len() > 0 {
		f.say("a workflow pwsh block stops on errors without saying what a native exit code means:" + wfNative.String())
	}
	if wfWriteErr.Len() > 0 {
		f.say("a workflow pwsh block reports through Write-Error, whose rendering wraps by host width:" + wfWriteErr.String())
	}
	if wfResidue.Len() > 0 {
		f.say("a workflow pwsh block reads $LASTEXITCODE and lets it fall through as the step's verdict:" + wfResidue.String())
	}
}
