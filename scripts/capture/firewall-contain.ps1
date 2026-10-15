# firewall-contain.ps1 - block every program a Windows capture runs from
# reaching anything off this host, without touching the host's own routes.
#
# ⭐ THE WINDOWS COUNTERPART OF contained.sh, AND NOT ITS TWIN. contained.sh
# runs the capture in a network namespace that holds loopback alone. Windows
# has no namespace, and the route cut that stood in for one severed runners
# from GitHub (`TODO/ci.md`, `CI-08`). So this writes one outbound block rule
# per executable the capture runs, and leaves the host's own traffic, the
# runner agent's included, alone. Settled by the operator on 2026-10-15.
#
# ⛔ IT WRITES AND DOES NOT JUDGE. Whether the capture is contained is the
# egress guard's answer, `assert-disposable.ps1 -Egress -Program`, which reads
# the rules back from the active store: a rule this wrote and a policy then
# overrode blocks nothing, and only a reading of what the host enforces says so.
#
# ⛔ THE PROGRAMS ARE ASKED FOR, NEVER COMPOSED: `capture-run.ps1 -Programs`
# names the executables a capture runs, and the workflow hands that list here
# and to the guard. A list written out in two places is a list that drifts, and
# a driver nobody blocked is a capture with a way off the host.
#
# ⚠ A rule covers the executable it names and nothing that executable starts
# under another name. The fixture capture's observer and `curl.exe` start
# nothing; a client that spawns a helper needs the helper named too.
#
# Usage:
#   pwsh -NoProfile -File scripts/capture/firewall-contain.ps1 -Program <path|path...>
#
# Exit codes: 0 every rule was written, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

[CmdletBinding(PositionalBinding = $false)]
param(
    [string]$Program = '',
    [switch]$Help
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# ⛔ A NATIVE COMMAND'S EXIT CODE IS THE VERDICT HERE, SO IT MUST NOT THROW.
# docs/conventions/shell.md section 8.
$PSNativeCommandUseErrorActionPreference = $false

# ⛔ NO `break` INSIDE ForEach-Object; it unwinds the script and exits 0.
function Show-Usage {
    $lines = @(Get-Content -LiteralPath $PSCommandPath | Select-Object -Skip 1)
    $out = New-Object System.Collections.ArrayList
    foreach ($line in $lines) {
        if ($line -notmatch '^#') { break }
        [void]$out.Add(($line -replace '^# ?', ''))
    }
    $out
}

if ($Help) { Show-Usage; exit 0 }

function Cannot([string]$Message) {
    [Console]::Error.WriteLine("firewall-contain: $Message")
    exit 2
}

# ⚠ Split on `|`, the guard's separator, because `pwsh -File` passes an array
# parameter as one string and no Windows path carries a `|`.
$programs = @($Program -split '\|' | Where-Object { -not [string]::IsNullOrEmpty($_) })
if ($programs.Count -eq 0) { Cannot '-Program is required; containing nothing is not containment' }
foreach ($one in $programs) {
    if (-not [System.IO.Path]::IsPathRooted($one)) { Cannot "program [$one] is not an absolute path" }
    # ⛔ A RULE FOR A PATH NOTHING RUNS FROM BLOCKS NOTHING. The observer is
    # built before this step, so a missing one is a step order gone wrong.
    if (-not (Test-Path -LiteralPath $one -PathType Leaf)) { Cannot "$one is not present" }
}

if (-not (Get-Command New-NetFirewallRule -ErrorAction SilentlyContinue)) {
    Cannot 'New-NetFirewallRule is not available; this host has no Windows Firewall to write to'
}

try {
    # ⛔ EVERY PROFILE ON, because a profile that is off enforces no rule, and
    # which profile the runner's network lands in is the image's choice. The
    # default outbound action stays Allow: the host's own traffic is not this
    # script's to stop.
    Set-NetFirewallProfile -Profile Domain, Private, Public -Enabled True -ErrorAction Stop
    foreach ($one in $programs) {
        $null = New-NetFirewallRule -DisplayName "bit-ids capture: $one" -Group 'bit-ids-capture' `
            -Direction Outbound -Action Block -Program $one -Profile Any -Enabled True `
            -ErrorAction Stop
        Write-Output "blocked $one"
    }
}
catch {
    Cannot ("the firewall refused a write: " + $_.Exception.Message)
}
exit 0
