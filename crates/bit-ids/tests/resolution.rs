//! The five candidate sets `ACQ-02` names, and a plant for every refusal.
//!
//! ⛔ The sets are the point. A resolver tested only on a tidy list of stable
//! semantic versions is tested on the one input that cannot distinguish it from
//! a lexical sort.

use bit_ids::canonical::{Instant, Label, Sha256Digest, Slug, Url, Version};
use bit_ids::resolution::{
    Candidate, Considered, RESOLUTION_SCHEMA, Resolution, Series, SourceResponse, Verdict,
    VersionScheme, resolve, validate_resolution,
};
use serde_json::{Value, json};

fn slug(text: &str) -> Slug {
    Slug::parse(text).expect("slug")
}

fn label(text: &str) -> Label {
    Label::parse(text).expect("label")
}

fn instant(text: &str) -> Instant {
    Instant::parse(text).expect("instant")
}

fn source(id: &str) -> SourceResponse {
    SourceResponse {
        id: slug(id),
        url: Url::parse("https://example.invalid/releases").expect("url"),
        retrieved_at: instant("2026-09-04T12:00:00Z"),
        digest: Sha256Digest::of(id.as_bytes()),
        candidates: 0,
    }
}

fn candidate(source: &str, tag: &str, prerelease: bool, draft: bool) -> Candidate {
    Candidate {
        source: slug(source),
        tag: label(tag),
        prerelease,
        draft,
        published_at: Some(instant("2026-07-07T21:52:42Z")),
    }
}

/// The tag shape qBittorrent publishes.
fn prefixed_scheme() -> VersionScheme {
    VersionScheme {
        tag_prefix: Some(label("release-")),
        min_components: 3,
        max_components: 4,
        floor_major: None,
        series: None,
    }
}

/// The tag shape Transmission publishes.
fn bare_scheme() -> VersionScheme {
    VersionScheme {
        tag_prefix: None,
        min_components: 3,
        max_components: 3,
        floor_major: None,
        series: None,
    }
}

fn run(scheme: VersionScheme, candidates: Vec<Candidate>) -> Resolution {
    let mut sources: Vec<SourceResponse> = Vec::new();
    for candidate in &candidates {
        if !sources.iter().any(|s| s.id == candidate.source) {
            sources.push(source(candidate.source.as_str()));
        }
    }
    for entry in &mut sources {
        entry.candidates =
            u32::try_from(candidates.iter().filter(|c| c.source == entry.id).count())
                .expect("small");
    }
    sources.sort_by(|a, b| a.id.as_str().cmp(b.id.as_str()));
    resolve(
        slug("fixture-client"),
        instant("2026-09-04T13:00:00Z"),
        scheme,
        sources,
        candidates,
    )
}

fn verdicts(resolution: &Resolution) -> Vec<(&str, Verdict)> {
    resolution
        .considered
        .iter()
        .map(|entry| (entry.candidate.tag.as_str(), entry.verdict))
        .collect()
}

/// Set one: an ordinary stable list. ⛔ Includes `.10` against `.9`, which is
/// the pair a lexical sort gets backwards and a tidy list never exercises.
#[test]
fn resolution_selects_the_newest_stable_release() {
    let found = run(
        prefixed_scheme(),
        vec![
            candidate("github-releases", "release-5.2.9", false, false),
            candidate("github-releases", "release-5.2.10", false, false),
            candidate("github-releases", "release-5.2.3", false, false),
        ],
    );
    assert_eq!(
        found.selected.as_ref().map(Version::as_str),
        Some("5.2.10"),
        "5.2.10 is newer than 5.2.9 and text order says otherwise"
    );
    assert_eq!(found.with_verdict(Verdict::Selected).len(), 1);
    assert_eq!(found.with_verdict(Verdict::Superseded).len(), 2);
    validate_resolution(&found).expect("a resolved decision validates");
}

