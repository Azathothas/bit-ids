//! What a capture's transcripts say the build put on the wire, as fields.
//!
//! `OBS-10`. A capture leaves one `bit-ids/transcript/1` document per surface;
//! this turns the tracker and dialled-peer transcripts into the
//! [`ObservedField`]s a record carries, every sample each one rests on, and the
//! [`SamplingPlan`] the transcripts show. `assemble-capture` is the capture
//! path's caller and `tests/cross_platform_events.rs` is the other: ⛔ **one
//! body, so the test measures the code a record is written with** rather than a
//! copy of it that could drift.
//!
//! ⚠ **Pure over text a caller has already read.** It takes the two documents
//! as strings and touches no file, clock, locale or path, so what it answers is
//! a function of the recorded bytes alone - which is the property a Linux and a
//! Windows compilation have to agree on.
//!
//! ⛔ **Every value is decoded with `bit-ids-wire`'s own codecs**, which are the
//! project's one reading of these formats. A second parser written here would
//! be a second reading, and `FOUND-03` records what that costs.

use std::collections::{BTreeMap, BTreeSet};

use bit_ids::canonical::{HexBytes, Slug};
use bit_ids::observation::{FieldPath, ObservedField};
use bit_ids::sampling::{Sample, SamplingPlan, field_state};
use bit_ids_lab::Journal;
use bit_ids_lab::evidence::parse_transcript_document;
use bit_ids_wire::peer_wire::Handshake;
use bit_ids_wire::tracker_http::HttpRequest;
use bit_ids_wire::tracker_udp::Direction;

/// Every observation of each field, in document order, keyed by field path.
///
/// ⛔ **Corroboration is per OBSERVATION, not per state.** A connector reads the
/// bundle and reports the bytes it saw on each connection; a sampled field has no
/// single value to compare that against. So the observer's own observations are
/// kept beside the state they produced, and `assemble-capture` lines the two
/// lists up. ⚠ The pattern over them is the record's claim; what is corroborated
/// is each observation it rests on.
pub type Readings = BTreeMap<String, Vec<HexBytes>>;

/// What two transcripts normalize to.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Normalized {
    /// One field per path, sorted by path.
    pub fields: Vec<ObservedField>,
    /// Every sample each field rests on, in document order, keyed by path.
    ///
    /// ⚠ The position travels with the value: two fields sampled from one set
    /// of announces have to agree about which session each came from.
    pub samples: BTreeMap<String, Vec<Sample>>,
    /// The widest plan any field shows, each dimension separately.
    pub plan: SamplingPlan,
}

impl Normalized {
    /// The values alone, which is what corroboration compares.
    #[must_use]
    pub fn readings(&self) -> Readings {
        self.samples
            .iter()
            .map(|(path, samples)| {
                (
                    path.clone(),
                    samples.iter().map(|sample| sample.value.clone()).collect(),
                )
            })
            .collect()
    }
}

/// The tracker and dialled-peer transcripts of one capture, as fields.
///
/// `evidence` names the evidence entry a field cites, from the surface's
/// suffix: `tracker` or `peer`.
///
/// ⛔ **EVERY OBSERVATION OF A FIELD IS A SAMPLE, AND THIS USED TO TAKE THE
/// FIRST AND CALL IT `constant`.** A transcript carries a segment per
/// connection, so a build that announced twice put two peer IDs in the evidence
/// and the record said `samples: 1`. The tail after a client prefix is
/// regenerated per connection, so two lanes then disagreed about a field neither
/// of them had measured once - which is `divergent`, and since `E-PUB-04` closed
/// it is what refuses publication.
///
/// ⭐ [`field_state`] is what turns the samples into a state, and the plan is
/// what the transcript shows: a session per separately started process, one
/// torrent, and a connection per sampled segment. ⚠ With one segment it answers
/// `constant` with one sample.
///
/// # Errors
///
/// A sentence naming the transcript and what in it this cannot describe: a
/// document that is not a transcript, an announce that is not an HTTP request,
/// a first segment on a connection that is not a handshake, or a surface the
/// build sent nothing on.
pub fn normalize(
    tracker: &str,
    peer: &str,
    evidence: &dyn Fn(&str) -> Slug,
) -> Result<Normalized, String> {
    let (_, tracker) = parse_transcript_document(tracker)
        .map_err(|error| format!("the tracker transcript: {error}"))?;
    let (_, peer) =
        parse_transcript_document(peer).map_err(|error| format!("the peer transcript: {error}"))?;
    let mut gathered = Gathered::default();
    tracker_observations(&tracker, &mut gathered, evidence)?;
    peer_observations(&tracker, &peer, &mut gathered, evidence)?;
    gathered.fields.sort_by_key(|entry| entry.path.to_string());
    let plan = gathered.plan()?;
    Ok(Normalized {
        fields: gathered.fields,
        samples: gathered.samples,
        plan,
    })
}

