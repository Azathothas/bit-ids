//go:build !windows

// adapter_aria2-next - the CLIENT-14 capture adapter, `bit-check adapter aria2-next`.
//
// CI-11. It was scripts/capture/adapters/aria2-next.sh, and the reasons that
// file gave are kept here with the lines they explain. The contract is
// scripts/capture/adapters/README.md.
//
// ⛔ A DIFFERENT PRODUCT FROM `aria2`, NOT A VARIANT OF IT. `AnInsomniacy/aria2-next`
// keeps aria2's command-line vocabulary and does not keep its internals: this
// build links `libtorrent/2.1.1` and carries its own version string. Every value
// below was measured against the installed build on 2026-09-09 rather than
// copied from the adapter beside it, and three of those measurements contradict
// what copying would have produced.
//
// -- ⭐ WHAT WAS MEASURED, AND WHERE COPYING WOULD HAVE LIED ------------------
//
// Measured on 2026-09-09 against `aria2-next-2.7.4-linux-x86_64`, fetched from
// the project's own release and verified against the vendor's published
// `aria2-next-2.7.4-checksums.sha256` with `sha256sum -c` before it was run.
// Installing a product and asking its version is not a capture, so it needed no
// disposable host; nothing here was driven against a tracker.
//
//   - ⛔ THE VERSION IS THE FOURTH FIELD, NOT THE THIRD. aria2 prints
//     `aria2 version 1.37.0` and this build prints `Aria2 Next version 2.7.4`.
//     The third field, which is what `aria2` correctly takes for ITS product, is
//     the literal string `version` here. So the parse below is anchored on the
//     `version` token rather than on a column, because the product name is
//     prose whose word count is the vendor's to change.
//
//   - ⛔ `--enable-dht6` DOES NOT EXIST AND IS ACCEPTED ANYWAY. The build logs
//     `Legacy aria2 input from command line: enable-dht6; accepted and skipped;
//     libtorrent uses one DHT switch for both families`. ⚠ A flag that is
//     accepted and skipped is worse than one that is refused: the step succeeds
//     and the switch it named was never set. `--enable-dht=false` covers both
//     families, and `start` VERIFIES that over RPC rather than trusting it.
//     ⚠ 2.8.6 still REPORTED `enable-dht6` in its options and 3.0.2 retires
//     it, so `start` no longer requires a key the build stopped having; the
//     family sweep would hold it to false if a build reported it again.
//
//   - ⭐ RPC ANSWERS `version` BETTER THAN THE COMMAND LINE DOES.
//     `aria2.getVersion` returns `{"product":"aria2-next","version":"2.7.4",...}`,
//     a structured field, with no prose to parse. ⛔ It is still NOT what
//     `version` below uses, and that is the contract rather than an oversight:
//     `install-client` calls `version` BEFORE the route runs, on a host where
//     nothing is installed and no daemon exists. An adapter whose `version`
//     started one would be changing the host at the one moment the caller is
//     trying to observe it unchanged. RPC cannot answer a question asked of a
//     host with nothing running, so this is exactly the case CLIENT-14's
//     "wherever RPC answers the same question" clause excludes.
//
//   - ⛔ AND THE FACT THIS PROJECT EXISTS TO CATCH: the build's default
//     `bt-peer-id-prefix` is `-qB5230-`, read back over `aria2.getGlobalOption`.
//     A stock aria2-next presents itself on the wire as qBittorrent 5.2.3.0.
//     ⚠ NOTHING HERE OVERRIDES IT. The identity a stock build emits is the
//     measurement; an adapter that set a prefix would be publishing this
//     project's own string and calling it an observation.
//
// ⛔ THERE IS NO PACKAGE ROUTE AND THAT IS MEASURED, NOT ASSUMED. `apt-cache
// search aria2` on `ubuntu-24.04` lists `aria2`, `libaria2-0`, `libaria2-0-dev`
// and `persepolis`, and no fork. ⚠ The dangerous failure is not the missing
// package: it is that `apt-get install aria2` SUCCEEDS and installs a DIFFERENT
// PRODUCT, after which `version` would answer 1.37.0 and the record would carry
// a second route that acquired somebody else's build. The package route below
// refuses by name for that reason.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func init() { registerAdapter("aria2-next", shAdapter("aria2-next", aria2NextAdapter)) }

