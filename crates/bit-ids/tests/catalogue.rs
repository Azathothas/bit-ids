//! `LIB-01`'s Prove: the public API loads a release fixture, rejects a digest
//! mismatch, and selects a platform/version profile with no network access.
//!
//! ⛔ **"No network access" is proved by the crate having no way to reach one.**
//! There is no socket, no HTTP client and no host anywhere in `bit-ids`, so
//! these cases cannot accidentally be passing because a fetch happened to
//! succeed. What is exercised instead is the seam a caller would fetch through,
//! and that seam refuses bytes that are not the publication's.
//!
//! ⚠ **The bundle is built by the same code that publishes one.** A fixture
//! hand-written to satisfy the reader would agree with the reader and say
//! nothing about a real publication, so the tree here goes through `assemble`
//! exactly as `PUB-01` does.

use bit_ids::adapter::{Answer, Claim, Shape, Unmeasured, check};
use bit_ids::canonical::{RelPath, Sha256Digest, Slug, Version};
use bit_ids::catalogue::{Bundle, Catalogue, INDEX_PATH, Retrieval, plan, retrieve};
use bit_ids::index::{IndexKind, Indexes, build};
use bit_ids::observation::FieldPath;
use bit_ids::release::{CHECKSUMS_FILE, RELEASE_MANIFEST_FILE, assemble};
use bit_ids::store::{Entry, ObjectRef, StoreKey, StoreTree};
use bit_ids::{Corpus, Profile, RunManifest};
use std::collections::BTreeMap;

const PROFILE: &str = include_str!("fixtures/valid-profile.json");
const MANIFEST: &str = include_str!("fixtures/valid-manifest.json");

fn slug(text: &str) -> Slug {
    Slug::parse(text).expect("slug")
}

/// The fixture record and its run, re-versioned to something a numeric scheme
/// can order, and re-identified.
///
/// ⛔ **The shipped fixture's version is `0.0.0-fixture`, which no numeric
/// scheme orders**, so a latest view over it is refused under `E-VIW-02`. That
/// is the fixture doing its job for the fail-closed path and being useless for
/// the ordering one, which is the same reason `check-indexes` asks
/// `build-store` for versions rather than using the shipped record.
///
/// ⚠ Every route's reported version moves with it, because `E-ACQ-04` refuses a
/// route that installed a version the record does not declare, and the
/// identifier is re-derived, because it digests the version.
fn record_at(version_text: &str) -> (Profile, RunManifest) {
    let mut record = Profile::from_json(PROFILE).expect("the fixture record validates");
    let mut run = RunManifest::from_json(MANIFEST).expect("the fixture run validates");
    let version = Version::parse(version_text).expect("a reported version");

    record.build.version = version.clone();
    for route in &mut record.acquisition {
        route.installed_version = version.clone();
        route.resolved_version = version.clone();
    }
    record.id = bit_ids::identity::RecordId::derive(&bit_ids::identity::RecordKey {
        schema: &record.schema,
        target: &record.target.id,
        version: &record.build.version,
        platform: &record.build.platform,
        arch: &record.build.arch,
        package: &record.build.package,
        capture: &record.capture.id,
    });
    run.version = version.clone();
    // ⚠ `E-MAN-12` compares the run's routes against the version it records, so
    // both documents move together or neither validates. That refusal is the
    // manifest holding its own overlap with the profile in step, and it fired
    // on the first draft of this helper.
    for route in &mut run.acquisition {
        route.installed_version = version.clone();
    }
    (record, run)
}

