// anacrolix - what both halves of the `anacrolix-torrent` adapter decide, and
// the install both halves run, with no platform in either, so any host can ask
// the decisions through `bit-check anacrolix-torrent --selftest`.
//
// ENGINE-02. The adapters are adapter_anacrolix-torrent.go (Linux) and
// adapter_anacrolix-torrent_windows.go; this file registers nothing.
//
// ⛔ A LIBRARY, SO THE PRODUCT IS A HARNESS THIS TREE COMMITS. A Go module
// version is not an installable client: it is source an embedding program
// links. The harness is tools/harness/anacrolix-torrent/, the smallest program
// that runs the library's NewDefaultClientConfig() on one torrent, in a module
// of its own whose go.mod and go.sum pin the library and every dependency.
// `describe` declares `target_kind=library`, as rqbit's does, and the record
// files a `library` target: no schema field is added.
//
// -- ⭐ WHAT UPSTREAM PUBLISHES, read on 2026-10-22 -----------------------------
//
// No GitHub release objects at all: `releases` is empty. The tags are the
// releases, and proxy.golang.org lists them as module versions; `@latest` is
// `v1.61.0`, 2025-12-17T07:39:03Z, origin `refs/tags/v1.61.0` at commit
// `486bcbe7...`. sum.golang.org holds `h1:vxo+B4Sw...` for it.
//
// -- ⭐ THE TWO ROUTES, ONE MODULE VERSION, ONE HARNESS SOURCE ------------------
//
// ⛔ `package`: the Go module proxy. `GOPROXY=https://proxy.golang.org`, the
// harness copied WITHOUT its go.sum and `-mod=mod`, so every module the build
// needs is fetched from the proxy and checked by the go command against
// sum.golang.org; the go.sum that build writes must then be byte-equal to the
// committed one. The proxy's `@latest` is the resolver and must name the pin.
// ⛔ `source`: the tag, from GitHub. `GOPROXY=direct`, so the go command
// clones the library from https://github.com/anacrolix/torrent at the tag the
// source lane resolved, AND every dependency from its own origin, builds each
// module's zip from the tree and checks it against the committed go.sum. No
// module comes from the proxy. ⚠ The toolchain is the one input both share:
// `GOTOOLCHAIN` names the pin, which the package route takes from
// proxy.golang.org and the source route from go.dev/dl/mod, both checked
// against sum.golang.org.
//
// ⭐ Measured on 2026-10-22, both routes built the same 29464958-byte
// executable, sha256 `3d684a21...`, and the same build information.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// The pins and the scheme, one declaration for both halves.
//
// ⛔ THE PIN IS THE HARNESS'S go.mod AND go.sum, and these constants say what
// they must hold: `check-bitcheck` and every `install` hold the files to them
// with anacrolixPinRefusal. When stable moves, the harness is re-pinned in one
// change and the record says so; a route never upgrades the library itself.
const (
	anacrolixTarget = "anacrolix-torrent"
	anacrolixRepo   = "anacrolix/torrent"
	anacrolixModule = "github.com/anacrolix/torrent"
	// anacrolixPin is the module version the harness pins, the newest stable
	// tag on 2026-10-22.
	anacrolixPin = "v1.61.0"
	// anacrolixPinSum and anacrolixPinModSum are sum.golang.org's statement
	// of that version, read through `lookup/` on 2026-10-22.
	anacrolixPinSum    = "h1:vxo+B4SwnoP5AQWbhvnTYIaTgPSX+llYUVuQVsN4Jg8="
	anacrolixPinModSum = "h1:yKUKuZSSDdyOsCbuH+rDOpswl/g546gICapdrU7aUmQ="
	// anacrolixPinnedCommit is the commit `refs/tags/v1.61.0` names, as the proxy's
	// origin record and the tags listing both state it.
	anacrolixPinnedCommit = "486bcbe758e0cff0f750fcee9c26268a11250d77"
	// anacrolixToolchain is the Go both routes build with: the newest stable
	// release on 2026-10-22, from go.dev/dl. ⚠ Named, because a runner's own Go
	// is whatever its image carries.
	anacrolixToolchain = "go1.27.3"
	// anacrolixHarnessDir is the harness's directory in this tree.
	anacrolixHarnessDir    = "tools/harness/anacrolix-torrent"
	anacrolixHarnessModule = "bit-ids/harness/anacrolix-torrent"
	anacrolixExeName       = "anacrolix-torrent-harness"
	anacrolixProxy         = "https://proxy.golang.org"
	// anacrolixProxyIndex is the index the record names for the package route,
	// the host the go command fetched from.
	anacrolixProxyIndex = "proxy.golang.org"
	anacrolixSumDB      = "sum.golang.org"
	// anacrolixDelivery names what delivered the package route's bytes: the go
	// command fetched every module from the proxy and compiled them here.
	anacrolixDelivery = "go-module-proxy-local-build"
	// anacrolixDefaultBep20 is the library's own peer-ID prefix at the pin,
	// `version.DefaultBep20Prefix`. ⚠ A self-report the harness prints and
	// `start` holds equal, so nothing set it; the observer's reading is what a
	// record carries.
	anacrolixDefaultBep20 = "-GT0003-"
)

// anacrolixDescribe is everything either half's `describe` prints after
// `binary=`.
//
// ⚠ NO `release_asset` AND NO DIGEST DISPOSITION, as `deluge`'s: the repository
// publishes tags and no release objects, so `resolve-release` has nothing to
// fetch and `resolve-source` reads the tags with this scheme.
func anacrolixDescribe() string {
	var b strings.Builder
	b.WriteString("target_kind=library\n")
	fmt.Fprintf(&b, "release_repo=%s\n", anacrolixRepo)
	b.WriteString("release_tag_prefix=v\n")
	b.WriteString("release_min_components=3\n")
	b.WriteString("release_max_components=3\n")
	b.WriteString("release_floor_major=-\n")
	// ⚠ Both routes compile the harness.
	b.WriteString("compiles=package,source\n")
	b.WriteString("routes=package,source\n")
	return b.String()
}

// anacrolixPackageFormat is how each route delivers the build. ⭐ One form per
// platform: `CGO_ENABLED=0` makes a static executable on Linux, and both routes
// build the same one.
func anacrolixPackageFormat(windows bool, route string) (string, bool) {
	if route != "package" && route != "source" {
		return "", false
	}
	if windows {
		return "pe-binary", true
	}
	return "elf-binary-static", true
}

// anacrolixExe is the executable's file name on a platform.
func anacrolixExe(windows bool) string {
	if windows {
		return anacrolixExeName + ".exe"
	}
	return anacrolixExeName
}

