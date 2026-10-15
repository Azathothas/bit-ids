// install-rootless - fetch a published build, identify its bytes, and place it
// where the current user already has write access. No privilege, no package
// index, no host state outside one prefix this derives.
//
// CI-11. It was scripts/acquisition/install-rootless.sh, which every release
// route and the resolver's checksums fetch call, so it leaves `sh` before
// they do.
//
// ⛔ IT EXISTS BECAUSE EVERY INSTALL THIS PROJECT PERFORMED NEEDED `sudo` AND
// WROTE TO `/usr/local/bin`. What such an install does is a property of a host
// this project does not control, and `ACQ-06` removed the variable rather than
// measuring it. `TODO/acquisition.md` carries the argument.
//
// -- the prefix -------------------------------------------------------------
//
// ⛔ THE PREFIX HAS ONE DERIVATION AND `--prefix` IS IT. Every caller asks this
// rather than composing a path of its own, for the reason
// `assert-disposable --marker` exists: a second spelling goes on reading the
// old place the day the first one moves. ⚠ `<home>/.local/share/bit-ids`
// rather than `<home>/.local`, because a build placed in a user's own prefix
// would shadow their tools on PATH. One directory carries all of it and is
// removed in one command. ⛔ AND THIS NEVER ESCALATES: a prefix the current
// user cannot write is a refusal naming it, never a `sudo` retry.
//
// -- identifying the bytes --------------------------------------------------
//
// ⛔ NOTHING IS MADE EXECUTABLE BEFORE ITS DIGEST IS SETTLED, and the
// disposition is declared rather than defaulted:
//
//	--sha256 <hex>              the SOURCE published this digest of the bytes
//	                            it serves; the same party as the one serving them
//	--sums <file> --sums-name <name>
//	                            the VENDOR published a document, and `sha256sum
//	                            -c` (`certutil` on Windows) is the verifier, so
//	                            neither the digest nor the comparison is this
//	                            project's code
//	--digest-unpublished <why>  neither exists, and <why> is the measured reason
//
// ⭐ The first two together are two statements from two parties and both are
// checked. ⛔ The third may not be combined with either.
//
// ⭐ `--from-resolution` READS ALL THREE OUT OF `resolve-release`'s RECORD, once,
// here, so four adapters do not each choose between four dispositions.
//
// -- ⚠ WHERE THIS DIFFERS FROM THE SHELL HALF ------------------------------
//
//   - The home is os.UserHomeDir: `$HOME` where the shell half read it, and
//     `%USERPROFILE%` on Windows, which has no `$HOME` to require.
//   - `install -m755` and `mv` are a copy at mode 0755 into a staged name and a
//     rename, which is the same one-step placement without either tool.
//   - The digest of what arrived is Go's SHA-256, as `sha256sum` printed it;
//     the VENDOR comparison is still a host tool, for the reason above.
//
// Usage:
//
//	bit-check install-rootless --prefix
//	bit-check install-rootless --fetch --url <u> --into <f> <disposition>
//	bit-check install-rootless --install --url <u> --into <f> --as <name> <disposition>
//
// `--fetch` prints the path of the verified file. `--install` does that and
// then places it executable at `<prefix>/bin/<name>`, printing that path.
//
// Exit codes: 0 done, 1 the fetch or the digest refused, 2 could not run.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type rootlessExit struct {
	code int
	msg  string
}

func (e rootlessExit) Error() string { return e.msg }

func rlRefuse(format string, a ...any) error {
	return rootlessExit{1, fmt.Sprintf(format, a...)}
}

func rlCannot(format string, a ...any) error {
	return rootlessExit{2, fmt.Sprintf(format, a...)}
}

const installRootlessUsage = `usage: bit-check install-rootless --prefix
       bit-check install-rootless --fetch --url <u> --into <f> <disposition>
       bit-check install-rootless --install --url <u> --into <f> --as <name> <disposition>

disposition: --sha256 <hex> | --sums <file> --sums-name <name> (either or both)
             | --digest-unpublished <why> | --from-resolution <record>
optional:    --report <file>

Exit codes: 0 done, 1 the fetch or the digest refused, 2 could not run.
`