/// A publication carrying one record, its run and the index over it, assembled
/// the way `PUB-01` assembles one.
fn publication() -> (Bundle, Sha256Digest) {
    let (profile, manifest) = record_at("1.2.3");

    let record_path = StoreKey::of_profile(&profile)
        .profile_path()
        .expect("the record has a path");
    let run_path = StoreKey::of_manifest(&manifest)
        .manifest_path()
        .expect("the run has a path");

    let mut files: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    files.insert(
        record_path.as_str().to_owned(),
        profile.to_json().expect("the record writes").into_bytes(),
    );
    files.insert(
        run_path.as_str().to_owned(),
        manifest.to_json().expect("the run writes").into_bytes(),
    );

    // The views, derived the one way `CORPUS-03` derives them.
    let mut tree = StoreTree::new();
    for (path, bytes) in &files {
        tree.insert(
            RelPath::parse(path).expect("path"),
            Entry::Object(ObjectRef {
                bytes: bytes.len() as u64,
                sha256: Sha256Digest::of(bytes),
            }),
        );
    }
    let mut corpus = Corpus::new(tree.clone());
    corpus.insert_profile(record_path.clone(), profile.clone());
    corpus.insert_manifest(run_path, manifest);
    let schemes = BTreeMap::from([(
        profile.target.id.clone(),
        bit_ids::resolution::VersionScheme {
            tag_prefix: None,
            min_components: 1,
            max_components: 4,
            floor_major: None,
            series: None,
        },
    )]);
    let indexes: Indexes = build(&corpus, &schemes).expect("the views build");
    files.insert(INDEX_PATH.to_owned(), indexes.to_json().into_bytes());

    let mut described = StoreTree::new();
    for (path, bytes) in &files {
        described.insert(
            RelPath::parse(path).expect("path"),
            Entry::Object(ObjectRef {
                bytes: bytes.len() as u64,
                sha256: Sha256Digest::of(bytes),
            }),
        );
    }
    let release = assemble(&described).expect("the tree assembles");
    let manifest_bytes = release.manifest_json().into_bytes();
    let checksums = release.checksums(&manifest_bytes).into_bytes();
    let checksums_digest = Sha256Digest::of(&checksums);

    let mut bundle = Bundle::new();
    for (path, bytes) in files {
        bundle.insert(path, bytes);
    }
    bundle.insert(RELEASE_MANIFEST_FILE, manifest_bytes);
    bundle.insert(CHECKSUMS_FILE, checksums);
    (bundle, checksums_digest)
}

struct FromBundle(Bundle);

impl Retrieval for FromBundle {
    type Error = String;
    fn fetch(&self, path: &str) -> Result<Vec<u8>, String> {
        self.0
            .get(path)
            .map(<[u8]>::to_vec)
            .ok_or_else(|| format!("{path} is not in this publication"))
    }
}

// -- The Prove ---------------------------------------------------------------

#[test]
fn catalogue_opens_a_release_fixture_and_answers_from_it() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("a verified publication opens");

    assert_eq!(catalogue.records().len(), 1);
    let record = catalogue.records()[0];
    // ⛔ Through `Profile::from_json`, so a consumer holds a record this project
    // would publish and not merely bytes shaped like one.
    assert_eq!(record.schema.as_str(), bit_ids::PROFILE_SCHEMA);
    assert_eq!(catalogue.record(record.id).map(|p| p.id), Some(record.id));
}

#[test]
fn catalogue_selects_a_platform_and_version_profile_without_a_network() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    let record = catalogue.records()[0];

    let selected = catalogue
        .latest(
            &record.target.id,
            &record.build.platform,
            &record.build.arch,
            &record.build.package,
        )
        .expect("the latest view answers this build line");
    assert_eq!(selected.id, record.id);

    // A build line the catalogue does not carry answers nothing rather than
    // answering with the nearest one.
    assert!(
        catalogue
            .latest(
                &record.target.id,
                &slug("some-other-platform"),
                &record.build.arch,
                &record.build.package,
            )
            .is_none()
    );

    let by_version = catalogue.at_version(&record.target.id, &record.build.version);
    assert_eq!(by_version.len(), 1);
    assert_eq!(by_version[0].id, record.id);
    assert!(
        catalogue
            .at_version(
                &record.target.id,
                &Version::parse("0.0.0-not-measured").expect("version")
            )
            .is_empty()
    );
}

