//go:build windows

// adapter_rqbit_windows - the ENGINE-03 capture adapter, Windows half.
//
// The contract is scripts/capture/adapters/README.md and the Linux half is
// adapter_rqbit.go; this file is the same five subcommands for the vendor's
// Windows console build, run as `bit-check adapter rqbit <subcommand>`
// because a Windows runner has no `sh`. What it decides without Windows in it
// is in rqbit.go, which any host can ask through `bit-check rqbit --selftest`.
//
// ⚠ THE MECHANICS ARE qBittorrent's WINDOWS HALF'S, REUSED AND NOT COPIED:
// qbtWinRun, qbtWinWaiter and the exit type. They carry no decision of that
// product.
//
// -- ⭐ WHAT IS THE SAME AS LINUX ------------------------------------------------
//
// `version` is `rqbit.exe --version` and nothing else. `start` passes the
// command line rqbitServerArgs builds, asked of rqbitArgsRefusal first, with
// every `RQBIT_*` dropped from the environment, and hands the torrent over the
// build's HTTP API on loopback through rqbitDrive, which both halves share. The
// API port is 3030 unless BIT_IDS_RQBIT_API_PORT says otherwise.
//
// -- ⛔ WHAT DIFFERS FROM LINUX, AND WHY EACH IS A FACT OF THE BUILD ------------
//
//   - `stop` KILLS. The build's signal handler is compiled for every platform
//     but Windows (`#[cfg(not(target_os = "windows"))]` in its `main.rs` at
//     9.0.1), and its API has no shutdown call, so there is no graceful exit to
//     ask for. ⚠ The build sends no `stopped` announce on either platform: its
//     tracker client never constructs that event.
//   - Persisted state is looked for where the build's `directories` crate puts
//     it on Windows, `%APPDATA%\rqbit` and `%LOCALAPPDATA%\rqbit`, because that
//     crate asks the shell for the known folders and not the environment.
//     `start` refuses a host that already has either; `stop` lists and removes
//     them.
//   - No PATH lookup and no `helpers=`: the API is this binary's own HTTP
//     client, and the build is found in the prefix alone.
//
// -- the routes -----------------------------------------------------------------
//
// ⛔ `release`: the vendor's `rqbit.exe`, placed by `install-rootless
// --install` at `<prefix>\bin\rqbit.exe` under the listing's digest.
// ⛔ `package`: crates.io's `rqbit`, `cargo install --locked` with no version
// and no feature flag into the same prefix, with the pinned Rust and MSVC.
// ⚠ winget, Scoop's `main` and `extras` buckets and Chocolatey carry no
// rqbit, measured on 2026-10-21, so crates.io is the one index on Windows too.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter(rqbitTarget, rqbitWinAdapter) }

func rqbitWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = rqbitWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = rqbitWinInstall(args[0], args[1])
		}
	case "version":
		err = rqbitWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = rqbitWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = rqbitWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	if e, ok := err.(qbtWinExit); ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", rqbitTarget, e.msg)
		return e.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", rqbitTarget, err)
	return 2
}

// rqbitWinExe is where both routes place the build. ⛔ THE PREFIX IS ASKED FOR,
// NEVER COMPOSED.
func rqbitWinExe() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "bin", rqbitWindowsAsset), nil
}

func rqbitWinBinary() string {
	if v := os.Getenv("BIT_IDS_RQBIT"); v != "" {
		return v
	}
	if exe, err := rqbitWinExe(); err == nil && isRegular(exe) {
		return exe
	}
	return ""
}

func rqbitWinDescribe() error {
	var b strings.Builder
	fmt.Fprintf(&b, "target=%s\n", rqbitTarget)
	b.WriteString("kind=stock\n")
	if exe := rqbitWinBinary(); exe != "" {
		fmt.Fprintf(&b, "binary=%s\n", exe)
	}
	b.WriteString(rqbitDescribe(rqbitWindowsAsset))
	fmt.Print(b.String())
	return nil
}

func rqbitWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := rqbitPackageFormat(true, route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	log := filepath.Join(workdir, "install.log")
	var err error
	switch route {
	case "release":
		err = rqbitWinRelease(workdir, log)
	case "package":
		err = rqbitWinCargo(workdir, log)
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

func rqbitWinRelease(workdir, log string) error {
	url := os.Getenv("BIT_IDS_RELEASE_URL")
	if url == "" {
		return qbtCannot("the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut")
	}
	res := os.Getenv("BIT_IDS_RELEASE_RESOLUTION")
	disposition := []string{"--digest-unpublished", rqbitDigestUnpublished}
	if res != "" {
		disposition = []string{"--from-resolution", res}
	}
	self, err := os.Executable()
	if err != nil {
		return qbtCannot("cannot find this binary to run install-rootless with")
	}
	args := append([]string{"install-rootless", "--install", "--url", url,
		"--into", filepath.Join(workdir, rqbitWindowsAsset), "--as", rqbitWindowsAsset}, disposition...)
	args = append(args, "--report", filepath.Join(workdir, "rootless-install.txt"))
	switch rc := qbtWinRun(log, self, args...); rc {
	case 0:
	case 1:
		return qbtRefuse("the release route could not install the vendor's build")
	default:
		return qbtCannot("the rootless installer could not run for the release route (exit %d)", rc)
	}
	if exe, err := rqbitWinExe(); err != nil || !isRegular(exe) {
		return qbtRefuse("the release route reported an install and left no %s in the prefix", rqbitWindowsAsset)
	}
	return nil
}

// rqbitWinCargo is the package route, as the Linux half's: cargo's state is
// the workdir's, the toolchain is the pin, and cargo's own record says what
// it built. ⚠ No bound of its own: `install-client` bounds the whole call and
// `install-step`'s job object holds every process it starts.
func rqbitWinCargo(workdir, log string) error {
	cargo, err := exec.LookPath("cargo")
	if err != nil {
		return qbtCannot("the package route compiles and cargo is not on this host")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return qbtCannot("the package route builds librqbit's web UI with npm and it is not on this host")
	}
	exe, err := rqbitWinExe()
	if err != nil {
		return qbtCannot("the rootless prefix could not be derived: %v", err)
	}
	prefix := filepath.Dir(filepath.Dir(exe))
	cargoHome := filepath.Join(workdir, "cargo-home")
	target := filepath.Join(workdir, "target")
	if err := os.MkdirAll(cargoHome, 0o755); err != nil {
		return qbtCannot("cannot create %s", cargoHome)
	}
	for _, kv := range rqbitCargoEnv(cargoHome, target) {
		name, value, _ := strings.Cut(kv, "=")
		os.Setenv(name, value)
	}
	rustc, _ := exec.Command("rustc", "-vV").Output()
	started := time.Now()
	if rc := qbtWinRun(log, cargo, rqbitCargoArgs(prefix)...); rc != 0 {
		return qbtRefuse("cargo install %s exited %d; install.log says why", rqbitCrate, rc)
	}
	took := int(time.Since(started) / time.Second)
	body, err := os.ReadFile(filepath.Join(prefix, ".crates2.json"))
	if err != nil {
		return qbtRefuse("cargo installed and wrote no .crates2.json in %s", prefix)
	}
	_ = os.WriteFile(filepath.Join(workdir, "crates2-json.txt"), body, 0o644)
	in, err := rqbitCargoRecord(body)
	if err != nil {
		return qbtRefuse("%v", err)
	}
	if !isRegular(exe) {
		return qbtRefuse("cargo reported an install and left no %s", exe)
	}
	matches, _ := filepath.Glob(filepath.Join(cargoHome, "registry", "cache", "*", rqbitCrate+"-"+in.version+".crate"))
	crateSum := ""
	if len(matches) == 1 {
		crateSum, _ = fileSHA256(matches[0])
	}
	if crateSum == "" {
		return qbtRefuse("cargo's registry cache holds no single %s-%s.crate", rqbitCrate, in.version)
	}
	note := rqbitFeaturesNote(in, crateSum)
	note += fmt.Sprintf("seconds=%d\nhost RUSTFLAGS=%s\nhost CARGO_ENCODED_RUSTFLAGS=%s\nrustc -vV:\n%s\n",
		took, os.Getenv("RUSTFLAGS"), os.Getenv("CARGO_ENCODED_RUSTFLAGS"), strings.TrimRight(strings.ReplaceAll(string(rustc), "\r", ""), "\n"))
	if err := os.WriteFile(filepath.Join(workdir, "build-features.txt"), []byte(note), 0o644); err != nil {
		return qbtRefuse("the route could not record the build's feature selection")
	}
	origin := "https://static.crates.io/crates/" + rqbitCrate + "/" + rqbitCrate + "-" + in.version + ".crate"
	if err := writePackageIndexVia(workdir, rqbitCratesIndex, rqbitCrate, in.version, origin, rqbitDelivery); err != nil {
		return qbtRefuse("the route could not record what the index named")
	}
	_ = os.RemoveAll(target)
	_ = os.RemoveAll(filepath.Join(cargoHome, "registry", "src"))
	return nil
}

// rqbitWinAsk runs `<exe> --version` with stdin empty and every `RQBIT_*`
// dropped, answering what it printed.
func rqbitWinAsk(exe string) (string, error) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		return "", err
	}
	defer null.Close()
	cmd := exec.Command(exe, "--version")
	cmd.Env = rqbitChildEnv(os.Environ())
	cmd.Stdin = null
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s --version exited %d", rqbitWindowsAsset, exitOf(err))
	}
	return string(out), nil
}

