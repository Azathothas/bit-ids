#!/bin/sh
# install-client.sh - install one target through one route, on a host that has
# already been claimed and still has a route off itself.
#
# ⛔ THIS IS THE ONE STEP THAT RUNS WITH THE NETWORK STILL UP, AND THE ORDER IS
# THE CONTAINMENT. `capture-client` refuses to install anything, so a capture
# that discovered a missing package would have to restore egress on the host
# that exists to have none. Everything a run needs is fetched here and nothing
# is fetched after.
#
# ⚠ IT DELIBERATELY DOES NOT CHECK THE EGRESS GUARD. Being able to reach the
# network is the precondition of this step, not a violation, and a guard here
# would refuse every host it is meant to run on. The claim IS checked: a host
# that already ran a capture must not have a second product installed on it, and
# that is the same marker `capture-client` reads later.
#
# -- ⛔ TWO ROUTES ARE TWO INVOCATIONS ----------------------------------------
#
# One route per call, so a caller that ran one and skipped the other has run one
# and the record says so. A script that installed both would make "two
# independent routes" a property of this file rather than of the workflow that
# is supposed to prove it, and `E-ACQ-07` and `E-ACQ-08` refuse a pair sharing a
# resolver or a delivery mechanism.
#
# ⭐ AND IT PRINTS WHAT THE BUILD SAYS AFTERWARDS, which is the only version
# this project believes. `ACQ-03`'s gate compares those, not the filenames.
#
# ⛔ IT ALSO ASKS THE HOST WHAT IT ALREADY HAD, BEFORE THE ROUTE RUNS, so the
# record says whether this route acquired anything or reported a build that was
# there before it started. See the block above that call for what a route that
# installs nothing does to the two-route rule.
#
# Usage:
#   sh scripts/acquisition/install-client.sh --adapter <path> --route <name>
#                                            --workdir <dir> [--record <file>]
#
# Exit codes: 0 installed and the build reported a version, 1 a guard or the
# route refuses, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

ADAPTER=""
ROUTE=""
WORKDIR=""
RECORD=""

usage() {
  awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
}

refuse() {
  printf 'install-client: %s\n' "$1" >&2
  exit 1
}

cannot() {
  printf 'install-client: %s\n' "$1" >&2
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --adapter)
      [ $# -ge 2 ] || cannot "--adapter takes a value"
      ADAPTER="$2"
      shift
      ;;
    --route)
      [ $# -ge 2 ] || cannot "--route takes a value"
      ROUTE="$2"
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
    -h | --help)
      usage
      exit 0
      ;;
    *)
      printf 'install-client: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

[ -n "$ADAPTER" ] || cannot "--adapter is required"
[ -n "$ROUTE" ] || cannot "--route is required"
[ -n "$WORKDIR" ] || cannot "--workdir is required"
[ -f "$ADAPTER" ] || cannot "the adapter $ADAPTER is not present"

# ⛔ EVERY ADAPTER CALL IS BOUNDED, AND THE BOUND IS HERE RATHER THAN ONLY IN THE
# ADAPTER. An adapter shells out to a product this project did not write, and a
# product that waits on a prompt waits forever: measured on 2026-09-08, the first
# client dispatch had two of three jobs sit in this step for over half an hour
# and report nothing at all, because a hung install is indistinguishable from a
# slow one until the job's own timeout kills the runner and takes the log with
# it. ⚠ A convention in each adapter would be a bound the next adapter forgets;
# this is the one call site every adapter passes through.
#
# ⛔ THAT LAST SENTENCE WAS FALSE FOR AS LONG AS IT HAD BEEN WRITTEN, corrected
# 2026-09-16. Three of this file's four adapter calls carried NO bound at all -
# one `describe` that ran before any other call, and two more inside
# `adapter_binary` - and the two that did carry one spent it inside a command
# substitution, which ends when the PIPE reaches end of file rather than when the
# child exits. ⭐ `adapter_run` is what makes the claim true: it is the only way
# to reach an adapter from here, it bounds the call, and it hands the adapter a
# FILE so nothing the product leaves behind can hold a descriptor this shell is
# still reading.
#
# ⭐ AND STDIN IS /dev/null, so a prompt gets end-of-file rather than a wait.
# ⚠ THE BOUND SITS WELL UNDER THE WORKFLOW STEP'S OWN TIMEOUT, so this reports
# rather than being killed with its log. Measured on 2026-09-08: a 900s bound
# under a 30-minute job did not end the job, and the runner was killed at the
# job timeout with the diagnosis still on it.
# ⛔ AND -k IS NOT DECORATION. `timeout` sends TERM and then waits; a child that
# blocks or ignores it is never killed at all, so the bound reports nothing.
INSTALL_SECONDS=${BIT_IDS_INSTALL_TIMEOUT:-420}
VERSION_SECONDS=${BIT_IDS_VERSION_TIMEOUT:-60}
# ⭐ `describe` PRINTS A FEW LINES AND IS THE CHEAPEST CALL AN ADAPTER HAS, which
# is why it had no bound at all until 2026-09-16 and why the number is small. The
# workflow's own *Probe the preinstalled product* step already used 30 for it.
DESCRIBE_SECONDS=${BIT_IDS_DESCRIBE_TIMEOUT:-30}
KILL_AFTER=${BIT_IDS_KILL_AFTER:-20}
command -v timeout >/dev/null 2>&1 || cannot "timeout is not on this host"

