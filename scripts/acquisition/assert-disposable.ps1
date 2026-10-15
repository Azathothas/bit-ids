# assert-disposable.ps1 - refuse to install or run a client on a Windows host
# somebody keeps.
#
# ⭐ THE WINDOWS HALF OF `bit-check assert-disposable`, which asks this file there;
# its Unix half was assert-disposable.sh until 2026-10-15 (`CI-11`). It exists because the sh
# half reads /proc/net/route and /etc/machine-id. Neither is on Windows, so a
# Windows capture had no boundary to run before the install and was refused
# outright. CI-03 owns the pair.
#
# ⛔ THIS RUNS BEFORE THE INSTALL. The run manifest already refuses to RECORD a
# capture whose host was not disposable, and that guard cannot stop one: by the
# time a manifest exists, an untrusted installer has run. A record-layer refusal
# is a report, not a boundary.
#
# -- ⭐ TWO INDEPENDENT GUARDS, AND WHY THEY ARE NOT ONE ---------------------
#
#   -Claim        the host has never been used for a capture before
#   -Egress       the capture's own programs cannot reach anything but loopback
#
# They fail for different reasons and neither implies the other. A fresh host
# with an open route leaks the capture onto the public network. A firewalled
# host that already ran a capture contaminates this one with the last one's
# state.
#
# -- ⛔ -Claim DETECTS THE FAILURE RATHER THAN TRUSTING THE CLAIM -------------
#
# The obvious design is a token the provisioner writes saying "this host is
# disposable". ⚠ That is a promise, and a promise fails silently: a runner
# misconfigured to persist its disk still carries the token. So this claims the
# host by WRITING a marker and refuses if one is already there. A second capture
# on one host means the host survived the first, whatever anything claimed.
#
# ⚠ THE MARKER LIVES WHERE A REAL TEARDOWN DESTROYS IT. ProgramData is the
# Windows counterpart of /var/lib: it survives a reboot, so a host that rebooted
# rather than being destroyed still reads as claimed. TEMP would not, and a
# guard cleared by a reboot reports a survived host as fresh.
#
# -- ⛔ -Egress READS THE FIREWALL, NOT THE ROUTING TABLE ----------------------
#
# ⚠ THIS IS WHERE THE TWINS PART, AND ON PURPOSE. The sh half reads the routing
# table of the network namespace the capture runs in, which holds loopback
# alone. Windows has no namespace, and cutting the host's default routes severed
# runners from GitHub (`TODO/ci.md`, `CI-08`), so the Windows capture is
# contained by an outbound block rule per executable it runs, written by
# scripts/capture/firewall-contain.ps1. This guard reads those rules back.
#
# ⛔ -Program NAMES WHAT MUST BE CONTAINED, and the guard refuses unless every
# one of them is covered: an enabled outbound block rule for exactly that path,
# narrowed by nothing (profile, address, protocol, interface, service, user),
# and every firewall profile on. A rule narrowed to one profile or one protocol
# blocks some traffic and reads as a block in any listing, which is why the
# filters are read and not only the rule. The paths are joined by `|`, which
# no Windows path can carry, because `pwsh -File` passes an array parameter as
# one string.
#
# ⚠ Loopback is not blocked by these rules: Windows Firewall does not filter
# loopback for a desktop process, so the observer and the driver still talk
# over 127.0.0.1. The capture itself proves that on every run.
#
# ⛔ THE FIREWALL SOURCE IS AN ARGUMENT, FOR THE REASON THE SH HALF GIVES its
# route source: -FirewallState takes a file of the lines the live reader below
# produces, so the runner test drives this on a host with no
# Get-NetFirewallRule at all, which is every Linux host this repository is
# developed on. A seam a test can use is a seam a misconfiguration can use, so
# a passing run prints which source it read.
#
# Usage:
#   pwsh -NoProfile -File scripts/acquisition/assert-disposable.ps1 -Claim <run-id>
#   pwsh -NoProfile -File scripts/acquisition/assert-disposable.ps1 -Egress -Program <path|path...> [-FirewallState <path>]
#   pwsh -NoProfile -File scripts/acquisition/assert-disposable.ps1 -Fingerprint
#   pwsh -NoProfile -File scripts/acquisition/assert-disposable.ps1 -Marker
#
# ⛔ -Marker PRINTS THE MARKER'S PATH AND IS THE ONLY DERIVATION OF IT, for the
# reason the sh half gives: a caller that composed ProgramData\bit-ids\host-claimed
# itself would go on reading the old place the day the state directory moves,
# and report a host nobody claimed as fresh. capture-run.ps1 is that caller.
#
# Exit codes: 0 the guard passed, 1 the guard refuses, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

