#!/bin/sh
# check-capture-client.sh - prove the client capture runner refuses what it
# exists to refuse.
#
# ⛔ THE THING BEING GUARDED IS A CLAIM ABOUT A BUILD. `capture-run` attests to a
# containment; `capture-client` attests that a named product, at a named
# version, put those bytes on the wire. Every one of the extra guards it carries
# exists because the alternative is an attestation that says `kind=client` over
# a run where nothing of the kind happened.
#
# -- ⭐ THE ADAPTER IS THE FIXTURE, AND THAT IS THE POINT --------------------
#
# A real adapter installs a product, which is a capture, which needs a host this
# session is not. So the cases here drive a STUB adapter: it declares itself
# `kind=stub`, it answers `version` with a version, and its `start` announces
# with `curl` exactly as a client's own HTTP would. Every guard between the
# adapter contract and the attestation is then exercised through the same door a
# stock build comes through, and the one thing it cannot establish - that a
# stock build behaves this way - is what a dispatched capture is for.
#
# ⛔ AND THE STUB PROVES THE `stock_client` FIELD IS A COPY RATHER THAN A
# CONSTANT. One case runs a stub declaring `kind=stub` and asserts the
# attestation says `stock_client=false`; another runs the same stub declaring
# `kind=stock` and asserts it says `true`. A runner that hardcoded either would
# pass one of those and fail the other, and a harness that ran only the first
# would have believed a field nothing varied.
#
# -- ⛔ THE STUB READS THE ANNOUNCE URL OUT OF THE TORRENT --------------------
#
# It is handed a `.torrent` and nothing else, which is what a stock client is
# handed, so the address it announces to is one it decoded rather than one this
# harness told it. A stub given the URL directly would pass over an observer
# that wrote a torrent naming somewhere else entirely.
#
# Usage:
#   sh scripts/capture/check-capture-client.sh
#   sh scripts/capture/check-capture-client.sh --json
#
# Exit codes: 0 every guard refused its defect, 1 one did not, 2 could not run.
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
      printf 'check-capture-client: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# ⚠ ME is set here and read by store-lib.sh, which this sources on the next
# line. shellcheck cannot see across a source it was not told to follow.
# shellcheck disable=SC2034
ME=check-capture-client
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo curl sha256sum awk sed od

RUNNER="$ROOT/scripts/capture/capture-client.sh"
GUARD="$ROOT/scripts/acquisition/assert-disposable.sh"
for f in "$RUNNER" "$GUARD"; do
  [ -f "$f" ] || {
    printf 'check-capture-client: %s is not present\n' "$f" >&2
    exit 2
  }
done

# ⛔ THE OBSERVER IS BUILT HERE AND NOT BY THE RUNNER, which is the contract the
# workflow keeps: the build happens while there is still a network.
if ! cargo build --manifest-path "$ROOT/Cargo.toml" -p bit-ids-probe --locked \
  --example client-capture >/dev/null 2>&1; then
  printf 'check-capture-client: cannot build the client-capture example\n' >&2
  exit 2
fi
OBSERVER="${CARGO_TARGET_DIR:-$ROOT/target}/debug/examples/client-capture"
[ -x "$OBSERVER" ] || {
  printf 'check-capture-client: %s is not executable after a successful build\n' "$OBSERVER" >&2
  exit 2
}

WORK=$(store_workdir checkclient) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⚠ SHORT ON PURPOSE, and `check-capture` carries what lengthening it costs:
# measured on 2026-09-09, this harness is 48 seconds at `5` and **168 at `20`**,
# because several cases here are ones the deadline itself has to end. ⛔ That is
# why the gate runs this harness apart from its saturating batch rather than
# buying robustness with a bigger number.
SECS=5
RUN=capture-0001

# ⚠ COMPUTED, NEVER TYPED, and it is the same value the stub's default derives.
# A commit object name is forty hex digits, which is what `check-no-secrets
# --public` refuses because it is also the shape of a credential;
# `docs/security/secrets.md` says to narrow a pattern rather than switch a rule
# off, and computing the value needs no narrowing at all.
STUB_COMMIT=$(printf 'fixture-commit' | sha256sum | cut -c1-40)

# The two routing tables the egress guard reads.
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-loopback"
printf 'lo\t0000007F\t00000000\t0001\t0\t0\t0\t000000FF\t0\t0\t0\n' >>"$WORK/route-loopback"
printf 'Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n' >"$WORK/route-open"
printf 'eth0\t00000000\t010200C0\t0003\t0\t0\t0\t00000000\t0\t0\t0\n' >>"$WORK/route-open"

