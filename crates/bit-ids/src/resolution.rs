//! Choosing the newest stable release, and keeping the reasoning.
//!
//! ⛔ **Version strings are not comparable in general, and a resolver that
//! pretends otherwise picks the wrong build silently.** Sorting tags as text
//! puts `4.1.10` before `4.1.9` and `release-5.2.3` before `release-5.2.10`. A
//! channel label is no better: a project can publish a preview without setting
//! the flag, and one that does set it can still tag a release the label calls
//! stable and the version string calls a beta.
//!
//! So nothing here guesses. A target declares how it spells versions, the
//! resolver compares only what that scheme can order, and anything it cannot
//! order **blocks the resolution** rather than being skipped.
//!
//! ⭐ **The skip is the dangerous case and it is worth being explicit about.**
//! An unrecognised tag that is quietly ignored does not produce an error; it
//! produces an older version, selected confidently, with nothing in the record
//! saying a newer one was seen and not understood. Every candidate is kept with
//! the verdict it got, so the resolution is a decision anyone can re-derive.
//!
//! ⚠ **Stability is judged pessimistically, by both signals.** A candidate that
//! either source or version text calls a prerelease is not stable. Being wrong
//! that way costs a skipped release; being wrong the other way publishes a
//! preview build as the stable one, which is a measurement about a build no
//! user runs.

use core::fmt;
use std::collections::BTreeSet;

use serde::{Deserialize, Serialize};

use crate::canonical::{Instant, Label, Sha256Digest, Slug, Url, Version};
use crate::json::DocumentError;
use crate::validate::{SchemaError, Violations, strictly_ascending};

/// Identifier carried by every second-generation resolution.
///
/// ⚠ **The second generation adds the scheme floor** and the verdict it gives,
/// `below_scheme_floor`. A first-generation reader refuses both by
/// `deny_unknown_fields`, which is why this is a new identifier rather than a
/// field slipped into the old one. `CLIENT-06`.
pub const RESOLUTION_SCHEMA: &str = "bit-ids/resolution/2";

/// The schema identifier a resolution declares.
///
/// ⭐ A type rather than a validated `String`, which is what `ManifestSchema`
/// already does. A wrong identifier is then unrepresentable rather than
/// refused, and the invariant that checked for one was a guard `from_json`
/// could never reach: the version probe answers first. It was written, found
/// unreachable while planting a defect for it, and deleted, which is the same
/// path `E-BND-10` took in `SCHEMA-02`.
#[derive(Clone, Debug, Eq, PartialEq, Ord, PartialOrd, Hash)]
pub struct ResolutionSchema(&'static str);

impl ResolutionSchema {
    /// The schema this build reads and writes.
    #[must_use]
    pub const fn current() -> Self {
        Self(RESOLUTION_SCHEMA)
    }

    /// Parses a declared schema identifier.
    ///
    /// # Errors
    ///
    /// Returns an error for any identifier other than [`RESOLUTION_SCHEMA`].
    pub fn parse(text: &str) -> Result<Self, crate::canonical::CanonicalError> {
        if text == RESOLUTION_SCHEMA {
            Ok(Self(RESOLUTION_SCHEMA))
        } else {
            Err(crate::canonical::CanonicalError::new(
                "resolution-schema-version",
                format!("unsupported schema {text:?}, this build reads {RESOLUTION_SCHEMA:?}"),
            ))
        }
    }

    /// The declared identifier.
    #[must_use]
    pub const fn as_str(&self) -> &'static str {
        self.0
    }
}

impl Serialize for ResolutionSchema {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(self.0)
    }
}

impl<'de> Deserialize<'de> for ResolutionSchema {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        use serde::de::Error as _;
        let text = String::deserialize(deserializer)?;
        Self::parse(&text).map_err(D::Error::custom)
    }
}

/// Tokens that mark a version as something other than a stable release.
///
/// ⚠ Matched as whole tokens, not as substrings. `rc` inside a longer word is
/// not a release candidate, and a rule that matched anywhere would call a
/// version carrying `march` a prerelease.
const PRERELEASE_TOKENS: &[&str] = &[
    "alpha", "beta", "rc", "pre", "preview", "dev", "nightly", "snapshot", "test", "canary",
    "insider", "eap", "unstable",
];

/// How a target spells the versions it publishes.
///
/// Product-specific by construction. `qbittorrent` tags `release-5.2.3` and
/// `transmission` tags `4.1.3`, and there is no general rule that reads both
/// without also reading things that are not versions at all.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct VersionScheme {
    /// A literal prefix stripped before parsing, such as `release-`.
    pub tag_prefix: Option<Label>,
    /// The fewest dot-separated numeric components a version may have.
    pub min_components: u8,
    /// The most it may have. Enhanced editions publish four; most publish three.
    pub max_components: u8,
    /// ⭐ The first major version this scheme speaks for, when the target
    /// declares one.
    ///
    /// A tag the scheme cannot order and whose leading component is a plain
    /// integer below this is the target's own history in another spelling, so
    /// it is released as [`Verdict::BelowSchemeFloor`] rather than blocking.
    /// Measured on 2026-10-15: `transmission`'s 132 refs carry 112 tags this
    /// scheme cannot read, from `0.5` to `3.00`, every one of them before the
    /// three-component spelling began at `4.0.0`, and refs carry no date for
    /// [`Verdict::PredatesSelection`] to use.
    ///
    /// ⛔ **It releases only against a selection at or above it.** A floor of 4
    /// over a winner of 3.x says nothing about a `3.99` it cannot read, so those
    /// stay unorderable and block, which is `E-RES-10` one layer down.
    pub floor_major: Option<u64>,
}

