#!/bin/sh
# check-step-bodies.sh - does a capture workflow's step body do what the runner
# would do with it, INCLUDING ending?
#
# ⛔ NOTHING IN THIS REPOSITORY RAN A CAPTURE STEP'S BODY, and `CI-06` and
# `CI-08` both recorded that as the gap they could not close.
# `check-workflow.sh` reads `capture.yml`'s step names, their order and the
# *Capture* step's command, and executes steps out of `ci.yml` alone - so the
# two restore blocks were proved statically by `check-project` and dynamically
# only by a dispatch. This is the harness that runs them.
#
# -- ⛔ A STEP DOES NOT END WHEN ITS COMMAND EXITS ----------------------------
#
# The runner reads a step's output through a pipe, so a step is over when the
# command has exited AND that pipe has reached end of file. A process the body
# leaves behind holding the step's stdout keeps the pipe open, and the runner
# goes on waiting for a command that finished. ⚠ Nothing about that is visible
# to an exit code, which is why every existing harness here would call such a
# body green: `check-workflow.sh` runs step bodies with their output redirected
# to a FILE, and a file has no reader to wait on.
#
# ⭐ SO THIS RECORDS TWO FACTS PER BODY AND KEEPS THEM APART: the exit status,
# and whether the output closed. A refusal and a step that will not end are
# different failures, and a harness that reported one number could not say which
# it had.
#
# ⚠ WHY IT IS WORTH THE MACHINERY, measured rather than supposed. Client capture
# runs 6 and 7 ran the SAME aria2 install command, from commits that differ only
# in where a later step sits - `git diff` over the two says so - and the step
# took **six seconds** in one and had not returned after sixteen minutes in the
# other. A command whose duration depends on which step follows it is not a
# command that is slow. `CI-08` carries the open question; this is the
# instrument.
#
# -- WHAT IT DOES NOT ESTABLISH ------------------------------------------------
#
# ⚠ This host is not a runner image, and a body that ends here is not a body
# that ends there. What it proves is the SHAPE: that a body leaving a process on
# the step's output does not end, that these bodies do not, and that the run-1
# defect reproduces under GitHub's own wrapper form.
#
# ⚠ AND ONLY THE STEPS A SESSION HOST CAN EXECUTE. The claim writes under
# /var/lib, the cut deletes a default route, and the capture needs both; the
# guards and the product are stubbed here, so what runs is the workflow's own
# block over a stub rather than a capture.
#
# Usage:
#   sh scripts/ci/check-step-bodies.sh
#   sh scripts/ci/check-step-bodies.sh --json
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
      printf 'check-step-bodies: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-step-bodies
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require awk mkfifo sha256sum tail timeout
WORK=$(store_workdir checkstepbodies) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

READER="$ROOT/scripts/ci/workflow-step.sh"
CAPTURE_WF="$ROOT/.github/workflows/capture.yml"
CLIENT_WF="$ROOT/.github/workflows/capture-client.yml"
for _need in "$READER" "$CAPTURE_WF" "$CLIENT_WF"; do
  [ -f "$_need" ] || {
    printf 'check-step-bodies: %s is missing\n' "$_need" >&2
    exit 2
  }
done

# ⚠ pwsh IS OPTIONAL AND ITS ABSENCE IS A SKIPPED CASE, NOT A PASS. Three of the
# cases below run a `shell: pwsh` block, and a host without it has verified
# nothing about them - so they are counted as failures of this harness only when
# it could have run them, and reported as could-not-run otherwise.
HAVE_PWSH=0
command -v pwsh >/dev/null 2>&1 && HAVE_PWSH=1

# ⭐ THREE SECONDS, AND THE NUMBER IS A MEASUREMENT RATHER THAN A FEELING. A
# pipe whose last writer has exited reaches end of file at once; every closing
# case here does so in under a tenth of a second, and the two that do not close
# hold it open deliberately. ⚠ A bigger bound would only make the non-closing
# cases slower, since it is not a deadline anything correct waits out.
CLOSE_SECONDS=3

# ⛔ HOW LONG A PLANTED LEAK HOLDS THE PIPE, AND IT IS A RACE THIS HARNESS LOST.
# The leaking process is spawned INSIDE the body, and the pipe is only examined
# after the body has exited plus `CLOSE_SECONDS`. So a plant only proves what it
# claims while
#
#     leak duration  >  body duration + CLOSE_SECONDS
#
# ⚠ It was `sleep 8` written as a literal at every use, which is ample for the
# two `probe` cases - their bodies are `echo` and `exit 0` - and marginal for the
# install-block case, whose body is the whole *Install the client* step: a
# `timeout` around `install-step.sh`, which runs `install-client` under `sudo`, a
# watchdog loop and a bounded holder report. ⛔ Measured on 2026-09-09: inside a
# full gate run, where twenty-nine checks run concurrently, that body ran long
# enough for the eight seconds to expire before the pipe was looked at, and the
# case reported `the output closed, so the planted hang did not happen` - a red
# row over a tree with no defect in it. Three gate runs went red that way before
# one was captured with its per-case output.
#
# ⭐ FORTY-FIVE, AND THE MARGIN IS THE POINT RATHER THAN THE NUMBER. The failure
# needs the body to reach about five seconds; this leaves roughly forty of slack,
# so the case now measures the redirection it is about instead of the host's load.
# ⚠ THE COST IS STATED: a `sleep` orphan can outlive the check by up to this many
# seconds, where the old value bounded that at eight. It exits on its own, it
# holds only a scratch pipe, and a row that is red at random is worse.
LEAK_SECONDS=45

