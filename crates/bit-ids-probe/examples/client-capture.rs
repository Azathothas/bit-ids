//! Serves a real client the torrent it is about to announce about, and records
//! what it emitted.
//!
//! ```text
//! cargo run -p bit-ids-probe --example client-capture -- <bundle> <seconds> <torrent> [peer-port]
//! ```
//!
//! ⭐ **This is the observer half of a client capture, and the difference from
//! `evidence-bundle` is the torrent.** That example generates one with a
//! hard-coded announce URL, which no client can be pointed at, so the only thing
//! that could drive it was something told the endpoint separately. Here the
//! announce URL is **this lab's own tracker endpoint**, written into a
//! `.torrent` on disk, so the driver is `<client> <torrent>` and the client
//! chooses the address out of the file. A stock build needs no argument it does
//! not already take.
//!
//! ⛔ **It never says which client announced.** `docs/capture-methodology.md`
//! lists a peer-ID table among the inputs that may seed a hypothesis and may not
//! populate the catalogue, and this prints the bytes rather than a name.
//!
//! ## The two roles, and why the peer one is dialled
//!
//! The tracker endpoint accepts, because the client is told where it is. The
//! peer endpoint **dials**, because it cannot be told: `Lab` binds port zero so
//! the operating system chooses, and `TrackerResponse` is cloned into the
//! responder while the lab is still being built, which is before any port
//! exists. A tracker answer naming this lab's peer port would have to predict
//! it.
//!
//! ⭐ So the client's own listen port is the argument instead, and the lab dials
//! it once an announce has arrived. That is `OBS-04`'s dialling role, it needs
//! no prediction, and it is the stronger measurement of the two: the side that
//! dials sends its handshake first, so the bytes that come back are the build
//! answering rather than the build opening.
//!
//! ⚠ The dial waits for an announce rather than happening at once, because a
//! client that has not yet read the torrent is not yet listening for peers on
//! it, and a refused connection would be recorded as a build that declined.
//!
//! ## What it prints
//!
//! One `key value` line per fact, because `capture-run` reads them with `awk`
//! and a shell script parsing prose is a second grammar to keep in step.

use core::fmt::Write as _;
use std::collections::BTreeMap;
use std::net::{IpAddr, Ipv4Addr, SocketAddr};
use std::time::{Duration, Instant};

use bit_ids::canonical::Slug;
use bit_ids::manifest::PhaseName;
use bit_ids::record::EvidenceKind;
use bit_ids_lab::{Bundle, Lab, SyntheticTorrent, TorrentSpec, TranscriptOf};
use bit_ids_probe::peer_wire::{ExtendedOffer, ExtensionProtocol, Offer, PeerWire};
use bit_ids_probe::{HttpTracker, TrackerResponse};

/// The name every fixture peer ID in this repository carries.
///
/// ⚠ It is what **this observer** calls itself, and never what the build under
/// measurement is called. A capture whose transcript held only this value
/// measured nothing.
///
/// ⛔ **It is the NAME rather than a whole peer ID, and it has to be.** The
/// observer varies the last four bytes per connection, so a comparison against
/// one twenty-byte value would answer *not mine* about every connection after
/// the first - which is a guard that sees one spelling of the thing it looks
/// for.
const OBSERVER_NAME: &[u8] = b"bit-ids-fixture-";

fn slug(text: &str) -> Slug {
    Slug::parse(text).expect("a canonical identifier")
}

fn hex(bytes: &[u8]) -> String {
    let mut out = String::new();
    for byte in bytes {
        write!(out, "{byte:02x}").expect("writing to a String cannot fail");
    }
    out
}

fn names(values: &[Vec<u8>]) -> String {
    values
        .iter()
        .map(|value| String::from_utf8_lossy(value).into_owned())
        .collect::<Vec<_>>()
        .join(",")
}

fn fail(message: &str) -> ! {
    eprintln!("client-capture: {message}");
    std::process::exit(1)
}