impl VersionScheme {
    /// Strips the prefix, or reports that the tag is not this target's.
    fn strip<'a>(&self, tag: &'a str) -> Option<&'a str> {
        match &self.tag_prefix {
            Some(prefix) => tag.strip_prefix(prefix.as_str()),
            None => Some(tag),
        }
    }

    /// The numeric components, when the text is orderable under this scheme,
    /// padded to the scheme's width so two spellings of one release compare
    /// equal.
    ///
    /// ⛔ **This is the project's only version ordering, and every caller uses
    /// it.** `resolve` picks the newest release to acquire, `CORPUS-03` picks
    /// the newest record to point at, and `CI-02` asks whether the first is
    /// newer than the second. Those are different questions over one
    /// comparison, and a second implementation of it would answer one of them
    /// differently on the day it drifted, with the version that reads wrongly
    /// being the one a consumer follows.
    ///
    /// ⚠ This sentence used to say "both callers" and name two. A third arrived
    /// and the count went stale in the one place a reader checks before writing
    /// a fourth implementation. It says "every" now, because a number in prose
    /// is a value in two places with nothing comparing them.
    ///
    /// Returns `None` for text this scheme cannot order, which is what lets a
    /// caller block rather than guess.
    #[must_use]
    pub fn components(&self, text: &str) -> Option<Vec<u64>> {
        let parts: Vec<&str> = text.split('.').collect();
        let count = u8::try_from(parts.len()).ok()?;
        if count < self.min_components || count > self.max_components {
            return None;
        }
        let mut components: Vec<u64> = parts
            .iter()
            .map(|part| {
                // ⚠ A component with a leading zero is refused rather than
                // parsed. `1.01` and `1.1` are two spellings that compare equal,
                // and an append-only store cannot tell those apart later.
                if part.is_empty() || (part.len() > 1 && part.starts_with('0')) {
                    return None;
                }
                part.parse::<u64>().ok()
            })
            .collect::<Option<Vec<u64>>>()?;
        // ⛔ Padded to the scheme's width before comparing. Without this, `4.1`
        // and `4.1.0` compare as different versions with the longer one newer,
        // because a shorter vector sorts first. They are the same release, and
        // treating them as an ordering rather than an ambiguity is the silent
        // wrong answer this module exists to refuse. Found by writing the test
        // for the ambiguous case and watching it select instead.
        components.resize(usize::from(self.max_components), 0);
        Some(components)
    }

    /// Whether a tag's text sits below this scheme's floor: a declared floor,
    /// and a leading dot-component that is a plain integer smaller than it.
    ///
    /// ⚠ A plain integer as [`Self::components`] reads one, so `03.1` and
    /// `3x.1` are not below anything and stay unorderable. Reading them
    /// generously is the guess this module refuses.
    #[must_use]
    pub fn below_floor(&self, text: &str) -> bool {
        let Some(floor) = self.floor_major else {
            return false;
        };
        let lead = text.split('.').next().unwrap_or_default();
        if lead.is_empty()
            || !lead.bytes().all(|b| b.is_ascii_digit())
            || (lead.len() > 1 && lead.starts_with('0'))
        {
            return false;
        }
        lead.parse::<u64>().is_ok_and(|major| major < floor)
    }
}

/// Whether a version's own text marks it as a prerelease.
///
/// Case-insensitive and token-bounded: every run of letters inside a token is
/// asked, so `rc1`, `beta5` and `0beta1` all carry a marker and `march` does
/// not.
///
/// ⛔ **A run of letters AFTER DIGITS is asked too, and was not until
/// 2026-10-15.** The reader took letters from a token's start only, so
/// `5.3.0beta1`, whose last token is `0beta1`, read as no prerelease at all.
/// Measured that day: qBittorrent's 260 refs carry 59 such tags, from
/// `release-1.0.0rc10` to `release-5.3.0rc1`, and every one was `unorderable`,
/// so the source route failed closed on them. The release route never met
/// them, because the listing flags them. Over every target's live tags the
/// change moves those 59 and nothing else. `CLIENT-01`.
/// ⚠ Pessimistic by design, as the module says: `rc1a` is a prerelease here.
/// Being wrong that way costs a skipped release.
///
/// ⛔ **And PEP 440's one-letter markers, `a`, `b` and `c`, between two
/// digits, since 2026-10-16.** `2.0.0b1` is Python's own spelling of a beta,
/// and a single letter is asked only where a digit stands on both sides of it,
/// so `x86`, `1.0b` and `b1.0` carry none. Measured that day over every
/// target's live refs: Deluge's `deluge-2.0.0b1` and `deluge-2.0b1` were
/// `unorderable` and failed its source route closed, and they move to a
/// prerelease. Transmission's thirteen `1.80b5` to `2.50b1` betas move from
/// `below_scheme_floor` to a prerelease, which is what they are. Nothing else
/// moves anywhere and no selection changes. `CLIENT-07`.
#[must_use]
pub fn text_marks_prerelease(text: &str) -> bool {
    let lowered = text.to_ascii_lowercase();
    let tokens = lowered.split(|c: char| !c.is_ascii_alphanumeric());
    tokens.filter(|token| !token.is_empty()).any(|token| {
        token
            .split(|c: char| !c.is_ascii_alphabetic())
            .filter(|letters| !letters.is_empty())
            .any(|letters| PRERELEASE_TOKENS.contains(&letters))
            || pep440_letter(token)
    })
}

/// Whether a token carries one of PEP 440's one-letter prerelease markers with
/// a digit on each side of it, as `0b1`, `0a2` and `1c1` do.
fn pep440_letter(token: &str) -> bool {
    let bytes = token.as_bytes();
    bytes.windows(3).any(|w| {
        w[0].is_ascii_digit() && matches!(w[1], b'a' | b'b' | b'c') && w[2].is_ascii_digit()
    })
}

/// What the resolver decided about one candidate, and why.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Verdict {
    /// The newest stable release, and the resolution's answer.
    Selected,
    /// The tag does not carry this target's prefix, so it is not this
    /// target's release.
    ForeignTag,
    /// The publishing source flagged it a prerelease or a draft.
    PrereleaseByLabel,
    /// The version text carries a prerelease token.
    PrereleaseByVersion,
    /// ⛔ The version cannot be ordered under this target's scheme. It blocks
    /// the resolution rather than being skipped, because nothing here can rule
    /// out that it is newer than the candidate that would otherwise win.
    Unorderable,
    /// Stable, orderable, and older than the selection.
    Superseded,
    /// ⭐ Could not be ordered by version, and was published before the
    /// candidate that won, so it cannot be the newest.
    ///
    /// This is what keeps a project's own history from blocking every run. A
    /// live dry run against `transmission` found 51 candidates in exactly this
    /// position: two-component tags from a decade ago that the current scheme
    /// cannot read. Refusing over them is correct only if the resolver has no
    /// second signal, and publication order is one, so nothing is guessed and
    /// the reason is on the record.
    PredatesSelection,
    /// ⭐ Could not be ordered by version, and its leading component is below
    /// the floor the scheme declares, so it cannot be newer than a selection at
    /// or above that floor. See [`VersionScheme::floor_major`].
    BelowSchemeFloor,
    /// Compares equal to another candidate and is spelled differently, so
    /// "newest" has no single answer.
    Ambiguous,
    /// Its source's newest stable disagrees with another source's.
    Divergent,
}

impl Verdict {
    /// Every variant, so a test can hold the two spellings in step.
    pub const ALL: &'static [Self] = &[
        Self::Selected,
        Self::ForeignTag,
        Self::PrereleaseByLabel,
        Self::PrereleaseByVersion,
        Self::Unorderable,
        Self::Superseded,
        Self::PredatesSelection,
        Self::BelowSchemeFloor,
        Self::Ambiguous,
        Self::Divergent,
    ];

    /// The canonical spelling, for a message a person reads.
    ///
    /// ⚠ This is a second spelling of the vocabulary serde already derives, so
    /// `resolution_verdict_spellings_agree_with_the_serialized_form` holds the
    /// two together. A name in two places with nothing comparing them is the
    /// drift this repository refuses everywhere else, and a door sweep found
    /// this pair unchecked.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Selected => "selected",
            Self::ForeignTag => "foreign_tag",
            Self::PrereleaseByLabel => "prerelease_by_label",
            Self::PrereleaseByVersion => "prerelease_by_version",
            Self::Unorderable => "unorderable",
            Self::Superseded => "superseded",
            Self::PredatesSelection => "predates_selection",
            Self::BelowSchemeFloor => "below_scheme_floor",
            Self::Ambiguous => "ambiguous",
            Self::Divergent => "divergent",
        }
    }
}

