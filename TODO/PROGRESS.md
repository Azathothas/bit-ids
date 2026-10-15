# Current progress

State instant: 2026-10-14
Total: 67
Open: 22
In progress: 0
Blocked: 0
Done: 45

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
| checking | `tools/check/` | ⭐ one Go binary, `bit-check`, that both gate lanes run, formatted, vetted and harness-tested on the Linux lane. `bit-check --rows` lists what it carries. `CI-10`. |

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
| has a `Profile` been assembled from a dispatch? | yes, from `capture-client` run 17's artifacts on 2026-09-15 and run 36's on 2026-10-14, each pair into a scratch store. Every record is `provisional`. `CI-09`. |
| has a record been produced publishable? | ⭐ yes, from a stock client: `capture-client` run 37's `aria2-next` 3.0.3 pair, re-assembled on 2026-10-14 once `SCHEMA-04` stopped calling a shared random byte fixed, is `publishable` on both routes and `build_equivalent` across them. ⭐ Run 38's 3.0.3 pair, assembled with a `manifest.json` beside each record, is a valid corpus: `validate-corpus` lets 2 of 2 records into a published view. ⭐ `assemble-bundle.sh` makes it a 30-file bundle that the publisher appends to a scratch bare repository and reads back. `CI-09`. |
| does a stock `aria2-next` cooperate? | yes, since run 37 on 2026-10-14: 3.0.3 on both routes answered both peer connections once the second came from `127.0.0.2`, and every sample was corroborated exactly. ⛔ The pair is refused over a byte two random announce tails happened to share, which the classifier calls fixed; `SCHEMA-04` owns that. `OBS-04`, `CLIENT-14`, `CI-09`. |
| whose identity does `aria2-next` present? | qBittorrent 5.2.3's: user agent `qBittorrent/5.2.3`, peer ID prefix `-qB5230-`, corroborated by the `cpython-stdlib` connector. `CLIENT-14`. |
| does a client install finish on a runner? | ⭐ **yes, and it always did.** The "hang" of runs 3 to 34 was *Cut the route* severing the runner agent from GitHub, measured on 2026-10-11 by a flight recorder posting off the host. The capture now runs in a network namespace; both `aria2-next` routes then completed, release in 1.5 minutes and source in 12.5, and 3.0.2's source build installed in 694 seconds on 2026-10-14. `CI-08`, `CI-09`. |
| how many connectors does the capture path declare? | two: the Rust observer and `scripts/capture/connectors/cpython-stdlib.py`. ⚠ The second is two readings of one capture's bytes, not a second observation of the wire. ⭐ Every sample of a field is corroborated, not the first, since 2026-10-14. `OBS-07`, `CI-09`. |
| has the publisher run against this repository's remote? | ⭐ yes: `publish-data` run 38663060205 appended `assemble-bundle` run 38574255232 to `data` on 2026-10-15, 30 objects read back and verified, and a fresh clone passes `sha256sum -c` and `validate-corpus`. `CI-09`. |
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

⛔ **A hang is measured from off the host before it is bounded.** `TODO/RULES.md`
owns the rule and `docs/AGENTS.md` absolute 16 restates it.
`scripts/ci/flight-recorder.sh` is the instrument, and its check run is closed by
the job's last step, in the request that carries its last snapshot, so one
still open is a host that never reached it.

⛔ **Rule 12 is asked of everything a capture uploads, success or refusal.**
`bit-check scan-bundle` is its one home, in `sh` until 2026-10-15 (`CI-11`).

## Work order

⛔ **Nothing here is blocked.** Take these in order.

1. **`CLIENT-06` and `CLIENT-01`**, the remaining vertical captures. ⭐
   `CLIENT-06`'s two 4.1.3 routes build without root and answer 4.1.3, driven
   as an unprivileged user on 2026-10-14. ⭐ The source route's resolution
   failed closed over 112 historical tags git refs cannot date; the scheme
   floor the operator chose is built, and the source route selects 4.1.3 from
   the live refs. ⭐ **Published on 2026-10-15**: run 38680098567's pair is on
   `data` at `4c6ccb1`, beside aria2-next, 4 of 4 records in the published view. ⭐ `CLIENT-01`'s two 5.2.4 routes are built,
   installed and captured, and **published on 2026-10-15**: run
   38705799000's pair is on `data` at `bfeab59`, 6 of 6 records across three
   targets in the published view. Its entry in
   [`clients.md`](clients.md) carries the drive.