func installRootless(args []string) int {
	err := runInstallRootless(args)
	if err == nil {
		return 0
	}
	if e, ok := err.(rootlessExit); ok {
		if e.msg != "" {
			fmt.Fprintf(os.Stderr, "install-rootless: %s\n", e.msg)
		}
		return e.code
	}
	fmt.Fprintf(os.Stderr, "install-rootless: %v\n", err)
	return 2
}

// The one derivation of the prefix.
//
// ⚠ THE HOME IS REQUIRED TO BE ABSOLUTE RATHER THAN MERELY SET. An empty or
// relative value composes into a path relative to wherever the caller stood,
// so two callers in two directories would install into two prefixes and the
// second would report the first's build as preexisting.
func rootlessPrefix() (string, error) {
	if p := os.Getenv("BIT_IDS_PREFIX"); p != "" {
		if !filepath.IsAbs(p) {
			return "", rlCannot("BIT_IDS_PREFIX is [%s], which is not an absolute path", p)
		}
		return p, nil
	}
	home, _ := os.UserHomeDir()
	if home == "" || !filepath.IsAbs(home) {
		return "", rlCannot("HOME is [%s], so there is no prefix this user is known to own", home)
	}
	return filepath.Join(home, ".local", "share", "bit-ids"), nil
}

