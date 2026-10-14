//! `OBS-06`, driven: the adjacent surfaces in a running lab, and the proof that
//! nothing they do leaves the lab's allowed address set.
//!
//! ⛔ **The egress claim is made three ways, because no one of them is enough.**
//! A unit test on the guard proves the guard refuses; it does not prove the
//! guard is reached. A source sweep proves no module went around it; it does not
//! prove the guard is right. A driven run proves what actually crossed a socket;
//! it cannot prove what a different input would have done. The three together
//! are what the entry's Prove asks for, and each one is named where it sits.
//!
//! ⚠ **What is not proved here is that no packet left the host.** That needs a
//! capture on the interface, which needs privileges a test runner does not have
//! and `docs/capture-host.md` does not grant. `TODO/observer.md` carries it as a
//! residual rather than letting the three checks above read as more than they
//! are.

use std::net::{IpAddr, Ipv4Addr, SocketAddr, SocketAddrV4, UdpSocket};
use std::time::Duration;

use bit_ids::canonical::Slug;
use bit_ids_lab::adjacent::{Capability, Surface, endpoint_name};
use bit_ids_lab::{Lab, bind};
use bit_ids_probe::dht::{Dht, OfferedPeers};
use bit_ids_probe::local_discovery::{self, GROUP_V4, GROUP_V6, LocalDiscovery};
use bit_ids_probe::mse::{self as mse_probe, Mse};
use bit_ids_probe::web_seed::WebSeedServer;
use bit_ids_wire::{bencode, dht, mse};

const ANNOUNCE: &[u8] = b"BT-SEARCH * HTTP/1.1\r\nHost: 239.192.152.143:6771\r\nPort: 51413\r\nInfohash: 0123456789abcdef0123456789abcdef01234567\r\ncookie: bit-ids\r\n\r\n\r\n";

/// Sends one datagram to a lab endpoint from a loopback socket.
fn announce_to(address: SocketAddr, payload: &[u8]) {
    let client = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    bind::send_to(&client, payload, address).expect("the lab is on loopback");
}

#[test]
fn a_local_discovery_announce_is_recorded_and_nothing_is_answered() {
    let observer = LocalDiscovery::new(Capability::enable(Surface::LocalDiscovery))
        .expect("the capability names this surface");
    let lab = Lab::builder()
        .deadline(Duration::from_secs(5))
        .datagram(endpoint_name(Surface::LocalDiscovery), observer.observing())
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");

    let endpoint = lab
        .endpoint(endpoint_name(Surface::LocalDiscovery))
        .expect("it was added");
    assert!(endpoint.address().ip().is_loopback());
    announce_to(endpoint.address(), ANNOUNCE);

    // The lab records on arrival, so wait for the segment rather than for a
    // fixed sleep: a timing guess is a flake on a loaded runner.
    let name = Slug::parse(endpoint_name(Surface::LocalDiscovery)).expect("a canonical name");
    let deadline = std::time::Instant::now() + Duration::from_secs(5);
    while observer.announces().is_empty() && std::time::Instant::now() < deadline {
        std::thread::sleep(Duration::from_millis(5));
    }

    let announces = observer.announces();
    assert_eq!(announces.len(), 1, "the announce was recorded");
    assert_eq!(
        announces[0].raw(),
        ANNOUNCE,
        "the bytes are the measurement"
    );
    assert!(
        announces[0].is_conforming(),
        "{:?}",
        announces[0].refusals()
    );
    assert_eq!(announces[0].port(), Some(51413));

    let journal = lab.shutdown();
    let segments = journal.for_endpoint(&name);
    // ⛔ **One segment, inbound.** BEP 14 defines no reply, so a second segment
    // would be this project inventing a protocol and then measuring what the
    // client did about it.
    assert_eq!(segments.len(), 1, "{segments:?}");
    assert_eq!(segments[0].bytes(), ANNOUNCE);
    assert_eq!(journal.received(&name), ANNOUNCE);
}

