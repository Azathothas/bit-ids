# Contributing a capture

What a contributor does, in order, and exactly what will not be accepted.
`DOC-02` owns this page.

⛔ **Read section 1 before anything else.** Most of what a well-meaning
contributor would produce is refused, and the reasons are not obvious from the
outside. Finding out after a capture run costs the run.

⭐ **Sections 6 and 7 are a walkthrough that runs.**
[`../scripts/common/check-handbook.sh`](../scripts/common/check-handbook.sh)
executes them in order, in one scratch directory, on every gate, and then hands
the result to the validator. A step this page does not carry is a step the
walkthrough does not have, which is what makes "no undocumented steps"
something that is measured rather than claimed.

---

## 1. What cannot be accepted

⛔ **A value read from source code, a client-ID table, release notes, a UI
label, a self-report or public swarm statistics.** Any of those may make a
target worth measuring. None may populate a field.
[`capture-methodology.md`](capture-methodology.md) is the authority on what
counts.

⛔ **A capture on a host you keep.** The boundary runs before the install, not
in the record: by the time a record exists an untrusted installer has already
run. [`capture-host.md`](capture-host.md) carries both guards and the runner
contracts.

⛔ **A field with no recoverable bytes.** A parsed value whose evidence cannot
be reread is not a measurement, and the validator refuses it as an unproven
field.

⛔ **A single observer.** Every profile needs the Rust active observer plus at
least one independent connector, and a field only one of them could see is
recorded as uncorroborated rather than published.

⛔ **A client binary, an installer, or any redistributable artifact in a
submission.** Keep the URL, the digest, the signature status and the package
metadata. `check-licences` refuses an installer-shaped file in the tree, and it
reads untracked files as well as tracked ones.

⛔ **An edited transcript.** The bytes a build put on the wire are the
measurement. Scrubbing belongs to text a host produced, and every removal is
declared with its count so that `raw` cannot quietly mean `edited`.

⚠ **A correction that edits what it corrects.** Published paths and bytes never
change. Section 8 is the flow.

---

## 2. Preparing a host

⛔ **Two guards, and both run before any client is installed.** They fail for
different reasons and neither implies the other: a fresh host with an open route
leaks the capture onto the public network, and a firewalled host that already
ran a capture contaminates this one with the last one's state.

```text
bit-check assert-disposable --claim
bit-check assert-disposable --egress
```

⚠ That block is shown rather than run. A session host has a public route, so
`--egress` refuses it, which is the guard working; running it here would be the
capture the boundary exists to refuse. `check-runner.sh` is what mutation-proves
both guards, and it is in the gate.

⛔ **A refusal is not something to work around.** If either guard refuses, the
host is not a capture host. There is no flag that makes it one.

⭐ **There is a workflow that provides such a host, and it is the easier route.**
`.github/workflows/capture.yml` claims a fresh hosted runner and builds. On
Linux it then runs the guard above and the capture in a network namespace with
loopback alone; on Windows it deletes its default routes, runs the guard,
captures, and restores the route only to upload.
[`capture-host.md`](capture-host.md) carries the step order and why the order is
forced. ⚠ It is dispatched by hand and it captures a fixture today: no client is
installed, and the attestation it uploads says so.

---

## 3. Acquiring the build

Two routes, and they must be independent in **both** halves: what resolved the
version and what delivered the bytes. Two package aliases pointing at one index
are one route, and the validator refuses a record whose routes share either.

Each route records the immutable identity of what it asked for, typed to the
kind of route it was. [`architecture.md`](architecture.md) section 7 lists the
three shapes and what each one has to carry.

⛔ **The version is read from the installed build**, by running it, and the
record cites the process output. A version taken from a filename, from package
metadata or from a packet capture is not the build speaking.

⛔ **And check that your route actually installed something.** A package manager
asked for something the host already has prints `already the newest version` and
exits 0, so a route can run cleanly and acquire nothing. Two such routes are
independent in both halves, agree on the version because there is only one
binary, and produce the strongest agreement this project can record. ⚠ Measured
here: `aria2` ships on the `ubuntu-24.04` image, and the capture workflow wrote
`route=package` over an install that installed nothing, twice.

