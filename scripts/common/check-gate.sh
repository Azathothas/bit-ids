#!/bin/sh
# check-gate.sh - run the whole local gate, in one command, and read every
# exit code from the process that produced it.
#
# The defect this exists to catch is a gate that is a LIST. Part (a) of
# docs/methodology/gate.md names several checks, and a list run by hand is run
# in the order somebody recalls it. ⛔ The session that first wrote this ran its
# gate five times and typed a different subset each time. Nothing failed; the
# gate simply was not the same gate twice.
#
# ⭐ IT DELEGATES. It holds no rules of its own and it is not a second opinion
# about anything. Every verdict here is some other script's, read unpiped.
#
# -- ⛔ A SKIPPED CHECK IS A SKIP, NEVER A PASS -----------------------------
#
# `pwsh`, `jq`, `gh` and `shellcheck` are not on every machine. A runner that
# quietly dropped one and printed green would be the row in
# docs/conventions/forbidden-patterns.md that reads *a step that exits 0 having
# done nothing it was asked to do*.
#
# So a skip is counted, named, and printed on its own line. ⚠ The exit code is
# still 0, because a machine that cannot run a check has not failed it; ⭐ pass
# --strict to make a skip a failure, which is what a CI job should do, since
# there the tools are installed on purpose and a skip means the install broke.
#
# -- ⛔ AND A CHECK THIS HOST CANNOT RUN IS NOT A CHECK THAT BROKE ------------
#
# Those are two different facts and --strict could not tell them apart, which
# made it unusable on the lane that needed it most. Measured on 2026-09-06:
# with check-project rewritten to exit 2, the Windows lane's own invocation
# still exited 0, because it ran without --strict; and it ran without --strict
# because six of its rows are checks Windows genuinely cannot run, so the flag
# would have refused every green tree.
#
# ⭐ So a row is one of two kinds. A DECLARED unavailability is written in the
# runner, with a reason and the entry that owns it, and prints as `n/a`. An
# OBSERVED skip is a check that ran and answered 2, or one whose file is gone,
# and prints as `SKIP`. --strict refuses the second and permits the first, so
# both lanes run strict and a check that quietly stops running turns a lane red
# wherever it happens.
#
# ⚠ --fast counts as declared, because the caller asked for it rather than the
# host failing to supply it. The summary names the two totals separately so a
# run cannot claim a strictness it did not have.
#
# -- ⛔ IT DOES NOT RUN ITSELF, AND THAT IS NOT THEORETICAL ------------------
#
# This runs check-twins.sh, which runs both halves of every pair. ⚠ A version
# of this idea in another repository hit an unbounded recursion with
# check-twins that left twenty stray shells holding their own files open. That
# is the reported symptom; the mechanism here is plain enough that it does not
# need re-deriving. A runner that appears in the pair list runs the comparison
# that runs the runner.
#
# So check-gate is NOT in check-twins.sh's pair list, and check-twins is
# invoked here directly rather than through anything that could re-enter.
# ⚠ The two exclusions are a shared contract: removing one reintroduces the
# hang.
#
# Usage:
#   sh scripts/common/check-gate.sh
#   sh scripts/common/check-gate.sh --fast     # skips check-twins
#   sh scripts/common/check-gate.sh --strict   # a skip is a failure
#   sh scripts/common/check-gate.sh --rows     # the row names, running nothing
#   sh scripts/common/check-gate.sh --json
#
# Exit codes: 0 nothing failed, 1 something failed, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

JSON=0
FAST=0
STRICT=0
# ⭐ --rows PRINTS THE ROW NAMES AND RUNS NOTHING, and it exists because nothing
# compared the two runners' row lists. The `sh` half derives its provers from a
# `for` list and the PowerShell half declares each one by hand, so a prover added
# to one and forgotten in the other is simply absent from that lane - which stays
# green because it never hears of it. ⚠ It bit immediately when it was written:
# `check-capture-client` went into the `sh` list and the Windows lane would have
# run one row fewer with nothing anywhere naming the missing one.
#
# ⛔ THE LIST IS THE ONE THE RUNNER QUEUES FROM, not a second copy of it. `queue`
# and `queue_row` print the name and return here, so a row this mode does not
# name is a row the runner does not run. A separate list would be the value in
# two places this mode exists to compare.
ROWS_ONLY=0

