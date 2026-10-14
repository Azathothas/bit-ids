#!/bin/sh
# scan-bundle.sh - rule 12 over a directory that is about to be uploaded.
#
# ⛔ ONE HOME FOR WHAT A CREDENTIAL LOOKS LIKE. `capture-client.sh` and
# `install-client.sh` each carried these two rules, written to be identical, and
# a comment in the second said why: two spellings drift, and the one a reader
# trusts is whichever they saw first. ⭐ They call this now, and so does the
# workflow step that scans a REFUSED capture before its evidence is uploaded -
# the third door, and the one that made a second copy a third.
#
# -- ⛔ WHY A CAPTURE'S OUTPUT IS SCANNED AT ALL -------------------------------
#
# A CAPTURE'S OUTPUT DIRECTORY IS ITS EVIDENCE BUNDLE, AND A PRODUCT THIS
# PROJECT DID NOT WRITE HAS BEEN WRITING INTO IT. Measured on `capture-client`
# run 11 by downloading the artifact and listing it: `aria2-next` generated a
# per-run RPC token, wrote it to `<workdir>/rpc-token` so `stop` could use it,
# and the runner packed that directory - so the token shipped inside the upload.
# ⚠ Neither step was wrong alone and no reading found it. `check-no-secrets`
# structurally cannot: the bundle is not in the tree it scans.
#
# ⛔ THE ADAPTER'S OWN FIX WAS A GATE ON ONE PATH. `aria2-next` unlinks the token
# in `stop`, which is correct and depends on every adapter remembering, on `stop`
# being reached, and on `stop` succeeding. So the guard belongs where every
# upload passes, which is here - and a refused capture is exactly the run whose
# `stop` may never have been reached.
#
# ⛔ TWO RULES, BECAUSE THE TWO FAILURES LOOK NOTHING ALIKE. A file whose NAME
# says it holds a credential is the run's own state; a secret-shaped MEMBER
# inside a document the product answered with is the product handing one back.
# ⚠ The second is the one no filename rule reaches: an options dump names its own
# `rpc-secret`.
#
# ⚠ A WORD BOUNDARY, NOT A SUBSTRING. `tokenizer` is not a token, and a rule that
# refused it would be one somebody switches off.
#
# ⚠ BOTH RULES ARE ALWAYS ASKED, and both are reported when both fire. A caller
# that learned of the second only after fixing the first would spend a dispatch
# on it.
#
# -- ⚠ THE SCOPE IS THE CALLER'S, BECAUSE IT IS WHAT THE CALLER SHIPS ----------
#
# `--top-level` reads the directory's own files and nothing below them. The
# install workdir is uploaded as `install-<route>/*.log` and `*.err`, globs that
# do not descend, and a source route clones a tree whose vendored test fixtures
# are credential-NAMED and public: run 15 was refused over
# `src/third_party/openssl/apps/server.pem`, which nothing shipped. ⛔ A capture
# bundle is uploaded whole, so it is scanned whole.
#
# Usage:
#   sh scripts/capture/scan-bundle.sh [--top-level] <dir>
#
# Exit codes: 0 nothing rule 12 forbids, 1 something would ship that it
# forbids, 2 could not run. The paths found are printed on stderr and the values
# never are, because a guard that echoed the secret it found would put it in the
# run log.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

DEPTH=""
DIR=""
while [ $# -gt 0 ]; do
  case "$1" in
    --top-level) DEPTH="-maxdepth 1" ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    -*)
      printf 'scan-bundle: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
    *)
      [ -z "$DIR" ] || {
        printf 'scan-bundle: one directory, not two\n' >&2
        exit 2
      }
      DIR=$1
      ;;
  esac
  shift
done

[ -n "$DIR" ] || {
  printf 'scan-bundle: a directory is required\n' >&2
  exit 2
}
# ⛔ A DIRECTORY THAT IS NOT THERE IS COULD-NOT-RUN, NOT CLEAN. A scan over
# nothing finds nothing, and a caller reading 0 would upload whatever a path it
# mistyped did not name.
[ -d "$DIR" ] || {
  printf 'scan-bundle: %s is not a directory\n' "$DIR" >&2
  exit 2
}
for tool in find awk grep; do
  command -v "$tool" >/dev/null 2>&1 || {
    printf 'scan-bundle: %s not found\n' "$tool" >&2
    exit 2
  }
done

RC=0

# ⚠ `$DEPTH` is unquoted on purpose: it is empty or one option and its value,
# and neither carries a space that is not a separator.
# shellcheck disable=SC2086
NAMED=$(find "$DIR" $DEPTH -type f 2>/dev/null | awk -F/ '
  {
    name = tolower($NF)
    if (name ~ /(^|[^a-z])(token|secret|passwd|password|credential)([^a-z]|$)/ ||
        name ~ /\.(pem|key|p12|pfx|jks|keystore)$/ ||
        name ~ /^id_(rsa|ed25519|ecdsa)/ ||
        name ~ /^\.env/) print
  }')
if [ -n "$NAMED" ]; then
  printf '%s\n' "$NAMED" | sed 's/^/          /' >&2
  printf 'scan-bundle: %s holds a file whose name says it holds a credential; rule 12\n' "$DIR" >&2
  RC=1
fi

# ⚠ `-I` SKIPS BINARY, so a generated torrent and any artifact bytes are not
# scanned as text. `-l` names the file and never prints the value.
# shellcheck disable=SC2086
VALUED=$(find "$DIR" $DEPTH -type f -exec grep -IlE \
  '"[A-Za-z0-9_-]*(secret|token|password)[A-Za-z0-9_-]*"[[:space:]]*:[[:space:]]*"[^"]{8,}"|--[A-Za-z0-9-]*(secret|token|password)[A-Za-z0-9-]*=[^[:space:]"]{8,}' \
  {} + 2>/dev/null)
if [ -n "$VALUED" ]; then
  printf '%s\n' "$VALUED" | sed 's/^/          /' >&2
  printf 'scan-bundle: %s holds a document carrying a secret-shaped value; rule 12\n' "$DIR" >&2
  RC=1
fi

exit "$RC"