// anacrolixPinRefusal is what is wrong with a harness's go.mod and go.sum, or
// "". ⛔ The library at the pin, required directly and never replaced; the
// toolchain line at the pin; and go.sum carrying sum.golang.org's two lines.
func anacrolixPinRefusal(gomod, gosum string) string {
	lines := strings.Split(strings.ReplaceAll(gomod, "\r", ""), "\n")
	if len(lines) == 0 || lines[0] != "module "+anacrolixHarnessModule {
		return "go.mod does not declare module " + anacrolixHarnessModule
	}
	direct, toolchain := 0, 0
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "replace") || strings.HasPrefix(t, "exclude") || strings.HasPrefix(t, "retract") {
			return "go.mod carries a `" + strings.Fields(t)[0] + "` directive, so the build would not be the pinned modules"
		}
		if strings.HasPrefix(t, "toolchain ") {
			toolchain++
			if t != "toolchain "+anacrolixToolchain {
				return "go.mod names [" + t + "], not toolchain " + anacrolixToolchain
			}
		}
		f := strings.Fields(strings.TrimPrefix(t, "require "))
		if len(f) >= 2 && f[0] == anacrolixModule {
			if f[1] != anacrolixPin {
				return fmt.Sprintf("go.mod requires %s %s, not the pin %s", anacrolixModule, f[1], anacrolixPin)
			}
			if strings.Contains(t, "// indirect") {
				return "go.mod requires " + anacrolixModule + " as indirect, and the harness imports it"
			}
			direct++
		}
	}
	if direct != 1 {
		return fmt.Sprintf("go.mod requires %s %d times, not once", anacrolixModule, direct)
	}
	if toolchain != 1 {
		return fmt.Sprintf("go.mod names a toolchain %d times, not once", toolchain)
	}
	want := map[string]bool{
		anacrolixModule + " " + anacrolixPin + " " + anacrolixPinSum:           false,
		anacrolixModule + " " + anacrolixPin + "/go.mod " + anacrolixPinModSum: false,
	}
	for _, l := range strings.Split(strings.ReplaceAll(gosum, "\r", ""), "\n") {
		f := strings.Fields(l)
		if len(f) == 3 && f[0] == anacrolixModule {
			key := strings.Join(f, " ")
			if _, ok := want[key]; !ok {
				return "go.sum states another hash or version of " + anacrolixModule + ": " + key
			}
			want[key] = true
		}
	}
	for k, seen := range want {
		if !seen {
			return "go.sum does not carry sum.golang.org's line [" + k + "]"
		}
	}
	return ""
}

// anacrolixGoEnv is the environment a route's go command runs with.
//
// ⛔ EVERY `GO*` AND `CGO_*` THE HOST CARRIES IS DROPPED, and the route's set
// is the whole of it: `GOENV=off` so no `go env -w` file takes part,
// `GOWORK=off`, a module cache and a build cache of the workdir's own, the
// pinned toolchain, `CGO_ENABLED=0`, and sum.golang.org. ⚠ `-modcacherw` so the
// cache can be removed afterwards. ⚠ The rest of the environment is the host's,
// git's configuration among it.
//
// ⭐ CGO OFF, AND THAT IS A CHOICE THE RECORD STATES. With cgo the library
// links libutp, C++, for uTP; without it, anacrolix/utp, which is Go. A runner
// has a C compiler on Linux and may not on Windows, so cgo's default would make
// the two platforms run two uTP implementations and tie the executable to the
// host's compiler. Off is one implementation on both and a reproducible build.
func anacrolixGoEnv(environ []string, route, workdir string) ([]string, error) {
	var proxy, flags string
	switch route {
	case "package":
		proxy, flags = anacrolixProxy, "-mod=mod -modcacherw"
	case "source":
		proxy, flags = "direct", "-mod=readonly -modcacherw"
	default:
		return nil, fmt.Errorf("no go environment for route [%s]", route)
	}
	if workdir == "" {
		return nil, fmt.Errorf("no workdir")
	}
	var out []string
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "GO") || strings.HasPrefix(upper, "CGO_") {
			continue
		}
		out = append(out, kv)
	}
	gopath := filepath.Join(workdir, "gopath")
	return append(out,
		"GOENV=off", "GOWORK=off",
		"GOPATH="+gopath, "GOMODCACHE="+filepath.Join(gopath, "pkg", "mod"),
		"GOCACHE="+filepath.Join(workdir, "gocache"),
		"GOTOOLCHAIN="+anacrolixToolchain, "CGO_ENABLED=0",
		"GOPROXY="+proxy, "GOSUMDB="+anacrolixSumDB, "GOFLAGS="+flags,
	), nil
}

// anacrolixBuildArgs is the build command. ⛔ `-trimpath` so no path of the
// host is in the executable, and `-buildvcs=false` because the harness sits
// in this repository: Go stamps the main module's version from the enclosing
// git tree otherwise, and the library writes that version into the extended
// handshake's `v`, so every commit of this tree would change the wire.
func anacrolixBuildArgs(out string) []string {
	return []string{"build", "-trimpath", "-buildvcs=false", "-o", out, "."}
}

// anacrolixVersion is the module version as the record spells it, without `v`.
func anacrolixVersion() string { return strings.TrimPrefix(anacrolixPin, "v") }

// anacrolixManifest is what the harness's `--version` says it was built from.
type anacrolixManifest struct {
	library, librarySum, goVersion string
	deps                           int
	build                          map[string]string
}

// anacrolixParseManifest reads the harness's `--version`: its library line,
// then the build information as `go version -m` prints it.
func anacrolixParseManifest(out string) (anacrolixManifest, error) {
	m := anacrolixManifest{build: map[string]string{}}
	lines := strings.Split(strings.ReplaceAll(out, "\r", ""), "\n")
	head := strings.Fields(lines[0])
	if len(head) < 5 || head[0] != anacrolixExeName || head[1] != "library" || head[2] != anacrolixModule {
		return m, fmt.Errorf("the harness's answer does not begin with its library line: [%s]", lines[0])
	}
	m.library, m.librarySum = head[3], head[4]
	if len(head) != 5 {
		return m, fmt.Errorf("the harness's library line carries more than a version and a sum: [%s]", lines[0])
	}
	path, mod := "", ""
	for _, l := range lines[1:] {
		f := strings.Split(l, "\t")
		switch f[0] {
		case "go":
			if len(f) == 2 {
				m.goVersion = f[1]
			}
		case "path":
			if len(f) == 2 {
				path = f[1]
			}
		case "mod":
			mod = l
		case "dep":
			if len(f) != 4 || !strings.HasPrefix(f[3], "h1:") {
				return m, fmt.Errorf("a dependency carries no checksum: [%s]", l)
			}
			m.deps++
		case "=>":
			return m, fmt.Errorf("the build replaced a module: [%s]", l)
		case "build":
			if len(f) == 2 {
				k, v, _ := strings.Cut(f[1], "=")
				m.build[k] = v
			}
		}
	}
	if path != anacrolixHarnessModule {
		return m, fmt.Errorf("the build's main package is [%s], not %s", path, anacrolixHarnessModule)
	}
	if mod != "mod\t"+anacrolixHarnessModule+"\t(devel)\t" {
		return m, fmt.Errorf("the build's main module line is [%s], not the harness at (devel)", mod)
	}
	return m, nil
}

// anacrolixBuildKeys are the build settings a harness build may carry.
var anacrolixBuildKeys = map[string]bool{"-buildmode": true, "-compiler": true, "-trimpath": true,
	"DefaultGODEBUG": true, "CGO_ENABLED": true, "GOARCH": true, "GOOS": true, "GOAMD64": true}

// anacrolixManifestRefusal is what is wrong with a build's manifest, or "".
//
// ⛔ THE PIN, THE TOOLCHAIN AND THE FLAGS, READ OUT OF THE EXECUTABLE: the
// library at the pin under sum.golang.org's hash, the pinned Go, `-trimpath`,
// cgo off, no `vcs.*` stamp, and no setting this list does not name. ⚠
// `DefaultGODEBUG` is the go.mod's `go 1.24.0` line speaking: a newer toolchain
// keeps that release's defaults, and the value is recorded as it is.
func anacrolixManifestRefusal(m anacrolixManifest, goos string) string {
	switch {
	case m.library != anacrolixPin:
		return fmt.Sprintf("the build links %s %s, not the pin %s", anacrolixModule, m.library, anacrolixPin)
	case m.librarySum != anacrolixPinSum:
		return fmt.Sprintf("the build links %s under [%s], not sum.golang.org's %s", anacrolixModule, m.librarySum, anacrolixPinSum)
	case m.goVersion != anacrolixToolchain:
		return fmt.Sprintf("the build was compiled by [%s], not %s", m.goVersion, anacrolixToolchain)
	case m.deps == 0:
		return "the build names no dependency"
	}
	var keys []string
	for k := range m.build {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !anacrolixBuildKeys[k] {
			return fmt.Sprintf("the build carries the setting [%s=%s], which a harness build does not", k, m.build[k])
		}
	}
	for k, want := range map[string]string{"-buildmode": "exe", "-compiler": "gc", "-trimpath": "true",
		"CGO_ENABLED": "0", "GOOS": goos} {
		if m.build[k] != want {
			return fmt.Sprintf("the build says %s=[%s], not %s", k, m.build[k], want)
		}
	}
	return ""
}

