//! How many separately initialized runs a claim about variability needs, and
//! what those runs can actually tell you.
//!
//! One observation establishes bytes for one connection. It does not establish
//! a lifetime. A peer ID that looked the same twice may be stored on disk, or
//! may be regenerated per process and simply have been read twice inside one
//! process. Those are different facts about a build and only a controlled
//! restart separates them.
//!
//! ⛔ **Nothing here is a confidence.** A classification is a function of the
//! samples: it says what the exercised runs prove and returns
//! [`Lifetime::Unknown`] for everything they do not. A dimension the plan never
//! varied cannot yield a conclusion about that dimension, however many samples
//! were taken along the others.

use core::num::NonZeroU32;

use serde::{Deserialize, Serialize};

use crate::canonical::HexBytes;
use crate::observation::{
    BytePattern, ConstantValue, FieldState, PatternRun, PatternedValue, VariableValue,
};

/// What a run varied between samples.
///
/// Each field is how many distinct instances of that dimension the run
/// exercised, so a plan of all ones is a single sample and can conclude
/// nothing about any lifetime.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SamplingPlan {
    /// Separately started processes.
    pub sessions: NonZeroU32,
    /// Distinct torrents offered, within each session.
    pub torrents: NonZeroU32,
    /// Distinct connections made, within each torrent.
    pub connections: NonZeroU32,
}

impl SamplingPlan {
    /// How many observations the plan produces if every combination is taken.
    #[must_use]
    pub const fn observations(&self) -> u64 {
        (self.sessions.get() as u64)
            * (self.torrents.get() as u64)
            * (self.connections.get() as u64)
    }

    /// Whether the plan varied anything at all.
    #[must_use]
    pub const fn varies_anything(&self) -> bool {
        self.sessions.get() > 1 || self.torrents.get() > 1 || self.connections.get() > 1
    }

    /// Whether restarting the process was exercised, which is the only thing
    /// that can distinguish a stored value from a regenerated one.
    #[must_use]
    pub const fn restarts(&self) -> bool {
        self.sessions.get() > 1
    }

    /// What a span behaving this way means under this plan, without the samples.
    ///
    /// ⛔ **THIS IS THE HALF A CONSUMER HOLDING TWO DOCUMENTS WAS MISSING.** A
    /// published [`crate::observation::PatternRun`] carries no lifetime, and
    /// that is `SCHEMA-04`'s own decision rather than an omission: the plan
    /// lives in the run manifest and the claim lives in the profile, so a value
    /// stated in both would be a value that can disagree with itself. ⚠ What
    /// followed from it and was never written down is that a consumer holding
    /// the pair had no way to ASK what a varying span means - it had to
    /// re-reason about `sessions`, `torrents` and `connections` itself, which is
    /// a second implementation of [`classify_offset`] in every consumer.
    ///
    /// ⛔ **IT IS DELIBERATELY WEAKER THAN [`classify`], AND NEVER DISAGREES
    /// WITH IT.** [`classify`] has the samples and can see which grouping a
    /// value is constant within; this has only the plan. So a plan that varied
    /// two dimensions cannot attribute a change to either and answers
    /// [`Lifetime::Unknown`], where the classifier reading real samples may well
    /// separate them. ⚠ The property that matters is that this never returns a
    /// lifetime the classifier would contradict, and a test holds it over every
    /// plan shape.
    ///
    /// `varied` is whether the bytes actually moved, which is what the record's
    /// run kind already says: a `Varying` run is `true` and a `Fixed` one is
    /// `false`.
    #[must_use]
    pub const fn lifetime_of(&self, varied: bool) -> Lifetime {
        if !varied {
            // ⚠ Holding still is only evidence of storage if the run restarted
            // the process. Otherwise every lifetime but per-connection would
            // have produced the same bytes, which is no answer at all.
            return if self.restarts() {
                Lifetime::Persistent
            } else {
                Lifetime::Unknown
            };
        }
        // ⛔ EXACTLY ONE VARIED DIMENSION, OR NOTHING CAN BE ATTRIBUTED. Two
        // varied dimensions both explain a change and the plan alone cannot say
        // which; that is the case the samples settle and this cannot.
        let varied_count = (self.connections.get() > 1) as u8
            + (self.torrents.get() > 1) as u8
            + (self.sessions.get() > 1) as u8;
        if varied_count != 1 {
            return Lifetime::Unknown;
        }
        if self.connections.get() > 1 {
            Lifetime::PerConnection
        } else if self.torrents.get() > 1 {
            Lifetime::PerTorrent
        } else {
            Lifetime::PerSession
        }
    }
}

