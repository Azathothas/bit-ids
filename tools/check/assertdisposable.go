// assert-disposable - refuse to install or run a client on a host somebody
// keeps.
//
// CI-11. The Unix half was scripts/acquisition/assert-disposable.sh and is
// here whole; the Windows half is assert-disposable.ps1 and is asked, not
// re-spelled, because its egress question is the firewall's, which
// firewall-contain.ps1 writes and only PowerShell's cmdlets read.
//
// ⛔ THIS RUNS BEFORE THE INSTALL, AND THAT IS THE WHOLE POINT. The run
// manifest refuses to RECORD a capture whose host was not disposable, per
// `E-MAN-30`, and that guard cannot stop one: by the time a manifest exists an
// untrusted installer has already run. A record-layer refusal is a report.
//
// -- ⭐ TWO INDEPENDENT GUARDS, AND WHY THEY ARE NOT ONE ---------------------
//
//	--claim    the host has never been used for a capture before
//	--egress   the host cannot reach anything but loopback
//
// A fresh host with an open route leaks the capture; a firewalled host that
// already ran a capture contaminates this one with the last one's state.
// `docs/AGENTS.md` section 5 requires both before any client is installed.
//
// ⛔ --claim DETECTS THE FAILURE RATHER THAN TRUSTING THE CLAIM. A token a
// provisioner wrote is a promise, and a runner misconfigured to persist its
// disk still carries it. So this claims the host by WRITING a marker, created
// exclusively, and refuses if one is there: a second capture on one host means
// the host survived the first. ⚠ The marker lives where a teardown destroys it
// and a reboot does not: `/run` and `/tmp` would report a rebooted host fresh.
// ⭐ It is the user's own state directory and not `/var/lib`, which needed
// `sudo` for every caller. `ACQ-06`. ⚠ Per-user is a narrower claim, recorded
// in `docs/capture-host.md`: the contract is one capture per host.
//
// ⛔ --egress READS THE KERNEL'S ROUTING TABLE AND SENDS NOTHING. A default
// route is a destination of all zeroes with the GATEWAY flag set, and the
// table is an argument so a test can hand it a file; `contained.sh` runs this
// inside the capture's namespace, where the table is loopback alone.
//
// ⚠ Override the state directory with BIT_IDS_STATE_DIR only to test this.
//
// Usage:
//
//	bit-check assert-disposable --claim <run-id>
//	bit-check assert-disposable --egress [routing-table]
//	bit-check assert-disposable --fingerprint
//	bit-check assert-disposable --marker
//
// Exit codes: 0 the guard passed, 1 the guard refuses, 2 could not run.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const assertDisposableUsage = `usage: bit-check assert-disposable --claim <run-id>
       bit-check assert-disposable --egress [routing-table]
       bit-check assert-disposable --fingerprint
       bit-check assert-disposable --marker

