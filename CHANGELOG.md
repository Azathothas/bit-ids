# Changelog

Nothing is released yet. Entries accumulate here until the first
`v1.*` tag, which is what `PUB-03` creates.

## Unreleased

### 2026-10-15T11:13:41Z

- ⭐ **The Windows firewall containment is proved on a runner**: `capture` run
  38722969612's blocked `curl.exe` could not connect, and the capture still
  verified over loopback.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-08`. No version bump and no deploy.

### 2026-10-15T10:00:05Z

- ⭐ **The Windows capture no longer cuts the host's route.** It blocks the
  capture's own programs at the firewall, reads the rules back, and proves on
  every dispatch that a blocked program cannot connect while the host can.
  Neither capture job severs its runner from GitHub now.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-08`. No version bump and no deploy.

### 2026-10-15T08:50:30Z

- ⭐ **qBittorrent 5.2.4 is published**, beside aria2-next 3.0.3 and
  Transmission 4.1.3: `data` holds 6 records across three targets, all in the
  published view, verified from a fresh clone. The first dispatch's source lane
  built and could not start the GUI on a runner, because the runtime set had
  been what one host lacked. It is now the GUI's whole library closure.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-01`. No version bump; the
  data branch is the deployment.

### 2026-10-15T08:30:05Z

- ⭐ **`check-assemble` runs on the Windows lane**, ported whole into
  `bit-check` with all 69 cases. It was compared against its shell half on the
  verdict, on every row label and by five mutants before that half was deleted.
  It was the last portable row.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-15T07:57:36Z

- ⭐ **Transmission 4.1.3 is published**, beside aria2-next 3.0.3: `data` holds
  4 records, all in the published view, verified from a fresh clone.
- ⚠ CI's workflow acceptance runs on five shards rather than four. One shard
  reached its 20-minute bound, and the bound stays.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-06` and
  [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump; the data branch is the
  deployment.

### 2026-10-15T06:55:02Z

- ⭐ **qBittorrent 5.2.4 has two routes, both driven.** The release route
  installs the `_lt20` AppImage, and the source route builds `release-5.2.4`
  against Qt 6.11.2's prebuilt archives and libtorrent 2.0.15 in about twelve
  minutes. Both run the GUI build under the adapter's own Xvfb. Captured on
  this host, the pair's records are publishable and build-equivalent.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-01`. No version bump and
  no deploy.

### 2026-10-15T06:02:19Z

- ⛔ **The second publication would have deleted the first.** Transmission
  4.1.3's pair assembled with both records publishable, and the publisher's
  dry run refused it with `E-STO-20`: the bundle carried one run's records
  alone. Nothing was pushed. A bundle now assembles onto the published tree, a
  path both carry must be byte-identical, and each bundle publishes the version
  schemes its records were ordered under, for the next assembly to read back.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-15T04:12:36Z

- ⛔ **A prerelease marker after digits was not read.** `5.3.0beta1`'s last
  token is `0beta1`, and the reader took letters from a token's start only, so
  qBittorrent's source route failed closed on 59 such tags. It reads every run
  of letters now and selects `release-5.2.4`; no other target's tags move.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-01`. No version bump and
  no deploy.

### 2026-10-15T03:24:40Z

- ⭐ **A capture runs three sessions, and the peer surface is dialled in every
  one.** A build that keeps one peer ID per session showed one value on that
  surface, and two such captures stated two different constants. The assembler
  now places each connection in the session it was dialled in. A third session
  makes it far less likely that a byte two random values happen to share is
  called fixed. Settled by the operator.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-06`. No version bump and
  no deploy.

### 2026-10-15T03:02:36Z

- ⛔ **The assembler read every segment on a peer connection as a handshake.**
  Transmission 4.1.3 sends an unchoke after its handshake on the same
  connection, so its first two-route capture was refused over five bytes. The
  first segment of a connection is its handshake now.
- ⚠ With that fixed, the pair's two records are provisional: the routes look
  divergent on two peer-ID fields, and neither is a difference between the
  builds. Recorded for the operator.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-06`. No version bump and
  no deploy.

### 2026-10-15T01:59:20Z

- ⭐ **No shell is left on the path from a dispatch to a bundle.**
  `assemble-dispatch.sh` and `assemble-bundle.sh` are now
  `bit-check assemble-dispatch` and `bit-check assemble-bundle`. They were
  compared with the shell halves on 41 inputs, including exit codes, messages
  and bundle bytes, before those halves were deleted. Only two differences
  remain, and both are intended.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-15T01:34:57Z

- ⭐ **A version scheme can declare a floor.** `transmission`'s refs carry 112
  tags its three-component scheme cannot order, from `0.5` to `3.00`, and no
  dates, so its source route failed closed on its own history. A floor of 4
  releases them as `below_scheme_floor`, only beside a selection at 4 or
  above, and the source route now selects 4.1.3. Resolutions are
  `bit-ids/resolution/2`; the bundle assembler reads both generations.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-06`. No version bump and
  no deploy.

### 2026-10-15T01:00:10Z

- ⭐ **THE FIRST PUBLICATION.** `capture-client` run 38's `aria2-next` 3.0.3
  pair is on the `data` branch: 30 objects, read back by the publisher and
  verified again from a fresh clone with `sha256sum -c` and `validate-corpus`.
- ⚠ The operator settled four questions, recorded in `TODO/PROGRESS.md`:
  - publish now;
  - rewrite the dispatch assembly in Go;
  - contain Windows captures with firewall rules;
  - keep the operator as the commit author.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump; the data branch
  is the deployment.

### 2026-10-14T22:07:27Z

- ⛔ **The second connector wrote carriage returns on Windows.** Its report and
  its `--describe` went through text-mode streams, so every value a reader took
  out ended in `\r`, and its own read-back, in text mode too, could not see it.
  It writes and reads back bytes now. Found by the Windows lane running
  `check-connector` for the first time.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T21:34:37Z

- ⭐ **`check-connector` runs on the Windows lane**, ported and compared like
  the five before it, with every transcript still composed by the harness and
  never by the observer.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T21:10:48Z

- ⭐ **`check-staleness` runs on the Windows lane**, ported and compared like
  the four before it. Its request identifier is re-derived with Go's SHA-256,
  independent of the Rust encoder as Python's was.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T20:58:13Z

- ⭐ **`check-formats` runs on the Windows lane**, ported and compared like the
  three before it, with Python's `sqlite3` still its database reader.
- ⚠ Two Rust comments said this check handed the CBOR to `cbor2`; it never
  did, and they say so now.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T20:24:35Z

- ⭐ **`check-release` runs on the Windows lane**, ported and compared like the
  two before it. Its checksums are read by `sha256sum` where a host has one and
  by Windows' own `certutil` where it does not, never by this project's own
  hashing.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T19:51:14Z

- ⭐ **`check-indexes` runs on the Windows lane too**, ported and compared the
  same way as `check-corpus`, over two plants in the index builder.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T19:37:05Z

- ⭐ **`check-corpus` runs on the Windows lane.** It is `bit-check check-corpus`
  now, ported case for case from its `sh` half and compared against it over the
  clean tree and two plants in the validator, before that half was deleted.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T19:25:50Z

- ⚠ `CI-07`'s Approach and Prove are rewritten to the route `CI-10` made: a
  declared Windows row whose subject is portable closes by its harness joining
  `bit-check`, and the other rows close by the event each one names.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-07`. No version bump and no deploy.

### 2026-10-14T19:14:26Z

- ⛔ **The install step ends what an install leaves in its session.** Nested
  `timeout`s each start their own process group, so a bound never reached the
  group beneath it, and a `sleep 600` outlived every bound. The step now lists
  its session after the wait, names what is left in `orphans.log`, and ends it.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-08`. No version bump and no deploy.

### 2026-10-14T18:19:57Z

- ⛔ **`check-workflow` runs a step as GitHub does, under `bash -e`.** Under
  `sh -c` a step that failed on one line and succeeded on its last was red on
  the lane and green in the harness. A probe hands the shell that shape.
- ⭐ **Transmission 4.1.3's release route captured on a runner**, with its
  evidence uploaded: 8 announces over 2 sessions, 2 peer streams, and peer IDs
  starting `-TR4130-`. `CLIENT-06`.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-08`,
  [`TODO/clients.md`](TODO/clients.md) `CLIENT-06`. No version bump and no
  deploy.

### 2026-10-14T16:51:01Z

- ⭐ **`capture.yml`'s Linux job no longer cuts the runner's route.** Its guard
  and fixture capture run in a network namespace, as `capture-client.yml`'s do;
  cutting the route severed the runner agent. The Windows job still cuts.
- ⛔ **A lane that failed before its install now arrives at the barrier**, as
  a failure, so the other lanes stop waiting at once. Before, it posted
  nothing, and they waited out the full bound.
- ⛔ **Transmission 4.1.3 writes an RPC verifier into its settings on exit**,
  which rule 12's scan refused to ship. The adapter's `stop` now deletes it.
  The release route installed on a runner in 562 seconds.
- ⛔ Transmission's source route fails closed: git refs carry no dates, and
  112 of the repository's historical tags are unorderable under its scheme.
  The two ways through change the version ordering and are the operator's to
  choose. `CLIENT-06`.
- ⛔ qBittorrent 5.2.3 has no second Linux route on this runner image: its
  build needs Qt 6.6 and noble ships 6.4.2. `CLIENT-01` records the four
  candidates.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-08`,
  [`TODO/clients.md`](TODO/clients.md) `CLIENT-01`. No version bump and no
  deploy.

### 2026-10-14T15:52:24Z

- ⭐ **TRANSMISSION 4.1.3 BUILDS ON BOTH ROUTES WITHOUT ROOT.** The release
  route builds the vendor's source tarball and the source route builds the
  tag's clone, both against libcurl headers fetched through a private package
  index. Driven as an unprivileged user: 1013 and 1078 seconds, both answering
  4.1.3.
- ⛔ **An install's bound follows whether its route compiles, not its name.**
  Each adapter declares `compiles`, and `capture-client.yml` reads it before the
  install. A release route that compiles had been bounded as a download.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-06`. No version bump and
  no deploy.

### 2026-10-14T13:46:19Z

- ⭐ **THE CAPTURE-TO-PUBLISHER PATH RAN ON RUNNERS, END TO END.** `assemble-bundle`
  run 38574255232 turned `capture-client` run 38 into a 30-object bundle, and
  `publish-data` run 38574390252 verified it as a dry run and pushed nothing.
  ⛔ The first append to `data` is the operator's call.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-14T12:49:29Z

- ⭐ **THE PUBLISHER'S `bundle` HAS A PRODUCER.** `assemble-bundle.yml` turns
  one `capture-client` dispatch's artifacts into records, run manifests and a
  publication tree, and uploads it as `bundle`. The version scheme comes from
  each resolver's own record. Run 38's pair makes a 30-file bundle.
- ⛔ **The publisher refuses a bundle with no `MANIFEST.json` or `SHA256SUMS`,
  or with a digest that does not match, before it pushes.** It used to push one
  and refuse only at the read-back.
- ⚠ A record's version output is stored as `version.txt`, because the release
  refused `version.err` as a file with no media type.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy;
  nothing is appended to `data`.

### 2026-10-14T12:30:39Z

- ⭐ **A CAPTURE PAIR IS A CORPUS.** `assemble-capture` writes a run manifest
  beside each record. The manifest describes the watched host, carries both
  routes' acquisitions and evidence, and gets its phases from the run's own
  timestamps. It is bound to its profile before it is written. Run 38's
  `aria2-next` 3.0.3 pair forms a valid store, and both records may enter a
  published view.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-14T12:13:51Z

- ⭐ **EVERY LANE'S CAPTURE NOW FOLLOWS EVERY ROUTE'S INSTALL.** The lanes of a
  `capture-client` dispatch install on separate runners, and run 37's release
  lane observed before the source route had installed. Each lane now arrives at
  a barrier, a check run on the commit named for its run attempt, and waits for
  the others; it never fails a lane, and writes `passed`, `failed` or `timeout`.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-14T11:19:01Z

- ⭐ **A CAPTURE KEEPS THE HOST FACTS A RUN MANIFEST NEEDS.** The claim marker
  records the runner image, the operating system, whether the host is GitHub's
  own and the uptime at the claim, and goes with the evidence; the attestation
  records the uptime at finish. The install record carries the installed size,
  a retrieval time and the signature status.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-14T10:53:52Z

- ⛔ **A RECORD'S CONNECTOR VERSIONS ARE MEASURED.** Both were `0.0.0`, written
  by the assembler, and for the second connector nobody's measurement. The
  observer prints its version, the connector's described version is kept, and a
  lane whose attestation carries neither is refused.
- ⭐ The operator settled how a run manifest fits a two-runner dispatch: a
  workflow barrier, and the watched host. `CI-09` carries the plan.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-14T10:23:06Z

- ⭐ **THE FIRST PUBLISHABLE PAIR FROM A STOCK CLIENT.** `capture-client` run
  37's `aria2-next` 3.0.3 records, assembled again, are `publishable` on both
  routes and `build_equivalent` across them.
- ⛔ **A byte two random values happen to share is no longer called fixed.** A
  fixed run between varying ones is kept only when its length times one less
  than the number of distinct values reaches two; five plants of the rule were
  each refused.
- ⚠ They are not a corpus yet: no `manifest.json` is written beside a record,
  which now leads the work order.
- Record: [`TODO/schema.md`](TODO/schema.md) `SCHEMA-04`,
  [`TODO/ci.md`](TODO/ci.md) `CI-09`. No version bump and no deploy.

### 2026-10-14T10:14:21Z

- ⭐ **BOTH PEER CONNECTIONS ARE ANSWERED ON A RUNNER.** `capture-client` run 37
  put `aria2-next` 3.0.3 on both routes, the re-dial left from `127.0.0.2`, and
  `peer_wire/peer_id` is `patterned` and agrees across routes.
- ⛔ **The pair is refused over a coincidence.** Each capture's two announced
  tails share one random byte, the sampling classifier calls it fixed, and two
  routes with fixed bytes in different places disagree. `SCHEMA-04` leads the
  work order.
- ⚠ `check-workflow`'s clean-tree gate control prints the gate's failed rows,
  because CI run 188 reported it failed and not why.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`, [`TODO/schema.md`](TODO/schema.md)
  `SCHEMA-04`. No version bump and no deploy.

### 2026-10-14T07:35:56Z

- ⭐ **THE LAB'S SECOND PEER CONNECTION LEAVES FROM A SECOND ADDRESS.**
  `aria2-next` 3.0.2 answered one of two connections from `127.0.0.1`, and its
  vendored libtorrent's source defaults to one connection per IP. Driven
  against it in a loopback namespace, a second from `127.0.0.2` was answered
  with a peer ID of its own, and through `client-capture` and the adapter the
  planted alternative gave one peer stream where this gives two. `bind::dial_from` and `Lab::dial_again_from`
  carry it, and `client-capture` re-dials from `127.0.0.2` upward.
- ⛔ **`socket2` 0.6.5 is a new dependency, for that one door**: the standard
  library cannot bind before it connects, and the workspace forbids the
  `unsafe` a hand-made `bind(2)` would need. Three packages in the lockfile,
  registered.
- Record: [`TODO/observer.md`](TODO/observer.md) `OBS-04`,
  [`docs/supply-chain.md`](docs/supply-chain.md). No version bump and no deploy.

### 2026-10-14T07:15:39Z

- ⭐ **`aria2-next` 3.0.2 IS ON THE WIRE THROUGH BOTH ROUTES.** `capture-client`
  run 36 is green on release and source, its source build installed in 694
  seconds, and both lanes assembled. The second connector corroborated every
  sample of every field exactly, and both recorder runs closed with their last
  snapshot.
- ⛔ **The pair is one sample short of publishable.** The build answered one of
  the lab's two peer connections, so `peer_wire/peer_id` is one value per
  capture and `E-PUB-03` refuses the pair. Both connections come from
  `127.0.0.1`; the work order now leads with dialling the second from another
  loopback address.
- Record: [`TODO/ci.md`](TODO/ci.md) `CI-09`, [`TODO/observer.md`](TODO/observer.md)
  `OBS-04`. No version bump and no deploy.

### 2026-10-14T05:03:32Z

- ⛔ **`aria2-next` 3.0.2 IS STARTED WITH EVERY DISCOVERY SWITCH IT REPORTS OFF.**
  `capture-client` run 35 refused it at the adapter's read-back: 3.0.2 no longer
  reports `enable-dht6`, and the options dump its refused capture uploaded showed
  `ed2k-port-mapping`, UPnP for its ED2K ports, on by default and named nowhere.
  The adapter switches it off, holds every reported key of the DHT,
  port-mapping, local-discovery and peer-exchange families to `false`, and
  refuses any option reported `true` that it has not judged; driven against the
  same bytes in a namespace holding loopback alone.
- ⛔ **THE SOURCE ROUTE BUILDS `v3.0.2` ON GCC 13.** `src/disk/File.cc` calls
  `std::exchange` without `<utility>`, which the vendor's GCC 11 tolerates. The
  route configures with `CXXFLAGS='-include utility'`, changes no vendor file,
  and writes the line it ran beside the build. Measured whole on the session
  host at 870 seconds, so the source lane's bounds rose: install 1500 seconds,
  job 45 minutes.
- ⛔ **A CLOSED FLIGHT-RECORDER RUN SHOWS ITS LAST SNAPSHOT.** The closing request
  carried a title and no text, which erased the snapshot posted just before it;
  both of run 35's closed runs were empty. It is one request now, asked once
  more without the text if the API refuses it.
- ⛔ **A FAILED INSTALL'S LOG REACHES THE JOB LOG.** *Install the client* read
  `rc=$?` on the line after the install, which `bash -e` never reaches on a
  failure. `check-step-bodies` asks a refusal's output now. ⭐ `ci.yml`'s shell
  lint had the same shape, two checks on two bare lines, and keeps both
  statuses now; `check-workflow` plants a finding only shellcheck makes.
- Record: [`TODO/clients.md`](TODO/clients.md) `CLIENT-14`,
  [`TODO/ci.md`](TODO/ci.md) `CI-08` and `CI-09`. No version bump and no deploy.

### 2026-10-14T03:48:10Z

- ⛔ **A REFUSED CAPTURE UPLOADS ITS EVIDENCE, AFTER RULE 12.** The two
  contained runs of 2026-10-11 left job logs and no bytes, because *Upload the
  evidence bundle* ran on success only. `if: always()` would have shipped a
  bundle the credential scan, which runs on the capture's success path, had
  never read; a scan step now runs on a refused capture and the upload follows a
  success or a scan that passed. `check-workflow` refuses an ungated upload.
- ⭐ `scripts/capture/scan-bundle.sh` is the one home for both rule-12 rules,
  which `capture-client.sh` and `install-client.sh` each carried a copy of.
- ⛔ **THE FLIGHT RECORDER CLOSES ITS CHECK RUN.** It ran to a limit no job
  reaches, so every run it opened stayed `in_progress` - the silence it reads as
  lost contact, from jobs that ended cleanly. The job's last step stops it, and
  it reads the closure back.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09` and `CI-08`. No version bump and
  no deploy.

### 2026-10-14T02:55:53Z

- ⛔ **THE SECOND CONNECTOR NO LONGER REFUSES A BUILD THAT VARIES.** Both
  contained `capture-client` runs of 2026-10-11 ended with `cpython-stdlib`
  refusing *two segments disagree about tracker_http/peer_id* over `aria2-next`
  2.8.6's two sessions. It now reports every sample of a field in document
  order, comma-separated, and a peer ID that moves between sessions is two
  samples of the `patterned` field `SCHEMA-04` already models.
- ⭐ **`assemble-capture` corroborates every sample, not the first.** It lines
  the connector's samples up against the observer's by position; a record
  carries the first position where they part, and a count that differs is a
  disagreement about what the build sent.
