//! `OBS-10`'s acceptance: one captured client run's recorded bytes normalize to
//! one digest, whichever platform compiled the observer.
//!
//! ```text
//! cargo test --workspace --locked --test cross_platform_events
//! ```
//!
//! ⛔ **"One client build" is one capture's bytes, not one binary.** A client
//! build cannot be byte-identical across operating systems, so what is held
//! fixed is the evidence: two transcripts a stock build wrote, fed to
//! [`bit_ids_probe::normalize::normalize`] - the body `assemble-capture` writes
//! a record with - as compiled on Linux and on Windows. Both CI lanes assert the
//! same committed constant, so each is checked independently and neither is
//! compared against the other's output.
//!
//! # Provenance
//!
//! ⭐ **These are measurements, not seeds.** Copied unmodified from the `data`
//! branch at `4221e28b71c9`, its tip on 2026-10-16 (the files were appended in
//! `477ade686836`), out of
//! `raw/v1/qbittorrent/5.2.4/windows/x86-64/nsis-installer/cap-package/package/`,
//! the raw evidence of the published record
//! `profiles/v1/qbittorrent/5.2.4/windows/x86-64/nsis-installer/cap-package.json`.
//! That record cites both files by these digests, as `ev-package-tracker` and
//! `ev-package-peer`:
//!
//! | file | bytes | sha256 |
//! | --- | ---: | --- |
//! | `tracker-http.transcript.json` | 7181 | `sha256:a13f7ec49a9efa6cbc7db2db9fccbc324aba62bed5224158f9803efd8b013fb4` |
//! | `peer-wire-dialled.transcript.json` | 6910 | `sha256:0067f74d475321e5b05310cfd86bc2d27aff3f3971095e1e8ca318523330ed3c` |
//!
//! ⚠ Read against rule 12 before they were committed: every address in them is
//! on loopback, the info hash is the lab's synthetic torrent's, and the rest is
//! what qBittorrent 5.2.4 put on the wire - peer IDs, its tracker `key`, its user
//! agent and its extension map. A transcript is never scrubbed, because those
//! bytes are the measurement.
//!
//! # What cannot leak into the digest
//!
//! ⛔ **The fixtures are compiled in with `include_str!`**, so no path is
//! composed at run time and no separator is read. ⛔ **`.gitattributes` marks
//! the directory `-text`**, so no checkout translates a line ending; the first
//! case checks the bytes against the record's digests, and a translated file
//! fails there rather than as a digest that merely moved. ⚠ Nothing here reads a
//! clock or a locale: the offsets in a transcript are the run's own, recorded
//! once, and the canonical form below is decimal integers, lowercase hex and
//! `serde_json`'s output, none of which consult either.

use std::fmt::Write as _;

use bit_ids::canonical::{HexBytes, Sha256Digest, Slug};
use bit_ids_probe::normalize::{Normalized, normalize};

const TRACKER: &str = include_str!("fixtures/cross-platform-events/tracker-http.transcript.json");
const PEER: &str = include_str!("fixtures/cross-platform-events/peer-wire-dialled.transcript.json");

/// The digest both lanes must produce. ⛔ Changing it is a claim that the
/// normalization changed on purpose, and the entry that changes it says why.
const DIGEST: &str = "sha256:e3e98c6647b1e35b7b0e55ad10c7fad9146b3390d4677acec5af0461dbb09038";

/// The evidence identifiers the published record gives the two files, so a
/// field here is the field the record carries.
fn evidence(suffix: &str) -> Slug {
    Slug::parse(&format!("ev-package-{suffix}")).expect("a canonical identifier")
}

fn normalized(tracker: &str, peer: &str) -> Normalized {
    normalize(tracker, peer, &evidence).expect("the published capture normalizes")
}

