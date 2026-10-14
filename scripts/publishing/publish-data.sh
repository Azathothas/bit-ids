#!/bin/sh
# publish-data.sh - append an assembled bundle to the data branch, and read the
# branch back before saying it happened.
#
# PUB-02's problem is a publisher that force-pushes, drops prior records, or
# leaves partial output behind. ⛔ Each of those is unrecoverable in a way this
# project cares about specifically: the record that goes is not anywhere else.
#
# -- ⛔ THE APPEND RULE IS CHECKED BEFORE THE PUSH, NOT AFTER --------------
#
# A branch protection setting refuses a force-push and says nothing at all about
# a commit that deletes a file, which is exactly the shape a latest-only
# regeneration takes. So the published branch is fetched, checked out, and
# compared against the bundle with check-store before anything is committed. A
# refusal there ends the run with nothing pushed.
#
# -- ⛔ NO FORCE, AND THAT IS ASSERTED RATHER THAN OMITTED -----------------
#
# git refuses a non-fast-forward push by default, so the guard is that nothing
# re-enables it: no --force, no --force-with-lease, and no leading + on the
# refspec. The refspec is built here and checked for that character before it is
# used, because a caller-supplied one is the door a flag rule does not cover.
#
# -- ⛔ A PUSH IS NOT COMPLETION ------------------------------------------
#
# The branch is fetched again afterwards, checked out into a third directory,
# and compared three ways: that it is byte-for-byte the bundle that was pushed,
# that it still appends to what was there, and that every digest in its own
# SHA256SUMS matches the bytes that came back.
#
# ⛔ THE FIRST OF THOSE THREE WAS MISSING AND A PLANTED REMOTE FOUND IT. A
# post-receive hook that accepted the push and then moved the ref back left a
# branch that still appended to the prior tree perfectly, because it WAS the
# prior tree. "What came back appends to what was there" and "what came back is
# what I pushed" are two facts, and only the second one notices a remote that
# discarded the push.
#
# Usage:
#   sh scripts/publishing/publish-data.sh --bundle DIR --remote URL [--branch B]
#   sh scripts/publishing/publish-data.sh ... --dry-run
#
# --dry-run does everything except the push and the read-back, which is what a
# pull request lane should run.
#
# Exit codes: 0 appended and read back, 1 refused, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

BUNDLE=""
REMOTE=""
BRANCH="data"
MESSAGE=""
DRY_RUN=0

while [ $# -gt 0 ]; do
  case "$1" in
    --bundle)
      shift
      [ $# -gt 0 ] || {
        printf 'publish-data: --bundle needs a directory\n' >&2
        exit 2
      }
      BUNDLE="$1"
      ;;
    --remote)
      shift
      [ $# -gt 0 ] || {
        printf 'publish-data: --remote needs a URL\n' >&2
        exit 2
      }
      REMOTE="$1"
      ;;
    --branch)
      shift
      [ $# -gt 0 ] || {
        printf 'publish-data: --branch needs a name\n' >&2
        exit 2
      }
      BRANCH="$1"
      ;;
    --message)
      shift
      [ $# -gt 0 ] || {
        printf 'publish-data: --message needs text\n' >&2
        exit 2
      }
      MESSAGE="$1"
      ;;
    --dry-run) DRY_RUN=1 ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    *)
      printf 'publish-data: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

if [ -z "$BUNDLE" ] || [ -z "$REMOTE" ]; then
  printf 'publish-data: --bundle and --remote are required\n' >&2
  exit 2
fi
[ -d "$BUNDLE" ] || {
  printf 'publish-data: %s is not a directory\n' "$BUNDLE" >&2
  exit 2
}

# ⛔ Resolved from this script's own location, never from the working directory.
HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)
BUNDLE=$(CDPATH='' cd -- "$BUNDLE" && pwd)

command -v git >/dev/null 2>&1 || {
  printf 'publish-data: git not found\n' >&2
  exit 2
}
command -v sha256sum >/dev/null 2>&1 || {
  printf 'publish-data: sha256sum not found\n' >&2
  exit 2
}
command -v cargo >/dev/null 2>&1 || {
  printf 'publish-data: cargo not found\n' >&2
  exit 2
}

