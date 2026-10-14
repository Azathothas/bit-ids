#!/bin/sh
# check-workflow.sh - does the workflow actually go red for each class of
# defect it is supposed to catch?
#
# CI-01's Prove has two halves. One is that every required check appears as a
# non-skipped CI result, which is what check-gate's --strict answers now that a
# declared unavailability and an observed skip are counted apart. This is the
# other half: an injected failure in docs, schema, fixtures or Rust turns the
# workflow red.
#
# -- ⛔ EVERY COMMAND IT RUNS IS READ OUT OF THE WORKFLOW -------------------
#
# Nothing here spells a build command. A harness carrying its own copy of
# `cargo test --workspace --locked --all-targets` proves that command refuses a
# defect and says nothing whatever about the one CI runs, and the two would
# drift in the direction that keeps this file green. So a case names a JOB and a
# STEP, the command is pulled from .github/workflows/ci.yml, and ⭐ A STEP THIS
# FILE NAMES AND THE WORKFLOW NO LONGER HAS IS A FAILURE, never a silent pass.
# That is the same defect docs/methodology/reviews.md calls two spellings of one
# layout, and CORPUS-01 paid for it twice.
#
# -- ⛔ IT IS NOT IN check-gate.sh, AND THAT IS A SHARED CONTRACT -----------
#
# Two of the cases below run the workflow's own *Repository gate* step, which is
# check-gate.sh. A runner that appears in the gate and also runs the gate
# recurses, which is precisely why check-gate.sh keeps check-twins out of its
# own pair list. ⚠ Removing this exclusion reintroduces the hang. The workflow
# runs this file as its own step instead, so it still runs on every push, and
# scripts/README.md carries the reasoning beside check-twins'.
#
# -- ⛔ THE TREE IT PLANTS IN IS THE TREE ON DISK ---------------------------
#
# It copies the tracked and untracked working tree into a scratch directory and
# commits it there, rather than checking out HEAD. A harness that tested the
# last commit would report green over a defect somebody had just introduced and
# not yet committed, which is the state every pre-push run is in.
#
# ⚠ THE PLANTS AND THE CONTROLS ARE BOTH REQUIRED. A command that refuses the
# planted tree proves nothing unless it accepts the clean one: a step that is
# broken for an unrelated reason refuses everything, and reads here as a guard
# working perfectly.
#
# -- ⭐ IT SHARDS, BECAUSE IT IS THE CI WALL CLOCK --------------------------
#
# `CI-10`. Measured on run 122, 2026-09-10, on a real runner: *Workflow
# acceptance* is **32.6 minutes** where the Linux gate is 5.8 and the Windows
# gate 2.9. It runs the whole repository gate NINE times, and a 130-second gate
# is about twenty minutes of that whatever language its rows are written in. ⛔ So
# the only thing that touches this number is running the units on more than one
# machine.
#
# ⛔ THE PARTITION IS `index mod N` AND IT IS PROVABLE BY CONSTRUCTION. A shard
# selector that silently claimed no case would report a faster green run over
# less work, which is this repository's oldest defect class and the one a
# wall-clock target makes most tempting. Every unit is named, the runner prints
# what it ran and what it skipped, `--units` lists the names and runs nothing,
# and `scripts/ci/check-shards.sh` is what asserts the N shards together cover
# every unit exactly once.
#
# -- ⛔ AND THE CONTROLS ARE NOT SHARDED. THIS IS THE WHOLE DESIGN ------------
#
# A plant that is refused proves nothing unless the clean tree is accepted: a
# step broken for an unrelated reason refuses everything, and reads here as a
# guard working perfectly. ⚠ So a shard carrying only plants would go GREEN over
# a tree where every single step was broken - which is the exact failure mode
# sharding is most likely to introduce, dressed as a speedup.
#
# ⭐ Every shard therefore runs the whole control block and the final clean-tree
# control, and shards only the plants. ⚠ That is a FLOOR on what a shard costs -
# the controls include two of the nine gate runs - and it is paid on purpose.
# `TODO/ci.md` carries the measured floor beside the measured total.
#
# Usage:
#   sh scripts/ci/check-workflow.sh
#   sh scripts/ci/check-workflow.sh --json
#   sh scripts/ci/check-workflow.sh --shard 2/4
#   sh scripts/ci/check-workflow.sh --units     # the unit names, running nothing
#
# Exit codes: 0 every case held, 1 one did not, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

JSON=0
SHARD_I=1
SHARD_N=1
UNITS_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON=1 ;;
    --units) UNITS_ONLY=1 ;;
    --shard)
      shift
      case "${1:-}" in
        [0-9]*/[0-9]*)
          SHARD_I=${1%%/*}
          SHARD_N=${1##*/}
          ;;
        *)
          printf 'check-workflow: --shard wants i/N, got: %s\n' "${1:-}" >&2
          exit 2
          ;;
      esac
      ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    *)
      printf 'check-workflow: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

# ⛔ THE SELECTOR IS VALIDATED BEFORE ANYTHING RUNS, because every wrong value
# fails SILENTLY in the direction of running less. `--shard 0/4` and `--shard
# 5/4` each select no unit at all, and a run of nothing but controls is exactly
# what a green shard covering no work looks like.
case "$SHARD_N" in
  '' | *[!0-9]*)
    printf 'check-workflow: --shard N is not a number: %s\n' "$SHARD_N" >&2
    exit 2
    ;;
esac
case "$SHARD_I" in
  '' | *[!0-9]*)
    printf 'check-workflow: --shard i is not a number: %s\n' "$SHARD_I" >&2
    exit 2
    ;;
esac
if [ "$SHARD_N" -lt 1 ]; then
  printf 'check-workflow: --shard N must be at least 1\n' >&2
  exit 2
fi
if [ "$SHARD_I" -lt 1 ] || [ "$SHARD_I" -gt "$SHARD_N" ]; then
  printf 'check-workflow: --shard i must be between 1 and %s, got %s\n' "$SHARD_N" "$SHARD_I" >&2
  exit 2
fi

# ⛔ Resolved from this script's own location, never from the working directory.
HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# ⚠ ME is set here and read by store-lib.sh, which this sources on the next
# line. shellcheck cannot see across a source it is not told to follow, so it
# reads as unused unless every file is handed to one invocation.
# shellcheck disable=SC2034
ME=check-workflow
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require git cargo awk tar sha256sum

WORKFLOW="$ROOT/.github/workflows/ci.yml"
[ -f "$WORKFLOW" ] || {
  printf 'check-workflow: %s is missing\n' "$WORKFLOW" >&2
  exit 2
}

WORK=$(store_workdir checkworkflow) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

TREE="$WORK/tree"
LOG="$WORK/log"
ORIG="$WORK/orig"

# ⭐ ONE TARGET DIRECTORY, SHARED BY EVERY CASE. A fresh one per plant would
# rebuild the workspace each time and turn a two-minute check into twenty.
CARGOTGT="$WORK/target"

# -- the scratch tree ---------------------------------------------------------
#
# ⚠ Tracked AND untracked-but-not-ignored, because an uncommitted new file is
# part of the tree the next push carries. target/ is ignored, so the 1G build
# directory is not copied.
mkdir -p "$TREE" || exit 2
(
  cd "$ROOT" || exit 1
  {
    git ls-files -z
    git ls-files -zo --exclude-standard
  } | tar --null -T - -cf -
) | (cd "$TREE" && tar -xf -) || {
  printf 'check-workflow: cannot copy the working tree\n' >&2
  exit 2
}

# ⛔ IT IS A REAL REPOSITORY, because half the checks the gate runs ask git what
# is tracked. A plain directory would make every one of them exit 2, and a run
# of nothing but skips is what --strict exists to refuse.
#
# ⚠ AND IT CARRIES THE ORIGIN, which is not decoration either. check-remote-items
# resolves the repository from that remote, so a scratch tree without one made
# that check answer 2 wherever it ran, including on a CI runner that could
# perfectly well have run it. The gate cases then never saw a clean tree exit 0
# and their exit codes did no work anywhere. Measured on run 37.
ORIGIN=$(cd "$ROOT" && git remote get-url origin 2>/dev/null) || ORIGIN=""
(
  cd "$TREE" || exit 1
  # ⚠ The identity carries no at-sign on purpose. check-no-secrets --public
  # reads one as an email address and refused this file over its own scratch
  # commit; git accepts a bare word here and the commit is thrown away anyway.
  git init -q -b main &&
    git add -A &&
    git -c user.email=gate -c user.name=gate commit -qm scratch &&
    { [ -z "$ORIGIN" ] || git remote add origin "$ORIGIN"; }
) >/dev/null 2>&1 || {
  printf 'check-workflow: cannot make the scratch tree a repository\n' >&2
  exit 2
}

# -- reading a command out of the workflow ------------------------------------
#
# ⛔ THE READER IS ASKED FOR, NOT SPELLED HERE. `check-step-bodies.sh` executes
# the capture workflows' bodies and this one executes `ci.yml`'s, so a copy of
# the parser in each is a second answer to what a step actually runs - and the
# copies would drift where it matters, because a reader that stopped matching
# after a re-indent reports an empty body and an empty body reads exactly like a
# step the workflow no longer has.
#
# ⚠ THE THREE STATUSES ARE COLLAPSED HERE AND NOT THERE, deliberately. This
# caller has always treated "no such step" and "an action rather than a command"
# alike, and `run_step` turns the empty answer into 127; the new harness keeps
# them apart because a step it names and the workflow no longer has is rot.
step_command() { # workflow job step
  sh "$ROOT/scripts/ci/workflow-step.sh" --workflow "$1" --job "$2" --step "$3" || true
}