# ⛔ PositionalBinding IS OFF, for the reason the gate runner gives: a stray
# expanded argument must fail to bind rather than land on the next free
# parameter.
[CmdletBinding(PositionalBinding = $false)]
param(
    [switch]$Claim,
    [switch]$Egress,
    [switch]$Fingerprint,
    [switch]$Marker,
    [string]$RunId = '',
    [string]$Program = '',
    [string]$FirewallState = '',
    [switch]$Help
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# ⛔ A NATIVE COMMAND'S EXIT CODE IS THE VERDICT HERE, SO IT MUST NOT THROW.
# This defaults to $true from PowerShell 7.5, which turns every non-zero exit
# into a terminating error under $ErrorActionPreference = 'Stop': a guard that
# REFUSES stops being a code a caller can read and becomes an exception nobody
# caught. docs/conventions/shell.md section 8.
$PSNativeCommandUseErrorActionPreference = $false

# ⛔ NO `break` INSIDE ForEach-Object, AND THAT IS NOT STYLE. `break` there does
# not stop the pipeline: it breaks the enclosing loop, and with none it unwinds
# the whole script, which exits 0. The first version did exactly that, so every
# misuse of this guard printed usage and reported success. `check-runner.ps1`
# caught it on its first run, over two cases written to check the opposite.
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

$stateDir = $env:BIT_IDS_STATE_DIR
if ([string]::IsNullOrEmpty($stateDir)) {
    $programData = $env:ProgramData
    if ([string]::IsNullOrEmpty($programData)) { $programData = '/var/lib' }
    $stateDir = Join-Path $programData 'bit-ids'
}
$markerPath = Join-Path $stateDir 'host-claimed'

# A value that differs between two hosts and survives within one.
#
# ⚠ SEVERAL INPUTS RATHER THAN ONE, which is the sh half's rule and holds here
# for the same reason: any single source is a coincidence away from colliding.
# The machine GUID is Windows' counterpart of /etc/machine-id, and the install
# date and computer name differ where a cloned image's GUID might not.
function Get-HostFingerprint {
    $parts = New-Object System.Collections.ArrayList
    try {
        $key = 'HKLM:\SOFTWARE\Microsoft\Cryptography'
        if (Test-Path -LiteralPath $key) {
            $guid = (Get-ItemProperty -LiteralPath $key -Name MachineGuid -ErrorAction SilentlyContinue).MachineGuid
            if ($guid) { [void]$parts.Add($guid) }
        }
    }
    catch { }
    try {
        $install = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion'
        if (Test-Path -LiteralPath $install) {
            $id = (Get-ItemProperty -LiteralPath $install -ErrorAction SilentlyContinue)
            if ($id.InstallDate) { [void]$parts.Add([string]$id.InstallDate) }
            if ($id.BuildLabEx) { [void]$parts.Add([string]$id.BuildLabEx) }
        }
    }
    catch { }
    # ⚠ These three exist on every platform PowerShell runs on, so a fingerprint
    # is never empty even where the registry is not. A guard whose fingerprint
    # is empty exits 2 below rather than claiming with nothing.
    [void]$parts.Add([System.Environment]::MachineName)
    [void]$parts.Add([System.Environment]::OSVersion.VersionString)
    [void]$parts.Add([System.Environment]::UserName)

    $text = ($parts -join "`n") + "`n"
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($text)
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        ($sha.ComputeHash($bytes) | ForEach-Object { $_.ToString('x2') }) -join ''
    }
    finally { $sha.Dispose() }
}

# The programs -Program names, split on `|`.
function Split-Programs([string]$Text) {
    @($Text -split '\|' | Where-Object { -not [string]::IsNullOrEmpty($_) })
}

# The one comparable spelling of a program path: environment variables
# expanded, `.` and `..` resolved. ⚠ The firewall stores a rule's path as it was
# written, and a rule written as %SystemRoot%\System32\curl.exe blocks the same
# file as C:\Windows\System32\curl.exe; comparing the strings as given would
# report that file uncovered, or a `..` spelling of another file covered.
function Get-ComparablePath([string]$Path) {
    [System.IO.Path]::GetFullPath([System.Environment]::ExpandEnvironmentVariables($Path))
}

