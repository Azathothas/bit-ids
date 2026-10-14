// catalogue.go - drive LIB-01's consumer library against a real publication,
// and plant every defect it exists to refuse.
//
// Ported from scripts/publishing/check-catalogue.sh and its PowerShell twin,
// CI-10. ⚠ Both halves drove the same five Rust examples over the same fixture
// publication, so their answers were identical by construction; what differed
// was how each PLANTED a defect and restored the publication afterwards, and one
// implementation cannot differ from itself there.
//
// ⛔ THE THIRD CLAUSE OF THE PROVE IS THE ONE A SEARCH ANSWERS BETTER THAN A RUN.
// "without network access" cannot be established by observing that a run did not
// use the network: a run that happened not to need one proves nothing about the
// next. What can be established is that the crate has no way to reach one, so
// this sweeps its source for every socket and HTTP constructor and reads its
// dependency list.
//
// ⚠ AND THE SWEEP'S OWN NEEDLE LIST IS THE PART THAT ROTS. `OBS-06`'s finding
// was that every needle named a constructor while a send is a method on a socket
// that already exists. The list is checked against the crate that really does
// carry sockets, so a sweep that had stopped matching anything is visible.
//
// ⛔ THE NEEDLES ARE CASE-SENSITIVE LITERALS, as the `sh` half's `grep` was. They
// are Rust paths and type names, and a language that distinguishes `TcpStream`
// from `tcpstream` is the one being searched.
//
// -- ⛔ THE ORDERING CASE IS THE ONE A TEXT SORT GETS WRONG ------------------
//
// The store carries 1.2.3 and 1.2.10. As text 1.2.3 sorts last, and a library
// that answered it would point a consumer at a superseded build with confidence.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// catalogueNeedles is the `sh` half's alternation, one literal per branch.
var catalogueNeedles = []string{
	"std::net", "TcpStream", "TcpListener", "UdpSocket", "SocketAddr",
	"reqwest", "ureq", "hyper::", "curl::", ".connect(", "to_socket_addrs",
}

// catalogueTransports is the `^(...)` of the manifest check: a dependency line
// beginning with one of these names a transport.
// ⚠ `socket2` since 2026-10-14: it is in the lockfile for the lab's one door,
// and a lockfile that already carries a crate no longer refuses it by itself.
var catalogueTransports = []string{"reqwest", "ureq", "hyper", "curl", "tokio", "async-std", "isahc", "socket2"}

