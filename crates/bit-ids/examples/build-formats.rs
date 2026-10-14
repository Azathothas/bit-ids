//! Render every published format over a store.
//!
//! ```text
//! cargo run -p bit-ids --example build-formats -- [--scheme T:P:MIN:MAX]... STORE OUT
//! ```
//!
//! This is the driving surface for `PUB-03`. `OUT` is a directory the format
//! files are written under, at the paths they are published at.
//!
//! ⛔ **The record set comes from the views, not from a second filter.** Which
//! records are published is `CORPUS-03`'s rule and `CORPUS-04` extended it, so
//! this builds the indexes and asks them. A renderer that filtered the store
//! again would publish a retracted measurement in the tabular view on the day
//! the two spellings drifted.
//!
//! ⛔ **It reads every rendering back before reporting.** The combined JSON is
//! parsed, the compact lines are parsed, and each is compared against the
//! canonical documents. A renderer that reported the digest of what it meant to
//! write cannot detect a short write, which is the same argument `PUB-01` makes
//! for handing its checksums to `sha256sum -c`.
//!
//! ⚠ The CBOR is **not** read back here, and that is deliberate rather than an
//! omission: this project has no CBOR reader and writing one would be checking
//! the writer against itself. `cbor2`, a reader nobody here wrote, read it in
//! `PUB-03`'s driven pass; no gate check decodes it.
//!
//! Exit codes follow `docs/capture-methodology.md`: 0 the formats were rendered
//! and read back, 1 something was refused, 2 the route could not run.

use std::collections::BTreeMap;
use std::io::Write as _;
use std::path::Path;
use std::process::ExitCode;

use bit_ids::canonical::Slug;
use bit_ids::formats::{JSON_FILE, JSONL_FILE, render};
use bit_ids::index::build;
use bit_ids::resolution::VersionScheme;

#[path = "support/reader.rs"]
mod support;
#[path = "support/scheme.rs"]
mod support_scheme;

use support::read_store;
use support_scheme::scheme;

struct Request {
    schemes: BTreeMap<Slug, VersionScheme>,
    store: std::ffi::OsString,
    out: std::ffi::OsString,
}

fn parse_args() -> Result<Request, ExitCode> {
    let mut schemes: BTreeMap<Slug, VersionScheme> = BTreeMap::new();
    let mut positional: Vec<std::ffi::OsString> = Vec::new();
    let mut args = std::env::args_os().skip(1);
    while let Some(arg) = args.next() {
        if arg != *"--scheme" {
            positional.push(arg);
            continue;
        }
        let Some(text) = args.next().and_then(|value| value.into_string().ok()) else {
            let _ = writeln!(std::io::stderr(), "--scheme needs TARGET:PREFIX:MIN:MAX");
            return Err(ExitCode::from(2));
        };
        match scheme(&text) {
            Ok((target, parsed)) => {
                schemes.insert(target, parsed);
            }
            Err(error) => {
                let _ = writeln!(std::io::stderr(), "{error}");
                return Err(ExitCode::from(2));
            }
        }
    }
    let [store, out] = positional.as_slice() else {
        let _ = writeln!(
            std::io::stderr(),
            "usage: build-formats [--scheme TARGET:PREFIX:MIN:MAX]... STORE OUT\n\
             renders every published format over the records the views include"
        );
        return Err(ExitCode::from(2));
    };
    Ok(Request {
        schemes,
        store: store.clone(),
        out: out.clone(),
    })
}

fn report(refusals: &[String]) -> ExitCode {
    let mut stderr = std::io::stderr();
    let _ = writeln!(stderr, "refused: {} finding(s)", refusals.len());
    for refusal in refusals {
        let _ = writeln!(stderr, "  {refusal}");
    }
    ExitCode::from(1)
}

