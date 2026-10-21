//! Acceptance for `SCHEMA-01`.
//!
//! ```text
//! cargo test --workspace profile_schema
//! ```
//!
//! Every test name carries `profile_schema` so that command selects this file
//! rather than running the workspace and reporting a number about something
//! else.
//!
//! The shape of the suite is deliberate. A golden record proves the schema can
//! express a complete measurement, and a table of planted defects proves each
//! guard can actually refuse one. A guard that has never been seen to refuse is
//! a guard nobody knows works, so the last test reads the validator's own
//! source and fails if the table has stopped covering a code.

use bit_ids::canonical::{Slug, Version};
use bit_ids::identity::{RecordId, RecordKey, SchemaVersion};
use bit_ids::{DocumentError, Profile, Violations};
use serde_json::{Value, json};

const GOLDEN: &str = include_str!("fixtures/valid-profile.json");
const CORRECTION: &str = include_str!("fixtures/valid-correction.json");
const UNSUPPORTED_SCHEMA: &str = include_str!("fixtures/unsupported-schema.json");
const UNPROVEN_FIELD: &str = include_str!("fixtures/unproven-field.json");

/// The validator's own source, read so the coverage test cannot drift from it.
const VALIDATOR_SOURCE: &str = include_str!("../src/validate.rs");

fn golden_value() -> Value {
    serde_json::from_str(GOLDEN).expect("the golden fixture is JSON")
}

/// Feeds a mutated document through the real read path and returns what it
/// refused.
///
/// It goes through `from_json` rather than calling `validate` on a hand-built
/// value, so a defect that serde would catch first is reported as such instead
/// of being credited to an invariant that never ran.
fn refuse(document: &Value) -> Violations {
    let text = serde_json::to_string(document).expect("a mutated document serializes");
    match Profile::from_json(&text) {
        Err(DocumentError::Invalid(violations)) => violations,
        Ok(_) => panic!("the mutated document validated, so the guard did not fire"),
        Err(other) => panic!("expected a refused invariant, got: {other}"),
    }
}

fn field_mut<'a>(document: &'a mut Value, path: &str) -> &'a mut Value {
    document["observations"]
        .as_array_mut()
        .expect("observations is an array")
        .iter_mut()
        .find(|field| field["path"] == path)
        .unwrap_or_else(|| panic!("the golden record observes {path}"))
}

fn corroboration_mut<'a>(document: &'a mut Value, path: &str) -> &'a mut Value {
    document["corroboration"]
        .as_array_mut()
        .expect("corroboration is an array")
        .iter_mut()
        .find(|entry| entry["path"] == path)
        .unwrap_or_else(|| panic!("the golden record corroborates {path}"))
}

fn swap(document: &mut Value, key: &str, left: usize, right: usize) {
    document[key]
        .as_array_mut()
        .expect("the section is an array")
        .swap(left, right);
}

/// A record identifier that is well formed and is not the golden record's, so
/// planting it exercises the correction rules rather than the identity ones.
const CORRECTION_ID: &str =
    "record:sha256:9308390b39915df84619e0b12f7983918539a1497b9061979115748dcecb64b7";

fn adjudication(evidence: &Value) -> Value {
    json!({
        "decided_at": "2026-09-04T14:02:11Z",
        "reason": "parser_defect",
        "summary": "planted",
        "evidence": evidence.clone()
    })
}

// -- the golden records ----------------------------------------------------

#[test]
fn profile_schema_accepts_the_golden_record() {
    let profile = Profile::from_json(GOLDEN).expect("the golden record validates");
    assert_eq!(profile.schema.as_str(), bit_ids::PROFILE_SCHEMA);
    assert_eq!(profile.target.id.as_str(), "fixture-client");
    assert_eq!(profile.observations.len(), 7);
    // Nine since `ACQ-01`: the seven observation artifacts plus the version
    // output each route's installed build printed when it was asked.
    assert_eq!(profile.evidence.len(), 9);
    assert!(profile.supersedes.is_none());
}

