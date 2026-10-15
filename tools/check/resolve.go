// resolve-release and resolve-source - decide what a route will fetch or build,
// on a host that still has its way out, and keep the evidence the decision was
// made from.
//
// CI-11. They were scripts/acquisition/resolve-release.sh and
// resolve-source.sh, and they are one file here because they are one shape:
// ask the adapter how its target spells versions, read one index of the
// vendor's repository, hand the bytes to `resolve-stable`, and record what was
// read beside what was chosen.
//
// -- ⭐ WHERE EACH DECISION LIVES ---------------------------------------------
//
// The adapter says WHICH repository, HOW that target spells versions and WHICH
// artifact is the installable one, through `describe`. `fetch-releases`
// retrieves and keeps the exact bytes, `resolve-stable` orders the versions,
// `select-asset` picks the artifact: this orchestrates and Rust parses, which is
// the line docs/architecture.md section 3 draws.
//
// ⛔ TWO INDEXES ARE TWO RESOLVERS, which is what `E-ACQ-07` compares. The
// release route reads the API's release OBJECTS; the source route reads the
// repository's own TAGS through `git ls-remote --tags --refs`, a different and
// much larger population: measured on 2026-09-05, `qbittorrent`'s releases
// endpoint answered four entries and its tags at least a hundred. ⭐ The
// ordering is the same and that is deliberate: `VersionScheme::components` is
// this project's one version ordering.
//
// -- ⛔ WHAT THE DESCRIPTION MUST DECLARE, AND NEVER DEFAULTED ----------------
//
// Every field is asked for and a missing one is could-not-run rather than an
// empty string that composes into a nonsense URL. ⛔ The digest disposition is
// declared, never defaulted: an adapter says which asset carries the vendor's
// checksums or, in as many words, that the vendor publishes none, and one that
// says neither or both is refused. ⛔ The floor is declared, `-` for none: a
// floor this invented would be a selection this made. `CLIENT-06`.
//
// -- ⚠ RECORDED SOURCES ARE SAID OUT LOUD ------------------------------------
//
// `--listing`, `--checksums` and `--refs` make each source a file, so a gate
// can exercise a rule without a vendor. Each prints that it read a file, because
// a run over a recorded response and one that asked the vendor are otherwise the
// same lines in a log. ⛔ `--listing` without `--checksums` is refused where the
// target declares a document: a recorded listing names real vendor URLs, and
// the digest fetch would reach one out of a harness written not to.
//
// -- ⚠ WHERE THIS DIFFERS FROM THE SHELL HALVES ------------------------------
//
//   - The adapter's `describe` is still `sh <adapter> describe`, bounded as
//     `timeout -k` bounded it: TERM at the limit, KILL after the grace. The
//     adapters are shell until they port, and this is the one door to them.
//   - Digests are Go's SHA-256, as `sha256sum` printed them, and the bounds are
//     this process's rather than coreutils `timeout`'s, which Windows lacks.
//   - The repository the examples are found in is the one `git` names from
//     where this runs, where the shell halves found theirs from their own path.
//
// Usage:
//
//	bit-check resolve-release --adapter <path> --workdir <dir> [--record <file>]
//	                          [--listing <file>] [--checksums <file>]
//	bit-check resolve-source --adapter <path> --workdir <dir> [--record <file>]
//	                         [--refs <file>]
//
// resolve-release prints the artifact URL and resolve-source the selected tag,
// on stdout. Exit codes: 0 resolved, 1 the resolver or the selection refused,
// 2 could not run.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// terminate is what `timeout` sends at its bound: TERM, so the adapter's own
// cleanup can run. ⚠ Windows has no TERM to send a process, so there the bound
// kills, which is the only stop that platform offers.
func terminate(p *os.Process) error {
	if runtime.GOOS == "windows" {
		return p.Kill()
	}
	return p.Signal(syscall.SIGTERM)
}

type resolveExit struct {
	code int
	msg  string
}

