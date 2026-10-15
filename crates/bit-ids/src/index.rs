//! Deterministic indexes and latest views, derived from canonical records only.
//!
//! `CORPUS-03` owns this. A consumer wants to go from something it saw on the
//! wire to the build that produced it, and from a target to the newest measured
//! release, without opening every record in the store.
//!
//! ⛔ **These are derived files and never authoritative.** Every row names the
//! record it came from, so a reader who doubts a row can open the record and
//! decide from the measurement. An index that answered a question the records
//! could not is an index that invented one.
//!
//! ⛔ **Nothing here maps a peer-ID prefix to a client name from a table.** The
//! key of a peer-prefix row is the fixed span of a peer ID *this project
//! measured*, and the row resolves to the record that measured it.
//! `docs/architecture.md` section 5 forbids the decoder table, and this is its
//! opposite: the lookup exists because the measurement does.
//!
//! ⚠ **Only publishable records enter a view.** A provisional record belongs in
//! the store and its disagreement is the evidence, but a lookup that answered
//! with one would publish the claim the record model refused to publish.
//! [`build`] reports how many it left out, because a count with no denominator
//! is what makes an omission invisible.

use std::collections::{BTreeMap, BTreeSet};
use std::fmt::Write as _;

use crate::agreement::publishable_among;
use crate::canonical::{Instant, RelPath, Sha256Digest, Slug, Version};
use crate::corpus::Corpus;
use crate::corpus::others_of;
use crate::identity::RecordId;
use crate::observation::{FieldPath, FieldState, PatternRun};
use crate::record::Profile;
use crate::resolution::VersionScheme;
use crate::validate::{SchemaError, Violations};

/// Identifier carried by every first-generation index document.
pub const INDEX_SCHEMA: &str = "bit-ids/index/1";

/// The field a peer-prefix row is keyed on.
pub const PEER_ID_FIELD: &str = "peer_wire/peer_id";

/// The field a client-string row is keyed on.
pub const BEP10_CLIENT_FIELD: &str = "peer_wire/bep10.client";

/// What one index is a lookup by.
///
/// ⚠ The names are the published spellings and are pinned to their literals by
/// `index_kinds_have_one_spelling`. They appear in a document a consumer parses,
/// so renaming one is a schema change rather than a rename.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Ord, PartialOrd)]
pub enum IndexKind {
    /// By catalogue target identifier.
    Target,
    /// By the fixed leading span of a measured peer ID.
    PeerPrefix,
    /// By the measured BEP 10 client string.
    Bep10Client,
    /// By host family.
    Platform,
    /// By the version the installed build reported.
    Version,
    /// By capture instant.
    CapturedAt,
}

impl IndexKind {
    /// Every kind, in the order a document lists them.
    pub const ALL: &'static [Self] = &[
        Self::Target,
        Self::PeerPrefix,
        Self::Bep10Client,
        Self::Platform,
        Self::Version,
        Self::CapturedAt,
    ];

    /// The published spelling.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Target => "target",
            Self::PeerPrefix => "peer_prefix",
            Self::Bep10Client => "bep10_client",
            Self::Platform => "platform",
            Self::Version => "version",
            Self::CapturedAt => "captured_at",
        }
    }
}

/// One lookup row: a key, and the record it resolves to.
#[derive(Clone, Debug, Eq, PartialEq, Ord, PartialOrd)]
pub struct IndexRow {
    /// What a consumer looked up.
    pub key: String,
    /// The record that answers it.
    pub record: RecordId,
    /// Where that record is filed.
    pub path: RelPath,
}

/// One index.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Index {
    /// What it is a lookup by.
    pub kind: IndexKind,
    /// The rows, ascending by key then by record.
    pub rows: Vec<IndexRow>,
}

/// The newest publishable record for one build line.
#[derive(Clone, Debug, Eq, PartialEq, Ord, PartialOrd)]
pub struct LatestRow {
    /// The catalogue target.
    pub target: Slug,
    /// Host family.
    pub platform: Slug,
    /// Machine architecture.
    pub arch: Slug,
    /// Package format.
    pub package: Slug,
    /// The version it selects.
    pub version: Version,
    /// The record that answers.
    pub record: RecordId,
    /// Where that record is filed.
    pub path: RelPath,
}

/// The build line a latest row is the newest of: target, platform, arch and
/// package. It is the identity tuple minus the two things that vary between
/// records of one line, which are the version and the capture.
type BuildLine = (Slug, Slug, Slug, Slug);

/// What decides which record of a build line is the latest: the scheme's
/// numeric ordering first, then the capture instant, then the record
/// identifier. Total by construction, so the answer does not depend on the
/// order the store was read in.
type Ranking = (Vec<u64>, Instant, RecordId);

/// One step of the correction chain, and where it ends.
///
/// ⛔ **A superseded record is dropped from the views and never from the store.**
/// The append-only rule keeps its bytes and its evidence; what a correction
/// changes is which record answers a question asked now. A consumer holding an
/// identifier from last month has no other way to discover that, so the chain is
/// published rather than left to be inferred from `supersedes` fields the
/// consumer would have to fetch one at a time.
///
/// ⚠ `by` is the immediate successor and `current` is the end of the walk. They
/// differ exactly when a correction was itself corrected, which is the case a
/// single-step row would answer wrongly while looking right.
#[derive(Clone, Debug, Eq, PartialEq, Ord, PartialOrd)]
pub struct CorrectionRow {
    /// The record that no longer answers.
    pub superseded: RecordId,
    /// The record that supersedes it directly.
    pub by: RecordId,
    /// The record at the end of the chain, which is what answers now.
    pub current: RecordId,
    /// Where `current` is filed.
    pub path: RelPath,
}

/// Everything a build produces, and what it left out.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Indexes {
    /// One per [`IndexKind`], in `IndexKind::ALL` order.
    pub indexes: Vec<Index>,
    /// The latest view, ascending by build line.
    pub latest: Vec<LatestRow>,
    /// Every superseded record, and what answers in its place.
    pub corrections: Vec<CorrectionRow>,
    /// How many records were in the store and not in any view.
    ///
    /// ⚠ Reported rather than inferred. A consumer comparing a row count with a
    /// store's record count would otherwise read an exclusion as a defect.
    pub excluded: usize,
    /// How many records were left out because something corrects them.
    ///
    /// ⚠ Counted apart from `excluded` because the two mean opposite things
    /// about a measurement. An excluded record was never publishable; a
    /// superseded one was, and a later run says it is no longer current.
    pub superseded: usize,
}

