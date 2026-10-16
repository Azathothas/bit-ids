//go:build !linux

package main

import (
	"fmt"
	"os"
)

// contained is Linux-only: a network namespace is a Linux kernel object, so
// anywhere else the namespace cannot be made and that is exit 2. Windows
// contains a capture at the firewall instead (`CI-08`); contained_linux.go
// carries the reasons.
func contained([]string) int {
	fmt.Fprintln(os.Stderr, "contained: a network namespace is Linux-only, so this host cannot make one")
	return 2
}
