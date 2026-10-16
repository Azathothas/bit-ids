//go:build linux

// contained - run a command, as the calling user, in a network namespace of
// its own that holds nothing but loopback.
//
// CI-11. It was scripts/capture/contained.sh and is here whole: the same
// `sudo -n env`, the same `unshare --net`, the same loopback and the same
// `setpriv`, in the same order and with the same messages. The inner step is
// this same binary, as the shell half's was `sh "$0"`.
//
// ⛔ WHY IT EXISTS: THE CAPTURE HANG WAS THE CONTAINMENT. `capture-client` cut
// the HOST's default routes before a capture and restored them after. The host
// is also where the runner agent lives, and the agent reaches GitHub over those
// routes - so from the cut onward the job could not report, and on the runner
// images of 2026-09 onward it never came back: no log, no artifact, and a step
// shown as running that had in fact finished. Measured on 2026-10-11 by a flight
// recorder posting off the host: every operation of *Install the client* ran to
// exit 0, the posts stopped the moment the next step cut the route, and they
// never resumed. Thirteen bounds failed on that step because nothing in it was
// ever hung.
//
// ⭐ SO THE CAPTURE IS CONTAINED, NOT THE HOST. A new network namespace starts
// with one interface, `lo`, and no route at all. The observer and the client
// both run inside it, so they reach each other over loopback and nothing else;
// the runner agent stays in the host's namespace and never loses its link.
// `bit-check assert-disposable --egress` reads `/proc/net/route`, which inside
// the namespace is the namespace's own table, so the guard asks the right
// question of the process it guards.
//
// ⚠ CREATING A NAMESPACE NEEDS `CAP_SYS_ADMIN`, so this asks `sudo -n` for
// exactly that and drops straight back to the caller's uid, gid and groups with
// `setpriv` before the command runs. Nothing the command starts is root.
//
// ⚠ LINUX ONLY. A network namespace is a Linux kernel object; elsewhere this
// answers 2 (contained_other.go). Windows contains a capture at the firewall
// instead, through firewall-contain.ps1 (`CI-08`).
//
// Usage:
//
//	bit-check contained -- <command> [args...]
//
// Exit codes: the command's own, or 2 when the namespace could not be made.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func contained(args []string) int {
	say := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "contained: "+f+"\n", a...) }

	if len(args) > 0 && args[0] == "--inner" {
		// ⚠ Root, inside the new namespace, for two commands only.
		if len(args) < 5 || args[3] != "--" {
			say("--inner takes <uid> <gid> -- <command>")
			return 2
		}
		uid, gid, command := args[1], args[2], args[4:]
		lo := exec.Command("ip", containedLoopbackArgv()[1:]...)
		lo.Stdin, lo.Stdout, lo.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := lo.Run(); err != nil {
			say("could not bring loopback up in the namespace")
			return 2
		}
		return containedExec("setpriv", containedInnerArgv(uid, gid, command))
	}

	if len(args) < 2 || args[0] != "--" {
		say("usage: bit-check contained -- <command> [args...]")
		return 2
	}
	command := args[1:]

	for _, tool := range []string{"sudo", "unshare", "setpriv", "ip"} {
		if _, err := exec.LookPath(tool); err != nil {
			say("%s is not on this host", tool)
			return 2
		}
	}
	self, err := os.Executable()
	if err != nil {
		say("cannot name this executable: %v", err)
		return 2
	}

	// ⚠ `sudo -n`, so a host that would prompt says no rather than waiting, and
	// the environment is carried explicitly: `env_reset` would otherwise hand the
	// command root's PATH and HOME and a capture would write its state somewhere
	// the caller does not own. ⛔ BIT_CHECK is carried too, when set: the capture
	// path finds bit-check through it alone, and capture run 38763293262 refused
	// for want of it inside the namespace. `CI-11`.
	return containedExec("sudo", containedOuterArgv(self, os.Getenv("PATH"), os.Getenv("HOME"),
		os.Getenv("BIT_CHECK"), os.Geteuid(), os.Getegid(), command))
}

// containedOuterArgv is the shell half's
//
//	sudo -n env "PATH=$PATH" "HOME=${HOME:-}" ${BIT_CHECK:+"BIT_CHECK=$BIT_CHECK"} \
//	  unshare --net -- sh "$0" --inner "$(id -u)" "$(id -g)" -- "$@"
//
// with this binary in place of `sh "$0"`. ⚠ BIT_CHECK is carried only when it
// is set and not empty, as `${BIT_CHECK:+...}` carried it.
func containedOuterArgv(self, path, home, bitCheck string, uid, gid int, command []string) []string {
	argv := []string{"sudo", "-n", "env", "PATH=" + path, "HOME=" + home}
	if bitCheck != "" {
		argv = append(argv, "BIT_CHECK="+bitCheck)
	}
	argv = append(argv, "unshare", "--net", "--", self, "contained", "--inner",
		strconv.Itoa(uid), strconv.Itoa(gid), "--")
	return append(argv, command...)
}

// containedLoopbackArgv is the one command the namespace needs before the drop.
func containedLoopbackArgv() []string { return []string{"ip", "link", "set", "lo", "up"} }

// containedInnerArgv drops back to the caller: its uid, its gid, and the groups
// its account has, which is what `--init-groups` asks initgroups(3) for.
func containedInnerArgv(uid, gid string, command []string) []string {
	argv := []string{"setpriv", "--reuid=" + uid, "--regid=" + gid, "--init-groups", "--"}
	return append(argv, command...)
}

// containedExec replaces this process, as the shell half's `exec` did, so the
// command's exit code is this command's and no parent is left holding its
// streams. It returns only when the exec could not happen.
func containedExec(tool string, argv []string) int {
	path, err := exec.LookPath(tool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "contained: %s is not on this host\n", tool)
		return 2
	}
	err = syscall.Exec(path, argv, os.Environ())
	fmt.Fprintf(os.Stderr, "contained: could not run %s: %v\n", tool, err)
	return 2
}
