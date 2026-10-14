# Scripts

Project scripts resolve the repository from their own location and may be run
from any working directory.

## ⭐ What to run before a push, by what changed

Each row is what the CI job that would refuse the change runs, so a green local
run is the same verdict CI will give. Durations were measured on 2026-10-11 on a
four-processor session host, cold, and on run 179 for CI.

| what changed | run locally | local time | the CI job that runs it |
| --- | --- | --- | --- |
| anything | `sh scripts/doctor/provision.sh` once per fresh host, then `sh scripts/common/check-gate.sh` | about 3 minutes | *Linux gate*, *Repository gate*, about 3 minutes of its 5.8 |
| Rust | `cargo fmt --all -- --check`, `cargo clippy --workspace --locked --all-targets -- -D warnings`, `cargo test --workspace --locked --all-targets` | under a minute warm | *Linux gate* and *Windows gate* |
| `tools/check/` | `gofmt -l tools/check`, `go vet ./...` from `tools/check`, `sh scripts/common/check-bitcheck.sh --harnesses` | about a minute | *Linux gate*: *Go formatting and vet*, *Checking harnesses* |
| a shell script | `shellcheck` and `shfmt -d -i 2 -ci` over it, then the gate | seconds, then 3 minutes | *Shell syntax and style* |
| `.github/workflows/` | `sh scripts/ci/check-workflow.sh`, unsharded, after the gate | about 30 minutes | *Workflow acceptance*, four shards of 12 to 16 minutes |

⛔ **Do not edit the tree or `git add` while the gate or `check-workflow` runs.**
Both copy and compare the working tree, and an edit mid-run fails a row with a
message naming a check rather than the edit.

⚠ **The one skip the gate prints on a session host is expected:**
`check-remote-items` needs an authenticated `gh`, and both CI lanes run it.
Any other skip is a missing tool, and `--strict` makes it a failure on CI.

## The directories

- [`doctor/`](doctor/README.md) reports host capabilities without changing the
  host. ⛔ One file there is the exception and says so:
  [`doctor/provision.sh`](doctor/provision.sh) installs the three tools the gate
  needs, verifying each download against a pinned digest first. The doctor does
  not call it.
- [`acquisition/`](acquisition/fetch-releases.sh) retrieves a release listing
  and keeps the exact bytes. It does not parse, sort or decide.
- `bit-check check-cache` drives the artifact cache through a source that moved
  and asks the register's one reader - the function `check-licences` itself calls
  - what the register allows, so the tie between the two is a call rather than a
  second reading of the register. ⭐ It was `acquisition/check-cache.sh` and a
  PowerShell twin until 2026-09-29, the first HARNESS to leave the twin layer.
- [`acquisition/resolve-release.sh`](acquisition/resolve-release.sh) decides
  which artifact a `release` route will fetch, while the network is in reach. ⛔ It
  carries no per-target knowledge: the repository, the version scheme and the
  asset pattern come out of the adapter's own `describe`, so a workflow driving
  it needs no case over which product it is running.
- [`acquisition/check-release-route.sh`](acquisition/check-release-route.sh)
  proves each shipped adapter's pattern selects exactly one artifact, over
  [recorded responses](acquisition/listings/README.md) rather than over three
  vendors. ⛔ The case it exists for is the ambiguous match: qBittorrent's
  release publishes two Linux `AppImage` files differing only by an `_lt20`, so a
  selector that took the first match would install whichever the vendor listed
  first.
- [`acquisition/install-rootless.sh`](acquisition/install-rootless.sh) is the one
  installer every `release` route goes through: it fetches with a bound, settles
  the artifact's digest before anything is made executable, writes into a prefix
  the current user already owns, and prints the path it placed. ⛔ It never
  escalates - a prefix this user cannot write is a refusal naming it - and
  `--prefix` is the ONE derivation of that path, so no adapter composes a second
  one.
