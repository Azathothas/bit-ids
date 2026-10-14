# Resume

Overwritten at every session start and whenever the in-flight item changes.
[`../methodology/sessions.md`](../methodology/sessions.md) says what it carries
and why it is not the record. The work order is
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md) and nowhere else.

| | |
| --- | --- |
| the task | Take the work order in `TODO/PROGRESS.md` in sequence. Session started `2026-10-14T02:33:13Z` on `30a9801`, which CI run 183 read back green on all six jobs. |
| the resume point | the work order's first item: a capture that refuses uploads no evidence, then a dispatch. |
| in flight | the connector fix is committed (every sample reported and corroborated, `CI-09`). Next: the rule-12 scan out of `capture-client.sh` into one script, so *Upload the evidence bundle* can run on a refused capture without shipping an unscanned workdir; `.github/workflows/capture-client.yml`, `scripts/capture/capture-client.sh`, `scripts/ci/check-workflow.sh`. |
| the state of the tree | gate 40 checks, 39 passed, 1 skipped, exit 0, measured after the connector change. |

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
- ⚠ A capture that refuses uploads no evidence bundle today, so runs 38109299467
  and 38111442119 left only their job logs: the connector's refusal is in them,
  the bytes are not.

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