/// The canonical text the digest is taken over.
///
/// One line per fact, each ending in `\n`: the banner, the plan, every field as
/// the record serializes it, and every sample each field rests on, in path order
/// and then document order. ⚠ The samples are in it as well as the states,
/// because two different sample lists can tile to one state and a platform
/// difference in the session reading would hide behind it.
fn canonical(normalized: &Normalized) -> String {
    let mut out = String::from("bit-ids/normalized-events/1\n");
    let plan = &normalized.plan;
    // ⚠ `writeln!` ends a line in `\n` on every platform; it is not the host's
    // line ending.
    writeln!(
        out,
        "plan {} {} {}",
        plan.sessions, plan.torrents, plan.connections
    )
    .expect("a String takes a write");
    for field in &normalized.fields {
        let json = serde_json::to_string(field).expect("a field serializes");
        writeln!(out, "field {json}").expect("a String takes a write");
    }
    for (path, samples) in &normalized.samples {
        for sample in samples {
            writeln!(
                out,
                "sample {path} {} {} {} {}",
                sample.session, sample.torrent, sample.connection, sample.value
            )
            .expect("a String takes a write");
        }
    }
    out
}

fn digest(normalized: &Normalized) -> String {
    Sha256Digest::of(canonical(normalized).as_bytes()).to_string()
}

#[test]
fn cross_platform_events_fixtures_are_the_published_bytes() {
    for (name, text, bytes, sha256) in [
        (
            "tracker-http.transcript.json",
            TRACKER,
            7181,
            "sha256:a13f7ec49a9efa6cbc7db2db9fccbc324aba62bed5224158f9803efd8b013fb4",
        ),
        (
            "peer-wire-dialled.transcript.json",
            PEER,
            6910,
            "sha256:0067f74d475321e5b05310cfd86bc2d27aff3f3971095e1e8ca318523330ed3c",
        ),
    ] {
        // ⛔ A checkout that wrote CRLF moves both numbers, so this is the case
        // that names a line-ending defect as one.
        assert!(!text.contains('\r'), "{name}: carries a carriage return");
        assert_eq!(text.len(), bytes, "{name}: length");
        assert_eq!(
            Sha256Digest::of(text.as_bytes()).to_string(),
            sha256,
            "{name}"
        );
    }
}

#[test]
fn cross_platform_events_normalize_to_the_committed_digest() {
    let normalized = normalized(TRACKER, PEER);
    // ⚠ What the digest is OF, stated so a moved constant has a reading: three
    // sessions of two connections, four fields, six samples each - which is what
    // the published record says about this capture.
    assert_eq!(
        (
            normalized.plan.sessions.get(),
            normalized.plan.torrents.get(),
            normalized.plan.connections.get()
        ),
        (3, 1, 2)
    );
    let paths: Vec<String> = normalized
        .fields
        .iter()
        .map(|field| field.path.to_string())
        .collect();
    assert_eq!(
        paths,
        [
            "peer_wire/peer_id",
            "peer_wire/reserved",
            "tracker_http/peer_id",
            "tracker_http/user_agent"
        ]
    );
    for (path, samples) in &normalized.samples {
        assert_eq!(samples.len(), 6, "{path}");
    }
    assert_eq!(digest(&normalized), DIGEST, "{}", canonical(&normalized));
}

#[test]
fn cross_platform_events_digest_moves_with_one_recorded_byte() {
    // ⛔ THE CONTROL: a digest that ignored its input would match any constant
    // it was once given. One hex digit of the first announce's peer ID is
    // changed - `-qB5240-BEe(` becomes `-qB5240-CEe(` - and the digest must
    // differ, while the document still parses. The segment bytes are hex in
    // the document, so the needle is spelled in hex here.
    let hex = |text: &[u8]| {
        HexBytes::new(text.to_vec())
            .expect("a short value")
            .to_string()
    };
    let needle = hex(b"peer_id=-qB5240-BEe");
    assert_eq!(
        TRACKER.matches(&needle).count(),
        2,
        "both announces of session 1"
    );
    let planted = TRACKER.replacen(&needle, &hex(b"peer_id=-qB5240-CEe"), 1);
    assert_ne!(planted, TRACKER);
    assert_ne!(
        digest(&normalized(&planted, PEER)),
        digest(&normalized(TRACKER, PEER))
    );
}