# The firewall as lines, one per profile and one per outbound rule, in the shape
# -FirewallState fixtures are written in:
#
#   profile <TAB> name=Public <TAB> enabled=True
#   rule <TAB> enabled=True <TAB> direction=Outbound <TAB> action=Block <TAB> profile=Any
#        <TAB> local=Any <TAB> remote=Any <TAB> protocol=Any <TAB> interfacetype=Any
#        <TAB> interface=Any <TAB> service=Any <TAB> localuser=Any <TAB> program=<path>
#
# ⚠ ONE RULE IS ONE LINE, with `program` last because a path may carry a space
# and no Windows path carries a tab.
#
# Returns the lines, or $null when the firewall could not be read.
function Read-FirewallState {
    param([string]$Source)

    if (-not [string]::IsNullOrEmpty($Source)) {
        if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) { return $null }
        $read = Get-Content -LiteralPath $Source -ErrorAction SilentlyContinue
        if ($null -eq $read) { return $null }
        return , @($read)
    }

    if (-not (Get-Command Get-NetFirewallRule -ErrorAction SilentlyContinue)) { return $null }
    $lines = New-Object System.Collections.ArrayList
    try {
        # ⛔ THE ACTIVE STORE, which is what the host enforces. The persistent
        # store is what was written; a group policy can override it, and a
        # rule that exists there and not here blocks nothing.
        foreach ($zone in @(Get-NetFirewallProfile -PolicyStore ActiveStore -ErrorAction Stop)) {
            [void]$lines.Add("profile`tname=$($zone.Name)`tenabled=$($zone.Enabled)")
        }
        # ⚠ SilentlyContinue because a filter that matches nothing is an error
        # to this cmdlet. No rule read is a refusal below, never a pass.
        # ⚠ Outbound blocks only, because each rule costs seven filter reads
        # and an image carries hundreds of allow rules. The lines still carry
        # direction and action, and Test-Containment still asks both.
        $rules = @(Get-NetFirewallRule -PolicyStore ActiveStore -Direction Outbound `
                -Action Block -ErrorAction SilentlyContinue)
        foreach ($rule in $rules) {
            $app = $rule | Get-NetFirewallApplicationFilter -ErrorAction Stop
            $address = $rule | Get-NetFirewallAddressFilter -ErrorAction Stop
            $port = $rule | Get-NetFirewallPortFilter -ErrorAction Stop
            $type = $rule | Get-NetFirewallInterfaceTypeFilter -ErrorAction Stop
            $face = $rule | Get-NetFirewallInterfaceFilter -ErrorAction Stop
            $service = $rule | Get-NetFirewallServiceFilter -ErrorAction Stop
            $security = $rule | Get-NetFirewallSecurityFilter -ErrorAction Stop
            [void]$lines.Add(("rule`tenabled=$($rule.Enabled)`tdirection=$($rule.Direction)" +
                    "`taction=$($rule.Action)`tprofile=$($rule.Profile)" +
                    "`tlocal=$(@($address.LocalAddress) -join ',')" +
                    "`tremote=$(@($address.RemoteAddress) -join ',')" +
                    "`tprotocol=$($port.Protocol)`tinterfacetype=$($type.InterfaceType)" +
                    "`tinterface=$(@($face.InterfaceAlias) -join ',')" +
                    "`tservice=$($service.Service)`tlocaluser=$($security.LocalUser)" +
                    "`tprogram=$($app.Program)"))
        }
    }
    catch { return $null }
    return , @($lines)
}

# ⛔ THE FILTERS A COVERING RULE LEAVES AT `Any`. Each one, set to anything else,
# narrows the block to part of the traffic: one profile, one address, one
# protocol, one interface, one service or one user.
$script:Unnarrowed = @('profile', 'local', 'remote', 'protocol', 'interfacetype',
    'interface', 'service', 'localuser')

# Returns '' when every program is contained, else the first reason one is not.
function Test-Containment {
    param([string[]]$State, [string[]]$Programs)

    $profiles = 0
    $rules = New-Object System.Collections.ArrayList
    foreach ($line in $State) {
        $fields = @($line -split "`t")
        $kind = $fields[0]
        if ($kind -cne 'profile' -and $kind -cne 'rule') { continue }
        $map = @{}
        foreach ($field in ($fields | Select-Object -Skip 1)) {
            $at = $field.IndexOf('=')
            if ($at -gt 0) { $map[$field.Substring(0, $at)] = $field.Substring($at + 1) }
        }
        if ($kind -ceq 'profile') {
            $profiles++
            # ⛔ A PROFILE THAT IS OFF ENFORCES NO RULE AT ALL, so a host whose
            # rules are perfect and whose active profile is off is open.
            if ($map['enabled'] -cne 'True') { return "the $($map['name']) firewall profile is off" }
        }
        else { [void]$rules.Add($map) }
    }
    if ($profiles -eq 0) { return 'no firewall profile was read' }

    foreach ($one in $Programs) {
        $want = Get-ComparablePath $one
        $narrowed = ''
        $covered = $false
        foreach ($rule in $rules) {
            if (-not $rule.ContainsKey('program')) { continue }
            if (-not [string]::Equals((Get-ComparablePath $rule['program']), $want,
                    [System.StringComparison]::OrdinalIgnoreCase)) { continue }
            if ($rule['enabled'] -cne 'True' -or $rule['direction'] -cne 'Outbound' -or
                $rule['action'] -cne 'Block') { continue }
            $why = ''
            foreach ($key in $script:Unnarrowed) {
                if ($rule[$key] -cne 'Any') { $why = "$key=$($rule[$key])"; break }
            }
            if ($why) { $narrowed = $why; continue }
            $covered = $true
            break
        }
        if ($covered) { continue }
        if ($narrowed) { return "the block for $one is narrowed to $narrowed" }
        return "nothing blocks $one"
    }
    return ''
}