// rqbitWinVersion asks the installed build what it is. ⛔ THE BUILD SPEAKING, a
// pure read, as on Linux.
func rqbitWinVersion() error {
	exe := rqbitWinBinary()
	if exe == "" {
		return qbtCannot("rqbit is not installed")
	}
	answer, err := rqbitWinAsk(exe)
	if err != nil {
		return qbtCannot("%v", err)
	}
	version, err := rqbitParseVersion(answer)
	if err != nil {
		return qbtCannot("%v", err)
	}
	fmt.Fprintf(os.Stderr, "%s reported:\n%s\n", rqbitTarget, strings.TrimRight(strings.ReplaceAll(answer, "\r", ""), "\n"))
	fmt.Println(version)
	return nil
}

// rqbitWinStateDirs is where the build's `directories` crate would persist.
func rqbitWinStateDirs() []string {
	var out []string
	if d, err := os.UserConfigDir(); err == nil && d != "" {
		out = append(out, filepath.Join(d, "rqbit"))
	}
	if d, err := os.UserCacheDir(); err == nil && d != "" {
		out = append(out, filepath.Join(d, "rqbit"))
	}
	return out
}

func rqbitWinStart(torrent, workdir, peerPort string) error {
	exe := rqbitWinBinary()
	if exe == "" {
		return qbtCannot("rqbit is not installed")
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	for _, d := range rqbitWinStateDirs() {
		if _, err := os.Stat(d); err == nil {
			return qbtRefuse("%s already exists, so state this run did not write would reach the build", d)
		}
	}
	downloads := filepath.Join(workdir, "downloads")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return qbtCannot("cannot create %s", downloads)
	}
	api := rqbitAPIPortFor()
	flags, err := rqbitServerArgs(downloads, peerPort, api)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if msg := rqbitArgsRefusal(flags); msg != "" {
		return qbtRefuse("the command line would start the build with %s", msg)
	}
	_ = os.WriteFile(filepath.Join(workdir, "command-line.txt"), []byte(strings.Join(append([]string{exe}, flags...), " ")+"\n"), 0o644)
	out, err := os.OpenFile(filepath.Join(workdir, "client.out"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return qbtCannot("cannot write client.out")
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		out.Close()
		return qbtCannot("cannot open %s", os.DevNull)
	}
	cmd := exec.Command(exe, flags...)
	cmd.Env = rqbitChildEnv(os.Environ())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, out, out
	// ⚠ Its own process group, so a console event aimed at the caller's group
	// does not reach a build that has to outlive the call that started it.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	err = cmd.Start()
	out.Close()
	null.Close()
	if err != nil {
		return qbtCannot("rqbit could not be started: %v", err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the pid of rqbit")
	}
	library, err := rqbitDrive(workdir, api, torrent, qbtWinWaiter(cmd.Process))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	// ⚠ NOT WAITED ON: the build keeps running after this call ends.
	fmt.Printf("started rqbit pid %d on port %s, library %s, API 127.0.0.1:%s\n", pid, peerPort, library, api)
	return nil
}

func rqbitWinStop(workdir string) error {
	// ⛔ PERSISTED STATE LEAVES THE HOST ON EVERY PATH OUT, listed first.
	defer func() {
		for _, d := range rqbitWinStateDirs() {
			if listing := rqbitStateListing(d); listing != "" {
				appendFile(filepath.Join(workdir, "state.txt"), utcNow()+" the build left under "+d+":\n"+listing)
			}
			_ = os.RemoveAll(d)
		}
	}()
	pid, err := strconv.Atoi(strings.TrimSpace(readTrimmed(filepath.Join(workdir, "pid"))))
	// ⛔ A pid at or below 1 is never acted on, as signalPID refuses it on Linux.
	if err != nil || pid <= 1 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: already exited\n", utcNow(), pid))
		return nil
	}
	exited := qbtWinWaiter(p)
	how := "already exited"
	if !exited(200 * time.Millisecond) {
		// ⛔ KILLED, because this build has no graceful exit on Windows; see the
		// header.
		how = "killed: the Windows build installs no shutdown handler"
		p.Kill()
		exited(5 * time.Second)
	}
	appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: %s\n", utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", rqbitWindowsAsset, pid, how)
	return nil
}
