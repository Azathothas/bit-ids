// bitcheck.go - does the Go checking binary refuse what it must? Every case plants
// a defect into a scratch copy of the tree and asks this binary for the exit code
// the case names.
//
// Ported from scripts/common/check-bitcheck.sh, `CI-10`. ⛔ The entry's own bound
// is why this exists: *a ported check refuses exactly what its shell half
// refused, over the same plants, with the same 0/1/2 vocabulary*. The halves are
// gone; the plants are not, because a plant does not need a second
// implementation to be a plant.
//
// -- ⭐ ONE CASE LIST, AND THE ORDER AND THE LABELS ARE THE SHELL HALF'S ---------
//
//	default      the plants are run against this binary. ⭐ THE PERMANENT GATE ROW.
//	--harnesses  the default mode plus the cases for ported HARNESSES, which plant
//	             into what a harness reads and so build its Rust subject out of
//	             the scratch tree.
//	--json       the one store_report line, and nothing else on stdout.
//
// ⛔ A PORTED HARNESS'S CASES ARE NOT IN THE DEFAULT MODE, measured rather than
// assumed: the scratch tree needs a target directory of its own - sharing a
// caller's would let a planted example be uplifted over the binary another gate
// row is executing - so the build is cold. On 2026-09-29 it made this row 39
// seconds where it had been 23, and `check-workflow` runs the gate about nine
// times per shard. The default mode says what it did not run, on stderr, rather
// than passing a row for it.
//
// -- ⛔ --compare AND THE PORT TABLE ARE NOT PORTED, AND THIS IS WHY ------------
//
// The shell half's `--compare` ran each case against the `.sh` and `.ps1` halves
// as well, and `check-bitcheck.sh check-assemble` ran one harness's cases alone so
// a port in progress could be held against its half. ⚠ Every half it compared is
// deleted - TODO/ci.md under `CI-10` (*Step 2 of 3: the twin layer is being
// deleted*, the four pairs of 87396c3, then 73bce00, 1603f80, 4c00fa9, a8cfedf,
// be26fe7, e6ae4c4 and 5d54c5f, the last of which left no twin pair at all) and
// under `CI-07` (*the rows are kinds*: `check-corpus` the first row that route
// closed on 2026-10-14, `check-assemble` the last portable one, cf1c06d). Read on
// 2026-10-15: no `scripts/<half>.sh` or `.ps1` that any case names exists, so
// every `--compare` row there said *the sh half is gone, not compared* or *a
// Go-only rule*, and the port table's one entry compared check-assemble with
// nothing. ⛔ A mode whose every row reports that it compared nothing is a mode
// that cannot fail for the reason it is named, so it is dropped rather than kept
// as ceremony; `--compare` is an unknown argument here, exit 2. With it go each
// case's predecessor path, the `-` for a Go-only rule and the PowerShell flag
// translation, which only ever chose what to run a half with.
//
// -- ⛔ WHY A CLEAN TREE PROVES ALMOST NOTHING HERE --------------------------
//
// Measured on 2026-08-28 with three divergences planted in a .ps1 half: a changed
// ceiling was caught, a dropped `md` scope was caught, and a dropped `py` scope
// was INVISIBLE, because this tree holds no .py file. ⭐ So every case below is a
// PLANT, and the clean tree is a control either side rather than the
// measurement. ⚠ And half the cases plant something the rule must NOT fire on,
// because that is where a port is likeliest to be over-strict.
//
// -- ⚠ WHAT THIS STILL SHELLS OUT FOR ----------------------------------------
//
// The stand-ins for `gh`, `curl` and `git` are the shell half's scripts, byte for
// byte, written into the scratch directory: they are programs the subject runs,
// and the subject finds them on PATH. The `gh` stand-in answers `--jq` with jq,
// and what mine-repo leaves behind is normalised with `jq -S -c` as the shell
// half did, so `sh` and `jq` are required and their absence is COULD NOT RUN. ⛔
// So this row is a Unix row, as its shell half was; a Windows lane needs stubs
// that are not shell scripts, which is a change to what is compared and not part
// of a port.
//
// ⛔ Read the exit code from this process, unpiped.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// optHarnesses is check-bitcheck's `--harnesses`.
var optHarnesses bool

// bitcheckStop carries a COULD NOT RUN out of a case list that is written as a
// list. ⚠ The shell half's `|| exit 2` after every save, ported as a panic that
// checkBitcheck alone recovers, so a case reads as a line rather than as an
// error check around a line.
type bitcheckStop struct{ err error }

type bitcheck struct {
	h     *harness
	self  string
	work  string
	tree  string
	outf  string
	env   []string
	goOut string // the shell half's AGREE_GO_JSON: what the last case printed

	// The stand-ins' state: where they serve from, and the PATH from before
	// either was put in front of it.
	stub            string
	pathBeforeStubs string

	// mine-repo's state, which its rows assert against.
	kept  string
	mrOut string
}

func (b *bitcheck) must(err error) {
	if err != nil {
		panic(bitcheckStop{err})
	}
}

func (b *bitcheck) p(rel string) string  { return filepath.Join(b.tree, filepath.FromSlash(rel)) }
func (b *bitcheck) w(name string) string { return filepath.Join(b.work, name) }

// note is a line the shell half wrote to stderr in both modes, as the run went.
// ⚠ Written straight to stderr rather than carried in the verdict: main prints a
// verdict's stderr IN PLACE OF the --json line, which is the one shape these
// lines must never take.
func (b *bitcheck) note(s string) { fmt.Fprintf(os.Stderr, "check-bitcheck: %s\n", s) }

// setenv sets one variable in what every subject is started with.
func (b *bitcheck) setenv(key, val string) {
	for i, kv := range b.env {
		if strings.HasPrefix(kv, key+"=") {
			b.env[i] = key + "=" + val
			return
		}
	}
	b.env = append(b.env, key+"="+val)
}

func (b *bitcheck) getenv(key string) string {
	for _, kv := range b.env {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v
		}
	}
	return ""
}

// -- planting -----------------------------------------------------------------
//
// ⛔ A PLANT IS REMOVED BY THE CASE THAT MADE IT, and the control after it is what
// says the removal worked. A harness whose plants accumulate measures the sum of
// everything it has done so far.
//
// ⚠ ONE DEPARTURE FROM THE SHELL HALF, AND IT ONLY FIRES ON A BROKEN HOST: a plant
// or a restore that cannot be written is COULD NOT RUN here. The shell half's
// `printf >` and restoring `cp` had no `|| exit 2`, so a write that failed left
// an ACCEPT case passing over a tree nobody had planted.

func (b *bitcheck) write(rel, content string) {
	b.must(os.WriteFile(b.p(rel), []byte(content), 0o644))
}