# -- the stub adapter ---------------------------------------------------------
#
# ⛔ EVERY TERM OF THE CONTRACT IS AN ENVIRONMENT VARIABLE, so one file provides
# every adapter defect. An adapter per case would be several files differing in
# one line each, and the day the contract grows a field only some of them would
# gain it.
#
# ⚠ The announce URL is DECODED from the torrent rather than passed in. The
# metainfo begins `d8:announce<len>:<url>`, which is the one bencode string a
# shell can read without a parser, and reading it is what makes this stub stand
# for a client rather than for a caller that already knew the address.
STUB="$WORK/stub-adapter.sh"
cat >"$STUB" <<'STUB'
#!/bin/sh
set -u
COMMAND="${1:-}"
shift 2>/dev/null || :
case "$COMMAND" in
  describe)
    [ "${BIT_IDS_STUB_DESCRIBE_RC:-0}" = 0 ] || exit "$BIT_IDS_STUB_DESCRIBE_RC"
    [ -z "${BIT_IDS_STUB_TARGET:-}" ] || printf 'target=%s\n' "$BIT_IDS_STUB_TARGET"
    [ -z "${BIT_IDS_STUB_KIND:-}" ] || printf 'kind=%s\n' "$BIT_IDS_STUB_KIND"
    [ -z "${BIT_IDS_STUB_BINARY:-}" ] || printf 'binary=%s\n' "$BIT_IDS_STUB_BINARY"
    ;;
  install)
    [ "${BIT_IDS_STUB_INSTALL_RC:-0}" = 0 ] || exit "$BIT_IDS_STUB_INSTALL_RC"
    mkdir -p "$2" || exit 2
    # ⛔ WHAT A ROUTE LEAVES IN THE INSTALL WORKDIR, WHICH IS ALSO UPLOADED.
    # A door sweep found `capture-client.yml` uploads this directory's logs as a
    # second artifact, with nothing scanning them - the capture bundle's guard
    # was a gate on one of two paths into the same rule.
    # ⚠ THE DROP MAY NAME A SUBDIRECTORY, because the scan's SCOPE is a branch
    # and a drop that could only land at the top level could not reach it. A
    # source route clones into this workdir, so the tree below it is where
    # somebody else's vendored test fixtures live.
    [ -z "${BIT_IDS_STUB_INSTALL_DROP:-}" ] || {
      mkdir -p "$(dirname "$2/$BIT_IDS_STUB_INSTALL_DROP")" || exit 2
      od -An -tx1 -N16 /dev/urandom 2>/dev/null | tr -d ' \n' \
        >"$2/$BIT_IDS_STUB_INSTALL_DROP"
    }
    printf 'route=%s\n' "$1" >"$2/installed"
    printf 'a route that hung would leave this\n' >"$2/install.log"
    # ⭐ A SOURCE ROUTE NAMES THE COMMIT IT BUILT, beside its log, because the
    # adapter is what cloned. `BIT_IDS_STUB_COMMIT` is what lets a case plant an
    # abbreviation or nothing at all, which is the shape `E-ACQ-06` refuses.
    [ "$1" = source ] &&
      printf '%s\n' "${BIT_IDS_STUB_COMMIT-$(printf 'fixture-commit' | sha256sum | cut -c1-40)}" \
        >"$2/source-commit"
    # ⭐ AND HOW THE ROUTE PACKAGED WHAT IT INSTALLED, which every shipped
    # adapter declares per route. `BIT_IDS_STUB_PACKAGE` is what lets a case
    # plant a non-slug or nothing at all.
    [ "${BIT_IDS_STUB_PACKAGE-elf-binary}" = none ] ||
      printf '%s\n' "${BIT_IDS_STUB_PACKAGE-elf-binary}" >"$2/package"
    # ⭐ THE ONE THING THAT MAKES A ROUTE AN ACQUISITION: what the product
    # answers afterwards CHANGED because this ran. When BIT_IDS_STUB_VERSION_FILE
    # names a file, `version` reads its answer out of that file and this writes
    # it, so the three shapes a host can be in are the three states of one file
    # rather than three separate flags that could disagree.
    [ -z "${BIT_IDS_STUB_VERSION_FILE:-}" ] ||
      printf '%s\n' "${BIT_IDS_STUB_VERSION-1.2.3}" >"$BIT_IDS_STUB_VERSION_FILE"
    # ⭐ AND AN INSTALL THAT REPLACES THE EXECUTABLE WITHOUT CHANGING THE VERSION.
    # That is the shape a version comparison cannot see and the one the aria2 pair
    # actually is: a package build and a vendor build both reporting 1.37.0.
    [ -z "${BIT_IDS_STUB_BINARY:-}" ] || {
      cat "$BIT_IDS_STUB_BINARY" 2>/dev/null >"$BIT_IDS_STUB_BINARY.prev"
      cat "$BIT_IDS_STUB_BINARY.prev" 2>/dev/null >"$BIT_IDS_STUB_BINARY"
      printf 'installed via %s\n' "$1" >>"$BIT_IDS_STUB_BINARY"
      chmod +x "$BIT_IDS_STUB_BINARY"
    }
    # ⛔ THE HANG DOES NOT DEPEND ON STDIN, and that matters. The caller redirects
    # /dev/null into every adapter call, so a stub that blocked on `read` would
    # return at once and prove the redirect instead of the time limit. A product
    # can hang on a lock or a socket with nothing on stdin at all.
    # ⚠ `exec`, so the process the caller's `timeout` signals is the one that is
    # blocking rather than a shell waiting on a child it would leave behind.
    [ "${BIT_IDS_STUB_HANG:-}" = install ] && exec tail -f /dev/null
    # ⭐ And this reports whether stdin reached end-of-file, which is the other
    # control and is proved by its own case rather than by this one.
    if [ "${BIT_IDS_STUB_READ_STDIN:-}" = yes ]; then
      if read -r _ignored; then printf 'stdin: read a line\n'; else printf 'stdin: eof\n'; fi
    fi
    ;;
  version)
    if [ "${BIT_IDS_STUB_VERSION_RC:-0}" != 0 ]; then
      # A real adapter says WHICH of its refusals fired, on stderr. The caller
      # is what decides whether anybody ever sees it.
      printf 'stub-adapter: the product exited 7 and here is why\n' >&2
      exit "$BIT_IDS_STUB_VERSION_RC"
    fi
    [ "${BIT_IDS_STUB_HANG:-}" = version ] && exec tail -f /dev/null
    # ⛔ A PRODUCT THAT IS NOT INSTALLED CANNOT ANSWER, and a stub that always
    # answers cannot stand for one. With the file set, absence of the file is
    # absence of the product, which is what the caller's pre-install ask reads.
    if [ -n "${BIT_IDS_STUB_VERSION_FILE:-}" ]; then
      [ -f "$BIT_IDS_STUB_VERSION_FILE" ] || {
        printf 'stub-adapter: nothing is installed on this host\n' >&2
        exit 2
      }
      cat "$BIT_IDS_STUB_VERSION_FILE"
      exit 0
    fi
    printf '%s\n' "${BIT_IDS_STUB_VERSION-1.2.3}"
    ;;
  start)
    [ "${BIT_IDS_STUB_START_RC:-0}" = 0 ] || exit "$BIT_IDS_STUB_START_RC"
    TORRENT="$1"
    # ⛔ WHAT A PRODUCT LEAVES IN THE WORKDIR, WHICH IS THE EVIDENCE BUNDLE.
    # `aria2-next` really did write a per-run RPC token there, and the workdir is
    # packed, so these three shapes stand for the class: a file NAMED like a
    # credential, a document ANSWERING with one, and a near-miss name that must
    # not be refused.
    # ⛔ THE VALUE IS GENERATED, NEVER TYPED. A typed run of hex in a tracked
    # file is the shape `check-no-secrets --public` refuses, and it is right to:
    # this harness's first draft went red on exactly that. ⭐ Generating it also
    # makes the stub do what the real product does rather than what a fixture
    # does.
    [ -z "${BIT_IDS_STUB_DROP_NAMED:-}" ] ||
      od -An -tx1 -N16 /dev/urandom 2>/dev/null | tr -d ' \n' \
        >"$2/$BIT_IDS_STUB_DROP_NAMED"
    [ -z "${BIT_IDS_STUB_DROP_VALUED:-}" ] ||
      printf '{"rpc-secret":"%s"}\n' \
        "$(od -An -tx1 -N12 /dev/urandom 2>/dev/null | tr -d ' \n')" \
        >"$2/$BIT_IDS_STUB_DROP_VALUED"
    [ "${BIT_IDS_STUB_ANNOUNCE:-yes}" = no ] && exit 0
    HEAD=$(dd if="$TORRENT" bs=1 count=200 2>/dev/null | tr -d '\000')
    REST=${HEAD#d8:announce}
    LEN=${REST%%:*}
    URL=$(printf '%s' "${REST#*:}" | cut -c1-"$LEN")
    case "$URL" in
      http://*) : ;;
      *) printf 'stub-adapter: no announce URL in %s\n' "$TORRENT" >&2; exit 1 ;;
    esac
    PEER="${BIT_IDS_STUB_PEER_ID:-stub-adapter-00000001}"
    # ⭐ A PEER ID REGENERATED PER SESSION, which is what `aria2-next` 2.8.6 put
    # on the wire on 2026-10-11 and what the second connector refused. The
    # workdir outlives a session, so a count kept there is which start this is,
    # and the last digit becomes it.
    if [ "${BIT_IDS_STUB_PEER_PER_SESSION:-}" = yes ]; then
      STARTS=$(($(cat "$2/stub-starts" 2>/dev/null || printf 0) + 1))
      printf '%s\n' "$STARTS" >"$2/stub-starts"
      PEER="${PEER%?}$STARTS"
    fi
    Q="?info_hash=%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a%5a"
    case "${BIT_IDS_STUB_ANNOUNCE:-yes}" in
      overlong)
        # A head over the codec's 64 KiB cap: bytes on the wire, no announce
        # kept. `head -c` on /dev/zero through tr is the portable way to a long
        # printable string without a loop.
        PAD=$(tr '\0' 'x' </dev/zero | head -c 70000)
        curl -sS --noproxy '*' --max-time 20 -o /dev/null -H "X-Pad: $PAD" "$URL$Q" >/dev/null 2>&1
        exit 0
        ;;
      nopeerid)
        Q="$Q&port=6881&uploaded=0&downloaded=0&left=0&compact=1&event=started"
        ;;
      *)
        Q="$Q&peer_id=$PEER&port=6881&uploaded=0&downloaded=0&left=0&compact=1&event=started"
        ;;
    esac
    curl -sS --noproxy '*' --max-time 20 -o /dev/null "$URL$Q" || exit 1
    # ⚠ A START THAT EATS THE OBSERVER'S WINDOW, planted AFTER the announce and
    # not before it. `start` is bounded at ADAPTER_SECONDS rather than at the
    # capture deadline, so a slow product really can return with nothing left to
    # stop inside - and the run is still a measurement of everything the
    # `started` announce carried. ⛔ Delaying before the announce plants a
    # different defect, a build that never announced at all, which the segment
    # guard refuses; the first draft did exactly that and the case went red
    # claiming the wrong thing.
    [ -z "${BIT_IDS_STUB_START_DELAY:-}" ] || sleep "$BIT_IDS_STUB_START_DELAY"
    ;;
  stop) exit "${BIT_IDS_STUB_STOP_RC:-0}" ;;
  *) exit 2 ;;
esac
exit 0
STUB
chmod +x "$STUB"

# -- the stub observer --------------------------------------------------------
#
# ⛔ ONE REFUSAL IS ABOUT WHAT THE OBSERVER REPORTED AND NO ADAPTER CAN PROVOKE
# IT. `the transcript does not carry the peer ID the observer reported` is the
# guard that refuses a bundle of empty artifacts verifying against their own
# empty digests, so it is planted by running the real observer and rewriting the
# peer ID it prints on the way out. ⚠ The transcript is left alone: a stub that
# edited the file would fire the digest check first, and this guard would never
# be reached.
# ⭐ AND THE SAME STUB CARRIES THE ANNOUNCE INTERVAL, which is the other run
# condition the runner reads out of this log. Each term is its own variable with
# an empty default meaning *leave it alone*, so one case plants one thing: a stub
# that always rewrote the peer ID could not reach a guard that sits before the
# peer-ID one.
STUB_OBS="$WORK/stub-observer.sh"
cat >"$STUB_OBS" <<'OBS'
#!/bin/sh
set -u
FAKE=${BIT_IDS_STUB_FAKE_PEER_HEX:-}
INTERVAL=${BIT_IDS_STUB_INTERVAL:-}
SERVING=${BIT_IDS_STUB_SERVING:-}
"$BIT_IDS_STUB_REAL" "$@" | while IFS= read -r line; do
  case "$line" in
    'announce '*' peer-id '*)
      if [ -n "$FAKE" ]; then
        printf '%s peer-id %s\n' "${line% peer-id *}" "$FAKE"
      else
        printf '%s\n' "$line"
      fi
      ;;
    'offered-interval '*)
      # `drop` removes the line entirely, which is the absent case; any other
      # value is substituted, which is the too-long one.
      case "$INTERVAL" in
        '') printf '%s\n' "$line" ;;
        drop) : ;;
        *) printf 'offered-interval %s\n' "$INTERVAL" ;;
      esac
      ;;
    'serving for '*)
      # ⚠ The prefix survives every substitution, because it is the line the
      # runner waits on before it starts the client. A stub that dropped it
      # would hang the wait rather than plant the guard below it.
      case "$SERVING" in
        '') printf '%s\n' "$line" ;;
        *) printf 'serving for %ss\n' "$SERVING" ;;
      esac
      ;;
    *) printf '%s\n' "$line" ;;
  esac
done
OBS
chmod +x "$STUB_OBS"

STATE="$WORK/state"
OUTS=0

# ⛔ THE REAL CONNECTOR, NOT A STUB. The adapter here stands in for a product
# because running one needs a disposable host; the connector needs none - it
# reads a file the observer just wrote - so substituting it would be throwing
# away the only driven pass this project has over the second reading.
CONNECTOR="$ROOT/scripts/capture/connectors/cpython-stdlib.py"
[ -f "$CONNECTOR" ] || {
  printf 'check-capture-client: %s is not present\n' "$CONNECTOR" >&2
  exit 2
}
command -v python3 >/dev/null 2>&1 || {
  printf 'check-capture-client: python3 not found\n' >&2
  exit 2
}