func (e resolveExit) Error() string { return e.msg }

func utcNow() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }

// keyField is `awk -F= -v k=KEY '$1 == k { sub(/^[^=]*=/, ""); print; exit }'`.
func keyField(body []byte, key string) string {
	for _, line := range strings.Split(string(body), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok && k == key {
			return v
		}
		if !ok && line == key {
			return ""
		}
	}
	return ""
}

// builtExample is a Rust example that is ALREADY built, or could-not-run. ⛔ A
// resolver never builds: the build happens with the observer, while the
// network is there, and a step that discovered a missing binary later would be
// building on the host the capture exists to contain.
func builtExample(name string) (string, error) {
	root, err := helperRoot()
	if err != nil {
		return "", err
	}
	target := os.Getenv("CARGO_TARGET_DIR")
	if target == "" {
		target = filepath.Join(root, "target")
	}
	dir := filepath.Join(target, "debug", "examples")
	bin := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if fi, err := os.Stat(bin); err != nil || fi.IsDir() || (runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0) {
		return "", fmt.Errorf("%s/%s is not built; run: cargo build -p bit-ids --locked --example %s", dir, name, name)
	}
	return bin, nil
}

func envSeconds(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

// runBounded runs a command with stdin empty and stdout and stderr in FILES,
// TERM at the bound and KILL after the grace, as `timeout -k grace bound` did.
// ⚠ Files, so nothing it spawns holds a pipe this process is reading.
func runBounded(stdout, stderr string, bound, grace int, prog string, args ...string) error {
	out, err := os.Create(stdout)
	if err != nil {
		return err
	}
	defer out.Close()
	errf, err := os.Create(stderr)
	if err != nil {
		return err
	}
	defer errf.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(bound)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Stdout, cmd.Stderr = out, errf
	cmd.Cancel = func() error { return terminate(cmd.Process) }
	if grace > 0 {
		cmd.WaitDelay = time.Duration(grace) * time.Second
	}
	return cmd.Run()
}

// describeAdapter asks the adapter to describe itself, bounded, into the
// workdir. ⛔ BOUNDED BECAUSE IT HAD NO LIMIT AT ALL UNTIL 2026-09-16, in a step
// whose job is to answer before the capture is contained: an adapter that never
// returned wedged the job with nothing to read, the way `capture-client` runs
// 21, 22 and 23 were cancelled with no log and no artifact.
func describeAdapter(me, adapter, workdir string) ([]byte, error) {
	desc := filepath.Join(workdir, "describe.txt")
	err := runBounded(desc, filepath.Join(workdir, "describe.err"),
		envSeconds("BIT_IDS_DESCRIBE_TIMEOUT", 30), envSeconds("BIT_IDS_KILL_AFTER", 20),
		"sh", adapter, "describe")
	if err != nil {
		return nil, resolveExit{2, "the adapter could not describe itself"}
	}
	return os.ReadFile(desc)
}

// The release and source routes' shared declarations.
type scheme struct {
	target, repo, prefix, minc, maxc, floor string
}

func readScheme(desc []byte, route string) (scheme, error) {
	var s scheme
	cannot := func(f string, a ...any) error { return resolveExit{2, fmt.Sprintf(f, a...)} }
	s.target = keyField(desc, "target")
	if s.target == "" {
		return s, cannot("the adapter named no target")
	}
	s.repo = keyField(desc, "release_repo")
	if s.repo == "" {
		if route == "source" {
			return s, cannot("%s declares no release_repo, so there is no repository to read refs from", s.target)
		}
		return s, cannot("%s declares no release_repo, so it has no release route to resolve", s.target)
	}
	if s.prefix = keyField(desc, "release_tag_prefix"); s.prefix == "" {
		return s, cannot("%s declares release_repo and no release_tag_prefix", s.target)
	}
	if s.minc = keyField(desc, "release_min_components"); s.minc == "" {
		return s, cannot("%s declares no release_min_components", s.target)
	}
	if s.maxc = keyField(desc, "release_max_components"); s.maxc == "" {
		return s, cannot("%s declares no release_max_components", s.target)
	}
	return s, nil
}

// The checks that follow the asset declarations in the release route and
// directly follow the component counts in the source route.
func (s scheme) checkNumbers(desc []byte) (string, error) {
	cannot := func(f string, a ...any) error { return resolveExit{2, fmt.Sprintf(f, a...)} }
	for _, n := range []string{s.minc, s.maxc} {
		if n == "" || strings.Trim(n, "0123456789") != "" {
			return "", cannot("%s declares a non-numeric component count [%s]", s.target, n)
		}
	}
	floor := keyField(desc, "release_floor_major")
	switch {
	case floor == "":
		return "", cannot("%s declares no release_floor_major; - says it has none", s.target)
	case floor == "-":
	case strings.Trim(floor, "0123456789") != "":
		return "", cannot("%s declares a non-numeric release_floor_major [%s]", s.target, floor)
	}
	return floor, nil
}

// indentFile is `sed 's/^/          /' FILE >&2`.
func indentFile(path string) {
	body, err := os.ReadFile(path)
	if err != nil || len(body) == 0 {
		return
	}
	for _, l := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
		fmt.Fprintf(os.Stderr, "          %s\n", l)
	}
}

