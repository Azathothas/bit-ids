# Foundation entries

## FOUND-01: Repository workspace and policy skeleton

Source: operator request and Azathothas/TEMPLATE bootstrap method
Priority: P0 | Effort: L | Status: DONE

Problem: An empty remote cannot support repeatable research or unattended CI.

Premise: A lean Rust workspace, explicit 0BSD licence, routed documentation,
target catalogue, work corpus, and local gate are the minimum coherent base.

Approach: Adopt the template's durable policies and checks, remove template
scaffolding, and specialize every front door for bit-ids.

Prove: `cargo test --workspace`, both fast gate runners, and the three review
passes named in `docs/history/BOOTSTRAP-REVIEW.md` pass before the bootstrap
commit is pushed.

Closure evidence: the bootstrap commit contains the complete skeleton and the
commands and review findings are recorded in
[`../docs/history/BOOTSTRAP-REVIEW.md`](../docs/history/BOOTSTRAP-REVIEW.md).

## FOUND-02: Reproducible Rust dependency and action pins

Source: b-ids workflow architecture and template public-CI policy
Priority: P0 | Effort: L | Status: DONE

Problem: Floating dependencies can change the observer or publisher without a
reviewed repository change.

Premise: Cargo.lock plus immutable action SHAs are necessary but do not by
themselves prove dependency provenance.

Approach: Commit the lockfile, deny unreviewed Git dependencies, inventory all
action SHAs, and add an update procedure with changelog review.

Decision: no separate register of pins. The lockfile is the crate inventory and
the workflow line is the action inventory, so a register would be a third copy
of commits that already live in two places with a check between them. The
inventory requirement is met instead by making both places checkable: a
lockfile entry must carry a crates.io source and a checksum, and an action pin
must carry the version comment that `check-remote-items` resolves against the
tag it names.

Prove: `cargo metadata --locked --format-version 1` succeeds and the project
checker rejects a workflow action referenced by a tag.

Closure evidence: `cargo metadata --locked --format-version 1` exits 0. The
pin rule was inverted from a denylist of known floating refs to an allowlist of
immutable forms, and seven defects were planted against the result on
2026-09-04, each refused by both halves with the same exit code: a tag, a
branch name, an abbreviated commit, a pin with no version comment, a lockfile
entry with a git source, a lockfile entry with no checksum, and a manifest git
dependency.

⭐ Three of those seven passed the old rule: a branch called anything other
than `main` or `master`, an abbreviated commit, and a bare commit with no
comment. That is measured, by running the old expression against each planted
case, and it is the argument for an allowlist over a denylist.

[`../docs/supply-chain.md`](../docs/supply-chain.md) carries the three layers
and the update procedure.

Capability note: `pwsh` was absent at session start, so the PowerShell halves
were unexercised. It was installed from the upstream release tarball, and both
halves were then compared on every planted case above rather than on a clean
tree alone. ⚠ `check-twins` compares answers on the tree it runs against, so a
rule that only differs on a defect is invisible to it; that gap is why the
comparison was run per mutation.

Door sweep findings, both fixed: the pin rule read `.github/workflows/` only,
so a composite action under `.github/actions/` would have carried unchecked
`uses:` lines with the same permissions. The scope now covers both and was
proven with a throwaway composite action carrying a floating pin, refused by
both halves. And the enumeration turned up a fourth layer the rule cannot
reach at all, `go install mvdan.cc/sh/v3/cmd/shfmt@v3.14.0` in a `run:` script;
it is safe for a reason particular to Go's checksum database rather than by
the form of the pin, so it is argued in `docs/supply-chain.md` rather than
waved through.

Residual: `check-remote-items` verifies the pins a pull request proposes, not
the pins already in the tree, so a tag moved or deleted upstream after a merge
is not detected. Named in `docs/supply-chain.md` and carried by `CI-04`. ⚠ It
needs `gh` and was not run in this session; the CI Linux lane runs it.

### Dependabot pull requests 3 and 4, taken by hand. 2026-10-21

Pull request 3 proposed `actions/download-artifact` 8.0.2 and
`actions/upload-artifact` 7.0.2; pull request 4 proposed `serde_json` 1.0.154
and `sha2` 0.11.1. Both are taken on `main` in one change rather than merged,
because work goes to `main` and pull request 4 was red only for want of its
`catalogue/licences.toml` rows (`FOUND-04`). Each crate is still
`MIT OR Apache-2.0` by its own manifest, so the rows move version and nothing
else. The action pins are commits with their version comments, the form this
item's rule requires, in all four workflows that use them. ⚠ Nothing here
closes either pull request; dependabot closes its own once `main` carries the
versions.