fn main() {
    let mut arguments = std::env::args().skip(1);
    let root = arguments.next().unwrap_or_else(|| "bundle".to_owned());
    let seconds: u64 = arguments
        .next()
        .and_then(|value| value.parse().ok())
        .unwrap_or(20);
    let torrent_path = arguments
        .next()
        .unwrap_or_else(|| "fixture.torrent".to_owned());
    // ⚠ Absent means "do not dial", which is a different run from a dial that
    // failed. A capture of a client with no peer port measures the tracker
    // surface alone and says so, rather than reporting a peer surface nobody
    // could reach.
    let peer_port: Option<u16> = arguments.next().and_then(|value| value.parse().ok());
    // ⛔ TWO BY DEFAULT, because one is the number that cannot describe a value
    // the build regenerates per connection. ⚠ It is an argument rather than a
    // constant so a capture can ask for more, and so a harness can ask for one
    // and get exactly what this did before.
    let dials: u32 = arguments
        .next()
        .and_then(|value| value.parse().ok())
        .filter(|count| *count > 0)
        .unwrap_or(2);

    // ⚠ What the tracker answers is a condition of the run. It offers no peer
    // at all: the peer surface is reached by dialling below, and a compact
    // peer list naming an address this lab does not serve would put a
    // connection refusal in the build's log and change what it does next.
    //
    // ⛔ THE INTERVAL IS DERIVED FROM THE DEADLINE, NOT WRITTEN HERE. It was a
    // literal 60 under a 45-second deadline, so `capture-client` run 18 never
    // had a re-announce fall due and every `tracker_http/*` field rested on one
    // sample. `TrackerResponse::within` is the one derivation and its own tests
    // pin it; a second reading of the rule here is how the two drift.
    let tracker = HttpTracker::new(TrackerResponse {
        incomplete: 1,
        ..TrackerResponse::within(Duration::from_secs(seconds))
    });

    // ⚠ Offered, and recorded as offered. A build's extension map may differ
    // with what it was shown, so the offer is part of the measurement.
    let offer = Offer {
        extension_protocol: ExtensionProtocol::Offered(ExtendedOffer {
            extensions: vec![(b"ut_metadata".to_vec(), 1), (b"ut_pex".to_vec(), 2)],
            client: Some(b"bit-ids-observer/1".to_vec()),
            request_queue: Some(250),
            metadata_size: None,
        }),
        dht: false,
        fast: false,
    };

    let mut lab = match Lab::builder()
        .deadline(Duration::from_secs(seconds))
        .stream("tracker-http", tracker.responder())
        .and_then(|builder| {
            builder.stream("peer-wire", |_connection, received: &[u8]| {
                // ⚠ The accepting side is present so a build that finds this lab by
                // some other route is still recorded. It answers nothing, because a
                // client cannot learn this port from the torrent and anything that
                // arrives here is unexplained rather than expected.
                bit_ids_lab::StreamReply::Answer {
                    consumed: received.len(),
                    send: Vec::new(),
                }
            })
        })
        .and_then(bit_ids_lab::LabBuilder::start)
    {
        Ok(lab) => lab,
        Err(error) => fail(&format!("the lab would not start: {error}")),
    };

    let announce_to = match lab.endpoint("tracker-http") {
        Some(endpoint) => endpoint.address(),
        None => fail("the lab reported no tracker-http endpoint"),
    };
    let announce_url = format!("http://{announce_to}/announce");

    // ⛔ The torrent is generated from a declared spec, so `capture.fixture` can
    // be re-derived from the record and compared against these bytes. Only the
    // announce URL varies per run, and it is the one field a client must read
    // out of the file for any of this to happen at all.
    let torrent = match SyntheticTorrent::generate(TorrentSpec {
        name: "bit-ids-client-capture".to_owned(),
        announce: Some(announce_url.clone()),
        ..TorrentSpec::default()
    }) {
        Ok(torrent) => torrent,
        Err(error) => fail(&format!("the spec describes no usable torrent: {error}")),
    };
    if let Err(error) = std::fs::write(&torrent_path, torrent.metainfo()) {
        fail(&format!(
            "the torrent could not be written to {torrent_path}: {error}"
        ));
    }

    for endpoint in lab.endpoints() {
        println!("endpoint {} {}", endpoint.name(), endpoint.address());
    }
    println!("announce-url {announce_url}");
    println!("torrent {torrent_path}");
    println!("info-hash {}", hex(torrent.info_hash()));
    println!("fixture-sha256 {}", torrent.digest());
    println!("offered-reserved {}", hex(&offer.reserved()));
    // ⚠ A condition of the run, printed where a reader of the log finds it. A
    // build that announced once under an interval longer than the deadline was
    // never asked again; one that announced once under a shorter interval
    // declined, and that is a measurement.
    println!("offered-interval {}", tracker.response().interval);
    match peer_port {
        Some(port) => println!("peer-dial 127.0.0.1:{port}"),
        None => println!("peer-dial none"),
    }
    // ⛔ LAST, and the line `capture-run` waits on. Everything a driver needs to
    // start a client is printed above it, so a driver that saw this line and
    // then read the others cannot race them.
    println!("serving for {seconds}s");

    let peer_wire = PeerWire::offering(offer, *torrent.info_hash());
    let (dialled, connections) =
        dial_when_announced(&mut lab, &tracker, &peer_wire, peer_port, seconds, dials);
    println!("peer-dialled {dialled}");
    // ⛔ ONE CONNECTION CANNOT ESTABLISH ANYTHING BUT A CONSTANT, so the count is
    // part of what a run reports. A peer ID carries a tail the build regenerates
    // per connection; a surface dialled once yields one observation, two captures
    // of one build then state two different constants, and the pair is
    // `divergent`. `SCHEMA-04` turns several into the pattern they share.
    println!("peer-connections {connections}");

    lab.wait();
    let expired = lab.deadline_expired();
    let journal = lab.shutdown();
    println!("deadline expired: {expired}");
    println!("segments: {}", journal.segments().len());

    report_announces(&tracker);
    report_offered(&peer_wire);
    report_peers(&peer_wire);
    write_bundle(&root, &journal, &torrent);
}

