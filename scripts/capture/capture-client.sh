#!/bin/sh
# capture-client.sh - a capture of an INSTALLED BUILD, on a host that has
# already been claimed and already had its route off the machine deleted.
#
# ⛔ THE DIFFERENCE FROM capture-run.sh IS THE DRIVER, AND IT IS THE WHOLE
# PRODUCT. `capture-run` drives this project's observer with `curl` and writes
# `kind=fixture`, `measured_build=none`, `stock_client=false`, because nothing
# is installed. Here an ADAPTER stands for one target: it answers what the
# installed build's exact version is, and it starts that build on the torrent
# the observer generated. Those two facts are what turn an attestation about a
# containment into an attestation about a build.
#
# ⛔ IT INSTALLS NOTHING AND BUILDS NOTHING. By the time this runs there is no
# network: the workflow deleted the default route before calling it. Acquisition
# and installation happen in an earlier step, while the network still exists,
# through `scripts/acquisition/install-client.sh`. A capture that discovered a
# missing package here would have to restore egress to fix it, on the host that
# exists to have none.
#
# -- ⛔ WHAT MAKES THIS A MEASUREMENT OF A BUILD -----------------------------
#
# Two things, and both are checked rather than declared:
#
#  1. The adapter answers `version` from the INSTALLED EXECUTABLE. A version
#     read from a filename or a package index is not the build speaking, which
#     is the rule `E-ACQ-10` states for a record and this restates for a run.
#  2. An announce arrived carrying a peer ID THIS OBSERVER DID NOT GENERATE,
#     and the raw transcript on disk carries those exact bytes. Everything else
#     here is satisfied by a bundle of empty artifacts that verify against their
#     own empty digests, or by the observer announcing to itself.
#  3. A SECOND CONNECTOR READ THE SAME BYTES AND SAID WHAT IT SAW. ⛔ Absolute 2
#     asks for the Rust observer plus at least one independent connector, and
#     `E-CAP-01` refuses a record declaring one as an INVALID DOCUMENT rather
#     than as an unpublishable one - so a capture without this step produces
#     evidence that cannot become a record however complete it looks. That is
#     not a reading: it is what refused run 14's four artifacts.
#
# ⚠ The adapter DECLARES what it is, and the attestation copies that rather
# than assuming. A stub adapter yields `stock_client=false`, because a run
# driven by a stand-in is not a run driven by a product, and a field that said
# otherwise would be the one sentence in the record nothing backs.
#
# Usage:
#   sh scripts/capture/capture-client.sh --run-id <id> --out <dir>
#                                        --observer <path> --adapter <path>
#                                        --connector <path>
#                                        [--seconds <n>] [--sessions <n>]
#                                        [--peer-port <n>] [--route-table <path>]
#
# Exit codes: 0 the capture ran and its evidence verifies, 1 a guard refuses,
# 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

RUN_ID=""
OUT=""
OBSERVER=""
ADAPTER=""
CONNECTOR=""
SECONDS_TO_SERVE=25
PEER_PORT=""
ROUTE_TABLE=""
# ⛔ TWO BY DEFAULT, because one session cannot separate a value the build stores
# from one it regenerates. `SamplingPlan::restarts` says so in as many words:
# restarting the process is the only thing that distinguishes the two, and
# `Lifetime::PerSession` is the answer only a second session can reach.
SESSIONS=2

usage() {
  awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
}

# ⛔ ONE MESSAGE PREFIX, so a refusal is attributable to this script rather than
# to the guard or the adapter it delegated to. Each of those prints its own name
# as well, which is how a log says which of the three refused.
refuse() { # message
  printf 'capture-client: %s\n' "$1" >&2
  exit 1
}

cannot() { # message
  printf 'capture-client: %s\n' "$1" >&2
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --run-id)
      [ $# -ge 2 ] || cannot "--run-id takes a value"
      RUN_ID="$2"
      shift
      ;;
    --out)
      [ $# -ge 2 ] || cannot "--out takes a value"
      OUT="$2"
      shift
      ;;
    --observer)
      [ $# -ge 2 ] || cannot "--observer takes a value"
      OBSERVER="$2"
      shift
      ;;
    --adapter)
      [ $# -ge 2 ] || cannot "--adapter takes a value"
      ADAPTER="$2"
      shift
      ;;
    --connector)
      [ $# -ge 2 ] || cannot "--connector takes a value"
      CONNECTOR="$2"
      shift
      ;;
    --seconds)
      [ $# -ge 2 ] || cannot "--seconds takes a value"
      SECONDS_TO_SERVE="$2"
      shift
      ;;
    --sessions)
      [ $# -ge 2 ] || cannot "--sessions takes a value"
      SESSIONS="$2"
      shift
      ;;
    --peer-port)
      [ $# -ge 2 ] || cannot "--peer-port takes a value"
      PEER_PORT="$2"
      shift
      ;;
    --route-table)
      [ $# -ge 2 ] || cannot "--route-table takes a value"
      ROUTE_TABLE="$2"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      printf 'capture-client: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

[ -n "$RUN_ID" ] || cannot "--run-id is required"
[ -n "$OUT" ] || cannot "--out is required"
[ -n "$OBSERVER" ] || cannot "--observer is required"
[ -n "$ADAPTER" ] || cannot "--adapter is required"
# ⛔ REQUIRED, WITH NO DEFAULT AND NO FALLBACK. Absolute 2 asks for the observer
# plus at least one independent connector, and `E-CAP-01` refuses a record
# declaring one at the VALIDITY gate - so a capture that ran without a second
# connector produces evidence that cannot become a record however good it is.
# ⚠ THE REJECTED ALTERNATIVE IS THE EXPENSIVE ONE: a capture that warned and
# carried on would look exactly like a green run, and this project has already
# spent a dispatch cycle learning that a bundle's gaps are invisible until
# something tries to assemble it.
[ -n "$CONNECTOR" ] || cannot "--connector is required; a one-connector capture cannot become a record"

# ⚠ The same slug rule the guard applies, restated here rather than delegated,
# because this value is a key in a document before the guard is ever consulted.
case "$RUN_ID" in
  '' | *[!a-z0-9-]*) cannot "run id must be lowercase a-z0-9-: $RUN_ID" ;;