// aria2NextRPCPort is the JSON-RPC port.
//
// ⚠ THE RPC PORT IS A DEFAULT THIS FILE STATES, not one inherited from the
// product. aria2's own default is 6800; a capture host runs one build at a time,
// and an override exists so a host already using that port is a configuration
// rather than a collision nobody can see.
func aria2NextRPCPort() string {
	if p := os.Getenv("BIT_IDS_ARIA2_NEXT_RPC_PORT"); p != "" {
		return p
	}
	return "6800"
}

// aria2NextBinary is the installed executable, looked up once. ⛔ A path
// composed per subcommand is a second spelling, and the day a route installs
// somewhere else one subcommand finds the build and another reports it absent.
func aria2NextBinary(k *adapterKit) (string, bool) {
	return k.lookup(os.Getenv("BIT_IDS_ARIA2_NEXT"), "aria2-next", false)
}

// aria2NextRPC is one JSON-RPC call, and the three streams stay apart.
//
// ⭐ The response body goes to a file the caller names, curl's own diagnosis to
// another, and the exit code comes back from curl itself rather than from a
// pipeline.
//
// ⛔ `--noproxy` IS NOT OPTIONAL. This is a loopback call to a daemon on this
// host; a proxy in the environment would send a capture's own control channel
// off the machine, which on a contained host is the one thing that must not
// happen.
func aria2NextRPC(port, rpcErr, method, params, body string) int {
	c := exec.Command("curl", "-sS", "--noproxy", "*", "--max-time", "20",
		"-X", "POST", "http://127.0.0.1:"+port+"/jsonrpc",
		"-H", "Content-Type: application/json",
		"-d", `{"jsonrpc":"2.0","id":"bit-ids","method":"`+method+`","params":`+params+`}`,
		"-o", body)
	return run(c, stdio{err: rpcErr, errAppend: true})
}

// aria2NextRPCFailed is the refusal a JSON-RPC answer carries.
//
// ⚠ A JSON-RPC ERROR ARRIVES WITH HTTP 200 AND AN `error` MEMBER. A caller
// reading only curl's status would take a refusal for an answer: an adapter
// failure and a product refusal reach this file through one channel and must
// not leave it as one status.
func aria2NextRPCFailed(body string) bool {
	b, err := os.ReadFile(body)
	return err == nil && bytes.Contains(b, []byte(`"error"`))
}

// aria2NextPackageFormat is what each route delivers, as `Build.package`
// spells it.
//
// ⭐ Both routes install a bare executable and neither installs a package: the
// release asset IS the binary, measured at 1.2 seconds for a download and a
// chmod, and the source route compiles one. ⚠ The `package` route refuses by
// name, so it reaches nothing here.
func aria2NextPackageFormat(route string) (string, bool) {
	switch route {
	case "release", "source":
		return "elf-binary", true
	}
	return "", false
}

