#!/bin/sh
# check-connector.sh - does the second connector read the BYTES, and does it
# refuse everything a corroboration must not be built on?
#
# ⛔ WHAT THIS EXISTS FOR IS THE ONE DEFECT A CONNECTOR CAN HAVE THAT LOOKS LIKE
# SUCCESS. A connector that copied the observer's answer, or that read the
# attestation, would agree with the observer on every field of every capture
# forever - and `E-COR-*` would record a corroborated measurement over one
# reading. So the load-bearing case here is not a refusal: it is that MOVING A
# BYTE IN THE TRANSCRIPT MOVES THE REPORT. Everything else guards the edges of
# that.
#
# -- ⭐ WHAT THE CONNECTOR IS AND WHY IT IS PYTHON ----------------------------
#
# `scripts/capture/connectors/cpython-stdlib.py` carries the whole argument in
# its own header, including the three rejected alternatives. In one line: the
# thing being corroborated is this project's Rust reading of these formats, so a
# second reader written in this workspace corroborates nothing.
#
# -- ⚠ WHAT THIS HARNESS DELIBERATELY DOES NOT DO ----------------------------
#
# It binds no socket and starts no observer. The transcripts here are written by
# this file, in the exact shape `bit-ids-lab`'s writer emits, and the CONTROL
# that a real bundle is one the connector accepts belongs where a real bundle
# exists: `check-capture-client` drives the real observer and asserts the report
# the runner wrote into that bundle. Two harnesses, because a socket case in
# here would make a decoder's mutation proofs wait on a deadline.
#
# ⛔ NOTHING HERE PLANTS IN A TRACKED FILE. Every bundle a case mutates is
# written into scratch state.
#
# ⚠ NO VALUE HERE IS ANY PRODUCT'S. The peer-ID prefix is `-XX0000-`, which no
# vendor uses, so nothing this harness writes can be mistaken for a measurement.
#
# Usage:
#   sh scripts/capture/check-connector.sh
#   sh scripts/capture/check-connector.sh --json
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
      printf 'check-connector: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-connector
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require python3 od

CONNECTOR="$ROOT/scripts/capture/connectors/cpython-stdlib.py"
[ -f "$CONNECTOR" ] || {
  printf 'check-connector: %s is not present\n' "$CONNECTOR" >&2
  exit 2
}

WORK=$(store_workdir checkconnector) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⛔ EVERY HEX RUN HERE IS COMPUTED, NEVER TYPED. An info hash and a peer ID are
# both forty hex digits, which is the shape `check-no-secrets --public` refuses
# because it is also the shape of a credential.
hexof() { # text
  printf '%s' "$1" | od -An -tx1 -v | tr -d ' \n'
}
repeat_byte() { # hex-pair count
  _out=""
  _n=0
  while [ "$_n" -lt "$2" ]; do
    _out="$_out$1"
    _n=$((_n + 1))
  done
  printf '%s' "$_out"
}

# Percent-encodes every byte of a value, which is what a stock client does with
# a peer ID: BEP 3 says the field is escaped, and a client that leaves the
# unreserved characters bare produces the same bytes either way.
pct_of() { # text
  hexof "$1" | sed 's/../%&/g'
}

PROTOCOL_HEX=$(hexof 'BitTorrent protocol')
INFO_HASH_HEX=$(repeat_byte aa 20)
RESERVED_HEX=0000000000100005
PEER='-XX0000-abcdefghijkl'
PEER_HEX=$(hexof "$PEER")
OTHER='-XX0000-mnopqrstuvwx'
OTHER_HEX=$(hexof "$OTHER")
AGENT='fixture-client/1.2.3'
AGENT_HEX=$(hexof "$AGENT")
LAB='-BI0000-labsidelabsi'
LAB_HEX=$(hexof "$LAB")

# One announce, in the shape a stock client sends.
announce_hex() { # escaped-peer-id [agent-header]
  hexof "GET /announce?info_hash=%aa%bb&peer_id=$1&port=51413 HTTP/1.1
Host: 127.0.0.1:1
${2-User-Agent: $AGENT
}
"
}

