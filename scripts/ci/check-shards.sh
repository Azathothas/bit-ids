#!/bin/sh
# check-shards.sh - do the N shards together run every unit exactly once?
#
# `CI-10` sharded `check-workflow` across runners, and its own residual named the
# failure mode before the work started:
#
#   ⛔ *A shard selector that silently claimed no case would report a faster green
#   run over less work, which is this repository's oldest defect class and the one
#   a wall-clock target makes most tempting.*
#
# ⭐ SO THE PARTITION IS CHECKED RATHER THAN ASSUMED, and it is checked the cheap
# way: `check-workflow --units` lists the names and runs nothing, so every
# assertion here costs one process start rather than a gate run. ⚠ That is the
# same argument `check-gate-rows` makes about the two gate runners' row lists -
# the expensive thing and the thing worth comparing are not the same thing.
#
# -- ⛔ THE THREE PROPERTIES, AND WHY NONE OF THEM IS THE OTHER --------------
#
#   COVERAGE     every unit is claimed by some shard. A unit no shard claims is a
#                rule that stops running while every lane stays green.
#   DISJOINTNESS every unit is claimed by at most one shard. Duplication is not a
#                correctness defect, it is a wall-clock one, and it hides in a
#                green run exactly as well as a gap does.
#   THE CONTROLS every shard runs all of them. ⛔ A shard carrying only plants
#                goes GREEN over a tree where every step is broken, because a
#                plant that is refused proves nothing unless the clean tree is
#                accepted.
#
# ⚠ A run that checked only the first would pass over a selector that ran
# everything in every shard, which is a partition of nothing and N times the work.
#
# Usage:
#   sh scripts/ci/check-shards.sh
#   sh scripts/ci/check-shards.sh --json
#
# Exit codes: 0 every N partitions, 1 one does not, 2 could not run.
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
      printf 'check-shards: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-shards
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require git awk sort

SUBJECT="$ROOT/scripts/ci/check-workflow.sh"
[ -f "$SUBJECT" ] || {
  printf 'check-shards: %s is missing\n' "$SUBJECT" >&2
  exit 2
}

WORK=$(store_workdir checkshards) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⛔ THE FULL LIST IS TAKEN FROM THE SUBJECT, NOT WRITTEN HERE. A copy of the unit
# names in this file would be the value in two places that every comparison in
# this repository exists to refuse, and it would go stale in the commit that adds
# a unit - which is the one commit where this check matters.
sh "$SUBJECT" --units >"$WORK/all" 2>/dev/null || {
  printf 'check-shards: the subject could not list its units\n' >&2
  exit 2
}

awk -F'\t' '$2 != "control" && $1 != "" { print $1 }' "$WORK/all" >"$WORK/plants"
awk -F'\t' '$2 == "control" { print $1 }' "$WORK/all" >"$WORK/controls"

NPLANTS=$(awk 'END { print NR + 0 }' "$WORK/plants")
NCONTROLS=$(awk 'END { print NR + 0 }' "$WORK/controls")

# ⛔ A SUBJECT THAT NAMES NOTHING SATISFIES EVERY PROPERTY BELOW. Two empty sets
# partition perfectly, so a `--units` mode that stopped printing would leave this
# harness reporting a proved partition over no units at all. That is the
# vacuous-pass shape this repository has shipped twice, so it is refused here
# before anything is compared.
if [ "$NPLANTS" -lt 2 ] || [ "$NCONTROLS" -lt 1 ]; then
  printf 'check-shards: the subject names %s shardable unit(s) and %s control(s);\n' \
    "$NPLANTS" "$NCONTROLS" >&2
  printf 'check-shards: that is too few to be a partition anybody could get wrong\n' >&2
  exit 2
fi
pass "control  the subject names $NPLANTS shardable units and $NCONTROLS controls"

# ⚠ EVERY N FROM 1 TO 7, not one convenient value. `index mod N` is correct for
# every N by construction, and a harness that checked only the N the workflow
# happens to use would pass over a selector that was right for four and wrong for
# three - which is exactly what a later change to the matrix would reach for.
for n in 1 2 3 4 5 6 7; do
  : >"$WORK/union"
  _bad=0
  _i=1
  while [ "$_i" -le "$n" ]; do
    if ! sh "$SUBJECT" --shard "$_i/$n" --units >"$WORK/shard" 2>/dev/null; then
      fail "shard    --shard $_i/$n could not list its units"
      _bad=1
      break
    fi
    # ⛔ THE CONTROLS ARE CHECKED PER SHARD AND NOT IN THE UNION. A control that
    # appeared in the union once would satisfy a coverage test while running in
    # one shard of six, which is the whole defect.
    awk -F'\t' '$2 == "control" { print $1 }' "$WORK/shard" | sort >"$WORK/shard-controls"
    if ! sort "$WORK/controls" | diff -q - "$WORK/shard-controls" >/dev/null 2>&1; then
      fail "shard    --shard $_i/$n does not name every control"
      _bad=1
      break
    fi
    awk -F'\t' '$2 != "control" && $1 != "" { print $1 }' "$WORK/shard" >>"$WORK/union"
    _i=$((_i + 1))
  done
  [ "$_bad" = 0 ] || continue

  # ⚠ `--units` prints what a shard WOULD run, so the union is over the selector
  # rather than over a run. That is deliberate and it is the whole reason this is
  # affordable: the selector is the thing that can be wrong, and running it is
  # what a shard does with the answer.
  sort "$WORK/union" >"$WORK/union-sorted"
  sort "$WORK/plants" >"$WORK/plants-sorted"

  _dups=$(uniq -d "$WORK/union-sorted" | awk 'END { print NR + 0 }')
  _count=$(awk 'END { print NR + 0 }' "$WORK/union-sorted")

  if [ "$_dups" != "0" ]; then
    fail "n=$n      $_dups unit(s) claimed by more than one shard"
    continue
  fi
  if ! diff -q "$WORK/plants-sorted" "$WORK/union-sorted" >/dev/null 2>&1; then
    fail "n=$n      the shards do not cover the unit list"
    [ "$JSON" = "1" ] || diff "$WORK/plants-sorted" "$WORK/union-sorted" | sed 's/^/          /'
    continue
  fi
  pass "n=$n      $n shards cover all $_count units exactly once, each naming every control"
