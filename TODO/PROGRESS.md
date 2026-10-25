# Current progress

State instant: 2026-10-25
Total: 70
Open: 12
In progress: 1
Blocked: 0
Done: 57

⚠ Those five counts are compared against [`INDEX.md`](INDEX.md) by
`check-project` on every gate. Nothing else in this header is checked, so no
commit is named here: a hash written into the commit that changes it can only
name the one before.

This page says what is true now. How it came to be true is in
[`../docs/history/`](../docs/history/); the record as it stood before this page
was rewritten is
[`../docs/history/PROGRESS-2026-09-29.md`](../docs/history/PROGRESS-2026-09-29.md).

## Current state

### Where the repository lives

`Azathothas/bit-ids`, since 2026-10-25. `main` and `data` were pushed there
with their whole history, so every commit hash this record cites still
resolves. Beyond this record, the changelog and the handoff, no tracked file
names the repository: workflows and scripts read it from `github.repository`
or the clone's `origin`.

⚠ **GitHub's own state stayed with the previous home.** Actions runs, check
runs, artifacts and pull requests are not part of the git history. A run
number recorded before 2026-10-25 is a citation and cannot be fetched here:
`repos/Azathothas/bit-ids/actions/runs/38663060205` answered 404 through rule
8's route on 2026-10-25.

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

⭐ **36 measured records are published on `data`**, as of `ab08fa3` on
2026-10-22: qBittorrent 5.2.4, qBittorrent Enhanced 5.2.4.10, Transmission
4.1.3, Deluge 2.2.0, BiglyBT 4.1.0.0, aria2-next 3.0.3 and 3.0.4, rqbit 9.0.1 and
anacrolix/torrent v1.61.0 on Linux and Windows, and aria2 1.37.0 on Linux. Every
pair was captured on two independent routes and is `build_equivalent`.
⚠ **Every record in the tree itself is still synthetic.** The
schema fixtures under
[`../crates/bit-ids/tests/fixtures/`](../crates/bit-ids/tests/fixtures/)
describe a target that does not exist, and the wire fixtures under
[`../crates/bit-ids-wire/tests/fixtures/`](../crates/bit-ids-wire/tests/fixtures/)
were written by hand from published BEPs. ⚠ The one exception is
[`../crates/bit-ids-probe/tests/fixtures/cross-platform-events/`](../crates/bit-ids-probe/tests/fixtures/cross-platform-events/),
two transcripts copied unmodified from a published capture's raw evidence for
`OBS-10`. They are test input, not records.

| question | answer, with where the evidence is |
| --- | --- |
| has a stock client been put on the wire? | yes. Transmission 4.0.5 and qBittorrent 4.6.3 announced to the HTTP tracker observer on 2026-09-08, one observer, one platform, one route. `OBS-07`. |
| has a `Profile` been assembled from a dispatch? | yes, from `capture-client` run 17's artifacts on 2026-09-15 and run 36's on 2026-10-14, each pair into a scratch store. Every record is `provisional`. `CI-09`. |
| has a record been produced publishable? | ⭐ yes, from a stock client: `capture-client` run 37's `aria2-next` 3.0.3 pair, re-assembled on 2026-10-14 once `SCHEMA-04` stopped calling a shared random byte fixed, is `publishable` on both routes and `build_equivalent` across them. ⭐ Run 38's 3.0.3 pair, assembled with a `manifest.json` beside each record, is a valid corpus: `validate-corpus` lets 2 of 2 records into a published view. ⭐ `assemble-bundle.sh` makes it a 30-file bundle that the publisher appends to a scratch bare repository and reads back. `CI-09`. |
| does a stock `aria2-next` cooperate? | yes, since run 37 on 2026-10-14: 3.0.3 on both routes answered both peer connections once the second came from `127.0.0.2`, and every sample was corroborated exactly. ⛔ The pair is refused over a byte two random announce tails happened to share, which the classifier calls fixed; `SCHEMA-04` owns that. `OBS-04`, `CLIENT-14`, `CI-09`. |
| whose identity does `aria2-next` present? | qBittorrent 5.2.3's: user agent `qBittorrent/5.2.3`, peer ID prefix `-qB5230-`, corroborated by the `cpython-stdlib` connector. `CLIENT-14`. |
| does a client install finish on a runner? | ⭐ **yes, and it always did.** The "hang" of runs 3 to 34 was *Cut the route* severing the runner agent from GitHub, measured on 2026-10-11 by a flight recorder posting off the host. The capture now runs in a network namespace; both `aria2-next` routes then completed, release in 1.5 minutes and source in 12.5, and 3.0.2's source build installed in 694 seconds on 2026-10-14. `CI-08`, `CI-09`. |
| how many connectors does the capture path declare? | two: the Rust observer and `scripts/capture/connectors/cpython-stdlib.py`. ⚠ The second is two readings of one capture's bytes, not a second observation of the wire. ⭐ Every sample of a field is corroborated, not the first, since 2026-10-14. `OBS-07`, `CI-09`. |
| has the publisher run against this repository's remote? | not since the move to `Azathothas/bit-ids`, whose `data` arrived by push. ⭐ On the previous home, yes: `publish-data` run 38663060205 appended `assemble-bundle` run 38574255232 to `data` on 2026-10-15, 30 objects read back and verified, and a fresh clone passes `sha256sum -c` and `validate-corpus`. `CI-09`. |
| has an engine target been acquired? | ⭐ yes, rqbit 9.0.1 on 2026-10-21: the vendor's release binary and crates.io's crate, built with the release's default features and profile, both installed rootless on the session host and answered 9.0.1. ⛔ Neither can run there: the host boots with IPv6 disabled and the build always binds an IPv6 socket. A runner dispatch is next. `ENGINE-03`. ⭐ And anacrolix/torrent v1.61.0 on 2026-10-22, through a harness this tree pins: the module proxy and the tag from GitHub, every module from its own origin, built the same executable byte for byte, captured `-GT0003-` and `anacrolix-torrent/v1.61.0` on this host, and assembled `build_equivalent`. A runner dispatch is next. `ENGINE-02`. ⭐ And libtorrent 2.0.15 on 2026-10-22, through a Python harness this tree commits: PyPI's wheel and the tag built by its own `setup.py` reported one feature set and the same 258 default settings, captured `-LT20F0-` and `libtorrent/2.0.15.0` on this host, and assembled `build_equivalent`. Linux only; Windows has no half yet. ⚠ The first runner capture, run 39783974176, assembled the same, and assemble-bundle run 39784872974 refused it with `E-VIW-02`: the scheme read the series as a prefix. The scheme now pins the series instead, and a new dispatch is next. `ENGINE-01`. |
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
`bit-check flight-recorder` is the instrument, and its check run is closed by
the job's last step, in the request that carries its last snapshot, so one
still open is a host that never reached it.

