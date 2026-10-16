# Work rules

## Status and closure

Statuses are `OPEN`, `IN_PROGRESS`, `BLOCKED`, or `DONE`. Only one entry may
be `IN_PROGRESS` unless the work is explicitly split across independent
agents. An entry becomes `DONE` only after its `Prove` commands pass and the
evidence is recorded in that entry in place.

A blocker does not close work. Record what was tried, the exact external fact
that prevents progress, and the event that would unblock it.

## ⛔ A hang is measured from off the host before it is bounded

⛔ **When a step looks hung, get evidence OFF the machine while it is running,
before adding any bound.** A host that stops reporting leaves no log and no
artifact, so an instrument that writes to its own disk measures nothing.
`bit-check flight-recorder` posts the host's state to a check run as it
goes, and its silence is itself a measurement: the moment the posts stop is the
moment the host lost contact.

⚠ **This replaced a rule that said the opposite, and the cost is the reason.**
`capture-client` runs 3 to 34 were read as a step that could not be ended,
thirteen bounds were added, and diagnosing it was forbidden. Measured on
2026-10-11: nothing was hung. The next step cut the host's default route, which
severed the runner agent from GitHub. The old rule and the measurement are in
[`../docs/history/HANG-2026-10-11.md`](../docs/history/HANG-2026-10-11.md).

⛔ **Two bounds is still the limit.** A second bound that does not end a step is
evidence that the step is not where the problem is.

⚠ **`Closure evidence` is a dated measurement of the tree at closure, not a
claim about the tree now.** Counts in it go stale the moment the next entry
lands and that is correct: rewriting one to match today would falsify what was
run. A `Prove` is the opposite and must stay runnable against the current tree,
which is why `CI-05`'s check reads `Prove` paragraphs and leaves closure
evidence alone.

## Priority and effort

- `P0`: foundational correctness, evidence integrity, or publication safety.
- `P1`: required coverage or a documented capability.
- `P2`: useful refinement after required coverage is operating.

Effort uses `S` for under a day, `M` for a few days, `L` for roughly a focused
week, and `XL` for a cross-cutting item that should be split before execution
when practical.

## Evidence

Published identity fields must come from active, replayable interaction with
the exact installed build. Source reading, peer-ID tables, search results, and
third-party statistics may create a hypothesis or set priority; they are not
corpus evidence. At least two independent connectors must observe each run,
and overlapping fields must agree before publication.

## Synchronization

A status change updates the entry, [`INDEX.md`](INDEX.md),
[`SUMMARY.md`](SUMMARY.md), and [`PROGRESS.md`](PROGRESS.md) in the same
change. Main documentation is amended in place. Narrative history belongs in
[`../docs/history/`](../docs/history/), never appended to current-state docs.
