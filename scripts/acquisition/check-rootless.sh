#!/bin/sh
# check-rootless.sh - `ACQ-06`'s acceptance: every release route installs with no
# privilege, into a prefix the current user owns, and behaves identically on a
# host where `sudo` does not exist at all.
#
# ⛔ WHAT THIS EXISTS TO REFUSE. Every install this project performed needed
# `sudo` and wrote to `/usr/local/bin`, so what it did depended on a host this
# project does not control - and that was the one variable between an install
# that exits 0 in a second here and a step no bound could end on a runner.
# `TODO/acquisition.md` carries the entry.
#
# -- ⛔ THE VENDOR IS A FILE, AND THAT IS THE ONLY WAY THIS CAN BE A GATE ROW ---
#
# A harness that fetched four real artifacts would make the gate depend on four
# vendors being up, and a red row for somebody else's outage is a row people
# learn to re-run rather than read. So each route is handed a `file://` URL, the
# way `resolve-release --listing` is handed a recorded response - and `curl`
# fetches it through the same call, with the same bounds, as a real one.
#
# ⚠ WHAT THAT DOES NOT ESTABLISH is what a vendor publishes today. It
# establishes what the ROUTE does with bytes, which is this entry's subject.
#
# -- ⚠ AND WHICH USER IT DROVE IS PRINTED, NEVER ASSUMED ----------------------
#
# ⛔ A SESSION HOST IS `root` AND A RUNNER IS NOT, and that difference hid two
# real defects for eight dispatches. A root run of a rootless install exercises
# the same code - nothing writes outside the prefix any more - but it cannot
# establish that an unprivileged user could have done it.
#
# ⭐ So `BIT_IDS_ROOTLESS_USER` names an existing unprivileged account to drive
# as, and the report says which user answered. On a runner the gate already runs
# as uid 1001 and the variable is unnecessary; on a session host it is how the
# driven pass is taken. ⛔ This file creates no account: a gate row that ran
# `useradd` would be mutating the host it is checking.
#
# Usage:
#   sh scripts/acquisition/check-rootless.sh [--json]
#   BIT_IDS_ROOTLESS_USER=runnerlike sh scripts/acquisition/check-rootless.sh
#
# Exit codes: 0 every case passed, 1 a case failed, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

JSON=""
[ "${1:-}" != "--json" ] || JSON=1

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/.." && pwd)
ROOT=$(CDPATH='' cd -- "$ROOT/.." && pwd)

# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require sha256sum curl install

WORK=$(store_workdir checkrootless) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

ROOTLESS="$ROOT/scripts/acquisition/install-rootless.sh"
INSTALLER="$ROOT/scripts/acquisition/install-client.sh"
GUARD="$ROOT/scripts/acquisition/assert-disposable.sh"
ADAPTERS="$ROOT/scripts/capture/adapters"
for _f in "$ROOTLESS" "$INSTALLER" "$GUARD"; do
  [ -f "$_f" ] || {
    printf 'check-rootless: %s is missing\n' "$_f" >&2
    exit 2
  }
done

# -- how a case is run ---------------------------------------------------------
#
# ⛔ EVERY CASE RUNS THROUGH ONE FUNCTION AND THAT FUNCTION IS THE PRIVILEGE
# BOUNDARY. A case that shelled out directly would be a second way to reach the
# subject, and the second way is the one that keeps running as root.
DRIVE_USER=${BIT_IDS_ROOTLESS_USER:-}
WHOAMI=$(id -un 2>/dev/null || printf 'unknown')
DRIVE_HOME=""
if [ -n "$DRIVE_USER" ]; then
  id "$DRIVE_USER" >/dev/null 2>&1 || {
    printf 'check-rootless: BIT_IDS_ROOTLESS_USER=%s is not an account on this host\n' \
      "$DRIVE_USER" >&2
    exit 2
  }
  command -v su >/dev/null 2>&1 || {
    printf 'check-rootless: BIT_IDS_ROOTLESS_USER is set and su is not on this host\n' >&2
    exit 2
  }
  # ⛔ `HOME` IS PASSED EXPLICITLY AND THAT IS NOT TIDINESS. `su user -c` without
  # `-l` leaves the INVOKING user's HOME in the environment, so a prefix derived
  # from it would resolve under root's home - a directory the drive user cannot
  # write - and the case would fail for a reason that is not the subject. ⚠ And
  # `-l` is not the answer either: a login shell discards the assignments each
  # case depends on.
  DRIVE_HOME=$(getent passwd "$DRIVE_USER" 2>/dev/null | cut -d: -f6)
  [ -n "$DRIVE_HOME" ] || {
    printf 'check-rootless: %s has no home directory to derive a prefix from\n' \
      "$DRIVE_USER" >&2
    exit 2
  }
  # ⚠ THE WORKDIR HAS TO BE REACHABLE BY THAT USER. `store_workdir` makes a
  # directory this user owns and a `mktemp -d` is 0700, so a drive user that
  # cannot traverse it would report every case failing for a reason that is not
  # the subject.
  chmod -R a+rX "$WORK" 2>/dev/null || :