esac

case "$SECONDS_TO_SERVE" in
  '' | *[!0-9]*) cannot "--seconds must be a whole number: $SECONDS_TO_SERVE" ;;
esac

case "$SESSIONS" in
  '' | *[!0-9]*) cannot "--sessions must be a whole number: $SESSIONS" ;;
esac
# ⚠ At least one, because zero sessions is a capture that starts nothing, and a
# run that measured no build is a refusal rather than an empty result.
[ "$SESSIONS" -ge 1 ] || cannot "--sessions must be at least 1: $SESSIONS"

if [ -n "$PEER_PORT" ]; then
  case "$PEER_PORT" in
    '' | *[!0-9]*) cannot "--peer-port must be a whole number: $PEER_PORT" ;;
  esac
fi

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)
GUARD="$ROOT/scripts/acquisition/assert-disposable.sh"
[ -f "$GUARD" ] || cannot "$GUARD is not present"

for tool in sha256sum awk od grep timeout; do
  command -v "$tool" >/dev/null 2>&1 || cannot "$tool not found"
done

# ⚠ THE CONNECTOR IS RUN WITH ITS INTERPRETER, THE WAY THE ADAPTER IS. This tree
# keeps its scripts unexecutable and invokes them as `sh <path>`, so a connector
# that depended on an executable bit would stop working on a checkout that lost
# one - and `git ls-files -s` says three of the seventy-seven scripts here carry
# one, which makes it the exception rather than the convention.
command -v python3 >/dev/null 2>&1 ||
  cannot "python3 not found; the second connector is deliberately not this project's Rust"

# ⛔ EVERY ADAPTER CALL IS BOUNDED HERE AS WELL AS IN install-client, and for the
# same measured reason: an adapter shells out to a product this project did not
# write, and a product that waits on a prompt waits forever. ⚠ Under containment
# it is worse than during an install, because the observer's own deadline expires
# meanwhile and the run reports "the build announced nothing" over a build that
# was never asked anything.
#
# ⭐ AND STDIN IS /dev/null on every one, so a prompt gets end-of-file.
ADAPTER_SECONDS=${BIT_IDS_ADAPTER_TIMEOUT:-120}
# ⛔ -k, because `timeout` sends TERM and then waits: a child that blocks or
# ignores it is never killed and the bound reports nothing.
ADAPTER_KILL_AFTER=${BIT_IDS_KILL_AFTER:-20}

# ⛔ THE BUILD RAN BEFORE THE NETWORK WAS CUT, OR IT DID NOT RUN. `could not
# run` rather than a guard refusing: nothing about the host is wrong.
[ -x "$OBSERVER" ] ||
  cannot "the observer $OBSERVER is not executable; it is built before the route is deleted, never here"

[ -f "$ADAPTER" ] ||
  cannot "the adapter $ADAPTER is not present"

[ -f "$CONNECTOR" ] ||
  cannot "the connector $CONNECTOR is not present"

# ⛔ A SECOND CAPTURE MUST NOT WRITE OVER THE FIRST ONE'S EVIDENCE.
[ -e "$OUT" ] && refuse "$OUT already exists; a capture never writes into another run's evidence"

# -- the host, as this run finds it -------------------------------------------
#
# ⛔ THE MARKER'S PATH IS ASKED FOR, NEVER COMPOSED. `--marker` is the one
# derivation; a second spelling would go on reading the old location the day the
# state directory moves and report an unclaimed host as claimed.
MARKER=$(sh "$GUARD" --marker) || cannot "the guard could not report its marker path"
[ -n "$MARKER" ] || cannot "the guard reported an empty marker path"

[ -f "$MARKER" ] ||
  refuse "the host was never claimed; $MARKER does not exist"

CLAIMED_RUN=$(awk -F= '$1 == "run" { print $2; exit }' "$MARKER")
[ "$CLAIMED_RUN" = "$RUN_ID" ] ||
  refuse "the host is claimed by run [$CLAIMED_RUN], not [$RUN_ID]"