# ⛔ AND A ZERO BOUND IS NOT A TIGHT BOUND, IT IS NO BOUND AT ALL. coreutils
# reads a duration of 0 as "no time limit", so a harness whose ceiling had been
# edited to 0 would wait for a leaking body forever and, when the leak happened
# to end by itself, report the pipe as having closed. Measured by planting it:
# the two cases that assert a hang went red over a bound that never fired.
for _bound in "$CLOSE_SECONDS" "$LEAK_SECONDS"; do
  case "$_bound" in
    '' | *[!0-9]* | 0)
      printf 'check-step-bodies: every bound must be a positive whole number of seconds\n' >&2
      exit 2
      ;;
  esac
done
# ⛔ AND THE LEAK MUST OUTLAST THE WINDOW IT IS MEASURED IN, which is the whole
# relation above written as a check rather than as a comment somebody keeps in
# step by hand. A leak shorter than the bound proves nothing: the pipe would
# close because the plant expired, and the case would read that as no defect.
if [ "$LEAK_SECONDS" -le "$CLOSE_SECONDS" ]; then
  printf 'check-step-bodies: LEAK_SECONDS must exceed CLOSE_SECONDS\n' >&2
  exit 2
fi

# -- lifting a body out of a workflow -----------------------------------------
#
# ⛔ THE READER IS THE ONE `check-workflow.sh` USES. A second parser here would
# be a second answer to what a step runs, and this harness's whole claim is that
# it executes the block the runner would.
lift() { # workflow job step outfile
  sh "$READER" --workflow "$1" --job "$2" --step "$3" >"$4"
  return $?
}

# -- running a body the way the runner does -----------------------------------
#
# BODY_RC is the status the step would report; BODY_CLOSED is `yes` when the
# step's output reached end of file within the bound and `no` when something the
# body left behind was still holding it.
BODY_RC=0
BODY_CLOSED=""
BODY_OUT=""

# ⛔ THE OUTPUT GOES THROUGH A REAL PIPE, WHICH IS THE ENTIRE POINT. Redirecting
# to a file is what every other harness here does and it is what makes this
# class invisible: a file is never waited on.
#
# ⚠ GitHub's default shell for a `run:` step on Linux is `bash -e {0}`, and a
# `shell: pwsh` step is `pwsh -command ". '<file>'"` with `$ErrorActionPreference
# = 'stop'` PREPENDED and the residual-exit epilogue APPENDED. That append is the
# entire mechanism behind capture run 1's failure, so a harness that omitted it
# would report the broken block passing.
run_body() { # bodyfile shell cwd
  BODY_OUT="$WORK/body.out"
  rm -f "$WORK/pipe" "$BODY_OUT"
  mkfifo "$WORK/pipe" || return 2

  cat <"$WORK/pipe" >"$BODY_OUT" &
  _reader=$!

  case "$2" in
    pwsh)
      {
        printf '%s\n' "\$ErrorActionPreference = 'stop'"
        cat "$1"
        # ⛔ THE DOLLAR SIGN IS THE POINT. This is PowerShell source that the
        # wrapper appends verbatim, so a shell that expanded `$LASTEXITCODE`
        # here would write the epilogue with the value blanked out and the case
        # below would pass over a wrapper that reads nothing.
        # shellcheck disable=SC2016
        printf '%s\n' 'if ((Test-Path -LiteralPath variable:/LASTEXITCODE)) { exit $LASTEXITCODE }'
      } >"$WORK/wrapped.ps1"
      # ⛔ -NoProfile, AND IT IS THE ONE PLACE THIS HARNESS DEPARTS FROM THE
      # RUNNER ON PURPOSE. GitHub's wrapper carries no such flag, so a profile is
      # host state a `shell: pwsh` step really does inherit; but a gate row that
      # loads whatever profile a contributor's machine has is a row that goes red
      # for something that is not in this repository, and every other `pwsh`
      # invocation here passes the flag for that reason. ⚠ The departure is
      # stated rather than silent: what this harness proves about a `pwsh` body
      # is proved with no profile loaded, and a defect a profile would cause is
      # outside what it can see. Decided by the operator on 2026-09-09.
      (cd "$3" && exec pwsh -NoProfile -command ". '$WORK/wrapped.ps1'") \
        >"$WORK/pipe" 2>&1 </dev/null &
      ;;
    *)
      (cd "$3" && exec bash -e "$1") >"$WORK/pipe" 2>&1 </dev/null &
      ;;
  esac
  _body=$!

  wait "$_body"
  BODY_RC=$?

  # ⛔ AND ONLY NOW IS THE CLOCK STARTED. The question is not how long the body
  # took; it is whether the output closed once the body was gone.
  #
  # ⭐ THE WAIT IS ON THE READER'S OWN EXIT AND THE BOUND IS AROUND IT, which is
  # `docs/conventions/shell.md` section 10's idiom - `tail --pid` ends when that
  # process does - with `timeout` supplying the ceiling. It blocks, so a body
  # whose pipe closes costs nothing, and it leaves nothing behind.
  #
  # ⚠ TWO EARLIER SHAPES WERE MEASURED AND REJECTED. A `kill -0` poll on a
  # one-second granularity cost a whole second on every case that DID close,
  # because a reader whose last writer has just gone has usually not been
  # scheduled yet - and on a saturated host it could report a closed pipe as
  # open, which is a red row for no defect in a gate that runs twenty-nine
  # checks at once. A `( sleep N && kill ) &` watchdog fixed the cost and left
  # an orphaned `sleep` per case, which is this harness's own subject in its own
  # instrument.
  if timeout "$CLOSE_SECONDS" tail --pid="$_reader" -f /dev/null 2>/dev/null; then
    BODY_CLOSED=yes
  else
    BODY_CLOSED=no
    kill "$_reader" 2>/dev/null || true
  fi
  wait "$_reader" 2>/dev/null || true
  return 0
}

