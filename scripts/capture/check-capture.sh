#!/bin/sh
# check-capture.sh - prove the capture runner refuses what it exists to refuse.
#
# ⛔ A CAPTURE IS WHERE THIS PROJECT WILL RUN SOMEBODY ELSE'S BINARY, and
# `capture-run` is what stands between that and a host with a route off it. Its
# guards are the last ones to fire before that process starts, so they are
# mutation-proven here rather than trusted.
#
# ⚠ WILL, not does. Nothing installs a client yet: `CLIENT-01` owns that, and
# what runs today is this project's own observer driven by `curl`. The guards
# are written for the harder case and proved on the easier one.
#
# -- ⭐ THE CONDITIONS ARE CONSTRUCTED, NOT PLANTED INTO THE SCRIPT -----------
#
# Every case here builds the host state the guard exists to catch: no claim, a
# claim naming another run, a route table with a default route, an unbuilt
# observer, an output directory that already holds a run. That is stronger than
# editing the script and re-running it, because it exercises the guard through
# the same door a real capture would come through.
#
# -- ⭐ EXCEPT FOR THE OBSERVER'S OWN CONTRACT, WHICH NEEDS A FIXTURE ---------
#
# Some refusals are about what the observer REPORTED, and no host state can
# provoke those. A STUB OBSERVER does: it runs the real one, so the socket and
# the driver are real, and then breaks exactly one term of the contract on the
# way out. ⚠ The `lost-token` mode recomputes the digest it prints, because
# otherwise the digest check fires first and the token check is never reached.
# Two guards answering one input is how one of them stops being tested.
#
# -- ⛔ EVERY SHARED EXIT CODE IS SEPARATED BY ITS MESSAGE --------------------
#
# The runner has many more refusals than it has exit codes, so most of these
# cases share one with several others. A case asserting the code alone passes
# when a different refusal fired, which is the defect the Windows guard pair
# shipped and `check-runner.ps1` found. ⚠ No count of them is written here: it
# was, and it was wrong within the hour, because the count is a property of the
# script rather than of this comment. Every case names what the runner said.
#
# -- ⭐ AND THE TWO HALVES ARE COMPARED, WHICH check-twins CANNOT DO ----------
#
# `check-twins.sh` pairs the `common/` checks, and these are not there: they
# take a host and a running process rather than a tree. So the comparison lives
# here. It compares the ATTESTATION KEY SET, because a field in one twin and not
# the other is drift no value comparison can see, and then the facts that
# describe one run. ⚠ It does NOT compare the values that must differ: the two
# halves verify with different readers, on different platform strings, and a
# twin reporting the same value there would be describing a run that did not
# happen.
#
# Usage:
#   sh scripts/capture/check-capture.sh
#   sh scripts/capture/check-capture.sh --json
#
# Exit codes: 0 every guard refused its defect, 1 one did not, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

JSON=0
while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON=1 ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    *)
      printf 'check-capture: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# ⚠ ME is set here and read by store-lib.sh, which this sources on the next
# line. shellcheck cannot see across a source it was not told to follow.
# shellcheck disable=SC2034
ME=check-capture
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo curl sha256sum awk sed od

RUNNER="$ROOT/scripts/capture/capture-run.sh"
GUARD="$ROOT/scripts/acquisition/assert-disposable.sh"
for f in "$RUNNER" "$GUARD"; do
  [ -f "$f" ] || {
    printf 'check-capture: %s is not present\n' "$f" >&2
    exit 2
  }
done

# ⛔ THE OBSERVER IS BUILT HERE AND NOT BY THE RUNNER, which is the same
# contract the workflow keeps: the build happens while there is still a network.
# store_build resolves CARGO_TARGET_DIR rather than composing target/, because
# composing it made five provers exit 2 on a host with that variable exported
# and report nothing at all.
if ! cargo build --manifest-path "$ROOT/Cargo.toml" -p bit-ids-lab --locked \
  --example evidence-bundle >/dev/null 2>&1; then
  printf 'check-capture: cannot build the evidence-bundle example\n' >&2
  exit 2
fi
OBSERVER="${CARGO_TARGET_DIR:-$ROOT/target}/debug/examples/evidence-bundle"
[ -x "$OBSERVER" ] || {
  printf 'check-capture: %s is not executable after a successful build\n' "$OBSERVER" >&2
  exit 2
}