// runExample runs an example with stdout and stderr in files and answers its
// exit code, or -1 when it could not be started.
func runExample(stdout, stderr, prog string, args ...string) int {
	out, err := os.Create(stdout)
	if err != nil {
		return -1
	}
	defer out.Close()
	errf, err := os.Create(stderr)
	if err != nil {
		return -1
	}
	defer errf.Close()
	cmd := exec.Command(prog, args...)
	cmd.Stdout, cmd.Stderr = out, errf
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return -1
	}
	return 0
}

type resolveArgs struct {
	adapter, workdir, record, listing, checksums, refs string
}

func parseResolveArgs(me string, args []string, flags map[string]*string) (bool, error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-h" || a == "--help" {
			return true, nil
		}
		dst, ok := flags[a]
		if !ok {
			return false, resolveExit{2, "unknown argument: " + a}
		}
		if i+1 >= len(args) {
			return false, resolveExit{2, a + " takes a value"}
		}
		i++
		*dst = args[i]
	}
	return false, nil
}

func resolveMain(me, usage string, args []string, run func([]string) error) int {
	err := run(args)
	if err == nil {
		return 0
	}
	if e, ok := err.(resolveExit); ok {
		if e.code == -1 {
			fmt.Print(usage)
			return 0
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", me, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", me, err)
	return 2
}

func prepareWorkdir(a *resolveArgs) error {
	if a.adapter == "" {
		return resolveExit{2, "--adapter is required"}
	}
	if a.workdir == "" {
		return resolveExit{2, "--workdir is required"}
	}
	if !isRegular(a.adapter) {
		return resolveExit{2, fmt.Sprintf("the adapter %s is not present", a.adapter)}
	}
	return nil
}

func absWorkdir(a *resolveArgs) error {
	if err := os.MkdirAll(a.workdir, 0o755); err != nil {
		return resolveExit{2, "cannot create " + a.workdir}
	}
	abs, err := filepath.Abs(a.workdir)
	if err != nil {
		return resolveExit{2, "cannot create " + a.workdir}
	}
	a.workdir = abs
	return nil
}

func copyIn(from, to, what string) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return resolveExit{2, fmt.Sprintf("cannot copy %s into the workdir", from)}
	}
	if err := os.WriteFile(to, b, 0o644); err != nil {
		return resolveExit{2, fmt.Sprintf("cannot copy %s into the workdir", from)}
	}
	_ = what
	return nil
}

func stampFetched(path string) error {
	if err := os.WriteFile(path+".fetched-at", []byte(utcNow()+"\n"), 0o644); err != nil {
		return resolveExit{2, "cannot stamp the retrieval instant"}
	}
	return nil
}

