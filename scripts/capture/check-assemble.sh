#!/bin/sh
# check-assemble.sh - can a dispatched capture's artifacts become a record, and
# does the assembler refuse every way they can fall short?
#
# ⛔ WHAT THIS EXISTS FOR IS A SET OF REFUSALS NOBODY HAD SEEN. `CI-09` waited on
# a two-route capture for weeks; `capture-client` run 14 produced one, and
# assembling it on 2026-09-09 refused for reasons no reading had found. None of
# them is visible in an attestation and all of them are visible the moment
# something tries to write the record.
#
# -- ⭐ IT RUNS OVER A SYNTHETIC PAIR AND SAYS SO -----------------------------
#
# The lanes here are written by this harness, not downloaded. A gate case that
# fetched two artifacts from a run would go red the day that run expired, and
# would be measuring GitHub's retention rather than this project's assembler.
# ⚠ SO THIS PROVES THE ASSEMBLER AND NOTHING ABOUT RUN 14. What run 14 answered
# is a driven pass, recorded in `TODO/ci.md` under `CI-09`.
#
# ⚠ NO VALUE HERE IS ANY PRODUCT'S. The peer-ID prefix is `-XX0000-`, which no
# vendor uses, so nothing this harness writes can be mistaken for a measurement.
# `docs/architecture.md` section 5 says why a fixture is never evidence.
#
# ⛔ NOTHING HERE PLANTS IN A TRACKED FILE. Every lane a case mutates is written
# into scratch state, because a gate check that edits a tracked file leaves a
# dirty tree behind when it is interrupted.
#
# Usage:
#   sh scripts/capture/check-assemble.sh
#   sh scripts/capture/check-assemble.sh --json
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
      printf 'check-assemble: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-assemble
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo sha256sum od
ASSEMBLE=$(store_build "$ROOT" assemble-capture bit-ids-probe) || exit 2
VALIDATE_RUN=$(store_build "$ROOT" validate-run bit-ids) || exit 2

WORK=$(store_workdir checkassemble) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⛔ EVERY HEX RUN HERE IS COMPUTED, NEVER TYPED, and that is a rule rather than
# a preference. An info hash and a peer ID are both forty hex digits, which is
# the shape `check-no-secrets --public` refuses because it is also the shape of
# a credential. `docs/security/secrets.md` says to narrow a pattern rather than
# switch a rule off; computing the value narrows nothing and needs no exclusion.
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

PROTOCOL_HEX=$(hexof 'BitTorrent protocol')
INFO_HASH_HEX=$(repeat_byte aa 20)
RESERVED_HEX=0000000000100005

# One announce, in the shape a stock client sends and `tracker_http` parses.
#
# ⚠ A SECOND ARGUMENT ADDS `event=`, and `started` is what delimits a session.
# BEP 3 makes it the first announce of a run, so a second one in a transcript is
# a second separately started process - which is the only thing that separates
# `Lifetime::PerSession` from `Lifetime::PerConnection`. ⛔ Omitting it is the
# default because every fixture here predates sessions, and a helper that always
# wrote one would have silently re-attributed every existing case.
announce_hex() { # peer-id [event]
  if [ -n "${2:-}" ]; then
    hexof "GET /announce?info_hash=$(printf 'X%.0s' 1)&peer_id=$1&port=51413&event=$2 HTTP/1.1
Host: 127.0.0.1:1
User-Agent: fixture-client/1.2.3

"
    return
  fi
  hexof "GET /announce?info_hash=$(printf 'X%.0s' 1)&peer_id=$1&port=51413 HTTP/1.1
Host: 127.0.0.1:1
User-Agent: fixture-client/1.2.3

"
}

# ⛔ A SCRAPE: a request the build sends on the same surface that is NOT an
# announce, because it carries no `peer_id`. ⚠ The tree contains no such segment
# and neither did this harness, so the assembler's rule for one - sample the
# announces, pass over the rest - survived a mutation pass with every case green.
# That is the blind spot `docs/methodology/reviews.md` names: a corpus only tests
# the defects it holds an example of.
scrape_hex() {
  hexof "GET /scrape?info_hash=$(printf 'X%.0s' 1) HTTP/1.1
Host: 127.0.0.1:1
User-Agent: fixture-client/1.2.3

"
}

# One peer handshake, in the shape `peer_wire` parses: a length byte, the
# protocol name, eight reserved bytes, the info hash and the peer ID.
handshake_hex() { # peer-id
  printf '13%s%s%s%s' \
    "$PROTOCOL_HEX" "$RESERVED_HEX" "$INFO_HASH_HEX" "$(hexof "$1")"
}

# One transcript document, in the exact shape `evidence.rs` writes and its
# reader accepts.
#
# ⭐ That reader is the writer's inverse and refuses anything the writer does not
# emit, so a document typed here that it accepts is a document the lab would
# have written. A harness whose fixtures the real reader rejected would prove
# nothing about a real bundle.
# ⚠ ONE SEGMENT PER PAYLOAD, because a transcript records a segment per
# connection and a field rests on every one of them. This took a single hex
# argument until 2026-09-15, so every fixture here had exactly one sample of
# every field and the assembler's sampling could not be exercised at all.
transcript() { # path endpoint hex...
  _path=$1
  _endpoint=$2
  shift 2
  _total=$#
  {
    printf '{\n'
    printf '  "schema": "bit-ids/transcript/1",\n'
    printf '  "endpoint": "%s",\n' "$_endpoint"
    printf '  "segments": [\n'
    _n=0
    for _hex in "$@"; do
      _n=$((_n + 1))
      printf '    {\n'
      printf '      "connection": %s,\n' "$_n"
      printf '      "offset_ms": %s,\n' "$((2000 + _n * 75))"
      printf '      "direction": "from_target",\n'
      printf '      "bytes": "%s"\n' "$_hex"
      if [ "$_n" = "$_total" ]; then printf '    }\n'; else printf '    },\n'; fi
    done
    printf '  ]\n}\n'
  } >"$_path"
}

