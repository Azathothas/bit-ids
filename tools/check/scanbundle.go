// scan-bundle - rule 12 over a directory that is about to be uploaded.
//
// CI-11. The first leaf of the capture path to leave `sh`, so a client capture
// can run on Windows: this was scripts/capture/scan-bundle.sh, and the reasons
// it gave are kept here with the rules.
//
// -- ⛔ ONE HOME FOR WHAT A CREDENTIAL LOOKS LIKE ---------------------------
//
// `capture-client` and `install-client` each carried these two rules, written
// to be identical, and two spellings drift. They call this, and so does the
// workflow step that scans a REFUSED capture before its evidence is uploaded.
//
// ⛔ A CAPTURE'S OUTPUT DIRECTORY IS ITS EVIDENCE BUNDLE, AND A PRODUCT THIS
// PROJECT DID NOT WRITE HAS BEEN WRITING INTO IT. Measured on `capture-client`
// run 11: `aria2-next` wrote its per-run RPC token to `<workdir>/rpc-token` and
// the runner packed that directory, so the token shipped inside the upload.
// `check-no-secrets` structurally cannot see it: the bundle is not in the tree
// it scans. And an adapter's own cleanup is a gate on one path, which a refused
// capture, whose `stop` may never have run, is exactly the run that skips.
//
// ⛔ TWO RULES, BECAUSE THE TWO FAILURES LOOK NOTHING ALIKE. A file whose NAME
// says it holds a credential is the run's own state; a secret-shaped MEMBER
// inside a document the product answered with is the product handing one
// back, which no filename rule reaches: an options dump names its own
// `rpc-secret`. ⚠ BOTH ARE ALWAYS ASKED and both are reported when both fire,
// so a caller does not spend a dispatch learning of the second.
//
// ⚠ A WORD BOUNDARY, NOT A SUBSTRING. `tokenizer` is not a token, and a rule
// that refused it would be one somebody switches off.
//
// -- ⚠ THE SCOPE IS THE CALLER'S, BECAUSE IT IS WHAT THE CALLER SHIPS -------
//
// `--top-level` reads the directory's own files and nothing below them. An
// install workdir is uploaded as `install-<route>/*.log` and `*.err`, globs
// that do not descend, and a source route clones a tree whose vendored test
// fixtures are credential-NAMED and public: run 15 was refused over
// `src/third_party/openssl/apps/server.pem`, which nothing shipped. ⛔ A
// capture bundle is uploaded whole, so it is scanned whole.
//
// -- ⚠ WHERE THIS DIFFERS FROM THE SHELL HALF, ON PURPOSE -------------------
//
//   - ⛔ A DIRECTORY THAT CANNOT BE READ IS COULD-NOT-RUN. The shell half sent
//     `find`'s errors to /dev/null, so a subdirectory it could not open was
//     skipped and the scan answered clean over files it never read. A scan
//     that did not read a file has not established it is safe to ship.
//   - The paths are reported in lexical order rather than in `find`'s
//     directory order, which no caller read and no two hosts agree on.
//   - A file is binary, and skipped by the value rule as `grep -I` skipped
//     it, when it holds a NUL byte: the C locale's reading, which scans more
//     than a UTF-8 locale's, where an encoding error also made a file binary.
//
// Usage:
//
//	bit-check scan-bundle [--top-level] <dir>
//
// Exit codes: 0 nothing rule 12 forbids, 1 something would ship that it
// forbids, 2 could not run. The paths found are printed on stderr and the
// values never are, because a guard that echoed the secret it found would put
// it in the run log.
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

// The NAME rule. Matched against the lower-cased base name, as `awk`'s
// `tolower($NF)` did.
var credentialNames = []*regexp.Regexp{
	regexp.MustCompile(`(^|[^a-z])(token|secret|passwd|password|credential)([^a-z]|$)`),
	regexp.MustCompile(`\.(pem|key|p12|pfx|jks|keystore)$`),
	regexp.MustCompile(`^id_(rsa|ed25519|ecdsa)`),
	regexp.MustCompile(`^\.env`),
}