while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON=1 ;;
    --fast) FAST=1 ;;
    --strict) STRICT=1 ;;
    --rows) ROWS_ONLY=1 ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    *)
      printf 'check-gate: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

# ⛔ RESOLVED FROM THIS SCRIPT'S OWN LOCATION, not from the working directory.
# A runner found by a relative path runs a different set depending on who
# called it, which is the same class of defect as a guard whose scope depends
# on the process working directory.
HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)

PASS=0
FAIL=0
SKIP=0
NA=0
# Every row label handed out so far, so a second check cannot claim one.
SEEN_NAMES=""
ROWS=""

row() { ROWS="$ROWS  $1
"; }

# ⛔ THE EXIT CODE IS TAKEN FROM THE PROCESS, UNPIPED. Output goes to a file
# and $? is read on the next line. `run ... | tee` would report tee's status,
# which is 0 whatever the check did, and that is the single defect this whole
# repository is most emphatic about.
OUT="${TMPDIR:-/tmp}/.checkgate.$$"
mkdir -p "$OUT" || {
  printf 'check-gate: cannot write to %s\n' "$OUT" >&2
  exit 2
}
trap 'rm -rf "$OUT"' EXIT INT TERM

# ⛔ WHAT THE TREE LOOKED LIKE BEFORE ANY CHECK RAN. A check plants defects in
# disposable state and must leave the working tree exactly as it found it; every
# harness here says so in its own header and nothing compared the two.
#
# ⚠ MEASURED ON 2026-09-09, BY BREAKING IT. Two new probe cases in
# `store-lib.sh` wrote their scratch files beside the file they were handed, and
# two callers hand that function a TRACKED path - so a gate run left four
# untracked files in `scripts/capture/` and the only thing that noticed was a
# person reading `git add -A`.
#
# ⛔ IT IS A COMPARISON AND NEVER A REQUIREMENT OF A CLEAN TREE. This runs while
# somebody is editing, so demanding a clean tree would refuse the ordinary case;
# what is refused is a tree the gate itself moved.
#
# ⚠ SO IT SEES THE RUN THAT MAKES THE MESS AND NOT THE ONE AFTER. Measured while
# planting the defect above: the first run over a planted tree went red and the
# second went green, because the droppings the first left were already there when
# the second read its `before`. ⭐ A CI lane starts from a fresh checkout, so
# there it fires every time; on a host that already carries the dirt it does not,
# which is a property of a before-and-after comparison rather than of this one.
TREE_BEFORE="$OUT/tree-before"
if command -v git >/dev/null 2>&1 &&
  git -C "$HERE" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  git -C "$HERE" status --porcelain >"$TREE_BEFORE" 2>/dev/null || : >"$TREE_BEFORE"
else
  rm -f "$TREE_BEFORE"
fi

# ⚠ NO PRESENCE TEST LIVES HERE. An earlier version tested `$1` after the
# shift, which is the interpreter rather than the script, so every row reported
# "not present" and the runner printed a green verdict having executed nothing.
# ⭐ That is the exact defect this script's header is about, produced by the
# script itself on its first run. Presence is decided by the caller, which is
# the only place that knows the path.
# ⭐ THE CHECKS RUN CONCURRENTLY AND THEIR VERDICTS ARE READ IN LIST ORDER.
# Every check here is hermetic - each makes its own scratch directory, binds only
# loopback and reads only the tree - so nothing about running them at once
# changes what any of them answers. What it changes is the wall clock, and that
# multiplies: `check-workflow` runs this whole gate NINE times, so a second saved
# here is nine seconds saved on the lane.
#
# ⚠ MEASURED BEFORE IT WAS CHANGED, because "the gate is slow" is not a place to
# start optimising. On this host the 29 checks took 198 seconds and three of them
# were 183 of it: check-twins 90.7s, check-capture-client 47.9s, check-capture
# 44.6s. Everything else together is under fifteen seconds, so serialising them
# behind those three was the whole cost.
#
# ⛔ THE EXIT CODE IS STILL READ FROM THE PROCESS THAT PRODUCED IT. `wait "$pid"`
# returns that child's status and nothing else's, which is the same guarantee the
# serial form had; a `for` loop over a pipeline's status is what this repository
# refuses, and there is no pipeline here.
#
# ⛔ AND THE ROWS ARE STILL IN LIST ORDER. Each check writes to a log of its own
# and its row is assembled at its own index, so a report cannot come out in the
# order the checks happened to finish - which would make two runs of one tree
# produce two different reports.
JOBS=0

queue() { # name command...
  if [ "$ROWS_ONLY" = 1 ]; then
    printf '%s\n' "$1"
    return 0
  fi
  JOBS=$((JOBS + 1))
  printf '%s\n' "$1" >"$OUT/name.$JOBS"
  rm -f "$OUT/miss.$JOBS"
  shift
  "$@" >"$OUT/log.$JOBS" 2>&1 &
  printf '%s\n' "$!" >"$OUT/pid.$JOBS"
}

# A row that needs no process: a check that is not present, or one this runner
# declares unavailable. ⚠ It is queued rather than printed so that it keeps its
# place among the rows the concurrent checks produce.
#
# ⛔ AN `n/a` IS DECLARED WITH A REASON OR IT IS NOT DECLARED. The reason is the
# whole difference between that row and an observed skip: one is a fact about the
# platform that somebody wrote down and can be argued with, and the other is a
# check that stopped working. A declared row without a reason turns `--strict`
# back into the flag that could not be used.
queue_row() { # name row-text kind
  if [ "$ROWS_ONLY" = 1 ]; then
    printf '%s\n' "$1"
    return 0
  fi
  JOBS=$((JOBS + 1))
  printf '%s\n' "$1" >"$OUT/name.$JOBS"
  printf '%s\n%s\n' "$3" "$2" >"$OUT/miss.$JOBS"
  rm -f "$OUT/pid.$JOBS"
}

HARVESTED=0

harvest() {
  _i=$((HARVESTED + 1))
  while [ "$_i" -le "$JOBS" ]; do
    _name=$(cat "$OUT/name.$_i")
    if [ -f "$OUT/miss.$_i" ]; then
      _kind=$(sed -n 1p "$OUT/miss.$_i")
      row "$(sed -n 2p "$OUT/miss.$_i")"
      case "$_kind" in
        skip) SKIP=$((SKIP + 1)) ;;
        *) NA=$((NA + 1)) ;;
      esac
      _i=$((_i + 1))
      continue
    fi
    wait "$(cat "$OUT/pid.$_i")"
    _rc=$?
    case "$_rc" in
      0)
        row "✅ ok    $_name"
        PASS=$((PASS + 1))
        ;;
      2)
        row "SKIP  $_name  ($(head -1 "$OUT/log.$_i" 2>/dev/null | cut -c1-60))"
        SKIP=$((SKIP + 1))
        ;;
      *)
        row "❌ FAIL  $_name  (exit $_rc)"
        FAIL=$((FAIL + 1))
        # ⛔ THE TAIL, NOT THE HEAD. Every check here prints its verdict last, and
        # the mutation harnesses print dozens of passing rows first. Measured on
        # 2026-09-08: a red CI lane showed `FAIL check-capture` followed by eleven
        # PASSING rows and nothing else, so the log did not contain the failure and
        # it had to be reproduced locally to be found. store_report reprints the
        # failing rows just above its summary so this excerpt lands on them.
        [ "$JSON" = "1" ] || sed 's/^/          /' "$OUT/log.$_i" | tail -20
        ;;
    esac
    _i=$((_i + 1))
  done
  HARVESTED=$JOBS
}