# -- ⛔ THE STUB CONNECTOR, AND WHY THERE HAS TO BE ONE ------------------------
#
# ⛔ SIX REFUSALS IN THE RUNNER'S CONNECTOR BLOCK HAD NEVER BEEN SEEN TO FIRE.
# Found by a guard-mutation pass on 2026-09-10: the real connector answers
# correctly, so every case that drove it exercised the ACCEPTING path, and a
# guard that has never refused is a guard nobody knows works. ⚠ Two of the eight
# were already covered - a missing `--connector` and a path resolving to nothing
# - because those refuse before anything is run.
#
# ⭐ THE SHAPE IS THE STUB ADAPTER'S, for the same reason: every term of the
# contract is an environment variable, so one file provides every connector
# defect and the day the contract grows a key only one file gains it.
STUBC="$WORK/stub-connector.py"
cat >"$STUBC" <<'STUBC'
import os
import sys

# A stand-in for the second connector, so the runner's refusals can be seen to
# refuse. It is never a measurement: nothing it writes is evidence.
name = os.environ.get('BIT_IDS_STUBC_ID', 'stub-connector')
if '--describe' in sys.argv:
    if name:
        sys.stdout.write('id=%s\nversion=0.0.0\n' % name)
    sys.exit(0)

rc = int(os.environ.get('BIT_IDS_STUBC_RC', '0'))
if rc:
    sys.stderr.write('stub-connector: refusing with %d as asked\n' % rc)
    sys.exit(rc)

bundle = sys.argv[sys.argv.index('--bundle') + 1] if '--bundle' in sys.argv else None
write = os.environ.get('BIT_IDS_STUBC_WRITE', 'full')
if write == 'none':
    sys.exit(0)
target = os.path.join(bundle, 'connector', '%s.txt' % name)
os.makedirs(os.path.dirname(target), exist_ok=True)
with open(target, 'w') as handle:
    if write == 'empty':
        handle.write('')
    elif write == 'nopeer':
        handle.write('peer_wire/peer_id=absent\n')
    else:
        handle.write('tracker_http/peer_id=00\ntracker_http/user_agent=absent\n')
        handle.write('peer_wire/peer_id=absent\npeer_wire/reserved=absent\n')
sys.exit(0)
STUBC

# -- running one case ---------------------------------------------------------
#
# ⛔ UNPIPED, AND $? READ ON THE NEXT LINE. Piping the runner into anything
# reports the pipeline's status, so a runner that failed to refuse reads as
# having refused.
# ⚠ EVERY CASE RUNS AT THE RUNNER'S OWN DEFAULT, and an attempt to run most of
# them at one session was measured and REVERTED. This harness went from 105
# seconds at 111 cases to 134 at 123, and the sessions default looked like the
# cause; running every case but the control at `--sessions 1` left it at **134
# seconds**, so it is not. ⛔ The growth is the twelve extra cases, and a harness
# complicated for a saving that was never measured is the shape this repository
# refuses. `CI-10` carries the row as a harness that grew.

run_case() { # want-code  saying  name  [extra-args...]
  _want="$1"
  _saying="$2"
  _name="$3"
  shift 3
  OUTS=$((OUTS + 1))
  CASE_OUT="$WORK/out-$OUTS"
  BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" \
    --run-id "$RUN" --out "$CASE_OUT" --observer "$OBSERVER" --adapter "$STUB" \
    --connector "$CONNECTOR" \
    --seconds "$SECS" --route-table "$WORK/route-loopback" "$@" \
    >"$WORK/out" 2>"$WORK/err"
  _got=$?
  if [ "$_got" != "$_want" ]; then
    fail "$_name (wanted exit $_want, got $_got)"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  elif [ -n "$_saying" ] && ! grep -q -F -e "$_saying" "$WORK/err"; then
    fail "$_name (exit $_want, but did not say '$_saying')"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
  else
    pass "$_name"
  fi
}

# -- the probe's own guards ---------------------------------------------------
store_probe_guards "$RUNNER" \
  "the host was never claimed" "capture-client"

# -- 1. the control, before anything is broken --------------------------------
#
# ⛔ A GUARD THAT REFUSES EVERYTHING IS NOT A GUARD. Every refusal below is
# worthless unless the clean case is accepted, and this is where the whole path
# runs: the real observer binds, writes a torrent, the stub decodes its announce
# URL and announces to it, the bundle is written and sha256sum verifies it.
rm -rf "$STATE"
BIT_IDS_STATE_DIR="$STATE" sh "$GUARD" --claim "$RUN" >/dev/null 2>&1 || {
  printf 'check-capture-client: could not claim a scratch host\n' >&2
  exit 2
}

# ⛔ EXPORTED ONCE, HERE, AND REASSIGNED PER CASE. The stub reads these out of
# its environment, so a case that set one without exporting it would run the
# stub in its default configuration and pass by testing nothing. ⚠ shellcheck
# says so as SC2034 for the ones a case only ever restores, which is the same
# defect seen from the other end.
export BIT_IDS_STUB_TARGET=stub-client
export BIT_IDS_STUB_KIND=stub
export BIT_IDS_STUB_DESCRIBE_RC=0
export BIT_IDS_STUB_VERSION_RC=0
export BIT_IDS_STUB_VERSION=1.2.3
export BIT_IDS_STUB_START_RC=0
export BIT_IDS_STUB_START_DELAY=""
export BIT_IDS_STUB_STOP_RC=0
export BIT_IDS_STUB_ANNOUNCE=yes
export BIT_IDS_STUB_PEER_ID=stub-adapter-00000001
export BIT_IDS_STUB_PEER_PER_SESSION=""
export BIT_IDS_STUB_DROP_NAMED=""
export BIT_IDS_STUB_DROP_VALUED=""
export BIT_IDS_STUB_INSTALL_DROP=""

run_case 0 "" "a claimed, contained host captures a stub client and verifies"
CONTROL_OUT="$WORK/out-$OUTS"

# ⛔ AND THE STOP IS LATE IN THE WINDOW RATHER THAN MERELY INSIDE IT. Deleting
# the wait before it leaves `stopped_within_window=yes` and a build shut down a
# moment after it launched, which is a capture of nothing.
#
# ⛔ **THE COMPARISON IS AGAINST THE DEADLINE, NEVER AGAINST THE MARGIN.** Its
# first spelling asked `stopped_after >= SECS - stop_margin`, reading the margin
# out of the attestation - so a plant setting the margin to the whole run moved
# the threshold with it and SURVIVED, exit 0. That is *a constant every test
# reads is a constant no test can check*, which `OBS-08` found twice before this.
# The invariant that matters needs no parameter: a build must be left more than
# half its own window.
#
# ⚠ The comparison is one-sided on purpose: `sleep` can return late on a loaded
# host and never early, so a floor is sound where a window would be the
# load-sensitive row `CI-08` already carries one of.
# ⚠ `-F=` and then strip the unit. A separator class of `[=s]` also splits on the
# `s` inside `stopped_after`, so `$1` is empty and every comparison below reads
# as a missing field - which is what the first draft reported.
STOPPED_AFTER=$(awk -F= '$1 == "stopped_after" { sub(/s$/, "", $2); print $2; exit }' "$CONTROL_OUT/attestation.txt")
STOP_MARGIN=$(awk -F= '$1 == "stop_margin" { sub(/s$/, "", $2); print $2; exit }' "$CONTROL_OUT/attestation.txt")
if [ -n "$STOPPED_AFTER" ] && [ "$((STOPPED_AFTER * 2))" -ge "$SECS" ]; then
  pass "the build was stopped late in the window, not at the start of it"
else
  fail "the build was stopped late in the window (after ${STOPPED_AFTER:-?}s of $SECS, margin ${STOP_MARGIN:-?}s)"
fi

# ⛔ THE ATTESTATION IS A CLAIM AND IT IS READ BACK. A runner that captured
# correctly and wrote `kind=fixture` would pass every case above.
# ⛔ `announces=2` AND `sessions_started=2` ARE WHAT SAY THE SECOND SESSION RAN.
# The stub announces once per start, so this count is the session loop's own
# receipt: it read `announces=1` until 2026-09-15, and a run that started the
# build twice and announced once would be caught by nothing else here.
for want in "kind=client" "target=stub-client" "adapter_kind=stub" \
  "stock_client=false" "measured_build=1.2.3" "announces=2" \
  "sessions=2" "sessions_started=2" "stopped_within_window=yes"; do
  if grep -q -F -e "$want" "$CONTROL_OUT/attestation.txt"; then
    pass "the attestation says $want"
  else
    fail "the attestation says $want"
  fi
done

# ⭐ AND THE PEER ID IN IT IS THE ONE THE STUB SENT, hex-encoded. A runner that
# reported its own observer's fixture identity would satisfy every count.
STUB_HEX=$(printf 'stub-adapter-00000001' | od -An -tx1 | tr -d ' \n')
if grep -q -F -e "measured_peer_id=$STUB_HEX" "$CONTROL_OUT/attestation.txt"; then
  pass "the attestation carries the peer ID the driver sent"
else
  fail "the attestation carries the peer ID the driver sent"
fi

# -- ⭐ THE SECOND CONNECTOR, DRIVEN OVER A BUNDLE THE LAB ACTUALLY WROTE -----
#
# ⛔ THIS IS THE PASS `check-connector` STRUCTURALLY CANNOT GIVE. That harness
# writes its own transcripts, in the shape the writer emits; these are the ones
# the writer emitted, from a run where a real socket carried real bytes. A
# fixture that agreed with a reader written from the same understanding is the
# self-consistency `OBS-07`'s Source names.
CONNECTOR_ID=$(python3 "$CONNECTOR" --describe | awk -F= '$1 == "id" { print $2; exit }')
REPORT="$CONTROL_OUT/bundle/connector/$CONNECTOR_ID.txt"
if [ -s "$REPORT" ]; then
  pass "the capture wrote the second connector's report into the bundle"