# Every job that does not declare KEY at the job level, one per line.
#
# ⛔ BOTH DIRECTIONS MATTER AND ONLY ONE IS OBVIOUS. Reporting the jobs that
# lack the key is the rule; what makes the rule hold as jobs are added is that
# it enumerates the jobs itself rather than being handed a list. A rule given
# the two job names it knows about would pass forever over a third.
jobs_missing() { # workflow key
  awk -v KEY="$2" '
    function indent(s,   i) { i = match(s, /[^ ]/); return i ? i - 1 : -1 }
    /^jobs:[ \t]*$/ { injobs = 1; next }
    !injobs { next }
    {
      line = $0
      sub(/\r$/, "", line)
      ind = indent(line)
      if (ind < 0) next
      if (ind == 0) {
        if (job != "" && !seen) print job
        job = ""; injobs = 0; next
      }
      if (ind == 2 && line ~ /^ *[A-Za-z0-9_-]+:[ \t]*$/) {
        if (job != "" && !seen) print job
        job = line; sub(/^ +/, "", job); sub(/:.*$/, "", job)
        seen = 0; next
      }
      if (ind == 4 && line ~ ("^ *" KEY ":")) seen = 1
    }
    END { if (job != "" && !seen) print job }
  ' "$1"
}

# ⭐ ONE READER PER RULE, CALLED BY THE CASE AND BY THE PROBE. A rule the probe
# re-implements is a rule the probe cannot refute.
gate_asks_strictly() { # workflow job flag
  case "$(step_command "$1" "$2" 'Repository gate')" in
    *"$3"*) return 0 ;;
    *) return 1 ;;
  esac
}

declares_key() { # workflow key
  _missing=$(jobs_missing "$1" "$2")
  [ -z "$_missing" ]
}

# The top-level trigger keys of a workflow, one per line.
triggers() { # workflow
  awk '
    function indent(s,   i) { i = match(s, /[^ ]/); return i ? i - 1 : -1 }
    /^on:[ \t]*$/ { inon = 1; next }
    /^on:/ { print "inline"; next }
    !inon { next }
    {
      line = $0
      sub(/\r$/, "", line)
      ind = indent(line)
      if (ind < 0) next
      if (ind == 0) { inon = 0; next }
      if (ind == 2 && line ~ /^ *[A-Za-z_][A-Za-z0-9_]*:/) {
        k = line; sub(/^ +/, "", k); sub(/:.*$/, "", k); print k
      }
    }
  ' "$1"
}

# Every script path a workflow names, one per line, deduplicated.
#
# ⛔ A WORKFLOW NAMING A SCRIPT NOBODY WROTE READS AS FINISHED AND FAILS ON ITS
# FIRST DISPATCH. That is not hypothetical: a draft of the capture workflow was
# written and deleted because it called two scripts that did not exist, and
# nothing in this repository would have caught it. A `run:` block is not
# compiled, not linted and, on a dispatch-only workflow, not executed by any
# push, so the file can sit in `main` looking complete for as long as nobody
# presses the button.
#
# ⚠ It scans the whole file rather than only `run:` bodies, because a path can
# reach a step through `with:`, an `env:` value or a composite input, and a rule
# that read one of those places would be a gate on one of several doors.
scripts_named() { # workflow
  grep -o -E 'scripts/[A-Za-z0-9_./-]+\.(sh|ps1)' "$1" 2>/dev/null | LC_ALL=C sort -u
}

# The ordered step names of one job, one per line - asked of the same reader
# `step_command` uses, so the set of steps this file reasons about and the set it
# can execute cannot disagree.
job_steps() { # workflow job
  sh "$ROOT/scripts/ci/workflow-step.sh" --workflow "$1" --job "$2" --steps || true
}

# The 1-based position of a step in its job, or nothing when the job has no such
# step. ⛔ THE TWO ANSWERS ARE KEPT APART BY THE CALLER: a step that is missing
# is a failure and never an ordering that happens to hold.
step_index() { # workflow job step
  job_steps "$1" "$2" | grep -n -x -F -e "$3" | head -1 | cut -d: -f1
}

# Every line of one named step, its `- name:` included.
#
# ⚠ A step ends where the next `- ` at its own indent begins, or where the job
# does. Reading forward to the next `name:` would stop at an artifact name inside
# `with:`, which is a `name:` belonging to something else - the same confusion
# `check-project`'s artifact rule had to be planted against.
step_block() { # workflow job step
  awk -v WANTJOB="$2" -v WANTSTEP="$3" '
    function indent(s,   i) { i = match(s, /[^ ]/); return i ? i - 1 : -1 }
    /^jobs:[ \t]*$/ { injobs = 1; next }
    !injobs { next }
    {
      line = $0
      sub(/\r$/, "", line)
      ind = indent(line)
      if (ind < 0) { if (inblock) print ""; next }
      if (ind == 0) { injobs = 0; inblock = 0; next }
      if (ind == 2 && line ~ /^ *[A-Za-z0-9_-]+:[ \t]*$/) {
        job = line; sub(/^ +/, "", job); sub(/:.*$/, "", job)
        inblock = 0; next
      }
      if (line ~ /^ *- /) {
        if (inblock && ind <= stepind) inblock = 0
        if (job == WANTJOB && line ~ /^ *- name:/) {
          s = line
          sub(/^ *- name:[ \t]*/, "", s)
          gsub(/^["'"'"']|["'"'"']$/, "", s)
          if (s == WANTSTEP) { inblock = 1; stepind = ind }
        }
      } else if (inblock && ind <= stepind) {
        inblock = 0
      }
      if (inblock) print $0
    }
  ' "$1"
}

# Whether an evidence upload that can run on a refused capture is gated on the
# rule-12 scan. Prints nothing when it is, and the reason when it is not.
#
# ⛔ `capture-client.sh` SCANS ITS OUTPUT ON ITS SUCCESS PATH ALONE, so an upload
# that also runs after a refusal ships a directory nothing has read for a
# credential - and a refusal is the run whose adapter `stop`, which unlinks a
# product's RPC token, may never have been reached. ⚠ The upload's own `if:` is
# the input, read by the caller off the step, so an upload with no condition is
# success-only and never asks this.
SCAN_STEP="Scan the evidence of a refused capture"
scan_gate() { # workflow job upload-if
  case "$3" in
    *"steps.scan.outcome == 'success'"*) : ;;
    *)
      printf 'the evidence upload runs on [%s] with no rule-12 scan gating it' "$3"
      return
      ;;
  esac
  if ! step_block "$1" "$2" "$SCAN_STEP" | grep -q -E '^ *(- )?id:[ ]*scan[ ]*$'; then
    printf 'no step named [%s] carries the id the upload names' "$SCAN_STEP"
    return
  fi
  # shellcheck disable=SC2016
  case "$(step_command "$1" "$2" "$SCAN_STEP")" in
    *'scripts/capture/scan-bundle.sh "$RUNNER_TEMP/capture"'*) : ;;
    *)
      printf '[%s] does not run scan-bundle.sh over the bundle' "$SCAN_STEP"
      return
      ;;
  esac
  _sc=$(step_index "$1" "$2" Capture)
  _ss=$(step_index "$1" "$2" "$SCAN_STEP")
  _su=$(step_index "$1" "$2" "Upload the evidence bundle")
  if [ -z "$_sc" ] || [ -z "$_su" ] || [ "$_sc" -ge "$_ss" ] || [ "$_ss" -ge "$_su" ]; then
    printf '[%s] does not sit between the capture and the upload' "$SCAN_STEP"
  fi
}

# Whether a job that starts the flight recorder ends by stopping it. Prints
# nothing when it does, and the reason when it does not.
#
# ⛔ A RECORDER NOBODY STOPS LEAVES ITS CHECK RUN OPEN FOREVER, and an open run is
# what this project reads as a host that lost contact. So the stop is the LAST
# step, runs on every outcome, and touches the file the recorder watches.
recorder_stopped() { # workflow job
  [ -n "$(step_index "$1" "$2" "Start the flight recorder")" ] || return 0
  if [ "$(job_steps "$1" "$2" | tail -1)" != "Stop the flight recorder" ]; then
    printf 'the job starts the flight recorder and its last step is not [Stop the flight recorder]'
    return
  fi
  if [ "$(step_if "$1" "$2" "Stop the flight recorder")" != "always()" ]; then
    printf '[Stop the flight recorder] does not run on every outcome'
    return
  fi
  # shellcheck disable=SC2016
  case "$(step_command "$1" "$2" "Stop the flight recorder")" in
    *'touch "$RUNNER_TEMP/flight-recorder.stop"'*) : ;;
    *) printf '[Stop the flight recorder] does not touch the file the recorder watches' ;;
  esac
}

# The `if:` of one step, or nothing when it has none.
step_if() { # workflow job step
  step_block "$1" "$2" "$3" | awk '
    { line = $0; sub(/\r$/, "", line); sub(/^ +/, "", line); sub(/^- /, "", line) }
    line ~ /^if:/ { sub(/^if:[ \t]*/, "", line); print line; exit }'
}

