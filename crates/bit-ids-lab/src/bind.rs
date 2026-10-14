//! The one door to a bound socket, and the guard on it.
//!
//! ⛔ **Every socket this crate creates is created here.** A capture points a
//! binary this project downloaded minutes earlier at a network, and the whole
//! containment argument in `docs/capture-host.md` assumes the lab is reachable
//! from the host and from nowhere else. A second module calling
//! `TcpListener::bind` would be a gate on one of two doors into the same
//! action, which `docs/methodology/reviews.md` names as the most recurring hole
//! there is. `tests/lab_supervisor.rs` greps this crate's own source for that,
//! because the rule is otherwise only a comment.
//!
//! The guard runs twice on purpose. A requested address outside loopback is
//! refused before the syscall, and the address the socket actually got is read
//! back and refused again. Those are different facts: a bind request is what
//! was asked for and `local_addr` is what the kernel gave, and only the second
//! one describes where traffic can reach.
//!
//! ⛔ **A bound socket is half the containment.** Where a socket listens and
//! where it sends are different questions, and [`send_to`] is the answer to the
//! second: every outbound datagram this crate emits is addressed here, and a
//! destination outside loopback is refused before the packet exists. `OBS-06`
//! is what needed it, because the adjacent surfaces are precisely the ones
//! whose protocols carry a destination of their own.
//!
//! ⚠ **Writing to a TCP stream is not a further door, and the reason is worth
//! stating.** A `TcpStream` has one peer, fixed when it was accepted or
//! connected, and both of those came through this module. So a write can only
//! reach an address a guard already approved. A datagram is the exception
//! because its destination is chosen per packet, which is why it needs a guard
//! per packet.
//!
//! ⛔ **There is a third door and it is not a socket at all.** [`check_offered`]
//! guards an address the lab puts *inside a message* for the target to contact:
//! a DHT `nodes` or `values` string, a peer-exchange list. Those packets leave
//! the build's socket rather than the lab's, so no guard on this crate's own
//! sockets can see them, and a routable address handed over that way reaches the
//! network just as surely as one this crate sent. `OBS-11` is what needed it.
//!
//! So the three questions are where the lab listens, where the lab sends, and
//! where the lab tells the target to go, and each has its own guard here.

use core::fmt;
use std::io;
use std::net::{IpAddr, SocketAddr, TcpListener, TcpStream, UdpSocket};
use std::time::Duration;

use socket2::{Domain, Protocol, SockAddr, Socket, Type};

/// Why a socket was not created.
#[derive(Debug)]
pub enum BindError {
    /// The requested address is not a loopback address.
    NotLoopback {
        /// What was asked for.
        requested: IpAddr,
    },
    /// The socket bound, and the address it got is not on loopback.
    BoundElsewhere {
        /// What was asked for.
        requested: IpAddr,
        /// What the kernel reported after binding.
        bound: SocketAddr,
    },
    /// An outbound datagram was addressed outside the lab's allowed set.
    ///
    /// ⛔ **This is the one `OBS-06` needs and the bind guards could not
    /// give.** A bind says where a socket listens; it says nothing about
    /// where that socket sends. The adjacent surfaces are exactly the ones
    /// whose protocols name a destination. Local discovery multicasts to a
    /// group fixed by BEP 14, and a DHT's first act is to reach a bootstrap
    /// node, so a lab that guarded only its binds would have contained
    /// every surface except the ones that leave.
    NotReachable {
        /// Where the datagram was addressed.
        destination: SocketAddr,
    },
    /// A dial asked to leave from one address, and the socket left from another.
    ///
    /// ⚠ The source was refused off loopback before the bind. What this
    /// refuses is a measurement condition the lab did not get: a dial from
    /// a second address exists so the build sees a second peer, and a socket the
    /// kernel put back on the first one would show it the same peer twice.
    NotFromSource {
        /// The source address that was asked for.
        requested: IpAddr,
        /// What the kernel reported after connecting.
        bound: SocketAddr,
    },
    /// The operating system refused the bind, the connection or the request.
    Io(io::Error),
}