# ⛔ THE ROUTE VOCABULARY IS CLOSED. `docs/client-matrix.md` names two candidate
# routes per target and `RouteKind` is the type they become, so a spelling
# arriving here that is not in this list would install through something no
# record can describe. ⭐ `source` was added on 2026-09-09 for `aria2-next`,
# which no package index carries: its two routes are the vendor's published
# binary and a build of the same tag, and `RouteKind::SourceBuild` is the type
# the second becomes.
case "$ROUTE" in
  package | release | source) : ;;
  *) cannot "unknown route: $ROUTE (package, release or source)" ;;
esac

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)
GUARD="$ROOT/scripts/acquisition/assert-disposable.sh"
[ -f "$GUARD" ] || cannot "$GUARD is not present"

# ⛔ THE CLAIM, AND ONLY THE CLAIM. A host that already ran a capture must not
# have a second product installed on it: the marker is the evidence the host
# survived, and installing over it would contaminate the next run with this
# one's state. ⚠ Egress is deliberately not asserted here; see the header.
MARKER=$(sh "$GUARD" --marker) || cannot "the guard could not report its marker path"
[ -n "$MARKER" ] || cannot "the guard reported an empty marker path"
[ -f "$MARKER" ] ||
  refuse "the host was never claimed; $MARKER does not exist"

mkdir -p "$WORKDIR" || cannot "cannot create $WORKDIR"
WORKDIR=$(CDPATH='' cd -- "$WORKDIR" && pwd)

# ⛔ EVERY ADAPTER CALL GOES THROUGH HERE, AND A BOUND IS ONLY HALF OF WHY.
# The other half is that NOTHING THE ADAPTER SPAWNS MAY HOLD A PIPE THIS SHELL IS
# READING. A command substitution finishes when the pipe reaches end of file, not
# when the command exits, so a product that leaves one process behind blocks
# `$( )` for as long as that process lives - and `timeout` firing does not help,
# because the child it kills is not the one holding the descriptor.
#
# ⚠ MEASURED ON 2026-09-16, which is what turned this from a tidy-up into a
# repair: an adapter whose `version` leaves a `sleep` behind was still blocking
# its substitution at 25 seconds under a 4-second bound. That is a six-second
# step becoming a thirty-minute one, which is exactly what `capture-client` runs
# 21, 22 and 23 did inside *Install the client*.
#
# ⛔ THIS FILE'S OWN HEADER CLAIMED THE BOUND WAS ALREADY UNIVERSAL and it was
# not: `describe` was called three times with no bound at all, and the two
# `version` calls were bounded inside a substitution that ignored the bound. The
# claim is true now because one function is the only way to reach an adapter.
#
# ⭐ STDOUT GOES TO A FILE AND THE CALLER READS THE FILE. `install-step.sh`
# already does this for THIS script's output; the door it closed was reopened
# four times here, one directory down.
#
# Usage: adapter_run <seconds> <out-file> <err-file> <subcommand> [args...]
# Sets no variable; the caller reads <out-file>. Returns the adapter's exit code,
# or 124 from `timeout`.
adapter_run() {
  _bound=$1
  _out=$2
  _err=$3
  shift 3
  # ⭐ AND STDIN IS /dev/null, so a product that prompts gets end-of-file.
  timeout -k "$KILL_AFTER" "$_bound" sh "$ADAPTER" "$@" \
    >"$_out" 2>"$_err" </dev/null
}