# The declared default of one workflow_dispatch input.
input_default() { # workflow input
  awk -v WANT="$2" '
    function indent(s,   i) { i = match(s, /[^ ]/); return i ? i - 1 : -1 }
    {
      line = $0
      sub(/\r$/, "", line)
      ind = indent(line)
    }
    ind == 6 && line ~ /^ *[A-Za-z_][A-Za-z0-9_]*:[ \t]*$/ {
      k = line; sub(/^ +/, "", k); sub(/:.*$/, "", k); cur = k; next
    }
    cur == WANT && ind == 8 && line ~ /^ *default:/ {
      v = line; sub(/^ *default:[ \t]*/, "", v); print v; exit
    }
  ' "$1"
}

# ⛔ 127 MEANS THE WORKFLOW HAS NO SUCH STEP, and it is reported separately from
# every other status. A missing step read as a refusal would let deleting a
# whole CI step register as the guard working.
run_step() { # job step
  _cmd=$(step_command "$WORKFLOW" "$1" "$2")
  [ -n "$_cmd" ] || return 127
  (
    cd "$TREE" || exit 2
    CARGO_TARGET_DIR="$CARGOTGT" sh -c "$_cmd"
  ) >"$LOG" 2>&1
}

# -- the two verdicts ---------------------------------------------------------

control() { # job step label
  run_step "$1" "$2"
  _rc=$?
  case "$_rc" in
    0) pass "control  $3 accepts the clean tree" ;;
    127) fail "control  $3: the workflow has no step named that" ;;
    *) fail "control  $3 refused the clean tree (exit $_rc)" ;;
  esac
}

# ⛔ THE GATE STEP NEEDS ITS OWN CONTROL, because its exit code on a developer
# host is not the one CI reads. The workflow runs it with --strict, and --strict
# refuses an observed skip: this host has no authenticated gh, so
# check-remote-items answers 2 and the clean tree exits 1 here and 0 on the
# Linux lane. Reporting that as a broken control would be false, and treating it
# as a pass would hide a genuinely failing check.
#
# ⭐ So the control reads the summary line the runner prints, which counts
# FAILURES, and records which of the two hosts this is. The plants below still
# read the exit code from the process that produced it; what this establishes is
# whether that code is discriminating here.
#
# ⚠ THE SUBJECT IS AN ARGUMENT because not every control is over a clean tree.
# The cases that prove a rule does NOT fire on correct usage plant something the
# gate must accept, and reporting those as "accepts the clean tree" would name
# the wrong input in the row a reader trusts.
GATE_NOTE=""
gate_control() { # label [subject]
  _what=${2:-the clean tree}
  run_step linux "Repository gate"
  _rc=$?
  if [ "$_rc" = 127 ]; then
    fail "control  $1: the workflow has no step named that"
    return
  fi
  if ! grep -q -E '^[0-9]+ checks: .*, 0 failed,' "$LOG"; then
    fail "control  $1: $_what failed a check"
    # ⛔ AND IT SAYS WHICH. CI run 188 reported this row and nothing else, so a
    # red shard could not be read from its own log. The gate's failed rows and
    # the lines under them are what a reader needs.
    [ "$JSON" = "1" ] || grep -A12 -F 'FAIL ' "$LOG" | head -40 | sed 's/^/          /'
    return
  fi
  if [ "$_rc" = 0 ]; then
    pass "control  $1 accepts $_what (exit 0)"
  else
    GATE_NOTE=" (this host cannot reach exit 0; see the header)"
    pass "control  $1 fails no check over $_what, and exits $_rc on a host with an observed skip"
  fi
}

# ⚠ The pattern argument is what stops a plant passing for the wrong reason. A
# step can go red because the scratch tree is broken, and without asking WHICH
# check refused, that reads here as the planted guard firing.
refuses() { # job step label pattern
  run_step "$1" "$2"
  _rc=$?
  if [ "$_rc" = 127 ]; then
    fail "$3: the workflow has no step named that"
    return
  fi
  if [ "$_rc" = 0 ]; then
    fail "$3 was NOT refused; the step exited 0"
    return
  fi
  if [ -n "$4" ] && ! grep -q -F -e "$4" "$LOG"; then
    fail "$3 was refused (exit $_rc) but not by $4"
    return
  fi
  pass "$3 turns the workflow red (exit $_rc)"
}

# ⚠ The gate plants carry the control's note, so a reader of this report can see
# whether the exit code was doing any of the work on the host it ran on.
gate_refuses() { # label pattern
  refuses linux "Repository gate" "$1$GATE_NOTE" "$2"
}

# -- ⭐ the shard selector -----------------------------------------------------
#
# ⛔ ONE DERIVATION, AND THE NAMES ARE THE LIST. `--units` prints from this same
# function and returns false, so a unit this mode does not name is a unit no
# shard runs - a separate list would be the value in two places the partition
# check exists to compare. It is the same argument `check-gate.sh --rows` makes
# about its own queue.
UNIT_INDEX=0
UNITS_RAN=""
UNITS_SKIPPED=""

# ⛔ THE SELECTOR IS APPLIED BEFORE THE MODE, and that ordering is the whole
# correctness of `--units`. Written the other way round - print the name, then
# decide - every shard LISTS every unit while running only its own, so the
# partition check compares six identical lists and reports every unit claimed six
# times. ⚠ That is exactly what it did on its first run, which is the argument for
# `check-shards.sh` existing rather than for trusting `index mod N` to be obvious.
unit() { # name -> true when this shard owns it
  UNIT_INDEX=$((UNIT_INDEX + 1))
  if [ "$(((UNIT_INDEX - 1) % SHARD_N))" -ne "$((SHARD_I - 1))" ]; then
    [ "$UNITS_ONLY" = 1 ] || UNITS_SKIPPED="$UNITS_SKIPPED $1"
    return 1
  fi
  if [ "$UNITS_ONLY" = 1 ]; then
    printf '%s\n' "$1"
    return 1
  fi
  UNITS_RAN="$UNITS_RAN $1"
  return 0
}

# ⛔ A CONTROL IS NAMED TOO, AND IT IS NAMED AS A CONTROL. `--units` prints it
# with the marker below, so the partition check can tell the two apart and assert
# that no control is ever sharded. ⚠ A control silently missing from that output
# is how "every shard runs the controls" becomes a sentence nobody checks.
always() { # name -> true unless this run is only listing
  if [ "$UNITS_ONLY" = 1 ]; then
    printf '%s\tcontrol\n' "$1"
    return 1
  fi
  return 0
}

# -- planting -----------------------------------------------------------------

keep() { # relative path
  cp "$TREE/$1" "$ORIG" || return 1
}
restore() { # relative path
  cp "$ORIG" "$TREE/$1" || return 1
}

# ⛔ AN APPEND IS VERIFIED LIKE A SUBSTITUTION. store-lib's replace_once refuses
# a literal that matched no times or twice; an append cannot miss that way, so
# what it owes instead is proof the file changed at all. A read-only file would
# otherwise leave the case reporting a guard that failed to fire.
append_once() { # file text
  _before=$(sha256sum "$1" | cut -d' ' -f1)
  printf '%s\n' "$2" >>"$1" || return 1
  _after=$(sha256sum "$1" | cut -d' ' -f1)
  [ "$_before" != "$_after" ]
}

# -- the probe's own guards ---------------------------------------------------
if always probe-guards; then
  store_probe_guards "$TREE/README.md" \
    "A dated, provenance-carrying catalogue" "bit-ids"
fi

# -- 0. the flag the whole first half of the Prove rests on -------------------
#
# ⛔ READ OUT OF THE WORKFLOW, NOT ASSERTED ABOUT IT. Every planted case below
# proves that a defect makes some check answer non-zero. None of them can prove
# that CI is asking strictly, because a skip is only a failure when the flag is
# there, and on a host with an observed skip of its own the exit code cannot
# tell the two apart. This case is what closes that, and it is why the two are
# written as separate rows rather than one.
if unit strict-flags; then
  if gate_asks_strictly "$WORKFLOW" linux '--strict'; then
    pass "workflow  the Linux gate step asks strictly"
  else
    fail "workflow  the Linux gate step does not pass --strict"
  fi

  if gate_asks_strictly "$WORKFLOW" windows '-Strict'; then
    pass "workflow  the Windows gate step asks strictly"
  else
    fail "workflow  the Windows gate step does not pass -Strict"
  fi
fi

# -- 0b. the job-scoped properties the entry's Approach names -----------------
#
# ⚠ Declared PER JOB rather than inherited. A job that states no permissions
# runs with whatever the workflow's floor happens to be that week, and the
# person who widens that floor for one job widens it for every job that never
# said anything. The same argument holds for a timeout: the default is six
# hours, so a job with none is a job that hangs for an afternoon.
#
# ⛔ EVERY WORKFLOW, NOT THE ONE THIS FILE CARES ABOUT. A rule that read ci.yml
# alone would be a gate on one of two doors the moment a second workflow landed,
# and a second workflow did land: the publisher is the one job in this
# repository that may write, so it is exactly the file a permissions rule must
# not skip. `check-project`'s action-pin rule generalises the same way.
if unit job-properties; then
  for wf in "$ROOT"/.github/workflows/*.yml; do
    [ -f "$wf" ] || continue
    wfname=${wf##*/}
    for key in permissions timeout-minutes; do
      if declares_key "$wf" "$key"; then
        pass "workflow  $wfname: every job declares $key"
      else
        fail "workflow  $wfname: jobs with no $key: $(jobs_missing "$wf" "$key" | tr '\n' ' ')"
      fi
    done
    if grep -q '^concurrency:' "$wf" && grep -q 'cancel-in-progress:' "$wf"; then
      pass "workflow  $wfname declares a concurrency group"
    else
      fail "workflow  $wfname declares no concurrency group"
    fi

    # ⛔ EVERY SCRIPT IT NAMES IS ON DISK. See scripts_named above for why this is
    # the rule a dispatch-only workflow most needs: nothing else in the pipeline
    # ever executes one.
    MISSING=""
    for s in $(scripts_named "$wf"); do
      [ -f "$ROOT/$s" ] || MISSING="$MISSING $s"
    done
    if [ -z "$MISSING" ]; then
      NAMED=$(scripts_named "$wf" | grep -c . || true)
      pass "workflow  $wfname: all $NAMED script(s) it names are in the tree"
    else
      fail "workflow  $wfname names scripts that do not exist:$MISSING"
    fi
  done
