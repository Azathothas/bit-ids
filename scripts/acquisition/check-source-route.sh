#!/bin/sh
# check-source-route.sh - does the `source` route resolve its own tag, from the
# repository's own refs, and refuse every way a refs listing can be wrong?
#
# ⛔ WHAT THIS EXISTS FOR IS AN INDEPENDENCE THAT WAS NEVER THERE. Until
# 2026-09-09 the source lane was handed the tag the RELEASE lane's listing had
# chosen, so both lanes resolved through one index and `E-ACQ-07` called them one
# route - measured by assembling `capture-client` run 14, whose two resolution
# records differ only in their timestamps. The workflow argued for it: one
# resolver is how two lanes cannot land on different versions.
#
# ⭐ ABSOLUTE 4 ANSWERS THAT BETTER. Version equality is checked AFTER
# installation, on what each build reported when asked, so two resolutions that
# disagree are a vendor that moved between two reads and catching it is the
# correct outcome. This harness is what proves the second resolver exists.
#
# -- ⭐ IT RUNS OVER GENERATED LISTINGS AND SAYS SO ---------------------------
#
# `resolve-source --refs` makes the source a file, the way
# `resolve-release --listing` does. ⚠ The listings here are WRITTEN by this
# harness rather than recorded from a vendor, which is a departure from
# `check-release-route` and has a reason: an object name is forty hex digits,
# which is what `check-no-secrets --public` refuses because it is also the shape
# of a credential. `docs/security/secrets.md` says to narrow a pattern rather
# than switch a rule off, and computing the values needs no narrowing at all.
# ⛔ SO THIS PROVES THE READER AND THE ORDERING, and nothing about what any
# vendor tags today. The live measurement is a driven pass in `TODO/ci.md`.
#
# ⛔ NOTHING HERE PLANTS IN A TRACKED FILE. Every listing a case reads is written
# into scratch state, because a gate check that edits a tracked file leaves a
# dirty tree behind when it is interrupted.
#
# Usage:
#   sh scripts/acquisition/check-source-route.sh
#   sh scripts/acquisition/check-source-route.sh --json
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
      printf 'check-source-route: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-source-route
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo sha256sum
store_build "$ROOT" resolve-stable >/dev/null || exit 2

WORK=$(store_workdir checksource) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⭐ THE RESOLVER IS `bit-check resolve-source` SINCE 2026-10-15, `CI-11`. These cases
# held its shell half to the same rows before that half was deleted.
mkdir -p "$WORK/bin" || exit 2
BIT_CHECK=$(store_bitcheck "$ROOT" "$WORK/bin") || exit 2
RESOLVE="$BIT_CHECK"

# ⛔ COMPUTED, NEVER TYPED. See the header: forty hex digits is the shape the
# public secret rule refuses, and it is right to.
oid() { # seed
  printf '%s' "$1" | sha256sum | cut -c1-40
}

# One `git ls-remote --tags --refs` line.
ref_line() { # tag
  printf '%s\trefs/tags/%s\n' "$(oid "$1")" "$1"
}

# The adapter this harness resolves for: a stub declaring the same four fields
# every shipped adapter declares, so nothing here depends on a real target's
# tagging convention holding still.
STUB="$WORK/stub.sh"
cat >"$STUB" <<'STUB'
#!/bin/sh
set -u
case "${1:-}" in
  describe)
    printf 'target=fixture-client\n'
    printf 'kind=stock\n'
    printf 'release_repo=fixture-owner/fixture-client\n'
    printf "release_tag_prefix=${BIT_IDS_STUB_PREFIX-v}\n"
    printf "release_min_components=${BIT_IDS_STUB_MINC-3}\n"
    printf "release_max_components=${BIT_IDS_STUB_MAXC-3}\n"
    printf "release_floor_major=${BIT_IDS_STUB_FLOOR--}\n"
    ;;
  *) exit 2 ;;
esac
STUB
chmod +x "$STUB"

# Runs the resolver over one refs listing, keeping the three streams apart.
# ⛔ The exit code is read from the process that produced it.
resolve() { # tag refs-file
  rm -rf "$WORK/$1.work"
  "$RESOLVE" resolve-source --adapter "$STUB" --workdir "$WORK/$1.work" \
    --record "$WORK/$1.rec" --refs "$2" \
    >"$WORK/$1.out" 2>"$WORK/$1.err"
  return $?
}