/// Dials the client's own listen port, once an announce says it is listening.
///
/// ⚠ Polling the observer rather than the socket. An announce is the first
/// evidence that the build read the torrent, and it is the earliest moment its
/// own listener is up for this info hash.
/// ⛔ **And it dials `dials` times, into ONE endpoint.** Each connection is its
/// own segment of the same transcript, so a value the build regenerates per
/// connection is observed more than once and can be stated as the pattern it is
/// rather than as one connection's bytes. ⚠ The connections are opened back to
/// back on purpose: they are the `connections` dimension of a sampling plan, not
/// the `sessions` one, and nothing here restarts the build.
fn dial_when_announced(
    lab: &mut Lab,
    tracker: &HttpTracker,
    peer_wire: &PeerWire,
    peer_port: Option<u16>,
    seconds: u64,
    dials: u32,
) -> (String, u32) {
    let Some(port) = peer_port else {
        return ("none".to_owned(), 0);
    };
    let address: SocketAddr = ([127, 0, 0, 1], port).into();
    let waiting_since = Instant::now();
    let wait_for = Duration::from_secs(seconds).min(Duration::from_secs(60));
    loop {
        if !tracker.announces().is_empty() {
            // ⛔ A FRESH IDENTITY PER CONNECTION, which is what run 18 lacked:
            // both its dials presented the same peer ID, so a build that drops a
            // second connection from a peer it already has would answer one and
            // ignore the other - which is exactly what it did.
            let presented = peer_wire.present();
            let first = match lab.dial(
                "peer-wire-dialled",
                address,
                presented.handshake().to_vec(),
                peer_wire.dialling(&presented),
            ) {
                Ok(_) => format!("{address}"),
                // ⚠ A refused dial is recorded and does not end the run. The
                // tracker surface is already measured by this point, and
                // throwing it away because a peer port was wrong would lose a
                // measurement to a detail of the driver.
                Err(error) => return (format!("refused: {error}"), 0),
            };
            let mut opened = 1;
            for index in 1..dials {
                let presented = peer_wire.present();
                // ⛔ AND FROM ANOTHER ADDRESS PER CONNECTION, which is what runs 18,
                // 19 and 36 lacked: every dial left from `127.0.0.1`, and
                // `aria2-next` answered the first of them only. Driven on
                // 2026-10-14 against 3.0.2 in a namespace holding loopback alone,
                // a second connection from `127.0.0.2` was answered, with a peer ID
                // of its own. ⚠ The source is a condition of the run, so it is
                // printed beside the connection it opened.
                let source = redial_source(index);
                // ⚠ A refused RE-dial ends the dialling and not the run, for the
                // same reason: a build that accepts one connection and not a
                // second has been measured once, which is a weaker record than
                // was asked for and a better one than none. The count says so.
                let redialled = match source {
                    Some(source) => lab.dial_again_from(
                        "peer-wire-dialled",
                        address,
                        source,
                        presented.handshake().to_vec(),
                        peer_wire.dialling(&presented),
                    ),
                    None => lab.dial_again(
                        "peer-wire-dialled",
                        address,
                        presented.handshake().to_vec(),
                        peer_wire.dialling(&presented),
                    ),
                };
                if let Err(error) = redialled {
                    println!("peer-redial refused: {error}");
                    break;
                }
                match source {
                    Some(source) => println!("peer-redial {index} from {source}"),
                    None => println!("peer-redial {index} from wherever the kernel chose"),
                }
                opened += 1;
            }
            return (first, opened);
        }
        if waiting_since.elapsed() >= wait_for {
            return ("no announce arrived".to_owned(), 0);
        }
        std::thread::sleep(Duration::from_millis(200));
    }
}

