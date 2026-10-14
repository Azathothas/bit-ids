// bit-check - this repository's checking layer, as one binary on both platforms.
//
// CI-10. The layer this replaces was a `.sh` and a hand-written `.ps1` twin per
// rule, plus `check-twins` to compare them, and the twin was most of the wall
// clock: measured on 2026-09-10 over eleven pairs, the `sh` halves together took
// 7.4 seconds and the PowerShell halves 96, of which `check-control-bytes.ps1`
// alone was 67.9. `check-workflow` runs the whole gate about nine times, so that
// cost is paid nine times on every push.
//
// ⭐ ONE BINARY REMOVES THE CLASS RATHER THAN CHECKING FOR IT. `check-twins`
// exists because two hand-written halves drift. A single implementation that
// runs on both platforms cannot drift from itself, so the comparison stops being
// necessary rather than getting faster.
//
// ⚠ AND ONE HELPER, WHICH WRITES RATHER THAN CHECKS. `mine-repo` was the last
// twin pair `check-twins` compared, so it came here for the same reason the
// checks did; it is no gate row, and minerepo.go says how it is kept apart.
//
// -- ⛔ WHAT A PORT MAY NOT CHANGE -------------------------------------------
//
// A ported check refuses exactly what its shell half refused, over the same
// planted defect, with the same exit-code vocabulary:
//
//	0  the rule held
//	1  the rule was refused
//	2  the check could not run
//
// ⚠ A port that changed one verdict while getting faster would be trading the
// thing being measured for the measurement. `scripts/common/check-bitcheck.sh`
// is what compares each ported check against the halves it replaces, case for
// case over the same plants, and it runs BEFORE any half is deleted.
//
// -- ⭐ NO DEPENDENCIES, WHICH IS A SUPPLY-CHAIN PROPERTY AND NOT A PREFERENCE -
//
// This module has an empty require list and therefore no `go.sum`. Nothing here
// is fetched at build time, so a build needs no network and no pin audit, and
// `CI-04`'s dependency surface does not grow by a language.
//
// Usage:
//
//	bit-check <check> [--json]
//	bit-check check-licences --permitted
//	bit-check check-no-secrets --public
//	bit-check check-remote-items [--repo OWNER/NAME]
//	bit-check mine-repo OWNER/NAME [--out DIR] [--route auto|gh|proxy] [--no-clone] [--json]
//	bit-check mine-repo --selftest [--json]
//	bit-check --rows
//
// ⛔ Read the exit code from this process, unpiped. ⚠ `go run . <check>` is a
// different process: it exits 1 for any non-zero exit of the program it ran, so
// a refusal and a could-not-run are one code through it. Measured 2026-09-29.
package main

import (
	"fmt"
	"os"
	"sort"
)

// cannotRun is the error type that means exit 2 rather than exit 1.
//
// ⛔ THE TWO ARE NEVER FOLDED TOGETHER. A harness exit of 2 is *could not run*
// and never *refused*; this repository has counted one as the other and reported
// a guard proved over a plant that had not compiled.
type cannotRun struct{ msg string }

func (e cannotRun) Error() string { return e.msg }

func errCannotRun(msg string) error { return cannotRun{msg} }

// optPermitted is check-licences' listing mode. ⚠ It is reported BEFORE any
// rule runs, because a caller asking what is permitted is asking about the
// file as written rather than about whether it is coherent.
var optPermitted bool

// verdict is what a check answers: the exit code plus the one JSON line that
// `check-bitcheck.sh` compares against the shell half's.
type verdict struct {
	code int
	json string
	text string
	// ⛔ stderr IS PRINTED IN BOTH MODES, and it exists for one shape: a refusal
	// that happens BEFORE a check has a countable answer. check-changelog's
	// no-entries case is the instance - a file whose headings the parser does not
	// recognise has no `problems` to report, so its shell half wrote to stderr
	// and exited 1 without emitting JSON at all. A port that invented a JSON line
	// there would be answering a question its predecessor refused to answer.
	stderr string
	// jsonReport is a report for a person that `--json` sends to STDERR, ahead of
	// the document on stdout. ⚠ One check has it: check-remote-items' shell half
	// kept its whole report in that mode and moved it off stdout, so that a
	// parser reading stdout gets the document alone.
	jsonReport string
}

