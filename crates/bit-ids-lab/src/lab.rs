//! The supervisor: what binds, what stops it, and what it hands back.

use core::fmt;
use std::io;
use std::net::{IpAddr, Ipv4Addr, SocketAddr, TcpListener, UdpSocket};
use std::sync::Arc;
use std::thread::JoinHandle;
use std::time::Duration;

use bit_ids::canonical::{CanonicalError, Slug};

use crate::bind::{self, BindError};
use crate::endpoint::{
    ConnectionId, DatagramResponder, Shared, StreamResponder, serve_datagram, serve_dialled,
    serve_stream,
};
use crate::journal::Journal;

/// How long a lab serves before it stops itself.
///
/// A capture that has not finished by then has a client that is not going to
/// answer, and a run that hangs is a CI job that hangs. The value is a default
/// and every lab may set its own.
pub const DEFAULT_DEADLINE: Duration = Duration::from_secs(30);

/// How long a worker waits on a quiet socket before re-checking the deadline.
///
/// It bounds how late a stop is noticed, not how fast the lab answers: a socket
/// with bytes waiting returns immediately.
pub const DEFAULT_POLL: Duration = Duration::from_millis(10);

/// How many connections one stream endpoint serves at once.
pub const DEFAULT_MAX_CONNECTIONS: usize = 64;

/// How long a dial waits before giving up.
pub const DEFAULT_DIAL_TIMEOUT: Duration = Duration::from_secs(5);

/// Which kind of socket an endpoint is.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Transport {
    /// TCP, accepting connections.
    Stream,
    /// UDP.
    Datagram,
    /// TCP, dialled out to a peer.
    ///
    /// A separate variant rather than a flag: a record of what a build did as a
    /// listener and what it did when something connected to it are different
    /// observations, and `OBS-04` exists because clients vary by role.
    Dialled,
}

/// One bound endpoint, and where a client reaches it.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Endpoint {
    name: Slug,
    address: SocketAddr,
    transport: Transport,
}

impl Endpoint {
    /// The name the lab knows this endpoint by, which is also its name in the
    /// journal.
    #[must_use]
    pub const fn name(&self) -> &Slug {
        &self.name
    }

    /// The address the socket reported after binding.
    ///
    /// ⭐ Read back from the socket, never the address that was requested. The
    /// port is one the operating system chose, and a lab that printed what it
    /// asked for would print `:0`.
    #[must_use]
    pub const fn address(&self) -> SocketAddr {
        self.address
    }

    /// TCP or UDP.
    #[must_use]
    pub const fn transport(&self) -> Transport {
        self.transport
    }
}

/// Why a lab did not start.
#[derive(Debug)]
pub enum LabError {
    /// An endpoint name is not a canonical identifier.
    Name(CanonicalError),
    /// Two endpoints were given the same name.
    DuplicateEndpoint(Slug),
    /// Another connection was asked for on an endpoint this lab has not dialled
    /// at that address.
    UnknownEndpoint(Slug),
    /// A lab with no endpoint is a lab nothing can reach.
    NoEndpoints,
    /// The lab has already stopped, so a new connection would serve nothing.
    Stopped,
    /// A stream endpoint was capped at zero connections, which accepts every
    /// connection and closes it at once. That reads to a client as a server
    /// that is up and broken, so it is refused rather than served.
    NoConnectionsAllowed,
    /// A socket was refused. [`BindError::NotLoopback`] is the interesting one.
    Bind(BindError),
    /// The host refused a worker thread.
    Thread(io::Error),
}

impl fmt::Display for LabError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Name(error) => write!(f, "endpoint name: {error}"),
            Self::DuplicateEndpoint(name) => {
                write!(f, "two endpoints are both called {name:?}")
            }
            Self::UnknownEndpoint(name) => {
                write!(
                    f,
                    "this lab has no dialled endpoint {name:?} at that address"
                )
            }
            Self::NoEndpoints => f.write_str("a lab with no endpoint observes nothing"),
            Self::Stopped => f.write_str("the lab has stopped, so nothing would serve this dial"),
            Self::NoConnectionsAllowed => {
                f.write_str("max_connections is zero, so every connection would be closed at once")
            }
            Self::Bind(error) => write!(f, "{error}"),
            Self::Thread(error) => write!(f, "could not start a worker: {error}"),
        }
    }
}