# Writes one lane's two artifact directories.
#
# ⚠ Every value a lane carries is a parameter, so a case can move exactly one
# and the rest stay fixed. A helper that hardcoded the fields would need a copy
# per case, and the copies are what drift.
write_lane() { # tag route resolver-url commit binary-digest version peer-id [connector-peer-id]
  _cap="$WORK/$1-capture"
  _ins="$WORK/$1-install"
  rm -rf "$_cap" "$_ins"
  mkdir -p "$_cap/capture/bundle/fixture" "$_ins/install-$2" "$_ins/release" || return 1

  {
    printf 'bit-ids/capture-attestation/1\nrun=cap-%s\nkind=client\n' "$1"
    printf 'target=fixture-client\nmeasured_build=%s\nstock_client=true\n' "$6"
    # ⚠ `uname -srm`, which is what the real attestation carries, so the
    # platform and the architecture in the record are DERIVED here exactly as
    # they are there - and `x86_64` is spelled with the underscore the kernel
    # uses rather than the hyphen this project's vocabulary uses.
    printf 'platform=Linux 0.0.0-fixture x86_64\n'
    printf 'started_at=2026-09-09T15:08:24Z\nfinished_at=2026-09-09T15:09:14Z\n'
    printf 'fixture=sha256:%s\n' "$(printf 'fixture-%s' "$1" | sha256sum | cut -d' ' -f1)"
    printf 'egress=closed\n'
    # ⛔ `E-CAP-01` NEEDS TWO CONNECTORS AT THE VALIDITY GATE, so a lane with no
    # eighth argument is run 14's shape and cannot become a record at all.
    [ -z "${8:-}" ] || printf 'connectors=fixture-oracle\n'
    # ⚠ Each connector's version, as the real attestation carries it since
    # 2026-10-14. `ASSEMBLE_DROP` names a key a case leaves out.
    [ "${ASSEMBLE_DROP:-}" = observer_version ] || printf 'observer_version=0.0.0\n'
    [ -z "${8:-}" ] || [ "${ASSEMBLE_DROP:-}" = connector_version ] ||
      printf 'connector_version=9.8.7\n'
    [ "${ASSEMBLE_DROP:-}" = uptime_at_finish ] || printf 'uptime_at_finish=164.50\n'
  } >"$_cap/capture/attestation.txt"

  # ⭐ WHAT A RUN MANIFEST IS READ FROM, `CI-09`: the host claim the capture keeps,
  # and the observer's own log for its serving window. ⚠ The times put every
  # install before the capture, which is what the barrier makes true on a runner.
  if [ "${ASSEMBLE_DROP:-}" != host-claim ]; then
    {
      printf 'bit-ids/host-claim/1\nrun=cap-%s\nclaimed_at=2026-09-09T15:08:10Z\n' "$1"
      printf 'fingerprint=%s\n' "$(printf 'host-%s' "$1" | sha256sum | cut -d' ' -f1)"
      printf 'image=fixture-image/1\nos=Fixture OS 1\n'
      printf 'runner_environment=%s\n' "${ASSEMBLE_RUNNER:-github-hosted}"
      printf 'uptime_at_claim=100.25\n'
    } >"$_cap/capture/host-claim.txt"
  fi
  printf 'serving for 30s\n' >"$_cap/capture/observer.log"

  # The second connector's own report, which is the contract `OBS-07` fills.
  # ⛔ EVERY SAMPLE, IN THE TRANSCRIPT'S ORDER, which is what the contract asks
  # since 2026-10-14: the eighth argument is what the connector read for the
  # FIRST segment of each surface - which a case moves to plant a conflict - and
  # a ninth adds the second segment's value as the transcript carries it.
  if [ -n "${8:-}" ]; then
    mkdir -p "$_cap/capture/bundle/connector" || return 1
    _agent=$(hexof 'fixture-client/1.2.3')
    _ids=$(hexof "$8")
    _agents=$_agent
    _reserved=$RESERVED_HEX
    if [ -n "${9:-}" ]; then
      _ids="$_ids,$(hexof "$9")"
      _agents="$_agents,$_agent"
      _reserved="$_reserved,$RESERVED_HEX"
    fi
    {
      printf 'tracker_http/peer_id=%s\n' "$_ids"
      printf 'tracker_http/user_agent=%s\n' "$_agents"
      printf 'peer_wire/peer_id=%s\n' "$_ids"
      printf 'peer_wire/reserved=%s\n' "$_reserved"
    } >"$_cap/capture/bundle/connector/fixture-oracle.txt"
  fi

  {
    printf 'bit-ids/install-record/1\ntarget=fixture-client\nroute=%s\n' "$2"
    printf 'reported_version=%s\npreexisting_version=\n' "$6"
    printf 'preexisting_binary=\npreexisting_binary_sha256=\n'
    printf 'installed_binary=/usr/local/bin/fixture-client\n'
    printf 'installed_binary_sha256=%s\n' "$5"
    printf 'acquired=%s\nsource_commit=%s\npackage=elf-binary\n' "${ACQUIRED:-yes}" "$4"
    printf 'adapter=scripts/capture/adapters/fixture.sh\n'
    printf 'started_at=2026-09-09T15:08:17Z\nfinished_at=%s\n' "${ASSEMBLE_INSTALLED:-2026-09-09T15:08:18Z}"
    [ "${ASSEMBLE_DROP:-}" = installed_binary_bytes ] || printf 'installed_binary_bytes=4096\n'
    printf 'retrieved_at=2026-09-09T15:08:18Z\nsignature=not_checked\n'
  } >"$_cap/install-$2.txt"

  # ⛔ A SOURCE LANE'S RESOLUTION IS A DIFFERENT DOCUMENT IN A DIFFERENT PLACE,
  # and this harness wrote a RELEASE one for every lane until 2026-09-15 - which
  # is why twenty-four cases passed over a shape no source lane has ever had.
  # `resolve-source.sh` writes `source/resolution.txt` with the banner
  # `bit-ids/source-resolution/1`, carrying the clone URL in `source_url`,
  # `refs_sha256` instead of a listing digest, and NO `asset_url` at all, because
  # a source route selects no asset.
  #
  # ⚠ THE FIXTURE WAS WRONG IN THE DIRECTION THAT HIDES A DEFECT. Giving every
  # lane an `asset_url` meant the assembler's demand for one was satisfiable by
  # every lane here, so the branch that refuses a source route for a field its
  # document is not supposed to carry was unreachable. Run 16 is what found it.
  if [ "$2" = source ]; then
    mkdir -p "$_ins/source" || return 1
    {
      printf 'bit-ids/source-resolution/1\ntarget=fixture-client\n'
      printf 'repository=fixture-owner/fixture-client\n'
      printf 'source_url=%s\n' "$3"
      printf 'refs_sha256=%s\n' "$(printf 'refs-%s' "$6" | sha256sum | cut -d' ' -f1)"
      printf 'selected_version=%s\nselected_tag=v%s\n' "$6" "$6"
      printf 'started_at=2026-09-09T15:08:12Z\nfinished_at=2026-09-09T15:08:13Z\n'
    } >"$_ins/source/resolution.txt"
  else
    {
      printf 'bit-ids/release-resolution/1\ntarget=fixture-client\n'
      printf 'repository=fixture-owner/fixture-client\n'
      printf 'source_url=%s\n' "$3"
      printf 'selected_version=%s\nselected_tag=v%s\n' "$6" "$6"
      printf 'asset=fixture-client-%s-linux-x86-64\n' "$6"
      printf 'asset_url=https://example.invalid/fixture-client-%s\n' "$6"
      printf 'started_at=2026-09-09T15:08:12Z\nfinished_at=2026-09-09T15:08:13Z\n'
    } >"$_ins/release/resolution.txt"
  fi

  # ⛔ `E-ACQ-10` NEEDS WHAT THE BUILD PRINTED, AND IT IS IN THE INSTALL
  # ARTIFACT rather than in the capture bundle. That split is the first thing
  # assembling run 14 measured, and the layout here is run 14's.
  printf 'fixture-client %s\ncompiled for the %s route\n' "$6" "$2" \
    >"$_ins/install-$2/version.err"

  printf 'd4:infod4:name7:fixtureee' >"$_cap/capture/bundle/fixture/generated.torrent"
  # ⭐ A ninth argument is a SECOND connection on each surface, carrying its own
  # peer ID - which is what a build with a per-connection tail really puts in a
  # transcript, and what turns `constant` with one sample into a pattern.
  if [ -n "${9:-}" ]; then
    # ⭐ A TENTH ARGUMENT MAKES THE SECOND ANNOUNCE A SECOND SESSION, by giving
    # it `event=started`. ⛔ The peer surface is left alone on purpose: nothing
    # in a handshake delimits a restart, and `client-capture` opens every peer
    # connection back to back once the first announce arrives, so they belong to
    # whichever session was live then.
    transcript "$_cap/capture/bundle/tracker-http.transcript.json" \
      tracker-http "$(announce_hex "$7")" "$(announce_hex "$9" "${10:-}")"
    transcript "$_cap/capture/bundle/peer-wire-dialled.transcript.json" \
      peer-wire-dialled "$(handshake_hex "$7")" "$(handshake_hex "$9")"
  else
    transcript "$_cap/capture/bundle/tracker-http.transcript.json" \
      tracker-http "$(announce_hex "$7")"
    transcript "$_cap/capture/bundle/peer-wire-dialled.transcript.json" \
      peer-wire-dialled "$(handshake_hex "$7")"
  fi
}

