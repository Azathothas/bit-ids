//go:build !windows

// adapter_transmission - the CLIENT-06 capture adapter, `bit-check adapter transmission`.
//
// CI-11. It was scripts/capture/adapters/transmission.sh, and the reasons that
// file gave are kept here with the lines they explain. The contract is
// scripts/capture/adapters/README.md.
//
// ⚠ THE ONE TARGET WHOSE ENTRY IS ABOUT NOT USING SOURCE. CLIENT-06 exists
// because published formulas for Transmission's peer ID are good hypotheses and
// violate the live-only evidence rule, so everything this adapter contributes is
// a running process and nothing it contributes is a formula.
//
// ⭐ ITS CONTROL SURFACE IS A SECOND PROCESS RATHER THAN A FILE. transmission-cli
// was removed upstream, so the supported unattended path is the daemon plus
// transmission-remote, and that is what this drives: the daemon is started with
// its whole settings file written, and the torrent is added by the tool the
// product ships for the purpose.
//
// -- ⛔ WHAT THIS ASSUMES ABOUT THE PRODUCT, AND HAS NOT MEASURED --------------
//
// Each line is a claim a dispatch settles:
//
//   - `transmission-daemon --version` names the version in its second field,
//     with a build identifier beside it that is a separate fact;
//   - `transmission-remote` arrives with `transmission-cli` on this
//     distribution, and the daemon in `transmission-daemon`;
//   - the `dht-enabled`, `pex-enabled` and `lpd-enabled` settings are the three
//     adjacent surfaces, and `settings.json` under `--config-dir` is read at
//     start;
//   - `--foreground` keeps the daemon as the process this backgrounds;
//   - the distribution starts the daemon as a service on install, which is why
//     the package route stops and disables it.
//
// ⚠ The version fields are the claim most likely to be wrong, and it fails
// loudly: `capture-client` refuses a version that is not printable text and one
// that is empty, so a mis-parse is a refused capture rather than a record
// naming a build nobody ran.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter("transmission", shAdapter("transmission", transmissionAdapter)) }

func transmissionRPCPort() string {
	if p := os.Getenv("BIT_IDS_TRANSMISSION_RPC"); p != "" {
		return p
	}
	return "9091"
}

// ⛔ ONE LOOKUP PER EXECUTABLE, so every subcommand finds the same build.
//
// ⛔ THE ROOTLESS INSTALLER, AND THE ONE PREFIX IT DERIVES. The release and
// source routes compile and place both executables at `<prefix>/bin`, so the
// lookups ask that prefix first. `daemon()` had no prefix branch until
// 2026-10-14, when neither route installed into one. `CLIENT-06`. ⚠ The prefix
// is asked quietly here, as `prefix 2>/dev/null` was, and one that cannot be
// derived falls through to PATH.
func transmissionDaemon(k *adapterKit) (string, bool) {
	return k.lookup(os.Getenv("BIT_IDS_TRANSMISSION_DAEMON"), "transmission-daemon", true)
}

func transmissionRemote(k *adapterKit) (string, bool) {
	return k.lookup(os.Getenv("BIT_IDS_TRANSMISSION_REMOTE"), "transmission-remote", true)
}

// transmissionPackageFormat is what each route delivers, as `Build.package`
// spells it. ⚠ The release and source routes compile, so what each delivers is
// an executable this host built, which is `aria2-next`'s source route's
// spelling.
func transmissionPackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "deb", true
	case "release", "source":
		return "elf-binary", true
	}
	return "", false
}

func transmissionAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Print("target=transmission\n")
		fmt.Print("kind=stock\n")
		// ⭐ AND WHERE THE BUILD IS, WHEN THERE IS ONE; the key is OMITTED when
		// nothing is installed. `aria2` says why at length.
		if where, _ := transmissionDaemon(k); where != "" {
			fmt.Printf("binary=%s\n", where)
		}
		// ⛔ WHAT THE RELEASE ROUTE NEEDS TO KNOW. These were declared while the
		// route still refused at the build, which made the build a named gap; the
		// route builds since 2026-10-14. `CLIENT-06`.
		//
		// ⚠ THE TAGS CARRY NO PREFIX, measured by `ACQ-02`'s live dry run: 83
		// candidates, bare three-component tags, `4.1.3` selected over 10 superseded,
		// 21 prerelease and 51 predating. `-` is how `resolve-stable` spells "no
		// prefix", and it is passed through rather than left empty so a missing
		// declaration stays distinguishable from a declared absence.
		//
		// ⚠ ONE SOURCE ARCHIVE AND NO LINUX BINARY. Measured 2026-09-09 against the
		// live listing: release 4.1.3 carries eleven assets - Windows `.msi` files,
		// `-pdb.7z` symbol archives, a macOS `.dmg`, a `-dsym.zip` and exactly one
		// `transmission-4.1.3.tar.xz`. ⛔ Two of them are capitalised `Transmission-`,
		// which is why the pattern is matched case-sensitively.
		fmt.Print("release_repo=transmission/transmission\n")
		fmt.Print("release_tag_prefix=-\n")
		fmt.Print("release_min_components=3\n")
		fmt.Print("release_max_components=3\n")
		// ⭐ The three-component spelling began at 4.0.0, and refs carry 112 tags
		// before it, from 0.5 to 3.00, that it cannot order and no date releases.
		// Measured on 2026-10-15; `CLIENT-06`.
		fmt.Print("release_floor_major=4\n")
		fmt.Print("release_series=-\n")
		fmt.Print("release_asset=transmission-{version}.tar.xz\n")
		// ⛔ THE DIGEST DISPOSITION, DECLARED ONCE AND READ BY THE ROUTE, AND IT IS
		// ABOUT THE VENDOR'S OWN CHECKSUMS DOCUMENT; `aria2` says why at length.
		// ⚠ MEASURED through rule 8's route on 2026-09-17: 4.1.3 carries installers,
		// debug archives and one source tarball, no checksums file, and `digest` on
		// all eleven.
		fmt.Print("release_digest_unpublished=this release publishes installers and a source tarball and no checksums asset\n")
		// ⛔ WHICH ROUTES COMPILE, SO THE WORKFLOW BOUNDS THEM AS BUILDS. The release
		// asset is a source tarball, so both routes here run a build measured at 695
		// seconds, and a release route bounded as a download would be killed
		// half-way. `CLIENT-06`.
		fmt.Print("compiles=release,source\n")

	case "install":
		route, workdir, format := k.installArgs(args, transmissionPackageFormat)
		switch route {
		case "package":
			// ⚠ BOTH HALVES, because the control surface is a separate binary. A
			// route that installed the daemon alone would leave `start` unable to
			// add a torrent and the failure would read as the build declining one.
			k.aptPackageRoute(workdir, "the package route did not install transmission",
				"transmission-daemon", "transmission-cli")
			// ⚠ The distribution starts the daemon as a service on install. A
			// capture must own the only running copy, or it would measure whichever
			// one the torrent reached.
			if _, ok := commandV("systemctl"); ok {
				log := filepath.Join(workdir, "install.log")
				c := exec.Command("systemctl", "stop", "transmission-daemon")
				run(c, stdio{out: log, outAppend: true, err: "&1"})
				c = exec.Command("systemctl", "disable", "transmission-daemon")
				run(c, stdio{out: log, outAppend: true, err: "&1"})
			}
		case "release":
			transmissionRelease(k, workdir)
		case "source":
			transmissionSource(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		binary, _ := transmissionDaemon(k)
		if binary == "" {
			k.cannot("transmission-daemon is not installed")
		}
		// ⛔ THE BUILD SPEAKING. `transmission-daemon 4.0.5 (fac9a5f1c8)` is the
		// shape; the field taken is the version alone and the build hash beside it
		// is left where it is, because a version and a build are two facts.
		// ⚠ Merged on purpose: this product prints its version to stderr, which
		// shell.md section 3 names as the case where merging is the decision.
		c := exec.Command(binary, "--version")
		output, rc := capture(c, stdio{in: os.DevNull, err: "&1"})
		if rc != 0 {
			k.cannot(fmt.Sprintf("transmission-daemon --version exited %d", rc))
		}
		line := firstLine(output)
		version := ""
		if f := shFields(line); len(f) >= 2 {
			version = f[1]
		}
		if version == "" {
			k.cannot("transmission-daemon reported no parseable version: [" + line + "]")
		}
		k.versionReport(output, version)

	case "start":
		transmissionStart(k, args)

	case "stop":
		transmissionStop(k, args)
	}
}