// A file name and never a path: `--as ../../bin/sh` would place a build outside
// the prefix this exists to confine it to, and the caller is an adapter reading
// a value out of a document.
func isPlainName(s string) bool {
	return s != "." && s != ".." && !strings.ContainsAny(s, `/\`)
}

// The value of key= in a key=value record, as `sed -n 's/^key=//p'` read it.
func recordField(body []byte, key string) string {
	for _, line := range strings.Split(string(body), "\n") {
		if v, ok := strings.CutPrefix(line, key+"="); ok {
			return v
		}
	}
	return ""
}

var lowerHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func runInstallRootless(args []string) error {
	var mode, url, into, as, sha, sums, sumsName, unpublished, report, resolution string
	setMode := func(m string) error {
		if mode != "" {
			return rlCannot("--prefix, --fetch and --install are three modes; name one")
		}
		mode = m
		return nil
	}
	values := map[string]*string{
		"--url": &url, "--into": &into, "--as": &as, "--sha256": &sha, "--sums": &sums,
		"--sums-name": &sumsName, "--digest-unpublished": &unpublished,
		"--from-resolution": &resolution, "--report": &report,
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--prefix", "--fetch", "--install":
			if err := setMode(strings.TrimPrefix(a, "--")); err != nil {
				return err
			}
		case "-h", "--help":
			fmt.Print(installRootlessUsage)
			return nil
		default:
			dst, ok := values[a]
			if !ok {
				return rlCannot("unknown argument: %s", a)
			}
			if i+1 >= len(args) {
				if a == "--digest-unpublished" {
					return rlCannot("--digest-unpublished takes the measured reason")
				}
				return rlCannot("%s takes a value", a)
			}
			i++
			*dst = args[i]
		}
	}
	if mode == "" {
		return rlCannot("name a mode: --prefix, --fetch or --install")
	}

	prefix, err := rootlessPrefix()
	if err != nil {
		return err
	}
	if mode == "prefix" {
		fmt.Println(prefix)
		return nil
	}

	if url == "" {
		return rlCannot("--url is required")
	}
	if into == "" {
		return rlCannot("--into is required")
	}
	if mode == "install" && as == "" {
		return rlCannot("--install needs --as <name>")
	}
	if as != "" && !isPlainName(as) {
		return rlCannot("--as takes a file name, not a path: [%s]", as)
	}

	// -- the resolution, read here so four adapters do not each read it ------
	//
	// ⚠ A record naming a `digest_source` this does not know is could-not-run
	// rather than an install: a vocabulary that grew on one side and not the
	// other is how a route installs under a disposition nobody implemented.
	if resolution != "" {
		body, err := os.ReadFile(resolution)
		if err != nil || !isRegular(resolution) {
			return rlCannot("the resolution %s is not present", resolution)
		}
		if sha+sums+unpublished != "" {
			return rlCannot("--from-resolution carries the disposition; do not pass one beside it")
		}
		src := recordField(body, "digest_source")
		switch src {
		case "vendor-document", "source-listing", "both":
			if src != "source-listing" {
				sums = recordField(body, "digest_document")
				sumsName = recordField(body, "asset")
				if sums == "" || sumsName == "" {
					return rlCannot("the resolution says %s and names no document or asset", src)
				}
			}
			if src != "vendor-document" {
				// ⚠ The record spells `sha256:<hex>`, because a bare digest cannot
				// say what produced it; the prefix is stripped here.
				sha = strings.TrimPrefix(recordField(body, "asset_sha256"), "sha256:")
				if sha == "" {
					return rlCannot("the resolution says %s and names no asset_sha256", src)
				}
			}
		case "unpublished":
			unpublished = recordField(body, "digest_unpublished")
			if unpublished == "" {
				unpublished = "the resolution records no published digest and no reason"
			}
		case "":
			return rlCannot("the resolution %s names no digest_source", resolution)
		default:
			return rlCannot("the resolution names digest_source=[%s], which this installer does not know", src)
		}
	}

	// ⛔ AT LEAST ONE DISPOSITION, AND `--digest-unpublished` ALONE.
	n := 0
	for _, v := range []string{sha, sums, unpublished} {
		if v != "" {
			n++
		}
	}
	if n == 0 {
		return rlCannot("name a digest disposition: --sha256, --sums or --digest-unpublished")
	}
	if unpublished != "" && n != 1 {
		return rlCannot("--digest-unpublished says there is no digest; it cannot be given beside one")
	}
	if sha != "" {
		// ⚠ Checked as a shape: a truncated or upper-case value would compare
		// unequal against every artifact and read as a vendor whose bytes changed.
		if strings.Trim(sha, "0123456789abcdef") != "" {
			return rlCannot("--sha256 [%s] is not lowercase hex", sha)
		}
		if !lowerHex64.MatchString(sha) {
			return rlCannot("--sha256 [%s] is not 64 digits", sha)
		}
	}
	if sums != "" {
		if sumsName == "" {
			return rlCannot("--sums needs --sums-name, the name the document uses")
		}
		if !isRegular(sums) {
			return rlCannot("the checksums document %s is not present", sums)
		}
		if !isPlainName(sumsName) {
			return rlCannot("--sums-name takes a file name, not a path: [%s]", sumsName)
		}
	}
	curl, err := exec.LookPath("curl")
	if err != nil {
		return rlCannot("curl is not on this host")
	}
	verifier := sumsReader()
	if sums != "" && verifier == "" {
		return rlCannot("neither sha256sum nor certutil is on this host, so the vendor's document cannot be checked")
	}

	// -- the fetch -----------------------------------------------------------
	//
	// ⛔ BOUNDED IN BOTH DIRECTIONS, AND THE SPEED FLOOR IS THE ONE THAT
	// CATCHES A STALL. `--max-time` alone has to be large enough for a slow link
	// to finish a large artifact, which is large enough to sit in a dead
	// transfer for minutes; under 1 KB/s for a minute is stopped whatever its
	// size. ⭐ AND THE OUTPUT GOES TO FILES, so nothing this spawns holds a
	// descriptor the caller is reading.
	seconds := os.Getenv("BIT_IDS_FETCH_TIMEOUT")
	if seconds == "" {
		seconds = "300"
	}
	intoDir := filepath.Dir(into)
	if err := os.MkdirAll(intoDir, 0o755); err != nil {
		return rlCannot("cannot create %s", intoDir)
	}
	intoDir, err = filepath.Abs(intoDir)
	if err != nil {
		return rlCannot("cannot resolve %s", intoDir)
	}
	into = filepath.Join(intoDir, filepath.Base(into))
	logf, err := os.Create(into + ".fetch.log")
	if err != nil {
		return rlCannot("cannot write %s.fetch.log", into)
	}
	cmd := exec.Command(curl, "-fsSL", "--retry", "2", "--connect-timeout", "20",
		"--max-time", seconds, "--speed-limit", "1024", "--speed-time", "60", "-o", into, url)
	cmd.Stdout, cmd.Stderr = logf, logf
	ferr := cmd.Run()
	logf.Close()
	if ferr != nil {
		return rlRefuse("the artifact could not be fetched from %s", url)
	}
	if fi, err := os.Stat(into); err != nil || fi.Size() == 0 {
		return rlRefuse("the fetch left an empty file at %s", into)
	}
	arrived, err := fileSHA256(into)
	if err != nil {
		return rlCannot("no digest could be read for %s", into)
	}

	// -- identifying what arrived, BEFORE ANY MODE BIT IS SET -----------------
	digestSource := ""
	if sha != "" {
		if arrived != sha {
			os.Remove(into)
			return rlRefuse("the artifact from %s digests as %s and the source published %s", url, arrived, sha)
		}
		digestSource = "source-listing"
	}
	if sums != "" {
		// ⭐ THE VERIFIER RUNS IN A DIRECTORY HOLDING THE ARTIFACT UNDER THE NAME
		// THE DOCUMENT USES, because a checksums file names assets and this
		// workdir names files. ⛔ `--ignore-missing` lets a document covering
		// seven platforms verify the one asset present and does NOT pass over
		// nothing: a document naming only absent files exits 1 with `no file was
		// verified`, measured on 2026-09-17.
		verify := filepath.Join(intoDir, fmt.Sprintf(".verify-%d", os.Getpid()))
		os.RemoveAll(verify)
		if err := os.MkdirAll(verify, 0o755); err != nil {
			return rlCannot("cannot create %s", verify)
		}
		if err := copyFileMode(into, filepath.Join(verify, sumsName), 0o644); err != nil {
			os.RemoveAll(verify)
			return rlCannot("cannot stage %s for verification", into)
		}
		if err := copyFileMode(sums, filepath.Join(verify, "sums.txt"), 0o644); err != nil {
			os.RemoveAll(verify)
			return rlCannot("cannot stage %s for verification", sums)
		}
		rc, out := vendorVerify(verifier, verify, sumsName)
		os.RemoveAll(verify)
		os.WriteFile(into+".verify.log", out, 0o644)
		if rc != 0 {
			for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
				fmt.Fprintf(os.Stderr, "          %s\n", l)
			}
			os.Remove(into)
			return rlRefuse("%s refused %s against the vendor's document (exit %d)", verifierName(verifier), sumsName, rc)
		}
		if digestSource == "source-listing" {
			digestSource = "both"
		} else {
			digestSource = "vendor-document"
		}
	}
	if digestSource == "" {
		// ⚠ SAID OUT LOUD: identified, not verified, and a reader of the step log
		// is who needs to know which of the two happened.
		fmt.Fprintf(os.Stderr, "install-rootless: %s publishes no digest for this asset (%s); the artifact is recorded as %s and was not verified\n",
			url, unpublished, arrived)
		digestSource = "unpublished"
	}

	if report != "" {
		body := fmt.Sprintf("bit-ids/rootless-install/1\nurl=%s\nartifact=%s\nartifact_sha256=%s\ndigest_source=%s\ndigest_note=%s\nprefix=%s\n",
			url, into, arrived, digestSource, unpublished, prefix)
		if err := os.WriteFile(report, []byte(body), 0o644); err != nil {
			return rlCannot("the install report could not be written to %s", report)
		}
	}

	if mode == "fetch" {
		fmt.Println(into)
		return nil
	}

	// -- placing it ------------------------------------------------------------
	bin := filepath.Join(prefix, "bin")
	// ⚠ What creating the prefix said goes beside the artifact, as the shell
	// half's `2>"$INTO.prefix.log"` put it, empty when it said nothing.
	merr := os.MkdirAll(bin, 0o755)
	said := ""
	if merr != nil {
		said = merr.Error() + "\n"
	}
	os.WriteFile(into+".prefix.log", []byte(said), 0o644)
	if merr != nil {
		fmt.Fprintf(os.Stderr, "          %v\n", merr)
		return rlRefuse("this user cannot create %s; a rootless install writes nowhere else", bin)
	}
	probe := filepath.Join(bin, fmt.Sprintf(".write-%d", os.Getpid()))
	if f, err := os.Create(probe); err != nil {
		return rlRefuse("%s is not writable by this user; a rootless install writes nowhere else", bin)
	} else {
		f.Close()
		os.Remove(probe)
	}
	// ⚠ INTO PLACE AS ONE STEP. A copy that is interrupted leaves a truncated
	// executable an adapter's `binary()` will find; a rename within one
	// directory either happened or did not, and the staged copy is written at
	// its final mode, so it is never both present and non-executable.
	staged := filepath.Join(bin, "."+as+".new")
	if err := copyFileMode(into, staged, 0o755); err != nil {
		os.Remove(staged)
		return rlRefuse("the artifact could not be staged in %s", bin)
	}
	final := filepath.Join(bin, as)
	if err := os.Rename(staged, final); err != nil {
		os.Remove(staged)
		return rlRefuse("the artifact could not be installed as %s", final)
	}
	if fi, err := os.Stat(final); err != nil || (runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0) {
		return rlRefuse("the install reported success and left no executable %s in %s", as, bin)
	}
	fmt.Println(final)
	fmt.Fprintf(os.Stderr, "install-rootless: %s -> %s (sha256:%s, %s)\n", url, final, arrived, digestSource)
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFileMode(from, to string, mode os.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(to, mode)
}

