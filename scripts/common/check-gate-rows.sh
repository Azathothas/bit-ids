#!/bin/sh
# check-gate-rows.sh - do the two gate runners run the same set of rows?
#
# ⛔ NOTHING COMPARED THEM, and the gap is structural rather than accidental. The
# `sh` runner derives most of its provers from a `for` list; the PowerShell
# runner declares each one by hand as a run or an `Add-Unavailable`. A prover
# added to the first and forgotten in the second is simply ABSENT from the
# Windows lane, which stays green because it never hears of it - and `--strict`
# cannot help, because a row that was never named cannot be counted as a skip.
#
# ⚠ IT BIT IMMEDIATELY WHEN THE GAP WAS FOUND. `check-capture-client` went into
# the `sh` list and the Windows lane would have run one row fewer, with nothing
# anywhere naming the one that was missing. `CI-07` carries the entry.
#
# -- ⛔ WHY THIS IS NOT check-twins' JOB -------------------------------------
#
# `check-twins.sh` deliberately does not pair the gate runners: a harness that
# ran both gates, running inside the gate, would re-enter itself. ⭐ `--rows`
# and `-Rows` run NO check at all - each prints the names its own queue would
# have used and returns - so this comparison costs two process starts and cannot
# recurse. That is what makes it a gate row rather than a thing somebody
# remembers to do.
#
# -- ⚠ IT COMPARES SETS, NOT SEQUENCES ---------------------------------------
#
# The two halves schedule differently on purpose - the `sh` half runs its checks
# concurrently and reads verdicts in list order, and the PowerShell half is
# serial - so the ORDER a row appears in is not a fact about what either lane
# runs. What must agree is the set, and a duplicate shows up as a count that does
# not match.
#
# Usage:
#   sh scripts/common/check-gate-rows.sh
#   sh scripts/common/check-gate-rows.sh --json
#
# Exit codes: 0 every case held, 1 one did not, 2 could not run.
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
      printf 'check-gate-rows: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-gate-rows
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require awk sha256sum sort
# ⛔ pwsh IS REQUIRED RATHER THAN OPTIONAL HERE, and exit 2 says so. This check
# has exactly one question and half of the answer comes from the PowerShell
# runner, so a host without it has verified NOTHING - which is a skip and never
# a pass. The Linux lane runs the gate with --strict and `ubuntu-24.04` carries
# pwsh; `scripts/doctor/provision.sh` installs it on a session host.
store_require pwsh

WORK=$(store_workdir checkgaterows) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

SH_GATE="$ROOT/scripts/common/check-gate.sh"
PS_GATE="$ROOT/scripts/common/check-gate.ps1"
for _need in "$SH_GATE" "$PS_GATE"; do
  [ -f "$_need" ] || {
    printf 'check-gate-rows: %s is missing\n' "$_need" >&2
    exit 2
  }
done

# ⚠ THE CARRIAGE RETURNS COME OFF THE PowerShell SIDE. A `.ps1` keeps CRLF and
# `Write-Output` ends a line the host's way, so a comparison that skipped this
# would report every single row as differing - which reads exactly like two
# runners with nothing in common.
rows_sh() { # gate-file outfile
  sh "$1" --rows 2>"$WORK/sh.err" | tr -d '\r' | LC_ALL=C sort >"$2"
}
rows_ps() { # gate-file outfile
  pwsh -NoProfile -File "$1" -Rows 2>"$WORK/ps.err" | tr -d '\r' | LC_ALL=C sort >"$2"
}

# ⛔ THE COMPARISON REFUSES TWO EMPTY LISTS. A runner that printed nothing at all
# agrees perfectly with another that printed nothing at all, and that is the
# shape `check-one-home` records about its own first run: zero findings over zero
# files reads exactly like a clean tree.
FLOOR=20

