# Current progress

State instant: 2026-10-11
Total: 66
Open: 21
In progress: 1
Blocked: 0
Done: 44

⚠ Those five counts are compared against [`INDEX.md`](INDEX.md) by
`check-project` on every gate. Nothing else in this header is checked, so no
commit is named here: a hash written into the commit that changes it can only
name the one before.

This page says what is true now. How it came to be true is in
[`../docs/history/`](../docs/history/); the record as it stood before this page
was rewritten is
[`../docs/history/PROGRESS-2026-09-29.md`](../docs/history/PROGRESS-2026-09-29.md).

## Current state

### What exists

Every layer a capture passes through is written and closed.
[`../docs/architecture.md`](../docs/architecture.md) is the technical authority
for all of it.

| layer | crate or scripts | what it owns |
| --- | --- | --- |
| record shape | `bit-ids` | the published record, the run manifest, agreement, sampling, and one validating read and write path |
| acquisition | `bit-ids`, `scripts/acquisition/` | which version is newest, whether two routes agree, where an artifact came from, the rootless installer, and the boundary that runs before an install |
| wire | `bit-ids-wire` | byte-exact codecs for every observed surface and the fixture corpus each is parsed against |
| lab | `bit-ids-lab` | the sockets, the deadline, the ordered byte record, the generated torrent, and the evidence a run leaves |
| observers | `bit-ids-probe` | what each surface answers with, one module per surface, and the capture assembler |
| store | `bit-ids` | where a record is filed, what a successor tree may do to it, and the views a consumer reads |
| capture | `scripts/capture/`, `.github/workflows/capture.yml`, `.github/workflows/capture-client.yml` | the host a capture may run on, the order that keeps it contained, and the evidence a run uploads |
| publishing | `bit-ids`, `scripts/publishing/` | the assembled bundle, the append-only push, the six renderings including SQLite, and the access contract |
| consuming | `bit-ids` | reading a publication back, verified, with no way to reach a network |
| maintenance | `bit-ids`, `scripts/ci/` | what a new stable release creates, and what it must not create twice |
| checking | `tools/check/` | ⭐ one Go binary, `bit-check`, that both gate lanes run. `bit-check --rows` lists what it carries. |

### What has been measured, and what has not

⛔ **Nothing is published, and every record in the tree is synthetic.** The
schema fixtures under
[`../crates/bit-ids/tests/fixtures/`](../crates/bit-ids/tests/fixtures/)
describe a target that does not exist, and the wire fixtures under
[`../crates/bit-ids-wire/tests/fixtures/`](../crates/bit-ids-wire/tests/fixtures/)
were written by hand from published BEPs.

| question | answer, with where the evidence is |
| --- | --- |
| has a stock client been put on the wire? | yes. Transmission 4.0.5 and qBittorrent 4.6.3 announced to the HTTP tracker observer on 2026-09-08, one observer, one platform, one route. `OBS-07`. |
| has a `Profile` been assembled from a dispatch? | yes, from `capture-client` run 17's artifacts, on 2026-09-15, into a scratch store. Both records are `provisional`. `CI-09`. |
| has a record been produced publishable? | only over a stub target written to announce twice and accept two connections, on 2026-09-15. That proves the path, not a product. `CI-09`. |
| does a stock `aria2-next` cooperate? | not yet. Runs 18 and 19 opened two peer connections and it answered one; the store's comparable capture conflicts, so `E-PUB-03` refuses the pair. `OBS-04` carries the next hypothesis, the source address. |
| whose identity does `aria2-next` present? | qBittorrent 5.2.3's: user agent `qBittorrent/5.2.3`, peer ID prefix `-qB5230-`, corroborated by the `cpython-stdlib` connector. `CLIENT-14`. |
| does a client install finish on a runner? | ⛔ **not reliably, and it is a dead end.** *Install the client* wedged in run 32 after `ACQ-06` removed every privilege from it, and in nine runs before that on byte-identical code. `TODO/RULES.md`; `ACQ-06` and `CI-08` carry the measurements. |
| how many connectors does the capture path declare? | two: the Rust observer and `scripts/capture/connectors/cpython-stdlib.py`. ⚠ The second is two readings of one capture's bytes, not a second observation of the wire. `OBS-07`. |
| has the publisher run against this repository's remote? | no, and it must not until a measured record exists. It cannot run at all yet: `publish-data.yml` downloads an artifact named `bundle` and no workflow uploads one. |
| has a published URL been fetched? | no. `docs/publishing.md` carries the forms and says they are unexercised. |
| does anything schedule the staleness monitor? | no. `CI-02` built the comparison; no capture request has been opened. |

### The rules that are load-bearing

⛔ **A published value is observed from a running build.** Source code, client-ID
tables, release notes, UI labels, self-reports and swarm statistics may set
priority and may not populate a field.

⛔ **A path is derived from the record's whole identity tuple, never composed.**
`store::StoreKey` is the one derivation.

⛔ **`VersionScheme::components` is the project's only version ordering.**
Sorting version text answers `1.2.9` over `1.2.10`.