⛔ **Rule 12 is asked of everything a capture uploads, success or refusal.**
`bit-check scan-bundle` is its one home, in `sh` until 2026-10-15 (`CI-11`).

## Work order

⚠ **Paused on 2026-10-22 for a transfer and refactor of the repository**, settled by
the operator. The transfer landed on 2026-10-25; the refactor has not started.
⭐ **`CI-13` is in progress**, asked by the operator on 2026-10-25: the CI wall
clock, shortened without dropping a guarantee. It closes when a CI run on this
remote measures it. [`ci.md`](ci.md).
12 entries are open. In the order a resuming session would take them:

1. **`ENGINE-01`**, libtorrent 2.0.15. The Linux pair is built and was captured
   on runner run 39783974176, and its view refusal is fixed by the release
   series in the scheme (`cc76d1e`). Next: dispatch `capture-client` with
   `adapters: ["libtorrent"]`, `routes: ["package","source"]`,
   `windows_adapters: []`, then assemble and publish. The Windows half (PyPI's
   wheel plus an MSVC source build) is unbuilt. [`engines.md`](engines.md).
2. **`CI-07`**: the Windows rows that wait on stand-ins that are not shell
   scripts. [`ci.md`](ci.md).
3. **`CI-12`**: `check-step-bodies` independent of host load.
4. **`CLIENT-15`**: aria2 as an independent connector.
5. **`CI-04`**, provenance and supply-chain hardening, once a release exists to
   bind attestations to.
6. **The proprietary Windows clients**, `CLIENT-03` uTorrent, `CLIENT-04`
   BitComet, `CLIENT-08` BitTorrent, `CLIENT-12` Free Download Manager and
   `CLIENT-13` Zona, which need silent installs and bounded UI automation, and
   `CLIENT-10` Tixati, freeware for Linux and Windows whose headless driving is
   unmeasured.