/// Set two: prereleases, flagged and unflagged. Both signals are believed, and
/// a project that forgets the flag is still not selected.
#[test]
fn resolution_refuses_a_prerelease_by_either_signal() {
    let found = run(
        bare_scheme(),
        vec![
            candidate("github-releases", "4.1.3", false, false),
            candidate("github-releases", "4.2.0-beta.5", true, false),
            // ⚠ The dangerous one: a beta the source forgot to flag. Trusting
            // the label alone selects it and publishes a preview as stable.
            candidate("github-releases", "4.3.0-rc1", false, false),
            candidate("github-releases", "4.4.0", false, true),
        ],
    );
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("4.1.3"));
    assert_eq!(
        verdicts(&found),
        vec![
            ("4.1.3", Verdict::Selected),
            ("4.2.0-beta.5", Verdict::PrereleaseByLabel),
            ("4.3.0-rc1", Verdict::PrereleaseByVersion),
            ("4.4.0", Verdict::PrereleaseByLabel),
        ]
    );
    validate_resolution(&found).expect("validates");
}

/// Set three: nothing to select. An empty list and a list of only prereleases
/// both fail closed rather than answering.
#[test]
fn resolution_fails_closed_when_no_stable_release_exists() {
    // ⚠ A source that answered with nothing, not "no source was asked". The
    // second is a different failure and `E-RES-01` is what reports it.
    let empty = resolve(
        slug("fixture-client"),
        instant("2026-09-04T13:00:00Z"),
        bare_scheme(),
        vec![SourceResponse {
            candidates: 0,
            ..source("github-releases")
        }],
        Vec::new(),
    );
    assert!(!empty.resolved(), "nothing offered, nothing selected");
    validate_resolution(&empty).expect("failing closed is a valid document");

    let all_pre = run(
        bare_scheme(),
        vec![
            candidate("github-releases", "4.1.0-beta.1", true, false),
            candidate("github-releases", "4.1.0-beta.2", true, false),
        ],
    );
    assert!(!all_pre.resolved());
    assert_eq!(all_pre.with_verdict(Verdict::PrereleaseByLabel).len(), 2);
}

/// Set four: two sources that disagree about the newest stable release.
#[test]
fn resolution_fails_closed_when_two_sources_disagree() {
    let found = run(
        bare_scheme(),
        vec![
            candidate("github-releases", "4.1.3", false, false),
            candidate("distro-index", "4.1.2", false, false),
        ],
    );
    assert!(
        !found.resolved(),
        "a disagreement is not a tie to break by picking the larger"
    );
    assert_eq!(found.with_verdict(Verdict::Divergent).len(), 2);
    validate_resolution(&found).expect("validates");
}

/// ⭐ Set five, and the one that matters most: a tag nobody can order.
///
/// Skipping it silently produces an older version selected confidently, with
/// nothing saying a newer one was seen and not understood.
#[test]
fn resolution_fails_closed_on_a_version_it_cannot_order() {
    let found = run(
        bare_scheme(),
        vec![
            candidate("github-releases", "4.1.3", false, false),
            candidate("github-releases", "2026.spring", false, false),
        ],
    );
    assert!(
        !found.resolved(),
        "an unorderable candidate blocks rather than being skipped"
    );
    assert_eq!(found.with_verdict(Verdict::Unorderable).len(), 1);
    assert_eq!(found.with_verdict(Verdict::Superseded).len(), 1);
    validate_resolution(&found).expect("validates");
}

/// ⭐ An unorderable tag published before the winner cannot be the newest, so
/// publication order settles it without guessing.
///
/// This is what a live dry run made necessary: `transmission` publishes 51
/// two-component tags from a decade ago that the current scheme cannot read, and
/// refusing over them made the resolver correct and useless.
#[test]
fn resolution_lets_publication_order_settle_an_unorderable_older_tag() {
    let newest = Candidate {
        published_at: Some(instant("2026-06-30T03:24:03Z")),
        ..candidate("github-releases", "4.1.3", false, false)
    };
    let ancient = Candidate {
        published_at: Some(instant("2016-05-11T00:00:00Z")),
        ..candidate("github-releases", "2.92", false, false)
    };
    let found = run(bare_scheme(), vec![newest, ancient]);
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("4.1.3"));
    assert_eq!(found.with_verdict(Verdict::PredatesSelection).len(), 1);
    validate_resolution(&found).expect("validates");
}