have_pwsh=0
command -v pwsh >/dev/null 2>&1 && have_pwsh=1

# ⭐ EVERY EXAMPLE THE HARNESSES NEED, BUILT ONCE, BEFORE ANY OF THEM STARTS.
# Most of the checks below call `cargo build --example` through `store_build`,
# and cargo takes a lock on the target directory: started at once they would
# queue behind each other and the concurrency above would buy nothing. ⚠ This is
# the same work, done once instead of nine times, so it costs nothing on a cold
# tree and is a no-op on a warm one.
#
# ⛔ ITS FAILURE IS NOT A GATE ROW. A build that fails here fails again inside
# whichever harness needed it, where it is reported as that check's own exit 2
# with that check's own name - and inventing a row for it would be a verdict
# about a subject no entry owns.
if command -v cargo >/dev/null 2>&1; then
  ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)
  cargo build --manifest-path "$ROOT/Cargo.toml" -p bit-ids -p bit-ids-probe \
    --locked --examples >/dev/null 2>&1 || :
fi

# ⭐ THE PORTED RULES, RUN FROM ONE BINARY THAT BOTH LANES SHARE. CI-10. Each of
# these had an `.sh` half and a hand-written `.ps1` twin, and `check-twins` ran
# both of every pair to catch the drift between them. ⛔ One implementation cannot
# drift from itself, so the pair is gone rather than compared - and the row is now
# the SAME row on both lanes rather than a row here and a declared gap there.
#
# ⚠ THE BUILD IS ONE COMMAND AND ITS FAILURE IS NOT A GATE ROW, for the reason the
# cargo build above is not: a build that fails here fails again inside
# `check-bitcheck`, where it is reported with that check's own name. What IS a row
# is each check, and a missing binary makes every one of them a SKIP rather than a
# pass, because `queue` would have nothing to run.
GOBIN="$OUT/bit-check"
if command -v go >/dev/null 2>&1; then
  (cd "$HERE/../../tools/check" && go build -o "$GOBIN" .) >/dev/null 2>&1 || :
