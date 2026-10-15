//! The run manifest: how a capture was produced, and from what.
//!
//! A profile says what a build put on the wire. This says what was running when
//! it did: which host, how isolated, which tools at which versions, which
//! phases in which order, what was retrieved from where, and what was scrubbed
//! out of the evidence before it was kept. Without it a profile is a
//! conclusion nobody can replay.
//!
//! It is a separate document from the profile, living beside the raw bytes it
//! describes. The two overlap deliberately, and [`bind`] is what stops the
//! overlap from drifting: a value in two places with nothing checking that they
//! agree is the copy a reader trusts being the wrong one.

use serde::{Deserialize, Serialize};

use crate::acquisition::SignatureStatus;
use crate::canonical::{CanonicalError, Instant, Label, RelPath, Sha256Digest, Slug, Url, Version};
use crate::record::{EvidenceKind, Profile};
use crate::sampling::SamplingPlan;
use crate::validate::{SchemaError, Violations, strictly_ascending};

/// Identifier carried by every first-generation run manifest.
pub const MANIFEST_SCHEMA: &str = "bit-ids/manifest/1";

/// The schema identifier a manifest declares.
///
/// A separate type from the profile's, because the two documents version
/// independently: extending the manifest does not invalidate a profile.
#[derive(Clone, Debug, Eq, PartialEq, Ord, PartialOrd, Hash)]
pub struct ManifestSchema(&'static str);

impl ManifestSchema {
    /// The schema this build of the crate reads and writes.
    #[must_use]
    pub const fn current() -> Self {
        Self(MANIFEST_SCHEMA)
    }

    /// Parses a declared schema identifier.
    ///
    /// # Errors
    ///
    /// Returns an error for any identifier other than [`MANIFEST_SCHEMA`].
    pub fn parse(text: &str) -> Result<Self, CanonicalError> {
        if text == MANIFEST_SCHEMA {
            Ok(Self(MANIFEST_SCHEMA))
        } else {
            Err(CanonicalError::new(
                "manifest-schema-version",
                format!("unsupported schema {text:?}, this build reads {MANIFEST_SCHEMA:?}"),
            ))
        }
    }

    /// The declared identifier.
    #[must_use]
    pub const fn as_str(&self) -> &'static str {
        self.0
    }
}

impl Serialize for ManifestSchema {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(self.0)
    }
}

impl<'de> Deserialize<'de> for ManifestSchema {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        use serde::de::Error as _;
        let text = String::deserialize(deserializer)?;
        Self::parse(&text).map_err(D::Error::custom)
    }
}

/// A step of the capture state machine in `docs/architecture.md` section 8.
///
/// The order is the state machine's order and the discriminants carry it. A run
/// advances one step at a time or falls to [`PhaseName::Provisional`]; it never
/// skips forward, because a phase nobody ran is a phase nobody can produce
/// evidence for.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Ord, PartialOrd, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PhaseName {
    /// The target and version to attempt were chosen.
    Planned,
    /// Every configured route independently resolved the newest stable version.
    Resolved,
    /// Two independent routes delivered the build.
    AcquiredTwice,
    /// Both installations reported the same version.
    Installed,
    /// The observer and the connectors ran against the fixture.
    Observed,
    /// Overlapping observations were compared.
    Corroborated,
    /// The record satisfied the publication invariants.
    Validated,
    /// Something did not hold. Evidence is retained and the run stops here.
    Provisional,
}

impl PhaseName {
    /// Position in the state machine, or `None` for the terminal failure.
    #[must_use]
    pub const fn ordinal(self) -> Option<u8> {
        match self {
            Self::Planned => Some(0),
            Self::Resolved => Some(1),
            Self::AcquiredTwice => Some(2),
            Self::Installed => Some(3),
            Self::Observed => Some(4),
            Self::Corroborated => Some(5),
            Self::Validated => Some(6),
            Self::Provisional => None,
        }
    }

    /// The canonical spelling.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Planned => "planned",
            Self::Resolved => "resolved",
            Self::AcquiredTwice => "acquired_twice",
            Self::Installed => "installed",
            Self::Observed => "observed",
            Self::Corroborated => "corroborated",
            Self::Validated => "validated",
            Self::Provisional => "provisional",
        }
    }
}

