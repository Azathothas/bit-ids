# check-runner.ps1 - prove the Windows disposable-host guards refuse what they
# exist to refuse.
#
# ⭐ THE TWIN OF check-runner.sh, and it exists because the guards stopped being
# Linux-only. CI-03 wrote assert-disposable.ps1; this is what says it works, and
# it is the reason `check-runner` is no longer a declared gap on the PowerShell
# lane.
#
# ⛔ EVERY CASE READS THE EXIT CODE FROM THE PROCESS THAT PRODUCED IT. pwsh's
# $LASTEXITCODE after `& pwsh -File ...` is that process's, and a pipeline's is
# not. This repository's oldest stated rule.
#
# ⚠ EVERY CLAIM GOES TO A SCRATCH DIRECTORY. The sh half's own header records
# what happens otherwise: a case run without the override claimed the real
# machine, passed once, and failed on every run afterwards. On a persistent
# runner that is a check which goes green the day it is written and red forever,
# for a reason that looks like the guard being broken rather than the test.
#
# ⭐ AND THE FIREWALL FIXTURES ARE WHY THIS RUNS ANYWHERE. assert-disposable.ps1
# takes -FirewallState so the firewall can be a file, which means the guard's
# logic is provable on a host with no Get-NetFirewallRule at all. What is NOT
# proved here is that the real cmdlets' output matches these fixtures; the
# capture workflow's own windows job is what establishes that, by running the
# guard with no -FirewallState on a real Windows host.
#
# Usage:
#   pwsh -NoProfile -File scripts/acquisition/check-runner.ps1
#   pwsh -NoProfile -File scripts/acquisition/check-runner.ps1 -Json
#
# Exit codes: 0 every guard refused its defect, 1 one did not, 2 could not run.

[CmdletBinding(PositionalBinding = $false)]
param([switch]$Json)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# ⛔ A NATIVE COMMAND'S EXIT CODE IS THE VERDICT HERE, SO IT MUST NOT THROW.
# This defaults to $true from PowerShell 7.5, which turns every non-zero exit
# into a terminating error under $ErrorActionPreference = 'Stop': a guard that
# REFUSES stops being a code a caller can read and becomes an exception nobody
# caught. docs/conventions/shell.md section 8.
$PSNativeCommandUseErrorActionPreference = $false

$here = Split-Path -Parent $PSCommandPath
$guard = Join-Path $here 'assert-disposable.ps1'
if (-not (Test-Path -LiteralPath $guard -PathType Leaf)) {
    [Console]::Error.WriteLine("check-runner: $guard is not present")
    exit 2
}