#[test]
fn catalogue_rejects_a_digest_mismatch() {
    // ⛔ The second clause of the Prove. One byte of one record moves and the
    // manifest still describes the old ones, so the bundle is refused before a
    // single question can be asked of it.
    let (bundle, digest) = publication();
    let record_path = bundle
        .paths()
        .find(|path| path.starts_with("profiles/"))
        .expect("a record")
        .to_owned();
    let mut tampered = bundle.clone();
    let mut bytes = bundle.get(&record_path).expect("bytes").to_vec();
    bytes.push(b'\n');
    tampered.insert(record_path, bytes);

    let violations =
        Catalogue::open(&tampered, Some(&digest)).expect_err("a moved byte is refused");
    assert!(violations.has("E-LIB-02"), "{violations}");
}

/// Whether one refusal, with that code, names that path.
///
/// ⛔ **The code alone is not enough here, and a mutation pass is what said so.**
/// Two independent checks both report `E-LIB-02`: the per-file digest comparison
/// and the manifest-against-the-bytes one. Each masks the other, so a case
/// asserting only the code passed with either of them deleted. The path is what
/// separates them.
fn refuses_at(violations: &bit_ids::Violations, code: &str, at: &str) {
    assert!(
        violations
            .errors()
            .iter()
            .any(|error| error.code() == code && error.at() == at),
        "expected {code} at {at}, got {:?}",
        violations
            .errors()
            .iter()
            .map(|e| (e.code(), e.at()))
            .collect::<Vec<_>>()
    );
}

#[test]
fn catalogue_names_the_file_whose_bytes_moved_and_not_only_the_manifest() {
    let (bundle, digest) = publication();
    let record_path = bundle
        .paths()
        .find(|path| path.starts_with("profiles/"))
        .expect("a record")
        .to_owned();
    let mut tampered = bundle.clone();
    let mut bytes = bundle.get(&record_path).expect("bytes").to_vec();
    bytes.push(b'\n');
    tampered.insert(record_path.clone(), bytes);

    let violations = Catalogue::open(&tampered, Some(&digest)).expect_err("refused");
    // ⛔ The per-file comparison, pinned to the file it is about.
    refuses_at(&violations, "E-LIB-02", &record_path);
}

#[test]
fn catalogue_refuses_a_manifest_whose_bytes_moved_while_its_digests_still_match() {
    // ⛔ The other half of the pair, isolated. A trailing newline leaves every
    // per-file digest correct and the manifest's own bytes different, so only
    // the manifest-against-the-bytes comparison can fire. Without a case of
    // this shape, deleting that comparison left the whole suite green.
    let (bundle, _) = publication();
    let mut moved = bundle.clone();
    let mut bytes = bundle
        .get(RELEASE_MANIFEST_FILE)
        .expect("manifest")
        .to_vec();
    bytes.push(b'\n');
    moved.insert(RELEASE_MANIFEST_FILE, bytes);

    let violations = Catalogue::open(&moved, None).expect_err("refused");
    refuses_at(&violations, "E-LIB-02", RELEASE_MANIFEST_FILE);
}

#[test]
fn catalogue_refuses_a_checksum_file_that_does_not_cover_the_manifest() {
    // ⚠ The checksum file rewritten alone. Every other document is untouched,
    // so nothing but the checksum comparison can refuse it, and deleting that
    // comparison left the suite green until this case existed.
    let (bundle, _) = publication();
    let mut moved = bundle.clone();
    let mut bytes = bundle.get(CHECKSUMS_FILE).expect("checksums").to_vec();
    bytes.push(b'\n');
    moved.insert(CHECKSUMS_FILE, bytes);

    let violations = Catalogue::open(&moved, None).expect_err("refused");
    refuses_at(&violations, "E-LIB-04", CHECKSUMS_FILE);
}

