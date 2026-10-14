//! Acceptance for `SCHEMA-04`.
//!
//! ```text
//! cargo test --workspace variability
//! ```
//!
//! Every test name carries `variability` so that command selects this file.
//!
//! The classifications here are a function of the samples, never a judgement
//! about them. A dimension the run did not vary produces `unknown`, and one
//! sample produces `unknown` for everything, because a value that looked the
//! same twice inside one process is not a value shown to survive a restart.

use core::num::NonZeroU32;

use bit_ids::canonical::HexBytes;
use bit_ids::observation::{FieldState, PatternRun};
use bit_ids::sampling::{Lifetime, Sample, SamplingPlan, classify, field_state};

fn plan(sessions: u32, torrents: u32, connections: u32) -> SamplingPlan {
    SamplingPlan {
        sessions: NonZeroU32::new(sessions).expect("a plan exercises at least one"),
        torrents: NonZeroU32::new(torrents).expect("a plan exercises at least one"),
        connections: NonZeroU32::new(connections).expect("a plan exercises at least one"),
    }
}

fn sample(session: u32, torrent: u32, connection: u32, hex: &str) -> Sample {
    Sample {
        session,
        torrent,
        connection,
        value: HexBytes::parse(hex).expect("a canonical byte string"),
    }
}

/// A 20-byte peer ID: the eight-byte client prefix `-XX0000-`, then the twelve
/// bytes the caller supplies as 24 hex digits.
fn peer_id(suffix: &str) -> String {
    assert_eq!(
        suffix.len(),
        24,
        "a peer ID is 20 bytes, so the suffix is 12 of them"
    );
    format!("2d5858303030302d{suffix}")
}

// -- the shape a peer ID actually has --------------------------------------

#[test]
fn variability_separates_a_fixed_prefix_from_a_changing_suffix() {
    let plan = plan(4, 1, 1);
    let samples: Vec<Sample> = (0..4)
        .map(|session| {
            sample(
                session,
                0,
                0,
                &peer_id(&format!("aabbccddeeff0011223344{session:02x}")),
            )
        })
        .collect();

    let report = classify(&samples, &plan).expect("four samples classify");
    assert_eq!(report.length, Some(20));
    assert_eq!(report.samples, 4);

    // The eight-byte client prefix never moved, across four separate process
    // starts, so it is a stored value rather than one that happened to repeat.
    for offset in 0..8 {
        assert_eq!(
            report.span_at(offset).map(|span| span.lifetime),
            Some(Lifetime::Persistent),
            "byte {offset} is part of the fixed prefix"
        );
    }
    // The last byte is the one that moved, and it moved with the session.
    assert_eq!(
        report.span_at(19).map(|span| span.lifetime),
        Some(Lifetime::PerSession)
    );
    assert!(report.any_variation());
}

#[test]
fn variability_reports_adjacent_bytes_that_behaved_alike_as_one_span() {
    let plan = plan(2, 1, 1);
    let samples = vec![
        sample(0, 0, 0, &peer_id("111111110000000000000000")),
        sample(1, 0, 0, &peer_id("222222220000000000000000")),
    ];
    let report = classify(&samples, &plan).expect("two samples classify");
    let spans: Vec<(usize, usize, Lifetime)> = report
        .spans
        .iter()
        .map(|span| (span.offset, span.length, span.lifetime))
        .collect();
    assert_eq!(
        spans,
        vec![
            (0, 8, Lifetime::Persistent),
            (8, 4, Lifetime::PerSession),
            (12, 8, Lifetime::Persistent),
        ],
        "the value is three runs, not twenty independent bytes"
    );
}

// -- one sample proves nothing about a lifetime ----------------------------

#[test]
fn variability_refuses_to_call_anything_stable_from_one_sample() {
    let plan = plan(1, 1, 1);
    let samples = vec![sample(0, 0, 0, &peer_id("aabbccddeeff001122334455"))];
    let report = classify(&samples, &plan).expect("one sample still reports");
    assert_eq!(report.samples, 1);
    for span in &report.spans {
        assert_eq!(
            span.lifetime,
            Lifetime::Unknown,
            "one observation establishes bytes for one connection and nothing more"
        );
    }
    assert!(!report.any_variation());
}

