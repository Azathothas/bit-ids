# Publishing

Nothing is published yet. This document is the contract the `PUB-*` and
`CI-*` entries implement.

## Data branch

The `data` branch is append-only and contains only generated publication
artifacts. It is never force-pushed. A run that assembles identical bytes
pushes nothing.

```text
LICENSE
MANIFEST.json
SHA256SUMS
raw/v1/<target>/<version>/<platform>/<arch>/<package>/<capture-id>/manifest.json
raw/v1/<target>/<version>/<platform>/<arch>/<package>/<capture-id>/...
profiles/v1/<target>/<version>/<platform>/<arch>/<package>/<capture-id>.json
indexes/v1/profiles.json
formats/bit-ids-v1.json
formats/bit-ids-v1.jsonl
formats/bit-ids-v1.csv
formats/bit-ids-v1.columns.json
formats/bit-ids-v1.sqlite3
formats/bit-ids-v1.cbor
```

⭐ **Every path here is written.** `PUB-05` closed the last one,
`bit-ids-v1.sqlite3`, and it is the only rendering that needs a third-party
encoder: [`../crates/bit-ids/src/sqlite.rs`](../crates/bit-ids/src/sqlite.rs) is
the one file that does. ⛔ **It is not behind a feature and cannot be**, because
`PUB-04` derives a consumer's caching contract from the set of published paths,
so a build that could omit one would publish a manifest with a hole in it and
say nothing.

⭐ **The database is the only rendering that is both queryable and lossless.**
The CSV cannot hold the acquisition routes, the observations, the corroboration
or the evidence list, and publishes a columns document beside it saying so; this
one tabulates all four and keeps each record's canonical bytes in a `document`
table as well, so a consumer can re-derive the published record from the file
rather than trusting its columns. ⚠ What it does not tabulate it says in an
`omission` table, because a database can describe itself and a CSV cannot.

⛔ **`routes/v1/<target>/<version>/<platform>/<arch>.json` was in this layout and
has been removed, because it cannot be written in that shape.** It omits
`<package>`, so the routes of a `deb` and an `AppImage` of one version on one
platform would be two different route sets at one file name: the same
non-injective layout `CORPUS-01` found in the profile path and for the same
reason. Nothing ever wrote it, and `PUB-04` is what noticed, by deriving the
documented path set from an assembled release rather than reading it here.
⚠ A per-route view is still worth having; it needs the identity tuple in full,
and it is a new rendering rather than a path this document can promise.

⛔ **A record's path carries its whole identity tuple, and `<package>` is in it
because the record identifier digests it.** This layout omitted that segment
until `CORPUS-01` derived the path in code and compared the two: a `deb` and an
`AppImage` of one version on one platform are two records the identifier tells
apart, and they were one file. Whether they also differ in capture identifier is
not the question, because `capture.id` is only documented unique per target,
version, platform and architecture, so the collision was resting on a uniqueness
rule nothing states or checks. The derivation is
[`store::StoreKey`](../crates/bit-ids/src/store.rs) and it is the only place a
path is composed.

⛔ **A correction appends and never edits.** The record it corrects keeps its
path and its bytes; `indexes/v1/profiles.json` is where it stops being the
answer, and that document's `corrections` list is how a consumer holding the old
identifier finds the one that answers now. `CORPUS-04` owns it.

`latest` selects the newest validated stable version only. It is a generated
pointer, not a profile, and a prerelease can never move it.
[`index`](../crates/bit-ids/src/index.rs) derives it and the lookup indexes
beside it, with `cargo run -p bit-ids --example build-indexes -- STORE OUT` as
the driving surface and
`bit-check check-indexes`, in
[`../tools/check/indexes.go`](../tools/check/indexes.go), as its prover. ⚠ It
selects nothing at all for a target whose version scheme is not declared, rather
than ordering under an assumed one.

## Append-only, and what checks it

A published path never changes and never disappears; a correction appends a
record carrying `supersedes`. [`store`](../crates/bit-ids/src/store.rs) holds
both halves of that as `E-STO-*` refusals: the structural rules a tree must
satisfy to be checked out at all on every platform in the matrix, and the
comparison between a published tree and the successor a run proposes.

`cargo run -p bit-ids --example check-store -- PRIOR NEXT` is the driving
surface, and [`scripts/corpus/check-store.sh`](../scripts/corpus/check-store.sh)
plants each refusal in a disposable tree and reads the exit code back.

