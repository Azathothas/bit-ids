// indexes.go - prove the three things a derived file owes: byte-identical clean
// builds, rows that resolve back to the records they came from, and a corrected
// record that has left the views while its bytes are still filed where they
// were. `CORPUS-03` and `CORPUS-04`.
//
// Ported from scripts/corpus/check-indexes.sh, `CI-07`, and compared case for
// case against it before that half was deleted. ⚠ It had no PowerShell twin,
// which is why the Windows lane declared this row.
//
// ⛔ THE STORE HOLDS 1.2.3, 1.2.10 AND 1.2.9, IN THAT ORDER. A lexical sort
// answers 1.2.9 and a numeric one 1.2.10, and writing them out of order means a
// builder that took the last one written would be wrong too.
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// findNamed is `find DIR -name NAME | head -1`, in walk order.
func findNamed(dir, name string) string {
	found := ""
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && found == "" && !d.IsDir() && d.Name() == name {
			found = p
		}
		return nil
	})
	return found
}

// section is `sed -n '/OPEN/,/^  \]/p'`: the lines from the first holding OPEN
// to the first after it that STARTS with two spaces and a bracket.
// ⚠ Starts with, as the regular expression does: `  ],` ends a section too,
// and a first port that wanted the exact line read on into the next one.
func section(lines []string, open string) []string {
	var out []string
	in := false
	for _, l := range lines {
		if !in && strings.Contains(l, open) {
			in = true
		}
		if in {
			out = append(out, l)
			if len(out) > 1 && strings.HasPrefix(l, "  ]") {
				break
			}
		}
	}
	return out
}

