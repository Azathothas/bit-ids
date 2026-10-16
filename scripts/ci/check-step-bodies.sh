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
# block over a stub rather than a capture. ⚠ The containment is the exception:
# its cases make a real network namespace, through `sudo -n`, and a host that
# cannot is a failed row rather than a pass.
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

# ⚠ AND WHAT IT SAID, where two refusals share a code. A case asserting exit 1
# alone passes when a different refusal fired.
said() { # label text
  if grep -q -F -e "$2" "$BODY_OUT" 2>/dev/null; then
    pass "$1"
  else
    fail "$1: the step did not say '$2'"
  fi
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
# repository's own `bit-check install-client`, so the command chain under test - the
# bound, the command substitutions, the record it writes - is the one a runner
# executes.
TREE="$WORK/tree"
mkdir -p "$TREE/scripts/acquisition" "$TREE/scripts/capture/adapters" "$TREE/bin" \
  "$TREE/state" "$TREE/temp" || exit 2
mkdir -p "$TREE/scripts/ci" || exit 2
# ⚠ THE STEP ITSELF IS `bit-check install-step` SINCE 2026-10-15, `CI-11`, so
# the tree needs no copy of it: BIT_CHECK, built below, is what the body runs.
# ⚠ THE HOLDER REPORT IS THE REAL ONE TOO. The install block calls it, so a
# scratch tree without it would make both install cases fail on a missing file
# rather than on what they are about.
cp "$ROOT/scripts/ci/report-holders.sh" "$TREE/scripts/ci/" || exit 2
# ⚠ AND THE RULE-12 SCAN `bit-check install-client` ASKS, for the same reason: without
# it every install case here answered 2, which is the scan saying it could not
# run. It is `bit-check scan-bundle` since 2026-10-15, `CI-11`, found through
# BIT_CHECK as the workflow's own *Build the observer* step exports it.
command -v go >/dev/null 2>&1 || {
  printf 'check-step-bodies: go is not on this host, so bit-check cannot be built\n' >&2
  exit 2
}
(cd "$ROOT/tools/check" && go build -o "$TREE/bin/bit-check" .) >/dev/null 2>&1 || {
  printf 'check-step-bodies: cannot build bit-check\n' >&2
  exit 2
}
BIT_CHECK="$TREE/bin/bit-check"
export BIT_CHECK

PATH="$TREE/bin:$PATH"
export PATH
RUNNER_TEMP="$TREE/temp"
export RUNNER_TEMP
BIT_IDS_STATE_DIR="$TREE/state"
export BIT_IDS_STATE_DIR

# -- 3. capture.yml's firewall proof, the Windows half -------------------------
#
# ⛔ THE RESTORE BLOCKS RETIRED WITH THE ROUTE CUT. Their cases were `CI-08`'s
# first acceptance: a harness refusing the *Restore the route* block as it stood
# on capture run 1. The Linux job stopped cutting on 2026-10-14 and the Windows
# job on 2026-10-15, so there is no restore left to run, and the `sudo` and `ip`
# stubs those cases needed went with them.
#
# ⭐ WHAT THE WINDOWS JOB RUNS INSTEAD is *Prove the firewall blocks*: the
# capture's own blocked driver tries to leave, and only `curl` exit 7 is a
# block. Run here under GitHub's wrapper with the driver and the program list
# stubbed, and the control pointed at a port nothing listens on, so no case
# sends a packet off this host. ⚠ That leaves the step's passing path unrun
# here; a capture dispatch is what runs it, against a real firewall.
if [ "$HAVE_PWSH" = 1 ]; then
  if lift "$CAPTURE_WF" windows "Prove the firewall blocks" "$WORK/prove.ps1"; then
    # The driver answers whatever exit the case asks for.
    cat >"$TREE/bin/curl.exe" <<'STUB'
#!/bin/sh
exit "${BIT_IDS_STUB_CURL_RC:-0}"
STUB
    chmod +x "$TREE/bin/curl.exe" || exit 2
    # ⚠ `capture-run.ps1 -Programs` is the list's only derivation, so the stub
    # stands in for it and nothing else: it names the stub driver, or a list
    # with no driver in it.
    programs_are() { # path...
      {
        for _p in "$@"; do printf "Write-Output '%s'\n" "$_p"; done
        printf 'exit 0\n'
      } >"$TREE/scripts/capture/capture-run.ps1"
    }
    OBSERVER="$TREE/bin/observer.exe"
    PROBE_URL="http://127.0.0.1:9/"
    export OBSERVER PROBE_URL

    programs_are "$OBSERVER" "$TREE/bin/curl.exe"
    BIT_IDS_STUB_CURL_RC=0
    export BIT_IDS_STUB_CURL_RC
    run_body "$WORK/prove.ps1" pwsh "$TREE"
    ended "pwsh     a blocked driver that reached the address fails the proof" 1
    said "pwsh     ... and says the driver did not fail to connect" "did not fail to connect (curl exit 0)"

    # ⛔ A TIMEOUT IS NOT A BLOCK. A firewall that refused nothing on a slow
    # network times out too, and a step that took 28 for 7 would pass it.
    BIT_IDS_STUB_CURL_RC=28
    export BIT_IDS_STUB_CURL_RC
    run_body "$WORK/prove.ps1" pwsh "$TREE"
    ended "pwsh     a driver that timed out is not proof of a block" 1
    said "pwsh     ... and says so, before the control is asked" "did not fail to connect (curl exit 28)"

    # ⛔ AND A REFUSAL WITH NO WORKING HOST BEHIND IT PROVES NOTHING. The control
    # is the host itself reaching the same address; here it reaches a port
    # nothing listens on, which is what a host with no network looks like.
    BIT_IDS_STUB_CURL_RC=7
    export BIT_IDS_STUB_CURL_RC
    run_body "$WORK/prove.ps1" pwsh "$TREE"
    ended "pwsh     a block with no reachable control proves nothing" 1
    said "pwsh     ... and says the refusal proves nothing" "the refusal proves nothing"

    # ⭐ AND ANY HTTP ANSWER IS A REACHED HOST. A server that answers 404 is
    # still the address answering; capture-client run 38829879023's control was
    # refused for an HTTP error status from api.github.com. `CI-11`.
    _port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
    mkdir -p "$WORK/www" || exit 2
    python3 -m http.server "$_port" --bind 127.0.0.1 --directory "$WORK/www" >/dev/null 2>&1 &
    _server=$!
    _waited=0
    until python3 -c 'import socket,sys; socket.create_connection(("127.0.0.1", int(sys.argv[1])), 1)' "$_port" 2>/dev/null; do
      _waited=$((_waited + 1))
      [ "$_waited" -lt 20 ] || break
      sleep 0.25
    done
    PROBE_URL="http://127.0.0.1:$_port/no-such-path"
    BIT_IDS_STUB_CURL_RC=7
    export PROBE_URL BIT_IDS_STUB_CURL_RC
    run_body "$WORK/prove.ps1" pwsh "$TREE"
    ended "pwsh     a control answered with an HTTP error still proves the block" 0
    kill "$_server" 2>/dev/null
    wait "$_server" 2>/dev/null
    PROBE_URL="http://127.0.0.1:9/"
    export PROBE_URL

    programs_are "$OBSERVER"
    BIT_IDS_STUB_CURL_RC=7
    export BIT_IDS_STUB_CURL_RC
    run_body "$WORK/prove.ps1" pwsh "$TREE"
    ended "pwsh     a program list with no driver in it cannot prove anything" 1
    said "pwsh     ... and says there is no driver" "names no curl.exe driver"
    rm -f "$TREE/scripts/capture/capture-run.ps1" "$TREE/bin/curl.exe"
    unset OBSERVER PROBE_URL BIT_IDS_STUB_CURL_RC
  else
    fail "pwsh     capture.yml has no windows step named Prove the firewall blocks (exit $?)"
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
# ⛔ THE REAL GUARD IS THE ONE `install-client` ASKS, because where the claim
# marker lives is the derivation under test. It is `bit-check assert-disposable`
# since 2026-10-15, `CI-11`, and `install-client` reads that derivation in
# process, so the tree needs no copy of a guard; a stub one, as the restore
# cases above once left behind, could no longer answer for it.
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

  # ⛔ AND WHAT IT LEAKED IS ENDED. The leak is an inner `timeout` with a group
  # of its own, which is the shape measured on 2026-09-16: a bound reaches its
  # own group and never the one beneath. `bit-check install-step` sweeps the install's
  # session, which every group in it shares. `CI-08`.
  stub_adapter "timeout 600 sleep 7123 >/dev/null 2>&1 &"
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  run_body "$WORK/install.sh" default "$TREE"
  ended "sh       the install block ends over a product that leaves a group behind" 0
  # ⛔ MATCHED ON THE WHOLE ARGUMENT VECTOR, never `pgrep -f`, which matches any
  # command line that merely CONTAINS the text - a shell whose script quotes it
  # included - and whose `pkill` twin then kills that shell. Measured on the
  # session host on 2026-10-14, where it ended the shell running this harness.
  _left=$(ps -eo pid=,args= 2>/dev/null |
    awk '{ $1 = $1 } / sleep 7123$/ && ($2 " " $3 == "sleep 7123" || $2 " " $3 " " $4 " " $5 == "timeout 600 sleep 7123") { print $1 }')
  if [ -n "$_left" ]; then
    fail "sh       the leaked group is still running after the step"
    # shellcheck disable=SC2086
    kill $_left 2>/dev/null
  elif grep -q 'sleep 7123' "$TREE/temp/install-package/orphans.log" 2>/dev/null; then
    pass "sh       the leaked group is ended, and orphans.log names it"
  else
    fail "sh       the leaked group is gone and orphans.log does not name it"
  fi

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
    # ⚠ THE LEAK OPENS A SESSION OF ITS OWN, and has since 2026-10-14.
    # `bit-check install-step` ends whatever the install leaves in its session,
    # so a leaked `sleep` there no longer holds anything open, and this control
    # went red over a step that had stopped being able to hang. A process that
    # left the session is the one no sweep can find, and the redirection is
    # still the only thing between it and the step's pipe.
    stub_adapter "setsid sleep $LEAK_SECONDS &"
    rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
    run_body "$WORK/install-leaky.sh" default "$TREE"
    hangs "sh       one leaked descriptor onto the step's output brings it back" 0
  else
    fail "sh       the leaked-descriptor plant did not apply"
  fi

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

  # ⛔ AND THE STEP'S OWN DEADLINE, INSIDE IT, IS SEEN TO FIRE TOO. The case
  # above fires the OUTER bound; this one leaves that bound as shipped and
  # shortens `install-step`'s own, which is what reports 124 and says the
  # install is hung while the step can still print what it has. ⚠ Nothing
  # exercised it until 2026-10-15, when a mutant of the Go port that answered 0
  # at that deadline passed every case here.
  stub_adapter 'sleep 20'
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  BIT_IDS_STEP_TIMEOUT=3
  BIT_IDS_INSTALL_INTERVAL=1
  export BIT_IDS_STEP_TIMEOUT BIT_IDS_INSTALL_INTERVAL
  run_body "$WORK/install.sh" default "$TREE"
  unset BIT_IDS_STEP_TIMEOUT BIT_IDS_INSTALL_INTERVAL
  ended "sh       the step's own deadline fires inside the outer bound" 124
  if grep -q -F -e "it is hung, not slow" "$BODY_OUT"; then
    pass "sh       ... and the step says the install is hung"
  else
    fail "sh       the step's own deadline fired without saying the install is hung"
  fi

  # ⛔ AND THE CONTROL: THE SHIPPED BOUND DOES NOT KILL A PRODUCT THAT FINISHES
  # INSIDE IT. Without it the case above passes equally over a block that kills
  # every install it is given, which would be a bound that refuses correct work.
  stub_adapter 'sleep 2'
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  run_body "$WORK/install.sh" default "$TREE"
  ended "sh       a product slower than one tick and inside the bound survives" 0

  # ⛔ AND A REFUSED INSTALL'S DIAGNOSIS REACHES THE STEP'S OUTPUT. The body
  # carried the exit code as `rc=$?` on the line after the install, and the
  # runner runs a body as `bash -e`: a failing install ended the shell on its own
  # line and the `cat` after it never ran. Measured on `capture-client` run 35,
  # whose source lane failed to compile and whose job log printed none of it.
  # ⚠ The exit code is asked too, because a body that printed the log and then
  # ended 0 would pass the half of this case that reads the output.
  stub_adapter 'printf "stub refused its install\n" >&2; exit 1'
  rm -rf "$TREE/temp/install-package" "$TREE/temp/install-package.txt"
  run_body "$WORK/install.sh" default "$TREE"
  if ! grep -q -F -e 'stub refused its install' "$BODY_OUT" 2>/dev/null; then
    fail "sh       a refused install's diagnosis never reached the step's output (exit $BODY_RC)"
  else
    ended "sh       a refused install ends 1 and its diagnosis reaches the step's output" 1
  fi
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
# ⚠ The *Start* body runs `"$BIT_CHECK" flight-recorder` since 2026-10-16
# (`CI-11`), so the tree needs no copy of a script, and the recorder's requests
# still go through `curl`, which is what the stub answers.
RECBIN="$WORK/recbin"
mkdir -p "$RECBIN" || exit 2
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
# Every request that asks to close the run is kept whole and in order, so a case
# can read what the first one said and whether a second was sent.
if [ "$status" = completed ]; then
  _n=1
  while [ -e "$STUB_CURL_LOG.completion.$_n" ]; do _n=$((_n + 1)); done
  cp "$body" "$STUB_CURL_LOG.completion.$_n"
fi
conclusion=$([ -z "$body" ] || jq -r '.conclusion // "none"' "$body" 2>/dev/null)
# ⚠ A request that never got an answer prints nothing and exits 28, which is
# what curl does when `--max-time` ends it; `STUB_CURL_SILENT` stands for that.
if [ "$status" = completed ] && [ "${STUB_CURL_SILENT:-}" = yes ]; then
  exit 28
fi
case "$method" in
  POST) printf '{"id": 7, "status": "in_progress"}\n' ;;
  PATCH)
    # ⚠ THE API'S OWN REFUSALS. A completion with no conclusion it knows is a
    # 422 with a message and no status, and so is a text it will not take,
    # which `STUB_CURL_REFUSE_TEXT` stands for.
    if [ "$status" = completed ] && ! printf '%s\n' "$conclusion" |
      grep -q -x -E 'action_required|cancelled|failure|neutral|success|skipped|timed_out'; then
      printf '{"message": "Invalid request. conclusion is required"}\n'
    elif [ "${STUB_CURL_REFUSE_TEXT:-}" = yes ] && jq -e '.output.text' "$body" >/dev/null 2>&1 &&
      [ "$status" = completed ]; then
      printf '{"message": "Validation Failed"}\n'
    # ⚠ A completion the API refused answers the run's old status, which is the
    # read-back the recorder has to notice.
    elif [ "$status" = completed ] && [ "${STUB_CURL_REFUSE:-}" = yes ]; then
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
STUB_CURL_REFUSE_TEXT=""
STUB_CURL_SILENT=""
export GH_TOKEN GITHUB_API_URL GITHUB_REPOSITORY GITHUB_SHA LABEL \
  BIT_IDS_RECORDER_INTERVAL BIT_IDS_RECORDER_LIMIT STUB_CURL_LOG STUB_CURL_REFUSE \
  STUB_CURL_REFUSE_TEXT STUB_CURL_SILENT