impl fmt::Display for Verdict {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

/// One release a source offered, before any judgement.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Candidate {
    /// Which source offered it.
    pub source: Slug,
    /// The tag exactly as published.
    pub tag: Label,
    /// The source's own prerelease flag.
    pub prerelease: bool,
    /// The source's own draft flag.
    pub draft: bool,
    /// When the source says it was published, when it says.
    pub published_at: Option<Instant>,
}

/// A candidate with the verdict it got and the version that was read out of it.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Considered {
    /// What was offered.
    pub candidate: Candidate,
    /// What was decided.
    pub verdict: Verdict,
    /// The version extracted from the tag, absent when none could be.
    pub version: Option<Version>,
}

/// The exact bytes one source answered with.
///
/// ⛔ A selection is only re-derivable if the input is recorded. The digest is
/// of the response as it arrived, before any parsing, so a later reader can
/// tell a resolver defect from a source that changed its answer.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SourceResponse {
    /// Source identifier, unique within the resolution.
    pub id: Slug,
    /// Where it was asked.
    pub url: Url,
    /// When it answered, UTC.
    pub retrieved_at: Instant,
    /// Digest of the bytes that arrived.
    pub digest: Sha256Digest,
    /// How many candidates were read out of it.
    pub candidates: u32,
}

/// One run of the resolver, with everything it looked at.
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct Resolution {
    /// The schema identifier, read before anything else.
    pub schema: ResolutionSchema,
    /// Which target this resolves.
    pub target: Slug,
    /// When the decision was made, UTC.
    pub resolved_at: Instant,
    /// How this target spells its versions.
    pub scheme: VersionScheme,
    /// The sources asked, and what they answered with.
    pub sources: Vec<SourceResponse>,
    /// The newest stable version, or absent when the resolver failed closed.
    pub selected: Option<Version>,
    /// Every candidate considered, in the order the sources offered them.
    pub considered: Vec<Considered>,
}

/// The same shape with the derive on it, so [`Resolution`]'s own `Deserialize`
/// validates. Same construction as `Profile`, and for the same reason.
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ResolutionFields {
    schema: ResolutionSchema,
    target: Slug,
    resolved_at: Instant,
    scheme: VersionScheme,
    sources: Vec<SourceResponse>,
    selected: Option<Version>,
    considered: Vec<Considered>,
}

impl From<ResolutionFields> for Resolution {
    fn from(fields: ResolutionFields) -> Self {
        Self {
            schema: fields.schema,
            target: fields.target,
            resolved_at: fields.resolved_at,
            scheme: fields.scheme,
            sources: fields.sources,
            selected: fields.selected,
            considered: fields.considered,
        }
    }
}

impl<'de> Deserialize<'de> for Resolution {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        use serde::de::Error as _;
        let resolution = Self::from(ResolutionFields::deserialize(deserializer)?);
        validate_resolution(&resolution).map_err(D::Error::custom)?;
        Ok(resolution)
    }
}

impl Resolution {
    /// Reads and validates a resolution document.
    ///
    /// # Errors
    ///
    /// Returns [`DocumentError::UnsupportedSchema`] for another generation,
    /// [`DocumentError::Malformed`] when it is not this shape, and
    /// [`DocumentError::Invalid`] when it parsed and refused an invariant.
    pub fn from_json(document: &str) -> Result<Self, DocumentError> {
        #[derive(Deserialize)]
        struct VersionProbe {
            schema: String,
        }

        let probe: VersionProbe =
            serde_json::from_str(document).map_err(DocumentError::Malformed)?;
        if probe.schema != RESOLUTION_SCHEMA {
            return Err(DocumentError::UnsupportedSchema {
                found: probe.schema,
                expected: RESOLUTION_SCHEMA,
            });
        }
        let fields: ResolutionFields =
            serde_json::from_str(document).map_err(DocumentError::Malformed)?;
        let resolution = Self::from(fields);
        validate_resolution(&resolution).map_err(DocumentError::Invalid)?;
        Ok(resolution)
    }

    /// Writes the resolution in the canonical form.
    ///
    /// # Errors
    ///
    /// Returns [`DocumentError::Invalid`] with the refused invariants. An
    /// invalid resolution has no canonical form and is not written.
    pub fn to_json(&self) -> Result<String, DocumentError> {
        validate_resolution(self).map_err(DocumentError::Invalid)?;
        let mut out = serde_json::to_string_pretty(self).map_err(DocumentError::Malformed)?;
        out.push('\n');
        Ok(out)
    }

    /// Whether the resolver reached an answer.
    #[must_use]
    pub const fn resolved(&self) -> bool {
        self.selected.is_some()
    }

    /// Every candidate that got one verdict.
    #[must_use]
    pub fn with_verdict(&self, verdict: Verdict) -> Vec<&Considered> {
        self.considered
            .iter()
            .filter(|entry| entry.verdict == verdict)
            .collect()
    }
}

/// Decides the newest stable release from what the sources offered.
///
/// ⛔ **It fails closed.** `selected` is absent whenever the answer is not
/// forced: no stable candidate at all, a candidate that could not be ordered,
/// two spellings that compare equal, or two sources that disagree. Every one of
/// those keeps its reason in the trace.
#[must_use]
pub fn resolve(
    target: Slug,
    resolved_at: Instant,
    scheme: VersionScheme,
    sources: Vec<SourceResponse>,
    candidates: Vec<Candidate>,
) -> Resolution {
    let mut considered: Vec<Considered> = candidates
        .into_iter()
        .map(|candidate| {
            let (verdict, version) = judge(&scheme, &candidate);
            Considered {
                candidate,
                verdict,
                version,
            }
        })
        .collect();
    // ⭐ One construction site. An earlier version returned a whole
    // `Resolution` from each fail-closed branch, and five copies of the same
    // struct literal is five places for a field to be forgotten.
    let selected = decide(&scheme, &sources, &mut considered);
    Resolution {
        schema: ResolutionSchema::current(),
        target,
        resolved_at,
        scheme,
        sources,
        selected,
        considered,
    }
}