⭐ **It did, two minutes after the push.** `c26db1a` reached `main` at 10:09
on 2026-10-21. Dependabot closed pull request 3 at 10:11:19 and pull request 4
at 10:11:25, both unmerged, read back through rule 8's route. CI run 245 on the
same commit is green.

## FOUND-03: Deterministic protocol fixture suite

Source: bit-cli peer, tracker, and loopback tests; reference sweep
Priority: P0 | Effort: L | Status: DONE

Problem: Live captures alone cannot distinguish an observer regression from a
client behavior change.

Premise: Byte-exact fixtures for announces, handshakes, extended handshakes,
and message sequences can exercise parsers without external software.

Approach: Store small generated fixtures with documented provenance and assert
both parse output and deterministic re-encoding in Rust tests.

Decision: a second crate, `bit-ids-wire`, rather than a `wire` module inside
`bit-ids`. `bit-ids` is the published record contract that `LIB-01` ships to
catalogue consumers, and none of them needs a BitTorrent codec; putting the
codec there would make the consumer library the shared point between the probe
and the corpus tool, which is backwards. The rejected alternative is cheaper
today by one manifest and would have to be undone at `LIB-01`. The arrow points
one way: `bit-ids-wire` depends on `bit-ids` for the canonical value forms and
nothing depends on it yet. No new third-party crate; the lockfile diff is the
workspace member and nothing else.

Decision: the fixtures are hex text inside a JSON document, not `.bin` files
beside one. A literal control byte in a tracked file is skipped by `grep` and
rendered as "Binary files differ" by `git diff`, which
[`../tools/check/controlbytes.go`](../tools/check/controlbytes.go)
exists to refuse. Hex is lossless, so byte-exactness gives up nothing for it.

Decision: the codecs observe rather than impose. Unsorted bencode keys, `i-0e`,
a bare `\n` terminator, an unassigned message id and a non-standard handshake
protocol string are all recorded rather than refused, because each is a
difference between builds and refusing one turns an observation into a parse
failure. One deviation is refused instead: a byte-string length prefix with a
leading zero, which is an artefact of the encoder's integer formatter rather
than a value the build chose, and preserving it would put a second spelling on
every string in the tree.

Prove: `cargo test -p bit-ids-wire --locked --all-targets` passes twice with
identical fixture digests.

⚠ Corrected on 2026-09-05 by `CI-05`. The Prove was authored as
`cargo test --workspace` followed by the bare word `fixtures`, which selects by
test **name** and exits 0 over nothing when no name matches. It worked here only
because every function in that file happens to begin with `fixtures`, which is a
convention nothing held. The closure evidence below records the command that was
actually run at the time and is left as it was.

Closure evidence: run on 2026-09-04. `cargo test --workspace --locked fixtures`
passed twice, 17 tests each time, 0 failed. `cargo run -p bit-ids-wire --example
fixture-digests` produced identical output both times, corpus digest
`sha256:ed574f760001ee5d8c79ccd357456ef4f169aebfdbb4a11efafe656da44911db`. The
committed [`../crates/bit-ids-wire/tests/fixtures/index.json`](../crates/bit-ids-wire/tests/fixtures/index.json)
carries the same digest, so the second half of that acceptance is asserted by
the suite rather than compared by eye. The whole suite is 118 passed, 0 failed
at `--workspace --locked --all-targets`, and `sh scripts/common/check-gate.sh`
is 10 passed, 0 failed, 1 skipped.

Driven pass: the fixture corpus was replayed over real loopback TCP, one byte
per write so every frame boundary lands mid-message, and decoded incrementally
the way `OBS-04` will have to. 158 bytes went out and came back identical, the
handshake, the extension bits, the extended handshake and the message order all
read correctly off the reassembled stream, and a deliberately short slice
reported `truncated` rather than inventing a message. ⭐ The one-gated-door
check was part of it: the same announce datagram read with the wrong
`Direction` yields `Action::Other(287454020)`, which is the high half of the
connection id, and `as_announce_request` returns `None` rather than a plausible
wrong answer.

⭐ Guard mutation: nine lossy defects were planted in the codecs one at a time,
each restored before the next, and **every one was refused by the fixture corpus
alone**: repairing a bare `\n` to `\r\n`, percent-decoding on arrival, folding a
header name, sorting dictionary keys on decode, canonicalising integer text,
dropping an unassigned message id, dropping keep-alives, truncating the reserved
block, and dropping the BEP 41 options tail.