# ⚠ READING THE FILE IS STILL A SUBSTITUTION AND THAT IS SAFE. `awk` is this
# project's own process over a regular file: it cannot spawn a product, and a
# file has an end where a pipe held open does not.
adapter_field() { # <out-file> <key>
  awk -F= -v k="$2" '$1 == k { sub(/^[^=]*=/, ""); print; exit }' <"$1"
}

# ⚠ THE WHOLE VALUE, NOT THE SECOND FIELD. `awk -F= '{ print $2 }'` stops at a
# second `=`, so a value carrying one arrives truncated with nothing saying so.
# Measured on 2026-09-09 over `weird=a=b`, which that form reads as `a`. ⛔ No key
# an adapter prints today carries one - `binary` is a path and `target` a slug -
# which is exactly when a reader is easiest to get wrong, and `release_asset` is
# a pattern whose future shapes nobody here has fixed.
# ⛔ THIS WAS THE FIRST UNBOUNDED ADAPTER CALL AND IT RAN BEFORE EVERY OTHER ONE,
# so a `describe` that never returned wedged the step before a single bound in
# this file had been reached.
adapter_run "$DESCRIBE_SECONDS" "$WORKDIR/describe.log" "$WORKDIR/describe.err" describe
DESCRIBE_RC=$?
[ "$DESCRIBE_RC" = 0 ] ||
  cannot "the adapter would not describe itself (exit $DESCRIBE_RC)"
TARGET=$(adapter_field "$WORKDIR/describe.log" target)
[ -n "$TARGET" ] || cannot "the adapter named no target"

STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# ⛔ WHAT THE HOST ALREADY HAD, ASKED BEFORE THE ROUTE RUNS. `install` is
# contracted to install the target and nothing established that it did: a route
# that installed nothing exits 0, and this file then reported a version the route
# had not put there as the version that route acquired.
#
# ⚠ MEASURED, NOT HYPOTHETICAL. `aria2` ships on the `ubuntu-24.04` image, so
# `apt-get install aria2` there prints `already the newest version`, installs
# nothing and exits 0; client capture runs 3 and 4 recorded `route=package` over
# exactly that. ⛔ AND THE `release` ROUTES HAVE THE SAME SHAPE FROM THE OTHER
# END: each fetches an artifact into the workdir and never makes it the
# executable `version` asks, so a release route run on a host that already has
# the package build reports the PACKAGE build's version as its own.
#
# ⛔ THAT IS THE TWO-ROUTE RULE FAILING THROUGH THE ONE DOOR NOTHING WATCHES.
# `E-ACQ-07` and `E-ACQ-08` compare what a route DECLARES - its resolver and its
# delivery - so two routes that each installed nothing declare two independent
# routes, agree on the version for the most trivial reason available, and hand
# `ACQ-03` one binary to compare against itself. That comparison is by executable
# digest, so the answer is `byte_identical`: the STRONGEST verdict the
# classification has, reached by acquiring nothing at all.
#
# ⚠ A REFUSAL HERE WOULD THROW AWAY A REAL MEASUREMENT. The build on such a host
# is genuine and its identity is worth capturing; what is not genuine is the
# claim that this route acquired it. So the fact is recorded and said out loud,
# and the gate that compares two routes is where it bites - the same argument
# this file's header makes for not owning the two-route rule itself.
#
# ⚠ ITS STDERR IS KEPT AND NEVER PRINTED AS A DIAGNOSIS. A target that is not yet
# installed is the ORDINARY case here, so the adapter's refusal goes to the
# workdir, where the install artifact carries it, rather than to a log where it
# would read as a failure of the step.
adapter_run "$VERSION_SECONDS" "$WORKDIR/version-before.log" \
  "$WORKDIR/version-before.err" version
PREEXISTING_RC=$?
PREEXISTING=$(cat "$WORKDIR/version-before.log")
[ "$PREEXISTING_RC" = 0 ] || PREEXISTING=""