compare() { # label sh-list ps-list expect(agree|differ)
  _n_sh=$(grep -c . "$2" || true)
  _n_ps=$(grep -c . "$3" || true)
  if [ "$_n_sh" -lt "$FLOOR" ] || [ "$_n_ps" -lt "$FLOOR" ]; then
    fail "$1: a runner named $_n_sh and $_n_ps rows; fewer than $FLOOR means one of them printed nothing"
    return
  fi
  if diff "$2" "$3" >"$WORK/rowdiff" 2>&1; then
    if [ "$4" = agree ]; then
      pass "$1 ($_n_sh rows on each side)"
    else
      fail "$1: the two lists agree, so the planted row was not noticed"
    fi
    return
  fi
  if [ "$4" = differ ]; then
    pass "$1 ($(grep -c '^[<>]' "$WORK/rowdiff" || true) row(s) on one side only)"
  else
    fail "$1: the runners disagree: $(grep '^[<>]' "$WORK/rowdiff" | tr '\n' ' ')"
  fi
}

# -- 1. the tree as it stands -------------------------------------------------
rows_sh "$SH_GATE" "$WORK/sh.rows"
rows_ps "$PS_GATE" "$WORK/ps.rows"
compare "control  the two runners name the same rows" "$WORK/sh.rows" "$WORK/ps.rows" agree

# ⛔ AND THE MODE RUNS NOTHING, WHICH IS WHY THIS CAN BE A GATE ROW. A --rows
# that ran the checks would put a whole gate inside the gate. The evidence is
# what it printed: a verdict row, a summary line or a skip notice would all be
# absent from a list of names.
if grep -q -E '✅|❌|SKIP|n/a|checks:' "$WORK/sh.rows" "$WORK/ps.rows"; then
  fail "probe    a rows list carries verdict text, so the mode ran something"
else
  pass "probe    both rows modes print names and no verdict of any kind"
fi

# -- 2. a row on one side only, planted three ways ----------------------------
#
# ⚠ EVERY PLANT IS IN A SCRATCH COPY. Both runners resolve their own directory
# from `$0`, and in rows mode neither reads a sibling check at all, so a copy
# anywhere names exactly what the original would.
cp "$SH_GATE" "$WORK/gate.sh" || exit 2
cp "$PS_GATE" "$WORK/gate.ps1" || exit 2

if replace_once "$WORK/gate.sh" 'ci/check-staleness ci/check-step-bodies' \
  'ci/check-staleness ci/check-step-bodies ci/check-planted'; then
  rows_sh "$WORK/gate.sh" "$WORK/sh.plant"
  compare "sh       a prover added to the sh list alone is caught" \
    "$WORK/sh.plant" "$WORK/ps.rows" differ
else
  fail "sh       the added-prover plant did not apply"
fi

cp "$SH_GATE" "$WORK/gate.sh" || exit 2
if replace_once "$WORK/gate.sh" 'publishing/check-formats ' ''; then
  rows_sh "$WORK/gate.sh" "$WORK/sh.plant"
  compare "sh       a prover dropped from the sh list alone is caught" \
    "$WORK/sh.plant" "$WORK/ps.rows" differ
else
  fail "sh       the dropped-prover plant did not apply"
fi

if replace_once "$WORK/gate.ps1" "Add-Unavailable 'check-staleness'" \
  "Add-Unavailable 'check-only-in-powershell' 'planted'; Add-Unavailable 'check-staleness'"; then
  rows_ps "$WORK/gate.ps1" "$WORK/ps.plant"
  compare "pwsh     a row declared on the PowerShell lane alone is caught" \
    "$WORK/sh.rows" "$WORK/ps.plant" differ
else
  fail "pwsh     the added-row plant did not apply"
fi

# -- 3. the control, after every plant ----------------------------------------
rows_sh "$SH_GATE" "$WORK/sh.after"
rows_ps "$PS_GATE" "$WORK/ps.after"
compare "control  the shipped runners still agree after every plant" \
  "$WORK/sh.after" "$WORK/ps.after" agree

store_probe_guards "$WORK/gate.sh" 'ROWS_ONLY=0' 'queue_row'

store_report check-gate-rows/1 cases "$JSON"