/// The leading fixed span of a measured peer ID, as a lookup key.
///
/// ⚠ A patterned value whose first run varies has no usable prefix and produces
/// no row, rather than a row keyed on nothing. `unknown`, `not_observed`,
/// `not_supported` and `variable` produce none either: a lookup key has to be
/// something a build was measured emitting.
fn peer_prefix(state: &FieldState) -> Option<String> {
    match state {
        FieldState::Constant(value) => Some(value.value.to_hex()),
        FieldState::Patterned(value) => match value.pattern.runs.first()? {
            PatternRun::Fixed { bytes } => Some(bytes.to_hex()),
            PatternRun::Varying { .. } => None,
        },
        FieldState::Unknown
        | FieldState::NotObserved
        | FieldState::NotSupported
        | FieldState::Variable(_) => None,
    }
}

/// The measured client string, as a lookup key.
///
/// ⚠ Constant only. A BEP 10 `v` string that varies between samples carries a
/// build number or a session token, and a prefix of one is a guess about which
/// half is which. `OBS-07`'s stock-client controls are what would settle that.
fn client_string(state: &FieldState) -> Option<String> {
    match state {
        FieldState::Constant(value) => Some(value.value.to_hex()),
        FieldState::Unknown
        | FieldState::NotObserved
        | FieldState::NotSupported
        | FieldState::Patterned(_)
        | FieldState::Variable(_) => None,
    }
}

fn field<'a>(profile: &'a Profile, path: &str) -> Option<&'a FieldState> {
    let path = FieldPath::parse(path).ok()?;
    profile.field(&path).map(|observed| &observed.state)
}

/// The correction graph over a store.
struct Chains {
    /// Each superseded record, and the record that directly corrects it.
    succeeds: BTreeMap<RecordId, RecordId>,
    /// Each superseded record, and the record at the end of its chain.
    current_of: BTreeMap<RecordId, RecordId>,
    /// Every record the store carries, and where it is filed.
    filed: BTreeMap<RecordId, RelPath>,
}

/// Reads the correction graph, refusing a fork and a cycle.
///
/// ⛔ **A PASS OF ITS OWN, BECAUSE A RECORD CANNOT KNOW IT HAS BEEN CORRECTED.**
/// The `supersedes` field points backwards, so the successor of a record is only
/// discoverable by reading every other record first. Deciding whether to index a
/// record in the same pass that discovers the correction would get it wrong in
/// whichever direction the store happened to be read in.
fn chains(corpus: &Corpus, errors: &mut Vec<SchemaError>) -> Chains {
    // ⚠ ONLY A PUBLISHABLE CORRECTION RETRACTS ANYTHING. A correction carrying
    // its own disagreement is provisional, and letting it drop the record it
    // corrects would leave the build line answering nothing at all: the views
    // would lose a measurement to a record that is not fit to replace it.
    let mut successors: BTreeMap<RecordId, Vec<(RecordId, RelPath)>> = BTreeMap::new();
    let mut filed: BTreeMap<RecordId, RelPath> = BTreeMap::new();
    for stored in corpus.profiles() {
        let profile = &stored.profile;
        filed.insert(profile.id, stored.path.clone());
        if publishable_among(profile, &others_of(corpus, profile)).is_err() {
            continue;
        }
        if let Some(prior) = profile.supersedes {
            successors
                .entry(prior)
                .or_default()
                .push((profile.id, stored.path.clone()));
        }
    }

    // ⛔ A FORK HAS NO ANSWER AND MUST NOT GET ONE BY ACCIDENT. Two records
    // correcting the same measurement is a real thing to discover and not a
    // thing to resolve here: picking the lower identifier would publish one
    // adjudication and silently discard the other.
    let mut succeeds: BTreeMap<RecordId, RecordId> = BTreeMap::new();
    for (prior, mut found) in successors {
        found.sort();
        if let [(first, _), rest @ ..] = found.as_slice()
            && !rest.is_empty()
        {
            let others = rest
                .iter()
                .map(|(id, _)| id.to_string())
                .collect::<Vec<_>>()
                .join(", ");
            errors.push(SchemaError::new(
                "E-VIW-03",
                filed.get(&prior).map_or("<absent>", RelPath::as_str),
                format!("{prior} is corrected by {first} and also by {others}"),
            ));
            continue;
        }
        if let Some((only, _)) = found.first() {
            succeeds.insert(prior, *only);
        }
    }

    // ⛔ BOUNDED, BECAUSE A CYCLE IS CONSTRUCTIBLE. A record identifier digests
    // the identity tuple and not `supersedes`, so two records can each name the
    // other and neither is self-superseding, which is the only shape the record
    // validator refuses. An unbounded walk here would hang rather than report.
    let mut current_of: BTreeMap<RecordId, RecordId> = BTreeMap::new();
    for start in succeeds.keys().copied() {
        let mut at = start;
        let mut steps = 0_usize;
        let end = loop {
            match succeeds.get(&at) {
                None => break Some(at),
                Some(next) => {
                    at = *next;
                    steps += 1;
                    if steps > succeeds.len() {
                        break None;
                    }
                }
            }
        };
        match end {
            Some(end) => {
                current_of.insert(start, end);
            }
            None => errors.push(SchemaError::new(
                "E-VIW-04",
                filed.get(&start).map_or("<absent>", RelPath::as_str),
                format!("the correction chain from {start} returns to where it started"),
            )),
        }
    }

    Chains {
        succeeds,
        current_of,
        filed,
    }
}

impl Chains {
    /// The published correction rows, ascending.
    ///
    /// ⚠ Only a chain whose ends the store actually carries becomes a row. A
    /// `supersedes` naming a record nobody has is `E-CRP-07`'s refusal at corpus
    /// level, and emitting a row for it here would put an identifier in a
    /// derived file that `rows_resolve` then could not resolve, which is the
    /// defect that check exists to catch rather than one to feed it.
    fn rows(&self) -> Vec<CorrectionRow> {
        let mut rows: Vec<CorrectionRow> = self
            .succeeds
            .iter()
            .filter_map(|(superseded, by)| {
                let current = *self.current_of.get(superseded)?;
                Some(CorrectionRow {
                    superseded: *superseded,
                    by: *by,
                    current,
                    path: self.filed.get(&current)?.clone(),
                })
            })
            .collect();
        rows.sort();
        rows
    }
}

/// The lookup rows one record contributes.
///
/// ⚠ Two of the six are conditional and the other four are not. A record always
/// has a target, a platform, a version and a capture instant; a measured peer
/// prefix and a measured client string are values a build may not have produced,
/// and a row keyed on their absence would answer every lookup.
fn push_lookups(profile: &Profile, path: &RelPath, rows: &mut BTreeMap<IndexKind, Vec<IndexRow>>) {
    let mut push = |kind: IndexKind, key: String| {
        rows.entry(kind).or_default().push(IndexRow {
            key,
            record: profile.id,
            path: path.clone(),
        });
    };

    push(IndexKind::Target, profile.target.id.to_string());
    push(IndexKind::Platform, profile.build.platform.to_string());
    push(IndexKind::Version, profile.build.version.to_string());
    push(
        IndexKind::CapturedAt,
        profile.capture.captured_at.to_string(),
    );
    if let Some(key) = field(profile, PEER_ID_FIELD).and_then(peer_prefix) {
        push(IndexKind::PeerPrefix, key);
    }
    if let Some(key) = field(profile, BEP10_CLIENT_FIELD).and_then(client_string) {
        push(IndexKind::Bep10Client, key);
    }
}