# Runs the assembler over two lanes into a fresh store, keeping the streams
# apart. ⛔ The exit code is read from the process that produced it.
assemble() { # tag lane...
  _tag=$1
  shift
  rm -rf "$WORK/store-$_tag"
  mkdir -p "$WORK/store-$_tag" || return 2
  _args=""
  for _lane in "$@"; do
    _args="$_args --lane $WORK/$_lane-capture,$WORK/$_lane-install"
  done
  # shellcheck disable=SC2086
  "$ASSEMBLE" $_args "$WORK/store-$_tag" >"$WORK/$_tag.out" 2>"$WORK/$_tag.err"
  return $?
}

said() { # tag literal
  grep -q -F -e "$2" "$WORK/$1.out" "$WORK/$1.err" 2>/dev/null
}

# A case: run the assembler over some lanes and compare the exit code and one
# literal. ⛔ Both, because several refusals here share an exit code and a case
# asserting the code alone passes when a different refusal fired - which is the
# defect `CI-03`'s Windows harness found in itself.
case_is() { # expected-rc literal name tag lane...
  _want=$1
  _needle=$2
  _name=$3
  shift 3
  assemble "$@"
  _rc=$?
  if [ "$_rc" != "$_want" ]; then
    fail "$_name (exit $_rc, expected $_want)"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/$1.err" | head -4
    return
  fi
  if [ -n "$_needle" ] && ! said "$1" "$_needle"; then
    fail "$_name (exit $_want, and it did not say '$_needle')"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/$1.err" | head -4
    return
  fi
  pass "$_name"
}

GH=https://api.example.invalid/repos/fixture-owner/fixture-client/releases
REFS=https://git.example.invalid/fixture-owner/fixture-client/refs
COMMIT=$(printf 'fixture-commit' | sha256sum | cut -c1-40)
DIGEST_A=$(printf 'build-a' | sha256sum | cut -d' ' -f1)
DIGEST_B=$(printf 'build-b' | sha256sum | cut -d' ' -f1)
PEER_A='-XX0000-aaaaaaaaaaaa'
PEER_B='-XX0000-bbbbbbbbbbbb'
# ⚠ A second connection's tail per lane. Same eight-byte prefix, different
# twelve bytes, which is what a per-connection tail looks like on the wire.
PEER_C='-XX0000-cccccccccccc'
PEER_D='-XX0000-dddddddddddd'

# -- ⭐ THE CONTROL, AND IT IS THE HALF THAT MATTERS MOST ---------------------
#
# Every refusal below passes equally over an assembler that refuses everything.
# This is the pair that has to be accepted: two routes, two resolvers, two
# connectors, one version, one identity on the wire.
write_lane good-release release "$GH" "" "$DIGEST_A" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
write_lane good-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
case_is 0 "" "two routes with independent resolvers assemble into records" \
  good good-release good-source
if said good build_equivalent; then
  pass "a pair whose observations agree reaches build_equivalent"
else
  fail "a pair whose observations agree reaches build_equivalent"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/good.out" | head -6
fi

# -- ⛔ A CONNECTOR'S VERSION IS MEASURED, NEVER WRITTEN HERE ------------------
#
# Both were `0.0.0` in every record until 2026-10-14, for the second connector a
# value nobody measured. The attestation's own versions are what a record says.
_rec=$(find "$WORK/store-good/profiles" -name 'cap-release.json' -type f | head -1)
if [ -n "$_rec" ] && python3 -I -c 'import json,sys
d=json.load(open(sys.argv[1]))
v={c["id"]:c["version"] for c in d["capture"]["connectors"]}
sys.exit(0 if v=={"bit-ids-probe":"0.0.0","fixture-oracle":"9.8.7"} else 1)' "$_rec"; then
  pass "versions  each connector's version is the one its attestation carries"
else
  fail "versions  each connector's version is the one its attestation carries"
  [ "$JSON" = "1" ] || printf '          record %s: %s\n' "${_rec:-none}" \
    "$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["capture"]["connectors"])' "$_rec" 2>&1)"
fi
for _drop in observer_version connector_version; do
  ASSEMBLE_DROP=$_drop
  write_lane drop-release release "$GH" "" "$DIGEST_A" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
  ASSEMBLE_DROP=""
  write_lane drop-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
  case_is 1 "$_drop" "a lane whose attestation records no $_drop is refused, naming it" \
    "drop-$_drop" drop-release drop-source
done

# -- ⭐ AND WHAT IT WROTE IS A CORPUS, WITH A RUN BESIDE EVERY RECORD ----------
#
# ⛔ `E-CRP-01` refused every store this assembler wrote until 2026-10-14,
# because no step of the capture path wrote a run manifest. `CI-09` carries the
# producers that now record what one is read from, and the barrier that makes
# its phases true.
if said good "corpus: valid, 2 record(s)"; then
  pass "corpus    a two-lane assembly is a valid corpus, with a run beside each record"
else
  fail "corpus    a two-lane assembly is a valid corpus, with a run beside each record"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/good.out" | tail -8
fi

