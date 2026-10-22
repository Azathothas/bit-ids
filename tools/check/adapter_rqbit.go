//go:build !windows

// adapter_rqbit - the ENGINE-03 capture adapter on Linux, `bit-check adapter
// rqbit`.
//
// The contract is scripts/capture/adapters/README.md. ⚠ The Windows half
// registers the same name from adapter_rqbit_windows.go; one platform builds
// one of them. What either half decides without a process in it is rqbit.go,
// which `bit-check rqbit --selftest` asks on any host.
//
// -- the routes, measured on 2026-10-21 -----------------------------------------
//
// ⛔ `release`: the vendor's `rqbit-linux-amd64`, a musl static-pie, placed by
// `install-rootless --install` at `<prefix>/bin/rqbit` under the listing's
// digest.
// ⛔ `package`: crates.io's `rqbit`, `cargo install --locked` with no version
// and no feature flag into the same prefix, compiled by this repository's
// pinned Rust. crates.io's index is the resolver and cargo the delivery, so
// neither is the release listing's; the crate is the vendor's own upload, so
// the ORIGIN is shared, as it is for every pair this project has. ⭐ 216
// seconds on the session host, rootless, as `ubuntu`.
//
// ⚠ NO `source` ROUTE: a build of the tag would read the same repository the
// release listing does, and crates.io is a second index. Ubuntu carries no
// rqbit.
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

func init() { registerAdapter(rqbitTarget, shAdapter(rqbitTarget, rqbitAdapter)) }

// rqbitBinary is the installed build, or "": an override, then the prefix, and
// ⛔ NEVER PATH. A `cargo install rqbit` somebody ran into `~/.cargo/bin` is a
// build neither route acquired.
func rqbitBinary(k *adapterKit) string {
	if v := os.Getenv("BIT_IDS_RQBIT"); v != "" {
		return v
	}
	p, ok := k.prefix(true)
	if !ok {
		return ""
	}
	if exe := filepath.Join(p, "bin", rqbitTarget); isExec(exe) {
		return exe
	}
	return ""
}

func rqbitAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Printf("target=%s\n", rqbitTarget)
		fmt.Print("kind=stock\n")
		if exe := rqbitBinary(k); exe != "" {
			fmt.Printf("binary=%s\n", exe)
		}
		fmt.Print(rqbitDescribe(rqbitLinuxAsset))

	case "install":
		route, workdir, format := k.installArgs(args, func(r string) (string, bool) { return rqbitPackageFormat(false, r) })
		switch route {
		case "release":
			rqbitRelease(k, workdir)
		case "package":
			rqbitCargo(k, workdir)
		default:
			k.cannot("unknown route: " + route)
		}
		k.writePackage(workdir, format)

	case "version":
		rqbitVersion(k)

	case "start":
		rqbitStart(k, args)

	case "stop":
		if len(args) < 1 {
			k.cannot("stop takes <workdir>")
		}
		rqbitStop(k, args[0])
	}
}

// rqbitRelease is the release route. ⛔ The URL is resolved before the route is
// cut and handed in; the fetch, the digest and the placing are install-rootless's
// one call. ⚠ `--install`: the artifact IS the build.
func rqbitRelease(k *adapterKit, workdir string) {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		k.cannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	install := append([]string{"--install", "--url", url, "--into", filepath.Join(workdir, rqbitLinuxAsset),
		"--as", rqbitTarget}, k.releaseDisposition()...)
	install = append(install, "--report", filepath.Join(workdir, "rootless-install.txt"))
	k.rootlessRelease(workdir, "the release route could not install the vendor's build", install...)
}