7. **`CLIENT-11`**, KTorrent: blocked until KDE ships a binary, by the
   operator's decision.

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
| how `CLIENT-07`'s *agreeing observed surfaces* is read | within a platform: each platform's two routes must agree. A cross-platform difference that follows the vendor's own engine choice is a recorded finding, not a blocker. Settled by the operator on 2026-10-22. `CLIENT-07` closed on it. |
| whether a client's Prove that says *a run* is met by Linux and Windows records from two runs | yes. Settled by the operator on 2026-10-22. `CLIENT-01` and `CLIENT-06` closed on it. |
| whether `CLIENT-05` closes before aria2 is a connector for other targets | yes, on its Prove; the connector role is `CLIENT-15`. Settled by the operator on 2026-10-22. |
| what happens to `CLIENT-11`, KTorrent, with no rootless KDE binary | it stays OPEN and blocked until KDE ships a binary at the newest stable. The nixpkgs route is not admitted. Settled by the operator on 2026-10-22. |
| whether `CI-08` closes with `check-step-bodies`' load-sensitive case unresolved | yes; the case is `CI-12`. Settled by the operator on 2026-10-22. |
| whether BiglyBT's two unswitchable vendor calls are measured on a Windows runner | no; the possible DNS query stays a recorded open question in `CLIENT-09`. Settled by the operator on 2026-10-22. |
| what follows 2026-10-22 | cleanup and a saved handoff: the repository is being transferred and refactored. Settled by the operator on 2026-10-22. |
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
| *Build bit-check* moved ahead of *Start the flight recorder* in `capture-client.yml`'s Linux job | the recorder is `bit-check flight-recorder` now and `BIT_CHECK` reaches only later steps. A Go build with no dependencies fetches nothing, so the one step the recorder cannot see cannot wedge on the network. Reversed by building the recorder in its own step | `capture-client.yml`, `CI-11` |
| `check-step-bodies` runs `bit-check contained` in a real network namespace through `sudo -n`, and a host that cannot make one is a failed row | a namespace that was never made has verified nothing, and the `pwsh` cases already treat an absent tool that way. The CI gate's runner has passwordless `sudo` and makes namespaces, as every capture dispatch shows | `check-step-bodies.sh`, `CI-11` |
| the recorder's two `sh -c` pipelines (the newest 40 tasks, the newest files) stay inline host-tool commands, byte for byte | they are not script files and the recorder is Linux-only; a Go spelling of `ps \| head` and `find \| sort \| tail` would change what a timed-out read reports. The request bodies are written in Go, so `jq` is no longer required | `flightrecorder.go`, `CI-11` |
| a recorder interval or limit that is not a whole number is exit 2 at the start | the shell half's `test` failed inside the loop instead: a bad limit closed the run at once and a bad interval posted without a pause. No workflow sets either | `flightrecorder.go`, `CI-11` |
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
| aria2-next's second Windows route is Scoop's `main/aria2-next`, installed with no version | it is the one package manager that carried the vendor's version: 3.0.3 from bucket commit `1359d92` and 3.0.4 since `2384d7c`. winget's `AnInsomniacy.Aria2Next` stops at 2.8.3, its 3.0.3 pull request closed unmerged, and Chocolatey's feed has no such package. The bucket is the route's resolver, as winget's index is the other Windows halves', so pinning the release route's version would share a resolver | `adapter_aria2-next_windows.go`, `a2nwindows.go`, `CLIENT-14` |
| the pair is recorded as sharing its origin, as Transmission's does | Scoop's x64 URL is the release route's own GitHub asset and its hash is the vendor's digest. The resolvers, deliveries and checks differ; the host serving the bytes does not, and Scoop's bot follows the vendor's releases | `CLIENT-14` |
| the adapter pins no version, so a dispatch now captures 3.0.4 on Windows, not 3.0.3 | the vendor published 3.0.4 on 2026-10-15 and Scoop followed, so neither route resolves 3.0.3 any more, and absolute 5 targets the newest stable only. The published Linux pair is 3.0.3 and no Linux 3.0.4 pair exists; whether to capture Linux 3.0.4 beside it is the operator's | `CLIENT-14` |
| Scoop is installed by its own installer, fetched through `install-rootless --fetch` with its digest recorded and none to check, run with `-ScoopDir %USERPROFILE%\scoop -RunAsAdmin`; a Scoop already on the host is refused | the root is Scoop's default, named so a host's `SCOOP` does not move it; a hosted runner's user is an administrator, which the installer refuses without the switch; an existing Scoop would bring an index state and builds this route did not acquire | `adapter_aria2-next_windows.go`, `CLIENT-14` |
| the Scoop route keeps the manifest and the bucket's git commit before installing, installs with `--no-update-scoop`, and refuses unless the placed file digests to the manifest's hash and answers its version, and unless the manifest's URL names that version's x64 file | the bucket at a named commit is the index's state, and the update switch keeps it there. An index whose fields disagree, or a placed file that is not the one the index pinned, is a defect of the route rather than a disagreement for `ACQ-03` | `adapter_aria2-next_windows.go`, `a2nwindows.go`, `CLIENT-14` |
| the build is found as the one version directory under `apps\aria2-next`, never `current`, never through PATH | Scoop puts a shim named `aria2-next.exe` on PATH, which is Scoop's executable; the firewall and `version` must be given the build's own path. Two version directories are refused rather than chosen between | `a2nwindows.go`, `CLIENT-14` |
| the `package` slug is `pe-binary` on both Windows routes | both deliver the vendor's bare PE executable as a file, which is the form; the Linux half calls both of its bare executables `elf-binary` | `a2nwindows.go`, `CLIENT-14` |
| the option read-back is one function, `aria2NextOptionsRefusal`, which the Linux half now calls with its messages unchanged | two copies of the switch list, the family sweep and the judged-true list would drift. The Linux `start` and `stop` were driven against the 3.0.3 build after the move, exit 0 and the token gone | `a2nwindows.go`, `adapter_aria2-next.go`, `CLIENT-14` |
| Windows `start` passes the Linux flags in their order, RPC port 6800 with the Linux override, the stock peer ID untouched; RPC is the adapter's own HTTP client with no proxy, so no `helpers=` | the Windows 3.0.3 executable carries every option key the Linux build reports and the same discovery strings; Scoop installs no service holding 6800; and RPC from `bit-check`, which is already on the firewall list, adds no second program to it | `adapter_aria2-next_windows.go`, `a2nwindows.go`, `CLIENT-14` |
| Windows `stop` deletes the token on every path, sends `aria2.shutdown`, waits ten seconds, then kills, and writes `stop.txt` | ten seconds is the Linux half's measured grace; Windows has no SIGTERM for a console process in another process group | `adapter_aria2-next_windows.go`, `CLIENT-14` |
| the Windows release route with no resolution is could-not-run | the adapter declares the vendor's checksums document and no reason to install without one, which is the Linux half's standalone behaviour | `adapter_aria2-next_windows.go` |
| the Windows version reading also requires `aria2` on the first line and a version-shaped field after `version` | a usage text or an error message is then refused rather than recorded, as Transmission's Windows half refuses one | `a2nwindows.go`, `CLIENT-14` |
| whether `CLIENT-01` and `CLIENT-06` close now that agreeing Linux and Windows profiles are published | left OPEN: the Prove says "a run", and the two platforms were published from two runs at one version, because the store never rewrites the Linux records. The operator decides. 2026-10-16. ⭐ Settled by the operator on 2026-10-22: closed. | `CLIENT-01`, `CLIENT-06` |
| whether `CLIENT-05` closes now that aria2 1.37.0's two rootless routes are published and `E-COR-20` refuses self-corroboration | left OPEN: the Prove is met as written, but the title and Approach also name aria2 as a connector for other targets, which nothing does yet. Closing on the Prove alone is the operator's call. 2026-10-21. ⭐ Settled by the operator on 2026-10-22: closed, the connector role is `CLIENT-15`. | `CLIENT-05` |
| whether `CLIENT-07` closes now that Deluge 2.2.0 is published on Linux and Windows | left OPEN: each platform's pair is `build_equivalent`, but Linux runs libtorrent 2.0.15 and Windows 1.2.19, so the user agent and reserved bytes differ across platforms. Whether the Prove's *agreeing observed surfaces* means within a platform or across them is the operator's reading. 2026-10-22. ⭐ Settled by the operator on 2026-10-22: closed, agreement read within a platform. | `CLIENT-07` |
| `OBS-10`'s "one client build" is one captured run's recorded bytes, fed to the observer as compiled on each platform | a client build cannot be byte-identical across operating systems, and comparing two platforms' own captures would mix the clients' platform differences into a test of the observer. Holding the bytes fixed makes a difference the observer's. The fixtures are qBittorrent 5.2.4's Windows `package` transcripts from `data` at `4221e28`. Reversed by replacing the fixtures and the constant. 2026-10-16. | `cross_platform_events.rs`, `OBS-10` |
| the normalization moved out of the `assemble-capture` example into `bit_ids_probe::normalize`, unchanged, and the example calls it | a test cannot call an example's functions, and a copy in the test would prove the copy. The four fields it gives for the fixtures equal the published record's | `normalize.rs`, `assemble-capture.rs`, `OBS-10` |
| `OBS-10` stays OPEN rather than DONE | its Prove names a Windows run, and this host is Linux. It closes on a `CI` run whose Windows job shows the step green | `OBS-10` |
| Deluge's routes are PyPI's wheel and the `deluge-2.2.0` tag on Linux, and winget's `DelugeTeam.Deluge` and the same tag on Windows; no `release` route on either | Deluge 2.2.0's GitHub repository has tags and no release objects, so `resolve-release` has nothing to read. Ubuntu 24.04 carries 2.1.2~dev0 and Chocolatey 2.1.1. Scoop names the installer winget does, so it would be a second package route on one host. PyPI and winget are the package managers at the newest stable, and the tag is the other route on each platform | `adapter_deluge.go`, `adapter_deluge_windows.go`, `CLIENT-07` |
| the prerelease reader reads PEP 440's lone `a`, `b` or `c` between two digits as a marker | `deluge-2.0.0b1` and `deluge-2.0b1` were unorderable at the floor's major and failed the source route closed. Over every target's live refs, the only other move is Transmission's thirteen `1.80b5` to `2.50b1`, from `below_scheme_floor` to `prerelease_by_version`. No selection changes, and published resolutions still validate, because `E-RES-09` does not ask about markers. Reversed by deleting `pep440_letter` | `resolution.rs`, `check-source-route.sh`, `CLIENT-07` |
| Deluge's scheme is `deluge-`, three components, floor 2 | the 0.5 series' four-component tags are below the floor; every other tag is ordered or marked | `adapter_deluge.go`, `CLIENT-07` |
| the pip routes pin `libtorrent==2.0.15` (1.2.19 on Windows since 2026-10-21, below), `setuptools<81` and `pyOpenSSL<26.3`, and nothing else; Deluge itself is unpinned on PyPI | Deluge declares no libtorrent, and PyPI's newest is the 2.1 series. 2.0.15 matches the published qBittorrent pair. setuptools 84 has no `pkg_resources` and pyOpenSSL 26.3 has no `X509Req`; each stopped the daemon at start in a measured drive | `deluge.go`, `CLIENT-07` |
| a pip route also runs an import check of libtorrent, the daemon, the console and `X509Req`, and refuses if it fails | both broken builds answered `--version`, which runs before those modules load | `adapter_deluge.go`, `adapter_deluge_windows.go`, `CLIENT-07` |
| the Windows routes run `deluged-debug.exe`, not `deluged.exe` | the vendor's `deluged.exe` is a windowed PyInstaller build whose `--version` prints nowhere; `-debug` is the same program with a console, and pip makes it on Windows too | `adapter_deluge_windows.go`, `CLIENT-07` |
| ⛔ REVERSED on 2026-10-21: the Windows package route no longer accepts winget's `http://` installer URL as the origin, and `wingetInstallerURL` is https-only again for every route | assemble-bundle run 39617383153 refused the pair: `Url::parse` refuses a plain http retrieval, rightly. The route now fetches the same bytes over https; see the rows dated 2026-10-21 below | `adapter.go`, `deluge.go`, `CLIENT-07` |
| the Windows source clone runs behind Git for Windows' coreutils `timeout.exe`, called by the name `timeout` with its path set | System32's `timeout.exe` is a pause, and `check-adapters` requires a clone behind a `timeout` named in the call; a `check-bitcheck` plant removes the name and is refused | `adapter_deluge_windows.go`, `bitcheck.go`, `CLIENT-07` |
| Deluge is driven by `deluge-console` over the daemon's RPC on a free loopback port, connected as `127.0.0.1` with no user, in a fresh config directory per session | the console then reads the `localclient` credential from the config directory it shares with the daemon, so the adapter never holds it; a previous session's state would resume the torrent before the add | `deluge.go`, `CLIENT-07` |
| encryption is `enc_in_policy` 1, `enc_out_policy` 1, `enc_level` 2, and uTP is left at its default | these are libtorrent's `pe_enabled` both ways, which is qBittorrent's `Session\Encryption=0` on the same engine; Deluge 2.2.0 has no uTP switch, as qBittorrent's adapter sets none | `deluge.go`, `CLIENT-07` |
| `stop` is SIGTERM with eight seconds on Linux, and `halt` through the console with ten seconds and then `taskkill /T` on Windows, and writes `stop.txt` on both | the other halves' graces; Windows has no SIGTERM for a console process in another group, and the source route's launcher runs the daemon's interpreter as a child | `adapter_deluge.go`, `adapter_deluge_windows.go`, `CLIENT-07` |
| `stop` deletes `auth*`, `hostlist.conf*`, everything under `config/ssl/`, and any workdir file holding the auth password | `scan-bundle` recognises neither `auth` nor `daemon.pkey`, a PEM private key the first green drive found in its bundle | `deluge.go`, `CLIENT-07` |
| the `package` slugs are `python-wheel` and `python-source` for the pip routes and `nsis-installer` for the Windows package route | the form each route delivers: what the vendor's NSIS installer placed, as qBittorrent's winget route records it | `adapter_deluge.go`, `adapter_deluge_windows.go`, `CLIENT-07` |
| the Windows job gained *Resolve the source tag* and uploads `source/*`, and nothing else changed | the Windows pair is winget beside the tag; the step is the Linux one in pwsh | `capture-client.yml`, `CI-11` |
| aria2's second rootless route is `source`, the `release-1.37.0` tag from `resolve-source`, built into the release route's prefix with the release route's configure line | the `package` route is privileged by construction and failed on run 39577251480 as `ACQ-06` intends. A tag differs from the tarball in resolver and delivery, which is the pair Transmission and aria2-next run. Reversed by removing the route. 2026-10-21. | `adapter_aria2.go`, `CLIENT-05` |
| aria2 declares `release_floor_major=1` | its refs carry eleven 0.x tags the scheme cannot order, and the source route failed closed on them. 1 is the lowest floor that releases them, and the release listing's selection does not move | `adapter_aria2.go`, `check-source-route.sh`, `CLIENT-05` |
| the source route fetches `autopoint` from the distribution's signed index into the workdir, rootless, and records it in `build-deps.txt` | a tag needs `autoreconf -i`, which runs `autopoint`, and the `ubuntu-24.04` image lists no gettext. Root-installing it is what `ACQ-06` forbids, and taking `configure` from the release tarball would merge the routes. Transmission's libcurl headers are the precedent | `adapter_aria2.go`, `CLIENT-05` |
| aria2's source route's `package` slug is `elf-binary-git`, where aria2-next and Transmission call both compiled routes `elf-binary` | the two routes install one form and differ in how it arrived: a vendor-generated `configure` against one generated on the host. Reversed by returning `elf-binary` | `adapter_aria2.go`, `CLIENT-05` |
| `E-COR-20` treats a connector and a target as one implementation when either identifier contains the other, and tests the target's `engine` and every declared connector, the observer included | the planned connector is `aria2c`, which equality and a hyphen-prefix test both accept beside `aria2`. A refused independent pairing costs a capture, and an accepted circular one corrupts the corpus. No catalogue target is refused beside `bit-ids-probe` or `cpython-stdlib`, and all 18 records on `data` still validate. Reversed by narrowing `same_implementation`. 2026-10-21. | `validate.rs`, `CLIENT-05` |
| the record's rule compares connector identifiers; the capture guard also tests the connector's described `runtime`, and refuses with exit 1 rather than 2 | a record carries no connector code, so its identifier has to name its implementation. `runtime=` catches a connector on aria2 under another name while the host still exists, and a circular pairing is a refusal rather than a run that could not happen | `selfcorroboration.go`, `captureclient.go`, `CLIENT-05` |
| `capture-client`'s peer-ID guard decodes each announce's `peer_id` from the transcript JSON, and `capture-run`'s token check is left a substring test | aria2's raw-byte peer ID is percent-encoded on the wire, so the substring test refused a sound capture. `capture-run`'s token is a run ID restricted to `[a-z0-9-]`, which `curl` sends verbatim | `transcriptpeer.go`, `captureclient.go`, `capturerun.go`, `CLIENT-05` |
| ⛔ SUPERSEDED on 2026-10-21: Deluge's winget line passed no `--scope` and named `--architecture x64` | it made run 39616084687's package lane install, and winget no longer installs at all: see the next row. The manifest's lack of a `Scope` is still why the scoped line found no installer | `deluge.go`, `CLIENT-07` |
| the pip routes add `windows-curses; sys_platform == "win32"`, unpinned, and do not take `requirements.txt`'s `twisted[tls]<23` cap on win32 | the requirement and its marker are upstream's, from the tag's `requirements.txt`, and run 39588980862's console import failed without it. Twisted is not on the wire, and run 39616084687's Windows source lane captured on Twisted 26 | `deluge.go`, `CLIENT-07` |
| Deluge's Windows `package` route takes the version and InstallerSha256 from winget's index, fetches the same file over https from the vendor's `ftp.osuosl.org/pub/deluge/windows/` with `install-rootless --fetch`, holds it to that digest and to the vendor's `.sha256`, runs it `/S /D=%ProgramFiles%\Deluge`, and records the https URL as the origin and `https-vendor-mirror-nsis` as the delivery | the index and Scoop both pin `http://download.deluge-torrent.org/...`, and a record may not name an http retrieval. `download.deluge-torrent.org` answered 526 over https through rule 9's route on 2026-10-21; osuosl answered 200 with the bytes `b2e414dd...` the index pins. It stays a `package` route because winget's index is what resolved it, and its delivery is named for what fetched it, not for winget. A URL of any other shape is refused rather than mapped. Reversed when the index pins an https URL | `adapter_deluge_windows.go`, `deluge.go`, `CLIENT-07` |
| a package route may name its own delivery: `package-index.txt` `delivery=`, carried by `install-client` as `package_delivery` and read by `assemble-capture`; absent, it is `host-package-manager` as before | the Deluge route above is resolved by a package manager's index and delivered by something else, and a record calling that `host-package-manager` would misdescribe it. A value that is not a slug is refused at install. Every other route writes none, so nothing else changes | `adapter.go`, `installclient.go`, `assemble-capture.rs`, `CLIENT-07` |
| the Windows source route pins `libtorrent==1.2.19` and runs on the tool cache's newest x64 Python 3.10, else 3.9; Linux keeps `libtorrent==2.0.15` | run 39616084687's Windows pair, patched to the https origin, assembled `divergent`: the vendor installer's libtorrent 1.2.19 and the source route's 2.0.15 differ in the peer wire's reserved bytes and in the tracker user agent. Routes on one host must run one engine, and 1.2.19's Windows wheels stop at cp310. Driven on this host with CPython 3.10.20: installs, passes the import check, answers `libtorrent: 1.2.19.0`, adds a torrent, and shuts down on SIGTERM | `deluge.go`, `adapter_deluge_windows.go`, `CLIENT-07` |
| the Windows source route takes libtorrent 1.2.19's two OpenSSL DLLs, `libcrypto-1_1-x64.dll` and `libssl-1_1-x64.dll`, out of the vendor's installer (fetched over https from ftp.osuosl.org, held to the vendor's `.sha256`, extracted with 7-Zip beside the `.pyd`), records both digests in `engine-deps.txt`, and refuses by name any other import of the `.pyd` nothing on the host carries | run 39623700677: PyPI's 1.2.19 Windows wheel is the `.pyd` alone and imports those two DLLs, so `import libtorrent` failed. The vendor ships PyPI's cp39 `.pyd` of the same release byte for byte beside exactly those two DLLs. CPython 3.10's own OpenSSL 1.1 would need renaming, and its `libssl-1_1.dll` imports `libcrypto-1_1.dll`, which would load two libcrypto copies. The pair then shares the vendor's origin for OpenSSL alone. Reversed when a wheel bundles them | `adapter_deluge_windows.go`, `deluge.go`, `CLIENT-07` |
| both halves run the import check before `--version` and keep each answer in its own log, `import-check.log` and `install-version.log`, printing the tail on a refusal | Deluge swallows a failed `import libtorrent` in `--version`, which then answers with no engine line. Run 39623700677 asked `--version` first, refused on that, and kept neither the answer nor the traceback where the run uploads | `adapter_deluge.go`, `adapter_deluge_windows.go`, `CLIENT-07` |
| qBittorrent Enhanced is its own adapter, `qbittorrent-enhanced`, the catalogue's id, with its own scheme: `release-`, exactly four components, floor 4 | the fork's 273 refs carry 167 three-component tags inherited from upstream, all leading with 3 or less. A floor of 4 releases them, and a three-component upstream tag at the current major blocks rather than being read as Enhanced's. The parent's three-or-four scheme would select such a tag. Reversed by changing `qbeMinComponents` and `qbeFloorMajor`. 2026-10-21 | `qbtenhanced.go`, `check-source-route.sh`, `CLIENT-02` |
| Enhanced's Linux pair is `release` (the vendor's static `qbittorrent-nox` zip) and `source` (the tag built `-DGUI=OFF` against Qt 6.11.2's archives and libtorrent `v1.2.20`), both running the headless build; no Linux package route | every vendor build carries libtorrent 1.2.20, and routes on one host must run one engine. The AppImage and the zip are one listing and one delivery, so the second route is the tag. Only the GUI needs Xvfb. No distribution carries Enhanced, and a package route is privileged. Reversed by building the GUI and taking the AppImage | `adapter_qbittorrent-enhanced.go`, `CLIENT-02` |
| Enhanced's Windows pair is `release` (the vendor's installer extracted by 7-Zip) and `package` (Scoop `extras/qbittorrent-enhanced`, no version), both `nsis-payload` | Scoop is the one index at 5.2.4.10: winget and Chocolatey stop at 5.2.3.10. The pair shares its origin, GitHub's asset, as aria2-next's does; resolver, downloader and check differ. Reversed when winget carries the version | `adapter_qbittorrent-enhanced_windows.go`, `qbtenhanced.go`, `CLIENT-02` |
| the profile writes Enhanced's `AutoBanUnknownPeer`, `AutoBanBTPlayerPeer`, `ShadowBan`, `AddTrackersFromURLEnabled` and `AutoUpdateTrackersEnabled` at their source default, false, and `Connection\ResolvePeerCountries=false`; `start` refuses a profile with any of them on | each changes which peers are dropped or which trackers are announced to. The always-on bad-peer filter cannot be switched off and is recorded instead: it matches neither the observer's peer ID nor its client name, and its country branch needs the GeoIP database the profile keeps unloaded. The peer ID and user agent have no setting | `qbtenhanced.go`, `CLIENT-02` |
| the build is installed as `qbittorrent-enhanced-nox` and found in the prefix only, never on PATH; the Windows half refuses a version that is not four components | a parent `qbittorrent-nox` or `qbittorrent.exe` on the same host is another product, and a lookup that could reach it would measure the parent under the fork's name | `adapter_qbittorrent-enhanced.go`, `adapter_qbittorrent-enhanced_windows.go`, `CLIENT-02` |
| `rqbit` is filed as a library harness: its adapter prints `target_kind=library`, `capture-client` copies it into the attestation's `kind`, and an adapter that prints none stays `client`. The harness is upstream's `rqbit` command at the tag, and none of this project's own is committed | the catalogue lists rqbit as `kind = "engine"` and `engines.md` says engine results are labeled harness observations. `TargetKind::Library` is the record's own word for that, so no field is added. The record has no field for the library version or the harness commit; `features.txt` and `build-features.txt` keep them as evidence and `ENGINE-03` records the gap. 2026-10-21 | `rqbit.go`, `captureclient.go`, `ENGINE-03` |
| rqbit's second route on both platforms is crates.io: route `package`, `cargo install --locked rqbit` with no version and no feature flag, `index=crates-io`. `assemble-capture` files such a lane as `cargo_install` with a `module_version` identity | winget, Scoop's `main` and `extras`, Chocolatey and Ubuntu carry no rqbit. crates.io resolves and cargo delivers, neither of them the release listing. A build of the tag would share the release's repository too. `RouteKind::CargoInstall` already exists, and calling crates.io a host package manager would name the wrong kind. Reversed by a dedicated `cargo` route name | `adapter_rqbit.go`, `adapter_rqbit_windows.go`, `assemble-capture.rs`, `ENGINE-03` |
| the cargo route runs with a private `CARGO_HOME` and `CARGO_TARGET_DIR`, `RUSTUP_TOOLCHAIN=1.98.0` and `CARGO_PROFILE_RELEASE_PANIC=abort`, deletes its build tree afterwards, and refuses a `.crates2.json` that records any feature selection but the default | both vendor builds are `--profile release-github`, `release` with `panic = "abort"`, and a published crate carries no profile, so `panic` is the one setting to match. Neither vendor workflow passes a feature flag. `check-bitcheck` holds the toolchain to `rust-toolchain.toml` | `rqbit.go`, `bitcheck.go`, `ENGINE-03` |
| `start` refuses a build that does not answer `/web/` and `/metrics` with 200 | those routes exist only with the default `webui` and `prometheus` features, so the probe sees two of the release's four defaults in the running build. A build missing either is not the release's feature set | `rqbit.go`, `ENGINE-03` |
| `start` drops every `RQBIT_*`, `RUST_LOG` and `CORS_ALLOW_REGEXP` from the build's environment and passes no `--ipv4-only` | each of the build's switches also reads an `RQBIT_*` variable. `--ipv4-only` does not stop the UDP tracker client binding `[::]:0`, so it does not make the build run on an IPv6-less host and would change the listener on one with IPv6 | `rqbit.go`, `ENGINE-03` |
| the Windows lane of `capture-client.yml` reads `compiles` as the Linux one does: 1500, 1980 and 2100 seconds and a 37-minute step for a compiling route, and the job bound is 55 minutes | rqbit's Windows `package` route is `cargo install`, and the fixed 600-second bound was a download's. Every other Windows route declares no `compiles` and keeps its bounds | `capture-client.yml`, `ENGINE-03` |
| rqbit's Windows `stop` kills; Linux's sends SIGTERM | the build compiles its signal handler out on Windows and its API has no shutdown. It constructs no `stopped` announce on either platform, so nothing on the wire is lost | `adapter_rqbit_windows.go`, `ENGINE-03` |
| BiglyBT's scheme is `v`, exactly four components, floor 2 | the 34 tags of 2026-10-21 are four components except `v1.8.00`, three under major 1, which the floor releases rather than letting it fail the source route closed. Every release object is one of those tags. Reversed by changing `bbDescribeScheme` | `biglybt.go`, `check-source-route.sh`, `CLIENT-09` |
| BiglyBT's Linux pair is `release` (the vendor's `GitHub_BiglyBT_unix.tar.gz`, extracted into the prefix) and `source` (the `v4.1.0.0` tag compiled by the host's JDK with upstream's Maven recipe for JDK 9+, run as `javac`), both on the host's JVM; no Linux package route | no distribution carries BiglyBT, Flathub and Snap are privileged, and the vendor's install4j `.sh` is the tarball's listing and delivery again. Maven itself would fetch SWT from a repository off GitHub; the tag carries its SWT jars. One JVM on both lanes keeps the pair one runtime. Reversed when an index carries 4.1.0.0 rootless | `adapter_biglybt.go`, `CLIENT-09` |
| BiglyBT's Windows pair is `release` (the vendor's installer with its bundled JRE, `-q -dir %ProgramFiles%\BiglyBT`) and `package` (winget `BiglySoftware.BiglyBT`, the NoJRE installer with its Temurin 25 JRE dependency), each run on its own JVM | winget is the one index at 4.1.0.0: Scoop's `nonportable` stops at 4.0.0.0 and Chocolatey at 3.5.0.0. The two are the vendor's two Windows packagings, which the entry asks to represent rather than flatten; `version` prints each JVM's properties | `adapter_biglybt_windows.go`, `CLIENT-09` |
| BiglyBT is driven by its console UI with no torrent at launch; the torrent is handed to the running instance by a second launch, and Windows stops it with the vendor's `--closedown` | a torrent on the first command line ends the JVM with *Start fails: Core not running*. The second launch is the vendor's own path for a torrent opened while the client runs. No remote UI is enabled, so no credential exists | `adapter_biglybt.go`, `adapter_biglybt_windows.go`, `CLIENT-09` |
| BiglyBT's `version` runs a class file written by the adapter, byte by byte, on the route's JVM against the installed jar | the build has no `--version`, a JRE has no compiler, and `BIGLYBT_VERSION` is a compile-time constant, so it is read by reflection. Nothing is shipped; the class is made per call into a temporary directory | `biglybt.go`, `CLIENT-09` |
| every bundled BiglyBT plugin is unloaded on both routes (uTP among them), and two vendor calls with no switch, the version check and the featured-template refresh, are left to containment | the source route carries no plugin, so loading them on the release route alone makes two programs: mlDHT alone flipped the DHT bit and made the first pair `divergent`. The vendor's `azupdater` is loaded by the core whatever its setting and is recorded. Reversed by building the plugins from their own repositories | `biglybt.go`, `CLIENT-09` |
| the JVM's option variables are dropped from every JVM BiglyBT's adapter starts; `stop` removes the whole config directory and keeps its logs | a host's `JAVA_TOOL_OPTIONS` changes the client's network behaviour; this host's sets a proxy. The config holds private keys and an API key the bundle scan does not name | `biglybt.go`, `adapter_biglybt.go`, `CLIENT-09` |
| anacrolix/torrent's harness is this tree's, `tools/harness/anacrolix-torrent/`, a module of its own whose `go.mod` and `go.sum` pin v1.61.0, every dependency and `toolchain go1.27.3`; it runs `NewDefaultClientConfig()` with every flag at the library's default unless the adapter passes it | a module version installs nothing and upstream publishes no binary. Its `cmd/torrent` is a downloader: it ends when its torrent completes unless told to seed, stops on a signal Windows cannot send, and reports none of its config. `client-matrix.md` already names a minimal harness. `bit-check` has no dependencies, so the harness cannot share its module. `anacrolixPinRefusal` holds the files to the adapter's constants at every install and in `check-bitcheck`; a newer stable is re-pinned in one change, never upgraded by a route. 2026-10-22 | `tools/harness/anacrolix-torrent/`, `anacrolix.go`, `ENGINE-02` |
| the Go engine's routes are `package`, proxy.golang.org with no `go.sum` given, checked by sum.golang.org and required to write the committed `go.sum`, and `source`, `GOPROXY=direct` from the tag `resolve-source` selected, every module cloned from its own origin and checked against the committed `go.sum`. `index=proxy.golang.org` files as `go_module` | the catalogue names `go-module` and `source-build`. The two share no resolver and no delivery for any module; they share the harness source and the toolchain, which is the point of a reproducible build. `vcs_repositories` tells them apart where the go command's own records cannot. Reversed by a dedicated `go-module` route name | `anacrolix.go`, `assemble-capture.rs`, `ENGINE-02` |
| both routes build with `GOTOOLCHAIN=go1.27.3`, `CGO_ENABLED=0`, `GOENV=off`, `-trimpath` and `-buildvcs=false`, every host `GO*` dropped | a runner's Go is its image's, so the toolchain is named. Cgo would link libutp on a host with a C compiler and anacrolix/utp elsewhere, two uTP implementations under one profile. A VCS stamp would put this tree's commit into the extended handshake's `v`. Measured: both routes then build one executable, byte for byte | `anacrolix.go`, `ENGINE-02` |
| the harness runs IPv6 off on every host, and uTP on | the library's default `tcp6` listen fails on a kernel without IPv6 and the client never starts; off everywhere keeps the profile one profile, and no handshake field moves. uTP is a default and stays | `anacrolix.go`, `ENGINE-02` |
| the harness closes on a stop file it polls, on both platforms, and is killed after 10 seconds | Windows has no SIGTERM for a console-less process, and one mechanism keeps the halves one harness. `Client.Close` is the library's own exit; it sent no `stopped` announce in any session | `tools/harness/anacrolix-torrent/main.go`, `adapter_anacrolix-torrent*.go`, `ENGINE-02` |
| the second connector and `normalize.rs` read a connection's first handshake across the segments that carry it | anacrolix/torrent writes its handshake in two writes and the lab recorded it in two segments, split at byte 20 on one lane and 28 on the other; both readers refused a capture the observer read. The join is limited to a first segment shorter than a handshake and to its own connection, and a handshake never completed is still refused | `cpython-stdlib.py`, `normalize.rs`, `connector.go`, `ENGINE-02` |
| KTorrent gets no adapter. `CLIENT-11` stays open with its blocker measured, and the nixpkgs route is recorded but not built | KDE ships only source and no Windows build. Every binary route at 26.08.2 either needs root (apt, Snap, a system Flatpak) or trails it (Flathub 26.04.3, the AppImage 26.08.1, noble 23.08.5). A rootless source build means about 30 KF6 frameworks and their C libraries on every lane. nixpkgs' 26.08.2 runs rootless only under a user-space Nix, and both its routes would be nixpkgs' build. Admitting that family is the operator's decision. Reversed by choosing one of the three options the entry lists. 2026-10-22 | `clients.md`, `client-matrix.md`, `CLIENT-11` |
| libtorrent's target is the 2.0 series, at 2.0.15, read by the library's own scheme, `v` and three components, with `release_series=2.0`. ⚠ Until 2026-10-22 it read `v2.0.` with one component, which the published view refused | RC_2_0 and RC_2_1 both released on 2026-09-25. PyPI carries 2.0.15 and stops at 2.1.1 in the newer series, so only 2.0 has PyPI and the tags at one version. 2.1, through the `v2.1.2` release's own wheels, and 1.2 each get their own pair later. Reversed by a dated pin per series in the catalogue. 2026-10-22 | `libtorrent.go`, `ENGINE-01` |
| libtorrent's harness is Python, this tree's `tools/harness/libtorrent/harness.py`, `lt.session` on the library's default settings with DHT, LSD, UPnP, NAT-PMP, `ut_pex` and IPv6 off by flag and the alert mask widened | the PyPI wheel delivers the Python binding and nothing else that runs, so a binding program is the one harness both routes can run; a C++ harness would link a third build. The alert mask changes nothing on the wire and lets the harness log what the library did | `tools/harness/libtorrent/harness.py`, `ENGINE-01` |
| libtorrent's routes are PyPI's wheel for `libtorrent>=2.0,<2.1` and the `v2.0.15` tag built by its own `setup.py` with no option, against Boost 1.83.0 from archives.boost.io; the PyPI lane files as `linux_package_manager`, as Deluge's | that is the vendor's own cibuildwheel path, so the source route builds the wheel's feature set rather than one of this project's choosing. Every install holds the b2 line, the binding's compiled-in symbols and the sha256 of all 258 default settings, measured equal on both routes. The OpenSSL release differs, both static, and is recorded | `libtorrent.go`, `adapter_libtorrent.go`, `ENGINE-01` |
| a target that keeps several stable series pins one with `release_series`, declared on every adapter, `-` for none. A stable tag whose leading components differ readably from the series is the new verdict `other_series`, not `foreign_tag` and not `superseded`; a tag that agrees as far as it goes or is unreadable before a difference still blocks. The scheme orders only versions inside its series, so the latest view never compares a record across series, and the series travels as a fifth field of the scheme line, `libtorrent:v:3:3:2.0`. It is `bit-ids/resolution/3`; assembly still reads the first two generations | `v2.1.2` is libtorrent's own stable release, so calling it foreign or older would be false. Each series stays current, so "latest" across them is not one build line's answer, and moving the pin is a scheme change the assembler refuses against the published one until it is decided. Over every adapter's live refs and listings on 2026-10-22 only libtorrent's verdicts moved, and the 36 published records rebuild a byte-identical view. Reversed by deleting the series field and verdict | `resolution.rs`, `resolve.go`, `dispatch.go`, `ENGINE-01` |
| capture-client run 39783974176 is not re-assembled; libtorrent needs a new dispatch | its `resolution.json` carries the old scheme, `v2.0.` with one component, and `selected_version=15`, and the assembler takes the scheme from there by design. Re-assembled as it is with this tree, it is refused with `E-VIW-02` again; editing the artifacts' scheme would be a scheme nobody resolved under | `ENGINE-01` |
| libtorrent has no Windows half; the PyPI wheel was measured for it and the source route was not built | a Windows source build needs MSVC, Boost's b2 and a static OpenSSL inside 1500 seconds and nothing here can measure it. The cp312 win_amd64 wheel links OpenSSL 3.6.1 statically and imports no OpenSSL DLL. Reversed by the half `ENGINE-01` names | `ENGINE-01` |
