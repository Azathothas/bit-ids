#!/bin/sh
# resolve-source.sh - decide which tag the `source` route will build, from the
# repository's own refs rather than from its release listing.
#
# ⛔ THIS EXISTS BECAUSE TWO LANES OF ONE DISPATCH WERE ONE ROUTE. Measured on
# 2026-09-09 by assembling `capture-client` run 14: both its lanes read the same
# release listing - their `resolution.txt` differ only in their timestamps, with
# one `source_url`, one `listing_sha256` and one `asset_url` - and `E-ACQ-07`
# calls two routes sharing a resolver one route. `capture-client.yml` did that
# deliberately, so that both lanes could not land on different versions.
#
# ⭐ ABSOLUTE 4 ALREADY ANSWERS THAT WORRY, AND ANSWERS IT BETTER. Version
# equality is checked AFTER installation, on what each build reported when asked
# - never trusted from a filename, a tag or a package index. So two independent
# resolutions that landed on two versions is a vendor that moved between two
# reads, and catching it is the correct outcome. Forcing one resolution
# manufactures the agreement instead of measuring it, which is exactly what
# `E-ACQ-07` exists to refuse.
#
# -- ⭐ WHAT MAKES THIS A DIFFERENT SOURCE ------------------------------------
#
# `git ls-remote --tags --refs` asks the repository what tags it has. A releases
# listing asks an API what release OBJECTS were published, which is a different
# and much smaller population: measured on 2026-09-05, `qbittorrent`'s releases
# endpoint answered with four entries while its tags endpoint answered with at
# least a hundred. Two indexes that can disagree about the newest version are
# two resolvers, which is what `E-ACQ-07` compares.
#
# ⛔ THE ORDERING IS THE SAME AND THAT IS DELIBERATE. `VersionScheme::components`
# is this project's ONE version ordering and every caller uses it; a second
# implementation here would be a second answer to which version is newest.
# `E-ACQ-07` compares what DECIDED - the index - not the arithmetic.
#
# ⚠ WHAT A REFS SOURCE CANNOT DO IS FLAG A PRERELEASE. A releases listing
# carries `prerelease` and `draft`; refs carry a name. What survives is the
# version text, and `resolution.rs` records the limit beside the reader.
#
# ⚠ `--refs <file>` MAKES THE SOURCE A FILE, the way `resolve-release --listing`
# does, and for the same reason: a rule that can only be exercised by reaching a
# vendor is a rule the gate cannot run. ⛔ It PRINTS that it read a file.
#
# Usage:
#   sh scripts/acquisition/resolve-source.sh --adapter <path> --workdir <dir>
#                                            [--record <file>] [--refs <file>]
#
# Prints the selected tag on stdout. Exit codes: 0 resolved, 1 the resolver
# refused, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

ADAPTER=""
WORKDIR=""
RECORD=""
REFS_IN=""

usage() {
  awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
}

refuse() {
  printf 'resolve-source: %s\n' "$1" >&2
  exit 1
}