// transmissionRelease is the release route.
//
// ⭐ THE VENDOR'S SOURCE TARBALL, VERIFIED AND THEN BUILT. Release 4.1.3
// publishes no Linux binary: `transmission-4.1.3.tar.xz`, Windows installers and
// a macOS image. ⚠ This route refused after its fetch until 2026-10-14, because
// nothing here could build it; a route that installs nothing must not report an
// install, and that still holds for every refusal below. `CLIENT-06`.
// ⛔ THE FETCH AND THE DIGEST ARE ONE CALL AND IT IS NOT THIS FILE'S.
// `bit-check install-rootless` bounds the transfer and settles the digest, and it
// needs no privilege.
func transmissionRelease(k *adapterKit, workdir string) {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	disposition := k.releaseDisposition()
	fetch := append([]string{"--fetch", "--url", url, "--into", filepath.Join(workdir, "transmission-release")},
		disposition...)
	fetch = append(fetch, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not fetch the vendor's source archive", fetch...)
	if !mkdirP(filepath.Join(workdir, "release-src")) {
		k.refuse("the release route could not create its source directory")
	}
	c := exec.Command("tar", "-xJf", filepath.Join(workdir, "transmission-release"),
		"-C", filepath.Join(workdir, "release-src"))
	if run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), outAppend: true, err: "&1"}) != 0 {
		k.refuse("the release route's archive did not unpack")
	}
	trees := shGlob(filepath.Join(workdir, "release-src") + "/*/CMakeLists.txt")
	if len(trees) != 1 || !isFile(trees[0]) {
		k.refuse("the release archive did not unpack into one source tree")
	}
	transmissionBuild(k, workdir, strings.TrimSuffix(trees[0], "/CMakeLists.txt"), "release")
}

// transmissionSource is the source route.
//
// ⭐ THE SECOND ROUTE: THE REPOSITORY'S OWN TAG, CLONED AND BUILT. It resolves
// through the repository's refs rather than the release listing, and arrives as
// a clone rather than an asset, which is the shape `aria2-next` measured.
// ⚠ Whoever controls the repository controls both.
func transmissionSource(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	k.need(func(string) string { return "the source route clones and git is not on this host" }, "git")
	repo := k.describeField("release_repo")
	if repo == "" {
		k.cannot("this adapter declares no release_repo to clone from")
	}
	// ⛔ THE SUBMODULES ARE PART OF THE SOURCE. 4.1.3 pins sixteen under
	// `third-party/`, which the release tarball carries inline; a clone without
	// them is a tree that does not build. Shallow, as the history is not the
	// measurement: 39 seconds and 137 megabytes on 2026-10-14. ⛔ Bounded, as
	// `aria2-next`'s clone is and for its reasons.
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag, "--recurse-submodules",
		"--shallow-submodules", "https://github.com/"+repo+".git", filepath.Join(workdir, "src"))
	if rc := run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	k.cloneCommit(workdir, tag)
	transmissionBuild(k, workdir, filepath.Join(workdir, "src"), "source")
}

