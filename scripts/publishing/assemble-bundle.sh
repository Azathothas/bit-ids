#!/bin/sh
# assemble-bundle.sh - turn an assembled capture store into the publication
# tree `publish-data.yml` downloads as `bundle`.
#
# ⛔ A CAPTURE STORE IS NOT A PUBLICATION. `assemble-capture` writes records and
# their run manifests; the data branch also carries the indexes, every rendering,
# the licence, and the two documents that describe each published byte. This is
# the one place those are composed, out of the readers that already exist for
# each, so nothing here decides what a record says. `CI-09`.
#
# ⛔ THE STORE MUST BE A CORPUS FIRST. `validate-corpus` reads it before anything
# is copied, and a refusal ends the run with no bundle. A provisional record is
# not a refusal: it belongs in the store, and the indexes are what keep it out
# of a published view. The count of records that may enter one is printed.
#
# ⛔ THE VERSION SCHEME IS NEVER DECLARED HERE. Each adapter declares it in
# `--describe`, each route's resolver records the scheme it ordered tags under
# in its `resolution.json`, and this reads it from there. Two resolutions that order one target differently are
# refused, because the latest view would depend on which was read first.
#
# ⛔ ONLY `profiles/` AND `raw/` ARE COPIED, AND ANYTHING ELSE IS REFUSED. A
# store is those two trees. A store carrying an index or a manifest of its own
# was published or assembled before, and copying it would describe another run.
#
# ⚠ IT DOES NOT READ ITS OWN OUTPUT BACK, AND THAT IS MEASURED RATHER THAN
# OMITTED. A first version ran `sha256sum -c` and `validate-corpus` over the
# bundle before reporting, and a plant disabling either was refused by no case,
# because nothing ahead of them can make them fire short of a broken
# `assemble-release`. The readers that do fire are elsewhere: `publish-data.sh`
# refuses a bundle whose checksums do not match before anything is pushed, and
# `check-bundle.sh` reads the bundle it was handed as a corpus and with
# `sha256sum`.
#
# Usage:
#   sh scripts/publishing/assemble-bundle.sh --resolution FILE... STORE OUT
#
# OUT must not exist. Exit codes: 0 assembled and read back, 1 refused, 2 could
# not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# shellcheck disable=SC2034
ME=assemble-bundle
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

refuse() {
  printf 'assemble-bundle: %s\n' "$1" >&2
  exit 1
}

RESOLUTIONS=""
while [ $# -gt 0 ]; do
  case "$1" in
    --resolution)
      shift
      [ $# -gt 0 ] || {
        printf 'assemble-bundle: --resolution needs a file\n' >&2
        exit 2
      }
      [ -f "$1" ] || {
        printf 'assemble-bundle: %s is not a file\n' "$1" >&2
        exit 2
      }
      RESOLUTIONS="$RESOLUTIONS
$1"
      ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    -*)
      printf 'assemble-bundle: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
    *) break ;;
  esac
  shift
done
[ $# -eq 2 ] || {
  printf 'usage: assemble-bundle.sh --resolution FILE... STORE OUT\n' >&2
  exit 2
}
STORE=$1
OUT=$2
[ -n "$RESOLUTIONS" ] || {
  printf 'assemble-bundle: at least one --resolution is required\n' >&2
  exit 2
}
[ -d "$STORE" ] || {
  printf 'assemble-bundle: %s is not a directory\n' "$STORE" >&2
  exit 2
}
[ ! -e "$OUT" ] || {
  printf 'assemble-bundle: %s already exists\n' "$OUT" >&2
  exit 2
}

store_require jq sha256sum cargo
VALIDATOR=$(store_build "$ROOT" validate-corpus) || exit 2
INDEXER=$(store_build "$ROOT" build-indexes) || exit 2
RENDERER=$(store_build "$ROOT" build-formats) || exit 2
ASSEMBLER=$(store_build "$ROOT" assemble-release) || exit 2

WORK=$(store_workdir assemblebundle) || exit 2
# ⛔ A bundle that was not read back is removed, whatever stopped it: a
# half-built one is a publishable-looking directory.
CREATED=0
DONE=0
# shellcheck disable=SC2317 # invoked by the trap below
cleanup() {
  rm -rf "$WORK"
  [ "$CREATED" = "0" ] || [ "$DONE" = "1" ] || rm -rf "$OUT"
}
trap cleanup EXIT
trap 'exit 2' INT TERM