/// Where one observation came from within the plan.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Sample {
    /// Which process start, counted from zero.
    pub session: u32,
    /// Which torrent within that session, counted from zero.
    pub torrent: u32,
    /// Which connection within that torrent, counted from zero.
    pub connection: u32,
    /// What the build emitted.
    pub value: HexBytes,
}

/// How long a byte stays the same.
///
/// ⚠ These are claims about a build, and each needs the plan to have varied the
/// dimension it names. `Unknown` is not a failure to classify; it is the
/// correct answer when nothing exercised the difference.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Lifetime {
    /// Regenerated for every connection.
    PerConnection,
    /// Stable for one torrent and different for another in the same process.
    PerTorrent,
    /// Stable for one process run and different after a restart.
    PerSession,
    /// Survived a restart, so the build is keeping it somewhere.
    Persistent,
    /// The samples do not separate the cases. ⛔ This is the honest answer for a
    /// single sample, and for any dimension the plan did not vary.
    Unknown,
}

impl Lifetime {
    /// The canonical spelling.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::PerConnection => "per_connection",
            Self::PerTorrent => "per_torrent",
            Self::PerSession => "per_session",
            Self::Persistent => "persistent",
            Self::Unknown => "unknown",
        }
    }

    /// Whether this lifetime asserts that the byte changes.
    #[must_use]
    pub const fn varies(self) -> bool {
        matches!(
            self,
            Self::PerConnection | Self::PerTorrent | Self::PerSession
        )
    }
}

/// A run of adjacent bytes that behaved the same way.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SpanClass {
    /// Where the run starts.
    pub offset: usize,
    /// How many bytes it covers.
    pub length: usize,
    /// What the samples prove about it.
    pub lifetime: Lifetime,
}

/// What a set of samples proves about one value.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct VariabilityReport {
    /// The width every sample shared, or `None` when the width itself varied.
    pub length: Option<usize>,
    /// The value's bytes, grouped into maximal runs that behaved the same way.
    pub spans: Vec<SpanClass>,
    /// How many observations went in.
    pub samples: usize,
}

impl VariabilityReport {
    /// Whether any span was shown to change.
    #[must_use]
    pub fn any_variation(&self) -> bool {
        self.spans.iter().any(|span| span.lifetime.varies())
    }

    /// The span covering one byte offset.
    #[must_use]
    pub fn span_at(&self, offset: usize) -> Option<&SpanClass> {
        self.spans
            .iter()
            .find(|span| offset >= span.offset && offset < span.offset + span.length)
    }
}

/// Whether every sample sharing a key also shares the byte at `offset`.
fn constant_within(
    samples: &[&Sample],
    offset: usize,
    key: impl Fn(&Sample) -> (u32, u32),
) -> bool {
    for (index, sample) in samples.iter().enumerate() {
        for other in &samples[index + 1..] {
            if key(sample) == key(other)
                && sample.value.as_slice()[offset] != other.value.as_slice()[offset]
            {
                return false;
            }
        }
    }
    true
}

/// Whether two samples anywhere differ at `offset`.
fn differs_anywhere(samples: &[&Sample], offset: usize) -> bool {
    let first = samples[0].value.as_slice()[offset];
    samples
        .iter()
        .any(|sample| sample.value.as_slice()[offset] != first)
}

fn classify_offset(samples: &[&Sample], plan: &SamplingPlan, offset: usize) -> Lifetime {
    if !differs_anywhere(samples, offset) {
        // It never changed. That is only a claim about persistence if the run
        // actually restarted the process; otherwise all it shows is that one
        // process kept using the same value, which every lifetime except
        // per-connection would also produce.
        return if plan.restarts() {
            Lifetime::Persistent
        } else {
            Lifetime::Unknown
        };
    }

    // It changed somewhere. Which dimension separates it is decided by the
    // narrowest grouping it is still constant within.
    if !constant_within(samples, offset, |sample| (sample.session, sample.torrent)) {
        return Lifetime::PerConnection;
    }
    if plan.torrents.get() > 1 && !constant_within(samples, offset, |sample| (sample.session, 0)) {
        return Lifetime::PerTorrent;
    }
    if plan.restarts() {
        return Lifetime::PerSession;
    }
    Lifetime::Unknown
}

