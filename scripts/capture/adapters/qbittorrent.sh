#!/bin/sh
# qbittorrent.sh - the CLIENT-01 capture adapter.
#
# ⚠ THE HIGHEST-PRIORITY TARGET AND NOT THE SIMPLEST ONE. qbittorrent-nox keeps
# a profile directory, asks once for a legal notice to be accepted, and reads
# its switches out of an INI file rather than off its command line. Each of
# those is a place a run can differ from the previous one, which is what a
# disposable host exists to remove, so this adapter writes the whole profile
# itself instead of inheriting whatever is in HOME.
#
# -- ⭐ DRIVEN ON 2026-10-15, BOTH ROUTES, 5.2.4 ----------------------------------
#
# The release route installs the `_lt20` AppImage and the source route builds
# `release-5.2.4` against Qt 6.11.2's archives and libtorrent 2.0.15. Each
# answered `qBittorrent v5.2.4` and captured three sessions in the namespace,
# under Xvfb: six announces and six peer streams, every peer ID `-qB5240-`. The
# two records assembled `publishable` and `build_equivalent`. `CLIENT-01`.
#
# -- ⛔ WHAT THIS ASSUMED ABOUT THE PRODUCT BEFORE THAT ------------------------
#
# Kept for the record. Each line was a claim a dispatch settled:
#
#   * `qbittorrent-nox --version` ends its first line with the version, with a
#     leading `v` this strips;
#   * `--profile` roots the whole configuration, so a written profile is the one
#     that is read;
#   * ⛔ **REFUTED on 2026-09-08.** `--confirm-legal-notice` was assumed to be
#     what stops a fresh profile blocking on a prompt. This build answers
#     `Bad command line: --confirm-legal-notice is an unknown command line
#     parameter`, so it was never a control; the acceptance is written into the
#     profile as `[LegalNotice] Accepted=true` instead, which is the next
#     assumption and is not measured either;
#   * the `Session\DHTEnabled`, `Session\PeXEnabled` and `Session\LSDEnabled`
#     keys under `[BitTorrent]` are the three adjacent surfaces;
#   * a torrent as a positional argument is added and started.
#
# ⚠ The prompt claim is the one worth watching. If it is wrong the capture does
# not lie, it stalls: the observer serves its whole deadline, nothing announces,
# and `capture-client` refuses over a build that put no bytes on the wire.
#
# See adapters/README.md for the contract.
#
# Exit codes: 0 done, 1 refused, 2 could not run.

set -u

# ⛔ AN AppImage MOUNTS ITSELF WITH FUSE, AND A CAPTURE HOST NEED NOT HAVE IT.
# This makes the AppImage extract itself and run instead, which costs an
# extraction per call and works on a host with FUSE as well as one without. ⚠ It
# is exported here rather than at each call site so `version`, `start` and any
# later subcommand inherit one answer; a build that is not an AppImage ignores
# it. ⛔ Neither branch has been driven: nothing has installed through the
# release route.
export APPIMAGE_EXTRACT_AND_RUN=1

ME=qbittorrent

refuse() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 1
}

cannot() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 2
}

# ⛔ THE PREFIX IS ASKED FOR, NEVER COMPOSED. `install-rootless.sh --prefix` is
# the one derivation in this tree, so this file cannot disagree with the
# installer about where a build went. ⚠ It defaulted to `/usr/local` until
# `ACQ-06`, which is a directory a runner needs a privilege to write.
ROOTLESS=$(CDPATH='' cd -- "$(dirname -- "$0")/../../acquisition" && pwd)/install-rootless.sh
prefix() {
  [ -f "$ROOTLESS" ] || cannot "$ROOTLESS is not present"
  sh "$ROOTLESS" --prefix
}

# ⛔ THE GUI BUILD, `qbittorrent`, AND NOT `qbittorrent-nox`. Measured on
# 2026-10-15 by reading both 5.2.4 AppImages' squashfs: neither carries
# `qbittorrent-nox`. The operator settled that both routes run the GUI build
# under Xvfb, so the two routes are one program. `CLIENT-01`.
EXE=qbittorrent

binary() {
  if [ -n "${BIT_IDS_QBITTORRENT:-}" ]; then
    printf '%s' "$BIT_IDS_QBITTORRENT"
    return 0
  fi
  # ⭐ THE ROUTE'S OWN PREFIX, BEFORE THE PATH. Without it the two routes would
  # race on PATH order, which is a property of the host rather than of the
  # acquisition.
  _prefix=$(prefix) || return 1
  if [ -x "$_prefix/bin/$EXE" ]; then
    printf '%s' "$_prefix/bin/$EXE"
    return 0
  fi
  command -v "$EXE" 2>/dev/null
}

# -- ⭐ WHAT BOTH ROUTES RUN AGAINST, FROM THE DISTRIBUTION'S SIGNED INDEX -----
#
# ⛔ AN AppImage LEAVES THE HOST'S GRAPHICS STACK OUT ON PURPOSE, and the Qt
# archives the source route builds against assume it. Measured on 2026-10-15:
# the `_lt20` AppImage stops on a missing `libEGL.so.1` before printing
# anything, and Qt's xcb plugin names `libxcb-cursor0` and seven more that a
# runner need not carry. So one set of noble's runtime packages is unpacked into
# the prefix for BOTH routes, which then load the same host libraries rather
# than whatever each runner happens to have.
# ⚠ Rootless: `apt-get` keeps its index and cache in the workdir and
# `dpkg-deb -x` unpacks, which is `transmission`'s technique for libcurl's
# headers. `ACQ-06` is why nothing here may escalate.
RUNTIME_PACKAGES="libegl1 libgl1 libglx0 libopengl0 libglvnd0 libgles2 \
libxcb-cursor0 libxcb-icccm4 libxcb-image0 libxcb-keysyms1 libxcb-render-util0 \
libxcb-util1 libxcb-xkb1 libxkbcommon0 libxkbcommon-x11-0"

