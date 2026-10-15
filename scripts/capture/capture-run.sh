#!/bin/sh
# capture-run.sh - the capture itself, on a host that has already been claimed
# and already had its route off the machine deleted.
#
# ⛔ IT BUILDS NOTHING, AND THAT IS THE WHOLE REASON IT IS A SEPARATE SCRIPT.
# By the time this runs there is no network: the workflow deleted the default
# route in both address families before calling it. A capture that discovered a
# missing dependency here would have to restore egress to fix it, which is
# restoring egress to a host running a binary this project downloaded minutes
# ago. So the observer is a path to an ALREADY BUILT binary, and a missing one
# is `could not run` rather than something to repair in place.
#
# -- ⛔ WHAT THIS IS NOT ------------------------------------------------------
#
# It is a FIXTURE capture. Nothing here installs a client, and no stock
# BitTorrent build is observed: `CLIENT-01` owns that and `capture.kind` in the
# attestation says `fixture` so a bundle cannot be mistaken for one. What is
# measured here is the containment and the evidence path, end to end, which is
# `CI-03`'s Prove and not `CLIENT-01`'s.
#
# -- ⭐ THE DRIVER IS A CLIENT THIS PROJECT DID NOT WRITE ---------------------
#
# `curl` is a complete HTTP client, it is on both hosted runner images, and it
# puts real bytes through the tracker observer. A driver written here would be
# this project checking its own reading of HTTP against itself.
#
# ⭐ AND IT SENDS A TOKEN IT KNOWS, so the transcript can be checked against
# what actually happened rather than against what the observer believed. The
# announce carries `key=<run-id>`; the transcript must hold those bytes.
# Without that, an observer that wrote an empty artifact and a bundle that
# verified against its own empty digest would pass every other check here.
#
# -- ⛔ EVERY REFUSAL SAYS SOMETHING DIFFERENT --------------------------------
#
# There are three exit codes and many more refusals, so nearly every one shares
# its code with several others, and two guards answering one code mask each
# other: deleting either leaves every case green because the survivor produces
# the code the case asserts. `check-capture.sh` asserts the MESSAGE on every
# case, not the code alone.
#
# Usage:
#   sh scripts/capture/capture-run.sh --run-id <id> --out <dir> --observer <path>
#                                     [--seconds <n>] [--route-table <path>]
#
# Exit codes: 0 the capture ran and its evidence verifies, 1 a guard refuses,
# 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

RUN_ID=""
OUT=""
OBSERVER=""
SECONDS_TO_SERVE=15
ROUTE_TABLE=""

usage() {
  awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
}

# ⛔ ONE MESSAGE PREFIX, so a refusal is attributable to this script rather than
# to the guard it delegated to. The guards print their own name as well, which
# is how a log says which of the two refused.
refuse() { # message
  printf 'capture-run: %s\n' "$1" >&2
  exit 1
}

cannot() { # message
  printf 'capture-run: %s\n' "$1" >&2
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
    --seconds)
      [ $# -ge 2 ] || cannot "--seconds takes a value"
      SECONDS_TO_SERVE="$2"
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
      printf 'capture-run: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

[ -n "$RUN_ID" ] || cannot "--run-id is required"
[ -n "$OUT" ] || cannot "--out is required"
[ -n "$OBSERVER" ] || cannot "--observer is required"

# ⚠ The same slug rule the guard applies, restated here rather than delegated,
# because this value becomes a token on the wire and a key in a document before
# the guard is ever consulted.
case "$RUN_ID" in
  '' | *[!a-z0-9-]*) cannot "run id must be lowercase a-z0-9-: $RUN_ID" ;;
esac

case "$SECONDS_TO_SERVE" in
  '' | *[!0-9]*) cannot "--seconds must be a whole number: $SECONDS_TO_SERVE" ;;
esac

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)
# ⭐ THE HOST GUARD IS `bit-check assert-disposable` SINCE 2026-10-15, `CI-11`,
# found through BIT_CHECK as the workflow exports it, and never built here.
if [ -z "${BIT_CHECK:-}" ] || [ ! -x "$BIT_CHECK" ]; then
  cannot "BIT_CHECK names no bit-check binary; it is built before the capture, never here"
fi

for tool in curl sha256sum awk od; do
  command -v "$tool" >/dev/null 2>&1 || cannot "$tool not found"
done

# ⛔ THE BUILD RAN BEFORE THE NETWORK WAS CUT, OR IT DID NOT RUN. This is the
# one refusal that reports a workflow whose step order has drifted, and it is
# `could not run` rather than a guard refusing: nothing about the host is wrong.
[ -x "$OBSERVER" ] ||
  cannot "the observer $OBSERVER is not executable; it is built before the route is deleted, never here"