# ⛔ THE BRANCH NAME IS THE REFSPEC'S SECOND HALF, so a name carrying a + would
# smuggle a force push past every flag rule below. It is checked here, once,
# before the refspec exists.
case "$BRANCH" in
  *+* | *:* | -*)
    printf 'publish-data: refusing branch name %s\n' "$BRANCH" >&2
    exit 2
    ;;
esac

# ⛔ CARGO_TARGET_DIR IS ASKED FOR HERE TOO, AND THAT IS THE SECOND DOOR. The
# same assumption lived in store-lib.sh's store_build, and fixing one left this
# one composing root/target while cargo obeyed the environment: the build
# succeeded, the binary was elsewhere, and the publisher exited 2 saying its own
# append checker was not executable. It fails closed, which is the right
# direction, and it is still a publisher that cannot run on a machine whose only
# oddity is a variable many Rust developers set.
CHECKER="${CARGO_TARGET_DIR:-$ROOT/target}/debug/examples/check-store"
if ! cargo build --manifest-path "$ROOT/Cargo.toml" -p bit-ids --locked \
  --example check-store >/dev/null 2>&1; then
  printf 'publish-data: cannot build the append checker\n' >&2
  exit 2
fi
[ -x "$CHECKER" ] || {
  printf 'publish-data: %s is not executable after a successful build\n' "$CHECKER" >&2
  exit 2
}

WORK="${TMPDIR:-/tmp}/.publishdata.$$"
mkdir -p "$WORK/prior" "$WORK/after" || {
  printf 'publish-data: cannot write to %s\n' "$WORK" >&2
  exit 2
}
trap 'rm -rf "$WORK"' EXIT INT TERM

# One digest over every file in a tree and its bytes, so two trees can be
# compared in one comparison rather than path by path.
tree_fingerprint() { # dir
  (
    cd "$1" 2>/dev/null || exit 1
    find . -type f | LC_ALL=C sort | while read -r p; do
      printf '%s %s\n' "$p" "$(sha256sum "$p" | cut -d' ' -f1)"
    done
  ) | sha256sum | cut -d' ' -f1
}

CLONE="$WORK/clone"
git init -q "$CLONE" 2>/dev/null || {
  printf 'publish-data: cannot create a workspace\n' >&2
  exit 2
}
git -C "$CLONE" remote add origin "$REMOTE" || exit 2

# ⚠ A branch that does not exist yet is a first publication, not an error. An
# empty prior tree is what the append comparison is given then, which is the
# same explicit gesture the checker asks a person for.
HAVE_BRANCH=0
if git -C "$CLONE" fetch -q origin "$BRANCH" 2>/dev/null; then
  HAVE_BRANCH=1
  git -C "$CLONE" checkout -q -B "$BRANCH" FETCH_HEAD || exit 2
  # ⛔ The prior tree is what is published, not what is in the workspace after
  # the copy below. It is taken out now and kept.
  git -C "$CLONE" archive FETCH_HEAD | tar -x -C "$WORK/prior" || exit 2
else
  git -C "$CLONE" checkout -q --orphan "$BRANCH" || exit 2
fi

# ⛔ THE BUNDLE DESCRIBES ITSELF, BEFORE ANYTHING IS COMMITTED. A bundle with no
# `MANIFEST.json` or `SHA256SUMS` used to be pushed and only then refused at the
# read-back, which left a published commit nothing could verify; a dry run
# reported it clean. Found on 2026-10-14 by the first bundle assembled from a
# real capture, before its descriptions were written. `CI-09`.
for _doc in MANIFEST.json SHA256SUMS; do
  [ -f "$BUNDLE/$_doc" ] || {
    printf 'publish-data: the bundle carries no %s; assemble it before publishing\n' "$_doc" >&2
    exit 1
  }
done
if ! (cd "$BUNDLE" && sha256sum -c SHA256SUMS --quiet) >"$WORK/bundle-sums.log" 2>&1; then
  printf "publish-data: a digest in the bundle's SHA256SUMS does not match its bytes\n" >&2
  sed 's/^/  /' "$WORK/bundle-sums.log" >&2
  exit 1
fi