- ⭐ `check-capture-client` reproduces the 2026-10-11 failure through the real
  observer, runner and connector, and passes it now. Twelve plants into the
  connector and the assembler were each refused.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`. No version bump and no deploy.

### 2026-10-11T04:47:32Z

- ⭐ **THE DOCS ARE AMENDED TO WHAT THE ROOT FIX MADE TRUE.** `capture-client.yml`'s
  comments no longer explain its bounds as the cure for a hang; section 10 of
  `docs/architecture.md` no longer says nothing has been installed or that
  nothing assembles a store; every *before the route is cut* in a page about
  `capture-client` says what containment is now.
- ⭐ **`scripts/README.md` opens with what to run before a push, by what
  changed**, each row naming the CI job it mirrors and how long both take.
- ⛔ The record's work order now leads with what stops a capture today: the
  second connector refusing a peer ID that differs between announces.
- Record: [`docs/history/SESSION-2026-10-11-CONTAINMENT.md`](docs/history/SESSION-2026-10-11-CONTAINMENT.md).
  No version bump and no deploy.

### 2026-10-11T03:53:05Z

- ⛔ **THE CAPTURE HANG IS FOUND AND REMOVED, AND IT WAS NEVER A HANG.** A
  flight recorder posting the host's state off the runner showed *Install the
  client* finishing in seconds and the host losing contact the moment the next
  step cut its default route, which the runner agent talks to GitHub over.
  Runs 3 to 34, thirteen bounds, and every reading recorded under `CI-08` were
  aimed at a step that had already ended.
- ⭐ `scripts/capture/contained.sh` runs the egress guard and the capture in a
  network namespace holding loopback alone; `capture-client.yml` no longer
  touches the host's routes, and a run completed in two minutes with its log
  and its install artifact. `check-workflow` refuses a capture job that neither
  cuts a route nor runs through the wrapper.
- ⭐ `scripts/ci/flight-recorder.sh` posts blocked tasks with their kernel
  stacks, memory, disk, the kernel ring buffer and the newest logs to a check
  run from the job's first step.
- ⛔ **The rule that forbade looking is rewritten**: `TODO/RULES.md` and
  `docs/AGENTS.md` absolute 16 now say a hang is measured from off the host
  before it is bounded. The old wording is in
  `docs/history/HANG-2026-10-11.md`. The `runner` input added earlier today on
  the wrong premise is removed.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-10-11T03:07:36Z

- ⭐ **`CI-10` IS CLOSED.** CI run 179 read both new steps back green on a runner,
  and its four acceptance shards took 12.3 to 15.9 minutes under the 20-minute
  bound that was lowered from 45. No `IN_PROGRESS` entry is left.
- ⛔ **`capture-client` run 33 wedged in *Install the client* on both lanes**,
  twenty-five days after run 32 did, past its job's 35-minute bound with no
  artifact. ⭐ `capture-client.yml` takes a `runner` input, a choice of
  `ubuntu-24.04` or `ubuntu-22.04`, which is the route absolute 16 asks for
  rather than another bound.
- ⭐ **`CLIENT-06` has a same-version pair for Transmission 4.1.3**: its release
  tarball, which built here in 695 seconds with libcurl's headers added, and its
  tag, on the model `aria2-next` measured.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`;
  [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-06`;
  [`TODO/clients.md`](TODO/clients.md), `CLIENT-06`. No version bump and no
  deploy.

### 2026-10-11T02:15:39Z

- ⭐ **THE GO MODULE IS FORMATTED AND VETTED IN CI, AND THE HARNESS CASES RUN.**
  Two Linux-lane steps close `CI-10`'s two residuals: *Go formatting and vet*
  over `tools/check`, and *Checking harnesses*, which runs
  `check-bitcheck --harnesses` on every push for the first time.
- ⛔ **A weakened checking rule passed every row CI ran.** With `check-cache`'s
  `E-CAC-01` refusal turned into a pass, the gate stays green and only the
  harness cases refuse it; `check-workflow` now plants exactly that, beside an
  unformatted Go file and a `Printf` that vet refuses.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-10-11T01:33:53Z

- ⛔ **THE RECORD IS REWRITTEN TO CURRENT TRUTH.** The 2026-09-29 session ended
  without running its ending, and `TODO/PROGRESS.md` and `docs/history/RESUME.md`
  were 672 and 372 lines of narrative. The record's old text moved verbatim to
  `docs/history/PROGRESS-2026-09-29.md`, the review lessons both carried moved to
  `docs/lessons.md`, and that session has a record reconstructed from the log
  and CI, `docs/history/SESSION-2026-09-29-PORT.md`.
- ⛔ **The work order sent `CLIENT-01` and `CLIENT-06` to a dispatch run 32 had
  already answered.** It says what is left now: a capture route that does not
  run *Install the client* as it stands. `TODO/ci.md` carried the same stale
  claim under `CI-08` and is corrected in place.
- ⭐ **`check-project` derives `TODO/SUMMARY.md`'s effort inventory from the
  index**, which said 49 L and 0 XL against 51 and 1, and a planted case in
  `check-bitcheck` refuses the stale sentence.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`, and
  [`docs/history/SESSION-2026-09-29-PORT.md`](docs/history/SESSION-2026-09-29-PORT.md).
  No version bump and no deploy.

### 2026-09-29T04:23:35Z

- ⛔ **BOTH HALVES OF `mine-repo` ARE DELETED, AND NO TWIN PAIR IS LEFT.**
  `c65a553` committed the comparison with its six repairs in it; this is the
  deletion, and `CI-10`'s port is done. `bit-check mine-repo` is the helper, and
  `check-twins` compares only the doctor's probe pair.
- ⚠ `check-twins` lost the machinery that ran pairs at once, which nothing called
  any more, and both gate runners stopped naming a `common/` pair that is gone.
- `docs/methodology/references.md` runs the helper from `tools/check`, and a
  relative `--out` still lands at the repository root.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T04:15:24Z

- ⭐ **`mine-repo` IS PORTED, AND COMPARED WHOLE FOR THE FIRST TIME.** The
  pair's only comparison was `--selftest`, four cases of its page joiner.
  `check-bitcheck` now serves both routes, the control, the clone and the trim
  from stub `gh`, `curl` and `git`, and compares every file each implementation
  leaves behind. The port is a helper, dispatched before the checks and in no
  gate row. ⚠ Nothing is deleted: this change is the COMPARISON.
- ⛔ **Six drifts, each repaired in the half that had it.** The `sh` half mined
  `a/b/c`, kept the last of two targets, and read `--route prxy` as `auto`, so a
  route asked for to keep the operator's token out probed gh and used it. The
  twin answered usage errors with exit 1, and on Linux never trimmed `.next` or
  `.venv` and left `api/.control.json` in every proxy-route corpus: pwsh hides a
  dot-name, and `Get-ChildItem` and `Remove-Item` pass one by without `-Force`.
- ⭐ Measured once against the real proxy and a real clone of one public target:
  after the repairs, the three leave the same corpus, file for file.
- ⚠ The port names a list cut off at the proxy route's page cap, which both
  halves reported as `ok`, and trims `vendor/bundle`, which neither ever did.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T03:21:20Z

- ⛔ **BOTH HALVES OF `check-remote-items` ARE DELETED**, after `f0e4f7f`
  committed the comparison and run 174 read it back. Both lanes now run the Go
  binary against the real open items, and `check-twins` compares one pair:
  `mine-repo`. `Invoke-Check` left the PowerShell runner with its last caller,
  and a ported row that cannot run on that lane now says why, as on the other.
- ⛔ **The operator's update procedure claimed a check nothing made.** Step 1 of
  `docs/supply-chain.md` said the tag is verified as a published release; the
  `sh` half's header said so and its code never asked. The sentence is gone, and
  the page's warning that no run backed it is replaced by the fifteen cases that
  now do.
- ⚠ Three comments beside the rows were false: both runners said every `common/`
  half was ported while this one was not, the PowerShell runner cited an
  `Invoke-Native` no commit ever defined, and `check-runner` was said to lack a
  PowerShell half it has had since `CI-03`.
- ⚠ `go run . <check>`, the runnable form `CI-05`, `ACQ-05`, `LIB-01` and the
  update procedure give, exits 1 for a refusal and for a check that could not
  run alike. Measured; the procedure now says to read the report, and the
  binary's usage comment says why.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T03:07:40Z

- ⭐ **`check-remote-items` IS PORTED, AND THE GATE'S ONE SKIP NOW RUNS ON EVERY
  HOST.** Its logic had only ever been proved by whatever happened to be open
  when a CI lane ran it; `check-bitcheck` now serves it from a stub `gh` and a
  stub `curl`, fifteen cases in the DEFAULT mode, with no network at all. ⚠ Nothing
  is deleted and no gate row has moved: this change is the COMPARISON.
- ⛔ **The comparison covers the whole report, and found two drifts, both the
  twin's**: it said *the platform* where the `sh` half says *GitHub*, and it
  discarded gh's own error on a failed listing. Repaired in the twin, CRLF kept.
  The twin fetched the three real `action.yml` files over the network and still
  agreed byte for byte.
- ⛔ **Two harness defects, found by running it**: a multi-line answer broke the
  report's own row accounting, so the run refused to print its findings - a
  disagreement is an indented `diff -u` now - and the dispatcher could not print
  a partial report before a refusal, which both halves do.
- ⚠ Eight plants into the port, all refused - two only after a case was added for
  each: a pin the diff removes, and two pins on one line.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T02:46:08Z

- ⛔ **BOTH HALVES OF `check-catalogue` ARE DELETED, AND `store-lib.ps1` WITH
  THEM**, after `c40784f` committed the comparison with the twin's repairs in it.
  The library's only two callers were the harness twins now in the Go binary, so
  it went rather than stay as code nothing calls. `check-twins` compares two pairs.
- ⚠ **The Windows lane's declared rows named an event that would never come.**
  Seven said *it needs store-lib.ps1*; each says *it closes when its harness is
  ported into bit-check* now, because a row whose reason names the wrong event
  closes on the wrong day - `CI-01` recorded that shape once already.
- ⭐ Run 171 put the Go `check-cache` on a real Windows runner for the first time:
  `ok`, with no skip on that lane.
- `LIB-01`'s `Prove` gains a runnable form under a dated note, and `CI-07` records
  that its first step is deleted and its class-A route changed.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10` and `CI-07`. No version bump and no
  deploy.

### 2026-09-29T02:39:46Z

- ⭐ **`check-catalogue` IS PORTED, THE SECOND HARNESS, and it needed nothing
  `harness.go` did not already have.** ⚠ Nothing is deleted and no gate row has
  moved: this change is the COMPARISON.
- ⛔ **Its plants go where it reads**: a socket named in the crate, a transport in
  the manifest, and the library's `E-LIB-02` disabled at both sites. All three
  implementations agree on all seven cases.
- ⛔ **The comparison found two drifts and both were the twin's**: `Select-String`
  and `-match` fold case, so the twin refused a lower-case `tcpstream` and a
  `Tokio` key that `grep` - and Rust - tell apart. A clean tree spells every needle
  one way and could never have shown it. Repaired in the twin, CRLF kept.
- ⚠ **A case was named for something it did not do.** *A library that accepts a
  moved record* is refused as `E-LIB-04`, which a mutation pass exposed; renamed,
  and kept as the one case proving the harness reads the code and not only the
  exit. Seven plants into the port: six refused, one equivalent on the fixture.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T02:09:27Z

- ⛔ **BOTH HALVES OF `check-cache` ARE DELETED**, after `621f93a` committed the
  comparison with the twin's repair in it, so the history carries a repaired twin
  before a deleted one. `bit-check check-cache` is the row on both lanes and
  `check-twins` compares three pairs.
- ⚠ **The deletion was not two files.** Both gate runners queued the row,
  `check-twins` compared the pair, and `check-defaults` ran it as a subject - which
  moved INSIDE the build gate, where leaving it below would be the one-gated-door
  defect that block's own comment records.
- ⚠ `--permitted` has no caller in this tree now: `check-cache` calls the register's
  reader directly. The mode stays as the command a person runs, and every comment
  that named the old caller is amended.
- ⛔ **The claim audit found a runnable form that does not run.** `CI-05`'s amended
  Prove gives `go run ./tools/check check-project` from the repository root; it
  exits 1, because the module is `tools/check`. Corrected under the original
  sentence, and `ACQ-05`'s new note uses the form that exits 0.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T02:03:01Z

- ⭐ **THE DEPENDABOT REPAIR WAS DRIVEN ON THE REAL SYSTEM, AND THE PUSH WAS THE
  WHOLE EXPERIMENT.** Dependabot ran on the changed configuration within seconds
  of `2451816` landing and closed pull request 2 itself at 01:25:41: *"rusqlite is
  no longer being updated by Dependabot"*. ⚠ Pull request 1's closure said *no
  longer updatable*; the new wording names the configuration rather than the tree,
  which is what says the ignore rule was read. Nothing here closed an item.
- CI run 168 on `2451816` is green on all six jobs, read back through rule 8's
  route. The settled-decision row, `FOUND-04` and RESUME say so in place.
- Record: [`TODO/foundation.md`](TODO/foundation.md), `FOUND-04`. No version bump
  and no deploy.

### 2026-09-29T01:58:49Z

- ⭐ **`check-cache` IS PORTED, AND IT IS THE FIRST HARNESS IN THE BINARY.**
  `bit-check check-cache` carries its thirteen rows, and `tools/check/harness.go`
  carries what it needed of `store-lib`. ⚠ Nothing is deleted and no gate row has
  moved: this change is the COMPARISON, which is the only way a pair may leave.
- ⛔ **Its plants go into what the harness reads**: a register permitting the
  scenario's own target, one permitting another, a missing register, and a second
  retrieval repeating the first. All three implementations agree on all six cases,
  on the exit code and byte for byte, after ONE drift was repaired - the twin's
  could-not-run message named a flag the call does not use.
- ⭐ **An exit code of 1 does not say which row failed**, so each case also pins
  the Go verdict line; three of seven mutation plants were caught by that line
  alone. One survived, the zero-pass guard, which nothing here can reach.
- ⛔ **The harness cases are out of the gate, measured**: they build the scenario
  cold in a scratch tree, which took `check-bitcheck` from 23 to 39 seconds and the
  gate from 168 to 193. `--harnesses` and `--compare` run them.
- ⭐ The example's path is asked of cargo rather than composed, and the register
  is read by `check-licences`' own reader, checked identical over eighteen runs.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-29T01:13:49Z

- ⛔ **A DECLINED BUMP CAME BACK AS A NEW PULL REQUEST.** Pull request 2 re-offers
  `rusqlite` 0.40.2, opened by dependabot on 2026-09-18 and checked again by its
  run of 2026-09-25, and red on all six jobs of CI run 167, because
  `check-licences` refuses 18 unregistered packages - the register working.
- ⚠ **The record's *pull request 1 is closed unmerged* was true and was not the
  decision's doing.** Dependabot closed it itself, *"no longer updatable"*, two
  minutes after `76c32ea` took 0.40.2 on 2026-09-15; the revert made the bump
  available again, so the bot offered it anew.
- ⭐ `.github/dependabot.yml` now ignores every `rusqlite` version from 0.38.
  ⚠ The first reasoning about how was wrong and was corrected by reading
  dependabot-core: its cargo rules call a change of `y` in `0.y` MAJOR, so both
  listed types resolve to `>= 0.38`, a patch inside 0.37 is still proposed, and a
  security update is not filtered by update type at all.
- ⚠ Pull request 2 is left to the operator: nothing here closes an item.
- Record: [`TODO/foundation.md`](TODO/foundation.md), `FOUND-04`, and
  [`TODO/PROGRESS.md`](TODO/PROGRESS.md) under *Settled decisions*. No version
  bump and no deploy.

### 2026-09-17T10:05:14Z

- ⛔ **BOTH HALVES OF `check-project` ARE DELETED**, 997 lines of `sh` and its
  hand-written PowerShell twin. It was the last `common/` pair and the biggest;
  `bit-check check-project` is the row on both lanes and `check-twins` is down to
  **four pairs**.
- ⚠ **It left that list the only way a pair may.** The pre-deletion comparison ran
  49 planted cases against all three implementations, agreeing on the exit code
  and byte for byte on the `--json` line. After the deletion the same cases print
  *the sh half is gone, not compared*: the window closes with the halves, which is
  why the run before is the one that counts.
- ⚠ **The deletion was not one file.** Both gate runners queued the row,
  `check-twins` compared the pair, `check-workflow` planted `exit 2` into that
  exact path, and `check-defaults` ran it as a subject.
- ⛔ **`check-workflow`'s unrunnable-check plant had to stay an `sh` row.** Planting
  into a Go source file would fail the BUILD, and a gate whose binary did not
  build skips every ported row at once - a different plant with a different
  answer. It is `check-shell.sh` now.
- ⛔ **And `check-defaults`' subject moved INSIDE the build gate**, not below it.
  Outside, a host where the build failed would run a missing command that answers
  127 under every environment, which `run_subject` reports as a subject answering
  identically everywhere - the one-gated-door defect that block's own comment
  records.
- ⚠ `TODO/ci.md`'s `CI-05` names the two deleted halves in its Approach and its
  Prove; both are amended in place with a dated note rather than rewritten, and
  the Closure evidence is untouched.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-17T09:08:38Z

- ⭐ **`check-project` IS PORTED, AND IT IS THE BIG ONE.** 997 lines of `sh` and a
  hand-written PowerShell twin, holding twenty-nine refusals over this
  repository's own invariants, are one Go check. The binary carries **eleven** checks now.
  Measured here: the `sh` half 0.72s, the twin **2.60s**, the Go check **0.18s**.
- ⚠ **Nothing is deleted and no gate row has moved.** This change is the
  COMPARISON, which is the only way a pair may leave `check-twins`' list. 49 cases
  run each rule against all three implementations: 143 cases in total, 143 passed,
  agreeing on the exit code and byte for byte on the `--json` line.
- ⛔ **AND THE COMPARISON FOUND THREE DRIFTS, ALL THE TWIN'S.** `@(...) |
  Sort-Object` wraps the input and leaves the output unwrapped, so an empty set
  arrived as `$null` and `.Count` THREW - turning the refusal *a result too small
  to be real* exists for into a crash. The Total row was compared as a whole LINE
  where the `sh` half compares five counts. And two checks were joined by `-or`,
  so a tree failing both counted one failure there and two here.
- ⭐ All three are fixed in the twin rather than absorbed by the port. The Windows
  gate lane runs that file today.
- ⛔ **AND THE HARNESS COULD NOT PRINT ITS OWN FAILURES.** `store_report` counted
  rows by LINES and a disagreement is three lines, so the first run with something
  to say returned 1 having printed no rows at all. It counts row STARTS now, in
  both halves of `store-lib`.
- ⚠ Two more defects in `check-bitcheck` itself: a case that overwrote a real
  tracked file and `unplant`ed it by deletion, and a plant rewritten three times
  that `git rm --cached` then refused to remove.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-17T06:51:42Z

- ⛔ **THE ASSEMBLER WROTE A STORE `validate_corpus` REFUSES AND SAID NOTHING.**
  A green two-lane assembly printed two stars and two paths, and a reader took
  that for a store a consumer could open. It is not one: every record it writes
  is a record with no run beside it.
- ⚠ **`CI-09`'s residual put the gate a whole release away** - it read *which
  `check-store` accepts and a publication would not*. `validate_corpus` refuses
  such a store today, `E-CRP-01` per record, so nothing was missing but the
  asking. The residual is amended in place rather than implemented.
- ⭐ **`report_corpus` asks it, over the records that reached the store**, and
  prints the violations plus the reason a reader would otherwise look up: no step
  of the capture path writes a `manifest.json`. ⚠ It reports rather than refuses,
  because validity and publishability are separate gates at every level here.
- ⭐ **The control is the half that matters**: `check-assemble` requires every
  record the assembler wrote to be NAMED, taken from the assembler's own write
  lines rather than retyped, so one fixed sentence cannot satisfy it. Five
  plants, each verified applied, all refused, clean either side.
- ⛔ **AND ASKING FOUND A DEFECT IN WHAT THE ASSEMBLER WRITES.** A record's
  evidence list covers every lane, because a field citing the other route's
  install record is what makes the pair comparable - and the write loop copied
  only its own lane's four files. Each record cited four artifacts at paths under
  its own evidence root that nothing had written to.
- ⚠ **`E-CRP-03` is that refusal and it cannot fire**: it is checked per run
  manifest and none exists. The harness accepted it too, counting *at least one*
  copied file where half the files satisfy it. ⭐ Fixed, and the assembler now
  names a citation it did not carry, because no rule can.
- ⚠ **The tree is read back off the disk**, digest and length of the bytes that
  arrived rather than the record's claim about them. An empty one is what hid the
  defect: `E-CRP-06` is checked over the tree.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`. No version bump and no deploy.

### 2026-09-17T06:34:11Z

- ⛔ **A `CI-09` RESIDUAL CALLED A `SCHEMA-04` DECISION A DEFECT, AND IT IS
  STRUCK.** It read *`field_state` DISCARDS the lifetime, so no record can say
  whether a tail is per-connection or per-session*. ⚠ That discard is stated as a
  Decision twice in `SCHEMA-04` and again in `sampling.rs`, which own the
  reasoning: a lifetime stated in two documents can disagree with itself.
- ⭐ **And the pair of documents does answer it** - `bind_sampling` already
  refuses a variation claim over a plan that varied nothing, `E-BND-20`. ⛔ What
  was genuinely missing is the ASKING: a consumer holding both had to re-reason
  about the plan itself, which is a second implementation of the classifier in
  every consumer.
- ⭐ **`SamplingPlan::lifetime_of(varied)` is that question**, and it is
  deliberately weaker than the classifier: a plan that varied two dimensions
  answers `unknown` where the samples may still separate them. ⛔ A test holds
  that the two derivations never name DIFFERENT lifetimes, over every plan
  shape, with a control asserting the derivation is not simply `unknown`
  everywhere.
- ⚠ **Acting on the residual as written would have reversed a recorded
  decision**, which is why this entry is a correction rather than a schema
  change. `PatternRun` is unchanged.
- Record: [`TODO/schema.md`](TODO/schema.md), `SCHEMA-04`. No version bump and no
  deploy.

### 2026-09-17T06:19:59Z

- ⭐ **`mine-repo`'s clone is bounded, in both halves, with ONE idiom.** The
  recorded blocker was that `timeout.exe` on Windows is a PAUSE, so a wrapper
  would have made the two halves need two idioms and given `check-twins` a
  difference to compare. ⛔ **That premise was the defect**: a bound does not
  have to be a wrapper. `git -c http.lowSpeedLimit -c http.lowSpeedTime` is
  git's own limit on a transfer that has stopped moving, spelled identically on
  every platform it runs on.
- ⭐ **Driven against a listener that accepts a connection and never answers.**
  The shipped settings, read out of the two files rather than retyped, abandoned
  the clone at **60 seconds**, exit 128; the same clone with them removed was
  **still waiting at 25 seconds**. Both halves carry
  `http.lowSpeedLimit=1024 -c http.lowSpeedTime=60`, character for character -
  the same numbers the adapters and `install-rootless.sh` already bound a stalled
  `curl` with.
- ⚠ **`--selftest` is offline and does not reach the clone**, so `check-twins`
  cannot see this pair agree about it. The comparison is a driven measurement
  rather than a standing check, and `TODO/ci.md` says so.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-17T05:55:24Z

- ⛔ **THE HANG IS NOT PRIVILEGE, MEASURED.** `capture-client` run 32 on
  `a709e2d` is the first dispatch of the rootless install: both lanes' *Install
  the client* never returned, both jobs were cancelled at **forty minutes**, and
  the run uploaded **zero artifacts**. ⚠ `ACQ-06`'s residual asked whether the
  step thirteen bounds could not end ends once `sudo` and `/usr/local` are gone.
  It does not.
- ⭐ **The same run proves the REST of `ACQ-06` on a real runner.** The
  unprivileged *Claim the host* succeeded in **0s** on both lanes, and *Resolve
  the release artifact* - which now selects and fetches the vendor's checksums
  document through the rootless installer - succeeded in **1s**. So the
  installer's bounded fetch, the rootless claim guard and the digest resolution
  all work on that host, and the install still wedges.
- ⛔ **And nothing further is being done about it.** `TODO/RULES.md` and
  `AGENTS.md` absolute 16: a hang is a dead end, not a subject. The measurement
  is recorded; no fifteenth bound follows it.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-06`, and
  [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-17T04:49:43Z

- ⭐ **THE RELEASE LISTING PUBLISHES A DIGEST AND THIS PROJECT WAS NOT READING
  IT.** Three of four adapters declared that no digest existed while one sat in
  the very response the resolver already holds. ⚠ Found by a claim audit that
  re-read those adapters' comments against the LIVE listings rather than against
  the fixtures they were written from.
- ⭐ **A resolution now reaches one of FOUR dispositions** - `both`,
  `vendor-document`, `source-listing`, `unpublished` - and two targets moved from
  identified to verified. ⛔ **A host is not a vendor**, and the record says which
  answered: the party publishing a listing digest is the party serving the bytes.
- ⭐ **`install-rootless.sh --from-resolution` keeps four adapters out of the
  decision.** Four adapters choosing between four dispositions would be four
  copies of one decision, which is the one-gated-door shape.
- ⚠ **The listing fixtures are re-recorded at the SAME tags** and gained the
  field the reader learned to read; not one asset name under test moved.
  `aria2`'s six assets carry no digest, which is the source's answer and the
  shape that keeps the other branch exercised.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-06`. No version bump
  and no deploy.

### 2026-09-17T03:32:23Z

- ⭐ **`ACQ-06` IS CLOSED AND WITH IT EVERY OPEN P0.** One rootless installer,
  [`scripts/acquisition/install-rootless.sh`](scripts/acquisition/install-rootless.sh),
  is what every release route now goes through. `TODO/acquisition.md` carries the
  closure evidence and the residuals.
- ⛔ **The digest disposition is declared and never defaulted.** An adapter names
  either the asset carrying its vendor's checksums or the measured reason there
  is none, and the installer refuses a caller that names neither. `sha256sum -c`
  over the vendor's own document is the verifier.
- ⭐ **The claim guard's state directory is the user's own**, so the marker check
  that runs before every route needs no privilege either. ⚠ It is per-user, which
  is a narrower claim than `/var/lib` made, and `docs/capture-host.md` says so.
- ⛔ **Ten diagnostic probe steps are deleted from `capture-client.yml`.** They
  were the instrument of a dead end and four of them measured a privileged path
  this tree no longer takes.
- ⭐ **`check-adapters`' floor caught its own subject moving.** The fetches left
  the adapters, the rule answered `could not run` rather than reporting a clean
  tree over a path it had stopped reading, and its scope followed them.
- ⚠ **`check-rootless` is the new gate row**, a real row on the `sh` lane and a
  declared `n/a` on the PowerShell one. The gate is **40 checks**.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-06`, which carries
  the closure evidence and the residuals. No version bump and no deploy.

### 2026-09-16T09:44:52Z

- ⭐ **`ACQ-06`: ROOTLESS, PORTABLE CLIENT INSTALLATION, P0 and item 0**, by
  operator direction. Every install today needs `sudo` and writes to
  `/usr/local/bin`, so what it does depends on a host this project does not
  control - which is the one variable between an install that exits 0 in **one
  second** here and a step that thirteen bounds could not end on a runner.
  ⭐ Fetch, verify the digest, unpack into a prefix the current user already owns,
  answer with the path: no privilege, no package index, no host state, and the
  same behaviour on a runner, a session host and a contributor's laptop.
- ⚠ **The `package` route does not come with it and should not.** It is
  privileged by construction, so it cannot be portable. `E-ACQ-01` wants two
  INDEPENDENT routes and `release` plus `source` already are.
- ⛔ **A HANG IS A DEAD END, NOT A SUBJECT - now a rule.** `TODO/RULES.md` carries
  it and `AGENTS.md` makes it absolute 16: record the hang with its evidence and
  take a route that does not depend on the hanging thing. **Two bounds is the
  limit**; if a second does not end it, no further bound will. ⚠ It is a rule
  because it was broken expensively - thirteen bounds and a dozen dispatches on
  one step, and a cancelled job leaves no log and no artifact, so most of them
  measured nothing at all.
- ⚠ **The work order was stale and is rewritten.** Three items were closed
  entries still listed as work, the hang was item 0, and the counts predated
  `ACQ-06`. `INDEX.md`, `SUMMARY.md` and `PROGRESS.md` now read 66 / 22 open /
  1 in progress / 43 done, cross-checked by `check-project`, and
  `docs/history/RESUME.md`'s own list is aligned with it rather than contradicting
  it.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-06`. No version bump
  and no deploy.

### 2026-09-16T01:43:40Z

- ⭐ **THE CAPTURE HANG IS REPRODUCED ON A SESSION HOST AND REPAIRED.** Thirteen
  `capture-client` dispatches have been cancelled rather than answered, counted
  from the API. ⛔ The two defects hid behind a privilege difference this
  repository had never simulated locally;
  [`TODO/PROGRESS.md`](TODO/PROGRESS.md) states it.
- ⛔ **A bound inside `$( )` is not a bound.** A command substitution ends when
  its PIPE reaches end of file, not when the command exits, so a product that
  leaves one process behind blocks it forever with the bound sitting there having
  fired. `install-client` reached its adapter four times through a pipe, and
  **three of those calls carried no bound at all** - while the file's own header
  claimed every adapter call was bounded.
- ⭐ **`adapter_run` is the single call site now**, bounded, and it hands the
  adapter a FILE. Mutation-proved with one plant against both halves, and the
  control reproduced the cancelled dispatches' signature precisely; the two
  columns are in [`TODO/PROGRESS.md`](TODO/PROGRESS.md).
- ⛔ **An unprivileged bound cannot signal a root process tree.** Measured:
  `timeout -k 2 5` around `sudo -E sh -c 'sleep 120'` from a uid-1001 shell
  exited **124 on schedule** and left the root `sleep` alive with **PPID 1**,
  orphaned and past its KILL grace. ⭐ That predicts the number `CI-08` could not
  explain: run 21's job carried `timeout-minutes: 25` and ended at **thirty**.
- ⭐ **The bound is inside the `sudo` now**, the one place seven bounds had not
  been put. Driven as uid 1001 against a 600s install under a 20s step bound:
  **exit 124 at 21 seconds, `watchdog.log` carrying four samples and
  `holders.log` naming the survivor.** A hung step ends and leaves its timeline
  for the upload to collect.
- ⛔ **The workflow's bound arithmetic did not add up and now does.** A source
  lane's own inner bounds could sum to 1110 seconds under an outer bound of
  1080, so the killing would have happened outside with nothing to read. Each
  bound outlasts the one inside it: inner sum < `BIT_IDS_STEP_TIMEOUT` < the
  step's `timeout` < the job's `timeout-minutes`, which is 35 rather than 25.
- ⭐ **RUN 27 LOCALISED THE HANG IN ONE SECOND, USING NAMES RATHER THAN A BOUND.**
  Three named probes walk the release route's own operations, and all three
  passed before the wedge: the whole artifact fetched in 1s, staging (`chmod`,
  `mkdir`, a `cp` across filesystems into `/usr/local/bin`) in 0s, and an `exec`
  of the downloaded binary in 0s. ⛔ The vendor's endpoint, the network, the
  filesystem and the binary are ruled out by measurement; *Install the client*
  performs those same operations and still never returns.
- ⛔ **RUN 29: ALL SEVEN PROBES PASSED IN FOUR SECONDS AND THE STEP STILL WEDGED.**
  Every operation *Install the client* performs is now measured individually - the
  fetch, the staging, the exec, the claim guard, a detached `sudo` launch, the
  rule-12 scan over the fourteen-megabyte artifact, and the `/proc` walk. ⭐ Each
  candidate is excluded on its own, so no further probe of a component can reach
  it; what is left is the composition.
- ⭐ **A STEP-LEVEL `timeout-minutes` ON THE INSTALL STEP**, which is the runner's
  bound rather than a twelfth shell bound. A step that exceeds it is marked
  **FAILED** rather than cancelling the job, so the `if: always()` upload runs and
  the timeline ships; and a failed install skips *Cut the route*, so the host
  still has the network that upload needs. ⚠ It was measured once not to fire, on
  a `uses:` step; this is a `run:` step, which the runner supervises directly.
- ⭐ **FOUR MORE NAMED PROBES, one per remaining candidate**: the claim guard, a
  detached `sudo` launch, the rule-12 scan over the real fourteen-megabyte
  artifact, and the `/proc` holder walk. The scan probe carries
  `install-client`'s pattern verbatim, compared programmatically rather than by
  eye.
- ⛔ **DRIVING THOSE BODIES FOUND A DEFECT THE READING DID NOT.** GitHub runs a
  `run:` block as `bash -e` and `grep` exits **1 on no match** - the ordinary
  outcome for a scan that finds no secret - so a bare `_rc=$?` on the next line is
  never reached and the trailing `true` never runs. ⚠ Two of the four probes would
  have **failed the capture** they were written to be harmless to. ⭐ A trailing
  `true` is not what makes a block harmless; reaching it is.
  `docs/conventions/shell.md` section 9 states the rule.
- ⭐ **The `rusqlite` question is settled by the operator: the answer is no.**
  Pull request 1 is closed unmerged and the pin is 0.37.0 in both the manifest
  and the lockfile, read back. It moves to *Settled decisions*, and the three
  pages that called it open are amended. ⚠ Amended 2026-09-29: dependabot, not
  the decision, closed pull request 1, and pull request 2 re-offered the bump;
  the entry of 2026-09-29T01:13:49Z carries it.
- ⛔ **RUN 28 EXCLUDED THE `ps` AND REPRODUCED THE LOCALISATION.** It carries the
  bound and hung anyway - twenty-five minutes, past the loop's own 780s and the
  outer 900s - so the watchdog's `ps` is not the wedge. ⭐ Its three probes
  completed inside the same second, so two independent runs agree that the
  route's operations are instantaneous while the step performing them does not
  return. ⭐ The next instrument is the same one a level deeper: split
  `install-step.sh`'s work into two named steps.
- ⛔ **`ps -e` WAS THE LAST UNBOUNDED COMMAND IN THE STEP'S OWN SHELL.** It reads
  `/proc` for every process, so the watchdog could block on the very condition it
  was written to record, and the deadline two lines above it would never be
  reached. It is bounded now, and a headed, empty sample is itself a measurement.
  ⚠ This changelog's earlier claim that the loop "only sleeps and compares two
  integers" was written without re-reading the loop; it is corrected in the entry.
- ⛔ **THE `ps` TIMELINE WAS NOT OBTAINED.** Runs 24, 25 and 26 each hung in
  *Install the client* and each uploaded **zero** artifacts, read back from the
  API. ⭐ Run 26 is the sharpest measurement the entry has: three independent
  endings, one needing no signal to reach anything, and all three passed without
  the step ending. ⛔ That reframes the question from a bound that fails to fire
  to a step that is not executing, which no bound repairs;
  [`TODO/PROGRESS.md`](TODO/PROGRESS.md) states the reasoning. ⭐ The next
  instrument is step names rather than a twelfth bound.
- ⛔ **THE CAUSE IS NOT IN THIS REPOSITORY, AND ONE COMMAND SETTLES IT.** The
  install path was **byte-identical** across the boundary between a six-second
  install and a thirty-minute hang; the recorded lead about a commit window is
  refuted by the tree, which shows the window moved files that step never reads.
  ⚠ What is left is the runner image, the vendor's endpoint, or the network.
- ⭐ **`install-step.sh` stops waiting on its own schedule**, detaching the
  install with `setsid` and never `wait`ing on a process that may not die. Driven
  as uid 1001 against an install that **ignores TERM**: the step returned 124 in
  **21 seconds** against a job timeout of 35 minutes, with the timeline intact and
  `holders.log` naming the survivors.
- ⛔ **RUN 24 REFUTED BOTH BOUNDS AND NAMED THE OTHER HALF.** The first dispatch
  after the repair hung identically - *Install the client* still open twenty
  minutes in, past its privileged 780s bound and its outer 900s one - taking the
  tally of bounds measured not to end this step to **ten**. ⭐ Ten bounds failing
  the same way is one problem: a step ends when its command has exited **and**
  its output pipe has reached end of file, and every bound ever added acts on the
  first condition only.
- ⭐ **So the repair stopped being a bound.** The step body hands
  `install-step.sh` and its whole subtree a file and cats it afterwards, so the
  step's pipe is held by the step's own shell and nothing else. ⚠ The workflow
  had claimed that guarantee in a comment while holding it one level too low.
  The leaked-descriptor control moved up with it, to where the pipe actually is.
- ⚠ **`report-holders.sh` is aimed at the wrong descriptor**, which is why
  `holders.log` reported `0 holder(s)` on every green run: the file that keeps a
  step open is the step's own stdout, and nothing has ever asked about it.
- ⚠ **A residual is filed rather than rushed**: nested `timeout`s each create
  their own process group, so the privileged bound does not reach the adapter's,
  and an orphan survives. It does not block the deliverable and the fix changes
  a mutation-proven guard's signal semantics, so it is its own unit.
- ⚠ **A hypothesis was refuted in forty seconds rather than by a dispatch**: the
  watchdog's `kill -0` was read as failing across the privilege boundary, and
  measurement showed `$!` is the `sudo` process, which keeps **ruid 1001** and is
  signallable. The timeline is written on a runner.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-15T16:04:12Z

- ⛔ **`capture-client` runs 21 and 22 both hung thirty minutes in *Install the
  client*** on a lane that took six seconds on runs 19 and 20, and were
  cancelled with no log and no artifact. Three more bounds were measured not to
  fire, taking `CI-08`'s tally from four to seven.
- ⭐ **Every adapter fetch is bounded, and `bit-check check-adapters` is the
  rule.** Four adapters ran `curl -fsSL --retry 2` with no time limit while
  `docs/conventions/shell.md` section 9 stated the rule. The gate is **39
  checks**.
- ⛔ **The door sweep found the same defect one directory away, twice.**
  `provision.sh` is bounded in the same change; `mine-repo`'s pair is NOT, and
  the attempt was reverted: `timeout` is a PAUSE on Windows, so its two halves
  need two idioms. ⭐ **Superseded 2026-09-17**: they need none. Git's own
  `http.lowSpeedLimit` and `http.lowSpeedTime` are one spelling on every
  platform, and the premise that a bound means a wrapper was what was wrong.
- ⛔ **AND THE LOCATED CAUSE WAS REFUTED BY THE NEXT RUN.** Run 23 carries
  `--max-time 300` on that fetch and its release install ran past twelve
  minutes, so the step is not waiting there. The bound is kept on its own terms
  and the record says what it did not buy.
- ⚠ **A control run separates the commit from the clock**: a re-run at
  `95e90f5` installed in six seconds, so the hang correlates with this session's
  commits; the two hangs are also contiguous in time.
- ⚠ **An optimisation was measured and reverted.** Running most
  `check-capture-client` cases at one session left it at 134 seconds, so the
  sessions default is not what grew it.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-15T14:42:24Z

- ⭐ **Run 20 gave two announces from a stock build**, `started` and `stopped`,
  which is the first time this project has had two in one capture. ⛔ Both carry
  the same peer ID, so the tail is stable within a session on that surface.
- ⭐ **`capture-client.sh --sessions` defaults to two.** A value the build stores
  and one it regenerates read identically in one session, which is what
  `SamplingPlan::restarts` says; a second session is what separates them.
- ⛔ **The assembler attributes a sample to its session, read out of the
  announce.** `event=started` delimits a run, so the record does not have to
  trust the runner's count.
- ⛔ **The record cannot say which lifetime a tail has.** `classify` computes one
  per span and `field_state` drops it, so `PatternedValue` carries a tiling and
  no lifetime. `check-assemble` asserts the absence so the gap stays measured.
- ⚠ **A loop variable named `TARGET` was the adapter's target**, and put
  `target=16` in an attestation. The shell twin of the `[switch]$Marker`
  collision, found by reading the document back.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-04`. No version bump and
  no deploy.

### 2026-09-15T14:08:32Z

- ⛔ **`capture-client` run 19 refuted both recorded readings.** Both repairs are
  on the wire - `offered-interval 15`, and connections 2 and 3 offered
  `bit-ids-fixture-0001` and `-0002` - and `aria2-next` announced once and
  answered one of the two connections anyway.
- ⭐ **The client is stopped inside the observer's window now**, so a `stopped`
  announce is heard rather than sent to a tracker that has gone. The negative
  control is the previous script: the same adapter reports `announces=1` where
  this one reports 2.
- ⚠ **`stopped_within_window` and `stopped_after` are attestation fields**, the
  verdict and the measurement it is read from. A start that outlives the window
  is recorded rather than refused.
- ⛔ **A plant survived because the assertion read its own parameter.** The
  lateness case compared against the margin out of the attestation, so a margin
  set to the whole run moved the threshold with it. It compares against the
  deadline now.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-04` and `OBS-02`. No
  version bump and no deploy.

### 2026-09-15T12:34:13Z

- ⭐ **The announce interval is derived from the deadline.** It was a literal 60
  under a 45-second deadline, so no re-announce was ever due and every
  `tracker_http/*` field rested on one sample. `TrackerResponse::within` is the
  one derivation, floored at five seconds and capped at the default's own 1800.
- ⛔ **`capture-client.sh` refuses an interval longer than the run**, reading
  both numbers out of the observer's own log rather than recomputing either.
  ⚠ Equality is permitted: below fifteen seconds the floor is the answer.
- ⛔ **A plant survived `cargo test` and was refused by `check-capture-client`.**
  A guard proved only from outside the crate that owns it is one that crate
  leaves unproven, so the case was added rather than the plant counted.
- ⚠ **The door sweep found the adapter's `stop` running after the deadline**, so
  a `stopped` announce reaches a tracker that has gone. Filed as its own unit.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-02`. No version bump and
  no deploy.

### 2026-09-15T12:05:20Z

- ⭐ **The observer offers a different peer ID on every connection.**
  `PeerWire::opening()` is deleted rather than fixed: it answered the same bytes
  however often it was called, so two dials offered one peer.
  `PeerWire::present()` allocates, and `dialling` takes the value it returned.
- ⛔ **One counter serves both roles**, because a dial takes its ordinal before
  the connection exists and an accept takes one when it answers.
- ⚠ **`is-observer-peer-id` compared against one twenty-byte value** and would
  have answered *not mine* about every connection after the first.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-04`. No version bump and
  no deploy.

### 2026-09-15T08:09:05Z

- ⛔ **`capture-client` run 18 is green on both lanes and the refusal MOVED.**
  Each lane opened two peer connections, `aria2-next` answered the handshake on
  one of them and sent nothing on the other, and announced once - so every field
  still rests on one sample. ⭐ The pair is refused by `E-PUB-03` rather than
  `E-PUB-04`: a comparable capture that conflicts, where before there was none.
- ⚠ **Two repairs are located precisely**, in
  [`TODO/observer.md`](TODO/observer.md) under `OBS-04`: one about what the lab
  offers on a second connection, one about when a re-announce falls due.
- ⛔ **The door sweep found `validate-profile` reading as a permanent refusal.**
  It asks the one-document question correctly, and `E-PUB-04` is not a property
  of one document; it says so on its own line now.
- ⛔ **Two mutation plants survived, both the same shape**: nothing varied the
  dial address, and no fixture held a request without a `peer_id`. Both repairs
  are fixtures rather than readings, and both plants are refused now.
- ⚠ **The claim audit caught two numbers**: the gate is 38 checks and 127
  seconds, not 37 and 125; and a case count quoted for `check-assemble` moved
  from 26 to 32 inside one session, so it is gone rather than corrected.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-01` and `CI-09`. No version bump and
  no deploy.

### 2026-09-15T07:57:25Z

- ⭐ **A gate row could ask the wrong question and stay green, and now cannot.**
  `check-no-secrets` runs twice and the second row exists only because
  `--public` is a different question; a runner that stopped passing the flag
  kept a green row verifying nothing, because the default question also holds on
  a clean tree. `check-public-row` reads the flag out of BOTH runners and runs
  it, and the binary's own `"public_rules"` is the verdict. The gate is **38
  checks**.
- ⛔ **A harness that spelled the flag itself could not have caught it**, which
  is why the flags are read out of the runners - the rule `check-workflow.sh` is
  built on, applied to the gate's own queue.
- ⚠ **Its first draft built two `awk` programs out of shell strings** and lost
  its quoting: one half died with a syntax error, the other matched the row's
  label rather than its invocation. Parameter expansion crosses no boundary.
- ⚠ **And editing the `.ps1` with a text-mode read collapsed its line endings**
  to `w/mixed` under `attr/text eol=crlf`, which `git diff` does not show. Redone
  on bytes.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-15T07:46:42Z

- ⭐ **A RECORD HAS BEEN PRODUCED PUBLISHABLE.** Two captures whose lanes
  installed different digests assembled into two records, each stating a
  pattern - a fixed eight-byte prefix and a varying twelve over two samples -
  with `classify_across` answering `build_equivalent` and `publishable` holding
  for both, exit 0. Both of the refusals the work order named are gone.
- ⛔ **What it measured is a stub**, written to announce twice and accept two
  peer connections. The path is established and the product is not.
- ⭐ **`Lab::dial_again` opens another connection to an endpoint the lab already
  dialled**, so the segments land in the same transcript with their own
  connection identifiers and nothing downstream changed. `client-capture` takes
  two by default, and a refused re-dial ends the dialling rather than the run.
- ⛔ **A mutation plant survived and found a real gap**: the address comparison
  in `dial_again` could be removed with every case still green, because nothing
  varied the address - so a re-dial to somewhere else would have filed segments
  under an endpoint naming a different peer. A case varies it now.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-04`. No version bump and
  no deploy.

### 2026-09-15T07:34:08Z

- ⛔ **The assembler took the first segment of each transcript and called it
  `samples: 1`.** A transcript records a segment per connection, so a build that
  announced twice had already put two peer IDs in the evidence and the record
  threw one away. Every `from_target` segment is a sample now, under the plan the
  transcript shows.
- ⭐ **Two connections per lane turn a per-connection tail into a pattern both
  lanes share**, so a pair that was `divergent` reaches `build_equivalent` and
  both records publish - measured on the assembler's own lanes, where all four
  tails differ and what agrees is the shape.
- ⛔ **Sampling broke corroboration through a second door and driving it is what
  found that.** The observer's seen value was its bytes for `constant` and
  `out_of_scope` for everything else, so a field became uncorroborated the
  moment it became `patterned` and `E-PUB-02` refused the record. Corroboration
  is per observation, not per state.
- ⚠ **One dial is what is left.** `client-capture` connects to the peer surface
  once, so `peer_wire/*` rests on one sample on a real run and two lanes still
  disagree there.
- ⚠ The harness wrote one transcript segment per lane, so its fixtures had
  exactly one sample of every field and none of this was reachable by a case.
- Record: [`TODO/schema.md`](TODO/schema.md), `SCHEMA-04`. No version bump and
  no deploy.

### 2026-09-15T07:22:37Z

- ⭐ **`E-PUB-04` is closed, and it could not be closed by any capture.** A
  record whose two routes installed different bytes - every release-against-
  source pair there is - was refused however many captures existed:
  `equivalence::classify` reaches `byte_identical` or `unresolved` and never
  `build_equivalent`, `build_equivalent` comes only from `classify_across`,
  which takes two records, and every publication gate asked the per-record
  question one record at a time.
- ⭐ **`agreement::publishable_among` is the same gate asked with the store.**
  `routes_publishable` had always said an unresolved record needs a second
  capture through the other route; nothing could act on it, because that capture
  is a different record.
- ⛔ **The door sweep found a fourth caller holding both records and asking the
  one-record question.** `assemble-capture` reported publishability inside its
  per-lane loop, then compared the pair two statements later. It asks once every
  lane is written now.
- ⛔ **Every comparable sibling is read, not the first that agrees**, and a
  store that holds one agreeing and one conflicting capture refuses the record
  with `E-PUB-03` rather than `E-PUB-04` - a conflict is a different finding
  from an absence of evidence.
- ⚠ **A mutation pass reported five refusals over nothing**, because it named a
  test target that does not exist and read the non-zero exit as a refusal. It
  prints its control first now. A harness exit is *could not run*, never
  *refused*.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`. No version bump and no deploy.

### 2026-09-15T07:08:37Z

- ⭐ **The sampling classifier reaches a record now.** `SCHEMA-04` closed with a
  residual saying nothing that writes a record calls it, and
  `sampling::field_state` is the join it named: samples and the run's plan in,
  `constant`, `patterned` or `variable` out.
- ⛔ **That residual is why nothing measured can publish.** Run 17's records
  spelled every field `constant` with one sample, a peer ID's tail is
  regenerated per connection, so the two lanes' constants conflicted and
  `classify_across` answered `divergent`. Recorded from four restarts each they
  are one pattern, the comparison answers `build_equivalent`, and both records
  pass `publishable`.
- ⛔ **A run is fixed when the BYTES match, derived per OFFSET, and a test found
  both halves of that.** The classifier merges adjacent offsets by lifetime and
  `unknown` covers two different facts, so one span holds bytes that moved
  beside bytes that did not; and reading the lifetime rather than the bytes
  reports a value that changed as fixed.
- ⛔ **`E-PUB-04` is structural, and the closer the record named does not
  exist.** A capture observes one route, `classify` can never answer
  `build_equivalent`, and the per-record gate never sees the sibling. Filed with
  the store-level rule that would close it rather than written.
- ⚠ **`cargo test` stops at the first failing binary**, so a mutation pass that
  counts refusals without `--no-fail-fast` counts the binaries that ran. One
  plant read as 1 case and is 6.
- Record: [`TODO/schema.md`](TODO/schema.md), `SCHEMA-04`. No version bump and
  no deploy.

### 2026-09-15T06:33:43Z

- ⛔ **`check-docs` shipped a Windows-only defect and CI run 132 caught it.** The
  port derived a repo-relative directory with `filepath.Dir`, which is the HOST's
  separator, while every path comes from `git ls-files` as forward slashes and the
  orphan set is keyed on that spelling. On Windows it answered `docs\methodology`,
  every joined key missed, and the lane reported **every page an orphan**.
- ⚠ **Green on the Linux lane and green on `check-gate.ps1` run on Linux**, because
  Go's `filepath` separator is the host's. Running the PowerShell lane here does
  not test Windows; only the Windows lane can.
- ⭐ `path.Dir` is slash-based and is what the keys need; `filepath.Join` stays
  where the code touches the filesystem.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-15T04:18:40Z

- ⛔ **`check-ignores` passed over nothing and now has a floor.** With its
  specimen list emptied it answered *every one of the 0 credential shapes is also
  ignored* and exited 0 - the shape `check-licences` refuses in its register,
  reproduced in a new rule by the session that wrote both. A floor of 12 refuses
  a list too small to be real and answers **2**, because an empty list means
  nothing was verified rather than that a rule was broken.
- ⛔ **The door sweep found citations to files that no longer exist**: five
  markdown links whose label named a deleted script while the target pointed at
  the Go file, `FOUND-04`'s Acceptance naming two deleted commands, and three
  comments citing a deleted header. All repointed; the dated closure evidence is
  left as it was.
- ⛔ **The claim audit caught a number that was decremented rather than
  measured.** The remaining twin pairs' PowerShell halves were carried at 12.3
  seconds and stepped down to 11 and 10 as pairs left; timed, they are **5.3**.
  `check-twins` is **10.0 seconds**, not the 15 the record carried.
- ⚠ **And `docs/history/RESUME.md` exceeded the marker density ceiling** at 31
  per 100 against 30, which is verbosity measured rather than argued. Thinned to
  27.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-01`. No version bump and no deploy.

### 2026-09-15T04:07:40Z

- ⛔ **`rusqlite` 0.40.2 was taken and then REVERTED to 0.37.0.** `PUB-05` had
  already measured both and chosen 0.37.0, for thirteen fewer locked packages;
  taking the bump to make a dependabot pull request green overrode a recorded
  judgement with a notification. The lockfile and manifest are byte-identical to
  their previous state.
- ⚠ **The pull request stays red and that is the licence register working.**
  [`TODO/PROGRESS.md`](TODO/PROGRESS.md) carries it as an open operator decision,
  with the one fact the original decision did not have: the two versions vendor
  different SQLite releases.
- ⭐ **Documentation amended in place rather than appended to.** The session's
  narrative was compressed out of `TODO/ci.md`, `TODO/acquisition.md`,
  `TODO/foundation.md` and `TODO/PROGRESS.md`, leaving current truth,
  measurements and residuals; `docs/history/RESUME.md` was rewritten as one
  coherent handoff. Stale counts corrected against `bit-check --rows` and
  `check-twins`: nine ported rules, five twin file pairs, five rows.
- Record: [`TODO/foundation.md`](TODO/foundation.md), `FOUND-04`. No version bump
  and no deploy.

### 2026-09-15T03:44:22Z

- ⭐ **`check-docs` is in the Go binary and both shell halves are deleted**,
  taking it to nine checks and `check-twins` to **five file pairs and five rows**.
- ⛔ **This is the pair that justified the pre-deletion comparison.** `--compare`
  found the two shell halves DISAGREEING on their first run together: the `sh`
  half read links with two awk programs where only the broken-link one stripped
  inline code spans, so a page cited only inside backticks was an orphan to the
  PowerShell twin and not to it. ⚠ No page in this tree is cited that way, so
  `check-twins` saw them agree on every run for as long as both existed.
- ⭐ **The twin is correct and the `sh` half was fixed to match**, in the same
  change: a code span is not a hyperlink, so a reader following links never
  arrives. A verdict changed deliberately and recorded, which is what the port
  contract exists to tell apart from one changed by accident.
- ⛔ **The port reproduced a defect already written down here.**
  `docs/conventions/forbidden-patterns.md` records a GLOBAL replace collapsing
  `a/../` where `[^/]+` also matches `..`; the first Go spelling was the same and
  was green over the tree and fourteen cases, because it is only wrong past two
  levels. ⭐ The hand-rolled collapse is deleted and `path.Join` does it.
- ⚠ **The four-level case cannot refute that defect**, which was established by
  planting rather than by reading; it is kept as a regression guard with its
  label saying so. [`TODO/ci.md`](TODO/ci.md) carries why.
- ⚠ **A harness that plants a pattern cannot spell it, and neither can the comment
  that says so** - the placeholder needle tripped `check-placeholders`, and then
  the comment explaining it did the same thing again.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-15T03:21:16Z

- ⭐ **`rusqlite` 0.37.0 to 0.40.2, with the licence register updated in the same
  change**, which is what turns a dependency bump into a decision somebody
  recorded. A dependabot pull request had been red on every job since 2026-09-11
  for exactly this: 18 locked packages with no register row and 5 register rows
  naming no locked package.
- ⛔ **The surface grew in the lockfile and not in the build.** `rusqlite` 0.40
  brings a WASM backend chain; measured with `cargo tree -e normal --target ...`,
  `x86_64-unknown-linux-gnu` pulls `rusqlite` and `libsqlite3-sys` and nothing
  else new, and the nine others appear only under `wasm32-unknown-unknown`. They
  still need rows: `Cargo.lock` is what a build fetches from, and a target gate
  is not a licence.
- ⭐ **Every new row's licence was read from that crate's own manifest at that
  version** and the rows were rebuilt from `Cargo.lock` rather than typed. ⚠
  `foldhash` is `Zlib`, not the `MIT OR Apache-2.0` the rest of the batch shares,
  which is why reading beats assuming.
- ⛔ **`libsqlite3-sys`' notice was re-read, not carried.** It said the crate
  vendors SQLite 3.50.2; 0.38.2 vendors **3.53.2**, read out of
  `#define SQLITE_VERSION` in the packaged amalgamation.
- Record: [`TODO/foundation.md`](TODO/foundation.md), `FOUND-04`. No version bump
  and no deploy.

### 2026-09-15T02:48:33Z

- ⭐ **`capture-client` run 17 produced a record with nothing recovered from a
  log.** Its source lane carries its own resolution, both lanes are green, and
  `assemble-capture` wrote two `Profile`s and eight raw evidence files, exit 0.
  Read back out of the JSON: two routes with two genuinely different resolvers,
  four observations, two connectors whose corroboration agrees byte for byte.
- ⛔ **The record is not in the tree, nothing is published, and both records are
  provisional**, so the settled decision that a green capture publishes
  automatically did not fire and could not have.
- ⛔ **The door sweep found the same mistake behind three more doors.** Five
  refusals in the assembler named `release/resolution.txt` as the file they read;
  one was fixed and the rest were left. All five name what was actually read now.
- ⛔ **The claim audit found two stale counts**: the binary carries **eight**
  checks and two documents still said six and seven.
- ⛔ **And `check-no-secrets --public` refused this very entry's source.** Writing
  a user agent out as thirty-four hex digits is the shape of a fingerprint; the
  citation is narrowed, never the rule, which is what the refusal message asks
  for.
- ⚠ **Filed, not fixed**: a gate row could stop passing `--public` and stay green,
  because the default question also passes on a clean tree.
  [`TODO/ci.md`](TODO/ci.md) carries the acceptance command.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`. No version bump and no deploy.

### 2026-09-15T02:32:27Z

- ⭐ **`assemble-capture` wrote TWO `Profile`s from `capture-client` run 16's
  artifacts and exited 0.** Both lanes green, both `measured_build=2.7.5`,
  different installed digests, and every one of run 14's four refusals gone.
- ⛔ **Both records are `provisional, not publishable`, correctly**: `E-PUB-04`
  because only one of the two installs was put on the wire, so equal version
  labels are all that connects them; and `classify_across: divergent` because
  the peer-ID tails differ per connection. ⭐ That makes `BuildEquivalent` being
  unreachable through this path a measurement rather than an argument, and it
  means nothing was published.
- ⛔ **Two scope defects stood between the run and the records, and both are the
  same shape as the day's earlier one.** The source route's resolution was never
  uploaded - the artifact path list named `release/` alone - and the assembler
  looked for every lane's resolution at a literal `release/resolution.txt`. Both
  lists were complete for exactly as long as the source lane resolved through
  the release listing, which is what `E-ACQ-07` refused run 14 for.
- ⛔ **And the origin field differs by route kind.** A release resolution names
  its asset; a source resolution has no asset and names the clone URL. Asking
  every route for `asset_url` refused a source lane for a field its document is
  not supposed to carry. ⚠ The refusal also named the wrong route kind - *a
  package route has no origin URL* - and now names the route and the path it
  looked for.
- ⛔ **Twenty-four `check-assemble` cases passed over a shape no source lane has
  ever had**: the harness wrote a release resolution for every lane. A source
  lane is now built the way `resolve-source.sh` really writes one, and with that
  fixture alone - no new case - dropping either half of the fix fails 8 and 5
  cases respectively.
- ⚠ **The assembly was a REHEARSAL and says so**: run 16 uploaded no source
  resolution, so it was recovered from the job log where the step `cat`s it.
  Every other byte is run 16's, but a dispatch producing a record is unproved.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`. No version bump and no deploy.

### 2026-09-15T02:08:43Z

- ⭐ **`capture-client` run 15 dispatched, and three of run 14's four refusals are
  repaired ON A RUNNER**: each lane resolved through its own index, the install
  record carries `package=elf-binary`, and the attestation declares
  `connectors=cpython-stdlib` with a connector report in the bundle - which with
  the observer is the two `E-CAP-01` needs. Run 14 carried no `connectors` key at
  all, which is the whole of why it was refused as declaring one.
- ⛔ **The source lane failed, on a guard whose scope was wider than the thing it
  guards.** `install-client`'s rule-12 scan read every file at every depth of the
  install workdir, while the artifact takes named files from that directory's top
  level only - so the scan refused vendored third-party test certificates that
  nothing can ship. [`TODO/acquisition.md`](TODO/acquisition.md) carries the two
  scopes and why they diverged.
- ⭐ **Both halves of the rule are fixed** - the name scan and the value scan -
  because `grep -r` descends exactly as `find` did, and fixing one would be the
  gate-on-one-path shape the guard was added to close. ⚠ Narrowing is not a
  weakening: rule 12 is about what REACHES a remote, and the scan still reads
  every file at the level that ships.
- ⚠ **One repair uncovered the next.** A `release` route installs a binary and
  never clones, so the shape was invisible for fifteen dispatches and appeared
  the moment `E-ACQ-07`'s repair made the source lane a real source lane.
- ⛔ **Two claims in `assemble-capture`'s own documentation were false** and are
  amended: *no capture has yet written that key* and *no capture has written
  one*. Run 15 wrote both.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-03`. No version bump
  and no deploy.

### 2026-09-15T01:46:22Z

- ⛔ **A second rule this repository said it had and did not.** `.gitignore`'s own
  header claims a name is listed *before the files exist, so one can never be
  staged by accident*; `check-no-secrets` refuses a tracked credential file by
  name. Two defences, and **nothing compared their lists**.
- ⚠ **`*.jks` and `id_ecdsa` were in the refusal list and in neither ignore
  line**, so a Java keystore and an ECDSA private key were takeable by
  `git add -A` with the gate as the only thing in front of them. Both lines added.
- ⭐ **`check-ignores` is the rule that keeps them together**, a gate row on both
  lanes. It asks `git check-ignore --no-index` rather than re-reading the ignore
  format, because a second parser would be a second answer to what git ignores;
  and every specimen is first asserted to be a name the refusal rule really
  refuses, so a shape narrowed out of it is a finding rather than a case that
  stops meaning anything.
- ⚠ **One plant reported NOT-PLANTED and that is a third status.** A `sed`
  expected `jks)$` where the source reads `jks)|id_rsa`, so nothing changed and
  the binary answered 0 - which reads as a surviving plant to anybody who has not
  diffed the file. Re-aimed, the guard refuses.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-01`. No version bump and no deploy.

