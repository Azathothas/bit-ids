//! Assemble the artifacts a dispatched capture left into records in a store.
//!
//! ```text
//! cargo run -p bit-ids-probe --example assemble-capture -- \
//!     --lane CAPTURE_DIR,INSTALL_DIR [--lane CAPTURE_DIR,INSTALL_DIR]... STORE_DIR
//! ```
//!
//! `CI-09`. A dispatched `capture-client` run uploads two artifacts per lane:
//! the capture bundle, which carries the attestation, the generated metainfo and
//! one transcript per surface, and the install artifact, which carries the
//! install record, the release resolution and the bytes the build printed when
//! it was asked its version. This reads a lane of each and writes the
//! [`Profile`] they support.
//!
//! ⚠ **It does NOT write the [`RunManifest`](bit_ids::RunManifest) that has to
//! sit beside a record**, and that is a declared gap rather than an oversight. A
//! manifest carries the run's ordered phases, both clocks, the sampling plan and
//! the host facts; an attestation carries a start, a finish, a platform string
//! and a claim fingerprint, and inventing the rest would be a document `bind`
//! then compares against a record that agrees with it for no reason. `CI-09`
//! carries it.
//!
//! # ⛔ Every field is derived from an artifact, and a field with no artifact is
//! a refusal
//!
//! **Nothing here invents a value to get a record past a gate.** That matters
//! most for the two fields `E-ACQ-07` and `E-ACQ-08` compare: the resolver is
//! read out of the resolution document's own source identifier and the delivery
//! out of the route the adapter ran, so two lanes that really did resolve
//! through one listing produce two routes naming one resolver and are refused.
//! ⚠ Deriving a resolver from the *lane* instead would manufacture exactly the
//! independence those codes exist to check, and the record would look correct.
//!
//! # ⛔ One capture is one record, and both routes are in both records
//!
//! `E-ACQ-01` needs two routes and `Capture::observed_route` names which one's
//! install was put on the wire. So two lanes of one target and one version are
//! **two** records over the **same** route list, differing in their capture and
//! in which route each watched - which is the pair
//! [`classify_across`](bit_ids::classify_across) compares, and the only input
//! shape that can reach `BuildEquivalent` rather than `Unresolved`.
//!
//! # What it prints
//!
//! Every refusal, with its code, the document it came from and the artifact the
//! value was read out of. A run that cannot produce a record says exactly what
//! the capture path would have to record for it to.
//!
//! Exit codes: 0 the store was written, 1 a record was refused, 2 an artifact
//! could not be read.

use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::io::Write as _;
use std::path::{Path, PathBuf};
use std::process::ExitCode;

use bit_ids::Agreement;
use bit_ids::acquisition::{AcquisitionRoute, RouteKind, SignatureStatus, SourceIdentity};
use bit_ids::agreement::{ConnectorObservation, FieldCorroboration, Projection, SeenValue};
use bit_ids::canonical::{HexBytes, Instant, Label, RelPath, Sha256Digest, Slug, Url, Version};
use bit_ids::identity::{RecordId, RecordKey, SchemaVersion};
use bit_ids::observation::ObservedField;
use bit_ids::record::{
    Build, Capture, Connector, EvidenceKind, EvidenceRef, Profile, Target, TargetKind,
};
use bit_ids::sampling::SamplingPlan;
use bit_ids::store::StoreKey;
use bit_ids::{PROFILE_SCHEMA, ReleaseChannel, RunManifest};
use bit_ids_probe::normalize::Readings;

/// A key-value record of the shape every `bit-ids/*/1` text document uses.
///
/// ⚠ Read rather than parsed as JSON on purpose: `attestation.txt`,
/// `install-<route>.txt` and `resolution.txt` are all written by shell as
/// `key=value` lines, and a reader that accepted more shapes than the writers
/// emit would accept an artifact this project did not produce.
type Record = BTreeMap<String, String>;

fn read_record(path: &Path) -> Result<Record, String> {
    let text =
        std::fs::read_to_string(path).map_err(|error| format!("{}: {error}", path.display()))?;
    let mut out = Record::new();
    for line in text.lines() {
        // ⚠ The banner line carries no `=` and is the document's own schema
        // name. Skipping it silently would accept a document with no banner, so
        // the callers below check the banner they expect.
        if let Some((key, value)) = line.split_once('=') {
            out.insert(key.to_owned(), value.to_owned());
        }
    }
    if !text.starts_with("bit-ids/") {
        return Err(format!(
            "{}: does not begin with a bit-ids document banner",
            path.display()
        ));
    }
    Ok(out)
}

fn need<'a>(record: &'a Record, key: &str, source: &Path) -> Result<&'a str, String> {
    record
        .get(key)
        .map(String::as_str)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| format!("{}: carries no {key}", source.display()))
}

/// One dispatched lane: what it installed and what it then observed.
struct Lane {
    /// The route slug the adapter ran, as the install record spells it.
    route: String,
    /// The install record, from the capture bundle's own copy.
    install: Record,
    /// The release resolution, when the route had one.
    resolution: Option<Record>,
    /// The capture attestation.
    attestation: Record,
    /// Where the capture bundle's evidence sits.
    bundle: PathBuf,
    /// The bytes the build printed when asked its version.
    version_output: PathBuf,
    /// Where the resolution document was looked for, so a refusal can name it.
    resolution_path: PathBuf,
    /// The host claim the capture kept beside its attestation, when it did.
    host_claim: Option<Record>,
    /// What the observer printed while it ran.
    observer_log: PathBuf,
    /// What the install barrier concluded, when the lane waited at one.
    barrier: Option<Record>,
}

impl Lane {
    fn read(capture_dir: &Path, install_dir: &Path) -> Result<Self, String> {
        let attestation_path = capture_dir.join("capture/attestation.txt");
        let attestation = read_record(&attestation_path)?;
        // ⛔ The route is taken off the install record's own field rather than
        // off the artifact's directory name. A directory is named by whoever
        // unpacked it; the record is written by the step that ran the route.
        let mut found = None;
        for entry in std::fs::read_dir(capture_dir)
            .map_err(|error| format!("{}: {error}", capture_dir.display()))?
        {
            let path = entry.map_err(|error| error.to_string())?.path();
            let name = path
                .file_name()
                .unwrap_or_default()
                .to_string_lossy()
                .into_owned();
            // ⚠ The extension is asked for rather than matched off the end of
            // the name, so `install-release.TXT` is the same file to this reader
            // as it is to a case-insensitive filesystem.
            let is_text = path
                .extension()
                .is_some_and(|ext| ext.eq_ignore_ascii_case("txt"));
            if name.starts_with("install-") && is_text {
                found = Some(path);
            }
        }
        let Some(install_path) = found else {
            return Err(format!(
                "{}: carries no install-<route>.txt, so nothing says what acquired the build",
                capture_dir.display()
            ));
        };
        let install = read_record(&install_path)?;
        let route = need(&install, "route", &install_path)?.to_owned();

        // ⛔ THE RESOLUTION IS LOOKED FOR UNDER THE ROUTE THAT WROTE IT, and this
        // read `release/resolution.txt` for every lane until 2026-09-15. That was
        // complete for exactly as long as the source lane resolved through the
        // release listing - which is what `E-ACQ-07` refused run 14 for - and it
        // stopped being complete the moment that was repaired. ⚠ `resolve-source`
        // writes `source/resolution.txt`, a `bit-ids/source-resolution/1`
        // document whose `source_url` is the clone URL, deliberately so that it
        // slugifies into a different resolver from the releases listing.
        //
        // ⚠ The release name is still tried as a fallback, because a `package`
        // lane has no directory of its own and a release lane's document has not
        // moved.
        let resolution_path = {
            let by_route = install_dir.join(format!("{route}/resolution.txt"));
            if by_route.is_file() {
                by_route
            } else {
                install_dir.join("release/resolution.txt")
            }
        };
        let resolution = if resolution_path.is_file() {
            Some(read_record(&resolution_path)?)
        } else {
            None
        };

        let version_output = install_dir.join(format!("install-{route}/version.err"));
        if !version_output.is_file() {
            // ⛔ `E-ACQ-10` needs the bytes the build printed, and they are in
            // the INSTALL artifact rather than in the capture bundle. A lane
            // handed only its capture cannot produce a record at all.
            return Err(format!(
                "{}: absent. `E-ACQ-10` needs what the build printed when asked its version, and \
                 that is in the install artifact rather than in the capture bundle",
                version_output.display()
            ));
        }
        let optional = |path: PathBuf| -> Result<Option<Record>, String> {
            if path.is_file() {
                read_record(&path).map(Some)
            } else {
                Ok(None)
            }
        };
        Ok(Self {
            route,
            install,
            resolution,
            attestation,
            bundle: capture_dir.join("capture/bundle"),
            version_output,
            resolution_path,
            host_claim: optional(capture_dir.join("capture/host-claim.txt"))?,
            observer_log: capture_dir.join("capture/observer.log"),
            barrier: optional(install_dir.join("install-barrier.txt"))?,
        })
    }

    /// The slug identifying what decided this route's version.
    ///
    /// ⛔ **Read out of the resolution document's own source identifier**, so
    /// two lanes that consulted one listing name one resolver and `E-ACQ-07`
    /// refuses the pair. A resolver named after the lane would be this
    /// project's own tooling manufacturing the independence the rule checks.
    fn resolver(&self) -> Result<Slug, String> {
        match &self.resolution {
            Some(record) => {
                let url = need(record, "source_url", &self.resolution_path)?;
                slugify_source(url)
            }
            // ⚠ A route with no resolution document consulted the host's own
            // index, which is a different resolver from any listing and is
            // named as one rather than left blank.
            None => Ok(slug("host-package-index")),
        }
    }

