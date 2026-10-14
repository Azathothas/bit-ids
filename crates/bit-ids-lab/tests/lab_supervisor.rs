//! `OBS-01`'s acceptance: the six properties the supervisor exists to hold.
//!
//! ⚠ Every test here drives real sockets on this host. That is the point:
//! `docs/methodology/gate.md` part (b) says a green suite proves the code and
//! not the platform, and a loopback guard tested against a mock of a socket
//! proves nothing about what the kernel will hand out.

use std::io::{Read, Write};
use std::net::{IpAddr, Ipv4Addr, SocketAddr, TcpListener, TcpStream, UdpSocket};
use std::sync::mpsc;
use std::time::Duration;

use bit_ids::canonical::Slug;
use bit_ids_lab::{ConnectionId, Lab, LabError, StreamReply};
use bit_ids_wire::tracker_udp::Direction;

fn slug(text: &str) -> Slug {
    Slug::parse(text).expect("a test endpoint name is a slug")
}

/// A responder that answers every read with one fixed reply and keeps reading.
fn echo_once(
    reply: &'static [u8],
) -> impl Fn(ConnectionId, &[u8]) -> StreamReply + Send + Sync + 'static {
    move |_connection, received: &[u8]| StreamReply::Answer {
        consumed: received.len(),
        send: reply.to_vec(),
    }
}

#[test]
fn a_lab_refuses_to_bind_anything_but_loopback() {
    for host in [
        IpAddr::V4(Ipv4Addr::UNSPECIFIED),
        IpAddr::V4(Ipv4Addr::new(192, 168, 1, 10)),
    ] {
        let started = Lab::builder()
            .host(host)
            .stream("tracker-http", echo_once(b"ok"))
            .expect("a canonical name")
            .start();
        match started {
            Err(LabError::Bind(error)) => {
                let text = error.to_string();
                assert!(
                    text.contains("loopback"),
                    "the refusal should say why: {text}"
                );
            }
            Err(other) => panic!("expected a bind refusal for {host}, got {other}"),
            Ok(_) => panic!("{host} is not loopback and the lab started on it"),
        }
    }
}

#[test]
fn the_refusal_covers_a_datagram_endpoint_too() {
    // ⛔ The guard exists once and both transports go through it. A lab that
    // refused a TCP bind and allowed the UDP one is exactly the one-gated-door
    // defect the crate documentation names.
    let started = Lab::builder()
        .host(IpAddr::V4(Ipv4Addr::UNSPECIFIED))
        .datagram("tracker-udp", |_: SocketAddr, _: &[u8]| None)
        .expect("a canonical name")
        .start();
    assert!(matches!(started, Err(LabError::Bind(_))));
}

#[test]
fn every_endpoint_reports_the_address_its_socket_actually_got() {
    let lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .datagram("tracker-udp", |_: SocketAddr, _: &[u8]| None)
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    assert_eq!(lab.endpoints().len(), 2);
    for endpoint in lab.endpoints() {
        let address = endpoint.address();
        assert!(
            address.ip().is_loopback(),
            "{} is not on loopback",
            endpoint.name()
        );
        assert_ne!(
            address.port(),
            0,
            "{} still reports the requested port rather than the assigned one",
            endpoint.name()
        );
    }

    // Two endpoints in one lab cannot share a port either.
    let ports: Vec<u16> = lab
        .endpoints()
        .iter()
        .map(|endpoint| endpoint.address().port())
        .collect();
    assert_ne!(ports[0], ports[1]);
}

#[test]
fn two_labs_on_one_host_get_distinct_ports() {
    let first = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    let second = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let one = first
        .endpoint("tracker-http")
        .expect("it was added")
        .address();
    let two = second
        .endpoint("tracker-http")
        .expect("it was added")
        .address();
    assert_ne!(
        one.port(),
        two.port(),
        "a named port would have collided here"
    );
}

