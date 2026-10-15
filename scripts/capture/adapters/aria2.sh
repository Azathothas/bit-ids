#!/bin/sh
# aria2.sh - the CLIENT-05 capture adapter.
#
# ⭐ THE SIMPLEST TARGET IN THE MATRIX, AND THAT IS WHY IT IS HERE FIRST AMONG
# EQUALS. aria2c takes every setting on its command line, needs no daemon, no
# profile directory and no interactive acceptance, and prints its version to
# stdout. Nothing about a run of it depends on state a previous run left behind,
# which is the property a capture host is built to guarantee and an adapter
# should not also have to.
#
# ⚠ CLIENT-05 ALSO WANTS aria2 AS A CONNECTOR, AND THIS IS NOT THAT. Driving
# aria2 as a target and using it to corroborate another target are the two roles
# the entry says must not create circular corroboration. This file is the target
# half alone; nothing here observes anything.
#
# -- ⛔ WHAT THIS ASSUMES ABOUT THE PRODUCT, AND WHAT IS NOW MEASURED ----------
#
# ⭐ MEASURED on 2026-09-08, against installed aria2 1.37.0 - both an Ubuntu
# package build and one compiled here from the vendor's release tarball.
# Installing a product and asking its version is not a capture, so it needed no
# disposable host:
#
#   * `aria2c --version` opens with `aria2 version 1.37.0`, so the third field
#     is the version - confirmed on both builds;
#   * the package route installs without a prompt, and on `ubuntu-24.04` it
#     installs NOTHING, because the image already ships aria2;
#   * the release route builds: 19 seconds to configure, 126 seconds to
#     `make -j4`, and the result answers with BitTorrent enabled.
#
# ⛔ STILL NOT MEASURED, because each needs a capture:
#
#   * `--enable-dht`, `--enable-dht6`, `--bt-enable-lpd` and
#     `--enable-peer-exchange` are the switches for the three adjacent surfaces;
#   * `--listen-port` refuses zero, which is why an unset peer port drops it;
#   * a torrent as a positional argument makes it announce to the tracker named
#     inside, needing no other control.
#
# ⚠ A claim here that turns out to be wrong shows up as a capture that refuses
# rather than as one that lies: `capture-client` will not attest to a build it
# did not observe announce.
#
# See adapters/README.md for the contract.
#
# Exit codes: 0 done, 1 refused, 2 could not run.

set -u

ME=aria2

refuse() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 1
}

cannot() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 2
}

# ⛔ THE INSTALLED EXECUTABLE, LOOKED UP ONCE. A path composed per subcommand is
# a second spelling, and the day a route installs somewhere else one subcommand
# finds the build and another reports it absent.
# ⛔ THE PREFIX IS ASKED FOR, NEVER COMPOSED. `bit-check install-rootless --prefix` is
# the one derivation in this tree, so this file cannot disagree with the
# installer about where a build went. ⚠ It defaulted to `/usr/local` until
# `ACQ-06`, which is a directory a runner needs a privilege to write.
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

binary() {
  if [ -n "${BIT_IDS_ARIA2:-}" ]; then
    printf '%s' "$BIT_IDS_ARIA2"
    return 0
  fi
  # ⭐ THE RELEASE ROUTE'S OWN PREFIX, BEFORE THE PATH. It lets a run point the
  # route somewhere else and still be measuring the build that route produced.
  # ⛔ Without it the two routes would race on PATH order, which is a property of
  # the host rather than of the acquisition.
  _prefix=$(prefix) || return 1
  if [ -x "$_prefix/bin/aria2c" ]; then
    printf '%s' "$_prefix/bin/aria2c"
    return 0
  fi
  command -v aria2c 2>/dev/null
}

[ $# -ge 1 ] || cannot "a subcommand is required"
COMMAND="$1"
shift

# What each route of this adapter delivers, as `Build.package` spells it.
#
# ⚠ The two differ, which is the point: the package route installs Ubuntu's
# `.deb`, whose `/usr/bin/aria2c` is a 14-kilobyte shim over `libaria2.so.0`,
# and the release route compiles the vendor's source tarball into a
# self-contained executable. `ACQ-03` measured both on 2026-09-08.
package_format() { # route
  case "$1" in
    package) printf 'deb\n' ;;
    release) printf 'elf-binary\n' ;;
    *) return 1 ;;
  esac
}

