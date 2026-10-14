// corpus.go - plant, in a real store on a disk, every defect the semantic corpus
// validator exists to refuse, and read the exit code from the process that
// produced it.
//
// `CORPUS-01` asks whether a successor may replace a predecessor. This asks the
// other question: is a tree a coherent corpus at all. ⛔ The one that matters
// most is a citation nothing resolves. `bind` compares the profile against the
// manifest, so a run that agreed with itself about an artifact nobody wrote
// satisfied every check this project had before `CORPUS-02`.
//
// Ported from scripts/corpus/check-corpus.sh, `CI-07` and `CI-10`, and compared
// case for case against it before that half was deleted. ⚠ The shell half was
// the only one; it had no PowerShell twin, which is why the Windows lane
// declared this row and why porting it is what closes that row.
//
// -- ⛔ THE STORE IS BUILT, NOT COMMITTED ---------------------------------------
//
// The schema fixtures declare digests for artifacts that were never written, so
// they are not a corpus. `build-store` writes the artifacts and rewrites each
// document to describe the bytes it put down. That is the clean case, and it
// has to be clean before any refusal below is attributable.
//
// -- ⛔ IT VERIFIES ITS OWN EDITS APPLIED -----------------------------------------
//
// Every plant is bracketed by a digest of the whole tree, a literal replacement
// must match exactly once, and a plant that did not move the tree is reported
// NOT-PLANTED and counted as a failure rather than as a refusal that did not
// come.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// treeDigest is store-lib's `tree_digest`: one digest over every file's path
// and bytes, with a link or a pipe named rather than followed.
func treeDigest(dir string) string {
	var lines []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = "./" + filepath.ToSlash(rel)
		switch t := d.Type(); {
		case t&fs.ModeSymlink != 0:
			lines = append(lines, rel+" symlink")
		case t&fs.ModeNamedPipe != 0:
			lines = append(lines, rel+" fifo")
		default:
			b, rerr := os.ReadFile(p)
			if rerr != nil {
				lines = append(lines, rel+" unreadable")
				return nil
			}
			lines = append(lines, fmt.Sprintf("%s %x", rel, sha256.Sum256(b)))
		}
		return nil
	})
	sort.Strings(lines)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(lines, "\n"))))
}