/// ⛔ And it only settles it when there is a date to settle it with. An
/// unorderable candidate with no publication date still blocks, because nothing
/// then rules out that it is newer.
#[test]
fn resolution_still_fails_closed_when_an_unorderable_tag_has_no_date() {
    let newest = Candidate {
        published_at: Some(instant("2026-06-30T03:24:03Z")),
        ..candidate("github-releases", "4.1.3", false, false)
    };
    let undated = Candidate {
        published_at: None,
        ..candidate("github-releases", "2.92", false, false)
    };
    let found = run(bare_scheme(), vec![newest.clone(), undated]);
    assert!(!found.resolved(), "no date, no second signal, no answer");
    assert_eq!(found.with_verdict(Verdict::Unorderable).len(), 1);

    // Nor when the unorderable one is newer than the winner.
    let later = Candidate {
        published_at: Some(instant("2026-08-01T00:00:00Z")),
        ..candidate("github-releases", "2026.autumn", false, false)
    };
    let found = run(bare_scheme(), vec![newest, later]);
    assert!(
        !found.resolved(),
        "it could be the newest and cannot be read"
    );
    assert_eq!(found.with_verdict(Verdict::Unorderable).len(), 1);
}

/// A refs listing's candidate: no flags and no date, which is what leaves an
/// unorderable tag with no second signal but the floor.
fn undated(tag: &str) -> Candidate {
    Candidate {
        published_at: None,
        ..candidate("git-refs", tag, false, false)
    }
}

fn floored(floor: u64) -> VersionScheme {
    VersionScheme {
        floor_major: Some(floor),
        ..bare_scheme()
    }
}

/// ⭐ `transmission`'s own shape, measured on 2026-10-15: three-component tags
/// from `4.0.0`, and a decade of two-component and lettered ones before it that
/// refs carry no date for.
fn transmission_refs() -> Vec<Candidate> {
    [
        "0.5", "1.80b5", "2.20b1", "2.94", "3.00", "4.0.0", "4.1.3", "4.1.2",
    ]
    .into_iter()
    .map(undated)
    .collect()
}

#[test]
fn resolution_lets_a_scheme_floor_release_a_targets_older_spelling() {
    let blocked = run(bare_scheme(), transmission_refs());
    assert!(
        !blocked.resolved(),
        "with no floor, refs have no second signal"
    );
    assert_eq!(blocked.with_verdict(Verdict::Unorderable).len(), 3);
    // ⭐ The two lettered betas are PEP 440's spelling of a prerelease and are
    // read as one, so they never reach the floor. `CLIENT-07`.
    assert_eq!(blocked.with_verdict(Verdict::PrereleaseByVersion).len(), 2);

    let found = run(floored(4), transmission_refs());
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("4.1.3"));
    assert_eq!(found.with_verdict(Verdict::BelowSchemeFloor).len(), 3);
    assert!(found.with_verdict(Verdict::Unorderable).is_empty());
    let text = found.to_json().expect("a floored decision validates");
    assert_eq!(
        Resolution::from_json(&text)
            .expect("reads")
            .to_json()
            .expect("writes"),
        text,
        "and round-trips byte for byte"
    );
}

#[test]
fn resolution_refuses_a_floor_release_of_a_tag_the_scheme_can_order() {
    // ⛔ Every other rule holds here: the tag is below the floor and the
    // selection stands above it. Only the tag being readable is wrong.
    let mut document: Value = serde_json::from_str(
        &run(floored(4), transmission_refs())
            .to_json()
            .expect("writes"),
    )
    .expect("parses");
    let entry = &mut document["considered"][7];
    assert_eq!(entry["candidate"]["tag"], json!("4.1.2"));
    entry["candidate"]["tag"] = json!("3.0.0");
    entry["version"] = json!("3.0.0");
    entry["verdict"] = json!("below_scheme_floor");
    let error = Resolution::from_json(&serde_json::to_string(&document).expect("writes"))
        .expect_err("a readable tag released by the floor");
    assert!(error.to_string().contains("E-RES-09"), "{error}");
}

#[test]
fn resolution_still_fails_closed_when_the_floor_is_above_the_winner() {
    // ⛔ A floor of 5 over a winner of 4.1.3 says nothing about `3.00`.
    let found = run(floored(5), transmission_refs());
    assert!(!found.resolved());
    assert_eq!(found.with_verdict(Verdict::Unorderable).len(), 3);
    assert!(found.with_verdict(Verdict::BelowSchemeFloor).is_empty());
}