// check is one rule. The name is the gate row label, so a row this map does not
// carry is a row the gate cannot run.
type check func(r *repo) (verdict, error)

var checks = map[string]check{
	"check-adapters":      checkAdapters,
	"check-cache":         checkCache,
	"check-catalogue":     checkCatalogue,
	"check-changelog":     checkChangelog,
	"check-control-bytes": checkControlBytes,
	"check-corpus":        checkCorpus,
	"check-docs":          checkDocs,
	"check-formats":       checkFormats,
	"check-ignores":       checkIgnores,
	"check-indexes":       checkIndexes,
	"check-licences":      checkLicences,
	"check-markers":       checkMarkers,
	"check-no-secrets":    checkNoSecrets,
	"check-one-home":      checkOneHome,
	"check-placeholders":  checkPlaceholders,
	"check-release":       checkRelease,
	"check-project":       checkProject,
	"check-remote-items":  checkRemoteItems,
	"check-staleness":     checkStaleness,
}

func names() []string {
	out := make([]string, 0, len(checks))
	for k := range checks {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "bit-check: which check? one of: %v\n", names())
		os.Exit(2)
	}

	// ⭐ --rows PRINTS THE NAMES AND RUNS NOTHING, so the gate can compare the
	// set it queues against the set this binary carries. A row the runner names
	// and the binary does not have is rot, exactly as a step a harness names and
	// a workflow no longer has is.
	if args[0] == "--rows" {
		for _, n := range names() {
			fmt.Println(n)
		}
		os.Exit(0)
	}

	// ⭐ A HELPER IS DISPATCHED BEFORE THE CHECKS ARE READ, which is why it is not
	// one of them: `mine-repo` writes, so it is no gate row and `--rows` does not
	// name it. It parses its own arguments and answers its own exit code.
	if args[0] == "mine-repo" {
		os.Exit(mineRepo(args[1:]))
	}

	name := args[0]
	jsonOut := false
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		switch a {
		case "--repo":
			// ⚠ check-remote-items' question about ANOTHER repository, and the
			// one flag here that takes a value. Missing, it is empty, as the shell
			// half's `${1:-}` made it: the repository is then the one git names.
			if i+1 < len(rest) {
				i++
				optRepo = rest[i]
			}
		case "--json":
			jsonOut = true
		case "--permitted":
			// ⚠ ONE CHECK'S MODE RATHER THAN A GLOBAL ONE, and it is parsed here
			// because the dispatcher owns the argument list. It prints what
			// `permittedTargets` answers, which is also what check-cache calls: one
			// derivation of which targets may be redistributed, asked two ways,
			// rather than the value in two places this repository refuses.
			optPermitted = true
		case "--public":
			// ⚠ check-no-secrets' second question rather than a stricter first
			// one. Emails, absolute home paths and long hex are legitimate
			// content in a private project, which is why the gate runs this as
			// its own row rather than as a flag on the default one.
			optPublic = true
		case "--all-history":
			// ⚠ Slow on purpose and deliberately not a gate row: it reads every
			// blob ever committed.
			optAllHistory = true
		default:
			fmt.Fprintf(os.Stderr, "bit-check: unknown argument: %s\n", a)
			os.Exit(2)
		}
	}

	fn, ok := checks[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "bit-check: no such check: %s\n", name)
		os.Exit(2)
	}

	r, err := openRepo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
		os.Exit(2)
	}

	v, err := fn(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
		os.Exit(2)
	}

	// ⚠ A REPORT MAY PRECEDE A REFUSAL. A check that has printed part of its
	// report and then cannot go on prints what it had, then why it stopped, and
	// no document - the order its shell half wrote them in.
	if jsonOut {
		fmt.Fprint(os.Stderr, v.jsonReport)
		if v.stderr != "" {
			fmt.Fprint(os.Stderr, v.stderr)
		} else {
			fmt.Println(v.json)
		}
	} else {
		fmt.Print(v.text)
		fmt.Fprint(os.Stderr, v.stderr)
	}
	os.Exit(v.code)
}
