# Corpus entries

## CORPUS-01: Append-only canonical store

Source: b-ids data branch architecture and operator publication requirement
Priority: P0 | Effort: L | Status: DONE

Problem: Regenerating a latest-only dataset would erase older stable-release
records and destroy auditability.

Approach: Store run manifests and profiles under immutable product, version,
platform, route-set, and capture identifiers. New releases append; corrections
add records rather than rewriting evidence.

Prove: the validator rejects deletion or byte changes against the prior data
branch and accepts a new version directory.

Acceptance, both run on 2026-09-05:

- `cargo test -p bit-ids --locked --all-targets`
- `sh scripts/corpus/check-store.sh`

### Decision: the path is the identity tuple, and the route set is not in it

The Approach names a route-set component and it is not in the path. A store path
is derived from the tuple [`RecordId`](../crates/bit-ids/src/identity.rs)
digests, in full and nothing else, because that is the only choice under which
the path and the identifier cannot disagree. A component the identifier does not
carry files one record under two names; a component the identifier carries and
the path drops files two records at one name. The route set is recorded where it
is measured, in the record's own `acquisition`, and `E-ACQ-07` and `E-ACQ-08`
already refuse a record whose routes are not independent.

⛔ **The rejected alternative was the published layout itself, and it was wrong
in exactly that second way.** `docs/publishing.md` filed a profile at
`profiles/v1/<target>/<version>/<platform>/<arch>/<capture-id>.json`, with no
`package` segment, while the identity tuple carries one. A `deb` and an
`AppImage` of one version on one platform are two records and were one file.
The layout is amended and `two_packages_of_one_build_are_two_paths` pins it.

### Decision: a version that cannot be a path segment blocks publication

⛔ **`Version` accepts `../../etc`, measured rather than assumed**, because a
version string is what the installed build printed and imposing a grammar on it
would refuse builds that number themselves some other way. That value reaches a
path, so the store refuses it as `E-STO-01` rather than escaping it.

Percent-encoding lost: it needs `%`, which `RelPath` refuses, and widening that
alphabet weakens the rule every evidence path already depends on. A lossy escape
lost harder, because it maps two versions onto one directory, which is the
collision the append-only store cannot survive. `version_is_not_a_path_segment`
carries the measurement.

### Closure evidence, 2026-09-05

| what | measured |
| --- | --- |
| `sh scripts/corpus/check-store.sh` | 14 cases, 14 passed, 0 failed |
| `cargo test -p bit-ids --locked --all-targets` | 13 store unit tests, 6 store integration tests, 0 failed |
| `cargo test --workspace --locked --all-targets` | 32 binaries, 308 passed, 0 failed |
| `cargo test --workspace --locked --doc` | 2 passed, 0 failed |
| `sh scripts/common/check-gate.sh` | 13 checks, 12 passed, 0 failed, 1 skipped |
| `pwsh -File scripts/common/check-gate.ps1` | 13 checks, 10 passed, 0 failed, 3 skipped |
| `cargo fmt`, `cargo clippy --workspace --locked --all-targets -- -D warnings`, `shellcheck`, `shfmt -d -i 2 -ci` | exit 0 |
| guard mutation over `store.rs` | 10 plants, 10 refused |
| guard mutation against a real filesystem | 9 defects planted, 9 refused, plus 4 harness self-guards exercised |

⭐ **Nine of the ten source plants were refused by the filesystem harness and
the suite between them; the tenth needed a unique literal before it could be
planted at all.** `E-STO-12`, `E-STO-22`, `E-STO-01` and `E-STO-04` cannot be
reached through a filesystem, so they are planted in the unit tests instead:
⭐ **that a tar carries `a` and `a/b` together while `mkdir` refuses the pair
over a file was measured here, not assumed.**

⭐ **The driven pass found a defect the suite could not.** `check-store`
selected a record to place by its *name* and opened it, so the named-pipe plant
blocked the process forever while `validate_tree` already carried `E-STO-15` for
it. One action, two doors, and only one had a gate. The reader now takes the
entry kind from the walk.

⛔ **The door sweep found a second spelling of the layout.** The example decided
whether a path was a record, and therefore whether the placement check ran at
all, from its own copy of the prefixes. A layout change would have left it
recognising nothing, every record skipping its placement check, and the suite
green: a gate that stops applying rather than one that fails.
`is_profile_path` and `is_manifest_path` now live beside the composer and
`every_derived_path_is_recognised` closes the loop.

⛔ **The review's own harness miscounted three plants and said so.**
`grep -F` splits a pattern containing a newline into separate alternatives, so a
unique multi-line literal counts as the sum of its lines and reads as ambiguous.
It failed safe, refusing to plant rather than planting wrongly, and three guards
went briefly unproven until the counter moved off `grep`. `replace_once` in the
committed harness now refuses a multi-line literal outright, and that refusal is
one of its four self-guards.