/// The loopback address the `index`th re-dial leaves from: `127.0.0.2` for the
/// first, counting up through `127.0.0.0/8`. ⚠ `None` only past
/// `127.255.255.254`, the last address below the `/8`'s broadcast, where a
/// re-dial leaves from wherever the kernel chooses.
fn redial_source(index: u32) -> Option<IpAddr> {
    let host = index.checked_add(1)?;
    if host >= 0x00ff_ffff {
        return None;
    }
    let [_, a, b, c] = host.to_be_bytes();
    Some(IpAddr::V4(Ipv4Addr::new(127, a, b, c)))
}

/// What the build put in its announces.
///
/// ⛔ Bytes, in hexadecimal, and never a client name. The peer ID is the
/// measurement; naming its prefix would put the one refused input inside the
/// component every observer trusts.
fn report_announces(tracker: &HttpTracker) {
    let announces = tracker.announces();
    println!("announces {}", announces.len());
    println!("announces-dropped {}", tracker.dropped());
    for (index, announce) in announces.iter().enumerate() {
        let peer_id = match announce.peer_id() {
            Some(Ok(bytes)) => hex(&bytes),
            Some(Err(error)) => format!("undecodable: {error}"),
            None => "absent".to_owned(),
        };
        println!("announce {index} peer-id {peer_id}");
        println!(
            "announce {index} query {}",
            names(&announce.query_key_order())
        );
        println!(
            "announce {index} headers {}",
            names(&announce.header_name_order())
        );
        let agent = announce
            .header(b"user-agent")
            .map_or_else(|| "absent".to_owned(), hex);
        println!("announce {index} user-agent {agent}");
        // ⚠ Whether the observer's own peer ID came back is how a reader tells a
        // build's announce from this example's own traffic in a shared log.
        let mine = announce
            .peer_id()
            .and_then(Result::ok)
            .is_some_and(|bytes| bytes.starts_with(OBSERVER_NAME));
        println!("announce {index} is-observer-peer-id {mine}");
    }
}

/// What this observer OFFERED on the peer wire, which is a run condition.
///
/// ⛔ **A build's answer means nothing without it.** `OBS-04` records what was
/// offered beside what came back, and the offer is per connection now: two
/// connections presenting one identity is a build being asked to talk to a peer
/// it already has, which run 18 measured as a build that answered once.
fn report_offered(peer_wire: &PeerWire) {
    let presented = peer_wire.presented();
    println!("offered-peer-ids {}", presented.len());
    for one in &presented {
        println!(
            "offered-peer-id {} {}",
            one.connection(),
            hex(one.peer_id())
        );
    }
}

