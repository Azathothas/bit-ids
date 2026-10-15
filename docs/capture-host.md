# The capture host

What a machine must be before this project installs a client on it, and what
refuses one that is not.

A capture runs a binary this project downloaded minutes earlier, from a vendor
it does not control, and points it at a network. That is the whole product and
it is also the one thing that can go wrong outside a record. Everything here
exists so the blast radius is a machine nobody keeps.

## The boundary is before the install, not in the record

⛔ **`E-MAN-30` refuses to record a capture whose host was not disposable. It
cannot stop one.** By the time a manifest exists, an untrusted installer has
already run somewhere. A record-layer refusal is a report.

So the boundary is
[`../scripts/acquisition/assert-disposable.sh`](../scripts/acquisition/assert-disposable.sh),
which runs before anything is installed, and which
[`../scripts/acquisition/check-runner.sh`](../scripts/acquisition/check-runner.sh)
mutation-proves on every gate.

## Two guards, and why they are not one

| guard | refuses | independent because |
| --- | --- | --- |
| `--claim` | a host that already ran a capture | it detects a survived host whatever the configuration claims |
| `--egress` | a host that can reach anything but loopback | it reads the routing table, which no claim can talk it out of |

A fresh host with an open route leaks the capture onto the public network. A
firewalled host that already ran a capture contaminates this one with the last
one's state. Neither guard implies the other, and
[`AGENTS.md`](AGENTS.md) section 5 requires both before any client is installed.

### The claim detects the failure rather than trusting the claim

⛔ **The obvious design is a token the provisioner writes saying "this host is
disposable", and it is the wrong one.** A promise is exactly what fails
silently: a runner misconfigured to persist its disk still carries the token,
still says disposable, and nothing notices until two captures share state.

So `--claim` writes a marker and refuses if one is already there. A second
capture on one host means the host survived the first, which means it was never
disposable whatever anything claimed. The evidence is the marker's existence,
and a misconfiguration cannot produce its absence.

⚠ **Where the marker lives is load-bearing.** `/run` and `/tmp` are cleared by a
reboot, so a host that rebooted rather than being destroyed would read as fresh.

⭐ **It is the capturing user's own state directory, `$HOME/.local/state/bit-ids`,
and it needs no privilege.** `ACQ-06`. `/var/lib/bit-ids` needs root to create,
so every caller of this guard needed `sudo` - including the install step, which
reads the marker before a route runs. A home directory keeps both properties the
marker rests on: a reboot does not clear it, and it goes when the host does.

⛔ **And it is per-user, which is a narrower claim than `/var/lib` made.** A host
where two different users each ran a capture carries two markers and neither
refuses the other. That is a real weakening rather than a tidy-up, and it is
stated here because the runner contract is one capture per host: a shared host
was never one this project would capture on. The Windows half still writes under
`ProgramData`, so the two platforms differ here until a Windows capture installs
a client.

The manifest records the guard's answer in `isolation.claim`: the fingerprint it
read and when it claimed the host. `E-MAN-33` refuses a claim stamped after the
run started, because a guard that ran after the install is a report again.

### The egress guard asks the kernel, not the network

⛔ **Nothing probes a third party.** Reaching out from a machine this guard
exists to establish is contained would be exactly the wrong order. It reads
`/proc/net/route` and looks for a default route that is up.

⚠ **POSIX awk only.** The first version used `and(strtonum(...))`, which are
gawk extensions. On a POSIX awk they are undefined functions, awk exits
non-zero, and the guard reported "could not establish" over a machine that
plainly had a default route. It failed closed, which is the right direction, but
a guard that cannot run on a minimal image does not run where it matters most.

## Reading the guards' output

Both guards print the input they trusted: `--egress` names the routing table it
read and `--claim` names the marker it wrote. The routing table is an optional
argument and the marker directory an environment variable, so the runner test
can drive them against fixtures.

⚠ **A seam a test can use is a seam a misconfiguration can use.** Printing the
input is what makes a run that passed over a fixture visible in a log rather
than indistinguishable from one that passed over the machine.