# ⛔ AND THE EXECUTABLE ITSELF, BECAUSE A VERSION IS NOT AN IDENTITY. Measured on
# 2026-09-08 with the real product: `aria2` ships on `ubuntu-24.04` at 1.37.0 and
# the vendor's newest release is 1.37.0, so the release route there installs a
# genuinely different build - different bytes, different features - and a
# comparison of version strings alone calls that no acquisition at all. ⚠ That is
# not a hypothetical; it is what the first two-route capture would have recorded.
#
# ⭐ `describe` names the executable it would ask, so the digest before and after
# is available without a sixth subcommand. An adapter that names none simply
# leaves the field empty and the verdict falls back to the version.
# ⚠ EACH CALL RE-ASKS THE ADAPTER AND KEEPS ITS OWN FILE. The two calls straddle
# the install, so they are two different answers about the host and folding them
# into one file would overwrite the `before` with the `after`.
# ⛔ AND A `describe` THAT FAILS HERE IS NOT FATAL, unlike the one above: the
# digest is a refinement on the version comparison and an adapter that names no
# executable leaves the field empty by design. `cannot` here would refuse a route
# over a diagnostic.
adapter_binary() { # <tag>
  adapter_run "$DESCRIBE_SECONDS" "$WORKDIR/describe-$1.log" \
    "$WORKDIR/describe-$1.err" describe || return 0
  adapter_field "$WORKDIR/describe-$1.log" binary
}
digest_of() {
  [ -n "$1" ] && [ -f "$1" ] || return 0
  sha256sum "$1" 2>/dev/null | cut -d' ' -f1
}
PRE_BINARY=$(adapter_binary before)
PRE_DIGEST=$(digest_of "$PRE_BINARY")

timeout -k "$KILL_AFTER" "$INSTALL_SECONDS" sh "$ADAPTER" install "$ROUTE" "$WORKDIR" </dev/null
INSTALL_RC=$?

# ⚠ THE LOGS THE ROUTE WROTE ARE PRINTED ON EVERY FAILURE, INDENTED. Without
# this the job log carries the refusal and not its cause, and the workdir is
# uploaded only when the capture that follows succeeds - so a route that failed
# took its own diagnosis with it.
show_route_logs() {
  for _log in "$WORKDIR/update.log" "$WORKDIR/install.log"; do
    [ -f "$_log" ] || continue
    printf '          -- %s (last 20 lines)\n' "$_log" >&2
    tail -20 "$_log" | sed 's/^/          /' >&2
  done
}

case "$INSTALL_RC" in
  0) : ;;
  # ⛔ 124 IS coreutils' VERDICT FOR "IT NEVER ANSWERED", and it is a different
  # fact from a route that refused. A run that folded them together would report
  # a package index as unavailable over a product that asked a question.
  124)
    show_route_logs
    refuse "the $ROUTE route did not answer within ${INSTALL_SECONDS}s; it is hung, not slow"
    ;;
  1)
    show_route_logs
    refuse "the $ROUTE route did not install $TARGET"
    ;;
  *)
    show_route_logs
    cannot "the $ROUTE route could not run (exit $INSTALL_RC)"
    ;;
esac

# ⛔ AND THE BUILD IS ASKED, HERE, WHILE THERE IS STILL SOMETHING TO DO ABOUT A
# FAILURE. An install whose executable cannot answer is a route that did not
# work, and finding that out under containment would be finding it out too late.
# ⛔ THE ADAPTER'S OWN MESSAGE IS KEPT, NOT DISCARDED. This read `2>/dev/null`,
# so a refusal arrived here as a bare exit code and the step printed "would not
# report a version" over an adapter that had said exactly which of its three
# refusals fired. ⚠ Measured on 2026-09-08: a client capture reported that and
# the cause could not be read from the run at all. Silencing the diagnosis and
# then reporting its absence is one defect, not two.
adapter_run "$VERSION_SECONDS" "$WORKDIR/version.log" "$WORKDIR/version.err" version
VERSION_RC=$?
VERSION=$(cat "$WORKDIR/version.log")
show_adapter_error() {
  [ -s "$WORKDIR/version.err" ] || return 0
  printf '          -- the adapter said\n' >&2
  sed 's/^/          /' "$WORKDIR/version.err" >&2
}
case "$VERSION_RC" in
  0) : ;;
  124)
    show_adapter_error
    refuse "$TARGET installed through the $ROUTE route and did not answer --version within ${VERSION_SECONDS}s"
    ;;
  *)
    show_adapter_error
    refuse "$TARGET installed through the $ROUTE route but would not report a version (adapter exit $VERSION_RC)"
    ;;
