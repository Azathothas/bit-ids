#!/bin/sh
# check-transmission.sh - drive Transmission's two compiling routes through the
# real adapter, with the build tools stubbed, and plant each defect its guards
# exist to refuse.
#
# ⭐ THE ADAPTER IS THE REAL ONE AND SO IS EVERYTHING BUT THE BUILD. The release
# route fetches a real `.tar.xz` through `bit-check install-rootless`, the headers come
# out of a real `.deb` that `dpkg-deb` packs here and unpacks there, and the
# runtime check reads this host's own library directory. ⚠ `apt-get`, `cmake`
# and `git` are stubs: a real build is 1013 seconds on the release route and
# 1078 on the source one, measured on 2026-10-14, and a harness that ran them
# would be one nobody runs. `CLIENT-06` carries the driven builds.
#
# ⛔ EVERY REFUSAL ALSO CHECKS THAT NOTHING WAS INSTALLED. A route that refuses
# and leaves a daemon in the prefix is a route the next lane would measure.
#
# Usage:
#   sh scripts/capture/check-transmission.sh
#   sh scripts/capture/check-transmission.sh --json
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
      printf 'check-transmission: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# shellcheck disable=SC2034
ME=check-transmission
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

store_require dpkg-deb tar xz sha256sum ninja c++ jq
# ⭐ `bit-check adapter transmission` since 2026-10-15, `CI-11`.
ADAPTER=transmission

WORK=$(store_workdir checktransmission) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

# ⚠ The adapter derives its prefix through `bit-check install-rootless` since
# 2026-10-15, `CI-11`, and finds it through BIT_CHECK, as the workflow exports it.
mkdir -p "$WORK/bin" || exit 2
BIT_CHECK=$(store_bitcheck "$ROOT" "$WORK/bin") || exit 2
export BIT_CHECK

# ⛔ The runtime the fixture headers name is whichever libcurl this host has, so
# the clean case passes on any host that carries one and the plant below names
# a file no host does.
RUNTIME=$(find /usr/lib /lib -name 'libcurl.so.4.*' -type f 2>/dev/null | head -1)
[ -n "$RUNTIME" ] || {
  printf 'check-transmission: this host has no libcurl runtime to build against\n' >&2
  exit 2
}
RUNDIR=${RUNTIME%/*}

# -- the fixtures ----------------------------------------------------------------
deb() { # out link-target [no-header]
  _t="$WORK/debroot"
  rm -rf "$_t"
  mkdir -p "$_t/DEBIAN" "$_t$RUNDIR" "$_t/usr/include/x86_64-linux-gnu/curl" || return 1
  printf 'Package: libcurl4-openssl-dev\nVersion: 9.9.9-fixture\nArchitecture: amd64\nMaintainer: fixture <fixture@invalid>\nDescription: fixture\n' \
    >"$_t/DEBIAN/control"
  [ -n "${3:-}" ] || printf '/* fixture */\n' >"$_t/usr/include/x86_64-linux-gnu/curl/curl.h"
  ln -s "$2" "$_t$RUNDIR/libcurl.so"
  dpkg-deb --root-owner-group -b "$_t" "$1" >/dev/null 2>&1
}
deb "$WORK/good.deb" "${RUNTIME##*/}" || exit 2
deb "$WORK/stale.deb" libcurl.so.4.999.0 || exit 2
deb "$WORK/headless.deb" "${RUNTIME##*/}" no-header || exit 2

# A source tree as a release archive carries it.
mkdir -p "$WORK/tree/transmission-9.9.9" || exit 2
printf 'project(transmission)\n' >"$WORK/tree/transmission-9.9.9/CMakeLists.txt"
tar -cJf "$WORK/transmission-9.9.9.tar.xz" -C "$WORK/tree" transmission-9.9.9 || exit 2
mkdir -p "$WORK/tree2/a" "$WORK/tree2/b" || exit 2
printf 'x\n' >"$WORK/tree2/a/CMakeLists.txt"
printf 'x\n' >"$WORK/tree2/b/CMakeLists.txt"
tar -cJf "$WORK/twotrees.tar.xz" -C "$WORK/tree2" a b || exit 2

# ⚠ Generated rather than written out: a forty-digit object name is the shape
# `check-no-secrets --public` refuses in this tree.
COMMIT=$(printf 'fixture-commit' | sha256sum | cut -c1-40)

# -- the stubs -------------------------------------------------------------------
# ⚠ Each reads its behaviour from the environment, so one PATH serves every case.
STUBS="$WORK/stubs"
mkdir -p "$STUBS" || exit 2
cat >"$STUBS/apt-get" <<'APT'
#!/bin/sh
for a in "$@"; do last=$a; done
case " $* " in
  *" update "*) exit 0 ;;
  *" download "*)
    cp "$STUB_DEB" "./${last}_9.9.9-fixture_amd64.deb" || exit 100
    [ -z "${STUB_SECOND_DEB:-}" ] || cp "$STUB_DEB" "./${last}_9.9.8-fixture_amd64.deb"
    exit 0
    ;;