⛔ **`--marker` prints where the claim marker lives, and is the only derivation
of that path.** A later step that wants to know whether this host was claimed,
and by which run, reads the file this names. `capture-run` is the caller it
exists for: it re-reads the claim at the moment of capture rather than trusting
that an earlier workflow step ran. A caller that composed the path itself would
be a second spelling, and it would go on reading the old place the day the state
directory moves - reporting a host nobody claimed as claimed.

## Linux runner contract

1. The host is created for one capture and destroyed after it. Not reset, not
   reimaged in place: destroyed.
2. Before any download or install: `assert-disposable.sh --claim <run-id>` and
   `assert-disposable.sh --egress`, both exit 0. Exit 2 is not a pass.
3. No route off the host except the loopback endpoints the run starts itself.
   A run that needs more records why, and `E-MAN-31` refuses one that does not.
4. No credential of any kind is present. The run needs none: everything it
   fetches is public, and
   [`security/secrets.md`](security/secrets.md) is the rule.
5. The account running the client owns nothing that outlives the host.
6. After the run, the host is destroyed. ⚠ On Linux the next job's
   `--fingerprint` then differs; on Windows it does not, and the paragraph after
   next says why.

⚠ **A fingerprint is comparable only against one the same half produced.** The
two guards digest different inputs, so a Linux job and a Windows job report
different values for one machine and would report different values for two. A
workflow comparing across platforms would read "a fresh host" from nothing at
all. Compare a Linux job's against the previous Linux job's.

⛔ **And on a hosted Windows runner the comparison answers nothing.** Measured by
`CI-06` across two dispatches: two Windows jobs on two hosts that were both fresh
reported the **same** fingerprint, while the two Linux jobs reported different
ones. The hosts really were fresh - each run's claim succeeded, so the marker the
previous run wrote was gone.

⭐ **That is structural rather than a defect to patch.** A fingerprint has to do
two things at once: differ between two hosts, and survive a reboot within one, so
that a machine which rebooted instead of being destroyed still reads as itself.
Every input with the second property is a property of the **image**, and hosted
runners are clones of one image, so on Windows the two requirements have no
common solution. ⚠ Adding a per-boot value would buy the first by destroying the
second, which is the failure the whole guard exists to catch.

⛔ **The claim marker is the guard that actually answers it**, and always was: it
detects a survived host by finding its own marker rather than by comparing a
value against a previous run's. The fingerprint is an image identity recorded
beside the measurement, and on Linux it happens to be a freshness signal too.

## Windows runner contract

The same six rules, and
[`../scripts/acquisition/assert-disposable.ps1`](../scripts/acquisition/assert-disposable.ps1)
is the pair that enforces them. `-Claim` writes an exclusively created marker
under `ProgramData`, which survives a reboot so a host that rebooted rather than
being destroyed still reads as claimed; `-Egress -Program` reads the firewall's
active store; `-Fingerprint` digests the machine GUID with the install identity
and the computer name.

⛔ **The Windows capture is contained at the firewall, not by its routes.**
Windows has no network namespace, and cutting the host's default routes severed
the runner from GitHub (`CI-08`).
[`../scripts/capture/firewall-contain.ps1`](../scripts/capture/firewall-contain.ps1)
turns every firewall profile on and writes one outbound block rule per
executable the capture runs, the list `capture-run.ps1 -Programs` names. The
guard passes only when every one of those programs is covered by an enabled
outbound block rule narrowed by nothing, under profiles that are all on: a rule
narrowed to one profile, address, protocol, interface, service or user blocks
some traffic and reads as a block in a listing. ⚠ Loopback is not filtered for a
desktop process, which is how the observer and the driver still meet on
127.0.0.1; the capture proves that on every run.

⚠ **`-FirewallState` makes the firewall a file**, which is how
[`check-runner.ps1`](../scripts/acquisition/check-runner.ps1) proves the logic on
a machine that has no `Get-NetFirewallRule`. ⭐ **That the host enforces the
rules is measured on every dispatch rather than assumed**: *Prove the firewall
blocks* has the capture's own blocked `curl.exe` try to reach GitHub and accepts
only exit 7, could not connect, with the host itself reaching the same address
as the control.