#[test]
fn a_stream_endpoint_records_what_arrived_and_what_was_sent_in_order() {
    let lab = Lab::builder()
        .stream("peer-wire", |_connection, received: &[u8]| {
            StreamReply::Answer {
                consumed: received.len(),
                // The reply is derived from the request so the transcript cannot
                // pass by recording a constant in the right place.
                send: received.to_ascii_uppercase(),
            }
        })
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("peer-wire").expect("it was added").address();
    let mut client = TcpStream::connect(address).expect("the endpoint accepts");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");

    // ⛔ Each round trip waits on the reply before sending the next request.
    // Sending three chunks and asserting the order would be asserting a
    // scheduling outcome this test does not control.
    for request in [&b"alpha"[..], b"bravo", b"charlie"] {
        client.write_all(request).expect("the endpoint is reading");
        client.flush().expect("flush");
        let mut reply = vec![0_u8; request.len()];
        client.read_exact(&mut reply).expect("the endpoint answers");
        assert_eq!(reply, request.to_ascii_uppercase());
    }
    drop(client);

    let journal = lab.shutdown();
    let peer = slug("peer-wire");
    let recorded: Vec<(Direction, Vec<u8>)> = journal
        .for_endpoint(&peer)
        .iter()
        .map(|segment| (segment.direction(), segment.bytes().to_vec()))
        .collect();

    assert_eq!(
        recorded,
        vec![
            (Direction::FromTarget, b"alpha".to_vec()),
            (Direction::ToTarget, b"ALPHA".to_vec()),
            (Direction::FromTarget, b"bravo".to_vec()),
            (Direction::ToTarget, b"BRAVO".to_vec()),
            (Direction::FromTarget, b"charlie".to_vec()),
            (Direction::ToTarget, b"CHARLIE".to_vec()),
        ]
    );
    assert_eq!(journal.received(&peer), b"alphabravocharlie".to_vec());
}

#[test]
fn a_datagram_endpoint_records_each_packet_and_its_answer() {
    let lab = Lab::builder()
        .datagram("tracker-udp", |_: SocketAddr, received: &[u8]| {
            Some(received.iter().rev().copied().collect())
        })
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("tracker-udp").expect("it was added").address();
    let client = UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).expect("a client socket binds");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");

    for request in [&b"\x00\x01\x02"[..], b"\xff\xfe"] {
        client
            .send_to(request, address)
            .expect("the endpoint is up");
        let mut reply = vec![0_u8; 64];
        let (read, _) = client.recv_from(&mut reply).expect("the endpoint answers");
        let expected: Vec<u8> = request.iter().rev().copied().collect();
        assert_eq!(&reply[..read], &expected[..]);
    }

    let journal = lab.shutdown();
    let recorded: Vec<(Direction, Vec<u8>)> = journal
        .for_endpoint(&slug("tracker-udp"))
        .iter()
        .map(|segment| (segment.direction(), segment.bytes().to_vec()))
        .collect();
    assert_eq!(
        recorded,
        vec![
            (Direction::FromTarget, b"\x00\x01\x02".to_vec()),
            (Direction::ToTarget, b"\x02\x01\x00".to_vec()),
            (Direction::FromTarget, b"\xff\xfe".to_vec()),
            (Direction::ToTarget, b"\xfe\xff".to_vec()),
        ]
    );
}

