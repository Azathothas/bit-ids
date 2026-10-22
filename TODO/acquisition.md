# Acquisition entries

## ACQ-01: Acquisition route schema

Source: operator same-version multi-source requirement
Priority: P0 | Effort: L | Status: DONE

Problem: Package, official download, release artifact, and source-build routes
need comparable provenance despite different metadata.

Premise: Resolver input, immutable source identity, retrieved bytes, installed
version output, and host facts form a common route record.

Approach: Define typed route kinds and evidence requirements; do not treat the
candidate routes in the catalogue as verified availability.

Decision: the route record is one shape with a typed source identity, not one
record shape per kind. What the eight kinds have in common is the shape of the
claim, which is what the premise says: something decided a version, something
delivered bytes, the bytes have a digest, and the installed thing was asked its
own version. Only the identity of what was asked for genuinely differs, so that
is the enum and the rest is flat. Eight parallel record types was the rejected
alternative: it would have made every consumer match on the kind to read a
digest.

Decision: `SignatureStatus` moved out of the manifest module into the route
module. Both documents record it and `bind` compares them, so a second copy of
the enum is a value in two places that can disagree about what its own variants
mean.

Decision: the catalogue-vocabulary test scans `clients.toml` rather than adding
a `toml` dependency. One hand-maintained file read by one test does not carry
the argument the supply-chain rules require for a third-party crate. The scan
fails loud instead of quietly finding nothing: it refuses a result too small to
be real, and that guard was planted against by renaming the key it reads.

Prove: schema tests cover all catalogue route kinds and reject an acquisition
without a content digest or observed installed version.

Closure evidence: run on 2026-09-04. `cargo test --workspace --locked
--all-targets` is 123 passed, 0 failed. `tests/acquisition.rs` asserts every
`candidate_routes` entry in the catalogue parses as a `RouteKind` and every
variant is asked for by the catalogue, over 42 scanned entries. A content digest
and an installed version are non-optional fields, so a record without either
fails to deserialize; what needed new invariants was everything around them.

⭐ The route record grew six invariants, each planted against in
[`../crates/bit-ids/tests/profile_schema.rs`](../crates/bit-ids/tests/profile_schema.rs):

- `E-ACQ-05` a kind carrying another kind's source identity, which reads as a
  complete record and names nothing anyone can resolve again;
- `E-ACQ-06` an abbreviated commit, the shape `FOUND-02` measured passing an
  action-pin rule written to refuse floating refs;
- `E-ACQ-07` and `E-ACQ-08` two routes sharing a resolver or a delivery
  mechanism. ⛔ This is the one that matters. Without it the two-route rule is
  satisfiable by asking one index twice under two names, which is the failure
  `architecture.md` section 7 exists to prevent rather than a technicality;
- `E-ACQ-09` an installed version citing evidence that is not in the record;
- `E-ACQ-10` one citing evidence that is not process output;
  [`../docs/architecture.md`](../docs/architecture.md) section 7 carries why.

`E-BND-13` compares the signature disposition across the two documents. It was
written after checking it can fail, which `E-BND-10` could not: nothing else
forces the run's record and the profile's into step, so `verified` over a run
that says `not_checked` is publishable without it.

Driven pass: the two example validators were run against the golden pair, which
they accept, and against three hand-planted records. Every refusal named the
route, the field and the reason. ⚠ It found a real defect the suite could not:
`E-BND-13` printed the Rust `Debug` spelling, telling an operator the run
recorded `Unsigned` over a document that says `unsigned`, and sending them to
look for a value written nowhere. `SignatureStatus` now has one canonical
spelling and the diagnostic quotes it.

⚠ Found while wiring the bind check: `E-BND-12` was already taken by the
capture-instant comparison, and the new code silently reused it. Two checks
under one code is worse than an unclear message, because a caller acting on the
class cannot tell which fired. Renumbered to `E-BND-13`, and the manifest
coverage test refused the change until it had a planted defect.

Residual: `origin` is an HTTPS URL for every kind, including a package-manager
route where the natural reference is a pool path rather than a page. That has
held for every catalogue route so far and is recorded here rather than assumed
to hold forever.

Residual: a bare `serde_json::from_str::<AcquisitionRoute>` produces an
unvalidated route. That is true of `Build` and `Capture` too and is not a new
door: the route-level invariants need the build tuple and the evidence list, so
a route alone cannot answer them. `Profile` remains the validated boundary.

## ACQ-02: Latest stable release resolver

Source: operator latest-stable-only scope and bit-cli tag-selection guards
Priority: P0 | Effort: L | Status: DONE

Problem: Lexical tag sorting and package channel labels can select previews,
older trains, or incomparable versions.

Premise: Stable selection must be product-specific, time-stamped, fail closed,
and retain every candidate considered.

Approach: Implement Rust resolvers with explicit prerelease rules and shell
fetch wrappers. Record source responses; never derive profile fields from
release source code.

Decision: the shell half fetches and nothing else. It writes the bytes to a
file and prints the URL that answered; the Rust half parses, orders and decides.
That is the line `../scripts/README.md` already draws, and it is what makes the
digest in a resolution mean something: it is of what arrived, not of what a
parser reconstructed. A shell script that also picked the newest version would
be a second implementation of the ordering rule with no test behind it.

Decision: an unorderable candidate blocks the resolution rather than being
skipped, and only a second signal releases it. Skipping was the rejected
alternative and it is the dangerous one: it produces an older version selected
confidently, with nothing saying a newer one was seen and not understood.

Decision: no date crate. One function formats an instant, and a dependency for
that is one this project would have to argue for under
`../docs/supply-chain.md`.

Prove: resolver fixtures cover stable, prerelease, missing, divergent, and
non-semantic version sets, and a live dry run emits a complete decision trace.

Closure evidence: run on 2026-09-04. `cargo test --workspace --locked
--all-targets` is 146 passed, 0 failed.
[`../crates/bit-ids/tests/resolution.rs`](../crates/bit-ids/tests/resolution.rs)
carries all five sets the acceptance names and eight more, with a planted defect
for every diagnostic code.

⭐ Live dry run, four real targets, through the route `docs/AGENTS.md` rule 8
prescribes, `https://api.gh.pkgforge.dev/`. Direct `api.github.com` answered 403
from this host, which is the reason that rule exists.

| target | tag shape | candidates | selected | trace |
| --- | --- | ---: | --- | --- |
| `qbittorrent` | `release-` + 3 or 4 | 4 | `5.2.3` | 1 selected, 3 superseded |
| `transmission` | bare, 3 | 83 | `4.1.3` | 1 selected, 10 superseded, 21 prerelease, 51 predating |
| `aria2` | `release-` + 3 | 24 | `1.37.0` | 1 selected, 23 superseded |
| `qbittorrent-enhanced` | `release-` + 4 | 100 | `5.2.3.10` | 1 selected, 99 superseded |

