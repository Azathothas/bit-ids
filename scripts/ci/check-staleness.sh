#!/bin/sh
# check-staleness.sh - drive CI-02's monitor over real stores and real
# resolutions, and read back what it opened.
#
# ⛔ EVERY CASE GOES THROUGH THE RESOLVER. The entry's Prove asks for no request
# over a preview, and a harness that handed the survey a stable version and
# observed no preview request would prove nothing at all: no preview was
# present. So each case writes the release list a source would answer with, runs
# resolve-stable over it, and surveys what that decided.
#
# ⛔ THE INPUTS ARE PLANTED, NOT THE SOURCE. check-store.sh and check-corpus
# plant into a tree because their guards are about trees. This unit's guards are
# about a comparison, so the plant is a different store, a different release
# list or a different tracker state, and every case declares the exit code and
# the verdict it expects. The source-level guard mutation over staleness.rs is
# part (c) of the gate and is recorded in TODO/ci.md.
#
# ⭐ CASE 1 AND CASE 2 ARE EACH OTHER'S CONTROL. They differ only in whether the
# tracker is handed the request the first one opened. A detection that never
# fired would fail case 2; one that always fired would fail case 1. Neither is
# provable from one run.
#
# -- ⚠ WHY THIS IS IN scripts/ci/ AND IN THE GATE -----------------------------
#
# check-workflow.sh is the other file here and is deliberately NOT in the gate,
# because two of its cases run the gate. This one runs no gate and re-enters
# nothing, so it is in check-gate.sh's prover list with the corpus and
# publishing harnesses.
#
# Usage:
#   sh scripts/ci/check-staleness.sh
#   sh scripts/ci/check-staleness.sh --json
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
      printf 'check-staleness: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# ⚠ ME is read by store-lib.sh, which this sources on the next line. shellcheck
# cannot follow a source it was not handed, so the directive is in this file
# rather than left to how CI groups its arguments.
# shellcheck disable=SC2034
ME=check-staleness
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

# ⚠ python3 is required rather than optional, and it earns that: it is the
# independent implementation the request identifier is checked against, and a
# constant this project wrote from its own encoder is not a control. It is
# present on ubuntu-24.04, which is the lane that runs the gate with --strict.
store_require cargo sha256sum python3
BUILDER=$(store_build "$ROOT" build-store) || exit 2
RESOLVER=$(store_build "$ROOT" resolve-stable) || exit 2
SURVEYOR=$(store_build "$ROOT" survey-staleness) || exit 2

WORK=$(store_workdir checkstaleness) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# A store carrying the given versions of the fixture target.
# ⚠ build-store's own fixture version is 0.0.0-fixture, which no numeric scheme
# orders, so every store here names its versions.
make_store() { # dir version...
  _dir="$1"
  shift
  rm -rf "$_dir"
  mkdir -p "$_dir" || return 1
  _args=""
  for _v in "$@"; do
    _args="$_args --version $_v"
  done
  # shellcheck disable=SC2086
  "$BUILDER" $_args "$_dir" >/dev/null 2>&1
}

# A release list, as a source would answer it, then the resolution over it.
# Each tag is TAG or TAG@prerelease.
make_resolution() { # name tag...
  _name="$1"
  shift
  _body="$WORK/$_name.releases.json"
  {
    printf '[\n'
    _first=1
    _day=1
    for _spec in "$@"; do
      _tag=${_spec%@prerelease}
      _pre=false
      [ "$_tag" = "$_spec" ] || _pre=true
      [ "$_first" = "1" ] || printf ',\n'
      _first=0
      printf '  {"tag_name": "%s", "prerelease": %s, "draft": false, ' "$_tag" "$_pre"
      printf '"published_at": "2026-0%s-02T00:00:00Z"}' "$_day"
      _day=$((_day + 1))
    done
    printf '\n]\n'
  } >"$_body"
  printf '2026-09-04T12:00:00Z\n' >"$_body.fetched-at"
  # ⛔ Unpiped. resolve-stable exits 1 when it fails closed, which is a case
  # below rather than a harness failure, so the status is kept and not read.
  # ⚠ `github-releases` rather than `upstream`, because the source id names the
  # FORMAT the body is read as and these fixture bodies are release listings.
  # It was `upstream` until 2026-09-09, when a second source format arrived and
  # the id became what chooses the reader.
  "$RESOLVER" fixture-client - 3 3 github-releases https://example.invalid/releases \
    "$_body" >"$WORK/$_name.resolution.json" 2>/dev/null
  [ -s "$WORK/$_name.resolution.json" ]
}