fi

# ⭐ EVERY `common/` CHECK THE GATE RUNS IS PORTED NOW. `check-project` joined
# this list on 2026-09-17 and the direct call below it is gone with the script it
# called; the comment that used to sit there named it *the last one that is not
# ported*. ⚠ This line then said *every `common/` half* while `check-remote-items`
# still ran from its `.sh` half, which is why it names what the gate RUNS.
# ⭐ And since 2026-09-29 no `common/` pair is left at all: `mine-repo`, a miner
# rather than a row, was the last, and it is `bit-check mine-repo` now.
# ⭐ AND THE HARNESSES, 2026-09-29: `check-cache` and `check-catalogue` left the
# prover list below for this one, each compared case for case against both of
# its halves first. They build Rust examples, which the build above has already
# made a no-op.
# ⚠ `check-remote-items` joined the same day and NEEDS gh AND THE NETWORK, so it
# exits 2 on a machine without them and that reads as a skip rather than a pass.
# That is correct: nothing was verified. Its logic is proved without either by
# `check-bitcheck`, which serves it from a stub `gh` and a stub `curl`.
for c in check-adapters check-cache check-catalogue check-changelog \
  check-control-bytes check-docs check-ignores check-licences check-markers \
  check-no-secrets check-one-home check-placeholders check-project \
  check-remote-items; do
  if [ -x "$GOBIN" ]; then
    queue "$c" "$GOBIN" "$c"
  else
    queue_row "$c" "SKIP  $c  (tools/check did not build)" skip
  fi
done

# ⚠ --public is a DIFFERENT question from the default run, not a stricter one.
# Emails, absolute home paths and long hex are legitimate content in a private
# project, so this row is a second call rather than a flag on the first.
# ⚠ THE ROW LABEL NAMES THE QUESTION RATHER THAN THE FLAG, and it is the same
# label on both lanes. It was built from the flag once, and because each half
# spelled its own - `--public` here and `-Public` there - the two runners' row
# lists differed on a row they both have, which is a false difference in the one
# comparison that exists to find real ones.
# ⭐ BOTH LANES NOW SPELL THE FLAG THE SAME WAY, because both call one binary.
# The label stays named after the question rather than the flag anyway: it is
# what a reader of the row list is looking for, and the next ported mode may
# differ again.
if [ -x "$GOBIN" ]; then
  queue "check-no-secrets (public)" "$GOBIN" check-no-secrets --public
else
  queue_row "check-no-secrets (public)" "SKIP  check-no-secrets (public)  (tools/check did not build)" skip
fi

