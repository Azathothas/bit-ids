# Resume

Overwritten at every session start and whenever the in-flight item changes.
[`../methodology/sessions.md`](../methodology/sessions.md) says what it carries
and why it is not the record. The work order is
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md) and nowhere else.

| | |
| --- | --- |
| the task | Take the work order in `TODO/PROGRESS.md` in sequence. |
| the resume point | the work order's first item: the second connector's refusal over a varying peer ID. |
| in flight | nothing. The 2026-10-11 session ended at a clean state; [`SESSION-2026-10-11-CONTAINMENT.md`](SESSION-2026-10-11-CONTAINMENT.md) is its record. |
| the state of the tree | clean, gate green, and both branches at one commit when this was written. Read CI back for the tip before trusting that. |

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

## The paste

```text
Read docs/AGENTS.md in full and execute its session-start protocol. Work on main.
Re-measure identity, branch, remote and clone depth before editing: the container
may start on a claude/* branch, with an agent identity, and a shallow clone.
Install tools with sh scripts/doctor/provision.sh. Then read
docs/history/RESUME.md, the record, and the required reading the router names.
```