# ⭐ THE CONTROL: AN INDEPENDENT READER. `validate-run` reads each manifest and
# its record off the disk and binds them, so a manifest the assembler judged by
# its own lights is judged again by the one tool `SCHEMA-02` names for it.
_pairs=0
_bound=0
find "$WORK/store-good" -name 'manifest.json' -type f >"$WORK/good.manifests"
# ⚠ Redirected rather than piped, so the counters survive the loop.
while IFS= read -r _manifest; do
  _pairs=$((_pairs + 1))
  _cap=$(sed -n 's/.*"capture": *"\([^"]*\)".*/\1/p' "$_manifest" | head -1)
  _profile=$(find "$WORK/store-good/profiles" -name "$_cap.json" -type f | head -1)
  [ -n "$_profile" ] && "$VALIDATE_RUN" "$_manifest" "$_profile" >/dev/null 2>&1 &&
    _bound=$((_bound + 1))
done <"$WORK/good.manifests"
if [ "$_pairs" = 2 ] && [ "$_bound" = 2 ]; then
  pass "corpus    validate-run binds each manifest to its record, read off the disk ($_bound)"
else
  fail "corpus    validate-run binds each manifest to its record, read off the disk ($_bound of $_pairs)"
fi

# ⛔ AND A MANIFEST NO ARTIFACT SUPPORTS IS NOT WRITTEN. One case per producer
# the manifest reads, each naming what was missing; the record is still written,
# and the run is refused.
for _drop in host-claim uptime_at_finish installed_binary_bytes; do
  ASSEMBLE_DROP=$_drop
  write_lane gap-release release "$GH" "" "$DIGEST_A" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
  ASSEMBLE_DROP=""
  write_lane gap-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
  case_is 1 "$_drop" "a lane with no $_drop gets no run manifest, and the run says why" \
    "gap-$_drop" gap-release gap-source
done
ASSEMBLE_RUNNER=self-hosted
write_lane gap-release release "$GH" "" "$DIGEST_A" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
ASSEMBLE_RUNNER=""
write_lane gap-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
case_is 1 "runner_environment=self-hosted" \
  "a host nothing says is destroyed after the job gets no run manifest" \
  gap-runner gap-release gap-source

# ⛔ RUN 37'S ORDER, AS DATA: the source route finished installing after the
# release lane's capture began. The barrier stops that on a runner; here the
# manifest is what must refuse it, because its phases would overlap.
write_lane late-release release "$GH" "" "$DIGEST_A" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
ASSEMBLE_INSTALLED=2026-09-09T15:08:40Z
write_lane late-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
ASSEMBLE_INSTALLED=""
case_is 1 "E-MAN-06" "a capture that began before every route had installed gets no run manifest" \
  late late-release late-source

# ⛔ AND THE PAIR PUBLISHES, WHICH IT DID NOT UNTIL 2026-09-15. These two lanes
# install DIFFERENT BYTES - `DIGEST_A` and `DIGEST_B` - which is every
# release-against-source pair there is, and a capture puts one of them on the
# wire. So `classify` answers `unresolved` and `E-PUB-04` refused every record of
# this shape however many captures existed. The other route's capture is what
# settles it and it is a DIFFERENT RECORD, so the gate is asked with the store.
# ⚠ Counted on a whole line rather than with a substring, because
# `provisional, not publishable` contains the word.
if [ "$(grep -c '^[[:space:]]*publishable$' "$WORK/good.out")" = "2" ]; then
  pass "a byte-different pair settled by the other route's capture publishes"
else
  fail "a byte-different pair settled by the other route's capture publishes"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/good.out" | head -12
fi
# ⛔ TWO RECORDS, NOT ONE. `E-ACQ-01` needs two routes in each and
# `observed_route` says which install each watched, so a lane per route is two
# documents over one route list. A store with one profile is an assembler that
# wrote the capture it was handed last.
# ⚠ UNDER `profiles/`, not every `.json` in the store. The transcripts this
# copies in are `.json` too, so a count over the whole tree answered six and
# read as a defect in the assembler. Found by running it.
if [ "$(find "$WORK/store-good/profiles" -name '*.json' -type f | grep -c .)" = "2" ]; then
  pass "one record per lane is written, over the same two routes"
else
  fail "one record per lane is written, over the same two routes"
fi
if [ "$(find "$WORK/store-good" -name 'version.txt' -type f | grep -c .)" -ge 1 ]; then
  pass "the process output the installed version cites is copied into the store"
else
  fail "the process output the installed version cites is copied into the store"
fi

# -- ⛔ AND EVERY ARTIFACT A RECORD CITES IS CARRIED, WHICH IT WAS NOT ---------
#
# ⛔ **A record's evidence list is built over ALL the lanes** - a field citing
# the other route's install record is what makes the pair comparable - and the
# assembler copied only ITS OWN lane's four files under its evidence root. So
# each record cited four artifacts at paths in its own store directory that
# nothing ever wrote to.
#
# ⚠ `E-CRP-03` is exactly that refusal and it CANNOT FIRE HERE: it is checked
# per run manifest and this path writes none. The case above accepted it too -
# `-ge 1` is satisfied by half the files. It was found on 2026-09-17 by handing
# `validate_corpus` a tree read back off the disk instead of an empty one, and
# the assembler now names any citation it did not carry.
if said good "cited and not carried"; then
  fail "evidence  a record cites an artifact the store does not carry"
  [ "$JSON" = "1" ] || grep -F 'cited and not carried' "$WORK/good.out" | sed 's/^/          /'
else
  pass "evidence  every artifact a record cites is carried in the store"
fi

# ⭐ THE CONTROL, AND IT IS STRUCTURAL RATHER THAN A COUNT. A record that got
# only its own lane's artifacts has ONE route directory under its evidence root;
# a self-contained one has every route the record's list names. ⚠ Derived by
# listing the store rather than from a number typed here, so it follows a lane
# being added or removed.
_captures=0
_whole=0
for _root in "$WORK"/store-good/raw/v1/*/*/*/*/*/*; do
  [ -d "$_root" ] || continue
  _captures=$((_captures + 1))
  if [ "$(find "$_root" -mindepth 1 -maxdepth 1 -type d | grep -c .)" -ge 2 ]; then
    _whole=$((_whole + 1))
  fi
done
if [ "$_captures" -ge 2 ] && [ "$_whole" = "$_captures" ]; then
  pass "evidence  each record carries both routes' artifacts, not just its own"
else
  fail "evidence  $_whole of $_captures record(s) carry every route's artifacts"
fi

# -- ⛔ AND THE ONE THAT REFUTES THE WORK ORDER -------------------------------
#
# ⛔ A PEER ID CARRIES A PER-CONNECTION RANDOM TAIL, SO TWO CAPTURES OF ONE
# BUILD NECESSARILY DISAGREE ON IT, and `classify_across` reads a disagreement
# on any field both records measured as `divergent`. `constant` with one sample
# is the only state a single capture can produce for such a field, so
# `build_equivalent` is unreachable for any real client through this path -
# which is what the work order said run 14 had made reachable.
# ⭐ `SCHEMA-04`'s sampling model is where several captures become a `patterned`
# field, and `sampling::field_state` is the join that reaches a record since
# 2026-09-15. ⚠ What is still missing is the SAMPLES: this path captures once per
# lane, so `constant` with one sample remains the only state it can produce.
write_lane vary-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_B" "$PEER_B" || exit 2
case_is 0 "divergent" \
  "two captures whose peer IDs differ are divergent rather than build_equivalent" \
  vary good-release vary-source