fi

# ⛔ AND THE SCRATCH THE CASES WRITE INTO IS OWNED BY THE DRIVE USER, WHICH
# TRAVERSAL ALONE DOES NOT GIVE. Measured on 2026-09-17: with the workdir merely
# readable, uid 1001 could not create the state directory, every route refused
# with `the host was never claimed`, and the run as root minutes earlier had been
# green on all nineteen. ⭐ That is the whole argument for this harness taking a
# drive user at all, arriving in the harness itself.
#
# ⚠ `chown` RATHER THAN A WORLD-WRITABLE MODE. A 0777 directory under a shared
# `/tmp` is a different problem being invented to solve this one.
SCRATCH="$WORK/drive"
mkdir -p "$SCRATCH" || exit 2
if [ -n "$DRIVE_USER" ]; then
  chown -R "$DRIVE_USER" "$SCRATCH" 2>/dev/null || {
    printf 'check-rootless: cannot give %s ownership of %s\n' "$DRIVE_USER" "$SCRATCH" >&2
    exit 2
  }
fi

# Runs a script as the drive user, or as this user when none is named. ⛔ Output
# to FILES, never a pipe: `conventions/shell.md` section 9, and the subject here
# is an installer that fetches.
#
# ⚠ THE ENVIRONMENT IS ONE STRING ON PURPOSE. It has to survive `su -c`, which
# takes a command line rather than an argument vector, so the two branches would
# otherwise pass their assignments in two different ways and only one of them
# would be the branch a runner takes.
run_as() { # script-path out err [env-string]
  _script=$1
  _out=$2
  _err=$3
  _env=${4:-}
  chmod a+rx "$_script" 2>/dev/null || :
  if [ -n "$DRIVE_USER" ]; then
    su "$DRIVE_USER" -s /bin/sh \
      -c "env HOME=$DRIVE_HOME $_env sh $_script" \
      </dev/null >"$_out" 2>"$_err"
  else
    # shellcheck disable=SC2086
    env $_env sh "$_script" </dev/null >"$_out" 2>"$_err"
  fi
}

# -- the stub vendor -----------------------------------------------------------
#
# ⛔ EACH STUB PRINTS WHAT ITS ADAPTER'S OWN PARSER READS, and the parsers differ:
# `aria2-next` anchors on the word `version` and takes the token after it,
# `qbittorrent` takes the LAST field of the first line naming the product. A stub
# that printed one shape for both would pass one case for the wrong reason.
VENDOR="$WORK/vendor"
mkdir -p "$VENDOR" || exit 2

stub_build() { # path first-line
  {
    printf '#!/bin/sh\n'
    printf 'printf %s\n' "'$2\\n'"
  } >"$1" || return 1
  chmod +x "$1"
}

stub_build "$VENDOR/aria2-next-9.9.9-linux-x86_64" 'Aria2 Next version 9.9.9' || exit 2
stub_build "$VENDOR/qbittorrent-9.9.9_x86_64.AppImage" 'qBittorrent v9.9.9' || exit 2
# ⚠ NOT A TARBALL, DELIBERATELY. aria2's release route compiles what it fetches,
# which is minutes of autotools and a C compiler - not a gate row. What this
# exercises is that the route reaches its unpack ROOTLESSLY; the refusal that
# follows is the archive's, and the case says so in its own name.
printf 'not an archive\n' >"$VENDOR/aria2-9.9.9.tar.bz2" || exit 2
printf 'not an archive\n' >"$VENDOR/transmission-9.9.9.tar.xz" || exit 2

