// libtorrent - what the `libtorrent` adapter decides with no process in it,
// so any host can ask the decisions through `bit-check libtorrent --selftest`.
//
// ENGINE-01. The adapter is adapter_libtorrent.go, Linux only; this file
// registers nothing.
//
// ⛔ A LIBRARY, SO THE PRODUCT IS A HARNESS THIS TREE COMMITS. libtorrent-
// rasterbar is C++ an embedding program links, and the one form either route
// delivers that runs on its own is the library's Python binding. The harness
// is tools/harness/libtorrent/harness.py, the smallest program that runs
// `lt.session` with the library's default settings on one torrent.
// `describe` declares `target_kind=library`, as rqbit's and anacrolix's do,
// and the record files a `library` target: no schema field is added.
//
// -- ⭐ WHAT UPSTREAM PUBLISHES, read on 2026-10-22 -----------------------------
//
// Two stable series at once. RC_2_0's newest is `v2.0.15`, 2026-09-25, commit
// `1eb18fae...`, whose release carries the source tarball alone. RC_2_1's is
// `v2.1.2`, the same day, whose release also carries wheels. PyPI's
// `libtorrent` names 2.1.1 as its newest and carries 2.0.15, cp39 to cp313;
// it does not carry 2.1.2. ⛔ So the target is the 2.0 series, where PyPI and
// the tags agree: a 2.1 pair on PyPI would measure a superseded 2.1.1.
//
// -- ⭐ THE TWO ROUTES, ONE VERSION, ONE HARNESS ---------------------------------
//
// ⛔ `package`: PyPI's index resolves `libtorrent>=2.0,<2.1` among wheels
// only, and pip's report must name the pin; the wheel the vendor's
// cibuildwheel workflow built is installed into a venv. ⛔ `source`: the tag
// `resolve-source` selected, cloned from GitHub, built into a wheel by the
// tag's own setup.py with no option given, which is the vendor's
// cibuildwheel path: b2 with `deprecated-functions=on boost-link=static
// libtorrent-link=static crypto=openssl variant=release`, against Boost 1.83.0
// from archives.boost.io, the version and the origin the vendor's
// `setup_boost.sh` names. ⚠ OpenSSL is the host's static libcrypto on the
// source route and a static 3.5.0 in the wheel; both link it in.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	libtorrentTarget = "libtorrent"
	libtorrentRepo   = "arvidn/libtorrent"
	// libtorrentPin is the version both routes must install: RC_2_0's newest
	// stable tag on 2026-10-22, and PyPI's newest 2.0 release the same day.
	libtorrentPin          = "2.0.15"
	libtorrentPinTag       = "v2.0.15"
	libtorrentPinnedCommit = "1eb18faeae156d8dbbab42935c082f8b81f50989"
	// libtorrentLibraryVersion is `lt.__version__` at the pin, which carries
	// LIBTORRENT_VERSION's fourth component.
	libtorrentLibraryVersion = "2.0.15.0"
	libtorrentPyPIName       = "libtorrent"
	// libtorrentPipSpec is the package route's request: the series, never the
	// pin, so PyPI's index chooses the version and the report says which.
	libtorrentPipSpec = "libtorrent>=2.0,<2.1"
	// The Boost the vendor's wheels build with, from where its setup_boost.sh
	// fetches it; the digest is the archive's own `.json` statement.
	libtorrentBoostVersion      = "1.83.0"
	libtorrentBoostURL          = "https://archives.boost.io/release/1.83.0/source/boost_1_83_0.tar.gz"
	libtorrentBoostPinnedSha256 = "c0685b68dd44cc46574cce86c4e17c0f611b15e195be9848dfd0769a0a207628"
	// libtorrentDefaultsPinnedSha256 is the sha256 of every default setting the
	// build reports, one `name=value` a line, sorted. ⭐ Measured on
	// 2026-10-22 out of PyPI's cp313 wheel AND out of the tag's own build, which
	// agreed line for line over 258 settings: a compile-time define that moved
	// a default would move it.
	libtorrentDefaultsPinnedSha256 = "2ec9886658383d6ae903489db363f8df404cf7d53f71646da8122e59198a3f70"
	libtorrentFingerprint          = "-LT20F0-"
	libtorrentUserAgent            = "libtorrent/2.0.15.0"
	libtorrentHarnessDir           = "tools/harness/libtorrent"
	libtorrentHarnessFile          = "harness.py"
)