/// ⭐ The source address a responder is handed is the one the kernel reported,
/// driven end to end rather than asserted on a hand-made value.
///
/// ⛔ **The two ends of this comparison come from two kernels' worth of
/// bookkeeping and neither from the test.** The client asks its own socket what
/// port it was given, and the lab asks `recv_from` what port the packet came
/// from. A plumbing defect that passed a placeholder, the endpoint's own address
/// or a zero would leave every other assertion in this file true, because
/// nothing else reads the argument.
///
/// ⚠ The port is deliberately not the one the announce claims, so `Matches` and
/// `Differs` cannot both be satisfied by the same number.
#[test]
fn the_responder_is_handed_the_port_the_datagram_actually_came_from() {
    let observer = LocalDiscovery::new(Capability::enable(Surface::LocalDiscovery))
        .expect("the capability names this surface");
    let lab = Lab::builder()
        .deadline(Duration::from_secs(5))
        .datagram(endpoint_name(Surface::LocalDiscovery), observer.observing())
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");
    let endpoint = lab
        .endpoint(endpoint_name(Surface::LocalDiscovery))
        .expect("it was added");

    // Kept alive for the whole exchange: a client socket dropped before the lab
    // reads would free the port, and the comparison below would be against a
    // number nothing holds any more.
    let client = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    let mine = client.local_addr().expect("a bound socket has an address");
    bind::send_to(&client, ANNOUNCE, endpoint.address()).expect("the lab is on loopback");

    let deadline = std::time::Instant::now() + Duration::from_secs(5);
    while observer.announces().is_empty() && std::time::Instant::now() < deadline {
        std::thread::sleep(Duration::from_millis(5));
    }

    let announces = observer.announces();
    assert_eq!(announces.len(), 1);
    assert_eq!(
        announces[0].source(),
        Some(mine),
        "the responder saw a different address than the client sent from"
    );
    // ⚠ 51413 is what ANNOUNCE claims and `mine` is an ephemeral port the kernel
    // chose, so this is the ordinary case a real build produces.
    assert_ne!(
        mine.port(),
        51413,
        "the ephemeral port collided with the claim"
    );
    assert_eq!(
        announces[0].port_claim(),
        local_discovery::PortClaim::Differs {
            claimed: 51413,
            observed: mine.port(),
        }
    );
    // ⛔ And it is still a conforming announce. The comparison describes; it
    // does not refuse.
    assert!(
        announces[0].is_conforming(),
        "{:?}",
        announces[0].refusals()
    );

    // ⛔ The same bytes read back without a source answer NotObserved, which is
    // what an analysis pass over the evidence bundle gets: the journal carries
    // no source address for a datagram.
    let journal = lab.shutdown();
    let name = Slug::parse(endpoint_name(Surface::LocalDiscovery)).expect("a canonical name");
    let recorded = local_discovery::read(&journal.received(&name));
    assert_eq!(recorded.raw(), announces[0].raw());
    assert_eq!(
        recorded.port_claim(),
        local_discovery::PortClaim::NotObserved
    );
}

/// ⛔ The driven half of the egress claim: every byte that crossed a socket in
/// a real run went to an address inside the allowed set.
#[test]
fn every_endpoint_a_lab_hands_out_for_an_adjacent_surface_is_on_loopback() {
    let observer = LocalDiscovery::new(Capability::enable(Surface::LocalDiscovery))
        .expect("the capability names this surface");
    let lab = Lab::builder()
        .deadline(Duration::from_secs(2))
        .datagram(endpoint_name(Surface::LocalDiscovery), observer.observing())
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");

    for endpoint in lab.endpoints() {
        assert!(
            endpoint.address().ip().is_loopback(),
            "{} is at {}",
            endpoint.name(),
            endpoint.address()
        );
        assert_ne!(endpoint.address().port(), 0);
    }
    // ⚠ And the address the protocol itself names is not one of them. A lab
    // that had joined the group would be reachable from a real LAN.
    for endpoint in lab.endpoints() {
        assert_ne!(endpoint.address(), GROUP_V4);
        assert_ne!(endpoint.address(), GROUP_V6);
    }
    drop(lab.shutdown());
}

/// ⛔ The guard, driven from the module that would otherwise be the one to
/// reach the group: a socket the lab itself bound cannot be aimed at it.
#[test]
fn a_lab_socket_cannot_be_aimed_at_the_group_the_protocol_names() {
    let socket = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    for group in [GROUP_V4, GROUP_V6] {
        let refusal = bind::send_to(&socket, ANNOUNCE, group).expect_err("outside loopback");
        assert!(
            matches!(refusal, bind::BindError::NotReachable { .. }),
            "{refusal}"
        );
        // The refusal names where it was aimed, so a reader of a failing run
        // does not have to guess which surface tried.
        assert!(refusal.to_string().contains(&group.ip().to_string()));
    }
}