    /// The slug identifying what delivered this route's bytes.
    ///
    /// ⛔ **A package route that did not let its package manager fetch says so.**
    /// `deluge`'s Windows route takes its version and digest from winget's index
    /// and fetches the vendor's installer over https itself, because the index
    /// pins an `http` URL no record may name. Its `package-index.txt` carries
    /// `delivery=`, `install-client` copies it into `package_delivery`, and it
    /// is read here. `install-client` refuses one that is not a slug, and a
    /// record with none is what every other package route is. `CLIENT-07`.
    fn delivery(&self) -> Slug {
        match self.route.as_str() {
            "release" => slug("https-release-asset"),
            "source" => slug("git-clone-and-local-build"),
            _ => self
                .install
                .get("package_delivery")
                .and_then(|value| Slug::parse(value).ok())
                .unwrap_or_else(|| slug("host-package-manager")),
        }
    }

    fn kind(&self) -> RouteKind {
        match self.route.as_str() {
            "release" => RouteKind::GithubRelease,
            "source" => RouteKind::SourceBuild,
            // ⚠ WHICH HOST'S INDEX IS THE INSTALL RECORD'S PLATFORM, the same
            // line the identity tuple is derived from. ⭐ winget on Windows,
            // measured on capture-client run 38788416994. `CI-11`.
            _ if self
                .install
                .get("platform")
                .is_some_and(|p| p.split_whitespace().next() == Some("Windows")) =>
            {
                RouteKind::WindowsPackageManager
            }
            _ => RouteKind::LinuxPackageManager,
        }
    }

    fn route_id(&self) -> Slug {
        slug(&format!("route-{}", self.route))
    }

    /// Every connector the attestation says observed the run.
    ///
    /// ⛔ **The observer is one of them, not a third thing beside them.** The
    /// attestation names it in `observer`; a second connector appears in
    /// `connectors`, comma-separated. ⭐ `capture-client` run 15 on 2026-09-15 is
    /// the first capture to write that key - `connectors=cpython-stdlib` - so a
    /// lane of it declares TWO and `E-CAP-01` is satisfied. ⚠ Run 14 carried no
    /// such key, which is the whole of why it was refused as declaring one.
    ///
    /// ⛔ **Each version is the attestation's, never this file's.** Both were
    /// `0.0.0` until 2026-10-14, written here for every connector; for the
    /// observer that happened to be its crate's version and for the second
    /// connector it was nobody's. `observer_version` is what the observer
    /// printed and `connector_version` what the connector described, and a lane
    /// without one is refused. ⚠ One `connector_version` names one connector, so
    /// an attestation listing two beside the observer is refused rather than
    /// given one version for both.
    fn connectors(&self) -> Result<Vec<Connector>, String> {
        let version_of = |key: &str| -> Result<Version, String> {
            let text = self
                .attestation
                .get(key)
                .filter(|text| !text.is_empty())
                .ok_or_else(|| format!("the attestation records no `{key}`"))?;
            Version::parse(text).map_err(|error| format!("{key} {text:?}: {error}"))
        };
        let mut ids = vec![slug("bit-ids-probe")];
        for extra in self
            .attestation
            .get("connectors")
            .map(String::as_str)
            .unwrap_or_default()
            .split(',')
            .filter(|text| !text.is_empty())
        {
            let id = Slug::parse(extra.trim())
                .map_err(|error| format!("a connector named {extra:?}: {error}"))?;
            if !ids.contains(&id) {
                ids.push(id);
            }
        }
        if ids.len() > 2 {
            return Err(format!(
                "the attestation names {} connectors beside the observer and one connector_version",
                ids.len() - 1
            ));
        }
        ids.sort_by_key(std::string::ToString::to_string);
        let observer = slug("bit-ids-probe");
        ids.into_iter()
            .map(|id| {
                let key = if id == observer {
                    "observer_version"
                } else {
                    "connector_version"
                };
                Ok(Connector {
                    id,
                    version: version_of(key)?,
                })
            })
            .collect()
    }

    fn evidence_id(&self, suffix: &str) -> Slug {
        slug(&format!("ev-{}-{suffix}", self.route))
    }

    /// Whether this route actually acquired the build, off the install record.
    ///
    /// ⛔ **`ACQ-03`'s sharpest residual, and this is the layer its own entry
    /// names.** A route that installs nothing exits 0, so two routes on a host
    /// that already ships the product declare two independent resolvers, satisfy
    /// `E-ACQ-07` and `E-ACQ-08`, agree on the version *because it is one
    /// binary*, and reach `classify` as `ByteIdentical` - the strongest verdict
    /// that function has, arrived at by acquiring nothing.
    ///
    /// ⚠ **Measured, in runs this repository already dispatched**: `aria2` ships
    /// on `ubuntu-24.04`, so client capture runs 3 and 4 recorded
    /// `route=package` over an `apt-get install` that installed nothing.
    ///
    /// ⛔ **The refusal belongs here rather than in `classify`.** That function
    /// compares route records and the field is not on the record type; adding it
    /// there is a schema change across every fixture, every rendering and every
    /// validator. What is wrong is upstream of the comparison: a route that
    /// acquired nothing is not a second route, so the record is never written.
    /// ⚠ So this closes the hole and does not close the residual - `classify`
    /// still cannot see the field, and a record hand-written past this assembler
    /// would still reach it.
    fn acquired(&self) -> Result<(), String> {
        let install = Path::new("install-<route>.txt");
        match need(&self.install, "acquired", install)? {
            "yes" => Ok(()),
            "no" => Err(format!(
                "the {} route records `acquired=no`: it ran, exited 0 and installed nothing, \
                 because the host already carried the build. Two such routes agree on a version \
                 because it is ONE binary, and a record over them would call that agreement \
                 evidence. `ACQ-03`",
                self.route
            )),
            other => Err(format!(
                "the {} route records `acquired={other}`, which is neither yes nor no",
                self.route
            )),
        }
    }
}

fn slug(text: &str) -> Slug {
    Slug::parse(text).expect("a derived identifier is canonical by construction")
}

/// A resolver identifier derived from the endpoint a resolution actually read.
///
/// ⛔ **The query string is dropped and everything else is kept.** A query
/// carries paging and per-request parameters, which are properties of one call
/// rather than of the index; the host and the path are what say *which* index
/// answered. Two routes that read one endpoint therefore slugify to one name and
/// `E-ACQ-07` refuses them, which is the whole point of deriving this rather
/// than naming it.
///
/// ⚠ Capped at [`Slug::MAX_LEN`] from the LEFT, so a long path cannot make two
/// different endpoints collide on a shared prefix - the host is the part most
/// likely to be shared, so the tail is what is kept when something must go.
fn slugify_source(url: &str) -> Result<Slug, String> {
    let without_query = url.split('?').next().unwrap_or(url);
    let mut out = String::new();
    for byte in without_query.to_ascii_lowercase().bytes() {
        let ch = char::from(byte);
        if ch.is_ascii_lowercase() || ch.is_ascii_digit() {
            out.push(ch);
        } else if !out.ends_with('-') {
            out.push('-');
        }
    }
    let trimmed = out.trim_matches('-');
    let capped = if trimmed.len() > Slug::MAX_LEN {
        trimmed[trimmed.len() - Slug::MAX_LEN..].trim_matches('-')
    } else {
        trimmed
    };
    Slug::parse(capped).map_err(|error| format!("a resolver derived from {url:?}: {error}"))
}

/// The immutable identity of what a route asked for.
///
/// ⛔ **A source route needs a full commit and no record the capture path
/// writes carries one.** `E-ACQ-06` refuses an abbreviation, the install record
/// has no field for it, and the object name survives only inside the git
/// chatter in `install.log`. That is a gap in what the capture records, and the
/// refusal here is what names it.
fn source_identity(lane: &Lane) -> Result<SourceIdentity, String> {
    let resolution = lane.resolution.as_ref();
    match lane.kind() {
        RouteKind::GithubRelease => {
            let record = resolution.ok_or_else(|| {
                format!(
                    "the release route has no resolution document at {}, so nothing names the \
                     asset it took",
                    lane.resolution_path.display()
                )
            })?;
            let path = lane.resolution_path.as_path();
            Ok(SourceIdentity::ReleaseAsset {
                repository: label(need(record, "repository", path)?)?,
                tag: label(need(record, "selected_tag", path)?)?,
                asset: label(need(record, "asset", path)?)?,
            })
        }
        RouteKind::SourceBuild => {
            let record = resolution.ok_or_else(|| {
                format!(
                    "the source route has no resolution document at {}. \
                         `capture-client` uploads it from the resolver's own workdir",
                    lane.resolution_path.display()
                )
            })?;
            let path = lane.resolution_path.as_path();
            let commit = lane
                .install
                .get("source_commit")
                .map(String::as_str)
                .unwrap_or_default();
            if commit.is_empty() {
                return Err(
                    "the source route records no `source_commit`, and `E-ACQ-06` refuses a source \
                     identity without a full object name. `install-client` writes the install \
                     record and does not carry one; the commit exists only in the git output \
                     inside install.log"
                        .to_owned(),
                );
            }
            Ok(SourceIdentity::SourceCommit {
                repository: label(need(record, "repository", path)?)?,
                commit: HexBytes::parse(commit)
                    .map_err(|error| format!("source_commit: {error}"))?,
            })
        }
        // ⭐ A PACKAGE ROUTE IS IDENTIFIED BY WHAT ITS INDEX NAMED: the index,
        // the package and the index's own version, which `install-client`
        // records from the adapter's `package-index.txt`. `CI-11`.
        RouteKind::WindowsPackageManager | RouteKind::LinuxPackageManager => {
            let field = |key: &str| {
                lane.install
                    .get(key)
                    .map(String::as_str)
                    .filter(|value| !value.is_empty())
                    .ok_or_else(|| {
                        format!(
                            "the package route records no `{key}`; its adapter writes \
                             package-index.txt and `install-client` carries it into the record"
                        )
                    })
            };
            let version = field("package_version")?;
            Ok(SourceIdentity::IndexedPackage {
                index: label(field("package_index")?)?,
                package: label(field("package_name")?)?,
                version: Version::parse(version)
                    .map_err(|error| format!("package_version {version:?}: {error}"))?,
            })
        }
        _ => Err(format!(
            "route {} has no source identity this example derives",
            lane.route
        )),
    }
}