#[test]
fn catalogue_refuses_a_latest_row_naming_a_record_the_bundle_lacks() {
    // ⚠ The `latest` view, separately from the lookup indexes. A plant that
    // blanked only the lookup loop survived, because the fixture's bad
    // identifier appears in both views and the other loop still caught it.
    // Reading what the plant changed is what said so.
    let (bundle, _) = publication();
    let (record, _) = record_at("1.2.3");
    let (other, _) = record_at("9.9.9");
    let document = core::str::from_utf8(bundle.get(INDEX_PATH).expect("index"))
        .expect("utf8")
        .replace(&record.id.to_string(), &other.id.to_string());
    let mut wrong = bundle.clone();
    wrong.insert(INDEX_PATH, document.into_bytes());
    let violations = Catalogue::open(&reassemble(&wrong), None).expect_err("refused");
    assert!(
        violations
            .errors()
            .iter()
            .any(|error| error.code() == "E-LIB-07" && error.at().starts_with("latest[")),
        "the latest view is checked too: {violations}"
    );
}

#[test]
fn catalogue_rejects_a_checksum_file_the_caller_did_not_expect() {
    // ⚠ The one file nothing in a publication proves. A caller that holds an
    // out-of-band digest is told when the publication's differs from it.
    let (bundle, _) = publication();
    let wrong = Sha256Digest::of(b"not this publication's checksum file");
    let violations =
        Catalogue::open(&bundle, Some(&wrong)).expect_err("an unexpected checksum file is refused");
    assert!(violations.has("E-LIB-05"), "{violations}");
}

#[test]
fn catalogue_rejects_a_file_nobody_described_and_one_that_is_missing() {
    let (bundle, digest) = publication();

    let mut extra = bundle.clone();
    extra.insert("formats/bit-ids-v1.json", b"undescribed".to_vec());
    let violations = Catalogue::open(&extra, Some(&digest)).expect_err("an extra file is refused");
    assert!(
        violations.has("E-LIB-02") || violations.has("E-LIB-03"),
        "{violations}"
    );

    // ⛔ And the other direction: the index is described and gone.
    let mut short = Bundle::new();
    for path in bundle.paths() {
        if path == INDEX_PATH {
            continue;
        }
        short.insert(path, bundle.get(path).expect("bytes").to_vec());
    }
    let violations = Catalogue::open(&short, Some(&digest)).expect_err("a missing file is refused");
    assert!(
        violations.has("E-LIB-08") || violations.has("E-LIB-02"),
        "{violations}"
    );
}

#[test]
fn catalogue_rejects_a_bundle_with_no_root_documents() {
    let (bundle, _) = publication();
    let mut bare = Bundle::new();
    for path in bundle.paths() {
        if path == RELEASE_MANIFEST_FILE || path == CHECKSUMS_FILE {
            continue;
        }
        bare.insert(path, bundle.get(path).expect("bytes").to_vec());
    }
    let violations = Catalogue::open(&bare, None).expect_err("a bundle with no roots is refused");
    assert!(violations.has("E-LIB-08"), "{violations}");
}

#[test]
fn catalogue_rejects_an_index_from_another_generation() {
    let (bundle, _) = publication();
    let mut wrong = bundle.clone();
    let document = core::str::from_utf8(bundle.get(INDEX_PATH).expect("index"))
        .expect("utf8")
        .replace(bit_ids::INDEX_SCHEMA, "bit-ids/index/2");
    wrong.insert(INDEX_PATH, document.into_bytes());
    // ⚠ Re-assembled, so the bundle is internally consistent and the only thing
    // wrong is the schema. Without this the digest check would fire first and
    // the case would prove nothing about compatibility.
    let violations = Catalogue::open(&reassemble(&wrong), None)
        .expect_err("another index generation is refused");
    assert!(violations.has("E-LIB-06"), "{violations}");
}