#[test]
fn profile_schema_accepts_a_correction_that_names_the_record_it_replaces() {
    let original = Profile::from_json(GOLDEN).expect("the golden record validates");
    let correction = Profile::from_json(CORRECTION).expect("the correction validates");
    assert_eq!(correction.supersedes, Some(original.id));
    assert_ne!(correction.id, original.id, "a correction is its own record");
}

#[test]
fn profile_schema_expresses_every_field_state() {
    let profile = Profile::from_json(GOLDEN).expect("the golden record validates");
    let mut seen: Vec<&str> = profile
        .observations
        .iter()
        .map(|field| field.state.as_str())
        .collect();
    seen.sort_unstable();
    seen.dedup();
    assert_eq!(
        seen,
        [
            "constant",
            "not_observed",
            "not_supported",
            "patterned",
            "unknown",
            "variable"
        ],
        "the golden record has to exercise all six states or it is not a golden record"
    );
}

#[test]
fn profile_schema_writes_the_canonical_form_it_read() {
    let profile = Profile::from_json(GOLDEN).expect("the golden record validates");
    let written = profile
        .to_json()
        .expect("a valid record has a canonical form");
    assert_eq!(
        written, GOLDEN,
        "two assemblies of one record must produce identical bytes"
    );
    let reread = Profile::from_json(&written).expect("the written record reads back");
    assert_eq!(reread, profile);
}

#[test]
fn profile_schema_derives_the_record_id_from_the_identity_tuple() {
    let profile = Profile::from_json(GOLDEN).expect("the golden record validates");
    let derived = RecordId::derive(&RecordKey {
        schema: &SchemaVersion::current(),
        target: &profile.target.id,
        version: &profile.build.version,
        platform: &profile.build.platform,
        arch: &profile.build.arch,
        package: &profile.build.package,
        capture: &profile.capture.id,
    });
    assert_eq!(derived, profile.id);
}

#[test]
fn profile_schema_gives_two_captures_of_one_build_two_record_ids() {
    let original = Profile::from_json(GOLDEN).expect("the golden record validates");
    let correction = Profile::from_json(CORRECTION).expect("the correction validates");
    assert_eq!(original.build, correction.build, "same build");
    assert_ne!(
        original.capture.id, correction.capture.id,
        "different capture run"
    );
    assert_ne!(original.id, correction.id);
}

// -- the schema version is answered before anything else -------------------

#[test]
fn profile_schema_rejects_an_unknown_schema_version() {
    match Profile::from_json(UNSUPPORTED_SCHEMA) {
        Err(DocumentError::UnsupportedSchema { found, expected }) => {
            assert_eq!(found, "bit-ids/profile/2");
            assert_eq!(expected, bit_ids::PROFILE_SCHEMA);
        }
        other => panic!("expected an unsupported schema, got: {other:?}"),
    }
}

#[test]
fn profile_schema_reports_the_version_before_complaining_about_a_field() {
    let mut document = golden_value();
    document["schema"] = json!("bit-ids/profile/9");
    document["a_field_from_a_later_generation"] = json!(true);
    let text = serde_json::to_string(&document).expect("serializes");
    match Profile::from_json(&text) {
        Err(DocumentError::UnsupportedSchema { found, expected }) => {
            assert_eq!(found, "bit-ids/profile/9");
            assert_eq!(expected, bit_ids::PROFILE_SCHEMA);
        }
        other => panic!(
            "a later generation must be told its version is unsupported, not that a field is \
             unknown; got: {other:?}"
        ),
    }
}

// -- the canonical forms, refused during deserialization -------------------

fn malformed(document: &Value) -> String {
    let text = serde_json::to_string(document).expect("serializes");
    match Profile::from_json(&text) {
        Err(DocumentError::Malformed(error)) => error.to_string(),
        other => panic!("expected a malformed document, got: {other:?}"),
    }
}

#[test]
fn profile_schema_rejects_a_key_it_does_not_declare() {
    let mut document = golden_value();
    document["peer_id_prefix"] = json!("-XX0000-");
    let message = malformed(&document);
    assert!(
        message.contains("peer_id_prefix"),
        "the refusal names the undeclared key: {message}"
    );
}