# The two verdicts, asserted together because a case that checked one would pass
# over the other.
ended() { # label want-rc
  if [ "$BODY_RC" != "$2" ]; then
    fail "$1: the step exited $BODY_RC, expected $2"
    return
  fi
  if [ "$BODY_CLOSED" != yes ]; then
    fail "$1: the step exited $2 and its output never closed"
    return
  fi
  pass "$1 (exit $2, output closed)"
}

hangs() { # label want-rc
  if [ "$BODY_RC" != "$2" ]; then
    fail "$1: the step exited $BODY_RC, expected $2"
    return
  fi
  if [ "$BODY_CLOSED" = yes ]; then
    fail "$1: the output closed, so the planted hang did not happen"
    return
  fi
  pass "$1 (exit $2, output HELD OPEN after the command exited)"
}

# -- 1. the instrument's own guards -------------------------------------------
#
# ⛔ A PROBE'S GUARD IS A GUARD LIKE ANY OTHER. Every case below rests on this
# harness being able to tell a step that ended from one that did not, and a
# harness that answered `closed` over everything would report each of them
# passing.
mkdir -p "$WORK/plain" || exit 2

printf 'set -eu\necho hello\n' >"$WORK/plain/ok.sh"
run_body "$WORK/plain/ok.sh" default "$WORK/plain"
ended "probe    a body that exits 0 and leaves nothing ends" 0

printf 'set -eu\necho no\nexit 3\n' >"$WORK/plain/refuse.sh"
run_body "$WORK/plain/refuse.sh" default "$WORK/plain"
ended "probe    a body that refuses reports its own code and still ends" 3

# ⭐ THE SHAPE THE aria2 HANG HAS, PLANTED. A background process that inherited
# the step's stdout holds the pipe after the body is gone.
# ⚠ A BOUNDED SLEEP RATHER THAN SOMETHING ENDLESS, so nothing outlives this
# check indefinitely whatever happens to it. `LEAK_SECONDS` says how long and why.
printf 'set -eu\necho starting\nsleep %s &\nexit 0\n' "$LEAK_SECONDS" >"$WORK/plain/leak.sh"
run_body "$WORK/plain/leak.sh" default "$WORK/plain"
hangs "probe    a body leaving a process on the step's output does not end" 0

# ⛔ AND THE CONTROL THAT SEPARATES THE TWO FACTS. The same background process,
# with its output somewhere else, ends normally - so the case above is about
# holding the OUTPUT and not about leaving a process.
printf 'set -eu\necho starting\nsleep %s >/dev/null 2>&1 &\nexit 0\n' "$LEAK_SECONDS" >"$WORK/plain/noleak.sh"
run_body "$WORK/plain/noleak.sh" default "$WORK/plain"
ended "probe    the same process with its output elsewhere ends" 0

# ⛔ AND THE WRAPPER IS FAITHFUL, WHICH THE pwsh CASES BELOW REST ON. A
# dot-sourced script's exit code collapses through `-command ". '<file>'"`: every
# non-zero value arrives as 1. A harness whose wrapper preserved the code would
# report the run-1 case passing for the wrong reason.
if [ "$HAVE_PWSH" = 1 ]; then
  printf 'exit 2\n' >"$WORK/plain/two.ps1"
  run_body "$WORK/plain/two.ps1" pwsh "$WORK/plain"
  ended "probe    a pwsh body exiting 2 arrives as 1 through the wrapper" 1
