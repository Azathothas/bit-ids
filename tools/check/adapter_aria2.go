//go:build !windows

// adapter_aria2 - the CLIENT-05 capture adapter, `bit-check adapter aria2`.
//
// CI-11. It was scripts/capture/adapters/aria2.sh, and the reasons that file
// gave are kept here with the lines they explain. The contract is
// scripts/capture/adapters/README.md.
//
// ⭐ THE SIMPLEST TARGET IN THE MATRIX, AND THAT IS WHY IT IS HERE FIRST AMONG
// EQUALS. aria2c takes every setting on its command line, needs no daemon, no
// profile directory and no interactive acceptance, and prints its version to
// stdout. Nothing about a run of it depends on state a previous run left behind,
// which is the property a capture host is built to guarantee and an adapter
// should not also have to.
//
// ⚠ CLIENT-05 ALSO WANTS aria2 AS A CONNECTOR, AND THIS IS NOT THAT. Driving
// aria2 as a target and using it to corroborate another target are the two roles
// the entry says must not create circular corroboration. This file is the target
// half alone; nothing here observes anything.
//
// -- ⛔ WHAT THIS ASSUMES ABOUT THE PRODUCT, AND WHAT IS NOW MEASURED ----------
//
// ⭐ MEASURED on 2026-09-08, against installed aria2 1.37.0 - both an Ubuntu
// package build and one compiled here from the vendor's release tarball.
// Installing a product and asking its version is not a capture, so it needed no
// disposable host:
//
//   - `aria2c --version` opens with `aria2 version 1.37.0`, so the third field
//     is the version - confirmed on both builds;
//   - the package route installs without a prompt, and on `ubuntu-24.04` it
//     installs NOTHING, because the image already ships aria2;
//   - the release route builds: 19 seconds to configure, 126 seconds to
//     `make -j4`, and the result answers with BitTorrent enabled.
//
// ⛔ STILL NOT MEASURED, because each needs a capture:
//
//   - `--enable-dht`, `--enable-dht6`, `--bt-enable-lpd` and
//     `--enable-peer-exchange` are the switches for the three adjacent surfaces;
//   - `--listen-port` refuses zero, which is why an unset peer port drops it;
//   - a torrent as a positional argument makes it announce to the tracker named
//     inside, needing no other control.
//
// ⚠ A claim here that turns out to be wrong shows up as a capture that refuses
// rather than as one that lies: `capture-client` will not attest to a build it
// did not observe announce.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() { registerAdapter("aria2", shAdapter("aria2", aria2Adapter)) }

// aria2Binary is the installed executable, looked up once.
//
// ⛔ A path composed per subcommand is a second spelling, and the day a route
// installs somewhere else one subcommand finds the build and another reports it
// absent. ⭐ THE RELEASE ROUTE'S OWN PREFIX, BEFORE THE PATH. It lets a run
// point the route somewhere else and still be measuring the build that route
// produced. ⛔ Without it the two routes would race on PATH order, which is a
// property of the host rather than of the acquisition.
func aria2Binary(k *adapterKit) (string, bool) {
	return k.lookup(os.Getenv("BIT_IDS_ARIA2"), "aria2c", false)
}

// aria2PackageFormat is what each route delivers, as `Build.package` spells it.
//
// ⚠ The two differ, which is the point: the package route installs Ubuntu's
// `.deb`, whose `/usr/bin/aria2c` is a 14-kilobyte shim over `libaria2.so.0`,
// and the release route compiles the vendor's source tarball into a
// self-contained executable. `ACQ-03` measured both on 2026-09-08.
func aria2PackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "deb", true
	case "release":
		return "elf-binary", true
	}
	return "", false
}