handshake_hex() { # peer-id-hex
  printf '13%s%s%s%s' "$PROTOCOL_HEX" "$RESERVED_HEX" "$INFO_HASH_HEX" "$1"
}

# One transcript document, in the exact shape `evidence.rs` writes.
#
# ⚠ Each argument after the endpoint is `direction:hex`, so a case can put the
# lab's own bytes in the same document as the build's - which is what the real
# peer-wire transcript always carries and is the trap this connector has to
# avoid walking into.
transcript() { # path endpoint direction:hex...
  _path="$1"
  _endpoint="$2"
  shift 2
  {
    printf '{\n'
    printf '  "schema": "bit-ids/transcript/1",\n'
    printf '  "endpoint": "%s",\n' "$_endpoint"
    printf '  "segments": [\n'
    _n=$#
    _i=0
    for _segment in "$@"; do
      _i=$((_i + 1))
      printf '    {\n'
      printf '      "connection": 1,\n'
      printf '      "offset_ms": %s,\n' "$((_i * 100))"
      printf '      "direction": "%s",\n' "${_segment%%:*}"
      printf '      "bytes": "%s"\n' "${_segment#*:}"
      if [ "$_i" = "$_n" ]; then printf '    }\n'; else printf '    },\n'; fi
    done
    printf '  ]\n}\n'
  } >"$_path"
}