else
  fail "probe    pwsh is not on this host, so three cases below did not run"
fi

# -- 2. the scratch tree the capture bodies run against ------------------------
#
# ⛔ THE GUARDS AND THE PRODUCT ARE STUBS AND NOTHING ELSE IS. The scripts these
# bodies call are the real ones: the *Install the client* case runs this
# repository's own `install-client.sh`, so the command chain under test - the
# bound, the command substitutions, the record it writes - is the one a runner
# executes.
TREE="$WORK/tree"
mkdir -p "$TREE/scripts/acquisition" "$TREE/scripts/capture/adapters" "$TREE/bin" \
  "$TREE/state" "$TREE/temp" || exit 2
mkdir -p "$TREE/scripts/ci" || exit 2
cp "$ROOT/scripts/acquisition/install-client.sh" "$TREE/scripts/acquisition/" || exit 2
# ⚠ AND THE STEP'S OWN FILE, WHICH IS WHAT THE BODY NOW RUNS. The install step is
# one `timeout` around this script, so the scratch tree needs the real one or the
# cases would be about a missing file rather than about what it does.
cp "$ROOT/scripts/acquisition/install-step.sh" "$TREE/scripts/acquisition/" || exit 2
# ⚠ THE HOLDER REPORT IS THE REAL ONE TOO. The install block calls it, so a
# scratch tree without it would make both install cases fail on a missing file
# rather than on what they are about.
cp "$ROOT/scripts/ci/report-holders.sh" "$TREE/scripts/ci/" || exit 2
# ⚠ AND THE RULE-12 SCAN `install-client.sh` ASKS, for the same reason: it left
# that file for `scan-bundle.sh` on 2026-10-14, and without it every install case
# here answered 2, which is the scan saying it could not run.
cp "$ROOT/scripts/capture/scan-bundle.sh" "$TREE/scripts/capture/" || exit 2

# ⚠ A `sudo` THAT IS NOT sudo. The RESTORE blocks run `sudo ip route add`, and
# this session is not a place to ask for privilege; dropping the flags and
# running the command is what the runner's passwordless sudo amounts to here.
#
# ⛔ THE INSTALL BODY NO LONGER REACHES IT, AND THAT IS `ACQ-06`. It ran
# `setsid sudo -E timeout` until 2026-09-17, which is why this stub was
# written; the install is rootless now, so the only callers left are the two
# restore blocks. ⚠ The stub is kept for them rather than deleted, and
# `check-rootless.sh` is what refuses a `sudo` returning to the install path.
cat >"$TREE/bin/sudo" <<'STUB'
#!/bin/sh
set -u
while [ $# -gt 0 ]; do
  case "$1" in
    -E | -H | -n) shift ;;
    *) break ;;
  esac
done
exec "$@"
STUB
# ⚠ `ip` IS A STUB TOO, and it reports success. The restore block's own comment
# says an add is an attempt and the routing table is the fact, so what these
# cases vary is the GUARD's answer rather than the add's.
cat >"$TREE/bin/ip" <<'STUB'
#!/bin/sh
set -u
exit 0
STUB
chmod +x "$TREE/bin/sudo" "$TREE/bin/ip" || exit 2

# The egress guard, whose three codes are what the restore blocks read.
guard_says() { # code
  cat >"$TREE/scripts/acquisition/assert-disposable.sh" <<STUB
#!/bin/sh
set -u
exit $1
STUB
  chmod +x "$TREE/scripts/acquisition/assert-disposable.sh"
}
guard_ps_says() { # code
  cat >"$TREE/scripts/acquisition/assert-disposable.ps1" <<STUB
exit $1
STUB
}

: >"$TREE/temp/routes-v4"
: >"$TREE/temp/routes-v6"
# ⚠ An empty CLIXML document, so `Import-Clixml` reads a real file and yields no
# routes. The restore loop then runs zero times, which is the case these three
# are about: the verdict is the guard's, not the add's.
cat >"$TREE/temp/routes.clixml" <<'XML'
<Objs Version="1.1.0.1" xmlns="http://schemas.microsoft.com/powershell/2004/04"></Objs>
XML

PATH="$TREE/bin:$PATH"
export PATH
RUNNER_TEMP="$TREE/temp"
export RUNNER_TEMP
BIT_IDS_STATE_DIR="$TREE/state"
export BIT_IDS_STATE_DIR

# -- 3. capture.yml's restore blocks, both halves ------------------------------
#
# ⛔ THIS IS `CI-08`'s WRITTEN ACCEPTANCE. The entry asks for a harness that
# refuses the *Restore the route* block as it stood on capture run 1 and accepts
# it as it stands, and until now that had been done once by hand in a session
# scratch directory - which is evidence rather than a control.
if lift "$CAPTURE_WF" linux "Restore the route" "$WORK/restore.sh"; then
  guard_says 1
  run_body "$WORK/restore.sh" default "$TREE"
  ended "sh       restore accepts a guard that REFUSES, which is the route back" 0

  guard_says 0
  run_body "$WORK/restore.sh" default "$TREE"
  ended "sh       restore refuses a guard that PASSES, which is a route still cut" 1

  guard_says 2
  run_body "$WORK/restore.sh" default "$TREE"
  ended "sh       restore refuses a guard that COULD NOT RUN" 1
