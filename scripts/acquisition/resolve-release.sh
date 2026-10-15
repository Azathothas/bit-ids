#!/bin/sh
# resolve-release.sh - decide which artifact the `release` route will fetch, on
# a host that still has a route off itself.
#
# ⛔ THIS IS WHY EVERY `release` ROUTE IN THIS TREE HAS BEEN UNRUNNABLE. Every
# adapter refuses without `BIT_IDS_RELEASE_URL` "resolved before the
# route was cut", and nothing in the repository resolved one: `resolve-stable`
# orders versions and does not choose an artifact, so `capture-client.yml` passed
# `--route package` and only that. This is the missing half.
#
# ⛔ AND IT RUNS BEFORE THE ROUTE IS CUT, FOR THE SAME REASON THE INSTALL DOES.
# It reads a vendor's release listing over the network; a host with no default
# route cannot, and restoring egress to find out which file to download would be
# undoing the containment on the machine that exists to have it.
#
# -- ⭐ WHERE EACH DECISION LIVES ---------------------------------------------
#
# The adapter says WHICH repository, HOW that target spells versions and WHICH
# artifact of a release is the installable one, because it is already the only
# file in this repository that knows how a particular product is installed. It
# says so through `describe`, which is a pure read the callers already parse for
# `binary=`; a sixth subcommand would be a second door into the same answer.
#
# `fetch-releases.sh` retrieves and keeps the exact bytes. `resolve-stable`
# orders the versions. `select-asset` picks the artifact. Shell orchestrates and
# Rust parses, which is the line docs/architecture.md section 3 draws.
#
# ⚠ ONE RESPONSE ANSWERS THREE QUESTIONS. The version, the asset and the name of
# the vendor's checksums document all come out of the same body, so the digest
# this record carries covers all three decisions and nothing is resolved twice.
#
# ⛔ AND THE DIGEST DISPOSITION IS DECLARED BY THE ADAPTER, NEVER DEFAULTED.
# A target says either which asset carries the vendor's checksums, or in as many
# words that the vendor publishes none. An adapter that says neither is refused,
# because a route that quietly installed bytes nothing identified is exactly what
# a default would buy. `ACQ-06`.
#
# ⚠ `--listing` MAKES THE SOURCE A FILE, the way `assert-disposable --route-table`
# makes the routing table one, and for the same reason: a rule that can only be
# exercised by reaching a vendor is a rule the gate cannot run. ⛔ It PRINTS that
# it read a file, so a run over a recorded response is visible in a log rather
# than indistinguishable from one that asked the vendor.
# ⛔ `--checksums` IS ITS PAIR AND IT EXISTS FOR THE SAME REASON. A recorded
# listing names a real vendor URL, so resolving a digest from one would reach the
# network out of a harness written not to - which is a rule that can only be
# exercised by reaching a vendor, arriving one seam later. It prints what it read
# too.
#
# Usage:
#   sh scripts/acquisition/resolve-release.sh --adapter <path> --workdir <dir>
#                                             [--record <file>] [--listing <file>]
#                                             [--checksums <file>]
#
# Prints the artifact URL on stdout. Exit codes: 0 resolved, 1 the resolver or
# the selection refused, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

ADAPTER=""
WORKDIR=""
RECORD=""
LISTING_IN=""
CHECKSUMS_IN=""

usage() {
  awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
}

refuse() {
  printf 'resolve-release: %s\n' "$1" >&2
  exit 1
}