/// What the gathering of every field leaves: the fields, the samples each
/// carried, and the widest plan any of them rests on.
///
/// ⭐ **The run's plan is the widest a field shows**, each dimension separately,
/// because a run manifest states one plan and `E-BND-21` holds every field to
/// it; it is still counted off the transcripts, never the attestation's request.
#[derive(Default)]
struct Gathered {
    fields: Vec<ObservedField>,
    samples: BTreeMap<String, Vec<Sample>>,
    sessions: u32,
    connections: u32,
}

impl Gathered {
    fn plan(&self) -> Result<SamplingPlan, String> {
        let nonzero = |value: u32, what: &str| {
            core::num::NonZeroU32::new(value).ok_or_else(|| format!("the run shows 0 {what}"))
        };
        Ok(SamplingPlan {
            sessions: nonzero(self.sessions, "sessions")?,
            torrents: core::num::NonZeroU32::new(1).expect("one is not zero"),
            connections: nonzero(self.connections, "connections")?,
        })
    }

    /// Builds one sampled field and keeps the samples corroboration compares.
    fn push(
        &mut self,
        path: &str,
        values: &[Observation],
        evidence: &dyn Fn(&str) -> Slug,
        suffix: &str,
    ) -> Result<(), String> {
        let (field, samples, plan) = sampled(path, values, evidence(suffix))?;
        self.samples.insert(path.to_owned(), samples);
        self.sessions = self.sessions.max(plan.sessions.get());
        self.connections = self.connections.max(plan.connections.get());
        self.fields.push(field);
        Ok(())
    }
}

/// The announce surface, whose samples are attributed to sessions.
///
/// ⚠ Split from [`peer_observations`] because the two surfaces answer the
/// session question differently, and a reader of either should not have to hold
/// the other's rule in mind to see why.
fn tracker_observations(
    tracker: &Journal,
    gathered: &mut Gathered,
    evidence: &dyn Fn(&str) -> Slug,
) -> Result<(), String> {
    let mut peer_ids = Vec::new();
    let mut agents = Vec::new();
    let mut sessions = Sessions::new();
    for segment in tracker
        .segments()
        .iter()
        .filter(|segment| segment.direction() == Direction::FromTarget)
    {
        let request = HttpRequest::parse(segment.bytes())
            .map_err(|error| format!("the announce is not an HTTP request: {error}"))?;
        // ⚠ A request with no `peer_id` is not an announce - a scrape is the
        // obvious one - so it is not a sample of an announce field. Reading it
        // as one would refuse a capture for carrying a request this record does
        // not describe.
        let queried = request.query_values(b"peer_id");
        let Some(raw) = queried.first() else {
            continue;
        };
        // ⛔ THE SESSION IS READ OUT OF THE ANNOUNCE, NOT OUT OF THE
        // ATTESTATION. BEP 3 makes `started` the first announce of a run, so a
        // second one delimits a second separately started process - which is
        // exactly what `SamplingPlan::sessions` counts. ⚠ Deriving it from the
        // runner's `sessions_started` instead would be this record trusting a
        // claim where a document is available, and it would be wrong whenever a
        // session started and announced nothing.
        let at = sessions.observe(&request);
        peer_ids.push(Observation {
            session: at.session,
            connection: at.connection,
            value: raw
                .decoded_value()
                .map_err(|error| format!("the announce peer_id: {error}"))?
                .ok_or_else(|| "the announce peer_id has no value".to_owned())?,
        });
        // ⛔ Taken from the SAME announces, so the two fields rest on one set of
        // observations. A header sampled from a different set would report two
        // sample counts for one connection.
        let agent = request
            .headers()
            .iter()
            .find(|header| header.name().eq_ignore_ascii_case(b"user-agent"))
            .ok_or_else(|| "the announce carries no User-Agent".to_owned())?;
        agents.push(Observation {
            session: at.session,
            connection: at.connection,
            value: agent.value().to_vec(),
        });
    }
    if peer_ids.is_empty() {
        return Err("the tracker transcript carries nothing the build sent".to_owned());
    }
    gathered.push("tracker_http/peer_id", &peer_ids, evidence, "tracker")?;
    gathered.push("tracker_http/user_agent", &agents, evidence, "tracker")?;
    Ok(())
}

