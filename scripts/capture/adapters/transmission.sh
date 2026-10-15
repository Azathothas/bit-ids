#!/bin/sh
# transmission.sh - the CLIENT-06 capture adapter.
#
# ⚠ THE ONE TARGET WHOSE ENTRY IS ABOUT NOT USING SOURCE. CLIENT-06 exists
# because published formulas for Transmission's peer ID are good hypotheses and
# violate the live-only evidence rule, so everything this adapter contributes is
# a running process and nothing it contributes is a formula.
#
# ⭐ ITS CONTROL SURFACE IS A SECOND PROCESS RATHER THAN A FILE. transmission-cli
# was removed upstream, so the supported unattended path is the daemon plus
# transmission-remote, and that is what this drives: the daemon is started with
# its whole settings file written, and the torrent is added by the tool the
# product ships for the purpose.
#
# -- ⛔ WHAT THIS ASSUMES ABOUT THE PRODUCT, AND HAS NOT MEASURED --------------
#
# Nothing below has been run against an installed Transmission. Each line is a
# claim a dispatch settles:
#
#   * `transmission-daemon --version` names the version in its second field,
#     with a build identifier beside it that is a separate fact;
#   * `transmission-remote` arrives with `transmission-cli` on this
#     distribution, and the daemon in `transmission-daemon`;
#   * the `dht-enabled`, `pex-enabled` and `lpd-enabled` settings are the three
#     adjacent surfaces, and `settings.json` under `--config-dir` is read at
#     start;
#   * `--foreground` keeps the daemon as the process this backgrounds;
#   * the distribution starts the daemon as a service on install, which is why
#     the package route stops and disables it.
#
# ⚠ The version fields are the claim most likely to be wrong, and it fails
# loudly: `capture-client` refuses a version that is not printable text and one
# that is empty, so a mis-parse is a refused capture rather than a record
# naming a build nobody ran.
#
# See adapters/README.md for the contract.
#
# Exit codes: 0 done, 1 refused, 2 could not run.

set -u

ME=transmission
RPC_PORT=${BIT_IDS_TRANSMISSION_RPC:-9091}

refuse() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 1
}

cannot() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 2
}

# ⛔ THE ROOTLESS INSTALLER, AND THE ONE PREFIX IT DERIVES. The release and
# source routes compile and place both executables at `<prefix>/bin`, so the
# lookups below ask that prefix first. `daemon()` had no prefix branch until
# 2026-10-14, when neither route installed into one. `CLIENT-06`.
# ⭐ `bit-check install-rootless` SINCE 2026-10-15, `CI-11`, found through
# BIT_CHECK, which the workflow exports with the observer, and never built here.
rootless() {
  if [ -z "${BIT_CHECK:-}" ] || [ ! -x "$BIT_CHECK" ]; then
    cannot "BIT_CHECK names no bit-check binary; it is built with the observer, never here"
  fi
  "$BIT_CHECK" install-rootless "$@"
}
prefix() {
  rootless --prefix
}

# ⛔ ONE LOOKUP PER EXECUTABLE, so every subcommand finds the same build.
installed() { # override name
  if [ -n "$1" ]; then
    printf '%s' "$1"
    return 0
  fi
  _prefix=$(prefix 2>/dev/null) || _prefix=""
  if [ -n "$_prefix" ] && [ -x "$_prefix/bin/$2" ]; then
    printf '%s' "$_prefix/bin/$2"
    return 0
  fi
  command -v "$2" 2>/dev/null
}
daemon() { installed "${BIT_IDS_TRANSMISSION_DAEMON:-}" transmission-daemon; }
remote() { installed "${BIT_IDS_TRANSMISSION_REMOTE:-}" transmission-remote; }