else
  fail "sh       capture.yml has no linux step named Restore the route (exit $?)"
fi

if [ "$HAVE_PWSH" = 1 ]; then
  if lift "$CAPTURE_WF" windows "Restore the route" "$WORK/restore.ps1"; then
    guard_ps_says 1
    run_body "$WORK/restore.ps1" pwsh "$TREE"
    ended "pwsh     restore accepts a guard that REFUSES, under GitHub's wrapper" 0

    guard_ps_says 0
    run_body "$WORK/restore.ps1" pwsh "$TREE"
    ended "pwsh     restore refuses a guard that PASSES" 1

    # ⛔ AND THE RUN-1 FORM, WHICH IS THE DEFECT THIS ENTRY EXISTS FOR. Nothing
    # in the block was wrong; what failed was what the block LEFT BEHIND. Take
    # the explicit `exit 0` away and the wrapper's epilogue reads the guard's
    # own refusal as the step's verdict - so the step fails at the moment the
    # guard proves the route came back, which is exactly what threw away capture
    # run 1's Windows evidence.
    cp "$WORK/restore.ps1" "$WORK/restore-run1.ps1"
    if replace_once "$WORK/restore-run1.ps1" 'exit 0' '# exit 0 - the run-1 form left this to the wrapper'; then
      guard_ps_says 1
      run_body "$WORK/restore-run1.ps1" pwsh "$TREE"
      ended "pwsh     the run-1 form FAILS over the guard that proves the restore" 1
    else
      fail "pwsh     the run-1 plant did not apply"
    fi
  else
    fail "pwsh     capture.yml has no windows step named Restore the route (exit $?)"
  fi
fi

# -- 4. capture-client.yml's install block, which is where the hang lives ------
#
# ⛔ THE STEP THE aria2 LANES DIE IN, RUN HERE. What a session host cannot do is
# reproduce a runner image; what it can do is establish that this block ends over
# a product that behaves, and does NOT end over one that leaves a process on the
# step's output. The second is the shape run 7 has and no existing check could
# have seen it.
#
# ⛔ AND THE REAL GUARD GOES BACK, because `install-client` asks it where the
# claim marker lives and that derivation is the one under test. ⚠ The section
# above leaves a stub answering 2 behind, and the first version of this file ran
# these two cases against it: `--marker` printed nothing, `install-client`
# reported it could not run, and both cases failed for a reason neither was
# about. A harness that shares scratch state with an earlier section owes the
# next one a known starting point.
cp "$ROOT/scripts/acquisition/assert-disposable.sh" "$TREE/scripts/acquisition/" || exit 2
: >"$TREE/state/host-claimed"

stub_adapter() { # leak-line
  cat >"$TREE/scripts/capture/adapters/stub.sh" <<STUB
#!/bin/sh
set -u
CMD="\$1"
shift
case "\$CMD" in
  describe)
    printf 'target=stub\n'
    printf 'kind=stock\n'
    ;;
  install)
    [ \$# -ge 2 ] || exit 2
    mkdir -p "\$2" || exit 2
    printf 'installed\n' >"\$2/install.log"
    # ⛔ EVERY ADAPTER SAYS HOW ITS ROUTE PACKAGED THE BUILD, and install-client
    # refuses one that installed without saying. A stub that skipped this would
    # make these four cases red for a reason the case did not plant - which is
    # exactly what it did on 2026-09-09, the day that guard landed.
    printf 'elf-binary\n' >"\$2/package"
    $1
    exit 0
    ;;
  version) printf '9.9.9\n' ;;
  *) exit 2 ;;
esac
STUB
  chmod +x "$TREE/scripts/capture/adapters/stub.sh"
}