### Residuals

- ⚠ `check-store.sh` has no PowerShell half, so the Windows lane reports it as a
  named skip. The rules are not platform-specific and the Rust suite exercises
  every one of them on both lanes; what the Windows lane does not do is plant a
  symbolic link and a named pipe against a real filesystem, neither of which an
  unprivileged Windows session can create. `CI-03` owns the Windows runner.
- ⚠ Nothing yet assembles a store to check. `PUB-01` builds the tree and
  `PUB-02` runs this comparison over the fetched `data` branch; until then the
  driving surface is pointed at two directories by hand.

## CORPUS-02: Semantic corpus validator

Source: operator accuracy requirement
Priority: P0 | Effort: L | Status: DONE

Problem: Schema validity alone cannot prove route count, connector independence,
field provenance, agreement, stable status, or evidence reachability.

Approach: Implement all publication invariants in Rust with stable diagnostic
codes and adversarial fixtures.

Prove: `cargo test -p bit-ids --locked --all-targets` rejects one fixture for
each invariant and validates the complete golden corpus.

⚠ **The `Prove` was rewritten on 2026-09-05 and the original is recorded here
rather than silently replaced.** It read
`cargo test --workspace --locked --test corpus_validator`, which is the second
half of the class `CI-05` closed: `--test <target>` runs one integration binary
and skips the library's own tests, so a rule proved in a `#[cfg(test)]` module
would not have been run by the entry's own acceptance. `CORPUS-01` left the same
note against this entry when it reordered the work.

### What was actually missing, which is narrower than the Problem reads

Route count, connector independence, field provenance, agreement and stable
status were already enforced, per record, by `SCHEMA-01`, `SCHEMA-03` and
`ACQ-01` through `ACQ-03`: `E-ACQ-01` counts routes, `E-ACQ-07` and `E-ACQ-08`
refuse routes that share a resolver or a delivery, `publishable` refuses a
disagreement and an uncorroborated measurement, and the channel is type-enforced
to one variant.

⛔ **Evidence reachability was the one nothing could answer, and the reason is
structural.** `bind` compares the profile against the manifest, so two documents
that agree about an artifact nobody wrote satisfy it completely. Only the store
turns a citation into bytes, so only a store-level pass can refuse one that does
not. That is `E-CRP-03` through `E-CRP-05`, and it is why this entry needed
`CORPUS-01` in front of it.

### Decision: a golden corpus is generated, not committed

The `Prove` names a golden corpus and the schema fixtures are not one. They
declare digests for artifacts that were never written, which is precisely the
defect `E-CRP-03` refuses, so committing them as a corpus would have made the
acceptance assert its own failure mode as the passing case.

`examples/build-store.rs` writes one instead: it puts the artifacts on the disk
first, then rewrites each document to describe the bytes it actually wrote, and
emits both through `to_json`, which validates. ⚠ Committing generated artifact
bytes was the alternative and it lost for the reason `OBS-08` already settled for
the torrent: a committed file can only be trusted, while a generated one is a
function of its inputs and can be rebuilt and compared.

### Decision: valid and publishable stay separate at store level too

`validate_corpus` refuses only what must hold of any store.
`publishable_view` separately reports which records may enter a published view.
A provisional record belongs in the store, because the disagreement it carries is
the evidence of that disagreement, and a validator that refused it would delete
exactly what the record model went to trouble to keep. `CORPUS-03` builds the
views on top of the report.

### Closure evidence, 2026-09-05

| what | measured |
| --- | --- |
| `sh scripts/corpus/check-corpus.sh` | 14 cases, 14 passed, 0 failed |
| `cargo test -p bit-ids --locked --all-targets` | 9 corpus unit tests, 0 failed |
| `cargo test --workspace --locked --all-targets` | 34 binaries, 317 passed, 0 failed |
| `cargo test --workspace --locked --doc` | 2 passed, 0 failed |
| guard mutation over `corpus.rs` | 8 plants, 8 refused |
| driven pass | `build-store` writes 11 objects; `validate-corpus` reads them back and reports 1 record, 1 run, 1 publishable |

⛔ **One plant reported a refusal that was not one, and the harness is why it was
caught.** The `E-CRP-03` plant did not compile, so `check-corpus` exited 2, which
is *could not run* rather than *refused*. The review harness counted any non-zero
status as a refusal on the filesystem path, having grown that guard only on the
Rust path. It now separates them, and the plant was rewritten to compile before
the guard could be called proved.

### Residuals

- ⚠ Placement is only asked of a path this build recognises as a record. A
  document filed outside `profiles/v1/` or `raw/v1/` is not read at all, so a
  record hidden under another root would not be placement-checked. Which roots
  a published tree may carry is `CORPUS-03`'s and `PUB-01`'s.