esac
[ -n "$VERSION" ] ||
  refuse "$TARGET installed through the $ROUTE route and reported an empty version"

# ⛔ AND HERE IS WHETHER THE ROUTE ACQUIRED ANYTHING AT ALL. Absent before and
# answering now is an install. A DIFFERENT version is an upgrade, which is also
# an install, and reading it as anything else would refuse the ordinary case of a
# package index carrying a newer build than the image. The same version over a
# target that was already there is neither, whatever the route reported.
POST_BINARY=$(adapter_binary after)
POST_DIGEST=$(digest_of "$POST_BINARY")
if [ -z "$PREEXISTING" ]; then
  ACQUIRED=yes
elif [ "$PREEXISTING" != "$VERSION" ]; then
  ACQUIRED=yes
elif [ -n "$PRE_DIGEST" ] && [ -n "$POST_DIGEST" ] && [ "$PRE_DIGEST" != "$POST_DIGEST" ]; then
  # ⭐ SAME VERSION, DIFFERENT EXECUTABLE. This is the branch a version
  # comparison cannot reach, and it is the one the aria2 pair actually needs.
  ACQUIRED=yes
else
  ACQUIRED=no
fi

# ⚠ SAID OUT LOUD, ON stderr, WHERE THE STEP LOG KEEPS IT. A field in a file
# nobody opens is how this went unnoticed for four dispatches.
[ "$ACQUIRED" = yes ] ||
  printf 'install-client: %s %s was ALREADY on this host and the %s route installed nothing; this run acquired no build\n' \
    "$TARGET" "$VERSION" "$ROUTE" >&2

FINISHED=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# ⚠ READ RATHER THAN DERIVED, AND EMPTY IS A FACT. A route that cloned nothing
# leaves no file here and records an empty commit; a `source` route that left one
# is refused below, because a source build whose commit nothing recorded cannot
# become a record and failing at the install is cheaper than failing at the
# assembly a dispatch later.
SOURCE_COMMIT=""
if [ -f "$WORKDIR/source-commit" ]; then
  SOURCE_COMMIT=$(tr -d ' \n\r' <"$WORKDIR/source-commit")
fi

# ⛔ HOW THE ROUTE PACKAGED WHAT IT INSTALLED, AND A ROUTE THAT INSTALLED
# WITHOUT SAYING IS REFUSED. `Build.package` is part of the identity tuple a
# store path is derived from, so a record filed without it cannot be filed at
# all and one filed with a guess files two packagings of a version at one name.
# ⚠ Measured on 2026-09-09: `assemble-capture` refused both of run 14's lanes
# for it, because no adapter wrote it and nothing else knew.
PACKAGE=""
if [ -f "$WORKDIR/package" ]; then
  PACKAGE=$(tr -d ' \n\r' <"$WORKDIR/package")
fi
case $PACKAGE in
  # A slug: lowercase letters, digits and hyphens, and neither leading nor
  # trailing. Checked here rather than at the record, because an adapter that
  # spelled it `ELF Binary` should fail at the install with a message naming
  # the adapter rather than a dispatch later with one naming a schema.
  "") refuse "the $ROUTE route installed and recorded no package format" ;;
  *[!a-z0-9-]* | -* | *-) refuse "the $ROUTE route recorded package [$PACKAGE], which is not a slug" ;;
  *) ;;