else
  fail "the capture wrote the second connector's report into the bundle"
fi

# ⛔ AND IT READ THE SAME BUILD THE OBSERVER DID. Two readings of one announce
# that disagree are a real finding; two that agree are the corroboration a
# record is allowed to carry. A connector reporting the OBSERVER's own fixture
# identity, or nothing, would satisfy the row above.
# ⚠ ONE SAMPLE PER ANNOUNCE, and the stub announces once per session, so the
# report carries the peer ID twice - the count `announces=2` above states.
if [ "$(awk -F= '$1 == "tracker_http/peer_id" { print $2; exit }' "$REPORT" 2>/dev/null)" = "$STUB_HEX,$STUB_HEX" ]; then
  pass "the connector read the same peer ID out of each announce as the observer"
else
  fail "the connector read the same peer ID out of each announce as the observer"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$REPORT" 2>/dev/null | head -4
fi

# ⛔ AND A BUILD WHOSE PEER ID MOVES BETWEEN SESSIONS IS CAPTURED, NOT REFUSED.
# Both contained `capture-client` runs of 2026-10-11 put `aria2-next` 2.8.6 on
# the wire for two sessions, each announced its own peer ID, and the connector
# refused with *two segments disagree about tracker_http/peer_id* - so the run
# went red and its evidence never left the host. This is that shape through the
# real observer, the real runner and the real connector.
BIT_IDS_STUB_PEER_PER_SESSION=yes
run_case 0 "" "a build that announces a different peer ID in each session is captured"
BIT_IDS_STUB_PEER_PER_SESSION=""
_first=$(printf 'stub-adapter-00000001' | od -An -tx1 | tr -d ' \n')
_second=$(printf 'stub-adapter-00000002' | od -An -tx1 | tr -d ' \n')
if [ "$(awk -F= '$1 == "tracker_http/peer_id" { print $2; exit }' \
  "$CASE_OUT/bundle/connector/$CONNECTOR_ID.txt" 2>/dev/null)" = "$_first,$_second" ]; then
  pass "and the connector reports both sessions' peer IDs, in the order they were sent"
else
  fail "and the connector reports both sessions' peer IDs, in the order they were sent"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$CASE_OUT/bundle/connector/$CONNECTOR_ID.txt" 2>/dev/null | head -4
fi

# ⛔ THE ATTESTATION DECLARES IT, WHICH IS THE KEY `assemble-capture` READS.
# A report in the bundle that no attestation names is a file the assembler never
# opens, and the record it would write declares one connector and is refused at
# the validity gate.
if grep -q -F -e "connectors=$CONNECTOR_ID" "$CONTROL_OUT/attestation.txt"; then
  pass "the attestation declares the second connector by the name it filed under"
else
  fail "the attestation declares the second connector by the name it filed under"
fi

# ⛔ AND EACH CONNECTOR'S VERSION, MEASURED. The record carried `0.0.0` for both
# until 2026-10-14; the observer prints its crate's version and the connector
# describes its own, and the attestation is where the assembler reads them.
_want_connector=$(python3 -I "$ROOT/scripts/capture/connectors/cpython-stdlib.py" --describe |
  awk -F= '$1 == "version" { print $2; exit }')
if grep -q -x -E 'observer_version=[0-9]+\.[0-9]+\.[0-9]+' "$CONTROL_OUT/attestation.txt" &&
  grep -q -x -F -e "connector_version=$_want_connector" "$CONTROL_OUT/attestation.txt"; then
  pass "the attestation carries the observer's and the connector's own versions"
else
  fail "the attestation carries the observer's and the connector's own versions"
fi

# ⛔ AND THE REPORT IS COVERED BY THE DOCUMENT THAT COVERS THE BUNDLE. Exactly
# one published file may be covered by neither of a publication's two documents;
# a capture bundle carrying a second uncovered file is a downloader verifying
# three artifacts and trusting a fourth.
if grep -q -F -e "bundle/connector/$CONNECTOR_ID.txt" "$CONTROL_OUT/SHA256SUMS" &&
  (cd "$CONTROL_OUT" && sha256sum -c SHA256SUMS >/dev/null 2>&1); then
  pass "the connector's report is covered by SHA256SUMS and the bundle still verifies"
else
  fail "the connector's report is covered by SHA256SUMS and the bundle still verifies"
fi

# -- ⛔ RULE 12, AT THE CHOKE POINT ------------------------------------------
#
# ⛔ MEASURED ON A REAL DISPATCH, NOT IMAGINED. `capture-client` run 11 shipped
# `client/rpc-token` inside its uploaded artifact: `aria2-next` generated a
# per-run RPC token, wrote it where `stop` could read it, and this runner packed
# the directory it was in. Neither step was wrong alone, no reading found it, and
# `check-no-secrets` cannot see it - the bundle is not in the tree it scans.
# ⚠ The adapter's own unlink is a gate on ONE path: it needs every adapter to
# remember, `stop` to be reached, and `stop` to succeed - and a non-zero `stop`
# is recorded here rather than refused.
BIT_IDS_STUB_DROP_NAMED=rpc-token
run_case 1 "name says it holds a credential" \
  "a capture that would ship a file named like a credential is refused"
BIT_IDS_STUB_DROP_NAMED=""

# ⛔ AND THE ONE NO FILENAME RULE REACHES. An options dump the product ANSWERED
# with names its own `rpc-secret`, and the file it lands in is called nothing in
# particular.
BIT_IDS_STUB_DROP_VALUED=rpc-options.json
run_case 1 "carrying a secret-shaped value" \
  "a capture that would ship a document carrying a secret is refused"
BIT_IDS_STUB_DROP_VALUED=""

# ⛔ AND THE CONTROL BOTH REFUSALS NEED, which is a NEAR MISS rather than a clean
# run. `tokenizer` contains `token` and is not one; a rule that refused it would
# be a rule somebody switches off, and a rule that refused everything would pass
# both cases above.
BIT_IDS_STUB_DROP_NAMED=tokenizer.log
run_case 0 "" "a file whose name merely contains the word is not refused"
BIT_IDS_STUB_DROP_NAMED=""

# -- ⛔ THE SCAN ITSELF, ASKED DIRECTLY ---------------------------------------
#
# ⛔ `scan-bundle.sh` IS THE ONE HOME FOR BOTH RULES, and three callers ask it:
# this runner, `install-client`, and the workflow step that scans a REFUSED
# capture before its evidence is uploaded. That third caller reaches it with no
# runner around it, so what it answers alone is what these cases hold.
SCAN="$ROOT/scripts/capture/scan-bundle.sh"
SCANDIR="$WORK/scan"
rm -rf "$SCANDIR"
mkdir -p "$SCANDIR/deep/er" || exit 2
printf 'nothing here\n' >"$SCANDIR/observer.log"

sh "$SCAN" "$SCANDIR" >/dev/null 2>"$WORK/scan.err"
_clean=$?
if [ "$_clean" = 0 ]; then
  pass "scan      a directory holding nothing rule 12 forbids is clean"
else
  fail "scan      a directory holding nothing rule 12 forbids is clean"
fi

# ⛔ A PATH THAT IS NOT THERE IS COULD-NOT-RUN. A scan over nothing finds
# nothing, and a workflow reading 0 would upload whatever a mistyped path did not
# name.
sh "$SCAN" "$WORK/there-is-no-bundle-here" >/dev/null 2>"$WORK/scan.err"
if [ $? = 2 ] && grep -q -F -e 'is not a directory' "$WORK/scan.err"; then
  pass "scan      a directory that is not there is could-not-run, not clean"
else
  fail "scan      a directory that is not there is could-not-run, not clean"
fi

# ⚠ BOTH RULES ARE ASKED AND BOTH ARE REPORTED. A scan that stopped at the first
# would cost a dispatch per secret.
od -An -tx1 -N16 /dev/urandom | tr -d ' \n' >"$SCANDIR/rpc-token"
printf '{"rpc-secret":"%s"}\n' "$(od -An -tx1 -N12 /dev/urandom | tr -d ' \n')" \
  >"$SCANDIR/options.json"
sh "$SCAN" "$SCANDIR" >/dev/null 2>"$WORK/scan.err"
if [ $? = 1 ] && grep -q -F -e 'name says it holds a credential' "$WORK/scan.err" &&
  grep -q -F -e 'carrying a secret-shaped value' "$WORK/scan.err"; then
  pass "scan      a credential-named file and a secret-shaped value are both reported"
else
  fail "scan      a credential-named file and a secret-shaped value are both reported"
fi
# ⛔ AND THE VALUE IS NEVER PRINTED. The file is named; what is in it is not.
if grep -q -F -e "$(cut -d'"' -f4 "$SCANDIR/options.json")" "$WORK/scan.err"; then
  fail "scan      the secret-shaped value is named by its file and not printed"
else
  pass "scan      the secret-shaped value is named by its file and not printed"
fi
rm -f "$SCANDIR/rpc-token" "$SCANDIR/options.json"