- ⚠ `bind` walks the profile's citations into the manifest and not the reverse,
  so a manifest artifact no profile cites is not compared. `E-CRP-06` catches the
  file-level version of that against the store; the document-level one is
  `SCHEMA-02`'s and is left where it is.
- ⭐ `check-corpus` runs on both lanes since 2026-10-14: it is `bit-check
  check-corpus`, ported from `check-corpus.sh` and compared case for case
  against it before that half was deleted. `CI-07`. ⚠ `check-store.sh` is
  still a named skip on Windows, for its symbolic-link and named-pipe plants.

## CORPUS-03: Deterministic indexes and latest views

Source: b-ids consumer-oriented indexes
Priority: P0 | Effort: L | Status: DONE

Problem: Consumers need convenient latest and lookup views without making
those derived files authoritative.

Approach: Generate sorted indexes by client, peer prefix, BEP 10 client value,
platform, version, and capture instant from canonical records only.

Prove: two clean builds have identical digests and every index row resolves to
one canonical profile.

Acceptance, both run on 2026-09-05:

- `cargo test -p bit-ids --locked --all-targets`
- `sh scripts/corpus/check-indexes.sh`

### Decision: one version ordering, exposed rather than re-implemented

A latest view has to order versions and `Version` is deliberately not orderable:
it is what the installed build printed. `ACQ-02` already solved this, so
`VersionScheme::components` became public and both callers use it. `resolve`
picks the newest release to acquire and this picks the newest record to point
at, which are different questions over one comparison, and a second
implementation would answer one of them differently on the day it drifted.

⛔ **The scheme is supplied, never defaulted.** `catalogue/clients.toml` does
not carry one today, and filling in a three-component shape for a target that
never declared it is the guess `ACQ-02` refuses. `build-indexes` takes
`--scheme TARGET:PREFIX:MIN:MAX` exactly as `resolve-stable` does, and a record
whose target has no scheme blocks the latest view under `E-VIW-01`. ⚠ Moving the
scheme into the catalogue is worth doing and is not this entry's: it changes a
document `check-project` validates, and `ACQ-02` owns the shape.

### Decision: the peer-prefix index is a measurement, not a decoder table

`docs/architecture.md` section 5 forbids a codec that maps a peer-ID prefix to a
client name. This index is the opposite of that rule rather than an exception to
it: the key is the fixed span of a peer ID **this project measured**, and the
row resolves to the record that measured it. A patterned value whose first span
varies has no prefix and produces no row, because a row keyed on an empty string
would answer every lookup.

### Closure evidence, 2026-09-05

| what | measured |
| --- | --- |
| `sh scripts/corpus/check-indexes.sh` | 10 cases, 10 passed, 0 failed |
| `cargo test -p bit-ids --locked --all-targets` | 10 index unit tests, 0 failed |
| `cargo test --workspace --locked --all-targets` | 35 binaries, 329 passed, 0 failed |
| `cargo test --workspace --locked --doc` | 2 passed, 0 failed |
| guard mutation over `index.rs` | 8 plants; 7 refused, 1 named below as unrefuted |
| driven pass | a three-version store indexes to 18 lookup rows and 1 latest row, byte-identical across two builds, latest `1.2.10` |

⛔ **The first mutation round refused two plants of eight, and six of the
survivors were not equivalent mutants.** Two were bad plants: one keyed the
ranking on the version's *length*, which happens to select `1.2.10` anyway, and
one fell back to a scheme map that was empty in the case it was aimed at. The
other four were real gaps, and each is now closed:

- rows were sorted and nothing could tell, because the corpus was read in one
  order both times. `the_document_does_not_depend_on_the_order_records_were_read_in`
  builds the same store from two insertion orders.
- nothing excluded a provisional record from a view. Removing the filter left
  every test green, because the only record in the fixtures is publishable.
- the peer-prefix rule for a varying first span was never reached, because the
  fixture's peer ID begins with a fixed run.
- `E-VIW-10` was asserted only where it could not fire.

⭐ **A bug the acceptance caught before the mutation pass did.** The ranking
carried a `Version` comparison beside the numeric one, and `Version` is `Ord` as
text, so `1.2.9 >= 1.2.10` held and the latest view answered `1.2.9`. One total
key replaced it.

### The guard that is not refuted

| guard | why nothing refutes it | what would have to be true |
| --- | --- | --- |
| `latest.sort()` | `best` is a `BTreeMap` keyed by the build line and `LatestRow` orders on those same four fields first, so removing it changes no output that can be produced today | `best` stops being an ordered map, or the row's ordering stops agreeing with the key's |

### Residuals

- ⚠ The version scheme belongs in `catalogue/clients.toml` and is passed on the
  command line instead. Named above with why it is not this entry's.