# ⛔ LIBCURL'S HEADERS, WITHOUT ROOT AND WITHOUT CHANGING THE HOST. Transmission
# 4.x links libcurl, and a runner carries its runtime and not its development
# package. Measured on 2026-10-14: `apt-get` with its index and cache in the
# workdir runs as an unprivileged user, fetches `libcurl4-openssl-dev` through
# the distribution's signed index, and `dpkg-deb -x` unpacks it there. The
# headers then belong to the runtime the host already has, which is checked
# rather than assumed. `ACQ-06` is why nothing here may escalate.
# Prints the include directory, and writes what it took and which runtime the
# headers describe to `build-deps.txt`.
curl_headers() { # workdir
  _wd=$1
  _deps="$_wd/deps"
  mkdir -p "$_deps/apt/lists/partial" "$_deps/apt/cache/archives/partial" \
    "$_deps/deb" "$_deps/root" || cannot "cannot create $_deps"
  for _need in apt-get dpkg-deb; do
    command -v "$_need" >/dev/null 2>&1 ||
      cannot "the build needs libcurl's headers and $_need is not on this host"
  done
  set -- -o "Dir::State::Lists=$_deps/apt/lists" -o "Dir::Cache=$_deps/apt/cache" \
    -o Debug::NoLocking=1
  timeout 300 apt-get "$@" update </dev/null >"$_wd/deps.log" 2>&1 ||
    refuse "a private package index could not be fetched for libcurl's headers"
  (cd "$_deps/deb" && timeout 120 apt-get "$@" download libcurl4-openssl-dev) \
    </dev/null >>"$_wd/deps.log" 2>&1 ||
    refuse "libcurl4-openssl-dev could not be downloaded"
  set -- "$_deps"/deb/libcurl4-openssl-dev_*.deb
  if [ $# != 1 ] || [ ! -f "$1" ]; then
    refuse "the download left no single libcurl4-openssl-dev package"
  fi
  _deb=$1
  dpkg-deb -x "$_deb" "$_deps/root" || refuse "libcurl4-openssl-dev could not be unpacked"
  _header=$(find "$_deps/root/usr/include" -path '*/curl/curl.h' -type f | head -1)
  [ -n "$_header" ] || refuse "libcurl4-openssl-dev carried no curl/curl.h"
  # ⛔ THE HEADERS MUST DESCRIBE THE LIBRARY THAT WILL BE LOADED. The package's
  # `libcurl.so` names the file of the runtime it was built against, and that
  # file must already be on this host.
  _link=$(find "$_deps/root/usr/lib" -name libcurl.so -type l | head -1)
  [ -n "$_link" ] || refuse "libcurl4-openssl-dev carried no libcurl.so link"
  _dir=${_link#"$_deps/root"}
  _runtime="${_dir%/*}/$(readlink "$_link")"
  [ -f "$_runtime" ] ||
    refuse "libcurl4-openssl-dev is for $_runtime, which this host does not have"
  {
    printf 'package=%s\n' "$(dpkg-deb -f "$_deb" Package)"
    printf 'version=%s\n' "$(dpkg-deb -f "$_deb" Version)"
    printf 'sha256=%s\n' "$(sha256sum "$_deb" | cut -d' ' -f1)"
    printf 'runtime=%s\n' "$_runtime"
  } >"$_wd/build-deps.txt" || refuse "the build could not record its dependency"
  printf '%s\n' "${_header%/curl/curl.h}"
}

# ⛔ ONE BUILD FOR BOTH ROUTES, so the pair differs in what it was built from and
# in nothing this file chose. The daemon and `transmission-remote` are built; the
# GTK and Qt clients, the CLI and the tests are not, because the capture drives
# the daemon. The configure line goes to `build-configure.txt` beside the
# install, as `aria2-next`'s source route does.
build_into_prefix() { # source-dir route
  _src=$1
  _route=$2
  _include=$(curl_headers "$WORKDIR") || exit $?
  _runtime=$(awk -F= '$1 == "runtime" { print $2 }' "$WORKDIR/build-deps.txt")
  for _need in cmake ninja c++; do
    command -v "$_need" >/dev/null 2>&1 || cannot "the $_route route compiles and $_need is not on this host"
  done
  PREFIX=$(prefix) || cannot "the rootless prefix could not be derived"
  mkdir -p "$PREFIX/bin" ||
    refuse "this user cannot create $PREFIX/bin; a rootless install writes nowhere else"
  set -- -G Ninja -DCMAKE_BUILD_TYPE=Release \
    -DENABLE_GTK=OFF -DENABLE_QT=OFF -DENABLE_MAC=OFF -DENABLE_CLI=OFF \
    -DENABLE_TESTS=OFF -DENABLE_DAEMON=ON -DENABLE_UTILS=ON \
    "-DCURL_INCLUDE_DIR=$_include" "-DCURL_LIBRARY=$_runtime"
  {
    printf 'cmake -S <%s> -B <build>' "$_route"
    for _a in "$@"; do printf " '%s'" "$_a"; done
    printf '\nhost CFLAGS=%s\n' "${CFLAGS-<unset>}"
    printf 'host CXXFLAGS=%s\n' "${CXXFLAGS-<unset>}"
    printf 'host LDFLAGS=%s\n' "${LDFLAGS-<unset>}"
  } >"$WORKDIR/build-configure.txt" ||
    refuse "the $_route route could not record how it configured the build"
  cmake -S "$_src" -B "$WORKDIR/build" "$@" </dev/null >>"$WORKDIR/install.log" 2>&1 ||
    refuse "the $_route route could not configure a build"
  cmake --build "$WORKDIR/build" </dev/null >>"$WORKDIR/install.log" 2>&1 ||
    refuse "the $_route route could not build transmission"
  # ⛔ BOTH ARE CHECKED BEFORE EITHER IS INSTALLED. A refusal that had already
  # placed the daemon would leave a build in the prefix for the next lane to
  # measure; `check-transmission` found that on its first run, 2026-10-14.
  for _exe in daemon/transmission-daemon utils/transmission-remote; do
    [ -x "$WORKDIR/build/$_exe" ] ||
      refuse "the $_route route built nothing at build/$_exe"
  done
  for _exe in daemon/transmission-daemon utils/transmission-remote; do
    _name=${_exe#*/}
    cp "$WORKDIR/build/$_exe" "$PREFIX/bin/.$_name.new" ||
      refuse "the $_route route could not stage $_name in $PREFIX/bin"
    mv "$PREFIX/bin/.$_name.new" "$PREFIX/bin/$_name" ||
      refuse "the $_route route could not install $_name into $PREFIX/bin"
  done
}

[ $# -ge 1 ] || cannot "a subcommand is required"
COMMAND="$1"
shift

# What each route of this adapter delivers, as `Build.package` spells it.
# ⚠ The release and source routes compile, so what each delivers is an
# executable this host built, which is `aria2-next`'s source route's spelling.
package_format() { # route
  case "$1" in
    package) printf 'deb\n' ;;
    release | source) printf 'elf-binary\n' ;;
    *) return 1 ;;
  esac
}

case "$COMMAND" in
  describe)
    printf 'target=transmission\n'
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
    _where=$(daemon) || _where=""
    [ -z "$_where" ] || printf 'binary=%s\n' "$_where"
    # ⛔ WHAT THE RELEASE ROUTE NEEDS TO KNOW. These were declared while the
    # route still refused at the build, which made the build a named gap; the
    # route builds since 2026-10-14. `CLIENT-06`.
    #
    # ⚠ THE TAGS CARRY NO PREFIX, measured by `ACQ-02`'s live dry run: 83
    # candidates, bare three-component tags, `4.1.3` selected over 10 superseded,
    # 21 prerelease and 51 predating. `-` is how `resolve-stable` spells "no
    # prefix", and it is passed through rather than left empty so a missing
    # declaration stays distinguishable from a declared absence.
    #
    # ⚠ ONE SOURCE ARCHIVE AND NO LINUX BINARY. Measured 2026-09-09 against the
    # live listing: release 4.1.3 carries eleven assets - Windows `.msi` files,
    # `-pdb.7z` symbol archives, a macOS `.dmg`, a `-dsym.zip` and exactly one
    # `transmission-4.1.3.tar.xz`. ⛔ Two of them are capitalised `Transmission-`,
    # which is why the pattern is matched case-sensitively.
    printf 'release_repo=transmission/transmission\n'
    printf 'release_tag_prefix=-\n'
    printf 'release_min_components=3\n'
    printf 'release_max_components=3\n'
    # ⭐ The three-component spelling began at 4.0.0, and refs carry 112 tags
    # before it, from 0.5 to 3.00, that it cannot order and no date releases.
    # Measured on 2026-10-15; `CLIENT-06`.
    printf 'release_floor_major=4\n'
    printf 'release_asset=transmission-{version}.tar.xz\n'
    # ⛔ THE DIGEST DISPOSITION, DECLARED ONCE AND READ BY THE ROUTE, AND IT IS
    # ABOUT THE VENDOR'S OWN CHECKSUMS DOCUMENT. ⚠ That is a narrower sentence
    # than "no digest", and the difference is measured: the release LISTING
    # carries a per-asset digest published by the party serving the bytes, which
    # `resolve-release.sh` resolves out of the same response and records as
    # `source-listing`. This key says only that the VENDOR uploaded no checksums
    # file - which is what an adapter can know by reading a release, and is what
    # decides whether the resolver goes looking for one.
    # ⚠ MEASURED through rule 8's route on 2026-09-17: 4.1.3 carries installers,
    # debug archives and one source tarball, no checksums file, and `digest` on
    # all eleven. ⚠ The route refuses after its fetch either way; the declaration
    # is what stops the fetch itself being a transfer nothing identified.
    printf 'release_digest_unpublished=this release publishes installers and a source tarball and no checksums asset\n'
    # ⛔ WHICH ROUTES COMPILE, SO THE WORKFLOW BOUNDS THEM AS BUILDS. The release
    # asset is a source tarball, so both routes here run a build measured at 695
    # seconds, and a release route bounded as a download would be killed
    # half-way. `CLIENT-06`.
    printf 'compiles=release,source\n'
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
        # ⚠ BOTH HALVES, because the control surface is a separate binary. A
        # route that installed the daemon alone would leave `start` unable to
        # add a torrent and the failure would read as the build declining one.
        DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=l apt-get install -y --no-install-recommends \
          transmission-daemon transmission-cli \
          </dev/null >"$WORKDIR/install.log" 2>&1 || refuse "the package route did not install transmission"
        # ⚠ The distribution starts the daemon as a service on install. A
        # capture must own the only running copy, or it would measure whichever
        # one the torrent reached.
        if command -v systemctl >/dev/null 2>&1; then
          systemctl stop transmission-daemon >>"$WORKDIR/install.log" 2>&1 || :
          systemctl disable transmission-daemon >>"$WORKDIR/install.log" 2>&1 || :
        fi
        ;;
      release)
        [ -n "${BIT_IDS_RELEASE_URL:-}" ] ||
          cannot "the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut"
        #
        # ⭐ THE VENDOR'S SOURCE TARBALL, VERIFIED AND THEN BUILT. Release 4.1.3
        # publishes no Linux binary: `transmission-4.1.3.tar.xz`, Windows
        # installers and a macOS image. ⚠ This route refused after its fetch
        # until 2026-10-14, because nothing here could build it; a route that
        # installs nothing must not report an install, and that still holds
        # for every refusal below. `CLIENT-06`.
        # ⛔ THE FETCH AND THE DIGEST ARE ONE CALL AND IT IS NOT THIS FILE'S.
        # `bit-check install-rootless` bounds the transfer and settles the digest, and
        # it needs no privilege, so this route reaches its refusal on a host
        # where nothing can be installed to `/usr/local` at all.
        #
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
        rootless --fetch \
          --url "$BIT_IDS_RELEASE_URL" \
          --into "$WORKDIR/transmission-release" \
          "$@" \
          --report "$WORKDIR/rootless-install.txt" \
          </dev/null >>"$WORKDIR/install.log" 2>&1
        _rc=$?
        [ "$_rc" = 0 ] || {
          tail -20 "$WORKDIR/install.log" >&2
          [ "$_rc" = 1 ] ||
            cannot "the rootless installer could not run for the release route (exit $_rc)"
          refuse "the release route could not fetch the vendor's source archive"
        }
        mkdir -p "$WORKDIR/release-src" ||
          refuse "the release route could not create its source directory"
        tar -xJf "$WORKDIR/transmission-release" -C "$WORKDIR/release-src" \
          </dev/null >>"$WORKDIR/install.log" 2>&1 ||
          refuse "the release route's archive did not unpack"
        set -- "$WORKDIR"/release-src/*/CMakeLists.txt
        if [ $# != 1 ] || [ ! -f "$1" ]; then
          refuse "the release archive did not unpack into one source tree"
        fi
        build_into_prefix "${1%/CMakeLists.txt}" release
        ;;
      source)
        # ⭐ THE SECOND ROUTE: THE REPOSITORY'S OWN TAG, CLONED AND BUILT. It
        # resolves through the repository's refs rather than the release listing,
        # and arrives as a clone rather than an asset, which is the shape
        # `aria2-next` measured. ⚠ Whoever controls the repository controls both.
        _tag=${BIT_IDS_SOURCE_TAG:-}
        [ -n "$_tag" ] ||
          cannot "the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut"
        command -v git >/dev/null 2>&1 || cannot "the source route clones and git is not on this host"
        _repo=$(sh "$0" describe | awk -F= '$1 == "release_repo" { sub(/^[^=]*=/, ""); print; exit }')
        [ -n "$_repo" ] || cannot "this adapter declares no release_repo to clone from"
        # ⛔ THE SUBMODULES ARE PART OF THE SOURCE. 4.1.3 pins sixteen under
        # `third-party/`, which the release tarball carries inline; a clone
        # without them is a tree that does not build. Shallow, as the history
        # is not the measurement: 39 seconds and 137 megabytes on 2026-10-14.
        timeout 600 git clone --depth 1 --branch "$_tag" --recurse-submodules \
          --shallow-submodules "https://github.com/$_repo.git" "$WORKDIR/src" \
          </dev/null >"$WORKDIR/install.log" 2>&1
        _clone=$?
        [ "$_clone" = 0 ] || {
          [ "$_clone" != 124 ] ||
            refuse "the source route's clone of $_tag did not finish within 600s"
          refuse "the source route could not clone $_tag"
        }
        # ⛔ THE COMMIT `E-ACQ-06` ASKS FOR, NOT THE TAG, which is a name.
        git -C "$WORKDIR/src" rev-parse HEAD >"$WORKDIR/source-commit" 2>>"$WORKDIR/install.log" ||
          refuse "the source route cloned $_tag and could not name its commit"
        build_into_prefix "$WORKDIR/src" source
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
    BINARY=$(daemon) || cannot "transmission-daemon is not installed"
    [ -n "$BINARY" ] || cannot "transmission-daemon is not installed"
    # ⛔ THE BUILD SPEAKING. `transmission-daemon 4.0.5 (fac9a5f1c8)` is the
    # shape; the field taken is the version alone and the build hash beside it
    # is left where it is, because a version and a build are two facts.
    # ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED. This
    # was `$(... | head -1) || cannot`, whose `||` reads HEAD's status: head
    # exits 0 over anything, so the guard was dead code and a build that refused
    # to answer reached the parse instead of the refusal. ⚠ Measured on
    # 2026-09-08, when a client capture reported "would not report a version" and
    # nothing said which of three refusals had fired. This repository's oldest
    # stated rule, broken in every adapter at once.
    # ⚠ Merged on purpose: this product prints its version to stderr, which
    # shell.md section 3 names as the case where merging is the decision.
    OUTPUT=$("$BINARY" --version 2>&1 </dev/null)
    VERSION_RC=$?
    [ "$VERSION_RC" = 0 ] ||
      cannot "transmission-daemon --version exited $VERSION_RC"
    LINE=$(printf '%s\n' "$OUTPUT" | head -1)
    VERSION=$(printf '%s' "$LINE" | awk '{ print $2 }')
    [ -n "$VERSION" ] || cannot "transmission-daemon reported no parseable version: [$LINE]"
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
    BINARY=$(daemon) || cannot "transmission-daemon is not installed"
    [ -n "$BINARY" ] || cannot "transmission-daemon is not installed"
    REMOTE=$(remote) || cannot "transmission-remote is not installed"
    [ -n "$REMOTE" ] || cannot "transmission-remote is not installed"
    [ -s "$TORRENT" ] || cannot "there is no torrent at $TORRENT"

    CONFIG="$WORKDIR/config"
    mkdir -p "$CONFIG" "$WORKDIR/downloads" || cannot "cannot create the config under $WORKDIR"

    # ⛔ THE WHOLE SETTINGS FILE IS WRITTEN. The three adjacent surfaces are
    # named individually, and port forwarding is off as well: a host with no
    # default route cannot reach a NAT gateway, and a build asking one is a
    # destination this run did not intend.
    {
      printf '{\n'
      printf '  "dht-enabled": false,\n'
      printf '  "pex-enabled": false,\n'
      printf '  "lpd-enabled": false,\n'
      printf '  "utp-enabled": false,\n'
      printf '  "port-forwarding-enabled": false,\n'
      printf '  "encryption": 0,\n'
      printf '  "download-dir": "%s/downloads",\n' "$WORKDIR"
      printf '  "incomplete-dir-enabled": false,\n'
      printf '  "rpc-enabled": true,\n'
      printf '  "rpc-bind-address": "127.0.0.1",\n'
      printf '  "rpc-port": %s,\n' "$RPC_PORT"
      printf '  "rpc-authentication-required": false,\n'
      printf '  "rpc-whitelist": "127.0.0.1",\n'
      printf '  "rpc-host-whitelist-enabled": false,\n'
      printf '  "ratio-limit-enabled": false,\n'
      [ "$PORT" = "0" ] || printf '  "peer-port": %s,\n' "$PORT"
      printf '  "peer-port-random-on-start": false\n'
      printf '}\n'
    } >"$CONFIG/settings.json" || cannot "the settings could not be written"

    # ⚠ --foreground, so the process this backgrounds is the daemon itself. A
    # daemon that forked would leave `stop` holding the pid of a shell that had
    # already returned, and the build would outlive the capture.
    "$BINARY" --foreground --config-dir "$CONFIG" --log-error \
      >"$WORKDIR/client.out" 2>&1 &
    printf '%s\n' "$!" >"$WORKDIR/pid"

    # ⚠ The daemon is asked for its session before a torrent is handed over,
    # because transmission-remote against a socket nothing is listening on
    # reports a connection error, which would read as the build refusing the
    # torrent rather than as a daemon that had not finished starting.
    WAITED=0
    while [ "$WAITED" -lt 20 ]; do
      "$REMOTE" "127.0.0.1:$RPC_PORT" --session-info >/dev/null 2>&1 && break
      kill -0 "$(cat "$WORKDIR/pid")" 2>/dev/null || refuse "the daemon exited before it answered"
      sleep 1
      WAITED=$((WAITED + 1))
    done
    [ "$WAITED" -lt 20 ] || refuse "the daemon never answered on 127.0.0.1:$RPC_PORT"

    "$REMOTE" "127.0.0.1:$RPC_PORT" --add "$TORRENT" >>"$WORKDIR/client.out" 2>&1 ||
      refuse "transmission-remote would not add the torrent"
    "$REMOTE" "127.0.0.1:$RPC_PORT" --torrent all --start >>"$WORKDIR/client.out" 2>&1 ||
      refuse "transmission-remote would not start the torrent"
    printf 'started transmission-daemon pid %s on port %s\n' "$(cat "$WORKDIR/pid")" "$PORT"
    ;;

  stop)
    [ $# -ge 1 ] || cannot "stop takes <workdir>"
    WORKDIR="$1"
    # ⛔ THE DAEMON WRITES AN RPC VERIFIER INTO ITS SETTINGS ON EXIT, AND RULE 12
    # KEEPS IT OFF THE BUNDLE. Measured on `capture-client` run 38600252161,
    # 2026-10-14: 4.1.3 saves `"rpc-password": "{<salted hash>"` even with
    # authentication off, and `scan-bundle.sh` refused the capture for it. So
    # once the process is gone, that key is deleted, and if `jq` is absent the
    # whole file is, rather than a line edit that could leave invalid JSON.
    # ⚠ Every exit below runs it, because each is a daemon no longer running.
    forget_verifier() {
      _settings="$WORKDIR/config/settings.json"
      [ -f "$_settings" ] || return 0
      if command -v jq >/dev/null 2>&1 &&
        jq 'del(."rpc-password")' "$_settings" >"$_settings.new" 2>/dev/null; then
        mv "$_settings.new" "$_settings"
      else
        rm -f "$_settings.new" "$_settings"
      fi
    }
    if [ ! -f "$WORKDIR/pid" ]; then
      forget_verifier
      exit 0
    fi
    PID=$(cat "$WORKDIR/pid")
    case "$PID" in
      '' | *[!0-9]*)
        forget_verifier
        exit 0
        ;;
    esac
    if ! kill "$PID" 2>/dev/null; then
      forget_verifier
      exit 0
    fi
    WAITED=0
    while [ "$WAITED" -lt 8 ]; do
      if ! kill -0 "$PID" 2>/dev/null; then
        forget_verifier
        exit 0
      fi
      sleep 1
      WAITED=$((WAITED + 1))
    done
    kill -9 "$PID" 2>/dev/null
    # ⚠ A killed process cannot write, but one being killed might still be; a
    # moment for the kernel to take it down before the file is read.
    sleep 1
    forget_verifier
    exit 0
    ;;

  *) cannot "unknown subcommand: $COMMAND" ;;
esac