func checkIndexes(r *repo) (verdict, error) {
	h := &harness{me: "check-indexes"}

	builder, err := exampleBinary(r.root, "build-store", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	indexer, err := exampleBinary(r.root, "build-indexes", "bit-ids")
	if err != nil {
		return verdict{}, err
	}

	work, err := scratch("checkindexes")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	store := filepath.Join(work, "store")
	const scheme = "fixture-client:-:3:3"
	null := filepath.Join(work, "null")

	buildInto := func(dir string, args ...string) int {
		if os.RemoveAll(dir) != nil || os.MkdirAll(dir, 0o755) != nil {
			return 2
		}
		return runTo(null, null, builder, append(args, dir)...)
	}
	buildStore := func() bool {
		return buildInto(store, "--version", "1.2.3", "--version", "1.2.10", "--version", "1.2.9") == 0
	}

	outPath := filepath.Join(work, "out")
	errPath := filepath.Join(work, "err")
	var out []string
	index := func(args ...string) int {
		rc := runTo(outPath, errPath, indexer, args...)
		out = append(fileLines(outPath), fileLines(errPath)...)
		return rc
	}
	runIndexes := func() int { return index("--scheme", scheme, store) }

	if !buildStore() {
		return verdict{}, errCannotRun("cannot build the fixture store")
	}

	if rc := runIndexes(); rc == 0 {
		h.ok("clean    a three-version store indexes and every row resolves")
	} else {
		h.bad(fmt.Sprintf("clean    the unmutated store was refused (exit %d): %s", rc, headJoin(out, 3)))
	}

	// -- determinism ----------------------------------------------------------
	first := filepath.Join(work, "first.json")
	second := filepath.Join(work, "second.json")
	firstRC := runTo(null, null, indexer, "--scheme", scheme, store, first)
	secondRC := runTo(null, null, indexer, "--scheme", scheme, store, second)
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	switch {
	case firstRC != 0 || secondRC != 0:
		h.bad(fmt.Sprintf("determinism  a build failed (%d, %d)", firstRC, secondRC))
	case bytes.Equal(a, b):
		h.ok("determinism  two clean builds are byte-identical")
	default:
		h.bad("determinism  two clean builds differ")
	}

	firstLines := fileLines(first)
	rows := 0
	for _, l := range firstLines {
		if strings.Contains(l, `"record"`) {
			rows++
		}
	}
	if rows >= 18 {
		h.ok(fmt.Sprintf("determinism  the document carries %d rows, so the comparison had something to compare", rows))
	} else {
		h.bad(fmt.Sprintf("determinism  the document carries only %d rows", rows))
	}

	// -- ordering -------------------------------------------------------------
	if anyContains(firstLines, `"version": "1.2.10"`) {
		h.ok("ordering  the latest view answers 1.2.10 over 1.2.9")
	} else {
		versions := regexp.MustCompile(`"version": "[^"]*"`).FindAllString(string(a), -1)
		last := ""
		if len(versions) > 0 {
			last = versions[len(versions)-1]
		}
		h.bad("ordering  the latest view did not answer 1.2.10: " + last)
	}

	// -- correction -----------------------------------------------------------
	plain := filepath.Join(work, "plain")
	corrected := filepath.Join(work, "corrected")
	plainRC := buildInto(plain, "--version", "1.2.3", "--version", "1.2.10")
	correctedRC := buildInto(corrected, "--version", "1.2.3", "--version", "1.2.10", "--correct", "1.2.10")
	before := findNamed(filepath.Join(plain, "profiles"), "cap-1-2-10.json")
	after := findNamed(filepath.Join(corrected, "profiles"), "cap-1-2-10.json")
	fix := findNamed(filepath.Join(corrected, "profiles"), "cap-1-2-10-fix.json")

	beforeBytes, _ := os.ReadFile(before)
	afterBytes, _ := os.ReadFile(after)
	switch {
	case plainRC != 0 || correctedRC != 0:
		h.bad(fmt.Sprintf("correction  a store could not be built (%d, %d)", plainRC, correctedRC))
	case before == "" || after == "" || fix == "":
		h.bad("correction  the corrected store does not carry both records")
	case bytes.Equal(beforeBytes, afterBytes):
		h.ok("correction  the corrected record's bytes are untouched, and it is still filed")
	default:
		h.bad("correction  writing the correction changed the record it corrects")
	}

	correctedJSON := filepath.Join(work, "corrected.json")
	if rc := runTo(filepath.Join(work, "corrected.out"), filepath.Join(work, "corrected.out"),
		indexer, "--scheme", scheme, corrected, correctedJSON); rc != 0 {
		h.bad(fmt.Sprintf("correction  the corrected store would not index (exit %d)", rc))
	} else {
		doc := fileLines(correctedJSON)
		originalID := ""
		if m := regexp.MustCompile(`"id": "(record:sha256:[0-9a-f]*)"`).FindSubmatch(afterBytes); m != nil {
			originalID = string(m[1])
		}
		fixName := filepath.Base(fix)

		if anyContains(doc, `"superseded": 1`) {
			h.ok("correction  the document counts one superseded record")
		} else {
			h.bad("correction  the document does not count the superseded record")
		}
		latest := section(doc, `"latest": [`)
		if anyContains(latest, fixName) {
			h.ok("correction  the latest view answers with the correction")
		} else {
			h.bad("correction  the latest view does not name " + fixName)
		}
		if originalID != "" && anyContains(latest, originalID) {
			h.bad("correction  the latest view still names the corrected record")
		} else {
			h.ok("correction  and no longer names the record it corrects")
		}
		if originalID != "" && anyContains(section(doc, `"corrections": [`), `"superseded": "`+originalID+`"`) {
			h.ok("correction  the chain names the record a reader may still hold")
		} else {
			h.bad("correction  the chain does not carry " + originalID)
		}
	}

	// -- what is refused ------------------------------------------------------
	// A record the scheme cannot order blocks the view rather than being
	// skipped. ⚠ Planted by adding the shipped fixture, whose version is
	// 0.0.0-fixture.
	plantCase(h, "a version the scheme cannot order", "E-VIW-02", store, buildStore,
		func() bool { return runTo(null, null, builder, store) == 0 },
		runIndexes, func() []string { return out })

	// ⛔ Asked without a scheme at all, which is a different refusal from a
	// scheme that cannot order.
	_ = buildStore()
	if rc := index(store); rc == 1 && anyContains(out, "E-VIW-01") {
		h.ok("E-VIW-01  a target with no declared version scheme")
	} else {
		h.bad(fmt.Sprintf("E-VIW-01  expected exit 1 with E-VIW-01, got %d: %s", rc, headJoin(out, 2)))
	}

	_ = buildStore()
	var profiles []string
	_ = filepath.WalkDir(filepath.Join(store, "profiles"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			profiles = append(profiles, p)
		}
		return nil
	})
	sort.Strings(profiles)
	if len(profiles) == 0 {
		return verdict{}, errCannotRun("the store holds no record to probe")
	}
	h.probeGuards(profiles[0], "fixture-client", "sha256:")

	return h.report("check-indexes/1", "cases"), nil
}
