#!/bin/sh
# check-release-route.sh - does every shipped adapter's release route resolve to
# exactly one artifact, and does the resolver refuse every way it can be wrong?
#
# ⛔ WHAT THIS EXISTS FOR IS THE AMBIGUOUS MATCH, and it is not hypothetical.
# qBittorrent's release-5.2.3 publishes TWO Linux AppImages differing only by an
# `_lt20` between the version and the architecture, so a pattern written
# `qbittorrent-*_x86_64.AppImage` matches both. A selector that took the first
# would install whichever the vendor happened to list first, and the record would
# look identical the month that order changed.
#
# -- ⭐ IT RUNS OVER RECORDED RESPONSES AND SAYS SO ---------------------------
#
# `resolve-release --listing` makes the source a file, the way
# `assert-disposable --route-table` makes the routing table one. A rule that can
# only be exercised by reaching three vendors is a rule the gate cannot run, and
# a gate that reached three vendors would go red when one of them was down.
#
# ⚠ SO THIS PROVES THE PATTERN AGAINST A RELEASE THAT PROVABLY DID NOT MOVE, and
# nothing about what those vendors publish today. `listings/README.md` says how
# the recorded responses were made and what re-making them costs.
#
# ⛔ NOTHING HERE PLANTS IN A TRACKED FILE. Every adapter a case mutates is a
# copy in scratch state, because a gate check that edits a tracked file leaves a
# dirty tree behind when it is interrupted.
#
# Usage:
#   sh scripts/acquisition/check-release-route.sh
#   sh scripts/acquisition/check-release-route.sh --json
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
      printf 'check-release-route: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck disable=SC2034
ME=check-release-route
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo sha256sum
store_build "$ROOT" resolve-stable >/dev/null || exit 2
store_build "$ROOT" select-asset >/dev/null || exit 2

WORK=$(store_workdir checkrelease) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

RESOLVE="$ROOT/scripts/acquisition/resolve-release.sh"
ADAPTERS="$ROOT/scripts/capture/adapters"
LISTINGS="$HERE/listings"
[ -f "$RESOLVE" ] || {
  printf 'check-release-route: %s is missing\n' "$RESOLVE" >&2
  exit 2
}

# ⛔ THE CHECKSUMS DOCUMENT IS BUILT HERE RATHER THAN COMMITTED, AND THAT IS TWO
# DECISIONS. A stored `.sha256` fixture is a file of bare 64-digit hex, which is
# exactly what `check-no-secrets --public` refuses and is right to - a digest in
# this project is spelled with its algorithm. And a document GENERATED from the
# listing's own asset names is about the resolver's reading rather than about a
# blob somebody pasted: the case below plants a name into it and watches the
# refusal.
#
# ⚠ The digests are not the vendor's and do not need to be. Nothing here fetches
# an artifact, so nothing verifies one; what this exercises is whether the
# resolver finds the document, reads it, and refuses one that does not name the
# asset it selected. `check-rootless.sh` is where a digest is actually compared.
sums_for() { # asset-name > document
  printf '%s  %s\n' \
    "$(printf 'not-the-vendors-bytes-%s' "$1" | sha256sum | cut -d' ' -f1)" "$1"
}

