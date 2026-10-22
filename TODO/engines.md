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