# ⚠ THE SCOPE IS THE CALLER'S, in both directions. A bundle ships whole and is
# scanned whole; an install workdir ships its top level, and `--top-level` must
# stop reading below it and only below it.
od -An -tx1 -N16 /dev/urandom | tr -d ' \n' >"$SCANDIR/deep/er/server.key"
sh "$SCAN" "$SCANDIR" >/dev/null 2>"$WORK/scan.err"
_whole=$?
sh "$SCAN" --top-level "$SCANDIR" >/dev/null 2>"$WORK/scan.err"
_top=$?
if [ "$_whole" = 1 ] && [ "$_top" = 0 ]; then
  pass "scan      a file below the top level is read whole and not with --top-level"
else
  fail "scan      a file below the top level is read whole ($_whole) and not with --top-level ($_top)"
fi

# -- ⛔ THE CONNECTOR REFUSALS, DRIVEN THROUGH A STUB -------------------------
#
# ⛔ EVERY ONE OF THESE HAD NEVER FIRED. The real connector answers correctly, so
# the cases above exercise only the accepting path, and a guard that has never
# been seen to refuse is a guard nobody knows works.
# ⚠ EXPORTED, because the runner invokes the stub through `python3` in a child
# process; a variable set without `export` would run the stub in its default
# configuration and the case would pass by testing nothing.
export BIT_IDS_STUBC_ID=stub-connector
export BIT_IDS_STUBC_RC=0
export BIT_IDS_STUBC_WRITE=full

# ⭐ THE CONTROL FIRST. Every refusal below passes equally over a runner that
# refuses every stub, so this establishes the stub itself is acceptable.
run_case 0 "" "a capture driven by an acceptable stub connector is accepted" \
  --connector "$STUBC"

BIT_IDS_STUBC_ID=""
run_case 2 "did not describe its own identifier" \
  "a connector that names no identifier is could-not-run" --connector "$STUBC"

# ⚠ AN UNDERSCORE, which is the shape a slug refuses and a filename accepts, so
# the report would be written where `assemble-capture` never looks.
BIT_IDS_STUBC_ID=stub_connector
run_case 2 "which is not a slug" \
  "a connector naming itself something that is not a slug is could-not-run" \
  --connector "$STUBC"
BIT_IDS_STUBC_ID=stub-connector

BIT_IDS_STUBC_RC=3
run_case 2 "could not run (exit 3)" \
  "a connector that could not run is could-not-run" --connector "$STUBC"
BIT_IDS_STUBC_RC=0

# ⛔ EXIT 1 IS THE CONNECTOR REFUSING THE EVIDENCE, WHICH IS A DIFFERENT FACT
# from a connector that could not run, and the two reach this runner through one
# channel. A case reading the code alone would pass when the other fired.
BIT_IDS_STUBC_RC=1
run_case 1 "could not read what the observer recorded" \
  "a connector that refuses the evidence refuses the capture" --connector "$STUBC"
BIT_IDS_STUBC_RC=0

BIT_IDS_STUBC_WRITE=none
run_case 1 "exited 0 and wrote no report" \
  "a connector that exits 0 and writes nothing is refused" --connector "$STUBC"

BIT_IDS_STUBC_WRITE=empty
run_case 1 "exited 0 and wrote no report" \
  "a connector whose report is empty is refused" --connector "$STUBC"

# ⛔ AND A REPORT THAT IS PRESENT, NON-EMPTY AND SILENT ABOUT THE ONE FIELD THAT
# IDENTIFIES THE BUILD. `assemble-capture` refuses a declared connector silent on
# a field a dispatch later; this refuses it while the host still exists.
BIT_IDS_STUBC_WRITE=nopeer
run_case 1 "says nothing about tracker_http/peer_id" \
  "a report silent on the field that identifies the build is refused" \
  --connector "$STUBC"
BIT_IDS_STUBC_WRITE=full

# ⛔ AND A CAPTURE WITHOUT ONE DOES NOT RUN AT ALL. `run_case` always passes
# `--connector`, so the two refusals below are invoked directly. ⚠ They are
# `could not run` rather than refusals: nothing about the host or the build is
# wrong, the caller asked for a capture that cannot become a record.
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" \
  --run-id "$RUN" --out "$WORK/no-connector" --observer "$OBSERVER" \
  --adapter "$STUB" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >/dev/null 2>"$WORK/noconn.err"
NOCONN_RC=$?
if [ "$NOCONN_RC" = 2 ] &&
  grep -q -F -e 'cannot become a record' "$WORK/noconn.err"; then
  pass "a capture asked for with no connector is could-not-run"
else
  fail "a capture asked for with no connector is could-not-run (exit $NOCONN_RC)"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/noconn.err" | head -3
fi

BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" \
  --run-id "$RUN" --out "$WORK/gone-connector" --observer "$OBSERVER" \
  --adapter "$STUB" --connector "$WORK/there-is-no-connector-here.py" \
  --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >/dev/null 2>"$WORK/goneconn.err"
GONECONN_RC=$?
if [ "$GONECONN_RC" = 2 ] &&
  grep -q -F -e 'is not present' "$WORK/goneconn.err"; then
  pass "a connector path that resolves to nothing is could-not-run"
else
  fail "a connector path that resolves to nothing is could-not-run (exit $GONECONN_RC)"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/goneconn.err" | head -3
fi

# -- 2. stock_client is copied from the adapter, not assumed ------------------
BIT_IDS_STUB_KIND=stock
run_case 0 "" "an adapter declaring kind=stock is taken at its word"
if grep -q -F -e "stock_client=true" "$WORK/out-$OUTS/attestation.txt"; then
  pass "stock_client follows the adapter's declaration"
else
  fail "stock_client follows the adapter's declaration"
fi
BIT_IDS_STUB_KIND=stub

# -- 3. the adapter contract --------------------------------------------------
BIT_IDS_STUB_DESCRIBE_RC=3
run_case 2 "could not describe itself" "an adapter that cannot describe itself is could-not-run"
BIT_IDS_STUB_DESCRIBE_RC=0

BIT_IDS_STUB_TARGET=""
run_case 2 "named no target" "an adapter naming no target is refused"
BIT_IDS_STUB_TARGET=stub-client

BIT_IDS_STUB_KIND=""
run_case 2 "declared no kind" "an adapter declaring no kind is refused"

BIT_IDS_STUB_KIND=product
run_case 2 "unknown kind" "an adapter declaring an unknown kind is refused"
BIT_IDS_STUB_KIND=stub

BIT_IDS_STUB_VERSION_RC=4
run_case 2 "could not ask the installed build its version" \
  "an adapter that cannot ask the build its version is could-not-run"
BIT_IDS_STUB_VERSION_RC=0

BIT_IDS_STUB_VERSION=""
run_case 2 "empty version" "an empty version is refused"

# ⚠ A TAB, which is not printable text and is exactly what a version parsed out
# of a column would carry. A record field holding one is a value no reader can
# compare, and it fails no length or emptiness check.
BIT_IDS_STUB_VERSION="$(printf '1.2\t3')"
run_case 2 "not printable text" "a version carrying a control byte is refused"
BIT_IDS_STUB_VERSION=1.2.3

BIT_IDS_STUB_START_RC=5
run_case 1 "could not start the build" "an adapter that cannot start the build is refused"
BIT_IDS_STUB_START_RC=0

# ⛔ AND A START THAT EATS THE WINDOW IS RECORDED RATHER THAN REFUSED. A run with
# nothing left to stop inside measured nothing about a shutdown, and a build that
# sends no `stopped` announce reads identically without this field. ⚠ The run is
# otherwise green: everything the `started` announce measured is still a
# measurement, and throwing it away over a shutdown nobody heard would be losing
# a capture to a detail of the driver.
BIT_IDS_STUB_START_DELAY=$((SECS + 2))
run_case 0 "" "a start that outlives the window still captures"
if grep -q -F -e "stopped_within_window=no" "$WORK/out-$OUTS/attestation.txt"; then
  pass "and the attestation says the shutdown was not observed"
else
  fail "and the attestation says the shutdown was not observed"
fi
BIT_IDS_STUB_START_DELAY=""

# -- 4. what makes it a measurement -------------------------------------------
#
# ⛔ THE TWO CASES THAT SAY A BUILD WAS MEASURED. A run where nothing announced
# and a run where only the observer's own identity announced are separate
# refusals, because the fixes differ: the first is a client that never started
# and the second is an observer talking to itself.
# ⚠ SILENCE FIRES THE SEGMENT GUARD, NOT THIS ONE, AND THAT IS CORRECT. A build
# that never started leaves no bytes at all. The case is written against what
# actually happens rather than against what the name suggests, because a case
# asserting a message a different guard produced is how two guards over one
# input come to mask each other.
BIT_IDS_STUB_ANNOUNCE=no
run_case 1 "the run recorded no bytes at all" \
  "a build that put nothing on the wire is refused by the segment guard"

# ⛔ AND THIS IS THE ANNOUNCE GUARD, REACHED THROUGH A CONNECTION THAT SPOKE.
# The head is over the codec's 64 KiB cap, so the request is refused and not
# kept as an observation while the bytes are still recorded: segments above
# zero, announces zero. Found by this harness on its own first run, where the
# case above passed for the wrong reason.
BIT_IDS_STUB_ANNOUNCE=overlong
run_case 1 "announced nothing" \
  "a connection that reached the tracker without announcing is refused"

# ⛔ AN ANNOUNCE WITH NO PEER ID IS A THIRD CASE. It is not the observer's own
# identity and it is not a build's either, and the earlier version of this
# runner searched the transcript for the literal word `absent`.
BIT_IDS_STUB_ANNOUNCE=nopeerid
run_case 1 "no announce carried a readable peer ID" \
  "an announce carrying no peer ID at all is refused"