/// Reads the two textual renderings back and compares them with each other.
///
/// ⛔ **The comparison is between two renderings, not between a rendering and
/// the writer's own idea of it.** Both are parsed from what was written, so a
/// field one carries and the other does not is visible here and nowhere else.
fn cross_check(json: &[u8], jsonl: &[u8], refusals: &mut Vec<String>) {
    let combined: Vec<serde_json::Value> = match serde_json::from_slice(json) {
        Ok(value) => value,
        Err(error) => {
            refusals.push(format!("{JSON_FILE}: does not parse: {error}"));
            return;
        }
    };
    let text = match std::str::from_utf8(jsonl) {
        Ok(text) => text,
        Err(error) => {
            refusals.push(format!("{JSONL_FILE}: is not text: {error}"));
            return;
        }
    };
    let mut lines: Vec<serde_json::Value> = Vec::new();
    for (number, line) in text.lines().enumerate() {
        match serde_json::from_str(line) {
            Ok(value) => lines.push(value),
            Err(error) => refusals.push(format!("{JSONL_FILE}:{}: {error}", number + 1)),
        }
    }
    if combined.len() != lines.len() {
        refusals.push(format!(
            "{} carries {} record(s) and {} carries {}",
            JSON_FILE,
            combined.len(),
            JSONL_FILE,
            lines.len()
        ));
        return;
    }
    for (index, (one, other)) in combined.iter().zip(&lines).enumerate() {
        if one != other {
            refusals.push(format!("record {index} differs between the two renderings"));
        }
    }
}

fn main() -> ExitCode {
    let request = match parse_args() {
        Ok(request) => request,
        Err(code) => return code,
    };
    let root = Path::new(&request.store);
    let out = Path::new(&request.out);
    for (label, directory) in [("store", root), ("out", out)] {
        if !directory.is_dir() {
            let _ = writeln!(
                std::io::stderr(),
                "{label}: {} is not a directory",
                directory.display()
            );
            return ExitCode::from(2);
        }
    }

    let mut refusals: Vec<String> = Vec::new();
    let corpus = match read_store(root, &mut refusals) {
        Ok(corpus) => corpus,
        Err(error) => {
            let _ = writeln!(std::io::stderr(), "cannot read the store: {error}");
            return ExitCode::from(2);
        }
    };

    let indexes = match build(&corpus, &request.schemes) {
        Ok(indexes) => indexes,
        Err(violations) => {
            for error in violations.errors() {
                refusals.push(error.to_string());
            }
            return report(&refusals);
        }
    };

    let formats = match render(&corpus, &indexes) {
        Ok(formats) => formats,
        Err(violations) => {
            for error in violations.errors() {
                refusals.push(error.to_string());
            }
            return report(&refusals);
        }
    };
    if !refusals.is_empty() {
        return report(&refusals);
    }

    for (path, bytes) in &formats.files {
        let target = out.join(path.as_str());
        if let Some(parent) = target.parent()
            && let Err(error) = std::fs::create_dir_all(parent)
        {
            let _ = writeln!(std::io::stderr(), "{}: {error}", parent.display());
            return ExitCode::from(2);
        }
        if let Err(error) = std::fs::write(&target, bytes) {
            let _ = writeln!(std::io::stderr(), "{}: {error}", target.display());
            return ExitCode::from(2);
        }
    }

    // ⛔ Read off the disk, never out of the buffer that was just written. A
    // short write leaves a file that agrees with itself.
    let mut written: BTreeMap<&str, Vec<u8>> = BTreeMap::new();
    for (path, _) in &formats.files {
        match std::fs::read(out.join(path.as_str())) {
            Ok(bytes) => {
                written.insert(path.as_str(), bytes);
            }
            Err(error) => refusals.push(format!("{path}: {error}")),
        }
    }
    if !refusals.is_empty() {
        return report(&refusals);
    }
    for (path, bytes) in &formats.files {
        if written.get(path.as_str()) != Some(bytes) {
            refusals.push(format!(
                "{path}: what is on the disk is not what was rendered"
            ));
        }
    }

    if let (Some(json), Some(jsonl)) = (written.get(JSON_FILE), written.get(JSONL_FILE)) {
        cross_check(json, jsonl, &mut refusals);
    }
    if !refusals.is_empty() {
        return report(&refusals);
    }

    let mut stdout = std::io::stdout();
    let _ = writeln!(
        stdout,
        "{} record(s) in {} file(s)",
        formats.records,
        formats.files.len()
    );
    for (path, bytes) in &formats.files {
        let _ = writeln!(stdout, "  {} {} byte(s)", path, bytes.len());
    }
    ExitCode::SUCCESS
}
