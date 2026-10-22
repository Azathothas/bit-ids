//go:build !windows

// adapter_qbittorrent - the CLIENT-01 capture adapter on Linux, `bit-check
// adapter qbittorrent`.
//
// CI-11. It was scripts/capture/adapters/qbittorrent.sh, and the reasons that
// file gave are kept here with the lines they explain. The contract is
// scripts/capture/adapters/README.md. ⚠ The Windows half registers the same
// name from adapter_qbittorrent_windows.go; one platform builds one of them.
//
// ⚠ THE HIGHEST-PRIORITY TARGET AND NOT THE SIMPLEST ONE. qBittorrent keeps a
// profile directory, asks once for a legal notice to be accepted, and reads its
// switches out of an INI file rather than off its command line. Each of those
// is a place a run can differ from the previous one, which is what a disposable
// host exists to remove, so this adapter writes the whole profile itself instead
// of inheriting whatever is in HOME.
//
// -- ⭐ DRIVEN ON 2026-10-15, BOTH ROUTES, 5.2.4 ----------------------------------
//
// The release route installs the `_lt20` AppImage and the source route builds
// `release-5.2.4` against Qt 6.11.2's archives and libtorrent 2.0.15. Each
// answered `qBittorrent v5.2.4` and captured three sessions in the namespace,
// under Xvfb: six announces and six peer streams, every peer ID `-qB5240-`. The
// two records assembled `publishable` and `build_equivalent`. `CLIENT-01`.
//
// -- ⛔ WHAT THIS ASSUMED ABOUT THE PRODUCT BEFORE THAT ------------------------
//
// Kept for the record. Each line was a claim a dispatch settled:
//
//   - `--version` ends its first line with the version, with a leading `v`
//     this strips;
//   - `--profile` roots the whole configuration, so a written profile is the
//     one that is read;
//   - ⛔ **REFUTED on 2026-09-08.** `--confirm-legal-notice` was assumed to be
//     what stops a fresh profile blocking on a prompt. This build answers
//     `Bad command line: --confirm-legal-notice is an unknown command line
//     parameter`, so it was never a control; the acceptance is written into the
//     profile as `[LegalNotice] Accepted=true` instead;
//   - the `Session\DHTEnabled`, `Session\PeXEnabled` and `Session\LSDEnabled`
//     keys under `[BitTorrent]` are the three adjacent surfaces;
//   - a torrent as a positional argument is added and started.
//
// ⚠ The prompt claim is the one worth watching. If it is wrong the capture does
// not lie, it stalls: the observer serves its whole deadline, nothing announces,
// and `capture-client` refuses over a build that put no bytes on the wire.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter("qbittorrent", shAdapter("qbittorrent", qbittorrentAdapter)) }

// qbittorrentExe is the program both routes install.
//
// ⛔ THE GUI BUILD, `qbittorrent`, AND NOT `qbittorrent-nox`. Measured on
// 2026-10-15 by reading both 5.2.4 AppImages' squashfs: neither carries
// `qbittorrent-nox`. The operator settled that both routes run the GUI build
// under Xvfb, so the two routes are one program. `CLIENT-01`.
const qbittorrentExe = "qbittorrent"

func qbittorrentBinary(k *adapterKit) (string, bool) {
	return k.lookup(os.Getenv("BIT_IDS_QBITTORRENT"), qbittorrentExe, false)
}

// -- ⭐ WHAT BOTH ROUTES RUN AGAINST, FROM THE DISTRIBUTION'S SIGNED INDEX -----
//
// ⛔ AN AppImage LEAVES THE HOST'S GRAPHICS STACK OUT ON PURPOSE, and the Qt
// archives the source route builds against assume it. Measured on 2026-10-15:
// the `_lt20` AppImage stops on a missing `libEGL.so.1` before printing
// anything, and Qt's xcb plugin names `libxcb-cursor0` and seven more that a
// runner need not carry. So one set of noble's runtime packages is unpacked into
// the prefix for BOTH routes, which then load the same host libraries rather
// than whatever each runner happens to have.
// ⚠ Rootless: `apt-get` keeps its index and cache in the workdir and
// `dpkg-deb -x` unpacks, which is `transmission`'s technique for libcurl's
// headers. `ACQ-06` is why nothing here may escalate.
// ⛔ THE WHOLE CLOSURE THE GUI LOADS, NOT WHAT ONE HOST LACKED. The first list
// was the eight libraries Qt's xcb plugin named on the session host, and
// `capture-client` run 38702739119's source lane stopped on the runner with Qt's
// own message about `xcb-cursor0`, which it prints whatever dependency is
// missing. So this is every library `ldd` resolves for Qt's xcb plugin, Qt's
// libraries and the build, outside Qt and outside the libraries every runner
// carries (glibc, libstdc++, zlib, OpenSSL and the compression and crypto
// libraries). Measured on 2026-10-15.
var qbittorrentRuntimePackages = strings.Fields(`libegl1 libgl1 libglx0 libopengl0 libglvnd0 libgles2
libxcb-cursor0 libxcb-icccm4 libxcb-image0 libxcb-keysyms1 libxcb-render-util0
libxcb-util1 libxcb-xkb1 libxkbcommon0 libxkbcommon-x11-0
libxcb1 libxcb-randr0 libxcb-render0 libxcb-shape0 libxcb-shm0 libxcb-sync1
libxcb-xfixes0 libx11-6 libx11-xcb1 libxau6 libxdmcp6 libbsd0 libmd0
libfontconfig1 libfreetype6 libpng16-16t64 libbrotli1 libexpat1 libdbus-1-3`)

