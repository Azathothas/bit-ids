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
// ⭐ MEASURED on 2026-10-21, by capturing both rootless routes in `bit-check
// contained` on the session host, and by `capture-client` run 39577251480's
// release lane before that:
//
//   - a torrent as a positional argument makes it announce to the tracker named
//     inside, needing no other control: three announces and six peer streams
//     in three sessions;
//   - its peer ID is `A2-1-37-0-` and ten RANDOM RAW BYTES, new per session,
//     which the announce percent-encodes. `capture-client`'s peer-ID guard read
//     the transcript as a substring until this target refused it.
//
// ⛔ STILL NOT MEASURED: that `--enable-dht`, `--enable-dht6`, `--bt-enable-lpd`
// and `--enable-peer-exchange` turn off every adjacent surface, and that
// `--listen-port` refuses zero, which is why an unset peer port drops it.
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
	"strings"
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
// ⚠ The three differ, which is the point: the package route installs Ubuntu's
// `.deb`, whose `/usr/bin/aria2c` is a 14-kilobyte shim over `libaria2.so.0`,
// and the release route compiles the vendor's source tarball into a
// self-contained executable. `ACQ-03` measured both on 2026-09-08.
//
// ⛔ AND THE SOURCE ROUTE IS NOT `elf-binary`, decided on 2026-10-21. Both
// compile one executable with one configure line, and they arrive differently:
// the release tarball is the vendor's DISTRIBUTION, carrying a `configure` the
// vendor generated, and the tag is the vendor's REPOSITORY, whose build system
// this host generates with `autoreconf -i` and an `autopoint` it fetched. So
// `elf-binary-git` names the same installed form by the delivery that differs,
// and a reader of two aria2 records can tell them apart by `package` alone.
// `CLIENT-05`.
func aria2PackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "deb", true
	case "release":
		return "elf-binary", true
	case "source":
		return "elf-binary-git", true
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
		// ⭐ A FLOOR OF 1, BECAUSE THE SOURCE ROUTE READS REFS AND REFS CARRY THE
		// 0.x HISTORY. Measured on 2026-10-21: `git ls-remote --tags --refs` answers
		// 152 tags, every one `release-`, and eleven of them the three-component
		// scheme cannot order - `0.4.0a` and ten package revisions such as
		// `0.10.0+1` - so with no floor `resolve-source` failed closed over its own
		// history. All eleven lead with 0 and no 1.x tag is unorderable, so 1 is the
		// lowest floor that releases them. ⚠ The release listing's selection does
		// not move: 1.37.0 stands above it. `CLIENT-05`.
		fmt.Print("release_floor_major=1\n")
		fmt.Print("release_series=-\n")
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
		// ⛔ THE ROUTES THAT COMPILE, SO THE WORKFLOW BOUNDS THEM AS BUILDS. 145
		// seconds measured here for the release route, under the 420 a download
		// gets, and a build still; the source route adds a clone, a private index
		// read and `autoreconf` to the same build.
		fmt.Print("compiles=release,source\n")

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
		case "source":
			aria2Source(k, workdir)
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
	aria2Build(k, workdir, srcdir, prefix, "release")
}

// aria2ConfigureFlags is the one configure line both compiling routes run.
//
// ⚠ THE CONFIGURE OPTIONS ARE PART OF WHAT A ROUTE INSTALLS. The same source
// configured differently is a build with different features, which `ACQ-03`
// measured: two 1.37.0 builds reporting one version and enabling different
// things. BitTorrent is the surface this project measures and it is on by
// default; the rest are dependencies a capture does not need.
// ⛔ ONE LIST FOR BOTH ROUTES, so the release and source builds differ in what
// they were built from and in nothing this file chose. `CLIENT-05`.
// ⛔ `--prefix` puts the result ahead of the system one on PATH, which is what
// makes the lookup find THIS build rather than a package.
func aria2ConfigureFlags(prefix string) []string {
	return []string{"--prefix=" + prefix, "--without-libxml2", "--without-libexpat",
		"--without-sqlite3", "--without-libcares", "--without-libssh2", "--with-openssl", "--disable-nls"}
}