- [`acquisition/check-rootless.sh`](acquisition/check-rootless.sh) is `ACQ-06`'s
  acceptance: it drives every adapter's release route as an unprivileged user
  into a scratch prefix, asks each installed build its version, and repeats the
  whole run with `sudo` absent from `PATH` entirely, refusing any difference.
  ⚠ The vendor is a `file://` URL, the way `resolve-release --listing` is a
  recorded response, so a gate row does not depend on four vendors being up.
- [`corpus/check-store.sh`](corpus/check-store.sh) plants, in a disposable tree,
  every defect the append-only store exists to refuse, and reads each exit code
  from the process that produced it.
- `bit-check check-corpus` does the same for the store-level invariants,
  against a store `build-store` wrote. ⭐ It was `corpus/check-corpus.sh` until
  2026-10-14, compared case for case against it before that half was deleted,
  so the Windows lane runs it too. `CI-07`.
- `bit-check check-indexes` proves the three things a derived file owes:
  byte-identical clean builds, rows that resolve back to the records they came
  from, and a corrected record that has left the views while its bytes are
  still filed where they were. ⭐ It was `corpus/check-indexes.sh` until
  2026-10-14, compared case for case against it before that half was deleted.
  `CI-07`.
- [`publishing/check-release.sh`](publishing/check-release.sh) assembles a
  release twice, compares the bytes, and hands the checksum file to `sha256sum
  -c`, which is a reader this project did not write.
- [`publishing/check-formats.sh`](publishing/check-formats.sh) renders every
  published format over a store that carries a correction, compares two renders
  as bytes, checks that the corrected record is not published as a record in any
  of them, and hands the assembled checksum file to `sha256sum -c`. ⚠ It does
  not decode the CBOR: this project has no CBOR reader, and using its own
  encoder to read it back would be checking the writer against itself.
- [`publishing/publish-data.sh`](publishing/publish-data.sh) appends an
  assembled bundle to the data branch: a bundle with no `MANIFEST.json`, no
  `SHA256SUMS` or a digest that does not match is refused before the push, the
  append rule is checked before the push, nothing re-enables force, and the branch is read back and verified
  before the run says it happened.
- [`publishing/assemble-bundle.sh`](publishing/assemble-bundle.sh) composes the
  publication tree over a store that is a valid corpus: the indexes, every
  rendering, the licence, `MANIFEST.json` and `SHA256SUMS`. It takes each
  target's version scheme from the resolutions it is handed, and removes a
  half-built bundle. [`publishing/check-bundle.sh`](publishing/check-bundle.sh)
  proves it, ending with the publisher appending its bundle to a scratch bare
  repository.
- [`publishing/check-publish.sh`](publishing/check-publish.sh) drives that
  publisher against a bare repository it creates in a scratch directory, so the
  push path runs for real with no network and no credential.
- [`publishing/check-access.sh`](publishing/check-access.sh) publishes twice to
  such a repository, fetches each publication back over git, and checks every
  documented access path against what came back: that it resolves, that
  `sha256sum` agrees with its stated digest, and that every path the contract
  calls immutable is byte-identical across the two. ⚠ At least one current path
  must have moved, or the immutability comparison passes over two identical
  publications.
- [`common/check-examples.sh`](common/check-examples.sh) extracts every shell
  example out of [`../docs/consuming.md`](../docs/consuming.md) and runs it
  against a publication it assembles. ⛔ The page is what runs, never a copy
  beside it, and its own selection rule is checked: a run that took every fenced
  block is a run whose language rule has stopped applying.
- `bit-check check-catalogue` drives the consumer library against a real
  publication: it opens, it answers `1.2.10` over `1.2.3`, and every defect a
  consumer could be handed is refused. ⭐ It also sweeps the crate for sockets
  and transports, because "no network access" is a property of what the code
  cannot do rather than of what one run did, and it checks its own needle list
  against the crate that owns the sockets. ⚠ It was
  `publishing/check-catalogue.sh` and a PowerShell twin until 2026-09-29.
