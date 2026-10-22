// rqbit - what both halves of the `rqbit` adapter decide, as functions with no
// platform in them, so any host can ask them through `bit-check rqbit
// --selftest`.
//
// ENGINE-03. The adapters are adapter_rqbit.go (Linux) and
// adapter_rqbit_windows.go; this file registers nothing.
//
// ⛔ AN ENGINE TARGET, AND THE PROFILE SAYS SO. The catalogue lists `rqbit` as
// `kind = "engine"`, and `TODO/engines.md` says engine results are labeled
// harness observations. The harness is the upstream `rqbit` command at the
// tag, which librqbit's own manifest calls "just a small wrapper on top of
// it"; this project commits no harness of its own. So `describe` declares
// `target_kind=library`, `capture-client` copies it into the attestation's
// `kind`, and `assemble-capture` files the record as `TargetKind::Library`.
// ⚠ That is the record's existing vocabulary: no schema field is added.
//
// -- ⭐ WHAT UPSTREAM PUBLISHES, read on 2026-10-21 -----------------------------
//
// `v9.0.1`, 2026-08-20, commit `a499d2f2...`, the newest release object and not
// a prerelease. 15 assets and no checksums document; the listing states a
// sha256 for every asset. The two this target installs:
//
//	rqbit-linux-amd64   static-pie, musl, built by `docker/Dockerfile.xx` with
//	                    `xx-cargo build --profile release-github`, OPENSSL_STATIC=1
//	rqbit.exe           built by `release-windows.yml` with
//	                    `cargo build --profile release-github`
//
// ⛔ NEITHER BUILD PASSES A FEATURE FLAG, read from both workflows at the tag, so
// both carry the `rqbit` crate's DEFAULT features: `default-tls`, `postgres`,
// `webui` and `prometheus`. crates.io's `rqbit` 9.0.1 declares the same default
// set, and `cargo install` with no feature flag builds it. That is the pairing
// this adapter makes, and `rqbitCargoRecord` refuses an install that recorded
// any other selection.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The release scheme and the routes, one declaration for both halves.
//
// ⭐ `v` AND THREE COMPONENTS, NO FLOOR. Measured over the 65 tags on
// 2026-10-21: every one is `v<major>.<minor>.<patch>`, some with a `-beta.N`
// suffix, which the scheme reads as a prerelease by version. ⚠ `v9.0.0-beta.2`
// is a release object NOT marked prerelease; its suffix is what keeps it out.
const (
	rqbitTarget        = "rqbit"
	rqbitRepo          = "ikatson/rqbit"
	rqbitTagPrefix     = "v"
	rqbitMinComponents = 3
	rqbitMaxComponents = 3
	rqbitLinuxAsset    = "rqbit-linux-amd64"
	rqbitWindowsAsset  = "rqbit.exe"
	// rqbitDigestUnpublished is the measured reason install-rootless is given.
	rqbitDigestUnpublished = "this release publishes no checksums document and no signatures; the listing states a digest for every asset"
	// rqbitCrate is the package the `package` route asks crates.io for.
	rqbitCrate = "rqbit"
	// rqbitCratesIndex is the index slug the record carries, and the source
	// cargo names in its own install record.
	rqbitCratesIndex  = "crates-io"
	rqbitCratesSource = "registry+https://github.com/rust-lang/crates.io-index"
	// rqbitDelivery names what delivered the `package` route's bytes: cargo
	// fetched the crate and its dependencies and compiled them here.
	rqbitDelivery = "cargo-install-local-build"
	// rqbitToolchain is the Rust the `package` route compiles with: this
	// repository's pin, `rust-toolchain.toml`. ⚠ Named rather than inherited,
	// because `cargo install` runs outside this tree and would otherwise get
	// the host's default toolchain. `check-bitcheck` holds the two equal.
	rqbitToolchain = "1.98.0"
	// rqbitAPIPort is the HTTP API's loopback port unless BIT_IDS_RQBIT_API_PORT
	// says otherwise; it is the build's own default for `server start`.
	rqbitAPIPort = "3030"
)

// rqbitDefaultFeatures is the `rqbit` crate's default feature set at 9.0.1, as
// crates.io's metadata and the tag's `crates/rqbit/Cargo.toml` both state it.
var rqbitDefaultFeatures = []string{"default-tls", "postgres", "webui", "prometheus"}