/// The decision itself. `None` is the fail-closed answer, and every path to it
/// leaves the reason in a verdict.
fn decide(
    scheme: &VersionScheme,
    sources: &[SourceResponse],
    considered: &mut [Considered],
) -> Option<Version> {
    let (best_components, best_version) = considered
        .iter()
        .filter(|entry| entry.verdict == Verdict::Superseded)
        .filter_map(|entry| Some((components(scheme, entry)?, entry.version.clone()?)))
        .max_by(|left, right| left.0.cmp(&right.0))?;

    let equal = equal_indices(considered, scheme, &best_components);

    // Two spellings that compare equal make "newest" ambiguous. Order is
    // checked, never imposed, which is the same rule one layer up.
    let spellings: BTreeSet<&str> = equal
        .iter()
        .filter_map(|index| considered[*index].version.as_ref().map(Version::as_str))
        .collect();
    if spellings.len() > 1 {
        mark(considered, &equal, Verdict::Ambiguous);
        return None;
    }

    // Every source that offered a stable candidate must have offered this one.
    // A source whose own newest differs is a divergence, not a tie to break.
    let divergent: Vec<usize> = sources
        .iter()
        .filter_map(|source| {
            let (found, index) = considered
                .iter()
                .enumerate()
                .filter(|(_, entry)| {
                    entry.candidate.source == source.id && entry.verdict == Verdict::Superseded
                })
                .filter_map(|(index, entry)| Some((components(scheme, entry)?, index)))
                .max_by(|left, right| left.0.cmp(&right.0))?;
            (found != best_components).then_some(index)
        })
        .collect();
    if !divergent.is_empty() {
        mark(considered, &equal, Verdict::Divergent);
        mark(considered, &divergent, Verdict::Divergent);
        return None;
    }

    // ⭐ The floor is the first second signal, and it needs no date. ⛔ Only
    // against a winner at or above it: below it, the floor says nothing about
    // what an unreadable tag might be.
    if scheme
        .floor_major
        .is_some_and(|floor| best_components.first().is_some_and(|major| *major >= floor))
    {
        let below: Vec<usize> = considered
            .iter()
            .enumerate()
            .filter(|(_, entry)| entry.verdict == Verdict::Unorderable)
            .filter(|(_, entry)| {
                scheme
                    .strip(entry.candidate.tag.as_str())
                    .is_some_and(|text| scheme.below_floor(text))
            })
            .map(|(index, _)| index)
            .collect();
        mark(considered, &below, Verdict::BelowSchemeFloor);
    }

    // ⛔ An unorderable candidate blocks unless a second signal rules it out.
    // Publication order is that signal: one published strictly before the
    // winner cannot be newer than it, whatever its tag says. One with no date,
    // or a date at or after the winner's, still blocks.
    let Some(best_published) = considered
        .iter()
        .filter(|entry| entry.verdict == Verdict::Superseded)
        .filter(|entry| components(scheme, entry).as_deref() == Some(&best_components))
        .find_map(|entry| entry.candidate.published_at.clone())
    else {
        return blocked_by_unorderable(considered)
            .then_some(())
            .map_or_else(
                || {
                    mark(considered, &equal, Verdict::Selected);
                    Some(best_version.clone())
                },
                |()| None,
            );
    };
    let predating: Vec<usize> = considered
        .iter()
        .enumerate()
        .filter(|(_, entry)| entry.verdict == Verdict::Unorderable)
        .filter(|(_, entry)| {
            entry
                .candidate
                .published_at
                .as_ref()
                .is_some_and(|published| *published < best_published)
        })
        .map(|(index, _)| index)
        .collect();
    mark(considered, &predating, Verdict::PredatesSelection);

    if blocked_by_unorderable(considered) {
        return None;
    }
    mark(considered, &equal, Verdict::Selected);
    Some(best_version)
}

fn blocked_by_unorderable(considered: &[Considered]) -> bool {
    considered
        .iter()
        .any(|entry| entry.verdict == Verdict::Unorderable)
}

fn mark(considered: &mut [Considered], indices: &[usize], verdict: Verdict) {
    for index in indices {
        considered[*index].verdict = verdict;
    }
}

fn equal_indices(considered: &[Considered], scheme: &VersionScheme, target: &[u64]) -> Vec<usize> {
    considered
        .iter()
        .enumerate()
        .filter(|(_, entry)| entry.verdict == Verdict::Superseded)
        .filter(|(_, entry)| components(scheme, entry).as_deref() == Some(target))
        .map(|(index, _)| index)
        .collect()
}

fn components(scheme: &VersionScheme, entry: &Considered) -> Option<Vec<u64>> {
    scheme.components(entry.version.as_ref()?.as_str())
}

/// The per-candidate rules, in the order they are asked.
///
/// ⚠ Stability before orderability. A long tail of `x.y.z-beta.n` tags would
/// otherwise be unorderable and block every resolution, when what they actually
/// are is not-stable and irrelevant.
fn judge(scheme: &VersionScheme, candidate: &Candidate) -> (Verdict, Option<Version>) {
    let Some(text) = scheme.strip(candidate.tag.as_str()) else {
        return (Verdict::ForeignTag, None);
    };
    let version = Version::parse(text).ok();
    if candidate.prerelease || candidate.draft {
        return (Verdict::PrereleaseByLabel, version);
    }
    if text_marks_prerelease(text) {
        return (Verdict::PrereleaseByVersion, version);
    }
    if scheme.components(text).is_none() {
        return (Verdict::Unorderable, version);
    }
    // `Superseded` is the provisional verdict for every orderable stable
    // candidate; `resolve` promotes the winner. Starting from "selected" and
    // demoting would leave a record claiming several selections if a later rule
    // returned early.
    (Verdict::Superseded, version)
}

/// The invariants a resolution document must satisfy.
///
/// # Errors
///
/// Returns the refusals, each with a stable code.
pub fn validate_resolution(resolution: &Resolution) -> Result<(), Violations> {
    let mut out = Vec::new();
    check_sources(resolution, &mut out);
    check_selection(resolution, &mut out);
    check_floor(resolution, &mut out);
    Violations::from_errors(out)
}

/// ⛔ A `below_scheme_floor` verdict is re-derived, never trusted: the scheme
/// recorded beside it must release that very tag, and the selection must stand
/// at or above the floor that released it.
fn check_floor(resolution: &Resolution, out: &mut Vec<SchemaError>) {
    let scheme = &resolution.scheme;
    let mut released = false;
    for (index, entry) in resolution.considered.iter().enumerate() {
        if entry.verdict != Verdict::BelowSchemeFloor {
            continue;
        }
        released = true;
        // ⚠ And one the scheme could order is `superseded`, never released:
        // the floor speaks only for what nothing else can read. Found by a
        // review handing the validator a readable `3.0.0` marked this way.
        let below = scheme
            .strip(entry.candidate.tag.as_str())
            .is_some_and(|text| scheme.components(text).is_none() && scheme.below_floor(text));
        if !below {
            out.push(SchemaError::new(
                "E-RES-09",
                format!("considered[{index}].verdict"),
                format!(
                    "{} is marked below the scheme floor and the recorded scheme does not put it there",
                    entry.candidate.tag
                ),
            ));
        }
    }
    let Some(floor) = scheme.floor_major else {
        return;
    };
    // ⚠ Nothing selected is no contradiction: a floor can release history and
    // a tag at or above it still block, which is the fail-closed answer with
    // both reasons on the record. Found by driving a floor of 3 over
    // `transmission`'s refs, whose `3.00` stays unorderable.
    let Some(selected) = &resolution.selected else {
        return;
    };
    if !released {
        return;
    }
    let major = scheme
        .components(selected.as_str())
        .and_then(|components| components.first().copied());
    if major.is_none_or(|major| major < floor) {
        out.push(SchemaError::new(
            "E-RES-10",
            "selected",
            format!(
                "candidates were released by a floor of {floor} and the selection does not stand at or above it"
            ),
        ));
    }
}