// aria2Build configures, builds and installs one source tree into the prefix,
// for the release route and the source route alike.
//
// ⭐ The configure line and the host's own flags go to `build-configure.txt`
// beside the install, as Transmission's and aria2-next's builds write theirs,
// so the install artifact says how the build was configured.
func aria2Build(k *adapterKit, workdir, srcdir, prefix, route string) {
	log := filepath.Join(workdir, "install.log")
	flags := aria2ConfigureFlags(prefix)
	var b strings.Builder
	fmt.Fprintf(&b, "./configure")
	for _, a := range flags {
		fmt.Fprintf(&b, " '%s'", a)
	}
	fmt.Fprintf(&b, "\nhost CFLAGS=%s\n", hostFlag("CFLAGS"))
	fmt.Fprintf(&b, "host CXXFLAGS=%s\n", hostFlag("CXXFLAGS"))
	fmt.Fprintf(&b, "host CPPFLAGS=%s\n", hostFlag("CPPFLAGS"))
	fmt.Fprintf(&b, "host LDFLAGS=%s\n", hostFlag("LDFLAGS"))
	if writeText(filepath.Join(workdir, "build-configure.txt"), b.String()) != nil {
		k.refuse("the " + route + " route could not record how it configured the build")
	}
	c := exec.Command("./configure", flags...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: srcdir}) != 0 {
		k.refuse("the " + route + " route could not configure a build")
	}
	c = exec.Command("getconf", "_NPROCESSORS_ONLN")
	jobs, rc := capture(c, stdio{err: os.DevNull})
	if rc != 0 {
		jobs = "2"
	}
	c = exec.Command("make", "-j"+jobs)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: srcdir}) != 0 {
		k.refuse("the " + route + " route could not build aria2")
	}
	c = exec.Command("make", "install")
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: srcdir}) != 0 {
		k.refuse("the " + route + " route built aria2 and could not install it")
	}
	if !isExec(filepath.Join(prefix, "bin", "aria2c")) {
		k.refuse("the " + route + " route reported an install and left no aria2c in " + prefix + "/bin")
	}
}

// aria2Source is the source route.
//
// ⭐ THE SECOND ROOTLESS ROUTE, AND IT EXISTS BECAUSE THE FIRST ONE CANNOT RUN
// ON THE CAPTURE PATH. Measured on `capture-client` run 39577251480, 2026-10-21:
// the package lane's `apt-get update` answered *Could not open lock file
// /var/lib/apt/lists/lock - open (13: Permission denied)*, which is `ACQ-06`
// working as designed - installs are rootless and a package route is privileged
// by construction. So the pair is the release tarball and the repository's own
// tag, resolved through `resolve-source` from refs rather than from the release
// listing, and cloned rather than fetched, which is the shape Transmission and
// aria2-next measured. ⚠ Whoever controls the repository controls both.
func aria2Source(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	k.need(func(n string) string { return "the source route builds a checkout and " + n + " is not on this host" },
		"git", "autoreconf", "autoconf", "automake", "aclocal", "libtoolize", "m4", "xz", "make", "cc", "c++")
	repo := k.describeField("release_repo")
	if repo == "" {
		k.cannot("this adapter declares no release_repo to clone from")
	}
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	log := filepath.Join(workdir, "install.log")
	// ⛔ BOUNDED, SHALLOW AND ONE TAG, as `aria2-next`'s clone is and for its
	// reasons. aria2 carries no submodules.
	src := filepath.Join(workdir, "src")
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/"+repo+".git", src)
	if rc := run(c, stdio{in: os.DevNull, out: log, err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	k.cloneCommit(workdir, tag)
	// ⛔ A TAG IS NOT A DISTRIBUTION: IT CARRIES NO `configure`. The vendor's
	// README says a checkout is built with `autoreconf -i`, and its
	// `configure.ac` calls `AM_GNU_GETTEXT_VERSION`, so `autoreconf` runs
	// `autopoint` - which a runner does not have. ⚠ Measured on 2026-10-21:
	// `ubuntu-24.04` image 20261012.334.1 lists autoconf 2.71, automake 1.16.5,
	// libtool 2.4.7, m4 and pkg-config and no gettext or autopoint, and the
	// session host, on the same versions, had no `autopoint` and no gettext m4
	// either. So `autopoint` comes from the distribution's signed index into the
	// workdir, rootless, as Transmission's libcurl headers do.
	autopoint, datadir := aria2Autopoint(k, workdir)
	c = exec.Command("autoreconf", "-i")
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: src,
		env: []string{"AUTOPOINT=" + autopoint, "gettext_datadir=" + datadir}}) != 0 {
		k.refuse("the source route could not generate the build system with autoreconf -i")
	}
	if !isExec(filepath.Join(src, "configure")) {
		k.refuse("autoreconf -i reported success and left no configure in the checkout")
	}
	aria2Build(k, workdir, src, prefix, "source")
}