# ⛔ NOT IN common/, AND IN THE GATE ANYWAY. `check-runner` mutation-proves the
# guards that stand between this project and installing an untrusted client on a
# machine somebody keeps. It is hermetic, it takes no network, and it is the one
# check whose silence would be worst, so it runs on every gate rather than only
# where captures happen. It is not in check-twins' pair list. ⚠ This said
# *because it has no PowerShell half* long after `CI-03` wrote one: the reason
# is that `check-runner.ps1` proves the Windows guards and this proves the Linux
# ones, so the two answer different questions and no one verdict holds both.
# scripts/README.md carries the history.
RUNNER="$HERE/../acquisition/check-runner.sh"
if [ -f "$RUNNER" ]; then
  queue "check-runner" sh "$RUNNER"
else
  queue_row "check-runner" "SKIP  check-runner  (not present)" skip
fi

# ⛔ MOSTLY NOT IN common/, AND IN THE GATE FOR THE SAME REASON. These
# mutation-prove the guards standing between this project and silently deleting
# or rewriting published evidence, publishing a record whose evidence nothing
# can resolve, pointing a consumer at a superseded build, publishing a retracted
# measurement in a rendering the lookups had stopped naming, shipping two
# different byte sets under one release label, force-pushing over the data
# branch, opening a capture request twice for one release, telling a consumer it
# may cache a path that moves, handing a consumer a record no manifest describes,
# documenting a command that does not work, publishing a contributor handbook
# whose walkthrough does not, capturing on a host that was never claimed or
# still has a route off it, attesting that a build was measured over a run where
# nothing announced, and keeping somebody
# else's installer in this repository. The first is
# unrecoverable afterwards and the rest are worse than errors, because each
# answers confidently. ⭐ Every one is hermetic:
# check-publish and check-access each create their own bare repository in a
# scratch directory and touch no real remote, and check-capture binds only
# loopback and reads a routing table it wrote itself.
# ⚠ They need cargo, so they exit 2 on a host without
# one, which is a skip and not a pass. ⚠ check-staleness needs python3 as well,
# which is what re-derives a request identifier independently of the encoder
# under test, and check-capture needs curl, which is the HTTP client it drives
# the observer with. None has a PowerShell half; scripts/README.md carries why,
# and ⭐ check-capture is the one that runs the PowerShell half of its SUBJECT
# anyway, because capture-run.ps1 takes a route table as a file.
#
# ⚠ THE COUNT IS NOT WRITTEN HERE. It was, twice, and both copies went stale in
# the commit that added a prover. The list below is the list.
#
# ⚠ common/check-examples AND common/check-handbook ARE IN THIS LIST AND NOT IN
# THE LOOP ABOVE, because each needs cargo and a scratch directory like the rest
# of this group rather than being a text rule over the tree, and because
# check-twins pairs the common/ checks and neither has a PowerShell half.
#
# ⚠ ci/check-staleness AND ci/check-step-bodies ARE IN THIS LIST AND
# ci/check-workflow IS NOT. They sit in one directory and differ in one property:
# two of check-workflow's cases run this gate, so a runner that listed it would
# re-enter itself. Neither of the other two runs a gate - check-step-bodies runs
# two capture workflows' step bodies against stubbed guards and a stub product,
# and the capture workflows are dispatch-only and call no gate.
#
# ⛔ THE TWO CAPTURE HARNESSES ARE NOT IN THIS LIST, AND THE BATCH BELOW IS WHY.
# Each drives real sockets against a run deadline, and the first concurrent run
# of this gate turned one of them red: `check-capture` reported *a run that
# recorded no bytes is refused (exit 1, but did not say 'recorded no bytes at
# all')* - a refusal that arrived for a reason the case had not planted - while
# the same harness on the same tree passed alone minutes later. A faster gate
# that is red for no defect is worse than a slow one.
#
# ⚠ RAISING THE DEADLINE WAS TRIED FIRST AND MEASURED, because it looked like the
# cheaper answer: those harnesses wait on the observer's own line, so a bigger
# maximum should be nearly free. ⛔ It is not. Several of their cases are ones the
# deadline itself has to end, so `check-capture` goes from 45 seconds at `3` to
# **79 at `6`** and `check-capture-client` from 48 at `5` to **168 at `20`** -
# more gate time than the whole concurrency saves, paid nine times over by
# `check-workflow`. The numbers live beside each `SECS`.
for spec in acquisition/check-release-route \
  acquisition/check-source-route acquisition/check-rootless \
  corpus/check-store \
  common/check-shell \
  corpus/check-corpus corpus/check-indexes publishing/check-release \
  publishing/check-formats publishing/check-publish publishing/check-access \
  publishing/check-bundle \
  ci/check-staleness ci/check-step-bodies \
  capture/check-assemble capture/check-connector common/check-examples \
  common/check-handbook common/check-gate-rows common/check-public-row \
  common/check-bitcheck; do
  PROVER="$HERE/../$spec.sh"
  NAME=${spec#*/}
  # ⛔ A ROW NAME IS A NAME AND TWO CHECKS MUST NOT SHARE ONE. The label is the
  # basename, so two harnesses in different directories collide into two rows a
  # reader cannot tell apart - and a red one then names a file that is not the
  # one that failed. ⚠ Measured on 2026-09-09: an acquisition harness called
  # `check-release` was added beside `publishing/check-release`, and the gate
  # would have printed the name twice with nothing saying so. It is the
  # non-injective-path defect this project already refuses in a store layout,
  # arriving in a runner's own labels.
  # ⛔ Exit 2: a gate that cannot label its own rows has not run.
  case " $SEEN_NAMES " in
    *" $NAME "*)
      printf 'check-gate: two checks are both named %s; a row name is a name\n' "$NAME" >&2
      exit 2
      ;;
  esac
  SEEN_NAMES="$SEEN_NAMES $NAME"
  if [ -f "$PROVER" ]; then
    queue "$NAME" sh "$PROVER"
  else
    queue_row "$NAME" "SKIP  $NAME  (not present)" skip
  fi