#[test]
fn the_largest_datagram_a_host_can_deliver_is_recorded_whole() {
    // ⭐ `recv_from` truncates a datagram that does not fit the buffer and
    // reports the truncated length, so the record would say a short packet
    // arrived whole. Measured on this host on 2026-09-05: loopback carries a
    // 65507-byte payload, which is the IPv4 maximum.
    const PAYLOAD: usize = 65507;

    let lab = Lab::builder()
        .datagram("tracker-udp", |_: SocketAddr, received: &[u8]| {
            // The answer is the length, so a truncated read cannot look right.
            Some(received.len().to_be_bytes().to_vec())
        })
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("tracker-udp").expect("added").address();
    let client = UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).expect("a client socket binds");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");

    // A repeating pattern with a period that is not a power of two, so a read
    // that lost or duplicated a block does not still compare equal.
    let sent: Vec<u8> = (0..PAYLOAD)
        .map(|index| u8::try_from(index % 251).expect("a value under 251 is a byte"))
        .collect();
    client
        .send_to(&sent, address)
        .expect("loopback carries a full-size datagram");
    let mut reply = [0_u8; 8];
    let (read, _) = client.recv_from(&mut reply).expect("the endpoint answers");
    assert_eq!(read, 8);
    assert_eq!(
        usize::from_be_bytes(reply),
        PAYLOAD,
        "the endpoint read short"
    );

    let journal = lab.shutdown();
    assert_eq!(journal.received(&slug("tracker-udp")), sent);
}

#[test]
fn a_client_that_connects_and_says_nothing_does_not_outlive_the_deadline() {
    let lab = Lab::builder()
        .deadline(Duration::from_millis(200))
        .poll(Duration::from_millis(5))
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab
        .endpoint("tracker-http")
        .expect("it was added")
        .address();
    let silent = TcpStream::connect(address).expect("the endpoint accepts");

    // ⚠ The bound is here to turn a hang into a failure, not to measure how
    // long the lab took. A supervisor that never notices its deadline would
    // otherwise block this test until the CI job's own timeout, which reports
    // as an infrastructure problem rather than as this defect.
    //
    // ⛔ `wait` rather than `shutdown`: shutdown asks the endpoints to stop, so
    // it would stop this lab itself and the flag it then reported would say
    // nothing about the deadline. Waiting means only the deadline can end it.
    let (sender, receiver) = mpsc::channel();
    std::thread::spawn(move || {
        let mut lab = lab;
        lab.wait();
        let _ = sender.send((lab.deadline_expired(), lab.shutdown()));
    });
    let (expired, journal) = receiver
        .recv_timeout(Duration::from_secs(30))
        .expect("the lab stopped itself");

    assert!(
        expired,
        "the lab stopped, and not because its deadline passed"
    );
    assert!(
        journal.segments().is_empty(),
        "a silent client sent nothing and nothing should be recorded"
    );
    drop(silent);
}

#[test]
fn a_lab_told_to_stop_does_not_report_that_it_ran_out_of_time() {
    // The other half of the pair above. A flag that is set whatever ended the
    // lab reports the deadline over every run, which is the same as reporting
    // nothing.
    let mut quick = Lab::builder()
        .deadline(Duration::from_secs(300))
        .poll(Duration::from_millis(5))
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    quick.stop();
    assert!(
        !quick.deadline_expired(),
        "this lab was told to stop 300 seconds before its deadline"
    );
}

#[test]
fn shutting_a_lab_down_releases_every_port_it_held() {
    let lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .datagram("tracker-udp", |_: SocketAddr, _: &[u8]| None)
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    let stream_port = lab
        .endpoint("tracker-http")
        .expect("added")
        .address()
        .port();
    let datagram_port = lab.endpoint("tracker-udp").expect("added").address().port();

    // ⚠ Nothing connects first. A port that carried a closed connection can sit
    // in TIME_WAIT, and rebinding it succeeds or fails for reasons that have
    // nothing to do with whether this lab released it.
    drop(lab.shutdown());

    TcpListener::bind((Ipv4Addr::LOCALHOST, stream_port))
        .expect("the listener was closed, so its port is free");
    UdpSocket::bind((Ipv4Addr::LOCALHOST, datagram_port))
        .expect("the socket was closed, so its port is free");
}

#[test]
fn dropping_a_lab_releases_its_ports_without_shutdown_being_called() {
    let port = {
        let lab = Lab::builder()
            .stream("tracker-http", echo_once(b"ok"))
            .expect("a canonical name")
            .start()
            .expect("loopback binds");
        lab.endpoint("tracker-http")
            .expect("added")
            .address()
            .port()
    };
    TcpListener::bind((Ipv4Addr::LOCALHOST, port))
        .expect("drop stops the endpoint and frees the port");
}