// appendTo is `printf ... >>FILE`, which makes the file when it is absent.
func (b *bitcheck) appendTo(rel, content string) {
	f, err := os.OpenFile(b.p(rel), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	b.must(err)
	_, werr := f.WriteString(content)
	b.must(errors.Join(werr, f.Close()))
}

func (b *bitcheck) mkdir(rel string) { b.must(os.MkdirAll(b.p(rel), 0o755)) }

// unplant is `rm -f`, and rmrf `rm -rf`: a path already gone is not a failure.
func (b *bitcheck) unplant(rel string) { _ = os.Remove(b.p(rel)) }
func (b *bitcheck) rmrf(rel string)    { _ = os.RemoveAll(b.p(rel)) }

// save keeps a tracked file's bytes before a case edits it; put returns them.
func (b *bitcheck) save(rel, name string) { b.must(copyFile(b.p(rel), b.w(name))) }
func (b *bitcheck) put(name, rel string)  { b.must(copyFile(b.w(name), b.p(rel))) }

// rewrite is `sed EXPR SAVED >FILE`, `awk ... SAVED >FILE` and `grep -v`: the saved
// copy, rewritten line by line, over the file in the tree.
func (b *bitcheck) rewrite(saved, rel string, fn func(string) (string, bool)) {
	b.must(rewriteLines(b.w(saved), b.p(rel), fn))
}

// gitIn runs the harness's own git in the scratch tree, never a subject's.
// ⚠ Its output goes to stderr, where the shell half's went: `git add -f` warns
// there about a line ending, and nothing it prints may reach a --json stdout.
func (b *bitcheck) gitIn(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = b.tree
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	return cmd.Run()
}

// ⛔ SOME PLANTS HAVE TO BE TRACKED, AND .gitignore IS WHY. check-no-secrets'
// first rule fires on a credential FILE, and this repository ignores most of the
// names it knows, so a plant left untracked is ignored, out of scope, and the case
// would report a rule holding over a file it never saw. ⭐ `git add -f` is also
// the truer fixture: the rule's own title is *a credential file is TRACKED*.
func (b *bitcheck) plantTracked(rel string) { _ = b.gitIn("add", "-f", "--", rel) }

// ⛔ AND IT LEAVES THE INDEX AS IT FOUND IT. Deleting the file alone leaves a
// tracked-but-deleted path, which `git ls-files` still reports - and a rule that
// reads a PATH would carry the finding into every case after it.
func (b *bitcheck) unplantTracked(rel string) {
	_ = b.gitIn("rm", "-q", "--cached", "--ignore-unmatch", "--", rel)
	b.unplant(rel)
}

// -- running the binary -------------------------------------------------------
//
// ⛔ EACH EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED. The output
// goes to a file, both streams to the one file in the order they were written, as
// `>"$OUTF" 2>&1` did.
func (b *bitcheck) run(args ...string) (int, string) {
	f, err := os.Create(b.outf)
	b.must(err)
	cmd := exec.Command(b.self, args...)
	cmd.Dir = b.tree
	cmd.Env = append(append([]string{}, b.env...), "PWD="+b.tree)
	cmd.Stdout, cmd.Stderr = f, f
	rc := exitOf(cmd.Run())
	_ = f.Close()
	return rc, readString(b.outf)
}

// exitOf answers what `$?` would have: the code, 128 plus a signal, or 127 for a
// program that never started.
func exitOf(err error) int {
	if err == nil {
		return 0
	}
	var x *exec.ExitError
	if !errors.As(err, &x) {
		return 127
	}
	if ws, ok := x.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return x.ExitCode()
}

// agree is one case: the plant has landed, the binary is asked, and the code it
// answers is the code the case named.
//
// ⛔ THE EXPECTED CODE IS AN ARGUMENT, AND THAT IS THE HALF THAT MAKES THIS A
// PROOF. A binary that answers 0 over a plant has missed it, so a case states what
// the plant is supposed to do and a wrong answer is a failure.
func (b *bitcheck) agree(label, check string, want int, flags ...string) {
	rc, out := b.run(append([]string{check, "--json"}, flags...)...)
	// ⚠ What `$(cat "$OUTF")` kept: every trailing newline is gone.
	b.goOut = strings.TrimRight(out, "\n")
	if rc != want {
		b.h.bad(fmt.Sprintf("%s: go answered %d, expected %d", label, rc, want))
		return
	}
	b.h.ok(fmt.Sprintf("%s: go (exit %d)", label, want))
}

// ⛔ AN EXIT CODE OF 1 SAYS A HARNESS FAILED A ROW AND NOT WHICH. Two plants aimed
// at different rows are indistinguishable by code alone - and a harness whose
// rows had been rewired would pass both. This asserts the whole line the binary
// gave for the case just run, and costs no second run.
//
// ⚠ THE VERDICT IS THE LAST LINE, NOT THE WHOLE OUTPUT. check-remote-items keeps
// a report for a person on stderr in --json mode, ahead of the document.
func (b *bitcheck) goSaid(label, want string) {
	said := b.goOut
	if i := strings.LastIndex(said, "\n"); i >= 0 {
		said = said[i+1:]
	}
	if said == want {
		b.h.ok(label + ": go said " + want)
	} else {
		b.h.bad(label + ": go said " + said + ", expected " + want)
	}
}

// planted is the shell half's `if replace_once ...; then CASE; else fail NOT
// PLANTED; fi`, with the file put back afterwards either way.
func (b *bitcheck) planted(label, rel, saved string, edits [][2]string, cases func()) {
	ok := true
	for _, e := range edits {
		if !replaceOnce(b.p(rel), e[0], e[1]) {
			ok = false
			break
		}
	}
	if ok {
		cases()
	} else {
		b.h.bad(label + ": NOT PLANTED")
	}
	b.put(saved, rel)
}

// copyScratch is the shell half's `git ls-files -z | tar -T - -cf - | tar -xf -`.
//
// ⚠ Tracked AND untracked-but-not-ignored, because an uncommitted new file is
// part of the tree the next push carries. ⚠ A tracked path missing from the
// working tree is passed over with tar's warning, which is what that pipeline did:
// its status was the extracting tar's, and that one succeeded.
func copyScratch(root, tree string) error {
	var paths []string
	for _, args := range [][]string{
		{"ls-files", "-z"},
		{"ls-files", "-zo", "--exclude-standard"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return err
		}
		for _, p := range strings.Split(string(out), "\x00") {
			if p != "" {
				paths = append(paths, p)
			}
		}
	}
	for _, rel := range paths {
		from := filepath.Join(root, filepath.FromSlash(rel))
		to := filepath.Join(tree, filepath.FromSlash(rel))
		st, err := os.Lstat(from)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %s: Cannot stat: No such file or directory\n", rel)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		switch {
		case st.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(from)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, to); err != nil {
				return err
			}
		case st.IsDir():
			if err := copyTree(from, to); err != nil {
				return err
			}
		default:
			if err := copyFile(from, to); err != nil {
				return err
			}
			if err := os.Chmod(to, st.Mode().Perm()); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBitcheck(r *repo) (v verdict, err error) {
	// ⚠ jq IS REQUIRED because the gh stand-in answers `--jq` with it and the
	// mine-repo cases normalise JSON through it. Without it those cases would FAIL
	// rather than say they could not run, which is the wrong one of the two.
	if err := needTools("git", "sh", "jq"); err != nil {
		return verdict{}, err
	}
	// ⛔ NOT ON WINDOWS, AND SAID RATHER THAN ATTEMPTED. A PATH lookup there only
	// finds a name with an executable extension, so the shell-script stand-ins
	// would be passed over and the remote-items and mine-repo cases would reach
	// the REAL `gh` and `curl` - the network, answering cases written for stubs.
	if runtime.GOOS == "windows" {
		return verdict{}, errCannotRun("the gh, curl and git stand-ins are shell scripts, which a Windows PATH lookup does not find")
	}
	// ⭐ THE SUBJECT IS THIS BINARY, which is the shell half's *built from the
	// tree on disk* made exact: the case list and the checks it asks cannot come
	// from two different builds.
	self, err := os.Executable()
	if err != nil {
		return verdict{}, errCannotRun("cannot name this binary, so there is nothing to ask")
	}
	work, err := scratch("checkbitcheck")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	onInterrupt(func() { _ = os.RemoveAll(work) })

	b := &bitcheck{
		h:    &harness{me: "check-bitcheck"},
		self: self,
		work: work,
		tree: filepath.Join(work, "tree"),
		outf: filepath.Join(work, "out"),
		env:  os.Environ(),
	}

	if err := os.MkdirAll(b.tree, 0o755); err != nil {
		return verdict{}, errCannotRun("cannot copy the working tree")
	}
	if copyScratch(r.root, b.tree) != nil {
		return verdict{}, errCannotRun("cannot copy the working tree")
	}
	// ⛔ IT IS A REAL REPOSITORY, because every check here asks git what is in the
	// tree. A plain directory would make every case exit 2.
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{"-c", "user.email=gate", "-c", "user.name=gate", "commit", "-qm", "scratch"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = b.tree
		if cmd.Run() != nil {
			return verdict{}, errCannotRun("cannot make the scratch tree a repository")
		}
	}

	defer func() {
		if x := recover(); x != nil {
			s, ok := x.(bitcheckStop)
			if !ok {
				panic(x)
			}
			v, err = verdict{}, errCannotRun(s.err.Error())
		}
	}()

	b.controlBytes()
	b.markers()
	b.oneHome()
	b.changelog()
	b.licences()
	b.cache()
	b.harnessSection("check-corpus", b.corpus)
	b.harnessSection("check-indexes", b.indexes)
	b.harnessSection("check-release", b.release)
	b.harnessSection("check-formats", b.formats)
	b.harnessSection("check-staleness", b.staleness)
	b.harnessSection("check-connector", b.connector)
	b.harnessSection("check-assemble", b.assemble)
	b.harnessSection("check-catalogue", b.catalogue)
	b.remoteItems()
	b.mineRepo()
	b.placeholders()
	b.noSecrets()
	b.ignores()
	b.adapters()
	b.docs()
	b.project()

	return b.h.report("check-bitcheck/1", "cases"), nil
}

// harnessSection runs one ported harness's cases under --harnesses, and says it
// did not otherwise. ⚠ The note no longer names `--compare`, which is not a mode
// of this port; it is the one line here worded differently from the shell half.
func (b *bitcheck) harnessSection(name string, cases func()) {
	if !optHarnesses {
		b.note("the " + name + " cases were NOT run; --harnesses runs them")
		return
	}
	cases()
}

// ============================================================================
// check-control-bytes
// ============================================================================

func (b *bitcheck) controlBytes() {
	const plant = "tools/check/plant.txt"
	b.agree("control-bytes clean tree", "check-control-bytes", 0)

	// ⛔ A C0 CONTROL BYTE IN A TEXT FILE. The byte is an escape rather than a
	// literal, which is this repository's own rule: a literal control byte here
	// would make THIS file the thing it exists to refuse.
	b.write(plant, "plant\x01here\n")
	b.agree("control-bytes a C0 byte is refused", "check-control-bytes", 1)
	b.unplant(plant)

	// ⛔ NUL IS A SEPARATE TEST, because the shell half found it by a different
	// route. A port that folded the two would miss the commonest offender there is.
	b.write(plant, "plant\x00here\n")
	b.agree("control-bytes a NUL byte is refused", "check-control-bytes", 1)
	b.unplant(plant)

	// ⭐ THE THREE THAT ARE LEGITIMATELY IN TEXT, the case a port is likeliest to
	// get wrong in the strict direction.
	b.write(plant, "tab\there\r\nand a return\n")
	b.agree("control-bytes tab, CR and LF are accepted", "check-control-bytes", 0)
	b.unplant(plant)

	// ⛔ DEL IS OUTSIDE THE CLASS, AND THIS CASE EXISTS BECAUSE A MUTATION SURVIVED
	// WITHOUT IT. Measured on 2026-09-10: adding 0x7f to the Go class left every
	// case above green, because nothing in the tree and nothing planted carried
	// the byte. ⚠ The repair is a fixture rather than a reading.
	b.write(plant, "plant\x7fhere\n")
	b.agree("control-bytes DEL is outside the class and accepted", "check-control-bytes", 0)
	b.unplant(plant)

	// ⚠ SCOPE IS PROVED WITH A FIXTURE RATHER THAN TRUSTED.
	b.write("tools/check/plant.bin", "plant\x01here\n")
	b.agree("control-bytes a byte outside the text extensions is out of scope", "check-control-bytes", 0)
	b.unplant("tools/check/plant.bin")

	b.agree("control-bytes clean tree again", "check-control-bytes", 0)
}

// ============================================================================
// check-markers
// ============================================================================

// dash is U+2014 and stop U+26D4, as escapes, so that neither is a character
// THIS file carries. ⚠ The em dash is outside the five and is the character the
// rule was originally written about.
const (
	dash = "\xe2\x80\x94"
	stop = "\xe2\x9b\x94"
	bom  = "\xef\xbb\xbf"
)

func (b *bitcheck) markers() {
	const plant = "tools/check/plant.md"
	b.agree("markers clean tree", "check-markers", 0)

	b.write(plant, "a plant "+dash+" here\n")
	b.agree("markers a character outside the five is refused", "check-markers", 1)
	b.unplant(plant)

	// ⭐ THE SPECIMEN EXEMPTION, and it is markdown only. Without it the rule is
	// unwritable: a page that bans a character cannot show a reader which one.
	b.write(plant, "text\n\n```\na specimen "+dash+" inside a fence\n```\n")
	b.agree("markers a specimen inside a fenced block is accepted", "check-markers", 0)
	b.unplant(plant)

	b.write(plant, "an inline `"+dash+"` span\n")
	b.agree("markers a specimen inside a code span is accepted", "check-markers", 0)
	b.unplant(plant)

	// ⛔ AND THERE IS NO EXEMPTION OUTSIDE MARKDOWN, which is the branch that
	// separates the two scopes.
	b.write("tools/check/plant.sh", "# a plant "+dash+" here\n")
	b.agree("markers the same character in a .sh is refused", "check-markers", 1)
	b.unplant("tools/check/plant.sh")

	// ⚠ A LEADING BYTE-ORDER MARK IS EXEMPT AND ONLY A LEADING ONE. Both
	// directions, because a port that stripped every BOM would lose the second.
	b.write(plant, bom+"ascii only\n")
	b.agree("markers a leading BOM is accepted", "check-markers", 0)
	b.unplant(plant)

	b.write(plant, "first line\n"+bom+"second line\n")
	b.agree("markers a BOM anywhere else is refused", "check-markers", 1)
	b.unplant(plant)

	// ⛔ THE DENSITY CEILING IS A SEPARATE RULE OVER THE SAME FILES: every byte
	// here is one of the five, so only the count refuses it.
	b.write(plant, "a line with markers "+strings.Repeat(stop, 10)+"\n")
	b.agree("markers a file over the density ceiling is refused", "check-markers", 1)
	b.unplant(plant)

	// ⛔ LICENSES/*.txt IS EXEMPT AND THIS TREE HAS NO SUCH FILE, so the exemption
	// is proved with a fixture: a `py` scope dropped from one half was invisible
	// to check-twins because the tree held no .py file at all.
	b.mkdir("LICENSES")
	b.write("LICENSES/PLANT.txt", "a canonical text with a dash "+dash+" in it\n")
	b.agree("markers LICENSES/*.txt is exempt", "check-markers", 0)
	b.rmrf("LICENSES")

	// ⚠ AND THE EXEMPTION IS THE DIRECTORY AND THE EXTENSION TOGETHER.
	b.mkdir("LICENSES")
	b.write("LICENSES/PLANT.md", "a dash "+dash+" in a markdown file\n")
	b.agree("markers LICENSES/*.md is NOT exempt", "check-markers", 1)
	b.rmrf("LICENSES")

	b.agree("markers clean tree again", "check-markers", 0)
}

// ============================================================================
// check-one-home
// ============================================================================

func (b *bitcheck) oneHome() {
	const a, c = "tools/check/plant-a.md", "tools/check/plant-b.md"
	b.agree("one-home clean tree", "check-one-home", 0)

	// A sentence of twelve or more words, which is the length below which a
	// repeated phrase is not a fact.
	const long = "the capture host is claimed before anything else writes to it and the route is cut afterwards"
	const short = "the capture host is claimed first"

	b.write(a, long+"\n")
	b.write(c, long+"\n")
	b.agree("one-home one sentence in two documents is refused", "check-one-home", 1)
	b.unplant(a)
	b.unplant(c)

	// ⚠ THE SAME SENTENCE TWICE IN ONE DOCUMENT IS NOT A DUPLICATE. A port that
	// counted occurrences rather than distinct files would refuse every document
	// that repeats a heading.
	b.write(a, long+"\n\n"+long+"\n")
	b.agree("one-home the same sentence twice in ONE document is accepted", "check-one-home", 0)
	b.unplant(a)

	b.write(a, short+"\n")
	b.write(c, short+"\n")
	b.agree("one-home a sentence under the word floor is accepted", "check-one-home", 0)
	b.unplant(a)
	b.unplant(c)

	// ⛔ ROUTERS ARE EXEMPT AS A SET AND THIS TREE CARRIES NEITHER OF THEM at the
	// paths the rule names, so the exemption is proved with a fixture.
	b.write("AGENTS.md", long+"\n")
	b.write("ROUTE.md", long+"\n")
	b.agree("one-home a sentence shared only between routers is accepted", "check-one-home", 0)
	b.unplant("AGENTS.md")

	// ⛔ AND THE EXEMPTION IS *EVERY* FILE, NOT *ANY*.
	b.write(a, long+"\n")
	b.agree("one-home a router sharing with an ordinary document is refused", "check-one-home", 1)
	b.unplant(a)
	b.unplant("ROUTE.md")

	// ⚠ docs/history/ IS OUT OF SCOPE: the history says what was believed then.
	b.write(a, long+"\n")
	b.mkdir("docs/history")
	b.write("docs/history/PLANT.md", long+"\n")
	b.agree("one-home docs/history is out of scope", "check-one-home", 0)
	b.unplant(a)
	b.unplant("docs/history/PLANT.md")

	// ⭐ A SHARED SENTENCE INSIDE A FENCED BLOCK IS NOT A SHARED SENTENCE.
	b.write(a, "```\n"+long+"\n```\n")
	b.write(c, "```\n"+long+"\n```\n")
	b.agree("one-home a sentence shared inside fenced blocks is accepted", "check-one-home", 0)
	b.unplant(a)
	b.unplant(c)

	b.agree("one-home clean tree again", "check-one-home", 0)
}

// ============================================================================
// check-changelog
// ============================================================================

func (b *bitcheck) changelog() {
	const cl = "CHANGELOG.md"
	b.agree("changelog clean tree", "check-changelog", 0)
	b.save(cl, "CHANGELOG.orig")

	// ⛔ THE ORDERING RESETS AT EVERY SECTION: an Unreleased section above a
	// versioned one is CORRECT even though its date is older.
	b.write(cl, "# Changelog\n\n## Unreleased\n\n"+
		"### 2026-01-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n\n"+
		"## 1.0.0\n\n"+
		"### 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n")
	b.agree("changelog an older date in a later section is accepted", "check-changelog", 0)

	b.write(cl, "# Changelog\n\n## Unreleased\n\n"+
		"### 2026-01-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n\n"+
		"### 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n")
	b.agree("changelog two entries out of order in ONE section is refused", "check-changelog", 1)

	b.write(cl, "# Changelog\n\n## Unreleased\n\n"+
		"### the day it happened\n\n- Record: TODO/ci.md. No deploy.\n")
	b.agree("changelog an entry with no date is refused", "check-changelog", 1)

	b.write(cl, "# Changelog\n\n## Unreleased\n\n"+
		"### 2026-06-01T00:00:00Z\n\n- Something happened. No deploy.\n")
	b.agree("changelog an entry naming no record is refused", "check-changelog", 1)

	b.write(cl, "# Changelog\n\n## Unreleased\n\n"+
		"### 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md.\n")
	b.agree("changelog an entry silent about deployment is refused", "check-changelog", 1)

	// ⛔ A FILE THAT PARSES TO NO ENTRIES IS A FAILURE, NOT A CLEAN RUN. ⚠ It is the
	// one refusal here that emits no JSON at all, so it is also the case that says
	// the port carried the stderr path rather than inventing an answer.
	b.write(cl, "# Changelog\n\n## 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n")
	b.agree("changelog a file with no entries the parser knows is refused", "check-changelog", 1)

	b.put("CHANGELOG.orig", cl)
	b.agree("changelog clean tree again", "check-changelog", 0)
}

// ============================================================================
// check-licences
// ============================================================================

const (
	licReg = "catalogue/licences.toml"
	licCat = "catalogue/clients.toml"
)

func (b *bitcheck) restoreLicences() {
	b.put("licences.orig", licReg)
	b.put("clients.orig", licCat)
}

func (b *bitcheck) licences() {
	b.agree("licences clean tree", "check-licences", 0)
	b.save(licReg, "licences.orig")
	b.save(licCat, "clients.orig")

	// ⛔ A CATALOGUE TARGET WITH NO REGISTER ROW: a target added and its licence
	// forgotten is a build nobody decided the disposition of.
	b.write(licCat, readString(b.w("clients.orig"))+"\n[[targets]]\nid = \"zz-plant\"\nopen_source = true\n")
	b.agree("licences a catalogue target with no register row is refused", "check-licences", 1)
	b.restoreLicences()

	// ⚠ AND THE OTHER DIRECTION, which would survive a check that only walked the
	// catalogue.
	b.write(licReg, readString(b.w("licences.orig"))+
		"\n[[targets]]\nid = \"zz-plant\"\nlicence = \"MIT\"\nlicence_source = \"x\"\nredistribute = \"refused\"\n")
	b.agree("licences a register row naming no catalogue target is refused", "check-licences", 1)
	b.restoreLicences()

	// ⛔ PERMITTED IS THE EXPENSIVE VALUE AND IT HAS TO BE EARNED.
	b.rewrite("licences.orig", licReg, func(l string) (string, bool) {
		if l == `redistribute = "refused"` {
			return `redistribute = "permitted"`, true
		}
		return l, true
	})
	b.agree("licences permitted without a verified licence and a notice is refused", "check-licences", 1)
	b.restoreLicences()

	// ⛔ A REGISTER OF NOTHING IS REFUSED - AND THIS CASE DOES NOT ESTABLISH BY
	// WHICH RULE. Measured on 2026-09-10 by planting: with the empty-register
	// refusal removed outright, every case here still passed, because a register
	// with no rows has no row for any catalogue target and rule 1 fires on that.
	// ⚠ The empty guard is defence in depth, and the label says so instead of
	// claiming a proof of it.
	b.write(licReg, "schema = \"bit-ids/licences/1\"\n")
	b.agree("licences an empty register is refused, by rule 1 rather than by the empty guard", "check-licences", 1)
	b.restoreLicences()

	b.agree("licences clean tree again", "check-licences", 0)
}

// ============================================================================
// check-cache
// ============================================================================

// permitTarget is the shell half's awk: the first `redistribute = "refused"`
// after the line naming the target becomes permitted.
func permitTarget(id string) func(string) (string, bool) {
	armed := false
	return func(l string) (string, bool) {
		if l == `id = "`+id+`"` {
			armed = true
		}
		if armed && l == `redistribute = "refused"` {
			armed = false
			return `redistribute = "permitted"`, true
		}
		return l, true
	}
}

func (b *bitcheck) cache() {
	// ⛔ CARGO_TARGET_DIR IS THIS RUN'S OWN from here on, in every mode. A harness
	// builds its example out of THIS scratch tree, so the build is paid once - and
	// it never lands in a target directory another check of the same gate is
	// executing examples out of.
	b.setenv("CARGO_TARGET_DIR", b.w("target"))

	b.harnessSection("check-cache", func() {
		b.agree("cache clean tree", "check-cache", 0)
		b.goSaid("cache clean tree", `{"schema":"check-cache/1","total":13,"passed":13,"failed":0}`)

		// ⛔ THE REGISTER PERMITS THE SCENARIO'S OWN TARGET. Three rows must fire
		// and only three.
		b.rewrite("licences.orig", licReg, permitTarget("aria2"))
		b.agree("cache a register permitting the scenario's target", "check-cache", 1)
		b.goSaid("cache a register permitting the scenario's target", `{"schema":"check-cache/1","total":13,"passed":10,"failed":3}`)
		b.restoreLicences()

		// ⚠ AND A REGISTER PERMITTING SOMETHING ELSE, which only the register row may
		// notice.
		b.rewrite("licences.orig", licReg, permitTarget("aria2-next"))
		b.agree("cache a register permitting another target", "check-cache", 1)
		b.goSaid("cache a register permitting another target", `{"schema":"check-cache/1","total":13,"passed":12,"failed":1}`)
		b.restoreLicences()

		// ⛔ A REGISTER THAT CANNOT BE READ IS COULD-NOT-RUN, never an empty
		// permitted list.
		b.unplant(licReg)
		b.agree("cache a missing register cannot run", "check-cache", 2)
		b.restoreLicences()

		// ⛔ THE SCENARIO'S SOURCE DID NOT MOVE. A retrieval is its location AND its
		// time, so the second is made an exact repeat of the first, and six rows must
		// say so. ⚠ Moving the location alone was tried first and changed nothing.
		const scen = "crates/bit-ids/examples/cache-scenario.rs"
		b.save(scen, "scenario.orig")
		b.planted("cache a retrieval that repeats the first", scen, "scenario.orig", [][2]string{
			{`"https://example.invalid/archive/2026/aria2.tar.gz"`, `"https://example.invalid/downloads/v1/aria2.tar.gz"`},
			{`"2026-06-18T09:00:00Z"`, `"2026-01-04T09:00:00Z"`},
		}, func() {
			b.agree("cache a retrieval that repeats the first", "check-cache", 1)
			b.goSaid("cache a retrieval that repeats the first", `{"schema":"check-cache/1","total":13,"passed":7,"failed":6}`)
		})

		b.agree("cache clean tree again", "check-cache", 0)
	})
}

// ============================================================================
// the harnesses ported after check-cache
// ============================================================================
//
// ⭐ Their plants go into the subject a harness drives, because over the real tree
// every one of its rows passes, so its own refusal branches are exercised by
// nothing else. ⛔ Each plant disables ONE refusal, and exactly the rows its label
// names must fail.

// harnessPlant is one plant into a harness's subject and the verdict line it must
// draw. ⚠ The line is written out whole rather than derived from a count, as the
// shell half wrote it, so a reader sees which rows a plant is meant to fail.
type harnessPlant struct {
	label string
	edits [][2]string
	said  string
}

// swap is one literal edit, old for new. ⛔ IT IS A FUNCTION SO THAT THIS FILE
// NEVER SPELLS A DOUBLE BRACE: a nested composite literal opens with two, which
// is exactly what check-placeholders refuses, and this file is in its scope. The
// first build of this port turned that check's clean tree red for that reason.
func swap(old, repl string) [2]string { return [2]string{old, repl} }

// harnessCases is one ported harness's cases: a clean control and its verdict
// line, each plant into one source file, and a clean control again.
func (b *bitcheck) harnessCases(short, check string, total int, rel, saved string, plants []harnessPlant) {
	b.agree(short+" clean tree", check, 0)
	b.goSaid(short+" clean tree", fmt.Sprintf(`{"schema":"%s/1","total":%d,"passed":%d,"failed":0}`, check, total, total))
	b.save(rel, saved)
	for _, p := range plants {
		b.planted(p.label, rel, saved, p.edits, func() {
			b.agree(p.label, check, 1)
			b.goSaid(p.label, p.said)
		})
	}
	b.agree(short+" clean tree again", check, 0)
}

func (b *bitcheck) corpus() {
	b.harnessCases("corpus", "check-corpus", 16, "crates/bit-ids/src/corpus.rs", "corpus.orig", []harnessPlant{
		{"corpus a validator that stopped comparing digests",
			[][2]string{swap("if object.sha256 != artifact.sha256 {", "if false && object.sha256 != artifact.sha256 {")},
			`{"schema":"check-corpus/1","total":16,"passed":15,"failed":1}`},
		{"corpus a validator that stopped sweeping undeclared files",
			[][2]string{swap("if text.starts_with(&raw) && !accounted.contains(path) {", "if false && text.starts_with(&raw) && !accounted.contains(path) {")},
			`{"schema":"check-corpus/1","total":16,"passed":15,"failed":1}`},
	})
}

// ⭐ One plant keeps a superseded record in the views, which the correction rows
// exist to see; one renames the refusal a version the scheme cannot order is
// given, which only that case may notice.
func (b *bitcheck) indexes() {
	b.harnessCases("indexes", "check-indexes", 17, "crates/bit-ids/src/index.rs", "index.orig", []harnessPlant{
		// ⚠ Three rows, and they are the correction rows: the superseded count, the
		// latest view's answer, and the corrected record still named in it.
		{"indexes a builder that keeps a superseded record in the views",
			[][2]string{swap("if graph.succeeds.contains_key(&profile.id) {", "if false && graph.succeeds.contains_key(&profile.id) {")},
			`{"schema":"check-indexes/1","total":17,"passed":14,"failed":3}`},
		{"indexes a builder that names the unorderable refusal wrongly",
			[][2]string{swap(`"E-VIW-02",`, `"E-VIW-0X",`)},
			`{"schema":"check-indexes/1","total":17,"passed":16,"failed":1}`},
	})
}

// ⭐ One plant makes the checksum file state the manifest's digest over the wrong
// bytes, which only the independent reader can see; one renames the refusal a
// symbolic link is given.
func (b *bitcheck) release() {
	b.harnessCases("release", "check-release", 15, "crates/bit-ids/src/release.rs", "release.orig", []harnessPlant{
		{"release checksums that misstate the manifest",
			[][2]string{swap("Sha256Digest::of(manifest)", `Sha256Digest::of(b"")`)},
			`{"schema":"check-release/1","total":15,"passed":14,"failed":1}`},
		{"release a link refused under the wrong code",
			[][2]string{swap(`"E-REL-02",`, `"E-REL-0X",`)},
			`{"schema":"check-release/1","total":15,"passed":14,"failed":1}`},
	})
}

// ⭐ Both plants go into the SQLite rendering, because that is the file only the
// independent reader looks inside.
func (b *bitcheck) formats() {
	b.harnessCases("formats", "check-formats", 28, "crates/bit-ids/src/sqlite.rs", "sqlite.orig", []harnessPlant{
		{"formats a database that miscounts its records",
			[][2]string{swap("params![records.to_string()]", "params![(records + 1).to_string()]")},
			`{"schema":"check-formats/1","total":28,"passed":27,"failed":1}`},
		{"formats a database that says nothing of what it omits",
			[][2]string{swap("for (subject, reason) in OMISSIONS {", "for (subject, reason) in OMISSIONS.iter().take(0) {")},
			`{"schema":"check-formats/1","total":28,"passed":27,"failed":1}`},
	})
}

// ⭐ One plant moves the request identifier's domain, which only the independent
// re-derivation can see; one never treats a held request as already open.
func (b *bitcheck) staleness() {
	b.harnessCases("staleness", "check-staleness", 19, "crates/bit-ids/src/staleness.rs", "staleness.orig", []harnessPlant{
		{"staleness a request identifier under another domain",
			[][2]string{swap(`const DOMAIN: &'static str = "bit-ids/capture-request/1";`, `const DOMAIN: &'static str = "bit-ids/capture-request/2";`)},
			`{"schema":"check-staleness/1","total":19,"passed":18,"failed":1}`},
		{"staleness a held request opened again",
			[][2]string{swap("state: if held {", "state: if false && held {")},
			`{"schema":"check-staleness/1","total":19,"passed":16,"failed":3}`},
	})
}

// ⭐ Its plants go into the connector it drives, the second reading of a capture:
// one reads the reserved bytes one byte late, one decodes a bare `+` as a space,
// which is the query-string reading a peer ID must not get.
func (b *bitcheck) connector() {
	b.harnessCases("connector", "check-connector", 54, "scripts/capture/connectors/cpython-stdlib.py", "connector.orig", []harnessPlant{
		{"connector reserved bytes read one byte late",
			[][2]string{swap(`"peer_wire/reserved": raw[reserved_at : reserved_at + 8],`, `"peer_wire/reserved": raw[reserved_at + 1 : reserved_at + 9],`)},
			`{"schema":"check-connector/1","total":54,"passed":50,"failed":4}`},
		{"connector a bare plus decoded as a space",
			[][2]string{swap("decoded = urllib.parse.unquote_to_bytes(value)", `decoded = urllib.parse.unquote_to_bytes(value.replace(b"+", b" "))`)},
			`{"schema":"check-connector/1","total":54,"passed":53,"failed":1}`},
		// ⭐ A handshake in two segments read segment by segment, which only the
		// split case notices. `ENGINE-02`.
		{"connector a split handshake read segment by segment",
			[][2]string{swap("        if reader is handshake_fields:", "        if False and reader is handshake_fields:")},
			`{"schema":"check-connector/1","total":54,"passed":52,"failed":2}`},
	})
}

// ⭐ Its plants go into the assembler it drives: one stops refusing a host nothing
// says is destroyed, which only the self-hosted case may notice; one rewords the
// refusal an `acquired` that is neither value is given.
func (b *bitcheck) assemble() {
	b.harnessCases("assemble", "check-assemble", 77, "crates/bit-ids-probe/examples/assemble-capture.rs", "assemble.orig", []harnessPlant{
		{"assemble a self-hosted runner taken as disposable",
			[][2]string{swap(`if environment != "github-hosted" {`, `if false && environment != "github-hosted" {`)},
			`{"schema":"check-assemble/1","total":77,"passed":76,"failed":1}`},
		{"assemble an unknown acquired value refused under other words",
			[][2]string{swap(`which is neither yes nor no",`, `which is neither yes or no",`)},
			`{"schema":"check-assemble/1","total":77,"passed":76,"failed":1}`},
		// ⭐ A crates.io lane filed as a host's package manager, which only the
		// cargo cases notice. `ENGINE-03`.
		{"assemble a crates.io lane filed as a host package manager",
			[][2]string{swap(`== Some("crates-io") => {`, `== Some("crates-io-unread") => {`)},
			`{"schema":"check-assemble/1","total":77,"passed":75,"failed":2}`},
		// ⭐ And a Go module proxy lane filed the same way, which only the
		// go_module cases notice. `ENGINE-02`.
		{"assemble a Go module proxy lane filed as a host package manager",
			[][2]string{swap(`== Some("proxy.golang.org") =>`, `== Some("proxy.golang.org-unread") =>`)},
			`{"schema":"check-assemble/1","total":77,"passed":75,"failed":2}`},
	})
}

// ============================================================================
// check-catalogue
// ============================================================================
//
// ⭐ Its plants go where the harness reads: the crate it sweeps for sockets, the
// manifest it reads for a transport, and the library it drives.
//
// ⛔ THE CASE-VARIANT PLANTS ARE THE ONES A CLEAN TREE CANNOT SEE. The `sh` half
// matched with `grep`, which is case-sensitive, and its twin with `-match`, which
// is not, so the two agreed over this tree and parted over a needle spelled
// another way.
func (b *bitcheck) catalogue() {
	const said = `{"schema":"check-catalogue/1","total":18,"passed":17,"failed":1}`
	b.agree("catalogue clean tree", "check-catalogue", 0)
	b.goSaid("catalogue clean tree", `{"schema":"check-catalogue/1","total":18,"passed":18,"failed":0}`)

	// ⛔ A SOCKET NAMED IN THE CRATE, in a file nothing compiles - the sweep reads
	// the directory, not the module tree.
	const plantRS = "crates/bit-ids/src/zz_planted.rs"
	b.write(plantRS, "// planted\nuse std::net::TcpStream;\n")
	b.agree("catalogue a socket named in the crate", "check-catalogue", 1)
	b.goSaid("catalogue a socket named in the crate", said)
	b.unplant(plantRS)

	// ⚠ THE SAME NAME IN ANOTHER CASE IS NOT THE TYPE: the needles are Rust paths.
	b.write(plantRS, "// a tcpstream spelled this way names nothing\n")
	b.agree("catalogue a needle in another case is not a socket", "check-catalogue", 0)
	b.unplant(plantRS)

	// ⛔ A TRANSPORT IN THE MANIFEST, under a table cargo ignores so the build
	// still succeeds: a plant that broke the build would be refused by the build.
	const mf = "crates/bit-ids/Cargo.toml"
	b.save(mf, "crate-manifest.orig")
	b.appendTo(mf, "\n[package.metadata.zz-plant]\ntokio = \"planted\"\n")
	b.agree("catalogue a transport in the manifest", "check-catalogue", 1)
	b.goSaid("catalogue a transport in the manifest", said)
	b.put("crate-manifest.orig", mf)

	b.appendTo(mf, "\n[package.metadata.zz-plant]\nTokio = \"planted\"\n")
	b.agree("catalogue a transport in another case is not one", "check-catalogue", 0)
	b.put("crate-manifest.orig", mf)

	// ⛔ AND THE SOCKET CRATE THE LAB TOOK. Until 2026-10-14 a manifest naming
	// `socket2` was refused by the licence register; the lab's one door gave it a
	// row, so this rule is what refuses it here now.
	b.appendTo(mf, "\n[package.metadata.zz-plant]\nsocket2 = \"planted\"\n")
	b.agree("catalogue the lab's socket crate in the manifest", "check-catalogue", 1)
	b.goSaid("catalogue the lab's socket crate in the manifest", said)
	b.put("crate-manifest.orig", mf)

	// ⛔ THE LIBRARY REFUSES A MOVED RECORD UNDER ANOTHER CODE. Both sites that
	// raise E-LIB-02 are disabled, and the record is still refused, as E-LIB-04;
	// the harness must call that the wrong refusal. ⚠ It was first called a
	// library that ACCEPTS a moved record, until a mutation pass showed the row
	// had only ever failed on the code.
	const cat = "crates/bit-ids/src/catalogue.rs"
	b.save(cat, "catalogue.rs.orig")
	b.planted("catalogue a moved record refused under another code", cat, "catalogue.rs.orig", [][2]string{
		{"if release.manifest_json().as_bytes() != manifest {", "if false && release.manifest_json().as_bytes() != manifest {"},
		{"if found != **digest {", "if false && found != **digest {"},
	}, func() {
		b.agree("catalogue a moved record refused under another code", "check-catalogue", 1)
		b.goSaid("catalogue a moved record refused under another code", said)
	})

	b.agree("catalogue clean tree again", "check-catalogue", 0)
}

// ============================================================================
// check-remote-items
// ============================================================================
//
// ⭐ THE ONE ROW NO SESSION HOST COULD RUN, RUN HERE WITH NO NETWORK AT ALL. A
// stand-in for `gh` and one for `curl`, first on PATH, serve every answer from
// files this section writes.
//
// ⛔ THE THREE REAL PINS ARE REAL, and their `runs:` lines are copies of what the
// commits carry, read on 2026-09-29: `actions/checkout` at v4.2.2 declares
// `node20`, at v5.0.0 `node24`, and `astral-sh/setup-uv` at v10.2.0 declares
// `"node24"` QUOTED - the spelling the check strips quotes for.
//
// ⚠ EVERY FORTY-DIGIT NAME IS ASSEMBLED FROM TWO HALVES, because a contiguous one
// in a tracked file is the shape `check-no-secrets --public` refuses.

// stubGH is the shell half's `gh` stand-in, byte for byte.
const stubGH = `#!/bin/sh
# A stand-in for gh that serves files from $STUB_GH_DIR and reaches nothing.
set -u
D=${STUB_GH_DIR:?}
case "$1 ${2:-}" in
  "auth status")
    [ -f "$D/auth-fails" ] && {
      printf 'You are not logged into any GitHub hosts.\n' >&2
      exit 1
    }
    exit 0
    ;;
  "issue list")
    [ -f "$D/issues.fail" ] && {
      cat "$D/issues.fail" >&2
      exit 1
    }
    cat "$D/issues.json"
    exit 0
    ;;
  "pr list")
    cat "$D/prs.json"
    exit 0
    ;;
  "pr diff")
    shift 2
    while [ $# -gt 1 ]; do shift; done
    [ -f "$D/diff.$1" ] || {
      printf 'could not find a diff for %s\n' "$1" >&2
      exit 1
    }
    cat "$D/diff.$1"
    exit 0
    ;;
esac
# ⚠ AN API PATH IS A FILE NAMED WITH ITS SLASHES AS ` + "`%`" + `, leading slash dropped,
# query kept. mine-repo asks for a repository AND for the lists under it, and a
# nested layout cannot hold ` + "`repos/o/r`" + ` as a file and a directory at once.
if [ "$1" = "api" ]; then
  shift
  path=""
  jqexpr=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --jq)
        jqexpr=$2
        shift 2
        ;;
      -f | -F)
        shift 2
        ;;
      -*) shift ;;
      *)
        [ -n "$path" ] || path=$1
        shift
        ;;
    esac
  done
  key=$(printf '%s' "${path#/}" | tr '/' '%')
  [ -f "$D/api/$key" ] || {
    printf 'gh: Not Found (HTTP 404)\n' >&2
    exit 1
  }
  if [ -n "$jqexpr" ]; then
    jq -r "$jqexpr" "$D/api/$key"
  else
    cat "$D/api/$key"
  fi
  exit 0
fi
printf 'stub gh: unsupported: %s\n' "$*" >&2
exit 1
`

// stubCurl is the shell half's `curl` stand-in. ⛔ The proxy refuses any
// user-agent but curl's own with 420, as the real one was measured to on
// 2026-08-28, so a caller that stops sending it fails here.
const stubCurl = `#!/bin/sh
# A stand-in for curl. It serves raw.githubusercontent.com out of
# $STUB_GH_DIR/raw, and the proxy out of $STUB_GH_DIR/proxy, keyed as the gh
# stub keys a path, and answers what each host answers for a path it lacks.
set -u
D=${STUB_GH_DIR:?}
url=""
out=""
fmt=""
ua=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o)
      out=$2
      shift 2
      ;;
    -w)
      fmt=$2
      shift 2
      ;;
    -A)
      ua=$2
      shift 2
      ;;
    -m | --max-time)
      shift 2
      ;;
    https://*)
      url=$1
      shift
      ;;
    *) shift ;;
  esac
done
case "$url" in
  https://raw.githubusercontent.com/*)
    rel=${url#https://raw.githubusercontent.com/}
    if [ -f "$D/raw/$rel" ]; then cat "$D/raw/$rel"; else printf '404: Not Found'; fi
    exit 0
    ;;
  https://api.gh.pkgforge.dev/*)
    key=$(printf '%s' "${url#https://api.gh.pkgforge.dev/}" | tr '/' '%')
    src=""
    if [ "$ua" != "curl/8" ]; then
      code=420
    elif [ -f "$D/proxy/$key.code" ]; then
      code=$(cat "$D/proxy/$key.code")
      src="$D/proxy/$key"
    elif [ -f "$D/proxy/$key" ]; then
      code=200
      src="$D/proxy/$key"
    else
      code=404
    fi
    {
      if [ -n "$src" ] && [ -f "$src" ]; then
        cat "$src"
      elif [ "$code" = 404 ]; then
        printf '{"message":"Not Found"}'
      fi
    } >"${out:-/dev/stdout}"
    [ -z "$fmt" ] || printf '%s' "$code"
    exit 0
    ;;
esac
printf 'stub curl: refused %s\n' "$url" >&2
exit 7
`

var (
	pinV4 = "11bd71901bbe5b1630ce" + "ea73d27597364c9af683"
	pinV5 = "08c6903cd8c0fde910a3" + "7f88322edcfb5dd907a8"
	pinUV = "c18668ad3cf93ea998be" + "f934396af7bb5c839dc7"
	// noSuch and tagObj are `printf '%040d' 0` and `printf '%040d' 7`.
	noSuch = fmt.Sprintf("%040d", 0)
	tagObj = fmt.Sprintf("%040d", 7)
)

// stubKey is the file name both stand-ins look a path up by.
func stubKey(path string) string {
	return strings.ReplaceAll(strings.TrimPrefix(path, "/"), "/", "%")
}

func (b *bitcheck) stubReset() {
	_ = os.RemoveAll(b.stub)
	for _, d := range []string{"api", "raw", "proxy"} {
		b.must(os.MkdirAll(filepath.Join(b.stub, d), 0o755))
	}
	b.stubFile("issues.json", "[]\n")
	b.stubFile("prs.json", "[]\n")
}

func (b *bitcheck) stubFile(name, content string) {
	b.must(os.WriteFile(filepath.Join(b.stub, filepath.FromSlash(name)), []byte(content), 0o644))
}

func (b *bitcheck) stubAPI(path, json string) { b.stubFile("api/"+stubKey(path), json+"\n") }

func (b *bitcheck) stubRuns(action, sha, runs string) {
	b.must(os.MkdirAll(filepath.Join(b.stub, "raw", filepath.FromSlash(action), sha), 0o755))
	b.stubFile("raw/"+action+"/"+sha+"/action.yml", runs)
}

// stubPR is one open pull request whose diff adds the given line.
func (b *bitcheck) stubPR(n int, line string) {
	b.stubFile("prs.json", fmt.Sprintf(`[{"number":%d,"title":"a proposal","author":{"login":"a-bot"},"headRefName":"b","files":[{"path":".github/workflows/ci.yml"}]}]`+"\n", n))
	b.stubFile(fmt.Sprintf("diff.%d", n), "diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml\n+"+line+"\n")
}

// stubWorld is what the real repositories answer, as far as these cases ask.
func (b *bitcheck) stubWorld() {
	b.stubAPI("repos/actions/checkout/commits/"+pinV4, `{"sha":"`+pinV4+`"}`)
	b.stubAPI("repos/actions/checkout/commits/"+pinV5, `{"sha":"`+pinV5+`"}`)
	b.stubAPI("repos/actions/checkout/git/ref/tags/v4.2.2", `{"object":{"sha":"`+pinV4+`","type":"commit"}}`)
	b.stubAPI("repos/actions/checkout/git/ref/tags/v5.0.0", `{"object":{"sha":"`+pinV5+`","type":"commit"}}`)
	b.stubAPI("repos/actions/checkout/releases/latest", `{"tag_name":"v7.0.1"}`)
	b.stubAPI("repos/astral-sh/setup-uv/commits/"+pinUV, `{"sha":"`+pinUV+`"}`)
	b.stubAPI("repos/astral-sh/setup-uv/git/ref/tags/v10.2.0", `{"object":{"sha":"`+pinUV+`","type":"commit"}}`)
	b.stubAPI("repos/astral-sh/setup-uv/releases/latest", `{"tag_name":"v10.2.0"}`)
	b.stubRuns("actions/checkout", pinV4, "runs:\n  using: node20\n  main: dist/index.js\n")
	b.stubRuns("actions/checkout", pinV5, "runs:\n  using: node24\n  main: dist/index.js\n")
	b.stubRuns("astral-sh/setup-uv", pinUV, "runs:\n  using: \"node24\"\n  main: \"dist/setup/index.cjs\"\n")
}

func rij(problems, needsHuman, openPRs int) string {
	return fmt.Sprintf(`{"schema":"check-remote-items/1","problems":%d,"needs_human":%d,"open_prs":%d}`, problems, needsHuman, openPRs)
}

func (b *bitcheck) remoteItems() {
	const ri = "check-remote-items"
	bin := b.w("stubbin")
	b.stub = b.w("stub")
	b.must(os.MkdirAll(bin, 0o755))
	b.must(os.WriteFile(filepath.Join(bin, "gh"), []byte(stubGH), 0o755))
	b.must(os.WriteFile(filepath.Join(bin, "curl"), []byte(stubCurl), 0o755))

	b.pathBeforeStubs = b.getenv("PATH")
	b.setenv("PATH", bin+string(os.PathListSeparator)+b.pathBeforeStubs)
	b.setenv("STUB_GH_DIR", b.stub)

	both := func(label string, want int, said string) {
		b.agree(label, ri, want)
		b.goSaid(label, said)
	}
	pin := func(n int, line string) {
		b.stubReset()
		b.stubWorld()
		b.stubPR(n, line)
	}

	b.stubReset()
	both("remote-items nothing open", 0, rij(0, 0, 0))

	b.stubReset()
	b.stubFile("issues.json", `[{"number":7,"title":"a report","author":{"login":"someone"},"createdAt":"2026-09-01T00:00:00Z"}]`+"\n")
	both("remote-items an open issue needs a reading and fails nothing", 0, rij(0, 1, 0))

	// ⚠ The shape pull request 2 had: a dependency bump with no action pin in it.
	b.stubReset()
	b.stubPR(2, `rusqlite = { version = "0.40.2" }`)
	both("remote-items a pull request with nothing checkable", 0, rij(0, 1, 1))

	pin(3, "      - uses: actions/checkout@"+noSuch+" # v4.2.2")
	both("remote-items a pin naming a commit the repository lacks", 1, rij(1, 0, 1))

	// ⛔ THE CHECK THE RUNTIME DEPRECATION GOT PAST, on the real commit that
	// declares it.
	pin(4, "      - uses: actions/checkout@"+pinV4+" # v4.2.2")
	both("remote-items a pin declaring a deprecated runtime", 1, rij(1, 1, 1))

	pin(5, "      - uses: actions/checkout@"+pinV5+" # v4.2.2")
	both("remote-items a label that has drifted from its pin", 1, rij(1, 1, 1))

	// ⭐ QUOTED, CURRENT AND THE LATEST: the one pin here every question passes.
	pin(6, "      - uses: astral-sh/setup-uv@"+pinUV+" # v10.2.0")
	both("remote-items a quoted current runtime at the latest release", 0, rij(0, 0, 1))

	// ⚠ An annotated tag is a tag object, followed to the commit it names.
	b.stubReset()
	b.stubWorld()
	b.stubAPI("repos/astral-sh/setup-uv/git/ref/tags/v9.9.9", `{"object":{"sha":"`+tagObj+`","type":"tag"}}`)
	b.stubAPI("repos/astral-sh/setup-uv/git/tags/"+tagObj, `{"object":{"sha":"`+pinUV+`","type":"commit"}}`)
	b.stubPR(7, "      - uses: astral-sh/setup-uv@"+pinUV+" # v9.9.9")
	both("remote-items an annotated tag followed to its commit", 0, rij(0, 1, 1))

	pin(8, "      - uses: astral-sh/setup-uv@"+pinUV+" # v0.0.0-none")
	both("remote-items a label that is not a tag", 0, rij(0, 2, 1))

	pin(9, "      - uses: actions/checkout@"+pinV5)
	both("remote-items a bare pin with no label", 0, rij(0, 1, 1))

	// ⚠ A PIN THE DIFF REMOVES IS NOT THE PROPOSAL. Only added lines are read.
	pin(11, "      - uses: astral-sh/setup-uv@"+pinUV+" # v10.2.0")
	f, err := os.OpenFile(filepath.Join(b.stub, "diff.11"), os.O_APPEND|os.O_WRONLY, 0o644)
	b.must(err)
	_, werr := fmt.Fprintf(f, "-      - uses: actions/checkout@%s # v4.2.2\n", pinV4)
	b.must(errors.Join(werr, f.Close()))
	both("remote-items a pin the diff removes is not checked", 0, rij(0, 0, 1))

	// ⚠ TWO PINS ON ONE LINE, which YAML's flow style allows. ⛔ Added when a
	// mutation reading only the first match per line survived.
	pin(12, "      steps: [{uses: astral-sh/setup-uv@"+pinUV+"}, {uses: actions/checkout@"+pinV4+"}]")
	both("remote-items two pins on one line are both read", 1, rij(1, 2, 1))

	b.stubReset()
	b.stubPR(10, "      - uses: actions/checkout@"+pinV5+" # v5.0.0")
	_ = os.Remove(filepath.Join(b.stub, "diff.10"))
	both("remote-items a diff that cannot be read", 0, rij(0, 1, 1))

	// ⛔ COULD NOT RUN, BOTH WAYS, and never a pass: a CLI with no session, and a
	// listing the API refused.
	b.stubReset()
	b.stubFile("auth-fails", "")
	b.agree("remote-items an unauthenticated gh cannot run", ri, 2)

	b.stubReset()
	b.stubFile("issues.fail", "HTTP 502: the listing failed\n")
	b.agree("remote-items a listing that fails cannot run", ri, 2)
}

// ============================================================================
// mine-repo
// ============================================================================
//
// ⭐ A HELPER, NOT A CHECK, AND ASSERTED BY WHAT IT LEAVES BEHIND: the exit code,
// what --json printed, and every file and directory it wrote, in the shell half's
// `mr_files` form.
//
// ⛔ THE STUBS ENFORCE WHAT THE HALVES ONLY SAID. The proxy stand-in answers 420 to
// any user-agent but curl's own, the git stand-in refuses a clone without the
// low-speed bound or a depth of one, and it refuses any URL but the target's - so
// a port that dropped any of them fails a case, not a reading.

// stubGit is the shell half's `git` stand-in, byte for byte.
const stubGit = `#!/bin/sh
# A stand-in for git that answers a clone out of a fixture and hands every other
# command to the real git. The commit it makes has a fixed author, committer and
# date, so every implementation cloning one fixture reads one commit.
set -u
REAL=${STUB_GIT_REAL:?}
D=${STUB_GH_DIR:?}
clone=0
bound=0
depth=0
prev=""
src=""
dst=""
for a in "$@"; do
  case "$a" in
    clone) clone=1 ;;
    http.lowSpeedLimit=1024 | http.lowSpeedTime=60) bound=$((bound + 1)) ;;
  esac
  [ "$prev" = "--depth" ] && [ "$a" = 1 ] && depth=1
  prev=$a
  src=$dst
  dst=$a
done
[ "$clone" = 1 ] || exec "$REAL" "$@"
if [ "$bound" != 2 ] || [ "$depth" != 1 ]; then
  printf 'stub git: an unbounded or full clone is refused\n' >&2
  exit 128
fi
if [ -f "$D/clone.fail" ] || [ "$src" != "https://github.com/stub-owner/stub-repo.git" ]; then
  printf 'fatal: repository %s not found\n' "$src" >&2
  exit 128
fi
mkdir -p "$dst" && cp -R "${STUB_GIT_FIXTURE:?}/." "$dst/" || exit 128
if [ -f "$D/fixture.vendor" ]; then
  mkdir -p "$dst/vendor/bundle/gems" && printf 'gem\n' >"$dst/vendor/bundle/gems/g.rb" || exit 128
fi
cd "$dst" || exit 128
GIT_AUTHOR_NAME=stub GIT_AUTHOR_EMAIL=stub GIT_COMMITTER_NAME=stub GIT_COMMITTER_EMAIL=stub
GIT_AUTHOR_DATE=2026-01-01T00:00:00Z GIT_COMMITTER_DATE=2026-01-01T00:00:00Z
export GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL
export GIT_AUTHOR_DATE GIT_COMMITTER_DATE
"$REAL" init -q -b main && "$REAL" add -A &&
  "$REAL" -c commit.gpgsign=false commit -qm fixture || exit 128
"$REAL" rev-parse HEAD >"$D/clone.sha"
`

const mrTarget = "stub-owner/stub-repo"

// mrExec runs mine-repo into a fresh output directory and keeps what it did: the
// exit code, everything it printed, and mrFiles - the shell half's `mr.kept`.
func (b *bitcheck) mrExec(flags ...string) {
	_ = os.RemoveAll(b.mrOut)
	rc, out := b.run(append([]string{"mine-repo", "--json"}, flags...)...)
	var k strings.Builder
	fmt.Fprintf(&k, "exit %d\n", rc)
	// ⚠ `sed` over the output: one substitution per line, and a last line with
	// no newline stays without one.
	k.WriteString(sedFirst(out, "choose another -Out,", "choose another --out,"))
	k.WriteString(b.mrFiles())
	b.kept = k.String()
}

// sedFirst is `sed 's/OLD/NEW/'` with a literal: the first occurrence on each line.
func sedFirst(text, old, repl string) string {
	lines := strings.SplitAfter(text, "\n")
	for i, l := range lines {
		lines[i] = strings.Replace(l, old, repl, 1)
	}
	return strings.Join(lines, "")
}

func (b *bitcheck) keptLines() []string { return strings.Split(b.kept, "\n") }

// mrFiles is the shell half's `mr_files`: every directory, then every file - a
// JSON file as its record count and its `jq -S -c` form, PROVENANCE.md whole but
// for the line naming its writer and the moment it ran, anything else by cksum.
func (b *bitcheck) mrFiles() string {
	if st, err := os.Stat(b.mrOut); err != nil || !st.IsDir() {
		return "no output directory\n"
	}
	var dirs, files []string
	_ = filepath.WalkDir(b.mrOut, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(b.mrOut, p)
		name := "."
		if rel != "." {
			name = "./" + filepath.ToSlash(rel)
		}
		switch {
		case d.IsDir():
			dirs = append(dirs, name)
		case d.Type().IsRegular():
			files = append(files, name)
		}
		return nil
	})
	sort.Strings(dirs)
	sort.Strings(files)
	var o strings.Builder
	for _, d := range dirs {
		o.WriteString("dir  " + d + "\n")
	}
	provWriter := regexp.MustCompile("^Fetched .* by `.*`\\.$")
	for _, f := range files {
		abs := filepath.Join(b.mrOut, filepath.FromSlash(f))
		switch {
		case strings.HasSuffix(f, ".json"):
			counts := jqOut(abs, `if type == "array" then length else "-" end`)
			fmt.Fprintf(&o, "file %s records %s\n", f, strings.ReplaceAll(counts, "\n", " "))
			for _, l := range splitLines([]byte(jqOut(abs, "-S", "-c", "."))) {
				o.WriteString("     " + string(l) + "\n")
			}
		case strings.HasSuffix(f, "/PROVENANCE.md"):
			o.WriteString("file " + f + "\n")
			for _, l := range strings.SplitAfter(readString(abs), "\n") {
				if l == "" {
					continue
				}
				body, nl := strings.CutSuffix(l, "\n")
				if provWriter.MatchString(body) {
					body = "Fetched <when> by <writer>."
				}
				body = strings.Replace(body, "tree: -NoClone was passed", "tree: --no-clone was passed", 1)
				o.WriteString("     " + body)
				if nl {
					o.WriteString("\n")
				}
			}
		default:
			data, _ := os.ReadFile(abs)
			fmt.Fprintf(&o, "file %s cksum %s\n", f, posixCksum(data))
		}
	}
	return o.String()
}

// jqOut is `jq ARGS FILE 2>/dev/null`: what jq printed, whatever it exited.
func jqOut(file string, args ...string) string {
	var out bytes.Buffer
	cmd := exec.Command("jq", append(args, file)...)
	cmd.Stdout = &out
	_ = cmd.Run()
	return out.String()
}

// posixCksum is `cksum <FILE | cut -d' ' -f1-2`: POSIX.2's CRC over the bytes and
// then the length, and the length.
func posixCksum(data []byte) string {
	var table [256]uint32
	for i := range table {
		c := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if c&0x80000000 != 0 {
				c = c<<1 ^ 0x04c11db7
			} else {
				c <<= 1
			}
		}
		table[i] = c
	}
	var crc uint32
	step := func(x byte) { crc = crc<<8 ^ table[byte(crc>>24)^x] }
	for _, x := range data {
		step(x)
	}
	for n := len(data); n > 0; n >>= 8 {
		step(byte(n))
	}
	return fmt.Sprintf("%d %d", ^crc, len(data))
}

