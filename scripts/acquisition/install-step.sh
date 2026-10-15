#!/bin/sh
# install-step.sh - what *Install the client* does, in a file, so the step's own
# process can be bounded from outside it.
#
# ⛔ EVERY BOUND TRIED SO FAR HAS FAILED, AND EACH FOR ITS OWN REASON. `timeout`
# inside `install-client` ends its own child, so it cannot end a shell blocked in
# a command substitution around that child. `timeout-minutes` is the runner's and
# was measured on run 6 not to end the step at all. A watchdog loop in the step's
# own shell was measured on run 9 not to fire either - the deadline passed by
# three minutes - which says the step's shell is not the thing running.
#
# ⭐ SO THE BOUND MOVES OUTSIDE THE SHELL ENTIRELY. The workflow runs this file
# under `timeout`, so the step's own process is the one being ended, and nothing
# inside it has to be reachable for that to work.
#
# ⚠ THE POINT IS NOT THE INSTALL. It is that the step ENDS: a job whose runner is
# killed leaves no log and no artifact - measured on runs 7, 8 and 9 - so an
# aria2 lane that hangs teaches nothing at all unless something makes the job
# reach its uploads.
#
# ⛔ AND NOTHING IT SPAWNS INHERITS THE STEP'S OUTPUT. A runner ends a step when
# the command has exited AND the step's output pipe has reached end of file, so
# the route's output goes to a file and is printed afterwards.
#
# Usage:
#   ADAPTER=<path> ROUTE=<name> RUNNER_TEMP=<dir> sh scripts/acquisition/install-step.sh
#
# Exit codes: 0 installed, 1 the route or a guard refused, 2 could not run,
# 124 from a `timeout` that fired. ⚠ TWO DIFFERENT ONES NOW PRODUCE 124, and the
# watchdog log separates them: this file's own PRIVILEGED bound around the
# install, which ends the root tree and lets this file print what it has; and the
# caller's bound around this file, which means even that did not answer.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

: "${ADAPTER:?install-step: ADAPTER is required}"
: "${ROUTE:?install-step: ROUTE is required}"
: "${RUNNER_TEMP:?install-step: RUNNER_TEMP is required}"

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

WORKDIR="$RUNNER_TEMP/install-$ROUTE"
RECORD="$RUNNER_TEMP/install-$ROUTE.txt"
mkdir -p "$WORKDIR" || {
  printf 'install-step: cannot create %s\n' "$WORKDIR" >&2
  exit 2
}

# ⚠ THE URL TRAVELS THROUGH A FILE RATHER THAN THROUGH GITHUB_ENV, so the value
# the install used is on disk beside the record that names it.
if [ "$ROUTE" = release ]; then
  [ -f "$RUNNER_TEMP/release/url.txt" ] || {
    printf 'install-step: the release lane has no resolved artifact\n' >&2
    exit 2
  }
  BIT_IDS_RELEASE_URL=$(cat "$RUNNER_TEMP/release/url.txt")
  export BIT_IDS_RELEASE_URL
  printf 'the release route will fetch %s\n' "$BIT_IDS_RELEASE_URL"

  # ⛔ THE WHOLE DIGEST DISPOSITION TRAVELS AS THE RESOLUTION ITSELF, and the
  # installer is what reads it. ⚠ This block used to export a document and an
  # asset name and each adapter chose between them; four adapters choosing
  # between four dispositions is four copies of one decision, which is the
  # one-gated-door shape `docs/methodology/reviews.md` names. `ACQ-06`.
  #
  # ⚠ `bit-check resolve-release` did the resolving while the host still had a route
  # off itself, which is the containment order; this step only points the route
  # at what is already on disk.
  BIT_IDS_RELEASE_RESOLUTION="$RUNNER_TEMP/release/resolution.txt"
  [ -f "$BIT_IDS_RELEASE_RESOLUTION" ] || {
    printf 'install-step: the release lane has no resolution to take a digest from\n' >&2
    exit 2
  }
  export BIT_IDS_RELEASE_RESOLUTION
  printf 'the release route takes its digest disposition from %s (%s)\n' \
    "$BIT_IDS_RELEASE_RESOLUTION" \
    "$(sed -n 's/^digest_source=//p' "$BIT_IDS_RELEASE_RESOLUTION")"
fi