/// One step of a run, with the wall clock either side of it.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Phase {
    /// Which step.
    pub name: PhaseName,
    /// When it started, UTC.
    pub started_at: Instant,
    /// When it ended, UTC.
    pub ended_at: Instant,
    /// What the step did, in one recorded line.
    pub detail: Label,
}

/// What a tool was doing in the run.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ToolRole {
    /// The project's own active observer. Exactly one run has this role.
    Observer,
    /// An independent implementation that overlaps the observation.
    Connector,
    /// Resolved, downloaded or installed the build.
    Acquisition,
    /// Generated the fixture or drove the target.
    Harness,
}

/// One tool that took part, at the exact version that took part.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Tool {
    /// Tool identifier, unique within the manifest.
    pub id: Slug,
    /// The build of it that ran.
    pub version: Version,
    /// What it was doing.
    pub role: ToolRole,
}

/// How isolated the run was.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum IsolationKind {
    /// A container created for this run and destroyed after it.
    DisposableContainer,
    /// A virtual machine created for this run and destroyed after it.
    DisposableVirtualMachine,
    /// A hosted runner that does not survive the job.
    EphemeralRunner,
}

/// What the target could reach.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum NetworkMode {
    /// Only endpoints this run started, on loopback.
    LoopbackOnly,
    /// A private bridge with no route off the host.
    IsolatedBridge,
    /// The target could reach the outside world.
    HostRouted,
}

/// The isolation the run was performed under.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Isolation {
    /// What the pre-flight guard established about the host, before anything
    /// was installed on it.
    ///
    /// ⛔ `host.disposable` is a claim the run makes about itself, and
    /// `E-MAN-30` refuses a run that admits otherwise. It cannot catch a run
    /// that is wrong: a runner misconfigured to persist its disk says
    /// `disposable: true` and means it. This is the guard's own answer, taken
    /// before the install, and `ACQ-04` owns it.
    pub claim: HostClaim,
    /// What kind of disposable host.
    pub host: IsolationKind,
    /// What the target could reach.
    pub network: NetworkMode,
    /// Why the target needed to reach anything beyond loopback. Required for
    /// any mode but [`NetworkMode::LoopbackOnly`] and forbidden for it, because
    /// an unexplained route off the host is a capture nobody can bound.
    pub external_reason: Option<Label>,
}

/// What the pre-flight guard read off the host before the run.
///
/// `bit-check assert-disposable` produces both values. The
/// fingerprint is what a later job compares to see it is on a different
/// machine; the instant is when the host was claimed, which is necessarily
/// before the run it was claimed for.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HostClaim {
    /// Digest of the host identity the guard read.
    pub fingerprint: Sha256Digest,
    /// When the guard claimed the host, UTC.
    pub claimed_at: Instant,
}

/// The machine the run happened on.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Host {
    /// The runner or base image, as the platform names it.
    pub image: Label,
    /// The kernel string, read from the machine.
    pub kernel: Label,
    /// The operating system, read from the machine.
    pub os: Label,
    /// Machine architecture.
    pub arch: Slug,
    /// Whether the host is destroyed after the run. A capture that could alter
    /// a host somebody keeps is refused, so this is recorded rather than
    /// assumed.
    pub disposable: bool,
}

/// The two clocks a run is measured against.
///
/// Wall time orders records and is comparable across runs. A monotonic elapsed
/// time is what timing claims rest on, because a wall clock can step.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Clocks {
    /// When the run started, UTC.
    pub wall_start: Instant,
    /// When the run ended, UTC.
    pub wall_end: Instant,
    /// How long it took, from a monotonic source.
    pub monotonic_elapsed_ns: u64,
}