/// Builds every index and the latest view over a store.
///
/// `schemes` says how each target spells its versions, which is what makes a
/// latest view orderable at all. A target with no scheme, or whose selected
/// records carry a version the scheme cannot order, is **left out of the latest
/// view with a refusal** rather than ordered by a guess.
///
/// # Errors
///
/// | code | refused |
/// | --- | --- |
/// | `E-VIW-01` | a publishable record whose target declares no version scheme |
/// | `E-VIW-02` | a publishable record whose version its own scheme cannot order |
/// | `E-VIW-03` | two publishable records correcting the same record |
/// | `E-VIW-04` | a correction chain that returns to where it started |
///
/// ⚠ The first two are refusals about the **latest view only**. The lookup
/// indexes carry the record regardless, because finding a measurement does not
/// require ordering it. The last two are about the whole document: a fork leaves
/// no answer to "what replaces this", and a cycle has no end to walk to.
pub fn build(
    corpus: &Corpus,
    schemes: &BTreeMap<Slug, VersionScheme>,
) -> Result<Indexes, Violations> {
    let mut errors = Vec::new();
    let mut rows: BTreeMap<IndexKind, Vec<IndexRow>> = BTreeMap::new();
    let mut best: BTreeMap<BuildLine, (Ranking, LatestRow)> = BTreeMap::new();
    let mut excluded = 0_usize;

    let graph = chains(corpus, &mut errors);
    let mut superseded = 0_usize;

    for stored in corpus.profiles() {
        let profile = &stored.profile;
        if publishable_among(profile, &others_of(corpus, profile)).is_err() {
            excluded += 1;
            continue;
        }
        // ⛔ Out of every view, and still in the store. A consumer asking a
        // question now is answered by the record that corrects this one; the
        // corrections list below is how they find that from an old identifier.
        if graph.succeeds.contains_key(&profile.id) {
            superseded += 1;
            continue;
        }
        push_lookups(profile, &stored.path, &mut rows);

        // ⛔ The latest view fails closed. An unorderable version blocks the
        // build line rather than being skipped, because a skip yields an older
        // record selected confidently, which is `ACQ-02`'s rule about
        // resolution applied to publication.
        let Some(scheme) = schemes.get(&profile.target.id) else {
            errors.push(SchemaError::new(
                "E-VIW-01",
                stored.path.as_str(),
                format!(
                    "target {} declares no version scheme, so no latest view can order it",
                    profile.target.id
                ),
            ));
            continue;
        };
        let Some(order) = scheme.components(profile.build.version.as_str()) else {
            errors.push(SchemaError::new(
                "E-VIW-02",
                stored.path.as_str(),
                format!(
                    "version {} is not orderable under target {}'s scheme",
                    profile.build.version, profile.target.id
                ),
            ));
            continue;
        };

        let line: BuildLine = (
            profile.target.id.clone(),
            profile.build.platform.clone(),
            profile.build.arch.clone(),
            profile.build.package.clone(),
        );
        let row = LatestRow {
            target: profile.target.id.clone(),
            platform: profile.build.platform.clone(),
            arch: profile.build.arch.clone(),
            package: profile.build.package.clone(),
            version: profile.build.version.clone(),
            record: profile.id,
            path: stored.path.clone(),
        };
        // ⛔ ONE TOTAL KEY, and the version is not in it. `Version` is `Ord` as
        // text, so `1.2.9` compares greater than `1.2.10` and a comparison that
        // mentions it at all silently loses the numeric ordering the scheme
        // computed. That is not hypothetical: it was written that way here and
        // `the_latest_view_names_the_newest_record_per_build_line` caught the
        // view answering `1.2.9`.
        //
        // ⚠ Two captures of one build tie on the ordering, so the instant and
        // then the record identifier break it. Neither is a preference: the rule
        // has to be total, or two runs of this over one store produce two
        // different documents.
        let ranking = (order, profile.capture.captured_at.clone(), profile.id);
        match best.get(&line) {
            Some((have, _)) if *have >= ranking => {}
            _ => {
                best.insert(line, (ranking, row));
            }
        }
    }

    if !errors.is_empty() {
        return Violations::from_errors(errors).map(|()| unreachable!());
    }

    let mut indexes = Vec::with_capacity(IndexKind::ALL.len());
    for kind in IndexKind::ALL {
        let mut sorted = rows.remove(kind).unwrap_or_default();
        // ⛔ Sorted here and never relied on from the store's iteration order.
        // Two clean builds have to produce identical bytes, and a map's order is
        // not a contract a consumer can check.
        sorted.sort();
        sorted.dedup();
        indexes.push(Index {
            kind: *kind,
            rows: sorted,
        });
    }
    let mut latest: Vec<LatestRow> = best.into_values().map(|(_, row)| row).collect();
    // ⚠ NOT REFUTED, AND KEPT. `best` is a `BTreeMap` keyed by the build line,
    // and `LatestRow` orders on those same four fields first, so removing this
    // changes no output that can be produced today. It would matter the moment
    // `best` stopped being an ordered map or the row's ordering stopped
    // agreeing with the key's, and a reader should not take an unreached guard
    // for a proven one.
    latest.sort();

    let corrections = graph.rows();

    Ok(Indexes {
        indexes,
        latest,
        corrections,
        excluded,
        superseded,
    })
}