⭐ **The install step measures that for you.** It asks the build for its version
before the route runs as well as after, and the record carries
`preexisting_version` and `acquired` beside `reported_version`. A route whose
record says `acquired=no` reported a build it did not put there.

---

## 4. Running the lab

The observers are handed to a lab that owns every socket, binds only on
loopback, holds one deadline and writes an ordered record of every byte each
endpoint moved. A capture points the client at a torrent the lab generates, so
the fixture is a function of its declared spec and can be re-derived from the
record.

⚠ **An adjacent surface is off unless somebody wrote the line that turns it
on.** Local discovery, peer exchange, the DHT, web seeding and message stream
encryption each name a destination of their own, so each sits behind a value
that has to be constructed rather than a flag with a default.

---

## 5. Reviewing the evidence

Before submitting, read the run manifest against the profile. They overlap on
purpose and `bind` compares every value they share, so a manifest paired with
the profile of a different run of the same build is refused.

⚠ Check that every phase the run walked is declared, that every artifact is
declared with the tool and phase that produced it, and that anything marked
redacted has a declaration saying what was removed and how many.

---

## 6. Building a submission

⛔ **From here down, the commands are the walkthrough and they run in order in
one directory.** Nothing below assumes a step this page does not carry.

Build the tools a submission needs, **from inside the checkout**:

```sh
cd "$REPO"
cargo build -p bit-ids --locked \
  --example build-store --example validate-corpus --example check-store
```

⛔ **The `cd` is not decoration.** `rust-toolchain.toml` pins the compiler and
rustup finds it by walking up from the working directory, so the same `cargo
build` run from elsewhere silently uses whatever toolchain that machine
defaults to. ⭐ This page's walkthrough was written with `--manifest-path` and no
`cd`, and the harness that runs it failed on the toolchain the first time,
which is the whole reason the page is executed rather than read.

Make an empty submission directory:

```sh
mkdir -p "$SUBMISSION"
```

Write the two documents and their evidence into it. ⚠ `build-store` is the
example that does this for a fixture; a real capture's tooling writes the same
shapes at the same derived paths, and the paths are derived rather than chosen
so that a path and an identifier cannot disagree.

```sh
"$BIN/build-store" --version 1.2.3 "$SUBMISSION"
```

⛔ **A record's path is derived from its whole identity tuple.** Do not compose
one by hand. A path built from fewer components files two measurements at one
name and one of them silently wins.

---

## 7. Checking it before you submit

The store-level pass, which is what turns a citation into bytes:

```sh
"$BIN/validate-corpus" "$SUBMISSION"
```

⛔ **This is the check that a submission cannot pass by agreeing with itself.**
A profile and a manifest that agree about an artifact nobody wrote satisfy every
per-record rule; only a store can say whether the bytes are there.

Then the structural rules a published tree must satisfy on every platform in the
matrix, asked as "what would appending this to an empty tree do":

```sh
mkdir -p "$EMPTY"
"$BIN/check-store" "$EMPTY" "$SUBMISSION"
```

⚠ Both exit 0 on a submission that will be accepted, and exit 1 naming a code on
one that will not. Read the exit code from the process, not through a pipe.

---

## 8. Correcting a published record

⛔ **Append, never edit.** The record being corrected keeps its path and its
bytes. A correction is a new record carrying `supersedes` and an adjudication
saying when it was settled, which of the four causes it was, what was decided
and the evidence behind it.

[`architecture.md`](architecture.md) section 4 names the four causes and why
they are the ones a disagreement actually has. ⚠ An original record has nothing
to adjudicate and carrying an adjudication is refused.

⭐ A superseded record leaves every view and stays in the store, and the
published `corrections` list carries both the record that directly corrects it
and the one at the end of the chain, so a consumer holding an old identifier
finds what answers now in one lookup.

---

## 9. What this page cannot walk

⛔ **Sections 2 through 5 are documented and not executed**, because a session
host is refused by the egress guard and running a client on one would be the
capture the boundary exists to prevent. The walkthrough starts where a
contributor's own capture ends: at the documents and the evidence.

⚠ **The submission the walkthrough builds is synthetic**, which is what
`build-store` writes and what everything in this repository is today. It proves
the acceptance path, not a measurement.
