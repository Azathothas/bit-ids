#!/bin/sh
# flight-recorder.sh - push the state of this runner OFF the runner, every few
# seconds, for as long as the job lasts.
#
# ⛔ WHY IT EXISTS. `capture-client` runs 3 to 34 wedged in *Install the client*
# and left NOTHING: no artifact, and no job log at all - the log endpoint answers
# `BlobNotFound` even for the steps that finished. Every instrument tried so far
# wrote to the runner's own disk and waited for an upload step that never ran, so
# every one of them measured nothing. This one does not wait: each snapshot leaves
# the host the moment it is taken, as the output of a check run on the commit.
#
# ⭐ WHAT ITS SILENCE MEANS IS PART OF THE MEASUREMENT. A runner that keeps
# posting is alive and the snapshot says what is stuck; a runner whose posts STOP
# lost contact at that instant, and the last snapshot is the state it lost it in.
#
# ⛔ AND A JOB THAT ENDED IN CONTACT SAYS SO, WHICH IT DID NOT UNTIL 2026-10-14.
# The loop ran to a 3000-second limit no job reaches, so the runner killed the
# recorder at the end of every job and every check run it had opened stayed
# `in_progress` forever: a host that lost contact and one that finished cleanly
# looked the same, and the two contained dispatches of 2026-10-11 still showed
# three of them three days later. ⭐ The job's last step touches
# `<log-dir>/flight-recorder.stop`; the recorder completes its check run in the
# request that posts its final snapshot, reads the answer back, and writes
# `<log-dir>/flight-recorder.done`. A check run still `in_progress` after its job
# ended is now a host that never reached that step, and one that closed still
# shows the state the job ended in.
#
# ⚠ IT READS NOTHING THAT CAN BLOCK IT. `ps` is asked for `stat` and `wchan`,
# which come from `/proc/<pid>/stat` and `wchan`; never for `args`, because
# reading a process's command line takes its memory lock and a reader can hang
# on exactly the process it is trying to describe. Every read is under
# `timeout`, and a read that times out is reported as such.
#
# ⚠ IT IS A DIAGNOSTIC. It never refuses anything and it never changes the host.
# The kernel's ring buffer and the stacks of blocked tasks need root, so they are
# asked through `sudo -n`, which answers nothing rather than prompting where it
# is not available.
#
# Usage (backgrounded by the first step of a job):
#   GH_TOKEN=... sh scripts/ci/flight-recorder.sh <label> <log-dir>
#
# Environment: GH_TOKEN, GITHUB_API_URL, GITHUB_REPOSITORY, GITHUB_SHA, which a
# hosted runner provides except the token, which the step passes.
# BIT_IDS_RECORDER_INTERVAL (seconds, default 15) and BIT_IDS_RECORDER_LIMIT
# (seconds, default 3000) bound how often and how long it posts.
#
# Exit codes: 0 it ran until the job's last step or its limit, 2 it could not
# start.

set -u

[ $# -eq 2 ] || {
  printf 'flight-recorder: takes <label> <log-dir>\n' >&2
  exit 2
}
LABEL=$1
LOGDIR=$2
for _v in GH_TOKEN GITHUB_API_URL GITHUB_REPOSITORY GITHUB_SHA; do
  eval "_val=\${$_v:-}"
  [ -n "$_val" ] || {
    printf 'flight-recorder: %s is not set\n' "$_v" >&2
    exit 2
  }
done
for _tool in curl jq timeout ps; do
  command -v "$_tool" >/dev/null 2>&1 || {
    printf 'flight-recorder: %s is not on this host\n' "$_tool" >&2
    exit 2
  }
done

INTERVAL=${BIT_IDS_RECORDER_INTERVAL:-15}
LIMIT=${BIT_IDS_RECORDER_LIMIT:-3000}
API="$GITHUB_API_URL/repos/$GITHUB_REPOSITORY/check-runs"
SCRATCH=$(mktemp -d) || exit 2

api() { # method url body-file -> response on stdout
  timeout 20 curl -sS --max-time 15 -X "$1" "$2" \
    -H "Authorization: Bearer $GH_TOKEN" \
    -H 'Accept: application/vnd.github+json' \
    --data-binary "@$3" 2>>"$SCRATCH/curl.err"
}