⚠ Two of those were not caught on the first attempt, and both are fixed here
rather than noted:

1. **The corpus did not exercise `bencode::encode` at all.** A message
   re-encodes from its payload bytes, held verbatim, so the bencode encoder is
   nowhere on the transcript round-trip path; the canonicalise-integers mutation
   was planted and the corpus passed. `E-FIX-10` now re-encodes every extension
   dictionary against its raw bytes, which is the only thing that reaches that
   encoder. This is the "grep an abstraction's callers before believing it is
   load-bearing" finding, and it was invisible until a mutation was planted.
2. **No fixture carried a bare `\n`**, so a terminator-repairing decoder was
   invisible to the corpus while the unit test caught it. That is the
   `check-twins` hazard in a different language: a rule that differs only on a
   defect the tree does not contain is not tested by the tree. The unusual-
   encoding fixture now mixes one `\n` line among `\r\n` lines.

Two further defects were found by the suite while writing it and fixed:
`d1:ae` reported "no bencode value starts with 'e'" instead of naming the key
that lost its value, and the synthetic-identity check scanned raw bytes, so it
passed six fixtures while missing the one that percent-encodes its peer ID,
which is the fixture that exists because clients do that.

⭐ Door sweep, three findings, all fixed here:

1. **`FixtureIndex` derived `Deserialize`**, so `serde_json::from_str` was a
   second and looser door that skipped the corpus-digest check entirely. An
   index nobody checked certifies whatever it happens to say. It now uses the
   private field mirror, the same construction `SCHEMA-01` used on `Profile`
   for the same reason, and a test drives both routes.
2. **`load_directory` filtered for `*.json` and listing is not recursive**, so a
   fixture added under `fixtures/peer/` would have been silently skipped: never
   loaded, never in the index, never failing, while the corpus went on claiming
   to cover its surface. Everything in the directory is now either loaded or
   refused with `E-FIX-11`, and a subdirectory and a stray file are both
   planted against.
3. **`HttpRequest::MAX_HEAD` was written and did not bind.** It only fired when
   there was no blank line at all, so a hundred megabytes of headers with a
   blank line at the end parsed in full. The cap is now on where the head ends.

Residual: `Surface::Mse` and `WebSeed` have no codec, and a fixture on either is
refused with `E-FIX-07` rather than silently passing. `OBS-11` owns both. ⚠ Three
moved on 2026-09-06: `Pex` rides inside a peer-wire transcript and `OBS-06` reads
it out of one, so a fixture for it is a `peer_wire` fixture and never its own
surface; `LocalDiscovery`, added to the vocabulary by the same entry, is read by
the HTTP codec that already existed and validates; and `OBS-11` gave `Dht` a
codec and two fixtures. ⛔ **The `E-FIX-07` control moved with it.** It named
`dht` as the surface with no codec, which stopped being true the moment that
codec landed, so a negative control has to name a surface that is still
uncovered: it names `mse` now. A control that quietly starts asserting the
opposite of what it was written for is the shape a guard-mutation pass exists to
find.

Residual: an incremental reader distinguishes "need more bytes" from "malformed"
only by `WireError::kind() == "truncated"` plus its own knowledge of whether the
peer may still send. That is inherent to a stream rather than a gap in the
codec, it is what the driven pass exercised, and `OBS-04` owns the read loop
that acts on it.

## FOUND-04: Third-party licence and redistribution register

Source: public-repository policy and proprietary client scope
Priority: P1 | Effort: M | Status: DONE

Problem: Free observation data does not grant permission to redistribute
installers, client assets, or third-party code.

Premise: The corpus can publish our measurements while retaining only hashes
and source locations for binaries that cannot be redistributed.

Approach: Record licence, artifact redistribution rule, source evidence, and
required notices for every dependency and acquisition route.

Prove: the licence checker reports every catalogue target and dependency with
a non-empty disposition and rejects bundled proprietary artifacts.

### ⛔ What the measurement found: most of them have no answer to give

Nine targets have a GitHub upstream, so their licence endpoint was asked. **Six
answered `NOASSERTION`**, which means a detector could not name a single
licence. Only `aria2`, `biglybt` and `anacrolix-torrent` came back with an SPDX
identifier. Writing one into the other six anyway would have invented exactly
the kind of fact this project refuses everywhere else, so the register records
`unverified` and says who was asked.

