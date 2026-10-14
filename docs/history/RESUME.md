# Resume

Overwritten at every session start and whenever the in-flight item changes.
[`../methodology/sessions.md`](../methodology/sessions.md) says what it carries
and why it is not the record. The work order is
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md) and nowhere else.

| | |
| --- | --- |
| the task | Take the work order in `TODO/PROGRESS.md` in sequence. Session started `2026-10-14T02:33:13Z` on `30a9801`, which CI run 183 read back green on all six jobs. |
| the resume point | `CI-09` step 5: `assemble-capture` writes a `manifest.json` per record from the keys steps 1 to 4 added, and the corpus check reads it back. Test it on the dispatch carrying those keys. |
| in flight | a `capture-client` dispatch on the barrier's commit, the first to carry every new key. |
| the state of the tree | `check-workflow` 115 of 115; `check-step-bodies` 39 of 39. |

## Starting, on this host

- ⚠ The container starts on a `claude/*` branch, with `user.name` set to an
  agent, and a shallow clone. Correct all three before editing: `main` per rule
  7, the operator's identity per rule 11 read from
  `git log --format='%an <%ae>' | sort -u`, and `git fetch --unshallow`.
- ⭐ `sh scripts/doctor/provision.sh` installs `pwsh`, `shellcheck` and `shfmt`.
  `go` is a gate dependency.
- ⭐ What to run before a push, by what changed, is the table at the top of
  [`../../scripts/README.md`](../../scripts/README.md).
- ⭐ A capture dispatch is debugged from off the host: the flight recorder's
  check run on the commit shows the host's state while the job runs.
- ⚠ The session host is `root` and a runner is uid 1001. Drive anything that
  uses `sudo` as an unprivileged user before believing a local pass.
- ⭐ A capture that refuses uploads its bundle after `scan-bundle.sh` passes, which
  is what run 35 was diagnosed from. ⚠ In that run a closed recorder check run
  showed no text and a failed install printed nothing to the job log; both are
  fixed since, so the next dispatch is the first to show either.

## The paste

```text
Read docs/AGENTS.md in full and execute its session-start protocol. Work on main.
Re-measure identity, branch, remote and clone depth before editing: the container
may start on a claude/* branch, with an agent identity, and a shallow clone.
Install tools with sh scripts/doctor/provision.sh. Then read
docs/history/RESUME.md, the record, and the required reading the router names.
Before every push, run what the table at the top of scripts/README.md names for
the kind of change. Debug a capture dispatch from its flight-recorder check run,
not from the job's step list.
```