# ⛔ THE SOURCE ROUTE TAKES ITS TAG FROM ITS OWN RESOLUTION, AND THIS BLOCK USED
# TO TAKE IT FROM THE RELEASE LANE'S. The paragraph that stood here argued that
# one resolver answering both lanes is how absolute 4's same-version rule is
# "made true rather than checked afterwards". ⛔ That is precisely what
# `E-ACQ-07` refuses: two routes sharing a resolver are one route, and
# `assemble-capture` measured it on capture-client run 14 - both lanes'
# resolution records differ only in their timestamps.
#
# ⭐ ABSOLUTE 4 ASKS FOR THE OPPOSITE OF WHAT THAT PARAGRAPH DID. Version
# equality is checked AFTER installation, on what each build reported when
# asked. Two independent resolutions landing on two versions is a vendor that
# moved between two reads, and catching it is the correct outcome; making them
# equal beforehand manufactures the agreement this project exists to measure.
if [ "$ROUTE" = source ]; then
  [ -f "$RUNNER_TEMP/source/resolution.txt" ] || {
    printf 'install-step: the source lane has no resolution of its own to take a tag from\n' >&2
    exit 2
  }
  BIT_IDS_SOURCE_TAG=$(sed -n 's/^selected_tag=//p' "$RUNNER_TEMP/source/resolution.txt")
  [ -n "$BIT_IDS_SOURCE_TAG" ] || {
    printf 'install-step: the source resolution record names no selected_tag\n' >&2
    exit 2
  }
  export BIT_IDS_SOURCE_TAG
  printf 'the source route will build %s, resolved from this repository refs\n' \
    "$BIT_IDS_SOURCE_TAG"
fi

# ⛔ THE INSTALL RUNS IN THE BACKGROUND SO THIS SHELL CAN RECORD WHAT THE HOST IS
# DOING WHILE IT RUNS. ⚠ That loop is NOT the bound any more - run 9 measured it
# not firing - it is the evidence: `stat` beside `etimes` is what separates a
# command that is slow from one that is stopped, and eight runs of step timings
# could not tell those apart.
# ⛔ THE REDIRECTION IS THIS SHELL'S, WHICH IS THE POINT RATHER THAN AN
# OVERSIGHT: the descriptor the install inherits must be that FILE and not this
# step's output pipe. A runner ends a step when the command has exited AND the
# step's pipe has reached end of file, so anything the route leaves behind
# holding that pipe keeps the step open whatever any bound does.
#
# -- ⛔ AND THERE IS NO `sudo` HERE ANY MORE. `ACQ-06`. -----------------------
#
# Every bound this entry measured failing was issued by an UNPRIVILEGED process
# at a process tree running as ROOT, and the kernel refuses that signal:
# measured on 2026-09-16, a uid-1001 `timeout -k 2 5` around `sudo -E sh -c
# 'sleep 120'` exited 124 on schedule and left the root `sleep` alive with PPID 1
# - orphaned, unkillable by its own bound, and still running after the KILL grace
# had passed. The repair then was to put the bound INSIDE the `sudo`.
#
# ⭐ THE PRIVILEGE IS GONE INSTEAD, WHICH REMOVES THE ASYMMETRY RATHER THAN
# WORKING AROUND IT. The install writes into a prefix this user already owns -
# `bit-check install-rootless --prefix` is the one derivation - so every process in this
# subtree runs as the same uid as this shell, and a bound issued here can reach
# every one of them. ⚠ That is not another bound: it is the condition under which
# the bounds that are already here can work at all.
#
# ⚠ THE RELATION IS WHAT MATTERS, not the number. This must be LARGER than
# `install-client`'s own inner bounds, so a slow route is refused by that script
# with a message naming its timeout rather than killed here with nothing to read;
# and SMALLER than the workflow step's own `timeout`, so this one is reached
# first. The workflow states both.
STEP_SECONDS=${BIT_IDS_STEP_TIMEOUT:-1020}
STEP_KILL_AFTER=${BIT_IDS_KILL_AFTER:-20}
command -v timeout >/dev/null 2>&1 || {
  printf 'install-step: timeout is not on this host\n' >&2
  exit 2
}
# ⛔ `setsid` PUTS THE INSTALL IN A SESSION OF ITS OWN, so it shares no process
# group, no session and no terminal with this shell. ⚠ It is NOT here to make the
# bound work - the bound above is what does that - it is here so that this shell
# owes the install NOTHING: nothing to signal, nothing to reap, and no descriptor
# in common. A detached tree cannot hold the step open however long it runs.
# ⚠ `setsid` is util-linux and ships on every runner image this project uses; a
# host without it is a host this file refuses rather than silently attaches to.
command -v setsid >/dev/null 2>&1 || {
  printf 'install-step: setsid is not on this host\n' >&2
  exit 2
}
setsid timeout -k "$STEP_KILL_AFTER" "$STEP_SECONDS" \
  sh "$ROOT/scripts/acquisition/install-client.sh" \
  --adapter "$ADAPTER" \
  --route "$ROUTE" \
  --workdir "$WORKDIR" \
  --record "$RECORD" \
  >"$WORKDIR/step.log" 2>&1 </dev/null &