case "$COMMAND" in
  describe)
    printf 'target=aria2\n'
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
    # ⛔ AND WHAT THE RELEASE ROUTE NEEDS TO KNOW, DECLARED HERE BECAUSE THIS IS
    # ALREADY THE FILE THAT KNOWS HOW THIS PRODUCT IS INSTALLED. Which repository
    # publishes it, how it spells a version and which of a release's artifacts is
    # the installable one are all target knowledge, and `resolve-release` reads
    # them from here rather than from a table a workflow would have to carry a
    # case statement over.
    #
    # ⭐ `{version}` IS WHAT MAKES THE PATTERN UNAMBIGUOUS. Measured 2026-09-09
    # against the live listing: release-1.37.0 carries six assets, three of them
    # source archives differing only in compression, so `aria2-*.tar.*` matches
    # three and `select-asset` refuses. `aria2-{version}.tar.bz2` matches one.
    #
    # ⚠ bz2 RATHER THAN gz OR xz, and the choice is recorded rather than
    # implied: the three carry the same source, the bz2 is what `ACQ-03` already
    # measured a build from, and its digest is the one this project can check a
    # fetch against without acquiring anything new.
    printf 'release_repo=aria2/aria2\n'
    printf 'release_tag_prefix=release-\n'
    printf 'release_min_components=3\n'
    printf 'release_max_components=3\n'
    printf 'release_floor_major=-\n'
    printf 'release_asset=aria2-{version}.tar.bz2\n'
    # ⛔ THE DIGEST DISPOSITION, DECLARED ONCE AND READ BY THE ROUTE, AND IT IS
    # ABOUT THE VENDOR'S OWN CHECKSUMS DOCUMENT. ⚠ That is a narrower sentence
    # than "no digest", and the difference is measured: the release LISTING
    # carries a per-asset digest published by the party serving the bytes, which
    # `resolve-release.sh` resolves out of the same response and records as
    # `source-listing`. This key says only that the VENDOR uploaded no checksums
    # file - which is what an adapter can know by reading a release, and is what
    # decides whether the resolver goes looking for one.
    # ⚠ AND THIS TARGET IS THE ONE THAT REALLY HAS NEITHER. Read through
    # `AGENTS.md` rule 8's route on 2026-09-17: release-1.37.0 carries six
    # archives, no checksums asset, and `digest` is null on every one of the six
    # - the only one of this project's four targets for which that is true.
    printf 'release_digest_unpublished=this release publishes six archives and no checksums asset\n'
    # ⛔ THE ROUTE THAT COMPILES, SO THE WORKFLOW BOUNDS IT AS A BUILD. 145
    # seconds measured here, under the 420 a download gets, and a build still.
    printf 'compiles=release\n'
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
        # ⚠ The distribution's own index. Its version trails upstream, and that
        # is a packaging observation rather than a defect: ACQ-03 compares what
        # the two installed builds REPORT, and a divergence is recorded.
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
        DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=l apt-get install -y --no-install-recommends aria2 \
          </dev/null >"$WORKDIR/install.log" 2>&1 || refuse "the package route did not install aria2"
        ;;
      release)
        # ⛔ THE VENDOR'S OWN PUBLISHED ARTIFACT, AND NOT A SECOND PACKAGE INDEX.
        # E-ACQ-07 and E-ACQ-08 refuse two routes sharing a resolver or a
        # delivery mechanism, so a "second route" that asked another mirror of
        # the same index would be one route under two names.
        #
        # ⛔ AND IT INSTALLS WHAT IT FETCHED. This route used to `curl` the
        # tarball into the workdir and return 0, which is a route that reports an
        # install it did not perform: `version` then answered from whatever was
        # already on PATH, so a second route would have measured the FIRST
        # route's build and the two would have agreed for the most trivial reason
        # available. Measured on 2026-09-08 and recorded in `ACQ-03`.
        #
        # ⛔ AN aria2 RELEASE CARRIES NO LINUX BINARY, which is a property of the
        # target rather than a gap here: the 1.37.0 release publishes source
        # tarballs, two Windows zips and an Android build. So the Linux release
        # route is a source build. Measured on this project's own host: 19
        # seconds to configure and 126 seconds to `make -j4`.
        #
        # ⚠ WHAT THIS ASSUMES AND HAS NOT MEASURED ON A RUNNER: that a C++
        # toolchain and OpenSSL headers are already present. Both are true on the
        # host this was driven on. If they are not, `configure` refuses and says
        # so with its own log, which is a route that failed rather than one that
        # silently produced nothing. ⛔ It deliberately does NOT apt-get its build
        # dependencies: that would make this route reach the package index, and
        # arguing afterwards about whether headers count as acquisition is worse
        # than refusing.
        [ -n "${BIT_IDS_RELEASE_URL:-}" ] ||
          cannot "the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut"
        command -v curl >/dev/null 2>&1 || cannot "curl is not on this host"
        for _need in tar make cc; do
          command -v "$_need" >/dev/null 2>&1 ||
            cannot "the release route builds from source and $_need is not on this host"
        done
        PREFIX=$(prefix) || cannot "the rootless prefix could not be derived"
        # ⛔ THE FETCH AND THE DIGEST ARE ONE CALL AND IT IS NOT THIS FILE'S.
        # This route BUILDS what it fetches rather than placing it, so it asks
        # the installer for `--fetch` alone - the bounded transfer and the digest
        # disposition - and unpacks the settled file itself. ⚠ A route that
        # unpacked bytes whose digest nothing had settled would be running a
        # vendor's build system over something nobody identified.
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
          --into "$WORKDIR/aria2.tar.bz2" \
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
        mkdir -p "$WORKDIR/src" || cannot "cannot create $WORKDIR/src"
        tar -xjf "$WORKDIR/aria2.tar.bz2" -C "$WORKDIR/src" \
          >>"$WORKDIR/install.log" 2>&1 || refuse "the release tarball could not be unpacked"
        # ⚠ ONE DIRECTORY, FOUND RATHER THAN COMPOSED. The tarball's top-level
        # name carries the version, and composing it here would be a second
        # spelling of a value the archive already states.
        SRCDIR=$(find "$WORKDIR/src" -mindepth 1 -maxdepth 1 -type d | head -1)
        [ -n "$SRCDIR" ] || refuse "the release tarball unpacked no source directory"
        # ⚠ THE CONFIGURE OPTIONS ARE PART OF WHAT THIS ROUTE INSTALLS. The same
        # source configured differently is a build with different features, which
        # `ACQ-03` measured: two 1.37.0 builds reporting one version and enabling
        # different things. BitTorrent is the surface this project measures and it
        # is on by default; the rest are dependencies a capture does not need.
        # ⛔ `--prefix` puts the result ahead of the system one on PATH, which is
        # what makes `binary()` find THIS build rather than a package.
        (
          cd "$SRCDIR" &&
            ./configure --prefix="$PREFIX" --without-libxml2 --without-libexpat \
              --without-sqlite3 --without-libcares --without-libssh2 --with-openssl \
              --disable-nls
        ) </dev/null >>"$WORKDIR/install.log" 2>&1 ||
          refuse "the release route could not configure a build"
        JOBS=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2)
        (cd "$SRCDIR" && make -j"$JOBS") </dev/null >>"$WORKDIR/install.log" 2>&1 ||
          refuse "the release route could not build aria2"
        (cd "$SRCDIR" && make install) </dev/null >>"$WORKDIR/install.log" 2>&1 ||
          refuse "the release route built aria2 and could not install it"
        [ -x "$PREFIX/bin/aria2c" ] ||
          refuse "the release route reported an install and left no aria2c in $PREFIX/bin"
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
    BINARY=$(binary) || cannot "aria2c is not installed"
    [ -n "$BINARY" ] || cannot "aria2c is not installed"
    # ⛔ THE BUILD SPEAKING. `aria2 version 1.37.0` on its first line; the field
    # taken is the version alone, and an empty answer is could-not-run rather
    # than an empty string somebody downstream renders as unknown.
    # ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED. This
    # was `$(... | head -1) || cannot`, whose `||` reads HEAD's status: head
    # exits 0 over anything, so the guard was dead code and a build that refused
    # to answer reached the parse instead of the refusal. ⚠ Measured on
    # 2026-09-08, when a client capture reported "would not report a version" and
    # nothing said which of three refusals had fired. This repository's oldest
    # stated rule, broken in every adapter at once.
    OUTPUT=$("$BINARY" --version 2>/dev/null </dev/null)
    VERSION_RC=$?
    [ "$VERSION_RC" = 0 ] ||
      cannot "aria2c --version exited $VERSION_RC"
    LINE=$(printf '%s\n' "$OUTPUT" | head -1)
    VERSION=$(printf '%s' "$LINE" | awk '{ print $3 }')
    [ -n "$VERSION" ] || cannot "aria2c reported no parseable version: [$LINE]"
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
    BINARY=$(binary) || cannot "aria2c is not installed"
    [ -n "$BINARY" ] || cannot "aria2c is not installed"
    [ -s "$TORRENT" ] || cannot "there is no torrent at $TORRENT"
    mkdir -p "$WORKDIR/downloads" || cannot "cannot create $WORKDIR/downloads"

    # ⛔ ALL THREE ADJACENT SURFACES OFF, EXPLICITLY. A host with no default
    # route cannot reach a DHT bootstrap node, and it can still multicast a
    # local discovery announce and gossip over peer exchange. Turning off one
    # and inheriting the others is the one-gated-door shape.
    #
    # ⚠ --seed-time=0 and a stop timeout keep the process from outliving the
    # observer's own deadline: this returns while the build runs, and `stop`
    # below is what ends it, but a build that ignored both would hold the port
    # after the evidence was written.
    set -- \
      --enable-dht=false \
      --enable-dht6=false \
      --bt-enable-lpd=false \
      --enable-peer-exchange=false \
      --bt-require-crypto=false \
      --dir="$WORKDIR/downloads" \
      --seed-time=0 \
      --allow-overwrite=true \
      --console-log-level=info \
      --summary-interval=0

    # ⚠ ZERO IS NOT A LISTEN PORT aria2 ACCEPTS, so an unset peer port drops the
    # flag rather than passing a value the build would refuse. It is appended
    # here rather than written into the list above, because a list built twice
    # to vary one element is two lists that drift.
    [ "$PORT" = "0" ] || set -- "$@" --listen-port="$PORT"

    "$BINARY" "$@" "$TORRENT" >"$WORKDIR/client.out" 2>&1 &
    printf '%s\n' "$!" >"$WORKDIR/pid"
    printf 'started aria2c pid %s on port %s\n' "$(cat "$WORKDIR/pid")" "$PORT"
    ;;

  stop)
    [ $# -ge 1 ] || cannot "stop takes <workdir>"
    WORKDIR="$1"
    # ⚠ A stop over a build that already exited is done, not refused. aria2c
    # ends itself when its seed time expires, and a capture whose client
    # finished early is a capture, not a failure.
    [ -f "$WORKDIR/pid" ] || exit 0
    PID=$(cat "$WORKDIR/pid")
    case "$PID" in
      '' | *[!0-9]*) exit 0 ;;
    esac
    kill "$PID" 2>/dev/null || exit 0
    # ⚠ Given a moment, then killed. A client asked to stop writes its state and
    # closes its sockets; one that ignores the ask holds the port open into the
    # upload step.
    WAITED=0
    while [ "$WAITED" -lt 5 ]; do
      kill -0 "$PID" 2>/dev/null || exit 0
      sleep 1
      WAITED=$((WAITED + 1))
    done
    kill -9 "$PID" 2>/dev/null
    exit 0
    ;;

  *) cannot "unknown subcommand: $COMMAND" ;;
esac
