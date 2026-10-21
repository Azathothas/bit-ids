// transcriptpeer - whether a tracker transcript carries the peer ID the
// observer reported, read the way the bytes were sent. `CLIENT-05`.
//
// ⛔ THE GUARD `capture-client` RESTS ON, AND THE DEFECT aria2 EXPOSED IN IT.
// `capture-client` refuses a capture whose raw transcript does not hold the peer
// ID the observer reported, and it used to ask that as `bytes.Contains(doc,
// hex(peer ID))` over the transcript file. The transcript stores the HTTP
// request's bytes, in which `peer_id` is PERCENT-ENCODED, so the two hex strings
// agree only when every byte of the peer ID is URL-unreserved and travels
// verbatim. That held for every client this project had captured and is false
// for aria2: measured on `capture-client` run 39577251480, 2026-10-21, aria2
// 1.37.0 announced `A2-1-37-0-` and ten random raw bytes, as
// `peer_id=A2-1-37-0-%A8%7C%CB%02Na%3DHX%17`, and a capture with three announces
// and six peer streams, its bundle verified, was refused as one that did not
// carry its own peer ID.
//
// ⭐ So this reads the transcript as what it is: the JSON document, each
// `from_target` segment's lowercase hex decoded, the request line's query, the
// `peer_id` field percent-decoded, and that compared as bytes. ⚠ It is the
// decoding `bit-ids-wire` and the `cpython-stdlib` connector each do with their
// own code, so this is a third reading rather than a borrowed one.
//
// ⛔ `+` IS NOT A SPACE HERE. That is form encoding (`application/x-www-form-
// urlencoded`), and a peer ID is twenty opaque bytes: `+` on the wire is 0x2b.
// `url.QueryUnescape` would turn it into 0x20, which is why it is not used.
//
// `bit-check capture-client --selftest` asks these functions on any host, and
// `check-bitcheck` runs it and plants the old substring test to prove the
// selftest refuses it.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// transcriptPeerIDs answers the peer_id of every `from_target` segment of a
// transcript document that is an HTTP request carrying one, decoded, as
// lowercase hex, in document order.
//
// ⛔ A DOCUMENT THAT WILL NOT READ IS AN ERROR, never an empty answer: an empty
// answer is a refusal for a different reason, and the caller says which.
func transcriptPeerIDs(doc []byte) ([]string, error) {
	var t struct {
		Schema   string `json:"schema"`
		Segments []struct {
			Direction string `json:"direction"`
			Bytes     string `json:"bytes"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(doc, &t); err != nil {
		return nil, fmt.Errorf("the transcript is not a JSON document: %v", err)
	}
	if t.Schema != "bit-ids/transcript/1" {
		return nil, fmt.Errorf("the transcript declares schema [%s], not bit-ids/transcript/1", t.Schema)
	}
	var out []string
	for i, s := range t.Segments {
		if s.Direction != "from_target" {
			continue
		}
		// ⚠ The schema says LOWERCASE hex, and `hex.DecodeString` takes either
		// case, so the case is checked first, as the connector checks it.
		if len(s.Bytes)%2 != 0 || strings.Trim(s.Bytes, "0123456789abcdef") != "" {
			return nil, fmt.Errorf("segment %d is not an even run of lowercase hex", i)
		}
		raw, err := hex.DecodeString(s.Bytes)
		if err != nil {
			return nil, fmt.Errorf("segment %d: %v", i, err)
		}
		if id, ok := requestPeerID(raw); ok {
			out = append(out, hex.EncodeToString(id))
		}
	}
	return out, nil
}

// requestPeerID is the decoded peer_id of one HTTP request, or false when the
// bytes are not a request carrying a readable one.
//
// ⚠ The request line is the first line, `GET <target> HTTP/1.x`; the query is
// what follows the first `?` of the target. A field whose name is not exactly
// `peer_id`, or whose value holds a `%` not followed by two hex digits, is not
// read as one.
func requestPeerID(raw []byte) ([]byte, bool) {
	line, _, _ := strings.Cut(string(raw), "\n")
	parts := strings.Fields(strings.TrimSuffix(line, "\r"))
	if len(parts) != 3 || !strings.HasPrefix(strings.ToUpper(parts[2]), "HTTP/") {
		return nil, false
	}
	_, query, ok := strings.Cut(parts[1], "?")
	if !ok {
		return nil, false
	}
	query, _, _ = strings.Cut(query, "#")
	for _, field := range strings.Split(query, "&") {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		if n, ok := percentDecode(name); !ok || string(n) != "peer_id" {
			continue
		}
		return percentDecode(value)
	}
	return nil, false
}

// percentDecode is RFC 3986's percent-decoding and nothing else: `%XY` is the
// byte 0xXY in either case, every other byte is itself, and `+` stays 0x2b.
func percentDecode(s string) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			out = append(out, s[i])
			continue
		}
		if i+2 >= len(s) {
			return nil, false
		}
		b, err := hex.DecodeString(s[i+1 : i+3])
		if err != nil {
			return nil, false
		}
		out = append(out, b[0])
		i += 2
	}
	return out, true
}

// transcriptCarriesPeerID is the guard: some announce in the transcript carries
// exactly the peer ID the observer reported, `peerHex` being its lowercase hex.
func transcriptCarriesPeerID(doc []byte, peerHex string) (bool, error) {
	ids, err := transcriptPeerIDs(doc)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == peerHex {
			return true, nil
		}
	}
	return false, nil
}

// -- the selftest ---------------------------------------------------------------

// captureClientSelftest is `bit-check capture-client --selftest`: the guard
// above over transcripts written here, byte by byte, and never by the observer.
func captureClientSelftest() int {
	segment := func(dir string, payload []byte) string {
		return fmt.Sprintf(`{"connection": 1, "offset_ms": 100, "direction": "%s", "bytes": "%s"}`,
			dir, hex.EncodeToString(payload))
	}
	doc := func(segments ...string) []byte {
		return []byte(`{"schema": "bit-ids/transcript/1", "endpoint": "tracker-http", "segments": [` +
			strings.Join(segments, ", ") + "]}\n")
	}
	announce := func(escaped string) []byte {
		return []byte("GET /announce?info_hash=%aa%bb&peer_id=" + escaped +
			"&port=6881&compact=1 HTTP/1.1\r\nHost: 127.0.0.1:1\r\nUser-Agent: aria2/1.37.0\r\n\r\n")
	}
	// ⭐ aria2 1.37.0's peer ID from run 39577251480, byte for byte, and the
	// request bytes it sent for it. ⚠ Spelt as bytes, because its forty hex
	// digits are the shape `check-no-secrets --public` refuses.
	aria2Peer := hex.EncodeToString([]byte("A2-1-37-0-\xa8\x7c\xcb\x02Na=HX\x17"))
	aria2Doc := doc(segment("from_target", announce("A2-1-37-0-%A8%7C%CB%02Na%3DHX%17")),
		segment("to_target", []byte("HTTP/1.1 200 OK\r\n\r\nd8:intervali10ee")))
	ascii := "-qB5230-abcdefghijkl"
	asciiHex := hex.EncodeToString([]byte(ascii))
	plusPeer := "-XX0000-ab+defghijkl"
	spacePeer := "-XX0000-ab defghijkl"

	cases, failed := 0, 0
	check := func(label string, want bool, doc []byte, peerHex string) {
		cases++
		got, err := transcriptCarriesPeerID(doc, peerHex)
		if err != nil || got != want {
			failed++
			fmt.Fprintf(os.Stderr, "capture-client --selftest: FAILED %s (carried=%v err=%v, wanted %v)\n", label, got, err, want)
		}
	}
	refusesDoc := func(label string, doc []byte) {
		cases++
		if _, err := transcriptCarriesPeerID(doc, asciiHex); err == nil {
			failed++
			fmt.Fprintf(os.Stderr, "capture-client --selftest: FAILED %s (no error)\n", label)
		}
	}

	check("a percent-encoded raw-byte peer ID, aria2's, is found", true, aria2Doc, aria2Peer)
	check("the same escapes in lowercase are the same bytes", true,
		doc(segment("from_target", announce("A2-1-37-0-%a8%7c%cb%02Na%3dHX%17"))), aria2Peer)
	check("an unreserved-ASCII peer ID sent verbatim is still found", true,
		doc(segment("from_target", announce(ascii))), asciiHex)
	check("an unreserved-ASCII peer ID sent fully escaped is found", true,
		doc(segment("from_target", announce(pctOf(ascii)))), asciiHex)
	check("a peer ID the transcript does not carry is refused", false, aria2Doc, asciiHex)
	check("a peer ID one byte away is refused", false, aria2Doc, aria2Peer[:len(aria2Peer)-2]+"18")
	check("a literal + is the byte 0x2b", true,
		doc(segment("from_target", announce(plusPeer))), hex.EncodeToString([]byte(plusPeer)))
	check("and is never decoded as a space", false,
		doc(segment("from_target", announce(plusPeer))), hex.EncodeToString([]byte(spacePeer)))
	// ⛔ THE OLD TEST'S OTHER HOLE: it read the whole file, so a peer ID's hex in
	// the LAB's own reply, or anywhere else in the document, passed it.
	check("a peer ID only in a to_target segment is not the build's", false,
		doc(segment("from_target", announce(ascii)), segment("to_target", []byte(aria2Peer))), aria2Peer)
	check("a peer ID only in a field that is not peer_id is refused", false,
		doc(segment("from_target", []byte("GET /announce?xpeer_id="+ascii+" HTTP/1.1\r\n\r\n"))), asciiHex)
	check("a segment that is not a request is passed over", true,
		doc(segment("from_target", []byte("\x13BitTorrent protocol")), segment("from_target", announce(ascii))), asciiHex)
	check("a truncated escape is not read as a peer ID", false,
		doc(segment("from_target", announce("A2-1-37-0-%A8%7C%CB%02Na%3DHX%1"))), aria2Peer)
	refusesDoc("a transcript that is not JSON is an error, not an absence", []byte("not json\n"))
	refusesDoc("a transcript of another schema is an error",
		[]byte(`{"schema": "bit-ids/transcript/0", "segments": []}`))
	refusesDoc("an uppercase hex segment is an error",
		[]byte(`{"schema": "bit-ids/transcript/1", "segments": [{"direction": "from_target", "bytes": "`+
			strings.ToUpper(hex.EncodeToString(announce(ascii)))+`"}]}`))

	fmt.Printf(`{"schema":"capture-client-selftest/1","cases":%d,"failed":%d}`+"\n", cases, failed)
	if failed != 0 {
		return 1
	}
	return 0
}