fn label(text: &str) -> Result<Label, String> {
    Label::parse(text).map_err(|error| format!("{text:?}: {error}"))
}

fn route_of(lane: &Lane, version: &Version) -> Result<AcquisitionRoute, String> {
    let install = Path::new("install-<route>.txt");
    // ⚠ THE ARTIFACT DIGEST IS THE INSTALLED EXECUTABLE'S, WHICHEVER ROUTE RAN.
    // The resolution records the asset URL and the size the source *declared*,
    // and nothing the capture uploads carries a digest of the bytes that
    // arrived. That is a narrower claim than the field's name suggests, it is
    // what the evidence supports, and `ACQ-05` is where a retrieval digest would
    // come from.
    let artifact = need(&lane.install, "installed_binary_sha256", install)?;
    // ⛔ THE ORIGIN FIELD DIFFERS BY ROUTE KIND, because the two documents record
    // different things. A release resolution names the asset it selected in
    // `asset_url`; a source resolution has no asset at all and names the
    // repository it will clone in `source_url`. ⚠ Asking every route for
    // `asset_url` refused a source lane for a field its document is not supposed
    // to carry, which reads as a capture-path gap and is a reader's assumption.
    let origin = match &lane.resolution {
        Some(record) => {
            let key = match lane.kind() {
                RouteKind::SourceBuild => "source_url",
                _ => "asset_url",
            };
            need(record, key, &lane.resolution_path)?.to_owned()
        }
        // ⛔ THIS MESSAGE USED TO SAY *a package route has no origin URL*, and
        // it named the wrong route kind on the only lanes that reach it. A
        // package route has no resolution because it consulted the host's index;
        // a release or source lane reaches this branch when its resolution
        // document was not UPLOADED, which is a different fact and a different
        // fix. ⚠ Measured on 2026-09-15: run 16's release lane was told it was a
        // package route, and the sentence sent a reader to the resolver rather
        // than to the artifact.
        // ⭐ A package route's origin is the installer URL its index pinned.
        None if lane.route == "package" => lane
            .install
            .get("package_origin")
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                "the package route records no `package_origin`, the installer URL its index \
                 pinned"
                    .to_owned()
            })?
            .to_owned(),
        None => {
            return Err(format!(
                "route {} has no resolution document at {}, so nothing names the origin. A \
                 `package` route legitimately has none; a `release` or `source` lane reaching \
                 this has one that the run did not upload",
                lane.route,
                lane.resolution_path.display()
            ));
        }
    };
    Ok(AcquisitionRoute {
        id: lane.route_id(),
        kind: lane.kind(),
        resolver: lane.resolver()?,
        delivery: lane.delivery(),
        origin: Url::parse(&origin).map_err(|error| format!("origin: {error}"))?,
        source: source_identity(lane)?,
        resolved_version: version.clone(),
        artifact: Sha256Digest::parse(&format!("sha256:{artifact}"))
            .map_err(|error| format!("artifact: {error}"))?,
        signature: SignatureStatus::NotChecked,
        installed_probe: Label::parse("aria2-next --version").map_err(|error| error.to_string())?,
        installed_evidence: lane.evidence_id("installed-version"),
        installed_executable: Sha256Digest::parse(&format!(
            "sha256:{}",
            need(&lane.install, "installed_binary_sha256", install)?
        ))
        .map_err(|error| format!("installed_executable: {error}"))?,
        installed_version: version.clone(),
    })
}

/// The evidence one lane contributes, with every size and digest read off disk.
///
/// ⛔ The digests are computed here rather than copied out of the bundle's
/// `SHA256SUMS`, because a record that repeated a document's own claim about
/// its bytes would agree with it whatever the bytes are.
fn evidence_of(lane: &Lane) -> Result<Vec<(EvidenceRef, PathBuf)>, String> {
    let mut out = Vec::new();
    let mut add = |id: Slug, kind: EvidenceKind, rel: &str, path: PathBuf| -> Result<(), String> {
        let bytes = std::fs::read(&path).map_err(|error| format!("{}: {error}", path.display()))?;
        if bytes.is_empty() {
            return Err(format!("{}: has no bytes", path.display()));
        }
        out.push((
            EvidenceRef {
                id,
                kind,
                path: RelPath::parse(rel).map_err(|error| format!("{rel}: {error}"))?,
                bytes: bytes.len() as u64,
                sha256: Sha256Digest::of(&bytes),
                connector: match kind {
                    EvidenceKind::Metainfo | EvidenceKind::ProcessOutput => None,
                    _ => Some(slug("bit-ids-probe")),
                },
            },
            path,
        ));
        Ok(())
    };
    add(
        lane.evidence_id("metainfo"),
        EvidenceKind::Metainfo,
        &format!("{}/fixture/generated.torrent", lane.route),
        lane.bundle.join("fixture/generated.torrent"),
    )?;
    add(
        lane.evidence_id("tracker"),
        EvidenceKind::TrackerCapture,
        &format!("{}/tracker-http.transcript.json", lane.route),
        lane.bundle.join("tracker-http.transcript.json"),
    )?;
    add(
        lane.evidence_id("peer"),
        EvidenceKind::PeerTranscript,
        &format!("{}/peer-wire-dialled.transcript.json", lane.route),
        lane.bundle.join("peer-wire-dialled.transcript.json"),
    )?;
    add(
        lane.evidence_id("installed-version"),
        EvidenceKind::ProcessOutput,
        &format!("{}/version.txt", lane.route),
        lane.version_output.clone(),
    )?;
    out.sort_by_key(|entry| entry.0.id.to_string());
    Ok(out)
}

/// What the transcripts say this build put on the wire.
///
/// ⛔ **The normalization itself is [`bit_ids_probe::normalize`]**, a library
/// body rather than this file's own since `OBS-10`, so the test that holds a
/// Linux and a Windows compilation to one digest runs the code a record is
/// written with. This reads the two documents off the bundle and names each
/// field's evidence after the lane.
fn observations_of(lane: &Lane) -> Result<(Vec<ObservedField>, Readings, SamplingPlan), String> {
    let tracker = std::fs::read_to_string(lane.bundle.join("tracker-http.transcript.json"))
        .map_err(|error| format!("the tracker transcript: {error}"))?;
    let peer = std::fs::read_to_string(lane.bundle.join("peer-wire-dialled.transcript.json"))
        .map_err(|error| format!("the peer transcript: {error}"))?;
    let normalized =
        bit_ids_probe::normalize::normalize(&tracker, &peer, &|suffix| lane.evidence_id(suffix))?;
    let readings = normalized.readings();
    Ok((normalized.fields, readings, normalized.plan))
}

/// What each declared connector saw, per field.
///
/// ⛔ **THE SECOND CONNECTOR'S VALUES COME OUT OF ITS OWN ARTIFACT, and this is
/// the contract `OBS-07` has to satisfy.** A connector declared in the
/// attestation writes `connector/<id>.txt` into the bundle, one
/// `field_path=value` line per field the observer measured, where the value is
/// `out_of_scope` or every sample of the field in document order,
/// comma-separated, each lowercase hex or `absent`. A declared connector that
/// reports nothing about a field is refused rather than recorded as silent:
/// `E-COR-07` already says silence is not the same as `not_corroborated`, and a
/// connector this example filled in for would be corroboration the run never
/// produced.
///
/// ⛔ **EVERY SAMPLE IS COMPARED, NOT THE FIRST.** This compared one value per
/// field until 2026-10-14, so a field resting on four announces was corroborated
/// on one of them and the record's `exact` vouched for three readings nobody
/// had made. ⭐ The lists are compared position by position, and a record
/// carries the first position where the readers part - or the first sample, when
/// they never do - because a corroboration entry holds one value per connector
/// and `E-COR-14` refuses a `disagrees` whose values match.
/// ⚠ **A count that differs is a disagreement about which segments the build
/// sent**, and the reader with no sample at that position is recorded as having
/// seen nothing there: [`SeenValue::Absent`], which is what it looked for and did
/// not find.
///
/// ⭐ **A capture HAS written one now, measured on 2026-09-15.**
/// `capture-client` run 15's release lane carries
/// `bundle/connector/cpython-stdlib.txt` with four lines, and its attestation
/// declares `connectors=cpython-stdlib`, so this is reached rather than cut off
/// by `E-CAP-01`. ⚠ That sentence used to read *no capture has written one*; run
/// 14 was the last attestation with no `connectors` key at all, which is why it
/// was refused as declaring a single connector.
///
/// ⛔ **AND THE REPORT DISAGREES WITH ITSELF ACROSS SURFACES, CORRECTLY.** Run
/// 15's connector read two different peer IDs out of one capture - the announce
/// and the handshake carry different twelve-byte tails after the same
/// `-qB5230-` prefix - which is the per-connection tail `docs/history/RESUME.md`
/// records, now measured by a reader this project did not write.
fn corroboration_of(
    lane: &Lane,
    fields: &[ObservedField],
    readings: &Readings,
    connectors: &[Connector],
) -> Result<Vec<FieldCorroboration>, String> {
    let mut reports: BTreeMap<String, Record> = BTreeMap::new();
    for connector in connectors {
        if connector.id.as_str() == "bit-ids-probe" {
            continue;
        }
        let path = lane.bundle.join(format!("connector/{}.txt", connector.id));
        let text = std::fs::read_to_string(&path).map_err(|error| {
            format!(
                "{}: {error}. A declared connector reports what it saw, one field per line",
                path.display()
            )
        })?;
        let mut record = Record::new();
        for line in text.lines() {
            if let Some((key, value)) = line.split_once('=') {
                record.insert(key.to_owned(), value.to_owned());
            }
        }
        reports.insert(connector.id.to_string(), record);
    }

    let mut out = Vec::new();
    for field in fields {
        let key = field.path.to_string();
        let mut read = Vec::new();
        for connector in connectors {
            let samples = if connector.id.as_str() == "bit-ids-probe" {
                // ⛔ THE OBSERVATIONS, NOT THE STATE. A sampled field has no
                // single value, and mapping everything but `constant` to
                // `out_of_scope` left every patterned field uncorroborated -
                // which `E-PUB-02` refuses, so sampling a field would have
                // blocked publication through a second door the moment it
                // stopped blocking it through the first.
                let values = readings
                    .get(&key)
                    .ok_or_else(|| format!("{key}: the observer kept no observation of it"))?;
                Some(values.iter().cloned().map(SeenValue::Bytes).collect())
            } else {
                let report = &reports[connector.id.as_str()];
                let Some(value) = report.get(&key) else {
                    return Err(format!(
                        "connector {} reports nothing about {key}, and silence is not the same \
                         as not_corroborated",
                        connector.id
                    ));
                };
                reported_samples(value)
                    .map_err(|error| format!("connector {} on {key}: {error}", connector.id))?
            };
            read.push((connector.id.clone(), samples));
        }
        let (seen, agreement, conflict) = compare_samples(&key, &read)?;
        let mut observations = Vec::new();
        for (connector, value) in read.iter().map(|(id, _)| id).zip(seen) {
            observations.push(ConnectorObservation {
                connector: connector.clone(),
                evidence: field
                    .evidence
                    .first()
                    .cloned()
                    .ok_or_else(|| format!("{} cites no evidence", field.path))?,
                projection: Projection::Raw,
                seen: value,
            });
        }
        observations.sort_by_key(|entry| entry.connector.to_string());
        out.push(FieldCorroboration {
            path: field.path.clone(),
            observations,
            agreement,
            conflict,
        });
    }
    out.sort_by_key(|entry| entry.path.to_string());
    Ok(out)
}