fi

# -- 0c. the publisher cannot fire on its own ---------------------------------
#
# ⛔ NOTHING HAS EVER BEEN PUBLISHED AND NOTHING MAY BE until a measured record
# exists, because everything in the tree is synthetic. The publisher's workflow
# is therefore dispatch-only, and its dry run is the default so that dispatching
# it by accident still pushes nothing. ⚠ Both are properties a reader can check,
# which is why they are cases here rather than a sentence in the file.
# ⛔ HOISTED OUT OF THE UNIT BELOW, because `static-readers` reads it too. A path
# assigned inside one unit and read inside another is a coupling sharding turns
# into an unbound variable the moment the two land on different runners - and
# `set -u` makes that a hard exit rather than an empty string, which is the only
# reason it was loud. `check-shards.sh` refuses the shape now.
PUBWF="$ROOT/.github/workflows/publish-data.yml"

if unit publisher-trigger; then
  if [ -f "$PUBWF" ]; then
    PUBTRIG=$(triggers "$PUBWF" | tr '\n' ' ' | sed 's/ *$//')
    if [ "$PUBTRIG" = "workflow_dispatch" ]; then
      pass "workflow  the publisher can only be dispatched by hand"
    else
      fail "workflow  the publisher has other triggers: $PUBTRIG"
    fi

    PUBDRY=$(input_default "$PUBWF" dry_run)
    if [ "$PUBDRY" = "true" ]; then
      pass "workflow  the publisher dry-runs unless told otherwise"
    else
      fail "workflow  the publisher's dry_run default is [$PUBDRY], not true"
    fi
  else
    fail "workflow  there is no publisher workflow to check"
  fi
fi

# -- 0c-bis. the capture workflow cannot be started by a fork -----------------
#
# ⛔ THE FORK GUARD IS AN ABSENCE, WHICH IS WHY IT IS ASSERTED HERE. A fork
# cannot cause a workflow to run in the base repository, so a capture workflow
# with no `pull_request` trigger has no path from a fork at all: there is no
# job-level condition to weaken and no `if:` to get subtly wrong. ⚠ An absence
# is exactly what a later edit restores unnoticed, and a comment in the file
# saying so is not a check.
#
# ⛔ AND NO `push` OR `schedule` EITHER. This is the file a client installer
# lands in, and a capture host is a machine whose route off itself has been
# deleted. Neither belongs one merge away.
# ⛔ EVERY CAPTURE WORKFLOW, NOT THE FIRST ONE. This block asserted
# `capture.yml` alone while `capture-client.yml` was being written, which is the
# one-gated-door shape over a rule rather than over a code path: a second file
# that installs somebody else's binary and deletes a route would have inherited
# none of it. ⚠ The list is derived from the directory rather than typed, so a
# third capture workflow is covered the day it lands and not the day somebody
# remembers this line.
# ⛔ HOISTED OUT OF THE UNIT BELOW, AND SO IS THE ONE PATH `static-readers` USES.
# The reader-refutation unit mutates a capture workflow, and it used to reach for
# whatever `$CAPWF` the loop below happened to leave behind - a dependency on a
# LOOP VARIABLE from another unit, which is the same coupling as a hoisted path
# and considerably harder to see. `CAPONE` is derived here instead, so each unit
# names what it reads.
CAPTURE_WORKFLOWS=""
for one in "$ROOT"/.github/workflows/capture*.yml "$ROOT"/.github/workflows/capture*.yaml; do
  [ -f "$one" ] || continue
  CAPTURE_WORKFLOWS="$CAPTURE_WORKFLOWS $one"
done
CAPONE=""
for one in $CAPTURE_WORKFLOWS; do CAPONE="$one"; done

