#!/bin/sh
# aria2-next.sh - the CLIENT-14 capture adapter.
#
# ⛔ A DIFFERENT PRODUCT FROM `aria2.sh`, NOT A VARIANT OF IT. `AnInsomniacy/aria2-next`
# keeps aria2's command-line vocabulary and does not keep its internals: this
# build links `libtorrent/2.1.1` and carries its own version string. Every value
# below was measured against the installed build on 2026-09-09 rather than
# copied from the adapter beside it, and three of those measurements contradict
# what copying would have produced.
#
# -- ⭐ WHAT WAS MEASURED, AND WHERE COPYING WOULD HAVE LIED ------------------
#
# Measured on 2026-09-09 against `aria2-next-2.7.4-linux-x86_64`, fetched from
# the project's own release and verified against the vendor's published
# `aria2-next-2.7.4-checksums.sha256` with `sha256sum -c` before it was run.
# Installing a product and asking its version is not a capture, so it needed no
# disposable host; nothing here was driven against a tracker.
#
#   ⛔ THE VERSION IS THE FOURTH FIELD, NOT THE THIRD. aria2 prints
#      `aria2 version 1.37.0` and this build prints `Aria2 Next version 2.7.4`.
#      `awk '{print $3}'`, which is what `aria2.sh` correctly does for ITS
#      product, returns the literal string `version` here. So the parse below is
#      anchored on the `version` token rather than on a column, because the
#      product name is prose whose word count is the vendor's to change.
#
#   ⛔ `--enable-dht6` DOES NOT EXIST AND IS ACCEPTED ANYWAY. The build logs
#      `Legacy aria2 input from command line: enable-dht6; accepted and skipped;
#      libtorrent uses one DHT switch for both families`. ⚠ A flag that is
#      accepted and skipped is worse than one that is refused: the step succeeds
#      and the switch it named was never set. `--enable-dht=false` covers both
#      families, and `start` VERIFIES that over RPC rather than trusting it.
#      ⚠ 2.8.6 still REPORTED `enable-dht6` in its options and 3.0.2 retires
#      it, so `start` no longer requires a key the build stopped having; the
#      family sweep would hold it to false if a build reported it again.
#
#   ⭐ RPC ANSWERS `version` BETTER THAN THE COMMAND LINE DOES.
#      `aria2.getVersion` returns `{"product":"aria2-next","version":"2.7.4",...}`
#      - a structured field, with no prose to parse. ⛔ It is still NOT what
#      `version` below uses, and that is the contract rather than an oversight:
#      `install-client` calls `version` BEFORE the route runs, on a host where
#      nothing is installed and no daemon exists. An adapter whose `version`
#      started one would be changing the host at the one moment the caller is
#      trying to observe it unchanged. RPC cannot answer a question asked of a
#      host with nothing running, so this is exactly the case CLIENT-14's
#      "wherever RPC answers the same question" clause excludes.
#
#   ⛔ AND THE FACT THIS PROJECT EXISTS TO CATCH: the build's default
#      `bt-peer-id-prefix` is `-qB5230-`, read back over `aria2.getGlobalOption`.
#      A stock aria2-next presents itself on the wire as qBittorrent 5.2.3.0.
#      ⚠ NOTHING HERE OVERRIDES IT. The identity a stock build emits is the
#      measurement; an adapter that set a prefix would be publishing this
#      project's own string and calling it an observation. A capture is what
#      establishes what actually goes on the wire, and this note is a reason to
#      run one rather than a result standing in for one.
#
# ⛔ THERE IS NO PACKAGE ROUTE AND THAT IS MEASURED, NOT ASSUMED. `apt-cache
# search aria2` on `ubuntu-24.04` lists `aria2`, `libaria2-0`, `libaria2-0-dev`
# and `persepolis`, and no fork. ⚠ The dangerous failure is not the missing
# package: it is that `apt-get install aria2` SUCCEEDS and installs a DIFFERENT
# PRODUCT, after which `version` would answer 1.37.0 and the record would carry
# a second route that acquired somebody else's build. The package route below
# refuses by name for that reason.
#
# ⚠ SO THIS TARGET HAS ONE ROUTE TODAY, and `E-ACQ-01` refuses a record with
# one. CLIENT-14 carries that as the open question it is; this file does not
# paper over it by declaring a route it cannot take.
#
# See adapters/README.md for the contract.
#
# Exit codes: 0 done, 1 refused, 2 could not run.

set -u

ME=aria2-next

# ⚠ THE RPC PORT IS A DEFAULT THIS FILE STATES, not one inherited from the
# product. aria2's own default is 6800; a capture host runs one build at a time,
# and an override exists so a host already using that port is a configuration
# rather than a collision nobody can see.
RPC_PORT="${BIT_IDS_ARIA2_NEXT_RPC_PORT:-6800}"

