//! Parse a declared version scheme from the command line.
//!
//! ⛔ **Separate from [`reader.rs`](reader.rs) because not every example that
//! reads a store declares a scheme.** `survey-staleness` takes its schemes out
//! of the resolutions it was given, so including this there would compile a
//! parser nothing calls, and the dead-code lint is what said so. The two
//! helpers had no reason to share a file beyond both being needed by the first
//! example that wanted them.

use bit_ids::canonical::{Label, Slug};
use bit_ids::resolution::VersionScheme;

/// Parses one `--scheme TARGET:PREFIX:MIN:MAX` argument.
///
/// `-` in the prefix position means the target publishes versions with no tag
/// prefix. ⛔ Nothing here has a default: a scheme this cannot parse is refused
/// rather than filled in, because a filled-in scheme orders versions under a
/// shape nobody declared.
pub fn scheme(text: &str) -> Result<(Slug, VersionScheme), String> {
    let parts: Vec<&str> = text.split(':').collect();
    let [target, prefix, min, max] = parts.as_slice() else {
        return Err(format!("{text:?}: expected TARGET:PREFIX:MIN:MAX"));
    };
    let target = Slug::parse(target).map_err(|error| format!("{text:?}: {error}"))?;
    let tag_prefix = if *prefix == "-" {
        None
    } else {
        Some(Label::parse(prefix).map_err(|error| format!("{text:?}: {error}"))?)
    };
    let min_components: u8 = min.parse().map_err(|_| format!("{text:?}: min"))?;
    let max_components: u8 = max.parse().map_err(|_| format!("{text:?}: max"))?;
    if min_components == 0 || min_components > max_components {
        return Err(format!("{text:?}: min must be 1 or more and at most max"));
    }
    Ok((
        target,
        VersionScheme {
            tag_prefix,
            min_components,
            max_components,
            floor_major: None,
        },
    ))
}