### 2026-09-15T01:39:49Z

- ⭐ **`check-no-secrets` is in the Go binary and both shell halves are deleted**,
  taking it to seven checks and `check-twins` to **six file pairs and six rows**.
  ⚠ It was the two-row pair, so one deletion removed two rows: `--public` is a
  different question from the default run rather than a stricter one.
- ⛔ **The patterns were the easy half.** Two properties of the pipeline they sat
  in decide verdicts and both are carried; [`TODO/ci.md`](TODO/ci.md) names them
  under `CI-10`.
- ⭐ **Twenty cases over both halves before either was deleted**, all three
  implementations agreeing on the exit code and byte for byte on `--json`. Eight
  of them plant something that must be ACCEPTED, which is where a port fails.
- ⛔ **The callers moved with it**, and `check-defaults`' subject moved INSIDE
  the block that gates on the Go build rather than beside it - the one-gated-door
  shape that block's own comment records.
- ⚠ **Two counts in this tree disagreed and neither said which unit it was in.**
  `docs/history/RESUME.md` said eight pairs and then named seven files;
  `TODO/ci.md` said seven. Both were real - seven file pairs, eight comparison
  rows - and `sh scripts/common/check-twins.sh` is what reconciled them.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-10T09:01:53Z

- ⭐ **The matrix is proved on runners: run 124 is green on every job**, and the
  longest shard is **11.8 minutes** where run 122 was 32.6 and run 123 22.2. The
  whole run's wall clock is that shard; the Linux gate is 4.4 and Windows 2.5.
- ⛔ The acceptance bound came **down** to 20 from 45, which is the entry's Prove:
  a bound that was not raised to accommodate the work.
- ⭐ **Two more pairs ported and deleted**, `check-licences` and
  `check-placeholders`, taking the binary to six checks and `check-twins` to
  eight pairs. Proved first over 50 cases, all three implementations agreeing.
- ⛔ **`check-licences` had two callers and both moved with it.** `check-cache`
  asked an `.sh` half here and a `.ps1` half there which targets may be
  redistributed; both ask the one binary now. ⚠ That comparison is weak and says
  so: this register permits no target, so the two agree on an empty list. What
  holds it up is `check-cache`'s own 13 cases, green on both halves.
- ⛔ **A harness that plants a pattern cannot spell it.** Adding placeholder
  plants turned the clean tree red, because `check-placeholders` reads every
  tracked file and the harness is one. The needles are assembled with `printf`
  now, as the marker harness already did.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-10T08:41:18Z

- ⛔ **The previous entry's headline claim was measured on the wrong machine and
  CI refuted it.** It said deleting the twin layer bought nothing. On a runner it
  bought **10.4 minutes**: *Workflow acceptance* was 32.6 on run 122 and **22.2**
  on run 123, and the Linux gate 5.8 against 4.7.
- ⚠ Both numbers are real and describe different hosts. This session host has
  **four** processors and a hosted `ubuntu-24.04` runner has **two**, so a gate
  that runs its checks concurrently is CPU-bound there and not here.
  `docs/methodology/gate.md` already names it: local is not production. The
  original wording is kept in the entry rather than deleted.
- ⭐ **`check-workflow` shards.** `--shard i/N` selects by `index mod N`, `--units`
  lists and runs nothing, and the CI matrix is four runners.
- ⛔ **The controls are not sharded**, because a shard of plants alone goes green
  over a tree where every step is broken. Every shard runs all three.
- ⭐ **`scripts/ci/check-shards.sh`** asserts coverage and disjointness for every
  N from 1 to 6, that each shard names every control, that no variable crosses a
  unit boundary, and that six malformed selectors are refused.
- ⛔ **It found three real defects the moment it existed**, all in the sharding: a
  `--units` mode that ignored `--shard`, and two variables assigned in one unit
  and read in another. ⚠ The guard written for the second missed the third,
  because it saw one spelling of an assignment; it reads three now.
- ⭐ **Measured on this host: unsharded 1282 seconds and 97 cases; the longest of
  four shards 606.** That is 2.1x rather than 4x, and the control floor is the
  whole difference.
- ⭐ The four shards ran 136 cases against the unsharded 97; the difference of 39
  is exactly three extra copies of the 13 control cases, so no plant ran twice
  and none was dropped.
- ⚠ The acceptance job's bound comes **down** to 20 minutes from the 45 it was
  raised to. Raising a bound to fit growth is the masking this project refuses.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-10T07:18:57Z

- ⭐ **Four twin pairs are DELETED rather than translated.** `check-changelog`,
  `check-control-bytes`, `check-markers` and `check-one-home` are one Go binary;
  eight files are gone and the pairs are out of `check-twins`' list. Both lanes
  run the same program, so four rows that were an `sh` row and a hand-written
  twin are now one row each.
- ⚠ They left only after `check-bitcheck --compare` ran them against both deleted
  halves. `TODO/ci.md` carries that run and its counts.
- ⭐ **`check-twins` went from 69 seconds to 15.9.**
- ⛔ **AND THE GATE DID NOT GET FASTER, WHICH REFUTES `CI-10`'s OWN PREMISE.**
  The whole gate is 129.7 seconds against the 118-125 recorded before any of
  this: the concurrent batch lost 53 seconds and the gate lost none of them.
- ⛔ **Every gate member was then timed and the wall clock is elsewhere.**
  `check-capture-client` is **105 seconds** - `CI-01` recorded 47.9 - because it
  has grown to 111 cases while the number in the record stayed still. It and
  `check-capture` run after the batch, alone and on purpose, so the gate is
  `max(batch) + max(those two)`.
- ⚠ The port is still unconditional and still removes the drift class, and it is
  **not** what makes CI faster. Sharding is, and it is the next unit.
- ⚠ Seven pairs remain; the entry carries what they are worth in wall clock and
  why porting them now comes after sharding.
- ⛔ One defect introduced and caught while doing this: a Python text-mode rewrite
  left `check-docs.ps1` at `w/lf` under `attr/text eol=crlf`, which `git diff`
  shows nothing for. `git ls-files --eol` is what found it.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-10T06:40:47Z

- ⛔ **The twin layer's wall clock is ONE ROW, measured before porting anything.**
  Every pair timed unpiped on this host: the eleven `.sh` halves together are
  **7.4 seconds** and the eleven `.ps1` halves **96.1**, of which
  `check-control-bytes.ps1` alone is **67.9**. `check-workflow` runs the whole
  gate about nine times, so that is roughly ten minutes of every push.