/// The peer surface, whose connections are placed in sessions by the tracker
/// transcript's opening announces.
fn peer_observations(
    tracker: &Journal,
    peer: &Journal,
    gathered: &mut Gathered,
    evidence: &dyn Fn(&str) -> Slug,
) -> Result<(), String> {
    let starts = session_starts(tracker)?;
    let mut handshake_ids = Vec::new();
    let mut reserved = Vec::new();
    let mut opened = Vec::new();
    let mut session = 0_u32;
    let mut index = 0_usize;
    for segment in peer
        .segments()
        .iter()
        .filter(|segment| segment.direction() == Direction::FromTarget)
    {
        // ⛔ The FIRST segment the build sent on each connection must open with
        // a handshake. The lab records one connection per dial and a handshake
        // is what opens one, so a first segment that is not one is a transcript
        // this record cannot describe rather than a segment to pass over.
        // ⚠ A LATER SEGMENT ON THAT CONNECTION IS THE CONVERSATION AFTER IT. This
        // demanded a handshake of every segment until 2026-10-15, when
        // Transmission 4.1.3 sent an unchoke, `0000000101`, 9.5 seconds after its
        // handshake on each connection of `capture-client` run 38667886527, and
        // the pair was refused over five bytes that were never a handshake. The
        // second connector, which finds a handshake by its protocol string,
        // found the same two.
        // ⛔ A segment carrying no connection still has to be a handshake: with
        // nothing to tie it to one, it is not known to follow one.
        if let Some(connection) = segment.connection() {
            if opened.contains(&connection) {
                continue;
            }
            opened.push(connection);
        }
        let handshake = Handshake::parse_prefix(segment.bytes())
            .map_err(|error| format!("the peer handshake: {error}"))?
            .0;
        // ⭐ THE SESSION IS THE ONE WHOSE ANNOUNCE OPENED LAST BEFORE THIS
        // CONNECTION. Nothing in a peer handshake delimits a restart, and this
        // said all connections belonged to one session until 2026-10-15, which
        // was true while `client-capture` dialled once. It dials after every
        // session's opening announce now, so the opening offsets in the tracker
        // transcript, on the same clock as this one, place each connection.
        // ⚠ A connection whose first segment came before any later session's
        // opening belongs to the first, which is the old reading exactly.
        let at = u32::try_from(
            starts
                .iter()
                .filter(|start| **start <= segment.offset_ms())
                .count(),
        )
        .map_err(|error| format!("the peer: {error}"))?;
        if at != session {
            session = at;
            index = 0;
        }
        let connection = u32::try_from(index).map_err(|error| format!("the peer: {error}"))?;
        index += 1;
        handshake_ids.push(Observation {
            session,
            connection,
            value: handshake.peer_id().to_vec(),
        });
        reserved.push(Observation {
            session,
            connection,
            value: handshake.reserved().to_vec(),
        });
    }
    if handshake_ids.is_empty() {
        return Err("the peer transcript carries nothing the build sent".to_owned());
    }
    gathered.push("peer_wire/peer_id", &handshake_ids, evidence, "peer")?;
    gathered.push("peer_wire/reserved", &reserved, evidence, "peer")?;
    Ok(())
}

/// One observation of one field, with where in the plan it came from.
///
/// ⚠ The position travels with the value rather than being recomputed from its
/// index later. Two fields sampled from one set of announces have to agree about
/// which session each came from, and an index into a filtered list agrees with
/// itself and with nothing else.
struct Observation {
    session: u32,
    connection: u32,
    value: Vec<u8>,
}

/// Where in the plan an observation sat.
struct At {
    session: u32,
    connection: u32,
}

/// Walks a run's announces and says which separately started process each
/// belongs to.
///
/// ⛔ **`event=started` IS THE DELIMITER AND BEP 3 IS WHY.** It names the first
/// announce of a run, so a second one is a second process - which is exactly
/// what `SamplingPlan::sessions` counts and the only thing that separates
/// `Lifetime::PerSession` from `Lifetime::Persistent`.
///
/// ⚠ **A build that never sends `started` reads as one session**, which is the
/// conservative answer rather than a wrong one: without a delimiter the run has
/// shown nothing about restarts, and `classify_offset` then answers `Unknown`
/// where it would otherwise answer `PerSession`.
struct Sessions {
    session: u32,
    connection: u32,
    seen: bool,
}

impl Sessions {
    const fn new() -> Self {
        Self {
            session: 0,
            connection: 0,
            seen: false,
        }
    }