cannot() {
  printf 'resolve-release: %s\n' "$1" >&2
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
    --listing)
      [ $# -ge 2 ] || cannot "--listing takes a value"
      LISTING_IN="$2"
      shift
      ;;
    --checksums)
      [ $# -ge 2 ] || cannot "--checksums takes a value"
      CHECKSUMS_IN="$2"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      printf 'resolve-release: unknown argument: %s\n' "$1" >&2
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

# ⛔ CARGO_TARGET_DIR IS ASKED FOR RATHER THAN ASSUMED AWAY. A great many Rust
# developers export it, and a harness that composed `target/` looked in a
# directory nothing had built into - which is how a whole tier of provers once
# answered "could not run" at once and read as a green gate.
BIN="${CARGO_TARGET_DIR:-$ROOT/target}/debug/examples"
for _tool in resolve-stable select-asset; do
  [ -x "$BIN/$_tool" ] ||
    cannot "$BIN/$_tool is not built; run: cargo build -p bit-ids --locked --example $_tool"
done

mkdir -p "$WORKDIR" || cannot "cannot create $WORKDIR"
WORKDIR=$(CDPATH='' cd -- "$WORKDIR" && pwd)

# ⛔ EVERY DECLARATION IS ASKED FOR AS A FIELD, and a missing one is could-not-run
# rather than an empty string that composes into a nonsense URL. An adapter with
# no release route simply declares none, which is a fact about that target: the
# caller learns this target has no second route rather than watching a fetch of
# `https:///releases` fail with something that names nothing.
DESCRIBE="$WORKDIR/describe.txt"
# ⛔ BOUNDED, BECAUSE THIS CALL HAD NO LIMIT AT ALL UNTIL 2026-09-16 and it runs
# in *Resolve the release artifact* - a step whose job is to answer before the
# host's route is cut. An adapter that never returned here wedged the job with
# nothing to read, which is the same way `capture-client` runs 21, 22 and 23 were
# cancelled with no log and no artifact. ⚠ The redirection was already right: the
# adapter gets a FILE, so nothing it spawns can hold this shell's pipe.
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
REPO=$(field release_repo)
[ -n "$REPO" ] ||
  cannot "$TARGET declares no release_repo, so it has no release route to resolve"
PREFIX=$(field release_tag_prefix)
[ -n "$PREFIX" ] || cannot "$TARGET declares release_repo and no release_tag_prefix"
MINC=$(field release_min_components)
[ -n "$MINC" ] || cannot "$TARGET declares no release_min_components"
MAXC=$(field release_max_components)
[ -n "$MAXC" ] || cannot "$TARGET declares no release_max_components"
PATTERN=$(field release_asset)
[ -n "$PATTERN" ] || cannot "$TARGET declares no release_asset pattern"

# ⛔ THE DIGEST DISPOSITION IS DECLARED, NEVER DEFAULTED. An adapter says either
# which asset carries the vendor's checksums or, in as many words, that the
# vendor publishes none - and an adapter that says neither is refused here rather
# than served a route that quietly installs bytes nothing identified.
# ⚠ Two declarations is a target whose two answers could disagree, and choosing
# between them here would be this file deciding which evidence counts.
DIGEST_PATTERN=$(field release_digest_asset)
DIGEST_NONE=$(field release_digest_unpublished)
if [ -n "$DIGEST_PATTERN" ] && [ -n "$DIGEST_NONE" ]; then
  cannot "$TARGET declares both release_digest_asset and release_digest_unpublished"
fi
if [ -z "$DIGEST_PATTERN" ] && [ -z "$DIGEST_NONE" ]; then
  cannot "$TARGET declares neither release_digest_asset nor release_digest_unpublished"
fi

# ⚠ THE COMPONENT COUNTS ARE CHECKED AS NUMBERS HERE, because everything after
# this hands them to a resolver that would refuse them anyway - with a message
# about a tag scheme, over a value this file could have named.
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

LISTING="$WORKDIR/releases.json"
if [ -n "$LISTING_IN" ]; then
  # ⛔ SAID OUT LOUD. A run that resolved against a recorded response and one
  # that asked the vendor are otherwise the same lines in a log, and the first
  # establishes nothing about what the vendor publishes today.
  [ -f "$LISTING_IN" ] || cannot "the listing $LISTING_IN is not present"
  cp "$LISTING_IN" "$LISTING" || cannot "cannot copy $LISTING_IN into the workdir"
  # ⚠ The resolver reads the retrieval instant from a sidecar the fetch writes,
  # so a recorded response needs one too. It is stamped now, because now is when
  # this run read those bytes; the response's own age is in the fixture.
  date -u +%Y-%m-%dT%H:%M:%SZ >"$LISTING.fetched-at" ||
    cannot "cannot stamp the retrieval instant"
  URL="https://recorded.invalid/$LISTING_IN"
  printf 'resolve-release: read the release listing from %s, not from %s\n' \
    "$LISTING_IN" "$REPO" >&2
else
  URL=$(sh "$ROOT/scripts/acquisition/fetch-releases.sh" "$REPO" "$LISTING")
  FETCH_RC=$?
  case "$FETCH_RC" in
    0) : ;;
    1) refuse "the release listing for $REPO could not be retrieved" ;;
    *) cannot "fetch-releases could not run (exit $FETCH_RC)" ;;
  esac
  [ -n "$URL" ] || cannot "fetch-releases printed no source URL"