# One read, bounded, labelled, never fatal.
section() { # title seconds command...
  _title=$1
  _secs=$2
  shift 2
  printf '\n### %s\n' "$_title"
  # ⛔ INTO A FILE, THEN THE TAIL: a pipe's status would be the tail's, and a
  # read that timed out would report as one that answered.
  timeout -k 2 "$_secs" "$@" >"$SCRATCH/section.out" 2>&1
  _rc=$?
  tail -n 60 "$SCRATCH/section.out"
  [ "$_rc" = 0 ] || printf '(exit %s%s)\n' "$_rc" "$([ "$_rc" = 124 ] && printf ', TIMED OUT')"
}

snapshot() { # elapsed
  printf 'elapsed %ss, %s\n' "$1" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  section 'uptime and load' 5 cat /proc/loadavg
  section 'memory' 5 free -m
  section 'disk' 5 df -h / /tmp "$LOGDIR"
  # ⚠ D is uninterruptible sleep and Z a zombie: the two states a signal cannot
  # end, so the two a bound cannot end either.
  printf '\n### blocked or zombie tasks\n'
  timeout -k 2 5 ps -eo pid,ppid,pgid,stat,wchan:32,etimes,comm >"$SCRATCH/ps.out" 2>&1 ||
    printf '(ps exit %s)\n' "$?"
  awk 'NR == 1 || $4 ~ /^[DZ]/' "$SCRATCH/ps.out"
  # ⭐ WHERE IN THE KERNEL EACH BLOCKED TASK IS WAITING, which is the one fact a
  # process table cannot give and the one that names a cause.
  awk 'NR > 1 && $4 ~ /^D/ { print $1 }' "$SCRATCH/ps.out" >"$SCRATCH/blocked.pids"
  while read -r _pid; do
    section "kernel stack of $_pid" 5 sudo -n cat "/proc/$_pid/stack"
  done <"$SCRATCH/blocked.pids"
  section 'the newest 40 tasks' 5 sh -c 'ps -eo pid,ppid,pgid,stat,wchan:32,etimes,comm --sort=etimes | head -41'
  section 'kernel ring buffer' 5 sudo -n dmesg --ctime
  for _f in "$LOGDIR"/install-step.log "$LOGDIR"/install-*/step.log "$LOGDIR"/install-*/watchdog.log "$LOGDIR"/install-*/install.log; do
    [ -f "$_f" ] || continue
    section "tail of $_f" 5 tail -n 30 "$_f"
  done
  # ⭐ THE NEWEST FILES ANYWHERE UNDER THE LOG DIRECTORY, which is where a
  # traced install writes as it goes: the last line that leaves the host is the
  # last thing the install did before it stopped being able to say so.
  # shellcheck disable=SC2016
  section 'newest files' 5 sh -c 'find "$1" -type f -newermt "-120 seconds" -printf "%TT %s %p\n" 2>/dev/null | sort | tail -n 15' sh "$LOGDIR"
  find "$LOGDIR" -type f -newermt "-60 seconds" \( -name '*.log' -o -name '*.err' -o -name '*.txt' \) -printf '%T@ %p\n' 2>/dev/null |
    sort -n | tail -n 4 | cut -d' ' -f2- >"$SCRATCH/newest.list"
  while read -r _f; do
    section "tail of $_f" 5 tail -n 25 "$_f"
  done <"$SCRATCH/newest.list"
  section 'route to the API' 10 curl -sS -o /dev/null -w '%{http_code} in %{time_total}s\n' --max-time 8 "$GITHUB_API_URL/zen"
}

# ⚠ A DRY RUN PRINTS ONE SNAPSHOT AND POSTS NOTHING, which is how this is driven
# anywhere but a runner.
if [ "${BIT_IDS_RECORDER_DRYRUN:-}" = 1 ]; then
  snapshot 0
  exit 0
fi

# ⚠ The first post creates the check run; every later one updates it. A token
# that cannot create one is reported once and the recorder stops, because a
# recorder that cannot post records nothing.
jq -n --arg name "flight-recorder ($LABEL)" --arg sha "$GITHUB_SHA" \
  '{name: $name, head_sha: $sha, status: "in_progress", output: {title: "starting", summary: "starting"}}' \
  >"$SCRATCH/create.json"