// transmissionCurlHeaders answers the include directory libcurl's headers were
// unpacked to, and writes what it took, and which runtime the headers describe,
// to `build-deps.txt`.
//
// ⛔ LIBCURL'S HEADERS, WITHOUT ROOT AND WITHOUT CHANGING THE HOST. Transmission
// 4.x links libcurl, and a runner carries its runtime and not its development
// package. Measured on 2026-10-14: `apt-get` with its index and cache in the
// workdir runs as an unprivileged user, fetches `libcurl4-openssl-dev` through
// the distribution's signed index, and `dpkg-deb -x` unpacks it there. The
// headers then belong to the runtime the host already has, which is checked
// rather than assumed. `ACQ-06` is why nothing here may escalate.
//
// ⚠ The shell half called this inside `$(...)`, so whatever `dpkg-deb -x`
// printed became part of the answer. It still does.
func transmissionCurlHeaders(k *adapterKit, workdir string) string {
	deps := filepath.Join(workdir, "deps")
	if !mkdirP(deps+"/apt/lists/partial", deps+"/apt/cache/archives/partial", deps+"/deb", deps+"/root") {
		k.cannot("cannot create " + deps)
	}
	k.need(func(n string) string { return "the build needs libcurl's headers and " + n + " is not on this host" },
		"apt-get", "dpkg-deb")
	apt := aptPrivate(deps + "/apt")
	log := filepath.Join(workdir, "deps.log")
	c := exec.Command("timeout", append(append([]string{"300", "apt-get"}, apt...), "update")...)
	if run(c, stdio{in: os.DevNull, out: log, err: "&1"}) != 0 {
		k.refuse("a private package index could not be fetched for libcurl's headers")
	}
	c = exec.Command("timeout", append(append([]string{"120", "apt-get"}, apt...), "download", "libcurl4-openssl-dev")...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: deps + "/deb"}) != 0 {
		k.refuse("libcurl4-openssl-dev could not be downloaded")
	}
	debs := shGlob(deps + "/deb/libcurl4-openssl-dev_*.deb")
	if len(debs) != 1 || !isFile(debs[0]) {
		k.refuse("the download left no single libcurl4-openssl-dev package")
	}
	deb := debs[0]
	c = exec.Command("dpkg-deb", "-x", deb, deps+"/root")
	printed, rc := capture(c, stdio{})
	if rc != 0 {
		k.refuse("libcurl4-openssl-dev could not be unpacked")
	}
	c = exec.Command("find", deps+"/root/usr/include", "-path", "*/curl/curl.h", "-type", "f")
	found, _ := capture(c, stdio{})
	header := firstLine(found)
	if header == "" {
		k.refuse("libcurl4-openssl-dev carried no curl/curl.h")
	}
	// ⛔ THE HEADERS MUST DESCRIBE THE LIBRARY THAT WILL BE LOADED. The package's
	// `libcurl.so` names the file of the runtime it was built against, and that
	// file must already be on this host.
	c = exec.Command("find", deps+"/root/usr/lib", "-name", "libcurl.so", "-type", "l")
	found, _ = capture(c, stdio{})
	link := firstLine(found)
	if link == "" {
		k.refuse("libcurl4-openssl-dev carried no libcurl.so link")
	}
	dir := strings.TrimPrefix(link, deps+"/root")
	target, _ := os.Readlink(link)
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		dir = dir[:i]
	}
	runtimeLib := dir + "/" + target
	if !isFile(runtimeLib) {
		k.refuse("libcurl4-openssl-dev is for " + runtimeLib + ", which this host does not have")
	}
	field := func(name string) string {
		c := exec.Command("dpkg-deb", "-f", deb, name)
		v, _ := capture(c, stdio{})
		return v
	}
	digest, _ := fileSHA256(deb)
	record := "package=" + field("Package") + "\n" + "version=" + field("Version") + "\n" +
		"sha256=" + digest + "\n" + "runtime=" + runtimeLib + "\n"
	if writeText(filepath.Join(workdir, "build-deps.txt"), record) != nil {
		k.refuse("the build could not record its dependency")
	}
	return strings.TrimRight(printed+strings.TrimSuffix(header, "/curl/curl.h"), "\n")
}