#[test]
fn catalogue_rejects_an_index_row_naming_a_record_the_bundle_lacks() {
    let (bundle, _) = publication();
    let mut wrong = bundle.clone();
    // ⚠ The real identifier is read out of the record and swapped for a
    // well-formed one that names nothing, rather than pattern-matched on the
    // prefix: a substitution over `record:sha256:` alone produces a 128-digit
    // string that does not parse, and the case would then pass for the wrong
    // reason.
    let (record, _) = record_at("1.2.3");
    // ⚠ The absent identifier is DERIVED rather than typed. A 64-digit hex
    // literal is the shape of a leaked key and `check-no-secrets --public`
    // refuses one, correctly; deriving it from a version nothing measured gives
    // a well-formed identifier that names nothing and needs no allowance in a
    // guard that exists to be blunt.
    let (other, _) = record_at("9.9.9");
    let absent = other.id.to_string();
    assert_ne!(absent, record.id.to_string());
    let document = core::str::from_utf8(bundle.get(INDEX_PATH).expect("index"))
        .expect("utf8")
        .replace(&record.id.to_string(), &absent);
    wrong.insert(INDEX_PATH, document.into_bytes());
    let violations =
        Catalogue::open(&reassemble(&wrong), None).expect_err("an unresolvable row is refused");
    assert!(violations.has("E-LIB-07"), "{violations}");
}

/// Rebuilds the two root documents over a bundle's current bytes, so a case can
/// change one file and still hand the reader a self-consistent publication.
fn reassemble(bundle: &Bundle) -> Bundle {
    let mut tree = StoreTree::new();
    for path in bundle.paths() {
        if path == RELEASE_MANIFEST_FILE || path == CHECKSUMS_FILE {
            continue;
        }
        let bytes = bundle.get(path).expect("bytes");
        tree.insert(
            RelPath::parse(path).expect("path"),
            Entry::Object(ObjectRef {
                bytes: bytes.len() as u64,
                sha256: Sha256Digest::of(bytes),
            }),
        );
    }
    let release = assemble(&tree).expect("assembles");
    let manifest = release.manifest_json().into_bytes();
    let checksums = release.checksums(&manifest).into_bytes();
    let mut out = Bundle::new();
    for path in bundle.paths() {
        if path == RELEASE_MANIFEST_FILE || path == CHECKSUMS_FILE {
            continue;
        }
        out.insert(path, bundle.get(path).expect("bytes").to_vec());
    }
    out.insert(RELEASE_MANIFEST_FILE, manifest);
    out.insert(CHECKSUMS_FILE, checksums);
    out
}

// -- Opt-in retrieval --------------------------------------------------------

#[test]
fn catalogue_verifies_what_a_callers_transport_returned() {
    let (bundle, _) = publication();
    let paths = plan(&bundle).expect("the plan derives");
    assert!(!paths.is_empty());

    let transport = FromBundle(bundle.clone());
    // ⭐ Every path in the plan, fetched through the caller's transport and
    // verified here. That is the whole division of labour: the consumer fetches
    // and this decides whether the bytes are the publication's.
    for (path, _, _, digest) in &paths {
        let bytes = retrieve(&transport, path, digest).expect("the publication's own bytes");
        assert_eq!(bytes, bundle.get(path).expect("bytes"));
    }

    // And a transport that answers with something else is refused, per path.
    let liar = FromBundle({
        let mut swapped = Bundle::new();
        for path in bundle.paths() {
            swapped.insert(path, b"different bytes".to_vec());
        }
        swapped
    });
    let (path, _, _, digest) = &paths[0];
    assert!(retrieve(&liar, path, digest).is_err());
}

#[test]
fn catalogue_plan_marks_measurements_immutable_and_derived_files_current() {
    use bit_ids::access::{Integrity, Stability};
    let (bundle, _) = publication();
    let paths = plan(&bundle).expect("the plan derives");
    for (path, stability, integrity, _) in &paths {
        let expected = if path.starts_with("profiles/") || path.starts_with("raw/") {
            Stability::Immutable
        } else {
            Stability::Current
        };
        assert_eq!(*stability, expected, "{path}");
        let expected = match path.as_str() {
            RELEASE_MANIFEST_FILE => Integrity::Checksums,
            CHECKSUMS_FILE => Integrity::OutOfBand,
            _ => Integrity::Manifest,
        };
        assert_eq!(*integrity, expected, "{path}");
    }
}