/// ⭐ A refusal is not the same as being unable to send. Without this, the case
/// above passes over a `send_to` that fails for every destination.
#[test]
fn the_same_socket_reaches_a_loopback_destination() {
    let listener = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    let address = listener.local_addr().expect("a bound socket has one");
    let socket = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    assert_eq!(
        bind::send_to(&socket, ANNOUNCE, address).expect("loopback is allowed"),
        ANNOUNCE.len()
    );
    listener
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("a read timeout");
    let mut buffer = vec![0_u8; ANNOUNCE.len() + 16];
    let (read, _) = listener.recv_from(&mut buffer).expect("it arrives");
    assert_eq!(&buffer[..read], ANNOUNCE);
}

/// ⛔ The sweep, over the crate that holds the protocols.
///
/// `bit-ids-lab` greps its own source because it owns the sockets. This crate
/// constructs none, and that is the claim: an observer here that opened one
/// would bypass every guard the lab has, and `local_discovery` is exactly the
/// module that would be tempted. BEP 14 is a multicast protocol, and joining a
/// group means binding a socket the lab did not hand out.
///
/// ⚠ **Naming `std::net` is not opening a socket**, and the needles say so:
/// `local_discovery` holds the two multicast groups as `SocketAddr` values so
/// the lab can be shown refusing them, and `peer_exchange` reads compact peer
/// lists into `SocketAddrV4`. Those are addresses. What is forbidden is the
/// call that turns one into a socket, or that sends on one.
#[test]
fn no_observer_opens_a_socket_of_its_own() {
    // ⚠ `socket2` since 2026-10-14, when the lab took it for one door: in the
    // lockfile, a crate here could name it and open a socket no needle above
    // spells.
    const FORBIDDEN: [&str; 9] = [
        "TcpListener",
        "UdpSocket::bind",
        "TcpStream::connect",
        "UdpSocket::connect",
        ".send_to(",
        "join_multicast",
        "set_multicast",
        "std::net::UdpSocket::",
        "socket2",
    ];

    let source = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let mut offenders = Vec::new();
    let mut checked = Vec::new();
    for entry in std::fs::read_dir(&source).expect("the crate has a src directory") {
        let path = entry.expect("a readable directory entry").path();
        if path.extension().is_none_or(|suffix| suffix != "rs") {
            continue;
        }
        let text = std::fs::read_to_string(&path).expect("a readable source file");
        checked.push(path.clone());
        for needle in FORBIDDEN {
            if text.contains(needle) {
                offenders.push(format!("{} names {needle}", path.display()));
            }
        }
    }
    // ⚠ A sweep that read nothing reports no offenders. The count is asserted
    // against the modules that exist, the way the lab's own sweep is, and the
    // floor moves up with them: `OBS-11` added `dht.rs`, `web_seed.rs` and
    // `mse.rs`, so eight are read now and a module quietly disappearing is a
    // failure rather than a smaller sweep.
    assert!(
        checked.len() >= 8,
        "only {} modules were read: {checked:?}",
        checked.len()
    );
    assert!(offenders.is_empty(), "{offenders:?}");
}