impl Indexes {
    /// The published document.
    ///
    /// ⛔ **Written by hand, and no clock is read.** The field order, the
    /// spacing and the escaping are the bytes a digest names, so a derive that
    /// changed any of them between versions would move every digest a consumer
    /// had recorded. `two_builds_are_byte_identical` is the check.
    #[must_use]
    pub fn to_json(&self) -> String {
        let mut out = String::new();
        out.push_str("{\n  \"schema\": \"");
        out.push_str(INDEX_SCHEMA);
        out.push_str("\",\n  \"excluded\": ");
        let _ = write!(out, "{}", self.excluded);
        out.push_str(",\n  \"superseded\": ");
        let _ = write!(out, "{}", self.superseded);
        out.push_str(",\n  \"indexes\": [\n");
        for (index, entry) in self.indexes.iter().enumerate() {
            out.push_str("    {\n      \"kind\": \"");
            out.push_str(entry.kind.as_str());
            out.push_str("\",\n      \"rows\": [\n");
            for (row_index, row) in entry.rows.iter().enumerate() {
                out.push_str("        {\"key\": \"");
                out.push_str(&row.key);
                out.push_str("\", \"record\": \"");
                let _ = write!(out, "{}", row.record);
                out.push_str("\", \"path\": \"");
                out.push_str(row.path.as_str());
                out.push_str("\"}");
                if row_index + 1 < entry.rows.len() {
                    out.push(',');
                }
                out.push('\n');
            }
            out.push_str("      ]\n    }");
            if index + 1 < self.indexes.len() {
                out.push(',');
            }
            out.push('\n');
        }
        out.push_str("  ],\n  \"latest\": [\n");
        for (index, row) in self.latest.iter().enumerate() {
            out.push_str("    {\"target\": \"");
            out.push_str(row.target.as_str());
            out.push_str("\", \"platform\": \"");
            out.push_str(row.platform.as_str());
            out.push_str("\", \"arch\": \"");
            out.push_str(row.arch.as_str());
            out.push_str("\", \"package\": \"");
            out.push_str(row.package.as_str());
            out.push_str("\", \"version\": \"");
            out.push_str(row.version.as_str());
            out.push_str("\", \"record\": \"");
            let _ = write!(out, "{}", row.record);
            out.push_str("\", \"path\": \"");
            out.push_str(row.path.as_str());
            out.push_str("\"}");
            if index + 1 < self.latest.len() {
                out.push(',');
            }
            out.push('\n');
        }
        out.push_str("  ],\n  \"corrections\": [\n");
        for (index, row) in self.corrections.iter().enumerate() {
            out.push_str("    {\"superseded\": \"");
            let _ = write!(out, "{}", row.superseded);
            out.push_str("\", \"by\": \"");
            let _ = write!(out, "{}", row.by);
            out.push_str("\", \"current\": \"");
            let _ = write!(out, "{}", row.current);
            out.push_str("\", \"path\": \"");
            out.push_str(row.path.as_str());
            out.push_str("\"}");
            if index + 1 < self.corrections.len() {
                out.push(',');
            }
            out.push('\n');
        }
        out.push_str("  ]\n}\n");
        out
    }

    /// The digest of the published document.
    #[must_use]
    pub fn digest(&self) -> Sha256Digest {
        Sha256Digest::of(self.to_json().as_bytes())
    }

    /// Reads a published index document back.
    ///
    /// ⛔ **`to_json` had no reader at all until `LIB-01` needed one**, and a
    /// published document with a writer and nothing that parses it is a format
    /// nobody has round-tripped. A consumer reads this file *instead of* the
    /// records, so the shape it is written in has to be one this project can
    /// also read; otherwise the only proof it is well formed is that the writer
    /// produced it.
    ///
    /// ⚠ **It parses the document rather than the writer's spacing.** The bytes
    /// are hand-written and their exact form is what a digest names, so a reader
    /// that depended on the layout would refuse a document a later writer
    /// reformatted while leaving the meaning alone.
    /// `an_index_document_round_trips_through_its_own_reader` is what holds the
    /// two together, in the direction that matters: bytes in, bytes out.
    ///
    /// # Errors
    ///
    /// Returns [`crate::DocumentError::UnsupportedSchema`] for another
    /// generation and [`crate::DocumentError::Malformed`] when the document is
    /// not this shape.
    pub fn from_json(document: &str) -> Result<Self, crate::DocumentError> {
        use crate::DocumentError;
        use serde_json::Value;

        let root: Value = serde_json::from_str(document).map_err(DocumentError::Malformed)?;
        let schema = reading::text(&root, "schema")?;
        if schema != INDEX_SCHEMA {
            return Err(DocumentError::UnsupportedSchema {
                found: schema.to_owned(),
                expected: INDEX_SCHEMA,
            });
        }

        // ⛔ The kinds are read positionally against `IndexKind::ALL` and the
        // spelling is checked, so a document listing them in another order, or
        // carrying one this build does not know, is refused rather than silently
        // mapped onto the wrong lookup.
        let listed = reading::rows(&root, "indexes")?;
        if listed.len() != IndexKind::ALL.len() {
            return Err(reading::malformed(format!(
                "expected {} indexes, found {}",
                IndexKind::ALL.len(),
                listed.len()
            )));
        }
        let mut indexes = Vec::with_capacity(listed.len());
        for (entry, kind) in listed.iter().zip(IndexKind::ALL) {
            let declared = reading::text(entry, "kind")?;
            if declared != kind.as_str() {
                return Err(reading::malformed(format!(
                    "index {} declares kind {declared:?}",
                    kind.as_str()
                )));
            }
            let mut parsed = Vec::new();
            for row in reading::rows(entry, "rows")? {
                parsed.push(IndexRow {
                    key: reading::text(row, "key")?.to_owned(),
                    record: reading::named(row, "record")?,
                    path: reading::at(row)?,
                });
            }
            indexes.push(Index {
                kind: *kind,
                rows: parsed,
            });
        }

        let mut latest = Vec::new();
        for row in reading::rows(&root, "latest")? {
            latest.push(LatestRow {
                target: reading::slug(row, "target")?,
                platform: reading::slug(row, "platform")?,
                arch: reading::slug(row, "arch")?,
                package: reading::slug(row, "package")?,
                version: Version::parse(reading::text(row, "version")?)
                    .map_err(reading::malformed)?,
                record: reading::named(row, "record")?,
                path: reading::at(row)?,
            });
        }

        let mut corrections = Vec::new();
        for row in reading::rows(&root, "corrections")? {
            corrections.push(CorrectionRow {
                superseded: reading::named(row, "superseded")?,
                by: reading::named(row, "by")?,
                current: reading::named(row, "current")?,
                path: reading::at(row)?,
            });
        }

        Ok(Self {
            indexes,
            latest,
            corrections,
            excluded: reading::count(&root, "excluded")?,
            superseded: reading::count(&root, "superseded")?,
        })
    }

    /// Every record these views are over, ascending.
    ///
    /// ⛔ **THE ONE ANSWER TO "WHICH RECORDS ARE PUBLISHED".** Only publishable
    /// records enter a view and never one something corrects, and that rule is
    /// applied in `build`. A caller that re-derived it would hold a second
    /// spelling of it, and the two would drift in the direction that publishes a
    /// retracted measurement in one rendering while the lookups had stopped
    /// naming it. `PUB-03` asks this rather than filtering a store again.
    ///
    /// ⚠ Read off the target index, which carries exactly one row per included
    /// record because every record has a target and nothing else keys on it.
    /// `the_record_set_is_what_the_views_include` is what pins that.
    #[must_use]
    pub fn records(&self) -> BTreeSet<RecordId> {
        self.indexes
            .iter()
            .filter(|index| index.kind == IndexKind::Target)
            .flat_map(|index| index.rows.iter().map(|row| row.record))
            .collect()
    }

    /// How many rows the whole set carries, latest included.
    #[must_use]
    pub fn rows(&self) -> usize {
        self.indexes
            .iter()
            .map(|index| index.rows.len())
            .sum::<usize>()
            + self.latest.len()
            + self.corrections.len()
    }
}

/// The reading helpers [`Indexes::from_json`] is built from.
///
/// ⚠ At module scope rather than nested inside it, because nine small readers
/// inside one function is a function no reader can hold, and clippy's line
/// ceiling is the check that said so.
mod reading {
    use serde_json::Value;

