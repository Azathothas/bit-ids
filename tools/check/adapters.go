// adapters.go - every network fetch a capture adapter makes is bounded.
//
// ⛔ **A CAPTURE HOST THAT HANGS PRODUCES NO EVIDENCE.** Measured on
// `capture-client` runs 21 and 22, 2026-09-15: the RELEASE lane's *Install the
// client* step ran for thirty minutes where runs 19 and 20 took six seconds,
// every bound around it failed to end the step, the job was cancelled at its
// own limit, and a cancelled job leaves no log and no artifact - which runs 7,
// 8 and 9 had already measured. So a stalled fetch does not cost one capture,
// it costs the whole dispatch and teaches nothing.
//
// ⚠ **THE RULE WAS ALREADY WRITTEN AND NOTHING ENFORCED IT.**
// `docs/conventions/shell.md` section 9: *a script that shells out to unknown
// tools without a limit is a script that hangs, and a script that hangs is one
// nobody runs twice.* Four adapters fetched a release artifact with
// `curl -fsSL --retry 2` and no limit at all. ⛔ That is the shape this
// repository calls a preference stated as a rule, and it is the fourth instance
// it has recorded.
//
// ⚠ **AND THE CONVENTION HELD ON ONE OF TWO PATHS INTO ONE PRODUCT.**
// `aria2-next.sh` has carried `--max-time 20` on its JSON-RPC call since it was
// written, ninety lines above a release fetch that had no bound - which is the
// one-gated-door defect `docs/methodology/reviews.md` calls the most recurring
// hole there is.
//
// -- ⭐ WHAT IS REQUIRED, AND WHAT IS DELIBERATELY NOT --------------------------
//
// A `curl` that RETRIEVES - one with `-o` or `-O` - carries `--max-time`. A
// `git clone` or `git fetch` is wrapped in `timeout`, because git has no flag of
// its own and `timeout` is the spelling section 9 gives for a tool that does not
// bound itself.
//
// ⚠ **THE NUMBER IS THE ADAPTER'S AND THE BOUND IS THE CONTRACT'S.**
// `adapters/README.md` says an adapter owns the switches its own product needs,
// and a source build legitimately needs longer than a 14-megabyte download. So
// this refuses an ABSENT bound and never argues with a value.
//
// ⚠ **A `curl` THAT ONLY ASKS IS OUT OF SCOPE**, and that is a decision rather
// than an oversight: a HEAD or a probe that prints to stdout is bounded by
// whatever reads it, and widening this to every invocation would make the rule
// fire on shapes nobody has been bitten by.
//
// -- ⛔ WHAT THE DOOR SWEEP FOUND THAT THIS SCOPE DOES NOT COVER ---------------
//
// The sweep that followed this rule found the same defect twice more, one
// directory away, which is the shape the rule itself is about:
//
//   - `scripts/doctor/provision.sh` fetched every pinned tool with no limit.
//     ⭐ Fixed in the same change. It is out of SCOPE here because a stall there
//     costs a session's start in front of somebody rather than a whole dispatch.
//   - `scripts/common/mine-repo.sh` and its `.ps1` twin cloned with no bound.
//     ⭐ FIXED on 2026-09-17, and the answer was not a wrapper. That pair was
//     left unfixed because `timeout.exe` on Windows is a PAUSE, so a wrapped
//     `.ps1` would sleep for the bound and then clone - which meant two idioms
//     and a difference for `check-twins` to compare. ⭐ `git -c
//     http.lowSpeedLimit -c http.lowSpeedTime` is git's OWN bound, spelled
//     identically on every platform, so the pair needs no second idiom at all.
//     ⚠ Still out of SCOPE here, for the reason above it: a stall there costs a
//     session's reference mining rather than a dispatch, and this rule's
//     expressions are shell-shaped where that pair is half PowerShell.
//
// ⚠ So this rule covers the capture install path and says so, rather than
// claiming a reach it does not have. A rule that named the whole tree and
// enforced one directory is the shape this repository calls a preference stated
// as a rule.
//
// -- ⛔ AND THE SUBJECT MOVED OUT FROM UNDER IT. `ACQ-06`, 2026-09-17 ----------
//
// ⭐ **THE FLOOR IS WHAT CAUGHT THAT, WHICH IS WHY IT IS HERE.** `ACQ-06` moved
// every release fetch out of the four adapters and into one rootless installer,
// so the adapters went from four bounded `curl -o` invocations to none - and
// this rule, reading adapters alone, answered `2 fetch(es) found across 4
// adapter(s), which is too few to be this tree` and exited 2. ⛔ Without the
// floor it would have reported a clean tree over a path whose every fetch it had
// stopped reading, which is a guard narrower than the class it is about: the
// defect this repository records more often than any other.
//
// ⚠ **SO THE SCOPE FOLLOWS THE FETCHES RATHER THAN THE DIRECTORY.**
// `scripts/acquisition/install-rootless.sh` is in scope because it is where the
// capture path now retrieves what it installs; `provision.sh` and `mine-repo` -
// a Go helper in minerepo.go since 2026-09-29, bounded the same way - stay out
// for the reasons above, which are about what a stall COSTS rather than about
// which directory a file is in.
//
// ⛔ AND IT FOLLOWED THEM AGAIN INTO GO, `CI-11`, 2026-10-15. The installer is
// `bit-check install-rootless` and the listing fetch `bit-check
// fetch-releases`, so the files that retrieve what a capture installs are
// installrootless.go and fetchreleases.go. ⚠ Both still run `curl`, and a Go
// call is not a shell line: an `exec.Command(curl, ...)` is read whole, to its
// closing parenthesis, and one that writes a file must pass `--max-time`.
//
// -- ⛔ AND THE ADAPTERS FOLLOWED, `CI-11`, 2026-10-15 ------------------------
//
// The four capture adapters are `bit-check adapter <name>`, in
// tools/check/adapter_<name>.go, with their shared mechanics in adapterkit.go.
// Every `exec.Command(...)` there is read whole and judged by what it RUNS,
// which is its first argument, or its third when the first is `"timeout"`:
//
//	a curl that writes a file   bounded by its own "--max-time", or by a
//	                            "timeout" in front of it
//	a git clone or fetch        bounded by a "timeout" in front of it, and
//	                            nothing else
//
// ⛔ A "timeout" IS A BOUND ONLY AS THE PROGRAM. `exec.Command("git", "clone",
// ..., "timeout", "600")` carries the word and bounds nothing, which is the Go
// shape of the shell rule's `git clone ... && timeout 5 true`.
// ⚠ A CONTEXT DEADLINE IS NOT ACCEPTED, and that is a choice rather than a gap:
// `exec.CommandContext` sends KILL to the one child it started when a context
// declared somewhere else expires, and a reader of the call cannot see that
// deadline. A `timeout` argv is read off the line, answers 124 as the shell's
// did, and is what every adapter route already reports by name.
// ⚠ An invocation whose argv is built elsewhere and passed as a slice cannot be
// judged, so every curl and git in an adapter is written as a literal call.
//
// ⭐ THE SHELL READER NEVER SAW A WRAPPED CURL. It matched `curl` in command
// position only, so `timeout 120 curl -fsSL ... -o` in qBittorrent's source
// route was neither counted nor judged. Bounded, as it happened; the Go reader
// counts it.
//
// ⛔ AND THE FLOOR ASKS FOR BOTH KINDS. A reader that stopped seeing Go git calls
// would still find enough curls to pass a count of three, so at least one git
// clone or fetch has to be read too.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// adapterRe is the scope: the capture adapters and the rootless installer they
// fetch through, which are the scripts that run on a contained host with no
// operator watching.
var adapterRe = regexp.MustCompile(
	`^(scripts/capture/adapters/[^/]+\.sh|tools/check/(installrootless|fetchreleases|adapterkit|adapter_[A-Za-z0-9_-]+)\.go)$`)