#[test]
fn a_lab_with_no_endpoint_or_a_repeated_name_is_refused() {
    assert!(matches!(Lab::builder().start(), Err(LabError::NoEndpoints)));

    let repeated = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start();
    assert!(matches!(repeated, Err(LabError::DuplicateEndpoint(_))));

    // A name that is not a canonical identifier is refused where it is given,
    // not at start, so the report names the call that was wrong.
    assert!(matches!(
        Lab::builder().stream("Tracker_HTTP", echo_once(b"ok")),
        Err(LabError::Name(_))
    ));
}

#[test]
fn a_partial_read_is_kept_until_the_responder_consumes_it() {
    // The responder answers only on a complete line, which is what a framed
    // surface does. Two writes make one unit, and the journal keeps both.
    let lab = Lab::builder()
        .stream("peer-wire", line_framed())
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("peer-wire").expect("added").address();
    let mut client = TcpStream::connect(address).expect("the endpoint accepts");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");
    client.write_all(b"half").expect("write");
    client.flush().expect("flush");
    client.write_all(b"-line\n").expect("write");
    client.flush().expect("flush");
    let mut reply = [0_u8; 8];
    client.read_exact(&mut reply).expect("the endpoint answers");
    assert_eq!(&reply, b"got-line");
    drop(client);

    let journal = lab.shutdown();
    let peer = slug("peer-wire");
    assert_eq!(journal.received(&peer), b"half-line\n".to_vec());
    let sent: Vec<&[u8]> = journal
        .for_endpoint(&peer)
        .iter()
        .filter(|segment| segment.direction() == Direction::ToTarget)
        .map(|segment| segment.bytes())
        .collect();
    assert_eq!(sent, vec![&b"got-line"[..]], "one line, one answer");
}

/// A line-framed responder, which is the shape a message surface has.
fn line_framed() -> impl Fn(ConnectionId, &[u8]) -> StreamReply + Send + Sync + 'static {
    |_connection, received: &[u8]| match received.iter().position(|byte| *byte == b'\n') {
        Some(end) => StreamReply::Answer {
            consumed: end + 1,
            send: b"got-line".to_vec(),
        },
        None => StreamReply::NeedMore,
    }
}

#[test]
fn two_units_in_one_write_are_both_answered_without_waiting_for_more_bytes() {
    // ⭐ The guard-mutation pass found this one. Replacing the buffer drain
    // with a clear changed no test result, because every fixture sent one unit
    // per write, and the real defect underneath was that the responder was
    // offered the buffer once per read rather than until it stopped consuming.
    // A client that sends two messages and waits for two answers would have
    // waited forever.
    let lab = Lab::builder()
        .stream("peer-wire", line_framed())
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("peer-wire").expect("added").address();
    let mut client = TcpStream::connect(address).expect("the endpoint accepts");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");
    client
        .write_all(b"one\ntwo\n")
        .expect("one write, two units");
    client.flush().expect("flush");

    let mut both = [0_u8; 16];
    client
        .read_exact(&mut both)
        .expect("both units are answered from the one read");
    assert_eq!(&both, b"got-linegot-line");

    let journal = lab.shutdown();
    let peer = slug("peer-wire");
    assert_eq!(journal.received(&peer), b"one\ntwo\n".to_vec());
    let sent: Vec<&[u8]> = journal
        .for_endpoint(&peer)
        .iter()
        .filter(|segment| segment.direction() == Direction::ToTarget)
        .map(|segment| segment.bytes())
        .collect();
    assert_eq!(sent.len(), 2, "one answer per unit, not per read");
}