func verifierName(v string) string {
	if v == "certutil" {
		return "certutil"
	}
	return "sha256sum -c"
}

// The vendor's document, asked of a host tool: `sha256sum -c --ignore-missing`
// where it exists, and on Windows the row naming the asset, hashed by
// `certutil`, with the same refusal over a document that names nothing present.
func vendorVerify(verifier, dir, name string) (int, []byte) {
	if verifier == "sha256sum" {
		cmd := exec.Command("sha256sum", "-c", "--ignore-missing", "sums.txt")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ExitCode(), out
			}
			return 2, out
		}
		return 0, out
	}
	f, err := os.Open(filepath.Join(dir, "sums.txt"))
	if err != nil {
		return 2, []byte(err.Error() + "\n")
	}
	defer f.Close()
	row := regexp.MustCompile(`^([0-9a-fA-F]{64})[ \t]+\*?(.*)$`)
	want := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := row.FindStringSubmatch(strings.TrimRight(sc.Text(), "\r")); m != nil && m[2] == name {
			want = strings.ToLower(m[1])
			break
		}
	}
	if want == "" {
		return 1, []byte("no file was verified\n")
	}
	var out bytes.Buffer
	cmd := exec.Command("certutil", "-hashfile", filepath.Join(dir, name), "SHA256")
	cmd.Stdout = &out
	if cmd.Run() != nil {
		return 2, out.Bytes()
	}
	for _, l := range strings.Split(out.String(), "\n") {
		l = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(l), " ", ""))
		if lowerHex64.MatchString(l) {
			if l == want {
				return 0, []byte(name + ": OK\n")
			}
			return 1, []byte(name + ": FAILED\n")
		}
	}
	return 2, out.Bytes()
}