fn check_sources(resolution: &Resolution, out: &mut Vec<SchemaError>) {
    if resolution.sources.is_empty() {
        out.push(SchemaError::new(
            "E-RES-01",
            "sources",
            "a resolution with no source asked nothing",
        ));
    }
    if let Some(index) = strictly_ascending(&resolution.sources, |source| source.id.to_string()) {
        out.push(SchemaError::new(
            "E-RES-02",
            format!("sources[{index}]"),
            format!(
                "source ids must be unique and ascending, found {}",
                resolution.sources[index].id
            ),
        ));
    }
    for (index, entry) in resolution.considered.iter().enumerate() {
        if !resolution
            .sources
            .iter()
            .any(|source| source.id == entry.candidate.source)
        {
            out.push(SchemaError::new(
                "E-RES-03",
                format!("considered[{index}].candidate.source"),
                format!("{} is not among the sources asked", entry.candidate.source),
            ));
        }
    }
    for (index, source) in resolution.sources.iter().enumerate() {
        let counted = resolution
            .considered
            .iter()
            .filter(|entry| entry.candidate.source == source.id)
            .count();
        if u64::from(source.candidates) != counted as u64 {
            out.push(SchemaError::new(
                "E-RES-04",
                format!("sources[{index}].candidates"),
                format!(
                    "{} declares {} candidate(s) and {counted} are carried",
                    source.id, source.candidates
                ),
            ));
        }
        // A decision cannot predate the answer it was made from.
        if resolution.resolved_at < source.retrieved_at {
            out.push(SchemaError::new(
                "E-RES-05",
                format!("sources[{index}].retrieved_at"),
                format!(
                    "{} answered at {} and the decision is stamped {}",
                    source.id, source.retrieved_at, resolution.resolved_at
                ),
            ));
        }
    }
}

fn check_selection(resolution: &Resolution, out: &mut Vec<SchemaError>) {
    let selected: Vec<&Considered> = resolution
        .considered
        .iter()
        .filter(|entry| entry.verdict == Verdict::Selected)
        .collect();
    match (&resolution.selected, selected.as_slice()) {
        (None, []) => {}
        (Some(version), []) => out.push(SchemaError::new(
            "E-RES-07",
            "selected",
            format!("selects {version} and no candidate is marked selected"),
        )),
        (None, entries) => out.push(SchemaError::new(
            "E-RES-07",
            "selected",
            format!(
                "selects nothing and {} candidate(s) are marked selected",
                entries.len()
            ),
        )),
        (Some(version), entries) => {
            for entry in entries {
                if entry.version.as_ref() != Some(version) {
                    out.push(SchemaError::new(
                        "E-RES-06",
                        "selected",
                        format!(
                            "the document selects {version} and {} is marked selected",
                            entry.candidate.tag
                        ),
                    ));
                }
            }
        }
    }
    // ⛔ The rule the whole module exists for. A selection standing beside a
    // candidate nobody could order is a version chosen without ruling out a
    // newer one, which is the silent-skip defect this fails closed on.
    if resolution.selected.is_some()
        && resolution
            .considered
            .iter()
            .any(|entry| entry.verdict == Verdict::Unorderable)
    {
        out.push(SchemaError::new(
            "E-RES-08",
            "selected",
            "a candidate could not be ordered, so nothing may be selected",
        ));
    }
}

/// One artifact attached to a release, exactly as the source listed it.
///
/// ⛔ **The location comes out of the listing and is never composed.** A URL
/// built here from an owner, a repository, a tag and a file name would be a
/// second derivation of something the source already states, and the day a
/// vendor reorganises its downloads the composed one goes on answering - with
/// nothing, or with bytes that are not the release's. The size is kept beside
/// it because it is the cheapest thing a fetch can be checked against.
/// ⭐ **And the digest the source publishes of the bytes it serves**, where it
/// publishes one. `ACQ-06`. It is `None` for a source that states none, which is
/// a fact about that source rather than a missing field: measured through
/// `AGENTS.md` rule 8's route on 2026-09-17, three of this project's four targets
/// carry one on every asset and `aria2/aria2` carries none on any.
///
/// ⛔ **IT IS THE HOST'S STATEMENT, NOT THE VENDOR'S, AND THE RECORD SAYS WHICH.**
/// The party publishing this digest is the same party serving the bytes, so it
/// establishes that nothing changed between reading the index and fetching the
/// artifact - and says nothing that would survive that index being wrong. A
/// vendor's own checksums document is a different claim, and `digest_source`
/// exists so a reader is never told one when the other is what happened.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ReleaseAsset {
    /// The file name the source published it under.
    pub name: Label,
    /// Where the source says the bytes are.
    pub url: Url,
    /// How many bytes the source says it is.
    pub size: u64,
    /// What the source says the bytes digest to, where it says anything.
    pub digest: Option<Sha256Digest>,
}

/// Why no single asset could be chosen.
///
/// ⛔ **Three refusals rather than one, because the fix differs.** A missing
/// release means the resolution and the listing disagree about what exists; no
/// match means the pattern is wrong or the vendor renamed an artifact; and an
/// ambiguous match means the pattern is too wide, which is the one a caller
/// would otherwise never learn about, because picking the first match hides it.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum AssetRefusal {
    /// The listing carries no release under that tag.
    NoSuchRelease,
    /// Nothing the release carries matches the pattern.
    NoMatch,
    /// More than one asset matched, so "the artifact" has no single answer.
    Ambiguous(Vec<Label>),
}

impl fmt::Display for AssetRefusal {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NoSuchRelease => f.write_str("the listing carries no release under that tag"),
            Self::NoMatch => f.write_str("no asset of that release matches the pattern"),
            Self::Ambiguous(names) => {
                write!(f, "{} assets match the pattern:", names.len())?;
                for name in names {
                    write!(f, " {name}")?;
                }
                Ok(())
            }
        }
    }
}

/// One piece of an [`AssetPattern`].
#[derive(Clone, Debug, Eq, PartialEq)]
enum PatternPart {
    /// Text that must appear exactly.
    Literal(String),
    /// Any run of characters, including none.
    AnyRun,
}