said() { # tag literal
  grep -q -F -e "$2" "$WORK/$1.err" "$WORK/$1.out" 2>/dev/null
}

case_is() { # expected-rc literal name tag refs-file
  resolve "$4" "$5"
  _rc=$?
  if [ "$_rc" != "$1" ]; then
    fail "$3 (exit $_rc, expected $1)"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/$4.err" | head -4
    return
  fi
  if [ -n "$2" ] && ! said "$4" "$2"; then
    fail "$3 (exit $1, and it did not say '$2')"
    [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/$4.err" | head -4
    return
  fi
  pass "$3"
}

# -- ⭐ THE CONTROL, WHICH EVERY REFUSAL BELOW NEEDS -------------------------
{
  ref_line v1.2.3
  ref_line v1.2.10
  ref_line v1.2.9
  ref_line v1.10.0
} >"$WORK/refs-good.txt"
case_is 0 "" "a refs listing resolves a tag" good "$WORK/refs-good.txt"
if [ "$(cat "$WORK/good.out" 2>/dev/null)" = "v1.10.0" ]; then
  pass "the newest tag is selected, and 1.10.0 beats 1.2.10 rather than sorting under it"
else
  fail "the newest tag is selected (got [$(cat "$WORK/good.out" 2>/dev/null)])"
fi
# ⛔ 1.2.10 OVER 1.2.9 IS THE ORDERING THIS PROJECT REFUSES TO GET WRONG, and a
# lexical sort answers the other way. The listing above puts them out of order on
# purpose so a reader that preserved input order would fail here.
if grep -q -F -e 'selected_version=1.10.0' "$WORK/good.rec" 2>/dev/null; then
  pass "the record names the version the tag carried, with the prefix stripped"
else
  fail "the record names the version the tag carried, with the prefix stripped"
fi

# ⛔ THE FIELD `assemble-capture` SLUGIFIES INTO A RESOLVER IDENTITY. If this
# named the releases endpoint, the two routes would slugify to one name and
# `E-ACQ-07` would call them one route - which is the defect this whole file
# exists to close, reappearing in the record it writes.
if grep -q -F -e 'source_url=https://recorded.invalid/' "$WORK/good.rec" 2>/dev/null; then
  pass "the record names the refs source it read rather than a releases endpoint"
else
  fail "the record names the refs source it read rather than a releases endpoint"
  [ "$JSON" = "1" ] || sed 's/^/          /' "$WORK/good.rec" | head -6
fi

# -- The refusals -----------------------------------------------------------
#
# ⛔ A LISTING THAT IS NOT A LISTING IS COULD-NOT-RUN, NEVER A REFUSAL, and these
# cases expected 1 until they were run. `resolve-stable` exits 2 when a reader
# cannot make candidates out of the bytes it was handed, which is the same
# verdict `check-release-route` asserts for a releases listing that is not JSON:
# nothing was judged, so nothing was refused. ⚠ Each case therefore asserts the
# REASON as well, because three of these share exit 2 and a case reading the code
# alone would pass when a different one fired - the defect `CI-03`'s Windows
# harness found in itself.

# ⛔ A PEELED REF IS REFUSED RATHER THAN SKIPPED. `ls-remote` without `--refs`
# emits a second `^{}` line per annotated tag; accepting it offers every such tag
# twice and skipping it hides a caller that forgot the flag.
{
  cat "$WORK/refs-good.txt"
  printf '%s\trefs/tags/v1.10.0^{}\n' "$(oid peeled)"
} >"$WORK/refs-peeled.txt"
case_is 2 "could not run" "a peeled ref is refused rather than counted twice" \
  peeled "$WORK/refs-peeled.txt"
said peeled "peeled ref" ||
  fail "the peeled-ref refusal names the peeled ref"
said peeled "peeled ref" &&
  pass "the peeled-ref refusal names the peeled ref"

# ⛔ A BRANCH IS NOT A TAG. `ls-remote` without `--tags` lists heads too, and a
# reader that took them would offer `main` as a version candidate.
{
  cat "$WORK/refs-good.txt"
  printf '%s\trefs/heads/main\n' "$(oid head)"
} >"$WORK/refs-head.txt"
case_is 2 "could not run" "a branch ref is refused" head "$WORK/refs-head.txt"
said head "is not a tag ref" ||
  fail "the branch-ref refusal names the ref"
said head "is not a tag ref" &&
  pass "the branch-ref refusal names the ref"

printf '%s refs/tags/v1.2.3\n' "$(oid space)" >"$WORK/refs-space.txt"
case_is 2 "could not run" "a line with no tab is refused" space "$WORK/refs-space.txt"

printf 'abc123\trefs/tags/v1.2.3\n' >"$WORK/refs-short.txt"
case_is 2 "could not run" "an abbreviated object name is refused" \
  short "$WORK/refs-short.txt"
said short "not a full object name" ||
  fail "the abbreviated-object refusal names the object"
said short "not a full object name" &&
  pass "the abbreviated-object refusal names the object"

: >"$WORK/refs-empty.txt"
case_is 1 "offered no tags at all" "an empty refs listing is refused" \
  empty "$WORK/refs-empty.txt"

# ⛔ THE ONE PRERELEASE SIGNAL A REFS SOURCE STILL HAS IS THE TAG TEXT, and a
# refs listing carries no `prerelease` flag at all. ⭐ So a marker after digits,
# `v2.0.0beta1`, is read as one since 2026-10-15 and passed over: until then
# this case planted it as a tag no scheme could order, which is how
# qBittorrent's source route failed closed on 59 of them. `CLIENT-01`.
{
  cat "$WORK/refs-good.txt"
  ref_line v2.0.0beta1
} >"$WORK/refs-pre.txt"
case_is 0 "" "a prerelease spelled after digits is passed over, not selected" \
  pre "$WORK/refs-pre.txt"
if [ "$(cat "$WORK/pre.out" 2>/dev/null)" = "v1.10.0" ]; then
  pass "and the stable newest still wins over it"
else
  fail "and the stable newest still wins over it (got [$(cat "$WORK/pre.out" 2>/dev/null)])"
fi
# ⛔ AND WHAT SURVIVES IS STILL A BLOCK: a tag whose version text no scheme can
# read and that carries no marker blocks the resolution rather than being
# selected.
{
  cat "$WORK/refs-good.txt"
  ref_line v2.0.0x1
} >"$WORK/refs-unord.txt"
case_is 1 "" "a tag no scheme can order blocks rather than being selected" \
  unord "$WORK/refs-unord.txt"

# ⛔ AND A MISSING DECLARATION IS COULD-NOT-RUN, never a resolution over
# defaults this file invented.
BIT_IDS_STUB_PREFIX='' case_is 2 "no release_tag_prefix" \
  "an adapter declaring no tag prefix cannot run" noprefix "$WORK/refs-good.txt"
BIT_IDS_STUB_MINC=x case_is 2 "non-numeric component count" \
  "a non-numeric component count cannot run" nonnum "$WORK/refs-good.txt"

BIT_IDS_STUB_FLOOR='' case_is 2 "no release_floor_major" \
  "an adapter declaring no scheme floor cannot run" nofloor "$WORK/refs-good.txt"
BIT_IDS_STUB_FLOOR=4x case_is 2 "non-numeric release_floor_major" \
  "a non-numeric scheme floor cannot run" badfloor "$WORK/refs-good.txt"

# -- ⭐ aria2'S OWN TAGS, AND THE FLOOR ITS SOURCE ROUTE NEEDS ----------------
#
# The 152 tags `git ls-remote --tags --refs` answered for `aria2/aria2` on
# 2026-10-21, every one `release-` and written here without it. Eleven are
# spellings the three-component scheme cannot order, `0.4.0a` and ten package
# revisions such as `0.10.0+1`, all under major 0, so without a floor the source
# route fails closed on its own history. `CLIENT-05`.
ARIA2_TAGS='0.1.0 0.2.0 0.2.1 0.2.2 0.3.0 0.3.1 0.3.2 0.3.3 0.4.0 0.4.0a 0.4.1
0.5.0 0.5.1 0.5.2 0.6.0 0.6.0+1 0.7.0 0.7.1 0.7.2 0.7.3 0.8.0 0.8.1 0.9.0
0.10.0 0.10.0+1 0.10.1 0.10.2 0.10.2+1 0.11.0 0.11.1 0.11.1+1 0.11.2 0.11.3
0.11.4 0.11.5 0.12.0 0.12.1 0.13.0 0.13.0+1 0.13.1 0.13.1+1 0.13.2 0.13.2+1
0.14.0 0.14.0+1 0.15.0 0.15.1 0.15.1+1 0.15.1+2 0.15.2 0.15.3 0.16.0 0.16.1
0.16.2 1.0.0 1.0.1 1.1.0 1.1.1 1.1.2 1.2.0 1.3.0 1.3.1 1.3.2 1.3.3 1.4.0 1.4.1
1.5.0 1.5.1 1.5.2 1.6.0 1.6.1 1.6.2 1.6.3 1.7.0 1.7.1 1.7.2 1.8.0 1.8.1 1.8.2
1.8.3 1.9.0 1.9.1 1.9.2 1.9.3 1.9.4 1.9.5 1.10.0 1.10.1 1.10.2 1.10.3 1.10.4
1.10.5 1.10.6 1.10.7 1.10.8 1.10.9 1.11.0 1.11.1 1.11.2 1.12.0 1.12.1 1.13.0
1.14.0 1.14.1 1.14.2 1.15.0 1.15.1 1.15.2 1.16.0 1.16.1 1.16.2 1.16.3 1.16.4
1.16.5 1.17.0 1.17.1 1.18.0 1.18.1 1.18.2 1.18.3 1.18.4 1.18.5 1.18.6 1.18.7
1.18.8 1.18.9 1.18.10 1.19.0 1.19.1 1.19.2 1.19.3 1.20.0 1.21.0 1.22.0 1.23.0
1.24.0 1.25.0 1.26.0 1.26.1 1.27.0 1.27.1 1.28.0 1.29.0 1.30.0 1.31.0 1.32.0
1.33.0 1.33.1 1.34.0 1.35.0 1.36.0 1.37.0'
for _t in $ARIA2_TAGS; do
  ref_line "release-$_t"
done >"$WORK/refs-aria2.txt"
_count=$(wc -l <"$WORK/refs-aria2.txt" | tr -d ' ')
if [ "$_count" = 152 ]; then
  pass "the replayed aria2 listing carries the 152 refs measured"
else
  fail "the replayed aria2 listing carries the 152 refs measured (got $_count)"
fi
BIT_IDS_STUB_PREFIX=release- case_is 1 "failed closed" \
  "with no floor, aria2's refs fail closed on their 0.x history" \
  a2nofloor "$WORK/refs-aria2.txt"
BIT_IDS_STUB_PREFIX=release- BIT_IDS_STUB_FLOOR=1 case_is 0 "" \
  "with a floor of 1, aria2's refs resolve" a2floor "$WORK/refs-aria2.txt"
if [ "$(cat "$WORK/a2floor.out" 2>/dev/null)" = "release-1.37.0" ]; then
  pass "and select release-1.37.0, the release route's answer on 2026-10-21"
else
  fail "and select release-1.37.0 (got [$(cat "$WORK/a2floor.out" 2>/dev/null)])"
fi
_released=$(grep -c -F -e '"verdict": "below_scheme_floor"' "$WORK/a2floor.work/resolution.json" 2>/dev/null)
if [ "$_released" = 11 ]; then
  pass "the floor released exactly the 11 tags the scheme could not order"
else
  fail "the floor released exactly the 11 tags the scheme could not order (got [$_released])"
fi
BIT_IDS_STUB_PREFIX=release- BIT_IDS_STUB_FLOOR=2 case_is 1 "failed closed" \
  "a floor of 2 is above aria2's winner and releases nothing" a2high "$WORK/refs-aria2.txt"

# -- ⭐ TRANSMISSION'S OWN TAGS, AND THE FLOOR THAT LETS THEM RESOLVE ---------
#
# The 132 tag names `git ls-remote --tags --refs` answered for
# `transmission/transmission` on 2026-10-15, with object names computed as
# everywhere above. 112 are spellings the three-component scheme cannot order,
# `0.5` to `3.00`, and refs carry no date, so without a floor the source route
# fails closed on its own history. `CLIENT-06`.
# ⚠ THE TAG NAMES ARE THE MEASUREMENT; the object names are not, and nothing here
# reads them past the reader's shape check.
# ⚠ Split on whitespace on purpose: no tag carries any.
TRANSMISSION_TAGS='0.5 0.6 0.6.1 0.7 0.71 0.72 0.80 0.82 0.90 0.91 0.92 0.93 0.94 0.95 0.96
1.00 1.01 1.02 1.03 1.04 1.05 1.06 1.10 1.11 1.20 1.21 1.22 1.31 1.32
1.33 1.34 1.40 1.41 1.42 1.50 1.51 1.52 1.53 1.54 1.60 1.61 1.70 1.71
1.72 1.73 1.74 1.75 1.76 1.80 1.80b5 1.81 1.82 1.83 1.90 1.91 1.92 1.93
2.00 2.01 2.02 2.03 2.04 2.05 2.10 2.11 2.12 2.13 2.20 2.20b1 2.20b2
2.20b3 2.20b4 2.21 2.22 2.30 2.30b1 2.30b2 2.30b3 2.30b4 2.31 2.32 2.33
2.40 2.40b1 2.40b2 2.40b3 2.41 2.42 2.50 2.50b1 2.51 2.52 2.60 2.61 2.70
2.71 2.72 2.73 2.74 2.75 2.76 2.77 2.80 2.81 2.82 2.83 2.84 2.90 2.91
2.92 2.93 2.94 3.00 4.0.0 4.0.0-beta.1 4.0.0-beta.2 4.0.0-beta.3 4.0.1
4.0.2 4.0.3 4.0.4 4.0.5 4.0.6 4.1.0 4.1.0-beta.1 4.1.0-beta.2
4.1.0-beta.3 4.1.0-beta.4 4.1.0-beta.5 4.1.1 4.1.2 4.1.3'
for _t in $TRANSMISSION_TAGS; do
  ref_line "$_t"
done >"$WORK/refs-transmission.txt"
_count=$(wc -l <"$WORK/refs-transmission.txt" | tr -d ' ')
if [ "$_count" = 132 ]; then
  pass "the replayed listing carries the 132 refs measured"
else
  fail "the replayed listing carries the 132 refs measured (got $_count)"
fi
BIT_IDS_STUB_PREFIX=- case_is 1 "failed closed" \
  "with no floor, transmission's refs fail closed on their own history" \
  trnofloor "$WORK/refs-transmission.txt"
BIT_IDS_STUB_PREFIX=- BIT_IDS_STUB_FLOOR=4 case_is 0 "" \
  "with a floor of 4, transmission's refs resolve" trfloor "$WORK/refs-transmission.txt"
if [ "$(cat "$WORK/trfloor.out" 2>/dev/null)" = "4.1.3" ]; then
  pass "and select 4.1.3, the release route's answer on 2026-10-15"
else
  fail "and select 4.1.3 (got [$(cat "$WORK/trfloor.out" 2>/dev/null)])"
fi
# ⚠ 99 AND NOT 112 SINCE 2026-10-16: thirteen of the 112, `1.80b5` to
# `2.50b1`, are PEP 440 betas the prerelease reader now reads as such, so they
# never reach the floor. `CLIENT-07`.
_released=$(grep -c -F -e '"verdict": "below_scheme_floor"' "$WORK/trfloor.work/resolution.json" 2>/dev/null)
if [ "$_released" = 99 ]; then
  pass "the floor released exactly the 99 tags the scheme could not order and no marker reads"
else
  fail "the floor released exactly the 99 tags the scheme could not order and no marker reads (got [$_released])"
fi
_betas=$(grep -c -F -e '"verdict": "prerelease_by_version"' "$WORK/trfloor.work/resolution.json" 2>/dev/null)
if [ "$_betas" = 21 ]; then
  pass "and the 13 lettered betas are prereleases beside the 8 dashed ones"
else
  fail "and the 13 lettered betas are prereleases beside the 8 dashed ones (got [$_betas])"
fi
# ⛔ A FLOOR ABOVE THE WINNER RELEASES NOTHING, and `3.00` sitting AT a floor of
# 3 still blocks: the floor speaks only for what is strictly below it.
BIT_IDS_STUB_PREFIX=- BIT_IDS_STUB_FLOOR=5 case_is 1 "failed closed" \
  "a floor above the winner releases nothing" trhigh "$WORK/refs-transmission.txt"
BIT_IDS_STUB_PREFIX=- BIT_IDS_STUB_FLOOR=3 case_is 1 "failed closed" \
  "a tag at the floor still blocks" trat "$WORK/refs-transmission.txt"

# -- ⭐ DELUGE'S OWN TAGS, AND THE PEP 440 BETAS THAT FAILED THEM CLOSED -----
#
# The 85 tag names `git ls-remote --tags --refs` answered for
# `deluge-torrent/deluge` on 2026-10-16. `deluge-2.0.0b1` and `deluge-2.0b1`
# are betas in PEP 440's one-letter spelling, which the reader did not know, so
# the scheme could not order them and the source route failed closed at the
# current major. `CLIENT-07`.
DELUGE_TAGS='archive/0.5-stable archive/1.0-stable archive/1.1-stable
archive/1.2-stable archive/1.3-stable deluge-0.5.0 deluge-0.5.1 deluge-0.5.1.1
deluge-0.5.2 deluge-0.5.3 deluge-0.5.4 deluge-0.5.4.1 deluge-0.5.5
deluge-0.5.5.95 deluge-0.5.5_RC1 deluge-0.5.6 deluge-0.5.6.1 deluge-0.5.6.2
deluge-0.5.6.95 deluge-0.5.6.96 deluge-0.5.7 deluge-0.5.7.1 deluge-0.5.8.1
deluge-0.5.8.4 deluge-0.5.8.5 deluge-0.5.8.7 deluge-1.0.0 deluge-1.0.1
deluge-1.0.3 deluge-1.0.4 deluge-1.0.6 deluge-1.0.7 deluge-1.1.0 deluge-1.1.1
deluge-1.1.2 deluge-1.1.3 deluge-1.1.4 deluge-1.1.5 deluge-1.1.6 deluge-1.1.7
deluge-1.1.8 deluge-1.1.9 deluge-1.2.0 deluge-1.2.0_rc1 deluge-1.2.0_rc2
deluge-1.2.0_rc3 deluge-1.2.0_rc4 deluge-1.2.0_rc5 deluge-1.2.1 deluge-1.2.2
deluge-1.2.3 deluge-1.3.0 deluge-1.3.0_rc1 deluge-1.3.0_rc2 deluge-1.3.1
deluge-1.3.10 deluge-1.3.11 deluge-1.3.12 deluge-1.3.13 deluge-1.3.14
deluge-1.3.15 deluge-1.3.2 deluge-1.3.3 deluge-1.3.4 deluge-1.3.5 deluge-1.3.6
deluge-1.3.7 deluge-1.3.8 deluge-1.3.9 deluge-2.0.0 deluge-2.0.0.dev0
deluge-2.0.0b1 deluge-2.0.1 deluge-2.0.2 deluge-2.0.3 deluge-2.0.4 deluge-2.0.5
deluge-2.0b1 deluge-2.1.0 deluge-2.1.0.dev0 deluge-2.1.1 deluge-2.1.1.dev0
deluge-2.1.2.dev0 deluge-2.2.0 deluge-2.2.1.dev0'
for _t in $DELUGE_TAGS; do
  ref_line "$_t"
done >"$WORK/refs-deluge.txt"
_count=$(wc -l <"$WORK/refs-deluge.txt" | tr -d ' ')
if [ "$_count" = 85 ]; then
  pass "the replayed deluge listing carries the 85 refs measured"
else
  fail "the replayed deluge listing carries the 85 refs measured (got $_count)"
fi
BIT_IDS_STUB_PREFIX=deluge- BIT_IDS_STUB_FLOOR=2 case_is 0 "" \
  "with a floor of 2, deluge's refs resolve" dlfloor "$WORK/refs-deluge.txt"
if [ "$(cat "$WORK/dlfloor.out" 2>/dev/null)" = "deluge-2.2.0" ]; then
  pass "and select deluge-2.2.0, the newest stable on 2026-10-16"
else
  fail "and select deluge-2.2.0 (got [$(cat "$WORK/dlfloor.out" 2>/dev/null)])"
fi
for _beta in deluge-2.0.0b1 deluge-2.0b1; do
  if grep -A5 -F -e "\"tag\": \"$_beta\"" "$WORK/dlfloor.work/resolution.json" 2>/dev/null |
    grep -q -F -e '"verdict": "prerelease_by_version"'; then
    pass "$_beta is read as the beta it is"
  else
    fail "$_beta is read as the beta it is"
  fi
done

store_report check-source-route/1 cases "$JSON"