#[test]
fn variability_refuses_persistence_when_the_run_never_restarted() {
    // Four identical samples, all from one process. Nothing distinguishes a
    // value stored on disk from one generated once per process start.
    let plan = plan(1, 2, 2);
    let samples: Vec<Sample> = (0..2)
        .flat_map(|torrent| {
            (0..2).map(move |connection| {
                sample(0, torrent, connection, &peer_id("aabbccddeeff001122334455"))
            })
        })
        .collect();
    let report = classify(&samples, &plan).expect("four samples classify");
    assert_eq!(report.samples, 4);
    for span in &report.spans {
        assert_eq!(
            span.lifetime,
            Lifetime::Unknown,
            "a value that never changed inside one process is not a persistent value"
        );
    }
}

#[test]
fn variability_calls_it_persistent_only_once_a_restart_has_been_exercised() {
    let plan = plan(2, 1, 1);
    let samples = vec![
        sample(0, 0, 0, &peer_id("aabbccddeeff001122334455")),
        sample(1, 0, 0, &peer_id("aabbccddeeff001122334455")),
    ];
    let report = classify(&samples, &plan).expect("two samples classify");
    for span in &report.spans {
        assert_eq!(span.lifetime, Lifetime::Persistent);
    }
}

// -- each dimension is separated by the one that varied --------------------

#[test]
fn variability_names_a_value_that_changes_per_connection() {
    let plan = plan(1, 1, 3);
    let samples: Vec<Sample> = (0..3)
        .map(|connection| {
            sample(
                0,
                0,
                connection,
                &peer_id(&format!("aabbccddeeff0011223344{connection:02x}")),
            )
        })
        .collect();
    let report = classify(&samples, &plan).expect("three samples classify");
    assert_eq!(
        report.span_at(19).map(|span| span.lifetime),
        Some(Lifetime::PerConnection),
        "it changed between connections inside one session and one torrent"
    );
}

#[test]
fn variability_names_a_value_that_changes_per_torrent() {
    let plan = plan(1, 3, 2);
    let samples: Vec<Sample> = (0..3)
        .flat_map(|torrent| {
            (0..2).map(move |connection| {
                sample(
                    0,
                    torrent,
                    connection,
                    &peer_id(&format!("aabbccddeeff0011223344{torrent:02x}")),
                )
            })
        })
        .collect();
    let report = classify(&samples, &plan).expect("six samples classify");
    assert_eq!(
        report.span_at(19).map(|span| span.lifetime),
        Some(Lifetime::PerTorrent),
        "it held across connections and moved with the torrent"
    );
}

#[test]
fn variability_names_a_value_that_changes_per_session() {
    let plan = plan(3, 2, 1);
    let samples: Vec<Sample> = (0..3)
        .flat_map(|session| {
            (0..2).map(move |torrent| {
                sample(
                    session,
                    torrent,
                    0,
                    &peer_id(&format!("aabbccddeeff0011223344{session:02x}")),
                )
            })
        })
        .collect();
    let report = classify(&samples, &plan).expect("six samples classify");
    assert_eq!(
        report.span_at(19).map(|span| span.lifetime),
        Some(Lifetime::PerSession),
        "it held across torrents and moved with the process"
    );
}

#[test]
fn variability_will_not_name_a_dimension_the_run_did_not_vary() {
    // The value changes with the session, but the plan only ran one session, so
    // whatever produced the difference, this run did not separate it.
    let plan = plan(1, 1, 2);
    let samples = vec![
        sample(0, 0, 0, &peer_id("aabbccddeeff001122334455")),
        sample(0, 0, 1, &peer_id("aabbccddeeff001122334466")),
    ];
    let report = classify(&samples, &plan).expect("two samples classify");
    assert_eq!(
        report.span_at(19).map(|span| span.lifetime),
        Some(Lifetime::PerConnection)
    );
    // And the bytes that held still cannot be called persistent.
    assert_eq!(
        report.span_at(0).map(|span| span.lifetime),
        Some(Lifetime::Unknown)
    );
}