#[test]
fn cross_platform_events_are_the_same_on_a_second_run() {
    // ⚠ Within one process: nothing in the normalization may depend on an
    // iteration order a hasher seeds per run.
    assert_eq!(
        canonical(&normalized(TRACKER, PEER)),
        canonical(&normalized(TRACKER, PEER))
    );
}

/// The peer transcript with the first segment the build sent on each
/// connection split into two at byte `at`, the second on the same connection
/// at the same offset, which is how the lab records a handshake written in two
/// writes.
fn split_opening_segments(peer: &str, at: usize) -> String {
    let mut out = String::new();
    let mut seen: Vec<String> = Vec::new();
    let (mut connection, mut offset, mut direction) = (String::new(), String::new(), String::new());
    for line in peer.split_inclusive('\n') {
        let trimmed = line.trim();
        let value = |prefix: &str| {
            trimmed
                .strip_prefix(prefix)
                .map(|rest| rest.trim_end_matches(',').trim_matches('"').to_owned())
        };
        if let Some(v) = value("\"connection\": ") {
            connection = v;
        } else if let Some(v) = value("\"offset_ms\": ") {
            offset = v;
        } else if let Some(v) = value("\"direction\": ") {
            direction = v;
        } else if let Some(bytes) = value("\"bytes\": ")
            && direction == "from_target"
            && !seen.contains(&connection)
            && bytes.len() > 2 * at
        {
            seen.push(connection.clone());
            let (head, tail) = bytes.split_at(2 * at);
            writeln!(out, "      \"bytes\": \"{head}\"").expect("a String takes a write");
            // ⚠ The segment boundary is pushed as it is rather than escaped
            // into a format string.
            out.push_str("    },\n    {\n");
            writeln!(out, "      \"connection\": {connection},").expect("a String takes a write");
            writeln!(out, "      \"offset_ms\": {offset},").expect("a String takes a write");
            writeln!(out, "      \"direction\": \"from_target\",").expect("a String takes a write");
            writeln!(out, "      \"bytes\": \"{tail}\"").expect("a String takes a write");
            continue;
        }
        out.push_str(line);
    }
    out
}

/// `ENGINE-02`: anacrolix/torrent v1.61.0 writes its handshake in two writes,
/// and the lab recorded it split after the protocol string on one lane and
/// after the reserved bytes on the other. ⛔ Either split normalizes exactly as
/// the whole handshake does.
#[test]
fn cross_platform_events_split_handshakes_normalize_as_whole_ones() {
    let whole = canonical(&normalized(TRACKER, PEER));
    for at in [20_usize, 28] {
        let split = split_opening_segments(PEER, at);
        // ⚠ Every from_target segment of the fixture opens its own connection,
        // so each is split and the count doubles.
        assert_eq!(
            split.matches("\"from_target\"").count(),
            2 * PEER.matches("\"from_target\"").count(),
            "every connection's opening segment is split at byte {at}"
        );
        assert_eq!(
            canonical(&normalized(TRACKER, &split)),
            whole,
            "split at byte {at}"
        );
    }
}

/// ⛔ And a handshake whose rest never arrives is still refused: the rest is
/// cut to one byte.
#[test]
fn cross_platform_events_a_handshake_cut_short_is_refused() {
    let split = split_opening_segments(PEER, 20);
    let tail = split
        .find("\"bytes\": \"0000000000100005")
        .expect("the planted tail is there");
    let end = split[tail..].find('\n').expect("a line") + tail;
    let mut cut = split.clone();
    cut.replace_range(tail..end, "\"bytes\": \"00\"");
    let error = normalize(TRACKER, &cut, &evidence).expect_err("a short handshake is refused");
    assert!(error.contains("the peer handshake"), "{error}");
}