// -- ⭐ THE SOURCE ROUTE: THE TAG, BUILT AGAINST QT'S OWN PREBUILT ARCHIVES ------
//
// Settled by the operator on 2026-10-15: Qt 6.11.2's `linux_gcc_64` archives
// from `download.qt.io`, libtorrent 2.0.15 built from its own tag so both routes
// run the libtorrent the `_lt20` AppImage carries, and noble's Boost headers.
// ⛔ QT'S ARCHIVES ARE NAMED BY QT'S OWN LISTING, never composed. `Updates.xml`
// names the package's version and its archives, and each archive's `.sha1`
// sidecar is checked before it is unpacked. ⚠ That sidecar is the host's
// statement about bytes the same host serves, as a release listing's digest is.
const (
	qbittorrentQtRepo        = "https://download.qt.io/online/qtsdkrepository/linux_x64/desktop/qt6_6112/qt6_6112"
	qbittorrentQtPackage     = "qt.qt6.6112.linux_gcc_64"
	qbittorrentLibtorrentTag = "v2.0.15"
	// The listing reader, run as `python3 -I -c`.
	qbittorrentQtListing = `
import sys, xml.etree.ElementTree as ET
root = ET.parse(sys.argv[1]).getroot()
for p in root.findall("PackageUpdate"):
    if p.findtext("Name") == sys.argv[2]:
        print("version", p.findtext("Version"))
        for a in (p.findtext("DownloadableArchives") or "").split(","):
            print("archive", a.strip())
`
)

// ⚠ Only what the build reads: qtbase, qtsvg, qttools for `lrelease`, which
// itself loads qtdeclarative's `libQt6Qml`, and Qt's own ICU.
var (
	qbittorrentQtModules     = []string{"qtbase", "qtsvg", "qttools", "qtdeclarative", "icu"}
	qbittorrentBuildPackages = strings.Fields(`7zip libboost1.83-dev libgl-dev libglx-dev libopengl-dev libegl-dev
libgles-dev libglvnd-dev libglvnd-core-dev mesa-common-dev libxkbcommon-dev`)
)

func qbittorrentRuntimeDir(k *adapterKit) (string, bool) {
	p, ok := k.prefix(false)
	if !ok {
		return "", false
	}
	return p + "/lib/qbittorrent-runtime", true
}

// qbittorrentAptUnpack fetches packages through a private index and unpacks
// them into root, recording each package, version and digest in record.
func qbittorrentAptUnpack(k *adapterKit, workdir, root, record string, packages ...string) {
	k.need(func(n string) string { return "the install needs " + n + " and it is not on this host" },
		"apt-get", "dpkg-deb")
	apt := workdir + "/apt"
	if !mkdirP(apt+"/lists/partial", apt+"/cache/archives/partial", apt+"/deb", root) {
		k.cannot("cannot create " + apt)
	}
	log := filepath.Join(workdir, "deps.log")
	if !exists(apt + "/updated") {
		c := exec.Command("timeout", append(append([]string{"300", "apt-get"}, aptPrivate(apt)...), "update")...)
		if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
			k.refuse("a private package index could not be fetched")
		}
		_ = writeText(apt+"/updated", "")
	}
	c := exec.Command("timeout", append(append(append([]string{"300", "apt-get"}, aptPrivate(apt)...), "download"), packages...)...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", dir: apt + "/deb"}) != 0 {
		k.refuse("the packages could not be downloaded: " + strings.Join(packages, " "))
	}
	for _, pkg := range packages {
		debs := shGlob(apt + "/deb/" + pkg + "_*.deb")
		if len(debs) != 1 || !isFile(debs[0]) {
			k.refuse("the download left no single " + pkg + " package")
		}
		deb := debs[0]
		c := exec.Command("dpkg-deb", "-x", deb, root)
		if run(c, stdio{}) != 0 {
			k.refuse(pkg + " could not be unpacked")
		}
		field := func(name string) string {
			c := exec.Command("dpkg-deb", "-f", deb, name)
			v, _ := capture(c, stdio{})
			return v
		}
		digest, _ := fileSHA256(deb)
		line := fmt.Sprintf("%s=%s sha256:%s\n", field("Package"), field("Version"), digest)
		if appendText(record, line) != nil {
			k.refuse("the install could not record " + pkg)
		}
	}
}