// rqbitCargo is the package route: crates.io's index resolves, cargo fetches
// and compiles, and cargo's own install record says what it built.
//
// ⛔ CARGO'S STATE IS THE WORKDIR'S AND ITS PROFILE IS THE VENDOR'S, which
// rqbitCargoEnv says why. ⚠ The librqbit build script runs `npm install` and `npm run build` for the web UI,
// which needs npm and the npm registry, as the vendor's own builds do.
func rqbitCargo(k *adapterKit, workdir string) {
	k.need(func(n string) string { return "the package route compiles and " + n + " is not on this host" },
		"cargo", "cc", "npm", "pkg-config", "timeout")
	log := filepath.Join(workdir, "install.log")
	// ⚠ The default `default-tls` feature links the host's OpenSSL, which
	// ubuntu-24.04 carries as `libssl-dev`.
	if run(exec.Command("pkg-config", "--exists", "openssl"), stdio{in: os.DevNull, out: log, err: "&1"}) != 0 {
		k.refuse("the package route links the host's OpenSSL and pkg-config finds no openssl; ubuntu-24.04 ships it as libssl-dev")
	}
	prefix, ok := k.prefix(false)
	if !ok {
		k.cannot("the rootless prefix could not be derived")
	}
	if !mkdirP(prefix + "/bin") {
		k.refuse("this user cannot create " + prefix + "/bin; a rootless install writes nowhere else")
	}
	cargoHome := filepath.Join(workdir, "cargo-home")
	target := filepath.Join(workdir, "target")
	if !mkdirP(cargoHome) {
		k.cannot("cannot create " + cargoHome)
	}
	env := rqbitCargoEnv(cargoHome, target)
	rustc, _ := capture(exec.Command("rustc", "-vV"), stdio{in: os.DevNull, err: "&1", env: env})
	started := time.Now()
	c := exec.Command("timeout", append([]string{"1400", "cargo"}, rqbitCargoArgs(prefix)...)...)
	rc := run(c, stdio{in: os.DevNull, out: log, outAppend: true, err: "&1", env: env})
	took := int(time.Since(started) / time.Second)
	if rc != 0 {
		run(exec.Command("tail", "-20", log), stdio{out: "&2"})
		if rc == 124 {
			k.refuse("cargo install did not finish within 1400s")
		}
		k.refuse(fmt.Sprintf("cargo install %s exited %d", rqbitCrate, rc))
	}
	body, err := os.ReadFile(filepath.Join(prefix, ".crates2.json"))
	if err != nil {
		k.refuse("cargo installed and wrote no .crates2.json in " + prefix)
	}
	_ = os.WriteFile(filepath.Join(workdir, "crates2-json.txt"), body, 0o644)
	in, err := rqbitCargoRecord(body)
	if err != nil {
		k.refuse(err.Error())
	}
	if !isExec(filepath.Join(prefix, "bin", rqbitTarget)) {
		k.refuse("cargo reported an install and left no " + rqbitTarget + " in " + prefix + "/bin")
	}
	// ⭐ THE CRATE CARGO VERIFIED, by its digest: cargo checks it against the
	// index's `cksum` before it unpacks it.
	crate := filepath.Join(cargoHome, "registry", "cache", "*", rqbitCrate+"-"+in.version+".crate")
	crates := shGlob(crate)
	crateSum := ""
	if len(crates) == 1 && isFile(crates[0]) {
		crateSum, _ = fileSHA256(crates[0])
	}
	if crateSum == "" {
		k.refuse("cargo's registry cache holds no single " + rqbitCrate + "-" + in.version + ".crate")
	}
	note := rqbitFeaturesNote(in, crateSum)
	note += fmt.Sprintf("seconds=%d\nhost RUSTFLAGS=%s\nhost CARGO_ENCODED_RUSTFLAGS=%s\nrustc -vV:\n%s\n",
		took, hostFlag("RUSTFLAGS"), hostFlag("CARGO_ENCODED_RUSTFLAGS"), rustc)
	if writeText(filepath.Join(workdir, "build-features.txt"), note) != nil {
		k.refuse("the route could not record the build's feature selection")
	}
	origin := "https://static.crates.io/crates/" + rqbitCrate + "/" + rqbitCrate + "-" + in.version + ".crate"
	if writePackageIndexVia(workdir, rqbitCratesIndex, rqbitCrate, in.version, origin, rqbitDelivery) != nil {
		k.refuse("the route could not record what the index named")
	}
	// ⚠ The build tree is a gigabyte nothing reads again; the crate cache stays.
	_ = os.RemoveAll(target)
	_ = os.RemoveAll(filepath.Join(cargoHome, "registry", "src"))
}