ID=$(api POST "$API" "$SCRATCH/create.json" | jq -r '.id // empty')
[ -n "$ID" ] || {
  printf 'flight-recorder: could not create a check run: %s\n' "$(cat "$SCRATCH/curl.err" 2>/dev/null)" >&2
  exit 2
}
printf 'flight-recorder: posting to check run %s every %ss\n' "$ID" "$INTERVAL"

STOPFILE="$LOGDIR/flight-recorder.stop"
DONEFILE="$LOGDIR/flight-recorder.done"

# One snapshot, as the body of a check run update. A third argument of
# `completed` closes the run in the same request.
update() { # title-prefix elapsed [completed]
  snapshot "$2" >"$SCRATCH/snap.txt" 2>&1
  # ⚠ The output text is capped well under the API's 65535 characters, keeping
  # the TAIL, because the newest lines are the ones a hang is about.
  tail -c 60000 "$SCRATCH/snap.txt" >"$SCRATCH/text.txt"
  jq -n --rawfile text "$SCRATCH/text.txt" --arg title "$1 at +${2}s" --arg close "${3:-}" \
    '{output: {title: $title, summary: $title, text: $text}}
      + (if $close == "completed" then {status: "completed", conclusion: "neutral"} else {} end)' \
    >"$SCRATCH/update.json"
}

# One snapshot, posted as the check run's output.
post() { # title-prefix elapsed
  update "$1" "$2"
  api PATCH "$API/$ID" "$SCRATCH/update.json" >/dev/null
}

START=$(date +%s)
N=0
ENDING="the limit of ${LIMIT}s was reached"
while :; do
  NOW=$(date +%s)
  ELAPSED=$((NOW - START))
  [ "$ELAPSED" -lt "$LIMIT" ] || break
  if [ -e "$STOPFILE" ]; then
    ENDING="the job reached its last step"
    break
  fi
  N=$((N + 1))
  post "snapshot $N" "$ELAPSED"
  # ⚠ ONE SECOND AT A TIME, so the stop is seen within a second rather than an
  # interval: the job's last step waits on this, and a host torn down while it
  # waited would leave the check run open over a job that ended in contact.
  _slept=0
  while [ "$_slept" -lt "$INTERVAL" ] && [ ! -e "$STOPFILE" ]; do
    sleep 1
    _slept=$((_slept + 1))
  done
done

# ⭐ THE LAST SNAPSHOT AND THE COMPLETION, AND THE COMPLETION IS READ BACK. A
# PATCH that was refused leaves the run `in_progress`, which is the silence this
# file says means lost contact; so the answer is what the done file reports, and
# a run the API did not confirm says that rather than claiming it closed.
#
# ⛔ ONE REQUEST, BECAUSE TWO ERASED THE SNAPSHOT. Until 2026-10-14 the last
# snapshot went up first and the completion after it, carrying an `output` with
# a title and no `text`; measured on `capture-client` run 35, both check runs that
# closed cleanly showed that title over an empty text, and the source lane's last
# state - the compile it failed in - was gone from the one place a dispatch is
# read. A check run's output is replaced by the update that carries one.
update "$ENDING" "$(($(date +%s) - START))" completed
api PATCH "$API/$ID" "$SCRATCH/update.json" >"$SCRATCH/answer.json"
ANSWER=$(jq -r '.status // empty' "$SCRATCH/answer.json" 2>/dev/null)
# ⚠ AND A SNAPSHOT THE API REFUSES MUST NOT KEEP THE RUN OPEN. One request
# carries both now, so a text the API would not take would refuse the
# completion with it. An answer that is not `completed` is asked once more with
# no `output`, which leaves the last snapshot the run accepted. ⚠ Only when the
# API answered: a request that timed out says nothing about its text, and a
# second one would spend the wait the job's last step gives this.
if [ "$ANSWER" != completed ] && [ -s "$SCRATCH/answer.json" ]; then
  jq -n '{status: "completed", conclusion: "neutral"}' >"$SCRATCH/done.json"
  ANSWER=$(api PATCH "$API/$ID" "$SCRATCH/done.json" | jq -r '.status // empty' 2>/dev/null)
fi
if [ "$ANSWER" = completed ]; then
  printf 'completed: %s\n' "$ENDING" >"$DONEFILE"
else
  printf 'not confirmed: the check run answered [%s]\n' "$ANSWER" >"$DONEFILE"
fi
cat "$DONEFILE"
exit 0
