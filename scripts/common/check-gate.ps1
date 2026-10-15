# check-gate.ps1 - run the whole local gate, in one command, and read every
# exit code from the process that produced it.
#
# ⭐ THE TWIN OF check-gate.sh, and the only one that runs at all on a native
# PowerShell session with no POSIX layer. It runs the .ps1 half of every pair,
# which is the half that host can actually execute.
#
# ⚠ IT IS NOT IN check-twins.sh's PAIR LIST, and that is deliberate rather than
# an omission. This runner INVOKES check-twins, so comparing the two runners
# from inside check-twins would recurse. scripts/README.md carries the same
# reasoning for why check-twins itself has no twin.
#
# The defect this exists to catch is a gate that is a LIST. A list run by hand
# is run in the order somebody recalls it. ⛔ The session that first wrote the
# sh half ran its gate five times and typed a different subset each time.
#
# ⭐ IT DELEGATES. It holds no rules of its own. Every verdict is some other
# script's, read unpiped.
#
# -- ⛔ A SKIPPED CHECK IS A SKIP, NEVER A PASS -----------------------------
#
# A runner that quietly dropped a check and printed green would be the row in
# docs/conventions/forbidden-patterns.md that reads *a step that exits 0 having
# done nothing it was asked to do*. A skip is counted, named, and printed on
# its own line. ⭐ -Strict makes a skip a failure, which is what a CI job should
# pass, since there the tools are installed on purpose and a skip means the
# install broke.
#
# -- ⛔ AND A CHECK THIS HOST CANNOT RUN IS NOT A CHECK THAT BROKE ------------
#
# ⛔ THIS LANE IS WHERE THAT DISTINCTION WAS MISSING AND IT COST THE FLAG.
# Six rows below are checks Windows genuinely cannot run, so -Strict would have
# refused every correct tree and the workflow ran without it. Measured on
# 2026-09-06: with check-project.ps1 rewritten to exit 2, this runner's own CI
# invocation still exited 0. A check that stopped running was indistinguishable
# from one that never could.
#
# ⭐ So a row is one of two kinds. A DECLARED unavailability is written here,
# with a reason and the entry that owns it, and prints as `n/a`. An OBSERVED
# skip is a check that ran and answered 2, or one whose file is gone, and
# prints as `SKIP`. -Strict refuses the second and permits the first.
#
# ⚠ -Fast counts as declared, because the caller asked for it rather than the
# host failing to supply it.
#
# ⛔ AND ZERO PASSES IS RED WHATEVER THE SKIPS SAY. The sh half produced
# exactly that on its first run, through a broken presence test: nine skips,
# zero failures, and a green verdict over nothing at all.
#
# Usage:
#   pwsh -NoProfile -File scripts/common/check-gate.ps1
#   pwsh -NoProfile -File scripts/common/check-gate.ps1 -Fast
#   pwsh -NoProfile -File scripts/common/check-gate.ps1 -Strict
#   pwsh -NoProfile -File scripts/common/check-gate.ps1 -Json
#   pwsh -NoProfile -File scripts/common/check-gate.ps1 -Rows
#
# ⭐ -Rows PRINTS THE ROW NAMES AND RUNS NOTHING. Nothing compared the two
# runners' row lists: this half declares each row by hand and the sh half derives
# most of its provers from a list, so one added to that list and forgotten here is
# simply absent from this lane, which stays green because it never hears of it.
# check-gate-rows.sh is what compares them, and it needs no gate run at all.
#
# Exit codes: 0 nothing failed, 1 something failed, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

