// connector.go - drive the second connector, `cpython-stdlib.py`, over
// transcripts written here, and plant each defect its reading exists to
// refuse. `OBS-07`.
//
// Ported from scripts/capture/check-connector.sh, `CI-07`, and compared case for
// case against it before that half was deleted. ⚠ It had no PowerShell twin,
// which is why the Windows lane declared this row.
//
// ⛔ THE TRANSCRIPTS ARE WRITTEN HERE, BYTE BY BYTE, AND NOT BY THE OBSERVER.
// A connector checked against what the observer wrote would be read against the
// project's own encoding of the wire, which is the agreement this connector
// exists to break. Every segment is hex this file composed.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func hexOf(s string) string { return hex.EncodeToString([]byte(s)) }

// pctOf escapes every byte, as the shell half's `od | sed 's/../%&/g'` did.
func pctOf(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		fmt.Fprintf(&b, "%%%02x", c)
	}
	return b.String()
}

func checkConnector(r *repo) (verdict, error) {
	h := &harness{me: "check-connector"}

	py := python3()
	if py == "" {
		return verdict{}, errCannotRun("python3 not found; the connector under test is a Python program")
	}
	connector := filepath.Join(r.root, "scripts", "capture", "connectors", "cpython-stdlib.py")
	if _, err := os.Stat(connector); err != nil {
		return verdict{}, errCannotRun(connector + " is not present")
	}

	work, err := scratch("checkconnector")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	at := func(name string) string { return filepath.Join(work, name) }

	protocolHex := hexOf("BitTorrent protocol")
	infoHashHex := strings.Repeat("aa", 20)
	const reservedHex = "0000000000100005"
	const peer = "-XX0000-abcdefghijkl"
	peerHex := hexOf(peer)
	const other = "-XX0000-mnopqrstuvwx"
	otherHex := hexOf(other)
	const agent = "fixture-client/1.2.3"
	agentHex := hexOf(agent)
	labHex := hexOf("-BI0000-labsidelabsi")

	// An announce as a build sends one. `agentLine` is the header and its
	// newline, and an empty one is an announce with no User-Agent.
	announce := func(escaped string, agentLine ...string) string {
		line := "User-Agent: " + agent + "\n"
		if len(agentLine) > 0 {
			line = agentLine[0]
		}
		return hexOf("GET /announce?info_hash=%aa%bb&peer_id=" + escaped + "&port=51413 HTTP/1.1\nHost: 127.0.0.1:1\n" + line + "\n")
	}
	handshake := func(peerIDHex string) string {
		return "13" + protocolHex + reservedHex + infoHashHex + peerIDHex
	}
	transcript := func(path, endpoint string, segments []string) error {
		var b strings.Builder
		b.WriteString("{\n")
		b.WriteString("  \"schema\": \"bit-ids/transcript/1\",\n")
		fmt.Fprintf(&b, "  \"endpoint\": \"%s\",\n", endpoint)
		b.WriteString("  \"segments\": [\n")
		for i, seg := range segments {
			dir, bytes, _ := strings.Cut(seg, ":")
			b.WriteString("    {\n")
			b.WriteString("      \"connection\": 1,\n")
			fmt.Fprintf(&b, "      \"offset_ms\": %d,\n", (i+1)*100)
			fmt.Fprintf(&b, "      \"direction\": \"%s\",\n", dir)
			fmt.Fprintf(&b, "      \"bytes\": \"%s\"\n", bytes)
			if i == len(segments)-1 {
				b.WriteString("    }\n")
			} else {
				b.WriteString("    },\n")
			}
		}
		b.WriteString("  ]\n}\n")
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	bundle := func(name string, tracker []string, peers ...string) string {
		dir := at(name)
		_ = os.RemoveAll(dir)
		if os.MkdirAll(dir, 0o755) != nil ||
			transcript(filepath.Join(dir, "tracker-http.transcript.json"), "tracker-http", tracker) != nil ||
			transcript(filepath.Join(dir, "peer-wire-dialled.transcript.json"), "peer-wire-dialled", peers) != nil {
			return ""
		}
		return dir
	}

	cleanTracker := "from_target:" + announce(pctOf(peer))
	cleanPeerLab := "to_target:" + handshake(labHex)
	cleanPeer := "from_target:" + handshake(peerHex)

	runs := 0
	report := func(run int) string { return at(fmt.Sprintf("run-%d.txt", run)) }
	runCase := func(wantRC int, wantText, name, dir string) {
		runs++
		errPath := at(fmt.Sprintf("run-%d.err", runs))
		rc := runTo(at("null"), errPath, py, connector, "--bundle", dir, "--out", report(runs))
		if rc != wantRC {
			h.bad(fmt.Sprintf("%s (exit %d, wanted %d)", name, rc, wantRC))
			return
		}
		if wantText != "" && !anyContains(fileLines(errPath), wantText) {
			h.bad(fmt.Sprintf("%s (the refusal did not say [%s])", name, wantText))
			return
		}
		h.ok(name)
	}
	// ⚠ What every check read is kept until the check is decided, so a row
	// that fails says what it saw, quoted, a carriage return included. A Windows
	// run failed seventeen rows with nothing beside them on 2026-10-14.
	var seen []string
	reported := func(run int, field string) string {
		v := ""
		for _, l := range fileLines(report(run)) {
			if k, val, ok := strings.Cut(l, "="); ok && k == field {
				v = val
				break
			}
		}
		seen = append(seen, fmt.Sprintf("%s=%q", field, v))
		return v
	}
	check := func(ok bool, name string) {
		if ok {
			h.ok(name)
		} else if len(seen) > 0 {
			h.bad(name + " (read " + strings.Join(seen, " ") + ")")
		} else {
			h.bad(name)
		}
		seen = nil
	}

	clean := bundle("clean", []string{cleanTracker}, cleanPeerLab, cleanPeer)
	if clean == "" {
		return verdict{}, errCannotRun("cannot write the clean bundle")
	}
	runCase(0, "", "a clean bundle is read", clean)
	cleanRun := runs
	for _, want := range [][2]string{
		{"tracker_http/peer_id", peerHex}, {"peer_wire/peer_id", peerHex},
		{"peer_wire/reserved", reservedHex}, {"tracker_http/user_agent", agentHex},
	} {
		check(reported(cleanRun, want[0]) == want[1], "the report carries the bytes the transcript holds for "+want[0])
	}
	check(reported(cleanRun, "peer_wire/peer_id") != labHex, "the lab's own to_target handshake is not read as the build's")

	runCase(0, "", "a bundle carrying different bytes is read",
		bundle("moved", []string{"from_target:" + announce(pctOf(other))}, cleanPeerLab, "from_target:"+handshake(otherHex)))
	check(reported(runs, "tracker_http/peer_id") == otherHex && reported(runs, "peer_wire/peer_id") == otherHex,
		"moving the peer ID in the transcript moves it in the report")

	const altReserved = "0000000000000004"
	runCase(0, "", "a handshake with different reserved bytes is read",
		bundle("reserved", []string{cleanTracker}, "from_target:13"+protocolHex+altReserved+infoHashHex+peerHex))
	check(reported(runs, "peer_wire/reserved") == altReserved && reported(runs, "peer_wire/peer_id") == peerHex,
		"the reserved bytes and the peer ID are read at their own offsets")

	const plus = "-XX0000-a+b+c+d+e+f"
	plusHex := hexOf(plus)
	runCase(0, "", "an announce leaving + bare is read",
		bundle("plusbare", []string{"from_target:" + announce(strings.ReplaceAll(plus, "-", "%2d"))}, cleanPeer))
	check(reported(runs, "tracker_http/peer_id") == plusHex, "a bare + in an escaped peer ID is the byte 0x2b, not a space")

	runCase(0, "", "an announce escaping + is read",
		bundle("plusescaped", []string{"from_target:" + announce(pctOf(plus))}, cleanPeer))
	check(reported(runs, "tracker_http/peer_id") == plusHex, "an escaped and a bare + decode to the same byte")

	highHex := peerHex[:len(peerHex)-2] + "ff"
	var highEsc strings.Builder
	for i := 0; i < len(highHex); i += 2 {
		highEsc.WriteString("%" + highHex[i:i+2])
	}
	runCase(0, "", "an announce carrying a byte above 0x7f is read",
		bundle("high", []string{"from_target:" + announce(highEsc.String())}, cleanPeer))
	check(reported(runs, "tracker_http/peer_id") == highHex, "a peer-ID byte above 0x7f survives the decoding")

	runCase(0, "", "two announces that agree are read",
		bundle("twice", []string{"from_target:" + announce(pctOf(peer)), "from_target:" + announce(pctOf(peer))}, cleanPeer))
	check(reported(runs, "tracker_http/peer_id") == peerHex+","+peerHex &&
		reported(runs, "tracker_http/user_agent") == agentHex+","+agentHex,
		"two agreeing announces are two samples, not one value")

	runCase(0, "", "two announces that differ are read rather than refused",
		bundle("varies", []string{"from_target:" + announce(pctOf(peer)), "from_target:" + announce(pctOf(other))}, cleanPeer))
	check(reported(runs, "tracker_http/peer_id") == peerHex+","+otherHex,
		"two differing announces report both peer IDs, in the order they were sent")

	runCase(0, "", "the same two announces sent the other way round are read",
		bundle("reversed", []string{"from_target:" + announce(pctOf(other)), "from_target:" + announce(pctOf(peer))}, cleanPeer))
	check(reported(runs, "tracker_http/peer_id") == otherHex+","+peerHex,
		"the samples follow the transcript's order rather than their own")

	runCase(0, "", "two handshakes with two tails are read",
		bundle("twohandshakes", []string{cleanTracker}, cleanPeerLab, cleanPeer, cleanPeerLab, "from_target:"+handshake(otherHex)))
	check(reported(runs, "peer_wire/peer_id") == peerHex+","+otherHex &&
		reported(runs, "peer_wire/reserved") == reservedHex+","+reservedHex,
		"two handshakes report both peer IDs and both reserved blocks")

	runCase(0, "", "an announce with no User-Agent is read",
		bundle("noagent", []string{"from_target:" + announce(pctOf(peer), "")}, cleanPeer))
	check(reported(runs, "tracker_http/user_agent") == "absent" && reported(runs, "tracker_http/peer_id") == peerHex,
		"a missing header is absent and the rest of the announce still reads")

	runCase(0, "", "two announces of which one has no User-Agent are read",
		bundle("agentonce", []string{"from_target:" + announce(pctOf(peer)), "from_target:" + announce(pctOf(peer), "")}, cleanPeer))
	check(reported(runs, "tracker_http/user_agent") == agentHex+",absent", "an absent header keeps its position among the samples")

	runCase(0, "", "a peer transcript the build never answered is read",
		bundle("nohandshake", []string{cleanTracker}, cleanPeerLab))
	check(reported(runs, "peer_wire/peer_id") == "absent", "a handshake the build never sent is absent")

	upper := strings.Replace(cleanPeer, infoHashHex, strings.ToUpper(infoHashHex), 1)
	if upper == cleanPeer || len(upper) != len(cleanPeer) {
		h.bad("the uppercase plant applied without changing the length")
	} else {
		h.ok("the uppercase plant applied without changing the length")
		runCase(1, "is not lowercase hex", "a transcript carrying uppercase hex is refused",
			bundle("upper", []string{cleanTracker}, upper))
	}

	runCase(1, "odd number of hex digits", "a transcript carrying odd-length hex is refused",
		bundle("odd", []string{cleanTracker}, "from_target:"+peerHex+"a"))

	schemaDir := bundle("schema", []string{cleanTracker}, cleanPeer)
	schemaPath := filepath.Join(schemaDir, "peer-wire-dialled.transcript.json")
	if replaceOnce(schemaPath, "bit-ids/transcript/1", "bit-ids/transcript/2") {
		h.ok("the schema plant applied")
		runCase(1, "not bit-ids/transcript/1", "a transcript declaring another schema is refused", schemaDir)
	} else {
		h.bad("the schema plant applied")
	}

	notJSON := bundle("notjson", []string{cleanTracker}, cleanPeer)
	_ = os.WriteFile(filepath.Join(notJSON, "tracker-http.transcript.json"), []byte("this is not a document\n"), 0o644)
	runCase(1, "is not a JSON document", "a transcript that is not JSON is refused", notJSON)

	missing := bundle("missing", []string{cleanTracker}, cleanPeer)
	_ = os.Remove(filepath.Join(missing, "peer-wire-dialled.transcript.json"))
	runCase(0, "", "a bundle carrying no peer transcript is read", missing)
	check(reported(runs, "peer_wire/peer_id") == "out_of_scope" && reported(runs, "peer_wire/reserved") == "out_of_scope" &&
		reported(runs, "tracker_http/peer_id") == peerHex,
		"a surface the bundle does not carry is out_of_scope, and the other one still reads")

	unreadable := bundle("unreadable", []string{cleanTracker}, cleanPeer)
	_ = os.Remove(filepath.Join(unreadable, "peer-wire-dialled.transcript.json"))
	if os.MkdirAll(filepath.Join(unreadable, "peer-wire-dialled.transcript.json"), 0o755) != nil {
		return verdict{}, errCannotRun("cannot plant a directory where a transcript belongs")
	}
	runCase(1, "peer-wire-dialled.transcript.json", "a transcript that is present and will not read is refused", unreadable)

	runCase(1, "not 68", "a segment that starts as a handshake and is short is refused",
		bundle("short", []string{cleanTracker}, "from_target:13"+protocolHex+reservedHex))

	runCase(0, "", "a segment that is neither surface is skipped",
		bundle("extra", []string{cleanTracker}, cleanPeer, "from_target:"+hexOf("not a handshake at all")))
	check(reported(runs, "peer_wire/peer_id") == peerHex, "a trailing segment does not disturb the handshake that preceded it")

	// -- the arguments, and what the connector says about itself ---------------
	rc := runTo(at("null"), at("nobundle.err"), py, connector)
	check(rc == 2 && anyContains(fileLines(at("nobundle.err")), "--bundle is required"), "no --bundle at all is could-not-run")

	rc = runTo(at("null"), at("notdir.err"), py, connector, "--bundle", filepath.Join(clean, "tracker-http.transcript.json"))
	check(rc == 2 && anyContains(fileLines(at("notdir.err")), "is not a directory"), "a --bundle that is not a directory is could-not-run")

	runTo(at("describe.out"), at("null"), py, connector, "--describe")
	described := ""
	for _, l := range fileLines(at("describe.out")) {
		if k, v, ok := strings.Cut(l, "="); ok && k == "id" {
			described = v
			break
		}
	}
	seen = append(seen, fmt.Sprintf("described=%q", described))
	check(described != "", "the connector describes its own identifier")
	seen = append(seen, fmt.Sprintf("described=%q", described))
	slug := described != "" && !strings.HasPrefix(described, "-") && !strings.HasSuffix(described, "-") &&
		!strings.Contains(described, "--")
	for _, c := range described {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			slug = false
		}
	}
	check(slug, "the described identifier is a Slug")

	def := bundle("default", []string{cleanTracker}, cleanPeer)
	filed := filepath.Join(def, "connector", described+".txt")
	defRC := runTo(at("null"), at("null"), py, connector, "--bundle", def)
	st, statErr := os.Stat(filed)
	check(defRC == 0 && statErr == nil && st.Size() > 0,
		"the default report is filed under the described identifier")
	lines := fileLines(filed)
	for _, field := range []string{"tracker_http/peer_id", "tracker_http/user_agent", "peer_wire/peer_id", "peer_wire/reserved"} {
		found := false
		for _, l := range lines {
			if strings.HasPrefix(l, field+"=") {
				found = true
			}
		}
		check(found, "the report speaks about "+field)
	}

	return h.report("check-connector/1", "cases"), nil
}