// rqbitDescribe is everything either half's `describe` prints after `binary=`.
func rqbitDescribe(asset string) string {
	var b strings.Builder
	// ⛔ THE RECORD'S TARGET KIND, NOT THE ADAPTER'S `kind`: `stock` says this
	// drives a product, `library` says the product is an engine's harness.
	b.WriteString("target_kind=library\n")
	fmt.Fprintf(&b, "release_repo=%s\n", rqbitRepo)
	fmt.Fprintf(&b, "release_tag_prefix=%s\n", rqbitTagPrefix)
	fmt.Fprintf(&b, "release_min_components=%d\n", rqbitMinComponents)
	fmt.Fprintf(&b, "release_max_components=%d\n", rqbitMaxComponents)
	b.WriteString("release_floor_major=-\n")
	b.WriteString("release_series=-\n")
	fmt.Fprintf(&b, "release_asset=%s\n", asset)
	fmt.Fprintf(&b, "release_digest_unpublished=%s\n", rqbitDigestUnpublished)
	// ⚠ The `package` route is `cargo install`, which compiles.
	b.WriteString("compiles=package\n")
	b.WriteString("routes=package,release\n")
	return b.String()
}

// rqbitPackageFormat is how each route delivers the build, as `Build.package`
// spells it.
//
// ⚠ Linux's two forms differ: the vendor's musl static-pie, and a build linked
// here against the host's glibc and libssl. Windows' two are one form, a PE
// both built by `cargo` with MSVC, which is how the vendor's own workflow
// builds `rqbit.exe`.
func rqbitPackageFormat(windows bool, route string) (string, bool) {
	switch {
	case !windows && route == "release":
		return "elf-binary-static", true
	case !windows && route == "package":
		return "elf-binary", true
	case windows && (route == "release" || route == "package"):
		return "pe-binary", true
	}
	return "", false
}

// rqbitCargoArgs is the `package` route's cargo command line.
//
// ⛔ NO VERSION, because crates.io's index is this route's resolver, as PyPI's
// is for `deluge` and winget's for the Windows halves. ⛔ `--locked`, so the
// crate's own `Cargo.lock` decides every dependency: measured on 2026-10-21,
// its 481 packages are all in the tag's workspace lock at the same versions.
// ⛔ NO FEATURE FLAG, so the build carries the defaults the vendor's builds do.
func rqbitCargoArgs(root string) []string {
	return []string{"install", "--locked", rqbitCrate, "--root", root}
}

// rqbitCargoEnv is the environment the `package` route's cargo runs with.
//
// ⛔ CARGO'S STATE IS THE WORKDIR'S: `CARGO_HOME` and `CARGO_TARGET_DIR` are
// made for the install, so no registry cache, no `config.toml` and no build
// the host already holds takes part, and `RUSTUP_TOOLCHAIN` names the pin.
// ⛔ AND THE VENDOR'S PROFILE. Both vendor builds use `--profile
// release-github`, which the tag's workspace manifest defines as `release`
// with `panic = "abort"`, `debug = false` and `lto = false`. A published crate
// carries no workspace profile, so `cargo install` would build cargo's default
// `release`, which differs from it in one setting: panics unwind. Measured on
// 2026-10-21: the crate's manifest has no `[profile]`.
// `CARGO_PROFILE_RELEASE_PANIC=abort` is that one setting, so both routes
// build one profile as well as one feature set.
func rqbitCargoEnv(cargoHome, target string) []string {
	return []string{"CARGO_HOME=" + cargoHome, "CARGO_TARGET_DIR=" + target, "RUSTUP_TOOLCHAIN=" + rqbitToolchain,
		"CARGO_TERM_COLOR=never", rqbitProfileOverride}
}

// rqbitProfileOverride is the one profile setting the `package` route changes.
const rqbitProfileOverride = "CARGO_PROFILE_RELEASE_PANIC=abort"

// rqbitServerArgs is the whole command line `start` gives the build.
//
// ⛔ EVERY SURFACE THE BUILD HAS, NAMED, read from `rqbit server start --help`
// and the source at 9.0.1, and driven on 2026-10-21:
//
//   - DHT is on by default: `--disable-dht`, and `--disable-dht-persistence` so
//     no routing table is read or written;
//   - UPnP port forwarding is on by default: `--disable-upnp-port-forward`;
//   - local peer discovery is on by default: `--disable-lsd`;
//   - the UPnP media server and mDNS are off by default and are not passed;
//   - session persistence is on by default: `--disable-persistence`;
//   - the HTTP API listens on loopback, `127.0.0.1:<api-port>`.
//
// ⚠ PEX HAS NO SWITCH. librqbit exchanges `ut_pex` with peers it is connected
// to, and on a contained host those are the observer's. ⚠ No update check and
// no telemetry exist in the command; the desktop app's updater is not in it.
// ⚠ The peer ID and the HTTP user agent are the build's own; nothing here sets
// them.
func rqbitServerArgs(downloads, peerPort, apiPort string) ([]string, error) {
	if downloads == "" {
		return nil, fmt.Errorf("no download directory")
	}
	if peerPort != "0" && !rqbitPortOK(peerPort) {
		return nil, fmt.Errorf("peer port [%s] is not a port", peerPort)
	}
	if !rqbitPortOK(apiPort) {
		return nil, fmt.Errorf("API port [%s] is not a port", apiPort)
	}
	args := []string{
		"--disable-dht", "--disable-dht-persistence", "--disable-upnp-port-forward", "--disable-lsd",
		"--http-api-listen-addr", "127.0.0.1:" + apiPort,
	}
	if peerPort != "0" {
		args = append(args, "--listen-port", peerPort)
	}
	return append(args, "server", "start", "--disable-persistence", downloads), nil
}