- ⚠ The measurement does not make `CI-10` conditional and is not offered as
  though it did. It decides the ORDER: those two rows are ported first because
  they are the wall clock.
- ⭐ **`tools/check/` is the Go checking binary**, with an empty require list and
  therefore no `go.sum` - nothing is fetched at build time, so a build needs no
  network and `CI-04`'s dependency surface does not grow by a language.
  `check-control-bytes` and `check-markers` are in it.
- ⭐ **`scripts/common/check-bitcheck.sh` is one case list in two modes.** The
  default plants against the Go binary alone, **1.1 seconds for 18 cases**
  including building it, and is the permanent gate row. `--compare` additionally
  runs both shell halves and refuses any difference in exit code or `--json`.
- ⭐ **The pre-deletion proof ran: 18 cases, 18 passed, all three implementations
  agreeing byte for byte.** Six of the cases plant something that must be
  ACCEPTED, because over-strictness is where a port fails.
- ⛔ **And the first mutation pass reproduced `check-twins`' own blind spot.**
  Adding DEL to the Go control class SURVIVED - nothing in the tree and nothing
  planted carried that byte - while three other plants were refused. The repair
  is a fixture, not a reading, and the same plant is refused with it.
- ⛔ Nothing is deleted yet and the gate is one row longer, at 36. The proof
  precedes the deletion.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`, now `IN_PROGRESS`. No version bump
  and no deploy.

### 2026-09-10T05:08:52Z

- ⛔ **`CI-10` is filed, it is FIRST in the work order, and it is not
  conditional.** Operator direction on 2026-09-10, recorded as given: port all
  the checking scripts to Go, port the slow CI parts to Go, and parallelise CI
  across multiple runners.
- ⚠ **The argument for it is this session.** *Workflow acceptance* was cancelled
  at its bound three times, two fixes were tried on the wrong cause, the bound
  was then raised to 45, and a fourth run was cancelled by this project's own
  push before it could prove even that.
- ⭐ **It deletes the twin layer rather than translating it.** One binary that
  runs on both platforms removes the drift `check-twins` exists to detect, and
  `CI-07`'s remaining class-A backlog stops being work at all.
- ⛔ **A ported check refuses exactly what its shell half refused**, over the same
  plants and with the same 0/1/2 vocabulary, and is compared against those halves
  before they are deleted.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-10`. No version bump and no deploy.

### 2026-09-10T04:31:42Z

- ⛔ **The first diagnosis was wrong and the job hit its bound again.** Moving
  `check-defaults` out of the gate saved about five minutes and did not bring
  *Workflow acceptance* back under 30; it was cancelled at 30m15s twice.
- ⭐ **Measured instead of guessed a second time:** `check-capture-client` is 105
  seconds against the 48 this repository's own gate comment records. `OBS-07`'s
  connector refusals were given cases and each one runs a whole capture - 57
  seconds the gate gained, multiplied by `check-workflow`'s ten gate runs into
  about nine and a half minutes.
- ⚠ **The bound is 45 and the growth is named rather than absorbed.** What bought
  those seconds is six refusals that had never fired; `CI-01`'s residual carries
  the real fix, which is sharding rather than more minutes.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-10T03:53:49Z

- ⛔ **A 28-second gate row cost the CI lane five minutes and its job was
  cancelled.** `check-workflow` runs the whole gate about ten times, so run 116's
  *Workflow acceptance* went from about 20 minutes to 30m15s and hit its bound.
  `check-defaults` is its own CI step now, the way `check-workflow` itself is.
- ⚠ **The file that predicted the cost put the row in the gate anyway.** Its own
  header said the 28 seconds are not free inside `check-workflow`; the number
  that was missing is the multiplier, and it is written down beside it now.
- ⭐ `check-shell` stays in the gate: it is two seconds and it is the row that
  stops a green local gate over a red lane.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-10T03:15:22Z

- ⛔ **The gate ran neither `shellcheck` nor `shfmt`, and the CI workflow's own
  comment said it did.** Run 113 failed on *Shell syntax and style* over one
  `SC2016`, on a commit whose local gate had reported 34 of 35 passing minutes
  earlier. That is a green local gate over a red lane, from a step nothing local
  covered.
- ⭐ **`check-shell` is a gate row now**, running the two tools exactly as CI
  does, as two rows because a red row has to say which refused - and a third
  that counts what was swept, since two clean tools over a `find` that stopped
  matching answer the same as over a clean tree.
- ⚠ `provision.sh` had installed both tools all along, for precisely this.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-10T03:05:58Z

- ⛔ **Six of the eight refusals in the capture runner's connector block had
  never fired.** The real connector answers correctly, so every case that drove
  it exercised the accepting path. A stub connector, in the stub adapter's shape,
  makes each one refuse - with a control first that the stub itself is
  acceptable.
- ⛔ **The credential scan added earlier today was itself on one of two paths.**
  `capture-client.yml` uploads the install workdir's logs as a second artifact,
  written by an adapter that clones, configures and builds. `install-client`
  carries the same two rules now, with a refusal and a near miss.
- ⛔ **A closure-evidence count said five where six cases were added.** Measured
  against the commit that added them.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-07`, and
  [`TODO/clients.md`](TODO/clients.md), `CLIENT-14`. No version bump and no
  deploy.

### 2026-09-10T02:28:31Z

- ⭐ **`check-catalogue` is a real row on the PowerShell lane**, the second
  class-A row to stop being declared, and it needed no new library function -
  which is `CI-07`'s sweep measured a second time. The entry carries the row
  counts.
- ⛔ **`check-twins` compares a VERDICT, so it cannot see a weakened assertion.**
  Five plants: removing a restore, expecting the wrong error code and dropping a
  case were all caught; replacing a passing case's assertion with `$true` was
  not, twice. A pair proves the two halves reach the same verdicts and says
  nothing about whether either reached them for a reason.
- ⛔ **An empty pipeline unrolls to `$null` under `Set-StrictMode`**, so the new
  twin died on the one case whose expected answer is an empty one. The outer
  `@()` is load-bearing, which `assert-disposable.ps1` already records.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-07`. No version bump and no deploy.

### 2026-09-10T02:16:47Z

- ⭐ **`docs/architecture.md` and `docs/capture-methodology.md` describe the
  connector that EXISTS**, not only the ones that are planned. The three in
  section 6's list put their own bytes on the wire and none of them is written;
  the one that is reads the bytes the lab already recorded.
- ⭐ **The acquisition section says what refuses a route that acquired nothing**,
  and that it closes the hole rather than the residual.
- ⛔ **Five commit subjects earlier today carry stamps that were computed rather
  than read**, each 10 to 43 minutes ahead of the commit git recorded, two of
  them in the future when written. The headings above are corrected to
  `git log`'s own values; the subjects are history on `main` and stay wrong.
  `docs/history/RESUME.md` names which.
- ⛔ **`check-project` refused this session's new sweep for composing a cargo
  output path**, on the one line that exists to talk about that very defect.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-07`. No version bump and
  no deploy.

### 2026-09-10T02:09:37Z

- ⭐ **`CI-08`'s sweep is an instrument rather than a reading.**
  `scripts/ci/check-defaults.sh` runs five checks under six environments and
  compares their machine-readable answers, so a default a check inherits is
  named by the variable that produced it whether or not anybody thought of it.
- ⛔ **Two controls, because "no difference" and "no experiment" look
  identical.** A probe that reads the variable must answer differently or the run
  exits 2, and a planted environment-sensitive answer is caught and named.
- ⛔ **One row is blind on this host and says so.** Replanting the historical
  `CARGO_TARGET_DIR` defect did NOT catch it: the tree already held a built
  example, so the defective path resolved to a stale binary.
- ⚠ **`IFS` and `umask` cannot be perturbed this way**, and are stated rather
  than dropped.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`. No version bump and no deploy.

### 2026-09-10T01:56:39Z

- ⭐ **`store-lib.ps1` exists**, which `CI-07`'s sweep named as the first step:
  every declared row whose subject is portable waits on ONE library rather than
  on fifteen translations.
- ⭐ **`check-cache` is a real row on the PowerShell lane**, the first class-A
  row to stop being declared. That lane went from 13 passed and 20 unavailable to
  14 and 19, over 34 rows both runners agree on.
- ⛔ **`check-twins` could only reach `common/`.** The first harness twin is in
  `acquisition/`, so a comparison scoped to one directory would have left it
  uncompared - the shape that file exists to refuse, in its own plumbing. Pair
  paths are relative to `scripts/` now.
- ⚠ **A clean tree proves nothing about a pair**, so three defects were planted
  in the new library one at a time and the halves compared on each: all three
  made them disagree.
- ⛔ **Three functions are deliberately absent from the library.** A function
  nothing calls is a function nobody knows works.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-07`. No version bump and no deploy.

### 2026-09-10T01:46:05Z

- ⛔ **Keeping a secret out of a capture artifact depended on each adapter
  remembering.** `capture-client` run 11 shipped `client/rpc-token`; the adapter
  was fixed to unlink it, and that fix needs `stop` to be reached and to succeed
  - while a non-zero `stop` is recorded rather than refused.
- ⭐ **The runner refuses at the choke point now**, on two rules: a file whose
  name says it holds a credential, and a secret-shaped member inside any text
  document it would ship. The second is the one no filename rule reaches, because
  an options dump the product answered with lands in a file called nothing in
  particular.
- ⚠ **A word boundary, not a substring**, with a near-miss control beside the two
  refusals: a rule that refused everything would pass both of them.
- Record: [`TODO/clients.md`](TODO/clients.md), `CLIENT-14`. No version bump and
  no deploy.

### 2026-09-10T01:28:06Z

- ⛔ **A route that ran and installed nothing reached the strongest verdict this
  project has.** `aria2` ships on `ubuntu-24.04`, so two routes on such a host
  declare two independent resolvers, agree on a version because it is ONE
  binary, and arrive at `byte_identical`. `assemble-capture` refuses a lane
  whose install record says `acquired=no` now, which is upstream of the
  comparison: such a route is not a second route, so no record is written.
- ⚠ **It closes the hole and not the residual.** `classify` still cannot see the
  field, because the route type has nowhere to put it.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-03`. No version bump
  and no deploy.

### 2026-09-10T01:24:17Z

- ⛔ **The client matrix was not pinned to the catalogue in both directions**,
  though it said it was. `check-project` carried a hardcoded list of seventeen
  ids, so a target added to the catalogue and forgotten in the matrix was caught
  by nothing, in either half.
- ⭐ **Both halves derive the target set from the catalogue now** and compare it
  against the matrix's own rows either way, refusing a set too small to be real
  because two empty sets agree perfectly. The list is deleted rather than
  extended.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-01`. No version bump and no deploy.

### 2026-09-10T01:08:02Z

- ⭐ **THERE IS A SECOND CONNECTOR, which is what absolute 2 asks for and what
  this project had never had.** `scripts/capture/connectors/cpython-stdlib.py`
  reads a capture bundle's raw transcripts and decodes them with implementations
  this project did not write: `json` over a document the lab serialises and
  parses by hand, `urllib.parse` over the announce's percent-encoding, and
  `http.client` over its headers.
- ⛔ **A one-connector capture cannot become a record**, so `capture-client` now
  requires `--connector` with no default and no fallback. A capture that warned
  and carried on would look exactly like a green run, which is how run 14's four
  artifacts came to be unusable without anything saying so.
- ⛔ **`+` is where two correct-looking readings differ.** A query-string parser
  decodes it as a space; an escaped peer ID's `+` is the byte `0x2b`. Both
  spellings are cases.
- ⭐ **A surface the bundle does not carry is `out_of_scope`, not `absent`.**
  Found by the driven pass: a capture whose target never accepted a peer
  connection writes no peer transcript at all, and the first version refused a
  perfectly good bundle over it.
- ⛔ **`assemble-capture` reported a written record without a publishability
  verdict.** It prints one, with every blocker, now; the entry has the finding.
- ⛔ **And a case in the new harness claimed more than it checked.** The
  uppercase-hex plant also changed the length, so the odd-length branch refused
  first and a connector planted to accept uppercase passed it. The plant
  preserves the length now and the refusal is two messages.
- ⛔ **`check-project` refused every `.py` while its message said "without an
  approved exception", and no way to approve one existed.** A rule nobody can
  comply with is the shape this repository calls a preference stated as a rule.
  Both halves now ask for the declaration instead, and four planted defects are
  refused with the twins agreeing character for character on each.
- Record: [`TODO/observer.md`](TODO/observer.md), `OBS-07`. No version bump and
  no deploy.

### 2026-09-09T19:57:55Z

- ⭐ **`CI-07`'s backlog is four kinds of row, not one**, measured by reading
  what each declared harness actually drives rather than the reason it declares.
  Only one kind is work that entry can do: the rows whose subject is portable
  Rust and whose only missing half is the harness, which is what makes
  `store-lib.ps1` the first step rather than fifteen translations.
- ⛔ **Several rows have an `sh` SUBJECT with no PowerShell half**, so a harness
  written for them would be a twin proving a script that does not exist:
  `publish-data.sh` behind `check-publish` and `check-access`, and the capture
  and resolution scripts behind the rest.
- ⛔ **And several are facts about the platform that close on nothing.**
  `check-examples` and `check-handbook` RUN a document's ```sh fenced blocks and
  there is no `sh` on a Windows runner; `check-gate-rows` runs the `sh` runner
  itself; `check-step-bodies` lifts a Linux-only workflow's bodies and already
  runs the `pwsh` ones; and `check-capture`'s subject has a PowerShell half the
  capture workflow exercises.
- ⭐ **Each row's declared reason now names its class and the event that would
  close it**, so the sweep lives in the runner rather than only in an entry.
  ⚠ No count of them is written in prose, for the reason that entry records
  twice: both previous counts went stale.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-07`. No version bump and no deploy.

### 2026-09-09T18:52:46Z

- ⭐ **The `source` route resolves its own tag**, from `git ls-remote --tags
  --refs` rather than from the release lane's listing. That closes the second of
  the four things `capture-client` run 14 could not supply: its two lanes read
  one index, and `E-ACQ-07` calls two routes sharing a resolver one route.
- ⛔ **The workflow argued for sharing it** - one resolver is how two lanes
  cannot land on different versions. ⭐ Absolute 4 answers that better: version
  equality is checked AFTER installation, on what each build reported when
  asked, so two resolutions that disagree are a vendor that moved between two
  reads and catching it is the point. Making them equal beforehand manufactures
  the agreement this project exists to measure.
- ⭐ **`sources::git_refs` is the second source reader**, and the source id now
  chooses the reader rather than a fourth argument doing it: one value says both
  what answered and how it is read. ⚠ Two sources of one format are told apart
  by their URLs, which is what a route records.
- ⚠ **Two things a refs source cannot do, stated rather than found later.** It
  carries no `prerelease` flag, so only the version text is left to catch a
  prerelease; and it carries no date, so a tag no scheme can order blocks the
  resolution rather than being released by `predates_selection`.
- ⛔ **A tag is not a commit.** `--refs` yields the tag object for an annotated
  tag, not the commit it points at, which is why `E-ACQ-06`'s object name still
  comes from the adapter's own `git rev-parse HEAD` after the clone.
- ⛔ **And the new resolver read the wrong tag**, caught by running it against a
  real resolution document before it shipped. It scanned forward from the
  `selected` verdict and took the next `tag` it met - and `tag` comes BEFORE
  `verdict` inside an entry, so it printed the FOLLOWING candidate's tag:
  `v2.7.4` over a resolution that had selected 2.7.5. It is the same reader
  defect `CI-09` records in `check-project`'s first artifact rule. The tag is
  remembered per entry now, with the entry boundary resetting it rather than a
  distance being assumed in either direction; the harness's control puts the
  winner last so a reader that took the first or the next entry fails it.
- ⚠ **One fact had two homes in one file**: `aria2-next.sh` spelled its
  repository as a literal in `describe` and again in its source route. The day
  one moved, the two routes would have acquired from different upstreams and
  every check here would have passed. The route asks `describe` now.
- ⭐ **And the line-endings row earned itself again.** Editing `check-gate.ps1`
  with a tool that emits LF left it `w/lf` under `attr/text eol=crlf`, and
  `check-project` refused it. That row was added on 2026-09-08 after the same
  thing happened and was caught only by an incidental `git commit` warning;
  this time a gate caught it, in the file the row was written for.
- Record: [`TODO/acquisition.md`](TODO/acquisition.md), `ACQ-02`;
  [`TODO/ci.md`](TODO/ci.md), `CI-09`. No version bump and no deploy.

### 2026-09-09T18:38:24Z

- ⭐ **An adapter now says how its route packaged the build**, which closes one
  of the four things `capture-client` run 14 could not supply. A successful
  `install <route> <workdir>` writes `<workdir>/package`, `install-client`
  copies it into the install record, and a route that installed without saying -
  or that said something which is not a slug - is refused at the install rather
  than at an assembly a dispatch later.
- ⛔ **It is per ROUTE, because the routes genuinely differ.** `qbittorrent`
  installs a `.deb` through its package route and an AppImage through its
  release route; `aria2` installs a `.deb` and a locally compiled binary;
  `aria2-next` installs a downloaded binary and a locally built one. A record
  calling any pair one form would say they delivered the same thing.
- ⚠ **`Build.package` is in the identity tuple `store::StoreKey` derives a path
  from**, so this was never cosmetic: a record filed under a guessed package
  files two packagings of one version at one name.
- ⛔ **The new guard found a third caller of `install-client` that neither
  updated harness covers.** `check-step-bodies` runs the *Install the client*
  step body against a stub adapter of its own, and four of its cases went red
  the moment a route that installs without declaring a packaging was refused.
  ⚠ That is the guard working and a door sweep arriving late: two stubs were
  updated and a third existed. It reproduced alone rather than only under a
  loaded gate, which is what separates it from that row's known flake.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`;
  [`scripts/capture/adapters/README.md`](scripts/capture/adapters/README.md) is
  the contract. No version bump and no deploy.

### 2026-09-09T17:26:57Z

- ⭐ **Something tried to assemble run 14 into records, and that is how four
  refusals were found.**
  [`assemble-capture`](crates/bit-ids-probe/examples/assemble-capture.rs) reads a
  lane's capture artifact and its install artifact and writes the `Profile` they
  support, deriving every field from a document rather than from a lane's name.
  `check-assemble.sh` proves it over
  synthetic lanes: 15 cases, and the control - two routes, two resolvers, two
  connectors, one identity on the wire - is accepted and reaches
  `build_equivalent`.
- ⛔ **Run 14 cannot become a record**, and the tool reports every reason in one
  run rather than the first it trips on. Both lanes' `release/resolution.txt`
  differ only in their timestamps, so `E-ACQ-07` calls them one route; every
  attestation declares one connector, which `E-CAP-01` refuses at the validity
  gate; nothing records how the artifact was packaged, and `package` is part of
  the identity tuple a store path is derived from; and nothing the capture path
  wrote carried the source route's commit, which `E-ACQ-06` needs.
- ⚠ **`classify_across` never ran on run 14** and could not - it takes two
  `Profile`s. What is measured is that the pair's shape is `Divergent`: a peer
  ID's tail is per-connection, a single capture can only state such a field as
  `constant` with one sample, and any disagreement on a shared field is
  `Divergent`. Both directions are cases in `check-assemble`.
- ⛔ **`platform`, `arch` and `package` are derived rather than written in.** The
  first draft of the assembler hardcoded `linux`, `x86-64` and `elf-binary`,
  which would have filed a Windows capture of one version at the Linux capture's
  path - the non-injective layout `store.rs` refuses at length. Found by a claim
  audit against the module's own "every field is derived" header.
- ⛔ **A capture declaring ONE connector is invalid, not merely unpublishable**,
  and three handoffs said otherwise. Measured by stripping the golden fixture two
  ways with each exit code read unpiped: two connectors declared and one
  observing each field is *valid* and `provisional, not publishable` with six
  `E-PUB-02` rows; one connector declared is `refused`, `invalid document`,
  `E-CAP-01`. `OBS-07` moved up the work order because of it.
- ⛔ **The peer ID differs between SURFACES inside one run.** Each bundle carries
  two transcripts and they disagree, so the twelve-byte tail is per-connection
  rather than per-run. Nothing had read the second transcript: an attestation
  records the tracker's value alone.
- ⭐ **Repaired here:** `aria2-next.sh`'s source route runs `git rev-parse HEAD`
  after its clone, and `install-client` records `source_commit` and refuses a
  `source` route whose commit is absent or abbreviated - so the gap fails at the
  install rather than at an assembly a dispatch later.
- ⭐ **A transcript can be read back.**
  [`parse_transcript_document`](crates/bit-ids-lab/src/evidence.rs) is the
  writer's inverse, beside it, refusing anything that writer does not emit.
  Nothing could read a bundle back before it.
- ⚠ **`aria2-next.sh` claimed its two routes "differ in resolver - the releases
  API against git refs".** The artifacts refute it: the source route is handed
  its tag. The comment is corrected rather than argued away.
- Record: [`TODO/ci.md`](TODO/ci.md), `CI-09`; [`TODO/clients.md`](TODO/clients.md),
  `CLIENT-14`. No version bump and no deploy.

### 2026-09-09T15:20:21Z

- ⭐ **The first two-route capture.** `capture-client` run 14 acquired
  `aria2-next` through its `release` and `source` routes on two hosts. Both
  report **2.7.5**, both `acquired=yes`, both `egress=closed`, and the installed
  binaries have **different digests** - one stable version and two provably
  different builds, which is what absolute 4 asks for and `ACQ-03` exists to
  classify. Both bundles verify with `sha256sum -c` outside the runs that wrote
  them.
- ⭐ **`aria2-next` has a second acquisition route**: a shallow clone of the
  resolved tag plus the project's own `cmake --preset default` build. Both lanes
  resolve through the SAME resolution, so the two routes cannot land on different
  versions. ⚠ Their independence is weaker than package-versus-vendor and the
  record says so: different resolver and delivery, same origin.
- ⭐ **The identity is the same from both builds**: `-qB5230-3SGS8~CB*gUf` and
  `-qB5230-*eQ2phy)!)RO`. Four captures of this target now agree on the
  eight-byte prefix and differ in every tail, and two of the four were built by
  different means - so the prefix is a property of the source rather than of the
  vendor's build pipeline.
- ⛔ **496 seconds against a 500-second bound is not a margin.** The same build
  takes 366 seconds on a session host, so a runner is about 1.35x slower and a
  locally measured build time is a lower bound, never an estimate. The inner
  bound is now 900 for `source` lanes and unchanged at 420 for every other; the
  step's own bound moved from 540 to 1080 so it stays the larger of the two, and
  `check-step-bodies` plants against that literal, so its plant string moved in
  the same change.
- ⚠ **Still not a `Profile`.** Nothing has assembled the two install records and
  two bundles into a record, `ACQ-03`'s comparison has not been run over the
  pair, and nothing is published.
- Record: [`TODO/clients.md`](TODO/clients.md), `CLIENT-14`. No version bump and
  no deploy.

### 2026-09-09T13:40:18Z

- ⭐ **A client capture reached the *Capture* step for the first time.**
  `capture-client` run 11 measured `aria2-next` 2.7.5 on a hosted runner: every
  step passed, in 2 minutes 1 second, with a one-second install. Ten earlier
  dispatches of `aria2` stopped inside *Install the client* and produced no log
  and no artifact.
- ⭐ **The evidence bundle verifies outside the run that wrote it.** Downloaded
  and checked with `sha256sum -c SHA256SUMS`: three files, all `OK`, exit 0.
- ⛔ **A stock `aria2-next` announces as qBittorrent.** The measured peer ID is
  `-qB5230-s2QbzYjt(LOQ`, whose first eight bytes are qBittorrent 5.2.3.0's
  prefix. Predicted from an RPC option read, then observed on the wire - and a
  peer-ID table would have filed this build under the wrong product.
- ⛔ **Run 11 found a secret in the artifact, and the gate structurally could
  not.** The adapter wrote its per-run RPC token into the workdir so `stop` could
  authenticate a shutdown, and `capture-client` packs that workdir, so the token
  shipped as `client/rpc-token`. `stop` now unlinks it before anything packs the
  directory and `start` writes it under `umask 077`; `adapters/README.md` carries
  the rule that a workdir IS the evidence bundle.
- ⚠ **The asset-less release was transient and the window is still real.** Run 11
  resolved `2.7.5`, the release that carried zero assets at `12:11:57Z`. The
  fixture and the harness case keep the refusal pinned whether or not the window
  is open today.
- ⭐ **The flaky `check-step-bodies` row is diagnosed and fixed.** It went red
  three times inside a gate and passed alone every time. The obvious reading -
  a too-tight pipe-close bound - was wrong, and two reproductions built on it came
  back green; capturing the failing run's own output named the case instead. Its
  planted leak was a `sleep 8` spawned inside a body that runs the whole install
  step, and the pipe is only examined after the body exits, so under load the
  plant expired before it was measured. `LEAK_SECONDS=45` names it once and a
  guard checks `LEAK_SECONDS > CLOSE_SECONDS`; both refusals are mutation-proven.
  Five consecutive gate runs green, against three red in roughly nine before. Its cases time a bound that must
  fire as `124` and a step whose output pipe must reach end of file, so a loaded
  host can move a case across its own boundary. Recorded in
  [`docs/history/RESUME.md`](docs/history/RESUME.md) so the next red is checked
  alone before anything is bisected.
- Record: [`TODO/clients.md`](TODO/clients.md), `CLIENT-14`. No version bump and
  no deploy: nothing is released, and one acquisition route means `E-ACQ-01`
  still refuses a record built from this capture.

### 2026-09-09T12:29:06Z

- ⭐ **`CLIENT-14`'s adapter exists and its target is measured.**
  `scripts/capture/adapters/aria2-next.sh` installs from the project's own
  releases and drives `start` and `stop` over JSON-RPC. Every subcommand was run
  on a session host: the build was fetched, verified against the vendor's
  published `sha256` sidecar with `sha256sum -c`, installed in **1.2 seconds**,
  asked its version, handed a torrent over `aria2.addTorrent` and shut down over
  `aria2.shutdown`.
- ⛔ **The version is the fourth field, not the third.** `aria2` prints `aria2
  version 1.37.0`; this prints `Aria2 Next version 2.7.4`. The adapter beside it
  would have published the literal string `version`, and nothing here would have
  refused it. The parse is anchored on the `version` token instead of a column.
- ⛔ **A fourth discovery surface, found by reading sockets rather than help
  text.** With DHT, LPD and peer exchange off, the running build was still bound
  to **UDP 1900**: `bt-port-mapping` - UPnP and NAT-PMP - defaults to true. The
  adapter switches it off and reads all five switches back from the build.
- ⛔ **No package index carries this fork**, so it has one route and `E-ACQ-01`
  refuses a record with one. The `package` route refuses by name, because
  `apt-get install aria2` would exit 0 having installed a different product.
- ⭐ **A newest release can carry no assets, and it was caught live.** `v2.7.5`
  was published at `12:11:10Z` with zero assets while `v2.7.4` beside it carried
  eight. `check-release-route.sh` now proves the resolver refuses rather than
  falling back to the older release, and `project-listing` takes several tags so
  a fixture can hold both.
- ⚠ **Two corrections.** `docs/client-matrix.md` claimed its target set is pinned
  "against the catalogue in both directions" and `check-project` in fact pins a
  hardcoded list of ids; the claim is withdrawn and `aria2-next` is on that list.
  A test comment said the catalogue carries 17 targets when it carried 16.
- Record: [`TODO/clients.md`](TODO/clients.md), `CLIENT-14`. No version bump and
  no deploy: nothing is released, and no capture has been dispatched.

### 2026-09-09T11:55:09Z

- ⭐ **`CLIENT-14` is filed, by operator direction**: the two-route capture is
  attempted through `AnInsomniacy/aria2-next`, acquired from that project's own
  releases and driven over RPC wherever RPC answers the same question as the
  command line. Record: [`TODO/clients.md`](TODO/clients.md).
- ⛔ **Nothing about that target is measured.** It has not been fetched, no
  release of it has been listed, no version installed and no RPC call made. The
  entry is the record of a direction, not of a capability.
- ⚠ `CLIENT-05` stays open and keeps its evidence: the hang is a measured
  property of this repository's capture path on a hosted runner rather than a
  fact about one product, and nothing establishes that a different target avoids
  it. What moves is which entry the first two-route capture goes through.
- ⚠ A new entry rather than an edit to `CLIENT-05`, because that entry carries
  ten dispatches' worth of measurement and rewriting its target would falsify a
  record this project needs to keep.
- Deployment: nothing deployed.


### 2026-09-09T11:14:54Z