fi

# ⛔ THE RESOLUTION DOCUMENT IS KEPT, NOT ONLY THE ANSWER. It carries every
# candidate with the verdict it got and a digest of the bytes that were read, so
# a selection is re-derivable rather than asserted. `select-asset` then reads the
# tag off the candidate that won, which is what keeps the version this resolves
# and the release it takes an asset from from being two decisions.
RESOLUTION="$WORKDIR/resolution.json"
"$BIN/resolve-stable" "$TARGET" "$PREFIX" "$MINC" "$MAXC" "$FLOOR" \
  github-releases "$URL" "$LISTING" \
  >"$RESOLUTION" 2>"$WORKDIR/resolve.err"
RESOLVE_RC=$?
case "$RESOLVE_RC" in
  0) : ;;
  1)
    sed 's/^/          /' "$WORKDIR/resolve.err" >&2
    refuse "the resolver failed closed for $TARGET; $RESOLUTION says why"
    ;;
  *)
    sed 's/^/          /' "$WORKDIR/resolve.err" >&2
    cannot "resolve-stable could not run (exit $RESOLVE_RC)"
    ;;
esac

SELECTION="$WORKDIR/selection.txt"
"$BIN/select-asset" "$RESOLUTION" "$LISTING" "$PATTERN" \
  >"$SELECTION" 2>"$WORKDIR/select.err"
SELECT_RC=$?
# ⚠ THE OFFER IS PRINTED WHATEVER HAPPENS. On a refusal it is the diagnosis, and
# on success it is the evidence that exactly one asset matched rather than the
# first of several.
sed 's/^/          /' "$WORKDIR/select.err" >&2
case "$SELECT_RC" in
  0) : ;;
  1) refuse "no single asset of $TARGET's release matches [$PATTERN]" ;;
  *) cannot "select-asset could not run (exit $SELECT_RC)" ;;
esac

selected() {
  awk -F= -v k="$1" '$1 == k { sub(/^[^=]*=/, ""); print; exit }' "$SELECTION"
}
VERSION=$(selected version)
TAG=$(selected tag)
ASSET=$(selected asset)
ASSET_URL=$(selected url)
SIZE=$(selected size)
# ⭐ WHAT THE SOURCE SAYS THE BYTES DIGEST TO, out of the SAME response that
# chose the version and the artifact. `ACQ-06`. ⚠ Empty means the source stated
# none, which is a fact about that source: measured through `AGENTS.md` rule 8's
# route on 2026-09-17, three of this project's four targets carry one on every
# asset and `aria2/aria2` carries none on any of its six.
ASSET_SHA256=$(selected digest)
[ -n "$ASSET_URL" ] || cannot "select-asset printed no url"