if unit capture-fork; then
  if [ -z "$CAPTURE_WORKFLOWS" ]; then
    fail "workflow  there is no capture workflow to check"
  fi

  for CAPWF in $CAPTURE_WORKFLOWS; do
    CAPNAME=$(basename "$CAPWF")
    # ⚠ The runner a capture step must call differs per workflow, and it is
    # derived from the file's own name rather than assumed: `capture.yml` runs
    # `capture-run` and `capture-client.yml` runs `capture-client`. A single
    # needle would pass on both by matching the shorter one inside the longer.
    case "$CAPNAME" in
      capture-client.*) CAPRUNNER=capture-client ;;
      *) CAPRUNNER=capture-run ;;
    esac
    # ⚠ Every job key the file declares, so a workflow that grew a platform is
    # checked on it. `job_steps` is keyed on these.
    CAPJOBS=$(awk '
    function indent(s,   i) { i = match(s, /[^ ]/); return i ? i - 1 : -1 }
    /^jobs:[ \t]*$/ { injobs = 1; next }
    !injobs { next }
    {
      line = $0; sub(/\r$/, "", line); ind = indent(line)
      if (ind < 0) next
      if (ind == 0) { injobs = 0; next }
      if (ind == 2 && line ~ /^ *[A-Za-z_][A-Za-z0-9_-]*:[ \t]*$/) {
        k = line; sub(/^ +/, "", k); sub(/:.*$/, "", k); print k
      }
    }' "$CAPWF")

    CAPTRIG=$(triggers "$CAPWF" | tr '\n' ' ' | sed 's/ *$//')
    if [ "$CAPTRIG" = "workflow_dispatch" ]; then
      pass "workflow  $CAPNAME can only be dispatched by hand"
    else
      fail "workflow  $CAPNAME has other triggers: $CAPTRIG"
    fi

    # ⛔ THE STEP ORDER IS THE CONTAINMENT, AND NOTHING ELSE ENFORCES IT. Building
    # after the route is deleted cannot work: there is no network left to fetch a
    # crate from, and the only way to repair it under containment is to restore
    # the route, on the host that exists to have none. Capturing before the guard
    # would measure a host nobody established was contained. Uploading before the
    # restore is a step that cannot reach GitHub. ⚠ Each of those reads as a
    # plausible ordering in a diff, and none of them is visible to any other check
    # in this repository.
    order_case() { # job earlier later
      _a=$(step_index "$CAPWF" "$1" "$2")
      _b=$(step_index "$CAPWF" "$1" "$3")
      if [ -z "$_a" ]; then
        fail "workflow  $CAPNAME $1 has no step named [$2]"
      elif [ -z "$_b" ]; then
        fail "workflow  $CAPNAME $1 has no step named [$3]"
      elif [ "$_a" -lt "$_b" ]; then
        pass "workflow  $CAPNAME $1: [$2] precedes [$3]"
      else
        fail "workflow  $CAPNAME $1: [$2] comes AFTER [$3]"
      fi
    }

    for job in $CAPJOBS; do
      # ⛔ TWO SHAPES OF CONTAINMENT, AND A JOB IS HELD TO THE ONE IT USES. A job
      # that cuts the host's route must build before the cut and upload after the
      # restore. A job that contains the capture in a network namespace never
      # cuts anything, so what must hold is that the guard and the capture both
      # RUN inside it - a capture step that dropped `contained.sh` would measure a
      # host with a route off it, and every ordering rule would still pass.
      # ⚠ `capture-client.yml` is the second shape since 2026-10-11, because the
      # first one severed the runner agent from GitHub: `TODO/ci.md`, `CI-08`.
      order_case "$job" "Claim the host" "Build the observer"
      if [ -n "$(step_index "$CAPWF" "$job" "Cut the route off this host")" ]; then
        CAPCUT=1
        order_case "$job" "Build the observer" "Cut the route off this host"
        order_case "$job" "Cut the route off this host" "Assert containment"
        order_case "$job" "Assert containment" "Capture"
        order_case "$job" "Capture" "Restore the route"
        order_case "$job" "Restore the route" "Upload the evidence bundle"
      else
        CAPCUT=0
        order_case "$job" "Build the observer" "Assert containment"
        order_case "$job" "Assert containment" "Capture"
        order_case "$job" "Capture" "Upload the evidence bundle"
        for _contained in "Assert containment" Capture; do
          case "$(step_command "$CAPWF" "$job" "$_contained")" in
            *"scripts/capture/contained.sh -- "*) pass "workflow  $CAPNAME $job: [$_contained] runs in its own network namespace" ;;
            *) fail "workflow  $CAPNAME $job: [$_contained] cuts no route and does not run through contained.sh" ;;
          esac
        done
      fi

      # ⛔ AN INSTALL IS TWO MORE ORDERING CONSTRAINTS AND BOTH ARE NEW. It must
      # come AFTER the claim, because a product installed on a host nothing
      # established was disposable is state on a machine that may be kept; and it
      # must come BEFORE the route is cut, because a package index is not
      # reachable from a host with no default route and capture-client refuses to
      # install anything itself. ⚠ A workflow with no such step is not failed
      # here: capture.yml installs nothing and its own Prove says so.
      if [ -n "$(step_index "$CAPWF" "$job" "Install the client")" ]; then
        order_case "$job" "Claim the host" "Install the client"
        if [ "$CAPCUT" = 1 ]; then
          order_case "$job" "Install the client" "Cut the route off this host"
        else
          order_case "$job" "Install the client" "Assert containment"
        fi
      fi

      # ⛔ AND CHOOSING THE ARTIFACT IS A THIRD. A `release` route refuses without
      # a URL, so a resolve step after the install would leave the route with
      # nothing to fetch. ⚠ Only this one constraint is asserted: resolve-before-cut
      # follows from it and the install rule above, and a rule that cannot fail
      # independently of another is a rule nobody can ever see refuse.
      if [ -n "$(step_index "$CAPWF" "$job" "Resolve the release artifact")" ]; then
        order_case "$job" "Resolve the release artifact" "Install the client"
      fi

      # ⛔ AND EVERY UPLOAD IS AFTER THE RESTORE, NOT ONLY THE EVIDENCE ONE.
      # Between *Cut the route* and *Restore the route* this host has no way off
      # itself, so an upload placed there cannot reach GitHub at all. ⚠ The install
      # logs sat BEFORE the cut for four dispatches, which worked because that is
      # also before the cut - and the rule a reader would infer from it, that the
      # step may go anywhere, is the one that breaks the day it moves.
      if [ "$CAPCUT" = 1 ] && [ -n "$(step_index "$CAPWF" "$job" "Upload the install logs")" ]; then
        order_case "$job" "Restore the route" "Upload the install logs"
      fi

      # ⚠ And the capture step must call the runner this workflow is for. Every
      # rule above holds over a workflow whose Capture step runs `true`.
      case "$(step_command "$CAPWF" "$job" Capture)" in
        *"$CAPRUNNER"*) pass "workflow  $CAPNAME $job runs $CAPRUNNER" ;;
        *) fail "workflow  $CAPNAME $job: the Capture step does not run $CAPRUNNER" ;;
      esac

      # ⛔ THE EVIDENCE UPLOAD IS FATAL AND STAYS FATAL. A capture that measured a
      # build and uploaded nothing must be red; `if-no-files-found: error` says so
      # and `continue-on-error: true` would silently undo it, leaving a green run
      # with no evidence anywhere - the worst outcome this workflow has, and one a
      # reader would have to notice two keys apart to see.
      #
      # ⚠ THE ASYMMETRY IS THE RULE. The install-logs upload IS non-fatal on
      # purpose: it is a diagnostic aid that already declares `warn`, and bounding
      # it is what stops a hang in that step throwing away a finished capture.
      # Enforcing "no step is continue-on-error" would refuse that, so the rule
      # names the step whose failure means there is no measurement.
      _bundle=$(step_block "$CAPWF" "$job" "Upload the evidence bundle")
      if [ -z "$_bundle" ]; then
        fail "workflow  $CAPNAME $job has no step named [Upload the evidence bundle]"
      else
        if printf '%s\n' "$_bundle" | grep -q 'if-no-files-found:[ ]*error'; then
          pass "workflow  $CAPNAME $job: the evidence upload is red when it finds nothing"
        else
          fail "workflow  $CAPNAME $job: the evidence upload does not declare if-no-files-found: error"
        fi
        if printf '%s\n' "$_bundle" | grep -q 'continue-on-error'; then
          fail "workflow  $CAPNAME $job: the evidence upload is continue-on-error, so a run with no evidence would be green"
        else
          pass "workflow  $CAPNAME $job: the evidence upload cannot be skipped over"
        fi
        # ⛔ AND AN UPLOAD THAT RUNS ON A REFUSED CAPTURE IS GATED ON RULE 12.
        # A refused capture is evidence and is uploaded since 2026-10-14; the
        # runner's own scan is on its success path, so a second asking has to
        # stand between a refusal and the upload.
        _upload_if=$(step_if "$CAPWF" "$job" "Upload the evidence bundle")
        if [ -z "$_upload_if" ]; then
          pass "workflow  $CAPNAME $job: the evidence upload runs on success alone"
        else
          _gate=$(scan_gate "$CAPWF" "$job" "$_upload_if")
          if [ -z "$_gate" ]; then
            pass "workflow  $CAPNAME $job: a refused capture is uploaded only after the rule-12 scan passes"
          else
            fail "workflow  $CAPNAME $job: $_gate"
          fi
        fi
      fi

      # ⛔ AND A JOB THAT STARTS THE FLIGHT RECORDER ENDS BY STOPPING IT. An
      # open check run reads as a host that lost contact, so one left open by a
      # job that ended cleanly is the recorder lying about the host.
      if [ -n "$(step_index "$CAPWF" "$job" "Start the flight recorder")" ]; then
        _stopped=$(recorder_stopped "$CAPWF" "$job")
        if [ -z "$_stopped" ]; then
          pass "workflow  $CAPNAME $job: the flight recorder is stopped by the job's last step"
        else
          fail "workflow  $CAPNAME $job: $_stopped"
        fi
      fi
    done
  done
fi