BIT_IDS_STUB_ANNOUNCE=yes

BIT_IDS_STUB_PEER_ID=bit-ids-fixture-0001
run_case 1 "every announce carried this observer's own peer ID" \
  "an announce carrying only the observer's own peer ID is refused"
BIT_IDS_STUB_PEER_ID=stub-adapter-00000001

# ⛔ AND THE REPORTED PEER ID MUST BE IN THE BYTES. This is the guard that
# refuses a bundle of empty artifacts verifying against their own empty digests.
#
# ⚠ THE FAKE IS DERIVED FROM A PEER ID RATHER THAN TYPED AS HEX. A forty-digit
# literal here is indistinguishable to `check-no-secrets --public` from a
# credential, and the derived form also says in the tree what it is: the
# encoding of a peer nobody sent.
FAKE_PEER_HEX=$(printf 'never-on-this-wire-x' | od -An -tx1 | tr -d ' \n')
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" BIT_IDS_STUB_REAL="$OBSERVER" \
  BIT_IDS_STUB_FAKE_PEER_HEX="$FAKE_PEER_HEX" \
  sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB_OBS" \
  --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "the transcript does not carry the peer ID" "$WORK/err"; then
  pass "a peer ID the transcript does not hold is refused"
else
  fail "a peer ID the transcript does not hold is refused (exit $_rc)"
fi

# -- ⛔ 4b. the run must be able to outlive the interval it answered -----------
#
# ⛔ `capture-client` run 18 answered 60 under a 45-second deadline, so no
# re-announce was ever due and every `tracker_http/*` field rested on one sample.
# ⚠ BOTH ARE exit 2 RATHER THAN 1. A condition of the experiment that was set
# before the build did anything is *could not run*, and reporting it as the
# build's behaviour would name the wrong thing - which is the distinction this
# repository keeps confusing in the other direction.
#
# ⭐ The control is the case above and every case before it: they run the real
# observer, whose interval is derived from the deadline, and they pass.
OUTS=$((OUTS + 1))
BIT_IDS_STUB_LONGER=$((SECS + 1))
BIT_IDS_STATE_DIR="$STATE" BIT_IDS_STUB_REAL="$OBSERVER" \
  BIT_IDS_STUB_INTERVAL="$BIT_IDS_STUB_LONGER" \
  sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB_OBS" \
  --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "no re-announce can fall due" "$WORK/err"; then
  pass "an interval longer than the run is refused"
else
  fail "an interval longer than the run is refused (exit $_rc)"
fi

# ⚠ AND THE INTERVAL EXACTLY EQUAL TO THE DEADLINE IS ACCEPTED, because the
# derivation's floor is five seconds and this harness runs a five-second
# capture: there is no shorter interval to ask for. ⛔ Without this case the
# refusal above could be tightened to `<` and every correct short run would be
# refused - which is what the first draft of that guard did, and the whole
# harness went red on a tree with no defect in it.
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" BIT_IDS_STUB_REAL="$OBSERVER" \
  BIT_IDS_STUB_INTERVAL="$SECS" \
  sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB_OBS" \
  --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 0 ]; then
  pass "an interval equal to the deadline is accepted, because the floor is there"
else
  fail "an interval equal to the deadline is accepted (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" BIT_IDS_STUB_REAL="$OBSERVER" \
  BIT_IDS_STUB_INTERVAL=drop \
  sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB_OBS" \
  --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "reported no announce interval" "$WORK/err"; then
  pass "an observer that reports no announce interval is refused"
else
  fail "an observer that reports no announce interval is refused (exit $_rc)"
fi

# ⛔ AND THE DEADLINE IS COMPARED AGAINST THE ONE THIS RUN PASSED. The interval
# guard above is read against what the observer says it served for, so an
# observer serving a different span would make that comparison true over the
# wrong pair of numbers.
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" BIT_IDS_STUB_REAL="$OBSERVER" \
  BIT_IDS_STUB_SERVING=$((SECS * 100)) \
  sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" --observer "$STUB_OBS" \
  --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" --route-table "$WORK/route-loopback" \
  >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "it was given" "$WORK/err"; then
  pass "an observer serving a span it was not given is refused"
else
  fail "an observer serving a span it was not given is refused (exit $_rc)"
fi

# -- 5. the host state guards -------------------------------------------------
#
# ⚠ THE SAME GUARDS capture-run CARRIES, ASKED AGAIN HERE. They are not
# inherited: this is a separate script, and a step order that dropped a claim or
# a route deletion would land on this runner just as readily.
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-open" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "egress is open" "$WORK/err"; then
  pass "an open default route is refused"
else
  fail "an open default route is refused (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/no-such-table" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "egress could not be established" "$WORK/err"; then
  pass "an unreadable route table is not a pass"
else
  fail "an unreadable route table is not a pass (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id capture-0002 --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "is claimed by run [$RUN], not [capture-0002]" "$WORK/err"; then
  pass "a host claimed by another run is refused"
else
  fail "a host claimed by another run is refused (exit $_rc)"
fi

# ⚠ An EMPTY state directory, not the populated one. This is the step-order
# defect: a workflow that dropped its claim step leaves the capture running on a
# host nothing ever claimed.
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$WORK/unclaimed" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "the host was never claimed" "$WORK/err"; then
  pass "an unclaimed host is refused"
else
  fail "an unclaimed host is refused (exit $_rc)"
fi

# -- 6. the argument guards ---------------------------------------------------
OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$WORK/no-such-binary" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "is not executable" "$WORK/err"; then
  pass "an observer that was never built is could-not-run"
else
  fail "an observer that was never built is could-not-run (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$WORK/no-such-adapter" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "is not present" "$WORK/err"; then
  pass "an adapter that is not there is could-not-run"
else
  fail "an adapter that is not there is could-not-run (exit $_rc)"
fi

# ⛔ A SECOND CAPTURE MUST NOT WRITE INTO THE FIRST ONE'S EVIDENCE.
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$CONTROL_OUT" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 1 ] && grep -q -F -e "already exists" "$WORK/err"; then
  pass "an output directory holding another run is refused"
else
  fail "an output directory holding another run is refused (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "Capture-0001" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "run id must be lowercase" "$WORK/err"; then
  pass "a run id that is not a slug is refused"
else
  fail "a run id that is not a slug is refused (exit $_rc)"
fi

OUTS=$((OUTS + 1))
BIT_IDS_STATE_DIR="$STATE" sh "$RUNNER" --run-id "$RUN" --out "$WORK/out-$OUTS" \
  --observer "$OBSERVER" --adapter "$STUB" --connector "$CONNECTOR" --seconds "$SECS" --peer-port "six" \
  --route-table "$WORK/route-loopback" >"$WORK/out" 2>"$WORK/err"
_rc=$?
if [ "$_rc" = 2 ] && grep -q -F -e "--peer-port must be a whole number" "$WORK/err"; then
  pass "a peer port that is not a number is refused"
else
  fail "a peer port that is not a number is refused (exit $_rc)"
fi