/// ⭐ The DHT observer in a running lab, driven by a client that is not this
/// project's test harness: a bencode query on the wire, an answer back, and the
/// transcript showing both.
///
/// ⛔ **This is the surface that could leave.** A real build's first DHT act is a
/// query to a bootstrap node nobody here owns, so the acceptance drives the two
/// guards with the address a build's own default actually names rather than one
/// chosen to pass, and shows the same socket reaching loopback so a refusal is
/// not confused with an inability to send.
#[test]
fn a_dht_query_is_answered_and_both_directions_are_recorded() {
    let observer = Dht::new(Capability::enable(Surface::Dht))
        .expect("the capability names this surface")
        .offering(
            OfferedPeers::of(&[SocketAddrV4::new(Ipv4Addr::LOCALHOST, 6881)])
                .expect("loopback is inside the allowed set"),
        );
    let lab = Lab::builder()
        .deadline(Duration::from_secs(5))
        .datagram(endpoint_name(Surface::Dht), observer.responder())
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");
    let endpoint = lab
        .endpoint(endpoint_name(Surface::Dht))
        .expect("it was added");
    assert!(endpoint.address().ip().is_loopback());

    // A get_peers, written here rather than by the observer, so the bytes on the
    // wire are the client's.
    let sent = concat!(
        "d1:ad2:id20:a-build-under-measur9:info_hash20:",
        "\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}",
        "\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}\u{11}",
        "e1:q9:get_peers1:t2:zz1:y1:qe"
    );
    let sent: Vec<u8> = sent.chars().map(|c| c as u8).collect();

    let client = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    client
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("a read timeout");
    bind::send_to(&client, &sent, endpoint.address()).expect("the lab is on loopback");

    let mut buffer = [0_u8; 2048];
    let (read, from) = client.recv_from(&mut buffer).expect("the observer answers");
    assert_eq!(from, endpoint.address(), "the answer came from the lab");

    // ⭐ Read back with the codec, not by eye. The answer is a message BEP 5
    // describes, carries the transaction the client chose, and names only the
    // loopback peer that went through `check_offered`.
    let answer = dht::Message::parse(&buffer[..read]).expect("the answer is bencode");
    assert!(answer.is_conforming(), "{:?}", answer.departures());
    assert_eq!(answer.kind(), dht::Kind::Response);
    assert_eq!(answer.transaction_id(), Some(&b"zz"[..]));
    assert_eq!(answer.node_id(), Some(&observer.node_id()[..]));
    let Some(bencode::Value::List(values)) = answer.argument(b"values") else {
        panic!("a get_peers answer carries values");
    };
    assert_eq!(values.len(), 1);
    assert_eq!(
        values[0],
        bencode::Value::Bytes(vec![127, 0, 0, 1, 0x1a, 0xe1]),
        "the only address offered is the loopback one"
    );

    let kept = observer.messages();
    assert_eq!(kept.len(), 1);
    assert_eq!(kept[0].raw(), sent.as_slice());
    assert_eq!(
        kept[0].source(),
        client.local_addr().expect("a bound socket has an address")
    );
    assert_eq!(kept[0].answered(), Some(&buffer[..read]));

    // ⛔ Two segments, one each way, and the outbound one is what the client
    // received rather than what the observer meant to send.
    let journal = lab.shutdown();
    let name = Slug::parse(endpoint_name(Surface::Dht)).expect("a canonical name");
    let segments = journal.for_endpoint(&name);
    assert_eq!(segments.len(), 2, "{segments:?}");
    assert_eq!(segments[0].bytes(), sent.as_slice());
    assert_eq!(segments[1].bytes(), &buffer[..read]);
}

/// ⛔ The address a real build's default names, refused by both doors, driven.
#[test]
fn the_bootstrap_node_a_dht_reaches_for_is_refused_and_loopback_is_not() {
    assert!(!bit_ids_probe::dht::A_REAL_BOOTSTRAP_NODE.ip().is_loopback());
    let socket = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    // Door two: where the lab sends.
    assert!(matches!(
        bind::send_to(
            &socket,
            b"d1:y1:qe",
            bit_ids_probe::dht::A_REAL_BOOTSTRAP_NODE
        ),
        Err(bind::BindError::NotReachable { .. })
    ));
    // Door three: where the lab tells the target to go.
    assert!(matches!(
        bind::check_offered(bit_ids_probe::dht::A_REAL_BOOTSTRAP_NODE),
        Err(bind::BindError::NotReachable { .. })
    ));
    // ⚠ And the control: a refusal is not an inability to send.
    let listener = bind::datagram(IpAddr::V4(Ipv4Addr::LOCALHOST)).expect("loopback binds");
    let address = listener.local_addr().expect("a bound socket has one");
    assert!(bind::send_to(&socket, b"d1:y1:qe", address).is_ok());
    assert!(bind::check_offered(address).is_ok());
}