# -- the vendor's digest, out of the same response -----------------------------
#
# ⭐ ONE RETRIEVAL ANSWERS THREE QUESTIONS. The listing that chose the version
# chose the artifact and now names the checksums document beside it, so the
# digest of that listing already covers all three decisions and none of them was
# made from a URL somebody composed. ⛔ Composing the checksums URL out of the
# artifact's would be guessing at a vendor's naming, which is the shape
# `select-asset` exists to refuse.
#
# ⚠ AND IT IS FETCHED HERE RATHER THAN AT INSTALL TIME, because this step runs
# while the host still has a route off itself. `capture-client.yml`'s step order
# is the containment: everything a run needs is fetched before the route is cut.
DIGEST_ASSET=""
DIGEST_URL=""
DIGEST_DOC=""
if [ -n "$DIGEST_PATTERN" ]; then
  DIGEST_SELECTION="$WORKDIR/digest-selection.txt"
  "$BIN/select-asset" "$RESOLUTION" "$LISTING" "$DIGEST_PATTERN" \
    >"$DIGEST_SELECTION" 2>"$WORKDIR/digest-select.err"
  DIGEST_RC=$?
  sed 's/^/          /' "$WORKDIR/digest-select.err" >&2
  case "$DIGEST_RC" in
    0) : ;;
    1) refuse "$TARGET declares a checksums asset and no single asset matches [$DIGEST_PATTERN]" ;;
    *) cannot "select-asset could not run for the checksums asset (exit $DIGEST_RC)" ;;
  esac
  DIGEST_ASSET=$(awk -F= '$1 == "asset" { sub(/^[^=]*=/, ""); print; exit }' "$DIGEST_SELECTION")
  DIGEST_URL=$(awk -F= '$1 == "url" { sub(/^[^=]*=/, ""); print; exit }' "$DIGEST_SELECTION")
  [ -n "$DIGEST_URL" ] || cannot "select-asset printed no url for the checksums asset"

  # ⛔ A RECORDED LISTING MAY NOT RESOLVE A DIGEST OVER THE NETWORK, AND THAT IS
  # REFUSED HERE RATHER THAN LEFT TO A CALLER. `--listing` exists so a harness
  # can exercise this file without a vendor; a listing fixture names real vendor
  # URLs, so the digest fetch below would reach one anyway and the gate would
  # depend on an endpoint being up. ⚠ A harness that simply forgot `--checksums`
  # would get that silently, which is the shape every other guard here exists to
  # refuse. So the two flags travel together or this stops.
  if [ -n "$LISTING_IN" ] && [ -z "$CHECKSUMS_IN" ]; then
    cannot "--listing reads a recorded response, so --checksums must record the digest document too"
  fi

  DIGEST_DOC="$WORKDIR/checksums.txt"
  if [ -n "$CHECKSUMS_IN" ]; then
    # ⛔ SAID OUT LOUD, THE WAY `--listing` ALREADY IS. A run that verified
    # against a recorded document and one that asked the vendor are otherwise
    # the same lines in a log, and only the second says what the vendor
    # publishes today.
    [ -f "$CHECKSUMS_IN" ] || cannot "the checksums document $CHECKSUMS_IN is not present"
    cp "$CHECKSUMS_IN" "$DIGEST_DOC" || cannot "cannot copy $CHECKSUMS_IN into the workdir"
    printf 'resolve-release: read the checksums document from %s, not from %s\n' \
      "$CHECKSUMS_IN" "$DIGEST_URL" >&2
  else
    # ⛔ THE FETCH IS THE INSTALLER'S, WHICH IS WHERE THE BOUND AND THE DIGEST
    # RULE LIVE. A second `curl` here would be a second copy of a rule four
    # adapters already broke once. ⚠ A checksums document is a document rather
    # than a build: it publishes no digest of itself, which is what
    # `--digest-unpublished` says, and nothing executes it.
    ROOTLESS="$ROOT/scripts/acquisition/install-rootless.sh"
    [ -f "$ROOTLESS" ] || cannot "$ROOTLESS is not present"
    sh "$ROOTLESS" --fetch --url "$DIGEST_URL" --into "$DIGEST_DOC" \
      --digest-unpublished "a checksums document publishes no digest of itself" \
      </dev/null >"$WORKDIR/digest-fetch.log" 2>&1
    DIGEST_FETCH_RC=$?
    case "$DIGEST_FETCH_RC" in
      0) : ;;
      1)
        sed 's/^/          /' "$WORKDIR/digest-fetch.log" >&2
        refuse "the checksums document $DIGEST_ASSET could not be retrieved"
        ;;
      *)
        sed 's/^/          /' "$WORKDIR/digest-fetch.log" >&2
        cannot "install-rootless could not run for the checksums asset (exit $DIGEST_FETCH_RC)"
        ;;
    esac
  fi

  # ⛔ AND THE DOCUMENT IS ASKED WHETHER IT COVERS THE ASSET THAT WAS SELECTED.
  # `sha256sum -c --ignore-missing` over a document naming none of the files
  # present exits 1 with `no file was verified`, so the install would refuse
  # anyway - but it would refuse at install time with a message about a checksum,
  # on a host whose route has already been cut. This refuses here, naming both.
  #
  # ⛔ THE NAME IS COMPARED, NEVER MATCHED. An asset name carries dots - every
  # version does - and a `.` in a pattern matches any byte, which is the defect
  # `install-client.sh` records about `reported_version=1.37.0`. It is also a
  # prefix question: `foo.tar.gz` is a substring of `foo.tar.gz.asc`, so a
  # substring search would report a document covering the signature as covering
  # the build. So the digest prefix is stripped and the remainder compared with
  # `==`, which is neither a regex nor a substring.
  awk -v want="$ASSET" '
    { line = $0
      sub(/^[0-9a-fA-F]+[ \t]+\*?/, "", line)
      if (line == want) found = 1 }
    END { exit found ? 0 : 1 }' "$DIGEST_DOC" ||
    refuse "$DIGEST_ASSET does not name $ASSET, so it cannot verify the artifact this route installs"