// mrAgree is one mine-repo case.
func (b *bitcheck) mrAgree(label string, want int, flags ...string) {
	b.mrExec(flags...)
	got := strings.TrimPrefix(b.keptLines()[0], "exit ")
	if got != strconv.Itoa(want) {
		b.h.bad(fmt.Sprintf("%s: go answered %s, expected %d", label, got, want))
		return
	}
	b.h.ok(fmt.Sprintf("%s: go (exit %d)", label, want))
}

// ⛔ AN EXIT CODE OF 0 SAYS A FETCH ENDED, NOT WHAT IT GOT. These assert what the
// case just run left behind.
func (b *bitcheck) mrSaid(label, want string) {
	said := ""
	if l := b.keptLines(); len(l) > 1 {
		said = l[1]
	}
	if said == want {
		b.h.ok(label + ": go said " + want)
	} else {
		b.h.bad(label + ": go said " + said + ", expected " + want)
	}
}

func (b *bitcheck) mrHas(label, needle string) {
	if anyContains(b.keptLines(), needle) {
		b.h.ok(label + ": go left " + needle)
	} else {
		b.h.bad(label + ": go left no " + needle)
	}
}

func (b *bitcheck) mrLacks(label, needle string) {
	if anyContains(b.keptLines(), needle) {
		b.h.bad(label + ": go left " + needle)
	} else {
		b.h.ok(label + ": go left no " + needle)
	}
}