⚠ **An SPDX id from that endpoint is a detector's conclusion, not a reading of
the licence text**, and `licence_source` says so per row rather than letting all
the answers look equally strong. The three sources are the repository endpoint,
a crate's own `Cargo.toml`, and `clients.toml`'s existing `open_source` flag.

⭐ The twenty-two dependency licences are the strongest rows here, because each
was read out of the package's own manifest at the version the lockfile pins.
⚠ `libc` is not built for this host's target, so its manifest was fetched with
`cargo fetch --locked --target` before it was read rather than assumed from its
siblings.

### Decision: `unverified` is a disposition, and refusal is the default

⛔ **Every row says `refused`, and that is the policy rather than a consequence
of the licences.** This project publishes measurements and never artifacts, so a
licence that would permit redistribution does not make this repository
redistribute. `unverified` then costs nothing: it records that nobody has
established the licence and that nobody may treat the target as permissive on
this file's authority.

⚠ `permitted` exists in the vocabulary and nothing uses it. It is what the check
has to be able to refuse: a row claiming it over an `unverified` licence, or
with no notice, is the combination that would publish somebody's bytes on
nobody's authority.

### ⛔ What the twin comparison found, which a clean tree could not

The pair agreed on the clean tree and disagreed on an empty register: the `sh`
half reported two failures where the PowerShell half reported three. The cause
is a shell idiom rather than a rule: `grep -c .` prints `0` **and exits 1** on an
empty file, so the `|| printf 0` fallback fired and the variable became two
zeroes on two lines. The comparison then said `Illegal number` and the guard did
not run at all.

⭐ **The guard that failed is the one that refuses a register of nothing, and it
was disabled by exactly the input it exists to catch.** It was invisible on a
clean tree, which is what `check-twins` compares, and visible immediately when
the two halves were compared per planted mutation. `wc -l` replaces the idiom.

### Acceptance

⚠ **The commands changed when the rule was ported**, 2026-09-15. Both shell
halves are deleted, so the pair below is one binary that runs on either lane;
the closure evidence under it is a dated measurement of the tree at closure and
is left as it was.

- `bit-check check-licences`, built from [`../tools/check/`](../tools/check/)
- `bit-check check-ignores`
- `sh scripts/common/check-gate.sh`

### Closure evidence, 2026-09-06

| what | measured |
| --- | --- |
| `sh scripts/common/check-licences.sh` | 16 target rows and 22 dependency rows, every one with a disposition |
| `pwsh -File scripts/common/check-licences.ps1` | the same line, character for character |
| twin comparison | 12 cases planted one at a time, both halves run on every one and their exit codes and machine-readable output compared; 12 of 12 agreed after the defect above was fixed |
| dependency licences | 22 of 22 read from each package's own manifest at the locked version, none asserted |
| target licences | 3 SPDX identifiers, 6 `unverified` where the endpoint answered `NOASSERTION`, 5 `proprietary` from the catalogue's own flag, 2 `unverified` with no route asked |

⭐ The twelve cases are the shape space rather than one example: a missing row,
a stale row, a version that moved, an empty licence, a value outside the
vocabulary, `permitted` unearned, a closed-source target under an open licence,
a foreign schema, an empty register, and an installer-shaped file in the tree.
The clean register is compared first and again at the end, because a restore
that silently failed would leave every later case running against a defective
one.

### Residuals

- ⚠ `ktorrent` is on KDE's own forge rather than GitHub, so no licence route was
  asked and its row is `unverified`. That is a gap in coverage and not in the
  register: adding a second source is `ACQ-05`'s kind of work.
- ⚠ The tracked-artifact rule matches on file extension, so an installer
  committed without one passes it. A content check would need to read every
  tracked file on every gate run; the extension list is the cheap half and the
  register is the half that says what may be here at all.
- ⚠ Nothing re-asks the licence endpoints. A target that gains a detectable
  licence keeps its `unverified` row until somebody measures again, which is
  honest and stale in the safe direction.

### ⚠ The `rusqlite` 0.40.2 bump, offered and declined again. 2026-09-15

A dependabot pull request has been red since 2026-09-11, on every job. The cause
is this register: the bump leaves **18 locked packages with no row** and **5 rows
naming no locked package**, and `check-licences` compares the two directions
separately.

⭐ **That is the register working.** A dependency bump is a decision about
somebody else's code, and this red is what makes it one.