func checkCatalogue(r *repo) (verdict, error) {
	h := &harness{me: "check-catalogue"}

	bins := map[string]string{}
	for _, ex := range []string{"build-store", "build-indexes", "build-formats", "assemble-release", "catalogue-lookup"} {
		b, err := exampleBinary(r.root, ex, "bit-ids")
		if err != nil {
			return verdict{}, err
		}
		bins[ex] = b
	}

	work, err := scratch("checkcatalogue")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)

	bundle := filepath.Join(work, "bundle")
	index := filepath.Join(bundle, "indexes", "v1", "profiles.json")
	const scheme = "fixture-client:-:3:3"
	outPath := filepath.Join(work, "out")
	errPath := filepath.Join(work, "err")

	build := func() bool {
		_ = os.RemoveAll(bundle)
		if os.MkdirAll(filepath.Dir(index), 0o755) != nil {
			return false
		}
		if runQuiet(bins["build-store"], "--version", "1.2.3", "--version", "1.2.10", bundle) != 0 {
			return false
		}
		if runQuiet(bins["build-indexes"], "--scheme", scheme, bundle, index) != 0 {
			return false
		}
		if runQuiet(bins["build-formats"], "--scheme", scheme, bundle, bundle) != 0 {
			return false
		}
		lic, err := os.ReadFile(filepath.Join(r.root, "LICENSE"))
		if err != nil || os.WriteFile(filepath.Join(bundle, "LICENSE"), lic, 0o644) != nil {
			return false
		}
		return runQuiet(bins["assemble-release"], bundle) == 0
	}

	// ⛔ Unpiped. Output to a file, the exit code from the process that gave it.
	ask := func(question ...string) int {
		return runTo(outPath, errPath, bins["catalogue-lookup"], append([]string{bundle}, question...)...)
	}

	if !build() {
		return verdict{}, errCannotRun("cannot build the fixture publication")
	}
	sums, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		return verdict{}, errCannotRun("cannot build the fixture publication")
	}
	sum := sha256.Sum256(sums)
	digest := "sha256:" + hex.EncodeToString(sum[:])

	rc := ask("latest", "fixture-client", "linux", "x86-64", "tar-gz")
	if rc == 0 {
		h.ok("clean    a real publication opens and answers")
	} else {
		h.bad(fmt.Sprintf("clean    the publication was refused (exit %d): %s", rc, headJoin(fileLines(errPath), 3)))
	}

	// ⛔ THE ORDERING. As text, 1.2.3 sorts after 1.2.10.
	ordered := false
	for _, l := range fileLines(outPath) {
		if strings.HasSuffix(l, " 1.2.10") {
			ordered = true
		}
	}
	if ordered {
		h.ok("ordering  latest answers 1.2.10 over 1.2.3")
	} else {
		h.bad("ordering  latest answered " + headJoin(fileLines(outPath), len(fileLines(outPath))))
	}

	// ⚠ A build line the catalogue does not carry answers nothing rather than the
	// nearest one, and the exit code says so.
	rc = ask("latest", "fixture-client", "windows", "x86-64", "tar-gz")
	if rc == 1 && fileSize(outPath) == 0 {
		h.ok("absent   an unmeasured build line answers nothing, with exit 1")
	} else {
		h.bad(fmt.Sprintf("absent   exit %d with %d row(s)", rc, newlines(outPath)))
	}

	rc = ask("lookup", "target", "fixture-client")
	if n := nonEmpty(fileLines(outPath)); rc == 0 && n == 2 {
		h.ok("lookup   the target index answers both published records")
	} else {
		h.bad(fmt.Sprintf("lookup   exit %d with %d row(s)", rc, n))
	}

	// ⛔ THE OUT-OF-BAND DIGEST, WHICH IS THE ONE FILE NOTHING IN A PUBLICATION
	// PROVES. Both branches are cases: the right digest is accepted and a wrong
	// one is refused, because a check that only ever passes is not a check.
	rc = ask("--expect-checksums", digest, "lookup", "target", "fixture-client")
	if rc == 0 && anyContains(fileLines(errPath), "verified against the caller's digest") {
		h.ok("outofband  the caller's digest is checked and the summary says so")
	} else {
		h.bad(fmt.Sprintf("outofband  exit %d: %s", rc, headJoin(fileLines(errPath), 2)))
	}

	// ⚠ COMPUTED, NEVER TYPED: a run of sixty-four hex digits in a tracked file
	// is the shape `check-no-secrets --public` refuses.
	wrong := "sha256:" + strings.Repeat("0", 64)
	rc = ask("--expect-checksums", wrong, "lookup", "target", "fixture-client")
	if rc == 1 && anyContains(fileLines(errPath), "E-LIB-05") {
		h.ok("E-LIB-05  a checksum file the caller did not expect is refused")
	} else {
		h.bad(fmt.Sprintf("E-LIB-05  expected exit 1 with E-LIB-05, got %d", rc))
	}

	// ⚠ And a run with no digest says it trusted the file, rather than reading as
	// though it had verified it.
	ask("lookup", "target", "fixture-client")
	if anyContains(fileLines(errPath), "checksums trusted") {
		h.ok("outofband  a run given no digest says the checksum file was trusted")
	} else {
		h.bad("outofband  the summary does not say which verification ran")
	}

	// ⛔ THE PLAN, AND BOTH CLASSES NON-EMPTY. A plan that is all one class
	// classified nothing.
	rc = ask("plan")
	planned := fileLines(outPath)
	immutable, current, outOfBand := 0, 0, 0
	for _, l := range planned {
		if strings.HasPrefix(l, "immutable ") {
			immutable++
		}
		if strings.HasPrefix(l, "current ") {
			current++
		}
		if strings.Contains(l, " out_of_band ") {
			outOfBand++
		}
	}
	if rc == 0 && immutable > 0 && current > 0 {
		h.ok(fmt.Sprintf("plan     %d immutable and %d current path(s), so neither class is empty", immutable, current))
	} else {
		h.bad(fmt.Sprintf("plan     exit %d, %d immutable, %d current", rc, immutable, current))
	}
	if outOfBand == 1 {
		h.ok("plan     exactly one path is proved by neither document")
	} else {
		h.bad(fmt.Sprintf("plan     %d path(s) marked out_of_band", outOfBand))
	}

	// -- The refusals, planted one at a time ----------------------------------
	//
	// ⛔ EVERY PLANT IS RESTORED AND THE RESTORED PUBLICATION IS RE-ASKED. A
	// harness that planted six defects in a row would be measuring their
	// composition rather than each one.
	plantCase := func(name, code string) {
		if rc := ask("lookup", "target", "fixture-client"); rc != 1 {
			h.bad(fmt.Sprintf("%s  %s: expected exit 1, got %d", code, name, rc))
			return
		}
		if !anyContains(fileLines(errPath), code) {
			h.bad(fmt.Sprintf("%s  %s: refused, but not as %s: %s", code, name, code, headJoin(fileLines(errPath), 2)))
			return
		}
		if !build() {
			h.bad(fmt.Sprintf("%s  %s: could not restore the publication", code, name))
			return
		}
		if rc := ask("lookup", "target", "fixture-client"); rc != 0 {
			h.bad(fmt.Sprintf("%s  %s: the restored publication is not clean (exit %d)", code, name, rc))
			return
		}
		h.ok(code + "  " + name)
	}

	// The first record in byte order, which is what `LC_ALL=C sort` gives.
	var records []string
	_ = filepath.WalkDir(filepath.Join(bundle, "profiles"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			records = append(records, p)
		}
		return nil
	})
	sort.Strings(records)
	if len(records) > 0 {
		appendBytes(records[0], []byte("\n"))
	}
	plantCase("a record whose bytes moved after publication", "E-LIB-02")

	_ = os.Remove(filepath.Join(bundle, "formats", "bit-ids-v1.csv"))
	plantCase("a described file the publication does not carry", "E-LIB-01")

	_ = os.WriteFile(filepath.Join(bundle, "formats", "bit-ids-v1.extra.json"), []byte("x\n"), 0o644)
	plantCase("a carried file nobody described", "E-LIB-03")

	_ = os.Remove(index)
	plantCase("a publication with no index", "E-LIB-08")

	_ = os.Remove(filepath.Join(bundle, "MANIFEST.json"))
	plantCase("a publication with no manifest", "E-LIB-08")

	// ⚠ The index rewritten to another generation, with the two root documents
	// reassembled over it so the digest check cannot fire first. Without that this
	// case would pass as E-LIB-02 and say nothing about compatibility. The first
	// occurrence on each line, as `sed` without `g` replaces.
	if b, err := os.ReadFile(index); err == nil {
		var out [][]byte
		for _, l := range bytes.SplitAfter(b, []byte("\n")) {
			out = append(out, bytes.Replace(l, []byte("bit-ids/index/1"), []byte("bit-ids/index/2"), 1))
		}
		_ = os.WriteFile(index, bytes.Join(out, nil), 0o644)
	}
	_ = os.Remove(filepath.Join(bundle, "MANIFEST.json"))
	_ = os.Remove(filepath.Join(bundle, "SHA256SUMS"))
	runQuiet(bins["assemble-release"], bundle)
	plantCase("an index document from another generation", "E-LIB-06")

	// -- ⛔ THE NO-NETWORK CLAIM, SWEPT RATHER THAN OBSERVED --------------------
	if hits := sweep(filepath.Join(r.root, "crates", "bit-ids", "src"), catalogueNeedles); len(hits) > 0 {
		h.bad("network  the crate names a socket or an HTTP client: " + headJoin(hits, 2))
	} else {
		h.ok("network  no socket, address type or HTTP client is named anywhere in the crate")
	}

	// ⚠ THE SWEEP IS CHECKED AGAINST A TREE THAT REALLY CARRIES ONE. A needle list
	// that had stopped matching anything would report the same clean answer over
	// a crate full of sockets, which is the shape OBS-06 found.
	lab := filepath.Join(r.root, "crates", "bit-ids-lab", "src")
	if st, err := os.Stat(lab); err == nil && st.IsDir() && len(sweep(lab, catalogueNeedles)) > 0 {
		h.ok("network  and the same needles do match the crate that owns the sockets")
	} else {
		h.bad("network  the needle list matches nothing even in bit-ids-lab; it has stopped sweeping")
	}

	// And the dependency list, because a socket can arrive through a crate rather
	// than through a line of source.
	transport := false
	for _, l := range fileLines(filepath.Join(r.root, "crates", "bit-ids", "Cargo.toml")) {
		for _, t := range catalogueTransports {
			if strings.HasPrefix(l, t) {
				transport = true
			}
		}
	}
	if transport {
		h.bad("network  the crate depends on a transport")
	} else {
		h.ok("network  and its dependency list carries no transport")
	}

	return h.report("check-catalogue/1", "cases"), nil
}