// The VALUE rule: a JSON member, or a command-line option, whose name says
// secret, token or password and whose value is eight characters or more.
// ⚠ ASKED OF ONE LINE AT A TIME, because `grep` is line-based and these
// classes never matched a newline there; Go's `[^"]` would match one.
var secretValue = regexp.MustCompile(
	`"[A-Za-z0-9_-]*(secret|token|password)[A-Za-z0-9_-]*"[[:space:]]*:[[:space:]]*"[^"]{8,}"` +
		`|--[A-Za-z0-9-]*(secret|token|password)[A-Za-z0-9-]*=[^[:space:]"]{8,}`)

const scanUsage = `usage: bit-check scan-bundle [--top-level] <dir>

Rule 12 over a directory that is about to be uploaded: a file whose name says
it holds a credential, or a text file carrying a secret-shaped value.
--top-level reads the directory's own files and nothing below them.

Exit codes: 0 nothing rule 12 forbids, 1 something would ship that it forbids,
2 could not run.
`

func scanBundle(args []string) int {
	topLevel := false
	dir := ""
	for _, a := range args {
		switch {
		case a == "--top-level":
			topLevel = true
		case a == "-h" || a == "--help":
			fmt.Print(scanUsage)
			return 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "scan-bundle: unknown argument: %s\n", a)
			return 2
		default:
			if dir != "" {
				fmt.Fprintln(os.Stderr, "scan-bundle: one directory, not two")
				return 2
			}
			dir = a
		}
	}
	if dir == "" {
		fmt.Fprintln(os.Stderr, "scan-bundle: a directory is required")
		return 2
	}
	// ⛔ A DIRECTORY THAT IS NOT THERE IS COULD-NOT-RUN, NOT CLEAN. A scan over
	// nothing finds nothing, and a caller reading 0 would upload whatever a path
	// it mistyped did not name.
	if !isDir(dir) {
		fmt.Fprintf(os.Stderr, "scan-bundle: %s is not a directory\n", dir)
		return 2
	}

	files, err := scanFiles(dir, topLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan-bundle: %s could not be read whole, so it is not established clean: %v\n", dir, err)
		return 2
	}

	var named, valued []string
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f))
		for _, re := range credentialNames {
			if re.MatchString(base) {
				named = append(named, f)
				break
			}
		}
		body, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan-bundle: %s could not be read: %v\n", f, err)
			return 2
		}
		// ⚠ BINARY IS SKIPPED, so a generated torrent and any artifact bytes are
		// not scanned as text.
		if bytes.IndexByte(body, 0) >= 0 {
			continue
		}
		for _, line := range bytes.Split(body, []byte{'\n'}) {
			if secretValue.Match(line) {
				valued = append(valued, f)
				break
			}
		}
	}

	rc := 0
	if len(named) > 0 {
		for _, f := range named {
			fmt.Fprintf(os.Stderr, "          %s\n", f)
		}
		fmt.Fprintf(os.Stderr, "scan-bundle: %s holds a file whose name says it holds a credential; rule 12\n", dir)
		rc = 1
	}
	if len(valued) > 0 {
		for _, f := range valued {
			fmt.Fprintf(os.Stderr, "          %s\n", f)
		}
		fmt.Fprintf(os.Stderr, "scan-bundle: %s holds a document carrying a secret-shaped value; rule 12\n", dir)
		rc = 1
	}
	return rc
}

// Every regular file under dir, or directly in it with topLevel, spelled as
// `find dir` spelled them. ⚠ REGULAR ONLY, as `-type f` was: a symbolic link
// is not followed and a named pipe is never opened, so a planted link cannot
// point the scan at the host and a fifo cannot hang it.
func scanFiles(dir string, topLevel bool) ([]string, error) {
	var out []string
	root := strings.TrimSuffix(dir, "/")
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if topLevel && p != dir {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			rel, rerr := filepath.Rel(dir, p)
			if rerr != nil {
				return rerr
			}
			out = append(out, root+"/"+filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}
