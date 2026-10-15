# CI entries

## CI-01: Complete cross-platform quality gate

Source: template gate method and operator best-in-class CI requirement
Priority: P0 | Effort: L | Status: DONE

Problem: The bootstrap CI verifies code and documents but not future schema,
fixtures, corpus determinism, or publication invariants.

Approach: Add focused Linux and Windows jobs, immutable action pins, explicit
permissions, concurrency, timeouts, caching by lockfile, and strict local gates.

Prove: every required check appears as a non-skipped CI result, and injected
failures in docs, schema, fixtures, and Rust each turn the workflow red.

⚠ **Most of the Problem was overtaken before this entry was started.** The
Linux lane already delegates to `check-gate.sh --strict`, which runs the corpus
and publishing provers, and the Rust suite already covers the schema and the
fixtures on both lanes. What was actually missing was the two halves of the
Prove, and neither was a matter of adding jobs.

### What the first half turned out to be: a flag nobody could use

⛔ **The Windows lane ran without `--strict`, and it had to.** Six of that
runner's rows are checks Windows genuinely cannot run, so the flag would have
refused every correct tree. ⚠ The consequence is the one that matters: a check
that *stopped* running there was counted as a skip beside those six, and the
lane stayed green. Measured on 2026-09-06 by rewriting `check-project.ps1` to
`exit 2`; that lane's own invocation exited 0.

⭐ The runner now counts a **declared** unavailability apart from an **observed**
skip. A declared row is written into the runner with its reason and the entry
that owns it and prints as `n/a`; an observed one is a check that answered 2 or
whose file has gone. `--strict` refuses only the second, so both lanes ask
strictly and the six documented gaps stay documented.

### What the second half turned out to be: a harness that cannot drift

`scripts/ci/check-workflow.sh` copies the working tree into a scratch
repository, plants a defect of each class, and runs the offending step against
it. ⛔ **Every command it runs is read out of the workflow by job and step
name.** A harness holding its own copy of a build command proves that command
refuses a defect and says nothing about the one CI runs, and a step this file
names and the workflow no longer has is reported rather than passed over.

⛔ **It is kept out of `check-gate.sh` and that is a shared contract.** Two of
its cases run the workflow's *Repository gate* step, so a runner inside the gate
that also runs the gate would re-enter itself. The workflow calls it as a step
of its own instead.

⚠ **The gate cases needed a control the exit code could not give.** The
workflow runs the gate with `--strict`, and a developer host with no
authenticated `gh` has an observed skip of its own, so the clean tree exits 1
here and 0 on the lane. The control reads the runner's failure count and records
which host it is on; the plants still read the exit code unpiped.

### ⛔ The door sweep found the same assumption behind two doors

`store_build` composed an example's path as `root/target/debug/examples`, and
cargo obeys `CARGO_TARGET_DIR`. With that variable set the build succeeded, the
binary landed elsewhere, and the check for it fired: exit 2, which the gate
reads as a **skip**. ⚠ All five corpus and publishing provers therefore proved
nothing at all on such a host and reported it nowhere. Fixing that one exposed
the second door: `publish-data.sh` composes the same path itself, so the
publisher exited 2 saying its own append checker was missing. Both resolve the
variable now. Found by driving this entry, whose harness sets it.

### The publisher's workflow, which `PUB-02` left as two residuals

`.github/workflows/publish-data.yml` carries the job-scoped `contents: write`
and the concurrency group. ⛔ **It has no automatic trigger and its dry run is
the default**, because nothing may be published until a measured record exists
and everything in the tree is synthetic; a push trigger here would be one merge
away from publishing that. ⚠ Its group does not cancel in flight, unlike the
gate's: cancelling a gate run costs a rerun, and cancelling a publisher between
its append comparison and its read-back leaves a branch nobody has verified.

⚠ **The token reaches git as a header rather than inside a remote URL.** The
first version wrote it into the URL and `check-no-secrets` refused the file,
which is the guard working: a credential in a remote is visible to anything
listing processes on the runner.

⛔ **The workflow has never run and cannot succeed today.** Its first step
downloads the assembled bundle from a capture run, no such run exists, and that
is the honest state rather than a guard. What is proved is its shape.

### Acceptance, all run on 2026-09-06

- `sh scripts/ci/check-workflow.sh`
- `sh scripts/common/check-gate.sh`
- `pwsh -NoProfile -File scripts/common/check-gate.ps1`
- `cargo test --workspace --locked --all-targets`

### Closure evidence, 2026-09-06

| what | measured |
| --- | --- |
| `sh scripts/ci/check-workflow.sh` | 34 cases, 34 passed, 0 failed |
| `sh scripts/common/check-gate.sh` | 17 checks, 16 passed, 0 failed, 1 skipped, 0 unavailable |
| `pwsh -File scripts/common/check-gate.ps1` | 17 checks, 10 passed, 0 failed, 1 skipped, 6 unavailable |
| `cargo test --workspace --locked --all-targets` | 36 binaries, 337 passed, 0 failed |
| CI run 37, Windows lane | 17 checks, 10 passed, 0 failed, **0 skipped**, 7 unavailable, under `-Strict` |
| CI run 37, Linux lane | green through *Workflow acceptance*, which reported 27 cases passing at that commit |
| driven pass | eight defect classes planted one at a time, each refused by the step that owns it, with the clean tree accepted either side |

⭐ **The Windows row is the Prove's first half, measured rather than argued.**
Zero skipped means every check that lane can run, ran. The seven `n/a` rows are
the six platform gaps and `-Fast`, each named with the entry that owns it.

### Guard mutation

Eight probes over the runner split, run on a scratch worktree: a check rewritten
to `exit 2`, one rewritten to `exit 1`, and one deleted outright, against each
half, plus the clean control on both. All eight landed on the intended verdict.
⛔ The blind spot reproduced exactly: without the flag, the `exit 2` plant left
the runner at 0.

The harness carries nineteen more, being its four inherited probe guards, five
mutation probes over its own static readers, and the ten plant-and-control pairs
in its case list. ⚠ Its static readers are checked against a workflow with each
property **removed**, because a reader that answers present over a file that
lacks it would pass on any workflow at all.

### ⭐ The lane was thirty minutes and one step was twenty-six of them

**Measured on run 83, 2026-09-09, before anything was changed**, because "CI is
slow" is not a place to start optimising:

| step | of a 30.5-minute Linux lane |
| --- | ---: |
| *Workflow acceptance* | **25.9 min** |
| *Repository gate* | 3.5 min |
| everything else together | under 1 min |
| the whole Windows lane | 2.4 min |

⛔ **And the gate is not one gate.** `check-workflow` plants against it, so
several of its cases run the whole thing: **nine gate runs** inside that one
step. A second saved in the gate is nine seconds saved on the lane, which is why
the gate was measured next rather than the harness.

⚠ **The gate was three checks.** On this host its 29 checks took **198 seconds**
and 183 of them were `check-twins` at 90.7s, `check-capture-client` at 47.9s and
`check-capture` at 44.6s; everything else together was under fifteen. Serialising
twenty-six sub-second checks behind those three was the whole cost.

⭐ **Three changes, each measured, none of them weakening a verdict.**

| change | measured |
| --- | --- |
| the gate's checks run concurrently, verdicts read in list order | 198s to 100s |
| `check-twins` runs its twelve pairs concurrently, and each pair's two halves at once | 90.7s to **69s**, taking the gate to 73s |
| the two socket harnesses run after that batch rather than inside it | back up to **118-125s**, and correct under load |
| *Workflow acceptance* becomes a job of its own, beside the two lanes | it no longer serialises behind a gate it repeats |