// -- edges -----------------------------------------------------------------

#[test]
fn variability_reports_a_changing_width_rather_than_guessing_at_offsets() {
    let plan = plan(2, 1, 1);
    let samples = vec![sample(0, 0, 0, "aabbcc"), sample(1, 0, 0, "aabbccdd")];
    let report = classify(&samples, &plan).expect("two samples classify");
    assert_eq!(
        report.length, None,
        "offsets do not line up, so nothing can be said per offset"
    );
    assert!(report.spans.is_empty());
    assert_eq!(report.samples, 2);
}

#[test]
fn variability_has_nothing_to_report_with_no_samples_at_all() {
    assert!(classify(&[], &plan(4, 2, 2)).is_none());
}

#[test]
fn variability_counts_what_a_plan_can_produce() {
    assert_eq!(plan(4, 2, 3).observations(), 24);
    assert_eq!(plan(1, 1, 1).observations(), 1);
    assert!(!plan(1, 1, 1).varies_anything());
    assert!(!plan(1, 1, 1).restarts());
    assert!(plan(1, 2, 1).varies_anything());
    assert!(
        !plan(1, 2, 1).restarts(),
        "varying torrents is not restarting the process"
    );
    assert!(plan(2, 1, 1).restarts());
}

// -- the state a record carries, derived from those samples ----------------
//
// `field_state` is the join this entry was closed without: the classifier said
// what the samples prove and nothing turned that into the `FieldState` a record
// holds. These assert the mapping, and the one that matters most is the third,
// which is the case where reading the lifetime rather than the bytes would
// report a value that changed as fixed.

fn runs_of(state: &FieldState) -> Vec<(&'static str, usize)> {
    let FieldState::Patterned(patterned) = state else {
        panic!("expected a patterned state, got {}", state.as_str());
    };
    patterned
        .pattern
        .runs
        .iter()
        .map(|run| match run {
            PatternRun::Fixed { bytes } => ("fixed", bytes.len()),
            PatternRun::Varying { length, .. } => ("varying", *length),
        })
        .collect()
}

/// Four restarts of a build whose eight-byte prefix holds and whose twelve-byte
/// tail is regenerated, which is the shape a real peer ID has.
fn restarted_peer_ids() -> Vec<Sample> {
    (0..4u32)
        .map(|session| {
            let mut bytes = b"-XX0000-".to_vec();
            bytes.extend((0..12u32).map(|index| {
                u8::try_from((session * 13 + index * 7 + 1) & 0xff).expect("masked to one byte")
            }));
            Sample {
                session,
                torrent: 0,
                connection: 0,
                value: HexBytes::new(bytes).expect("twenty bytes is not empty"),
            }
        })
        .collect()
}

#[test]
fn variability_turns_a_restarted_peer_id_into_the_patterned_state_a_record_carries() {
    let state =
        field_state(&restarted_peer_ids(), &plan(4, 1, 1)).expect("four samples support a state");
    assert_eq!(
        runs_of(&state),
        vec![("fixed", 8), ("varying", 12)],
        "the client prefix is the identifying half and it is kept"
    );
    let FieldState::Patterned(patterned) = &state else {
        unreachable!("runs_of already refused anything else");
    };
    assert_eq!(patterned.pattern.length, 20);
    assert_eq!(patterned.samples.get(), 4);
}

#[test]
fn variability_reports_a_value_that_never_moved_as_constant() {
    let plan = plan(3, 1, 1);
    let samples: Vec<Sample> = (0..3)
        .map(|session| sample(session, 0, 0, &peer_id("aabbccddeeff001122334455")))
        .collect();
    let state = field_state(&samples, &plan).expect("three samples support a state");
    let FieldState::Constant(constant) = &state else {
        panic!("expected constant, got {}", state.as_str());
    };
    assert_eq!(constant.samples.get(), 3);
    assert!(
        !state.claims_variation(),
        "nothing moved, so nothing may claim it did"
    );
}