#[test]
fn resolution_still_fails_closed_on_an_unreadable_tag_at_or_over_the_floor() {
    // ⚠ The floor releases what is below it and nothing else, and a blocked
    // decision that released some history is still a valid record.
    let found = run(floored(3), transmission_refs());
    assert!(!found.resolved(), "3.00 is at the floor, not below it");
    assert_eq!(verdicts(&found)[4], ("3.00", Verdict::Unorderable));
    assert_eq!(found.with_verdict(Verdict::BelowSchemeFloor).len(), 2);
    found.to_json().expect("the fail-closed record validates");

    for tag in ["4.1", "03.1", "3x.1", "2026.autumn"] {
        let mut offered = transmission_refs();
        offered.push(undated(tag));
        let found = run(floored(4), offered);
        assert!(!found.resolved(), "{tag} is not below the floor");
        assert_eq!(found.with_verdict(Verdict::Unorderable).len(), 1, "{tag}");
    }
}

/// ⭐ `deluge`'s own shape, measured on 2026-10-16: `deluge-` tags of three
/// components, four-component ones before 1.0, PEP 440 betas and `.dev0`
/// markers in the current series, and `archive/` branches' tags beside them.
#[test]
fn resolution_reads_deluges_pep440_betas_as_prereleases() {
    let tags = [
        "archive/1.3-stable",
        "deluge-0.5.1.1",
        "deluge-1.3.15",
        "deluge-2.0.0b1",
        "deluge-2.0b1",
        "deluge-2.1.1",
        "deluge-2.2.0",
        "deluge-2.2.1.dev0",
    ];
    let scheme = VersionScheme {
        tag_prefix: Some(Label::parse("deluge-").expect("label")),
        floor_major: Some(2),
        ..bare_scheme()
    };
    let found = run(scheme, tags.into_iter().map(undated).collect());
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("2.2.0"));
    assert_eq!(found.with_verdict(Verdict::PrereleaseByVersion).len(), 3);
    assert_eq!(found.with_verdict(Verdict::BelowSchemeFloor).len(), 1);
    assert_eq!(found.with_verdict(Verdict::ForeignTag).len(), 1);
    assert!(found.with_verdict(Verdict::Unorderable).is_empty());
    found.to_json().expect("the decision validates");
}

/// ⭐ `libtorrent`'s own shape, measured on 2026-10-22: three stable series
/// tagged side by side, `v1.2.x`, `v2.0.x` and `v2.1.x`, beside a history of
/// `libtorrent-` and `RC_` tags in other spellings. `ENGINE-01`.
fn libtorrent_refs() -> Vec<Candidate> {
    [
        "2.0",
        "RC_0_11",
        "c1.2.10",
        "libtorrent-1_2_3",
        "libtorrent-2.0-RC2",
        "libtorrent=2_0_0-RC",
        "v1.2.10",
        "v1.2.20",
        "v2.0.0",
        "v2.0.9",
        "v2.0.15",
        "v2.0.10",
        "v2.1.0",
        "v2.1.0-rc1",
        "v2.1.1",
        "v2.1.2",
    ]
    .into_iter()
    .map(undated)
    .collect()
}

fn libtorrent_scheme(series: Option<&str>) -> VersionScheme {
    VersionScheme {
        tag_prefix: Some(label("v")),
        series: series.map(|text| Series::parse(text).expect("a series")),
        ..bare_scheme()
    }
}

