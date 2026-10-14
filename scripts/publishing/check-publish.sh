#!/bin/sh
# check-publish.sh - publish to a disposable bare repository, plant every defect
# the publisher exists to refuse, and read the branch back each time.
#
# ⛔ NOTHING HERE TOUCHES A REAL REMOTE. Every case runs against a bare
# repository this script creates in a scratch directory and deletes on exit, so
# the publisher's push path is exercised for real without a network or a
# credential anywhere in it.
#
# -- ⛔ WHAT THE BRANCH LOOKS LIKE AFTER A REFUSAL IS PART OF THE CASE ------
#
# A publisher that refuses loudly and half-pushes anyway is worse than one that
# does neither. Every refusal below checks the commit count on the branch as
# well as the exit code.
#
# -- ⭐ TWO CASES PROVE PROPERTIES THIS PUBLISHER RELIES ON RATHER THAN ITS
#       OWN CODE ------------------------------------------------------------
#
# That git refuses a non-fast-forward push under the refspec this publisher
# uses is a property of git, measured here on this host rather than assumed from
# the manual. And that no force flag has crept into the source is a reading of
# the source itself, because a flag added later would pass every behavioural
# case above it right up until the day two publishers raced.
#
# Usage:
#   sh scripts/publishing/check-publish.sh
#   sh scripts/publishing/check-publish.sh --json
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
      printf 'check-publish: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

# ⛔ Resolved from this script's own location, never from the working directory.
HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# ⚠ ME is set here and read by store-lib.sh, which this sources on the next
# line. shellcheck cannot see across a source it is not told to follow.
# shellcheck disable=SC2034
ME=check-publish
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require cargo sha256sum git tar
BUILDER=$(store_build "$ROOT" build-store) || exit 2
INDEXER=$(store_build "$ROOT" build-indexes) || exit 2
ASSEMBLER=$(store_build "$ROOT" assemble-release) || exit 2
PUBLISHER="$ROOT/scripts/publishing/publish-data.sh"
[ -f "$PUBLISHER" ] || {
  printf 'check-publish: %s not found\n' "$PUBLISHER" >&2
  exit 2
}

WORK=$(store_workdir checkpublish) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

BUNDLE="$WORK/bundle"
REMOTE="$WORK/remote"

# Rebuilds the derived half of a bundle over whatever records it holds.
reassemble() {
  rm -f "$BUNDLE/MANIFEST.json" "$BUNDLE/SHA256SUMS"
  mkdir -p "$BUNDLE/indexes/v1" || return 1
  "$INDEXER" --scheme fixture-client:-:3:3 "$BUNDLE" "$BUNDLE/indexes/v1/profiles.json" \
    >/dev/null 2>&1 || return 1
  cp "$ROOT/LICENSE" "$BUNDLE/LICENSE" || return 1
  "$ASSEMBLER" "$BUNDLE" >/dev/null 2>&1
}

fresh_remote() {
  rm -rf "$REMOTE" "$BUNDLE"
  mkdir -p "$BUNDLE" || return 1
  git init -q --bare "$REMOTE" || return 1
  "$BUILDER" --version 1.2.3 "$BUNDLE" >/dev/null 2>&1 || return 1
  reassemble
}

# ⛔ Unpiped. The output goes to a file and $? is read on the next line.
run_publish() {
  sh "$PUBLISHER" --bundle "$BUNDLE" --remote "$REMOTE" >"$WORK/out" 2>&1
  RC=$?
}

commits() {
  git -C "$REMOTE" rev-list --count data 2>/dev/null || printf '0\n'
}

if ! fresh_remote; then
  printf 'check-publish: cannot build the fixture bundle\n' >&2
  exit 2
fi

run_publish
if [ "$RC" = "0" ] && [ "$(commits)" = "1" ]; then
  pass "clean    a first publication creates the branch with one commit"
else
  fail "clean    exit $RC, $(commits) commit(s): $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

# ⛔ THE CASE A DRIVEN RUN FOUND. A second bundle changes every derived file by
# design, and an append rule applied to those would make a correct second
# publication impossible.
"$BUILDER" --version 1.2.10 "$BUNDLE" >/dev/null 2>&1
reassemble
run_publish
if [ "$RC" = "0" ] && [ "$(commits)" = "2" ]; then
  pass "append   a second version appends, and the derived files move with it"