esac
if [ "$ROUTE" = source ]; then
  case $SOURCE_COMMIT in
    # ⛔ Forty lowercase hex digits, checked as a shape rather than for
    # emptiness: `rev-parse` prints an abbreviation when asked for one, and
    # `E-ACQ-06` exists because an abbreviated commit is not an immutable
    # reference.
    *[!0-9a-f]* | "") refuse "the source route recorded no full commit object name" ;;
    *)
      [ ${#SOURCE_COMMIT} -eq 40 ] ||
        refuse "the source route recorded [$SOURCE_COMMIT], which is not a full object name"
      ;;
  esac
fi

# -- rule 12, on the OTHER path that ships an artifact -------------------------
#
# ⛔ THE INSTALL WORKDIR IS UPLOADED TOO, AND THAT WAS A GATE ON ONE PATH.
# `capture-client` gained a credential scan on 2026-09-10, after a per-run RPC
# token shipped inside a capture bundle; a door sweep the same day found that
# `capture-client.yml` uploads THIS directory's logs as a second artifact and
# nothing scanned them. ⚠ The route runs an adapter's `install`, which clones,
# configures and builds - all of which log - and a route that ever needs a
# credential for a private index would put it here rather than in the bundle.
#
# ⚠ NO SUCH ROUTE EXISTS TODAY, which is exactly the condition
# `docs/methodology/reviews.md` names as the easiest place to get a scope wrong:
# every reading agrees on every file. The guard is written for the shape.
#
# ⛔ THE RULES ARE `capture-client`'s, AND SINCE 2026-10-14 THEY ARE ONE FILE.
# Two spellings of "what a credential looks like" would drift, and the day they
# did, the one a reader trusts would be whichever they saw first; this carried a
# copy written to be identical until `scan-bundle.sh` took both.
#
# -- ⛔ AND THE SCOPE IS THIS DIRECTORY'S TOP LEVEL, NOT THE TREE BELOW IT ----
#
# ⛔ THE GUARD'S SCOPE AND THE ARTIFACT'S SCOPE DISAGREED, AND A REAL CAPTURE IS
# WHAT SAID SO. `capture-client.yml` uploads `install-<route>/*.log` and
# `install-<route>/*.err` - globs with no `**`, which do not descend - and its
# own comment says why: run 5 uploaded a build tree, 42.6 megabytes and 1867
# entries, for a step called *Upload the install logs*. ⚠ This scan was written
# when that step took the whole directory. The upload narrowed and the guard did
# not follow, so it went on reading a subtree that nothing ships.
#
# ⛔ RUN 15 ON 2026-09-15 IS THE MEASUREMENT. Its `source` lane was refused over
# `src/third_party/openssl/apps/server.pem` and
# `src/third_party/nghttp2/integration-tests/server.key` - openssl's and
# nghttp2's own test fixtures, vendored in the tree this route clones, public by
# construction, and uploaded by nothing. ⚠ A `release` route never meets them
# because it installs a binary, so the shape was invisible for fifteen dispatches
# and appeared the moment the source route first resolved its own tag.
#
# ⚠ NARROWING TO THE TOP LEVEL IS NOT A WEAKENING, AND THE DIRECTION IS WHAT
# SAYS SO. Rule 12 is about what REACHES a remote. What ships from here is
# `*.log` and `*.err` at this level; this scan still reads EVERY file at this
# level, which is deliberately wider than those two globs, and the only thing it
# stops reading is a subtree no artifact can carry.
#
# ⛔ THE DANGEROUS DIRECTION IS THE OTHER ONE, and it is filed rather than
# guarded: an upload path that gained a `**` would ship a subtree this no longer
# reads. `TODO/acquisition.md` carries it with the command that would settle it.
#
# ⚠ THE SAME SCOPE FOR BOTH RULES, for the same reason. `grep -r` descends and
# the value rule here did too, so a source tree's own test configuration was in
# scope for a scan over documents that ship; both halves had the defect, and
# `--top-level` is one flag over both now rather than a fix applied twice.
sh "$ROOT/scripts/capture/scan-bundle.sh" --top-level "$WORKDIR"
SCAN_RC=$?
case "$SCAN_RC" in
  0) : ;;
  1) refuse "the install would ship what rule 12 forbids, named above" ;;
  *) cannot "the rule-12 scan could not run (exit $SCAN_RC)" ;;
esac