#[test]
fn profile_schema_rejects_a_key_it_does_not_declare_inside_a_field_state() {
    // `deny_unknown_fields` on a nested, adjacently tagged enum is worth its
    // own test: the top-level check passing says nothing about whether serde
    // applied the attribute inside the variant payload.
    let mut document = golden_value();
    let field = field_mut(&mut document, "peer_wire/bep10.client");
    field["state"]["detail"]["confidence"] = json!(0.9);
    let message = malformed(&document);
    assert!(
        message.contains("confidence"),
        "the refusal names the undeclared key: {message}"
    );

    let mut document = golden_value();
    let run =
        &mut field_mut(&mut document, "peer_wire/peer_id")["state"]["detail"]["pattern"]["runs"][1];
    run["detail"]["entropy"] = json!(12);
    let message = malformed(&document);
    assert!(
        message.contains("entropy"),
        "the refusal names the undeclared key: {message}"
    );
}

#[test]
fn profile_schema_validates_on_every_serde_route_not_just_from_json() {
    // A door sweep found this one live: `Profile` derived `Deserialize`, so a
    // caller who reached for serde directly got an unvalidated record while
    // the crate documentation said `from_json` was the only way in.
    let error = serde_json::from_str::<Profile>(UNPROVEN_FIELD)
        .expect_err("the generic serde route must not be the loose one");
    assert!(
        error.to_string().contains("E-OBS-05"),
        "the refusal carries the code: {error}"
    );

    serde_json::from_str::<Profile>(GOLDEN).expect("a valid record still reads through serde");
}

#[test]
fn profile_schema_rejects_uppercase_hex() {
    let mut document = golden_value();
    let field = field_mut(&mut document, "peer_wire/bep10.client");
    field["state"]["detail"]["value"] = json!("666978747572652F302E302E30");
    let message = malformed(&document);
    assert!(
        message.contains("not-lowercase-hex"),
        "one value must have one spelling: {message}"
    );
}

#[test]
fn profile_schema_rejects_a_sample_count_of_zero() {
    let mut document = golden_value();
    let field = field_mut(&mut document, "peer_wire/bep10.client");
    field["state"]["detail"]["samples"] = json!(0);
    let message = malformed(&document);
    assert!(!message.is_empty(), "zero samples is not a measurement");
}

#[test]
fn profile_schema_rejects_a_capture_instant_that_is_not_utc_seconds() {
    for spelling in [
        "2026-09-04 13:31:33Z",
        "2026-09-04T13:31:33+00:00",
        "2026-09-04T13:31:33.500Z",
        "2026-02-30T00:00:00Z",
        "2026-09-04T13:31:60Z",
    ] {
        let mut document = golden_value();
        document["capture"]["captured_at"] = json!(spelling);
        let message = malformed(&document);
        assert!(
            message.contains("instant-"),
            "{spelling} must be refused as an instant, got: {message}"
        );
    }
}

#[test]
fn profile_schema_rejects_an_evidence_path_that_leaves_the_bundle() {
    for path in [
        "../outside.bin",
        "/etc/passwd",
        "observer\\events.jsonl",
        "observer//events.jsonl",
        "./events.jsonl",
    ] {
        let mut document = golden_value();
        document["evidence"][0]["path"] = json!(path);
        let message = malformed(&document);
        assert!(
            message.contains("path-"),
            "{path} must be refused as an evidence path, got: {message}"
        );
    }
}

#[test]
fn profile_schema_rejects_a_record_id_that_is_not_a_record_id() {
    let mut document = golden_value();
    let digest = document["id"]
        .as_str()
        .expect("the id is a string")
        .trim_start_matches("record:")
        .to_owned();
    document["id"] = json!(digest);
    let message = malformed(&document);
    assert!(
        message.contains("record-id"),
        "a content digest is not a record identifier: {message}"
    );
}

// -- the unproven-field rule, which is what this entry exists for ----------

#[test]
fn profile_schema_rejects_an_unproven_field() {
    match Profile::from_json(UNPROVEN_FIELD) {
        Err(DocumentError::Invalid(violations)) => {
            assert!(
                violations.has("E-OBS-05"),
                "a field asserting a measurement with no evidence must be refused: {violations}"
            );
        }
        other => panic!("expected an unproven field, got: {other:?}"),
    }
}