$work = Join-Path ([System.IO.Path]::GetTempPath()) ("checkrunner." + $PID)
$null = New-Item -ItemType Directory -Force -Path $work
try {
    $pass = 0
    $fail = 0
    $rows = New-Object System.Collections.ArrayList

    function Add-Pass([string]$Name) {
        [void]$rows.Add("  ✅ ok    $Name"); $script:pass++
    }
    function Add-Fail([string]$Name) {
        [void]$rows.Add("  ❌ FAIL  $Name"); $script:fail++
    }

    # ⛔ Unpiped. Output is redirected to a file and $LASTEXITCODE is read on the
    # next line.
    function Invoke-Guard {
        param([string[]]$Arguments)
        $log = Join-Path $work 'out'
        & pwsh -NoProfile -File $guard @Arguments *> $log
        return $LASTEXITCODE
    }

    # ⛔ A CASE MAY REQUIRE THE VERDICT AS WELL AS THE CODE, AND SOME MUST. Three
    # of this guard's refusals share exit 2, and two share exit 1, so a case
    # asserting the code alone passes when a different refusal fired. A mutation
    # pass found exactly that: blanking the mode check left both mode cases green,
    # because the Egress branch it then fell into also answers 2.
    function Expect {
        param([int]$Want, [string]$Name, [string[]]$Arguments, [string]$Saying = '')
        $got = Invoke-Guard -Arguments $Arguments
        $said = ''
        $log = Join-Path $work 'out'
        if (Test-Path -LiteralPath $log) {
            $said = (Get-Content -LiteralPath $log -Raw -ErrorAction SilentlyContinue)
        }
        if ($null -eq $said) { $said = '' }
        if ($got -ne $Want) { Add-Fail "$Name (wanted exit $Want, got $got)" }
        elseif ($Saying -and $said -notmatch [regex]::Escape($Saying)) {
            Add-Fail "$Name (exit $Want, but did not say '$Saying')"
        }
        else { Add-Pass $Name }
    }

    # --- forbidden egress ---------------------------------------------------
    #
    # ⛔ THE FIREWALL, NOT THE ROUTING TABLE, since 2026-10-15. The Windows
    # capture is contained by an outbound block rule per program it runs, so
    # these fixtures are firewall state in the lines the guard's live reader
    # writes, and every program named must be covered by one rule narrowed by
    # nothing, under profiles that are all on.
    $sep = [System.IO.Path]::DirectorySeparatorChar
    $observer = Join-Path $work 'observer.exe'
    $driver = Join-Path $work 'curl.exe'
    $both = "$observer|$driver"
    $zones = @("profile`tname=Domain`tenabled=True", "profile`tname=Private`tenabled=True",
        "profile`tname=Public`tenabled=True")
    function Get-RuleLine([string]$Path, [hashtable]$Over = @{}) {
        $fields = [ordered]@{
            enabled = 'True'; direction = 'Outbound'; action = 'Block'; profile = 'Any'
            local = 'Any'; remote = 'Any'; protocol = 'Any'; interfacetype = 'Any'
            interface = 'Any'; service = 'Any'; localuser = 'Any'
        }
        foreach ($key in $Over.Keys) { $fields[$key] = $Over[$key] }
        $parts = @('rule') + @($fields.Keys | ForEach-Object { "$_=$($fields[$_])" }) + @("program=$Path")
        $parts -join "`t"
    }
    $stateN = 0
    function New-State([string[]]$Lines) {
        $script:stateN++
        $path = Join-Path $work "firewall-$script:stateN"
        $Lines | Set-Content -LiteralPath $path -Encoding utf8NoBOM
        $path
    }

    $good = New-State ($zones + @((Get-RuleLine $observer), (Get-RuleLine $driver)))
    Expect 0 'every program blocked under every profile passes' @(
        '-Egress', '-Program', $both, '-FirewallState', $good) 'no way off this host for 2 program(s)'

    # ⛔ EVERY PROGRAM, NOT THE FIRST. A guard that stopped at one covered
    # program would pass a capture whose driver has a way out.
    $one = New-State ($zones + @((Get-RuleLine $observer)))
    Expect 1 'a program nothing blocks is refused' @(
        '-Egress', '-Program', $both, '-FirewallState', $one) "nothing blocks $driver"

    # ⛔ AND EVERY FILTER, ONE CASE EACH. A block narrowed to one profile,
    # address, protocol, interface, service or user blocks some traffic and reads
    # as a block in a listing; each key below is one way to leave the rest open.
    foreach ($key in @('profile', 'local', 'remote', 'protocol', 'interfacetype',
            'interface', 'service', 'localuser')) {
        $narrow = New-State ($zones + @((Get-RuleLine $observer @{ $key = 'Narrowed' }), (Get-RuleLine $driver)))
        Expect 1 "a block narrowed by $key is refused" @(
            '-Egress', '-Program', $both, '-FirewallState', $narrow) "narrowed to $key=Narrowed"
    }

    # A rule that is off, that allows, or that is inbound is not an outbound block.
    foreach ($case in @(@('enabled', 'False'), @('action', 'Allow'), @('direction', 'Inbound'))) {
        $notBlock = New-State ($zones + @((Get-RuleLine $observer @{ $case[0] = $case[1] }), (Get-RuleLine $driver)))
        Expect 1 "a rule with $($case[0])=$($case[1]) blocks nothing" @(
            '-Egress', '-Program', $both, '-FirewallState', $notBlock) "nothing blocks $observer"
    }

    # ⚠ A NARROWED RULE BESIDE A COVERING ONE IS COVERED. A guard that gave up at
    # the first rule for a path would refuse a host whose image ships a narrow
    # rule of its own.
    $mixed = New-State ($zones + @((Get-RuleLine $observer @{ remote = 'LocalSubnet' }),
            (Get-RuleLine $observer), (Get-RuleLine $driver)))
    Expect 0 'a covering rule beside a narrowed one covers' @(
        '-Egress', '-Program', $both, '-FirewallState', $mixed)

    # ⛔ ONE PROFILE OFF ENFORCES NOTHING, whatever its rules say.
    $off = New-State (@($zones[0], $zones[1], "profile`tname=Public`tenabled=False") +
        @((Get-RuleLine $observer), (Get-RuleLine $driver)))
    Expect 1 'a firewall profile that is off is refused' @(
        '-Egress', '-Program', $both, '-FirewallState', $off) 'the Public firewall profile is off'
    $none = New-State @((Get-RuleLine $observer), (Get-RuleLine $driver))
    Expect 1 'no profile read is refused' @(
        '-Egress', '-Program', $both, '-FirewallState', $none) 'no firewall profile was read'

    # ⚠ ONE FILE, SPELLED ANOTHER WAY, IS COVERED. The firewall keeps a rule's
    # path as written, so an environment variable, a `..` or another case names
    # the same file and must compare as it.
    $env:BIT_IDS_FW_DIR = $work
    try {
        $spelled = New-State ($zones + @(
                (Get-RuleLine ("%BIT_IDS_FW_DIR%" + $sep + 'observer.exe')),
                (Get-RuleLine (Join-Path (Join-Path $work 'sub') ('..' + $sep + 'CURL.EXE')))))
        Expect 0 'a path spelled through a variable, a .. or another case is covered' @(
            '-Egress', '-Program', $both, '-FirewallState', $spelled)
    }
    finally { Remove-Item Env:\BIT_IDS_FW_DIR -ErrorAction SilentlyContinue }

    # ⛔ NOTHING NAMED, A RELATIVE NAME, OR AN UNREADABLE STATE IS EXIT 2. Not
    # knowing is not a pass, and a guard asked about nothing passes everywhere.
    Expect 2 'no program named is not a pass' @('-Egress', '-FirewallState', $good) 'needs -Program'
    Expect 2 'a relative program is not a pass' @(
        '-Egress', '-Program', 'curl.exe', '-FirewallState', $good) 'is not an absolute path'
    Expect 2 'an unreadable firewall state is not a pass' @(
        '-Egress', '-Program', $both, '-FirewallState', (Join-Path $work 'no-such-state')) 'could not be established'

    # --- persistent state ---------------------------------------------------
    $fresh = Join-Path $work 'host-a'
    $env:BIT_IDS_STATE_DIR = $fresh
    try {
        $first = Invoke-Guard -Arguments @('-Claim', '-RunId', 'capture-0001')
        if ($first -eq 0) { Add-Pass 'a fresh host claims' }
        else { Add-Fail "a fresh host claims (exit $first)" }

        # ⛔ TWO INDEPENDENT REFUSALS GUARD A SECOND CLAIM, AND THEY MASKED EACH
        # OTHER. The marker check and the exclusive create both answer exit 1, so
        # deleting either left this case green. The message is what separates
        # them: this asserts the marker check fired, and the exclusive create is
        # the one below.
        Expect 1 'a second capture on one host is refused' `
            @('-Claim', '-RunId', 'capture-0002') 'already ran a capture'

        # ⚠ Destroying the marker is what a real teardown does. This shows the
        # NEXT job is unblocked once it happens, which is the other half of the
        # same guard: it must refuse a survived host without refusing every host
        # forever.
        Remove-Item -LiteralPath $fresh -Recurse -Force
        $third = Invoke-Guard -Arguments @('-Claim', '-RunId', 'capture-0003')
        if ($third -eq 0) { Add-Pass 'a torn-down host claims again for the next job' }
        else { Add-Fail "a torn-down host claims again for the next job (exit $third)" }

        # ⚠ A run id that is not a slug is exit 2, not a claim under a name the
        # record could not carry.
        $bad = Join-Path $work 'host-b'
        $env:BIT_IDS_STATE_DIR = $bad
        Expect 2 'a run id that is not a slug is refused' @('-Claim', '-RunId', 'Capture 0001') 'run id must be lowercase'

        # ⛔ AND TWO MODES AT ONCE IS REFUSED. A caller that passed both switches
        # would otherwise get whichever branch is written first, silently.
        Expect 2 'two modes at once is refused' @('-Claim', '-RunId', 'x', '-Egress') 'assert-disposable.ps1 -Claim'
        Expect 2 'no mode at all is refused' @() 'assert-disposable.ps1 -Claim'
    }
    finally {
        Remove-Item Env:\BIT_IDS_STATE_DIR -ErrorAction SilentlyContinue
    }

    # --- the marker path, which is the only derivation of it ----------------
    #
    # ⛔ ASKED FOR, NEVER COMPOSED. capture-run.ps1 reads this to find out
    # whether the host was claimed and by which run. ⚠ The case is not "it
    # printed something": it claims into a scratch directory and requires the
    # file this mode NAMES to be the one that appeared. A mode returning a
    # plausible path nothing writes to would pass any weaker check.
    #
    # ⚠ ADDING THIS SWITCH BROKE THE WHOLE GUARD ONCE. `[switch]$Marker` and the
    # existing `$marker` local were one variable, because PowerShell names are
    # case-insensitive, so every invocation in every mode failed to bind a
    # string to a SwitchParameter. The local is $markerPath now.
    $marked = Join-Path $work 'host-c'
    $env:BIT_IDS_STATE_DIR = $marked
    try {
        $markerLog = Join-Path $work 'marker'
        & pwsh -NoProfile -File $guard -Marker *> $markerLog
        $markerRc = $LASTEXITCODE
        $markerPath = (Get-Content -LiteralPath $markerLog -TotalCount 1 -ErrorAction SilentlyContinue)
        if ($markerRc -eq 0 -and $markerPath -and -not (Test-Path -LiteralPath $markerPath)) {
            Add-Pass 'the marker path is reported before anything is claimed'
        }
        else {
            Add-Fail "the marker path is reported before anything is claimed (exit $markerRc)"
        }

        $claimed = Invoke-Guard -Arguments @('-Claim', '-RunId', 'capture-0004')
        if ($claimed -eq 0 -and $markerPath -and (Test-Path -LiteralPath $markerPath -PathType Leaf)) {
            Add-Pass '-Marker names the file -Claim actually wrote'
        }
        else {
            Add-Fail "-Marker names the file -Claim actually wrote (claim exit $claimed)"
        }

        Expect 2 '-Marker with another mode is refused' @('-Marker', '-Fingerprint') 'assert-disposable.ps1 -Claim'
    }
    finally {
        Remove-Item Env:\BIT_IDS_STATE_DIR -ErrorAction SilentlyContinue
    }

    # The fingerprint the next job compares against.
    $log = Join-Path $work 'print'
    & pwsh -NoProfile -File $guard -Fingerprint *> $log
    $printRc = $LASTEXITCODE
    $print = (Get-Content -LiteralPath $log -TotalCount 1 -ErrorAction SilentlyContinue)
    if ($printRc -eq 0 -and $print -match '^[0-9a-f]{64}$') {
        Add-Pass 'the host fingerprint is a sha256'
    }
    else {
        Add-Fail "the host fingerprint is not a sha256 (exit $printRc)"
    }

    $total = $pass + $fail
    if ($Json) {
        # ⚠ Built by interpolation rather than with -f. A .NET format string's
        # {0} is indistinguishable from an unfilled placeholder to
        # check-placeholders, which refused the first version of this line. The
        # guard is right to be blunt about that shape, so the line avoids it.
        Write-Output ('{"schema":"check-runner/1","total":' + $total +
            ',"passed":' + $pass + ',"failed":' + $fail +
            ',"fingerprint":"' + $print + '"}')
        if ($fail -gt 0) { exit 1 }
        exit 0
    }

    Write-Output ''
    $rows | ForEach-Object { Write-Output $_ }
    Write-Output ''
    Write-Output ("$total guard case(s): $pass passed, $fail failed")
    Write-Output ("host fingerprint for the next job: $print")
    # ⛔ Zero cases passing is not a green run, whatever the failure count says.
    if ($pass -eq 0) { Write-Output '❌ NOTHING RAN.'; exit 1 }
    if ($fail -gt 0) { Write-Output '❌ a guard did not refuse what it exists to refuse.'; exit 1 }
    Write-Output '✅ every guard refused its own defect.'
    exit 0
}
finally {
    Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}