    use crate::DocumentError;
    use crate::canonical::{RelPath, Slug};
    use crate::identity::RecordId;

    pub(super) fn malformed(message: impl core::fmt::Display) -> DocumentError {
        DocumentError::Malformed(serde_json::Error::io(std::io::Error::other(
            message.to_string(),
        )))
    }

    pub(super) fn text<'a>(value: &'a Value, key: &str) -> Result<&'a str, DocumentError> {
        value
            .get(key)
            .and_then(Value::as_str)
            .ok_or_else(|| malformed(format!("{key} is missing or not a string")))
    }

    pub(super) fn count(value: &Value, key: &str) -> Result<usize, DocumentError> {
        value
            .get(key)
            .and_then(Value::as_u64)
            .and_then(|n| usize::try_from(n).ok())
            .ok_or_else(|| malformed(format!("{key} is missing or not a count")))
    }

    pub(super) fn rows<'a>(value: &'a Value, key: &str) -> Result<&'a Vec<Value>, DocumentError> {
        value
            .get(key)
            .and_then(Value::as_array)
            .ok_or_else(|| malformed(format!("{key} is missing or not a list")))
    }

    pub(super) fn named(value: &Value, key: &str) -> Result<RecordId, DocumentError> {
        RecordId::parse(text(value, key)?).map_err(malformed)
    }

    pub(super) fn at(value: &Value) -> Result<RelPath, DocumentError> {
        RelPath::parse(text(value, "path")?).map_err(malformed)
    }

    pub(super) fn slug(value: &Value, key: &str) -> Result<Slug, DocumentError> {
        Slug::parse(text(value, key)?).map_err(malformed)
    }
}

/// Whether every row resolves to exactly one record the store carries.
///
/// ⛔ **This is the half a derived file cannot be trusted about.** An index is
/// built from records and is then read *instead of* them, so a row naming a
/// record nobody can open is an answer with nothing behind it. It is checked
/// against the corpus rather than against the builder that produced it, because
/// a builder checked against itself agrees with itself.
///
/// # Errors
///
/// | code | refused |
/// | --- | --- |
/// | `E-VIW-10` | a row naming a record the store does not carry |
/// | `E-VIW-11` | a row whose path is not where that record is filed |
pub fn rows_resolve(indexes: &Indexes, corpus: &Corpus) -> Result<(), Violations> {
    let mut known: BTreeMap<RecordId, &RelPath> = BTreeMap::new();
    for stored in corpus.profiles() {
        known.insert(stored.profile.id, &stored.path);
    }

    let mut errors = Vec::new();
    let mut check = |at: String, record: RecordId, path: &RelPath| match known.get(&record) {
        None => errors.push(SchemaError::new(
            "E-VIW-10",
            at,
            format!("names {record}, which this store does not carry"),
        )),
        Some(filed) if *filed != path => errors.push(SchemaError::new(
            "E-VIW-11",
            at,
            format!("says {path}, the store files that record at {filed}"),
        )),
        Some(_) => {}
    };

    for index in &indexes.indexes {
        for row in &index.rows {
            check(
                format!("{}[{}]", index.kind.as_str(), row.key),
                row.record,
                &row.path,
            );
        }
    }
    for row in &indexes.latest {
        check(
            format!("latest[{} {} {}]", row.target, row.platform, row.package),
            row.record,
            &row.path,
        );
    }
    // ⛔ ALL THREE IDENTIFIERS, NOT ONLY THE ONE CARRYING A PATH. A correction
    // row is the one place a derived file names a record it is *not* pointing a
    // reader at, and a superseded identifier that resolves to nothing would make
    // the chain unwalkable in exactly the case it exists for. `current` is
    // checked against its path; the other two only have to be records the store
    // carries.
    for row in &indexes.corrections {
        check(
            format!("corrections[{}]", row.superseded),
            row.current,
            &row.path,
        );
    }

    // ⛔ ALL THREE IDENTIFIERS, NOT ONLY THE ONE CARRYING A PATH. A correction
    // row is the one place a derived file names records it is *not* pointing a
    // reader at, and a superseded identifier resolving to nothing would make the
    // chain unwalkable in exactly the case it exists for. `current` is checked
    // against its path above; these two only have to be records the store has.
    for row in &indexes.corrections {
        for (label, id) in [("superseded", row.superseded), ("by", row.by)] {
            if !known.contains_key(&id) {
                errors.push(SchemaError::new(
                    "E-VIW-10",
                    format!("corrections[{}].{label}", row.superseded),
                    format!("names {id}, which this store does not carry"),
                ));
            }
        }
    }

    Violations::from_errors(errors)
}

#[cfg(test)]
pub(crate) mod tests {
    use super::{BEP10_CLIENT_FIELD, INDEX_SCHEMA, IndexKind, PEER_ID_FIELD, build, rows_resolve};
    use crate::canonical::{RelPath, Sha256Digest, Slug, Version};
    use crate::corpus::Corpus;
    use crate::identity::{RecordId, RecordKey};
    use crate::resolution::VersionScheme;
    use crate::store::{Entry, ObjectRef, StoreKey, StoreTree};
    use crate::{Profile, RunManifest};
    use std::collections::BTreeMap;

    const PROFILE: &str = include_str!("../tests/fixtures/valid-profile.json");
    const MANIFEST: &str = include_str!("../tests/fixtures/valid-manifest.json");
    const CORRECTION: &str = include_str!("../tests/fixtures/valid-correction.json");

    fn profile() -> Profile {
        Profile::from_json(PROFILE).expect("the fixture record validates")
    }

    pub(crate) fn slug(text: &str) -> Slug {
        Slug::parse(text).expect("a canonical slug")
    }

    /// Three dotted numbers, which is how most of the catalogue spells a
    /// version.
    pub(crate) fn schemes() -> BTreeMap<Slug, VersionScheme> {
        BTreeMap::from([(
            slug("fixture-client"),
            VersionScheme {
                tag_prefix: None,
                min_components: 3,
                max_components: 3,
                floor_major: None,
            },
        )])
    }

    /// The fixture record, re-versioned and re-identified.
    ///
    /// ⛔ **The fixture's own version is `0.0.0-fixture`, which no numeric
    /// scheme can order**, so it proves the fail-closed path and cannot prove
    /// the ordering one. A record at a real version is built here instead, with
    /// every route's reported version moved with it, because `E-ACQ-04` refuses
    /// a route that installed a version the record does not declare, and the
    /// identifier re-derived, because it digests the version.
    pub(crate) fn record_at(version_text: &str) -> Profile {
        let mut record = profile();
        let version = Version::parse(version_text).expect("a reported version");
        record.build.version = version.clone();
        for route in &mut record.acquisition {
            route.installed_version = version.clone();
            route.resolved_version = version.clone();
        }
        record.capture.id = Slug::parse(&format!("cap-{}", version_text.replace('.', "-")))
            .expect("a canonical slug");
        record.id = RecordId::derive(&RecordKey {
            schema: &record.schema,
            target: &record.target.id,
            version: &record.build.version,
            platform: &record.build.platform,
            arch: &record.build.arch,
            package: &record.build.package,
            capture: &record.capture.id,
        });
        crate::validate::validate(&record).expect("a re-versioned record still validates");
        record
    }