else
  fail "append   exit $RC, $(commits) commit(s): $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

# ⚠ Identical bytes push nothing, which is what docs/publishing.md says a run
# that assembles the same tree does.
run_publish
if [ "$RC" = "0" ] && [ "$(commits)" = "2" ]; then
  pass "idempotent  an unchanged bundle pushes no commit"
else
  fail "idempotent  exit $RC, $(commits) commit(s): $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

# ⛔ A refusal leaves the branch exactly where it was.
BEFORE=$(commits)
rm -rf "$BUNDLE/raw/v1/fixture-client/1.2.3"
reassemble
run_publish
if [ "$RC" = "1" ] && grep -q -F -e "E-STO-20" "$WORK/out" && [ "$(commits)" = "$BEFORE" ]; then
  pass "E-STO-20  a bundle that drops published evidence, and the branch is untouched"
else
  fail "E-STO-20  exit $RC, $(commits) commit(s), expected $BEFORE: $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

fresh_remote >/dev/null 2>&1
run_publish
BEFORE=$(commits)
RECORD=$(find "$BUNDLE/profiles" -name '*.json' | LC_ALL=C sort | head -1)
if replace_once "$RECORD" "Schema Fixture Client" "Schema Fixture Cliant"; then
  reassemble
  run_publish
  if [ "$RC" = "1" ] && grep -q -F -e "E-STO-21" "$WORK/out" && [ "$(commits)" = "$BEFORE" ]; then
    pass "E-STO-21  a bundle that rewrites a published record, and the branch is untouched"
  else
    fail "E-STO-21  exit $RC, $(commits) commit(s), expected $BEFORE: $(head -3 "$WORK/out" | tr '\n' ' ')"
  fi
else
  fail "E-STO-21  NOT-PLANTED (the record edit did not apply)"
fi

# ⛔ A PUSH THAT REPORTED SUCCESS AND LANDED SOMETHING ELSE. A post-receive hook
# moves the ref back after the push is accepted, which is a remote doing exactly
# what the read-back exists to catch. Without a case like this the read-back is a
# guard nothing has ever seen refuse.
#
# ⭐ Writing it found that the publisher could not catch this at all. It compared
# what came back against the prior tree, and a rewound ref appends to the prior
# tree perfectly, because it IS the prior tree. The comparison against the bundle
# that was pushed is what this case now proves.
fresh_remote >/dev/null 2>&1
run_publish
FIRST=$(git -C "$REMOTE" rev-parse data)
"$BUILDER" --version 1.2.10 "$BUNDLE" >/dev/null 2>&1
reassemble
mkdir -p "$REMOTE/hooks"
cat >"$REMOTE/hooks/post-receive" <<EOF
#!/bin/sh
git update-ref refs/heads/data $FIRST
EOF
chmod +x "$REMOTE/hooks/post-receive"
run_publish
rm -f "$REMOTE/hooks/post-receive"
if [ "$RC" = "1" ] && grep -q 'what came back is not what was pushed' "$WORK/out"; then
  pass "read-back  a remote that moves the ref after accepting the push is caught"
else
  fail "read-back  exit $RC: $(head -3 "$WORK/out" | tr '\n' ' ')"
fi

# ⛔ A bundle with nothing to verify is refused BEFORE THE PUSH, because "no
# checksums" and "checksums all matched" are the same silence. ⚠ This case used
# to accept a refusal at the read-back, after the push had landed: the branch
# then carried a commit nothing could verify. `CI-09`, 2026-10-14.
fresh_remote >/dev/null 2>&1
rm -f "$BUNDLE/SHA256SUMS"
run_publish
if [ "$RC" = "1" ] && grep -q 'carries no SHA256SUMS' "$WORK/out" && [ "$(commits)" = "0" ]; then
  pass "bundle   a bundle with no checksum file is refused, and nothing is pushed"
else
  fail "bundle   expected a refusal before the push, exit $RC, $(commits) commit(s): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

