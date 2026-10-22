// dispatch.go - turn one `capture-client` dispatch's downloaded artifacts into
// its records, their run manifests, and the bundle `publish-data.yml`
// downloads. `bit-check assemble-dispatch` and `bit-check assemble-bundle`.
//
// ⭐ PORTED FROM scripts/capture/assemble-dispatch.sh AND
// scripts/publishing/assemble-bundle.sh, `CI-07`, so nothing on the path from a
// dispatch's artifacts to a bundle is shell. Settled by the operator on
// 2026-10-15: `check-assemble`'s dispatch rows could then reach the Windows
// lane only by running `sh` there, and a non-bash step's path may not carry
// one. Each was compared against its shell half, bundle bytes and messages,
// before that half was deleted.
//
// ⛔ THESE WRITE, SO THEY ARE HELPERS AND NOT CHECKS, like `mine-repo`: no gate
// row, and `--rows` does not name them. Each parses its own arguments and
// answers its own exit code: 0 assembled, 1 refused, 2 could not run.
//
// -- assemble-dispatch ----------------------------------------------------------
//
// ⭐ THE LANES ARE READ FROM THE ARTIFACT NAMES, NOT DECLARED. Each lane uploads
// `capture-client-<adapter>-<route>-<run>-<attempt>` and
// `install-<adapter>-<route>-<run>-<attempt>`, and this pairs them. An adapter
// may carry a hyphen, so the name is read from its end: the attempt, the run,
// then the route, which is one of package, release or source.
//
// ⛔ EVERY ADAPTER IS ASSEMBLED OR THE RUN IS REFUSED. An adapter with one
// lane, or a capture with no install artifact beside it, is a dispatch that did
// not produce a pair, and a bundle that quietly left it out would read as
// complete. So is a directory carrying two runs or two attempts.
//
// ⛔ NOTHING HERE DECIDES WHAT A RECORD SAYS. `assemble-capture` writes the
// records and their manifests, and the bundle takes each target's version
// scheme from the resolutions the install artifacts carry. `CI-09`.
//
// -- assemble-bundle ------------------------------------------------------------
//
// ⛔ A CAPTURE STORE IS NOT A PUBLICATION. `assemble-capture` writes records and
// their run manifests; the data branch also carries the indexes, every
// rendering, the licence, and the two documents that describe each published
// byte. This is the one place those are composed, out of the readers that
// already exist for each.
//
// ⛔ THE STORE MUST BE A CORPUS FIRST. `validate-corpus` reads it before
// anything is copied, and a refusal ends the run with no bundle. A provisional
// record is not a refusal: the indexes are what keep it out of a published
// view. The count of records that may enter one is printed.
//
// ⛔ THE VERSION SCHEME IS NEVER DECLARED HERE. Each route's resolver records
// the scheme it ordered tags under in its `resolution.json`, and this reads it
// from there. Two resolutions that order one target differently are refused,
// because the latest view would depend on which was read first. ⚠ Every
// generation is read: the scheme floor `bit-ids/resolution/2` added releases
// candidates and never changes how two versions order. `CLIENT-06`. ⭐ The
// release series `bit-ids/resolution/3` added DOES order, so it travels in the
// scheme line as a fifth field, `libtorrent:v:3:3:2.0`; a scheme with no series
// keeps the four fields it always had, so every line published before reads
// back byte for byte. `ENGINE-01`.
//
// ⛔ ONLY `profiles/` AND `raw/` ARE COPIED, AND ANYTHING ELSE IS REFUSED. A
// store carrying an index or a manifest of its own was published or assembled
// before, and copying it would describe another run.
//
// ⚠ IT DOES NOT READ ITS OWN OUTPUT BACK, AND THAT IS MEASURED RATHER THAN
// OMITTED. A first version of the shell half did, and a plant disabling either
// reader was refused by no case. `publish-data.sh` refuses a bundle whose
// checksums do not match before anything is pushed, and `check-bundle.sh`
// reads the bundle it was handed as a corpus and with `sha256sum`.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// interrupted runs every cleanup registered so far and exits 2, on SIGINT or
// SIGTERM. ⛔ The shell halves did this with `trap 'exit 2' INT TERM`, and a
// deferred removal never runs on a signal: an operator who interrupted a build
// was left a directory that looks publishable. Found by a review of the port.
var (
	cleanupMu sync.Mutex
	cleanups  []func()
	notifyOne sync.Once
)