    /// A store carrying one record per listed version, on one build line.
    fn corpus_at(versions: &[&str]) -> Corpus {
        let mut tree = StoreTree::new();
        let mut records = Vec::new();
        for text in versions {
            let record = record_at(text);
            let path = StoreKey::of_profile(&record)
                .profile_path()
                .expect("a publishable path");
            tree.insert(
                path.clone(),
                Entry::Object(ObjectRef {
                    bytes: PROFILE.len() as u64,
                    sha256: Sha256Digest::of(PROFILE.as_bytes()),
                }),
            );
            records.push((path, record));
        }
        let mut corpus = Corpus::new(tree);
        for (path, record) in records {
            corpus.insert_profile(path, record);
        }
        corpus
    }

    fn corpus() -> Corpus {
        corpus_at(&["1.2.3"])
    }

    /// A store carrying exactly these records, each at its derived path.
    pub(crate) fn corpus_of(records: Vec<Profile>) -> Corpus {
        let mut tree = StoreTree::new();
        let mut placed = Vec::new();
        for record in records {
            let path = StoreKey::of_profile(&record)
                .profile_path()
                .expect("a publishable path");
            tree.insert(
                path.clone(),
                Entry::Object(ObjectRef {
                    bytes: PROFILE.len() as u64,
                    sha256: Sha256Digest::of(PROFILE.as_bytes()),
                }),
            );
            placed.push((path, record));
        }
        let mut corpus = Corpus::new(tree);
        for (path, record) in placed {
            corpus.insert_profile(path, record);
        }
        corpus
    }

    /// A record on the fixture's build line that corrects `prior`.
    ///
    /// ⚠ The capture is a parameter rather than derived from the version,
    /// because a correction of one build is a second run of it: same version,
    /// different capture, and therefore a different identifier. Passing the
    /// original's own capture reproduces the original's identifier exactly,
    /// which is how the cycle case below is built at all.
    pub(crate) fn correction_at(version_text: &str, capture: &str, prior: RecordId) -> Profile {
        let mut record = record_at(version_text);
        record.capture.id = slug(capture);
        record.id = RecordId::derive(&RecordKey {
            schema: &record.schema,
            target: &record.target.id,
            version: &record.build.version,
            platform: &record.build.platform,
            arch: &record.build.arch,
            package: &record.build.package,
            capture: &record.capture.id,
        });
        record.supersedes = Some(prior);
        // The fixture correction's adjudication cites `ev-connector-report` and
        // `ev-packet-capture`, which the base record also carries, so `E-ADJ-04`
        // is satisfied without inventing evidence.
        record.adjudication = Profile::from_json(CORRECTION)
            .expect("the fixture correction validates")
            .adjudication;
        crate::validate::validate(&record).expect("a correction of the fixture record validates");
        record
    }

    /// Makes a record provisional, which is what `publishable` refuses.
    ///
    /// A disagreement is what does it, and the record model keeps one on
    /// purpose: `validate` accepts it and `publishable` refuses it.
    /// ⚠ The observations have to actually differ. `E-COR-14` refuses a claimed
    /// disagreement over observations that are all equal, which is the schema
    /// keeping a conflict honest rather than an obstacle here.
    pub(crate) fn make_provisional(record: &mut Profile) {
        let entry = record
            .corroboration
            .first_mut()
            .expect("the fixture record carries corroboration");
        entry.observations[0].seen = crate::agreement::SeenValue::Bytes(
            crate::canonical::HexBytes::parse("00112233").expect("observed bytes"),
        );
        entry.agreement = crate::Agreement::Disagrees;
        entry.conflict = Some(
            crate::canonical::Label::parse("the two connectors read different bytes")
                .expect("a recorded fact"),
        );
        crate::validate::validate(record).expect("a record carrying a conflict still validates");
        assert!(
            crate::agreement::publishable(record).is_err(),
            "a disagreement is not publishable"
        );
    }

    /// The store as the fixtures ship it, whose version no scheme orders.
    fn fixture_corpus() -> Corpus {
        let record = profile();
        let manifest = RunManifest::from_json(MANIFEST).expect("the fixture manifest validates");
        let key = StoreKey::of_profile(&record);
        let profile_path = key.profile_path().expect("a publishable path");
        let manifest_path = key.manifest_path().expect("a publishable path");
        let mut tree = StoreTree::new();
        tree.insert(
            profile_path.clone(),
            Entry::Object(ObjectRef {
                bytes: PROFILE.len() as u64,
                sha256: Sha256Digest::of(PROFILE.as_bytes()),
            }),
        );
        let mut corpus = Corpus::new(tree);
        corpus.insert_profile(profile_path, record);
        corpus.insert_manifest(manifest_path, manifest);
        corpus
    }

    /// ⛔ Literals, not the constants that spell them. These strings are in a
    /// document a consumer parses, so a rename is a schema change.
    #[test]
    fn index_kinds_have_one_spelling() {
        assert_eq!(INDEX_SCHEMA, "bit-ids/index/1");
        assert_eq!(PEER_ID_FIELD, "peer_wire/peer_id");
        assert_eq!(BEP10_CLIENT_FIELD, "peer_wire/bep10.client");
        let spelled: Vec<&str> = IndexKind::ALL.iter().map(|kind| kind.as_str()).collect();
        assert_eq!(
            spelled,
            [
                "target",
                "peer_prefix",
                "bep10_client",
                "platform",
                "version",
                "captured_at"
            ]
        );
    }

    /// ⛔ The Prove's first half, and it is a real question rather than a
    /// tautology: the builder walks maps, and a map's iteration order is not a
    /// contract a consumer can check.
    #[test]
    fn two_builds_are_byte_identical() {
        let first = build(&corpus(), &schemes()).expect("a publishable store");
        let second = build(&corpus(), &schemes()).expect("a publishable store");
        assert_eq!(first.to_json(), second.to_json());
        assert_eq!(first.digest(), second.digest());
        assert!(first.rows() > 0, "an empty index proves nothing");
    }

    /// ⛔ **Determinism against the insertion order, not just against a second
    /// identical run.** Two builds over one corpus agreed even with the sort
    /// removed, because the corpus was read in one order both times: the sort
    /// was load-bearing and nothing could see it. Found by planting against it.
    #[test]
    fn the_document_does_not_depend_on_the_order_records_were_read_in() {
        let forward = corpus_at(&["1.2.3", "1.2.10", "1.2.9"]);
        let backward = corpus_at(&["1.2.9", "1.2.10", "1.2.3"]);
        let first = build(&forward, &schemes()).expect("a publishable store");
        let second = build(&backward, &schemes()).expect("a publishable store");
        assert_eq!(first.to_json(), second.to_json());
        assert!(first.rows() >= 18, "an empty document proves nothing");
    }