#[test]
fn profile_schema_rejects_an_absence_that_no_control_backs() {
    let mut document = golden_value();
    let field = field_mut(&mut document, "dht/node_id");
    field["evidence"] = json!(["ev-observer-stream"]);
    let violations = refuse(&document);
    assert!(
        violations.has("E-OBS-07"),
        "an observer that was never listening and a build that never answered are not the same \
         record: {violations}"
    );
}

#[test]
fn profile_schema_refuses_to_write_a_record_it_would_refuse_to_read() {
    let mut profile = Profile::from_json(GOLDEN).expect("the golden record validates");
    profile.observations[3].evidence.clear();
    match profile.to_json() {
        Err(DocumentError::Invalid(violations)) => assert!(violations.has("E-OBS-05")),
        other => panic!("an invalid record has no canonical form, got: {other:?}"),
    }
}

// -- a connector never corroborates itself, `CLIENT-05` ----------------------

/// Renames the golden record's independent connector everywhere it is named,
/// keeping the lists the schema orders ascending, so a refusal can only be
/// about the name.
fn rename_connector(document: &mut Value, to: &str) {
    let text = serde_json::to_string(document)
        .expect("serializes")
        .replace("\"packet-oracle\"", &format!("\"{to}\""));
    *document = serde_json::from_str(&text).expect("still JSON");
    document["capture"]["connectors"]
        .as_array_mut()
        .expect("an array")
        .sort_by(|a, b| a["id"].as_str().cmp(&b["id"].as_str()));
    for entry in document["corroboration"].as_array_mut().expect("an array") {
        entry["observations"]
            .as_array_mut()
            .expect("an array")
            .sort_by(|a, b| a["connector"].as_str().cmp(&b["connector"].as_str()));
    }
}

/// Puts another target under test, with the record identifier derived again so
/// `E-ID-01` stays quiet.
fn retarget(document: &mut Value, target: &str, engine: Option<&str>) {
    document["target"]["id"] = json!(target);
    document["target"]["engine"] = json!(engine);
    let slug = |value: &Value| Slug::parse(value.as_str().expect("a string")).expect("a slug");
    let target = slug(&document["target"]["id"]);
    let platform = slug(&document["build"]["platform"]);
    let arch = slug(&document["build"]["arch"]);
    let package = slug(&document["build"]["package"]);
    let capture = slug(&document["capture"]["id"]);
    let version =
        Version::parse(document["build"]["version"].as_str().expect("a string")).expect("version");
    let id = RecordId::derive(&RecordKey {
        schema: &SchemaVersion::current(),
        target: &target,
        version: &version,
        platform: &platform,
        arch: &arch,
        package: &package,
        capture: &capture,
    });
    document["id"] = json!(id.to_string());
}

fn paired(target: &str, engine: Option<&str>, connector: &str) -> Value {
    let mut document = golden_value();
    rename_connector(&mut document, connector);
    retarget(&mut document, target, engine);
    document
}

fn validates(document: &Value) -> Result<(), String> {
    let text = serde_json::to_string(document).expect("serializes");
    Profile::from_json(&text)
        .map(|_| ())
        .map_err(|error| error.to_string())
}

#[test]
fn profile_schema_rejects_aria2_corroborating_aria2() {
    let violations = refuse(&paired("aria2", None, "aria2"));
    assert!(violations.has("E-COR-20"), "{violations}");
    assert_eq!(
        violations.codes().collect::<Vec<_>>(),
        ["E-COR-20"],
        "the pairing is the only defect planted"
    );
}

#[test]
fn profile_schema_accepts_cpython_stdlib_corroborating_aria2() {
    // ⛔ THE CONTROL. Every refusal here passes over a rule that refused all.
    validates(&paired("aria2", None, "cpython-stdlib")).expect("an independent connector");
    validates(&paired("aria2-next", None, "cpython-stdlib")).expect("an independent connector");
}