# ⛔ PositionalBinding IS OFF. A stray expanded argument must fail to bind
# rather than land on the next free parameter. A sibling script in this
# directory committed under a fabricated author for exactly that reason.
[CmdletBinding(PositionalBinding = $false)]
param(
    [switch]$Json,
    [switch]$Fast,
    [switch]$Strict,
    [switch]$Rows
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# ⛔ A NATIVE COMMAND'S EXIT CODE IS THE VERDICT HERE, SO IT MUST NOT THROW.
# This defaults to $true from PowerShell 7.5, which turns every non-zero exit
# into a terminating error under $ErrorActionPreference = 'Stop': a guard that
# REFUSES stops being a code a caller can read and becomes an exception nobody
# caught. docs/conventions/shell.md section 8.
$PSNativeCommandUseErrorActionPreference = $false

# ⛔ RESOLVED FROM THIS SCRIPT'S OWN LOCATION, not from the working directory.
# A runner found by a relative path runs a different set depending on who
# called it.
$here = Split-Path -Parent $PSCommandPath

$pass = 0
$fail = 0
$skip = 0
$na = 0
# ⛔ NOT $rows, AND THE NAME IS THE WHOLE REASON. PowerShell variable names are
# case-insensitive, so a `[switch]$Rows` parameter and a local called `$rows` are
# ONE variable: the accumulator was assigned to the switch and every invocation
# failed to bind with "Cannot convert System.Collections.ArrayList to
# SwitchParameter". Measured on 2026-09-09 by adding -Rows, which is the third
# time this class has been found here - `$args` in
# docs/conventions/shell.md section 8, and `[switch]$Marker` against `$marker` in
# assert-disposable.ps1, which failed to bind in every mode.
$rowText = New-Object System.Collections.ArrayList
function Add-Row([string]$T) { [void]$rowText.Add('  ' + $T) }

# ⛔ DECLARED HERE, WITH A REASON, OR IT IS NOT DECLARED. The reason is the
# whole difference between this row and an observed skip: one is a fact about
# the platform that somebody wrote down and can be argued with, and the other
# is a check that stopped working.
function Add-Unavailable([string]$Name, [string]$Reason) {
    if ($Rows) { Write-Output $Name; return }
    Add-Row ('n/a   ' + $Name + '  (' + $Reason + ')')
    $script:na++
}

$logFile = Join-Path ([System.IO.Path]::GetTempPath()) ("checkgate." + $PID + ".log")

# ⛔ WHAT THE TREE LOOKED LIKE BEFORE ANY CHECK RAN. A check plants defects in
# disposable state and must leave the working tree exactly as it found it, and
# until 2026-09-09 nothing compared the two on either lane: two probe cases wrote
# their scratch files beside the file they were handed, two callers hand that a
# TRACKED path, and a gate run left four untracked files in the source tree.
#
# ⛔ A COMPARISON, NEVER A REQUIREMENT OF A CLEAN TREE. This runs while somebody
# is editing. ⚠ And it sees the run that makes the mess rather than the one
# after, because the second run reads the first run's droppings as its own
# `before`; a lane that starts from a fresh checkout fires every time.
$treeBefore = $null
$gitPresent = [bool](Get-Command git -CommandType Application -ErrorAction SilentlyContinue)
if ($gitPresent) {
    $probe = & git -C $here rev-parse --is-inside-work-tree 2>$null
    if ($LASTEXITCODE -eq 0 -and $probe -eq 'true') {
        $treeBefore = (& git -C $here status --porcelain 2>$null) -join "`n"
    }
}

# ⭐ THE PORTED RULES, AND THEY ARE THE SAME BINARY THIS LANE'S TWIN RUNS. CI-10.
# Each of these was a hand-written `.ps1` twin of an `.sh` half, kept in step by
# check-twins. ⛔ One implementation cannot drift from itself, so both halves are
# deleted rather than compared - and what this lane runs is now the same program
# rather than a second reading of the same rule.
#
# ⛔ THE EXIT CODE IS TAKEN FROM THE PROCESS, UNPIPED, which is what
# Invoke-Ported below does: `&` into a redirect, then $LASTEXITCODE on the next
# line. Piping into anything reports the pipeline's status, so a check that
# failed reads green, and that is the defect this repository is most emphatic
# about. ⚠ This named an `Invoke-Native` no version of this file defined, and an
# `Invoke-Check` that ran a `.ps1` twin; the last twin it ran was deleted on
# 2026-09-29, and the function went with it.
$goBin = Join-Path ([System.IO.Path]::GetTempPath()) ("bit-check." + $PID + $(if ($IsWindows) { '.exe' } else { '' }))
$goPresent = [bool](Get-Command go -CommandType Application -ErrorAction SilentlyContinue)
if ($goPresent -and -not $Rows) {
    $toolDir = Join-Path $here '..' '..' 'tools' 'check'
    Push-Location $toolDir
    & go build -o $goBin . *> $logFile
    Pop-Location
}

# ⚠ THE ROW LABEL AND THE CHECK NAME ARE TWO PARAMETERS, because one row is no
# longer one check: `check-no-secrets (public)` is the same check under a mode,
# and the label is what the row lists compare. ⛔ The extra arguments are NOT
# called $Args: that is an automatic variable inside a function and it silently
# swallows a parameter of that name, and PowerShell names are case-insensitive so
# $args collides too. docs/conventions/shell.md section 8.
function Invoke-Ported([string]$Name, [string]$Check = '', [string[]]$ExtraArgs = @()) {
    # ⛔ THE NAME COMES OUT OF THE CALL THAT WOULD HAVE RUN THE CHECK, so a row
    # this mode does not name is a row this runner does not run. A separate list
    # would be the value in two places the comparison exists to catch.
    if ($Rows) { Write-Output $Name; return }
    $target = if ($Check) { $Check } else { $Name }
    if (-not (Test-Path -LiteralPath $goBin -PathType Leaf)) {
        Add-Row ("SKIP  " + $Name + "  (tools/check did not build)")
        $script:skip++
        return
    }
    & $goBin $target @ExtraArgs *> $logFile
    $rc = $LASTEXITCODE
    switch ($rc) {
        0 { Add-Row ("✅ ok    " + $Name); $script:pass++ }
        2 {
            # ⚠ THE REASON IS THE CHECK'S OWN FIRST LINE, cut at 60, which is
            # what the sh runner prints and what `Invoke-Check` printed. This
            # said *could not run* alone, so moving `check-remote-items` here on
            # 2026-09-29 would have cost its row the *gh not found* that says why.
            $first = ''
            if (Test-Path -LiteralPath $logFile) {
                $first = (Get-Content -LiteralPath $logFile -TotalCount 1 -ErrorAction SilentlyContinue)
            }
            if ($first -and $first.Length -gt 60) { $first = $first.Substring(0, 60) }
            Add-Row ("SKIP  " + $Name + "  (" + $first + ")")
            $script:skip++
        }
        default {
            Add-Row ("❌ FAIL  " + $Name + "  (exit " + $rc + ")")
            $script:fail++
            # ⛔ THE TAIL, NOT THE HEAD. Every check prints its verdict last and
            # the mutation harnesses print dozens of passing rows first, so a
            # head excerpt of a red harness contains no failure at all. The sh
            # half carries the measurement that found this.
            if (-not $Json -and (Test-Path -LiteralPath $logFile)) {
                Get-Content -LiteralPath $logFile -ErrorAction SilentlyContinue |
                    Select-Object -Last 20 |
                    ForEach-Object { Write-Output ('          ' + $_) }
            }
        }
    }
}

# ⭐ EVERY `common/` CHECK THE GATE RUNS IS PORTED NOW. `check-project` joined
# this list on 2026-09-17 and the direct call below it is gone with the twin it
# called. ⚠ This line then said *every `common/` half* while this lane still ran
# `check-remote-items.ps1`, which is why it names what the gate RUNS. ⭐ And
# since 2026-09-29 no `common/` pair is left at all: `mine-repo`, a miner rather
# than a row, was the last, and it is `bit-check mine-repo` now.
foreach ($c in 'check-adapters', 'check-changelog', 'check-control-bytes',
                'check-docs', 'check-ignores', 'check-licences',
                'check-markers', 'check-no-secrets', 'check-one-home',
                'check-placeholders', 'check-project') {
    Invoke-Ported $c
}

# ⚠ --public is a DIFFERENT question from the default run, not a stricter one.
# Emails, absolute home paths and long hex are legitimate content in a private
# project, so this is a second call rather than a flag on the first.
# ⭐ AND BOTH LANES NOW SPELL THE FLAG THE SAME WAY, because both call the one
# binary. This lane used to pass `-Public` while the other passed `--public`,
# which is why the row LABEL is named after the question rather than built from
# the flag: a label built from the flag made the two runners' row lists differ on
# a row they both have.
Invoke-Ported 'check-no-secrets (public)' 'check-no-secrets' @('--public')

# ⭐ THE FIRST HARNESS TO LEAVE THE TWIN LAYER, 2026-09-29. This row ran
# `check-cache.ps1`, the first harness twin CI-07 wrote, and it was compared case
# for case against both halves before they were deleted. ⚠ It builds a Rust
# example, so on this lane it needs cargo as well as go; a host with neither
# answers could-not-run, which reads as a skip and never as a pass.
Invoke-Ported 'check-cache'

# ⭐ AND THE SECOND HARNESS, 2026-09-29. This row ran `check-catalogue.ps1`, and
# like the first it was compared case for case against both halves before they
# were deleted. With it gone, `store-lib.ps1` had no caller and went too.
Invoke-Ported 'check-catalogue'

# ⭐ AND A THIRD, 2026-10-14. check-corpus was declared here because its
# harness was sh alone; it is a bit-check row now, compared case for case
# against that half before it was deleted. CI-07.
Invoke-Ported 'check-corpus'

# ⭐ AND check-indexes, THE SAME DAY AND THE SAME WAY. CI-07.
Invoke-Ported 'check-indexes'

# ⭐ AND check-release. Its checksums are read by certutil here, Windows'
# own reader, because this lane has no sha256sum on a non-bash step's path. CI-07.
Invoke-Ported 'check-release'

# ⭐ AND check-formats, whose database reader is Python's sqlite3. CI-07.
Invoke-Ported 'check-formats'

# ⭐ AND check-staleness, which was declared here because its harness was sh;
# its subject never needed a POSIX feature. CI-07.
Invoke-Ported 'check-staleness'

# ⭐ AND check-connector, whose subject is stdlib Python and runs here as on
# the Linux lane; its harness was sh alone. CI-07.
Invoke-Ported 'check-connector'

# ⭐ AND THE LAST `common/` TWIN THIS LANE RAN, 2026-09-29. This row ran
# `check-remote-items.ps1`, compared case for case against both halves before
# they were deleted. ⚠ It NEEDS gh AND THE NETWORK, so it exits 2 on a machine
# without them and that reads as a skip rather than a pass. Correct: nothing was
# verified.
Invoke-Ported 'check-remote-items'

# ⭐ A REAL ROW NOW, AND IT USED TO BE A DECLARED GAP. The disposable-host guards
# were Linux-only because they read /proc/net/route and /etc/machine-id;
# assert-disposable.ps1 reads Get-NetRoute and the machine GUID, and this runs
# the harness that mutation-proves it. ⚠ Its route fixtures are files, so the
# logic is provable on any host; that Get-NetRoute's real output matches them is
# established by a Windows job rather than here.
$runner = Join-Path $here '..' 'acquisition' 'check-runner.ps1'
if ($Rows) { Write-Output 'check-runner' }
elseif (Test-Path -LiteralPath $runner -PathType Leaf) {
    & pwsh -NoProfile -File $runner *> $logFile
    $rc = $LASTEXITCODE
    switch ($rc) {
        0 { Add-Row '✅ ok    check-runner'; $pass++ }
        2 { Add-Row 'SKIP  check-runner  (could not run)'; $skip++ }
        default { Add-Row ('❌ FAIL  check-runner  (exit ' + $rc + ')'); $fail++ }
    }
}
else {
    Add-Row 'SKIP  check-runner  (not present)'
    $skip++
}

# ⛔ DECLARED FOR THE SAME REASON. check-store plants a symbolic link and a
# named pipe in a disposable tree, and neither is available to an unprivileged
# Windows session. ⚠ The rules they prove are
# not Linux-only and the Rust suite exercises every one of them on both lanes;
# what this lane does not do is plant them against a real filesystem.
#
# ⛔ EVERY ROW BELOW SAID `CI-03 residual`, AND THAT NAMED THE WRONG EVENT.
# CI-03 is the capture runner matrix; landing it does not give an unprivileged
# Windows session a named pipe, and it does not write a PowerShell half for any
# harness here. These close when somebody writes that half, and the reason now
# says so. ⚠ A documented gap whose reason is wrong is worse than an undocumented
# one, because --strict permits it forever on the strength of a sentence nobody
# re-read. Found by a claim audit while CI-03 was being closed.
Add-Unavailable 'check-store' 'its plants are a symlink and a named pipe; reconsider them, do not translate; CI-07 class B'
Add-Unavailable 'check-publish' 'its subject publish-data.sh has no PowerShell half; CI-07 class C'

Add-Unavailable 'check-access' 'its subject publish-data.sh has no PowerShell half; CI-07 class C'
Add-Unavailable 'check-bundle' 'its subject publish-data.sh has no PowerShell half, and it drives bit-check assemble-bundle from sh; CI-07 class C'
Add-Unavailable 'check-examples' 'it RUNS a document sh fenced blocks and there is no sh here; CI-07 class D'
Add-Unavailable 'check-handbook' 'it RUNS a document sh fenced blocks and there is no sh here; CI-07 class D'

# ⛔ AND THIS ONE IS THE ROW THAT WATCHES THIS LIST. check-gate-rows compares the
# names this runner declares against the ones the sh runner queues, so a prover
# added there and forgotten here stops being invisible. ⚠ It is declared for the
# ordinary reason - it is an sh harness - and a PowerShell half would have to run
# the sh runner anyway to ask the question, so what it would add is a second
# implementation of a comparison rather than a second platform's answer.
Add-Unavailable 'check-gate-rows' 'it runs the sh runner itself, so there is no half to write; CI-07 class D'

# ⛔ AND THIS ONE ASKS WHETHER THIS RUNNER PASSES THE FLAG IT SAYS IT DOES.
# check-public-row reads the --public flag out of BOTH runners and runs it, so a
# lane that quietly stopped asking the public question is caught rather than
# reported green by the default question. ⚠ It reads this file, which is why it
# is declared here rather than written twice: a PowerShell half would read the
# same two files and answer the same thing.
Add-Unavailable 'check-public-row' 'it reads both runners itself, so there is no half to write; CI-07 class D'

# ⛔ AND THIS ONE'S REASON IS AN ENTRY IN FLIGHT RATHER THAN A PLATFORM FACT.
# check-bitcheck plants defects against the Go binary that CI-10 is porting the
# checking layer into. ⭐ That binary runs natively on this platform - it is the
# whole point of the port - so the gap is the HARNESS, which is still `sh`.
# ⚠ The event that closes this row is therefore CI-10 finishing the port and
# moving the harness into Go beside its subject, not CI-07 writing a twin: a
# PowerShell twin of this file would be a new member of the layer this entry
# exists to delete.
Add-Unavailable 'check-bitcheck' 'an sh harness over a Go binary that does run here; closes when CI-10 moves the harness into Go'

# ⛔ AND THIS ONE'S REASON IS THE MOST SPECIFIC ON THE LANE, because a
# PowerShell half of it would not be the same check. check-step-bodies runs a
# workflow step's body through a POSIX PIPE and asks whether that pipe reached
# end of file once the body exited - which is how a runner decides a step is
# over. ⚠ A Windows half would need the same instrument over a named pipe and
# would be measuring a different operating system's answer to the same question,
# rather than the second implementation of one rule that check-twins compares.
# ⭐ It runs the `pwsh` half of its SUBJECT anyway: three of its cases execute
# capture.yml's Windows *Restore the route* block under GitHub's own wrapper
# form, which is why this row is declared and not a gap in what is proved.
Add-Unavailable 'check-step-bodies' 'it lifts a Linux-only workflow bodies and already runs the pwsh ones; CI-07 class D'

# ⛔ ITS SUBJECT IS EVERY `.sh` IN THE TREE, and the tools that read them are
# `shellcheck` and `shfmt`. Both run on this platform, so this is not a platform
# fact - it is that the equivalent question for THIS lane is a PowerShell linter
# over the `.ps1` files, which is a different tool and its own work. ⚠ Declared
# with its own reason rather than a neighbour's: a row whose reason names the
# wrong event closes on the wrong day.
Add-Unavailable 'check-shell' 'its subject is the sh scripts; a ps1 linter is its own work; CI-08'

# ⚠ AND THIS ONE IS DECLARED WHILE ITS SUBJECT IS EXERCISED ANYWAY, which is the
# third kind of reason on this lane. check-capture is an sh harness, so it cannot
# run here; what it drives includes capture-run.ps1, which the Windows job of the
# capture workflow runs on a real Windows host with no route table argument. That
# is a stronger control than this lane could give and it happens elsewhere.
Add-Unavailable 'check-capture' 'its subject PowerShell half exists and the capture workflow runs it; CI-07 class D'

# ⛔ ITS REASON IS NOT check-capture's, AND COPYING THAT ONE WOULD RECORD A GAP
# THAT CLOSES ON THE WRONG EVENT. check-capture is declared because its SUBJECT
# has a PowerShell half that runs somewhere else. capture-client.sh has no
# PowerShell half at all, and capture-client.yml is a Linux-only workflow, so
# nothing on any lane exercises a Windows client capture. ⚠ That closes when
# CI-07 writes the twin, not when a capture is dispatched.
Add-Unavailable 'check-capture-client' 'an sh harness whose subject has no PowerShell half yet; CI-07'

# ⛔ AND ITS REASON IS A THIRD ONE AGAIN. check-release-route drives
# resolve-release.sh, which is `sh`, over recorded listings; the decision it
# proves is in Rust and the Rust suite exercises it on both lanes. What this lane
# does not exercise is the shell that composes the three steps. ⚠ It closes when
# CI-07 writes that twin, and a Windows adapter is what would need one.
Add-Unavailable 'check-release-route' 'an sh harness whose subject has no PowerShell half yet; CI-07'

# ⛔ AND A FOURTH REASON. check-assemble's subject is a Rust example, which both
# lanes build and neither lane runs anywhere else; what has no PowerShell half is
# the harness that writes the synthetic lanes and plants in them. ⚠ It is
# `store-lib.sh`'s again - the shared library CI-07 names as its first step -
# rather than anything about the assembler, which is platform-independent.
Add-Unavailable 'check-assemble' 'a portable Rust subject; it closes when its harness is ported into bit-check; CI-10'
Add-Unavailable 'check-transmission' 'its subject transmission.sh is an sh adapter with no PowerShell half; CI-07 class C'


# ⚠ Its reason is `check-release-route`'s exactly: the decision it proves is in
# Rust and the Rust suite exercises it on both lanes; what this lane does not
# exercise is the shell that composes the fetch and the resolution.
Add-Unavailable 'check-source-route' 'an sh harness whose subject has no PowerShell half yet; CI-07'

# ⛔ AND A SIXTH REASON, WHICH IS A FACT ABOUT THE PLATFORM RATHER THAN A MISSING
# TWIN. check-rootless drives every release route under a uid that is not root,
# with `sudo` absent from PATH, into a prefix the current user owns. ⚠ Windows
# has no `sudo`, no uid to drop to and no `su`, so the comparison the harness is
# built on has nothing to remove - and `capture-client.yml`, the only workflow
# that installs a client, is Linux-only. ⭐ What would close this is a Windows
# client capture, which is `CLIENT-01`'s, not a PowerShell twin of this file.
Add-Unavailable 'check-rootless' 'its subject is a POSIX privilege boundary; no Windows capture installs a client yet; ACQ-06'

# ⭐ THE SLOW ONE, and ⚠ it is the one part of this gate that needs a POSIX
# shell: check-twins runs the sh half of every pair, so it cannot run on a host
# without one. That is reported as a skip, never as a pass.
# ⚠ THE TWO BRANCHES BELOW ARE DIFFERENT KINDS AND ARE COUNTED DIFFERENTLY.
# -Fast is the caller declining the check, so it is declared. A host with no
# POSIX shell is a capability this runner probed for and did not find, which is
# the same shape as a missing gh, so it is an observed skip and -Strict refuses
# it: nothing compared the two halves, and no flag was passed saying that was
# intended.
if ($Rows) {
    Write-Output 'check-twins'
}
elseif ($Fast) {
    Add-Unavailable 'check-twins' '-Fast'
}
elseif (-not (Get-Command sh -ErrorAction SilentlyContinue)) {
    Add-Row 'SKIP  check-twins  (no POSIX shell on this host)'
    $skip++
}
else {
    $twins = Join-Path $here 'check-twins.sh'
    if (Test-Path -LiteralPath $twins -PathType Leaf) {
        & sh $twins *> $logFile
        $rc = $LASTEXITCODE
        if ($rc -eq 0) { Add-Row '✅ ok    check-twins'; $pass++ }
        else { Add-Row ("❌ FAIL  check-twins  (exit " + $rc + ")"); $fail++ }
    }
    else { Add-Row 'SKIP  check-twins  (not present)'; $skip++ }
}

Remove-Item -LiteralPath $logFile -ErrorAction SilentlyContinue

# ⛔ AND THE ROW THIS RUNNER DECIDES FOR ITSELF IS NAMED IN -Rows TOO.
# tree-unchanged is not a check it invokes, so a rows mode that listed only the
# invocations would be short by exactly the row nothing else names.
if ($Rows) {
    Write-Output 'tree-unchanged'
    exit 0
}

# ⛔ AND THE TREE IS COMPARED AGAINST WHAT IT WAS, the same row the sh half
# carries. A harness whose scratch file lands in the working tree is one whose
# next run starts from a tree the last one wrote.
if ($null -ne $treeBefore) {
    $treeAfter = (& git -C $here status --porcelain 2>$null) -join "`n"
    if ($treeAfter -ceq $treeBefore) {
        Add-Row '✅ ok    tree-unchanged'
        $pass++
    }
    else {
        Add-Row '❌ FAIL  tree-unchanged  (a check moved the working tree)'
        $fail++
    }
}
else {
    Add-Row 'SKIP  tree-unchanged  (not a git work tree)'
    $skip++
}

$total = $pass + $fail + $skip + $na

# ⛔ A RUN THAT PASSED NOTHING IS NOT A GREEN RUN.
# ⚠ -Strict reads $skip and never $na. A declared row is a documented gap and
# refusing it would refuse every correct tree on this lane.
if ($pass -eq 0)               { $rc = 1 }
elseif ($Strict -and $skip -gt 0) { $rc = 1 }
elseif ($fail -gt 0)           { $rc = 1 }
else                           { $rc = 0 }

if ($Json) {
    Write-Output ('{"schema":"check-gate/2","total":' + $total + ',"passed":' + $pass +
                  ',"failed":' + $fail + ',"skipped":' + $skip +
                  ',"unavailable":' + $na +
                  ',"strict":' + $(if ($Strict) { 'true' } else { 'false' }) + '}')
    exit $rc
}

Write-Output ''
$rowText | ForEach-Object { Write-Output $_ }
Write-Output ''
Write-Output ("{0} checks: {1} passed, {2} failed, {3} skipped, {4} unavailable" -f $total, $pass, $fail, $skip, $na)

if ($skip -gt 0) {
    Write-Output '⚠ A SKIP IS NOT A PASS. Those checks did not run and nothing about'
    Write-Output 'their subject was verified. Pass -Strict to make a skip a failure.'
}
if ($na -gt 0) {
    Write-Output '⚠ An n/a row is a gap this runner declares, with the reason beside it.'
    Write-Output '-Strict permits those and refuses a SKIP, so read both numbers.'
}
if ($pass -eq 0) {
    Write-Output '❌ NOTHING RAN. Zero checks passed, so this is red whatever the skips say.'
}
elseif ($fail -gt 0) {
    Write-Output '❌ the gate is red.'
}
else {
    Write-Output '✅ nothing failed.'
    Write-Output '⚠ That is part (a) of the gate only. Driving the real thing and the'
    Write-Output 'deep reviews are the other two, and each is blind to what this catches.'
    Write-Output 'docs/methodology/gate.md.'
}
exit $rc