- [`capture/capture-run.sh`](capture/capture-run.sh) and
  [`capture/capture-run.ps1`](capture/capture-run.ps1) run a capture on a host
  that has already been claimed and already had its route off itself deleted.
  ⛔ They build nothing: the observer is a path to an already built binary,
  because a capture that discovered a missing dependency under containment would
  have to restore egress to fix it. ⭐ The driver is `curl` and the verifier is
  `sha256sum -c` or `Get-FileHash`, so neither the bytes on the wire nor the
  digests are this project checking itself, and the announce carries a token the
  driver knows it sent which the transcript must hold.
- [`capture/check-capture.sh`](capture/check-capture.sh) constructs the host
  state each of those guards exists to catch, and drives a stub observer for the
  refusals no host state can provoke. ⭐ It is also where the two capture
  runners are compared, because `check-twins.sh` pairs the `common/` checks and
  these take a host and a running process rather than a tree.
- [`capture/capture-client.sh`](capture/capture-client.sh) is the same shape for
  a capture of an **installed build**. ⛔ The difference is the driver: an
  adapter answers what the installed executable's version is and starts that
  build on the torrent the observer generated, which is what turns an
  attestation about a containment into one about a product. ⭐ Two guards say a
  build was measured rather than assumed: an announce must carry a peer ID this
  observer did not generate, and the raw transcript on disk must hold those
  exact bytes.
- [`capture/scan-bundle.sh`](capture/scan-bundle.sh) is rule 12 over a
  directory about to be uploaded: a file whose name says it holds a credential,
  and a document carrying a secret-shaped value, both always asked. ⛔ It is the
  one home for both rules, and three callers ask it: `capture-client.sh` over its
  bundle, `install-client.sh` over its workdir's top level with `--top-level`,
  and `capture-client.yml`'s *Scan the evidence of a refused capture*, without
  which a refused capture's bundle would be uploaded unscanned. ⚠ A directory
  that is not there is exit 2, never clean.
- [`capture/adapters/`](capture/adapters/) holds one file per target and is the
  only place that knows how a particular product is installed, asked its
  version, started and stopped. Its [README](capture/adapters/README.md) carries
  the five-subcommand contract. ⛔ `kind` is a declaration the attestation
  copies rather than assumes, so a run driven by a stand-in writes
  `stock_client=false`.
- [`capture/connectors/`](capture/connectors/) holds the SECOND connector, which
  is what absolute 2 asks for and what nothing in this project had. ⛔ Its whole
  value is being a reading this project did not write, so it is `python3` rather
  than shell or Rust and its own header carries the three rejected alternatives.
  It reads the bundle's transcripts, decodes the announce with `urllib.parse`
  and the headers with `http.client`, and writes `connector/<id>.txt` - the
  contract `assemble-capture` already refuses a capture for not carrying.
  ⛔ A line carries every sample of its field, comma-separated in document
  order, and the assembler compares them position by position; two announces
  that differ are both reported rather than refused.
  ⚠ It names itself with `--describe`; a caller never composes the identifier.
- [`capture/check-connector.sh`](capture/check-connector.sh) proves it, and the
  load-bearing case is not a refusal. ⛔ A connector that echoed the observer
  would agree on every field of every capture forever, so what this asserts is
  that MOVING A BYTE IN THE TRANSCRIPT MOVES THE REPORT. ⭐ Nine defects were
  planted in the connector and the last one to be caught was found by planting
  rather than by reading: the uppercase-hex case had also changed the length, so
  the odd-length branch refused first and a connector planted to accept
  uppercase passed it.
- [`acquisition/install-client.sh`](acquisition/install-client.sh) is the one
  step that runs with the network still up. ⛔ It asserts the claim and
  deliberately not egress, because being able to reach a package index is the
  precondition of an install rather than a violation, and a guard there would
  refuse every host the step is meant to run on. ⚠ One route per invocation, so
  a caller that ran one and skipped the other has run one. ⛔ It asks the adapter
  for a version **before** the route runs as well as after, and records
  `preexisting_version` and `acquired`: a route that installs nothing exits 0,
  and two of those agree on one binary's version while declaring two independent
  routes.