# ⭐ THE VENDOR'S DOCUMENT IS GENERATED FROM THE STUB'S OWN BYTES, so the digest
# the installer verifies is one this harness did not type. A literal here would
# be a value in two places and would go stale the moment the stub changed.
(cd "$VENDOR" && sha256sum aria2-next-9.9.9-linux-x86_64 >checksums.sha256) || exit 2
chmod -R a+rX "$VENDOR" 2>/dev/null || :

# -- the claim, which must also need no privilege ------------------------------
#
# ⛔ `install-client` REFUSES A HOST NOTHING CLAIMED, so a rootless install is
# unreachable unless the claim guard is rootless too. That is the second half of
# `ACQ-06`'s Approach and it is exercised here rather than asserted.
STATE="$SCRATCH/state"

# -- case runner ---------------------------------------------------------------
#
# ⛔ EVERY ROUTE RUNS TWICE: once with this host's PATH and once with every
# directory holding a `sudo` removed from it. The entry's Prove asks for the two
# runs to be byte-identical, which is a stronger statement than both exiting 0 -
# a route that fell back to a different path would still exit 0 both times.
# ⛔ AND "ABSENT FROM PATH" IS A SHADOW DIRECTORY, NOT A DIRECTORY REMOVED.
# The first version of this dropped every PATH entry holding a `sudo`, which on
# this host is `/usr/bin` - so it removed `sh`, `sed`, `mktemp`, `curl` and
# everything else, and all four routes produced no output at all. ⚠ Both runs
# then "differed", which reads exactly like the finding this comparison exists to
# make and is instead the instrument destroying its own subject.
#
# ⭐ So one directory is built holding a link to every executable on PATH EXCEPT
# `sudo`, and PATH becomes that directory. That is literally the entry's wording
# - sudo absent from PATH entirely - with nothing else taken away.
SHADOW="$WORK/nosudo-bin"
mkdir -p "$SHADOW" || exit 2
_IFS=$IFS
IFS=:
for _d in $PATH; do
  if [ -z "$_d" ] || [ ! -d "$_d" ]; then continue; fi
  for _x in "$_d"/*; do
    if [ ! -f "$_x" ] || [ ! -x "$_x" ]; then continue; fi
    _n=$(basename "$_x")
    [ "$_n" != sudo ] || continue
    # ⚠ FIRST ENTRY WINS, the way PATH itself resolves. A later directory's
    # copy must not shadow an earlier one, or this would reorder the host's
    # own resolution and answer about a different set of tools.
    [ ! -e "$SHADOW/$_n" ] || continue
    ln -s "$_x" "$SHADOW/$_n" 2>/dev/null || :
  done
done
IFS=$_IFS
chmod -R a+rX "$SHADOW" 2>/dev/null || :
NOSUDO="$SHADOW"
HAD_SUDO=no
command -v sudo >/dev/null 2>&1 && HAD_SUDO=yes
SHADOW_SUDO=no
[ ! -e "$SHADOW/sudo" ] || SHADOW_SUDO=yes

# ⭐ ONE DRIVER SCRIPT, PARAMETERISED THROUGH THE ENVIRONMENT. A per-case script
# assembled with `printf` would be this repository writing shell through a shell,
# which `conventions/shell.md` section 1 is entirely about: the first draft of
# this file did exactly that and every line of it was a quoting hazard.
#
# ⛔ IT PRINTS FACTS AND ALWAYS EXITS 0. The route's own status is one of those
# facts, so a case reads `rc=` rather than this script's exit code - a driver
# that failed and a route that refused would otherwise be one number.
DRIVER="$WORK/route-driver.sh"
cat >"$DRIVER" <<'ROUTE_DRIVER'
#!/bin/sh
set -u
: "${CR_ROOTLESS:?}" "${CR_INSTALLER:?}" "${CR_GUARD:?}" "${CR_ADAPTER:?}"
: "${CR_BINARY:?}" "${CR_URL:?}"
PREFIX=$(sh "$CR_ROOTLESS" --prefix) || exit 2
rm -rf "$PREFIX"
BIT_IDS_RELEASE_URL="$CR_URL"
export BIT_IDS_RELEASE_URL
if [ -n "${CR_RESOLUTION:-}" ]; then
  BIT_IDS_RELEASE_RESOLUTION="$CR_RESOLUTION"
  export BIT_IDS_RELEASE_RESOLUTION
fi
W=$(mktemp -d) || exit 2
sh "$CR_GUARD" --claim rootless-case >/dev/null 2>&1
sh "$CR_INSTALLER" --adapter "$CR_ADAPTER" --route release \
  --workdir "$W/wd" --record "$W/rec"
_rc=$?
printf 'rc=%s\n' "$_rc"
printf 'prefix=%s\n' "$PREFIX"
if [ -x "$PREFIX/bin/$CR_BINARY" ]; then
  printf 'installed=yes\n'
else
  printf 'installed=no\n'
fi
sed -n 's/^reported_version=/version=/p' "$W/rec" 2>/dev/null
sed -n 's/^digest_source=/digest=/p' "$W/wd/rootless-install.txt" 2>/dev/null
rm -rf "$W"
exit 0
ROUTE_DRIVER

# ⛔ THE STATE DIRECTORY AND THE PREFIX ARE BOTH UNDER THIS HARNESS'S WORKDIR,
# so a case cannot claim the real host - which is the defect `ACQ-04` records
# from its own harness, where one case wrote `/var/lib/bit-ids/host-claimed` on
# a session host and every run after it failed.
CASE_ENV="BIT_IDS_STATE_DIR=$STATE BIT_IDS_PREFIX=$SCRATCH/prefix"

# ⛔ THE DISPOSITION TRAVELS AS A RESOLUTION RECORD, BECAUSE THAT IS WHAT THE REAL
# PATH PASSES. `install-step.sh` exports the resolution `resolve-release.sh`
# wrote and the installer reads it; a harness that handed the installer flags
# directly would be exercising a door the capture path does not use.
#
# ⚠ The four values are four cases. A record naming none of them is its own case
# further down.
write_resolution() { # file source [document] [asset] [sha256] [reason]
  {
    printf 'bit-ids/release-resolution/1\n'
    printf 'digest_source=%s\n' "$2"
    printf 'digest_document=%s\n' "${3:-}"
    printf 'asset=%s\n' "${4:-}"
    printf 'asset_sha256=%s\n' "${5:-}"
    printf 'digest_unpublished=%s\n' "${6:-}"
  } >"$1"
}

route_case() { # label adapter url binary [resolution-file]
  _label=$1
  _env="$CASE_ENV CR_ROOTLESS=$ROOTLESS CR_INSTALLER=$INSTALLER CR_GUARD=$GUARD"
  _env="$_env CR_ADAPTER=$ADAPTERS/$2.sh CR_URL=file://$3 CR_BINARY=$4"
  [ -z "${5:-}" ] || _env="$_env CR_RESOLUTION=$5"

  rm -rf "$SCRATCH/prefix" "$STATE"
  run_as "$DRIVER" "$WORK/$_label.a.out" "$WORK/$_label.a.err" "$_env"
  rm -rf "$SCRATCH/prefix" "$STATE"
  run_as "$DRIVER" "$WORK/$_label.b.out" "$WORK/$_label.b.err" "$_env PATH=$NOSUDO"

  # ⭐ THE TWO RUNS ARE COMPARED BYTE FOR BYTE, which is the entry's own wording.
  if cmp -s "$WORK/$_label.a.out" "$WORK/$_label.b.out"; then
    pass "sudo-free $_label: the run with sudo on PATH and the run without are identical"
  else
    fail "sudo-free $_label: the two runs differ: $(diff "$WORK/$_label.a.out" "$WORK/$_label.b.out" | head -3 | tr '\n' ' ')"
  fi
}

field() { # label key
  sed -n "s/^$2=//p" "$WORK/$1.a.out"
}

# -- 1. the prefix is not a privileged directory -------------------------------
#
# ⛔ THIS IS THE CASE THE WHOLE ENTRY TURNS ON. A prefix of `/usr/local` is what
# made every install need `sudo`; a prefix under a directory the current user
# owns is what makes the same install work on a runner, a session host and a
# laptop. ⚠ It is asked of the one derivation rather than read out of a comment.
cat >"$WORK/prefix-probe.sh" <<PREFIX_PROBE
#!/bin/sh
set -u
sh $ROOTLESS --prefix
PREFIX_PROBE
run_as "$WORK/prefix-probe.sh" "$WORK/prefix.out" "$WORK/prefix.err"
DERIVED=$(cat "$WORK/prefix.out")
case "$DERIVED" in
  /usr/local | /usr/local/* | /usr | /usr/* | /opt | /opt/*)
    fail "prefix    the derived prefix is [$DERIVED], which needs a privilege to write"
    ;;
  /*)
    pass "prefix    the derived prefix is under this user's own tree: $DERIVED"
    ;;
  *)
    fail "prefix    the derivation printed [$DERIVED], which is not an absolute path"
    ;;
esac

# ⚠ AND THE DERIVATION IS THE ONLY ONE. A second spelling anywhere in the tree is
# what `--marker` exists to prevent for the claim marker, and the same argument
# applies here: the day the prefix moves, a second speller goes on reading the
# old place.
# ⛔ AND THE NEEDLE IS ASSEMBLED RATHER THAN WRITTEN, because a harness that
# plants a pattern cannot spell it. The first version searched for the literal,
# found its own two grep lines, and reported the rule broken by the file
# enforcing it - which is the class `check-placeholders`' harness already
# records and this session reproduced by hand.
NEEDLE="BIT_IDS_PREFIX$(printf ':-')"
grep -rn -e "$NEEDLE" "$ROOT/scripts" 2>/dev/null |
  grep -v 'install-rootless.sh' >"$WORK/stray.txt" || :
STRAY=$(grep -c . "$WORK/stray.txt") || STRAY=0
if [ "$STRAY" = 0 ]; then
  pass "prefix    install-rootless is the only file that defaults the prefix"
else
  sed 's/^/          /' "$WORK/stray.txt" >&2
  fail "prefix    $STRAY other file(s) default the prefix themselves"
fi

# ⛔ AND NOTHING ON THE INSTALL PATH INVOKES `sudo` AT ALL. The cases below drive
# the path and would catch a privilege that is NEEDED; this catches one that is
# merely asked for and happens to be available, which on a runner with
# passwordless sudo is every case passing over a path that is not rootless.
# ⚠ The needle is a command-position match, so a comment naming the tool - and
# this file is full of them - is not a finding. That is the same construction
# `check-project`'s `Write-Error` rule uses, and it was written after that rule
# fired on its own enforcing file.
SUDO_NEEDLE="^[[:space:]]*(sudo|setsid[[:space:]]+sudo)[[:space:]]"
: >"$WORK/sudo-hits.txt"
for _f in "$ROOT/scripts/acquisition/install-step.sh" \
  "$ROOT/scripts/acquisition/install-client.sh" \
  "$ROOT/scripts/acquisition/install-rootless.sh" \
  "$ROOT/scripts/acquisition/assert-disposable.sh" \
  "$ADAPTERS"/*.sh; do
  [ -f "$_f" ] || continue
  grep -nE -e "$SUDO_NEEDLE" "$_f" |
    sed "s|^|$(basename "$_f"):|" >>"$WORK/sudo-hits.txt" || :
done
SUDO_HITS=$(grep -c . "$WORK/sudo-hits.txt") || SUDO_HITS=0
if [ "$SUDO_HITS" = 0 ]; then
  pass "privilege no file on the install path invokes sudo"
else
  sed 's/^/          /' "$WORK/sudo-hits.txt" >&2
  fail "privilege $SUDO_HITS invocation(s) of sudo remain on the install path"
fi

# ⭐ AND THE NEEDLE IS SHOWN TO FIRE, because a rule that has never been seen to
# refuse is a rule nobody knows works - and one whose expression stopped matching
# reports a clean tree exactly as a clean tree does.
printf '#!/bin/sh\n  sudo -E sh -c true\n' >"$WORK/needle-probe.sh"
if grep -qE -e "$SUDO_NEEDLE" "$WORK/needle-probe.sh"; then
  pass "privilege the needle refuses a planted sudo invocation"
else
  fail "privilege the needle does not match a sudo invocation at all"
fi

# -- 2. the claim guard needs no privilege ------------------------------------
# ⚠ EVERY CALL'S STDOUT IS DISCARDED EXCEPT THE VERDICT. The first version let
# `--marker` print to this script's own stdout, so the output was the marker path
# AND the word, and the case compared the pair against the word alone - a red row
# over a guard that had worked.
cat >"$WORK/claim-probe.sh" <<CLAIM_PROBE
#!/bin/sh
set -u
sh $GUARD --marker >/dev/null || exit 2
sh $GUARD --claim rootless-probe >/dev/null || exit 1
M=\$(sh $GUARD --marker) || exit 2
[ -f "\$M" ] || exit 1
printf 'claimed\n'
CLAIM_PROBE
rm -rf "$STATE"
run_as "$WORK/claim-probe.sh" "$WORK/claim.out" "$WORK/claim.err" "BIT_IDS_STATE_DIR=$STATE"
if [ "$(cat "$WORK/claim.out")" = claimed ]; then
  pass "claim     the host is claimed with no privilege at all"
else
  fail "claim     the unprivileged claim failed: $(tail -1 "$WORK/claim.err")"
fi

# ⭐ AND A SECOND CLAIM IS STILL REFUSED, which is the guard's whole subject. A
# rootless marker that stopped refusing would be a guard made convenient.
#
# ⛔ THE GUARD'S OWN WORDS ARE READ, NOT JUST THE ABSENCE OF OUTPUT. The first
# version asserted an empty stdout and passed on a run where the FIRST claim had
# failed too - a case satisfied by a different code path, which
# `docs/methodology/reviews.md` names as one of the two shapes to test for. It
# passed green while the row above it was red, which is what gave it away.
run_as "$WORK/claim-probe.sh" "$WORK/claim2.out" "$WORK/claim2.err" "BIT_IDS_STATE_DIR=$STATE"
if [ -s "$WORK/claim2.out" ]; then
  fail "claim     a second claim on one host was allowed"
elif grep -q -F -e 'this host already ran a capture' "$WORK/claim2.err"; then
  pass "claim     a second claim on one host is refused, by the marker it finds"
else
  fail "claim     the second claim failed for another reason: $(tail -1 "$WORK/claim2.err")"
fi

# -- 3. every adapter's release route, driven ---------------------------------
# ⭐ THE FOUR DISPOSITIONS ARE FOUR ROUTES HERE, so each value is reached through
# the real path rather than asserted about the installer in isolation.
STUB_SHA=$(sha256sum "$VENDOR/aria2-next-9.9.9-linux-x86_64" | cut -d' ' -f1)
write_resolution "$WORK/res-both.txt" both \
  "$VENDOR/checksums.sha256" aria2-next-9.9.9-linux-x86_64 "sha256:$STUB_SHA"
write_resolution "$WORK/res-document.txt" vendor-document \
  "$VENDOR/checksums.sha256" aria2-next-9.9.9-linux-x86_64
write_resolution "$WORK/res-listing.txt" source-listing "" "" "sha256:$(
  sha256sum "$VENDOR/qbittorrent-9.9.9_x86_64.AppImage" | cut -d' ' -f1
)"
write_resolution "$WORK/res-none.txt" unpublished "" "" "" \
  "this stub vendor publishes neither"

route_case aria2-next aria2-next "$VENDOR/aria2-next-9.9.9-linux-x86_64" aria2-next \
  "$WORK/res-both.txt"
route_case aria2-next-doc aria2-next "$VENDOR/aria2-next-9.9.9-linux-x86_64" aria2-next \
  "$WORK/res-document.txt"
route_case qbittorrent qbittorrent "$VENDOR/qbittorrent-9.9.9_x86_64.AppImage" qbittorrent-nox \
  "$WORK/res-listing.txt"
route_case aria2 aria2 "$VENDOR/aria2-9.9.9.tar.bz2" aria2c "$WORK/res-none.txt"
route_case transmission transmission "$VENDOR/transmission-9.9.9.tar.xz" transmission-daemon

# ⭐ THE ONE THAT INSTALLS AND ANSWERS. This is the Prove's own sentence: the
# route installs as an unprivileged user into a scratch prefix and the installed
# build is asked its version.
if [ "$(field aria2-next rc)" = 0 ] &&
  [ "$(field aria2-next installed)" = yes ] &&
  [ "$(field aria2-next version)" = 9.9.9 ]; then
  pass "install   aria2-next installs into the prefix and the build answers 9.9.9"
else
  sed 's/^/          /' "$WORK/aria2-next.a.err" | tail -6 >&2
  fail "install   aria2-next: rc=$(field aria2-next rc) installed=$(field aria2-next installed) version=$(field aria2-next version)"
fi

# ⭐ AND ITS DIGEST WAS VERIFIED AGAINST THE VENDOR'S DOCUMENT rather than merely
# recorded, which is the difference the installer's report exists to state.
if [ "$(field aria2-next digest)" = both ]; then
  pass "digest    aria2-next verified against the vendor's document AND the listing digest"
else
  fail "digest    aria2-next recorded digest_source=[$(field aria2-next digest)]"
fi

# ⛔ AND EACH OF THE OTHER THREE VALUES IS REACHED, because a run in which every
# route answered one of them would look the same as this one on a single case.
# ⚠ `both` above is the strongest and the others are not it; a reader told
# "verified" without being told BY WHOM has been told something weaker than it
# sounds.
while read -r label want; do
  [ -n "$label" ] || continue
  got=$(field "$label" digest)
  if [ "$got" = "$want" ]; then
    pass "digest    $label reaches digest_source=$want"
  else
    fail "digest    $label recorded digest_source=[$got], expected [$want]"
  fi
done <<DISPOSITIONS
aria2-next-doc vendor-document
qbittorrent source-listing
aria2 unpublished
DISPOSITIONS

if [ "$(field qbittorrent rc)" = 0 ] &&
  [ "$(field qbittorrent installed)" = yes ] &&
  [ "$(field qbittorrent version)" = 9.9.9 ]; then
  pass "install   qbittorrent installs into the prefix and the build answers 9.9.9"
else
  sed 's/^/          /' "$WORK/qbittorrent.a.err" | tail -6 >&2
  fail "install   qbittorrent: rc=$(field qbittorrent rc) installed=$(field qbittorrent installed) version=$(field qbittorrent version)"
fi

# ⚠ AND THE TWO ROUTES THAT COMPILE, WHICH REFUSE HERE ON PURPOSE. Each
# fetches through the installer, identifies the bytes, and then refuses on an
# archive this harness deliberately did not make one: `rc=1`, a recorded
# digest, and each one's own words.
#
# ⚠ transmission's refused BEFORE any fetch until 2026-10-14, at
# `package_format`, which named no packaging for its release route; the fetch
# below it was unreachable. The route builds now. `CLIENT-06`.
#
# ⛔ What matters in both is that each reaches its OWN refusal with no privilege,
# rather than dying on a directory it cannot write.
if [ "$(field aria2 rc)" = 1 ] && [ -n "$(field aria2 digest)" ]; then
  pass "refusal   aria2 fetched and identified rootlessly, then refused on its archive"
else
  sed 's/^/          /' "$WORK/aria2.a.err" | tail -4 >&2
  fail "refusal   aria2: rc=$(field aria2 rc) digest=$(field aria2 digest)"
fi

if [ "$(field transmission rc)" = 1 ] && [ -n "$(field transmission digest)" ] &&
  grep -q -F -e "the release route's archive did not unpack" "$WORK/transmission.a.err"; then
  pass "refusal   transmission fetched and identified rootlessly, then refused on its archive"
else
  sed 's/^/          /' "$WORK/transmission.a.err" | tail -4 >&2
  fail "refusal   transmission: rc=$(field transmission rc), and not for its own reason"
fi

# -- 3b. the resolution reader, which is where four adapters' decision now lives
#
# ⛔ A VOCABULARY THAT GREW ON ONE SIDE AND NOT THE OTHER IS THE FAILURE THIS
# READER CAN HAVE. `resolve-release.sh` writes `digest_source` and this reads it;
# a value one knows and the other does not must be could-not-run, never an
# install under a disposition nobody implemented.
cat >"$WORK/unknown-res.txt" <<'UNKNOWN'
digest_source=quantum-notary
UNKNOWN
cat >"$WORK/reader-probe.sh" <<READER
#!/bin/sh
set -u
sh $ROOTLESS --fetch --url "file://\$CR_URL" --into "\$CR_INTO" \\
  --from-resolution "\$CR_RES"
READER
run_as "$WORK/reader-probe.sh" "$WORK/unknown.out" "$WORK/unknown.err" \
  "$CASE_ENV CR_URL=$VENDOR/aria2-next-9.9.9-linux-x86_64 CR_INTO=$SCRATCH/probe1 CR_RES=$WORK/unknown-res.txt"
rc=$?
if [ "$rc" = 2 ] &&
  grep -q -F -e 'which this installer does not know' "$WORK/unknown.err"; then
  pass "reader    a digest_source the installer does not know is could-not-run"
else
  fail "reader    the unknown disposition exited $rc: $(tail -1 "$WORK/unknown.err")"
fi

# ⭐ AND THE LISTING DIGEST IS REALLY COMPARED. Without this every
# `source-listing` row above passes equally over a reader that took the value and
# never checked it - which is the shape a disposition record makes easy, because
# the record itself looks right either way.
cat >"$WORK/wrong-res.txt" <<'WRONG'
digest_source=source-listing
asset_sha256=sha256:0000000000000000000000000000000000000000000000000000000000000000
WRONG
run_as "$WORK/reader-probe.sh" "$WORK/wrong.out" "$WORK/wrong.err" \
  "$CASE_ENV CR_URL=$VENDOR/aria2-next-9.9.9-linux-x86_64 CR_INTO=$SCRATCH/probe2 CR_RES=$WORK/wrong-res.txt"
rc=$?
if [ "$rc" = 1 ] && grep -q -F -e 'and the source published' "$WORK/wrong.err"; then
  pass "reader    a listing digest that does not match the bytes is refused"
else
  fail "reader    the mismatched listing digest exited $rc: $(tail -1 "$WORK/wrong.err")"
fi

# -- 4. the controls ----------------------------------------------------------
#
# ⛔ WITHOUT THIS, EVERY CASE ABOVE PASSES OVER A HOST THAT SIMPLY HAS NO `sudo`.
# The two runs being identical means nothing if the first one had no `sudo` on
# PATH either, so the harness says which host it was on and refuses to claim the
# comparison when there was nothing to remove.
if [ "$HAD_SUDO" = yes ] && [ "$SHADOW_SUDO" = no ]; then
  pass "control   sudo IS on this host's PATH and is NOT in the shadow one"
elif [ "$HAD_SUDO" = no ]; then
  fail "control   sudo is not on this host, so the sudo-free comparison established nothing"
else
  fail "control   the shadow PATH still carries a sudo, so nothing was removed"
fi

# ⭐ AND THE SHADOW PATH IS OTHERWISE WHOLE. Without this the row above passes
# over a directory holding nothing at all, which also has no `sudo` in it - and
# that was the first version's actual defect, arriving from the other side.
MISSING=""
for _need in sh sed grep curl sha256sum install mktemp awk; do
  [ -e "$SHADOW/$_need" ] || MISSING="${MISSING:+$MISSING }$_need"
done
if [ -z "$MISSING" ]; then
  pass "control   the shadow PATH still resolves every tool the routes need"
else
  fail "control   the shadow PATH is missing: $MISSING"
fi

# ⭐ AND THE BUILD IS FOUND IN THE PREFIX RATHER THAN ON PATH. An adapter whose
# `binary()` fell through to `command -v` would answer about whatever the host
# already had, which is the two-routes-one-binary defect `ACQ-03` records.
#
# ⚠ THE INSTALL IS REDONE FIRST, because every route case clears the prefix
# before it runs and the LAST of them installed nothing. The first version of
# this case asked after transmission's refusal and read an empty prefix as an
# adapter that could not find its build - a red row about the harness's own
# ordering.
route_case aria2-next-again aria2-next "$VENDOR/aria2-next-9.9.9-linux-x86_64" aria2-next \
  "$WORK/res-both.txt"
cat >"$WORK/where.sh" <<WHERE
#!/bin/sh
set -u
sh $ADAPTERS/aria2-next.sh describe | sed -n 's/^binary=/found=/p'
WHERE
run_as "$WORK/where.sh" "$WORK/where.out" "$WORK/where.err" "$CASE_ENV"
FOUND=$(sed -n 's/^found=//p' "$WORK/where.out")
case "$FOUND" in
  "$SCRATCH/prefix/bin/aria2-next")
    pass "control   the adapter finds the build in the prefix, not on PATH"
    ;;
  "")
    fail "control   the adapter found no build after the install"
    ;;
  *)
    fail "control   the adapter found [$FOUND], which is not the prefix this run installed into"
    ;;
esac

# ⚠ WHICH USER DROVE THIS, PRINTED RATHER THAN IMPLIED. A root run of a rootless
# install exercises the same code and cannot establish that an unprivileged user
# could have done it, and a reader of a green report is owed that difference.
if [ -n "$DRIVE_USER" ]; then
  pass "driven    as $DRIVE_USER (uid $(id -u "$DRIVE_USER")), dropped from $WHOAMI"
elif [ "$(id -u)" = 0 ]; then
  pass "driven    as root, so this run says nothing about an unprivileged user; set BIT_IDS_ROOTLESS_USER"
else
  pass "driven    as $WHOAMI (uid $(id -u)), which is already unprivileged"
fi

store_report check-rootless/1 cases "$JSON"
