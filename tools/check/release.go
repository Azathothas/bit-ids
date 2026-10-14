// release.go - assemble a release twice, compare the bytes, and hand the
// checksum file to a reader this project did not write. `PUB-01`.
//
// Ported from scripts/publishing/check-release.sh, `CI-07`, and compared case
// for case against it before that half was deleted. ⚠ It had no PowerShell
// twin, which is why the Windows lane declared this row.
//
// ⛔ THE CHECKSUMS ARE READ BY A PROGRAM NOBODY HERE WROTE, ON BOTH LANES. The
// shell half handed them to `sha256sum -c`, so a manifest that agreed with its
// own writer was still caught. A Windows runner carries no `sha256sum` on the
// path a non-bash step sees, and it does carry `certutil`, which is Windows'
// own: this asks `sha256sum` where it exists and `certutil -hashfile` for each
// row where it does not. ⚠ Reading the file with this binary's own SHA-256
// would be the writer checking itself, and is the one thing not done.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// sumsReader names the independent reader this host has, or nothing.
func sumsReader() string {
	for _, r := range []string{"sha256sum", "certutil"} {
		if _, err := exec.LookPath(r); err == nil {
			return r
		}
	}
	return ""
}

// verifySums answers whether that reader agrees with every row of a tree's
// SHA256SUMS.
func verifySums(dir string) bool {
	switch sumsReader() {
	case "sha256sum":
		cmd := exec.Command("sha256sum", "-c", "SHA256SUMS", "--quiet")
		cmd.Dir = dir
		return cmd.Run() == nil
	case "certutil":
	default:
		return false
	}
	f, err := os.Open(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return false
	}
	defer f.Close()
	hex := regexp.MustCompile(`(?i)^[0-9a-f]{64}$`)
	rows := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		want, name, ok := strings.Cut(line, "  ")
		if !ok || !hex.MatchString(want) {
			return false
		}
		var out bytes.Buffer
		cmd := exec.Command("certutil", "-hashfile", filepath.Join(dir, filepath.FromSlash(name)), "SHA256")
		cmd.Stdout = &out
		if cmd.Run() != nil {
			return false
		}
		got := ""
		for _, l := range strings.Split(out.String(), "\n") {
			l = strings.ReplaceAll(strings.TrimSpace(l), " ", "")
			if hex.MatchString(l) {
				got = l
				break
			}
		}
		if !strings.EqualFold(got, want) {
			return false
		}
		rows++
	}
	return sc.Err() == nil && rows > 0
}

func checkRelease(r *repo) (verdict, error) {
	h := &harness{me: "check-release"}

	builder, err := exampleBinary(r.root, "build-store", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	indexer, err := exampleBinary(r.root, "build-indexes", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	assembler, err := exampleBinary(r.root, "assemble-release", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	if sumsReader() == "" {
		return verdict{}, errCannotRun("neither sha256sum nor certutil is on this host, and this project reading its own checksums proves nothing")
	}

	work, err := scratch("checkrelease")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	tree := filepath.Join(work, "tree")
	null := filepath.Join(work, "null")
	licence := filepath.Join(r.root, "LICENSE")

	buildTree := func() bool {
		if os.RemoveAll(tree) != nil || os.MkdirAll(filepath.Join(tree, "indexes", "v1"), 0o755) != nil {
			return false
		}
		if runTo(null, null, builder, "--version", "1.2.3", "--version", "1.2.10", tree) != 0 {
			return false
		}
		if runTo(null, null, indexer, "--scheme", "fixture-client:-:3:3", tree,
			filepath.Join(tree, "indexes", "v1", "profiles.json")) != 0 {
			return false
		}
		return copyFile(licence, filepath.Join(tree, "LICENSE")) == nil
	}

	outPath := filepath.Join(work, "out")
	errPath := filepath.Join(work, "err")
	var out []string
	assemble := func(dir string) int {
		rc := runTo(outPath, errPath, assembler, dir)
		out = append(fileLines(outPath), fileLines(errPath)...)
		return rc
	}
	runAssemble := func() int { return assemble(tree) }

	if !buildTree() {
		return verdict{}, errCannotRun("cannot build the fixture tree")
	}

	if rc := runAssemble(); rc == 0 {
		h.ok("clean    a store, its indexes and a licence assemble and read back")
	} else {
		h.bad(fmt.Sprintf("clean    the unmutated tree was refused (exit %d): %s", rc, headJoin(out, 3)))
	}

	// -- determinism ----------------------------------------------------------
	_ = buildTree()
	_ = runAssemble()
	firstManifest, _ := os.ReadFile(filepath.Join(tree, "MANIFEST.json"))
	firstSums, _ := os.ReadFile(filepath.Join(tree, "SHA256SUMS"))
	_ = buildTree()
	_ = runAssemble()
	secondManifest, _ := os.ReadFile(filepath.Join(tree, "MANIFEST.json"))
	secondSums, _ := os.ReadFile(filepath.Join(tree, "SHA256SUMS"))
	if len(firstManifest) > 0 && bytes.Equal(firstManifest, secondManifest) && bytes.Equal(firstSums, secondSums) {
		h.ok("determinism  two assemblies of one input are byte-identical")
	} else {
		h.bad("determinism  two assemblies differ")
	}

	// ⚠ `grep -c` counts LINES holding the word, as the shell half did.
	described := 0
	for _, l := range strings.Split(string(firstManifest), "\n") {
		if strings.Contains(l, `"path"`) {
			described++
		}
	}
	if described >= 20 {
		h.ok(fmt.Sprintf("determinism  the manifest describes %d files, so the comparison had something to compare", described))
	} else {
		h.bad(fmt.Sprintf("determinism  the manifest describes only %d files", described))
	}

	if verifySums(tree) {
		h.ok("checksums  sha256sum -c agrees with every row, MANIFEST.json included")
	} else {
		h.bad("checksums  sha256sum -c disagreed")
	}

	// -- what is refused ------------------------------------------------------
	output := func() []string { return out }
	plant := func(name, code string, mutate func() bool) {
		plantCase(h, name, code, tree, buildTree, mutate, runAssemble, output)
	}
	write := func(rel, body string) func() bool {
		return func() bool {
			return os.WriteFile(filepath.Join(tree, filepath.FromSlash(rel)), []byte(body), 0o644) == nil
		}
	}
	plant("a file with no known media type", "E-REL-01", write("notes.unknown", "notes\n"))
	plant("a symbolic link in a release", "E-REL-02", func() bool {
		return os.Symlink(filepath.Join(tree, "LICENSE"), filepath.Join(tree, "indexes", "v1", "link.json")) == nil
	})
	plant("a published file with no bytes", "E-REL-03", write("indexes/v1/empty.json", ""))
	plant("a tree already carrying a manifest", "E-REL-04", write("MANIFEST.json", "{\"stale\": true}\n"))

	empty := filepath.Join(work, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		return verdict{}, errCannotRun("cannot make an empty tree")
	}
	if rc := assemble(empty); rc == 1 && anyContains(out, "E-REL-05") {
		h.ok("E-REL-05  an empty release")
	} else {
		h.bad(fmt.Sprintf("E-REL-05  expected exit 1 with E-REL-05, got %d: %s", rc, headJoin(out, 2)))
	}

	_ = buildTree()
	h.probeGuards(filepath.Join(tree, "LICENSE"), "Permission", "e")

	return h.report("check-release/1", "cases"), nil
}
