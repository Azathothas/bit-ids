# Resume

Overwritten at every session start and whenever the in-flight item changes.
[`../methodology/sessions.md`](../methodology/sessions.md) says what it carries
and why it is not the record. The work order is
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md) and nowhere else.

| | |
| --- | --- |
| the task | Reconcile the record and the docs with the tree after the 2026-09-29 session ended without running its ending, then take the work order in sequence. |
| started | 2026-10-11T01:24:45Z, on `5d54c5f`, which CI run 177 read back green through rule 8's route. |
| the resume point | land the namespace containment on `main` and prove it with a dispatch from there; then `capture.yml` the same way. |
| in flight | the capture-hang root fix: `contained.sh`, `flight-recorder.sh`, `capture-client.yml`, `check-workflow.sh`, and the record. Diagnostic runs live on the `claude/` branch only. |
| the state of the tree | gate 40 checks, 39 passed, 1 skipped, on `ad5126a` before its push. |

## Starting, on this host

- ⚠ The container starts on a `claude/*` branch, with `user.name` set to an
  agent, and a shallow clone. Correct all three before editing: `main` per rule
  7, the operator's identity per rule 11 read from
  `git log --format='%an <%ae>' | sort -u`, and `git fetch --unshallow`.
- ⭐ `sh scripts/doctor/provision.sh` installs `pwsh`, `shellcheck` and `shfmt`.
  `go` is a gate dependency; without it the ported rows skip.
- ⛔ The session host is `root` and a runner is uid 1001. Drive anything that
  uses `sudo` as an unprivileged user before believing a local pass.

## The paste

```text
Read docs/AGENTS.md in full and execute its session-start protocol. Work on main.
Re-measure identity, branch, remote and clone depth before editing: the container
may start on a claude/* branch, with an agent identity, and a shallow clone.
Install tools with sh scripts/doctor/provision.sh. Then read
docs/history/RESUME.md, the record, and the required reading the router names.
```
