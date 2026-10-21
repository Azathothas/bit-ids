//go:build !windows

// adapter_qbittorrent-enhanced - the CLIENT-02 capture adapter on Linux,
// `bit-check adapter qbittorrent-enhanced`.
//
// The contract is scripts/capture/adapters/README.md. ⚠ The Windows half
// registers the same name from adapter_qbittorrent-enhanced_windows.go; one
// platform builds one of them. What either half decides without a process in
// it is qbtenhanced.go, which `bit-check qbittorrent-enhanced --selftest` asks
// on any host.
//
// ⛔ NOT THE `qbittorrent` ADAPTER UNDER ANOTHER NAME. Enhanced is a fork whose
// observable identity this project measures for itself, so this adapter
// resolves Enhanced's own releases and tags, installs under its own executable
// name, finds nothing through PATH, and refuses a build that does not answer
// Enhanced's four-component version. It borrows the `qbittorrent` adapter's
// MECHANICS for the source build - the private apt unpack and the Qt archive
// fetch, each checked against its sidecar - and none of its decisions.
//
// -- the routes, measured on 2026-10-21 -----------------------------------------
//
// ⛔ `release`: the vendor's `qbittorrent-enhanced-nox_x86_64-linux-musl_static.zip`
// from `release-5.2.4.10`, fetched by `install-rootless --fetch` under the
// listing's digest, and its one `qbittorrent-nox` placed at
// `<prefix>/bin/qbittorrent-enhanced-nox`. ⭐ The headless build, not the
// AppImage beside it: both carry libtorrent 1.2.20, and only the GUI needs a
// display server and a graphics stack the runner does not carry.
// ⛔ `source`: the `release-5.2.4.10` tag from the repository's own refs, built
// with `-DGUI=OFF -DSTACKTRACE=OFF` as the vendor's `cross_build.sh` builds its
// nox, against Qt 6.11.2's archives and libtorrent `v1.2.20` built static from
// its own tag, which is the libtorrent series all three vendor builds carry.
// ⚠ The vendor's build takes libtorrent from the moving `RC_1_2` branch, which
// was ahead of the tag when measured; the tag is the immutable reference.
//
// ⛔ NO PACKAGE ROUTE ON LINUX. Ubuntu carries no Enhanced, the AUR is not a
// route this host can use, and a package route is privileged, which `ACQ-06`
// keeps off the capture path.
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

func init() {
	registerAdapter(qbeTarget, shAdapter(qbeTarget, qbittorrentEnhancedAdapter))
}

// qbeInstalledName is the file both routes install. ⛔ NOT `qbittorrent-nox`,
// which is the parent's name too: a parent's nox on the same host must never
// be the file this adapter finds.
const qbeInstalledName = "qbittorrent-enhanced-nox"

// qbeLibtorrentTag is the libtorrent the source route builds.
const qbeLibtorrentTag = "v1.2.20"

// qbeBuildPackages is what the source route unpacks from noble's index: 7-Zip
// for Qt's archives and Boost's headers. ⚠ The nox build loads no graphics
// stack, so none of the `qbittorrent` adapter's runtime set is needed.
var qbeBuildPackages = []string{"7zip", "libboost1.83-dev"}

// qbeBinary is the installed build, or "": an override, then the prefix, and
// ⛔ NEVER PATH.
func qbeBinary(k *adapterKit) string {
	if v := os.Getenv("BIT_IDS_QBITTORRENT_ENHANCED"); v != "" {
		return v
	}
	p, ok := k.prefix(true)
	if !ok {
		return ""
	}
	if exe := filepath.Join(p, "bin", qbeInstalledName); isExec(exe) {
		return exe
	}
	return ""
}

func qbittorrentEnhancedAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Printf("target=%s\n", qbeTarget)
		fmt.Print("kind=stock\n")
		if exe := qbeBinary(k); exe != "" {
			fmt.Printf("binary=%s\n", exe)
		}
		fmt.Print(qbeDescribeScheme(qbeLinuxAsset))
		// ⚠ The source route compiles libtorrent and the nox build.
		fmt.Print("compiles=source\n")
		fmt.Print("routes=release,source\n")

	case "install":
		route, workdir, format := k.installArgs(args, func(r string) (string, bool) { return qbePackageFormat(false, r) })
		switch route {
		case "release":
			qbeRelease(k, workdir)
		case "source":
			qbeSource(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		qbeVersion(k)

	case "start":
		qbeStart(k, args)

	case "stop":
		if len(args) < 1 {
			k.cannot("stop takes <workdir>")
		}
		workdir := args[0]
		// ⛔ The profile is scrubbed on every path out, though the Linux profile
		// writes no password: the build rewrites its own file at exit.
		defer func() {
			if n, err := qbtWinScrub(filepath.Join(workdir, "profile")); err == nil && n > 0 {
				k.say(fmt.Sprintf("removed %d credential line(s) from the profile", n))
			}
		}()
		pid := readPID(filepath.Join(workdir, "pid"))
		if pid == "" {
			return
		}
		// ⭐ SIGTERM is the nox build's own graceful exit: it shuts its session
		// down, which announces `stopped`. KILL after eight seconds.
		termThenKill(pid, 8)
	}
}