- [`acquisition/install-step.sh`](acquisition/install-step.sh) is what
  *Install the client* runs, in a file so the step's own process can be bounded
  from outside it. ⛔ Every bound tried before it was measured failing: `timeout`
  inside `install-client` ends its own child and cannot end a shell blocked
  around it, the runner's `timeout-minutes` did not end the step on run 6, and a
  watchdog loop in the step's own shell did not fire on run 9 - its deadline
  passed by seven minutes. ⭐ It runs the install in the background, appends a
  process table with `stat` and `etimes` every few seconds, and sends the route's
  output to a file so nothing it spawns inherits the step's own pipe.
- [`capture/check-capture-client.sh`](capture/check-capture-client.sh) proves
  both of those. ⭐ Its stub adapter reads the announce URL out of the
  `.torrent` rather than being handed it, which is what makes it stand for a
  client rather than for a caller that already knew the address, and it runs
  each shipped adapter's `describe` so a typo in one is found by a gate rather
  than by a runner.
- [`acquisition/resolve-source.sh`](acquisition/resolve-source.sh) decides which
  tag the `source` route builds, from `git ls-remote --tags --refs` rather than
  from a release listing. ⛔ It exists because the two lanes of a dispatch used to
  share one resolution, which `E-ACQ-07` calls one route.
  [`acquisition/check-source-route.sh`](acquisition/check-source-route.sh) proves
  it: the newest tag is selected with `1.2.10` beating `1.2.9`, and a peeled ref,
  a branch ref, an abbreviated object name, a line with no tab and an empty
  listing are each refused. ⚠ Its listings are generated rather than recorded,
  because an object name is forty hex digits and `check-no-secrets --public`
  refuses that shape.
- [`capture/check-assemble.sh`](capture/check-assemble.sh) proves
  [`assemble-capture`](../crates/bit-ids-probe/examples/assemble-capture.rs),
  which is what turns a dispatched capture's artifacts into records. ⭐ Its
  control is the half that matters: two routes, two resolvers, two connectors and
  one identity on the wire assemble into two records and reach
  `build_equivalent`. ⛔ Its refusals are what `capture-client` run 14 met - one
  listing behind two lanes, a source route naming no commit, and a capture
  declaring one connector - and each is a fact about the capture path rather than
  about the assembler. ⚠ Its lanes are written by the harness rather than
  downloaded: a case that fetched a run's artifacts would go red the day that run
  expired.
- [`capture/check-transmission.sh`](capture/check-transmission.sh) drives
  Transmission's release and source routes through the real adapter, with
  `apt-get`, `cmake` and `git` stubbed and a real `.deb` and `.tar.xz`. It
  plants headers for a runtime the host lacks, a package without `curl.h`, two
  packages, a build missing an executable, and an archive with two trees, and
  checks that nothing reaches the prefix after any refusal.
- [`capture/assemble-dispatch.sh`](capture/assemble-dispatch.sh) turns one
  `capture-client` dispatch's downloaded artifacts into its records, their run
  manifests and the `bundle` the publisher downloads. It pairs the lanes by
  artifact name and refuses an adapter with one route, a capture with no install
  beside it, two attempts, and a lane whose resolver wrote no `resolution.json`.
  [`assemble-bundle.yml`](../.github/workflows/assemble-bundle.yml) runs it, and
  `check-assemble` drives it.
- [`ci/check-staleness.sh`](ci/check-staleness.sh) drives the staleness monitor
  over real stores and real resolutions: a new stable release opens one request,
  a preview and a release already measured open none, and a second run over a
  tracker holding the first run's request opens nothing. ⭐ It re-derives the
  request identifier with `python3`'s SHA-256, which is an implementation this
  project did not write.
- [`common/check-shell.sh`](common/check-shell.sh) runs `shellcheck` and `shfmt`
  over every tracked shell script, as two rows rather than one, because a red row
  has to say which tool refused. ⛔ The gate ran neither until 2026-09-10, which
  is what let CI go red on a tree whose local gate was green minutes earlier -
  and the CI workflow's own comment already claimed the gate ran them.
  ⚠ A third row counts what was swept: two clean tools over a `find` that stopped
  matching would report the same answer over a repository full of shell.