// -- The documented example, compiled ---------------------------------------

/// The Rust example in `docs/consuming.md`, copied here so something compiles
/// it.
///
/// ⛔ **A documented example nothing compiles is a documented example that
/// stops working silently**, which is the same defect as a shell example nobody
/// runs. `check-examples.sh` runs the shell blocks and checks that this case
/// exists; this is what makes the Rust block real.
///
/// ⚠ It is a copy, and a copy drifts. What keeps it honest is that both are
/// small enough to compare by eye and the harness refuses a document whose Rust
/// block count is zero. Extracting and compiling the block itself would need a
/// build script, which is a dependency this entry does not need.
fn newest(bundle: &Bundle) -> Option<String> {
    let catalogue = Catalogue::open(bundle, None).ok()?;
    let slug = |t: &str| Slug::parse(t).ok();
    let profile = catalogue.latest(
        &slug("fixture-client")?,
        &slug("linux")?,
        &slug("x86-64")?,
        &slug("tar-gz")?,
    )?;
    Some(profile.build.version.as_str().to_owned())
}

#[test]
fn catalogue_documentation_example_compiles() {
    let (bundle, _) = publication();
    // ⚠ The fixture publication is one target on one line, and the example asks
    // for `fixture-client linux x86-64 tar-gz`, which is exactly what
    // `build-store` writes. A `None` here would mean the documented example
    // answers nothing on the publication the harness builds for it.
    let record = Profile::from_json(PROFILE).expect("the fixture record validates");
    let answered = newest(&bundle);
    if record.build.platform.as_str() == "linux" && record.build.package.as_str() == "tar-gz" {
        assert_eq!(answered.as_deref(), Some("1.2.3"));
    } else {
        assert!(answered.is_none());
    }
}

// -- The reader the published index never had --------------------------------

#[test]
fn an_index_document_round_trips_through_its_own_reader() {
    // ⛔ `Indexes::to_json` had no reader until this entry needed one. A
    // published document nothing parses is a format nobody has round-tripped,
    // and the direction that matters is bytes in, bytes out: a reader that
    // dropped a field would still write a document, just not this one.
    let (bundle, _) = publication();
    let document = core::str::from_utf8(bundle.get(INDEX_PATH).expect("index")).expect("utf8");
    let read = Indexes::from_json(document).expect("the published index reads back");
    assert_eq!(read.to_json(), document);
    assert!(read.rows() > 0, "the comparison had rows to compare");
}

#[test]
fn a_release_manifest_round_trips_through_its_own_reader() {
    // ⛔ The second published document that had a writer and no reader. The gap
    // was not cosmetic: without it a consumer compares a bundle against a
    // manifest re-derived from that bundle, which agrees with itself, so
    // `E-LIB-01` and `E-LIB-03` could not fire at all.
    use bit_ids::release::Release;
    let (bundle, _) = publication();
    let document =
        core::str::from_utf8(bundle.get(RELEASE_MANIFEST_FILE).expect("manifest")).expect("utf8");
    let read = Release::from_json(document).expect("the published manifest reads back");
    assert_eq!(read.manifest_json(), document);
    assert!(!read.is_empty(), "the comparison had files to compare");
}

#[test]
fn a_release_manifest_declaring_another_media_type_is_refused() {
    // ⚠ A media type is checked against this build's own table rather than
    // taken. A manifest declaring one this build does not know is a later
    // generation wearing this schema, and believing it hands a consumer bytes
    // with a type nothing checked.
    use bit_ids::release::Release;
    let (bundle, _) = publication();
    let document =
        core::str::from_utf8(bundle.get(RELEASE_MANIFEST_FILE).expect("manifest")).expect("utf8");
    let wrong = document.replace("application/json", "application/x-invented");
    assert_ne!(wrong, document, "the plant applied");
    assert!(Release::from_json(&wrong).is_err());

    let another = document.replace(bit_ids::RELEASE_SCHEMA, "bit-ids/release/2");
    let error = Release::from_json(&another).expect_err("another generation is refused");
    assert!(error.to_string().contains("bit-ids/release/2"), "{error}");
}

