// anacrolix-torrent-harness - the ENGINE-02 reference harness, the smallest
// program that runs github.com/anacrolix/torrent on one torrent.
//
// ⛔ THE LIBRARY'S DEFAULTS, EXCEPT WHERE A FLAG SAYS OTHERWISE. The client is
// torrent.NewDefaultClientConfig(); every flag below defaults to that config's
// own value, so the harness run with no flag is the library as an embedding
// program gets it. The capture adapter, tools/check/adapter_anacrolix-torrent.go,
// passes the flags that switch the network surfaces off, and reads back what
// this program reports it configured.
//
// ⛔ NOTHING ON THE WIRE IS SET HERE. The peer ID, its BEP 20 prefix, the HTTP
// user agent and the extended handshake's `v` are the library's: its `version`
// package derives the last two from this program's build information. This
// program prints them as a self-report, which is never a published value.
//
// ⚠ ITS OWN MODULE. The library has dependencies and `bit-check` has none, so
// the harness is a separate module with its own go.mod and go.sum, pinned in
// this tree and reviewed with it. `TODO/engines.md` `ENGINE-02` records why.
//
// Usage:
//
//	anacrolix-torrent-harness --version
//	anacrolix-torrent-harness -torrent F -data D -ready R -stop-file S [flags]
//
// Exit codes: 0 done, 1 the library refused, 2 could not run.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/version"
)

// libraryPath is the module this harness exists to run.
const libraryPath = "github.com/anacrolix/torrent"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		os.Exit(printVersion())
	}
	os.Exit(run())
}

// printVersion is the build answering for itself: the library's module line
// first, then the whole build information `go version -m` prints. ⭐ This is
// the build manifest, read out of the executable rather than out of a go.mod.
func printVersion() int {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Fprintln(os.Stderr, "anacrolix-torrent-harness: this build carries no build information")
		return 2
	}
	lib := ""
	for _, d := range bi.Deps {
		if d.Path == libraryPath {
			lib = d.Version + " " + d.Sum
			if d.Replace != nil {
				lib += " => " + d.Replace.Path + " " + d.Replace.Version + " " + d.Replace.Sum
			}
		}
	}
	if lib == "" {
		fmt.Fprintln(os.Stderr, "anacrolix-torrent-harness: the build information names no "+libraryPath)
		return 2
	}
	fmt.Printf("anacrolix-torrent-harness library %s %s\n", libraryPath, lib)
	fmt.Print(bi.String())
	if !strings.HasSuffix(bi.String(), "\n") {
		fmt.Println()
	}
	return 0
}

// peerIDHex is the client's peer ID as hex. ⚠ Over the bytes: the type has a
// String method, which `%x` would format instead.
func peerIDHex(cl *torrent.Client) string {
	id := cl.PeerID()
	return hex.EncodeToString(id[:])
}