func (b *bitcheck) mrJSON(route, commit string, gaps int) string {
	return fmt.Sprintf(`{"schema":"mine-repo/1","target":"%s","route":"%s","commit":"%s","gaps":%d,"dest":"%s"}`,
		mrTarget, route, commit, gaps, b.mrOut+"/stub-owner__stub-repo")
}

func (b *bitcheck) mrProxy(path, json string) {
	b.stubFile("proxy/"+stubKey(path), json+"\n")
}

// mrFullPage is a proxy page of exactly one hundred records, each carrying one
// "url", which is what decides whether another page follows.
func (b *bitcheck) mrFullPage(path string) {
	var s strings.Builder
	s.WriteString("[")
	for i := 1; i <= 100; i++ {
		if i > 1 {
			s.WriteString(",")
		}
		fmt.Fprintf(&s, `{"url":"https://example.invalid/%d","body":"[unclosed %d"}`, i, i)
	}
	s.WriteString("]\n")
	b.stubFile("proxy/"+stubKey(path), s.String())
}

func (b *bitcheck) mrWorldGH() {
	t := mrTarget
	b.stubAPI("rate_limit", `{"resources":{}}`)
	b.stubAPI("repos/pkgforge-dev/reverse-proxies", `{"full_name":"pkgforge-dev/reverse-proxies"}`)
	b.stubAPI("repos/"+t, `{"full_name":"stub-owner/stub-repo","open_issues_count":2}`)
	b.stubAPI("repos/"+t+"/issues?state=all&per_page=100", `[{"url":"https://example.invalid/i/1","body":"see [the report](x"},{"url":"https://example.invalid/i/2","pull_request":{"url":"https://example.invalid/p/2"}}]`)
	b.stubAPI("repos/"+t+"/issues/comments?per_page=100", `[{"url":"https://example.invalid/c/1","body":"log: [ERROR] ]["}]`)
	b.stubAPI("repos/"+t+"/pulls/comments?per_page=100", `[]`)
	b.stubAPI("repos/"+t+"/releases?per_page=100", `[{"url":"https://example.invalid/r/1","tag_name":"v1.0.0"}]`)
	b.stubAPI("repos/"+t+"/tags?per_page=100", `[{"name":"v1.0.0"}]`)
	b.stubAPI("graphql", `{"data":{"repository":{"discussions":{"nodes":[]}}}}`)
}