# ⛔ A SECOND CAPTURE MUST NOT WRITE OVER THE FIRST ONE'S EVIDENCE. The bundle
# writer would happily add artifacts to a directory that already holds some, and
# the two runs would then be described by one attestation.
[ -e "$OUT" ] && refuse "$OUT already exists; a capture never writes into another run's evidence"

# -- the host, as this run finds it -------------------------------------------
#
# ⛔ THE MARKER'S PATH IS ASKED FOR, NEVER COMPOSED. `--marker` is the one
# derivation; a second spelling here would go on reading the old location the
# day the state directory moves and report an unclaimed host as claimed.
MARKER=$("$BIT_CHECK" assert-disposable --marker) || cannot "the guard could not report its marker path"
[ -n "$MARKER" ] || cannot "the guard reported an empty marker path"

# ⛔ THE CLAIM IS CHECKED HERE AS WELL AS RUN EARLIER, and that is not
# belt-and-braces. The claim is a separate workflow step, so a step order that
# dropped it leaves this script capturing on a host nothing ever claimed. A
# guard on one of two paths into the same action is the one-gated-door defect.
[ -f "$MARKER" ] ||
  refuse "the host was never claimed; $MARKER does not exist"

CLAIMED_RUN=$(awk -F= '$1 == "run" { print $2; exit }' "$MARKER")
[ "$CLAIMED_RUN" = "$RUN_ID" ] ||
  refuse "the host is claimed by run [$CLAIMED_RUN], not [$RUN_ID]"

# ⛔ AND THE ROUTE IS READ AGAIN AT THE MOMENT OF CAPTURE. The workflow runs the
# same guard a step earlier; what this adds is that nothing between the two
# steps put the route back. The verdict is recorded in the attestation, so a run
# that passed over a fixture table is visible rather than indistinguishable from
# one that passed over the machine.
if [ -n "$ROUTE_TABLE" ]; then
  "$BIT_CHECK" assert-disposable --egress "$ROUTE_TABLE"
  EGRESS_RC=$?
  EGRESS_SOURCE="$ROUTE_TABLE"
else
  "$BIT_CHECK" assert-disposable --egress
  EGRESS_RC=$?
  EGRESS_SOURCE=/proc/net/route
fi
case "$EGRESS_RC" in
  0) : ;;
  1) refuse "egress is open; the default route is still there and a capture would leak" ;;
  *) cannot "egress could not be established from $EGRESS_SOURCE" ;;
esac

FINGERPRINT=$("$BIT_CHECK" assert-disposable --fingerprint) || cannot "the host fingerprint could not be read"

mkdir -p "$OUT" || cannot "cannot create $OUT"
OUT=$(CDPATH='' cd -- "$OUT" && pwd)
BUNDLE="$OUT/bundle"
LOG="$OUT/observer.log"
SUMS="$OUT/SHA256SUMS"
ATTEST="$OUT/attestation.txt"

STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# -- the run ------------------------------------------------------------------
#
# ⚠ The observer prints its endpoints and then serves until its own deadline.
# Waiting on the line rather than on a fixed delay is what makes this work on a
# slow runner: a sleep long enough for the worst case is dead time on every
# other run, and one tuned to this host is a race on a busier one.
"$OBSERVER" "$BUNDLE" "$SECONDS_TO_SERVE" >"$LOG" 2>&1 &
OBSERVER_PID=$!

WAITED=0
while [ "$WAITED" -lt 30 ]; do
  grep -q '^serving for ' "$LOG" 2>/dev/null && break
  kill -0 "$OBSERVER_PID" 2>/dev/null || break
  sleep 1
  WAITED=$((WAITED + 1))
done

TRACKER=$(awk '$1 == "endpoint" && $2 == "tracker-http" { print $3; exit }' "$LOG")
if [ -z "$TRACKER" ]; then
  wait "$OBSERVER_PID" 2>/dev/null
  sed 's/^/          /' "$LOG" >&2
  refuse "the observer never reported a tracker-http endpoint"
fi

# ⭐ A TOKEN THE DRIVER KNOWS IT SENT. BEP 3's `key` is an opaque client value,
# so a tracker observer keeps it verbatim and the exact bytes are recoverable
# from the transcript below.
DRIVER_VERSION=$(curl --version 2>/dev/null | head -1)
INFO_HASH="%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a"
ANNOUNCE="http://$TRACKER/announce?info_hash=$INFO_HASH&peer_id=bit-ids-fixture-0001"
ANNOUNCE="$ANNOUNCE&port=6881&uploaded=0&downloaded=0&left=0&compact=1"
ANNOUNCE="$ANNOUNCE&event=started&key=$RUN_ID"