- [`ci/check-defaults.sh`](ci/check-defaults.sh) runs several checks under a
  perturbed environment and compares their machine-readable answers, so a value a
  script takes from its host without saying so is named by the variable that
  produced it. ⛔ It is a CI step rather than a gate row, for the reason
  `check-workflow` is: a row worth 28 seconds locally is worth nearly five
  minutes of the CI wall clock, because `check-workflow` runs the whole gate
  about ten times. ⛔ Two controls come first, because "no difference" and "no
  experiment" look identical: a probe that reads the variable must answer
  differently, or the run exits 2. ⚠ One row is blind on a host that has built
  before, and it says which state it is in rather than reporting a pass that
  sounds stronger than it is.
- [`ci/check-workflow.sh`](ci/check-workflow.sh) copies the working tree into a
  scratch repository, plants a defect of each class the pipeline exists to
  catch, and runs the offending workflow step against it. Every command it runs
  is read out of `.github/workflows/ci.yml` by job and step name, so a harness
  that has drifted from CI reports a missing step rather than a pass.
  ⭐ **It shards.** `--shard i/N` selects units by `index mod N` and `--units`
  lists the names and runs nothing. ⛔ **The controls are NOT sharded**: a shard
  carrying only plants goes green over a tree where every step is broken, so
  every shard runs all three and that is a floor on what one costs.
- [`ci/check-shards.sh`](ci/check-shards.sh) asserts that the N shards together
  run every unit exactly once, for every N from 1 to 6, that each names every
  control, that no variable is assigned in one unit and read in another, and that
  a malformed selector is refused rather than clamped. ⚠ It costs seconds, because
  it asks `--units` rather than running anything. ⛔ It found three real defects
  the moment it existed, all of them in the sharding: a `--units` mode that
  ignored `--shard`, and two variables that crossed a unit boundary.
- [`ci/workflow-step.sh`](ci/workflow-step.sh) is the one reader that lifts a
  step out of a workflow: its job's ordered step names, its `run:` body, or the
  shell it declares. ⛔ Two harnesses execute step bodies now, and a copy of the
  parser in each would be a second answer to what a step runs. ⚠ It keeps three
  statuses apart that a caller reading a string cannot: a step the workflow does
  not have, a step that runs an action rather than a command, and an empty body.
- [`ci/check-step-bodies.sh`](ci/check-step-bodies.sh) runs the capture
  workflows' own step bodies the way a runner does, which nothing here did
  before: under GitHub's default `bash -e` or its `pwsh -command ". '<file>'"`
  wrapper, with the output on a **pipe**. ⛔ A step is over when its command has
  exited *and* that pipe has reached end of file, so a process the body leaves
  behind holding the step's stdout keeps the runner waiting on a command that
  finished - and no exit code says so. ⭐ It records both facts per body and
  keeps them apart, which is what lets it accept `capture.yml`'s *Restore the
  route* block as it stands, refuse it in the form that failed capture run 1, and
  measure `capture-client.yml`'s install block ending over a product that leaks
  one process - with a planted `3>&1` bringing the hang straight back, which is
  what says the redirection is doing the work rather than the case being easy.
- [`ci/flight-recorder.sh`](ci/flight-recorder.sh) posts a runner's state to a
  check run on the commit every few seconds while a capture job runs, so the
  evidence leaves the host as it is taken. ⭐ Posts that stop mean the host lost
  contact at that moment. ⛔ The job's last step, *Stop the flight recorder*,
  closes the check run in the request that posts its last snapshot, and the
  recorder reads the closure back, so a run still open after its job is a host
  that never reached that step and one that closed shows the state its job ended
  in. `check-step-bodies` drives both steps against a stub `curl`.
- [`ci/install-barrier.sh`](ci/install-barrier.sh) holds each lane of a
  `capture-client` dispatch until every route of its adapter has installed,
  across runners that share nothing but the commit: each posts a check run named
  for its run attempt, adapter and route, and waits for the others. ⚠ It never
  fails a lane; it writes `passed`, `failed` or `timeout` into
  `install-barrier.txt`, which the install artifact uploads. `check-step-bodies`
  drives both steps against a stub `curl`. `CI-09`.