impl fmt::Display for BindError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NotLoopback { requested } => write!(
                f,
                "refusing to bind {requested}: a lab endpoint is on loopback and nowhere else"
            ),
            Self::BoundElsewhere { requested, bound } => write!(
                f,
                "asked for {requested} and the socket reports {bound}, which is not loopback"
            ),
            Self::NotReachable { destination } => write!(
                f,
                "refusing to send to {destination}: a lab packet goes to loopback and nowhere else"
            ),
            Self::NotFromSource { requested, bound } => write!(
                f,
                "asked to dial from {requested} and the socket reports {bound}"
            ),
            Self::Io(error) => write!(f, "the operating system refused: {error}"),
        }
    }
}

impl core::error::Error for BindError {
    fn source(&self) -> Option<&(dyn core::error::Error + 'static)> {
        match self {
            Self::Io(error) => Some(error),
            _ => None,
        }
    }
}

impl From<io::Error> for BindError {
    fn from(error: io::Error) -> Self {
        Self::Io(error)
    }
}

/// Refuses a requested address that is not on loopback.
///
/// ⚠ An IPv4-mapped IPv6 address such as `::ffff:127.0.0.1` is refused, because
/// [`std::net::Ipv6Addr::is_loopback`] answers for `::1` alone. That is the
/// pessimistic direction and it costs a spelling nobody needs here; the
/// permissive reading would have to unwrap the mapping correctly to stay safe,
/// and being wrong there puts a capture on a routable address.
///
/// # Errors
///
/// Returns [`BindError::NotLoopback`] for anything else, including
/// `0.0.0.0` and `::`, which listen on every interface the host has.
pub const fn check_requested(requested: IpAddr) -> Result<(), BindError> {
    if requested.is_loopback() {
        Ok(())
    } else {
        Err(BindError::NotLoopback { requested })
    }
}

/// Refuses an address a socket actually got that is not on loopback.
///
/// Split out from the bind itself so it can be driven with an address a real
/// socket on this host cannot be made to produce on demand. The bind path calls
/// it with what `local_addr` reported.
///
/// # Errors
///
/// Returns [`BindError::BoundElsewhere`] when `bound` is not on loopback.
pub const fn check_bound(requested: IpAddr, bound: SocketAddr) -> Result<(), BindError> {
    if bound.ip().is_loopback() {
        Ok(())
    } else {
        Err(BindError::BoundElsewhere { requested, bound })
    }
}

/// Binds a TCP listener on loopback, on a port the operating system chooses.
///
/// Port zero is not a convenience. Two labs on one host that both named a port
/// collide, and the second one fails in whichever of them happens to start
/// second, which is a flake rather than a defect report.
///
/// # Errors
///
/// Returns [`BindError::NotLoopback`] before the syscall,
/// [`BindError::BoundElsewhere`] after it, and [`BindError::Io`] when the
/// operating system refuses.
pub fn stream(requested: IpAddr) -> Result<TcpListener, BindError> {
    check_requested(requested)?;
    let listener = TcpListener::bind(SocketAddr::new(requested, 0))?;
    check_bound(requested, listener.local_addr()?)?;
    Ok(listener)
}

/// Dials a loopback address, with a bounded wait.
///
/// ⛔ **A dial goes through this guard for the reason a bind does.** The module
/// documentation says every socket this crate creates is created here, and an
/// outbound connection is a socket: an observer that dialled wherever it was
/// told would reach off the host from inside a lab whose whole argument is that
/// it cannot. `tests/lab_supervisor.rs` greps for it.
///
/// ⚠ The timeout is not a convenience. `TcpStream::connect` to an address that
/// drops rather than refuses waits on the operating system's own retry
/// schedule, which outlasts any capture deadline, and the lab's deadline cannot
/// interrupt a thread blocked in a syscall.
///
/// # Errors
///
/// Returns [`BindError::NotLoopback`] before the syscall,
/// [`BindError::BoundElsewhere`] when the connected peer is not on loopback,
/// and [`BindError::Io`] when the connection is refused or times out.
pub fn dial(address: SocketAddr, timeout: Duration) -> Result<TcpStream, BindError> {
    check_requested(address.ip())?;
    let stream = TcpStream::connect_timeout(&address, timeout)?;
    // Read back, for the reason the bind path reads back: what was asked for
    // and what the kernel connected are different facts.
    check_bound(address.ip(), stream.peer_addr()?)?;
    Ok(stream)
}