// sweep is `grep -rn` over a directory for any of several literals: every
// matching line as `path:lineno:line`, in the order the walk visits them.
func sweep(dir string, needles []string) []string {
	var hits []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		for i, l := range fileLines(p) {
			for _, n := range needles {
				if strings.Contains(l, n) {
					hits = append(hits, fmt.Sprintf("%s:%d:%s", p, i+1, l))
					break
				}
			}
		}
		return nil
	})
	return hits
}

// runQuiet is `PROG ARGS >/dev/null 2>&1`, answering the exit code.
func runQuiet(prog string, args ...string) int {
	cmd := exec.Command(prog, args...)
	if err := cmd.Run(); err != nil {
		if x, ok := err.(*exec.ExitError); ok {
			return x.ExitCode()
		}
		return -1
	}
	return 0
}

// fileSize is `[ -s FILE ]` turned into a number: 0 for a missing file.
func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

// newlines is `wc -l <FILE`: the newline bytes, not the lines.
func newlines(path string) int {
	b, _ := os.ReadFile(path)
	return bytes.Count(b, []byte("\n"))
}

// nonEmpty is `grep -c .`: the lines holding at least one character.
func nonEmpty(lines []string) int {
	n := 0
	for _, l := range lines {
		if l != "" {
			n++
		}
	}
	return n
}

// appendBytes is `printf ... >>FILE`.
func appendBytes(path string, b []byte) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(b)
}