- [`ci/report-holders.sh`](ci/report-holders.sh) names the processes still
  holding an open descriptor on a file, and prints the process table beside them.
  ⚠ It reports and never refuses: a step that went red because its diagnostic
  found nothing would replace a missing answer with a wrong one. ⛔ It has to run
  as the user the work ran as - an unprivileged reader of `/proc` reports nobody
  holding a file three root processes are holding.
- [`corpus/store-lib.sh`](corpus/store-lib.sh) is sourced by **every** mutation
  harness except `acquisition/check-runner.sh`, and by
  `publishing/publish-data.sh`, `publishing/assemble-bundle.sh` and
  `capture/assemble-dispatch.sh`, which are not harnesses. It is never run.
  ⚠ The two exceptions are named rather than counted, and the count that used to
  stand here is gone on purpose: it said "twelve" and went stale the next time a
  harness was added, which is the second time a number in this file has done
  that. A rule with its exceptions named describes the set on any tree; a number
  describes the tree it was written on. ⚠ It sits under `corpus/` because that is
  where the first harness to need it was, and a publishing check sources it
  across directories rather than growing a second copy. It holds what a mutation
  harness needs: build an example, make a scratch tree, digest a directory,
  verify a plant landed, count a row.
- ⛔ **`corpus/store-lib.ps1` is deleted, 2026-09-29, because nothing called it.**
  It was this library's twin and `CI-07`'s first step, and its only two callers
  were the harness twins `CI-10` put in the Go binary, where
  [`../tools/check/harness.go`](../tools/check/harness.go) carries what they
  needed. A portable class-A row now closes by joining that binary rather than
  by gaining a PowerShell half, and `check-gate.ps1`'s declared rows say so.
- [`common/check-gate.sh`](common/check-gate.sh) and
  [`common/check-gate.ps1`](common/check-gate.ps1) run the local gate. ⭐ The
  `sh` half runs its checks **concurrently** and reads their verdicts in list
  order, because every check here is hermetic and the wall clock is not: it went
  from 198 seconds to 73 on the host that measured it, and `check-workflow` runs
  the whole gate nine times, so the saving multiplies. ⛔ Each exit code is still
  read from the process that produced it - `wait` on that child and nothing
  else - and each row is still assembled at its own index, so two runs over one
  tree produce one report.
- `bit-check check-project` validates bit-ids
  structure, catalogue coverage, todo counts, action pins, the shell-first
  implementation rule, that a `.ps1` carrying non-ASCII starts with a UTF-8 BOM,
  that a `.ps1` which stops on errors also says what a native command's exit code
  means, and that none reports through `Write-Error`, whose rendering wraps by
  host width. ⚠ All three of the last were conventions until something counted:
  eleven files had the BOM and four did not, sixteen relied on a PowerShell
  default that changed in 7.5, and ten wrote to stderr directly while five did
  not. The last two were found by CI rather than by a reading.
  ⭐ **It was `common/check-project.sh` and a hand-written twin until 2026-09-17**,
  the last `common/` pair and the biggest at 997 lines. Both halves are deleted,
  and their refusals are one Go check on both lanes. `TODO/ci.md` under
  `CI-10` says what the pre-deletion comparison found.
  ⛔ **The last two also read the `pwsh` blocks of every workflow**, which they
  did not until `CI-06`'s first dispatch found `capture.yml` breaking both, one
  door away from the rules that forbid them. A third rule lives only there: a
  block that reads `$LASTEXITCODE` ends in an explicit `exit`, because GitHub's
  wrapper reads whatever the block left behind as the step's verdict, so an
  inverted guard fails the step by refusing exactly as it was designed to.