/// Where one route's artifact came from, and what arrived.
///
/// `ACQ-01` owns the full route record: resolver evidence, package metadata and
/// the mirrors tried. What is here is the identity a replay needs.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct AcquisitionIdentity {
    /// Route identifier, matching the profile's.
    pub route: Slug,
    /// Where the artifact was retrieved from.
    pub source: Url,
    /// When it was retrieved, UTC.
    pub retrieved_at: Instant,
    /// Digest of the bytes that arrived.
    pub artifact: Sha256Digest,
    /// How many bytes arrived.
    pub bytes: u64,
    /// What was done about the signature.
    pub signature: SignatureStatus,
    /// The version the installed build reported when asked.
    pub installed_version: Version,
}

/// A class of value scrubbed from evidence before it was kept.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RedactionRule {
    /// An environment variable's value.
    EnvironmentVariable,
    /// An absolute path naming a user or a machine.
    AbsolutePath,
    /// A hostname.
    Hostname,
    /// An address of the host or its network.
    IpAddress,
    /// Anything credential shaped.
    Credential,
    /// An account name.
    UserName,
}

/// What was removed from one artifact, and how much of it.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Redaction {
    /// The evidence record it was removed from.
    pub evidence: Slug,
    /// What class of value.
    pub rule: RedactionRule,
    /// How many were replaced. Zero would mean the rule did nothing and should
    /// not be declared.
    pub occurrences: u32,
}

/// One artifact in the bundle, addressed by what it contains.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct EvidenceRecord {
    /// Evidence identifier, matching the profile's citation.
    pub id: Slug,
    /// What kind of artifact.
    pub kind: EvidenceKind,
    /// The readable path the profile cites.
    pub path: RelPath,
    /// Its exact size.
    pub bytes: u64,
    /// Its digest.
    pub sha256: Sha256Digest,
    /// The tool that produced it.
    pub produced_by: Slug,
    /// The phase it came out of.
    pub phase: PhaseName,
    /// Whether anything was scrubbed from it. A true here needs a matching
    /// declaration in `redactions`, and a declaration needs a true here.
    pub redacted: bool,
}

impl EvidenceRecord {
    /// Where the artifact sits in a content-addressed store.
    ///
    /// Derived from the digest rather than recorded, so it cannot disagree with
    /// the bytes it names. Two runs that captured identical bytes land on one
    /// object, which is what keeps an append-only store from growing a copy per
    /// capture.
    ///
    /// The slicing below is total: [`Sha256Digest`] renders exactly its prefix
    /// plus 64 ASCII hex digits, so the string is always long enough and every
    /// boundary is a character boundary.
    #[must_use]
    pub fn object_path(&self) -> String {
        let hex = self.sha256.to_string();
        let hex = hex.strip_prefix(Sha256Digest::PREFIX).unwrap_or(&hex);
        format!("objects/sha256/{}/{}/{}", &hex[0..2], &hex[2..4], &hex[4..])
    }
}

/// The manifest's fields, as they sit in a document.
///
/// ⛔ Exists so that [`RunManifest`] does not derive `Deserialize`. The same
/// door was found open on `Profile` during `SCHEMA-01`: a derived
/// implementation lets `serde_json::from_str` hand back an unvalidated record,
/// which is a control on one path into an operation and not its siblings.
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct RunManifestFields {
    pub(crate) schema: ManifestSchema,
    pub(crate) capture: Slug,
    pub(crate) target: Slug,
    pub(crate) version: Version,
    pub(crate) platform: Slug,
    pub(crate) arch: Slug,
    pub(crate) package: Slug,
    pub(crate) host: Host,
    pub(crate) isolation: Isolation,
    pub(crate) clocks: Clocks,
    pub(crate) sampling: SamplingPlan,
    pub(crate) tools: Vec<Tool>,
    pub(crate) acquisition: Vec<AcquisitionIdentity>,
    pub(crate) phases: Vec<Phase>,
    pub(crate) evidence: Vec<EvidenceRecord>,
    pub(crate) redactions: Vec<Redaction>,
}