⚠ Until 2026-10-15 `-Egress` read `Get-NetRoute`, and `CI-06`'s dispatch
measured that cmdlet's output against the route fixtures on a runner whose
default routes had just been removed.

## The workflow that runs on such a host

[`../.github/workflows/capture.yml`](../.github/workflows/capture.yml) is the
only thing that runs a capture on a hosted runner, and `CI-03` owns it. One job
per platform, and the step order is the containment:

| step | why it is where it is |
| --- | --- |
| claim the host | first, before anything else writes to it: the claim detects a survived host by finding its own marker, and a step that left state earlier would not be detected |
| build the observer | ⛔ **while the network still exists.** After the next step nothing can be fetched |
| cut the route off this host | both address families, with the routes saved first |
| assert containment | the guard reads the kernel, not the step above. Those are two facts and only the second is evidence |
| capture | `capture-run` re-reads the marker and the routing table itself |
| restore the route | only to upload, after the measurement is on disk. ⛔ Every path through it ends in an explicit `exit`, because the step's status is otherwise whatever the block left behind |
| upload the evidence bundle | `if-no-files-found: error`, so an upload that found nothing is red |
| report the host fingerprint | compared against the previous run **of the same platform**, and on Windows that comparison answers nothing |

⛔ **There is no `pull_request` trigger and that absence is the fork guard.** A
fork cannot cause a workflow to run in the base repository, so there is no
job-level condition for anyone to weaken and no `if:` to get subtly wrong.
[`../scripts/ci/check-workflow.sh`](../scripts/ci/check-workflow.sh) asserts the
absence, because an absence is what a later edit restores unnoticed, and it
asserts the step order for the same reason: every wrong ordering above reads as
plausible in a diff.

⭐ **The restore is verified by the same guard, inverted.** A host that can reach
the network again is one `--egress` **refuses**, so a guard that still passes
after the restore means the route never came back and the upload would have
failed with a network error naming nothing.

⛔ **`capture-run` builds nothing.** A capture that discovered a missing
dependency under containment would have to restore egress to fix it, on the host
that exists to have none, so the observer is a path to an already built binary
and a missing one is *could not run*. That makes the step order enforced rather
than remembered.

⚠ **What it captures today is a fixture, and the attestation says so.**
`kind=fixture`, `measured_build=none` and `stock_client=false` are fields rather
than prose, because a bundle that outlived its context would otherwise read as a
measurement of a client. Nothing is installed there.

## The workflow that installs a product

[`../.github/workflows/capture-client.yml`](../.github/workflows/capture-client.yml)
is the second capture workflow and the one that runs somebody else's binary.
`CLIENT-01`, `CLIENT-05` and `CLIENT-06` own the adapters it drives.

⛔ **It is a separate file rather than a branch inside the first**, because the
step order it needs is different and the order *is* the containment. A
conditional install inside `capture.yml` would be a step that sometimes runs and
sometimes does not, in the one place where every reader has to be able to see
what happened without reading an expression.

| step | why it is where it is |
| --- | --- |
| claim the host | ⛔ first, and before the install as well as before the capture: a product installed on a host nothing established was disposable is state on a machine that may be kept |
| build the observer | while the network still exists, and it is every binary the job runs rather than only the observer |
| resolve the release artifact | ⛔ before the install, which needs its answer, and outside the capture's namespace, which has no route. Only on the `release` lane: a package index needs no artifact chosen |
| install the client | ⛔ also outside the namespace, with the network in reach. `capture-client` refuses to install anything itself. ⛔ The route's output goes to a **file** and is printed afterwards, so nothing the product spawns inherits the step's own output pipe: a runner ends a step when the command has gone *and* that pipe has reached end of file, and one process left holding it keeps the step open with a zero exit code in it |
| start the flight recorder | second, so every later step is visible from off the host: blocked tasks, memory, disk, the kernel ring buffer and the newest logs, posted to a check run on the commit. Its posts stopping is the moment the host lost contact |
| assert containment | ⛔ inside a network namespace that holds loopback alone, through [`../scripts/capture/contained.sh`](../scripts/capture/contained.sh); the guard reads the namespace's own route table |
| capture | in that same kind of namespace, so the observer and the build reach each other over loopback and nothing else |
| upload the evidence bundle | the bundle and the install record together, `if-no-files-found: error` |
| upload the install logs | last, and `if: always()`, so a lane whose install refused still uploads its logs |

