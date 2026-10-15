#!/bin/sh
# check-runner.sh - prove the disposable-host guards refuse what they exist to refuse.
#
# ⛔ A GUARD THAT HAS NEVER BEEN SEEN TO REFUSE IS A GUARD NOBODY KNOWS WORKS.
# `bit-check assert-disposable` stands between this project and installing an untrusted
# client on a machine somebody keeps. It is the last thing that runs before a
# capture and the first thing whose silence would be catastrophic, so it is
# mutation-proven here rather than trusted.
#
# Each case plants the exact condition the guard exists to catch and reads the
# exit code from the guard's own process, unpiped.
#
# ⚠ This does NOT prove the guards are sufficient. It proves each one fires. A
# host can be non-disposable in ways neither guard models, which is why
# `docs/security/remote-ops.md` and the runner contract in `TODO/acquisition.md`
# carry the rest.
#
# Usage:
#   sh scripts/acquisition/check-runner.sh
#   sh scripts/acquisition/check-runner.sh --json
#
# Exit codes: 0 every guard refused what it should, 1 one did not, 2 could not run.
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
      printf 'check-runner: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

WORK="${TMPDIR:-/tmp}/.checkrunner.$$"
mkdir -p "$WORK" || {
  printf 'check-runner: cannot write to %s\n' "$WORK" >&2
  exit 2
}
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⭐ THE GUARD IS `bit-check assert-disposable` SINCE 2026-10-15, `CI-11`, and
# these cases held its shell half to the same rows before it was deleted.
command -v go >/dev/null 2>&1 || {
  printf 'check-runner: go is not on this host, so bit-check cannot be built\n' >&2
  exit 2
}
(cd "$ROOT/tools/check" && go build -o "$WORK/bit-check" .) >/dev/null 2>&1 || {
  printf 'check-runner: cannot build bit-check\n' >&2
  exit 2
}
BIT_CHECK="$WORK/bit-check"

PASS=0
FAIL=0
ROWS=""

# ⛔ The guard runs unpiped and $? is read on the next line. Piping it into
# anything reports the pipe's status, so a guard that failed to refuse would
# read as having refused. That is this repository's oldest stated rule.
expect() { # want-code  name  args...
  _want="$1"
  _name="$2"
  shift 2
  expect_says "$_want" "" "$_name" "$@"
}

# ⛔ A CASE MAY REQUIRE THE VERDICT AS WELL AS THE CODE, AND SOME MUST. Four of
# this guard's refusals answer 2 and two answer 1, so a case asserting the code
# alone passes when a different refusal fired. The PowerShell twin learned that
# from a mutation pass and this half did not have the cases at all: the run-id
# rule, the usage refusal and the two-modes refusal were unproven here while
# check-runner.ps1 proved all three.
expect_says() { # want-code  saying  name  args...
  _want="$1"
  _saying="$2"
  _name="$3"
  shift 3
  "$BIT_CHECK" assert-disposable "$@" >"$WORK/out" 2>"$WORK/err"
  _got=$?
  if [ "$_got" != "$_want" ]; then
    ROWS="$ROWS  ❌ FAIL  $_name (wanted exit $_want, got $_got)
"
    FAIL=$((FAIL + 1))
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -4
  elif [ -n "$_saying" ] && ! grep -q -F -e "$_saying" "$WORK/err"; then
    ROWS="$ROWS  ❌ FAIL  $_name (exit $_want, but did not say '$_saying')
"
    FAIL=$((FAIL + 1))
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -4
  else
    ROWS="$ROWS  ✅ ok    $_name
"
    PASS=$((PASS + 1))
  fi
}

# --- forbidden egress -------------------------------------------------------
# A table with a default route is a host that can reach the public network, and
# a capture there puts an untrusted client on it.
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-open"
printf 'eth0\t00000000\t010200C0\t0003\t0\t0\t0\t00000000\t0\t0\t0\n' >>"$WORK/route-open"
expect 1 "a default route is refused" --egress "$WORK/route-open"

# ⚠ The same table with the route DOWN. The destination alone is not the test:
# a rule that matched only `00000000` would refuse a host whose default route is
# configured and inactive, and more importantly would have no reason to look at
# the flags at all, which is where a real difference lives.
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-down"
printf 'eth0\t00000000\t010200C0\t0002\t0\t0\t0\t00000000\t0\t0\t0\n' >>"$WORK/route-down"
expect 0 "a default route that is not up is not egress" --egress "$WORK/route-down"

# Loopback only, which is what a capture host looks like.
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-loopback"
printf 'lo\t0000007F\t00000000\t0001\t0\t0\t0\t000000FF\t0\t0\t0\n' >>"$WORK/route-loopback"
expect 0 "loopback only passes" --egress "$WORK/route-loopback"

# ⛔ An unreadable table is exit 2, not exit 0. Not knowing is not a pass.
expect_says 2 "is unreadable" "an unreadable table is not a pass" --egress "$WORK/no-such-table"

# --- the argument surface ---------------------------------------------------
#
# ⚠ THESE THREE WERE PROVED ONLY BY THE POWERSHELL TWIN. All four of this guard's
# `could not run` refusals share exit 2, so without the message each of these
# cases would pass over any of the others; and a usage refusal that exits 0 is
# the exact defect the ps1 half shipped, where `break` inside `ForEach-Object`
# unwound the script.
expect_says 2 "run id must be lowercase" "a run id that is not a slug is refused" \
  --claim "Capture 0001"
expect_says 2 "assert-disposable --claim" "no mode at all is refused"
expect_says 2 "assert-disposable --claim" "an unknown mode is refused" --nonsense