WORK=$(store_workdir checkcapture) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⚠ SHORT ON PURPOSE, AND THE COST OF LENGTHENING IT IS NOW MEASURED. The runner
# waits on the observer's own line rather than on a delay, so a longer deadline
# costs only gate time - but "only gate time" turned out to be steep, because
# several cases here are ones the deadline itself has to end. Measured on
# 2026-09-09: this harness is 45 seconds at `3` and **79 at `6`**, so roughly
# eleven seconds of gate per second of deadline, and `check-capture-client` is 48
# at `5` and **168 at `20`**.
#
# ⛔ SO A LONGER DEADLINE IS NOT THE FIX FOR A LOADED MACHINE, and it was tried.
# On 2026-09-09 the gate first ran its checks concurrently and this harness
# reported *a run that recorded no bytes is refused (exit 1, but did not say
# 'recorded no bytes at all')* - a refusal that arrived for a reason the case had
# not planted - while the same harness passed alone on the same tree minutes
# later. Raising the bound fixed that and cost more gate time than the whole
# concurrency saved. ⭐ `check-gate.sh` runs the two socket harnesses apart from
# the saturating batch instead.
SECS=3
RUN=capture-0001

# The two routing tables, in the two formats the two guards read.
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-loopback"
printf 'lo\t0000007F\t00000000\t0001\t0\t0\t0\t000000FF\t0\t0\t0\n' >>"$WORK/route-loopback"
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-open"
printf 'eth0\t00000000\t010200C0\t0003\t0\t0\t0\t00000000\t0\t0\t0\n' >>"$WORK/route-open"

# -- the stub observer --------------------------------------------------------
#
# ⚠ It streams the real observer's output rather than buffering it, because the
# runner reads the endpoint line WHILE the observer is still serving. A stub
# that printed everything at the end would make the runner dial a port nothing
# is listening on any more, and every case below would then report the driver's
# refusal instead of its own.
#
# ⛔ THE SPLITTER IS `read`, NOT `awk`, AND THAT IS MEASURED RATHER THAN
# STYLISTIC. mawk reads its INPUT in blocks, so `{ print; fflush() }` flushed an
# output it had not produced yet: with awk in this pipeline the log was still
# empty a second into a five-second run, and with `cat` it held three lines.
# `fflush` is about the wrong end of the pipe.
STUB="$WORK/stub-observer.sh"
cat >"$STUB" <<'STUB'
#!/bin/sh
set -u
BUNDLE="$1"
SECS="$2"
ROWS="$BIT_IDS_STUB_ROWS"
: >"$ROWS"
"$BIT_IDS_STUB_REAL" "$BUNDLE" "$SECS" 2>&1 | while IFS= read -r line; do
  case "$line" in
    'evidence '*) printf '%s\n' "$line" >>"$ROWS" ;;
    *) printf '%s\n' "$line" ;;
  esac
done
TRANSCRIPT="$BUNDLE/tracker-http.transcript.json"
case "$BIT_IDS_STUB_MODE" in
  pass) cat "$ROWS" ;;
  no-evidence) : ;;
  bad-digest)
    sed 's/sha256:[0-9a-f]*/sha256:0000000000000000000000000000000000000000000000000000000000000000/' "$ROWS"
    ;;
  lost-token)
    SAFE=$(printf '%s' "$BIT_IDS_STUB_TOKEN_HEX" | tr '0-9a-f' '3')
    sed -i "s/$BIT_IDS_STUB_TOKEN_HEX/$SAFE/" "$TRANSCRIPT"
    NEW=$(sha256sum "$TRANSCRIPT" | cut -d' ' -f1)
    awk -v NEW="$NEW" '{
      if ($3 == "tracker-http.transcript.json") sub(/sha256:[0-9a-f]*/, "sha256:" NEW)
      print
    }' "$ROWS"
    ;;
  *) cat "$ROWS" ;;
esac
exit "$BIT_IDS_STUB_RC"
STUB
chmod +x "$STUB"