# ⛔ AND THE ROUTE IS READ AGAIN AT THE MOMENT OF CAPTURE. What this adds over
# the workflow's own step is that nothing between the two put the route back.
if [ -n "$ROUTE_TABLE" ]; then
  sh "$GUARD" --egress "$ROUTE_TABLE"
  EGRESS_RC=$?
  EGRESS_SOURCE="$ROUTE_TABLE"
else
  sh "$GUARD" --egress
  EGRESS_RC=$?
  EGRESS_SOURCE=/proc/net/route
fi
case "$EGRESS_RC" in
  0) : ;;
  1) refuse "egress is open; the default route is still there and a capture would leak" ;;
  *) cannot "egress could not be established from $EGRESS_SOURCE" ;;
esac

FINGERPRINT=$(sh "$GUARD" --fingerprint) || cannot "the host fingerprint could not be read"

mkdir -p "$OUT" || cannot "cannot create $OUT"
OUT=$(CDPATH='' cd -- "$OUT" && pwd)

# ⭐ THE CLAIM ITSELF GOES WITH THE EVIDENCE. It carries when the host was
# claimed and the host facts a run manifest needs, taken in the job's own
# environment; `assemble-capture` reads them from here. `CI-09`.
cp "$MARKER" "$OUT/host-claim.txt" ||
  cannot "cannot keep the host claim beside the attestation"
BUNDLE="$OUT/bundle"
LOG="$OUT/observer.log"
SUMS="$OUT/SHA256SUMS"
ATTEST="$OUT/attestation.txt"
WORKDIR="$OUT/client"
TORRENT="$OUT/run.torrent"
mkdir -p "$WORKDIR" || cannot "cannot create $WORKDIR"

# -- what the adapter says it is ----------------------------------------------
#
# ⛔ ASKED BEFORE ANYTHING IS STARTED, so a run whose adapter cannot describe
# itself never launches a process at all. `describe` prints key=value lines: the
# target it drives and whether it is a stock product or a stand-in.
ADAPTER_DESC="$OUT/adapter.describe"
timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" sh "$ADAPTER" describe </dev/null \
  >"$ADAPTER_DESC" 2>"$OUT/adapter.err" ||
  cannot "the adapter could not describe itself"

TARGET=$(awk -F= '$1 == "target" { sub(/^[^=]*=/, ""); print; exit }' "$ADAPTER_DESC")
ADAPTER_KIND=$(awk -F= '$1 == "kind" { sub(/^[^=]*=/, ""); print; exit }' "$ADAPTER_DESC")
[ -n "$TARGET" ] || cannot "the adapter named no target"
case "$ADAPTER_KIND" in
  stock | stub) : ;;
  '') cannot "the adapter declared no kind" ;;
  *) cannot "the adapter declared an unknown kind: $ADAPTER_KIND" ;;
esac

# ⛔ THE VERSION COMES OUT OF THE INSTALLED EXECUTABLE. An adapter that cannot
# ask it is `could not run`: there is no fixture fallback here, because a run
# reporting `measured_build=none` beside `kind=client` would be the record
# claiming a build it never identified.
# ⛔ AND IT GOES TO A FILE, NOT THROUGH A SUBSTITUTION. This read
# `MEASURED_BUILD=$(timeout -k N M sh "$ADAPTER" version)` until 2026-09-16,
# which is a bound that
# cannot bound: a command substitution ends when its PIPE reaches end of file
# rather than when the child exits, so one process the product leaves behind
# holds this step open with the bound sitting there having fired correctly.
# ⚠ The `describe` call twenty lines above already had the right shape, which is
# what makes this the one-gated-door defect rather than an unknown: the same
# question, asked two ways, in one file.
ADAPTER_VERSION="$OUT/adapter.version"
timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" sh "$ADAPTER" version \
  </dev/null >"$ADAPTER_VERSION" 2>>"$OUT/adapter.err"
VERSION_RC=$?
MEASURED_BUILD=$(cat "$ADAPTER_VERSION")
[ "$VERSION_RC" != 124 ] ||
  cannot "the installed build did not answer --version within ${ADAPTER_SECONDS}s"
[ "$VERSION_RC" = 0 ] ||
  cannot "the adapter could not ask the installed build its version"
[ -n "$MEASURED_BUILD" ] ||
  cannot "the adapter reported an empty version for the installed build"
case "$MEASURED_BUILD" in
  *[!\ -~]*) cannot "the reported version is not printable text: $MEASURED_BUILD" ;;
esac

STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# -- the run ------------------------------------------------------------------
#
# ⚠ The observer prints its endpoints, writes the torrent, and then serves until
# its own deadline. Waiting on its line rather than on a delay is what makes
# this work on a slow runner.
"$OBSERVER" "$BUNDLE" "$SECONDS_TO_SERVE" "$TORRENT" ${PEER_PORT:+"$PEER_PORT"} >"$LOG" 2>&1 &
OBSERVER_PID=$!
# ⚠ Read once, here, because every deadline below is measured from the moment
# the observer began its own. Re-reading the clock later would compare a span
# against a start that had moved.
OBSERVER_EPOCH=$(date +%s)