func aria2NextAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Print("target=aria2-next\n")
		fmt.Print("kind=stock\n")
		if where, _ := aria2NextBinary(k); where != "" {
			fmt.Printf("binary=%s\n", where)
		}
		// ⭐ THE RELEASE DECLARATIONS, MEASURED AGAINST THE LIVE LISTING on
		// 2026-09-09. Tags are `v2.7.4`, so the prefix is `v` and a version is three
		// components. A release publishes eight assets - seven builds over four
		// platforms, Android arm64 only and the other three in two architectures,
		// plus a checksums file - and exactly one of them is a Linux x86-64 build, so
		// the pattern needs no wildcard beyond `{version}`.
		//
		// ⚠ AND THE ASSET IS A BINARY, WHICH IS THE ONE PLACE THIS TARGET IS EASIER
		// THAN `aria2`. That project's release carries no Linux binary at all, so its
		// release route is a source build taking minutes; this one is a download and
		// a `chmod`, measured at 1.2 seconds.
		fmt.Print("release_repo=AnInsomniacy/aria2-next\n")
		fmt.Print("release_tag_prefix=v\n")
		fmt.Print("release_min_components=3\n")
		fmt.Print("release_max_components=3\n")
		fmt.Print("release_floor_major=-\n")
		fmt.Print("release_series=-\n")
		fmt.Print("release_asset=aria2-next-{version}-linux-x86_64\n")
		// ⭐ THE VENDOR PUBLISHES A DIGEST AND THIS IS WHERE IT SAYS SO. Its release
		// carries `aria2-next-{version}-checksums.sha256` beside the seven builds,
		// read out of the live listing on 2026-09-09 and still in the fixture, so
		// the release route is verified against the vendor rather than merely
		// recorded. `bit-check resolve-release` selects it with the same call that
		// selects the artifact, from the same response.
		fmt.Print("release_digest_asset=aria2-next-{version}-checksums.sha256\n")
		// ⛔ THE ROUTE THAT COMPILES, SO THE WORKFLOW BOUNDS IT AS A BUILD: 870
		// seconds for `v3.0.2` on the session host. The release route downloads.
		fmt.Print("compiles=source\n")

	case "install":
		route, workdir, format := k.installArgs(args, aria2NextPackageFormat)
		switch route {
		case "package":
			// ⛔ REFUSED BY NAME, AND THE REFUSAL IS THE MEASUREMENT. No package
			// index reachable from a capture host carries this fork; what they carry
			// is `aria2`, a different product at a different version. An
			// `apt-get install aria2` here would exit 0, install somebody else's
			// build, and hand `ACQ-03` a second route that agreed with nothing.
			// ⚠ Unreachable since `package_format` names no package route, and kept
			// so the reason survives the day one is declared.
			k.cannot("aria2-next is published only through its own releases; " +
				"no package index carries it, and installing aria2 would acquire a different product")
		case "release":
			aria2NextRelease(k, workdir)
		case "source":
			aria2NextSource(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		binary, _ := aria2NextBinary(k)
		if binary == "" {
			k.cannot("aria2-next is not installed")
		}
		// ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED.
		c := exec.Command(binary, "--version")
		output, rc := capture(c, stdio{in: os.DevNull, err: os.DevNull})
		if rc != 0 {
			k.cannot(fmt.Sprintf("aria2-next --version exited %d", rc))
		}
		line := firstLine(output)
		// ⛔ ANCHORED ON THE `version` TOKEN, NOT ON A COLUMN. Measured: this build's
		// first line is `Aria2 Next version 2.7.4`, so the third field is the word
		// `version` and the fourth is the number. A column index here would have
		// published the string `version` as this product's version, and it would have
		// validated: it is a non-empty field.
		version := ""
		f := shFields(line)
		for i := 0; i+1 < len(f); i++ {
			if f[i] == "version" {
				version = f[i+1]
				break
			}
		}
		if version == "" {
			k.cannot("aria2-next reported no parseable version: [" + line + "]")
		}
		// ⭐ This build's own answer names `libtorrent/2.1.1` among its libraries,
		// which is a fact about what it will put on the wire that no version number
		// carries.
		k.versionReport(output, version)

	case "start":
		aria2NextStart(k, args)

	case "stop":
		aria2NextStop(k, args)
	}
}

