# Engine entries

Engine results are labeled harness observations. They must never be silently
substituted for the behavior of an application that embeds the engine.

## ENGINE-01: libtorrent engine matrix

Source: operator scope and libtorrent upstream
Priority: P1 | Effort: L | Status: OPEN

Problem: libtorrent defaults and compile-time options influence many clients,
but package and source builds can differ.

Approach: Build a minimal stock harness, compare official/source and package
routes, record build features, and observe through the Rust lab plus alerts or
a packet oracle.

Prove: same-version route records and two-connector captures exist for Linux
and Windows without claiming that an embedding client shares the profile.

### ⭐ What libtorrent publishes, and which version a pair can reach. 2026-10-22

Read through `AGENTS.md` rule 8's route, PyPI's JSON API and
archives.boost.io, the last two answering directly.

| source | what it carries |
| --- | --- |
| GitHub `arvidn/libtorrent` releases | two stable series at once. RC_2_0: `v2.0.15`, 2026-09-25T08:34:25Z, commit `1eb18fae...`, one asset, `libtorrent-rasterbar-2.0.15.tar.gz`. RC_2_1: `v2.1.2`, 2026-09-25T08:38:14Z, the tarball and 45 wheels. RC_1_2: `v1.2.20`, 2025-01-28 |
| PyPI `libtorrent` | `info.version` 2.1.1. It carries 2.0.15, 2026-09-25, 45 wheels, cp39 to cp313, manylinux, musllinux, macOS and win32/win_amd64. ⛔ It does not carry 2.1.2, and its newest 1.2 is 1.2.19 |
| the tag's `cibuildwheel.yml` and `pyproject.toml` | the PyPI wheels: cibuildwheel 2.23.4 runs the tag's `setup.py` with no option, against Boost 1.83.0 from archives.boost.io (`setup_boost.sh`) and a static OpenSSL (`setup_openssl.sh` on Linux, vcpkg's `x64-windows-static` on Windows) |
| the tag's `bindings/python/setup.py` | given no b2 option, it adds `deprecated-functions=on boost-link=static libtorrent-link=static crypto=openssl variant=release`; its `setup.cfg` adds `cxxstd=14`. *"The feature set will match the version found on pypi"* |
| archives.boost.io | `boost_1_83_0.tar.gz`, sha256 `c0685b68...`, the archive's own `.json` statement |

⛔ **The target is the 2.0 series, at 2.0.15, because that is where PyPI and
the tags agree.** A 2.1 pair on PyPI would measure 2.1.1, which `v2.1.2`
superseded the same day 2.0.15 was published. ⚠ 2.1 deserves its own pair:
the `v2.1.2` release carries the vendor's wheels as assets, so `release`
beside `source` reaches it today, or PyPI once it carries 2.1.2. 1.2 deserves
one later as well, since qBittorrent Enhanced and Deluge's Windows installer
embed it, and no index carries 1.2.20. Absolute 5 reads per series here: each
is a live stable line the vendor still releases.

⭐ **The harness is this tree's: [`../tools/harness/libtorrent/harness.py`](../tools/harness/libtorrent/harness.py).**
It is Python because the PyPI wheel delivers the library's Python binding and
nothing else that runs, so the one harness both routes can run is a binding
program. It is about 270 lines, carries `bit-ids:python-exception=ENGINE-01`,
and runs `lt.session(settings)` with the library's default settings on one
torrent. A tiny C++ program was weighed and set aside: it would link a library
neither route installs, so a third build would be measured.

| surface | library default, read from `lt.default_settings()` | what `start` passes |
| --- | --- | --- |
| DHT, bootstrapping from `dht.libtorrent.org:25401` | on | `--no-dht`, `enable_dht=false` |
| local service discovery | on | `--no-lsd` |
| UPnP, NAT-PMP | on, two settings | `--no-upnp`, `--no-natpmp` |
| PEX, `ut_pex` | one of the session's three default plugins | `--no-pex`: the session gets no default plugin and `ut_metadata` and `smart_ban` are added back |
| listen | `0.0.0.0:6881,[::]:6881` | `--ipv4-only --port 51413`: `0.0.0.0:51413` |
| uTP, encryption policy | on, `pe_enabled` both ways | nothing |
| peer fingerprint, user agent, handshake `v` | `-LT20F0-`, `libtorrent/2.0.15.0`, unset | nothing |
| alert mask | errors only | error, status, tracker, peer, connect and port-mapping alerts, written to `alerts-N.log`. No peer or tracker sees it |

- ⛔ **IPv4-only on every host, and recorded.** This host boots with IPv6
  disabled, and how the default `[::]` listen behaves on such a kernel was not
  measured. Off everywhere keeps the profile independent of the host's kernel.
  No handshake field moves.
- ⭐ **The harness reports what it configured**, `ready.txt`, read back from
  `ses.get_settings()`, and `start` holds it with `libtorrentReadyRefusal`:
  every surface off, `extensions=smart_ban,ut_metadata`, no proxy, no
  anonymous mode, the listen interface IPv4 on the peer port and every
  listener on it, and the library's identity defaults standing.
- ⚠ libtorrent expands `0.0.0.0` into one listener per interface, so the
  listen alerts name `127.0.0.1` inside the namespace. The first capture
  refused on that, and the check now takes any IPv4 address on the port.
- ⭐ **`stop` is the library's own shutdown**: a stop file the harness polls
  every 200 ms, then the session's destructor. It kills after 10 seconds.
  One mechanism for a Windows half, which has no SIGTERM to send.
- ⭐ **`binary=` is the binding's extension module**, the library as each route
  built it. The interpreter is the host's on both routes and would digest
  alike.

### ⭐ Two routes at 2.0.15 build one feature set, and both capture on this host. 2026-10-22

[`../tools/check/adapter_libtorrent.go`](../tools/check/adapter_libtorrent.go)
registers `libtorrent` on Linux. Its pure decisions are
[`../tools/check/libtorrent.go`](../tools/check/libtorrent.go), and `bit-check
libtorrent --selftest` asks them on any host: 125 cases, all held.

⭐ **The routes, every origin https, both rootless:**

| route | resolver | delivery | `package` |
| --- | --- | --- | --- |
| `package` | PyPI's index, asked for `libtorrent>=2.0,<2.1`, wheels only. pip's report must name 2.0.15 | the cp3XX manylinux wheel from files.pythonhosted.org into a venv at `<prefix>/opt/libtorrent` | `python-wheel` |
| `source` | `resolve-source` over the repository's refs, prefix `v`, three components, `release_series=2.0` | `git clone --depth 1` of `v2.0.15`, commit held to `1eb18fae...`, `deps/try_signal` alone; Boost 1.83.0 from archives.boost.io held to its digest; `pip wheel -v` of the tag with no option; the wheel installed into the same venv path | `python-source` |

- ⛔ **The scheme pins the 2.0 series.** Over the 136 refs on 2026-10-22,
  `resolve-source` selects `v2.0.15` as `2.0.15`, with 15 `superseded`, 14
  `other_series` (every 1.2 and 2.1 release), 3 prereleases and 103
  `foreign_tag`. ⚠ Until the runs below the series was a prefix, `v2.0.` with
  one component, which the published view could not order.
- ⛔ **A route never upgrades the pin.** Both refuse a 2.0.16 with *re-pin it,
  and its defaults digest*.
- ⛔ **The feature set is held at install, three ways.** The b2 line the tag's
  setup.py ran is read out of the log and must be exactly the vendor's
  features. The binding's own symbols must show the deprecated API, the DHT,
  logging, I2P and encryption compiled in, and OpenSSL linked in. And the
  sha256 of all 258 default settings must be `2ec98866...`.
- ⚠ **The source build's environment drops every `BOOST*`, compiler, flag,
  `PIP_*` and `PYTHON*` variable**, and pip runs with `PIP_CONFIG_FILE` set
  to `/dev/null`.
- ⚠ **What still differs, recorded rather than closed.** The wheel links a
  static OpenSSL 3.5.0 and was built in manylinux2014 by its devtoolset's
  compiler. The source route links the host's static `libcrypto.a` from
  `libssl-dev`, 3.0.13 here, with the host's g++ 13.3. Neither touches a field
  the wire carries. The source route takes setuptools from PyPI for its build.

⭐ **Driven on the session host as `ubuntu`, each lane with its own `HOME`,
prefix, claim and `RUNNER_TEMP`, through `bit-check install-step`, on the
host's Python 3.13.16:**

| step | package | source |
| --- | --- | --- |
| `install-step` | exit 0 in 6 seconds, `acquired=yes` | exit 0 in 309 and 321 seconds on two drives, the build itself 254, `acquired=yes`, commit `1eb18fae...` |
| the module | 19727328 bytes, sha256 `713eab7f...`, the wheel's | 19317816 bytes, `3fe19e8d...` on both drives in one workdir. A probe build of the same tag in another directory digested `731d183b...` |
| the answer | `library_version=2.0.15.0`, every define symbol present, `openssl=static` with `OpenSSL 3.5.0 8 Apr 2025` | the same, `openssl=static` with no version text |
| default settings | 258, `2ec98866...` | **the same, line for line** |
| b2 | | `cxxstd=14 deprecated-functions=on boost-link=static libtorrent-link=static crypto=openssl variant=release address-model=64` |
| `version` | `2.0.15` | `2.0.15` |
| capture, 45 s, 3 sessions, in a loopback-only namespace as uid 1000 | exit 0: 6 announces, 6 peer streams, 30 segments, verified by `sha256sum` | the same |
| tracker `peer_id` | `-LT20F0-` and a new twelve-byte tail per session | the same |
| tracker `user_agent` | `libtorrent/2.0.15.0` | the same |
| peer-wire `reserved` | `0000000000100005` | the same |
| `stop` | `closed on the stop file` in every session, in about 100 ms | the same |
| sockets | TCP `127.0.0.1:51413` listening and UDP `127.0.0.1:51413`, uTP. Nothing else | the same |
| `assemble-dispatch`, by hand | both records `publishable`, `classify_across` `build_equivalent`, 4 fields agreeing. Refused only for the run manifests a dispatch writes and the runner environment this host's claim has not got | |

- The announce is `uploaded=0`, `downloaded=0`, `left`, `corrupt=0`, an
  eight-digit hex `key`, `event=started`, `numwant=200`, `compact=1`,
  `no_peer_id=1`, `supportcrypto=1` and `redundant=0`. Its headers are `Host`,
  `User-Agent`, `Accept-Encoding: gzip` and `Connection: close`.
- ⭐ **A `stopped` announce in every session**, which the destructor sends.
- ⚠ **The peer-wire peer ID is not the tracker's.** Each connection carries
  `-LT20F0-` with its own tail, while each session's announces share one. That
  is the library's behaviour and it is recorded.
- ⚠ **The reserved bytes advertise the DHT though it is off**, as
  anacrolix/torrent's do: DHT, LTEP and Fast.
- The extension handshake offers `lt_donthave`, `share_mode`, `upload_only`,
  `ut_holepunch` and `ut_metadata`, and no `ut_pex`. It carries
  `complete_ago=-1`, `metadata_size`, `reqq=2000`, `yourip` and `v`
  `libtorrent/2.0.15.0`, the user agent, since `handshake_client_version` is
  unset.
- ⚠ `ss` names the process `python`, the venv's interpreter.

⚠ **Gap, as rqbit's and anacrolix's:** the record has no field for the library
version or the harness commit. No field is invented. The record's version is
`2.0.15`, `lt.__version__` without its fourth component. `build-features.txt`
keeps the harness's digest, the tree's commit, the whole answer, the b2 line,
Boost's digest, `libssl-dev`'s version and the wheel's digest.
`default-settings.txt` keeps all 258 defaults, and `source-commit` the tag's
commit. ⚠ The PyPI lane files as `linux_package_manager`, as Deluge's does.

⭐ `check-bitcheck` runs the selftest, removes `--no-dht` from a scratch copy
of the command line and sees the selftest refuse, changes `crypto=openssl`
in the b2 features held and sees the vendor's own line refused, and unpins
the release series and sees the describe refused.

⛔ **Windows has one honest route measured and none built.**

- ⭐ The `package` route reaches it. PyPI's
  `libtorrent-2.0.15-cp312-cp312-win_amd64.whl`, 3999252 bytes, holds a
  10726912-byte `.pyd` whose imports are system DLLs, the MSVC runtime and
  `python312.dll`, and no OpenSSL DLL. It links OpenSSL 3.6.1 statically. So
  CLIENT-07's 1.2.19 lesson, the vendor's OpenSSL DLLs, does not apply here.
- ⛔ The `source` route is unbuilt and unmeasured. It needs MSVC, Boost 1.83.0
  bootstrapped with b2, and a static OpenSSL, which the vendor makes with
  vcpkg's `x64-windows-static`, all inside 1500 seconds. Linux's build alone
  took 254 seconds on four cores. Nothing here can measure a Windows build.
- ⛔ So no Windows half is registered, and a Windows dispatch of `libtorrent`
  could not run. The next unit writes the half: the same harness and pure
  decisions, the PyPI route, and a source route whose OpenSSL is either
  vcpkg's static build or the runner's preinstalled OpenSSL. The second
  would change the linkage that the answer holds `static`.

⛔ **What only a runner can prove.** Dispatch `capture-client` with `adapters:
["libtorrent"]`, `routes: ["package","source"]` and `windows_adapters: []`.
The workflow needs no change: the source lane reads `compiles=source` and is
bounded at 1500 seconds. Check, in reading order:

1. whether the package lane's pip finds a cp312 manylinux wheel for
   `ubuntu-24.04`'s Python 3.12, and whether its answer's defaults digest is
   still `2ec98866...` there;
2. whether the source lane's `git clone`, archives.boost.io and the build
   finish inside 1500 seconds. `build-features.txt` carries `build_seconds`,
   the b2 line and `libssl_dev`;
3. the two captures, then `assemble-bundle`: the pair `publishable` and
   `build_equivalent`, with `target.kind` `library`.

### ⛔ Runs 39783974176 and 39784872974: the view could not order the scheme's series. 2026-10-22

⭐ **The runner captured both Linux lanes.** `capture-client` run 39783974176,
at `df7f9c67`, dispatched on 2026-10-22 at 13:06:33Z, finished `success` in
six minutes: *Linux client capture (libtorrent, package)* and *(libtorrent,
source)* both green, the Windows matrix skipped. `assemble-bundle` run
39784872974 then built both records, `publishable` each and
`classify_across: build_equivalent`, and `corpus: valid, 2 record(s)`.

⛔ **And the published view refused the bundle**, both records:

```text
E-VIW-02 at profiles/v1/libtorrent/2.0.15/linux/x86-64/python-source/cap-source.json: version 2.0.15 is not orderable under target libtorrent's scheme
E-VIW-02 at profiles/v1/libtorrent/2.0.15/linux/x86-64/python-wheel/cap-package.json: version 2.0.15 is not orderable under target libtorrent's scheme
```

⛔ **The cause was the scheme, which pinned the series as a prefix.**
`describe` declared `release_tag_prefix=v2.0.` and one component, so the
resolver read `v2.0.15` as the version `15`. The source lane's own artifact
says so: `resolution.txt` carries `selected_version=15` beside
`selected_tag=v2.0.15`, and `resolution.json` the scheme `v2.0.`, 1, 1. The
assembler takes each target's scheme from there, by design, and the view
orders a record's `build.version` under it. `2.0.15` is three components.
⚠ The tag was right and only the tag is read downstream, which is why every
lane went green and nothing failed until the view.

⭐ **The scheme is now the library's own, with the series pinned beside it.**
`describe` declares `release_tag_prefix=v`, three components, floor `-` and
the new `release_series=2.0`.

| piece | rule |
| --- | --- |
| `release_series` | declared on every adapter, `-` for none, never defaulted, as the floor is. Dotted plain integers, fewer than `release_min_components`; `2.`, `.0`, `02.0`, `2.x` and a series as long as a version cannot run |
| `other_series` | the new verdict for a stable tag whose leading components differ readably from the series: libtorrent's own release on another line. ⚠ Not `foreign_tag`, which says the tag is not this target's, and not `superseded`, since `v2.1.2` is newer. A prerelease of another series stays a prerelease |
| what still blocks | a tag that agrees with the series as far as it goes, `v2.0` or `v2`, or is unreadable before a difference, `v2.0x.1`: it could be the series's newest |
| `VersionScheme::components` | orders only versions inside the series. ⛔ So the latest view never compares across series: a 2.1 record under a target pinned to 2.0 is `E-VIW-02`, not the latest. A build line is target, platform, arch and package, and both series stay current, so "newest" across them would answer for a line nobody captured |
| the scheme line | `libtorrent:v:3:3:2.0`, the fifth field only where a series is pinned, so every line in the published `schemes.txt` reads back byte for byte. ⛔ Moving the pin is a scheme change, and the assembler refuses two schemes for one target, so a move to 2.1 is a decision, not a drift |
| the document | `bit-ids/resolution/3`, with `scheme.series` stated, `null` for none. `E-RES-11` re-derives every `other_series`, `E-RES-12` refuses an ordered candidate outside the series, `E-RES-13` a series that leaves nothing to order. The assembler still reads `/1` and `/2` |

⭐ **Measured over every adapter's live inputs on 2026-10-22**, the same bytes
read before and after the change: each adapter's refs through `resolve-source
--refs`, and each listing through `resolve-stable` under its `describe`.
Eighteen of the twenty verdict tables are identical, and nothing else
selects differently. libtorrent's two move:

| input | before | after |
| --- | --- | --- |
| 136 refs | `selected=15`; 15 `superseded`, 120 `foreign_tag` | `selected=2.0.15`; 15 `superseded`, 14 `other_series` (`v1.2.10` to `v1.2.20`, `v2.1.0` to `v2.1.2`), 3 `prerelease_by_version` (2.1's release candidates), 103 `foreign_tag` |
| 69 releases | `selected=15`; 14 `superseded`, 54 `foreign_tag` | `selected=2.0.15`; 14 `superseded`, 13 `other_series`, 3 `prerelease_by_label`, 38 `foreign_tag` |

⭐ **Run 39783974176's own artifacts, re-assembled here onto `origin/data`:**

| artifacts | `assemble-dispatch --published` |
| --- | --- |
| as published | exit 1, the same two `E-VIW-02`. ⛔ The scheme is the artifacts', so a re-assembly is not a fix |
| a scratch copy whose two `resolution.json` carry the new scheme | exit 0: `38 of 38 record(s) may enter a published view`, `schemes.txt` gains `libtorrent:v:3:3:2.0`, and the latest view names `python-source` and `python-wheel` at `2.0.15`, 29 rows |

⚠ The second is a driven proof that the view orders these records, in
disposable state, and is not a publication: its scheme was written, not
resolved.

⭐ The 36 records on `origin/data`, at `ab08fa3`, still validate, `36 of 36
record(s) may enter a published view`, and `build-indexes` under the nine
published schemes rebuilds `indexes/v1/profiles.json` byte for byte, 27
latest rows, with or without libtorrent's line beside them.

⛔ **So libtorrent needs a new dispatch, not a re-assembly.** Dispatch
`capture-client` with `adapters: ["libtorrent"]`, `routes:
["package","source"]` and `windows_adapters: []` at this commit, then
`assemble-bundle` over that run. The record's version was `2.0.15` already;
what changes is the resolution each lane writes. Check that the source lane's
`resolution.txt` names `selected_version=2.0.15` and `resolution.json`
`bit-ids/resolution/3` with `series` `2.0`.

⭐ Tests: `cargo test` 629 across the workspace, 308 in `bit-ids`, five new
unit tests and three new integration tests among them, and the planted
defects for `E-RES-11` to `E-RES-13`. `check-source-route` replays the 136
refs: `v2.0.15` selected as `2.0.15`, the 14 set aside, `v2.1.2` without the
pin, and the old prefix pin's version `15`; 67 cases. `check-bundle` carries
the series as a fifth field, refuses a record outside it and a
third-generation scheme that omits it, and holds a pinned and an unpinned
scheme apart; 26 cases. `bit-check libtorrent --selftest` holds 125 cases, and
`check-bitcheck` plants an unpinned series in a scratch copy of the describe
and sees the selftest refuse it; 350 cases. ⚠ Mutation-proved: the series
guard in `components` removed fails two unit tests, and the assembler's
"series omitted" guard removed fails its `check-bundle` case.

## ENGINE-02: anacrolix/torrent engine matrix

Source: operator scope and anacrolix/torrent upstream
Priority: P1 | Effort: L | Status: DONE

Problem: A Go module version is not directly an installable client release and
needs a reproducible harness definition.

Approach: Pin a minimal harness source, compare module-proxy and source-checkout
routes at one tag, record Go toolchain facts, and actively capture behavior.

Prove: two independently resolved harness builds at the same module version
produce corroborated profiles and reproducible build manifests.

### ⭐ What anacrolix/torrent publishes, and the harness this tree pins. 2026-10-22

Read through `AGENTS.md` rule 8's route, proxy.golang.org, sum.golang.org and
go.dev, each answering directly.

| source | what it carries |
| --- | --- |
| GitHub `anacrolix/torrent` releases | **none**: the listing is empty. The tags are the releases |
| the tags | 153 refs. `resolve-source` selects `v1.61.0`, 114 superseded, 3 `prerelease_by_version`, 35 `foreign_tag` |
| proxy.golang.org `@latest` | `v1.61.0`, 2025-12-17T07:39:03Z, origin `refs/tags/v1.61.0` at `486bcbe7...`. The newest of its `@v/list` |
| sum.golang.org | `h1:vxo+B4Sw...` for the module and `h1:yKUKuZSS...` for its `go.mod` |
| the module's `go.mod` | `go 1.24.0`. Its default build links `go-libutp`, C++, when cgo is on, and anacrolix/utp, Go, when it is off |
| go.dev/dl | newest stable `go1.27.3`. The session host's own Go is 1.24.7 |

⭐ **The harness is this tree's, because a module version installs nothing.**
[`../tools/harness/anacrolix-torrent/`](../tools/harness/anacrolix-torrent/)
is a single `main.go` of about 200 lines and its own `go.mod` and `go.sum`.
It starts `torrent.NewDefaultClientConfig()`, adds the observer's torrent,
whose tracker hands out the observer's peer, and writes back what it
configured. Every flag defaults to the library's own value. The adapter passes
the flags that switch surfaces off.

- ⚠ **Not upstream's `cmd/torrent`**, which the module also carries. It is a
  downloader: it ends when its torrent completes unless told to seed, stops on
  a signal Windows cannot send a console-less process, and reports none of its
  config. `docs/client-matrix.md` already names a minimal harness.
- ⛔ **Its own module**, because the library has 70 dependencies in the build
  and `bit-check` has none. `go.mod` requires `github.com/anacrolix/torrent
  v1.61.0` and names `toolchain go1.27.3`. `go.sum` is the 140 lines the proxy
  route writes from sum.golang.org's answers, not `go mod tidy`'s 507.
- ⛔ **The pin is held in two places with one function.** `anacrolixPinRefusal`
  holds `go.mod` and `go.sum` to the adapter's constants: the version, both
  hashes and the toolchain, no `replace`, no `exclude`. Every `install` asks it,
  and `check-bitcheck` asks it of the tree. When stable moves, the harness is
  re-pinned in one change. A route refuses a newer `@latest` or tag rather than
  upgrading.
- ⚠ **`-buildvcs=false` is load-bearing.** The library's `version` package
  writes the main module's path and version into the extended handshake's `v`.
  Go 1.24 and later stamp that version from the enclosing git tree, so a harness
  built inside this repository would change its wire identity with every
  commit. With the flag the main module is `(devel)`.

| surface | library default | what `start` passes |
| --- | --- | --- |
| DHT, bootstrapping from global nodes | on | `-dht=false` |
| PEX, `ut_pex` | on | `-pex=false` |
| UPnP and NAT-PMP port forwarding | on | `-port-forward=false` |
| webseeds | on | `-webseeds=false` |
| WebRTC peers through websocket trackers | on | `-webtorrent=false` |
| IPv6 | on | `-ipv6=false` |
| local peer discovery | none in the library | nothing |
| uTP | on | nothing |
| peer port | 42069 | the capture's, `-port 51413` |
| peer ID, user agent, `v`, BEP 20 prefix | the library's | nothing |

- ⛔ **IPv6 is off on every host, and that is measured.** The library listens
  on `tcp6` by default. On a kernel booted with IPv6 disabled that fails with
  `socket: address family not supported`, which the library does not treat as
  an unsupported network, so the client never starts. Off everywhere keeps the
  profile independent of the host. It changes no handshake field.
- ⭐ **The harness reports what it configured** in `ready.txt`, and `start`
  holds it with `anacrolixReadyRefusal`. Every surface must be off, no proxy and
  no ICE server, every listener IPv4 on the peer port. The library's three
  identity defaults must stand as self-reports, unset: prefix `-GT0003-`, user
  agent `anacrolix-torrent/v1.61.0`, and `v` `bit-ids/harness/anacrolix-torrent
  (devel) (anacrolix/torrent v1.61.0)`.
- The harness's environment drops every `GO*` (the runtime's knobs, the
  library's `GO_LOG`, anacrolix/utp's `GO_UTP_*`), every `TORRENT_*` and
  `WEBTORRENT_*`, and every proxy variable.
- ⭐ **`stop` is the library's own `Client.Close`**, on a stop file the harness
  polls every 200 ms, because Windows has no SIGTERM to send. One mechanism on
  both platforms keeps the two halves one harness. It kills after 10 seconds.

### ⭐ Two routes at v1.61.0 build one executable, byte for byte. 2026-10-22

⭐ **The routes, one module version and one harness source:**

| route | resolver | what the go command fetches | checked against |
| --- | --- | --- | --- |
| `package` | proxy.golang.org's `@latest`, which must name the pin | every module from `GOPROXY=https://proxy.golang.org`, with no `go.sum` given, `-mod=mod` | sum.golang.org, by the go command. The `go.sum` it writes must be the committed one byte for byte |
| `source` | `resolve-source`: `git ls-remote --tags` of the repository, which must select the pin | every module from its own origin, `GOPROXY=direct`, `-mod=readonly`. The library is cloned by git from `https://github.com/anacrolix/torrent` at `refs/tags/v1.61.0` | the committed `go.sum`. Each module's zip is built from its tree and hashed |

- ⛔ **Nothing on the source route comes from the proxy**, and the record
  shows it. The go command's download record carries the same origin on both
  routes, so it cannot tell them apart. `vcs_repositories` counts what git
  cloned instead: 66 on the source route and 0 on the package route, and each
  route refuses the other's count.
- ⛔ **One build environment, every host value dropped.** `GOENV=off`,
  `GOWORK=off`, a module cache and a build cache of the workdir's own,
  `GOTOOLCHAIN=go1.27.3`, `CGO_ENABLED=0`, `GOSUMDB=sum.golang.org`. The build
  is `go build -trimpath -buildvcs=false`.
- ⚠ **The toolchain is the one input both routes share.** `GOTOOLCHAIN` names
  the pin. The package route takes it from proxy.golang.org and the source
  route from go.dev/dl/mod, `golang.org/toolchain`'s own origin. Both are
  checked against sum.golang.org.
- ⭐ **CGO is off, and that is a decision.** With cgo the library links libutp
  through C++. Without it, anacrolix/utp, which is Go. A Linux runner has a C
  compiler and a Windows one may not, so cgo's default would put two uTP
  implementations under one profile and tie the executable to the host's
  compiler.
- ⚠ The build carries `DefaultGODEBUG`, Go 1.27.3's compatibility defaults for
  a `go 1.24.0` module, `tlssha1=1` among them. It is recorded as it is.

⭐ **The build manifest is the executable's own answer.** The harness's
`--version` prints its library line and then the whole build information, as
`go version -m` prints it. `anacrolixManifestRefusal` holds it to the pin: the
library at `v1.61.0` under sum.golang.org's hash, `go1.27.3`,
`-trimpath=true`, `CGO_ENABLED=0`, `-buildmode=exe`, the main module at
`(devel)`, every dependency with an `h1:` sum. It refuses a `vcs.*` stamp, a
`replace` and any setting it does not name. `version` reports `1.61.0`, the
library's version, which is what the record's version is.

⭐ **Driven on the session host as `ubuntu`, each lane with its own `HOME`,
prefix, claim and `RUNNER_TEMP`, through `bit-check install-step`:**

| step | package | source |
| --- | --- | --- |
| resolve | the proxy's `@latest`: `v1.61.0` | `resolve-source`: 153 refs, `v1.61.0` selected |
| `install-step` | exit 0 in 59 seconds, go itself 45, `acquired=yes`, `package=elf-binary-static` | exit 0 in 134 seconds, go itself 125, `acquired=yes`, commit `486bcbe7...` |
| what was fetched | every module from the proxy, 0 repositories cloned | 66 repositories cloned, no module from the proxy |
| `go.sum` written | the committed one, byte for byte | unchanged, `-mod=readonly` |
| the executable | 29464958 bytes, sha256 `59e39159...`, static | **the same bytes** |
| the build manifest | 70 dependencies, `go1.27.3`, `-trimpath`, `CGO_ENABLED=0`, `GOAMD64=v1` | **byte-identical** |
| the answer | `1.61.0` | `1.61.0` |

⚠ The routes were driven three times each. An earlier harness revision built
to `3d684a21...` on both routes and the final one to `59e39159...` on both,
twice each. The harness's digest, over its three files, is `9457a73b...`.
Each route then removes its caches. ⚠ `-modcacherw` does not reach the
toolchain, which the go command extracts read-only: 253 megabytes were left
behind until the cleanup made each directory writable first.

⭐ **The capture ran on this host**, because the harness is IPv4-only. Each
lane ran 45 seconds and 3 sessions in a loopback-only namespace as uid 1000,
which root made because `ubuntu` has no `sudo` here:

| | package | source |
| --- | --- | --- |
| `capture-client` | exit 0: 3 announces, 6 peer streams, verified by `sha256sum` | the same |
| tracker `peer_id` | `-GT0003-` and a new twelve-byte tail per session | the same |
| tracker `user_agent` | `anacrolix-torrent/v1.61.0` | the same |
| peer-wire `reserved` | `0000000000100005` | the same |
| `stop` | `closed on the stop file` in every session, stopped after 40 s with 5 s margin | the same |
| sockets | TCP `0.0.0.0:51413` listening and UDP `0.0.0.0:51413`, uTP. Nothing else | |
| `assemble-dispatch`, by hand | both records `publishable`, `classify_across` `build_equivalent`, 4 fields agreeing. Refused only for the run manifests a dispatch writes and the runner environment this host's claim has not got | |

- The announce is `compact=1`, `downloaded=0`, `event=started`, a signed
  decimal `key`, `left`, `port`, `supportcrypto=1` and `uploaded=0`. Its
  headers are `User-Agent`, `Accept-Encoding: gzip` and `Connection: close`.
- ⚠ **The reserved bytes advertise the DHT though it is off.** The library's
  default extension bits are DHT, LTEP and Fast, whatever `NoDHT` says. That is
  the library's handshake and it is recorded.
- The extension handshake offers `ut_holepunch` and `ut_metadata`, `ut_pex`
  being off. It carries `e=1`, `reqq=1024`, `p=51413` and `yourip`. Its `v` is
  the self-report above, measured on the wire.
- ⚠ **No `stopped` announce.** `Client.Close` returned in 0 ms in every
  session and the transcript holds one `started` per session.

⛔ **A handshake split across segments refused both readers, and both were
wrong.**
The library writes its handshake in more than one write, and the lab records
what each read returned. Over the final two captures' twelve connections it
arrived whole on four, as 20 and 48 bytes on three, as 28 and 40 on two, and
as 20, 8 and the rest on three. The second connector refused
with *a segment begins with the protocol string and is 20 bytes, not 68*.
`assemble-capture` refused with *the peer handshake: truncated*. Both now read
the connection's byte stream: a first segment that opens as a handshake and
is shorter than one takes the following segments of its own connection until
it is one. A connection that opens with anything else is not joined, and a
handshake its connection never completes is still refused. `normalize.rs`'s
`opening_streams` and the connector's `joined_handshakes` make the same test.
⭐ `check-connector` has three new cases, and a plant that reads segment by
segment fails two. `cross_platform_events` has two new tests: the committed
qBittorrent fixture split at byte 20 and at byte 28 normalizes to the same
canonical text, and a cut handshake is refused. Removing the join fails the
first. `check-assemble`'s unchoke-first case, which a join of any first
segment let through, is what made the opening test a prefix test.

⭐ **The record path is proved.** `check-assemble` has three new rows. A
`package` lane whose index is `proxy.golang.org` files as the record's own
`go_module` route with a `module_version` identity, never a host's package
manager. 77 of 77. `check-bitcheck --harnesses` disables the branch in
`assemble-capture`, and exactly two rows fail. `bit-check anacrolix-torrent
--selftest` asks every pure decision on any host: 180 cases, all held.
`check-bitcheck` runs it, removes `-dht=false` from a scratch copy and sees
the selftest refuse, and holds the harness to the pin.

⚠ **Gap, as rqbit's:** `docs/client-matrix.md` says a library profile records
the library version and the harness commit, and the record has no field for
either. No field is invented. The record's version is the library's,
`1.61.0`, and the route identity names the module. `build-env.txt` keeps the
harness's digest, the tree's commit when git can read it, the toolchain and
the build environment. `build-manifest.txt` keeps the build information, and
`library-origin.txt` the tag's commit.

⛔ **What only a runner can prove.** Dispatch `capture-client` with `adapters:
["anacrolix-torrent"]`, `routes: ["package","source"]`, `windows_adapters:
["anacrolix-torrent"]` and `windows_routes: ["package","source"]`. The
workflow needs no change: both platforms read `compiles=package,source`. Check,
in reading order:

1. whether both Linux lanes install inside 1500 seconds, and whether a
   runner's executable digests as `59e39159...`. `build-env.txt` names the
   bootstrap Go, which must be 1.21 or later to switch toolchains;
2. whether the Windows lanes' git and Go reach every module's origin, and
   whether both Windows executables digest alike. Nothing has built them yet;
3. whether the stop file ends the harness on Windows. `client/stop.txt` says
   `closed on the stop file` or `killed`;
4. the four captures, then `assemble-bundle`: each platform's pair
   `publishable` and `build_equivalent`, with `target.kind` `library`.

### ⭐ Closed: two independent builds, one binary per platform. 2026-10-22

capture-client run 39774331484 built the pinned harness at anacrolix/torrent
v1.61.0 on both routes on both platforms. Each platform's two routes, the
module proxy and the tag fetched with `GOPROXY=direct`, installed the same
executable: Linux `59e39159…` on both (the same digest as the session host's
build) and Windows `cdc6a2d8…` on both. assemble-bundle run 39775716205 found
all four records `publishable`, and each pair `build_equivalent`, with
`target.kind` `library`. publish-data run 39775861918 appended them to `data`.

## ENGINE-03: rqbit engine matrix

Source: operator scope and rqbit upstream
Priority: P1 | Effort: L | Status: DONE

Problem: rqbit is available as application artifacts and Rust packages, which
may encode different features or revisions.

Approach: Compare upstream release and Cargo/source routes, record enabled
features, and observe the same synthetic torrent through independent connectors.

Prove: a two-route run proves build equivalence and publishes a harness-labeled
profile with raw evidence.

### ⭐ What rqbit publishes, and which routes exist. 2026-10-21

Read through `AGENTS.md` rule 8's route, crates.io's API and the package
indexes. The Linux asset was fetched by the URL the listing names and checked
against the digest the listing states.

| source | what it carries |
| --- | --- |
| GitHub `ikatson/rqbit` releases | `v9.0.1`, 2026-08-20T18:21:53Z, commit `a499d2f2...`, not a prerelease. 15 assets, **no checksums document and no signatures**; the listing states a sha256 for every asset |
| `rqbit-linux-amd64` | 28696960 bytes, sha256 `82ed2c23...`, static-pie (musl). It answers `rqbit 9.0.1` |
| `rqbit.exe` | 12706816 bytes, sha256 `2ed68320...`. The rest are `rqbit-desktop`, the Tauri app |
| the tag's workflows | Linux: `docker/Dockerfile.xx`, `xx-cargo build --profile release-github`, `OPENSSL_STATIC=1`. Windows: `cargo build --profile release-github`. ⛔ **Neither passes a feature flag** |
| the tag's workspace `Cargo.toml` | `release-github` is `release` with `panic = "abort"`, `debug = false`, `lto = false` |
| crates.io `rqbit` | newest stable 9.0.1, 2026-08-20T18:23:13Z, checksum `234059b3...`. Default features `default-tls`, `postgres`, `webui`, `prometheus`. The crate's manifest has **no `[profile]`** |
| the crate's `Cargo.lock` | 481 packages, every one in the tag's 740-package workspace lock at the same version |
| the tags | 65, all `v<major>.<minor>.<patch>`, some with `-beta.N`. ⚠ `v9.0.0-beta.2` is a release object **not** marked prerelease; its suffix is what keeps it out |

| index | carries rqbit? |
| --- | --- |
| winget | no: `manifests/i` and `manifests/r` hold no publisher or package naming it |
| Scoop `main`, `extras` | no |
| Chocolatey | no: `FindPackagesById('rqbit')` is empty |
| Ubuntu 24.04 apt | no |
| Repology | unmeasurable from this host: direct reset the connection, the read proxy answered 522 |

⭐ **The routes, every origin https:**

| platform | route | what it does | `package` |
| --- | --- | --- | --- |
| Linux | `release` | `rqbit-linux-amd64`, placed by `install-rootless --install` at `<prefix>/bin/rqbit` under the listing's digest | `elf-binary-static` |
| Linux | `package` | crates.io's `rqbit`: `cargo install --locked rqbit --root <prefix>`, no version and no feature flag | `elf-binary` |
| Windows | `release` | `rqbit.exe`, placed the same way at `<prefix>\bin\rqbit.exe` | `pe-binary` |
| Windows | `package` | the same `cargo install`, with MSVC | `pe-binary` |

Why these:

- **crates.io is the second index, on both platforms.** No package manager
  carries rqbit. crates.io resolves the version and cargo delivers it, so
  neither is the release listing's. ⚠ The origin is shared: the vendor uploads
  both. A build of the tag would share the release's repository as well, so
  `source` is weaker than this.
- ⛔ **The feature set is the release's.** The release's features were
  established by reading both workflows at the tag: no `--features`, no
  `--no-default-features`, so the defaults. `cargo install` with no feature
  flag builds the same defaults, and the route refuses a `.crates2.json` that
  recorded any other selection.
- ⛔ **So is the profile.** A published crate carries no workspace profile, so
  `cargo install` alone would build cargo's `release`, which unwinds on panic.
  The route sets `CARGO_PROFILE_RELEASE_PANIC=abort`, the one setting in which
  `release-github` differs. `build-features.txt` records it.
- ⚠ **What still differs, recorded rather than closed.** Linux's release is
  musl and links OpenSSL statically. The cargo build links the host's glibc and
  `libssl.so.3`. Neither the tag nor the crate carries a `package-lock.json`
  for the web UI, so `npm install` resolves it at build time on both routes.
  cargo warns that `chacha20 0.10.1` in the lock is yanked.

⛔ **The target is a harness, and the record says so with its own vocabulary.**
`Target.kind` has two values, `client` and `library`, "a committed reference
harness built against a library release". The harness here is upstream's
`rqbit` command, which librqbit's manifest calls "just a small wrapper on top
of it". No harness of this project's own is committed. The adapter's
`describe` prints `target_kind=library`. `capture-client` copies it into the
attestation's `kind`, and `assemble-capture` files `TargetKind::Library`. ⚠
**Gap:** `docs/client-matrix.md` says a library profile records the library
version and the harness commit, and the record has no field for either. No
field is invented here. `features.txt` keeps the library version the running
build reports, `GET /`'s `version`, and `build-features.txt` keeps cargo's
record of the crate. Nothing a run writes names the tag's commit. It is
`a499d2f2...`, read here from `git/ref/tags/v9.0.1`.

⛔ **The slug is the catalogue's `rqbit`.** `E-COR-20` refuses a connector whose
identifier contains it. The connectors are `bit-ids-probe` and
`cpython-stdlib`.

### ⭐ The adapter is built, both Linux routes install, and this host cannot run the build. 2026-10-21

[`../tools/check/adapter_rqbit.go`](../tools/check/adapter_rqbit.go) and
[`../tools/check/adapter_rqbit_windows.go`](../tools/check/adapter_rqbit_windows.go)
register `rqbit`. Their pure decisions are one untagged file,
[`../tools/check/rqbit.go`](../tools/check/rqbit.go), and `bit-check rqbit
--selftest` asks them on any host: 115 cases, all held. `check-bitcheck` runs
it, removes `--disable-lsd` from a scratch copy and sees the selftest refuse,
and holds the route's Rust to `rust-toolchain.toml`'s `1.98.0`.

⭐ **Driving, read from `--help` and the 9.0.1 source:**

| surface | default | what `start` passes |
| --- | --- | --- |
| DHT, and its routing-table file | on | `--disable-dht`, `--disable-dht-persistence` |
| UPnP port forwarding | on | `--disable-upnp-port-forward` |
| local peer discovery | on | `--disable-lsd` |
| session persistence | on | `server start --disable-persistence` |
| UPnP media server, mDNS | off | nothing, and `start` refuses either flag |
| HTTP API | `127.0.0.1:3030` | `--http-api-listen-addr 127.0.0.1:3030` |
| PEX | no switch | nothing. It reaches connected peers only, the observer's |
| update check, telemetry | none in the command | nothing |

- **Every `RQBIT_*` is dropped from the build's environment**, with `RUST_LOG`
  and `CORS_ALLOW_REGEXP`. Each switch also reads an environment variable.
- The torrent goes in as bytes, `POST /torrents?overwrite=true&is_url=false`.
  The tracker in it hands out the observer's peer. The peer ID and the user
  agent are the build's own.
- ⭐ **The features are probed in the running build.** `/web/` is routed only
  with `webui` and `/metrics` only with `prometheus`. `start` writes both
  statuses and the library version to `features.txt`, and refuses a build that
  does not answer both with 200.
- `stop` on Linux sends SIGTERM, which the build's handler turns into a
  shutdown within five seconds, and KILL after eight. ⚠ The Windows build
  compiles that handler out and its API has no shutdown, so Windows kills.
  The build constructs no `stopped` announce on either platform.
- Each session gets a fresh `HOME` under the workdir. `stop` lists what the
  build left there into `state.txt` and removes it. On Windows it looks in
  `%APPDATA%\rqbit` and `%LOCALAPPDATA%\rqbit`, where the build's
  `directories` crate would write, and `start` refuses a host that has either.

⭐ **Driven on the session host as `ubuntu`, each lane with its own `HOME`,
prefix, claim and `RUNNER_TEMP`, through `bit-check install-step`:**

| step | release | package |
| --- | --- | --- |
| resolve | `resolve-release`: `rqbit-linux-amd64` of 15 assets, `digest_source=source-listing` | none: crates.io's index resolves, as apt's would |
| `install-step` | exit 0 in 2 seconds, `acquired=yes`, `package=elf-binary-static` | exit 0 in 230 seconds, cargo itself 222, under the 1500-second bound, `acquired=yes`, `package=elf-binary` |
| the build's answer | `rqbit 9.0.1` | `rqbit 9.0.1` |
| the executable | 28696960 bytes, sha256 `82ed2c23...`, the listing's asset | 21936776 bytes, sha256 `7828bce1...`, linked against `libssl.so.3` and glibc. ⚠ A first build without the panic override was 27457096 bytes, `6df4f0c5...`: the unwind tables are what `abort` drops |
| cargo's record | | `features: []`, `all_features` and `no_default_features` false, `profile` `release`, target `x86_64-unknown-linux-gnu`, `rustc 1.98.0`; the crate digests as `234059b3...`, crates.io's checksum |

⛔ **The capture cannot run on this host, and that is measured.** The session
host boots with `ipv6.disable=1`. Under `bit-check contained`, both builds exit
at once with `error creating socket: Address family not supported by protocol
(os error 97)`. The TCP listener binds `[::]` by default. `--ipv4-only` moves
it to `0.0.0.0`, and then the UDP tracker client fails the same way, because
librqbit always binds it at `[::]:0`. No switch avoids that. So `start`
passes no `--ipv4-only`: it would buy nothing here and change the listener on
a host with IPv6. `start` refused with *rqbit exited before its HTTP API
answered*, `stop` recorded *already exited*, and the build left no state. ⚠ So
the API calls, the feature probe, the sockets and the wire identity are
unmeasured until a runner runs them. A runner's kernel has IPv6, and a
namespace's loopback carries `::1`.

⭐ **The record path is proved without a capture.** `check-assemble` has four
new rows. A `package` lane whose index is `crates-io` files as a
`cargo_install` route with a `module_version` identity, never a host's package
manager, and an attestation with `kind=library` files a `library` target. 74
of 74. `check-bitcheck --harnesses` disables the branch in `assemble-capture`,
and exactly the two route rows fail.
`capture-client --selftest` holds seven new `target_kind` cases.

⛔ **What only a runner can prove.** Dispatch `capture-client` with `adapters:
["rqbit"]`, `routes: ["release","package"]`, `windows_adapters: ["rqbit"]`
and `windows_routes: ["release","package"]`. ⚠ The workflow changed: the
Windows lane now reads `compiles` and bounds a compiling route at 1500 seconds,
and its job bound is 55 minutes. Check, in reading order:

1. whether `cargo install` finishes inside 1500 seconds on both runners.
   `build-features.txt` carries the seconds, the profile override and cargo's
   own record;
2. whether `start` answers on both platforms: `features.txt` with
   `web_status=200`, `metrics_status=200` and `library_version=9.0.1`, and
   `sockets.txt` on Linux, which is the first read of the build's sockets;
3. the peer ID, the user agent and the reserved bytes, which nothing has
   observed yet;
4. then `assemble-bundle`: each platform's pair `publishable` and
   `build_equivalent`, with `target.kind` `library`.

### ⛔ Run 39777039760: the Windows cargo record names `rqbit.exe`. 2026-10-22

capture-client run 39777039760 was rqbit's first capture anywhere. Both Linux
lanes, `release` and `package`, installed and captured. On Windows, cargo
built rqbit 9.0.1 in 7 minutes 48 seconds, and then the adapter refused it:
*cargo installed the executables [rqbit.exe], not one rqbit*. Windows' cargo
records the executable with its suffix, and the check compared the bare name.
⭐ The record now accepts `rqbit` or `rqbit.exe` and nothing else. The selftest
reads a Windows-shaped record and refuses `rqbit2.exe` (117 of 117).

### ⭐ Closed: build equivalence on both platforms, published as a library harness. 2026-10-22

capture-client run 39780037879 on `f87cc67` captured rqbit 9.0.1 on both
platforms, from the vendor's release binary and from crates.io through
`cargo install`. assemble-bundle run 39781553203 found all four records
`publishable`, and each platform's pair `build_equivalent`, with `target.kind`
`library`. publish-data run 39781744625 appended them to `data` with their raw
evidence. The earlier run, 39777039760, is the one whose Windows cargo record
named `rqbit.exe`.