# -- 0d. the static readers, refuted ------------------------------------------
#
# ⛔ EVERY RULE ABOVE IS A READER THAT HAS NEVER BEEN SEEN TO REFUSE ANYTHING.
# Each is checked against a copy of the workflow with the property removed,
# because a reader that answers "present" over a file that does not have it
# would pass all four cases above on any workflow at all.
if unit static-readers; then
  MUTWF="$WORK/workflow-without.yml"

  sed 's/ --strict//' "$WORKFLOW" >"$MUTWF"
  if gate_asks_strictly "$MUTWF" linux '--strict'; then
    fail "probe    the strict-flag reader passed a workflow with no --strict"
  else
    pass "probe    the strict-flag reader refuses a workflow with no --strict"
  fi

  awk '!/^    permissions:$/ || n++' "$WORKFLOW" >"$MUTWF"
  if declares_key "$MUTWF" permissions; then
    fail "probe    the job-key reader passed a job with no permissions"
  else
    pass "probe    the job-key reader refuses a job with no permissions"
  fi

  awk '!/^    timeout-minutes:/ || n++' "$WORKFLOW" >"$MUTWF"
  if declares_key "$MUTWF" timeout-minutes; then
    fail "probe    the job-key reader passed a job with no timeout"
  else
    pass "probe    the job-key reader refuses a job with no timeout"
  fi

  # ⚠ The trigger reader is checked against a workflow that HAS an automatic
  # trigger rather than against one with none, because the direction that matters
  # is the one where the publisher gains a way to fire by itself.
  if [ -f "$PUBWF" ]; then
    awk '/^on:$/ { print; print "  push:"; next } { print }' "$PUBWF" >"$MUTWF"
    case "$(triggers "$MUTWF" | tr '\n' ' ')" in
      *push*) pass "probe    the trigger reader sees an added push trigger" ;;
      *) fail "probe    the trigger reader missed an added push trigger" ;;
    esac

    sed 's/        default: true/        default: false/' "$PUBWF" >"$MUTWF"
    if [ "$(input_default "$MUTWF" dry_run)" = "true" ]; then
      fail "probe    the input-default reader still reports true after the flip"
    else
      pass "probe    the input-default reader follows the declared default"
    fi
  fi

  # ⚠ The script-name reader is checked against a workflow naming a script that is
  # NOT in the tree, because a reader that reported every path present would pass
  # the case above on any workflow at all. That is the exact failure the deleted
  # draft of the capture workflow would have needed catching.
  sed 's|scripts/capture/capture-run.sh|scripts/capture/no-such-script.sh|' "$WORKFLOW" >"$MUTWF"
  printf '        run: sh scripts/capture/no-such-script.sh\n' >>"$MUTWF"
  MISSING=""
  for s in $(scripts_named "$MUTWF"); do
    [ -f "$ROOT/$s" ] || MISSING="$MISSING $s"
  done
  if [ -n "$MISSING" ]; then
    pass "probe    the script-name reader sees a script that is not in the tree"
  else
    fail "probe    the script-name reader missed a script that is not in the tree"
  fi

  # ⚠ AND THE ORDER READER IS REFUTED BY SWAPPING TWO STEP NAMES rather than by
  # reading a workflow that is already correct. A reader that answered "in order"
  # over any input would pass all twelve ordering cases above.
  if [ -f "$CAPONE" ]; then
    sed -e 's/- name: Build the observer/- name: BIT-IDS-PROBE-SWAP/' \
      -e 's/- name: Cut the route off this host/- name: Build the observer/' \
      -e 's/- name: BIT-IDS-PROBE-SWAP/- name: Cut the route off this host/' \
      "$CAPONE" >"$MUTWF"
    if cmp -s "$CAPONE" "$MUTWF"; then
      fail "probe    the ordering mutation did not change the workflow"
    else
      _a=$(step_index "$MUTWF" linux "Build the observer")
      _b=$(step_index "$MUTWF" linux "Cut the route off this host")
      if [ -n "$_a" ] && [ -n "$_b" ] && [ "$_a" -gt "$_b" ]; then
        pass "probe    the order reader sees a build moved after the route is cut"
      else
        fail "probe    the order reader missed a build moved after the route is cut"
      fi
    fi

    # ⚠ And a workflow that gained a pull_request trigger, which is the direction
    # that matters: the fork guard is the absence, so the reader has to be able to
    # see the absence end.
    awk '/^on:$/ { print; print "  pull_request:"; next } { print }' "$CAPONE" >"$MUTWF"
    case "$(triggers "$MUTWF" | tr '\n' ' ')" in
      *pull_request*) pass "probe    the trigger reader sees an added pull_request trigger" ;;
      *) fail "probe    the trigger reader missed an added pull_request trigger" ;;
    esac
  fi

  # ⛔ THE TWO INSTALL-ORDER RULES ARE REFUTED SEPARATELY, BECAUSE THEY ARE TWO
  # RULES. An install after the route is cut cannot reach a package index; an
  # install before the claim puts a stranger's binary on a host nothing
  # established was disposable. A single probe would leave whichever one it did
  # not move as a reader nobody has seen refuse anything.
  #
  # ⚠ NAMED EXPLICITLY RATHER THAN INHERITED FROM THE LOOP ABOVE. `$CAPWF` after a
  # `for` holds whatever the glob happened to end on, which is a probe whose
  # subject depends on a file name.
  CLIENTWF="$ROOT/.github/workflows/capture-client.yml"
  if [ -f "$CLIENTWF" ]; then
    swap_steps() { # file a b out
      sed -e "s/- name: $1/- name: BIT-IDS-PROBE-SWAP/" \
        -e "s/- name: $2/- name: $1/" \
        -e "s/- name: BIT-IDS-PROBE-SWAP/- name: $2/" \
        "$CLIENTWF" >"$3"
    }

    swap_steps "Install the client" "Assert containment" "$MUTWF"
    _a=$(step_index "$MUTWF" linux "Install the client")
    _b=$(step_index "$MUTWF" linux "Assert containment")
    if [ -n "$_a" ] && [ -n "$_b" ] && [ "$_a" -gt "$_b" ]; then
      pass "probe    the order reader sees an install moved after containment"
    else
      fail "probe    the order reader missed an install moved after containment"
    fi

    # ⛔ AND THE NAMESPACE RULE IS REFUTED BY TAKING THE WRAPPER AWAY, which is
    # the one edit that leaves every ordering rule above green.
    sed 's|sh scripts/capture/contained.sh -- sh scripts/capture/capture-client.sh|sh scripts/capture/capture-client.sh|' \
      "$CLIENTWF" >"$MUTWF"
    if cmp -s "$CLIENTWF" "$MUTWF"; then
      fail "probe    the namespace mutation did not change the workflow"
    else
      case "$(step_command "$MUTWF" linux Capture)" in
        *"scripts/capture/contained.sh -- "*) fail "probe    the namespace reader still sees contained.sh after it was removed" ;;
        *) pass "probe    the namespace reader sees a capture step that left its namespace" ;;
      esac
    fi

    swap_steps "Claim the host" "Install the client" "$MUTWF"
    _a=$(step_index "$MUTWF" linux "Claim the host")
    _b=$(step_index "$MUTWF" linux "Install the client")
    if [ -n "$_a" ] && [ -n "$_b" ] && [ "$_a" -gt "$_b" ]; then
      pass "probe    the order reader sees an install moved before the claim"
    else
      fail "probe    the order reader missed an install moved before the claim"
    fi

    # ⚠ AND THE RESOLVE RULE IS REFUTED ON ITS OWN, because it is conditional on a
    # step the sibling workflow does not have: a probe that only moved the install
    # would leave it a reader nobody has seen refuse anything.
    swap_steps "Resolve the release artifact" "Install the client" "$MUTWF"
    _a=$(step_index "$MUTWF" linux "Resolve the release artifact")
    _b=$(step_index "$MUTWF" linux "Install the client")
    if [ -n "$_a" ] && [ -n "$_b" ] && [ "$_a" -gt "$_b" ]; then
      pass "probe    the order reader sees a resolve moved after the install"
    else
      fail "probe    the order reader missed a resolve moved after the install"
    fi

    # ⛔ AND THE RULE MUST BE SILENT ON A WORKFLOW WITH NO SUCH STEP, for the same
    # reason the install rule is: `capture.yml` resolves nothing.
    if [ -n "$(step_index "$ROOT/.github/workflows/capture.yml" linux "Resolve the release artifact")" ]; then
      fail "probe    the resolve-order rule found a resolve step in the fixture workflow"
    else
      pass "probe    the resolve-order rule is silent on a workflow that resolves nothing"
    fi

    if [ -n "$(step_index "$ROOT/.github/workflows/capture.yml" linux "Upload the install logs")" ]; then
      fail "probe    the upload-order rule found an install-logs step in the fixture workflow"
    else
      pass "probe    the upload-order rule is silent on a workflow that uploads no install logs"
    fi

    # ⛔ AND THE RULE MUST NOT FIRE ON A WORKFLOW THAT INSTALLS NOTHING. It is
    # conditional on the step existing, and a condition that was always true would
    # have failed capture.yml, whose own Prove says it installs nothing.
    if [ -n "$(step_index "$ROOT/.github/workflows/capture.yml" linux "Install the client")" ]; then
      fail "probe    the install-order rule found an install step in the fixture workflow"
    else
      pass "probe    the install-order rule is silent on a workflow that installs nothing"
    fi

    # ⛔ THE STEP READER IS REFUTED BEFORE THE TWO RULES BUILT ON IT. A reader that
    # returned the whole job would satisfy both of them from a neighbouring step's
    # keys: the install-logs upload immediately after carries exactly the
    # `continue-on-error` the evidence rule forbids, so an over-wide block would
    # report the evidence upload as non-fatal and the rule would look like it
    # works. ⚠ That is the scope defect this repository has shipped twice.
    PROBE_BLOCK="$WORK/step-block"
    step_block "$CLIENTWF" linux "Upload the evidence bundle" >"$PROBE_BLOCK"
    if grep -q -F -e 'Upload the evidence bundle' "$PROBE_BLOCK" &&
      ! grep -q -F -e 'Upload the install logs' "$PROBE_BLOCK"; then
      pass "probe    the step reader returns one step and not its neighbours"
    else
      fail "probe    the step reader returned $(grep -c 'name:' "$PROBE_BLOCK") name line(s)"
    fi

    # ⚠ And each rule is refuted by planting what it forbids, separately, because
    # they are two rules over one step and a single plant would leave whichever it
    # did not move a reader nobody has seen refuse anything.
    sed 's/if-no-files-found: error/if-no-files-found: warn/' "$CLIENTWF" >"$MUTWF"
    step_block "$MUTWF" linux "Upload the evidence bundle" >"$PROBE_BLOCK"
    if cmp -s "$CLIENTWF" "$MUTWF"; then
      fail "probe    the if-no-files-found mutation did not change the workflow"
    elif grep -q 'if-no-files-found:[ ]*error' "$PROBE_BLOCK"; then
      fail "probe    the evidence-upload reader still sees error after the flip"
    else
      pass "probe    the evidence-upload reader sees if-no-files-found weakened"
    fi

    awk '/- name: Upload the evidence bundle/ { print; print "        continue-on-error: true"; next } { print }' \
      "$CLIENTWF" >"$MUTWF"
    step_block "$MUTWF" linux "Upload the evidence bundle" >"$PROBE_BLOCK"
    if cmp -s "$CLIENTWF" "$MUTWF"; then
      fail "probe    the continue-on-error mutation did not change the workflow"
    elif grep -q 'continue-on-error' "$PROBE_BLOCK"; then
      pass "probe    the evidence-upload reader sees an added continue-on-error"
    else
      fail "probe    the evidence-upload reader missed an added continue-on-error"
    fi

    # ⛔ THE RULE-12 GATE, REFUTED BOTH WAYS IT CAN BE LOST. An upload that runs
    # on any outcome with no scan named is the shortcut; a scan step that runs
    # something else keeps the shape and drops the reading. ⚠ And the control is
    # the workflow as it stands, because a gate reader answering "ungated" over
    # everything would pass both plants.
    _probe_if=$(step_if "$CLIENTWF" linux "Upload the evidence bundle")
    if [ -n "$_probe_if" ] && [ -z "$(scan_gate "$CLIENTWF" linux "$_probe_if")" ]; then
      pass "probe    the rule-12 gate reader accepts the upload as it stands"
    else
      fail "probe    the rule-12 gate reader refuses the upload as it stands: $(scan_gate "$CLIENTWF" linux "$_probe_if")"
    fi
    awk '/- name: Upload the evidence bundle/ { print; getline; if ($1 == "if:") print "        if: always()"; else print; next } { print }' \
      "$CLIENTWF" >"$MUTWF"
    _probe_if=$(step_if "$MUTWF" linux "Upload the evidence bundle")
    if [ "$_probe_if" != "always()" ]; then
      fail "probe    the always() mutation did not reach the upload's if (read [$_probe_if])"
    elif [ -n "$(scan_gate "$MUTWF" linux "$_probe_if")" ]; then
      pass "probe    the rule-12 gate reader refuses an upload on always() with no scan"
    else
      fail "probe    the rule-12 gate reader accepted an upload on always() with no scan"
    fi
    # shellcheck disable=SC2016
    sed 's|run: sh scripts/capture/scan-bundle.sh "$RUNNER_TEMP/capture"|run: "true"|' \
      "$CLIENTWF" >"$MUTWF"
    _probe_if=$(step_if "$MUTWF" linux "Upload the evidence bundle")
    if cmp -s "$CLIENTWF" "$MUTWF"; then
      fail "probe    the scan-command mutation did not change the workflow"
    elif [ -n "$(scan_gate "$MUTWF" linux "$_probe_if")" ]; then
      pass "probe    the rule-12 gate reader refuses a scan step that scans nothing"
    else
      fail "probe    the rule-12 gate reader accepted a scan step that scans nothing"
    fi

    # ⛔ THE RECORDER-STOP READER, REFUTED BY THE TWO WAYS A STOP IS LOST: the
    # step deleted, and the step kept while it no longer runs on a failure. The
    # workflow as it stands is the control, for the reason the gate's is.
    if [ -z "$(recorder_stopped "$CLIENTWF" linux)" ]; then
      pass "probe    the recorder-stop reader accepts the workflow as it stands"
    else
      fail "probe    the recorder-stop reader refuses the workflow as it stands: $(recorder_stopped "$CLIENTWF" linux)"
    fi
    awk '/- name: Stop the flight recorder/ { skip = 1; next }
         skip && /^      - name:/ { skip = 0 }
         !skip { print }' "$CLIENTWF" >"$MUTWF"
    if cmp -s "$CLIENTWF" "$MUTWF"; then
      fail "probe    the stop-step removal did not change the workflow"
    elif [ -n "$(recorder_stopped "$MUTWF" linux)" ]; then
      pass "probe    the recorder-stop reader refuses a job that never stops the recorder"
    else
      fail "probe    the recorder-stop reader accepted a job that never stops the recorder"
    fi
    awk '/- name: Stop the flight recorder/ { print; getline; if ($1 == "if:") print "        if: success()"; else print; next } { print }' \
      "$CLIENTWF" >"$MUTWF"
    if cmp -s "$CLIENTWF" "$MUTWF"; then
      fail "probe    the stop-step condition mutation did not change the workflow"
    elif [ -n "$(recorder_stopped "$MUTWF" linux)" ]; then
      pass "probe    the recorder-stop reader refuses a stop that skips a failed job"
    else
      fail "probe    the recorder-stop reader accepted a stop that skips a failed job"
    fi
  fi