#[test]
fn a_connection_that_goes_quiet_between_writes_is_not_closed_under_the_client() {
    // ⚠ The sleep creates the condition rather than waiting for one: a socket
    // with nothing on it is what makes the read time out, and a read timeout is
    // what a wrongly narrow error match treats as a broken connection. Twenty
    // poll intervals is a margin, not a measurement.
    let poll = Duration::from_millis(5);
    let lab = Lab::builder()
        .poll(poll)
        .deadline(Duration::from_secs(300))
        .stream("peer-wire", line_framed())
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("peer-wire").expect("added").address();
    let mut client = TcpStream::connect(address).expect("the endpoint accepts");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");

    std::thread::sleep(poll * 20);
    client
        .write_all(b"late\n")
        .expect("the endpoint is still there");
    client.flush().expect("flush");
    let mut reply = [0_u8; 8];
    client
        .read_exact(&mut reply)
        .expect("a quiet connection is still a connection");
    assert_eq!(&reply, b"got-line");
}

#[test]
fn a_close_reply_ends_the_connection_and_the_client_sees_the_end_of_stream() {
    let lab = Lab::builder()
        .stream("tracker-http", |_connection, _: &[u8]| StreamReply::Close {
            send: b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n".to_vec(),
        })
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("tracker-http").expect("added").address();
    let mut client = TcpStream::connect(address).expect("the endpoint accepts");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");
    client.write_all(b"GET /announce\r\n\r\n").expect("write");
    client.flush().expect("flush");

    let mut answer = Vec::new();
    client
        .read_to_end(&mut answer)
        .expect("the endpoint answers and then closes");
    assert!(answer.starts_with(b"HTTP/1.1 200 OK"));
}

/// ⭐ The loopback guard is only a guard if it is the only way to a socket.
///
/// The rule is stated in this crate's documentation and would otherwise be
/// enforced by nobody: an observer that reached for `UdpSocket::bind` directly
/// would compile, pass every other test here, and listen wherever it was told
/// to. `docs/methodology/reviews.md` lens 1 says to grep for the doors you did
/// not enumerate, so this greps.
///
/// ⛔ **Dialling out is on the list even though nothing dials yet.** The door
/// sweep found that the guard covered listeners and said nothing about outbound
/// connections, and `OBS-04` is authored to implement an active dial role. A
/// needle here now means that dial has to be written where the guard is, rather
/// than being reviewed on the day it is added.
///
/// ⛔ **`.send_to(` is on the list because the sweep that did not name it missed
/// a live one.** `endpoint::serve_datagram` replied on the bound socket
/// directly, so the loopback guard approved where that socket listened and
/// nothing decided where it sent. A needle spelled as a method call is what
/// distinguishes `socket.send_to(..)` from `bind::send_to(..)`, which is the
/// door rather than a way around it.
#[test]
fn no_module_outside_the_bind_guard_reaches_the_network() {
    // ⚠ `socket2` by its crate name, not by one call: `bind::dial_from` is its
    // one door, and `SockRef` or an aliased import would pass a needle that
    // named `Socket::new` alone.
    const FORBIDDEN: [&str; 7] = [
        "TcpListener::bind",
        "UdpSocket::bind",
        "TcpStream::connect",
        "UdpSocket::connect",
        ".send_to(",
        "UdpSocket::send",
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
        if path.file_name().is_some_and(|name| name == "bind.rs") {
            continue;
        }
        let text = std::fs::read_to_string(&path).expect("a readable source file");
        checked.push(path.clone());
        for needle in FORBIDDEN {
            if text.contains(needle) {
                offenders.push(format!("{} calls {needle}", path.display()));
            }
        }
    }
    // ⚠ A sweep that read nothing reports no offenders, which is how a guard
    // becomes theatre. The count is asserted against the modules that exist.
    assert!(
        checked.len() >= 3,
        "only {} modules were read: {checked:?}",
        checked.len()
    );
    assert!(offenders.is_empty(), "{offenders:?}");
}

#[test]
fn an_endpoint_that_can_serve_no_connection_is_refused_rather_than_silently_deaf() {
    // A lab built with a zero cap accepts every connection and closes it at
    // once, which reads to a client as a server that is up and broken. The
    // door sweep found it: the cap is a bound on threads and nothing said it
    // had a floor.
    let refused = Lab::builder()
        .max_connections(0)
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start();
    assert!(matches!(refused, Err(LabError::NoConnectionsAllowed)));
}