// qbittorrentInstallRuntime puts the runtime set into the prefix. Both routes
// call it.
func qbittorrentInstallRuntime(k *adapterKit, workdir string) {
	rt, ok := qbittorrentRuntimeDir(k)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	qbittorrentAptUnpack(k, workdir, workdir+"/runtime-root", workdir+"/runtime-packages.txt",
		qbittorrentRuntimePackages...)
	if os.RemoveAll(rt+".new") != nil {
		k.refuse("cannot clear " + rt + ".new")
	}
	if !mkdirP(rt + ".new") {
		k.refuse("cannot stage " + rt)
	}
	libs := shGlob(workdir + "/runtime-root/usr/lib/x86_64-linux-gnu/*.so*")
	if !cpFile(append(append([]string{"-P"}, libs...), rt+".new/")...) {
		k.refuse("the runtime set carried no libraries")
	}
	if os.RemoveAll(rt) != nil {
		k.refuse("cannot clear " + rt)
	}
	if !mvFile(rt+".new", rt) {
		k.refuse("cannot install " + rt)
	}
}

// qbittorrentStartDisplay starts a display, writes its pid to pidfile, and
// answers the display.
//
// ⭐ THE DISPLAY THE GUI BUILD NEEDS, and only that. Both AppImages carry the
// xcb and wayland platform plugins and nothing headless, so the build cannot
// start without a server. ⚠ Xvfb listens on a filesystem socket only, which a
// network namespace leaves working, and on no TCP port.
func qbittorrentStartDisplay(k *adapterKit, pidfile string) string {
	k.need(func(string) string { return "the GUI build needs Xvfb and it is not on this host" }, "Xvfb")
	for n := 91; n < 120; n++ {
		socket := fmt.Sprintf("/tmp/.X11-unix/X%d", n)
		if exists(socket) || exists(fmt.Sprintf("/tmp/.X%d-lock", n)) {
			continue
		}
		c := exec.Command("Xvfb", fmt.Sprintf(":%d", n), "-nolisten", "tcp", "-screen", "0", "1280x800x24")
		pid, err := startBackground(c, stdio{in: os.DevNull, out: os.DevNull, err: os.DevNull})
		if err != nil {
			k.cannot("Xvfb could not be started: " + err.Error())
		}
		_ = writeText(pidfile, fmt.Sprintf("%d\n", pid))
		for waited := 0; waited < 50; waited++ {
			if isSocket(socket) {
				return fmt.Sprintf(":%d", n)
			}
			time.Sleep(100 * time.Millisecond)
		}
		signalPID(readPID(pidfile), syscall.SIGTERM)
		k.cannot(fmt.Sprintf("Xvfb :%d did not open its socket within five seconds", n))
	}
	k.cannot("no free X display between :91 and :119")
	return ""
}

func qbittorrentStopDisplay(pidfile string) {
	if !isFile(pidfile) {
		return
	}
	xpid := readPID(pidfile)
	if xpid == "" {
		return
	}
	signalPID(xpid, syscall.SIGTERM)
	_ = os.Remove(pidfile)
}

// qbittorrentBuild is one call into the build.
//
// ⛔ ONE ENVIRONMENT FOR EVERY CALL INTO THE BUILD, so `version` and `start` run
// the same program the same way on both routes.
// ⛔ THE RUNTIME DIRECTORY IS AN ARGUMENT, resolved by the caller before it
// changes anything. The prefix follows `HOME`, and `start` runs the build with
// `HOME` set to its workdir; resolved in here, the directory was the workdir's,
// which holds no libraries, and every session stopped on a missing
// `libEGL.so.1`. Driven on 2026-10-15.
func qbittorrentBuildEnv(display, rt string) []string {
	return []string{"DISPLAY=" + display, "QT_QPA_PLATFORM=xcb", "LD_LIBRARY_PATH=" + rt}
}

// qbittorrentPackageFormat is what each route delivers, as `Build.package`
// spells it.
//
// ⚠ An AppImage is not an ELF binary this project can treat as one: it is a
// self-mounting archive carrying its own runtime, so a record calling it
// `elf-binary` would say the two routes delivered one form when they did not.
func qbittorrentPackageFormat(route string) (string, bool) {
	switch route {
	case "package":
		return "deb", true
	case "release":
		return "appimage", true
	case "source":
		return "elf-binary", true
	}
	return "", false
}