// transmissionBuild is the one build both routes run.
//
// ⛔ ONE BUILD FOR BOTH ROUTES, so the pair differs in what it was built from and
// in nothing this file chose. The daemon and `transmission-remote` are built; the
// GTK and Qt clients, the CLI and the tests are not, because the capture drives
// the daemon. The configure line goes to `build-configure.txt` beside the
// install, as `aria2-next`'s source route does.
func transmissionBuild(k *adapterKit, workdir, src, route string) {
	include := transmissionCurlHeaders(k, workdir)
	deps, _ := os.ReadFile(filepath.Join(workdir, "build-deps.txt"))
	var runtimes []string
	for _, line := range strings.Split(string(deps), "\n") {
		f := strings.Split(line, "=")
		if f[0] == "runtime" && len(f) > 1 {
			runtimes = append(runtimes, f[1])
		}
	}
	runtimeLib := strings.Join(runtimes, "\n")
	k.need(func(n string) string { return "the " + route + " route compiles and " + n + " is not on this host" },
		"cmake", "ninja", "c++")
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	bin := filepath.Join(prefix, "bin")
	if !mkdirP(bin) {
		k.refuse("this user cannot create " + prefix + "/bin; a rootless install writes nowhere else")
	}
	flags := []string{"-G", "Ninja", "-DCMAKE_BUILD_TYPE=Release",
		"-DENABLE_GTK=OFF", "-DENABLE_QT=OFF", "-DENABLE_MAC=OFF", "-DENABLE_CLI=OFF",
		"-DENABLE_TESTS=OFF", "-DENABLE_DAEMON=ON", "-DENABLE_UTILS=ON",
		"-DCURL_INCLUDE_DIR=" + include, "-DCURL_LIBRARY=" + runtimeLib}
	var b strings.Builder
	fmt.Fprintf(&b, "cmake -S <%s> -B <build>", route)
	for _, a := range flags {
		fmt.Fprintf(&b, " '%s'", a)
	}
	fmt.Fprintf(&b, "\nhost CFLAGS=%s\n", hostFlag("CFLAGS"))
	fmt.Fprintf(&b, "host CXXFLAGS=%s\n", hostFlag("CXXFLAGS"))
	fmt.Fprintf(&b, "host LDFLAGS=%s\n", hostFlag("LDFLAGS"))
	if writeText(filepath.Join(workdir, "build-configure.txt"), b.String()) != nil {
		k.refuse("the " + route + " route could not record how it configured the build")
	}
	log := filepath.Join(workdir, "install.log")
	build := filepath.Join(workdir, "build")
	c := exec.Command("cmake", append([]string{"-S", src, "-B", build}, flags...)...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("the " + route + " route could not configure a build")
	}
	c = exec.Command("cmake", "--build", build)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("the " + route + " route could not build transmission")
	}
	// ⛔ BOTH ARE CHECKED BEFORE EITHER IS INSTALLED. A refusal that had already
	// placed the daemon would leave a build in the prefix for the next lane to
	// measure; `check-transmission` found that on its first run, 2026-10-14.
	exes := []string{"daemon/transmission-daemon", "utils/transmission-remote"}
	for _, exe := range exes {
		if !isExec(filepath.Join(build, exe)) {
			k.refuse("the " + route + " route built nothing at build/" + exe)
		}
	}
	for _, exe := range exes {
		name := exe[strings.Index(exe, "/")+1:]
		if !cpFile(filepath.Join(build, exe), filepath.Join(bin, "."+name+".new")) {
			k.refuse("the " + route + " route could not stage " + name + " in " + prefix + "/bin")
		}
		if !mvFile(filepath.Join(bin, "."+name+".new"), filepath.Join(bin, name)) {
			k.refuse("the " + route + " route could not install " + name + " into " + prefix + "/bin")
		}
	}
}

func transmissionStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	rpcPort := transmissionRPCPort()
	binary, _ := transmissionDaemon(k)
	if binary == "" {
		k.cannot("transmission-daemon is not installed")
	}
	remote, _ := transmissionRemote(k)
	if remote == "" {
		k.cannot("transmission-remote is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	config := filepath.Join(workdir, "config")
	if !mkdirP(config, filepath.Join(workdir, "downloads")) {
		k.cannot("cannot create the config under " + workdir)
	}
	// ⛔ THE WHOLE SETTINGS FILE IS WRITTEN. The three adjacent surfaces are
	// named individually, and port forwarding is off as well: a host with no
	// default route cannot reach a NAT gateway, and a build asking one is a
	// destination this run did not intend.
	var s strings.Builder
	s.WriteString("{\n")
	s.WriteString("  \"dht-enabled\": false,\n")
	s.WriteString("  \"pex-enabled\": false,\n")
	s.WriteString("  \"lpd-enabled\": false,\n")
	s.WriteString("  \"utp-enabled\": false,\n")
	s.WriteString("  \"port-forwarding-enabled\": false,\n")
	s.WriteString("  \"encryption\": 0,\n")
	fmt.Fprintf(&s, "  \"download-dir\": \"%s/downloads\",\n", workdir)
	s.WriteString("  \"incomplete-dir-enabled\": false,\n")
	s.WriteString("  \"rpc-enabled\": true,\n")
	s.WriteString("  \"rpc-bind-address\": \"127.0.0.1\",\n")
	fmt.Fprintf(&s, "  \"rpc-port\": %s,\n", rpcPort)
	s.WriteString("  \"rpc-authentication-required\": false,\n")
	s.WriteString("  \"rpc-whitelist\": \"127.0.0.1\",\n")
	s.WriteString("  \"rpc-host-whitelist-enabled\": false,\n")
	s.WriteString("  \"ratio-limit-enabled\": false,\n")
	if port != "0" {
		fmt.Fprintf(&s, "  \"peer-port\": %s,\n", port)
	}
	s.WriteString("  \"peer-port-random-on-start\": false\n")
	s.WriteString("}\n")
	if writeText(filepath.Join(config, "settings.json"), s.String()) != nil {
		k.cannot("the settings could not be written")
	}

	// ⚠ --foreground, so the process this backgrounds is the daemon itself. A
	// daemon that forked would leave `stop` holding the pid of a shell that had
	// already returned, and the build would outlive the capture.
	out := filepath.Join(workdir, "client.out")
	c := exec.Command(binary, "--foreground", "--config-dir", config, "--log-error")
	n, err := startBackground(c, stdio{out: out, err: "&1"})
	if err != nil {
		k.cannot("transmission-daemon could not be started: " + err.Error())
	}
	pid := fmt.Sprint(n)
	if writeText(filepath.Join(workdir, "pid"), pid+"\n") != nil {
		k.cannot("cannot record the pid of transmission-daemon")
	}

	// ⚠ The daemon is asked for its session before a torrent is handed over,
	// because transmission-remote against a socket nothing is listening on
	// reports a connection error, which would read as the build refusing the
	// torrent rather than as a daemon that had not finished starting.
	at := "127.0.0.1:" + rpcPort
	waited := 0
	for waited < 20 {
		c := exec.Command(remote, at, "--session-info")
		if run(c, stdio{out: os.DevNull, err: "&1"}) == 0 {
			break
		}
		if !alive(pid) {
			k.refuse("the daemon exited before it answered")
		}
		time.Sleep(time.Second)
		waited++
	}
	if waited >= 20 {
		k.refuse("the daemon never answered on " + at)
	}
	c = exec.Command(remote, at, "--add", torrent)
	if run(c, stdio{out: out, outAppend: true, err: "&1"}) != 0 {
		k.refuse("transmission-remote would not add the torrent")
	}
	c = exec.Command(remote, at, "--torrent", "all", "--start")
	if run(c, stdio{out: out, outAppend: true, err: "&1"}) != 0 {
		k.refuse("transmission-remote would not start the torrent")
	}
	fmt.Printf("started transmission-daemon pid %s on port %s\n", pid, port)
}

func transmissionStop(k *adapterKit, args []string) {
	if len(args) < 1 {
		k.cannot("stop takes <workdir>")
	}
	workdir := args[0]
	// ⛔ THE DAEMON WRITES AN RPC VERIFIER INTO ITS SETTINGS ON EXIT, AND RULE 12
	// KEEPS IT OFF THE BUNDLE. Measured on `capture-client` run 38600252161,
	// 2026-10-14: 4.1.3 saves `"rpc-password": "{<salted hash>"` even with
	// authentication off, and the bundle scan refused the capture for it. So once
	// the process is gone, that key is deleted, and if `jq` is absent the whole
	// file is, rather than a line edit that could leave invalid JSON.
	// ⚠ Every return below runs it, because each is a daemon no longer running.
	defer func() {
		settings := filepath.Join(workdir, "config", "settings.json")
		if !isFile(settings) {
			return
		}
		if _, ok := commandV("jq"); ok {
			c := exec.Command("jq", `del(."rpc-password")`, settings)
			if run(c, stdio{out: settings + ".new", err: os.DevNull}) == 0 {
				_ = os.Rename(settings+".new", settings)
				return
			}
		}
		_ = os.Remove(settings + ".new")
		_ = os.Remove(settings)
	}()
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	if !signalPID(pid, syscall.SIGTERM) || waitGone(pid, 8) {
		return
	}
	signalPID(pid, syscall.SIGKILL)
	// ⚠ A killed process cannot write, but one being killed might still be; a
	// moment for the kernel to take it down before the file is read.
	time.Sleep(time.Second)
}
