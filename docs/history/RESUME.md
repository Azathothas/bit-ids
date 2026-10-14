# Resume

Overwritten at every session start and whenever the in-flight item changes.
[`../methodology/sessions.md`](../methodology/sessions.md) says what it carries
and why it is not the record. The work order is
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md) and nowhere else.

| | |
| --- | --- |
| the task | Take the work order in `TODO/PROGRESS.md` in sequence. Session started `2026-10-14T02:33:13Z` on `30a9801`, which CI run 183 read back green on all six jobs. |
| the resume point | dispatch `capture-client` and assemble what it uploads. |
| in flight | `157b133` (the connector reports every sample) is pushed and CI run on it is green on all six jobs. Uncommitted: `scripts/capture/scan-bundle.sh` and a scan step so a refused capture's bundle is uploaded after rule 12, and *Stop the flight recorder* so the recorder closes its check run; `capture-client.yml`, `check-workflow.sh`, `check-step-bodies.sh`, `check-capture-client.sh`, `flight-recorder.sh`, `capture-client.sh`, `install-client.sh`. Still to run before the push: the recorder plants, the gate, and `check-workflow.sh` unsharded. |
| the state of the tree | dirty with the work above; the harnesses it touches pass. |

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