impl From<RunManifestFields> for RunManifest {
    fn from(fields: RunManifestFields) -> Self {
        let RunManifestFields {
            schema,
            capture,
            target,
            version,
            platform,
            arch,
            package,
            host,
            isolation,
            clocks,
            sampling,
            tools,
            acquisition,
            phases,
            evidence,
            redactions,
        } = fields;
        Self {
            schema,
            capture,
            target,
            version,
            platform,
            arch,
            package,
            host,
            isolation,
            clocks,
            sampling,
            tools,
            acquisition,
            phases,
            evidence,
            redactions,
        }
    }
}

impl<'de> Deserialize<'de> for RunManifest {
    /// Every serde route into a manifest validates, for the reason on
    /// [`RunManifestFields`].
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        use serde::de::Error as _;
        let manifest = Self::from(RunManifestFields::deserialize(deserializer)?);
        validate_manifest(&manifest).map_err(D::Error::custom)?;
        Ok(manifest)
    }
}

/// Everything about one capture run except what it observed.
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct RunManifest {
    /// The schema this manifest is written against.
    pub schema: ManifestSchema,
    /// The capture run this describes, matching the profile's.
    pub capture: Slug,
    /// The catalogue target.
    pub target: Slug,
    /// The version the installed build reported.
    pub version: Version,
    /// Host family.
    pub platform: Slug,
    /// Machine architecture.
    pub arch: Slug,
    /// Package format the artifact was delivered in.
    pub package: Slug,
    /// The machine it ran on.
    pub host: Host,
    /// How isolated it was.
    pub isolation: Isolation,
    /// When, by both clocks.
    pub clocks: Clocks,
    /// What the run varied between samples. A plan that varied nothing can
    /// support no claim about how long a value lasts.
    pub sampling: SamplingPlan,
    /// Every tool that took part, sorted by identifier.
    pub tools: Vec<Tool>,
    /// Every route the build was obtained through, sorted by identifier.
    pub acquisition: Vec<AcquisitionIdentity>,
    /// The steps the run went through, in order.
    pub phases: Vec<Phase>,
    /// Every artifact kept, sorted by identifier.
    pub evidence: Vec<EvidenceRecord>,
    /// What was scrubbed, and from where.
    pub redactions: Vec<Redaction>,
}

impl RunManifest {
    /// The tool with one identifier, when the manifest declares it.
    #[must_use]
    pub fn tool(&self, id: &Slug) -> Option<&Tool> {
        self.tools.iter().find(|tool| &tool.id == id)
    }

    /// The evidence record with one identifier, when the manifest declares it.
    #[must_use]
    pub fn evidence_record(&self, id: &Slug) -> Option<&EvidenceRecord> {
        self.evidence.iter().find(|record| &record.id == id)
    }

    /// The single tool acting as the observer, when exactly one does.
    #[must_use]
    pub fn observer(&self) -> Option<&Tool> {
        let mut observers = self
            .tools
            .iter()
            .filter(|tool| tool.role == ToolRole::Observer);
        let first = observers.next()?;
        if observers.next().is_some() {
            return None;
        }
        Some(first)
    }
}

// -- invariants -------------------------------------------------------------

fn check_phases(manifest: &RunManifest, out: &mut Vec<SchemaError>) {
    let phases = &manifest.phases;
    if phases.is_empty() {
        out.push(SchemaError::new(
            "E-MAN-01",
            "phases",
            "a run with no phases describes nothing",
        ));
        return;
    }
    if phases[0].name != PhaseName::Planned {
        out.push(SchemaError::new(
            "E-MAN-02",
            "phases[0]",
            format!("a run starts planned, not {}", phases[0].name.as_str()),
        ));
    }
    for (index, phase) in phases.iter().enumerate() {
        let at = format!("phases[{index}] {}", phase.name.as_str());
        if phase.ended_at < phase.started_at {
            out.push(SchemaError::new(
                "E-MAN-05",
                &at,
                format!(
                    "ends {} before it starts {}",
                    phase.ended_at, phase.started_at
                ),
            ));
        }
        if phase.name == PhaseName::Provisional && index + 1 != phases.len() {
            out.push(SchemaError::new(
                "E-MAN-04",
                &at,
                "a run that fell to provisional does not continue",
            ));
        }
        if phases[..index].iter().any(|prior| prior.name == phase.name) {
            out.push(SchemaError::new("E-MAN-07", &at, "runs twice"));
        }
        if index > 0 {
            let previous = &phases[index - 1];
            if phase.started_at < previous.ended_at {
                out.push(SchemaError::new(
                    "E-MAN-06",
                    &at,
                    format!(
                        "starts {} before the previous phase ended {}",
                        phase.started_at, previous.ended_at
                    ),
                ));
            }
            // ⚠ A step at a time or a fall to provisional. Skipping forward
            // would claim a phase produced evidence nobody ran.
            if let (Some(here), Some(before)) = (phase.name.ordinal(), previous.name.ordinal())
                && here != before + 1
            {
                out.push(SchemaError::new(
                    "E-MAN-03",
                    &at,
                    format!(
                        "follows {} without the steps between",
                        previous.name.as_str()
                    ),
                ));
            }
        }
    }
}