Whether a tree is a coherent corpus at all is the separate question
[`corpus`](../crates/bit-ids/src/corpus.rs) answers, under `E-CRP-*`, with
`cargo run -p bit-ids --example validate-corpus -- STORE` as its driving surface
and `bit-check check-corpus`, in
[`../tools/check/corpus.go`](../tools/check/corpus.go), as its mutation prover. ⭐ `build-store` writes a store for either to be pointed at.

⚠ **A version is not a path segment and the store is what says so.**
[`Version`](../crates/bit-ids/src/canonical.rs) accepts whatever the installed
build printed, `../../etc` included, because imposing a grammar on a measurement
would refuse builds that number themselves some other way. `E-STO-01` blocks
publication for a version that cannot be a segment rather than mangling it into
one, because an escape that is not injective merges two measurements into one
directory.

## Assembly

One Rust assembler builds the publication tree once and reads no wall clock. The
data-branch job and release job consume the same uploaded tree.
[`release`](../crates/bit-ids/src/release.rs) is that assembler, with
`cargo run -p bit-ids --example assemble-release -- DIR` as its driving surface
and `bit-check check-release`, in
[`../tools/check/release.go`](../tools/check/release.go), as its prover.

⛔ **What it consumes is a tree built from the STORE.** A capture bundle is
evidence and a publication is assembled from records, and the distance between
those two was the whole of `CI-09`. ⭐ Since 2026-10-14 one dispatch closes it:
[`assemble-bundle.yml`](../.github/workflows/assemble-bundle.yml) downloads one
`capture-client` run's artifacts, and `bit-check assemble-dispatch`, in
[`../tools/check/dispatch.go`](../tools/check/dispatch.go), writes the
records and their run manifests with `assemble-capture`. Then
`bit-check assemble-bundle` refuses a store
that is not a valid corpus and composes the indexes, the renderings, the
licence and the two documents above. The result is uploaded as `bundle`.
`capture-client` run 38's `aria2-next` 3.0.3 pair is the first bundle, 30
files, with both records admitted to the published view.

⚠ **The producer was absent on purpose until a measured record existed.** While
every record was synthetic, a bundle would have left a publishable-looking
artifact one boolean away from the data branch. The publisher is still a
separate dispatch whose dry run is its default, and it names the bundle's run.

⛔ **The publisher refuses a bundle that does not describe itself before it
pushes.** One with no `MANIFEST.json` or `SHA256SUMS`, or a digest that does not
match, used to be pushed and refused only at the read-back. The first bundle
assembled from a real capture found that on 2026-10-14.

⚠ `MANIFEST.json` describes every file's media type, schema and SHA-256 digest
except its own and the checksum file's, and `SHA256SUMS` covers every file
except its own, the manifest included. Neither can state its own digest, so
between them each published byte is covered once.

CSV is a lossy tabular view and names which nested fields it omits. JSON and
CBOR carry the complete normalized model. SQLite provides indexed tables and
foreign-key integrity. Raw binary evidence is never embedded into CSV.

[`formats`](../crates/bit-ids/src/formats.rs) renders them and `PUB-03` owns it,
with `cargo run -p bit-ids --example build-formats -- STORE OUT` as the driving
surface and `bit-check check-formats`, in
[`../tools/check/formats.go`](../tools/check/formats.go), as its prover.

⛔ **Every rendering is a function of the canonical document.** The combined
JSON carries each record's own bytes verbatim, so slicing one out yields exactly
what was published; the others are produced by reading that document back. A
rendering cannot then carry a field the published record does not. ⚠ Which
records are rendered is not decided here either: it is
[`index`](../crates/bit-ids/src/index.rs)'s answer, so a record a correction
retracted leaves the tabular view at the same moment it leaves the lookups.

⚠ **`bit-ids-v1.columns.json` is how the tabular view names its omissions.** A
consumer reading only the CSV would otherwise have no way to learn that the
acquisition routes and the evidence list exist.

## Workflow permissions

[`publish-data.sh`](../scripts/publishing/publish-data.sh) is the publisher and
[`check-publish.sh`](../scripts/publishing/check-publish.sh) drives it against a
bare repository created for the run. ⛔ It has never run against this
repository's own remote. A measured record exists since 2026-10-14, and the
first publication is the operator's call.

