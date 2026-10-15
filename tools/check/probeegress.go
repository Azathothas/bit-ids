// probe-egress - have THIS program try to leave the host, and say whether the
// host stopped it.
//
// CI-11, CI-08. `capture.yml`'s Windows job proves its firewall rules by having
// the capture's own blocked driver, `curl.exe`, try to reach GitHub. A client
// capture has no such driver: what it runs is the observer, the product, the
// connector's interpreter and this binary, which runs the adapter and the
// capture. ⭐ So the prover is this binary, which is on the blocked list for
// that reason and needs no second program whose rule would prove nothing about
// the others.
//
// ⛔ ONLY A CONNECTION THE HOST REFUSED IS A BLOCK. A name that did not resolve
// or a dial that timed out is something else failing, and calling it the block
// would pass a firewall that never refused anything - the reason the curl
// prover accepts exit 7 alone. ⚠ The name is resolved first and on its own, so
// a DNS failure is told apart from a refused connect rather than inferred.
//
// ⚠ This says nothing about whether the host itself can reach the address. The
// workflow step that runs it has the host make the same request as its
// control, as capture.yml does.
//
// Usage:
//
//	bit-check probe-egress <host:port>
//
// Exit codes: 0 the connection was refused by this host, 1 it connected, or
// failed in a way that is not a block, 2 could not run.
package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

func probeEgress(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "probe-egress: usage: bit-check probe-egress <host:port>")
		return 2
	}
	host, port, err := net.SplitHostPort(args[0])
	if err != nil || host == "" || port == "" {
		fmt.Fprintf(os.Stderr, "probe-egress: [%s] is not host:port\n", args[0])
		return 2
	}
	addrs, err := net.LookupHost(host)
	if err != nil || len(addrs) == 0 {
		fmt.Fprintf(os.Stderr, "probe-egress: %s did not resolve (%v), which is not a block\n", host, err)
		return 1
	}
	target := net.JoinHostPort(addrs[0], port)
	started := time.Now()
	conn, err := net.DialTimeout("tcp", target, 20*time.Second)
	took := time.Since(started).Round(time.Millisecond)
	if err == nil {
		conn.Close()
		fmt.Fprintf(os.Stderr, "probe-egress: %s connected to %s (%s); nothing blocks it\n", probeSelf(), target, took)
		return 1
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		fmt.Fprintf(os.Stderr, "probe-egress: the connect to %s timed out after %s, which is not a refusal\n", target, took)
		return 1
	}
	fmt.Printf("blocked: %s could not connect to %s (%s) after %s\n", probeSelf(), target, err, took)
	return 0
}

func probeSelf() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "bit-check"
}