fn check_tools(manifest: &RunManifest, out: &mut Vec<SchemaError>) {
    if let Some(index) = strictly_ascending(&manifest.tools, |tool| tool.id.to_string()) {
        out.push(SchemaError::new(
            "E-MAN-22",
            format!("tools[{index}]"),
            format!(
                "tool ids must be unique and ascending, found {}",
                manifest.tools[index].id
            ),
        ));
    }
    if manifest.observer().is_none() {
        let count = manifest
            .tools
            .iter()
            .filter(|tool| tool.role == ToolRole::Observer)
            .count();
        out.push(SchemaError::new(
            "E-MAN-20",
            "tools",
            format!("exactly one tool is the observer, found {count}"),
        ));
    }
    if !manifest
        .tools
        .iter()
        .any(|tool| tool.role == ToolRole::Connector)
    {
        out.push(SchemaError::new(
            "E-MAN-21",
            "tools",
            "no independent connector took part, so nothing corroborates the observer",
        ));
    }
}

fn check_environment(manifest: &RunManifest, out: &mut Vec<SchemaError>) {
    if !manifest.host.disposable {
        out.push(SchemaError::new(
            "E-MAN-30",
            "host.disposable",
            "a capture that could alter a host somebody keeps is refused",
        ));
    }
    let loopback = manifest.isolation.network == NetworkMode::LoopbackOnly;
    match (&manifest.isolation.external_reason, loopback) {
        (None, false) => out.push(SchemaError::new(
            "E-MAN-31",
            "isolation.external_reason",
            "a route off loopback needs a recorded reason, or the capture is unbounded",
        )),
        (Some(reason), true) => out.push(SchemaError::new(
            "E-MAN-32",
            "isolation.external_reason",
            format!("loopback-only needs no reason, found {reason}"),
        )),
        _ => {}
    }
    // ⛔ The host is claimed before the run, never during or after. A claim
    // stamped inside the run is a guard that ran too late to be a boundary,
    // which is the whole distinction between this and `E-MAN-30`.
    if manifest.isolation.claim.claimed_at > manifest.clocks.wall_start {
        out.push(SchemaError::new(
            "E-MAN-33",
            "isolation.claim.claimed_at",
            format!(
                "claimed at {} and the run started at {}; a guard that ran after the \
                 install is a report, not a boundary",
                manifest.isolation.claim.claimed_at, manifest.clocks.wall_start
            ),
        ));
    }
    if manifest.clocks.wall_end < manifest.clocks.wall_start {
        out.push(SchemaError::new(
            "E-MAN-40",
            "clocks",
            format!(
                "ends {} before it starts {}",
                manifest.clocks.wall_end, manifest.clocks.wall_start
            ),
        ));
    }
    if manifest.clocks.monotonic_elapsed_ns == 0 {
        out.push(SchemaError::new(
            "E-MAN-41",
            "clocks.monotonic_elapsed_ns",
            "a run that took no measurable time did not run",
        ));
    }
    for (index, phase) in manifest.phases.iter().enumerate() {
        if phase.started_at < manifest.clocks.wall_start
            || phase.ended_at > manifest.clocks.wall_end
        {
            out.push(SchemaError::new(
                "E-MAN-42",
                format!("phases[{index}] {}", phase.name.as_str()),
                "falls outside the window the run records for itself",
            ));
        }
    }
}

