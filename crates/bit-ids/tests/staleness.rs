//! `CI-02`'s three acceptance cases, and a plant for every refusal.
//!
//! ⛔ **The three are driven through the resolver rather than around it.** The
//! preview case is the one that would be a tautology otherwise: handing the
//! survey a stable version and observing that it opens no request for a preview
//! proves nothing, because no preview was present. So each case builds the
//! candidate list a source would answer with, resolves it, and surveys the
//! resolution. What the survey sees is what the resolver decided.
//!
//! ⚠ The store side is a `LatestRow` list rather than a whole corpus, which is
//! what [`bit_ids::index::Indexes::latest`] is. `bit-check check-staleness` is the half
//! that builds a real store, indexes it and feeds the document back in, because
//! a suite that constructs both sides of a comparison agrees with itself about
//! the shape they share.

use bit_ids::ReleaseChannel;
use bit_ids::canonical::{Instant, Label, RelPath, Sha256Digest, Slug, Url, Version};
use bit_ids::identity::{RecordId, RecordKey, SchemaVersion};
use bit_ids::index::LatestRow;
use bit_ids::resolution::{Candidate, Resolution, SourceResponse, VersionScheme, resolve};
use bit_ids::staleness::{
    Assessment, CaptureRequest, OpenRequest, REQUEST_SCHEMA, RequestId, RequestKey, RequestSet,
    RequestState, Staleness, Watch, survey, validate_requests,
};

fn slug(text: &str) -> Slug {
    Slug::parse(text).expect("slug")
}

fn label(text: &str) -> Label {
    Label::parse(text).expect("label")
}

fn instant(text: &str) -> Instant {
    Instant::parse(text).expect("instant")
}

fn version(text: &str) -> Version {
    Version::parse(text).expect("version")
}

/// The tag shape the fixture target publishes: bare, three components.
fn scheme() -> VersionScheme {
    VersionScheme {
        tag_prefix: None,
        min_components: 3,
        max_components: 3,
    }
}

/// A resolution over one source offering the given tags.
///
/// `prerelease` marks the tags the source itself flags, which is one of the two
/// stability signals; a tag whose text carries a token is the other and needs no
/// flag.
fn resolution(target: &str, tags: &[(&str, bool)]) -> Resolution {
    let source_id = slug("upstream-releases");
    let candidates: Vec<Candidate> = tags
        .iter()
        .map(|(tag, prerelease)| Candidate {
            source: source_id.clone(),
            tag: label(tag),
            prerelease: *prerelease,
            draft: false,
            published_at: Some(instant("2026-07-07T21:52:42Z")),
        })
        .collect();
    let response = SourceResponse {
        id: source_id,
        url: Url::parse("https://example.invalid/releases").expect("url"),
        retrieved_at: instant("2026-09-04T12:00:00Z"),
        digest: Sha256Digest::of(target.as_bytes()),
        candidates: u32::try_from(candidates.len()).expect("count"),
    };
    resolve(
        slug(target),
        instant("2026-09-04T12:30:00Z"),
        scheme(),
        vec![response],
        candidates,
    )
}

/// A latest row, which is what the published views carry for one build line.
fn measured(target: &str, platform: &str, package: &str, text: &str) -> LatestRow {
    let schema = SchemaVersion::current();
    let target = slug(target);
    let value = version(text);
    let platform = slug(platform);
    let arch = slug("x86-64");
    let package = slug(package);
    let capture = slug("cap-1");
    let record = RecordId::derive(&RecordKey {
        schema: &schema,
        target: &target,
        version: &value,
        platform: &platform,
        arch: &arch,
        package: &package,
        capture: &capture,
    });
    LatestRow {
        target,
        platform,
        arch,
        package,
        version: value,
        record,
        path: RelPath::parse("profiles/v1/fixture.json").expect("path"),
    }
}

fn linux() -> Vec<Slug> {
    vec![slug("linux")]
}

fn watch<'a>(resolution: &'a Resolution, platforms: &'a [Slug]) -> Watch<'a> {
    Watch {
        resolution,
        platforms,
    }
}

fn run(resolution: &Resolution, latest: &[LatestRow], open: &[OpenRequest]) -> RequestSet {
    let platforms = linux();
    survey(
        instant("2026-09-04T13:00:00Z"),
        &[watch(resolution, &platforms)],
        latest,
        open,
    )
}