if lift "$CLIENT_WF" linux "Install the client" "$WORK/install.sh"; then
  ADAPTER=scripts/capture/adapters/stub.sh
  ROUTE=package
  # ⚠ A ONE-SECOND TICK RATHER THAN THE SHIPPED FIVE, so a case that is over in
  # milliseconds does not pay a whole tick for it. The DEADLINE stays the
  # shipped one except where a case is about the deadline.
  BIT_IDS_INSTALL_INTERVAL=1
  export ADAPTER ROUTE BIT_IDS_INSTALL_INTERVAL

  stub_adapter ':'
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  run_body "$WORK/install.sh" default "$TREE"
  ended "sh       the install block ends over a product that behaves" 0

  # ⭐ AND THE ONE THAT MATTERS: THE SAME BLOCK OVER A PRODUCT THAT LEAKS ONE
  # PROCESS, WHICH NOW ENDS. The route's output goes to a file, so nothing it
  # spawns inherits the step's own pipe and the runner has nothing left to wait
  # on. ⚠ The leaked process is still there - this does not stop a product
  # leaking, it stops a leak holding the step open.
  stub_adapter "sleep $LEAK_SECONDS &"
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  run_body "$WORK/install.sh" default "$TREE"
  ended "sh       the install block ends over a product that leaks one process" 0

  # ⛔ AND THE CONTROL THAT SAYS THE REDIRECTION IS WHAT DOES IT. Without this
  # the case above passes equally over a block that never had the problem. The
  # plant hands the route one extra descriptor onto the step's own output -
  # `3>&1`, which is how such a thing arrives in real life - and the same
  # product, leaking the same single process, then holds the step open with an
  # exit code of 0 in it.
  #
  # ⚠ THE ORDER OF THE REDIRECTIONS IS THE WHOLE PLANT, and the first version
  # got it wrong: a shell applies them left to right, so `>log 2>&1 3>&1` points
  # fd 3 at the LOG - fd 1 is already the log by then - and the case reported the
  # hang not happening over a plant that had duplicated the wrong file. Written
  # before the redirection, `3>&1` is the step's own pipe.
  #
  # ⛔ IT PLANTS IN THE LIFTED BODY NOW, AND THAT MOVE IS THE 2026-09-16 REPAIR.
  # It used to plant in the tree's copy of `install-step.sh`, on the reasoning
  # that the body was one `timeout` around that script. ⚠ That reasoning was
  # correct and the LEVEL was wrong: the step's pipe was reachable from
  # `install-step.sh` at all only because that script inherited it, which is
  # exactly the defect run 24 measured - a step still open twenty minutes in,
  # having passed both its bounds. The body hands the whole subtree a FILE now,
  # so `install-step.sh` can no longer reach the step's pipe and a plant there
  # would duplicate a log rather than the pipe, reporting no hang over a tree
  # that had genuinely stopped being able to hang.
  # ⭐ So the plant goes where the pipe actually is: the body's own redirection.
  # ⚠ THE ORDER OF THE REDIRECTIONS IS STILL THE WHOLE PLANT. Written BEFORE the
  # `>`, `3>&1` duplicates the step's own pipe; written after, it would duplicate
  # the log and prove nothing.
  # ⛔ THE DOLLAR SIGNS ARE THE POINT. This is the body's own text, matched and
  # replaced literally; a shell that expanded them would look for a line the file
  # does not contain and the plant would not apply.
  cp "$WORK/install.sh" "$WORK/install-leaky.sh"
  # shellcheck disable=SC2016
  if replace_once "$WORK/install-leaky.sh" \
    '>"$RUNNER_TEMP/install-step.log" 2>&1 </dev/null' \
    '3>&1 >"$RUNNER_TEMP/install-step.log" 2>&1 </dev/null'; then
    stub_adapter "sleep $LEAK_SECONDS &"
    rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
    run_body "$WORK/install-leaky.sh" default "$TREE"
    hangs "sh       one leaked descriptor onto the step's output brings it back" 0
  else
    fail "sh       the leaked-descriptor plant did not apply"
  fi
  cp "$ROOT/scripts/acquisition/install-step.sh" "$TREE/scripts/acquisition/" || exit 2

  # ⛔ AND THE STEP'S OWN BOUND IS SEEN TO FIRE, because a bound nobody has
  # watched fire is a bound nobody knows works - and this repository has shipped
  # THREE that did not: `timeout` inside install-client, the runner's
  # `timeout-minutes`, and a watchdog loop in the step's own shell. This one is
  # around the step's own process, which is why it can end a step whose insides
  # are not reachable.
  # ⚠ 124 IS coreutils' VERDICT FOR "IT NEVER ANSWERED", and it is the whole
  # point: the step ENDS, so the job goes on to upload the evidence.
  #
  # ⛔ THE LITERAL CARRIES THE WHOLE `${OUTER_TIMEOUT:-1080}` AND THAT IS
  # DELIBERATE. The bound became per route on 2026-09-16 and moved into the
  # step's `env:`, because a `${{ }}` expression left in the `run:` line is not
  # shell and this harness LIFTS that line - measured the same day, five cases
  # failed at once over an unexpanded expression. ⚠ A literal that drifts is
  # caught rather than skipped: `replace_once` demands exactly one hit and that
  # the file actually changed, so a stale pattern reports *did not apply* rather
  # than a plant that silently proved nothing.
  cp "$WORK/install.sh" "$WORK/install-bounded.sh"
  # ⚠ SC2016 is the point rather than an oversight: `replace_once` matches a
  # LITERAL, so the `${OUTER_TIMEOUT:-1080}` must arrive unexpanded. Expanding it
  # here would plant against this harness's own environment instead of against
  # the text the workflow ships.
  # shellcheck disable=SC2016
  if replace_once "$WORK/install-bounded.sh" \
    'timeout -k 30 "${OUTER_TIMEOUT:-1080}" ' 'timeout -k 1 2 '; then
    stub_adapter 'sleep 20'
    rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
    run_body "$WORK/install-bounded.sh" default "$TREE"
    ended "sh       the bound around the step's own process fires and it ends" 124
  else
    fail "sh       the step-bound plant did not apply"
  fi

  # ⛔ AND THE CONTROL: THE SHIPPED BOUND DOES NOT KILL A PRODUCT THAT FINISHES
  # INSIDE IT. Without it the case above passes equally over a block that kills
  # every install it is given, which would be a bound that refuses correct work.
  stub_adapter 'sleep 2'
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  run_body "$WORK/install.sh" default "$TREE"
  ended "sh       a product slower than one tick and inside the bound survives" 0
