#!/bin/sh
# check-bundle.sh - drive `bit-check assemble-bundle` over a disposable corpus, plant each
# defect it exists to refuse, and hand the bundle it makes to the publisher.
#
# ⭐ THE GREEN CASE ENDS AT THE PUBLISHER, NOT AT THE ASSEMBLER. A bundle is only
# what `publish-data.yml` can append, so the clean bundle goes through
# `publish-data.sh` into a scratch bare repository and is read back there. An
# assembler checked against its own idea of a bundle would agree with itself.
#
# ⛔ EVERY REFUSAL ALSO CHECKS THAT NO BUNDLE WAS LEFT BEHIND. A half-written
# bundle is a publishable-looking directory, which is what `CI-09` kept a
# producer absent to avoid.
#
# Usage:
#   sh scripts/publishing/check-bundle.sh
#   sh scripts/publishing/check-bundle.sh --json
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
      printf 'check-bundle: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# shellcheck disable=SC2034
ME=check-bundle
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo sha256sum git jq tar
BUILDER=$(store_build "$ROOT" build-store) || exit 2
PUBLISHER="$ROOT/scripts/publishing/publish-data.sh"

WORK=$(store_workdir checkbundle) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM
# ⭐ The assembler is `bit-check assemble-bundle` since 2026-10-15, `CI-07`. It
# finds the repository from where it is run, hence the `cd`.
BITCHECK=$(store_bitcheck "$ROOT" "$WORK") || exit 2

# A resolution as `resolve-release.sh` writes it, carrying only what is read.
# ⚠ The generation is the sixth argument, `bit-ids/resolution/2` by default.
resolution() { # file target prefix-or-null min max [schema]
  jq -n --arg t "$2" --argjson p "$3" --argjson a "$4" --argjson b "$5" \
    --arg s "${6:-bit-ids/resolution/2}" \
    '{schema: $s, target: $t,
      scheme: {tag_prefix: $p, min_components: $a, max_components: $b}}' >"$1"
}

fresh_store() {
  rm -rf "$WORK/store" "$WORK/bundle"
  mkdir -p "$WORK/store" || return 1
  "$BUILDER" --version 1.2.3 "$WORK/store" >/dev/null 2>&1
}

# ⛔ Unpiped. The output goes to a file and $? is read on the next line.
run_bundle() { # resolution-args...
  (cd "$ROOT" && "$BITCHECK" assemble-bundle "$@" "$WORK/store" "$WORK/bundle") >"$WORK/out" 2>&1
  RC=$?
}

left() {
  [ -e "$WORK/bundle" ] && printf 'a bundle was left behind' || printf 'no bundle left'
}

resolution "$WORK/res-a.json" fixture-client null 3 3
# ⭐ A first-generation resolution beside a second, as a re-assembly of a capture
# made before the scheme floor meets one made after it.
resolution "$WORK/res-b.json" fixture-client null 3 3 bit-ids/resolution/1
resolution "$WORK/res-gen3.json" fixture-client null 3 3 bit-ids/resolution/3
resolution "$WORK/res-other.json" fixture-client '"v"' 3 3
printf '{"schema": "bit-ids/resolution/2", "target": "fixture-client"}\n' >"$WORK/res-none.json"

fresh_store || {
  printf 'check-bundle: cannot build the fixture store\n' >&2
  exit 2
}

# -- the clean bundle, all the way to the branch ---------------------------------
run_bundle --resolution "$WORK/res-a.json" --resolution "$WORK/res-b.json"
if [ "$RC" = "0" ] && [ -f "$WORK/bundle/MANIFEST.json" ] && [ -f "$WORK/bundle/SHA256SUMS" ]; then
  pass "clean    a corpus assembles into a bundle that describes itself"
else
  fail "clean    exit $RC: $(head -3 "$WORK/out" | tr '\n' ' ')"
fi
MISSING=""
for _path in LICENSE indexes/v1/profiles.json formats/bit-ids-v1.json formats/bit-ids-v1.jsonl \
  formats/bit-ids-v1.csv formats/bit-ids-v1.columns.json formats/bit-ids-v1.sqlite3 \
  formats/bit-ids-v1.cbor; do
  [ -f "$WORK/bundle/$_path" ] || MISSING="$MISSING $_path"
done
if [ -z "$MISSING" ] && [ -n "$(find "$WORK/bundle/raw" -name manifest.json -type f)" ]; then
  pass "layout   every documented path is written, the run manifests among them"
else
  fail "layout   missing:${MISSING:- a run manifest}"
fi
if [ "$RC" = "0" ] && (cd "$WORK/bundle" && sha256sum -c --quiet SHA256SUMS) >/dev/null 2>&1; then
  pass "sums     sha256sum reads every digest the bundle states and they match"
else
  fail "sums     the bundle's checksums do not describe its bytes"
fi
if grep -q 'scheme fixture-client:-:3:3' "$WORK/out"; then
  pass "scheme   the scheme is the one the resolutions state, with no prefix as -"