fi

# -- which of the four dispositions this resolution reached --------------------
#
# ⛔ TWO INDEPENDENT STATEMENTS ABOUT ONE ARTIFACT ARE NOT ONE. The vendor's
# checksums document is authored by the vendor; the listing's per-asset digest is
# authored by the party SERVING the bytes. They answer different questions - what
# the vendor meant to publish, and whether anything changed between reading the
# index and fetching - so a record naming only the stronger would tell a reader
# less than happened. ⭐ `both` is its own value for that reason.
if [ -n "$DIGEST_DOC" ] && [ -n "$ASSET_SHA256" ]; then
  DIGEST_SOURCE=both
elif [ -n "$DIGEST_DOC" ]; then
  DIGEST_SOURCE=vendor-document
elif [ -n "$ASSET_SHA256" ]; then
  DIGEST_SOURCE=source-listing
else
  DIGEST_SOURCE=unpublished
fi

# ⛔ AND AN ADAPTER THAT DECLARED NO DOCUMENT WHILE THE SOURCE PUBLISHES A DIGEST
# IS NOT A CONTRADICTION. `release_digest_unpublished` is about the VENDOR's
# checksums file, which is what an adapter can know by reading a release; the
# listing digest is the host's and appears without the vendor doing anything.
# ⚠ Stated here because the two sound like the same sentence and are not: of the
# four targets, three declare no document and two of those still verify.

FINISHED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# ⚠ Key=value, the shape install-client's own record uses. ⛔ It carries the
# digest of the listing as well as the URL, because the answer is only
# re-derivable from the bytes the decision was made from: a vendor that edits a
# release afterwards leaves a record that still says what was read.
if [ -n "$RECORD" ]; then
  {
    printf 'bit-ids/release-resolution/1\n'
    printf 'target=%s\n' "$TARGET"
    printf 'repository=%s\n' "$REPO"
    printf 'source_url=%s\n' "$URL"
    printf 'listing_sha256=%s\n' "$(sha256sum "$LISTING" | cut -d' ' -f1)"
    printf 'selected_version=%s\n' "$VERSION"
    printf 'selected_tag=%s\n' "$TAG"
    printf 'asset_pattern=%s\n' "$PATTERN"
    printf 'asset=%s\n' "$ASSET"
    printf 'asset_url=%s\n' "$ASSET_URL"
    printf 'asset_declared_size=%s\n' "$SIZE"
    # ⛔ WHICH OF THE TWO DISPOSITIONS THIS TARGET HAS, AS A FIELD. A reader
    # taking an absent `digest_asset` for "the vendor publishes none" would be
    # reading a truncated record as a measurement, which is why every field here
    # is printed on every path and `digest_source` is never empty.
    printf 'digest_source=%s\n' "$DIGEST_SOURCE"
    printf 'asset_sha256=%s\n' "$ASSET_SHA256"
    printf 'digest_asset=%s\n' "$DIGEST_ASSET"
    printf 'digest_asset_url=%s\n' "$DIGEST_URL"
    printf 'digest_document=%s\n' "$DIGEST_DOC"
    printf 'digest_unpublished=%s\n' "$DIGEST_NONE"
    printf 'adapter=%s\n' "$ADAPTER"
    printf 'adapter_sha256=%s\n' "$(sha256sum "$ADAPTER" | cut -d' ' -f1)"
    printf 'started_at=%s\n' "$STARTED"
    printf 'finished_at=%s\n' "$FINISHED"
  } >"$RECORD" || cannot "the resolution record could not be written to $RECORD"

  # ⛔ EVERY FIELD IS ASKED FOR AS A FIELD, the way install-client asks for its
  # own. A record truncated part-way through the block would carry a correct
  # target and no URL at all, and a caller reading an absent one as empty is the
  # failure this whole file exists to remove.
  for _field in target repository source_url listing_sha256 selected_version \
    selected_tag asset_pattern asset asset_url digest_source; do
    grep -q "^$_field=" "$RECORD" ||
      cannot "the resolution record was written without $_field"
  done
fi

printf '%s\n' "$ASSET_URL"
printf 'resolve-release: %s %s -> %s (%s bytes declared)\n' \
  "$TARGET" "$VERSION" "$ASSET" "$SIZE" >&2
exit 0