// ⭐ THE ISSUES SPAN TWO PAGES AND THE SECOND IS SHORT, with brackets in the bodies
// of both - the shape that broke the halves' first joiner.
func (b *bitcheck) mrWorldProxy() {
	t := mrTarget
	b.mrProxy("repos/pkgforge-dev/reverse-proxies", `{"full_name":"pkgforge-dev/reverse-proxies"}`)
	b.mrProxy("repos/"+t, `{"full_name":"stub-owner/stub-repo"}`)
	b.mrFullPage("repos/" + t + "/issues?state=all&per_page=100&page=1")
	b.mrProxy("repos/"+t+"/issues?state=all&per_page=100&page=2", `[{"url":"https://example.invalid/i/101","body":"]["},{"url":"https://example.invalid/i/102","body":"[x](y"}]`)
	b.mrProxy("repos/"+t+"/issues/comments?per_page=100&page=1", `[{"url":"https://example.invalid/c/1","body":"see [the report](x"},{"url":"https://example.invalid/c/2","body":"log: [ERROR] [WARN]"}]`)
	b.mrProxy("repos/"+t+"/pulls/comments?per_page=100&page=1", `[]`)
	b.mrProxy("repos/"+t+"/releases?per_page=100&page=1", `[{"url":"https://example.invalid/r/1","tag_name":"v1.0.0"}]`)
	b.mrProxy("repos/"+t+"/tags?per_page=100&page=1", `[{"name":"v1.0.0"}]`)
}

func (b *bitcheck) mineRepo() {
	t := mrTarget
	b.mrOut = b.w("mined")
	gitBin := b.w("stubgit")
	fix := b.w("mrfixture")
	b.must(os.MkdirAll(gitBin, 0o755))
	b.must(os.WriteFile(filepath.Join(gitBin, "git"), []byte(stubGit), 0o755))

	// ⭐ THE FIXTURE CARRIES EVERY SHAPE THE TRIM DECIDES ON: each directory name
	// it deletes, one of them nested, a FILE named like one - `find -type d` must
	// keep it - and a `vendor/` that stays. Dot-directories are in it on purpose.
	for _, d := range []string{"src", "docs", "node_modules/dep", "sub/target", "build", "dist", ".next",
		".venv/bin", "lib/__pycache__", "vendor"} {
		b.must(os.MkdirAll(filepath.Join(fix, filepath.FromSlash(d)), 0o755))
	}
	for name, content := range map[string]string{
		"README.md":      "fixture\n",
		"src/main.c":     "int main(void) { return 0; }\n",
		"docs/build":     "a file named like a directory the trim deletes\n",
		"vendor/keep.rb": "kept\n",
	} {
		b.must(os.WriteFile(filepath.Join(fix, filepath.FromSlash(name)), []byte(content), 0o644))
	}
	for _, d := range []string{"node_modules/dep", "sub/target", "build", "dist", ".next", ".venv/bin", "lib/__pycache__"} {
		b.must(os.WriteFile(filepath.Join(fix, filepath.FromSlash(d), "junk"), []byte("junk\n"), 0o644))
	}

	realGit, err := exec.LookPath("git")
	b.must(err)
	b.setenv("STUB_GIT_REAL", realGit)
	b.setenv("STUB_GIT_FIXTURE", fix)
	pathBeforeGit := b.getenv("PATH")
	b.setenv("PATH", gitBin+string(os.PathListSeparator)+pathBeforeGit)

	const mrf = "./stub-owner__stub-repo"

	// -- what it refuses to start with ------------------------------------------

	b.stubReset()
	b.mrAgree("mine-repo --selftest joins and guards", 0, "--selftest")
	b.mrSaid("mine-repo --selftest joins and guards", `{"schema":"mine-repo-selftest/1","cases":4,"failed":0}`)

	b.mrAgree("mine-repo no target cannot run", 2, "--out", b.mrOut)
	b.mrAgree("mine-repo a target with two slashes is refused", 2, "--out", b.mrOut, "a/b/c")
	b.mrAgree("mine-repo an unknown argument is refused", 2, "--bogus", t)
	b.mrAgree("mine-repo a second target is refused, not taken", 2, t, "other/x")
	b.mrAgree("mine-repo an unknown route is refused, not probed", 2, "--route", "prxy", t)
	b.mrAgree("mine-repo a route in another case is refused", 2, "--route", "GH", t)
	b.mrAgree("mine-repo --out with no value cannot run", 2, t, "--out")

	// ⛔ THE CORPUS IS THE EVIDENCE, so an --out this repository ignores is refused
	// before anything is fetched. The rule is planted in the scratch tree's own
	// exclude file rather than in its tracked .gitignore, and taken out after.
	b.stubReset()
	b.mrWorldGH()
	saved := b.mrOut
	b.mrOut = b.p("mined")
	const excl = ".git/info/exclude"
	if _, err := os.Stat(b.p(excl)); err == nil {
		b.save(excl, "exclude.saved")
	} else {
		b.must(os.WriteFile(b.w("exclude.saved"), nil, 0o644))
	}
	b.appendTo(excl, "mined/\n")
	b.mrAgree("mine-repo an --out the repository ignores is refused", 2, "--route", "gh", "--out", b.mrOut, t)
	b.mrHas("mine-repo an --out the repository ignores is refused", "is ignored by this repository")
	b.mrLacks("mine-repo an --out the repository ignores is refused", "api/repo.json")
	b.put("exclude.saved", excl)
	_ = os.RemoveAll(b.mrOut)
	b.mrOut = saved

	// -- the gh route -------------------------------------------------------------

	b.stubReset()
	b.mrWorldGH()
	b.mrAgree("mine-repo the gh route fetches every source and keeps the tree", 0, "--route", "gh", "--out", b.mrOut, t)
	// ⚠ Read once and used by every case after, as the shell half's MR_SHA was:
	// the stand-in's fixed author and date make every clone the same commit.
	sha := strings.TrimRight(readString(filepath.Join(b.stub, "clone.sha")), "\n")
	b.mrSaid("mine-repo the gh route fetches every source and keeps the tree", b.mrJSON("gh", sha, 0))
	b.mrHas("mine-repo the gh route keeps the source", "file "+mrf+"/tree/src/main.c")
	b.mrHas("mine-repo the gh route keeps a FILE named like junk", "file "+mrf+"/tree/docs/build")
	b.mrHas("mine-repo the gh route keeps vendor/", "file "+mrf+"/tree/vendor/keep.rb")
	b.mrLacks("mine-repo the gh route drops the git directory", mrf+"/tree/.git")
	b.mrLacks("mine-repo the gh route trims node_modules", "node_modules")
	b.mrLacks("mine-repo the gh route trims a nested target", mrf+"/tree/sub/target")
	b.mrLacks("mine-repo the gh route trims .next", mrf+"/tree/.next")
	b.mrLacks("mine-repo the gh route trims .venv", mrf+"/tree/.venv")
	b.mrLacks("mine-repo the gh route trims __pycache__", "__pycache__")
	b.mrHas("mine-repo the gh route fetches discussions", "file "+mrf+"/api/discussions.json")
	b.mrHas("mine-repo the gh route says it missed nothing", "Nothing. Every source above answered.")

	b.stubReset()
	b.mrWorldGH()
	b.mrAgree("mine-repo auto takes gh when gh answers", 0, "--out", b.mrOut, t)
	b.mrSaid("mine-repo auto takes gh when gh answers", b.mrJSON("gh", sha, 0))

	b.stubReset()
	b.mrWorldGH()
	_ = os.Remove(filepath.Join(b.stub, "api", stubKey("repos/"+t)))
	b.mrAgree("mine-repo a subject gh cannot fetch fails", 1, "--route", "gh", "--out", b.mrOut, t)
	b.mrHas("mine-repo a subject gh cannot fetch fails", "mine-repo: could not fetch repos/"+t)
	b.mrHas("mine-repo a subject gh cannot fetch fails", "mine-repo: control says: reachable")

	b.stubReset()
	b.mrWorldGH()
	_ = os.Remove(filepath.Join(b.stub, "api", stubKey("repos/pkgforge-dev/reverse-proxies")))
	b.mrAgree("mine-repo a control gh cannot reach is written down", 0, "--route", "gh", "--out", b.mrOut, t)
	b.mrHas("mine-repo a control gh cannot reach is written down", "UNREACHABLE (pkgforge-dev/reverse-proxies did not answer)")

	b.stubReset()
	b.mrWorldGH()
	_ = os.Remove(filepath.Join(b.stub, "api", stubKey("repos/"+t+"/tags?per_page=100")))
	b.mrAgree("mine-repo a list gh cannot fetch is a named gap", 0, "--route", "gh", "--out", b.mrOut, t)
	b.mrSaid("mine-repo a list gh cannot fetch is a named gap", b.mrJSON("gh", sha, 1))
	b.mrHas("mine-repo a list gh cannot fetch is a named gap", "tags: gh could not fetch /repos/"+t+"/tags")

	b.stubReset()
	b.mrWorldGH()
	_ = os.Remove(filepath.Join(b.stub, "api", "graphql"))
	b.mrAgree("mine-repo failed discussions are a named gap and no file", 0, "--route", "gh", "--out", b.mrOut, t)
	b.mrHas("mine-repo failed discussions are a named gap and no file", "discussions: the GraphQL query failed")
	b.mrLacks("mine-repo failed discussions are a named gap and no file", "api/discussions.json")

	b.stubReset()
	b.mrWorldGH()
	b.mrAgree("mine-repo --no-clone keeps no tree and says so", 0, "--route", "gh", "--no-clone", "--out", b.mrOut, t)
	b.mrSaid("mine-repo --no-clone keeps no tree and says so", b.mrJSON("gh", "-", 1))
	b.mrHas("mine-repo --no-clone keeps no tree and says so", "tree: --no-clone was passed")

	b.stubReset()
	b.mrWorldGH()
	b.stubFile("clone.fail", "")
	b.mrAgree("mine-repo a failed clone is a named gap", 0, "--route", "gh", "--out", b.mrOut, t)
	b.mrSaid("mine-repo a failed clone is a named gap", b.mrJSON("gh", "-", 1))
	b.mrHas("mine-repo a failed clone is a named gap", "tree: the clone failed")

	// -- the proxy route ----------------------------------------------------------

	b.stubReset()
	b.mrWorldProxy()
	b.mrAgree("mine-repo the proxy route pages and joins by hand", 0, "--route", "proxy", "--out", b.mrOut, t)
	b.mrSaid("mine-repo the proxy route pages and joins by hand", b.mrJSON("proxy", sha, 1))
	b.mrHas("mine-repo the proxy route joins both pages", "file "+mrf+"/api/issues.json records 102")
	b.mrHas("mine-repo the proxy route keeps brackets in bodies", "file "+mrf+"/api/comments.json records 2")
	b.mrLacks("mine-repo the proxy route leaves no page behind", ".page.")
	b.mrHas("mine-repo the proxy route names what it cannot reach", "discussions: NOT FETCHED")
	b.mrHas("mine-repo the proxy route reads its control", "reachable (pkgforge-dev/reverse-proxies answered 200)")

	b.stubReset()
	b.mrWorldProxy()
	b.stubFile("auth-fails", "")
	b.mrAgree("mine-repo auto takes the proxy when gh is not authenticated", 0, "--out", b.mrOut, t)
	b.mrSaid("mine-repo auto takes the proxy when gh is not authenticated", b.mrJSON("proxy", sha, 1))

	b.stubReset()
	b.mrWorldProxy()
	b.stubFile("proxy/"+stubKey("repos/"+t+"/issues/comments?per_page=100&page=1")+".code", "502\n")
	b.mrAgree("mine-repo a failed page is a named gap and leaves nothing", 0, "--route", "proxy", "--out", b.mrOut, t)
	b.mrHas("mine-repo a failed page is a named gap and leaves nothing", "comments: proxy returned 502 on page 1")
	b.mrLacks("mine-repo a failed page is a named gap and leaves nothing", "api/comments.json")

	b.stubReset()
	b.mrWorldProxy()
	_ = os.Remove(filepath.Join(b.stub, "proxy", stubKey("repos/"+t)))
	b.mrAgree("mine-repo a subject the proxy cannot find fails beside its control", 1, "--route", "proxy", "--out", b.mrOut, t)
	b.mrHas("mine-repo a subject the proxy cannot find fails beside its control", "mine-repo: proxy returned 404 for repos/"+t)

	b.stubReset()
	b.mrWorldProxy()
	b.stubFile("proxy/"+stubKey("repos/pkgforge-dev/reverse-proxies")+".code", "500\n")
	b.mrAgree("mine-repo a control the proxy cannot reach is written down", 0, "--route", "proxy", "--out", b.mrOut, t)
	b.mrHas("mine-repo a control the proxy cannot reach is written down", "UNREACHABLE (pkgforge-dev/reverse-proxies answered 500)")

	// -- what only the port does --------------------------------------------------

	// ⛔ A LIST CUT OFF AT THE PAGE CAP IS NAMED. Both halves stopped after ten full
	// pages and said "ok"; ten full pages and an eleventh behind them is the shape.
	b.stubReset()
	b.mrWorldProxy()
	for page := 1; page <= 11; page++ {
		b.mrFullPage(fmt.Sprintf("repos/%s/tags?per_page=100&page=%d", t, page))
	}
	b.mrAgree("mine-repo a list cut off at the page cap is a named gap", 0, "--route", "proxy", "--out", b.mrOut, t)
	b.mrHas("mine-repo a list cut off at the page cap is a named gap", "tags: stopped at 10 pages of 100")
	b.mrHas("mine-repo a list cut off at the page cap keeps what it got", "file "+mrf+"/api/tags.json records 1000")

	// ⚠ `vendor/bundle` IS A PATH. The sh half listed it where `find -name` takes a
	// name, so it never matched, and its twin left it out.
	b.stubReset()
	b.mrWorldGH()
	b.stubFile("fixture.vendor", "")
	b.mrAgree("mine-repo vendor/bundle is trimmed and vendor/ is not", 0, "--route", "gh", "--out", b.mrOut, t)
	b.mrLacks("mine-repo vendor/bundle is trimmed and vendor/ is not", "vendor/bundle")
	b.mrHas("mine-repo vendor/bundle is trimmed and vendor/ is not", "file "+mrf+"/tree/vendor/keep.rb")

	_ = os.RemoveAll(b.mrOut)
	// ⛔ AND THE STAND-INS LEAVE PATH BEFORE THE NEXT SECTION, so no check below is
	// answered by a stub it was never meant to meet.
	b.setenv("PATH", b.pathBeforeStubs)
}

// ============================================================================
// check-placeholders
// ============================================================================
//
// ⛔ EVERY NEEDLE BELOW IS ASSEMBLED RATHER THAN WRITTEN, and that is not style.
// check-placeholders reads every tracked file, this file is one, and the shell
// half's first version turned the clean tree red for exactly that reason.

var (
	br  = "{" + "{"
	brc = "}" + "}"
)

func (b *bitcheck) placeholders() {
	const md, yml = "tools/check/plant.md", "tools/check/plant.yml"
	b.agree("placeholders clean tree", "check-placeholders", 0)

	// ⛔ THE FOUR CATEGORIES ARE FOUR RULES and each is planted on its own.
	b.write(md, "a survived "+br+"PLACEHOLDER"+brc+" here\n")
	b.agree("placeholders a double-brace placeholder is refused", "check-placeholders", 1)
	b.unplant(md)

	b.write(md, "text\n<!-- "+"TEMPLATE"+": fill this in -->\n")
	b.agree("placeholders a template guidance comment is refused", "check-placeholders", 1)
	b.unplant(md)

	b.write(md, "contact "+"CHANGE"+"ME"+" for details\n")
	b.agree("placeholders a stand-in value is refused", "check-placeholders", 1)
	b.unplant(md)

	// ⚠ THE OWNER-SLASH-REPO GENERIC IS RECOMMENDED IN A PUBLIC DOCUMENT, so it is a
	// defect only where it is configuration rather than prose.
	b.write(md, "clone "+"OWNER"+"/"+"REPO"+" to begin\n")
	b.agree("placeholders the generic in a .md is accepted", "check-placeholders", 0)
	b.unplant(md)

	b.write(yml, "repo: "+"OWNER"+"/"+"REPO"+"\n")
	b.agree("placeholders the generic in a config file is refused", "check-placeholders", 1)
	b.unplant(yml)

	// ⭐ THE TWO SHAPES THAT MUST NOT FIRE: an Actions expression and a Go template
	// both carry a double brace.
	b.write(yml, "run: echo $"+br+" github.sha "+brc+"\n")
	b.agree("placeholders an Actions expression is accepted", "check-placeholders", 0)
	b.unplant(yml)

	b.write(yml, "info --format "+br+"json .Host.Arch"+brc+"\n")
	b.agree("placeholders a Go template is accepted", "check-placeholders", 0)
	b.unplant(yml)

	b.agree("placeholders clean tree again", "check-placeholders", 0)
}

// ============================================================================
// check-no-secrets
// ============================================================================
//
// ⛔ EVERY NEEDLE BELOW IS ASSEMBLED AND NONE OF THEM IS WRITTEN. A credential's
// SHAPE is exactly what check-no-secrets looks for, and this file is in its scope.