⛔ **The bump was taken and then reverted, because `PUB-05` had already measured
and rejected it.** [`publishing.md`](publishing.md) and
[`../docs/supply-chain.md`](../docs/supply-chain.md) record the reasoning:
0.40.2 resolves thirteen further packages - a WebAssembly stack this project
never builds for - for the same API and the same bundled library. ⚠ Re-measured
here with `cargo tree -e normal --target ...`, which agrees: on
`x86_64-unknown-linux-gnu` nothing new compiles, and the whole chain appears only
under `wasm32-unknown-unknown`. The recorded decision weighed that and declined
it; taking the bump to make a bot green would have overridden a measured
judgement with a notification.

⚠ **One fact the original decision did not have** is the vendored SQLite version,
which differs between the two. That was a reason to revisit and not a reason to
override, and it was revisited: ⭐ **the operator settled it on 2026-09-16 and the
answer is no.** [`PROGRESS.md`](PROGRESS.md) carries it under *Settled decisions*
with the numbers and where they were read from. ⛔ Pull request 1 is closed
unmerged and the pin is 0.37.0 in both the manifest and the lockfile.

⛔ **Correction, 2026-09-29: that closure was not the decision's, and a second
pull request stood open behind it.** Read back through `AGENTS.md` rule 8's route:
dependabot closed pull request 1 ITSELF at 03:26 on 2026-09-15, commenting *"no
longer updatable"*, because `76c32ea` had taken 0.40.2 at 03:23. `42789a4`
reverted that at 04:10, which made the bump available again, and dependabot
opened pull request 2 for the same bump on 2026-09-18 - red on all six jobs of
CI run 167, over the same 18 unregistered packages and 5 orphaned rows.

⭐ **The decision now lives where the bot reads it.** `.github/dependabot.yml`
ignores `rusqlite` under `version-update:semver-major` and `semver-minor`.
⚠ Read out of dependabot-core rather than assumed, through rule 8's route on
2026-09-29 - blobs `a56c4f35` (`config/ignore_condition.rb`), `13ad05ff` (the
common `version.rb`) and `104cccc4` (the cargo one) - because the first version
of this reasoning was wrong: its cargo `Version` applies Cargo's pre-1.0 rules, so
for 0.37.0 BOTH types resolve to `>= 0.38` - a change of `y` in `0.y` is major
there, not minor - while a patch inside 0.37 is still proposed. And
`IgnoreCondition#ignored_versions` returns only explicit `versions` for a
security update, so the rule cannot silence one. ⚠ Pull request 2 stayed the
operator's: `docs/supply-chain.md` says nothing here closes an item.

⭐ **Driven on the real system, 2026-09-29: the push was the whole experiment.**
Dependabot ran on the changed configuration within seconds of `2451816` landing -
its three update jobs at 01:25:38 and 01:25:39 - and closed pull request 2 ITSELF
at 01:25:41, commenting *"Looks like rusqlite is no longer being updated by
Dependabot, so this is no longer needed."* ⚠ That is a different sentence from
pull request 1's *"no longer updatable"*, which is what says the ignore rule was
read rather than the pin: one names the configuration and the other the tree.
CI run 168 on the same commit is green on all six jobs.

## FOUND-05: The session host, provisioned by something rather than by memory

Source: three tools installed by hand at the start of every session
Priority: P1 | Effort: L | Status: DONE

Problem: `pwsh`, `shellcheck` and `shfmt` are absent on a fresh container, and
without them this host runs a smaller gate than CI: every paired check loses its
PowerShell half, and shell syntax and style are not checked at all. The
commands live in `TODO/PROGRESS.md` as prose, which means every session either
runs them from a document or runs a smaller gate without noticing.

Approach: A provisioning script the doctor can call, pinned by version and
digest the way `FOUND-02` pins actions, that installs what the gate needs and
reports what it already has. ⚠ The `chmod +x` on the PowerShell tarball is not
optional and the failure reads as `Permission denied` rather than as a missing
file, which is exactly the kind of step a prose instruction loses.

Prove: on a container with none of the three, one command installs all three and
`sh scripts/common/check-gate.sh --strict` then differs from a run without them
only in `check-remote-items`; each download is verified against its pinned
digest before it is executed; and the doctor reports the same versions the
script pinned.

⚠ **This entry said `P1` and the index said `P2`, from the commit that created
both.** Found on 2026-09-08 by a sweep of every field the two places share, which
found exactly one disagreement in sixty-three entries. The entry's own
declaration is what the index now carries, on merit as well as on authority: a
local gate that is quietly smaller than CI is required coverage rather than a
refinement after coverage is operating, which is what `RULES.md` separates `P1`
and `P2` on, and `PROGRESS.md`'s work order already grouped this with `CI-07`
and `CI-08`, both `P1`.

