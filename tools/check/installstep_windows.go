//go:build windows

// The Windows half of install-step: the install's container is a JOB OBJECT.
//
// ⭐ A process a job member starts is in the job too unless it asks to break
// away, so the job is the whole population a sweep has to reach, as the
// session is on Unix. ⚠ The child is assigned to the job right after it
// starts, and `bit-check install-client` reads its arguments and its adapter's
// description before it starts anything, so nothing it spawns can precede the
// assignment.
//
// ⚠ Called through kernel32 directly, because this module takes no
// dependencies and the standard library does not wrap job objects.

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObject       = kernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJob    = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject    = kernel32.NewProc("TerminateJobObject")
	procQueryInformationJob   = kernel32.NewProc("QueryInformationJobObject")
	procOpenProcess           = kernel32.NewProc("OpenProcess")
	jobObjectBasicProcessList = uintptr(3)
)

const (
	processSetQuota  = 0x0100
	processTerminate = 0x0001
)

type container struct {
	job uintptr
	pid int
	cmd *exec.Cmd
}

func startContained(cmd *exec.Cmd) (*container, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &container{pid: cmd.Process.Pid, cmd: cmd}
	job, _, _ := procCreateJobObject.Call(0, 0)
	if job == 0 {
		return c, nil
	}
	h, _, _ := procOpenProcess.Call(processSetQuota|processTerminate, 0, uintptr(c.pid))
	if h != 0 {
		if ok, _, _ := procAssignProcessToJob.Call(job, h); ok != 0 {
			c.job = job
		}
		syscall.CloseHandle(syscall.Handle(h))
	}
	return c, nil
}

// terminate ends the install itself; Windows has no TERM to send it.
func (c *container) terminate(int) { c.cmd.Process.Kill() }

// sweep lists the job's live processes and terminates the job.
func (c *container) sweep(int) (int, string) {
	if c.job == 0 {
		return 0, ""
	}
	// JOBOBJECT_BASIC_PROCESS_ID_LIST: two DWORDs, then ULONG_PTR ids.
	buf := make([]uintptr, 2+1024)
	ok, _, _ := procQueryInformationJob.Call(c.job, jobObjectBasicProcessList,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))*unsafe.Sizeof(buf[0]), 0)
	var lines []string
	if ok != 0 {
		// ⚠ The two DWORDs share the first word on 64-bit and take two on 32-bit.
		n, first := 0, 1
		if unsafe.Sizeof(buf[0]) == 8 {
			n = int(uint32(uint64(buf[0]) >> 32))
		} else {
			n, first = int(buf[1]), 2
		}
		for i := 0; i < n && first+i < len(buf); i++ {
			lines = append(lines, fmt.Sprintf("%d", buf[first+i]))
		}
	}
	procTerminateJobObject.Call(c.job, 1)
	if len(lines) == 0 {
		return 0, ""
	}
	return len(lines), strings.Join(lines, "\n") + "\n"
}

func processTable(seconds int) (string, bool) {
	return boundedOutput(seconds, "tasklist", "/v")
}

func signalCode(*exec.ExitError) int { return 1 }

// reportHolders has nothing to read here: Windows has no /proc, and the
// question it answers on Unix is answered by the job object having ended
// everything in it. ⚠ Said in the log rather than left empty.
func reportHolders(_, _, holdersLog string) int {
	appendFile(holdersLog, "## this host has no /proc, so holders are not readable; the job object ended its members\n")
	return 2
}

// rootDevice has no counterpart here; the Windows guard digests its own inputs.
func rootDevice() (string, bool) { return "", false }