// -- resolve-release ----------------------------------------------------------

const resolveReleaseUsage = `usage: bit-check resolve-release --adapter <path> --workdir <dir> [--record <file>]
                                  [--listing <file>] [--checksums <file>]

Prints the artifact URL. Exit codes: 0 resolved, 1 the resolver or the
selection refused, 2 could not run.
`

func resolveRelease(args []string) int {
	return resolveMain("resolve-release", resolveReleaseUsage, args, runResolveRelease)
}

func runResolveRelease(args []string) error {
	var a resolveArgs
	help, err := parseResolveArgs("resolve-release", args, map[string]*string{
		"--adapter": &a.adapter, "--workdir": &a.workdir, "--record": &a.record,
		"--listing": &a.listing, "--checksums": &a.checksums,
	})
	if help {
		return resolveExit{-1, ""}
	}
	if err != nil {
		return err
	}
	if err := prepareWorkdir(&a); err != nil {
		return err
	}
	cannot := func(f string, x ...any) error { return resolveExit{2, fmt.Sprintf(f, x...)} }
	refuse := func(f string, x ...any) error { return resolveExit{1, fmt.Sprintf(f, x...)} }

	resolver, err := builtExample("resolve-stable")
	if err != nil {
		return cannot("%v", err)
	}
	selector, err := builtExample("select-asset")
	if err != nil {
		return cannot("%v", err)
	}
	if err := absWorkdir(&a); err != nil {
		return err
	}
	desc, err := describeAdapter("resolve-release", a.adapter, a.workdir)
	if err != nil {
		return err
	}
	s, err := readScheme(desc, "release")
	if err != nil {
		return err
	}
	pattern := keyField(desc, "release_asset")
	if pattern == "" {
		return cannot("%s declares no release_asset pattern", s.target)
	}
	// ⚠ Two declarations is a target whose two answers could disagree, and
	// choosing between them here would be this deciding which evidence counts.
	digestPattern := keyField(desc, "release_digest_asset")
	digestNone := keyField(desc, "release_digest_unpublished")
	if digestPattern != "" && digestNone != "" {
		return cannot("%s declares both release_digest_asset and release_digest_unpublished", s.target)
	}
	if digestPattern == "" && digestNone == "" {
		return cannot("%s declares neither release_digest_asset nor release_digest_unpublished", s.target)
	}
	floor, err := s.checkNumbers(desc)
	if err != nil {
		return err
	}

	started := utcNow()
	listing := filepath.Join(a.workdir, "releases.json")
	var url string
	if a.listing != "" {
		if !isRegular(a.listing) {
			return cannot("the listing %s is not present", a.listing)
		}
		if err := copyIn(a.listing, listing, "listing"); err != nil {
			return err
		}
		// ⚠ The resolver reads the retrieval instant from the sidecar a fetch
		// writes, so a recorded response needs one, stamped now, because now is
		// when this run read those bytes.
		if err := stampFetched(listing); err != nil {
			return err
		}
		url = "https://recorded.invalid/" + a.listing
		fmt.Fprintf(os.Stderr, "resolve-release: read the release listing from %s, not from %s\n", a.listing, s.repo)
	} else {
		// ⭐ THE FETCH IS THIS BINARY'S OWN `fetch-releases`, in this process,
		// rather than a second spelling of it.
		var out bytes.Buffer
		self, err := os.Executable()
		if err != nil {
			return cannot("fetch-releases could not run (exit 2)")
		}
		cmd := exec.Command(self, "fetch-releases", s.repo, listing)
		cmd.Stdout, cmd.Stderr = &out, os.Stderr
		rc := 0
		if err := cmd.Run(); err != nil {
			rc = 2
			if ee, ok := err.(*exec.ExitError); ok {
				rc = ee.ExitCode()
			}
		}
		switch rc {
		case 0:
		case 1:
			return refuse("the release listing for %s could not be retrieved", s.repo)
		default:
			return cannot("fetch-releases could not run (exit %d)", rc)
		}
		url = strings.TrimRight(out.String(), "\n")
		if url == "" {
			return cannot("fetch-releases printed no source URL")
		}
	}

	// ⛔ THE RESOLUTION DOCUMENT IS KEPT, NOT ONLY THE ANSWER: every candidate
	// with the verdict it got and a digest of the bytes that were read, so a
	// selection is re-derivable rather than asserted.
	resolution := filepath.Join(a.workdir, "resolution.json")
	rerr := filepath.Join(a.workdir, "resolve.err")
	switch rc := runExample(resolution, rerr, resolver, s.target, s.prefix, s.minc, s.maxc, floor,
		"github-releases", url, listing); rc {
	case 0:
	case 1:
		indentFile(rerr)
		return refuse("the resolver failed closed for %s; %s says why", s.target, resolution)
	default:
		indentFile(rerr)
		return cannot("resolve-stable could not run (exit %d)", rc)
	}

	selection := filepath.Join(a.workdir, "selection.txt")
	serr := filepath.Join(a.workdir, "select.err")
	rc := runExample(selection, serr, selector, resolution, listing, pattern)
	// ⚠ THE OFFER IS PRINTED WHATEVER HAPPENS: on a refusal it is the
	// diagnosis, and on success the evidence that exactly one asset matched.
	indentFile(serr)
	switch rc {
	case 0:
	case 1:
		return refuse("no single asset of %s's release matches [%s]", s.target, pattern)
	default:
		return cannot("select-asset could not run (exit %d)", rc)
	}
	sel, _ := os.ReadFile(selection)
	version := keyField(sel, "version")
	tag := keyField(sel, "tag")
	asset := keyField(sel, "asset")
	assetURL := keyField(sel, "url")
	size := keyField(sel, "size")
	// ⭐ WHAT THE SOURCE SAYS THE BYTES DIGEST TO, out of the SAME response that
	// chose the version and the artifact. Empty means it stated none.
	assetSHA := keyField(sel, "digest")
	if assetURL == "" {
		return cannot("select-asset printed no url")
	}

	// -- the vendor's digest, out of the same response ------------------------
	var digestAsset, digestURL, digestDoc string
	if digestPattern != "" {
		dsel := filepath.Join(a.workdir, "digest-selection.txt")
		dserr := filepath.Join(a.workdir, "digest-select.err")
		rc := runExample(dsel, dserr, selector, resolution, listing, digestPattern)
		indentFile(dserr)
		switch rc {
		case 0:
		case 1:
			return refuse("%s declares a checksums asset and no single asset matches [%s]", s.target, digestPattern)
		default:
			return cannot("select-asset could not run for the checksums asset (exit %d)", rc)
		}
		db, _ := os.ReadFile(dsel)
		digestAsset = keyField(db, "asset")
		digestURL = keyField(db, "url")
		if digestURL == "" {
			return cannot("select-asset printed no url for the checksums asset")
		}
		if a.listing != "" && a.checksums == "" {
			return cannot("--listing reads a recorded response, so --checksums must record the digest document too")
		}
		digestDoc = filepath.Join(a.workdir, "checksums.txt")
		if a.checksums != "" {
			if !isRegular(a.checksums) {
				return cannot("the checksums document %s is not present", a.checksums)
			}
			if err := copyIn(a.checksums, digestDoc, "checksums"); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "resolve-release: read the checksums document from %s, not from %s\n", a.checksums, digestURL)
		} else {
			// ⛔ THE FETCH IS THE INSTALLER'S, WHERE THE BOUND AND THE DIGEST
			// RULE LIVE. A checksums document publishes no digest of itself,
			// which is what `--digest-unpublished` says, and nothing executes it.
			self, err := os.Executable()
			if err != nil {
				return cannot("install-rootless could not run for the checksums asset (exit 2)")
			}
			flog := filepath.Join(a.workdir, "digest-fetch.log")
			rc := runExample(flog, flog+".err", self, "install-rootless", "--fetch", "--url", digestURL,
				"--into", digestDoc, "--digest-unpublished", "a checksums document publishes no digest of itself")
			mergeLog(flog, flog+".err")
			switch rc {
			case 0:
			case 1:
				indentFile(flog)
				return refuse("the checksums document %s could not be retrieved", digestAsset)
			default:
				indentFile(flog)
				return cannot("install-rootless could not run for the checksums asset (exit %d)", rc)
			}
		}
		// ⛔ AND THE DOCUMENT IS ASKED WHETHER IT COVERS THE ASSET SELECTED,
		// here, rather than at install time on a contained host. ⛔ THE NAME IS
		// COMPARED, NEVER MATCHED: a `.` in a pattern matches any byte, and
		// `foo.tar.gz` is a substring of `foo.tar.gz.asc`.
		doc, _ := os.ReadFile(digestDoc)
		if !docNames(doc, asset) {
			return refuse("%s does not name %s, so it cannot verify the artifact this route installs", digestAsset, asset)
		}
	}

	// -- which of the four dispositions this resolution reached -----------------
	//
	// ⛔ TWO INDEPENDENT STATEMENTS ABOUT ONE ARTIFACT ARE NOT ONE: the vendor's
	// document and the serving party's listing digest answer different
	// questions, so `both` is its own value.
	var digestSource string
	switch {
	case digestDoc != "" && assetSHA != "":
		digestSource = "both"
	case digestDoc != "":
		digestSource = "vendor-document"
	case assetSHA != "":
		digestSource = "source-listing"
	default:
		digestSource = "unpublished"
	}
	finished := utcNow()

	if a.record != "" {
		lsum, _ := fileSHA256(listing)
		asum, _ := fileSHA256(a.adapter)
		var b strings.Builder
		b.WriteString("bit-ids/release-resolution/1\n")
		for _, kv := range [][2]string{
			{"target", s.target}, {"repository", s.repo}, {"source_url", url},
			{"listing_sha256", lsum}, {"selected_version", version}, {"selected_tag", tag},
			{"asset_pattern", pattern}, {"asset", asset}, {"asset_url", assetURL},
			{"asset_declared_size", size}, {"digest_source", digestSource},
			{"asset_sha256", assetSHA}, {"digest_asset", digestAsset},
			{"digest_asset_url", digestURL}, {"digest_document", digestDoc},
			{"digest_unpublished", digestNone}, {"adapter", a.adapter},
			{"adapter_sha256", asum}, {"started_at", started}, {"finished_at", finished},
		} {
			fmt.Fprintf(&b, "%s=%s\n", kv[0], kv[1])
		}
		if err := os.WriteFile(a.record, []byte(b.String()), 0o644); err != nil {
			return cannot("the resolution record could not be written to %s", a.record)
		}
		if err := recordHas(a.record, "target", "repository", "source_url", "listing_sha256",
			"selected_version", "selected_tag", "asset_pattern", "asset", "asset_url", "digest_source"); err != nil {
			return err
		}
	}

	fmt.Println(assetURL)
	fmt.Fprintf(os.Stderr, "resolve-release: %s %s -> %s (%s bytes declared)\n", s.target, version, asset, size)
	return nil
}