// aria2NextRelease is the release route.
//
// ⭐ THE VENDOR'S OWN BUILD, INSTALLED RATHER THAN MERELY FETCHED. Every release
// route in this tree once dropped its artifact in the workdir and returned 0, so
// `version` answered from whatever was already on PATH. This one puts the
// executable where the lookup looks and then asks the file whether it is there.
func aria2NextRelease(k *adapterKit, workdir string) {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	// ⛔ THE FETCH, THE DIGEST AND THE PLACING ARE ONE CALL, AND IT IS NOT THIS
	// FILE'S. `bit-check install-rootless` bounds the transfer, settles the digest
	// before anything is made executable, writes into a prefix this user already
	// owns, and prints the path it placed. ⚠ This route used to `curl`, `chmod +x`
	// and `cp` into `/usr/local/bin` itself, which is four adapters each carrying
	// its own copy of a rule and each needing `sudo` to finish. `ACQ-06` carries
	// what that privilege cost.
	disposition := k.releaseDisposition()
	install := append([]string{"--install", "--url", url, "--into", filepath.Join(workdir, "aria2-next"),
		"--as", "aria2-next"}, disposition...)
	install = append(install, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not install the vendor's build", install...)
}

// aria2NextSource is the source route.
//
// ⭐ THE SECOND ROUTE, AND IT EXISTS BECAUSE THIS TARGET HAS NO PACKAGE.
// `E-ACQ-01` refuses a record built from one route, and absolute 4 wants two
// routes resolving the SAME stable version. Measured on 2026-09-09: this build
// from tag `v2.7.5` answers `Aria2 Next version 2.7.5`, the same version the
// release asset reports, from a 256-megabyte `RelWithDebInfo` executable where
// the published one is 14 megabytes stripped. ⭐ One version, two provably
// different builds, which is the case `ACQ-03` exists to classify.
//
// ⛔ THESE TWO ROUTES DIFFER IN DELIVERY AND NOT IN ORIGIN: whoever controls the
// repository controls both. That is weaker than a distribution index against a
// vendor release, and the record says so rather than letting two green checks
// imply otherwise.
//
// ⚠ MEASURED COST: about 350 seconds for configure and build together on a
// four-core host for 2.7.5, and 870 for `v3.0.2` on the session host on
// 2026-10-14, whose superbuild compiles ten dependencies first, among them
// OpenSSL, curl, libtorrent, FFmpeg and GPAC. The caller's bound is
// `capture-client.yml`'s source-lane `BIT_IDS_INSTALL_TIMEOUT`, stated there
// alone. It builds the project's test binaries too, because `--preset default`
// is what the project's own README and CI run and a narrower target would be
// this repository inventing a build the vendor does not document.
func aria2NextSource(k *adapterKit, workdir string) {
	// ⛔ ITS OWN TAG, FROM ITS OWN RESOLUTION. This read `BIT_IDS_RELEASE_TAG`
	// until 2026-09-09, which is the tag the RELEASE lane's listing chose - so both
	// lanes resolved through one index and `E-ACQ-07` called them one route.
	// `bit-check resolve-source` reads the repository's refs instead, and the two
	// versions are then COMPARED after installation rather than made equal
	// beforehand, which is what absolute 4 asks for.
	// ⚠ `BIT_IDS_RELEASE_TAG` is still accepted as a fallback so a caller that has
	// only the older variable is refused by name rather than silently building
	// whatever `main` points at.
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		tag = os.Getenv("BIT_IDS_RELEASE_TAG")
	}
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	k.need(func(n string) string { return "the source route builds from source and " + n + " is not on this host" },
		"git", "cmake", "ninja", "c++")
	// ⛔ THE SAME PREFIX THE RELEASE ROUTE USES, ASKED THE SAME WAY. Two routes
	// installing to two places would make the lookup answer about whichever ran
	// last rather than about the route being measured.
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	bin := filepath.Join(prefix, "bin")
	if !mkdirP(bin) {
		k.refuse("this user cannot create " + prefix + "/bin; a rootless install writes nowhere else")
	}
	// ⚠ THE REPOSITORY IS THE ONE `describe` ALREADY NAMES, so the two routes
	// cannot drift onto different upstreams. ⛔ ASKED OF THIS ADAPTER'S OWN
	// `describe` RATHER THAN SPELLED AGAIN: a literal here and a literal in
	// `describe` is one fact in two places, and the day one moved the release
	// route and the source route would have acquired from two different
	// upstreams and every check here would have passed.
	repo := k.describeField("release_repo")
	if repo == "" {
		k.cannot("this adapter declares no release_repo to clone from")
	}
	log := filepath.Join(workdir, "install.log")
	// ⛔ BOUNDED: a clone that stalls has no limit of its own and waits forever.
	// ⚠ `git` has no `--max-time`, so the bound is `timeout` around it - which is
	// the spelling `shell.md` section 9 gives for a tool that does not bound
	// itself. ⛔ 600 rather than 300: this clone is the input to a build the
	// workflow gives a source lane many minutes for, so a bound sized like the
	// release fetch's would refuse a source lane that was working. ⛔ A shallow
	// clone of one tag: the history is not the measurement.
	// ⚠ Exit 124 is `timeout`'s verdict and is reported as a refusal naming the
	// bound, never as a clone that failed for its own reasons.
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/"+repo+".git", filepath.Join(workdir, "src"))
	if rc := run(c, stdio{in: os.DevNull, out: log, err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	// ⛔ THE COMMIT IS WHAT `E-ACQ-06` ASKS FOR AND A TAG IS NOT IT. A tag is a
	// name somebody can move; `SourceIdentity::SourceCommit` takes a full object
	// name and refuses an abbreviation. ⚠ It is `rev-parse HEAD` rather than the
	// tag's own object, because `v2.7.5` is an annotated tag - the clone said `is
	// not a commit!` and checked out what it points at, which is what was built.
	k.cloneCommit(workdir, tag)
	// ⛔ ONE FLAG THE VENDOR DOES NOT PASS, AND IT IS A HEADER ITS SOURCE LEFT
	// OUT. Measured on `capture-client` run 35, 2026-10-14: `v3.0.2` stops in
	// `src/disk/File.cc`, which calls `std::exchange` and never includes
	// `<utility>`. The vendor builds on `ubuntu-22.04` with GCC 11.4.0; this route
	// builds where the capture runs, on GCC 13, which refuses the file. ⚠ So it
	// passes `-include utility`: the header the compiler's own diagnostic names,
	// holding a declaration the source already calls, with no file the vendor
	// wrote changed. The superbuild forwards `CMAKE_CXX_FLAGS` to the inner
	// project and its seven CMake-built dependencies; OpenSSL, FFmpeg and GPAC
	// configure themselves and do not receive it.
	// ⛔ SET, NOT APPENDED TO, so the configure never carries a host's own
	// `CXXFLAGS` into those caches. ⚠ `ACQ-03` says a source route's configure
	// options belong to its identity and that nothing records them; the line goes
	// into `source-configure.txt` beside the build, which the install artifact
	// uploads, and the record still has no field for it. The host's own C,
	// preprocessor and linker flags are written beside it, because the superbuild
	// passes those on to its dependencies.
	const cxxflags = "-include utility"
	configure := fmt.Sprintf("CXXFLAGS='%s' cmake --preset default\n", cxxflags) +
		"host CFLAGS=" + hostFlag("CFLAGS") + "\n" +
		"host CPPFLAGS=" + hostFlag("CPPFLAGS") + "\n" +
		"host LDFLAGS=" + hostFlag("LDFLAGS") + "\n"
	if writeText(filepath.Join(workdir, "source-configure.txt"), configure) != nil {
		k.refuse("the source route could not record how it configured the build")
	}
	src := filepath.Join(workdir, "src")
	c = exec.Command("cmake", "--preset", "default")
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: src,
		env: []string{"CXXFLAGS=" + cxxflags}}) != 0 {
		k.refuse("the source route could not configure a build")
	}
	c = exec.Command("cmake", "--build", "--preset", "default")
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: src}) != 0 {
		k.refuse("the source route could not build aria2-next")
	}
	// ⚠ THE BUILT PATH IS THE PRESET'S, found rather than composed twice: the
	// `default` preset writes into `build/default` and that is where the
	// project's own CI reads its binary from.
	built := filepath.Join(src, "build", "default", "aria2-next")
	if !isExec(built) {
		k.refuse("the source route built nothing at build/default/aria2-next")
	}
	if !cpFile(built, filepath.Join(bin, ".aria2-next.new")) {
		k.refuse("the source route could not stage its build in " + prefix + "/bin")
	}
	if !mvFile(filepath.Join(bin, ".aria2-next.new"), filepath.Join(bin, "aria2-next")) {
		k.refuse("the source route could not install into " + prefix + "/bin")
	}
	if !isExec(filepath.Join(bin, "aria2-next")) {
		k.refuse("the source route reported an install and left no aria2-next in " + prefix + "/bin")
	}
}