- ⛔ **Run 10: a `timeout` around the step's own command did not fire either.**
  Both aria2 lanes entered *Install the client* and neither had returned twelve
  minutes later, against a bound that would have ended it at 570 seconds.
  Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`.
- ⛔ **Four independent bounds have now been measured not to fire on this step** -
  inside `install-client`, the runner's `timeout-minutes`, a watchdog loop in the
  step's own shell, and a `timeout` around the step's own process - and no such
  job has ever produced a log or an artifact.
- ⚠ That combination is what separates a fifth reading from the four guesses
  before it: a step that were merely stuck would leave a runner able to enforce
  one of four bounds and to upload a log, and nothing here does either. It is
  recorded as a reading rather than as a cause, and what it changes is where to
  look.
- ⭐ **So the next dispatch bisects with step NAMES**, which is the one signal
  that survives a job producing nothing: three bounded probes run before the
  install - host resources, sudo, and the preinstalled product - so a job that
  wedges localises itself to a handful of commands without a log, an artifact or
  a bound that fires.
- ⚠ The third probe is the asymmetry the whole thread rests on: `install-client`
  asks the adapter for a version before the route runs, and `aria2` ships on the
  image while `transmission-daemon` does not - so an aria2 lane executes the
  preinstalled product there and a transmission lane runs nothing. That
  difference is in every hung run and absent from every green one.
- ⚠ They are diagnostic rather than guards, and they come out when the question
  is answered.
- Deployment: nothing deployed.


### 2026-09-09T10:29:21Z

- ⭐ **Transmission captured green a fourth time** on client capture run 9, in
  three minutes, as the control beside an aria2 lane on the same image and the
  same step. Record: [`TODO/clients.md`](TODO/clients.md).
- ⛔ **And the watchdog in the step's own shell did not fire.** aria2's
  *Install the client* ran fifteen minutes against a 480-second deadline that
  loop would have enforced had the shell been looping. That is the third bound
  measured not to fire here, after `timeout` inside `install-client` and the
  runner's own `timeout-minutes`.
- ⚠ Transmission's install was 121 seconds where earlier runs recorded fourteen,
  so these hosts are slow today - and slow is what the aria2 lane is not, because
  a slow install would have been ended by the 420-second bound around the install
  call.
- ⭐ **So the bound moves outside the shell entirely.**
  [`scripts/acquisition/install-step.sh`](scripts/acquisition/install-step.sh)
  holds what the step did inline, and the workflow runs it under `timeout`, so
  the bounded process is the step's own and nothing inside it has to be reachable.
- ⛔ What that buys is not a faster failure but a job that reaches its uploads at
  all: runs 7, 8 and 9 each ended with no log and no artifact, so a hanging aria2
  lane has taught nothing three times running.
- ⭐ Two cases hold the new bound: it fires and the step ends with coreutils' 124,
  and a product that finishes inside it is not killed.
- Deployment: nothing deployed.


### 2026-09-09T09:39:11Z

- ⛔ **Client capture run 8 refuted the reading it was dispatched to test.** Both
  aria2 lanes sat in *Install the client* from 09:13:18Z and had not returned
  twenty-two minutes later, on the first run where nothing the route spawned
  could inherit the step's output pipe. Record:
  [`TODO/clients.md`](TODO/clients.md), `CLIENT-05`.
- ⭐ The same run says where the step is **not**: `install-client` bounds the
  install call at 420 seconds and kills 20 seconds later, so twenty-two minutes
  is three times past the last moment that call could have ended. Whatever is
  slow or stopped is outside the bounded call.
- ⚠ That dispatch changed two things at once, and the confound is recorded rather
  than argued away: a holder report was added to the same step and it walks every
  process's descriptors. It is bounded by `timeout 60` now.
- ⭐ *Install the client* bounds itself now. The install runs in the background
  and the step's own shell watches it, because that shell is outside every bound
  that has failed here - `timeout` ends its own child, and `timeout-minutes` was
  measured on run 6 not to end the step at all. Every tick leaves a process
  snapshot with `stat` and `etimes`, which is what separates a command that is
  slow from one that is stopped.
- ⛔ And the deadline is what makes any of it reach a reader: a job whose runner
  is killed leaves no log and no artifact, and a step that ends leaves both.
- ⭐ Two cases make that bound fire rather than assume it: a product slower than
  the deadline is killed and the step still ends, and one slower than a tick but
  inside the deadline is not.
- ⚠ **`AGENTS.md` rule 8 gained what run 8 measured about job logs.** The route
  answers **302** and redirects to a blob that answers `BlobNotFound` while the
  job is still running, so a `curl` without `-L` reads a status that is not the
  answer - and a general-purpose GitHub tool answers a bare 404 for the same job,
  hiding the distinction between *not written yet* and *not found*.
- ⚠ The step-body harness passes `-NoProfile` where GitHub's wrapper does not,
  and that departure is stated rather than silent: a gate row that loaded a
  contributor's profile would go red for something outside this repository.
  Decided by the operator.
- Deployment: nothing deployed.


### 2026-09-09T09:19:46Z

- ⭐ The two gate runners' row lists are compared.
  [`scripts/common/check-gate-rows.sh`](scripts/common/check-gate-rows.sh) is the
  row that does it, and `--rows` / `-Rows` are what make it cheap: each prints
  the names its own queue would have used and runs nothing. Record:
  [`TODO/ci.md`](TODO/ci.md), `CI-07`.
- ⛔ Nothing compared them before, and the gap is structural: the `sh` runner
  derives most of its provers from a list and the PowerShell one declares each by
  hand, so a prover added to the first and forgotten in the second is absent from
  that lane. `--strict` cannot help, because a row that was never named cannot be
  counted as a skip.
- ⚠ It compares sets rather than sequences, because the two halves schedule
  differently on purpose, and it refuses two short lists: a runner that printed
  nothing agrees perfectly with another that printed nothing.
- ⛔ Writing it found a live instance of this repository's own PowerShell hazard.
  `[switch]$Rows` collided with the runner's `$rows` accumulator - names are
  case-insensitive, so they are one variable - and **every** invocation of that
  runner then failed to bind. It is the third instance of the class, after
  `$args` and `[switch]$Marker` against `$marker`, and all three were found by
  running something once rather than by reading it. Record:
  [`docs/conventions/shell.md`](docs/conventions/shell.md) section 8.
- ⚠ One row's label differed between the halves - each spelled its own flag - so
  a comparison built on labels reported a false difference on a row both lanes
  have. Both name the question now.
- Deployment: nothing deployed.


### 2026-09-09T08:42:57Z

- ⭐ *Install the client* sends the route's output to a **file** and prints it
  afterwards, so nothing the product spawns inherits the step's own output pipe.
  Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`, and
  [`TODO/clients.md`](TODO/clients.md), `CLIENT-05`.
- ⛔ Three cases hold it: the block ends over a product that behaves, it ends
  over one that leaks a single process, and a planted extra descriptor onto the
  step's own output brings the hang straight back. Without the third the second
  passes equally over a block that never had the problem.
- ⚠ That plant was wrong on its first run and the reason is worth keeping: a
  shell applies redirections left to right, so a duplicate written after
  `>log 2>&1` points at the log rather than at the step's pipe. The case
  reported the hang not happening, over a plant that had duplicated the wrong
  file.
- ⭐ [`scripts/ci/report-holders.sh`](scripts/ci/report-holders.sh) names every
  process still holding an open descriptor on the route's log and prints the
  process table beside it, into the workdir the `always()` upload collects.
  ⚠ It runs under `sudo` because the install did: an unprivileged reader of
  `/proc` reports nobody holding a file that root processes are holding.
- ⛔ It is a mitigation and its outcome is the measurement. A lane that gets past
  that step says the class is what the hang was; one that hangs anyway refutes
  the reading with something that separates it rather than with another guess.
- Deployment: nothing deployed.


### 2026-09-09T07:58:50Z

- ⭐ A capture workflow's step bodies are now RUN, which nothing here did.
  [`scripts/ci/check-step-bodies.sh`](scripts/ci/check-step-bodies.sh) lifts a
  block out of `capture.yml` or `capture-client.yml` and executes it the way the
  runner does - under GitHub's default `bash -e` or its
  `pwsh -command ". '<file>'"` wrapper, prologue and residual-exit epilogue
  included. Record: [`TODO/ci.md`](TODO/ci.md), `CI-08`.
- ⛔ **A step does not end when its command exits.** The runner reads a step's
  output through a pipe, so it is over when the command has gone *and* that pipe
  has reached end of file - and a process the body left behind holding the
  step's stdout keeps it open with an exit code of 0 sitting in it. Every
  harness here redirected step output to a file, and a file has no reader to
  wait on, so the whole class was invisible. This one records both facts per
  body and keeps them apart.
- ⛔ `CI-08`'s written acceptance is met: the harness **refuses**
  `capture.yml`'s Windows *Restore the route* block in the form that failed
  capture run 1 and **accepts** it as it stands, over a stubbed guard that
  refuses. That comparison had been made once by hand in a session scratch
  directory, which is evidence rather than a control.
- ⭐ And `capture-client.yml`'s *Install the client* block is reported as a step
  that would not end when the product it drives leaks one process - the shape
  client capture run 7 has, in the exact step it happens in.
- ⚠ **Measured, and it is what pointed the instrument here.** Runs 6 and 7 ran
  the same aria2 install from commits whose only functional difference is where
  a later step sits - `git diff` over the two says so - and that step took six
  seconds in one and had not returned after sixteen minutes in the other.
- ⭐ [`scripts/ci/workflow-step.sh`](scripts/ci/workflow-step.sh) is now the one
  reader that lifts a step out of a workflow, and `check-workflow.sh` asks it
  rather than carrying a second parser. Its extraction was checked against the
  reader it replaced over every job and step in every workflow here: identical
  output on all sixty-six.
- ⛔ A missing step now answers differently from one that runs an action, so
  deleting a step from a workflow is reported instead of reading as a rule that
  passed. Record: [`scripts/README.md`](scripts/README.md).
- ⚠ A zero close-bound is refused, because `timeout 0` means **no limit** in
  coreutils: a ceiling edited to 0 would wait for a leaking body forever and
  then report the pipe as closed when the leak ended by itself. Found by
  planting it.
- ⚠ `scripts/README.md` said `ci/check-workflow.sh` is "a thirteenth mutation
  prover", which this change made wrong. The ordinal is gone rather than
  corrected - the third count in that file to go stale the same way.
- Deployment: nothing deployed.


### 2026-09-09T06:54:06Z

- ⭐ The three deep-review passes, and what each found. Record:
  [`docs/history/SESSION-2026-09-09-SECONDROUTE.md`](docs/history/SESSION-2026-09-09-SECONDROUTE.md),
  which `check-docs` refuses unless the history index links it.
- ⛔ The door sweep found four readers of `describe`'s `key=value` format in three
  spellings, one of which - `awk -F= '{ print $2 }'` - truncates at a second `=`.
  Measured over `weird=a=b`, which it reads as `a`. All four read the whole value
  now. ⚠ No key an adapter prints today carries one, which is exactly when a
  reader is easiest to get wrong.
- ⭐ The same pass established that no **existing** caller of `replace_once` had
  been mis-planting: their literals carry no regex metacharacter, so the defect
  fixed earlier today was latent for every shipped case and live only for the one
  added with it.
- ⛔ The guard-mutation pass found the one new rule without its own refutation -
  every upload comes after *Restore the route* - and planted against it, plus its
  silence on a workflow that uploads no install logs.
- ⛔ The claim audit checked two numbers against the tree. "Nine gate runs inside
  `check-workflow`" is right; "nine checks call `cargo build --example`" was
  wrong - it is fourteen - and is now "most of them", because a count in prose is
  a value in two places with nothing comparing them.
- Deployment: nothing deployed.


### 2026-09-09T05:15:32Z

- ⭐ CI was 30.5 minutes and one step was 25.9 of them. Measured on run 83 before
  anything was changed: *Workflow acceptance* 25.9 min, *Repository gate* 3.5,
  everything else on that lane under one, and the whole Windows lane 2.4.
  Record: [`TODO/ci.md`](TODO/ci.md).
- ⛔ That step runs the whole gate **nine times**, so the gate was measured next:
  198 seconds for 29 checks, of which 183 were three - `check-twins` 90.7s,
  `check-capture-client` 47.9s, `check-capture` 44.6s. Twenty-six sub-second
  checks were serialised behind them.
- ⭐ The gate now runs its checks concurrently (198s to 100s) and `check-twins`
  runs its twelve pairs concurrently with each pair's two halves at once (90.7s
  to 69s, taking the gate to 73s). *Workflow acceptance* is a job of its own
  beside the two lanes rather than the tail of one.
- ⛔ **But 73 seconds was wrong.** `check-capture` drives real sockets against a
  three-second deadline, and under twenty-seven concurrent checks it reported a
  refusal that arrived for a reason the case had not planted - passing alone on
  the same tree minutes later. The two socket harnesses run after that batch now,
  and the shipped gate is **118-125 seconds**, measured twice.
- ⚠ Raising the deadline was tried first and measured: `check-capture` is 45
  seconds at `3` and 79 at `6`, `check-capture-client` 48 at `5` and 168 at `20`,
  because several of their cases are ones the deadline itself has to end. Eight
  to eleven seconds of gate per second of deadline, paid nine times over, to buy
  back forty-seven.
- ⛔ And an ad-hoc `sed -i 's/^SECS=[0-9]*/.../'` run while measuring that edited
  a second line the pattern was not meant to reach - `SECS="$2"` inside an
  embedded stub, because `[0-9]*` matches no digits at all. Record:
  [`docs/conventions/shell.md`](docs/conventions/shell.md) section 2.
- ⭐ **Measured on the lane rather than promised.** Run 85 against run 83: the
  wall clock is **30.5 minutes to 21.3**, and the Linux gate - compile, tests,
  lints and the gate, which is the answer a contributor waits for - is **30.5
  minutes to 3.7**. Both fast jobs were green while the acceptance harness was
  still running. `check-workflow` is 95 of 95 locally at 1195 seconds.
- ⛔ Every exit code is still read from the process that produced it, and every
  row is still assembled at its own index, so two runs over one tree produce one
  report. There is no pipeline in either change.
- ⭐ Most of the checks call `cargo build --example` and cargo locks the target
  directory, so the examples are built once before the queue - the same work,
  done once instead of nine times, without which the concurrency bought nothing.
- ⚠ What was not changed is what `check-workflow` runs: it executes the
  workflow's own step commands read out of `ci.yml`, so the saving comes from the
  gate being faster rather than from it doing less.
- ⛔ And the harness caught a defect the work introduced, which is the argument
  for it: `shellcheck file && shfmt -d file >/dev/null && echo "LINT OK"` runs
  shfmt only when shellcheck says nothing, so an info-level finding meant shfmt
  never ran and the absent "LINT OK" read as an empty diff. Two checks joined by
  `&&` are one check. Record:
  [`docs/conventions/shell.md`](docs/conventions/shell.md) section 2.
- Deployment: nothing deployed.


### 2026-09-09T04:14:03Z

- ⛔ The bound added an hour earlier does not fire. Client capture run 6 dispatched
  both aria2 routes; the package lane's *Upload the install logs* began at
  03:50:11Z, wrote its 3247-byte artifact at 03:50:12Z and was still running at
  04:07:39Z - seventeen minutes under a five-minute `timeout-minutes`. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⭐ That also closes the size question the same run opened: three kilobytes hangs
  exactly as a thirteen-times-larger artifact does, so it is not the upload's
  volume.
- ⭐ The step moved to the end of the job instead, which needs no diagnosis.
  `if: always()` runs after a failed step wherever it sits, so sitting before the
  route cut bought nothing the last position does not - and from the end a hang
  costs the job's tail rather than the capture and its evidence.
- ⚠ The bound is removed rather than kept beside the move. A bound measured not to
  bound the one failure it was added for is decoration, and decoration in a
  workflow reads as a control the next reader will trust.
- ⭐ `check-workflow` asserts the constraint the move creates: every upload comes
  after *Restore the route*, because between the cut and the restore the host has
  no way off itself.
- ⭐ Three things in run 6 worked and are worth separating from the hang: the
  resolve step skipped on the package lane and ran in one second on the release
  lane, the release route installed in 144 seconds for the second time, and the
  named upload paths cut that artifact from 42.6 megabytes to 43179 bytes.
- Deployment: nothing deployed.


### 2026-09-09T03:42:01Z

- ⭐ Client capture run 5: **a `release` route acquired a build on a capture
  host**, the first time any route of that kind in this tree has installed
  anything on one. The resolve step took one second and the source build took 144
  seconds. Record: [`TODO/clients.md`](TODO/clients.md).
- ⭐ The tarball the runner fetched came back inside the install artifact,
  unauthenticated, and `sha256sum -c` verifies it against the digest `ACQ-03`
  recorded on 2026-09-08 from a different fetch.
- ⛔ Two builds, one version, measured on the capture host rather than argued
  for: the image's aria2 1.37.0 enables Async DNS, Firefox3 Cookie, Metalink,
  XML-RPC and SFTP against GnuTLS 3.8.3, and the 1.37.0 the route compiled enables
  none of them and speaks HTTPS through OpenSSL. Different compilers too.
- ⛔ **The aria2 hang is not the package and not apt.** The release route runs no
  package operation of any kind and *Upload the install logs* hung identically.
  Two named causes have now been refuted, neither by reading.
- ⭐ And there is a clock on it: the artifact was written five seconds into that
  step, and the step had not returned thirteen minutes later. The step is bounded
  and non-fatal now, which needs no diagnosis - the artifact is on the server
  before the hang begins, so the capture that follows stops being thrown away.
  `CI-08` still owns the cause.
- ⛔ That upload was sending a build tree: 1867 entries and 42.6 megabytes for a
  step called *Upload the install logs*, because a source-build route compiles
  inside the workdir the path named. It also omitted the install record itself.
  The paths are named now.
- ⭐ `check-workflow` refuses an evidence upload that is `continue-on-error` or
  that stops declaring `if-no-files-found: error`, because a green capture with
  no evidence is the worst outcome this workflow has. Its step reader is refuted
  first: an over-wide one would read the neighbouring upload's own
  `continue-on-error` and the rule would look like it works.
- Deployment: nothing deployed.


### 2026-09-09T02:58:42Z

- ⛔ The work order said a single-route capture could become a record and it
  cannot. `docs/history/RESUME.md` has carried "a single-route, single-connector
  capture VALIDATES and refuses to publish"; one connector does exactly that and
  one route is refused at the **validity** gate by `E-ACQ-01`, so
  `Profile::to_json` will not write it and the store cannot hold it. Every
  capture this project has run is one route. Record: [`TODO/ci.md`](TODO/ci.md).
- ⭐ Measured by stripping the golden fixture twice and running
  `validate-profile` over each copy, and both facts were already in the suite -
  only the prose disagreed. Nothing in the code changed; `TODO/ci.md` carries why
  the refusal is the right one and which two tests already prove it.
- ⛔ So `CI-09` waits on a two-route capture rather than on assembling code, and
  aria2 is the only target whose two routes currently resolve the same version.
- ⛔ The workflow's resolve step redirected into a directory the script it calls
  had not created yet: a shell opens `>` targets before it execs, so the step
  would have died on the redirection rather than on anything the script did.
  Found by running that step's body verbatim on this host, which is the first
  time any capture workflow step body has been executed outside a runner.
- Deployment: nothing deployed.


### 2026-09-09T02:51:47Z

- ⛔ A gate run left four untracked files in `scripts/capture/`, and the only
  thing that noticed was a person reading `git add -A`. The two probe cases added
  minutes earlier wrote their scratch files beside the file they were handed, and
  two callers hand `store_probe_guards` a **tracked** path. Every harness header
  says it plants only in disposable state; nothing compared the tree before and
  after. Record: [`scripts/common/check-gate.sh`](scripts/common/check-gate.sh).
- ⭐ `tree-unchanged` is a row in both gate halves now, and each was proved by
  planting its own defect: the sh half over probes writing beside a tracked file,
  the PowerShell half over a check that creates one.
- ⚠ It is a comparison and not a demand for a clean tree, because the gate runs
  while somebody is editing - so it sees the run that makes the mess and not the
  one after. Measured: the second run over the same planted tree went green,
  reading the first run's droppings as its own `before`. A lane that starts from
  a fresh checkout fires every time.
- Deployment: nothing deployed.


### 2026-09-09T02:21:06Z

- ⛔ The second acquisition route was unrunnable and the missing piece was the
  artifact, not the plumbing. Every `release` route refuses without
  `BIT_IDS_RELEASE_URL` "resolved before the route was cut", and nothing in the
  tree produced one: `resolve-stable` orders versions and selects no asset. Four
  dispatches passed `--route package` and only that. Record:
  [`TODO/acquisition.md`](TODO/acquisition.md).
- ⭐ `AssetPattern` and `select_asset` choose one artifact of the selected
  release, from the same response the version came out of, so one recorded digest
  covers both decisions. The location is read from the listing and never
  composed.
- ⭐ `{version}` in a pattern is what makes it unambiguous on a real release, and
  the two ambiguous cases are measured rather than defensive: qBittorrent's
  release-5.2.3 carries **fourteen** assets including two Linux `AppImage` files
  differing only by an `_lt20`, and aria2's release-1.37.0 carries three source
  archives differing only in compression. Two matches refuses; taking the first
  would choose by the order the source listed them in.
- ⛔ `CLIENT-01`'s record said that release "offers a Linux AppImage, a Windows
  `x64_setup.exe` and a source `tar.xz`". It was one build short, in exactly the
  place a route has to choose. Record: [`TODO/clients.md`](TODO/clients.md).
- ⭐ Which artifact is target knowledge, so each adapter declares its repository,
  its tag scheme and its asset pattern through `describe`; `resolve-release.sh`
  reads them, so the workflow step carries no case over which product it runs.
- ⭐ `capture-client.yml` takes the route as a matrix dimension beside the
  adapter. One route per host is the design: two routes on one machine means the
  second installs over the first, and `ACQ-03` would compare one host's final
  state rather than two acquisitions. ⚠ The artifact names carry the route now,
  because two legs otherwise upload under one name in one run.
- ⭐ Driven end to end here against all three live listings, and the URL aria2's
  selection produced served bytes `sha256sum -c` verified against the digest a
  different session recorded from a different fetch on 2026-09-08.
- ⛔ `replace_once` counted with `grep -F` and edited with `sed`, so its literal
  and its pattern were different languages. Over `axb then a.b` the literal
  `a.b` occurs once, the count accepted, and sed replaced `axb`: the plant
  applied somewhere no case named, so nothing reported NOT-PLANTED and the
  refusal that followed was read as proof. A literal carrying a `/` could not
  plant at all, which turned `store_probe_guards`' no-op row into a pass over a
  sed that never parsed its own expression. Both are probes now, and both refuse
  the old implementation. Record: [`scripts/corpus/store-lib.sh`](scripts/corpus/store-lib.sh).
- ⛔ `check-gate` labels a row with a check's basename, so two harnesses in
  different directories collide into two rows a reader cannot tell apart. Found
  by writing `acquisition/check-release` beside `publishing/check-release`. The
  runner refuses a duplicate label with exit 2 now, and the new harness is
  `check-release-route`.
- Deployment: nothing deployed.


### 2026-09-08T23:16:05Z

- ⛔ A third rule that existed only in prose. `docs/history/README.md` carries a
  note saying three session records were missing from it until 2026-09-08; a
  fourth was missing from it by the end of that same day, held in the tree only
  by a link from `RESUME.md`, which is overwritten every session. The note
  observed the pattern and changed nothing. Record:
  [`docs/history/README.md`](docs/history/README.md).
- ⭐ `check-docs` refuses a `SESSION-*.md` the index does not link, in both
  halves, mutation-proved. It is a stronger rule than the orphan one beside it:
  a record linked from `RESUME.md` alone passes that and is still absent from the
  page that exists to list them. It fired on this session's own record first.
- ⚠ And `RESUME.md` claimed those records were "linked from here and nowhere
  else", which was false - the index links them - and false in the direction that
  mattered, because the one record it was true of was the missing one.
- ⭐ The line-endings rule added earlier today caught its own author within the
  hour: writing that twin with a tool that emits LF left `check-docs.ps1` at
  `w/lf` under `attr/text eol=crlf`, and the gate refused it. Before the rule
  existed, the same state reached a commit and only `git` mentioned it.
- ⭐ The session record is
  [`docs/history/SESSION-2026-09-08-ROUTES.md`](docs/history/SESSION-2026-09-08-ROUTES.md).
- Deployment: nothing deployed.


### 2026-09-08T23:05:56Z

- ⛔ All three `release` routes fetched an artifact and returned 0 having
  installed nothing, so `version` answered from whatever was already on PATH.
  A second route would have measured the FIRST route's build. Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⭐ aria2's release route now unpacks, configures, builds and installs into a
  prefix `binary()` prefers, refusing at each step: 132 seconds from URL to a
  runnable `aria2c` answering `1.37.0`, driven end to end here.
- ⭐ qBittorrent's installs the AppImage, which IS the program, with
  `APPIMAGE_EXTRACT_AND_RUN=1` exported so a host without FUSE can run it.
  ⛔ Neither branch is driven. Transmission's refuses outright and says what it
  would take, because writing a second product's CMake build untested is
  guessing.
- ⛔ A version is not an identity. `describe` names the executable it would ask,
  and `install-client` records both binaries and both digests: `acquired=yes`
  now also when the version did not change and the executable did. That branch is
  the aria2 pair, not a hypothetical - the same run recorded `acquired=no` before
  it existed, over a route that had just compiled and installed a different
  program. Record: [`TODO/acquisition.md`](TODO/acquisition.md).
- ⚠ Two builds of one tarball at two prefixes differ, so a source route can never
  reach `byte_identical`. That is a property of the route, and it is the same
  conclusion `ACQ-03` reaches when it requires a capture per route.
- Deployment: nothing deployed. Eighty harness cases pass, five of them new.


### 2026-09-08T22:46:54Z

- ⭐ Two independent acquisition routes for one target were held at once for the
  first time, with no disposable host, because building and asking for a version
  is not a capture: Ubuntu's aria2 1.37.0 and a 1.37.0 compiled from the vendor's
  own release tarball. Record: [`TODO/acquisition.md`](TODO/acquisition.md).
- ⛔ They report one version and are not one build. Their feature lists differ -
  the package build enables Async DNS, Metalink, XML-RPC, SFTP and Firefox3
  Cookie - and features are what a build does on the wire.
- ⭐ So every adapter's `version` writes the build's whole answer to stderr, which
  both callers already keep: `version.err` beside the install record and
  `adapter.err` inside the evidence bundle. The parsed value on stdout is
  unchanged.
- ⛔ And `installed_executable` records a SHIM for that package route:
  `/usr/bin/aria2c` is 14 kilobytes linking `libaria2.so.0`, so a per-route digest
  comparison over a library-split package compares two launchers. `ACQ-03` carries
  it as a residual.
- ⭐ aria2 is the one target whose two routes currently resolve the same version;
  Ubuntu trails upstream by a major version for both other clients, and rule 5
  forbids backfilling to make a pair agree. The vendor publishes no Linux binary,
  so that route is a source build: 19s to configure, 126s to `make -j4`.
- ⚠ The `acquired=no` verdict fired on the real product for the first time, not
  just on a stub: the real aria2 adapter through `install-client` on this host.
- Deployment: nothing deployed. The release routes still fetch without installing.


### 2026-09-08T22:24:32Z

- ⭐ `CI-08`'s second Prove half is done: the gate gives the same verdict under a
  hostile environment - `CARGO_TARGET_DIR` elsewhere, `TMPDIR` moved,
  `COLUMNS=80`, the locale changed - row for row. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⛔ The first locale run tested nothing: `LC_ALL=C` on a host already in `POSIX`
  changes nothing, which `conventions/shell.md` states about a different check.
  The host was measured and the run repeated under `C.utf8`.
- ⭐ Three defaults are stated rather than inherited, in both halves and
  mutation-proved: `set -u` as the first code line of every executable script,
  a cargo output path that must name `CARGO_TARGET_DIR`, and the `shfmt` version
  compared between the workflow and the provisioning script.
- ⚠ The `set -u` rule reads the FIRST code line, because this tree holds a
  heredoc whose body begins `set -u`; a looser rule would accept a script whose
  only `set -u` belongs to a stub it writes. `store-lib.sh` is exempt by name:
  it is sourced, so an option set there changes the caller's shell.
- Deployment: nothing deployed. `CI-08` stays open on the harness gap - nothing
  runs a capture step's body.


### 2026-09-08T22:05:54Z

- ⭐ `FOUND-05` closes. `sh scripts/doctor/provision.sh` installs `pwsh`,
  `shellcheck` and `shfmt` in four seconds on a host with none of them, verifying
  every download against a pinned digest with `sha256sum -c` before anything is
  executed. Record: [`TODO/foundation.md`](TODO/foundation.md).
- ⛔ A host without `pwsh` does not run a smaller gate, it runs a RED one:
  measured by taking all three away, `--strict` answered `FAIL check-capture` and
  `SKIP check-twins`. With them back the only row left is `check-remote-items`.
- ⛔ The first version of the script reported two tools wrong that it had
  installed correctly. `fetch_verified` and `provide` shared `_want`, because a
  POSIX shell function has no locals, so the caller compared a version against a
  digest. `shfmt` was the one row that reported correctly - `go install` is the
  route that never calls the fetcher, so the two tools that downloaded were
  exactly the two that lied. Found by running it; invisible to `shellcheck`.
- ⭐ The commands left `PROGRESS.md`, the version left `supply-chain.md`, and the
  two places that still name it - the workflow and the script - are compared by
  `check-project` in both halves.
- ⚠ `CI-07` said `thirteen` `n/a` rows and the runner declares fourteen. That is
  the second time a count of those rows went stale in prose; the number is
  removed rather than corrected. Record: [`TODO/ci.md`](TODO/ci.md).
- Deployment: nothing deployed.


### 2026-09-08T21:50:34Z