func aria2Adapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Print("target=aria2\n")
		fmt.Print("kind=stock\n")
		// ⭐ AND WHERE THE BUILD IS, WHEN THERE IS ONE. `install-client` asks twice -
		// once before the route runs and once after - so a route that replaced the
		// executable is visible as a changed path or a changed digest even when both
		// builds report the same version. ⛔ That case is real rather than
		// theoretical: `aria2` ships on `ubuntu-24.04` at the same version the vendor
		// publishes, so a release route there installs a genuinely different build
		// and a version comparison alone reports it as no acquisition at all.
		// ⚠ The key is OMITTED when nothing is installed, which is an answer rather
		// than an empty value: a caller that saw `binary=` could not tell a missing
		// build from an adapter that declines to say.
		if where, _ := aria2Binary(k); where != "" {
			fmt.Printf("binary=%s\n", where)
		}
		// ⛔ AND WHAT THE RELEASE ROUTE NEEDS TO KNOW, DECLARED HERE BECAUSE THIS IS
		// ALREADY THE FILE THAT KNOWS HOW THIS PRODUCT IS INSTALLED. Which repository
		// publishes it, how it spells a version and which of a release's artifacts is
		// the installable one are all target knowledge, and `resolve-release` reads
		// them from here rather than from a table a workflow would have to carry a
		// case statement over.
		//
		// ⭐ `{version}` IS WHAT MAKES THE PATTERN UNAMBIGUOUS. Measured 2026-09-09
		// against the live listing: release-1.37.0 carries six assets, three of them
		// source archives differing only in compression, so `aria2-*.tar.*` matches
		// three and `select-asset` refuses. `aria2-{version}.tar.bz2` matches one.
		//
		// ⚠ bz2 RATHER THAN gz OR xz, and the choice is recorded rather than
		// implied: the three carry the same source, the bz2 is what `ACQ-03` already
		// measured a build from, and its digest is the one this project can check a
		// fetch against without acquiring anything new.
		fmt.Print("release_repo=aria2/aria2\n")
		fmt.Print("release_tag_prefix=release-\n")
		fmt.Print("release_min_components=3\n")
		fmt.Print("release_max_components=3\n")
		fmt.Print("release_floor_major=-\n")
		fmt.Print("release_asset=aria2-{version}.tar.bz2\n")
		// ⛔ THE DIGEST DISPOSITION, DECLARED ONCE AND READ BY THE ROUTE, AND IT IS
		// ABOUT THE VENDOR'S OWN CHECKSUMS DOCUMENT. ⚠ That is a narrower sentence
		// than "no digest", and the difference is measured: the release LISTING
		// carries a per-asset digest published by the party serving the bytes, which
		// `bit-check resolve-release` resolves out of the same response and records as
		// `source-listing`. This key says only that the VENDOR uploaded no checksums
		// file - which is what an adapter can know by reading a release, and is what
		// decides whether the resolver goes looking for one.
		// ⚠ AND THIS TARGET IS THE ONE THAT REALLY HAS NEITHER. Read through
		// `AGENTS.md` rule 8's route on 2026-09-17: release-1.37.0 carries six
		// archives, no checksums asset, and `digest` is null on every one of the six
		// - the only one of this project's four targets for which that is true.
		fmt.Print("release_digest_unpublished=this release publishes six archives and no checksums asset\n")
		// ⛔ THE ROUTE THAT COMPILES, SO THE WORKFLOW BOUNDS IT AS A BUILD. 145
		// seconds measured here, under the 420 a download gets, and a build still.
		fmt.Print("compiles=release\n")

	case "install":
		route, workdir, format := k.installArgs(args, aria2PackageFormat)
		switch route {
		case "package":
			// ⚠ The distribution's own index. Its version trails upstream, and that
			// is a packaging observation rather than a defect: ACQ-03 compares what
			// the two installed builds REPORT, and a divergence is recorded.
			k.aptPackageRoute(workdir, "the package route did not install aria2", "aria2")
		case "release":
			aria2Release(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		binary, _ := aria2Binary(k)
		if binary == "" {
			k.cannot("aria2c is not installed")
		}
		// ⛔ THE BUILD SPEAKING. `aria2 version 1.37.0` on its first line; the field
		// taken is the version alone, and an empty answer is could-not-run rather
		// than an empty string somebody downstream renders as unknown.
		// ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED. The
		// shell half was once `$(... | head -1) || cannot`, whose `||` read HEAD's
		// status: head exits 0 over anything, so the guard was dead code and a
		// build that refused to answer reached the parse instead of the refusal.
		// ⚠ Measured on 2026-09-08, when a client capture reported "would not
		// report a version" and nothing said which of three refusals had fired.
		c := exec.Command(binary, "--version")
		output, rc := capture(c, stdio{in: os.DevNull, err: os.DevNull})
		if rc != 0 {
			k.cannot(fmt.Sprintf("aria2c --version exited %d", rc))
		}
		line := firstLine(output)
		version := ""
		if f := shFields(line); len(f) >= 3 {
			version = f[2]
		}
		if version == "" {
			k.cannot("aria2c reported no parseable version: [" + line + "]")
		}
		k.versionReport(output, version)

	case "start":
		if len(args) < 3 {
			k.cannot("start takes <torrent> <workdir> <peer-port>")
		}
		torrent, workdir, port := args[0], args[1], args[2]
		binary, _ := aria2Binary(k)
		if binary == "" {
			k.cannot("aria2c is not installed")
		}
		if !isNonEmpty(torrent) {
			k.cannot("there is no torrent at " + torrent)
		}
		if !mkdirP(filepath.Join(workdir, "downloads")) {
			k.cannot("cannot create " + workdir + "/downloads")
		}
		// ⛔ ALL THREE ADJACENT SURFACES OFF, EXPLICITLY. A host with no default
		// route cannot reach a DHT bootstrap node, and it can still multicast a
		// local discovery announce and gossip over peer exchange. Turning off one
		// and inheriting the others is the one-gated-door shape.
		//
		// ⚠ --seed-time=0 and a stop timeout keep the process from outliving the
		// observer's own deadline: this returns while the build runs, and `stop`
		// below is what ends it, but a build that ignored both would hold the port
		// after the evidence was written.
		flags := []string{
			"--enable-dht=false",
			"--enable-dht6=false",
			"--bt-enable-lpd=false",
			"--enable-peer-exchange=false",
			"--bt-require-crypto=false",
			"--dir=" + workdir + "/downloads",
			"--seed-time=0",
			"--allow-overwrite=true",
			"--console-log-level=info",
			"--summary-interval=0",
		}
		// ⚠ ZERO IS NOT A LISTEN PORT aria2 ACCEPTS, so an unset peer port drops the
		// flag rather than passing a value the build would refuse. It is appended
		// here rather than written into the list above, because a list built twice
		// to vary one element is two lists that drift.
		if port != "0" {
			flags = append(flags, "--listen-port="+port)
		}
		c := exec.Command(binary, append(flags, torrent)...)
		pid, err := startBackground(c, stdio{out: filepath.Join(workdir, "client.out"), err: "&1"})
		if err != nil {
			k.cannot("aria2c could not be started: " + err.Error())
		}
		if writeText(filepath.Join(workdir, "pid"), fmt.Sprintf("%d\n", pid)) != nil {
			k.cannot("cannot record the pid of aria2c")
		}
		fmt.Printf("started aria2c pid %d on port %s\n", pid, port)

	case "stop":
		if len(args) < 1 {
			k.cannot("stop takes <workdir>")
		}
		// ⚠ A stop over a build that already exited is done, not refused. aria2c
		// ends itself when its seed time expires, and a capture whose client
		// finished early is a capture, not a failure.
		pid := readPID(filepath.Join(args[0], "pid"))
		if pid == "" {
			return
		}
		// ⚠ Given a moment, then killed. A client asked to stop writes its state and
		// closes its sockets; one that ignores the ask holds the port open into the
		// upload step.
		termThenKill(pid, 5)
	}
}