func onInterrupt(fn func()) {
	cleanupMu.Lock()
	cleanups = append(cleanups, fn)
	cleanupMu.Unlock()
	notifyOne.Do(func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-ch
			cleanupMu.Lock()
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
			}
			os.Exit(2)
		}()
	})
}

const dispatchUsage = "usage: bit-check assemble-dispatch [--published DIR] ARTIFACTS OUT\n"
const bundleUsage = "usage: bit-check assemble-bundle [--published DIR] --resolution FILE... STORE OUT\n"

// schemesFile is where a bundle publishes the version schemes it ordered under,
// relative to the bundle root.
//
// ⛔ A PUBLISHED RECORD'S SCHEME HAS TO BE PUBLISHED TOO, and was not until
// 2026-10-15. The second publication assembles onto the first, and the first's
// records need the scheme they were ordered under for the latest view to be
// rebuilt; their resolutions were in a capture's artifacts, which expire. So
// the scheme travels with the records, as a derived file the next assembly
// reads back. `CI-09`.
const schemesFile = "indexes/v1/schemes.txt"

// schemeShape is the one shape a scheme line has, `target:prefix:min:max`,
// and `:series` after it where the target pins a release series.
var schemeShape = regexp.MustCompile(`^[^:]+:[^:]+:[0-9]+:[0-9]+(:(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))*)?$`)

// helperRoot is the repository the binary was run in, found without changing
// directory, because every path a caller passes is relative to where it stands.
func helperRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	root := strings.TrimSpace(string(out))
	if err != nil || root == "" {
		return "", errCannotRun("not inside the repository, so there is nothing to build the examples from")
	}
	return root, nil
}

// indent is `sed 's/^/  /'` over a file.
func indent(w io.Writer, path string) {
	for _, l := range fileLines(path) {
		fmt.Fprintf(w, "  %s\n", l)
	}
}