2. **`CI-08`**: `capture.yml`'s Linux job runs in `contained.sh`, proved on
   `capture` run 38612214714. ⭐ Its Windows job no longer cuts the host's
   route: on 2026-10-15 it blocks the capture's own programs at the firewall,
   reads the rules back and proves them, the operator's settled shape, proved
   on `capture` run 38722969612: the blocked `curl.exe` exited 7 and the
   capture still verified over loopback. The
   orphaned-subprocess residual and the `sh -c` and `bash -e` split are closed.
3. **`CI-11`**: the capture path in Go, leaves first, then a Windows lane of
   `capture-client.yml` capturing qBittorrent. Every client entry's
   two-platform Prove, and `OBS-10`, wait on it.
4. **`OBS-10`**, which needs a Windows client capture, so it follows `CI-11`.
5. **`CI-07`**: its Approach and Prove were rewritten on 2026-10-14 to the route
   `CI-10` made. `check-corpus`, `check-indexes`, `check-release`,
   `check-formats` and `check-staleness` are ported and run on both lanes;
   `check-connector` too, and `check-assemble` on 2026-10-15, the last portable
   row. ⭐ `check-bitcheck` is Go too since 2026-10-15; its Windows row waits on
   stand-ins that are not shell scripts.
6. **`CI-04`**, provenance and supply-chain hardening, once a release exists to
   bind attestations to.
7. The remaining client and engine breadth, then refinements.

## Settled decisions

Every question an earlier session recorded as needing the operator is answered.
Do not re-raise one.

| question | answer |
| --- | --- |
| how the SQLite rendering gets written | `rusqlite` 0.37.0 with `bundled` and `serialize`, checked against Python's own SQLite on every gate. `PUB-05`. |
| how `LIB-02` reaches bit-cli's tests | the clone works with no credential and no grant. `LIB-02` closed on it and wrote nothing there. |
| whether Windows captures are permitted | yes. A hosted `windows-latest` runner is a fresh virtual machine per job, and its default routes are removed before the capture. `CI-03`. |
| what happens to a first measured record | it publishes automatically once the capture is green. No manual gate. |
| how a two-runner dispatch fits `E-MAN-03`'s phase order | a workflow barrier: every lane's capture waits until every route has installed, so a record's phases are true as written. Settled by the operator on 2026-10-14. `CI-09`. |
| which host a run manifest describes when each route installs on its own runner | the watched host; the other route contributes its acquisition and evidence from its own artifacts, with no schema change. Settled by the operator on 2026-10-14. `CI-09`. |
| whether to append capture run 38's `aria2-next` 3.0.3 bundle to `data` | yes, now. Settled by the operator on 2026-10-15. `CI-09`. |
| how `check-assemble`'s dispatch rows reach the Windows lane | `assemble-dispatch.sh` is rewritten in Go, so nothing on that path is shell, and `check-assemble` ports whole. Settled by the operator on 2026-10-15. `CI-07`. |
| how `capture.yml`'s Windows job is contained | firewall rules blocking outbound traffic for the capture's own executables, with the egress guard reading those rules, in place of cutting the host's route. Settled by the operator on 2026-10-15. `CI-08`. |
| how the Windows client capture lane runs the capture path | the capture path is ported into Go first, then the Windows lane is added; not `sh` under Git for Windows, not PowerShell twins. Settled by the operator on 2026-10-15. `CI-11`. |
| which client is the first Windows capture | qBittorrent, so `CLIENT-01`'s Linux and Windows Prove closes first. Settled by the operator on 2026-10-15. `CI-11`. |
| how the Windows release route acquires qBittorrent | the vendor's `qbittorrent_<version>_lt20_x64_setup.exe` from the GitHub release, its payload extracted with the runner's 7-Zip into the rootless prefix, so no installer code runs. Settled by the operator on 2026-10-15. `CI-11`, `CLIENT-01`. |
| qBittorrent's second Windows route | a package manager, `choco`, `scoop` or `winget`, as long as it installs the same version. Settled by the operator on 2026-10-15. `CI-11`, `CLIENT-01`. |
| how the Linux adapter port and the Windows lane are ordered | in parallel, by subagents, on one Go adapter layer. Settled by the operator on 2026-10-15. `CI-11`. |
| which qBittorrent engine the Windows pair measures | `_lt20` (libtorrent 2.0) on both routes, through whichever of `choco`, `scoop` or `winget` installs it at the same version; the default build on both only if none can. Settled by the operator on 2026-10-15. `CLIENT-01`. |
| what happens if the qBittorrent GUI will not run on a hosted Windows runner | find a headless way; failing that, record the measured blocker in `CLIENT-01` and move on, for the operator to decide. Settled by the operator on 2026-10-15. |
| what follows qBittorrent's Windows pair | Windows captures of the clients already published, Transmission 4.1.3 then aria2-next 3.0.3, before new clients. Settled by the operator on 2026-10-15. |
| how freely workflows are dispatched unattended | as needed, the publish chain for green records included; every red run is debugged and re-dispatched. Settled by the operator on 2026-10-15. |
| a field on which a client's Linux and Windows profiles disagree | a platform-scoped field: each platform's profile is published and the field marked as varying by platform, with its evidence; publication is not held. Settled by the operator on 2026-10-15. |
| whether `capture-run.sh` ports too | yes, into `bit-check capture-run`, with `capture-run.ps1` its Windows half until the Go one runs there. Settled by the operator on 2026-10-15. `CI-11`. |
| a decision nothing settled, met while working unattended | take the conservative, reversible option and record it below under *Decided unattended*. Settled by the operator on 2026-10-15. |
| who authors a commit | the operator, `Azathothas`, per `AGENTS.md` rule 11, whatever a host's hook asks. Settled by the operator on 2026-10-15. |
| how Transmission's source route orders a history git cannot date | a scheme floor: the adapter declares the lowest major its scheme covers, and an unorderable tag below it gets its own verdict rather than blocking. Under `bit-ids/resolution/2`. Settled by the operator on 2026-10-15. `CLIENT-06`. |
| which qBittorrent version the pair targets, and its second route | 5.2.4, the newest stable, built on the `source` lane from the signed `release-5.2.4` tag against Qt's own prebuilt archives. Settled by the operator on 2026-10-15. `CLIENT-01`. |
| how a capture tells a per-session value from a constant one on the peer surface | the peer surface is dialled again in every session, and each connection is attributed to the session its offset falls in. Settled by the operator on 2026-10-15. `CLIENT-06`, `SCHEMA-04`. |
| how a capture keeps a coincidence at a varying span's edge from lengthening a fixed run | three sessions instead of two, so a span has three distinct values; the classifier is unchanged. Settled by the operator on 2026-10-15. `CLIENT-06`, `SCHEMA-04`. |
| which 5.2.4 build the qBittorrent pair measures | the `_lt20` AppImage, libtorrent 2.0.15, on the release route; the source route links libtorrent 2.0.15 built from its own tag. Both run the GUI `qbittorrent` under Xvfb inside the capture namespace. Qt stays 6.11.2 from the prebuilt archives, though the AppImage bundles 6.10.3. Settled by the operator on 2026-10-15, after the AppImages were measured. `CLIENT-01`. |
| whether to take `rusqlite` 0.40.2 | no, settled by the operator on 2026-09-16. It resolves thirteen further packages for the same API. `.github/dependabot.yml` ignores every `rusqlite` version from 0.38, and dependabot closed pull request 2 itself on 2026-09-29 after reading it. |