fresh_remote >/dev/null 2>&1
rm -f "$BUNDLE/MANIFEST.json"
run_publish
if [ "$RC" = "1" ] && grep -q 'carries no MANIFEST.json' "$WORK/out" && [ "$(commits)" = "0" ]; then
  pass "bundle   a bundle with no manifest is refused, and nothing is pushed"
else
  fail "bundle   expected a refusal before the push, exit $RC, $(commits) commit(s): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

# A byte changed after the bundle was assembled is one its own checksums name.
fresh_remote >/dev/null 2>&1
printf 'x' >>"$BUNDLE/LICENSE"
run_publish
if [ "$RC" = "1" ] && grep -q 'does not match its bytes' "$WORK/out" && [ "$(commits)" = "0" ]; then
  pass "bundle   a bundle edited after assembly is refused, and nothing is pushed"
else
  fail "bundle   expected a refusal before the push, exit $RC, $(commits) commit(s): $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

# ⛔ A branch name carrying a + would be the second half of a forcing refspec.
fresh_remote >/dev/null 2>&1
sh "$PUBLISHER" --bundle "$BUNDLE" --remote "$REMOTE" --branch '+data' >"$WORK/out" 2>&1
RC=$?
if [ "$RC" = "2" ] && grep -q 'refusing branch name' "$WORK/out"; then
  pass "refspec  a branch name that could smuggle a force is refused"
else
  fail "refspec  exit $RC: $(head -2 "$WORK/out" | tr '\n' ' ')"
fi

# ⭐ A READING OF THE SOURCE, not of its behaviour. A force flag added later
# would pass every case above right up until two publishers raced.
#
# ⚠ COMMENTS ARE STRIPPED FIRST, and that is not tidiness. The publisher's own
# header explains that it uses no force, and the first version of this case
# matched that sentence and reported the guard broken. A source check that reads
# prose is a check that fires on the documentation of the rule it enforces.
sed 's/[[:space:]]*#.*$//' "$PUBLISHER" >"$WORK/code.sh"
if grep -nE -- '--force|--force-with-lease|push[[:space:]].*[[:space:]]-f([[:space:]]|$)|refs/heads/[^"]*"?[[:space:]]*\+|"\+refs' "$WORK/code.sh" >"$WORK/force.log" 2>&1; then
  fail "no-force  the publisher's code carries a forcing flag: $(head -1 "$WORK/force.log")"
else
  pass "no-force  the publisher's code carries no forcing flag or refspec"
fi

# ⛔ And the stripper is itself exercised, because a sed that stripped everything
# would make the case above pass over any source at all.
printf 'git push --force origin HEAD:data\n' >"$WORK/forcing.sh"
sed 's/[[:space:]]*#.*$//' "$WORK/forcing.sh" >"$WORK/forcing-code.sh"
if grep -qE -- '--force' "$WORK/forcing-code.sh"; then
  pass "no-force  the source check still sees a real force flag after stripping"
else
  fail "no-force  the comment stripper removed a force flag that was code"
fi

# ⭐ A PROPERTY OF GIT, MEASURED ON THIS HOST rather than assumed from the
# manual. The publisher's whole non-fast-forward defence is that git refuses one
# under a refspec with no leading +.
NFF="$WORK/nff"
rm -rf "$NFF"
mkdir -p "$NFF"
git init -q --bare "$NFF/remote"
for side in a b; do
  git init -q "$NFF/$side"
  printf '%s\n' "$side" >"$NFF/$side/file"
  git -C "$NFF/$side" add -A
  git -C "$NFF/$side" -c user.name=t -c user.email=t@invalid commit -q -m "$side"
done
git -C "$NFF/a" push -q "$NFF/remote" HEAD:refs/heads/data
if git -C "$NFF/b" push -q "$NFF/remote" HEAD:refs/heads/data 2>/dev/null; then
  fail "non-ff   git accepted a divergent push under a plain refspec on this host"
else
  pass "non-ff   git refuses a divergent push under a plain refspec, measured here"
fi

fresh_remote >/dev/null 2>&1
store_probe_guards "$BUNDLE/LICENSE" "Permission" "e"

store_report check-publish/1 cases "$JSON"