/// One connector's report line for one field, as samples in document order.
///
/// `None` is `out_of_scope`: the connector could not see the surface at all,
/// which is not a list of anything. ⛔ An empty element is refused rather than
/// skipped, because `a,,b` is a report that lost a sample somewhere and a reader
/// that closed the gap would line the rest up against the wrong segments.
fn reported_samples(value: &str) -> Result<Option<Vec<SeenValue>>, String> {
    if value == "out_of_scope" {
        return Ok(None);
    }
    let mut out = Vec::new();
    for (index, sample) in value.split(',').enumerate() {
        out.push(match sample {
            "absent" => SeenValue::Absent,
            "" => return Err(format!("sample {} is empty", index + 1)),
            hex => SeenValue::Bytes(
                HexBytes::parse(hex).map_err(|error| format!("sample {}: {error}", index + 1))?,
            ),
        });
    }
    Ok(Some(out))
}

/// One value per connector, and the outcome, from every connector's samples.
///
/// ⛔ **The position recorded is the first at which the in-scope readers part,
/// or the first sample when they never do.** A corroboration entry holds one
/// value per connector, so the record shows the pair that decided the verdict:
/// `exact` over the first sample means every position matched, and `disagrees`
/// carries the position that did not. ⚠ A reader with no sample at a position
/// that another reader has is recorded as [`SeenValue::Absent`] there.
///
/// ⛔ **A difference the record cannot show is refused rather than written as
/// agreement.** A missing sample beside an `absent` one is a different count and
/// the same value, so `disagrees` over two matching values is what `E-COR-14`
/// refuses and `exact` would be false. No real capture produces it - the
/// observer refuses an announce with no `User-Agent` - so `check-assemble`
/// plants it, and the day a capture does, it reaches a reader rather than a
/// record.
fn compare_samples(
    key: &str,
    read: &[(Slug, Option<Vec<SeenValue>>)],
) -> Result<(Vec<SeenValue>, Agreement, Option<Label>), String> {
    let in_scope: Vec<&Vec<SeenValue>> = read.iter().filter_map(|(_, s)| s.as_ref()).collect();
    let longest = in_scope
        .iter()
        .map(|samples| samples.len())
        .max()
        .unwrap_or(0);
    let parted = (0..longest).find(|&at| {
        let mut values = in_scope.iter().map(|samples| samples.get(at));
        let first = values.next().flatten();
        values.any(|value| value != first)
    });
    let at = parted.unwrap_or(0);
    let seen: Vec<SeenValue> = read
        .iter()
        .map(|(_, samples)| match samples {
            None => SeenValue::OutOfScope,
            Some(samples) => samples.get(at).cloned().unwrap_or(SeenValue::Absent),
        })
        .collect();
    if in_scope.len() < 2 {
        return Ok((seen, Agreement::NotCorroborated, None));
    }
    let Some(at) = parted else {
        return Ok((seen, Agreement::Exact, None));
    };
    let shown: Vec<&SeenValue> = seen.iter().filter(|value| value.in_scope()).collect();
    if shown.iter().all(|value| *value == shown[0]) {
        return Err(format!(
            "{key}: the readers part at sample {} only in how many samples each found, and a \
             record holding one value per connector cannot show that",
            at + 1
        ));
    }
    // ⚠ In the order the observations are listed, which is by connector, so a
    // count is attributable without a name: two slugs can fill a label alone.
    let counts: Vec<String> = in_scope
        .iter()
        .map(|samples| samples.len().to_string())
        .collect();
    let text = if counts.iter().all(|count| *count == counts[0]) {
        format!(
            "connectors read different bytes at sample {} of {}",
            at + 1,
            counts[0]
        )
    } else {
        format!(
            "connectors read {} samples, and part at sample {}",
            counts.join(" and "),
            at + 1
        )
    };
    let conflict = Label::parse(&text).map_err(|error| format!("{key}: {error}"))?;
    Ok((seen, Agreement::Disagrees, Some(conflict)))
}

/// The host family, the architecture and the package format the record is filed
/// under.
///
/// ⛔ **THESE THREE ARE IN [`StoreKey`], SO HARDCODING THEM IS THE
/// NON-INJECTIVE-PATH DEFECT `store.rs` EXISTS TO REFUSE.** A path built from
/// fewer components than the identity tuple files two measurements at one name
/// and one of them silently wins. ⚠ The first draft of this example wrote
/// `linux`, `x86-64` and `elf-binary` as literals, which would have filed a
/// Windows capture of the same version at the Linux capture's path.
///
/// ⭐ The first two are derived from the attestation's own `platform` line,
/// which is `uname -srm`: a system name, a kernel release and a machine.
/// ⭐ The third comes from the INSTALL record, because the install is what
/// delivered the build and the adapter is what knows the form. ⚠ Nothing wrote
/// it until 2026-09-09 - not the attestation, not the install record, not the
/// release resolution, and not `catalogue/clients.toml`, which carries
/// `candidate_routes` and no package format - so a lane from an older run is
/// refused here rather than filed under a guess.
fn identity_of(lane: &Lane) -> Result<(Slug, Slug, Slug), String> {
    let attestation = Path::new("capture/attestation.txt");
    let platform_line = need(&lane.attestation, "platform", attestation)?;
    let mut parts = platform_line.split_whitespace();
    let (Some(system), Some(_release), Some(machine)) = (parts.next(), parts.next(), parts.next())
    else {
        return Err(format!(
            "the attestation's platform is {platform_line:?}, which is not `uname -srm` output"
        ));
    };
    let platform = Slug::parse(&system.to_ascii_lowercase())
        .map_err(|error| format!("a platform derived from {system:?}: {error}"))?;
    // ⚠ `uname -m` spells it `x86_64` and this project's vocabulary spells it
    // `x86-64`, so the underscore is mapped rather than the value passed
    // through. A slug cannot hold an underscore at all, so an unmapped machine
    // name is a refusal naming the value rather than a silently wrong path.
    let arch = Slug::parse(&machine.to_ascii_lowercase().replace('_', "-"))
        .map_err(|error| format!("an architecture derived from {machine:?}: {error}"))?;
    let package = match lane.install.get("package").map(String::as_str) {
        Some(text) if !text.is_empty() => {
            Slug::parse(text).map_err(|error| format!("package: {error}"))?
        }
        _ => {
            return Err(
                "the install record carries no `package`, and it is part of the identity tuple a \
                 store path is derived from. The adapter's `install` writes it beside its log \
                 because the adapter is what knows the form; hardcoding one here would file two \
                 packagings of one version at one path"
                    .to_owned(),
            );
        }
    };
    Ok((platform, arch, package))
}