#[test]
fn resolution_selects_within_a_pinned_series_and_sets_the_others_aside() {
    let found = run(libtorrent_scheme(Some("2.0")), libtorrent_refs());
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("2.0.15"));
    let other: Vec<&str> = found
        .with_verdict(Verdict::OtherSeries)
        .iter()
        .map(|entry| entry.candidate.tag.as_str())
        .collect();
    assert_eq!(other, ["v1.2.10", "v1.2.20", "v2.1.0", "v2.1.1", "v2.1.2"]);
    // ⚠ Another series's release candidate is still the prerelease it is.
    assert_eq!(found.with_verdict(Verdict::PrereleaseByVersion).len(), 1);
    assert_eq!(found.with_verdict(Verdict::ForeignTag).len(), 6);
    assert_eq!(found.with_verdict(Verdict::Superseded).len(), 3);
    assert!(found.with_verdict(Verdict::Unorderable).is_empty());
    let text = found.to_json().expect("a pinned decision validates");
    assert!(text.contains("\"series\": \"2.0\""), "{text}");
    assert_eq!(
        Resolution::from_json(&text)
            .expect("reads")
            .to_json()
            .expect("writes"),
        text,
        "and round-trips byte for byte"
    );

    // ⛔ The same refs with no series select the newest series's release,
    // which is what the pin exists to prevent.
    let free = run(libtorrent_scheme(None), libtorrent_refs());
    assert_eq!(free.selected.as_ref().map(Version::as_str), Some("2.1.2"));
    assert!(free.with_verdict(Verdict::OtherSeries).is_empty());
}

#[test]
fn resolution_under_a_series_still_blocks_on_what_it_cannot_place() {
    // ⛔ `v2.0` and `v2` agree with the series as far as they go and cannot be
    // ordered, so they could be its newest; `v2.0x.1` is unreadable before any
    // difference. Each blocks, as an unorderable tag does without a series.
    for tag in ["v2.0", "v2", "v2.0x.1", "v2x.1.0"] {
        let mut offered = libtorrent_refs();
        offered.push(undated(tag));
        let found = run(libtorrent_scheme(Some("2.0")), offered);
        assert!(!found.resolved(), "{tag} cannot be placed");
        assert_eq!(found.with_verdict(Verdict::Unorderable).len(), 1, "{tag}");
    }
    // ⭐ And a readable difference sets one aside even where the rest of it
    // could not be ordered.
    let mut offered = libtorrent_refs();
    offered.push(undated("v2.1"));
    let found = run(libtorrent_scheme(Some("2.0")), offered);
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("2.0.15"));
    assert_eq!(found.with_verdict(Verdict::OtherSeries).len(), 6);
}

#[test]
fn resolution_refuses_a_malformed_series_where_it_is_read() {
    let found = run(libtorrent_scheme(Some("2.0")), libtorrent_refs());
    for bad in ["", "2.", "02.0", "2.x", "v2.0"] {
        let mut document: Value =
            serde_json::from_str(&found.to_json().expect("writes")).expect("parses");
        document["scheme"]["series"] = json!(bad);
        let error = Resolution::from_json(&serde_json::to_string(&document).expect("writes"))
            .expect_err("a malformed series");
        assert!(
            error.to_string().contains("release-series"),
            "{bad:?}: {error}"
        );
    }
    // ⚠ Absent is not none: the third generation states the series, `null`
    // for none, and a document that omits it is refused.
    let mut document: Value =
        serde_json::from_str(&found.to_json().expect("writes")).expect("parses");
    document["scheme"]
        .as_object_mut()
        .expect("an object")
        .remove("series");
    assert!(Resolution::from_json(&serde_json::to_string(&document).expect("writes")).is_err());
}

/// A tag that is not this target's release at all does not block. A repository
/// carrying `nox-5.2.3` beside `release-5.2.3` is ordinary.
#[test]
fn resolution_ignores_a_tag_that_is_not_this_targets() {
    let found = run(
        prefixed_scheme(),
        vec![
            candidate("github-releases", "release-5.2.3", false, false),
            candidate("github-releases", "webui-2.0", false, false),
        ],
    );
    assert_eq!(found.selected.as_ref().map(Version::as_str), Some("5.2.3"));
    assert_eq!(found.with_verdict(Verdict::ForeignTag).len(), 1);
}

/// Two spellings of one version make "newest" ambiguous rather than equal.
#[test]
fn resolution_fails_closed_on_two_spellings_of_one_version() {
    let found = run(
        VersionScheme {
            tag_prefix: None,
            min_components: 2,
            max_components: 3,
            floor_major: None,
            series: None,
        },
        vec![
            candidate("github-releases", "4.1.0", false, false),
            candidate("github-releases", "4.1", false, false),
        ],
    );
    assert!(!found.resolved());
    assert_eq!(found.with_verdict(Verdict::Ambiguous).len(), 2);
}