/// ⚠ The observer records what arrives and does not correct it.
///
/// A build that announces a port it is not listening on, or an info hash for a
/// torrent it does not have, is making a claim. The claim is the measurement.
#[test]
fn a_malformed_announce_is_recorded_with_what_is_wrong_with_it() {
    let observer = LocalDiscovery::new(Capability::enable(Surface::LocalDiscovery))
        .expect("the capability names this surface");
    let responder = observer.observing();
    let source = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 45123);
    assert!(responder(source, b"BT-SEARCH * HTTP/1.1\r\nPort: nope\r\n\r\n").is_none());

    let announces = observer.announces();
    assert_eq!(announces.len(), 1);
    assert!(!announces[0].is_conforming());
    let reasons: Vec<String> = announces[0]
        .refusals()
        .iter()
        .map(local_discovery::Refusal::describe)
        .collect();
    assert_eq!(reasons.len(), 2, "{reasons:?}");
    assert!(
        reasons.iter().any(|why| why.contains("no Infohash")),
        "{reasons:?}"
    );
    assert!(
        reasons.iter().any(|why| why.contains("not a number")),
        "{reasons:?}"
    );
}

/// A datagram endpoint answers through the lab and both directions are
/// recorded, whichever responder it carries.
///
/// ⚠ **This does NOT prove the reply went through the egress guard**, and it
/// was named as though it did until a plant showed otherwise: reverting
/// `serve_datagram` to `socket.send_to(..)` leaves every assertion here true,
/// because a loopback echo works either way. Proving the routing behaviourally
/// would need a datagram arriving with a forged source address, which needs a
/// raw socket. `no_module_outside_the_bind_guard_reaches_the_network` in
/// `bit-ids-lab` is what refuses that revert, by reading the source.
#[test]
fn a_datagram_endpoint_answers_and_both_directions_are_recorded() {
    let lab = Lab::builder()
        .deadline(Duration::from_secs(2))
        .datagram("echo", |_: SocketAddr, received: &[u8]| {
            Some(received.to_vec())
        })
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");
    let endpoint = lab.endpoint("echo").expect("it was added");

    let client = UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).expect("a client socket binds");
    client
        .send_to(b"ping", endpoint.address())
        .expect("the endpoint is on loopback");
    client
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("a read timeout");
    let mut buffer = [0_u8; 16];
    let (read, _) = client.recv_from(&mut buffer).expect("the echo comes back");
    assert_eq!(&buffer[..read], b"ping");

    let journal = lab.shutdown();
    let name = Slug::parse("echo").expect("a canonical name");
    assert_eq!(journal.for_endpoint(&name).len(), 2, "in and back out");
}