# ⛔ AND THE DIVERGENCE MUST REFUSE PUBLICATION, or the rule above would be a
# search for any sibling that agrees. ⚠ `E-PUB-03` rather than `E-PUB-04`: the
# store held a comparable capture and it CONFLICTED, which is a different
# finding from no capture at all, and folding them would lose which.
if said vary "provisional, not publishable" && said vary "E-PUB-03" &&
  ! said vary "E-PUB-04"; then
  pass "a pair the store shows diverging is refused as a conflict, not an absence"
else
  fail "a pair the store shows diverging is refused as a conflict, not an absence"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/vary.out" | head -12
fi

# -- ⭐ AND THE SAMPLES THAT TURN THAT DIVERGENCE INTO AN AGREEMENT -----------
#
# ⛔ THE PAIR ABOVE DIVERGES BECAUSE EACH FIELD RESTS ON ONE SAMPLE. A peer ID's
# tail is regenerated per connection, so one observation of it is a value the
# next connection will not repeat, and two lanes then state two different
# constants. ⭐ A build that connects TWICE puts two of them in the transcript,
# and `sampling::field_state` reads the eight-byte prefix that held and the
# twelve bytes that moved: both lanes describe ONE pattern, and a pattern is
# what they can agree on.
# ⚠ All four tails here are different. That is the point - what agrees is the
# shape, not the bytes.
write_lane samp-release release "$GH" "" "$DIGEST_A" 1.2.3 \
  "$PEER_A" "$PEER_A" "$PEER_C" || exit 2
write_lane samp-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 \
  "$PEER_B" "$PEER_B" "$PEER_D" || exit 2
case_is 0 "build_equivalent" \
  "two connections per lane make a per-connection tail a pattern both lanes share" \
  samp samp-release samp-source
if [ "$(grep -c '^[[:space:]]*publishable$' "$WORK/samp.out")" = "2" ]; then
  pass "and the pair that was divergent on one sample each publishes"
else
  fail "and the pair that was divergent on one sample each publishes"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/samp.out" | head -14
fi
# ⛔ AND THE STATE IS `patterned`, NOT `constant`. Without this the case above
# would pass equally over an assembler that read the first segment and got lucky
# on lanes whose first tails happened to match.
if grep -q '"kind": "patterned"' \
  "$(find "$WORK/store-samp/profiles" -name '*.json' -type f | head -1)"; then
  pass "the record states the pattern rather than one connection's bytes"
else
  fail "the record states the pattern rather than one connection's bytes"
fi
# ⚠ AND ONE CONNECTION IS STILL `constant` WITH ONE SAMPLE, which is what says
# this changed the sampling and not the default: every lane above has one
# segment per surface and every record they produce is unchanged.
if grep -q '"kind": "constant"' \
  "$(find "$WORK/store-good/profiles" -name '*.json' -type f | head -1)"; then
  pass "a lane that connected once still states a constant"
else
  fail "a lane that connected once still states a constant"
fi

# -- ⛔ AND WHICH LIFETIME THE TAIL HAS, WHICH ONE SESSION CANNOT ANSWER -------
#
# ⛔ `capture-client` run 20 announced twice in ONE session and carried the same
# peer ID both times, so the tail is per session on that surface - and a record
# calling it `per_connection` would be wrong about the build.
# `classify_offset` can only answer `per_session` when the samples are attributed
# to more than one, and the assembler filed every one of them under session 0
# until 2026-09-15.
#
# ⭐ THE SESSION IS READ OUT OF THE ANNOUNCE rather than out of the attestation,
# so this fixture carries no new field: the second announce says `event=started`,
# which BEP 3 makes the first announce of a run.
write_lane sess-release release "$GH" "" "$DIGEST_A" 1.2.3 \
  "$PEER_A" "$PEER_A" "$PEER_C" started || exit 2
write_lane sess-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 \
  "$PEER_B" "$PEER_B" "$PEER_D" started || exit 2
case_is 0 "build_equivalent" \
  "two sessions per lane make a per-session tail a pattern both lanes share" \
  sess sess-release sess-source
_sess_record=$(find "$WORK/store-sess/profiles" -name '*.json' -type f | head -1)
if grep -q '"kind": "patterned"' "$_sess_record"; then
  pass "a tail that moves across sessions is a pattern, as one that moves across connections is"
else
  fail "a tail that moves across sessions is a pattern, as one that moves across connections is"
fi

# ⛔ AND THE RECORD CANNOT SAY WHICH OF THE TWO IT IS. `sampling::classify`
# computes a `Lifetime` per span - `per_connection`, `per_session`, `persistent`
# - and `field_state` DISCARDS it: the state is rebuilt from `differs_at` alone,
# so `PatternedValue` carries a tiling and no lifetime, and no record in this
# repository has ever carried one. ⚠ **This case exists to keep that gap
# measured rather than remembered.** It passes while the schema has no such
# field and goes red the day one appears with nothing filling it, which is the
# event that should make somebody read the residual in `TODO/observer.md`.
if grep -q '"lifetime"' "$_sess_record"; then
  fail "the record carries no lifetime, and this says so rather than assuming it"
else
  pass "the record carries no lifetime: classify computes one and field_state drops it"
fi

# ⛔ AND A REQUEST THAT IS NOT AN ANNOUNCE IS NOT A SAMPLE OF ONE. A scrape
# carries no `peer_id`, so it says nothing about the fields an announce does;
# reading it as one would refuse a capture for carrying a request the record
# does not describe. ⚠ This case exists because the rule SURVIVED a mutation
# pass - no fixture here had such a segment, so every reading agreed.
write_lane scrape-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 \
  "$PEER_B" "$PEER_B" "$PEER_D" || exit 2
transcript "$WORK/scrape-source-capture/capture/bundle/tracker-http.transcript.json" \
  tracker-http "$(announce_hex "$PEER_B")" "$(scrape_hex)" "$(announce_hex "$PEER_D")"
case_is 0 "build_equivalent" \
  "a scrape between two announces is passed over rather than refused" \
  scrape samp-release scrape-source
# ⛔ AND IT IS PASSED OVER RATHER THAN COUNTED. Three segments, two announces:
# a field resting on three samples would mean the scrape became one, which is a
# different defect from a refusal and looks identical in the line above.
if grep -q '"samples": 2' \
  "$(find "$WORK/store-scrape/profiles" -name 'cap-source.json' -type f | head -1)"; then
  pass "the scrape is not counted as a sample of an announce field"
else
  fail "the scrape is not counted as a sample of an announce field"
  [ "$JSON" = "1" ] || find "$WORK/store-scrape/profiles" -name 'cap-source.json' \
    -exec grep -o '"samples": [0-9]*' {} + | sed 's/^/          /' | head -6
fi

# -- The refusals ------------------------------------------------------------