WAITED=0
while [ "$WAITED" -lt 60 ]; do
  grep -q '^serving for ' "$LOG" 2>/dev/null && break
  kill -0 "$OBSERVER_PID" 2>/dev/null || break
  sleep 1
  WAITED=$((WAITED + 1))
done

ANNOUNCE_URL=$(awk '$1 == "announce-url" { print $2; exit }' "$LOG")
if [ -z "$ANNOUNCE_URL" ]; then
  wait "$OBSERVER_PID" 2>/dev/null
  sed 's/^/          /' "$LOG" >&2
  refuse "the observer never reported an announce URL"
fi

# ⛔ THE TORRENT IS A FILE THE CLIENT READS, AND IT MUST EXIST BEFORE THE CLIENT
# IS STARTED. An adapter handed a path to nothing would report its own failure,
# which is a different fact from the observer never having written one.
[ -s "$TORRENT" ] || {
  wait "$OBSERVER_PID" 2>/dev/null
  refuse "the observer wrote no torrent at $TORRENT"
}

# -- ⛔ THE SESSIONS, WHICH ARE THE ONLY DIMENSION THAT SEPARATES TWO LIFETIMES --
#
# ⛔ **A VALUE THE BUILD STORES AND ONE IT REGENERATES READ IDENTICALLY IN ONE
# SESSION.** `SamplingPlan::restarts` says so in as many words, and
# `Lifetime::PerSession` is the answer only a restart can reach. Run 20 measured
# exactly that gap: `aria2-next` announced twice, `started` and `stopped`, and
# both carried the SAME peer ID - so the field is `constant` with two samples,
# two lanes state two different constants, and `classify_across` says
# `divergent`. A second session is what makes the tail vary inside one capture.
#
# ⛔ **THE STOP IS INSIDE THE OBSERVER'S WINDOW AND USED NOT TO BE.** It ran
# after `wait "$OBSERVER_PID"`, so whatever a build said on its way out reached a
# tracker that had already gone. ⚠ Run 19 answered `interval: 15` under a
# 45-second deadline and the build announced once anyway, so asking a build to
# re-announce is a condition this project can set and cannot enforce; a shutdown
# is different, because the build sends it or does not and the only question was
# whether anything was listening.
#
# ⚠ **The margin is a fifth of the run, capped at five seconds and floored at
# one**, so the last `stopped` announce and the observer's record of it both fit
# inside what is left. A margin as long as the run would stop the build before it
# had announced at all.
STOP_MARGIN=$((SECONDS_TO_SERVE / 5))
[ "$STOP_MARGIN" -le 5 ] || STOP_MARGIN=5
[ "$STOP_MARGIN" -ge 1 ] || STOP_MARGIN=1
SESSION_WINDOW=$((SECONDS_TO_SERVE - STOP_MARGIN))

SESSION=0
SESSIONS_STARTED=0
STOP_RC=0
STOPPED_AFTER=0
while [ "$SESSION" -lt "$SESSIONS" ]; do
  SESSION=$((SESSION + 1))

  # ⭐ THE CLIENT IS STARTED WITH THE TORRENT AND NOTHING ELSE. It learns the
  # tracker's address by reading the file, which is what a stock build does with
  # any torrent, so nothing here is a control the product does not already have.
  timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" sh "$ADAPTER" start "$TORRENT" "$WORKDIR" "${PEER_PORT:-0}" \
    </dev/null >>"$OUT/client.log" 2>&1
  START_RC=$?
  if [ "$START_RC" != 0 ]; then
    # ⛔ THE FIRST SESSION REFUSES AND A LATER ONE ENDS THE LOOP. A build that
    # never started is a run with no measurement in it; a build that started once
    # and would not start again has been measured once, which is weaker than was
    # asked for and better than nothing - the same argument `client-capture`
    # makes about a refused re-dial. The count is in the attestation either way.
    if [ "$SESSIONS_STARTED" -gt 0 ]; then
      printf 'capture-client: session %s would not start (exit %s); %s of %s ran\n' \
        "$SESSION" "$START_RC" "$SESSIONS_STARTED" "$SESSIONS" >&2
      break
    fi
    timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" sh "$ADAPTER" stop "$WORKDIR" </dev/null >/dev/null 2>&1
    wait "$OBSERVER_PID" 2>/dev/null
    sed 's/^/          /' "$OUT/client.log" >&2
    if [ "$START_RC" = 124 ]; then
      refuse "the adapter did not return from start within ${ADAPTER_SECONDS}s; it launches and returns"
    fi
    refuse "the adapter could not start the build (exit $START_RC)"
  fi
  SESSIONS_STARTED=$((SESSIONS_STARTED + 1))

  # ⛔ EACH SESSION'S BOUNDARY IS MEASURED FROM THE OBSERVER'S OWN START, never
  # from this line. `start` is bounded at ADAPTER_SECONDS rather than at the
  # capture deadline and a stop takes as long as the build takes to go, so a
  # sleep computed per session from here would drift past a deadline that had
  # already expired.
  # ⚠ NOT `TARGET`. That name is already the adapter's target and is read much
  # later to write `target=` into the attestation; a loop variable spelled the
  # same way put `target=16` in a document nothing else would have questioned.
  # Found by driving the run and reading the attestation back, which is the
  # shell twin of the `[switch]$Marker` collision `CI-03` records.
  SESSION_ENDS_AT=$((SESSION_WINDOW * SESSION / SESSIONS))
  ELAPSED=$(($(date +%s) - OBSERVER_EPOCH))
  STOP_IN=$((SESSION_ENDS_AT - ELAPSED))
  [ "$STOP_IN" -le 0 ] || sleep "$STOP_IN"

  # ⛔ READ BEFORE THE STOP IS CALLED, NEVER AFTER IT RETURNS, and run 20 is why.
  # Its first spelling timed the stop's RETURN: `aria2.shutdown` took five
  # seconds, the stop began at 40s of a 45-second window and returned at 45, and
  # the field read `stopped_within_window=no` over a run whose `stopped` announce
  # had been heard - `announces=2`, both in the transcript. ⚠ The question this
  # answers is whether the observer was still recording when the build was ASKED
  # to shut down, which is the condition this runner controls; how long the build
  # then took is the build's. A shutdown slower than the margin is still missed,
  # and that is a limit of the margin rather than something a second timestamp
  # would fix.
  STOPPED_AFTER=$(($(date +%s) - OBSERVER_EPOCH))

  timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" sh "$ADAPTER" stop "$WORKDIR" </dev/null >>"$OUT/client.log" 2>&1
  STOP_RC=$?