done

# ⭐ THE SLOW ONE. Measured on one Windows 11 Pro 26200 machine, 2026-08-28:
# check-twins alone is most of a full run's wall time, because it starts both
# halves of every pair. --fast drops it and nothing else.
#
# ⭐ AND IT IS WHY THE CONCURRENCY IS WORTH HAVING: a concurrent run finishes
# when its longest member does, and this is that member. ⚠ Queue order is not
# start order here - nothing bounds how many run at once, so every check is
# already running by the time this one is queued - and the ROW still prints in
# list order, so the report does not move.
if [ "$FAST" = "1" ]; then
  queue_row "check-twins" "n/a   check-twins  (--fast)" na
elif [ -f "$HERE/check-twins.sh" ]; then
  queue "check-twins" sh "$HERE/check-twins.sh"
else
  queue_row "check-twins" "SKIP  check-twins  (not present)" skip
fi

# ⛔ NOTHING HAS BEEN WAITED ON UNTIL HERE. Every queue above returned as soon as
# it had a process; this is where their exit codes are read, in list order.
harvest

# ⛔ AND THE TWO THAT DRIVE SOCKETS RUN AFTER THAT BATCH HAS FINISHED, with the
# machine to themselves and to each other. Each measures whether an observer
# answered inside a deadline, so a saturated host changes their ANSWER and not
# merely their duration - and a refusal that arrives for the wrong reason is a
# case reporting a guard proved by something it did not plant.
#
# ⚠ TWO AT ONCE IS DELIBERATE AND ONE AT A TIME IS NOT NEEDED. They bind port
# zero, so the operating system separates them, and each holds its own scratch
# directory; two processes on a four-processor host is not the condition that
# broke either.
#
# ⚠ THEIR ROWS THEREFORE PRINT LAST, which is a change to the report's order and
# should be read as what it is: these two ran apart from the rest.
for spec in capture/check-capture capture/check-capture-client; do
  PROVER="$HERE/../$spec.sh"
  NAME=${spec#*/}
  case " $SEEN_NAMES " in
    *" $NAME "*)
      printf 'check-gate: two checks are both named %s; a row name is a name\n' "$NAME" >&2
      exit 2
      ;;
  esac
  SEEN_NAMES="$SEEN_NAMES $NAME"
  if [ -f "$PROVER" ]; then
    queue "$NAME" sh "$PROVER"
  else
    queue_row "$NAME" "SKIP  $NAME  (not present)" skip
  fi
done
harvest