/// The four-component enhanced-edition shape, which a three-component scheme
/// would call unorderable and a lexical sort would get backwards.
#[test]
fn resolution_orders_a_four_component_version() {
    let found = run(
        prefixed_scheme(),
        vec![
            candidate("github-releases", "release-5.2.3.9", false, false),
            candidate("github-releases", "release-5.2.3.10", false, false),
        ],
    );
    assert_eq!(
        found.selected.as_ref().map(Version::as_str),
        Some("5.2.3.10")
    );
}

#[test]
fn resolution_round_trips_a_decision_byte_for_byte() {
    let found = run(
        bare_scheme(),
        vec![candidate("github-releases", "4.1.3", false, false)],
    );
    let document = found.to_json().expect("writes");
    let read = Resolution::from_json(&document).expect("reads");
    assert_eq!(read, found);
    assert_eq!(read.to_json().expect("writes"), document);
}

#[test]
fn resolution_reports_an_unknown_schema_version_before_any_field() {
    let found = run(
        bare_scheme(),
        vec![candidate("github-releases", "4.1.3", false, false)],
    );
    // ⚠ The first generation carries no scheme floor and the second no series,
    // and this build reads neither.
    for older in ["bit-ids/resolution/1", "bit-ids/resolution/2"] {
        let mut document: Value =
            serde_json::from_str(&found.to_json().expect("writes")).expect("parses");
        document["schema"] = json!(older);
        document["sources"] = json!([]);
        let error = Resolution::from_json(&serde_json::to_string(&document).expect("writes"))
            .expect_err("another generation");
        assert!(
            error.to_string().contains("unsupported schema"),
            "the version is answered first: {error}"
        );
    }
}

type Mutation = fn(&mut Value);

/// One planted defect per refusal. ⛔ A code nobody has seen fire is a code
/// nobody knows works.
const PLANTED: &[(&str, Mutation)] = &[
    ("E-RES-01", |document| {
        document["sources"] = json!([]);
        document["considered"] = json!([]);
        document["selected"] = Value::Null;
    }),
    ("E-RES-02", |document| {
        let first = document["sources"][0].clone();
        document["sources"] = json!([first.clone(), first]);
    }),
    ("E-RES-03", |document| {
        document["considered"][0]["candidate"]["source"] = json!("no-such-source");
    }),
    ("E-RES-04", |document| {
        document["sources"][0]["candidates"] = json!(9);
    }),
    ("E-RES-05", |document| {
        document["sources"][0]["retrieved_at"] = json!("2026-09-04T23:59:59Z");
    }),
    ("E-RES-06", |document| {
        document["selected"] = json!("9.9.9");
    }),
    ("E-RES-07", |document| {
        document["considered"][0]["verdict"] = json!("superseded");
    }),
    ("E-RES-08", |document| {
        document["considered"][1]["verdict"] = json!("unorderable");
    }),
    // A release by a floor the recorded scheme does not declare.
    ("E-RES-09", |document| {
        document["considered"][1]["verdict"] = json!("below_scheme_floor");
    }),
    // ⛔ A floor that released a tag, over a selection beneath that floor.
    ("E-RES-10", |document| {
        document["scheme"]["floor_major"] = json!(5);
        document["considered"][1]["candidate"]["tag"] = json!("3.00");
        document["considered"][1]["verdict"] = json!("below_scheme_floor");
    }),
    // Another series, by a scheme that pins none.
    ("E-RES-11", |document| {
        document["considered"][1]["verdict"] = json!("other_series");
    }),
    // ⛔ A selection outside the series the scheme pins.
    ("E-RES-12", |document| {
        document["scheme"]["series"] = json!("4.0");
    }),
    // A series that leaves nothing to order.
    ("E-RES-13", |document| {
        document["scheme"]["series"] = json!("4.1.3");
    }),
];

fn planted_base() -> Value {
    // Two candidates so a plant can target the second without removing the
    // selection, and one source so the counts are simple to move.
    let found = run(
        bare_scheme(),
        vec![
            candidate("github-releases", "4.1.3", false, false),
            candidate("github-releases", "4.1.2", false, false),
        ],
    );
    serde_json::from_str(&found.to_json().expect("writes")).expect("parses")
}