/// A pattern over an asset's file name, held by the target that knows it.
///
/// Two constructs and nothing else: `{version}` stands for the version the
/// resolver selected, and `*` matches any run of characters including none.
/// Everything else is literal, and an unrecognised `{...}` is refused rather
/// than matched as text, because a typo that matched literally would be a
/// pattern that silently stops finding anything.
///
/// ⛔ **The version is expanded into a LITERAL and never spliced into the
/// glob.** [`Version`] accepts what a build printed, `*` included, so a pattern
/// built by string substitution could be widened by the very value it was meant
/// to pin. Expanding into a literal part cannot.
///
/// ⭐ **`{version}` is what makes a pattern narrow enough to be unambiguous on
/// a real vendor's release.** Measured on 2026-09-09 against qBittorrent 5.2.3,
/// whose release carries fourteen assets and **two** Linux `AppImage` files:
/// `qbittorrent-5.2.3_x86_64.AppImage` and
/// `qbittorrent-5.2.3_lt20_x86_64.AppImage`. So `qbittorrent-*_x86_64.AppImage`
/// matches both and `qbittorrent-{version}_x86_64.AppImage` matches one.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AssetPattern {
    parts: Vec<PatternPart>,
}

impl AssetPattern {
    /// Reads a pattern, expanding `{version}` with the selected version.
    ///
    /// # Errors
    ///
    /// Returns a message when the pattern is empty, carries an unterminated
    /// `{`, or names a placeholder other than `version`.
    pub fn parse(pattern: &str, version: &str) -> Result<Self, String> {
        if pattern.is_empty() {
            return Err("an asset pattern is not empty".to_owned());
        }
        let mut parts: Vec<PatternPart> = Vec::new();
        let mut literal = String::new();
        let mut rest = pattern;
        while !rest.is_empty() {
            if let Some(after) = rest.strip_prefix('*') {
                if !literal.is_empty() {
                    parts.push(PatternPart::Literal(core::mem::take(&mut literal)));
                }
                // ⚠ Adjacent stars collapse here rather than in the matcher, so
                // the matcher never has to reason about an empty run between two.
                if parts.last() != Some(&PatternPart::AnyRun) {
                    parts.push(PatternPart::AnyRun);
                }
                rest = after;
            } else if let Some(after) = rest.strip_prefix('{') {
                let Some(end) = after.find('}') else {
                    return Err(format!("{pattern:?} has an unterminated placeholder"));
                };
                let name = &after[..end];
                if name != "version" {
                    return Err(format!(
                        "{pattern:?} names placeholder {{{name}}}; the only one is {{version}}"
                    ));
                }
                literal.push_str(version);
                rest = &after[end + 1..];
            } else {
                // ⚠ `rest` begins with neither construct here, so the next
                // occurrence is at least one byte in and this always advances.
                let next = rest.find(['*', '{']).unwrap_or(rest.len());
                literal.push_str(&rest[..next]);
                rest = &rest[next..];
            }
        }
        if !literal.is_empty() {
            parts.push(PatternPart::Literal(literal));
        }
        Ok(Self { parts })
    }

    /// Whether a file name matches.
    ///
    /// ⚠ Linear and backtracking-free: the pattern is anchored at both ends by
    /// its own outermost parts, and every literal between two runs is then found
    /// left to right. A recursive matcher over a name a vendor chose is a place
    /// to spend exponential time on an input this project does not control.
    #[must_use]
    pub fn matches(&self, name: &str) -> bool {
        let mut segments: Vec<&str> = Vec::new();
        let mut anchored_start = true;
        let mut anchored_end = true;
        for (at, part) in self.parts.iter().enumerate() {
            match part {
                PatternPart::Literal(text) => segments.push(text),
                PatternPart::AnyRun => {
                    if at == 0 {
                        anchored_start = false;
                    }
                    if at + 1 == self.parts.len() {
                        anchored_end = false;
                    }
                }
            }
        }
        let mut rest = name;
        if anchored_start && anchored_end {
            // No run at either end: the pattern is one literal, or it is
            // literal-run-literal and both ends are pinned.
            if segments.len() == 1 {
                return rest == segments[0];
            }
        }
        if anchored_start {
            let Some(first) = segments.first() else {
                return rest.is_empty();
            };
            let Some(after) = rest.strip_prefix(*first) else {
                return false;
            };
            rest = after;
            segments.remove(0);
        }
        if anchored_end && let Some(last) = segments.pop() {
            let Some(before) = rest.strip_suffix(last) else {
                return false;
            };
            rest = before;
        }
        for segment in segments {
            let Some(at) = rest.find(segment) else {
                return false;
            };
            rest = &rest[at + segment.len()..];
        }
        true
    }
}

/// Picks the one asset a route should fetch.
///
/// ⛔ **Zero matches and two matches are both refusals.** Taking the first of
/// several would choose by the order the source happened to list them in, which
/// is a property of the source rather than of the target, and a route that
/// installed a different artifact next month would look identical in the record.
/// That is the same argument [`resolve`] makes for blocking on a candidate it
/// cannot order rather than skipping it.
///
/// # Errors
///
/// Returns [`AssetRefusal::NoMatch`] or [`AssetRefusal::Ambiguous`].
pub fn select_asset<'a>(
    assets: &'a [ReleaseAsset],
    pattern: &AssetPattern,
) -> Result<&'a ReleaseAsset, AssetRefusal> {
    let matched: Vec<&ReleaseAsset> = assets
        .iter()
        .filter(|asset| pattern.matches(asset.name.as_str()))
        .collect();
    match matched.as_slice() {
        [] => Err(AssetRefusal::NoMatch),
        [one] => Ok(one),
        many => Err(AssetRefusal::Ambiguous(
            many.iter().map(|asset| asset.name.clone()).collect(),
        )),
    }
}

/// Reading a source's own answer into candidates.
///
/// ⛔ One reader per source shape, and each one is the only place that knows
/// that shape. A resolver that accepted a pre-digested list would move the
/// parsing somewhere with no test and no record of what arrived.
pub mod sources {
    use super::{Candidate, ReleaseAsset};
    use crate::canonical::{Instant, Label, Sha256Digest, Slug, Url};