# ⛔ ONE CONNECTOR IS REFUSED AT THE VALIDITY GATE, NOT AT PUBLICATION, and this
# is the case that refutes the standing handoff. `CI-09` recorded that a
# single-connector capture "validates, then `E-PUB-02` refuses to publish it";
# measured on 2026-09-09 by stripping the golden fixture two ways, that is true
# of a record declaring TWO connectors where only one observed each field, and
# false of one declaring one. Every capture this project has run declares one.
write_lane lonely-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" || exit 2
write_lane lonely-release release "$GH" "" "$DIGEST_A" 1.2.3 "$PEER_A" || exit 2
case_is 1 "E-CAP-01" "a capture declaring one connector cannot become a record" \
  lonely lonely-release lonely-source

# ⛔ AND A DECLARED CONNECTOR THAT REPORTS NOTHING ABOUT A FIELD IS REFUSED
# rather than recorded as silent. `E-COR-07` already says silence is not the
# same as not_corroborated, and an assembler that filled the gap in would be
# writing corroboration the run never produced.
write_lane partial-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
grep -v '^peer_wire/reserved=' \
  "$WORK/partial-source-capture/capture/bundle/connector/fixture-oracle.txt" \
  >"$WORK/partial.txt"
cp "$WORK/partial.txt" \
  "$WORK/partial-source-capture/capture/bundle/connector/fixture-oracle.txt"
case_is 1 "reports nothing about peer_wire/reserved" \
  "a declared connector silent on a field is refused" \
  partial good-release partial-source

# -- ⭐ AND THE CASE `OBS-07`'s PROVE ASKS FOR --------------------------------
#
# ⛔ A DELIBERATELY ALTERED FIELD MUST PRODUCE A CONNECTOR CONFLICT, and until
# there was a second connector nothing could produce one. This lane's connector
# reports a peer ID the transcript does not carry, on both surfaces, and agrees
# with the observer on the other two fields - so the record has to keep a
# conflict on exactly the altered fields rather than drop an observation or
# average the two.
# ⚠ IT IS A VALID RECORD. Refusing it would lose the evidence of the
# disagreement, which is the split this project holds everywhere: `publishable`
# is what refuses it, and the assembler says so on its own line.
write_lane conflict-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_B" || exit 2
case_is 0 "E-PUB-01" \
  "a connector that read different bytes is recorded as a conflict" \
  conflict good-release conflict-source
if said conflict "provisional, not publishable"; then
  pass "a record carrying a connector conflict is not publishable"
else
  fail "a record carrying a connector conflict is not publishable"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/conflict.out" | head -8
fi

# ⛔ AND IT FIRES ON THE ALTERED FIELDS AND ONLY ON THEM. `E-PUB-01` on every
# field would pass the case above while saying the connector agreed about
# nothing, which is a different defect wearing the same row. The lane's report
# alters both peer IDs and leaves the User-Agent and the reserved bytes alone.
CONFLICTED=$(grep -c -F -e 'E-PUB-01' "$WORK/conflict.out" 2>/dev/null || true)
if [ "$CONFLICTED" = "2" ]; then
  pass "the conflict is on the two altered fields and not on the two agreed ones"
else
  fail "the conflict is on the two altered fields and not on the two agreed ones ($CONFLICTED rows)"
  [ "$JSON" = "1" ] || grep -F -e 'E-PUB-01' "$WORK/conflict.out" | sed 's/^/          /'
fi

# -- ⛔ AND EVERY SAMPLE IS CORROBORATED, NOT THE FIRST ------------------------
#
# ⛔ THE ASSEMBLER COMPARED ONE VALUE PER FIELD UNTIL 2026-10-14, so a field
# resting on two announces was corroborated on the first and the record's
# `exact` vouched for a second reading nobody had compared. This lane's
# connector agrees on every first sample and reads a different peer ID out of
# the SECOND announce only: an assembler comparing first samples calls it exact
# and publishes it.
write_lane late-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 \
  "$PEER_B" "$PEER_B" "$PEER_D" || exit 2
_late_report="$WORK/late-source-capture/capture/bundle/connector/fixture-oracle.txt"
_late_from="tracker_http/peer_id=$(hexof "$PEER_B"),$(hexof "$PEER_D")"
_late_to="tracker_http/peer_id=$(hexof "$PEER_B"),$(hexof "$PEER_C")"
sed "s|^$_late_from\$|$_late_to|" "$_late_report" >"$WORK/late.txt"
if cmp -s "$WORK/late.txt" "$_late_report"; then
  fail "the second-sample plant applied"
else
  pass "the second-sample plant applied"
  cp "$WORK/late.txt" "$_late_report"
  case_is 0 "E-PUB-01" \
    "a connector that parts from the observer on the second sample is a conflict" \
    late samp-release late-source
  if [ "$(grep -c -F -e 'E-PUB-01' "$WORK/late.out")" = "1" ]; then
    pass "the late conflict is on the one field it was planted in"
  else
    fail "the late conflict is on the one field it was planted in"
    [ "$JSON" = "1" ] || grep -F -e 'E-PUB-01' "$WORK/late.out" | sed 's/^/          /'
  fi
  # ⭐ AND THE RECORD CARRIES THE SAMPLE THAT PARTED. A corroboration entry holds
  # one value per connector, and `E-COR-14` refuses a `disagrees` whose values
  # match - so the pair recorded is the second, and both of its values are the
  # transcript's and the planted report's rather than the first sample twice.
  _late_record=$(find "$WORK/store-late/profiles" -name 'cap-source.json' -type f | head -1)
  if grep -q -F -e 'at sample 2 of 2' "$_late_record" &&
    grep -q -F -e "\"$(hexof "$PEER_C")\"" "$_late_record" &&
    grep -q -F -e "\"$(hexof "$PEER_D")\"" "$_late_record"; then
    pass "the record names sample 2 and carries what each reader saw there"
  else
    fail "the record names sample 2 and carries what each reader saw there"
  fi
fi

# ⛔ AND A REPORT THAT COUNTED FEWER SAMPLES DISAGREES ABOUT WHAT WAS SENT. One
# value over a field the observer saw twice is the contract this replaced, and
# reading it as a corroboration of the first sample is what the old assembler
# did. The reader with no second sample is recorded as having seen nothing
# there, which is what it looked for and did not find.
write_lane short-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 \
  "$PEER_B" "$PEER_B" "$PEER_D" || exit 2
_short_report="$WORK/short-source-capture/capture/bundle/connector/fixture-oracle.txt"
sed "s|^tracker_http/user_agent=\\([0-9a-f]*\\),.*\$|tracker_http/user_agent=\\1|" \
  "$_short_report" >"$WORK/short.txt"
if cmp -s "$WORK/short.txt" "$_short_report"; then
  fail "the one-sample plant applied"