esac
exit 100
APT
cat >"$STUBS/cmake" <<'CMAKE'
#!/bin/sh
printf '%s\n' "$*" >>"$STUB_LOG"
if [ "$1" = --build ]; then
  for exe in daemon/transmission-daemon utils/transmission-remote; do
    [ "$exe" != "${STUB_SKIP:-}" ] || continue
    mkdir -p "$2/${exe%/*}"
    printf '#!/bin/sh\necho "%s 9.9.9 (stub)" >&2\n' "${exe#*/}" >"$2/$exe"
    chmod +x "$2/$exe"
  done
fi
exit 0
CMAKE
cat >"$STUBS/git" <<'GIT'
#!/bin/sh
printf 'git %s\n' "$*" >>"$STUB_LOG"
case "$1" in
  clone)
    for a in "$@"; do dest=$a; done
    mkdir -p "$dest" && printf 'project(transmission)\n' >"$dest/CMakeLists.txt"
    ;;
  -C) printf '%s\n' "$STUB_COMMIT" ;;
esac
GIT
chmod +x "$STUBS/apt-get" "$STUBS/cmake" "$STUBS/git" || exit 2

# ⛔ Unpiped. Every case starts from an empty prefix and an empty log.
run_route() { # name route deb [env...]
  _name=$1
  _route=$2
  _deb=$3
  shift 3
  rm -rf "$WORK/prefix" "$WORK/wd-$_name"
  : >"$WORK/$_name.log"
  env PATH="$STUBS:$PATH" BIT_IDS_PREFIX="$WORK/prefix" STUB_DEB="$_deb" \
    STUB_LOG="$WORK/$_name.log" STUB_COMMIT="$COMMIT" "$@" \
    "$BIT_CHECK" adapter "$ADAPTER" install "$_route" "$WORK/wd-$_name" \
    >"$WORK/$_name.out" 2>"$WORK/$_name.err" </dev/null
  RC=$?
}
release_env="BIT_IDS_RELEASE_URL=file://$WORK/transmission-9.9.9.tar.xz"
nothing_installed() {
  [ ! -e "$WORK/prefix/bin/transmission-daemon" ] && [ ! -e "$WORK/prefix/bin/transmission-remote" ]
}
refused() { # name literal case-name
  if [ "$RC" = 1 ] && grep -q -F -e "$2" "$WORK/$1.err" && nothing_installed; then
    pass "$3"
  else
    fail "$3 (exit $RC, installed: $(find "$WORK/prefix/bin" -type f 2>/dev/null | tr '\n' ' '), $(tail -1 "$WORK/$1.err"))"
  fi
}

# -- the two routes, clean -------------------------------------------------------
run_route release release "$WORK/good.deb" "$release_env"
if [ "$RC" = 0 ] && [ -x "$WORK/prefix/bin/transmission-daemon" ] &&
  [ -x "$WORK/prefix/bin/transmission-remote" ] &&
  [ "$(cat "$WORK/wd-release/package")" = elf-binary ]; then
  pass "release  the tarball is fetched, unpacked and built into the prefix"
else
  fail "release  exit $RC: $(tail -2 "$WORK/release.err" | tr '\n' ' ')"
fi
if [ "$RC" = 0 ] && [ "$(env BIT_IDS_PREFIX="$WORK/prefix" "$BIT_CHECK" adapter "$ADAPTER" version 2>/dev/null)" = 9.9.9 ]; then
  pass "release  the build in the prefix is the one the adapter asks its version"
else
  fail "release  the adapter did not find the prefix's build"
fi
if grep -q "^runtime=$RUNTIME$" "$WORK/wd-release/build-deps.txt" 2>/dev/null &&
  grep -q '^package=libcurl4-openssl-dev$' "$WORK/wd-release/build-deps.txt" &&
  grep -q -F "'-DCURL_INCLUDE_DIR=$WORK/wd-release/deps/root/usr/include/x86_64-linux-gnu'" \
    "$WORK/wd-release/build-configure.txt" &&
  grep -q -F -e "-DCURL_LIBRARY=$RUNTIME" "$WORK/release.log"; then
  pass "release  the headers' package and runtime are recorded, and the build is pointed at both"
else
  fail "release  build-deps.txt or the configure line does not carry the headers and runtime"
fi

run_route source source "$WORK/good.deb" BIT_IDS_SOURCE_TAG=9.9.9
if [ "$RC" = 0 ] && [ -x "$WORK/prefix/bin/transmission-daemon" ] &&
  [ "$(cat "$WORK/wd-source/source-commit")" = "$COMMIT" ]; then
  pass "source   the tag is cloned, its commit named, and built into the prefix"