// anacrolixLatest reads the go command's answer for `<module>@latest`.
func anacrolixLatest(body []byte) (string, error) {
	var a struct {
		Path, Version string
	}
	if err := json.Unmarshal(body, &a); err != nil {
		return "", fmt.Errorf("the proxy's answer for @latest is not JSON: %v", err)
	}
	if a.Path != anacrolixModule {
		return "", fmt.Errorf("the proxy answered for [%s], not %s", a.Path, anacrolixModule)
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(a.Version) {
		return "", fmt.Errorf("the proxy's latest is [%s], which is not a stable version", a.Version)
	}
	return a.Version, nil
}

// anacrolixOriginRefusal is what is wrong with the go command's download
// record of the library, `<GOMODCACHE>/cache/download/<module>/@v/<pin>.info`,
// or "". ⛔ The tag's commit, from GitHub, at the pin: the record the source
// route takes its `source-commit` from.
func anacrolixOriginRefusal(body []byte) string {
	var info struct {
		Version string
		Origin  *struct{ VCS, URL, Hash, Ref string }
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "the library's download record is not JSON: " + err.Error()
	}
	switch o := info.Origin; {
	case info.Version != anacrolixPin:
		return fmt.Sprintf("the library's download record names [%s], not %s", info.Version, anacrolixPin)
	case o == nil:
		return "the library's download record names no origin"
	case o.VCS != "git" || o.URL != "https://github.com/"+anacrolixRepo:
		return fmt.Sprintf("the library came from [%s %s], not git at https://github.com/%s", o.VCS, o.URL, anacrolixRepo)
	case o.Ref != "refs/tags/"+anacrolixPin:
		return fmt.Sprintf("the library came from [%s], not refs/tags/%s", o.Ref, anacrolixPin)
	case o.Hash != anacrolixPinnedCommit:
		return fmt.Sprintf("the tag names commit [%s], not the pinned %s", o.Hash, anacrolixPinnedCommit)
	}
	return ""
}

// anacrolixVCSRepos counts the repositories the go command cloned into a
// module cache's `cache/vcs`: one directory each, beside a `.info` and a
// `.lock` file.
func anacrolixVCSRepos(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}

// anacrolixVCSRefusal is what is wrong with the number of repositories a
// route cloned, or "". ⛔ The package route clones none, because every module
// came from the proxy; the source route clones at least the library's.
func anacrolixVCSRefusal(route string, repos int) string {
	switch {
	case route == "package" && repos != 0:
		return fmt.Sprintf("the package route cloned %d repositories, so some module did not come from the proxy", repos)
	case route == "source" && repos == 0:
		return "the source route cloned no repository, so its modules came from somewhere else"
	}
	return ""
}

// -- the harness's command line and its report --------------------------------

// anacrolixRequiredSwitches is every switch the harness must be passed, once.
// ⛔ Each is a surface the library's default config has ON, read from
// `NewDefaultClientConfig` at the pin and from the harness's own report:
//
//   - `-dht=false`: the DHT, bootstrapping from the library's global nodes;
//   - `-pex=false`: `ut_pex`, offered in the extended handshake and sent;
//   - `-port-forward=false`: UPnP and NAT-PMP, `NoDefaultPortForwarding`;
//   - `-webseeds=false` and `-webtorrent=false`: HTTP seeds and WebRTC peers;
//   - `-ipv6=false`: IPv6, which the capture's lab does not carry. ⚠ Measured
//     on 2026-10-22: on a kernel booted with IPv6 disabled the library's
//     default listen on `tcp6` fails `socket: address family not supported`,
//     which it does not treat as an unsupported network, so the client never
//     starts. Off on every host, so the profile does not depend on the host.
//
// ⚠ The library has no local peer discovery at the pin: no code names it.
var anacrolixRequiredSwitches = []string{"-dht=false", "-pex=false", "-port-forward=false", "-webseeds=false",
	"-webtorrent=false", "-ipv6=false"}

// anacrolixStartArgs is the whole command line `start` gives the harness.
// ⚠ uTP is left at the library's default, on, and so is the peer port when
// the caller says 0: 42069.
func anacrolixStartArgs(torrent, workdir, peerPort string) ([]string, error) {
	if torrent == "" || workdir == "" {
		return nil, fmt.Errorf("no torrent or no workdir")
	}
	if peerPort != "0" && !rqbitPortOK(peerPort) {
		return nil, fmt.Errorf("peer port [%s] is not a port", peerPort)
	}
	args := []string{"-torrent", torrent, "-data", filepath.Join(workdir, "downloads"),
		"-ready", filepath.Join(workdir, "ready.txt"), "-stop-file", filepath.Join(workdir, "stop-request")}
	// ⛔ SPELLED HERE, NOT TAKEN FROM anacrolixRequiredSwitches, so the refusal
	// is a second list the line is held to rather than the line's own source.
	args = append(args,
		"-dht=false", "-pex=false", "-port-forward=false", "-webseeds=false", "-webtorrent=false", "-ipv6=false",
	)
	if peerPort != "0" {
		args = append(args, "-port", peerPort)
	}
	return args, nil
}

// anacrolixArgsRefusal answers what is wrong with a harness command line, or
// "". ⛔ Every switch once, no surface switched back on, nothing it does not
// name.
func anacrolixArgsRefusal(args []string) string {
	count := map[string]int{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, _, _ := strings.Cut(a, "=")
		switch name {
		case "-torrent", "-data", "-ready", "-stop-file", "-port":
			if strings.Contains(a, "=") || i+1 >= len(args) {
				return "[" + a + "] is not followed by its value"
			}
			count[name]++
			i++
			continue
		case "-dht", "-pex", "-port-forward", "-webseeds", "-webtorrent", "-ipv6":
			if a != name+"=false" {
				return "[" + a + "] switches " + name + " on"
			}
			count[name]++
			continue
		}
		return "[" + a + "] is not a switch this adapter passes"
	}
	for _, s := range anacrolixRequiredSwitches {
		name, _, _ := strings.Cut(s, "=")
		if count[name] != 1 {
			return fmt.Sprintf("%s is passed %d times, not once", s, count[name])
		}
	}
	for _, s := range []string{"-torrent", "-data", "-ready", "-stop-file"} {
		if count[s] != 1 {
			return fmt.Sprintf("%s is passed %d times, not once", s, count[s])
		}
	}
	if count["-port"] > 1 {
		return "-port is passed more than once"
	}
	return ""
}

// anacrolixChildEnv is the environment the harness runs with.
//
// ⛔ DROPPED: every `GO*`, which is the runtime's own knobs and the library's
// `GO_LOG` and anacrolix/utp's `GO_UTP_*`; every `TORRENT_*` and
// `WEBTORRENT_*`, which the library reads at start; and every proxy variable,
// which the library's tracker client ignores and Go's default transport does
// not. `extra` is appended after the filter.
func anacrolixChildEnv(environ []string, extra ...string) []string {
	var out []string
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "GO") || strings.HasPrefix(upper, "TORRENT_") ||
			strings.HasPrefix(upper, "WEBTORRENT_") || strings.HasSuffix(upper, "_PROXY") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, extra...)
}