/// ⭐ The web seed in a running lab, fetched over a real TCP connection.
///
/// ⛔ **The bytes served are the torrent's own payload**, so every piece hashes.
/// A seed answering anything else would be blacklisted by the build and the run
/// would measure a build reacting to a broken server rather than a build using a
/// web seed. This drives the whole path: the torrent names the endpoint, the
/// endpoint serves the payload, and the span asked for is the span returned.
#[test]
fn a_web_seed_fetch_is_answered_with_the_torrents_own_payload() {
    use std::io::{Read as _, Write as _};

    let torrent = bit_ids_lab::SyntheticTorrent::generate(bit_ids_lab::TorrentSpec::default())
        .expect("the default spec describes a usable torrent");
    let observer = WebSeedServer::new(
        Capability::enable(Surface::WebSeed),
        torrent.payload().to_vec(),
    )
    .expect("the capability names this surface");

    let lab = Lab::builder()
        .deadline(Duration::from_secs(10))
        .stream(endpoint_name(Surface::WebSeed), observer.responder())
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");
    let endpoint = lab
        .endpoint(endpoint_name(Surface::WebSeed))
        .expect("it was added");
    assert!(endpoint.address().ip().is_loopback());

    // ⭐ And the torrent can name it, through the guard. This is the whole
    // reason `TorrentSpec` grew the field.
    let SocketAddr::V4(address) = endpoint.address() else {
        panic!("the lab binds IPv4 loopback here");
    };
    let seed = bit_ids_lab::torrent::WebSeed::new(address, "/payload")
        .expect("a lab endpoint is inside the allowed set");
    assert!(seed.url().starts_with("http://127.0.0.1:"));

    let mut stream = bind::dial(endpoint.address(), Duration::from_secs(5)).expect("it connects");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("a read timeout");
    let request = format!(
        "GET /payload HTTP/1.1\r\nHost: {address}\r\nUser-Agent: bit-ids-driver/1\r\nRange: bytes=64-127\r\n\r\n"
    );
    stream
        .write_all(request.as_bytes())
        .expect("the endpoint is up");
    stream.flush().expect("flushed");

    // Read until the body is complete: 64 bytes plus whatever head precedes it.
    let mut reply = Vec::new();
    let mut buffer = [0_u8; 4096];
    while reply.len() < 64 {
        let read = stream.read(&mut buffer).expect("the observer answers");
        if read == 0 {
            break;
        }
        reply.extend_from_slice(&buffer[..read]);
        if let Some(end) = bit_ids_wire::tracker_http::head_end(&reply)
            && reply.len() - end >= 64
        {
            break;
        }
    }
    let end = bit_ids_wire::tracker_http::head_end(&reply).expect("a reply has a head");
    let head = String::from_utf8_lossy(&reply[..end]).into_owned();
    assert!(head.starts_with("HTTP/1.1 206 Partial Content"), "{head}");
    assert!(head.contains("Content-Range: bytes 64-127/65536"), "{head}");
    assert!(head.contains("Accept-Ranges: bytes"), "{head}");

    // ⛔ The bytes are the torrent's own, so a build's piece hashes would pass.
    assert_eq!(&reply[end..end + 64], &torrent.payload()[64..128]);

    let fetches = observer.fetches();
    assert_eq!(fetches.len(), 1);
    assert!(fetches[0].is_conforming(), "{:?}", fetches[0].refusals());
    assert_eq!(
        fetches[0].requested(),
        bit_ids_probe::web_seed::Requested::Span {
            first: 64,
            last: 127
        }
    );
    assert_eq!(fetches[0].user_agent(), Some(&b"bit-ids-driver/1"[..]));
    assert_eq!(fetches[0].status(), Some(206));
    assert_eq!(fetches[0].served(), 64);

    drop(stream);
    let journal = lab.shutdown();
    let name = Slug::parse(endpoint_name(Surface::WebSeed)).expect("a canonical name");
    // ⚠ Both directions recorded: the request as sent and the reply as written.
    assert_eq!(journal.received(&name), request.as_bytes());
    assert!(!journal.for_endpoint(&name).is_empty());
}