# -- the mid-stream stub ------------------------------------------------------
#
# ⛔ A SECOND STUB, BECAUSE TWO OF THE RUNNER'S TERMS ARE READ FROM LINES THE
# OBSERVER PRINTS WHILE IT IS STILL SERVING. The stub above can only change what
# is printed after the run, and a `no-segments` mode written there fired the
# EVIDENCE guard instead: the runner checks the segment count first, so removing
# the rows removed the wrong thing. That is two guards over one input with the
# earlier one masking the later, found by this harness on its own first run.
#
# It substitutes rather than only dropping, so `segments: 0` and a missing
# `segments:` line are separate cases. They are separate refusals in the runner
# and a stub that could only drop would leave one of them untested.
STUB_FILTER="$WORK/stub-filter.sh"
cat >"$STUB_FILTER" <<'FILTER'
#!/bin/sh
set -u
"$BIT_IDS_STUB_INNER" "$1" "$2" 2>&1 | while IFS= read -r line; do
  case "$line" in
    "$BIT_IDS_STUB_DROP "*)
      [ -z "$BIT_IDS_STUB_SUB" ] || printf '%s\n' "$BIT_IDS_STUB_SUB"
      ;;
    *) printf '%s\n' "$line" ;;
  esac
done
exit "$BIT_IDS_STUB_RC"
FILTER
chmod +x "$STUB_FILTER"

TOKEN_HEX=$(printf 'key=%s' "$RUN" | od -An -tx1 | tr -d ' \n')

# -- running one case ---------------------------------------------------------
#
# ⛔ UNPIPED, AND $? READ ON THE NEXT LINE. Piping the runner into anything
# reports the pipeline's status, so a runner that failed to refuse reads as
# having refused. This repository's oldest stated rule.
STATE="$WORK/state"
OUTS=0

# ⚠ THE OBSERVER IS A PARAMETER RATHER THAN A FIXED ARGUMENT WITH AN OVERRIDE
# APPENDED. sh takes the last of two `--observer` flags and PowerShell refuses
# the pair outright, so a helper that appended one would have made the two
# halves' cases mean different things. The PowerShell binder said so on the
# first run.
sh_case() { # want-code  saying  name  observer  extra-args...
  _want="$1"
  _saying="$2"
  _name="$3"
  _observer="$4"
  shift 4
  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" \
    --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$_observer" \
    --seconds "$SECS" "$@" >"$WORK/out" 2>"$WORK/err"
  _got=$?
  if [ "$_got" != "$_want" ]; then
    fail "sh   $_name (wanted exit $_want, got $_got)"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  elif [ -n "$_saying" ] && ! grep -q -F -e "$_saying" "$WORK/err"; then
    fail "sh   $_name (exit $_want, but did not say '$_saying')"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  else
    pass "sh   $_name"
  fi
}

# -- the probe's own guards ---------------------------------------------------
store_probe_guards "$RUNNER" \
  "the host was never claimed" "capture-run"

# -- 1. the control, before anything is broken --------------------------------
#
# ⛔ A GUARD THAT REFUSES EVERYTHING IS NOT A GUARD. Every refusal below is
# worthless unless the clean case is accepted, and this is where the whole path
# actually runs: the real observer binds, curl puts real bytes through it, the
# bundle is written, and sha256sum verifies it.
rm -rf "$STATE"
BIT_IDS_STATE_DIR="$STATE" sh "$GUARD" --claim "$RUN" >/dev/null 2>&1 || {
  printf 'check-capture: could not claim a scratch host\n' >&2
  exit 2
}
sh_case 0 "" "a claimed, contained host captures and verifies" \
  "$OBSERVER" --route-table "$WORK/route-loopback"
CONTROL_OUT="$WORK/out-$OUTS"

# -- 2. the host state guards -------------------------------------------------
sh_case 1 "egress is open" "an open default route is refused" \
  "$OBSERVER" --route-table "$WORK/route-open"
sh_case 2 "egress could not be established" "an unreadable route table is not a pass" \
  "$OBSERVER" --route-table "$WORK/no-such-table"

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id capture-0002 --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "is claimed by run [$RUN], not [capture-0002]" "$WORK/err"; then
  pass "sh   a host claimed by another run is refused"
else
  fail "sh   a host claimed by another run is refused (exit $_rc)"
fi

# ⚠ An EMPTY state directory, not the populated one. This is the step-order
# defect: a workflow that dropped its claim step leaves the capture running on a
# host nothing ever claimed, and the marker check is the only thing that sees it.
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$WORK/unclaimed" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "the host was never claimed" "$WORK/err"; then
  pass "sh   an unclaimed host is refused"