[`publish-data.yml`](../.github/workflows/publish-data.yml) is what calls it,
and `CI-01` wired it. ⛔ **It has no automatic trigger and its dry run is the
default**, so it cannot fire on its own and dispatching it by accident still
pushes nothing; both are cases in
[`check-workflow.sh`](../scripts/ci/check-workflow.sh) rather than assurances in
prose. The token reaches git as a header rather than inside a remote URL, which
is the form `check-no-secrets` refuses in this tree.

- capture lanes: `contents: read`, no secrets, upload artifacts only;
- collector/validator: `contents: read`, cannot push;
- data publisher: job-scoped `contents: write`, append-only push without force;
- release publisher: job-scoped `contents: write`, triggered only by a pushed
  `v1.*` tag;
- pull requests from forks: no write token and no secrets.

Every third-party action is pinned to a full commit. Dependency automation
proposes pin updates; CI resolves the commit and checks the action runtime.

## Release assets

A tagged release will contain the manifest, checksums, each generated format,
and deterministic `.tar.gz` and `.zip` archives. A published tag or asset is
immutable. `latest` is the hosting platform's release pointer; no Git tag is
moved.

## Access paths, and how long each is good for

`PUB-04` owns this and
[`access`](../crates/bit-ids/src/access.rs) is where it is derived, with
`cargo run -p bit-ids --example access-contract -- TREE` as the driving surface
and [`check-access.sh`](../scripts/publishing/check-access.sh) as its prover.

⛔ **A consumer is told two things per path and both are derived, not
declared.** How long the bytes are good for, and which published document proves
them.

| class | which paths | what a consumer may do |
| --- | --- | --- |
| `immutable` | everything under `profiles/` and `raw/` | cache the bytes forever; the path never changes and never disappears, and a correction appends a new one |
| `current` | `MANIFEST.json`, `SHA256SUMS`, `LICENSE`, `indexes/`, `formats/` | refetch and compare the digest; these exist in order to change |

⛔ **The class comes from `store::CANONICAL_ROOTS` and never from a second
list.** That constant is already what the append-only rule is about, so a path
is immutable exactly when the publisher refuses to rewrite it. A second
enumeration here would be the same rule spelled twice, and the copy that drifted
would be the one telling a consumer to cache a file that moves.

⚠ **A path this build cannot classify blocks the contract** under `E-ACC-01`,
rather than being published with a guessed stability. That is `PUB-01`'s
media-type rule applied to caching, and it is what refused the `routes/` path
above.

| integrity | which paths | what proves the bytes |
| --- | --- | --- |
| `manifest` | everything except the two below | `MANIFEST.json` carries the digest, and `SHA256SUMS` carries it too |
| `checksums` | `MANIFEST.json` | `SHA256SUMS`, because no document states its own digest |
| `out_of_band` | `SHA256SUMS` | nothing in the publication. A consumer takes it from the release asset listing or from a digest it recorded on an earlier fetch |

⛔ **Exactly one published file is covered by neither document**, and a consumer
that assumed either covered everything would find the gap precisely where the
other one is. `exactly_one_published_file_is_covered_by_neither_document` is what
pins that to one file.

### The URL forms

A path in the contract is relative to the `data` branch. ⚠ **No host appears in
the crate**: a public URL built from a hardcoded host is dead everywhere except
the machine that made it, so a caller composes one from a base it was given.

```text
raw bytes   https://raw.githubusercontent.com/<owner>/<repo>/data/<path>
API         https://api.github.com/repos/<owner>/<repo>/contents/<path>?ref=data
mirror      https://api.gh.pkgforge.dev/repos/<owner>/<repo>/contents/<path>?ref=data
```

⛔ **None of those has ever been fetched, because nothing has ever been
published.** What `check-access.sh` proves is the path set, the two classes and
every digest, against a publication pushed to a bare repository and fetched back
over git. The URL forms above are unexercised and `TODO/publishing.md` carries
that as a residual with the event that closes it.

⭐ [`consuming.md`](consuming.md) is the same contract written for the reader
rather than for the publisher, with every command on it run on each gate.

## Read-back

A successful push is not completion. The workflow fetches the remote branch,
compares the commit and tree object it intended to publish, then verifies every
manifest digest. Release creation similarly reads the release and asset list
back from GitHub.