#[test]
fn variability_will_not_call_a_span_fixed_because_its_lifetime_does_not_vary() {
    // ⛔ `Lifetime::Unknown` covers two facts: a value that never changed, and a
    // value that changed where no dimension the plan varied separates the
    // change. A state derived from the lifetime would report the second as
    // fixed bytes. These samples are labelled across sessions under a plan that
    // declares one, which is the combination that reaches it.
    let plan = plan(1, 2, 1);
    let samples = vec![
        sample(0, 0, 0, &peer_id("aabbccddeeff001122334455")),
        sample(1, 0, 0, &peer_id("aabbccddeeff00112233ffff")),
    ];
    let offset = 18;
    let report = classify(&samples, &plan).expect("two samples classify");
    assert_eq!(
        report.span_at(offset).map(|span| span.lifetime),
        Some(Lifetime::Unknown),
        "this is the case the guard exists for; if it stops being unknown the \
         test is no longer testing anything"
    );

    let state = field_state(&samples, &plan).expect("two samples support a state");
    assert_eq!(
        runs_of(&state),
        vec![("fixed", 18), ("varying", 2)],
        "the bytes differ over those two, whatever the lifetime could not say"
    );
}

#[test]
fn variability_merges_touching_spans_the_record_cannot_tell_apart() {
    // `classify` splits on lifetime and a record's pattern splits on bytes, so
    // a per-connection byte beside a per-session one is two spans there and one
    // varying run here. Two shapes for one measurement would be two records.
    let plan = plan(2, 1, 2);
    let samples: Vec<Sample> = [(0, 0), (0, 1), (1, 0), (1, 1)]
        .into_iter()
        .map(|(session, connection): (u32, u32)| {
            let tail = format!("{connection:02x}{session:02x}");
            sample(
                session,
                0,
                connection,
                &peer_id(&format!("aabbccddeeff00112233{tail}")),
            )
        })
        .collect();
    let report = classify(&samples, &plan).expect("four samples classify");
    assert_eq!(
        report.span_at(18).map(|span| span.lifetime),
        Some(Lifetime::PerConnection)
    );
    assert_eq!(
        report.span_at(19).map(|span| span.lifetime),
        Some(Lifetime::PerSession),
        "two different lifetimes, which is what makes them two spans"
    );

    let state = field_state(&samples, &plan).expect("four samples support a state");
    assert_eq!(runs_of(&state), vec![("fixed", 18), ("varying", 2)]);
}

#[test]
fn variability_reports_a_value_whose_every_byte_moved_as_variable() {
    // ⛔ A pattern needs a fixed run and a varying one, so a value with no
    // stable span is a variable rather than a pattern of one run. `E-OBS-10`
    // refuses the other spelling.
    let plan = plan(2, 1, 1);
    let samples = vec![sample(0, 0, 0, "aabbcc"), sample(1, 0, 0, "ddeeff")];
    let state = field_state(&samples, &plan).expect("two samples support a state");
    let FieldState::Variable(variable) = &state else {
        panic!("expected variable, got {}", state.as_str());
    };
    assert_eq!(variable.length, Some(3));
    assert_eq!(variable.samples.get(), 2);
    assert_eq!(variable.distinct.get(), 2);
}

#[test]
fn variability_reports_a_changing_width_as_a_variable_with_no_length() {
    let plan = plan(2, 1, 1);
    let samples = vec![sample(0, 0, 0, "aabbcc"), sample(1, 0, 0, "aabbccdd")];
    let state = field_state(&samples, &plan).expect("two samples support a state");
    let FieldState::Variable(variable) = &state else {
        panic!("expected variable, got {}", state.as_str());
    };
    assert_eq!(
        variable.length, None,
        "the width itself moved, so there is no common length to state"
    );
    assert_eq!(variable.distinct.get(), 2);
}

#[test]
fn variability_supports_no_state_at_all_over_no_samples() {
    assert!(
        field_state(&[], &plan(4, 2, 2)).is_none(),
        "a state over nothing is a measurement nobody took"
    );
}