func qbittorrentAdapter(k *adapterKit, sub string, args []string) {
	// ⛔ AN AppImage MOUNTS ITSELF WITH FUSE, AND A CAPTURE HOST NEED NOT HAVE IT.
	// This makes the AppImage extract itself and run instead, which costs an
	// extraction per call and works on a host with FUSE as well as one without. ⚠
	// It is set here rather than at each call site so `version`, `start` and any
	// later subcommand inherit one answer; a build that is not an AppImage ignores
	// it.
	_ = os.Setenv("APPIMAGE_EXTRACT_AND_RUN", "1")
	switch sub {
	case "describe":
		fmt.Print("target=qbittorrent\n")
		fmt.Print("kind=stock\n")
		// ⭐ AND WHERE THE BUILD IS, WHEN THERE IS ONE; the key is OMITTED when
		// nothing is installed. `aria2` says why at length.
		if where, _ := qbittorrentBinary(k); where != "" {
			fmt.Printf("binary=%s\n", where)
		}
		// ⛔ WHAT THE RELEASE ROUTE NEEDS TO KNOW. `CLIENT-01` recorded on 2026-09-05
		// that this release "offers a Linux AppImage, a Windows x64_setup.exe and a
		// source tar.xz"; re-read on 2026-09-09, release-5.2.3 carries FOURTEEN
		// assets and TWO Linux AppImages - `qbittorrent-5.2.3_x86_64.AppImage` and
		// `qbittorrent-5.2.3_lt20_x86_64.AppImage` - so the summary was one build
		// short in exactly the place a route has to choose.
		//
		// ⭐ `{version}` IS WHAT SEPARATES THEM. `qbittorrent-*_x86_64.AppImage`
		// matches both, because a run swallows the `_lt20`, and `select-asset`
		// refuses the ambiguity rather than taking whichever the source listed
		// first.
		//
		// ⚠ THREE OR FOUR COMPONENTS, from `ACQ-02`'s live dry run. The newest
		// release object on 2026-09-09 is `release-5.3.0beta1`, which the source
		// flags a prerelease and whose version text says so too, so the resolver
		// selects 5.2.3 - which is the pessimistic stability rule doing its job on a
		// live listing rather than on a fixture.
		fmt.Print("release_repo=qbittorrent/qBittorrent\n")
		fmt.Print("release_tag_prefix=release-\n")
		fmt.Print("release_min_components=3\n")
		fmt.Print("release_max_components=4\n")
		fmt.Print("release_floor_major=-\n")
		fmt.Print("release_series=-\n")
		// ⛔ THE `_lt20` ASSET, settled by the operator on 2026-10-15 once both were
		// measured: the unsuffixed AppImage carries libtorrent 1.2.20 and this one
		// 2.0.15, and the reserved bytes a pair compares are libtorrent's.
		fmt.Print("release_asset=qbittorrent-{version}_lt20_x86_64.AppImage\n")
		// ⛔ THE DIGEST DISPOSITION, DECLARED ONCE AND READ BY THE ROUTE, AND IT IS
		// ABOUT THE VENDOR'S OWN CHECKSUMS DOCUMENT; `aria2` says why at length.
		// ⚠ MEASURED through rule 8's route on 2026-09-17: release-5.2.3 carries
		// fourteen assets, seven of them detached `.asc` signatures, and no checksums
		// file - and `digest` is stated on all fourteen. So this route verifies, and
		// it verifies against the host rather than against the vendor.
		fmt.Print("release_digest_unpublished=this release signs each asset with a detached .asc and publishes no checksums document\n")
		// ⚠ The source route compiles libtorrent and qBittorrent, measured at about
		// nine minutes on a four-processor session host on 2026-10-15.
		fmt.Print("compiles=source\n")

	case "install":
		route, workdir, format := k.installArgs(args, qbittorrentPackageFormat)
		switch route {
		case "package":
			k.aptPackageRoute(workdir, "the package route did not install qbittorrent", "qbittorrent")
		case "release":
			qbittorrentRelease(k, workdir)
		case "source":
			qbittorrentSource(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		qbittorrentVersion(k)

	case "start":
		qbittorrentStart(k, args)

	case "stop":
		if len(args) < 1 {
			k.cannot("stop takes <workdir>")
		}
		workdir := args[0]
		// ⛔ THE DISPLAY GOES WITH THE BUILD, on every path out of here.
		defer qbittorrentStopDisplay(filepath.Join(workdir, "xvfb.pid"))
		pid := readPID(filepath.Join(workdir, "pid"))
		if pid == "" {
			return
		}
		termThenKill(pid, 8)
	}
}

// qbittorrentRelease is the release route.
//
// ⛔ THE VENDOR'S OWN ARTIFACT. The URL is resolved before the route is cut and
// handed in; resolving it here would be a network call under containment.
// ⛔ AND IT INSTALLS WHAT IT FETCHED. This route used to leave the AppImage in
// the workdir and return 0, which is a route reporting an install it did not
// perform: `version` then answered from whatever was already on PATH. Measured
// on 2026-09-08 and recorded in `ACQ-03`.
// ⭐ AN AppImage IS THE INSTALLED PROGRAM, so installing it is placing it where
// the lookup looks. ⛔ THE FETCH, THE DIGEST AND THE PLACING ARE ONE CALL, AND IT
// IS NOT THIS FILE'S: `bit-check install-rootless` bounds the transfer, settles
// the digest before anything is made executable, and writes into a prefix this
// user already owns. ⚠ This vendor publishes no checksums document, which is
// declared rather than passed over, so the artifact is IDENTIFIED - its digest
// is recorded - and checked against the listing rather than the vendor.
func qbittorrentRelease(k *adapterKit, workdir string) {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	k.need(func(string) string { return "curl is not on this host" }, "curl")
	disposition := k.releaseDisposition()
	install := append([]string{"--install", "--url", url, "--into", filepath.Join(workdir, "qbittorrent.AppImage"),
		"--as", qbittorrentExe}, disposition...)
	install = append(install, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not install the vendor's AppImage", install...)
	qbittorrentInstallRuntime(k, workdir)
}

// qbittorrentSource is the source route.
//
// ⭐ THE SECOND ROUTE: THE REPOSITORY'S OWN TAG, CLONED AND BUILT, as
// `transmission` and `aria2-next` do. It resolves through the repository's refs,
// which since 2026-10-15 read `release-5.3.0beta1`'s marker after its digits and
// select `release-5.2.4`. `CLIENT-01`.
func qbittorrentSource(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	k.need(func(string) string { return "the source route clones and git is not on this host" }, "git")
	repo := k.describeField("release_repo")
	if repo == "" {
		k.cannot("this adapter declares no release_repo to clone from")
	}
	// ⚠ A clone that hit its bound is told from one that failed, as the other
	// two source routes tell it; the shell half did not. 2026-10-15.
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/"+repo+".git", filepath.Join(workdir, "src"))
	if rc := run(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "install.log"), err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	k.cloneCommit(workdir, tag)
	qbittorrentBuildSource(k, workdir, filepath.Join(workdir, "src"))
	qbittorrentInstallRuntime(k, workdir)
}

func qbittorrentFetchQt(k *adapterKit, workdir, qt string) {
	k.need(func(string) string { return "reading Qt's listing needs python3" }, "python3")
	log := filepath.Join(workdir, "install.log")
	listing := filepath.Join(workdir, "qt-Updates.xml")
	c := exec.Command("timeout", "120", "curl", "-fsSL", "--retry", "2", "-o", listing, qbittorrentQtRepo+"/Updates.xml")
	if run(c, stdio{in: os.DevNull, err: log, errAppend: true}) != 0 {
		k.refuse("Qt's listing could not be fetched")
	}
	read := filepath.Join(workdir, "qt-listing.txt")
	c = exec.Command("python3", "-I", "-c", qbittorrentQtListing, listing, qbittorrentQtPackage)
	if run(c, stdio{out: read}) != 0 {
		k.refuse("Qt's listing could not be read")
	}
	text, _ := os.ReadFile(read)
	rows := strings.Split(string(text), "\n")
	version := ""
	for _, row := range rows {
		if f := shFields(row); len(f) >= 1 && f[0] == "version" {
			if len(f) > 1 {
				version = f[1]
			}
			break
		}
	}
	if version == "" {
		k.refuse("Qt's listing names no " + qbittorrentQtPackage)
	}
	if os.RemoveAll(qt) != nil {
		k.refuse("cannot clear " + qt)
	}
	if !mkdirP(qt, filepath.Join(workdir, "qt-archives")) {
		k.refuse("cannot create " + qt)
	}
	for _, module := range qbittorrentQtModules {
		archive := ""
		for _, row := range rows {
			if f := shFields(row); len(f) >= 2 && f[0] == "archive" && strings.HasPrefix(f[1], module+"-") {
				archive = f[1]
				break
			}
		}
		if archive == "" {
			k.refuse("Qt's listing names no " + module + " archive for " + qbittorrentQtPackage)
		}
		file := filepath.Join(workdir, "qt-archives", version+archive)
		for _, suffix := range []string{"", ".sha1"} {
			c := exec.Command("timeout", "300", "curl", "-fsSL", "--retry", "2", "-o", file+suffix,
				qbittorrentQtRepo+"/"+qbittorrentQtPackage+"/"+version+archive+suffix)
			if run(c, stdio{in: os.DevNull, err: log, errAppend: true}) != 0 {
				k.refuse("Qt's " + module + " archive could not be fetched")
			}
		}
		sidecar, _ := os.ReadFile(file + ".sha1")
		var want []string
		for _, l := range strings.Split(strings.TrimSuffix(string(sidecar), "\n"), "\n") {
			if len(l) > 40 {
				l = l[:40]
			}
			want = append(want, l)
		}
		wanted := strings.TrimRight(strings.Join(want, "\n"), "\n")
		got := ""
		if f, err := os.Open(file); err == nil {
			h := sha1.New()
			if _, err := io.Copy(h, f); err == nil {
				got = hex.EncodeToString(h.Sum(nil))
			}
			f.Close()
		}
		if wanted != got {
			k.refuse("Qt's " + module + " archive digests to " + got + " and its sidecar says " + wanted)
		}
		c := exec.Command(workdir+"/build-root/usr/lib/7zip/7z", "x", "-y", "-bd", "-o"+qt, file)
		if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
			k.refuse("Qt's " + module + " archive could not be unpacked")
		}
		_ = appendText(filepath.Join(workdir, "build-deps.txt"),
			fmt.Sprintf("qt %s %s sha1:%s\n", version, archive, got))
	}
	// ⚠ Qt's ICU archive unpacks at the root, and Qt's libraries find it beside
	// themselves through their own `$ORIGIN` run path.
	for _, icu := range shGlob(qt + "/libicu*.so*") {
		if !exists(icu) {
			continue
		}
		if !mvFile(icu, qt+"/lib/") {
			k.refuse("Qt's ICU could not be placed beside Qt's libraries")
		}
	}
}

func qbittorrentBuildSource(k *adapterKit, workdir, src string) {
	k.need(func(n string) string { return "the source route compiles and " + n + " is not on this host" },
		"cmake", "ninja", "c++", "git", "curl", "sha1sum")
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	if !mkdirP(prefix+"/bin", prefix+"/opt") {
		k.refuse("this user cannot create " + prefix + "/bin; a rootless install writes nowhere else")
	}
	deps := filepath.Join(workdir, "build-deps.txt")
	_ = writeText(deps, "")
	qbittorrentAptUnpack(k, workdir, workdir+"/build-root", deps, qbittorrentBuildPackages...)
	// ⚠ The runtime set is unpacked into the build root too, so the build links
	// against the libraries it will load.
	qbittorrentAptUnpack(k, workdir, workdir+"/build-root", deps, qbittorrentRuntimePackages...)
	// ⛔ QT LIVES IN THE PREFIX, because the build's run path names it and the
	// capture runs the build after this workdir's purpose is over.
	qt := prefix + "/opt/qt-6.11.2"
	qbittorrentFetchQt(k, workdir, qt)
	log := filepath.Join(workdir, "install.log")
	lt := filepath.Join(workdir, "libtorrent")
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", qbittorrentLibtorrentTag,
		"--recurse-submodules", "--shallow-submodules", "https://github.com/arvidn/libtorrent.git", lt)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("libtorrent " + qbittorrentLibtorrentTag + " could not be cloned")
	}
	c = exec.Command("git", "-C", lt, "rev-parse", "HEAD")
	commit, _ := capture(c, stdio{})
	if appendText(deps, fmt.Sprintf("libtorrent %s %s\n", qbittorrentLibtorrentTag, commit)) != nil {
		k.refuse("the build could not record libtorrent's commit")
	}
	usr := workdir + "/build-root/usr"
	ltBuild := workdir + "/libtorrent-build"
	flags := []string{"-G", "Ninja", "-DCMAKE_BUILD_TYPE=Release", "-DBUILD_SHARED_LIBS=OFF",
		"-DCMAKE_CXX_STANDARD=17", "-DBoost_INCLUDE_DIR=" + usr + "/include",
		"-DCMAKE_INSTALL_PREFIX=" + workdir + "/libtorrent-prefix"}
	c = exec.Command("cmake", append([]string{"-S", lt, "-B", ltBuild}, flags...)...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("libtorrent could not be configured")
	}
	c = exec.Command("cmake", "--build", ltBuild)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("libtorrent could not be built")
	}
	c = exec.Command("cmake", "--install", ltBuild)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("libtorrent could not be staged")
	}
	// ⛔ `-rpath-link`, BECAUSE THE RUN PATH NAMES QT ALONE. Qt's `libQt6Gui`
	// needs `libEGL.so.1`, which is in the unpacked runtime set, and with the
	// install run path in force the linker found it nowhere: the first driven
	// build on 2026-10-15 stopped on `undefined reference to eglBindAPI`. At run
	// time the same set is on `LD_LIBRARY_PATH` for both routes.
	flags = []string{"-G", "Ninja", "-DCMAKE_BUILD_TYPE=Release", "-DGUI=ON", "-DSTACKTRACE=OFF",
		"-DCMAKE_PREFIX_PATH=" + qt + ";" + workdir + "/libtorrent-prefix;" + usr,
		"-DCMAKE_LIBRARY_PATH=" + usr + "/lib/x86_64-linux-gnu", "-DCMAKE_INCLUDE_PATH=" + usr + "/include",
		"-DBoost_INCLUDE_DIR=" + usr + "/include",
		"-DCMAKE_INSTALL_RPATH=" + qt + "/lib", "-DCMAKE_BUILD_WITH_INSTALL_RPATH=ON",
		"-DCMAKE_EXE_LINKER_FLAGS=-Wl,-rpath-link," + usr + "/lib/x86_64-linux-gnu"}
	var b strings.Builder
	b.WriteString("cmake -S <source> -B <build>")
	for _, a := range flags {
		fmt.Fprintf(&b, " '%s'", a)
	}
	fmt.Fprintf(&b, "\nhost CFLAGS=%s\n", hostFlag("CFLAGS"))
	fmt.Fprintf(&b, "host CXXFLAGS=%s\n", hostFlag("CXXFLAGS"))
	fmt.Fprintf(&b, "host LDFLAGS=%s\n", hostFlag("LDFLAGS"))
	if writeText(filepath.Join(workdir, "build-configure.txt"), b.String()) != nil {
		k.refuse("the source route could not record how it configured the build")
	}
	build := filepath.Join(workdir, "build")
	c = exec.Command("cmake", append([]string{"-S", src, "-B", build}, flags...)...)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("the source route could not configure a build")
	}
	c = exec.Command("cmake", "--build", build)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("the source route could not build qbittorrent")
	}
	if !isExec(build + "/qbittorrent") {
		k.refuse("the source route built nothing at build/qbittorrent")
	}
	staged := prefix + "/bin/." + qbittorrentExe + ".new"
	if !cpFile(build+"/qbittorrent", staged) {
		k.refuse("the source route could not stage " + qbittorrentExe + " in " + prefix + "/bin")
	}
	if !mvFile(staged, prefix+"/bin/"+qbittorrentExe) {
		k.refuse("the source route could not install " + qbittorrentExe + " into " + prefix + "/bin")
	}
}