    fn observe(&mut self, request: &HttpRequest) -> At {
        let started = request
            .query_values(b"event")
            .first()
            .and_then(|pair| pair.decoded_value().ok().flatten())
            .is_some_and(|value| value == b"started");
        // ⚠ The FIRST announce does not open a new session however it is
        // spelled: it opens the first one. Incrementing on it would number every
        // run's sessions from one and claim a restart that never happened.
        if started && self.seen {
            self.session = self.session.saturating_add(1);
            self.connection = 0;
        } else if self.seen {
            self.connection = self.connection.saturating_add(1);
        }
        self.seen = true;
        At {
            session: self.session,
            connection: self.connection,
        }
    }
}

/// The offsets at which the build's announces opened a second and every later
/// session, by the rule [`Sessions`] reads them with.
///
/// ⛔ Read from the tracker transcript rather than from the attestation's
/// `sessions_started`, for the reason [`Sessions`] gives: a document is
/// available, and the claim would be wrong whenever a session announced nothing.
fn session_starts(tracker: &Journal) -> Result<Vec<u64>, String> {
    let mut sessions = Sessions::new();
    let mut current = 0_u32;
    let mut starts = Vec::new();
    for segment in tracker
        .segments()
        .iter()
        .filter(|segment| segment.direction() == Direction::FromTarget)
    {
        let request = HttpRequest::parse(segment.bytes())
            .map_err(|error| format!("the announce is not an HTTP request: {error}"))?;
        if request.query_values(b"peer_id").is_empty() {
            continue;
        }
        let at = sessions.observe(&request);
        if at.session != current {
            current = at.session;
            starts.push(segment.offset_ms());
        }
    }
    Ok(starts)
}

/// One field, from every observation of it the transcript carries.
///
/// ⛔ **The plan is read off the evidence rather than invented.** One torrent,
/// because the lab generates one; a session per separately started process, read
/// out of the announce events; and a connection per sampled segment within a
/// session. ⚠ A plan claiming more than the transcript shows would let a field
/// rest on samples the run could not have produced, which is `E-BND-21`.
///
/// ⛔ **THE SESSION IS WHAT SEPARATES TWO LIFETIMES AND IT USED TO BE HARDCODED
/// TO ONE.** `classify_offset` answers `PerConnection` for a value that differs
/// within a session and `PerSession` for one that differs only across them, and
/// with every sample filed under session 0 the second is unreachable. ⚠ Run 20
/// measured exactly the value that needs it: `aria2-next` announced twice in one
/// session and carried the SAME peer ID both times, so the tail is per session
/// on that surface and a record claiming `per_connection` would be wrong about
/// the build.
fn sampled(
    path: &str,
    values: &[Observation],
    evidence: Slug,
) -> Result<(ObservedField, Vec<Sample>, SamplingPlan), String> {
    if values.is_empty() {
        return Err(format!("{path}: 0 observation(s)"));
    }
    let mut samples = Vec::new();
    for value in values {
        samples.push(Sample {
            session: value.session,
            torrent: 0,
            connection: value.connection,
            value: HexBytes::new(value.value.clone())
                .map_err(|error| format!("{path}: {error}"))?,
        });
    }
    // ⚠ Counted from the samples rather than taken from the attestation. A run
    // that asked for two sessions and achieved one supports exactly what one
    // supports, and the evidence is what says which happened.
    let sessions = samples
        .iter()
        .map(|sample| sample.session)
        .collect::<BTreeSet<_>>()
        .len();
    // ⛔ The WIDEST session, not the total. `observations()` multiplies the
    // dimensions, so a total here would claim a grid the run never ran - and a
    // narrower one would trip `E-BND-21` on a run whose sessions differ in
    // length.
    let mut widest = 0_usize;
    for session in samples.iter().map(|sample| sample.session) {
        let count = samples
            .iter()
            .filter(|sample| sample.session == session)
            .count();
        widest = widest.max(count);
    }
    let nonzero = |value: usize, what: &str| {
        u32::try_from(value)
            .ok()
            .and_then(core::num::NonZeroU32::new)
            .ok_or_else(|| format!("{path}: {value} {what}"))
    };
    let one = core::num::NonZeroU32::new(1).expect("one is not zero");
    let plan = SamplingPlan {
        sessions: nonzero(sessions, "session(s)")?,
        torrents: one,
        connections: nonzero(widest, "connection(s)")?,
    };
    let state = field_state(&samples, &plan)
        .ok_or_else(|| format!("{path}: no state rests on those observations"))?;
    Ok((
        ObservedField {
            path: FieldPath::parse(path).map_err(|error| format!("{path}: {error}"))?,
            state,
            evidence: vec![evidence],
        },
        samples,
        plan,
    ))
}