/// Classifies every byte of a value from the samples that produced it.
///
/// Adjacent bytes that behaved the same way are reported as one span, which is
/// what turns a peer ID into the shape it actually has: a fixed prefix and a
/// suffix the build regenerates.
///
/// # Errors
///
/// Returns `None` when there are no samples. A report over nothing would be a
/// classification nobody measured.
#[must_use]
pub fn classify(samples: &[Sample], plan: &SamplingPlan) -> Option<VariabilityReport> {
    if samples.is_empty() {
        return None;
    }
    let refs: Vec<&Sample> = samples.iter().collect();
    let width = refs[0].value.len();
    let common = refs.iter().all(|sample| sample.value.len() == width);

    if !common {
        // ⚠ Nothing can be said per offset when the offsets do not line up. The
        // width itself varying is the finding.
        return Some(VariabilityReport {
            length: None,
            spans: Vec::new(),
            samples: samples.len(),
        });
    }

    let mut spans: Vec<SpanClass> = Vec::new();
    for offset in 0..width {
        let lifetime = classify_offset(&refs, plan, offset);
        match spans.last_mut() {
            Some(span) if span.lifetime == lifetime => span.length += 1,
            _ => spans.push(SpanClass {
                offset,
                length: 1,
                lifetime,
            }),
        }
    }

    Some(VariabilityReport {
        length: Some(width),
        spans,
        samples: samples.len(),
    })
}

/// Whether two samples anywhere carry a different byte at `offset`.
///
/// ⚠ Per offset rather than per span, which is what the runs below are built
/// from. [`differs_anywhere`] above answers the same question and takes the
/// borrowed form [`classify`] works in.
fn differs_at(samples: &[Sample], offset: usize) -> bool {
    let Some(first) = samples.first() else {
        return false;
    };
    let head = first.value.as_slice()[offset];
    samples
        .iter()
        .any(|sample| sample.value.as_slice()[offset] != head)
}

/// How many distinct values the samples carried.
fn distinct_values(samples: &[Sample]) -> Option<NonZeroU32> {
    let mut seen: Vec<&[u8]> = Vec::new();
    for sample in samples {
        let bytes = sample.value.as_slice();
        if !seen.contains(&bytes) {
            seen.push(bytes);
        }
    }
    u32::try_from(seen.len()).ok().and_then(NonZeroU32::new)
}

/// Appends a fixed run, merging it into the previous one when that is also
/// fixed.
///
/// [`classify`] splits on **lifetime** and this splits on **bytes**, so two
/// adjacent spans can land on one kind here. The record has no spelling for the
/// difference between one fixed run and two touching ones, so emitting two
/// would make the same evidence produce two record shapes.
fn push_fixed(
    runs: &mut Vec<PatternRun>,
    bytes: &[u8],
) -> Result<(), crate::canonical::CanonicalError> {
    if let Some(PatternRun::Fixed { bytes: previous }) = runs.last() {
        let mut joined = previous.as_slice().to_vec();
        joined.extend_from_slice(bytes);
        let merged = HexBytes::new(joined)?;
        runs.pop();
        runs.push(PatternRun::Fixed { bytes: merged });
        return Ok(());
    }
    runs.push(PatternRun::Fixed {
        bytes: HexBytes::new(bytes.to_vec())?,
    });
    Ok(())
}

/// Appends a varying run, merging it into the previous one when that is also
/// varying.
fn push_varying(runs: &mut Vec<PatternRun>, length: usize) {
    if let Some(PatternRun::Varying {
        length: previous, ..
    }) = runs.last_mut()
    {
        *previous += length;
        return;
    }
    runs.push(PatternRun::Varying {
        length,
        // ⚠ An alphabet is a claim about what the build **can** emit, and a
        // handful of runs cannot establish one. `observation.rs` makes it
        // optional for exactly this reason, so it is left unstated rather than
        // inferred from the samples in hand.
        alphabet: None,
    });
}

/// Whether a fixed run between two varying ones is more than two random
/// values agreeing by chance.
///
/// ⛔ **Measured on `capture-client` run 37, 2026-10-14.** Each capture of
/// `aria2-next` 3.0.3 carried two distinct announce peer IDs, and in each the
/// random tails shared one byte at one offset; read per offset, that byte was a
/// fixed run inside the tail, and two routes with such a byte in different
/// places disagreed over bytes neither build fixes.
///
/// ⚠ The rule is the evidence a run of `length` bytes shared by `distinct`
/// values carries: two values match at one offset about one time in
/// sixty-four if a byte draws from that many symbols, so a run is kept once
/// `length × (distinct − 1)` reaches two, about one chance in four thousand.
/// A leading or trailing run is never absorbed, because nothing on its outer
/// side varies; a coincidence at the edge of a varying span is not caught here.
const fn established(length: usize, distinct: u32) -> bool {
    length.saturating_mul(distinct.saturating_sub(1) as usize) >= 2
}