impl core::error::Error for LabError {
    fn source(&self) -> Option<&(dyn core::error::Error + 'static)> {
        match self {
            Self::Name(error) => Some(error),
            Self::Bind(error) => Some(error),
            Self::Thread(error) => Some(error),
            _ => None,
        }
    }
}

impl From<BindError> for LabError {
    fn from(error: BindError) -> Self {
        Self::Bind(error)
    }
}

enum Spec {
    Stream(Slug, Arc<StreamResponder>),
    Datagram(Slug, Arc<DatagramResponder>),
}

impl Spec {
    const fn name(&self) -> &Slug {
        match self {
            Self::Stream(name, _) | Self::Datagram(name, _) => name,
        }
    }
}

enum Bound {
    Stream(Slug, TcpListener, Arc<StreamResponder>),
    Datagram(Slug, UdpSocket, Arc<DatagramResponder>),
}

/// Collects endpoints, then starts them together.
///
/// Nothing binds until [`LabBuilder::start`], so a lab either has every
/// endpoint it was asked for or none of them. A half-started lab is a client
/// pointed at an address that answers for one surface and refuses on another,
/// which reads as a client defect.
pub struct LabBuilder {
    host: IpAddr,
    deadline: Duration,
    poll: Duration,
    max_connections: usize,
    max_pending: usize,
    specs: Vec<Spec>,
}

impl Default for LabBuilder {
    fn default() -> Self {
        Self {
            host: IpAddr::V4(Ipv4Addr::LOCALHOST),
            deadline: DEFAULT_DEADLINE,
            poll: DEFAULT_POLL,
            max_connections: DEFAULT_MAX_CONNECTIONS,
            max_pending: crate::endpoint::DEFAULT_MAX_PENDING_BYTES,
            specs: Vec::new(),
        }
    }
}

impl LabBuilder {
    /// The address every endpoint binds.
    ///
    /// Anything but loopback is refused by [`LabBuilder::start`] rather than
    /// here, so one guard answers for every socket instead of one per entry
    /// point. `crates/bit-ids-lab/src/bind.rs` is that guard.
    #[must_use]
    pub const fn host(mut self, host: IpAddr) -> Self {
        self.host = host;
        self
    }

    /// How long the lab serves before stopping itself.
    #[must_use]
    pub const fn deadline(mut self, deadline: Duration) -> Self {
        self.deadline = deadline;
        self
    }

    /// How long a worker waits on a quiet socket before re-checking.
    #[must_use]
    pub const fn poll(mut self, poll: Duration) -> Self {
        self.poll = poll;
        self
    }

    /// How many connections one stream endpoint serves at once.
    ///
    /// Zero is refused by [`LabBuilder::start`] when the lab has a stream
    /// endpoint, because a zero cap accepts every connection and closes it
    /// immediately, which a client reads as a server that is up and broken.
    #[must_use]
    pub const fn max_connections(mut self, max_connections: usize) -> Self {
        self.max_connections = max_connections;
        self
    }

    /// How many unconsumed bytes one connection may hold before it is closed.
    #[must_use]
    pub const fn max_pending_bytes(mut self, max_pending: usize) -> Self {
        self.max_pending = max_pending;
        self
    }

    /// Adds a TCP endpoint.
    ///
    /// # Errors
    ///
    /// Returns [`LabError::Name`] when `name` is not a canonical identifier.
    pub fn stream<R>(mut self, name: &str, responder: R) -> Result<Self, LabError>
    where
        R: Fn(ConnectionId, &[u8]) -> crate::StreamReply + Send + Sync + 'static,
    {
        let name = Slug::parse(name).map_err(LabError::Name)?;
        self.specs.push(Spec::Stream(name, Arc::new(responder)));
        Ok(self)
    }

    /// Adds a UDP endpoint.
    ///
    /// ⚠ The responder is handed the source address of each packet as well as
    /// its bytes. [`crate::endpoint::DatagramResponder`] says why a surface can
    /// need it and why it is not something to trust.
    ///
    /// # Errors
    ///
    /// Returns [`LabError::Name`] when `name` is not a canonical identifier.
    pub fn datagram<R>(mut self, name: &str, responder: R) -> Result<Self, LabError>
    where
        R: Fn(SocketAddr, &[u8]) -> Option<Vec<u8>> + Send + Sync + 'static,
    {
        let name = Slug::parse(name).map_err(LabError::Name)?;
        self.specs.push(Spec::Datagram(name, Arc::new(responder)));
        Ok(self)
    }

