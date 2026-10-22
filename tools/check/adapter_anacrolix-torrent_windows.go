//go:build windows

// adapter_anacrolix-torrent_windows - the ENGINE-02 capture adapter, Windows
// half.
//
// The contract is scripts/capture/adapters/README.md and the Linux half is
// adapter_anacrolix-torrent.go; this file is the same five subcommands, run
// as `bit-check adapter anacrolix-torrent <subcommand>` because a Windows
// runner has no `sh`. ⭐ The install is anacrolix.go's, the same function the
// Linux half runs: the two routes are the go command's, and the go command is
// the same program on both platforms.
//
// ⚠ THE MECHANICS ARE qBittorrent's WINDOWS HALF'S, REUSED AND NOT COPIED:
// qbtWinWaiter and the exit type.
//
// -- ⛔ WHAT DIFFERS FROM LINUX ----------------------------------------------------
//
//   - The executable is `anacrolix-torrent-harness.exe`, a PE.
//   - `stop` writes the same stop file and the harness runs the same
//     `Client.Close`; a harness still running after the grace is killed, as
//     on Linux. There is no SIGTERM to send on Windows, which is why the
//     harness polls a file at all.
//   - No socket table: Windows has no `ss`. No `helpers=`: the capture runs
//     the harness alone.
//
// Exit codes: the contract's, 0 done, 1 refused, 2 could not run.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() { registerAdapter(anacrolixTarget, anacrolixWinAdapter) }

func anacrolixWinAdapter(sub string, args []string) int {
	var err error
	switch sub {
	case "describe":
		err = anacrolixWinDescribe()
	case "install":
		if len(args) < 2 {
			err = qbtCannot("install takes <route> <workdir>")
		} else {
			err = anacrolixWinInstall(args[0], args[1])
		}
	case "version":
		err = anacrolixWinVersion()
	case "start":
		if len(args) < 3 {
			err = qbtCannot("start takes <torrent> <workdir> <peer-port>")
		} else {
			err = anacrolixWinStart(args[0], args[1], args[2])
		}
	case "stop":
		if len(args) < 1 {
			err = qbtCannot("stop takes <workdir>")
		} else {
			err = anacrolixWinStop(args[0])
		}
	default:
		err = qbtCannot("unknown subcommand: %s", sub)
	}
	if err == nil {
		return 0
	}
	var q qbtWinExit
	if errors.As(err, &q) {
		fmt.Fprintf(os.Stderr, "%s: %s\n", anacrolixTarget, q.msg)
		return q.code
	}
	var a anacrolixExit
	if errors.As(err, &a) {
		fmt.Fprintf(os.Stderr, "%s: %s\n", anacrolixTarget, a.msg)
		return a.code
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", anacrolixTarget, err)
	return 2
}

// anacrolixWinBin is the prefix's `bin`. ⛔ THE PREFIX IS ASKED FOR, NEVER
// COMPOSED.
func anacrolixWinBin() (string, error) {
	prefix, err := rootlessPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "bin"), nil
}

func anacrolixWinBinary() string {
	if v := os.Getenv("BIT_IDS_ANACROLIX_TORRENT"); v != "" {
		return v
	}
	if bin, err := anacrolixWinBin(); err == nil && isRegular(filepath.Join(bin, anacrolixExe(true))) {
		return filepath.Join(bin, anacrolixExe(true))
	}
	return ""
}

func anacrolixWinDescribe() error {
	var b strings.Builder
	fmt.Fprintf(&b, "target=%s\n", anacrolixTarget)
	b.WriteString("kind=stock\n")
	if exe := anacrolixWinBinary(); exe != "" {
		fmt.Fprintf(&b, "binary=%s\n", exe)
	}
	b.WriteString(anacrolixDescribe())
	fmt.Print(b.String())
	return nil
}