fn profile_of(
    lanes: &[Lane],
    watched: usize,
    version: &Version,
) -> Result<(Profile, SamplingPlan), String> {
    let lane = &lanes[watched];
    let attestation = Path::new("capture/attestation.txt");
    let (platform, arch, package) = identity_of(lane)?;
    let mut acquisition = Vec::new();
    for other in lanes {
        acquisition.push(route_of(other, version)?);
    }
    acquisition.sort_by_key(|entry| entry.id.to_string());

    let evidence: Vec<EvidenceRef> = lanes
        .iter()
        .map(evidence_of)
        .collect::<Result<Vec<_>, _>>()?
        .into_iter()
        .flatten()
        .map(|(entry, _)| entry)
        .collect();

    let capture = Capture {
        id: slug(&format!("cap-{}", lane.route)),
        captured_at: Instant::parse(need(&lane.attestation, "started_at", attestation)?)
            .map_err(|error| format!("started_at: {error}"))?,
        fixture: Sha256Digest::parse(need(&lane.attestation, "fixture", attestation)?)
            .map_err(|error| format!("fixture: {error}"))?,
        observed_route: lane.route_id(),
        observer: slug("bit-ids-probe"),
        // ⛔ READ OFF THE ATTESTATION, NEVER PADDED. Every capture this project
        // has run declared its own Rust observer and nothing else, and
        // `E-CAP-01` refuses a record with one connector at the VALIDITY gate -
        // not at publication. Listing a second here to get a record written
        // would be this example inventing the corroboration the whole project
        // exists to measure.
        connectors: lane.connectors()?,
    };

    let mut profile = Profile {
        schema: SchemaVersion::parse(PROFILE_SCHEMA).map_err(|error| error.to_string())?,
        id: RecordId::derive(&RecordKey {
            schema: &SchemaVersion::parse(PROFILE_SCHEMA).map_err(|error| error.to_string())?,
            target: &slug(need(&lane.attestation, "target", attestation)?),
            version,
            platform: &platform,
            arch: &arch,
            package: &package,
            capture: &capture.id,
        }),
        target: Target {
            id: slug(need(&lane.attestation, "target", attestation)?),
            // ⚠ THE IDENTIFIER, BECAUSE NOTHING A CAPTURE UPLOADS CARRIES A
            // DISPLAY NAME. `catalogue/clients.toml` does, and reading it would
            // mean a TOML dependency for one cosmetic field, which
            // `docs/supply-chain.md` would refuse. It is not part of the
            // identity tuple, so unlike the three above it cannot file a record
            // at the wrong path; whatever publishes a record is where a display
            // name should be joined on.
            display_name: need(&lane.attestation, "target", attestation)?.to_owned(),
            kind: match need(&lane.attestation, "kind", attestation)? {
                "client" => TargetKind::Client,
                "library" => TargetKind::Library,
                // ⛔ `kind=fixture` is what `capture.yml` writes, and a fixture
                // is not a measurement of anything. Refusing it here is what
                // stops a fixture bundle becoming a record that reads like one.
                other => {
                    return Err(format!(
                        "the attestation says kind={other}, which is not a target kind. A \
                         `fixture` capture measured no build"
                    ));
                }
            },
            // ⚠ `libtorrent/2.1.1` is in what the build printed, and nothing has
            // measured the relationship. `null` is the honest value until
            // something does.
            engine: None,
        },
        build: Build {
            version: version.clone(),
            channel: ReleaseChannel::Stable,
            platform,
            arch,
            package,
            executable: Sha256Digest::parse(&format!(
                "sha256:{}",
                need(
                    &lane.install,
                    "installed_binary_sha256",
                    Path::new("install-<route>.txt")
                )?
            ))
            .map_err(|error| format!("executable: {error}"))?,
        },
        acquisition,
        capture,
        observations: Vec::new(),
        corroboration: Vec::new(),
        normalizations: Vec::new(),
        evidence,
        supersedes: None,
        adjudication: None,
    };
    let (observations, readings, plan) = observations_of(lane)?;
    profile.observations = observations;
    profile.corroboration = corroboration_of(
        lane,
        &profile.observations,
        &readings,
        &profile.capture.connectors,
    )?;
    profile.evidence.sort_by_key(|entry| entry.id.to_string());
    Ok((profile, plan))
}

// -- the run manifest ---------------------------------------------------------

/// Seconds since the epoch for a canonical instant, and back.
///
/// ⚠ [`Instant`] orders as text and carries no arithmetic; the one sum a manifest
/// needs is the observer's serving window added to the capture's start.
fn epoch_of(instant: &Instant) -> Result<i64, String> {
    let text = instant.to_string();
    let field = |range: core::ops::Range<usize>| -> Result<i64, String> {
        text.get(range)
            .and_then(|part| part.parse::<i64>().ok())
            .ok_or_else(|| format!("{text}: not a canonical instant"))
    };
    let (year, month, day) = (field(0..4)?, field(5..7)?, field(8..10)?);
    let (hour, minute, second) = (field(11..13)?, field(14..16)?, field(17..19)?);
    // Days from the civil date, Howard Hinnant's algorithm.
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y - era * 400;
    let mp = (month + 9) % 12;
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    let days = era * 146_097 + doe - 719_468;
    Ok(days * 86_400 + hour * 3_600 + minute * 60 + second)
}

fn instant_at(seconds: i64) -> Result<Instant, String> {
    let days = seconds.div_euclid(86_400);
    let rest = seconds.rem_euclid(86_400);
    let z = days + 719_468;
    let era = z.div_euclid(146_097);
    let doe = z - era * 146_097;
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = doy - (153 * mp + 2) / 5 + 1;
    let month = if mp < 10 { mp + 3 } else { mp - 9 };
    let year = yoe + era * 400 + i64::from(month <= 2);
    Instant::parse(&format!(
        "{year:04}-{month:02}-{day:02}T{:02}:{:02}:{:02}Z",
        rest / 3_600,
        rest % 3_600 / 60,
        rest % 60
    ))
    .map_err(|error| error.to_string())
}

fn instant_in(record: &Record, key: &str, source: &str) -> Result<Instant, String> {
    let text = record
        .get(key)
        .filter(|text| !text.is_empty())
        .ok_or_else(|| format!("{source} records no `{key}`"))?;
    Instant::parse(text).map_err(|error| format!("{source} {key}: {error}"))
}

/// Seconds, as `/proc/uptime` prints them, in nanoseconds.
fn uptime_ns(text: &str, what: &str) -> Result<u64, String> {
    let (whole, fraction) = text.split_once('.').unwrap_or((text, ""));
    let seconds: u64 = whole
        .parse()
        .map_err(|_| format!("{what} {text:?} is not an uptime"))?;
    let mut digits = fraction.to_owned();
    digits.truncate(9);
    while digits.len() < 9 {
        digits.push('0');
    }
    let nanos: u64 = digits
        .parse()
        .map_err(|_| format!("{what} {text:?} is not an uptime"))?;
    seconds
        .checked_mul(1_000_000_000)
        .and_then(|ns| ns.checked_add(nanos))
        .ok_or_else(|| format!("{what} {text:?} overflows"))
}

/// The run manifest one record needs beside it, from the run's own artifacts.
///
/// ⛔ **Every field is read off an artifact, as the record's are.** `CI-09`
/// carries the survey that found each one and the producer that now writes it.
/// ⭐ Settled by the operator on 2026-10-14: the manifest describes the WATCHED
/// host, and the other route contributes its acquisition and its evidence from
/// its own artifacts.
///
/// ⛔ **The phases are milestones, and the barrier is what makes them true.**
/// `resolved` ends when the last route resolved and `acquired_twice` when the
/// last route installed, across both runners; `observed` starts at this
/// capture. A capture that began before every route had installed overlaps,
/// and `validate_manifest` refuses it rather than this function bending it.
///
/// ⚠ **`validated` is not a phase of the run.** It happens here, after the run
/// ended, and every phase lies inside the run's wall clock; a manifest stops at
/// `corroborated`.
fn manifest_of(
    lanes: &[Lane],
    watched: usize,
    profile: &Profile,
    plan: SamplingPlan,
) -> Result<RunManifest, String> {
    let lane = &lanes[watched];
    let claim = lane.host_claim.as_ref().ok_or(
        "the capture kept no host-claim.txt, so nothing says when or where the host was claimed",
    )?;
    let (host, isolation) = host_of(lane, claim, profile)?;
    let clocks = clocks_of(lane, claim, &isolation)?;
    let phases = phases_of(lanes, watched, profile, &clocks)?;
    Ok(RunManifest {
        schema: bit_ids::manifest::ManifestSchema::current(),
        capture: profile.capture.id.clone(),
        target: profile.target.id.clone(),
        version: profile.build.version.clone(),
        platform: profile.build.platform.clone(),
        arch: profile.build.arch.clone(),
        package: profile.build.package.clone(),
        host,
        isolation,
        clocks,
        sampling: plan,
        tools: tools_of(profile)?,
        acquisition: routes_of(lanes, profile)?,
        phases,
        evidence: manifest_evidence_of(lane, profile)?,
        redactions: Vec::new(),
    })
}

/// A value a record carries, refused by name when it is absent or empty.
fn text_in(record: &Record, key: &str, source: &str) -> Result<String, String> {
    record
        .get(key)
        .filter(|text| !text.is_empty())
        .cloned()
        .ok_or_else(|| format!("{source} records no `{key}`"))
}

/// The watched host and its isolation, from the claim it kept and its attestation.
fn host_of(
    lane: &Lane,
    claim: &Record,
    profile: &Profile,
) -> Result<(bit_ids::manifest::Host, bit_ids::manifest::Isolation), String> {
    use bit_ids::manifest::{Host, HostClaim, Isolation, IsolationKind, NetworkMode};
    let claim_name = "host-claim.txt";
    let att = "capture/attestation.txt";
    let environment = text_in(claim, "runner_environment", claim_name)?;
    // ⛔ ONLY A HOST GITHUB DESTROYS IS DISPOSABLE HERE. A self-hosted runner
    // says nothing a manifest can rest `disposable` on.
    if environment != "github-hosted" {
        return Err(format!(
            "{claim_name} says runner_environment={environment}, and only a github-hosted \
             runner is one this project knows is destroyed after the job"
        ));
    }
    let platform = text_in(&lane.attestation, "platform", att)?;
    let kernel = platform
        .split_whitespace()
        .nth(1)
        .ok_or_else(|| format!("{att} platform {platform:?} names no kernel"))?;
    let host = Host {
        image: label(&text_in(claim, "image", claim_name)?)?,
        kernel: label(kernel)?,
        os: label(&text_in(claim, "os", claim_name)?)?,
        arch: profile.build.arch.clone(),
        disposable: true,
    };
    let fingerprint = text_in(claim, "fingerprint", claim_name)?;
    let isolation = Isolation {
        claim: HostClaim {
            fingerprint: Sha256Digest::parse(&format!("sha256:{fingerprint}"))
                .map_err(|error| format!("{claim_name} fingerprint: {error}"))?,
            claimed_at: instant_in(claim, "claimed_at", claim_name)?,
        },
        host: IsolationKind::EphemeralRunner,
        network: match text_in(&lane.attestation, "egress", att)?.as_str() {
            "closed" => NetworkMode::LoopbackOnly,
            other => {
                return Err(format!(
                    "{att} says egress={other}, which is not loopback only"
                ));
            }
        },
        external_reason: None,
    };
    Ok((host, isolation))
}

