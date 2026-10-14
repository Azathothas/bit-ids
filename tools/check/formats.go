// formats.go - render every published format over a store that carries a
// correction, compare two renders as bytes, check that the corrected record is
// not published as a record in any of them, and hand the assembled checksums
// and the database to readers this project did not write. `PUB-03`, `PUB-05`.
//
// Ported from scripts/publishing/check-formats.sh, `CI-07`, and compared case
// for case against it before that half was deleted. ⚠ It had no PowerShell
// twin, which is why the Windows lane declared this row.
//
// ⛔ TWO INDEPENDENT READERS, AND NEITHER IS THIS PROJECT'S. The checksums go to
// `sha256sum`, or to `certutil` where a host has no `sha256sum`, as
// `check-release` does. The database goes to Python's `sqlite3`, through the
// same script the shell half wrote, unchanged. ⚠ The CBOR is not decoded here,
// for the shell half's reason: this project has no CBOR reader, and its own
// encoder reading it back would be the writer checking itself. `cbor2` read it
// in `PUB-03`'s driven pass. ⚠ Two Rust comments said this check handed it to
// `cbor2`; neither half ever did, and both comments say so now.
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// readDBScript is the shell half's reader, byte for byte.
const readDBScript = `"""Read a rendered formats tree with a SQLite this project did not write."""

import json
import sqlite3
import sys

tree = sys.argv[1]
out = []


def case(ok, text):
    out.append(("PASS" if ok else "FAIL", text))


con = sqlite3.connect(tree + "/formats/bit-ids-v1.sqlite3")
case(
    con.execute("PRAGMA integrity_check").fetchone()[0] == "ok",
    "sqlite    integrity_check passes under python's sqlite3 %s" % sqlite3.sqlite_version,
)
case(not con.execute("PRAGMA foreign_key_check").fetchall(), "sqlite    every foreign key resolves")

meta = dict(con.execute("SELECT key, value FROM meta"))
combined = json.load(open(tree + "/formats/bit-ids-v1.json", encoding="utf-8"))
case(
    meta.get("records") == str(len(combined)),
    "sqlite    meta names the same record count the combined JSON has",
)
case(con.execute("SELECT count(*) FROM omission").fetchone()[0] > 0,
     "sqlite    the file says what it does not carry, in a table")

rows = dict(con.execute("SELECT record, canonical FROM document"))
by_id = {record["id"]: record for record in combined}
case(set(rows) == set(by_id), "sqlite    it publishes exactly the record set the JSON rendering does")
case(
    all(json.loads(rows[key]) == by_id[key] for key in by_id if key in rows),
    "sqlite    every stored document round-trips to the JSON rendering's record",
)

# ⛔ The columns are compared against the document they were derived from,
# because a schema that dropped a section would still round-trip ` + "`" + `document` + "`" + `.
agree = True
counted = 0
for key, canonical in rows.items():
    record = json.loads(canonical)
    row = con.execute(
        "SELECT schema, target, target_kind, version, platform, arch, package,"
        " executable, capture, captured_at, supersedes FROM record WHERE id = ?",
        (key,),
    ).fetchone()
    want = (
        record["schema"],
        record["target"]["id"],
        record["target"]["kind"],
        record["build"]["version"],
        record["build"]["platform"],
        record["build"]["arch"],
        record["build"]["package"],
        record["build"]["executable"],
        record["capture"]["id"],
        record["capture"]["captured_at"],
        record.get("supersedes"),
    )
    agree = agree and row == want
    for table, section in (
        ("acquisition", "acquisition"),
        ("observation", "observations"),
        ("corroboration", "corroboration"),
        ("normalization", "normalizations"),
        ("evidence", "evidence"),
    ):
        held = con.execute(
            "SELECT count(*) FROM %s WHERE record = ?" % table, (key,)
        ).fetchone()[0]
        agree = agree and held == len(record[section])
        counted += held
case(agree, "sqlite    every column and every list agrees with the document it came from")

# ⚠ A count nothing compares is a count worth nothing: the tables the CSV cannot
# hold have to be non-empty, or the case above passes over a database of zeroes.
case(counted > 0, "sqlite    the sections the table cannot hold are tabulated here, and are not empty")

for status, text in out:
    print("%s %s" % (status, text))
sys.exit(0 if all(status == "PASS" for status, _ in out) else 1)
`