# Writes one bundle. With no arguments it is the clean one every refusal below
# is measured against.
write_bundle() { # dir [tracker-segments...] -- [peer-segments...]
  _dir="$1"
  shift
  rm -rf "$_dir"
  mkdir -p "$_dir" || return 1
  _tracker=""
  while [ $# -gt 0 ] && [ "$1" != "--" ]; do
    _tracker="$_tracker $1"
    shift
  done
  [ $# -eq 0 ] || shift
  # ⚠ Unquoted on purpose: the accumulator is a space-separated argument list
  # and no value in it carries a space.
  # shellcheck disable=SC2086
  transcript "$_dir/tracker-http.transcript.json" tracker-http $_tracker
  transcript "$_dir/peer-wire-dialled.transcript.json" peer-wire-dialled "$@"
}

CLEAN_TRACKER="from_target:$(announce_hex "$(pct_of "$PEER")")"
CLEAN_PEER_LAB="to_target:$(handshake_hex "$LAB_HEX")"
CLEAN_PEER="from_target:$(handshake_hex "$PEER_HEX")"

# Runs the connector over a bundle and checks the exit code and the report.
#
# ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED, and the
# three statuses are kept apart: 0 wrote a report, 1 the evidence does not
# support one, 2 could not run. A case that read 1 and 2 as one status would
# report a guard proved by a harness that failed to start.
CASES_RUN=0
run_case() { # want-rc want-text name dir
  CASES_RUN=$((CASES_RUN + 1))
  _out="$WORK/run-$CASES_RUN.txt"
  _err="$WORK/run-$CASES_RUN.err"
  python3 "$CONNECTOR" --bundle "$4" --out "$_out" >/dev/null 2>"$_err"
  _rc=$?
  if [ "$_rc" != "$1" ]; then
    fail "$3 (exit $_rc, wanted $1)"
    return 1
  fi
  if [ -n "$2" ]; then
    if grep -q -F -e "$2" "$_err"; then
      pass "$3"
    else
      fail "$3 (the refusal did not say [$2])"
      return 1
    fi
  else
    pass "$3"
  fi
  return 0
}

reported() { # run-number field
  awk -F= -v want="$2" '$1 == want { sub(/^[^=]*=/, ""); print; exit }' \
    "$WORK/run-$1.txt"
}

# -- 1. the control -----------------------------------------------------------
#
# ⛔ EVERY REFUSAL BELOW IS WORTHLESS UNLESS THE CLEAN BUNDLE IS ACCEPTED, and
# accepted with the right VALUES. A connector that wrote four `absent` lines
# would satisfy an exit code and corroborate nothing.
write_bundle "$WORK/clean" "$CLEAN_TRACKER" -- "$CLEAN_PEER_LAB" "$CLEAN_PEER" || exit 2
run_case 0 "" "a clean bundle is read" "$WORK/clean"
CLEAN_RUN=$CASES_RUN

for want in "tracker_http/peer_id $PEER_HEX" "peer_wire/peer_id $PEER_HEX" \
  "peer_wire/reserved $RESERVED_HEX" "tracker_http/user_agent $AGENT_HEX"; do
  _field=${want%% *}
  _value=${want#* }
  if [ "$(reported "$CLEAN_RUN" "$_field")" = "$_value" ]; then
    pass "the report carries the bytes the transcript holds for $_field"
  else
    fail "the report carries the bytes the transcript holds for $_field"
  fi
done

# ⛔ AND THE LAB'S OWN HANDSHAKE IS IN THAT SAME DOCUMENT, `to_target`. A reader
# that took every segment would have reported the LAB's peer ID as the build's,
# and the report would still have four lines and a plausible shape. This is the
# case that says the direction is read.
if [ "$(reported "$CLEAN_RUN" peer_wire/peer_id)" = "$LAB_HEX" ]; then
  fail "the lab's own to_target handshake is not read as the build's"
else
  pass "the lab's own to_target handshake is not read as the build's"
fi

# -- 2. the report follows the BYTES ------------------------------------------
#
# ⭐ THE LOAD-BEARING PAIR. A connector that echoed the observer, read the
# attestation, or hardcoded anything survives every refusal in this file and
# fails exactly here.
write_bundle "$WORK/moved" \
  "from_target:$(announce_hex "$(pct_of "$OTHER")")" -- \
  "$CLEAN_PEER_LAB" "from_target:$(handshake_hex "$OTHER_HEX")" || exit 2
run_case 0 "" "a bundle carrying different bytes is read" "$WORK/moved"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$OTHER_HEX" ] &&
  [ "$(reported "$CASES_RUN" peer_wire/peer_id)" = "$OTHER_HEX" ]; then
  pass "moving the peer ID in the transcript moves it in the report"
else
  fail "moving the peer ID in the transcript moves it in the report"
fi

# ⛔ AND THE RESERVED BYTES ARE THEIR OWN FIELD. They sit next to the peer ID in
# one fixed-layout message, so an off-by-one in the offsets moves both together
# and the case above alone would not see it.
ALT_RESERVED=0000000000000004
write_bundle "$WORK/reserved" "$CLEAN_TRACKER" -- \
  "from_target:$(printf '13%s%s%s%s' "$PROTOCOL_HEX" "$ALT_RESERVED" "$INFO_HASH_HEX" "$PEER_HEX")" || exit 2
run_case 0 "" "a handshake with different reserved bytes is read" "$WORK/reserved"
if [ "$(reported "$CASES_RUN" peer_wire/reserved)" = "$ALT_RESERVED" ] &&
  [ "$(reported "$CASES_RUN" peer_wire/peer_id)" = "$PEER_HEX" ]; then
  pass "the reserved bytes and the peer ID are read at their own offsets"
else
  fail "the reserved bytes and the peer ID are read at their own offsets"
fi

# -- 3. the percent-decoding, which is where the readings can differ ----------
#
# ⛔ `+` IS THE ONE THAT SEPARATES TWO CORRECT-LOOKING READINGS. A query-string
# parser decodes `+` as a space, because that is what an HTML form means by it;
# a peer ID is an escaped byte string and `+` in one is the byte 0x2b. A
# connector built on `parse_qsl` reports a peer ID one byte wrong, agrees with
# the observer on nineteen of twenty bytes, and the record calls that a conflict
# over a build that did nothing unusual.
PLUS='-XX0000-a+b+c+d+e+f'
PLUS_HEX=$(hexof "$PLUS")
write_bundle "$WORK/plusbare" \
  "from_target:$(announce_hex "$(printf '%s' "$PLUS" | sed 's/-/%2d/g')")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "an announce leaving + bare is read" "$WORK/plusbare"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$PLUS_HEX" ]; then
  pass "a bare + in an escaped peer ID is the byte 0x2b, not a space"
else
  fail "a bare + in an escaped peer ID is the byte 0x2b, not a space"
fi

write_bundle "$WORK/plusescaped" \
  "from_target:$(announce_hex "$(pct_of "$PLUS")")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "an announce escaping + is read" "$WORK/plusescaped"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$PLUS_HEX" ]; then
  pass "an escaped and a bare + decode to the same byte"
else
  fail "an escaped and a bare + decode to the same byte"
fi

# ⚠ AND A BYTE ABOVE 0x7f, which is what a peer ID's random tail can carry and
# what a reading that went through a text decoding on the way would corrupt.
HIGH_HEX="${PEER_HEX%??}ff"
write_bundle "$WORK/high" \
  "from_target:$(announce_hex "$(printf '%s' "$HIGH_HEX" | sed 's/../%&/g')")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "an announce carrying a byte above 0x7f is read" "$WORK/high"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$HIGH_HEX" ]; then
  pass "a peer-ID byte above 0x7f survives the decoding"