// anacrolixReadyRefusal is what is wrong with the harness's report of the
// client it configured, `ready.txt`, or "".
//
// ⛔ THE CLIENT'S OWN CONFIG, READ BACK: every surface off as asked, nothing
// pointing at a proxy or an ICE server, every listener IPv4, and the three
// identity defaults exactly the library's, so nothing set them. ⚠ The last are
// self-reports and are only compared, never published.
func anacrolixReadyRefusal(body string) string {
	if !strings.HasPrefix(body, "bit-ids/anacrolix-torrent-harness-ready/1\n") {
		return "the harness's report does not begin with its schema line"
	}
	b := []byte(body)
	for k, want := range map[string]string{
		"no_dht": "true", "disable_pex": "true", "no_default_port_forwarding": "true",
		"disable_webseeds": "true", "disable_webtorrent": "true", "disable_ipv6": "true",
		"disable_trackers": "false", "http_proxy_set": "false", "ice_servers": "0",
		"self_report_bep20":                anacrolixDefaultBep20,
		"library_default_bep20":            anacrolixDefaultBep20,
		"self_report_http_user_agent":      "anacrolix-torrent/" + anacrolixPin,
		"self_report_extended_handshake_v": anacrolixHarnessModule + " (devel) (anacrolix/torrent " + anacrolixPin + ")",
	} {
		if got := keyField(b, k); got != want {
			return fmt.Sprintf("the harness reports %s=[%s], not %s", k, got, want)
		}
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(keyField(b, "infohash")) {
		return "the harness reports no info hash"
	}
	if !strings.HasPrefix(keyField(b, "self_report_peer_id"), hex.EncodeToString([]byte(anacrolixDefaultBep20))) {
		return "the harness's peer ID does not begin with the library's prefix"
	}
	listen := keyField(b, "listen")
	if listen == "" {
		return "the harness reports no listener"
	}
	port := keyField(b, "listen_port")
	for _, l := range strings.Split(listen, ",") {
		f := strings.Fields(l)
		if len(f) != 2 || (f[0] != "tcp" && f[0] != "udp") {
			return "the harness reports the listener [" + l + "]"
		}
		host, p, ok := strings.Cut(f[1], ":")
		if !ok || strings.Contains(p, ":") || strings.HasPrefix(host, "[") {
			return "the harness listens on [" + f[1] + "], which is not IPv4"
		}
		if p != port {
			return "the harness listens on [" + f[1] + "], not port " + port
		}
	}
	return ""
}

// anacrolixStateListing lists what the session left under dir, as rqbit's.
func anacrolixStateListing(dir string) string { return rqbitStateListing(dir) }

// -- the install both halves run ------------------------------------------------

// anacrolixExit is a refusal (1) or a could-not-run (2) with its words.
type anacrolixExit struct {
	code int
	msg  string
}

func (e anacrolixExit) Error() string { return e.msg }

func anRefuse(f string, a ...any) error { return anacrolixExit{1, fmt.Sprintf(f, a...)} }
func anCannot(f string, a ...any) error { return anacrolixExit{2, fmt.Sprintf(f, a...)} }

// anacrolixHarnessSource is the harness's directory: BIT_IDS_ANACROLIX_HARNESS
// for a standalone drive, else this tree's.
func anacrolixHarnessSource() (string, error) {
	if d := os.Getenv("BIT_IDS_ANACROLIX_HARNESS"); d != "" {
		return d, nil
	}
	root, err := captureRoot()
	if err != nil {
		return "", anCannot("%v", err)
	}
	return filepath.Join(root, filepath.FromSlash(anacrolixHarnessDir)), nil
}

// anacrolixRun runs one go command in dir with env, both streams appended to
// log, bounded by seconds. ⚠ install-client bounds the whole call as well.
func anacrolixRun(log, dir string, env []string, seconds int, prog string, args ...string) (int, error) {
	f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fmt.Fprintf(f, "## %s %s %s\n", utcNow(), filepath.Base(prog), strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, env, f, f
	if null, err := os.Open(os.DevNull); err == nil {
		defer null.Close()
		cmd.Stdin = null
	}
	err = cmd.Run()
	if ctx.Err() != nil {
		return 124, nil
	}
	return exitOf(err), nil
}

// anacrolixInstall is one route's install, the same on both platforms: copy
// the pinned harness into the workdir, resolve, build with the route's go
// environment, hold what the build wrote and what the executable says to the
// pin, and place the executable at `<bin>/<exe>`.
func anacrolixInstall(route, workdir, bin string, windows bool) error {
	src, err := anacrolixHarnessSource()
	if err != nil {
		return err
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		return anCannot("the %s route compiles the harness and go is not on this host", route)
	}
	if route == "source" {
		if _, err := exec.LookPath("git"); err != nil {
			return anCannot("the source route clones every module and git is not on this host")
		}
	}
	read := func(name string) (string, error) {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return "", anCannot("the harness has no %s in %s", name, src)
		}
		return string(b), nil
	}
	mainGo, err := read("main.go")
	if err != nil {
		return err
	}
	gomod, err := read("go.mod")
	if err != nil {
		return err
	}
	gosum, err := read("go.sum")
	if err != nil {
		return err
	}
	if msg := anacrolixPinRefusal(gomod, gosum); msg != "" {
		return anRefuse("the harness is not pinned: %s", msg)
	}
	digest := sha256.New()
	for _, kv := range [][2]string{
		{"main.go", mainGo},
		{"go.mod", gomod},
		{"go.sum", gosum},
	} {
		s := sha256.Sum256([]byte(kv[1]))
		fmt.Fprintf(digest, "%x  %s\n", s, kv[0])
	}
	harnessSum := hex.EncodeToString(digest.Sum(nil))

	// ⛔ THE VERSION THE ROUTE'S OWN RESOLVER NAMED, held to the pin.
	tag := ""
	if route == "source" {
		tag = os.Getenv("BIT_IDS_SOURCE_TAG")
		if tag == "" {
			return anCannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from the repository's own refs before the route was cut")
		}
		if tag != anacrolixPin {
			return anRefuse("the source lane resolved %s and the harness pins %s; re-pin the harness to the new stable tag", tag, anacrolixPin)
		}
	}

	build := filepath.Join(workdir, "harness")
	out := filepath.Join(workdir, "out")
	_ = os.RemoveAll(build)
	if err := os.MkdirAll(build, 0o755); err != nil {
		return anCannot("cannot create %s", build)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return anCannot("cannot create %s", out)
	}
	files := map[string]string{"main.go": mainGo, "go.mod": gomod}
	// ⛔ THE PACKAGE ROUTE GETS NO go.sum: every module it fetches is checked
	// against sum.golang.org, and the go.sum it writes is compared afterwards.
	if route == "source" {
		files["go.sum"] = gosum
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(build, name), []byte(body), 0o644); err != nil {
			return anCannot("cannot write %s", name)
		}
	}
	env, err := anacrolixGoEnv(os.Environ(), route, workdir)
	if err != nil {
		return anCannot("%v", err)
	}
	log := filepath.Join(workdir, "install.log")
	defer func() {
		// ⚠ The caches are a gigabyte nothing reads again; what the build
		// says of itself is in the workdir's own files. So is the executable,
		// once it is placed.
		for _, d := range []string{"gopath", "gocache", "out"} {
			anacrolixRemoveTree(filepath.Join(workdir, d))
		}
	}()
	bootstrap, _ := exec.Command(goTool, "version").Output()

	if route == "package" {
		// ⭐ THE PROXY'S `@latest` IS THE RESOLVER, asked through the go command
		// under the route's own environment.
		// ⚠ `.txt` though it is JSON: the install artifact ships the
		// workdir's `*.txt`, `*.log` and `*.err` and nothing else.
		latest := filepath.Join(workdir, "proxy-latest.txt")
		lf, err := os.Create(latest)
		if err != nil {
			return anCannot("cannot write %s", latest)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
		cmd := exec.CommandContext(ctx, goTool, "list", "-m", "-json", anacrolixModule+"@latest")
		cmd.Dir, cmd.Env, cmd.Stdout = build, env, lf
		errLog, _ := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if errLog != nil {
			cmd.Stderr = errLog
		}
		err = cmd.Run()
		cancel()
		lf.Close()
		if errLog != nil {
			errLog.Close()
		}
		if err != nil {
			return anRefuse("the proxy could not be asked for %s@latest (exit %d); install.log says why", anacrolixModule, exitOf(err))
		}
		body, _ := os.ReadFile(latest)
		v, err := anacrolixLatest(body)
		if err != nil {
			return anRefuse("%v", err)
		}
		if v != anacrolixPin {
			return anRefuse("the proxy's latest is %s and the harness pins %s; re-pin the harness to the new stable version", v, anacrolixPin)
		}
		tag = v
	}

	exe := filepath.Join(out, anacrolixExe(windows))
	started := time.Now()
	rc, err := anacrolixRun(log, build, env, 1400, goTool, anacrolixBuildArgs(exe)...)
	took := int(time.Since(started) / time.Second)
	if err != nil {
		return anCannot("cannot write %s", log)
	}
	switch rc {
	case 0:
	case 124:
		return anRefuse("go build did not finish within 1400s")
	default:
		return anRefuse("go build exited %d; install.log says why", rc)
	}

	// ⛔ WHAT THE BUILD WROTE, HELD TO THE PIN. The module file is never
	// changed, and the package route's go.sum, written from sum.golang.org's
	// answers, is the committed one byte for byte.
	if b, _ := os.ReadFile(filepath.Join(build, "go.mod")); string(b) != gomod {
		return anRefuse("the build changed the harness's go.mod")
	}
	wrote, _ := os.ReadFile(filepath.Join(build, "go.sum"))
	_ = os.WriteFile(filepath.Join(workdir, "go-sum.txt"), wrote, 0o644)
	if string(wrote) != gosum {
		return anRefuse("the go.sum this build wrote is not the committed one; go-sum.txt holds it")
	}

	// ⛔ AND WHERE THE LIBRARY CAME FROM, in the go command's own record.
	info := filepath.Join(workdir, "gopath", "pkg", "mod", "cache", "download",
		filepath.FromSlash(anacrolixModule), "@v", anacrolixPin+".info")
	ib, err := os.ReadFile(info)
	if err != nil {
		return anRefuse("the build left no download record of %s", anacrolixModule)
	}
	_ = os.WriteFile(filepath.Join(workdir, "library-origin.txt"), ib, 0o644)
	if msg := anacrolixOriginRefusal(ib); msg != "" {
		return anRefuse("%s", msg)
	}

	// ⭐ THE BUILD MANIFEST IS THE EXECUTABLE'S OWN ANSWER.
	answer, err := exec.Command(exe, "--version").Output()
	if err != nil {
		return anRefuse("the built harness would not say what it is: %v", err)
	}
	_ = os.WriteFile(filepath.Join(workdir, "build-manifest.txt"), answer, 0o644)
	m, err := anacrolixParseManifest(string(answer))
	if err != nil {
		return anRefuse("%v", err)
	}
	goos := "linux"
	if windows {
		goos = "windows"
	}
	if msg := anacrolixManifestRefusal(m, goos); msg != "" {
		return anRefuse("%s", msg)
	}
	exeSum, _ := fileSHA256(exe)
	// ⭐ HOW MANY REPOSITORIES GIT CLONED, which is what tells the two routes
	// apart: the proxy's download records carry the same origin as a direct
	// fetch's, so the record alone cannot.
	repos := anacrolixVCSRepos(filepath.Join(workdir, "gopath", "pkg", "mod", "cache", "vcs"))
	if msg := anacrolixVCSRefusal(route, repos); msg != "" {
		return anRefuse("%s", msg)
	}

	var note strings.Builder
	fmt.Fprintf(&note, "route=%s\nmodule=%s\nversion=%s\nsum=%s\ncommit=%s\nharness=%s\nharness_sha256=%s\n",
		route, anacrolixModule, anacrolixPin, anacrolixPinSum, anacrolixPinnedCommit, anacrolixHarnessDir, harnessSum)
	fmt.Fprintf(&note, "toolchain=%s\nbootstrap=%s\nbuild=go %s\nexecutable_sha256=%s\nseconds=%d\n",
		anacrolixToolchain, strings.TrimSpace(string(bootstrap)), strings.Join(anacrolixBuildArgs("<out>"), " "), exeSum, took)
	fmt.Fprintf(&note, "vcs_repositories=%d\n", repos)
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "GO") || strings.HasPrefix(name, "CGO_") {
			fmt.Fprintf(&note, "env %s\n", kv)
		}
	}
	if root, err := captureRoot(); err == nil {
		if c, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
			fmt.Fprintf(&note, "tree_commit=%s\n", strings.TrimSpace(string(c)))
		}
	}
	if err := os.WriteFile(filepath.Join(workdir, "build-env.txt"), []byte(note.String()), 0o644); err != nil {
		return anRefuse("the route could not record how it built")
	}

	// ⛔ THE SOURCE ROUTE'S COMMIT IS THE ONE GIT FETCHED, from the go command's
	// own record, which `E-ACQ-06` asks for.
	if route == "source" {
		if err := os.WriteFile(filepath.Join(workdir, "source-commit"), []byte(anacrolixPinnedCommit+"\n"), 0o644); err != nil {
			return anRefuse("the route could not record its commit")
		}
	} else {
		origin := anacrolixProxy + "/" + anacrolixModule + "/@v/" + anacrolixPin + ".zip"
		if err := writePackageIndexVia(workdir, anacrolixProxyIndex, anacrolixModule, anacrolixVersion(), origin, anacrolixDelivery); err != nil {
			return anRefuse("the route could not record what the index named")
		}
	}

	if err := os.MkdirAll(bin, 0o755); err != nil {
		return anRefuse("this user cannot create %s; a rootless install writes nowhere else", bin)
	}
	if err := anacrolixCopy(exe, filepath.Join(bin, anacrolixExe(windows))); err != nil {
		return anRefuse("the harness could not be placed in %s: %v", bin, err)
	}
	return nil
}