// treeFiles is `tree_files`: every entry that is not a directory.
func treeFiles(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

// copyFile copies one file's bytes, which is all `place` needs.
func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func checkCorpus(r *repo) (verdict, error) {
	h := &harness{me: "check-corpus"}
	fixtures := filepath.Join(r.root, "crates", "bit-ids", "tests", "fixtures")

	builder, err := exampleBinary(r.root, "build-store", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	validator, err := exampleBinary(r.root, "validate-corpus", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	locator, err := exampleBinary(r.root, "check-store", "bit-ids")
	if err != nil {
		return verdict{}, err
	}

	work, err := scratch("checkcorpus")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	store := filepath.Join(work, "store")
	inStore := func(rel string) string { return filepath.Join(store, filepath.FromSlash(rel)) }

	// Paths inside the store, asked for rather than spelled here.
	where := func(record string) (string, error) {
		var out bytes.Buffer
		cmd := exec.Command(locator, "--where", record)
		cmd.Stdout = &out
		if cmd.Run() != nil {
			return "", errCannotRun("check-store --where refused " + filepath.Base(record))
		}
		rel := strings.TrimSpace(out.String())
		if rel == "" {
			return "", errCannotRun("check-store --where named no path for " + filepath.Base(record))
		}
		return rel, nil
	}
	profileRel, err := where(filepath.Join(fixtures, "valid-profile.json"))
	if err != nil {
		return verdict{}, err
	}
	manifestRel, err := where(filepath.Join(fixtures, "valid-manifest.json"))
	if err != nil {
		return verdict{}, err
	}
	bundleRel := path.Dir(manifestRel)

	null := filepath.Join(work, "null")
	buildStore := func() bool {
		if os.RemoveAll(store) != nil || os.MkdirAll(store, 0o755) != nil {
			return false
		}
		return runTo(null, null, builder, store) == 0
	}

	// ⛔ The validator's two streams are read together, as the shell half's
	// `2>&1` did, so a code on either one is found.
	outPath := filepath.Join(work, "out")
	errPath := filepath.Join(work, "err")
	var out []string
	runCheck := func() int {
		rc := runTo(outPath, errPath, validator, store)
		out = append(fileLines(outPath), fileLines(errPath)...)
		return rc
	}

	if !buildStore() {
		return verdict{}, errCannotRun("cannot build the fixture store")
	}

	// ⛔ The clean case first. A harness whose clean case is not green is
	// measuring its own setup.
	if rc := runCheck(); rc == 0 {
		h.ok("clean    a record, its run, and every artifact they cite")
	} else {
		h.bad(fmt.Sprintf("clean    the unmutated store was refused (exit %d): %s", rc, headJoin(out, 3)))
	}

	// ⚠ The clean store's shape is pinned, because every case below takes one
	// thing away from it.
	if n := treeFiles(store); n == 11 {
		h.ok("clean    the store holds 11 objects: one record, one run, nine artifacts")
	} else {
		h.bad(fmt.Sprintf("clean    the store holds %d objects, expected 11", n))
	}

	plant := func(name, code string, mutate func() bool) {
		if !buildStore() {
			h.bad(name + "  could not build the store")
			return
		}
		before := treeDigest(store)
		if !mutate() {
			h.bad(name + "  NOT-PLANTED (the mutation reported it did not apply)")
			return
		}
		if before == treeDigest(store) {
			h.bad(name + "  NOT-PLANTED (the tree digest did not move)")
			return
		}
		rc := runCheck()
		if rc != 1 {
			h.bad(fmt.Sprintf("%s  expected exit 1, got %d", name, rc))
			return
		}
		if !anyContains(out, code) {
			h.bad(fmt.Sprintf("%s  refused, but not as %s: %s", name, code, headJoin(out, 3)))
			return
		}
		if !buildStore() {
			h.bad(name + "  could not restore the store")
			return
		}
		if rc := runCheck(); rc != 0 {
			h.bad(fmt.Sprintf("%s  the restored store is not clean (exit %d)", name, rc))
			return
		}
		h.ok(code + "  " + name)
	}
	remove := func(rel string) func() bool {
		return func() bool { return os.Remove(inStore(rel)) == nil }
	}
	events := path.Join(bundleRel, "observer", "events.jsonl")

	plant("a record with no run", "E-CRP-01", remove(manifestRel))
	plant("a run no record cites", "E-CRP-02", remove(profileRel))
	// ⛔ The invariant this entry exists for. Both documents still agree about
	// the artifact; the bytes are simply not there.
	plant("a citation nothing resolves", "E-CRP-03", remove(events))
	plant("an artifact longer than declared", "E-CRP-04", func() bool {
		f, err := os.OpenFile(inStore(events), os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			return false
		}
		_, werr := f.WriteString("appended\n")
		return f.Close() == nil && werr == nil
	})
	// ⚠ Same length, different bytes, so this reaches the digest comparison
	// rather than the length one.
	plant("an artifact edited without changing its length", "E-CRP-05", func() bool {
		return replaceOnce(inStore(events), "artifact", "artifacX")
	})
	plant("a file the run does not declare", "E-CRP-06", func() bool {
		return os.WriteFile(inStore(path.Join(bundleRel, "stray.log")), []byte("not declared anywhere\n"), 0o644) == nil
	})
	// A correction whose target is not in the store: a chain with a hole in it.
	plant("a correction naming an absent record", "E-CRP-07", func() bool {
		record := filepath.Join(fixtures, "valid-correction.json")
		rel, err := where(record)
		if err != nil || os.MkdirAll(filepath.Dir(inStore(rel)), 0o755) != nil ||
			copyFile(record, inStore(rel)) != nil {
			return false
		}
		return os.Remove(inStore(profileRel)) == nil
	})
	// The store's own placement rule, reached through the corpus reader rather
	// than through check-store, because a rule enforced on one of two paths into
	// the same action is the hole this project keeps finding.
	plant("a record filed at another platform", "E-STO-30", func() bool {
		wrong := "profiles/v1/fixture-client/0.0.0-fixture/windows/x86-64/tar-gz/fixture-capture-0001.json"
		return os.MkdirAll(filepath.Dir(inStore(wrong)), 0o755) == nil &&
			os.Rename(inStore(profileRel), inStore(wrong)) == nil
	})

	_ = buildStore()
	h.probeGuards(inStore(profileRel), "fixture-client", "sha256:")

	return h.report("check-corpus/1", "cases"), nil
}