// mergeLog appends the second file to the first and removes it, which is what
// `>log 2>&1` gave the shell half in one file.
func mergeLog(log, extra string) {
	b, _ := os.ReadFile(extra)
	if len(b) > 0 {
		f, err := os.OpenFile(log, os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			f.Write(b)
			f.Close()
		}
	}
	os.Remove(extra)
}

// docNames is the awk that strips a digest prefix and compares the rest with
// `==`: neither a regex nor a substring.
func docNames(doc []byte, want string) bool {
	for _, line := range strings.Split(string(doc), "\n") {
		rest := line
		i := 0
		for i < len(rest) && strings.ContainsRune("0123456789abcdefABCDEF", rune(rest[i])) {
			i++
		}
		if i > 0 {
			j := i
			for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
				j++
			}
			if j > i {
				rest = rest[j:]
				rest = strings.TrimPrefix(rest, "*")
			}
		}
		if rest == want {
			return true
		}
	}
	return false
}

// recordHas asks each field of a written record for itself: a record truncated
// part-way through would carry a correct target and no URL, and a caller
// reading an absent field as empty is the failure these records exist to stop.
func recordHas(path string, keys ...string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return resolveExit{2, "the resolution record could not be read back"}
	}
	for _, k := range keys {
		if !bytes.Contains(append([]byte("\n"), body...), []byte("\n"+k+"=")) {
			return resolveExit{2, "the resolution record was written without " + k}
		}
	}
	return nil
}