# --- persistent state -------------------------------------------------------
#
# ⛔ EVERY CLAIM HERE GOES TO A SCRATCH DIRECTORY. The first version of this
# ran one case without the override, so the test claimed the real machine: it
# wrote /var/lib/bit-ids/host-claimed, passed once, and failed on every run
# after. On a persistent CI runner that is a check which goes green on the day
# it is written and red forever, for a reason that looks like the guard being
# broken rather than the test being. Found by running it twice.
FRESH="$WORK/host-a"
BIT_IDS_STATE_DIR="$FRESH" "$BIT_CHECK" assert-disposable --claim capture-0001 >"$WORK/out" 2>&1
_first=$?
BIT_IDS_STATE_DIR="$FRESH" "$BIT_CHECK" assert-disposable --claim capture-0002 >"$WORK/out2" 2>&1
_second=$?
# ⛔ AND BY THE MARKER CHECK, NAMED BY ITS OWN WORDS. Two refusals guard a second
# claim and both answer 1: the marker check and the exclusive create. A mutant
# of the Go port with the marker check off passed this case on its exit code
# alone, 2026-10-15, because the create refused instead; the PowerShell half's
# harness had recorded the same masking. The message separates them.
if [ "$_first" = "0" ] && [ "$_second" = "1" ] &&
  grep -q -F -e 'already ran a capture' "$WORK/out2"; then
  ROWS="$ROWS  ✅ ok    a second capture on one host is refused
"
  PASS=$((PASS + 1))
else
  ROWS="$ROWS  ❌ FAIL  a second capture on one host is refused (first $_first, second $_second)
"
  FAIL=$((FAIL + 1))
fi

# --- a fresh host for the next job -----------------------------------------
# ⚠ Destroying the marker is what a real teardown does. This shows the NEXT job
# is unblocked once it happens, which is the other half of the same guard: it
# must refuse a survived host without refusing every host forever.
rm -rf "$FRESH"
BIT_IDS_STATE_DIR="$FRESH" "$BIT_CHECK" assert-disposable --claim capture-0002 >"$WORK/out3" 2>&1
_third=$?
if [ "$_third" = "0" ]; then
  ROWS="$ROWS  ✅ ok    a torn-down host claims again for the next job
"
  PASS=$((PASS + 1))
else
  ROWS="$ROWS  ❌ FAIL  a torn-down host claims again for the next job (exit $_third)
"
  FAIL=$((FAIL + 1))
fi

# --- the marker path, which is the only derivation of it --------------------
#
# ⛔ ASKED FOR, NEVER COMPOSED. `capture-run` reads this to find out whether the
# host was claimed and by which run, and a second spelling of the path in that
# caller would go on reading the old place the day the state directory moves.
# ⚠ The case is not "it printed something": it claims into a scratch directory
# and then requires the file this mode NAMES to be the one that appeared. A mode
# returning a plausible path nothing writes to would pass any weaker check.
MARKED="$WORK/host-c"
MARKER_PATH=$(BIT_IDS_STATE_DIR="$MARKED" "$BIT_CHECK" assert-disposable --marker)
if [ -n "$MARKER_PATH" ] && [ ! -e "$MARKER_PATH" ]; then
  ROWS="$ROWS  ✅ ok    the marker path is reported before anything is claimed
"
  PASS=$((PASS + 1))
else
  ROWS="$ROWS  ❌ FAIL  the marker path is reported before anything is claimed [$MARKER_PATH]
"
  FAIL=$((FAIL + 1))
fi

BIT_IDS_STATE_DIR="$MARKED" "$BIT_CHECK" assert-disposable --claim capture-0004 >"$WORK/out" 2>&1
_claimed=$?
if [ "$_claimed" = "0" ] && [ -f "$MARKER_PATH" ]; then
  ROWS="$ROWS  ✅ ok    --marker names the file --claim actually wrote
"
  PASS=$((PASS + 1))
else
  ROWS="$ROWS  ❌ FAIL  --marker names the file --claim actually wrote (claim exit $_claimed)
"
  FAIL=$((FAIL + 1))
fi

expect_says 2 "assert-disposable --claim" "--marker takes no argument" --marker extra

# The fingerprint the next job compares against.
PRINT=$("$BIT_CHECK" assert-disposable --fingerprint)
case "$PRINT" in
  [0-9a-f]*)
    if [ "${#PRINT}" = "64" ]; then
      ROWS="$ROWS  ✅ ok    the host fingerprint is a sha256
"
      PASS=$((PASS + 1))
    else
      ROWS="$ROWS  ❌ FAIL  the host fingerprint is ${#PRINT} characters
"
      FAIL=$((FAIL + 1))
    fi
    ;;
  *)
    ROWS="$ROWS  ❌ FAIL  the host fingerprint is not hexadecimal
"
    FAIL=$((FAIL + 1))
    ;;
esac

TOTAL=$((PASS + FAIL))
if [ "$JSON" = "1" ]; then
  printf '{"schema":"check-runner/1","total":%s,"passed":%s,"failed":%s,"fingerprint":"%s"}\n' \
    "$TOTAL" "$PASS" "$FAIL" "$PRINT"
  [ "$FAIL" -eq 0 ] || exit 1
  exit 0
fi

printf '\n%s\n' "$ROWS"
printf '%s guard case(s): %s passed, %s failed\n' "$TOTAL" "$PASS" "$FAIL"
printf 'host fingerprint for the next job: %s\n' "$PRINT"
# ⛔ Zero cases passing is not a green run, whatever the failure count says.
if [ "$PASS" -eq 0 ]; then
  printf -- '❌ NOTHING RAN.\n'
  exit 1
fi
if [ "$FAIL" -gt 0 ]; then
  printf -- '❌ a guard did not refuse what it exists to refuse.\n'
  exit 1
fi
printf -- '✅ every guard refused its own defect.\n'