refuse() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 1
}

cannot() {
  printf '%s: %s\n' "$ME" "$1" >&2
  exit 2
}

# ⛔ THE PREFIX IS ASKED FOR, NEVER COMPOSED. `bit-check install-rootless --prefix` is
# the one derivation in this tree, so this file cannot disagree with the
# installer about where a build went. ⚠ It used to default to `/usr/local`,
# which needs a privilege a runner spends and a session host does not; `ACQ-06`
# is the entry that removed it.
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

# ⛔ THE INSTALLED EXECUTABLE, LOOKED UP ONCE. A path composed per subcommand is
# a second spelling, and the day a route installs somewhere else one subcommand
# finds the build and another reports it absent.
binary() {
  if [ -n "${BIT_IDS_ARIA2_NEXT:-}" ]; then
    printf '%s' "$BIT_IDS_ARIA2_NEXT"
    return 0
  fi
  _prefix=$(prefix) || return 1
  if [ -x "$_prefix/bin/aria2-next" ]; then
    printf '%s' "$_prefix/bin/aria2-next"
    return 0
  fi
  command -v aria2-next 2>/dev/null
}

# ⭐ ONE JSON-RPC CALL, AND THE THREE STREAMS STAY APART. The response body goes
# to a file the caller names, curl's own diagnosis to another, and the exit code
# comes back from curl itself rather than from a pipeline.
#
# ⛔ `--noproxy` IS NOT OPTIONAL. This is a loopback call to a daemon on this
# host; a proxy in the environment would send a capture's own control channel off
# the machine, which on a contained host is the one thing that must not happen.
rpc() { # method params-json body-file
  _method="$1"
  _params="$2"
  _body="$3"
  curl -sS --noproxy '*' --max-time 20 \
    -X POST "http://127.0.0.1:$RPC_PORT/jsonrpc" \
    -H 'Content-Type: application/json' \
    -d "{\"jsonrpc\":\"2.0\",\"id\":\"bit-ids\",\"method\":\"$_method\",\"params\":$_params}" \
    -o "$_body" 2>>"${RPC_ERR:-/dev/null}"
}

# ⚠ A JSON-RPC ERROR ARRIVES WITH HTTP 200 AND AN `error` MEMBER. A caller
# reading only curl's status would take a refusal for an answer, which is the
# shape adapters/README.md warns about: an adapter failure and a product refusal
# reach this file through one channel and must not leave it as one status.
rpc_failed() { # body-file
  grep -q '"error"' "$1" 2>/dev/null
}

[ $# -ge 1 ] || cannot "a subcommand is required"
COMMAND="$1"
shift

# What each route of this adapter delivers, as `Build.package` spells it.
#
# ⭐ Both routes install a bare executable and neither installs a package: the
# release asset IS the binary, measured at 1.2 seconds for a download and a
# chmod, and the source route compiles one. ⚠ The `package` route refuses by
# name, so it reaches nothing here.
package_format() { # route
  case "$1" in
    release | source) printf 'elf-binary\n' ;;
    *) return 1 ;;
  esac
}