⚠ **The driven pass changed the design, which is what it is for.** The first
live run against `transmission` failed closed over 51 candidates and was
correct to: they are two-component tags from a decade ago that a
three-component scheme cannot read, and nothing in the resolver could rule out
that one of them was newer. Refusing over a project's own history made the
resolver correct and useless. The fix is not a loosened rule but a second
signal: a candidate published strictly before the winner cannot be the newest
whatever its tag says, so it is recorded as `predates_selection`. One with no
date, or a date at or after the winner's, still blocks, and both cases are
tested.

Two more findings, both fixed:

1. **`4.1` and `4.1.0` were ordered rather than compared equal.** A shorter
   component vector sorts first, so the resolver called them different versions
   with the longer one newer. They are one release. Components are now padded to
   the scheme's width, which makes the pair ambiguous and fails closed. Found by
   writing the ambiguity test and watching it select.
2. **`E-RES-01` checked a schema identifier `from_json` could never reach**,
   because the version probe answers first. The schema became a type that cannot
   hold a wrong value, the same construction the manifest already used, and the
   invariant was deleted rather than left as a guard nobody knows works. That is
   the path `E-BND-10` took in `SCHEMA-02`.

⭐ Door sweep, three findings, all fixed:

1. **`Verdict::excludes_from_ordering` had no callers.** It was a leftover from
   an earlier shape of `decide`, and a little dead code is a safety margin only
   when it is one. Deleted.
2. **`Verdict::as_str` and serde's `rename_all` were two spellings of one
   vocabulary with nothing comparing them**, which is the drift refused
   everywhere else here. A test now holds them together and refuses a variant
   added without a spelling.
3. ⛔ **`retrieved_at` was the response file's modification time.** An mtime
   survives a copy, an archive restore and a checkout, so a published retrieval
   instant would have been one no retrieval produced. The fetch wrapper now
   writes the instant beside the body and the resolver reads that, refusing to
   proceed when it is absent.

Residual: `min_components` and `max_components` describe a target's current
tagging convention, and a project that changes convention will produce
unorderable candidates until the scheme is updated. That is the intended
behaviour rather than a gap: it fails closed and says so. The catalogue does not
yet carry a scheme per target; `ACQ-03` needs one to compare two routes and is
where it lands.

Residual, closed 2026-10-15 by `CI-11`, which ported it to `bit-check fetch-releases`: `fetch-releases.sh` has no PowerShell twin. `../scripts/README.md`
carries why, and `ACQ-04` owns the Windows runner contract where one is needed.

### ⭐ A SECOND SOURCE FORMAT, added 2026-09-09: a repository's own refs

⛔ **This resolver had one reader and that made two routes impossible.**
`read_source` parsed GitHub release listings and nothing else, so every route in
this project resolved through one index - and `E-ACQ-07` refuses two routes that
share a resolver. Measured by assembling `capture-client` run 14: its two lanes'
resolution records differ only in their timestamps.

⭐ **`sources::git_refs` reads `git ls-remote --tags --refs`**, and the source id
now chooses the reader: `github-releases` or `git-refs`. ⚠ One value therefore
says both what answered and how it is read, where a separate flag would be a
second place for those to disagree; two sources of one format are told apart by
their URLs, which is what a route records and `E-ACQ-07` compares.

⛔ **Two things a refs source cannot do, stated rather than discovered later.**
A releases listing carries `prerelease` and `draft` and refs carry a name, so a
prerelease this project could only recognise from a flag would be selected -
what survives is the version text, which is what caught `release-5.3.0beta1`.
And refs carry no date, so `predates_selection` cannot release a candidate whose
tag no scheme can order: it blocks, which is the resolver failing closed and is
correct, because nothing can rule out that the unreadable tag is the newest.

⭐ **A target can now declare a second signal that needs no date: a scheme
floor.** `release_floor_major=4` says the scheme speaks for 4.x and later, so an
unorderable tag whose leading component is a plain integer below 4 is recorded
as `below_scheme_floor` and does not block, and only against a winner at or
above the floor. `E-RES-09` re-derives each such verdict from the recorded
scheme and `E-RES-10` refuses a selection below the floor that released
anything. It was `bit-ids/resolution/2`. Every other adapter declares `-`.
Record: [`clients.md`](clients.md) `CLIENT-06`.

⭐ **And a release series, for a target that keeps several stable series
current.** `release_series=2.0` says the scheme speaks for the 2.0 line alone:
a stable tag whose leading components differ readably from it is
`other_series`, and does not block or compete, and a version outside it is
one the scheme cannot order, so the published view never picks across series.
`E-RES-11` re-derives each such verdict, `E-RES-12` refuses an ordered
candidate outside the series and `E-RES-13` a series that leaves nothing to
order. It is `bit-ids/resolution/3`. `libtorrent` declares `2.0`; every other
adapter declares `-`, and over every adapter's live refs and listings on
2026-10-22 no other verdict moved. Record: [`engines.md`](engines.md)
`ENGINE-01`.

⚠ **`git ls-remote` is git-over-HTTPS rather than a REST read**, so `AGENTS.md`
rule 8's route does not apply. Measured on 2026-09-09: the direct call answered,
and the record says which route answered.

⚠ **A tag is not a commit.** `--refs` yields the tag object for an annotated tag,
not the commit it points at - `v2.7.5` is `5310afa8…` and the clone built
`a9784ea8…`, which is why the adapter's own `git rev-parse HEAD` after the clone
is what `E-ACQ-06` gets its object name from.

#### ⭐ Driven against the live repository, 2026-09-09

`resolve-source.sh` over `AnInsomniacy/aria2-next` answered **`v2.7.5`** in two
seconds, recording `source_url=https://github.com/AnInsomniacy/aria2-next.git`.

⭐ **The two indexes agreed, and agreeing is now a measurement rather than a
construction.** The release lane's listing selected 2.7.5 and the refs selected
2.7.5 from a separate read of a separate endpoint - which is what absolute 4
asks for, and what one shared resolution made unaskable.
⭐ **And the two resolver identities now differ**, which is what `E-ACQ-07`
compares: `assemble-capture` slugifies the release route's
`…/repos/AnInsomniacy/aria2-next/releases` and this route's
`https://github.com/AnInsomniacy/aria2-next.git` into two names.

⚠ **Unproved on a runner.** No dispatch has taken the new *Resolve the source
tag* step, so what is established is the resolver, the record it writes and the
harness - not a lane.

#### ⛔ Two defects the harness found before either shipped

⛔ **The tag reader took the wrong tag.** It scanned forward from the `selected`
verdict and took the next `tag` it met, and `tag` comes BEFORE `verdict` inside
an entry - so against run 14's real resolution it printed `v2.7.4` over a
resolution that had selected 2.7.5. It is the reader defect `CI-09` records in
`check-project`'s first artifact rule, arriving in a different file. The tag is
remembered per entry now, with the entry boundary resetting it rather than a
distance assumed in either direction, and the harness's control puts the winner
last so either wrong reading fails it.