func anacrolixWinInstall(route, workdir string) error {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return qbtCannot("cannot create %s", workdir)
	}
	format, ok := anacrolixPackageFormat(true, route)
	if !ok {
		return qbtCannot("this adapter does not say how the %s route packages a build", route)
	}
	bin, err := anacrolixWinBin()
	if err != nil {
		return qbtCannot("the rootless prefix could not be derived: %v", err)
	}
	if err := anacrolixInstall(route, workdir, bin, true); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workdir, "package"), []byte(format+"\n"), 0o644); err != nil {
		return qbtRefuse("the route installed and could not record how it packaged the build")
	}
	return nil
}

// anacrolixWinVersion asks the installed harness, a pure read, as on Linux.
func anacrolixWinVersion() error {
	exe := anacrolixWinBinary()
	if exe == "" {
		return qbtCannot("%s is not installed", anacrolixTarget)
	}
	version, answer, err := anacrolixAsk(exe)
	if err != nil {
		return qbtCannot("%v", err)
	}
	fmt.Fprintf(os.Stderr, "%s reported:\n%s\n", anacrolixTarget, answer)
	fmt.Println(version)
	return nil
}

func anacrolixWinStart(torrent, workdir, peerPort string) error {
	exe := anacrolixWinBinary()
	if exe == "" {
		return qbtCannot("%s is not installed", anacrolixTarget)
	}
	if fi, err := os.Stat(torrent); err != nil || fi.Size() == 0 {
		return qbtCannot("there is no torrent at %s", torrent)
	}
	downloads := filepath.Join(workdir, "downloads")
	for _, f := range []string{downloads, filepath.Join(workdir, "ready.txt"), filepath.Join(workdir, "stop-request")} {
		_ = os.RemoveAll(f)
	}
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return qbtCannot("cannot create %s", downloads)
	}
	flags, err := anacrolixStartArgs(torrent, workdir, peerPort)
	if err != nil {
		return qbtCannot("%v", err)
	}
	if msg := anacrolixArgsRefusal(flags); msg != "" {
		return qbtRefuse("the command line would start the harness with %s", msg)
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
	cmd.Env = anacrolixChildEnv(os.Environ())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, out, out
	// ⚠ Its own process group, so a console event aimed at the caller's group
	// does not reach a harness that has to outlive the call that started it.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	err = cmd.Start()
	out.Close()
	null.Close()
	if err != nil {
		return qbtCannot("%s could not be started: %v", anacrolixTarget, err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(filepath.Join(workdir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return qbtCannot("cannot record the pid of the harness")
	}
	listen, err := anacrolixDrive(workdir, qbtWinWaiter(cmd.Process))
	if err != nil {
		return qbtRefuse("%v", err)
	}
	// ⚠ NOT WAITED ON: the harness keeps running after this call ends.
	fmt.Printf("started %s pid %d on %s\n", anacrolixTarget, pid, listen)
	return nil
}

func anacrolixWinStop(workdir string) error {
	downloads := filepath.Join(workdir, "downloads")
	// ⛔ WHAT THE SESSION LEFT LEAVES THE HOST ON EVERY PATH OUT, listed first.
	defer func() {
		if listing := anacrolixStateListing(downloads); listing != "" {
			appendFile(filepath.Join(workdir, "state.txt"), utcNow()+" the session left under downloads:\n"+listing)
		}
		_ = os.RemoveAll(downloads)
		_ = os.Remove(filepath.Join(workdir, "stop-request"))
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
		_ = os.WriteFile(filepath.Join(workdir, "stop-request"), []byte(utcNow()+"\n"), 0o644)
		how = "closed on the stop file"
		if !exited(anacrolixGraceSeconds * time.Second) {
			how = fmt.Sprintf("killed after the stop file went unanswered for %ds", anacrolixGraceSeconds)
			p.Kill()
			exited(5 * time.Second)
		}
	}
	appendFile(filepath.Join(workdir, "stop.txt"), fmt.Sprintf("%s pid %d: %s\n", utcNow(), pid, how))
	fmt.Printf("stopped %s pid %d: %s\n", anacrolixExe(true), pid, how)
	return nil
}