- [`common/check-gate-rows.sh`](common/check-gate-rows.sh) compares the two gate
  runners' row lists. ⛔ Nothing did: the `sh` runner derives most of its provers
  from a list and the PowerShell one declares each by hand, so a prover added to
  the first and forgotten in the second is absent from that lane, which stays
  green because it never hears of it - and `--strict` cannot help, since a row
  that was never named cannot be counted as a skip. ⭐ `--rows` and `-Rows` print
  the names each runner's own queue would have used and run nothing, so the
  comparison costs two process starts and cannot re-enter the gate. ⚠ It compares
  SETS: the two halves schedule differently on purpose, so the order a row
  appears in is not a fact about what either lane runs.
- [`common/check-public-row.sh`](common/check-public-row.sh) asks whether each
  gate runner really passes `--public` on the row that exists to ask the public
  question. ⛔ A row can ask the WRONG question and stay green: the default
  question also holds on a clean tree, so a runner that stopped passing the flag
  would keep a row verifying nothing about emails, absolute home paths or long
  hex. ⚠ `check-gate-rows` compares row NAMES and both runners would go on
  naming it; `check-bitcheck` proves the BINARY honours the flag and says nothing
  about what the runner passes. ⭐ The flags are READ OUT OF the two runners and
  then run, and the binary's own `"public_rules"` is the answer - so a harness
  spelling the flag itself could not have caught the defect being guarded
  against. Its control is the default invocation, which must answer
  `public_rules:false` or the field is not one that moves.
- `tools/check/licences.go` checks the register in `catalogue/licences.toml`
  against the catalogue and the lockfile in both directions, refuses a row with no
  disposition, and refuses an installer-shaped file in the tree. ⭐ Ported from a
  pair on 2026-09-10, `CI-10`; `--permitted` lists the redistributable target ids.
  ⚠ `check-cache` used to build the binary and ask that mode; in the binary it
  calls the same reader directly, so the mode has no caller in this tree and is
  kept as the command a person runs to ask what the register permits.
- [`common/check-bitcheck.sh`](common/check-bitcheck.sh) plants a defect per rule
  against the Go checking binary in [`../tools/check/`](../tools/check/) and
  refuses one it does not catch. ⭐ **`--compare` additionally runs the `.sh` and
  `.ps1` halves and refuses any difference in exit code or in the `--json`
  output**, stderr included, which is `CI-10`'s bound: a ported check refuses
  exactly what its shell half refused, over the same plants, and is compared
  case for case BEFORE either half is deleted. ⛔ That mode is deliberately not
  in the gate - one PowerShell half alone is 67.9 seconds, so running it per
  case would cost more than the layer this removes - and the default mode is the
  permanent row, which goes on being a row after every half has gone.
  ⚠ Its own first mutation pass reproduced `check-twins`' documented blind spot:
  adding DEL to the Go control class left every case green, because nothing in
  the tree and nothing planted carried that byte. The repair is a fixture.
  ⛔ **`--harnesses` adds the cases for a ported HARNESS**, and the default mode
  leaves them out and says so on stderr. A harness's cases build its Rust subject
  out of the scratch tree, cold, which measured this row at 39 seconds against
  23 and the whole gate at 193 against 168 - and `check-workflow` runs the gate
  about nine times per shard. `--compare` runs them too, and so does the Linux
  CI lane's *Checking harnesses* step, which is the only place they run on
  every push.

Shell is the default orchestration language and **Go is the checking layer**.
Rust owns parsing, normalization, validation, indexing, and publishing. Python
requires a recorded need that cannot reasonably be met by those layers.

⭐ **The Go layer exists to DELETE the twin layer rather than to translate it.**
`check-twins` exists because two hand-written halves drift; one binary that runs
on both platforms cannot drift from itself, so the comparison stops being
necessary rather than getting faster. `CI-10` owns the port and
[`../tools/check/`](../tools/check/) is where it lands. ⚠ The module has an empty
require list and therefore no `go.sum`: nothing is fetched at build time, so a
build needs no network and `CI-04`'s dependency surface does not grow by a
language.

`check-twins.sh` has no PowerShell twin because it executes and compares both
halves of every listed pair. The gate runners are deliberately absent from its
pair list because including a runner would recurse.

