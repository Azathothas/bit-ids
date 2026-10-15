//go:build windows

// The Windows answer to `uname -srm`, for the platform line every record
// carries.
//
// ⛔ NEVER `uname` ON WINDOWS. A Windows runner has Git for Windows on PATH,
// whose MSYS `uname -srm` answers `MSYS_NT-10.0-26100 3.6.10-... x86_64`: the
// emulation layer's name, not the system's. `assemble-capture` derives the
// platform slug from its first word, and refused capture-client run
// 38788416994's Windows lanes for it. `CI-11`.
//
// ⭐ The version is the kernel's own, from RtlGetVersion, which unlike
// GetVersionEx is not shimmed to an older answer for an unmanifested program.

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	procRtlGetVersion  = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")
	procGetTickCount64 = syscall.NewLazyDLL("kernel32.dll").NewProc("GetTickCount64")
)

type osVersionInfo struct {
	size, major, minor, build, platform uint32
	csd                                 [128]uint16
}

// windowsPlatform is `Windows <major>.<minor>.<build> <machine>`, three fields
// in the shape `uname -srm` gives on Linux, or "" when the kernel will not say.
func windowsPlatform() string {
	var v osVersionInfo
	v.size = uint32(unsafe.Sizeof(v))
	if r, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&v))); r != 0 {
		return ""
	}
	return fmt.Sprintf("Windows %d.%d.%d %s", v.major, v.minor, v.build, unameMachine(runtime.GOARCH))
}

// hostUptime is seconds since boot in /proc/uptime's spelling, two decimals,
// from GetTickCount64: the run manifest's second clock. `CI-11`.
func hostUptime() string {
	ms, _, _ := procGetTickCount64.Call()
	return fmt.Sprintf("%d.%02d", ms/1000, (ms%1000)/10)
}
