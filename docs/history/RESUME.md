# Resume

Overwritten at every session start and whenever the in-flight item changes.
[`../methodology/sessions.md`](../methodology/sessions.md) says what it carries
and why it is not the record. The work order is
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md) and nowhere else.

| | |
| --- | --- |
| the task | ⛔ **Paused for a transfer and refactor of the repository**, settled by the operator on 2026-10-22. A session resuming here takes the work order in `TODO/PROGRESS.md` in sequence. |
| the resume point | `ENGINE-01`: dispatch libtorrent's Linux capture on `main` (inputs in `TODO/PROGRESS.md`'s work order), then assemble and publish. |
| in flight | nothing. Every commit is pushed, no dispatch is running, and no worktree carries unpushed work. |
| the state of the tree | `main` green in CI; `data` holds 36 records; 12 of 69 entries open. |

## Where things are

- ⭐ **Published data** is the `data` branch, appended by `publish-data.yml`
  from an `assemble-bundle.yml` run, which reads a `capture-client.yml` run.
  Assemble against the current `data`: a bundle assembled before the last
  publish landed omits that publish's records, so assemble again rather than
  publishing it.
- ⭐ **One adapter per target** under `tools/check/adapter_*.go`, with its pure
  decisions and selftest in `tools/check/<target>.go`. The contract is
  [`../../scripts/capture/adapters/README.md`](../../scripts/capture/adapters/README.md).
  Engine targets declare `target_kind=library`; their pinned harnesses live
  under `tools/harness/`.
- ⭐ **Every scheme declares `release_series`**, `-` for none, since
  `cc76d1e`. libtorrent's is `2.0`.
- ⚠ **`capture-client` keeps only the newest pending run** in its concurrency
  group. Dispatch one capture at a time, and the next when the queue is free.
- ⚠ **CI runs workflow acceptance on eight shards** under a 20-minute bound.
  The slowest took 1091 seconds on CI run 39742567476. A unit that runs the
  whole gate or the whole harness is what makes a shard heavy.
- ⚠ **The session host's writable disk is a fixed allowance.** A full disk
  makes `check-workflow` report a guard as could-not-run. Clear build output
  before a long run.
- ⚠ **This session host has IPv6 disabled**: rqbit and anacrolix/torrent can
  only be captured here IPv4-only, or on a runner.

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
Before every push, run what the table at the top of scripts/README.md names for
the kind of change. Debug a capture dispatch from its flight-recorder check run,
not from the job's step list.
```