#[test]
fn profile_schema_rejects_every_spelling_of_aria2_beside_aria2() {
    // Both directions of the fork, the executable's name, and a wrapper.
    for (target, connector) in [
        ("aria2", "aria2-next"),
        ("aria2-next", "aria2"),
        ("aria2", "aria2c"),
        ("aria2", "aria2c-rpc"),
        ("aria2", "libaria2"),
        ("aria2-next", "aria2-next"),
        ("libtorrent", "libtorrent-rasterbar"),
    ] {
        let violations = refuse(&paired(target, None, connector));
        assert!(
            violations.has("E-COR-20"),
            "{connector} beside {target}: {violations}"
        );
    }
}

#[test]
fn profile_schema_rejects_a_connector_that_is_the_targets_engine() {
    let violations = refuse(&paired(
        "qbittorrent",
        Some("libtorrent"),
        "libtorrent-alerts",
    ));
    assert!(violations.has("E-COR-20"), "{violations}");
    validates(&paired("qbittorrent", Some("libtorrent"), "cpython-stdlib"))
        .expect("an engine is only refused when it is the connector");
}

#[test]
fn profile_schema_rejects_a_self_corroborating_observer_too() {
    // ⚠ The observer is a declared connector like any other; a target that is
    // this project's observer would be read by itself.
    let mut document = golden_value();
    retarget(&mut document, "bit-ids-probe", None);
    assert!(refuse(&document).has("E-COR-20"));
}

#[test]
fn profile_schema_answers_the_self_corroboration_test_in_both_directions() {
    use bit_ids::validate::same_implementation;
    assert!(same_implementation("aria2", "aria2"));
    assert!(same_implementation("aria2-next", "aria2"));
    assert!(same_implementation("aria2", "aria2-next"));
    assert!(!same_implementation("cpython-stdlib", "aria2"));
    assert!(!same_implementation("bit-ids-probe", "bittorrent"));
    assert!(!same_implementation("", "aria2"));
    // ⛔ Every target in the catalogue against the two connectors this project
    // ships: none is refused, so the rule costs no capture it should not.
    let catalogue = include_str!("../../../catalogue/clients.toml");
    let targets: Vec<&str> = catalogue
        .lines()
        .filter_map(|line| line.strip_prefix("id = \""))
        .filter_map(|rest| rest.strip_suffix('"'))
        .collect();
    assert!(targets.len() > 10, "read {} targets", targets.len());
    for target in targets {
        for connector in ["bit-ids-probe", "cpython-stdlib"] {
            assert!(
                !same_implementation(connector, target),
                "{connector} would be refused beside {target}"
            );
        }
    }
}

// -- every guard is planted against, and every code is covered -------------

type Mutation = fn(&mut Value);