else
  fail "scheme   expected fixture-client:-:3:3: $(grep schemes "$WORK/out")"
fi

VALIDATOR=$(store_build "$ROOT" validate-corpus) || exit 2
"$VALIDATOR" "$WORK/bundle" >"$WORK/again" 2>&1
VRC=$?
if [ "$VRC" = "0" ] && grep -q '1 of 1 record(s) may enter a published view' "$WORK/again"; then
  pass "corpus   the bundle is still the corpus it was built from"
else
  fail "corpus   exit $VRC: $(head -3 "$WORK/again" | tr '\n' ' ')"
fi

git init -q --bare "$WORK/remote"
sh "$PUBLISHER" --bundle "$WORK/bundle" --remote "$WORK/remote" >"$WORK/pub" 2>&1
PRC=$?
if [ "$PRC" = "0" ] && grep -q 'read back and verified' "$WORK/pub"; then
  pass "publish  the publisher appends the bundle and reads it back"
else
  fail "publish  exit $PRC: $(head -3 "$WORK/pub" | tr '\n' ' ')"
fi

# -- what is refused, and that nothing is left ----------------------------------
fresh_store
run_bundle --resolution "$WORK/res-a.json" --resolution "$WORK/res-other.json"
if [ "$RC" = "1" ] && grep -q 'under two schemes' "$WORK/out" && [ ! -e "$WORK/bundle" ]; then
  pass "scheme   two resolutions ordering one target differently are refused"
else
  fail "scheme   exit $RC, $(left): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

run_bundle --resolution "$WORK/res-none.json"
if [ "$RC" = "1" ] && grep -q 'states no target and version scheme' "$WORK/out" && [ ! -e "$WORK/bundle" ]; then
  pass "scheme   a resolution with no scheme is refused rather than defaulted"
else
  fail "scheme   exit $RC, $(left): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

# ⛔ A generation this assembler has never read is refused, not read as a guess.
run_bundle --resolution "$WORK/res-gen3.json"
if [ "$RC" = "1" ] && grep -q 'states no target and version scheme' "$WORK/out" && [ ! -e "$WORK/bundle" ]; then
  pass "scheme   a resolution of an unknown generation is refused"
else
  fail "scheme   exit $RC, $(left): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

run_bundle
if [ "$RC" = "2" ] && [ ! -e "$WORK/bundle" ]; then
  pass "scheme   no resolution at all cannot run"
else
  fail "scheme   exit $RC, $(left): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

fresh_store
find "$WORK/store/raw" -name manifest.json -type f -exec rm -f {} +
run_bundle --resolution "$WORK/res-a.json"
if [ "$RC" = "1" ] && grep -q 'the store is not a corpus' "$WORK/out" && grep -q 'E-CRP-01' "$WORK/out" &&
  [ ! -e "$WORK/bundle" ]; then
  pass "corpus   a record with no run manifest is refused before anything is copied"
else
  fail "corpus   exit $RC, $(left): $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

fresh_store
mkdir -p "$WORK/store/indexes/v1"
printf '{}\n' >"$WORK/store/indexes/v1/profiles.json"
run_bundle --resolution "$WORK/res-a.json"
if [ "$RC" = "1" ] && grep -q 'carries indexes' "$WORK/out" && [ ! -e "$WORK/bundle" ]; then
  pass "store    a store carrying an index of its own is refused"
else
  fail "store    exit $RC, $(left): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

# ⛔ The defect the first real bundle found: a stored file whose type the release
# cannot name. The release refuses it, and the half-built bundle is removed.
# ⚠ Renamed in place, in the file and in both documents that cite it, so the
# corpus still binds: an undeclared file is `E-CRP-06` and never reaches the
# release, which a first version of this case found.
fresh_store
_dir=$(dirname "$(find "$WORK/store/raw" -name manifest.json -type f | head -1)")
mv "$_dir/control/dht.log" "$_dir/control/dht.err"
for _doc in "$_dir/manifest.json" $(find "$WORK/store/profiles" -name '*.json' -type f); do
  sed 's#"control/dht.log"#"control/dht.err"#' "$_doc" >"$WORK/doc" && cat "$WORK/doc" >"$_doc"
done
run_bundle --resolution "$WORK/res-a.json"
if [ "$RC" = "1" ] && grep -q 'E-REL-01' "$WORK/out" && [ ! -e "$WORK/bundle" ]; then
  pass "release  a stored file with no media type is refused, and no bundle is left"
else
  fail "release  exit $RC, $(left): $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

fresh_store
mkdir -p "$WORK/bundle"
run_bundle --resolution "$WORK/res-a.json"
if [ "$RC" = "2" ] && grep -q 'already exists' "$WORK/out"; then
  pass "out      a bundle directory that already exists is not written into"
else
  fail "out      exit $RC: $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

store_report check-bundle/1 cases "$JSON"