// -- resolve-source -----------------------------------------------------------

const resolveSourceUsage = `usage: bit-check resolve-source --adapter <path> --workdir <dir> [--record <file>]
                                 [--refs <file>]

Prints the selected tag. Exit codes: 0 resolved, 1 the resolver refused, 2
could not run.
`

func resolveSource(args []string) int {
	return resolveMain("resolve-source", resolveSourceUsage, args, runResolveSource)
}

func runResolveSource(args []string) error {
	var a resolveArgs
	help, err := parseResolveArgs("resolve-source", args, map[string]*string{
		"--adapter": &a.adapter, "--workdir": &a.workdir, "--record": &a.record, "--refs": &a.refs,
	})
	if help {
		return resolveExit{-1, ""}
	}
	if err != nil {
		return err
	}
	if err := prepareWorkdir(&a); err != nil {
		return err
	}
	cannot := func(f string, x ...any) error { return resolveExit{2, fmt.Sprintf(f, x...)} }
	refuse := func(f string, x ...any) error { return resolveExit{1, fmt.Sprintf(f, x...)} }

	resolver, err := builtExample("resolve-stable")
	if err != nil {
		return cannot("%v", err)
	}
	if err := absWorkdir(&a); err != nil {
		return err
	}
	desc, err := describeAdapter("resolve-source", a.adapter, a.workdir)
	if err != nil {
		return err
	}
	// ⚠ THE SAME REPOSITORY THE RELEASE ROUTE NAMES, and that is the point: the
	// two routes differ in which INDEX of it decided the version. `CLIENT-14`
	// records that this pair is weaker than a distribution index against a
	// vendor release, because whoever controls the repository controls both.
	s, err := readScheme(desc, "source")
	if err != nil {
		return err
	}
	floor, err := s.checkNumbers(desc)
	if err != nil {
		return err
	}

	started := utcNow()
	refs := filepath.Join(a.workdir, "refs.txt")
	var url string
	if a.refs != "" {
		if !isRegular(a.refs) {
			return cannot("the refs listing %s is not present", a.refs)
		}
		if err := copyIn(a.refs, refs, "refs"); err != nil {
			return err
		}
		if err := stampFetched(refs); err != nil {
			return err
		}
		url = "https://recorded.invalid/" + a.refs
		fmt.Fprintf(os.Stderr, "resolve-source: read the refs from %s, not from %s\n", a.refs, s.repo)
	} else {
		git, err := exec.LookPath("git")
		if err != nil {
			return cannot("git is not on this host")
		}
		url = "https://github.com/" + s.repo + ".git"
		// ⛔ --refs, SO AN ANNOTATED TAG IS OFFERED ONCE: without it `ls-remote`
		// emits a second `^{}` line per annotated tag, which the reader refuses
		// rather than skipping. ⚠ git-over-HTTPS rather than a REST read, so
		// `AGENTS.md` rule 8's route does not apply; measured on 2026-09-09.
		if err := runBounded(refs, filepath.Join(a.workdir, "ls-remote.err"), 120, 0,
			git, "ls-remote", "--tags", "--refs", url); err != nil {
			return refuse("the refs for %s could not be retrieved", s.repo)
		}
		if err := stampFetched(refs); err != nil {
			return err
		}
	}
	if fi, err := os.Stat(refs); err != nil || fi.Size() == 0 {
		return refuse("%s offered no tags at all", s.repo)
	}

	resolution := filepath.Join(a.workdir, "resolution.json")
	rerr := filepath.Join(a.workdir, "resolve.err")
	// ⛔ THE READER'S OWN WORDS ARE PRINTED ON EVERY NON-ZERO STATUS, not only
	// on a refusal: the first shell version swallowed them for exit 2, so a
	// listing this project could not read reported "could not run" with the
	// sentence naming the bad line in a workdir file nobody opens.
	switch rc := runExample(resolution, rerr, resolver, s.target, s.prefix, s.minc, s.maxc, floor,
		"git-refs", url, refs); rc {
	case 0:
	case 1:
		indentFile(rerr)
		return refuse("the resolver failed closed over %s's tags", s.repo)
	default:
		indentFile(rerr)
		return cannot("resolve-stable could not run (exit %d)", rc)
	}

	doc, _ := os.ReadFile(resolution)
	selected, tag := selectedTag(doc)
	if selected == "" {
		return cannot("the resolution names no selected version")
	}
	if tag == "" {
		return cannot("the resolution names no tag for the selected version")
	}
	digest, _ := fileSHA256(refs)
	finished := utcNow()
	if a.record != "" {
		asum, _ := fileSHA256(a.adapter)
		var b strings.Builder
		b.WriteString("bit-ids/source-resolution/1\n")
		// ⛔ `source_url` IS THE FIELD `assemble-capture` SLUGIFIES INTO A
		// RESOLVER IDENTITY, so it must differ from the release route's.
		for _, kv := range [][2]string{
			{"target", s.target}, {"repository", s.repo}, {"source_url", url},
			{"refs_sha256", digest}, {"selected_version", selected}, {"selected_tag", tag},
			{"adapter", a.adapter}, {"adapter_sha256", asum},
			{"started_at", started}, {"finished_at", finished},
		} {
			fmt.Fprintf(&b, "%s=%s\n", kv[0], kv[1])
		}
		if err := os.WriteFile(a.record, []byte(b.String()), 0o644); err != nil {
			return cannot("the resolution record could not be written to %s", a.record)
		}
		if err := recordHas(a.record, "target", "repository", "source_url", "refs_sha256",
			"selected_version", "selected_tag"); err != nil {
			return err
		}
	}
	fmt.Println(tag)
	return nil
}

// selectedTag reads the selection and the tag OF THE ENTRY THAT WON out of a
// resolution document, as the shell half's two awk programs did.
//
// ⛔ NOT THE NEXT TAG AFTER THE WORD `selected`. `tag` comes BEFORE `verdict`
// inside an entry, so the first shell version printed the FOLLOWING candidate's
// tag: `v2.7.4` over a resolution that selected 2.7.5. The tag is remembered
// per entry and printed when THAT entry's verdict is the selection, with the
// entry boundary resetting it.
func selectedTag(doc []byte) (selected, tag string) {
	quoted := func(line string, n int) string {
		parts := strings.Split(line, `"`)
		if len(parts) > n {
			return parts[n]
		}
		return ""
	}
	lines := strings.Split(string(doc), "\n")
	for _, l := range lines {
		if strings.Contains(l, `"selected"`) {
			selected = quoted(l, 3)
			break
		}
	}
	cur := ""
	for _, l := range lines {
		if strings.Contains(l, `"candidate":`) {
			cur = ""
		}
		if strings.Contains(l, `"tag":`) && cur == "" {
			cur = quoted(l, 3)
		}
		if strings.Contains(l, `"verdict": "selected"`) && cur != "" {
			return selected, cur
		}
	}
	return selected, ""
}