else
  fail "sh   an unclaimed host is refused (exit $_rc)"
fi

# -- 3. the argument guards ---------------------------------------------------
sh_case 2 "is not executable" "an observer that was never built is could-not-run" \
  "$WORK/no-such-binary" --route-table "$WORK/route-loopback"

BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$CONTROL_OUT" \
  --observer "$OBSERVER" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "already exists" "$WORK/err"; then
  pass "sh   a capture never writes into another run's evidence"
else
  fail "sh   a capture never writes into another run's evidence (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "Capture 0001" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --seconds "$SECS" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "run id must be lowercase" "$WORK/err"; then
  pass "sh   a run id that is not a slug is refused"
else
  fail "sh   a run id that is not a slug is refused (exit $_rc)"
fi

sh "$RUNNER" --nonsense >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "unknown argument" "$WORK/err"; then
  pass "sh   an unknown argument is refused"
else
  fail "sh   an unknown argument is refused (exit $_rc)"
fi

# -- 4. the observer contract, through the stub -------------------------------
#
# ⚠ Each mode breaks ONE term. A stub that broke two would leave the case
# reporting whichever guard is written first, and the other untested.
stub_case() { # mode  rc  saying  name
  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$STATE" \
    BIT_IDS_STUB_REAL="$OBSERVER" \
    BIT_IDS_STUB_ROWS="$WORK/rows-$OUTS" \
    BIT_IDS_STUB_MODE="$1" \
    BIT_IDS_STUB_RC="$2" \
    BIT_IDS_STUB_TOKEN_HEX="$TOKEN_HEX" \
    sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB" \
    --seconds "$SECS" --route-table "$WORK/route-loopback" \
    >"$WORK/out" 2>"$WORK/err"
  _got=$?
  if [ "$_got" != 1 ]; then
    fail "sh   $4 (wanted exit 1, got $_got)"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  elif ! grep -q -F -e "$3" "$WORK/err"; then
    fail "sh   $4 (exit 1, but did not say '$3')"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  else
    pass "sh   $4"
  fi
}

# ⭐ THE STUB IS ITSELF CONTROLLED. In `pass` mode it reproduces the real
# observer's contract exactly, so a stub that had simply broken the run would
# make every mode below refuse for the wrong reason and read as five guards
# working.
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" \
  BIT_IDS_STUB_REAL="$OBSERVER" \
  BIT_IDS_STUB_ROWS="$WORK/rows-$OUTS" \
  BIT_IDS_STUB_MODE=pass \
  BIT_IDS_STUB_RC=0 \
  BIT_IDS_STUB_TOKEN_HEX="$TOKEN_HEX" \
  sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB" \
  --seconds "$SECS" --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 0 ]; then
  pass "sh   the stub observer in pass mode is accepted"
else
  fail "sh   the stub observer in pass mode is accepted (exit $_rc)"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
fi

stub_case no-evidence 0 "described no evidence" "an observer that describes no evidence is refused"
stub_case bad-digest 0 "does not verify against the digests" \
  "evidence that does not match its declared digest is refused"
stub_case lost-token 0 "does not carry the bytes the driver sent" \
  "a transcript missing the driver's own bytes is refused"
stub_case pass 3 "the observer exited 3" "an observer that exits non-zero is refused"

# ⚠ Three lines the runner keys on WHILE the observer is still serving, so each
# mutation has to happen mid-stream.
filter_case() { # drop  substitute  saying  name
  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$STATE" \
    BIT_IDS_STUB_INNER="$OBSERVER" \
    BIT_IDS_STUB_DROP="$1" \
    BIT_IDS_STUB_SUB="$2" \
    BIT_IDS_STUB_RC=0 \
    sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB_FILTER" \
    --seconds "$SECS" --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
  _got=$?
  if [ "$_got" != 1 ]; then
    fail "sh   $4 (wanted exit 1, got $_got)"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  elif ! grep -q -F -e "$3" "$WORK/err"; then
    fail "sh   $4 (exit 1, but did not say '$3')"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  else
    pass "sh   $4"
  fi
}