#[test]
fn variability_states_the_bytes_even_when_the_plan_varied_nothing() {
    // ⚠ The plan is the caller's claim and the samples are the measurement, so
    // this reports what it saw. The pair is contradictory and `E-BND-20` is
    // what refuses it, against the run manifest - not this function, which
    // would otherwise be a second gate answering a different way.
    let plan = plan(1, 1, 1);
    let samples = vec![sample(0, 0, 0, "aabbcc"), sample(0, 0, 0, "aabbdd")];
    let state = field_state(&samples, &plan).expect("two samples support a state");
    assert!(
        state.claims_variation(),
        "the bytes moved; calling that constant would publish a value no sample carried"
    );
}

/// What a consumer holding a profile and a run manifest can say about a span,
/// and the boundary where it must say nothing.
///
/// ⛔ **A PUBLISHED `PatternRun` CARRIES NO LIFETIME, AND THAT IS A DECISION.**
/// `SCHEMA-04` puts the plan in the manifest and the claim in the profile, so a
/// lifetime stated in both would be a value that can disagree with itself. ⚠ The
/// half that was missing is the ASKING: a consumer holding the pair had to
/// re-reason about the plan itself, which is a second implementation of the
/// classifier in every consumer.
#[test]
fn a_plan_says_what_a_varying_span_means_or_says_nothing() {
    // One varied dimension attributes a change to exactly that dimension.
    assert_eq!(plan(1, 1, 4).lifetime_of(true), Lifetime::PerConnection);
    assert_eq!(plan(1, 3, 1).lifetime_of(true), Lifetime::PerTorrent);
    assert_eq!(plan(2, 1, 1).lifetime_of(true), Lifetime::PerSession);

    // ⛔ TWO VARIED DIMENSIONS BOTH EXPLAIN A CHANGE, so the plan alone cannot
    // say which and must not pick one. That is the case only the samples settle.
    assert_eq!(plan(2, 1, 4).lifetime_of(true), Lifetime::Unknown);
    assert_eq!(plan(1, 3, 4).lifetime_of(true), Lifetime::Unknown);
    assert_eq!(plan(2, 3, 4).lifetime_of(true), Lifetime::Unknown);

    // A plan that varied nothing cannot attribute a change at all.
    assert_eq!(plan(1, 1, 1).lifetime_of(true), Lifetime::Unknown);

    // ⚠ HOLDING STILL IS ONLY EVIDENCE OF STORAGE IF THE PROCESS RESTARTED.
    // Inside one session every lifetime but per-connection produces the same
    // bytes, so a fixed span there says nothing.
    assert_eq!(plan(2, 1, 1).lifetime_of(false), Lifetime::Persistent);
    assert_eq!(plan(1, 4, 4).lifetime_of(false), Lifetime::Unknown);
}

/// The plan-only derivation never contradicts the one that reads the samples.
///
/// ⭐ **TWO DERIVATIONS OF ONE FACT, COMPARED.** `classify` sees which grouping a
/// value is constant within; `lifetime_of` has only the plan, so it is
/// deliberately weaker and answers `Unknown` where the classifier can still
/// separate the cases. ⛔ What must never happen is the two naming DIFFERENT
/// lifetimes: that would be a consumer reading the published pair and getting an
/// answer the measurement refutes.
///
/// ⚠ The samples are built so the value changes per connection, which is the
/// shape this project actually measured on a peer ID's tail.
#[test]
fn the_plan_only_derivation_never_contradicts_the_classifier() {
    for sessions in [1u32, 2] {
        for torrents in [1u32, 2] {
            for connections in [1u32, 2] {
                let p = plan(sessions, torrents, connections);
                let mut samples = Vec::new();
                let mut tail = 0u8;
                for s in 0..sessions {
                    for t in 0..torrents {
                        for c in 0..connections {
                            samples.push(sample(s, t, c, &format!("aa{tail:02x}")));
                            tail += 1;
                        }
                    }
                }
                let report = classify(&samples, &p).expect("a report");

                // Offset 0 never moves; offset 1 moves on every observation.
                let fixed = report.span_at(0).expect("a span").lifetime;
                let varying = report.span_at(1).expect("a span").lifetime;

                for (measured, derived) in [
                    (fixed, p.lifetime_of(false)),
                    (varying, p.lifetime_of(true)),
                ] {
                    assert!(
                        derived == Lifetime::Unknown || derived == measured,
                        "plan {sessions}/{torrents}/{connections}: the plan says {} and \
                         the samples say {}",
                        derived.as_str(),
                        measured.as_str()
                    );
                }
            }
        }
    }
}