# -- the controls -------------------------------------------------------------
#
# ⚠ Run first and in this order, because the first cargo command warms the
# shared target directory that every later case reuses.
fi
if always clean-tree-controls; then
  control linux "Rust check" "cargo check"
  control linux "Rust formatting" "cargo fmt"
  control linux "Rust tests" "cargo test"
  control linux "Rust lints" "cargo clippy"
  control linux "Shell syntax and style" "shellcheck and shfmt"
  control linux "Go formatting and vet" "gofmt and go vet"
  gate_control "the repository gate"
  # ⚠ THE COSTLIEST CONTROL HERE AFTER THE GATE, about a minute, because it
  # builds every harness's Rust subject cold. It is paid in every shard for the
  # reason the block above gives: the plant below proves nothing without it.
  control linux "Checking harnesses" "the checking harnesses"
fi

# -- 1. a Rust defect that does not compile -----------------------------------
# ⛔ HOISTED FOR THE SAME REASON: `rust-clippy` and `rust-fmt` plant into this
# same file, and each is its own shard unit.
LIB="crates/bit-ids/src/lib.rs"

if unit rust-nocompile; then
  if keep "$LIB" && append_once "$TREE/$LIB" "fn workflow_plant() -> u8 { \"not a u8\" }"; then
    refuses linux "Rust check" "an uncompilable Rust defect" "error"
  else
    fail "plant    could not plant the Rust compile defect"
  fi
  restore "$LIB" || exit 2
fi

# -- 2. a Rust defect clippy refuses ------------------------------------------
#
# ⚠ It compiles and it is correctly formatted on purpose. A plant that also
# broke the build or the formatting would be refused by an earlier step, and the
# case would report the lint step working when it had never been reached.
if unit rust-clippy; then
  if keep "$LIB" && append_once "$TREE/$LIB" "pub fn workflow_plant() -> u8 { return 1; }"; then
    refuses linux "Rust lints" "a lint the workspace denies" "needless_return"
  else
    fail "plant    could not plant the clippy defect"
  fi
  restore "$LIB" || exit 2
fi

# -- 3. a Rust file that is not formatted -------------------------------------
if unit rust-fmt; then
  if keep "$LIB" && append_once "$TREE/$LIB" "pub  fn   workflow_plant ( )  ->  u8  {  1  }"; then
    refuses linux "Rust formatting" "an unformatted Rust file" "Diff in"
  else
    fail "plant    could not plant the formatting defect"
  fi
  restore "$LIB" || exit 2
fi

# -- 4. a schema fixture whose identity no longer derives ----------------------
#
# ⭐ The platform is in the tuple the record identifier digests, so changing it
# makes the declared id disagree with the record's own contents. That is the
# rule docs/architecture.md section 4 states, planted rather than read.
if unit schema-fixture; then
  PROFILE="crates/bit-ids/tests/fixtures/valid-profile.json"
  if keep "$PROFILE" &&
    replace_once "$TREE/$PROFILE" '"platform": "linux"' '"platform": "windows"'; then
    refuses linux "Rust tests" "a schema fixture that no longer validates" "FAILED"
  else
    fail "plant    could not plant the schema defect"
  fi
  restore "$PROFILE" || exit 2
fi

# -- 5. a wire fixture whose bytes moved --------------------------------------
#
# ⚠ Not a byte of the frame. index.json digests each fixture FILE, so any change
# to one is what the corpus digest is there to catch, and the summary is a field
# no decoder reads: a plant in the frame bytes could be refused by a codec
# instead, which is a different guard than the one this case names.
if unit wire-fixture; then
  FIXTURE="crates/bit-ids-wire/tests/fixtures/tracker-http-announce-started.json"
  if keep "$FIXTURE" &&
    replace_once "$TREE/$FIXTURE" 'An ordinary started announce' 'An ordinary drifted announce'; then
    refuses linux "Rust tests" "a wire fixture whose digest moved" "FAILED"
  else
    fail "plant    could not plant the fixture defect"
  fi
  restore "$FIXTURE" || exit 2
fi

# -- 6. a shell script that does not parse ------------------------------------
# ⚠ The pattern is shellcheck's own parse-error code rather than nothing at all.
# This case accepted any non-zero status at first, which `shfmt` would also have
# produced for a reformatting difference, so the case would have passed while
# the syntax checker said nothing.
if unit shell-parse; then
  SCRIPT="scripts/doctor/doctor.sh"
  if keep "$SCRIPT" && append_once "$TREE/$SCRIPT" "if [ 1 = 1 ; then :"; then
    refuses linux "Shell syntax and style" "a shell script that does not parse" "SC1073"
  else
    fail "plant    could not plant the shell defect"
  fi
  restore "$SCRIPT" || exit 2
fi

# -- 6b. a shell script that parses and fails only shellcheck ----------------
# ⛔ THE STEP RUNS TWO CHECKS, AND EACH MUST FAIL IT ALONE. Until 2026-10-14 the
# body ran `shellcheck` and `shfmt` on two bare lines, so under this harness's
# `sh -c` a finding only shellcheck makes was answered by shfmt's exit 0. ⚠ The
# case above passes either way, because shfmt also rejects a script that does
# not parse. SC2086 is a finding shfmt does not make: the planted line is
# already formatted as shfmt would write it.
if unit shell-lint-only; then
  SCRIPT="scripts/doctor/doctor.sh"
  # shellcheck disable=SC2016
  if keep "$SCRIPT" && append_once "$TREE/$SCRIPT" 'echo $BIT_IDS_UNQUOTED_PLANT'; then
    refuses linux "Shell syntax and style" "a shell script only shellcheck refuses" "SC2086"
  else
    fail "plant    could not plant the shellcheck-only defect"
  fi
  restore "$SCRIPT" || exit 2
fi

# -- 7. a document whose link resolves to nothing -----------------------------
#
# ⚠ The pattern is the runner's FAILURE row and not the check's name. The name
# alone also matches the row that reads `ok    check-docs`, so a plant that was
# never refused would have matched the control's own output.
if unit docs-deadlink; then
  README="README.md"
  if keep "$README" &&
    append_once "$TREE/$README" "A link to [nothing](docs/this-file-does-not-exist.md)."; then
    gate_refuses "a document with a dead link" "FAIL  check-docs"
  else
    fail "plant    could not plant the docs defect"
  fi
  restore "$README" || exit 2
fi

# -- 8. a check that stopped running ------------------------------------------
#
# ⛔ THIS IS THE ONE THE LANE COULD NOT SEE. A check that answers 2 is counted as
# a skip, and the workflow's gate step ran without --strict on the Windows lane
# because six of its rows are genuinely unavailable there. Measured on
# 2026-09-06: with this same plant, that invocation exited 0. The declared and
# observed counts are separate now, so --strict permits the six and refuses
# this. ⚠ Case 0 above is the other half: this shows the runner reports the
# skip, and that one shows CI is asking with the flag that makes it fatal.
#
# ⚠ THE SUBJECT MOVED ON 2026-09-17 AND THE CASE DID NOT CHANGE. It planted
# into `check-project.sh`, which is now one check of a Go binary rather than a
# script; `check-shell.sh` is a queued `sh` row of the same shape. ⛔ It has to be
# an `sh` ROW rather than a ported one: planting `exit 2` into a Go source file
# would fail the BUILD, and a gate whose binary did not build skips every ported
# row at once - a different plant with a different answer.
if unit check-stopped; then
  BROKEN="scripts/common/check-shell.sh"
  if keep "$BROKEN" && printf 'exit 2\n' >"$TREE/$BROKEN"; then
    gate_refuses "a check that can no longer run" "SKIP  check-shell"
  else
    fail "plant    could not plant the unrunnable check"
  fi
  restore "$BROKEN" || exit 2