/// Dials a loopback address from a chosen loopback source, with a bounded wait.
///
/// ⛔ **A build can key its peers on the address they come from, so the lab
/// needs more than one.** Measured on 2026-10-14 against `aria2-next` 3.0.2 in
/// a namespace holding loopback alone: two connections from `127.0.0.1` got one
/// handshake back, and the same two with the second from `127.0.0.2` got two,
/// each with its own peer ID. libtorrent's `allow_multiple_connections_per_ip`
/// is `false` by default, and a lab whose every connection came from one
/// address could never show a value the build regenerates per connection.
///
/// ⚠ **The standard library cannot bind before it connects**, which is why
/// this one door uses `socket2`; `docs/supply-chain.md` carries the decision.
/// The rest of this module is `std::net` and stays so.
///
/// The guards are the dial's, and one more: the source and the destination
/// are both refused off loopback before the syscall, the peer is read back as
/// [`dial`] reads it, and so is the address the socket left from, because a
/// source the kernel did not honour is a second peer the build never saw.
///
/// # Errors
///
/// Returns [`BindError::NotLoopback`] before the syscall for either address,
/// [`BindError::BoundElsewhere`] when the connected peer is not on loopback,
/// [`BindError::NotFromSource`] when the socket left from another address, and
/// [`BindError::Io`] when the families differ or the connection is refused or
/// times out.
pub fn dial_from(
    source: IpAddr,
    address: SocketAddr,
    timeout: Duration,
) -> Result<TcpStream, BindError> {
    check_requested(source)?;
    check_requested(address.ip())?;
    // ⚠ As `TcpStream::connect_timeout` answers it, so the two doors agree.
    if timeout.is_zero() {
        return Err(BindError::Io(io::Error::new(
            io::ErrorKind::InvalidInput,
            "a dial cannot wait for no time at all",
        )));
    }
    if source.is_ipv4() != address.is_ipv4() {
        return Err(BindError::Io(io::Error::new(
            io::ErrorKind::InvalidInput,
            format!("a dial from {source} cannot reach {address}: the families differ"),
        )));
    }
    let socket = Socket::new(
        Domain::for_address(address),
        Type::STREAM,
        Some(Protocol::TCP),
    )?;
    socket.bind(&SockAddr::from(SocketAddr::new(source, 0)))?;
    socket.connect_timeout(&SockAddr::from(address), timeout)?;
    let stream = TcpStream::from(socket);
    check_bound(address.ip(), stream.peer_addr()?)?;
    let left = stream.local_addr()?;
    if left.ip() != source {
        return Err(BindError::NotFromSource {
            requested: source,
            bound: left,
        });
    }
    Ok(stream)
}

/// Sends one datagram, to a destination inside the lab's allowed address set.
///
/// ⛔ **The door sweep found this one open.** Every socket went through this
/// module and the reply path did not: `endpoint::serve_datagram` called
/// `send_to` on the bound socket directly, with the source address of whatever
/// had just arrived. A UDP source address is a claim by the sender rather than
/// a fact the kernel checked, so a local process able to forge one could have
/// aimed a lab's replies off the host, from a socket the loopback guard had
/// already approved. That is the shape `docs/methodology/reviews.md` names: a
/// gate on one of several doors into the same action.
///
/// ⚠ **The guard here is the check before the syscall, and that is all it can
/// be.** A bind and a dial each read back what the kernel actually did, because
/// a listener has a local address and a connected stream has a peer. A datagram
/// socket has neither for the packet it just sent: there is no
/// after-the-fact reading of where a datagram went. So the destination must be
/// correct before it is passed, and this is the only place that decides.
///
/// # Errors
///
/// Returns [`BindError::NotReachable`] before the syscall for any destination
/// outside loopback, and [`BindError::Io`] when the operating system refuses
/// the send.
pub fn send_to(
    socket: &UdpSocket,
    payload: &[u8],
    destination: SocketAddr,
) -> Result<usize, BindError> {
    if !destination.ip().is_loopback() {
        return Err(BindError::NotReachable { destination });
    }
    Ok(socket.send_to(payload, destination)?)
}