    /// Reads the GitHub releases list.
    ///
    /// # Errors
    ///
    /// Returns a message naming the entry that could not be read. A release
    /// with an unusable tag is refused rather than dropped: a candidate list
    /// quietly missing an entry is the silent skip this module exists to
    /// prevent, one layer earlier.
    pub fn github_releases(body: &[u8], source: &Slug) -> Result<Vec<Candidate>, String> {
        #[derive(serde::Deserialize)]
        struct Release {
            tag_name: String,
            #[serde(default)]
            prerelease: bool,
            #[serde(default)]
            draft: bool,
            published_at: Option<String>,
        }

        let releases: Vec<Release> =
            serde_json::from_slice(body).map_err(|error| format!("not a release list: {error}"))?;
        releases
            .into_iter()
            .map(|release| {
                let tag = Label::parse(&release.tag_name)
                    .map_err(|error| format!("tag {:?}: {error}", release.tag_name))?;
                let published_at = release
                    .published_at
                    .as_deref()
                    .map(Instant::parse)
                    .transpose()
                    .map_err(|error| format!("tag {tag}: published_at: {error}"))?;
                Ok(Candidate {
                    source: source.clone(),
                    tag,
                    prerelease: release.prerelease,
                    draft: release.draft,
                    published_at,
                })
            })
            .collect()
    }

    /// Reads `git ls-remote --tags --refs` output.
    ///
    /// ⭐ **This is the second SOURCE, which is what `E-ACQ-07` compares.** A
    /// release listing and a repository's refs are two indexes that can disagree
    /// about what the newest version is, and a route that resolves through each
    /// is two routes. Measured on 2026-09-09: `capture-client` resolved once and
    /// handed both lanes the answer, so its two lanes were one route and no
    /// record could be written from them.
    ///
    /// ⛔ **`prerelease` and `draft` are false because refs carry no such
    /// flags, and that is a real weakening rather than a default.** A releases
    /// listing told `ACQ-02`'s resolver that `release-5.3.0beta1` was a
    /// prerelease; refs say only the tag. What survives is the version text,
    /// which said so too in that case - so a prerelease this project can only
    /// recognise from a flag would be selected here. `ACQ-02` carries it.
    ///
    /// ⚠ **`published_at` is `None`, and refs have no date to offer.** A
    /// candidate whose tag no scheme can order therefore blocks the resolution
    /// rather than being released by the `predates_selection` signal, which is
    /// the resolver failing closed and is correct: nothing here can rule out
    /// that the unreadable tag is the newest.
    ///
    /// # Errors
    ///
    /// Returns a message naming the line that could not be read. ⛔ A peeled
    /// ref - the `^{}` entry `ls-remote` emits for an annotated tag without
    /// `--refs` - is refused rather than skipped, because accepting it would
    /// offer every annotated tag twice and dropping it would hide a caller that
    /// forgot the flag.
    pub fn git_refs(body: &[u8], source: &Slug) -> Result<Vec<Candidate>, String> {
        let text = core::str::from_utf8(body).map_err(|error| format!("not UTF-8: {error}"))?;
        let mut out = Vec::new();
        for line in text.lines() {
            if line.trim().is_empty() {
                continue;
            }
            let Some((object, reference)) = line.split_once('\t') else {
                return Err(format!("{line:?}: expected `<object>\\t<ref>`"));
            };
            if object.len() < 40 || !object.bytes().all(|b| b.is_ascii_hexdigit()) {
                return Err(format!("{object:?} is not a full object name"));
            }
            let Some(tag) = reference.strip_prefix("refs/tags/") else {
                return Err(format!("{reference:?} is not a tag ref"));
            };
            if tag.ends_with("^{}") {
                return Err(format!(
                    "{reference:?} is a peeled ref; pass --refs so an annotated tag is offered once"
                ));
            }
            out.push(Candidate {
                source: source.clone(),
                tag: Label::parse(tag).map_err(|error| format!("tag {tag:?}: {error}"))?,
                prerelease: false,
                draft: false,
                published_at: None,
            });
        }
        Ok(out)
    }

    /// Reads the assets of one release out of the same listing bytes.
    ///
    /// ⭐ **The same response, read a second time for a different question.**
    /// Nothing is fetched again: the version was decided from these bytes and
    /// the artifact is chosen from them too, so the digest the resolution
    /// records covers both decisions.
    ///
    /// `Ok(None)` means the listing carries no release under that tag, which is
    /// a different fact from a release with no assets and is kept apart from it.
    ///
    /// # Errors
    ///
    /// Returns a message naming the asset that could not be read. An asset with
    /// an unusable name or location is refused rather than dropped, for the
    /// reason [`github_releases`] gives about a quietly shorter list.
    pub fn github_release_assets(
        body: &[u8],
        tag: &Label,
    ) -> Result<Option<Vec<ReleaseAsset>>, String> {
        #[derive(serde::Deserialize)]
        struct Release {
            tag_name: String,
            #[serde(default)]
            assets: Vec<Asset>,
        }

        #[derive(serde::Deserialize)]
        struct Asset {
            name: String,
            browser_download_url: String,
            #[serde(default)]
            size: u64,
            // ⚠ ABSENT AND NULL ARE THE SAME ANSWER HERE, and both mean the
            // source states no digest. An older release predating this field
            // serves `null`; a projection that dropped it serves neither.
            #[serde(default)]
            digest: Option<String>,
        }

        let releases: Vec<Release> =
            serde_json::from_slice(body).map_err(|error| format!("not a release list: {error}"))?;
        let Some(release) = releases
            .into_iter()
            .find(|release| release.tag_name == tag.as_str())
        else {
            return Ok(None);
        };
        release
            .assets
            .into_iter()
            .map(|asset| {
                let name = Label::parse(&asset.name)
                    .map_err(|error| format!("asset {:?}: {error}", asset.name))?;
                let url = Url::parse(&asset.browser_download_url)
                    .map_err(|error| format!("asset {name}: url: {error}"))?;
                // ⛔ A DIGEST THIS PROJECT CANNOT READ IS A REFUSAL, NEVER AN
                // ABSENCE. The day a source publishes `sha512:` or a bare hex
                // run, treating it as "no digest" would silently downgrade a
                // verified route to an identified one with nothing saying so -
                // which is the failure this whole field exists to remove.
                // ⚠ `None` and a value are different answers; a value that
                // cannot be parsed is a third.
                let digest = asset
                    .digest
                    .as_deref()
                    .map(Sha256Digest::parse)
                    .transpose()
                    .map_err(|error| format!("asset {name}: digest: {error}"))?;
                Ok(ReleaseAsset {
                    name,
                    url,
                    size: asset.size,
                    digest,
                })
            })
            .collect::<Result<Vec<_>, String>>()
            .map(Some)
    }
}

#[cfg(test)]
mod tests {
    use super::{AssetPattern, VersionScheme, text_marks_prerelease};
    use crate::canonical::Label;

    fn pattern(text: &str, version: &str) -> AssetPattern {
        AssetPattern::parse(text, version).expect("pattern")
    }

    fn scheme(prefix: Option<&str>, min: u8, max: u8) -> VersionScheme {
        VersionScheme {
            tag_prefix: prefix.map(|p| Label::parse(p).expect("label")),
            min_components: min,
            max_components: max,
            floor_major: None,
        }
    }