fi

# -- 8b. the three rules over a workflow's own pwsh blocks --------------------
#
# ⛔ THE RULES THE FIRST DISPATCH BOUGHT. check-project's three PowerShell rules
# iterated `git ls-files '*.ps1'` and so never reached a `pwsh` block inside a
# workflow, which is the same language with the same hazards; and the third rule
# below exists only here, because only here does a harness read whatever
# $LASTEXITCODE the block left behind.
#
# ⚠ EACH PLANT GOES INTO THE STEP THAT ACTUALLY BROKE. Capture run 1's Windows
# job restored the route, the inverted egress guard refused as designed, and the
# step failed on that refusal's exit code - so the evidence upload was skipped
# over a guard doing its job. These are the plants that would have caught it.
# ⭐ EACH PLANT IS A WHOLE WORKFLOW RATHER THAN AN EDIT TO capture.yml, for two
# reasons that are not style. `replace_once` refuses a multi-line literal
# outright - that is CORPUS-01's grep -F finding made structural - and every
# needle here is a line whose neighbours are what make it a defect. And a
# purpose-built file exercises the rules' SCOPE: they read every workflow, not
# the one this harness happens to know about.
#
# ⚠ THE PLANT IS UNTRACKED IN THE SCRATCH REPOSITORY, which is deliberate. An
# uncommitted new workflow is part of the tree the next push carries, so the
# rules enumerate `git ls-files --others --exclude-standard` beside the tracked
# set; a rule that read only the tracked set would pass over exactly this file.
#
# ⚠ NO `uses:` LINE ANYWHERE IN THESE. The action-pin rule globs the workflow
# directory rather than asking git, so an unpinned plant would turn the gate red
# for a reason that is not the rule under test - and the accepted cases below
# would break outright.
PLANT="$TREE/.github/workflows/zz-plant.yml"
plant_head() { # step-name
  cat <<YAML
name: Plant
on:
  workflow_dispatch:
permissions:
  contents: read
jobs:
  plant:
    runs-on: ubuntu-24.04
    timeout-minutes: 5
    permissions:
      contents: read
    steps:
      - name: $1
YAML
}

# 8b-i. a pwsh block that stops on errors and says nothing about a native exit.
if unit pwsh-native-preference; then
  {
    plant_head "Stop with no native preference"
    cat <<'YAML'
        shell: pwsh
        run: |
          $ErrorActionPreference = 'Stop'
          Write-Output 'nothing here says what a native exit code means'
YAML
  } >"$PLANT"
  gate_refuses "a workflow pwsh block that stops without saying what a native exit means" \
    "FAIL  check-project"
  rm -f "$PLANT"
fi

# 8b-ii. a pwsh block reporting through a display layer.
if unit pwsh-write-error; then
  {
    plant_head "Report through Write-Error"
    cat <<'YAML'
        shell: pwsh
        run: |
          Write-Error 'a refusal a harness would match on, wrapped by host width'
YAML
  } >"$PLANT"
  gate_refuses "a workflow pwsh block reporting through Write-Error" "FAIL  check-project"
  rm -f "$PLANT"
fi

# ⛔ 8b-iii. THE DEFECT VERBATIM, and the only case here that reproduces a
# measured failure rather than constructing one. This is capture.yml's Windows
# *Restore the route* block as it stood on capture run 1, minus nothing: the
# logic is correct, the guard refuses exactly as designed, and the block ends on
# that refusal's exit code. GitHub's pwsh wrapper reads it as the step's verdict,
# so a restored route failed the step and the evidence upload was skipped.
# ⚠ No reading caught it. A dispatch did.
if unit pwsh-lastexitcode; then
  {
    plant_head "Leave the guard's refusal in LASTEXITCODE"
    cat <<'YAML'
        shell: pwsh
        run: |
          $ErrorActionPreference = 'Stop'
          $PSNativeCommandUseErrorActionPreference = $false
          ./scripts/acquisition/assert-disposable.ps1 -Egress
          if ($LASTEXITCODE -eq 0) {
            [Console]::Error.WriteLine('the default routes were not restored')
            exit 1
          }
YAML
  } >"$PLANT"
  gate_refuses "a workflow pwsh block leaving \$LASTEXITCODE as the step's verdict" \
    "FAIL  check-project"
  rm -f "$PLANT"
fi

# ⛔ 8b-iv AND 8b-v ARE THE CASES THAT WOULD MAKE THESE RULES FIRE ON CORRECT
# USAGE, and a rule that refused either is a rule somebody switches off. The
# first has every needle in comment position and ends its status deliberately;
# the second has them in command position in a step whose shell is not pwsh at
# all, which is the scope half rather than the comment half.
if unit pwsh-needles-in-comments; then
  {
    plant_head "The needles in comments, and an exit that is a decision"
    cat <<'YAML'
        shell: pwsh
        run: |
          $ErrorActionPreference = 'Stop'
          $PSNativeCommandUseErrorActionPreference = $false
          # Write-Error is named here as prose, and so is $LASTEXITCODE.
          ./scripts/acquisition/assert-disposable.ps1 -Fingerprint
          if ($LASTEXITCODE -ne 0) { exit 1 }
          exit 0
YAML
  } >"$PLANT"
  gate_control "the repository gate" "a pwsh block whose needles are comments"
  rm -f "$PLANT"
fi

if unit pwsh-non-pwsh-step; then
  {
    plant_head "The needles in a step whose shell is not pwsh"
    cat <<'YAML'
        shell: bash
        run: |
          echo 'Write-Error is a word here and $LASTEXITCODE is another'
YAML
  } >"$PLANT"
  gate_control "the repository gate" "a non-pwsh step carrying both needles"
  rm -f "$PLANT"
fi

# -- 9. a Go file that is not formatted ---------------------------------------
#
# ⚠ It compiles, so the refusal is gofmt's and not the compiler's. The pattern
# is the step's own line, which only the formatting branch prints.
GOFILE="tools/check/project.go"

if unit go-fmt; then
  if keep "$GOFILE" && append_once "$TREE/$GOFILE" "func  workflowPlant( ) {}"; then
    refuses linux "Go formatting and vet" "an unformatted Go file" "gofmt would rewrite"
  else
    fail "plant    could not plant the Go formatting defect"
  fi
  restore "$GOFILE" || exit 2
fi

# -- 10. a Go defect vet refuses ----------------------------------------------
#
# ⚠ Formatted and compiling on purpose, so the step reaches its last line: a
# plant gofmt also refused would report vet working when vet never ran.
if unit go-vet; then
  if keep "$GOFILE" && append_once "$TREE/$GOFILE" 'func workflowPlant() { fmt.Printf("%d", "x") }'; then
    refuses linux "Go formatting and vet" "a Go defect vet refuses" "wrong type"
  else
    fail "plant    could not plant the vet defect"
  fi
  restore "$GOFILE" || exit 2
fi

# -- 11. a checking rule weakened so it no longer refuses ---------------------
#
# ⛔ THE GATE PASSES THIS TREE. The rule is only ever run against a clean tree
# there, and a rule that reports success where it should report a refusal still
# answers 0 over a clean one. Only the harness cases plant the subject the rule
# exists to catch, which is why the step this refutes exists.
if unit harness-weakened; then
  CACHEGO="tools/check/cache.go"
  if keep "$CACHEGO" &&
    replace_once "$TREE/$CACHEGO" 'h.bad("E-CAC-01  the refusal did not appear: "' \
      'h.ok("E-CAC-01  the refusal did not appear: "'; then
    refuses linux "Checking harnesses" "a checking rule that no longer refuses" "a guard did not refuse its defect"
  else
    fail "plant    could not plant the weakened rule"
  fi
  restore "$CACHEGO" || exit 2
fi

# -- 12. the clean tree again -------------------------------------------------
#
# ⛔ THE LAST CASE IS A CONTROL, NOT A REPEAT. Every plant above restores the
# file it touched, and nothing so far has checked that any of them did. A
# restore that silently failed would leave every later case running against a
# defective tree, and the run would still report every guard refusing.
if always clean-tree-after-restores; then
  gate_control "the repository gate, after every restore"
fi

# ⛔ A LISTING IS NOT A RUN, so it does not go through the report. store_report
# refuses a run in which nothing passed - correctly, since that is what a broken
# harness looks like - and --units is a mode in which nothing is supposed to pass.
if [ "$UNITS_ONLY" = 1 ]; then
  exit 0
fi

# ⭐ WHAT THIS SHARD RAN AND WHAT IT DID NOT, PRINTED RATHER THAN IMPLIED. A shard
# that silently claimed no unit is the defect this whole mechanism is most likely
# to introduce, and the only thing that makes it visible is the runner saying
# which names it took. ⚠ It prints on every run, including an unsharded one, so
# the line means the same thing in both and a reader never has to know which mode
# produced the log they are holding.
if [ "$JSON" = "0" ] && [ "$SHARD_N" -gt 1 ]; then
  printf 'shard %s of %s\n' "$SHARD_I" "$SHARD_N"
  printf '  ran:     %s\n' "${UNITS_RAN# }"
  printf '  skipped: %s\n' "${UNITS_SKIPPED# }"
fi

store_report "check-workflow/1" cases "$JSON"