else
  fail "sh       capture-client.yml has no linux step named Install the client (exit $?)"
fi

# -- 4b. the flight recorder, started and stopped the way a job does ---------
#
# ⛔ NOTHING CLOSED ITS CHECK RUN UNTIL 2026-10-14. The recorder ran to a limit no
# job reaches and the runner killed it with the job, so every run it opened
# stayed `in_progress` - the silence it exists to give meaning to, produced by a
# job that ended cleanly. ⭐ These cases run the workflow's own *Start* and *Stop*
# bodies against a stub `curl` that answers the way the checks API does.
#
# ⚠ THE STUB IS ON ITS OWN PATH ENTRY, prepended only while these cases run, so no
# other body in this file can reach a `curl` that answers nothing true.
RECBIN="$WORK/recbin"
mkdir -p "$RECBIN" "$TREE/scripts/ci" || exit 2
cp "$ROOT/scripts/ci/flight-recorder.sh" "$TREE/scripts/ci/" || exit 2
cat >"$RECBIN/curl" <<'STUBCURL'
#!/bin/sh
# Answers the checks API's create and update, and logs what each call asked for.
set -u
method=GET
url=""
body=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method=$2; shift ;;
    --data-binary) body=${2#@}; shift ;;
    -o | -w | -H | --max-time) shift ;;
    http*) url=$1 ;;
  esac
  shift
done
status=none
[ -z "$body" ] || status=$(jq -r '.status // "none"' "$body" 2>/dev/null)
printf '%s %s %s\n' "$method" "$url" "$status" >>"$STUB_CURL_LOG"
case "$method" in
  POST) printf '{"id": 7, "status": "in_progress"}\n' ;;
  PATCH)
    # ⚠ A completion the API refused answers the run's old status, which is the
    # read-back the recorder has to notice.
    if [ "$status" = completed ] && [ "${STUB_CURL_REFUSE:-}" = yes ]; then
      printf '{"id": 7, "status": "in_progress"}\n'
    else
      printf '{"id": 7, "status": "%s"}\n' "$([ "$status" = none ] && printf in_progress || printf '%s' "$status")"
    fi
    ;;
  *) printf '200 in 0.001s\n' ;;
esac
STUBCURL
chmod +x "$RECBIN/curl"

RECLABEL="step-bodies-recorder-$$"
GH_TOKEN=stub-token
GITHUB_API_URL=http://127.0.0.1:9
GITHUB_REPOSITORY=fixture-owner/fixture-repo
GITHUB_SHA=0000000
LABEL=$RECLABEL
BIT_IDS_RECORDER_INTERVAL=1
BIT_IDS_RECORDER_LIMIT=120
STUB_CURL_LOG="$WORK/curl.log"
STUB_CURL_REFUSE=""
export GH_TOKEN GITHUB_API_URL GITHUB_REPOSITORY GITHUB_SHA LABEL \
  BIT_IDS_RECORDER_INTERVAL BIT_IDS_RECORDER_LIMIT STUB_CURL_LOG STUB_CURL_REFUSE

# ⛔ `[f]light`, NOT `flight`. The pattern is a regular expression matched against
# every command line, including that of a shell carrying the pattern as an
# argument; written plainly, the bounded wait below matched its own `sh -c` and
# reported a recorder that had exited as still running. The bracket matches the
# recorder and cannot match its own spelling.
recorder_running() {
  pgrep -f "[f]light-recorder.sh $RECLABEL" >/dev/null 2>&1
}

# ⚠ The done file is written a moment before the recorder exits, so "gone" is
# asked with a bound rather than at one instant.
recorder_gone() {
  # shellcheck disable=SC2016
  timeout 5 sh -c 'while pgrep -f "$1" >/dev/null 2>&1; do sleep 1; done' sh \
    "[f]light-recorder.sh $RECLABEL"
}

# One start, a wait, and one stop, through the lifted bodies.
record_once() { # temp-dir
  RUNNER_TEMP=$1
  rm -rf "$RUNNER_TEMP" "$STUB_CURL_LOG"
  mkdir -p "$RUNNER_TEMP" || return 2
  _path=$PATH
  PATH="$RECBIN:$PATH"
  run_body "$WORK/rec-start.sh" default "$TREE"
  PATH=$_path
}

