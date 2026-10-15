# Recorded release listings

⛔ **These are recorded source responses, not measurements of a client.** Each
one is what a vendor's release endpoint answered on the day it was read, reduced
to the fields this project's readers read. Nothing here is evidence about a
build: no file was fetched, nothing was installed, and no byte in these files
came from a running program.

They exist so
[`check-release-route.sh`](../check-release-route.sh) can prove each adapter's
asset pattern selects exactly one artifact without reaching three vendors. A
rule that can only be exercised over the network is a rule the gate cannot run,
and a gate that reached three vendors would go red the day one of them was down.

## What each one is

| file | recorded from | kept |
| --- | --- | --- |
| `aria2.json` | `https://api.gh.pkgforge.dev/repos/aria2/aria2/releases?per_page=100` | `release-1.37.0` alone |
| `aria2-next.json` | `https://api.gh.pkgforge.dev/repos/AnInsomniacy/aria2-next/releases?per_page=100` | `v2.7.4` alone |
| `aria2-next-unpublished.json` | the same response | `v2.7.5` and `v2.7.4`, in that order |
| `qbittorrent.json` | `https://api.gh.pkgforge.dev/repos/qbittorrent/qBittorrent/releases?per_page=100` | every release the response carried |
| `transmission.json` | `https://api.gh.pkgforge.dev/repos/transmission/transmission/releases?per_page=100` | `4.1.3` alone |

⛔ **`aria2-next-unpublished.json` is the one fixture here that records a vendor
mid-publish, and it was not constructed - it was caught.** Read on 2026-09-09,
`v2.7.5` had been published at `12:11:10Z` and carried **zero** assets 47 seconds
later, while `v2.7.4` beside it carried eight. A release exists from the moment
it is created and its binaries arrive when whatever builds them finishes, so
every target with a release route has a window in which its newest release is
empty.

⭐ **It keeps both releases because the question is whether the resolver falls
back, and a fixture holding the empty release alone cannot ask that.** The
fallback needs an older release with assets sitting right there to fall back to.
⚠ Rule 5 targets the newest stable release, so quietly resolving `v2.7.4`
because `v2.7.5` had nothing to offer would be wrong in a way that looks like
success. Measured: it refuses, and `check-release-route.sh` reads the version
back out of the refusal so "it refused" cannot be satisfied by refusing for some
other reason.

⭐ **qBittorrent's keeps every release on purpose**, because the newest release
object in that response is `release-5.3.0beta1` and the point of keeping it is
that the resolver does not select it. A fixture holding only the stable release
would prove the selection over an input with nothing to reject.

⚠ **The other two keep one release** because what they prove is the asset
pattern, and `ACQ-02`'s own suite already covers ordering over a long candidate
list. aria2's full response is 420 kilobytes and transmission's is a megabyte,
almost all of it release-note prose belonging to somebody else.

## Why they are reduced, and what the reduction is

⛔ **The projection is by construction everything the readers read.** It is
produced by deserializing into the same shapes
[`sources::github_releases`](../../../crates/bit-ids/src/resolution.rs) and
`sources::github_release_assets` use and re-emitting them, so a field a reader
needs cannot be missing from a fixture: the projection would not have compiled
past it. A hand-transcribed listing would risk exactly the transcription error
that the asset names under test are.

⭐ **That property was paid out on 2026-09-17.** `ACQ-06` taught the asset reader
to read the per-asset `digest` the source publishes, and these fixtures carried
none - so every route resolved `unpublished` over three sources that publish one.
Adding the field to `project-listing` and re-recording at the SAME tags was the
whole repair, and the asset names under test did not move.

⚠ **`aria2.json` states no digest and that is the source's answer, not a gap.**
Read through `AGENTS.md` rule 8's route on 2026-09-17: all six of
`release-1.37.0`'s assets carry `digest: null`, where the other three targets
carry one on every asset. ⛔ So the fixtures hold both shapes, which is what lets
`check-release-route` assert three different dispositions instead of one.

⛔ **`aria2-next-unpublished.json` is deliberately NOT re-recorded**, and its
assets state no digest for that reason. It records a vendor mid-publish - `v2.7.5`
with zero assets - and that release has had its eight assets since. Re-recording
it would destroy the moment it was kept for.

## What they do not establish

⚠ **Nothing about what those vendors publish today.** A fixture's bytes provably
did not move, which is what makes a changed answer over one the fault of this
project rather than of a vendor - and it is also why passing here says nothing
about the release that exists this morning. Only
`bit-check resolve-release` without `--listing` asks a vendor, and only a dispatch
installs anything.

⚠ **And the digest of a listing identifies a response, not a release.** Two
fetches of one endpoint minutes apart differ, because the counters GitHub keeps
beside an asset move. Measured on 2026-09-09: two reads of the aria2 endpoint
inside one session produced two different digests over one unchanged release.

## Regenerating

Fetch, then project. Both commands are in this tree and neither invents a
value:

```sh
"$BIT_CHECK" fetch-releases aria2/aria2 /tmp/aria2-full.json
cargo run --quiet -p bit-ids --example project-listing -- \
  /tmp/aria2-full.json \
  "https://api.gh.pkgforge.dev/repos/aria2/aria2/releases?per_page=100" \
  release-1.37.0 >scripts/acquisition/listings/aria2.json
```

⭐ **`project-listing` takes more than one tag**, which is what lets a fixture
hold two releases in the vendor's own order:

```sh
cargo run --quiet -p bit-ids --example project-listing -- \
  /tmp/aria2-next-full.json \
  "https://api.gh.pkgforge.dev/repos/AnInsomniacy/aria2-next/releases?per_page=100" \
  v2.7.5 v2.7.4 >scripts/acquisition/listings/aria2-next-unpublished.json
```

⛔ **Every named tag must match or the projection is refused.** A non-empty
result is not the result that was asked for: two tags of which one is a typo
keep one release and look exactly like a fixture meant to hold one.

`project-listing` prints the digest of the full response it reduced, on stderr,
so a regeneration can be compared against the one it replaced.

⛔ **A regeneration changes what the harness expects.** The asset a pattern
selects is named in `check-release-route.sh`, so re-recording a listing at a
newer release means updating that name in the same change - which is the point:
a vendor who renames an artifact should turn the gate red rather than quietly
change what a route installs.