    #[test]
    fn dotted_components_order_numerically_not_lexically() {
        let s = scheme(None, 2, 4);
        // ⛔ The headline defect: as text, "4.1.10" sorts before "4.1.9".
        assert!(s.components("4.1.10") > s.components("4.1.9"));
        assert!(s.components("5.2.10") > s.components("5.2.3"));
        assert!(s.components("5.2.3.10") > s.components("5.2.3.9"));
        assert!("4.1.10" < "4.1.9", "text order really is the wrong answer");
    }

    #[test]
    fn a_component_count_outside_the_scheme_is_not_orderable() {
        let s = scheme(None, 3, 3);
        assert!(s.components("4.1.3").is_some());
        assert!(s.components("4.1").is_none());
        assert!(s.components("4.1.3.1").is_none());
    }

    #[test]
    fn a_padded_component_is_refused_rather_than_parsed() {
        let s = scheme(None, 2, 4);
        assert!(s.components("1.01").is_none(), "1.01 and 1.1 compare equal");
        assert!(s.components("1.0").is_some());
    }

    #[test]
    fn the_floor_reads_a_plain_leading_integer_and_nothing_else() {
        let mut s = scheme(None, 3, 3);
        assert!(
            !s.below_floor("3.00"),
            "no floor declared, nothing is below it"
        );
        s.floor_major = Some(4);
        for text in ["3.00", "0.5", "2.20b1", "1.80b5", "3"] {
            assert!(s.below_floor(text), "{text} is below 4");
        }
        // ⛔ At the floor is not below it, and a leading component that is not
        // a plain integer is not read at all.
        for text in ["4.0", "4", "10.1", "03.1", "3x.1", "", ".3", "x"] {
            assert!(!s.below_floor(text), "{text} is not below 4");
        }
    }

    #[test]
    fn a_tag_prefix_is_stripped_and_a_foreign_tag_is_not_this_target() {
        let s = scheme(Some("release-"), 3, 3);
        assert_eq!(s.strip("release-5.2.3"), Some("5.2.3"));
        assert_eq!(s.strip("v5.2.3"), None);
        assert_eq!(scheme(None, 3, 3).strip("5.2.3"), Some("5.2.3"));
    }

    #[test]
    fn a_literal_pattern_matches_that_name_and_nothing_longer() {
        let p = pattern("aria2-{version}.tar.bz2", "1.37.0");
        assert!(p.matches("aria2-1.37.0.tar.bz2"));
        // ⛔ The anchored case a naive prefix/suffix matcher gets wrong: with no
        // run in the pattern, a longer name is not a match.
        assert!(!p.matches("aria2-1.37.0.tar.bz2.asc"));
        assert!(!p.matches("xaria2-1.37.0.tar.bz2"));
        assert!(!p.matches("aria2-1.37.1.tar.bz2"));
    }

    #[test]
    fn the_version_placeholder_is_what_makes_a_real_release_unambiguous() {
        // ⭐ qBittorrent 5.2.3's own asset list, measured 2026-09-09: two Linux
        // AppImages, distinguished only by an `_lt20` between the version and
        // the architecture.
        let wide = pattern("qbittorrent-*_x86_64.AppImage", "5.2.3");
        assert!(wide.matches("qbittorrent-5.2.3_x86_64.AppImage"));
        assert!(
            wide.matches("qbittorrent-5.2.3_lt20_x86_64.AppImage"),
            "a run is what makes this pattern match both, which is the defect"
        );
        let pinned = pattern("qbittorrent-{version}_x86_64.AppImage", "5.2.3");
        assert!(pinned.matches("qbittorrent-5.2.3_x86_64.AppImage"));
        assert!(!pinned.matches("qbittorrent-5.2.3_lt20_x86_64.AppImage"));
    }

    #[test]
    fn a_run_matches_any_span_including_none_and_orders_its_literals() {
        assert!(pattern("*", "1.0").matches("anything at all"));
        assert!(pattern("a*b", "1.0").matches("ab"), "a run may be empty");
        assert!(pattern("a*b", "1.0").matches("axxxb"));
        assert!(
            !pattern("a*b", "1.0").matches("a"),
            "the suffix is still due"
        );
        assert!(pattern("*mid*", "1.0").matches("xxmidxx"));
        assert!(!pattern("*mid*", "1.0").matches("xxdimxx"));
        // ⚠ Literals between two runs are found in order, not as a set.
        assert!(pattern("*a*b*", "1.0").matches("__a__b__"));
        assert!(!pattern("*a*b*", "1.0").matches("__b__a__"));
    }

    #[test]
    fn a_version_carrying_a_wildcard_is_matched_as_text() {
        // ⛔ `Version` accepts what a build printed. Splicing one into a glob
        // would let the value widen the pattern that was meant to pin it.
        let p = pattern("thing-{version}.tar", "1.*");
        assert!(p.matches("thing-1.*.tar"));
        assert!(!p.matches("thing-1.37.0.tar"));
    }

    #[test]
    fn an_unknown_placeholder_is_refused_rather_than_matched_as_text() {
        assert!(AssetPattern::parse("thing-{ver}.tar", "1.0").is_err());
        assert!(AssetPattern::parse("thing-{version.tar", "1.0").is_err());
        assert!(AssetPattern::parse("", "1.0").is_err());
        assert!(AssetPattern::parse("thing-{version}.tar", "1.0").is_ok());
    }

    #[test]
    fn adjacent_runs_collapse_and_a_case_difference_is_not_a_match() {
        assert!(pattern("a**b", "1.0").matches("axb"));
        // ⚠ An asset name is a file name, so case is part of it. Transmission's
        // release carries both `transmission-4.1.3.tar.xz` and
        // `Transmission-4.1.3.dmg`.
        assert!(
            !pattern("transmission-{version}.tar.xz", "4.1.3").matches("Transmission-4.1.3.tar.xz")
        );
    }

    #[test]
    fn prerelease_tokens_match_whole_tokens_and_their_numbered_forms() {
        for text in [
            "4.1.0-beta.5",
            "1.0.0-rc1",
            "2.0-ALPHA",
            "3.1.0.nightly",
            // ⛔ A marker after digits inside one token, as qBittorrent tags.
            "5.3.0beta1",
            "1.0.0rc10",
            "4.0.0dev",
            // ⛔ PEP 440's one-letter markers between digits, as Deluge and
            // Transmission's old betas tag. `CLIENT-07`.
            "2.0.0b1",
            "2.0b1",
            "2.20b1",
            "1.80b5",
            "1.0a2",
            "3.1c1",
        ] {
            assert!(text_marks_prerelease(text), "{text} is a prerelease");
        }
        // ⚠ A single letter with no digit on one side of it is no marker.
        for text in [
            "4.1.3",
            "5.2.3.10",
            "1.2.3-final",
            "2.0.0-march",
            "x86",
            "1.0b",
            "b1.0",
            "2.0.0-b",
            "1.0ab1",
        ] {
            assert!(
                !text_marks_prerelease(text),
                "{text} carries no prerelease token"
            );
        }
    }
}