fn check_acquisition(manifest: &RunManifest, out: &mut Vec<SchemaError>) {
    let routes = &manifest.acquisition;
    if routes.len() < 2 {
        out.push(SchemaError::new(
            "E-MAN-10",
            "acquisition",
            format!(
                "{} route(s); two independent routes are required",
                routes.len()
            ),
        ));
    }
    if let Some(index) = strictly_ascending(routes, |route| route.route.to_string()) {
        out.push(SchemaError::new(
            "E-MAN-11",
            format!("acquisition[{index}]"),
            format!(
                "route ids must be unique and ascending, found {}",
                routes[index].route
            ),
        ));
    }
    for (index, route) in routes.iter().enumerate() {
        if route.installed_version != manifest.version {
            out.push(SchemaError::new(
                "E-MAN-12",
                format!("acquisition[{index}].installed_version"),
                format!(
                    "route {} installed {}, the run records {}",
                    route.route, route.installed_version, manifest.version
                ),
            ));
        }
        if route.bytes == 0 {
            out.push(SchemaError::new(
                "E-MAN-13",
                format!("acquisition[{index}].bytes"),
                format!("route {} delivered no bytes", route.route),
            ));
        }
    }
}

fn check_evidence(manifest: &RunManifest, out: &mut Vec<SchemaError>) {
    let evidence = &manifest.evidence;
    if let Some(index) = strictly_ascending(evidence, |record| record.id.to_string()) {
        out.push(SchemaError::new(
            "E-MAN-50",
            format!("evidence[{index}]"),
            format!(
                "evidence ids must be unique and ascending, found {}",
                evidence[index].id
            ),
        ));
    }
    for (index, record) in evidence.iter().enumerate() {
        let at = format!("evidence[{index}] {}", record.id);
        if record.bytes == 0 {
            out.push(SchemaError::new(
                "E-MAN-51",
                &at,
                "a parsed value with no recoverable bytes is not a measurement",
            ));
        }
        if manifest.tool(&record.produced_by).is_none() {
            out.push(SchemaError::new(
                "E-MAN-52",
                &at,
                format!(
                    "produced by {}, which the run does not declare",
                    record.produced_by
                ),
            ));
        }
        if !manifest
            .phases
            .iter()
            .any(|phase| phase.name == record.phase)
        {
            out.push(SchemaError::new(
                "E-MAN-53",
                &at,
                format!(
                    "came out of {}, a phase this run did not run",
                    record.phase.as_str()
                ),
            ));
        }
        if evidence[..index]
            .iter()
            .any(|prior| prior.path == record.path)
        {
            out.push(SchemaError::new(
                "E-MAN-54",
                &at,
                format!("{} is claimed by more than one record", record.path),
            ));
        }
    }
}

fn check_redactions(manifest: &RunManifest, out: &mut Vec<SchemaError>) {
    for (index, redaction) in manifest.redactions.iter().enumerate() {
        let at = format!("redactions[{index}] {}", redaction.evidence);
        let Some(record) = manifest.evidence_record(&redaction.evidence) else {
            out.push(SchemaError::new(
                "E-MAN-60",
                &at,
                "names an evidence record the run does not carry",
            ));
            continue;
        };
        if redaction.occurrences == 0 {
            out.push(SchemaError::new(
                "E-MAN-61",
                &at,
                "a rule that replaced nothing is not a redaction",
            ));
        }
        if !record.redacted {
            out.push(SchemaError::new(
                "E-MAN-63",
                &at,
                "the evidence it names is not marked redacted",
            ));
        }
        if manifest.redactions[..index]
            .iter()
            .any(|prior| prior.evidence == redaction.evidence && prior.rule == redaction.rule)
        {
            out.push(SchemaError::new("E-MAN-64", &at, "declared twice"));
        }
    }
    for (index, record) in manifest.evidence.iter().enumerate() {
        if record.redacted
            && !manifest
                .redactions
                .iter()
                .any(|redaction| redaction.evidence == record.id)
        {
            out.push(SchemaError::new(
                "E-MAN-62",
                format!("evidence[{index}] {}", record.id),
                "is marked redacted and nothing says what was removed",
            ));
        }
    }
}