/// Both clocks: the wall from the claim to the finish, and the host's uptime across them.
fn clocks_of(
    lane: &Lane,
    claim: &Record,
    isolation: &bit_ids::manifest::Isolation,
) -> Result<bit_ids::manifest::Clocks, String> {
    let att = "capture/attestation.txt";
    let from = uptime_ns(
        &text_in(claim, "uptime_at_claim", "host-claim.txt")?,
        "uptime_at_claim",
    )?;
    let to = uptime_ns(
        &text_in(&lane.attestation, "uptime_at_finish", att)?,
        "uptime_at_finish",
    )?;
    Ok(bit_ids::manifest::Clocks {
        wall_start: isolation.claim.claimed_at.clone(),
        wall_end: instant_in(&lane.attestation, "finished_at", att)?,
        monotonic_elapsed_ns: to
            .checked_sub(from)
            .ok_or("the uptime at finish is before the uptime at the claim")?,
    })
}

/// Every tool that took part.
///
/// ⚠ This repository's own tools carry its version, as the observer reports it;
/// their digests in the attestation say which build. The connector carries the
/// version it described.
fn tools_of(profile: &Profile) -> Result<Vec<bit_ids::manifest::Tool>, String> {
    use bit_ids::manifest::{Tool, ToolRole};
    let ours = profile
        .capture
        .connectors
        .iter()
        .find(|connector| connector.id == profile.capture.observer)
        .map(|connector| connector.version.clone())
        .ok_or("the record names no observer among its connectors")?;
    let mut tools = vec![
        Tool {
            id: slug("bit-ids-probe"),
            version: ours.clone(),
            role: ToolRole::Observer,
        },
        Tool {
            id: slug("install-client"),
            version: ours,
            role: ToolRole::Acquisition,
        },
    ];
    for connector in &profile.capture.connectors {
        if connector.id != profile.capture.observer {
            tools.push(Tool {
                id: connector.id.clone(),
                version: connector.version.clone(),
                role: ToolRole::Connector,
            });
        }
    }
    tools.sort_by_key(|tool| tool.id.to_string());
    Ok(tools)
}

/// Every route's identity, each from its own lane's install record.
fn routes_of(
    lanes: &[Lane],
    profile: &Profile,
) -> Result<Vec<bit_ids::manifest::AcquisitionIdentity>, String> {
    let mut acquisition = Vec::new();
    for other in lanes {
        let ins = format!("install-{}.txt", other.route);
        let route = route_of(other, &profile.build.version)?;
        let bytes: u64 = text_in(&other.install, "installed_binary_bytes", &ins)?
            .parse()
            .map_err(|_| format!("{ins} installed_binary_bytes is not a size"))?;
        acquisition.push(bit_ids::manifest::AcquisitionIdentity {
            route: route.id,
            source: route.origin,
            retrieved_at: instant_in(&other.install, "retrieved_at", &ins)?,
            artifact: route.artifact,
            bytes,
            signature: match text_in(&other.install, "signature", &ins)?.as_str() {
                "not_checked" => SignatureStatus::NotChecked,
                "verified" => SignatureStatus::Verified,
                "unsigned" => SignatureStatus::Unsigned,
                other => return Err(format!("{ins} signature={other} is not a status")),
            },
            installed_version: Version::parse(&text_in(&other.install, "reported_version", &ins)?)
                .map_err(|error| format!("{ins} reported_version: {error}"))?,
        });
    }
    acquisition.sort_by_key(|entry| entry.route.to_string());
    Ok(acquisition)
}

/// The phases, as milestones across both runners.
fn phases_of(
    lanes: &[Lane],
    watched: usize,
    profile: &Profile,
    clocks: &bit_ids::manifest::Clocks,
) -> Result<Vec<bit_ids::manifest::Phase>, String> {
    use bit_ids::manifest::{Phase, PhaseName};
    let lane = &lanes[watched];
    let resolution_of = |other: &Lane, key: &str| -> Result<Instant, String> {
        // ⚠ A PACKAGE ROUTE RESOLVES INSIDE ITS INSTALL: the index is asked as
        // the install's first act (`winget show`), so both of its resolution
        // instants are the install's start. `CI-11`.
        if other.resolution.is_none() && other.route == "package" {
            return instant_in(
                &other.install,
                "started_at",
                &format!("install-{}.txt", other.route),
            );
        }
        let record = other
            .resolution
            .as_ref()
            .ok_or_else(|| format!("route {} uploaded no resolution document", other.route))?;
        instant_in(record, key, &other.resolution_path.display().to_string())
    };
    let mut resolved = Vec::new();
    let mut installed = Vec::new();
    for other in lanes {
        resolved.push(resolution_of(other, "finished_at")?);
        installed.push(instant_in(
            &other.install,
            "finished_at",
            &format!("install-{}.txt", other.route),
        )?);
    }
    let resolved = resolved.into_iter().max().ok_or("no route")?;
    let installed = installed.into_iter().max().ok_or("no route")?;
    let resolve_start = resolution_of(lane, "started_at")?;
    let observe_start = profile.capture.captured_at.clone();
    let serving = std::fs::read_to_string(&lane.observer_log)
        .map_err(|error| format!("{}: {error}", lane.observer_log.display()))?
        .lines()
        .find_map(|line| {
            line.strip_prefix("serving for ")
                .and_then(|rest| rest.strip_suffix('s'))
        })
        .and_then(|seconds| seconds.parse::<i64>().ok())
        .ok_or("the observer's log says for how long it served nowhere")?;
    let observe_end = instant_at(epoch_of(&observe_start)? + serving)?;
    let span = |name: PhaseName, started_at: &Instant, ended_at: &Instant, detail: &str| {
        Ok::<_, String>(Phase {
            name,
            started_at: started_at.clone(),
            ended_at: ended_at.clone(),
            detail: label(detail)?,
        })
    };
    Ok(vec![
        span(
            PhaseName::Planned,
            &clocks.wall_start,
            &resolve_start,
            "the host was claimed and the routes chosen",
        )?,
        span(
            PhaseName::Resolved,
            &resolve_start,
            &resolved,
            "every route resolved its version",
        )?,
        span(
            PhaseName::AcquiredTwice,
            &resolved,
            &installed,
            "every route installed, the last at this instant",
        )?,
        span(
            PhaseName::Installed,
            &installed,
            &installed,
            "every installed build reported its version",
        )?,
        span(
            PhaseName::Observed,
            &observe_start,
            &observe_end,
            "the observer served the build for its window",
        )?,
        span(
            PhaseName::Corroborated,
            &observe_end,
            &clocks.wall_end,
            "the connector read the bundle and the readings were compared",
        )?,
    ])
}

/// The record's evidence, as the manifest declares it.
///
/// ⚠ Nothing on this path scrubs: the observer prints `false` for every row it
/// writes, and the metainfo and the version output pass no scrubber.
fn manifest_evidence_of(
    lane: &Lane,
    profile: &Profile,
) -> Result<Vec<bit_ids::manifest::EvidenceRecord>, String> {
    use bit_ids::manifest::{EvidenceRecord, PhaseName};
    let log = std::fs::read_to_string(&lane.observer_log)
        .map_err(|error| format!("{}: {error}", lane.observer_log.display()))?;
    if log
        .lines()
        .any(|line| line.starts_with("evidence ") && line.split_whitespace().nth(5) == Some("true"))
    {
        return Err(
            "the observer redacted a transcript, and its declarations are not uploaded".to_owned(),
        );
    }
    let mut evidence = Vec::new();
    for entry in &profile.evidence {
        let (produced_by, phase) = match entry.kind {
            EvidenceKind::ProcessOutput => (slug("install-client"), PhaseName::Installed),
            _ => (slug("bit-ids-probe"), PhaseName::Observed),
        };
        evidence.push(EvidenceRecord {
            id: entry.id.clone(),
            kind: entry.kind,
            path: entry.path.clone(),
            bytes: entry.bytes,
            sha256: entry.sha256,
            produced_by,
            phase,
            redacted: false,
        });
    }
    evidence.sort_by_key(|record| record.id.to_string());
    Ok(evidence)
}

/// Everything one lane's artifacts cannot supply, probed independently.
///
/// ⛔ **Each derivation is asked separately so one gap cannot hide another.** The
/// record-building path is a chain of `?`, which reports what it tripped on
/// first and nothing about the rest; this file promises a reader every field the
/// capture path would have to record, and that promise needs a probe per field.
fn lane_gaps(lane: &Lane) -> Vec<String> {
    let mut out = Vec::new();
    if let Err(error) = identity_of(lane) {
        out.push(error);
    }
    match lane.connectors() {
        // ⛔ COUNTED HERE AND NOT ONLY AT VALIDATION. `E-CAP-01` fires inside
        // `validate`, which a lane missing any other field never reaches - so
        // the one gap every capture this project has run shares would be the
        // one a report never mentioned.
        Ok(connectors) if connectors.len() < 2 => out.push(format!(
            "the attestation declares {} connector(s). `E-CAP-01` refuses a record with fewer \
             than two at the VALIDITY gate, so this is not a publication question: name the \
             second in `connectors=` and give it a report in the bundle",
            connectors.len()
        )),
        Ok(_) => {}
        Err(error) => out.push(error),
    }
    if let Err(error) = source_identity(lane) {
        out.push(error);
    }
    if let Err(error) = lane.resolver() {
        out.push(error);
    }
    if let Err(error) = lane.acquired() {
        out.push(error);
    }
    out
}