fn only(set: &RequestSet) -> &Assessment {
    assert_eq!(set.assessments.len(), 1, "one target, one platform");
    &set.assessments[0]
}

// -- The three acceptance cases ---------------------------------------------

#[test]
fn staleness_opens_one_request_for_a_new_stable_release() {
    let resolution = resolution("fixture-client", &[("1.2.3", false), ("1.2.10", false)]);
    assert_eq!(
        resolution.selected.as_ref().map(Version::as_str),
        Some("1.2.10")
    );

    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.3")];
    let set = run(&resolution, &latest, &[]);

    assert_eq!(only(&set).staleness, Staleness::Stale);
    assert_eq!(set.requests.len(), 1, "exactly one request");
    assert_eq!(set.opened().len(), 1);
    let request = &set.requests[0];
    assert_eq!(request.version.as_str(), "1.2.10");
    assert_eq!(request.platform.as_str(), "linux");
    assert_eq!(request.state, RequestState::Opened);
    assert_eq!(only(&set).request, Some(request.id));
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_opens_nothing_for_a_preview_or_for_a_release_already_measured() {
    // ⛔ The preview is present in the source's answer and the resolver is what
    // refuses it. A survey handed only stable tags would pass this vacuously.
    let resolution = resolution(
        "fixture-client",
        &[("1.2.10", false), ("1.3.0-beta1", false), ("1.4.0", true)],
    );
    assert_eq!(
        resolution.selected.as_ref().map(Version::as_str),
        Some("1.2.10"),
        "the preview by version text and the one by source flag are both refused"
    );

    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.10")];
    let set = run(&resolution, &latest, &[]);

    assert_eq!(only(&set).staleness, Staleness::Current);
    assert!(set.requests.is_empty(), "a known release opens nothing");
    assert!(set.superseded.is_empty());
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_opens_no_second_request_when_the_same_run_repeats() {
    let resolution = resolution("fixture-client", &[("1.2.3", false), ("1.2.10", false)]);
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.3")];

    let first = run(&resolution, &latest, &[]);
    assert_eq!(first.opened().len(), 1);

    // The tracker now holds what the first run opened. The second run is handed
    // exactly that and must produce the same request in a state a caller acts
    // on differently.
    let held: Vec<OpenRequest> = first
        .requests
        .iter()
        .map(|request| OpenRequest {
            target: request.target.clone(),
            version: request.version.clone(),
            channel: request.channel,
            platform: request.platform.clone(),
        })
        .collect();

    let second = run(&resolution, &latest, &held);
    assert_eq!(second.requests.len(), 1);
    assert_eq!(
        second.requests[0].id, first.requests[0].id,
        "one identifier"
    );
    assert_eq!(second.requests[0].state, RequestState::AlreadyOpen);
    assert!(
        second.opened().is_empty(),
        "a caller acting on opened() does nothing"
    );
    assert!(
        second.superseded.is_empty(),
        "the held request still answers"
    );

    // ⭐ And a third run over the same facts is byte-identical to the second,
    // which is the property a scheduled monitor rests on.
    let third = run(&resolution, &latest, &held);
    assert_eq!(
        second.to_json().expect("document"),
        third.to_json().expect("document")
    );
}

// -- What the comparison declines to answer ----------------------------------

#[test]
fn staleness_opens_a_first_request_when_nothing_is_measured_at_all() {
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let set = run(&resolution, &[], &[]);
    assert_eq!(only(&set).staleness, Staleness::Uncaptured);
    assert_eq!(only(&set).measured, None);
    assert_eq!(set.opened().len(), 1);
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_opens_nothing_when_a_measurement_is_newer_than_the_selection() {
    // A source that dropped its newest release, or a resolver reading the wrong
    // one. Either way a request here would ask a runner to capture a downgrade.
    let resolution = resolution("fixture-client", &[("1.2.3", false)]);
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.10")];
    let set = run(&resolution, &latest, &[]);
    assert_eq!(only(&set).staleness, Staleness::Regressed);
    assert!(set.requests.is_empty());
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_reports_a_blocked_resolution_rather_than_skipping_it() {
    // ⛔ The resolver failed closed on an unorderable candidate. A monitor that
    // skipped the target would make a resolution blocked for a month look like a
    // target with no work.
    let resolution = resolution("fixture-client", &[("1.2.10", false), ("1.2", false)]);
    assert!(resolution.selected.is_none(), "the resolver failed closed");
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.3")];
    let set = run(&resolution, &latest, &[]);
    assert_eq!(only(&set).staleness, Staleness::Unresolved);
    assert_eq!(only(&set).selected, None);
    assert_eq!(
        only(&set).measured.as_ref().map(Version::as_str),
        Some("1.2.3"),
        "what is measured is still reported"
    );
    assert!(set.requests.is_empty());
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_opens_nothing_over_a_measured_version_the_scheme_cannot_order() {
    // ⚠ The latest view refuses this under `E-VIW-02`, so it reaches a survey
    // only from a caller that assembled rows by hand. The comparison still has
    // to decline: nothing can rule out that the unorderable version is newer.
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let latest = vec![measured("fixture-client", "linux", "deb", "0.0.0-fixture")];
    let set = run(&resolution, &latest, &[]);
    assert_eq!(only(&set).staleness, Staleness::Unorderable);
    assert!(set.requests.is_empty());
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_declines_when_two_spellings_of_one_release_compare_equal() {
    // ⛔ `components` pads to the scheme's width, so a four-component scheme
    // makes `1.2.10` and `1.2.10.0` one release with two names. Calling it
    // current leaves a differently spelled measurement standing for the
    // selection; calling it stale opens a request for a capture already taken.
    let source_id = slug("upstream-releases");
    let candidates = vec![Candidate {
        source: source_id.clone(),
        tag: label("1.2.10"),
        prerelease: false,
        draft: false,
        published_at: Some(instant("2026-07-07T21:52:42Z")),
    }];
    let padded = VersionScheme {
        tag_prefix: None,
        min_components: 3,
        max_components: 4,
    };
    let resolution = resolve(
        slug("fixture-client"),
        instant("2026-09-04T12:30:00Z"),
        padded,
        vec![SourceResponse {
            id: source_id,
            url: Url::parse("https://example.invalid/releases").expect("url"),
            retrieved_at: instant("2026-09-04T12:00:00Z"),
            digest: Sha256Digest::of(b"padded"),
            candidates: 1,
        }],
        candidates,
    );
    assert_eq!(
        resolution.selected.as_ref().map(Version::as_str),
        Some("1.2.10")
    );

    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.10.0")];
    let set = run(&resolution, &latest, &[]);
    assert_eq!(only(&set).staleness, Staleness::Ambiguous);
    assert!(set.requests.is_empty());
    validate_requests(&set).expect("the survey validates");
}

// -- The "or update" half ----------------------------------------------------

#[test]
fn staleness_retires_an_open_request_the_release_has_moved_past() {
    let resolution = resolution("fixture-client", &[("1.2.10", false), ("1.2.11", false)]);
    assert_eq!(
        resolution.selected.as_ref().map(Version::as_str),
        Some("1.2.11")
    );
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.3")];

    let stale_request = OpenRequest {
        target: slug("fixture-client"),
        version: version("1.2.10"),
        channel: ReleaseChannel::Stable,
        platform: slug("linux"),
    };
    let set = run(&resolution, &latest, std::slice::from_ref(&stale_request));

    assert_eq!(set.requests.len(), 1, "one request, at the newest version");
    assert_eq!(set.requests[0].version.as_str(), "1.2.11");
    assert_eq!(set.requests[0].state, RequestState::Opened);
    assert_eq!(
        set.superseded,
        vec![stale_request.id()],
        "the request the release moved past is retired rather than left beside it"
    );
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_leaves_an_open_request_for_a_line_it_was_not_asked_about() {
    // ⛔ A survey that retired work it was not asked about would delete a
    // capture request because a target was left out of one run's input.
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.10")];
    let elsewhere = OpenRequest {
        target: slug("other-client"),
        version: version("9.9.9"),
        channel: ReleaseChannel::Stable,
        platform: slug("linux"),
    };
    let windows = OpenRequest {
        target: slug("fixture-client"),
        version: version("1.2.10"),
        channel: ReleaseChannel::Stable,
        platform: slug("windows"),
    };
    let set = run(&resolution, &latest, &[elsewhere, windows]);
    assert!(
        set.superseded.is_empty(),
        "neither line was assessed, so neither request is this run's to retire"
    );
}

#[test]
fn staleness_answers_each_platform_separately() {
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let platforms = vec![slug("linux"), slug("windows")];
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.10")];
    let set = survey(
        instant("2026-09-04T13:00:00Z"),
        &[watch(&resolution, &platforms)],
        &latest,
        &[],
    );
    assert_eq!(set.assessments.len(), 2);
    assert_eq!(set.assessments[0].platform.as_str(), "linux");
    assert_eq!(set.assessments[0].staleness, Staleness::Current);
    assert_eq!(set.assessments[1].platform.as_str(), "windows");
    assert_eq!(set.assessments[1].staleness, Staleness::Uncaptured);
    assert_eq!(
        set.requests.len(),
        1,
        "only the uncaptured platform is work"
    );
    assert_eq!(set.requests[0].platform.as_str(), "windows");
    validate_requests(&set).expect("the survey validates");
}

#[test]
fn staleness_asks_whether_the_version_is_measured_on_the_platform_at_all() {
    // ⚠ Two packages of one platform, one of them behind. The question a
    // request answers is whether the selected version has a measurement on the
    // platform, so a lagging package variant is coverage rather than staleness
    // and opening a request for it would ask for a capture already taken.
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let latest = vec![
        measured("fixture-client", "linux", "deb", "1.2.10"),
        measured("fixture-client", "linux", "appimage", "1.2.3"),
    ];
    let set = run(&resolution, &latest, &[]);
    assert_eq!(only(&set).staleness, Staleness::Current);
    assert!(set.requests.is_empty());
}

#[test]
fn staleness_assesses_a_platform_listed_twice_exactly_once() {
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let platforms = vec![slug("linux"), slug("linux")];
    let set = survey(
        instant("2026-09-04T13:00:00Z"),
        &[watch(&resolution, &platforms)],
        &[],
        &[],
    );
    assert_eq!(
        set.assessments.len(),
        1,
        "a catalogue typo is not two lines"
    );
    assert_eq!(set.requests.len(), 1);
    validate_requests(&set).expect("the survey validates");
}

// -- The document ------------------------------------------------------------

#[test]
fn staleness_documents_round_trip_through_their_canonical_form() {
    let resolution = resolution("fixture-client", &[("1.2.3", false), ("1.2.10", false)]);
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.3")];
    let set = run(&resolution, &latest, &[]);
    let document = set.to_json().expect("document");
    let read = RequestSet::from_json(&document).expect("a written survey reads back");
    assert_eq!(read, set);
    assert_eq!(read.to_json().expect("document"), document);
}

#[test]
fn staleness_refuses_a_document_from_another_generation() {
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let set = run(&resolution, &[], &[]);
    let document = set
        .to_json()
        .expect("document")
        .replace(REQUEST_SCHEMA, "bit-ids/capture-requests/2");
    let error = RequestSet::from_json(&document).expect_err("another generation is refused");
    assert!(
        error.to_string().contains("bit-ids/capture-requests/2"),
        "the message names what was found: {error}"
    );
}

// -- Every refusal, planted one at a time ------------------------------------

/// A survey with one stale line, as the base for a plant.
fn planted_base() -> RequestSet {
    let resolution = resolution("fixture-client", &[("1.2.3", false), ("1.2.10", false)]);
    let latest = vec![measured("fixture-client", "linux", "deb", "1.2.3")];
    let set = run(&resolution, &latest, &[]);
    validate_requests(&set).expect("the base is clean");
    set
}

fn refuses(set: &RequestSet, code: &str) {
    let violations = validate_requests(set).expect_err("the plant is refused");
    assert!(
        violations.has(code),
        "expected {code}, got {:?}",
        violations.codes().collect::<Vec<_>>()
    );
}

#[test]
fn staleness_refuses_two_assessments_of_one_line() {
    let mut set = planted_base();
    let duplicate = set.assessments[0].clone();
    set.assessments.push(duplicate);
    refuses(&set, "E-REQ-01");
}

#[test]
fn staleness_refuses_assessments_out_of_canonical_order() {
    let resolution = resolution("fixture-client", &[("1.2.10", false)]);
    let platforms = vec![slug("linux"), slug("windows")];
    let mut set = survey(
        instant("2026-09-04T13:00:00Z"),
        &[watch(&resolution, &platforms)],
        &[],
        &[],
    );
    set.assessments.swap(0, 1);
    refuses(&set, "E-REQ-02");
}

#[test]
fn staleness_refuses_a_request_filed_under_a_name_that_describes_other_work() {
    let mut set = planted_base();
    // ⛔ The identifier is left alone and the key moved, which is the shape a
    // rewritten document takes: the name still resolves, and it now names a
    // capture nobody asked for.
    set.requests[0].version = version("9.9.9");
    refuses(&set, "E-REQ-03");
}

#[test]
fn staleness_refuses_one_identifier_carried_twice() {
    let mut set = planted_base();
    let duplicate = set.requests[0].clone();
    set.requests.push(duplicate);
    refuses(&set, "E-REQ-04");
}

#[test]
fn staleness_refuses_two_requests_for_one_line() {
    let mut set = planted_base();
    let mut second = set.requests[0].clone();
    second.version = version("1.2.11");
    second.id = RequestId::derive(&RequestKey {
        target: &second.target,
        version: &second.version,
        channel: second.channel,
        platform: &second.platform,
    });
    set.requests.push(second);
    set.requests.sort_by_key(|request| request.id.to_string());
    refuses(&set, "E-REQ-05");
}

#[test]
fn staleness_refuses_an_assessment_naming_a_request_that_is_not_carried() {
    let mut set = planted_base();
    set.requests.clear();
    refuses(&set, "E-REQ-06");
}

#[test]
fn staleness_refuses_a_request_no_assessment_names() {
    let mut set = planted_base();
    set.assessments[0].staleness = Staleness::Current;
    set.assessments[0].request = None;
    // Both ends of the pairing fire here: the request is orphaned, and it is a
    // request standing beside a verdict that asks for no capture.
    refuses(&set, "E-REQ-06");
}

#[test]
fn staleness_refuses_a_verdict_that_asks_for_work_and_names_no_request() {
    let mut set = planted_base();
    set.assessments[0].request = None;
    refuses(&set, "E-REQ-07");
}

#[test]
fn staleness_refuses_a_verdict_that_asks_for_nothing_and_names_a_request() {
    let mut set = planted_base();
    set.assessments[0].staleness = Staleness::Regressed;
    refuses(&set, "E-REQ-07");
}

#[test]
fn staleness_refuses_a_request_for_a_version_its_assessment_did_not_select() {
    let mut set = planted_base();
    // The assessment's selection moves and the request stays, which is the shape
    // a survey rewritten against a newer resolution takes.
    set.assessments[0].selected = Some(version("1.2.11"));
    refuses(&set, "E-REQ-08");
}

#[test]
fn staleness_refuses_a_retired_identifier_carried_twice() {
    let mut set = planted_base();
    let stale = OpenRequest {
        target: slug("fixture-client"),
        version: version("1.2.9"),
        channel: ReleaseChannel::Stable,
        platform: slug("linux"),
    };
    set.superseded = vec![stale.id(), stale.id()];
    refuses(&set, "E-REQ-09");
}

#[test]
fn staleness_refuses_an_identifier_both_retired_and_asked_for() {
    let mut set = planted_base();
    set.superseded = vec![set.requests[0].id];
    refuses(&set, "E-REQ-10");
}

#[test]
fn staleness_writes_nothing_it_would_refuse_to_read() {
    let mut set = planted_base();
    set.assessments[0].request = None;
    // ⛔ The write path validates, so an unproven survey has no canonical form.
    // Same rule as `Profile::to_json` and for the same reason: a document on
    // disk is one a reader will trust.
    assert!(set.to_json().is_err());
}

#[test]
fn staleness_request_ids_are_stable_across_the_whole_pipeline() {
    // ⭐ The identifier a survey derives and the identifier a tracker derives
    // from what it holds are the same value, computed by two callers from two
    // types. That equality is what makes a repeated run safe, and it is checked
    // rather than assumed.
    let set = planted_base();
    let request: &CaptureRequest = &set.requests[0];
    let held = OpenRequest {
        target: request.target.clone(),
        version: request.version.clone(),
        channel: request.channel,
        platform: request.platform.clone(),
    };
    assert_eq!(held.id(), request.id);
}
