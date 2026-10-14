# Lessons

The defect classes this repository has shipped and caught, kept where a review
can read them before it starts. [`methodology/reviews.md`](methodology/reviews.md)
is how a review is run; this is what it should already know.

Moved here on 2026-10-11 from the record and from `history/RESUME.md`, which is
overwritten every session and was carrying them as the only copy. ⚠ A lesson
that is mechanical belongs in a check, and one that is grep-able also belongs
in [`conventions/forbidden-patterns.md`](conventions/forbidden-patterns.md).

---

## Before a review starts

⛔ **A step that never ends may have ended.** GitHub shows the last step a runner
reported, and a host that loses contact reports nothing more. Runs 3 to 34 of
`capture-client` were read as *Install the client* hanging; it had finished in
seconds, and the next step had cut the route the runner agent talks over. ⭐ Get
the host's state off the machine while the job runs before believing which step
is stuck: `scripts/ci/flight-recorder.sh`.

⛔ **A bound that never fires is evidence about the bound's target.** Thirteen
bounds on one step all failed the same way because the step was not running.

⛔ **A guard whose SCOPE is wider or narrower than the thing it guards.** Three
instances in one day, all in the capture path. Ask what actually ships.

⛔ **A guard that sees ONE SPELLING of the thing it forbids reports clean over
the others.**

⛔ **An upload widened to the failure path inherits none of the success path's
guards.** The obvious fix for a refused capture uploading nothing was `if:
always()`, and it would have shipped a bundle the rule-12 scan, which runs only
when a capture succeeds, had never read. Ask what the path being added skips.

⛔ **A rule against CHOOSING one value grew into a rule against REPORTING two.**
The second connector refused two announces that differed, so a build whose peer
ID moves between sessions - a shape the record already models - turned two
captures red. Ask whether a refusal forbids the defect or the measurement.

⛔ **A one-gated-door defect arrives in the change that fixes something else.** A
subject left outside a build gate answers 127 under every environment, which
reads as *the same answer under all 6*.

⛔ **A corpus only tests the defects it contains an example of.** A rule
differing only on a shape the tree does not contain is invisible to any
comparison over that tree. Plant the shape.

⛔ **A RULE WHOSE PRECONDITION IS ABSENT READS CLEAN, FOREVER.** `E-CRP-03`
refuses an artifact a store does not carry and is checked per run manifest, so
with no manifest in the store the assembler filed four citations per record at
paths nothing had written and every gate stayed green. ⚠ Ask which rules a
missing document takes with it.

⛔ **A validator handed an EMPTY input answers about nothing and looks green.**
The same corpus check, given an empty tree instead of one read off the disk,
found half of what was wrong and said so in the same confident sentence.

⛔ **A report that cannot print its own failures is a silent run.** `store_report`
counted rows by LINES, a disagreement is three lines, and the one comparison with
something to say returned 1 having printed nothing at all.

⚠ **`@(...) | Sort-Object` wraps the INPUT and leaves the output unwrapped**, so
an empty result is `$null` and `.Count` throws. A rule written for the empty case
was the one case it could not reach.

⛔ **A harness that plants a pattern cannot spell it** - and neither can the
comment explaining why.

⚠ **A plant that did not APPLY is a third status**, not a survivor. Diff the file
before believing either answer. ⚠ A harness exit of 2 is *could not run*, never
*refused*. ⚠ A plant whose expected outcome is a PASS proves nothing, except
where the rule's risk is over-strictness.

⛔ **A guard that nothing can refute is still worth keeping, and saying so is the
work.** Label the case with the rule that really refuses it.

⛔ **A global regex replace where the pattern also matches what it preserves.**

⛔ **A step does not end when its command exits.** ⛔ **A shell applies
redirections left to right.** ⛔ **`timeout 0` MEANS NO LIMIT.** ⛔ **A PowerShell
`[switch]` collides with a local differing only in case**, and `$args` inside a
function is automatic.

⛔ **A BOUND INSIDE `$( )` IS NOT A BOUND.** A command substitution ends on the
PIPE's end of file, not on the child's exit, so one process the product leaves
behind blocks it forever while the bound reports success. Redirect an untrusted
process to a FILE and read the file.

**AN UNPRIVILEGED BOUND CANNOT END A ROOT PROCESS.** Put the bound inside the
`sudo`, never around it. A bound outside fires on schedule, reports 124, and
leaves the work orphaned at PPID 1. And nested `timeout`s each create their own
process group, so an outer bound does not reach an inner one's children.

**A LOCAL PASS AS `root` SAYS NOTHING ABOUT A RUNNER.** Both of the above were
invisible to every local drive for eight dispatches, because this host is `root`
and `ubuntu-24.04`'s runner is uid 1001 with `sudo`.

⛔ **A Python text-mode rewrite of a `.ps1` silently converts CRLF to LF**, and
`git diff` shows nothing. `git ls-files --eol` is the only thing that does.

⛔ **In Go, `filepath` is the HOST's separator and `path` is slashes.** A
repo-relative key derived with `filepath.Dir` matches on Linux and misses on
Windows. ⚠ Running `check-gate.ps1` on Linux does not test Windows: only the
Windows lane can.

**A count in prose is a value in two places with nothing comparing them.**
**An in-place `sed` edits every line that matches, not the one you meant.**

⛔ **A rule a document says this repository has is not a rule it has.** Grep for
the check before believing it runs. Three were found missing this way.

**The strongest control available is a reader this project did not write.**

⛔ **A CAPTURE'S WORKDIR IS ITS EVIDENCE BUNDLE**, so rule 12 applies to what an
adapter writes there - bounded by what the artifact actually ships.