/// Checks every invariant a run manifest owns.
///
/// # Errors
///
/// Returns every refused invariant, each with a stable code.
pub fn validate_manifest(manifest: &RunManifest) -> Result<(), Violations> {
    let mut out = Vec::new();
    check_phases(manifest, &mut out);
    check_tools(manifest, &mut out);
    check_environment(manifest, &mut out);
    check_acquisition(manifest, &mut out);
    check_evidence(manifest, &mut out);
    check_redactions(manifest, &mut out);
    Violations::from_errors(out)
}

// -- binding ----------------------------------------------------------------

fn bind_identity(manifest: &RunManifest, profile: &Profile, out: &mut Vec<SchemaError>) {
    if manifest.capture != profile.capture.id {
        out.push(SchemaError::new(
            "E-BND-01",
            "capture",
            format!(
                "manifest {}, profile {}",
                manifest.capture, profile.capture.id
            ),
        ));
    }
    if manifest.target != profile.target.id {
        out.push(SchemaError::new(
            "E-BND-02",
            "target",
            format!(
                "manifest {}, profile {}",
                manifest.target, profile.target.id
            ),
        ));
    }
    let manifest_build = (
        &manifest.version,
        &manifest.platform,
        &manifest.arch,
        &manifest.package,
    );
    let profile_build = (
        &profile.build.version,
        &profile.build.platform,
        &profile.build.arch,
        &profile.build.package,
    );
    if manifest_build != profile_build {
        out.push(SchemaError::new(
            "E-BND-03",
            "build",
            format!(
                "manifest {} {} {} {}, profile {} {} {} {}",
                manifest.version,
                manifest.platform,
                manifest.arch,
                manifest.package,
                profile.build.version,
                profile.build.platform,
                profile.build.arch,
                profile.build.package
            ),
        ));
    }
    if profile.capture.captured_at < manifest.clocks.wall_start
        || profile.capture.captured_at > manifest.clocks.wall_end
    {
        out.push(SchemaError::new(
            "E-BND-12",
            "capture.captured_at",
            format!(
                "profile records {}, the run ran {} to {}",
                profile.capture.captured_at, manifest.clocks.wall_start, manifest.clocks.wall_end
            ),
        ));
    }
}

fn bind_evidence(manifest: &RunManifest, profile: &Profile, out: &mut Vec<SchemaError>) {
    for entry in &profile.evidence {
        let Some(record) = manifest.evidence_record(&entry.id) else {
            out.push(SchemaError::new(
                "E-BND-04",
                format!("evidence {}", entry.id),
                "the profile cites it and the run does not carry it",
            ));
            continue;
        };
        if record.kind != entry.kind
            || record.path != entry.path
            || record.bytes != entry.bytes
            || record.sha256 != entry.sha256
        {
            out.push(SchemaError::new(
                "E-BND-05",
                format!("evidence {}", entry.id),
                "the two documents describe the same artifact differently",
            ));
        }
    }
}

fn bind_connectors(manifest: &RunManifest, profile: &Profile, out: &mut Vec<SchemaError>) {
    for connector in &profile.capture.connectors {
        let Some(tool) = manifest.tool(&connector.id) else {
            out.push(SchemaError::new(
                "E-BND-06",
                format!("connector {}", connector.id),
                "the profile names it and the run does not declare it",
            ));
            continue;
        };
        if tool.version != connector.version {
            out.push(SchemaError::new(
                "E-BND-07",
                format!("connector {}", connector.id),
                format!(
                    "run ran {}, profile records {}",
                    tool.version, connector.version
                ),
            ));
        }
    }
    match manifest.observer() {
        Some(observer) if observer.id == profile.capture.observer => {}
        Some(observer) => out.push(SchemaError::new(
            "E-BND-08",
            "capture.observer",
            format!(
                "run observed with {}, profile records {}",
                observer.id, profile.capture.observer
            ),
        )),
        None => out.push(SchemaError::new(
            "E-BND-08",
            "capture.observer",
            "the run declares no single observer to compare against",
        )),
    }
}