// isDir is `[ -d PATH ]`, which follows a symbolic link.
func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// globDirs is `for d in DIR/PREFIX*; do [ -d "$d" ] || continue`, in the
// byte order a C-locale glob expands to.
func globDirs(dir, prefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && isDir(filepath.Join(dir, e.Name())) {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// cutLast is the pair `${s##*-}` and `${s%-*}`: what follows the last hyphen and
// what precedes it, and the whole string both times when there is none.
func cutLast(s string) (string, string) {
	i := strings.LastIndex(s, "-")
	if i < 0 {
		return s, s
	}
	return s[i+1:], s[:i]
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

type lane struct{ adapter, route, run string }

func assembleDispatch(args []string) int {
	refuse := func(msg string) int {
		fmt.Fprintf(os.Stderr, "assemble-dispatch: %s\n", msg)
		return 1
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(dispatchUsage)
		return 0
	}
	var published []string
	if len(args) >= 1 && args[0] == "--published" {
		if len(args) < 2 {
			fmt.Fprint(os.Stderr, dispatchUsage)
			return 2
		}
		published = []string{"--published", args[1]}
		args = args[2:]
	}
	if len(args) != 2 {
		fmt.Fprint(os.Stderr, dispatchUsage)
		return 2
	}
	artifacts, out := args[0], args[1]
	if !isDir(artifacts) {
		fmt.Fprintf(os.Stderr, "assemble-dispatch: %s is not a directory\n", artifacts)
		return 2
	}
	if _, err := os.Lstat(out); err == nil {
		fmt.Fprintf(os.Stderr, "assemble-dispatch: %s already exists\n", out)
		return 2
	}
	root, err := helperRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "assemble-dispatch: %v\n", err)
		return 2
	}
	assemble, err := exampleBinary(root, "assemble-capture", "bit-ids-probe")
	if err != nil {
		fmt.Fprintf(os.Stderr, "assemble-dispatch: %v\n", err)
		return 2
	}
	work, err := scratch("assembledispatch")
	if err != nil {
		fmt.Fprintf(os.Stderr, "assemble-dispatch: %v\n", err)
		return 2
	}
	defer os.RemoveAll(work)
	onInterrupt(func() { os.RemoveAll(work) })

	// -- the lanes, from the names ------------------------------------------------
	var lanes []lane
	for _, name := range globDirs(artifacts, "capture-client-") {
		attempt, rest := cutLast(strings.TrimPrefix(name, "capture-client-"))
		run, rest := cutLast(rest)
		route, adapter := cutLast(rest)
		if !allDigits(attempt + run) {
			return refuse(name + " does not end in a run id and an attempt")
		}
		switch route {
		case "package", "release", "source":
		default:
			return refuse(name + " names no route this workflow runs")
		}
		if adapter == "" || adapter == route {
			return refuse(name + " names no adapter")
		}
		ins := fmt.Sprintf("install-%s-%s-%s-%s", adapter, route, run, attempt)
		if !isDir(filepath.Join(artifacts, ins)) {
			return refuse(fmt.Sprintf("%s has no %s beside it", name, ins))
		}
		lanes = append(lanes, lane{adapter, route, run + "-" + attempt})
	}
	if len(lanes) == 0 {
		return refuse(artifacts + " holds no capture-client artifact")
	}

	runSet := map[string]bool{}
	for _, l := range lanes {
		runSet[l.run] = true
	}
	if len(runSet) != 1 {
		var runs []string
		for r := range runSet {
			runs = append(runs, r)
		}
		sort.Strings(runs)
		return refuse(fmt.Sprintf("the artifacts come from %d runs or attempts: %s ",
			len(runs), strings.Join(runs, " ")))
	}
	run := lanes[0].run

	// ⚠ An install artifact with no capture beside it is a lane that installed
	// and never captured, and the pair it belonged to is refused below by its
	// count.
	for _, name := range globDirs(artifacts, "install-") {
		if !strings.HasSuffix(name, "-"+run) {
			return refuse(fmt.Sprintf("%s is not from run %s", name, run))
		}
	}

	// -- one store, every adapter in it -------------------------------------------
	store := filepath.Join(work, "store")
	if os.MkdirAll(store, 0o755) != nil {
		return 2
	}
	byAdapter := map[string][]string{}
	for _, l := range lanes {
		byAdapter[l.adapter] = append(byAdapter[l.adapter], l.route)
	}
	var adapters []string
	for a := range byAdapter {
		adapters = append(adapters, a)
	}
	sort.Strings(adapters)

	var resolutions []string
	for _, adapter := range adapters {
		routes := byAdapter[adapter]
		if len(routes) < 2 {
			return refuse(fmt.Sprintf("%s captured on %d route; a record needs two independent routes",
				adapter, len(routes)))
		}
		sort.Strings(routes)
		var laneArgs []string
		resolved := 0
		for _, route := range routes {
			capDir := filepath.Join(artifacts, fmt.Sprintf("capture-client-%s-%s-%s", adapter, route, run))
			insDir := filepath.Join(artifacts, fmt.Sprintf("install-%s-%s-%s", adapter, route, run))
			laneArgs = append(laneArgs, "--lane", capDir+","+insDir)
			res := filepath.Join(insDir, route, "resolution.json")
			if !isRegular(res) {
				res = filepath.Join(insDir, "release", "resolution.json")
			}
			if !isRegular(res) {
				// ⚠ A PACKAGE ROUTE HAS NO RESOLVER: the manager's own index
				// chose the version, as apt's and winget's do. Its scheme is the
				// target's, which a sibling lane's resolver states; every other
				// route without one is a lane whose resolver did not run.
				// Measured on capture-client run 38788416994, the first package
				// route a dispatch carried. `CI-11`.
				if route == "package" {
					continue
				}
				return refuse(fmt.Sprintf("the %s %s lane has no resolution.json", adapter, route))
			}
			resolutions = append(resolutions, res)
			resolved++
		}
		if resolved == 0 {
			return refuse(fmt.Sprintf("no %s lane has a resolution.json, so its version scheme is stated nowhere", adapter))
		}
		log := filepath.Join(work, "assemble.out")
		rc := runLogged(log, assemble, append(laneArgs, store)...)
		indent(os.Stdout, log)
		if rc != 0 {
			return refuse(fmt.Sprintf("%s did not assemble (exit %d)", adapter, rc))
		}
	}

	bundleArgs := append([]string{}, published...)
	for _, res := range resolutions {
		bundleArgs = append(bundleArgs, "--resolution", res)
	}
	if rc := assembleBundle(append(bundleArgs, store, out)); rc != 0 {
		return rc
	}
	fmt.Printf("assemble-dispatch: run %s, %d lane(s), bundle at %s\n", run, len(lanes), out)
	return 0
}

// isRegular is `[ -f PATH ]`.
func isRegular(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

// runLogged runs a program with stdin empty and stdout and stderr both in one
// FILE, `prog ... </dev/null >log 2>&1`, and answers its exit code.
func runLogged(log, prog string, args ...string) int {
	f, err := os.Create(log)
	if err != nil {
		return -1
	}
	defer f.Close()
	cmd := exec.Command(prog, args...)
	cmd.Stdout, cmd.Stderr = f, f
	if err := cmd.Run(); err != nil {
		if x, ok := err.(*exec.ExitError); ok {
			return x.ExitCode()
		}
		return -1
	}
	return 0
}

// schemeLine is the shell half's `jq -er` reading of one resolution: the
// target and the scheme it ordered under, as `target:prefix:min:max`, or false.
//
// ⚠ The text interpolation is jq's: a null or false prefix is `-`, a number is
// its integer spelling and a string itself, and anything else fails the shape
// the line must have.
func schemeLine(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var doc map[string]any
	if dec.Decode(&doc) != nil {
		return "", false
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", false
	}
	schema, _ := doc["schema"].(string)
	if schema != "bit-ids/resolution/1" && schema != "bit-ids/resolution/2" && schema != "bit-ids/resolution/3" {
		return "", false
	}
	scheme, _ := doc["scheme"].(map[string]any)
	text := func(v any, dflt bool) (string, bool) {
		switch x := v.(type) {
		case nil:
			if dflt {
				return "-", true
			}
			return "null", true
		case bool:
			if dflt && !x {
				return "-", true
			}
			return strconv.FormatBool(x), true
		case string:
			return x, true
		case json.Number:
			if i, err := strconv.ParseInt(x.String(), 10, 64); err == nil {
				return strconv.FormatInt(i, 10), true
			}
			return x.String(), true
		}
		return "", false
	}
	var field func(string) any = func(k string) any {
		if scheme == nil {
			return nil
		}
		return scheme[k]
	}
	target, ok1 := text(doc["target"], false)
	prefix, ok2 := text(field("tag_prefix"), true)
	minc, ok3 := text(field("min_components"), false)
	maxc, ok4 := text(field("max_components"), false)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return "", false
	}
	line := target + ":" + prefix + ":" + minc + ":" + maxc
	// ⛔ THE THIRD GENERATION STATES ITS SERIES, `null` for none, and a document
	// of it that omits the field is no scheme: an absent pin read as none would
	// order a series-pinned target across its series.
	if schema == "bit-ids/resolution/3" {
		raw, present := scheme["series"]
		if !present {
			return "", false
		}
		switch v := raw.(type) {
		case nil:
		case string:
			line += ":" + v
		default:
			return "", false
		}
	}
	if !schemeShape.MatchString(line) {
		return "", false
	}
	return line, true
}

// copyTree is `cp -R FROM TO` for a tree of directories, files and symbolic
// links: modes kept, links copied as links so a reader downstream refuses them.
func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(to, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, dst)
		case d.IsDir():
			return os.MkdirAll(dst, info.Mode().Perm()|0o700)
		case info.Mode().IsRegular():
			if err := copyFile(p, dst); err != nil {
				return err
			}
			return os.Chmod(dst, info.Mode().Perm())
		}
		return fmt.Errorf("%s is neither a file, a directory nor a link", p)
	})
}