⛔ **And a could-not-run discarded the reader's own words.** The first version
printed `resolve-stable`'s stderr for a refusal and swallowed it for exit 2, so
a listing this project could not read reported "could not run" with the sentence
naming the bad line sitting in a workdir file nobody opens. That is the defect
`CLIENT-06` records in `install-client`, where a caller discarded an adapter's
stderr and then reported its absence. ⚠ Found by a case asserting the REASON
rather than only the code - three of these refusals share exit 2, and a case
reading the code alone passes when a different one fired.

### ⛔ Residual, closed 2026-09-09: the resolver named a version and nothing named an artifact

**This entry decided which version to acquire and the acquisition still could not
start**, because every `release` route in the tree refuses without a URL
"resolved before the route was cut" and nothing in this repository resolved one.
`capture-client.yml` therefore passed `--route package` and only that, through
four dispatches, and the second route existed as three adapters that could not be
called.

⭐ **`select_asset` and `AssetPattern` are the missing half**, beside `resolve`
because they answer the second half of one question, and
`scripts/acquisition/resolve-release.sh` (`bit-check resolve-release` since 2026-10-15)
composes the fetch, the resolution and the selection into the one command a
workflow step runs.

⛔ **The asset comes out of the same response the version did.** One retrieval
answers both questions, so the digest the resolution already records covers both
decisions, and `select-asset` takes the tag off the candidate that actually won
rather than re-deriving a version of its own.

⚠ **Which asset is target knowledge, so the adapter declares it** - the
repository, the tag prefix, the component bounds and the pattern, all through
`describe`, which both callers already parse. The parsing stays in Rust.
`scripts/capture/adapters/README.md` carries the five keys.

⭐ **`{version}` in a pattern is what makes it unambiguous on a real release, and
that is measured rather than argued.** Against the live listings on 2026-09-09:

| target | assets in the newest stable release | a `*` pattern matches | the pinned pattern matches |
| --- | ---: | ---: | ---: |
| `aria2` 1.37.0 | 6 | 3 (`aria2-*.tar.*`) | 1 |
| `qbittorrent` 5.2.3 | 14 | 2 (`qbittorrent-*_x86_64.AppImage`) | 1 |
| `transmission` 4.1.3 | 11 | 1 | 1 |

⛔ **Two matches refuses rather than taking the first.** Choosing by the order a
source listed its assets is choosing by a property of the source, and the record
would look identical the month that order changed. ⚠ The version is expanded into
a **literal** rather than spliced into the glob, because `Version` accepts what a
build printed and a value carrying a `*` would widen the pattern meant to pin it.

⭐ **Driven end to end on this host, with the strongest control available.** The
URL the selection produced was fetched and `sha256sum -c` verified it against the
`sha256:a75c85…` the two-route table below recorded on 2026-09-08, from a
different fetch in a different session. The declared size, 2392193, is the size
on disk. ⚠ Fetching a source tarball is not a capture and needs no disposable
host.

⚠ **The digest is quoted short here and in full in that table**, because
`check-no-secrets --public` hunts a bare forty- or sixty-four-digit hex run and
is right to: this project spells a digest with its algorithm, and a value written
without one is not the canonical form. The check refused this paragraph's first
draft.

⚠ **A listing's digest identifies a response and not a release.** Two reads of
the aria2 endpoint minutes apart inside one session produced two different
digests over one unchanged release, because the counters the source keeps beside
an asset move.

Residual: the selection is not in the `Profile`. `ReleaseAsset` carries the name,
the location and the declared size, and `AcquisitionRoute`'s `ReleaseAsset`
identity carries a repository, a tag and an asset name - so a record can say
which asset it asked for and cannot yet say what the source declared about it.
Whatever first assembles a publication from two real captures is where they meet.

## ACQ-03: Same-version multi-route verifier

Source: operator requirement to install the same build from two or more routes
Priority: P0 | Effort: L | Status: DONE

Problem: Equal display versions can identify repackaged or patched binaries,
while different version formats can identify the same upstream build.

Premise: Installed version output, upstream version mapping, artifact hashes,
binary metadata, and observed profile comparison can make equality explicit.

Approach: Require at least two independent routes on the same host family;
classify byte-identical, build-equivalent, divergent, or unresolved and block
publication for the last two.

Decision: the executable digest moved from the record to the route, and the
capture now names which route's install was observed. One digest per record
collapsed exactly the difference this entry exists to detect, and a record that
did not say which install was watched let a reader assume both were. Everything
else here follows from those two fields.

Decision: a single capture of two byte-different installs is `unresolved`, not
`build_equivalent`. That was the fork. Calling it equivalent is what the version
labels invite, and it publishes a claim about bytes this project never put on
the wire. Reaching `build_equivalent` needs a capture per route, which is what
`classify_across` compares, and the rejected alternative would have made the
whole classification decorative.

Decision: two refusal codes rather than one. A divergence needs adjudicating and
an unresolved record needs a second capture; collapsing them into "not
publishable" loses which.

Decision: the rule is asked from `publishable` rather than beside it. A second
publication gate a caller has to remember is the shape that lets a record ship
past one of them.

Prove: an end-to-end fixture accepts two equivalent routes and rejects version
label equality when binary or observed identity evidence conflicts.

Closure evidence: run on 2026-09-04. `cargo test --workspace --locked
--all-targets` is 157 passed, 0 failed.
[`../crates/bit-ids/tests/equivalence.rs`](../crates/bit-ids/tests/equivalence.rs)
accepts the golden pair as `byte_identical` and publishes it, and refuses four
distinct ways of being wrong: equal labels over installs only one of which was
observed, a field both captures measured disagreeing, two captures of one route,
and one record passed twice.

Driven pass: the golden pair validates and publishes. A copy whose second route
installed different bytes still **validates** and refuses to publish with
`E-PUB-04`, naming both digests and the route that was watched. That split is
the point: the difference has to be recordable or there is nowhere to keep the
evidence of it. A copy whose observed route does not match the record's
executable digest is refused outright by `E-CAP-06`.

⭐ Door sweep, one finding, fixed: `classify_across` guarded against comparing
two captures of one **route** and not against comparing two records of one
**run**. One record passed twice would have agreed on every field for the most
trivial reason available. It is refused now, and the test that was meant to
cover the route case was passing through the wrong guard until the capture ids
were made distinct.

⚠ Found while writing the divergence test: the first version set the field it
was changing to the value the fixture already carried, so the records agreed and
the test passed for the wrong reason. An `assert_ne!` against the current value
now stands in front of the edit. That is the "when a negative test passes
suspiciously well, doubt the harness" rule firing.

Residual: the manifest carries no `observed_route`, so `bind` covers it only
transitively, through the route sets already having to match. That is enough
today because both documents list the same routes; a manifest that recorded
which install was driven would let `bind` compare it directly, and `CI-03` is
where a capture matrix would produce one.