func qbittorrentVersion(k *adapterKit) {
	binary, _ := qbittorrentBinary(k)
	if binary == "" {
		k.cannot(qbittorrentExe + " is not installed")
	}
	// ⛔ THE BUILD SPEAKING. `qBittorrent v5.0.2` on its first line; the leading
	// `v` is stripped because a version is compared by VersionScheme::components
	// and a label carrying a sigil is a second spelling of one value.
	// ⛔ NO --confirm-legal-notice HERE, AND THAT IS MEASURED RATHER THAN
	// ASSUMED. This build answers `Bad command line: --confirm-legal-notice is an
	// unknown command line parameter`, so the flag was not a control at all; it
	// was an argument the product refuses. Client capture run 3 said so in those
	// words. ⚠ Stdin is still the null device and the caller still bounds this,
	// because a prompt is a separate hazard from a rejected flag.
	//
	// ⚠ stderr is KEPT here rather than discarded. A product that refuses to
	// answer says why on it, and the caller prints what this prints.
	// ⚠ THE GUI BUILD ANSWERS ONLY ON A DISPLAY: it constructs its application
	// before it reads `--version`. Measured on 2026-10-15 with both routes'
	// builds; each prints `qBittorrent v5.2.4` under Xvfb.
	tmp, err := os.CreateTemp("", "tmp.")
	if err != nil {
		k.cannot("cannot make a temporary file")
	}
	xpid := tmp.Name()
	tmp.Close()
	// ⚠ DEFERRED, so a display that never starts, whose refusal unwinds from
	// inside qbittorrentStartDisplay, leaves no temporary file behind either.
	// The shell half left it. 2026-10-15.
	defer os.Remove(xpid)
	display := qbittorrentStartDisplay(k, xpid)
	rt, ok := qbittorrentRuntimeDir(k)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	c := exec.Command(binary, "--version")
	output, rc := capture(c, stdio{in: os.DevNull, err: "&1", env: qbittorrentBuildEnv(display, rt)})
	qbittorrentStopDisplay(xpid)
	if rc != 0 {
		// ⚠ `printf '%s' "$OUTPUT" | head -3 | tr '\n' ' '`: the first three
		// lines with their newlines, or the whole answer when it is shorter.
		head := output
		for i, at := 0, 0; i < 3; i++ {
			j := strings.IndexByte(output[at:], '\n')
			if j < 0 {
				break
			}
			at += j + 1
			if i == 2 {
				head = output[:at]
			}
		}
		k.cannot(fmt.Sprintf("%s --version exited %d: %s", qbittorrentExe, rc, strings.ReplaceAll(head, "\n", " ")))
	}
	line := ""
	for _, l := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(l), "qbittorrent") {
			line = l
			break
		}
	}
	if line == "" {
		line = firstLine(output)
	}
	// ⚠ `awk '{ print $NF }' | sed 's/^v//'`, including awk's answer for a line
	// with no fields at all, which is the line itself.
	version := strings.TrimPrefix(line, "v")
	if f := shFields(line); len(f) > 0 {
		version = strings.TrimPrefix(f[len(f)-1], "v")
	}
	if version == "" {
		k.cannot(qbittorrentExe + " reported no parseable version: [" + line + "]")
	}
	k.versionReport(output, version)
}

func qbittorrentStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	binary, _ := qbittorrentBinary(k)
	if binary == "" {
		k.cannot(qbittorrentExe + " is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	profile := workdir + "/profile"
	config := profile + "/qBittorrent/config"
	if !mkdirP(config, workdir+"/downloads") {
		k.cannot("cannot create the profile under " + workdir)
	}
	// ⛔ THE WHOLE PROFILE IS WRITTEN, NOT PATCHED. A capture that edited an
	// existing configuration would measure a build under whatever else was in it,
	// and the settings a run used are part of what that run recorded. ⚠ The three
	// adjacent surfaces are named individually: DHT, peer exchange and local peer
	// discovery are three destinations, and a host with no default route still
	// multicasts.
	var s strings.Builder
	s.WriteString("[BitTorrent]\n")
	s.WriteString("Session\\DHTEnabled=false\n")
	s.WriteString("Session\\PeXEnabled=false\n")
	s.WriteString("Session\\LSDEnabled=false\n")
	s.WriteString("Session\\AnonymousModeEnabled=false\n")
	s.WriteString("Session\\Encryption=0\n")
	fmt.Fprintf(&s, "Session\\DefaultSavePath=%s/downloads\n", workdir)
	s.WriteString("Session\\TempPathEnabled=false\n")
	s.WriteString("Session\\AddTorrentStopped=false\n")
	s.WriteString("Session\\GlobalMaxSeedingMinutes=1\n")
	if port != "0" {
		fmt.Fprintf(&s, "Session\\Port=%s\n", port)
	}
	s.WriteString("\n")
	s.WriteString("[Preferences]\n")
	s.WriteString("Connection\\UPnP=false\n")
	fmt.Fprintf(&s, "Downloads\\SavePath=%s/downloads\n", workdir)
	s.WriteString("General\\Locale=en\n")
	s.WriteString("WebUI\\Enabled=false\n")
	s.WriteString("\n")
	s.WriteString("[LegalNotice]\n")
	s.WriteString("Accepted=true\n")
	// ⚠ THE GUI ASKS ABOUT A TORRENT IT IS HANDED, and nothing would answer.
	// `AddNewTorrentDialog/Enabled` is the 5.2.4 key, read from its source.
	s.WriteString("\n")
	s.WriteString("[AddNewTorrentDialog]\n")
	s.WriteString("Enabled=false\n")
	if writeText(config+"/qBittorrent.conf", s.String()) != nil {
		k.cannot("the profile could not be written")
	}

	// ⛔ THE LEGAL NOTICE IS ACCEPTED IN THE PROFILE, NOT ON THE COMMAND LINE.
	// `--confirm-legal-notice` is an argument this build REFUSES - measured on
	// 2026-09-08, in those words - so passing it here would have failed the start
	// exactly as it failed the version call.
	//
	// ⛔ AND THE TORRENT IS A POSITIONAL ARGUMENT, which is the one control this
	// product already has that needs no web interface, no credential and no second
	// process. An adapter driving the WebUI would be measuring a build through an
	// authenticated API it had to configure first.
	// ⭐ A DISPLAY PER SESSION, stopped with the build, so a session's server is
	// never the previous session's.
	rt, ok := qbittorrentRuntimeDir(k)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	display := qbittorrentStartDisplay(k, workdir+"/xvfb.pid")
	c := exec.Command(binary, "--profile="+profile, "--relative-fastresume", "--skip-dialog=true", torrent)
	env := append([]string{"HOME=" + workdir}, qbittorrentBuildEnv(display, rt)...)
	n, err := startBackground(c, stdio{in: os.DevNull, out: workdir + "/client.out", outAppend: true, err: "&1", env: env})
	if err != nil {
		k.cannot(qbittorrentExe + " could not be started: " + err.Error())
	}
	pid := fmt.Sprint(n)
	_ = writeText(workdir+"/pid", pid+"\n")
	fmt.Printf("started %s pid %s on port %s, display %s\n", qbittorrentExe, pid, port, display)
}