func assembleBundle(args []string) int {
	var resolutions []string
	published := ""
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		if a == "--published" {
			if i+1 >= len(args) {
				fmt.Fprint(os.Stderr, "assemble-bundle: --published needs a directory\n")
				return 2
			}
			i++
			if !isDir(args[i]) {
				fmt.Fprintf(os.Stderr, "assemble-bundle: %s is not a directory\n", args[i])
				return 2
			}
			published = args[i]
			continue
		}
		if a == "--resolution" {
			if i+1 >= len(args) {
				fmt.Fprint(os.Stderr, "assemble-bundle: --resolution needs a file\n")
				return 2
			}
			i++
			if !isRegular(args[i]) {
				fmt.Fprintf(os.Stderr, "assemble-bundle: %s is not a file\n", args[i])
				return 2
			}
			resolutions = append(resolutions, args[i])
			continue
		}
		if a == "-h" || a == "--help" {
			fmt.Print(bundleUsage)
			return 0
		}
		if strings.HasPrefix(a, "-") {
			fmt.Fprintf(os.Stderr, "assemble-bundle: unknown argument: %s\n", a)
			return 2
		}
		break
	}
	rest := args[i:]
	if len(rest) != 2 {
		fmt.Fprint(os.Stderr, bundleUsage)
		return 2
	}
	store, out := rest[0], rest[1]
	if len(resolutions) == 0 {
		fmt.Fprint(os.Stderr, "assemble-bundle: at least one --resolution is required\n")
		return 2
	}
	if !isDir(store) {
		fmt.Fprintf(os.Stderr, "assemble-bundle: %s is not a directory\n", store)
		return 2
	}
	if _, err := os.Lstat(out); err == nil {
		fmt.Fprintf(os.Stderr, "assemble-bundle: %s already exists\n", out)
		return 2
	}

	root, err := helperRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "assemble-bundle: %v\n", err)
		return 2
	}
	tools := map[string]string{}
	for _, ex := range []string{"validate-corpus", "build-indexes", "build-formats", "assemble-release"} {
		bin, err := exampleBinary(root, ex, "bit-ids")
		if err != nil {
			fmt.Fprintf(os.Stderr, "assemble-bundle: %v\n", err)
			return 2
		}
		tools[ex] = bin
	}
	work, err := scratch("assemblebundle")
	if err != nil {
		fmt.Fprintf(os.Stderr, "assemble-bundle: %v\n", err)
		return 2
	}
	defer os.RemoveAll(work)

	// ⛔ A bundle that was not read back is removed, whatever stopped it: a
	// half-built one is a publishable-looking directory.
	var state sync.Mutex
	created, done := false, false
	removeHalf := func() {
		state.Lock()
		defer state.Unlock()
		if created && !done {
			os.RemoveAll(out)
		}
	}
	defer removeHalf()
	onInterrupt(func() { os.RemoveAll(work) })
	onInterrupt(removeHalf)
	mark := func(c, d bool) {
		state.Lock()
		created, done = c, d
		state.Unlock()
	}
	refuse := func(msg string) int {
		fmt.Fprintf(os.Stderr, "assemble-bundle: %s\n", msg)
		return 1
	}

	// -- the schemes, one per target, from what each resolver ordered under -----
	seen := map[string]bool{}
	var schemes []string
	for _, res := range resolutions {
		line, ok := schemeLine(res)
		if !ok {
			return refuse(res + " states no target and version scheme a resolver ordered under")
		}
		if !seen[line] {
			seen[line] = true
			schemes = append(schemes, line)
		}
	}
	// ⭐ AND THE SCHEMES THE PUBLISHED TREE WAS ORDERED UNDER, read back from
	// where the last bundle wrote them. ⛔ A line that is not a scheme is a
	// refusal, never a line passed over: a published record left with no scheme
	// would fail the index later with a reason that names nothing here.
	if published != "" {
		if lines := fileLines(filepath.Join(published, filepath.FromSlash(schemesFile))); lines != nil {
			for n, line := range lines {
				if !schemeShape.MatchString(line) {
					return refuse(fmt.Sprintf("the published %s line %d is not a scheme: %q", schemesFile, n+1, line))
				}
				if !seen[line] {
					seen[line] = true
					schemes = append(schemes, line)
				}
			}
		}
	}
	sort.Strings(schemes)
	for j := 1; j < len(schemes); j++ {
		t := strings.SplitN(schemes[j], ":", 2)[0]
		if t != strings.SplitN(schemes[j-1], ":", 2)[0] {
			continue
		}
		var named strings.Builder
		for _, s := range schemes {
			if strings.HasPrefix(s, t+":") {
				named.WriteString(s + " ")
			}
		}
		return refuse(fmt.Sprintf("the resolutions order %s under two schemes: %s", t, named.String()))
	}
	var schemeArgs []string
	for _, s := range schemes {
		schemeArgs = append(schemeArgs, "--scheme", s)
	}

	// -- the store is a corpus, and is only a store ------------------------------
	entries, err := os.ReadDir(store)
	if err != nil {
		return 2
	}
	// ⚠ The shell half's glob expands the visible names, then the hidden ones.
	sort.SliceStable(entries, func(a, b int) bool {
		ha, hb := strings.HasPrefix(entries[a].Name(), "."), strings.HasPrefix(entries[b].Name(), ".")
		return !ha && hb
	})
	for _, e := range entries {
		if e.Name() != "profiles" && e.Name() != "raw" {
			return refuse(fmt.Sprintf("the store carries %s, which no store holds; it describes another run", e.Name()))
		}
	}
	if !isDir(filepath.Join(store, "profiles")) {
		return refuse("the store has no profiles/")
	}

	// -- ⭐ THE PUBLISHED TREE, UNDER WHAT THIS RUN ADDS --------------------------
	//
	// ⛔ THE DATA BRANCH ONLY GROWS, so a bundle is the whole store and not this
	// run's records. The second publication on 2026-10-15 was refused by the
	// publisher with `E-STO-20` on every aria2-next path, because the bundle
	// carried Transmission alone. ⛔ A file both carry must be byte-identical:
	// the store never rewrites a published path, and a run whose record lands on
	// one with other bytes is refused here rather than at the publisher.
	if published != "" {
		merged := filepath.Join(work, "merged")
		for _, root := range []string{"profiles", "raw"} {
			if isDir(filepath.Join(store, root)) {
				if err := copyTree(filepath.Join(store, root), filepath.Join(merged, root)); err != nil {
					fmt.Fprintf(os.Stderr, "assemble-bundle: cannot copy %s/: %v\n", root, err)
					return 2
				}
			}
		}
		for _, root := range []string{"profiles", "raw"} {
			from := filepath.Join(published, root)
			if !isDir(from) {
				continue
			}
			var clash string
			err := filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, err := filepath.Rel(published, p)
				if err != nil {
					return err
				}
				dst := filepath.Join(merged, rel)
				if mine, err := os.ReadFile(dst); err == nil {
					theirs, err := os.ReadFile(p)
					if err != nil {
						return err
					}
					if !bytes.Equal(mine, theirs) {
						clash = filepath.ToSlash(rel)
						return fs.SkipAll
					}
					return nil
				}
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return err
				}
				return copyFile(p, dst)
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "assemble-bundle: cannot read the published tree: %v\n", err)
				return 2
			}
			if clash != "" {
				return refuse(clash + " is published, and this run carries other bytes at that path; the store never rewrites one")
			}
		}
		store = merged
	}

	corpus := filepath.Join(work, "corpus.out")
	if rc := runLogged(corpus, tools["validate-corpus"], store); rc != 0 {
		fmt.Fprintf(os.Stderr, "assemble-bundle: the store is not a corpus (exit %d)\n", rc)
		indent(os.Stderr, corpus)
		return 1
	}

	// -- compose -----------------------------------------------------------------
	mark(true, false)
	// ⚠ Each failure names itself, as `cp` did for the shell half. A review
	// found these exiting 2 with nothing said.
	cannot := func(what string, err error) int {
		fmt.Fprintf(os.Stderr, "assemble-bundle: cannot %s: %v\n", what, err)
		return 2
	}
	if err := os.MkdirAll(filepath.Join(out, "indexes", "v1"), 0o755); err != nil {
		return cannot("make the bundle", err)
	}
	if err := copyTree(filepath.Join(store, "profiles"), filepath.Join(out, "profiles")); err != nil {
		return cannot("copy profiles/", err)
	}
	if isDir(filepath.Join(store, "raw")) {
		if err := copyTree(filepath.Join(store, "raw"), filepath.Join(out, "raw")); err != nil {
			return cannot("copy raw/", err)
		}
	}
	if err := copyFile(filepath.Join(root, "LICENSE"), filepath.Join(out, "LICENSE")); err != nil {
		return cannot("copy the licence", err)
	}

	step := func(log, failure, prog string, args ...string) bool {
		if runLogged(log, prog, args...) != 0 {
			indent(os.Stderr, log)
			fmt.Fprintf(os.Stderr, "assemble-bundle: %s\n", failure)
			return false
		}
		return true
	}
	if !step(filepath.Join(work, "indexes.out"), "the indexes were not built", tools["build-indexes"],
		append(append([]string{}, schemeArgs...), store, filepath.Join(out, "indexes", "v1", "profiles.json"))...) {
		return 1
	}
	if !step(filepath.Join(work, "formats.out"), "the formats were not rendered", tools["build-formats"],
		append(append([]string{}, schemeArgs...), store, out)...) {
		return 1
	}
	var listed strings.Builder
	for _, s := range schemes {
		listed.WriteString(s + "\n")
	}
	if err := os.WriteFile(filepath.Join(out, filepath.FromSlash(schemesFile)), []byte(listed.String()), 0o644); err != nil {
		return cannot("write "+schemesFile, err)
	}
	release := filepath.Join(work, "release.out")
	if !step(release, "the release was not assembled", tools["assemble-release"], out) {
		return 1
	}

	mark(true, true)
	objects := 0
	_ = filepath.WalkDir(out, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			objects++
		}
		return nil
	})
	fmt.Printf("assemble-bundle: %d object(s) in %s\n", objects, out)
	fmt.Printf("  schemes:%s\n", " "+strings.Join(schemeArgs, " "))
	indent(os.Stdout, corpus)
	first := ""
	if l := fileLines(release); len(l) > 0 {
		first = l[0]
	}
	fmt.Printf("  release: %s\n", first)
	return 0
}