cannot() {
  printf 'resolve-source: %s\n' "$1" >&2
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --adapter)
      [ $# -ge 2 ] || cannot "--adapter takes a value"
      ADAPTER="$2"
      shift
      ;;
    --workdir)
      [ $# -ge 2 ] || cannot "--workdir takes a value"
      WORKDIR="$2"
      shift
      ;;
    --record)
      [ $# -ge 2 ] || cannot "--record takes a value"
      RECORD="$2"
      shift
      ;;
    --refs)
      [ $# -ge 2 ] || cannot "--refs takes a value"
      REFS_IN="$2"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      printf 'resolve-source: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

[ -n "$ADAPTER" ] || cannot "--adapter is required"
[ -n "$WORKDIR" ] || cannot "--workdir is required"
[ -f "$ADAPTER" ] || cannot "the adapter $ADAPTER is not present"

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# ⛔ CARGO_TARGET_DIR IS ASKED FOR RATHER THAN ASSUMED AWAY, for the reason
# `resolve-release.sh` gives at the same line: a composed `target/` once put a
# whole tier of provers into could-not-run and read as a green gate.
BIN="${CARGO_TARGET_DIR:-$ROOT/target}/debug/examples"
[ -x "$BIN/resolve-stable" ] ||
  cannot "$BIN/resolve-stable is not built; run: cargo build -p bit-ids --locked --example resolve-stable"

mkdir -p "$WORKDIR" || cannot "cannot create $WORKDIR"
WORKDIR=$(CDPATH='' cd -- "$WORKDIR" && pwd)

DESCRIBE="$WORKDIR/describe.txt"
# ⛔ BOUNDED, for the reason its release-route sibling gives: this call carried no
# limit until 2026-09-16 while the `git ls-remote` fifty lines below had carried
# `timeout 120` all along. ⚠ That is the convention holding on one of two paths
# through one file, which is the shape `check-adapters` exists for and does not
# reach - its scope is a fetch INSIDE an adapter, not a call site that invokes
# one.
DESCRIBE_SECONDS=${BIT_IDS_DESCRIBE_TIMEOUT:-30}
DESCRIBE_KILL_AFTER=${BIT_IDS_KILL_AFTER:-20}
command -v timeout >/dev/null 2>&1 || cannot "timeout is not on this host"
timeout -k "$DESCRIBE_KILL_AFTER" "$DESCRIBE_SECONDS" sh "$ADAPTER" describe \
  </dev/null >"$DESCRIBE" 2>"$WORKDIR/describe.err" ||
  cannot "the adapter could not describe itself"
field() {
  awk -F= -v k="$1" '$1 == k { sub(/^[^=]*=/, ""); print; exit }' "$DESCRIBE"
}

TARGET=$(field target)
[ -n "$TARGET" ] || cannot "the adapter named no target"
# ⚠ THE SAME REPOSITORY THE RELEASE ROUTE NAMES, and that is the point rather
# than an oversight: the two routes differ in which INDEX of that repository
# decided the version, not in whose code it is. `CLIENT-14` records that this
# pair is weaker than a distribution index against a vendor release, because
# whoever controls the repository controls both.
REPO=$(field release_repo)
[ -n "$REPO" ] ||
  cannot "$TARGET declares no release_repo, so there is no repository to read refs from"
PREFIX=$(field release_tag_prefix)
[ -n "$PREFIX" ] || cannot "$TARGET declares release_repo and no release_tag_prefix"
MINC=$(field release_min_components)
[ -n "$MINC" ] || cannot "$TARGET declares no release_min_components"
MAXC=$(field release_max_components)
[ -n "$MAXC" ] || cannot "$TARGET declares no release_max_components"

for _n in "$MINC" "$MAXC"; do
  case "$_n" in
    '' | *[!0-9]*) cannot "$TARGET declares a non-numeric component count [$_n]" ;;
  esac
done

# ⛔ THE FLOOR IS DECLARED, NEVER DEFAULTED, `-` for none. It releases tags a
# scheme cannot order, so a floor this file invented would be a selection this
# file made. `CLIENT-06`.
FLOOR=$(field release_floor_major)
case "$FLOOR" in
  '') cannot "$TARGET declares no release_floor_major; - says it has none" ;;
  -) : ;;
  *[!0-9]*) cannot "$TARGET declares a non-numeric release_floor_major [$FLOOR]" ;;
esac

STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

REFS="$WORKDIR/refs.txt"
if [ -n "$REFS_IN" ]; then
  [ -f "$REFS_IN" ] || cannot "the refs listing $REFS_IN is not present"
  cp "$REFS_IN" "$REFS" || cannot "cannot copy $REFS_IN into the workdir"
  date -u +%Y-%m-%dT%H:%M:%SZ >"$REFS.fetched-at" ||
    cannot "cannot stamp the retrieval instant"
  URL="https://recorded.invalid/$REFS_IN"
  printf 'resolve-source: read the refs from %s, not from %s\n' "$REFS_IN" "$REPO" >&2
else
  command -v git >/dev/null 2>&1 || cannot "git is not on this host"
  URL="https://github.com/$REPO.git"
  # ⛔ --refs, SO AN ANNOTATED TAG IS OFFERED ONCE. Without it `ls-remote` emits
  # a second `^{}` line per annotated tag carrying the commit it points at, and
  # the reader refuses one rather than skipping it - a silent skip would hide a
  # caller that forgot the flag. ⚠ So this file yields TAGS and not commits;
  # `v2.7.5` is an annotated tag whose own object is not the commit that was
  # built, which the adapter's `git rev-parse HEAD` after the clone is what
  # answers.
  #
  # ⚠ MEASURED DIRECTLY on 2026-09-09 rather than through a proxy: `git
  # ls-remote` is git-over-HTTPS and not a REST read, so `AGENTS.md` rule 8's
  # route does not apply, and the direct call answered.
  if ! timeout 120 git ls-remote --tags --refs "$URL" >"$REFS" 2>"$WORKDIR/ls-remote.err"; then
    refuse "the refs for $REPO could not be retrieved"
  fi
  # ⭐ The instant is written by the thing that fetched, never inferred from the
  # file's mtime: an mtime survives a copy, an archive restore and a checkout,
  # so a resolution built from one would carry a retrieval instant that is not
  # one. A door sweep found `ACQ-02` resting on exactly that.
  date -u +%Y-%m-%dT%H:%M:%SZ >"$REFS.fetched-at" ||
    cannot "cannot stamp the retrieval instant"