case "$COMMAND" in
  describe)
    printf 'target=aria2-next\n'
    printf 'kind=stock\n'
    _where=$(binary) || _where=""
    [ -z "$_where" ] || printf 'binary=%s\n' "$_where"
    # ⭐ THE RELEASE DECLARATIONS, MEASURED AGAINST THE LIVE LISTING on
    # 2026-09-09. Tags are `v2.7.4`, so the prefix is `v` and a version is three
    # components. A release publishes eight assets - seven builds over four
    # platforms, Android arm64 only and the other three in two architectures,
    # plus a checksums file - and exactly one of them is a Linux x86-64 build, so
    # the pattern needs no wildcard beyond `{version}`.
    #
    # ⚠ AND THE ASSET IS A BINARY, WHICH IS THE ONE PLACE THIS TARGET IS EASIER
    # THAN `aria2`. That project's release carries no Linux binary at all, so its
    # release route is a source build taking minutes; this one is a download and
    # a `chmod`, measured at 1.2 seconds.
    printf 'release_repo=AnInsomniacy/aria2-next\n'
    printf 'release_tag_prefix=v\n'
    printf 'release_min_components=3\n'
    printf 'release_max_components=3\n'
    printf 'release_floor_major=-\n'
    printf 'release_asset=aria2-next-{version}-linux-x86_64\n'
    # ⭐ THE VENDOR PUBLISHES A DIGEST AND THIS IS WHERE IT SAYS SO. Its release
    # carries `aria2-next-{version}-checksums.sha256` beside the seven builds,
    # read out of the live listing on 2026-09-09 and still in the fixture, so
    # the release route is verified against the vendor rather than merely
    # recorded. `bit-check resolve-release` selects it with the same call that selects
    # the artifact, from the same response.
    printf 'release_digest_asset=aria2-next-{version}-checksums.sha256\n'
    # ⛔ THE ROUTE THAT COMPILES, SO THE WORKFLOW BOUNDS IT AS A BUILD: 870
    # seconds for `v3.0.2` on the session host. The release route downloads.
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
        # ⛔ REFUSED BY NAME, AND THE REFUSAL IS THE MEASUREMENT. No package
        # index reachable from a capture host carries this fork; what they carry
        # is `aria2`, a different product at a different version. An
        # `apt-get install aria2` here would exit 0, install somebody else's
        # build, and hand `ACQ-03` a second route that agreed with nothing.
        cannot "aria2-next is published only through its own releases; \
no package index carries it, and installing aria2 would acquire a different product"
        ;;
      release)
        # ⭐ THE VENDOR'S OWN BUILD, INSTALLED RATHER THAN MERELY FETCHED. Every
        # release route in this tree once dropped its artifact in the workdir and
        # returned 0, so `version` answered from whatever was already on PATH.
        # This one puts the executable where `binary()` looks and then asks the
        # file whether it is there.
        [ -n "${BIT_IDS_RELEASE_URL:-}" ] ||
          cannot "the release route needs BIT_IDS_RELEASE_URL, resolved before the route was cut"
        # ⛔ THE FETCH, THE DIGEST AND THE PLACING ARE ONE CALL, AND IT IS NOT
        # THIS FILE'S. `bit-check install-rootless` bounds the transfer, settles the
        # digest before anything is made executable, writes into a prefix this
        # user already owns, and prints the path it placed. ⚠ This route used to
        # `curl`, `chmod +x` and `cp` into `/usr/local/bin` itself, which is four
        # adapters each carrying its own copy of a rule and each needing `sudo`
        # to finish. `ACQ-06` carries what that privilege cost.
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
        rootless --install \
          --url "$BIT_IDS_RELEASE_URL" \
          --into "$WORKDIR/aria2-next" \
          --as aria2-next \
          "$@" \
          --report "$WORKDIR/rootless-install.txt" \
          </dev/null >>"$WORKDIR/install.log" 2>&1
        _rc=$?
        [ "$_rc" = 0 ] || {
          tail -20 "$WORKDIR/install.log" >&2
          [ "$_rc" = 1 ] ||
            cannot "the rootless installer could not run for the release route (exit $_rc)"
          refuse "the release route could not install the vendor's build"
        }
        ;;
      source)
        # ⭐ THE SECOND ROUTE, AND IT EXISTS BECAUSE THIS TARGET HAS NO PACKAGE.
        # `E-ACQ-01` refuses a record built from one route, and absolute 4 wants
        # two routes resolving the SAME stable version. Measured on 2026-09-09:
        # this build from tag `v2.7.5` answers `Aria2 Next version 2.7.5`, the
        # same version the release asset reports, from a 256-megabyte
        # `RelWithDebInfo` executable where the published one is 14 megabytes
        # stripped. ⭐ One version, two provably different builds, which is the
        # case `ACQ-03` exists to classify.
        #
        # ⛔ THESE TWO ROUTES DIFFER IN DELIVERY AND NOT IN RESOLVER, AND THIS
        # COMMENT SAID OTHERWISE UNTIL 2026-09-09. It claimed they "differ in
        # resolver - the releases API against git refs"; the artifacts refute it.
        # This route WAS handed `BIT_IDS_RELEASE_TAG` by *Resolve the release
        # artifact* - the same step and the same listing the release lane reads,
        # measured on capture-client run 14, whose two `release/resolution.txt`
        # differ only in their timestamps. ⭐ It resolves its own tag now.
        # ⛔ `E-ACQ-07` calls two routes sharing a resolver ONE route, so the
        # pair cannot become a record until this route resolves its own tag.
        # `CI-09` carries the measurement and the argument; what a fix needs is a
        # resolution from git refs here, with the two versions COMPARED after
        # installation rather than made equal beforehand.
        # ⛔ They also do NOT differ in origin: whoever controls the repository
        # controls both. That is weaker than a distribution index against a
        # vendor release, and the record says so rather than letting two green
        # checks imply otherwise.
        #
        # ⚠ MEASURED COST: about 350 seconds for configure and build together on
        # a four-core host for 2.7.5, and 870 for `v3.0.2` on the session host
        # on 2026-10-14, whose superbuild compiles ten dependencies first, among
        # them OpenSSL, curl, libtorrent, FFmpeg and GPAC.
        # The caller's bound is `capture-client.yml`'s source-lane
        # `BIT_IDS_INSTALL_TIMEOUT`, stated there alone. It builds the project's
        # test binaries too, because `--preset default` is what the project's
        # own README and CI run and a narrower target would be this repository
        # inventing a build the vendor does not document.
        # ⛔ ITS OWN TAG, FROM ITS OWN RESOLUTION. This read
        # `BIT_IDS_RELEASE_TAG` until 2026-09-09, which is the tag the RELEASE
        # lane's listing chose - so both lanes resolved through one index and
        # `E-ACQ-07` called them one route. `bit-check resolve-source` reads the
        # repository's refs instead, and the two versions are then COMPARED
        # after installation rather than made equal beforehand, which is what
        # absolute 4 asks for.
        # ⚠ `BIT_IDS_RELEASE_TAG` is still accepted as a fallback so a caller
        # that has only the older variable is refused by name rather than
        # silently building whatever `main` points at.
        _tag=${BIT_IDS_SOURCE_TAG:-${BIT_IDS_RELEASE_TAG:-}}
        [ -n "$_tag" ] ||
          cannot "the source route needs BIT_IDS_SOURCE_TAG, resolved from this repository's own refs before the route was cut"
        for _need in git cmake ninja c++; do
          command -v "$_need" >/dev/null 2>&1 ||
            cannot "the source route builds from source and $_need is not on this host"
        done
        # ⛔ THE SAME PREFIX THE RELEASE ROUTE USES, ASKED THE SAME WAY. Two
        # routes installing to two places would make `binary()` answer about
        # whichever ran last rather than about the route being measured.
        PREFIX=$(prefix) || cannot "the rootless prefix could not be derived"
        mkdir -p "$PREFIX/bin" ||
          refuse "this user cannot create $PREFIX/bin; a rootless install writes nowhere else"
        # ⚠ THE REPOSITORY IS THE ONE `describe` ALREADY NAMES, so the two routes
        # cannot drift onto different upstreams. ⛔ A shallow clone of one tag:
        # the history is not the measurement and fetching it would be minutes of
        # network for bytes nothing reads.
        # ⛔ ASKED OF THIS ADAPTER'S OWN `describe` RATHER THAN SPELLED
        # AGAIN. It was a literal here and a literal in `describe`, which is
        # one fact in two places: the day one moved, the release route and the
        # source route would have acquired from two different upstreams and
        # every check here would have passed.
        _repo=$(sh "$0" describe | awk -F= '$1 == "release_repo" { sub(/^[^=]*=/, ""); print; exit }')
        [ -n "$_repo" ] || cannot "this adapter declares no release_repo to clone from"
        # ⛔ BOUNDED, for the same reason the release fetch above is: a clone
        # that stalls has no limit of its own and waits forever. ⚠ `git` has no
        # `--max-time`, so the bound is `timeout` around it - which is the
        # spelling `shell.md` section 9 gives for a tool that does not bound
        # itself. ⛔ 600 rather than 300: this clone is the input to a build the
        # workflow gives a source lane many minutes for, so a bound sized like
        # the release fetch's would refuse a source lane that was working.
        # ⚠ Exit 124 is `timeout`'s verdict and is reported as a refusal naming
        # the bound, never as a clone that failed for its own reasons.
        timeout 600 git clone --depth 1 --branch "$_tag" \
          "https://github.com/$_repo.git" "$WORKDIR/src" \
          </dev/null >"$WORKDIR/install.log" 2>&1
        _clone=$?
        [ "$_clone" = 0 ] || {
          [ "$_clone" != 124 ] ||
            refuse "the source route's clone of $_tag did not finish within 600s"
          refuse "the source route could not clone $_tag"
        }
        # ⛔ THE COMMIT IS WHAT `E-ACQ-06` ASKS FOR AND A TAG IS NOT IT. A tag is
        # a name somebody can move; `SourceIdentity::SourceCommit` takes a full
        # object name and refuses an abbreviation. Measured on 2026-09-09 by
        # assembling run 14: nothing the capture path wrote carried one, so a
        # source route could not become a record at all, and the object name
        # existed only inside the git chatter in `install.log`. ⚠ It is
        # `rev-parse HEAD` rather than the tag's own object, because
        # `v2.7.5` is an annotated tag - the clone said `is not a commit!` and
        # checked out what it points at, which is what was built.
        git -C "$WORKDIR/src" rev-parse HEAD >"$WORKDIR/source-commit" 2>>"$WORKDIR/install.log" ||
          refuse "the source route cloned $_tag and could not name its commit"
        # ⛔ ONE FLAG THE VENDOR DOES NOT PASS, AND IT IS A HEADER ITS SOURCE
        # LEFT OUT. Measured on `capture-client` run 35, 2026-10-14: `v3.0.2`
        # stops in `src/disk/File.cc`, which calls `std::exchange` and never
        # includes `<utility>`. The vendor builds on `ubuntu-22.04`, and its
        # release asset's `.comment` reads GCC 11.4.0; this route builds where
        # the capture runs, on GCC 13, which refuses the file. ⚠ So it passes
        # `-include utility`: the header the compiler's own diagnostic names,
        # holding a declaration the source already calls, with no file the
        # vendor wrote changed. The superbuild forwards `CMAKE_CXX_FLAGS` to
        # the inner project and its seven CMake-built dependencies; OpenSSL,
        # FFmpeg and GPAC configure themselves and do not receive it.
        # ⛔ SET, NOT APPENDED TO, so the configure never carries a host's own
        # `CXXFLAGS` into those caches. ⚠ `ACQ-03` says a source route's
        # configure options belong to its identity and that nothing records
        # them; the line goes into `source-configure.txt` beside the build,
        # which the install artifact uploads, and the record still has no field
        # for it.
        # ⚠ The line is quoted so it replays as it ran, and the host's own C,
        # preprocessor and linker flags are written beside it: the superbuild
        # passes those on to its dependencies, and they come from whatever host
        # the build is on.
        _cxxflags='-include utility'
        {
          printf "CXXFLAGS='%s' cmake --preset default\n" "$_cxxflags"
          printf 'host CFLAGS=%s\n' "${CFLAGS-<unset>}"
          printf 'host CPPFLAGS=%s\n' "${CPPFLAGS-<unset>}"
          printf 'host LDFLAGS=%s\n' "${LDFLAGS-<unset>}"
        } >"$WORKDIR/source-configure.txt" ||
          refuse "the source route could not record how it configured the build"
        (cd "$WORKDIR/src" && CXXFLAGS="$_cxxflags" cmake --preset default) \
          </dev/null >>"$WORKDIR/install.log" 2>&1 ||
          refuse "the source route could not configure a build"
        (cd "$WORKDIR/src" && cmake --build --preset default) \
          </dev/null >>"$WORKDIR/install.log" 2>&1 ||
          refuse "the source route could not build aria2-next"
        # ⚠ THE BUILT PATH IS THE PRESET'S, found rather than composed twice: the
        # `default` preset writes into `build/default` and that is where the
        # project's own CI reads its binary from.
        [ -x "$WORKDIR/src/build/default/aria2-next" ] ||
          refuse "the source route built nothing at build/default/aria2-next"
        cp "$WORKDIR/src/build/default/aria2-next" "$PREFIX/bin/.aria2-next.new" ||
          refuse "the source route could not stage its build in $PREFIX/bin"
        mv "$PREFIX/bin/.aria2-next.new" "$PREFIX/bin/aria2-next" ||
          refuse "the source route could not install into $PREFIX/bin"
        [ -x "$PREFIX/bin/aria2-next" ] ||
          refuse "the source route reported an install and left no aria2-next in $PREFIX/bin"
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
    BINARY=$(binary) || cannot "aria2-next is not installed"
    [ -n "$BINARY" ] || cannot "aria2-next is not installed"
    # ⛔ THE EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED.
    OUTPUT=$("$BINARY" --version 2>/dev/null </dev/null)
    VERSION_RC=$?
    [ "$VERSION_RC" = 0 ] || cannot "aria2-next --version exited $VERSION_RC"
    LINE=$(printf '%s\n' "$OUTPUT" | head -1)
    # ⛔ ANCHORED ON THE `version` TOKEN, NOT ON A COLUMN. Measured: this build's
    # first line is `Aria2 Next version 2.7.4`, so the third field is the word
    # `version` and the fourth is the number. A column index here would have
    # published the string `version` as this product's version, and it would have
    # validated: it is a non-empty field.
    VERSION=$(printf '%s' "$LINE" |
      awk '{ for (i = 1; i < NF; i++) if ($i == "version") { print $(i + 1); exit } }')
    [ -n "$VERSION" ] || cannot "aria2-next reported no parseable version: [$LINE]"
    # ⭐ WHAT THE BUILD SAID ABOUT ITSELF IS KEPT, not only the field parsed out
    # of it. This build's own answer names `libtorrent/2.1.1` among its
    # libraries, which is a fact about what it will put on the wire that no
    # version number carries.
    printf '%s reported:\n' "$ME" >&2
    printf '%s\n' "$OUTPUT" >&2
    printf '%s\n' "$VERSION"
    ;;

  start)
    [ $# -ge 3 ] || cannot "start takes <torrent> <workdir> <peer-port>"
    TORRENT="$1"
    WORKDIR="$2"
    PORT="$3"
    BINARY=$(binary) || cannot "aria2-next is not installed"
    [ -n "$BINARY" ] || cannot "aria2-next is not installed"
    [ -s "$TORRENT" ] || cannot "there is no torrent at $TORRENT"
    mkdir -p "$WORKDIR/downloads" || cannot "cannot create $WORKDIR/downloads"
    command -v curl >/dev/null 2>&1 || cannot "curl is not on this host"
    command -v base64 >/dev/null 2>&1 || cannot "base64 is not on this host"
    RPC_ERR="$WORKDIR/rpc.err"

    # ⛔ A PER-RUN TOKEN, GENERATED HERE AND WRITTEN TO RUN STATE. A literal in
    # this file would be a shared secret in the tree, which rule 12 refuses; a
    # daemon with no token would accept an RPC call from anything that reached
    # the port.
    #
    # ⛔ AND `stop` DELETES IT, BECAUSE THE WORKDIR IS THE EVIDENCE BUNDLE.
    # Measured on capture-client run 11: this file shipped inside the uploaded
    # artifact as `client/rpc-token`, which is rule 12's "a secret never enters
    # ... artifacts" in the one place no reading had looked - the run wrote it,
    # the run bundled the directory it was in, and nothing between them was
    # wrong. ⚠ A short-lived token for a loopback port on a host that is about to
    # be destroyed is a weak secret, and rule 12 does not grade them.
    #
    # ⚠ 077 SO IT IS NEVER WORLD-READABLE EVEN WHILE IT EXISTS, and the umask is
    # restored straight after: a subshell would not do, because the file has to
    # outlive it.
    TOKEN=$(od -An -tx1 -N16 /dev/urandom 2>/dev/null | tr -d ' \n')
    [ -n "$TOKEN" ] || cannot "cannot generate an RPC token"
    _oldmask=$(umask)
    umask 077
    printf '%s\n' "$TOKEN" >"$WORKDIR/rpc-token" ||
      cannot "cannot record the RPC token"
    umask "$_oldmask"
    printf '%s\n' "$RPC_PORT" >"$WORKDIR/rpc-port" ||
      cannot "cannot record the RPC port"

    # ⛔ FOUR DISCOVERY SURFACES OFF, NOT THE THREE THE CONTRACT NAMES.
    # adapters/README.md says an adapter disables DHT, peer exchange and local
    # peer discovery; this product has a FOURTH. `bt-port-mapping` is
    # "Enable UPnP and NAT-PMP port mapping" and it defaults to TRUE, which was
    # not read off the help text but found by driving the build on 2026-09-09 and
    # reading its own sockets out of `/proc`: with the three documented switches
    # off, the process was still bound to **UDP 1900**, the SSDP multicast port.
    # ⚠ A contained host has no default route, so a tracker or a DHT bootstrap
    # fails - and a multicast to 239.255.255.250 is link-local and does not. This
    # is the one-gated-door shape the README calls the most recurring hole there
    # is, found in the file that quotes it.
    #
    # ⛔ AND A FIFTH, ON THE SECOND PROTOCOL THIS BUILD SPEAKS. 3.0.0's release
    # notes add `--ed2k-port-mapping`, "forwards the ED2K ports through UPnP
    # IGD", and the options dump `capture-client` run 35 uploaded on 2026-10-14,
    # from the bundle of a 3.0.2 capture this guard refused, has it `true`: the
    # class `bt-port-mapping` is switched off for, on the ED2K ports. ⚠ A fixed
    # list of switches is a guard that sees the spellings it was written with,
    # so the read-back below also sweeps every switch of those families the
    # build reports, named here or not.
    #
    # ⛔ AND `--enable-dht6` IS DELIBERATELY ABSENT. Measured: this build does not
    # have that option, accepts it from the command line anyway and skips it with
    # a warning, because libtorrent uses one DHT switch for both families.
    # Passing it would put a flag in the record that set nothing.
    # `--enable-dht=false` is what covers both, and the verification below is
    # what establishes that it did rather than assuming it.
    #
    # ⛔ AND `--bt-require-crypto` IS GONE, BECAUSE IT SET NOTHING. 3.0.2 maps
    # it onto `bt-encryption`, and only a value other than `false` sets it. What
    # applies is the stock `bt-encryption`, `preferred`, under which libtorrent
    # prefers an encrypted handshake on the connections it opens; that is the
    # product's default and the capture's to observe, so nothing here sets it.
    #
    # ⛔ THE PEER PORT IS PASSED HERE, ON THE COMMAND LINE, and zero still means
    # "leave it": aria2 refuses a listen port of zero, so an unset peer port
    # drops the flag rather than sending a value the build would reject.
    #
    # ⛔ RPC IS BOUND TO LOOPBACK. `--rpc-listen-all=false` is the difference
    # between a control channel and a service on the capture host's network.
    set -- \
      --enable-rpc=true \
      --rpc-listen-port="$RPC_PORT" \
      --rpc-listen-all=false \
      --rpc-secret="$TOKEN" \
      --enable-dht=false \
      --bt-enable-lpd=false \
      --enable-peer-exchange=false \
      --bt-port-mapping=false \
      --ed2k-port-mapping=false \
      --dir="$WORKDIR/downloads" \
      --seed-time=0 \
      --allow-overwrite=true \
      --console-log-level=info \
      --summary-interval=0
    [ "$PORT" = "0" ] || set -- "$@" --listen-port="$PORT"

    "$BINARY" "$@" >"$WORKDIR/client.out" 2>&1 &
    printf '%s\n' "$!" >"$WORKDIR/pid"
    PID=$(cat "$WORKDIR/pid")

    # ⚠ A BOUNDED WAIT ON THE WORK, NOT ON THE CLOCK. The daemon is up when it
    # answers, so this asks it; a fixed sleep would be too short on a loaded
    # runner and wasted everywhere else. conventions/shell.md section 10.
    WAITED=0
    until rpc aria2.getVersion "[\"token:$TOKEN\"]" "$WORKDIR/rpc-version.json" &&
      ! rpc_failed "$WORKDIR/rpc-version.json"; do
      # ⛔ A DEAD DAEMON IS REPORTED AS ONE, rather than waited out. A build that
      # refused its own options exits immediately, and without this the step
      # would spend the whole deadline discovering it.
      kill -0 "$PID" 2>/dev/null ||
        refuse "aria2-next exited before its RPC server answered; $WORKDIR/client.out says why"
      WAITED=$((WAITED + 1))
      [ "$WAITED" -lt 30 ] ||
        refuse "aria2-next did not answer RPC on port $RPC_PORT within 30 seconds"
      sleep 1
    done

    # ⭐ THE CONTAINMENT SWITCHES ARE READ BACK FROM THE BUILD, WHICH IS THE ONE
    # QUESTION RPC ANSWERS AND THE COMMAND LINE CANNOT. A command line reports
    # what was PASSED; `aria2.getGlobalOption` reports what took EFFECT. This
    # target accepts an option it does not have and skips it, so the difference
    # between those two is not theoretical here - it is the measured behaviour of
    # `--enable-dht6`.
    rpc aria2.getGlobalOption "[\"token:$TOKEN\"]" "$WORKDIR/rpc-options.json" ||
      refuse "aria2-next would not report its global options"
    ! rpc_failed "$WORKDIR/rpc-options.json" ||
      refuse "aria2-next refused to report its global options"
    # ⛔ EVERY DISCOVERY SWITCH THIS FILE PASSES IS REPORTED, AND FALSE. A switch
    # the build stopped reporting is one nothing here can show took effect.
    for _switch in enable-dht bt-enable-lpd enable-peer-exchange bt-port-mapping ed2k-port-mapping; do
      grep -q "\"$_switch\":\"false\"" "$WORKDIR/rpc-options.json" ||
        refuse "aria2-next reports $_switch is not false; a discovery surface is live"
    done
    # ⛔ AND EVERY KEY OF THOSE FAMILIES THE BUILD REPORTS, NAMED OR NOT, reads
    # `false` too. `ed2k-port-mapping` was on in 3.0.2 and named nowhere.
    # ⚠ The KEYS are what is taken off the dump, `"key":` whatever follows it,
    # and each is then asked for the exact pair: a value this parse cannot read
    # refuses rather than passing unread. ⛔ A here-document, not a pipe: a loop
    # fed by a pipe runs in a subshell, where `refuse` would end the loop and
    # not this adapter.
    while read -r _key; do
      [ -n "$_key" ] || continue
      grep -q "\"$_key\":\"false\"" "$WORKDIR/rpc-options.json" ||
        refuse "aria2-next reports $_key and not as false; a discovery surface is live"
    done <<KEYS
$(grep -o -E '"[^"]+":' "$WORKDIR/rpc-options.json" | tr -d '":' |
      grep -E '^(enable-dht[a-z0-9-]*|[a-z0-9-]*port-mapping|[a-z0-9-]*-lpd|[a-z0-9-]*-lsd|[a-z0-9-]*peer-exchange)$')
KEYS
    # ⛔ AND A SWITCH THAT IS ON IS ONE THIS FILE HAS JUDGED. A family is still
    # a set of spellings, and the next surface need not share one: libtorrent
    # calls local discovery LSD, and 3.0.0's notes say its ED2K speaks Kad. So
    # every option the build reports as `true` must be named below, each read
    # off 3.0.2's own dump and none a discovery surface, and a new default
    # reported as `true` refuses whatever it is called. ⚠ A surface a build
    # turns on with another value, such as a port or `auto`, is not caught here.
    _harmless=' allow-overwrite auto-file-renaming check-certificate enable-color enable-http-keep-alive enable-rpc follow-metalink follow-torrent http-accept-gzip keep-unfinished-download-result rpc-save-upload-metadata save-not-found show-console-readout '
    while read -r _key; do
      [ -n "$_key" ] || continue
      case "$_harmless" in
        *" $_key "*) ;;
        *) refuse "aria2-next reports $_key as true, and this file has not judged it; a discovery surface may be live" ;;
      esac
    done <<KEYS