#[test]
fn catalogue_lookups_answer_from_the_published_index() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    let record = catalogue.records()[0];

    let by_target = catalogue.lookup(IndexKind::Target, record.target.id.as_str());
    assert_eq!(by_target.len(), 1);
    assert_eq!(by_target[0].id, record.id);

    // ⚠ A key nothing measured answers nothing, rather than answering with
    // whatever sorts nearest.
    assert!(
        catalogue
            .lookup(IndexKind::Target, "a-target-nobody-measured")
            .is_empty()
    );
    // And an identifier with nothing correcting it is its own current answer.
    assert_eq!(catalogue.current(record.id), Some(record.id));
}

// -- ⭐ LIB-02: what a tool asks about its OWN identity -----------------------
//
// `LIB-01` above answers "what did you measure". These answer the different
// question `bit-cli` has, which is whether what it says about itself is what
// this project measured, and the whole design is that not knowing is never a
// pass.
//
// ⚠ The publication is `LIB-01`'s, unchanged. A second helper building a
// second bundle would be a second spelling of a publication, and the shipped
// fixture already carries every state the comparison distinguishes: a patterned
// peer id, two constants, an absence of each kind, an unknown and a variable.

/// The claim `bit-cli` would make, built the way its own `peer_id.rs` builds
/// the bytes.
///
/// ⛔ **Read out of that repository rather than remembered.** `-CL`, then one
/// character per version component from its `ALPHABET`, then the build slot and
/// a dash. At its `0.2.0` that is `-CL0200-`. ⚠ Nothing here resolves `CL`
/// through a client-ID table, and nothing in this repository records that
/// prefix as a measurement: it is a third-party *claim*, which is exactly the
/// side of the comparison a caller supplies.
const BIT_CLI_PREFIX: &[u8] = b"-CL0200-";

fn claim_of(target: &str, version: &str, field: &str, bytes: &[u8], shape: Shape) -> Claim {
    Claim {
        target: slug(target),
        version: Version::parse(version).expect("a version"),
        field: FieldPath::parse(field).expect("a field path"),
        bytes: bytes.to_vec(),
        shape,
    }
}

#[test]
fn adapter_confirms_a_prefix_the_record_measured() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    // The fixture's peer id is patterned with the fixed run `-XX0000-`.
    let answer = check(
        &catalogue,
        &claim_of(
            "fixture-client",
            "1.2.3",
            "peer_wire/peer_id",
            b"-XX0000-",
            Shape::Prefix,
        ),
    );
    assert!(answer.agrees(), "{answer:?}");
}

#[test]
fn adapter_contradicts_a_prefix_the_record_did_not_measure() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    let answer = check(
        &catalogue,
        &claim_of(
            "fixture-client",
            "1.2.3",
            "peer_wire/peer_id",
            BIT_CLI_PREFIX,
            Shape::Prefix,
        ),
    );
    assert!(answer.disagrees(), "{answer:?}");
    assert!(!answer.agrees(), "{answer:?}");
    // ⚠ And it says what WAS measured, so a caller has something to act on
    // rather than a boolean.
    match answer {
        Answer::Disagrees { measured, .. } => assert_eq!(measured, "2d5858303030302d"),
        other => panic!("{other:?}"),
    }
}

#[test]
fn adapter_refuses_a_whole_value_claim_over_a_value_that_varies() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    // ⛔ A disagreement rather than a gap: the tool says the value never
    // changes and the measurement says it does.
    let answer = check(
        &catalogue,
        &claim_of(
            "fixture-client",
            "1.2.3",
            "peer_wire/peer_id",
            b"-XX0000-",
            Shape::Whole,
        ),
    );
    assert!(answer.disagrees(), "{answer:?}");
}