⭐ **Residual closed, 2026-09-15: `classify_across` is a gate now rather than a
function.** It said `CORPUS-02` was where the cross-record comparison would stop
being something nobody calls, and the reason it mattered turned out to be
sharper than the residual put it: **`E-PUB-04` could not be closed by any
capture at all**, because `classify` never reaches `build_equivalent` and every
publication gate asked the per-record question one record at a time.
`agreement::publishable_among` asks it with the store, and the three gates that
hold a corpus plus `assemble-capture` all use it. [`ci.md`](ci.md) under `CI-09`
carries the cases and the mutation pass.

⛔ **Residual, found 2026-09-08 and half-fixed: this whole classification assumes
each route installed something, and nothing established that.** A route that
installs nothing exits 0, so two routes on a host that already ships the product
declare two independent resolvers, satisfy `E-ACQ-07` and `E-ACQ-08`, agree on
the version because it is one binary, and arrive here as `byte_identical` - the
strongest verdict this entry has, reached by acquiring nothing.

### Two routes were actually acquired for one target, measured 2026-09-08

⭐ **The first time this project has held two independent installs of one target
at once**, and it needed no disposable host, because building and asking for a
version is not a capture.

| | route A | route B |
| --- | --- | --- |
| resolver | Ubuntu 24.04 package index | the vendor's GitHub release |
| delivery | `apt-get install aria2` | `aria2-1.37.0.tar.bz2`, `sha256:a75c8534902ff1a64a23d7a47c033032329002553634251a89d53a9cb8b20efa`, configured and compiled |
| reported version | `aria2 version 1.37.0` | `aria2 version 1.37.0` |
| the executable | 14584 bytes, stripped, links `libaria2.so.0` | 2833064 bytes stripped, self-contained |
| cost | seconds | 19s configure, 126s `make -j4` |

⛔ **They report one version and they are not one build.** `aria2c --version`
lists what each was compiled with, and the two lists differ: the package build
enables Async DNS, Firefox3 Cookie, Metalink, XML-RPC and SFTP, and a source
build configured without those dependencies does not. ⚠ Features are what a
build DOES, so a record holding `1.37.0` and nothing else has dropped the only
evidence that would have distinguished them. ⭐ Every adapter's `version` writes
the build's whole answer to stderr now, which both callers already keep -
`version.err` beside the install record and `adapter.err` inside the bundle.

⛔ **And `installed_executable` records a SHIM for the package route.**
`/usr/bin/aria2c` from Ubuntu is 14 kilobytes and links `libaria2.so.0`; the code
that speaks the protocol is in `libaria2-0` at `1.37.0+debian-1build3`, whose
digest this record model has nowhere to put. ⚠ So the per-route digest comparison
this entry is built on is, for a library-split package, a comparison of two
launchers. Neither `byte_identical` nor `unresolved` is wrong about what it
measured; what is wrong is reading either as a statement about the implementation.
⛔ **Residual**, and the sharpest one this entry has: a route's evidence needs the
digest of what actually runs, not only of what was invoked.

⚠ **A source route's identity is not the tarball alone.** The same bytes
configured differently produce builds with different features, so the configure
options belong to the route the way a package's exact version does. Nothing
records them yet.

⛔ **And a source route can never reach `byte_identical`, which is a property of
the route rather than a failure.** Two builds of the same tarball at two prefixes
were compared here and differ: `configure --prefix` is compiled in, so the bytes
carry where they were going. ⚠ That is the argument this entry already makes for
`build_equivalent` needing a capture per route, arriving from the other side -
digest equality is unavailable for a route that compiles, so behaviour on the
wire is the only thing left to compare.

### What the install record carries now

⭐ **The verdict is a digest comparison rather than a version comparison.**
`install-client` asks `describe` before and after the route, records
`preexisting_binary`, `preexisting_binary_sha256`, `installed_binary` and
`installed_binary_sha256`, and reads `acquired=yes` when the target was absent,
when the version changed, **or when the version did not change and the executable
did**. ⛔ That third branch is not decoration: it is the aria2 pair. Driven on
2026-09-08, the release route on a host already carrying the package build
recorded one version, two paths, two digests and `acquired=yes`; the same run
before the digest branch existed had recorded `acquired=no` over a route that had
just compiled and installed a different program.

⚠ **This is the install record, not the `Profile`.** The record type in
`acquisition.rs` still carries one `installed_executable` per route and no
digest of what it links, so the residual above stands: the shell layer measures
it and the published record has nowhere to put it.

⚠ **Read out of `equivalence.rs` rather than inferred**: `classify` returns
`ByteIdentical` when every route's `installed_executable` matches the first, and
`ByteIdentical.publishable()` is true. The reason it records is *"every route
installed &lt;digest&gt;, so there is one build and observing one observed it"* -
which is exactly true, and exactly the wrong conclusion, when what actually
happened is that no route installed anything. ⛔ **The defect is not in this
function.** It is that nothing upstream could tell it the difference, because the
route type has no field for whether the install did anything.

⚠ **Measured, in the runs this repository already dispatched.** `aria2` ships on
`ubuntu-24.04`, so client capture runs 3 and 4 recorded `route=package` over an
`apt-get install` that installed nothing; and every `release` route in
`scripts/capture/adapters/` fetches its artifact without making it the executable
`version` asks, so a release route reports the package build. Two routes, one
binary, and every existing check satisfied.

⭐ **The half that is fixed is the evidence.** `install-client` asks the host for
a version before the route runs and records `preexisting_version` and `acquired`,
so a route that installed nothing says so in its own record; the three shapes and
a planted derivation are cases in `check-capture-client`.

### ⭐ And the refusal exists now, 2026-09-10, at the layer this residual named

⛔ **`assemble-capture` refuses a lane whose install record says `acquired=no`.**
That is where this residual pointed - "whatever first assembles a publication
from two real captures" - and it is the right layer for a reason stronger than
convenience: what is wrong is *upstream of the comparison*. A route that acquired
nothing is not a second route, so the record is never written and `classify`
never sees the pair.

⚠ **Proved by removing the probe rather than by reading the code.** With it
deleted, the `acquired=no` lane assembles into a record and the tool exits 0 -
which is the hole, demonstrated rather than argued. Two cases in
`check-assemble`, and the third value is its own case: `acquired` is written by
shell, so a record carrying anything but `yes` or `no` is a field nothing
derived, and reading an unknown value as `no` would refuse it for a reason that
is not true.

⛔ **THIS CLOSES THE HOLE AND NOT THE RESIDUAL.** `classify` still cannot see the
field, because `AcquisitionRoute` has nowhere to put it, so a record
hand-written past the assembler still reaches `ByteIdentical` by acquiring
nothing. ⚠ Putting it on the route type is a schema change across every fixture,
every rendering and every validator, and it is worth doing when a record exists
to migrate rather than while every record in the tree is synthetic.