/// One planted defect per diagnostic code.
///
/// The mutation is applied to the golden record, so anything the guard reports
/// is caused by that one change and nothing else in the document.
const PLANTED: &[(&str, Mutation)] = &[
    ("E-ID-01", |document| {
        document["id"] =
            json!("record:sha256:9308390b39915df84619e0b12f7983918539a1497b9061979115748dcecb64b7");
    }),
    ("E-ID-02", |document| {
        document["supersedes"] = document["id"].clone();
    }),
    ("E-TGT-01", |document| {
        document["target"]["display_name"] = json!("");
    }),
    ("E-TGT-02", |document| {
        document["target"]["engine"] = document["target"]["id"].clone();
    }),
    ("E-ACQ-01", |document| {
        document["acquisition"]
            .as_array_mut()
            .expect("an array")
            .truncate(1);
    }),
    ("E-ACQ-02", |document| {
        document["acquisition"][1]["id"] = document["acquisition"][0]["id"].clone();
    }),
    ("E-ACQ-03", |document| swap(document, "acquisition", 0, 1)),
    ("E-ACQ-04", |document| {
        document["acquisition"][0]["installed_version"] = json!("0.0.1-fixture");
    }),
    // A package-manager route carrying a release-asset identity. It reads as a
    // complete record and names nothing anyone could resolve again.
    ("E-ACQ-05", |document| {
        document["acquisition"][0]["source"] = json!({
            "form": "release_asset",
            "detail": {
                "repository": "fixture/fixture-client",
                "tag": "v0.0.0-fixture",
                "asset": "fixture-client.tar.gz",
            },
        });
    }),
    // An abbreviated commit. `FOUND-02` measured this exact shape passing a
    // pin rule written to refuse floating refs, because it looked like a commit.
    ("E-ACQ-06", |document| {
        document["acquisition"][0]["kind"] = json!("source_build");
        document["acquisition"][0]["source"] = json!({
            "form": "source_commit",
            "detail": {
                "repository": "fixture/fixture-client",
                "commit": "abcdef01",
            },
        });
    }),
    // Two routes, one resolver: the two-route rule satisfied by asking one
    // index twice under two names.
    ("E-ACQ-07", |document| {
        document["acquisition"][1]["resolver"] = document["acquisition"][0]["resolver"].clone();
    }),
    ("E-ACQ-08", |document| {
        document["acquisition"][1]["delivery"] = document["acquisition"][0]["delivery"].clone();
    }),
    ("E-ACQ-09", |document| {
        document["acquisition"][0]["installed_evidence"] = json!("ev-does-not-exist");
    }),
    // The installed version cited against an artifact the build did not print.
    ("E-ACQ-10", |document| {
        document["acquisition"][0]["installed_evidence"] = json!("ev-metainfo");
    }),
    ("E-CAP-05", |document| {
        document["capture"]["observed_route"] = json!("route-nobody-ran");
    }),
    // The record's executable digest naming the install nobody watched, which
    // would publish observations against bytes that were never on the wire.
    ("E-CAP-06", |document| {
        document["acquisition"][0]["installed_executable"] =
            json!("sha256:1111111111111111111111111111111111111111111111111111111111111111");
    }),
    ("E-CAP-01", |document| {
        document["capture"]["connectors"]
            .as_array_mut()
            .expect("an array")
            .truncate(1);
    }),
    ("E-CAP-02", |document| {
        document["capture"]["connectors"][1]["id"] =
            document["capture"]["connectors"][0]["id"].clone();
    }),
    ("E-CAP-03", |document| {
        document["capture"]["connectors"]
            .as_array_mut()
            .expect("an array")
            .swap(0, 1);
    }),
    ("E-CAP-04", |document| {
        document["capture"]["observer"] = json!("a-connector-that-did-not-run");
    }),
    ("E-EVD-01", |document| {
        document["evidence"][1]["id"] = document["evidence"][0]["id"].clone();
    }),
    ("E-EVD-02", |document| swap(document, "evidence", 0, 1)),
    ("E-EVD-03", |document| {
        document["evidence"][1]["path"] = document["evidence"][0]["path"].clone();
    }),
    ("E-EVD-04", |document| {
        document["evidence"][0]["bytes"] = json!(0);
    }),
    ("E-EVD-05", |document| {
        document["evidence"][0]["connector"] = json!("a-connector-that-did-not-run");
    }),
    ("E-OBS-01", |document| {
        document["observations"][1]["path"] = document["observations"][0]["path"].clone();
    }),
    ("E-OBS-02", |document| swap(document, "observations", 0, 1)),
    ("E-OBS-03", |document| {
        field_mut(document, "dht/node_id")["evidence"] = json!(["ev-that-was-never-collected"]);
    }),
    ("E-OBS-04", |document| {
        field_mut(document, "peer_wire/peer_id")["evidence"] =
            json!(["ev-peer-transcript", "ev-peer-transcript"]);
    }),
    ("E-OBS-05", |document| {
        field_mut(document, "peer_wire/peer_id")["evidence"] = json!([]);
    }),
    ("E-OBS-06", |document| {
        field_mut(document, "tracker_http/announce.query_order")["evidence"] =
            json!(["ev-metainfo"]);
    }),
    ("E-OBS-07", |document| {
        field_mut(document, "web_seed/user_agent")["evidence"] = json!(["ev-observer-stream"]);
    }),
    ("E-OBS-08", |document| {
        field_mut(document, "peer_wire/reserved")["state"]["detail"]["value"] =
            json!("00000000001000");
    }),
    ("E-OBS-09", |document| {
        field_mut(document, "peer_wire/peer_id")["state"]["detail"]["pattern"]["length"] =
            json!(21);
    }),
    ("E-OBS-10", |document| {
        let pattern = &mut field_mut(document, "peer_wire/peer_id")["state"]["detail"]["pattern"];
        pattern["runs"] = json!([{
            "kind": "fixed",
            "detail": { "bytes": "2d5858303030302d2d5858303030302d2d585830" }
        }]);
    }),
    ("E-OBS-11", |document| {
        let pattern = &mut field_mut(document, "peer_wire/peer_id")["state"]["detail"]["pattern"];
        pattern["length"] = json!(19);
        pattern["runs"][0]["detail"]["bytes"] = json!("2d58583030302d");
    }),
    ("E-OBS-12", |document| {
        field_mut(document, "mse/handshake_padding")["state"]["detail"]["samples"] = json!(1);
    }),
    ("E-OBS-13", |document| {
        field_mut(document, "mse/handshake_padding")["state"]["detail"]["distinct"] = json!(9);
    }),
    ("E-OBS-14", |document| {
        field_mut(document, "mse/handshake_padding")["state"]["detail"]["distinct"] = json!(1);
    }),
    ("E-OBS-15", |document| {
        field_mut(document, "peer_wire/peer_id")["state"]["detail"]["pattern"]["runs"][1]["detail"]
            ["alphabet"] = json!("39383736353433323130");
    }),
    ("E-OBS-16", |document| {
        document["observations"] = json!([{
            "path": "peer_wire/peer_id",
            "state": { "kind": "unknown" },
            "evidence": []
        }]);
        document["corroboration"] = json!([]);
    }),
    ("E-COR-01", |document| {
        corroboration_mut(document, "dht/node_id")["path"] = json!("dht/announce_token");
    }),
    ("E-COR-02", |document| {
        document["corroboration"][1]["path"] = document["corroboration"][0]["path"].clone();
    }),
    ("E-COR-03", |document| swap(document, "corroboration", 0, 1)),
    ("E-COR-04", |document| {
        corroboration_mut(document, "dht/node_id")["observations"][1]["connector"] =
            json!("a-connector-that-did-not-run");
    }),
    ("E-COR-05", |document| {
        // One connector could not see it, so the pair never overlapped. Calling
        // that agreement is the mistake this whole model exists to refuse.
        corroboration_mut(document, "dht/node_id")["observations"][1]["seen"] =
            json!({ "kind": "out_of_scope" });
    }),
    ("E-COR-06", |document| {
        corroboration_mut(document, "dht/node_id")["observations"][1]["connector"] =
            json!("bit-ids-probe");
    }),
    ("E-COR-07", |document| {
        document["corroboration"]
            .as_array_mut()
            .expect("an array")
            .remove(0);
    }),
    ("E-COR-08", |document| {
        corroboration_mut(document, "dht/node_id")["observations"]
            .as_array_mut()
            .expect("an array")
            .swap(0, 1);
    }),
    ("E-COR-09", |document| {
        corroboration_mut(document, "dht/node_id")["observations"][0]["evidence"] =
            json!("ev-that-was-never-collected");
    }),
    ("E-COR-10", |document| {
        corroboration_mut(document, "peer_wire/bep10.client")["observations"][0]["projection"] =
            json!({ "kind": "normalized", "detail": "a-normalization-nobody-declared" });
    }),
    ("E-COR-11", |document| {
        document["normalizations"][0]["preserves_unknown_bytes"] = json!(false);
    }),
    ("E-COR-12", |document| {
        corroboration_mut(document, "dht/node_id")["agreement"] = json!("not_corroborated");
    }),
    ("E-COR-13", |document| {
        corroboration_mut(document, "peer_wire/reserved")["observations"][1]["seen"] =
            json!({ "kind": "bytes", "detail": "0000000000100006" });
    }),
    ("E-COR-14", |document| {
        let entry = corroboration_mut(document, "peer_wire/reserved");
        entry["agreement"] = json!("disagrees");
        entry["conflict"] = json!("planted, but the observations are identical");
    }),
    ("E-COR-15", |document| {
        corroboration_mut(document, "peer_wire/bep10.client")["agreement"] = json!("exact");
    }),
    ("E-COR-16", |document| {
        corroboration_mut(document, "peer_wire/peer_id")["agreement"] = json!("normalized");
    }),
    ("E-COR-17", |document| {
        let entry = corroboration_mut(document, "peer_wire/reserved");
        entry["observations"][1]["seen"] = json!({ "kind": "bytes", "detail": "0000000000100006" });
        entry["agreement"] = json!("disagrees");
    }),
    ("E-COR-18", |document| {
        corroboration_mut(document, "peer_wire/reserved")["conflict"] =
            json!("nothing is in conflict here");
    }),
    ("E-COR-19", |document| {
        let entry = corroboration_mut(document, "peer_wire/reserved");
        entry["agreement"] = json!("not_corroborated");
        for index in 0..2 {
            entry["observations"][index]["seen"] = json!({ "kind": "out_of_scope" });
        }
    }),
    // The independent connector renamed after the target it corroborates.
    ("E-COR-20", |document| {
        rename_connector(document, "fixture-client");
    }),
    ("E-NRM-01", |document| {
        let first = document["normalizations"][0].clone();
        document["normalizations"]
            .as_array_mut()
            .expect("an array")
            .push(first);
    }),
    ("E-NRM-02", |document| {
        document["normalizations"]
            .as_array_mut()
            .expect("an array")
            .push(json!({
                "id": "zz-never-applied",
                "summary": "declared and used by nothing",
                "preserves_order": true,
                "preserves_unknown_bytes": true
            }));
    }),
    ("E-ADJ-01", |document| {
        document["supersedes"] = json!(CORRECTION_ID);
    }),
    ("E-ADJ-02", |document| {
        document["adjudication"] = adjudication(&json!(["ev-packet-capture"]));
    }),
    ("E-ADJ-03", |document| {
        document["supersedes"] = json!(CORRECTION_ID);
        document["adjudication"] = adjudication(&json!([]));
    }),
    ("E-ADJ-04", |document| {
        document["supersedes"] = json!(CORRECTION_ID);
        document["adjudication"] = adjudication(&json!(["ev-that-was-never-collected"]));
    }),
    ("E-ADJ-05", |document| {
        document["supersedes"] = json!(CORRECTION_ID);
        document["adjudication"] = adjudication(&json!(["ev-packet-capture", "ev-packet-capture"]));
    }),
];