var (
	akid     = "AKI" + "A" + "IOSFODNN7EXAMPLE"
	keyBlock = "BEGIN" + " " + "PRIVATE KEY"
	urlCred  = "https:" + "//" + "bob" + ":" + "hunter22x" + "@" + "example.invalid"
	mail     = "someone" + "@" + "example.invalid"
	homeP    = "/home" + "/" + "alice" + "/work"
	homeG    = "/home" + "/" + "runner" + "/work"

	// ⚠ SIXTEEN HEX DIGITS IS UNDER THE TWENTY-FOUR FLOOR, which is what lets the
	// pieces be written and the assembled runs not be.
	hx    = "0123456789abcdef"
	hex40 = hx + hx + "01234567"
	hex64 = hx + hx + hx + hx
	hex46 = hex40 + "012345"
)

func (b *bitcheck) noSecrets() {
	const ns, md = "check-no-secrets", "tools/check/plant.md"
	b.agree("no-secrets clean tree", ns, 0)

	// ⛔ A CREDENTIAL FILE, TRACKED. It is force-added, because the rule is about
	// a file that reached the index and a fixture that only works while an ignore
	// list has a hole is a fixture that breaks when the hole is closed.
	b.write("tools/check/id_ecdsa", "not a real key\n")
	b.plantTracked("tools/check/id_ecdsa")
	b.agree("no-secrets a tracked credential file is refused", ns, 1)
	b.unplantTracked("tools/check/id_ecdsa")

	// ⭐ AND THE WAIVER, WHICH IS A DIFFERENT BRANCH: the file a project is
	// SUPPOSED to commit.
	b.write("tools/check/.env.example", "TOKEN=\n")
	b.agree("no-secrets an .example credential template is accepted", ns, 0)
	b.unplant("tools/check/.env.example")

	b.write(md, "key "+akid+" here\n")
	b.agree("no-secrets an aws access key id is refused", ns, 1)
	b.unplant(md)

	b.write(md, "-----"+keyBlock+"-----\n")
	b.agree("no-secrets a private key block is refused", ns, 1)
	b.unplant(md)

	b.write(md, "clone "+urlCred+" now\n")
	b.agree("no-secrets a password in a url is refused", ns, 1)
	b.unplant(md)

	// ⛔ AN EMAIL IS NOT A DEFAULT-MODE FINDING: --public is a second question, not
	// the same rules stricter.
	b.write(md, "write to "+mail+"\n")
	b.agree("no-secrets an email address is accepted without --public", ns, 0)
	b.unplant(md)

	// ⚠ A BINARY FILE IS SKIPPED, and the plant carries a real key so the case
	// cannot pass by the file being uninteresting.
	b.write(md, "key "+akid+"\x00here\n")
	b.agree("no-secrets a key inside a binary file is out of scope", ns, 0)
	b.unplant(md)

	b.agree("no-secrets clean tree again", ns, 0)

	// -- the public rules ---------------------------------------------------------

	b.agree("no-secrets clean tree, public", ns, 0, "--public")

	b.write(md, "write to "+mail+"\n")
	b.agree("no-secrets an email address is refused with --public", ns, 1, "--public")
	b.unplant(md)

	b.write(md, "built in "+homeP+"\n")
	b.agree("no-secrets an absolute home path is refused", ns, 1, "--public")
	b.unplant(md)

	// ⚠ Narrowed rather than switched off: /home/runner/ is a well-known generic
	// path, not a fingerprint of anybody's machine.
	b.write(md, "built in "+homeG+"\n")
	b.agree("no-secrets a generic runner home path is accepted", ns, 0, "--public")
	b.unplant(md)

	b.write(md, "a bare "+hex40+" here\n")
	b.agree("no-secrets a bare long hex run is refused", ns, 1, "--public")
	b.unplant(md)

	// ⚠ A PINNED ACTION IS THE SAFE PRACTICE.
	b.write("tools/check/plant.yml", "uses: owner/repo@"+hex40+"\n")
	b.agree("no-secrets a pinned action commit is accepted", ns, 0, "--public")
	b.unplant("tools/check/plant.yml")

	// ⛔ THE ALLOW EXPRESSIONS READ THE OUTPUT LINE AND NOT THE TEXT. The lockfile
	// allowance is anchored to the `path:lineno:` prefix, so a port that ran its
	// allowances over the file's own line would refuse every registry digest.
	b.write("tools/check/plant-Cargo.lock", "checksum = \""+hex64+"\"\n")
	b.agree("no-secrets a registry lockfile digest is accepted", ns, 0, "--public")
	b.unplant("tools/check/plant-Cargo.lock")

	// ⛔ AN ALLOWED ITEM IS DELETED FROM THE LINE RATHER THAN THE LINE BEING
	// DROPPED, or an allowed digest beside a real one takes the real one with it.
	b.write(md, "sha256:"+hex64+" and "+hex40+"\n")
	b.agree("no-secrets an allowed digest beside a bare one is still refused", ns, 1, "--public")
	b.unplant(md)

	b.write(md, "Infohash: "+hex40+"\n")
	b.agree("no-secrets an announce infohash is accepted", ns, 0, "--public")
	b.unplant(md)

	// ⛔ THE TRAILING CLASS IS THE ANCHOR: without it `{40}` blanks the first forty
	// characters of a LONGER run and leaves a remainder too short to report.
	b.write(md, "Infohash: "+hex46+"\n")
	b.agree("no-secrets a longer run after the infohash field is refused", ns, 1, "--public")
	b.unplant(md)

	b.agree("no-secrets clean tree again, public", ns, 0, "--public")
}

// ============================================================================
// check-ignores
// ============================================================================
//
// ⚠ WHAT THESE CASES CANNOT REACH is the specimen self-check - the branch that
// fires when a shape is narrowed OUT of check-no-secrets while its specimen stays
// behind. That plant is a change to Go source, and this binary is the subject, so
// a case cannot rebuild it. ⛔ It is mutation-proved in TODO/ci.md and recorded
// as a measurement rather than claimed as a case here.

func (b *bitcheck) ignores() {
	const gi = ".gitignore"
	b.agree("ignores clean tree", "check-ignores", 0)

	// ⛔ A HOLE IN THE IGNORE LIST, the shape found by hand on 2026-09-15: a name
	// check-no-secrets refuses that `git add -A` would stage anyway.
	b.save(gi, "gitignore.orig")
	b.rewrite("gitignore.orig", gi, func(l string) (string, bool) { return l, l != "*.pem" })
	b.agree("ignores a credential shape missing from .gitignore is refused", "check-ignores", 1)
	b.put("gitignore.orig", gi)

	// ⚠ AND MORE THAN ONE HOLE IS STILL ONE REFUSAL. A rewrite that stopped after
	// the first finding would pass the case above and lose the rest of the report.
	b.rewrite("gitignore.orig", gi, func(l string) (string, bool) {
		return l, l != "*.pem" && l != "id_rsa" && l != "credentials.json"
	})
	b.agree("ignores several holes at once are still one refusal", "check-ignores", 1)
	b.put("gitignore.orig", gi)

	b.agree("ignores clean tree again", "check-ignores", 0)
}

// ============================================================================
// check-adapters
// ============================================================================
//
// ⛔ It exists because `capture-client` runs 21 and 22 both hung in *Install the
// client* on the RELEASE lane, and an unbounded fetch is the only thing in that
// route that can wait forever.

// The Go adapter's bounded clone, and its argument list to the end of the line.
var timeoutAfter = regexp.MustCompile(`exec\.Command\("timeout", "600", "git", "clone", (.*),$`)

// plantedPeerIDSelftest plants the defect `capture-client` run 39577251480
// found - the peer ID's hex looked for as a substring of the transcript file -
// into a scratch copy of the guard, builds that copy and asks its selftest,
// which must refuse. `CLIENT-05`.
//
// ⛔ A SELFTEST NOTHING HAS SEEN FAIL MAY BE ASKING NOTHING. The plant is the
// old line itself, put first in transcriptCarriesPeerID so the rest of the
// function is never reached, and the case that has to fail is named: aria2's
// percent-encoded peer ID. ⚠ The planted copy is BUILT, because the selftest
// is in this binary; the tree is put back before anything else runs.
func (b *bitcheck) plantedPeerIDSelftest() {
	const tp = "tools/check/transcriptpeer.go"
	const line = "\tids, err := transcriptPeerIDs(doc)"
	b.save(tp, "transcriptpeer.orig")
	defer b.put("transcriptpeer.orig", tp)
	landed := false
	b.rewrite("transcriptpeer.orig", tp, func(l string) (string, bool) {
		if l == line && !landed {
			landed = true
			return "\treturn strings.Contains(string(doc), peerHex), nil\n" + l, true
		}
		return l, true
	})
	if !landed {
		b.h.bad("adapters the old substring test could not be planted: transcriptpeer.go no longer has [" + strings.TrimSpace(line) + "]")
		return
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		b.h.bad("adapters the planted peer-ID guard cannot be built: go is not on PATH")
		return
	}
	planted := b.w("planted-bit-check")
	if runtime.GOOS == "windows" {
		planted += ".exe"
	}
	build := exec.Command(goTool, "build", "-o", planted, ".")
	build.Dir = b.p("tools/check")
	if out, err := build.CombinedOutput(); err != nil {
		b.h.bad("adapters the planted peer-ID guard did not build: " + strings.TrimSpace(string(out)))
		return
	}
	cmd := exec.Command(planted, "capture-client", "--selftest")
	cmd.Dir = b.tree
	out, err := cmd.CombinedOutput()
	rc := exitOf(err)
	const named = "FAILED a percent-encoded raw-byte peer ID, aria2's, is found"
	if rc == 1 && strings.Contains(string(out), named) {
		b.h.ok("adapters the old substring test, planted, is refused by the selftest on aria2's peer ID: go (exit 1)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the old substring test, planted, was not refused on aria2's peer ID: exit %d: %s",
			rc, strings.TrimSpace(string(out))))
	}
}

// plantedSelfCorroboration removes capture-client's self-corroboration guard
// in a scratch copy - connectorIsTarget answers "" before it tests anything -
// builds that copy and asks its selftest, which must refuse on aria2's case.
// `CLIENT-05`, `E-COR-20`.
func (b *bitcheck) plantedSelfCorroboration() {
	const sc = "tools/check/selfcorroboration.go"
	const line = "\tfor _, key := range []string{\"id\", \"runtime\"} {"
	b.save(sc, "selfcorroboration.orig")
	defer b.put("selfcorroboration.orig", sc)
	landed := false
	b.rewrite("selfcorroboration.orig", sc, func(l string) (string, bool) {
		if l == line && !landed {
			landed = true
			return "\tif target != \"\" {\n\t\treturn \"\"\n\t}\n" + l, true
		}
		return l, true
	})
	if !landed {
		b.h.bad("adapters the self-corroboration guard could not be removed: selfcorroboration.go no longer has [" + strings.TrimSpace(line) + "]")
		return
	}
	b.plantedSelftest("the self-corroboration guard, removed,", "FAILED aria2 is refused as the connector of aria2")
}

// plantedEnhancedSwitch writes one of Enhanced's own switches ON into the
// profile both halves of `qbittorrent-enhanced` write, in a scratch copy,
// builds that copy and asks its selftest, which must refuse the profile it now
// makes. `CLIENT-02`.
//
// ⛔ `Session\ShadowBan` is the plant because it is the fork's and not the
// parent's: a profile check that knew only the three contract surfaces would
// pass it.
func (b *bitcheck) plantedEnhancedSwitch() {
	const qe = "tools/check/qbtenhanced.go"
	const line = "\tline(`Session\\ShadowBan=false`)"
	b.save(qe, "qbtenhanced.orig")
	defer b.put("qbtenhanced.orig", qe)
	landed := false
	b.rewrite("qbtenhanced.orig", qe, func(l string) (string, bool) {
		if l == line && !landed {
			landed = true
			return "\tline(`Session\\ShadowBan=true`)", true
		}
		return l, true
	})
	if !landed {
		b.h.bad("adapters Enhanced's shadow-ban switch could not be planted: qbtenhanced.go no longer has [" + strings.TrimSpace(line) + "]")
		return
	}
	b.plantedSelftestOf("Enhanced's shadow-ban switch, planted on,", []string{"qbittorrent-enhanced", "--selftest"},
		"FAILED the Linux profile is held")
}

// plantedRqbitSwitch takes `--disable-lsd` off the command line both halves
// of `rqbit` build, in a scratch copy, builds that copy and asks its selftest,
// which must refuse the line it now makes. `ENGINE-03`.
//
// ⛔ Local peer discovery is the plant because it is the build's own default
// ON and the one surface a contained host can still reach: a multicast is
// link-local, so a missing switch fails nothing else first.
func (b *bitcheck) plantedRqbitSwitch() {
	const rq = "tools/check/rqbit.go"
	const line = "\t\t\"--disable-dht\", \"--disable-dht-persistence\", \"--disable-upnp-port-forward\", \"--disable-lsd\","
	b.save(rq, "rqbit.orig")
	defer b.put("rqbit.orig", rq)
	landed := false
	b.rewrite("rqbit.orig", rq, func(l string) (string, bool) {
		if l == line && !landed {
			landed = true
			return "\t\t\"--disable-dht\", \"--disable-dht-persistence\", \"--disable-upnp-port-forward\",", true
		}
		return l, true
	})
	if !landed {
		b.h.bad("adapters rqbit's LSD switch could not be planted: rqbit.go no longer has [" + strings.TrimSpace(line) + "]")
		return
	}
	b.plantedSelftestOf("rqbit's LSD switch, removed,", []string{"rqbit", "--selftest"},
		"FAILED the command line is held")
}

// rqbitToolchainPin holds the Rust `rqbit`'s package route compiles with to
// this repository's own pin. ⛔ The route runs `cargo install` outside this
// tree, where `rust-toolchain.toml` does not reach, so the adapter names the
// channel; this is what stops the two drifting apart. `ENGINE-03`.
func (b *bitcheck) rqbitToolchainPin() {
	body, err := os.ReadFile(b.p("rust-toolchain.toml"))
	if err != nil {
		b.h.bad("adapters rqbit's toolchain pin: rust-toolchain.toml could not be read")
		return
	}
	channel := ""
	for _, l := range strings.Split(string(body), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "channel = "); ok {
			channel = strings.Trim(v, `"`)
		}
	}
	if channel != "" && channel == rqbitToolchain {
		b.h.ok("adapters rqbit's package route compiles with this tree's pinned Rust " + channel)
	} else {
		b.h.bad(fmt.Sprintf("adapters rqbit's package route names Rust %s and rust-toolchain.toml pins [%s]", rqbitToolchain, channel))
	}
}

// plantedAnacrolixSwitch takes `-dht=false` off the command line both halves
// of `anacrolix-torrent` build, in a scratch copy, builds that copy and asks
// its selftest, which must refuse the line it now makes. `ENGINE-02`.
//
// ⛔ The DHT is the plant because the library's default config has it on and
// bootstraps from public nodes, and because the harness's own report is a
// second check of it that a line missing the switch would reach only at a
// capture.
func (b *bitcheck) plantedAnacrolixSwitch() {
	const an = "tools/check/anacrolix.go"
	const line = "\t\t\"-dht=false\", \"-pex=false\", \"-port-forward=false\", \"-webseeds=false\", \"-webtorrent=false\", \"-ipv6=false\","
	b.save(an, "anacrolix.orig")
	defer b.put("anacrolix.orig", an)
	landed := false
	b.rewrite("anacrolix.orig", an, func(l string) (string, bool) {
		if l == line && !landed {
			landed = true
			return "\t\t\"-pex=false\", \"-port-forward=false\", \"-webseeds=false\", \"-webtorrent=false\", \"-ipv6=false\",", true
		}
		return l, true
	})
	if !landed {
		b.h.bad("adapters anacrolix-torrent's DHT switch could not be planted: anacrolix.go no longer has [" + strings.TrimSpace(line) + "]")
		return
	}
	b.plantedSelftestOf("anacrolix-torrent's DHT switch, removed,", []string{anacrolixTarget, "--selftest"},
		"FAILED the command line is held")
}

// anacrolixHarnessPin holds the harness this tree commits to the adapter's
// pin: the library's version and sum.golang.org's hashes in its go.mod and
// go.sum, and the toolchain. ⛔ The routes build the harness's files and the
// adapter's constants are what they are held to at install; this row is the
// one that sees the two drift apart before a dispatch does. `ENGINE-02`.
func (b *bitcheck) anacrolixHarnessPin() {
	gomod, e1 := os.ReadFile(b.p(anacrolixHarnessDir + "/go.mod"))
	gosum, e2 := os.ReadFile(b.p(anacrolixHarnessDir + "/go.sum"))
	if e1 != nil || e2 != nil {
		b.h.bad("adapters anacrolix-torrent's harness: " + anacrolixHarnessDir + " has no go.mod or go.sum")
		return
	}
	if msg := anacrolixPinRefusal(string(gomod), string(gosum)); msg != "" {
		b.h.bad("adapters anacrolix-torrent's harness is not pinned: " + msg)
		return
	}
	b.h.ok("adapters anacrolix-torrent's harness pins " + anacrolixModule + " " + anacrolixPin + " and " + anacrolixToolchain)
	// ⛔ AND A PLANTED DRIFT IS SEEN: the go.mod moved to another version.
	planted := strings.Replace(string(gomod), "torrent "+anacrolixPin, "torrent v1.60.0", 1)
	if anacrolixPinRefusal(planted, string(gosum)) != "" {
		b.h.ok("adapters a harness re-pinned without the adapter is refused")
	} else {
		b.h.bad("adapters a harness re-pinned without the adapter was not refused")
	}
}

// plantedSelftest builds the scratch tree's planted copy and asks its
// capture-client selftest, which must exit 1 naming the case `named` ends.
func (b *bitcheck) plantedSelftest(what, named string) {
	b.plantedSelftestOf(what, []string{"capture-client", "--selftest"}, named)
}