/// The one version every lane must have reported, or a refusal naming the pair.
///
/// ⛔ Absolute 4 is checked here rather than trusted: two lanes that reported two
/// versions are two builds, and a record over them would be a claim no capture
/// made. ⭐ This is also why one shared resolution is the wrong fix for that
/// worry - the comparison happens after installation, on what each build said.
fn one_version(lanes: &[Lane]) -> Result<Version, String> {
    let install = Path::new("install-<route>.txt");
    let version = Version::parse(need(&lanes[0].install, "reported_version", install)?)
        .map_err(|error| format!("reported_version: {error}"))?;
    for lane in lanes {
        let reported = need(&lane.install, "reported_version", install)?;
        if reported != version.to_string() {
            return Err(format!(
                "lane {} reported {reported} and lane {} reported {version}; two routes must \
                 resolve one stable version",
                lane.route, lanes[0].route
            ));
        }
    }
    Ok(version)
}

/// What `E-ACQ-07` and `E-ACQ-08` compare, asked of the artifacts.
///
/// ⛔ **It runs before any record is built.** Those codes fire inside `validate`,
/// which is reached only once every route has been assembled - so a lane missing
/// some other field would hide a resolver collision behind its own refusal, and
/// the collision is the more expensive fact to learn late.
fn independence(lanes: &[Lane]) -> Result<Vec<String>, String> {
    let mut out = Vec::new();
    for (index, lane) in lanes.iter().enumerate() {
        for earlier in &lanes[..index] {
            if earlier.resolver()? == lane.resolver()? {
                out.push(format!(
                    "  ⛔ E-ACQ-07: lanes {} and {} both resolved through {}, so they are one \
                     route. Read out of each lane's own resolution document, which name one \
                     source and carry one listing digest.",
                    earlier.route,
                    lane.route,
                    lane.resolver()?
                ));
            }
            if earlier.delivery() == lane.delivery() {
                out.push(format!(
                    "  ⛔ E-ACQ-08: lanes {} and {} were both delivered by {}",
                    earlier.route,
                    lane.route,
                    lane.delivery()
                ));
            }
        }
    }
    Ok(out)
}

/// ⛔ **What this wrote is not a valid corpus, which nothing said.**
///
/// ⚠ `CI-09` carried this as *the store this writes is a store of records with
/// no runs, which `check-store` accepts and a publication would not*. Half of
/// that is right and half was never checked: `validate_corpus` REFUSES such a
/// store today, with `E-CRP-01` per record - *a record without its run cannot be
/// replayed* - so it is not a publication gate that would catch it, it is the
/// corpus validator, and it would have caught it all along. ⛔ What was true is
/// that nothing on THIS path asked, so a reader of a green run saw two stars and
/// a path and took the store for one a consumer could open.
///
/// ⭐ **So it is asked here, over what was actually written.** The records are
/// the ones handed to `to_json`, and the manifests are the ones this run wrote
/// beside them. Until `CI-09` step 5 no step of a capture wrote a `RunManifest`,
/// so every record this reported on was `E-CRP-01`.
///
/// ⛔ **The tree is READ BACK OFF THE DISK, not assembled from the records.** A
/// first version built the corpus over an EMPTY `StoreTree`, which silently
/// narrowed the question: `E-CRP-06` is checked over the tree, so a store whose
/// evidence no run declares answered clean because the validator was handed no
/// evidence to look at. ⚠ Digesting the bytes that arrived is also what keeps
/// this from being a record agreeing with itself - the failure
/// `validate-corpus`'s own header names.
///
/// ⚠ **It asks about what THIS run wrote and says so.** Only the paths the
/// records account for are read, so a stray file already in the output directory
/// is invisible here. `cargo run -p bit-ids --example validate-corpus -- STORE`
/// is the whole-directory question and walks it.
///
/// ⛔ **It is a report and not a refusal**, for the reason `classify` above
/// gives: validity and publishability are separate gates at every level here,
/// and a third gate collapsing them would stop an incomplete corpus being
/// recordable at all. ⚠ The honest state is a store that exists, is not a corpus
/// yet, and says which document is missing.
///
/// Asked from its own function rather than inline because `run` is at clippy's
/// line limit.
fn report_corpus(
    report: &mut String,
    written: &[Profile],
    manifests: &[(RelPath, RunManifest)],
    out: &Path,
) -> Result<(), String> {
    let mut tree = bit_ids::store::StoreTree::new();
    let mut filed = Vec::new();
    let mut absent = Vec::new();
    for profile in written {
        let key = StoreKey::of_profile(profile);
        let path = key.profile_path().map_err(|error| error.to_string())?;
        if !measure_into(&mut tree, out, &path)? {
            absent.push(path.to_string());
        }
        for entry in &profile.evidence {
            let rel = key
                .evidence_path(&entry.path)
                .map_err(|error| error.to_string())?;
            if !measure_into(&mut tree, out, &rel)? {
                absent.push(rel.to_string());
            }
        }
        filed.push((path, profile.clone()));
    }
    // ⛔ The manifests are read back off the disk as the records' evidence is,
    // so the tree holds what was written rather than what was meant.
    for (path, _) in manifests {
        if !measure_into(&mut tree, out, path)? {
            absent.push(path.to_string());
        }
    }
    // ⛔ SAID HERE BECAUSE NO RULE CAN SAY IT. `E-CRP-03` is the refusal for an
    // artifact a store does not carry and it is checked per run manifest, so
    // with none in the store a record may cite bytes that are not there and
    // every gate this project has stays green. That is how the assembler came to
    // file four citations per record at paths it never wrote to.
    for path in &absent {
        writeln!(report, "  ⛔ cited and not carried: {path}").expect("a String cannot fail");
    }
    let mut corpus = bit_ids::corpus::Corpus::new(tree);
    for (path, profile) in filed {
        corpus.insert_profile(path, profile);
    }
    for (path, manifest) in manifests {
        corpus.insert_manifest(path.clone(), manifest.clone());
    }
    match bit_ids::corpus::validate_corpus(&corpus) {
        Ok(()) => {
            writeln!(report, "  corpus: valid, {} record(s)", written.len())
                .expect("a String cannot fail");
        }
        Err(violations) => {
            writeln!(
                report,
                "  ⚠ corpus: what was written is NOT a valid corpus yet"
            )
            .expect("a String cannot fail");
            for line in violations.to_string().lines() {
                writeln!(report, "       {line}").expect("a String cannot fail");
            }
            // ⚠ NAMED RATHER THAN LEFT TO BE INFERRED. Every capture this
            // project runs lands here, so a reader is owed the reason rather than
            // a code to look up: no step of the capture path writes a run
            // manifest, and inventing one would produce a document `bind` then
            // compares against a record that agrees with it for no reason.
            //
            // ⛔ PRINTED ONLY WHEN THE VIOLATIONS ARE ACTUALLY THAT. A first
            // version printed it under every refusal, so a placement error or an
            // invalid record would have been explained by a missing manifest -
            // an explanation attached to a finding it does not fit.
            if violations.has("E-CRP-01") {
                writeln!(
                    report,
                    "       nothing on the capture path writes a {}; TODO/ci.md, CI-09",
                    bit_ids::store::MANIFEST_FILE
                )
                .expect("a String cannot fail");
            }
        }
    }
    Ok(())
}

/// Digests one file the store carries and records what is there.
///
/// ⛔ The length and the digest are the bytes' own rather than the record's
/// claim about them, which is the whole point of handing the validator a tree.
///
/// ⚠ **A path that is not there is reported, not refused.** `Ok(false)` says the
/// store does not carry it and the tree gets no entry, which is what a walk of
/// the directory would have found. Aborting instead would turn a store this
/// example can describe into `no record was written`, which is a lie about a run
/// that wrote two.
fn measure_into(
    tree: &mut bit_ids::store::StoreTree,
    out: &Path,
    path: &RelPath,
) -> Result<bool, String> {
    let full = out.join(path.as_str());
    let bytes = match std::fs::read(&full) {
        Ok(bytes) => bytes,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(false),
        Err(error) => return Err(format!("{}: {error}", full.display())),
    };
    let length = u64::try_from(bytes.len()).map_err(|error| format!("{path}: {error}"))?;
    tree.insert(
        path.clone(),
        bit_ids::store::Entry::Object(bit_ids::store::ObjectRef {
            bytes: length,
            sha256: Sha256Digest::of(&bytes),
        }),
    );
    Ok(true)
}

