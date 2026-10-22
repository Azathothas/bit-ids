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

## ENGINE-02: anacrolix/torrent engine matrix

Source: operator scope and anacrolix/torrent upstream
Priority: P1 | Effort: L | Status: OPEN

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

## ENGINE-03: rqbit engine matrix

Source: operator scope and rqbit upstream
Priority: P1 | Effort: L | Status: OPEN

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