⛔ **A capture host was never a blocker.** A hosted runner is a fresh virtual
machine per job, so `--claim` passes, and `--egress` passes once the default
route is deleted. Measured on 2026-09-08 with the guard's own route-table
argument.

## The local gate and CI

⭐ **What to run before a push, by what changed, and how long each takes, is
the table at the top of [`../scripts/README.md`](../scripts/README.md).** Each row
mirrors the CI job that would refuse the change.

| fact | state |
| --- | --- |
| the gate, measured 2026-10-11 | 40 checks, 39 passed, 0 failed, 1 skipped, exit 0 |
| the one expected skip | `check-remote-items` against the real open items, which needs an authenticated `gh`; installing `gh` does not fix it on a session host. Both CI lanes run it |
| CI on `main` | read back through `AGENTS.md` rule 8's route for the tip commit before trusting any recorded green |
| `go` | a gate dependency; without it the ported rows skip, which `--strict` turns red on CI |

⚠ The defect classes this repository has shipped and caught, which every review
should know before it starts, are in [`../docs/lessons.md`](../docs/lessons.md).

## Decided unattended

Choices made while working unattended under the operator's standing rule, each for review.

| decision | why | where |
| --- | --- | --- |
| the Windows lane runs only when the new `windows_adapters` input names an adapter, and the default is `[]` | a bare dispatch then runs what it ran before, and a Windows lane nobody asked for costs a runner. Reversed by changing one default | `capture-client.yml`, `CI-11` |
| the winget route installs with no `--version` | winget's index is that route's resolver, as apt's is on Linux. Pinning the release route's version would make the two routes share a resolver, which `E-ACQ-07` refuses, and absolute 4 is checked after install anyway | `adapter_qbittorrent_windows.go`, `CLIENT-01` |
| `version` and `stop` drive the build through its Web UI on loopback, with `LocalHostAuth=false` and the hash of a discarded random password, which `stop` scrubs from the profile | the Windows GUI build compiles `--version` out and has no signal for a graceful exit. Its Web UI will not start with no password hash. The scrub, plus `capture-client`'s rule-12 scan, keeps the hash out of the bundle | `adapter_qbittorrent_windows.go`, `qbtwindows.go`, `CLIENT-01` |
| the `package` slugs are `nsis-payload` for the release route and `nsis-installer` for the winget route | one installer's bytes reach the host in two forms, extracted with no installer code run and placed by the installer, and `Build.package` names the form | `adapter_qbittorrent_windows.go`, `CLIENT-01` |
| the prover is `bit-check probe-egress`, run as the capture's own blocked `bit-check`, rather than adding `curl.exe` to the list | a client capture runs no `curl`, and blocking a program only to prove with it proves nothing about the programs that run. It passes only on a connect the host refused | `probeegress.go`, `capture-client.yml`, `CI-11` |
| the Windows lane's barrier arrives as `<adapter>-windows` | its pair is a separate run manifest. Under the bare adapter name a Linux `release` lane and a Windows `release` lane of one dispatch would satisfy each other | `capture-client.yml`, `CI-11` |
| no `compiles` key on the Windows half; it declares `routes=package,release` | neither route compiles, and a key with no value could not be told from one that declines to say. The Windows install bound is fixed in the workflow | `adapter_qbittorrent_windows.go` |
| the barrier's token goes to `curl` through a header file, `-H @file`, mode 0600, and not as an argument | a token in an argument list can be read by any process on the host. The harness stub's `-H` arm already skips either spelling | `installbarrier.go` |
| Transmission's Windows release route lays the MSI out with `msiexec /a` (an administrative image) rather than 7-Zip | the MSI's AdminExecuteSequence, read on 2026-10-15, runs no custom action, service, registry write or launch condition, and Windows Installer lays the files out by the package's own Directory and File tables. 7-Zip would give cabinet members named by File-table keys in one flat directory, so this project would have to reimplement that layout | `adapter_transmission_windows.go`, `CLIENT-06` |
| its release asset is `transmission-{version}-x64.msi`, not the `qt5` variant | it is the MSI winget's x64 installer names, with the same digest, so both routes deliver one artifact; the literal matches one of the four MSIs | `adapter_transmission_windows.go`, `CLIENT-06` |
| the winget route passes `--custom ADDLOCAL=CommonLibs,Daemon,CmdTools` | the MSI's `Daemon` and `CmdTools` features are level 2 and no `INSTALLLEVEL` is set, so a default install places neither executable. The features are named rather than everything installed, and a misspelt one makes msiexec refuse loudly | `trwindows.go`, `CLIENT-06` |
| the winget route stops and disables the `Transmission` service the MSI installs and starts, and refuses if it is still running after 30 seconds | the Linux half stops the distribution's service for the same reason: a capture must own the only running daemon. A service that is not installed is recorded, not refused | `adapter_transmission_windows.go`, `CLIENT-06` |
| the winget route refuses when the installed build's version differs from the version winget's index named | an index that installed something other than what it named is a defect of that route, not a disagreement for `ACQ-03`. The record's version is still the one `version` asks for | `adapter_transmission_windows.go`, `trwindows.go`, `CLIENT-06` |
| Transmission's Windows `stop` sends `transmission-remote --exit`, waits 10 seconds, then kills, and writes `stop.txt` | `session-close` reaches the same `stop()` that SIGTERM reaches on Linux, and Windows has no signal for a console process in another process group. The grace matches the qBittorrent half's | `adapter_transmission_windows.go`, `CLIENT-06` |
| its scrub is Go, removes every settings member whose name holds `password`, keeps the rest in order, and deletes a file it cannot parse | a Windows runner has no `jq`. The broader key match is the safer side, and deleting the file is the Linux half's own fallback | `trwindows.go`, `CLIENT-06` |
| its RPC port is a free loopback port drawn per session, written to `rpc-port`; `BIT_IDS_TRANSMISSION_RPC` still overrides | the package route's own service is installed to listen on 9091, the Linux half's fixed port | `adapter_transmission_windows.go`, `CLIENT-06` |
| the `package` slugs are `msi-payload` for the release route and `msi-installer` for the winget route | the qBittorrent half's distinction for one installer's bytes reaching the host in two forms, applied to an MSI | `adapter_transmission_windows.go`, `CLIENT-06` |
| Transmission's version reading requires `transmission-daemon` as the first field and a version-shaped second field, where the Linux half takes any second field | a usage text or an error message is then refused rather than recorded as a version | `trwindows.go`, `CLIENT-06` |