// goExecRe finds a Go call that runs a program, up to the parenthesis that
// opens its arguments. goExecCalls reads each to the parenthesis that closes it.
var goExecRe = regexp.MustCompile(`exec\.Command(Context)?\(`)

// goExecCalls answers each `exec.Command(...)` in a Go file as its top-level
// arguments, with the line it starts on. Comment lines are not read.
//
// ⚠ BALANCED RATHER THAN TO THE FIRST `)`, because an argument such as
// `fmt.Sprint(n)` closes a parenthesis of its own, and a call read only to that
// would lose every flag after it. ⚠ AND A STRING IS SKIPPED WHOLE, because
// `jq`'s `del(."rpc-password")` is a parenthesis inside an argument.
func goExecCalls(text string) (calls [][]string, lines []int) {
	var b strings.Builder
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimLeft(l, " \t"), "//") {
			l = ""
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	text = b.String()
	for _, at := range goExecRe.FindAllStringIndex(text, -1) {
		var args []string
		depth, start := 0, at[1]
		for i := at[1] - 1; i < len(text); i++ {
			switch c := text[i]; c {
			case '"', '`':
				j := i + 1
				for j < len(text) && text[j] != c {
					if c == '"' && text[j] == '\\' {
						j++
					}
					j++
				}
				i = j
			case '(', '[', '{':
				depth++
			case ')', ']', '}':
				depth--
				if depth == 0 {
					args = append(args, strings.TrimSpace(text[start:i]))
					i = len(text)
				}
			case ',':
				if depth == 1 {
					args = append(args, strings.TrimSpace(text[start:i]))
					start = i + 1
				}
			}
		}
		if at[0]+len("exec.CommandContext(") == at[1] && len(args) > 0 {
			args = args[1:]
		}
		calls = append(calls, args)
		lines = append(lines, strings.Count(text[:at[0]], "\n")+1)
	}
	return calls, lines
}