    /// ⛔ A provisional record is in the store and in no view. Nothing tested
    /// this until a plant removed the filter and every test stayed green.
    #[test]
    fn a_provisional_record_is_left_out_of_every_view() {
        let mut record = record_at("1.2.3");
        make_provisional(&mut record);

        let path = StoreKey::of_profile(&record)
            .profile_path()
            .expect("a publishable path");
        let mut corpus = Corpus::new(StoreTree::new());
        corpus.insert_profile(path, record);

        let indexes = build(&corpus, &schemes()).expect("a store of one provisional record");
        assert_eq!(indexes.excluded, 1);
        assert_eq!(indexes.latest.len(), 0, "no latest row");
        assert_eq!(indexes.rows(), 0, "and no lookup row either");
    }

    /// ⛔ A corrected record answers nothing and is still in the store. This is
    /// the whole of `CORPUS-04`: the append-only rule keeps the bytes, and the
    /// views stop pointing at them.
    #[test]
    fn a_superseded_record_leaves_every_view() {
        let original = record_at("1.2.3");
        let fix = correction_at("1.2.3", "cap-re-run", original.id);
        assert_ne!(original.id, fix.id, "a re-capture is a different record");

        let corpus = corpus_of(vec![original.clone(), fix.clone()]);
        let indexes = build(&corpus, &schemes()).expect("a store with one correction");

        assert_eq!(
            indexes.superseded, 1,
            "the original is counted as corrected"
        );
        assert_eq!(indexes.excluded, 0, "and it was publishable, not excluded");

        assert_eq!(indexes.latest.len(), 1, "one build line, one row");
        assert_eq!(
            indexes.latest[0].record, fix.id,
            "the latest view names the correction"
        );

        for index in &indexes.indexes {
            for row in &index.rows {
                assert_ne!(
                    row.record,
                    original.id,
                    "{} still points at the corrected record",
                    index.kind.as_str()
                );
            }
        }

        assert_eq!(indexes.corrections.len(), 1);
        let row = &indexes.corrections[0];
        assert_eq!(row.superseded, original.id);
        assert_eq!(row.by, fix.id);
        assert_eq!(row.current, fix.id);
        rows_resolve(&indexes, &corpus).expect("every correction row resolves");
    }

    /// ⛔ A correction can itself be corrected, and a row that stopped at the
    /// first step would answer with a record that no longer answers. `by` and
    /// `current` are two facts and this is the case that tells them apart.
    #[test]
    fn a_corrected_correction_names_the_end_of_the_chain() {
        let original = record_at("1.2.3");
        let first = correction_at("1.2.3", "cap-re-run", original.id);
        let second = correction_at("1.2.3", "cap-third-run", first.id);

        let corpus = corpus_of(vec![original.clone(), first.clone(), second.clone()]);
        let indexes = build(&corpus, &schemes()).expect("a store with a two-step chain");

        assert_eq!(indexes.superseded, 2, "both earlier records are corrected");
        assert_eq!(indexes.latest.len(), 1);
        assert_eq!(indexes.latest[0].record, second.id);

        let from_original = indexes
            .corrections
            .iter()
            .find(|row| row.superseded == original.id)
            .expect("the original has a correction row");
        assert_eq!(
            from_original.by, first.id,
            "the immediate successor is the first correction"
        );
        assert_eq!(
            from_original.current, second.id,
            "and the end of the chain is the second"
        );
        rows_resolve(&indexes, &corpus).expect("every correction row resolves");
    }

    /// ⛔ Two corrections of one record leave no answer to "what replaces this",
    /// and choosing one would discard the other's adjudication silently.
    #[test]
    fn two_records_correcting_one_are_refused() {
        let original = record_at("1.2.3");
        let one = correction_at("1.2.3", "cap-re-run", original.id);
        let two = correction_at("1.2.3", "cap-other-run", original.id);
        assert_ne!(one.id, two.id);

        let corpus = corpus_of(vec![original, one, two]);
        let violations = build(&corpus, &schemes()).expect_err("a fork has no answer");
        assert!(violations.has("E-VIW-03"), "{violations}");
    }

    /// ⛔ A cycle is constructible, which is the reason the walk is bounded. A
    /// record identifier digests the identity tuple and not `supersedes`, so
    /// two records can each name the other while neither supersedes itself,
    /// and self-supersession is the only shape `validate` refuses.
    #[test]
    fn a_correction_chain_that_returns_to_its_start_is_refused() {
        let original = record_at("1.2.3");
        let other = correction_at("1.2.4", "cap-1-2-4", original.id);
        // Rebuilt at the original's own version and capture, so this carries the
        // original's identifier exactly and closes the loop.
        let back = correction_at("1.2.3", "cap-1-2-3", other.id);
        assert_eq!(back.id, original.id, "the loop is actually closed");

        let corpus = corpus_of(vec![back, other]);
        let violations = build(&corpus, &schemes()).expect_err("a cycle has no end");
        assert!(violations.has("E-VIW-04"), "{violations}");

        // ⚠ MEASURED, NOT ASSUMED, because it is the residual this leaves, and
        // the first version of this assertion was wrong. `validate_corpus` does
        // refuse this store, on `E-CRP-01`: it carries no run manifests, which
        // has nothing to do with the cycle. What it does not report is
        // `E-CRP-07`, whose whole question is whether the record a correction
        // names exists, and in a cycle both do. So nothing at corpus level sees
        // a cycle; it is caught when a view is derived over the store, which is
        // before anything can be published and after a caller who never derives
        // one would have noticed nothing.
        let at_corpus = crate::corpus::validate_corpus(&corpus)
            .expect_err("this store carries no run manifests either");
        assert!(
            !at_corpus.has("E-CRP-07"),
            "the cycle's own records both exist, so the supersession rule is satisfied: \
             {at_corpus}"
        );
    }

    /// ⛔ A correction that is not itself publishable retracts nothing. Letting
    /// it would leave the build line answering with no record at all, which is
    /// losing a measurement to one that is not fit to replace it.
    #[test]
    fn a_provisional_correction_retracts_nothing() {
        let original = record_at("1.2.3");
        let mut fix = correction_at("1.2.3", "cap-re-run", original.id);
        make_provisional(&mut fix);

        let corpus = corpus_of(vec![original.clone(), fix]);
        let indexes = build(&corpus, &schemes()).expect("a store with a provisional correction");

        assert_eq!(indexes.superseded, 0, "nothing was retracted");
        assert_eq!(indexes.excluded, 1, "the correction itself is left out");
        assert_eq!(
            indexes.corrections.len(),
            0,
            "and there is no chain to walk"
        );
        assert_eq!(indexes.latest.len(), 1);
        assert_eq!(
            indexes.latest[0].record, original.id,
            "the original still answers"
        );
    }