else
  fail "a peer-ID byte above 0x7f survives the decoding"
fi

# -- 4. two announces, which is what a real client sends ----------------------
#
# ⭐ TRANSMISSION ANNOUNCED TWICE IN A REAL CAPTURE, so this is the shape rather
# than an invented one. ⛔ EVERY ANNOUNCE IS A SAMPLE AND EVERY SAMPLE IS
# REPORTED, in document order, whether or not two of them agree: the assembler
# compares the list position by position against the observer's.
write_bundle "$WORK/twice" \
  "from_target:$(announce_hex "$(pct_of "$PEER")")" \
  "from_target:$(announce_hex "$(pct_of "$PEER")")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "two announces that agree are read" "$WORK/twice"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$PEER_HEX,$PEER_HEX" ] &&
  [ "$(reported "$CASES_RUN" tracker_http/user_agent)" = "$AGENT_HEX,$AGENT_HEX" ]; then
  pass "two agreeing announces are two samples, not one value"
else
  fail "two agreeing announces are two samples, not one value"
fi

# ⛔ AND TWO THAT DIFFER ARE BOTH REPORTED, WHICH IS WHAT THIS FILE USED TO
# REFUSE. Both contained `capture-client` runs of 2026-10-11 put `aria2-next`
# on the wire for two sessions, each announced its own peer ID, and the old
# refusal - *two segments disagree about tracker_http/peer_id* - kept either
# capture from becoming a record. A tail that moves between sessions is
# `SCHEMA-04`'s `patterned`, and the samples are what the record needs.
write_bundle "$WORK/varies" \
  "from_target:$(announce_hex "$(pct_of "$PEER")")" \
  "from_target:$(announce_hex "$(pct_of "$OTHER")")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "two announces that differ are read rather than refused" "$WORK/varies"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$PEER_HEX,$OTHER_HEX" ]; then
  pass "two differing announces report both peer IDs, in the order they were sent"
else
  fail "two differing announces report both peer IDs, in the order they were sent"
fi

# ⛔ THE ORDER IS THE TRANSCRIPT'S, NOT A SORT. The case above passes over a
# reader that collected a sorted set, because `-XX0000-a` sorts before
# `-XX0000-m`; the same two announces sent the other way round separate the two.
# The assembler lines samples up by position, so a reordered report is a
# disagreement over a build that did nothing unusual.
write_bundle "$WORK/reversed" \
  "from_target:$(announce_hex "$(pct_of "$OTHER")")" \
  "from_target:$(announce_hex "$(pct_of "$PEER")")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "the same two announces sent the other way round are read" "$WORK/reversed"