# ⛔ `[f]light`, NOT `flight`. The pattern is a regular expression matched against
# every command line, including that of a shell carrying the pattern as an
# argument; written plainly, the bounded wait below matched its own `sh -c` and
# reported a recorder that had exited as still running. The bracket matches the
# recorder and cannot match its own spelling.
recorder_running() {
  pgrep -f "[f]light-recorder $RECLABEL" >/dev/null 2>&1
}

# ⚠ The done file is written a moment before the recorder exits, so "gone" is
# asked with a bound rather than at one instant.
recorder_gone() {
  # shellcheck disable=SC2016
  timeout 5 sh -c 'while pgrep -f "$1" >/dev/null 2>&1; do sleep 1; done' sh \
    "[f]light-recorder $RECLABEL"
}

# One start, a wait, and one stop, through the lifted bodies.
record_once() { # temp-dir
  RUNNER_TEMP=$1
  rm -rf "$RUNNER_TEMP" "$STUB_CURL_LOG" "$STUB_CURL_LOG".completion.*
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

  # ⛔ AND THE CLOSED RUN STILL SHOWS THE STATE THE JOB ENDED IN. Measured on
  # `capture-client` run 35: the completion went up as a second request with a
  # title and no `text`, and both check runs that closed cleanly showed an empty
  # one. ⚠ `route to the API` is the section every snapshot ends with, so a text
  # that carries it carries a whole snapshot rather than a title; the title is
  # the ending's, so it is the LAST snapshot rather than a loop's left over; and
  # one request closed it, so nothing was retried.
  if jq -e '(.output.text | test("### route to the API")) and
      (.output.title | startswith("the job reached its last step at +")) and
      .conclusion == "neutral"' "$STUB_CURL_LOG.completion.1" >/dev/null 2>&1 &&
    [ ! -e "$STUB_CURL_LOG.completion.2" ]; then
    pass "sh       one request closes the check run, carrying the last snapshot"
  else
    fail "sh       one request closes the check run, carrying the last snapshot"
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

  # ⚠ AND A LAST SNAPSHOT THE API REFUSES DOES NOT KEEP THE RUN OPEN. One
  # request carries the text and the completion, so a text the API would not
  # take would refuse both; the recorder asks again with no `output`.
  STUB_CURL_REFUSE_TEXT=yes
  record_once "$WORK/rec-text-refused"
  run_body "$WORK/rec-stop.sh" default "$TREE"
  STUB_CURL_REFUSE_TEXT=""
  if [ "$BODY_RC" = 0 ] &&
    grep -q -F -e 'completed: the job reached its last step' "$RUNNER_TEMP/flight-recorder.done" 2>/dev/null &&
    jq -e 'has("output") | not' "$STUB_CURL_LOG.completion.2" >/dev/null 2>&1; then
    pass "sh       a last snapshot the API refuses still closes the run, without it"
  else
    fail "sh       a last snapshot the API refuses still closes the run, without it"
  fi

  # ⚠ AND A CLOSING REQUEST THAT GOT NO ANSWER IS NOT SENT AGAIN. Nothing came
  # back to say the text was refused, and a second request would spend the wait
  # the job's last step gives the recorder.
  STUB_CURL_SILENT=yes
  record_once "$WORK/rec-silent"
  run_body "$WORK/rec-stop.sh" default "$TREE"
  STUB_CURL_SILENT=""
  if [ "$BODY_RC" = 0 ] &&
    grep -q -F -e 'not confirmed' "$RUNNER_TEMP/flight-recorder.done" 2>/dev/null &&
    [ -e "$STUB_CURL_LOG.completion.1" ] && [ ! -e "$STUB_CURL_LOG.completion.2" ]; then
    pass "sh       a closing request that got no answer is not sent again"
  else
    fail "sh       a closing request that got no answer is not sent again"
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
  pkill -f "[f]light-recorder $RECLABEL" 2>/dev/null || true
else
  fail "sh       capture-client.yml has no linux step named Start or Stop the flight recorder"
fi

# -- 4c. the install barrier, between lanes that share nothing but a commit ---
#
# ⛔ RUN 37'S RELEASE LANE OBSERVED BEFORE THE SOURCE ROUTE HAD INSTALLED, on
# another runner, so a run manifest could not say the build was acquired twice
# first. These cases run the workflow's own *Arrive* and *Wait* bodies against a
# stub `curl` that keeps the check runs it is sent in a directory and answers a
# query by name, which is the one place two runners can meet.
# ⚠ The bodies run `"$BIT_CHECK" install-barrier` since 2026-10-15 (`CI-11`), so
# the tree needs no copy of a script: BIT_CHECK, built above, is what they run.
# Its `curl` takes the token from a header file, `-H @file`, which the stub's
# `-H` arm skips as it skipped the literal header.
BARBIN="$WORK/barbin"
BARDB="$WORK/bardb"
mkdir -p "$BARBIN" || exit 2
cat >"$BARBIN/curl" <<'STUBCURL'
#!/bin/sh
# Keeps posted check runs by name; answers a query for one name with it.
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
mkdir -p "$STUB_BARRIER_DB"
case "$method" in
  POST)
    name=$(jq -r '.name' "$body")
    key=$(printf '%s' "$name" | sha256sum | cut -d' ' -f1)
    cp "$body" "$STUB_BARRIER_DB/$key.json"
    printf '{"id": 9}\n'
    ;;
  GET)
    q=${url#*check_name=}
    q=${q%%&*}
    name=$(printf '%s' "$q" | python3 -I -c 'import sys,urllib.parse; print(urllib.parse.unquote(sys.stdin.read()), end="")')
    key=$(printf '%s' "$name" | sha256sum | cut -d' ' -f1)
    if [ -f "$STUB_BARRIER_DB/$key.json" ]; then
      jq '{total_count: 1, check_runs: [.]}' "$STUB_BARRIER_DB/$key.json"
    else
      printf '{"total_count": 0, "check_runs": []}\n'
    fi
    ;;
esac
STUBCURL
chmod +x "$BARBIN/curl"

if lift "$CLIENT_WF" linux "Arrive at the install barrier" "$WORK/bar-arrive.sh" &&
  lift "$CLIENT_WF" linux "Wait at the install barrier" "$WORK/bar-wait.sh"; then
  _path=$PATH
  PATH="$BARBIN:$PATH"
  GH_TOKEN=stub-token
  GITHUB_API_URL=http://127.0.0.1:9
  GITHUB_REPOSITORY=fixture-owner/fixture-repo
  GITHUB_SHA=0000000
  BARRIER_ADAPTER=stub
  # shellcheck disable=SC2089,SC2090 # a JSON array, which the body hands on literally
  ROUTES='["release","source"]'
  BIT_IDS_BARRIER_INTERVAL=1
  BIT_IDS_BARRIER_TIMEOUT=4
  STUB_BARRIER_DB=$BARDB
  RUNNER_TEMP="$WORK/bar-temp"
  mkdir -p "$RUNNER_TEMP"
  # shellcheck disable=SC2090
  export GH_TOKEN GITHUB_API_URL GITHUB_REPOSITORY GITHUB_SHA BARRIER_ADAPTER ROUTES \
    BIT_IDS_BARRIER_INTERVAL BIT_IDS_BARRIER_TIMEOUT STUB_BARRIER_DB RUNNER_TEMP

  arrive() { # run route conclusion
    BARRIER_RUN=$1 ROUTE=$2 INSTALL_CONCLUSION=$3 run_body "$WORK/bar-arrive.sh" default "$TREE"
  }
  verdict() { # run -> the verdict a wait wrote
    rm -f "$RUNNER_TEMP/install-barrier.txt"
    BARRIER_RUN=$1 run_body "$WORK/bar-wait.sh" default "$TREE"
    sed -n 's/^verdict=//p' "$RUNNER_TEMP/install-barrier.txt" 2>/dev/null
  }

  rm -rf "$BARDB"
  arrive 7-1 release success
  arrive 7-1 source success
  _v=$(verdict 7-1)
  if [ "$BODY_RC" = 0 ] && [ "$_v" = passed ]; then
    pass "sh       both routes arrived, so the barrier passes"
  else
    fail "sh       both routes arrived, so the barrier passes (exit $BODY_RC, verdict [$_v])"
  fi

  rm -rf "$BARDB"
  arrive 8-1 release success
  arrive 8-1 source failure
  _v=$(verdict 8-1)
  if [ "$BODY_RC" = 0 ] && [ "$_v" = failed ] &&
    grep -q -x 'failed_routes=source' "$RUNNER_TEMP/install-barrier.txt"; then
    pass "sh       a route that could not install ends the wait at once, named, and fails nothing"
  else
    fail "sh       a route that could not install ends the wait at once, named, and fails nothing (exit $BODY_RC, verdict [$_v])"
  fi

  rm -rf "$BARDB"
  arrive 9-1 release success
  _v=$(verdict 9-1)
  if [ "$BODY_RC" = 0 ] && [ "$_v" = timeout ] &&
    grep -q -x 'missing_routes=source' "$RUNNER_TEMP/install-barrier.txt"; then
    pass "sh       a route that never arrives is a timeout, named, and fails nothing"
  else
    fail "sh       a route that never arrives is a timeout, named, and fails nothing (exit $BODY_RC, verdict [$_v])"
  fi

  # ⛔ AND AN EARLIER DISPATCH OF THE SAME COMMIT IS NOT THIS ONE. Check runs
  # outlive their run, so a barrier keyed on the commit alone would pass on a
  # route that installed an hour ago on a host since destroyed.
  rm -rf "$BARDB"
  arrive 10-1 release success
  arrive 10-1 source success
  _v=$(verdict 10-2)
  if [ "$_v" = timeout ]; then
    pass "sh       a barrier another run attempt left on the commit does not satisfy this one"
  else
    fail "sh       a barrier another run attempt left on the commit does not satisfy this one (verdict [$_v])"
  fi
  PATH=$_path
  RUNNER_TEMP="$TREE/temp"
else
  fail "sh       capture-client.yml has no linux step named Arrive at or Wait at the install barrier"
fi

# -- 4d. the containment, a network namespace with loopback alone -----------
#
# ⛔ THE CAPTURE IS CONTAINED, NOT THE HOST (`CI-08`), and until 2026-10-16
# nothing here ran the containment: `check-workflow` reads that the guard and the
# capture are wrapped, and only a dispatch showed what the wrapper did. These
# cases run both workflows' own *Assert containment* bodies and the wrapper
# itself, for real, which needs `sudo -n` and a kernel that makes a namespace.
# ⚠ A host without them has verified nothing about it, so that is a failed row
# here, as a host without pwsh is above, and never a pass.
# ⚠ The wrapper is `bit-check contained` since 2026-10-16 (`CI-11`), which
# re-runs this same binary as its inner step.
contain() { "$BIT_CHECK" contained "$@"; }
mkdir -p "$WORK/contained-home" || exit 2
if ! sudo -n unshare --net -- true >/dev/null 2>&1; then
  fail "sh       sudo -n cannot make a network namespace on this host, so the containment cases did not run"
else
  # ⛔ THE CONTROL FIRST: the guard refuses this host as it stands. Without a
  # route outside, a guard that passes inside would say nothing about the
  # namespace.
  "$BIT_CHECK" assert-disposable --egress >/dev/null 2>&1
  _rc=$?
  if [ "$_rc" = 1 ]; then
    pass "sh       control: outside any namespace the egress guard refuses this host (exit 1)"
  else
    fail "sh       control: outside any namespace the egress guard answered $_rc, so this host has no route to contain"
  fi
  for _wf in "$CAPTURE_WF" "$CLIENT_WF"; do
    _name=$(basename "$_wf")
    if lift "$_wf" linux "Assert containment" "$WORK/contain-assert.sh"; then
      run_body "$WORK/contain-assert.sh" default "$TREE"
      ended "sh       $_name's Assert containment passes inside its namespace" 0
    else
      fail "sh       $_name has no linux step named Assert containment"
    fi
  done

  # ⛔ BIT_CHECK REACHES THE COMMAND, which capture run 38763293262 refused for
  # want of, and only when it is set, as `${BIT_CHECK:+...}` carried it.
  # shellcheck disable=SC2016
  _got=$(contain -- sh -c 'printf "%s" "${BIT_CHECK-unset}"' 2>&1)
  if [ "$_got" = "$BIT_CHECK" ]; then
    pass "sh       BIT_CHECK is carried into the namespace"
  else
    fail "sh       BIT_CHECK is not carried into the namespace: [$_got]"
  fi
  # shellcheck disable=SC2016
  _got=$(env -u BIT_CHECK "$BIT_CHECK" contained -- sh -c 'printf "%s" "${BIT_CHECK-unset}"' 2>&1)
  if [ "$_got" = unset ]; then
    pass "sh       an unset BIT_CHECK stays unset in the namespace"
  else
    fail "sh       an unset BIT_CHECK arrived in the namespace as [$_got]"
  fi
  # ⚠ The caller's HOME and PATH, not root's: a capture writes its state under
  # HOME. The HOME here is one no account has, so root's cannot pass for it.
  # shellcheck disable=SC2016
  _got=$(HOME="$WORK/contained-home" contain -- sh -c 'printf "%s|%s" "$HOME" "$PATH"' 2>&1)
  if [ "$_got" = "$WORK/contained-home|$PATH" ]; then
    pass "sh       the caller's HOME and PATH are carried into the namespace"
  else
    fail "sh       the namespace has another HOME or PATH: [$_got]"
  fi
  # ⭐ LOOPBACK IS UP, which is how the observer and the product meet.
  _got=$(contain -- ip -o link show lo 2>&1)
  case "$_got" in
    *[\<,]UP[,\>]*) pass "sh       loopback is up inside the namespace" ;;
    *) fail "sh       loopback is not up inside the namespace: [$_got]" ;;
  esac
  # ⚠ The caller's uid, and the command's own exit code.
  _got=$(contain -- id -u 2>&1)
  if [ "$_got" = "$(id -u)" ]; then
    pass "sh       the command runs as the caller, uid $_got"
  else
    fail "sh       the command runs as uid [$_got], not the caller's $(id -u)"
  fi
  contain -- sh -c 'exit 7' >/dev/null 2>&1
  _rc=$?
  if [ "$_rc" = 7 ]; then
    pass "sh       the command's exit code is the wrapper's"
  else
    fail "sh       the command exited 7 and the wrapper answered $_rc"
  fi
  # ⛔ AND ROOT IS DROPPED, which a root caller cannot see through the case
  # above: the inner step, as root in a fresh namespace, hands the command an
  # unprivileged uid and that account's groups.
  _got=$(sudo -n unshare --net -- "$BIT_CHECK" contained --inner 65534 65534 -- sh -c 'printf "%s %s" "$(id -u)" "$(id -g)"' 2>&1)
  if [ "$_got" = "65534 65534" ]; then
    pass "sh       the inner step drops root to the uid and gid it is given"
  else
    fail "sh       the inner step ran the command as [$_got], not 65534 65534"
  fi
  contain -- >/dev/null 2>&1
  _rc=$?
  if [ "$_rc" = 2 ]; then
    pass "sh       no command is could-not-run (exit 2)"
  else
    fail "sh       no command answered $_rc, not 2"
  fi
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

store_probe_guards "$WORK/install.sh" 'rc=0' 'rc'

store_report check-step-bodies/1 cases "$JSON"