# ⛔ Unpiped: the survey writes to a file and $? is read on the next line.
run_survey() { # store resolution-name out [open-file]
  if [ $# -ge 4 ]; then
    "$SURVEYOR" "$1" --resolution "$WORK/$2.resolution.json" --platforms linux \
      --open "$4" "$WORK/$3.json" >"$WORK/$3.out" 2>&1
  else
    "$SURVEYOR" "$1" --resolution "$WORK/$2.resolution.json" --platforms linux \
      "$WORK/$3.json" >"$WORK/$3.out" 2>&1
  fi
  RC=$?
}

# One field out of a survey, read with a parser rather than with grep, because a
# key that appears in two objects is what a line-oriented read gets wrong.
field() { # file python-expression
  python3 -c "
import json,sys
d=json.load(open(sys.argv[1]))
print($2)
" "$1" 2>/dev/null
}

# A case: name, expected exit code, expected verdict, expected request count.
expect() { # name rc-expected verdict-expected requests-expected survey-name
  _name="$1"
  _rc="$2"
  _verdict="$3"
  _count="$4"
  _file="$WORK/$5.json"
  if [ "$RC" != "$_rc" ]; then
    fail "$_name  expected exit $_rc, got $RC: $(head -2 "$WORK/$5.out" | tr '\n' ' ')"
    return 0
  fi
  _got_verdict=$(field "$_file" "d['assessments'][0]['staleness']")
  _got_count=$(field "$_file" "len(d['requests'])")
  if [ "$_got_verdict" != "$_verdict" ]; then
    fail "$_name  expected verdict $_verdict, got ${_got_verdict:-none}"
    return 0
  fi
  if [ "$_got_count" != "$_count" ]; then
    fail "$_name  expected $_count request(s), got ${_got_count:-none}"
    return 0
  fi
  pass "$_name  exit $_rc, $_verdict, $_count request(s)"
}

STORE="$WORK/store"
if ! make_store "$STORE" 1.2.3 1.2.10; then
  printf 'check-staleness: cannot build the fixture store\n' >&2
  exit 2
fi
EMPTY="$WORK/empty"
mkdir -p "$EMPTY"

for spec in "moved 1.2.3 1.2.10 1.3.0" "known 1.2.3 1.2.10" \
  "preview 1.2.3 1.2.10 1.3.0-beta1 1.4.0@prerelease" \
  "behind 1.2.3" "blocked 1.2.10 1.2"; do
  # shellcheck disable=SC2086
  if ! make_resolution $spec; then
    printf 'check-staleness: cannot resolve %s\n' "${spec%% *}" >&2
    exit 2
  fi
done

# ⛔ CASE 1. A stable release newer than anything measured opens exactly one
# request, and the exit code says so without a JSON parser.
run_survey "$STORE" moved first
expect "new      a new stable release opens one request" 1 stale 1 first
FIRST_ID=$(field "$WORK/first.json" "d['requests'][0]['id']")
FIRST_STATE=$(field "$WORK/first.json" "d['requests'][0]['state']")
if [ "$FIRST_STATE" = "opened" ]; then
  pass "new      and the request is in the state a caller acts on"
else
  fail "new      the request is ${FIRST_STATE:-absent}, not opened"
fi

# ⭐ THE STRONGEST CONTROL HERE IS NOT THIS PROJECT'S CODE. The identifier is
# re-derived from the encoding restated in staleness.rs, by a SHA-256 this
# project did not write. A survey compared against its own encoder agrees with
# itself.
DERIVED=$(python3 -c "
import hashlib, struct
parts = ['bit-ids/capture-request/1', 'bit-ids/capture-requests/1',
         'fixture-client', '1.3.0', 'stable', 'linux']
buf = b''.join(struct.pack('>I', len(p.encode())) + p.encode() for p in parts)
print('request:sha256:' + hashlib.sha256(buf).hexdigest())
")
if [ -n "$FIRST_ID" ] && [ "$DERIVED" = "$FIRST_ID" ]; then
  pass "derived  an independent SHA-256 re-derives the request identifier"
else
  fail "derived  independent derivation gives $DERIVED, the survey gives ${FIRST_ID:-none}"
fi

# ⛔ CASE 2, WHICH IS CASE 1 WITH THE TRACKER HOLDING WHAT IT OPENED. The Prove's
# third clause. Same identifier, a state a caller does not act on, exit 0.
python3 -c "
import json,sys
d=json.load(open(sys.argv[1]))
json.dump([{k: r[k] for k in ('target','version','channel','platform')}
           for r in d['requests']], open(sys.argv[2],'w'), indent=2)
" "$WORK/first.json" "$WORK/open.json"
run_survey "$STORE" moved second "$WORK/open.json"
expect "repeat   a second run over the same facts opens nothing" 0 stale 1 second
SECOND_ID=$(field "$WORK/second.json" "d['requests'][0]['id']")
SECOND_STATE=$(field "$WORK/second.json" "d['requests'][0]['state']")
if [ "$SECOND_ID" = "$FIRST_ID" ] && [ "$SECOND_STATE" = "already_open" ]; then
  pass "repeat   one identifier across both runs, and it is already_open"
else
  fail "repeat   id ${SECOND_ID:-none} state ${SECOND_STATE:-none}, expected $FIRST_ID already_open"
fi

# ⚠ And a third run is the second one's bytes. Two runs agreeing is what a
# scheduled monitor rests on; the clock is the one field that moves.
run_survey "$STORE" moved third "$WORK/open.json"
if [ "$RC" != "0" ]; then
  fail "repeat   the third run exited $RC"
elif python3 -c "
import json,sys
a=json.load(open(sys.argv[1])); b=json.load(open(sys.argv[2]))
del a['surveyed_at']; del b['surveyed_at']
sys.exit(0 if a==b else 1)
" "$WORK/second.json" "$WORK/third.json"; then
  pass "repeat   a third run is the second one's document apart from the clock"
else
  fail "repeat   two runs over one tracker state produced different documents"
fi

# ⛔ CASE 3. The source published a preview by each of the two stability signals
# and the resolver refused both, so the selection is the release already
# measured and nothing is opened.
run_survey "$STORE" preview preview
expect "preview  a preview by either signal opens nothing" 0 current 0 preview
SELECTED=$(field "$WORK/preview.json" "d['assessments'][0]['selected']")
if [ "$SELECTED" = "1.2.10" ]; then
  pass "preview  and the selection is the newest stable, not the preview"
else
  fail "preview  the survey selected ${SELECTED:-none}, not 1.2.10"
fi

run_survey "$STORE" known known
expect "known    a release already measured opens nothing" 0 current 0 known

# ⛔ CASE 4. Nothing measured at all is the first capture rather than a silence.
run_survey "$EMPTY" moved fresh
expect "first    an unmeasured platform opens its first request" 1 uncaptured 1 fresh

# ⛔ CASE 5. A measurement newer than the selection opens nothing: a request here
# would ask a runner to capture a downgrade.
run_survey "$STORE" behind behind
expect "behind   a selection older than what is measured opens nothing" 0 regressed 0 behind

# ⛔ CASE 6. The resolver failed closed. Reported rather than skipped, because a
# target blocked for a month otherwise looks like a target with no work.
run_survey "$STORE" blocked blocked
expect "blocked  a resolution that failed closed is reported" 0 unresolved 0 blocked

# ⛔ CASE 7. The release moved past a request the tracker still holds. One
# request at the new version, and the old one retired rather than left beside it.
python3 -c "
import json,sys
json.dump([{'target':'fixture-client','version':'1.2.11','channel':'stable',
            'platform':'linux'}], open(sys.argv[1],'w'), indent=2)
" "$WORK/stale-open.json"
run_survey "$STORE" moved retired "$WORK/stale-open.json"
expect "retired  a request the release moved past is superseded" 1 stale 1 retired
RETIRED=$(field "$WORK/retired.json" "len(d['superseded'])")
RETIRED_VERSION=$(field "$WORK/retired.json" "d['requests'][0]['version']")
if [ "$RETIRED" = "1" ] && [ "$RETIRED_VERSION" = "1.3.0" ]; then
  pass "retired  one identifier retired, and the request asks for 1.3.0"
else
  fail "retired  ${RETIRED:-none} retired, request at ${RETIRED_VERSION:-none}"
fi

# ⚠ CASE 8. A tracker file present and unreadable is could-not-run, never an
# empty tracker. Reading it as empty opens every request again, which is the
# duplicate this entry exists to prevent arriving through an I/O error.
printf 'not json\n' >"$WORK/broken-open.json"
run_survey "$STORE" moved broken "$WORK/broken-open.json"
if [ "$RC" = "2" ]; then
  pass "tracker  an unreadable tracker is exit 2, not an empty tracker"
else
  fail "tracker  expected exit 2 over an unreadable tracker, got $RC"
fi

# ⚠ CASE 9. A platform list with no resolution before it is refused rather than
# applied to the run, because a list that applied to every target would ask for
# a capture on a platform a target does not ship on.
"$SURVEYOR" "$STORE" --platforms linux "$WORK/orphan.json" >"$WORK/orphan.out" 2>&1
RC=$?
if [ "$RC" = "2" ]; then
  pass "args     --platforms with no --resolution before it is refused"
else
  fail "args     expected exit 2, got $RC: $(head -2 "$WORK/orphan.out" | tr '\n' ' ')"
fi

"$SURVEYOR" "$STORE" --resolution "$WORK/moved.releases.json" --platforms linux \
  "$WORK/notaresolution.json" >"$WORK/notaresolution.out" 2>&1
RC=$?
if [ "$RC" = "2" ]; then
  pass "args     a document that is not a resolution is refused"
else
  fail "args     expected exit 2 over a release list, got $RC"
fi

# ⚠ THE HARNESS'S OWN READER IS CHECKED. Every case above rests on `field`
# answering from the document; a reader that answered nothing would turn every
# comparison into a mismatch, which reads as a failing guard rather than as a
# broken harness. These two say it reads a value that is there and none that is
# not.
if [ "$(field "$WORK/first.json" "d['schema']")" = "bit-ids/capture-requests/1" ]; then
  pass "probe    the harness reader answers from the document"
else
  fail "probe    the harness reader cannot read the survey it just wrote"
fi
if [ -z "$(field "$WORK/first.json" "d['no-such-key']")" ]; then
  pass "probe    and answers nothing for a key the document does not carry"
else
  fail "probe    the harness reader invented a value"
fi

store_report check-staleness/1 cases "$JSON"