    /// Binds every endpoint and starts serving.
    ///
    /// # Errors
    ///
    /// Returns [`LabError::NoEndpoints`] for an empty lab,
    /// [`LabError::DuplicateEndpoint`] when two endpoints share a name,
    /// [`LabError::Bind`] when a socket is refused, and [`LabError::Thread`]
    /// when the host will not start a worker. On any of them nothing is left
    /// running.
    pub fn start(self) -> Result<Lab, LabError> {
        if self.specs.is_empty() {
            return Err(LabError::NoEndpoints);
        }
        if self.max_connections == 0
            && self
                .specs
                .iter()
                .any(|spec| matches!(spec, Spec::Stream(_, _)))
        {
            return Err(LabError::NoConnectionsAllowed);
        }
        for (index, spec) in self.specs.iter().enumerate() {
            if self.specs[..index]
                .iter()
                .any(|earlier| earlier.name() == spec.name())
            {
                return Err(LabError::DuplicateEndpoint(spec.name().clone()));
            }
        }

        // Every socket is bound before any thread starts, so a refused bind
        // leaves nothing to unwind.
        let mut bound = Vec::with_capacity(self.specs.len());
        let mut endpoints = Vec::with_capacity(self.specs.len());
        for spec in self.specs {
            match spec {
                Spec::Stream(name, responder) => {
                    let listener = bind::stream(self.host)?;
                    let address = listener
                        .local_addr()
                        .map_err(|error| LabError::Bind(BindError::Io(error)))?;
                    endpoints.push(Endpoint {
                        name: name.clone(),
                        address,
                        transport: Transport::Stream,
                    });
                    bound.push(Bound::Stream(name, listener, responder));
                }
                Spec::Datagram(name, responder) => {
                    let socket = bind::datagram(self.host)?;
                    let address = socket
                        .local_addr()
                        .map_err(|error| LabError::Bind(BindError::Io(error)))?;
                    endpoints.push(Endpoint {
                        name: name.clone(),
                        address,
                        transport: Transport::Datagram,
                    });
                    bound.push(Bound::Datagram(name, socket, responder));
                }
            }
        }

        // The clock starts here, so an offset in the journal is measured from
        // the moment the lab began serving rather than from the first bind.
        let shared = Arc::new(Shared::new(
            self.deadline,
            self.poll,
            self.max_connections,
            self.max_pending,
        ));
        let mut workers: Vec<JoinHandle<()>> = Vec::with_capacity(bound.len());
        for one in bound {
            let worker_shared = Arc::clone(&shared);
            let spawned = match one {
                Bound::Stream(name, listener, responder) => std::thread::Builder::new()
                    .name(format!("bit-ids-lab-accept-{name}"))
                    .spawn(move || serve_stream(&worker_shared, &name, listener, &responder)),
                Bound::Datagram(name, socket, responder) => std::thread::Builder::new()
                    .name(format!("bit-ids-lab-datagram-{name}"))
                    .spawn(move || serve_datagram(&worker_shared, &name, socket, &responder)),
            };
            match spawned {
                Ok(worker) => workers.push(worker),
                Err(error) => {
                    // Anything already serving is stopped before the error is
                    // returned, so a failed start leaves no bound port behind.
                    shared.request_stop();
                    for worker in workers {
                        let _ = worker.join();
                    }
                    return Err(LabError::Thread(error));
                }
            }
        }

        Ok(Lab {
            shared,
            endpoints,
            workers,
            dial_timeout: DEFAULT_DIAL_TIMEOUT,
        })
    }
}

/// A running lab.
///
/// Dropping one stops every endpoint and releases every port. A test or a
/// capture that returns early therefore cannot leave a listener behind for the
/// next run to collide with.
pub struct Lab {
    shared: Arc<Shared>,
    endpoints: Vec<Endpoint>,
    workers: Vec<JoinHandle<()>>,
    dial_timeout: Duration,
}

impl Lab {
    /// A builder with the defaults applied.
    #[must_use]
    pub fn builder() -> LabBuilder {
        LabBuilder::default()
    }

    /// Every endpoint, in the order they were added.
    #[must_use]
    pub fn endpoints(&self) -> &[Endpoint] {
        &self.endpoints
    }