- ⚠ `captured_at` rows in a store written by `build-store` all carry one instant,
  because the fixture's capture instant does not move with `--version`. It makes
  that index's rows share a key, which is correct behaviour and a thin test.

## CORPUS-04: Supersession and correction records

Source: append-only publication constraint
Priority: P1 | Effort: M | Status: DONE

Problem: A proven bad record must stop appearing in current views without
deleting the historical evidence.

Approach: Define signed correction records naming the original digest, reason,
replacement, and review evidence; derive current views accordingly.

Prove: fixtures retain the original bytes, exclude a superseded record from
latest views, and expose the full correction chain.

⚠ **Half the Approach was already built and the entry does not re-do it.**
`SCHEMA-01` gave a record `supersedes`, `SCHEMA-03` gave it `adjudication` with
its reason and cited evidence, `E-ADJ-01` through `E-ADJ-05` hold the pair
together, and `E-CRP-07` refuses a correction naming a record the store does not
carry. What none of that did was change any answer: a corrected record was still
in every lookup and could still be the latest view's reply.

### The three things a correction changes, and the one it must not

⛔ **A superseded record leaves every view and stays in the store.** The
append-only rule keeps its bytes, its evidence and its path forever; what a
correction changes is which record answers a question asked now.

⛔ **The chain is published rather than left to be inferred.** A consumer
holding an identifier from last month has no other way to discover that it was
corrected, so `corrections` carries the superseded identifier, the record that
directly corrects it, and the record at the end of the chain. ⚠ `by` and
`current` differ exactly when a correction was itself corrected, which is the
case a single-step row answers wrongly while looking right.

⚠ **Only a publishable correction retracts anything.** A correction carrying its
own disagreement is provisional, and letting it drop the record it corrects
would leave the build line answering nothing at all: a measurement lost to a
record that is not fit to replace it.

### Decision: two refusals rather than a resolution

⛔ **A fork is refused, not resolved.** Two records correcting one measurement
has no answer to "what replaces this", and picking the lower identifier would
publish one adjudication and silently discard the other. `E-VIW-03`.

⛔ **A cycle is constructible, so the walk is bounded.** A record identifier
digests the identity tuple and not `supersedes`, so two records can each name
the other while neither supersedes itself, which is the only shape `validate`
refuses. An unbounded walk would hang rather than report. `E-VIW-04`.

### Acceptance, all run on 2026-09-06

- `cargo test --workspace --locked --all-targets`
- `sh scripts/corpus/check-indexes.sh`
- `sh scripts/common/check-gate.sh`

### Closure evidence, 2026-09-06

| what | measured |
| --- | --- |
| `cargo test --workspace --locked --all-targets` | 36 binaries, 343 passed, 0 failed, six of them new here |
| `sh scripts/corpus/check-indexes.sh` | 15 cases, 15 passed, 0 failed, five of them new here |
| `sh scripts/common/check-gate.sh` | 17 checks, 16 passed, 0 failed, 1 skipped, 0 unavailable |
| guard mutation | 6 plants over the new guards, 6 refused |
| driven pass | one store built with the correction and one without; the corrected record byte-identical between them, the latest view answering the correction's path, and the chain naming the identifier a reader may still hold |

⭐ **The retention half needed two stores rather than one.** Reading the
corrected store and finding the original still there proves it was written, not
that it was left alone. `build-store --correct V` writes the correction beside
the record it corrects, and `check-indexes` compares that record's bytes
between the two stores, which is the only comparison that can see a correction
that edited what it corrected.

### ⛔ What the claim audit corrected in its own first draft

This entry was about to record, as a residual, that a cycle is a coherent store
the corpus validator accepts. Asserting it instead of writing it down showed
`validate_corpus` refusing that store outright, on `E-CRP-01`, because the
fixture carries no run manifests. The residual is real and the reason was wrong:
what the validator does not report is `E-CRP-07`, whose question is whether the
record a correction names exists, and in a cycle both do.

### Residuals

- ⚠ Nothing at corpus level refuses a fork or a cycle. Both are caught when a
  view is derived, which is before anything can be published, and a caller who
  never derives one sees neither. `crates/bit-ids/src/index.rs` holds both
  codes; moving them beside `E-CRP-07` would put one rule behind two doors and
  is worth doing only with a check comparing the two.
- ⚠ `publishable_view` still reports a superseded record as publishable, and
  that is correct rather than an oversight: it is a valid measurement and it is
  no longer current, which are different questions. The driven run reports three
  of three publishable beside one superseded.
- ⚠ `bit-ids/index/1` gained two fields and a list, so every digest a consumer
  had recorded moves. Nothing has ever been published, so there is no consumer
  and no generation to bump; a later change to this document is not free.