/// Writes the run manifest beside one record, and says whether it could.
///
/// ⭐ Written only when it validates and binds to the record it describes;
/// anything else is a refusal naming why, because a store with a record and no
/// run is not a corpus. `CI-09`.
#[allow(clippy::too_many_arguments)]
fn write_manifest(
    report: &mut String,
    lanes: &[Lane],
    index: usize,
    profile: &Profile,
    plan: SamplingPlan,
    key: &StoreKey,
    out: &Path,
    manifests: &mut Vec<(RelPath, RunManifest)>,
) -> Result<bool, String> {
    if let Some(barrier) = &lanes[index].barrier {
        let verdict = barrier.get("verdict").map_or("unrecorded", String::as_str);
        if verdict != "passed" {
            writeln!(
                report,
                "  ⚠ lane {}: the install barrier says {verdict}",
                lanes[index].route
            )
            .expect("a String cannot fail");
        }
    }
    let made = manifest_of(lanes, index, profile, plan).and_then(|manifest| {
        let document = manifest.to_json().map_err(|error| error.to_string())?;
        bit_ids::bind(&manifest, profile).map_err(|error| error.to_string())?;
        Ok((manifest, document))
    });
    match made {
        Ok((manifest, document)) => {
            let rel = key.manifest_path().map_err(|error| error.to_string())?;
            let target = out.join(rel.as_str());
            std::fs::write(&target, document.as_bytes())
                .map_err(|error| format!("{}: {error}", target.display()))?;
            writeln!(report, "  ⭐ lane {}: {}", lanes[index].route, rel)
                .expect("a String cannot fail");
            manifests.push((rel, manifest));
            Ok(true)
        }
        Err(error) => {
            writeln!(report, "  ⛔ lane {}: no run manifest", lanes[index].route)
                .expect("a String cannot fail");
            for line in error.lines() {
                writeln!(report, "       {line}").expect("a String cannot fail");
            }
            Ok(false)
        }
    }
}

/// Whether each written record publishes among the others, and how the pair
/// classifies.
fn report_publishable(report: &mut String, written: &[Profile]) {
    for (index, profile) in written.iter().enumerate() {
        let others: Vec<&Profile> = written
            .iter()
            .enumerate()
            .filter(|(other, _)| *other != index)
            .map(|(_, other)| other)
            .collect();
        writeln!(report, "  lane {}:", profile.capture.observed_route)
            .expect("a String cannot fail");
        match bit_ids::agreement::publishable_among(profile, &others) {
            Ok(()) => writeln!(report, "       publishable").expect("a String cannot fail"),
            Err(blockers) => {
                writeln!(report, "       provisional, not publishable")
                    .expect("a String cannot fail");
                for line in blockers.to_string().lines() {
                    writeln!(report, "       {line}").expect("a String cannot fail");
                }
            }
        }
    }

    if written.len() == 2 {
        // ⭐ The pair is what `ACQ-03` exists to classify, and this is the first
        // input that can reach `BuildEquivalent` rather than `Unresolved`.
        let comparison = bit_ids::classify_across(&[&written[0], &written[1]]);
        writeln!(report, "  classify_across: {}", comparison.outcome)
            .expect("a String cannot fail");
        for reason in &comparison.reasons {
            writeln!(report, "       {reason}").expect("a String cannot fail");
        }
    }
}

fn run(lanes: &[Lane], out: &Path) -> Result<String, String> {
    let version = one_version(lanes)?;

    let mut report = String::new();
    let mut refusals = 0_usize;
    let mut written = Vec::new();
    let mut manifests = Vec::new();
    for line in independence(lanes)? {
        refusals += 1;
        writeln!(report, "{line}").expect("a String cannot fail");
    }

    for index in 0..lanes.len() {
        // ⛔ EVERY GAP, NOT THE FIRST. This file's own header promises that a run
        // which cannot produce a record says exactly what the capture path would
        // have to record, and a `?` chain says only what it tripped on first.
        // ⚠ Measured on run 14: adding the `package` derivation moved that
        // refusal in front of the commit one, and two gaps a previous run had
        // named silently disappeared from the report.
        let gaps = lane_gaps(&lanes[index]);
        if !gaps.is_empty() {
            for gap in gaps {
                refusals += 1;
                writeln!(report, "  ⛔ lane {}: {gap}", lanes[index].route)
                    .expect("a String cannot fail");
            }
            continue;
        }
        let (profile, plan) = match profile_of(lanes, index, &version) {
            Ok(found) => found,
            Err(error) => {
                refusals += 1;
                writeln!(report, "  ⛔ lane {}: {error}", lanes[index].route)
                    .expect("a String cannot fail");
                continue;
            }
        };
        // ⛔ `to_json` validates, so a record that cannot be written is a record
        // that does not exist rather than one filed under a caveat.
        match profile.to_json() {
            Ok(document) => {
                let key = StoreKey::of_profile(&profile);
                let path = key.profile_path().map_err(|error| error.to_string())?;
                let full = out.join(path.as_str());
                if let Some(parent) = full.parent() {
                    std::fs::create_dir_all(parent)
                        .map_err(|error| format!("{}: {error}", parent.display()))?;
                }
                std::fs::write(&full, document.as_bytes())
                    .map_err(|error| format!("{}: {error}", full.display()))?;
                // ⛔ EVERY LANE'S ARTIFACTS, NOT THIS LANE'S. The record's
                // evidence list is built over ALL the lanes, because a field
                // citing the other route's install record is what makes the pair
                // comparable at all - and this copied only `lanes[index]`'s four
                // files, so each record cited four artifacts at paths under its
                // own evidence root that nothing ever wrote there.
                //
                // ⚠ `E-CRP-03` is exactly that refusal and it could not fire,
                // because it is checked per RUN MANIFEST and no step of this path
                // writes one. Found on 2026-09-17 by handing `validate_corpus` a
                // tree read back off the disk rather than an empty one.
                //
                // ⭐ The paths are namespaced by route already, so both lanes'
                // artifacts sit side by side under each capture and neither
                // record depends on the other one's directory surviving.
                for lane in lanes {
                    for (entry, source) in evidence_of(lane)? {
                        let rel = key
                            .evidence_path(&entry.path)
                            .map_err(|error| error.to_string())?;
                        let target = out.join(rel.as_str());
                        if let Some(parent) = target.parent() {
                            std::fs::create_dir_all(parent)
                                .map_err(|error| format!("{}: {error}", parent.display()))?;
                        }
                        std::fs::copy(&source, &target)
                            .map_err(|error| format!("{}: {error}", target.display()))?;
                    }
                }
                writeln!(report, "  ⭐ lane {}: {}", lanes[index].route, path)
                    .expect("a String cannot fail");
                // ⭐ AND THE RUN MANIFEST BESIDE IT, `CI-09`.
                if !write_manifest(
                    &mut report,
                    lanes,
                    index,
                    &profile,
                    plan,
                    &key,
                    out,
                    &mut manifests,
                )? {
                    refusals += 1;
                }
                written.push(profile);
            }
            Err(error) => {
                refusals += 1;
                writeln!(report, "  ⛔ lane {}: refused", lanes[index].route)
                    .expect("a String cannot fail");
                for line in error.to_string().lines() {
                    writeln!(report, "       {line}").expect("a String cannot fail");
                }
            }
        }
    }

    // ⛔ WRITTEN IS NOT PUBLISHABLE, AND THIS REPORT SAID NOTHING ABOUT THE
    // DIFFERENCE. Found on 2026-09-10 by giving a lane a second connector that
    // read different bytes: the record kept the conflict, `to_json` accepted it
    // - correctly, because refusing it would lose the evidence of the
    // disagreement - and the report printed a star and a path. A session reading
    // that would have taken a record carrying a connector conflict for a
    // finished one.
    // ⚠ It is NOT a refusal. `validate` and `publishable` are separate gates at
    // every level here, and collapsing them would stop the disagreement being
    // recordable at all.
    // ⛔ AND IT IS ASKED ONCE EVERY LANE IS WRITTEN, not inside the loop, because
    // it is not a property of one document: `E-PUB-04` is settled by the OTHER
    // route's capture. Asked per lane it reported every record of a
    // byte-different pair provisional while holding the record that settles it.
    report_publishable(&mut report, &written);
    report_corpus(&mut report, &written, &manifests, out)?;
    if refusals > 0 {
        return Err(report);
    }
    Ok(report)
}

fn main() -> ExitCode {
    let mut lanes = Vec::new();
    let mut positional = Vec::new();
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        if arg == "--lane" {
            let Some(pair) = args.next() else {
                let _ = writeln!(std::io::stderr(), "--lane needs CAPTURE_DIR,INSTALL_DIR");
                return ExitCode::from(2);
            };
            let Some((capture, install)) = pair.split_once(',') else {
                let _ = writeln!(std::io::stderr(), "--lane needs CAPTURE_DIR,INSTALL_DIR");
                return ExitCode::from(2);
            };
            lanes.push((PathBuf::from(capture), PathBuf::from(install)));
            continue;
        }
        positional.push(arg);
    }
    let [out] = positional.as_slice() else {
        let _ = writeln!(
            std::io::stderr(),
            "usage: assemble-capture --lane CAPTURE_DIR,INSTALL_DIR ... STORE_DIR"
        );
        return ExitCode::from(2);
    };
    if lanes.len() < 2 {
        let _ = writeln!(
            std::io::stderr(),
            "assemble-capture: {} lane(s). `E-ACQ-01` needs two routes, so a record needs a lane \
             per route",
            lanes.len()
        );
        return ExitCode::from(2);
    }
    let out = Path::new(out);
    if !out.is_dir() {
        let _ = writeln!(std::io::stderr(), "{}: not a directory", out.display());
        return ExitCode::from(2);
    }

    let mut read = Vec::new();
    for (capture, install) in &lanes {
        match Lane::read(capture, install) {
            Ok(lane) => read.push(lane),
            Err(error) => {
                let _ = writeln!(std::io::stderr(), "assemble-capture: {error}");
                return ExitCode::from(2);
            }
        }
    }

    match run(&read, out) {
        Ok(report) => {
            let _ = write!(std::io::stdout(), "{report}");
            let _ = writeln!(
                std::io::stdout(),
                "assemble-capture: {} record(s) written into {}",
                read.len(),
                out.display()
            );
            ExitCode::SUCCESS
        }
        Err(report) => {
            let _ = write!(std::io::stderr(), "{report}");
            let _ = writeln!(
                std::io::stderr(),
                "assemble-capture: no record was written. Every line above is a field the capture \
                 path would have to record."
            );
            ExitCode::from(1)
        }
    }
}