// libtorrentDescribe is everything `describe` prints after `binary=`.
//
// ⛔ THE SCHEME READS THE 2.0 SERIES ALONE. The prefix is `v2.0.` and a
// version is one component, so `v2.1.2` and every `v1.2.x` are another
// series's tags, `foreign_tag`, and `v2.0.15` is selected. ⚠ No
// `release_asset`: the route that compiles is the tag, and the release's
// tarball is not fetched.
func libtorrentDescribe() string {
	var b strings.Builder
	b.WriteString("target_kind=library\n")
	fmt.Fprintf(&b, "release_repo=%s\n", libtorrentRepo)
	b.WriteString("release_tag_prefix=v2.0.\n")
	b.WriteString("release_min_components=1\n")
	b.WriteString("release_max_components=1\n")
	b.WriteString("release_floor_major=-\n")
	b.WriteString("compiles=source\n")
	b.WriteString("routes=package,source\n")
	return b.String()
}

// libtorrentPackageFormat is what each route delivers, as Deluge's routes
// spell it: the vendor's wheel, or a wheel this host built from the tag.
func libtorrentPackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "python-wheel", true
	case "source":
		return "python-source", true
	}
	return "", false
}

// libtorrentFields reads a `schema` line and `key=value` lines, refusing a
// key twice.
func libtorrentFields(body, schema string) (map[string]string, error) {
	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(body, "\r", ""), "\n"), "\n")
	if lines[0] != schema {
		return nil, fmt.Errorf("the harness's answer does not begin with %s: [%s]", schema, lines[0])
	}
	m := map[string]string{}
	for _, l := range lines[1:] {
		k, v, ok := strings.Cut(l, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("the harness's answer carries the line [%s]", l)
		}
		if _, dup := m[k]; dup {
			return nil, fmt.Errorf("the harness's answer names %s twice", k)
		}
		m[k] = v
	}
	return m, nil
}