done

# ⚠ The LAST stop is what this describes, because it is the one whose `stopped`
# announce the observer had to still be there for.
STOPPED_WITHIN=no
[ "$STOPPED_AFTER" -ge "$SECONDS_TO_SERVE" ] || STOPPED_WITHIN=yes

wait "$OBSERVER_PID"
OBSERVER_RC=$?

# ⚠ STOPPED WHATEVER HAPPENED ABOVE. A client left running holds a port and its
# own executable open, and on a host that is about to upload evidence that is a
# process writing into the directory being uploaded. ⛔ Called again rather than
# only on the failing path: the contract says a stop over a build that already
# exited is done rather than refused, and a second call is how a stop that was
# interrupted by its own time limit still leaves nothing running.
if [ "$STOP_RC" != 0 ]; then
  timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" sh "$ADAPTER" stop "$WORKDIR" </dev/null >>"$OUT/client.log" 2>&1
fi

[ "$OBSERVER_RC" = "0" ] || {
  sed 's/^/          /' "$LOG" >&2
  refuse "the observer exited $OBSERVER_RC"
}

# ⚠ THE SEGMENT GUARD IS EARLIER THAN THE ANNOUNCE GUARD AND MASKS IT ON THE
# EASY INPUT. A build that never started leaves no bytes at all, so `segments`
# is what refuses that run and `announces` is never reached; the announce guard
# only fires where something DID reach the socket without announcing, which is a
# connection that opened and said nothing a tracker could read. Both are real
# and they are separate refusals, which is why the harness provokes the second
# one through a request the codec refuses rather than through silence.
SEGMENTS=$(awk '$1 == "segments:" { print $2; exit }' "$LOG")
[ -n "$SEGMENTS" ] || refuse "the observer reported no segment count"
[ "$SEGMENTS" -gt 0 ] || refuse "the run recorded no bytes at all"

ANNOUNCES=$(awk '$1 == "announces" { print $2; exit }' "$LOG")
[ -n "$ANNOUNCES" ] || refuse "the observer reported no announce count"
[ "$ANNOUNCES" -gt 0 ] || refuse "the build announced nothing; there is no measurement here"

# ⛔ AND THE INTERVAL MUST NOT OUTLIVE THE RUN. `capture-client` run 18 answered
# 60 under a 45-second deadline, so no re-announce was ever due: the tracker
# surface rested on one sample, `SCHEMA-04` could state every field only as a
# constant, and two lanes then stated two different constants. These are
# CANNOT-RUN rather than refusals, because they are conditions of the experiment
# set before the build did anything, and reporting either as the build's
# behaviour would name the wrong thing.
#
# ⚠ BOTH VALUES ARE READ OUT OF THE OBSERVER'S OWN LOG, never recomputed here. A
# second reading of the derivation would agree with itself while the observer
# answered something else, which is the rule `check-workflow.sh` is built on;
# what the suite pins is `TrackerResponse::within`, and what this pins is that
# the observer was given this run's deadline and answered under it.
#
# ⚠ EQUALITY IS PERMITTED AND THAT IS NOT A WEAKENING. The derivation has a
# five-second floor, so a capture asked for five seconds or fewer answers an
# interval equal to its own deadline and no shorter one exists. The case this
# refuses is an interval LONGER than the run, which is the only one a different
# deadline could have fixed.
SERVING_FOR=$(awk '$1 == "serving" && $2 == "for" { sub(/s$/, "", $3); print $3; exit }' "$LOG")
case "$SERVING_FOR" in
  '' | *[!0-9]*) cannot "the observer did not report how long it served" ;;
esac
[ "$SERVING_FOR" = "$SECONDS_TO_SERVE" ] ||
  cannot "the observer served for ${SERVING_FOR}s, not the ${SECONDS_TO_SERVE}s it was given"