// plantedSelftestOf is plantedSelftest over any selftest command.
func (b *bitcheck) plantedSelftestOf(what string, selftest []string, named string) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		b.h.bad("adapters " + what + " cannot be built: go is not on PATH")
		return
	}
	planted := b.w("planted-bit-check")
	if runtime.GOOS == "windows" {
		planted += ".exe"
	}
	build := exec.Command(goTool, "build", "-o", planted, ".")
	build.Dir = b.p("tools/check")
	if out, err := build.CombinedOutput(); err != nil {
		b.h.bad("adapters " + what + " did not build: " + strings.TrimSpace(string(out)))
		return
	}
	cmd := exec.Command(planted, selftest...)
	cmd.Dir = b.tree
	out, err := cmd.CombinedOutput()
	rc := exitOf(err)
	if rc == 1 && strings.Contains(string(out), named) {
		b.h.ok("adapters " + what + " is refused by the selftest: go (exit 1)")
	} else {
		b.h.bad(fmt.Sprintf("adapters %s was not refused: exit %d: %s", what, rc, strings.TrimSpace(string(out))))
	}
}

// plantedBiglybtDHT writes BiglyBT's DHT row ON into the settings both halves
// of `biglybt` write, in a scratch copy, builds that copy and asks its
// selftest, which must refuse the document it now makes. `CLIENT-09`.
//
// ⛔ THE ROW, NOT THE CHECK: bbSurfaces is a second list read off the document
// itself, so a row changed in bbSettings is what the refusal exists to catch.
func (b *bitcheck) plantedBiglybtDHT() {
	const bb = "tools/check/biglybt.go"
	const line = "\t{\"Plugin.DHT.dht.enabled\", 0},"
	b.save(bb, "biglybt.orig")
	defer b.put("biglybt.orig", bb)
	landed := false
	b.rewrite("biglybt.orig", bb, func(l string) (string, bool) {
		if l == line && !landed {
			landed = true
			return "\t{\"Plugin.DHT.dht.enabled\", 1},", true
		}
		return l, true
	})
	if !landed {
		b.h.bad("adapters BiglyBT's DHT row could not be planted: biglybt.go no longer has [" + strings.TrimSpace(line) + "]")
		return
	}
	b.plantedSelftestOf("BiglyBT's DHT row, planted on,", []string{"biglybt", "--selftest"},
		"FAILED the Linux settings are held")
}