### ⛔ The rule-12 install scan read more than the install ships. 2026-09-15

`install-client` scanned `find "$WORKDIR" -type f`, every file at every depth.
`capture-client.yml` uploads `install-<route>/*.log` and `*.err` - globs with no
`**`, which do not descend - plus the install record and the resolution. ⚠ The
scan was written against an older upload that took the whole directory; that step
narrowed after run 5 uploaded a build tree of 42.6 MB and 1867 entries, and the
guard did not follow.

⛔ **Run 15 refused its `source` lane over openssl's and nghttp2's own vendored
test certificates**, `src/third_party/openssl/apps/server.pem` and
`src/third_party/nghttp2/integration-tests/server.key` - public by construction
and uploaded by nothing. ⚠ A `release` route installs a binary and never clones,
so the shape was invisible until `E-ACQ-07`'s repair made the source lane real.

⭐ **Both halves of the rule are fixed**: the name scan and the value scan read
the workdir's top level. `grep -r` descends exactly as `find` did, so fixing one
would be the gate-on-one-path shape this guard was added to close.

⚠ **Amended 2026-10-14: the two rules are one file now.** They were copied into
`install-client.sh` from `capture-client.sh` and kept identical by a comment;
both call `scan-bundle.sh`, `bit-check scan-bundle` since 2026-10-15 (`CI-11`),
this one with `--top-level`, which is the scope this section settled stated as
one flag over both rules. `CI-09` carries why a third caller needed it.

⚠ **Narrowing is not weakening, and the direction says so.** Rule 12 is about
what REACHES a remote. What ships from that directory is `*.log` and `*.err` at
its top level; the scan still reads EVERY file at that level, which is wider than
those two globs.

Guard mutation, each plant verified to have changed the file:

| plant | verdict |
| --- | --- |
| a credential-named file in `src/third_party/.../server.pem` | accepted |
| the same name at the workdir's top level | refused |
| `-maxdepth 1` reverted, subtree case re-run | the case fails, so it can fire |
| `-maxdepth 1` reverted, top-level case re-run | still refused, so the branches are separate |

#### Residuals

- ⛔ **The dangerous direction is unguarded.** An upload path that gained a `**`
  under the install workdir would ship a subtree this scan no longer reads.
  Acceptance: a check that reads the *Upload the install logs* path block and
  refuses a pattern reaching below the top level, mutation-proved by planting
  `install-${{ matrix.route }}/**`.
- ⛔ **`$RUNNER_TEMP/release/` and `$RUNNER_TEMP/source/` ship and nothing scans
  them.** The resolver writes them and `install-client` is handed `--workdir` and
  nothing else, so a resolution document carrying a credential in a URL would
  travel. ⚠ No such route exists today, which is the condition
  `docs/methodology/reviews.md` names as the easiest place to get a scope wrong.

## ACQ-04: Disposable-host execution boundary

Source: proprietary installers and active network client execution
Priority: P0 | Effort: L | Status: DONE

Problem: Running untrusted clients on persistent CI hosts can leak state,
escape the lab, contaminate later samples, or violate package terms.

Premise: Fresh VM images, restricted egress, non-secret accounts, and verified
teardown can make captures reproducible and bounded.

Approach: Define Linux and Windows runner contracts, deny public BitTorrent
traffic, snapshot inputs, wipe the guest, and attest cleanup after every run.

Decision: the claim guard detects the failure of disposal rather than trusting
a declaration of it. A token the provisioner writes saying "this host is
disposable" was the rejected alternative and it is the shape that fails
silently: a runner misconfigured to persist its disk still carries the token and
still means it. Claiming by writing a marker makes a survived host produce
evidence of itself, and a misconfiguration cannot produce the marker's absence.

Decision: the egress guard reads `/proc/net/route` and never probes anything.
Reaching out from a machine the guard exists to establish is contained would be
the wrong order, and a connection attempt against a third party is not this
project's to make.

Decision: `check-runner` runs on every gate rather than only where captures
happen. It is hermetic and takes no network, and it is the check whose silence
would be worst.

Prove: the runner test detects forbidden egress and persistent state, then
shows a fresh host fingerprint for the next job.

Closure evidence: run on 2026-09-04.
[`../scripts/acquisition/check-runner.sh`](../scripts/acquisition/check-runner.sh)
is 7 guard cases, 7 passed, 0 failed, run twice with the exit code read unpiped
from its own process both times. It plants a default route that is up (refused),
one that is configured and down (allowed, so the flags are actually read), a
loopback-only table (allowed), an unreadable table (exit 2, because not knowing
is not a pass), a second claim on one host (refused), and a claim after teardown
(allowed). It prints the host fingerprint the next job compares.

The gate is 12 checks, 11 passed, 0 failed, 1 skipped, with `check-runner`
among them. The Windows gate reports it as a skip naming the reason rather than
omitting the row, because a lane that leaves a check out looks like one that
checked it.

⚠ **Two defects found by driving it, both fixed:**

1. **The egress guard could not run on this host.** It tested the route flags
   with `and(strtonum(...))`, which are gawk extensions; against a POSIX awk
   they are undefined functions and the guard reported "could not establish"
   over a machine with a plain default route. The direction of the failure was
   right and its reach was not;
   [`../docs/capture-host.md`](../docs/capture-host.md) says why that matters.
   The UP bit is the low bit of the flags word, so the last hex digit being odd
   is the same test with no extensions.
2. ⛔ **The runner test claimed the real machine.** One case ran `--claim`
   without the scratch state directory, so it wrote `/var/lib/bit-ids/host-claimed`
   on the session host: it passed once and failed every run after. On a
   persistent CI runner that is a check which goes green the day it is written
   and red forever, for a reason that looks like the guard being broken rather
   than the test being. Found by running it twice, which is the only thing that
   finds it. The marker it left was removed.

⚠ A third finding was in the measurement rather than the code: the exit code was
first read after a pipe into `tail`, so a script that exited 1 read as 0. That
is this repository's oldest stated rule, broken while checking a guard.

The manifest gained `isolation.claim`, carrying the fingerprint the guard read
and when it claimed the host. `E-MAN-33` refuses a claim stamped after the run
started, which is the distinction the whole entry rests on: a guard that ran
after the install is a report, not a boundary.

Residual, closed: the executable guards were Linux-only, because they read
`/proc/net/route` and `/etc/machine-id`. `assert-disposable.ps1` is the Windows
pair, reading `Get-NetRoute` and the machine GUID, and
[`../docs/capture-host.md`](../docs/capture-host.md) carries both contracts.
⚠ The workflow that runs them on a Windows host is `CI-03`'s:
[`../.github/workflows/capture.yml`](../.github/workflows/capture.yml). ⭐ **It
has been dispatched, and the guard read the live cmdlet**: with no `-RouteTable`
on a hosted `windows-2025` runner it answered `no route off this host (read
Get-NetRoute)`, so the fixtures `check-runner.ps1` proves the logic against are
the shapes the real output has. `CI-06` is that run.