# -- 7. the adapters this repository ships ------------------------------------
#
# ⛔ EVERY TRACKED ADAPTER ANSWERS `describe`, AND THE ANSWER IS CHECKED. An
# adapter is only reached by a dispatched capture, so a typo in one would
# otherwise be found by a runner rather than by a gate. ⚠ Nothing here installs
# or starts anything: `describe` is the one subcommand that needs no product.
for adapter in "$ROOT"/scripts/capture/adapters/*.sh; do
  _name=$(basename "$adapter" .sh)
  _desc=$(sh "$adapter" describe 2>"$WORK/err")
  _rc=$?
  _target=$(printf '%s\n' "$_desc" | awk -F= '$1 == "target" { print $2; exit }')
  _kind=$(printf '%s\n' "$_desc" | awk -F= '$1 == "kind" { print $2; exit }')
  if [ "$_rc" = 0 ] && [ -n "$_target" ] && [ "$_kind" = stock ]; then
    pass "adapter $_name describes itself as stock $_target"
  else
    fail "adapter $_name describes itself as stock (exit $_rc, target [$_target], kind [$_kind])"
  fi
  if sh "$adapter" no-such-subcommand >/dev/null 2>&1; then
    fail "adapter $_name refuses an unknown subcommand"
  else
    pass "adapter $_name refuses an unknown subcommand"
  fi
done

# -- 8. the install step, which runs before the route is cut -------------------
#
# ⛔ THE OTHER HALF OF THE SAME VERTICAL, AND ITS GUARDS ARE DIFFERENT ONES.
# `install-client` runs with the network still up, so it asserts the CLAIM and
# deliberately does not assert egress: a guard there would refuse every host the
# step is meant to run on. ⚠ Nothing here installs a product; the stub adapter's
# `install` writes a file, which is enough to drive every refusal the step owns.
INSTALLER="$ROOT/scripts/acquisition/install-client.sh"
if [ ! -f "$INSTALLER" ]; then
  fail "install-client is present"
else
  pass "install-client is present"

  install_case() { # want-code  saying  name  extra-args...
    _want="$1"
    _saying="$2"
    _name="$3"
    shift 3
    OUTS=$((OUTS + 1))
    BIT_IDS_STATE_DIR="$STATE" sh "$INSTALLER" "$@" >"$WORK/out" 2>"$WORK/err"
    _got=$?
    if [ "$_got" != "$_want" ]; then
      fail "$_name (wanted exit $_want, got $_got)"
      [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
    elif [ -n "$_saying" ] && ! grep -q -F -e "$_saying" "$WORK/err"; then
      fail "$_name (exit $_want, but did not say '$_saying')"
      [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
    else
      pass "$_name"
    fi
  }

  install_case 0 "" "a claimed host installs through the package route" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-ok" --record "$WORK/inst-ok.txt"

  # ⛔ RULE 12 ON THIS PATH TOO, WHICH A DOOR SWEEP FOUND ON 2026-09-10.
  # `capture-client.yml` uploads this workdir's logs as a second artifact, and
  # the credential scan added to the capture runner earlier the same day covered
  # only the bundle - a gate on one of two paths into the same rule.
  BIT_IDS_STUB_INSTALL_DROP=registry-token
  install_case 1 "name says it holds a credential" \
    "an install that would ship a file named like a credential is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-secret" \
    --record "$WORK/inst-secret.txt"

  # ⚠ AND THE NEAR MISS, because a rule that refused everything would pass the
  # case above. `tokenizer.log` contains the word and is not one.
  BIT_IDS_STUB_INSTALL_DROP=tokenizer.log
  install_case 0 "" \
    "an install whose file merely contains the word is not refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-nearmiss" \
    --record "$WORK/inst-nearmiss.txt"

  # ⛔ AND THE SCOPE, WHICH IS THE BRANCH A REAL CAPTURE FOUND. The scan reads
  # this workdir's TOP LEVEL, because that is what `capture-client.yml` uploads:
  # `install-<route>/*.log` and `*.err`, globs with no `**`. Run 15 on
  # 2026-09-15 refused its `source` lane over openssl's and nghttp2's vendored
  # test fixtures, `src/third_party/openssl/apps/server.pem` and
  # `src/third_party/nghttp2/integration-tests/server.key`, which the route
  # clones and nothing ships.
  #
  # ⚠ BOTH DIRECTIONS ARE PLANTED, because they are different branches and a
  # scan that stopped descending entirely would pass this one and lose the case
  # above. The same name at the top level is still refused; below it is not.
  BIT_IDS_STUB_INSTALL_DROP=src/third_party/openssl/apps/server.pem
  install_case 0 "" \
    "a credential-named file in a subtree no artifact ships is not refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-subtree" \
    --record "$WORK/inst-subtree.txt"

  BIT_IDS_STUB_INSTALL_DROP=server.pem
  install_case 1 "name says it holds a credential" \
    "the same name at the workdir's top level, which does ship, is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-toplevel" \
    --record "$WORK/inst-toplevel.txt"
  BIT_IDS_STUB_INSTALL_DROP=""

  # ⛔ THE RECORD IS A CLAIM AND IT IS READ BACK. A step that installed and
  # wrote nothing would pass every exit-code case above.
  for want in "target=stub-client" "route=package" "reported_version=1.2.3"; do
    if grep -q -F -e "$want" "$WORK/inst-ok.txt" 2>/dev/null; then
      pass "the install record says $want"
    else
      fail "the install record says $want"
    fi
  done

  # -- ⛔ WHETHER THE ROUTE ACTUALLY ACQUIRED ANYTHING -------------------------
  #
  # ⛔ A ROUTE IS CONTRACTED TO INSTALL THE TARGET AND NOTHING ESTABLISHED THAT
  # IT DID. `aria2` ships on the `ubuntu-24.04` image, so client capture runs 3
  # and 4 wrote `route=package` over an `apt-get install` that printed `already
  # the newest version` and installed nothing. Two such routes declare two
  # independent resolvers, agree on the version because it is one binary, and
  # reach `ACQ-03` as `byte_identical` - the strongest verdict there is, over an
  # acquisition that did not happen.
  #
  # ⚠ THREE SHAPES, AND THE MIDDLE ONE IS THE ONLY FAILURE. Absent then present
  # is an install; a different version is an upgrade and is also an install; the
  # same version over a target that was already there is neither. A check that
  # refused any preexisting build would refuse the upgrade too, which is the
  # ordinary case of an index carrying more than the image.
  # ⚠ THE FILE IS THE HOST'S STATE and each case owns its own, so a case that
  # left one behind cannot decide the next one's verdict.
  export BIT_IDS_STUB_VERSION_FILE="$WORK/host-fresh"
  rm -f "$BIT_IDS_STUB_VERSION_FILE"
  install_case 0 "" "an absent target installed by the route is an acquisition" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-a" --record "$WORK/inst-a.txt"
  for want in "preexisting_version=" "acquired=yes" "reported_version=1.2.3"; do
    if grep -q -x -F -e "$want" "$WORK/inst-a.txt" 2>/dev/null; then
      pass "an install onto a bare host records $want"
    else
      fail "an install onto a bare host records $want"
    fi
  done

  # ⛔ THE ARIA2 SHAPE, REPRODUCED. The host already answers the version the
  # route would have installed, so the route installs nothing and the record must
  # say so rather than reporting an acquisition.
  BIT_IDS_STUB_VERSION_FILE="$WORK/host-present"
  printf '1.2.3\n' >"$BIT_IDS_STUB_VERSION_FILE"
  install_case 0 "installed nothing" \
    "a target the host already had is recorded as no acquisition" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-b" --record "$WORK/inst-b.txt"
  for want in "preexisting_version=1.2.3" "acquired=no"; do
    if grep -q -x -F -e "$want" "$WORK/inst-b.txt" 2>/dev/null; then
      pass "a route that installed nothing records $want"
    else
      fail "a route that installed nothing records $want"
    fi
  done

  # ⛔ AND AN UPGRADE IS AN ACQUISITION. A guard that read any preexisting build
  # as a no-op would refuse this, which is the ordinary case of a package index
  # carrying a newer build than the image ships.
  BIT_IDS_STUB_VERSION_FILE="$WORK/host-old"
  printf '1.2.2\n' >"$BIT_IDS_STUB_VERSION_FILE"
  install_case 0 "" "a route that upgrades what the host had is an acquisition" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-c" --record "$WORK/inst-c.txt"
  for want in "preexisting_version=1.2.2" "reported_version=1.2.3" "acquired=yes"; do
    if grep -q -x -F -e "$want" "$WORK/inst-c.txt" 2>/dev/null; then
      pass "an upgrade records $want"
    else
      fail "an upgrade records $want"
    fi
  done
  unset BIT_IDS_STUB_VERSION_FILE

  # ⛔ SAME VERSION, DIFFERENT EXECUTABLE, AND THAT IS AN ACQUISITION. A version
  # comparison alone cannot reach this branch, and it is the one the real pair
  # needs: `aria2` ships on `ubuntu-24.04` at the version the vendor publishes,
  # so a release route there installs a genuinely different build and reports the
  # same number. ⚠ Without this case the digest half of the verdict would be
  # written and never exercised, which is a guard nobody knows works.
  BIT_IDS_STUB_VERSION_FILE="$WORK/host-same"
  export BIT_IDS_STUB_VERSION_FILE
  printf '1.2.3\n' >"$BIT_IDS_STUB_VERSION_FILE"
  BIT_IDS_STUB_BINARY="$WORK/stub-build"
  export BIT_IDS_STUB_BINARY
  printf 'the build that was already here\n' >"$BIT_IDS_STUB_BINARY"
  chmod +x "$BIT_IDS_STUB_BINARY"
  install_case 0 "" "one version over two executables is still an acquisition" \
    --adapter "$STUB" --route release --workdir "$WORK/inst-d" --record "$WORK/inst-d.txt"
  for want in "preexisting_version=1.2.3" "reported_version=1.2.3" "acquired=yes"; do
    if grep -q -x -F -e "$want" "$WORK/inst-d.txt" 2>/dev/null; then
      pass "a replaced executable records $want"
    else
      fail "a replaced executable records $want"
    fi
  done
  # ⛔ AND THE TWO DIGESTS IT COMPARED ARE IN THE RECORD, DIFFERENT. A verdict
  # whose evidence is absent is a verdict nobody can re-derive.
  _d1=$(sed -n 's/^preexisting_binary_sha256=//p' "$WORK/inst-d.txt")
  _d2=$(sed -n 's/^installed_binary_sha256=//p' "$WORK/inst-d.txt")
  if [ -n "$_d1" ] && [ -n "$_d2" ] && [ "$_d1" != "$_d2" ]; then
    pass "the record carries the two digests the verdict compared"
  else
    fail "the record carries the two digests the verdict compared [$_d1] [$_d2]"
  fi
  unset BIT_IDS_STUB_BINARY
  unset BIT_IDS_STUB_VERSION_FILE

  # -- ⛔ THE SELF-CHECK, PROVED BY PLANTING THE DEFECT IT EXISTS TO CATCH ------
  #
  # ⛔ THE DERIVING PATH AND THE CHECKING PATH AGREE BY CONSTRUCTION, so no input
  # this caller accepts can ever make its own verdict wrong. That is the shape
  # `reviews.md` calls a refusal nothing tests, and the only way to reach it is
  # to break the derivation and run the broken copy. ⚠ Asserting the same
  # re-derivation here in the harness would test the harness against itself and
  # would pass with the guard deleted.
  #
  # ⚠ THE COPY KEEPS ITS DIRECTORY SHAPE, because the script finds the host guard
  # by walking up two levels from its own path. A copy dropped anywhere else
  # exits 2 for a missing guard, which is could-not-run and would have been read
  # as a refusal.
  # ⚠ And the rule-12 scan it asks, under the same shape, for the same reason.
  mkdir -p "$WORK/planted/scripts/acquisition" "$WORK/planted/scripts/capture"
  sed 's/^  ACQUIRED=no$/  ACQUIRED=yes/' "$INSTALLER" \
    >"$WORK/planted/scripts/acquisition/install-client.sh"
  cp "$GUARD" "$WORK/planted/scripts/acquisition/assert-disposable.sh"
  cp "$ROOT/scripts/capture/scan-bundle.sh" "$WORK/planted/scripts/capture/scan-bundle.sh"
  if cmp -s "$INSTALLER" "$WORK/planted/scripts/acquisition/install-client.sh"; then
    fail "the acquired-verdict plant applied to install-client"
  else
    pass "the acquired-verdict plant applied to install-client"
    BIT_IDS_STUB_VERSION_FILE="$WORK/host-planted"
    export BIT_IDS_STUB_VERSION_FILE
    printf '1.2.3\n' >"$BIT_IDS_STUB_VERSION_FILE"
    OUTS=$((OUTS + 1))
    BIT_IDS_STATE_DIR="$STATE" sh "$WORK/planted/scripts/acquisition/install-client.sh" \
      --adapter "$STUB" --route package --workdir "$WORK/inst-plant" \
      --record "$WORK/inst-plant.txt" >"$WORK/out" 2>"$WORK/err"
    _rc=$?
    if [ "$_rc" = 1 ] && grep -q -F -e "says acquired=[yes] over preexisting [1.2.3]" "$WORK/err"; then
      pass "a caller whose verdict does not follow from its own record is refused"
    else
      fail "a caller whose verdict does not follow from its own record is refused (exit $_rc)"
      [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/err" | head -3
    fi
    unset BIT_IDS_STUB_VERSION_FILE
  fi

  # ⚠ THE SECOND ROUTE IS A SECOND INVOCATION, and it writes its own record.
  # `ACQ-03` compares two of these, so a step that merged them would leave
  # nothing to compare.
  install_case 0 "" "the release route is a separate invocation with its own record" \
    --adapter "$STUB" --route release --workdir "$WORK/inst-rel" --record "$WORK/inst-rel.txt"
  if grep -q -F -e "route=release" "$WORK/inst-rel.txt" 2>/dev/null &&
    ! grep -q -F -e "route=package" "$WORK/inst-rel.txt" 2>/dev/null; then
    pass "each route's record names that route alone"
  else
    fail "each route's record names that route alone"
  fi

  # ⛔ THE SOURCE ROUTE'S IMMUTABLE IDENTITY, WHICH `E-ACQ-06` NEEDS AND NOTHING
  # RECORDED UNTIL 2026-09-09. Assembling capture-client run 14 refused both
  # lanes because no document the capture path writes carried a commit; the
  # object name was in the git chatter inside `install.log` and nowhere a reader
  # could ask for it. ⚠ The empty case below is the other half: a field that is
  # absent for every other route reads as a field that is sometimes forgotten.
  install_case 0 "" "a source route records the commit its adapter named" \
    --adapter "$STUB" --route source --workdir "$WORK/inst-src" --record "$WORK/inst-src.txt"
  if grep -q -F -e "source_commit=$STUB_COMMIT" "$WORK/inst-src.txt" 2>/dev/null; then
    pass "the record carries the full object name the route built"
  else
    fail "the record carries the full object name the route built"
  fi
  if grep -q -F -e "source_commit=" "$WORK/inst-rel.txt" 2>/dev/null; then
    pass "a route that cloned nothing records an empty commit rather than omitting the field"
  else
    fail "a route that cloned nothing records an empty commit rather than omitting the field"
  fi

  # ⛔ AN ABBREVIATION IS THE DEFECT `E-ACQ-06` EXISTS FOR, and `rev-parse` will
  # print one when asked. Without this case the guard passes equally over a check
  # that only asks whether the field is non-empty.
  BIT_IDS_STUB_COMMIT=a9784ea8 install_case 1 "is not a full object name" \
    "a source route naming an abbreviated commit is refused" \
    --adapter "$STUB" --route source --workdir "$WORK/inst-src-short"
  BIT_IDS_STUB_COMMIT='' install_case 1 "recorded no full commit object name" \
    "a source route naming no commit is refused" \
    --adapter "$STUB" --route source --workdir "$WORK/inst-src-none"

  # ⛔ THE PACKAGE FORMAT, WHICH IS PART OF THE IDENTITY TUPLE A STORE PATH IS
  # DERIVED FROM. Nothing wrote it until 2026-09-09, and `assemble-capture`
  # refused both of capture-client run 14's lanes for it. ⚠ A route that
  # installed and did not say how is refused here rather than at an assembly a
  # dispatch later.
  if grep -q -F -e "package=elf-binary" "$WORK/inst-src.txt" 2>/dev/null; then
    pass "the record carries the package format the route declared"
  else
    fail "the record carries the package format the route declared"
  fi
  BIT_IDS_STUB_PACKAGE=none install_case 1 "recorded no package format" \
    "a route that installed and did not say how it packaged the build is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-nopkg"
  BIT_IDS_STUB_PACKAGE='Elf Binary' install_case 1 "which is not a slug" \
    "a package format that is not a slug is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-badpkg"

  install_case 2 "unknown route" "a route outside the closed vocabulary is refused" \
    --adapter "$STUB" --route mirror --workdir "$WORK/inst-bad"

  install_case 2 "is not present" "an adapter that is not there is could-not-run" \
    --adapter "$WORK/no-such-adapter" --route package --workdir "$WORK/inst-none"

  # ⛔ THE TIME LIMIT, WHICH THE FIRST CLIENT DISPATCH BOUGHT. Two of its three
  # jobs sat in the install step for over half an hour and reported nothing,
  # because a hung install is indistinguishable from a slow one until the job's
  # own timeout kills the runner and takes the log with it. ⚠ 124 is coreutils'
  # verdict for "it never answered" and is a separate refusal from a route that
  # said no, because the fixes differ.
  export BIT_IDS_STUB_HANG=install
  BIT_IDS_INSTALL_TIMEOUT=2 install_case 1 "is hung, not slow" \
    "a route that never answers is refused rather than waited on" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-hang"
  # ⭐ And the route's own log is printed with the refusal, because the workdir
  # is only uploaded when the capture that follows succeeds.
  if grep -q -F -e "a route that hung would leave this" "$WORK/err"; then
    pass "the refusal carries the route's own log"
  else
    fail "the refusal carries the route's own log"
  fi

  BIT_IDS_STUB_HANG=version
  BIT_IDS_VERSION_TIMEOUT=2 install_case 1 "did not answer --version within" \
    "a build that never answers --version is refused rather than waited on" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-hangv"
  BIT_IDS_STUB_HANG=

  # ⛔ THE SECOND CONTROL, PROVED SEPARATELY. The time limit above catches a
  # product that hangs for any reason; this catches the specific one the first
  # dispatch is most likely to have met, a product asking a question. An adapter
  # call reads /dev/null, so a prompt gets end-of-file at once rather than a
  # wait, and the case asserts the stub SAW that rather than asserting a run that
  # finished quickly.
  export BIT_IDS_STUB_READ_STDIN=yes
  install_case 0 "" "an adapter that reads stdin gets end-of-file, not a wait" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-stdin"
  if grep -q -F -e 'stdin: eof' "$WORK/out"; then
    pass "the adapter's stdin is at end-of-file rather than open"
  else
    fail "the adapter's stdin is at end-of-file rather than open"
  fi
  BIT_IDS_STUB_READ_STDIN=

  BIT_IDS_STUB_INSTALL_RC=1
  export BIT_IDS_STUB_INSTALL_RC
  install_case 1 "did not install" "a route that refuses is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-fail"
  BIT_IDS_STUB_INSTALL_RC=2
  install_case 2 "could not run" "a route that could not run is could-not-run" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-cannot"
  BIT_IDS_STUB_INSTALL_RC=0

  # ⛔ AN INSTALL WHOSE EXECUTABLE CANNOT ANSWER IS A ROUTE THAT DID NOT WORK,
  # and finding that out under containment would be finding it out too late.
  BIT_IDS_STUB_VERSION_RC=3
  install_case 1 "would not report a version" \
    "an install whose build will not report a version is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-noversion"
  # ⛔ AND THE ADAPTER'S OWN MESSAGE REACHES THE LOG. The caller read the
  # adapter's stderr into /dev/null and then reported that no version arrived,
  # which is silencing a diagnosis and complaining about its absence. Measured
  # on 2026-09-08 by a client capture whose cause could not be read at all.
  if grep -q -F -e "the product exited 7 and here is why" "$WORK/err"; then
    pass "the refusal carries what the adapter said"
  else
    fail "the refusal carries what the adapter said"
  fi
  BIT_IDS_STUB_VERSION_RC=0

  BIT_IDS_STUB_VERSION=""
  install_case 1 "reported an empty version" \
    "an install reporting an empty version is refused" \
    --adapter "$STUB" --route package --workdir "$WORK/inst-empty"
  BIT_IDS_STUB_VERSION=1.2.3

  # ⚠ An EMPTY state directory. A workflow that installed before claiming would
  # put a product on a host nothing had established was disposable.
  OUTS=$((OUTS + 1))
  BIT_IDS_STATE_DIR="$WORK/unclaimed" sh "$INSTALLER" --adapter "$STUB" \
    --route package --workdir "$WORK/inst-unclaimed" >"$WORK/out" 2>"$WORK/err"
  _rc=$?
  if [ "$_rc" = 1 ] && grep -q -F -e "the host was never claimed" "$WORK/err"; then
    pass "installing on an unclaimed host is refused"
  else
    fail "installing on an unclaimed host is refused (exit $_rc)"
  fi
fi

store_report "check-capture-client/1" case "$JSON"
