// repo.go - the scope every check shares, derived once and in one place.
//
// ⛔ THE SCOPE IS ASKED OF git, NEVER RE-DERIVED. The shell halves this file
// replaces each ran `git ls-files` plus `git ls-files --others
// --exclude-standard`, and a Go reimplementation walking the filesystem with
// its own idea of .gitignore would be a SECOND answer to what is in the tree.
// Where two answers exist they drift, and the direction they drift is the one
// that keeps a check green. So the derivation shells out to the same commands.
//
// ⛔ AND IT IS PINNED TO THE REPOSITORY ROOT. `git ls-files` is relative to the
// process working directory, so a guard invoked from a subdirectory silently
// reports on a smaller tree and calls it clean. Measured in the shell half this
// replaces: run from the root it saw 1071 files and from one package directory
// 391, and a literal NUL rode through two handoffs that each reported green.
package main

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// repo is the tree a check reads, resolved once per process.
type repo struct {
	root string
}

// openRepo resolves the repository root, or reports why it could not.
//
// ⚠ Every failure here is a COULD NOT RUN and never a refusal. A check that
// cannot find a tree has verified nothing about one, and reporting that as a
// pass is the defect the whole 0/1/2 vocabulary exists to keep separate.
func openRepo() (*repo, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, errCannotRun("git not found")
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, errCannotRun("not a git repository")
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return nil, errCannotRun("not a git repository")
	}
	if err := os.Chdir(root); err != nil {
		return nil, errCannotRun("cannot enter " + root)
	}
	return &repo{root: root}, nil
}

// files lists tracked plus untracked-but-not-ignored paths, sorted and unique.
//
// ⛔ TRACKED ALONE IS NOT ENOUGH. `git ls-files` cannot see a file that has
// never been staged, which is exactly when a new file is likeliest to carry a
// defect, and it is what the next `git add -A` would take.
func (r *repo) files() []string {
	seen := map[string]bool{}
	for _, args := range [][]string{
		{"ls-files", "-z"},
		{"ls-files", "-zo", "--exclude-standard"},
	} {
		// ⚠ `-z` for `gitLines`' reason: a quoted name is a path no file has.
		out, err := exec.Command("git", args...).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\x00") {
			if line != "" {
				seen[line] = true
			}
		}
	}
	list := make([]string, 0, len(seen))
	for f := range seen {
		list = append(list, f)
	}
	sort.Strings(list)
	return list
}

// matching returns the files in scope for a pattern, skipping any that is
// tracked but no longer on disk.
//
// ⚠ A tracked-but-deleted path is skipped rather than reported. git says so
// itself, and a check that refused it would go red on every half-finished
// rename.
func (r *repo) matching(re *regexp.Regexp, exclude *regexp.Regexp) []string {
	var out []string
	for _, f := range r.files() {
		if !re.MatchString(f) {
			continue
		}
		if exclude != nil && exclude.MatchString(f) {
			continue
		}
		st, err := os.Stat(f)
		if err != nil || st.IsDir() {
			continue
		}
		out = append(out, f)
	}
	return out
}

// textRe is the set of extensions asserted to be TEXT.
//
// ⚠ BINARIES ARE OUT OF SCOPE BY CONSTRUCTION, not by an allowlist. This says
// what IS text; an "allowlist of binaries that are fine" is the kind of list
// that quietly absorbs a real finding.
var textRe = regexp.MustCompile(`\.(ts|tsx|js|mjs|cjs|jsx|json|md|sql|css|scss|html|toml|yaml|yml|sh|ps1|py|rs|go|c|h|cpp|hpp|java|rb|php|txt|cfg|ini|conf|env\.example)$`)

// markerTextRe is the same set WITHOUT `env.example`.
//
// ⛔ THE TWO LISTS DIFFER AND THAT DIFFERENCE IS INHERITED RATHER THAN INVENTED.
// The shell halves carried two spellings, and a port that unified them would be
// changing a verdict while claiming only to change a language. Either list may
// be argued with; neither may be changed here.
var markerTextRe = regexp.MustCompile(`\.(ts|tsx|js|mjs|cjs|jsx|json|md|sql|css|scss|html|toml|yaml|yml|sh|ps1|py|rs|go|c|h|cpp|hpp|java|rb|php|txt|cfg|ini|conf)$`)

// licensesRe exempts the canonical SPDX texts.
//
// ⛔ Those are somebody else's bytes. GPL-3.0 and LGPL-3.0 carry typographic
// quotes and a copyright sign, and four of the twelve must never have their
// notice altered at all. A check that asked anybody to edit these would be
// asking for a corruption.
var licensesRe = regexp.MustCompile(`^LICENSES/.*\.txt$`)

// splitLines cuts a file into records the way awk does: on "\n", with no record
// after a trailing newline, and with any "\r" left exactly where it was.
//
// ⛔ THE CARRIAGE RETURN IS NOT STRIPPED. A .ps1 keeps CRLF in this tree, and
// awk's `[^ \t]` counts a bare "\r" as a non-blank line. Stripping it here would
// change the denominator of every density in the PowerShell half of the tree,
// which is a verdict changed by a port.
func splitLines(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	lines := bytes.Split(b, []byte("\n"))
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}