filter_case endpoint "" "never reported a tracker-http endpoint" \
  "an observer that names no endpoint is refused"
filter_case segments: "" "reported no segment count" \
  "an observer that reports no segment count is refused"
filter_case segments: "segments: 0" "recorded no bytes at all" \
  "a run that recorded no bytes is refused"

# -- 5. the PowerShell half ---------------------------------------------------
#
# ⚠ DECLARED WHEN pwsh IS ABSENT, never skipped silently. A comparison dropped
# for convenience is a comparison that stops happening, and this is the only
# thing in the tree that compares the two capture runners at all.
PS_RUNNER="$ROOT/scripts/capture/capture-run.ps1"
if ! command -v pwsh >/dev/null 2>&1; then
  fail "ps   the PowerShell half was not exercised: no pwsh on PATH"
elif [ ! -f "$PS_RUNNER" ]; then
  fail "ps   $PS_RUNNER is not present"
else
  PS_STATE="$WORK/ps-state"
  rm -rf "$PS_STATE"
  BIT_IDS_STATE_DIR="$PS_STATE" pwsh -NoProfile -File "$ROOT/scripts/acquisition/assert-disposable.ps1" \
    -Claim -RunId "$RUN" >/dev/null 2>&1 || {
    printf 'check-capture: could not claim a scratch host for the PowerShell half\n' >&2
    exit 2
  }

  ps_case() { # want-code  saying  name  observer  extra-args...
    _want="$1"
    _saying="$2"
    _name="$3"
    _observer="$4"
    shift 4
    OUTS=$((OUTS + 1))
    BIT_IDS_STATE_DIR="$PS_STATE" pwsh -NoProfile -File "$PS_RUNNER" \
      -RunId "$RUN" -Out "$WORK/out-$OUTS" -Observer "$_observer" \
      -Seconds "$SECS" "$@" >"$WORK/out" 2>"$WORK/err"
    _got=$?
    if [ "$_got" != "$_want" ]; then
      fail "ps   $_name (wanted exit $_want, got $_got)"
      [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
    elif [ -n "$_saying" ] && ! grep -q -F -e "$_saying" "$WORK/err"; then
      fail "ps   $_name (exit $_want, but did not say '$_saying')"
      [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
    else
      pass "ps   $_name"
    fi
  }

  # ⛔ THE WINDOWS HALF IS CONTAINED AT THE FIREWALL, so its fixtures are
  # firewall state, and the programs they block are ASKED of the runner, which
  # is the only derivation of that list. ⚠ A fixture written from a list
  # composed here would pass over a runner whose -Programs forgot its driver.
  fw_state() { # file observer...
    _file="$1"
    shift
    {
      printf 'profile\tname=Domain\tenabled=True\n'
      printf 'profile\tname=Private\tenabled=True\n'
      printf 'profile\tname=Public\tenabled=True\n'
    } >"$_file"
    for _obs in "$@"; do
      pwsh -NoProfile -File "$PS_RUNNER" -Programs -Observer "$_obs" >"$WORK/programs" 2>"$WORK/err" || {
        printf 'check-capture: capture-run.ps1 -Programs could not run\n' >&2
        exit 2
      }
      while IFS= read -r _prog; do
        printf 'rule\tenabled=True\tdirection=Outbound\taction=Block\tprofile=Any\tlocal=Any\tremote=Any\tprotocol=Any\tinterfacetype=Any\tinterface=Any\tservice=Any\tlocaluser=Any\tprogram=%s\n' "$_prog"
      done <"$WORK/programs" >>"$_file"
    done
  }
  fw_state "$WORK/fw-contained" "$OBSERVER" "$STUB"
  # The observer alone: the driver has a way off the host.
  pwsh -NoProfile -File "$PS_RUNNER" -Programs -Observer "$OBSERVER" >"$WORK/programs" 2>/dev/null
  head -1 "$WORK/programs" >"$WORK/observer-only"
  {
    grep '^profile' "$WORK/fw-contained"
    grep -F -f "$WORK/observer-only" "$WORK/fw-contained" | grep '^rule'
  } >"$WORK/fw-open"

  _programs=$(wc -l <"$WORK/programs" | tr -d ' ')
  if [ "$_programs" = 2 ] && grep -q 'curl' "$WORK/programs"; then
    pass "ps   -Programs names the observer and the driver"
  else
    fail "ps   -Programs names $_programs program(s), not the observer and the driver"
  fi

  # ⛔ AND THE SCRIPT THAT WRITES THE RULES REFUSES WHAT IT CANNOT CONTAIN.
  # Its writes need a Windows firewall and are proved by a capture dispatch,
  # whose guard reads them back; what runs here is every way it can be asked
  # to contain nothing, each of which would leave the guard as the only door.
  CONTAIN="$ROOT/scripts/capture/firewall-contain.ps1"
  contain_case() { # saying name args...
    _saying="$1"
    _name="$2"
    shift 2
    pwsh -NoProfile -File "$CONTAIN" "$@" >"$WORK/out" 2>"$WORK/err"
    _got=$?
    if [ "$_got" = 2 ] && grep -q -F -e "$_saying" "$WORK/err"; then
      pass "ps   $_name"
    else
      fail "ps   $_name (exit $_got)"
    fi
  }
  contain_case "-Program is required" "firewall-contain refuses to contain nothing"
  contain_case "is not an absolute path" "firewall-contain refuses a relative program" -Program curl.exe
  contain_case "is not present" "firewall-contain refuses a program nothing runs from" \
    -Program "$WORK/no-such-binary"
  if pwsh -NoProfile -Command 'if (Get-Command New-NetFirewallRule -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }' >/dev/null 2>&1; then
    fail "ps   this host has a Windows firewall, so firewall-contain was not run against it"
  else
    contain_case "has no Windows Firewall" "firewall-contain on a host with no firewall is could-not-run" \
      -Program "$OBSERVER|$(head -1 "$WORK/programs")"
  fi

  ps_case 0 "" "a claimed, contained host captures and verifies" \
    "$OBSERVER" -FirewallState "$WORK/fw-contained"
  PS_CONTROL_OUT="$WORK/out-$OUTS"
  ps_case 1 "egress is open" "a driver the firewall does not block is refused" \
    "$OBSERVER" -FirewallState "$WORK/fw-open"
  ps_case 2 "egress could not be established" "an unreadable firewall state is not a pass" \
    "$OBSERVER" -FirewallState "$WORK/no-such-state"
  ps_case 2 "is not present" "an observer that was never built is could-not-run" \
    "$WORK/no-such-binary" -FirewallState "$WORK/fw-contained"

  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$WORK/ps-unclaimed" pwsh -NoProfile -File "$PS_RUNNER" \
    -RunId "$RUN" -Out "$WORK/out-$OUTS" -Observer "$OBSERVER" -Seconds "$SECS" \
    -FirewallState "$WORK/fw-contained" >"$WORK/out" 2>"$WORK/err"
  _rc=$?
  if [ "$_rc" = 1 ] && grep -q -F -e "the host was never claimed" "$WORK/err"; then
    pass "ps   an unclaimed host is refused"
  else
    fail "ps   an unclaimed host is refused (exit $_rc)"
  fi

  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$PS_STATE" pwsh -NoProfile -File "$PS_RUNNER" \
    -RunId capture-0002 -Out "$WORK/out-$OUTS" -Observer "$OBSERVER" -Seconds "$SECS" \
    -FirewallState "$WORK/fw-contained" >"$WORK/out" 2>"$WORK/err"
  _rc=$?
  if [ "$_rc" = 1 ] && grep -q -F -e "is claimed by run [$RUN], not [capture-0002]" "$WORK/err"; then
    pass "ps   a host claimed by another run is refused"
  else
    fail "ps   a host claimed by another run is refused (exit $_rc)"
  fi

  BIT_IDS_STATE_DIR="$PS_STATE" pwsh -NoProfile -File "$PS_RUNNER" \
    -RunId "$RUN" -Out "$PS_CONTROL_OUT" -Observer "$OBSERVER" -Seconds "$SECS" \
    -FirewallState "$WORK/fw-contained" >"$WORK/out" 2>"$WORK/err"
  _rc=$?
  if [ "$_rc" = 1 ] && grep -q -F -e "already exists" "$WORK/err"; then
    pass "ps   a capture never writes into another run's evidence"
  else
    fail "ps   a capture never writes into another run's evidence (exit $_rc)"
  fi

  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$PS_STATE" \
    BIT_IDS_STUB_REAL="$OBSERVER" \
    BIT_IDS_STUB_ROWS="$WORK/rows-$OUTS" \
    BIT_IDS_STUB_MODE=lost-token \
    BIT_IDS_STUB_RC=0 \
    BIT_IDS_STUB_TOKEN_HEX="$TOKEN_HEX" \
    pwsh -NoProfile -File "$PS_RUNNER" -RunId "$RUN" -Out "$WORK/out-$OUTS" \
    -Observer "$STUB" -Seconds "$SECS" -FirewallState "$WORK/fw-contained" \
    >"$WORK/out" 2>"$WORK/err"
  _rc=$?
  if [ "$_rc" = 1 ] && grep -q -F -e "does not carry the bytes the driver sent" "$WORK/err"; then
    pass "ps   a transcript missing the driver's own bytes is refused"
  else
    fail "ps   a transcript missing the driver's own bytes is refused (exit $_rc)"
  fi

  # -- 6. the two attestations, compared --------------------------------------
  #
  # ⛔ THE KEY SET FIRST. A field in one half and not the other is the commonest
  # shape of drift and is invisible to anything that compares values.
  SH_KEYS=$(awk -F= 'NR > 1 { print $1 }' "$CONTROL_OUT/attestation.txt" | LC_ALL=C sort)
  PS_KEYS=$(awk -F= 'NR > 1 { print $1 }' "$PS_CONTROL_OUT/attestation.txt" | LC_ALL=C sort)
  if [ "$SH_KEYS" = "$PS_KEYS" ]; then
    pass "twin the two attestations carry the same fields"
  else
    fail "twin the two attestations carry different fields"
    [ "$JSON" = "1" ] || printf '          sh: %s\n' "$(printf '%s' "$SH_KEYS" | tr '\n' ' ')"
    [ "$JSON" = "1" ] || printf '          ps: %s\n' "$(printf '%s' "$PS_KEYS" | tr '\n' ' ')"
  fi

  SH_SCHEMA=$(head -1 "$CONTROL_OUT/attestation.txt")
  PS_SCHEMA=$(head -1 "$PS_CONTROL_OUT/attestation.txt")
  if [ "$SH_SCHEMA" = "$PS_SCHEMA" ]; then
    pass "twin both attestations declare $SH_SCHEMA"
  else
    fail "twin the schema differs: sh=[$SH_SCHEMA] ps=[$PS_SCHEMA]"
  fi

  # ⚠ ONLY THE FACTS THAT DESCRIBE ONE RUN. platform, observer_sha256, the
  # clocks, the fingerprint, the marker, the route source and the verifier all
  # differ honestly: the two halves ran at different instants, read different
  # route formats, and verify with different readers. A comparison that included
  # those would have to be widened the first time it was run, which is how a
  # check stops checking.
  for field in run kind measured_build stock_client egress driver driver_status segments evidence; do
    _a=$(awk -F= -v K="$field" '$1 == K { print $2; exit }' "$CONTROL_OUT/attestation.txt")
    _b=$(awk -F= -v K="$field" '$1 == K { print $2; exit }' "$PS_CONTROL_OUT/attestation.txt")
    if [ "$_a" = "$_b" ] && [ -n "$_a" ]; then
      pass "twin $field = $_a in both"
    else
      fail "twin $field: sh=[$_a] ps=[$_b]"
    fi
  done

  # ⭐ AND THE VERIFIERS MUST DIFFER, which is the other direction and the one a
  # twin comparison never asks. Two halves reporting one reader would mean one of
  # them is not running the reader it claims: coreutils sha256sum is not on
  # Windows and Get-FileHash is not on a minimal Linux image.
  _a=$(awk -F= '$1 == "evidence_verified_by" { print $2; exit }' "$CONTROL_OUT/attestation.txt")
  _b=$(awk -F= '$1 == "evidence_verified_by" { print $2; exit }' "$PS_CONTROL_OUT/attestation.txt")
  if [ -n "$_a" ] && [ -n "$_b" ] && [ "$_a" != "$_b" ]; then
    pass "twin the halves verify with different readers ($_a, $_b)"
  else
    fail "twin the halves report the same verifier: sh=[$_a] ps=[$_b]"
  fi
fi

store_report "check-capture/1" cases "$JSON"