#[test]
fn a_stopped_lab_refuses_a_dial_rather_than_writing_bytes_nothing_serves() {
    // ⛔ The door sweep found this. A dial's opening bytes are written before
    // the worker's first stop check, so a dial on a stopped lab would put a
    // handshake on the wire that nothing was ever going to answer.
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a listener");
    let address = listener.local_addr().expect("an address");

    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    lab.stop();

    let refused = lab.dial("peer-dial", address, b"hello".to_vec(), echo_once(b"ok"));
    assert!(matches!(refused, Err(LabError::Stopped)));

    // Nothing connected, so the listener has nothing waiting.
    listener
        .set_nonblocking(true)
        .expect("a nonblocking listener");
    assert!(
        listener.accept().is_err(),
        "the refused dial opened no connection"
    );
}

#[test]
fn a_dial_goes_through_the_same_loopback_guard_a_bind_does() {
    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    // ⛔ TEST-NET-2, reserved for documentation. If the guard stopped firing
    // this would try to reach it, which is the failure being prevented.
    let elsewhere = std::net::SocketAddr::from(([198, 51, 100, 7], 6881));
    lab.set_dial_timeout(Duration::from_millis(50));
    let refused = lab.dial("peer-dial", elsewhere, Vec::new(), echo_once(b"ok"));
    match refused {
        Err(LabError::Bind(error)) => assert!(error.to_string().contains("loopback")),
        other => panic!("expected a loopback refusal, got {other:?}"),
    }
    assert_eq!(lab.endpoints().len(), 1, "a refused dial adds no endpoint");
}

#[test]
fn a_connection_that_never_completes_a_unit_is_closed_at_the_pending_cap() {
    // ⛔ A responder that keeps answering `NeedMore` never drains the buffer,
    // and the target is untrusted by construction.
    let lab = Lab::builder()
        .max_pending_bytes(4096)
        .stream("peer-wire", |_connection, _buffered: &[u8]| {
            StreamReply::NeedMore
        })
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let address = lab.endpoint("peer-wire").expect("added").address();
    let mut client = TcpStream::connect(address).expect("the endpoint accepts");
    client
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("a read timeout is settable");

    // ⚠ Write until the endpoint closes on us. The first version stopped after
    // a fixed byte budget and asserted the close had happened by then, which is
    // a scheduling outcome this test does not control: it passed alone and
    // failed twice in three loaded workspace runs. The bound below turns a hang
    // into a failure and is not a measurement of how fast a close arrives.
    let block = vec![b'x'; 1024];
    let deadline = std::time::Instant::now() + Duration::from_secs(20);
    let closed = loop {
        if client.write_all(&block).is_err() || client.flush().is_err() {
            break true;
        }
        if std::time::Instant::now() >= deadline {
            break false;
        }
    };
    drop(client);
    let journal = lab.shutdown();

    assert!(
        closed,
        "the endpoint never closed a connection it could not drain"
    );
    // ⭐ The deterministic half, which holds however the writes were scheduled:
    // the lab stops reading one buffer after the cap, whatever the client sent.
    let recorded = journal.received(&slug("peer-wire")).len();
    assert!(
        recorded <= 4096 + 64 * 1024,
        "it kept reading well past the cap: {recorded} bytes"
    );
}

