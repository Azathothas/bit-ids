//! Resolves the newest stable release from responses already fetched, and
//! prints the decision trace.
//!
//! It does not fetch. `bit-check fetch-releases`, in
//! [`../../../tools/check/fetchreleases.go`](../../../tools/check/fetchreleases.go),
//! does that, so the bytes a decision was made from exist as a file before
//! anything reads them. Splitting it that way is what makes the digest in the
//! resolution mean something: it is of what arrived, not of what a parser
//! reconstructed.
//!
//! ```text
//! resolve-stable <target> <tag-prefix|-> <min> <max> <floor-major|-> <source-id> <url> <body-file> [more...]
//! ```
//!
//! ⚠ **The floor is positional and always given**, `-` for none, so a caller
//! that forgot it is a usage error rather than a source id read as a number.
//! `CLIENT-06`.
//!
//! ⛔ **The source id names the FORMAT the body is read as**, and the set is
//! closed: `github-releases` for a releases listing, `git-refs` for
//! `git ls-remote --tags --refs` output. One value therefore says both what
//! answered and how it is read, where a separate flag would be a second place
//! for those to disagree. ⚠ Two sources of one format are told apart by their
//! URLs, which is also what `E-ACQ-07` compares once a route records them.
//!
//! Exit codes: 0 a version was selected, 1 the resolver failed closed, 2 it
//! could not run. ⛔ Failing closed is exit 1, not exit 0 with an empty answer:
//! a caller that ignored the distinction would install nothing and report
//! success.

use std::process::ExitCode;

use bit_ids::canonical::{Instant, Label, Sha256Digest, Slug, Url};
use bit_ids::resolution::{Candidate, SourceResponse, VersionScheme, resolve, sources};

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args.len() < 8 || !(args.len() - 5).is_multiple_of(3) {
        eprintln!(
            "usage: resolve-stable <target> <tag-prefix|-> <min> <max> <floor-major|-> \
             (<source-id> <url> <body-file>)..."
        );
        return ExitCode::from(2);
    }
    let Ok(scheme) = build_scheme(&args) else {
        eprintln!("the tag scheme is not usable");
        return ExitCode::from(2);
    };
    let Ok(target) = Slug::parse(&args[0]) else {
        eprintln!("target {:?} is not a slug", args[0]);
        return ExitCode::from(2);
    };

    let mut responses = Vec::new();
    let mut candidates = Vec::new();
    for group in args[5..].chunks(3) {
        match read_source(&group[0], &group[1], &group[2]) {
            Ok((response, found)) => {
                responses.push(response);
                candidates.extend(found);
            }
            Err(message) => {
                eprintln!("{}: {message}", group[0]);
                return ExitCode::from(2);
            }
        }
    }
    responses.sort_by(|a, b| a.id.as_str().cmp(b.id.as_str()));

    // ⚠ Stamped after every source answered. A decision that predates its own
    // input is refused by `E-RES-05`, so the clock is read here and not earlier.
    let Ok(now) = now_utc() else {
        eprintln!("the host clock is not readable as UTC seconds");
        return ExitCode::from(2);
    };
    let resolution = resolve(target, now, scheme, responses, candidates);

    let document = match resolution.to_json() {
        Ok(document) => document,
        Err(error) => {
            eprintln!("the decision is not a valid document: {error}");
            return ExitCode::from(2);
        }
    };
    print!("{document}");
    if let Some(version) = &resolution.selected {
        eprintln!(
            "selected {version} from {} candidate(s)",
            resolution.considered.len()
        );
        ExitCode::SUCCESS
    } else {
        eprintln!(
            "failed closed over {} candidate(s); the trace says why",
            resolution.considered.len()
        );
        ExitCode::FAILURE
    }
}

fn build_scheme(args: &[String]) -> Result<VersionScheme, ()> {
    let tag_prefix = if args[1] == "-" {
        None
    } else {
        Some(Label::parse(&args[1]).map_err(|_| ())?)
    };
    let floor_major = if args[4] == "-" {
        None
    } else {
        Some(args[4].parse().map_err(|_| ())?)
    };
    Ok(VersionScheme {
        tag_prefix,
        min_components: args[2].parse().map_err(|_| ())?,
        max_components: args[3].parse().map_err(|_| ())?,
        floor_major,
    })
}

fn read_source(
    id: &str,
    url: &str,
    path: &str,
) -> Result<(SourceResponse, Vec<Candidate>), String> {
    let id = Slug::parse(id).map_err(|error| format!("source id: {error}"))?;
    let url = Url::parse(url).map_err(|error| format!("url: {error}"))?;
    let body = std::fs::read(path).map_err(|error| format!("{path}: {error}"))?;
    // ⛔ THE SOURCE ID CHOOSES THE PARSER, so a caller cannot hand a refs
    // listing to the releases reader and get "not a release list" from a file
    // that is perfectly good. ⚠ It is the id rather than a fourth argument
    // because the id is already what the resolution records as the source, so
    // one value says both what answered and how it is read; a separate flag
    // would be a second place for those to disagree.
    let candidates = match id.as_str() {
        "git-refs" => sources::git_refs(&body, &id)?,
        "github-releases" => sources::github_releases(&body, &id)?,
        other => {
            return Err(format!(
                "source {other:?} has no reader; this resolver knows github-releases and git-refs"
            ));
        }
    };
    let retrieved_at = file_instant(path)?;
    let count = u32::try_from(candidates.len()).map_err(|_| "too many candidates".to_owned())?;
    Ok((
        SourceResponse {
            id,
            url,
            // ⛔ The digest is of the bytes on disk, so a later reader can tell
            // a resolver defect from a source that changed its answer.
            digest: Sha256Digest::of(&body),
            retrieved_at,
            candidates: count,
        },
        candidates,
    ))
}

/// When the fetch retrieved the body, as the fetch recorded it.
///
/// ⛔ Read from the `.fetched-at` sidecar, never inferred from the file's
/// modification time. An mtime survives a copy, an archive restore and a
/// checkout, so a resolution built from one would publish a retrieval instant
/// that is not one. A door sweep found this resting on mtime.
fn file_instant(path: &str) -> Result<Instant, String> {
    let sidecar = format!("{path}.fetched-at");
    let text = std::fs::read_to_string(&sidecar).map_err(|error| {
        format!("{sidecar}: {error}; the fetch wrapper writes this beside the body")
    })?;
    Instant::parse(text.trim()).map_err(|error| format!("{sidecar}: {error}"))
}

fn now_utc() -> Result<Instant, ()> {
    let seconds = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_err(|_| ())?
        .as_secs();
    Instant::parse(&format_epoch(seconds)).map_err(|_| ())
}

/// Formats seconds since the epoch as the one instant spelling this schema
/// accepts. Written out rather than pulled in: a date crate for one function is
/// a dependency this project would have to argue for.
fn format_epoch(seconds: u64) -> String {
    let days = seconds / 86_400;
    let rest = seconds % 86_400;
    let (hour, minute, second) = (rest / 3600, (rest % 3600) / 60, rest % 60);
    let (year, month, day) = civil_from_days(days);
    format!("{year:04}-{month:02}-{day:02}T{hour:02}:{minute:02}:{second:02}Z")
}

/// Days since 1970-01-01 to a civil date, by Howard Hinnant's `civil_from_days`.
fn civil_from_days(days: u64) -> (u64, u64, u64) {
    let z = days + 719_468;
    let era = z / 146_097;
    let doe = z % 146_097;
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    (if m <= 2 { y + 1 } else { y }, m, d)
}