⛔ **THE THIRD ROW IS A COST PAID ON PURPOSE AND IT IS MOST OF THE SAVING.** The
73-second gate was **wrong**: `check-capture` drives real sockets against a
three-second deadline, and under a batch of twenty-seven other checks it reported
*a run that recorded no bytes is refused (exit 1, but did not say 'recorded no
bytes at all')* - a refusal that arrived for a reason the case had not planted.
⚠ The same harness passed alone on the same tree minutes later, which is what
identifies it as load and not a defect. A faster gate that is red for no defect
is worse than a slow one.

⚠ **Raising the deadline was tried first, and measured, and rejected.** It looked
free - those harnesses wait on the observer's own line rather than on a delay -
and it is not, because several of their cases are ones the deadline itself has to
end: `check-capture` goes from **45 seconds at `3` to 79 at `6`**, and
`check-capture-client` from **48 at `5` to 168 at `20`**. That is roughly eight
to eleven seconds of gate per second of deadline, paid nine times over by
`check-workflow`, to buy back forty-seven.

⭐ **So the shipped gate is 118-125 seconds rather than 73**, measured twice at
each stage, and the reason it is not 73 is written where the schedule is.

### ⭐ What it did to the lane, measured on run 85

| | run 83, before | run 85, after |
| --- | ---: | ---: |
| **wall clock** | **30.5 min** | **21.3 min** |
| Linux gate | 30.5 min | **3.7 min** |
| Workflow acceptance | (inside the Linux gate, 25.9 min) | 21.2 min |
| Windows gate | 2.4 min | 2.6 min |

⭐ **The number a contributor feels is the first row of the lane, and it went from
thirty minutes to under four.** Compile, tests, lints and the gate now answer in
3.7 minutes; the acceptance harness answers separately and no longer holds them.

⛔ **And the wall is now that harness alone**, which is where the next work is
and why the residual below names sharding rather than anything inside the gate.
⚠ Both fast jobs were green while it was still running, which is the split doing
what it was written for rather than a claim about it.

⛔ **Every exit code is still read from the process that produced it.** `wait
"$pid"` returns that child's status and nothing else's, which is the same
guarantee the serial form had; there is no pipeline anywhere in either change,
because a pipeline's status is this repository's oldest refusal.

⛔ **And the rows are still in list order.** Each check and each pair writes to a
file of its own and its row is assembled at its own index, so a report cannot
come out in the order things happened to finish - which would make two runs over
one tree produce two different reports.

⭐ **The concurrency is safe because every check is hermetic**, which each one's
own header already claimed: each makes its own scratch directory, binds only
loopback, and reads the tree without writing to it. ⚠ `tree-unchanged` is the row
that keeps that claim honest, and it stayed green through the change.

⭐ **One thing had to be added rather than only reordered.** Most of the checks
call `cargo build --example` and cargo locks the target directory, so started at
once they queued behind each other and the concurrency bought nothing. The
examples are built once, before the queue - the same work, done once instead of
nine times. ⚠ Its failure is deliberately not a gate row: a build that fails
there fails again inside whichever harness needed it, where it is reported with
that check's own name.

⚠ **What was NOT changed is what `check-workflow` runs.** It executes the
workflow's own step commands, read out of `ci.yml` by job and step name, so
passing `--fast` to make its nine inner gate runs cheaper would mean changing
what the lane itself runs. The saving comes from the gate being faster, not from
it doing less.

### ⭐ A rule this repository said it had and did not, closed 2026-09-10

⛔ **`check-project` pinned the target set with a HARDCODED LIST of seventeen
ids**, asking that each appear in `catalogue/clients.toml` and in
`docs/client-matrix.md` - while that document claimed the set was "pinned by
`check-project` against the catalogue in both directions". ⚠ A target added to
the catalogue and forgotten in the matrix was caught by nothing, in either half,
and so was a matrix row naming a target the catalogue had dropped. The claim had
already been withdrawn in place on 2026-09-09; nothing had made it true.

⭐ **Both halves derive the set from the catalogue now and compare it against the
ids in the matrix's own rows, refusing a difference either way.** ⛔ And refusing
a set too small to be real, which is the guard `ACQ-01`'s catalogue scan already
carries: two empty sets agree perfectly, so a parser that stopped matching would
report a pinned matrix over nothing at all.

⚠ **The list is deleted rather than extended.** A list is a value in two places,
and this file already records three counts that went stale in prose for exactly
that reason; extending it only resets the clock.

Guard mutation, three plants, both halves run on each and their output compared:
a target added to the catalogue alone, a matrix row the catalogue does not carry,
and a matrix id renamed away from the catalogue - which fires BOTH directions at
once and is the case that says the two comparisons are separate. All three
refused, the twins agreeing character for character, and the clean tree accepted.

### ⭐ The ignore list and the secret-file rule, compared. 2026-09-15

`.gitignore` acts before a credential file exists; `check-no-secrets` rule 1
refuses one that is already tracked. ⛔ **Nothing compared their lists**, and two
names were in the second and neither line of the first: `*.jks` and `id_ecdsa`.
Both lines are added.

⭐ **`check-ignores` in [`../tools/check/`](../tools/check/) is the rule.** It
asks `git check-ignore --no-index` rather than re-reading the ignore format,
because a second parser of it would be a second answer to what git ignores. Each
specimen is first asserted to be a name rule 1 really refuses, using that rule's
own expressions, so a shape narrowed out of it is a finding here rather than a
case that quietly stops meaning anything.

Guard mutation, each plant verified to have changed the file:

| plant | verdict |
| --- | --- |
| `*.jks` dropped from `.gitignore` | refused, naming `release.jks` |
| `*.jks` and `id_ecdsa` dropped | refused, `"problems":2` |
| `jks` narrowed out of `check-no-secrets`' expression | refused, *stale specimen* |
| restored | accepted, 15 shapes |

⚠ An earlier attempt at the third plant reported NOT-PLANTED - the `sed` expected
`jks)$` where the source reads `jks)|id_rsa` - which is a third status and is
counted as neither.

⛔ **AND IT PASSED OVER NOTHING UNTIL A LATER MUTATION PASS.** With the specimen
list emptied it answered *every one of the 0 credential shapes is also ignored*
and exited 0 - the shape `check-licences` refuses in its register and `ACQ-01`'s
scan refuses in its vocabulary, reproduced in a new rule by the session that
wrote both. ⭐ A floor of 12 refuses a list too small to be real, and it answers
**2** rather than 1: an empty list means nothing was verified, not that a rule was
broken. ⚠ It is mutation-proved and is not a `check-bitcheck` case, for the same
reason the specimen self-check is not - the plant changes Go source and the
harness builds the binary once. ⛔ `check-bitcheck` carries the first two and not the third:
that one changes Go source and the harness builds the binary once before its
first case. It is also the first rule there that never had a shell half, so its
cases name `-` for the predecessor.

### Residuals

- ⚠ `check-ignores` asks about names at the repository ROOT. `.gitignore`
  patterns with no slash match at every level, so a root answer covers the tree;
  a nested `.gitignore` that re-permitted one of these names deeper down is
  outside what it asks.
- ⚠ **The other direction is unchecked**: `.gitignore` deliberately re-permits
  `.env.example` and `.dev.vars.example`, and nothing refuses a change that
  swallowed them. A project that lost those exceptions would find its committed
  template silently untrackable, which is a different defect from this one and
  needs its own specimens.
- ⚠ `check-workflow.sh` is not in `check-gate.sh` and cannot be, so a
  contributor's local gate does not run it. The workflow runs it on every push
  and it is this entry's acceptance; a gate runner that listed it would re-enter
  itself.
- ⚠ The concurrency is unbounded: every check is launched at once. On a
  four-processor host that is why the gate stops at 73 seconds rather than at the
  90 one check needs - the machine is saturated, not the ordering. A bound would
  be worth having on a host that cannot afford twenty-nine processes, and nothing
  here has met one.
- ⚠ `check-gate.ps1` is still serial. The Windows lane is 2.4 minutes end to end
  and its gate is 59 seconds, so there is nothing there to win; the two halves
  now differ in how they schedule and not in what they answer, and
  `check-twins` does not pair the runners.
- ⚠ **`check-workflow` is now the lane, and the next win is sharding it across
  runners rather than anything inside it.** Measured after the change: 764
  seconds locally, of which nine gate runs at roughly 70 seconds each are about
  ten minutes. ⛔ Those nine cannot be overlapped within one job - the gate
  already saturates four processors, which is why it stops at 73 seconds rather
  than the 69 its longest check needs - so the only way down is a shard per
  runner.
  ⛔ **It is deliberately not done here, and the reason is the failure mode.** A
  shard selector that silently claims no case leaves a rule nobody runs while
  every lane stays green, which is the exact defect this entry's own `--strict`
  work exists to prevent. It needs the partition to be provable by construction -
  `index mod N`, with the runner printing what it ran and skipped - and the
  workflow to be checked for running all `N`. That is a change to the harness
  every other guard is proved by, and it should be made as its own unit with its
  own plants rather than alongside a speedup.
- ⚠ On a host with no authenticated `gh` the harness's gate cases cannot use the
  exit code as their control, because the clean tree exits 1 there. They read
  the runner's failure count instead and say so in the row. `check-remote-items`
  is the only observed skip in either half.
- ⚠ `append_once` verifies that a file changed but has not itself been refuted,
  the way `replace_once` has. Every plant that uses it did land, which is
  evidence and not a proof.
- ⛔ **This entry said the `n/a` rows on the Windows lane "close only when
  `CI-03` lands", and that named the wrong event.** `CI-03` has landed and not
  one of them closed: they are `sh` harnesses, and what they need is a
  PowerShell half, which the capture runner matrix was never going to write.
  ⚠ The count was wrong too: it said six, and the runner had twelve rows
  carrying that label by the time anyone re-read it. Both are corrected in
  `check-gate.ps1`, where each row now names the event that would actually close
  it, and no count of them is written into prose anywhere. Found by a claim audit while closing `CI-03`, which
  is the pass that reads a sentence against the tree rather than against the
  sentence next to it.
- ⛔ **A row this gate was documented as having did not exist.**
  [`../docs/conventions/shell.md`](../docs/conventions/shell.md) section 5 says
  the repository compares every tracked file's working-tree line endings against
  what `.gitattributes` resolves for it, naming `git ls-files --eol` as the way.
  Nothing did, until 2026-09-08. ⚠ Found by breaking it rather than by reading
  it: a file-writing tool that emits LF left `check-project.ps1` at `w/lf` under
  `attr/text eol=crlf`, a full gate passed over that tree, and the only thing
  that reported it was an incidental warning from `git commit`. ⭐ The check is a
  row in both halves now, planted in both directions - LF where CRLF is required
  and CRLF where LF is - because they are different branches, and `git diff`
  prints nothing for either.

### The three deep reviews, 2026-09-15

⛔ **The door sweep asked which other door reaches a publication gate**, because
this session moved one. Four callers ask it and three hold a corpus - two in
`index.rs` and `corpus::publishable_view` - and all three were repointed;
`index.rs`'s third occurrence is inside `#[cfg(test)]` and correctly asks the
per-record question. ⚠ **The finding is `validate-profile`**, which reads ONE
document and therefore asks the one-document question correctly - and whose
`E-PUB-04` line now reads as a permanent refusal when a sibling capture is what
settles it. It says so on its own line when that code is among the blockers.
⭐ The sweep also enumerated every writer of a record `FieldState` outside the
schema: there is one, and it is the assembler.

⛔ **The guard mutation pass ran over four surfaces and two plants survived.**
Eighteen plants across `sampling::field_state`, the store-aware publication rule,
`Lab::dial_again` and the assembler's sampling fold, each verified to have
changed the file, with the clean tree run either side.

| survivor | what it says |
| --- | --- |
| the address comparison in `dial_again` | nothing varied the address, so a re-dial to somewhere else would have filed its segments under an endpoint naming a different peer. ⭐ A case varies it now and the plant is refused |
| a request with no `peer_id` read as an announce | no fixture here held one. ⭐ A scrape between two announces is a case now, and the plant is refused |

⚠ **Both survivors are the same shape and it is the one this file keeps
recording**: a corpus only tests the defects it contains an example of. Neither
was a gap in the rule; both were gaps in the fixtures, and the repair is a
fixture rather than a reading.

⛔ **And the mutation harness itself reported five refusals over nothing** on its
first run, because it named a `--test` target that does not exist: every plant
exited non-zero and read as REFUSED, with `0 case(s)` beside it and an empty
control. ⚠ A harness exit is *could not run*, never *refused*. It prints its
control first now. ⚠ A second instance of the same class: `cargo test` stops at
the first failing binary, so a refusal count taken without `--no-fail-fast` is a
count of the binaries that ran - one plant read as 1 case and is 6.

⛔ **The claim audit re-read every number written this session against the
tree**, and two did not hold. `docs/history/RESUME.md` carried **37 checks,
about 125 seconds** for a gate that is now **38 checks, 127 seconds**, timed
rather than assumed. And the `E-PUB-04` section above quoted a case count for
`check-assemble` that the same session's later work moved from 26 to 32; the
count is gone rather than corrected, which is what this file already says to do
about a number in prose. ⚠ What holds: 54 test binaries and **579 passed, 0
failed**; five twin pairs; 20 variability cases and 16 equivalence cases; and
`check-public-row`'s 5.

## CI-02: Stable-release staleness monitor

Source: operator automatic maintenance requirement
Priority: P1 | Effort: L | Status: DONE

Problem: New stable client versions must create bounded work without silently
overwriting previous records or repeatedly opening duplicates.

Approach: Schedule source-specific resolvers, compare against data indexes,
deduplicate by product/version/channel/platform, and open or update one tracked
capture request.

Prove: `sh scripts/ci/check-staleness.sh` and
`cargo test -p bit-ids --locked --test staleness` both pass, with a new stable
release opening one request, a preview and a release already measured opening
none, and a second run over the tracker the first one filled opening nothing.

### Decision: the identifier is derived, and that is the whole of "no duplicate"

⛔ **A request identifier is a digest of its key, never an allocated token.** Two
runs over the same facts derive the same identifier, so a tracker keyed on it
cannot hold two; a counter, a timestamp or a random token would each make the
second run's request a different request, which is the defect the Problem names.
`RequestKey` digests the four components the Approach lists, domain-separated and
length-prefixed the way `RecordKey` is, because joining them with a separator
lets two tuples encode to one string the moment a component carries the
separator.

⚠ **Architecture and package are deliberately not in the key.** They are outcomes
of the acquisition and are unknown when a request is opened, so a request that
carried them would multiply one release into a request per packaging, most of
which no route can satisfy. The question a request answers is whether the
selected version has a measurement on the platform at all; a package variant
lagging is coverage rather than staleness.

### Decision: the survey takes the whole resolution, not a version

⭐ **Nothing here judges stability, and that is why the preview case is not a
tautology.** A preview is refused by `ACQ-02`'s resolver, and `survey` takes the
`Resolution` rather than a `Version` so it cannot be handed a selection that came
from somewhere else. Every case in both the suite and the harness therefore
builds the release list a source would answer with and resolves it first.
A second stability rule here would be a second place for the answer to differ.

### Decision: the comparison is against the views, not the store

⛔ **An index is what a consumer reads instead of the records, and a monitor is a
consumer.** A record in the store and in no view is a measurement nobody can look
up, so counting it as coverage closes work a consumer cannot see was done.
⭐ The corollary is free: `CORPUS-04` drops a superseded record from every view,
so a retracted measurement re-opens its capture with nothing added here.

⚠ **The driving example re-derives the views rather than parsing the published
index document.** `index::build` is the one derivation, and a second reader of
that document would be a second reading of it. The `data` branch *is* the store,
so pointing `survey-staleness` at a checkout of it is the real path.

### ⛔ Four verdicts open nothing, and each is a comparison that did not hold

`regressed` is the one worth naming: a measurement newer than the selection means
the source dropped a release or the resolver read the wrong one, and a request
there asks a runner to capture a downgrade. `unresolved` is reported rather than
skipped, because a target whose resolution has been blocked for a month otherwise
looks like a target with no work. `ambiguous` is `ACQ-02`'s rule applied here:
`components` pads to the scheme's width, so `1.2.10` and `1.2.10.0` compare
equal, and calling that current leaves a differently spelled measurement standing
for the selection while calling it stale opens a request for a capture already
taken. `unorderable` blocks for the reason the resolver blocks.

⭐ `Staleness::opens_work` is the **one** answer to whether a verdict asks for a
capture. `survey` filters on it and `validate_requests` checks against it, so a
verdict wired into one and forgotten in the other is not expressible: it becomes
an `E-REQ-07` refusal naming the verdict.

### ⛔ What the guard mutation pass found, which nothing else could have

Eighteen plants over `staleness.rs`, one at a time into a scratch copy, each
compiled before being judged so that *could not run* stays separate from
*refused*. ⚠ The first pass reported fourteen refusals and four non-results, and
the four were the harness's own defects rather than surviving guards: three
literals no longer matched the formatted source and one plant did not compile.
They were repaired and re-run rather than counted either way, because a plant
that did not apply says nothing about the guard it was aimed at.
⛔ **Two of the refusals were by the harness alone, and that is the finding.**
Removing the platform from the request key, and replacing the length prefixes
with a separator join, both left every Rust case green:
`request_key_components_are_length_prefixed_rather_than_joined` collides only
under a `-` join and nothing varied the platform at all. ⚠ That is a test whose
**name** claimed more than it **checked**, which
[`../docs/methodology/reviews.md`](../docs/methodology/reviews.md) names as a
shape to test for specifically.

⭐ The repair is to pin the encoding rather than to add another collision pair.
`request_key_bytes_are_the_encoding_restated_independently` compares
`canonical_bytes` against the format restated from its own doc comment, so any
dropped component, any reordering and any join moves the bytes;
`a_request_id_moves_with_every_component_of_its_key` varies each of the four in
turn. Re-measured after the repair: all five key plants, including two written
only to check the repair, are refused by the unit tests alone.

Re-measured after the repair, over the whole set: eighteen plants, eighteen
refused, none surviving and none unable to run.

### What the door sweep found, which the mutation pass could not

Three, each about a door this entry opened rather than about the code it wrote.

⛔ **`VersionScheme::components` said "both callers use it" and named two.**
This is the third. A count in prose is a value in two places with nothing
comparing them, and it went stale in the one paragraph a reader checks before
writing a fourth implementation of the ordering. It says "every" now and names
all three questions.

⛔ **The channel's published spelling was written inside `staleness`.**
`docs/architecture.md` section 5 already made this rule for `Surface`: the
vocabulary has one home and it is the record's. A spelling of `ReleaseChannel`
living in a consumer is the shape where the next consumer writes a third. It is
`ReleaseChannel::as_str` beside the type now, with the serde pairing checked
where it always was.

⚠ **`scripts/README.md` said `store-lib.sh` is sourced by "all eight of the
harnesses above".** Measured: nine harnesses source it, `publish-data.sh` does
too and is not a harness, and `check-runner.sh` is listed above and does not
source it. The sentence was describing a set two files differ from before this
entry touched it. It carries the measured count and both exceptions now.

⚠ A fourth was looked for and not found: nothing else in the tree reaches
`survey`, `RequestId` or `validate_requests`, and `serde_json::to_string` over a
`RequestSet` does bypass validation, which is the same open in-memory
construction `Profile` has by design. For that to be a defect the write path
would have to stop validating, and `staleness_writes_nothing_it_would_refuse_to_read`
is what would fire.

### Acceptance, all run on 2026-09-08

- `sh scripts/ci/check-staleness.sh`
- `cargo test -p bit-ids --locked --test staleness`
- `sh scripts/common/check-gate.sh`
- `pwsh -NoProfile -File scripts/common/check-gate.ps1`

### Closure evidence, 2026-09-08

| what | measured |
| --- | --- |
| `sh scripts/ci/check-staleness.sh` | 19 cases, 19 passed, 0 failed |
| `cargo test -p bit-ids --locked --test staleness` | 29 passed, 0 failed |
| `cargo test --workspace --locked --all-targets` | 46 binaries, 503 passed, 0 failed |
| `sh scripts/common/check-gate.sh` | 21 checks, 20 passed, 0 failed, 1 skipped, 0 unavailable |
| `pwsh -File scripts/common/check-gate.ps1` | 21 checks, 11 passed, 0 failed, 1 skipped, 9 unavailable |
| guard mutation over `staleness.rs` | 18 plants, 18 refused, 0 survived, 0 could not run. ⚠ Two were refused by the harness alone before the tests were repaired |
| driven pass | a real store at 1.2.10, a real release list carrying 1.3.0, resolved and surveyed: one request opened; the tracker filled from it and re-surveyed twice, the same identifier `already_open` both times and the two documents identical apart from the clock |
| independent verification | `python3`'s SHA-256, over the encoding restated from the doc comment, re-derives `request:sha256:d6ce71ee…` byte for byte |

⭐ **The strongest control here is not this project's code.** The request
identifier is what the whole no-duplicate property rests on, and a survey
compared against its own encoder agrees with itself. Python's `hashlib` is a
SHA-256 this project did not write, and the harness carries that comparison as a
case rather than leaving it to a session that ran it once.

### Residuals

- ⚠ **Nothing schedules this yet.** The Approach says "schedule source-specific
  resolvers", and what exists is the comparison and its driving surface, not a
  cron trigger or an issue writer. A workflow that opened issues would need
  `issues: write` on this repository, and `docs/security/remote-ops.md` governs
  that; the tracker's shape is `--open`'s JSON array, which any writer can
  produce. It is a residual rather than a gap in the Prove: the Prove asks
  whether a request is created, updated and not duplicated, and all three are
  driven.
- ⚠ `check-staleness` needs `python3`, so it exits 2 on a host without one.
  Driven rather than asserted: run under a `PATH` carrying every other tool it
  needs and no python3, it prints `python3 not found` and exits 2, which the gate
  reads as a skip and never as a pass. The Linux lane runs the gate with
  `--strict` and `ubuntu-24.04` carries python3.
- ⚠ `survey-staleness` builds the views under the schemes its resolutions carry,
  so a store holding a target no resolution covers blocks under `E-VIW-01`. That
  is the right refusal and it means a real run monitors every target in the store
  or none. It is a residual because nothing today has more than one target in a
  store.
- ⚠ The survey's exit code says whether a request was opened and says nothing
  about a blocked line. The alternative, a third code, was rejected: a caller
  that only wants to know about new work would have to special-case it. Blocked
  lines are named in the summary and carried in the document.
- ⚠ An open request for a line a run was not asked about is left alone rather
  than retired. A survey that retired work outside its input would delete a
  capture request because a target was left out of one run's arguments.

## CI-03: Trusted capture runner matrix

Source: proprietary clients, multiple host families, and active observation
Priority: P1 | Effort: L | Status: DONE

Problem: Public pull-request jobs cannot safely run installers, privileged
network isolation, or publication credentials.

Approach: Separate untrusted validation from trusted capture, use fresh Linux
and Windows guests, environment protection, no fork secrets, per-client
timeouts, retained evidence, and manual approval where terms require it.

Prove: a forked change cannot reach capture credentials or runners; a trusted
fixture capture attests isolation and uploads its complete evidence bundle.

### ⛔ A hosted runner is a capture host, and recording otherwise was an error

Earlier sessions recorded nineteen entries as waiting on a host that
`assert-disposable.sh` would not refuse, and treated that as an external
blocker. It is not one.

- `--claim` writes a marker and refuses if one exists. A hosted runner is a fresh
  virtual machine per job, so no marker exists and it passes.
- `--egress` reads `/proc/net/route`, sends no packet, and refuses only because a
  default route exists. Deleting the default route satisfies it, and runners have
  passwordless sudo.

Measured on 2026-09-08 with the guard's own optional route-table argument: exit 1
over this host's real table, exit 0 over the same table with the default route
stripped, exit 0 over a loopback-only table. ⚠ **The guard was testable in one
command the whole time.** A blocker nobody tested is the shape this entry now
exists to stop being.

### The Windows guard pair, which is the part that is done

`assert-disposable.ps1` is the twin of the sh guard: `-Claim` writes an
exclusively created marker under `ProgramData`, `-Egress` reads `Get-NetRoute`,
and `-Fingerprint` digests the machine GUID with the install identity and the
computer name.

⛔ **It checks both address families.** A Windows host with IPv4 unplugged and
IPv6 up still reaches the internet, so a guard reading only `0.0.0.0/0` would
pass it.

⭐ **`-RouteTable` takes a file, which is what makes the logic provable on a host
with no `Get-NetRoute` at all.** `check-runner.ps1` drives every case against
fixtures, and `check-runner` is a real row on the PowerShell gate lane rather
than a declared gap. ⚠ The count is not written here: this paragraph said
"eleven" for as long as it took the claim audit to reach it, and the harness
prints its own total. ⚠ What that does **not** establish is that `Get-NetRoute`'s
real output matches the fixtures; only a Windows job running the guard with no
`-RouteTable` does, and that is part of the workflow below.

### Guard mutation over the Windows pair

Seven plants, one at a time. Four were refused on the first pass and three
survived; reading what each changed separated one real limit from two weak cases.

⛔ **Two of the harness's own cases passed for the wrong reason.** Three of the
guard's refusals share exit 2 and two share exit 1, so a case asserting the code
alone passes when a different refusal fired: blanking the mode check left both
mode cases green, because the branch they then fell into also answers 2. Every
case that shares a code now asserts what the guard said as well.

⚠ **One survivor is a guard nothing can refute and it is kept.** Rewriting the
marker's exclusive create to a plain create changes no behaviour a test can
observe, because the `Test-Path` check answers first on every constructible
input. It matters only in the window between that check and the open, which is
the race it exists for. Recorded rather than removed, the way `PUB-01` records
`entries.sort()`.

⛔ **Writing the harness found two defects in the guard on its first run.**
`Show-Usage` used `break` inside `ForEach-Object`, which does not stop the
pipeline: it breaks the enclosing loop, and with none it unwinds the script,
which exits 0. Every misuse of the guard therefore printed usage and reported
success. And `@(...) | Where-Object` yields a scalar for a single match under
`Set-StrictMode`, so `.Count` threw and every invocation exited 1 with a property
error.

### The capture workflow, which is the rest of it

[`../.github/workflows/capture.yml`](../.github/workflows/capture.yml) is
`workflow_dispatch`-only with one job per platform, and the step order is the
containment rather than a convention:

1. check out, then claim the host before anything else writes to it;
2. install the toolchain and build **while the network still exists**, because
   after the route goes nothing can be fetched and a capture that discovered a
   missing dependency under containment would have to restore egress to fix it;
3. delete the default route in both address families, saving them first;
4. run the egress guard, which must now pass;
5. capture;
6. restore the route only to upload, after the measurement is finished and on a
   host that is destroyed either way;
7. upload the evidence bundle and print the fingerprint the next job compares
   against.

⛔ **There is no `pull_request` trigger and that absence is the fork guard.** A
fork cannot cause a workflow to run in the base repository, so there is no
job-level condition for anyone to weaken. `check-workflow.sh` is where the
absence is asserted, because an absence is what a later edit restores unnoticed.

⚠ **A first draft of that workflow was written the previous session and
removed**, because it called `capture-run.sh` and `capture-run.ps1`, which did
not exist. A workflow naming a script nobody wrote is a file that reads as
finished and fails on its first dispatch. ⭐ **The scripts exist now, and so does
the rule that would have caught the draft**: `check-workflow.sh` pulls every
`scripts/…` path out of every workflow and refuses one that is not in the tree.
It is refuted against a copy naming a script that is not there, because a reader
answering "present" over any input would have passed the case on the draft too.

### ⛔ The step order is enforced, because every wrong order reads as plausible

Building after the route is deleted cannot work, and the only way to repair it
under containment is to restore the route on the host that exists to have none.
Capturing before the guard measures a host nobody established was contained.
Uploading before the restore is a step that cannot reach GitHub. ⚠ **None of
those is visible to any other check in this repository**, and each is a
two-line move in a diff. So `check-workflow.sh` reads the ordered step names of
each capture job and asserts every precedence the containment rests on, with a
step it names and the workflow no longer has reported rather than passed over. The reader is
refuted against a copy with two step names swapped.

⚠ **And a capture job whose `Capture` step ran `true` would satisfy all six.**
That is a separate case: the step's command must name the runner.

### The runner itself, and the three things it refuses that no workflow can

`capture-run.sh` and `capture-run.ps1` take an already built observer and a host
that a previous step claimed and contained.

⛔ **It builds nothing, and a missing observer is `could not run`.** That is what
makes the step order enforced rather than remembered: a workflow that moved the
build after the containment gets a refusal naming the binary, not a cargo
invocation reaching for a network that is gone.

⛔ **It re-reads the claim marker and the routing table itself.** Both were
already checked by earlier steps, and that is the point: a step order that
dropped the claim leaves the capture running on a host nothing claimed, and a
gate on one of two paths into the same action is the one-gated-door defect. ⭐ The
marker's path is asked for through `assert-disposable --marker`, which is a new
mode added for this and is the only derivation of that path; composing
`$STATE_DIR/host-claimed` a second time would go on reading the old place the
day the state directory moves.

⭐ **The driver and the verifier are both somebody else's code.** `curl` is a
complete HTTP client on both runner images and puts real bytes through the
tracker observer; `sha256sum -c` and `Get-FileHash` test the digests the
observer declared. The bundle writer already reads its files back against the
buffer it wrote, which is the writer checking itself.

⛔ **And the announce carries `key=<run-id>`, a token the driver knows it
sent.** Every other check here is satisfied by a bundle of empty artifacts that
verify against their own empty digests. This is the only one that says a
client's bytes reached the record, and it is the same argument `OBS-09` makes
about reading a bundle back with the client that wrote it.

⚠ **What it captures is a fixture and the attestation says so in fields rather
than in prose**: `kind=fixture`, `measured_build=none`, `stock_client=false`.
Nothing is installed and no stock build is observed; `CLIENT-01` is what points
a real one at the same lab. A bundle that outlived its context would otherwise
read as a measurement of a client.

### ⛔ What driving the PowerShell half found, in one command

`[switch]$Marker` collided with the existing `$markerPath` local, which was
called `$marker`. PowerShell variable names are case-insensitive, so the
`param()` switch and the local were **one variable**: the script assigned a
string to a `SwitchParameter` and every single invocation of the guard, in every
mode, failed to bind. ⚠ It is the exact hazard
[`../docs/conventions/shell.md`](../docs/conventions/shell.md) section 8 records
about `$args`, and it was found by running the guard once rather than by reading
it.

### ⚠ And what the harness found about itself on its first run

`check-capture.sh`'s stub observer streamed the real observer's output through
`awk`, with `fflush()` after every line. **`fflush` is about the wrong end of the
pipe**: mawk reads its INPUT in blocks, so a second into a five-second run the
log held nothing, the runner dialled a port whose process had already gone, and
six cases reported the driver's refusal instead of their own. Measured against
`cat` in the same pipeline, which had three lines at the same instant. The
splitter is a `read` loop now.

⛔ **And a `no-segments` mode written into the wrong stub fired the wrong
guard.** The runner checks the segment count before the evidence, so removing
the evidence rows produced `described no evidence` and the segment guard stayed
untested. Two guards over one input with the earlier one masking the later,
inside the harness written to stop exactly that. It is a second, mid-stream stub
now, which substitutes as well as drops so that a missing `segments:` line and
`segments: 0` are separate cases.

### ⛔ What CI found that no local run could: a default that changed under us

⛔ **Run 58's Linux lane went red on `check-capture` while the Windows lane went
green, and the local gate had been green on both halves minutes earlier.** The
cause is `$PSNativeCommandUseErrorActionPreference`, which is `$false` in
PowerShell 7.4 and **`$true` from 7.5**. Under `$ErrorActionPreference = 'Stop'`
it turns a native command's non-zero exit into a terminating error, so
`capture-run.ps1` calling the guard and reading `$LASTEXITCODE` got an uncaught
exception every time the guard **refused**. ⚠ The two cases that went red are
exactly the two that assert a refusal.

⚠ **It is this repository's oldest rule broken by an upgrade rather than by an
edit.** Nothing in the tree changed; the runner image did. The Windows lane
stayed green because its `pwsh` is older, so two lanes disagreed about one
script for a reason neither of them printed.

⭐ **Sixteen `.ps1` files relied on that default and one carried a comment
stating it as a guarantee** - `check-control-bytes.ps1` said the preference "is
false by default from pwsh 7.4", which is a fact about one version read as a
promise. Every one of the sixteen sets it explicitly now, and `check-project`
refuses a `.ps1` that sets `$ErrorActionPreference = 'Stop'` without it. The
rule takes no judgement, so there is nothing to argue about per file.

Four cases over the new rule, both halves run on each and their `--json`
compared: the clean tree; the preference removed from one file, refused by both
naming it; a `.ps1` that stops on nothing, accepted by both because it is not
asked for a preference it has no use for; and the clean tree again. ⭐ And the
end-to-end half, which is the one that matters: with the preference forced
`$true`, `check-capture` reproduces run 58's two failures exactly; with it
`$false`, the harness is green. The fix is what carries it, not the version.

### ⛔ And then run 59, which the fixed reporting caught in one line

⛔ **A refusal message that a harness matches on must not go through a display
layer.** `capture-run.ps1` reported through `Write-Error`, which inside a script
renders a source-context block and **wraps the message to the host's width**.
Measured on 2026-09-08: the refusal `the host is claimed by run [capture-0001],
not [capture-0002]` is one line at width 200 and **two** at width 80, so a
fixed-string match succeeds on a developer host and fails on a CI runner. The
case went red with the right exit code and the wrong message.

⚠ **Ten `.ps1` files already wrote through `[Console]::Error.WriteLine` and five
did not**, which is the BOM shape a third time: a convention held on most of the
paths into one mistake. All five write the bytes now, and `check-project`
refuses `Write-Error` in a tracked `.ps1`, in both halves.

⭐ **The rule's needle is an INVOCATION, not the word, and its first run proved
why: it fired on its own PowerShell twin**, because the failure message that
rule raises contains the name. A rule has to be describable in the file that
enforces it, so a match is the name in command position and a comment line is
skipped outright. ⚠ That is the needle-list lesson in a new shape - a needle
that matches its own description.

Five cases, both halves on each with `--json` compared: the clean tree; an
invocation at a statement start; an invocation after a pipe, which is the shape
`assert-disposable.ps1` used; the name in a comment and inside a string,
accepted by both; and the clean tree again. ⭐ And the end-to-end half at the
width that broke it: with `[Console]::Error.WriteLine` the fixed-string match is
1, with `Write-Error` it is 0.

⭐ **Run 59 is also what shows the reporting fix works.** Its log named the
failing case on its own line - `❌ ps a host claimed by another run is refused` -
where run 58's contained no failure at all.

### ⛔ And a red gate that did not say what failed

⚠ **Run 58's log reported `FAIL check-capture (exit 1)` and then eleven
PASSING rows.** `check-gate` printed the first twelve lines of the failed
check's log, and a mutation harness prints dozens of passing rows before the one
that failed, so the CI log did not contain the failure at all and it had to be
reproduced locally to be seen. The excerpt is the **tail** now, in both halves,
and `store_report` reprints the failing rows immediately above its summary so
the tail lands on them.

### Acceptance, all run on 2026-09-08

- `sh scripts/capture/check-capture.sh`
- `sh scripts/acquisition/check-runner.sh`
- `pwsh -NoProfile -File scripts/acquisition/check-runner.ps1`
- `sh scripts/ci/check-workflow.sh`
- `sh scripts/common/check-gate.sh`
- `pwsh -NoProfile -File scripts/common/check-gate.ps1`
- `cargo test --workspace --locked --all-targets`

### Closure evidence, 2026-09-08

| what | measured |
| --- | --- |
| `sh scripts/capture/check-capture.sh` | 41 cases, 41 passed, 0 failed |
| `sh scripts/acquisition/check-runner.sh` | 13 guard cases, 13 passed, 0 failed |
| `pwsh -File scripts/acquisition/check-runner.ps1` | 15 guard cases, 15 passed, 0 failed |
| `sh scripts/ci/check-workflow.sh` | 58 cases, 58 passed, 0 failed |
| `sh scripts/common/check-gate.sh` | 26 checks, 25 passed, 0 failed, 1 skipped, 0 unavailable |
| `pwsh -File scripts/common/check-gate.ps1` | 26 checks, 12 passed, 0 failed, 1 skipped, 13 unavailable |
| `cargo test --workspace --locked --all-targets` | 50 binaries, 542 passed, 0 failed |
| `cargo clippy --workspace --locked --all-targets -- -D warnings` | clean |
| driven pass, sh | the guards refused in order on a real host: no claim, a claim naming another run, a table with a default route, an unbuilt observer, an output directory that already held a run. Then the capture ran: `curl` announced, the transcript carried its request including `User-Agent: curl/8.5.0`, `sha256sum -c` verified both artifacts, and the attestation was read back |
| driven pass, PowerShell | the same set through `capture-run.ps1` under `pwsh` on Linux, against a `Get-NetRoute`-shaped fixture table. ⛔ It failed to bind on the first attempt and that is the `$Marker` collision above |
| independent readers | `curl` 8.5.0 put the bytes on the wire, `sha256sum` verified the sh half's evidence and `Get-FileHash` the PowerShell half's. None of the three is this project's code |
| CI run 58, first attempt | ⛔ **Linux lane RED**, Windows lane green, over the PowerShell 7.5 default described above. The local gate had been green on both halves minutes earlier, which is the whole point of recording it |
| CI run 58, what it cost | one cycle, because the gate's failure excerpt was the first twelve lines of a harness that prints its failures last. Both are fixed |
| CI run 59, first attempt | ⛔ **Linux lane RED again**, over `Write-Error` wrapping a refusal message at the runner's console width. ⭐ Its log named the failing case in one line, which is the reporting fix working |
| CI run 60 | ⭐ **both lanes green**, first attempt after both fixes, on `0a37c2a` |

### Guard mutation

Eleven plants into the two capture runners, one at a time, each verified to have
changed the file before it was judged: the claim-marker check, the claimed-run
comparison, the egress refusal, the existing-output refusal, the observer-present
refusal, the segment-count refusal, the zero-segment refusal, the digest
verification and the driver-token check in the `sh` half, and the claim and token
checks in the PowerShell one. ⛔ **Eleven refused, none survived, none failed to
apply, and in every case the harness row that went red was the one the plant was
aimed at** rather than some other row going red for its own reasons.

⚠ **The plant verifier reproduced this repository's own `grep -F` defect on its
first run.** Five of the eleven literals span two lines, `grep -o -F | wc -l`
counted each as 2, and all five were reported NOT-PLANTED over plants that would
have applied. `store-lib.sh`'s `replace_once` carries the same finding from
`CORPUS-01`, in a comment the harness author had read. Counted with something
that understands a multi-line literal, all eleven are unique.

One more over the gate's own wiring: `check-capture.sh` moved aside and the gate
run with `--strict`. It reports `SKIP check-capture (not present)` and exits 1,
so a harness that vanished cannot read as a pass. ⭐ **And a second, independent
door fired at the same time**: `check-docs` refused `scripts/README.md`'s link to
the missing file. Neither was written to catch the other, which is what makes the
pair worth recording.

Three more over the workflow, planted together rather than one at a time: an
added `pull_request` trigger, the build step moved after the route is cut, and
the capture step pointed at a script that is not in the tree. ⚠ **Together is
defensible here and would not be for the eleven above**: each is a different
property, read by a different reader, reported on its own line, so a refusal
stays attributable. The clean tree either side is the control.

### Residuals

- ⭐ **Closed by `CI-06`, which dispatched it.** The three things only a run
  establishes were all answered, and the run also found a defect in this file's
  own Windows restore step that no reading here had: the inverted egress guard's
  refusal was left in `$LASTEXITCODE` and GitHub reads that as the step's
  verdict. ⚠ Everything this entry says about the workflow's *shape* still
  stands; what it could not say is what a runner does with it.
- ⚠ **What it captures is a fixture.** No client is installed and no stock build
  is observed; the attestation says `kind=fixture`, `measured_build=none`,
  `stock_client=false`. `CLIENT-01` points a real build at the same lab.
- ⚠ **The restore step is the least proved thing in the file.** `ip route add`
  over a line `ip route show default` printed works on the hosts this was
  written against and is not exercised anywhere: `check-capture` runs against a
  route table it wrote itself and never touches the machine's. ⭐ Two things
  make that survivable. A failed add is **reported and does not end the step**,
  because under `set -e` the first refusal would skip the upload and throw away
  a finished capture over a route; and the inverted egress guard after it is the
  verdict, so an add is an attempt and the routing table is the fact. A guard
  that still passes there means the route never came back, which is a named
  failure rather than a network error naming nothing.
- ⚠ **Nothing pins the runner's PowerShell version, and nothing should.** The
  fix is that every `.ps1` states the behaviour it needs rather than inheriting
  it, which is version-independent. ⚠ What remains unproved is the rest of the
  7.5 surface: this found one default that changed by being bitten by it, and a
  second would be found the same way.
- ⚠ **The artifact pin is verified and the pairing is not.** The capture uploads
  with `actions/upload-artifact` v7.0.1 and the publisher downloads with
  `actions/download-artifact` v8.0.1. The pin was resolved and then re-read
  through a second route, `raw.githubusercontent.com`, which confirms the commit
  carries the *Upload a Build Artifact* action and accepts every input this
  workflow passes it. What no read establishes is that a v8 download reads a v7
  upload: neither workflow has ever run.
- ⚠ **Cancelling a dispatch between *Cut the route* and *Restore the route*
  spends the host.** The concurrency group does not cancel in flight, which stops
  a second dispatch doing it, and nothing stops a person pressing the button.
- ⚠ `capture-run` reads the observer's report from its stdout, so the observer's
  output contract is a shape held in two places. `check-capture`'s stub is what
  compares them, and it compares them against the runner rather than against a
  written specification.
- ⚠ The new BOM rule below has no fixture in the tree that exercises its
  ASCII-only branch, because every `.ps1` here carries markers. The mutation
  pass plants one; nothing holds it permanently, which is the same shape
  `check-twins.sh` warns about and the reason that pass found the drift in the
  first place.

### ⛔ What the door sweep found, one file away from this entry

Writing `capture-run.ps1` raised the question of whether a `.ps1` needs a UTF-8
BOM. [`../docs/conventions/shell.md`](../docs/conventions/shell.md) section 8
says one carrying non-ASCII does, because Windows PowerShell 5.1 decodes a
BOM-less file as the system ANSI code page. ⚠ **Eleven files had it and four did
not**, and every one of the four carries this project's markers on hundreds of
lines. Nothing was failing, because both CI lanes run `pwsh` 7 - and
`check-twins` still falls back to `powershell` when `pwsh` is absent, which is
where it would have bitten.

⭐ **A convention held in eleven places and broken in four is the one-gated-door
defect**, so the fix is both: the four files carry the BOM now, and
`check-project` refuses a `.ps1` that has non-ASCII and no BOM, in both halves.
The test is on the bytes rather than on a list, so an ASCII-only file is never
asked for a BOM it has no use for.

⛔ **The two halves disagreed on their first run and `check-twins` could not
have seen it.** A `.ps1` keeps CRLF, so the sh half's `[^ -~<tab>]` matched the
carriage return on every line and demanded a BOM for a file that is pure ASCII;
the PowerShell half excluded 9, 10 and 13 explicitly. ⚠ `check-twins` compares
the two on the tree it runs against, and **no `.ps1` in this tree is
ASCII-only**, so the branch that differed had nothing to exercise it. That blind
spot is written in `check-twins.sh` itself, and the fixture that found it is the
one the tree lacks.

Five cases, both halves run on each, exit codes and `--json` compared: the clean
tree; a BOM stripped from a file that carries markers, refused by both naming
that file; a new ASCII-only `.ps1` with no BOM, accepted by both; a new `.ps1`
with one marker and no BOM, refused by both naming it; and the clean tree again
after every restore. All five agree, character for character.

## CI-04: Build provenance and supply-chain hardening

Source: public autonomous publisher threat model
Priority: P1 | Effort: L | Status: OPEN

Problem: A valid-looking release can still be produced by an unexpected source
revision, dependency set, or workflow identity.

Approach: Add artifact attestations, dependency review, SBOMs, minimal token
permissions, environment rules, signed release metadata, and reproducible
builder facts.

Prove: release verification binds every asset to the expected repository,
workflow, commit, lockfile, and checksum manifest.

## CI-06: The first dispatched capture run

Source: `CI-03`'s residual, which is a workflow that has never been dispatched
Priority: P1 | Effort: L | Status: DONE

Problem: `.github/workflows/capture.yml` is checked in every way a reader can
check it and has never run. Three facts only a dispatch establishes: that
`Get-NetRoute`'s real output matches the fixtures `check-runner.ps1` proves the
Windows guard against, that deleting and restoring a default route works on a
hosted runner of each platform, and that the evidence bundle survives the
upload.

Approach: Dispatch both jobs, read the run back, download the artifact, verify
it against its own `SHA256SUMS` with a reader this project did not write, and
compare each job's reported fingerprint against the next run's. Record what the
route save and restore actually printed on each platform.

Prove: `sh scripts/capture/check-capture.sh` still passes, both dispatched jobs
end green, each uploaded bundle verifies under `sha256sum -c`, the two runs of
one platform report different fingerprints, and the attestation carried in each
artifact names `kind=fixture`.

### ⭐ Capture run 1, and the thing no reading had caught

The first dispatch was on `b992a35`, the commit `CI-03` closed at. ⛔ **The Linux
job was green end to end on the first attempt and the Windows job failed**, and
the failure is worth more than the success.

Three facts stood unmeasured and the dispatch answered all three:

| fact | what the run said |
| --- | --- |
| `Get-NetRoute`'s real output matches the fixtures `check-runner.ps1` proves the guard against | ⭐ it does. *Assert containment* ran the guard with no `-RouteTable` on a real `windows-2025` host and printed `no route off this host (read Get-NetRoute)`, which is the guard reading the live cmdlet and agreeing with the fixture corpus |
| a hosted runner's default route can be deleted and put back | ⭐ deleted on both platforms; restored on both. ⛔ The Windows *step* nevertheless failed, for a reason that is not the route |
| the evidence bundle survives the upload | ⭐ on Linux, yes. On Windows the upload never ran, because the step before it failed |

### ⛔ The defect: a restore that worked, and a step that failed on it

`Restore the route` ends by running the egress guard **inverted**. A host that
can reach the network again is one the guard REFUSES, so a refusal there is the
proof the route came back and a pass would mean it had not. The Windows job's
guard printed `assert-disposable: a public route exists; a capture host reaches
loopback only` - its refusal, exit 1, exactly as designed - and the step then
failed with `Process completed with exit code 1`.

⛔ **Nothing in the block was wrong. What failed was what the block left
behind.** GitHub's `pwsh` wrapper reads the residual `$LASTEXITCODE` as the
step's verdict, so a guard succeeding at its job set the code that failed the
step, the upload was skipped as a consequence, and a finished capture was thrown
away over a route that had already come back. ⚠ The `Write-Error` branch never
ran: the log carries no such line, which is how the two are told apart.

⭐ **The Linux twin never had this**, because `if sh …; then` consumes the
status and an `if` whose condition is false is a compound command with status 0.
⛔ **One of two paths into one mistake**, which is the shape
[`../docs/methodology/reviews.md`](../docs/methodology/reviews.md) calls the most
recurring hole there is - and it was invisible to every check in the repository
because the wrapper that reads the code is GitHub's, not this project's.

Both restore steps now end in an explicit `exit`, so the step's status is a
decision rather than a residue.

### ⛔ And the same guard's three codes were folded into two, on the green half

Reading the fix exposed a second defect in the half that had passed. The Linux
step asked `if …; then route-not-restored; fi`, which treats the guard's **2**
the same as its **1**. Those are `could not run` and `refuses`, and
[`../docs/history/RESUME.md`](../docs/history/RESUME.md) has carried that
distinction since a review pass counted one as the other. A guard that could not
run says nothing about the routing table, and uploading on it is uploading from a
host nothing established was reachable. Both halves now read all three codes and
name which one they got.

⚠ **This one was never going to fail a run.** `--egress` answers 2 only when it
cannot read a routing table at all, which has not happened on a hosted runner. It
is a latent conflation found by making the two halves symmetrical, and that is
the argument for making them symmetrical.

### ⛔ What the door sweep found: the same language behind a second door

`check-project` carries three rules over PowerShell - a UTF-8 BOM, the native
command preference beside `$ErrorActionPreference = 'Stop'`, and no `Write-Error`
where a machine reads the message. ⛔ **All three iterate `git ls-files '*.ps1'`
and none of them had ever reached a `pwsh` block inside a workflow**, which is
the same language, on the same runners, with the same two hazards.

⚠ **And `capture.yml` was carrying a live instance of each**: the `Write-Error`
above, and three blocks setting `Stop` with nothing saying what a native exit
code means. The rule that turned CI red twice last session had a whole second
door open the entire time.

⭐ The two rules now read the `pwsh` blocks of every workflow as well, and a
third rule exists that has no `.ps1` counterpart: **a block that reads
`$LASTEXITCODE` ends in an explicit `exit`.** That one is workflow-scoped on
purpose - a `.ps1`'s own exit code is its author's to choose, and it is only in a
workflow step that a harness reads whatever was left over.

⚠ The rules enumerate `git ls-files --others --exclude-standard` beside the
tracked set, as the `.ps1` rules already did. An uncommitted new workflow is part
of the tree the next push carries, and it is also how `check-workflow.sh` plants
against these three rules.

### ⭐ What the driven pass measured that the dispatch could not

⛔ **Nothing in this repository ran a capture step's body**, which is why the two
restore blocks were driven by hand here: each lifted verbatim and run against a
stub guard exiting 0, 1 and 2 in turn, with the exit code read from the process
that produced it. `check-workflow.sh` reads `capture.yml`'s step names, their
order and the *Capture* step's command; it executes steps out of `ci.yml` alone.
⭐ `CI-08`'s `check-step-bodies.sh` runs them as a gate row since 2026-09-09, so
what this paragraph describes is how it was done before there was a harness.

⛔ **The `pwsh` half has to be run the way GitHub runs it or the defect does not
reproduce.** The wrapper prepends `$ErrorActionPreference = 'stop'` and
**appends** `if ((Test-Path -LiteralPath variable:/LASTEXITCODE)) { exit
$LASTEXITCODE }`, then invokes `pwsh -command ". '<file>'"`. That append is the
entire mechanism. ⚠ A first version of the driver omitted it and reported the
broken block **passing**, which would have been a harness saying the defect was
not there over the block that had just failed on a runner.

⭐ With the wrapper faithful: the block as it stood on capture run 1 exits **1**
over a guard that refuses, which is run 1's failure reproduced on this machine;
the block as it stands now exits **0** over the same input, and **1** over a
guard that passes or one that could not run. The `sh` half answers identically
on all three.

⛔ **And a fact about the wrapper worth pinning: a dot-sourced script's exit code
collapses.** Measured across 0, 1, 2, 3 and 42 on pwsh 7.4.6: through
`-command ". '<file>'"` every non-zero value arrives as **1**, while `-File`
preserves it exactly. ⚠ So a `pwsh` step's exit code says *failed* and never says
which refusal fired - which is a second, independent argument for
`[Console]::Error.WriteLine`, because the message is the only channel that
survives. [`../docs/conventions/shell.md`](../docs/conventions/shell.md) section
8 carries the table.

### Guard mutation over the three new rules

Both halves of `check-project` were run on every case and their **exit codes and
their whole output** compared, because `check-twins` compares the pair on the
tree it runs against and a rule that differs only on a defect this tree does not
contain is invisible to it.

Fourteen cases: the tree as it stands; a block setting `Stop` with no native
preference; `Write-Error` at a statement start; `Write-Error` after a pipe, which
is the shape `assert-disposable.ps1` itself once used; the residual exit code
verbatim; every needle in comment position; both needles in a step whose shell is
not `pwsh`; a block that stops on nothing, asked for no preference it has no use
for; a single-line `run:` with no block scalar; a block that reads
`$LASTEXITCODE` and ends on an explicit `exit`; the three scope plants below; and
the tree again after every restore. ⭐ **All fourteen landed on the intended
verdict and the two halves agreed on all fourteen, character for character.**
Each refusal was also checked to be named by the rule under test rather than by
some other rule going red.

⭐ The five acceptances are the half that matters most here. A rule that refused
a needle in a comment, or one in a `bash` step, is a rule somebody switches off.

### ⛔ What the door sweep found in the new rules themselves

The first version matched `shell: pwsh` in `.github/workflows/*.yml` and nothing
else, which is the defect this entry is about, in the code written to catch it.
`shell: powershell` is Windows PowerShell 5.1 rather than a different language; a
composite action under `.github/actions/` carries its own steps and the same
permissions, which is the argument `check-project`'s **action-pin rule already
makes in a comment** two hundred lines above; and a workflow may be `.yaml`.
⚠ **None of the three exists in this tree**, which is precisely when a scope is
easiest to get wrong: every reading agrees with the correct rule on every file
here. The scope is the pin rule's now, and it is proved by planting each of the
three fixtures the tree lacks.

### ⛔ And what the per-plant comparison found that a clean tree could not

Widening the scope broke the PowerShell half outright, and **both halves went on
agreeing perfectly on the clean tree** while it was broken. `Resolve-Path
-Relative` prepends `./` to most paths and **not** to one that already begins
with a dot, so a fixed `Substring(2)` ate the `.g` of `.github`, every file then
failed its `Test-Path`, and that half examined nothing at all. A file set only
appears in the output when something in it fails, so the clean tree could not
show it. ⭐ **Seven of the fourteen cases went red at once and named it**, which
is the whole argument for comparing the twins per planted mutation rather than on
a clean tree - the argument `check-twins.sh` makes about itself and `CI-07`
carries. The root is stripped now rather than a prefix assumed.

⚠ **The extractor was checked against what it hunts for**, because a reader
answering "clean" over an empty set passes on any tree: it finds every `pwsh`
step in the tree, which is the same set `shell: pwsh` appears on.

### ⛔ What the claim audit found: this entry's own Prove, refuted on one platform

The Prove asks that *the two runs of one platform report different
fingerprints*, which is how a host that was not destroyed would be caught.
Measured across the two dispatches:

| platform | run 1 | run 2 | same? |
| --- | --- | --- | --- |
| Linux | `051ae9dd…` | `49f8714f…` | ⭐ different, as the Prove asks |
| Windows | `c5cde27f…` | `c5cde27f…` | ⛔ **identical** |

⛔ **And both Windows hosts were fresh.** Each run's *Claim the host* step
succeeded, and that step writes an exclusively created marker under
`ProgramData` and refuses if one is already there. Run 1 wrote one; run 2 found
none. So the comparison the Prove asks for reports "a survived host" over two
hosts that demonstrably were not.

⭐ **That is structural, and recording it as a defect to patch would be the
error.** A fingerprint has to differ between two hosts *and* survive a reboot
within one, so that a machine which rebooted rather than being destroyed still
reads as itself. Every input with the second property is a property of the
**image**, and hosted runners are clones of one image. ⚠ Adding a per-boot value
would buy the first requirement by destroying the second, which is the failure
the guard exists to catch.

⛔ **The claim marker is what actually answers the question, and always was.** It
detects a survived host by finding its own marker rather than by comparing a
value against a previous run's, which is the difference
[`../docs/capture-host.md`](../docs/capture-host.md) draws between detecting a
failure and trusting a claim. The workflow's Windows step summary said "A repeat
means the host was not destroyed" and now says what the value is instead.

⚠ **What is not established is WHICH of the guard's inputs is constant across
clones.** The digest is over the machine GUID, the install date, the build label,
the computer name, the OS version and the user name; at least one of the varying
candidates does not vary, and nothing here says which. Settling it needs a mode
that prints a digest per part and two more dispatches. ⚠ **The alternative was
considered and rejected**: it widens `CI-03`'s mutation-proven guard for an
answer that changes nothing, because the fix is the same whichever input it is.
It is a residual below.

### Acceptance, all run on 2026-09-08

- `sh scripts/capture/check-capture.sh`
- `sh scripts/ci/check-workflow.sh`
- `sh scripts/common/check-gate.sh`
- `pwsh -NoProfile -File scripts/common/check-gate.ps1`
- `cargo test --workspace --locked --all-targets`
- `cargo clippy --workspace --locked --all-targets -- -D warnings`

### Closure evidence, 2026-09-08

| what | measured |
| --- | --- |
| capture run 1, first dispatch ever | ⛔ Linux green, **Windows red on *Restore the route*** over a restore that had worked |
| capture run 2, after the fix | ⭐ **both jobs green**, every step of both, first attempt |
| `Get-NetRoute` against the fixtures | ⭐ the Windows guard ran with no `-RouteTable` on a real host and answered `no route off this host (read Get-NetRoute)`, in both runs |
| the default route, deleted and put back | ⭐ both platforms, both runs. The inverted guard is the verdict: it refused after the restore, which is a routing table with a public route on it |
| the evidence bundle through the upload | ⭐ every bundle survived: `sha256sum -c` exits 0 over each, naming the metainfo and the transcript |
| the attestation | ⭐ `kind=fixture`, `measured_build=none`, `stock_client=false` in every artifact |
| what put the bytes on the wire | `curl` 8.5.0 on the Linux runner and `curl` 8.16.0 with Schannel on the Windows one. The transcript carries the announce, its `User-Agent`, and `key=<run-id>`, the token the driver knows it sent |
| the fingerprint comparison | ⭐ different on Linux; ⛔ identical on Windows, over two hosts whose claims both succeeded. The paragraph above is the finding |
| `sh scripts/ci/check-workflow.sh` | 63 cases, 63 passed, 0 failed |
| `sh scripts/common/check-gate.sh` | 26 checks, 25 passed, 0 failed, 1 skipped, 0 unavailable |
| `pwsh -File scripts/common/check-gate.ps1` | 26 checks, 12 passed, 0 failed, 1 skipped, 13 unavailable |
| `cargo test --workspace --locked --all-targets` | 50 binaries, 542 passed, 0 failed |
| guard mutation, the three new rules | 14 cases, both halves on each, exit codes and whole output compared; all 14 on the intended verdict and the halves identical on all 14. ⛔ Seven of them went red first and caught a break in the PowerShell half that the clean tree agreed with perfectly |
| driven pass, the restore blocks | 13 cases. Run 1's failure reproduced on this machine, and the fixed block exits 0 over the same input |
| CI run 62 | ⭐ both lanes green on `1938672` |

⭐ **The strongest control here is not this project's code.** `sha256sum -c` and
`Get-FileHash` verified the evidence, `curl` put the bytes on the wire from both
runner images, and the run that decided whether the fix works is GitHub's, on
hosts this session cannot reach.

### Residuals

- ⚠ **Which of the Windows fingerprint's inputs is constant across clones is not
  established.** A `-FingerprintParts` mode printing a digest per part would
  settle it in two dispatches; it was rejected for now because it widens
  `CI-03`'s mutation-proven guard for an answer that changes no decision. The
  finding that matters - that the comparison is not a freshness signal there - is
  measured and recorded.
- ⭐ **Closed on 2026-09-09.** This residual said nothing in the tree runs a
  capture step's body, and named `CI-08` as carrying the harness that would
  close it. `scripts/ci/check-step-bodies.sh` is that harness: it runs both
  restore blocks under GitHub's own wrapper form against stubbed guards, and it
  refuses the Windows block in the shape that failed capture run 1. ⚠ What
  remains true is the sentence's premise about `check-workflow.sh`, which still
  reads `capture.yml` statically and executes steps out of `ci.yml` alone.
- ⚠ **The `sh` restore step's three-code reading is not driven on a runner.**
  `--egress` answers 2 only when it cannot read a routing table at all, which has
  not happened on a hosted runner; the branch is driven against a stub here and
  by nothing on the real path.
- ⚠ **The upload/download pairing is still unexercised.** The capture uploads
  with `actions/upload-artifact` v7 and the publisher downloads with
  `actions/download-artifact` v8. `CI-09` owns it; what these two runs establish
  is only that a v7 upload survives and can be fetched over the REST route.
- ⚠ **What was captured is a fixture and every attestation says so.** No client
  was installed, no stock build was observed, and nothing was published.
  `CLIENT-01` points a real build at the same lab.
- ⛔ **Reading the run back is what found the last one.** The five new cases each
  run the whole gate, and the Linux lane went from 13.4 minutes on run 61 to
  **24.4 of the 30 it then had** on run 62, 20.4 of them inside *Workflow
  acceptance*. It passed, with under six minutes to spare, and a lane that runs
  out of time reports as infrastructure rather than as a defect. The budget is 45
  now and the measurement is in the workflow beside it. ⚠ The Windows lane is
  unaffected at 2.6 minutes: it does not run this harness.

## CI-07: PowerShell halves for the declared gate rows

Source: every `n/a` row on the Windows lane, each naming a missing half
Priority: P1 | Effort: L | Status: OPEN

Problem: The Windows gate declares every mutation harness unavailable except
`check-runner`, whose PowerShell half exists. Each of the rest is an `sh`
harness with no PowerShell implementation, so the guards they prove are proved
on one platform and asserted on the other. ⛔ `--strict`
permits a declared row forever, which is correct and is also why the gap does
not shrink by itself.

⚠ **This Source said `thirteen` and the runner declares fourteen.** Found by a
claim audit on 2026-09-08, and it is the SECOND time a count of these rows has
gone stale in prose: `CI-01` records the first, where six had become twelve, and
its own fix says in as many words that no count of them is written into prose
anywhere. One had survived here. The number is gone rather than corrected,
because correcting it only resets the clock.

Approach: ⛔ **Rewritten 2026-10-14, because `CI-10` changed the route.** A
row closes by the event its own reason names, and the rows are kinds rather
than one population:

| kind | rows, as `check-gate.ps1` declares them | closes when |
| --- | --- | --- |
| a portable subject | `check-assemble`, `check-bitcheck` | its harness is ported into `bit-check` by `CI-10`'s method: written in Go, compared case for case against the `sh` half, then the `sh` half deleted |
| class B | `check-store` | its plants are reconsidered for Windows: a symbolic link and a named pipe cannot be made by an unprivileged Windows session, and a half that skipped them would report a smaller pass under the same name |
| class C, an `sh` subject | `check-access`, `check-bundle`, `check-publish`, `check-transmission` | the subject gains a Windows half; not this entry's to close |
| an `sh` subject not yet classed | `check-capture-client`, `check-release-route`, `check-source-route` | class C as well: each subject is an `sh` script; their reasons say so in the next change to them |
| class D, nothing to write | `check-capture`, `check-examples`, `check-gate-rows`, `check-handbook`, `check-public-row`, `check-step-bodies` | never: each reason says why there is no half |
| another entry's | `check-rootless` (`ACQ-06`), `check-shell` (`CI-08`) | that entry |

⚠ The table is a reading of `check-gate.ps1` on 2026-10-14, and that file is the
list: a row added there is in this entry without being added here.

⭐ **`check-corpus` is the first row this rewritten route closed, 2026-10-14.**
[`../tools/check/corpus.go`](../tools/check/corpus.go) ports it case for case:
the same 16 rows, the same labels, the same plants into a store `build-store`
wrote. `check-bitcheck --compare` ran both halves over the clean tree and over
two plants in the validator they drive. One stopped comparing digests and one
stopped sweeping undeclared files, and each failed exactly its own row, 15 of
16, in both halves. 268 of 268. Then the `sh` half was deleted, and both gate
runners call `bit-check check-corpus`, so the Windows lane runs it. ⚠
`check-gate-rows`' dropped-prover plant had removed `corpus/check-corpus` from
the `sh` list; it removes `corpus/check-indexes` now.

⭐ **And `check-indexes` the second, the same day.**
[`../tools/check/indexes.go`](../tools/check/indexes.go) ports its 17 rows,
sharing `plantCase` with `check-corpus`. Two plants went into the index builder,
and both halves agreed on each. One kept a superseded record in the views, which
failed the three correction rows, 14 of 17. The other renamed `E-VIW-02`, which
failed that case alone, 16 of 17. ⚠ The first port read the `latest` section
until the exact line `  ]`; the `sh` half's `sed` range ends at any line
starting `  ]`, so the port read on into the next section and found the
corrected record there. It reads as `sed` does now. The dropped-prover plant in
`check-gate-rows` removes `publishing/check-release` now.

⭐ **CI run on `b63228e` read both ports back on the Windows lane**: `✅ ok
check-corpus` and `✅ ok check-indexes`, 42 checks, 19 passed, 0 failed, 0
skipped, 23 declared.

⭐ **And `check-release` the third.**
[`../tools/check/release.go`](../tools/check/release.go) ports its 15 rows.
⛔ Its checksum row is the one that needed a decision. The `sh` half handed
`SHA256SUMS` to `sha256sum -c` because a reader this project did not write is
the point. A non-bash step on a Windows runner sees no `sha256sum`, so the port
asks `sha256sum` where it exists and `certutil -hashfile` for each row where it
does not. Hashing with this binary's own SHA-256 would be the writer checking
itself. A host with neither cannot run the check. The `certutil` reader was
driven here with a stub printing Windows' format, on a `PATH` with no
`sha256sum`: 15 of 15, and 14 of 15 with one wrong digest. Two plants went into
the assembler, and both halves agreed on each. One stated the manifest's digest
over the wrong bytes and failed the checksum row alone. One renamed `E-REL-02`
and failed that case alone. ⚠ Its `E-REL-02` plant makes a symbolic link, which
a Windows session needs privilege for; a hosted runner's user has it, and the
next CI run is what shows it. `check-gate-rows`' dropped-prover plant removes
`publishing/check-formats` now.

⭐ **CI on `5b7c586` read `check-release` back on the Windows lane:** `✅ ok
check-release`, 20 passed, 0 failed, 0 skipped. The symbolic-link plant and the
checksum reader both ran there.

⭐ **And `check-formats` the fourth.**
[`../tools/check/formats.go`](../tools/check/formats.go) ports its 28 rows.
Python's `sqlite3` is still the independent database reader, through the shell
half's own script, embedded byte for byte. Python is looked up as `python3` or
`python`, and must be Python 3, because a Windows runner may carry only the
second. Two plants went into the SQLite rendering, and both halves agreed on
each. One stated one record more than the database holds; one wrote no
`omission` rows. Each failed one row, 27 of 28. ⛔ Two Rust comments said this
check handed the CBOR to `cbor2`. Neither half ever did: `cbor2` read it once,
in `PUB-03`'s driven pass. Both comments say so now. `check-gate-rows`'
dropped-prover plant removes `publishing/check-publish` now.

⭐ **And `check-staleness` the fifth.**
[`../tools/check/staleness.go`](../tools/check/staleness.go) ports its 19 rows.
The request identifier is re-derived with Go's SHA-256 where the `sh` half used
Python's. Either is independent of the Rust encoder, which is the point. Two
plants went into the survey, and both halves agreed on each. Moving the
identifier's domain to `capture-request/2` failed the re-derivation row alone,
18 of 19. Never treating a held request as already open failed the three repeat
rows, 16 of 19. `check-gate-rows` anchored two plants on `check-staleness`: its
place in the `sh` list and its declared row in the `.ps1`. They anchor on
`check-step-bodies` and `check-assemble` now.

⭐ **CI on `a276766` read `check-formats` back on the Windows lane**, so the
runner's Python serves as the database reader: 21 passed, 0 failed, 0 skipped.

⭐ **And `check-connector` the sixth.**
[`../tools/check/connector.go`](../tools/check/connector.go) ports its 51 rows.
Every transcript is still hex composed by the harness itself, never written by
the observer. Two plants went into the connector, and both halves agreed on
each. Reading the reserved bytes one byte late failed the three reserved-byte
rows, 48 of 51. Decoding a bare `+` as a space failed the bare-plus row alone,
50 of 51. ⚠ The first spelling of that plant replaced a `str` in a `bytes`
value, so it crashed the connector instead of misreading it. The halves then
disagreed, because the `sh` half's `awk` printed every missing report into its
output; that was the plant's defect, and the halves agree over the corrected
one. ⚠ The port's first run failed its own default-report row: it looked for
the report before running the connector that writes it.

⭐ **CI on `628b88a` read `check-staleness` back on the Windows lane:** 22 passed,
0 failed, 0 skipped, 20 declared.

⛔ **And CI on `1166bc8` failed `check-connector` there, which is the port doing
its job.** The connector wrote its report with `open(out, "w")`, which on
Windows is a text-mode file, so every line ended `\r\n` and every value a
reader took out of it carried a carriage return. Its `--describe` went through
a text-mode stdout the same way, so the identifier read back was
`cpython-stdlib\r`. The connector's own read-back opened the file in text mode,
whose universal newlines turned `\r\n` back into `\n`, so it agreed with a file
that did not carry what was meant. ⭐ It writes bytes now on every stream a
caller parses, and reads its report back as bytes. This was a defect in the
second connector on Windows that no Linux run could show; the Windows capture
path would have read every connector value with a trailing carriage return.
`check-connector`'s failing rows now print what they read, quoted, because the
Windows run named seventeen failures and nothing beside them.

⚠ **`check-assemble` is the one portable row left, and it is not a straight
port.** Most of its 64 rows drive `assemble-capture`, a Rust example, over
lanes the harness writes, and those port as the six above did. ⛔ Its six
dispatch rows, added on 2026-10-14 under `CI-09`, run
`scripts/capture/assemble-dispatch.sh`, which is an `sh` script. A Go port would
have to run `sh` on the Windows lane, where a non-bash step's path may not carry
one. The other way is to move those six rows to a harness of their own, declared
class C like `check-bundle`, whose subject they share. ⚠ Undecided, and the
port waits on it rather than choosing by accident.

⭐ **Settled on 2026-10-15 and built the same day: the dispatch path is Go.**
`assemble-dispatch.sh` and the `assemble-bundle.sh` it called are
`bit-check assemble-dispatch` and `bit-check assemble-bundle`, in
[`../tools/check/dispatch.go`](../tools/check/dispatch.go). They are helpers
like `mine-repo`, not rows, because they write. `assemble-bundle.yml` builds
`bit-check` and runs it, and `check-assemble` and `check-bundle` drive it
through `store_bitcheck`. ⛔ **Compared before either half was deleted.** A
wrapper ran both halves on every input the two harnesses produce, 14 cases,
and compared exit codes, stdout, stderr and a digest of every bundle file.
Then 27 more cases written for the paths no harness reaches, an existing output among them: every name the
lane reader refuses, an install from another run, a scheme no record orders
under, three resolutions over two schemes, a hidden store entry, a store
without `raw/` or `profiles/`, a string and a `false` in the scheme, argument
errors, and a file that is not JSON. They all agree except two, and both are
intended:

- the usage line names the new command;
- a resolution file holding two JSON documents is refused. `jq` read the first
  and passed it; one file is one document.

Eight mutants of the helpers were each killed by a harness case. The first
mutant of the half-bundle cleanup did not compile, and its exit 2 was not
counted as a kill; a compiling one was killed. ⚠ The helpers find the
repository from the working directory, as every `bit-check` check does, where
the shell halves found it from their own path. Both harnesses `cd` to the
root, and the workflow step runs from the checkout.

⚠ **A review found two small things, both fixed before the push.** A signal
skipped the deferred removal, so an interrupted run could leave a half-built
bundle; the shell halves trapped INT and TERM. Now a handler removes it and
exits 2. That was driven, not cased, because it depends on timing:
SIGTERM was sent at 50 ms steps through the window. The fix left no half
bundle; a mutant without the handler left 4 in 12 runs. And a failed copy
exited 2 with nothing said, where `cp` had named it. Next: `check-assemble`
ports whole.

Prove: ⛔ **Rewritten with the Approach.** Every portable-subject row is a
`bit-check` row on the Windows lane, each port compared against its `sh` half
before that half is deleted. `check-store` runs on Windows with a plant set
whose every plant is made there. Every row still declared names an event in its
reason that `check-gate-rows` can read.

⚠ **Superseded, kept for the record:** *Write the missing halves, starting with
the ones whose subject is not platform-specific at all. ⚠ `check-store` plants a
symbolic link and a named pipe, which an unprivileged Windows session cannot
create; that one needs its plant set reconsidered rather than translated, and a
half that silently skipped two plants would report a smaller pass under the
same name.*

### ⭐ The first step is ONE library, not fifteen harnesses. Found 2026-09-09

⛔ **Every declared-unavailable harness sources
[`../scripts/corpus/store-lib.sh`](../scripts/corpus/store-lib.sh), and there is
no PowerShell twin of it.** `check-cache`, `check-corpus`, `check-store`,
`check-indexes`, `check-release`, `check-formats`, `check-publish`,
`check-access`, `check-catalogue`, `check-examples`, `check-handbook`,
`check-staleness`, `check-release-route`, `check-capture-client` and
`check-step-bodies` all begin the same way. ⭐ Measured:
`find scripts -name '*lib*.ps1'` returns nothing, while twelve `check-*.ps1`
files exist - every one of them a harness that needs no shared library.

⭐ **So this entry is one library and then thin twins, rather than fifteen
translations.** `store-lib.sh` is 321 lines and twelve functions:
`store_require`, `store_build`, `store_workdir`, `row`, `fail`, `pass`, `place`,
`tree_digest`, `tree_files`, `replace_once`, `store_probe_guards` and
`store_report`. ⚠ Four of those are the mutation-probe machinery, and
`check-twins` compares the two halves' answers per planted mutation - so a
`store-lib.ps1` whose `replace_once` or `store_probe_guards` differed in
semantics would make every twin that used it disagree at once. That is an
argument for writing the library carefully first, and against porting a harness
by inlining the few helpers it happens to need.

⚠ **Not every declared row is a missing half.** `check-examples` runs the ```sh
fenced blocks of `docs/consuming.md` as a reader would; there is no `sh` on a
Windows runner, so its `n/a` is a fact about the platform rather than work
nobody has done. ⛔ Counting all eighteen declared rows as this entry's backlog
overstates it, and the sweep should say which are which before any are written.

Prove: `pwsh -NoProfile -File scripts/common/check-gate.ps1 -Strict` passes with
fewer declared rows than it has today, each new half is mutation-proven against
the same plants as its twin, and `check-twins` compares the pair per planted
mutation rather than on a clean tree.

### ⭐ THE SWEEP, done 2026-09-09: the declared rows are FOUR kinds, not one

⛔ **This entry's backlog was every declared row, and that overstates it.** The
Approach already said `check-store` needs its plants reconsidered rather than
translated, and the paragraph below it said `check-examples` is a platform fact.
Nothing had gone through the rest. ⚠ Measured by reading what each harness
actually drives - the Rust examples it builds and the `sh` scripts it runs -
rather than by reading the reason each row declares, because those reasons were
written one at a time.

**A. The harness is the only missing half, and `store-lib.ps1` is most of it.**
Each drives Rust examples this project builds on both lanes, plus the shared
library. Nothing in the subject is `sh`.

| row | what it drives |
| --- | --- |
| `check-cache` | `cache-scenario`, and `check-licences`, which HAS a PowerShell twin |
| `check-corpus` | `build-store`, `validate-corpus`, `check-store` |
| `check-indexes` | `build-store`, `build-indexes` |
| `check-release` | `build-store`, `build-indexes`, `assemble-release` |
| `check-formats` | `build-store`, `build-formats`, `assemble-release` |
| `check-catalogue` | the four above plus `catalogue-lookup` |
| `check-staleness` | `build-store`, `resolve-stable`, `survey-staleness` |
| `check-assemble` | `assemble-capture` |

⭐ **So the library is the first step and it is not a translation of fifteen
harnesses.** `store-lib.sh` is twelve functions, four of them the mutation-probe
machinery, and `check-twins` compares the two halves' answers per planted
mutation - so a `store-lib.ps1` whose `replace_once` or `store_probe_guards`
differed in semantics would make every twin that used it disagree at once.

**B. The plants need reconsidering, not translating.** `check-store` plants a
symbolic link and a named pipe, which an unprivileged Windows session cannot
create. ⚠ A half that silently skipped two plants would report a smaller pass
under the same name.

**C. The SUBJECT is `sh` and has no PowerShell half, so the harness cannot
precede it.** Writing the harness first would be a twin proving a script that
does not exist.

| row | the subject it has no half of |
| --- | --- |
| `check-publish`, `check-access` | `publish-data.sh` |
| `check-capture-client` | `capture-client.sh` |
| `check-release-route` | `resolve-release.sh` |
| `check-source-route` | `resolve-source.sh` |

**D. A fact about the platform rather than work nobody has done.** ⛔ These
close on nothing this entry can write.

| row | why |
| --- | --- |
| `check-examples`, `check-handbook` | they RUN the ```sh fenced blocks of a document, and there is no `sh` on a Windows runner |
| `check-gate-rows` | it runs `check-gate.sh --rows`, which is the `sh` runner itself |
| `check-step-bodies` | it lifts and runs a Linux-only workflow's step bodies; the `pwsh` half of its subject is what it already runs |
| `check-capture` | its subject's PowerShell half exists and the capture workflow exercises it |

⚠ **No count of any of these is written here**, for the reason this entry
already records twice: a number in prose is a value in two places with nothing
comparing them, and both previous counts went stale. `check-gate.ps1` declares
each row with its own reason and `check-gate-rows` compares the two lists.

⭐ **`CI-06` built and ran the last clause of that Prove and did not keep it.**
Its three new rules were proved by planting into a scratch workflow, running both
halves of `check-project` on each plant, and comparing their exit codes and their
whole output: eleven cases, four refusals, six acceptances and a control either
side, all agreeing character for character. ⚠ It lived in a session scratch
directory and is therefore evidence rather than a control, which is exactly the
gap this entry names. ⛔ **And it is the cheap half of what `CI-06` had to do
instead**: `check-workflow.sh` proves the same three rules by running the whole
gate per plant, which cost the Linux lane eleven minutes. A permanent per-plant
twin harness would let that shrink back to one gate-level case.

### ⭐ THE LIBRARY EXISTS AND THE FIRST CLASS-A ROW IS REAL. 2026-09-10

⭐ **`scripts/corpus/store-lib.ps1` is the first step this entry named** - deleted
on 2026-09-29 with its last caller, when `CI-10` put both harness twins in the
Go binary - and `scripts/acquisition/check-cache.ps1` - deleted
on 2026-09-29, when `CI-10` put the harness in the Go binary - is the first
harness twin that proves it. `check-gate.ps1` runs `check-cache` as
a row rather than declaring it: measured on 2026-09-10, that lane went from 13
passed and 20 unavailable to **14 passed and 19 unavailable**, over 34 rows both
runners agree on.

⛔ **THE TWO HALVES CANNOT DISAGREE ABOUT THE CACHE AND THAT IS NOT THE POINT.**
Both drive the same Rust example, so the subject is identical by construction;
what they can differ on is the machinery underneath - the plant probes, the row
accounting and the verdict - which is exactly what the library supplies.
⚠ Their `--json` answers are byte-identical on a clean tree, and a clean tree
proves nothing about a pair, so three defects were planted in the PowerShell
library one at a time and the two halves compared on each:

| plant in `store-lib.ps1` | the halves |
| --- | --- |
| a multi-line literal accepted rather than refused | ⛔ disagree, caught |
| an ambiguous literal accepted rather than refused | ⛔ disagree, caught |
| a no-op edit reported as planted | ⛔ disagree, caught |

⛔ **AND `check-twins` COULD ONLY REACH `common/`.** Every pair it compared lived
there, so the directory was spelled once in `compare_pair` and each call site
named a bare file - and the first class-A twin is in `acquisition/`. A comparison
that could reach one directory would have left it uncompared, which is the shape
that whole file exists to refuse, arriving in its own plumbing. The paths are
relative to `scripts/` now and all twelve call sites carry their directory.

⛔ **THREE FUNCTIONS ARE DELIBERATELY ABSENT FROM THE LIBRARY.** `place`,
`tree_digest` and `tree_files` are used only by `check-store`, `check-corpus` and
`check-indexes`, and `check-store` is class B. ⚠ A function nothing calls is a
function nobody knows works, and shipping three of those would make the library
look more complete than it is measured to be; they land with the first twin that
exercises them.

### ⭐ AND A SECOND CLASS-A ROW, WHICH NEEDED NO NEW LIBRARY FUNCTION

⭐ **`check-catalogue` is a row on both lanes now**, and that is the sweep's
finding measured a second time: `check-cache` proved the library, and this
harness uses the same four calls with nothing added. That lane reports **15
passed and 19 unavailable** over 35 rows both runners agree on.

⛔ **WHERE THIS PAIR REALLY DIFFERS IS THE HARNESS, NOT THE SUBJECT.** Both
halves drive the same five Rust examples over the same fixture publication, so
the answers are identical by construction; what differs is how each PLANTS a
defect and restores the publication afterwards. A half that restored it
differently would report the same eighteen cases over a different experiment.

#### ⛔ AND THE LIMIT OF THE TWIN COMPARISON, MEASURED RATHER THAN ASSUMED

⛔ **`check-twins` COMPARES A VERDICT, SO IT CANNOT SEE A WEAKENED ASSERTION.**
Four defects were planted in the PowerShell half one at a time:

| plant | the pair |
| --- | --- |
| a plant's restore is removed | ⛔ disagree, caught |
| a case expects the wrong error code | ⛔ disagree, caught |
| a case is dropped from one half | ⛔ disagree, caught |
| a passing case's assertion is replaced by `$true` | ⭐ **agree, survived** |
| a second passing case's assertion is replaced by `$true` | ⭐ **agree, survived** |

⚠ **The last two are the comparison working as specified, not a defect in it.**
A case that already passed still passes, so the counts do not move and the JSON
is identical. ⛔ **So a twin comparison proves the two halves reach the same
VERDICTS and says nothing about whether either reached them for a reason.** That
is the same blind spot this entry already records from the other side - a rule
differing only on a defect the tree does not contain is invisible to it - and it
is why a new half is mutation-proved against its own subject as well as compared
against its twin.

⚠ **WHAT IS LEFT IS THE REST OF CLASS A**, which the table above this section
lists: `check-corpus`, `check-indexes`, `check-release`, `check-formats`,
`check-staleness` and `check-assemble`. ⛔ No count of them is written here, for
the reason this entry already records twice. ⚠ Three of those need
`tree_digest`, `tree_files` or `place`, which `store-lib.ps1` deliberately does
not carry yet.

### ⛔ What the door sweep found on 2026-09-08: the two runners' row lists

**Nothing compares the set of rows `check-gate.sh` runs against the set
`check-gate.ps1` names.** The `sh` runner derives its provers from a `for` list;
the `ps1` runner declares each one by hand as a run or an `Add-Unavailable`. A
prover added to the first and forgotten in the second is simply absent from the
Windows lane, which stays green because it never hears of it.

⚠ **It bit immediately.** `check-capture-client` went into the `sh` list, and the
Windows lane would have run twenty-six rows to the Linux lane's twenty-seven with
nothing anywhere naming the one that was missing. The row is written, so that
instance is closed; the class is not. ⛔ `check-twins` cannot see it either,
because it deliberately does not pair the gate runners: a harness that runs both
gates, running inside the gate, would re-enter itself.

⭐ **The cheap fix belongs with this entry's other half.** Give each runner a
mode that prints its row names and nothing else, then compare the two lists. That
needs no gate run at all, so it can live in `check-project` rather than in
`check-workflow`. Acceptance: with a prover added to the `sh` list alone the
comparison refuses, and with the pair in step it passes.

### ⭐ That half is done, 2026-09-09, and it is its own row rather than a rule in `check-project`

`--rows` and `-Rows` print the names each runner's own queue would have used and
run nothing, and
[`../scripts/common/check-gate-rows.sh`](../scripts/common/check-gate-rows.sh)
compares the two lists. ⛔ **The names come out of the calls that would have run
the checks**, so a row the mode does not print is a row that runner does not run;
a separate list would be the value in two places this comparison exists to catch.

⚠ **It is a gate row rather than a `check-project` rule because it RUNS both
runners**, which is a different kind of thing from reading the tree - and it can,
because a rows mode executes no check and therefore cannot re-enter the gate.
That is the same contract that keeps `check-twins` from pairing the two runners.

⚠ **It compares sets rather than sequences.** The `sh` half runs its checks
concurrently and the PowerShell half is serial, so the order a row appears in is
not a fact about what either lane runs; a duplicate still shows up, as a count
that does not match.

⛔ **The comparison refuses two short lists**, because a runner that printed
nothing agrees perfectly with another that printed nothing - which is the shape
`check-one-home` records about its own first run.

⭐ **Six cases, four of them plants:** a prover added to the `sh` list alone, one
dropped from it, a row declared on the PowerShell lane alone, a control either
side, and a probe that the rows output carries no verdict text of any kind, which
is what says the mode ran nothing.

### ⛔ Writing it found a live instance of this repository's own PowerShell hazard

**Measured 2026-09-09, on the first invocation.** `[switch]$Rows` collided with
the runner's own `$rows` accumulator: PowerShell variable names are
case-insensitive, so the parameter and the local are ONE variable, the ArrayList
was assigned to a `SwitchParameter`, and **every** invocation of that runner
failed to bind with `Cannot convert value "System.Collections.ArrayList"`.

⛔ **That is the third instance of one class here.** `docs/conventions/shell.md`
section 8 records `$args` inside a function, and `CI-03` records `[switch]$Marker`
against a local called `$marker` in `assert-disposable.ps1` - which failed to bind
in every mode, and was also found by running the guard once rather than by reading
it. ⚠ The rule that document already states is *name locals so they cannot
collide*, and the local is renamed rather than the parameter, because the flag has
to match the `sh` half's.

⚠ **And the two runners' row LABELS had to be made to agree on one row.** Each
half spelled its own flag - `check-no-secrets --public` against
`check-no-secrets -Public` - so a label built from the flag made the lists differ
on a row both lanes have, which is a false difference in the one comparison that
exists to find real ones. Both name the question now.

### ⛔ The first step is deleted, 2026-09-29, and the class-A route changed

⛔ **`store-lib.ps1` is gone, and so are both harness twins it served.** `CI-10`
ported `check-cache` and `check-catalogue` into the Go binary, compared each
against both halves first, and deleted them - which left the library with no
caller, and a function nothing calls is a function nobody knows works.

⭐ **So a class-A row closes by its harness JOINING THE BINARY, not by gaining a
PowerShell half.** Both of the rows this entry had turned real are real on the
Windows lane still, as `bit-check` rows. ⚠ The declared rows' reasons named the
old event - *it needs store-lib.ps1* - which would now never happen; each says
*it closes when its harness is ported into bit-check* instead, because a row
whose reason names the wrong event closes on the wrong day, which `CI-01`
records once already.

## CI-08: Runner-default drift, swept rather than waited for

Source: `$PSNativeCommandUseErrorActionPreference`, found by CI going red
Priority: P1 | Effort: L | Status: OPEN

Problem: A PowerShell default changed between 7.4 and 7.5 and turned a green
lane red over a script nobody had edited. That default was one of a class: every
behaviour a script inherits from its host rather than states is a defect waiting
for an image bump, and this project found the first one by being bitten.

Approach: Enumerate what the shell scripts, the PowerShell scripts and the
workflows inherit rather than state - shell options, output encodings, locale,
`$ErrorActionPreference` and its native-command companion, `git` defaults,
`cargo` environment variables, and the runner images' own tool versions. State
each one or record why inheriting it is safe. ⛔ The doctor already reports tool
versions; what is missing is the comparison against what the code assumes.

Prove: a rule per stated default in `check-project`, both halves, each
mutation-proven; and a driven pass that runs the gate under a deliberately
hostile environment - a different locale, a narrow console, `CARGO_TARGET_DIR`
set, and `TMPDIR` moved - with the same verdict.

⭐ **`CI-06` added the first three rules of this shape and left a harness gap it
owns rather than closes.** Nothing in the tree runs a capture step's *body*:
`check-workflow.sh` reads `capture.yml`'s step names, their order and the
*Capture* step's command, and executes steps out of `ci.yml` alone. So the two
restore blocks are proved statically by `check-project` and dynamically only by a
dispatch. ⚠ A harness that lifted each step body out of the workflow and ran it
under GitHub's own wrapper form - prepending `$ErrorActionPreference = 'stop'`,
appending `if ((Test-Path -LiteralPath variable:/LASTEXITCODE)) { exit
$LASTEXITCODE }`, invoking `pwsh -command ". '<file>'"` - against stubbed guards
would close it, and it is this entry's shape rather than `CI-06`'s: it is about
what a script inherits from its host. Acceptance would be that harness refusing
the *Restore the route* block as it stood on capture run 1 and accepting it as it
stands.

### ⭐ A capture step's body has now been run, by hand, and it found a defect

**Measured 2026-09-09.** `capture-client.yml`'s *Resolve the release artifact*
step was lifted out of the workflow with the same reader `check-workflow` already
uses and executed verbatim on this host, with `RUNNER_TEMP` pointed at a scratch
directory. ⛔ **It failed, and the cause was in the step rather than in anything
it called**: the block redirected into `$RUNNER_TEMP/release/url.txt` while the
directory was created by `resolve-release.sh` itself. A shell opens `>` targets
before it execs, so the step would have died on the redirection - on a runner,
after the claim, with a diagnosis naming a file rather than the mistake.

⚠ **No reader would have found it.** The script creates the directory and the
step names it; the two are correct separately and wrong composed, which is the
composition failure `docs/methodology/gate.md` part (b) exists for. Every
PowerShell rule in `check-project` passed the file, and so did every ordering
case in `check-workflow`.

⛔ **The harness gap this entry owns is still open, and running it by hand is why
its shape is now clear.** Two things stand between the measurement above and a
case:

1. ⚠ **This step reaches a vendor.** A case that fetched a release listing on
   every Linux lane would make CI depend on GitHub's release endpoint being up,
   which is a flake this repository would then be reasoning about instead of the
   rule. `resolve-release --listing` makes the source a file, and a **fixed step
   body cannot pass that flag** - so the seam a harness needs is one the step can
   inherit rather than be given, and that is a decision rather than an omission.
2. ⚠ **Most capture steps cannot run here at all.** The claim writes under
   `/var/lib`, the cut deletes a default route, and the capture needs both. This
   one is the first step of any capture workflow that a session host can execute,
   which is what made it available to try.

⭐ **What it establishes is that the reader is enough.** `step_command` lifted a
capture workflow's block correctly on the first attempt, so what the harness
needs is the environment and the seam, not a new parser.

### ⛔ The hang this entry owns: a second reading refuted, and a clock on it

**Client capture run 5, 2026-09-09.** The `release` route runs no package
operation of any kind, and *Upload the install logs* hung exactly as it does on
the `package` route. `TODO/clients.md` carries the table; what belongs here is
what it does to this entry.

⭐ **The step's own work finishes in five seconds and the step runs for
thirteen minutes.** The artifact was written at 03:17:23Z into a step that began
at 03:17:18Z, and the step had not returned at 03:30:43Z. Earlier runs could say
only that the artifact was complete; this one times it.

⛔ **Two named causes have now been refuted and neither was found by reading.**
`NEEDRESTART_MODE` fell to install logs saying `0 newly installed`; "the aria2
package rather than the route" fell to a route that touches no package index.
⚠ A third guess is what this entry exists to avoid: what is left is a runner
default nobody has swept, and the sweep is the Approach above rather than another
dispatch.

### ⛔ Run 6: the bound does not fire, and the step moved instead

**`timeout-minutes: 5` was the mitigation run 5 argued for, and run 6 measured it
not working.** Both lanes sat in that step far past the bound:

| lane | step began | its artifact was written | still running at |
| --- | --- | --- | --- |
| `package` | 03:50:11Z | 03:50:12Z, **one second in**, 3247 bytes | 04:07:39Z, **17 minutes** |
| `release` | 03:52:32Z | 03:52:34Z, **two seconds in**, 43179 bytes | 04:07:39Z, **15 minutes** |

⛔ **So GitHub's own per-step bound does not stop it**, which is a fact about the
failure and not only about the fix: `timeout-minutes` is enforced by the runner
process, so a step it cannot end is a runner that is not enforcing its own bound.
⚠ Naming what that implies would be the third guess this entry exists to avoid.

⭐ **The size question is closed by the same run.** The package lane's artifact is
**3247 bytes** - two logs and two stderr files - and it hangs identically to a
lane whose artifact is thirteen times larger. It is not the upload's volume.

⭐ **What did work needs no diagnosis at all: the step moved to the end of the
job.** It sat between the install and the route cut on the reasoning that a job
which never reached the capture would otherwise upload nothing - and `if:
always()` runs after a failed step wherever it sits, so the early position bought
nothing the last position does not. From there a hang costs the job's tail rather
than the measurement, because the capture and the evidence upload have already
happened. ⚠ `check-workflow` asserts the new constraint the move creates: every
upload is after *Restore the route*, because between the cut and the restore this
host has no way off itself.

⚠ **The bound is removed rather than kept alongside.** A bound measured not to
bound the one failure it was added for is decoration, and decoration in a
workflow reads as a control the next reader will trust.

⚠ That is a mitigation and not an answer, and this entry stays open on the answer.

### ⚠ Three routes to the answer that were tried and did not give one

⛔ **The hung jobs' logs are not retrievable.** An authenticated read of both of
run 6's jobs answers **HTTP 404**, as it does for run 4's aria2 job: a job whose
runner was killed or cancelled has no log to fetch. ⭐ That is why the artifact
route matters and why the install logs go up at all - the log is the one piece of
evidence a killed job does not leave, and this is now measured rather than
inferred from two dispatches that "lost their diagnosis".

⚠ **The last line of a job that DID complete is `Cleaning up orphan
processes`**, read from run 4's qbittorrent job. That is the runner's own
post-job step and it appears on every successful job, so it is a pointer and not
a finding: it says the runner tracks processes a step leaves behind, which is a
place to look rather than a cause.

⛔ **And the aria2 package install leaves nothing running, measured here.** The
adapter's package route was run directly on this host - no claim, no
`install-client`, because installing a product and asking its version is not a
capture - and the process count was **82 before and 82 after**, with nothing
matching `apt`, `dpkg`, `aria2`, `needrestart` or `unattended` alive afterwards.
⚠ This host is not the runner image, and it proved that in the same run: `aria2`
was **absent** here, so the install actually installed, where on `ubuntu-24.04`
it is a no-op. So the negative result is about this host and narrows rather than
settles.

⚠ **What would discriminate is a non-aria2 job reaching that step**, which every
completed job so far has passed through without hanging. With the step at the end
of the job, a `transmission` lane on the `release` route reaches it after an
install that refuses by design - so one dispatch would say whether the step or
the target is the subject. That is a case this entry can name rather than a guess
about a mechanism.

### ⭐ Run 7 ran that case, and it moves the boundary twice

**Four lanes: two transmission and two aria2, over both routes.**

⛔ **The step is not the subject.** Both transmission lanes ran *Upload the
install logs* in **one second** - one after a green capture and one after an
install that refused - in the same run, on the same image, as two aria2 lanes
that hung.

⛔ **And the hang is not attached to that step at all.** Moved to the end of the
job, it stopped hanging; what hung instead was *Install the client*, the step it
used to follow. Run 6's aria2 package install took **six seconds**; run 7's took
**ten minutes** and never completed. ⚠ Nothing about the install changed between
those two runs except which step comes after it.

⭐ **So the subject is the boundary after the aria2 install, not any particular
action.** That is a sharper statement than four dispatches could make and it is
what this entry now owns: whatever the aria2 install leaves behind, the runner
does not finish the adjacent step over it, and it does so whether that step is a
`uses:` upload or a `run:` block.

⚠ **Two local reproductions came back negative and neither settles it.** The
aria2 package route run directly on this host left 82 processes before and after,
and `aria2c --version` under the same bound `install-client` uses left 78 before
and after. ⛔ This host is not the runner image - `aria2` was absent here, so the
install actually installed, where on `ubuntu-24.04` it is a no-op - so both are
about a different code path than the one that hangs.

⚠ **And the move has a cost this entry records rather than trades away.** With
the install step hanging, run 7 produced no aria2 artifact at all, where runs 3
to 6 produced one. The four already collected say what that route does; what was
bought instead is where the hang sits.

### The second half of the Prove is done, measured 2026-09-08

⭐ **The gate gives the same verdict under a hostile environment.**
`CARGO_TARGET_DIR` pointed elsewhere, `TMPDIR` moved, `COLUMNS=80` and the locale
changed: `27 checks: 26 passed, 0 failed, 1 skipped, 0 unavailable`, row for row
identical to the ordinary run. ⛔ `CARGO_TARGET_DIR` is the one that had fired
before - it once put every built example where the harnesses did not look and
five provers exited 2 at once - so this is a regression that stayed fixed rather
than a variable nobody had tried.

⛔ **The first locale run tested nothing, and finding that out is what asking
"what would have made this fire" is for.** `LC_ALL=C` on a host whose locale is
already `POSIX` changes nothing, which is a trap
[`../docs/conventions/shell.md`](../docs/conventions/shell.md) states in as many
words about a different check. The host was measured - `LANG` empty, `LC_CTYPE`
`POSIX` - and the run repeated under `C.utf8`, which is the setting that differs.
Same verdict again.

### Three defaults are stated rather than inherited now

⭐ **`set -u` is the first code line of every executable script**, checked in
both halves and mutation-proved twice: a script without it, and a script that has
it below something else. ⚠ The rule reads the FIRST code line rather than
anywhere in the file, because this tree contains a heredoc whose body begins
`set -u`, so a looser rule would accept a script whose only `set -u` belongs to a
stub it writes. Measured before writing it: the convention already held in all
forty, which is what makes the precise rule the true one.
⚠ [`../scripts/corpus/store-lib.sh`](../scripts/corpus/store-lib.sh) is exempt by
name and for a reason - it is sourced, so an option set there changes the
caller's shell and stays changed, the same shared-namespace hazard that made this
repository prefix that library's globals.

⭐ **A cargo output path is asked for, never composed.** Five places compose one
and all five honour `CARGO_TARGET_DIR`; a sixth that did not is refused now, in
both halves. That is `CI-01`'s defect turned into a rule: two places composed
that path and fixing one left the other.

⭐ **A tool version the code assumes is compared against what installs it.**
`check-project` reads `SHFMT_VERSION` out of
[`../scripts/doctor/provision.sh`](../scripts/doctor/provision.sh) and refuses any
workflow or script naming a different one. That is the first row of "the runner
images' own tool versions" in the Approach above.

### ⭐ The harness gap is closed, and the instrument it needed was not the one this entry asked for

**Written 2026-09-09.**
[`../scripts/ci/check-step-bodies.sh`](../scripts/ci/check-step-bodies.sh) lifts
a block out of `capture.yml` or `capture-client.yml` and runs it under GitHub's
own form: `bash -e` for a default `run:`, and for a `shell: pwsh` step the
`pwsh -command ". '<file>'"` invocation with `$ErrorActionPreference = 'stop'`
prepended and the residual-exit epilogue appended.

⭐ **The acceptance this entry wrote is met.** It refuses the Windows *Restore
the route* block in the form that failed capture run 1 - the explicit `exit 0`
taken away, so the wrapper reads the guard's own refusal as the step's verdict -
and accepts it as it stands, over a stub guard that refuses. Both halves of the
`sh` block's three-code reading are cases too.

⛔ **But the reason a body was worth RUNNING turned out to be a second fact this
entry had not stated: a step does not end when its command exits.** The runner
reads a step's output through a pipe, so the step is over when the command has
gone *and* that pipe has reached end of file. A process the body leaves behind
holding the step's stdout keeps it open, with an exit code of 0 sitting in it.
⚠ Every harness in this repository redirects a step body's output to a **file**,
and a file has no reader to wait on, so no existing check could see the class at
all.

⭐ **So the harness records two facts per body and keeps them apart**, and the
two failures are reported differently: a body that refuses, and a body that will
not end.

⚠ **What sent the instrument there is a measurement rather than a hypothesis.**
Client capture runs 6 and 7 ran the same aria2 install command from commits whose
only functional difference is where a later step sits - `git diff` over the two
says so, and nothing in `install-client.sh` or the adapter changed - and that
step took **six seconds** in one and had not returned after **sixteen minutes**
in the other. A command whose duration depends on which step follows it is not a
command that is slow. ⛔ **That is not a fourth cause and is not offered as one.**
It is where an instrument can be pointed, which is what this entry asked for
instead of another dispatch.

⛔ **And the same block is proved to hang here, in the step it hangs in there.**
`capture-client.yml`'s *Install the client* body, run against this repository's
real `install-client.sh` and a stub product, ends over a product that behaves and
does **not** end over one that leaves a single process on the step's output.

⚠ **What it does not establish is a runner image.** A body that ends here is not
a body that ends there. What it proves is the shape, and that no check in this
tree could previously tell the two apart.

⚠ **THE WRAPPER PASSES `-NoProfile` AND GITHUB'S DOES NOT, and that departure is
stated rather than silent.** A `shell: pwsh` step really does inherit whatever
profile its host has, so the faithful invocation would omit the flag; a gate row
that loaded a contributor's profile is a row that goes red for something outside
this repository, and every other `pwsh` invocation here passes the flag for
exactly that reason. ⛔ **So what this harness proves about a `pwsh` body is
proved with no profile loaded**, and a defect a profile would cause is outside
what it can see. Decided by the operator on 2026-09-09, after the flag was
briefly removed on the fidelity argument alone.

#### Guard mutation over the instrument, 2026-09-09

Seven plants into a scratch copy of the whole tree, one at a time, each verified
to have applied before its result was believed, with the clean copy run either
side.

| plant | outcome |
| --- | --- |
| the instrument always answers CLOSED | ⭐ refused, by the two cases that assert a hang |
| the instrument always answers NOT CLOSED | ⭐ refused, by every case that asserts a body ends |
| the pipe becomes an ordinary file, which is what every other harness uses | ⭐ refused, by the same two |
| GitHub's residual-exit epilogue dropped from the wrapper | ⭐ refused, by the run-1 case alone |
| the close bound raised above the planted leak | ⭐ refused, by the two hang cases |
| the close bound set to zero | ⭐ **could not run**, which is the guard below |
| the `pwsh` prologue dropped | ⚠ **survived**, and the reason is this entry's own rule |

⚠ **A surviving plant is a question, and this one has an answer.** Dropping the
prologue changes nothing for these bodies because every `pwsh` block in this tree
sets `$ErrorActionPreference` itself - which is the rule `check-project` enforces
and this entry wrote. The plant would bite the day that rule stopped holding, so
it is recorded rather than turned into a case that passes for a reason of its
own.

⛔ **`timeout 0` MEANS NO LIMIT, and a ceiling edited to zero is therefore an
infinite one.** Found by planting it: the harness waited for the leaking body
until the leak ended by itself and then reported the pipe as closed. A bound
below one second is refused now with exit 2.

⚠ **Two earlier shapes of the wait were measured and rejected**, and both are
this harness's own subject arriving in its instrument. A `kill -0` poll on a
one-second granularity cost a whole second on every case that DID close, because
a reader whose last writer has just gone has usually not been scheduled yet - and
on a saturated host it could report a closed pipe as open, which is a red row for
no defect in a gate that runs thirty checks at once. A `( sleep N && kill ) &`
watchdog fixed the cost and left an orphaned `sleep` per case. What ships is
`timeout N tail --pid` on the reader, which blocks, bounds, and leaves nothing.

⭐ **And the reader is one home now.**
[`../scripts/ci/workflow-step.sh`](../scripts/ci/workflow-step.sh) is what lifts
a step out of a workflow, and `check-workflow.sh` asks it rather than carrying a
second parser. ⚠ The extraction was compared against the reader it replaced over
every job and step in every workflow here - sixty-six of them - and the output is
identical on all of them. ⛔ It answers with an exit status rather than a string,
and `scripts/README.md` names the three it distinguishes. `check-workflow.sh`
collapses two of them, as it always has; the new harness does not, because a step
it names and the workflow no longer has is rot rather than a rule that quietly
passed.

### ⭐ And what the instrument says to do about the install step

**Changed 2026-09-09.** *Install the client* sends the route's output to a file
and prints it afterwards, so nothing the product spawns inherits the step's own
output pipe. Three cases in the harness carry it: the block ends over a product
that behaves, it ends over one that leaks a single process, and a planted `3>&1`
- one extra descriptor onto the step's own output, written before the
redirection - brings the hang straight back. ⛔ Without that third case the
second passes equally over a block that never had the problem.

⚠ **THE ORDER OF THE REDIRECTIONS IS THE PLANT, and the first version of it was
wrong**: a shell applies them left to right, so `>log 2>&1 3>&1` points fd 3 at
the log rather than at the step's pipe, and the case reported the hang not
happening over a plant that had duplicated the wrong file. It is the same class
as an in-place `sed` editing a line nobody named - a plant that applied
somewhere other than where the case says.

⭐ **The step also names what is holding the file now.**
[`../scripts/ci/report-holders.sh`](../scripts/ci/report-holders.sh) lists every
process with an open descriptor on the route's log and prints the process table
beside it, into `holders.log` in the workdir the `always()` upload collects. ⚠ It
runs under `sudo` because the install did: an unprivileged reader of `/proc`
reports nobody holding a file that root processes are holding, which is a
diagnostic that answers confidently and wrongly. Driven here against a known
holder: it named the process by pid, comm and arguments, reported `0 holder(s)`
for a path nothing held, and exited 2 with no arguments.

⛔ **It is a mitigation and its outcome is the measurement.** If an aria2 lane
now gets past *Install the client*, the class is what the hang was; if it hangs
anyway, this reading is refuted by something that separates it rather than by
another guess. ⚠ Either way the lane leaves `holders.log` behind, which is the
first positive evidence this project will have about what that route leaves
running.

### ⛔ Run 8 answered it, and the answer is no

**Measured 2026-09-09.** Both aria2 lanes sat in *Install the client* from
09:13:18Z and had not returned at 09:35:17Z, twenty-two minutes later, on a run
where nothing the route spawned could inherit the step's output pipe. ⛔ **So
that class is refuted**, the way the three before it were.

⭐ **What the same run establishes is where the step is NOT.** `install-client`
bounds the install call at 420 seconds and kills 20 seconds after that, so the
last moment that call could have ended is 440 seconds in; the step ran three
times as long. ⛔ Whatever is slow or stopped is outside the bounded call, which
leaves the unbounded parts of that script - its command substitutions and its
digests - and the step itself.

⚠ **The dispatch changed two things at once and that is a defect in the
experiment, not only in the record.** A holder report was added to the same step,
it walks every process's descriptors, and it runs on the branch a 22-minute step
cannot distinguish from a slow install. ⛔ It is bounded by `timeout 60` now, and
the step no longer waits on the install at all.

### ⭐ So the next step bounds itself and records a timeline from inside the window

**Written 2026-09-09.** The install runs in the background and the step's own
shell watches it, because that shell is outside every bound that has failed:
`timeout` ends its own child and cannot end a shell blocked in a substitution
around it, and `timeout-minutes` is the runner's and was measured on run 6 not to
end the step at all.

⭐ **Every tick leaves a process snapshot** - `pid,ppid,pgid,stat,etimes,comm,args`
every five seconds into `watchdog.log` in the uploaded workdir. ⚠ `etimes` beside
`stat` is what separates a command that is SLOW from one that is stopped, which
is the question this entry cannot answer from timings alone.

⛔ **And the deadline is what makes any of it reach a reader.** A job whose runner
is killed leaves no log and no artifact; a step that ends leaves both. Measured
again on run 8: rule 8's route answers **302** for a running job's log and
redirects to a blob that answers `BlobNotFound`, while a general-purpose GitHub
tool answers a bare **404** for the same job - so the log is not written until
the job finishes, and only an uploaded artifact survives.

⭐ **The bound is seen to fire rather than assumed to.** Two cases in
`check-step-bodies`: a product made slower than the deadline is killed and the
step still ENDS, and a product slower than one tick but inside the deadline is
not killed. ⚠ Without the second, the first passes equally over a block that
kills every install it is given.

### ⛔ Run 9: the watchdog in the step's own shell did not fire either

**Dispatched 2026-09-09 as `["aria2","transmission"] × ["package"]` on
`42bd206`**, with transmission as the control.

| lane | outcome |
| --- | --- |
| `transmission` `package` | ⭐ **green: a complete capture**, install 121s, whole job three minutes |
| `aria2` `package` | ⛔ *Install the client* began 10:13:54Z and had not returned at 10:28:45Z - **fifteen minutes** |

⛔ **The deadline was 480 seconds and it passed by seven minutes.** That loop runs
in the step's own shell, so a shell that was looping would have fired it. ⚠ So
the step's shell is not reaching the loop, or is not running at all - which is a
fact about the step rather than about the install, and it is the third bound
measured not to fire here.

⭐ **And transmission is the control that keeps this attributable.** Same run,
same image, same step, same `apt-get`: 121 seconds and green. ⚠ That install is
itself far slower than the 14 seconds earlier runs recorded, so these hosts are
slow today - and slow is exactly what the aria2 lane is not, because a slow
install would have been ended by `install-client`'s own 420-second bound.

### ⭐ So the bound moves outside the shell entirely

[`../scripts/acquisition/install-step.sh`](../scripts/acquisition/install-step.sh)
holds what the step used to do inline, and the workflow runs it as
`timeout -k 30 540 sh …`. ⛔ **The bounded process is now the step's own**, so
nothing inside the step has to be reachable for the bound to work.

⚠ **What that buys is not a faster failure. It is a job that reaches its uploads
at all**: runs 7, 8 and 9 each ended with no log and no artifact, so an aria2
lane that hangs has taught nothing three times running. ⭐ A step that ends leaves
`watchdog.log`, `step.log` and `holders.log` behind, and `stat` beside `etimes`
is what separates a command that is slow from one that is stopped.

⭐ **Two cases hold it**, both in `check-step-bodies`: the bound around the step's
own process fires and the step ends with coreutils' 124, and a product that
finishes inside the bound is not killed.

### ⛔ Run 10: that bound did not fire either, and four is a pattern

**Dispatched 2026-09-09 as `["aria2"] × ["package","release"]` on `3b5793d`.**
Both lanes entered *Install the client* at 10:57:50Z and 10:57:58Z, and neither
had returned at 11:10:12Z - **twelve minutes** against a `timeout -k 30 540`
around the step's own command, which would have ended it at 570 seconds.

⛔ **So four independent bounds have now been measured not to fire on this
step**, each at a different level:

| the bound | where it sits | measured |
| --- | --- | --- |
| `timeout -k 20 420` on the adapter call | inside `install-client` | run 8: the step ran three times past it |
| `timeout-minutes: 5` | the runner's, on the step | run 6: the step ran seventeen minutes |
| a watchdog loop with a 480s deadline | the step's own shell | run 9: passed by seven minutes |
| `timeout -k 30 540` on the step's command | the step's own process | run 10: passed by three minutes |

⛔ **And no such job has ever produced a log or an artifact** - runs 7, 8, 9 and
10. The log blob is never written, which is why rule 8's route answers 302 to a
`BlobNotFound`, and the run ends `cancelled` around the job timeout.

⚠ **That combination is what separates a fifth reading from the four guesses
before it.** A step that were merely stuck would still leave a runner able to
enforce one of four bounds and to upload a log; nothing here does either. What
the evidence describes is a job that stops being served, not a command that does
not return - and `TODO/clients.md` carries transmission passing through the same
step on the same image in the same runs.

⛔ **It is still a reading and it is not recorded as a cause.** What it changes is
where to look: at the host rather than at the shell.

### ⭐ So the next dispatch bisects with step NAMES, which is the one signal that survives

When nothing inside a job survives - no log, no artifact, no bound - the only
thing an outside reader still has is **which step the API last reported in
progress**. Three bounded probes now run before the install, each named, so a job
that wedges localises itself to a handful of commands without needing any of the
things these runs do not produce.

⚠ **The third probe is the asymmetry the whole thread rests on.**
`install-client` asks the adapter for a version *before* the route runs; `aria2`
ships on `ubuntu-24.04` and `transmission-daemon` does not, so on an aria2 lane
that call executes the preinstalled product and on a transmission lane it finds
no binary and refuses without running anything. ⛔ That difference is present in
every hung run and absent from every green one, and no dispatch has yet isolated
it.

⚠ **The probes guard nothing and each ends in a `true`**, so a diagnostic cannot
fail a capture. They come out when the question is answered.

### Acceptance for the harness, run on 2026-09-09

- `sh scripts/ci/check-step-bodies.sh`
- `sh scripts/common/check-gate.sh`
- `sh scripts/ci/check-workflow.sh`

⛔ **What is left of this entry is the sweep**, unchanged: the Approach asks for
an enumeration of what the shell scripts, the PowerShell scripts and the
workflows inherit rather than state, and three defaults are stated so far. ⚠ And
a second question this entry owns: the Linux lane pins `shfmt` and takes
`shellcheck` and `pwsh` from the runner image, so a session host runs a MORE
pinned set of tools than the lane it exists to match.

### ⭐ THE SWEEP IS AN INSTRUMENT NOW, NOT A READING. 2026-09-10

⛔ **This entry's Approach asks for an enumeration of what the scripts inherit
rather than state, and an enumeration written by READING is a list of the
defaults somebody thought of.**
[`../scripts/ci/check-defaults.sh`](../scripts/ci/check-defaults.sh) runs five
checks under six environments and compares their machine-readable answer against
the baseline: a difference is a default that check inherits, named by the
variable that produced it, whether or not anybody had thought of it.

| the environment | what it would change if a check read it |
| --- | --- |
| `LC_ALL=C` | byte collation and byte character classes |
| `LC_ALL=C.utf8` | multibyte collation and classes |
| `TMPDIR` | where a harness puts the tree it plants in |
| `CARGO_TARGET_DIR` | where a built example lands - the one that has already bitten |
| `POSIXLY_CORRECT` | GNU tool behaviour in several places |
| `SOURCE_DATE_EPOCH` | read by build tooling that wants a fixed clock |

⭐ **The result is that all five answer identically under all six**, and that
sentence is worth exactly as much as the controls behind it.

#### ⛔ Two controls, because "no difference" and "no experiment" look identical

⭐ **A probe that READS the variable is run first and must answer differently.**
An instrument whose environment never reached the child would report perfect
agreement over every subject; this exits **2**, not 0, if that probe agrees -
`could not run` rather than a subject that passed.

⭐ **And a planted environment-sensitive answer is caught.** `check-licences` was
made to print `SOURCE_DATE_EPOCH` in its own JSON, and the run named the subject,
the variable and both answers - and did **not** fire on the other five.

#### ⛔ AND ONE ROW IS BLIND ON THIS HOST, MEASURED BY REPLANTING THE REAL DEFECT

⛔ **The historical `CARGO_TARGET_DIR` defect was replanted - `store_build`
reverted to composing `$ROOT/target` and ignoring the variable - AND THIS FILE
DID NOT CATCH IT.** `$ROOT/target` already held the example from an earlier
build, so the defective path resolved to a **stale binary** and the check
answered normally.

⚠ **So that row fires on a clean checkout and is blind on any host that has built
before**, which is every host a contributor runs it on twice. ⭐ The condition is
reported as its own row rather than hidden: a reader told "the same answer under
all 6" without being told that one of the six could not have answered otherwise
has been told something weaker than it sounds.

#### ⚠ Two things this instrument cannot perturb, stated rather than dropped

⛔ **`IFS` does not survive into a child.** Measured here:
`env IFS=: sh -c 'printf %s "$IFS"'` prints the default, because a shell resets
it at startup. A case setting it would report a guard proved by a value the child
never saw, so it is a control row saying so.

⚠ **`umask` is a shell attribute rather than an environment variable**, so `env`
cannot pass it, and setting it in this harness would change the modes of
everything the harness itself writes. It belongs to whatever check reads a file
mode, and nothing here does.

#### ⚠ What it costs, and the subject that was dropped for it

**28 seconds**, because every subject runs once per environment.

⛔ **AND IT IS NOT A GATE ROW, BECAUSE IT WAS ONE AND THE LANE MEASURED THE
COST.** The paragraph here said the 28 seconds are "concurrent with the rest of
the gate and free in local wall-clock terms, and NOT free inside `check-workflow`,
which runs the whole gate about ten times and is already the CI wall clock" - and
then put it in the gate anyway. ⚠ CI run 116's *Workflow acceptance* job ran
**30m15s** and was cancelled at its 30-minute bound, having been about 20 minutes
before. ⭐ It is its own CI step now, which is exactly how `check-workflow` itself
is handled and for a related reason.

⚠ **The cost of a gate row is multiplied by ten, and that number belongs beside
any future one.** A row worth 28 seconds locally is worth nearly five minutes of
the CI wall clock.

#### ⛔ AND THAT READING WAS WRONG. The 28 seconds were not the cause

⛔ **THE JOB HIT ITS BOUND AGAIN AFTER `check-defaults` LEFT THE GATE**, at
30m15s both times. Moving it saved about five minutes and did not bring the job
back under 30, which is what makes the first diagnosis a hypothesis that was
tested rather than a conclusion.

⭐ **Measured instead of guessed a second time**: `check-capture-client` is
**105 seconds**, where this repository's own gate comment records it at **48 at
`SECS=5`**. `OBS-07`'s connector refusals were given cases - eight stub-connector
captures and three more - and each `run_case` runs a whole capture. That is 57
seconds the gate gained, multiplied into about nine and a half minutes here.

⚠ **The bound is 45 now and the growth is named rather than absorbed.** Raising a
timeout to fit growth is masking unless the growth is written down; what bought
those 57 seconds is six refusals that had never fired, and `CI-01`'s residual
already carries the real fix - sharding this harness across runners rather than
buying it more minutes.

⛔ **The lesson is the one this file keeps recording.** A gate row's local cost is
not its cost: the multiplier is `check-workflow`'s ten gate runs, and the two
socket harnesses run SERIALLY after the concurrent batch, so a second added there
is a second on the wall clock rather than a second shared with something else.

`check-docs` is the one subject left out:
it resolves links and parses fenced blocks, which is the least plausible thing
for a locale or a scratch directory to reach. ⚠ `check-markers` stays because it
decodes UTF-8 **by hand**, which is the most plausible.

#### ⛔ And a defect in this file, found by running it rather than by reading it

**`grep -c` over a file with no matches PRINTS `0` and EXITS 1**, so
`$(grep -c ... || printf 0)` ran the fallback as well and the value became two
lines. It went into a row, and `store_report` counted fourteen rows against eight
cases and refused. ⭐ That self-check is why this was a red line on the first run
rather than a miscounted report nobody read.

### ⛔ THE GATE WAS SMALLER THAN THE LANE AND NOTHING SAID SO. 2026-09-10

⛔ **CI run 113 failed on *Shell syntax and style* over one `SC2016`, on a commit
whose local gate had reported 34 of 35 passing minutes earlier.** The gate ran
neither `shellcheck` nor `shfmt` over the scripts. That is this entry's own
subject arriving from the other side: not a default inherited from the host, but
a STEP the lane runs and the local gate did not.

⚠ **And the workflow already claimed otherwise.** `.github/workflows/ci.yml`
installs the pinned `shfmt` on the Windows lane with the comment *"several of the
cases below run the gate, and the gate runs shell checks"*. No row did. ⛔ That is
the fourth instance this repository has recorded of a rule a document says it has
and does not, and the comment is corrected rather than deleted, because the
sentence is true now.

⭐ **[`../scripts/common/check-shell.sh`](../scripts/common/check-shell.sh) is the
row**, running CI's own command over every tracked script found rather than
listed. ⛔ Two rows and not one, because `&&` between two checks is one check and
a reader of a red row has to know which tool refused. ⭐ And a third that counts
what was swept: two clean tools over a `find` that stopped matching answer
exactly as they do over a clean tree.

⚠ **`provision.sh` had been installing both tools all along**, with a header
saying a host without them "runs a gate that is quietly smaller than CI's". The
tools were there; the row was not.

Guard mutation, one plant per tool, each verified to apply: a removed
`shellcheck` directive and an added indent. Both refused, and each row named
itself rather than the other.

### ⛔ A FIFTH, SIXTH AND SEVENTH BOUND MEASURED NOT TO FIRE. 2026-09-15

**`capture-client` runs 21 and 22, both on `40ed628`, both cancelled with no log
and no artifact.** The RELEASE lane's *Install the client* step ran for about
thirty minutes where runs 19 and 20 took **six seconds**, measured from the step
timings this entry's own route reads:

| run | commit | release install | verdict |
| --- | --- | ---: | --- |
| 19 | `c3c9f6c` | 12:38:25 to 12:38:31, **6s** | green |
| 20 | `95e90f5` | 14:12:58 to 14:13:04, **6s** | green |
| 21 | `40ed628` | 14:46:22, never completed | ⛔ cancelled at 15:15:51 |
| 22 | `40ed628` | 15:20:36, never completed | ⛔ cancelled |

⛔ **THREE MORE BOUNDS DID NOT END IT, which takes this entry's tally from four
to seven.** `install-client`'s inner 420 seconds should have refused at 14:53
and did not. The step's own `timeout -k 30 1080` - the bound this entry moved
outside the shell precisely because nothing inside it was reachable - should
have ended the process at 15:04 and did not. The job's `timeout-minutes: 25`
ended it at **thirty** minutes rather than twenty-five.

⛔ **AND A CANCELLED JOB WRITES NO LOG AT ALL.** Its log endpoint answered
`BlobNotFound` for four minutes of polling and never resolved, which is the same
answer `AGENTS.md` rule 8 records for a job that has not finished. ⚠ So runs 21
and 22 measured nothing about the build and cost a dispatch each - which is the
whole argument this entry already makes about runs 7, 8 and 9.

⛔ **A CAUSE WAS LOCATED, FIXED, AND THEN REFUTED BY THE NEXT RUN.** The release
route ran `curl -fsSL --retry 2` with **no time limit**, in four adapters at
once, and a stalled transfer with no limit waits forever - which is exactly the
shape of a six-second step becoming a thirty-minute one. ⚠ **It is not the
cause.** Run **23**, on `80ec75a`, carries `--max-time 300` on that fetch and
its release install ran past **twelve minutes**: a bound that would have refused
at five did not fire, so the step is not waiting in the fetch. ⭐ The fix is kept
because the rule is right on its own terms and `shell.md` section 9 states it;
this paragraph says what it did NOT buy rather than leaving a repair reading as
an explanation.

⚠ **What that leaves unexplained is the whole of it.** The release route after
the fetch is `chmod`, `cp`, `mv` and a `test -x`; `install-client` then runs its
rule-12 scan and asks the adapter for a version. One of those is where the step
sits, and no run has produced a log to say which - ⛔ which is the same wall
runs 7, 8 and 9 hit. ⭐ `install-step.sh` writes a `ps` timeline into the workdir
every five seconds precisely for this, and the workdir ships as the install
artifact: **the next dispatch that reaches an upload answers it in one file.**
Runs 21, 22 and 23 uploaded nothing because the job was cancelled first.

The original reasoning, kept because a corrected claim with its reasoning
deleted teaches nothing:
[`../docs/conventions/shell.md`](../docs/conventions/shell.md) section 9 had
stated that rule for as long as the adapters had been breaking it, and
`aria2-next.sh` has carried `--max-time 20` on its JSON-RPC call ninety lines
above the fetch that had none: the convention held on one of two paths into one
product. ⭐ `bit-check check-adapters` is the rule now rather than a comment, and
the gate is **39 checks**.

⛔ **THE DOOR SWEEP FOUND THE SAME DEFECT TWICE MORE, ONE DIRECTORY AWAY**, which
is the rule being narrower than the class it is about:

| where | verdict |
| --- | --- |
| `scripts/doctor/provision.sh` fetched every pinned tool with no limit | ⭐ bounded in the same change; out of `check-adapters`' scope because a stall there costs a session's start in front of somebody rather than a dispatch |
| `scripts/common/mine-repo.sh` and its `.ps1` twin clone with no bound | ⭐ **fixed 2026-09-17**, and not with a wrapper; see below |

⚠ **The revert was the finding.** `timeout` is not a bound on Windows -
`timeout.exe` is a PAUSE - so a `.ps1` wrapped that way would sleep for ten
minutes and then clone. The two halves would have needed two idioms,
`check-twins` compares that pair, and doing it during a wrap-up would have
shipped a Windows defect to avoid leaving a residual.

#### ⭐ AND IT IS FIXED, 2026-09-17, BY NOT USING A WRAPPER AT ALL

⛔ **THE PREMISE OF THE REVERT WAS THAT A BOUND MEANS A WRAPPER, AND IT DOES
NOT.** `git -c http.lowSpeedLimit -c http.lowSpeedTime` is git's own limit on a
transfer that has stopped moving - the same idea `curl --speed-limit
--speed-time` gives the adapters - and it is spelled identically on every
platform git runs on. ⭐ So the two halves carry ONE idiom, not two, and
`check-twins` has no difference to compare. ⚠ The numbers are this project's own
`1024` and `60` rather than new ones.

⭐ **Driven on 2026-09-17 against a listener that accepts a connection and never
answers**, which is the failure that waits forever:

| what | exit | elapsed |
| --- | ---: | ---: |
| the shipped settings, read out of the two files | 128 | **60s**, git abandoned it |
| the control: the same clone with the settings removed | 124 | **still waiting at 25s** |

⚠ **The settings were extracted from the shipped files rather than retyped**, and
compared between the halves: both carry `http.lowSpeedLimit=1024 -c
http.lowSpeedTime=60`, character for character. ⛔ `--selftest` is offline and
does not reach the clone, so `check-twins` cannot see this pair agree about it -
which is why the comparison is a driven measurement here rather than a claim.

⚠ **IT IS LOCATED RATHER THAN CONFIRMED, AND THE DIFFERENCE MATTERS HERE.** No
run since carries the fix. What was measured from this host is that the vendor's
release listing answers in 0.9 seconds and its asset in 0.6, so the endpoint is
reachable from somewhere - which is evidence about this host's network and not
about the runner's. ⛔ **The control is a re-run of run 20 at `95e90f5`**, the
same workflow without this session's commits: if it hangs too, the cause is
outside this repository and the bound is still the right fix for a different
reason. It was still running when this was written.

### ⭐ THE HANG IS REPRODUCED ON THIS HOST, AND TWO DEFECTS CARRY IT. 2026-09-16

⛔ **Thirteen `capture-client` dispatches have been cancelled rather than
answered** - runs 1, 3 to 10, 13, and 21 to 23, counted from the API on
2026-09-16 - and the last three are this hang's.
The instruction was to read the `ps` timeline before theorising again; what the
timeline needed first was a run that reached an upload, and what stopped that is
now measured rather than argued. ⚠ **Neither defect was found by reading the
install path.** Both were found by driving it as an UNPRIVILEGED user, which is
what a runner is and what no local drive had ever been: every session host here
is `root`, so every previous local pass ran the step with privileges the runner
does not have.

⛔ **DEFECT 1: A BOUND INSIDE `$( )` IS NOT A BOUND.** A command substitution ends
when its PIPE reaches end of file, not when the command exits. `install-client`
reached its adapter four times and **every one of them handed the adapter a
pipe**:

| line | call | bound it carried |
| --- | --- | --- |
| `TARGET=$(sh "$ADAPTER" describe \| awk …)` | before any other call | ⛔ **none** |
| `PREEXISTING=$(timeout … version)` | before the route | fires, and `$( )` keeps reading |
| `adapter_binary()`, called twice | around the install | ⛔ **none** |
| `VERSION=$(timeout … version)` | after the route | fires, and `$( )` keeps reading |

⚠ **Measured**: an adapter whose `version` leaves one `sleep` behind was still
blocking its substitution at **25 seconds** under a **4-second** bound. ⛔ **And
this file's own header claimed the opposite** - *"every adapter call is bounded"*
and *"this is the one call site every adapter passes through"* - which is this
project's own recurring class, a rule a document says it has and does not.

⭐ **MUTATION-PROVED, the same plant against both halves.** An adapter whose
`version` leaves a process behind, driven through the real `install-client`:

| tree | exit | elapsed | output |
| --- | --- | ---: | --- |
| `HEAD` (`ff14fe5`) | **124** | still blocked at **70s** | ⛔ none at all |
| with `adapter_run` | **0** | **0s** | the full install record |

⛔ That control is runs 21, 22 and 23's signature exactly: no log, no artifact,
and a bound that had fired.

⛔ **DEFECT 2: AN UNPRIVILEGED BOUND CANNOT END A ROOT PROCESS.** Every bound this
entry has measured failing was issued by a uid-1001 process at a tree running as
root, and the kernel refuses that signal. Measured here: `timeout -k 2 5` around
`sudo -E sh -c 'sleep 120'` exited **124 on schedule** and left the root `sleep`
alive with **PPID 1** - orphaned, past its KILL grace, still running.

⭐ **That predicts the one number this entry could not explain.** Run 21's job
carried `timeout-minutes: 25` and ended at **thirty**: the runner asked the step
to stop, the root tree did not answer, and GitHub force-terminated after its own
grace. ⚠ A cancelled job uploads nothing, which is why runs 21, 22 and 23
measured nothing.

⭐ **The bound is inside the `sudo` now**, which is the one place seven bounds had
not been put. Driven as `runnerlike` (uid 1001) against an install that blocks
for 600s under a 20s step bound: **exit 124 at 21 seconds, with
`watchdog.log` carrying four samples and `holders.log` naming the survivor.**
⛔ That is the whole deliverable - a hung step that ENDS and leaves its timeline
on the runner for the upload to collect.

#### ⛔ THE INSTALL PATH WAS BYTE-IDENTICAL ACROSS THE GREEN/HUNG BOUNDARY

⛔ **The recorded lead - *the hang correlates with this session's commits* - is
refuted by the tree, in one command.** `git diff 95e90f5 40ed628` over
`scripts/acquisition/`, `scripts/capture/adapters/` and `capture-client.yml` is
**empty**, and so is the same diff to `80ec75a`. Run 20 installed in **six
seconds** and runs 21 and 23 hung for thirty minutes **on byte-identical install
code**.

⚠ **What those commits did touch** is `capture-client.sh`, `assemble-capture.rs`
and two check scripts. None of them is read by *Install the client*: the capture
script runs two steps later, and the assembler never runs on the runner at all.

⭐ **So the cause is not in this repository**, and every repair below is a real
defect that is not it. What is left is the runner image, the vendor's endpoint, or
the network between them - and the `ps` timeline is what separates those three.
⛔ A correlation with a commit window is not a correlation with a change, and
nothing had asked the tree which files the window actually moved.

#### ⛔ RUN 24 REFUTED THE BOUNDS AND NAMED THE OTHER HALF. 2026-09-16

⭐ **The first dispatch after the repair hung too, and that is the measurement
rather than a setback.** `capture-client` run 24 on `14f4acd`, one release lane:
*Install the client* started at 02:50:45 and was **still open twenty minutes
later**, having passed BOTH of its bounds - the privileged 780s at 03:03:45 and
the outer 900s at 03:05:45. ⛔ That takes this entry's tally of bounds measured
not to end this step from seven to **ten**.

⛔ **AND TEN BOUNDS FAILING THE SAME WAY IS NOT TEN PROBLEMS.** A `run:` step ends
when its command has exited **AND** its output pipe has reached end of file. Every
bound this entry has ever added acts on the first condition. Not one of them can
act on the second, so a held pipe defeats all ten identically - and adding an
eleventh would too.

⛔ **THIS WORKFLOW ALREADY STATED THE GUARANTEE AND HELD IT ONE LEVEL TOO LOW.**
Its own comment reads *"nothing the route spawns inherits this step's output"*,
and `install-step.sh` does redirect everything IT starts - then inherits the
step's pipe **itself**. So the promise covered the route's children and not the
shell holding the door open for them. ⚠ That is this project's recurring class
again, a guard whose scope is narrower than the thing it guards, this time inside
the sentence that claims otherwise.

⭐ **The repair is structural rather than another bound.** The step body hands
`install-step.sh` and its whole subtree a **file**, and cats it afterwards, so the
step's pipe is held by the step's own shell and by nothing else. The step then
ends when that shell does, whatever the route left running - which is the first
version of this guarantee that does not depend on a signal reaching anything.

⚠ **`report-holders.sh` has been aimed at the wrong descriptor the whole time.**
It is pointed at `step.log`, and `holders.log` duly reported `0 holder(s)` on
every green run. The file that keeps a step open is the step's own **stdout**,
which nothing has ever asked about.

#### ⛔ RUNS 24, 25 AND 26 ALL HUNG, AND RUN 26 IS THE SHARPEST MEASUREMENT YET

⚠ **The `ps` timeline was NOT obtained. Three dispatches, three cancellations,
zero artifacts**, read back from the API rather than assumed:

| run | commit | install step | ended by | artifacts |
| --- | --- | --- | --- | ---: |
| 24 | `14f4acd` | 02:50:45, open at 20 min | job cancel | **0** |
| 25 | `4b6c33d` | 03:24:08, open at 26 min | job cancel | **0** |
| 26 | `21993d8` | 04:13:57, open at 27 min | job cancel | **0** |

⛔ **RUN 26 IS THE ONE THAT CHANGES THE QUESTION.** It carried THREE independent
endings, and the third depends on no signal reaching anything: `install-step.sh`'s
own loop breaks at its deadline and exits without `wait`ing. All three passed
without the step ending - the step's own 780s at 04:26:57, the outer 900s at
04:29:27, and even the `:-1080` default at 04:32:27.

⛔ **A SHELL THAT DOES NOT REACH ITS OWN `[ -lt ]` IS NOT A SHELL THAT IS
WAITING.** Every previous reading treated this as a bound failing to fire. Run 26
says the step's process is not getting there.

⛔ **AND THE FIRST VERSION OF THIS PARAGRAPH WAS WRONG ABOUT WHY.** It said *a
loop that only sleeps and compares two integers cannot overrun its deadline while
executing*, which was written without re-reading the loop. ⚠ **It also runs
`ps -eo … args` every iteration**, and that was the last UNBOUNDED command in
this shell: `ps -e` reads `/proc` for every process on the host, and a read of
`/proc` for a task wedged in the kernel blocks in exactly the way this loop exists
to observe. ⭐ **So the instrument could hang on the condition it was written to
record**, and the deadline two lines above it would never be reached - which is
precisely what run 26 looked like from outside. The `ps` carries `timeout` now,
and a sample that is headed and empty is itself the measurement.

⭐ **THE NEXT INSTRUMENT IS NAMES, NOT BOUNDS, and this workflow already argues
it.** Its three probe steps exist because *"when nothing inside a job survives,
the one signal that does is WHICH STEP the API last reported in progress"*. Eleven
bounds have now been spent on a step that does not end; splitting *Install the
client* into named sub-steps - the fetch, the install of what was fetched, the
version call - localises the wedge to a handful of commands with no log, no
artifact and no bound required. ⛔ Do that before adding a twelfth bound.

⭐ **THOSE STEPS ARE BUILT, 2026-09-16.** Three more named probes walk the four
operations the release route actually performs, between *Resolve the release
artifact* and *Install the client*:

| step | what wedging there would mean |
| --- | --- |
| *Probe the release fetch* | the vendor's endpoint or the network, not this tree |
| *Probe staging the fetched artifact* | `chmod`, `mkdir`, or a `cp` across filesystems into `/usr/local/bin` |
| *Probe executing the fetched artifact* | `exec` of the downloaded binary itself |

⚠ **They guard nothing and each ends in `true`**, so a probe cannot fail a
capture. ⛔ **And none of them touches the install target**: the staging probe
writes `.probe-only.new` and removes it, a name `binary()` can never find.
⭐ **A job that walks all three and still wedges in *Install the client* has
exonerated the whole release route**, which is a different and equally useful
answer.

#### ⭐ RUN 27 GAVE EXACTLY THAT ANSWER, IN ONE SECOND. 2026-09-16

⛔ **The release route is exonerated and the wedge is in the plumbing around it.**
`capture-client` run 27 on `c98ad6d` walked all three probes and then wedged in
*Install the client*:

| step | started | finished |
| --- | --- | --- |
| *Probe the release fetch* | 05:52:48 | 05:52:49 (**1s**, whole artifact) |
| *Probe staging the fetched artifact* | 05:52:49 | 05:52:49 (**0s**) |
| *Probe executing the fetched artifact* | 05:52:49 | 05:52:49 (**0s**) |
| *Install the client* | 05:52:49 | ⛔ **never** |

⭐ **Every operation the route performs completes instantly**: the fetch of the
whole artifact from the vendor, `chmod`, `mkdir`, a `cp` across filesystems into
`/usr/local/bin`, and an `exec` of the downloaded binary. ⛔ So the vendor's
endpoint, the network, the filesystem and the binary are all ruled OUT by
measurement rather than by argument - and *Install the client*, which performs
those same operations, still does not return.

⚠ **What is left is what the install does and the probes do not**: the `sudo`
plumbing, `install-client`'s guard and its rule-12 scan over the workdir, the
holder report's walk of `/proc`, and the watchdog's own `ps`. ⭐ **The last of
those was unbounded until this change**, and it is the only one of them that runs
in the step's own shell - which makes it the first thing the next dispatch
should be able to exclude.

#### ⛔ RUN 28 EXCLUDED THE `ps`, AND REPRODUCED THE LOCALISATION

⚠ **`capture-client` run 28 on `ae76d03` carries the bounded `ps` and hung
anyway**: install began 06:05:12 and the step was still open **twenty-five
minutes** later, past the loop's own 780s deadline at 06:18:12 and past the outer
900s at 06:20:12. ⛔ So the watchdog's `ps` is **not** the wedge, which is what
bounding it was for - a candidate excluded by measurement rather than argued away.

⭐ **And run 28 REPRODUCED run 27's localisation.** Its three probes completed
inside the same second, 06:05:12, so two independent runs agree that the release
route's own operations are instantaneous while the step that performs them does
not return.

⛔ **THE CANDIDATE SET IS NOW SMALL AND NAMED.** What *Install the client* does
that the probes do not, minus the excluded `ps`:

| candidate | where it runs |
| --- | --- |
| the `setsid`/`sudo` plumbing that launches the install | `install-step.sh`, this shell |
| `assert-disposable --marker`, the claim guard | `install-client.sh`, as root |
| the rule-12 secret scan: `find` plus `grep -IlE` over the workdir | `install-client.sh`, as root |
| `report-holders.sh` walking `/proc/[0-9]*/fd/*` | after the loop, already bounded at 60s |

⭐ **THOSE FOUR ARE PROBES NOW, 2026-09-16, one named step each.** The same
technique that localised the route, applied to the remainder:

| step | what wedging there would mean |
| --- | --- |
| *Probe the claim guard* | `assert-disposable --marker` under `sudo` |
| *Probe a detached sudo launch* | the `setsid sudo -E timeout` shape, over a command that prints and exits |
| *Probe the rule-12 scan over a workdir* | `grep -IlE` with two quantified alternations across the fourteen-megabyte artifact |
| *Probe the holder walk of /proc* | reading every process's descriptor table |

⛔ **THE SCAN PROBE CARRIES `install-client`'s PATTERN VERBATIM**, compared
programmatically rather than by eye; if the two drift the probe stops being about
the thing it names, and the step says so in its own comment.

⛔ **DRIVING THEM ON THIS HOST FOUND A DEFECT THE READING DID NOT.** GitHub runs a
`run:` block as `bash -e`, and `grep` exits **1 when it matches nothing** - the
ORDINARY outcome for a scan that finds no secret. A bare `_rc=$?` on the next line
is never reached: `-e` ends the step first, so the `true` below it never runs and
**a probe would fail a capture**, which is exactly what this workflow's own
comment forbids. ⚠ Two of the four had it. The status is captured on the line that
produces it now, and all four were re-driven under `bash -e` to exit 0 together.

⚠ **A HANDWRITTEN `true` AT THE END OF A BLOCK IS NOT WHAT MAKES A STEP
HARMLESS** - reaching it is - and nothing in the gate checks that a probe can be
reached. Filed here rather than guarded.

#### ⛔ IT IS NOT THE RELEASE LANE. BOTH ROUTES WEDGE, AND NOBODY LOOKED

⛔ **This entry has said *the RELEASE lane* since run 21 and that is wrong.** The
source lane wedged in the SAME step, in the same runs, and its jobs were read for
the first time on 2026-09-16:

| run | source lane's last step | release lane's last step |
| --- | --- | --- |
| 21 | *Install the client*, in progress | *Install the client*, in progress |
| 22 | *Install the client*, in progress | *Install the client*, in progress |
| 23 | *Install the client*, in progress | *Install the client*, in progress |

⚠ **Both jobs in all three runs are `cancelled`**, which was recorded, and the
step each of them stopped at was not. ⛔ So every route-shaped reading this entry
carries - the fetch, the asset, the vendor's endpoint - was reasoning about a lane
that is not distinguishable from the other one. ⭐ Two routes that share nothing
but `install-step.sh` and `install-client.sh` wedge identically, which says the
subject is the step's own machinery rather than anything a route does.

#### ⛔ RUN 29: EVERY COMPONENT PASSES AND THE COMPOSITION STILL WEDGES

⭐ **All seven probes completed in FOUR SECONDS**, on `2608ce0`, and the job then
wedged in *Install the client* exactly as before:

| probe | 07:47:36 to 07:47:40 |
| --- | ---: |
| the release fetch, whole artifact | 1s |
| staging: `chmod`, `mkdir`, `cp` into `/usr/local/bin` | 0s |
| executing the fetched artifact | 0s |
| the claim guard | 0s |
| a detached `sudo -E timeout` launch | 2s |
| the rule-12 scan over the fourteen-megabyte artifact | 0s |
| the `/proc` holder walk | 1s |

⛔ **So every operation that step performs has now been measured individually,
they sum to four seconds, and the step that composes them runs for thirty
minutes.** Each candidate is excluded on its own; what is left is the
composition, and no further probe of a component can reach it.

⭐ **WHICH MAKES THE ARTIFACT, NOT THE LOCALISATION, THE THING TO BUY NEXT** - and
there is one mechanism never tried on this step. A **step-level**
`timeout-minutes` is the RUNNER's bound rather than a twelfth shell bound, and it
differs in the one way that matters: exceeding it marks the step **FAILED** rather
than cancelling the job, and a failed step still runs the `if: always()` steps
after it. ⚠ It was measured once, on run 6, not to fire - on *Upload the install
logs*, a `uses:` step whose work happens inside an action. This is a `run:` step,
which the runner supervises directly. ⛔ And because the install FAILS rather than
succeeding, *Cut the route* is skipped, so the host still has the network the
upload needs.

#### ⛔ RUN 30: THE RUNNER'S OWN STEP SUPERVISOR CANNOT END IT EITHER

⚠ **`timeout-minutes: 17` on the install `run:` step did not fire.** Run 30 on
`bba735c` began the step at 08:43:52, so the runner's bound was due at
**09:00:52**; the step was still in progress **thirteen minutes past it**. That is
the **thirteenth** bound measured not to end this step, and the first that is not
a shell bound at all.

⛔ **THE DIFFERENCE IS WHAT MAKES IT DECISIVE.** Every bound before it was issued
by a process inside the step, and could be argued away as a signal that did not
reach its target. This one is GitHub's own supervisor, which owns the step's
process tree and is the thing that cancels jobs. It could not end the step either.

⭐ **So the step cannot be made to end, and the artifact cannot be collected
through any path that requires it to.**

#### ⛔ AND THE HANG IS INTERMITTENT ON IDENTICAL BYTES, WHICH REFUTES TWO READINGS

⛔ **Read the digests before believing either of the readings above.** The install
record a green run writes carries `adapter_sha256`, and the green run 20 attempt 2
recorded `a90c0441...`:

| commit | adapter sha256 | install-client | install-step | runs |
| --- | --- | --- | --- | --- |
| `c3c9f6c` | `a90c0441` | `db3044b9` | `1e3cf608` | 19 ⭐ green |
| `95e90f5` | `a90c0441` | `db3044b9` | `1e3cf608` | 20 a1 and a2 ⭐ green |
| `40ed628` | `a90c0441` | `db3044b9` | `1e3cf608` | 21, 22 ⛔ hung |

⛔ **Runs 21 and 22 ran the SAME INSTALL BYTES as the green run 20 attempt 2**,
which finished at **15:59:43** - between run 21 ending at 15:15 and run 23
starting at 16:07. So a green run and two hung runs, on identical code, inside one
hour.

⛔ **TWO THINGS THIS ENTRY RECORDED TODAY ARE THEREFORE WRONG.** *The cause is not
in this repository* was read as a defect in the host, and a host-wide defect does
not go green at 15:50 between two hangs. *The commit is the variable* is refuted
by the digest table above. ⭐ What is actually established is narrower and
stronger: **the same bytes hang sometimes and not others**, three green then nine
consecutive hangs, with no code difference between the two groups.

⚠ **AND THE `aria2` LEAD DOES NOT FIT THESE RUNS**, checked on 2026-09-16 because
the operator remembered the original hang being resolved by replacing `aria2`
with `aria2-next` - which is true and is `CLIENT-14`. Every run from 19 to 31
dispatched `aria2-next`, by job name; the adapter references bare `aria2` only as
JSON-RPC method names; `aria2c-next` appears nowhere in this repository's history;
and the vendor binary prints its version in 0s under **any** `argv[0]`, measured
over `aria2c-next`, `aria2-next`, `aria2c` and an unrelated name. ⛔ The real
`install-client`, driven here against the real adapter and the real release URL,
exits 0 in **one second**.

#### ⛔ The door sweep found the same class behind THREE more doors

⚠ **`install-client` was not the only caller, and the enumeration written from
memory did not contain them.** Every place this tree hands an adapter to a shell,
swept on 2026-09-16:

| door | bounded | output |
| --- | --- | --- |
| `capture-client.sh` `describe` | ✅ | ✅ a file |
| `capture-client.sh` `version` | ✅ | ⛔ **a substitution** |
| `resolve-release.sh` `describe` | ⛔ **none at all** | ✅ a file |
| `resolve-source.sh` `describe` | ⛔ **none at all** | ✅ a file |

⭐ **All three are repaired in this change**, and the first row is why they are
defects rather than unknowns: `capture-client.sh` asks the SAME question the
right way twenty lines above asking it the wrong way, which is the one-gated-door
shape [`../docs/methodology/reviews.md`](../docs/methodology/reviews.md) calls the
most recurring hole there is.

⚠ **Two of them are outside *Install the client* entirely.** The resolvers run in
*Resolve the release artifact* and *Resolve the source tag*, before the host's
route is cut; `capture-client.sh` runs in *Capture*, after it. So a hang in any
of the three would have been read as a different defect from the one this entry
has been chasing.

⛔ **AND `check-adapters` DOES NOT COVER THIS.** Its scope is a network fetch
*inside* an adapter, stated in its own header; a *call site* that invokes an
adapter is a different population, and all four defects above live there. That is
a rule narrower than the class it is about - which is the thing this entry keeps
finding, now including in its own newest rule.

#### ⚠ Residual, measured 2026-09-16: the bound reaches one process group

⛔ **An orphan survives the privileged bound and it is named rather than assumed
away.** Nested `timeout`s each create their **own** process group: the outer one
around `install-client` signals its group, and the inner one around the adapter
is a different group that never receives it. The driven pass above left
`sleep 600` running as root with PPID 1, and `holders.log` named it in one line:
`pid=3160 comm=timeout args=timeout -k 5 600 sh …/hang.sh install release …`.

⚠ **It does not block the deliverable**: the step ends, the evidence ships, and
the runner is destroyed after the job. ⛔ **It is still host state a capture must
not inherit**, and the fix changes the signal semantics of a bound that is
mutation-proven at its current values - which is its own unit, for the reason the
`mine-repo` revert above gives. Filed rather than rushed.

#### ⛔ RUN 32 REFUTES THE PRIVILEGE READING, 2026-09-17

⚠ **`ACQ-06` removed `sudo` and `/usr/local` from the install path and the step
wedges exactly as before.** `capture-client` run 32 on `a709e2d`: both lanes'
*Install the client* never returned, both jobs cancelled at **forty minutes**,
**zero artifacts**. ⛔ So *privilege is the variable* joins the four readings this
entry has already refuted, and it joins them the same way - by a dispatch rather
than by an argument.

⭐ **The same run narrows it from the other side.** *Resolve the release
artifact* used `install-rootless.sh`'s bounded fetch to retrieve a document over
the network in **one second**, on that host, minutes before the install wedged;
the unprivileged claim took **0s**. So the installer, the fetch, the claim guard
and the resolver are all exonerated on that machine by its own step timings.

⛔ **AND NOTHING FURTHER IS DONE ABOUT IT.** `TODO/RULES.md` owns that rule and
`AGENTS.md` makes it absolute 16. ⚠ This paragraph is the record and not the
start of a fifteenth attempt.

#### ⭐ THE ROOT CAUSE, MEASURED 2026-10-11: THE ROUTE CUT, NOT THE INSTALL

⛔ **Nothing in *Install the client* ever hung.** The operator directed the hang
be fixed at its root, and the instrument that found it was the one this entry
had never built: evidence that leaves the host while the job runs.
`scripts/ci/flight-recorder.sh` posts blocked tasks with their kernel stacks,
memory, disk, the kernel ring buffer and the newest log tails to a check run on
the commit, from the job's first step onward.

| run, on the diagnostic branch | what left the host |
| --- | --- |
| `1ec75f7`, 15-second posts | a healthy host at 03:30:50: 14 GB available, load 1.7, no blocked task, the API answering in 0.1 s. The install began at 03:30:54 and no post arrived after it |
| `8a29cfb`, the install traced, 2-second posts | `install-client` exited 0 at 03:34:23, `aria2-next 2.8.6 through the release route (acquired=yes)`; posts stopped about three seconds later |
| `16a146b`, a marker around each operation of `install-step.sh` | the watchdog `ps`, the loop's end, the `cat`, the descriptor read and `report-holders` all returned, the last at 03:40:22; posts stopped as the step ended and the next one, *Cut the route off this host*, ran |
| `a4b6053`, the capture in a network namespace and no route cut | ⭐ the job completed in two minutes, every step reported, the job log exists and the install artifact uploaded |
| `ca77cbb`, `main`'s workflow as committed in `3f3d773`, both routes | ⭐ both lanes completed: release in 1.5 minutes, source - which builds `aria2-next` - in 12.5; both logs and both install artifacts exist |

⭐ **So the mechanism is this.** *Cut the route* deletes the host's default
routes, and the runner agent reaches GitHub over those routes. From that moment
the job cannot report, GitHub keeps showing the last step it heard about - which
was *Install the client*, because a step's completion is sent a moment after it
ends - and on the images of 2026-09 onward the host never regained contact after
*Restore the route*. A job that never reports again leaves no log blob, which is
why the log endpoint answered `BlobNotFound`. ⛔ Every bound was aimed at a step
that had already finished, so none could fire on anything.

⚠ **It also explains every reading this entry refuted.** It is not the package,
not privilege, not the image (run 34 on `ubuntu-22.04` behaved the same), not
the vendor's endpoint, and it moved between steps on runs 6 and 7 because what
GitHub displays is the last step reported before the cut, not the step running.
⚠ What it does NOT yet explain is why runs 11 to 20 came back after their
restore; the restore was never instrumented, and with the namespace there is
nothing left to restore.

⭐ **The fix is structural.** `scripts/capture/contained.sh` runs the egress
guard and the capture in a network namespace that holds loopback alone, as the
calling user; the host's routes are never touched. Driven on the session host
as uid 1001: inside, `lo` only, an empty route table, the guard passing, an
outbound request failing to resolve, and the command's exit code carried out;
outside, the default route intact.

⚠ **What both contained runs then refused is a measurement, not a hang.** The
observer recorded the announces and `cpython-stdlib` declined to read them:
*two segments disagree about tracker_http/peer_id*, so `aria2-next` sent a
different peer ID on two announces of one run. That is `OBS-04`'s per-connection
identity reaching the connector, and `CI-09`'s to settle.

⚠ **`capture.yml` still cuts the route**, on both platforms. It installs
nothing and its last runs were green; it is the same fragile shape and is the
next thing this entry should move. Windows has no network namespace, so its job
needs a different containment, such as a firewall rule scoped to the capture's
processes.

#### ⛔ The recorder never closed its check run, so its silence meant nothing. 2026-10-14

⛔ **Every check run the recorder opened is still `in_progress`.** Read back on
2026-10-14 through rule 8's route: the three `flight-recorder (...)` runs of
dispatches 38109299467 and 38111442119 had no conclusion three days after their
jobs ended. The loop ran to a 3000-second limit no job reaches, so the runner
killed it with the job and the final `completed` was never sent. ⚠ That is the
signal this instrument is built on - *posts that stop mean the host lost
contact* - produced by jobs that ended cleanly.

⭐ **The job's last step stops it.** *Stop the flight recorder* runs on every
outcome and touches `$RUNNER_TEMP/flight-recorder.stop`; the recorder, which now
waits a second at a time, posts a final snapshot, completes the run, reads the
API's answer back and writes `flight-recorder.done`, which the step waits on for
at most 60 seconds and prints. ⚠ It never fails the job, because the recorder is
a diagnostic. A run still open after its job is now a host that never reached
the step.

| measured on this host, 2026-10-14, by `check-step-bodies` against a stub `curl` | result |
| --- | --- |
| the workflow's own *Start* body | ends, output closed, while the recorder posts on |
| nothing stopping it, three intervals later | the run is still open - the defect, shown |
| the workflow's own *Stop* body | ends 0; the last request is the completion; the done file says `completed`; the recorder is gone |
| a completion the API answers as still `in_progress` | `not confirmed`, never `completed` |
| a stop with no recorder answering | ends 0 after its bound and says the run was not confirmed |

⛔ **The harness found a defect in itself first.** Its "the recorder is gone"
wait ran `pgrep -f` inside a `sh -c` whose own arguments carried the pattern, so
it matched itself and reported an exited recorder as running. `[f]light` matches
the recorder and cannot match its own spelling.

`check-workflow` refuses a job that starts the recorder and whose last step is
not *Stop the flight recorder* on `always()`, touching the file the recorder
watches; refuted by deleting the step and by narrowing it to `success()`.

Guard mutation, five plants into `flight-recorder.sh` in a scratch copy of the
tree, each replaced exactly once and verified to have changed the file, run
through `check-step-bodies`, clean control either side:

| plant | verdict |
| --- | --- |
| the stop file is never looked for at the top of the loop | ⭐ refused, 2 cases |
| the API's answer to the completion is not read | ⭐ refused, by the refused-completion case |
| the completion is never sent | ⭐ refused, 2 cases |
| no done file is written | ⭐ refused, by the stop case |
| the wait between snapshots ignores the stop file | ⚠ **survived** |

⚠ **The survivor is equivalent at the harness's interval and is kept.** The
harness posts every second, so a wait that sleeps the whole interval and one
that looks every second see the stop at the same moment. What the second buys is
latency: a stop seen within a second rather than within an interval, which only
matters against the step's 60-second bound once the interval approaches it. A
case timing it would be the load-sensitive row this entry already carries one of,
so it is recorded rather than turned into one.

#### ⛔ A closed run showed no snapshot, and a failed install printed nothing. 2026-10-14

⭐ **The stop works on a runner.** `capture-client` run 35, on `a39d933`, closed
both of its recorder runs `completed`/`neutral`, *the job reached its last
step*, each one second after its job's last step began.

⛔ **And both closed runs carried a title over an empty text.** The last
snapshot went up first and the completion after it, as a second request whose
`output` had a title and no `text`; a check run's output is replaced by the
update that carries one. ⚠ The source lane's last state was the compile it
failed in, and it was gone from the one place this entry says a dispatch is
read. ⭐ The recorder now closes the run in the request that posts the final
snapshot. ⚠ That couples the two, so a text the API refuses would refuse the
completion with it; a refusal that came back is asked once more with no
`output`, which keeps the last snapshot the run accepted, and a request that got
no answer is not sent again, because a second would spend the stop step's wait.

⛔ **The job log had nothing either.** *Install the client* carried its exit
code as `rc=$?` on the line after the install, and the runner runs a body as
`bash -e`: the failing install ended the shell on its own line, so the `cat`
that prints `install-step.log` never ran, on any failure, since that body was
written. ⚠ `check-step-bodies` runs this body under `bash -e` and asked its
exit code and whether its output closed, which are the same either way. ⭐ It is
`rc=0` and `|| rc=$?` now,
and the install artifact, which carried the log throughout, is what said why.

| measured on this host, 2026-10-14, by `check-step-bodies` | result |
| --- | --- |
| the request that closes the check run | one request, `neutral`, titled with the ending, carrying the last snapshot through to its `route to the API` section |
| a closing snapshot the stub refuses as the API would | the run still closes, by a second request with no `output` |
| a closing request the stub never answers | `not confirmed`, and no second request |
| *Install the client* over a stub that refuses | before the fix: exit 1 and none of the stub's refusal in the output, run 35 reproduced; after it: exit 1, the refusal printed, the output closed |
| the harness | 35 cases, 35 passed |

⚠ **The stub now answers a completion the way the API does**: one with no
conclusion it knows is a 422 with a message and no status. It accepted that
before, which the review of this change showed by planting it.

Guard mutation, each plant replaced exactly once in a scratch copy of the tree
and verified to have changed the file, run through `check-step-bodies`, clean
control either side:

| plant | verdict |
| --- | --- |
| `flight-recorder.sh`: the completion as a second request with a title and no text, the shape run 35 measured | ⭐ refused, 2 cases: the snapshot case and the refused-text case |
| `flight-recorder.sh`: the text emptied in the closing request only | ⭐ refused, by the snapshot case |
| `flight-recorder.sh`: the closing request reuses the loop's last snapshot | ⭐ refused, by the snapshot case, on its title |
| `flight-recorder.sh`: the closing request carries no conclusion | ⭐ refused, by the snapshot case |
| `flight-recorder.sh`: a refused closing snapshot is not asked again | ⭐ refused, by the refused-text case |
| `flight-recorder.sh`: a closing request that got no answer is asked again | ⭐ refused, by the no-answer case; it survived until that case was written |
| the workflow: `rc=$?` back on the line after the install | ⭐ refused, by the refusal case |
| the workflow: the log printed and the step ended 0 | ⭐ refused, 2 cases: the refusal case and the step's own bound |

⛔ **The door sweep of the same change found the shape once more**, in
`ci.yml`'s *Shell syntax and style*: `shellcheck` and `shfmt` on two bare lines,
so on the runner a shellcheck finding ended the block before shfmt's diff
printed, and under `check-workflow`'s `sh -c` a finding only shellcheck makes was
answered by shfmt's exit 0. Driven over a tree with an SC2086 line appended,
which shfmt leaves alone: the old body exited 0 under `sh` and 123 under
`bash -e`; the fixed body exits 123 under both and 0 on the clean tree.
`check-workflow` plants that line now, beside its parse-error plant, which
passed either way.

Residuals from that sweep:

- ⚠ **The bound chain is stated in comments and checked by nothing.**
  `check-workflow` asks only that `timeout-minutes` exists. Every source-lane
  value moved by 600 seconds together on 2026-10-14, and one moving alone would
  have passed. Acceptance: a `check-workflow` case that reads each lane's
  `BIT_IDS_INSTALL_TIMEOUT`, `BIT_IDS_STEP_TIMEOUT`, `OUTER_TIMEOUT`, the step's
  `timeout-minutes` and the job's, and refuses a chain that does not nest,
  refuted by moving one value alone.
- ⚠ **The stop step waits 60 seconds for a final snapshot that grows with every
  blocked task**, and asks `[ -e ]` of a done file the recorder writes in two
  steps. Past 60 seconds the job ends with its run open, which reads as lost
  contact. Acceptance: the done file is written by rename, and a
  `check-step-bodies` case whose final snapshot outlasts the wait says so.
- ⚠ **The `rc=$?` shape has now shipped twice and nothing refuses it by
  reading.** `2608ce0` fixed it in two probe steps on 2026-09-16 and wrote the
  rule into `docs/conventions/shell.md`, four and a half hours after `4b6c33d`
  wrote this body with it. The cases here guard two bodies, not the shape. Acceptance:
  `check-project` refuses a line `NAME=$?` standing alone in a workflow `run:`
  body that is not `pwsh` and has no `set +e` before it, refuted by a plant of
  the old *Install the client* body; `check-bitcheck --harnesses` holds.
- ⚠ **The job's `timeout-minutes` is one number for both lanes**, so the release
  lane's rose to 45 minutes with the source lane's. It is the bound that should
  never fire, and a release lane ends in minutes.

#### ⛔ AND THE PROBES ARE DELETED, 2026-09-17, BECAUSE `ACQ-06` LANDED

⭐ **Ten diagnostic steps stood in `capture-client.yml` and they are gone.** Their
own comment said they come out when the question is answered, and run 29 spent
the last of what they could buy: all seven passed in FOUR seconds while the step
composing them ran for thirty minutes, so every component was excluded on its own
and no further probe of a component could reach the composition.

⛔ **AND FOUR OF THEM MEASURED A PATH THIS TREE NO LONGER TAKES.** *Probe sudo*,
*Probe staging the fetched artifact* - a `sudo -n cp` into `/usr/local/bin` -
*Probe the claim guard* under `sudo`, and *Probe a detached sudo launch* all
describe operations `ACQ-06` removed. ⚠ The claim probe would have been worse
than stale: the marker is under the capturing user's own state directory now, so
a privileged `--marker` prints ROOT's path over a host claimed by the runner,
which is a diagnostic answering confidently and wrongly.

⚠ **`TODO/RULES.md` is why they go rather than stay.** A hang is a dead end, not
a subject; keeping the instruments of the dead end in the file is keeping the
subject open. ⛔ What is NOT claimed is that the hang is fixed: run 32
dispatched the rootless install and it wedged, which the section above records
and `TODO/acquisition.md` carries under `ACQ-06`'s residuals.

### ⚠ Residual, filed 2026-09-09: `check-step-bodies` is a load-sensitive row

⛔ **A gate row that fails under load and passes alone is the same class this
entry is about** - a behaviour inherited from the host rather than stated - and
it is now measured **twice** on one session host. Both times the gate reported
`FAIL check-step-bodies (exit 1)`; both times
`sh scripts/ci/check-step-bodies.sh` run alone immediately afterwards reported
`24 cases: 24 passed, 0 failed`, exit 0; and both times the next gate over the
same tree was green.

⭐ **The mechanism is in the harness's own subject.** Its cases time things: a
bound that must fire as `124`, a case named "a product slower than one tick and
inside the bound survives", and whether a step's output pipe has reached end of
file. A loaded host moves a case across its own boundary, which is a timing
assumption inherited from whatever else is running.

⚠ **What is NOT established is that this explains every red seen today.**
`check-workflow` failed one `gate_control` case on each of two runs, a different
case each time, and those cases report only "the clean tree failed a check"
without naming which - and the harness deletes its workdir, so the naming log is
gone. ⛔ It is consistent with this row flaking and it is not proof of it.

⛔ **It can turn the CI Linux lane red over a correct tree**, because that lane
runs the gate with `--strict`. Not observed there yet: CI runs 93 and 94 were
green on all three jobs.

#### ⭐ DIAGNOSED AND FIXED on 2026-09-09, and the first two readings were wrong

⛔ **The obvious reading was that `CLOSE_SECONDS=3` is too tight, and it was
wrong.** Two reproductions were attempted against it and both came back green:
twelve CPU spinners on four cores, then the harness beside `check-capture`,
`check-capture-client`, `check-store`, `check-catalogue` and `check-examples` -
which is the faithful shape, because the gate backgrounds every check and runs
them concurrently. **24 of 24 passed** in each.

⭐ **What found it was capturing the failing run's own output.** The gate prints
the failing check's log; a loop of gate runs reproduced the red on the first
attempt, and it names one case:

```text
❌ sh  one leaked descriptor onto the step's output brings it back:
       the output closed, so the planted hang did not happen
```

⛔ **So the bound was not too tight - the PLANT EXPIRED.** That case asserts the
pipe is held open, and it reported the pipe closing. The leaking process is
spawned *inside* the body, and the pipe is only examined after the body exits
plus `CLOSE_SECONDS`, so the plant only proves what it claims while

```text
leak duration > body duration + CLOSE_SECONDS
```

⚠ **`sleep 8` was ample for the two `probe` cases and marginal for this one.**
Their bodies are `echo` and `exit 0`; this one's body is the whole *Install the
client* step - a `timeout` around `install-step.sh`, which runs `install-client`
under `sudo`, a watchdog loop and a bounded holder report. Under a full gate the
body reached roughly five seconds and ate the eight.

⭐ **Fixed:** the duration is now `LEAK_SECONDS=45`, named once and used at every
plant instead of a literal, with the relation checked rather than commented -
`LEAK_SECONDS` must be a positive integer and must exceed `CLOSE_SECONDS`, both
mutation-proven (exit 2, each naming its own reason). ⚠ The cost is stated in the
file: a `sleep` orphan can outlive the check by up to 45 seconds where the old
value bounded that at 8.

**Verified twice over:** 5 consecutive full gate runs green, against 3 red in
roughly 9 before the change - and `sh scripts/ci/check-workflow.sh` came back
**97 cases, 97 passed, 0 failed** on the final tree. ⭐ That last one matters
because it is the harness that was failing: it runs the whole gate about ten
times, and both of its runs before the fix lost exactly one `gate_control` case,
a different one each time. After the fix it lost none. ⚠ That is a rate on one host and not a proof; the failure is
load-dependent, and `LEAK_SECONDS=4` still passes on an idle host. ⛔ Which is
also why the guard compares the two bounds: the relation is what matters, and a
number that happens to work today is what went stale.

### ⭐ `capture.yml`'s Linux job contains its capture in a namespace. 2026-10-14

The Linux job deleted the runner's default routes, captured, and restored
them. `capture-client` measured on 2026-10-11 that the same cut severs the
runner agent. The job now runs *Assert containment* and *Capture* through
`contained.sh`, and *Cut the route off this host* and *Restore the route* are
gone from it. `check-workflow` already held a job to one shape or the other,
so it now holds this job to the namespace shape.

Driven on the session host first: the egress guard inside the namespace read
*no route off this host*, and `capture-run.sh` wrote 2 segments and 2
artifacts, verified by `sha256sum`. ⚠ This host has no `ip`, so the drive gave
`contained.sh` a one-command shim that brings loopback up by ioctl; a runner
has `iproute2`.

⚠ **Two checks moved with it.** `check-step-bodies`' three `sh` cases over the
Linux *Restore the route* body retired with the step, 36 of 36 now; the Windows
ones stay. `check-workflow`'s probe that moves a build after the cut now reads
the Windows job, which still cuts.

⛔ **Still open:** the Windows job's containment, which needs something other
than a namespace, such as a firewall rule scoped to the capture's processes.

⭐ **The orphaned-subprocess residual is closed.** Nested `timeout`s each start
their own process group, so a bound reaches only the group it started. That was
the `sleep 600` of 2026-09-16, left with PPID 1. Every group stays in the
session `install-step.sh` opens with `setsid`, so after its wait the step lists
that session, writes it to `orphans.log`, sends TERM, then KILL to a fresh
listing. `check-step-bodies` plants a product that leaves `timeout 600 sleep
7123` behind: the step ends, nothing runs it afterwards, and `orphans.log` names
it, 38 of 38. Two plants were each refused: the sweep turned off, and the sweep
listing the wrong session. ⚠ The leaked-descriptor control now leaks from a
session of its own. A process that left the session is the one the sweep cannot
find, and the step's own redirection is still what keeps it off the pipe.

⭐ **Proved on a runner: `capture` run 38612214714 on `038e12c`, both jobs
`success`.** The Linux job's guard and fixture capture ran in the namespace
and never cut the host's route.

⭐ **`check-workflow` runs a step body as GitHub does now**, through one
`step_shell`: `bash --noprofile --norc -e -c`. Under `sh -c` a body whose first
command failed and whose last succeeded was red on the lane and green in the
harness. A new probe unit, `step-shell`, hands the shell exactly that body and
one where every line succeeds. Measured: `sh -c` answers 0 on the first, which
is the defect, and `bash -e` answers 1. `check-shards` 14 of 14 with the new
unit. A dispatch of `capture.yml` on this
commit is what proves the Linux half on a runner.

## CI-09: The capture-to-publisher path, end to end

Source: two workflows that have each never run, joined by an artifact
Priority: P1 | Effort: L | Status: OPEN

Problem: The capture workflow uploads with `actions/upload-artifact` v7 and the
publisher downloads with `actions/download-artifact` v8. Neither has run, so the
pairing is unexercised, and the publisher's first step is a download of a bundle
no run has ever produced.

Approach: Take a dispatched capture's artifact through the publisher's dry run,
which is its default, and read what the download step actually handed it.
⛔ Nothing may be published until a measured record exists, so the dry run is
the whole of this entry and the push stays refused.

### ⭐ What run 14 supplies, stated exactly. 2026-09-09

⛔ **The route blocker on this entry is gone and the connector blocker is not.**
`capture-client` run 14 acquired one target through two routes on two hosts and
captured on BOTH, which is the first time this project has had a capture per
route.

⭐ **That is precisely the input `equivalence::classify_across` needs.** Read the
four outcomes rather than assume them: `ByteIdentical` is every route installing
the same bytes; `BuildEquivalent` is installs that differ in bytes where every
one was **observed** and no overlapping field disagrees; `Divergent` is equal
versions over conflicting evidence; `Unresolved` is not enough evidence, and the
module says in as many words that it is what a **single** capture of two
byte-different installs produces. ⛔ Run 14's installs differ in bytes and both
were captured, so this pair is the first that can reach `BuildEquivalent` -
which `publishable()` accepts - instead of `Unresolved`, which it refuses.

⚠ **What it does NOT supply is a `Profile`, and `classify_across` takes two of
them.** Run 14 produced two install records, two attestations and two evidence
bundles. Assembling those into two records is this entry's work and it needs a
store to write into.

⛔ **AND A PUBLISHED RECORD NEEDS MORE THAN THIS ENTRY CAN GIVE IT.** `E-PUB-02`
keeps any measured field provisional while only one connector could see it, and
both lanes of run 14 used one - this project's own Rust observer. So the honest
target for this entry is a record that is **written, stored and provisional**;
`OBS-07` owns the second connector that would make it publishable. ⚠ Two gates,
and running them together is what made the old handoff say no record could be
written at all.

Prove: the publisher's dry run completes against a real capture artifact, its
`sha256sum -c` step passes on the downloaded bundle, and
`sh scripts/ci/check-workflow.sh` still asserts the publisher's dispatch-only
trigger and its `dry_run` default.

### ⛔ SOMETHING TRIED TO ASSEMBLE RUN 14 AND IT CANNOT BECOME A RECORD. 2026-09-09

⭐ **[`assemble-capture`](../crates/bit-ids-probe/examples/assemble-capture.rs)
is what this entry was missing**: it reads a lane's capture artifact and its
install artifact and writes the `Profile` they support, deriving every field
from a document rather than from a lane's name.
[`check-assemble.sh`](../scripts/capture/check-assemble.sh) proves it over
synthetic lanes - 15 cases, and the control is the half that matters, because
every refusal below passes equally over an assembler that refuses everything.

⛔ **Driven over run 14's four real artifacts, it refuses.** The section above
said run 14 "is precisely the input `equivalence::classify_across` needs". It is
not, and the reasons were invisible to every reading because none of them is in
an attestation:

| what the artifacts say | what refuses it |
| --- | --- |
| both lanes' `resolution.txt` differed **only in their timestamps** - one `source_url`, one `listing_sha256`, one `asset_url` | ⛔ `E-ACQ-07`: two routes sharing a resolver are one route - ⭐ **repaired**: the source route resolves its own tag from the repository's refs, `ACQ-02` |
| every attestation declares **one** connector | ⛔ `E-CAP-01`, at the **validity** gate |
| nothing records how the artifact was **packaged**, and `package` is in `StoreKey` | ⛔ a record filed under a guessed package is the non-injective path `store.rs` refuses |
| no document the capture path writes carries the source route's commit | ⛔ `E-ACQ-06`: a source identity needs a full object name - ⭐ **repaired below** |

⚠ **`classify_across` is NOT in that table, because it never ran on run 14**: it
takes two `Profile`s and neither exists. What is measured is that the pair's
**shape** reaches `Divergent` - `check-assemble` builds two records differing
only in their peer-ID tails and reads that verdict off them.

⭐ **The tool reports all of them in one run** rather than the first it trips on.
⚠ It did not at first: adding the `package` derivation moved that refusal in
front of the commit one and two gaps a previous run had named vanished from the
report. Each derivation is probed separately now, because this file's own header
promises a reader every field the capture path would have to record.

⚠ **The first is a design decision rather than an oversight, and
`capture-client.yml` argues for it in a comment**: *"THE SOURCE LANE RESOLVES
THROUGH THE SAME STEP, because both routes must land on ONE stable version... a
second resolution per lane would let a vendor whose newest release moves between
two reads hand the two routes different versions"*. ⛔ **Absolute 4 already
answers that worry**: version equality is *checked after installation*, not
trusted. Two independent resolutions that landed on two versions would be caught
by the comparison, and that is the correct outcome - it means the vendor moved
mid-capture and the pair is not a pair. Forcing one resolution manufactures the
agreement instead of measuring it, which is what `E-ACQ-07` exists to refuse.
⚠ `aria2-next.sh`'s own comment claims the two routes "differ in resolver - the
releases API against git refs". The artifacts refute it: the source lane is
*handed* `BIT_IDS_RELEASE_TAG` and resolves nothing.

### ⛔ AND THIS ENTRY'S OWN TABLE ABOUT CONNECTORS WAS WRONG

The table above records that a one-connector capture *validates* and is refused
only by `E-PUB-02` at publication. ⚠ **That is true of one shape and false of
the one this project produces**, measured by stripping the golden fixture two
ways and reading each exit code unpiped:

| what "one connector" means | `validate-profile` |
| --- | --- |
| two declared in `capture.connectors`, one **observing** each field | ⭐ exit 0, `valid`, then `provisional, not publishable` with six `E-PUB-02` rows |
| one **declared** in `capture.connectors` | ⛔ exit 1, `refused`, `invalid document`, `E-CAP-01` |

⛔ **Every capture this project has run is the second row.** So `OBS-07` is not a
publishability nicety this entry can note and move past: a second connector is a
**validity** requirement, exactly as `E-ACQ-01`'s second route is, and the
"written, stored and provisional" target this entry set itself is unreachable
without it.

### ⛔ `BuildEquivalent` is unreachable for a real client through this path

`classify_across` compares every field both records measured and calls any
disagreement `Divergent`. A peer ID carries a per-connection random tail, and a
single capture can only ever state such a field as `constant` with one sample -
`patterned` and `variable` both need two. ⛔ **So two records of two lanes
necessarily disagree on `peer_wire/peer_id` and land on `Divergent`.** Proved
both ways in `check-assemble`: a pair whose observations agree reaches
`build_equivalent`, and a pair differing only in its peer-ID tails is
`divergent`. ⭐ `SCHEMA-04`'s sampling model is where several captures become a
`patterned` field; it sits above the record and nothing has run it.

### ⭐ A RECORD EXISTS. `capture-client` run 17, 2026-09-15

⭐ **`assemble-capture` wrote two `Profile`s and eight raw evidence files out of
run 17's own artifacts, exit 0**, with nothing recovered from a log:

```sh
./target/debug/examples/assemble-capture \
  --lane "$CAP_RELEASE,$INST_RELEASE" \
  --lane "$CAP_SOURCE,$INST_SOURCE" \
  "$STORE"
```

where each `CAP_*` is an unpacked `capture-client-aria2-next-<route>-...`
artifact and each `INST_*` the matching `install-aria2-next-<route>-...` one.

What a record carries, read back out of the written JSON:

| part | run 17's `cap-source.json` |
| --- | --- |
| routes | two: `route-release` (`github_release`), `route-source` (`source_build`) |
| resolvers | two, a releases listing and a clone URL - `E-ACQ-07` satisfied by two indexes rather than by a lane name |
| observations | four, each `constant` with `samples: 1`, each naming its evidence |
| connectors | two, `bit-ids-probe` and `cpython-stdlib`, reporting the same bytes |
| evidence | eight entries with sizes and `sha256`, every one resolving into `raw/` |

⛔ **BOTH RECORDS WERE `provisional` AND NOT PUBLISHABLE**, for two reasons,
both correct at the time and neither a defect in the capture:

- `E-PUB-04` at acquisition: the two routes installed different digests and the
  capture put only ONE of them on the wire, so equal version labels are all that
  connects them. ⭐ **Closed 2026-09-15**, in the section below.
- `classify_across: divergent`: a peer ID's tail is per connection, so two
  captures necessarily disagree - which is the section above, now measured on
  real bytes rather than argued. ⚠ It is a report rather than a publication
  gate: `publishable` never asked it. `SCHEMA-04` carries the join that closes
  it and the samples it still needs.

⛔ **The store is scratch state rather than the tree, and nothing is published.**

### ⛔ Run 18: the refusal MOVED, which is what says the repair landed

**Dispatched 2026-09-15 on `66df90b`, both lanes green**, same inputs as run 17
and one difference: the observer dials the peer surface twice. Each lane's
transcript carries **connections 2 and 3**, so the second dial works on a runner;
`aria2-next` answered the handshake on one of them and sent nothing on the other,
and announced once. Every field therefore still rests on one sample.

⭐ **And the pair is refused by `E-PUB-03` rather than `E-PUB-04`.** That is the
whole measurement: the store now holds a comparable capture of the other route
and the two DISAGREE, where every assembly before this was refused for holding no
comparable capture at all. The refusal names `cap-release`, `cap-source` and the
two fields - `peer_wire/peer_id` and `tracker_http/peer_id`, `2 of 4 overlapping
field(s) disagree`.

⚠ [`observer.md`](observer.md) under `OBS-04` carries the two repairs the run
located: the lab presents the same peer ID on both connections, and the tracker
answers a 60-second interval under a 45-second deadline.

### ⭐ `E-PUB-04` was unclosable by any capture, and is closed. 2026-09-15

⛔ **A record whose routes installed different bytes could not publish however
many captures existed.** `equivalence::classify` reaches `byte_identical` or
`unresolved` and never `build_equivalent`; `build_equivalent` is reachable only
from `classify_across`, which takes two records; and every publication gate
asked the per-record question one record at a time. ⚠ So the closer this
project's own record named - *a capture that observes both installs* - was not
expressible: `Capture::observed_route` is one route, deliberately.

⭐ **`agreement::publishable_among` is the same gate asked where the store is
visible**, and `routes_publishable`'s own wording had named it all along: *an
unresolved record needs a second capture through the other route*. Nothing could
act on that, because the second capture is a different record.

⛔ **The door sweep is what made it real rather than theoretical.** Four callers
gate publication and three of them hold a corpus: `index::chains`,
`index::build` and `corpus::publishable_view`. ⚠ **And a fourth had both records
in hand and asked the one-record question**: `assemble-capture` reported
publishability inside its per-lane loop, before the other lane existed, then
compared the pair two statements later. It asks once every lane is written now.

⛔ **Every comparable sibling is read, not the first that agrees.** A store
holding one capture that agrees and one that conflicts refuses the record, and
the refusal is `E-PUB-03` rather than `E-PUB-04`: the store held a comparable
capture and it disagreed, which is a different finding from holding none.

Guard mutation, five plants, each verified to have changed the file, clean
control either side:

| plant | verdict |
| --- | --- |
| the store ignored, as the per-record gate was | ⭐ refused, 3 cases |
| the first agreeing sibling wins, conflicts ignored | ⭐ refused, 1 case |
| an incomparable record counted as settling it | ⭐ refused, 1 case |
| a diverging sibling treated as saying nothing | ⭐ refused, 2 cases |
| the record left in its own sibling list | ⛔ **SURVIVED** |

⚠ **The survivor is a guard nothing can refute and it is kept.**
`classify_across` already answers `unresolved` for a record paired with itself -
*two records of one run are one record* - so removing the filter changes no
verdict. Recorded rather than deleted, and `corpus.rs` says which rule really
refuses it.

⛔ **AND THE FIRST RUN OF THAT PASS REPORTED FIVE REFUSALS OVER NOTHING.** The
harness named a `--test corpus` target that does not exist, so every plant
exited non-zero and it read each one as REFUSED - with `0 case(s)` beside it and
an empty control, which is what gave it away. ⚠ A harness exit is *could not
run*, never *refused*, and this is that rule arriving in the instrument written
to apply it. It prints the control first and refuses a non-zero exit with no red
case now.

Driven pass: `sh scripts/capture/check-assemble.sh`, every case passing.
⭐ Its control pair already installs two different digests, so those two records
were provisional and now publish; a plant that ignores the store turns exactly
the two new cases red and nothing else. ⚠ **No count of its cases is written
here**, for the reason this file records three times over: the harness grows and
a number in prose is a value in two places with nothing comparing them. It
printed 26 at this change and 32 by the end of the same session.

#### ⛔ Three scope defects stood between run 14 and that record

Each list was complete until `E-ACQ-07`'s repair made the source lane resolve its
own tag, and each defect was found by a dispatch rather than by a reading.

| what | where it is recorded |
| --- | --- |
| the rule-12 install scan read the whole workdir while the artifact ships its top level | [`acquisition.md`](acquisition.md), `ACQ-03` |
| the source route's resolution was never uploaded - the path list named `release/` alone | fixed in `capture-client.yml` |
| the reader looked for every lane's resolution at a literal `release/resolution.txt`, and asked every route for `asset_url` | fixed in `assemble-capture` |

⚠ A release resolution names its asset; a source resolution has no asset and
names the clone URL in `source_url`, deliberately, so the two slugify into
different resolvers. ⛔ The refusal for a missing resolution said *a package route
has no origin URL*, which is true of a package route and misleading for a lane
whose document did not ship; it names the route and the path it looked for now.

⚠ **`check-assemble` wrote a RELEASE resolution for every lane**, so twenty-four
cases passed over a shape no source lane has ever had. A source lane is built the
way `resolve-source.sh` really writes one now, and with that fixture alone - and
no new case - dropping the per-route origin fails 8 cases and dropping the
per-route resolution path fails 5.

### ⭐ What was repaired here rather than only recorded

- `aria2-next.sh`'s source route runs `git rev-parse HEAD` after its clone and
  writes `source-commit` beside its log; `install-client` records
  `source_commit` and **refuses a `source` route whose commit is absent or
  abbreviated**, so the gap fails at the install rather than at an assembly a
  dispatch later.
- ⭐ **And every adapter now says how its route packaged the build**, written to
  `<workdir>/package` and copied into the install record, with a route that
  installed and did not say refused there. ⛔ It is per **route**: `qbittorrent`
  delivers a `.deb` one way and an AppImage the other, and a record calling both
  `elf-binary` would say they delivered one form. `adapters/README.md` is the
  contract. That closes the third row of the table above; the first two remain.
- [`parse_transcript_document`](../crates/bit-ids-lab/src/evidence.rs) is the
  transcript writer's inverse, beside it, refusing anything that writer does not
  emit - uppercase hex, a reordered key, a missing final newline. ⛔ Nothing
  could read a bundle back before it; the assembler needed one, and a second
  parser in the assembler would have been a second reading of this project's own
  format. Two tests pin the pair, one of them byte for byte.
- `store_build` takes a package, because `assemble-capture` lives in
  `bit-ids-probe`: it decodes transcripts with `bit-ids-wire`, and `bit-ids`
  cannot depend on that crate - the dependency runs the other way.

### ⭐ What the assembler declared, and what now produces it. 2026-09-10

A connector other than the observer reports what it saw in
`connector/<id>.txt` inside the bundle, one `field_path=value` line per field the
observer measured, where the value is lowercase hex, `absent` or `out_of_scope`.
⛔ **A declared connector silent on a field is refused rather than recorded as
silent**, which is `E-COR-07`'s rule applied where the record is written.

⭐ **`OBS-07` closed on 2026-09-10 and the producer exists.**
`scripts/capture/connectors/cpython-stdlib.py` writes that file and
`capture-client` requires it, so the `E-CAP-01` row of this entry's own refusal
table is repaired. ⚠ **Unproved on a runner**: no dispatch has taken the step.

⭐ **And this entry's assembler gained the line it was missing.** It wrote a
record and never said whether it could be PUBLISHED - found the moment a lane
could carry a connector conflict, because the record kept the conflict, `to_json`
accepted it and the report printed a star and a path. It prints
`publishable` or `provisional, not publishable` with every blocker now, and
`check-assemble` asserts that a lane whose connector read different bytes lands
on `E-PUB-01` for exactly the altered fields.

### ⚠ Residuals

- ⛔ **It writes a `Profile` and not the `RunManifest` that has to sit beside
  one.** A manifest carries the run's ordered phases, both clocks, the sampling
  plan and the host facts; an attestation carries a start, a finish, a platform
  string and a claim fingerprint. Inventing the rest would produce a document
  `bind` then compares against a record that agrees with it for no reason. ⚠ So
  the store this writes is a store of records with no runs.

  ⚠ **Amended 2026-09-17: the second half of that was wrong and is struck.** It
  read *which `check-store` accepts and a publication would not*, which put the
  gate in the wrong place and a whole release away. `validate_corpus` refuses
  such a store TODAY, with `E-CRP-01` per record - *a record without its run
  cannot be replayed* - so the gate existed all along and was never a
  publication gate. ⛔ What was true is that nothing on this path **asked** it:
  a reader of a green assembly saw two stars and a path and took the store for
  one a consumer could open.

  ⭐ **The asking is what was missing and the assembler now does it.**
  `report_corpus` builds a `Corpus` from the records that reached the store,
  runs `validate_corpus` over it, and prints the violations under
  `⚠ corpus: what was written is NOT a valid corpus yet` followed by the reason
  a reader is otherwise owed a code lookup for - that no step of the capture
  path writes a `manifest.json`. ⚠ **It reports rather than refuses**, because
  validity and publishability are separate gates at every level here and a third
  gate collapsing them would stop an incomplete corpus being recordable at all.
  `check-assemble` holds both halves: that a green two-lane assembly says so,
  and - the control - that every record it wrote is named, taken out of the
  assembler's own write lines rather than retyped, so one fixed sentence cannot
  satisfy it. Driven with three plants: a fixed sentence carrying `E-CRP-01`, a
  corpus built from no records at all reported as `corpus: valid, 2 record(s)`,
  and a failure branch that says nothing. All three refused, clean either side.

  ⛔ **AND ASKING IT FOUND A REAL DEFECT IN WHAT THE ASSEMBLER WRITES.** The
  first version built the corpus over an EMPTY `StoreTree`, which narrowed the
  question without saying so - `E-CRP-06` is checked over the tree, so a store
  whose evidence no run declares answered clean because the validator had been
  handed no evidence to look at. ⭐ Reading the tree back **off the disk**, with
  the length and digest of the bytes that arrived rather than the record's claim
  about them, is what surfaced it: each record's evidence list is built over ALL
  the lanes - a field citing the other route's install record is what makes the
  pair comparable - and the write loop copied only its OWN lane's four files. So
  every record cited four artifacts at paths under its own evidence root that
  nothing had ever written to.

  ⛔ **`E-CRP-03` is exactly that refusal and it could not fire**, because it is
  checked per run manifest and this path writes none. `check-assemble`'s
  `-ge 1` file count accepted it too: half the files satisfy *at least one*. ⭐ The
  fix is one loop - every lane's artifacts under every record, which the
  route-namespaced paths already allow - and the assembler now names any citation
  it did not carry, since no rule can. Two standing cases hold it, with a fourth
  plant reverting the loop to its own lane and a fifth restoring the empty tree;
  both refused, and the empty-tree plant is caught only by the case written for
  it, which is the point of adding it.
- ⛔ **The door sweep found two more readers of a transcript, and both are
  `grep`.** `capture-run.sh` and `capture-client.sh` each assert a token appears
  in `tracker-http.transcript.json` with a fixed-string match. ⚠ A substring
  match cannot tell what the BUILD sent from what the lab sent, because both
  directions are in the document; the peer-ID check is meant to establish the
  first. `parse_transcript_document` is what would answer it properly, and
  wiring a Rust reader into the capture runner is a change to the capture path
  that needs a dispatch to prove. Filed rather than made unproven.
- ⚠ **`display_name` is the target identifier**, because nothing a capture
  uploads carries a display name and `catalogue/clients.toml` does. Reading it
  would mean a TOML dependency for one cosmetic field. It is not in the identity
  tuple, so unlike `platform`, `arch` and `package` it cannot file a record at
  the wrong path.
- ⚠ **The artifact digest is the installed executable's**, because nothing the
  capture uploads carries a digest of the bytes that arrived. `ACQ-05` is where
  a retrieval digest would come from.

### ⛔ What is left, and it is no longer this entry's alone

The publisher's dry run still cannot be reached, for the reason below and now
for several more. ⭐ The order is fixed rather than open: `OBS-07`'s second
connector, an independent resolution for the source route, and a recorded
package format are all **prerequisites for a record existing**, and the v7/v8
question sits behind them.

### What a real capture artifact answered on 2026-09-08, and what it refused

⭐ **A v7 upload survives a download and verifies.** The transmission bundle from
client capture run 4 was downloaded through `AGENTS.md` rule 8's route,
unauthenticated, and `sha256sum -c SHA256SUMS` inside it reports `OK` for all
three evidence files. That is the first time anything in this repository read a
capture bundle back outside the run that wrote it, and it is a reader this
project did not write.

⛔ **But the pairing cannot be exercised, and that is stronger than this entry's
premise.** The problem above says the publisher downloads a bundle *no run has
produced*. It is worse: nothing in the tree **can** produce it. The four uploads
here are `install-*`, `capture-client-*`, `capture-linux-*` and
`capture-windows-*`, and the publisher asks for `bundle`. Its first step fails on
every run that exists and on every run that could be dispatched today, so the
v7/v8 question cannot be reached at all.

⛔ **And a capture bundle is not a publication bundle**, so renaming would not
join them either. A capture carries `capture/SHA256SUMS` over three evidence
files and no `MANIFEST.json`; the publisher expects both documents at the root of
what it downloaded, and `publish-data.sh` pushes that tree as the publication.
What sits between them is `assemble-release`, which reads a **store of records** -
and no record has been written, so there is nothing to assemble.

⚠ **The producer is deliberately not stubbed, and this is the residual.** Every
record in the store today is synthetic, so a job that assembled one and uploaded
it as `bundle` would put a publishable-looking artifact one boolean away from
being pushed to the data branch. The missing piece is a record, which is
`CLIENT-01`'s remaining gap rather than this entry's.

### ⛔ What that record actually needs, measured 2026-09-09

**A single-route capture cannot become a record at all**, and the work order said
it could. `docs/history/RESUME.md` has carried "a single-route, single-connector
capture VALIDATES and refuses to publish"; half of that is right and the half
this entry waits on is not.

| the capture that exists | what the library does | where it bites |
| --- | --- | --- |
| one connector | **validates**, then `E-PUB-02` refuses to publish it | publishability |
| one route | `E-ACQ-01`: *1 route(s); two independent routes are required* | ⛔ **validity** |

⚠ **Measured rather than read**, by stripping one route and one connector from
the golden fixture and running `validate-profile` over each. The single-connector
copy validates and reports `provisional, not publishable` with six `E-PUB-02`
rows; the single-route copy is refused outright.

⭐ **Both facts were already in the suite and only the prose disagreed.**
`profile_schema.rs` plants `E-ACQ-01` by truncating the route list, in the loop
that asserts each plant is a **validation** refusal, and
`agreement_refuses_to_publish_a_measurement_no_second_connector_saw` reads its
document back through `Profile::from_json` - which validates - before asking
`publishable`. Nothing was wrong in the code; a sentence in a handoff was, and it
is the sentence this entry's dependency was written from.

⛔ **So `Profile::to_json` will not write it and the store cannot hold it.** Every
capture this project has run - transmission twice, qBittorrent once - is one
route, so none of them can become a record however much of the assembling code is
written. ⭐ `E-ACQ-01` is right and the document was wrong: the product IS the
two-route claim, and a record carrying one route would publish a weaker
measurement under the same schema.

⭐ **What unblocks it is a two-route capture, and there is exactly one target it
can be run on.** aria2 is the only target whose two routes currently resolve the
same version, `capture-client.yml` can now dispatch a route per host, and
`resolve-release` chooses the artifact the release route fetches. That dispatch
is `CLIENT-05`'s and this entry waits on its output.

⭐ **What is fixed is that the gap can no longer be invisible.** `check-project`
compares every `download-artifact` name against every `upload-artifact` name in
the tree, with `${{ ... }}` normalised to `*` on both sides, and refuses a
download nothing produces. The publisher declares this one with
`bit-ids:no-producer=CI-09`, the declaration must name an entry `INDEX.md`
really carries, and ⛔ **a declaration over a name that HAS gained a producer is
refused too**, so the marker cannot outlive its reason: the day the producer
lands, the gate says so.

⚠ **The rule's own first version was wrong in two ways, and only planting found
either.** It scanned forward from `uses:` and took the first `name:` it met, and
it read `name:` alone:

- a step written `- with:` / `name:` / `uses:`, which YAML permits because a
  mapping has no key order, made it report the **step's** name as the artifact's;
- `download-artifact` also accepts `pattern:`, so every download written that way
  was skipped in silence.

⛔ **Neither shape exists in this tree**, which is the condition
[`reviews.md`](../docs/methodology/reviews.md) names as the easiest place to get
a scope wrong: every reading agrees on every file. Both are planted now, and the
artifact name is taken from inside `with:`, identified by the column of that key
rather than by being the first one seen. ⚠ The first plant *passed* before the
name was checked, because a rule that never found the row and a rule that found
it and accepted it both exit 0; removing the declaration is what separated them.

⭐ Six states, both halves, same verdict on each: the clean tree, a declaration
removed, a declaration naming no real entry, a declaration gone stale, a
`pattern:` download, and a reordered step.

### ⛔ The second connector refused a build that varies, and it reports every sample now. 2026-10-14

⛔ **Both contained runs of 2026-10-11 ended in the connector, not in the build.**
`capture-client` runs 38109299467 and 38111442119 put `aria2-next` 2.8.6 on the
wire through both routes, and each *Capture* step exited 1 with the same two
lines, read from the job logs through `AGENTS.md` rule 8's route:

```text
cpython-stdlib: two segments disagree about tracker_http/peer_id; a connector that picked one would be manufacturing the constant the record declares
capture-client: the second connector could not read what the observer recorded
```

⚠ **That is all that survived.** *Upload the evidence bundle* runs only on
success, so the transcripts were destroyed with the hosts; the artifact list for
both runs holds `install-*` and no `capture-client-*`. What the lines establish
is that two announces of one capture carried two peer IDs; the capture runs two
sessions by default, so a peer ID regenerated per process is the reading the
attestation's own `sessions=2` predicts, and it is not measured.

⛔ **The refusal was the defect.** A tail that moves between sessions is what
`SCHEMA-04` calls `patterned`, and `sampling::field_state` already reads it;
the connector's rule against *picking* one of two announces had grown into a rule
against *reporting* two. ⭐ The repair is the widening `SCHEMA-04` assigned to
`OBS-07`:

| where | what changed |
| --- | --- |
| `cpython-stdlib.py` | a report line carries every sample, comma-separated in document order, each lowercase hex or `absent`; `out_of_scope` stays a whole-field value |
| `assemble-capture` | the observer keeps every observation, not the first, and `compare_samples` lines the two lists up by position: `exact` only if every position matches |
| the record | one value per connector, as before: the first position where the readers part, or the first sample when they never do, because `E-COR-14` refuses a `disagrees` whose values match |
| a count that differs | a disagreement about which segments the build sent; the reader with no sample at that position is recorded as `absent` there, and a count difference no record can show - a missing sample beside an `absent` one - is refused by name |

⭐ **Reproduced first, then shown passing.** `check-capture-client` gains a stub
that announces a different peer ID in each session, driven through the real
observer, the real `capture-client.sh` and the real connector. Against the
connector at `30a9801` it fails with exactly the two lines above; against this
one it passes and the report carries both peer IDs in the order they were sent.

| measured on this host, 2026-10-14 | result |
| --- | --- |
| `sh scripts/capture/check-connector.sh` | 51 cases, 51 passed; against `30a9801`'s connector, 9 of them fail |
| `sh scripts/capture/check-assemble.sh` | 51 cases, 51 passed |
| `sh scripts/capture/check-capture-client.sh` | 125 cases, 125 passed; against `30a9801`'s connector, 3 fail, the per-session capture among them |

Guard mutation, twelve plants into scratch copies of the tree, each replaced
exactly once and verified to have changed the file, the harness's own exit code
read unpiped, clean control either side:

| plant | verdict |
| --- | --- |
| the assembler compares the first sample only, which is what it did before | ⭐ refused, 6 cases |
| it compares up to the SHORTER list, so a count never differs | ⭐ refused, 3 cases |
| it records the first sample rather than the one that parted | ⭐ refused, 5 cases: `E-COR-14` refuses the record |
| an empty sample is skipped rather than refused | ⭐ refused, by the case written for it |
| the count-only refusal is removed | ⭐ refused, by its message alone: the validator refuses the record anyway, naming neither reader |
| the observer keeps its first observation only | ⭐ refused, 4 cases |
| a missing sample matches anything | ⭐ refused, 3 cases |
| the connector keeps its first sample only | ⭐ refused, 5 cases |
| it reports a sorted list | ⭐ refused, 2 cases, one of them the pair of announces sent the other way round |
| it reports each distinct value once | ⭐ refused, 2 cases |
| it drops `absent` samples | ⭐ refused, 2 cases |
| it refuses two that differ, which is the defect | ⭐ refused, 8 cases |

⚠ **Unproved on a runner.** No dispatch has taken this connector, so whether
`aria2-next` regenerates its peer ID per session, and whether the pair then
reaches `build_equivalent`, is the next dispatch's to say.

### ⛔ A refused capture uploaded nothing, and the obvious fix would have shipped an unscanned bundle. 2026-10-14

⛔ **The bytes above were lost because *Upload the evidence bundle* ran on
success only.** A capture that refuses is evidence: `capture-methodology.md`
says partial evidence is retained with the reason, and for `capture-client.yml`
that sentence was false.

⛔ **`if: always()` was the shortcut, and it was a rule-12 hole.** The
credential scan lives in `capture-client.sh` on its SUCCESS path, after the
connector, so a refusal anywhere before it leaves a bundle nothing has read -
and a refusal is the run whose adapter `stop`, which unlinks `aria2-next`'s RPC
token, may never have been reached. That token shipping inside an upload is what
run 11 measured.

| where | what changed |
| --- | --- |
| [`scan-bundle.sh`](../scripts/capture/scan-bundle.sh) | the two rule-12 rules, out of `capture-client.sh` and `install-client.sh` - which carried copies a comment kept identical - into one file, with `--top-level` for the install workdir's scope; a missing directory is exit 2, never clean, and both rules are reported when both fire |
| `capture-client.yml` | *Scan the evidence of a refused capture* runs only when *Capture* failed, and *Upload the evidence bundle* runs on a capture that succeeded or a scan that passed |
| `check-workflow` | an evidence upload carrying an `if:` must name the scan's outcome, and the scan step must carry its id, run `scan-bundle.sh` over the bundle, and sit between the capture and the upload; refuted by an upload on `always()` and a scan step that runs `true` |

⚠ **A door this found in two harnesses.** `check-step-bodies` and
`check-capture-client` each build a partial tree to run `install-client.sh` in,
and both answered 2 the first time: the scan the script now asks was not in
either tree. ⭐ That is the scan's could-not-run doing its job rather than
reading a missing file as clean.

| measured on this host, 2026-10-14 | result |
| --- | --- |
| `sh scripts/capture/check-capture-client.sh` | 130 cases, 130 passed, five of them asking `scan-bundle.sh` directly |
| `sh scripts/ci/check-step-bodies.sh` | 31 cases, 31 passed, with `scan-bundle.sh` in its tree; seven of them are `CI-08`'s recorder cases |
| `sh scripts/acquisition/check-rootless.sh` | 26 cases, 26 passed |
| `sh scripts/ci/check-workflow.sh`, unsharded | 111 cases, 111 passed, three probes each for the rule-12 gate and the recorder-stop rule among them |
| `sh scripts/common/check-gate.sh` | 40 checks, 39 passed, 1 skipped |

Guard mutation, seven plants into `scan-bundle.sh` in a scratch copy of the
tree, each replaced exactly once and verified to have changed the file, run
through `check-capture-client` with its exit code read unpiped, clean control
either side:

| plant | verdict |
| --- | --- |
| a missing directory answers clean | ⭐ refused, by the case written for it |
| the scan stops at the first rule that fires | ⭐ refused, by the case that plants both |
| `--top-level` is ignored, so the install scan descends | ⭐ refused, 2 cases: run 15's vendored certificate is refused again |
| the whole-tree scan stops descending | ⭐ refused, 3 cases, both capture-path refusals among them |
| a substring rather than a word, so `tokenizer` is a token | ⭐ refused, 2 cases, one per caller |
| the value is printed rather than its file named | ⭐ refused, by the case written for it |
| the scan always answers clean | ⭐ refused, 6 cases: two through the capture runner, two through `install-client`, two asking the scan directly |

⚠ **`capture.yml` still uploads on success alone**, and its fixture captures are
not scanned at all: `capture-run.sh` installs nothing and writes no product
output. `docs/capture-methodology.md` says so beside the sentence this makes
true for client captures.

### ⛔ Run 35: the evidence came back, and `3.0.2` stopped both lanes. 2026-10-14

`capture-client` run 35, id 38511515697, dispatched on `a39d933` with the
defaults: `aria2-next` over release and source. Read from its two
`flight-recorder (...)` check runs, its job steps and its three artifacts.

| | release | source |
| --- | --- | --- |
| resolved | `3.0.2`, its asset `OK` against the vendor's checksums | `v3.0.2` from the repository's refs, commit `d518e905…` |
| *Install the client* | 6 seconds, `acquired=yes` | failed after 508 seconds, in the compile |
| *Capture* | ⛔ refused at the adapter's `start`: *reports enable-dht6 is not false* | not reached |
| evidence | ⭐ *Scan the evidence of a refused capture* passed and the bundle uploaded, carrying the options dump that says why | the install logs |
| the recorder | closed, `completed`/`neutral`, over an empty text | the same |

⭐ **The section above held on a runner, the first time it was asked.** A
capture refused, the scan read its bundle, and the upload followed the scan:
without it nothing below could have been diagnosed from anything but a job log.

⛔ **The release lane: a guard was right for the smaller reason.** `3.0.2` no
longer reports `enable-dht6`, and its dump shows `ed2k-port-mapping` on by
default, a surface the adapter never named. The adapter now switches it off and
sweeps every reported switch of the four discovery families; driven against the
same bytes in a namespace holding loopback alone. `CLIENT-14` in
[`clients.md`](clients.md) carries the measurement.

⛔ **Two ways the dispatch could not be read, both now fixed.** The recorder's
closing request erased the snapshot it had just posted, and *Install the
client*'s `rc=$?` under `bash -e` meant its log never reached the job log.
`CI-08` carries both.

⛔ **The source lane: `v3.0.2` does not compile on the capture host.**
`src/disk/File.cc` calls `std::exchange` and never includes `<utility>`, and the
runner's GCC 13 stops there. The vendor's CI and its Linux release jobs build on
`ubuntu-22.04`, and the asset the release lane installed says so in its own
`.comment` section: `GCC: (Ubuntu 11.4.0-1ubuntu1~22.04.3) 11.4.0`. This route
builds where the capture runs, on `ubuntu-24.04`. Reproduced on the session
host twice: from `File.cc` and the two headers it includes, by GCC 13 and by
clang 18 over the same libstdc++, and from the whole tag through
`cmake --preset default`, one failure on the same line. ⚠ The vendor's `main`
carries the same file.

⭐ **Fixed in the route, with no file the vendor wrote changed.** `aria2-next.sh`
configures with `CXXFLAGS='-include utility'`: the header the compiler's
diagnostic names, holding a declaration the source already calls. The
superbuild forwards `CMAKE_CXX_FLAGS` to the inner project and its seven
CMake-built dependencies, and the flag is set rather than appended to a host's
own; OpenSSL, FFmpeg and GPAC configure themselves and do not receive it. The
line it ran is written to
`source-configure.txt`, which the install artifact uploads. ⚠ The record has no
field for it, which `ACQ-03` already says of every source route's configure
options.

| measured on the session host, 2026-10-14 | result |
| --- | --- |
| `v3.0.2` unmodified, `cmake --preset default` | configured, then stopped at `File.cc:207`: the runner's failure, and the only one |
| the same tag with `CXXFLAGS='-include utility'` | configured and built every target in 870 seconds: sha256 `6743a8b2…87cd4bb`, `.comment` `GCC: (Ubuntu 13.3.0-6ubuntu2~24.04.1) 13.3.0`, answering `aria2-next version 3.0.2` |
| the adapter's `start` over that build, in the loopback namespace | exit 0, through every guard `CLIENT-14` describes |
| the adapter's own `install source`, its prefix under a scratch `HOME` | exit 0 in 869 seconds: commit `d518e905…`, `elf-binary`, `version` answering `3.0.2`, and `source-configure.txt` holding the quoted line and the host's `CFLAGS`, `CPPFLAGS` and `LDFLAGS`, each `<unset>` |

⚠ **The bound moved with it.** 870 seconds here sits against 900 on the runner.
A runner has run slower than a session host - 2.7.5 took 496 seconds there
against 366 on the host of 2026-09-09 - and faster: run 35 reached the compile
error in 508 seconds, its clone included, where this host took 588. The
source lane's install bound is 1500 seconds now, the step's 1860, the outer
1980, the step's timeout 35 minutes and the job's 45; the release lane's are
unchanged. The next dispatch measures the real number.

### ⭐ Run 36: both lanes put 3.0.2 on the wire, and the pair is one sample short. 2026-10-14

`capture-client` run 36, id 38525418038, dispatched on `8870eed`, green on both
lanes, read from its recorder runs and assembled from its four artifacts into
two records in a scratch store.

| | release | source |
| --- | --- | --- |
| *Install the client* | 6 seconds | **694 seconds**, under the 1500 this lane now allows and the 900 it had |
| installed executable | `09e34467…afb0d4`, the vendor's asset | `6cdfcdd5…db5c10`, built on the runner |
| `--version` | 544 bytes | the same 544 bytes, byte for byte |
| *Capture* | 45 seconds | 45 seconds |
| announces | four, in two sessions; each session's pair carries one peer ID, and the second session draws a new tail | the same shape |
| peer connections | the lab dialled two and the build answered the first only | the same |
| `tracker_http/user_agent` | `qBittorrent/5.2.3`, constant over four samples | the same |
| `peer_wire/peer_id` | `-qB5230-` and a tail matching neither session's announces, one sample | the same, another tail |

⭐ **The second connector corroborated every sample**: all four fields `exact`
in both records, four samples of `tracker_http/peer_id` among them. That is the
every-sample fix of 2026-10-14 on a real dispatch for the first time, over the
build that varies its tail between sessions.

⭐ **Both recorder runs closed with their last snapshot**, 19,112 and 20,923
characters, each through its `route to the API` section. `CI-08`.

⛔ **The pair is `provisional` and not publishable, and the reason is one
sample.** `E-PUB-03` refuses it because `peer_wire/peer_id` is `constant` in each
capture with a different value, and `classify_across` answers `divergent`. One
answered connection per lane gives one handshake, so nothing in either record
can show the tail varies; the announces show it does. ⚠ `OBS-04` carries the
hypothesis this run strengthens and does not test: both connections reach the
build from `127.0.0.1`. libtorrent's `allow_multiple_connections_per_ip` is
`false` by default in the vendored `settings_pack.cpp`, and nothing in
`aria2-next`'s own source sets it. That is source, which sets priority and
populates nothing; a lab dialling its second connection from another loopback
address is what measures it.

⚠ **And the corpus refusals are this entry's own residual**: nothing on the
capture path writes a `manifest.json`, so both records fail `E-CRP-01` and their
evidence `E-CRP-06`.

### ⭐ Run 37: both peer connections answered on a runner, and a coincidence refuses the pair. 2026-10-14

`capture-client` run 37, id 38543482073, on `18b21b8`, green on both lanes.
The vendor had published **3.0.3** since run 36, and both routes resolved it:
the release asset `0350294c…` and the runner's source build `7bb82fc9…`,
installed in 6 and 455 seconds. Both lanes' adapters passed every discovery
guard `CLIENT-14` describes.

⭐ **`OBS-04`'s change held on a runner.** On each lane the re-dial left from
`127.0.0.2`, `peer-streams` was 2, and `peer_wire/peer_id` is `patterned` over
two samples: `-qB5230-` and a twelve-byte tail. It **agrees across routes**,
which is the field run 36 was refused over.

⛔ **The refusal moved to `tracker_http/peer_id`, and it is a coincidence the
classifier reads as a fact.** Each capture's four announces carry two distinct
peer IDs, one per session, and in each capture those two random tails happen to
share one byte at one offset:

| lane | the two tails | what the classifier made of them |
| --- | --- | --- |
| release | `A(Mq~CueHfWD`, `bEMwBhHOPtBp` | `-qB5230-`, two varying, **`M` fixed**, nine varying |
| source | `)4*kPYzfVub6`, `-bsHBBNc_uUC` | `-qB5230-`, nine varying, **`u` fixed**, two varying |

`SCHEMA-04`'s per-offset rule calls a byte fixed when every sample carries it,
and two random values match at any one offset about one time in sixty-four if
the tail draws from some sixty-four symbols, which is an estimate rather than a
measurement; a twelve-byte tail sampled twice would then show such a byte in
about a sixth of captures. ⛔ **Two such patterns cannot agree**, so `E-PUB-03` refuses the pair
over one byte neither build fixes. `SCHEMA-04` owns the repair: a fixed byte
inside a span that varies has to be one the samples can establish, not one two
values happened to share. The work order leads with it.

⚠ CI run 188 on the same commit failed one control in `check-workflow` shard 1,
*the repository gate, after every restore*, without saying which check. The same
shard passed locally, 45 of 45; the job was re-run once, and the control now
prints the gate's failed rows.

⭐ **And assembled again once `SCHEMA-04` stopped calling a shared random byte
fixed: the first publishable pair from a stock client.** The same four
artifacts, the same assembler with that one change: both records
`publishable`, `classify_across` `build_equivalent`, four fields agreeing
across the release and source routes of `aria2-next` 3.0.3. ⚠ They are not a
corpus yet: nothing on the capture path writes the `manifest.json` a record
needs beside it, so `E-CRP-01` refuses both, and nothing uploads the `bundle`
artifact the publisher reads. Those are this entry's residuals and now the
whole of what stands between a capture and a publication.

### ⛔ The run manifest a pair needs, and the two facts it cannot state yet. 2026-10-14

Surveyed field by field against run 37's four artifacts: what a `RunManifest`
needs, where each value exists today, and what writes it.

⭐ **Most of it is a producer change, and none is invented.** The assembler can
take the target, the build tuple, the kernel and architecture, the network
isolation, the end of the run, the sampling plan, the routes, their sources and
digests, and every evidence row from what the run already uploads. What it
cannot take is written nowhere yet:

| missing | the smallest producer that writes it |
| --- | --- |
| the runner image, the operating system, whether the host is GitHub's own, and an uptime at claim | `assert-disposable.sh --claim` writes them into the claim marker, and `capture-client.sh` uploads the marker |
| a monotonic span | an uptime at claim and one at finish |
| the connector's and the observer's versions | `capture-client.sh` keeps the version the connector's `--describe` prints and drops today; the observer prints its own |
| each route's byte count, retrieval time and signature status | `install-client.sh` writes them beside the digest |
| each evidence row's kind, producer and phase | `client-capture` prints them on the rows it already prints |

⚠ **And the profile carries one invented value today**: both connectors at
version `0.0.0`. For the observer it happens to equal the crate's version; for
`cpython-stdlib` it is nobody's measurement. A manifest that copied it would
bind for no reason, so it is repaired first.

⛔ **Two facts a dispatch produces do not fit the manifest's model, and they are
the operator's to decide.**

1. **One host or two.** Each route installs on its own runner, and a manifest
   carries one host, one isolation and one pair of clocks while `E-MAN-10`
   requires both routes in it. Either the manifest describes the watched host
   and the other route contributes its acquisition and evidence, or the schema
   carries a host per route.
2. **The phases overlap across hosts.** `E-MAN-03` puts both routes' installs
   before the observation, and run 37's release lane observed from 09:36:22 to
   09:37:07, before the source route finished installing at 09:43:40 on
   another machine. So `cap-release` cannot have a truthful manifest under the
   present state machine. The options are phase spans read as milestones with
   the release record refused, a workflow barrier so every lane waits for every
   install, or a state machine that lets one route be observed before the other
   is installed.

⭐ **Both settled by the operator on 2026-10-14**: a workflow barrier, so every
lane's capture waits for every route's install, and a manifest that describes
the watched host. The order of work: the invented connector version first, then
the producers above, then the barrier, then the assembler writing the manifest
and reading it back into the corpus check.

⭐ **Step 1, the invented version, is done.** `client-capture` prints
`observer-version`, `capture-client.sh` keeps the connector's own described
version, both land in the attestation as `observer_version` and
`connector_version`, and `assemble-capture` reads each connector's version from
there and refuses a lane without one. `check-assemble` 54 of 54 and
`check-capture-client` 131 of 131; a plant writing `0.0.0` again was refused by
three cases, and one dropping the attestation key by one. ⚠ Run 37's artifacts
predate the keys, so they assemble no longer: that is the refusal working, and
the next dispatch carries them.

⭐ **Steps 2 and 3, the producers, are done.** The claim marker carries the
runner image, the operating system's `PRETTY_NAME`, `RUNNER_ENVIRONMENT` and
`/proc/uptime` at the claim, each written as read and empty where the host says
nothing; `capture-client.sh` uploads it as `host-claim.txt` and writes the
uptime at finish into the attestation. The install record carries
`installed_binary_bytes`, the size of the file its digest names, a
`retrieved_at`, and `signature=not_checked`, because no route checks a
signature. ⚠ `check-capture-client`'s install stub has no executable, so only the
empty half of the size rule is driven here; the next dispatch drives the other.

⭐ **Step 4, the barrier, is written.** After *Install the client* each lane
arrives with a check run named for its run attempt, adapter and route,
`success` or `failure`, and waits until every route of its adapter has arrived;
the capture follows. `scripts/ci/install-barrier.sh` never fails a lane, so a
route that cannot be satisfied still leaves the others' measurements intact,
with `failed` or `timeout` in the `install-barrier.txt` the install artifact
uploads. `check-step-bodies` drives both bodies against a stub `curl` that keeps
check runs by name: both routes arriving pass, a failed route ends the wait at
once and is named, a route that never arrives is a timeout, and an earlier run
attempt's barrier on the same commit does not count. Three plants were each
refused: the attempt left out of the name, a failure read as missing, and a
missing route read as arrived. `check-workflow` puts the wait after the install
and before the capture.

⭐ **Step 4 proved on a runner: run 38, id 38562743496.** The release lane
waited 415 seconds at the barrier for the source route, and both lanes'
`install-barrier.txt` record `verdict=passed`. Both records are `publishable`
and `build_equivalent` on `aria2-next` 3.0.3.

⭐ **Step 5, the assembler writes the manifest, is done.** `assemble-capture`
builds one `RunManifest` beside each record at `raw/v1/<package>/<version>/
<os>/<arch>/<artifact>/cap-<route>/manifest.json`. It describes the watched
host from that lane's claim marker. Both routes' acquisitions and evidence are
in it. Its tools are the observer, `install-client` and the connector, at their
measured versions. Its phases come from the timestamps the run wrote. Before it
writes, it validates the manifest and binds it to its profile with
`bit_ids::bind`. A barrier verdict other than `passed` is a warning, because
the timestamps decide the order. The corpus line now counts manifests as well
as records.

Run 38 assembled into a store that `validate-corpus` reads as `valid store: 2
record(s), 2 run(s), 20 object(s); 2 of 2 record(s) may enter a published
view`, and `validate-run` binds both manifests. `check-assemble` 58 of 58: its
lanes carry the new keys, the corpus cases bind 2 manifests found anywhere in
the store, and five cases each refuse a defect by name: no host claim, no
uptime at finish, no installed byte count, a self-hosted runner, and an install
that finished after the observation (`E-MAN-06`). Two plants were each refused
with clean controls on both sides: accepting any runner environment, and
reading the install's end as the capture's start. ⛔ What is left of this
entry's residuals is the `bundle` artifact `publish-data.yml` downloads.

### ⭐ The `bundle` has a producer, and the first one found two defects. 2026-10-14

[`assemble-bundle.yml`](../.github/workflows/assemble-bundle.yml) is a dispatch
that takes a `capture-client` run's id. It downloads that run's captures and
installs, and `assemble-dispatch.sh` (since 2026-10-15 `bit-check assemble-dispatch`)
pairs the lanes by artifact name and runs `assemble-capture` for every adapter
into one store. `assemble-bundle.sh` (since 2026-10-15 `bit-check assemble-bundle`)
then refuses a store that is not a valid corpus. It composes the indexes, every
rendering, the licence, `MANIFEST.json` and `SHA256SUMS`, and the job uploads
the tree as `bundle`. ⚠ It is a workflow of its own rather than a job in
`capture-client.yml`, because `check-workflow` holds every job in a capture
workflow to claim, install and capture, and this one does none of them.

⭐ **The version scheme is not declared a second time.** Each adapter declares
it in `--describe`, each resolver records the scheme it ordered tags under in
its `resolution.json`, and `assemble-bundle.sh` takes it from there. Two resolutions that order one target differently are refused.

⛔ **Two defects, each found by assembling run 38 by hand before anything was
written:**

1. `assemble-release` refused the store with `E-REL-01` four times. The
   assembler stored each route's version output as `version.err`, a name with
   no media type. It is stored as `version.txt` now; the bytes are the same.
2. `publish-data.sh --dry-run` reported a bundle with no `MANIFEST.json` and no
   `SHA256SUMS` clean, and a real run would have pushed it and refused only at
   the read-back, leaving a commit on `data` nothing could verify.
   `check-publish` asserted that shape as a pass. The publisher now refuses
   such a bundle, or one whose digests do not match, before anything is
   committed, and the case asserts the branch is untouched.

Driven: run 38's artifacts, under their uploaded names, became a 30-file
bundle with release digest
`sha256:77cb519cfa123edf8d462492ea0739a494d63b5159e0a2e57e6033865d93df0c`,
identical across two assemblies. Both records are admitted to the published
view. `publish-data.sh` appended it to a scratch bare repository and read it
back. `check-bundle` 13 of 13, `check-assemble` 64 of 64 with six dispatch
cases, `check-publish` 19 of 19.

Mutation table, each plant against the case written for it:

| plant | refused by |
| --- | --- |
| the publisher skips the `MANIFEST.json` and `SHA256SUMS` presence check | `check-publish`, 2 cases |
| the publisher skips the bundle's checksum check | `check-publish` |
| two schemes for one target are not compared | `check-bundle` |
| a half-built bundle is not removed | `check-bundle` |
| the store is not validated before the copy | `check-bundle` |
| any top-level entry is copied | `check-bundle` |
| an adapter with one route is accepted | `check-assemble` |
| two attempts are mixed | `check-assemble` |
| a capture with no install beside it is accepted | `check-assemble` |
| a lane with no `resolution.json` is skipped | `check-assemble` |

⚠ **Two read-backs were removed, not kept.** `assemble-bundle.sh` first ran
`sha256sum -c` and `validate-corpus` over its own output. A plant disabling
either was refused by no case, because nothing ahead of them can make them
fire short of a broken `assemble-release`. `check-bundle` reads the bundle back
from outside instead, and the publisher's pre-push check is proven.

⭐ **Driven on runners, end to end, on `c23bd37`.** `assemble-bundle` run
38574255232 assembled `capture-client` run 38 into a 30-object `bundle`, with
the same release digest `sha256:77cb519c…` as the session host. Both records
are admitted to the published view. `publish-data` run 38574390252, with
`dry_run` true, downloaded it and verified every digest with the runner's own
`sha256sum`. It reported *dry run, 30 object(s) staged for data, nothing
pushed*. The `data` branch still does not exist.

⛔ **Nothing has been appended to `data`.** Whether to append run 38's bundle,
by dispatching `publish-data.yml` on run 38574255232 with `dry_run` false, is
the operator's call. Every other step of this entry's path has run.

### ⭐ THE FIRST PUBLICATION. 2026-10-15

The operator settled it on 2026-10-15, and `publish-data` run 38663060205
appended `assemble-bundle` run 38574255232 to `data` with `dry_run` false:
*appended to data: 30 object(s), read back and verified*. The branch is at
`99dcb486`, one commit. ⭐ Read back from outside the publisher: a fresh clone
of `data` holds 30 files, `sha256sum -c` agrees with every row of its
`SHA256SUMS`, and `validate-corpus` over the exported tree reads *valid store:
2 record(s), 2 run(s), 30 object(s); 2 of 2 record(s) may enter a published
view*. Those are `capture-client` run 38's `aria2-next` 3.0.3 records, one per
route. ⚠ Every later publication appends to this one, and `check-store` is
what refuses one that would not.

### ⛔ The second publication was refused, rightly, and the bundle now assembles onto the published tree. 2026-10-15

`capture-client` run 38680098567's Transmission 4.1.3 pair assembled as
`assemble-bundle` run 38689243666. Both records were `publishable`, the routes
`build_equivalent`, and the 4 overlapping fields agreed. `publish-data` run
38689473368, a dry run, then refused it with `E-STO-20` on every path the first
publication had written: the bundle carried Transmission alone, so appending it
would have deleted aria2-next. Nothing was pushed.

⭐ **A bundle is the whole store now.** `bit-check assemble-bundle --published
DIR` copies the published tree's `profiles/` and `raw/` under the run's own
records. A path both carry must be byte-identical, or the assembly is refused
before the publisher sees it. `assemble-dispatch` passes the flag through.
`assemble-bundle.yml` reads the `data` branch into a directory first, and only
`ls-remote --exit-code` answering 2, a ref that does not exist, makes it a first
publication. Any other failure ends the step.

⛔ **And a published record's scheme is published too.** The first records'
resolutions were in their capture's artifacts, which expire, so nothing could
order aria2-next for the latest view. Driven on the real artifacts and the real
`data` tree, the index refused with `E-VIW-01`. A bundle now writes
`indexes/v1/schemes.txt`, one `target:prefix:min:max` per line, and the next
assembly reads it back. A line that is not a scheme, and a published scheme the
resolutions contradict, are refused. ⚠ The one tree published before this
carries no such file. Its bootstrap is to re-assemble run 38 onto it: the same
records land on their own paths with the same bytes, which is accepted, and the
scheme is written. Run 38's install artifacts expire on 2026-11-14.

`check-bundle`, 21 of 21, adds seven cases:

- a bundle names its scheme;
- the next one carries both sets of records;
- the publisher appends the second onto the first;
- the same records re-assembled onto their own publication are accepted;
- a changed byte at a published path is refused;
- a contradicting published scheme is refused;
- a malformed published scheme line is refused.

Four mutants were each killed:

- the merge skipped;
- a rewrite accepted;
- the published schemes ignored;
- the schemes file not written.

The workflow step's own body was driven against the real remote.

## CI-05: Acceptance commands that cannot pass over nothing

Source: found while closing `OBS-01` on 2026-09-05
Priority: P1 | Effort: S | Status: DONE

Problem: An entry's `Prove` is the acceptance, and one of them ran nothing while
exiting 0. `cargo test --workspace --locked lab_supervisor` filters by test
**name**; the name matched none, so every binary in the workspace printed
`running 0 tests` and the command succeeded. Nothing in the gate can tell that
from an acceptance that passed.

Premise: Measured on 2026-09-05, not read. The nine `cargo test` acceptance
commands in `TODO/` were all of the bare-filter form, and they worked only
because of a convention nothing checks: all 110 test functions across the eight
pre-existing test files begin with their file's name. `lab_supervisor` did not,
and the acceptance went green over zero tests.

⚠ **This entry claimed all nine had been rewritten, and that was false.** Only
the observer entries were. Re-measured while closing this one: five `Prove`
commands were still of the bare-filter form, in `FOUND-03` and in all four
`SCHEMA-*` entries. They are corrected now, and each corrected command was run
before it was written down. That the claim survived a session is the argument for
this entry: a fact about the record that only a person checks is a fact that
drifts.

Approach: A rule in `scripts/common/check-project.sh` and its PowerShell twin
that reads every `cargo test` invocation and refuses a bare word argument unless
it follows a flag that takes a value. ⚠ **Amended 2026-09-17: both halves are
gone and the rule is `bit-check check-project`**, ported and compared case for
case under `CI-10`. The paragraph is left as it was written because it is what
was built; this note says where the rule lives now. ⚠ The parsing is the work: a code span can
wrap across lines, so the paragraph is joined before the spans are found, and the
two twins must agree per planted mutation rather than on a clean tree.

⭐ **Scoped to `Prove:` paragraphs, and that scope is the whole rule rather than
an exclusion list.** A `Prove` is the live acceptance and has to be runnable. A
`Closure evidence` paragraph records what was run on a past tree, and rewriting
one would falsify the record; and this entry and `OBS-01` both have to quote the
command that caused the defect. A rule that fired on those is a rule somebody
switches off, and the seven remaining bare filters in the tree are all of exactly
those two kinds.

⛔ **The door sweep found a second door and it is the one that matters more.**
An entry's `Prove` is the acceptance a person runs; the workflow's `run:` is the
one every push runs, and a bare filter there would report green over zero tests
on every commit with nobody reading it. The rule covers
`.github/workflows/*.yml` as well, with separate extractors and one tokeniser,
because a rule on one of two doors into the same mistake is the shape
`docs/methodology/reviews.md` names.

Decision: a shape rule over a naming convention. Renaming every test function to
start with its file's name would also make the bare filter work, and it makes
the filter's correctness depend on a convention no check holds, while
`docs/conventions/code.md` asks for test names that describe behaviour.

Prove: `sh scripts/common/check-project.sh` and
`pwsh -File scripts/common/check-project.ps1` both exit 1 when a `Prove` line is
rewritten to the bare-filter form, both exit 0 on the tree as it stands, and
both agree on every planted mutation.
⚠ **Amended 2026-09-17**: the two halves no longer exist, so the runnable form
of this acceptance is `go run ./tools/check check-project` from the repository
root, and `sh scripts/common/check-bitcheck.sh` plants the bare-filter form as a
standing case. The line above is kept as the acceptance that was actually run.
⛔ **Corrected 2026-09-29: that runnable form does not run.** From the
repository root it exits 1 with *cannot find main module*, because the module is
`tools/check`; measured, not read. `go run . check-project` from that directory
is the form that exits 0, and it is the one `ACQ-05`'s amended acceptance uses.

Closure evidence: run on 2026-09-05. Both halves exit 0 on the tree as it
stands, and `sh scripts/common/check-gate.sh` and
`pwsh -File scripts/common/check-gate.ps1` both pass with `check-twins` green.

Guard mutation: 21 cases planted one at a time, 16 into a `Prove` paragraph and
5 into the workflow, each verified to have changed the file. Both halves were run
on every one and their **exit codes and their output** compared, because
`check-twins` compares the two on the tree it runs against and a rule that
differs only on a defect the tree does not contain is invisible to it. All 21
landed on the intended verdict and the twins agreed on all 21, character for
character.

⭐ The cases are the shape space rather than one example: a bare filter with and
without other flags, a filter after `--`, a filter after a value-taking flag, a
filter after a flag whose value was attached with `=`, a filter in a code span
that wraps across lines, a target in a span that wraps, two commands in one
`Prove`, a `Prove` running no `cargo test` at all, a bare filter in a paragraph
that is not a `Prove`, and a commented-out one in the workflow. The last three
are the ones that would make the rule fire on correct usage, and each is
accepted.

Residual: the rule cannot see the other half of the class, which is
`--test <target>` skipping the library's own tests. That form runs something, so
it is not "passing over nothing", and refusing it outright would fire on a
package with no library tests. `docs/conventions/forbidden-patterns.md` carries
the class. ⚠ Two open entries still carry it in their `Prove`, `OBS-10` and
`CORPUS-02`, and both are corrected when they close, as `OBS-09`'s was.

## CI-10: Every checking script in Go, and CI across multiple runners

Source: operator direction on 2026-09-10, after a job was cancelled at its bound
three times in one session
Priority: P1 | Effort: XL | Status: DONE

Problem: The checking layer is shell and PowerShell, and it is slow in a way no
single row explains. `check-workflow.sh` is ONE job that runs the whole gate
about ten times in sequence; the gate itself forks a process per assertion,
copies a tree per case, and keeps a `.ps1` twin of every `.sh` so the same work
is written and run twice. ⛔ It is the CI wall clock and has been for several
sessions.

⛔ **What deferring it cost, measured on 2026-09-10.** The *Workflow acceptance*
job was cancelled at its 30-minute bound **three times**, at about 30m15s each.
Two fixes were tried on the wrong cause - `check-defaults` was moved out of the
gate, which saved roughly five minutes and did not bring the job back under the
bound - and the session then RAISED THE BOUND to 45, which is the masking move
this repository refuses everywhere else. ⚠ A fourth run was cancelled by this
project's own next push before it could prove the raised bound, so even that is
unmeasured.

⛔ **AND THE DIRECTION IS NOT "MEASURE, THEN MAYBE PORT".** This entry's first
draft made the port conditional on a measurement. The operator's direction is
unconditional and is recorded as given: **port all the checking scripts to Go,
port the slow CI parts to Go, and parallelise CI across multiple runners.**

Premise: The cases are independent. Each copies the working tree into a scratch
repository, plants one defect, runs one step against it and restores - so nothing
in the design requires them to share a machine, and the sequence is an accident
of how the harness grew rather than a property of what it proves.

Approach:

1. ⭐ **Port the checking scripts to Go.** One binary, cases as goroutines over
   one tree snapshot: no fork per assertion, no tree copy per case, and no twin
   to keep in step. ⛔ **The twin layer is not translated, it is DELETED.**
   `check-twins` exists because two hand-written halves drift; one binary that
   runs on both platforms removes the class rather than checking for it, and
   `CI-07`'s remaining class-A backlog stops being work at all.
2. ⭐ **Port the slow CI parts to Go**, which is `check-workflow` above all - the
   harness that runs the whole gate about ten times.
3. ⭐ **Parallelise CI across multiple runners.** A shard selector on the harness
   and a GitHub matrix, so the concurrency comes from runners rather than from
   one machine's cores. ⚠ Shard the LOCAL run too, or a contributor keeps paying
   a sequence CI no longer runs.

⚠ **`docs/AGENTS.md` section 5 changes with this entry, in the same change as
the work.** It currently makes shell the default orchestration language and Rust
the language of every core component, and names Python only under a documented
constraint. A Go checking layer is a fourth language and the router has to say so
rather than leave the next session to discover it from a file extension.

⛔ **What must NOT change while porting.** Every guard's *verdict* is the
contract, not its implementation: a ported check refuses exactly what its shell
half refused, over the same planted defect, with the same exit-code vocabulary -
0 held, 1 refused, 2 could not run. ⚠ A port that changed one verdict while
getting faster would be trading the thing being measured for the measurement.

Prove: the Go harness refuses every defect its shell predecessor refused, case
for case, over the same plants; `--shard 1/N` runs a strict subset and the N
shards together run every case exactly once with none dropped and none
duplicated; and the CI matrix completes inside a bound that was not raised to
accommodate it.

Residual, before the work starts: the shard split must be checked rather than
assumed. A harness that silently dropped a case would report a faster green run
over less work, which is this repository's oldest defect class and the one a
wall-clock target makes most tempting. ⛔ And the port needs a case-for-case
comparison against the shell halves BEFORE they are deleted, because a Go
harness compared only against itself is the self-consistency `OBS-07` was
opened for, arriving in the checking layer.

### ⛔ WHERE THE WALL CLOCK ACTUALLY IS, measured 2026-09-10 before porting

The entry's Problem says the layer is slow "in a way no single row explains".
⚠ **That was wrong, and one row explains most of it.** Every twin pair was timed,
each half unpiped, on this host:

| pair | `.sh` | `.ps1` |
| --- | ---: | ---: |
| `check-control-bytes` | 1.71s | **67.88s** |
| `check-markers` | 1.30s | 12.74s |
| `check-project` | 0.84s | 2.91s |
| `check-one-home` | 0.23s | 2.74s |
| `check-catalogue` | 1.50s | 2.57s |
| `check-placeholders` | 0.06s | 1.96s |
| `check-no-secrets` | 0.13s | 1.63s |
| `check-docs` | 1.39s | 1.42s |
| `check-cache` | 0.18s | 1.06s |
| `check-licences` | 0.06s | 0.67s |
| `check-changelog` | 0.02s | 0.50s |
| **together** | **7.4s** | **96.1s** |

⭐ **The PowerShell halves are thirteen times the shell halves, and ONE of them is
seventy per cent of that.** `check-twins` runs both halves of every pair and was
69 seconds; this is what those 69 seconds are. ⛔ And `check-workflow` runs the
whole gate about nine times, so the twin layer is roughly ten minutes of every
push.

⚠ **The measurement does not make the port conditional and is not offered as
though it did.** The direction is unconditional and recorded as given. What it
decides is the ORDER: the two rows above are ported first because they are the
wall clock, not because they were the easiest.

### ⭐ Step 1 of 3: the binary exists and two rules are in it, 2026-09-10

[`../tools/check/`](../tools/check/) is a Go module with an **empty require list
and therefore no `go.sum`**. ⚠ That is a supply-chain property rather than a
preference: nothing is fetched at build time, so a build needs no network and no
pin audit, and `CI-04`'s dependency surface does not grow by a language.

`check-control-bytes` and `check-markers` are ported. The scope is still asked of
`git ls-files` by shelling out, deliberately: a Go reimplementation walking the
filesystem with its own idea of `.gitignore` would be a SECOND answer to what is
in the tree, and two answers drift in the direction that keeps a check green.

### ⛔ The comparison that had to happen before anything is deleted

`scripts/common/check-bitcheck.sh` is one case list in two modes.

| mode | what it asks | cost |
| --- | --- | ---: |
| default | the plants, against the Go binary alone | **1.1s** for 18 cases, including building it |
| `--compare` | each case additionally against the `.sh` half and the `.ps1` half, refusing any difference in exit code **or** in the `--json` line | about seven minutes, because it starts the 67.9-second PowerShell half once per case |

⭐ **Measured 2026-09-10: 18 cases, 18 passed, 0 failed, all three implementations
agreeing on the exit code and byte for byte on the JSON.** Eight are plants that
must be refused and six are plants that must be ACCEPTED - a specimen inside a
fenced block, a leading byte-order mark, tab and carriage return, a control byte
in a file the extension list does not call text - because over-strictness is
where a port fails and a harness of refusals alone never looks there.

⛔ **`--compare` is deliberately not a gate row.** Running the layer this entry
exists to delete, once per case, would cost more than the layer does. The default
mode is the permanent row and stays one after every half has gone, because a
plant does not need a second implementation to be a plant.

### ⚠ The blind spot reproduced in the replacement, on its first mutation pass

Four divergences were planted into the Go half, one at a time, each verified to
have changed the file before it was judged:

| plant | verdict |
| --- | --- |
| the density ceiling 30 changed to 20 | ⭐ refused, exit 1 |
| `md` dropped from the marker scope | ⭐ refused, exit 1 |
| the leading-BOM exemption removed | ⭐ refused, exit 1 |
| **DEL (0x7f) added to the control class** | ⛔ **SURVIVED, exit 0** |

⛔ **The fourth is `check-twins`' own documented blind spot arriving in its
replacement.** `check-markers`' deleted shell half recorded that a `py` scope dropped from
one half was invisible because this tree holds no `.py` file; here, a class
widened to include DEL was invisible because nothing in the tree and nothing
planted carried that byte. ⭐ The repair is a FIXTURE and not a reading: a case
plants a DEL and requires all three to accept it. Re-measured with it, the same
plant is refused, exit 1.

⚠ **A fifth probe reported NOT-PLANTED and that is a harness result rather than a
finding.** The scope substitution was aimed at `markers.go` and the pattern lives
in `repo.go`; re-aimed, it is the second row above. A plant that did not apply is
a third status and is counted as neither.

### Residuals of step 1

- ⚠ **Nothing has run this on a runner.** The Go toolchain is assumed present on
  both hosted images. It is not asserted here: `check-bitcheck` calls
  `store_require go` and exits 2 without one, which `--strict` turns into a
  failure on the lane, so an absent toolchain says so rather than skipping.
- ⚠ **The harness is `sh` while its subject is a Go binary that runs on both
  platforms.** The Windows lane declares the row with that reason and names this
  entry as the event that closes it, rather than `CI-07`: a PowerShell twin of
  this file would be a new member of the layer this entry exists to delete.
### ⭐ Step 2 of 3: the twin layer is being deleted, four pairs on 2026-09-10

`check-changelog`, `check-control-bytes`, `check-markers` and `check-one-home`
are one Go binary. ⛔ **Both halves of each pair are gone** - eight files - and
the pairs are out of `check-twins`' list. The gate runs the same binary on both
lanes, so four rows that were an `sh` row here and a hand-written twin there are
now one row on each.

⚠ **They left the pair list only after `--compare` ran them against both deleted
halves**: 35 cases, 35 passed, agreeing on the exit code and byte for byte on
`--json`. A pair removed without that is a rule nobody checks.

⭐ **`check-twins` went from 69 seconds to 15.9.** Four pairs removed, and one of
them carried 67.9 seconds by itself.

### ⛔ THE SECTION BELOW IS MEASURED ON THE WRONG MACHINE, AND CI REFUTED IT

⚠ **It said the deletion bought nothing. On a runner it bought ten and a half
minutes.** Both numbers are real and they are measurements of different hosts:

| | run 122, twins present | run 123, twins deleted |
| --- | ---: | ---: |
| *Workflow acceptance* | **32.6 min** | **22.2 min** |
| Linux gate | 5.8 min | 4.7 min |
| Windows gate | 2.9 min | 2.8 min |

⛔ **This session host has FOUR processors and a hosted `ubuntu-24.04` runner has
two.** The gate runs its checks concurrently, so on four processors the batch
finishes well before the two socket harnesses that run after it and removing 96
seconds of PowerShell from the batch changes nothing at all; on two processors
the batch is CPU-bound, so the same 96 seconds come off the gate directly - nine
times over inside `check-workflow`, which is the 10.4 minutes.

⭐ **`docs/methodology/gate.md` already names this: *local is not production*.**
⚠ The failure here was not the measurement, which was correct, but the SCOPE
claimed for it: a wall clock read on one host was written up as a property of the
change. The section below is kept exactly as it was written, because a corrected
claim with the wrong reasoning deleted teaches nothing.

### ⛔ AND THE GATE DID NOT GET FASTER, WHICH REFUTES THIS ENTRY'S OWN PREMISE

⚠ **Measured immediately after the deletion: the whole gate is 129.7 seconds,
against the 118-125 `CI-01` recorded before any of this.** The concurrent batch
lost 53 seconds and the gate lost none of them, so the twin layer was not on the
critical path at all.

⛔ **Every gate member was then timed, and the wall clock is somewhere nobody had
looked:**

| check | seconds |
| --- | ---: |
| `check-capture-client` | **105.0** |
| `check-capture` | 45.1 |
| `check-step-bodies` | 17.9 |
| `check-twins` (after the deletion) | 16.3 |
| `check-shell` | 10.9 |
| everything else, each | under 4 |

⛔ **`CI-01` measured `check-capture-client` at 47.9 seconds and it is 105 now.**
Nothing changed its `SECS=5` deadline; it has **111 cases**, and it grew one case
at a time while the number in the record stayed still. ⚠ Those two harnesses run
AFTER the concurrent batch, alone, on purpose - `CI-01` records buying that on
purpose because a saturated host changes their ANSWER - so the gate is
`max(batch) + max(those two)`, and the second term is now four fifths of it.

⭐ **This is the entry's own lesson arriving against the entry.** `CI-01`'s
header says "CI is slow" is not a place to start optimising, and this entry's
Problem said the layer is slow "in a way no single row explains". One row
explains it, and it is not the row this entry named. ⚠ The port is still
unconditional and still correct - it removes the drift class, and `CI-07`'s
class-A backlog with it - but **it is not what makes CI faster**, and saying so
here is worth more than the 53 seconds.

⛔ **So the next unit is step 3, sharding, rather than more porting.**
`check-workflow` runs the whole gate about nine times, and a 130-second gate is
about twenty minutes of that job whatever language its rows are written in.
Dividing that across runners is the only thing that touches it.

### ⭐ Step 3 of 3: `check-workflow` shards, and the partition is checked

`--shard i/N` selects units by `index mod N`. The harness names **21 units**: 18
shardable and **3 controls that every shard runs**.

⛔ **The controls are not sharded and that is the whole design.** A shard carrying
only plants goes GREEN over a tree where every step is broken, because a plant
that is refused proves nothing unless the clean tree is accepted. ⚠ That is a
FLOOR on what a shard costs, paid on purpose, and it is why the numbers below are
not the unsharded time divided by four.

`scripts/ci/check-shards.sh` is the guard the entry's own residual asked for, and
it costs a few seconds because it uses `--units`, which lists and runs nothing:

| what it asserts | why it is not one of the others |
| --- | --- |
| coverage, for every N in 1..6 | a unit no shard claims is a rule that stops running while every lane stays green |
| disjointness | duplication is a wall-clock defect and hides in a green run exactly as well as a gap |
| every shard names every control | a control appearing once in the union satisfies coverage while running in one shard of six |
| no variable crosses a unit | a perfect partition still dies when two units land on different runners |
| six malformed selectors are refused | `--shard 0/4` and `--shard 5/4` each select nothing, which looks exactly like a fast green shard |

⛔ **Three real defects were found by driving it rather than by reading it**, and
each was in the sharding itself:

| defect | how it surfaced |
| --- | --- |
| `--units` ignored `--shard`, so every shard listed every unit | `check-shards` on its first run: 18 units "claimed by more than one shard" for N=2..6 |
| `PUBWF` assigned in `publisher-trigger`, read in `static-readers` | the first real shard exited 2 on `parameter not set` |
| `CAPWF`, a `for`-loop variable in `capture-fork`, read in `static-readers` | the second real shard, after the first was fixed |

⚠ **The coupling guard written for the second missed the third**, because its
regex matched `NAME=` at the start of a line and saw neither a `for` variable nor
an assignment inside a `case` branch. ⭐ A guard that sees one spelling of the
thing it forbids reports clean over the other two; it reads three now.
⛔ **And it was only ever LOUD because of `set -u`.** Without it an unset path is
an empty string, `[ -f "" ]` is false, and the case reports the workflow missing -
a plausible failure naming the wrong thing.

⭐ **Four shards driven end to end on this host, 2026-09-10, every one green:**

| shard | seconds | cases |
| --- | ---: | ---: |
| 1/4 | 592 | 36 |
| 2/4 | 606 | 33 |
| 3/4 | 476 | 18 |
| 4/4 | 605 | 49 |

⚠ **The spread is the control floor showing.** Shard 3 draws the cheap static
readers and still costs 476 seconds, which is very nearly what the other three
cost: almost all of a shard is the controls every shard pays.

⭐ **Against the unsharded run on the same host and tree, measured rather than
divided: 1282 seconds and 97 cases.** The longest shard is 606, so this is
**2.1x** and not 4x, and the control floor is the whole of the difference.

⛔ **AND THE CASE ARITHMETIC IS AN INDEPENDENT CHECK ON THE PARTITION.** The four
shards ran 36+33+18+49 = **136** cases where the unsharded run ran **97**. The
difference is **39**, which is exactly three extra copies of the **13** control
cases - the three shards beyond the first each re-running them. ⭐ So
`97 = 84 plants + 13 controls` and `136 = 84 plants + 4 x 13 controls`, with no
plant run twice and none dropped. ⚠ That is a check on the RUN, where
`check-shards` is a check on the SELECTOR, and neither is the other: a selector
can partition perfectly while a unit dies silently.

### ⭐ Step 3 proved on a runner, and the whole arc in three numbers

⭐ **Run 124 is green on every job with the matrix**, so the shard selector, the
partition check and the four-way split are measured on real runners rather than
argued for:

| run | what changed | *Workflow acceptance* |
| --- | --- | ---: |
| 122 | the layer as it stood | **32.6 min** |
| 123 | four twin pairs deleted | **22.2 min** |
| 124 | sharded across four runners | **11.8 min** (longest shard; 9.6, 11.2, 11.7, 11.8) |

⭐ **32.6 to 11.8 is 2.8x, and the whole run's wall clock is now that shard**: the
Linux gate is 4.4 minutes and the Windows gate 2.5. ⛔ **And the bound came DOWN
to 20 from the 45 it had been raised to**, which is the entry's Prove: *a bound
that was not raised to accommodate it.*

### ⭐ Two more pairs ported and deleted, 2026-09-10

`check-licences` and `check-placeholders` bring the binary to **six** checks and
`check-twins` to **eight** pairs. ⚠ Proved first: `--compare` over 50 cases, all
three implementations agreeing on the exit code and byte for byte on `--json`.

⛔ **`check-licences` had two CALLERS and both moved with it.** `check-cache`
asks `--permitted` which targets may be redistributed, and it asked it of an
`.sh` half here and a `.ps1` half there; both now ask the one binary, so the tie
between the cache and the register is a call rather than two readings.
⚠ **That comparison is weak and says so**: this register permits no target, so
old and new `--permitted` agree on an EMPTY list. Two empty sets agree perfectly.
What holds the mode up is `check-cache`'s own 13 cases, which pass on both
halves against the ported reader.

⛔ **AND A HARNESS THAT PLANTS A PATTERN CANNOT SPELL IT.** `check-bitcheck`
gained placeholder plants and turned the clean tree red: `check-placeholders`
reads every tracked file, this harness is one, and its literals were findings.
⭐ The needles are assembled with `printf` now, the way `check-markers`' harness
already built its marker bytes - and the check's own Go file is exempt from
itself, exactly as both shell halves already were.

### ⭐ `check-no-secrets` and `check-docs` ported, both halves of each deleted. 2026-09-15

The binary carries **nine** checks and `check-twins` is down to **five file pairs
and five rows**. ⚠ `check-no-secrets` was the two-row pair, because `--public` is
a different question rather than a stricter one.

⛔ **What a port of `check-no-secrets` loses silently is not the patterns.** Two
properties of the pipeline they sat in decide verdicts and both are carried: the
allow expressions run over the grep OUTPUT line `path:lineno:text`, because the
lockfile allowance is anchored to that prefix; and an allowed item is deleted
FROM the line rather than the line being dropped, because `grep -v` drops lines
and would take a real credential out of the report beside an allowed digest.
⚠ The home-path rule is the opposite shape, and that difference is inherited
rather than tidied.

⛔ **`check-docs` is the pair that justified the pre-deletion comparison.**
`--compare` found the two shell halves DISAGREEING: the `sh` half read links with
two awk programs and only the broken-link one stripped inline code spans, so a
page cited only inside backticks was an orphan to the PowerShell twin and not to
it. ⚠ No page in this tree is cited that way, so `check-twins` saw them agree for
as long as both existed - its own documented blind spot.

⭐ **The twin is correct** - a code span is not a hyperlink, so a reader following
links never arrives - and the `sh` half was fixed in the same change. That is a
verdict changed deliberately and recorded, which is what the port contract exists
to tell apart from one changed by accident.

⛔ **And the port reproduced a defect already written down here.**
[`../docs/conventions/forbidden-patterns.md`](../docs/conventions/forbidden-patterns.md)
records a GLOBAL replace collapsing `a/../` where `[^/]+` also matches `..`. The
first Go spelling was the same, and was green over the tree and fourteen cases,
because it is only wrong past two levels. ⭐ The hand-rolled collapse is deleted
and `path.Join` does it. ⚠ The case written to prove that cannot refute it,
measured by planting: `path.Join` normalises first, so the differing branch is
unreachable. It is kept as a regression guard with its label saying so.

Cases compared before either half was deleted: **20** for `check-no-secrets` and
**15** for `check-docs`, all three implementations agreeing on the exit code and
byte for byte on `--json`. ⚠ The last `check-docs` case was compared against
halves restored from `git show HEAD:`, because the collapse defect was found
after the deletion.

⛔ **The callers moved with them.** `check-defaults`' `secrets` subject moved
INSIDE the block that gates on the Go build rather than beside it - the
one-gated-door shape that block's own comment records - and `Invoke-Ported`
gained a row label separate from the check name, because one row is no longer one
check.

⚠ **A harness that plants a pattern cannot spell it, and neither can the comment
that says so.** The placeholder needle used the stand-in word
`check-placeholders` looks for, turning the harness into that rule's finding; the
comment explaining it then did the same thing again.

⛔ **AND THE PORT SHIPPED A WINDOWS-ONLY DEFECT THAT ONLY CI COULD SEE.** It used
`filepath.Dir` - the HOST's separator - to derive a repo-relative directory, while
every path here comes from `git ls-files` as forward slashes and the orphan set is
keyed on that spelling. On Windows it answered `docs\methodology`, so every joined
key missed and run 132's Windows gate reported **every page an orphan**.

⚠ **Green on the Linux lane, and green on `check-gate.ps1` run on Linux.** Go's
`filepath` separator is the host's, so driving the PowerShell lane here does not
test Windows - which is the *local is not production* rule arriving in a place
this project had not met it. ⭐ `path.Dir` is slash-based and is what the map keys
need; `filepath.Join` stays where the code touches the filesystem, because that
one must be the host's.

### ⭐ Residual closed: a gate row could ask the wrong question and stay green

The gate runs `check-no-secrets` twice and the second row exists only because
`--public` is a different question. ⛔ **If either lane stopped passing that flag
the row would still be green**, because the default question also passes on a
clean tree. `check-gate-rows` compares row NAMES; `check-bitcheck` proves the
BINARY honours the flag and says nothing about what the runner passes.

⚠ Not new and not caused by the port: each lane used to spell its own flag, with
the same silence. What the port changed is that there is one spelling now, so a
check could assert it once rather than twice.

⭐ **[`../scripts/common/check-public-row.sh`](../scripts/common/check-public-row.sh)
is that check, 2026-09-15**, and it is a gate row on both lanes - a real row on
the `sh` one and a declared `n/a` on the PowerShell one, because it reads both
runners itself and a second implementation would answer the same thing from the
same two files. The gate is **39 checks** now, `check-adapters` having joined
it on 2026-09-15.

⛔ **The flags are READ OUT OF the runners and then RUN.** A harness that spelled
`--public` itself could not catch the defect being guarded against, which is a
runner that stopped passing it - the rule `check-workflow.sh` is built on,
applied to the gate's own queue. The verdict is the binary's own
`"public_rules"` field rather than the text of a line in a script.

⚠ **Its control is the half that matters**: the default invocation must answer
`public_rules:false`, or the field is not one that moves and the assertion would
hold over a runner passing nothing at all.

Guard mutation, four plants into the runners, each verified to have changed the
file, clean control either side. All four refused:

| plant | what it reported |
| --- | --- |
| the `sh` runner stops passing `--public` | names no flags for that row |
| the PowerShell runner stops passing `--public` | names no flags for that row |
| the row is renamed in the `sh` runner | names no flags for that row |
| the PowerShell runner passes `--all-history` instead | ran it and got `"public_rules":false` |

⛔ **AND ITS FIRST DRAFT BUILT TWO `awk` PROGRAMS OUT OF SHELL STRINGS**, which
is [`../docs/conventions/shell.md`](../docs/conventions/shell.md) section 1
exactly: one half died with `awk: syntax error` and the other matched the row's
LABEL rather than its invocation, reporting the flags as
`(public)" "$GOBIN" check-no-secrets --public`. ⚠ Both halves read with parameter
expansion now, which crosses no boundary.

⛔ **AND EDITING THE `.ps1` WITH A TEXT-MODE READ COLLAPSED ITS LINE ENDINGS.**
`git ls-files --eol` reported `w/mixed` under `attr/text eol=crlf` and `git diff`
showed nothing, which is the same document's section 5. The edit was redone on
bytes; `check-project`'s line-ending row is what would have caught it.

### The three deep reviews, 2026-09-10

⛔ **The door sweep found the one-gated-door defect, in this entry's own change.**
`check-defaults` builds the Go binary for its `markers` subject and the condition
was around that row alone; `licences` sat outside it and would have run a binary
that is not there. ⚠ A missing command answers 127 under every environment, so
`run_subject` would have reported *the same answer under all 6* - a green row over
a subject that never started. ⭐ The build is gated once, in front of both rows,
and the else branch names both.

⛔ **The guard mutation pass found a guard nothing can refute.** Four plants into
the two new checks, each verified to have changed the file before it was judged:

| plant | verdict |
| --- | --- |
| the "permitted must be earned" rule removed | ⭐ refused, exit 1 |
| the markdown exemption on the owner generic dropped | ⭐ refused, exit 1 |
| the Go-template exclusion dropped | ⭐ refused, exit 1 |
| **the empty-register refusal removed** | ⛔ **SURVIVED, exit 0** |

⚠ **The survivor is not a gap and it is not a pass either.** The plant is still
refused - a register with no rows has no row for any catalogue target, so rule 1
fires - which makes that case *a check that passes because a different code path
happens to satisfy it*, one of the two shapes
[`../docs/methodology/reviews.md`](../docs/methodology/reviews.md) names. ⭐ The
guard is kept and the CASE LABEL is corrected to say which rule refuses it,
rather than claiming a proof it does not carry.

⚠ **And one plant did not compile**, which is a third status and is counted as
neither: dropping the markdown test left a variable unused, the build failed, and
the harness answered 2. ⭐ Re-aimed in a form that compiles, it is the second row
above. A harness exit of 2 is *could not run*, never *refused*.

⭐ **The claim audit re-read every number here against the tree**, and each one
holds: six ported checks (`--rows`), eight remaining pairs (`check-twins`),
eighteen shardable units and three controls (`--units`), no `go.sum`, none of the
twelve deleted files on disk, and every `.ps1` at `w/crlf`. ⚠ What would have made
it fire is exactly what it found earlier in the session - the run-122 comparison
written up as a property of the change rather than of the host.

### Residuals of step 3

- ⚠ **The CI matrix is four shards and the bound came DOWN to 20 from 45.** That
  is the point rather than a detail: the bound was raised to fit growth on
  2026-09-10, which this repository calls masking everywhere else. ⛔ Unproved on
  a runner at the time of writing - the shards are driven here, the matrix is not.
- ⚠ **The floor, not the divisor, is what limits this.** Almost all of a shard is
  the controls, so N=8 would not be twice as fast as N=4. Cutting the floor means
  either fewer control gate runs per shard or a faster gate, and the first trades
  away the thing that keeps a shard honest.
- ⚠ **`check-shards` is a static read and says so.** It matches upper-case names
  by text, so a variable name built at runtime is outside what it can see.
- ⛔ **`--units` still copies the working tree**, which it does not need: it lists
  and runs nothing. That is why `check-shards` costs seconds rather than
  milliseconds. Left as it is because the cost is paid once per shard in parallel.

### Residuals of step 2

- ⚠ **`check-capture-client` at 105 seconds is unowned by this entry.** It is a
  harness that grew, not a defect, and shrinking it is a decision about how many
  cases a gate should carry rather than a port. Recorded here because the
  measurement was taken here; it belongs to whoever next opens `CI-01`.
- ⚠ **No file pair remains and `check-twins` prints no pair row**, 2026-09-29:
  `check-cache`, `check-catalogue`, `check-remote-items` and `mine-repo` all left
  that day. This said *four* from 2026-09-17, when `check-project` left, through
  the first two deletions of that session, and *one* for the next two commits.
  The five that remained before `check-project` left together took
  **5.3 seconds**, timed on 2026-09-15, against the 96.1 the layer started at,
  and `check-project.ps1` was **2.60** of those five. ⛔ The figure had been decremented from 12.3 as
  pairs left rather than re-timed, which is the value-in-two-places defect in a
  different costume. ⛔ The
  remaining wall-clock value of porting them is therefore small and the DRIFT
  value is unchanged, which is the honest ordering argument for doing them after
  sharding rather than before.
  ⛔ **SAY WHICH UNIT A COUNT IS IN.** This residual once said seven pairs and
  `docs/history/RESUME.md` said eight while naming seven files; both were real,
  because `check-no-secrets` was compared TWICE and seven pairs of files produced
  eight rows. ⚠ Reconciled by running `sh scripts/common/check-twins.sh` rather
  than by picking one. It is **four pairs and four rows** now, and the two are
  equal only because the two-row pair has gone.
  ⚠ **Amended 2026-09-29**: `check-cache` has left too, so `check-twins.sh`
  compares three pairs as three rows, which
  `grep -c '^compare_pair ' scripts/common/check-twins.sh` counts. The numbers
  above are what was true on the dates they carry.
- ⚠ **`check-defaults` runs the Go binary as a subject now.** A Go program
  inherits a different set of host values than a shell script - no `IFS`, and
  `TMPDIR` through the runtime rather than a shell expansion - so that row is
  asking the same question of a subject with its own answers.

### ⭐ `check-project` ported, and the comparison found three drifts. 2026-09-17

⛔ **THE BIG ONE.** 997 lines of `sh` and a hand-written PowerShell twin beside
it, holding **twenty-nine refusals** over this repository's own invariants: the
files a reader is promised, the catalogue against the client matrix in both
directions,
the TODO bookkeeping in six places, the Python exception, the action pins, the
artifact names, the line endings, the shfmt pin, `set -u`, the cargo output path,
the lockfile, the acceptance commands, the manifests, three `.ps1` rules and the
same three again inside a workflow's `pwsh` blocks.

⭐ **`bit-check check-project` is all of them**, and the binary carries **eleven**
checks now. ⚠ The count is the greppable one rather than a count of *rules*,
which is a judgement: `f.say(`, `say_fail "` and `failures.Add(` each appear
twenty-nine times, in the port, the `sh` half and the twin. ⚠ Nothing is deleted yet and no gate row has moved: this change is
the COMPARISON, which is the only way a pair may leave `check-twins`' list.

**Measured on this host**: the `sh` half 0.72s, the PowerShell twin **2.60s**,
the Go check **0.18s**. The twin is half of what is left of `check-twins`' 5.3
seconds.

#### ⛔ `--compare` found three drifts, and the twin was wrong in all three

49 cases were written, one per rule plus the accept cases a harness of refusals
never looks at. The first three-way run reported **three disagreements**, all of
them `go and pwsh` - the Go port and the `sh` half agreeing, the twin dissenting:

| planted | the twin |
| --- | --- |
| a client matrix with no rows | **CRASHED**: `The property 'Count' cannot be found on this object` |
| the Total row's open count changed | refused for a second reason as well |
| an index row duplicated | reported **9** failures where the other two reported 10 |

1. ⛔ **`@(...) | Sort-Object -Unique` WRAPS THE INPUT AND LEAVES THE OUTPUT
   UNWRAPPED.** An empty set therefore arrived as `$null`, and `.Count` threw -
   turning the refusal *a result too small to be real* exists for into a crash a
   caller reads as could-not-run. ⚠ The rule was written precisely because two
   empty sets agree perfectly; the twin could not reach it.
2. ⛔ **The Total row was compared as a whole LINE** against a reconstructed
   string, where the `sh` half compares the five counts. One rule with two
   meanings: anything else in that row - a changed prefix cell, a different run
   of spaces - was refused on one lane and ignored on the other.
3. ⛔ **Two checks joined by `-or` are one check**, which is this repository's own
   rule arriving inside a check. The `sh` half says *the body count does not
   match* and *the fields disagree* separately; the twin folded them, so a tree
   failing both counted one failure there and two here.

⭐ All three are fixed in the twin rather than absorbed by the port, and only then
did the run come back clean. ⚠ The Windows gate lane runs that file today, so
this is a repair of a live lane rather than work thrown away before a deletion.

#### ⛔ And the harness could not report its own failures

⛔ **The first `--compare` run printed NO ROWS AT ALL**, only
`check-bitcheck: 149 rows recorded, 143 counted; the report does not describe
itself`, and returned 1. `store_report`'s self-check counted LINES, and a row may
be several: a disagreement is the label plus the two JSON lines that differ. So
three real failures made the list six lines longer than the count, the self-check
fired, and the one run with something to say said nothing.

⭐ **It counts ROW STARTS now**, in both halves of `store-lib` - `row`'s own
two-space prefix followed by a non-space, where a continuation is indented
further. ⚠ The check itself is right and is kept: its own record says an
accumulator was once overwritten and the summary went on claiming a count the
rows did not support.

#### ⚠ Two more differences the comparison cannot settle, recorded rather than hidden

- **The TODO entry bodies.** `awk` carries the current entry id ACROSS files and
  the twin resets it per file, so a heading left unterminated at the end of one
  file would pair with the next file's first `Priority:` line in one
  implementation and not the other. ⭐ The port resets, with the twin: pairing
  across files is an answer nobody wrote down. No file here ends that way, so no
  comparison over this tree can see it.
- **The `Priority:` line itself.** The twin's pattern is anchored and strict; the
  `sh` half's is loose. A malformed line is DROPPED by the twin and read by the
  other two. ⭐ The port is loose, because dropping is the direction that keeps a
  check green.

#### ⚠ What the port had to reproduce rather than tidy

- **`grep -n` prints no path over a SINGLE file**, and the manifest rule pipes
  exactly that through `xargs`. A port that always prefixed would differ from
  both halves on a tree with one manifest.
- **A command substitution strips trailing newlines**, so the pin rule
  concatenates one workflow's findings onto the previous one's last line.
- **The `with:` column is the KEY's, not the line's.** In `- with:` the dash is
  part of the indent, so the step's other keys sit at the key's column.
- **An indent is measured in SPACES only** in the `pwsh`-block reader, so a
  tab-led line reads as column 0.
- **`case "$subject" in $pattern)` is sh's globbing**, where `*` crosses a slash.
  `path.Match` is a different rule, so the class is implemented rather than the
  instances this tree happens to have.

#### ⛔ And the harness became the thing it refuses, twice

`check-bitcheck.sh` is under `scripts/`, which is exactly the scope two of these
rules read. Written whole, the plant strings `mvdan.cc/sh/v3/cmd/shfmt@v0.0.1`
and `target/debug/examples` made the clean tree fail: a harness that plants a
pattern cannot spell it. ⚠ A third, a bare forty-character hex pin fixture, is
what `check-no-secrets --public` refuses and is right to. All three are assembled
from parts.

#### ⛔ Two defects in the harness itself, both found by this

- **A case overwrote a REAL tracked file and `unplant`ed it**, and `unplant` is
  `rm`. The tree then carried a path git lists with no bytes behind it - harmless
  to every other rule and reported by `check-project`'s line-ending row the
  moment it ran. It is saved and put back now.
- **A plant rewritten three times could not be unplanted.** `git rm --cached`
  refuses a path whose staged content matches neither HEAD nor the working tree,
  so the file survived into every case after it. Each rewrite re-stages.

### ⭐ `check-cache` ported and compared: the first HARNESS. 2026-09-29

`bit-check check-cache` carries the harness's thirteen rows, and
[`../tools/check/harness.go`](../tools/check/harness.go) carries what it needed of
`store-lib`: rows, a report that checks itself, an example build, a scratch
directory and the literal plant with its six probes. ⚠ Nothing is deleted and no
gate row has moved; this change is the COMPARISON.

⛔ **A HARNESS PLANTS INTO WHAT IT READS, SO ITS CASES DO TOO.** Over the real
tree every row passes, so its refusal branches are exercised by nothing but a
plant: a register permitting the scenario's own target, one permitting another,
a register that is missing, and a scenario whose second retrieval repeats the
first. Six cases in `check-bitcheck.sh`, run against all three implementations:

| case | all three answered |
| --- | --- |
| the clean tree, either side | exit 0, `13` passed |
| the register permits `aria2`, the scenario's target | exit 1, `10` passed and `3` failed |
| the register permits `aria2-next` | exit 1, `12` passed and `1` failed |
| the register is missing | exit 2 |
| the second retrieval repeats the first | exit 1, `7` passed and `6` failed |

⭐ **THE COMPARISON FOUND ONE DRIFT AND IT WAS THE TWIN'S.** Its could-not-run
message named `-Permitted`, a flag the call it describes does not use, where the
`sh` half and the port say `--permitted`. Repaired in the twin, byte for byte
with its CRLF kept, and only then did the run come back clean.

⛔ **AN EXIT CODE OF 1 DOES NOT SAY WHICH ROW FAILED.** A harness answers 1 for one
failed row and for six, so `go_said` asserts the verdict line the Go binary gave
for each case. ⭐ The mutation pass is what proved it earns its place: three of
the plants below were caught by that line ALONE, every exit code unchanged.

⚠ **A PLANT DID NOT PRODUCE ITS DEFECT ON THE FIRST ATTEMPT.** Moving the second
retrieval's location changed nothing, because a retrieval is its location and
its time; the case went green and said so. It repeats both now.

#### Guard mutation over the port

Seven plants into the Go files, one at a time, each verified to have changed the
file, restored from a scratch copy after each:

| plant | verdict |
| --- | --- |
| the register row inverted | refused |
| the E-CAC-01 row always passes | refused, by the verdict line alone |
| the diff count always 2 | refused, by the verdict line alone |
| a literal occurring twice accepted by the plant function | refused |
| identity needing one of its two facts | refused, by the verdict line alone |
| the control permitting the wrong target | refused |
| a run that passed nothing reported green | ⛔ **survived** |

⚠ **The survivor is a guard nothing here can refute and it is kept.** The six
probe rows always pass, so no plant through this harness produces a run with no
passes; the guard is the shell library's, carried across, and the label says what
it is rather than claiming a proof. ⚠ And the diff plant's first spelling did not
compile - an unused variable - so it exited 2, which is could-not-run and was
counted as neither; re-aimed, it is the row above.

#### ⛔ Why the harness cases are not in the default mode, measured

The cases build `cache-scenario` out of the scratch tree, and the build is COLD:
the scratch tree needs a target directory of its own, because sharing a caller's
would let a planted scenario be uplifted over a binary another gate row is
executing. Measured on this host:

| | before | with the cases in the default mode |
| --- | ---: | ---: |
| `check-bitcheck` | 23s | 39s |
| the whole gate, same host, same tree otherwise | 168s | 193s |

⛔ `check-workflow` runs the gate about nine times per shard, so that is minutes
per shard for plants that do not change between pushes - `check-defaults`'
reason for being a step of its own. ⭐ So `--harnesses` runs them, `--compare`
runs them, and the default mode says on stderr that it did not rather than
passing a row for it.

#### What the port changed on purpose, and what it could not settle

- ⭐ **The example's path is ASKED OF CARGO**, from `--message-format=json`, where
  the `sh` half composed it out of `CARGO_TARGET_DIR` or the tree's `target`.
  That composition is `CI-01`'s old hole, and asking removes the class rather
  than guarding it; a configured target directory is the one input where the two
  would differ, and this tree configures none.
- ⭐ **The register is read by the one reader.** `readRegister` and
  `permittedTargets` are what `check-licences` itself calls, so `check-cache`
  asks a function where it used to build a binary and ask it for `--permitted`.
  ⚠ Moving that code was checked rather than trusted: the binary built from
  `HEAD` and the one built from this change gave identical output and exit codes
  over eighteen runs - three modes over the clean register and four plants.
- ⚠ **The two halves counted the difference between two runs two ways.** The
  `sh` half asks `diff` and the twin asked `Compare-Object -SyncWindow 0`, which
  compares by position; they agree on every case here and would part over a
  reordered line, which counts 2 to `diff` and 4 by position. ⭐ The port answers
  `diff`'s question, over a longest common subsequence, because that half is the
  one whose argument each case carries.

#### Residuals

- ⚠ **Nothing runs the harness cases automatically.** They cost too much for the
  gate and nothing else invokes `--harnesses`; a CI step of their own, the way
  `check-defaults` has one, would close it and is a workflow change.
  ⭐ **Closed 2026-10-11** by the Linux lane's *Checking harnesses* step; see the
  last section of this entry.
- ⛔ **CI runs no `gofmt`, no `go vet` and no `go test` over `tools/check`**,
  found by the door sweep over this change. Earlier commits in this entry say
  *gofmt and go vet clean*, which was a session running them by hand: a rule a
  commit message says this repository has is not a rule it has.
  ⭐ **Closed 2026-10-11** for `gofmt` and `go vet` by the Linux lane's *Go
  formatting and vet* step. ⚠ `go test` has nothing to run: the module carries
  no `_test.go`, and its cases are `check-bitcheck`'s.

#### ⛔ And both halves are deleted, 2026-09-29, after the comparison

The comparison was committed first, as `621f93a`, with the twin's repair in it,
so the history carries a repaired twin before it carries a deleted one. ⚠ The
deletion was not two files:

| door | what changed |
| --- | --- |
| `check-gate.sh` | the row left the `sh` prover list for the ported list |
| `check-gate.ps1` | the `pwsh -File check-cache.ps1` block became `Invoke-Ported 'check-cache'` |
| `check-twins.sh` | the pair is gone, with a note saying how it left |
| `check-defaults.sh` | the `cache` subject moved INSIDE the block that gates on the Go build |
| `scripts/README.md`, `docs/architecture.md`, the scenario's own doc comment, `ACQ-05` | each named the deleted script or its `--permitted` call |

⛔ **Moving `cache` below the build gate would have been the defect that block's
own comment records**, a third time: a host where the build failed runs a missing
command, which answers 127 under every environment and reads as a subject
answering identically everywhere.

⚠ **`--permitted` has no caller in this tree now.** It was how `check-cache`
asked the register; in one binary that is a function call. The mode is kept as
the command a person runs to ask what the register permits, and its comment says
so rather than naming a caller that is gone.

⛔ **THE CLAIM AUDIT FOUND A RUNNABLE FORM THAT DOES NOT RUN.** `CI-05`'s amended
Prove, written on 2026-09-17, gives `go run ./tools/check check-project` *from the
repository root*, and this change first copied that form into `ACQ-05`. Measured:
it exits 1 with *cannot find main module*, because the module is `tools/check`.
`go run . <check>` from that directory exits 0; both places carry that now, with
`CI-05`'s correction written under its original sentence.

### ⭐ `check-catalogue` ported and compared, the second harness. 2026-09-29

`bit-check check-catalogue` carries its eighteen rows, using only what
`harness.go` already had - the second measurement of the library's shape, after
`CI-07` measured the same thing once in PowerShell. ⚠ Nothing is deleted and no
gate row has moved; this change is the COMPARISON.

⛔ **THE PLANTS GO WHERE THE HARNESS READS**, which here is three places: the crate
it sweeps for sockets, the manifest it reads for a transport, and the library it
drives. Seven cases, all three implementations agreeing on every one:

| case | all three answered |
| --- | --- |
| the clean tree, either side | exit 0, `18` passed |
| a socket named in a crate file nothing compiles | exit 1, `17` and `1` |
| the same needle spelled in another case | exit 0 |
| a transport line in the crate's manifest, under a table cargo ignores | exit 1, `17` and `1` |
| the same line with the name capitalised | exit 0 |
| both `E-LIB-02` sites disabled in the library | exit 1, `17` and `1` |

⛔ **THE COMPARISON FOUND TWO DRIFTS AND BOTH WERE THE TWIN'S.** `Select-String`
and `-match` are case-INSENSITIVE by default, where the `sh` half's `grep` is
not, so the twin refused a lower-case `tcpstream` in a comment and a `Tokio` key
that name nothing - Rust tells the two spellings apart, and the needles are Rust
paths. ⚠ **A clean tree cannot see this**: every needle in it is spelled one way,
so the halves agreed for as long as both existed. Repaired with
`-CaseSensitive` at both sweeps and `-cmatch` on the manifest, CRLF kept, and
only then did the run come back clean.

⛔ **A CASE WAS NAMED FOR SOMETHING IT DID NOT DO, AND THE MUTATION PASS SAID SO.**
The library plant was first called *a library that accepts a moved record*.
Disabling the harness's own code comparison turned it green, and driving the
planted library by hand showed why: the record is still REFUSED, as `E-LIB-04` -
the checksum file no longer covers the bytes - so the row had only ever failed
on *refused, but not as E-LIB-02*. ⭐ It is renamed for that and kept, because
it is the one case that proves the harness reads the CODE and not only the exit.

#### Guard mutation over the port

| plant | verdict |
| --- | --- |
| the ordering row reads `1.2.3` | refused |
| the absent row ignores what the lookup printed | ⛔ **survived** |
| a plant is never restored | refused |
| two needles dropped from the list | refused |
| the sweep folds case | refused, by the lower-case plant alone |
| the manifest check folds case | refused, by the capitalised plant alone |
| a refusal under any code accepted | refused, by the relabelled case alone |

⚠ **The survivor is equivalent on this fixture and is recorded rather than
chased.** The unmeasured build line prints nothing, so no plant through this
harness separates "exit 1 and no rows" from "exit 1"; a library that answered the
nearest line would, and none here does.

#### What the halves did differently that no case separates

- ⚠ The twin chose the first record with `Sort-Object`, which is culture-aware,
  where `sh` sorts under `LC_ALL=C`; the port sorts bytes. Any record serves the
  plant, so the three can pick different files and give one verdict.
- ⚠ The twin appended with `Add-Content`, which writes CRLF on Windows, where the
  `sh` half appends one newline. The bytes moved either way.
- ⚠ `sed` without `g` replaces the first occurrence on each LINE and the twin's
  `.Replace` replaced every one; the port does what `sed` does. The index carries
  its generation once.

#### ⛔ And both halves are deleted, and a library with them. 2026-09-29

`c40784f` committed the comparison with the twin's repairs in it; the deletion
came after. ⚠ It took one more file than it names: **`store-lib.ps1` had these
two harness twins as its only callers**, so it went with them rather than
staying as a library nothing calls. The doors the deletion touched:

| door | what changed |
| --- | --- |
| `check-gate.sh` | the row left the `sh` prover list for the ported list |
| `check-gate.ps1` | the `pwsh -File` block became `Invoke-Ported 'check-catalogue'`, and seven declared rows' reasons stopped naming `store-lib.ps1` |
| `check-twins.sh` | the pair is gone, with a note saying how it left |
| `scripts/README.md`, `LIB-01`, `CI-07` | each named the deleted script or library; `LIB-01`'s `Prove` gains a runnable form under a dated note |

⚠ **Nothing ran the catalogue harness as a `check-defaults` subject**, so there
was no fourth door there, which a grep over that file established rather than a
reading of its header.

### ⭐ `check-remote-items` ported and compared, and it runs everywhere now. 2026-09-29

⛔ **THE GATE'S ONE SKIP HAD NEVER RUN ON A SESSION HOST**, because it asks an
authenticated `gh`, and so its logic was proved only by whatever items happened
to be open when a CI lane ran it. ⭐ `check-bitcheck` now serves it from a stub
`gh` and a stub `curl`, first on `PATH`, answering from files: the issues, the
pull requests, a diff, the API's view of a commit, a tag and a release, and the
runtime a pinned commit declares. **Fifteen cases, in the DEFAULT mode, with no
network at all** - so the gate row runs them on every host, `gh` or not.

⚠ **The pins are real where the runtime matters.** `actions/checkout` at its
v4.2.2 commit declares `node20` and at v5.0.0 `node24`, and `astral-sh/setup-uv`
at v10.2.0 declares `"node24"` quoted - read on 2026-09-29. Under `--compare` the
PowerShell half fetches with `Invoke-WebRequest`, which no stub intercepts, so it
read the REAL bytes at those commits while the other two read the copies, and
all three agreed; a fake commit is a 404 on both routes.

⛔ **THE COMPARISON NOW COVERS THE WHOLE REPORT, and it found two drifts, both the
twin's.** This check keeps a report for a person on stderr in `--json` mode, so
`agree` compared every line of it: the twin said *which the platform has
deprecated* where the `sh` half says *which GitHub has*, and on a failed listing
it discarded gh's own error where the `sh` half prints it. Both repaired in the
twin, CRLF kept, before the clean run.

#### ⛔ Two defects in the harness, found by running it

- **A multi-line answer broke the report's own accounting.** `go_said` compared
  the whole combined output, and `agree` printed both answers inline on a
  disagreement; with a report in them, report lines landed where `store_report`
  counts row starts, and it refused to print anything - *202 rows recorded, 167
  counted*. `go_said` reads the last line now, and a disagreement is a `diff -u`
  indented under its row, which is also the readable form.
- **The dispatcher could not print a report before a refusal.** Both halves
  print the `OPEN ISSUES` header before a listing fails, so the port must too:
  `verdict.jsonReport` goes to stderr in `--json` mode, and a partial report now
  precedes the reason it stopped, in both modes. ⚠ Every existing check's output
  is unchanged by that, because none sets a report beside a refusal.

#### Guard mutation over the port

| plant | verdict |
| --- | --- |
| `node20` no longer deprecated | refused |
| quotes left on the declared runtime | refused, three cases |
| an annotated tag not followed to its commit | refused |
| the commit never looked up | refused, by the verdict line alone |
| removed lines read as proposed | refused, by a case ADDED for it |
| an open issue asking for no reading | refused, by the verdict line alone |
| an unauthenticated `gh` used anyway | refused |
| only the first pin on a line | refused, by a case ADDED for it |

⚠ **Two cases exist because a plant survived without them.** No diff removed a
line, and no line carried two pins - which YAML's flow style allows - so the
plants that read removed lines and read one match per line both passed every
case. ⚠ And the last plant's first spelling appended an empty match per line and
PANICKED, exiting 2: that is the mutant failing on its own nil path, counted as
neither, and the spelling that cannot crash is the row above.

#### ⛔ And both halves are deleted, and a function with them. 2026-09-29

`f0e4f7f` committed the comparison with the twin's repairs in it, and CI run 174
read it back green on all six jobs; the deletion came after. ⚠ **Both lanes now
run the binary against the real open items**, where until this commit each ran
its own half. The doors the deletion touched:

| door | what changed |
| --- | --- |
| `check-gate.sh` | the row left its `sh` call for the ported list, keeping its note that it needs `gh` and the network |
| `check-gate.ps1` | the row became `Invoke-Ported 'check-remote-items'`, and `Invoke-Check` went with its last caller; its two lessons moved into `Invoke-Ported`, and so did its skip reason - a ported row that could not run said only *could not run*, which would have cost this row its *gh not found*, and now prints the check's first line as the `sh` lane does |
| `check-twins.sh` | the pair is gone, with a note saying how it left; `mine-repo` is the one pair left |
| `check-bitcheck.sh` | the section's note on the PowerShell half's real fetches is past tense, and the header says what `--compare` compares: stderr as well as stdout |
| `docs/supply-chain.md`, `docs/security/remote-ops.md` | each linked or ran the deleted script, and now names the Go file |

⛔ **THE PAGE THAT TELLS AN OPERATOR WHAT THE CHECK VERIFIES CLAIMED ONE THING IT
NEVER DID.** Step 1 of `docs/supply-chain.md`'s update procedure said the check
verifies *that the tag is a published release*. The `sh` half's header said so
too, and its code never asked it; the step now says what the code does ask
about releases. That page already carried a warning that it described the check
from its own documentation with no run behind it; fifteen cases now back each
sentence of the step, and the sentence no case could back is gone rather than
tested.

⚠ **Two comments beside the row were false for a reason other than this pair.**
Both runners said *every `common/` half is ported* from 2026-09-17, while this
row still ran from its halves; each now says what the gate RUNS is ported and
names `mine-repo` as the pair left. And `check-gate.ps1` cited an
`Invoke-Native` for its unpiped exit codes, which no commit in this repository
ever defined - `git log -S 'function Invoke-Native'` over the full history finds
nothing - so it now names the function that does the work.

⚠ **A third was false about a different row.** `check-gate.sh` said `check-runner`
is not in `check-twins`' pair list *because it has no PowerShell half*;
`check-runner.ps1` has existed since `CI-03` closed. The true reason is that the
two prove different guards - the Linux ones and the Windows ones - so no one
verdict holds both.

⚠ **THE RUNNABLE FORM THIS SESSION WROTE HIDES ONE CODE.**
`go run . <check>` from `tools/check` is what `CI-05`, `ACQ-05`, `LIB-01` and
now the update procedure give, and it is right about exit 0. Measured over this
check on a host with no `gh`: the binary printed *gh not found*, `go run`
printed *exit status 2* and then exited **1** - so through it, a refusal and a
could-not-run are one code. The three `Prove` forms state exit 0, which it
reports faithfully; the update procedure says to read the report, and the
binary's usage comment says why.

### ⭐ `mine-repo` ported and compared: a helper, and the last pair. 2026-09-29

⚠ **IT IS NOT A CHECK, AND IT CAME INTO THE CHECKING BINARY ANYWAY.** It
writes a corpus - the API's answers, a clone and `PROVENANCE.md` - so it is no
gate row, `--rows` does not name it, and `bit-check mine-repo` is dispatched
before the checks are read. It came here because it was the last pair
`check-twins` compared, and this entry's direction is that the twin layer is
deleted rather than translated.

⛔ **`--selftest` WAS THE WHOLE COMPARISON THE PAIR EVER HAD, AND IT COVERS FOUR
CASES OF ONE FUNCTION.** Paging, both routes, the control, the clone, the trim
and the provenance had never been run against each other. `check-bitcheck` now
serves all of them from stubs first on `PATH` - `gh`, `curl` and `git` - and
compares what each implementation LEFT: the exit code, what `--json` printed,
and every file and directory it wrote, JSON through `jq -S`, the tree by
checksum, `PROVENANCE.md` whole but for the line naming its writer and moment.

⭐ **The stubs enforce what the halves only said.** The proxy stub answers 420
to any user-agent but curl's own, as the real proxy was measured to; the git
stub refuses a clone without the low-speed bound or a depth of one, and any URL
but the target's. ⚠ The proxy route is compared between the port and the `sh`
half alone: the twin fetches with `Invoke-WebRequest`, which no stub on `PATH`
intercepts, and the rows say so.

#### ⛔ Six drifts, and only running the three against each other found the last two

| drift | whose | found by |
| --- | --- | --- |
| a target with two slashes, `a/b/c`, was mined into `a__b/c` | `sh` | running both halves before the port |
| of two targets, the last was mined and the first dropped in silence | `sh` | the same |
| an unknown route was read as `auto`, so `--route prxy` - asked for to keep the operator's token out of it - probed gh and used the token | `sh` | the same |
| an unknown argument, a second target and a bad route were binder errors, which pwsh answers with exit 1 where the vocabulary says 2 | twin | the same |
| `.next` and `.venv` were never trimmed on Linux: pwsh treats a leading dot as hidden, and `Get-ChildItem` skips hidden items without `-Force` | twin | `check-bitcheck --compare`, five cases |
| the proxy route left `api/.control.json` in the corpus: the same hidden name, and `Remove-Item` refuses a hidden item without `-Force`, silently under `SilentlyContinue` | twin | a real fetch, below; no stub reaches this path |

All six are repaired in the half that had them, CRLF and BOM kept, and the
comparison then came back clean, as did the real fetch.

#### ⭐ What the port does that neither half did

- ⛔ **A list cut off at the page cap is a named gap.** Both halves stopped the
  proxy route after ten pages of a hundred and said `ok`, so a tracker with
  more than a thousand items arrived short with nothing anywhere saying so.
- ⚠ **`vendor/bundle` is trimmed.** The `sh` half listed it where `find -name`
  takes a NAME, which never matches a slash, and the twin left it out; neither
  ever removed one.
- ⚠ **A relative `--out` is under the repository root** when it runs inside one.
  The halves resolved it from the working directory, and `go run .` starts in
  `tools/check`.
- **No python3 or node.** The halves needed one of them to join pages and
  probed each by running it; the port parses with the standard library.

The first two are cases marked `-` in `check-bitcheck`, with a third for an
`--out` given no value: a rule the port has and no half ever did, which is a
different fact from a half that agreed. ⚠ The relative `--out` has no case,
because every case passes an absolute one.

#### ⭐ Measured once against the real proxy and a real clone

`pkgforge-dev/cross-libc-dlopen`, the target the twin's own header recorded,
through `api.gh.pkgforge.dev` with read-only requests. Without the clone, the
three wrote the same corpus - 40 issues and pull requests, 26 comments, no
review comments, 9 releases and 9 tags - once the `.control.json` repair was
in; before it, the twin's corpus held one file more. With the clone, all three
wrote the same 169 files and directories at commit `3095c15`, `.git` gone.

#### Guard mutation over the port

| plant | verdict |
| --- | --- |
| the proxy user-agent is a browser's | refused, 16 rows |
| the clone loses its low-speed bound | refused, 10 rows |
| the clone loses its depth | refused, 10 rows |
| a full page is taken for the last | refused, 3 rows |
| the empty-join guard never fires | refused, by the self-test |
| each page is kept as one record | refused, 3 rows - and NOT by the self-test |
| any slashed target is taken | refused |
| an unknown route is taken | refused, 2 rows |
| a second target is taken | refused |
| an ignored `--out` is written anyway | refused, 3 rows |
| the commit is read from nowhere | refused, 5 rows |
| the trim takes files as well as directories | refused, by the FILE named like junk alone |
| a truncation at the cap is not named | refused, by a Go-only case |
| `vendor/bundle` is left | refused, by a Go-only case |
| failed discussions leave their file | refused |
| pages are never removed | refused, 2 rows |
| the separator is always a question mark | refused, 9 rows |
| the gaps never reach `PROVENANCE.md` | refused, 7 rows |
| any answer from the control is reachable | refused, by the control case alone |

Nineteen plants, each verified to apply and to change the file, and the file
restored from a saved copy after each: all refused, none survived, none failed
to compile.

⚠ **The self-test the pair was compared by would have passed one of them.** A
joiner that keeps each page as one record still writes four `"url"` strings and
one line beginning `[`, which is everything `--selftest` counts. What refuses
it is the proxy cases' record counts - 102 issues where that joiner writes 2.

#### ⛔ And both halves are deleted: the last pair, and the port is done. 2026-09-29

`c65a553` committed the comparison with the six repairs in it; the deletion came
after. ⭐ **No twin file pair is left**:
`grep -c '^compare_pair ' scripts/common/check-twins.sh` answers 0, and
`check-twins` compares only the doctor's probe pair. The doors the deletion
touched:

| door | what changed |
| --- | --- |
| `check-twins.sh` | the pair is gone, and so is the machinery that ran pairs at once and read each verdict in list order, which nothing called any more; its header and section 7 say where the pairs went |
| `check-bitcheck.sh` | the section's notes on the twin's proxy route and its flag spellings are past tense; every case runs the port alone and says so |
| `check-gate.sh`, `check-gate.ps1` | each named `mine-repo` as the one `common/` pair left, and now says none is |
| `docs/methodology/references.md` | its first step ran the two deleted scripts; it runs `go run . mine-repo` from `tools/check`, and says why a relative `--out` still lands at the root |
| `docs/agent-tooling.md`, `.gitignore`, `adapters.go`, `check-defaults.sh` | each named `mine-repo.sh`, and the last named `check-placeholders.sh` beside it, deleted since `CI-10` ported it |

⚠ **`mine-repo` never was a gate row**, so no runner queued it and no row moved;
the doors were prose and one pair list.

### ⭐ A thirtieth refusal, written in Go with no half to compare. 2026-10-11

⚠ **`TODO/SUMMARY.md`'s effort inventory said 49 L and 0 XL while the index
carried 51 and 1**, found by the reconciliation reading of 2026-10-11 rather
than by any check: the sentence was prose beside four checked tables.
⭐ `check-project` derives it from the index now, and refuses a size the
sentence omits as well as an index effort outside the four.

| what | measured, 2026-10-11 |
| --- | --- |
| the clean tree | `bit-check check-project` exit 0 |
| the stale sentence restored | exit 1, `declares "1 13 49 0", computed "1 13 51 1"` |
| the sentence renamed away | exit 1, `declares ""` |
| an index and body effort of `XS` | exit 1, naming one row outside S, M, L and XL |
| `sh scripts/common/check-bitcheck.sh` | 238 cases, 238 passed, the new plant among them |

⚠ It is the first rule this check carries that no shell half ever held, so
`--compare` has nothing to run it against; the planted case runs the port alone.

### ⭐ Both residuals closed: the Go module is linted and the harnesses run. 2026-10-11

Two steps on the Linux lane of `.github/workflows/ci.yml`, each with a control
and a plant in `scripts/ci/check-workflow.sh`:

| step | what it runs | its plant in `check-workflow` |
| --- | --- | --- |
| *Go formatting and vet* | `gofmt -l tools/check`, refused when it lists anything, then `go vet ./...` | `go-fmt`: a function with stray spaces; `go-vet`: a `Printf` whose verb disagrees with its argument |
| *Checking harnesses* | `sh scripts/common/check-bitcheck.sh --harnesses` | `harness-weakened`: `check-cache`'s `E-CAC-01` refusal turned into a pass |

⛔ **The third plant is the reason for the second step.** Measured here before the
step existed: with `E-CAC-01`'s refusal turned into a pass, the gate stays green,
because the rule is only run against a clean tree there, and
`check-bitcheck --harnesses` exits 1 with two cases naming it.

⚠ **Every line of the Go step ends in its own verdict.** GitHub runs a step as
`bash -e` and `check-workflow` runs it as `sh -c`, which stops at nothing, so a
step whose first command fails and whose last succeeds is red on the lane and
green in the harness. ⚠ The *Shell syntax and style* step has that shape:
`shellcheck` then `shfmt`, with nothing carrying the first one's status under
`sh -c`. Its plant does not parse, so both tools refuse it and the case cannot
tell that the first one's status is dropped; a control over a tree only
`shellcheck` refuses would pass in the harness and fail on the lane. ⚠ The gate's
`check-shell` row runs both tools with each status read, which is why this is a
residual rather than a hole, and it is filed rather than widened into this
change.

⚠ **Linux only.** gofmt answers about bytes, and a Windows checkout that
converted line endings would have it list every file.

⚠ **The harness control is the dearest control after the gate**, about a minute,
and every shard pays it. Run 177's shards took 11.7 to 14.1 minutes against a
20-minute bound.

| measured here, 2026-10-11 | result |
| --- | --- |
| `sh scripts/ci/check-workflow.sh`, unsharded | 102 cases, 102 passed, exit 0, 01:42:43Z to 02:12:29Z |
| `sh scripts/ci/check-shards.sh` | 14 cases, 14 passed |
| `sh scripts/common/check-bitcheck.sh --harnesses` | 259 cases, 259 passed, 63 seconds |
| the same, `E-CAC-01`'s refusal turned into a pass | 257 passed, 2 failed, exit 1 |
| `sh scripts/common/check-gate.sh` with that plant | not run: the plant changes no clean-tree verdict, which is the point the third row of the table above makes |
| `sh scripts/common/check-gate.sh` | 40 checks, 39 passed, 1 skipped |

⛔ **What closes the entry is a CI run on a runner** reading both steps back
green, which this table cannot be.

### Closure evidence, 2026-10-11

⭐ **CI run 179 on `8d6ed5e`, read back through rule 8's route: all six jobs
`success`.** Each clause of the Prove, against what was measured:

| Prove clause | measured |
| --- | --- |
| the Go harness refuses every defect its shell predecessor refused, case for case | every pair compared by `check-bitcheck --compare` against both halves before deletion; no twin file pair is left, the last two deleted on 2026-09-29 |
| `--shard i/N` partitions the units exactly | `check-shards`, 14 cases, every N from 1 to 6, run in each of run 179's four shards |
| the matrix completes inside a bound that was not raised | run 179's four shards took 12.3, 14.6, 14.8 and 15.9 minutes under 20, lowered from 45 |
| the residuals | *Go formatting and vet* 2 seconds and *Checking harnesses* 50 seconds in run 179's Linux gate, which took 5.8 minutes |

⚠ **What is left belongs to other entries.** The `sh -c` and `bash -e` split in
*Shell syntax and style* is `CI-08`'s shape, a default inherited rather than
stated. The class-A Windows rows are `CI-07`'s, whose route this entry changed:
a row closes by its harness joining `bit-check`.