#[test]
fn adapter_compares_a_constant_as_a_whole_value_and_as_a_prefix() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    let whole = check(
        &catalogue,
        &claim_of(
            "fixture-client",
            "1.2.3",
            "peer_wire/bep10.client",
            b"fixture/0.0.0",
            Shape::Whole,
        ),
    );
    assert!(whole.agrees(), "{whole:?}");
    // ⭐ A constant answers a prefix claim too. Refusing to compare here would
    // report a client whose whole value is fixed as unmeasured.
    let prefix = check(
        &catalogue,
        &claim_of(
            "fixture-client",
            "1.2.3",
            "peer_wire/bep10.client",
            b"fixture/",
            Shape::Prefix,
        ),
    );
    assert!(prefix.agrees(), "{prefix:?}");
}

/// ⛔ **Every way of not knowing, and not one of them agrees.** This is the
/// fail-closed rule as a case rather than as a comment, and each reason is
/// asserted by name because the thing to do about each differs.
#[test]
fn adapter_never_agrees_when_there_is_nothing_to_compare() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    let cases: Vec<(&str, &str, &str, Shape, Unmeasured)> = vec![
        (
            "fixture-client",
            "1.2.3",
            "dht/node_id",
            Shape::Prefix,
            Unmeasured::Absent {
                state: "not_observed",
            },
        ),
        (
            "fixture-client",
            "1.2.3",
            "web_seed/user_agent",
            Shape::Prefix,
            Unmeasured::Absent {
                state: "not_supported",
            },
        ),
        (
            "fixture-client",
            "1.2.3",
            "tracker_http/announce.query_order",
            Shape::Prefix,
            Unmeasured::Unknown,
        ),
        (
            "fixture-client",
            "1.2.3",
            "mse/handshake_padding",
            Shape::Prefix,
            Unmeasured::NoFixedPrefix,
        ),
        (
            "fixture-client",
            "1.2.3",
            "peer_wire/handshake.protocol",
            Shape::Prefix,
            Unmeasured::NoField,
        ),
        (
            "fixture-client",
            "9.9.9",
            "peer_wire/peer_id",
            Shape::Prefix,
            Unmeasured::NoRecord,
        ),
    ];
    for (target, version, field, shape, want) in cases {
        let answer = check(
            &catalogue,
            &claim_of(target, version, field, b"-XX0000-", shape),
        );
        assert!(!answer.agrees(), "{field} agreed: {answer:?}");
        assert!(!answer.disagrees(), "{field} disagreed: {answer:?}");
        match answer {
            Answer::Unmeasured { reason } => assert_eq!(reason, want, "{field}"),
            other => panic!("{field}: {other:?}"),
        }
    }
}

/// ⛔ **The honest answer about a real client today, driven end to end.**
/// `bit-cli` declares `-CL0200-` and this project has measured nothing, so the
/// only correct answer is that there is no record - and it is emphatically not
/// agreement. ⚠ No document anywhere in this repository claims to have measured
/// `bit-cli`: the claim is the third-party input and the catalogue is this
/// project's, which is the only arrangement that does not fabricate one.
#[test]
fn adapter_answers_not_measured_for_a_real_client_because_nothing_is_measured() {
    let (bundle, digest) = publication();
    let catalogue = Catalogue::open(&bundle, Some(&digest)).expect("opens");
    let answer = check(
        &catalogue,
        &claim_of(
            "bit-cli",
            "0.2.0",
            "peer_wire/peer_id",
            BIT_CLI_PREFIX,
            Shape::Prefix,
        ),
    );
    assert_eq!(
        answer,
        Answer::Unmeasured {
            reason: Unmeasured::NoRecord
        }
    );
    assert!(!answer.agrees());
}