INSTALL_PID=$!

# ⛔ THE LOOP HAS A DEADLINE OF ITS OWN AND THAT IS THE WHOLE REPAIR. It used to
# spin until the install died and then `wait` for it, which makes this shell's
# exit depend on the install's - so an install that never ends is a step that
# never ends, whatever any bound does. ⚠ TEN bounds have now been measured not to
# end this step, on `capture-client` runs 3 to 10, 21 to 23 and 24 to 25; the
# eleventh would fail the same way, because a step ends when its command has
# exited AND its pipe has reached end of file, and a bound only ever acts on the
# first.
# ⭐ So this shell stops waiting on its own schedule. The install may still be
# running when it does, and that is deliberate: the runner is destroyed after the
# job, and a finished step with a timeline in the artifact is worth more than a
# tidy host nobody can read.
INTERVAL=${BIT_IDS_INSTALL_INTERVAL:-5}
WAITED=0
while kill -0 "$INSTALL_PID" 2>/dev/null; do
  [ "$WAITED" -lt "$STEP_SECONDS" ] || break
  sleep "$INTERVAL"
  WAITED=$((WAITED + INTERVAL))
  # ⛔ `ps` IS BOUNDED, AND IT WAS THE LAST UNBOUNDED COMMAND IN THIS SHELL.
  # `ps -e` reads `/proc` for every process on the host, including
  # `/proc/<pid>/cmdline` for `args`, and a read of `/proc` for a task wedged in
  # the kernel can block in exactly the way this loop exists to observe. ⚠ So the
  # instrument could hang on the condition it was written to record, and the loop
  # would never reach the deadline two lines above - which is precisely what
  # `capture-client` run 26 looked like from outside.
  # ⛔ THIS ENTRY'S OWN RECORD SAID THE LOOP "ONLY SLEEPS AND COMPARES TWO
  # INTEGERS", WRITTEN ON 2026-09-16 WITHOUT RE-READING IT. It also runs this,
  # which is the one thing in it that can block. The claim is corrected where it
  # was made rather than only here.
  # ⚠ A `ps` that times out leaves its sample headed and empty, which is itself
  # the measurement: a timeline that stops having process tables while still
  # having timestamps says the host stopped answering, not that the loop stopped.
  {
    printf '## +%ss %s\n' "$WAITED" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    timeout "${BIT_IDS_PS_TIMEOUT:-20}" \
      ps -eo pid,ppid,pgid,stat,etimes,comm,args 2>&1 ||
      printf '## ps did not answer within %ss\n' "${BIT_IDS_PS_TIMEOUT:-20}"
  } >>"$WORKDIR/watchdog.log"
done

# ⛔ AND IT NEVER `wait`s ON A PROCESS THAT MAY NOT DIE. `wait` is the one call
# here with no bound of its own, so it is reached only once the install is known
# to be gone. ⚠ 124 is coreutils' vocabulary for "it never answered" and this
# reports the same thing in the same number, from the other side of the bound.
if kill -0 "$INSTALL_PID" 2>/dev/null; then
  RC=124
  printf 'install-step: the install was still running after %ss; it is hung, not slow\n' \
    "$WAITED" >&2
else
  wait "$INSTALL_PID" && RC=0 || RC=$?