⭐ **Both halves grew a `--marker` / `-Marker` mode while `CI-03` closed**, which
prints the claim marker's path and is the only derivation of it. `capture-run`
re-reads the claim at the moment of capture rather than trusting that an earlier
workflow step ran, and a caller that composed `$STATE_DIR/host-claimed` itself
would be a second spelling that goes on reading the old place the day the state
directory moves. `check-runner` has a case for it in each half.

Residual: the guards prove they fire, not that they are sufficient. A host can
be non-disposable in ways neither models, and
[`../docs/capture-host.md`](../docs/capture-host.md) lists the ones considered
rather than leaving them implied.

Residual: the fingerprint's discriminating power is argued from its inputs
rather than measured across two machines, because this session had one host. It
is a digest of the machine id, the boot id, the hostname, the kernel string and
the root device number; any single one of those can collide and the combination
is what is relied on.

## ACQ-05: Artifact cache and authenticity evidence

Source: repeatability and upstream availability risk
Priority: P1 | Effort: M | Status: DONE

Problem: Upstream URLs and package indexes change, but storing installers may
be disallowed.

Premise: Hashes, signatures, certificate metadata, package receipts, and
permitted artifacts can preserve verification without illegal redistribution.

Approach: Implement a policy-aware cache that stores bytes only when allowed
and otherwise stores sufficient authenticity and retrieval evidence.

Prove: cache tests enforce the licence register and reproduce artifact
identity after a simulated source URL change.

### ⭐ The second half of the Prove asks for less than it sounds like

"Reproduce artifact identity after a source URL change" needs no reproduction at
all once the identity is the digest. A URL is where bytes were found, which is a
fact about a moment; the digest is what they were. So a retrieval from a new
location is recorded against the artifact the digest already names, and the
lookup that finds it never mentions a URL. ⛔ **A cache keyed on the location
would stop answering the moment a vendor reorganised a download page**, which is
exactly the failure this entry exists to survive.

⚠ Repeating a retrieval is not a second retrieval either. A cache that grew a
row every time somebody re-ran an acquisition would report a popularity contest
rather than a provenance, so `observe` reports `Unchanged` and adds nothing.

### Decision: the register is asked, never re-read

⛔ `catalogue/licences.toml` has one parser per twin and it is `check-licences`.
The Rust cache takes a disposition map, the way the store rules take a tree a
caller has already read; the driving example takes `--permitted TARGET` flags;
and `check-cache.sh` fills them from `check-licences --permitted`. A second
reader of that file would be a second answer to what it permits.
⚠ **Amended 2026-09-29:** both twins of each are gone, and the rule holds more
tightly than it was written - `bit-check check-cache` calls the one reader
`check-licences` itself calls, in the same binary, rather than building it and
asking a mode. `CI-10` carries the port and its comparison.

⭐ `--permitted` was added to **both** halves and both were run against a
register with a planted `permitted` row: both printed `aria2` and both still
agreed on `--json`. ⚠ That matters because the flag's normal answer is empty,
which is also what a broken parser prints.

### ⛔ The refusal case needed a control, and the harness carries it

`FOUND-04`'s register refuses every target, so the scenario's storage attempt is
refused. On its own that case passes equally over a cache that can never store
anything, which is a different program. The harness runs the same scenario with
a target explicitly permitted, watches the same cache accept it, and asserts the
two runs differ on exactly the one line. ⚠ Without that, the whole policy half
would be theatre.

### Acceptance, all run on 2026-09-06

- `cargo test -p bit-ids --locked --all-targets`
- `sh scripts/acquisition/check-cache.sh`
- `sh scripts/common/check-gate.sh`

⚠ **Amended 2026-09-29**: the second line's script is deleted, so the runnable
form of that acceptance is `go run . check-cache` from `tools/check`, whose
directory holds the module, and `sh scripts/common/check-bitcheck.sh --harnesses`
plants against it. The list above is kept as what was actually run.

### Closure evidence, 2026-09-06

| what | measured |
| --- | --- |
| `sh scripts/acquisition/check-cache.sh` | 11 cases, 11 passed, 0 failed |
| `cargo test --workspace --locked --all-targets` | 38 binaries, 356 passed, 0 failed, six of them new here |
| guard mutation | 9 plants over `cache.rs`, 9 refused, each verified to compile first |
| driven pass | one artifact observed from two locations: one artifact, two retrievals, the digest still resolving, keeping nothing accepted and keeping the bytes refused as `E-CAC-01` |
| register tie | the permitted set came from `check-licences --permitted`, which answers nothing today, and both halves were separately shown to report a planted `permitted` row |

⚠ **Nothing here fetches anything.** The scenario is about identity and
permission, and a real download would make the run depend on a vendor's uptime
to answer a question about this project's own rules. The bytes are generated in
the example and are nobody's installer.

### Residuals

- ⚠ The cache has no on-disk document yet. `CACHE_SCHEMA` is declared and
  nothing writes one, because the first thing worth writing is a real
  acquisition's cache and no acquisition has run. The rules are pure over a
  cache a caller holds, which is what let this close without one.
- ⚠ `check-cache.sh` does not plant in the register, because that file is
  tracked and a gate check that edits one leaves a dirty tree when it is
  interrupted. The planted-register measurement is in the table above instead.
  ⭐ **Closed differently, 2026-09-29**: `check-bitcheck.sh --harnesses` plants a
  register permitting the scenario's own target, and one permitting another, into
  a SCRATCH copy of the tree, which edits nothing tracked. `CI-10` carries the
  verdicts.
- ⚠ Certificate metadata and package receipts from the Premise are not modelled.
  `SignatureStatus` records what was done about a signature and the manifest
  carries the rest; a receipt has no producer until a real acquisition runs.

## ACQ-06: Rootless, portable client installation

Source: operator direction on 2026-09-16, after thirteen dispatches hung
Priority: P0 | Effort: L | Status: DONE

Problem: Every install this project performs needs `sudo` and writes to
`/usr/local/bin`, so what an install does depends on a host this project does
not control. `capture-client` runs 21 to 31 hung inside *Install the client* on
byte-identical code, across two targets and both routes, and no bound - including
GitHub's own step supervisor - could end the step.

Premise: A client that is fetched, verified against a published digest and
unpacked into a prefix the current user already owns needs no privilege, no
package index and no host state. That install is the same on a runner, a session
host and a contributor's laptop, which makes it deterministic and ours.

Approach: One rootless installer that every adapter's `release` and `source`
route goes through. It resolves a prefix under the caller's own workdir, fetches
with a bound, verifies the digest before anything is executed, unpacks or copies
into `$PREFIX/bin`, and answers with the absolute path of the executable it
placed. No `sudo`, no `/usr/local`, no `apt`, no PATH mutation - callers take the
path the installer returns. The disposable-host state directory moves under the
same prefix so the claim guard needs no privilege either.