    /// ⛔ A correction row names two records it is not pointing a reader at, and
    /// `rows_resolve` has to check those too. A superseded identifier resolving
    /// to nothing makes the chain unwalkable in exactly the case it exists for.
    #[test]
    fn a_correction_row_naming_an_absent_record_is_refused() {
        let original = record_at("1.2.3");
        let fix = correction_at("1.2.3", "cap-re-run", original.id);
        let corpus = corpus_of(vec![original.clone(), fix]);
        let mut indexes = build(&corpus, &schemes()).expect("a store with one correction");

        indexes.corrections[0].superseded = record_at("9.9.9").id;
        let violations =
            rows_resolve(&indexes, &corpus).expect_err("the superseded record is not in the store");
        assert!(violations.has("E-VIW-10"), "{violations}");
    }

    /// ⚠ A peer ID whose first span varies has no prefix to key on, and a row
    /// keyed on an empty string would answer every lookup. The fixture's own
    /// peer ID begins with a fixed run, so this builds one that does not.
    #[test]
    fn a_peer_id_whose_first_span_varies_has_no_prefix_row() {
        use crate::observation::{BytePattern, FieldState, PatternRun, PatternedValue};
        use core::num::NonZeroU32;

        let mut record = record_at("1.2.3");
        let path = crate::observation::FieldPath::parse(PEER_ID_FIELD).expect("a field path");
        let observed = record
            .observations
            .iter_mut()
            .find(|field| field.path == path)
            .expect("the fixture record measures a peer id");
        observed.state = FieldState::Patterned(PatternedValue {
            pattern: BytePattern {
                length: 20,
                runs: vec![
                    PatternRun::Varying {
                        length: 12,
                        alphabet: None,
                    },
                    PatternRun::Fixed {
                        bytes: crate::canonical::HexBytes::parse("2d5858303030302d")
                            .expect("observed bytes"),
                    },
                ],
            },
            samples: NonZeroU32::new(8).expect("a sample count"),
        });
        crate::validate::validate(&record).expect("a varying-first pattern is a valid record");

        let path = StoreKey::of_profile(&record)
            .profile_path()
            .expect("a publishable path");
        let mut corpus = Corpus::new(StoreTree::new());
        corpus.insert_profile(path, record);

        let indexes = build(&corpus, &schemes()).expect("a publishable store");
        let prefixes = indexes
            .indexes
            .iter()
            .find(|index| index.kind == IndexKind::PeerPrefix)
            .expect("a peer prefix index");
        assert!(
            prefixes.rows.is_empty(),
            "a varying first span is not a prefix: {:?}",
            prefixes.rows
        );
        // The record is still findable by everything it does declare.
        let targets = indexes
            .indexes
            .iter()
            .find(|index| index.kind == IndexKind::Target)
            .expect("a target index");
        assert_eq!(targets.rows.len(), 1);
    }

    /// ⛔ The Prove's second half, checked against the store rather than against
    /// the builder that produced the rows.
    #[test]
    fn every_row_resolves_to_one_record() {
        let corpus = corpus();
        let indexes = build(&corpus, &schemes()).expect("a publishable store");
        rows_resolve(&indexes, &corpus).expect("every row names a record the store carries");

        // A row pointing at a store that does not carry it is refused, so the
        // check above is not vacuous.
        let empty = Corpus::new(StoreTree::new());
        let violations =
            rows_resolve(&indexes, &empty).expect_err("no record resolves in an empty store");
        assert!(violations.has("E-VIW-10"), "{violations}");
    }

    #[test]
    fn a_row_filed_elsewhere_is_refused() {
        let corpus = corpus();
        let mut indexes = build(&corpus, &schemes()).expect("a publishable store");
        indexes.indexes[0].rows[0].path =
            RelPath::parse("profiles/v1/elsewhere.json").expect("a canonical path");
        let violations = rows_resolve(&indexes, &corpus).expect_err("a row filed elsewhere");
        assert!(violations.has("E-VIW-11"), "{violations}");
    }

    /// ⭐ The peer prefix is the fixed span of a measured value, which is the
    /// whole product: a consumer sees `-XX0000-` on the wire and this says which
    /// build emitted it. The fixture's peer ID is patterned with that prefix.
    #[test]
    fn the_peer_prefix_row_is_the_measured_fixed_span() {
        let indexes = build(&corpus(), &schemes()).expect("a publishable store");
        let prefixes = indexes
            .indexes
            .iter()
            .find(|index| index.kind == IndexKind::PeerPrefix)
            .expect("a peer prefix index");
        assert_eq!(prefixes.rows.len(), 1);
        // `-XX0000-`, the fixed run of the fixture's patterned peer ID.
        assert_eq!(prefixes.rows[0].key, "2d5858303030302d");
    }

    /// ⛔ Ordered numerically, not as text. `1.2.10` sorts before `1.2.9` as a
    /// string, and a latest view that answered `1.2.9` would point a consumer at
    /// a superseded build with full confidence.
    #[test]
    fn the_latest_view_names_the_newest_record_per_build_line() {
        let corpus = corpus_at(&["1.2.3", "1.2.10", "1.2.9"]);
        let indexes = build(&corpus, &schemes()).expect("a publishable store");
        assert_eq!(indexes.latest.len(), 1, "one build line");
        assert_eq!(indexes.latest[0].target.as_str(), "fixture-client");
        assert_eq!(indexes.latest[0].version.as_str(), "1.2.10");
        assert_eq!(indexes.excluded, 0);

        // Every record still appears in the lookup indexes: finding a
        // measurement does not require it to be the newest one.
        let versions = indexes
            .indexes
            .iter()
            .find(|index| index.kind == IndexKind::Version)
            .expect("a version index");
        assert_eq!(versions.rows.len(), 3);
    }

    /// ⛔ Fails closed. An unorderable version blocks the view rather than being
    /// skipped, because a skip yields an older record selected confidently.
    #[test]
    fn an_unorderable_version_blocks_the_latest_view() {
        let strict = BTreeMap::from([(
            slug("fixture-client"),
            VersionScheme {
                tag_prefix: None,
                min_components: 4,
                max_components: 4,
                floor_major: None,
            },
        )]);
        let violations = build(&corpus(), &strict).expect_err("1.2.3 is not four numbers");
        assert!(violations.has("E-VIW-02"), "{violations}");

        let none = BTreeMap::new();
        let violations = build(&corpus(), &none).expect_err("no scheme, no ordering");
        assert!(violations.has("E-VIW-01"), "{violations}");

        // ⚠ And the shipped fixture is itself unorderable, which is why the
        // cases above build their own records rather than using it.
        let violations =
            build(&fixture_corpus(), &schemes()).expect_err("0.0.0-fixture is not three numbers");
        assert!(violations.has("E-VIW-02"), "{violations}");
    }
}