#[test]
fn profile_schema_refuses_every_planted_defect() {
    for (code, plant) in PLANTED {
        let mut document = golden_value();
        plant(&mut document);
        let violations = refuse(&document);
        assert!(
            violations.has(code),
            "planting the defect for {code} produced {violations}"
        );
    }
}

#[test]
fn profile_schema_accepts_the_golden_record_the_defects_are_planted_in() {
    // The table above is only evidence if the record it mutates is otherwise
    // clean. A golden record that already failed would make every row pass for
    // the wrong reason.
    let document = golden_value();
    let text = serde_json::to_string(&document).expect("serializes");
    assert!(Profile::from_json(&text).is_ok());
}

#[test]
fn profile_schema_plants_a_defect_for_every_diagnostic_code() {
    let mut declared: Vec<&str> = VALIDATOR_SOURCE
        .match_indices("\"E-")
        .filter_map(|(start, _)| {
            let rest = &VALIDATOR_SOURCE[start + 1..];
            let end = rest.find('"')?;
            Some(&rest[..end])
        })
        .collect();
    declared.sort_unstable();
    declared.dedup();
    assert!(
        declared.len() > 30,
        "the code scan found only {} codes, so it is not reading the validator",
        declared.len()
    );

    let mut planted: Vec<&str> = PLANTED.iter().map(|(code, _)| *code).collect();
    planted.sort_unstable();
    planted.dedup();
    assert_eq!(
        planted.len(),
        PLANTED.len(),
        "the table plants a defect for one code twice"
    );

    let uncovered: Vec<&&str> = declared
        .iter()
        .filter(|code| !planted.contains(code))
        .collect();
    assert!(
        uncovered.is_empty(),
        "these diagnostics have never been seen to refuse anything: {uncovered:?}"
    );
    let unknown: Vec<&&str> = planted
        .iter()
        .filter(|code| !declared.contains(code))
        .collect();
    assert!(
        unknown.is_empty(),
        "these planted codes are not in the validator: {unknown:?}"
    );
}