func (b *bitcheck) adapters() {
	b.agree("adapters clean tree", "check-adapters", 0)

	// ⭐ capture-client's PEER-ID GUARD, ASKED ON THIS HOST: the transcript is
	// read as JSON, each announce's `peer_id` percent-decoded and compared as
	// bytes. transcriptpeer.go, `CLIENT-05`.
	if rc, out := b.run("capture-client", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters capture-client's peer-ID selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters capture-client's peer-ID selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	b.plantedPeerIDSelftest()
	b.plantedSelfCorroboration()

	// ⭐ THE WINDOWS ADAPTER'S DECISIONS, ASKED ON THIS HOST. The adapter builds
	// for Windows alone; what it decides without Windows in it - the profile, the
	// answer, the payload, the scrub - is qbtwindows.go, and its selftest is the
	// one place a Linux gate sees it. `CI-11`.
	if rc, out := b.run("qbittorrent-windows", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the Windows qbittorrent adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the Windows qbittorrent adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	// ⭐ AND TRANSMISSION'S WINDOWS HALF, for the same reason: the settings
	// document, the two answers, the image's layout, msiexec's command line, the
	// service state and the verifier scrub are trwindows.go. `CI-11`,
	// `CLIENT-06`.
	if rc, out := b.run("transmission-windows", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the Windows transmission adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the Windows transmission adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	// ⭐ AND aria2-next's: the command line, the version reading, Scoop's
	// manifest and where Scoop placed the build are a2nwindows.go, and so is the
	// option read-back both halves of that adapter ask. `CI-11`, `CLIENT-14`.
	if rc, out := b.run("aria2-next-windows", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the Windows aria2-next adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the Windows aria2-next adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	// ⭐ AND deluge's, ONE SELFTEST FOR BOTH HALVES: the settings, the command
	// lines, the version reading, pip's and winget's statements, the pins and
	// the credential scrub are deluge.go, untagged. `CLIENT-07`.
	if rc, out := b.run("deluge", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the deluge adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the deluge adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	// ⭐ AND qbittorrent-enhanced's, ONE SELFTEST FOR BOTH HALVES: the scheme, the
	// profile and its refusal, the version reading, the zip, Scoop's manifest
	// and Scoop's tree are qbtenhanced.go, untagged. `CLIENT-02`.
	if rc, out := b.run("qbittorrent-enhanced", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the qbittorrent-enhanced adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the qbittorrent-enhanced adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	b.plantedEnhancedSwitch()
	// ⭐ AND rqbit's, ONE SELFTEST FOR BOTH HALVES: the scheme, the command
	// line and its refusal, the environment, the answers, cargo's install
	// record and the feature probe are rqbit.go, untagged. `ENGINE-03`.
	if rc, out := b.run("rqbit", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the rqbit adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the rqbit adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	b.plantedRqbitSwitch()
	b.rqbitToolchainPin()
	// ⭐ AND biglybt's, ONE SELFTEST FOR BOTH HALVES: the scheme, the settings and
	// their refusal, the probe class, the answer, the launch, the source build's
	// file split, the tarball's members and the scrub are biglybt.go, untagged.
	// `CLIENT-09`.
	if rc, out := b.run("biglybt", "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the biglybt adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the biglybt adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	b.plantedBiglybtDHT()
	// ⭐ AND anacrolix-torrent's, ONE SELFTEST FOR BOTH HALVES: the pin, the go
	// environment of each route, the build manifest, the proxy's answer, the
	// library's origin, the harness's command line and its report are
	// anacrolix.go, untagged. `ENGINE-02`.
	if rc, out := b.run(anacrolixTarget, "--selftest"); rc == 0 && strings.Contains(out, `"failed":0}`) {
		b.h.ok("adapters the anacrolix-torrent adapter's selftest holds: go (exit 0)")
	} else {
		b.h.bad(fmt.Sprintf("adapters the anacrolix-torrent adapter's selftest: go answered %d: %s", rc, strings.TrimSpace(out)))
	}
	b.plantedAnacrolixSwitch()
	b.anacrolixHarnessPin()

	// ⛔ A RELEASE FETCH WITH NO TIME LIMIT. ⚠ Planted in the installer, because
	// `ACQ-06` moved every release retrieval there, and in its Go file since
	// `CI-11` moved the installer into this binary: the fetch is an
	// `exec.Command(curl, ...)` the rule reads whole. ⛔ The flag is dropped
	// rather than renamed: the rule asks whether the call CONTAINS
	// `"--max-time"`, so a renamed flag carrying the words would satisfy it.
	const rl = "tools/check/installrootless.go"
	b.save(rl, "rootless.orig")
	b.rewrite("rootless.orig", rl, func(l string) (string, bool) {
		return strings.Replace(l, `"--max-time", `, "", 1), true
	})
	b.agree("adapters a curl that writes a file with no --max-time is refused", "check-adapters", 1)
	b.put("rootless.orig", rl)

	// ⛔ AND A CLONE THAT NOTHING BOUNDS: `git` has no flag of its own, so the rule
	// is a `timeout` wrapper. ⚠ Planted in the Go adapter since `CI-11` moved the
	// adapters into this binary: the clone is an `exec.Command("timeout", "600",
	// "git", ...)` and the plant takes the wrapper off it.
	const ad = "tools/check/adapter_aria2-next.go"
	b.save(ad, "adapter.orig")
	b.rewrite("adapter.orig", ad, func(l string) (string, bool) {
		return strings.Replace(l, `exec.Command("timeout", "600", "git", "clone"`, `exec.Command("git", "clone"`, 1), true
	})
	b.agree("adapters a git clone with no timeout wrapper is refused", "check-adapters", 1)
	b.put("adapter.orig", ad)

	// ⚠ AND A BOUND THAT BOUNDS SOMETHING ELSE IS NOT A BOUND: the words moved
	// to the end of git's own arguments, where they are something git is handed.
	b.rewrite("adapter.orig", ad, func(l string) (string, bool) {
		if m := timeoutAfter.FindStringSubmatchIndex(l); m != nil {
			return l[:m[0]] + `exec.Command("git", "clone", ` + l[m[2]:m[3]] + `, "timeout", "600",`, true
		}
		return l, true
	})
	b.agree("adapters a timeout after the command it should wrap is refused", "check-adapters", 1)
	b.put("adapter.orig", ad)

	// ⛔ AND THE RPC CALL, WHICH WRITES ITS ANSWER TO A FILE: a curl with `-o`
	// is a retrieval whatever it retrieves.
	b.rewrite("adapter.orig", ad, func(l string) (string, bool) {
		return strings.Replace(l, `"--max-time", "20",`, "", 1), true
	})
	b.agree("adapters a Go adapter's curl with no --max-time is refused", "check-adapters", 1)
	b.put("adapter.orig", ad)

	// ⭐ AND A CURL BOUNDED BY `timeout` ALONE IS ACCEPTED, which is qBittorrent's
	// Qt fetch; taking the wrapper off is what refuses it.
	const qb = "tools/check/adapter_qbittorrent.go"
	b.save(qb, "qbittorrent.orig")
	b.rewrite("qbittorrent.orig", qb, func(l string) (string, bool) {
		return strings.Replace(l, `exec.Command("timeout", "120", "curl"`, `exec.Command("curl"`, 1), true
	})
	b.agree("adapters a curl that loses its timeout wrapper is refused", "check-adapters", 1)
	b.put("qbittorrent.orig", qb)

	// ⛔ AND A WINDOWS CLONE IS READ AS ONE: deluge's Windows half bounds its
	// clone with Git for Windows' coreutils `timeout`, named as the program, and
	// taking that name off is refused like any other. `CLIENT-07`.
	const dw = "tools/check/adapter_deluge_windows.go"
	b.save(dw, "deluge.orig")
	b.rewrite("deluge.orig", dw, func(l string) (string, bool) {
		return strings.Replace(l, `exec.Command("timeout", "600", "git", "clone"`, `exec.Command("git", "clone"`, 1), true
	})
	b.agree("adapters a Windows clone with no timeout wrapper is refused", "check-adapters", 1)
	b.put("deluge.orig", dw)

	// ⭐ AND THE ACCEPTING HALF THAT MATTERS MOST: a curl that only ASKS is out of
	// scope.
	b.agree("adapters clean tree again", "check-adapters", 0)
}

// ============================================================================
// check-docs
// ============================================================================
//
// ⛔ EVERY PLANT THAT MUST BE ACCEPTED SITS AT THE REPOSITORY ROOT, and that is
// forced rather than chosen: a new `.md` in a subdirectory is an ORPHAN unless
// something links to it, so a case expecting 0 would fail for a reason it was not
// written to ask about.

func (b *bitcheck) docs() {
	const dc, root = "check-docs", "plant.md"
	b.agree("docs clean tree", dc, 0)

	b.write(root, "see [the thing](nope-does-not-exist.md) here\n")
	b.agree("docs a broken relative link is refused", dc, 1)
	b.unplant(root)

	// ⭐ THE TWO READERS ARE NOT THE SAME READER. The broken-link pass strips inline
	// code spans, because markdown does not linkify one.
	b.write(root, "see `[the thing](nope-does-not-exist.md)` here\n")
	b.agree("docs a broken link inside a code span is accepted", dc, 0)
	b.unplant(root)

	b.write(root, "text\n\n```\n[the thing](nope-does-not-exist.md)\n```\n")
	b.agree("docs a broken link inside a fenced block is accepted", dc, 0)
	b.unplant(root)

	// ⚠ AN ABSOLUTE URL IS NOT A PATH: a check that needs the network goes red
	// when somebody else's server does.
	b.write(root, "see [the thing](https://example.invalid/nope) here\n")
	b.agree("docs an unresolvable http link is out of scope", dc, 0)
	b.unplant(root)

	b.write(root, "text\n\n```sh\nif [ 1 = 1 ; then echo broken\n```\n")
	b.agree("docs a shell block that does not parse is refused", dc, 1)
	b.unplant(root)

	// ⛔ THE PLACEHOLDER A HUMAN READS AS *fill this in* AND bash READS AS A
	// REDIRECT. The block parses; that is the point. ⛔ The needle is assembled,
	// and the word inside the brackets is one no other rule looks for.
	angle := "<" + "endpoint" + ">"
	b.write(root, "text\n\n```sh\ncurl -sS "+angle+"\n```\n")
	b.agree("docs an angle-bracket placeholder in a shell block is refused", dc, 1)
	b.unplant(root)

	// ⭐ AND A BLOCK THAT IS SIMPLY CORRECT, because a rule that refused every
	// fenced block would pass both cases above.
	b.write(root, "text\n\n```sh\nset -eu\nprintf \"%s\\n\" \"ok\"\n```\n")
	b.agree("docs a shell block that parses is accepted", dc, 0)
	b.unplant(root)

	// -- the orphan rule ----------------------------------------------------------

	b.mkdir("docs")
	b.write("docs/plant-orphan.md", "a page nothing points at\n")
	b.agree("docs a page nothing links to is refused", dc, 1)

	// ⛔ AND THE SAME PAGE, CITED ONLY INSIDE BACKTICKS, IS STILL AN ORPHAN. On the
	// shell half's first --compare run its two halves ANSWERED DIFFERENTLY here;
	// both were made to agree with the twin, which is correct on the rule's own
	// reasoning, so this asserts a REFUSAL rather than freezing the defect.
	b.write(root, "cited as `[the page](docs/plant-orphan.md)` above\n")
	b.agree("docs a page cited only inside backticks is still an orphan", dc, 1)
	b.unplant(root)
	b.unplant("docs/plant-orphan.md")

	// ⚠ A LINK THAT CLIMBS MORE THAN TWO LEVELS, and this case is a GUARD rather
	// than a proof. ⛔ Measured by planting: with the global `segment/../` collapse
	// restored, the case still PASSED, because `path.Join` normalises first. The
	// label says what it guards rather than claiming more.
	//
	// ⛔ AND THIS CASE OVERWRITES A REAL TRACKED FILE, so it is SAVED AND PUT BACK
	// rather than unplanted: deleting it left a path git still lists with no bytes
	// behind it, which `check-project` later read as a line ending.
	const fixReadme = "crates/bit-ids/tests/fixtures/README.md"
	b.save(fixReadme, "fixtures-README.orig")
	b.write("docs/plant-orphan.md", "a page nothing else points at\n")
	b.write(fixReadme, "see [the page](../../../../docs/plant-orphan.md)\n")
	b.agree("docs a link climbing four levels resolves to the same page", dc, 0)
	b.put("fixtures-README.orig", fixReadme)
	b.unplant("docs/plant-orphan.md")

	// ⚠ A README IS AN ENTRY POINT and the rule skips it by name.
	b.mkdir("docs/plantdir")
	b.write("docs/plantdir/README.md", "an entry point\n")
	b.agree("docs an unlinked subdirectory README is accepted", dc, 0)
	b.rmrf("docs/plantdir")

	// -- the history index --------------------------------------------------------
	//
	// ⛔ A RECORD LINKED FROM ANYWHERE IS NOT AN ORPHAN AND IS STILL MISSING FROM
	// THE INDEX. ⚠ The plant is LINKED on purpose, so this case can only fail for
	// the index reason.
	b.write("docs/history/SESSION-PLANT.md", "an old record\n")
	b.write(root, "see [the record](docs/history/SESSION-PLANT.md)\n")
	b.agree("docs a session record the history index omits is refused", dc, 1)
	b.unplant("docs/history/SESSION-PLANT.md")
	b.unplant(root)

	// -- the template exemption ---------------------------------------------------
	//
	// ⛔ docs/templates/ IS EXEMPT FROM LINK RESOLUTION AND THIS TREE HAS NO SUCH
	// DIRECTORY, so the branch is proved with a fixture. ⚠ It is linked from the
	// root file, because the orphan rule has NO template exemption.
	b.mkdir("docs/templates")
	b.write("docs/templates/plant.md", "see [gate](methodology/gate.md), which is not here\n")
	b.write(root, "see [the template](docs/templates/plant.md)\n")
	b.agree("docs a template's unresolvable link is exempt", dc, 0)
	b.rmrf("docs/templates")
	b.unplant(root)

	b.agree("docs clean tree again", dc, 0)
}

// ============================================================================
// check-project
// ============================================================================
//
// ⛔ THIRTY REFUSALS IN ONE CHECK, so one case per rule is the floor rather than
// thoroughness. Several plants are not narrowed - an invalid status necessarily
// moves four derived totals - because the case states the exit code and nothing
// finer.

var projFiles = []string{"TODO/INDEX.md", "TODO/PROGRESS.md", "TODO/SUMMARY.md", "docs/client-matrix.md",
	"catalogue/clients.toml", "Cargo.lock", "scripts/doctor/provision.sh"}

func (b *bitcheck) restoreProject() {
	for _, f := range projFiles {
		b.put(filepath.Base(f)+".projorig", f)
	}
}

// awkFields is `awk -F'|' 'BEGIN { OFS = "|" } $2 == KEY { $4 = VAL }'`.
func awkFields(key, val string) func(string) (string, bool) {
	return func(l string) (string, bool) {
		f := strings.Split(l, "|")
		if len(f) > 1 && f[1] == key {
			for len(f) < 4 {
				f = append(f, "")
			}
			f[3] = val
			return strings.Join(f, "|"), true
		}
		return l, true
	}
}

// subFirst is awk's `sub(RE, REPL)`: the first match on the line, replaced.
func subFirst(re *regexp.Regexp, l, repl string) string {
	if m := re.FindStringIndex(l); m != nil {
		return l[:m[0]] + repl + l[m[1]:]
	}
	return l
}

var (
	effortXL   = regexp.MustCompile(`, [0-9]+ XL\.`)
	priorityN  = regexp.MustCompile(`\| [0-9]+ \|`)
	priorityP1 = regexp.MustCompile(`^\| P1 \|`)
)

func (b *bitcheck) project() {
	const pc = "check-project"
	b.agree("project clean tree", pc, 0)

	for _, f := range projFiles {
		b.save(f, filepath.Base(f)+".projorig")
	}
	// ⚠ Kept apart from the list above because two cases edit it and then put it
	// back on their own, rather than through the blanket restore.
	b.save("TODO/ci.md", "ci.md.projsrc")

	// -- 1. the files a reader is promised ----------------------------------------
	b.must(os.Rename(b.p("docs/reference-sweeps/bit-cli.md"), b.w("bit-cli.projmoved")))
	b.agree("project a promised file that is missing is refused", pc, 1)
	b.must(os.Rename(b.w("bit-cli.projmoved"), b.p("docs/reference-sweeps/bit-cli.md")))

	// -- 2. the target set, in both directions ------------------------------------
	b.appendTo("docs/client-matrix.md", "| `plant-client` | x | x | x |\n")
	b.agree("project a matrix row the catalogue does not carry is refused", pc, 1)
	b.restoreProject()

	b.appendTo("catalogue/clients.toml", "\n[[client]]\nid = \"plant-client\"\n")
	b.agree("project a catalogue target the matrix lacks is refused", pc, 1)
	b.restoreProject()

	// ⛔ TWO EMPTY SETS AGREE PERFECTLY. Without this floor a parser that stopped
	// matching would report a pinned matrix over nothing at all.
	b.write("docs/client-matrix.md", "# nothing here\n")
	b.agree("project a target set too small to be real is refused", pc, 1)
	b.restoreProject()

	// -- 3-8. the TODO bookkeeping ------------------------------------------------
	b.rewrite("PROGRESS.md.projorig", "TODO/PROGRESS.md", func(l string) (string, bool) {
		if rest, ok := strings.CutPrefix(l, "Total: "); ok {
			return "Total: 9" + rest, true
		}
		return l, true
	})
	b.agree("project a declared total that disagrees is refused", pc, 1)
	b.restoreProject()

	b.rewrite("INDEX.md.projorig", "TODO/INDEX.md", func(l string) (string, bool) {
		return strings.Replace(l, "| OPEN |", "| WIP |", 1), true
	})
	b.agree("project an invalid status in the index is refused", pc, 1)
	b.restoreProject()

	// ⚠ THE OPEN COUNT IN THE TOTAL ROW, by field rather than by a number typed
	// here, so the plant follows the file.
	b.rewrite("SUMMARY.md.projorig", "TODO/SUMMARY.md", awkFields(" Total ", " 99 "))
	b.agree("project a summary total that disagrees is refused", pc, 1)
	b.restoreProject()

	// ⛔ THE CATEGORY ROWS ARE ELEVEN OF TWELVE, and the total row agreeing says
	// nothing about them.
	b.rewrite("SUMMARY.md.projorig", "TODO/SUMMARY.md", awkFields(" Foundation ", " 99 "))
	b.agree("project a summary category row that disagrees is refused", pc, 1)
	b.restoreProject()

	// ⚠ THE EFFORT SENTENCE IS PROSE AND IT DRIFTED AS PROSE. The plant rewrites
	// the XL count whatever it is.
	b.rewrite("SUMMARY.md.projorig", "TODO/SUMMARY.md", func(l string) (string, bool) {
		if strings.HasPrefix(l, "Effort inventory: ") {
			return subFirst(effortXL, l, ", 7 XL."), true
		}
		return l, true
	})
	b.agree("project a summary effort inventory that disagrees is refused", pc, 1)
	b.restoreProject()

	done := false
	b.rewrite("INDEX.md.projorig", "TODO/INDEX.md", func(l string) (string, bool) {
		if !done && priorityP1.MatchString(l) {
			done = true
			return subFirst(priorityN, l, "| 98 |"), true
		}
		return l, true
	})
	b.agree("project a priority table row that disagrees is refused", pc, 1)
	b.restoreProject()

	// ⚠ A DUPLICATED ROW IS TWO FINDINGS, not one: the duplicate itself and every
	// total it moves.
	done = false
	b.rewrite("INDEX.md.projorig", "TODO/INDEX.md", func(l string) (string, bool) {
		if !done && strings.HasPrefix(l, "| CI-0") {
			done = true
			return l + "\n" + l, true
		}
		return l, true
	})
	b.agree("project a duplicated index row is refused", pc, 1)
	b.restoreProject()

	// -- 9. Python exists only with an approved exception -------------------------
	const py = "tools/check/plant.py"
	b.write(py, "print(\"plant\")\n")
	b.agree("project a .py declaring no exception is refused", pc, 1)

	b.write(py, "# bit-ids:python-exception=ZZZ-99\nprint(\"plant\")\n")
	b.agree("project a .py naming an entry the index lacks is refused", pc, 1)

	// ⛔ THE OWNING ENTRY'S OWN SECTION, NOT ITS FILE. A plant naming a real but
	// unrelated entry survived this rule twice before it was narrowed this far.
	b.write(py, "# bit-ids:python-exception=CI-10\nprint(\"plant\")\n")
	b.agree("project a .py whose entry never mentions it is refused", pc, 1)

	// ⭐ AND THE ACCEPT CASE: the argument is written into the entry's section.
	b.rewrite("ci.md.projsrc", "TODO/ci.md", func(l string) (string, bool) {
		if strings.HasPrefix(l, "## CI-10:") {
			return l + "\n" + "tools/check/plant.py is approved here.", true
		}
		return l, true
	})
	b.agree("project a .py its entry argues for is accepted", pc, 0)
	b.unplant(py)
	b.put("ci.md.projsrc", "TODO/ci.md")

	// -- 10. every action is pinned to an immutable form --------------------------
	const pwf = ".github/workflows/plant.yml"
	const job = "on: push\njobs:\n  j:\n    steps:\n"

	b.write(pwf, job+"      - uses: actions/checkout@v4\n")
	b.agree("project an action pinned to a tag is refused", pc, 1)

	b.write(pwf, job+"      - uses: actions/checkout@"+hex40+"\n")
	b.agree("project a pin with no version comment is refused", pc, 1)

	b.write(pwf, job+"      - uses: actions/checkout@"+hex40+" # v4.2.2\n")
	b.agree("project a pin with a version comment is accepted", pc, 0)

	b.write(pwf, job+"      - uses: actions/checkout\n")
	b.agree("project a uses: with no ref at all is refused", pc, 1)

	b.write(pwf, job+"      - uses: docker://alpine:3\n")
	b.agree("project a container not pinned to a digest is refused", pc, 1)

	// ⭐ A LOCAL ACTION IS THIS REPOSITORY, reviewed with everything else.
	b.write(pwf, job+"      - uses: ./.github/actions/plant\n")
	b.agree("project a local action needs no pin", pc, 0)

	// -- 11. an artifact a workflow downloads is one some workflow uploads --------
	up := "      - uses: actions/upload-artifact@" + hex40 + " # v7.0.0\n"
	down := "      - uses: actions/download-artifact@" + hex40 + " # v8.0.0\n"

	b.write(pwf, job+down+"        with:\n          name: plant-nobody-uploads\n")
	b.agree("project a download nothing uploads is refused", pc, 1)

	b.write(pwf, job+up+"        with:\n          name: plant-pair\n"+down+"        with:\n          name: plant-pair\n")
	b.agree("project a download its own workflow uploads is accepted", pc, 0)

	// ⛔ A MARKER THAT OUTLIVED ITS REASON IS REFUSED TOO. ⚠ It sits INSIDE the step
	// it declares, which is where the tree spells it.
	b.write(pwf, job+up+"        with:\n          name: plant-pair\n"+down+
		"        # bit-ids:no-producer=CI-09\n"+"        with:\n          name: plant-pair\n")
	b.agree("project a no-producer marker over a produced name is refused", pc, 1)

	// ⚠ `pattern:` COUNTS TOO: a rule reading only `name:` is a gate on one of two
	// doors.
	b.write(pwf, job+down+"        with:\n          pattern: plant-nobody-uploads-*\n")
	b.agree("project a download written as pattern: is read too", pc, 1)

	// -- 22-24. the same language behind a second door ----------------------------
	const pwsh = "      - name: plant\n        shell: pwsh\n        run: |\n"

	b.write(pwf, job+pwsh+"          $ErrorActionPreference = 'Stop'\n"+"          Write-Output ok\n")
	b.agree("project a workflow pwsh block that stops without the native rule is refused", pc, 1)

	b.write(pwf, job+pwsh+"          $ErrorActionPreference = 'Stop'\n"+
		"          $PSNativeCommandUseErrorActionPreference = $false\n"+"          Write-Output ok\n")
	b.agree("project a workflow pwsh block that says both is accepted", pc, 0)

	b.write(pwf, job+pwsh+"          Write-Error 'plant'\n")
	b.agree("project a workflow pwsh block reporting through Write-Error is refused", pc, 1)

	// ⛔ AN INVERTED GUARD FAILS THE STEP BY SUCCEEDING AT ITS JOB, because GitHub
	// reads the block's residual exit code as the verdict.
	b.write(pwf, job+pwsh+"          git status\n"+"          if ($LASTEXITCODE -ne 0) { Write-Output no }\n")
	b.agree("project a pwsh block leaving LASTEXITCODE as the verdict is refused", pc, 1)

	b.write(pwf, job+pwsh+"          git status\n"+"          if ($LASTEXITCODE -ne 0) { Write-Output no }\n"+"          exit 0\n")
	b.agree("project a pwsh block that ends in an explicit exit is accepted", pc, 0)

	b.write(pwf, job+pwsh+"          Add-Content -LiteralPath $env:GITHUB_STEP_SUMMARY -Value ok\n")
	b.agree("project a workflow pwsh block writing with the host's encoding is refused", pc, 1)

	b.write(pwf, job+pwsh+"          Add-Content -LiteralPath $env:GITHUB_STEP_SUMMARY -Value ok -Encoding utf8NoBOM\n")
	b.agree("project a workflow pwsh block naming its encoding is accepted", pc, 0)
	b.unplant(pwf)

	// -- 12. the working tree agrees with .gitattributes --------------------------
	//
	// ⚠ A `.bat` RATHER THAN A `.ps1`, so this plant trips the line-ending rule and
	// nothing else.
	b.write("plant.bat", "echo plant\n")
	b.plantTracked("plant.bat")
	b.agree("project a working tree ending that contradicts .gitattributes is refused", pc, 1)
	b.unplantTracked("plant.bat")

	// -- 13-15, 17. the tracked-only scopes ---------------------------------------
	// ⚠ THE NEEDLES ARE ASSEMBLED: a harness that plants a pattern cannot spell it.
	const ps = "scripts/plant.sh"
	b.write(ps, "set -u\n# mvdan.cc/sh/v3/cmd/"+"shfmt"+"@v0.0.1\n")
	b.plantTracked(ps)
	b.agree("project an shfmt version that disagrees with the pin is refused", pc, 1)

	// ⛔ RE-STAGED AFTER EVERY REWRITE. `git rm --cached` refuses a path whose staged
	// content matches neither HEAD nor the working tree, so a plant rewritten three
	// times could not be unplanted and survived into every case after it.
	b.write(ps, "echo plant\n")
	b.plantTracked(ps)
	b.agree("project a script that does not state set -u first is refused", pc, 1)

	b.write(ps, "set -u\nBIN=target/"+"debug"+"/examples/thing\n")
	b.plantTracked(ps)
	b.agree("project a cargo output path composed by hand is refused", pc, 1)

	b.write(ps, "set -u\n# see: cargo test some_name\n")
	b.plantTracked(ps)
	b.agree("project a cargo test in a script is out of this rule's scope", pc, 0)
	b.unplantTracked(ps)

	b.write(ps, "set -u\nps -eo pid,args\n")
	b.plantTracked(ps)
	b.agree("project a ps a narrow console truncates is refused", pc, 1)

	b.write(ps, "set -u\nps -ww -eo pid,args\n")
	b.plantTracked(ps)
	b.agree("project a ps asked for whole lines is accepted", pc, 0)
	b.unplantTracked(ps)

	b.rewrite("provision.sh.projorig", "scripts/doctor/provision.sh", func(l string) (string, bool) {
		if rest, ok := strings.CutPrefix(l, "SHFMT_VERSION="); ok {
			return "SHFMT_PIN_REMOVED=" + rest, true
		}
		return l, true
	})
	b.agree("project a provision script with no shfmt pin is refused", pc, 1)
	b.restoreProject()

	// -- 16. every dependency comes from the registry, with a checksum ------------
	b.rewrite("Cargo.lock.projorig", "Cargo.lock", func(l string) (string, bool) {
		if l == `source = "registry+https://github.com/rust-lang/crates.io-index"` {
			return `source = "git+https://example.invalid/x"`, true
		}
		return l, true
	})
	b.agree("project a dependency from somewhere other than the registry is refused", pc, 1)
	b.restoreProject()

	b.rewrite("Cargo.lock.projorig", "Cargo.lock", func(l string) (string, bool) {
		return l, !strings.HasPrefix(l, `checksum = "`)
	})
	b.agree("project a dependency with no checksum is refused", pc, 1)
	b.restoreProject()

	// -- 17. an acceptance that can pass over nothing -----------------------------
	//
	// ⛔ `cargo test` WITH A BARE WORD SELECTS BY NAME, and a filter matching none
	// prints `running 0 tests` for every binary and exits 0.
	b.appendTo("TODO/ci.md", "\nProve: run `cargo test plant_name` and read it\n")
	b.agree("project a Prove selecting tests by name is refused", pc, 1)
	b.put("ci.md.projsrc", "TODO/ci.md")

	b.appendTo("TODO/ci.md", "\nProve: run `cargo test --workspace --locked` and read it\n")
	b.agree("project a Prove selecting by package rather than name is accepted", pc, 0)
	b.put("ci.md.projsrc", "TODO/ci.md")

	// ⚠ THE WORKFLOW IS THE SECOND DOOR into the same mistake. ⚠ TRACKED, because
	// this rule's scope is `git ls-files` rather than a glob.
	b.write(pwf, job+"      - run: cargo test plant_name\n")
	b.plantTracked(pwf)
	b.agree("project a workflow cargo test selecting by name is refused", pc, 1)
	b.unplantTracked(pwf)

	// -- 18. a git dependency in a manifest ---------------------------------------
	b.mkdir("plantcrate")
	b.write("plantcrate/Cargo.toml", "[dependencies]\nthing = { git = \"https://example.invalid/x\" }\n")
	b.agree("project a git dependency in a manifest is refused", pc, 1)
	b.rmrf("plantcrate")

	// -- 19-21. the three .ps1 rules ----------------------------------------------
	const p1 = "plant.ps1"
	b.write(p1, "Write-Output \"a plant "+stop+" here\"\n")
	b.agree("project a .ps1 with non-ASCII and no BOM is refused", pc, 1)

	b.write(p1, bom+"Write-Output \"a plant "+stop+" here\"\n")
	b.agree("project a .ps1 with non-ASCII and a BOM is accepted", pc, 0)

	b.write(p1, "$ErrorActionPreference = 'Stop'\nWrite-Output ok\n")
	b.agree("project a .ps1 that stops without the native rule is refused", pc, 1)

	b.write(p1, "$ErrorActionPreference = 'Stop'\n$PSNativeCommandUseErrorActionPreference = $false\n")
	b.agree("project a .ps1 that says both is accepted", pc, 0)

	b.write(p1, "Write-Error 'plant'\n")
	b.agree("project a .ps1 reporting through Write-Error is refused", pc, 1)

	// ⚠ THE NEEDLE IS AN INVOCATION, NOT THE WORD: a rule matching the name
	// anywhere fires on the file that describes it.
	b.write(p1, "# Write-Error is what this comment is about\nWrite-Output ok\n")
	b.agree("project a .ps1 that only mentions Write-Error is accepted", pc, 0)

	b.write(p1, "$rows | Set-Content -LiteralPath out.txt\n")
	b.agree("project a .ps1 writing with the host's encoding is refused", pc, 1)

	b.write(p1, "$rows | Set-Content -LiteralPath out.txt -Encoding utf8NoBOM\n")
	b.agree("project a .ps1 naming its encoding is accepted", pc, 0)

	b.write(p1, "# Set-Content is what this comment is about\nWrite-Output ok\n")
	b.agree("project a .ps1 that only mentions Set-Content is accepted", pc, 0)
	b.unplant(p1)

	// ⛔ A NAME GIT WOULD QUOTE is read under its own name, tracked or not:
	// `core.quotepath` turned this one into a path no file has, and the rule
	// skipped it unread.
	const pq = "plant-caf\u00e9.ps1"
	b.write(pq, "Write-Error 'plant'\n")
	b.agree("project an untracked .ps1 git would quote the name of is read", pc, 1)
	b.plantTracked(pq)
	b.agree("project a tracked .ps1 git would quote the name of is read", pc, 1)
	b.unplantTracked(pq)

	b.agree("project clean tree again", pc, 0)
}