else
  pass "the one-sample plant applied"
  cp "$WORK/short.txt" "$_short_report"
  case_is 0 "E-PUB-01" \
    "a connector that read fewer samples than the observer is a conflict" \
    shortrep samp-release short-source
  if grep -q -F -e 'read 2 and 1 samples, and part at sample 2' \
    "$(find "$WORK/store-shortrep/profiles" -name 'cap-source.json' -type f | head -1)"; then
    pass "the record says how many samples each reader found"
  else
    fail "the record says how many samples each reader found"
  fi
fi

# ⛔ AND A DIFFERENCE THE RECORD CANNOT SHOW IS REFUSED BY NAME. One announce,
# and a connector that reports a second, absent, `User-Agent` after it: the two
# readers differ only in how many samples each found, the one with no second
# sample is recorded as seeing nothing there, and so is the one that reported an
# absence. `disagrees` over two matching values is what `E-COR-14` refuses and
# `exact` would be false, so the assembler says which it is rather than leaving
# the validator to refuse the record for a reason that names neither reader.
write_lane extra-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
_extra_report="$WORK/extra-source-capture/capture/bundle/connector/fixture-oracle.txt"
sed 's|^\(tracker_http/user_agent=[0-9a-f]*\)$|\1,absent|' "$_extra_report" >"$WORK/extra.txt"
if cmp -s "$WORK/extra.txt" "$_extra_report"; then
  fail "the extra-absent-sample plant applied"
else
  pass "the extra-absent-sample plant applied"
  cp "$WORK/extra.txt" "$_extra_report"
  case_is 1 "only in how many samples each found" \
    "a count-only difference no record can show is refused by name" \
    extra good-release extra-source
fi

# ⛔ AND A LIST WITH A HOLE IN IT IS REFUSED, NOT CLOSED UP. `a,,b` lost a sample
# somewhere, and a reader that skipped the gap would line every later sample up
# against the wrong segment.
write_lane hole-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 \
  "$PEER_B" "$PEER_B" "$PEER_D" || exit 2
_hole_report="$WORK/hole-source-capture/capture/bundle/connector/fixture-oracle.txt"
sed 's|^peer_wire/reserved=\([0-9a-f]*\),|peer_wire/reserved=\1,,|' \
  "$_hole_report" >"$WORK/hole.txt"
if cmp -s "$WORK/hole.txt" "$_hole_report"; then
  fail "the empty-sample plant applied"
else
  pass "the empty-sample plant applied"
  cp "$WORK/hole.txt" "$_hole_report"
  case_is 1 "sample 2 is empty" "a connector report with an empty sample is refused" \
    hole samp-release hole-source
fi

# -- ⛔ A ROUTE THAT ACQUIRED NOTHING IS NOT A SECOND ROUTE --------------------
#
# ⛔ `ACQ-03`'s SHARPEST RESIDUAL, AND IT IS MEASURED RATHER THAN IMAGINED.
# `aria2` ships on `ubuntu-24.04`, so client capture runs 3 and 4 recorded
# `route=package` over an `apt-get install` that installed nothing. A route like
# that exits 0, declares its own resolver, satisfies `E-ACQ-07` and `E-ACQ-08`,
# agrees on the version BECAUSE IT IS ONE BINARY, and reaches `classify` as
# `byte_identical` - the strongest verdict there is, reached by acquiring
# nothing.
# ⚠ THE THIRD VALUE IS ITS OWN CASE. `acquired` is written by shell, so a record
# carrying anything but `yes` or `no` is a field nothing derived; reading an
# unknown value as `no` would refuse it for a reason that is not true.
ACQUIRED=no
write_lane idle-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
ACQUIRED=yes
case_is 1 "acquired=no" \
  "a route that ran and installed nothing is refused as a second route" \
  idle good-release idle-source

ACQUIRED=maybe
write_lane odd-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
ACQUIRED=yes
case_is 1 "neither yes nor no" \
  "an install record whose acquired field is not yes or no is refused" \
  odd good-release odd-source

# ⛔ ONE LISTING, TWO LANES. This is run 14's shape: `capture-client.yml` resolves
# once and hands the release lane an asset URL and the source lane a tag, and
# argues for it in a comment. `E-ACQ-07` calls that one route.
write_lane same-source source "$GH" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
case_is 1 "E-ACQ-07" "two lanes that read one listing are refused as one route" \
  same good-release same-source

write_lane no-commit source "$REFS" "" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
case_is 1 "E-ACQ-06" "a source route with no commit is refused" \
  nocommit good-release no-commit

write_lane short-commit source "$REFS" abbrev1 "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
case_is 1 "" "a source route with an abbreviated commit is refused" \
  short good-release short-commit

# ⛔ TWO VERSIONS ARE TWO BUILDS. Absolute 4 wants two routes resolving ONE
# stable version, and a record over a pair that reported two would be a claim no
# capture made.
write_lane other-version source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.4 "$PEER_A" "$PEER_A" || exit 2
case_is 1 "must resolve one stable version" \
  "two lanes reporting two versions are refused" \
  versions good-release other-version

# ⛔ THE PROCESS OUTPUT LIVES IN THE INSTALL ARTIFACT, and a lane handed only its
# capture bundle cannot produce a record at all. That is could-not-run rather
# than a refusal: nothing was judged.
rm -f "$WORK/good-source-install/install-source/version.err"
case_is 2 "E-ACQ-10" "a lane with no process output for its version cannot run" \
  novers good-release good-source
write_lane good-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2

# ⛔ THE THREE FIELDS A STORE PATH IS DERIVED FROM ARE DERIVED, NOT WRITTEN IN.
# `platform`, `arch` and `package` are in `StoreKey`, so an assembler that
# hardcoded them would file a Windows capture of one version at the Linux
# capture's path - the non-injective-path defect `store.rs` refuses at length.
# ⚠ The first draft did hardcode all three, and only a claim audit found it.
grep -v '^package=' "$WORK/good-source-capture/install-source.txt" >"$WORK/nopkg.txt"
cp "$WORK/nopkg.txt" "$WORK/good-source-capture/install-source.txt"
case_is 1 "carries no \`package\`" \
  "an attestation with no package cannot be filed, because the path is derived from it" \
  nopkg good-release good-source
write_lane good-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2

# ⛔ AND A FIXTURE CAPTURE IS NOT A MEASUREMENT. `capture.yml` writes
# `kind=fixture` over a run that installed nothing, and a record built from one
# would read exactly like a record of a build.
sed 's/^kind=client$/kind=fixture/' "$WORK/good-source-capture/capture/attestation.txt" \
  >"$WORK/fixture.txt"
cp "$WORK/fixture.txt" "$WORK/good-source-capture/capture/attestation.txt"
case_is 1 "which is not a target kind" \
  "a fixture capture is refused rather than recorded as a build" \
  fixturekind good-release good-source
write_lane good-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2

# ⛔ E-ACQ-08 HAD NO CASE UNTIL A GUARD-MUTATION PASS ASKED FOR ONE. Two lanes
# that were delivered the same way are one route however their resolvers differ,
# and a guard nobody has seen refuse is a guard nobody knows works.
write_lane twin-release release "$REFS" "" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
case_is 1 "E-ACQ-08" "two lanes delivered the same way are refused as one route" \
  twin good-release twin-release