# Runs the resolver over one adapter and one recorded listing, keeping the three
# streams apart. ⛔ The exit code is read from the process that produced it.
# ⚠ `--checksums` travels with `--listing` because the resolver refuses the pair
# apart: a recorded listing names real vendor URLs, so resolving a digest from
# one would reach the network out of a harness written not to.
resolve() { # tag adapter listing [checksums-asset]
  _out="$WORK/$1.out"
  _err="$WORK/$1.err"
  _sums="$WORK/$1.sums"
  if [ $# -ge 4 ] && [ -n "$4" ]; then
    sums_for "$4" >"$_sums"
  else
    : >"$_sums"
  fi
  sh "$RESOLVE" --adapter "$2" --workdir "$WORK/$1.work" \
    --record "$WORK/$1.rec" --listing "$3" --checksums "$_sums" >"$_out" 2>"$_err"
  return $?
}

said() { # tag literal
  grep -q -F -e "$2" "$WORK/$1.err"
}

# -- 1. every shipped adapter selects exactly one artifact --------------------
#
# ⛔ THE EXPECTED ASSET IS NAMED HERE AND THE PATTERN IS NAMED IN THE ADAPTER,
# and the recorded listing is a third file. Three places, and this is the thing
# that compares them: a pattern widened in an adapter changes which asset comes
# out, and the case says so by name rather than by counting matches.
#
# ⚠ A TABLE THROUGH A HERE-DOCUMENT, not a pipe. conventions/shell.md section 4:
# a `while read` on the right of a pipe runs in a subshell and its counters are
# discarded, so every row would vanish from the report.
while read -r target expected; do
  [ -n "$target" ] || continue
  if ! resolve "$target" "$ADAPTERS/$target.sh" "$LISTINGS/$target.json" "$expected"; then
    fail "asset     $target: the resolver exited $? - $(tail -1 "$WORK/$target.err")"
    continue
  fi
  got=$(sed -n 's/^asset=//p' "$WORK/$target.rec")
  url=$(sed -n 's/^asset_url=//p' "$WORK/$target.rec")
  if [ "$got" != "$expected" ]; then
    fail "asset     $target: selected [$got], expected [$expected]"
  elif [ "$url" != "$(cat "$WORK/$target.out")" ]; then
    fail "asset     $target: the printed URL is not the record's asset_url"
  else
    pass "asset     $target: one asset, $got"
  fi
done <<ADAPTER_TABLE
aria2 aria2-1.37.0.tar.bz2
aria2-next aria2-next-2.7.4-linux-x86_64
qbittorrent qbittorrent-5.2.3_lt20_x86_64.AppImage
transmission transmission-4.1.3.tar.xz
ADAPTER_TABLE

# -- 1b. a newest release that carries no assets at all ----------------------
#
# ⛔ THIS IS NOT A HYPOTHETICAL AND IT IS NOT A VENDOR BEHAVING BADLY. Measured
# on 2026-09-09: `AnInsomniacy/aria2-next` published `v2.7.5` at 12:11:10Z and
# the release carried ZERO assets when it was read 47 seconds later, while
# `v2.7.4` beside it carried eight. A release exists from the moment it is
# created and its binaries arrive when whatever builds them finishes, so every
# target with a release route has a window in which its newest release is empty.
#
# ⭐ THE QUESTION IS NOT WHETHER THE RUN FAILS - IT IS WHETHER IT FALLS BACK.
# `resolve-stable` picks the newest release and `select-asset` then reads assets
# off the one that won. A resolver that quietly took `v2.7.4` because `v2.7.5`
# had nothing to offer would be targeting a release that is not the newest,
# which rule 5 forbids, and it would do it on a lane nobody was watching. ⚠ A
# fixture holding the empty release ALONE cannot ask this: the fallback needs an
# older release with assets sitting right there to fall back TO, which is why
# `aria2-next-unpublished.json` keeps both.
resolve unpublished "$ADAPTERS/aria2-next.sh" "$LISTINGS/aria2-next-unpublished.json"
rc=$?
if [ "$rc" = 1 ] && said unpublished "no asset of that release matches"; then
  # ⛔ AND THE VERSION IT REFUSED OVER IS READ BACK, because "it refused" is
  # satisfied by refusing for any reason at all. This asserts it got as far as
  # the NEWEST release and stopped there.
  if said unpublished "2.7.5"; then
    pass "unpublished a newest release with no assets is refused, naming 2.7.5"
  else
    fail "unpublished refused without naming 2.7.5: $(tail -2 "$WORK/unpublished.err")"
  fi
else
  fail "unpublished a newest release with no assets exited $rc: $(tail -1 "$WORK/unpublished.err")"
fi

# -- 2. the declared repository is the catalogue's upstream ------------------
#
# ⛔ ONE FACT, TWO FILES, AND THIS IS WHAT COMPARES THEM. `catalogue/clients.toml`
# carries each target's upstream and the adapter carries the path a route
# resolves through; they are different sentences about one repository, and a
# vendor that moves leaves them disagreeing with nothing to say so.
CATALOGUE="$ROOT/catalogue/clients.toml"
for one in "$ADAPTERS"/*.sh; do
  name=$(basename "$one" .sh)
  repo=$(sh "$one" describe 2>/dev/null | sed -n 's/^release_repo=//p')
  [ -n "$repo" ] || continue
  upstream=$(awk -v want="\"$name\"" '
    $1 == "id" && $3 == want { found = 1; next }
    found && $1 == "upstream" { print $3; exit }
  ' "$CATALOGUE" | tr -d '"')
  if [ -z "$upstream" ]; then
    fail "catalogue $name declares release_repo=$repo and the catalogue names no upstream"
  elif [ "$upstream" = "https://github.com/$repo" ]; then
    pass "catalogue $name: $repo is the catalogue's upstream"
  else
    fail "catalogue $name declares $repo and the catalogue says $upstream"
  fi
done

# -- 3. the refusals, each planted into a COPY and read by its own message ----
#
# ⛔ A PLANT WHOSE EXPECTED OUTCOME IS A PASS PROVES NOTHING. Each case below
# makes the run FAIL and then reads WHICH failure it was: the codes are 1
# (refused) and 2 (could not run), and a case that only asserted "non-zero" would
# be satisfied by a resolver that could not start.
STUB="$WORK/no-release.sh"
cat >"$STUB" <<'STUB_ADAPTER'
#!/bin/sh
# A stub that declares no release route at all, which is the ordinary state of
# every adapter this project has not measured a second route for.
[ "${1:-}" = describe ] || exit 2
printf 'target=stub\n'
printf 'kind=stub\n'
STUB_ADAPTER

resolve no-release "$STUB" "$LISTINGS/aria2.json"
rc=$?
if [ "$rc" = 2 ] && said no-release "declares no release_repo"; then
  pass "refusal   an adapter with no release route is could-not-run, and says so"
else
  fail "refusal   an adapter with no release route exited $rc: $(tail -1 "$WORK/no-release.err")"
fi

# ⚠ A PARTIAL DECLARATION IS ITS OWN CASE. An adapter that names the repository
# and forgets the pattern would otherwise reach `select-asset` with an empty
# pattern, which parses as nothing and matches nothing - a refusal, but the wrong
# one, naming a vendor's assets rather than the adapter's own omission.
PARTIAL="$WORK/partial.sh"
cat >"$PARTIAL" <<'PARTIAL_ADAPTER'
#!/bin/sh
[ "${1:-}" = describe ] || exit 2
printf 'target=stub\n'
printf 'kind=stub\n'
printf 'release_repo=aria2/aria2\n'
printf 'release_tag_prefix=release-\n'
printf 'release_min_components=3\n'
printf 'release_max_components=3\n'
PARTIAL_ADAPTER

resolve partial "$PARTIAL" "$LISTINGS/aria2.json"
rc=$?
if [ "$rc" = 2 ] && said partial "declares no release_asset pattern"; then
  pass "refusal   a declaration missing the asset pattern is could-not-run"
else
  fail "refusal   a partial declaration exited $rc: $(tail -1 "$WORK/partial.err")"
fi

# ⛔ THE CASE THIS HARNESS EXISTS FOR. The aria2 release carries three source
# archives differing only in compression, so a widened pattern matches three and
# the run must refuse rather than take one.
plant() { # tag literal replacement
  cp "$ADAPTERS/aria2.sh" "$WORK/$1.sh" || return 1
  replace_once "$WORK/$1.sh" "$2" "$3"
}

if plant ambiguous 'release_asset=aria2-{version}.tar.bz2' 'release_asset=aria2-*.tar.*'; then
  resolve ambiguous "$WORK/ambiguous.sh" "$LISTINGS/aria2.json"
  rc=$?
  if [ "$rc" = 1 ] && said ambiguous "3 assets match the pattern"; then
    pass "refusal   a pattern matching three assets is refused and names all three"
  else
    fail "refusal   the ambiguous pattern exited $rc: $(tail -1 "$WORK/ambiguous.err")"
  fi
else
  fail "plant     the ambiguous pattern did not apply"
fi

if plant nomatch 'release_asset=aria2-{version}.tar.bz2' 'release_asset=aria2-{version}.deb'; then
  resolve nomatch "$WORK/nomatch.sh" "$LISTINGS/aria2.json"
  rc=$?
  if [ "$rc" = 1 ] && said nomatch "no asset of that release matches"; then
    pass "refusal   a pattern matching nothing is refused"
  else
    fail "refusal   the unmatched pattern exited $rc: $(tail -1 "$WORK/nomatch.err")"
  fi
else
  fail "plant     the unmatched pattern did not apply"
fi

# ⚠ A TYPO IN A PLACEHOLDER IS could-not-run RATHER THAN refused, because it is
# this repository's defect and not a vendor's. A pattern reader that matched
# `{ver}` as literal text would find nothing and report the vendor's asset list.
if plant typo 'release_asset=aria2-{version}.tar.bz2' 'release_asset=aria2-{ver}.tar.bz2'; then
  resolve typo "$WORK/typo.sh" "$LISTINGS/aria2.json"
  rc=$?
  if [ "$rc" = 2 ] && said typo "the only one is {version}"; then
    pass "refusal   an unknown placeholder is could-not-run, not a silent no-match"
  else
    fail "refusal   the placeholder typo exited $rc: $(tail -1 "$WORK/typo.err")"
  fi
else
  fail "plant     the placeholder typo did not apply"
fi

# ⛔ AND A SCHEME THE TAGS DO NOT FIT MUST FAIL CLOSED AT THE RESOLVER, not
# arrive here as an asset nobody chose. aria2 tags carry `release-`; a prefix of
# `v` makes every candidate foreign and the resolution selects nothing.
if plant prefix 'release_tag_prefix=release-' 'release_tag_prefix=v'; then
  resolve prefix "$WORK/prefix.sh" "$LISTINGS/aria2.json"
  rc=$?
  if [ "$rc" = 1 ] && said prefix "the resolver failed closed"; then
    pass "refusal   a tag prefix no candidate carries fails closed before any asset"
  else
    fail "refusal   the wrong tag prefix exited $rc: $(tail -1 "$WORK/prefix.err")"
  fi
else
  fail "plant     the tag prefix did not apply"
fi

# ⚠ AND A LISTING THAT IS NOT A LISTING. The readers are handed bytes from a
# vendor, so the case a gate cannot skip is the one where those bytes are not
# what the reader expects.
printf 'this is not a release listing\n' >"$WORK/garbage.json"
resolve garbage "$ADAPTERS/aria2.sh" "$WORK/garbage.json"
rc=$?
if [ "$rc" = 2 ] && said garbage "resolve-stable could not run"; then
  pass "refusal   a listing that is not JSON is could-not-run"
else
  fail "refusal   the unreadable listing exited $rc: $(tail -1 "$WORK/garbage.err")"
fi

# -- 3b. the digest disposition, which is `ACQ-06`'s half of this file --------
#
# ⛔ THE CONTROL COMES FIRST, BECAUSE EVERY REFUSAL BELOW PASSES EQUALLY OVER A
# RESOLVER THAT NEVER READ A DISPOSITION AT ALL. Section 1 already resolved all
# four adapters; this reads what each RECORDED, so a digest block that silently
# did nothing is a red row rather than four green ones.
while read -r target want; do
  [ -n "$target" ] || continue
  got=$(sed -n 's/^digest_source=//p' "$WORK/$target.rec")
  if [ "$got" = "$want" ]; then
    pass "digest    $target records digest_source=$got"
  else
    fail "digest    $target records digest_source=[$got], expected [$want]"
  fi
  #
  # ⭐ AND THE FOUR TARGETS REACH THREE DIFFERENT DISPOSITIONS, which is what makes
  # this a table rather than four copies of one case. Measured through `AGENTS.md`
  # rule 8's route on 2026-09-17 and recorded into the fixtures by
  # `project-listing`:
  #
  #   * `aria2-next` publishes a checksums document AND the listing states a
  #     digest, so two parties describe those bytes: `both`;
  #   * `qbittorrent` and `transmission` publish no checksums file and the listing
  #     states a digest: `source-listing`;
  #   * `aria2` publishes neither - its six assets carry `digest: null` - so it is
  #     the one target that is genuinely `unpublished`.
  #
  # ⛔ A table where every row expected the same value would pass over a resolver
  # that answered that value unconditionally.
done <<DISPOSITION_TABLE
aria2-next both
aria2 unpublished
qbittorrent source-listing
transmission source-listing
DISPOSITION_TABLE

# ⭐ AND THE DIGEST THE LISTING STATED IS THE ONE RECORDED, in the canonical form
# that carries its algorithm. ⛔ A bare hex run is what this project refuses
# everywhere, so a record spelling one would be a value no reader could tell from
# a different algorithm's.
recorded=$(sed -n 's/^asset_sha256=//p' "$WORK/transmission.rec")
listed=$(python3 -c "
import json,sys
d=json.load(open('$LISTINGS/transmission.json'))
for r in d:
    for a in r.get('assets', []):
        if a['name'] == 'transmission-4.1.3.tar.xz':
            print(a.get('digest','')); raise SystemExit
" 2>/dev/null)
if [ -n "$recorded" ] && [ "$recorded" = "$listed" ]; then
  pass "digest    the recorded asset_sha256 is the digest the listing stated"
elif [ -z "$listed" ]; then
  fail "digest    the fixture states no digest for the asset, so nothing was compared"
else
  fail "digest    recorded [$recorded], the listing states [$listed]"
fi

# ⛔ AND A TARGET WHOSE SOURCE STATES NONE RECORDS AN EMPTY FIELD RATHER THAN
# INVENTING ONE. ⚠ Without this the row above passes equally over a resolver that
# copied some other asset's digest into every record.
empty=$(sed -n 's/^asset_sha256=//p' "$WORK/aria2.rec")
if [ -z "$empty" ]; then
  pass "digest    a source that states no digest records an empty asset_sha256"
else
  fail "digest    aria2 records asset_sha256=[$empty] over a listing that states none"
fi

# ⭐ AND THE ONE TARGET WITH A DOCUMENT CARRIES THE DOCUMENT'S OWN NAME, which is
# what says the second `select-asset` call really ran over the same listing.
digest_asset=$(sed -n 's/^digest_asset=//p' "$WORK/aria2-next.rec")
if [ "$digest_asset" = "aria2-next-2.7.4-checksums.sha256" ]; then
  pass "digest    aria2-next selected its checksums asset out of the same listing"
else
  fail "digest    aria2-next selected digest_asset=[$digest_asset]"
fi

# ⛔ A DOCUMENT THAT DOES NOT NAME THE SELECTED ASSET IS REFUSED HERE, NOT AT THE
# INSTALL. `sha256sum -c --ignore-missing` over such a document exits 1 with `no
# file was verified`, so the install would refuse anyway - on a host whose route
# has already been cut, with a message about a checksum rather than about a
# vendor's document. ⚠ This is the case the generated document exists for: the
# name is planted and the refusal read.
sums_for "some-other-asset" >"$WORK/wrong-sums.txt"
_out="$WORK/wrongsums.out"
_err="$WORK/wrongsums.err"
sh "$RESOLVE" --adapter "$ADAPTERS/aria2-next.sh" --workdir "$WORK/wrongsums.work" \
  --record "$WORK/wrongsums.rec" --listing "$LISTINGS/aria2-next.json" \
  --checksums "$WORK/wrong-sums.txt" >"$_out" 2>"$_err"
rc=$?
if [ "$rc" = 1 ] && said wrongsums "does not name aria2-next-2.7.4-linux-x86_64"; then
  pass "digest    a checksums document naming another asset is refused"
else
  fail "digest    the wrong checksums document exited $rc: $(tail -1 "$_err")"
fi

# ⛔ AND THE NAME IS COMPARED RATHER THAN MATCHED, WHICH IS A SEPARATE BRANCH.
# A `.asc` beside an asset is the real shape: `foo.tar.gz` is a substring of
# `foo.tar.gz.asc`, so a document naming only the signature would satisfy a
# substring search and verify nothing. ⚠ Without this case the comparison above
# passes equally over a reader that searches for a substring.
sums_for "aria2-next-2.7.4-linux-x86_64.asc" >"$WORK/prefix-sums.txt"
_err="$WORK/prefixsums.err"
sh "$RESOLVE" --adapter "$ADAPTERS/aria2-next.sh" --workdir "$WORK/prefixsums.work" \
  --record "$WORK/prefixsums.rec" --listing "$LISTINGS/aria2-next.json" \
  --checksums "$WORK/prefix-sums.txt" >"$WORK/prefixsums.out" 2>"$_err"
rc=$?
if [ "$rc" = 1 ] && said prefixsums "does not name aria2-next-2.7.4-linux-x86_64"; then
  pass "digest    a document naming only the signature beside the asset is refused"
else
  fail "digest    the signature-only document exited $rc: $(tail -1 "$_err")"
fi

# ⛔ A RECORDED LISTING WITHOUT A RECORDED DOCUMENT IS could-not-run. Without
# this the resolver would fetch the vendor's checksums over a URL the fixture
# names, so a harness written to need no network would quietly need one.
_err="$WORK/nosums.err"
sh "$RESOLVE" --adapter "$ADAPTERS/aria2-next.sh" --workdir "$WORK/nosums.work" \
  --record "$WORK/nosums.rec" --listing "$LISTINGS/aria2-next.json" \
  >"$WORK/nosums.out" 2>"$_err"
rc=$?
if [ "$rc" = 2 ] && said nosums "--checksums must record the digest document too"; then
  pass "digest    a recorded listing with no recorded document is could-not-run"
else
  fail "digest    the unrecorded document exited $rc: $(tail -1 "$_err")"
fi

# ⚠ AND AN ADAPTER THAT DECLARES NEITHER DISPOSITION IS REFUSED, which is the
# whole reason the declaration is positive rather than defaulted: a target added
# without one would otherwise install bytes nothing identified.
if plant nodigest 'release_digest_unpublished=this release publishes six archives and no checksums asset' 'release_note=none'; then
  resolve nodigest "$WORK/nodigest.sh" "$LISTINGS/aria2.json"
  rc=$?
  if [ "$rc" = 2 ] && said nodigest "declares neither release_digest_asset nor release_digest_unpublished"; then
    pass "digest    an adapter declaring no disposition is could-not-run"
  else
    fail "digest    the undeclared disposition exited $rc: $(tail -1 "$WORK/nodigest.err")"
  fi
else
  fail "plant     the removed disposition did not apply"
fi

# ⚠ AND ONE THAT DECLARES BOTH, because two answers that could disagree are not
# for this file to choose between.
#
# ⛔ A STUB RATHER THAN A PLANT, AND THE FIRST ATTEMPT IS WHY. `replace_once`
# takes a LITERAL, so a replacement carrying an embedded newline and nested
# quotes emitted only its first line: the copy declared the asset alone, the run
# refused for a pattern that matched nothing, and the case read that as its own
# refusal firing. ⚠ A plant that did not apply is a third status, and this one
# announced itself only because the message named a different rule.
BOTH="$WORK/both-digest.sh"
cat >"$BOTH" <<'BOTH_ADAPTER'
#!/bin/sh
[ "${1:-}" = describe ] || exit 2
printf 'target=stub\n'
printf 'kind=stub\n'
printf 'release_repo=aria2/aria2\n'
printf 'release_tag_prefix=release-\n'
printf 'release_min_components=3\n'
printf 'release_max_components=3\n'
printf 'release_asset=aria2-{version}.tar.bz2\n'
printf 'release_digest_asset=aria2-{version}.sha256\n'
printf 'release_digest_unpublished=and also this\n'
BOTH_ADAPTER

resolve bothdigest "$BOTH" "$LISTINGS/aria2.json" aria2-1.37.0.tar.bz2
rc=$?
if [ "$rc" = 2 ] && said bothdigest "declares both release_digest_asset and release_digest_unpublished"; then
  pass "digest    an adapter declaring both dispositions is could-not-run"
else
  fail "digest    the double disposition exited $rc: $(tail -1 "$WORK/bothdigest.err")"
fi

# -- 4. the control the refusals need ----------------------------------------
#
# ⛔ WITHOUT THIS EVERY CASE ABOVE PASSES OVER A RESOLVER THAT REFUSES
# EVERYTHING. The clean adapter and the clean listing were already run in
# section 1; this asserts that the one that refused and the one that did not
# differ in exactly the planted line.
if diff "$ADAPTERS/aria2.sh" "$WORK/ambiguous.sh" >"$WORK/plant.diff" 2>&1; then
  fail "control   the planted adapter is identical to the shipped one"
elif [ "$(grep -c '^[<>]' "$WORK/plant.diff")" = "2" ]; then
  pass "control   the refusing adapter differs from the passing one on one line"
else
  fail "control   the plant changed $(grep -c '^[<>]' "$WORK/plant.diff") line(s), expected 2"
fi

store_probe_guards "$WORK/ambiguous.sh" 'release_repo=aria2/aria2' 'printf'

store_report check-release-route/1 cases "$JSON"