/// ⛔ And the control the case above needs: the derivation is not simply
/// `Unknown` everywhere.
///
/// ⚠ Without this, `derived == Lifetime::Unknown ||` satisfies every assertion
/// over a function that never says anything - which is a check that passes
/// because a different branch happens to satisfy it.
#[test]
fn the_plan_only_derivation_is_not_unknown_everywhere() {
    let answered = [
        plan(1, 1, 4).lifetime_of(true),
        plan(1, 3, 1).lifetime_of(true),
        plan(2, 1, 1).lifetime_of(true),
        plan(2, 1, 1).lifetime_of(false),
    ];
    assert!(
        answered.iter().all(|l| *l != Lifetime::Unknown),
        "every one of these plans separates exactly one dimension"
    );
}

// -- a byte two random values happen to share --------------------------------

#[test]
fn variability_will_not_call_a_byte_fixed_because_two_random_tails_shared_it() {
    // ⛔ `capture-client` run 37's release lane, as announced: two sessions of
    // `aria2-next` 3.0.3, whose random tails share `M` at offset 10. Per offset
    // that was a fixed run inside the tail, and the source lane's own
    // coincidence sat elsewhere, so the two routes disagreed.
    let tails = [b"-qB5230-A(Mq~CueHfWD", b"-qB5230-bEMwBhHOPtBp"];
    let samples: Vec<Sample> = tails
        .iter()
        .zip(0u32..)
        .map(|(bytes, session)| Sample {
            session,
            torrent: 0,
            connection: 0,
            value: HexBytes::new(bytes.to_vec()).expect("twenty bytes is not empty"),
        })
        .collect();
    let state = field_state(&samples, &plan(2, 1, 1)).expect("two samples support a state");
    assert_eq!(runs_of(&state), vec![("fixed", 8), ("varying", 12)]);
}

#[test]
fn variability_keeps_an_interior_byte_three_distinct_values_share() {
    // ⚠ Three distinct values agreeing at one offset is about one chance in four
    // thousand at random, which is a byte the samples establish.
    let samples = vec![
        sample(0, 0, 0, &peer_id("aabb7accddeeff0011223344")),
        sample(1, 0, 0, &peer_id("11227a33445566778899aabb")),
        sample(2, 0, 0, &peer_id("99887a6655443322110fedcb")),
    ];
    let state = field_state(&samples, &plan(3, 1, 1)).expect("three samples support a state");
    assert_eq!(
        runs_of(&state),
        vec![("fixed", 8), ("varying", 2), ("fixed", 1), ("varying", 9)]
    );
}

#[test]
fn variability_keeps_two_interior_bytes_two_distinct_values_share() {
    // ⚠ The boundary: two bytes shared by two values is as unlikely as one byte
    // shared by three, so it is kept.
    let samples = vec![
        sample(0, 0, 0, &peer_id("aabb7a7bccddeeff00112233")),
        sample(1, 0, 0, &peer_id("11227a7b3344556677889900")),
    ];
    let state = field_state(&samples, &plan(2, 1, 1)).expect("two samples support a state");
    assert_eq!(
        runs_of(&state),
        vec![("fixed", 8), ("varying", 2), ("fixed", 2), ("varying", 8)]
    );
}

#[test]
fn variability_never_absorbs_a_leading_or_trailing_fixed_run() {
    // ⚠ A one-byte fixed run at the end has nothing varying beyond it, so the
    // rule does not reach it, however few values there are.
    let samples = vec![
        sample(0, 0, 0, &peer_id("aabbccddeeff00112233447f")),
        sample(1, 0, 0, &peer_id("112233445566778899aabb7f")),
    ];
    let state = field_state(&samples, &plan(2, 1, 1)).expect("two samples support a state");
    assert_eq!(
        runs_of(&state),
        vec![("fixed", 8), ("varying", 11), ("fixed", 1)]
    );
}