# ⛔ AND A DOCUMENT WITH NO BANNER IS NOT ONE OF THIS PROJECT'S. Every artifact
# the capture path writes opens with its own schema name, and a reader that took
# any key=value file would accept an artifact somebody else produced.
grep -v '^bit-ids/install-record/1$' "$WORK/good-source-capture/install-source.txt" \
  >"$WORK/nobanner.txt"
cp "$WORK/nobanner.txt" "$WORK/good-source-capture/install-source.txt"
case_is 2 "bit-ids document banner" "a record with no schema banner cannot run" \
  nobanner good-release good-source
write_lane good-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2

# ⛔ ONE LANE IS ONE ROUTE. `E-ACQ-01` refuses a record with one, so the
# assembler refuses before reading anything.
rm -rf "$WORK/store-one"
mkdir -p "$WORK/store-one"
"$ASSEMBLE" --lane "$WORK/good-release-capture,$WORK/good-release-install" \
  "$WORK/store-one" >"$WORK/one.out" 2>"$WORK/one.err"
_rc=$?
if [ "$_rc" = 2 ] && said one "E-ACQ-01"; then
  pass "a single lane is refused before anything is read"
else
  fail "a single lane is refused before anything is read (exit $_rc)"
fi

# ⛔ A TRANSCRIPT THIS PROJECT DID NOT WRITE IS NOT EVIDENCE. The reader is the
# writer's inverse, so a document with uppercase hex - which digests differently
# and would have been written as a different artifact - is refused rather than
# decoded.
sed 's/"bytes": "13/"bytes": "13A/' \
  "$WORK/good-source-capture/capture/bundle/peer-wire-dialled.transcript.json" \
  >"$WORK/bad-transcript.json"
if cmp -s "$WORK/bad-transcript.json" \
  "$WORK/good-source-capture/capture/bundle/peer-wire-dialled.transcript.json"; then
  fail "the transcript plant applied"
else
  pass "the transcript plant applied"
  cp "$WORK/bad-transcript.json" \
    "$WORK/good-source-capture/capture/bundle/peer-wire-dialled.transcript.json"
  case_is 1 "transcript" "a transcript this project's writer would not emit is refused" \
    badtrans good-release good-source
  write_lane good-source source "$REFS" "$COMMIT" "$DIGEST_B" 1.2.3 "$PEER_A" "$PEER_A" || exit 2
fi

# -- ⭐ A DISPATCH'S ARTIFACTS, ALL THE WAY TO A BUNDLE ---------------------------
#
# ⛔ The good pair again, under the names a dispatch uploads them as, with the
# `resolution.json` each resolver writes beside its text record. `CI-09`.
# ⭐ `bit-check assemble-dispatch` since 2026-10-15, `CI-07`. It finds the
# repository from where it is run, hence the `cd` below.
BITCHECK=$(store_bitcheck "$ROOT" "$WORK") || exit 2
artifacts() { # dir run-attempt route...
  _dir=$1
  _ra=$2
  shift 2
  rm -rf "$_dir"
  mkdir -p "$_dir" || return 1
  for _route in "$@"; do
    cp -R "$WORK/good-$_route-capture" "$_dir/capture-client-fixture-client-$_route-$_ra" || return 1
    cp -R "$WORK/good-$_route-install" "$_dir/install-fixture-client-$_route-$_ra" || return 1
    mkdir -p "$_dir/install-fixture-client-$_route-$_ra/$_route" || return 1
    printf '{"schema": "bit-ids/resolution/2", "target": "fixture-client", "scheme": {"tag_prefix": "v", "min_components": 3, "max_components": 3, "floor_major": null}}\n' \
      >"$_dir/install-fixture-client-$_route-$_ra/$_route/resolution.json" || return 1
  done
}
# ⛔ Unpiped: the output goes to a file and $? is read on the next line.
dispatch_is() { # expected-rc literal name dir
  rm -rf "$WORK/dispatch-bundle"
  (cd "$ROOT" && "$BITCHECK" assemble-dispatch "$4" "$WORK/dispatch-bundle") >"$WORK/dispatch.out" 2>&1
  _rc=$?
  if [ "$_rc" != "$1" ]; then
    fail "$3 (exit $_rc, expected $1)"
    [ "$JSON" = "1" ] || tail -4 "$WORK/dispatch.out" | sed 's/^/          /'
  elif ! grep -q -F -e "$2" "$WORK/dispatch.out"; then
    fail "$3 (exit $1, and it did not say '$2')"
    [ "$JSON" = "1" ] || tail -4 "$WORK/dispatch.out" | sed 's/^/          /'
  elif [ "$1" != 0 ] && [ -e "$WORK/dispatch-bundle" ]; then
    fail "$3 (refused, and a bundle was left behind)"
  else
    pass "$3"
  fi
}

artifacts "$WORK/arts" 7-1 release source || exit 2
dispatch_is 0 "2 of 2 record(s) may enter a published view" \
  "dispatch  a two-route dispatch's artifacts become a bundle a publisher can verify" "$WORK/arts"
if [ -f "$WORK/dispatch-bundle/MANIFEST.json" ] &&
  (cd "$WORK/dispatch-bundle" && sha256sum -c --quiet SHA256SUMS) >/dev/null 2>&1 &&
  [ "$(find "$WORK/dispatch-bundle/raw" -name manifest.json -type f | grep -c .)" = "2" ]; then
  pass "dispatch  the bundle carries both records' run manifests and its own checksums"
else
  fail "dispatch  the bundle carries both records' run manifests and its own checksums"
fi

artifacts "$WORK/arts" 7-1 release || exit 2
dispatch_is 1 "a record needs two independent routes" \
  "dispatch  an adapter that captured on one route is refused, not left out" "$WORK/arts"

artifacts "$WORK/arts" 7-1 release source || exit 2
rm -rf "$WORK/arts/install-fixture-client-source-7-1"
dispatch_is 1 "has no install-fixture-client-source-7-1 beside it" \
  "dispatch  a capture with no install artifact beside it is refused" "$WORK/arts"

artifacts "$WORK/arts" 7-1 release source || exit 2
cp -R "$WORK/arts/capture-client-fixture-client-source-7-1" "$WORK/arts/capture-client-fixture-client-source-7-2"
cp -R "$WORK/arts/install-fixture-client-source-7-1" "$WORK/arts/install-fixture-client-source-7-2"
dispatch_is 1 "from 2 runs or attempts" \
  "dispatch  artifacts from two attempts of a run are refused rather than mixed" "$WORK/arts"

artifacts "$WORK/arts" 7-1 release source || exit 2
rm -f "$WORK/arts/install-fixture-client-source-7-1/source/resolution.json"
dispatch_is 1 "source lane has no resolution.json" \
  "dispatch  a lane whose resolver wrote no scheme is refused" "$WORK/arts"

store_report check-assemble/1 cases "$JSON"