// libtorrentAnswerRefusal is what is wrong with the harness's `--version`, or
// "". ⛔ THE BUILD'S FEATURES, READ OUT OF THE BINDING THAT RUNS: the pinned
// version, the symbols each define compiles in, OpenSSL linked in, and every
// default setting by digest, so the two routes are one feature set or one of
// them is refused at install.
func libtorrentAnswerRefusal(m map[string]string) string {
	for _, kv := range [][2]string{
		{"library", "libtorrent"},
		{"library_version", libtorrentLibraryVersion},
		{"python_implementation", "cpython"},
		{"abi_version_1", "true"},
		{"dht", "true"},
		{"logging", "true"},
		{"i2p", "true"},
		{"encryption", "true"},
		{"openssl", "static"},
		{"openssl_mapped", "-"},
		{"default_settings_sha256", libtorrentDefaultsPinnedSha256},
		{"default.user_agent", libtorrentUserAgent},
		{"default.peer_fingerprint", libtorrentFingerprint},
		{"default.handshake_client_version", "-"},
	} {
		if m[kv[0]] != kv[1] {
			return fmt.Sprintf("the build reports %s=[%s], not %s", kv[0], m[kv[0]], kv[1])
		}
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(m["module_sha256"]) {
		return "the build reports no digest of its module"
	}
	return ""
}

// libtorrentRecordVersion is the record's spelling of `lt.__version__`: the
// fourth component is LIBTORRENT_VERSION's and is dropped only when it is 0.
func libtorrentRecordVersion(lib string) (string, error) {
	f := strings.Split(lib, ".")
	if len(f) != 4 || f[3] != "0" || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.0$`).MatchString(lib) {
		return "", fmt.Errorf("the library reports version [%s], which is not x.y.z.0", lib)
	}
	return strings.Join(f[:3], "."), nil
}

// libtorrentPinRefusal is what is wrong with the version an index or a tag
// named, or "". ⛔ A route never upgrades the pin: a newer 2.0 release is
// re-pinned in one change, with its defaults digest measured again.
func libtorrentPinRefusal(from, version string) string {
	if version != libtorrentPin {
		return fmt.Sprintf("%s named %s and this adapter pins %s; re-pin it, and its defaults digest, to the new release", from, version, libtorrentPin)
	}
	return ""
}

// libtorrentB2Line is the b2 command line the tag's setup.py ran, out of
// `pip wheel -v`'s log: the one line that begins `b2 ` and names the module's
// install target.
func libtorrentB2Line(log string) (string, error) {
	found := ""
	for _, l := range strings.Split(strings.ReplaceAll(log, "\r", ""), "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "b2 ") && strings.Contains(t, " install_module") {
			if found != "" && found != t {
				return "", fmt.Errorf("the build ran b2 twice, differently")
			}
			found = t
		}
	}
	if found == "" {
		return "", fmt.Errorf("the build log names no b2 command line")
	}
	return found, nil
}

// libtorrentB2Refusal is what is wrong with that line, or "". ⛔ THE VENDOR'S
// FEATURE SET, which setup.py adds when it is given none: the deprecated API,
// Boost and libtorrent linked static, OpenSSL as the crypto, a release build.
// Anything else on it that sets a feature is refused.
func libtorrentB2Refusal(line string) string {
	want := []string{"deprecated-functions=on", "boost-link=static", "libtorrent-link=static", "crypto=openssl",
		"variant=release", "cxxstd=14", "address-model=64", "libtorrent-python=on", "install_module"}
	seen := map[string]int{}
	for _, f := range strings.Fields(line)[1:] {
		switch {
		case strings.HasPrefix(f, "--project-config="), strings.HasPrefix(f, "library-path="),
			strings.HasPrefix(f, "python-install-path="), strings.HasPrefix(f, "python="):
			continue
		}
		seen[f]++
		known := false
		for _, w := range want {
			if f == w {
				known = true
			}
		}
		if !known {
			return "the build passed b2 [" + f + "], which the vendor's wheel build does not"
		}
	}
	for _, w := range want {
		if seen[w] != 1 {
			return fmt.Sprintf("the build passed b2 %s %d times, not once", w, seen[w])
		}
	}
	return ""
}

// libtorrentBuildEnv is the source route's environment for `pip wheel`.
// ⛔ Every compiler and Boost variable the host carries is dropped, and the
// Boost the route fetched is the one b2 finds.
func libtorrentBuildEnv(environ []string, boost string) []string {
	var out []string
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		switch {
		case strings.HasPrefix(upper, "BOOST"), strings.HasPrefix(upper, "PYTHON"), upper == "PATH",
			upper == "CC", upper == "CXX", upper == "CFLAGS", upper == "CXXFLAGS", upper == "CPPFLAGS",
			upper == "LDFLAGS", upper == "B2", upper == "BJAM", strings.HasPrefix(upper, "PIP_"):
			continue
		}
		out = append(out, kv)
	}
	path := ""
	for _, kv := range environ {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	return append(out, "BOOST_ROOT="+boost, "BOOST_BUILD_PATH="+filepath.Join(boost, "tools", "build"),
		"PATH="+boost+string(os.PathListSeparator)+path)
}

// libtorrentRequiredSwitches is every switch the harness must be passed, once.
// ⛔ Each is a surface the library's default settings have ON, read from
// `lt.default_settings()` at the pin and from the harness's own report:
//
//   - `--no-dht`: the DHT, bootstrapping from `dht.libtorrent.org:25401`;
//   - `--no-lsd`: local service discovery, multicast on the LAN;
//   - `--no-upnp` and `--no-natpmp`: port mapping, each its own setting;
//   - `--no-pex`: the `ut_pex` plugin, one of the session's three defaults;
//   - `--ipv4-only`: the default listens on `[::]` as well. ⚠ This host boots
//     with IPv6 disabled, and the capture's lab carries IPv4; off on every
//     host, so the profile does not depend on the host.
var libtorrentRequiredSwitches = []string{"--ipv4-only", "--no-dht", "--no-lsd", "--no-upnp", "--no-natpmp", "--no-pex"}

// libtorrentStartArgs is the whole command line `start` gives the
// interpreter. ⚠ uTP and the encryption policy are left at the library's
// defaults, and so is the port when the caller says 0: 6881.
func libtorrentStartArgs(harness, torrent, workdir, alerts, peerPort string) ([]string, error) {
	if harness == "" || torrent == "" || workdir == "" || alerts == "" {
		return nil, fmt.Errorf("no harness, torrent, workdir or alerts file")
	}
	if peerPort != "0" && !rqbitPortOK(peerPort) {
		return nil, fmt.Errorf("peer port [%s] is not a port", peerPort)
	}
	args := []string{"-I", harness, "--torrent", torrent, "--data", filepath.Join(workdir, "downloads"),
		"--ready", filepath.Join(workdir, "ready.txt"), "--stop-file", filepath.Join(workdir, "stop-request"),
		"--alerts", alerts}
	// ⛔ SPELLED HERE, NOT TAKEN FROM libtorrentRequiredSwitches, so the refusal
	// is a second list the line is held to rather than the line's own source.
	args = append(args,
		"--ipv4-only", "--no-dht", "--no-lsd", "--no-upnp", "--no-natpmp", "--no-pex",
	)
	if peerPort != "0" {
		args = append(args, "--port", peerPort)
	}
	return args, nil
}

// libtorrentArgsRefusal answers what is wrong with a harness command line, or
// "". ⛔ `-I` first, every switch once, nothing it does not name.
func libtorrentArgsRefusal(args []string) string {
	if len(args) < 2 || args[0] != "-I" || filepath.Base(args[1]) != libtorrentHarnessFile {
		return "the line does not run the harness isolated, `-I <harness.py>`"
	}
	count := map[string]int{}
	for i := 2; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--torrent", "--data", "--ready", "--stop-file", "--alerts", "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return "[" + a + "] is not followed by its value"
			}
			count[a]++
			i++
			continue
		case "--ipv4-only", "--no-dht", "--no-lsd", "--no-upnp", "--no-natpmp", "--no-pex":
			count[a]++
			continue
		}
		return "[" + a + "] is not a switch this adapter passes"
	}
	for _, s := range libtorrentRequiredSwitches {
		if count[s] != 1 {
			return fmt.Sprintf("%s is passed %d times, not once", s, count[s])
		}
	}
	for _, s := range []string{"--torrent", "--data", "--ready", "--stop-file", "--alerts"} {
		if count[s] != 1 {
			return fmt.Sprintf("%s is passed %d times, not once", s, count[s])
		}
	}
	if count["--port"] > 1 {
		return "--port is passed more than once"
	}
	return ""
}

// libtorrentChildEnv is the environment the harness runs with. ⛔ Every
// `PYTHON*` is dropped, which `-I` ignores anyway, and every proxy variable,
// which the library does not read and nothing in the process should.
func libtorrentChildEnv(environ []string, extra ...string) []string {
	var out []string
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "PYTHON") || strings.HasSuffix(upper, "_PROXY") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, extra...)
}

// libtorrentReadyRefusal is what is wrong with the harness's report of the
// session it configured, `ready.txt`, or "".
//
// ⛔ THE SESSION'S OWN SETTINGS, READ BACK: every surface off, no proxy, no
// anonymous mode, `ut_pex` not loaded, the listen interface IPv4 on the peer
// port and every listener on it, and the library's identity defaults standing
// unset. ⚠ The last are self-reports and are only compared, never published.
func libtorrentReadyRefusal(body, peerPort string) string {
	m, err := libtorrentFields(body, "bit-ids/libtorrent-harness-ready/1")
	if err != nil {
		return err.Error()
	}
	for _, kv := range [][2]string{
		{"enable_dht", "false"}, {"enable_lsd", "false"}, {"enable_upnp", "false"}, {"enable_natpmp", "false"},
		{"extensions", "smart_ban,ut_metadata"}, {"proxy_type", "0"}, {"anonymous_mode", "false"},
		{"self_report_user_agent", libtorrentUserAgent}, {"library_default_user_agent", libtorrentUserAgent},
		{"self_report_peer_fingerprint", libtorrentFingerprint}, {"library_default_peer_fingerprint", libtorrentFingerprint},
		{"self_report_handshake_client_version", "-"},
	} {
		if m[kv[0]] != kv[1] {
			return fmt.Sprintf("the harness reports %s=[%s], not %s", kv[0], m[kv[0]], kv[1])
		}
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(m["infohash"]) {
		return "the harness reports no info hash"
	}
	port := peerPort
	if port == "0" {
		port = "6881"
	}
	if m["listen_interfaces"] != "0.0.0.0:"+port {
		return "the harness listens on [" + m["listen_interfaces"] + "], not 0.0.0.0:" + port + " alone"
	}
	if m["listen_port"] != port {
		return "the harness reports listen port [" + m["listen_port"] + "], not " + port
	}
	tcp := false
	for _, l := range strings.Split(m["listen"], ",") {
		f := strings.Fields(l)
		if len(f) != 2 {
			return "the harness reports the listener [" + l + "]"
		}
		// ⚠ libtorrent expands 0.0.0.0 into one listener per interface, so
		// each alert names an interface's own address: 127.0.0.1 in a
		// loopback-only namespace, measured on 2026-10-22.
		host, p, ok := strings.Cut(f[1], ":")
		if !ok || p != port || !regexp.MustCompile(`^[0-9]{1,3}(\.[0-9]{1,3}){3}$`).MatchString(host) {
			return "the harness listens on [" + l + "], which is not IPv4 on port " + port
		}
		tcp = tcp || f[0] == "tcp"
	}
	if !tcp {
		return "the harness reports no TCP listener"
	}
	return ""
}

// libtorrentDrive waits for the harness's report and holds it.
func libtorrentDrive(workdir, peerPort string, gone func(time.Duration) bool) (string, error) {
	ready := filepath.Join(workdir, "ready.txt")
	for waited := 0; ; waited++ {
		if b, err := os.ReadFile(ready); err == nil && len(b) > 0 && strings.HasSuffix(string(b), "\n") {
			if msg := libtorrentReadyRefusal(string(b), peerPort); msg != "" {
				return "", fmt.Errorf("%s", msg)
			}
			return keyField(b, "listen"), nil
		}
		if gone(time.Second) {
			return "", fmt.Errorf("the harness exited before it reported its session; %s says why", filepath.Join(workdir, "client.out"))
		}
		if waited >= 30 {
			return "", fmt.Errorf("the harness did not report its session within 30 seconds")
		}
	}
}

// libtorrentGraceSeconds is how long `stop` waits for the library's own
// shutdown after the stop file appears, before it kills.
const libtorrentGraceSeconds = 10

// -- the selftest -------------------------------------------------------------

func libtorrentSelftest(args []string) int {
	if len(args) != 1 || args[0] != "--selftest" {
		fmt.Fprintln(os.Stderr, "libtorrent: usage: bit-check libtorrent --selftest")
		return 2
	}
	cases, failed := 0, 0
	check := func(name string, ok bool) {
		cases++
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "libtorrent: FAILED %s\n", name)
		}
	}

	d := libtorrentDescribe()
	for _, want := range []string{"target_kind=library", "release_repo=arvidn/libtorrent", "release_tag_prefix=v2.0.",
		"release_min_components=1", "release_max_components=1", "release_floor_major=-", "compiles=source",
		"routes=package,source"} {
		check("describe carries "+want, strings.Contains(d, want+"\n"))
	}
	check("describe declares no release asset", !strings.Contains(d, "release_asset"))
	for _, c := range []struct {
		route, want string
		ok          bool
	}{
		{"package", "python-wheel", true},
		{"source", "python-source", true},
		{"release", "", false},
	} {
		f, ok := libtorrentPackageFormat(c.route)
		check("route "+c.route+" packages as ["+c.want+"]", ok == c.ok && f == c.want)
	}

	// The answer, in the shape the PyPI wheel gave on 2026-10-22.
	answer := "bit-ids/libtorrent-harness-version/1\nlibrary=libtorrent\nlibrary_version=2.0.15.0\npython=3.13.16\n" +
		"python_implementation=cpython\nmodule=libtorrent/__init__.cpython-313-x86_64-linux-gnu.so\n" +
		"module_sha256=" + strings.Repeat("7e", 32) + "\nabi_version_1=true\n" +
		"dht=true\nlogging=true\ni2p=true\nencryption=true\nopenssl=static\nopenssl_version_text=OpenSSL 3.5.0 8 Apr 2025\n" +
		"openssl_mapped=-\ndefault_settings=258\ndefault_settings_sha256=" + libtorrentDefaultsPinnedSha256 + "\n" +
		"default.user_agent=libtorrent/2.0.15.0\ndefault.peer_fingerprint=-LT20F0-\ndefault.handshake_client_version=-\n" +
		"default.listen_interfaces=0.0.0.0:6881,[::]:6881\ndefault.enable_dht=true\n"
	m, err := libtorrentFields(answer, "bit-ids/libtorrent-harness-version/1")
	check("the answer is read", err == nil && m["library_version"] == "2.0.15.0")
	check("the answer is held", err == nil && libtorrentAnswerRefusal(m) == "")
	src := strings.Replace(answer, "OpenSSL 3.5.0 8 Apr 2025", "-", 1)
	ms, _ := libtorrentFields(src, "bit-ids/libtorrent-harness-version/1")
	check("the source build's answer, whose OpenSSL keeps no text, is held", libtorrentAnswerRefusal(ms) == "")
	_, err = libtorrentFields(strings.ReplaceAll(answer, "\n", "\r\n"), "bit-ids/libtorrent-harness-version/1")
	check("a CRLF answer is read", err == nil)
	for name, bad := range map[string]string{
		"another version":       strings.Replace(answer, "library_version=2.0.15.0", "library_version=2.1.2.0", 1),
		"no deprecated API":     strings.Replace(answer, "abi_version_1=true", "abi_version_1=false", 1),
		"no DHT compiled":       strings.Replace(answer, "\ndht=true", "\ndht=false", 1),
		"no logging":            strings.Replace(answer, "logging=true", "logging=false", 1),
		"no I2P":                strings.Replace(answer, "i2p=true", "i2p=false", 1),
		"no encryption":         strings.Replace(answer, "encryption=true", "encryption=false", 1),
		"no OpenSSL":            strings.Replace(answer, "openssl=static", "openssl=none", 1),
		"a mapped OpenSSL":      strings.Replace(strings.Replace(answer, "openssl=static", "openssl=mapped", 1), "openssl_mapped=-", "openssl_mapped=libssl.so.3", 1),
		"another defaults set":  strings.Replace(answer, libtorrentDefaultsPinnedSha256, strings.Repeat("0", 64), 1),
		"another user agent":    strings.Replace(answer, "default.user_agent=libtorrent/2.0.15.0", "default.user_agent=bit-ids", 1),
		"another fingerprint":   strings.Replace(answer, "-LT20F0-", "-BI0001-", 1),
		"a handshake version":   strings.Replace(answer, "default.handshake_client_version=-", "default.handshake_client_version=x", 1),
		"PyPy":                  strings.Replace(answer, "=cpython", "=pypy", 1),
		"no module digest":      strings.Replace(answer, strings.Repeat("7e", 32), "", 1),
		"a key twice":           answer + "dht=true\n",
		"no schema line":        strings.SplitN(answer, "\n", 2)[1],
		"a line with no equals": answer + "garbage\n",
	} {
		mm, err := libtorrentFields(bad, "bit-ids/libtorrent-harness-version/1")
		check("an answer with "+name+" is refused", err != nil || libtorrentAnswerRefusal(mm) != "")
	}
	for lib, want := range map[string]string{"2.0.15.0": "2.0.15", "2.0.15.1": "", "2.0.15": "", "2.0.x.0": "", "": ""} {
		got, err := libtorrentRecordVersion(lib)
		check("the record spells ["+lib+"] as ["+want+"]", got == want && (err == nil) == (want != ""))
	}
	check("the pin is held", libtorrentPinRefusal("PyPI", "2.0.15") == "")
	check("a newer 2.0 release asks for a re-pin", strings.Contains(libtorrentPinRefusal("PyPI", "2.0.16"), "re-pin"))
	check("another series is refused", libtorrentPinRefusal("the tag", "2.1.2") != "")

	// The b2 line, as the tag's setup.py ran it on 2026-10-22.
	b2 := `b2 cxxstd=14 deprecated-functions=on boost-link=static libtorrent-link=static crypto=openssl variant=release address-model=64 python=3.13 libtorrent-python=on library-path="/usr/lib/x86_64-linux-gnu" python-install-path=/w/src/bindings/python/build/lib install_module --project-config=/tmp/tmp25ujdq8t`
	line, err := libtorrentB2Line("  running build_ext\n  " + b2 + "\n  ...\n")
	check("the b2 line is found", err == nil && line == b2)
	check("the b2 line is held", libtorrentB2Refusal(b2) == "")
	_, err = libtorrentB2Line("nothing here\n")
	check("a log with no b2 line is refused", err != nil)
	_, err = libtorrentB2Line(b2 + "\n" + strings.Replace(b2, "release", "debug", 1) + "\n")
	check("two different b2 lines are refused", err != nil)
	for name, bad := range map[string]string{
		"crypto built in":        strings.Replace(b2, "crypto=openssl", "crypto=built-in", 1),
		"a debug build":          strings.Replace(b2, "variant=release", "variant=debug", 1),
		"no deprecated API":      strings.Replace(b2, "deprecated-functions=on", "deprecated-functions=off", 1),
		"Boost linked shared":    strings.Replace(b2, "boost-link=static", "boost-link=shared", 1),
		"a define":               b2 + " define=TORRENT_DISABLE_DHT",
		"a second crypto":        b2 + " crypto=openssl",
		"no crypto":              strings.Replace(b2, " crypto=openssl", "", 1),
		"another C++ standard":   strings.Replace(b2, "cxxstd=14", "cxxstd=17", 1),
		"a toolset":              b2 + " toolset=clang",
		"logging off":            b2 + " logging=off",
		"no install_module":      strings.Replace(b2, " install_module", "", 1),
		"a 32-bit address model": strings.Replace(b2, "address-model=64", "address-model=32", 1),
	} {
		check("a b2 line with "+name+" is refused", libtorrentB2Refusal(bad) != "")
	}

	env := libtorrentBuildEnv([]string{"PATH=/usr/bin", "BOOST_ROOT=/evil", "CXXFLAGS=-O0", "CC=clang", "HOME=/h",
		"PYTHONPATH=/x", "PIP_INDEX_URL=http://evil.invalid", "b2=/x"}, "/w/boost")
	ej := "\n" + strings.Join(env, "\n") + "\n"
	check("the build finds the route's Boost", strings.Contains(ej, "\nBOOST_ROOT=/w/boost\n") &&
		strings.Contains(ej, "\nBOOST_BUILD_PATH="+filepath.Join("/w/boost", "tools", "build")+"\n") &&
		strings.Contains(ej, "\nPATH=/w/boost"+string(os.PathListSeparator)+"/usr/bin\n"))
	check("the host's Boost, compiler, flags, pip and Python variables are dropped", !strings.Contains(ej, "/evil") &&
		!strings.Contains(ej, "CXXFLAGS") && !strings.Contains(ej, "CC=") && !strings.Contains(ej, "PYTHONPATH") &&
		!strings.Contains(ej, "evil.invalid") && !strings.Contains(ej, "b2="))
	check("the rest is kept", strings.Contains(ej, "\nHOME=/h\n") && strings.Count(ej, "\nPATH=") == 1)

	// The command line.
	good, err := libtorrentStartArgs("/h/harness.py", "/t.torrent", "/w", "/w/alerts-1.log", "51413")
	check("the command line is built", err == nil)
	check("the command line is held", libtorrentArgsRefusal(good) == "")
	check("the peer port is passed", strings.Contains(strings.Join(good, " "), "--port 51413"))
	zero, _ := libtorrentStartArgs("/h/harness.py", "/t.torrent", "/w", "/w/a.log", "0")
	check("peer port 0 leaves the library's", !strings.Contains(strings.Join(zero, " "), "--port") && libtorrentArgsRefusal(zero) == "")
	for _, bad := range [][5]string{
		{"", "/t", "/w", "/a", "1"}, {"/h", "", "/w", "/a", "1"}, {"/h", "/t", "", "/a", "1"},
		{"/h", "/t", "/w", "", "1"}, {"/h", "/t", "/w", "/a", "x"}, {"/h", "/t", "/w", "/a", "70000"}} {
		_, err := libtorrentStartArgs(bad[0], bad[1], bad[2], bad[3], bad[4])
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
	for _, s := range libtorrentRequiredSwitches {
		check("the planted line without "+s+" is refused", strings.Contains(libtorrentArgsRefusal(without(s)), s))
		check("the planted second "+s+" is refused", libtorrentArgsRefusal(append(append([]string(nil), good...), s)) != "")
	}
	for _, extra := range [][]string{
		{"--dht"}, {"-x"}, {"--port", "1"}, {"--ready"}, {"--proxy", "x"},
	} {
		check(fmt.Sprintf("the planted %q is refused", extra), libtorrentArgsRefusal(append(append([]string(nil), good...), extra...)) != "")
	}
	check("a line that is not isolated is refused", libtorrentArgsRefusal(good[1:]) != "")
	check("a line running another script is refused", libtorrentArgsRefusal(append([]string{"-I", "/h/other.py"}, good[2:]...)) != "")

	ce := libtorrentChildEnv([]string{"PATH=/bin", "PYTHONPATH=/x", "PYTHONHOME=/y", "HTTPS_PROXY=http://p", "no_proxy=x",
		"HOME=/old"}, "HOME=/new")
	cj := "\n" + strings.Join(ce, "\n") + "\n"
	check("PYTHON* and proxies are dropped", !strings.Contains(cj, "PYTHON") && !strings.Contains(strings.ToUpper(cj), "_PROXY="))
	check("the rest is kept and the extra comes last", strings.Contains(cj, "\nPATH=/bin\n") && ce[len(ce)-1] == "HOME=/new")

	// The report, in the shape the harness wrote on 2026-10-22.
	ready := "bit-ids/libtorrent-harness-ready/1\npid=7\ninfohash=" + strings.Repeat("ab", 20) + "\n" +
		"listen_interfaces=0.0.0.0:51413\nlisten_port=51413\nlisten=tcp 0.0.0.0:51413,utp 0.0.0.0:51413\n" +
		"enable_dht=false\nenable_lsd=false\nenable_upnp=false\nenable_natpmp=false\nextensions=smart_ban,ut_metadata\n" +
		"proxy_type=0\nanonymous_mode=false\nenable_outgoing_utp=true\nenable_incoming_utp=true\nalert_mask=4325\n" +
		"self_report_user_agent=libtorrent/2.0.15.0\nself_report_peer_fingerprint=-LT20F0-\n" +
		"self_report_handshake_client_version=-\nlibrary_default_user_agent=libtorrent/2.0.15.0\n" +
		"library_default_peer_fingerprint=-LT20F0-\n"
	check("the report is held", libtorrentReadyRefusal(ready, "51413") == "")
	for name, bad := range map[string]string{
		"the DHT on":            strings.Replace(ready, "enable_dht=false", "enable_dht=true", 1),
		"LSD on":                strings.Replace(ready, "enable_lsd=false", "enable_lsd=true", 1),
		"UPnP on":               strings.Replace(ready, "enable_upnp=false", "enable_upnp=true", 1),
		"NAT-PMP on":            strings.Replace(ready, "enable_natpmp=false", "enable_natpmp=true", 1),
		"ut_pex loaded":         strings.Replace(ready, "smart_ban,ut_metadata", "smart_ban,ut_metadata,ut_pex", 1),
		"a proxy":               strings.Replace(ready, "proxy_type=0", "proxy_type=2", 1),
		"anonymous mode":        strings.Replace(ready, "anonymous_mode=false", "anonymous_mode=true", 1),
		"another user agent":    strings.Replace(ready, "self_report_user_agent=libtorrent/2.0.15.0", "self_report_user_agent=x", 1),
		"another fingerprint":   strings.Replace(ready, "self_report_peer_fingerprint=-LT20F0-", "self_report_peer_fingerprint=-BI0001-", 1),
		"a handshake version":   strings.Replace(ready, "self_report_handshake_client_version=-", "self_report_handshake_client_version=x", 1),
		"an IPv6 interface":     strings.Replace(ready, "listen_interfaces=0.0.0.0:51413", "listen_interfaces=0.0.0.0:51413,[::]:51413", 1),
		"an IPv6 listener":      strings.Replace(ready, "utp 0.0.0.0:51413", "utp [::]:51413", 1),
		"an IPv6 address":       strings.Replace(ready, "utp 0.0.0.0:51413", "utp ::1:51413", 1),
		"another port":          strings.Replace(ready, "listen_port=51413", "listen_port=6881", 1),
		"no TCP listener":       strings.Replace(ready, "tcp 0.0.0.0:51413,", "", 1),
		"no listener":           strings.Replace(ready, "listen=tcp 0.0.0.0:51413,utp 0.0.0.0:51413", "listen=", 1),
		"no info hash":          strings.Replace(ready, strings.Repeat("ab", 20), "", 1),
		"no schema line":        strings.SplitN(ready, "\n", 2)[1],
		"a default set by hand": strings.Replace(ready, "library_default_peer_fingerprint=-LT20F0-", "library_default_peer_fingerprint=-XX0000-", 1),
	} {
		check("a report with "+name+" is refused", libtorrentReadyRefusal(bad, "51413") != "")
	}
	check("a report of one interface's address is held", libtorrentReadyRefusal(strings.Replace(ready,
		"listen=tcp 0.0.0.0:51413,utp 0.0.0.0:51413", "listen=tcp 127.0.0.1:51413,utp 127.0.0.1:51413", 1), "51413") == "")
	check("a report on the library's port is held for peer port 0",
		libtorrentReadyRefusal(strings.ReplaceAll(ready, "51413", "6881"), "0") == "")

	work, err := os.MkdirTemp("", "libtorrent-selftest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "libtorrent: cannot make a scratch directory")
		return 2
	}
	defer os.RemoveAll(work)
	never := func(time.Duration) bool { return false }
	_ = os.WriteFile(filepath.Join(work, "ready.txt"), []byte(ready), 0o644)
	listen, err := libtorrentDrive(work, "51413", never)
	check("the drive reads the report", err == nil && listen == "tcp 0.0.0.0:51413,utp 0.0.0.0:51413")
	_ = os.WriteFile(filepath.Join(work, "ready.txt"), []byte(strings.Replace(ready, "enable_dht=false", "enable_dht=true", 1)), 0o644)
	_, err = libtorrentDrive(work, "51413", never)
	check("the drive refuses a report with a surface on", err != nil && strings.Contains(err.Error(), "enable_dht"))
	_ = os.Remove(filepath.Join(work, "ready.txt"))
	_, err = libtorrentDrive(work, "51413", func(time.Duration) bool { return true })
	check("the drive reports a harness that exited", err != nil && strings.Contains(err.Error(), "exited"))

	// The harness the tree carries declares its exception and passes every
	// switch the adapter names.
	if root, err := captureRoot(); err == nil {
		if b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(libtorrentHarnessDir), libtorrentHarnessFile)); err == nil {
			h := string(b)
			check("this tree's harness declares its Python exception", strings.Contains(h, "bit-ids:python-exception=ENGINE-01"))
			for _, s := range libtorrentRequiredSwitches {
				check("this tree's harness takes "+s, strings.Contains(h, `"`+strings.TrimPrefix(s, "--")+`"`))
			}
		}
	}

	fmt.Printf(`{"schema":"libtorrent-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