fn bind_acquisition(manifest: &RunManifest, profile: &Profile, out: &mut Vec<SchemaError>) {
    let manifest_routes: Vec<&Slug> = manifest.acquisition.iter().map(|r| &r.route).collect();
    let profile_routes: Vec<&Slug> = profile.acquisition.iter().map(|r| &r.id).collect();
    if manifest_routes != profile_routes {
        out.push(SchemaError::new(
            "E-BND-09",
            "acquisition",
            "the two documents were built from different route sets",
        ));
        return;
    }
    // ⚠ THE INSTALLED VERSION IS NOT COMPARED HERE, and leaving it out is
    // deliberate. The manifest already requires every route to have installed
    // its own recorded version, the profile requires the same of its build, and
    // E-BND-03 requires those two to agree. A fourth comparison could not fail
    // while the other three hold, and a guard that cannot fail is one nobody
    // knows works. It was written, found unreachable while planting a defect
    // for it, and removed.
    for (run, published) in manifest.acquisition.iter().zip(&profile.acquisition) {
        if run.artifact != published.artifact {
            out.push(SchemaError::new(
                "E-BND-11",
                format!("acquisition {}", run.route),
                "the artifact digest differs between the two documents",
            ));
        }
        // ⚠ This one CAN fail, which is why it is here and E-BND-10 is not.
        // Nothing else forces the two records of the signature into step: the
        // run writes what it did about the signature, the profile publishes
        // what it claims was done, and a record that says `verified` over a run
        // that says `not_checked` is the publishable half of a claim nobody
        // made. `ACQ-05` owns the authenticity evidence behind the status.
        if run.signature != published.signature {
            out.push(SchemaError::new(
                "E-BND-13",
                format!("acquisition {}", run.route),
                format!(
                    "the run recorded {} and the record publishes {}",
                    run.signature, published.signature
                ),
            ));
        }
    }
}

fn bind_sampling(manifest: &RunManifest, profile: &Profile, out: &mut Vec<SchemaError>) {
    let plan = &manifest.sampling;
    for field in &profile.observations {
        let at = format!("observations {}", field.path);
        if field.state.claims_variation() && !plan.varies_anything() {
            out.push(SchemaError::new(
                "E-BND-20",
                &at,
                format!(
                    "state {} says the value changes, and the run varied nothing: {} session(s), \
                     {} torrent(s), {} connection(s)",
                    field.state.as_str(),
                    plan.sessions,
                    plan.torrents,
                    plan.connections
                ),
            ));
        }
        if let Some(samples) = field.state.samples()
            && u64::from(samples.get()) > plan.observations()
        {
            out.push(SchemaError::new(
                "E-BND-21",
                &at,
                format!(
                    "rests on {samples} samples and the run could produce at most {}",
                    plan.observations()
                ),
            ));
        }
    }
}

/// Checks that a manifest and a profile describe the same run.
///
/// The two documents overlap by design: a profile has to stand alone for a
/// consumer, and a manifest has to stand alone for a replay. Every value that
/// appears in both is compared here, because the alternative is two documents
/// that disagree and a reader with no way to tell which one is wrong.
///
/// ⚠ This answers agreement, not validity. Each document has its own
/// invariants and its own reader that enforces them, and a caller that built
/// one in memory rather than reading it has not had those run. Reading both
/// through `from_json` is what makes the answer here mean what it says.
///
/// # Errors
///
/// Returns every disagreement, each with a stable code.
pub fn bind(manifest: &RunManifest, profile: &Profile) -> Result<(), Violations> {
    let mut out = Vec::new();
    bind_identity(manifest, profile, &mut out);
    bind_evidence(manifest, profile, &mut out);
    bind_connectors(manifest, profile, &mut out);
    bind_acquisition(manifest, profile, &mut out);
    bind_sampling(manifest, profile, &mut out);
    Violations::from_errors(out)
}