else
  fail "source   exit $RC: $(tail -2 "$WORK/source.err" | tr '\n' ' ')"
fi
if grep -q -e '--recurse-submodules' "$WORK/source.log" &&
  grep -q -e '--branch 9.9.9' "$WORK/source.log"; then
  pass "source   the clone carries the tag and its submodules"
else
  fail "source   the clone was: $(grep '^git clone' "$WORK/source.log")"
fi

if env BIT_IDS_PREFIX="$WORK/none" "$BIT_CHECK" adapter "$ADAPTER" describe | grep -q '^compiles=release,source$'; then
  pass "describe both compiling routes are declared, so the workflow bounds them as builds"
else
  fail "describe the adapter does not declare compiles=release,source"
fi

# -- what is refused -------------------------------------------------------------
run_route stale release "$WORK/stale.deb" "$release_env"
refused stale "which this host does not have" \
  "headers  headers for a libcurl this host does not run are refused"

run_route headless release "$WORK/headless.deb" "$release_env"
refused headless "carried no curl/curl.h" \
  "headers  a package with no curl.h is refused"

run_route twodebs release "$WORK/good.deb" "$release_env" STUB_SECOND_DEB=1
refused twodebs "no single libcurl4-openssl-dev package" \
  "headers  two downloaded packages are refused rather than one picked"

run_route noremote release "$WORK/good.deb" "$release_env" STUB_SKIP=utils/transmission-remote
refused noremote "built nothing at build/utils/transmission-remote" \
  "build    a build with no transmission-remote installs nothing"

run_route twotrees release "$WORK/good.deb" "BIT_IDS_RELEASE_URL=file://$WORK/twotrees.tar.xz"
refused twotrees "did not unpack into one source tree" \
  "release  an archive holding two source trees is refused"

run_route notag source "$WORK/good.deb"
if [ "$RC" = 2 ] && grep -q -F 'needs BIT_IDS_SOURCE_TAG' "$WORK/notag.err" && nothing_installed; then
  pass "source   a source route with no resolved tag cannot run"
else
  fail "source   exit $RC: $(tail -1 "$WORK/notag.err")"
fi

# -- stop forgets the RPC verifier the daemon writes on exit ---------------------
#
# ⛔ MEASURED ON `capture-client` RUN 38600252161: 4.1.3 saves a salted
# `rpc-password` into its settings when it exits, with authentication off, and
# rule 12's scan refused the bundle for it. This stub daemon does the same on
# SIGTERM, which is the signal `stop` sends.
cat >"$WORK/fake-daemon.sh" <<'FAKE'
#!/bin/sh
trap 'printf "{\n  \"dht-enabled\": false,\n  \"rpc-password\": \"{0123456789abcdef0123456789abcdef01234567saltsalt\",\n  \"rpc-port\": 9091\n}\n" >"$1/settings.json"; exit 0' TERM
while :; do sleep 1; done
FAKE
stop_case() { # name path-for-stop
  _wd="$WORK/stop-$1"
  rm -rf "$_wd"
  mkdir -p "$_wd/config" || return 1
  printf '{"rpc-port": 9091}\n' >"$_wd/config/settings.json"
  sh "$WORK/fake-daemon.sh" "$_wd/config" &
  printf '%s\n' "$!" >"$_wd/pid"
  sleep 1
  env PATH="$2" "$BIT_CHECK" adapter "$ADAPTER" stop "$_wd" >"$WORK/stop-$1.out" 2>&1
  RC=$?
}
stop_case jq "$PATH"
if [ "$RC" = 0 ] && [ -f "$WORK/stop-jq/config/settings.json" ] &&
  ! grep -q rpc-password "$WORK/stop-jq/config/settings.json" &&
  jq -e '."rpc-port" == 9091' "$WORK/stop-jq/config/settings.json" >/dev/null 2>&1; then
  pass "stop     the verifier the daemon wrote on exit is deleted and the rest is kept"
else
  fail "stop     exit $RC; settings: $(tr '\n' ' ' <"$WORK/stop-jq/config/settings.json" 2>/dev/null)"
fi

# ⚠ A PATH with every tool the stop needs and no `jq`.
NOJQ="$WORK/nojq-bin"
mkdir -p "$NOJQ" || exit 2
for _t in sh cat kill sleep rm mv command; do
  _p=$(command -v "$_t" 2>/dev/null) || continue
  case "$_p" in /*) ln -s "$_p" "$NOJQ/$_t" 2>/dev/null || : ;; esac
done
stop_case nojq "$NOJQ"
if [ "$RC" = 0 ] && [ ! -e "$WORK/stop-nojq/config/settings.json" ]; then
  pass "stop     with no jq the whole settings file goes rather than a line edit"
else
  fail "stop     exit $RC with no jq, and the settings file is still there"
fi

store_report check-transmission/1 cases "$JSON"