// qbePlace stages a file at <prefix>/bin/.<name>.new and renames it into place.
func qbePlace(k *adapterKit, write func(staged string) error) string {
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	if !mkdirP(prefix + "/bin") {
		k.refuse("this user cannot create " + prefix + "/bin; a rootless install writes nowhere else")
	}
	staged := prefix + "/bin/." + qbeInstalledName + ".new"
	if err := write(staged); err != nil {
		_ = os.Remove(staged)
		k.refuse("the build could not be staged in " + prefix + "/bin: " + err.Error())
	}
	final := prefix + "/bin/" + qbeInstalledName
	if !mvFile(staged, final) {
		k.refuse("the build could not be installed at " + final)
	}
	return final
}

// qbeRelease is the release route. ⛔ The URL is resolved before the route is
// cut and handed in; the fetch and the digest are install-rootless's one call.
// ⚠ `--fetch`, not `--install`: the artifact is a zip, and placing it as the
// executable would be the one mistake this route exists not to make.
func qbeRelease(k *adapterKit, workdir string) {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	k.need(func(string) string { return "curl is not on this host" }, "curl")
	zipPath := filepath.Join(workdir, "qbittorrent-enhanced-nox.zip")
	fetch := append([]string{"--fetch", "--url", url, "--into", zipPath}, k.releaseDisposition()...)
	fetch = append(fetch, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not fetch the vendor's zip", fetch...)
	var size int64
	exe := qbePlace(k, func(staged string) error {
		n, err := qbeUnzipNox(zipPath, staged)
		size = n
		return err
	})
	zipSum, _ := fileSHA256(zipPath)
	exeSum, _ := fileSHA256(exe)
	note := fmt.Sprintf("archive=%s\narchive_sha256=%s\nentry=%s\nexecutable=%s\nexecutable_bytes=%d\nexecutable_sha256=%s\n",
		filepath.Base(zipPath), zipSum, qbeNoxExe, exe, size, exeSum)
	if writeText(filepath.Join(workdir, "extract.txt"), note) != nil {
		k.refuse("the route could not record what it took out of the zip")
	}
}

// qbeSource is the source route: the repository's own tag, cloned and built.
func qbeSource(k *adapterKit, workdir string) {
	tag := os.Getenv("BIT_IDS_SOURCE_TAG")
	if tag == "" {
		k.cannot("the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut")
	}
	k.need(func(n string) string { return "the source route compiles and " + n + " is not on this host" },
		"git", "cmake", "ninja", "c++", "curl", "python3")
	log := filepath.Join(workdir, "install.log")
	src := filepath.Join(workdir, "src")
	c := exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", tag,
		"https://github.com/"+qbeRepo+".git", src)
	if rc := run(c, stdio{in: os.DevNull, out: log, err: "&1"}); rc != 0 {
		if rc == 124 {
			k.refuse("the source route's clone of " + tag + " did not finish within 600s")
		}
		k.refuse("the source route could not clone " + tag)
	}
	k.cloneCommit(workdir, tag)
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	if !mkdirP(prefix+"/bin", prefix+"/opt") {
		k.refuse("this user cannot create " + prefix + "/bin; a rootless install writes nowhere else")
	}
	deps := filepath.Join(workdir, "build-deps.txt")
	_ = writeText(deps, "")
	qbittorrentAptUnpack(k, workdir, workdir+"/build-root", deps, qbeBuildPackages...)
	// ⛔ Qt LIVES IN THE PREFIX, because the build's run path names it and the
	// capture runs the build after this workdir's purpose is over.
	qt := prefix + "/opt/qt-6.11.2"
	qbittorrentFetchQt(k, workdir, qt)

	lt := filepath.Join(workdir, "libtorrent")
	c = exec.Command("timeout", "600", "git", "clone", "--depth", "1", "--branch", qbeLibtorrentTag,
		"--recurse-submodules", "--shallow-submodules", "https://github.com/arvidn/libtorrent.git", lt)
	if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
		k.refuse("libtorrent " + qbeLibtorrentTag + " could not be cloned")
	}
	commit, _ := capture(exec.Command("git", "-C", lt, "rev-parse", "HEAD"), stdio{})
	if appendText(deps, fmt.Sprintf("libtorrent %s %s\n", qbeLibtorrentTag, commit)) != nil {
		k.refuse("the build could not record libtorrent's commit")
	}
	usr := workdir + "/build-root/usr"
	ltPrefix := workdir + "/libtorrent-prefix"
	cmake := func(what string, args ...string) {
		c := exec.Command("cmake", args...)
		if run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1"}) != 0 {
			k.refuse(what)
		}
	}
	ltBuild := workdir + "/libtorrent-build"
	cmake("libtorrent could not be configured", "-S", lt, "-B", ltBuild, "-G", "Ninja",
		"-DCMAKE_BUILD_TYPE=Release", "-DBUILD_SHARED_LIBS=OFF", "-DCMAKE_CXX_STANDARD=17",
		"-DBoost_INCLUDE_DIR="+usr+"/include", "-DCMAKE_INSTALL_PREFIX="+ltPrefix)
	cmake("libtorrent could not be built", "--build", ltBuild)
	cmake("libtorrent could not be staged", "--install", ltBuild)

	// ⛔ THE VENDOR'S NOX SWITCHES, `-DGUI=OFF -DSTACKTRACE=OFF`, read from its
	// `cross_build.sh`; the rest is where this host's pieces are.
	flags := []string{"-G", "Ninja", "-DCMAKE_BUILD_TYPE=Release", "-DGUI=OFF", "-DSTACKTRACE=OFF",
		"-DCMAKE_CXX_STANDARD=17",
		"-DCMAKE_PREFIX_PATH=" + qt + ";" + ltPrefix + ";" + usr,
		"-DCMAKE_INCLUDE_PATH=" + usr + "/include", "-DBoost_INCLUDE_DIR=" + usr + "/include",
		"-DCMAKE_INSTALL_RPATH=" + qt + "/lib", "-DCMAKE_BUILD_WITH_INSTALL_RPATH=ON"}
	var b strings.Builder
	b.WriteString("cmake -S <source> -B <build>")
	for _, a := range flags {
		fmt.Fprintf(&b, " '%s'", a)
	}
	fmt.Fprintf(&b, "\nhost CFLAGS=%s\nhost CXXFLAGS=%s\nhost LDFLAGS=%s\n", hostFlag("CFLAGS"), hostFlag("CXXFLAGS"), hostFlag("LDFLAGS"))
	if writeText(filepath.Join(workdir, "build-configure.txt"), b.String()) != nil {
		k.refuse("the source route could not record how it configured the build")
	}
	build := filepath.Join(workdir, "build")
	cmake("the source route could not configure a build", append([]string{"-S", src, "-B", build}, flags...)...)
	cmake("the source route could not build qbittorrent-nox", "--build", build)
	built := build + "/" + qbeNoxExe
	if !isExec(built) {
		k.refuse("the source route built nothing at build/" + qbeNoxExe)
	}
	qbePlace(k, func(staged string) error {
		if !cpFile(built, staged) {
			return fmt.Errorf("cannot copy %s", built)
		}
		return nil
	})
}