// rqbitRequiredSwitches is every switch the command line must carry once.
var rqbitRequiredSwitches = []string{"--disable-dht", "--disable-dht-persistence", "--disable-upnp-port-forward",
	"--disable-lsd", "--disable-persistence"}

// rqbitForbiddenSwitches turn a surface ON or point the build at a host that is
// not the lab, and must never be passed.
var rqbitForbiddenSwitches = []string{"--enable-upnp-server", "--enable-mdns", "--trackers-filename",
	"--blocklist-url", "--allowlist-url", "--dht-bootstrap-addrs", "--socks-url", "--persistence-location",
	"--watch-folder", "--http-api-allow-create"}

// rqbitArgsRefusal answers what is wrong with a command line, or "". ⛔ Both
// halves ask it of the line they are about to run, so a line that left a
// surface on never reaches the build.
func rqbitArgsRefusal(args []string) string {
	count := map[string]int{}
	for _, a := range args {
		name, _, _ := strings.Cut(a, "=")
		count[name]++
	}
	for _, s := range rqbitRequiredSwitches {
		if count[s] != 1 {
			return fmt.Sprintf("%s is passed %d times, not once", s, count[s])
		}
	}
	for _, s := range rqbitForbiddenSwitches {
		if count[s] > 0 {
			return s + " is passed"
		}
	}
	addr := ""
	for i, a := range args {
		if a == "--http-api-listen-addr" && i+1 < len(args) {
			addr = args[i+1]
		}
		if v, ok := strings.CutPrefix(a, "--http-api-listen-addr="); ok {
			addr = v
		}
	}
	if count["--http-api-listen-addr"] != 1 || !strings.HasPrefix(addr, "127.0.0.1:") ||
		!rqbitPortOK(strings.TrimPrefix(addr, "127.0.0.1:")) {
		return fmt.Sprintf("the HTTP API is not on one loopback address: [%s]", addr)
	}
	n := len(args)
	if n < 4 || args[n-4] != "server" || args[n-3] != "start" {
		return "the command line does not end in `server start --disable-persistence <folder>`"
	}
	return ""
}

// rqbitChildEnv is the environment the build is started with.
//
// ⛔ EVERY `RQBIT_*` VARIABLE IS DROPPED. Each of the build's switches also
// reads one (`RQBIT_UPNP_SERVER_ENABLE`, `RQBIT_DHT_DISABLE`, ...), so a host
// that carried one would change what the command line says. ⚠ `RUST_LOG` and
// `CORS_ALLOW_REGEXP` are dropped too: the first changes what the build logs,
// the second who its API answers. `extra` is appended after the filter.
func rqbitChildEnv(environ []string, extra ...string) []string {
	var out []string
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "RQBIT_") || upper == "RUST_LOG" || upper == "CORS_ALLOW_REGEXP" {
			continue
		}
		out = append(out, kv)
	}
	return append(out, extra...)
}

func rqbitPortOK(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 65535 && strconv.Itoa(n) == s
}

var rqbitVersionRe = regexp.MustCompile(`^rqbit ([0-9]+\.[0-9]+\.[0-9]+)$`)

// rqbitParseVersion reads `rqbit --version`, which clap prints as `rqbit
// <CARGO_PKG_VERSION>`: `rqbit 9.0.1`, measured on both Linux builds. ⚠ The
// name is clap's `name` and not the executable's, so `rqbit.exe` and
// `rqbit-linux-amd64` both answer `rqbit`. A line that is not that shape is
// refused rather than trimmed into a version.
func rqbitParseVersion(output string) (string, error) {
	for _, l := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		if m := rqbitVersionRe.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("rqbit reported no parseable version: [%s]", strings.TrimSpace(firstLineOf(output)))
}

