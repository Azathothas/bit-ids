# Session 2026-09-29: the last twin pairs ported

⚠ **Reconstructed on 2026-10-11 from the tree, the commit log and CI**, because
this session ended without running its ending: no session record, no summary
table, and a resume file still naming its first in-flight item. Nothing here
comes from the session's own account of itself.

## What it did

Started at `2026-09-29T01:02:41Z` on `ff690a3`, per the resume file it wrote.
Ten commits, `2451816` through `5d54c5f`, 38 files, 4145 lines added and 3472
removed.

| commit | entry | what landed |
| --- | --- | --- |
| `2451816` | `FOUND-04` | `.github/dependabot.yml` ignores every `rusqlite` version from 0.38 |
| `621f93a` | `CI-10` | `check-cache` ported into `bit-check` and compared against both halves |
| `ab1e2a7` | `FOUND-04` | dependabot read the ignore and closed pull request 2 itself |
| `a8cfedf` | `CI-10` | both halves of `check-cache` deleted |
| `c40784f` | `CI-10` | `check-catalogue` ported and compared |
| `be26fe7` | `CI-10` | both halves of `check-catalogue` deleted, and `store-lib.ps1` with them |
| `f0e4f7f` | `CI-10` | `check-remote-items` ported, with fifteen stubbed cases in the default mode |
| `e6ae4c4` | `CI-10` | both halves of `check-remote-items` deleted, and `Invoke-Check` with them |
| `c65a553` | `CI-10` | `mine-repo` ported and compared whole for the first time |
| `5d54c5f` | `CI-10` | both halves of `mine-repo` deleted; no twin pair is left |

CI runs 170 to 177 cover those commits through rule 8's route: every one
`success` except run 172 on `c40784f`, which is `cancelled`. Its successor, run
173 on `be26fe7`, is green.

## What it left stale

Read on 2026-10-11, before the record was rewritten.

- ⛔ **The work order sent `CLIENT-01` and `CLIENT-06` to a dispatch that had
  already happened.** It said the next dispatch would say whether a rootless
  install finishes on a runner. Run 32, on 2026-09-17, had said it does not, and
  the same record carried that a few lines above.
- ⚠ **`TODO/SUMMARY.md`'s effort inventory said 49 L and 0 XL** where the index
  carries 51 and 1. `CI-10` is the XL entry; nothing compared the sentence to
  the index.
- ⚠ **The record and the resume file were one long narrative each**, 672 and
  372 lines, holding current state, superseded readings and review lessons
  together. Both were rewritten to current truth. The record's old text is
  [`PROGRESS-2026-09-29.md`](PROGRESS-2026-09-29.md), verbatim, and the lessons
  are [`../lessons.md`](../lessons.md).
- ⚠ **`CI-10` stayed `IN_PROGRESS` with two residuals and no session closing
  it.** That is the right status; it is named here so the next reader does not
  take the missing ending for a missing step.

## What was not reconstructed

The session's own gate results, review passes and timings are not recoverable
from the tree beyond what each commit's entry carries in `TODO/ci.md` under
`CI-10`. No number above is taken from anywhere but the log and the CI API.