done

# -- ⛔ the coupling sharding creates, which no partition check would see ------
#
# ⛔ A VARIABLE ASSIGNED IN ONE UNIT AND READ IN ANOTHER IS A UNIT THAT ONLY WORKS
# WHEN ITS NEIGHBOUR RAN. The partition above can be perfect and every shard still
# die, because the two units land on different runners. ⚠ Measured on 2026-09-10,
# on the first real shard ever driven: `--shard 1/4` exited 2 on
# `PUBWF: parameter not set`, because the publisher's path was assigned inside
# `publisher-trigger` and read inside `static-readers`. A second, `LIB`, was
# assigned in `rust-nocompile` and read by two more units.
#
# ⭐ IT WAS LOUD ONLY BECAUSE OF `set -u`. Without it the path would have been an
# empty string, `[ -f "" ]` would have been false, and the case would have
# reported the publisher workflow missing - a plausible-looking failure naming the
# wrong thing entirely, or worse, a branch that quietly passed.
#
# ⚠ THIS IS A STATIC READ AND IT SAYS SO. It matches assignments and expansions of
# upper-case names by text, so a name built at runtime is outside what it can see.
# What it covers is the shape both instances had.
# ⚠ THE TEST IS "READS A NAME IT NEVER ASSIGNS", not "another unit also assigns
# it". Two units that each set and use their own `MISSING` share a spelling and
# nothing else, and a first version of this reported exactly that as a coupling.
# ⭐ A false positive here is not harmless: it is a red row that teaches a reader
# to stop believing the row.
COUPLING=$(awk '
  /^if (unit|always) [A-Za-z0-9_-]+; then$/ { u = $3; sub(/;$/, "", u); next }
  u != "" && /^fi$/ { u = ""; next }
  # A COMMENT LINE IS SKIPPED OUTRIGHT, because a rule has to be describable in
  # the file it constrains. check-workflow explains this very coupling in a
  # comment naming the variable, and without this line that sentence is itself
  # reported as the defect - the same shape the Write-Error rule in check-project
  # hit when its needle matched its own failure message.
  u != "" && /^[ \t]*#/ { next }
  u != "" {
    # THREE SPELLINGS OF AN ASSIGNMENT, BECAUSE A FIRST VERSION SAW ONE. It
    # matched NAME= at the start of a line only, so it missed a for-loop variable
    # and it missed an assignment inside a case branch - and BOTH were real
    # couplings in the subject. A guard that sees one spelling of the thing it
    # forbids reports clean over the other two.
    rest = $0
    while (match(rest, /(^|[ \t;()&|])[A-Z][A-Z0-9_]*=/)) {
      name = substr(rest, RSTART, RLENGTH)
      sub(/^[ \t;()&|]/, "", name)
      sub(/=$/, "", name)
      assigned[u SUBSEP name] = 1
      anywhere[name] = anywhere[name] " " u
      rest = substr(rest, RSTART + RLENGTH)
    }
    if (match($0, /^[ \t]*for[ \t]+[A-Z][A-Z0-9_]*[ \t]+in[ \t]/)) {
      name = $2
      assigned[u SUBSEP name] = 1
      anywhere[name] = anywhere[name] " " u
    }
    body[NR] = $0
    owner[NR] = u
  }
  END {
    for (r in body)
      for (n in anywhere)
        if (!((owner[r] SUBSEP n) in assigned) && body[r] ~ ("\\$\\{?" n "([^A-Z0-9_]|$)"))
          if (!((n SUBSEP owner[r]) in seen)) {
            seen[n SUBSEP owner[r]] = 1
            printf "%s assigned in%s, read in %s which never assigns it\n", \
              n, anywhere[n], owner[r]
          }
  }
' "$SUBJECT")

if [ -z "$COUPLING" ]; then
  pass "coupling no variable is assigned in one unit and read in another"
else
  fail "coupling a unit depends on a variable another unit sets"
  [ "$JSON" = "1" ] || printf '%s\n' "$COUPLING" | sed 's/^/          /'
fi

# -- ⛔ the selector's own refusals -------------------------------------------
#
# ⛔ EVERY WRONG SELECTOR FAILS IN THE DIRECTION OF RUNNING LESS, so each one has
# to be refused rather than clamped. `--shard 0/4` and `--shard 5/4` each select
# no unit at all, and a run of controls alone is exactly what a green shard
# covering no work looks like.
for bad in 0/4 5/4 1/0 x/4 4 ''; do
  if [ -z "$bad" ]; then
    sh "$SUBJECT" --shard >/dev/null 2>&1
  else
    sh "$SUBJECT" --shard "$bad" >/dev/null 2>&1
  fi
  _rc=$?
  if [ "$_rc" = 2 ]; then
    pass "refusal  --shard ${bad:-<nothing>} is refused as could-not-run (exit 2)"
  else
    fail "refusal  --shard ${bad:-<nothing>} answered $_rc, expected 2"
  fi
done

store_report check-shards/1 cases "$JSON"