/// Folds every interior fixed run the samples cannot establish into the
/// varying runs around it.
fn absorb_coincidences(runs: Vec<PatternRun>, distinct: u32) -> Vec<PatternRun> {
    let last = runs.len().saturating_sub(1);
    let mut out: Vec<PatternRun> = Vec::new();
    for (index, run) in runs.into_iter().enumerate() {
        match run {
            // ⚠ Runs alternate, because each push merges into a neighbour of its
            // own kind, so a fixed run that is neither first nor last sits
            // between two varying ones.
            PatternRun::Fixed { bytes }
                if index > 0 && index < last && !established(bytes.len(), distinct) =>
            {
                push_varying(&mut out, bytes.len());
            }
            PatternRun::Varying { length, .. } => push_varying(&mut out, length),
            fixed @ PatternRun::Fixed { .. } => out.push(fixed),
        }
    }
    out
}

/// The record state a set of samples supports.
///
/// ⛔ **This is the join `SCHEMA-04` closed without**, and its absence is why
/// the classifier above had no caller that writes a record. [`classify`] says
/// what the samples prove; a record carries a [`FieldState`]. With nothing
/// between them every writer here spelled `constant` with one sample by hand,
/// so a peer ID whose tail is regenerated per connection was recorded as a
/// constant - and two records of one build then disagreed about it, which
/// [`crate::equivalence::classify_across`] reports as `divergent`.
///
/// ⛔ **A run is fixed when the BYTES are identical in every sample, and the
/// runs are derived per OFFSET rather than from [`VariabilityReport::spans`].**
/// Two spans are one run here whenever their bytes behave alike, because
/// [`PatternRun`] carries no lifetime and two shapes for one measurement would
/// be two records. ⚠ And one span can hold both kinds: [`Lifetime::Unknown`]
/// covers a value that never changed **and** a value that changed where no
/// dimension the plan varied separates the change, so a span-wide reading calls
/// eighteen fixed bytes varying because two bytes beside them moved. Measured
/// by a test that did exactly that.
///
/// ⚠ **The lifetimes are deliberately not carried into the record.** The plan
/// lives in the run manifest and the claim lives in the profile, which is
/// `SCHEMA-04`'s own split; what a byte's lifetime is belongs beside the plan
/// that established it, and `bind` is what holds the two documents together.
///
/// ⚠ **The plan is the caller's claim and the bytes are the measurement.**
/// Samples that differ under a plan that varied nothing still report variation
/// here, because the samples are the evidence. That pair is contradictory and
/// `E-BND-20` is what refuses it, against the run manifest.
///
/// # Errors
///
/// Returns `None` when there are no samples, when there are more than
/// [`u32::MAX`] of them, or when a span cannot be represented - each of which is
/// a state nobody measured rather than a state to invent.
#[must_use]
pub fn field_state(samples: &[Sample], plan: &SamplingPlan) -> Option<FieldState> {
    let report = classify(samples, plan)?;
    let count = u32::try_from(report.samples)
        .ok()
        .and_then(NonZeroU32::new)?;

    let Some(width) = report.length else {
        // ⚠ The widths did not line up, so no offset is comparable and there is
        // no span to describe. That is what `variable` with no length says.
        return Some(FieldState::Variable(VariableValue {
            length: None,
            samples: count,
            distinct: distinct_values(samples)?,
        }));
    };

    let first = samples.first()?.value.as_slice().to_vec();
    let mut runs: Vec<PatternRun> = Vec::new();
    for offset in 0..width {
        if differs_at(samples, offset) {
            push_varying(&mut runs, 1);
        } else {
            push_fixed(&mut runs, &first[offset..=offset]).ok()?;
        }
    }
    let runs = absorb_coincidences(runs, distinct_values(samples)?.get());
    let varying: usize = runs
        .iter()
        .map(|run| match run {
            PatternRun::Varying { length, .. } => *length,
            PatternRun::Fixed { .. } => 0,
        })
        .sum();

    if varying == 0 {
        return Some(FieldState::Constant(ConstantValue {
            value: HexBytes::new(first).ok()?,
            samples: count,
        }));
    }
    if varying == width {
        // ⛔ Every byte moved, so there is no fixed run to report and
        // `E-OBS-10` refuses a pattern without one. A value with no stable span
        // is a variable, which is the state that says exactly that.
        return Some(FieldState::Variable(VariableValue {
            length: Some(width),
            samples: count,
            distinct: distinct_values(samples)?,
        }));
    }
    Some(FieldState::Patterned(PatternedValue {
        pattern: BytePattern {
            length: width,
            runs,
        },
        samples: count,
    }))
}