# ⚠ THE OUTER @() IS load-bearing UNDER Set-StrictMode. A pipeline that yields
# exactly one object yields the object, not a one-element array, so `.Count`
# throws and every invocation of this script exits 1 with a property error. It
# did, on the first run of all eight cases.
$modes = @(@($Claim, $Egress, $Fingerprint, $Marker) | Where-Object { $_ })
if ($modes.Count -ne 1) {
    Show-Usage | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 2
}

if ($Fingerprint) {
    Write-Output (Get-HostFingerprint)
    exit 0
}

if ($Marker) {
    Write-Output $markerPath
    exit 0
}

if ($Egress) {
    $read = if ([string]::IsNullOrEmpty($FirewallState)) { 'Get-NetFirewallRule' } else { $FirewallState }
    $programs = @(Split-Programs $Program)
    # ⛔ NOTHING NAMED IS NOT NOTHING TO CONTAIN. A guard asked about no program
    # would pass on every host, so a caller that lost its list is could-not-run.
    if ($programs.Count -eq 0) {
        [Console]::Error.WriteLine(
            'assert-disposable: -Egress needs -Program; a guard told to contain nothing establishes nothing')
        exit 2
    }
    foreach ($one in $programs) {
        if (-not [System.IO.Path]::IsPathRooted($one)) {
            [Console]::Error.WriteLine("assert-disposable: program [$one] is not an absolute path")
            exit 2
        }
    }
    $state = Read-FirewallState -Source $FirewallState
    if ($null -eq $state) {
        [Console]::Error.WriteLine(
            "assert-disposable: $read is unreadable, so egress could not be established. That is not a pass.")
        exit 2
    }
    $refusal = Test-Containment -State $state -Programs $programs
    if ($refusal) {
        [Console]::Error.WriteLine("assert-disposable: egress is open: $refusal")
        exit 1
    }
    Write-Output ("no way off this host for " + $programs.Count + " program(s) (read $read)")
    exit 0
}

# -Claim
if ($RunId -notmatch '^[a-z0-9-]+$') {
    [Console]::Error.WriteLine("assert-disposable: run id must be lowercase a-z0-9-: $RunId")
    exit 2
}

if (Test-Path -LiteralPath $markerPath) {
    [Console]::Error.WriteLine(
        'assert-disposable: this host already ran a capture, so it was not destroyed')
    exit 1
}

try { $null = New-Item -ItemType Directory -Force -Path $stateDir -ErrorAction Stop }
catch {
    [Console]::Error.WriteLine("assert-disposable: cannot create $stateDir")
    exit 2
}

$print = Get-HostFingerprint
if ([string]::IsNullOrEmpty($print)) {
    [Console]::Error.WriteLine('assert-disposable: the host fingerprint is empty')
    exit 2
}

# ⛔ CREATED EXCLUSIVELY. Two captures racing on one host must not both believe
# they claimed it, so the file is opened CreateNew and a second opener fails
# rather than truncating. This is `set -C` in the sh half.
#
# ⚠ NOT REFUTED, AND KEPT. The Test-Path check above answers first on every input
# a test can construct, so rewriting CreateNew to Create changes no observable
# behaviour and check-runner.ps1 stays green over it. It matters only in the
# window between that check and this open, which is exactly the race it exists
# for and which no test here can enter. A reader should not take an unreached
# guard for a proven one. Same shape and same treatment as `entries.sort()` in
# PUB-01.
$stamp = [System.DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
$body = "bit-ids/host-claim/1`nrun=$RunId`nclaimed_at=$stamp`nfingerprint=$print`n"
try {
    $stream = [System.IO.File]::Open(
        $markerPath, [System.IO.FileMode]::CreateNew,
        [System.IO.FileAccess]::Write, [System.IO.FileShare]::None)
    try {
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($body)
        $stream.Write($bytes, 0, $bytes.Length)
    }
    finally { $stream.Dispose() }
}
catch {
    [Console]::Error.WriteLine('assert-disposable: another capture claimed this host first')
    exit 1
}

Write-Output "$print (claimed $markerPath)"
exit 0