/// ⭐ The MSE exchange in a running lab, over a real TCP connection.
///
/// ⛔ **The point is the last assertion.** A build that negotiates encryption
/// puts its `BitTorrent` handshake inside `IA`, so its peer ID is not on the wire
/// in the clear. Reading it back out here and comparing it against what
/// `OBS-04`'s own reader makes of the same bytes is two observations of one
/// value, arriving through two different doors, which is what `SCHEMA-03` calls
/// corroboration.
#[test]
fn an_mse_exchange_completes_and_the_handshake_comes_out_of_it() {
    use std::io::{Read as _, Write as _};

    const INFO_HASH: [u8; 20] = [0x11; 20];
    const IA: &[u8] = b"\x13BitTorrent protocol\x00\x00\x00\x00\x00\x10\x00\x05\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11\x11a-build-under-measur";

    let observer = Mse::new(
        Capability::enable(Surface::Mse),
        INFO_HASH,
        mse_probe::Selection::Rc4,
    )
    .expect("the capability names this surface")
    .with_pad_b(vec![0xBB; 23]);
    let their_key = observer.public_key();

    let lab = Lab::builder()
        .deadline(Duration::from_secs(10))
        .stream(endpoint_name(Surface::Mse), observer.responder())
        .expect("a canonical endpoint name")
        .start()
        .expect("loopback binds");
    let endpoint = lab
        .endpoint(endpoint_name(Surface::Mse))
        .expect("it was added");
    assert!(endpoint.address().ip().is_loopback());

    // The initiator's side, written by the module rather than copied here: a
    // second reading of the specification in a test file is how two copies
    // disagree.
    let private = *b"a-build-private-key0";
    let opening = mse_probe::initiate(
        &private,
        &their_key,
        &INFO_HASH,
        mse::CRYPTO_PLAINTEXT | mse::CRYPTO_RC4,
        &[0xAA; 41],
        &[0xCC; 7],
        IA,
    );

    let mut stream = bind::dial(endpoint.address(), Duration::from_secs(5)).expect("it connects");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("a read timeout");
    stream.write_all(&opening).expect("the endpoint is up");
    stream.flush().expect("flushed");

    // The reply is the observer's key, its padding, and the encrypted selection.
    let want = mse::KEY_LEN + 23 + 8 + 4 + 2 + 23;
    let mut reply = Vec::new();
    let mut buffer = [0_u8; 4096];
    while reply.len() < want {
        let read = stream.read(&mut buffer).expect("the observer answers");
        if read == 0 {
            break;
        }
        reply.extend_from_slice(&buffer[..read]);
    }
    assert_eq!(&reply[..mse::KEY_LEN], &their_key[..], "the observer's key");

    // ⭐ Read the selection with the key the build would derive, not with a
    // value the observer handed over.
    let secret = mse::shared_secret(&their_key, &private);
    let mut plain = reply[mse::KEY_LEN + 23..].to_vec();
    mse::Rc4::new(&mse::key_b(&secret, &INFO_HASH)).apply(&mut plain);
    assert_eq!(&plain[..8], &mse::VC, "the verification constant");
    assert_eq!(&plain[8..12], &mse::CRYPTO_RC4.to_be_bytes());

    let exchanges = observer.exchanges();
    assert_eq!(exchanges.len(), 1);
    assert!(
        exchanges[0].is_conforming(),
        "{:?}",
        exchanges[0].refusals()
    );
    assert_eq!(
        exchanges[0].pad_a_len(),
        41,
        "the padding length is a choice"
    );
    let provide = exchanges[0].provide().expect("it decrypted");
    assert!(provide.offers_plaintext() && provide.offers_rc4());
    assert_eq!(provide.pad, vec![0xCC; 7]);

    // ⛔ Two doors, one value. The peer ID inside the encrypted `IA` is the one
    // `OBS-04`'s reader finds in the same bytes read as a plain handshake.
    let inside = exchanges[0].initial_payload().expect("IA");
    assert_eq!(inside, IA);
    let handshake = bit_ids_wire::peer_wire::Transcript::parse(inside)
        .expect("IA is a peer-wire handshake")
        .handshake()
        .peer_id()
        .to_vec();
    assert_eq!(handshake, b"a-build-under-measur".to_vec());

    drop(stream);
    let journal = lab.shutdown();
    let name = Slug::parse(endpoint_name(Surface::Mse)).expect("a canonical name");
    assert_eq!(journal.received(&name), opening.as_slice());
}

/// ⛔ The third door, on the two surfaces that had it open longest.
///
/// ⚠ **This case exists because a door sweep found the hole after `OBS-11` had
/// already closed it twice elsewhere.** `bind::check_offered` was added for the
/// DHT's `values` list and for BEP 19's `url-list`, and `OfferedPeer`, the type
/// *both* tracker observers hand a build to dial, had public fields and no
/// check at all, since `OBS-02`. A gate on some of several paths into one action
/// is what `docs/methodology/reviews.md` calls the most recurring hole there is,
/// and grepping for the callers not on the list is how that document says to
/// find it.
#[test]
fn a_tracker_cannot_offer_a_peer_outside_the_lab() {
    use bit_ids_probe::OfferedPeer;

    // Addresses a build would actually dial if it were handed one.
    for (address, port) in [
        ([198_u8, 51, 100, 7], 6881_u16),
        ([8, 8, 8, 8], 80),
        ([0, 0, 0, 0], 6881),
        ([192, 168, 1, 10], 6881),
    ] {
        assert!(
            matches!(
                OfferedPeer::new(address, port, *b"bit-ids-fixture-0001"),
                Err(bind::BindError::NotReachable { .. })
            ),
            "{address:?}:{port} was offered"
        );
    }

    // ⚠ The control, so a constructor that refused everything would fail here,
    // and the compact form is the six bytes BEP 15 and BEP 23 both write.
    let allowed = OfferedPeer::new([127, 0, 0, 1], 6881, *b"bit-ids-fixture-0001")
        .expect("loopback is inside the allowed set");
    assert_eq!(allowed.compact(), vec![127, 0, 0, 1, 0x1a, 0xe1]);
    assert_eq!(allowed.address(), [127, 0, 0, 1]);
    assert_eq!(allowed.port(), 6881);
}