if lift "$CLIENT_WF" linux "Start the flight recorder" "$WORK/rec-start.sh" &&
  lift "$CLIENT_WF" linux "Stop the flight recorder" "$WORK/rec-stop.sh"; then
  record_once "$WORK/rec-temp"
  ended "sh       the recorder's start step ends while the recorder runs on" 0
  if recorder_running && grep -q '^POST ' "$STUB_CURL_LOG" 2>/dev/null; then
    pass "sh       the recorder opened its check run and is still posting"
  else
    fail "sh       the recorder opened its check run and is still posting"
  fi

  # ⛔ THE DEFECT, SHOWN RATHER THAN ASSUMED: with nothing to stop it, three of
  # its intervals later the run is still open. Without this the case below passes
  # equally over a recorder that closes itself at once and posts nothing.
  sleep 3
  if recorder_running && ! grep -q ' completed$' "$STUB_CURL_LOG"; then
    pass "sh       with nothing to stop it, the recorder's check run stays open"
  else
    fail "sh       with nothing to stop it, the recorder's check run stays open"
  fi

  run_body "$WORK/rec-stop.sh" default "$TREE"
  ended "sh       the stop step ends" 0
  if [ "$(tail -1 "$STUB_CURL_LOG")" = "PATCH $GITHUB_API_URL/repos/$GITHUB_REPOSITORY/check-runs/7 completed" ] &&
    grep -q -F -e 'completed: the job reached its last step' "$RUNNER_TEMP/flight-recorder.done" 2>/dev/null &&
    recorder_gone; then
    pass "sh       the stop step closes the check run, reads it back, and the recorder is gone"
  else
    fail "sh       the stop step closes the check run, reads it back, and the recorder is gone"
    [ "$JSON" = "1" ] || tail -3 "$STUB_CURL_LOG" 2>/dev/null | sed 's/^/          /'
  fi

  # ⛔ AND A COMPLETION THE API DID NOT ACCEPT IS NOT REPORTED AS ONE. A recorder
  # that wrote `completed` because it had sent the request would be a claim about
  # a check run nobody read back.
  STUB_CURL_REFUSE=yes
  record_once "$WORK/rec-refused"
  run_body "$WORK/rec-stop.sh" default "$TREE"
  STUB_CURL_REFUSE=""
  if [ "$BODY_RC" = 0 ] &&
    grep -q -F -e 'not confirmed' "$RUNNER_TEMP/flight-recorder.done" 2>/dev/null; then
    pass "sh       a completion the API did not accept is reported as not confirmed"
  else
    fail "sh       a completion the API did not accept is reported as not confirmed"
  fi

  # ⚠ AND THE STOP NEVER FAILS A JOB, because the recorder is a diagnostic. With
  # no recorder to answer, the step waits out its bound - shortened here, which
  # `replace_once` refuses to do silently - and still ends 0, saying so.
  cp "$WORK/rec-stop.sh" "$WORK/rec-stop-short.sh"
  if replace_once "$WORK/rec-stop-short.sh" 'timeout 60 ' 'timeout 2 '; then
    RUNNER_TEMP="$WORK/rec-nobody"
    rm -rf "$RUNNER_TEMP"
    mkdir -p "$RUNNER_TEMP" || exit 2
    run_body "$WORK/rec-stop-short.sh" default "$TREE"
    if [ "$BODY_RC" = 0 ] && grep -q -F -e 'did not confirm' "$BODY_OUT"; then
      pass "sh       a stop with no recorder answering ends 0 and says it was not confirmed"
    else
      fail "sh       a stop with no recorder answering ends 0 (exit $BODY_RC) and says so"
    fi
  else
    fail "sh       the shortened-wait plant did not apply"
  fi
  RUNNER_TEMP="$TREE/temp"
  # ⚠ A case that failed can leave a recorder posting to nothing until its
  # limit, so whatever is left of this label's is ended here.
  pkill -f "[f]light-recorder.sh $RECLABEL" 2>/dev/null || true
else
  fail "sh       capture-client.yml has no linux step named Start or Stop the flight recorder"
fi

# -- 5. a step this harness names and the workflow does not have --------------
#
# ⛔ A HARNESS THAT NAMED A DELETED STEP WOULD REPORT NOTHING RATHER THAN A
# FAILURE, which is the rot `check-workflow.sh` already refuses for step order.
# The reader answers 3 for that and 1 for a step that runs an action, and the
# two are separated here rather than downstream.
lift "$CAPTURE_WF" linux "A step nobody wrote" "$WORK/absent" 2>/dev/null
case "$?" in
  3) pass "probe    a step the workflow does not have is reported, not skipped" ;;
  *) fail "probe    an absent step answered $? rather than 3" ;;
esac

lift "$CLIENT_WF" linux "Upload the install logs" "$WORK/uses" 2>/dev/null
case "$?" in
  1) pass "probe    a step that runs an action is a different answer from an absent one" ;;
  *) fail "probe    an action step answered $? rather than 1" ;;
esac

store_probe_guards "$WORK/restore.sh" 'egress=0' 'exit 1'

store_report check-step-bodies/1 cases "$JSON"