    /// One endpoint by name.
    #[must_use]
    pub fn endpoint(&self, name: &str) -> Option<&Endpoint> {
        self.endpoints
            .iter()
            .find(|endpoint| endpoint.name().as_str() == name)
    }

    /// Whether the lab stopped because its deadline passed.
    #[must_use]
    pub fn deadline_expired(&self) -> bool {
        self.shared.expired()
    }

    /// A copy of what has been recorded so far, without stopping the lab.
    #[must_use]
    pub fn journal(&self) -> Journal {
        Journal::from_segments(self.shared.snapshot())
    }

    /// Dials a loopback address and serves `responder` over the connection.
    ///
    /// ⭐ **This is the other role.** `OBS-04` exists because a client can
    /// behave differently as the side that dialled and the side that accepted,
    /// so a lab that could only accept would measure half of the peer surface.
    ///
    /// `opening` is written as soon as the connection is up, which is what the
    /// peer wire needs: the side that dials sends its handshake first, and a
    /// responder is only called once bytes have arrived.
    ///
    /// The address goes through the same guard every bound socket does, so a
    /// dial off loopback is refused before a packet leaves.
    ///
    /// # Errors
    ///
    /// Returns [`LabError::Name`] for a name that is not canonical,
    /// [`LabError::DuplicateEndpoint`] when the lab already has one by that
    /// name, [`LabError::Bind`] when the address is not on loopback or the
    /// connection is refused, and [`LabError::Thread`] when the host will not
    /// start a worker.
    pub fn dial<R>(
        &mut self,
        name: &str,
        address: SocketAddr,
        opening: Vec<u8>,
        responder: R,
    ) -> Result<Endpoint, LabError>
    where
        R: Fn(ConnectionId, &[u8]) -> crate::StreamReply + Send + Sync + 'static,
    {
        // ⛔ Before the syscall. A stopped lab's worker returns at its first
        // check, but `serve_dialled` writes the opening before that check, so a
        // dial on a stopped lab would put a handshake on the wire that nothing
        // was ever going to answer.
        if self.shared.stopped() {
            return Err(LabError::Stopped);
        }
        let name = Slug::parse(name).map_err(LabError::Name)?;
        if self.endpoints.iter().any(|one| one.name() == &name) {
            return Err(LabError::DuplicateEndpoint(name));
        }
        let stream = bind::dial(address, self.dial_timeout)?;
        let endpoint = Endpoint {
            name: name.clone(),
            address,
            transport: Transport::Dialled,
        };

        let shared = Arc::clone(&self.shared);
        let responder: Arc<StreamResponder> = Arc::new(responder);
        let worker = std::thread::Builder::new()
            .name(format!("bit-ids-lab-dial-{name}"))
            .spawn(move || {
                serve_dialled(&shared, &name, stream, &opening, responder.as_ref());
            })
            .map_err(LabError::Thread)?;
        self.workers.push(worker);
        self.endpoints.push(endpoint.clone());
        Ok(endpoint)
    }

    /// Opens **another** connection to an endpoint this lab already dialled.
    ///
    /// ⛔ **One connection cannot establish anything but a constant.** A peer ID
    /// carries a tail the build regenerates per connection, so a surface dialled
    /// once yields one observation, and `SCHEMA-04`'s classifier can only report
    /// what one sample supports. Two connections are what turn that into the
    /// pattern the value actually has - a fixed client prefix and a span that
    /// moves - and a pattern is the only thing two captures of one build can
    /// agree on.
    ///
    /// ⚠ **The segments land in the SAME endpoint**, so they land in the same
    /// transcript, each with its own connection identifier. That is the whole
    /// reason this is not a second endpoint with a second name: a reader of the
    /// evidence gets one document per surface, with a segment per connection,
    /// which is the shape [`Journal`] and the transcript writer already have.
    ///
    /// ⛔ **It is a separate method rather than a relaxation of [`Self::dial`]'s
    /// duplicate check.** That check is what stops two *different* endpoints
    /// sharing a name, and weakening it would buy this at the cost of the thing
    /// it guards. This one asks for another connection to an endpoint that
    /// exists, at the address it was opened on, and refuses anything else.
    ///
    /// # Errors
    ///
    /// Returns [`LabError::Stopped`] on a stopped lab, [`LabError::Name`] for a
    /// name that is not canonical, [`LabError::UnknownEndpoint`] when this lab
    /// has no dialled endpoint of that name at `address`, [`LabError::Bind`]
    /// when the connection is refused, and [`LabError::Thread`] when the host
    /// will not start a worker.
    pub fn dial_again<R>(
        &mut self,
        name: &str,
        address: SocketAddr,
        opening: Vec<u8>,
        responder: R,
    ) -> Result<(), LabError>
    where
        R: Fn(ConnectionId, &[u8]) -> crate::StreamReply + Send + Sync + 'static,
    {
        self.dial_another(name, address, None, opening, responder)
    }