// rqbitAPIRoot reads `GET /`, which librqbit answers with `server` and its own
// `CARGO_PKG_VERSION`: the LIBRARY's version, spoken by the running build.
func rqbitAPIRoot(body []byte) (string, error) {
	var root struct {
		Server  string          `json:"server"`
		Version string          `json:"version"`
		APIs    json.RawMessage `json:"apis"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return "", fmt.Errorf("the API root is not JSON: %v", err)
	}
	if root.Server != "rqbit" {
		return "", fmt.Errorf("the API root names server [%s], not rqbit", root.Server)
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(root.Version) {
		return "", fmt.Errorf("the API root names no library version: [%s]", root.Version)
	}
	if len(root.APIs) == 0 {
		return "", fmt.Errorf("the API root lists no APIs")
	}
	return root.Version, nil
}

// rqbitAddAnswer reads `POST /torrents`'s answer and returns the torrent's id.
func rqbitAddAnswer(body []byte) (int, error) {
	var a struct {
		ID      *int `json:"id"`
		Details struct {
			InfoHash string `json:"info_hash"`
		} `json:"details"`
	}
	if err := json.Unmarshal(body, &a); err != nil {
		return 0, fmt.Errorf("the add answer is not JSON: %v", err)
	}
	if a.ID == nil || *a.ID < 0 {
		return 0, fmt.Errorf("the add answer names no torrent id")
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(a.Details.InfoHash) {
		return 0, fmt.Errorf("the add answer names no info hash: [%s]", a.Details.InfoHash)
	}
	return *a.ID, nil
}

// rqbitCargoInstall is what cargo's own install record says it built.
type rqbitCargoInstall struct {
	version, profile, target, rustc string
}

// rqbitCargoRecord reads `<root>/.crates2.json`, the record cargo writes of
// every crate it installed into a root.
//
// ⭐ THE BUILD'S FEATURE SELECTION, IN CARGO'S OWN WORDS: `features`,
// `all_features`, `no_default_features`, `profile`, `target` and `rustc`.
// ⛔ Exactly one install, of `rqbit` from crates.io, with no explicit feature
// and neither `--all-features` nor `--no-default-features`: the default set,
// which is the release's. Anything else is refused, because the pair would no
// longer be one build's two routes.
func rqbitCargoRecord(body []byte) (rqbitCargoInstall, error) {
	var rec struct {
		Installs map[string]struct {
			Bins              []string `json:"bins"`
			Features          []string `json:"features"`
			AllFeatures       bool     `json:"all_features"`
			NoDefaultFeatures bool     `json:"no_default_features"`
			Profile           string   `json:"profile"`
			Target            string   `json:"target"`
			Rustc             string   `json:"rustc"`
		} `json:"installs"`
	}
	if err := json.Unmarshal(body, &rec); err != nil {
		return rqbitCargoInstall{}, fmt.Errorf("cargo's install record is not JSON: %v", err)
	}
	if len(rec.Installs) != 1 {
		var keys []string
		for k := range rec.Installs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return rqbitCargoInstall{}, fmt.Errorf("cargo's install record names %d installs, not one: %s", len(rec.Installs), strings.Join(keys, "; "))
	}
	for key, in := range rec.Installs {
		f := strings.Fields(key)
		if len(f) != 3 || f[0] != rqbitCrate || f[2] != "("+rqbitCratesSource+")" {
			return rqbitCargoInstall{}, fmt.Errorf("cargo installed [%s], not %s from crates.io", key, rqbitCrate)
		}
		if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(f[1]) {
			return rqbitCargoInstall{}, fmt.Errorf("cargo installed %s at [%s], which is not a stable version", rqbitCrate, f[1])
		}
		// ⚠ Windows' cargo records the executable with its suffix: capture-client
		// run 39777039760's Windows package lane built rqbit and was refused for
		// naming `rqbit.exe`. The name or the name plus `.exe`, nothing else.
		if len(in.Bins) != 1 || (in.Bins[0] != rqbitCrate && in.Bins[0] != rqbitCrate+".exe") {
			return rqbitCargoInstall{}, fmt.Errorf("cargo installed the executables %v, not one %s", in.Bins, rqbitCrate)
		}
		if len(in.Features) != 0 || in.AllFeatures || in.NoDefaultFeatures {
			return rqbitCargoInstall{}, fmt.Errorf("cargo built a feature selection that is not the default: features=%v all_features=%v no_default_features=%v",
				in.Features, in.AllFeatures, in.NoDefaultFeatures)
		}
		if in.Profile != "release" {
			return rqbitCargoInstall{}, fmt.Errorf("cargo built profile [%s], not release", in.Profile)
		}
		if in.Target == "" || in.Rustc == "" {
			return rqbitCargoInstall{}, fmt.Errorf("cargo's install record names no target or no compiler")
		}
		return rqbitCargoInstall{version: f[1], profile: in.Profile, target: in.Target,
			rustc: firstLineOf(in.Rustc)}, nil
	}
	return rqbitCargoInstall{}, fmt.Errorf("unreachable")
}

// rqbitFeaturesNote is `build-features.txt`, what the `package` route records
// of the build it made, beside how the release's set was established.
func rqbitFeaturesNote(in rqbitCargoInstall, crateSHA string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "crate=%s\nversion=%s\nsource=%s\ncrate_sha256=%s\n", rqbitCrate, in.version, rqbitCratesSource, crateSHA)
	b.WriteString("explicit_features=none\nall_features=false\nno_default_features=false\n")
	fmt.Fprintf(&b, "default_features=%s\n", strings.Join(rqbitDefaultFeatures, ","))
	fmt.Fprintf(&b, "profile=%s\nprofile_override=%s\ntarget=%s\nrustc=%s\nlocked=true\n", in.profile, rqbitProfileOverride, in.target, in.rustc)
	b.WriteString("release_profile=release-github: release with panic=abort, debug=false, lto=false, from the tag's workspace Cargo.toml\n")
	b.WriteString("release_features=the vendor's workflows at the tag pass no feature flag, so the release carries the same defaults\n")
	return b.String()
}

// -- the HTTP API, on loopback, for both halves ---------------------------------

// rqbitClient never reaches a proxy: every request it makes is to loopback.
// ⛔ A proxy in the environment would send a capture's own control channel off
// the host.
var rqbitClient = &http.Client{
	Timeout:   20 * time.Second,
	Transport: &http.Transport{Proxy: nil},
}

// rqbitAPI is one request to the build's API, answering its status and body.
func rqbitAPI(port, method, path, contentType string, body []byte) (int, []byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://127.0.0.1:"+port+path, r)
	if err != nil {
		return 0, nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := rqbitClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, answer, err
}

// rqbitAPIPortFor is the API port, the default or BIT_IDS_RQBIT_API_PORT.
func rqbitAPIPortFor() string {
	if p := os.Getenv("BIT_IDS_RQBIT_API_PORT"); p != "" {
		return p
	}
	return rqbitAPIPort
}

// rqbitDrive is what `start` does once the build is launched, on either
// platform: wait for its API, record what it says it is and which default
// features it serves, and hand it the observer's torrent.
//
// ⚠ A BOUNDED WAIT ON THE WORK, NOT ON THE CLOCK: `gone` says whether the build
// has exited, so a build that refused its command line is reported at once.
//
// ⛔ THE FEATURE PROBE IS THE RUNNING BUILD'S, NOT A MANIFEST'S. `/web/` is
// routed only when `webui` was compiled in, `/metrics` only with `prometheus`,
// so both answering 200 is two of the release's four defaults observed. A build
// missing either is refused: it is not the release's feature set.
func rqbitDrive(workdir, port, torrent string, gone func(time.Duration) bool) (string, error) {
	var root []byte
	for waited := 0; ; waited++ {
		if code, body, err := rqbitAPI(port, http.MethodGet, "/", "", nil); err == nil && code == http.StatusOK {
			root = body
			break
		}
		if gone(time.Second) {
			return "", fmt.Errorf("rqbit exited before its HTTP API answered; %s says why", filepath.Join(workdir, "client.out"))
		}
		if waited >= 30 {
			return "", fmt.Errorf("rqbit's HTTP API did not answer on 127.0.0.1:%s within 30 seconds", port)
		}
	}
	_ = os.WriteFile(filepath.Join(workdir, "api-root.json"), root, 0o644)
	library, err := rqbitAPIRoot(root)
	if err != nil {
		return "", err
	}
	web, _, _ := rqbitAPI(port, http.MethodGet, "/web/", "", nil)
	metrics, _, _ := rqbitAPI(port, http.MethodGet, "/metrics", "", nil)
	note := fmt.Sprintf("library_version=%s\nweb_status=%d\nmetrics_status=%d\n", library, web, metrics)
	_ = os.WriteFile(filepath.Join(workdir, "features.txt"), []byte(note), 0o644)
	if msg := rqbitFeatureRefusal(web, metrics); msg != "" {
		return "", fmt.Errorf("%s", msg)
	}
	body, err := os.ReadFile(torrent)
	if err != nil || len(body) == 0 {
		return "", fmt.Errorf("the torrent at %s could not be read", torrent)
	}
	// ⭐ THE TORRENT GOES IN AS BYTES, `is_url=false`, so the build parses the
	// object the observer wrote and fetches nothing to get it.
	code, answer, err := rqbitAPI(port, http.MethodPost, "/torrents?overwrite=true&is_url=false", "application/x-bittorrent", body)
	_ = os.WriteFile(filepath.Join(workdir, "api-add.json"), answer, 0o644)
	if err != nil {
		return "", fmt.Errorf("rqbit would not accept the torrent: %v", err)
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("rqbit refused the torrent: HTTP %d: %s", code, strings.TrimSpace(firstLineOf(string(answer))))
	}
	id, err := rqbitAddAnswer(answer)
	if err != nil {
		return "", err
	}
	_ = os.WriteFile(filepath.Join(workdir, "torrent-id"), []byte(strconv.Itoa(id)+"\n"), 0o644)
	return library, nil
}

// rqbitFeatureRefusal is the feature probe's verdict, or "".
func rqbitFeatureRefusal(web, metrics int) string {
	if web != http.StatusOK {
		return fmt.Sprintf("the build answers /web/ with HTTP %d, so it lacks the default `webui` feature the release carries", web)
	}
	if metrics != http.StatusOK {
		return fmt.Sprintf("the build answers /metrics with HTTP %d, so it lacks the default `prometheus` feature the release carries", metrics)
	}
	return ""
}

// rqbitStateListing lists every file under dir, relative and with its size,
// one per line, for `stop` to record before it removes the directory.
func rqbitStateListing(dir string) string {
	var b strings.Builder
	_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil || p == dir {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if fi.IsDir() {
			fmt.Fprintf(&b, "%s/\n", filepath.ToSlash(rel))
		} else {
			fmt.Fprintf(&b, "%s %d\n", filepath.ToSlash(rel), fi.Size())
		}
		return nil
	})
	return b.String()
}

// -- the selftest -------------------------------------------------------------

// rqbitSelftest is `bit-check rqbit --selftest`: every decision above asked
// over a case whose answer is known, with the defects planted, on any host.
func rqbitSelftest(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "rqbit: usage: bit-check rqbit --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "rqbit: FAILED %s\n", name)
		}
	}

	// The description: a harness-labeled target, this repository's scheme.
	d := rqbitDescribe(rqbitLinuxAsset)
	for _, want := range []string{"target_kind=library", "release_repo=ikatson/rqbit", "release_tag_prefix=v",
		"release_min_components=3", "release_max_components=3", "release_floor_major=-", "release_series=-",
		"release_asset=rqbit-linux-amd64", "compiles=package", "routes=package,release"} {
		check("describe carries "+want, strings.Contains(d, want+"\n"))
	}
	check("describe declares a digest disposition, and only one",
		strings.Count(d, "release_digest_unpublished=") == 1 && !strings.Contains(d, "release_digest_asset="))
	check("the Windows asset is the vendor's console build",
		strings.Contains(rqbitDescribe(rqbitWindowsAsset), "release_asset=rqbit.exe\n"))
	check("the desktop app is never the asset", !strings.Contains(d, "desktop"))

	// The packaging slugs.
	for _, c := range []struct {
		win         bool
		route, want string
		ok          bool
	}{{false, "release", "elf-binary-static", true}, {false, "package", "elf-binary", true},
		{false, "source", "", false}, {true, "release", "pe-binary", true}, {true, "package", "pe-binary", true},
		{true, "source", "", false}} {
		f, ok := rqbitPackageFormat(c.win, c.route)
		check(fmt.Sprintf("windows=%v route %s packages as [%s]", c.win, c.route, c.want), ok == c.ok && f == c.want)
	}

	// cargo's command line: the index resolves, the lock decides, no feature flag.
	ca := strings.Join(rqbitCargoArgs("/p"), " ")
	check("cargo installs rqbit --locked into the route's root", ca == "install --locked rqbit --root /p")
	check("cargo is passed no version, so the index resolves it", !strings.Contains(ca, "@") && !strings.Contains(ca, "--version"))
	check("cargo is passed no feature flag", !strings.Contains(ca, "feature"))
	ce := "\n" + strings.Join(rqbitCargoEnv("/w/cargo-home", "/w/target"), "\n") + "\n"
	for _, want := range []string{"CARGO_HOME=/w/cargo-home", "CARGO_TARGET_DIR=/w/target", "RUSTUP_TOOLCHAIN=" + rqbitToolchain,
		"CARGO_PROFILE_RELEASE_PANIC=abort"} {
		check("cargo runs with "+want, strings.Contains(ce, "\n"+want+"\n"))
	}
	check("cargo changes no other profile setting", strings.Count(ce, "CARGO_PROFILE_") == 1)

	// The command line: every surface off, the API on loopback, refused otherwise.
	good, err := rqbitServerArgs("/w/downloads", "51413", "3030")
	check("the command line is built", err == nil)
	check("the command line is held", rqbitArgsRefusal(good) == "")
	check("the peer port is passed", strings.Contains(strings.Join(good, " "), "--listen-port 51413"))
	zero, _ := rqbitServerArgs("/w/downloads", "0", "3030")
	check("peer port 0 leaves the port to the build", !strings.Contains(strings.Join(zero, " "), "--listen-port") && rqbitArgsRefusal(zero) == "")
	check("the folder is the last argument", good[len(good)-1] == "/w/downloads")
	for _, bad := range [][3]string{
		{"/w", "x", "3030"}, {"/w", "70000", "3030"}, {"/w", "51413", "0"}, {"", "51413", "3030"}, {"/w", "051413", "3030"},
	} {
		_, err := rqbitServerArgs(bad[0], bad[1], bad[2])
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
	for _, s := range rqbitRequiredSwitches {
		check("the planted line without "+s+" is refused", strings.Contains(rqbitArgsRefusal(without(s)), s))
		twice := append([]string{s}, good...)
		check("the planted second "+s+" is refused", strings.Contains(rqbitArgsRefusal(twice), s))
	}
	for _, s := range rqbitForbiddenSwitches {
		on := append([]string{s}, good...)
		check("the planted "+s+" is refused", strings.Contains(rqbitArgsRefusal(on), s))
		eq := append([]string{s + "=x"}, good...)
		check("the planted "+s+"=x is refused", strings.Contains(rqbitArgsRefusal(eq), s))
	}
	swap := func(from, to string) []string {
		out := append([]string(nil), good...)
		for i := range out {
			if out[i] == from {
				out[i] = to
			}
		}
		return out
	}
	check("an API on every interface is refused", rqbitArgsRefusal(swap("127.0.0.1:3030", "0.0.0.0:3030")) != "")
	check("an API with no port is refused", rqbitArgsRefusal(swap("127.0.0.1:3030", "127.0.0.1:")) != "")
	check("a line that is not `server start` is refused", rqbitArgsRefusal(swap("start", "download")) != "")
	check("a second API address is refused", rqbitArgsRefusal(append([]string{"--http-api-listen-addr", "127.0.0.1:1"}, good...)) != "")

	// The environment: every RQBIT_* dropped, whatever its case.
	env := rqbitChildEnv([]string{"PATH=/bin", "RQBIT_UPNP_SERVER_ENABLE=1", "rqbit_dht_disable=", "RUST_LOG=trace",
		"CORS_ALLOW_REGEXP=.*", "HOME=/old", "RQBITX=kept"}, "HOME=/new")
	j := "\n" + strings.Join(env, "\n") + "\n"
	check("RQBIT_* is dropped from the build's environment", !strings.Contains(strings.ToUpper(j), "\nRQBIT_"))
	check("RUST_LOG and CORS_ALLOW_REGEXP are dropped", !strings.Contains(j, "\nRUST_LOG=") && !strings.Contains(j, "\nCORS_ALLOW_REGEXP="))
	check("the rest is kept and the extra comes last", strings.Contains(j, "\nPATH=/bin\n") && strings.Contains(j, "\nRQBITX=kept\n") &&
		env[len(env)-1] == "HOME=/new")

	// The build's answers.
	for in, want := range map[string]string{"rqbit 9.0.1\n": "9.0.1", "rqbit 9.0.1": "9.0.1", "rqbit 10.2.0\r\n": "10.2.0"} {
		v, err := rqbitParseVersion(in)
		check("the answer ["+strings.TrimSpace(in)+"] reads as "+want, err == nil && v == want)
	}
	for _, in := range []string{"", "rqbit", "rqbit 9.0", "rqbit 9.0.1-beta.2", "rqbit v9.0.1", "rqbit-desktop 9.0.1",
		"error: unexpected argument '--version'", "rqbit 9.0.1 extra"} {
		_, err := rqbitParseVersion(in)
		check("the answer ["+in+"] is refused", err != nil)
	}
	rootJSON := `{"apis":{"GET /":"list all available APIs"},"server":"rqbit","version":"9.0.1"}`
	lib, err := rqbitAPIRoot([]byte(rootJSON))
	check("the API root names library 9.0.1", err == nil && lib == "9.0.1")
	for name, bad := range map[string]string{
		"another server": strings.Replace(rootJSON, `"server":"rqbit"`, `"server":"qbt"`, 1),
		"no version":     strings.Replace(rootJSON, `"version":"9.0.1"`, `"version":""`, 1),
		"no APIs":        `{"server":"rqbit","version":"9.0.1"}`,
		"not JSON":       rootJSON[:20],
	} {
		_, err := rqbitAPIRoot([]byte(bad))
		check("an API root with "+name+" is refused", err != nil)
	}
	add := `{"id":0,"details":{"info_hash":"` + strings.Repeat("ab", 20) + `","name":"x"},"output_folder":"/w","seen_peers":null}`
	id, err := rqbitAddAnswer([]byte(add))
	check("the add answer names torrent 0", err == nil && id == 0)
	for name, bad := range map[string]string{
		"no id":        strings.Replace(add, `"id":0,`, ``, 1),
		"a null id":    strings.Replace(add, `"id":0`, `"id":null`, 1),
		"no info hash": strings.Replace(add, strings.Repeat("ab", 20), "", 1),
		"not JSON":     add[:10],
	} {
		_, err := rqbitAddAnswer([]byte(bad))
		check("an add answer with "+name+" is refused", err != nil)
	}
	check("both default features served are held", rqbitFeatureRefusal(200, 200) == "")
	check("a build with no webui is refused", strings.Contains(rqbitFeatureRefusal(404, 200), "webui"))
	check("a build with no prometheus is refused", strings.Contains(rqbitFeatureRefusal(200, 404), "prometheus"))

	// cargo's install record, in the shape cargo 1.98.0 wrote on 2026-10-21.
	rec := `{"installs":{"rqbit 9.0.1 (registry+https://github.com/rust-lang/crates.io-index)":{"version_req":null,"bins":["rqbit"],"features":[],"all_features":false,"no_default_features":false,"profile":"release","target":"x86_64-unknown-linux-gnu","rustc":"rustc 1.98.0 (88d9e12ae 2026-08-18)\nbinary: rustc\n"}}}`
	in, err := rqbitCargoRecord([]byte(rec))
	check("cargo's record is read", err == nil && in.version == "9.0.1" && in.profile == "release" &&
		in.target == "x86_64-unknown-linux-gnu" && in.rustc == "rustc 1.98.0 (88d9e12ae 2026-08-18)")
	winRec := strings.Replace(strings.Replace(rec, `"bins":["rqbit"]`, `"bins":["rqbit.exe"]`, 1), "x86_64-unknown-linux-gnu", "x86_64-pc-windows-msvc", 1)
	_, err = rqbitCargoRecord([]byte(winRec))
	check("cargo's Windows record, naming rqbit.exe, is read", err == nil)
	note := rqbitFeaturesNote(in, strings.Repeat("0", 64))
	check("the features note names the default set", strings.Contains(note, "default_features=default-tls,postgres,webui,prometheus\n"))
	check("the features note names the compiler", strings.Contains(note, "rustc=rustc 1.98.0 (88d9e12ae 2026-08-18)\n"))
	check("the features note names the profile override", strings.Contains(note, "profile_override=CARGO_PROFILE_RELEASE_PANIC=abort\n"))
	for name, bad := range map[string]string{
		"an explicit feature":   strings.Replace(rec, `"features":[]`, `"features":["rust-tls"]`, 1),
		"--all-features":        strings.Replace(rec, `"all_features":false`, `"all_features":true`, 1),
		"--no-default-features": strings.Replace(rec, `"no_default_features":false`, `"no_default_features":true`, 1),
		"a debug profile":       strings.Replace(rec, `"profile":"release"`, `"profile":"dev"`, 1),
		"another crate":         strings.Replace(rec, `"rqbit 9.0.1`, `"rqbit-desktop 9.0.1`, 1),
		"a git source":          strings.Replace(rec, `(registry+https://github.com/rust-lang/crates.io-index)`, `(git+https://github.com/ikatson/rqbit#a499d2f)`, 1),
		"a prerelease":          strings.Replace(rec, `"rqbit 9.0.1 `, `"rqbit 9.0.0-rc.0 `, 1),
		"a second executable":   strings.Replace(rec, `"bins":["rqbit"]`, `"bins":["rqbit","x"]`, 1),
		"another executable":    strings.Replace(rec, `"bins":["rqbit"]`, `"bins":["rqbit2.exe"]`, 1),
		"no compiler":           strings.Replace(rec, `"rustc":"rustc 1.98.0 (88d9e12ae 2026-08-18)\nbinary: rustc\n"`, `"rustc":""`, 1),
		"a second install":      strings.Replace(rec, `}}}`, `},"x 1.0.0 (registry+https://github.com/rust-lang/crates.io-index)":{}}}`, 1),
		"no install at all":     `{"installs":{}}`,
		"something that is not": rec[:30],
	} {
		_, err := rqbitCargoRecord([]byte(bad))
		check("cargo's record with "+name+" is refused", err != nil)
	}

	// The state listing over a scratch tree.
	work, err := os.MkdirTemp("", "rqbit-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "rqbit: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	_ = os.MkdirAll(filepath.Join(work, ".local", "share", "com.rqbit.session"), 0o755)
	_ = os.WriteFile(filepath.Join(work, ".local", "share", "com.rqbit.session", "session.json"), []byte("{}"), 0o644)
	listing := rqbitStateListing(work)
	check("the state listing names what the build left", strings.Contains(listing, ".local/share/com.rqbit.session/session.json 2\n"))
	check("an empty directory lists nothing", rqbitStateListing(filepath.Join(work, "absent")) == "")

	fmt.Printf(`{"schema":"rqbit-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