# ⛔ THE APPEND GATE, BEFORE ANYTHING IS COMMITTED. Unpiped: the output goes to a
# file and $? is read on the next line.
"$CHECKER" "$WORK/prior" "$BUNDLE" >"$WORK/append.log" 2>&1
APPEND_RC=$?
if [ "$APPEND_RC" != "0" ]; then
  printf 'publish-data: the bundle does not append to %s (exit %s)\n' "$BRANCH" "$APPEND_RC" >&2
  sed 's/^/  /' "$WORK/append.log" >&2
  exit 1
fi

# ⚠ Everything tracked is removed first and the bundle copied in whole, so a
# path the bundle no longer carries would show as a deletion. The gate above is
# what refuses that; this is only how the tree is made to match.
if [ "$HAVE_BRANCH" = "1" ]; then
  git -C "$CLONE" rm -r -q --cached . >/dev/null 2>&1
  find "$CLONE" -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
fi
tar -c -C "$BUNDLE" . | tar -x -C "$CLONE" || exit 2
git -C "$CLONE" add -A || exit 2

if git -C "$CLONE" diff --cached --quiet; then
  printf 'publish-data: the bundle is already published; nothing to append\n'
  exit 0
fi

[ -n "$MESSAGE" ] || MESSAGE="append an assembled bundle"
git -C "$CLONE" -c user.name="${GIT_AUTHOR_NAME:-bit-ids}" \
  -c user.email="${GIT_AUTHOR_EMAIL:-bit-ids@invalid}" \
  commit -q -m "$MESSAGE" || exit 2

if [ "$DRY_RUN" = "1" ]; then
  printf 'publish-data: dry run, %s object(s) staged for %s, nothing pushed\n' \
    "$(find "$BUNDLE" -type f | wc -l | tr -d ' ')" "$BRANCH"
  exit 0
fi

# ⛔ NO FORCE. The refspec has no leading +, and git refuses a non-fast-forward
# push by default, so a branch that moved under this run fails here rather than
# overwriting whatever moved it.
if ! git -C "$CLONE" push -q origin "HEAD:refs/heads/$BRANCH" 2>"$WORK/push.log"; then
  printf 'publish-data: the push was refused\n' >&2
  sed 's/^/  /' "$WORK/push.log" >&2
  exit 1
fi

# ⛔ A PUSH IS NOT COMPLETION. Fetch it back, take the tree out, and check all
# three: that it is the bundle, that it appends to what was there, and that its
# own checksums describe it.
git -C "$CLONE" fetch -q origin "$BRANCH" || {
  printf 'publish-data: cannot read the branch back\n' >&2
  exit 1
}
git -C "$CLONE" archive FETCH_HEAD | tar -x -C "$WORK/after" || exit 1

# ⛔ IS IT THE BUNDLE? A remote that accepted the push and stored something else
# satisfies every other check below, because those compare what came back
# against the tree that was already there.
if [ "$(tree_fingerprint "$BUNDLE")" != "$(tree_fingerprint "$WORK/after")" ]; then
  printf 'publish-data: what came back is not what was pushed\n' >&2
  exit 1
fi

"$CHECKER" "$WORK/prior" "$WORK/after" >"$WORK/readback.log" 2>&1
READBACK_RC=$?
if [ "$READBACK_RC" != "0" ]; then
  printf 'publish-data: what came back does not append to what was there (exit %s)\n' \
    "$READBACK_RC" >&2
  sed 's/^/  /' "$WORK/readback.log" >&2
  exit 1
fi

if [ -f "$WORK/after/SHA256SUMS" ]; then
  if ! (cd "$WORK/after" && sha256sum -c SHA256SUMS --quiet) >"$WORK/sums.log" 2>&1; then
    printf 'publish-data: a digest in the published SHA256SUMS does not match\n' >&2
    sed 's/^/  /' "$WORK/sums.log" >&2
    exit 1
  fi
else
  printf 'publish-data: the published tree carries no SHA256SUMS to verify\n' >&2
  exit 1
fi

printf 'appended to %s: %s object(s), read back and verified\n' \
  "$BRANCH" "$(find "$WORK/after" -type f | wc -l | tr -d ' ')"