    /// Opens another connection to an endpoint this lab already dialled, from
    /// a loopback source address of the caller's choosing.
    ///
    /// ⛔ **A second connection from the same address may be a connection the
    /// build never answers.** Measured on 2026-10-14: `aria2-next` 3.0.2
    /// answered one of two connections from `127.0.0.1` and both when the
    /// second came from `127.0.0.2`, each with its own peer ID. The source is
    /// part of what the lab offers, so the caller chooses it and records it.
    ///
    /// # Errors
    ///
    /// As [`Self::dial_again`], and [`LabError::Bind`] when the source is not on
    /// loopback, is of the other address family, or is not the address the
    /// socket left from.
    pub fn dial_again_from<R>(
        &mut self,
        name: &str,
        address: SocketAddr,
        source: IpAddr,
        opening: Vec<u8>,
        responder: R,
    ) -> Result<(), LabError>
    where
        R: Fn(ConnectionId, &[u8]) -> crate::StreamReply + Send + Sync + 'static,
    {
        self.dial_another(name, address, Some(source), opening, responder)
    }

    fn dial_another<R>(
        &mut self,
        name: &str,
        address: SocketAddr,
        source: Option<IpAddr>,
        opening: Vec<u8>,
        responder: R,
    ) -> Result<(), LabError>
    where
        R: Fn(ConnectionId, &[u8]) -> crate::StreamReply + Send + Sync + 'static,
    {
        if self.shared.stopped() {
            return Err(LabError::Stopped);
        }
        let name = Slug::parse(name).map_err(LabError::Name)?;
        let known = self.endpoints.iter().any(|one| {
            one.name() == &name && one.address() == address && one.transport() == Transport::Dialled
        });
        if !known {
            return Err(LabError::UnknownEndpoint(name));
        }
        let stream = match source {
            Some(source) => bind::dial_from(source, address, self.dial_timeout)?,
            None => bind::dial(address, self.dial_timeout)?,
        };

        let shared = Arc::clone(&self.shared);
        let responder: Arc<StreamResponder> = Arc::new(responder);
        let worker = std::thread::Builder::new()
            .name(format!("bit-ids-lab-dial-{name}"))
            .spawn(move || {
                serve_dialled(&shared, &name, stream, &opening, responder.as_ref());
            })
            .map_err(LabError::Thread)?;
        self.workers.push(worker);
        // ⚠ No endpoint is pushed. The lab already has this one, and a second
        // entry would make `endpoints()` report a surface twice.
        Ok(())
    }

    /// How long a dial waits before giving up.
    pub const fn set_dial_timeout(&mut self, timeout: Duration) {
        self.dial_timeout = timeout;
    }

    /// Waits until every endpoint has stopped on its own.
    ///
    /// The only thing that stops a lab on its own is the deadline, so this is
    /// how a capture runs one for its whole allotted time. ⚠ It does not ask
    /// the endpoints to stop: calling it on a lab with a long deadline waits
    /// for that deadline.
    pub fn wait(&mut self) {
        self.join_workers();
    }

    /// Asks every endpoint to stop and waits for it, leaving the journal
    /// readable.
    ///
    /// Idempotent, and what [`Lab::shutdown`] and [`Drop`] both use.
    pub fn stop(&mut self) {
        self.shared.request_stop();
        self.join_workers();
    }

    /// Stops every endpoint and returns everything observed.
    #[must_use]
    pub fn shutdown(mut self) -> Journal {
        self.stop();
        Journal::from_segments(self.shared.take_journal())
    }

    fn join_workers(&mut self) {
        for worker in core::mem::take(&mut self.workers) {
            // A worker that panicked has already poisoned nothing that matters:
            // the journal lock is recovered rather than propagated. Joining is
            // still what releases the port.
            let _ = worker.join();
        }
    }
}

impl Drop for Lab {
    fn drop(&mut self) {
        self.stop();
    }
}