OFFERED_INTERVAL=$(awk '$1 == "offered-interval" { print $2; exit }' "$LOG")
case "$OFFERED_INTERVAL" in
  '' | *[!0-9]*) cannot "the observer reported no announce interval" ;;
esac
[ "$OFFERED_INTERVAL" -le "$SERVING_FOR" ] ||
  cannot "the tracker answered interval ${OFFERED_INTERVAL}s under a ${SERVING_FOR}s deadline; no re-announce can fall due"

# -- the evidence, checked by something else ----------------------------------
#
# ⭐ THE ROWS ARE THE OBSERVER'S OWN CLAIMS AND sha256sum IS WHAT TESTS THEM.
awk '$1 == "evidence" {
       digest = $5
       sub(/^sha256:/, "", digest)
       printf "%s  bundle/%s\n", digest, $3
     }' "$LOG" >"$SUMS"

EVIDENCE=$(grep -c . "$SUMS")
[ "$EVIDENCE" -gt 0 ] || refuse "the observer described no evidence"

(cd "$OUT" && sha256sum -c SHA256SUMS) >"$OUT/verify.log" 2>&1 ||
  refuse "the evidence does not verify against the digests the observer declared"

# -- and the transcript must carry a peer ID this observer did not generate ----
#
# ⛔ THIS IS THE CASE THAT SAYS A BUILD WAS MEASURED. Everything above is
# satisfied by a bundle of empty artifacts that verify against their own empty
# digests, or by an observer announcing to itself. Two separate refusals,
# because the fixes differ: an announce carrying the observer's own fixture peer
# ID means nothing external ran, and a peer ID the transcript does not hold
# means the observer reported an announce the bytes do not show.
OBSERVED_ONLY=$(awk '$1 == "announce" && $3 == "is-observer-peer-id" && $4 == "false" { found = 1 }
                     END { print (found ? "no" : "yes") }' "$LOG")
[ "$OBSERVED_ONLY" = "no" ] ||
  refuse "every announce carried this observer's own peer ID; no build was measured"

# ⛔ A THIRD REFUSAL, BECAUSE AN ANNOUNCE CAN CARRY NO PEER ID AT ALL. The
# observer prints `absent` or `undecodable: ...` for those, and an earlier
# version of this took `$4` unconditionally and then searched the transcript for
# the literal word `absent` - which is a guard whose verdict depends on whether
# an unrelated English word happens to appear in a JSON document. The hex is
# required to BE hex, and it is taken from the first announce that was not this
# observer's own, because those are the bytes a build chose.
#
# ⚠ The peer-id line precedes the is-observer-peer-id line for the same index,
# which is what makes one pass in printed order enough.
PEER_ID_HEX=$(awk '
  $1 == "announce" && $3 == "peer-id" { pending = $4; index_of = $2 }
  $1 == "announce" && $3 == "is-observer-peer-id" && $2 == index_of && $4 == "false" {
    if (!found && pending ~ /^[0-9a-f]+$/) { print pending; found = 1 }
  }' "$LOG")
[ -n "$PEER_ID_HEX" ] ||
  refuse "no announce carried a readable peer ID; no build was identified"
grep -q "$PEER_ID_HEX" "$BUNDLE/tracker-http.transcript.json" 2>/dev/null ||
  refuse "the transcript does not carry the peer ID the observer reported"

# -- and a SECOND reading of those same bytes ---------------------------------
#
# ⛔ THIS IS WHAT MAKES THE BUNDLE A RECORD RATHER THAN AN ATTESTATION. Absolute
# 2 asks for the Rust observer plus at least one independent connector, and
# `E-CAP-01` refuses a record declaring one connector as an INVALID DOCUMENT -
# not as an unpublishable one. Every capture this project ran before this step
# existed declared one, which is why run 14's four artifacts could not become a
# record however complete they looked.
#
# ⛔ IT RUNS AFTER `sha256sum -c`, DELIBERATELY. The connector reads the same
# transcripts the observer wrote, so reading them before they had been verified
# would be corroborating bytes nothing had yet established were the ones the
# observer declared.
#
# ⚠ IT DOES NOT RE-DRIVE THE BUILD, and this is the limit to state rather than
# to leave implied. Two connectors on one wire would be two products; what this
# gives is two READINGS of one capture's raw bytes, which is what catches a
# decoding defect and is not what catches a lab that recorded the wrong bytes.
# `OBS-07` carries the distinction.
CONNECTOR_ID=$(timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" \
  python3 "$CONNECTOR" --describe </dev/null 2>"$OUT/connector.err" |
  awk -F= '$1 == "id" { sub(/^[^=]*=/, ""); print; exit }')
[ -n "$CONNECTOR_ID" ] ||
  cannot "the connector did not describe its own identifier"
# ⛔ AND ITS VERSION, which this file dropped until 2026-10-14 while the record
# carried `0.0.0` for it, a value nobody measured.
CONNECTOR_VERSION=$(timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" \
  python3 "$CONNECTOR" --describe </dev/null 2>>"$OUT/connector.err" |
  awk -F= '$1 == "version" { sub(/^[^=]*=/, ""); print; exit }')
[ -n "$CONNECTOR_VERSION" ] ||
  cannot "the connector did not describe its own version"
case "$CONNECTOR_ID" in
  -* | *- | *--* | *[!a-z0-9-]*)
    cannot "the connector named itself [$CONNECTOR_ID], which is not a slug"
    ;;
esac

# ⛔ THE REPORT'S PATH IS THE CONNECTOR'S OWN DEFAULT, NEVER COMPOSED HERE. A
# second spelling of `connector/<id>.txt` in this file would go on writing the
# old place the day the connector's layout moves, and `assemble-capture` would
# read a file nothing wrote.
timeout -k "$ADAPTER_KILL_AFTER" "$ADAPTER_SECONDS" \
  python3 "$CONNECTOR" --bundle "$BUNDLE" </dev/null \
  >>"$OUT/connector.log" 2>>"$OUT/connector.err"
CONNECTOR_RC=$?
case "$CONNECTOR_RC" in
  0) : ;;
  1)
    sed 's/^/          /' "$OUT/connector.err" >&2
    refuse "the second connector could not read what the observer recorded"
    ;;
  124) cannot "the connector did not return within ${ADAPTER_SECONDS}s" ;;
  *)
    sed 's/^/          /' "$OUT/connector.err" >&2
    cannot "the connector could not run (exit $CONNECTOR_RC)"
    ;;