runtime_dir() {
  _prefix=$(prefix) || return 1
  printf '%s/lib/qbittorrent-runtime' "$_prefix"
}

# Fetches packages through a private index and unpacks them into <root>,
# recording each package, version and digest in <record>.
apt_unpack() { # workdir root record package...
  _wd=$1
  _root=$2
  _record=$3
  shift 3
  for _need in apt-get dpkg-deb; do
    command -v "$_need" >/dev/null 2>&1 || cannot "the install needs $_need and it is not on this host"
  done
  _apt="$_wd/apt"
  mkdir -p "$_apt/lists/partial" "$_apt/cache/archives/partial" "$_apt/deb" "$_root" ||
    cannot "cannot create $_apt"
  if [ ! -f "$_apt/updated" ]; then
    timeout 300 apt-get -o "Dir::State::Lists=$_apt/lists" -o "Dir::Cache=$_apt/cache" \
      -o Debug::NoLocking=1 update </dev/null >>"$_wd/deps.log" 2>&1 ||
      refuse "a private package index could not be fetched"
    : >"$_apt/updated"
  fi
  (cd "$_apt/deb" && timeout 300 apt-get -o "Dir::State::Lists=$_apt/lists" \
    -o "Dir::Cache=$_apt/cache" -o Debug::NoLocking=1 download "$@") \
    </dev/null >>"$_wd/deps.log" 2>&1 || refuse "the packages could not be downloaded: $*"
  for _pkg in "$@"; do
    set -- "$_apt/deb/${_pkg}_"*.deb
    if [ $# != 1 ] || [ ! -f "$1" ]; then
      refuse "the download left no single $_pkg package"
    fi
    dpkg-deb -x "$1" "$_root" || refuse "$_pkg could not be unpacked"
    printf '%s=%s sha256:%s\n' "$(dpkg-deb -f "$1" Package)" "$(dpkg-deb -f "$1" Version)" \
      "$(sha256sum "$1" | cut -d' ' -f1)" >>"$_record" ||
      refuse "the install could not record $_pkg"
  done
}

# The runtime set, into the prefix. Both routes call this.
install_runtime() { # workdir
  _rt=$(runtime_dir) || cannot "the rootless prefix could not be derived"
  # shellcheck disable=SC2086 # the package list is split on purpose
  apt_unpack "$1" "$1/runtime-root" "$1/runtime-packages.txt" $RUNTIME_PACKAGES
  rm -rf "$_rt.new" || refuse "cannot clear $_rt.new"
  mkdir -p "$_rt.new" || refuse "cannot stage $_rt"
  cp -P "$1/runtime-root/usr/lib/x86_64-linux-gnu/"*.so* "$_rt.new/" ||
    refuse "the runtime set carried no libraries"
  rm -rf "$_rt" || refuse "cannot clear $_rt"
  mv "$_rt.new" "$_rt" || refuse "cannot install $_rt"
}

# ⭐ THE DISPLAY THE GUI BUILD NEEDS, and only that. Both AppImages carry the
# xcb and wayland platform plugins and nothing headless, so the build cannot
# start without a server. ⚠ Xvfb listens on a filesystem socket only, which a
# network namespace leaves working, and on no TCP port.
# Starts one, writes its pid to <pidfile>, and prints the display.
start_display() { # pidfile
  command -v Xvfb >/dev/null 2>&1 || cannot "the GUI build needs Xvfb and it is not on this host"
  _n=91
  while [ "$_n" -lt 120 ]; do
    if [ ! -e "/tmp/.X11-unix/X$_n" ] && [ ! -e "/tmp/.X$_n-lock" ]; then
      Xvfb ":$_n" -nolisten tcp -screen 0 1280x800x24 </dev/null >/dev/null 2>&1 &
      printf '%s\n' "$!" >"$1"
      _waited=0
      while [ "$_waited" -lt 50 ]; do
        [ -S "/tmp/.X11-unix/X$_n" ] && {
          printf ':%s' "$_n"
          return 0
        }
        sleep 0.1
        _waited=$((_waited + 1))
      done
      kill "$(cat "$1")" 2>/dev/null
      cannot "Xvfb :$_n did not open its socket within five seconds"
    fi
    _n=$((_n + 1))
  done
  cannot "no free X display between :91 and :119"
}

stop_display() { # pidfile
  [ -f "$1" ] || return 0
  _xpid=$(cat "$1")
  case "$_xpid" in
    '' | *[!0-9]*) return 0 ;;
  esac
  kill "$_xpid" 2>/dev/null
  rm -f "$1"
}

