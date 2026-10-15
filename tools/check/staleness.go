// staleness.go - drive the staleness monitor over real stores and real
// resolutions: a new stable release opens one request, a preview and a release
// already measured open none, and a second run over a tracker holding the first
// run's request opens nothing. `CI-02`.
//
// Ported from scripts/ci/check-staleness.sh, `CI-07`, and compared case for case
// against it before that half was deleted. ⚠ It had no PowerShell twin, which is
// why the Windows lane declared this row.
//
// ⛔ THE INPUTS ARE PLANTED, NOT THE SOURCE. This unit's guards are about a
// comparison, so a plant is a different store, a different release listing, or
// a different tracker, each built here.
//
// ⭐ THE REQUEST IDENTIFIER IS RE-DERIVED BY A SHA-256 THIS PROJECT DID NOT
// WRITE. The shell half used Python's `hashlib`; this uses Go's `crypto/sha256`.
// Either is independent of the Rust encoder under test, which is the point: the
// survey agreeing with itself would prove nothing.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// jsonDoc reads a document, or nothing.
func jsonDoc(path string) map[string]any {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var d map[string]any
	if json.Unmarshal(body, &d) != nil {
		return nil
	}
	return d
}

// firstOf answers `d[list][0][key]` as text, or "" when any step is absent, as
// the shell half's `python3 -c "print(...)"` answered nothing on an exception.
func firstOf(d map[string]any, list, key string) string {
	items, ok := d[list].([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	switch v := item[key].(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

// lenOf answers `len(d[list])` as text, or "".
func lenOf(d map[string]any, list string) string {
	items, ok := d[list].([]any)
	if !ok {
		return ""
	}
	return fmt.Sprint(len(items))
}

func checkStaleness(r *repo) (verdict, error) {
	h := &harness{me: "check-staleness"}

	builder, err := exampleBinary(r.root, "build-store", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	resolver, err := exampleBinary(r.root, "resolve-stable", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	surveyor, err := exampleBinary(r.root, "survey-staleness", "bit-ids")
	if err != nil {
		return verdict{}, err
	}

	work, err := scratch("checkstaleness")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	at := func(name string) string { return filepath.Join(work, name) }
	null := at("null")

	makeStore := func(dir string, versions ...string) bool {
		if os.RemoveAll(dir) != nil || os.MkdirAll(dir, 0o755) != nil {
			return false
		}
		var args []string
		for _, v := range versions {
			args = append(args, "--version", v)
		}
		return runTo(null, null, builder, append(args, dir)...) == 0
	}

	// A release listing as GitHub serves one, a day apart per tag, and the
	// resolution `resolve-stable` makes of it. `@prerelease` flags a tag.
	makeResolution := func(name string, specs ...string) bool {
		var b strings.Builder
		b.WriteString("[\n")
		for i, spec := range specs {
			tag, pre := strings.CutSuffix(spec, "@prerelease")
			if i > 0 {
				b.WriteString(",\n")
			}
			fmt.Fprintf(&b, `  {"tag_name": "%s", "prerelease": %t, "draft": false, "published_at": "2026-0%d-02T00:00:00Z"}`,
				tag, pre, i+1)
		}
		b.WriteString("\n]\n")
		body := at(name + ".releases.json")
		if os.WriteFile(body, []byte(b.String()), 0o644) != nil ||
			os.WriteFile(body+".fetched-at", []byte("2026-09-04T12:00:00Z\n"), 0o644) != nil {
			return false
		}
		out := at(name + ".resolution.json")
		// ⚠ The fifth argument is the scheme floor, `-` for none. `CLIENT-06`
		// added it, and this caller was missed until a review found the gate
		// reporting this harness as a skip.
		runTo(out, null, resolver, "fixture-client", "-", "3", "3", "-", "github-releases",
			"https://example.invalid/releases", body)
		st, err := os.Stat(out)
		return err == nil && st.Size() > 0
	}

	rc := 0
	survey := func(store, resolution, out string, open ...string) {
		args := []string{store, "--resolution", at(resolution + ".resolution.json"), "--platforms", "linux"}
		if len(open) > 0 {
			args = append(args, "--open", open[0])
		}
		args = append(args, at(out+".json"))
		rc = runTo(at(out+".out"), at(out+".out"), surveyor, args...)
	}

	expect := func(name string, wantRC int, wantVerdict, wantCount, out string) {
		if rc != wantRC {
			h.bad(fmt.Sprintf("%s  expected exit %d, got %d: %s", name, wantRC, rc, headJoin(fileLines(at(out+".out")), 2)))
			return
		}
		d := jsonDoc(at(out + ".json"))
		gotVerdict := firstOf(d, "assessments", "staleness")
		gotCount := lenOf(d, "requests")
		if gotVerdict != wantVerdict {
			h.bad(fmt.Sprintf("%s  expected verdict %s, got %s", name, wantVerdict, orNone(gotVerdict)))
			return
		}
		if gotCount != wantCount {
			h.bad(fmt.Sprintf("%s  expected %s request(s), got %s", name, wantCount, orNone(gotCount)))
			return
		}
		h.ok(fmt.Sprintf("%s  exit %d, %s, %s request(s)", name, wantRC, wantVerdict, wantCount))
	}

	store := at("store")
	if !makeStore(store, "1.2.3", "1.2.10") {
		return verdict{}, errCannotRun("cannot build the fixture store")
	}
	empty := at("empty")
	if os.MkdirAll(empty, 0o755) != nil {
		return verdict{}, errCannotRun("cannot make an empty store")
	}

	for _, spec := range [][]string{
		{"moved", "1.2.3", "1.2.10", "1.3.0"},
		{"known", "1.2.3", "1.2.10"},
		{"preview", "1.2.3", "1.2.10", "1.3.0-beta1", "1.4.0@prerelease"},
		{"behind", "1.2.3"},
		{"blocked", "1.2.10", "1.2"},
	} {
		if !makeResolution(spec[0], spec[1:]...) {
			return verdict{}, errCannotRun("cannot resolve " + spec[0])
		}
	}

	survey(store, "moved", "first")
	expect("new      a new stable release opens one request", 1, "stale", "1", "first")
	first := jsonDoc(at("first.json"))
	firstID := firstOf(first, "requests", "id")
	if state := firstOf(first, "requests", "state"); state == "opened" {
		h.ok("new      and the request is in the state a caller acts on")
	} else {
		h.bad(fmt.Sprintf("new      the request is %s, not opened", orAbsent(state)))
	}

	var buf []byte
	for _, p := range []string{"bit-ids/capture-request/1", "bit-ids/capture-requests/1",
		"fixture-client", "1.3.0", "stable", "linux"} {
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(p)))
		buf = append(buf, p...)
	}
	derived := fmt.Sprintf("request:sha256:%x", sha256.Sum256(buf))
	if firstID != "" && derived == firstID {
		h.ok("derived  an independent SHA-256 re-derives the request identifier")
	} else {
		h.bad(fmt.Sprintf("derived  independent derivation gives %s, the survey gives %s", derived, orNone(firstID)))
	}

	// The tracker a second run is handed: the first run's requests, by their
	// four identifying fields.
	var tracker []map[string]any
	if items, ok := first["requests"].([]any); ok {
		for _, it := range items {
			if m, ok := it.(map[string]any); ok {
				tracker = append(tracker, map[string]any{
					"target": m["target"], "version": m["version"],
					"channel": m["channel"], "platform": m["platform"],
				})
			}
		}
	}
	openBody, _ := json.MarshalIndent(tracker, "", "  ")
	if os.WriteFile(at("open.json"), openBody, 0o644) != nil {
		return verdict{}, errCannotRun("cannot write the tracker")
	}
	survey(store, "moved", "second", at("open.json"))
	expect("repeat   a second run over the same facts opens nothing", 0, "stale", "1", "second")
	second := jsonDoc(at("second.json"))
	secondID := firstOf(second, "requests", "id")
	secondState := firstOf(second, "requests", "state")
	if secondID == firstID && secondState == "already_open" {
		h.ok("repeat   one identifier across both runs, and it is already_open")
	} else {
		h.bad(fmt.Sprintf("repeat   id %s state %s, expected %s already_open", orNone(secondID), orNone(secondState), firstID))
	}

	survey(store, "moved", "third", at("open.json"))
	if rc != 0 {
		h.bad(fmt.Sprintf("repeat   the third run exited %d", rc))
	} else {
		a, b := jsonDoc(at("second.json")), jsonDoc(at("third.json"))
		if a != nil && b != nil {
			delete(a, "surveyed_at")
			delete(b, "surveyed_at")
		}
		if a != nil && reflect.DeepEqual(a, b) {
			h.ok("repeat   a third run is the second one's document apart from the clock")
		} else {
			h.bad("repeat   two runs over one tracker state produced different documents")
		}
	}

	survey(store, "preview", "preview")
	expect("preview  a preview by either signal opens nothing", 0, "current", "0", "preview")
	if sel := firstOf(jsonDoc(at("preview.json")), "assessments", "selected"); sel == "1.2.10" {
		h.ok("preview  and the selection is the newest stable, not the preview")
	} else {
		h.bad(fmt.Sprintf("preview  the survey selected %s, not 1.2.10", orNone(sel)))
	}

	survey(store, "known", "known")
	expect("known    a release already measured opens nothing", 0, "current", "0", "known")

	survey(empty, "moved", "fresh")
	expect("first    an unmeasured platform opens its first request", 1, "uncaptured", "1", "fresh")

	survey(store, "behind", "behind")
	expect("behind   a selection older than what is measured opens nothing", 0, "regressed", "0", "behind")

	survey(store, "blocked", "blocked")
	expect("blocked  a resolution that failed closed is reported", 0, "unresolved", "0", "blocked")

	// ⚠ Built in two steps: a nested literal's double brace is the shape
	// `check-placeholders` hunts for in this tree.
	held := map[string]string{"target": "fixture-client", "version": "1.2.11", "channel": "stable", "platform": "linux"}
	staleOpen, _ := json.MarshalIndent([]map[string]string{held}, "", "  ")
	if os.WriteFile(at("stale-open.json"), staleOpen, 0o644) != nil {
		return verdict{}, errCannotRun("cannot write the stale tracker")
	}
	survey(store, "moved", "retired", at("stale-open.json"))
	expect("retired  a request the release moved past is superseded", 1, "stale", "1", "retired")
	retired := jsonDoc(at("retired.json"))
	retiredN := lenOf(retired, "superseded")
	retiredVersion := firstOf(retired, "requests", "version")
	if retiredN == "1" && retiredVersion == "1.3.0" {
		h.ok("retired  one identifier retired, and the request asks for 1.3.0")
	} else {
		h.bad(fmt.Sprintf("retired  %s retired, request at %s", orNone(retiredN), orNone(retiredVersion)))
	}

	if os.WriteFile(at("broken-open.json"), []byte("not json\n"), 0o644) != nil {
		return verdict{}, errCannotRun("cannot write the broken tracker")
	}
	survey(store, "moved", "broken", at("broken-open.json"))
	if rc == 2 {
		h.ok("tracker  an unreadable tracker is exit 2, not an empty tracker")
	} else {
		h.bad(fmt.Sprintf("tracker  expected exit 2 over an unreadable tracker, got %d", rc))
	}

	if rc := runTo(at("orphan.out"), at("orphan.out"), surveyor, store, "--platforms", "linux", at("orphan.json")); rc == 2 {
		h.ok("args     --platforms with no --resolution before it is refused")
	} else {
		h.bad(fmt.Sprintf("args     expected exit 2, got %d: %s", rc, headJoin(fileLines(at("orphan.out")), 2)))
	}

	if rc := runTo(null, null, surveyor, store, "--resolution", at("moved.releases.json"), "--platforms", "linux",
		at("notaresolution.json")); rc == 2 {
		h.ok("args     a document that is not a resolution is refused")
	} else {
		h.bad(fmt.Sprintf("args     expected exit 2 over a release list, got %d", rc))
	}

	// ⚠ The reader's own guards: it answers from the document, and answers
	// nothing for a key the document does not carry.
	if s, _ := first["schema"].(string); s == "bit-ids/capture-requests/1" {
		h.ok("probe    the harness reader answers from the document")
	} else {
		h.bad("probe    the harness reader cannot read the survey it just wrote")
	}
	if _, ok := first["no-such-key"]; !ok {
		h.ok("probe    and answers nothing for a key the document does not carry")
	} else {
		h.bad("probe    the harness reader invented a value")
	}

	return h.report("check-staleness/1", "cases"), nil
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func orAbsent(s string) string {
	if s == "" {
		return "absent"
	}
	return s
}