// rqbitVersion asks the installed build. ⛔ THE BUILD SPEAKING: clap prints and
// exits before any session starts. ⚠ HOME is a directory made for this call and
// removed after it, and every `RQBIT_*` is dropped, as `start` does.
func rqbitVersion(k *adapterKit) {
	exe := rqbitBinary(k)
	if exe == "" {
		k.cannot(rqbitTarget + " is not installed")
	}
	home, err := os.MkdirTemp("", "rqbit-version-")
	if err != nil {
		k.cannot("cannot make a temporary home")
	}
	defer os.RemoveAll(home)
	c := exec.Command(exe, "--version")
	c.Env = rqbitChildEnv(os.Environ(), "HOME="+home)
	output, rc := capture(c, stdio{in: os.DevNull, err: "&1"})
	if rc != 0 {
		k.cannot(fmt.Sprintf("%s --version exited %d: %s", rqbitTarget, rc, firstLine(output)))
	}
	version, err := rqbitParseVersion(output)
	if err != nil {
		k.cannot(err.Error())
	}
	k.versionReport(output, version)
}

func rqbitStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	exe := rqbitBinary(k)
	if exe == "" {
		k.cannot(rqbitTarget + " is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	// ⛔ A HOME OF ITS OWN PER SESSION, so whatever a session persists is found
	// by `stop` and nothing reaches the next session through it.
	home := filepath.Join(workdir, "home")
	_ = os.RemoveAll(home)
	downloads := filepath.Join(workdir, "downloads")
	if !mkdirP(home, downloads) {
		k.cannot("cannot create the build's home under " + workdir)
	}
	api := rqbitAPIPortFor()
	flags, err := rqbitServerArgs(downloads, port, api)
	if err != nil {
		k.cannot(err.Error())
	}
	if msg := rqbitArgsRefusal(flags); msg != "" {
		k.refuse("the command line would start the build with " + msg)
	}
	_ = writeText(filepath.Join(workdir, "command-line.txt"), strings.Join(append([]string{exe}, flags...), " ")+"\n")
	c := exec.Command(exe, flags...)
	c.Env = rqbitChildEnv(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+home+"/.config",
		"XDG_DATA_HOME="+home+"/.local/share", "XDG_CACHE_HOME="+home+"/.cache")
	pid, err := startBackground(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "client.out"), outAppend: true, err: "&1"})
	if err != nil {
		k.cannot(rqbitTarget + " could not be started: " + err.Error())
	}
	ps := fmt.Sprint(pid)
	_ = writeText(filepath.Join(workdir, "pid"), ps+"\n")
	gone := func(d time.Duration) bool {
		time.Sleep(d)
		return !alive(ps)
	}
	library, err := rqbitDrive(workdir, api, torrent, gone)
	if err != nil {
		k.refuse(err.Error())
	}
	// ⭐ THE SOCKETS THE BUILD HOLDS, read rather than assumed, as the contract
	// asks: a surface this list does not name shows up here whether or not it
	// was suspected.
	if _, ok := commandV("ss"); ok {
		out, _ := capture(exec.Command("ss", "-tuanp"), stdio{in: os.DevNull, err: os.DevNull})
		var mine []string
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "pid="+ps+",") {
				mine = append(mine, l)
			}
		}
		_ = writeText(filepath.Join(workdir, "sockets.txt"), strings.Join(mine, "\n")+"\n")
	}
	fmt.Printf("started %s pid %s on port %s, library %s, API 127.0.0.1:%s\n", rqbitTarget, ps, port, library, api)
}

// rqbitStop ends the build and scrubs what it persisted.
//
// ⭐ SIGTERM IS THE BUILD'S OWN GRACEFUL EXIT: its handler cancels the session
// and exits within five seconds, then KILL after eight. ⛔ The home is listed
// into `state.txt` and removed on every path out, before `capture-client`
// packs the workdir.
func rqbitStop(k *adapterKit, workdir string) {
	home := filepath.Join(workdir, "home")
	defer func() {
		if listing := rqbitStateListing(home); listing != "" {
			appendText(filepath.Join(workdir, "state.txt"), utcNow()+" the build left:\n"+listing)
		}
		_ = os.RemoveAll(home)
	}()
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	how := "already exited"
	if signalPID(pid, syscall.SIGTERM) {
		how = "sigterm"
		if !waitGone(pid, 8) {
			signalPID(pid, syscall.SIGKILL)
			waitGone(pid, 5)
			how = "killed after a SIGTERM it did not finish"
		}
	}
	appendText(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %s: %s\n", utcNow(), pid, how))
}