if [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$OTHER_HEX,$PEER_HEX" ]; then
  pass "the samples follow the transcript's order rather than their own"
else
  fail "the samples follow the transcript's order rather than their own"
fi

# ⚠ AND THE PEER SURFACE IS THE SAME RULE, on the field that carries a
# per-connection tail: a build that answers two dials with two tails is two
# samples, and the lab's own handshake between them is still not one.
write_bundle "$WORK/twohandshakes" "$CLEAN_TRACKER" -- \
  "$CLEAN_PEER_LAB" "$CLEAN_PEER" "$CLEAN_PEER_LAB" \
  "from_target:$(handshake_hex "$OTHER_HEX")" || exit 2
run_case 0 "" "two handshakes with two tails are read" "$WORK/twohandshakes"
if [ "$(reported "$CASES_RUN" peer_wire/peer_id)" = "$PEER_HEX,$OTHER_HEX" ] &&
  [ "$(reported "$CASES_RUN" peer_wire/reserved)" = "$RESERVED_HEX,$RESERVED_HEX" ]; then
  pass "two handshakes report both peer IDs and both reserved blocks"
else
  fail "two handshakes report both peer IDs and both reserved blocks"
fi

# -- 5. absence is a value and it is not a refusal ----------------------------
#
# ⚠ `absent` MEANS THE CONNECTOR LOOKED AND THE FIELD WAS NOT THERE, which is a
# thing two connectors can corroborate. It is not `out_of_scope`, which is the
# reason a silence proves nothing.
write_bundle "$WORK/noagent" \
  "from_target:$(announce_hex "$(pct_of "$PEER")" "")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "an announce with no User-Agent is read" "$WORK/noagent"
if [ "$(reported "$CASES_RUN" tracker_http/user_agent)" = "absent" ] &&
  [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$PEER_HEX" ]; then
  pass "a missing header is absent and the rest of the announce still reads"
else
  fail "a missing header is absent and the rest of the announce still reads"
fi

# ⚠ AND A MISSING HEADER IS A SAMPLE TOO, IN ITS PLACE. Two announces of which
# the second carries no `User-Agent` are two samples of that field, and an
# absence that vanished from the list would line the rest up against the wrong
# announce.
write_bundle "$WORK/agentonce" \
  "from_target:$(announce_hex "$(pct_of "$PEER")")" \
  "from_target:$(announce_hex "$(pct_of "$PEER")" "")" -- "$CLEAN_PEER" || exit 2
run_case 0 "" "two announces of which one has no User-Agent are read" "$WORK/agentonce"
if [ "$(reported "$CASES_RUN" tracker_http/user_agent)" = "$AGENT_HEX,absent" ]; then
  pass "an absent header keeps its position among the samples"
else
  fail "an absent header keeps its position among the samples"
fi

write_bundle "$WORK/nohandshake" "$CLEAN_TRACKER" -- "$CLEAN_PEER_LAB" || exit 2
run_case 0 "" "a peer transcript the build never answered is read" "$WORK/nohandshake"
if [ "$(reported "$CASES_RUN" peer_wire/peer_id)" = "absent" ]; then
  pass "a handshake the build never sent is absent"
else
  fail "a handshake the build never sent is absent"
fi

# -- 6. a document this project's writer would not have emitted ---------------
#
# ⛔ THE TRANSCRIPT SCHEMA DECLARES LOWERCASE HEX. `bytes.fromhex` accepts either
# case, so being permissive here was the easier code and the wrong answer: a
# connector that read a document the lab could not have written would be
# corroborating bytes whose origin it had just declined to check.
#
# ⛔ AND THE PLANT PRESERVES THE LENGTH, WHICH IT DID NOT ON THIS FILE'S FIRST
# RUN. It inserted a digit, so the odd-length branch refused before the case
# branch was reached, and a connector planted to accept uppercase passed this
# case - the "a test whose name claims more than it checks" shape, found by
# planting rather than by reading. The two refusals are separate messages now
# and each case asserts its own.
UPPER=$(printf '%s' "$CLEAN_PEER" | sed "s/$INFO_HASH_HEX/$(printf '%s' "$INFO_HASH_HEX" | tr 'a-f' 'A-F')/")
if [ "$UPPER" = "$CLEAN_PEER" ] ||
  [ "${#UPPER}" != "${#CLEAN_PEER}" ]; then
  fail "the uppercase plant applied without changing the length"
else
  pass "the uppercase plant applied without changing the length"
  write_bundle "$WORK/upper" "$CLEAN_TRACKER" -- "$UPPER" || exit 2
  run_case 1 "is not lowercase hex" "a transcript carrying uppercase hex is refused" "$WORK/upper"
fi

write_bundle "$WORK/odd" "$CLEAN_TRACKER" -- "from_target:${PEER_HEX}a" || exit 2
run_case 1 "odd number of hex digits" "a transcript carrying odd-length hex is refused" "$WORK/odd"

write_bundle "$WORK/schema" "$CLEAN_TRACKER" -- "$CLEAN_PEER" || exit 2
sed 's|bit-ids/transcript/1|bit-ids/transcript/2|' \
  "$WORK/schema/peer-wire-dialled.transcript.json" >"$WORK/schema.json"
if cmp -s "$WORK/schema.json" "$WORK/schema/peer-wire-dialled.transcript.json"; then
  fail "the schema plant applied"
else
  pass "the schema plant applied"
  cp "$WORK/schema.json" "$WORK/schema/peer-wire-dialled.transcript.json"
  run_case 1 "not bit-ids/transcript/1" \
    "a transcript declaring another schema is refused" "$WORK/schema"
fi

write_bundle "$WORK/notjson" "$CLEAN_TRACKER" -- "$CLEAN_PEER" || exit 2
printf 'this is not a document\n' >"$WORK/notjson/tracker-http.transcript.json"
run_case 1 "is not a JSON document" "a transcript that is not JSON is refused" "$WORK/notjson"

# ⛔ A TRANSCRIPT THE BUNDLE DOES NOT CARRY IS `out_of_scope`, AND THAT IS NOT
# THE SAME FACT AS ONE THAT WILL NOT READ. A capture whose target never accepted
# a peer connection writes no peer transcript at all - which is what the stub
# capture in `check-capture-client` produces, and what made this connector refuse
# a perfectly good bundle on its first run inside the real runner.
# ⚠ `absent` WOULD BE THE WRONG ANSWER and it is the tempting one: it claims the
# condition was created and the build produced nothing, which two connectors can
# corroborate. This connector is in no position to assert that about a surface it
# never saw a record of.
write_bundle "$WORK/missing" "$CLEAN_TRACKER" -- "$CLEAN_PEER" || exit 2
rm -f "$WORK/missing/peer-wire-dialled.transcript.json"
run_case 0 "" "a bundle carrying no peer transcript is read" "$WORK/missing"
if [ "$(reported "$CASES_RUN" peer_wire/peer_id)" = "out_of_scope" ] &&
  [ "$(reported "$CASES_RUN" peer_wire/reserved)" = "out_of_scope" ] &&
  [ "$(reported "$CASES_RUN" tracker_http/peer_id)" = "$PEER_HEX" ]; then
  pass "a surface the bundle does not carry is out_of_scope, and the other one still reads"
else
  fail "a surface the bundle does not carry is out_of_scope, and the other one still reads"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/run-$CASES_RUN.txt"
fi

# ⛔ AND A TRANSCRIPT THAT IS PRESENT AND WILL NOT READ IS REFUSED. A connector
# that answered a broken bundle with `out_of_scope` would turn a defect in the
# evidence into a quiet gap in the corroboration, and `E-PUB-02` would hold the
# record back for a reason that names the wrong thing.
# ⚠ A DIRECTORY RATHER THAN A CHMOD: this harness can run as root, where a mode
# of 000 stops nothing, so a permission plant would report a guard proved by a
# read that succeeded.
write_bundle "$WORK/unreadable" "$CLEAN_TRACKER" -- "$CLEAN_PEER" || exit 2
rm -f "$WORK/unreadable/peer-wire-dialled.transcript.json"
mkdir -p "$WORK/unreadable/peer-wire-dialled.transcript.json" || exit 2
run_case 1 "peer-wire-dialled.transcript.json" \
  "a transcript that is present and will not read is refused" "$WORK/unreadable"

# -- 7. a handshake that begins right and is not one --------------------------
#
# ⛔ A TRUNCATED HANDSHAKE IS NOT AN ABSENT ONE. Reading twenty bytes from an
# offset past the end of a short segment is where a fixed-layout reader is
# wrong, and Python would answer a SHORT slice rather than an error - so the
# report would carry a peer ID of fewer than twenty bytes and nothing would say
# so.
SHORT=$(printf '13%s%s' "$PROTOCOL_HEX" "$RESERVED_HEX")
write_bundle "$WORK/short" "$CLEAN_TRACKER" -- "from_target:$SHORT" || exit 2
run_case 1 "not 68" "a segment that starts as a handshake and is short is refused" "$WORK/short"

# ⚠ AND A SEGMENT THAT IS NEITHER IS SKIPPED RATHER THAN REFUSED. A real
# transcript carries whatever the build sent after its handshake.
write_bundle "$WORK/extra" "$CLEAN_TRACKER" -- \
  "$CLEAN_PEER" "from_target:$(hexof 'not a handshake at all')" || exit 2
run_case 0 "" "a segment that is neither surface is skipped" "$WORK/extra"
if [ "$(reported "$CASES_RUN" peer_wire/peer_id)" = "$PEER_HEX" ]; then
  pass "a trailing segment does not disturb the handshake that preceded it"
else
  fail "a trailing segment does not disturb the handshake that preceded it"
fi

# -- 8. could-not-run is a third status, not a refusal ------------------------
#
# ⛔ A HARNESS EXIT OF 2 IS *COULD NOT RUN*, NEVER *REFUSED*, and this project
# has recorded a review pass counting one as the other. The connector keeps the
# two apart at its own boundary.
python3 "$CONNECTOR" >/dev/null 2>"$WORK/nobundle.err"
if [ $? = 2 ] && grep -q -F -e '--bundle is required' "$WORK/nobundle.err"; then
  pass "no --bundle at all is could-not-run"
else
  fail "no --bundle at all is could-not-run"
fi

python3 "$CONNECTOR" --bundle "$WORK/clean/tracker-http.transcript.json" \
  >/dev/null 2>"$WORK/notdir.err"
if [ $? = 2 ] && grep -q -F -e 'is not a directory' "$WORK/notdir.err"; then
  pass "a --bundle that is not a directory is could-not-run"
else
  fail "a --bundle that is not a directory is could-not-run"
fi

# -- 9. the identifier is asked for, never composed ---------------------------
#
# ⛔ THE REPORT'S FILENAME AND THE ATTESTATION'S `connectors=` MUST BE ONE NAME.
# A caller that spelled it itself would go on writing the old one the day this
# changes, and `assemble-capture` would read a file nothing wrote.
DESCRIBED=$(python3 "$CONNECTOR" --describe | awk -F= '$1 == "id" { print $2; exit }')
if [ -n "$DESCRIBED" ]; then
  pass "the connector describes its own identifier"
else
  fail "the connector describes its own identifier"
fi

# ⚠ A `Slug` is a-z0-9 separated by single hyphens, and a connector id that is
# not one is refused by `assemble-capture` a dispatch later rather than here.
case "$DESCRIBED" in
  -* | *- | *--* | *[!a-z0-9-]*) fail "the described identifier is a Slug" ;;
  '') fail "the described identifier is a Slug" ;;
  *) pass "the described identifier is a Slug" ;;
esac

# ⛔ AND THE DEFAULT OUTPUT PATH IS DERIVED FROM THAT SAME NAME. Every case above
# passes `--out`, so nothing so far has exercised the path a real capture uses.
write_bundle "$WORK/default" "$CLEAN_TRACKER" -- "$CLEAN_PEER" || exit 2
if python3 "$CONNECTOR" --bundle "$WORK/default" >/dev/null 2>&1 &&
  [ -s "$WORK/default/connector/$DESCRIBED.txt" ]; then
  pass "the default report is filed under the described identifier"
else
  fail "the default report is filed under the described identifier"
fi

# ⭐ AND EVERY FIELD THE ASSEMBLER ASKS FOR IS ON A LINE OF ITS OWN. A declared
# connector silent about a field is refused by `assemble-capture`, so a report
# short of one line is a capture that cannot become a record - found there, a
# dispatch after it could have been found here.
for field in tracker_http/peer_id tracker_http/user_agent peer_wire/peer_id \
  peer_wire/reserved; do
  if grep -q -e "^$field=" "$WORK/default/connector/$DESCRIBED.txt"; then
    pass "the report speaks about $field"
  else
    fail "the report speaks about $field"
  fi
done

store_report check-connector/1 cases "$JSON"
