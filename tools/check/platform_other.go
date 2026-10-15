//go:build !windows

package main

import (
	"os"
	"strings"
)

// windowsPlatform is only ever asked on Windows; see platform_windows.go.
func windowsPlatform() string { return "" }

// hostUptime is the first field of /proc/uptime, or "" where there is none.
func hostUptime() string {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}
	up, _, _ := strings.Cut(strings.TrimSpace(string(b)), " ")
	return up
}