⛔ **The capture is contained, not the host, and the reason is measured.** This
workflow used to delete the host's default routes before the capture and put
them back after. The runner agent reaches GitHub over those routes, so from the
cut onward the job could not report, and on the runner images of 2026-09 onward
it never regained contact: runs 21 to 34 each showed *Install the client*
running until cancelled, with no log and no artifact, over a step that had
finished in seconds. `TODO/ci.md` under `CI-08` carries the runs that showed it.
[`../scripts/ci/check-workflow.sh`](../scripts/ci/check-workflow.sh) refuses a
job that neither cuts a route nor runs its guard and capture through
`contained.sh`, and refuses any job that cuts the host's route.
⭐ [`../.github/workflows/capture.yml`](../.github/workflows/capture.yml)'s
Linux job runs through `contained.sh` since 2026-10-14. Its Windows job is
contained at the firewall since 2026-10-15, because Windows has no network
namespace to run a process in, and the same check holds it to containing the
programs after the build, reading the rules back and proving them before it
captures.

⛔ **The evidence upload is fatal and the install-logs upload is not, and the
asymmetry is enforced.** A capture that measured a build and uploaded nothing
must be red, so that step declares `if-no-files-found: error` and carries no
`continue-on-error`; a `continue-on-error: true` two keys away would silently
undo it and leave a green run with no evidence anywhere. The install-logs upload
is a diagnostic aid and is non-fatal on purpose.

⚠ **Those two install constraints are new and nothing else enforces them**, so
[`../scripts/ci/check-workflow.sh`](../scripts/ci/check-workflow.sh) asserts both
as ordering cases. ⛔ Its capture block reads **every** `capture*.yml` in the
workflow directory rather than the one file it was written for: a rule over one
workflow is not a rule over the sibling that installs a stranger's binary, which
is the one-gated-door shape applied to a rule instead of to a code path.

⭐ **One job per adapter AND per route, and `fail-fast: false`.** Each is a
separate host acquiring one product through one route, so a matrix that stopped
at the first red would throw away measurements already taken on machines that are
about to be destroyed.

⛔ **One route per host is the design rather than the shape the matrix happened
to take.** `ACQ-03` compares what two routes installed, and two routes on one
machine means the second installs over the first: the digest comparison then has
one host's final state to look at rather than two independent acquisitions.
⚠ The default dispatch runs `["package"]` alone, which is what every dispatch so
far ran; adding `"release"` runs a second host per adapter.

⛔ **An artifact name carries the route, and it has to.** Two legs now share an
adapter and differ only in the route, so a name built from the adapter alone
would be two uploads under one name in one run - which the upload action refuses,
turning a green capture into a failed upload for a reason nothing in the step
says.

⭐ **The driver and the verifier are both somebody else's code.** `curl` is a
complete HTTP client and puts real bytes through the observer; `sha256sum -c`
and `Get-FileHash` test the digests the observer declared, rather than the
observer checking itself. ⛔ And the announce carries `key=<run-id>`, a token the
driver knows it sent, which the transcript must hold: everything else is
satisfied by a bundle of empty artifacts that verify against their own empty
digests.

## What this does not establish

⚠ **The guards prove they fire. They do not prove they are sufficient.** A host
can be non-disposable in ways neither models: a mounted network share, a
hypervisor snapshot restored between runs, a container sharing a writable layer
with its siblings. Those are not detected here, and
[`security/remote-ops.md`](security/remote-ops.md) carries what the operator
owns.