# ⛔ ONE ENVIRONMENT FOR EVERY CALL INTO THE BUILD, so `version` and `start`
# run the same program the same way on both routes.
# ⛔ THE RUNTIME DIRECTORY IS AN ARGUMENT, resolved by the caller before it
# changes anything. The prefix follows `HOME`, and `start` runs the build with
# `HOME` set to its workdir; resolved in here, the directory was the workdir's,
# which holds no libraries, and every session stopped on a missing
# `libEGL.so.1`. Driven on 2026-10-15.
run_build() { # display runtime-dir command...
  _display=$1
  _rt=$2
  shift 2
  DISPLAY=$_display QT_QPA_PLATFORM=xcb LD_LIBRARY_PATH="$_rt" "$@"
}

# -- ⭐ THE SOURCE ROUTE: THE TAG, BUILT AGAINST QT'S OWN PREBUILT ARCHIVES ------
#
# Settled by the operator on 2026-10-15: Qt 6.11.2's `linux_gcc_64` archives
# from `download.qt.io`, libtorrent 2.0.15 built from its own tag so both routes
# run the libtorrent the `_lt20` AppImage carries, and noble's Boost headers.
# ⛔ QT'S ARCHIVES ARE NAMED BY QT'S OWN LISTING, never composed. `Updates.xml`
# names the package's version and its archives, and each archive's `.sha1`
# sidecar is checked before it is unpacked. ⚠ That sidecar is the host's
# statement about bytes the same host serves, as a release listing's digest is.
QT_REPO=https://download.qt.io/online/qtsdkrepository/linux_x64/desktop/qt6_6112/qt6_6112
QT_PACKAGE=qt.qt6.6112.linux_gcc_64
# ⚠ Only what the build reads: qtbase, qtsvg, qttools for `lrelease`, which
# itself loads qtdeclarative's `libQt6Qml`, and Qt's own ICU.
QT_MODULES="qtbase qtsvg qttools qtdeclarative icu"
LIBTORRENT_TAG=v2.0.15
BUILD_PACKAGES="7zip libboost1.83-dev libgl-dev libglx-dev libopengl-dev libegl-dev \
libgles-dev libglvnd-dev libglvnd-core-dev mesa-common-dev libxkbcommon-dev"

fetch_qt() { # workdir qt-dir
  _wd=$1
  _qt=$2
  command -v python3 >/dev/null 2>&1 || cannot "reading Qt's listing needs python3"
  timeout 120 curl -fsSL --retry 2 -o "$_wd/qt-Updates.xml" "$QT_REPO/Updates.xml" \
    </dev/null 2>>"$_wd/install.log" || refuse "Qt's listing could not be fetched"
  python3 -I -c '
import sys, xml.etree.ElementTree as ET
root = ET.parse(sys.argv[1]).getroot()
for p in root.findall("PackageUpdate"):
    if p.findtext("Name") == sys.argv[2]:
        print("version", p.findtext("Version"))
        for a in (p.findtext("DownloadableArchives") or "").split(","):
            print("archive", a.strip())
' "$_wd/qt-Updates.xml" "$QT_PACKAGE" >"$_wd/qt-listing.txt" || refuse "Qt's listing could not be read"
  _ver=$(awk '$1 == "version" { print $2; exit }' "$_wd/qt-listing.txt")
  [ -n "$_ver" ] || refuse "Qt's listing names no $QT_PACKAGE"
  rm -rf "$_qt" || refuse "cannot clear $_qt"
  mkdir -p "$_qt" "$_wd/qt-archives" || refuse "cannot create $_qt"
  for _module in $QT_MODULES; do
    _archive=$(awk -v m="$_module" '$1 == "archive" && index($2, m "-") == 1 { print $2; exit }' "$_wd/qt-listing.txt")
    [ -n "$_archive" ] || refuse "Qt's listing names no $_module archive for $QT_PACKAGE"
    _file="$_wd/qt-archives/$_ver$_archive"
    for _suffix in "" .sha1; do
      timeout 300 curl -fsSL --retry 2 -o "$_file$_suffix" "$QT_REPO/$QT_PACKAGE/$_ver$_archive$_suffix" \
        </dev/null 2>>"$_wd/install.log" || refuse "Qt's $_module archive could not be fetched"
    done
    _want=$(cut -c1-40 "$_file.sha1")
    _got=$(sha1sum "$_file" | cut -c1-40)
    [ "$_want" = "$_got" ] || refuse "Qt's $_module archive digests to $_got and its sidecar says $_want"
    "$_wd/build-root/usr/lib/7zip/7z" x -y -bd "-o$_qt" "$_file" </dev/null >>"$_wd/install.log" 2>&1 ||
      refuse "Qt's $_module archive could not be unpacked"
    printf 'qt %s %s sha1:%s\n' "$_ver" "$_archive" "$_got" >>"$_wd/build-deps.txt"
  done
  # ⚠ Qt's ICU archive unpacks at the root, and Qt's libraries find it beside
  # themselves through their own `$ORIGIN` run path.
  for _icu in "$_qt"/libicu*.so*; do
    [ -e "$_icu" ] || continue
    mv "$_icu" "$_qt/lib/" || refuse "Qt's ICU could not be placed beside Qt's libraries"
  done
}