⚠ **GitHub runs a `run:` block as `bash -e`, so a failing command ends the block
on its own line.** A trailing `true` after a `grep` that matched nothing is
unreachable, and so is an `rc=$?` on the next line: until 2026-10-14 this entry
said to capture the status on its own line, and *Install the client* did, so no
failed install printed its log to the job log from 2026-09-16, when that body
was written, to run 35. ⭐ Capture it in the same
command: `rc=0`, then `cmd || rc=$?`.

⛔ **A push chained after a gate with `;` is a push over a red gate.**
`e4132c8` went up on 2026-10-14 with `check-markers` and `check-bitcheck`
failing, because the command ran the gate, then committed and pushed whatever
it answered; `82cc37d` repaired it. ⭐ Read the gate's exit code and push only on
0: `check-gate.sh && git push`, never `check-gate.sh; git push`.

⛔ **An update that replaces a whole object erases what it does not carry.** The
flight recorder posted its last snapshot and then closed the check run with an
`output` holding a title alone, so every run that closed cleanly showed an empty
text. Send the last state in the request that closes.

⛔ **A guard that refuses may be refusing for the smaller reason.** `aria2-next`
3.0.2 was refused over a switch it stopped reporting, and the options dump that
came back held a discovery switch, on by default, that no guard named. Read what
the refusal uploaded before narrowing the guard to let the build through.

---

---

## From the local gate

⛔ **A prover that could not run reported nothing, and the gate read that as a
skip.** Exporting `CARGO_TARGET_DIR`, which a great many Rust developers do, put
every built example somewhere the harnesses did not look, so all five corpus and
publishing provers exited 2 and the whole tier silently stopped proving
anything. Two places composed that path and fixing one left the other; `CI-01`
carries both. ⚠ The lesson is about the status rather than the path: exit 2 is
the honest answer for a harness that cannot run, and a tier of them answering it
at once still looked like a green gate.

⭐ **The same hazard is not confined to the shell twins.** `FOUND-03` planted
nine lossy defects in the Rust codecs and two were missed on the first pass,
each because the corpus lacked the shape that would have failed: no fixture used
a bare newline, and no path reached the bencode encoder at all. A corpus only
tests the defects it contains an example of.

⚠ **A constant every test reads is a constant no test can check.** `OBS-08`
found two of that shape before planting against them: nothing pinned
`PIECE_HASH_LEN`, so narrowing it re-chunked the `pieces` string and the
comparison against it together, and the test spec was built at
`MIN_PIECE_LENGTH`, which made the declared piece length and the floor
indistinguishable. Pin a specification's own values to their literals, and build
a fixture at a value no default or bound also has.

⭐ **Third-party readers are installable here and are a much stronger driven
pass than a decoder written for the purpose.** `libtorrent` 2.1.1.0 and `torf`
4.3.1 install into a virtualenv from the package index and read a `.torrent`
without touching the network. Parsing a file is not a capture and needs no
disposable host; running a client still does.

⭐ **A driven pass gets its strength from the client knowing what it sent.**
`OBS-09`'s reads the bundle back with the same Python client that put the bytes
on the wire, so the transcript can be checked against what actually happened
rather than against what the lab believed happened. A reader that only
re-computes digests is checking the writer against itself.

⚠ **`grep -F` splits a pattern containing a newline into separate
alternatives**, so a unique multi-line literal counts as the sum of its lines and
a plant verifier built on it reports NOT-PLANTED over a plant that would have
applied. Measured on 2026-09-05 while reviewing `CORPUS-01`, where it miscounted
three plants. It fails safe, and it still costs the guards it silently declines
to prove. Count a multi-line literal with something that understands one, or
refuse it outright as `scripts/corpus/check-store.sh` does.

⭐ **The corpus harnesses need `cargo`, `sha256sum` and, for `check-store`,
`mkfifo`, and exit 2 without them.** The Linux CI lane runs the gate with
`--strict`, so such a skip would be a failure there; all three are present on
`ubuntu-24.04`.

⛔ **A sourced shell library shares one namespace with its caller.** A harness
assigned its own `ROWS` for a row count, overwrote the library's accumulator, and
printed ten passes over eight lines. The globals are prefixed now and
`store_report` compares the rows it holds against the count it is about to
print, because a naming convention is a rule nobody checks.

⛔ **A plant that survives is not automatically a gap, and a plant that is
refused is not automatically proof.** Of eight over the index builder, two
survived because the mutation was equivalent on the fixture data and one
"refusal" on an earlier pass was a harness exit of 2. Read what the plant
actually changed before believing either answer.

⛔ **A harness exit of 2 is *could not run*, never *refused*.** A review pass
counted it as a refusal on one of its two paths and reported a guard proved over
a plant that had not compiled. Separate the two statuses on every path, not on
the one where it first bit.

⚠ **`shellcheck` answers differently depending on how the files were grouped on
its command line.** A script that sources another is clean when both are handed
to one invocation and warns when checked alone, because it cannot follow a source
it was not given. CI passes every script at once and a contributor checking one
file does not, so the directives belong in each file rather than in the
invocation.

⛔ **`pgrep -f` and `pkill -f` match any command line that merely contains the
text.** A harness case looked for a leaked `sleep 7123` with `pgrep -f` and
cleaned up with `pkill -f`. On 2026-10-14 that killed the session shell running
the harness, because the shell's own command line quoted the text. Match the
whole argument vector, as `ps -eo pid=,args=` read by `awk` lets you, and never
signal a process found by substring.