// qbeVersion asks the installed build. ⛔ THE BUILD SPEAKING: `--version`
// prints and exits before any session starts. ⚠ HOME is a directory made for
// this call and removed after it, because Qt makes its cache and config
// directories even for `--version`, measured on 2026-10-21.
func qbeVersion(k *adapterKit) {
	exe := qbeBinary(k)
	if exe == "" {
		k.cannot(qbeInstalledName + " is not installed")
	}
	home, err := os.MkdirTemp("", "qbe-version-")
	if err != nil {
		k.cannot("cannot make a temporary home")
	}
	defer os.RemoveAll(home)
	output, rc := capture(exec.Command(exe, "--version"), stdio{in: os.DevNull, err: "&1", env: []string{"HOME=" + home}})
	if rc != 0 {
		k.cannot(fmt.Sprintf("%s --version exited %d: %s", qbeInstalledName, rc, strings.ReplaceAll(firstLine(output), "\n", " ")))
	}
	version, err := qbeParseNoxVersion(output)
	if err != nil {
		k.cannot(err.Error())
	}
	k.versionReport(output, version)
}

func qbeStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	exe := qbeBinary(k)
	if exe == "" {
		k.cannot(qbeInstalledName + " is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	profile := workdir + "/profile"
	config := profile + "/qBittorrent/config"
	if !mkdirP(config, workdir+"/downloads") {
		k.cannot("cannot create the profile under " + workdir)
	}
	// ⛔ THE WHOLE PROFILE IS WRITTEN, NOT PATCHED, AND ASKED BEFORE IT IS:
	// every surface and every Enhanced switch at false, or nothing starts.
	text := qbeProfile(workdir+"/downloads", port, 0, "")
	if msg := qbeProfileRefusal(text); msg != "" {
		k.refuse("the profile would start the build with " + msg)
	}
	if writeText(config+"/qBittorrent.conf", text) != nil {
		k.cannot("the profile could not be written")
	}
	// ⭐ THE TORRENT IS A POSITIONAL ARGUMENT, the one control the product
	// already has that needs no Web UI, no credential and no second process.
	c := exec.Command(exe, "--profile="+profile, "--relative-fastresume", torrent)
	n, err := startBackground(c, stdio{in: os.DevNull, out: workdir + "/client.out", outAppend: true, err: "&1",
		env: []string{"HOME=" + workdir}})
	if err != nil {
		k.cannot(qbeInstalledName + " could not be started: " + err.Error())
	}
	_ = writeText(workdir+"/pid", fmt.Sprintf("%d\n", n))
	fmt.Printf("started %s pid %d on port %s\n", qbeInstalledName, n, port)
}