#[test]
fn a_second_connection_to_one_endpoint_is_one_transcript_with_two_connections() {
    // ⛔ ONE CONNECTION CANNOT ESTABLISH ANYTHING BUT A CONSTANT. A peer ID
    // carries a tail the build regenerates per connection, so a surface dialled
    // once yields one observation and two captures of one build then state two
    // different constants - which `classify_across` reads as `divergent`.
    // `dial_again` is what puts a second observation in the SAME transcript, so
    // the segments differ in their connection identifier and not in their
    // endpoint.
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a listener");
    let address = listener.local_addr().expect("an address");
    let answering = std::thread::spawn(move || {
        for reply in [b"one".to_vec(), b"two".to_vec()] {
            let (mut stream, _) = listener.accept().expect("a connection");
            let mut seen = [0_u8; 5];
            let _ = std::io::Read::read(&mut stream, &mut seen);
            std::io::Write::write_all(&mut stream, &reply).expect("a reply");
        }
    });

    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    lab.dial("peer-dial", address, b"hello".to_vec(), echo_once(b"ok"))
        .expect("the first connection opens");
    lab.dial_again("peer-dial", address, b"hello".to_vec(), echo_once(b"ok"))
        .expect("and so does the second");
    answering.join().expect("the stub answered both");

    assert_eq!(
        lab.endpoints().len(),
        2,
        "a second connection is not a second endpoint: the tracker and the dial"
    );
    lab.stop();
    let journal = lab.shutdown();

    let name = Slug::parse("peer-dial").expect("a canonical name");
    let mut connections: Vec<u64> = journal
        .for_endpoint(&name)
        .iter()
        .filter_map(|segment| segment.connection().map(ConnectionId::get))
        .collect();
    connections.sort_unstable();
    connections.dedup();
    assert_eq!(
        connections.len(),
        2,
        "one endpoint, two connections: {connections:?}"
    );
}

#[test]
#[cfg_attr(
    not(any(target_os = "linux", target_os = "windows")),
    ignore = "loopback here may hold 127.0.0.1 alone; `ifconfig lo0 alias 127.0.0.2` first"
)]
fn a_second_connection_from_another_loopback_address_arrives_from_it() {
    // ⛔ A BUILD CAN KEY ITS PEERS ON THEIR ADDRESS. `aria2-next` 3.0.2 answered
    // one of two connections from `127.0.0.1` and both when the second came
    // from `127.0.0.2`, measured on 2026-10-14. What the target sees as the
    // second connection's source is the fact this asserts, read off the
    // target's own `accept` rather than off the lab's socket.
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a listener");
    let address = listener.local_addr().expect("an address");
    let answering = std::thread::spawn(move || {
        let mut sources = Vec::new();
        for _ in 0..2 {
            let (mut stream, from) = listener.accept().expect("a connection");
            let mut seen = [0_u8; 5];
            let _ = std::io::Read::read(&mut stream, &mut seen);
            std::io::Write::write_all(&mut stream, b"ok").expect("a reply");
            sources.push(from.ip());
        }
        sources
    });

    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    lab.dial("peer-dial", address, b"hello".to_vec(), echo_once(b"ok"))
        .expect("the first connection opens");
    lab.dial_again_from(
        "peer-dial",
        address,
        IpAddr::V4(Ipv4Addr::new(127, 0, 0, 2)),
        b"hello".to_vec(),
        echo_once(b"ok"),
    )
    .expect("and so does the second, from the other address");
    let sources = answering.join().expect("the stub answered both");
    assert_eq!(
        sources,
        [
            IpAddr::V4(Ipv4Addr::LOCALHOST),
            IpAddr::V4(Ipv4Addr::new(127, 0, 0, 2))
        ]
    );
    assert_eq!(
        lab.endpoints().len(),
        2,
        "still one dialled endpoint beside the tracker"
    );
    lab.stop();
    let journal = lab.shutdown();
    let name = Slug::parse("peer-dial").expect("a canonical name");
    let mut connections: Vec<u64> = journal
        .for_endpoint(&name)
        .iter()
        .filter_map(|segment| segment.connection().map(ConnectionId::get))
        .collect();
    connections.sort_unstable();
    connections.dedup();
    assert_eq!(
        connections.len(),
        2,
        "one transcript, two connections: {connections:?}"
    );
}