- ⛔ A gate row this repository was documented as having did not exist.
  `conventions/shell.md` section 5 says every tracked file's working-tree line
  endings are compared against what `.gitattributes` resolves for it, naming
  `git ls-files --eol`; nothing did. Record: [`TODO/ci.md`](TODO/ci.md).
- ⚠ Found by breaking it: a file-writing tool that emits LF left
  `check-project.ps1` at `w/lf` under `attr/text eol=crlf`, a full gate passed
  over that tree, and only an incidental `git commit` warning said so. The index
  is normalised either way, which is why nothing else could see it.
- ⭐ The check is a row in both halves now, planted in both directions - LF where
  CRLF is required and CRLF where LF is - because those are different branches
  and `git diff` prints nothing for either.
- Deployment: nothing deployed.


### 2026-09-08T21:20:35Z

- ⭐ A real capture bundle was read back outside the run that wrote it for the
  first time: run 4's transmission artifact downloaded through rule 8's route,
  and `sha256sum -c` reports `OK` for all three evidence files. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⛔ The capture-to-publisher pairing cannot be exercised at all. The publisher
  downloads an artifact named `bundle`; the four uploads in this tree are
  `install-*`, `capture-client-*`, `capture-linux-*` and `capture-windows-*`. Its
  first step fails on every run that exists and every run that could be
  dispatched, so the v7/v8 question is unreachable rather than merely untried.
- ⛔ And a capture bundle is not a publication bundle: no `MANIFEST.json`, and
  what sits between them is an assembler that reads records, of which none has
  been written.
- ⭐ `check-project` compares every `download-artifact` name against every
  `upload-artifact` name in the tree, `${{ ... }}` normalised to `*` on both
  sides. A missing producer may be declared with `bit-ids:no-producer=<ENTRY>`,
  the entry must be one `INDEX.md` carries, and a declaration over a name that
  has gained a producer is refused - so the marker cannot outlive its reason.
- ⚠ That rule's own first version was wrong twice, and only planting found
  either: it took the STEP's name for a step written `- with:` / `name:` /
  `uses:`, which YAML permits, and it ignored `pattern:`, which
  `download-artifact` also accepts. Neither shape exists in this tree. ⛔ The
  first plant passed before the reported name was checked, because a rule that
  never found the row and a rule that accepted it both exit 0.
- ⭐ Six states, both halves, same verdict on each.
- ⚠ `publishing.md` said the assembler consumes capture artifacts. It consumes a
  tree built from the store, and that sentence is amended.
- ⛔ A second sweep, a second value in two places: `check-project` compared IDs
  and statuses between `INDEX.md` and the entries and never priority or effort.
  `FOUND-05` was `P2` in the index and `P1` in its own entry, from the commit
  that created both. It compares all four now, in both halves, and the index
  carries `P1`. Record: [`TODO/foundation.md`](TODO/foundation.md).
- ⚠ The index's priority table is derived from the index's own rows, so it had
  agreed with the wrong half and corroborated nothing. An effort-only plant
  fires the new check alone, which is what shows the two are not one guard.
- Deployment: nothing deployed. The producer is absent on purpose: every record
  in the store is synthetic, and a `bundle` artifact would be one boolean from
  the data branch.


### 2026-09-08T21:03:10Z

- ⭐ The aria2 install logs were read out of the uploaded artifact rather than
  dispatched for, and they refute the recorded cause: both hung runs say
  `0 newly installed`, so no package operation happened and `needrestart` never
  ran. Run 3 used `NEEDRESTART_MODE=a` and run 4 used `l`; the jobs hung
  identically. Record: [`TODO/clients.md`](TODO/clients.md).
- ⛔ And the hang is in *Upload the install logs*, not the install: the install
  step succeeded in six seconds in both runs, and the artifact that step produced
  is complete and downloadable, so its work finished and the step still did not
  return.
- ⛔ The finding that outlives it: a route that installs nothing exits 0. `aria2`
  ships on `ubuntu-24.04` and every `release` route fetches an artifact without
  making it the executable the adapter asks, so two routes can declare two
  independent resolvers, agree on one binary's version, and reach `ACQ-03` as
  `byte_identical` - its strongest verdict - having acquired nothing.
- ⭐ `install-client` asks the host for a version before the route runs and
  records `preexisting_version` and `acquired`. Three shapes have cases and the
  derivation is proved by planting it. Record:
  [`TODO/acquisition.md`](TODO/acquisition.md).
- ⭐ First driven pass of the aria2 adapter against a real build: install 8s,
  the no-op repeat 3s, `version` answering `1.37.0`.
- Deployment: nothing deployed. No route has been shown to acquire a build and
  no record exists.


### 2026-09-08T19:20:22Z

- ⭐ Client capture run 4 measured a second client and re-measured the first:
  qBittorrent `4.6.3` announced once and Transmission `4.0.5` announced twice,
  each under containment with its bundle verified. `CLIENT-01` has its first
  measurement. Record: [`TODO/clients.md`](TODO/clients.md).
- ⭐ The `[LegalNotice]` key in the profile does what the refused flag was
  supposed to do.
- ⛔ Neither entry closes: one route, one connector, no record in the store and
  no Windows, for both.
- ⚠ `NEEDRESTART_MODE=l` did not fix the aria2 hang. What run 4 establishes is
  the boundary instead: it is that one package install, it survives the step
  that caused it, and two other adapters doing the same `apt-get` on the same
  image do not hang.
- Deployment: nothing deployed. Three evidence bundles exist and no record does.


### 2026-09-08T19:07:10Z

- ⭐ Client capture run 3 read the product's own words back: `--confirm-legal-notice
  is an unknown command line parameter`. Both of run 2's fixes worked, which is
  what made that message reachable. Record: [`TODO/clients.md`](TODO/clients.md).
- ⛔ So that flag was never a control, and it was on both paths into the product:
  adding it to the version call to match `start` looked like closing a one-gated
  door and spread a refused argument instead. The acceptance is written into the
  profile now, which is the next assumption and is not measured either.
- ⚠ `aria2` moved its hang rather than losing it: the install succeeded and the
  step after it hung. `NEEDRESTART_MODE=a` stops the dialog by restarting the
  services it lists, which on a runner means daemons the job stands on; both
  adapters use `l` now, which reports and touches nothing.
- ⭐ The install logs uploaded this time, so nothing was lost.
- Deployment: nothing deployed. One capture exists and it is an evidence bundle.


### 2026-09-08T18:52:06Z

- ⛔ Client capture run 2 found two defects in code written the same day. The
  adapters read `--version` through a pipe, so `head`'s status masked the
  product's and the refusal was dead code; and `install-client` discarded the
  adapter's stderr and then reported that no version arrived. Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⭐ `qbittorrent` now fails in fifty seconds with a readable verdict rather than
  in half an hour with none: the install succeeds and the `version` call
  refuses.
- ⚠ `aria2` hung again and the 900-second bound did not end the job. `-k` forces
  a kill and the bound sits under the job timeout now, and neither is confirmed
  to be the cause.
- ⭐ The certain fix does not depend on that diagnosis: the workflow uploads the
  install logs on `always()`, so the next hang leaves its own evidence whatever
  kills the runner.
- Deployment: nothing deployed. One capture exists and it is an evidence bundle,
  not a record.


### 2026-09-08T17:54:06Z

- ⭐ **The first client capture.** Client capture run 1 installed Transmission
  4.0.5 from the Ubuntu package index on a hosted runner, deleted its default
  route, and handed the build a torrent naming this project's lab; it announced
  twice and its bundle verified under `sha256sum -c`. Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⛔ It does not close `CLIENT-06`. One route ran rather than two, one connector
  observed it, nothing wrote a `Profile` into the store, and the Prove names
  Windows.
- ⛔ Two of the run's three jobs sat in the install step for over half an hour
  and reported nothing, because no adapter call had a time limit. Every one is
  bounded now, in the callers rather than in each adapter, and a refusal names
  the timeout and prints the route's own log.
- ⛔ Three narrower causes were fixed with it: stdin is `/dev/null` on every
  adapter call, `NEEDRESTART_MODE=a` is set for apt, and `qbittorrent-nox
  --version` carries `--confirm-legal-notice` - which the `start` call had and
  the `version` call did not. ⚠ Which of the four it was is not established, and
  the log is gone.
- ⭐ `check-no-secrets --public` allows a measured peer ID cited with its phrase
  and backticks, which is the shape a published record will carry constantly.
  Proved narrow in both twins: no phrase, no backticks, a longer run, and a
  second bare run on the same line are each still refused.
- Deployment: nothing deployed and nothing published. The capture produced an
  evidence bundle and an attestation, not a record.


### 2026-09-08T17:02:04Z

- The client capture path, every layer of it except a measurement:
  [`capture-client.sh`](scripts/capture/capture-client.sh),
  [`install-client.sh`](scripts/acquisition/install-client.sh), an adapter
  contract with a file per target under
  [`scripts/capture/adapters/`](scripts/capture/adapters/), and
  [`capture-client.yml`](.github/workflows/capture-client.yml). Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⛔ Two conditions are what make a run a measurement rather than an
  attestation, and [`TODO/clients.md`](TODO/clients.md) states them and says
  what each refuses.
- ⛔ `stock_client` is the adapter's own declaration rather than a constant, and
  the harness proves the field varies by running one stub under both.
- ⭐ Two defects found by the harness on its first run. `the build announced
  nothing` was unreachable, because the segment guard refuses silence first; the
  case is written against a connection that reaches the tracker without
  announcing instead. And the peer-ID guard took the observer's word
  unconditionally, so an announce carrying none made it search the transcript
  for the literal word `absent`.
- ⭐ A door sweep found the Windows gate would have run one row fewer than the
  Linux gate with nothing naming the missing one. The row is written with its
  own reason; the class, that nothing compares the two runners' lists, is filed
  in `CI-07`.
- ⛔ `check-workflow`'s capture block reads every `capture*.yml` now rather than
  the one file it was written for, and the two new install-ordering rules are
  refuted by moving the step in both directions.
- Deployment: nothing deployed and nothing installed. No adapter has ever run
  against a product: a session host is not disposable, so the whole path was
  driven against a stub adapter and `curl`. Every attestation in this tree still
  says `kind=fixture`.


### 2026-09-08T16:52:00Z

- `check-no-secrets --public` allows an info hash cited in this project's one
  prose spelling, and refuses everything adjacent to it. Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⛔ It turned CI run 67 red on both lanes, over a real measurement written into
  a record the commit before: a torrent's identifier is forty hex digits and the
  rule that hunts long hex could not tell it from a credential.
- ⚠ The narrowing is the phrase and both backticks, not the shape. A bare
  forty-digit run elsewhere in a sentence is still a finding, and a rule
  allowing any backticked forty digits would have allowed a commit SHA.
- ⭐ Proved by planting: a bare run, a backticked run under no phrase, a
  forty-four digit run under the phrase and a token shape are each still
  refused, and both twins agree on every one.
- Deployment: nothing deployed. No record was published.


### 2026-09-08T16:21:40Z

- `CLIENT-01` gains the observer half a stock build can actually be pointed at:
  [`client-capture`](crates/bit-ids-probe/examples/client-capture.rs) runs the real
  HTTP tracker observer and generates the torrent with `announce` set to the
  endpoint the operating system just gave it. Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⛔ `evidence-bundle` could not be driven by a client and that was structural.
  Its torrent names a hard-coded `http://127.0.0.1:6969/announce`, so the only
  thing that could reach its observer was something told the endpoint
  separately. A client reads the address out of the file or it never announces.
- ⚠ The peer surface is dialled rather than offered, because `TrackerResponse`
  is cloned into the responder while the lab is still being built and `Lab`
  binds port zero: a tracker answer naming this lab's own peer port would have
  to predict one that does not exist yet. ⭐ Dialling is also the stronger role,
  since the side that dials sends its handshake first.
- ⭐ Driven with two readers this project did not write. `torf` 4.3.1 took the
  announce URL out of the generated `.torrent` and it matched the address the
  observer printed and the port `curl/8.5.0` then announced to; `torf`
  re-derived the info hash from the metainfo in the written bundle and agreed
  with the observer's own line.
- Deployment: nothing deployed. No client is installed anywhere this project may
  capture on, so every run of this so far was driven by `curl` and no build has
  been measured.


### 2026-09-08T15:05:08Z

- `LIB-02` closes: the bit-cli adapter, which is a comparison rather than a
  generated table. Record: [`TODO/library.md`](TODO/library.md).
- ⭐ The catalogue answers "what did you measure"; this answers the question a
  client actually has, which is whether what it says about itself is what this
  project measured.
- ⛔ It fails closed. `Answer::agrees` is true for one variant, and every way of
  not knowing - no record, no observation, a state that asserts nothing, a
  measured absence, a value with no fixed span - is `Unmeasured` and never a
  pass. Today every answer about a real client is `Unmeasured`, because nothing
  has been measured; a design that let that read as agreement would report this
  project's own emptiness as a clean bill of health.
- ⛔ A disagreement anywhere outranks agreement everywhere else: a claim is one
  constant in the tool's source, so it is a claim about every build of that
  version, and counting a majority would be a vote over a measurement.
- ⭐ The rejected interface is the one the Problem describes: emitting a Rust
  source file for bit-cli to vendor puts a second copy of the measurement inside
  the tool, drifting the same way and now with this project's name on it.
- ⭐ Driven with a third party's real input: bit-cli's prefix `-CL0200-` was
  derived from its own `peer_id.rs` and workspace version, and asked of a real
  publication the adapter answers `NoRecord`. ⛔ No document here claims to have
  measured bit-cli, and nothing was written to that repository.
- Deployment: nothing deployed. No record was published and nothing was written
  to the data branch.

### 2026-09-08T14:09:21Z

- `PUB-05` closes: the SQLite rendering, and with it every path
  [`docs/publishing.md`](docs/publishing.md) promises is written.
  Record: [`TODO/publishing.md`](TODO/publishing.md).
- ⭐ It is the only rendering that is both queryable and lossless. The CSV cannot
  hold the acquisition routes, the observations, the corroboration or the
  evidence list; this tabulates all of them, keeps each record's canonical bytes
  in a `document` table, and says what it does not carry in an `omission` table
  because a database can describe itself and a CSV cannot.
- ⛔ It is not behind a cargo feature. `PUB-04` derives a consumer's caching
  contract from the set of published paths, so a build that could omit one would
  assemble a manifest with a hole in it. The cost is a vendored SQLite on every
  consumer of the crate and [`docs/consuming.md`](docs/consuming.md) says so.
- ⭐ The version was measured rather than taken: `rusqlite` 0.40.2 resolves
  thirteen further packages that 0.37.0 does not, a WebAssembly stack this
  project never builds for, for the same API and the same bundled library.
- ⛔ Reproducible bytes are not free for this format. The page size is stated
  rather than inherited from the library, the rows go in as one transaction in
  ascending record order, and the file is vacuumed so no freelist encodes the
  order rows happened to arrive in.
- ⭐ Python's `sqlite3` is in the standard library, so unlike `cbor2` the
  independent reader runs on every gate rather than once in an entry's evidence.
  It is older than the writer, which is the direction that matters, and it has
  been seen to refuse a truncated database.
- Deployment: nothing deployed. No record was published and nothing was written
  to the data branch.

### 2026-09-08T12:37:28Z

- `CI-06` closes. Capture run 2 is green on both platforms, first attempt after
  the fix. Record: [`TODO/ci.md`](TODO/ci.md).
- ⭐ Every uploaded bundle verifies under `sha256sum -c`, and each attestation
  names `kind=fixture`. `curl` put the bytes on the wire from both runner images
  and the transcript carries the run-id token the driver knows it sent.
- ⛔ The entry's own Prove asked that two runs of one platform report different
  fingerprints, and on Windows they did not: two hosts that were both fresh, each
  run's claim succeeding, reported the same value. The Linux pair differed.
- ⭐ That is structural rather than a defect to patch, and
  [`docs/capture-host.md`](docs/capture-host.md) carries the reasoning. The
  workflow's Windows summary said "A repeat means the host was not destroyed" and
  now says what the value actually identifies.
- ⛔ A door sweep over the three new rules found the same defect inside them:
  they matched `shell: pwsh` in `.github/workflows/*.yml` alone, while
  `shell: powershell` is the same language, a composite action carries its own
  steps and the same permissions, and a workflow may be `.yaml`. None of the
  three exists here, which is when a scope is easiest to get wrong. The scope is
  the action-pin rule's now, proved by planting each missing fixture.
- ⛔ Widening it broke the PowerShell half and both halves went on agreeing on
  the clean tree: `Resolve-Path -Relative` prepends `./` to most paths and not to
  one already starting with a dot, so a fixed `Substring(2)` ate the `.g` of
  `.github`. Seven of the fourteen per-plant cases named it at once. A file set
  only shows in the output when something in it fails.
- ⚠ The Linux lane went from 13.4 minutes to 24.4 of the 30 it had, because each
  new plant runs the whole gate. Its budget is 45 now with the measurement beside
  it; a lane that runs out of time reports as infrastructure, not as a defect.
- Deployment: nothing deployed. Three fixture captures have run, no client was
  installed, no record was published and nothing was written to the data branch.

### 2026-09-08T12:00:55Z

- `CI-06` opens and the capture workflow is dispatched for the first time.
  Record: [`TODO/ci.md`](TODO/ci.md).
- ⭐ Capture run 1 on `b992a35`: the Linux job green end to end, its bundle
  verified under `sha256sum -c`, its transcript carrying the announce `curl`
  made and the run-id token the driver knows it sent. The attestation says
  `kind=fixture`, `measured_build=none`, `stock_client=false`.
- ⭐ `Get-NetRoute`'s real output matches the fixtures `check-runner.ps1` proves
  the Windows guard against, which no fixture could ever establish.
  [`docs/capture-host.md`](docs/capture-host.md) carries what was measured.
- ⛔ The Windows job went red on *Restore the route* and the restore had worked.
  The step ends by running the egress guard inverted - a host that can reach the
  network again is one that guard refuses - and that refusal's exit code was left
  in `$LASTEXITCODE`, which GitHub's `pwsh` wrapper reads as the step's verdict.
  A guard succeeding at its job failed the step and the upload was skipped.
- ⭐ Both restore steps end in an explicit `exit` now, so a step's status is a
  decision rather than a residue. The `sh` twin never had the defect, because
  `if …; then` consumes the status.
- ⛔ And the `sh` twin folded the guard's `2` into its `1`: *could not run* read
  as *refuses*, so a guard that never reached a routing table would have looked
  like one that read a restored route off it. Both halves read all three codes.
- ⛔ `check-project`'s PowerShell rules iterated `git ls-files '*.ps1'` and never
  reached a `pwsh` block inside a workflow, which is the same language with the
  same hazards. `capture.yml` was carrying a live instance of each: a
  `Write-Error`, and three blocks setting `$ErrorActionPreference = 'Stop'` with
  nothing saying what a native exit code means.
- ⭐ Both rules read every workflow's `pwsh` blocks now, and a third rule lives
  only there: a block that reads `$LASTEXITCODE` ends in an explicit `exit`.
- Deployment: nothing deployed. What was captured is a fixture, no client was
  installed, no record was published and nothing was written to the data branch.

### 2026-09-08T07:07:54Z

- `CI-03` closes: the capture workflow, the two capture runners it calls, and
  the harness that mutation-proves them. Record: [`TODO/ci.md`](TODO/ci.md).
- ⛔ The step order is the containment, and `check-workflow.sh` asserts it.
  Building after the route is deleted cannot work; capturing before the guard
  measures a host nobody established was contained; uploading before the restore
  is a step that cannot reach GitHub. Each reads as plausible in a diff and none
  was visible to any other check here.
- ⛔ Every `scripts/…` path a workflow names must be in the tree. That is the
  rule that would have caught the draft deleted last session, which called two
  scripts nobody had written.
- ⛔ `capture-run` builds nothing, so a workflow whose build moved after the
  containment is refused by name rather than by a cargo invocation reaching for
  a network that is gone. It re-reads the claim marker and the routing table
  itself, because a step order that dropped the claim would otherwise capture on
  a host nothing claimed.
- ⭐ The driver is `curl` and the verifier is `sha256sum -c` or `Get-FileHash`,
  and the announce carries a token the driver knows it sent which the transcript
  must hold. Without that last one, a bundle of empty artifacts verifies against
  its own empty digests.
- ⛔ Adding `[switch]$Marker` beside an existing `$marker` local made every
  invocation of the Windows guard fail to bind: PowerShell variable names are
  case-insensitive, so the parameter and the local were one variable. Found by
  running it once.
- ⛔ `awk '{ print; fflush() }'` streams nothing: mawk reads its input in blocks,
  so there was no output to flush yet. Six harness cases reported the driver's
  refusal instead of their own until the splitter became a `read` loop.
- ⛔ Twelve declared rows on the PowerShell gate lane said `CI-03 residual`,
  and `CI-01` said there were six of them and that they close when `CI-03`
  lands. `CI-03` has landed and not one closed: they are `sh` harnesses needing
  a PowerShell half. Each row names the event that would actually close it now,
  and the count is no longer written down anywhere.
- ⛔ A door sweep found the UTF-8 BOM rule for `.ps1` held in eleven files and
  broken in four, all four carrying markers on hundreds of lines. Both halves of
  `check-project` refuse it now, and the four files carry the BOM.
- ⛔ Those two halves disagreed on their first run and `check-twins` could not
  have seen it: a `.ps1` keeps CRLF, the sh half read the carriage return as
  non-ASCII, and no `.ps1` in this tree is ASCII-only, so the branch that
  differed had nothing to exercise it.
- ⛔ CI run 58 went red on the Linux lane and green on Windows, over a default
  that changed: `$PSNativeCommandUseErrorActionPreference` is `$false` in
  PowerShell 7.4 and `$true` from 7.5, where a native command's non-zero exit
  becomes a terminating error under `$ErrorActionPreference = 'Stop'`. A guard
  that REFUSES stopped being a code the caller could read. Sixteen `.ps1` files
  relied on that default, one of them in a comment stating it as a guarantee;
  all sixteen set it explicitly now and `check-project` refuses a `.ps1` that
  does not.
- ⛔ That run also showed a red gate that did not say what failed: the excerpt
  was the first twelve lines of a harness that prints its failures last. The
  excerpt is the tail now and `store_report` reprints the failing rows above its
  summary.
- ⛔ CI run 59 was red again, and its log named the failing case in one line,
  which is the reporting fix above working. The cause: `Write-Error` inside a
  script renders a source-context block and wraps the message to the host's
  width, so a refusal a harness matches on is one line at width 200 and two at
  width 80. Ten `.ps1` files already wrote through `[Console]::Error.WriteLine`
  and five did not; all five do now and `check-project` refuses the rest.
- ⚠ That rule's needle fired on its own enforcing file on its first run, because
  the failure message contains the name. It matches an invocation now, not the
  word.
- ⭐ The two remote read routes are absolutes rather than preferences, and a new
  absolute says this document's routes beat a general habit: a session polled CI
  through a general-purpose tool while rule 8's route answered the same question
  in one `curl`.
- ⭐ CI run 60 is green on both lanes, first attempt after both fixes.
- ⚠ The capture workflow has never been dispatched, and what it would capture is
  a fixture: nothing installs a client, and the attestation says so in fields.
  `CI-06` is the entry that dispatches it.
- Deployment: nothing deployed. No capture was taken and nothing was published.

### 2026-09-08T04:49:30Z

- The Windows disposable-host guard pair and its mutation harness, which is the
  part of `CI-03` that does not need a runner. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⛔ A capture host was never the blocker nineteen entries were recorded as
  waiting on. A hosted runner is a fresh VM per job, and the egress guard reads
  `/proc/net/route` and refuses only because a default route exists. Measured
  with the guard's own route-table argument, which made it settleable in one
  command the whole time.
- ⛔ Writing the harness found two defects in the guard on its first run.
  `break` inside `ForEach-Object` unwinds the script rather than the pipeline,
  so every misuse printed usage and exited 0; and a single-match pipeline yields
  a scalar under `Set-StrictMode`, so the mode check threw.
- ⚠ Both address families are checked. A host with IPv4 unplugged and IPv6 up
  still reaches the internet.
- ⭐ `check-runner` is a real row on the PowerShell gate lane now rather than a
  declared gap.
- ⛔ No repository owner or name is hardcoded anywhere in the tree. The project
  moves to another owner once it is finished here, so workflows derive it, the
  issue template uses a repository-relative path, and the README names no clone
  URL.
- All four operator decisions are settled and recorded, so no entry is blocked.
- Deployment: nothing deployed. No capture was taken and nothing was published.

### 2026-09-08T04:19:03Z

- `DOC-02`, the contributor handbook and the walkthrough that runs it. Record:
  [`TODO/documentation.md`](TODO/documentation.md).
- ⛔ The harness binds four names and runs the page's own blocks and nothing
  else, which is what makes "without undocumented steps" a measurement rather
  than a claim.
- ⭐ Running it found a documented build command that uses the wrong compiler:
  `rust-toolchain.toml` is found by walking up from the working directory, so
  the same `cargo build` from elsewhere silently uses another toolchain.
- ⛔ Two of the harness's own checks could not fail. The page's validation steps
  could be deleted while everything stayed green, and the harness's own
  validation could be replaced by `true`. Both are closed; a third survivor is a
  weakened assertion inside the harness and is recorded rather than fixed.
- ⚠ The host guards are shown and never run: a session host is refused by the
  egress guard, and `--claim` would write a capture marker onto this machine.
  The harness has a case asserting the extractor left that block alone.
- Deployment: nothing deployed. No capture was taken, nothing was published, and
  the walkthrough begins where a contributor's own capture would end.

### 2026-09-08T03:55:44Z

- `DOC-01`, the reader's page and the harness that runs it. Record:
  [`TODO/documentation.md`](TODO/documentation.md).
- ⛔ Every shell command on `docs/consuming.md` is extracted from the page and
  executed against a fixture publication. A documented command that is a copy of
  a tested snippet is a copy that drifts.
- ⭐ It paid on its first run: an example written as `cmd && exit 1` exits on the
  failure it was demonstrating, under the `set -e` the harness runs blocks with.
- ⚠ The fence language is the whole selection rule, and it is checked against
  itself: a run that took every fenced block is a run whose rule has stopped
  applying, and it fails.
- ⚠ The field reference is a pointer rather than a generated copy. A second copy
  of the record shape is the hazard `check-one-home` exists to refuse.
- ⛔ `LIB-02` is blocked on repository access, measured rather than assumed: its
  Prove names `bit-cli`'s own tests and that repository is not reachable here.
- Deployment: nothing deployed. No URL was fetched and nothing was published;
  the page says so where a reader would otherwise assume otherwise.

### 2026-09-08T03:18:05Z

- `LIB-01`, the Rust consumer library. Record:
  [`TODO/library.md`](TODO/library.md).
- ⛔ Nothing in the crate can reach a network, and that is swept for rather than
  promised. A catalogue is opened over bytes the caller holds; retrieval is a
  trait the consumer implements while the verifying stays in the library.
- ⚠ The sweep's needle list is checked against `bit-ids-lab`, which really does
  carry sockets, because needles that have stopped matching report the same
  clean answer over a crate full of them.
- ⛔ `Indexes::to_json` and `Release::manifest_json` both wrote a document
  nothing could read back. The manifest gap was a live defect: without a reader,
  a consumer compares a bundle against a manifest re-derived from that bundle
  and agrees with itself, so two refusals could never fire.
- ⛔ Two guards answering one code masked each other, and a mutation pass is what
  showed it. They are separated by the path a refusal names now.
- ⭐ One plant that survived was the design working: `Profile`'s hand-written
  `Deserialize` means the generic serde route validates as well.
- ⚠ A 64-digit hex literal in a test was refused by `check-no-secrets --public`,
  correctly. The absent identifier is derived rather than typed.
- Deployment: nothing deployed. No network was reached, nothing was published,
  and the "embedded minimal indexes" half of the Approach waits on there being a
  published catalogue to embed.

### 2026-09-08T02:28:10Z

- `PUB-04`, the documented access paths and their stability rules. Record:
  [`TODO/publishing.md`](TODO/publishing.md).
- ⛔ A consumer is told two things per path and both are derived. Stability comes
  from `store::CANONICAL_ROOTS`, which is already what the append-only rule is
  about, so a path is immutable exactly when the publisher refuses to rewrite it.
- ⛔ Exactly one published file is covered by neither `MANIFEST.json` nor
  `SHA256SUMS`, and the contract names it per path rather than leaving a reader
  to find where the gap is.
- ⛔ `routes/v1/<target>/<version>/<platform>/<arch>.json` is out of the
  published layout. Nothing ever wrote it, and it cannot be written in that
  shape: the path omits `<package>` while the acquisition routes differ per
  package. Found by deriving the documented set from an assembled release.
- ⛔ Three of the five refusals were unreachable through the deriving path,
  because `contract` derives stability, integrity and order and the two halves
  agree by construction. They are reachable from a file, so they have
  document-level cases now.
- ⚠ A mutation harness reported a false SURVIVED before that was believed: its
  test selection excluded the integration target the new cases live in.
- ⭐ The immutability rule is proved by two publications rather than by a label.
  A record published, fetched back over git, and still byte-identical after a
  second publication is a comparison; a document asserting immutability is not.
- ⚠ A store test named a `routes/` path as its example of a derived file. It
  still passed and illustrated nothing, so it names `LICENSE` now.