Prove: `sh scripts/acquisition/check-rootless.sh` installs every adapter's
release route as an unprivileged user into a scratch prefix and asks each
installed build its version; the run is repeated with `sudo` absent from `PATH`
entirely and must be byte-identical.

### ⛔ What this replaces, and why the `package` route cannot come with it

⛔ **A package route is privileged by construction** - it writes to directories
the distribution owns - so it is not portable and cannot be made rootless. ⚠ That
is a reason to stop treating it as one of the two routes a record needs, not a
reason to keep `sudo`: `E-ACQ-01` wants two INDEPENDENT routes, and `release`
plus `source` already are. The package route stays for hosts where it is wanted
and stops being on the path a capture depends on.

### ⛔ The measurement this entry exists to make impossible

⚠ **Thirteen bounds were spent on a step nobody could end**, and the install
itself was never the slow part: every operation it performs was measured on a
real runner at **two seconds or less**, and the whole sequence driven on a
session host exits 0 in **one second**. ⛔ The difference between those and the
hung step is `sudo` and the host it reaches into. Removing the privilege removes
the variable rather than instrumenting it.

### ⭐ What was built, 2026-09-17

`scripts/acquisition/install-rootless.sh`, `bit-check install-rootless` since
2026-10-15 (`CI-11`), is the one installer, and every release route goes through it. It fetches with a
bound, settles the digest before anything is made executable, writes into a
prefix the current user already owns, and prints the path it placed.

⛔ **THE PREFIX HAS ONE DERIVATION AND `--prefix` IS IT.** Every adapter asks
this file rather than composing a path, for the reason `assert-disposable
--marker` already exists: a second spelling goes on reading the old place the day
the first one moves, and an adapter's `binary()` and its `install` route are two
such spellings inside one file. ⚠ It is
`$HOME/.local/share/bit-ids` rather than `$HOME/.local`, so a build cannot
shadow the user's own tools on PATH and one directory holds everything this
project installed.

⛔ **AND IT NEVER ESCALATES.** A prefix the current user cannot write is a
refusal naming the prefix. An installer that reaches for privilege when it runs
out of permission is the thing this entry removes.

#### ⭐ The digest disposition, which is declared rather than defaulted

⛔ **Exactly one of three, and a caller that names none is refused.**
`--sha256 <hex>`, `--sums <file> --sums-name <name>`, or
`--digest-unpublished <measured reason>`. ⚠ The third is not a way round the
first two: it records that the bytes were IDENTIFIED and not VERIFIED, which is
a different sentence, and `digest_source` in the installer's report is where the
difference is made rather than left to a reader.

⭐ **The verifier is `sha256sum -c` over the vendor's own document**, so neither
the digest nor the comparison is this project's code. ⛔ **And it does not pass
over nothing**: measured 2026-09-17, `--ignore-missing` over a document naming
only absent files exits **1** with `no file was verified`, which is the refusal a
hand-written reader would have had to remember to add.

⚠ **Of the four targets, one vendor publishes a digest and three do not**, read
out of the listings on 2026-09-17: `aria2-next` ships
`aria2-next-{version}-checksums.sha256`; qBittorrent signs each asset with a
detached `.asc`; aria2 and Transmission publish neither. Each adapter declares
which, once, and its route reads its own declaration.

⛔ **`resolve-release.sh` resolves the document out of the same listing** that
chose the version and the artifact, so one retrieval answers three questions and
the recorded `listing_sha256` covers all three decisions. ⚠ `--checksums` is
`--listing`'s pair and the two are refused apart: a recorded listing names real
vendor URLs, so a digest resolved from one would reach the network out of a
harness written not to.

#### ⛔ AND THE CLAIM AUDIT FOUND THE FIRST VERSION SAYING LESS THAN IT COULD

⚠ **Three of the four adapters declared `release_digest_unpublished` and the
source was publishing a digest the whole time.** The declarations were literally
true - those vendors upload no checksums FILE - and the sentence they produced,
*this artifact could not be verified*, was not: a release listing carries a
per-asset `digest`, and it was sitting in the very response the resolver already
holds. ⛔ Found by re-reading the adapters' own comments against the LIVE
listings rather than against the fixtures they were written from, which is what
lens 3 is for.

⭐ **Measured through `AGENTS.md` rule 8's route on 2026-09-17**, the four
targets:

| target | vendor checksums document | listing `digest` | reaches |
| --- | --- | --- | --- |
| `aria2-next` | ⭐ yes | ⭐ on all 8 assets | `both` |
| `qbittorrent` | ⛔ no, 7 detached `.asc` | ⭐ on all 14 | `source-listing` |
| `transmission` | ⛔ no | ⭐ on all 11 | `source-listing` |
| `aria2` | ⛔ no | ⛔ `null` on all 6 | `unpublished` |

⛔ **A HOST IS NOT A VENDOR, AND THE RECORD SAYS WHICH ANSWERED.** The party
publishing a listing digest is the party serving the bytes, so it establishes
that nothing changed between reading the index and fetching the artifact and
says nothing that would survive that index being wrong. A vendor's uploaded
document is a different claim. ⭐ `both` is its own value because two parties
describing one artifact is a stronger sentence than either, and a record naming
only the stronger would tell a reader less than happened.

⭐ **`--from-resolution` is what keeps four adapters out of the decision.** The
installer reads the resolution record itself; four adapters choosing between
four dispositions would have been four copies of one decision, which is the
one-gated-door shape this project records most often. ⚠ It also deleted the two
environment variables the first version passed.

⚠ **The fixture contract paid for itself here.** `listings/README.md` says a
projection is by construction everything the readers read, and it was: teaching
the reader the field and re-recording at the SAME tags was the whole repair, and
not one asset name under test moved.

#### ⭐ The claim guard moved with it

`assert-disposable.sh`'s state directory is `$HOME/.local/state/bit-ids`.
⛔ `/var/lib/bit-ids` needs root to create, so `install-client`'s marker check -
which runs before any route - was itself a reason the install needed privilege.
⚠ [`../docs/capture-host.md`](../docs/capture-host.md) owns why a home directory
keeps the two properties the marker rests on.

⛔ **It is per-user, which is a NARROWER claim than `/var/lib` made**, and that
is recorded rather than hidden: a host where two users each ran a capture carries
two markers and neither refuses the other. The runner contract is one capture per
host and `docs/capture-host.md` carries it.

### Closure evidence, 2026-09-17