// goFetch judges one Go call: whether it fetches, which kind, and whether it is
// bounded. The program is the first argument, or the third behind a `timeout`.
func goFetch(args []string) (kind string, bounded bool) {
	has := func(want ...string) bool {
		for _, a := range args {
			for _, w := range want {
				if a == w {
					return true
				}
			}
		}
		return false
	}
	if len(args) == 0 {
		return "", false
	}
	prog, wrapped := args[0], false
	if prog == `"timeout"` && len(args) >= 3 {
		prog, wrapped = args[2], true
	}
	switch prog {
	case "curl", `"curl"`:
		if !has(`"-o"`, `"-O"`) {
			return "", false
		}
		return "curl", wrapped || has(`"--max-time"`)
	case `"git"`:
		if !has(`"clone"`, `"fetch"`) {
			return "", false
		}
		return "git", wrapped
	}
	return "", false
}

// retrievingCurlRe matches a curl invocation that writes a file.
//
// ⚠ In command position or after a pipe, which is the needle discipline
// `check-project`'s `Write-Error` rule already uses: a `curl` inside a comment
// or a string is not an invocation, and a rule that fired on its own
// description is one this repository has already shipped once.
// ⚠ LEADING WHITESPACE IS PART OF COMMAND POSITION. The first spelling
// anchored on `^` alone, and every invocation in these adapters is indented
// inside a `case` arm - so it matched nothing at all and the floor below
// reported *0 fetches across 4 adapters*. ⭐ That floor is what caught it: a
// rule that found nothing to ask about had asked nothing, and without it this
// would have reported every adapter bounded while reading none of them.
var retrievingCurlRe = regexp.MustCompile(`(^\s*|[|;&(]\s*)curl\b`)

// timeoutWrapRe matches a `timeout` that wraps the command after it.
var timeoutWrapRe = regexp.MustCompile(`(^\s*|[|;&(]\s*)timeout\s`)

// gitFetchRe matches the git subcommands that reach a network.
//
// ⚠ `clone` and `fetch` only. `git rev-parse` and `git log` are local and
// bounding them would be noise around the rule this exists for.
//
// ⛔ IT MATCHES WHEREVER THE COMMAND SITS, INCLUDING AFTER A WRAPPER, and the
// first spelling did not. Anchored in command position it could not match
// `timeout 600 git clone` at all - so a BOUNDED clone was invisible and only an
// unbounded one was counted. The rule was still right and the COUNT meant two
// different things: every curl whether bounded or not, and only the clones that
// were broken. A floor over a number like that is a floor over nothing.
var gitFetchRe = regexp.MustCompile(`(^|\s)git\s+(clone|fetch)\b`)