- Deployment: nothing deployed. No GitHub URL was fetched and nothing was
  published; every case ran against a bare repository in a scratch directory.

### 2026-09-08T01:39:27Z

- `CI-02`, the stable-release staleness monitor. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⛔ A capture request's identifier is a digest of its key, never an allocated
  token. Two runs over the same facts derive the same identifier, so a tracker
  keyed on it cannot hold two; that is the whole of "no duplicate after repeated
  runs". Length-prefixed and domain-separated, the way a record identifier is.
- ⚠ Architecture and package are not in the key. Both are outcomes of the
  acquisition and are unknown when the request is opened, so a key carrying them
  would multiply one release into a request per packaging.
- ⭐ `survey` takes the whole `Resolution` rather than a version, so nothing here
  judges stability and a preview cannot reach a request. A second stability rule
  would be a second place for the answer to differ.
- ⛔ Some verdicts decline to answer, and none of those is a silent skip.
  [`TODO/ci.md`](TODO/ci.md) names each one and what would make it fire.
- ⛔ The mutation pass found a test whose name claimed more than it checked.
  Dropping the platform from the key, and replacing the length prefixes with a
  separator join, both left every Rust case green: the collision test they
  slipped past holds for one pair under one separator, and nothing varied the
  platform. The encoding is pinned against its restated specification now.
- ⭐ The request identifier is re-derived by `python3`'s SHA-256 in the harness,
  because a survey compared against its own encoder agrees with itself.
- ⚠ `support/scheme.rs` split out of `support/reader.rs`. The first example to
  need a store reader and no scheme parser compiled one nothing called, and the
  dead-code lint is what said so.
- Eighteen plants over `staleness.rs`, seventeen refused; the eighteenth did not
  compile as first written, which is "could not run" and never "refused", and it
  was rewritten until it did and then refused.
- Deployment: nothing deployed. No capture was taken, nothing was published, and
  no resolver was scheduled: the monitor's comparison and its driving surface
  exist, and the trigger that would open a tracked issue is a named residual.

### 2026-09-06T23:10:00Z

- `OBS-06`, the adjacent protocol observer suite, over local discovery and peer
  exchange. Record: [`TODO/observer.md`](TODO/observer.md).
- ⛔ The lab had no egress guard and a door sweep is what found it. Every socket
  went through `bind.rs`; every send did not. `endpoint::serve_datagram` answered
  on the bound socket with the source address the sender wrote on the packet, and
  nothing verifies a UDP source address. `bind::send_to` is the one door now.
- ⚠ The sweep's needle list is why it lasted: every needle on it named a
  constructor, and a send is a method on a socket that already exists, so the
  whole category was missing rather than one entry.
- ⚠ A send guard checks before the syscall and cannot check after. A bind reads
  `local_addr` and a dial reads `peer_addr`; a datagram socket reports nothing
  about the packet it sent.
- ⭐ An adjacent surface is behind a value that has to be constructed, not a flag
  whose default is false. `Capability::enable(surface)` is the only constructor,
  and a boolean default is what a later `..Default::default()` flips.
- ⛔ A second `Surface` enum was written for the lab and removed.
  `bit_ids::observation::Surface` already named four of these, so the lab
  re-exports the record vocabulary and `local_discovery` was added to it.
- Three surfaces split out as `OBS-11`: message stream encryption, the DHT and
  web seeding are each an `L` on their own and `TODO/RULES.md` says to split
  before execution.
- Twenty-three plants, twenty-two refused. The two survivors were findings: a
  test named for a guarantee it did not check, and one guard nothing can refute,
  which is kept with the reason written where it is.
- Deployment: nothing deployed. No capture was taken and nothing was published.
  The lab never joined the multicast group BEP 14 names; the driven run sent
  three announces to a loopback address from a client on the same host.

### 2026-09-06T21:45:00Z

- `ACQ-05`, the artifact cache and its authenticity evidence. Record:
  [`TODO/acquisition.md`](TODO/acquisition.md).
- ⭐ The second half of the Prove asks for less than it sounds like. Reproducing
  an artifact's identity after a source URL change needs no reproduction once the
  identity is the digest: a retrieval from a new location is recorded against the
  artifact the digest already names, and the lookup never mentions a URL.
- ⚠ A repeated retrieval adds nothing. A cache that grew a row every time
  somebody re-ran an acquisition would report a popularity contest rather than a
  provenance.
- ⛔ Keeping the bytes is a permission and not a capability. `E-CAC-01` refuses
  stored bytes where the register refuses them, and `E-CAC-02` refuses a target
  the register does not mention rather than defaulting to permitted.
- ⛔ The register is asked, never re-read. `check-licences` gained `--permitted`
  on both halves, the Rust cache takes a disposition map, and `check-cache.sh`
  fills it from that call, so the tie is one parser rather than two readings.
- ⛔ The refusal case needed a control and the harness carries it: the same
  scenario is run with a target explicitly permitted, and the two runs are
  asserted to differ on exactly one line. Without it the policy half would pass
  equally over a cache that can never store anything.
- Nine plants over `cache.rs`, all refused, each verified to compile first.
- Deployment: nothing deployed. No capture was taken. Nothing was published, and
  nothing was fetched: the scenario's bytes are generated and are nobody's
  installer.

### 2026-09-06T19:30:00Z

- `FOUND-04`, the third-party licence and redistribution register. Record:
  [`TODO/foundation.md`](TODO/foundation.md).
- ⛔ The measurement is the finding: six of the nine targets with a GitHub
  upstream answer `NOASSERTION` when their licence endpoint is asked, so a
  detector cannot name one. Those rows say `unverified` and name who was asked,
  rather than carrying an identifier nobody established.
- ⭐ The twenty-two dependency rows are read out of each package's own manifest
  at the version the lockfile pins. `libc` is not built for this host's target,
  so its manifest was fetched before it was read rather than assumed from its
  siblings.
- ⛔ Every row refuses redistribution, which is the policy and not a consequence
  of the licences. `permitted` exists so the check can refuse it over an
  unverified licence or with no notice.
- ⛔ The twin comparison found a defect a clean tree could not show. On an empty
  register the two halves disagreed, because `grep -c .` prints `0` and exits 1,
  so a `|| printf 0` fallback made the count two zeroes on two lines and the
  guard that refuses a register of nothing was disabled by exactly the input it
  exists to catch. `wc -l` replaces it.
- `check-licences` joins `check-twins` and both gate runners as the tenth pair.
- Deployment: nothing deployed. No capture was taken. Nothing was published.

### 2026-09-06T17:05:00Z

- `PUB-03`, the multi-format publisher, closed over four of its five renderings.
  Record: [`TODO/publishing.md`](TODO/publishing.md).
- ⛔ The SQLite rendering is split out as `PUB-05` and is blocked on an operator
  decision rather than on work. Both routes to it cost something this project has
  been deliberate about; [`docs/supply-chain.md`](docs/supply-chain.md) carries
  the argument and the entry carries the recommendation. Nothing was dropped and
  the split is recorded.
- ⛔ Every rendering is a function of the canonical document. The combined JSON
  carries each record's own bytes verbatim, so slicing one out yields exactly
  what was published and digested; JSONL and CBOR are produced by reading that
  document back; the tabular cells are read out of it by pointer.
- ⛔ Which records are rendered is `CORPUS-04`'s answer and not a second filter.
  A renderer that selected on its own would have kept publishing a retracted
  measurement in the table, which is the rendering a reader is least likely to
  cross-check.
- ⭐ The CBOR encoder is written here and checked against `cbor2` 6.1.4 rather
  than trusted, and not by a round trip: that reader's own canonical encoding of
  what it read is byte-identical to ours. Two implementations of RFC 8949
  section 4.2.1 agreeing is a stronger result than one agreeing with itself.
- ⚠ `formats/bit-ids-v1.columns.json` publishes the columns and the seven record
  sections no row can hold, so a consumer reading only the CSV can discover what
  it is not being told.
- ⛔ A test got something wrong, correctly. It first asserted the corrected
  record's identifier appears nowhere; that is false by design, because a
  correction names what it corrects. The rule is that it is not published as a
  record, so identifiers are compared rather than text searched.
- `check-formats.sh` joins the `sh` gate as the seventh mutation prover, and the
  two examples now share one store reader rather than growing a second copy.
- Deployment: nothing deployed. No capture was taken. Nothing was published.

### 2026-09-06T14:40:00Z

- `CORPUS-04`, supersession and correction records. Record:
  [`TODO/corpus.md`](TODO/corpus.md).
- ⚠ Half the Approach already existed: `supersedes`, `adjudication` and
  `E-CRP-07` were all in place. What none of them did was change an answer, so a
  corrected record was still in every lookup and could still be the latest view's
  reply.
- ⛔ A superseded record now leaves every view and stays in the store, and the
  two counts are kept apart: an excluded record was never publishable, a
  superseded one was.
- The `corrections` list carries the superseded identifier, the record that
  directly corrects it, and the end of the chain. ⚠ The last two differ exactly
  when a correction was itself corrected, which is the case a single-step row
  answers wrongly while looking right.
- ⚠ Only a publishable correction retracts anything. A provisional one would
  otherwise leave the build line answering nothing at all.
- ⛔ A fork is refused rather than resolved, and the chain walk is bounded
  because a cycle is constructible: a record identifier digests the identity
  tuple and not `supersedes`, so two records can name each other while neither
  supersedes itself.
- ⭐ The retention half needed two stores rather than one. Finding the original
  still there proves it was written, not that it was left alone, so
  `build-store --correct V` writes the correction beside it and the harness
  compares that record's bytes between a store built with the correction and one
  built without.
- ⛔ The claim audit corrected this entry's own draft. It was about to record
  that a cycle is a store the corpus validator accepts; asserting that instead
  of writing it down showed the validator refusing the store for an unrelated
  reason. The residual is real and the reason was wrong.
- Deployment: nothing deployed. No capture was taken. Nothing was published.

### 2026-09-06T12:15:00Z

- `CI-01` closed, the complete cross-platform quality gate, and with it the last
  open `P0`. Record: [`TODO/ci.md`](TODO/ci.md).
- ⭐ The Windows lane reports **zero skipped** on CI run 37 under `-Strict`: ten
  passed, none failed, and seven rows named unavailable with the entry that owns
  each. That is the Prove's first half measured rather than argued.
- `.github/workflows/publish-data.yml` carries the job-scoped `contents: write`
  and the concurrency group `PUB-02` left as residuals. ⛔ It has no automatic
  trigger and its dry run is the default, so it cannot fire on its own and a
  dispatch by accident still pushes nothing; both are cases in the harness.
- ⚠ Its group does not cancel in flight, unlike the gate's. Cancelling a gate run
  costs a rerun; cancelling a publisher between its append comparison and its
  read-back leaves a branch nobody has verified.
- ⚠ The token reaches git as a header rather than inside a remote URL. The first
  version wrote it into the URL and `check-no-secrets` refused the file, which is
  the guard working rather than an obstacle.
- ⛔ That workflow has never run and cannot succeed today: its first step wants
  the assembled bundle of a capture run, and there are no captures.
- The harness now covers every workflow rather than the one it was written for,
  and its scratch tree carries the origin remote, without which
  `check-remote-items` could not run there and the gate cases never saw a clean
  tree exit 0.
- Deployment: nothing deployed. No capture was taken. Nothing was published. No
  remote was written other than this repository's `main`.

### 2026-09-06T10:30:00Z

- `CI-01` in progress, the cross-platform quality gate. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⛔ The Windows lane ran without `--strict` and could not do otherwise: six of
  its rows are checks that platform cannot run, so the flag would have refused
  every correct tree. A check that stopped running was therefore counted beside
  them and the lane stayed green. Measured by rewriting `check-project.ps1` to
  `exit 2`, which that lane's own invocation exited 0 over.
- Both gate runners now count a declared unavailability apart from an observed
  skip. A declared row carries its reason and prints as `n/a`; `--strict`
  refuses only the observed kind, so both lanes ask strictly now.
- `scripts/ci/check-workflow.sh` plants a defect of each class into a scratch
  copy of the working tree and runs the offending workflow step against it.
  ⭐ Every command it runs is read out of the workflow by job and step name, so
  it cannot drift from CI and a step that has gone is reported rather than
  passed over.
- ⛔ It is deliberately absent from `check-gate.sh`: two of its cases run the
  gate, and a runner listed in the gate that also runs the gate re-enters
  itself. The workflow calls it as a step of its own.
- ⛔ The door sweep found one assumption behind two doors. `store_build` and
  `publish-data.sh` each composed an example path as `target/debug/examples`
  while cargo obeys `CARGO_TARGET_DIR`, so on a host exporting that variable all
  five corpus and publishing provers exited 2 and the publisher refused to run.
  Exit 2 is a skip to the gate, so the tier proved nothing and said so nowhere.
- The workflow declares per-job permissions and timeouts, caches downloaded
  crates and build artifacts under separate keys, and pins `actions/cache`.
- Deployment: nothing deployed. No capture was taken. No remote was written
  other than this repository's `main`.

### 2026-09-06T09:25:41Z

⚠ **This entry is newer than the seven above it and sits below them, because
their stamps are not machine-read.** `docs/conventions/git.md` section 3 asks for
`date -u`; the entries dated 2026-09-06T10:30Z through 23:55Z were written by
hand while their commits were made between 2026-09-05T17:42Z and 23:37Z. This
stamp is the machine's. File order here is stamp order and is not work order,
and the older stamps are not retro-corrected: rewriting somebody else's record of
when they worked is worse than the gap it would close.

- `OBS-11`, message stream encryption, DHT and web-seed observers, closed. The
  observer layer now covers every surface a build reaches for. Record:
  [`TODO/observer.md`](TODO/observer.md).
- ⭐ A datagram responder is handed the source address of the packet it answers,
  which `OBS-06` carried over. `implied_port` in BEP 5 means *use the source port
  of this packet*, so an observer blind to it cannot record what port a build
  announced. The address reaches the record and never a syscall.
- ⛔ **A third door, and it is not a socket.** A DHT `values` list, a BEP 19
  `url-list` and a tracker's peer list all hand the build addresses it dials
  *itself*, so `bind::send_to` is never called on those packets and no guard on
  this project's sockets can see them. `bind::check_offered` is the guard.
- ⛔ **A door sweep found the same hole in the two oldest observers.**
  `OfferedPeer`, which both tracker surfaces hand a build, had public fields and
  no check at all since `OBS-02`. Its fields are private now and `OfferedPeer::new`
  is the only constructor.
- ⚠ The hazard had been written down against the wrong surface: `adjacent::reaches`
  said `pex` hands out addresses a client will then dial and said nothing of the
  kind about `dht`, which does the same through a different field.
- ⭐ `bit_ids_wire::dht` reads BEP 5's KRPC and keeps the whole decoded document,
  so key order, transaction-id width and integer spelling survive a re-encode.
  `Surface::Dht` has two fixtures and is no longer refused with `E-FIX-07`.
- ⛔ The `E-FIX-07` negative control named `dht` as *the surface with no codec*,
  in two places, which stopped being true the moment the codec landed. Both name
  `mse` now.
- ⚠ "Peer ID" was the wrong name for what the corpus guard reads, since a KRPC
  message carries a *node* id, and nothing checked a version string at all until
  `dht` put a `v` on a second surface. Both are checked now.
- ⭐ `bit_ids_probe::web_seed` answers BEP 19, where the identity belongs to the
  build's HTTP library rather than to the build. `TorrentSpec` carries
  `web_seeds`; an empty list writes no key, so no recorded digest moves.
- ⭐ `bit_ids_wire::mse` and `bit_ids_probe::mse` complete the entry. MSE comes
  first or not at all, so the offer is a condition of the measurement, and the
  peer ID read back out of `IA` is `OBS-04`'s measurement through a second door.
  The 768-bit arithmetic is written out and added **no package** to the lockfile.
- ⛔ A mutation pass found `VerificationFailed` unreachable: the only case
  exercising a wrong key relies on random plaintext, which trips the pad-length
  check first. The case that reaches it is a build that keys correctly and writes
  the wrong constant.
- ⭐ Each module has a driven run against an outside implementation: `curl`
  8.5.0, a `libtorrent`-encoded KRPC exchange, and an MSE initiator written from
  the specification in Python. ⛔ None is a stock client, which is unchanged and
  is `OBS-07`'s to fix.
- ⛔ `synthetic-torrent` defaulted its output into the repository root, so running
  it turned `check-licences` red. The path is required now.
- ⚠ `check-no-secrets --public` gained an allowance for the MSE test vectors,
  anchored on the `MSE_` name, on both quotes and on exactly 192 digits. Five
  planted inputs, both twins agreeing on every one.
- Deployment: nothing deployed. No capture was taken and nothing was published.


### 2026-09-05T15:00:00Z

- `PUB-02`, the append-only data branch publisher. Record:
  [`TODO/publishing.md`](TODO/publishing.md).
- ⛔ Driving it found that the append rule and the derived files collided: a
  correct second publication changes `MANIFEST.json`, `SHA256SUMS` and the
  indexes by design, and treating every published path as immutable refused it.
  `store::CANONICAL_ROOTS` now names the roots the rule is about.
- The append comparison runs before the push, not after, because a branch
  protection setting refuses a force and says nothing about a commit that deletes
  a file. A refusal ends the run with nothing pushed and the branch where it was.
- No force, asserted three ways: no flag, a branch name carrying `+` or `:` is
  refused before a refspec exists, and the harness reads the publisher's own
  source with the comments stripped. ⚠ The first version of that source check
  matched the header sentence explaining the rule and reported the guard broken.
- ⭐ That git refuses a divergent push under a plain refspec is measured on this
  host rather than taken from the manual, because the publisher's whole
  non-fast-forward defence rests on it.
- ⛔ The acceptance touches no real remote: it creates a bare repository in a
  scratch directory and deletes it. The publisher has never run against this
  repository's own remote and will not until a measured record exists.
- ⛔ The closing mutation pass found that the read-back could not catch a remote
  that discarded the push: it compared the fetched tree only against the prior
  one, and a rewound ref appends to the prior tree because it is the prior tree.
  The publisher now compares against the bundle as well, and a `post-receive`
  hook that rewinds the ref is a case in the harness.
- Deployment: nothing deployed. No capture was taken. No remote was written
  other than this repository's `main`.

### 2026-09-05T14:20:00Z

- `PUB-01`, the deterministic release assembler. Record:
  [`TODO/publishing.md`](TODO/publishing.md).
- Two documents describe the bundle and cover different sets, because neither
  can state its own digest: `MANIFEST.json` describes everything except itself
  and `SHA256SUMS`, and `SHA256SUMS` covers everything except itself.
- ⭐ The media-type rule paid on its first driven run, refusing a real evidence
  bundle over `fixture/generated.torrent`, which every capture writes and the
  table did not carry. A default would have shipped it as opaque bytes.
- ⭐ The strongest control is a reader this project did not write: `sha256sum -c`
  agrees with every row, so a run that agreed with itself about what it wrote is
  still caught.
- ⚠ The archives and databases in that entry's Prove are `PUB-03`'s. This
  assembles the tree once and describes it; the renderings inherit the
  determinism rather than each recomputing it.
- Guard mutation: 10 plants over the assembler, 9 refused. `entries.sort()` is
  recorded as unrefuted with what would have to change for it to fire, the same
  shape and the same treatment as `CORPUS-03`'s `latest.sort()`.
- Deployment: nothing deployed. No capture was taken.

### 2026-09-05T13:40:00Z

- `CORPUS-03`, the deterministic indexes and latest views. Record:
  [`TODO/corpus.md`](TODO/corpus.md).
- Six lookups and one latest row per build line, every row naming the record it
  came from. `rows_resolve` checks them against the store rather than against the
  builder that emitted them.
- ⭐ The peer-prefix lookup inverts the rule the codecs hold rather than waiving
  it: its key is the fixed span of a peer ID this project measured.
- ⚠ `VersionScheme::components` is now public, so the resolver and the latest
  view share one version ordering. A target with no declared scheme blocks the
  view rather than being ordered under an assumed shape.
- ⭐ The acceptance caught a real defect before the mutation pass did: the
  ranking carried a `Version` comparison beside the numeric one, and `Version` is
  ordered as text, so `1.2.9` beat `1.2.10` and the view answered a superseded
  build. One total key replaced it.
- ⛔ The first mutation round refused two plants of eight. Two survivors were bad
  plants; four were real gaps, now closed: the sort was invisible because the
  store was read in one order, nothing excluded a provisional record, the
  varying-first peer-ID rule was never reached, and `E-VIW-10` was asserted where
  it could not fire.
- ⛔ A harness overwrote the shared library's row accumulator with its own
  variable and printed ten passes over eight lines. The globals are prefixed and
  `store_report` now compares the rows it holds against the count it prints.
- ⛔ `validate_corpus` reported a store of nine orphan artifacts as valid: the
  bundle sweep ran per manifest and there was no manifest to run it. It sweeps
  the whole tree now, and `E-CRP-08` covers the record root.
- `build-store` takes `--version`, so a store can be written at a version a
  scheme can order; the fixture's own is deliberately not one.
- Deployment: nothing deployed. No capture was taken.

### 2026-09-05T12:55:00Z

- `CORPUS-02`, the semantic corpus validator. Record:
  [`TODO/corpus.md`](TODO/corpus.md).
- ⛔ Evidence reachability was the only invariant in that entry's Problem that
  nothing could already answer. `bind` compares the profile against the manifest,
  so two documents agreeing about an artifact nobody wrote satisfied every check
  this project had. `E-CRP-03` resolves a citation against the store instead, and
  `E-CRP-04` to `E-CRP-06` compare the stored bytes with what the run declared,
  in both directions.
- ⚠ Route count, connector independence, field provenance, agreement and stable
  status were already enforced per record. The entry says which code holds each,
  rather than adding a second spelling of it.
- ⭐ `examples/build-store.rs` generates the golden corpus the acceptance needs,
  writing the artifacts first and then each document to describe the bytes
  actually put down, out through `to_json`. The entry says why a committed one
  could not serve.
- `validate_corpus` refuses what must hold of any store; `publishable_view`
  separately reports which records may enter a published view, because a store of
  provisional records is a correct store.
- ⛔ The entry's `Prove` selected one integration binary with `--test`, which
  skips the library's own tests. Rewritten and the original recorded, per the
  residual `CI-05` left.
- ⛔ A review plant reported a refusal that was not one: the harness counted a
  harness exit of 2, *could not run*, as a refusal on its filesystem path. It now
  separates them and the plant was rewritten to compile.
- ⚠ `shellcheck` answers differently depending on how the files are grouped on
  its command line, so the two harnesses carry their own directives rather than
  depending on CI passing every script at once. The shared helpers moved to
  `scripts/corpus/store-lib.sh` rather than being copied.
- Guard mutation: 8 plants over the corpus validator, 8 refused;
  `check-corpus.sh` plants 9 defects against a real store and refuses all 9, with
  4 harness self-guards.
- Deployment: nothing deployed. No capture was taken.

### 2026-09-05T12:10:00Z

- `CORPUS-01`, the append-only canonical store. Record:
  [`TODO/corpus.md`](TODO/corpus.md).
- A record's path is derived from the identity tuple `RecordId` digests, in full,
  and a published path never changes or disappears. `E-STO-*` carries both
  halves: the structural rules a tree must satisfy to be checked out at all, and
  the comparison between a published tree and the successor a run proposes.
- ⛔ The published layout was not injective over the identity tuple. A profile
  was filed with no `package` segment while the tuple carries one, so a `deb` and
  an `AppImage` of one version on one platform were two records at one file name.
  [`docs/publishing.md`](docs/publishing.md) is amended and a test pins it.
- ⛔ `Version` accepts `../../etc`, measured rather than assumed, because a
  version string is what the build printed. The store refuses a version that
  cannot be a path segment rather than escaping it, since a non-injective escape
  merges two measurements into one directory.
- ⭐ The driven pass found what the suite could not: the placement reader
  selected a record by name and opened it, so a named pipe blocked it forever
  while `validate_tree` already refused that entry kind. One action, two doors,
  one gate. The reader now takes the kind from the walk.
- ⛔ The door sweep found a second spelling of the layout, in the recogniser that
  decides whether the placement check runs at all. A layout change would have
  left it recognising nothing, with the suite still green. The recognisers moved
  beside the composer and a round-trip test closes the loop.
- Guard mutation: 10 plants over the store source, 10 refused; 9 planted against
  a real filesystem, 9 refused; 4 harness self-guards exercised. ⚠ `grep -F`
  splits a multi-line pattern into alternatives and miscounted three plants as
  ambiguous, which fails safe and still leaves guards unproven; the committed
  harness refuses a multi-line literal outright.
- `scripts/corpus/check-store.sh` joins the `sh` gate and is a named skip on the
  PowerShell half, because its plants are a symbolic link and a named pipe.
- Deployment: nothing deployed. No capture was taken.

### 2026-09-05T11:20:00Z

- The session's four closing reviews. Record:
  [`docs/history/SESSION-2026-09-05-EVIDENCE.md`](docs/history/SESSION-2026-09-05-EVIDENCE.md).
- ⛔ The door sweep found the capture path untested as a whole. Each leg was
  covered and nothing drove a torrent through an observer into a bundle, which
  is the composition class `gate.md` names.
  `crates/bit-ids-probe/tests/generated_torrent.rs` closes it, and the two
  plants written for it are refused.
- `REDACTED` was public with no reader outside its own module. The acceptance
  suite now asserts the constant and its literal, so the placeholder a bundle
  reader recognises cannot drift and the constant is not surface with no
  consumer.
- ⭐ The claim audit's fourth lens paid: the qBittorrent release listing
  answered with four releases and that is the source's own shape, not
  truncation. Page two is empty and `tags` answers with at least a hundred, so
  a resolution reading only `releases` selects from a much smaller population
  than the target's versions. Recorded in [`TODO/clients.md`](TODO/clients.md).
- `TODO/RULES.md` now states that `Closure evidence` is a dated measurement and
  a `Prove` is a live command, which is the distinction `CI-05`'s check encodes
  and nothing had written down.
- One code comment argued from platform folklore where the acceptance suite
  builds the case; it now points at the test.
- ⛔ The Windows CI lane is red on `f9239a5` at *Install pinned Rust toolchain*,
  a TCP connect timeout to `static.rust-lang.org` before any repository code
  ran, with every later step skipped and the Linux lane green. Not a defect in
  the change, and not evidence about the tree either.
- Deployment: nothing deployed. No capture was taken.

### 2026-09-05T10:55:00Z

- ⛔ Moved the client entries behind `CORPUS-01` in the work order, on a
  measurement rather than a preference. A client acceptance needs a capture, a
  capture needs a host `assert-disposable.sh --egress` does not refuse, and a
  session host is refused; the Windows guard pair does not exist at all. Record:
  [`TODO/clients.md`](TODO/clients.md).
- ⭐ The resolver met a real target for the first time.
  `fetch-releases.sh qbittorrent/qBittorrent` answered through
  `api.gh.pkgforge.dev` and `resolve-stable` selected 5.2.3, published
  2026-07-07, over three superseded candidates with every verdict kept and a
  digest of the bytes read. The release offers a Linux AppImage, a Windows
  setup.exe and a source tarball, each with a detached signature.
- ⚠ The run has nowhere durable to be recorded, which is what put `CORPUS-01`
  first: a measurement with no store is a file nobody can cite.
- ⚠ The listing answered with four releases, fewer than the project has, and
  whether the mirror paginates or answers a subset was not established. Nothing
  depends on it yet; the next entry to use the route measures it.
- Deployment: nothing deployed. Network: one release listing read, no artifact
  downloaded and nothing installed.

### 2026-09-05T10:30:00Z

- Closed `CI-05`. `check-project` and its PowerShell twin now refuse a `cargo
  test` invocation that selects by test **name**, because a filter matching none
  prints `running 0 tests` for every binary and exits 0. Record:
  [`TODO/ci.md`](TODO/ci.md).
- ⚠ The entry claimed all nine bare-filter acceptance commands had been
  rewritten and that was false: only the observer entries were. Five `Prove`
  commands were still of that form, in `FOUND-03` and all four `SCHEMA-*`
  entries. Each is corrected, and each corrected command was run before it was
  written down.
- ⛔ The door sweep found the second door and it is the one that matters more.
  An entry's `Prove` is run by a person; a workflow's `run:` is run by every
  push, and a bare filter there reports green over zero tests with nobody
  reading it. Both are covered, with separate extractors and one tokeniser.
- ⭐ Scoped to `Prove:` paragraphs rather than carrying an exclusion list. A
  `Closure evidence` paragraph records what was run on a past tree and rewriting
  one would falsify the record, and two entries have to quote the command that
  caused the defect. Every bare filter left in the tree is one of those two.
- Guard mutation: 21 cases, each verified to have changed the file, both halves
  compared on exit code **and** output. All 21 landed on the intended verdict
  and the twins agreed on all 21. The cases include the three that would make a
  careless rule fire on correct usage.
- The `forbidden-patterns.md` row now points at the check instead of asking a
  reader to remember the rule, which is what that page asks for.
- Deployment: nothing deployed.

### 2026-09-05T09:45:00Z

- Closed `OBS-09`, the raw evidence journal and bundle writer, which completes
  the observer layer. A run's transcript now becomes one
  `bit-ids/transcript/1` artifact per endpoint plus the manifest rows describing
  them, and a first vertical capture is possible from here. Record:
  [`TODO/observer.md`](TODO/observer.md).