esac

REPORT="$BUNDLE/connector/$CONNECTOR_ID.txt"
[ -s "$REPORT" ] ||
  refuse "the connector exited 0 and wrote no report at $REPORT"

# ⛔ AND THE CONNECTOR HAD BETTER HAVE READ THE SAME BUILD. The attestation's
# `measured_peer_id` is what the observer decoded out of the announce; a
# connector reporting different bytes for that field is a real finding and it is
# recorded rather than refused - `E-PUB-01` is what holds such a record back,
# and refusing here would lose the evidence of the disagreement.
# ⚠ SO THIS IS A LINE IN THE ATTESTATION, NOT A GUARD. The one thing that IS
# refused is a report that says nothing about the field at all, because silence
# is not a disagreement and `assemble-capture` refuses it a dispatch later.
CONNECTOR_PEER_ID=$(awk -F= '$1 == "tracker_http/peer_id" { sub(/^[^=]*=/, ""); print; exit }' "$REPORT")
[ -n "$CONNECTOR_PEER_ID" ] ||
  refuse "the connector's report says nothing about tracker_http/peer_id"

# ⛔ THE REPORT JOINS THE DOCUMENT THAT COVERS THE BUNDLE. The rows above are the
# observer's own claims tested by sha256sum; this row is the connector's, tested
# the same way, and one document means a downloader verifies the whole bundle in
# one command rather than learning that one file was covered by nothing.
# ⚠ THE WHOLE FILE IS RE-VERIFIED, not just the row that was added: a second
# `sha256sum -c` over everything is what says the append did not disturb what
# was already there.
printf '%s  bundle/connector/%s.txt\n' \
  "$(sha256sum "$REPORT" | cut -d' ' -f1)" "$CONNECTOR_ID" >>"$SUMS"
(cd "$OUT" && sha256sum -c SHA256SUMS) >"$OUT/verify.log" 2>&1 ||
  refuse "the bundle no longer verifies once the connector's report is covered"
EVIDENCE=$(grep -c . "$SUMS")

# -- rule 12, at the choke point rather than in each adapter -------------------
#
# ⛔ A CAPTURE'S OUTPUT DIRECTORY IS ITS EVIDENCE BUNDLE, AND A PRODUCT THIS
# PROJECT DID NOT WRITE HAS BEEN WRITING INTO IT: `aria2-next` once left its
# per-run RPC token there and it shipped. `scan-bundle.sh` carries the measurement
# and both rules; it is the one home for them, and the workflow asks it again
# about a capture that refused before reaching this line.
sh "$ROOT/scripts/capture/scan-bundle.sh" "$OUT"
SCAN_RC=$?
case "$SCAN_RC" in
  0) : ;;
  1) refuse "the capture would ship what rule 12 forbids, named above" ;;
  *) cannot "the rule-12 scan could not run (exit $SCAN_RC)" ;;
esac

FINISHED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
OBSERVER_DIGEST=$(sha256sum "$OBSERVER" | cut -d' ' -f1)
CONNECTOR_DIGEST=$(sha256sum "$CONNECTOR" | cut -d' ' -f1)
ADAPTER_DIGEST=$(sha256sum "$ADAPTER" | cut -d' ' -f1)
INFO_HASH=$(awk '$1 == "info-hash" { print $2; exit }' "$LOG")
FIXTURE=$(awk '$1 == "fixture-sha256" { print $2; exit }' "$LOG")
PEER_STREAMS=$(awk '$1 == "peer-streams" { print $2; exit }' "$LOG")
PEER_DIALLED=$(awk '$1 == "peer-dialled" { $1 = ""; sub(/^ /, ""); print; exit }' "$LOG")