# ⚠ --noproxy is not tidiness. A runner with a proxy variable set would send the
# announce to the proxy, which is off this host, which is the one thing the
# containment exists to stop. A loopback address must be dialled directly.
DRIVER_STATUS=$(curl -sS --noproxy '*' --max-time 20 -o /dev/null -w '%{http_code}' "$ANNOUNCE" 2>"$OUT/driver.err")
DRIVER_RC=$?

wait "$OBSERVER_PID"
OBSERVER_RC=$?

[ "$DRIVER_RC" = "0" ] || {
  sed 's/^/          /' "$OUT/driver.err" >&2
  refuse "the driver could not reach the observer (curl exit $DRIVER_RC)"
}
[ "$OBSERVER_RC" = "0" ] || {
  sed 's/^/          /' "$LOG" >&2
  refuse "the observer exited $OBSERVER_RC"
}

SEGMENTS=$(awk '$1 == "segments:" { print $2; exit }' "$LOG")
[ -n "$SEGMENTS" ] || refuse "the observer reported no segment count"
[ "$SEGMENTS" -gt 0 ] || refuse "the run recorded no bytes at all"

# -- the evidence, checked by something else ----------------------------------
#
# ⭐ THE ROWS ARE THE OBSERVER'S OWN CLAIMS AND sha256sum IS WHAT TESTS THEM.
# The bundle writer already read its files back and compared them against the
# buffer it wrote; that is the writer checking itself. This is coreutils
# checking the writer.
awk '$1 == "evidence" {
       digest = $5
       sub(/^sha256:/, "", digest)
       printf "%s  bundle/%s\n", digest, $3
     }' "$LOG" >"$SUMS"

EVIDENCE=$(grep -c . "$SUMS")
[ "$EVIDENCE" -gt 0 ] || refuse "the observer described no evidence"

(cd "$OUT" && sha256sum -c SHA256SUMS) >"$OUT/verify.log" 2>&1 ||
  refuse "the evidence does not verify against the digests the observer declared"

# ⛔ AND THE TRANSCRIPT MUST CARRY WHAT THE DRIVER SENT. Everything above is
# satisfied by a bundle of empty artifacts that verify against their own empty
# digests. This is the only case that says a client's bytes reached the record.
TOKEN_HEX=$(printf 'key=%s' "$RUN_ID" | od -An -tx1 | tr -d ' \n')
grep -q "$TOKEN_HEX" "$BUNDLE/tracker-http.transcript.json" 2>/dev/null ||
  refuse "the transcript does not carry the bytes the driver sent"

FINISHED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
OBSERVER_DIGEST=$(sha256sum "$OBSERVER" | cut -d' ' -f1)

# -- the attestation ----------------------------------------------------------
#
# ⚠ Key=value, one per line, the shape assert-disposable's own marker uses.
# Nothing in Rust parses it: this is what a human and a later job read, and a
# second document format for four facts would be a second thing to keep in step.
{
  printf 'bit-ids/capture-attestation/1\n'
  printf 'run=%s\n' "$RUN_ID"
  printf 'kind=fixture\n'
  printf 'measured_build=none\n'
  printf 'stock_client=false\n'
  printf 'platform=%s\n' "$(uname -srm)"
  printf 'started_at=%s\n' "$STARTED"
  printf 'finished_at=%s\n' "$FINISHED"
  printf 'host_fingerprint=%s\n' "$FINGERPRINT"
  printf 'claim_marker=%s\n' "$MARKER"
  printf 'claimed_run=%s\n' "$CLAIMED_RUN"
  printf 'egress=closed\n'
  printf 'egress_source=%s\n' "$EGRESS_SOURCE"
  printf 'observer=%s\n' "$OBSERVER"
  printf 'observer_sha256=%s\n' "$OBSERVER_DIGEST"
  printf 'driver=curl\n'
  printf 'driver_version=%s\n' "$DRIVER_VERSION"
  printf 'driver_status=%s\n' "$DRIVER_STATUS"
  printf 'segments=%s\n' "$SEGMENTS"
  printf 'evidence=%s\n' "$EVIDENCE"
  printf 'evidence_verified_by=sha256sum\n'
} >"$ATTEST"

# ⛔ READ BACK, NOT ASSUMED. A run that reported an attestation it failed to
# write is the same defect as a bundle described without being verified.
grep -q "^run=$RUN_ID\$" "$ATTEST" ||
  refuse "the attestation was not written"

printf '%s\n' "$FINGERPRINT"
printf 'capture-run: %s segment(s), %s artifact(s), verified by sha256sum\n' \
  "$SEGMENTS" "$EVIDENCE"
printf 'capture-run: attestation %s\n' "$ATTEST"
exit 0