// aria2NextResultRe reads the download's gid out of `aria2.addTorrent`'s answer.
// ⚠ The option families, the options judged harmless when `true` and `lines`
// are in a2nwindows.go, which both halves build, because the read-back they
// serve is one function for both: aria2NextOptionsRefusal.
var aria2NextResultRe = regexp.MustCompile(`^.*"result":"([^"]*)".*$`)

func aria2NextStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	rpcPort := aria2NextRPCPort()
	binary, _ := aria2NextBinary(k)
	if binary == "" {
		k.cannot("aria2-next is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	if !mkdirP(filepath.Join(workdir, "downloads")) {
		k.cannot("cannot create " + workdir + "/downloads")
	}
	k.need(func(string) string { return "curl is not on this host" }, "curl")
	k.need(func(string) string { return "base64 is not on this host" }, "base64")
	rpcErr := filepath.Join(workdir, "rpc.err")
	rpc := func(method, params, body string) bool {
		return aria2NextRPC(rpcPort, rpcErr, method, params, filepath.Join(workdir, body)) == 0
	}
	failed := func(body string) bool { return aria2NextRPCFailed(filepath.Join(workdir, body)) }

	// ⛔ A PER-RUN TOKEN, GENERATED HERE AND WRITTEN TO RUN STATE. A literal in
	// this file would be a shared secret in the tree, which rule 12 refuses; a
	// daemon with no token would accept an RPC call from anything that reached
	// the port.
	//
	// ⛔ AND `stop` DELETES IT, BECAUSE THE WORKDIR IS THE EVIDENCE BUNDLE.
	// Measured on capture-client run 11: this file shipped inside the uploaded
	// artifact as `client/rpc-token`, which is rule 12's "a secret never enters
	// ... artifacts" in the one place no reading had looked. ⚠ A short-lived
	// token for a loopback port on a host that is about to be destroyed is a weak
	// secret, and rule 12 does not grade them.
	//
	// ⚠ 0600 SO IT IS NEVER WORLD-READABLE EVEN WHILE IT EXISTS, which the shell
	// half spelled `umask 077` around the write.
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		k.cannot("cannot generate an RPC token")
	}
	token := hex.EncodeToString(raw)
	if os.WriteFile(filepath.Join(workdir, "rpc-token"), []byte(token+"\n"), 0o600) != nil {
		k.cannot("cannot record the RPC token")
	}
	if writeText(filepath.Join(workdir, "rpc-port"), rpcPort+"\n") != nil {
		k.cannot("cannot record the RPC port")
	}

	// ⛔ FOUR DISCOVERY SURFACES OFF, NOT THE THREE THE CONTRACT NAMES.
	// adapters/README.md says an adapter disables DHT, peer exchange and local
	// peer discovery; this product has a FOURTH. `bt-port-mapping` is "Enable UPnP
	// and NAT-PMP port mapping" and it defaults to TRUE, which was not read off
	// the help text but found by driving the build on 2026-09-09 and reading its
	// own sockets out of `/proc`: with the three documented switches off, the
	// process was still bound to **UDP 1900**, the SSDP multicast port.
	// ⚠ A contained host has no default route, so a tracker or a DHT bootstrap
	// fails - and a multicast to 239.255.255.250 is link-local and does not.
	//
	// ⛔ AND A FIFTH, ON THE SECOND PROTOCOL THIS BUILD SPEAKS. 3.0.0's release
	// notes add `--ed2k-port-mapping`, "forwards the ED2K ports through UPnP IGD",
	// and the options dump `capture-client` run 35 uploaded on 2026-10-14 has it
	// `true`. ⚠ A fixed list of switches is a guard that sees the spellings it
	// was written with, so the read-back below also sweeps every switch of those
	// families the build reports, named here or not.
	//
	// ⛔ AND `--enable-dht6` IS DELIBERATELY ABSENT. Measured: this build does not
	// have that option, accepts it from the command line anyway and skips it with
	// a warning. Passing it would put a flag in the record that set nothing.
	//
	// ⛔ AND `--bt-require-crypto` IS GONE, BECAUSE IT SET NOTHING. 3.0.2 maps it
	// onto `bt-encryption`, and only a value other than `false` sets it. What
	// applies is the stock `bt-encryption`, `preferred`; that is the product's
	// default and the capture's to observe, so nothing here sets it.
	//
	// ⛔ THE PEER PORT IS PASSED HERE, ON THE COMMAND LINE, and zero still means
	// "leave it": aria2 refuses a listen port of zero.
	//
	// ⛔ RPC IS BOUND TO LOOPBACK. `--rpc-listen-all=false` is the difference
	// between a control channel and a service on the capture host's network.
	flags := []string{
		"--enable-rpc=true",
		"--rpc-listen-port=" + rpcPort,
		"--rpc-listen-all=false",
		"--rpc-secret=" + token,
		"--enable-dht=false",
		"--bt-enable-lpd=false",
		"--enable-peer-exchange=false",
		"--bt-port-mapping=false",
		"--ed2k-port-mapping=false",
		"--dir=" + workdir + "/downloads",
		"--seed-time=0",
		"--allow-overwrite=true",
		"--console-log-level=info",
		"--summary-interval=0",
	}
	if port != "0" {
		flags = append(flags, "--listen-port="+port)
	}
	c := exec.Command(binary, flags...)
	n, err := startBackground(c, stdio{out: filepath.Join(workdir, "client.out"), err: "&1"})
	if err != nil {
		k.cannot("aria2-next could not be started: " + err.Error())
	}
	pid := fmt.Sprint(n)
	if writeText(filepath.Join(workdir, "pid"), pid+"\n") != nil {
		k.cannot("cannot record the pid of aria2-next")
	}

	// ⚠ A BOUNDED WAIT ON THE WORK, NOT ON THE CLOCK. The daemon is up when it
	// answers, so this asks it; a fixed sleep would be too short on a loaded
	// runner and wasted everywhere else. conventions/shell.md section 10.
	tokenParam := `["token:` + token + `"]`
	for waited := 0; !(rpc("aria2.getVersion", tokenParam, "rpc-version.json") && !failed("rpc-version.json")); {
		// ⛔ A DEAD DAEMON IS REPORTED AS ONE, rather than waited out. A build that
		// refused its own options exits immediately, and without this the step
		// would spend the whole deadline discovering it.
		if !alive(pid) {
			k.refuse("aria2-next exited before its RPC server answered; " + workdir + "/client.out says why")
		}
		waited++
		if waited >= 30 {
			k.refuse("aria2-next did not answer RPC on port " + rpcPort + " within 30 seconds")
		}
		time.Sleep(time.Second)
	}

	// ⭐ THE CONTAINMENT SWITCHES ARE READ BACK FROM THE BUILD, WHICH IS THE ONE
	// QUESTION RPC ANSWERS AND THE COMMAND LINE CANNOT. A command line reports
	// what was PASSED; `aria2.getGlobalOption` reports what took EFFECT. This
	// target accepts an option it does not have and skips it, so the difference
	// between those two is not theoretical here - it is the measured behaviour of
	// `--enable-dht6`.
	if !rpc("aria2.getGlobalOption", tokenParam, "rpc-options.json") {
		k.refuse("aria2-next would not report its global options")
	}
	if failed("rpc-options.json") {
		k.refuse("aria2-next refused to report its global options")
	}
	options, _ := os.ReadFile(filepath.Join(workdir, "rpc-options.json"))
	// ⛔ EVERY DISCOVERY SWITCH THIS FILE PASSES IS REPORTED, AND FALSE; EVERY KEY
	// OF THOSE FAMILIES THE BUILD REPORTS, NAMED OR NOT, reads `false` too
	// (`ed2k-port-mapping` was on in 3.0.2 and named nowhere); and a switch that
	// is ON is one this file has judged. A family is still a set of spellings, and
	// the next surface need not share one: libtorrent calls local discovery LSD,
	// and 3.0.0's notes say its ED2K speaks Kad. So every option the build reports
	// as `true` must be named in aria2NextHarmless, each read off 3.0.2's own dump
	// and none a discovery surface. ⚠ A surface a build turns on with another
	// value, such as a port or `auto`, is not caught here.
	// ⛔ AND THE PORT THE OBSERVER IS WATCHING IS READ BACK FROM THE BUILD. A peer
	// port that silently did not take is a capture watching a port nothing is on,
	// and it would look exactly like a client that never announced.
	// ⭐ One function, in a2nwindows.go, so the Windows half asks the same
	// questions of its build in the same words.
	if msg := aria2NextOptionsRefusal(options, port); msg != "" {
		k.refuse(msg)
	}

	// ⭐ THE TORRENT GOES IN OVER RPC, base64-ENCODED, WHICH IS THE CASE
	// CLIENT-14 IS ABOUT. `aria2.addTorrent` takes the torrent's bytes rather than
	// a path, so the build is handed the same object the observer wrote instead of
	// a filename it re-reads.
	body, err := os.ReadFile(torrent)
	b64 := base64.StdEncoding.EncodeToString(body)
	if err != nil || b64 == "" {
		k.cannot("the torrent at " + torrent + " could not be encoded")
	}
	if !rpc("aria2.addTorrent", `["token:`+token+`","`+b64+`"]`, "rpc-add.json") {
		k.refuse("aria2-next would not accept the torrent over RPC")
	}
	if failed("rpc-add.json") {
		added, _ := os.ReadFile(filepath.Join(workdir, "rpc-add.json"))
		k.refuse("aria2-next refused the torrent: " + strings.TrimRight(string(added), "\n"))
	}
	added, _ := os.ReadFile(filepath.Join(workdir, "rpc-add.json"))
	var gids []string
	for _, line := range lines(added) {
		if m := aria2NextResultRe.FindStringSubmatch(line); m != nil {
			gids = append(gids, m[1])
		}
	}
	gid := strings.TrimRight(strings.Join(gids, "\n"), "\n")
	if gid == "" {
		k.refuse("aria2-next accepted the torrent and named no download")
	}
	_ = writeText(filepath.Join(workdir, "gid"), gid+"\n")
	fmt.Printf("started aria2-next pid %s on port %s, download %s, RPC %s\n", pid, port, gid, rpcPort)
}