# ⛔ AND THE ROW THIS RUNNER DECIDES FOR ITSELF IS NAMED HERE TOO. `tree-unchanged`
# is not queued - it is answered after every check has run - so a rows mode that
# listed the queue alone would be short by exactly the row nothing else names,
# and the comparison would then report two lanes in step over a row one of them
# does not have.
if [ "$ROWS_ONLY" = 1 ]; then
  printf 'tree-unchanged\n'
  exit 0
fi

# ⚠ THE POWERSHELL HALVES ARE NOT RE-RUN HERE. check-twins already runs both
# halves of every pair and compares them, so running them again would double
# the slowest part of the gate to learn nothing. On a machine with no pwsh at
# all, check-twins reports that itself.
[ "$have_pwsh" = "1" ] || row "note  pwsh absent; the PowerShell halves were not exercised"

# ⛔ AND THE TREE IS COMPARED AGAINST WHAT IT WAS. A harness that planted into
# the working tree, or left a scratch file in it, is a harness whose next run
# starts from a tree the last one wrote. It counts as a failed check rather than
# a note, because every case downstream of such a run is measuring something
# nobody chose.
if [ -f "$TREE_BEFORE" ]; then
  git -C "$HERE" status --porcelain >"$OUT/tree-after" 2>/dev/null || : >"$OUT/tree-after"
  if diff "$TREE_BEFORE" "$OUT/tree-after" >"$OUT/tree-diff" 2>&1; then
    row "✅ ok    tree-unchanged"
    PASS=$((PASS + 1))
  else
    row "❌ FAIL  tree-unchanged  (a check moved the working tree)"
    row "note  $(grep '^[<>]' "$OUT/tree-diff" | head -6 | tr '\n' ' ')"
    FAIL=$((FAIL + 1))
  fi
else
  row "SKIP  tree-unchanged  (not a git work tree)"
  SKIP=$((SKIP + 1))
fi

TOTAL=$((PASS + FAIL + SKIP + NA))

# ⛔ A RUN THAT PASSED NOTHING IS NOT A GREEN RUN. Zero failures out of zero
# checks executed is the shape this script exists to refuse, and it produced
# exactly that on its own first run through a broken presence test. Nothing
# passing is a failure of the gate regardless of --strict.
#
# ⚠ --strict reads SKIP and never NA. A declared row is a documented gap and
# refusing it would refuse every correct tree on the lane that declared it.
if [ "$PASS" -eq 0 ]; then
  RC=1
elif [ "$STRICT" = "1" ] && [ "$SKIP" -gt 0 ]; then
  RC=1
elif [ "$FAIL" -gt 0 ]; then
  RC=1
else
  RC=0
fi

if [ "$JSON" = "1" ]; then
  printf '{"schema":"check-gate/2","total":%s,"passed":%s,"failed":%s,"skipped":%s,"unavailable":%s,"strict":%s}\n' \
    "$TOTAL" "$PASS" "$FAIL" "$SKIP" "$NA" "$([ "$STRICT" = "1" ] && printf true || printf false)"
  exit "$RC"
fi

printf '\n%s\n' "$ROWS"
printf '%s checks: %s passed, %s failed, %s skipped, %s unavailable\n' \
  "$TOTAL" "$PASS" "$FAIL" "$SKIP" "$NA"

if [ "$SKIP" -gt 0 ]; then
  printf -- '⚠ A SKIP IS NOT A PASS. Those checks did not run and nothing about\n'
  printf 'their subject was verified. Pass --strict to make a skip a failure.\n'
fi
if [ "$NA" -gt 0 ]; then
  printf -- '⚠ An n/a row is a gap this runner declares, with the reason beside it.\n'
  printf -- '--strict permits those and refuses a SKIP, so read both numbers.\n'
fi
if [ "$PASS" -eq 0 ]; then
  printf -- '❌ NOTHING RAN. Zero checks passed, so this is red whatever the skips say.\n'
elif [ "$FAIL" -gt 0 ]; then
  printf -- '❌ the gate is red.\n'
else
  printf -- '✅ nothing failed.\n'
  printf -- '⚠ That is part (a) of the gate only. Driving the real thing and the\n'
  printf 'deep reviews are the other two, and each is blind to what this catches.\n'
  printf 'docs/methodology/gate.md.\n'
fi
exit "$RC"