- ⛔ The digest is of the file and the file is compared against the buffer. A
  writer that digests what it meant to write cannot detect a short write, and a
  truncated file digests to a value matching itself, so reading back closes only
  half of it. One comparison serves the write path and a later `verify`, so the
  guard that cannot be provoked at write time is proved by the caller that can.
- ⛔ The door sweep found a gate on how a path is spelled and none on where it
  resolves. A symlink in a reused bundle root satisfies every canonical-path
  rule and lands the artifact outside, with the manifest citing a path that
  reads as inside. The root is now resolved once and every artifact's directory
  must resolve under it; a path already held by a symlink, a file or a directory
  is refused rather than followed or overwritten.
- ⭐ That check has a second half that is easy to miss: without resolving the
  root, it refuses **every** write on a host whose root is itself reached
  through a symlink. The acceptance suite builds that case rather than arguing
  from platform folklore, and the mutation dropping it is refused by that test.
- ⛔ A transcript is never scrubbed and the type has no argument for it: the
  bytes a build put on the wire are the measurement. Scrubbing belongs to text a
  host produced, with every removal declared and counted, and the scrubber
  replaces what the caller names rather than guessing.
- An endpoint the transcript plan does not name is refused, not defaulted. A
  derived identifier and an assumed kind produce a manifest that validates and
  lies, because `E-MAN-52` and `E-MAN-53` only require the tool and the phase to
  name something the run declares.
- Guard mutation: 35 plants, 34 refused. ⚠ The first round refused 26, and all
  six misses were real gaps in the tests rather than equivalent mutants: the
  transcript schema, the producing tool and the phase were each asserted against
  a constant that moves with the code, or not at all. `TODO/observer.md` records
  it beside the `OBS-08` finding it repeats.
- Driven by a Python client that sent the bytes itself and then read the bundle
  off disk: 32 comparisons, 32 agree, including that each transcript holds
  exactly what that client sent and read back, request before answer. Four
  negative controls, and a rerun into the same root refused by name.
- `serde_json` added as a dev-dependency of `bit-ids-lab`. The lockfile diff is
  one line and no new package, because `bit-ids` already depends on it.
  [`docs/supply-chain.md`](docs/supply-chain.md) carries the layers.
- Deployment: nothing deployed. No capture was taken.

### 2026-09-05T08:30:00Z

- Closed `OBS-08`, the synthetic torrent. The generator was checkpointed in the
  tree last session; what landed now is the acceptance suite at
  `crates/bit-ids-lab/tests/synthetic_torrent.rs`, the guard-mutation pass and a
  driven pass. Record: [`TODO/observer.md`](TODO/observer.md).
- ⭐ The acceptance suite reads the info hash out of the **file**, by walking the
  raw metainfo and cutting out the byte range the `info` key maps to. Comparing
  a re-encode of the value the generator kept cannot see an info hash naming a
  dictionary the file does not contain, because both halves move together; the
  mutation pass plants exactly that and it is refused by this test alone.
- ⛔ Pinned the payload's byte stream, which nothing was checking. A generated
  torrent is citable only while its bytes are a function of its declared inputs,
  and a drift in the `SplitMix64` arithmetic invalidates every
  `capture.fixture` already recorded while staying reproducible, seed-dependent
  and prefix-stable, which is all a naive test asserts. Four plants that nothing
  else catches are now refused.
- ⚠ Pinned `PIECE_HASH_LEN`, `MIN_PIECE_LENGTH` and `MAX_PAYLOAD_BYTES` to their
  literals, and moved the test spec off the piece-length floor. A constant every
  test reads is a constant no test can check: narrowing the piece hash re-chunked
  the `pieces` string and the comparison against it in one step, and a spec built
  at the floor made the declared piece length and the constant indistinguishable.
- `piece()` now checks both halves of its offset. The unchecked addition beside a
  checked multiplication is a guard on one of two arithmetic steps; it is
  unreachable on a 64-bit target, which is why the mutation for it is the one
  this entry could not refute.
- Guard mutation: 33 plants, 31 refused. Each was a literal replacement required
  to match exactly once, verified against the file's SHA-256 either side, with
  the acceptance exit code read unpiped.
- Driven by `libtorrent` 2.1.1.0, the engine `ENGINE-01` targets, and `torf`
  4.3.1, neither of which shares this project's reading of BEP 3, over a file
  written by a new
  `cargo run -p bit-ids-lab --example synthetic-torrent`. 26 comparisons, 26
  agree, including four negative controls: a reader that agrees with everything
  has agreed with nothing.
- Deployment: nothing deployed. No capture was taken and none is possible until
  `OBS-09` writes the evidence a manifest cites.

### 2026-09-05T07:50:00Z

- ⚠ Corrected the session record's CI row. It reported the seventh run as
  confirmed when that run had been cancelled, and a cancelled run is no
  evidence. Record:
  [`docs/history/SESSION-2026-09-05-OBSERVERS.md`](docs/history/SESSION-2026-09-05-OBSERVERS.md).
- The class now has a row in
  [`docs/conventions/forbidden-patterns.md`](docs/conventions/forbidden-patterns.md).
- Deployment: nothing deployed.

### 2026-09-05T07:20:00Z

- Checkpointed `OBS-08`, which stays OPEN. The synthetic torrent generator and
  its unit tests are in `crates/bit-ids-lab/src/torrent.rs`; the acceptance
  suite, the guard-mutation pass and the driven pass are not done and the entry
  names all three. Record: [`TODO/observer.md`](TODO/observer.md).
- Added `sha1` 0.11.0, the first third-party crate since `SCHEMA-01`. The info
  hash is SHA-1 by BEP 3, and the lockfile diff is one package because `sha2`
  already brought the same RustCrypto tree.
  [`docs/supply-chain.md`](docs/supply-chain.md) carries the argument.
- ⚠ `check-no-secrets --public` refused RFC 3174's test vectors as long hex, and
  was right to: forty lowercase hex digits is what a token looks like. Narrowed
  rather than switched off, and proven with a credential beside an allowed
  vector on one line.
- The door sweep made six response encoders private: one internal caller each
  and no external one is API surface with no consumer.
- ⛔ The session's guard-mutation pass found a defect in its own probe for the
  third time: one script never received the checksum guard the first two were
  given, and reported the twins agreeing over source it had not mutated.
- The session record is saved at
  [`docs/history/SESSION-2026-09-05-OBSERVERS.md`](docs/history/SESSION-2026-09-05-OBSERVERS.md).
- Deployment: no data branch, release or capture service was created.

### 2026-09-05T06:30:00Z

- Closed `OBS-05`, the BEP 10 and early-message observer, which completes all
  four core observer surfaces. Record: [`TODO/observer.md`](TODO/observer.md).
- ⭐ What an observer offers is a condition of the measurement, and the type says
  so: the reserved block is derived from the same value the extended handshake
  is, so a run cannot claim an offer it did not make.
- ⛔ Three states rather than a flag beside an option. The fourth state a flag
  would allow means the observer invented a negotiation, and the guard-mutation
  pass found that deleting the guard against it changed no test result.
- Fourteen defects planted one at a time, all fourteen refused. The first round
  found four misses and all four were real, including a send-once flag that
  could be cleared with nothing noticing. That is the second entry running in
  which a send-once flag went unproven.
- Driven with the BEP 3 peer from `OBS-04`, extended to negotiate BEP 10. The
  peer answered with a deliberately unsorted extension map and an unregistered
  top-level key, and the observer recorded the map in the order sent.
- Deployment: no data branch, release or capture service was created.

### 2026-09-05T05:10:00Z

- Closed `OBS-04`, the peer-wire handshake observer, in both roles. Record:
  [`TODO/observer.md`](TODO/observer.md).
- ⭐ The lab dials now, and the dial went into the loopback guard rather than
  beside it. `OBS-01`'s door sweep had already put `TcpStream::connect` on the
  list its own test greps for, so there was nowhere else to put it.
- The responder signature grew a connection identity. One responder serves every
  connection an endpoint accepts, so without it a peer observer sends a second
  handshake down the first connection. The journal carries the connection too,
  which is what separates a transcript of two concurrent peer connections.
- Nineteen defects planted one at a time, eighteen refused. The one that is not
  is `rebuilds_from_raw` returning true unconditionally, and the entry says what
  would have to be true for it to fire: it is a codec-regression detector, and
  planting a lossy encoder in `bit-ids-wire` **is** refused.
- The door sweep found three on the dial path: a stopped lab wrote opening bytes
  nothing would answer, a role could be attached to the wrong side, and a
  connection past the observer's cap was left buffering rather than closed.
- ⚠ A test asserted a scheduling outcome, passed alone, and failed twice in
  three loaded runs. That is this session's second finding of that shape. The
  whole workspace suite now runs four times in succession with no failure.
- Driven with a BEP 3 peer written from the specification, in both roles at
  once. The two roles produced different reserved blocks and different peer IDs,
  which is the role dependence the entry exists for.
- Deployment: no data branch, release or capture service was created.

### 2026-09-05T03:40:00Z

- Closed `OBS-03`, the UDP tracker observer, which completes both tracker
  surfaces. Record: [`TODO/observer.md`](TODO/observer.md).
- ⭐ The BEP 15 exchange is stateful and that is the measurement. A client
  connects before it announces, so an announce carrying a connection id the
  tracker never issued means the build reused a stale one, invented one or
  skipped the connect. Each is answered with the protocol's error action and
  recorded with its reason.
- The connection ids are a contiguous deterministic range, which inverts this
  project's rule about identifiers coming from a random source. The entry
  carries the argument and the rejected alternative.
- Seventeen defects planted one at a time, all seventeen refused on the first
  round, which is the first round this session to miss nothing.
- The door sweep found one rule enforced in one of two places, twice. The
  connection id was read by the codec for an announce and by the observer's own
  byte slice for a scrape. And the datagram list was capped while the refusal
  list was not.
- ⚠ The claim audit found `Datagram::connection_id` reporting BEP 15's magic
  value as a connection id for a connect request. `TODO/observer.md` says how
  that was found.
- Driven with a BEP 15 client written from the specification in Python, which
  connected, announced, and was refused when it used an id the tracker never
  issued.
- ⛔ No observer has been driven by a stock `BitTorrent` client, and none can be
  on a session host. `TODO/PROGRESS.md` carries which guard refuses it and why.
  `OBS-07` owns the stock-client controls and `CI-03` owns the runner.
- Deployment: no data branch, release or capture service was created.

### 2026-09-05T02:30:00Z

- Closed `OBS-02`. The `bit-ids-probe` crate holds the observers, one module per
  surface, and the HTTP tracker is the first. It keeps the exact head bytes,
  decodes with `bit-ids-wire`, and answers a bencoded response. Record:
  [`TODO/observer.md`](TODO/observer.md).
- ⚠ What an observer answers is part of the experiment. A client that asked for
  a compact peer list and got a peer list reports an error and changes what it
  does next, and that change would be recorded as identity when it is the
  observer's doing, so `compact` and `no_peer_id` are read out of the announce
  and honoured.
- ⛔ Framing uses the codec's own `head_end`, added to `bit-ids-wire` rather than
  written a second time in the observer. `TODO/observer.md` says what a second
  framer costs.
- Seventeen defects planted one at a time, all seventeen refused. ⚠ The first
  round's script used `sed` with `|` as its delimiter over Rust closures and four
  plants silently matched nothing, which is the same probe defect as the round
  before. The replacement asserts the match count instead.
- ⭐ The first round also found that nothing exercised a head terminated with
  bare newlines, so the framer's answer could be shortened by a byte with every
  test still passing. A corpus only tests the defects it contains an example of.
- The door sweep found the record unbounded, so a build announcing in a loop
  would grow it until the host ran out of memory; the cap now counts what it
  stopped keeping. A second `Content-Length` header was taking the first value,
  and two lengths that disagree cannot both frame the rest of the connection, so
  the request is refused.
- Driven with `curl`. Two announces answered `200`, recorded with curl's query
  order, curl's header spelling, the percent-encoding case per value, and a peer
  ID of 20 bytes for one and 11 for the other. The 11-byte one was reported
  rather than refused: the width is the record layer's rule, and a build that
  sends the wrong one is the measurement.
- Deployment: no data branch, release or capture service was created.

### 2026-09-05T01:20:00Z

- Closed `OBS-01`. The `bit-ids-lab` crate is the loopback observation lab that
  `OBS-02` through `OBS-05` plug into: it binds the sockets, holds the run
  deadline, records every byte in order with its direction, and releases every
  port on shutdown or on drop. Record: [`TODO/observer.md`](TODO/observer.md).
- ⛔ Every socket in the crate is created by one function, which refuses an
  address outside loopback before the syscall and reads the address back off the
  socket afterwards. A bind request and a bound address are different facts, and
  only the second says where traffic can reach.
- The lab speaks no protocol. That is what lets one deadline, one loopback guard
  and one journal serve every surface instead of each observer growing its own.
- No new third-party crate. `std::net` with one thread per endpoint, over an
  async runtime, for the reason [`docs/supply-chain.md`](docs/supply-chain.md)
  requires in the entry: the lab serves a handful of local connections and the
  runtime would be a large dependency in the component that must be reviewable.
- ⛔ The entry's acceptance command ran nothing and exited 0. `cargo test`
  filters by test name and a filter matching none succeeds, so it printed
  `running 0 tests` for every binary in the workspace. The nine `cargo test`
  acceptance commands in `TODO/` were all of that form; all nine now name a
  target or a package, and `CI-05` is filed for the check that would stop it
  returning.
- Fifteen defects planted one at a time, all fifteen refused. The first round
  found three misses, and one was a defect in the shipped code rather than in
  the tests: the responder was offered its buffer once per read, so a client
  sending two units in one write and waiting for two answers would have waited
  forever.
- ⚠ A finding about the probe rather than the code. The first mutation script
  did not check that its edits applied and one pattern silently matched nothing,
  so a run over unmutated source read as a guard that failed to fire. Every
  plant now compares the file's checksum either side.
- The door sweep added outbound connections to the greps that hold the
  one-door rule, because `OBS-04` is authored to dial, and refused a lab capped
  at zero connections, which accepted every connection and closed it at once.
- Deployment: no data branch, release or capture service was created.

### 2026-09-05T00:10:00Z

- Split `OBS-01`, which was the one XL entry and carried an instruction to split
  itself if its acceptance could not stay atomic. It could not: the Prove named
  a known client fixture and a Linux-and-Windows comparison, and no client
  adapter exists while a Windows capture is not permitted at all. The supervisor
  stays in `OBS-01` at L, the synthetic torrent is `OBS-08`, the evidence
  journal is `OBS-09`, and the cross-platform comparison is `OBS-10` with its
  three blocking entries named. Record:
  [`TODO/observer.md`](TODO/observer.md).
- ⛔ `check-project` compared one row of `TODO/SUMMARY.md` against the index and
  eleven against nothing. Setting `Observer` to 9 over ten open observer entries
  passed the whole gate. Found by planting the count while checking that the
  split's arithmetic was held by something, not by reading the file.
- The category-to-identifier mapping now lives in `TODO/SUMMARY.md` as a
  `prefix` column rather than in the checks, so the two twins read one mapping
  instead of holding one each. Both directions are checked: a prefix with no row
  and a row naming nothing are both refusals.
- ⚠ The first version of that check used `^\| [A-Z]` in both halves and the two
  halves disagreed, because PowerShell's `-match` is case-insensitive and awk's
  bracket expression is not. The regex was character for character identical.
  A data row is now recognised by its shape. Both new classes have rows in
  [`docs/conventions/forbidden-patterns.md`](docs/conventions/forbidden-patterns.md).
- Eight defects were planted one at a time and both halves refused all eight and
  agreed on every one, with the exit codes read unpiped.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T19:20:00Z

- Closed `ACQ-04`, and with it the acquisition group's blocking work. A client
  is now installed only on a host two independent guards have refused to
  disqualify, and they run before the install. Record:
  [`TODO/acquisition.md`](TODO/acquisition.md).
- ⛔ The boundary could not live in the record. `E-MAN-30` refuses to record a
  capture on a host somebody keeps, and cannot stop one: by the time a manifest
  exists, an untrusted installer has already run somewhere.
- A host is claimed by writing a marker and refused when one is already there,
  so a survived host produces evidence of itself. A provisioner token was the
  rejected design: a runner misconfigured to persist its disk still carries one
  and still means it.
- The egress guard reads the kernel's routing table and probes nothing. Reaching
  out from a machine the guard exists to establish is contained would be the
  wrong order.
- Two defects found by driving it. The egress test used gawk-only functions, so
  on a POSIX awk it reported "could not establish" over a machine with a plain
  default route; it failed closed, but a guard that cannot run on a minimal
  image does not run where it matters. And the runner test claimed the real
  machine, so it passed once and failed every run after, which only running it
  twice finds.
- The manifest records what the guard read and when. A claim stamped after the
  run started is refused, because that is a report rather than a boundary.
- `check-runner` joins the gate on every run. The Windows gate reports it as a
  skip naming the reason: ⛔ the guards are Linux-only, so a Windows capture is
  not permitted rather than permitted with a warning.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T18:30:00Z

- Closed `ACQ-03`. Every route already had to report the version the record
  declares, so the labels always agreed; what was missing was whether that
  agreement was backed by anything. Record:
  [`TODO/acquisition.md`](TODO/acquisition.md).
- Two schema changes carry the rest: an executable digest per route rather than
  one per record, and a capture that says which route's install went on the
  wire. One digest collapsed the very difference this entry detects, and a
  record silent about which install was watched let a reader assume both were.
- ⛔ A single capture of two byte-different installs is unresolved, not
  equivalent. Nothing put the other bytes on the wire, so nothing can say they
  behave the same. Reaching a positive verdict over differing bytes takes a
  capture through each route, which is what the cross-record comparison is for.
- Two routes that installed identical bytes are one build, and observing one
  observed it. That is the only case where a single capture settles it.
- The refusals are two codes, not one: a divergence needs adjudicating and an
  unresolved record needs a second capture. Both are asked from the existing
  publication gate rather than beside it.
- A door sweep found the cross-record comparison guarding against two captures
  of one route but not against one record passed twice, which would have agreed
  on every field for the most trivial reason available.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T17:35:00Z

- Closed `ACQ-02`. Something can now decide which version to acquire, and the
  decision is a document that keeps every candidate it weighed, the bytes each
  source answered with, and the instant it was made. Record:
  [`TODO/acquisition.md`](TODO/acquisition.md).
- ⛔ Version strings are not comparable in general. Sorting tags as text puts
  `4.1.10` before `4.1.9`, and a project can publish a preview without setting
  the prerelease flag. A target declares how it spells versions, both stability
  signals are believed, and a candidate the scheme cannot order blocks the
  resolution rather than being skipped: skipping yields an older version chosen
  confidently, with nothing saying a newer one was seen and not understood.
- What settles an unorderable candidate is a second signal, not a looser rule.
  One published strictly before the winner cannot be the newest whatever its tag
  says. With no date, or a later one, it still blocks.
- The shell half fetches and nothing else. The digest in a resolution is then of
  what arrived rather than of what a parser reconstructed.
- Driven against four real projects through the route `docs/AGENTS.md` rule 8
  prescribes, since direct `api.github.com` answers 403 from this host. The
  first run failed closed over 51 of `transmission`'s own decade-old tags and
  was right to; that is what produced the publication-order rule.
- Two defects the tests found: `4.1` and `4.1.0` were ordered rather than
  compared equal, because a shorter component vector sorts first; and a schema
  check `from_json` could never reach was deleted rather than left as a guard
  nobody knows works.
- CI now finds every tracked shell script rather than listing two directories.
  A list is what let a new directory arrive unchecked on the lane meant to
  check it.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T16:40:00Z

- Closed `ACQ-01`. A route record now carries the typed kind, what resolved it,
  what delivered it, the original URL, the immutable identity of what it asked
  for, the artifact digest and signature, and the evidence of the installed
  version. Record: [`TODO/acquisition.md`](TODO/acquisition.md).
- ⛔ Two routes are two routes only if nothing they depend on is shared. The
  resolver and the delivery mechanism are separate values and a record whose
  routes share either is refused, because the two-route rule was otherwise
  satisfiable by asking one index twice under two names.
- The identity of what a route asked for is typed to its kind, so a release
  asset is a repository, a tag and a file name, and a source build is a full
  commit. An abbreviated commit is refused: that is the shape `FOUND-02`
  measured passing an action-pin rule written to refuse floating references.
- An installed version cites the process output the build printed. A version
  read out of a packet capture or a filename is not the build speaking, and the
  citation is checked to resolve and to be process output.
- The signature disposition is compared across the two documents. That
  comparison can fail, which is why it exists where `E-BND-10` was deleted:
  nothing else forces the run's record and the record's claim into step.
- A new code silently reused `E-BND-12`, which the capture-instant comparison
  already held. Two checks under one code is worse than an unclear message, and
  the manifest coverage test refused the change until the renumbered one had a
  planted defect.
- Driving the validators found what the suite could not: the signature
  diagnostic printed a spelling that appears nowhere in the document it was
  telling an operator to go and read.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T15:40:00Z

- Closed `FOUND-03`, the last foundation piece before acquisition. A new
  `bit-ids-wire` crate carries byte-exact codecs for the HTTP tracker, the UDP
  tracker and the peer wire, plus a synthetic fixture corpus with a committed
  digest index. Record: [`TODO/foundation.md`](TODO/foundation.md).
- The invariant is that decode then encode reproduces the input byte for byte.
  Every retention rule in `docs/architecture.md` section 5, meaning query and
  header order, duplicate fields, percent-encoding hex case, all eight reserved
  bytes and early message order, is destroyed by the convenient implementation, and a
  round trip catches all of them at once because a decoder that dropped a
  detail has nothing to write back.
- The codecs observe rather than impose. Unsorted bencode keys, `i-0e`, a bare
  newline terminator, an unassigned message id and a non-standard handshake
  protocol string are recorded, because each is a difference between builds and
  refusing one turns an observation into a parse failure. Nothing maps a
  peer-ID prefix or a `v` string to a client name, which would put a refused
  input inside the component every observer trusts.
- Nine lossy defects were planted in the codecs one at a time and every one was
  refused by the fixture corpus alone. Two were not caught on the first attempt
  and both are fixed here: the corpus never reached `bencode::encode`, because a
  message re-encodes from payload bytes held verbatim, and no fixture carried a
  bare newline for a terminator-repairing decoder to fail against.
- A door sweep found three holes and all three are fixed: `FixtureIndex`
  derived `Deserialize`, so `serde_json::from_str` skipped the corpus-digest
  check; `load_directory` filtered for `*.json` over a non-recursive listing, so
  a fixture in a subdirectory would have been silently skipped rather than
  refused; and the HTTP head cap only fired when there was no blank line at all,
  so a head that ended just past the cap parsed in full.
- The corpus was replayed over real loopback TCP one byte per write and decoded
  incrementally. Reading the same announce datagram with the wrong direction
  yields an unassigned action and refuses to be an announce, rather than a
  plausible wrong answer.
- No new third-party crate. The lockfile diff is the workspace member and
  nothing else.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T14:47:22Z

- Closed `SCHEMA-04`, and with it the whole schema group. A run records what it
  varied, and a classifier turns samples into per-byte lifetimes, so a peer ID
  comes out as the shape it actually has: a fixed prefix and a suffix the build
  regenerates. Record: [`TODO/schema.md`](TODO/schema.md).
- Nothing here is a confidence. A dimension the run never varied yields
  `unknown` rather than a guess, so a value that held still inside one process
  is not called persistent: only a restart separates a stored value from a
  regenerated one, and one sample yields `unknown` for everything.
- `bind` now refuses a field claiming variation from a run that varied nothing,
  and a field resting on more samples than the plan could produce. The manifest
  coverage test refused the change until both had a planted defect.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T14:40:23Z

- Closed `SCHEMA-03`. Corroboration now keeps what each connector saw, the
  artifact it read the value out of and what was applied before comparing,
  rather than a verdict and a list of names. Record:
  [`TODO/schema.md`](TODO/schema.md).
- A connector that cannot see a surface says so. Left out, a single observation
  looks like a pair that happened to agree, which is the easiest false
  agreement available; named, it is the reason that connector's silence proves
  nothing. An outcome of exact or normalized over fewer than two observers who
  could actually see the field is refused.
- Validity and publishability became separate gates, and both are needed. A
  disagreement has to be recordable or the project loses the evidence of one,
  so a conflicted record reads and validates; `publishable` is what refuses to
  ship it. A record that supersedes another now has to say why, out of the four
  causes a disagreement actually has.
- A normalization used to reach agreement must declare that order and unknown
  bytes survive it. That rule was written in the architecture and enforced by
  nothing.
- A door sweep found `is_publishable` and `publishable` answering the same
  question at different scopes with nothing holding them together. A test now
  drives a record through all four outcomes and asserts the two agree.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T14:29:09Z

- Closed `SCHEMA-02`. The run manifest records how a capture was produced: the
  host, the isolation it ran under, both clocks, every tool at the version that
  ran, where each artifact came from, the phases of the state machine the run
  walked, the content-addressed evidence and what was scrubbed from it. Record:
  [`TODO/schema.md`](TODO/schema.md).
- It is a second document rather than a larger section inside the profile, so
  a consumer of the catalogue does not have to carry a whole run. `bind`
  compares every value the two share, which is what stops a deliberate overlap
  from becoming drift. Pairing a manifest with the profile of a different run
  of the same build is refused.
- Absence stays as constrained as it is in a profile: a run that reached beyond
  loopback says why, a host that is not disposable is refused outright, a phase
  cannot be skipped, and an artifact marked redacted must say what was taken out
  of it so that "raw" cannot quietly mean "edited".
- One guard was written and then deleted. `E-BND-10` compared the installed
  version across the two documents and could not fail, because three existing
  invariants already implied it. It was found while trying to plant a defect
  for it, which is the only way that class of dead guard shows up.
- `ProfileError` is now `DocumentError` and carries the schema it expected. It
  covered both documents while naming only one, and its message told a reader
  with a manifest problem to go and look at the profile schema.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T14:11:29Z

- Closed `FOUND-02`. The action pin rule is now an allowlist of forms that are
  immutable by construction rather than a denylist of the floating ones
  somebody thought of, and the lockfile is checked for a crates.io source and a
  checksum on every package. Record: [`TODO/foundation.md`](TODO/foundation.md).
- Three shapes passed the old rule and are refused now: a branch named anything
  other than `main` or `master`, an abbreviated commit, and a bare commit with
  no version comment. The comment is load-bearing, because it is what
  `check-remote-items` resolves against the tag it claims to be.
- Added [`docs/supply-chain.md`](docs/supply-chain.md) with the three pinned
  layers and the procedure for updating one. No register of pins was added: the
  lockfile and the workflow already hold those commits, and a third copy would
  be the drift the rules exist to prevent.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T14:02:39Z

- Fixed `check-docs.ps1`, which resolved a link going up more than two
  directory levels to the wrong path and reported it broken. `[^/]+` matches
  `..` as readily as a directory name and PowerShell's `-replace` is global, so
  one call collapsed a real segment and the `../..` pair after it. It now
  replaces one leftmost match per pass, which is what the `sed` loop in the sh
  twin was already doing. Record:
  [`docs/conventions/forbidden-patterns.md`](docs/conventions/forbidden-patterns.md).
- The defect was latent from the bootstrap and surfaced by the first link in
  the tree that goes up four levels, added with the schema fixtures. That link
  is now the standing regression case: `check-twins` compares the two halves
  against it on every run.
- Deployment: no data branch, release or capture service was created.

### 2026-09-04T13:57:55Z

- Closed `SCHEMA-01`. The `bit-ids` crate carries the versioned profile record,
  the six field states, the derived record identifier, the canonical value
  forms and every publication invariant this schema owns, read and written
  through one validating path. Record:
  [`TODO/schema.md`](TODO/schema.md).
- The unproven-field rule is the point of it: a state that asserts anything
  about a build must cite recoverable evidence, and a claim that a build
  produced nothing must cite a positive control. Without the second half, an
  observer that was never listening and a build that never answered would be
  the same record.
- Added the first third-party crates, `serde`, `serde_json` and `sha2`, with
  the lockfile committed. A hand-written JSON reader was rejected: it would put
  a new silent-corruption surface in the one layer that must not corrupt.
- `Profile` no longer derives `Deserialize`. It did until the door sweep on
  this entry found that `serde_json::from_str::<Profile>` handed back an
  unvalidated record while the crate documentation said `from_json` was the
  only way in. The derive now sits on a private field mirror and every serde
  route validates.
- `check-no-secrets` grew two exclusions, for registry lockfile digests and for
  the algorithm-tagged digests and observed bytes a profile record is made of.
  Its long-hex rule had also been dropping whole lines to allow one item on
  them, so a credential beside a pinned action commit was never reported. Both
  halves now delete the allowed item and re-test what is left. Record:
  [`docs/conventions/forbidden-patterns.md`](docs/conventions/forbidden-patterns.md).
- Deployment: no data branch, release or capture service was created.

### 2026-09-04

- Added the initial repository foundation, research sweep, architecture,
  machine-readable target catalogue, Rust library skeleton, CI gate and full
  implementation backlog. Record: [`TODO/PROGRESS.md`](TODO/PROGRESS.md).
- Deployment: no data branch, release or capture service was created.
