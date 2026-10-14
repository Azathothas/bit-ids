#!/bin/sh
# assemble-dispatch.sh - turn one `capture-client` dispatch's downloaded
# artifacts into its records, their run manifests, and the bundle
# `publish-data.yml` downloads.
#
# ⭐ THE LANES ARE READ FROM THE ARTIFACT NAMES, NOT DECLARED. Each lane uploads
# `capture-client-<adapter>-<route>-<run>-<attempt>` and
# `install-<adapter>-<route>-<run>-<attempt>`, and this pairs them. An adapter
# may carry a hyphen, so the name is read from its end: the attempt, the run,
# then the route, which is one of package, release or source.
#
# ⛔ EVERY ADAPTER IS ASSEMBLED OR THE RUN IS REFUSED. An adapter with one lane,
# or a capture with no install artifact beside it, is a dispatch that did not
# produce a pair, and a bundle that quietly left it out would read as complete.
# So is a directory carrying two runs or two attempts.
#
# ⛔ NOTHING HERE DECIDES WHAT A RECORD SAYS. `assemble-capture` writes the
# records and their manifests; `assemble-bundle.sh` composes the publication,
# taking each target's version scheme from the resolutions the install
# artifacts carry. `CI-09`.
#
# Usage:
#   sh scripts/capture/assemble-dispatch.sh ARTIFACTS OUT
#
# ARTIFACTS holds one directory per downloaded artifact, named as uploaded. OUT
# must not exist. Exit codes: 0 assembled and read back, 1 refused, 2 could not
# run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# shellcheck disable=SC2034
ME=assemble-dispatch
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

refuse() {
  printf 'assemble-dispatch: %s\n' "$1" >&2
  exit 1
}

[ $# -eq 2 ] || {
  printf 'usage: assemble-dispatch.sh ARTIFACTS OUT\n' >&2
  exit 2
}
ARTIFACTS=$1
OUT=$2
[ -d "$ARTIFACTS" ] || {
  printf 'assemble-dispatch: %s is not a directory\n' "$ARTIFACTS" >&2
  exit 2
}
[ ! -e "$OUT" ] || {
  printf 'assemble-dispatch: %s already exists\n' "$OUT" >&2
  exit 2
}

ASSEMBLE=$(store_build "$ROOT" assemble-capture bit-ids-probe) || exit 2

WORK=$(store_workdir assembledispatch) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# -- the lanes, from the names --------------------------------------------------
# One row per capture: adapter route run-attempt.
: >"$WORK/lanes"
for _dir in "$ARTIFACTS"/capture-client-*; do
  [ -d "$_dir" ] || continue
  _rest=${_dir##*/capture-client-}
  _attempt=${_rest##*-}
  _rest=${_rest%-*}
  _run=${_rest##*-}
  _rest=${_rest%-*}
  _route=${_rest##*-}
  _adapter=${_rest%-*}
  case "$_attempt$_run" in
    '' | *[!0-9]*) refuse "${_dir##*/} does not end in a run id and an attempt" ;;
  esac
  case "$_route" in
    package | release | source) : ;;
    *) refuse "${_dir##*/} names no route this workflow runs" ;;
  esac
  if [ -z "$_adapter" ] || [ "$_adapter" = "$_route" ]; then
    refuse "${_dir##*/} names no adapter"
  fi
  [ -d "$ARTIFACTS/install-$_adapter-$_route-$_run-$_attempt" ] ||
    refuse "${_dir##*/} has no install-$_adapter-$_route-$_run-$_attempt beside it"
  printf '%s %s %s-%s\n' "$_adapter" "$_route" "$_run" "$_attempt" >>"$WORK/lanes"
done
[ -s "$WORK/lanes" ] || refuse "$ARTIFACTS holds no capture-client artifact"

_runs=$(cut -d' ' -f3 "$WORK/lanes" | sort -u | wc -l | tr -d ' ')
[ "$_runs" = "1" ] ||
  refuse "the artifacts come from $_runs runs or attempts: $(cut -d' ' -f3 "$WORK/lanes" | sort -u | tr '\n' ' ')"
RUN=$(cut -d' ' -f3 "$WORK/lanes" | head -1)

# ⚠ An install artifact with no capture beside it is a lane that installed and
# never captured, and the pair it belonged to is refused below by its count.
for _dir in "$ARTIFACTS"/install-*; do
  [ -d "$_dir" ] || continue
  _rest=${_dir##*/install-}
  _rest=${_rest%-"$RUN"}
  [ "$_rest" != "${_dir##*/install-}" ] ||
    refuse "${_dir##*/} is not from run $RUN"
done

# -- one store, every adapter in it ---------------------------------------------
mkdir -p "$WORK/store" || exit 2
: >"$WORK/resolutions"
# ⚠ Redirected files rather than pipes, so `refuse` exits this script rather
# than a subshell feeding the loop.
cut -d' ' -f1 "$WORK/lanes" | sort -u >"$WORK/adapters"
while IFS= read -r _adapter; do
  _count=$(awk -v a="$_adapter" '$1 == a' "$WORK/lanes" | wc -l | tr -d ' ')
  [ "$_count" -ge 2 ] ||
    refuse "$_adapter captured on $_count route; a record needs two independent routes"
  _args=""
  awk -v a="$_adapter" '$1 == a { print $2 }' "$WORK/lanes" | sort >"$WORK/routes"
  while IFS= read -r _route; do
    _cap="$ARTIFACTS/capture-client-$_adapter-$_route-$RUN"
    _ins="$ARTIFACTS/install-$_adapter-$_route-$RUN"
    _args="$_args --lane $_cap,$_ins"
    _res="$_ins/$_route/resolution.json"
    [ -f "$_res" ] || _res="$_ins/release/resolution.json"
    [ -f "$_res" ] || refuse "the $_adapter $_route lane has no resolution.json"
    printf '%s\n' "$_res" >>"$WORK/resolutions"
  done <"$WORK/routes"
  # ⚠ Split on purpose: each lane is one word, and an artifact name carries no
  # space.
  # shellcheck disable=SC2086
  "$ASSEMBLE" $_args "$WORK/store" </dev/null >"$WORK/assemble.out" 2>&1
  RC=$?
  sed 's/^/  /' "$WORK/assemble.out"
  [ "$RC" = "0" ] || refuse "$_adapter did not assemble (exit $RC)"
done <"$WORK/adapters"

_resargs=""
while IFS= read -r _res; do
  _resargs="$_resargs --resolution $_res"
done <"$WORK/resolutions"
# shellcheck disable=SC2086
sh "$ROOT/scripts/publishing/assemble-bundle.sh" $_resargs "$WORK/store" "$OUT"
RC=$?
[ "$RC" = "0" ] || exit "$RC"
printf 'assemble-dispatch: run %s, %s lane(s), bundle at %s\n' "$RUN" \
  "$(wc -l <"$WORK/lanes" | tr -d ' ')" "$OUT"
exit 0