/// Refuses an address the lab would hand the target as somewhere to go.
///
/// ⛔ **The third door, and `OBS-11` is what needed it.** The first two are
/// where the lab listens and where the lab sends. This is where the lab tells
/// the *target* to go, and it is a door neither of the others can cover: a DHT
/// `find_node` response carries a `nodes` string and a `get_peers` response
/// carries `values`, and both are lists of addresses the build will then dial
/// itself. Those packets leave the build's socket, not the lab's, so
/// [`send_to`] never sees them and the loopback guard on the lab's own sockets
/// is no protection at all.
///
/// ⚠ **`adjacent::reaches` already named the shape and named it on the wrong
/// surface.** It says `pex` "hands out peer addresses a client will then dial",
/// which is exactly this, and a DHT response does the same thing through a
/// different field. A hazard recorded against one surface and not the sibling
/// that shares it is the one-gated-door defect `docs/methodology/reviews.md`
/// names.
///
/// ⚠ This guard is a check and never a rewrite. An address outside the set is
/// refused so the caller has to decide what to offer instead; silently
/// substituting loopback would put an address in a transcript that the observer
/// invented, and the transcript is the measurement.
///
/// # Errors
///
/// Returns [`BindError::NotReachable`] for any address outside loopback.
pub const fn check_offered(address: SocketAddr) -> Result<(), BindError> {
    if address.ip().is_loopback() {
        Ok(())
    } else {
        Err(BindError::NotReachable {
            destination: address,
        })
    }
}

/// Binds a UDP socket on loopback, on a port the operating system chooses.
///
/// # Errors
///
/// The same three as [`stream`].
pub fn datagram(requested: IpAddr) -> Result<UdpSocket, BindError> {
    check_requested(requested)?;
    let socket = UdpSocket::bind(SocketAddr::new(requested, 0))?;
    check_bound(requested, socket.local_addr()?)?;
    Ok(socket)
}

#[cfg(test)]
mod tests {
    use super::{BindError, check_bound, check_requested, datagram, stream};
    use std::net::{IpAddr, Ipv4Addr, Ipv6Addr, SocketAddr};
    use std::time::Duration;

    fn v4(a: u8, b: u8, c: u8, d: u8) -> IpAddr {
        IpAddr::V4(Ipv4Addr::new(a, b, c, d))
    }

    #[test]
    fn loopback_is_accepted_in_both_families() {
        assert!(check_requested(v4(127, 0, 0, 1)).is_ok());
        assert!(check_requested(v4(127, 9, 9, 9)).is_ok());
        assert!(check_requested(IpAddr::V6(Ipv6Addr::LOCALHOST)).is_ok());
    }

    #[test]
    fn the_wildcard_and_a_routable_address_are_refused() {
        for requested in [
            v4(0, 0, 0, 0),
            v4(192, 168, 1, 10),
            v4(8, 8, 8, 8),
            IpAddr::V6(Ipv6Addr::UNSPECIFIED),
            // ⚠ The mapped form of a loopback address. Refused, and the doc
            // comment says why that is the direction to be wrong in.
            IpAddr::V6(Ipv6Addr::new(0, 0, 0, 0, 0, 0xffff, 0x7f00, 1)),
        ] {
            assert!(
                matches!(
                    check_requested(requested),
                    Err(BindError::NotLoopback { .. })
                ),
                "{requested} was not refused"
            );
        }
    }

    #[test]
    fn an_address_the_socket_reports_off_loopback_is_refused_after_binding() {
        let bound = SocketAddr::new(v4(192, 168, 1, 10), 6881);
        assert!(matches!(
            check_bound(v4(127, 0, 0, 1), bound),
            Err(BindError::BoundElsewhere { .. })
        ));
        let ok = SocketAddr::new(v4(127, 0, 0, 1), 6881);
        assert!(check_bound(v4(127, 0, 0, 1), ok).is_ok());
    }

    #[test]
    fn a_bound_socket_reports_a_loopback_address_and_a_chosen_port() {
        let listener = stream(v4(127, 0, 0, 1)).expect("loopback binds");
        let address = listener
            .local_addr()
            .expect("a bound listener has an address");
        assert!(address.ip().is_loopback());
        assert_ne!(address.port(), 0, "port zero means the kernel chose one");

        let socket = datagram(v4(127, 0, 0, 1)).expect("loopback binds");
        let address = socket.local_addr().expect("a bound socket has an address");
        assert!(address.ip().is_loopback());
        assert_ne!(address.port(), 0);
    }

