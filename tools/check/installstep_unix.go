//go:build !windows

// The Unix half of install-step: the install's container is a SESSION.
//
// ⭐ A process group cannot leave its session, so every group the install
// starts, nested bounds included, is still in the session this opens, and the
// session is the whole population a sweep has to reach. Its id is the child's
// own pid, because the child is the session leader. It cannot reach this
// process's own session: that id belongs to an ancestor of this process.

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type container struct{ sid int }

func startContained(cmd *exec.Cmd) (*container, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &container{sid: cmd.Process.Pid}, nil
}

// terminate is what `timeout` sends its child at the bound. ⚠ The KILL after
// the grace is the sweep's: this process does not stay to send it.
func (c *container) terminate(int) { syscall.Kill(c.sid, syscall.SIGTERM) }

// sessionMembers lists the processes in the container's session, as `ps`
// printed them, one line each.
func (c *container) sessionMembers(seconds int) ([]string, []int) {
	out, ok := boundedOutput(seconds, "ps", "-eo", "pid=,sid=,args=")
	if !ok {
		return nil, nil
	}
	var lines []string
	var pids []int
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[1] != strconv.Itoa(c.sid) {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		lines = append(lines, line)
		pids = append(pids, pid)
	}
	return lines, pids
}

// sweep ends what the install left in its session: TERM, two seconds, then
// KILL to a FRESH listing, so a pid that exited and was handed to something
// else in those two seconds is not the one that gets KILL.
func (c *container) sweep(seconds int) (int, string) {
	lines, pids := c.sessionMembers(seconds)
	if len(pids) == 0 {
		return 0, ""
	}
	for _, p := range pids {
		syscall.Kill(p, syscall.SIGTERM)
	}
	time.Sleep(2 * time.Second)
	_, again := c.sessionMembers(seconds)
	for _, p := range again {
		syscall.Kill(p, syscall.SIGKILL)
	}
	return len(pids), strings.Join(lines, "\n") + "\n"
}

func processTable(seconds int) (string, bool) {
	return boundedOutput(seconds, "ps", "-eo", "pid,ppid,pgid,stat,etimes,comm,args")
}

func signalCode(ee *exec.ExitError) int {
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return 1
}

// reportHolders runs scripts/ci/report-holders.sh over the step log and this
// process's own stdout, bounded at 60 seconds, appending to the holders log,
// and answers its exit code.
//
// ⛔ THIS PROCESS'S OWN STDOUT IS ASKED ABOUT TOO: the report used to be
// pointed at `step.log` alone and answered `0 holder(s)` on every green run,
// truthfully, and about the wrong descriptor. ⚠ `/proc/self/fd/1` is read
// here, in this process, because inside the reporter it would name the
// reporter's own output.
func reportHolders(adapter, stepLog, holdersLog string) int {
	root, err := captureRoot(filepath.Dir(adapter))
	if err != nil {
		appendFile(holdersLog, "## no tree to find report-holders.sh in\n")
		return 2
	}
	args := []string{filepath.Join(root, "scripts", "ci", "report-holders.sh"), stepLog}
	if out, err := os.Readlink("/proc/self/fd/1"); err == nil && out != "" {
		args = append(args, out)
	}
	f, err := os.OpenFile(holdersLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 2
	}
	defer f.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", args...)
	cmd.Stdout, cmd.Stderr = f, f
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 124
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return 2
	}
	return 0
}

// rootDevice is `stat -c '%d' /`, the device the root filesystem is on.
func rootDevice() (string, bool) {
	var st syscall.Stat_t
	if err := syscall.Stat("/", &st); err != nil {
		return "", false
	}
	return strconv.FormatUint(uint64(st.Dev), 10), true
}