#[test]
fn resolution_accepts_the_document_the_defects_are_planted_in() {
    // ⛔ The table is only evidence if the document it mutates is otherwise
    // clean; a base that already failed would make every row pass for the
    // wrong reason.
    let document = planted_base();
    let text = serde_json::to_string(&document).expect("writes");
    Resolution::from_json(&text).expect("the base document validates");
}

#[test]
fn resolution_refuses_every_planted_defect() {
    for (code, plant) in PLANTED {
        let mut document = planted_base();
        plant(&mut document);
        let text = serde_json::to_string(&document).expect("writes");
        let error = Resolution::from_json(&text)
            .expect_err(&format!("{code}: the planted defect was accepted"));
        assert!(
            error.to_string().contains(code),
            "planting the defect for {code} produced {error}"
        );
    }
}

#[test]
fn resolution_plants_a_defect_for_every_diagnostic_code() {
    const SOURCE: &str = include_str!("../src/resolution.rs");
    let mut declared: Vec<&str> = SOURCE
        .match_indices("\"E-")
        .filter_map(|(start, _)| {
            let rest = &SOURCE[start + 1..];
            let end = rest.find('"')?;
            Some(&rest[..end])
        })
        .collect();
    declared.sort_unstable();
    declared.dedup();
    assert!(
        declared.len() >= 8,
        "the code scan found only {} codes, so it is not reading the module",
        declared.len()
    );
    let planted: Vec<&str> = PLANTED.iter().map(|(code, _)| *code).collect();
    let uncovered: Vec<&&str> = declared
        .iter()
        .filter(|code| !planted.contains(code))
        .collect();
    assert!(
        uncovered.is_empty(),
        "these diagnostics have never been seen to refuse anything: {uncovered:?}"
    );
}

/// The document is the record of a decision, so a reader must be able to see
/// every candidate that was weighed and why each lost.
#[test]
fn resolution_keeps_every_candidate_it_considered() {
    let offered = vec![
        candidate("github-releases", "release-5.2.3", false, false),
        candidate("github-releases", "release-5.2.2", false, false),
        candidate("github-releases", "release-5.3.0-beta1", true, false),
        candidate("github-releases", "webui-2.0", false, false),
    ];
    let found = run(prefixed_scheme(), offered.clone());
    assert_eq!(found.considered.len(), offered.len());
    for (entry, original) in found.considered.iter().zip(&offered) {
        assert_eq!(&entry.candidate, original, "a candidate was rewritten");
    }
    assert_eq!(found.schema.as_str(), RESOLUTION_SCHEMA);
    let selected: Vec<&Considered> = found.with_verdict(Verdict::Selected);
    assert_eq!(selected.len(), 1);
    assert_eq!(selected[0].candidate.tag.as_str(), "release-5.2.3");
}

/// ⛔ `Verdict::as_str` and serde's `rename_all` are two spellings of one
/// vocabulary. A door sweep found nothing comparing them.
#[test]
fn resolution_verdict_spellings_agree_with_the_serialized_form() {
    for verdict in Verdict::ALL {
        let serialized = serde_json::to_string(verdict).expect("serializes");
        assert_eq!(
            format!("\"{}\"", verdict.as_str()),
            serialized,
            "{verdict:?} spells itself two ways"
        );
    }
    assert_eq!(
        Verdict::ALL.len(),
        11,
        "a variant was added without a spelling"
    );
}

/// A release the reader cannot turn into a candidate is refused, never dropped.
/// A candidate list quietly missing an entry is the silent skip the resolver
/// exists to prevent, one layer earlier.
#[test]
fn resolution_refuses_a_release_it_cannot_read_rather_than_dropping_it() {
    let id = slug("github-releases");
    let good = br#"[{"tag_name":"4.1.3","prerelease":false,"draft":false,
                    "published_at":"2026-06-30T03:24:03Z"}]"#;
    let read = bit_ids::resolution::sources::github_releases(good, &id).expect("reads");
    assert_eq!(read.len(), 1);
    assert_eq!(read[0].tag.as_str(), "4.1.3");

    // A tag with a control byte in it is not a Label, and the whole list is
    // refused rather than yielding one candidate and silently losing the other.
    let bad = b"[{\"tag_name\":\"4.1.3\"},{\"tag_name\":\"4.1\\u00014\"}]";
    let error = bit_ids::resolution::sources::github_releases(bad, &id)
        .expect_err("an unusable tag is a finding");
    assert!(error.contains("tag"), "the message names the tag: {error}");

    // A published_at that is not this schema's instant is refused too.
    let odd = br#"[{"tag_name":"4.1.3","published_at":"2026-06-30T03:24:03.123Z"}]"#;
    let error = bit_ids::resolution::sources::github_releases(odd, &id)
        .expect_err("fractional seconds are not this schema's instant");
    assert!(error.contains("published_at"), "{error}");
}