    #[test]
    fn neither_binder_reaches_the_syscall_for_a_non_loopback_address() {
        assert!(matches!(
            stream(v4(0, 0, 0, 0)),
            Err(BindError::NotLoopback { .. })
        ));
        assert!(matches!(
            datagram(v4(0, 0, 0, 0)),
            Err(BindError::NotLoopback { .. })
        ));
    }

    #[test]
    fn dialling_off_loopback_is_refused_before_a_packet_leaves() {
        // ⛔ 198.51.100.0/24 is TEST-NET-2, reserved for documentation. If this
        // guard ever stopped firing the test would try to reach it, which is
        // the failure being prevented, so it must be an address that goes
        // nowhere rather than one that answers.
        let elsewhere = SocketAddr::new(v4(198, 51, 100, 7), 6881);
        assert!(matches!(
            super::dial(elsewhere, Duration::from_millis(50)),
            Err(BindError::NotLoopback { .. })
        ));
    }

    #[test]
    #[cfg_attr(
        not(any(target_os = "linux", target_os = "windows")),
        ignore = "loopback here may hold 127.0.0.1 alone; `ifconfig lo0 alias 127.0.0.2` first"
    )]
    fn a_dial_from_a_second_loopback_address_reaches_the_listener_from_it() {
        // ⛔ The measurement this exists for: the listener sees the source the
        // dial asked for, not the address the kernel would have chosen.
        let listener = stream(v4(127, 0, 0, 1)).expect("loopback binds");
        let address = listener.local_addr().expect("a bound listener has one");
        let dialled = super::dial_from(v4(127, 0, 0, 2), address, Duration::from_secs(5))
            .expect("it connects");
        let (_accepted, from) = listener.accept().expect("the dial arrives");
        assert_eq!(from.ip(), v4(127, 0, 0, 2));
        assert_eq!(
            dialled
                .local_addr()
                .expect("a connected stream has one")
                .ip(),
            v4(127, 0, 0, 2)
        );
    }

    #[test]
    fn a_dial_from_off_loopback_is_refused_before_a_packet_leaves() {
        // ⚠ The destination is a real listener, so a refusal here is the
        // source's own guard and not a connection that failed.
        let listener = stream(v4(127, 0, 0, 1)).expect("loopback binds");
        let address = listener.local_addr().expect("a bound listener has one");
        assert!(matches!(
            super::dial_from(v4(198, 51, 100, 7), address, Duration::from_millis(50)),
            Err(BindError::NotLoopback { .. })
        ));
        let elsewhere = SocketAddr::new(v4(198, 51, 100, 7), 6881);
        assert!(matches!(
            super::dial_from(v4(127, 0, 0, 2), elsewhere, Duration::from_millis(50)),
            Err(BindError::NotLoopback { .. })
        ));
        // ⚠ The kernel would refuse this bind too; what is asserted is that the
        // refusal says why, which only the family check does.
        match super::dial_from(
            IpAddr::V6(Ipv6Addr::LOCALHOST),
            address,
            Duration::from_millis(50),
        ) {
            Err(BindError::Io(error)) => assert!(error.to_string().contains("families differ")),
            other => panic!("expected a family refusal, got {other:?}"),
        }
    }

    #[test]
    fn dialling_a_loopback_listener_connects_and_reads_the_peer_back() {
        let listener = stream(v4(127, 0, 0, 1)).expect("loopback binds");
        let address = listener.local_addr().expect("a bound listener has one");
        let dialled = super::dial(address, Duration::from_secs(5)).expect("it connects");
        assert!(
            dialled
                .peer_addr()
                .expect("a connected stream has a peer")
                .ip()
                .is_loopback()
        );
    }

    /// ⛔ The egress-negative case `OBS-06` asks for, over the addresses the
    /// adjacent protocols actually name.
    #[test]
    fn a_datagram_addressed_outside_loopback_never_leaves() {
        let socket = datagram(v4(127, 0, 0, 1)).expect("loopback binds");
        // The BEP 14 groups, the ones a local-discovery observer would be
        // handed by its own specification; 198.51.100.0/24 is TEST-NET-2,
        // standing in for a DHT bootstrap node without naming somebody's host.
        let elsewhere = [
            SocketAddr::new(v4(239, 192, 152, 143), 6771),
            SocketAddr::new(
                IpAddr::V6(Ipv6Addr::new(0xff15, 0, 0, 0, 0, 0, 0xefc0, 0x988f)),
                6771,
            ),
            SocketAddr::new(v4(198, 51, 100, 7), 6881),
            SocketAddr::new(v4(255, 255, 255, 255), 6771),
            SocketAddr::new(v4(0, 0, 0, 0), 6881),
        ];
        for destination in elsewhere {
            assert!(
                matches!(
                    super::send_to(&socket, b"BT-SEARCH * HTTP/1.1\r\n", destination),
                    Err(BindError::NotReachable { .. })
                ),
                "{destination} was not refused"
            );
        }
    }

    #[test]
    fn a_datagram_addressed_to_loopback_arrives_and_reports_its_length() {
        let listener = datagram(v4(127, 0, 0, 1)).expect("loopback binds");
        let address = listener.local_addr().expect("a bound socket has one");
        let sender = datagram(v4(127, 0, 0, 1)).expect("loopback binds");
        let payload = b"BT-SEARCH * HTTP/1.1\r\n\r\n\r\n";
        let sent = super::send_to(&sender, payload, address).expect("loopback is allowed");
        assert_eq!(sent, payload.len());

        listener
            .set_read_timeout(Some(Duration::from_secs(5)))
            .expect("a read timeout");
        let mut buffer = [0_u8; 64];
        let (read, from) = listener
            .recv_from(&mut buffer)
            .expect("the datagram arrives");
        assert_eq!(&buffer[..read], payload);
        assert!(from.ip().is_loopback());
    }

    /// ⛔ The third door, tested where it lives.
    ///
    /// ⚠ **This test exists because a mutation pass found the guard proved only
    /// from another crate.** Blanking [`super::check_offered`] to accept
    /// everything was refused by two cases in `bit-ids-probe` and by nothing
    /// here, so the module that owns the rule had no case of its own and a
    /// reader of this file would have found the guard unproven. A guard is
    /// tested beside the code that decides it, not only beside the code that
    /// happens to call it today.
    #[test]
    fn an_address_offered_to_the_target_is_refused_outside_the_allowed_set() {
        use super::check_offered;

        // The addresses the adjacent protocols actually name, which is what a
        // DHT response or a peer-exchange list would otherwise carry off-host.
        // 198.51.100.0/24 is TEST-NET-2 and 82.221.103.244 is a real bootstrap
        // node's address literal, present to be refused rather than reached.
        for offered in [
            SocketAddr::new(v4(198, 51, 100, 7), 6881),
            SocketAddr::new(v4(82, 221, 103, 244), 6881),
            SocketAddr::new(v4(0, 0, 0, 0), 6881),
            SocketAddr::new(v4(239, 192, 152, 143), 6771),
            SocketAddr::new(IpAddr::V6(Ipv6Addr::UNSPECIFIED), 6881),
        ] {
            assert!(
                matches!(check_offered(offered), Err(BindError::NotReachable { .. })),
                "{offered} was not refused"
            );
        }

        // ⚠ And the control, so a guard that refused everything would fail too.
        assert!(check_offered(SocketAddr::new(v4(127, 0, 0, 1), 6881)).is_ok());
        assert!(check_offered(SocketAddr::new(IpAddr::V6(Ipv6Addr::LOCALHOST), 6881)).is_ok());
    }

    #[test]
    fn dialling_a_closed_loopback_port_fails_rather_than_hanging() {
        // The port is released before the dial, so nothing is listening on it.
        let port = {
            let listener = stream(v4(127, 0, 0, 1)).expect("loopback binds");
            listener.local_addr().expect("an address").port()
        };
        let address = SocketAddr::new(v4(127, 0, 0, 1), port);
        assert!(matches!(
            super::dial(address, Duration::from_secs(5)),
            Err(BindError::Io(_))
        ));
    }
}