// python3 finds a Python 3 under either name a host may give it. ⚠ A Windows
// runner carries `python` and may carry no `python3`, and a `python` that is
// Python 2 is not this reader.
func python3() string {
	for _, name := range []string{"python3", "python"} {
		p, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if exec.Command(p, "-c", "import sys; sys.exit(0 if sys.version_info[0] == 3 else 1)").Run() == nil {
			return p
		}
	}
	return ""
}

func checkFormats(r *repo) (verdict, error) {
	h := &harness{me: "check-formats"}

	builder, err := exampleBinary(r.root, "build-store", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	renderer, err := exampleBinary(r.root, "build-formats", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	assembler, err := exampleBinary(r.root, "assemble-release", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	py := python3()
	if py == "" {
		return verdict{}, errCannotRun("python3 not found; the database's independent reader is Python's sqlite3")
	}
	if sumsReader() == "" {
		return verdict{}, errCannotRun("neither sha256sum nor certutil is on this host, and this project reading its own checksums proves nothing")
	}

	work, err := scratch("checkformats")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	store := filepath.Join(work, "store")
	first := filepath.Join(work, "first")
	second := filepath.Join(work, "second")
	const scheme = "fixture-client:-:3:3"
	csvRel := filepath.Join("formats", "bit-ids-v1.csv")
	sqliteRel := filepath.Join("formats", "bit-ids-v1.sqlite3")
	combinedRel := filepath.Join("formats", "bit-ids-v1.json")
	null := filepath.Join(work, "null")
	for _, d := range []string{store, first, second} {
		if os.MkdirAll(d, 0o755) != nil {
			return verdict{}, errCannotRun("cannot make " + d)
		}
	}
	if runTo(null, null, builder, "--version", "1.2.3", "--version", "1.3.0", "--correct", "1.3.0", store) != 0 {
		return verdict{}, errCannotRun("cannot build the fixture store")
	}

	firstOut := filepath.Join(work, "first.out")
	firstRC := runTo(firstOut, firstOut, renderer, "--scheme", scheme, store, first)
	secondRC := runTo(null, null, renderer, "--scheme", scheme, store, second)

	if firstRC == 0 {
		h.ok("clean    the store renders and every file reads back")
	} else {
		h.bad(fmt.Sprintf("clean    the render was refused (exit %d): %s", firstRC, headJoin(fileLines(firstOut), 3)))
	}

	switch {
	case secondRC != 0:
		h.bad(fmt.Sprintf("determinism  the second render failed (exit %d)", secondRC))
	case treeDigest(first) == treeDigest(second):
		h.ok("determinism  two renders of one store are byte-identical")
	default:
		h.bad("determinism  two renders of one store differ")
	}

	if n := treeFiles(first); n == 6 {
		h.ok("determinism  six files were rendered, so the comparison had something to compare")
	} else {
		h.bad(fmt.Sprintf("determinism  %d file(s) rendered, expected 6", n))
	}

	a, _ := os.ReadFile(filepath.Join(first, sqliteRel))
	b, _ := os.ReadFile(filepath.Join(second, sqliteRel))
	if len(a) > 0 && bytes.Equal(a, b) {
		h.ok("determinism  two builds of the database are byte-identical")
	} else {
		h.bad("determinism  the database differs between two builds of one store")
	}

	// -- the record set ---------------------------------------------------------
	idOf := func(path string) string {
		body, _ := os.ReadFile(path)
		if m := regexp.MustCompile(`"id": "(record:sha256:[0-9a-f]*)"`).FindSubmatch(body); m != nil {
			return string(m[1])
		}
		return ""
	}
	originalID := idOf(findNamed(filepath.Join(store, "profiles"), "cap-1-3-0.json"))
	fixedID := idOf(findNamed(filepath.Join(store, "profiles"), "cap-1-3-0-fix.json"))
	if originalID == "" || fixedID == "" || originalID == fixedID {
		h.bad("records  could not read two distinct identifiers out of the store")
	} else {
		published := map[string]bool{}
		for i, l := range fileLines(filepath.Join(first, csvRel)) {
			if i == 0 {
				continue
			}
			id, _, _ := strings.Cut(strings.TrimRight(l, "\r"), ",")
			published[id] = true
		}
		if published[originalID] {
			h.bad("records  the table publishes the corrected record")
		} else {
			h.ok("records  the table does not publish the corrected record")
		}
		if published[fixedID] {
			h.ok("records  the table publishes the correction")
		} else {
			h.bad("records  the table does not publish the correction")
		}
		combined := fileLines(filepath.Join(first, combinedRel))
		if anyContains(combined, `"id": "`+originalID+`"`) {
			h.bad("records  the combined document publishes the corrected record")
		} else {
			h.ok("records  the combined document does not publish it either")
		}
		if anyContains(combined, `"supersedes": "`+originalID+`"`) {
			h.ok("records  and the correction still names what it corrects")
		} else {
			h.bad("records  the correction does not name what it corrects")
		}
	}

	csvBody, _ := os.ReadFile(filepath.Join(first, csvRel))
	omitted := 0
	for _, section := range []string{"acquisition", "observations", "corroboration", "normalizations", "evidence"} {
		if bytes.Contains(csvBody, []byte(section)) {
			omitted++
		}
	}
	if omitted == 0 {
		h.ok("tabular  the table carries none of the sections it declares it omits")
	} else {
		h.bad(fmt.Sprintf("tabular  %d declared-omitted section(s) appear in the table", omitted))
	}

	// -- the release ------------------------------------------------------------
	if copyFile(filepath.Join(r.root, "LICENSE"), filepath.Join(first, "LICENSE")) != nil {
		return verdict{}, errCannotRun("cannot copy the licence")
	}
	assembleOut := filepath.Join(work, "assemble.out")
	if rc := runTo(assembleOut, assembleOut, assembler, first); rc != 0 {
		h.bad(fmt.Sprintf("release  the rendered tree would not assemble (exit %d): %s", rc, headJoin(fileLines(assembleOut), 3)))
	} else {
		h.ok("release  the rendered tree assembles and the manifest covers it")

		manifest, _ := os.ReadFile(filepath.Join(first, "MANIFEST.json"))
		missing := 0
		for _, name := range []string{"bit-ids-v1.cbor", "bit-ids-v1.columns.json", "bit-ids-v1.csv",
			"bit-ids-v1.json", "bit-ids-v1.jsonl", "bit-ids-v1.sqlite3"} {
			if !bytes.Contains(manifest, []byte("formats/"+name)) {
				missing++
			}
		}
		if missing == 0 {
			h.ok("release  every rendering has a manifest row")
		} else {
			h.bad(fmt.Sprintf("release  %d rendering(s) have no manifest row", missing))
		}

		if verifySums(first) {
			h.ok("release  sha256sum -c agrees with every published digest")
		} else {
			h.bad("release  sha256sum -c disagrees")
		}

		f, err := os.OpenFile(filepath.Join(first, csvRel), os.O_APPEND|os.O_WRONLY, 0)
		if err == nil {
			_, _ = f.WriteString("x")
			_ = f.Close()
		}
		if !verifySums(first) {
			h.ok("release  and refuses a rendering whose bytes moved")
		} else {
			h.bad("release  sha256sum -c accepted a file that had changed")
		}
	}

	// -- the database, read by a SQLite this project did not write -------------
	script := filepath.Join(work, "read-db.py")
	if os.WriteFile(script, []byte(readDBScript), 0o644) != nil {
		return verdict{}, errCannotRun("cannot write the database reader")
	}
	readOut := filepath.Join(work, "read-db.out")
	readRC := runTo(readOut, readOut, py, script, first)
	readLines := fileLines(readOut)
	if readRC == 2 || len(readLines) == 0 {
		h.bad(fmt.Sprintf("sqlite    the independent reader could not run (exit %d): %s", readRC, headJoin(readLines, 3)))
	} else {
		for _, l := range readLines {
			if rest, ok := strings.CutPrefix(l, "PASS "); ok {
				h.ok(rest)
			} else {
				h.bad(strings.TrimPrefix(l, "FAIL "))
			}
		}
	}

	// ⛔ AND THE READER IS SHOWN TO REFUSE, over a database cut 4096 bytes short.
	planted := filepath.Join(work, "planted")
	if os.MkdirAll(filepath.Join(planted, "formats"), 0o755) != nil ||
		copyFile(filepath.Join(first, combinedRel), filepath.Join(planted, combinedRel)) != nil {
		return verdict{}, errCannotRun("cannot make the truncated-database tree")
	}
	db, _ := os.ReadFile(filepath.Join(first, sqliteRel))
	if len(db) <= 4096 || os.WriteFile(filepath.Join(planted, sqliteRel), db[:len(db)-4096], 0o644) != nil {
		return verdict{}, errCannotRun("cannot truncate the database")
	}
	if rc := runTo(null, null, py, script, planted); rc == 0 {
		h.bad("sqlite    the independent reader accepted a truncated database")
	} else {
		h.ok(fmt.Sprintf("sqlite    and it refuses a database whose bytes were truncated (exit %d)", rc))
	}

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

	return h.report("check-formats/1", "cases"), nil
}