⛔ **The pair list is SHRINKING, and a pair leaves it by being deleted rather
than exempted.** `CI-10` ports each rule into `tools/check/` and removes both
halves; four went on 2026-09-10, taking `check-twins` from **69 seconds to
15.9**. ⚠ A pair may only leave after
`sh scripts/common/check-bitcheck.sh --compare` has run it against both halves
over the same plants, which is the entry's own bound. This file's list of twinned
scripts above is therefore a list of what has NOT been ported yet.

⚠ The twin rule is about `common/`, where every script is a check that emits a
comparable `--json` verdict. `acquisition/fetch-releases.sh` has no twin and
does not belong in the pair list: what it emits depends on the network, so
running two implementations against one tree would compare the clock rather
than the answer, which is the exact failure `check-twins.sh` documents. A
Windows capture host will need a PowerShell fetcher; `ACQ-04` owns the runner
contract and is where that lands, rather than a second implementation written
now with nothing exercising it.

⭐ **Every `check-*` script outside `common/` is a mutation prover**, and so are
`common/check-examples.sh` and `common/check-handbook.sh`. That is the rule, and
it is stated as one rather than as a list with a count: the list here was short
by `common/check-handbook.sh` for a whole session, and the count beside it was
short by one for the same reason. All of them run in the `sh` gate except
`ci/check-workflow.sh`, which cannot, and each is a declared row in the
PowerShell one.

⚠ `acquisition/check-runner.sh` is the one exception to that last clause, and
this paragraph said the opposite until `CI-03` closed: `assert-disposable.ps1`
and `check-runner.ps1` exist now, so `check-runner` is a **real** row on the
PowerShell lane rather than a declared gap. A sentence saying a twin is not
written yet outlives the day it is written unless something re-reads it.

The corpus and publishing provers hold rules that are not platform-specific at
all, and the Rust suite exercises every one of them on both CI lanes; ⚠ what
they plant includes a symbolic link and a named pipe against a real filesystem,
and neither is available to an unprivileged Windows session. A second
implementation that skipped those two plants would report a smaller pass under
the same name, which is the shape `check-twins.sh` calls invisible drift.

⚠ `capture/check-capture.sh` is declared on the PowerShell lane for a third
reason again, and its row says which: it is an `sh` harness, and what it drives
includes `capture-run.ps1`, which the capture workflow's Windows job runs on a
real Windows host with no `-RouteTable`. That is a stronger control than this
lane could give and it happens somewhere else.

⚠ `check-staleness` is declared on the PowerShell lane for a **different**
reason, and its row says which: it plants nothing on a filesystem and needs no
POSIX-only feature. It is an `sh` harness with no PowerShell half, and it needs
`python3` for the independent derivation. Copying the neighbouring reason would
have recorded a gap that closes on the wrong event.

`ci/check-workflow.sh` is the one mutation prover deliberately kept **out** of
the gate. ⚠ This sentence carried an ordinal - "a thirteenth" - and it was wrong
the moment `ci/check-step-bodies.sh` landed, which is the third count in this
file to go stale that way. It is gone rather than corrected, because correcting
one only resets the clock. Two of its cases run the workflow's own *Repository
gate* step, so a runner listed in the gate that also invokes the gate would
re-enter itself; `check-gate.sh` keeps `check-twins` out of its pair list for
that reason and this is the same contract. The workflow runs it as a step of its
own instead, after every other step has passed, so it still runs on every push.

⚠ A script that sources another is clean to `shellcheck` only when both are
handed to one invocation, because it will not follow a source it was not given.
CI passes every script at once and a contributor checking one file does not, so
the directives live in each file rather than in how it is called.

⚠ A prover that builds an example resolves it through `CARGO_TARGET_DIR` when
that is set. Composing the path as `target/debug/examples` instead was measured
on 2026-09-06 to make all five corpus and publishing provers exit 2 on a host
with that variable exported, which the gate reads as a skip rather than as a
failure, so a whole tier of guards stopped proving anything and said so nowhere.