# -- the attestation ----------------------------------------------------------
#
# ⚠ Key=value, one per line, the shape assert-disposable's own marker uses and
# `capture-run`'s attestation shares. ⛔ `stock_client` is the adapter's own
# declaration rather than an assumption: a stub adapter drives every guard above
# and is not a product, so a run it drove says so in a field.
{
  printf 'bit-ids/capture-attestation/1\n'
  printf 'run=%s\n' "$RUN_ID"
  printf 'kind=client\n'
  printf 'target=%s\n' "$TARGET"
  printf 'measured_build=%s\n' "$MEASURED_BUILD"
  printf 'stock_client=%s\n' "$([ "$ADAPTER_KIND" = stock ] && printf true || printf false)"
  printf 'adapter=%s\n' "$ADAPTER"
  printf 'adapter_kind=%s\n' "$ADAPTER_KIND"
  printf 'adapter_sha256=%s\n' "$ADAPTER_DIGEST"
  printf 'adapter_stop_status=%s\n' "$STOP_RC"
  # ⛔ WHETHER THE BUILD WAS STOPPED WHILE ANYTHING WAS STILL LISTENING. A run
  # that answers `no` measured nothing about a shutdown, so a build that sends no
  # `stopped` announce and a run that was not there to hear one read the same
  # without this field - and only the first is a fact about the build.
  printf 'stopped_within_window=%s\n' "$STOPPED_WITHIN"
  # ⚠ The measurement the verdict above is read from, kept beside it rather than
  # left to be re-derived. `within` alone cannot tell a stop LATE in the window
  # from one at the very start, and a build stopped a second after it launched
  # has been asked to shut down before it did anything worth recording.
  printf 'stopped_after=%ss\n' "$STOPPED_AFTER"
  printf 'stop_margin=%ss\n' "$STOP_MARGIN"
  # ⛔ ASKED FOR AND ACHIEVED, as two numbers. A plan that asked for two sessions
  # and ran one supports exactly the conclusions one session supports, and
  # `SamplingPlan` is built from what the run DID rather than from what it wanted.
  printf 'sessions=%s\n' "$SESSIONS"
  printf 'sessions_started=%s\n' "$SESSIONS_STARTED"
  printf 'platform=%s\n' "$(uname -srm)"
  printf 'started_at=%s\n' "$STARTED"
  printf 'finished_at=%s\n' "$FINISHED"
  printf 'host_fingerprint=%s\n' "$FINGERPRINT"
  printf 'claim_marker=%s\n' "$MARKER"
  # ⭐ The other end of the monotonic span the claim started. `/proc/uptime` is
  # the host's, and the namespace this runs in does not virtualise it.
  printf 'uptime_at_finish=%s\n' "$(cut -d' ' -f1 /proc/uptime 2>/dev/null)"
  printf 'claimed_run=%s\n' "$CLAIMED_RUN"
  printf 'egress=closed\n'
  printf 'egress_source=%s\n' "$EGRESS_SOURCE"
  printf 'observer=%s\n' "$OBSERVER"
  printf 'observer_sha256=%s\n' "$OBSERVER_DIGEST"
  printf 'observer_version=%s\n' "$(awk '$1 == "observer-version" { print $2; exit }' "$LOG")"
  # ⛔ `connectors=` IS THE KEY `assemble-capture` READS, and it lists the ones
  # BESIDE the observer: the observer is already named above and the assembler
  # adds it, so a list naming it twice would declare two connectors where one
  # ran. ⚠ Comma-separated, because a second independent connector is a thing
  # this project expects to gain rather than a slot for exactly one.
  printf 'connectors=%s\n' "$CONNECTOR_ID"
  printf 'connector=%s\n' "$CONNECTOR"
  printf 'connector_sha256=%s\n' "$CONNECTOR_DIGEST"
  printf 'connector_version=%s\n' "$CONNECTOR_VERSION"
  printf 'connector_peer_id=%s\n' "$CONNECTOR_PEER_ID"
  printf 'announce_url=%s\n' "$ANNOUNCE_URL"
  printf 'info_hash=%s\n' "$INFO_HASH"
  printf 'fixture=%s\n' "$FIXTURE"
  printf 'announces=%s\n' "$ANNOUNCES"
  printf 'measured_peer_id=%s\n' "$PEER_ID_HEX"
  printf 'peer_streams=%s\n' "${PEER_STREAMS:-0}"
  printf 'peer_dialled=%s\n' "${PEER_DIALLED:-none}"
  printf 'segments=%s\n' "$SEGMENTS"
  printf 'evidence=%s\n' "$EVIDENCE"
  printf 'evidence_verified_by=sha256sum\n'
} >"$ATTEST"

# ⛔ READ BACK, NOT ASSUMED.
grep -q "^run=$RUN_ID\$" "$ATTEST" ||
  refuse "the attestation was not written"

printf '%s\n' "$FINGERPRINT"
printf 'capture-client: %s %s, %s announce(s), peer id %s\n' \
  "$TARGET" "$MEASURED_BUILD" "$ANNOUNCES" "$PEER_ID_HEX"
printf 'capture-client: %s segment(s), %s artifact(s), verified by sha256sum\n' \
  "$SEGMENTS" "$EVIDENCE"
printf 'capture-client: attestation %s\n' "$ATTEST"
exit 0