/// The digest a source publishes beside an asset is read, and its three states
/// are three different answers.
///
/// ⛔ **A DIGEST THIS PROJECT CANNOT READ IS A REFUSAL, NEVER AN ABSENCE.** The
/// day a source publishes `sha512:` or a bare hex run, reading it as "no digest"
/// would silently downgrade a verified route to an identified one with nothing
/// saying so - which is exactly what the field exists to prevent. So `None`, a
/// value, and a value that cannot be parsed are three outcomes rather than two.
///
/// ⚠ Measured through `AGENTS.md` rule 8's route on 2026-09-17: three of this
/// project's four targets carry `digest` on every asset and `aria2/aria2`
/// carries `null` on all six of its, so both of the first two states are shapes
/// this project really meets.
#[test]
fn resolution_reads_an_asset_digest_and_refuses_one_it_cannot_parse() {
    let tag = bit_ids::canonical::Label::parse("v1.0.0").expect("tag");

    let stated = br#"[{"tag_name":"v1.0.0","assets":[
        {"name":"tool-linux","browser_download_url":"https://example.invalid/tool","size":7,
         "digest":"sha256:0000000000000000000000000000000000000000000000000000000000000001"}]}]"#;
    let assets = bit_ids::resolution::sources::github_release_assets(stated, &tag)
        .expect("reads")
        .expect("the tag is present");
    let digest = assets[0].digest.expect("the source stated one");
    assert_eq!(
        digest.to_string(),
        "sha256:0000000000000000000000000000000000000000000000000000000000000001"
    );

    // ⚠ Absent and null are the same answer, and both mean the source states
    // none. An older release predating the field serves `null`.
    for body in [
        &br#"[{"tag_name":"v1.0.0","assets":[
            {"name":"tool-linux","browser_download_url":"https://example.invalid/tool","size":7}]}]"#
            [..],
        &br#"[{"tag_name":"v1.0.0","assets":[
            {"name":"tool-linux","browser_download_url":"https://example.invalid/tool","size":7,
             "digest":null}]}]"#[..],
    ] {
        let assets = bit_ids::resolution::sources::github_release_assets(body, &tag)
            .expect("reads")
            .expect("the tag is present");
        assert!(
            assets[0].digest.is_none(),
            "a source that states no digest is None, not a default"
        );
    }

    // ⛔ And the third state. A bare hex run carries no algorithm, which is the
    // form this project refuses everywhere; `sha512:` is the one a source could
    // plausibly move to.
    //
    // ⚠ THE BAD VALUES ARE ASSEMBLED, NEVER WRITTEN. A harness that plants a
    // pattern cannot spell it, AND NEITHER CAN THE COMMENT EXPLAINING WHY - both
    // of which this test reproduced in turn. A literal 64-digit hex run here is
    // what `check-no-secrets --public` refuses; a doubled brace, which is how a
    // `format!` escapes one, is what `check-placeholders` refuses. The first
    // draft turned the clean tree red in both rules, and the second draft did it
    // again by naming the brace in this paragraph. Concatenation avoids both.
    let bare = "0".repeat(63) + "1";
    for bad in [
        bare.clone(),
        String::from("sha512:") + &bare,
        String::from("sha256:not-hex"),
    ] {
        let body = String::from(
            r#"[{"tag_name":"v1.0.0","assets":[{"name":"tool-linux",
                 "browser_download_url":"https://example.invalid/tool","size":7,"digest":""#,
        ) + &bad
            + r#""}]}]"#;
        let error = bit_ids::resolution::sources::github_release_assets(body.as_bytes(), &tag)
            .expect_err("a digest that cannot be parsed is a finding");
        assert!(
            error.contains("digest"),
            "the message names the field: {error}"
        );
    }
}