# -- the schemes, one per target, from what each resolver ordered under --------
# ⚠ BOTH GENERATIONS, because a capture made before `CLIENT-06`'s scheme floor
# still assembles: the floor releases candidates and never changes how two
# versions order, so the index scheme is the same four fields in each.
: >"$WORK/schemes"
while IFS= read -r _file; do
  [ -n "$_file" ] || continue
  _scheme=$(jq -er '
    select(.schema == "bit-ids/resolution/1" or .schema == "bit-ids/resolution/2")
    | "\(.target):\(.scheme.tag_prefix // "-"):\(.scheme.min_components):\(.scheme.max_components)"
    | select(test("^[^:]+:[^:]+:[0-9]+:[0-9]+$"))' "$_file" 2>/dev/null) ||
    refuse "$_file states no target and version scheme a resolver ordered under"
  printf '%s\n' "$_scheme" >>"$WORK/schemes"
done <<EOF
$RESOLUTIONS
EOF
sort -u "$WORK/schemes" >"$WORK/schemes.u"
_conflict=$(cut -d: -f1 "$WORK/schemes.u" | uniq -d | head -1)
[ -z "$_conflict" ] ||
  refuse "the resolutions order $_conflict under two schemes: $(grep "^$_conflict:" "$WORK/schemes.u" | tr '\n' ' ')"
SCHEME_ARGS=""
while IFS= read -r _s; do
  SCHEME_ARGS="$SCHEME_ARGS --scheme $_s"
done <"$WORK/schemes.u"

# -- the store is a corpus, and is only a store --------------------------------
for _entry in "$STORE"/* "$STORE"/.[!.]*; do
  [ -e "$_entry" ] || continue
  case "${_entry##*/}" in
    profiles | raw) : ;;
    *) refuse "the store carries ${_entry##*/}, which no store holds; it describes another run" ;;
  esac
done
[ -d "$STORE/profiles" ] || refuse "the store has no profiles/"

"$VALIDATOR" "$STORE" >"$WORK/corpus.out" 2>&1
RC=$?
if [ "$RC" != "0" ]; then
  printf 'assemble-bundle: the store is not a corpus (exit %s)\n' "$RC" >&2
  sed 's/^/  /' "$WORK/corpus.out" >&2
  exit 1
fi

# -- compose -------------------------------------------------------------------
CREATED=1
mkdir -p "$OUT/indexes/v1" || exit 2
cp -R "$STORE/profiles" "$OUT/profiles" || exit 2
[ ! -d "$STORE/raw" ] || cp -R "$STORE/raw" "$OUT/raw" || exit 2
cp "$ROOT/LICENSE" "$OUT/LICENSE" || exit 2

# ⚠ SCHEME_ARGS is split on purpose: each scheme is one word that cannot hold a
# space, because a target is a slug and the prefix a label.
# shellcheck disable=SC2086
"$INDEXER" $SCHEME_ARGS "$STORE" "$OUT/indexes/v1/profiles.json" >"$WORK/indexes.out" 2>&1 || {
  sed 's/^/  /' "$WORK/indexes.out" >&2
  refuse "the indexes were not built"
}
# shellcheck disable=SC2086
"$RENDERER" $SCHEME_ARGS "$STORE" "$OUT" >"$WORK/formats.out" 2>&1 || {
  sed 's/^/  /' "$WORK/formats.out" >&2
  refuse "the formats were not rendered"
}
"$ASSEMBLER" "$OUT" >"$WORK/release.out" 2>&1 || {
  sed 's/^/  /' "$WORK/release.out" >&2
  refuse "the release was not assembled"
}

DONE=1
printf 'assemble-bundle: %s object(s) in %s\n' "$(find "$OUT" -type f | wc -l | tr -d ' ')" "$OUT"
printf '  schemes:%s\n' "$SCHEME_ARGS"
sed 's/^/  /' "$WORK/corpus.out"
printf '  release: %s\n' "$(head -1 "$WORK/release.out")"
exit 0