⛔ **The rejected alternative was to make the entry say `P2`**, which would have
agreed with the index's priority table. That table is derived from the index's
own rows, so it corroborated nothing: it agreed with whichever half it was
computed from. `check-project` compares priority and effort between the two
places now, in both halves, and an effort-only plant fires it alone - which is
what shows it is not the priority table's check wearing a second name.

Closure evidence: run on 2026-09-08, on this session host, by taking all three
tools away and putting them back.

⭐ **One command, four seconds, from a host with none of them.**
`sh scripts/doctor/provision.sh` reported `installed pwsh 7.4.6`,
`installed shellcheck 0.10.0`, `installed shfmt 3.14.0`, exit 0, and a fresh
shell then got those three versions out of the executables themselves.

⭐ **The gate difference is exactly two rows, and one of them is a FAILURE
rather than a skip.** With the three taken away, `check-gate.sh --strict`
answered `24 passed, 1 failed, 2 skipped`: `FAIL check-capture (exit 1)` and
`SKIP check-twins (no pwsh or powershell on PATH)`. With them back it answered
`26 passed, 0 failed, 1 skipped`, the one skip being `check-remote-items`, which
is the documented gap no session host can close. ⚠ A host without `pwsh` does not
merely run a smaller gate; it runs a RED one, and a session could read that
`check-capture` failure as a defect in the tree.

⭐ **A wrong digest refuses and installs nothing.** With
`SHELLCHECK_PINNED_SHA256` planted to sixty-four zeroes and `shellcheck` removed,
the run printed `does not match its pinned digest; nothing was installed`, exited
1 naming that tool alone, still installed the other two, and `shellcheck` was
still absent afterwards.

⭐ **A tool present at the wrong version is a third answer.** A stub `shfmt`
earlier on `PATH` answering `v3.7.0` produced `differs shfmt 3.7.0 (pinned
3.14.0)` and exit 1, so absent, wrong and right are three states rather than two.

⭐ **The doctor agrees.** `sh scripts/doctor/doctor.sh` reports `shellcheck
0.10.0`, `shfmt 3.14.0` and `pwsh 7.4.6`, all resolved from `/usr/local/bin`.
⚠ `--fast` prints a `-` in the version column, so the version claim needs the
full pass; the fast one establishes presence and path only.

Driven pass, and what it found: ⛔ **the first version reported two tools wrong
that it had installed correctly.** `fetch_verified` and `provide` both used
`_want`, a POSIX shell function has no locals, and the caller therefore compared
a version against the digest the callee had left behind - printing `pwsh answered
[7.4.6] after installing 6f6015...` and exiting 1 over a host that was correct.
⚠ Invisible to `shellcheck` and to reading either function alone. ⭐ `shfmt` was
the row that reported correctly, because `go install` is the one route that never
calls the fetcher, so the two tools that downloaded were exactly the two that
lied. Every name in both functions is prefixed now. `PROGRESS.md` already carried
this defect class from a sourced library; this is the same collision inside one
file.

Decision: **the doctor does not call it**, though this entry asked for a script
"the doctor can call". A read-only pass that installs software the moment
somebody runs it is the surprise `scripts/doctor/README.md`'s first sentence
rules out. `--check` is the read-only half and installs nothing.

Decision: **`shfmt` comes from `go install` when Go is present**, which is the
route CI uses and is verified against Go's public checksum database, so both
hosts run one build rather than two builds of one version. The pinned binary is
the fallback. ⚠ The two places that now name that version are compared by
`check-project`, in both halves, over workflows and scripts - and deliberately
not over `TODO/`, because closure evidence is a dated measurement `RULES.md`
forbids rewriting to match today.

⚠ **The three pins are not the same kind of evidence and the file says so.**
PowerShell publishes `hashes.sha256`; the digest here is the line that file
carries, and it matched the bytes this project downloaded. `shellcheck` and
`shfmt` publish no checksum file, so theirs are the bytes observed on
2026-09-08, fetched twice and identical - immutability pins, not authenticity
ones.

⚠ **Residual: CI does not use this script and its `shellcheck` is unpinned.**
The Linux lane pins `shfmt` and takes `shellcheck` and `pwsh` from the runner
image, so the session host now runs a *more* pinned set than CI does. That is a
runner-default question and `CI-08` owns it.