/// What the build put on the peer wire.
fn report_peers(peer_wire: &PeerWire) {
    let streams = peer_wire.streams();
    println!("peer-streams {}", streams.len());
    println!("peer-dropped {}", peer_wire.dropped());
    for (index, stream) in streams.iter().enumerate() {
        println!("peer {index} role {:?}", stream.role());
        // ⚠ The offer this connection carried, beside what came back on it. A
        // reader comparing two connections' answers needs to see that the two
        // conditions differed, or a build answering once reads as a build that
        // refuses a second connection rather than one shown the same peer twice.
        match stream.presented_peer_id() {
            Some(peer_id) => println!("peer {index} presented {}", hex(peer_id)),
            None => println!("peer {index} presented none"),
        }
        println!("peer {index} messages {}", stream.messages().len());
        println!("peer {index} rebuilds {}", stream.rebuilds_from_raw());
        match stream.extended_handshake() {
            Some(Ok(message)) => {
                // ⛔ The undecoded dictionary is the evidence and the decode is
                // an interpretation of it, so both are printed and neither
                // replaces the other. The extension names a build invents are
                // identity as much as the ones it was offered.
                println!("peer {index} extended {}", hex(message.raw()));
                let offered: Vec<String> = message
                    .extension_ids()
                    .into_iter()
                    .map(|(name, id)| format!("{}={id}", String::from_utf8_lossy(&name)))
                    .collect();
                println!("peer {index} extensions {}", offered.join(","));
            }
            Some(Err(error)) => println!("peer {index} extended undecodable: {error}"),
            None => println!("peer {index} extended absent"),
        }
    }
}

/// Writes the run out as evidence and prints the rows a manifest would carry.
fn write_bundle(root: &str, journal: &bit_ids_lab::Journal, torrent: &SyntheticTorrent) {
    let mut bundle = match Bundle::create(root, slug("bit-ids-probe"), PhaseName::Observed) {
        Ok(bundle) => bundle,
        Err(error) => fail(&format!("could not open the bundle at {root}: {error}")),
    };

    // ⛔ Every endpoint the lab may have is named. `transcripts` refuses an
    // endpoint the plan does not carry rather than filing it under a guessed
    // kind, and the dialled peer endpoint only exists on some runs, so a plan
    // written from the endpoints that happen to be present would refuse the run
    // that dialled.
    let plan = BTreeMap::from([
        (
            slug("tracker-http"),
            TranscriptOf {
                id: slug("ev-tracker-capture"),
                kind: EvidenceKind::TrackerCapture,
            },
        ),
        (
            slug("peer-wire"),
            TranscriptOf {
                id: slug("ev-peer-transcript"),
                kind: EvidenceKind::PeerTranscript,
            },
        ),
        (
            slug("peer-wire-dialled"),
            TranscriptOf {
                id: slug("ev-peer-dialled"),
                kind: EvidenceKind::PeerTranscript,
            },
        ),
    ]);
    if let Err(error) = bundle.transcripts(journal, &plan) {
        fail(&format!("the run could not be written: {error}"));
    }
    if let Err(error) = bundle.transcript(
        slug("ev-metainfo"),
        EvidenceKind::Metainfo,
        "fixture/generated.torrent",
        torrent.metainfo(),
    ) {
        fail(&format!("the metainfo could not be written: {error}"));
    }

    // ⛔ Read back before it is described.
    if let Err(error) = bundle.verify() {
        fail(&format!("the bundle does not verify: {error}"));
    }

    println!("root {}", bundle.root().display());
    for record in bundle.evidence() {
        println!(
            "evidence {} {} {} {} {} {}",
            record.id,
            record.path,
            record.bytes,
            record.sha256,
            record.redacted,
            record.object_path()
        );
    }
    println!("verified: true");
}
