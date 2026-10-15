// fetch-releases - retrieve one release listing and keep the exact bytes.
//
// CI-11. It was scripts/acquisition/fetch-releases.sh, a leaf of the capture
// path that has to run on Windows too.
//
// ⭐ IT FETCHES AND NOTHING ELSE. It does not parse, sort, filter or decide:
// shell, and now this binary, orchestrates, and Rust parses. A fetch that also
// picked the newest version would be a second implementation of the rule in
// crates/bit-ids/src/resolution.rs, with no test behind it and no record of
// what it read. The bytes land in a file and `resolve-stable` reads that file,
// which is what makes the digest in a resolution mean something: it is of what
// arrived, not of what a parser reconstructed afterwards.
//
// -- ⛔ THE ROUTE IS NOT A PREFERENCE ----------------------------------------
//
// docs/AGENTS.md rule 8: GitHub is read through
// https://api.gh.pkgforge.dev/<PATH>. ⚠ An unauthenticated api.github.com is
// rate limited per address and answers 403 from a shared runner, so falling
// back to it silently would work locally and fail in CI. The route that
// answered is printed, because a resolution that cannot say where its bytes
// came from is not replayable.
//
// ⛔ `curl`, NOT GO'S OWN CLIENT, AND THAT WAS MEASURED. Ported first onto
// net/http, the live route answered 420 where `curl` got 200 for the same URL
// on 2026-10-15: the gateway answers user agents that name curl or wget and
// refuses the rest, Go's default among them. Sending a user agent that claimed
// to be curl would be this binary misdescribing itself to a service. So the
// fetch is the same `curl -sS -L --max-time 60` the shell half ran, and
// `curl.exe` ships with Windows, so the leaf still runs on both platforms.
//
// Usage:
//
//	bit-check fetch-releases <owner/repo> <output-file>
//	bit-check fetch-releases --url <https url> <output-file>
//
// Prints the URL that answered on stdout. Exit codes: 0 fetched, 1 the source
// refused or answered with nothing, 2 could not run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const fetchReleasesUsage = `usage: bit-check fetch-releases <owner/repo> <output-file>
       bit-check fetch-releases --url <https url> <output-file>

Retrieves one release listing and keeps the exact bytes, with the retrieval
instant beside them in <output-file>.fetched-at. Prints the URL that answered.

Exit codes: 0 fetched, 1 the source refused or answered with nothing, 2 could
not run.
`

const releasesPerPage = 100

func fetchReleases(args []string) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(fetchReleasesUsage)
		return 0
	}
	var url, out string
	if len(args) > 0 && args[0] == "--url" {
		if len(args) != 3 {
			fmt.Fprint(os.Stderr, fetchReleasesUsage)
			return 2
		}
		url, out = args[1], args[2]
	} else {
		if len(args) != 2 {
			fmt.Fprint(os.Stderr, fetchReleasesUsage)
			return 2
		}
		if !strings.Contains(args[0], "/") {
			fmt.Fprintf(os.Stderr, "fetch-releases: expected owner/repo, got %s\n", args[0])
			return 2
		}
		url = fmt.Sprintf("https://api.gh.pkgforge.dev/repos/%s/releases?per_page=%d", args[0], releasesPerPage)
		out = args[1]
	}
	if !strings.HasPrefix(url, "https://") {
		fmt.Fprintf(os.Stderr, "fetch-releases: a retrieval location is https: %s\n", url)
		return 2
	}
	if dir := filepath.Dir(out); !isDir(dir) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "fetch-releases: cannot create %s\n", dir)
			return 2
		}
	}

	// ⛔ THE STATUS IS READ FROM THE RESPONSE AND THE BODY GOES TO A FILE, and a
	// body that is not a 200, or is empty, is removed rather than left for a
	// resolver to read as a listing.
	refused := func(format string, a ...any) int {
		fmt.Fprintf(os.Stderr, "fetch-releases: "+format+"\n", a...)
		os.Remove(out)
		return 1
	}
	curl, err := exec.LookPath("curl")
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch-releases: curl not found")
		return 2
	}
	// ⛔ THE STATUS IS ASKED OF curl AND ITS EXIT CODE IS READ FROM THE PROCESS,
	// never through a pipe: `curl | tee` would report tee's status, so a 404
	// body would be written and reported as a successful fetch.
	cmd := exec.Command(curl, "-sS", "-L", "--max-time", "60", "-o", out, "-w", "%{http_code}", url)
	cmd.Stderr = os.Stderr
	status, err := cmd.Output()
	if err != nil {
		code := -1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return refused("curl exited %d for %s", code, url)
	}
	if string(status) != "200" {
		return refused("%s answered %s", url, status)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		return refused("%s answered with an empty body", url)
	}

	// ⛔ THE RETRIEVAL INSTANT IS WRITTEN BY THE FETCH, not inferred later from
	// the file's modification time. An mtime survives a copy, an archive restore
	// and a checkout, so a resolution reading one would publish a retrieval time
	// that is not one. The sidecar is what the resolver reads.
	stamp := time.Now().UTC().Format("2006-01-02T15:04:05Z") + "\n"
	if err := os.WriteFile(out+".fetched-at", []byte(stamp), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "fetch-releases: cannot record the retrieval instant")
		os.Remove(out)
		return 2
	}
	fmt.Println(url)
	return 0
}