func aria2NextStop(k *adapterKit, args []string) {
	if len(args) < 1 {
		k.cannot("stop takes <workdir>")
	}
	workdir := args[0]
	// ⚠ A stop over a build that already exited is done, not refused.
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	rpcErr := filepath.Join(workdir, "rpc.err")
	rpcPort := aria2NextRPCPort()

	// ⭐ THE PRODUCT'S OWN SHUTDOWN FIRST. `aria2.shutdown` is the same question a
	// signal asks and a better-behaved answer to it: the build closes its sockets,
	// writes its session state and exits on its own terms. ⚠ The signal path below
	// is kept, because a build that ignores its own RPC is exactly the case a
	// capture must still be able to end.
	tokenFile := filepath.Join(workdir, "rpc-token")
	if isFile(tokenFile) {
		b, _ := os.ReadFile(tokenFile)
		token := strings.TrimRight(string(b), "\n")
		if isFile(filepath.Join(workdir, "rpc-port")) {
			p, _ := os.ReadFile(filepath.Join(workdir, "rpc-port"))
			rpcPort = strings.TrimRight(string(p), "\n")
		}
		aria2NextRPC(rpcPort, rpcErr, "aria2.shutdown", `["token:`+token+`"]`,
			filepath.Join(workdir, "rpc-shutdown.json"))
		// ⛔ GONE BEFORE ANYTHING BUNDLES THIS DIRECTORY, and gone whether or not
		// the shutdown was accepted. `capture-client` calls `stop` and then packs
		// the workdir, so this unlink is the whole of what keeps the token out of
		// the artifact. ⚠ It is removed HERE rather than at the end of `stop`,
		// because every path below this line can return.
		_ = os.Remove(tokenFile)
		// ⚠ TEN SECONDS, AND THE NUMBER IS MEASURED RATHER THAN ROUND. Driven on
		// 2026-09-09: `aria2.shutdown` answered `OK` and the build was still alive
		// five seconds later, so a five-second grace fell through to a signal every
		// time and the graceful path this branch exists for never once completed.
		if waitGone(pid, 10) {
			return
		}
	}
	termThenKill(pid, 5)
}