Exit codes: 0 the guard passed, 1 the guard refuses, 2 could not run.
`

func assertDisposable(args []string) int {
	if runtime.GOOS == "windows" {
		return assertDisposableWindows(args)
	}
	say := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "assert-disposable: "+f+"\n", a...) }
	stateDir, marker, err := claimMarker()
	if err != nil {
		say("%v", err)
		return 2
	}
	usage := func() int { fmt.Fprint(os.Stderr, assertDisposableUsage); return 2 }
	if len(args) == 0 {
		return usage()
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Print(assertDisposableUsage)
		return 0
	}

	switch args[0] {
	case "--fingerprint":
		if len(args) != 1 {
			return usage()
		}
		fmt.Println(hostFingerprint())
		return 0
	case "--marker":
		if len(args) != 1 {
			return usage()
		}
		fmt.Println(marker)
		return 0
	case "--egress":
		if len(args) > 2 {
			return usage()
		}
		table := "/proc/net/route"
		if len(args) == 2 {
			table = args[1]
		}
		via, named, err := publicRoutes(table)
		switch {
		case err != nil:
			say("%s is unreadable, so egress could not be", table)
			say("established. That is not a pass.")
			return 2
		case via:
			say("a public route exists; a capture host reaches loopback only")
			for _, iface := range named {
				say("default route via %s", iface)
			}
			return 1
		}
		fmt.Printf("no route off this host (read %s)\n", table)
		return 0
	case "--claim":
		if len(args) != 2 {
			return usage()
		}
		run := args[1]
		if run == "" || strings.Trim(run, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
			say("run id must be lowercase a-z0-9-: %s", run)
			return 2
		}
		if _, err := os.Stat(marker); err == nil {
			say("this host already ran a capture, so it was not destroyed")
			say("%s", firstLines(marker, 3))
			return 1
		}
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			say("cannot create %s", stateDir)
			return 2
		}
		print := hostFingerprint()
		if print == "" {
			say("the host fingerprint is empty")
			return 2
		}
		// ⛔ CREATED EXCLUSIVELY: two captures racing on one host must not both
		// believe they claimed it, which is `set -C` in the shell half.
		f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			say("another capture claimed this host first")
			return 1
		}
		image := os.Getenv("ImageOS")
		if v := os.Getenv("ImageVersion"); v != "" {
			image += "/" + v
		}
		uptime := ""
		if b, err := os.ReadFile("/proc/uptime"); err == nil {
			uptime, _, _ = strings.Cut(strings.TrimSpace(string(b)), " ")
		}
		// ⭐ THE HOST FACTS A RUN MANIFEST NEEDS, TAKEN AT THE CLAIM, each as
		// read and empty where the host does not say. `CI-09`.
		fmt.Fprintf(f, "bit-ids/host-claim/1\nrun=%s\nclaimed_at=%s\nfingerprint=%s\nimage=%s\nos=%s\nrunner_environment=%s\nuptime_at_claim=%s\n",
			run, utcNow(), print, image, prettyOS(), os.Getenv("RUNNER_ENVIRONMENT"), uptime)
		f.Close()
		fmt.Printf("%s (claimed %s)\n", print, marker)
		return 0
	}
	return usage()
}

// claimMarker is the ONE derivation of where the claim marker lives, which
// `--marker` prints and `install-client` reads in process. ⛔ A caller that
// composed the path itself would go on reading the old place the day the state
// directory moved, and report a host nobody claimed as fresh.
func claimMarker() (stateDir, marker string, err error) {
	stateDir = os.Getenv("BIT_IDS_STATE_DIR")
	if stateDir == "" {
		home := os.Getenv("HOME")
		if !strings.HasPrefix(home, "/") {
			return "", "", fmt.Errorf("HOME is [%s], so there is no state directory this user is known to own", home)
		}
		stateDir = filepath.Join(home, ".local", "state", "bit-ids")
	}
	return stateDir, stateDir + "/host-claimed", nil
}

// hostFingerprint is a value that differs between two hosts and survives
// within one. ⚠ Several inputs rather than one, because any single source is a
// coincidence away from colliding. The bytes are the shell half's exactly: each
// file as it is, then the host name, `uname -srm` and the root's device, each
// on its own line, digested together.
func hostFingerprint() string {
	var b strings.Builder
	for _, p := range []string{"/etc/machine-id", "/proc/sys/kernel/random/boot_id"} {
		if c, err := os.ReadFile(p); err == nil {
			b.Write(c)
		}
	}
	if h, err := os.Hostname(); err == nil {
		b.WriteString(h + "\n")
	}
	b.WriteString(unameSRM())
	if dev, ok := rootDevice(); ok {
		b.WriteString(dev + "\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func unameSRM() string {
	p := hostPlatform()
	if p == "" {
		return ""
	}
	return p + "\n"
}

// publicRoutes answers whether a default route exists, every interface a
// zero destination names (which is what the refusal prints), or an error when
// the table cannot be read. ⛔ A default route is destination 00000000 with
// the GATEWAY flag, whose low bit is set in the last hex digit of Flags, which
// is the shell half's reading exactly: a host route to 0.0.0.0 without a
// gateway is not one.
func publicRoutes(table string) (bool, []string, error) {
	f, err := os.Open(table)
	if err != nil {
		return false, nil, err
	}
	defer f.Close()
	found := false
	var named []string
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 || fields[1] != "00000000" {
			continue
		}
		named = append(named, fields[0])
		flags := fields[3]
		if flags != "" && strings.ContainsRune("13579bBdDfF", rune(flags[len(flags)-1])) {
			found = true
		}
	}
	if err := sc.Err(); err != nil {
		return false, nil, errors.New("unreadable")
	}
	return found, named, nil
}

// assertDisposableWindows asks assert-disposable.ps1, the Windows half, in
// its own spelling: `--claim <id>` is `-Claim -RunId <id>`, and `--egress`
// takes `--program <a|b>` and `--firewall-state <file>` for its `-Program`
// and `-FirewallState`.
func assertDisposableWindows(args []string) int {
	root, err := captureRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "assert-disposable: %v\n", err)
		return 2
	}
	ps := []string{"-NoProfile", "-File", filepath.Join(root, "scripts", "acquisition", "assert-disposable.ps1")}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--claim":
			ps = append(ps, "-Claim")
			if i+1 < len(args) {
				i++
				ps = append(ps, "-RunId", args[i])
			}
		case "--egress":
			ps = append(ps, "-Egress")
		case "--program", "--firewall-state":
			name := map[string]string{"--program": "-Program", "--firewall-state": "-FirewallState"}[args[i]]
			if i+1 < len(args) {
				i++
				ps = append(ps, name, args[i])
			}
		case "--fingerprint":
			ps = append(ps, "-Fingerprint")
		case "--marker":
			ps = append(ps, "-Marker")
		default:
			fmt.Fprintf(os.Stderr, "assert-disposable: unknown argument: %s\n", args[i])
			return 2
		}
	}
	cmd := exec.Command("pwsh", ps...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return 2
	}
	return 0
}

func firstLines(path string, n int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.SplitAfter(string(b), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.ReplaceAll(strings.Join(lines, ""), "\n", " ")
}

// prettyOS is PRETTY_NAME out of /etc/os-release, unquoted, as the shell
// half's sed read it.
func prettyOS() string {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			v = strings.TrimPrefix(v, `"`)
			v = strings.TrimSuffix(v, `"`)
			if !strings.Contains(v, `"`) {
				return v
			}
		}
	}
	return ""
}