⛔ **Valid and publishable are separate gates at every level.** A record carrying
a disagreement validates, so the evidence of the disagreement is kept;
`publishable` refuses it. ⭐ `publishable_among` asks that question with the store
visible, and every comparable sibling is read rather than the first that agrees.

⛔ **A capture declaring one connector is invalid, not merely unpublishable.**
`E-CAP-01` fires inside `validate`.

⛔ **Every socket, outbound datagram and address handed to a build goes through
`bind.rs`.** `bind::check_offered` guards an address the lab puts inside a
message for the target to dial.

⛔ **A transcript is never scrubbed.** The bytes a build put on the wire are the
measurement.

⛔ **Nothing in the `bit-ids` crate can reach a network.** Retrieval is a trait
the caller implements.

⛔ **A hang is a dead end, not a subject.** `TODO/RULES.md` owns the rule and
`docs/AGENTS.md` absolute 16 restates it. Thirteen bounds failed on the capture
install step; do not diagnose it, instrument it, or add a fourteenth.

## Work order

⛔ **Nothing here is blocked.** Take these in order.

1. **`CI-10`'s residuals**, the entry in progress. The port is done: no twin file
   pair is left, `check-twins` compares only the doctor's probe pair, and
   `bit-check --rows` lists the ported rules. What is left, under the entry in
   [`ci.md`](ci.md): nothing runs `check-bitcheck --harnesses` automatically, and
   CI runs no `gofmt` and no `go vet` over [`../tools/check/`](../tools/check/).
2. **`CLIENT-01` and `CLIENT-06`**, the remaining vertical captures. ⛔ They do
   not wait on a dispatch any more: run 32 answered it, and *Install the client*
   wedges with no privilege in it. The work is a capture route that does not run
   that step as it stands, which absolute 16 asks for and nothing has tried.
3. **`CI-09`'s residuals**: the `RunManifest` a record needs beside it, and an
   artifact named `bundle` for the publisher to download. The assembler already
   asks `validate_corpus` about the store it writes.
4. **`OBS-10`**, which needs a second platform, so it follows the captures.
5. **`CI-07`**, whose premise the port changed. A declared Windows row now closes
   by its harness joining `bit-check`, not by gaining a PowerShell half, so the
   entry's Approach and Prove need rewriting to that route before any row is
   worked.
6. **`CI-08`**, runner-default drift, including the orphaned-subprocess residual
   under it.
7. **`CI-04`**, provenance and supply-chain hardening, once a release exists to
   bind attestations to.
8. The remaining client and engine breadth, then refinements.

## Settled decisions

Every question an earlier session recorded as needing the operator is answered.
Do not re-raise one.

| question | answer |
| --- | --- |
| how the SQLite rendering gets written | `rusqlite` 0.37.0 with `bundled` and `serialize`, checked against Python's own SQLite on every gate. `PUB-05`. |
| how `LIB-02` reaches bit-cli's tests | the clone works with no credential and no grant. `LIB-02` closed on it and wrote nothing there. |
| whether Windows captures are permitted | yes. A hosted `windows-latest` runner is a fresh virtual machine per job, and its default routes are removed before the capture. `CI-03`. |
| what happens to a first measured record | it publishes automatically once the capture is green. No manual gate. |
| whether to take `rusqlite` 0.40.2 | no, settled by the operator on 2026-09-16. It resolves thirteen further packages for the same API. `.github/dependabot.yml` ignores every `rusqlite` version from 0.38, and dependabot closed pull request 2 itself on 2026-09-29 after reading it. |

⛔ **A capture host was never a blocker.** A hosted runner is a fresh virtual
machine per job, so `--claim` passes, and `--egress` passes once the default
route is deleted. Measured on 2026-09-08 with the guard's own route-table
argument.

## The local gate

⛔ **Run `sh scripts/common/check-gate.sh` after the last edit.** A subset chosen
by hand is not the same gate twice; `check-no-secrets` runs twice in it and only
the second invocation carries the public rules.

| fact | detail |
| --- | --- |
| measured 2026-10-11 | 40 checks, 39 passed, 0 failed, 1 skipped, exit 0, cold, on `5d54c5f` |
| the one skip | `check-remote-items` against the real open items, which needs an authenticated `gh`. ⚠ Installing `gh` does not fix it on a session host. Both CI lanes run it, and `check-bitcheck` runs its logic here against a stub `gh`. |
| tools | ⭐ `sh scripts/doctor/provision.sh` installs `pwsh`, `shellcheck` and `shfmt`, each verified against a pinned digest. Run it before touching a script. |
| `go` | a gate dependency. Without it the ported rows skip, and both CI lanes run `--strict`, so an absent toolchain turns a lane red. |
| not in the gate | `cargo clippy`, `cargo fmt --check`, the test suite, `sh scripts/ci/check-workflow.sh`, and `check-bitcheck --harnesses`. |
| ⛔ while it runs | do not edit the tree: `tree-unchanged` compares before and after. |

⚠ The defect classes this repository has shipped and caught, which every review
should know before it starts, are in [`../docs/lessons.md`](../docs/lessons.md).