func run() int {
	cfg := torrent.NewDefaultClientConfig()
	fs := flag.NewFlagSet("anacrolix-torrent-harness", flag.ContinueOnError)
	torrentFile := fs.String("torrent", "", "the torrent file to add")
	data := fs.String("data", "", "the directory the torrent's data is stored in")
	ready := fs.String("ready", "", "where to write what the client was configured with once it runs")
	stopFile := fs.String("stop-file", "", "the client closes when this file appears")
	port := fs.Int("port", cfg.ListenPort, "the peer port")
	// ⛔ EACH SURFACE DEFAULTS TO THE LIBRARY'S OWN VALUE.
	dht := fs.Bool("dht", !cfg.NoDHT, "run the DHT")
	pex := fs.Bool("pex", !cfg.DisablePEX, "exchange peers (ut_pex)")
	portForward := fs.Bool("port-forward", !cfg.NoDefaultPortForwarding, "UPnP and NAT-PMP port forwarding")
	webseeds := fs.Bool("webseeds", !cfg.DisableWebseeds, "fetch from a torrent's web seeds")
	webtorrent := fs.Bool("webtorrent", !cfg.DisableWebtorrent, "WebRTC peers through websocket trackers")
	ipv6 := fs.Bool("ipv6", !cfg.DisableIPv6, "listen on and dial IPv6")
	utp := fs.Bool("utp", !cfg.DisableUTP, "uTP as well as TCP")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *torrentFile == "" || *data == "" || *ready == "" || *stopFile == "" {
		fmt.Fprintln(os.Stderr, "anacrolix-torrent-harness: -torrent, -data, -ready and -stop-file are required, and nothing else")
		return 2
	}
	cfg.DataDir = *data
	cfg.ListenPort = *port
	cfg.NoDHT = !*dht
	cfg.DisablePEX = !*pex
	cfg.NoDefaultPortForwarding = !*portForward
	cfg.DisableWebseeds = !*webseeds
	cfg.DisableWebtorrent = !*webtorrent
	cfg.DisableIPv6 = !*ipv6
	cfg.DisableUTP = !*utp

	cl, err := torrent.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "anacrolix-torrent-harness: the client would not start: %v\n", err)
		return 1
	}
	t, err := cl.AddTorrentFromFile(*torrentFile)
	if err != nil {
		cl.Close()
		fmt.Fprintf(os.Stderr, "anacrolix-torrent-harness: the torrent would not load: %v\n", err)
		return 1
	}
	t.DownloadAll()

	var addrs []string
	for _, a := range cl.ListenAddrs() {
		addrs = append(addrs, a.Network()+" "+a.String())
	}
	sort.Strings(addrs)
	var b strings.Builder
	b.WriteString("bit-ids/anacrolix-torrent-harness-ready/1\n")
	for _, kv := range [][2]string{
		{"pid", fmt.Sprint(os.Getpid())},
		{"infohash", t.InfoHash().HexString()},
		{"listen_port", fmt.Sprint(cfg.ListenPort)},
		{"listen", strings.Join(addrs, ",")},
		{"no_dht", fmt.Sprint(cfg.NoDHT)},
		{"disable_pex", fmt.Sprint(cfg.DisablePEX)},
		{"no_default_port_forwarding", fmt.Sprint(cfg.NoDefaultPortForwarding)},
		{"disable_webseeds", fmt.Sprint(cfg.DisableWebseeds)},
		{"disable_webtorrent", fmt.Sprint(cfg.DisableWebtorrent)},
		{"disable_ipv6", fmt.Sprint(cfg.DisableIPv6)},
		{"disable_utp", fmt.Sprint(cfg.DisableUTP)},
		{"disable_trackers", fmt.Sprint(cfg.DisableTrackers)},
		{"http_proxy_set", fmt.Sprint(cfg.HTTPProxy != nil)},
		{"ice_servers", fmt.Sprint(len(cl.ICEServers()))},
		// ⚠ SELF-REPORTS, printed so the record can set them beside what the
		// observer measured. None is ever a published value.
		{"self_report_peer_id", peerIDHex(cl)},
		{"self_report_bep20", cfg.Bep20},
		{"self_report_http_user_agent", cfg.HTTPUserAgent},
		{"self_report_extended_handshake_v", cfg.ExtendedHandshakeClientVersion},
		{"self_report_upnp_id", cfg.UpnpID},
		{"library_default_bep20", version.DefaultBep20Prefix},
	} {
		fmt.Fprintf(&b, "%s=%s\n", kv[0], kv[1])
	}
	if err := os.WriteFile(*ready, []byte(b.String()), 0o644); err != nil {
		cl.Close()
		fmt.Fprintf(os.Stderr, "anacrolix-torrent-harness: cannot write %s: %v\n", *ready, err)
		return 2
	}
	fmt.Printf("running %s on %s\n", t.InfoHash().HexString(), strings.Join(addrs, ","))

	// ⭐ THE CLOSE IS THE LIBRARY'S OWN, on a file appearing or on a signal: a
	// file because Windows has no SIGTERM to send a console-less process, and
	// one mechanism for both platforms keeps the two halves one harness.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	why := ""
	for why == "" {
		select {
		case s := <-sig:
			why = "signal " + s.String()
		case <-tick.C:
			if _, err := os.Stat(*stopFile); err == nil {
				why = "stop file"
			}
		}
	}
	started := time.Now()
	errs := cl.Close()
	fmt.Printf("closed on %s in %dms, %d errors\n", why, time.Since(started).Milliseconds(), len(errs))
	for _, e := range errs {
		fmt.Printf("close error: %v\n", e)
	}
	return 0
}
