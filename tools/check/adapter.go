// adapter - the one door to a capture adapter, Go or shell.
//
// CI-11. An adapter is the only code in this repository that knows how one
// product is installed, asked its version, started and stopped; its contract is
// scripts/capture/adapters/README.md and it does not change here. ⭐ What this
// adds is a second FORM of the same contract: an adapter written in Go and run
// as `bit-check adapter <name> <subcommand> [args...]`, so a Windows runner,
// which has no `sh`, can run one.
//
// ⛔ AN ADAPTER STAYS A SEPARATE PROCESS IN BOTH FORMS. Every caller bounds an
// adapter call, gives it the null device for stdin and FILES for its output,
// because an adapter runs a product this project did not write, and a product
// that waits on a prompt or leaves a process holding a pipe would otherwise hang
// the caller. A Go adapter called in process would escape every one of those
// bounds, so it is run as this binary in a child process, exactly as an `sh`
// adapter is run as `sh <path>`.
//
// -- the reference ---------------------------------------------------------
//
// `--adapter` names one of two things, and the form says which:
//
//	a registered name, with no path separator   a Go adapter, run as this binary
//	anything else                               an `sh` adapter file, run as `sh <path>`
//
// ⚠ The harnesses' stub adapters are `sh` files written into scratch trees,
// which is why the file form stays: a stub is a test seam, not a product.
//
// ⛔ THE DIGEST A RECORD CARRIES IS OF WHAT RAN. For an `sh` adapter that is the
// file; for a Go adapter it is this binary, which is the code that ran.
//
// Usage:
//
//	bit-check adapter <name> describe
//	bit-check adapter <name> install <route> <workdir>
//	bit-check adapter <name> version
//	bit-check adapter <name> start <torrent> <workdir> <peer-port>
//	bit-check adapter <name> stop <workdir>
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// goAdapter implements the five subcommands for one target, taking the
// subcommand's own arguments and answering the contract's exit code. It prints
// to stdout and stderr as the `sh` form does.
type goAdapter func(sub string, args []string) int

// goAdapters is filled by each adapter file's init. ⚠ A platform may register
// a target the other does not, and one target's two platform halves register
// the same name from build-tagged files.
var goAdapters = map[string]goAdapter{}

func registerAdapter(name string, a goAdapter) {
	if _, dup := goAdapters[name]; dup {
		panic("two adapters registered as " + name)
	}
	goAdapters[name] = a
}

func adapterNames() []string {
	var out []string
	for n := range goAdapters {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// runAdapterCmd is `bit-check adapter <name> <subcommand> [args...]`.
func runAdapterCmd(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "adapter: usage: bit-check adapter <name> <subcommand> [args...]; adapters: %v\n", adapterNames())
		return 2
	}
	a, ok := goAdapters[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "adapter: no adapter named %s on this platform; adapters: %v\n", args[0], adapterNames())
		return 2
	}
	switch args[1] {
	case "describe", "install", "version", "start", "stop":
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand: %s\n", args[0], args[1])
		return 2
	}
	return a(args[1], args[2:])
}

// isGoAdapterRef says whether an `--adapter` value names a Go adapter.
func isGoAdapterRef(ref string) bool {
	if ref == "" || strings.ContainsAny(ref, `/\`) {
		return false
	}
	_, ok := goAdapters[ref]
	return ok
}

// adapterPresent is the callers' `[ -f "$ADAPTER" ]`, for either form.
func adapterPresent(ref string) bool {
	return isGoAdapterRef(ref) || isRegular(ref)
}

// adapterArgv is the program and leading arguments that run one subcommand of
// the adapter: `sh <path> <sub>...` or `<this binary> adapter <name> <sub>...`.
func adapterArgv(ref string, sub ...string) (string, []string) {
	if isGoAdapterRef(ref) {
		if self, err := os.Executable(); err == nil {
			return self, append([]string{"adapter", ref}, sub...)
		}
	}
	return "sh", append([]string{ref}, sub...)
}

// adapterDigest is the digest of the code an adapter call runs.
func adapterDigest(ref string) string {
	path := ref
	if isGoAdapterRef(ref) {
		if self, err := os.Executable(); err == nil {
			path = self
		}
	}
	d, _ := fileSHA256(path)
	return d
}

// writePackageIndex is how a package route says what its index named, for
// `install-client` to record and `assemble-capture` to take as the route's
// source identity: the index, the package's name in it, and the version the
// index gave. ⚠ The version is the index's statement; the build's own answer is
// `version`'s.
//
// ⭐ The origin is the URL the index pins the installer at, which is where the
// route's bytes came from.
func writePackageIndex(workdir, index, pkg, version, origin string) error {
	return writePackageIndexVia(workdir, index, pkg, version, origin, "")
}

// writePackageIndexVia is writePackageIndex for a route whose package manager
// resolved the build and did not fetch it, naming what did. ⚠ Only `deluge`'s
// Windows route: winget's index pins an `http` URL, so the route fetches the
// same bytes over https itself. The origin is then the URL really fetched.
func writePackageIndexVia(workdir, index, pkg, version, origin, delivery string) error {
	body := fmt.Sprintf("index=%s\npackage=%s\nversion=%s\norigin=%s\n", index, pkg, version, origin)
	if delivery != "" {
		body += "delivery=" + delivery + "\n"
	}
	return os.WriteFile(filepath.Join(workdir, "package-index.txt"), []byte(body), 0o644)
}

// wingetInstallerURL is the one `Installer Url:` `winget show` prints under
// `Installer:`. ⛔ Two different ones is a refusal: the route would not know
// which the install fetched.
func wingetInstallerURL(show string) (string, error) {
	found := ""
	for _, l := range strings.Split(show, "\n") {
		l = strings.TrimSpace(strings.TrimRight(l, "\r"))
		v, ok := strings.CutPrefix(l, "Installer Url:")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if found != "" && found != v {
			return "", fmt.Errorf("winget's index named two installer URLs, %s and %s", found, v)
		}
		found = v
	}
	if !strings.HasPrefix(found, "https://") {
		return "", fmt.Errorf("winget's index named no https installer URL: [%s]", found)
	}
	return found, nil
}
