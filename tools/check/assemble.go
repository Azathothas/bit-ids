// assemble.go - can a dispatched capture's artifacts become a record, and does
// the assembler refuse every way they can fall short? `CI-09`.
//
// ⛔ WHAT THIS EXISTS FOR IS A SET OF REFUSALS NOBODY HAD SEEN. `CI-09` waited on
// a two-route capture for weeks; `capture-client` run 14 produced one, and
// assembling it on 2026-09-09 refused for reasons no reading had found. None of
// them is visible in an attestation and all of them are visible the moment
// something tries to write the record.
//
// Ported from scripts/capture/check-assemble.sh, `CI-07` and `CI-10`, and
// compared case for case against it before that half is deleted. ⚠ It had no
// PowerShell twin, which is why the Windows lane declared this row. Its six
// dispatch rows drive `bit-check assemble-dispatch`, which is this binary, so
// the whole harness reaches the Windows lane with no `sh` on its path.
//
// -- ⭐ IT RUNS OVER A SYNTHETIC PAIR AND SAYS SO -----------------------------
//
// The lanes here are written by this harness, not downloaded. A gate case that
// fetched two artifacts from a run would go red the day that run expired, and
// would be measuring GitHub's retention rather than this project's assembler.
// ⚠ SO THIS PROVES THE ASSEMBLER AND NOTHING ABOUT RUN 14. What run 14 answered
// is a driven pass, recorded in `TODO/ci.md` under `CI-09`.
//
// ⚠ NO VALUE HERE IS ANY PRODUCT'S. The peer-ID prefix is `-XX0000-`, which no
// vendor uses, so nothing this harness writes can be mistaken for a measurement.
// `docs/architecture.md` section 5 says why a fixture is never evidence.
//
// ⛔ NOTHING HERE PLANTS IN A TRACKED FILE. Every lane a case mutates is written
// into scratch state, because a gate check that edits a tracked file leaves a
// dirty tree behind when it is interrupted.
//
// -- ⛔ EVERY FIXTURE IS THE SHELL HALF'S, BYTE FOR BYTE -------------------------
//
// The lanes, transcripts, connector reports and resolutions below are written
// with the bytes `check-assemble.sh` wrote, so a case that agrees with that half
// agrees over the same input rather than over a translation of it. ⚠ Its `od`
// and `sha256sum` are `encoding/hex` and `crypto/sha256` here, and its `sed`
// plants are line rewrites with the same expression, so nothing is shelled out
// for a fixture.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// sha256Hex is `printf '%s' TEXT | sha256sum | cut -d' ' -f1`.
func sha256Hex(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

// findFiles is `find DIR -name PATTERN -type f`, in walk order, which is the
// order `head -1` then takes the first of.
func findFiles(dir, pattern string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		if ok, _ := filepath.Match(pattern, d.Name()); ok {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func firstFound(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// rewriteLines is `sed EXPR FROM >TO` and `grep -v PATTERN FROM >TO`: each line
// handed to fn, which answers its replacement and whether it is kept.
func rewriteLines(from, to string, fn func(string) (string, bool)) error {
	var b strings.Builder
	for _, l := range fileLines(from) {
		if out, keep := fn(l); keep {
			b.WriteString(out)
			b.WriteString("\n")
		}
	}
	return os.WriteFile(to, []byte(b.String()), 0o644)
}

// sameBytes is `cmp -s A B`.
func sameBytes(a, b string) bool {
	x, err1 := os.ReadFile(a)
	y, err2 := os.ReadFile(b)
	return err1 == nil && err2 == nil && bytes.Equal(x, y)
}

// countMatching is `grep -c PATTERN FILE`: the lines a pattern matches.
func countMatching(lines []string, re *regexp.Regexp) int {
	n := 0
	for _, l := range lines {
		if re.MatchString(l) {
			n++
		}
	}
	return n
}

func countContaining(lines []string, needle string) int {
	n := 0
	for _, l := range lines {
		if strings.Contains(l, needle) {
			n++
		}
	}
	return n
}

func checkAssemble(r *repo) (verdict, error) {
	h := &harness{me: "check-assemble"}

	assemble, err := exampleBinary(r.root, "assemble-capture", "bit-ids-probe")
	if err != nil {
		return verdict{}, err
	}
	validateRun, err := exampleBinary(r.root, "validate-run", "bit-ids")
	if err != nil {
		return verdict{}, err
	}
	// ⭐ `bit-check assemble-dispatch` since 2026-10-15, `CI-07`, and that is
	// THIS binary: the dispatch rows run it as the shell half ran the one it
	// built, as a process of its own whose two streams go to one file.
	self, err := os.Executable()
	if err != nil {
		return verdict{}, errCannotRun("cannot name this binary, so the dispatch rows have nothing to run")
	}

	work, err := scratch("checkassemble")
	if err != nil {
		return verdict{}, err
	}
	defer os.RemoveAll(work)
	at := func(name string) string { return filepath.Join(work, name) }

	// ⚠ What a failing row would show a person, in the order the shell half
	// printed it: ahead of the report, indented under nothing, and never in
	// --json mode.
	var notes strings.Builder
	note := func(lines []string) {
		for _, l := range lines {
			fmt.Fprintf(&notes, "          %s\n", l)
		}
	}
	head := func(lines []string, n int) []string {
		if len(lines) > n {
			return lines[:n]
		}
		return lines
	}
	tail := func(lines []string, n int) []string {
		if len(lines) > n {
			return lines[len(lines)-n:]
		}
		return lines
	}
	grepF := func(lines []string, needle string) []string {
		var out []string
		for _, l := range lines {
			if strings.Contains(l, needle) {
				out = append(out, l)
			}
		}
		return out
	}

	// ⛔ EVERY HEX RUN HERE IS COMPUTED, NEVER TYPED, and that is a rule rather
	// than a preference. An info hash and a peer ID are both forty hex digits,
	// which is the shape `check-no-secrets --public` refuses because it is also
	// the shape of a credential. `docs/security/secrets.md` says to narrow a
	// pattern rather than switch a rule off; computing the value narrows nothing
	// and needs no exclusion.
	protocolHex := hexOf("BitTorrent protocol")
	infoHashHex := strings.Repeat("aa", 20)
	const reservedHex = "0000000000100005"
	const agent = "fixture-client/1.2.3"

	// One announce, in the shape a stock client sends and `tracker_http` parses.
	//
	// ⚠ AN EVENT ADDS `event=`, and `started` is what delimits a session. BEP 3
	// makes it the first announce of a run, so a second one in a transcript is a
	// second separately started process - which is the only thing that separates
	// `Lifetime::PerSession` from `Lifetime::PerConnection`. ⛔ Omitting it is the
	// default because every fixture here predates sessions.
	announceHex := func(peer, event string) string {
		query := "info_hash=X&peer_id=" + peer + "&port=51413"
		if event != "" {
			query += "&event=" + event
		}
		return hexOf("GET /announce?" + query + " HTTP/1.1\nHost: 127.0.0.1:1\nUser-Agent: " + agent + "\n\n")
	}
	// ⛔ A SCRAPE: a request the build sends on the same surface that is NOT an
	// announce, because it carries no `peer_id`. ⚠ The rule for one - sample the
	// announces, pass over the rest - survived a mutation pass with every case
	// green until a fixture held one.
	scrapeHex := hexOf("GET /scrape?info_hash=X HTTP/1.1\nHost: 127.0.0.1:1\nUser-Agent: " + agent + "\n\n")
	// One peer handshake: a length byte, the protocol name, eight reserved
	// bytes, the info hash and the peer ID.
	handshakeHex := func(peer string) string {
		return "13" + protocolHex + reservedHex + infoHashHex + hexOf(peer)
	}

	// One transcript document, in the exact shape `evidence.rs` writes and its
	// reader accepts. ⭐ That reader is the writer's inverse and refuses anything
	// the writer does not emit, so a document typed here that it accepts is a
	// document the lab would have written.
	//
	// Each segment is `connection:offset:hex`, which is the shell half's
	// `segments_at`; its `transcript` and `peer_segments` are the two callers
	// below that derive the connection and the offset as it did.
	segmentsAt := func(path, endpoint string, segs []string) error {
		var b strings.Builder
		b.WriteString("{\n  \"schema\": \"bit-ids/transcript/1\",\n")
		fmt.Fprintf(&b, "  \"endpoint\": \"%s\",\n  \"segments\": [\n", endpoint)
		for i, seg := range segs {
			conn, rest, _ := strings.Cut(seg, ":")
			offset, hexBytes, _ := strings.Cut(rest, ":")
			fmt.Fprintf(&b, "    {\n      \"connection\": %s,\n", conn)
			fmt.Fprintf(&b, "      \"offset_ms\": %s,\n", offset)
			b.WriteString("      \"direction\": \"from_target\",\n")
			fmt.Fprintf(&b, "      \"bytes\": \"%s\"\n", hexBytes)
			if i == len(segs)-1 {
				b.WriteString("    }\n")
			} else {
				b.WriteString("    },\n")
			}
		}
		b.WriteString("  ]\n}\n")
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	// ⚠ ONE SEGMENT PER PAYLOAD, each on a connection of its own, because a
	// transcript records a segment per connection and a field rests on every one.
	transcript := func(path, endpoint string, payloads ...string) error {
		segs := make([]string, len(payloads))
		for i, p := range payloads {
			segs[i] = fmt.Sprintf("%d:%d:%s", i+1, 2000+(i+1)*75, p)
		}
		return segmentsAt(path, endpoint, segs)
	}
	// The conversation after a handshake, segment by segment: `connection:hex`.
	peerSegments := func(path string, segs ...string) error {
		out := make([]string, len(segs))
		for i, s := range segs {
			conn, hexBytes, _ := strings.Cut(s, ":")
			out[i] = fmt.Sprintf("%s:%d:%s", conn, 2000+(i+1)*75, hexBytes)
		}
		return segmentsAt(path, "peer-wire-dialled", out)
	}

	// What the shell half set in its environment around one `write_lane`: a key
	// the attestation or install record leaves out, the runner a host claims, the
	// install record's `acquired`, and when the install finished.
	drop, runner, acquired, installed := "", "", "", ""

	// writeLane writes one lane's two artifact directories.
	//
	// ⚠ Every value a lane carries is a parameter, so a case can move exactly
	// one and the rest stay fixed. `more` is the shell half's eighth to tenth
	// arguments: the peer ID the second connector read, a second connection's
	// peer ID, and that second announce's event.
	writeLane := func(tag, route, resolver, commit, digest, version, peer string, more ...string) error {
		arg := func(i int) string {
			if i < len(more) {
				return more[i]
			}
			return ""
		}
		connPeer, second, event := arg(0), arg(1), arg(2)
		capDir := at(tag + "-capture")
		insDir := at(tag + "-install")
		_ = os.RemoveAll(capDir)
		_ = os.RemoveAll(insDir)
		for _, d := range []string{
			filepath.Join(capDir, "capture", "bundle", "fixture"),
			filepath.Join(insDir, "install-"+route),
			filepath.Join(insDir, "release"),
		} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return err
			}
		}
		write := func(path, body string) error { return os.WriteFile(path, []byte(body), 0o644) }

		var a strings.Builder
		fmt.Fprintf(&a, "bit-ids/capture-attestation/1\nrun=cap-%s\nkind=client\n", tag)
		fmt.Fprintf(&a, "target=fixture-client\nmeasured_build=%s\nstock_client=true\n", version)
		// ⚠ `uname -srm`, which is what the real attestation carries, so the
		// platform and the architecture in the record are DERIVED here exactly as
		// they are there - and `x86_64` is spelled with the underscore the kernel
		// uses rather than the hyphen this project's vocabulary uses.
		a.WriteString("platform=Linux 0.0.0-fixture x86_64\n")
		a.WriteString("started_at=2026-09-09T15:08:24Z\nfinished_at=2026-09-09T15:09:14Z\n")
		fmt.Fprintf(&a, "fixture=sha256:%s\n", sha256Hex("fixture-"+tag))
		a.WriteString("egress=closed\n")
		// ⛔ `E-CAP-01` NEEDS TWO CONNECTORS AT THE VALIDITY GATE, so a lane with
		// no connector peer is run 14's shape and cannot become a record at all.
		if connPeer != "" {
			a.WriteString("connectors=fixture-oracle\n")
		}
		if drop != "observer_version" {
			a.WriteString("observer_version=0.0.0\n")
		}
		if connPeer != "" && drop != "connector_version" {
			a.WriteString("connector_version=9.8.7\n")
		}
		if drop != "uptime_at_finish" {
			a.WriteString("uptime_at_finish=164.50\n")
		}
		if err := write(filepath.Join(capDir, "capture", "attestation.txt"), a.String()); err != nil {
			return err
		}

		// ⭐ WHAT A RUN MANIFEST IS READ FROM, `CI-09`: the host claim the capture
		// keeps, and the observer's own log for its serving window. ⚠ The times put
		// every install before the capture, which the barrier makes true on a runner.
		if drop != "host-claim" {
			env := runner
			if env == "" {
				env = "github-hosted"
			}
			var c strings.Builder
			fmt.Fprintf(&c, "bit-ids/host-claim/1\nrun=cap-%s\nclaimed_at=2026-09-09T15:08:10Z\n", tag)
			fmt.Fprintf(&c, "fingerprint=%s\n", sha256Hex("host-"+tag))
			c.WriteString("image=fixture-image/1\nos=Fixture OS 1\n")
			fmt.Fprintf(&c, "runner_environment=%s\n", env)
			c.WriteString("uptime_at_claim=100.25\n")
			if err := write(filepath.Join(capDir, "capture", "host-claim.txt"), c.String()); err != nil {
				return err
			}
		}
		if err := write(filepath.Join(capDir, "capture", "observer.log"), "serving for 30s\n"); err != nil {
			return err
		}

		// The second connector's own report, which is the contract `OBS-07` fills.
		// ⛔ EVERY SAMPLE, IN THE TRANSCRIPT'S ORDER: the connector peer is what it
		// read for the FIRST segment of each surface - which a case moves to plant a
		// conflict - and a second peer adds that segment's value as the transcript
		// carries it.
		if connPeer != "" {
			if err := os.MkdirAll(filepath.Join(capDir, "capture", "bundle", "connector"), 0o755); err != nil {
				return err
			}
			agentHex := hexOf(agent)
			ids, agents, reserved := hexOf(connPeer), agentHex, reservedHex
			if second != "" {
				ids += "," + hexOf(second)
				agents += "," + agentHex
				reserved += "," + reservedHex
			}
			body := fmt.Sprintf("tracker_http/peer_id=%s\ntracker_http/user_agent=%s\npeer_wire/peer_id=%s\npeer_wire/reserved=%s\n",
				ids, agents, ids, reserved)
			if err := write(filepath.Join(capDir, "capture", "bundle", "connector", "fixture-oracle.txt"), body); err != nil {
				return err
			}
		}

		acq := acquired
		if acq == "" {
			acq = "yes"
		}
		fin := installed
		if fin == "" {
			fin = "2026-09-09T15:08:18Z"
		}
		var i strings.Builder
		fmt.Fprintf(&i, "bit-ids/install-record/1\ntarget=fixture-client\nroute=%s\n", route)
		fmt.Fprintf(&i, "reported_version=%s\npreexisting_version=\n", version)
		i.WriteString("preexisting_binary=\npreexisting_binary_sha256=\n")
		i.WriteString("installed_binary=/usr/local/bin/fixture-client\n")
		fmt.Fprintf(&i, "installed_binary_sha256=%s\n", digest)
		fmt.Fprintf(&i, "acquired=%s\nsource_commit=%s\npackage=elf-binary\n", acq, commit)
		i.WriteString("adapter=scripts/capture/adapters/fixture.sh\n")
		fmt.Fprintf(&i, "started_at=2026-09-09T15:08:17Z\nfinished_at=%s\n", fin)
		if drop != "installed_binary_bytes" {
			i.WriteString("installed_binary_bytes=4096\n")
		}
		i.WriteString("retrieved_at=2026-09-09T15:08:18Z\nsignature=not_checked\n")
		if err := write(filepath.Join(capDir, "install-"+route+".txt"), i.String()); err != nil {
			return err
		}

		// ⛔ A SOURCE LANE'S RESOLUTION IS A DIFFERENT DOCUMENT IN A DIFFERENT
		// PLACE, carrying the clone URL in `source_url`, `refs_sha256` instead of a
		// listing digest, and NO `asset_url` at all, because a source route selects
		// no asset. ⚠ Giving every lane an `asset_url` once made the branch that
		// refuses a source route for a field its document does not carry
		// unreachable. Run 16 is what found it.
		if route == "source" {
			if err := os.MkdirAll(filepath.Join(insDir, "source"), 0o755); err != nil {
				return err
			}
			var s strings.Builder
			s.WriteString("bit-ids/source-resolution/1\ntarget=fixture-client\n")
			s.WriteString("repository=fixture-owner/fixture-client\n")
			fmt.Fprintf(&s, "source_url=%s\n", resolver)
			fmt.Fprintf(&s, "refs_sha256=%s\n", sha256Hex("refs-"+version))
			fmt.Fprintf(&s, "selected_version=%s\nselected_tag=v%s\n", version, version)
			s.WriteString("started_at=2026-09-09T15:08:12Z\nfinished_at=2026-09-09T15:08:13Z\n")
			if err := write(filepath.Join(insDir, "source", "resolution.txt"), s.String()); err != nil {
				return err
			}
		} else {
			var s strings.Builder
			s.WriteString("bit-ids/release-resolution/1\ntarget=fixture-client\n")
			s.WriteString("repository=fixture-owner/fixture-client\n")
			fmt.Fprintf(&s, "source_url=%s\n", resolver)
			fmt.Fprintf(&s, "selected_version=%s\nselected_tag=v%s\n", version, version)
			fmt.Fprintf(&s, "asset=fixture-client-%s-linux-x86-64\n", version)
			fmt.Fprintf(&s, "asset_url=https://example.invalid/fixture-client-%s\n", version)
			s.WriteString("started_at=2026-09-09T15:08:12Z\nfinished_at=2026-09-09T15:08:13Z\n")
			if err := write(filepath.Join(insDir, "release", "resolution.txt"), s.String()); err != nil {
				return err
			}
		}

		// ⛔ `E-ACQ-10` NEEDS WHAT THE BUILD PRINTED, AND IT IS IN THE INSTALL
		// ARTIFACT rather than in the capture bundle, which is run 14's layout.
		if err := write(filepath.Join(insDir, "install-"+route, "version.err"),
			fmt.Sprintf("fixture-client %s\ncompiled for the %s route\n", version, route)); err != nil {
			return err
		}
		if err := write(filepath.Join(capDir, "capture", "bundle", "fixture", "generated.torrent"),
			"d4:infod4:name7:fixtureee"); err != nil {
			return err
		}
		// ⭐ A second peer is a SECOND CONNECTION on each surface, carrying its own
		// peer ID - which is what a build with a per-connection tail really puts in
		// a transcript. ⭐ An event makes the second announce a second session.
		// ⛔ The peer surface is left alone on purpose: nothing in a handshake
		// delimits a restart.
		bundle := filepath.Join(capDir, "capture", "bundle")
		if second != "" {
			if err := transcript(filepath.Join(bundle, "tracker-http.transcript.json"), "tracker-http",
				announceHex(peer, ""), announceHex(second, event)); err != nil {
				return err
			}
			return transcript(filepath.Join(bundle, "peer-wire-dialled.transcript.json"), "peer-wire-dialled",
				handshakeHex(peer), handshakeHex(second))
		}
		if err := transcript(filepath.Join(bundle, "tracker-http.transcript.json"), "tracker-http",
			announceHex(peer, "")); err != nil {
			return err
		}
		return transcript(filepath.Join(bundle, "peer-wire-dialled.transcript.json"), "peer-wire-dialled",
			handshakeHex(peer))
	}
	// ⛔ A lane the harness cannot write is could-not-run, as the shell half's
	// `|| exit 2` made it.
	var laneErr error
	lane := func(tag, route, resolver, commit, digest, version, peer string, more ...string) {
		if laneErr == nil {
			if err := writeLane(tag, route, resolver, commit, digest, version, peer, more...); err != nil {
				laneErr = errCannotRun("cannot write the " + tag + " lane: " + err.Error())
			}
		}
	}

	// Runs the assembler over some lanes into a fresh store, keeping the streams
	// apart. ⛔ The exit code is read from the process that produced it.
	assembleInto := func(tag string, lanes ...string) int {
		store := at("store-" + tag)
		if os.RemoveAll(store) != nil || os.MkdirAll(store, 0o755) != nil {
			return 2
		}
		var args []string
		for _, l := range lanes {
			args = append(args, "--lane", at(l+"-capture")+","+at(l+"-install"))
		}
		return runTo(at(tag+".out"), at(tag+".err"), assemble, append(args, store)...)
	}
	out := func(tag string) []string { return fileLines(at(tag + ".out")) }
	said := func(tag, literal string) bool {
		return anyContains(out(tag), literal) || anyContains(fileLines(at(tag+".err")), literal)
	}

	// A case: run the assembler over some lanes and compare the exit code and one
	// literal. ⛔ Both, because several refusals here share an exit code and a
	// case asserting the code alone passes when a different refusal fired.
	caseIs := func(want int, needle, name, tag string, lanes ...string) {
		if laneErr != nil {
			return
		}
		rc := assembleInto(tag, lanes...)
		if rc != want {
			h.bad(fmt.Sprintf("%s (exit %d, expected %d)", name, rc, want))
			note(head(fileLines(at(tag+".err")), 4))
			return
		}
		if needle != "" && !said(tag, needle) {
			h.bad(fmt.Sprintf("%s (exit %d, and it did not say '%s')", name, want, needle))
			note(head(fileLines(at(tag+".err")), 4))
			return
		}
		h.ok(name)
	}
	check := func(ok bool, name string, detail ...string) {
		if ok {
			h.ok(name)
			return
		}
		h.bad(name)
		note(detail)
	}
	// recordHas is `grep -q LITERAL "$(find DIR -name NAME -type f | head -1)"`.
	recordHas := func(dir, name, literal string) bool {
		b, err := os.ReadFile(firstFound(findFiles(dir, name)))
		return err == nil && bytes.Contains(b, []byte(literal))
	}
	publishable := regexp.MustCompile(`^[[:space:]]*publishable$`)

	const gh = "https://api.example.invalid/repos/fixture-owner/fixture-client/releases"
	const refs = "https://git.example.invalid/fixture-owner/fixture-client/refs"
	commit := sha256Hex("fixture-commit")[:40]
	digestA := sha256Hex("build-a")
	digestB := sha256Hex("build-b")
	const peerA = "-XX0000-aaaaaaaaaaaa"
	const peerB = "-XX0000-bbbbbbbbbbbb"
	// ⚠ A second connection's tail per lane. Same eight-byte prefix, different
	// twelve bytes, which is what a per-connection tail looks like on the wire.
	const peerC = "-XX0000-cccccccccccc"
	const peerD = "-XX0000-dddddddddddd"

	goodRelease := func() { lane("good-release", "release", gh, "", digestA, "1.2.3", peerA, peerA) }
	goodSource := func() { lane("good-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA) }

	// -- ⭐ THE CONTROL, AND IT IS THE HALF THAT MATTERS MOST ---------------------
	//
	// Every refusal below passes equally over an assembler that refuses
	// everything. This is the pair that has to be accepted: two routes, two
	// resolvers, two connectors, one version, one identity on the wire.
	goodRelease()
	goodSource()
	if laneErr != nil {
		return verdict{}, laneErr
	}
	caseIs(0, "", "two routes with independent resolvers assemble into records", "good", "good-release", "good-source")
	check(said("good", "build_equivalent"), "a pair whose observations agree reaches build_equivalent",
		head(out("good"), 6)...)

	// -- ⛔ A CONNECTOR'S VERSION IS MEASURED, NEVER WRITTEN HERE ------------------
	//
	// Both were `0.0.0` in every record until 2026-10-14, for the second
	// connector a value nobody measured. The attestation's own versions are what
	// a record says. ⚠ The shell half asked Python; this reads the same keys.
	rec := firstFound(findFiles(filepath.Join(work, "store-good", "profiles"), "cap-release.json"))
	versionsOK, versionsSeen := false, ""
	if b, err := os.ReadFile(rec); err == nil {
		var doc struct {
			Capture struct {
				Connectors []struct {
					ID      string `json:"id"`
					Version any    `json:"version"`
				} `json:"connectors"`
			} `json:"capture"`
		}
		if json.Unmarshal(b, &doc) == nil {
			v := map[string]any{}
			for _, c := range doc.Capture.Connectors {
				v[c.ID] = c.Version
			}
			versionsOK = len(v) == 2 && v["bit-ids-probe"] == "0.0.0" && v["fixture-oracle"] == "9.8.7"
			seen, _ := json.Marshal(doc.Capture.Connectors)
			versionsSeen = string(seen)
		}
	}
	if rec == "" {
		rec = "none"
	}
	check(versionsOK, "versions  each connector's version is the one its attestation carries",
		fmt.Sprintf("record %s: %s", rec, versionsSeen))
	for _, d := range []string{"observer_version", "connector_version"} {
		drop = d
		lane("drop-release", "release", gh, "", digestA, "1.2.3", peerA, peerA)
		drop = ""
		lane("drop-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
		caseIs(1, d, "a lane whose attestation records no "+d+" is refused, naming it",
			"drop-"+d, "drop-release", "drop-source")
	}

	// -- ⭐ AND WHAT IT WROTE IS A CORPUS, WITH A RUN BESIDE EVERY RECORD ----------
	//
	// ⛔ `E-CRP-01` refused every store this assembler wrote until 2026-10-14,
	// because no step of the capture path wrote a run manifest.
	check(said("good", "corpus: valid, 2 record(s)"),
		"corpus    a two-lane assembly is a valid corpus, with a run beside each record", tail(out("good"), 8)...)

	// ⭐ THE CONTROL: AN INDEPENDENT READER. `validate-run` reads each manifest
	// and its record off the disk and binds them, so a manifest the assembler
	// judged by its own lights is judged again by the one tool `SCHEMA-02` names.
	// ⚠ The capture is the LAST `"capture": "..."` on the first line carrying
	// one, which is what the shell half's greedy `sed` took.
	captureOf := regexp.MustCompile(`.*"capture": *"([^"]*)".*`)
	pairs, bound := 0, 0
	for _, manifest := range findFiles(filepath.Join(work, "store-good"), "manifest.json") {
		pairs++
		capture := ""
		for _, l := range fileLines(manifest) {
			if m := captureOf.FindStringSubmatch(l); m != nil {
				capture = m[1]
				break
			}
		}
		profile := firstFound(findFiles(filepath.Join(work, "store-good", "profiles"), capture+".json"))
		if profile != "" && runTo(at("null"), at("null"), validateRun, manifest, profile) == 0 {
			bound++
		}
	}
	if pairs == 2 && bound == 2 {
		h.ok(fmt.Sprintf("corpus    validate-run binds each manifest to its record, read off the disk (%d)", bound))
	} else {
		h.bad(fmt.Sprintf("corpus    validate-run binds each manifest to its record, read off the disk (%d of %d)", bound, pairs))
	}

	// ⛔ AND A MANIFEST NO ARTIFACT SUPPORTS IS NOT WRITTEN. One case per
	// producer the manifest reads, each naming what was missing; the record is
	// still written, and the run is refused.
	for _, d := range []string{"host-claim", "uptime_at_finish", "installed_binary_bytes"} {
		drop = d
		lane("gap-release", "release", gh, "", digestA, "1.2.3", peerA, peerA)
		drop = ""
		lane("gap-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
		caseIs(1, d, "a lane with no "+d+" gets no run manifest, and the run says why",
			"gap-"+d, "gap-release", "gap-source")
	}
	runner = "self-hosted"
	lane("gap-release", "release", gh, "", digestA, "1.2.3", peerA, peerA)
	runner = ""
	lane("gap-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
	caseIs(1, "runner_environment=self-hosted",
		"a host nothing says is destroyed after the job gets no run manifest",
		"gap-runner", "gap-release", "gap-source")

	// ⛔ RUN 37'S ORDER, AS DATA: the source route finished installing after the
	// release lane's capture began. The barrier stops that on a runner; here the
	// manifest is what must refuse it, because its phases would overlap.
	lane("late-release", "release", gh, "", digestA, "1.2.3", peerA, peerA)
	installed = "2026-09-09T15:08:40Z"
	lane("late-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
	installed = ""
	caseIs(1, "E-MAN-06", "a capture that began before every route had installed gets no run manifest",
		"late", "late-release", "late-source")

	// ⛔ AND THE PAIR PUBLISHES. These two lanes install DIFFERENT BYTES, which
	// is every release-against-source pair there is; the other route's capture
	// is what settles it, and it is a DIFFERENT RECORD, so the gate is asked
	// with the store. ⚠ Counted on a whole line rather than with a substring,
	// because `provisional, not publishable` contains the word.
	check(countMatching(out("good"), publishable) == 2,
		"a byte-different pair settled by the other route's capture publishes", head(out("good"), 12)...)
	// ⛔ TWO RECORDS, NOT ONE. ⚠ UNDER `profiles/`, not every `.json` in the
	// store: the transcripts it copies in are `.json` too.
	check(len(findFiles(filepath.Join(work, "store-good", "profiles"), "*.json")) == 2,
		"one record per lane is written, over the same two routes")
	check(len(findFiles(filepath.Join(work, "store-good"), "version.txt")) >= 1,
		"the process output the installed version cites is copied into the store")

	// -- ⛔ AND EVERY ARTIFACT A RECORD CITES IS CARRIED, WHICH IT WAS NOT ---------
	//
	// ⛔ A record's evidence list is built over ALL the lanes, and the assembler
	// copied only ITS OWN lane's files under its evidence root. ⚠ `E-CRP-03`
	// cannot fire here, so the assembler names any citation it did not carry.
	if said("good", "cited and not carried") {
		h.bad("evidence  a record cites an artifact the store does not carry")
		note(grepF(out("good"), "cited and not carried"))
	} else {
		h.ok("evidence  every artifact a record cites is carried in the store")
	}

	// ⭐ THE CONTROL, AND IT IS STRUCTURAL RATHER THAN A COUNT. A record that got
	// only its own lane's artifacts has ONE route directory under its evidence
	// root; a self-contained one has every route the record's list names.
	captures, whole := 0, 0
	roots, _ := filepath.Glob(filepath.Join(work, "store-good", "raw", "v1", "*", "*", "*", "*", "*", "*"))
	for _, root := range roots {
		if !isDir(root) {
			continue
		}
		captures++
		entries, _ := os.ReadDir(root)
		dirs := 0
		for _, e := range entries {
			if e.IsDir() {
				dirs++
			}
		}
		if dirs >= 2 {
			whole++
		}
	}
	if captures >= 2 && whole == captures {
		h.ok("evidence  each record carries both routes' artifacts, not just its own")
	} else {
		h.bad(fmt.Sprintf("evidence  %d of %d record(s) carry every route's artifacts", whole, captures))
	}

	// -- ⛔ AND THE ONE THAT REFUTES THE WORK ORDER -------------------------------
	//
	// ⛔ A PEER ID CARRIES A PER-CONNECTION RANDOM TAIL, SO TWO CAPTURES OF ONE
	// BUILD NECESSARILY DISAGREE ON IT, and `classify_across` reads a
	// disagreement on any field both records measured as `divergent`.
	lane("vary-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB)
	caseIs(0, "divergent", "two captures whose peer IDs differ are divergent rather than build_equivalent",
		"vary", "good-release", "vary-source")
	// ⛔ AND THE DIVERGENCE MUST REFUSE PUBLICATION, as a conflict and not as an
	// absence: `E-PUB-03` rather than `E-PUB-04`.
	check(said("vary", "provisional, not publishable") && said("vary", "E-PUB-03") && !said("vary", "E-PUB-04"),
		"a pair the store shows diverging is refused as a conflict, not an absence", head(out("vary"), 12)...)

	// -- ⭐ AND THE SAMPLES THAT TURN THAT DIVERGENCE INTO AN AGREEMENT -----------
	//
	// ⭐ A build that connects TWICE puts two tails in the transcript, and
	// `sampling::field_state` reads the prefix that held and the bytes that
	// moved: both lanes describe ONE pattern. ⚠ All four tails differ; what
	// agrees is the shape, not the bytes.
	lane("samp-release", "release", gh, "", digestA, "1.2.3", peerA, peerA, peerC)
	lane("samp-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB, peerD)
	caseIs(0, "build_equivalent", "two connections per lane make a per-connection tail a pattern both lanes share",
		"samp", "samp-release", "samp-source")
	check(countMatching(out("samp"), publishable) == 2,
		"and the pair that was divergent on one sample each publishes", head(out("samp"), 14)...)
	// ⛔ AND THE STATE IS `patterned`, NOT `constant`.
	check(recordHas(filepath.Join(work, "store-samp", "profiles"), "*.json", `"kind": "patterned"`),
		"the record states the pattern rather than one connection's bytes")
	// ⚠ AND ONE CONNECTION IS STILL `constant` WITH ONE SAMPLE, which is what
	// says this changed the sampling and not the default.
	check(recordHas(filepath.Join(work, "store-good", "profiles"), "*.json", `"kind": "constant"`),
		"a lane that connected once still states a constant")

	// -- ⛔ AND WHICH LIFETIME THE TAIL HAS, WHICH ONE SESSION CANNOT ANSWER -------
	//
	// ⭐ THE SESSION IS READ OUT OF THE ANNOUNCE rather than out of the
	// attestation: the second announce says `event=started`.
	lane("sess-release", "release", gh, "", digestA, "1.2.3", peerA, peerA, peerC, "started")
	lane("sess-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB, peerD, "started")
	caseIs(0, "build_equivalent", "two sessions per lane make a per-session tail a pattern both lanes share",
		"sess", "sess-release", "sess-source")
	sessRecord, _ := os.ReadFile(firstFound(findFiles(filepath.Join(work, "store-sess", "profiles"), "*.json")))
	check(bytes.Contains(sessRecord, []byte(`"kind": "patterned"`)),
		"a tail that moves across sessions is a pattern, as one that moves across connections is")

	// ⛔ AND THE RECORD CANNOT SAY WHICH OF THE TWO IT IS. `field_state` discards
	// the `Lifetime` `sampling::classify` computes. ⚠ This case keeps that gap
	// measured rather than remembered: it goes red the day the schema grows such
	// a field with nothing filling it.
	if bytes.Contains(sessRecord, []byte(`"lifetime"`)) {
		h.bad("the record carries no lifetime, and this says so rather than assuming it")
	} else {
		h.ok("the record carries no lifetime: classify computes one and field_state drops it")
	}

	// ⛔ AND A REQUEST THAT IS NOT AN ANNOUNCE IS NOT A SAMPLE OF ONE. ⚠ This
	// case exists because the rule SURVIVED a mutation pass.
	lane("scrape-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB, peerD)
	if laneErr == nil {
		if err := transcript(filepath.Join(work, "scrape-source-capture", "capture", "bundle", "tracker-http.transcript.json"),
			"tracker-http", announceHex(peerB, ""), scrapeHex, announceHex(peerD, "")); err != nil {
			return verdict{}, errCannotRun("cannot write the scrape transcript")
		}
	}
	caseIs(0, "build_equivalent", "a scrape between two announces is passed over rather than refused",
		"scrape", "samp-release", "scrape-source")
	// ⛔ AND IT IS PASSED OVER RATHER THAN COUNTED. Three segments, two announces.
	scrapeRecord := firstFound(findFiles(filepath.Join(work, "store-scrape", "profiles"), "cap-source.json"))
	scrapeBytes, _ := os.ReadFile(scrapeRecord)
	check(bytes.Contains(scrapeBytes, []byte(`"samples": 2`)), "the scrape is not counted as a sample of an announce field",
		head(regexp.MustCompile(`"samples": [0-9]*`).FindAllString(string(scrapeBytes), -1), 6)...)

	// -- The refusals ------------------------------------------------------------

	// ⛔ ONE CONNECTOR IS REFUSED AT THE VALIDITY GATE, NOT AT PUBLICATION.
	lane("lonely-source", "source", refs, commit, digestB, "1.2.3", peerA)
	lane("lonely-release", "release", gh, "", digestA, "1.2.3", peerA)
	caseIs(1, "E-CAP-01", "a capture declaring one connector cannot become a record",
		"lonely", "lonely-release", "lonely-source")

	// ⛔ AND A DECLARED CONNECTOR THAT REPORTS NOTHING ABOUT A FIELD IS REFUSED
	// rather than recorded as silent.
	lane("partial-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
	if laneErr == nil {
		report := filepath.Join(work, "partial-source-capture", "capture", "bundle", "connector", "fixture-oracle.txt")
		if rewriteLines(report, at("partial.txt"), func(l string) (string, bool) {
			return l, !strings.HasPrefix(l, "peer_wire/reserved=")
		}) != nil || copyFile(at("partial.txt"), report) != nil {
			return verdict{}, errCannotRun("cannot plant the partial report")
		}
	}
	caseIs(1, "reports nothing about peer_wire/reserved", "a declared connector silent on a field is refused",
		"partial", "good-release", "partial-source")

	// -- ⭐ AND THE CASE `OBS-07`'s PROVE ASKS FOR --------------------------------
	//
	// ⛔ A DELIBERATELY ALTERED FIELD MUST PRODUCE A CONNECTOR CONFLICT. This
	// lane's connector reports a peer ID the transcript does not carry, on both
	// surfaces. ⚠ IT IS A VALID RECORD: `publishable` is what refuses it.
	lane("conflict-source", "source", refs, commit, digestB, "1.2.3", peerA, peerB)
	caseIs(0, "E-PUB-01", "a connector that read different bytes is recorded as a conflict",
		"conflict", "good-release", "conflict-source")
	check(said("conflict", "provisional, not publishable"), "a record carrying a connector conflict is not publishable",
		head(out("conflict"), 8)...)
	// ⛔ AND IT FIRES ON THE ALTERED FIELDS AND ONLY ON THEM.
	conflicted := countContaining(out("conflict"), "E-PUB-01")
	if conflicted == 2 {
		h.ok("the conflict is on the two altered fields and not on the two agreed ones")
	} else {
		h.bad(fmt.Sprintf("the conflict is on the two altered fields and not on the two agreed ones (%d rows)", conflicted))
		note(grepF(out("conflict"), "E-PUB-01"))
	}

	// -- ⛔ AND EVERY SAMPLE IS CORROBORATED, NOT THE FIRST ------------------------
	//
	// ⛔ This lane's connector agrees on every first sample and reads a different
	// peer ID out of the SECOND announce only: an assembler comparing first
	// samples calls it exact and publishes it.
	lane("late-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB, peerD)
	if laneErr != nil {
		return verdict{}, laneErr
	}
	lateReport := filepath.Join(work, "late-source-capture", "capture", "bundle", "connector", "fixture-oracle.txt")
	lateFrom := "tracker_http/peer_id=" + hexOf(peerB) + "," + hexOf(peerD)
	lateTo := "tracker_http/peer_id=" + hexOf(peerB) + "," + hexOf(peerC)
	_ = rewriteLines(lateReport, at("late.txt"), func(l string) (string, bool) {
		if l == lateFrom {
			return lateTo, true
		}
		return l, true
	})
	if sameBytes(at("late.txt"), lateReport) {
		h.bad("the second-sample plant applied")
	} else {
		h.ok("the second-sample plant applied")
		_ = copyFile(at("late.txt"), lateReport)
		caseIs(0, "E-PUB-01", "a connector that parts from the observer on the second sample is a conflict",
			"late", "samp-release", "late-source")
		if countContaining(out("late"), "E-PUB-01") == 1 {
			h.ok("the late conflict is on the one field it was planted in")
		} else {
			h.bad("the late conflict is on the one field it was planted in")
			note(grepF(out("late"), "E-PUB-01"))
		}
		// ⭐ AND THE RECORD CARRIES THE SAMPLE THAT PARTED, with both of its values.
		lateRecord, _ := os.ReadFile(firstFound(findFiles(filepath.Join(work, "store-late", "profiles"), "cap-source.json")))
		check(bytes.Contains(lateRecord, []byte("at sample 2 of 2")) &&
			bytes.Contains(lateRecord, []byte(`"`+hexOf(peerC)+`"`)) &&
			bytes.Contains(lateRecord, []byte(`"`+hexOf(peerD)+`"`)),
			"the record names sample 2 and carries what each reader saw there")
	}

	// ⛔ AND A REPORT THAT COUNTED FEWER SAMPLES DISAGREES ABOUT WHAT WAS SENT.
	lane("short-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB, peerD)
	if laneErr != nil {
		return verdict{}, laneErr
	}
	shortReport := filepath.Join(work, "short-source-capture", "capture", "bundle", "connector", "fixture-oracle.txt")
	shortAgent := regexp.MustCompile(`^tracker_http/user_agent=([0-9a-f]*),.*$`)
	_ = rewriteLines(shortReport, at("short.txt"), func(l string) (string, bool) {
		return shortAgent.ReplaceAllString(l, "tracker_http/user_agent=$1"), true
	})
	if sameBytes(at("short.txt"), shortReport) {
		h.bad("the one-sample plant applied")
	} else {
		h.ok("the one-sample plant applied")
		_ = copyFile(at("short.txt"), shortReport)
		caseIs(0, "E-PUB-01", "a connector that read fewer samples than the observer is a conflict",
			"shortrep", "samp-release", "short-source")
		check(recordHas(filepath.Join(work, "store-shortrep", "profiles"), "cap-source.json",
			"read 2 and 1 samples, and part at sample 2"),
			"the record says how many samples each reader found")
	}

	// ⛔ AND A DIFFERENCE THE RECORD CANNOT SHOW IS REFUSED BY NAME. One
	// announce, and a connector that reports a second, absent, `User-Agent`.
	lane("extra-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
	if laneErr != nil {
		return verdict{}, laneErr
	}
	extraReport := filepath.Join(work, "extra-source-capture", "capture", "bundle", "connector", "fixture-oracle.txt")
	extraAgent := regexp.MustCompile(`^(tracker_http/user_agent=[0-9a-f]*)$`)
	_ = rewriteLines(extraReport, at("extra.txt"), func(l string) (string, bool) {
		return extraAgent.ReplaceAllString(l, "$1,absent"), true
	})
	if sameBytes(at("extra.txt"), extraReport) {
		h.bad("the extra-absent-sample plant applied")
	} else {
		h.ok("the extra-absent-sample plant applied")
		_ = copyFile(at("extra.txt"), extraReport)
		caseIs(1, "only in how many samples each found", "a count-only difference no record can show is refused by name",
			"extra", "good-release", "extra-source")
	}

	// ⛔ AND A LIST WITH A HOLE IN IT IS REFUSED, NOT CLOSED UP.
	lane("hole-source", "source", refs, commit, digestB, "1.2.3", peerB, peerB, peerD)
	if laneErr != nil {
		return verdict{}, laneErr
	}
	holeReport := filepath.Join(work, "hole-source-capture", "capture", "bundle", "connector", "fixture-oracle.txt")
	holeReserved := regexp.MustCompile(`^peer_wire/reserved=([0-9a-f]*),`)
	_ = rewriteLines(holeReport, at("hole.txt"), func(l string) (string, bool) {
		if loc := holeReserved.FindStringSubmatchIndex(l); loc != nil {
			return "peer_wire/reserved=" + l[loc[2]:loc[3]] + ",," + l[loc[1]:], true
		}
		return l, true
	})
	if sameBytes(at("hole.txt"), holeReport) {
		h.bad("the empty-sample plant applied")
	} else {
		h.ok("the empty-sample plant applied")
		_ = copyFile(at("hole.txt"), holeReport)
		caseIs(1, "sample 2 is empty", "a connector report with an empty sample is refused",
			"hole", "samp-release", "hole-source")
	}

	// -- ⛔ A ROUTE THAT ACQUIRED NOTHING IS NOT A SECOND ROUTE --------------------
	//
	// ⛔ `ACQ-03`'s SHARPEST RESIDUAL, measured: an `apt-get install` that
	// installed nothing reaches `byte_identical` by acquiring nothing. ⚠ The third
	// value is its own case, because reading an unknown one as `no` would refuse
	// it for a reason that is not true.
	acquired = "no"
	lane("idle-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
	acquired = "yes"
	caseIs(1, "acquired=no", "a route that ran and installed nothing is refused as a second route",
		"idle", "good-release", "idle-source")

	acquired = "maybe"
	lane("odd-source", "source", refs, commit, digestB, "1.2.3", peerA, peerA)
	acquired = "yes"
	caseIs(1, "neither yes nor no", "an install record whose acquired field is not yes or no is refused",
		"odd", "good-release", "odd-source")

	// ⛔ ONE LISTING, TWO LANES. This is run 14's shape, and `E-ACQ-07` calls it
	// one route.
	lane("same-source", "source", gh, commit, digestB, "1.2.3", peerA, peerA)
	caseIs(1, "E-ACQ-07", "two lanes that read one listing are refused as one route",
		"same", "good-release", "same-source")

	lane("no-commit", "source", refs, "", digestB, "1.2.3", peerA, peerA)
	caseIs(1, "E-ACQ-06", "a source route with no commit is refused", "nocommit", "good-release", "no-commit")

	lane("short-commit", "source", refs, "abbrev1", digestB, "1.2.3", peerA, peerA)
	caseIs(1, "", "a source route with an abbreviated commit is refused", "short", "good-release", "short-commit")

	// ⛔ TWO VERSIONS ARE TWO BUILDS.
	lane("other-version", "source", refs, commit, digestB, "1.2.4", peerA, peerA)
	caseIs(1, "must resolve one stable version", "two lanes reporting two versions are refused",
		"versions", "good-release", "other-version")

	// ⛔ THE PROCESS OUTPUT LIVES IN THE INSTALL ARTIFACT, and a lane handed only
	// its capture bundle cannot produce a record at all: could-not-run.
	_ = os.Remove(filepath.Join(work, "good-source-install", "install-source", "version.err"))
	caseIs(2, "E-ACQ-10", "a lane with no process output for its version cannot run",
		"novers", "good-release", "good-source")
	goodSource()

	// ⛔ THE THREE FIELDS A STORE PATH IS DERIVED FROM ARE DERIVED, NOT WRITTEN
	// IN. `platform`, `arch` and `package` are in `StoreKey`.
	sourceInstall := filepath.Join(work, "good-source-capture", "install-source.txt")
	if laneErr == nil {
		if rewriteLines(sourceInstall, at("nopkg.txt"), func(l string) (string, bool) {
			return l, !strings.HasPrefix(l, "package=")
		}) != nil || copyFile(at("nopkg.txt"), sourceInstall) != nil {
			return verdict{}, errCannotRun("cannot plant the install record with no package")
		}
	}
	caseIs(1, "carries no `package`", "an attestation with no package cannot be filed, because the path is derived from it",
		"nopkg", "good-release", "good-source")
	goodSource()

	// ⛔ AND A FIXTURE CAPTURE IS NOT A MEASUREMENT.
	sourceAttestation := filepath.Join(work, "good-source-capture", "capture", "attestation.txt")
	if laneErr == nil {
		if rewriteLines(sourceAttestation, at("fixture.txt"), func(l string) (string, bool) {
			if l == "kind=client" {
				return "kind=fixture", true
			}
			return l, true
		}) != nil || copyFile(at("fixture.txt"), sourceAttestation) != nil {
			return verdict{}, errCannotRun("cannot plant the fixture attestation")
		}
	}
	caseIs(1, "which is not a target kind", "a fixture capture is refused rather than recorded as a build",
		"fixturekind", "good-release", "good-source")
	goodSource()

	// ⛔ E-ACQ-08 HAD NO CASE UNTIL A GUARD-MUTATION PASS ASKED FOR ONE.
	lane("twin-release", "release", refs, "", digestB, "1.2.3", peerA, peerA)
	caseIs(1, "E-ACQ-08", "two lanes delivered the same way are refused as one route",
		"twin", "good-release", "twin-release")

	// ⛔ AND A DOCUMENT WITH NO BANNER IS NOT ONE OF THIS PROJECT'S.
	if laneErr == nil {
		if rewriteLines(sourceInstall, at("nobanner.txt"), func(l string) (string, bool) {
			return l, l != "bit-ids/install-record/1"
		}) != nil || copyFile(at("nobanner.txt"), sourceInstall) != nil {
			return verdict{}, errCannotRun("cannot plant the install record with no banner")
		}
	}
	caseIs(2, "bit-ids document banner", "a record with no schema banner cannot run",
		"nobanner", "good-release", "good-source")
	goodSource()
	if laneErr != nil {
		return verdict{}, laneErr
	}

	// ⛔ ONE LANE IS ONE ROUTE. `E-ACQ-01` refuses a record with one, so the
	// assembler refuses before reading anything.
	if rc := assembleInto("one", "good-release"); rc == 2 && said("one", "E-ACQ-01") {
		h.ok("a single lane is refused before anything is read")
	} else {
		h.bad(fmt.Sprintf("a single lane is refused before anything is read (exit %d)", rc))
	}

	// -- ⭐ THE CONVERSATION AFTER A HANDSHAKE ----------------------------------------
	//
	// ⛔ Transmission 4.1.3 sent an unchoke, `0000000101`, after its handshake on
	// each connection of `capture-client` run 38667886527, and the pair was
	// refused because every segment was read as a handshake.
	peerFile := func(l string) string {
		return filepath.Join(work, l+"-capture", "capture", "bundle", "peer-wire-dialled.transcript.json")
	}
	plantPeer := func(l string, segs ...string) error {
		if err := peerSegments(peerFile(l), segs...); err != nil {
			return errCannotRun("cannot write the " + l + " peer transcript")
		}
		return nil
	}
	for _, l := range []string{"good-release", "good-source"} {
		if err := plantPeer(l, "1:"+handshakeHex(peerA), "1:0000000101"); err != nil {
			return verdict{}, err
		}
	}
	caseIs(0, "", "a message after the handshake on the same connection is the conversation, not a refusal",
		"afterhs", "good-release", "good-source")
	if err := plantPeer("good-source", "1:0000000101", "1:"+handshakeHex(peerA)); err != nil {
		return verdict{}, err
	}
	caseIs(1, "the peer handshake", "a connection whose first segment is not a handshake is refused",
		"nothsfirst", "good-release", "good-source")
	if err := plantPeer("good-source", "1:"+handshakeHex(peerA), "2:0000000101"); err != nil {
		return verdict{}, err
	}
	caseIs(1, "the peer handshake", "a second connection opening with something else is refused",
		"nothssecond", "good-release", "good-source")
	goodRelease()
	goodSource()

	// -- ⭐ THE PEER SURFACE IN EVERY SESSION -----------------------------------------
	//
	// ⛔ `client-capture` dials after every session's opening announce since
	// 2026-10-15, and a connection belongs to the session whose announce opened
	// last before it. Here three sessions each announce their own peer ID and
	// answer two connections with it, on one clock across both transcripts.
	const s0 = "-XX0000-dddddddddddd"
	const s1 = "-XX0000-eeeeeeeeeeee"
	const s2 = "-XX0000-ffffffffffff"
	if laneErr != nil {
		return verdict{}, laneErr
	}
	for _, l := range []string{"good-release", "good-source"} {
		bundle := filepath.Join(work, l+"-capture", "capture", "bundle")
		agentHex := hexOf(agent)
		res := reservedHex
		body := fmt.Sprintf("tracker_http/peer_id=%s,%s,%s\n", hexOf(s0), hexOf(s1), hexOf(s2)) +
			fmt.Sprintf("tracker_http/user_agent=%s,%s,%s\n", agentHex, agentHex, agentHex) +
			fmt.Sprintf("peer_wire/peer_id=%s,%s,%s,%s,%s,%s\n", hexOf(s0), hexOf(s0), hexOf(s1), hexOf(s1), hexOf(s2), hexOf(s2)) +
			fmt.Sprintf("peer_wire/reserved=%s,%s,%s,%s,%s,%s\n", res, res, res, res, res, res)
		if segmentsAt(filepath.Join(bundle, "tracker-http.transcript.json"), "tracker-http", []string{
			"1:1000:" + announceHex(s0, "started"), "2:5000:" + announceHex(s1, "started"),
			"3:9000:" + announceHex(s2, "started"),
		}) != nil || segmentsAt(filepath.Join(bundle, "peer-wire-dialled.transcript.json"), "peer-wire-dialled", []string{
			"4:1200:" + handshakeHex(s0), "5:1300:" + handshakeHex(s0),
			"6:5200:" + handshakeHex(s1), "7:5300:" + handshakeHex(s1),
			"8:9200:" + handshakeHex(s2), "9:9300:" + handshakeHex(s2),
		}) != nil || os.WriteFile(filepath.Join(bundle, "connector", "fixture-oracle.txt"), []byte(body), 0o644) != nil {
			return verdict{}, errCannotRun("cannot write the " + l + " session plan")
		}
	}
	caseIs(0, "", "three sessions, each dialled twice, assemble", "persession", "good-release", "good-source")
	// ⚠ The plan is what tells the readings apart: three sessions of two
	// connections read this way, and one session of six read the old way. It is
	// every manifest joined with its spaces and newlines removed, as `tr` did.
	var joined strings.Builder
	for _, m := range findFiles(filepath.Join(work, "store-persession"), "manifest.json") {
		b, _ := os.ReadFile(m)
		joined.Write(b)
	}
	squeezed := strings.NewReplacer(" ", "", "\n", "").Replace(joined.String())
	planSet := map[string]bool{}
	for _, p := range regexp.MustCompile(`"sessions":[0-9]*,"torrents":1,"connections":[0-9]*`).FindAllString(squeezed, -1) {
		planSet[p] = true
	}
	var plans []string
	for p := range planSet {
		plans = append(plans, p)
	}
	sort.Strings(plans)
	plan := strings.Join(plans, "\n")
	if plan == `"sessions":3,"torrents":1,"connections":2` {
		h.ok("and each connection is placed in its own session: three sessions of two")
	} else {
		h.bad("and each connection is placed in its own session: three sessions of two (got [" + plan + "])")
	}
	goodRelease()
	goodSource()
	if laneErr != nil {
		return verdict{}, laneErr
	}

	// ⛔ A TRANSCRIPT THIS PROJECT DID NOT WRITE IS NOT EVIDENCE. A document with
	// uppercase hex - which digests differently and would have been written as a
	// different artifact - is refused rather than decoded.
	sourcePeer := peerFile("good-source")
	_ = rewriteLines(sourcePeer, at("bad-transcript.json"), func(l string) (string, bool) {
		return strings.Replace(l, `"bytes": "13`, `"bytes": "13A`, 1), true
	})
	if sameBytes(at("bad-transcript.json"), sourcePeer) {
		h.bad("the transcript plant applied")
	} else {
		h.ok("the transcript plant applied")
		_ = copyFile(at("bad-transcript.json"), sourcePeer)
		caseIs(1, "transcript", "a transcript this project's writer would not emit is refused",
			"badtrans", "good-release", "good-source")
		goodSource()
		if laneErr != nil {
			return verdict{}, laneErr
		}
	}

	// -- ⭐ A DISPATCH'S ARTIFACTS, ALL THE WAY TO A BUNDLE ---------------------------
	//
	// ⛔ The good pair again, under the names a dispatch uploads them as, with the
	// `resolution.json` each resolver writes beside its text record. `CI-09`.
	// ⭐ It finds the repository from where it is run, hence the root below.
	const resolutionJSON = `{"schema": "bit-ids/resolution/2", "target": "fixture-client", "scheme": {"tag_prefix": "v", "min_components": 3, "max_components": 3, "floor_major": null}}` + "\n"
	artifacts := func(dir, ra string, routes ...string) error {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for _, route := range routes {
			ins := filepath.Join(dir, "install-fixture-client-"+route+"-"+ra)
			if err := copyTree(at("good-"+route+"-capture"), filepath.Join(dir, "capture-client-fixture-client-"+route+"-"+ra)); err != nil {
				return err
			}
			if err := copyTree(at("good-"+route+"-install"), ins); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Join(ins, route), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(ins, route, "resolution.json"), []byte(resolutionJSON), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
	arts := at("arts")
	bundle := at("dispatch-bundle")
	// ⛔ Unpiped: the output goes to a file and the code is read from the process.
	dispatchIs := func(want int, literal, name string) {
		_ = os.RemoveAll(bundle)
		log := at("dispatch.out")
		rc := -1
		if f, err := os.Create(log); err == nil {
			cmd := exec.Command(self, "assemble-dispatch", arts, bundle)
			cmd.Dir = r.root
			cmd.Stdout, cmd.Stderr = f, f
			rc = 0
			if err := cmd.Run(); err != nil {
				rc = -1
				if x, ok := err.(*exec.ExitError); ok {
					rc = x.ExitCode()
				}
			}
			f.Close()
		}
		lines := fileLines(log)
		_, statErr := os.Lstat(bundle)
		switch {
		case rc != want:
			h.bad(fmt.Sprintf("%s (exit %d, expected %d)", name, rc, want))
			note(tail(lines, 4))
		case !anyContains(lines, literal):
			h.bad(fmt.Sprintf("%s (exit %d, and it did not say '%s')", name, want, literal))
			note(tail(lines, 4))
		case want != 0 && statErr == nil:
			h.bad(name + " (refused, and a bundle was left behind)")
		default:
			h.ok(name)
		}
	}
	setUp := func(routes ...string) error {
		if err := artifacts(arts, "7-1", routes...); err != nil {
			return errCannotRun("cannot lay out the dispatch artifacts: " + err.Error())
		}
		return nil
	}

	if err := setUp("release", "source"); err != nil {
		return verdict{}, err
	}
	dispatchIs(0, "2 of 2 record(s) may enter a published view",
		"dispatch  a two-route dispatch's artifacts become a bundle a publisher can verify")
	// ⚠ The checksums are read by `sha256sum` where it exists and `certutil`
	// where it does not, as check-release reads its own: this binary's SHA-256
	// reading the file `assemble-release` wrote with the same crate would be the
	// writer checking itself.
	check(isRegular(filepath.Join(bundle, "MANIFEST.json")) && verifySums(bundle) &&
		len(findFiles(filepath.Join(bundle, "raw"), "manifest.json")) == 2,
		"dispatch  the bundle carries both records' run manifests and its own checksums")

	if err := setUp("release"); err != nil {
		return verdict{}, err
	}
	dispatchIs(1, "a record needs two independent routes",
		"dispatch  an adapter that captured on one route is refused, not left out")

	if err := setUp("release", "source"); err != nil {
		return verdict{}, err
	}
	_ = os.RemoveAll(filepath.Join(arts, "install-fixture-client-source-7-1"))
	dispatchIs(1, "has no install-fixture-client-source-7-1 beside it",
		"dispatch  a capture with no install artifact beside it is refused")

	if err := setUp("release", "source"); err != nil {
		return verdict{}, err
	}
	if copyTree(filepath.Join(arts, "capture-client-fixture-client-source-7-1"), filepath.Join(arts, "capture-client-fixture-client-source-7-2")) != nil ||
		copyTree(filepath.Join(arts, "install-fixture-client-source-7-1"), filepath.Join(arts, "install-fixture-client-source-7-2")) != nil {
		return verdict{}, errCannotRun("cannot lay out a second attempt")
	}
	dispatchIs(1, "from 2 runs or attempts",
		"dispatch  artifacts from two attempts of a run are refused rather than mixed")

	if err := setUp("release", "source"); err != nil {
		return verdict{}, err
	}
	_ = os.Remove(filepath.Join(arts, "install-fixture-client-source-7-1", "source", "resolution.json"))
	dispatchIs(1, "source lane has no resolution.json",
		"dispatch  a lane whose resolver wrote no scheme is refused")

	// ⭐ A PACKAGE ROUTE HAS NO RESOLVER, so its lane passes the scheme check on
	// its sibling's resolution and goes on to be assembled. ⚠ The fixture's
	// source lane renamed is not a package capture, so the assembly refuses it
	// afterwards; what the case asserts is that the refusal is no longer the
	// scheme's. Measured on capture-client run 38788416994. `CI-11`.
	if err := setUp("release", "source"); err != nil {
		return verdict{}, err
	}
	for _, kind := range []string{"capture-client", "install"} {
		from := filepath.Join(arts, kind+"-fixture-client-source-7-1")
		if os.Rename(from, filepath.Join(arts, kind+"-fixture-client-package-7-1")) != nil {
			return verdict{}, errCannotRun("cannot lay out a package lane")
		}
	}
	_ = os.RemoveAll(filepath.Join(arts, "install-fixture-client-package-7-1", "source"))
	dispatchIs(1, "fixture-client did not assemble",
		"dispatch  a package lane takes its scheme from a sibling's resolution")

	if laneErr != nil {
		return verdict{}, laneErr
	}
	v := h.report("check-assemble/1", "cases")
	v.text = notes.String() + v.text
	return v, nil
}