fi

[ -s "$REFS" ] || refuse "$REPO offered no tags at all"

RESOLUTION="$WORKDIR/resolution.json"
"$BIN/resolve-stable" "$TARGET" "$PREFIX" "$MINC" "$MAXC" "$FLOOR" \
  git-refs "$URL" "$REFS" \
  >"$RESOLUTION" 2>"$WORKDIR/resolve.err"
RESOLVE_RC=$?
# ⛔ THE READER'S OWN WORDS ARE PRINTED ON EVERY NON-ZERO STATUS, not only on a
# refusal. The first version printed them for exit 1 and swallowed them for
# exit 2, so a listing this project could not read reported "could not run" with
# the sentence naming the bad line sitting in a workdir file nobody opens - which
# is exactly the defect `CLIENT-06` records in `install-client`, where a caller
# discarded an adapter's stderr and then reported its absence. Found by a harness
# case asserting the reason rather than only the code.
case "$RESOLVE_RC" in
  0) : ;;
  1)
    sed 's/^/          /' "$WORKDIR/resolve.err" >&2
    refuse "the resolver failed closed over $REPO's tags"
    ;;
  *)
    sed 's/^/          /' "$WORKDIR/resolve.err" >&2
    cannot "resolve-stable could not run (exit $RESOLVE_RC)"
    ;;
esac

# ⛔ THE TAG IS READ OUT OF THE RESOLUTION, NOT RE-DERIVED. Composing
# "$PREFIX$VERSION" here would be a second answer to which tag won, and it would
# be wrong for any target whose tag is not the prefix and the version joined.
SELECTED=$(awk -F'"' '/"selected"/ { print $4; exit }' "$RESOLUTION")
[ -n "$SELECTED" ] || cannot "the resolution names no selected version"

# ⛔ THE TAG OF THE ENTRY THAT WON, NOT THE NEXT TAG AFTER THE WORD `selected`.
# ⚠ The first version of this scanned forward from the verdict and took the tag
# it then met - and `tag` comes BEFORE `verdict` inside an entry, so it printed
# the FOLLOWING candidate's tag: `v2.7.4` over a resolution that selected 2.7.5.
# Found by running it against a real resolution document before it shipped. It
# is the same reader defect `CI-09` records in `check-project`'s first artifact
# rule, which scanned forward from `uses:` and took the first `name:` it met.
#
# ⭐ So the tag is remembered per entry and printed when THAT entry's verdict is
# the selection, with the entry boundary resetting it rather than a distance
# being assumed in either direction.
TAG=$(awk -F'"' '
  /"candidate":/ { tag = "" }
  /"tag":/ && tag == "" { tag = $4 }
  /"verdict": "selected"/ && tag != "" { print tag; exit }
' "$RESOLUTION")
[ -n "$TAG" ] || cannot "the resolution names no tag for the selected version"

DIGEST=$(sha256sum "$REFS" | cut -d' ' -f1)
FINISHED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

if [ -n "$RECORD" ]; then
  {
    printf 'bit-ids/source-resolution/1\n'
    printf 'target=%s\n' "$TARGET"
    printf 'repository=%s\n' "$REPO"
    # ⛔ THE FIELD `assemble-capture` SLUGIFIES INTO A RESOLVER IDENTITY, so it
    # has to differ from the release route's. A releases listing answers at
    # `.../repos/<owner>/<name>/releases` and this answers at
    # `https://github.com/<owner>/<name>.git`, which slugify to two names.
    printf 'source_url=%s\n' "$URL"
    printf 'refs_sha256=%s\n' "$DIGEST"
    printf 'selected_version=%s\n' "$SELECTED"
    printf 'selected_tag=%s\n' "$TAG"
    printf 'adapter=%s\n' "$ADAPTER"
    printf 'adapter_sha256=%s\n' "$(sha256sum "$ADAPTER" | cut -d' ' -f1)"
    printf 'started_at=%s\n' "$STARTED"
    printf 'finished_at=%s\n' "$FINISHED"
  } >"$RECORD" || cannot "the resolution record could not be written to $RECORD"

  for _field in target repository source_url refs_sha256 selected_version selected_tag; do
    grep -q "^$_field=" "$RECORD" ||
      cannot "the resolution record was written without $_field"
  done
fi

printf '%s\n' "$TAG"