// aria2Autopoint fetches the distribution's `autopoint` into the workdir and
// answers the script and the data directory it reads its archive from. What it
// took is written to `build-deps.txt`.
//
// ⛔ ROOTLESS AND WITHOUT CHANGING THE HOST, because `ACQ-06` forbids
// escalating. `apt-get` with its index and cache in the workdir runs as an
// unprivileged user and downloads through the distribution's signed index;
// `dpkg-deb -x` unpacks the package there. ⚠ `autopoint` is a shell script
// whose data directory is compiled in as `/usr/share/gettext` and overridable
// through `gettext_datadir`, which is how the unpacked copy finds its own
// `archive.dir.tar.xz` rather than a host's.
func aria2Autopoint(k *adapterKit, workdir string) (string, string) {
	deps := filepath.Join(workdir, "deps")
	if !mkdirP(deps+"/apt/lists/partial", deps+"/apt/cache/archives/partial", deps+"/deb", deps+"/root") {
		k.cannot("cannot create " + deps)
	}
	k.need(func(n string) string { return "the source route needs autopoint and " + n + " is not on this host" },
		"apt-get", "dpkg-deb")
	apt := aptPrivate(deps + "/apt")
	log := filepath.Join(workdir, "deps.log")
	c := exec.Command("timeout", append(append([]string{"300", "apt-get"}, apt...), "update")...)
	if run(c, stdio{in: os.DevNull, out: log, err: "&1"}) != 0 {
		k.refuse("a private package index could not be fetched for autopoint")
	}
	c = exec.Command("timeout", append(append([]string{"120", "apt-get"}, apt...), "download", "autopoint")...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: deps + "/deb"}) != 0 {
		k.refuse("autopoint could not be downloaded")
	}
	debs := shGlob(deps + "/deb/autopoint_*.deb")
	if len(debs) != 1 || !isFile(debs[0]) {
		k.refuse("the download left no single autopoint package")
	}
	deb := debs[0]
	c = exec.Command("dpkg-deb", "-x", deb, deps+"/root")
	if run(c, stdio{out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("autopoint could not be unpacked")
	}
	script := deps + "/root/usr/bin/autopoint"
	datadir := deps + "/root/usr/share/gettext"
	if !isExec(script) || !isFile(datadir+"/archive.dir.tar.xz") {
		k.refuse("the autopoint package carried no usr/bin/autopoint beside usr/share/gettext/archive.dir.tar.xz")
	}
	field := func(name string) string {
		c := exec.Command("dpkg-deb", "-f", deb, name)
		v, _ := capture(c, stdio{})
		return v
	}
	digest, _ := fileSHA256(deb)
	record := "package=" + field("Package") + "\n" + "version=" + field("Version") + "\n" +
		"sha256=" + digest + "\n"
	if writeText(filepath.Join(workdir, "build-deps.txt"), record) != nil {
		k.refuse("the build could not record its dependency")
	}
	return script, datadir
}
