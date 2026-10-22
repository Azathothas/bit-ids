//go:build !windows

// adapter_anacrolix-torrent - the ENGINE-02 capture adapter on Linux,
// `bit-check adapter anacrolix-torrent`.
//
// The contract is scripts/capture/adapters/README.md. ⚠ The Windows half
// registers the same name from adapter_anacrolix-torrent_windows.go; one
// platform builds one of them. What either half decides without a process in
// it, and the install both run, is anacrolix.go, which `bit-check
// anacrolix-torrent --selftest` asks on any host.
//
// ⛔ THE PRODUCT IS THE HARNESS IN tools/harness/anacrolix-torrent/, built by
// each route from the module version the harness pins; see anacrolix.go for
// the two routes.
//
// Exit codes: 0 done, 1 refused, 2 could not run.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func init() {
	registerAdapter(anacrolixTarget, shAdapter(anacrolixTarget, anacrolixAdapter))
}

// anacrolixBinary is the installed harness, or "": an override, then the
// prefix, and ⛔ NEVER PATH.
func anacrolixBinary(k *adapterKit) string {
	if v := os.Getenv("BIT_IDS_ANACROLIX_TORRENT"); v != "" {
		return v
	}
	p, ok := k.prefix(true)
	if !ok {
		return ""
	}
	if exe := filepath.Join(p, "bin", anacrolixExe(false)); isExec(exe) {
		return exe
	}
	return ""
}

// anacrolixExitOf answers an anacrolix.go error the way this kit ends a call.
func anacrolixExitOf(k *adapterKit, err error) {
	if err == nil {
		return
	}
	var e anacrolixExit
	if errors.As(err, &e) && e.code == 1 {
		k.refuse(e.msg)
	}
	k.cannot(err.Error())
}

func anacrolixAdapter(k *adapterKit, sub string, args []string) {
	switch sub {
	case "describe":
		fmt.Printf("target=%s\n", anacrolixTarget)
		fmt.Print("kind=stock\n")
		if exe := anacrolixBinary(k); exe != "" {
			fmt.Printf("binary=%s\n", exe)
		}
		fmt.Print(anacrolixDescribe())

	case "install":
		route, workdir, format := k.installArgs(args, func(r string) (string, bool) { return anacrolixPackageFormat(false, r) })
		prefix, ok := k.prefix(false)
		if !ok {
			k.cannot("the rootless prefix could not be derived")
		}
		anacrolixExitOf(k, anacrolixInstall(route, workdir, filepath.Join(prefix, "bin"), false))
		k.writePackage(workdir, format)

	case "version":
		exe := anacrolixBinary(k)
		if exe == "" {
			k.cannot(anacrolixTarget + " is not installed")
		}
		version, answer, err := anacrolixAsk(exe)
		if err != nil {
			k.cannot(err.Error())
		}
		k.versionReport(answer, version)

	case "start":
		anacrolixStart(k, args)

	case "stop":
		if len(args) < 1 {
			k.cannot("stop takes <workdir>")
		}
		anacrolixStop(k, args[0])

	default:
		k.cannot("unknown subcommand: " + sub)
	}
}

func anacrolixStart(k *adapterKit, args []string) {
	if len(args) < 3 {
		k.cannot("start takes <torrent> <workdir> <peer-port>")
	}
	torrent, workdir, port := args[0], args[1], args[2]
	exe := anacrolixBinary(k)
	if exe == "" {
		k.cannot(anacrolixTarget + " is not installed")
	}
	if !isNonEmpty(torrent) {
		k.cannot("there is no torrent at " + torrent)
	}
	// ⛔ A HOME AND A DATA DIRECTORY OF ITS OWN PER SESSION, so whatever a
	// session writes is found by `stop` and nothing reaches the next one.
	home := filepath.Join(workdir, "home")
	downloads := filepath.Join(workdir, "downloads")
	for _, f := range []string{home, downloads, filepath.Join(workdir, "ready.txt"), filepath.Join(workdir, "stop-request")} {
		_ = os.RemoveAll(f)
	}
	if !mkdirP(home, downloads) {
		k.cannot("cannot create the harness's directories under " + workdir)
	}
	flags, err := anacrolixStartArgs(torrent, workdir, port)
	if err != nil {
		k.cannot(err.Error())
	}
	if msg := anacrolixArgsRefusal(flags); msg != "" {
		k.refuse("the command line would start the harness with " + msg)
	}
	_ = writeText(filepath.Join(workdir, "command-line.txt"), strings.Join(append([]string{exe}, flags...), " ")+"\n")
	c := exec.Command(exe, flags...)
	c.Env = anacrolixChildEnv(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+home+"/.config",
		"XDG_DATA_HOME="+home+"/.local/share", "XDG_CACHE_HOME="+home+"/.cache")
	pid, err := startBackground(c, stdio{in: os.DevNull, out: filepath.Join(workdir, "client.out"), outAppend: true, err: "&1"})
	if err != nil {
		k.cannot(anacrolixTarget + " could not be started: " + err.Error())
	}
	ps := fmt.Sprint(pid)
	_ = writeText(filepath.Join(workdir, "pid"), ps+"\n")
	listen, err := anacrolixDrive(workdir, func(d time.Duration) bool {
		time.Sleep(d)
		return !alive(ps)
	})
	if err != nil {
		k.refuse(err.Error())
	}
	// ⭐ THE SOCKETS THE HARNESS HOLDS, read rather than assumed.
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
	fmt.Printf("started %s pid %s on %s\n", anacrolixTarget, ps, listen)
}

// anacrolixStop asks the library to close and scrubs what the session left.
//
// ⭐ THE CLOSE IS THE LIBRARY'S OWN: the stop file, which the harness polls,
// and `Client.Close` after it. KILL after anacrolixGraceSeconds. ⛔ The home
// and the data directory are listed into `state.txt` and removed on every path
// out, before `capture-client` packs the workdir.
func anacrolixStop(k *adapterKit, workdir string) {
	home := filepath.Join(workdir, "home")
	downloads := filepath.Join(workdir, "downloads")
	defer func() {
		for _, d := range []string{home, downloads} {
			if listing := anacrolixStateListing(d); listing != "" {
				appendText(filepath.Join(workdir, "state.txt"), utcNow()+" the session left under "+filepath.Base(d)+":\n"+listing)
			}
			_ = os.RemoveAll(d)
		}
		_ = os.Remove(filepath.Join(workdir, "stop-request"))
	}()
	pid := readPID(filepath.Join(workdir, "pid"))
	if pid == "" {
		return
	}
	how := "already exited"
	if alive(pid) {
		_ = writeText(filepath.Join(workdir, "stop-request"), utcNow()+"\n")
		how = "closed on the stop file"
		if !waitGone(pid, anacrolixGraceSeconds) {
			signalPID(pid, syscall.SIGKILL)
			waitGone(pid, 5)
			how = fmt.Sprintf("killed after the stop file went unanswered for %ds", anacrolixGraceSeconds)
		}
	}
	appendText(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %s: %s\n", utcNow(), pid, how))
}