$(grep -o -E '"[^"]+": *"?true' "$WORKDIR/rpc-options.json" | sed -E 's/^"([^"]+)".*/\1/')
KEYS

    # ⛔ AND THE PORT THE OBSERVER IS WATCHING IS READ BACK FROM THE BUILD. A
    # peer port that silently did not take is a capture watching a port nothing
    # is on, and it would look exactly like a client that never announced.
    if [ "$PORT" != "0" ]; then
      grep -q "\"listen-port\":\"$PORT\"" "$WORKDIR/rpc-options.json" ||
        refuse "aria2-next reports a listen port other than $PORT"
    fi

    # ⭐ THE TORRENT GOES IN OVER RPC, base64-ENCODED, WHICH IS THE CASE
    # CLIENT-14 IS ABOUT. `aria2.addTorrent` takes the torrent's bytes rather
    # than a path, so the build is handed the same object the observer wrote
    # instead of a filename it re-reads. ⚠ base64 is the one encoding no shell
    # interprets - conventions/shell.md section 1 - and it is what this method
    # already requires, so nothing is being encoded twice.
    B64=$(base64 -w0 <"$TORRENT" 2>/dev/null) || B64=$(base64 <"$TORRENT" | tr -d '\n')
    [ -n "$B64" ] || cannot "the torrent at $TORRENT could not be encoded"
    rpc aria2.addTorrent "[\"token:$TOKEN\",\"$B64\"]" "$WORKDIR/rpc-add.json" ||
      refuse "aria2-next would not accept the torrent over RPC"
    ! rpc_failed "$WORKDIR/rpc-add.json" ||
      refuse "aria2-next refused the torrent: $(cat "$WORKDIR/rpc-add.json")"
    GID=$(sed -n 's/.*"result":"\([^"]*\)".*/\1/p' "$WORKDIR/rpc-add.json")
    [ -n "$GID" ] || refuse "aria2-next accepted the torrent and named no download"
    printf '%s\n' "$GID" >"$WORKDIR/gid"
    printf 'started aria2-next pid %s on port %s, download %s, RPC %s\n' \
      "$PID" "$PORT" "$GID" "$RPC_PORT"
    ;;

  stop)
    [ $# -ge 1 ] || cannot "stop takes <workdir>"
    WORKDIR="$1"
    # ⚠ A stop over a build that already exited is done, not refused.
    [ -f "$WORKDIR/pid" ] || exit 0
    PID=$(cat "$WORKDIR/pid")
    case "$PID" in
      '' | *[!0-9]*) exit 0 ;;
    esac
    RPC_ERR="$WORKDIR/rpc.err"

    # ⭐ THE PRODUCT'S OWN SHUTDOWN FIRST. `aria2.shutdown` is the same question a
    # signal asks and a better-behaved answer to it: the build closes its
    # sockets, writes its session state and exits on its own terms. ⚠ The signal
    # path below is kept, because a build that ignores its own RPC is exactly the
    # case a capture must still be able to end.
    if [ -f "$WORKDIR/rpc-token" ]; then
      TOKEN=$(cat "$WORKDIR/rpc-token")
      [ ! -f "$WORKDIR/rpc-port" ] || RPC_PORT=$(cat "$WORKDIR/rpc-port")
      rpc aria2.shutdown "[\"token:$TOKEN\"]" "$WORKDIR/rpc-shutdown.json" || :
      # ⛔ GONE BEFORE ANYTHING BUNDLES THIS DIRECTORY, and gone whether or not
      # the shutdown was accepted. `capture-client` calls `stop` and then packs
      # the workdir, so this unlink is the whole of what keeps the token out of
      # the artifact. ⚠ It is removed HERE rather than at the end of `stop`,
      # because every path below this line can exit.
      rm -f "$WORKDIR/rpc-token"
      # ⚠ TEN SECONDS, AND THE NUMBER IS MEASURED RATHER THAN ROUND. Driven on
      # 2026-09-09: `aria2.shutdown` answered `OK` and the build was still alive
      # five seconds later, so a five-second grace fell through to a signal every
      # time and the graceful path this branch exists for never once completed.
      WAITED=0
      while [ "$WAITED" -lt 10 ]; do
        kill -0 "$PID" 2>/dev/null || exit 0
        sleep 1
        WAITED=$((WAITED + 1))
      done
    fi

    kill "$PID" 2>/dev/null || exit 0
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