// aria2Release is the release route.
func aria2Release(k *adapterKit, workdir string) {
	// ⛔ THE VENDOR'S OWN PUBLISHED ARTIFACT, AND NOT A SECOND PACKAGE INDEX.
	// E-ACQ-07 and E-ACQ-08 refuse two routes sharing a resolver or a delivery
	// mechanism, so a "second route" that asked another mirror of the same index
	// would be one route under two names.
	//
	// ⛔ AND IT INSTALLS WHAT IT FETCHED. This route used to `curl` the tarball
	// into the workdir and return 0, which is a route that reports an install it
	// did not perform: `version` then answered from whatever was already on PATH,
	// so a second route would have measured the FIRST route's build and the two
	// would have agreed for the most trivial reason available. Measured on
	// 2026-09-08 and recorded in `ACQ-03`.
	//
	// ⛔ AN aria2 RELEASE CARRIES NO LINUX BINARY, which is a property of the
	// target rather than a gap here: the 1.37.0 release publishes source tarballs,
	// two Windows zips and an Android build. So the Linux release route is a
	// source build. Measured on this project's own host: 19 seconds to configure
	// and 126 seconds to `make -j4`.
	//
	// ⚠ WHAT THIS ASSUMES AND HAS NOT MEASURED ON A RUNNER: that a C++ toolchain
	// and OpenSSL headers are already present. If they are not, `configure`
	// refuses and says so with its own log, which is a route that failed rather
	// than one that silently produced nothing. ⛔ It deliberately does NOT apt-get
	// its build dependencies: that would make this route reach the package index,
	// and arguing afterwards about whether headers count as acquisition is worse
	// than refusing.
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	k.need(func(string) string { return "curl is not on this host" }, "curl")
	k.need(func(n string) string { return "the release route builds from source and " + n + " is not on this host" },
		"tar", "make", "cc")
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	// ⛔ THE FETCH AND THE DIGEST ARE ONE CALL AND IT IS NOT THIS FILE'S. This
	// route BUILDS what it fetches rather than placing it, so it asks the
	// installer for `--fetch` alone - the bounded transfer and the digest
	// disposition - and unpacks the settled file itself. ⚠ A route that unpacked
	// bytes whose digest nothing had settled would be running a vendor's build
	// system over something nobody identified.
	disposition := k.releaseDisposition()
	fetch := append([]string{"--fetch", "--url", url, "--into", filepath.Join(workdir, "aria2.tar.bz2")},
		disposition...)
	fetch = append(fetch, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not fetch the vendor's source archive", fetch...)
	log := filepath.Join(workdir, "install.log")
	if !mkdirP(filepath.Join(workdir, "src")) {
		k.cannot("cannot create " + workdir + "/src")
	}
	c := exec.Command("tar", "-xjf", filepath.Join(workdir, "aria2.tar.bz2"), "-C", filepath.Join(workdir, "src"))
	if run(c, stdio{out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("the release tarball could not be unpacked")
	}
	// ⚠ ONE DIRECTORY, FOUND RATHER THAN COMPOSED. The tarball's top-level name
	// carries the version, and composing it here would be a second spelling of a
	// value the archive already states.
	c = exec.Command("find", filepath.Join(workdir, "src"), "-mindepth", "1", "-maxdepth", "1", "-type", "d")
	found, _ := capture(c, stdio{})
	srcdir := firstLine(found)
	if srcdir == "" {
		k.refuse("the release tarball unpacked no source directory")
	}
	// ⚠ THE CONFIGURE OPTIONS ARE PART OF WHAT THIS ROUTE INSTALLS. The same
	// source configured differently is a build with different features, which
	// `ACQ-03` measured: two 1.37.0 builds reporting one version and enabling
	// different things. BitTorrent is the surface this project measures and it is
	// on by default; the rest are dependencies a capture does not need.
	// ⛔ `--prefix` puts the result ahead of the system one on PATH, which is what
	// makes the lookup find THIS build rather than a package.
	c = exec.Command("./configure", "--prefix="+prefix, "--without-libxml2", "--without-libexpat",
		"--without-sqlite3", "--without-libcares", "--without-libssh2", "--with-openssl", "--disable-nls")
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: srcdir}) != 0 {
		k.refuse("the release route could not configure a build")
	}
	c = exec.Command("getconf", "_NPROCESSORS_ONLN")
	jobs, rc := capture(c, stdio{err: os.DevNull})
	if rc != 0 {
		jobs = "2"
	}
	c = exec.Command("make", "-j"+jobs)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: srcdir}) != 0 {
		k.refuse("the release route could not build aria2")
	}
	c = exec.Command("make", "install")
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: srcdir}) != 0 {
		k.refuse("the release route built aria2 and could not install it")
	}
	if !isExec(filepath.Join(prefix, "bin", "aria2c")) {
		k.refuse("the release route reported an install and left no aria2c in " + prefix + "/bin")
	}
}