| what | measured |
| --- | --- |
| `BIT_IDS_ROOTLESS_USER=runnerlike sh scripts/acquisition/check-rootless.sh` | **26 cases, 26 passed, 0 failed**, driven as uid **1001**, all four dispositions reached through the real path |
| the same harness as `root` | green - and its own last row says that run establishes nothing about an unprivileged user |
| `sh scripts/acquisition/check-release-route.sh` | 35 cases, 35 passed, 0 failed, with eleven digest cases |
| `sh scripts/common/check-bitcheck.sh` | 94 cases, 94 passed, 0 failed |
| driven pass, the real step body | `install-step.sh` run end to end as uid 1001 against a `file://` vendor: **exit 0 in 5 seconds**, a complete install record, `acquired=yes`, `digest_source=both`, and the build in a prefix that user owns |
| guard mutation, the installer | 10 plants, 10 on the intended verdict, plus an acceptance control that still installs |
| guard mutation, the resolution reader | 7 plants, 7 on the intended verdict, plus two acceptance controls including a digest and a document together |

⭐ **THE PROVE'S SECOND HALF IS THE ONE THAT NEEDED AN INSTRUMENT.** "With `sudo`
absent from `PATH` entirely" cannot be done by removing PATH entries: `sudo` is
`/usr/bin/sudo` on this image, so dropping its directory drops `sh`, `sed`,
`mktemp` and `curl` with it. The first version did exactly that, all four routes
produced no output at all, and the two runs duly "differed" - which reads exactly
like the finding the comparison exists to make. ⭐ The harness builds a shadow
directory holding a link to every executable on PATH except `sudo`, and two
control rows assert both halves: that `sudo` was really there to remove, and that
the shadow still resolves every tool the routes need.

#### ⛔ What driving it as uid 1001 found that the root run could not

⚠ **The root run was green on nineteen cases and the uid-1001 run was red on
eight**, minutes apart, on the same tree. The harness's own workdir was
root-owned and merely readable, so the drive user could not create the state
directory, every route refused with *the host was never claimed*, and nothing in
the root run could have said so. ⭐ The scratch is `chown`ed to the drive user
now - not made world-writable, which would be a different problem invented to
solve this one.

⛔ **AND ONE CASE PASSED FOR THE WRONG REASON IN THAT SAME RUN.** *A second claim
is still refused* asserted an empty stdout, and the FIRST claim had failed too -
so it went green beside the red row that caused it. It reads the guard's own
refusal text now. That is a check satisfied by a different code path, which
[`../docs/methodology/reviews.md`](../docs/methodology/reviews.md) names as one
of the two shapes to test for.

#### ⛔ And what the gate found that the harness did not

⭐ **`check-adapters` answered 2 and exited `could not run`**, because this
change moved every release fetch out of the four adapters and into one installer
the rule did not read. ⛔ **Its floor is what caught it** - *a rule that found
nothing to ask about has asked nothing* - and without that floor it would have
reported a clean tree over a path whose every fetch it had stopped reading. The
scope follows the fetches now, and `check-bitcheck`'s `--max-time` plant is
re-aimed at the file that holds them.

⚠ **Three more, each mechanical and each found by the gate rather than by
reading**: an ellipsis outside this project's five markers, a `A && B || C` that
`shellcheck` refuses, and a plant whose `sed` carried a `$`.

### Residuals

- ⛔ **The `package` route is untouched and still privileged**, deliberately. It
  writes to directories the distribution owns, so it cannot be made rootless;
  what changed is that it is no longer on the path a capture depends on.
  `E-ACQ-01` wants two INDEPENDENT routes and `release` plus `source` already
  are.
- ⚠ **`report-holders.sh` now runs unprivileged and its claim is narrower.** It
  was under `sudo` because the install was, and an unprivileged reader of
  `/proc` reports nobody holding a file that root processes hold. Every process
  the step starts is now this uid, so it sees all of them - and a holder
  belonging to some OTHER user is outside what it can see. Measured on the driven
  pass: it reported `0 holder(s)` and said in words that one descriptor did not
  resolve to a path, rather than answering confidently about a pipe.
- ⭐ **`transmission`'s lookups have a prefix branch now**, since 2026-10-14.
  Its release and source routes compile and install into the prefix, so the
  branch landed with the first route that installs one, as this residual said.
  Both routes fetch libcurl's headers without privilege, through an `apt-get`
  whose index lives in the workdir. `CLIENT-06`.
- ⚠ **The Windows guard still writes under `ProgramData`.** `capture.yml` is a
  FIXTURE capture that installs nothing, so the privilege question this entry is
  about does not arise there. It closes when a Windows capture installs a client,
  which is `CLIENT-01`'s.
- ⛔ **IT HAS RUN ON A RUNNER NOW, AND THE ANSWER IS NO.** `capture-client` run
  32 on `a709e2d`, dispatched 2026-09-17: both lanes' *Install the client* never
  returned, both jobs were cancelled at **forty minutes** - 05:13:15 to 05:53:15
  - and the run uploaded **zero artifacts**. ⚠ This residual asked whether the
  step thirteen bounds could not end ends once the privilege is gone. It does
  not. **Removing `sudo` and `/usr/local` did not remove the wedge**, which
  refutes the reading that privilege was the variable.

  ⭐ **What the same run DOES establish, on a real runner, is the rest of this
  entry:**

  | step | run 32 |
  | --- | --- |
  | *Claim the host*, now **unprivileged** | ⭐ success, **0s**, both lanes |
  | *Resolve the release artifact*, which now also selects and fetches the vendor's checksums document | ⭐ success, **1s** |
  | *Resolve the source tag* | ⭐ success, 1s |
  | *Install the client* | ⛔ never returned, either lane |

  ⭐ **So `install-rootless.sh`'s bounded fetch demonstrably works on that host**:
  the resolve step used it to retrieve a document over the network in one second,
  on the same machine, minutes before the install wedged. The rootless claim
  guard and the digest resolution are proved on a runner; the install is not.

  ⛔ **AND THIS IS WHERE IT STOPPED, under a rule rewritten on 2026-10-11**:
  a hang is a dead end, not a subject. Two bounds is the limit and this entry's
  tally is far past it. Do not diagnose run 32, do not instrument it, and do not
  add another bound. What is recorded is the measurement and nothing inferred
  from it.
  ⛔ **RUN 33 REPEATED IT ON 2026-10-11**, on `8d6ed5e`, twenty-five days and
  several runner-image updates later, with the workflow's default inputs: both
  `aria2-next` lanes entered *Install the client* at 02:17:58 and were still in
  it until GitHub cancelled the run at 02:57:25 with **zero artifacts**, where a
  release lane takes seconds.
  ⭐ **AND THE CAUSE WAS FOUND THE SAME DAY, AND IT WAS NOT THE INSTALL.** A
  flight recorder posting off the host showed the install finishing in seconds
  and the host losing contact when the next step cut its route. `TODO/ci.md`
  under `CI-08` carries the runs; the capture now runs in a network namespace,
  and the rule that forbade looking is rewritten.
- ⚠ **`check-rootless` is a declared `n/a` on the Windows lane**, and its reason
  is a platform fact rather than a missing twin: Windows has no `sudo`, no uid to
  drop to and no `su`, so the comparison the harness rests on has nothing to
  remove.