build_source() { # workdir qbittorrent-src
  _wd=$1
  _src=$2
  for _need in cmake ninja c++ git curl sha1sum; do
    command -v "$_need" >/dev/null 2>&1 || cannot "the source route compiles and $_need is not on this host"
  done
  PREFIX=$(prefix) || cannot "the rootless prefix could not be derived"
  mkdir -p "$PREFIX/bin" "$PREFIX/opt" ||
    refuse "this user cannot create $PREFIX/bin; a rootless install writes nowhere else"
  : >"$_wd/build-deps.txt"
  # shellcheck disable=SC2086 # the package list is split on purpose
  apt_unpack "$_wd" "$_wd/build-root" "$_wd/build-deps.txt" $BUILD_PACKAGES
  # ⚠ The runtime set is unpacked into the build root too, so the build links
  # against the libraries it will load.
  # shellcheck disable=SC2086
  apt_unpack "$_wd" "$_wd/build-root" "$_wd/build-deps.txt" $RUNTIME_PACKAGES
  # ⛔ QT LIVES IN THE PREFIX, because the build's run path names it and the
  # capture runs the build after this workdir's purpose is over.
  _qt="$PREFIX/opt/qt-6.11.2"
  fetch_qt "$_wd" "$_qt"
  timeout 600 git clone --depth 1 --branch "$LIBTORRENT_TAG" --recurse-submodules \
    --shallow-submodules https://github.com/arvidn/libtorrent.git "$_wd/libtorrent" \
    </dev/null >>"$_wd/install.log" 2>&1 || refuse "libtorrent $LIBTORRENT_TAG could not be cloned"
  printf 'libtorrent %s %s\n' "$LIBTORRENT_TAG" "$(git -C "$_wd/libtorrent" rev-parse HEAD)" \
    >>"$_wd/build-deps.txt" || refuse "the build could not record libtorrent's commit"
  _usr="$_wd/build-root/usr"
  set -- -G Ninja -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF \
    -DCMAKE_CXX_STANDARD=17 "-DBoost_INCLUDE_DIR=$_usr/include" \
    "-DCMAKE_INSTALL_PREFIX=$_wd/libtorrent-prefix"
  cmake -S "$_wd/libtorrent" -B "$_wd/libtorrent-build" "$@" </dev/null >>"$_wd/install.log" 2>&1 ||
    refuse "libtorrent could not be configured"
  cmake --build "$_wd/libtorrent-build" </dev/null >>"$_wd/install.log" 2>&1 ||
    refuse "libtorrent could not be built"
  cmake --install "$_wd/libtorrent-build" </dev/null >>"$_wd/install.log" 2>&1 ||
    refuse "libtorrent could not be staged"
  # ⛔ `-rpath-link`, BECAUSE THE RUN PATH NAMES QT ALONE. Qt's `libQt6Gui`
  # needs `libEGL.so.1`, which is in the unpacked runtime set, and with the
  # install run path in force the linker found it nowhere: the first driven
  # build on 2026-10-15 stopped on `undefined reference to eglBindAPI`. At run
  # time the same set is on `LD_LIBRARY_PATH` for both routes.
  set -- -G Ninja -DCMAKE_BUILD_TYPE=Release -DGUI=ON -DSTACKTRACE=OFF \
    "-DCMAKE_PREFIX_PATH=$_qt;$_wd/libtorrent-prefix;$_usr" \
    "-DCMAKE_LIBRARY_PATH=$_usr/lib/x86_64-linux-gnu" "-DCMAKE_INCLUDE_PATH=$_usr/include" \
    "-DBoost_INCLUDE_DIR=$_usr/include" \
    "-DCMAKE_INSTALL_RPATH=$_qt/lib" -DCMAKE_BUILD_WITH_INSTALL_RPATH=ON \
    "-DCMAKE_EXE_LINKER_FLAGS=-Wl,-rpath-link,$_usr/lib/x86_64-linux-gnu"
  {
    printf 'cmake -S <source> -B <build>'
    for _a in "$@"; do printf " '%s'" "$_a"; done
    printf '\nhost CFLAGS=%s\n' "${CFLAGS-<unset>}"
    printf 'host CXXFLAGS=%s\n' "${CXXFLAGS-<unset>}"
    printf 'host LDFLAGS=%s\n' "${LDFLAGS-<unset>}"
  } >"$_wd/build-configure.txt" || refuse "the source route could not record how it configured the build"
  cmake -S "$_src" -B "$_wd/build" "$@" </dev/null >>"$_wd/install.log" 2>&1 ||
    refuse "the source route could not configure a build"
  cmake --build "$_wd/build" </dev/null >>"$_wd/install.log" 2>&1 ||
    refuse "the source route could not build qbittorrent"
  [ -x "$_wd/build/qbittorrent" ] || refuse "the source route built nothing at build/qbittorrent"
  cp "$_wd/build/qbittorrent" "$PREFIX/bin/.$EXE.new" ||
    refuse "the source route could not stage $EXE in $PREFIX/bin"
  mv "$PREFIX/bin/.$EXE.new" "$PREFIX/bin/$EXE" ||
    refuse "the source route could not install $EXE into $PREFIX/bin"
}

[ $# -ge 1 ] || cannot "a subcommand is required"
COMMAND="$1"
shift

# What each route of this adapter delivers, as `Build.package` spells it.
#
# ⚠ An AppImage is not an ELF binary this project can treat as one: it is a
# self-mounting archive carrying its own runtime, so a record calling it
# `elf-binary` would say the two routes delivered one form when they did not.
package_format() { # route
  case "$1" in
    package) printf 'deb\n' ;;
    release) printf 'appimage\n' ;;
    source) printf 'elf-binary\n' ;;
    *) return 1 ;;
  esac
}

