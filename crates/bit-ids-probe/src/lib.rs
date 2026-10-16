//! The observers: the endpoints a build under measurement talks to.
//!
//! Each one is a responder handed to a `bit-ids-lab` endpoint. The lab owns the
//! socket, the deadline and the ordered byte record; this crate owns what the
//! bytes mean and what to answer with, one module per surface.
//!
//! ```text
//! bit-ids-wire   the codecs, and the invariant that a decode loses nothing
//! bit-ids-lab    the sockets, the deadline, the transcript
//! bit-ids-probe  the protocols: what to answer, and what was observed
//! ```
//!
//! ⛔ **No observer maps a peer-ID prefix, a user agent or a client string to a
//! client name.** `docs/capture-methodology.md` lists a decoder table among the
//! inputs that may seed a hypothesis and may not populate the catalogue, and an
//! observer that answered "this is client X" would put that refused input inside
//! the component every measurement passes through.
//!
//! ⚠ **What an observer answers is part of the experiment.** A client that
//! receives the wrong shape of tracker response, or a peer that never completes
//! a handshake, changes what it does next, and that change would be recorded as
//! identity. Each module says what it answers and why.

pub mod dht;
pub mod local_discovery;
pub mod mse;
pub mod normalize;
pub mod peer_exchange;
pub mod peer_wire;
pub mod tracker_http;
pub mod tracker_udp;
pub mod web_seed;

pub use peer_wire::{
    ExtendedOffer, ExtensionProtocol, Offer, PeerIdentity, PeerWire, Presented, Role, Stream,
};
pub use tracker_http::{Announce, HttpTracker, OfferedPeer, TrackerResponse};
pub use tracker_udp::{Observed, Refusal, UdpTracker, UdpTrackerResponse};