fi
# ⛔ AND WHAT THE INSTALL LEFT IN ITS SESSION IS NAMED AND ENDED. Nested
# `timeout`s each start a process group of their own, so a bound reaches the
# group it started and not the one beneath it: measured on 2026-09-16, a
# `sleep 600` outlived every bound with PPID 1. `CI-08`. ⭐ Every one of them
# is still in the SESSION `setsid` opened, because a process group cannot leave
# its session, so the session is the whole population. Its id is the `setsid`
# child's own pid: a background job here is not a process-group leader, so
# `setsid` does not fork. ⚠ If it ever did, no session would carry that id and
# the sweep would find nothing, which is the safe failure. It cannot reach this
# shell's own session: that id belongs to an ancestor of this shell, and
# `INSTALL_PID` is a child of it.
SWEPT=0
timeout 20 ps -eo pid=,sid=,args= 2>/dev/null |
  awk -v s="$INSTALL_PID" '$2 == s' >"$WORKDIR/orphans.log"
if [ -s "$WORKDIR/orphans.log" ]; then
  SWEPT=$(wc -l <"$WORKDIR/orphans.log" | tr -d ' ')
  # shellcheck disable=SC2046
  kill -TERM $(awk '{ print $1 }' "$WORKDIR/orphans.log") 2>/dev/null
  sleep 2
  # ⚠ The session is listed again rather than the first list reused, so a pid
  # that exited and was handed to something else in those two seconds is not
  # the one that gets KILL.
  # shellcheck disable=SC2046
  kill -KILL $(timeout 20 ps -eo pid=,sid= 2>/dev/null |
    awk -v s="$INSTALL_PID" '$2 == s { print $1 }') 2>/dev/null
  printf 'install-step: ended %s process(es) the install left in its session; orphans.log names them\n' \
    "$SWEPT" >&2
fi
cat "$WORKDIR/step.log"

# ⭐ AND WHATEVER STILL HOLDS THAT FILE IS NAMED. ⚠ Bounded, because it walks
# every process's descriptors and is the one thing added to this step that could
# itself be slow; a diagnostic that held the step open would be the failure it
# exists to explain, wearing its own name.
#
# ⛔ AND IT NO LONGER RUNS UNDER `sudo`, BECAUSE THE INSTALL NO LONGER DOES.
# The reason it was privileged is that an unprivileged reader of `/proc` reports
# nobody holding a file that ROOT processes are holding - which was a true fact
# about a privileged install. ⭐ Every process this step starts now runs as this
# uid, so this reader sees all of them, and the report stops depending on a
# privilege the step exists to do without. ⚠ A holder belonging to some OTHER
# user is outside what it can see, and that is a narrower claim than before
# rather than the same one: `TODO/acquisition.md` carries it under `ACQ-06`.
#
# ⛔ AND THIS SHELL'S OWN STDOUT IS ASKED ABOUT TOO, WHICH IT NEVER WAS. The
# report was pointed at `step.log` alone, so `holders.log` answered `0 holder(s)`
# on every green run - truthfully, and about the wrong descriptor. The file that
# keeps a STEP open is the step's own output, and nothing had ever asked who held
# it. ⚠ `readlink` is resolved HERE rather than inside the reporter, because
# `/proc/self/fd/1` there would name the reporter's own stdout. ⚠ When it names a
# pipe rather than a file the reporter says so, which is itself the answer: it
# means this shell is still talking down a pipe somebody may be holding.
# ⛔ fd 1 IS DUPLICATED FIRST, BECAUSE INSIDE `$( )` fd 1 IS THE SUBSTITUTION'S
# OWN PIPE. Reading `/proc/self/fd/1` there names that pipe and never this
# shell's real output - measured on 2026-09-16, when this line reported a pipe
# over a step whose stdout was a file. ⚠ It is the same confusion between a pipe
# and a destination that this whole entry is about, reproduced in the diagnostic
# written to explain it.
exec 9>&1
STEP_STDOUT=$(readlink -f /proc/self/fd/9 2>/dev/null) || STEP_STDOUT=""
exec 9>&-
HELD=0
timeout 60 sh "$ROOT/scripts/ci/report-holders.sh" \
  "$WORKDIR/step.log" ${STEP_STDOUT:+"$STEP_STDOUT"} \
  >>"$WORKDIR/holders.log" 2>&1 || HELD=$?
printf 'the holder report exited %s\n' "$HELD"

[ "$RC" = 0 ] || exit "$RC"
cat "$RECORD"
exit 0