// anacrolixRemoveTree removes a module cache. ⚠ `-modcacherw` does not reach
// the toolchain: the go command that switches to the pinned one extracts it
// read-only, measured on 2026-10-22 as 253 megabytes `RemoveAll` left behind,
// so every directory is made writable first. ⚠ And every file: on Windows a
// read-only file is an attribute, which only a chmod of the file clears.
func anacrolixRemoveTree(dir string) {
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
		case d.IsDir():
			_ = os.Chmod(p, 0o755)
		case d.Type().IsRegular():
			_ = os.Chmod(p, 0o644)
		}
		return nil
	})
	_ = os.RemoveAll(dir)
}

// anacrolixCopy places a built executable.
func anacrolixCopy(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := to + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, in); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, to)
}

// anacrolixAsk runs `<exe> --version`, a pure read, and answers the library's
// version as the record spells it and the whole answer.
func anacrolixAsk(exe string) (string, string, error) {
	cmd := exec.Command(exe, "--version")
	cmd.Env = anacrolixChildEnv(os.Environ())
	if null, err := os.Open(os.DevNull); err == nil {
		defer null.Close()
		cmd.Stdin = null
	}
	out, err := cmd.Output()
	answer := strings.TrimRight(strings.ReplaceAll(string(out), "\r", ""), "\n")
	if err != nil {
		return "", answer, fmt.Errorf("%s --version exited %d", filepath.Base(exe), exitOf(err))
	}
	m, err := anacrolixParseManifest(answer)
	if err != nil {
		return "", answer, err
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(m.library) {
		return "", answer, fmt.Errorf("the harness links %s at [%s], which is not a stable version", anacrolixModule, m.library)
	}
	return strings.TrimPrefix(m.library, "v"), answer, nil
}

// anacrolixDrive waits for the harness's report and holds it. ⚠ A bounded
// wait on the work: `gone` says whether the harness has exited.
func anacrolixDrive(workdir string, gone func(time.Duration) bool) (string, error) {
	ready := filepath.Join(workdir, "ready.txt")
	for waited := 0; ; waited++ {
		if b, err := os.ReadFile(ready); err == nil && len(b) > 0 && strings.HasSuffix(string(b), "\n") {
			if msg := anacrolixReadyRefusal(string(b)); msg != "" {
				return "", errors.New(msg)
			}
			return keyField(b, "listen"), nil
		}
		if gone(time.Second) {
			return "", fmt.Errorf("the harness exited before it reported its client; %s says why", filepath.Join(workdir, "client.out"))
		}
		if waited >= 30 {
			return "", fmt.Errorf("the harness did not report its client within 30 seconds")
		}
	}
}

// anacrolixGraceSeconds is how long `stop` waits for the library's own Close
// after the stop file appears, before it kills.
const anacrolixGraceSeconds = 10

// -- the selftest -------------------------------------------------------------

// anacrolixSelftest is `bit-check anacrolix-torrent --selftest`: every
// decision above asked over a case whose answer is known, with the defects
// planted, on any host.
func anacrolixSelftest(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "anacrolix-torrent: usage: bit-check anacrolix-torrent --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "anacrolix-torrent: FAILED %s\n", name)
		}
	}

	// The description.
	d := anacrolixDescribe()
	for _, want := range []string{"target_kind=library", "release_repo=anacrolix/torrent", "release_tag_prefix=v",
		"release_min_components=3", "release_max_components=3", "release_floor_major=-",
		"compiles=package,source", "routes=package,source"} {
		check("describe carries "+want, strings.Contains(d, want+"\n"))
	}
	check("describe declares no release asset and no digest", !strings.Contains(d, "release_asset") && !strings.Contains(d, "release_digest"))
	for _, c := range []struct {
		win         bool
		route, want string
		ok          bool
	}{{false, "package", "elf-binary-static", true}, {false, "source", "elf-binary-static", true},
		{false, "release", "", false}, {true, "package", "pe-binary", true}, {true, "source", "pe-binary", true},
		{true, "release", "", false}} {
		f, ok := anacrolixPackageFormat(c.win, c.route)
		check(fmt.Sprintf("windows=%v route %s packages as [%s]", c.win, c.route, c.want), ok == c.ok && f == c.want)
	}
	check("the executable is named per platform", anacrolixExe(false) == "anacrolix-torrent-harness" &&
		anacrolixExe(true) == "anacrolix-torrent-harness.exe")

	// The pin, over the harness's own files in the shape they are committed.
	gomod := "module bit-ids/harness/anacrolix-torrent\n\ngo 1.24.0\n\ntoolchain go1.27.3\n\nrequire github.com/anacrolix/torrent v1.61.0\n\nrequire (\n\tgithub.com/anacrolix/dht/v2 v2.23.0 // indirect\n)\n"
	gosum := "github.com/anacrolix/dht/v2 v2.23.0 h1:EuD17ykTTEkAMPLjBsS5QjGOwuBgLTdQhds6zPAjeVY=\n" +
		"github.com/anacrolix/torrent v1.61.0 " + anacrolixPinSum + "\n" +
		"github.com/anacrolix/torrent v1.61.0/go.mod " + anacrolixPinModSum + "\n"
	check("the committed shape is pinned", anacrolixPinRefusal(gomod, gosum) == "")
	check("a CRLF checkout is pinned", anacrolixPinRefusal(strings.ReplaceAll(gomod, "\n", "\r\n"), strings.ReplaceAll(gosum, "\n", "\r\n")) == "")
	for name, bad := range map[string][2]string{
		"another version":       {strings.Replace(gomod, "torrent v1.61.0", "torrent v1.60.0", 1), gosum},
		"a replace":             {gomod + "replace github.com/anacrolix/torrent => ../torrent\n", gosum},
		"an exclude":            {gomod + "exclude github.com/anacrolix/torrent v1.60.0\n", gosum},
		"another toolchain":     {strings.Replace(gomod, "toolchain go1.27.3", "toolchain go1.27.2", 1), gosum},
		"no toolchain":          {strings.Replace(gomod, "toolchain go1.27.3\n", "", 1), gosum},
		"an indirect library":   {strings.Replace(gomod, "torrent v1.61.0\n", "torrent v1.61.0 // indirect\n", 1), gosum},
		"no library":            {strings.Replace(gomod, "require github.com/anacrolix/torrent v1.61.0\n", "", 1), gosum},
		"another module":        {strings.Replace(gomod, "module bit-ids/harness/anacrolix-torrent", "module x", 1), gosum},
		"another library hash":  {gomod, strings.Replace(gosum, anacrolixPinSum, "h1:"+strings.Repeat("A", 43)+"=", 1)},
		"no go.mod hash":        {gomod, strings.Replace(gosum, "github.com/anacrolix/torrent v1.61.0/go.mod "+anacrolixPinModSum+"\n", "", 1)},
		"a second version hash": {gomod, gosum + "github.com/anacrolix/torrent v1.60.0 h1:x=\n"},
	} {
		check("a harness with "+name+" is refused", anacrolixPinRefusal(bad[0], bad[1]) != "")
	}

	// The go environment: the host's GO* gone, the route's set whole.
	host := []string{"PATH=/bin", "GOFLAGS=-mod=vendor", "GOPROXY=https://evil.invalid", "GONOSUMDB=*", "GOPRIVATE=*",
		"GOINSECURE=*", "GOTOOLCHAIN=local", "CGO_ENABLED=1", "goamd64=v3", "HOME=/h", "GIT_SSL_CAINFO=/ca"}
	for _, route := range []string{"package", "source"} {
		env, err := anacrolixGoEnv(host, route, "/w")
		j := "\n" + strings.Join(env, "\n") + "\n"
		check(route+": the go environment is built", err == nil)
		for _, want := range []string{"GOENV=off", "GOWORK=off", "GOPATH=" + filepath.Join("/w", "gopath"),
			"GOMODCACHE=" + filepath.Join("/w", "gopath", "pkg", "mod"), "GOCACHE=" + filepath.Join("/w", "gocache"),
			"GOTOOLCHAIN=go1.27.3", "CGO_ENABLED=0", "GOSUMDB=sum.golang.org", "PATH=/bin", "HOME=/h", "GIT_SSL_CAINFO=/ca"} {
			check(route+": the go environment carries "+want, strings.Contains(j, "\n"+want+"\n"))
		}
		for _, gone := range []string{"GONOSUMDB=", "GOPRIVATE=", "GOINSECURE=", "GOTOOLCHAIN=local", "CGO_ENABLED=1",
			"goamd64=", "evil.invalid", "-mod=vendor"} {
			check(route+": the host's "+gone+" is dropped", !strings.Contains(j, gone))
		}
		check(route+": GOPROXY is set once", strings.Count(j, "\nGOPROXY=") == 1 && strings.Count(j, "\nGOFLAGS=") == 1)
	}
	pe, _ := anacrolixGoEnv(host, "package", "/w")
	se, _ := anacrolixGoEnv(host, "source", "/w")
	pj, sj := "\n"+strings.Join(pe, "\n")+"\n", "\n"+strings.Join(se, "\n")+"\n"
	check("the package route reads the module proxy", strings.Contains(pj, "\nGOPROXY=https://proxy.golang.org\n") &&
		strings.Contains(pj, "\nGOFLAGS=-mod=mod -modcacherw\n"))
	check("the source route reads every module from its origin", strings.Contains(sj, "\nGOPROXY=direct\n") &&
		strings.Contains(sj, "\nGOFLAGS=-mod=readonly -modcacherw\n"))
	_, err := anacrolixGoEnv(host, "release", "/w")
	check("a release route has no go environment", err != nil)
	_, err = anacrolixGoEnv(host, "package", "")
	check("no workdir has no go environment", err != nil)
	ba := strings.Join(anacrolixBuildArgs("/o/x"), " ")
	check("the build is trimmed and unstamped", ba == "build -trimpath -buildvcs=false -o /o/x .")
	check("the record's version has no v", anacrolixVersion() == "1.61.0")

	// The manifest, in the shape go1.27.3 wrote on 2026-10-22.
	man := "anacrolix-torrent-harness library github.com/anacrolix/torrent v1.61.0 " + anacrolixPinSum + "\n" +
		"go\tgo1.27.3\npath\tbit-ids/harness/anacrolix-torrent\nmod\tbit-ids/harness/anacrolix-torrent\t(devel)\t\n" +
		"dep\tgithub.com/anacrolix/dht/v2\tv2.23.0\th1:EuD17ykTTEkAMPLjBsS5QjGOwuBgLTdQhds6zPAjeVY=\n" +
		"dep\tgithub.com/anacrolix/torrent\tv1.61.0\t" + anacrolixPinSum + "\n" +
		"build\t-buildmode=exe\nbuild\t-compiler=gc\nbuild\t-trimpath=true\n" +
		"build\tDefaultGODEBUG=containermaxprocs=0,tlssha1=1\nbuild\tCGO_ENABLED=0\nbuild\tGOARCH=amd64\n" +
		"build\tGOOS=linux\nbuild\tGOAMD64=v1\n"
	m, err := anacrolixParseManifest(man)
	check("the manifest is read", err == nil && m.library == "v1.61.0" && m.goVersion == "go1.27.3" && m.deps == 2)
	check("the manifest is held on Linux", err == nil && anacrolixManifestRefusal(m, "linux") == "")
	check("a Linux manifest is refused on Windows", anacrolixManifestRefusal(m, "windows") != "")
	check("a CRLF answer is read", func() bool {
		mm, err := anacrolixParseManifest(strings.ReplaceAll(man, "\n", "\r\n"))
		return err == nil && anacrolixManifestRefusal(mm, "linux") == ""
	}())
	for name, bad := range map[string]string{
		"another library":       strings.Replace(man, "torrent v1.61.0 h1", "torrent v1.60.0 h1", 1),
		"another library hash":  strings.Replace(man, "v1.61.0 "+anacrolixPinSum+"\n", "v1.61.0 h1:x=\n", 1),
		"a replaced library":    strings.Replace(man, anacrolixPinSum+"\n", anacrolixPinSum+" => ../t  \n", 1),
		"a replace line":        man + "=>\t../t\t\t\n",
		"another toolchain":     strings.Replace(man, "go\tgo1.27.3", "go\tgo1.24.7", 1),
		"cgo on":                strings.Replace(man, "CGO_ENABLED=0", "CGO_ENABLED=1", 1),
		"no -trimpath":          strings.Replace(man, "build\t-trimpath=true\n", "", 1),
		"a vcs stamp":           man + "build\tvcs=git\nbuild\tvcs.revision=abc\n",
		"ldflags":               man + "build\t-ldflags=-X main.v=1\n",
		"build tags":            man + "build\t-tags=disable_libutp\n",
		"a stamped main module": strings.Replace(man, "\t(devel)\t", "\tv0.0.0-20261022-abc+dirty\t", 1),
		"another main package":  strings.Replace(man, "path\tbit-ids/harness/anacrolix-torrent", "path\tx", 1),
		"an unsummed dep":       strings.Replace(man, "\tv2.23.0\th1:EuD17ykTTEkAMPLjBsS5QjGOwuBgLTdQhds6zPAjeVY=", "\tv2.23.0", 1),
		"no library line":       strings.SplitN(man, "\n", 2)[1],
		"no dependency":         strings.Replace(strings.Replace(man, "dep\tgithub.com/anacrolix/dht/v2\tv2.23.0\th1:EuD17ykTTEkAMPLjBsS5QjGOwuBgLTdQhds6zPAjeVY=\n", "", 1), "dep\tgithub.com/anacrolix/torrent\tv1.61.0\t"+anacrolixPinSum+"\n", "", 1),
	} {
		mm, err := anacrolixParseManifest(bad)
		check("a manifest with "+name+" is refused", err != nil || anacrolixManifestRefusal(mm, "linux") != "")
	}

	check("a package route that cloned nothing is held", anacrolixVCSRefusal("package", 0) == "")
	check("a package route that cloned is refused", anacrolixVCSRefusal("package", 1) != "")
	check("a source route that cloned is held", anacrolixVCSRefusal("source", 66) == "")
	check("a source route that cloned nothing is refused", anacrolixVCSRefusal("source", 0) != "")

	// The proxy's answer, and the go command's download record.
	lat := `{"Path":"github.com/anacrolix/torrent","Version":"v1.61.0","Time":"2025-12-17T07:39:03Z"}`
	v, err := anacrolixLatest([]byte(lat))
	check("the proxy's latest reads as v1.61.0", err == nil && v == "v1.61.0")
	for name, bad := range map[string]string{
		"another module": strings.Replace(lat, "anacrolix/torrent", "anacrolix/dht", 1),
		"a prerelease":   strings.Replace(lat, "v1.61.0", "v1.62.0-rc.1", 1),
		"a pseudo":       strings.Replace(lat, "v1.61.0", "v1.61.1-0.20261001000000-abcdefabcdef", 1),
		"not JSON":       lat[:10],
	} {
		_, err := anacrolixLatest([]byte(bad))
		check("a latest answer with "+name+" is refused", err != nil)
	}
	org := `{"Version":"v1.61.0","Time":"2025-12-17T07:39:03Z","Origin":{"VCS":"git","URL":"https://github.com/anacrolix/torrent","Hash":"` + anacrolixPinnedCommit + `","Ref":"refs/tags/v1.61.0"}}`
	check("the library's origin is held", anacrolixOriginRefusal([]byte(org)) == "")
	for name, bad := range map[string]string{
		"another commit":  strings.Replace(org, anacrolixPinnedCommit, strings.Repeat("a", 40), 1),
		"a branch":        strings.Replace(org, "refs/tags/v1.61.0", "refs/heads/master", 1),
		"a fork":          strings.Replace(org, "github.com/anacrolix/torrent", "github.com/someone/torrent", 1),
		"another version": strings.Replace(org, `"Version":"v1.61.0"`, `"Version":"v1.60.0"`, 1),
		"no origin":       `{"Version":"v1.61.0"}`,
		"not JSON":        org[:12],
	} {
		check("a download record with "+name+" is refused", anacrolixOriginRefusal([]byte(bad)) != "")
	}

	// The command line: every surface off, refused otherwise.
	good, err := anacrolixStartArgs("/t.torrent", "/w", "51413")
	check("the command line is built", err == nil)
	check("the command line is held", anacrolixArgsRefusal(good) == "")
	check("the peer port is passed", strings.Contains(strings.Join(good, " "), "-port 51413"))
	zero, _ := anacrolixStartArgs("/t.torrent", "/w", "0")
	check("peer port 0 leaves the library's", !strings.Contains(strings.Join(zero, " "), "-port ") && anacrolixArgsRefusal(zero) == "")
	check("uTP is left at the library's default", !strings.Contains(strings.Join(good, " "), "-utp"))
	for _, bad := range [][3]string{
		{"", "/w", "1"}, {"/t", "", "1"}, {"/t", "/w", "x"}, {"/t", "/w", "70000"}, {"/t", "/w", "01"},
	} {
		_, err := anacrolixStartArgs(bad[0], bad[1], bad[2])
		check(fmt.Sprintf("the command line over %q is refused", bad), err != nil)
	}
	without := func(drop string) []string {
		var out []string
		for _, a := range good {
			if a != drop {
				out = append(out, a)
			}
		}
		return out
	}
	for _, s := range anacrolixRequiredSwitches {
		check("the planted line without "+s+" is refused", strings.Contains(anacrolixArgsRefusal(without(s)), s))
		check("the planted second "+s+" is refused", anacrolixArgsRefusal(append([]string{s}, good...)) != "")
		name, _, _ := strings.Cut(s, "=")
		on := append(without(s), name+"=true")
		check("the planted "+name+"=true is refused", strings.Contains(anacrolixArgsRefusal(on), name))
		bare := append(without(s), name)
		check("the planted bare "+name+" is refused", anacrolixArgsRefusal(bare) != "")
	}
	for _, extra := range [][]string{
		{"-utp=false"}, {"-x"}, {"--dht=false"}, {"-port", "1"}, {"-ready"},
	} {
		check(fmt.Sprintf("the planted %q is refused", extra), anacrolixArgsRefusal(append(append([]string(nil), good...), extra...)) != "")
	}

	// The environment.
	ce := anacrolixChildEnv([]string{"PATH=/bin", "GODEBUG=tlssha1=0", "GO_LOG=debug", "go_utp_logging=1",
		"TORRENT_MAX_ACTIVE_PIECE_HASHERS=1", "WEBTORRENT_MAX_WRITE_SIZE=1", "HTTPS_PROXY=http://p", "no_proxy=x",
		"GOPHER=kept?", "HOME=/old", "TORRENTX=kept"}, "HOME=/new")
	cj := "\n" + strings.Join(ce, "\n") + "\n"
	check("GO* is dropped from the harness's environment", !strings.Contains(strings.ToUpper(cj), "\nGO"))
	check("TORRENT_* and WEBTORRENT_* are dropped", !strings.Contains(cj, "TORRENT_MAX") && !strings.Contains(cj, "WEBTORRENT_"))
	check("proxy variables are dropped", !strings.Contains(strings.ToUpper(cj), "_PROXY="))
	check("the rest is kept and the extra comes last", strings.Contains(cj, "\nPATH=/bin\n") && strings.Contains(cj, "\nTORRENTX=kept\n") &&
		ce[len(ce)-1] == "HOME=/new")

	// The harness's report, in the shape it wrote on 2026-10-22.
	ready := "bit-ids/anacrolix-torrent-harness-ready/1\npid=7\ninfohash=" + strings.Repeat("ab", 20) + "\nlisten_port=51413\n" +
		"listen=tcp 0.0.0.0:51413,udp 0.0.0.0:51413\nno_dht=true\ndisable_pex=true\nno_default_port_forwarding=true\n" +
		"disable_webseeds=true\ndisable_webtorrent=true\ndisable_ipv6=true\ndisable_utp=false\ndisable_trackers=false\n" +
		"http_proxy_set=false\nice_servers=0\nself_report_peer_id=" + hex.EncodeToString([]byte("-GT0003-")) + strings.Repeat("0", 24) + "\n" +
		"self_report_bep20=-GT0003-\nself_report_http_user_agent=anacrolix-torrent/v1.61.0\n" +
		"self_report_extended_handshake_v=bit-ids/harness/anacrolix-torrent (devel) (anacrolix/torrent v1.61.0)\n" +
		"self_report_upnp_id=bit-ids/harness/anacrolix-torrent (devel)\nlibrary_default_bep20=-GT0003-\n"
	check("the report is held", anacrolixReadyRefusal(ready) == "")
	for name, bad := range map[string]string{
		"the DHT on":              strings.Replace(ready, "no_dht=true", "no_dht=false", 1),
		"PEX on":                  strings.Replace(ready, "disable_pex=true", "disable_pex=false", 1),
		"port forwarding on":      strings.Replace(ready, "no_default_port_forwarding=true", "no_default_port_forwarding=false", 1),
		"webseeds on":             strings.Replace(ready, "disable_webseeds=true", "disable_webseeds=false", 1),
		"webtorrent on":           strings.Replace(ready, "disable_webtorrent=true", "disable_webtorrent=false", 1),
		"IPv6 on":                 strings.Replace(ready, "disable_ipv6=true", "disable_ipv6=false", 1),
		"trackers off":            strings.Replace(ready, "disable_trackers=false", "disable_trackers=true", 1),
		"a proxy":                 strings.Replace(ready, "http_proxy_set=false", "http_proxy_set=true", 1),
		"an ICE server":           strings.Replace(ready, "ice_servers=0", "ice_servers=1", 1),
		"another prefix":          strings.Replace(ready, "self_report_bep20=-GT0003-", "self_report_bep20=-BI0001-", 1),
		"another user agent":      strings.Replace(ready, "anacrolix-torrent/v1.61.0\n", "bit-ids\n", 1),
		"another handshake v":     strings.Replace(ready, "(devel) (anacrolix", "v0.0.1 (anacrolix", 1),
		"an IPv6 listener":        strings.Replace(ready, "udp 0.0.0.0:51413", "udp [::]:51413", 1),
		"another port":            strings.Replace(ready, "tcp 0.0.0.0:51413", "tcp 0.0.0.0:42069", 1),
		"no listener":             strings.Replace(ready, "listen=tcp 0.0.0.0:51413,udp 0.0.0.0:51413", "listen=", 1),
		"no info hash":            strings.Replace(ready, strings.Repeat("ab", 20), "", 1),
		"a peer ID of its own":    strings.Replace(ready, hex.EncodeToString([]byte("-GT0003-")), hex.EncodeToString([]byte("-BI0001-")), 1),
		"no schema line":          strings.SplitN(ready, "\n", 2)[1],
		"no handshake self-check": strings.Replace(ready, "self_report_extended_handshake_v=", "x=", 1),
	} {
		check("a report with "+name+" is refused", anacrolixReadyRefusal(bad) != "")
	}

	// The drive, over a scratch workdir.
	work, err := os.MkdirTemp("", "anacrolix-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "anacrolix-torrent: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	never := func(time.Duration) bool { return false }
	_ = os.WriteFile(filepath.Join(work, "ready.txt"), []byte(ready), 0o644)
	listen, err := anacrolixDrive(work, never)
	check("the drive reads the report", err == nil && listen == "tcp 0.0.0.0:51413,udp 0.0.0.0:51413")
	_ = os.WriteFile(filepath.Join(work, "ready.txt"), []byte(strings.Replace(ready, "no_dht=true", "no_dht=false", 1)), 0o644)
	_, err = anacrolixDrive(work, never)
	check("the drive refuses a report with a surface on", err != nil && strings.Contains(err.Error(), "no_dht"))
	_ = os.Remove(filepath.Join(work, "ready.txt"))
	_, err = anacrolixDrive(work, func(time.Duration) bool { return true })
	check("the drive reports a harness that exited", err != nil && strings.Contains(err.Error(), "exited"))

	// A read-only tree, as the toolchain is extracted, is still removed.
	ro := filepath.Join(work, "gopath", "pkg", "mod", "golang.org", "toolchain-go1.27.3.linux-amd64", "src")
	_ = os.MkdirAll(ro, 0o755)
	_ = os.WriteFile(filepath.Join(ro, "f"), []byte("x"), 0o444)
	_ = os.Chmod(ro, 0o555)
	_ = os.Chmod(filepath.Dir(ro), 0o555)
	anacrolixRemoveTree(filepath.Join(work, "gopath"))
	_, statErr := os.Stat(filepath.Join(work, "gopath"))
	check("a read-only module cache is removed", os.IsNotExist(statErr))

	// The pin the tree carries, when this runs in a tree.
	if src, err := anacrolixHarnessSource(); err == nil {
		gm, e1 := os.ReadFile(filepath.Join(src, "go.mod"))
		gs, e2 := os.ReadFile(filepath.Join(src, "go.sum"))
		if e1 == nil && e2 == nil {
			msg := anacrolixPinRefusal(string(gm), string(gs))
			check("this tree's harness is pinned ["+msg+"]", msg == "")
		}
	}

	fmt.Printf(`{"schema":"anacrolix-torrent-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