# ⚠ Key=value, the shape every other document in this directory uses. ⛔ It
# records the route and the version SEPARATELY per call, because `ACQ-03`
# compares two of these and a single file holding one merged answer would have
# nothing to compare.
if [ -n "$RECORD" ]; then
  {
    printf 'bit-ids/install-record/1\n'
    printf 'target=%s\n' "$TARGET"
    printf 'route=%s\n' "$ROUTE"
    printf 'reported_version=%s\n' "$VERSION"
    # ⚠ EMPTY MEANS THE TARGET WAS NOT INSTALLED WHEN THIS ROUTE STARTED, which
    # is a different fact from a target whose version could not be read. The
    # adapter's own words for the second are in `version-before.err` beside this.
    printf 'preexisting_version=%s\n' "$PREEXISTING"
    # ⚠ EMPTY MEANS THE ADAPTER NAMED NO EXECUTABLE, which is a different fact
    # from one it named and could not digest. Both read as empty here and the
    # verdict falls back to the version either way; an adapter that names one is
    # what makes the digest branch reachable.
    printf 'preexisting_binary=%s\n' "$PRE_BINARY"
    printf 'preexisting_binary_sha256=%s\n' "$PRE_DIGEST"
    printf 'installed_binary=%s\n' "$POST_BINARY"
    printf 'installed_binary_sha256=%s\n' "$POST_DIGEST"
    printf 'acquired=%s\n' "$ACQUIRED"
    # ⛔ THE IMMUTABLE IDENTITY OF A SOURCE ROUTE, AND EMPTY FOR EVERY OTHER.
    # `E-ACQ-06` refuses a source identity without a full object name, and
    # nothing this path wrote carried one until 2026-09-09: assembling
    # capture-client run 14 refused both lanes on it, and the commit was
    # recoverable only from the git output inside `install.log`. The adapter
    # writes it beside its log because the adapter is what cloned; a caller that
    # re-derived it would be reading a tree the route already left.
    printf 'source_commit=%s\n' "$SOURCE_COMMIT"
    printf 'package=%s\n' "$PACKAGE"
    printf 'adapter=%s\n' "$ADAPTER"
    printf 'adapter_sha256=%s\n' "$(sha256sum "$ADAPTER" | cut -d' ' -f1)"
    printf 'workdir=%s\n' "$WORKDIR"
    printf 'started_at=%s\n' "$STARTED"
    printf 'finished_at=%s\n' "$FINISHED"
    printf 'platform=%s\n' "$(uname -srm)"
  } >"$RECORD" || cannot "the install record could not be written to $RECORD"

  # ⛔ EVERY FIELD IS PRESENT, ASKED AS A FIELD RATHER THAN AS A VALUE. A record
  # truncated part-way through the block would carry a correct
  # `reported_version` and no verdict at all, and a reader taking an absent
  # `acquired` for `yes` is the failure this whole change exists to remove.
  for _field in target route reported_version preexisting_version \
    preexisting_binary preexisting_binary_sha256 installed_binary \
    installed_binary_sha256 acquired source_commit package; do
    grep -q "^$_field=" "$RECORD" ||
      cannot "the install record was written without $_field"
  done

  # ⛔ AND THE RECORD IS CHECKED AGAINST ITSELF RATHER THAN RESTATED. `acquired`
  # is derived from two fields that are also in this file, which is one value in
  # two places; this reads all three back off disk and re-derives the third, so a
  # record whose verdict does not follow from its own evidence is refused here
  # instead of being believed downstream.
  # ⚠ READ WITH `sed`, COMPARED WITH `=`. The old check was
  # `grep -q "^reported_version=$VERSION\$"`, where a version is a regular
  # expression: every `.` in `1.37.0` matches any byte, so the check passed over
  # values it was written to distinguish.
  _pre=$(sed -n 's/^preexisting_version=//p' "$RECORD")
  _rep=$(sed -n 's/^reported_version=//p' "$RECORD")
  _acq=$(sed -n 's/^acquired=//p' "$RECORD")
  _pred=$(sed -n 's/^preexisting_binary_sha256=//p' "$RECORD")
  _postd=$(sed -n 's/^installed_binary_sha256=//p' "$RECORD")
  [ "$_rep" = "$VERSION" ] ||
    refuse "the install record says reported_version=[$_rep] over a build that said [$VERSION]"
  if [ -z "$_pre" ]; then
    _want=yes
  elif [ "$_pre" != "$_rep" ]; then
    _want=yes
  elif [ -n "$_pred" ] && [ -n "$_postd" ] && [ "$_pred" != "$_postd" ]; then
    _want=yes
  else
    _want=no
  fi
  [ "$_acq" = "$_want" ] ||
    refuse "the install record says acquired=[$_acq] over preexisting [$_pre] and reported [$_rep]"
fi

printf '%s\n' "$VERSION"
printf 'install-client: %s %s through the %s route (acquired=%s)\n' \
  "$TARGET" "$VERSION" "$ROUTE" "$ACQUIRED"
exit 0