case "$COMMAND" in
  describe)
    printf 'target=qbittorrent\n'
    printf 'kind=stock\n'
    # ⭐ AND WHERE THE BUILD IS, WHEN THERE IS ONE. `install-client` asks twice -
    # once before the route runs and once after - so a route that replaced the
    # executable is visible as a changed path or a changed digest even when both
    # builds report the same version. ⛔ That case is real rather than
    # theoretical: `aria2` ships on `ubuntu-24.04` at the same version the vendor
    # publishes, so a release route there installs a genuinely different build
    # and a version comparison alone reports it as no acquisition at all.
    # ⚠ The key is OMITTED when nothing is installed, which is an answer rather
    # than an empty value: a caller that saw `binary=` could not tell a missing
    # build from an adapter that declines to say.
    _where=$(binary) || _where=""
    [ -z "$_where" ] || printf 'binary=%s\n' "$_where"
    # ⛔ WHAT THE RELEASE ROUTE NEEDS TO KNOW. `CLIENT-01` recorded on 2026-09-05
    # that this release "offers a Linux AppImage, a Windows x64_setup.exe and a
    # source tar.xz"; re-read on 2026-09-09, release-5.2.3 carries FOURTEEN
    # assets and TWO Linux AppImages - `qbittorrent-5.2.3_x86_64.AppImage` and
    # `qbittorrent-5.2.3_lt20_x86_64.AppImage` - so the summary was one build
    # short in exactly the place a route has to choose.
    #
    # ⭐ `{version}` IS WHAT SEPARATES THEM. `qbittorrent-*_x86_64.AppImage`
    # matches both, because a run swallows the `_lt20`; the pinned form matches
    # the unsuffixed one alone and `select-asset` refuses the ambiguity rather
    # than taking whichever the source listed first.
    #
    # ⚠ WHAT THIS DOES NOT ESTABLISH IS WHICH LIBTORRENT EACH APPIMAGE CARRIES.
    # The suffix plainly names a variant and nothing here has measured either, so
    # the pattern selects the vendor's unsuffixed build and the record names the
    # asset it took: a capture through this route measures whichever it
    # installed, and says which.
    #
    # ⚠ THREE OR FOUR COMPONENTS, from `ACQ-02`'s live dry run. The newest
    # release object on 2026-09-09 is `release-5.3.0beta1`, which the source
    # flags a prerelease and whose version text says so too, so the resolver
    # selects 5.2.3 - which is the pessimistic stability rule doing its job on a
    # live listing rather than on a fixture.
    printf 'release_repo=qbittorrent/qBittorrent\n'
    printf 'release_tag_prefix=release-\n'
    printf 'release_min_components=3\n'
    printf 'release_max_components=4\n'
    printf 'release_floor_major=-\n'
    # ⛔ THE `_lt20` ASSET, settled by the operator on 2026-10-15 once both were
    # measured: the unsuffixed AppImage carries libtorrent 1.2.20 and this one
    # 2.0.15, and the reserved bytes a pair compares are libtorrent's.
    printf 'release_asset=qbittorrent-{version}_lt20_x86_64.AppImage\n'
    # ⛔ THE DIGEST DISPOSITION, DECLARED ONCE AND READ BY THE ROUTE, AND IT IS
    # ABOUT THE VENDOR'S OWN CHECKSUMS DOCUMENT. ⚠ That is a narrower sentence
    # than "no digest", and the difference is measured: the release LISTING
    # carries a per-asset digest published by the party serving the bytes, which
    # `resolve-release.sh` resolves out of the same response and records as
    # `source-listing`. This key says only that the VENDOR uploaded no checksums
    # file - which is what an adapter can know by reading a release, and is what
    # decides whether the resolver goes looking for one.
    # ⚠ MEASURED through rule 8's route on 2026-09-17: release-5.2.3 carries
    # fourteen assets, seven of them detached `.asc` signatures, and no checksums
    # file - and `digest` is stated on all fourteen. So this route verifies, and
    # it verifies against the host rather than against the vendor.
    printf 'release_digest_unpublished=this release signs each asset with a detached .asc and publishes no checksums document\n'
    # ⚠ The source route compiles libtorrent and qBittorrent, measured at about
    # nine minutes on a four-processor session host on 2026-10-15.
    printf 'compiles=source\n'
    ;;

  install)
    [ $# -ge 2 ] || cannot "install takes <route> <workdir>"
    ROUTE="$1"
    WORKDIR="$2"
    mkdir -p "$WORKDIR" || cannot "cannot create $WORKDIR"
    _format=$(package_format "$ROUTE") ||
      cannot "this adapter does not say how the $ROUTE route packages a build"
    case "$ROUTE" in
      package)
        command -v apt-get >/dev/null 2>&1 || cannot "apt-get is not on this host"
        # ⚠ THE INDEX IS REFRESHED FIRST, and this is the one network call the
        # package route makes that is not the install. A runner image ships a
        # package list that was current when the image was built, and an install
        # against a stale one fails with a 404 on a version that has moved rather
        # than with anything naming the cause.
        # ⛔ NEEDRESTART_MODE=l IS `list`, AND THE LETTER IS THE WHOLE POINT.
        # Ubuntu 24.04 ships needrestart, which after a package install opens an
        # interactive dialog listing the services to restart; DEBIAN_FRONTEND
        # does not suppress it, and a dialog on a runner is a step that never
        # returns. ⚠ `a` stops the dialog by RESTARTING those services instead,
        # which on a runner means restarting daemons the job is standing on:
        # measured on 2026-09-08, an install under `a` finished and the step
        # after it then hung. `l` reports and touches nothing.
        # ⚠ Every apt call reads /dev/null, so anything that still asks gets
        # end-of-file rather than a wait.
        DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=l apt-get update \
          </dev/null >"$WORKDIR/update.log" 2>&1 || refuse "the package index could not be refreshed"
        DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=l apt-get install -y --no-install-recommends qbittorrent \
          </dev/null >"$WORKDIR/install.log" 2>&1 || refuse "the package route did not install qbittorrent"
        ;;
      release)
        # ⛔ THE VENDOR'S OWN ARTIFACT. CLIENT-01 measured on 2026-09-05 that the
        # release carries a Linux AppImage, a Windows setup executable and a
        # source archive, each with a detached signature beside it. The URL is
        # resolved before the route is cut and handed in; resolving it here
        # would be a network call under containment.
        [ -n "${BIT_IDS_RELEASE_URL:-}" ] ||
          cannot "the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut"
        command -v curl >/dev/null 2>&1 || cannot "curl is not on this host"
        #
        # ⛔ AND IT INSTALLS WHAT IT FETCHED. This route used to leave the
        # AppImage in the workdir and return 0, which is a route reporting an
        # install it did not perform: `version` then answered from whatever was
        # already on PATH, so a second route would have measured the FIRST
        # route's build and the two would have agreed for the most trivial reason
        # available. Measured on 2026-09-08 and recorded in `ACQ-03`.
        #
        # ⭐ AN AppImage IS THE INSTALLED PROGRAM, so installing it is placing it
        # where `binary()` looks. There is nothing to unpack and no prefix to
        # configure, which is the one thing this route has that aria2's does not.
        #
        # ⚠ WHAT THIS ASSUMES AND HAS NOT MEASURED: that the AppImage can run on
        # the capture host. An AppImage mounts itself with FUSE, and a runner
        # without it needs `APPIMAGE_EXTRACT_AND_RUN=1`; that variable is set on
        # every call this adapter makes into the build, which costs an extraction
        # per call and works either way. ⛔ Neither branch has been driven: this
        # adapter has never installed through this route.
        # ⛔ THE FETCH, THE DIGEST AND THE PLACING ARE ONE CALL, AND IT IS NOT
        # THIS FILE'S. `install-rootless.sh` bounds the transfer, settles the
        # digest before anything is made executable, writes into a prefix this
        # user already owns, and prints the path it placed. ⚠ This route used to
        # `curl` and `install -m755` into `/usr/local/bin` itself, which needs a
        # privilege `ACQ-06` removed from the capture path.
        #
        # ⚠ AND THIS VENDOR PUBLISHES NO DIGEST, WHICH IS DECLARED RATHER THAN
        # PASSED OVER, so the artifact is IDENTIFIED - its digest is recorded -
        # and not verified. Those are different sentences and the installer's own
        # report is where the difference is made.
        # ⛔ THE DIGEST DISPOSITION IS THE RESOLUTION'S AND THE INSTALLER READS
        # IT. `ACQ-06`. Four adapters choosing between four dispositions would be
        # four copies of one decision, which is the one-gated-door shape
        # `docs/methodology/reviews.md` names the most recurring hole there is.
        # ⚠ The second branch is the standalone drive - an adapter run with a URL
        # and no resolution - and it carries this adapter's own declaration,
        # asked of `describe` rather than spelled a second time.
        if [ -n "${BIT_IDS_RELEASE_RESOLUTION:-}" ]; then
          set -- --from-resolution "$BIT_IDS_RELEASE_RESOLUTION"
        else
          _why=$(sh "$0" describe |
            awk -F= '$1 == "release_digest_unpublished" { sub(/^[^=]*=/, ""); print; exit }')
          [ -n "$_why" ] ||
            cannot "this adapter declares no digest disposition for its release route"
          set -- --digest-unpublished "$_why"
        fi
        sh "$ROOTLESS" --install \
          --url "$BIT_IDS_RELEASE_URL" \
          --into "$WORKDIR/qbittorrent.AppImage" \
          --as "$EXE" \
          "$@" \
          --report "$WORKDIR/rootless-install.txt" \
          </dev/null >>"$WORKDIR/install.log" 2>&1
        _rc=$?
        [ "$_rc" = 0 ] || {
          tail -20 "$WORKDIR/install.log" >&2
          [ "$_rc" = 1 ] ||
            cannot "the rootless installer could not run for the release route (exit $_rc)"
          refuse "the release route could not install the vendor's AppImage"
        }
        install_runtime "$WORKDIR"
        ;;
      source)
        # ⭐ THE SECOND ROUTE: THE REPOSITORY'S OWN TAG, CLONED AND BUILT, as
        # `transmission` and `aria2-next` do. It resolves through the
        # repository's refs, which since 2026-10-15 read `release-5.3.0beta1`'s
        # marker after its digits and select `release-5.2.4`. `CLIENT-01`.
        _tag=${BIT_IDS_SOURCE_TAG:-}
        [ -n "$_tag" ] ||
          cannot "the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut"
        command -v git >/dev/null 2>&1 || cannot "the source route clones and git is not on this host"
        _repo=$(sh "$0" describe | awk -F= '$1 == "release_repo" { sub(/^[^=]*=/, ""); print; exit }')
        [ -n "$_repo" ] || cannot "this adapter declares no release_repo to clone from"
        timeout 600 git clone --depth 1 --branch "$_tag" "https://github.com/$_repo.git" "$WORKDIR/src" \
          </dev/null >"$WORKDIR/install.log" 2>&1 || refuse "the source route could not clone $_tag"
        # ⛔ THE COMMIT `E-ACQ-06` ASKS FOR, NOT THE TAG, which is a name.
        git -C "$WORKDIR/src" rev-parse HEAD >"$WORKDIR/source-commit" 2>>"$WORKDIR/install.log" ||
          refuse "the source route cloned $_tag and could not name its commit"
        build_source "$WORKDIR" "$WORKDIR/src"
        install_runtime "$WORKDIR"
        ;;
      *) cannot "unknown route: $ROUTE" ;;
    esac
    # ⛔ WHAT THE ROUTE DELIVERED, WRITTEN WHERE `install-client` READS IT.
    # `Build.package` is part of the identity tuple a store path is derived
    # from, so a record filed without it - or with one this project guessed -
    # files two packagings of one version at one name. Measured on 2026-09-09 by
    # assembling capture-client run 14: nothing any adapter wrote said how the
    # build arrived, and `assemble-capture` refused both lanes for it.
    # ⚠ THE KEY IS `package` BECAUSE THAT IS THE RECORD'S OWN VOCABULARY, and
    # the ROUTE also called `package` is the host's package-manager route. They
    # are different things and this is the one file where both appear.
    printf '%s\n' "$_format" >"$WORKDIR/package" ||
      refuse "the route installed and could not record how it packaged the build"
    ;;

  version)
    BINARY=$(binary) || cannot "$EXE is not installed"
    [ -n "$BINARY" ] || cannot "$EXE is not installed"
    # ⛔ THE BUILD SPEAKING. `qBittorrent v5.0.2` on its first line; the leading
    # `v` is stripped because a version is compared by VersionScheme::components
    # and a label carrying a sigil is a second spelling of one value.
    # ⚠ THE FLAG THAT USED TO BE HERE WAS AN ASSUMPTION AND IT WAS WRONG ON BOTH
    # PATHS AT ONCE. Adding it to this call to match `start` looked like closing
    # a one-gated door; what it actually did was spread a refused argument to a
    # second place. ⭐ Reading the product's own message is what settled it.
    # ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED. This
    # was `$(... | head -1) || cannot`, whose `||` reads HEAD's status: head
    # exits 0 over anything, so the guard was dead code and a build that refused
    # to answer reached the parse instead of the refusal. ⚠ Measured on
    # 2026-09-08, when a client capture reported "would not report a version" and
    # nothing said which of three refusals had fired. This repository's oldest
    # stated rule, broken in every adapter at once.
    # ⛔ NO --confirm-legal-notice HERE, AND THAT IS MEASURED RATHER THAN
    # ASSUMED. This build answers `Bad command line: --confirm-legal-notice is
    # an unknown command line parameter`, so the flag was not a control at all;
    # it was an argument the product refuses. Client capture run 3 said so in
    # those words. ⚠ Stdin is still /dev/null and the caller still bounds this,
    # because a prompt is a separate hazard from a rejected flag.
    #
    # ⚠ stderr is KEPT here rather than discarded. A product that refuses to
    # answer says why on it, and the caller prints what this prints.
    # ⚠ THE GUI BUILD ANSWERS ONLY ON A DISPLAY: it constructs its application
    # before it reads `--version`. Measured on 2026-10-15 with both routes'
    # builds; each prints `qBittorrent v5.2.4` under Xvfb.
    _xpid=$(mktemp) || cannot "cannot make a temporary file"
    _display=$(start_display "$_xpid") || exit $?
    _rt=$(runtime_dir) || cannot "the rootless prefix could not be derived"
    OUTPUT=$(run_build "$_display" "$_rt" "$BINARY" --version </dev/null 2>&1)
    VERSION_RC=$?
    stop_display "$_xpid"
    rm -f "$_xpid"
    [ "$VERSION_RC" = 0 ] ||
      cannot "$EXE --version exited $VERSION_RC: $(printf '%s' "$OUTPUT" | head -3 | tr '\n' ' ')"
    LINE=$(printf '%s\n' "$OUTPUT" | grep -i qbittorrent | head -1)
    [ -n "$LINE" ] || LINE=$(printf '%s\n' "$OUTPUT" | head -1)
    VERSION=$(printf '%s' "$LINE" | awk '{ print $NF }' | sed 's/^v//')
    [ -n "$VERSION" ] || cannot "$EXE reported no parseable version: [$LINE]"
    # ⛔ WHAT THE BUILD SAID ABOUT ITSELF IS KEPT, NOT ONLY THE FIELD PARSED OUT
    # OF IT. Two routes can install builds that report ONE version and are not
    # one build: measured on 2026-09-08, Ubuntu's aria2 1.37.0 and a 1.37.0 built
    # from the vendor's own release tarball answer the same version and enable
    # different features - the package build lists Async DNS, Metalink, XML-RPC,
    # SFTP and Firefox3 Cookie, and a source build configured without those
    # dependencies does not. ⚠ Features are what a build DOES on the wire, so a
    # record holding only `1.37.0` has dropped the evidence that would have
    # distinguished them.
    # ⭐ stderr, because both callers already keep it: install-client writes it to
    # `version.err` beside the install record and capture-client appends it to
    # `adapter.err` inside the evidence bundle. Nothing about the parsed value on
    # stdout changes.
    printf '%s reported:\n' "$ME" >&2
    printf '%s\n' "$OUTPUT" >&2
    printf '%s\n' "$VERSION"
    ;;

  start)
    [ $# -ge 3 ] || cannot "start takes <torrent> <workdir> <peer-port>"
    TORRENT="$1"
    WORKDIR="$2"
    PORT="$3"
    BINARY=$(binary) || cannot "$EXE is not installed"
    [ -n "$BINARY" ] || cannot "$EXE is not installed"
    [ -s "$TORRENT" ] || cannot "there is no torrent at $TORRENT"

    PROFILE="$WORKDIR/profile"
    CONFIG="$PROFILE/qBittorrent/config"
    mkdir -p "$CONFIG" "$WORKDIR/downloads" || cannot "cannot create the profile under $WORKDIR"

    # ⛔ THE WHOLE PROFILE IS WRITTEN, NOT PATCHED. A capture that edited an
    # existing configuration would measure a build under whatever else was in
    # it, and the settings a run used are part of what that run recorded. ⚠ The
    # three adjacent surfaces are named individually: DHT, peer exchange and
    # local peer discovery are three destinations, and a host with no default
    # route still multicasts.
    {
      printf '[BitTorrent]\n'
      printf 'Session\\DHTEnabled=false\n'
      printf 'Session\\PeXEnabled=false\n'
      printf 'Session\\LSDEnabled=false\n'
      printf 'Session\\AnonymousModeEnabled=false\n'
      printf 'Session\\Encryption=0\n'
      printf 'Session\\DefaultSavePath=%s/downloads\n' "$WORKDIR"
      printf 'Session\\TempPathEnabled=false\n'
      printf 'Session\\AddTorrentStopped=false\n'
      printf 'Session\\GlobalMaxSeedingMinutes=1\n'
      [ "$PORT" = "0" ] || printf 'Session\\Port=%s\n' "$PORT"
      printf '\n'
      printf '[Preferences]\n'
      printf 'Connection\\UPnP=false\n'
      printf 'Downloads\\SavePath=%s/downloads\n' "$WORKDIR"
      printf 'General\\Locale=en\n'
      printf 'WebUI\\Enabled=false\n'
      printf '\n'
      printf '[LegalNotice]\n'
      printf 'Accepted=true\n'
      # ⚠ THE GUI ASKS ABOUT A TORRENT IT IS HANDED, and nothing would answer.
      # `AddNewTorrentDialog/Enabled` is the 5.2.4 key, read from its source.
      printf '\n'
      printf '[AddNewTorrentDialog]\n'
      printf 'Enabled=false\n'
    } >"$CONFIG/qBittorrent.conf" || cannot "the profile could not be written"

    # ⛔ THE LEGAL NOTICE IS ACCEPTED IN THE PROFILE, NOT ON THE COMMAND LINE.
    # `--confirm-legal-notice` is an argument this build REFUSES - measured on
    # 2026-09-08, in those words - so passing it here would have failed the
    # start exactly as it failed the version call. ⚠ The same wrong assumption
    # was on both paths into this product, which is why removing it from one
    # would have left the other.
    #
    # ⛔ AND THE TORRENT IS A POSITIONAL ARGUMENT, which is the one control this
    # product already has that needs no web interface, no credential and no
    # second process. An adapter driving the WebUI would be measuring a build
    # through an authenticated API it had to configure first.
    # ⭐ A DISPLAY PER SESSION, stopped with the build, so a session's server is
    # never the previous session's.
    _rt=$(runtime_dir) || cannot "the rootless prefix could not be derived"
    _display=$(start_display "$WORKDIR/xvfb.pid") || exit $?
    HOME="$WORKDIR" run_build "$_display" "$_rt" "$BINARY" \
      --profile="$PROFILE" \
      --relative-fastresume \
      --skip-dialog=true \
      "$TORRENT" </dev/null >>"$WORKDIR/client.out" 2>&1 &
    printf '%s\n' "$!" >"$WORKDIR/pid"
    printf 'started %s pid %s on port %s, display %s\n' "$EXE" "$(cat "$WORKDIR/pid")" "$PORT" "$_display"
    ;;

  stop)
    [ $# -ge 1 ] || cannot "stop takes <workdir>"
    WORKDIR="$1"
    # ⛔ THE DISPLAY GOES WITH THE BUILD, on every path out of here.
    trap 'stop_display "$WORKDIR/xvfb.pid"' EXIT
    [ -f "$WORKDIR/pid" ] || exit 0
    PID=$(cat "$WORKDIR/pid")
    case "$PID" in
      '' | *[!0-9]*) exit 0 ;;
    esac
    kill "$PID" 2>/dev/null || exit 0
    WAITED=0
    while [ "$WAITED" -lt 8 ]; do
      kill -0 "$PID" 2>/dev/null || exit 0
      sleep 1
      WAITED=$((WAITED + 1))
    done
    kill -9 "$PID" 2>/dev/null
    exit 0
    ;;

  *) cannot "unknown subcommand: $COMMAND" ;;
esac
