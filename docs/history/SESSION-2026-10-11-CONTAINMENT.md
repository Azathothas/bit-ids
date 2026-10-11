# Session 2026-10-11: the record reconciled, CI-10 closed, the capture hang found

Started `2026-10-11T01:24:45Z` on `5d54c5f`, which CI run 177 had read back
green. Worked on `main` with the operator identity, per `docs/AGENTS.md`.

## What landed

| commit | entry | what |
| --- | --- | --- |
| `ad5126a` | docs | the record and the resume file rewritten to current truth after the 2026-09-29 session ended without its ending; its record reconstructed in [`SESSION-2026-09-29-PORT.md`](SESSION-2026-09-29-PORT.md); `check-project` derives `TODO/SUMMARY.md`'s effort line |
| `8d6ed5e` | `CI-10` | *Go formatting and vet* and *Checking harnesses* on the Linux lane, with a control and a plant each in `check-workflow` |
| `9df7699` | `CI-10` | closed on CI run 179; a `runner` input on the wrong premise, removed in the next commit |
| `3f3d773` | `CI-08` | the capture "hang" found: the route cut severed the runner agent. `contained.sh` runs the capture in a network namespace; `flight-recorder.sh` posts the host's state off it; the rule that forbade diagnosing the hang rewritten |
| `21ae69f` | `CI-08` | the contained capture proved on both `aria2-next` routes from `main`'s workflow |

## How the hang was found

The operator directed it be fixed at the root, overriding the rule against
diagnosing it. Every earlier instrument wrote to the runner's own disk and was
lost with the job. The recorder posted to a check run while the job ran, and
four pushes to the session's `claude/` branch narrowed it in about fifteen
minutes: a healthy host, an install that exited 0, every operation after it
returning, and silence the instant *Cut the route* ran.
[`HANG-2026-10-11.md`](HANG-2026-10-11.md) keeps the old rule and the
measurement; `TODO/ci.md` under `CI-08` keeps the runs.

## What it got wrong on the way

- ⚠ It first took a second runner image for the cause, added a `runner` input,
  and dispatched on `ubuntu-22.04`, which hung the same way. The input is
  removed.
- ⚠ It reported a watchdog loop that would spin on an unreaped child; the
  markers showed the loop ending after one pass. No such defect was filed.

## What it left

- The second connector refuses `aria2-next` 2.8.6's captures over a peer ID that
  differs between two announces. That is the work order's first item.
- `capture.yml` still cuts the route. `CI-08`.
- The session's GitHub dispatch tool lost its session mid-way, so the proof runs
  were started by pushes to the `claude/` branch through a throwaway workflow;
  that branch was reset to `main` afterwards and the workflow never reached
  `main`.