// logicalLines joins a script's backslash continuations, so a flag on the second
// physical line of one invocation is read as part of it.
//
// ⛔ WITHOUT THIS THE RULE IS BACKWARDS. Every bounded fetch in this tree spans
// three physical lines, so a per-line reader would refuse exactly the
// invocations that comply and accept a one-line unbounded one.
func logicalLines(text string) []string {
	var out []string
	var cur strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if strings.HasSuffix(trimmed, `\`) {
			cur.WriteString(strings.TrimSuffix(trimmed, `\`))
			cur.WriteString(" ")
			continue
		}
		cur.WriteString(trimmed)
		out = append(out, cur.String())
		cur.Reset()
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// isComment reports whether a logical line is only a comment.
func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "#")
}

func checkAdapters(r *repo) (verdict, error) {
	files := r.matching(adapterRe, nil)
	// ⛔ A SCOPE TOO SMALL TO BE REAL IS REFUSED, which is the guard
	// `check-ignores` and `ACQ-01`'s catalogue scan both carry. An empty set
	// satisfies every rule below perfectly, so a matcher that stopped matching
	// would report every adapter bounded over no adapters at all.
	if len(files) < 4 {
		return verdict{}, errCannotRun(
			fmt.Sprintf("%d file(s) in scope, which is too few to be this tree", len(files)))
	}

	var problems []string
	fetches, clones := 0, 0
	for _, f := range files {
		text, err := os.ReadFile(f)
		if err != nil {
			return verdict{}, errCannotRun("cannot read " + f)
		}
		if strings.HasSuffix(f, ".go") {
			calls, at := goExecCalls(string(text))
			for i, call := range calls {
				kind, bounded := goFetch(call)
				switch kind {
				case "curl":
					fetches++
					if !bounded {
						problems = append(problems, fmt.Sprintf(
							"%s:%d a curl that writes a file carries no --max-time", f, at[i]))
					}
				case "git":
					fetches++
					clones++
					if !bounded {
						problems = append(problems, fmt.Sprintf(
							"%s:%d a git clone or fetch is not wrapped in timeout", f, at[i]))
					}
				}
			}
			continue
		}
		for n, line := range logicalLines(string(text)) {
			if isComment(line) {
				continue
			}
			switch {
			case retrievingCurlRe.MatchString(line):
				// ⚠ Only a curl that RETRIEVES. One that prints is bounded by
				// its reader; see the header.
				if !strings.Contains(line, " -o ") && !strings.Contains(line, " -O") {
					continue
				}
				fetches++
				if !strings.Contains(line, "--max-time") {
					problems = append(problems, fmt.Sprintf(
						"%s:%d a curl that writes a file carries no --max-time", f, n+1))
				}
			case gitFetchRe.MatchString(line):
				fetches++
				clones++
				// ⛔ THE WRAPPER MUST COME BEFORE THE COMMAND IT BOUNDS. Asking
				// only whether `timeout` appears somewhere on the line would
				// accept `git clone ... && timeout 5 true`, which bounds
				// nothing. So the prefix up to the git invocation is what is
				// asked, and it is asked for a `timeout` in command position.
				at := gitFetchRe.FindStringIndex(line)
				if at == nil || !timeoutWrapRe.MatchString(line[:at[0]]) {
					problems = append(problems, fmt.Sprintf(
						"%s:%d a git clone or fetch is not wrapped in timeout", f, n+1))
				}
			}
		}
	}

	// ⛔ AND A RULE THAT FOUND NOTHING TO ASK ABOUT HAS ASKED NOTHING. A reader
	// that matched no invocation at all would report a clean tree while examining
	// none of them. ⭐ This is the floor that caught `ACQ-06` moving every release
	// fetch out of the adapters: it answered 2 where the tree has more, and the
	// scope followed the subject rather than the rule going quietly blind.
	if fetches < 3 {
		return verdict{}, errCannotRun(
			fmt.Sprintf("%d fetch(es) found across %d file(s), which is too few to be this tree",
				fetches, len(files)))
	}
	if clones == 0 {
		return verdict{}, errCannotRun(
			fmt.Sprintf("no git clone or fetch found across %d file(s), which is too few to be this tree", len(files)))
	}

	v := verdict{
		json: fmt.Sprintf(`{"schema":"check-adapters/2","problems":%d,"files":%d,"fetches":%d}`,
			len(problems), len(files), fetches),
	}
	if len(problems) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "a capture-path fetch has no time limit, %d problem(s):\n\n", len(problems))
		for _, p := range problems {
			fmt.Fprintf(&b, "  %s\n", p)
		}
		b.WriteString("\n⛔ A capture host that hangs produces no evidence: the job is cancelled at\n")
		b.WriteString("its own bound and a cancelled job leaves no log and no artifact.\n")
		b.WriteString("docs/conventions/shell.md section 9 is the rule.\n")
		v.code = 1
		v.text = b.String()
		return v, nil
	}
	v.text = fmt.Sprintf("all %d fetch(es) across %d file(s) are bounded\n", fetches, len(files))
	return v, nil
}