#[test]
fn a_second_connection_from_off_loopback_is_refused_before_a_packet_leaves() {
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a listener");
    let address = listener.local_addr().expect("an address");
    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    lab.dial("peer-dial", address, b"hello".to_vec(), echo_once(b"ok"))
        .expect("the first connection opens");
    let (_first, _) = listener.accept().expect("the first dial arrives");
    // ⛔ TEST-NET-2 as the source. If the guard stopped firing the socket would
    // try to leave from an address this host should not have.
    let refused = lab.dial_again_from(
        "peer-dial",
        address,
        IpAddr::V4(Ipv4Addr::new(198, 51, 100, 7)),
        b"hello".to_vec(),
        echo_once(b"ok"),
    );
    match refused {
        Err(LabError::Bind(error)) => assert!(error.to_string().contains("loopback")),
        other => panic!("expected a loopback refusal, got {other:?}"),
    }
    listener
        .set_nonblocking(true)
        .expect("a nonblocking listener");
    assert!(
        listener.accept().is_err(),
        "the refused dial opened no connection"
    );
}

#[test]
fn a_second_connection_is_refused_on_an_endpoint_this_lab_never_dialled() {
    // ⚠ `dial_again` is a separate method rather than a relaxation of `dial`'s
    // duplicate check, and this is the half that says so: it will not open a
    // connection to a name the lab does not already hold as a dial.
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a listener");
    let address = listener.local_addr().expect("an address");

    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");

    let refused = lab.dial_again("peer-dial", address, Vec::new(), echo_once(b"ok"));
    assert!(
        matches!(refused, Err(LabError::UnknownEndpoint(_))),
        "{refused:?}"
    );
    // ⛔ And a name the lab DOES hold, as something other than a dial, is
    // refused too: a listener is not a thing another connection can be made to.
    let bound = lab
        .endpoint("tracker-http")
        .expect("the tracker is bound")
        .address();
    let refused = lab.dial_again("tracker-http", bound, Vec::new(), echo_once(b"ok"));
    assert!(
        matches!(refused, Err(LabError::UnknownEndpoint(_))),
        "{refused:?}"
    );

    listener
        .set_nonblocking(true)
        .expect("a nonblocking listener");
    assert!(
        listener.accept().is_err(),
        "neither refusal opened a connection"
    );
}

#[test]
fn a_second_connection_is_refused_at_an_address_the_endpoint_was_not_opened_on() {
    // ⛔ THE MUTATION PASS FOUND THIS ONE. Dropping the address comparison left
    // every case green, because nothing varied the address - so `dial_again`
    // would have opened a connection to somewhere else entirely and filed its
    // segments under an endpoint that names a different peer. An endpoint is a
    // name AND an address; a transcript that mixed two of them would read as one
    // surface observed twice.
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a listener");
    let address = listener.local_addr().expect("an address");
    let elsewhere = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).expect("a second listener");
    let other = elsewhere.local_addr().expect("a second address");
    assert_ne!(address, other, "two listeners, two ports");
    let answering = std::thread::spawn(move || {
        let (mut stream, _) = listener.accept().expect("a connection");
        let mut seen = [0_u8; 5];
        let _ = std::io::Read::read(&mut stream, &mut seen);
        std::io::Write::write_all(&mut stream, b"one").expect("a reply");
    });

    let mut lab = Lab::builder()
        .stream("tracker-http", echo_once(b"ok"))
        .expect("a canonical name")
        .start()
        .expect("loopback binds");
    lab.dial("peer-dial", address, b"hello".to_vec(), echo_once(b"ok"))
        .expect("the first connection opens");
    answering.join().expect("the stub answered");

    let refused = lab.dial_again("peer-dial", other, b"hello".to_vec(), echo_once(b"ok"));
    assert!(
        matches!(refused, Err(LabError::UnknownEndpoint(_))),
        "{refused:?}"
    );
    elsewhere
        .set_nonblocking(true)
        .expect("a nonblocking listener");
    assert!(
        elsewhere.accept().is_err(),
        "the refusal opened no connection to the other address"
    );
}
