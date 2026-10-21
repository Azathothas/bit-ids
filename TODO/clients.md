# Client adapter entries

Every adapter resolves the latest stable release at run time, proves at least
two same-version acquisition routes on each supported host family, disables
public peer discovery, drives the isolated lab, and produces two-connector
evidence. Candidate routes remain hypotheses until the entry closes.

## The machinery every entry below shares

⭐ **An adapter is five subcommands and nothing else**, specified in
[`../scripts/capture/adapters/README.md`](../scripts/capture/adapters/README.md).
The parts that are not per-target live once:
[`install-client`](../tools/check/installclient.go) runs a route while
the network is up, `capture-client` runs
the measurement under containment, and
[`client-capture`](../crates/bit-ids-probe/examples/client-capture.rs) is the
observer that hands out a torrent naming its own tracker.

⛔ **`stock_client` is the adapter's own declaration, copied into the
attestation.** A stub drives every guard in that path and is not a product, so a
run it drove records `false`. ⚠ The harness proves the field varies by running
one stub twice under both declarations; a runner that hardcoded either would
pass one of those cases and fail the other.

⛔ **Two things make a run a measurement of a build, and both are checked.** The
adapter answers `version` from the installed executable, and an announce arrives
carrying a peer ID this observer did not generate whose bytes the raw transcript
holds. ⚠ Without the second, a bundle of empty artifacts verifying against their
own empty digests satisfies everything else.

## CLIENT-01: qBittorrent capture adapter

Source: operator scope, upstream repository, and August 2026 priority sample
Priority: P1 | Effort: L | Status: OPEN

Problem: The highest-priority observed client lacks a reproducible live
profile across Linux and Windows acquisition routes.

Approach: Automate the Web UI or command controls, compare official releases
with package-manager routes, and capture all core observer surfaces.

Prove: a trusted run publishes agreeing Linux and Windows profiles for the
same stable version with two verified routes per host.

### What was measured on 2026-09-05, and why the entry stays open

⛔ **The acceptance cannot run on a session host, and three routes were tried
before saying so.** It is the same shape as `OBS-01`'s: an acceptance naming a
Windows capture that this repository is not permitted to perform at all.

1. Run it whole. Refused on a session host: `--egress` exits 1 there, so
   installing and driving a client would be the capture that boundary exists to
   refuse. ⚠ **That is a fact about a session host and never was one about
   hosted runners.** A runner is a fresh virtual machine per job and its default
   route can be deleted before the install, which is what `CI-03`'s workflow
   does; the Windows guard pair exists now too.
2. Run the Linux half alone. Refused for the first half of the same reason: a
   capture on one platform is still a capture, and the boundary does not care
   how much of the acceptance it satisfies.
3. Split off the provable prefix and close that. ⭐ **Partly possible, and the
   part that is was done.** Deciding which version to acquire needs no host, so
   it was run. What it cannot do is leave a record: `CORPUS-01` owns the
   append-only store and is open, so a measurement has nowhere durable to go and
   would be a file nobody can cite. The split is therefore not filed as an entry
   yet; it becomes one once there is a store to write into.

⭐ **The resolver met a real target for the first time.**
`sh scripts/acquisition/fetch-releases.sh qbittorrent/qBittorrent <file>`
answered through `https://api.gh.pkgforge.dev/`, which is the route
`docs/AGENTS.md` rule 8 names, and
`cargo run -p bit-ids --example resolve-stable -- qbittorrent release- 3 3 ...`
selected **5.2.3**, published 2026-07-07, over three superseded candidates with
every verdict kept and a digest of the bytes that were read. ⚠ One source is not
two: the resolution is single-sourced and `ACQ-01`'s independence rule is about
acquiring the artifact, which is a separate requirement this did not touch.

⛔ **The listing answered with four releases, and that is a property of the
source rather than of the mirror.** It looked like truncation and it is not:
page two of the same endpoint is empty, so four is the whole answer, while the
`tags` endpoint on the same route answers with at least a hundred, beginning
`release-5.2.3`, `release-5.2.2`, `release-5.2.1`, `release-5.2.0`,
`release-5.1.4`. The project tags far more versions than it publishes as
releases.

⭐ **That matters to every future resolution and it is not a defect in the
resolver.** `resolve` decides from the candidates it is handed, and the caller
chooses the endpoint; a caller that reads only `releases` is selecting from a
different and much smaller population than the project's versions, and a
resolution saying `candidates: 4` is a claim about that endpoint rather than
about the target. It changed nothing here because `5.2.3` leads both
populations, and it would change the answer for a target that stops minting
release objects. ⚠ A second source for a resolution has to be a different
endpoint or a different index, not the same one asked twice, which is the
independence rule `ACQ-01` already states for routes.

Measured from the same listing, the release offers a Linux `AppImage`, a Windows
`x64_setup.exe` and a source `tar.xz`, each with a detached `.asc` signature
beside it. That is two platform routes and a signature disposition to verify, and
`ACQ-05` owns the authenticity evidence.

### The observer a client can actually be pointed at, measured 2026-09-08

⭐ **The lab now hands out a torrent that names its own tracker**, which is what
a stock build needs and what nothing here provided before.
[`client-capture`](../crates/bit-ids-probe/examples/client-capture.rs) runs the
real `HttpTracker` observer, generates the torrent with `announce` set to the
endpoint the operating system just gave it, writes it to a path, and prints the
address, the info hash and the fixture digest before the line a driver waits on.

⛔ **`evidence-bundle` could not be driven by a client and that was structural
rather than an omission.** Its torrent carries a hard-coded
`http://127.0.0.1:6969/announce`, so the only thing that could reach its
observer was something told the endpoint separately - which `curl` is and a
stock build is not. A client reads the address out of the file or it does not
announce at all.

⭐ **A reader this project did not write confirmed the file says what the
observer claims.** `torf` 4.3.1, from the package index into a virtualenv, took
the announce URL out of the generated `.torrent` and it matched both the address
printed by the observer and the port `curl` then announced to; `torf` also
re-derived the info hash `4bc6a5c90be5c4fc6a3c4281f2400af2d2c7700d` from the
metainfo in the written bundle, which the observer had printed independently.
⚠ Parsing a file is not a capture and needs no disposable host.

Driven on 2026-09-08 with `curl/8.5.0` announcing as `driven-by-curl-000001`:
one announce kept, query key order
`info_hash,peer_id,port,uploaded,downloaded,left,compact,event,key`, header
order `Host,User-Agent,Accept`, two segments, both artifacts verifying.

⚠ **The peer surface is dialled rather than offered, and the reason is a
constructor.** `TrackerResponse` is cloned into the responder while the lab is
still being built, and `Lab` binds port zero, so a tracker answer naming this
lab's own peer port would have to predict a port that does not exist yet. The
client's listen port is an argument instead and the lab dials it once an
announce proves the build read the torrent. ⭐ That is also the stronger role:
the side that dials sends its handshake first, so what comes back is the build
answering.

⚠ **What this still does not establish is a client.** No build is installed on
any host this project may capture on, so every run of it so far was driven by
`curl`. The adapter is what changes that.

### The adapter, written 2026-09-08

`qbittorrent.sh` is the highest
priority target and not the simplest one. `qbittorrent-nox` keeps a profile
directory, asks once for a legal notice, and reads its switches out of an INI
file, so the adapter writes the whole profile rather than patching whatever is
in `HOME`: a capture that edited an existing configuration would measure a build
under settings the run did not record.

⭐ **The torrent is a positional argument**, which is the one unattended control
this product already has that needs no web interface, no credential and no
second process. An adapter driving the WebUI would be measuring a build through
an authenticated API it had to configure first.

⚠ **`--confirm-legal-notice` is not optional.** Without it the first run of a
fresh profile blocks on a prompt, and the capture would sit there until the
observer's deadline with nothing on the wire and no error to read.

⛔ **None of that is measured yet.** The adapter has never installed anything:
this session could not run it, because a session host is not disposable, and the
dispatch is what establishes whether it works. The Windows half of the Prove
above is untouched.

### ⛔ The release listing was read once and summarised one build short

**Measured 2026-09-09, re-reading the same endpoint.** The paragraph above says
release-5.2.3 "offers a Linux `AppImage`, a Windows `x64_setup.exe` and a source
`tar.xz`, each with a detached `.asc` signature beside it". It carries
**fourteen** assets, and two of them are Linux AppImages:

```text
qbittorrent-5.2.3_x86_64.AppImage
qbittorrent-5.2.3_lt20_x86_64.AppImage
```

⛔ **The summary was one build short in exactly the place a route has to
choose.** A pattern written from it - `qbittorrent-*_x86_64.AppImage` - matches
both, and a selector that took the first match would install whichever the vendor
listed first, with the record reading the same either way.
`qbittorrent-{version}_x86_64.AppImage` matches one, and two matches is a refusal
rather than a first-match answer. `ACQ-02` carries the selector.

⚠ **What is still not measured is which libtorrent each AppImage carries.** The
suffix plainly names a variant and nothing here has established either, so the
adapter selects the vendor's unsuffixed build and the record names the asset it
took. A capture through this route measures whichever it installed, and says
which.

⭐ **And the resolver skipped a prerelease on a live listing rather than on a
fixture.** The newest release object on 2026-09-09 is `release-5.3.0beta1`, which
the source flags a prerelease and whose version text says so too; the resolution
selected 5.2.3 from five candidates. That listing is the recorded response
`check-release-route` runs over, kept whole for exactly that reason.

### What the info hash cost, measured 2026-09-08

⛔ **Writing that measurement into this record turned CI run 67 red on both
lanes**, and the check was right to fire: an info hash is forty hex digits, and
`check-no-secrets --public` hunts long hex because that is the shape of a
credential. ⭐ The rule is narrowed rather than switched off, to the phrase and
both backticks this project spells one with; a bare forty-digit run elsewhere in
a sentence is still a finding. ⚠ A local gate had been green over this file
minutes earlier, because the checks re-run after the edit were the ones the edit
looked like it touched and `--public` is a second invocation of a check whose
first invocation passes.

⚠ **Residual, from the door sweep of 2026-10-14:** `qbittorrent.sh` writes its
discovery settings into the profile and nothing reads back what the running
build applied; `CLIENT-06` carries it with Transmission's.

### ⛔ No second Linux route for 5.2.3 builds on this runner image. 2026-10-14

Measured on 2026-10-14 while looking for the pair `CLIENT-06` found for
Transmission:

| candidate | what it carries | why it is not a half |
| --- | --- | --- |
| Ubuntu noble's package | `qbittorrent-nox 4.6.3-1build2`, read from noble's own index | another version; `AGENTS.md` rule 5 forbids backfilling |
| the qBittorrent team's stable PPA | `4.6.7.99~202504281220…` for noble, read from Launchpad's API | a development snapshot of 4.6, not 5.2.3 |
| the `release-5.2.3` tag, built | needs Qt 6.6.0 or newer, by its own `CMakeLists.txt`; noble's `qt6-base-dev` is `6.4.2+dfsg-21.1build5` | the build cannot configure against the distribution's Qt, and building Qt first is a second product's build |
| the `_lt20` AppImage | the same vendor, listing and delivery as the unsuffixed one | `E-ACQ-07` and `E-ACQ-08` call it the same route |

⛔ **So the Linux pair for this target waits on a host whose distribution
carries Qt 6.6 or newer, or on a decision to build Qt as part of the route.**
That decision belongs to the operator. The Prove's Windows half, the vendor's
`x64_setup.exe` against a package manager's, is the other way to a pair. It
needs a Windows lane in `capture-client.yml`, which has only a Linux job today.

⭐ **Settled by the operator on 2026-10-15: 5.2.4, through a source build
against Qt's own prebuilt archives.** An explorer found the same day that
`release-5.2.4` was published on 2026-09-28, so the release route now selects
it, and the recorded listing predates it. Qt 6.11.2's prebuilt `linux_gcc_64`
archives are on `download.qt.io`, each with a `.sha1` beside it, built on a
glibc older than noble's. That is downloading a built Qt rather than building
one. noble's `libtorrent-rasterbar-dev` 2.0.10 and Boost 1.83 meet 5.2.x's
minimums. The `release-5.2.4` tag is signed by the vendor. Other candidates
were weighed and set aside:

- Ubuntu 26.10's `qbittorrent-nox` `.deb` is 5.2.3 only.
- Debian sid carries 5.2.4.
- The userdocs static build is a release asset, which `E-ACQ-08` calls the
  AppImage's delivery.
- The vendor's container image builds libtorrent from a moving branch.

⭐ **Measured on 2026-10-15, by reading each AppImage's squashfs without running
it.** The squashfs starts where the ELF runtime's section headers end. Neither
AppImage was executed.

| asset | digest prefix | libtorrent | peer prefix it embeds | Qt | binary |
| --- | --- | --- | --- | --- | --- |
| `qbittorrent-5.2.4_x86_64.AppImage` | `9ab0e97a3245a1d5` | 1.2.20 | `-LT12K0-` | 6.10.3, GCC 11.4.0 | `usr/bin/qbittorrent` |
| `qbittorrent-5.2.4_lt20_x86_64.AppImage` | `eda6c0ca39f9befb` | 2.0.15 | `-LT20F0-` | 6.10.3, GCC 11.4.0 | `usr/bin/qbittorrent` |

⛔ **Three things the adapter assumed were wrong.** The release asset it selects
carries libtorrent 1.2, and the settled source build linked 2.0. The reserved
bytes a pair compares are libtorrent's, so the two routes would have diverged
over the library rather than the product. Neither AppImage carries
`qbittorrent-nox`; both carry the GUI `qbittorrent`. And the only platform
plugins bundled are `libqxcb.so` and `libqwayland.so`, so the binary cannot start
without a display server.

⭐ **Settled by the operator the same day:**

- the release route takes the `_lt20` AppImage;
- the source route links libtorrent 2.0.15 built from its own tag;
- both routes run the GUI build under Xvfb started inside the capture namespace;
- Qt stays 6.11.2 from the prebuilt archives, though the AppImage bundles 6.10.3.

⭐ **Both routes built, installed and captured on this host on 2026-10-15.**

- **Release route.** `resolve-release` selects
  `qbittorrent-5.2.4_lt20_x86_64.AppImage`, 101542392 bytes, and
  `install-step.sh` installs it in 12 seconds. It answers `5.2.4`.
- **Source route.** `resolve-source` selects `release-5.2.4`, and
  `install-step.sh` builds it from nothing in 11 minutes 55 seconds, under the
  1500-second bound, at commit `a1e4649c`. That covers Qt's five archives,
  noble's packages, libtorrent `v2.0.15` and qBittorrent. It answers `5.2.4`.
- **Captures.** Each route captured three sessions, 60 seconds, in the
  namespace, under the adapter's own Xvfb: 6 announces, 6 peer streams, 3
  sessions dialled, and every peer ID `-qB5240-` with a new tail per session.
- **Assembly.** Assembled by hand from the two drives, both records are
  `publishable` and `classify_across` answers `build_equivalent`, with 4
  overlapping fields agreeing. The run manifests are absent only because the
  hand-made layout carries no barrier or host claim, which a dispatch writes.

⛔ **The drive found three defects before anything shipped.**

1. The build's run path names Qt alone, so `libQt6Gui`'s `libEGL.so.1` was
   found nowhere at link time. The build now passes `-rpath-link` to the
   unpacked runtime set.
2. `start` set `HOME` before the runtime directory was resolved, and the prefix
   follows `HOME`, so every session loaded no libraries. The directory is now
   resolved first and handed in.
3. `lrelease` from qttools loads qtdeclarative's `libQt6Qml`, so that archive
   is fetched too.

⚠ The release route's runtime set is the same packages as the source route's.
Both load noble's GL and xcb libraries from the prefix, and Qt's xcb plugin
named eight a runner need not carry. `check-rootless` stubs `apt-get` and
`dpkg-deb` for this adapter's case, because fetching for real cost the gate 15
seconds of network.

⛔ **The first dispatch, `capture-client` run 38702739119 on `ab83dc5`.**

- **Release lane: green.** It installed in 11 seconds and captured on the
  runner, so Xvfb and the runtime set work there.
- **Source lane: built, then could not run.** It compiled in 7.5 minutes,
  faster than this host, and stopped at `--version` on Qt's own message about
  `xcb-cursor0`, which Qt prints whatever dependency is missing.

The runtime set had been the eight libraries this host lacked, and a runner
lacks more. It is now the whole closure `ldd` resolves for Qt's xcb plugin, Qt's
libraries and the build, outside Qt and outside what every runner carries:
glibc, libstdc++, zlib, OpenSSL and the compression and crypto libraries. That
adds the X11 and xcb core libraries, fontconfig, freetype, libpng, D-Bus and
their own dependencies. Checked before the second dispatch: every one of the 47
libraries the closure resolves is either in the new set or in that carried list,
with none uncovered.

⭐ **PUBLISHED, 2026-10-15.** `capture-client` run 38705799000 on `63e6fcd`
captured both routes on runners. `assemble-bundle` run 38707559892 assembled
them onto the published tree: both records `publishable`, `build_equivalent`,
and 4 fields agreeing. The release record is filed under `appimage` and the
source record under `elf-binary`. `publish-data` run 38708274728 appended it
after the dry run 38707944536 passed. `data` is at `bfeab59`. ⭐ Read back from a
fresh clone: 71 files, `sha256sum -c` clean, *valid store: 6 record(s), 6
run(s), 71 object(s); 6 of 6 record(s) may enter a published view*.
`schemes.txt` names `qbittorrent:release-:3:4` beside the other two targets.

⛔ **And the source route's resolver fails closed today, measured on
2026-10-15** while building `CLIENT-06`'s floor. The live refs carry 260
candidates, and 59 are `unorderable` under `release-` with three or four
components. 53 lead with 1, 2 or 4, which a floor of 5 would release. Six do
not: `release-5.0.0beta1`, `release-5.0.0rc1`, `release-5.1.0beta1`,
`release-5.1.0rc1`, `release-5.3.0beta1` and `release-5.3.0rc1`. They are
prereleases that `text_marks_prerelease` misses, because the marker follows a
digit inside one token, `0beta1`, and the reader takes letters only from a
token's start. The release route never met them, because the listing flags
them. So this route needs two changes: a floor, and a token reader that also
finds a marker after leading digits. The second changes what the project calls
a prerelease, so it gets its own suite and mutants, not a ride on the first.

⭐ **The token reader was built on 2026-10-15, and it was the only change
needed.** Every run of letters inside a token is asked now, so `0beta1` and
`0rc10` carry their marker. A scratch reading over every target's live tags
first found the 59 tags moving from `unorderable` to a prerelease, every one of
them qBittorrent's, and nothing else moving anywhere. No floor is needed:
driven on the live refs, the source route selects `release-5.2.4` with 59
`prerelease_by_version` and 200 `superseded`. Transmission, aria2-next and
aria2 answer as before. ⛔ `check-source-route` had planted `v2.0.0beta1` as a
tag no scheme can order, and the change turned that row red. It is now two rows:
that spelling is passed over and the stable newest still wins, and a tag with
no marker, `v2.0.0x1`, still blocks. 26 of 26. A mutant that reads letters from
a token's start only is killed by the unit test.

### ⭐ The Windows half of the adapter is built, and not yet run. 2026-10-15

[`../tools/check/adapter_qbittorrent_windows.go`](../tools/check/adapter_qbittorrent_windows.go)
registers `qbittorrent` on Windows, run as `bit-check adapter qbittorrent`.
[`../tools/check/qbtwindows.go`](../tools/check/qbtwindows.go) holds what it
decides without Windows in it, and `bit-check qbittorrent-windows --selftest`
asks each of those functions on any host: 35 cases, all held, and two mutants
were refused (DHT left on, and the NSIS plugin directory read as payload).
`check-bitcheck` runs the selftest.

| route | what it does | `package` |
| --- | --- | --- |
| `release` | `qbittorrent_5.2.4_lt20_x64_setup.exe` from GitHub, fetched by `install-rootless --fetch` under the resolution's digest, then extracted by the runner's 7-Zip into `<prefix>\opt\qbittorrent`. No installer code runs | `nsis-payload` |
| `package` | `winget install --id qBittorrent.qBittorrent.lt2`, with no `--version`, so winget's own index is the resolver, into `%ProgramFiles%\qBittorrent` | `nsis-installer` |

⭐ **winget is the one manager that installs the `_lt20` build at 5.2.4**,
measured through `AGENTS.md` rule 8's route on 2026-10-15:

| manager | 5.2.4 | build |
| --- | --- | --- |
| winget | `manifests/q/qBittorrent/qBittorrent/lt2/5.2.4/`, PackageIdentifier `qBittorrent.qBittorrent.lt2`, PackageName *qBittorrent (libtorrent 2)* | `_lt20`. Its installer URL is SourceForge's `qbittorrent_5.2.4_lt20_x64_setup.exe`, and its InstallerSha256 `28888FE5...928CFA` is the digest GitHub's listing states for the release route's asset |
| Chocolatey | `qbittorrent` 5.2.4 | default only: the package embeds `qbittorrent_5.2.4_x64_setup.exe`, and `chocolateyinstall.ps1` takes no parameter that chooses another |
| Scoop (extras) | `qbittorrent` 5.2.4 | default only: `qbittorrent_5.2.4_x64_setup.exe` from SourceForge |

So both Windows routes deliver one installer's bytes, resolved by two indexes
and delivered by two hosts. That makes this the strongest pair `ACQ-03` can be
given: the same build, acquired twice.

⛔ **The Windows GUI build has no `--version`.** That was read from
`release-5.2.4`'s source. `cmdoptions.cpp` declares the option under
`#if !defined(Q_OS_WIN) || defined(DISABLE_GUI)`, so on Windows the flag is an
unknown parameter, and `main.cpp` answers an unknown parameter with a modal
*Bad command line* box. So `version` starts the build in a throwaway profile,
with the Web UI on a free loopback port and every surface off, and asks
`/api/v2/app/version`. That route returns `QBT_VERSION`, compiled into the
executable. It also keeps `/api/v2/app/buildInfo` (Qt, libtorrent, Boost,
OpenSSL, zlib) on stderr, which is what tells `_lt20` from the default build.
The adapter then stops the build through `/api/v2/app/shutdown`. ⚠ This is
the one adapter whose `version` starts the product, and the contract says so.

⚠ **The GUI build will not start its Web UI with no password hash**
(`webui.cpp`: *Credentials are not set*). So the profile holds a PBKDF2 hash
of 32 random bytes that are discarded, and `LocalHostAuth=false` admits
loopback without a session. `stop` scrubs every password key out of the
profile before the workdir is packed, and `capture-client`'s rule-12 scan is
the second check. The source showed that five settings default differently
from what a capture needs: the tray and close-to-tray, the exit confirmation,
the update check and the GeoIP fetch. The profile turns each of them off, and
the file logger too.

⛔ **What needs a runner, in the order to read a dispatch:**

1. Whether `winget` exists and works on `windows-2025` under the runner's
   account. The package lane's `install.log` and `winget-show.txt` show it.
2. Whether the GUI build creates its window and opens its Web UI in the
   runner's session. `version.err` beside the install record carries
   `buildInfo`, or the build's output if it never answered.
3. Whether `7z x` lays the payload out as `qbittorrent.exe` with its DLLs
   beside it. `extract.txt` names the executable found.
4. Whether the Web UI's shutdown ends the build, so it announces `stopped`.
   `client/stop.txt` says `webui-shutdown` or `killed`.
5. The capture itself: announces from `-qB5240-` under the firewall rules.

### ⭐ The Windows half captured on both routes, first time. 2026-10-15

Capture-client run 38788416994 on `7735dff`:

- **Release route:** the `_lt20` installer, extracted with 7-Zip.
- **Package route:** winget `qBittorrent.qBittorrent.lt2`. Its index pins
  SourceForge's copy of the same installer.

Each lane installed 5.2.4, was blocked at the firewall (proved by
`probe-egress`), and made 6 announces as `-qB5240-`. Evidence verified by
sha256sum. The Linux pair in the same run captured through the Go adapter.

⚠ **Not yet a record.** The first assembly refused the Windows lanes, and
`CI-11` names the four gaps it found. A re-capture on the fixed tree and its
assembly is what moves this Prove.

### ⭐ The Windows pair is published, and agrees with Linux. 2026-10-16

- **Capture:** capture-client run 38838965380 on `05d0d21`, Windows only.
- **Assembly:** assemble-bundle run 38840840749.
- **Publish:** publish-data run 38841340795 appended 111 objects to `data`,
  read back and verified.
- **Records:** at `windows/x86-64/nsis-payload` (the 7-Zip extraction) and
  `nsis-installer` (winget), both `publishable` and `build_equivalent` across
  the routes.
- **Independent check:** a fresh clone of `data` passes `sha256sum -c` and
  `validate-corpus`, with 14 of 14 records in the published view.

Published Linux and Windows profiles agree on every observed field (peer-wire
`peer_id` and `reserved`, tracker `peer_id` and `user_agent`), read from `data`
at `4221e28`.

⚠ **Left OPEN on purpose, decided unattended.** The Prove reads "a trusted run
publishes agreeing Linux and Windows profiles". The two platforms' published
profiles come from two runs: Linux from 38705799000 and Windows from
38838965380, at one version. The one run that captured all eight lanes,
38820888070, assembled and agreed but could not be published, because the
store never rewrites the Linux records already on `data`. Whether two runs
satisfy the Prove is the operator's call; `TODO/PROGRESS.md` records it.

## CLIENT-02: qBittorrent Enhanced capture adapter

Source: operator scope and upstream Enhanced Edition repository
Priority: P1 | Effort: L | Status: OPEN

Problem: Enhanced Edition may alter observable identity and must not inherit a
qBittorrent profile by name or source similarity.

Approach: Acquire and drive it independently, retaining flavor metadata and
comparing observations only after both profiles exist.

Prove: two-route live runs either publish a distinct corroborated profile or
prove byte-for-byte observable equivalence without copying source-derived data.

## CLIENT-03: uTorrent capture adapter

Source: operator scope and August 2026 priority sample
Priority: P1 | Effort: L | Status: OPEN

Problem: The proprietary Windows client lacks reproducible active evidence.

Approach: Use a disposable Windows runner, official and package-manager
routes, silent automation where supported, and UI automation only as a bounded
fallback.

Prove: two same-version Windows installations emit agreeing connector records
and leave no state or traffic outside the laboratory.

## CLIENT-04: BitComet capture adapter

Source: operator scope and bit-cli historical identity mismatch
Priority: P1 | Effort: L | Status: OPEN

Problem: Historical BitComet-like identity assumptions make guessed mapping
especially unsafe.

Approach: Drive official and package-managed Windows builds through the lab
and retain raw handshake, tracker, and extension behavior.

Prove: a two-route Windows capture passes agreement without consulting a
source-code peer-ID mapping.

## CLIENT-05: aria2 capture adapter and connector

Source: operator scope and requirement for an independent CLI connector
Priority: P1 | Effort: L | Status: OPEN

Problem: aria2 is both a required target and a useful independently controlled
client, but those roles must not create circular corroboration.

Approach: Use JSON-RPC for lifecycle control, capture aria2 as a target with a
separate packet oracle, and use it as a connector only for other targets.

Prove: tests reject aria2 self-corroboration and a live two-route capture
publishes a profile supported by the Rust observer plus packet decoding.

### The adapter, written 2026-09-08

⭐ `aria2.sh` is the simplest target in
the matrix. `aria2c` takes every setting on its command line, needs no daemon,
no profile directory and no interactive acceptance, and prints its version to
stdout, so nothing about one run of it depends on state a previous run left
behind.

⚠ **This is the target half alone and not the connector.** Driving aria2 as a
target and using it to corroborate another target are the two roles this entry
says must not create circular corroboration; nothing in that file observes
anything.

### What the four dispatches actually establish, read 2026-09-08

⭐ **The install logs answer the hang, and they answer it by refuting the
question.** `install-aria2-<run>-1` from client capture runs 3 and 4 was
downloaded through the route `AGENTS.md` rule 8 names, unauthenticated, and both
`install.log` files say the same thing:

```text
aria2 is already the newest version (1.37.0+debian-1build3).
0 upgraded, 0 newly installed, 0 to remove
```

⛔ **So no package was ever installed.** `apt-get install aria2` on
`ubuntu-24.04` is a no-op, `needrestart` runs only after a package operation, and
`NEEDRESTART_MODE` therefore could not have been the cause under any letter. Run
3 carried `a` and run 4 carried `l`; the two jobs hung identically. ⚠ Three
sessions of reasoning about that variable were reasoning about a code path the
runs did not take.

⛔ **And the hang is not in the install step.** The job step records, read from
the same route, put it somewhere else entirely:

| run | install step | where the job stopped |
| --- | --- | --- |
| 1 | never returned | *Install the client*, unbounded |
| 2 | never returned | *Install the client*, a bound that sent TERM and waited |
| 3 | success in 6s | *Upload the install logs* |
| 4 | success in 6s | *Upload the install logs* |

⚠ **The uploaded artifact from both hung runs is complete and downloadable**, so
that step's work finished and the step still never returned. What holds a runner
open after its work is done is not something these four runs can separate, and
naming a cause would be a guess with an artifact beside it rather than a
measurement. It is `CI-08`'s shape - a runner default nobody swept - and it is
recorded there rather than guessed at here.

⭐ **The adapter itself is sound, and this is its first driven pass.** Run on a
real `ubuntu-24.04` host: a genuine install in 8s, the no-op repeat in 3s, and
`version` answering `1.37.0` in under a second. Two of the four assumptions in
the file's own unmeasured block are now measured: `aria2c --version` opens
`aria2 version 1.37.0`, so the third field is the version, and the package route
installs without a prompt. The two that remain are about `start`, which needs a
capture.

⛔ **The finding that outlives the hang is what the no-op means.** A route that
installs nothing exits 0, and `install-client` reported `1.37.0` through the
`package` route for a binary that came with the image. `acquired` is the field
that says so now, `ACQ-03` carries what two such routes do to the two-route rule,
and the guard is proved by planting the derivation in
[`../scripts/capture/check-capture-client.sh`](../scripts/capture/check-capture-client.sh).

### The second route exists, and what it costs, measured 2026-09-08

⭐ **aria2 is the one target in the matrix whose two routes currently resolve the
SAME version.** Ubuntu 24.04 ships 1.37.0 and the vendor's newest release is
`release-1.37.0`, published 2023-11-15 - aria2 has not released since. ⛔ The
other two are not close: Ubuntu ships Transmission 4.0.5 against upstream 4.1.3,
and qBittorrent 4.6.3 against upstream 5.2.3, and `AGENTS.md` rule 5 forbids
backfilling an old version to make a pair agree. So the first two-route capture
this project can attempt is this entry's, not `CLIENT-01`'s.

⛔ **The vendor publishes no Linux binary**, which is a measured property of the
target rather than a gap in the adapter: the 1.37.0 release carries source
tarballs, two Windows zips and an Android build. A Linux release route is
therefore a source build, and it was driven here: 19 seconds to configure and 126
seconds to `make -j4`, producing a binary that answers `aria2 version 1.37.0`
with BitTorrent enabled.

⚠ **Two builds, one version, different features and different bytes.** `ACQ-03`
carries the table and the consequence; the short form is that the package route's
`/usr/bin/aria2c` is a 14-kilobyte shim over `libaria2.so.0` while the source
build is self-contained, and their feature lists differ.

⭐ **The release route installs now, and was driven end to end here.** It fetches
the tarball, unpacks it, configures, builds and installs into a prefix
`binary()` prefers, and refuses at each step rather than returning 0 over an
install it did not perform. 132 seconds from URL to a runnable `aria2c` that
answers `1.37.0`. ⚠ It deliberately does not `apt-get` its build dependencies:
that would make the release route reach the package index, and arguing afterwards
about whether headers count as acquisition is worse than refusing.

⛔ **Still not captured.** No run of this adapter has started a build against the
lab, so the entry stays open on the same four gaps every client entry has. What
changed is that the second route now exists rather than being fetched and
discarded.

### ⭐ The second route is wired into the workflow, 2026-09-09

**`capture-client.yml` can now dispatch it**, which it could not before: the
route is a matrix dimension beside the adapter, and the artifact it fetches is
resolved in a step of its own before the route is cut.

⛔ **What was actually missing was the artifact, not the plumbing.** This
adapter's release route refuses without `BIT_IDS_RELEASE_URL` "resolved before
the route was cut", and nothing in the repository produced one - `resolve-stable`
orders versions and selects no asset. `ACQ-02` carries the half that closes it;
what lands here is the declaration:

```text
release_repo=aria2/aria2
release_tag_prefix=release-
release_min_components=3
release_max_components=3
release_asset=aria2-{version}.tar.bz2
```

⚠ **`bz2` rather than `gz` or `xz`, recorded rather than implied.** The release
carries all three and they are the same source; the `bz2` is the one `ACQ-03`
already built from, so its digest is a control this project can check a fetch
against without acquiring anything new. ⛔ A pattern written `aria2-*.tar.*`
matches all three and is refused rather than resolved - measured, as a case in
[`../scripts/acquisition/check-release-route.sh`](../scripts/acquisition/check-release-route.sh).

⭐ **Driven here, end to end, against the live listing**: 24 candidates, 1.37.0
selected, one of six assets matched, and the URL that came out served bytes
`sha256sum -c` verified against the digest the 2026-09-08 measurement recorded.

### ⭐ Client capture run 5: a release route acquired a build on a capture host

**Dispatched 2026-09-09 as `["aria2"] × ["release"]`**, deliberately without the
package route: that lane is the one with the unexplained hang, and running it
again would have cost twenty-five minutes to re-observe something four dispatches
already established. What was unknown was whether the new path works at all.

| step | measured |
| --- | --- |
| *Resolve the release artifact* | 03:14:53Z to 03:14:54Z, **one second**, success |
| *Install the client* | 03:14:54Z to 03:17:18Z, **144 seconds**, success |

⭐ **That is the first time any `release` route in this tree has acquired
anything on a host this project may capture on.** It fetched the tarball,
configured, compiled and installed into `/usr/local`, and the executable answered
`aria2 version 1.37.0`.

⭐ **And the tarball the runner fetched is the one this project already
measured.** Its bytes came down inside the install artifact, unauthenticated
through rule 8's route, and `sha256sum -c` verifies them against the digest
`ACQ-03` recorded on 2026-09-08 from an entirely different fetch.

⛔ **Two builds, one version, measured on the capture host rather than argued
for.** `install-client` asks the adapter for a version before the route runs and
after, and both answers are in the artifact:

| | before the route | after the route |
| --- | --- | --- |
| version | `1.37.0` | `1.37.0` |
| features | Async DNS, BitTorrent, Firefox3 Cookie, GZip, HTTPS, Message Digest, Metalink, XML-RPC, SFTP | BitTorrent, GZip, HTTPS, Message Digest |
| libraries | zlib/1.3 libxml2/2.9.14 sqlite3/3.45.1 GnuTLS/3.8.3 nettle GMP/6.3.0 c-ares/1.27.0 libssh2/1.11.0 | zlib/1.3 OpenSSL/3.0.0m |
| compiler | gcc 13.2.0 | gcc 13.3.0 |

⚠ **The TLS library is not the same one.** A record holding `1.37.0` and nothing
else would say these two builds are one, and one of them speaks HTTPS through
GnuTLS and the other through OpenSSL. That is the argument for keeping the
build's whole answer, arriving from a runner instead of from a session host.

⛔ **The capture did not run**, because the step after the install hung. The next
section is what that establishes.

⚠ Nothing here establishes that the runner image would carry the toolchain if it
changed; it carries one today, measured by a build that completed.

### ⛔ A SIXTH READING, REFUTED ON 2026-09-09 BY A CONTROL IN THE SAME JOB

⛔ **"The version call against the preinstalled `aria2c` is what hangs" is
WRONG.** That reading was the sharpest correlation this entry had: `aria2` ships
on `ubuntu-24.04`, so an aria2 lane's *Install the client* asks the adapter for a
version and that call executes a real binary, while a transmission lane finds no
binary and runs nothing - present in every hung run, absent from every green one.

⭐ **`CI-08`'s probes were built to test exactly that and had never been
dispatched against `aria2`.** capture-client run 13, on 2026-09-09, is the first
aria2 lane to run them. *Probe the preinstalled product* executes
`timeout 30 sh scripts/capture/adapters/aria2.sh version` - the same call, on the
same host, in the same job, one step earlier.

**It took 0 seconds and succeeded.** The very next step, *Install the client*,
hung as it has in all ten previous dispatches, and run 13's terminal state was
read back: `cancelled`, **zero artifacts**, no log - the same ending as every
other aria2 lane. ⚠ The other two probes also returned in 0 seconds, so host
resources and `sudo` are cleared as well.

⛔ **So the call is not slow and is not stopped**, and the correlation was a
correlation. ⚠ What survives is narrower and better: whatever hangs is inside
*Install the client* and is NOT the adapter's `version`, NOT `describe`, NOT
`sudo` itself and NOT host resources - the other two probes also returned in 0
seconds. What that step still does and the probes do not is run `install-client`
under `sudo -E`, which for the `package` route is an `apt-get update` and an
`apt-get install`, and for the `release` route is a source build.

⚠ **A refuted reading is not a diagnosis.** This narrows where to look and names
nothing. ⛔ Do not record a seventh reading without something that separates it
from the sixth, which looked stronger than any of them and was still wrong.

### ⛔ Run 39577251480: the package lane cannot run, and the release lane was refused by a guard. 2026-10-21

`capture-client` run 39577251480 on `e33b476` dispatched `adapters: ["aria2"]`
and `routes: ["package","release"]`. It read as follows:

| lane | where it ended | why |
| --- | --- | --- |
| `package` | *Install the client* | `apt-get update` answered *Could not open lock file /var/lib/apt/lists/lock - open (13: Permission denied)* |
| `release` | *Capture*, refused | aria2 1.37.0 installed from the vendor's tarball and captured: 3 announces, 6 peer streams, the bundle verified. Then *the transcript does not carry the peer ID the observer reported* |

⛔ **The package lane is `ACQ-06` working as designed.** Installs are rootless,
and a package route is privileged by construction, so it is kept off the
capture path. aria2 needed a second route that is not privileged, and the
next section adds one.

⛔ **The release lane's refusal was a defect in `capture-client`, not in the
capture.** aria2 1.37.0's peer ID is `A2-1-37-0-` followed by ten **random raw
bytes**, for example `A2-1-37-0-` and hex `a87ccb024e613d485817`. The tracker
transcript stores the request's bytes, and in them `peer_id` is
percent-encoded: `peer_id=A2-1-37-0-%A8%7C%CB%02Na%3DHX%17`. The guard asked
`bytes.Contains(transcript, hex(peer ID))`. That works only when every byte of
the peer ID is URL-unreserved and travels verbatim, which held for every
client before this one.

### ⭐ The guard reads the request, and the old test is planted. 2026-10-21

[`../tools/check/transcriptpeer.go`](../tools/check/transcriptpeer.go) is the
guard now, and `capture-client` calls it:

1. It reads the transcript as JSON, schema `bit-ids/transcript/1`.
2. It takes each `from_target` segment and requires lowercase hex.
3. It decodes the segment and takes the request line's query.
4. It percent-decodes the `peer_id` field. ⛔ `+` stays the byte 0x2b and is
   never read as a space: that is form encoding, and a peer ID is opaque bytes.
5. It compares the decoded bytes, as hex, with the observer's peer ID.

⚠ No Go reader of transcripts existed to reuse. `assemble.go`'s `transcript`
and `segmentsAt` are fixture writers, closures inside a harness. The connector
does this decoding in Python with `urllib.parse`, and this is a third reading
with its own code.

⭐ **`bit-check capture-client --selftest`: 15 cases, all held.** They include:

- aria2's percent-encoded raw-byte peer ID is found, in either hex case;
- an unreserved-ASCII peer ID is still found, sent verbatim or fully escaped;
- a peer ID the transcript does not carry is refused, and so is one a byte away;
- a literal `+` is 0x2b and is not decoded as a space;
- a peer ID that is only in a `to_target` segment, or only in a field other
  than `peer_id`, is refused. The old test accepted both, because it read the
  whole file;
- a truncated escape is not read as a peer ID;
- a document that is not JSON, has another schema, or holds uppercase hex is
  an error, not an absence.

⛔ **The old defect is planted and refused.** `check-bitcheck` writes the old
substring test into the first line of `transcriptCarriesPeerID` in its scratch
tree, builds that copy and runs its selftest. The selftest must exit 1 and must
name aria2's case. Measured: *the old substring test, planted, is refused by
the selftest on aria2's peer ID: go (exit 1)*.

⚠ **`capture-run`'s token check has the same shape and is not affected.** It
asks `bytes.Contains(transcript, hex("key=" + run id))`. The run ID is refused
unless it is `[a-z0-9-]`, which is all URL-unreserved, and `curl` sends the URL
it is given unencoded. So the bytes on the wire are the bytes compared. It is
left as it is, and the same reasoning sits beside it here.

### ⭐ aria2 has a `source` route, and both rootless routes captured on the session host. 2026-10-21

`bit-check adapter aria2 install source` clones the tag `resolve-source`
selects from `github.com/aria2/aria2`'s refs, and builds it into the same
prefix as the release route, with the same configure line. That is the shape
Transmission's and aria2-next's source routes have.

⛔ **The resolver failed closed first.** `git ls-remote --tags --refs` answers
152 tags, all `release-`. Eleven of them are spellings the three-component
scheme cannot order: `0.4.0a`, and ten package revisions such as `0.10.0+1`.
All eleven lead with 0, and no 1.x tag is unorderable. So the adapter now
declares `release_floor_major=1`, the lowest floor that releases them. The
release listing still selects 1.37.0. `check-source-route` replays the 152 tag
names: no floor fails closed, a floor of 1 selects `release-1.37.0` with exactly
11 released, and a floor of 2 releases nothing. 32 of 32.

⛔ **A tag carries no `configure`, and a runner has no `autopoint`.** The tag
is built with `autoreconf -i`, and its `configure.ac` calls
`AM_GNU_GETTEXT_VERSION`, so `autoreconf` runs `autopoint`. Measured on
2026-10-21:

| host | autoconf | automake | libtool | m4, pkg-config | gettext or autopoint |
| --- | --- | --- | --- | --- | --- |
| `ubuntu-24.04` image 20261012.334.1, its README read directly | 2.71-3 | 1.16.5 | 2.4.7 | yes | not listed |
| the session host | 2.71-3 | 1.16.5 | 2.4.7 | yes | no `autopoint`, no gettext m4 |

⭐ **So the route fetches `autopoint` rootless**, the way Transmission's build
fetches libcurl's headers. It runs `apt-get update` and `apt-get download
autopoint` with the index and cache in the workdir, then `dpkg-deb -x`.
`autoreconf` gets `AUTOPOINT=<that script>` and
`gettext_datadir=<its share/gettext>`. `autopoint` 0.21-14ubuntu2 depends only
on `xz-utils`, which the runner lists. `build-deps.txt` records the package,
version and digest.

⚠ **Why this and not the alternatives:**

- Root-installing `gettext` is what `ACQ-06` forbids.
- Generating `configure` from the release tarball would merge the two routes.
- Stubbing the gettext macros would build something the vendor does not
  document.

A host where the private index cannot be read gets a refusal naming it.

⭐ **The `package` slug is `elf-binary-git`.** The two routes install one form,
a self-contained executable from one configure line. They differ in delivery:
the release tarball carries a `configure` the vendor generated, and the tag's
build system is generated on the host. ⚠ aria2-next and Transmission call
both of their compiled routes `elf-binary`, so this is a decision rather than
the house rule, and it is listed under *Decided unattended*.

⭐ **Driven on the session host on 2026-10-21.** Each lane ran as `ubuntu`,
with its own `HOME`, prefix, claim and `RUNNER_TEMP`, through `bit-check
install-step`:

| step | release | source |
| --- | --- | --- |
| resolve | `resolve-release`: `aria2-1.37.0.tar.bz2`, `digest_source=unpublished`, 1 second | `resolve-source`: `release-1.37.0`, live refs, under a second |
| `install-step` | exit 0 in 255 seconds, `acquired=yes`, `package=elf-binary` | exit 0 in 241 seconds, `acquired=yes`, `package=elf-binary-git`, commit `02f2d0d8...` |
| the build's answer | `aria2 version 1.37.0`; BitTorrent, GZip, HTTPS, Message Digest; zlib/1.3, OpenSSL/3.0.0m; gcc 13.3.0 | the same features, libraries and compiler |
| the executable | 85039600 bytes, sha256 `aa2cd83b...` | 85039576 bytes, sha256 `d1b46956...` |
| capture, 45 s, 3 sessions, in `bit-check contained` | exit 0: 3 announces, 6 peer streams, 28 segments, verified by `sha256sum` | the same |
| peer ID | `A2-1-37-0-` and a raw ten-byte tail, new per session | the same |
| user agent | `aria2/1.37.0` | `aria2/1.37.0` |
| `stop` | exit 0, stopped after 40 s with 5 s margin | the same |

⚠ The capture ran as root, inside the namespace. `ubuntu` has no
passwordless `sudo` on this host, and `contained` asks for it, whereas a
runner's user has it. The install and the prefix were `ubuntu`'s.

⭐ The source capture's first peer ID was `A2-1-37-0-` and hex
`f23eaa6bd6e407869e2c`. Its hex occurs **zero** times in the transcript file,
so the old guard would have refused this capture too. The new guard passed it,
and the connector reported the same three peer IDs. ⚠ Each announce's `key=`
equals the last eight bytes of that session's peer ID tail. That is an
observation for `SCHEMA-04`, not a field.

⛔ **What only a runner can prove.** Dispatch `capture-client` with `adapters:
["aria2"]` and `routes: ["release","source"]`. The workflow needs no change:
both lanes read `compiles`, so each gets the build bounds. Then check:

1. whether the source lane's private `apt-get update` reaches the archive from
   a runner, and whether `autoreconf -i` and the build finish inside the
   1500-second bound;
2. whether both captures pass the new guard on a runner;
3. whether `assemble-bundle` makes two `publishable` records with
   `build_equivalent` across them. It was not run by hand here.

### ⛔ aria2 cannot corroborate aria2: `E-COR-20` and the capture guard. 2026-10-21

⚠ **Until this section nothing enforced the Problem's last clause.** A record's
`capture.connectors[]` carries an `id` and a `version`, and no rule compared
either with the target. `capture-client` read the connector's described `id`
and never compared it with the adapter's `target`.

⭐ **Two homes now enforce it, with one identity test.** A connector and a
target are one implementation when either identifier contains the other:

| layer | what it refuses | where |
| --- | --- | --- |
| the record | `E-COR-20`: any declared connector whose `id` is the target, or the target's `engine`, by that test, at `capture.connectors[i]` | `crates/bit-ids/src/validate.rs`, `same_implementation` |
| the capture | exit 1 when the connector's `--describe` `id` or `runtime` is the adapter's `target`, before the connector reads anything | [`../tools/check/selfcorroboration.go`](../tools/check/selfcorroboration.go), called from `captureclient.go` |

⚠ **Why containment and not equality.** The connector the architecture plans is
`aria2c`, which equality and a hyphen-prefix test both let through. Containment
refuses `aria2` beside `aria2`, `aria2c`, `aria2c-rpc`, `libaria2` and
`aria2-next`, in both directions. It also refuses `qbittorrent` beside
`qbittorrent-enhanced`, a fork that shares code, which is the pessimistic side.
⛔ Every declared connector is tested, the observer included, rather than only
those that observe a field. A declared connector is what `E-CAP-01` counts.

⚠ **The identifier is the identity because it is all a record carries.** A
connector built on aria2 under an unrelated name would pass the record's rule.
The capture guard narrows that by testing `runtime=` too, and a connector names
its implementation, as `cpython-stdlib` does. Both choices are under *Decided
unattended*.

⭐ **Measured on 2026-10-21:**

- `cargo test -p bit-ids --test profile_schema`: 28 passed, 0 failed. Six tests
  are new. `aria2` beside `aria2` is refused with `E-COR-20` and nothing else.
  `cpython-stdlib` beside `aria2` and `aria2-next` is accepted. Seven spellings
  are refused, and so are an engine connector and an observer that is the
  target. No catalogue target is refused beside `bit-ids-probe` or
  `cpython-stdlib`. `E-COR-20` has its row in the planted table.
- ⛔ With the call to `check_self_corroboration` removed, 5 tests fail.
- `capture-client --selftest`: 22 cases, all held, 7 of them new.
  `check-bitcheck` removes the guard in its scratch tree, builds that copy, and
  the selftest exits 1 on *aria2 is refused as the connector of aria2*.
- `check-capture-client`: 138 of 138. Three new cases drive the real runner with
  a stub connector named `stub-client`, `stub-client-next` and `stub`, beside
  the stub target `stub-client`. With the call in `captureclient.go` disabled,
  135 of 138, 3 failed.
- All 18 records on `data` at `5a3eb7a` still read as valid under the new
  rule, aria2 1.37.0's two among them. Their second connector is
  `cpython-stdlib`.

⚠ **Whether the Prove is met.** Its first half, *tests reject aria2
self-corroboration*, is met by this section. Its second half is met by
`capture-client` run 39590380931: two routes, `release` and `source`, both
`publishable`, `build_equivalent` across them. Those records rest on the Rust
observer and `cpython-stdlib`'s decoding of the captured bytes. ⛔ Two things
stop this section from closing the entry:

- The *Approach* names a separate packet oracle. None exists; the second reading
  is of the bytes the observer recorded, which `OBS-07` states the cost of.
  `CLIENT-14` closed on the same kind of reading.
- The title and *Approach* also name aria2 **as a connector** for other
  targets. Nothing in the tree drives aria2 that way yet. `E-COR-20` is what
  will keep it honest when something does.

So the Status is left for the operator.

### ⭐ Published: aria2 1.37.0 on two rootless routes. 2026-10-21

capture-client run 39590380931 on `8474d52` captured aria2 1.37.0 on Linux's
`release` and `source` routes, each in `bit-check contained`. Both lanes
passed the install barrier and the decoded peer-ID guard. assemble-bundle run
39594403593 found both records `publishable` and `classify_across` found them
`build_equivalent`, and 18 of 18 records could enter a published view.
publish-data run 39594611777 appended them to `data` as `5a3eb7a`:
`profiles/v1/aria2/1.37.0/linux/x86-64/elf-binary/cap-release.json` and
`.../elf-binary-git/cap-source.json`.

⚠ An earlier assembly of the same capture, run 39593010968, was not published.
It began 20 seconds before Deluge's records reached `data`, so its view held
16 records without them. It was assembled again rather than published over
the newer branch.

⚠ **Still OPEN, decided unattended.** The Prove is met as written: the tests
refuse aria2 corroborating aria2 (`E-COR-20`), and a live two-route capture is
published, supported by the Rust observer plus `cpython-stdlib`'s decoding of
the same bytes. But the title and the Approach also name aria2 as a
*connector* for other targets, and nothing drives aria2 that way yet. The
operator decides whether that half belongs to this entry. It is recorded in
[`PROGRESS.md`](PROGRESS.md) beside `CLIENT-01` and `CLIENT-06`.

## CLIENT-06: Transmission capture adapter

Source: operator scope and bit-cli source-profile generator study
Priority: P1 | Effort: L | Status: OPEN

Problem: Existing source-derived formulas are useful hypotheses but violate
this corpus's live-only evidence rule.

Approach: Automate transmission-remote, compare official and package routes,
and test all formula expectations solely against emitted traffic.

Prove: a two-route Linux and Windows run derives every published field from
raw observations and labels source expectations only as non-authoritative notes.

### The adapter, written 2026-09-08

⭐ `transmission.sh` drives the
daemon through `transmission-remote`, which is the supported unattended path
since `transmission-cli` was removed upstream. Its whole settings file is
written, and the torrent is added by the tool the product ships for that.

⚠ **The package route stops and disables the service the distribution starts on
install.** A capture must own the only running copy, or it would measure
whichever one the torrent reached.

⚠ **`--foreground`, so the backgrounded process is the daemon itself.** A daemon
that forked would leave `stop` holding the identifier of a shell that had already
returned, and the build would outlive the capture.

⛔ **Nothing published here comes from a formula.** This entry exists because
source-derived peer-ID expectations are hypotheses, and the adapter contributes a
running process rather than one.

### ⭐ The first client capture, measured 2026-09-08

**Client capture run 1, job `transmission`, on `ubuntu-24.04`.** The first time
this project has observed a `BitTorrent` identity from a running build.

| what | value |
| --- | --- |
| installed | `transmission-daemon` `4.0.5`, package route, `Linux 6.17.0-1022-azure x86_64` |
| version asked of | the installed executable, by the adapter, before the route was cut |
| containment | `no route off this host (read /proc/net/route)`, asserted after the deletion and again by `capture-client` |
| announces | 2 |
| peer ID on the wire | `2d5452343035302d756435383564356171646f73` |
| transcript | 9 segments, 3 artifacts, verified by `sha256sum -c` |
| restore | the inverted egress guard refused, which is the route proved back |
| evidence | uploaded, 18 files |

⛔ **The peer ID is the measurement and the prefix is not a lookup.** What this
establishes is that a build reporting `4.0.5` put those exact bytes on the wire,
which is section 4's peer-prefix rule inverted rather than waived: a prefix this
project measured, against the run that measured it.

⛔ **It does not close this entry, and four things are why.** One route ran
rather than two, so `ACQ-03` has nothing to compare; one connector observed it,
so `SCHEMA-03`'s overlap is `not_corroborated` by construction; nothing wrote a
`Profile` into the store, so there is no record to cite; and the Prove names
Windows, which is untouched.

### ⛔ What the same run cost, and the defect it bought

**Two of its three jobs sat in `Install the client` for over half an hour and
reported nothing at all.** The job timeout killed the runner and took the log
with it, so the cause is not recoverable from that run.

⭐ **The defect is in the contract rather than in either adapter: no adapter call
had a time limit.** A hung install and a slow one are indistinguishable until
something bounds them, and `shell.md` section 9 already said so about any tool a
script shells out to. Every adapter call is bounded now, in
[`install-client`](../tools/check/installclient.go) and
`capture-client` rather than only in the
adapters, because a convention in each adapter is a bound the next adapter
forgets. ⚠ A refusal now names the timeout and prints the route's own log, since
the workdir is uploaded only when the capture that follows succeeds.

Three narrower causes were fixed with it, each of which would produce exactly
that silence:

- stdin is `/dev/null` on every adapter call, so a product that asks a question
  gets end-of-file instead of a wait;
- `NEEDRESTART_MODE=a` is set for apt, because Ubuntu 24.04 ships `needrestart`
  and it opens an interactive dialog that `DEBIAN_FRONTEND` does not suppress;
- ⛔ `qbittorrent-nox --version` carries `--confirm-legal-notice`, which the
  `start` call had and the `version` call did not. That is a control on one of
  two paths into the same product, which is the shape this project's reviews
  call the most recurring hole there is.

⚠ **Which of the four it actually was is not established**, and saying so is the
honest form: the log is gone. The next dispatch answers it, because a bounded
call reports its own failure with the product's log beside it.

### ⛔ Client capture run 2, and the two defects it found in the fix

**It answered for one of the two and not the other**, which is worth stating in
that order because the second is the more interesting failure.

⭐ **`qbittorrent` now fails in fifty seconds with a readable verdict**:
`installed through the package route but would not report a version`. The
install itself succeeded in fourteen seconds; it is the `version` call that
refuses. ⛔ **And the run could not say WHICH refusal fired, because of two
defects in code written the same day:**

1. **The adapters read `--version` through a pipe.** `LINE=$(... | head -1) ||
   cannot` reads `head`'s status, and `head` exits 0 over anything, so the guard
   was dead code and a build that refused to answer reached the parse instead of
   the refusal. ⚠ This repository's oldest stated rule, broken in every adapter
   at once, on the day they were written.
2. **The caller discarded the adapter's stderr and then reported its absence.**
   `install-client` ran `version` under `2>/dev/null`, so an adapter that said
   exactly which of its three refusals fired arrived as a bare exit code.

Both are fixed and both have a case: the exit code is read unpiped, the
adapter's message is printed with the refusal, and the harness asserts a
planted message reaches the log.

⛔ **`aria2` is not answered, and the bound did not fire.** Its install step ran
for thirty-five minutes under a 900-second bound and the job's own timeout killed
the runner, taking the log with it for the second time. ⚠ **Two possibilities
remain open and the run cannot separate them**: `timeout` sends `TERM` and then
waits, so a child that blocks or ignores it is never killed; or the bound fired
and something the adapter left running held the step open. Both are now
addressed - `-k` forces a `KILL`, the bound sits at 420 seconds under a
25-minute job so it reports with margin - and neither is confirmed.

⭐ **The certain fix is the one that does not depend on the diagnosis.** The
workflow uploads the install logs on `always()` now, so the next hang leaves its
own evidence whatever kills the runner. ⚠ Two dispatches lost their diagnosis to
a step whose only upload came after a capture that never happened.

### ⭐ Client capture run 3: both fixes worked, and the product answered

**The log carried the product's own words for the first time:**

```text
-- the adapter said
qbittorrent: qbittorrent-nox --version exited 1: Bad command line:
  --confirm-legal-notice is an unknown command line parameter.
install-client: qbittorrent installed through the package route but would not
  report a version (adapter exit 2)
```

⛔ **So `--confirm-legal-notice` was never a control. It is an argument this
build refuses**, and it was on BOTH paths into the product: adding it to the
version call to match `start` looked like closing a one-gated door, and what it
actually did was spread a refused argument to a second place. ⭐ The acceptance
is written into the profile as `[LegalNotice] Accepted=true` now, which is the
next assumption and is not measured either.

⚠ **That is what the unmeasured-assumption block in each adapter is for**, and
this is the first entry in one to be refuted rather than confirmed. The block
said the flag was what stops the prompt; the product says the flag does not
exist.

### ⛔ And aria2 moved its hang rather than losing it

**Its install succeeded in run 3 and the step AFTER it hung instead.** The
install logs uploaded, so nothing was lost this time.

⚠ **The most likely reading is the letter in `NEEDRESTART_MODE`.** Run 1 had no
such variable, so `needrestart` was interactive and the install itself hung; run
3 set `a`, which stops the dialog by RESTARTING the services it lists - on a
runner that means restarting daemons the job is standing on. `l` is `list`: it
reports and touches nothing, and both adapters use it now.

⛔ **That is a reading and not a measurement**, and it is the third guess in this
area. What is measured is the shape: the hang moved from the install step to the
one after it when that variable changed, and nothing else about the adapter did.

### ⭐ Client capture run 4: a second client, and a second capture of the first

**Two of the three jobs captured a build, and the third isolated its own
failure.**

| target | build | announces | peer ID on the wire | transcript |
| --- | --- | ---: | --- | --- |
| qbittorrent | `4.6.3` | 1 | peer ID `2d7142343633302d596939654d4d7e38664f7866` | 5 segments, 3 artifacts |
| transmission | `4.0.5` | 2 | peer ID `2d5452343035302d756435383564356171646f73` | 9 segments, 3 artifacts |

⭐ **`CLIENT-01` has its first measurement.** The profile the adapter writes is
what a fresh `qbittorrent-nox` reads, the torrent is a positional argument, and
the build announced to the lab under containment; the `[LegalNotice]` key in the
profile is what the refused flag was supposed to do, and it works.

⛔ **Neither entry closes.** Every one of the four gaps the first capture had is
still open for both: one route, one connector, no `Profile` in the store, and no
Windows.

⚠ **Transmission's second capture reports the same build and a different peer
ID**, which is what a per-session identity looks like and is exactly what
`SCHEMA-04`'s sampling model exists for. Two samples is not a lifetime
measurement and nothing here claims one.

### ⛔ What is left of the aria2 hang, isolated

**`NEEDRESTART_MODE=l` did not fix it.** Its install succeeded again and the step
after it hung again, so the reading that the letter was the cause is refuted for
`l` and unsettled for `a`.

⭐ **What run 4 does establish is the boundary.** The hang is specific to this one
package install, it survives the step that caused it, and it is not the install
command's own duration: `install-client` returned, the record was written, and
the next step is where the job stops. ⚠ Two adapters doing the same `apt-get`
on the same image do not hang, so it is the `aria2` package rather than the
route.

### ⛔ Run 5 refutes that boundary: it is not the package and not apt

**The release route runs no package operation at all** - it fetches a tarball
with `curl`, unpacks it, configures, compiles and installs - and *Upload the
install logs* hung identically.

| | runs 3 and 4 | run 5 |
| --- | --- | --- |
| route | `package` | `release` |
| apt operations | `update` and `install`, installing nothing | **none** |
| install step | success in 6s | success in 144s |
| where the job stopped | *Upload the install logs* | *Upload the install logs* |

⛔ **So the second guess is refuted the way the first one was.** `NEEDRESTART_MODE`
was refuted by install logs saying `0 newly installed`; "the aria2 package rather
than the route" is refuted by a route that touches no package index. Two readings
of this hang have now been named and both were wrong.

⭐ **And run 5 puts a clock on what was previously only "complete and
downloadable".** The artifact was written at 03:17:23Z, five seconds into a step
that started at 03:17:18Z, and the step had still not returned at 03:30:43Z -
**thirteen minutes** later. The upload's work finishes almost immediately; what
holds the runner open afterwards is not the upload.

⛔ **What remains common across all five is the aria2 job and nothing else that
has been isolated.** Naming a cause here would be the third guess, and `CI-08` is
the entry for a runner default nobody swept.

### ⛔ Run 6: two lanes, one target, and the bound that did not fire

**Dispatched as `["aria2"] × ["package","release"]`** - the two-route pair this
entry needs. Both lanes claimed a host, both installed, and both then hung in the
same step.

⭐ **Three things worked and are worth separating from the hang.** The resolve
step **skipped** on the package lane and succeeded in one second on the release
lane, so the matrix condition is right; the release route installed in 144
seconds again, so run 5 was not a one-off; and the named upload paths cut that
lane's artifact from 42.6 megabytes to **43179 bytes**.

⛔ **`timeout-minutes: 5` did not stop either lane.** The package lane's step
began at 03:50:11Z, its artifact was written at 03:50:12Z - **one second in**,
3247 bytes - and the step was still running at 04:07:39Z, seventeen minutes later.
⭐ That also closes the size question: three kilobytes hangs exactly as thirteen
times more does.

⭐ **So the step moved to the end of the job instead**, which needs no diagnosis:
`if: always()` runs after a failed step wherever it sits, so the early position
bought nothing, and from the end a hang costs the job's tail rather than the
capture. `CI-08` carries the measurement and what it does to the mitigation.

### ⭐ Run 7: four lanes, and the hang stops being about the upload

**Dispatched as two adapters over both routes**, which is the first run of the
matrix at its full width. Two lanes finished and two did not, and the split is
the finding.

| lane | outcome |
| --- | --- |
| `transmission` `package` | ⭐ **green: a complete capture in three minutes** |
| `transmission` `release` | the install refuses by design, and every `always()` step still ran |
| `aria2` `package` | ⛔ stuck in *Install the client* from 04:41:15Z until it was cancelled |
| `aria2` `release` | ⛔ stuck in *Install the client* from 04:41:17Z until it was cancelled |

⛔ **The hang is not in `actions/upload-artifact`.** Two transmission lanes ran
*Upload the install logs* in **one second** each - one after a successful capture
and one after a failed install - from the same step, on the same image, in the
same run as two aria2 lanes that hung.

⛔ **And it is not "the upload step" either.** With that step moved to the end,
the hang appeared in the step it used to follow: on run 6 the aria2 package
install took **six seconds** and on run 7 the same install ran **ten minutes**
without completing. ⚠ So what hangs is not a particular action but whatever step
sits next to the aria2 install, which is a boundary no previous run could draw.

⭐ **The move is measured to buy what it was argued to buy, and the transmission
release lane is the proof.** Its install failed at 04:41:16Z, every step between
there and the end was skipped, and *Upload the install logs* still ran and
succeeded at 04:41:16Z. ⛔ **The early position bought nothing**: the reason it
was placed before the route cut was that a job which never reached the capture
would otherwise upload nothing, and an `always()` step at the end uploads it.

⚠ **What the move costs is aria2's own diagnosis, and that is on the record
rather than traded away quietly.** With the install step itself hanging, run 7
produced no aria2 artifact at all, where runs 3 to 6 produced one. A conditional
early step for one adapter was the rejected alternative:
[`../docs/capture-host.md`](../docs/capture-host.md) argues against a step that
sometimes runs, in the one file where every reader has to see what happened
without reading an expression. ⭐ The four aria2 install logs already collected
say what that route does; what run 7 bought instead is where the hang actually
sits.

### ⛔ The two-route attempt moves to `CLIENT-14`, by operator direction

**2026-09-09.** Ten dispatches, no aria2 capture, four bounds that did not fire
and five readings refuted. The operator's direction is to change the target
rather than to keep diagnosing: `CLIENT-14` acquires `AnInsomniacy/aria2-next`
from that project's own releases and drives it over RPC.

⚠ **This entry stays open and keeps its evidence.** The hang is a measured
property of this repository's capture path on a hosted runner, not a fact about
one product, and nothing here has established that a different target avoids it.
⛔ What moves is which entry the first two-route capture is attempted through.

### ⛔ Run 10, and the fourth bound that did not fire

**Dispatched 2026-09-09 as `["aria2"] × ["package","release"]` on `3b5793d`**,
the first run where the step's own command was wrapped in `timeout -k 30 540`.
Both lanes entered *Install the client* at 10:57:50Z and 10:57:58Z and neither
had returned twelve minutes later.

⛔ **Four bounds at four levels have now been measured not to fire on this step,
and no such job has ever produced a log or an artifact.** `CI-08` carries the
table and what the combination separates. ⚠ What it changes for this entry is
that the next dispatch bisects with step names rather than adding a fifth bound.

### ⭐ Transmission's fourth capture, and aria2's fifth hang

**Client capture run 9, 2026-09-09, `["aria2","transmission"] × ["package"]` on
`42bd206`.** Transmission captured green in three minutes with a 121-second
install; aria2's *Install the client* began at 10:13:54Z and had not returned
fifteen minutes later.

⛔ **The step's own watchdog deadline was 480 seconds and it passed by seven
minutes.** That loop runs in the step's shell, so a shell that was looping would
have fired it - which makes this a fact about the step rather than about the
install, and the third bound measured not to fire.

⚠ **Transmission is the control that keeps it attributable**: same run, same
image, same step and the same `apt-get`. ⚠ Its install is itself far slower than
the fourteen seconds earlier runs recorded, so these hosts are slow today - and
slow is what the aria2 lane is not, because `install-client` bounds the install
call at 420 seconds and would have refused.

### ⛔ Run 8 refuted the reading it was dispatched to test

**Dispatched 2026-09-09 as `["aria2"] × ["package","release"]` on `5911c0d`**, the
first run in which the install step's output went to a file rather than to the
step's own pipe.

| lane | *Install the client* |
| --- | --- |
| `aria2` `package` | ⛔ began 09:13:18Z and had not returned at 09:35:17Z - **22 minutes** |
| `aria2` `release` | ⛔ began 09:13:18Z and had not returned at 09:35:17Z |

⛔ **So the step's own output pipe being held is not the mechanism.** Nothing the
route spawned could inherit that pipe on this run, and the step hung exactly as
before.

⭐ **And the same run says where it is NOT.** `install-client` bounds the install
call at 420 seconds with a `-k 20` kill, so the latest moment that call could
have ended is 440 seconds in. Twenty-two minutes is three times that. ⛔ **Whatever
is slow or stopped is therefore not inside the bounded install call**, which is a
narrower statement than any of the previous eight runs could make and it is a
measurement rather than a reading.

⛔ **And it left nothing behind, which is the argument for the next design.** The
run ended `cancelled` at 09:42:49Z, roughly thirty minutes in; both jobs' logs
answer 404 through rule 8's route and the run carries **zero artifacts**. ⭐ So a
step that does not end is a job that teaches nothing, and the only fix is a step
that ends.

⚠ **And this dispatch carries a confound, which is on the record rather than
argued away.** It changed two things at once: the redirection, and a holder
report added to the same step. The report walks every process's descriptors, so
it is itself a candidate for a slow step - it runs only after `install-client`
returns, but that is precisely the branch a 22-minute step cannot distinguish.
⛔ Both are bounded now: the diagnostic is wrapped in `timeout 60`, and the step
watches its own install rather than waiting on it.

### ⛔ What the eighth dispatch is for, and what it changes first

**Prepared 2026-09-09.** Nothing about aria2's adapter or its routes changed.
What changed is the workflow step they run in: the route's output goes to a file
rather than to the step's own pipe, because a runner ends a step when the command
has exited **and** that pipe has reached end of file. `CI-08` carries the
instrument that measures it and the plant that proves the redirection is what
does the work.

⚠ **This is not a fourth cause and it is not offered as one.** What made it worth
doing is a comparison of two runs rather than a theory about a mechanism: runs 6
and 7 ran the same install from commits whose only functional difference is where
a later step sits, and it took six seconds in one and had not returned after
sixteen minutes in the other.

⭐ **Both outcomes of the next dispatch are informative.** A lane that gets past
*Install the client* says the class is what the hang was; a lane that hangs
anyway refutes it with something that separates rather than with another guess.
⚠ Either way the job now leaves `holders.log` behind - every process still
holding the route's log, with the process table beside it - which is the first
positive evidence about what an aria2 install leaves running on a runner.

### ⭐ Transmission's third capture, and a third peer ID

| what | value |
| --- | --- |
| build | `transmission-daemon` `4.0.5`, package route, `acquired=yes` |
| executable | `/usr/bin/transmission-daemon`, `sha256:4444bc9e…` |
| announces | 2 |
| peer ID on the wire | `2d5452343035302d336b37613967686a37383172`, which is `-TR4050-3k7a9ghj781r` |
| transcript | 9 segments, 3 artifacts, `sha256sum -c` reports `OK` for all three |
| peer surface | one stream, dialled at `127.0.0.1:51413` |

⭐ **Three captures of one build, three different peer IDs.** Runs 1 and 4 each
reported the peer ID `2d5452343035302d756435383564356171646f73`, and this one
differs after the `-TR4050-` prefix. That is what a per-session identity looks like and it is
exactly what `SCHEMA-04`'s sampling model exists for; three samples is still not a
lifetime measurement and nothing here claims one.

⚠ **The install record now travels inside the evidence bundle**, so a reader who
downloads one artifact has both the measurement and the verdict about what
acquired it. It says `preexisting_version=` empty and `acquired=yes`, which is a
route that genuinely installed - unlike the aria2 package route, whose whole
`acquired` machinery exists because it does not.

⚠ Its install log uploaded on every run since the `always()` step landed, so the
next session reads `update.log` and `install.log` from
`install-aria2-<run>-1` rather than dispatching to find out. ⚠ That artifact name
carries the route now - `install-aria2-package-<run>-1` - because two matrix legs
share an adapter.

⛔ **And what that step uploaded was wrong on the release lane, measured on run
5.** The path was the install workdir, which for a source-build route is where
the tarball was unpacked and compiled: **1867 entries, 42.6 megabytes**, for a
step called *Upload the install logs*. ⚠ It looked right for four dispatches
because the package route's workdir happens to hold two log files. It also
omitted `install-<route>.txt`, the record `ACQ-03` compares, which went up only
inside the evidence bundle - so a job that hung before the capture uploaded its
logs and not its verdict. The paths are named now.

### ⭐ Transmission's fourth capture, read back outside the run that wrote it

**Client capture run 9, 2026-09-09.** Downloaded through rule 8's route,
unauthenticated, and verified with `sha256sum -c`: `OK` for all three evidence
files.

| what | value |
| --- | --- |
| build | `transmission-daemon` `4.0.5`, package route, `stock_client=true` |
| announces | 2 |
| peer ID on the wire | peer ID `2d5452343035302d6e61387866306133756c7578` |
| transcript | 9 segments, 3 artifacts |
| peer surface | one stream, dialled at `127.0.0.1:51413` |
| containment | `egress=closed`, read from `/proc/net/route` |

⛔ **Four captures of this build have now reported THREE distinct peer IDs**, and
the two counts being different is the finding rather than a rounding of it: runs
1 and 4 reported the same identity and runs 7 and 9 each reported another. ⚠ A
per-session identity that repeats across two sessions is not what a naive reading
of "per-session" predicts, and four samples is not a lifetime measurement.

### ⚠ The release route declares its artifact and still refuses, 2026-09-09

**Both halves are true at once and the split is the honest one.** This adapter
now declares which file a release route would fetch:

```text
release_repo=transmission/transmission
release_tag_prefix=-
release_min_components=3
release_max_components=3
release_asset=transmission-{version}.tar.xz
```

and the route still refuses, because nothing here knows how to build it. ⛔ This
project can say exactly which artifact a route would take and cannot say how to
compile it; declaring the first is what makes the second a named gap rather than
a route that is simply unreachable.

⚠ **Measured against the live listing on 2026-09-09**: release 4.1.3 carries
eleven assets - four `.msi` installers, four `-pdb.7z` symbol archives, a macOS
`.dmg`, a `-dsym.zip` and exactly one `transmission-4.1.3.tar.xz`. ⛔ Two of them
are capitalised `Transmission-`, which is why the pattern is matched
case-sensitively and a case in the Rust suite pins it.

⚠ **And the versions still would not meet.** Ubuntu 24.04 ships 4.0.5 against
upstream 4.1.3, so a `["package","release"]` dispatch of this adapter is two
hosts that acquire two different versions - which `AGENTS.md` rule 5 forbids
resolving by backfilling. The pair to attempt first is `CLIENT-05`'s.

### ⭐ A same-version pair exists for 4.1.3, and it needs no new route. 2026-10-11

⛔ **The package route cannot be one half.** Ubuntu 24.04 ships 4.0.5, so the
pair has to come from somewhere that carries 4.1.3. Four were read, through the
routes `AGENTS.md` rules 8 and 9 name:

| candidate | what it carries for this target | why it is or is not a half |
| --- | --- | --- |
| the `release` asset, `transmission-4.1.3.tar.xz` | the source tarball, digest `ce7d2d8b…02617da` in the listing, matched by `sha256sum` here | ⭐ a half: it builds, see below |
| the repository's `4.1.3` tag | the same source through `git` | ⭐ a half, on the model `aria2-next` already measured: a different resolver (refs, not the release listing) and a different delivery (a clone, not an asset) |
| Homebrew, `transmission-cli` | 4.1.3, with an `x86_64_linux` bottle | ⚠ preinstalled on the runner image but not on `PATH`; a `package`-kind route that needs no root. Not taken first, because it would be a third packaging this project has not modelled |
| Flathub, `com.transmissionbt.Transmission` | 4.1.3 | ⚠ the GTK client, and `flatpak` is not on the runner image |

⭐ **The tarball builds with one dependency the runner image lacks.** Measured on
the session host on 2026-10-11: `cmake -G Ninja` with GTK, Qt, tests and the CLI
off and the daemon and utilities on refuses only for want of libcurl's headers;
with `libcurl4-openssl-dev` added it built in **695 seconds** on four processors
and the daemon answered `transmission-daemon 4.1.3 (838877323f)`. ⚠ The runner
image's own package list names `libssl-dev` and `curl` and no libcurl
development package, read from `actions/runner-images` on the same day, so a
build lane installs that one package first. ⚠ 695 seconds here is not an
estimate for a runner: `CLIENT-14` measured one at about 1.35 times slower, and
`capture-client` run 35 one faster. ⭐ The source lane's install bound rose to
1500 seconds on 2026-10-14, for `aria2-next`'s build, and that bound is chosen by
the route, so it holds this one too. ⛔ **The release tarball's build does not
get it**: it is the release route, whose bound is 420 seconds, and a release
route that compiles is a case the workflow's per-route bound does not yet know.

⭐ **And a capture host finishes an install.** The recorded hang was the route
cut severing the runner agent, measured on 2026-10-11; `CI-08` carries it.

⚠ **Residual, from the door sweep of 2026-10-14: `transmission.sh` writes its
discovery settings and asks the running build nothing back.** `aria2-next` 3.0.2
is what a renamed or new switch does to a list like that, and `CLIENT-14`
carries the read-back that caught it. `qbittorrent.sh`, under `CLIENT-01`, has
the same shape and no channel to ask through, its WebUI being off. ⚠ The same
sweep read `"encryption": 0` in `transmission.sh` as not Transmission's default,
which is a claim from memory until the build's own `session-get` answers it.
Acceptance: each adapter's `start` reads what took effect - Transmission's
`session-get` over its RPC - and refuses a discovery surface that is not off,
refuted by a plant that leaves one on.

### ⭐ Both 4.1.3 routes build without root, driven as an unprivileged user. 2026-10-14

`transmission.sh` now builds
both halves of the pair the section above found:

- **release** fetches `transmission-4.1.3.tar.xz` through `install-rootless.sh`,
  identified by the listing's digest `ce7d2d8b…02617da`, unpacks it and builds.
- **source** clones tag `4.1.3` with its sixteen submodules, shallow, names the
  commit `838877323f…`, the build hash the daemon reports, and builds.

Both use one build: CMake and Ninja with the daemon and `transmission-remote`
on, and the GTK and Qt clients, the CLI and the tests off. Both executables go
to the rootless prefix, and the configure line is written to
`build-configure.txt`.

⭐ **libcurl's headers without root.** The runner image carries libcurl's
runtime and not its development package, and `ACQ-06` forbids an install that
escalates. `apt-get` runs with its index and cache in the workdir, as an
unprivileged user, and fetches `libcurl4-openssl-dev` through the
distribution's signed index. `dpkg-deb -x` unpacks it there. The package's
`libcurl.so` must name a runtime file this host already has, or the route
refuses. `build-deps.txt` records the package, its version, its digest and
that runtime.

⛔ **The release route's bound was the defect the section above named.** The
workflow bounded installs by the route's name, 1500 seconds for `source` and
420 for everything else, and this release route compiles. Each adapter now
declares `compiles` in `describe`, and a new step, *Read whether this route
compiles*, sets the bounds from it before the install. `transmission` declares
`release,source`, `aria2-next` `source`, and `aria2` `release`, whose release
route has always compiled. `check-workflow` holds the step ahead of the
install, with a probe that swaps them.

Driven on the session host as uid 1000, through the adapter itself:

| route | install | answers | from |
| --- | --- | --- | --- |
| release | 1013 s | `4.1.3` | `<prefix>/bin/transmission-daemon` |
| source | 1078 s | `4.1.3` | `<prefix>/bin/transmission-daemon`, commit `838877323f…` |

Both linked the host's `libcurl.so.4.8.0` through headers from
`libcurl4-openssl-dev` `8.5.0-2ubuntu10.15`. ⚠ The source route's first drive
failed at the clone because this host's CA bundle sits under `/root`. That is
the session host, not the route, and the drive was repeated with a readable
copy.

⛔ **And a defect the new harness found on its first run.** The install loop
placed the daemon before checking that `transmission-remote` had been built,
so a refused build left a daemon in the prefix. Both are checked now before
either is installed.
[`check-transmission.sh`](../scripts/capture/check-transmission.sh) drives the
real adapter with `apt-get`, `cmake` and `git` stubbed and a real `.deb` and
`.tar.xz`, 12 of 12. Nine plants were each refused:

| plant | refused by |
| --- | --- |
| the runtime the headers name is not checked | headers for a libcurl this host does not run |
| a package without `curl.h` is accepted | a package with no `curl.h` |
| two downloaded packages are accepted | two packages |
| a missing executable is not checked | a build with no `transmission-remote` |
| an archive with two trees is accepted | two source trees |
| the clone drops `--recurse-submodules` | the clone carries the tag and its submodules |
| the release route names no packaging | every release case |
| `compiles` omits the release route | the declaration case |
| the prefix lookup is skipped | the version case |

⚠ **Still open here:** a dispatch of `["release","source"]` for this adapter.
On a runner the builds are probably slower than 1013 and 1078 seconds; the
1500-second bound is what that dispatch tests. The discovery read-back residual
above is unchanged.

### ⛔ Run 38600252161: the source route fails closed on the repository's history. 2026-10-14

The first dispatch on `b7486b3` stopped the source lane in *Resolve the source
tag*, before any install. Read from its install artifact's `resolution.json`:
132 refs, and 112 of them `unorderable` under the declared three-component
scheme. 99 are two-component tags such as `0.71` and `2.94`, and 13 are old
betas such as `1.80b5`. An unorderable candidate blocks the resolution, and
`git_refs` says why that is correct: refs carry no date, so nothing rules out
that the unreadable tag is the newest. The release listing never met this.
Its 51 old candidates carry a publication date before the winner's, and
`predates_selection` releases them.

⚠ Widening the scheme to two components would not be enough. `3.00` and
`1.00` are refused for their leading zero, and `1.80b5` reads as neither a
number nor a prerelease marker.

⛔ **The two ways through are both changes to the project's one version
ordering, so they are the operator's to choose:**

1. Give refs a date. A treeless bare clone fetches the tags and commits with
   no file contents, and `git for-each-ref` reads each tag's creation date.
   ⚠ That date is written by whoever made the tag, which is weaker than a
   release listing's publication time, set by the server.
2. Let a scheme declare a floor below which tags are not this scheme's, for
   example `4`, with the record saying so.

⭐ **Settled by the operator on 2026-10-15: the floor.** Measured by an
explorer the same day: `git ls-remote` returns 132 refs in 0.7 seconds, and 131
of the tags are lightweight, so git holds no date of the tagging for them at
all. Only the commit's committer date, which the committer sets, is there. 71
of the 112 unorderable tags are ancestors of 4.1.3's commit, and 41 sit on
maintenance branches. The floor releases all 112 from the refs bytes already
recorded, with nothing fetched and nothing author-supplied. A two-component
`5.0` still blocks, and so does a tag with no leading integer.

⭐ **Built on 2026-10-15, as `bit-ids/resolution/2`.**
`VersionScheme::floor_major` is the declaration, `release_floor_major` the
describe key both resolvers now require (`-` for none), and
`below_scheme_floor` the verdict. ⛔ It releases only against a winner whose
major is at or above the floor, and `E-RES-09` and `E-RES-10` re-derive every
release from the recorded scheme rather than trusting the verdict. Driven
against the live refs that day, 132 lines: no floor fails closed with 112
`unorderable`; a floor of 4 selects 4.1.3 with exactly 112
`below_scheme_floor`; a floor of 5 releases nothing; a floor of 3 releases 111
and fails closed on `3.00`, which sits at the floor rather than below it.
⚠ That last one first came out as exit 2, an invalid document, because
`E-RES-10` refused a release beside no selection. A blocked decision that
released some history is a valid record with both reasons in it, so the rule
now asks only about a selection that exists. `check-source-route` replays the
132 tag names with computed object names, 24 of 24, and seven mutants of the
resolver were each killed by a test: `<` to `<=`, a release regardless of the
winner, a padded leading component accepted, each code removed, the verdict
missing from `ALL`, and the floor never marking.

⚠ **The review found two things, both fixed before the push.** The Go port of
`check-staleness` still called `resolve-stable` without the floor argument, so
it reported *cannot run*. The gate printed that as a second `SKIP` and exited
0, and the failure-only read of its log missed it; `docs/lessons.md` carries
that. And `E-RES-09` accepted a floor release of a tag the scheme can order,
such as a readable `3.0.0`. That cannot hide a newer version, but the record
would not be the resolver's own output. It is refused now, with a test, and
the mutant that drops the condition is killed.

⛔ **And the barrier waited for a lane that could not arrive.** *Arrive at the
install barrier* ran only when the install had run, so a lane that failed
earlier posted nothing, and the release lane waited out the bound. Every lane
arrives now, as `failure` when its install was skipped, which ends the others'
wait at once and names the route. `check-workflow` holds the arrival to
`if: always()`, and its reader was driven on the clean workflow and on one
carrying the old condition.

⭐ **The release route installed on a runner in 562 seconds**, 16:42:14 to
16:51:36, faster than the session host's 1013 and well under the 1500-second
bound. The record names
`installed_binary=/home/runner/.local/share/bit-ids/bin/transmission-daemon`,
4404456 bytes, `reported_version=4.1.3`.

⛔ **And rule 12 refused its capture, rightly.** After 1506 seconds at the
barrier, with `verdict=timeout`, the capture ran in its namespace and the
daemon took the torrent. On exit 4.1.3 writes `"rpc-password":
"{<salted hash>"` into `settings.json`, with authentication off, and
`scan-bundle.sh` refused the bundle naming that file. Nothing was uploaded.
Reproduced on the session host's build. The adapter's `stop` now deletes that
one key with `jq` once the daemon has gone, or the whole file if `jq` is
absent, and every exit path of `stop` runs it. Driven on the real daemon:
the key is gone, the file is valid JSON, and `scan-bundle.sh` passes.
`check-transmission` adds a stub daemon that writes the key on SIGTERM, 14 of
14. Two plants were each refused: not deleting the key, and keeping the file
when `jq` is absent.

⭐ **Proved on a runner: `capture-client` run 38612218902 on `038e12c`,
`["release"]` alone, `success`, with its evidence uploaded.** The capture ran
in its namespace with `egress=closed`. The daemon announced 8 times over 2
sessions and answered 2 peer streams. Every peer ID starts `-TR4130-`, hex
`2d5452343133302d`, and each session's tail is its own random one. The second
connector read the same six peer IDs. The bundle passed `scan-bundle.sh` with
the verifier gone from `settings.json`. ⚠ One route is not a record:
`E-ACQ-01` needs two, and the source route waits on the resolution decision
above.

⭐ **Both routes captured: `capture-client` run 38667886527 on `8044d72`,
`["release","source"]`, `success` in 18 minutes.** The source lane resolved
4.1.3 from the refs through the floor and built it, and both lanes uploaded
their evidence. The four artifacts' digests match what GitHub lists.

⛔ **Assembling the pair was refused, and the first reason was the assembler's.**
`assemble-capture` read every segment the build sent on the peer surface as a
handshake. Transmission sends its handshake, then 9.5 seconds later an unchoke,
`0000000101`, on the same connection, so five bytes were refused as a truncated
handshake. The first segment of each connection is now its handshake, and a
later one on that connection is the conversation after it. A first segment that
is not a handshake, and a segment with no connection, are still refused. The
second connector finds a handshake by its protocol string and read the same two.
`check-assemble` adds three cases, 67 of 67, and two mutants were each killed:
every segment read as a handshake again, and a first segment that is not one
skipped.

⛔ **With that fixed, both records are written and both are provisional.**
`classify_across` answers `divergent` on two fields of four, and neither is a
difference between the builds:

1. **`peer_wire/peer_id` is `constant` in each record, with a different value.**
   4.1.3 keeps one peer ID per session, `-TR4130-` and twelve random
   characters, and `client-capture` dials every peer connection in the first
   session. So each record saw one value twice and calls it constant.
   ⚠ **The divergence is what kept a false claim out of the published view**:
   that value is random per session, which `tracker_http/peer_id` shows across
   the restart in the same records.
2. **`tracker_http/peer_id` is patterned in both, and the prefixes differ by one
   byte.** The source capture's two session tails are `i13fphydqmcu` and
   `iigl5aa9agex`, which share their first byte, so its fixed run reads
   `-TR4130-i` against the release capture's `-TR4130-`. This is the residual
   `SCHEMA-04` recorded on 2026-10-14: a coincidence at the edge of a varying
   span lengthens a fixed run and is not caught. Its odds are about one in
   sixty-four per edge with two distinct values.

⭐ **Settled by the operator on 2026-10-15, and built the same day:** the peer
surface is dialled in every session, and a capture runs three sessions.

- **The probe.** `client-capture` dials after the first announce, as before, and
  again after every later announce carrying `event=started`, by the rule
  `assemble-capture` reads sessions with. Each session starts again from
  `127.0.0.1` and redials from `127.0.0.2`, because a new process has seen no
  address. It prints `peer-sessions`.
- **The runner.** `capture-client.sh` records `peer_sessions` and
  `peer_connections` in the attestation, and says so when fewer sessions were
  dialled than started. Its default is three sessions.
- **The assembler.** It places each connection in the session whose opening
  announce came last before the connection's first segment. Both transcripts
  share one clock, the lab's.

Driven on this host against `aria2-next` 3.0.3, release route, 60 seconds, in
the namespace: 3 sessions dialled, 2 connections each, 6 peer streams, and each
session announced its own peer ID. A mutant that dialled in the first session
alone gave `peer-sessions 1` and `peer-connections 2` on the same drive.

⭐ **PUBLISHED, 2026-10-15.** `capture-client` run 38680098567 captured the pair
under three sessions on `eb3db46`. `assemble-bundle` run 38696647239 assembled
it onto the published tree: both records `publishable`, `build_equivalent`,
and 4 fields agreeing. `publish-data` run 38697299991 appended it after a
passing dry run. `data` is at `4c6ccb1`. ⭐ Read back from a fresh clone: 51
files, `sha256sum -c` clean, *valid store: 4 record(s), 4 run(s), 51 object(s);
4 of 4 record(s) may enter a published view*. `schemes.txt` names
`transmission:-:3:3` beside `aria2-next:v:3:3`.
`check-assemble` adds a case of three sessions with two connections each,
69 of 69. The manifest's plan reads three sessions of two, and the mutant
that puts every connection in session 0 reads one of six and is killed.
Reassembling run 38667886527's artifacts, dialled in one session, gives the
same records as before, byte for byte. `check-capture-client` moves its
session counts to three. ⚠ The pair needs capturing again under this.

### ⭐ Transmission's Windows half is built, and not yet run. 2026-10-15

[`../tools/check/adapter_transmission_windows.go`](../tools/check/adapter_transmission_windows.go)
registers `transmission` on Windows, run as `bit-check adapter transmission`.
[`../tools/check/trwindows.go`](../tools/check/trwindows.go) holds what it
decides without Windows in it: the settings document, the two version readings,
the image's layout, msiexec's command line, the service state and the verifier
scrub. `bit-check transmission-windows --selftest` asks each of them on any
host, 51 cases, all held. Five mutants were each refused, and `check-bitcheck`
runs the selftest. This is the Windows half this entry's Prove names.

| route | what it does | `package` |
| --- | --- | --- |
| `release` | `transmission-4.1.3-x64.msi` from GitHub, fetched by `install-rootless --fetch` under the resolution's digest, then laid out by `msiexec /a /qn TARGETDIR=` into `<prefix>\opt\transmission`. No install runs | `msi-payload` |
| `package` | `winget install --id Transmission.Transmission` with `--custom ADDLOCAL=CommonLibs,Daemon,CmdTools` and no `--version`, into `%ProgramFiles%\Transmission`. The installer's own service is then stopped and disabled, and the build's answer must equal the version winget's index named | `msi-installer` |

⭐ **The MSI was read rather than assumed**, on 2026-10-15, with msitools on
the session host, over the asset whose sha256 is
`c8ea492d...21741f8`, the digest
the listing states:

| table | what it holds |
| --- | --- |
| File | `transmission-daemon.exe` and `transmission-remote.exe`, both `4.1.3.0`, beside `transmission-qt.exe`, `-create`, `-edit` and `-show`. `msiextract` lays them out as `PFiles/Transmission/`, with `libcurl.dll`, `libcrypto-3-x64.dll`, `libssl-3-x64.dll` and `zlib.dll` beside them |
| Feature | `CommonLibs` and `QtClient` at level 1. `Daemon`, `CmdTools` and `WebUi` at level 2. The Property table sets no `INSTALLLEVEL`, so the default of 1 applies, and **a default install places neither executable the capture runs** |
| ServiceInstall, ServiceControl | a service `Transmission` in the daemon's own component: own process, automatic start, `NT AUTHORITY\LocalService`. Event 163 starts it on install |
| AdminExecuteSequence | cost, validate, `InstallAdminPackage`, `InstallFiles`, finalize. No custom action, no service, no registry and no launch condition |
| LaunchCondition | the Visual C++ runtime 14.30 or later and the UCRT. The daemon imports `MSVCP140`, `VCRUNTIME140` and `VCRUNTIME140_1`, and the MSI carries none of them |
| Property | ProductName `Transmission 4.1.3 (838877323f) (x64)` |

⭐ **The daemon was read rather than assumed too**, from the `4.1.3` tag's
`daemon/daemon.cc` and `daemon/daemon-win32.cc`. `--version` prints
`transmission-daemon <LONG_VERSION_STRING>` to stderr and exits, so `version`
is the pure read the contract describes, unlike qBittorrent's Windows build.
Without `--foreground`, `spawn` calls `StartServiceCtrlDispatcher`, which fails
from a console, so the flag the Linux half passes is required here. The RPC
`session-close`, which is what `transmission-remote --exit` sends, calls the
same `stop()` as a console control event. The daemon then saves its settings,
which is where the verifier comes back, and closes the session.

⛔ **The two routes deliver one artifact from one host.** winget's x64
installer URL is the release route's own GitHub asset URL, with the same
InstallerSha256. The resolvers differ (winget's index against the release
listing), and so does the delivery (winget against `install-rootless`). That is
what `E-ACQ-07` and `E-ACQ-08` compare. ⚠ The host serving the bytes does not
differ, which makes this pair weaker than qBittorrent's, whose winget route
fetches from SourceForge. It is the same build acquired twice, so `ACQ-03` is
handed the strongest version agreement and the weakest origin independence.

⚠ **What differs from the Linux half, each for a reason in the adapter's
header:** the scrub is Go and takes every member naming a password, and it
deletes the file only when it cannot parse it. The RPC port is a free loopback
port drawn per session, because the installed service holds 9091. The settings
values are JSON-encoded, because a Windows path's backslashes are escapes. The
version reading requires the daemon's own name and a version-shaped field.
`--log-level=error` replaces the deprecated `--log-error`. And `helpers=` names
`transmission-remote.exe` for the firewall.

⛔ **What needs a runner, in the order to read a dispatch:**

1. Whether `msiexec /a` lays the image out on `windows-2025` as measured here,
   `PFiles\Transmission\`. The release lane's `install.log`, `msiexec.log` and
   `extract.txt` show it, and `extract.txt` names the runtime DLLs the build
   will load.
2. Whether winget installs with `--custom ADDLOCAL=...` and places both
   executables. The package lane's `install.log`, `winget.log` and
   `winget-show.txt` show it. `winget-version.txt` holds the index's version
   beside the build's.
3. Whether the installer's service starts and then stops and disables.
   `service.txt` records the state before and after.
4. Whether `transmission-daemon --version` answers on both routes, which
   `version.err` beside each install record carries.
5. Whether `--exit` ends the daemon, so it announces `stopped`.
   `client/stop.txt` says `session-close` or `killed`, and
   `client/config/settings.json` must carry no `rpc-password`.
6. The capture itself: announces from `-TR4130-` with the daemon and
   `transmission-remote.exe` both under the firewall rules.

### ⭐ Transmission's Windows half captured on both routes, first time. 2026-10-15

Capture-client run 38794881216 on `469c7e8` is green on all four lanes:

- **Linux:** release and source.
- **Windows:** release, the MSI laid out with `msiexec /a`, and package, winget
  with the daemon and the command-line tools added.

⚠ Like qBittorrent's, the Windows pair becomes a record only after a
re-capture on the tree that closes `CI-11`'s four assembly gaps.

### ⭐ The Windows pair is published, and agrees with Linux. 2026-10-16

- **Capture:** capture-client run 38838965380 on `05d0d21`.
- **Assembly and publish:** assemble-bundle run 38840840749 and publish-data
  run 38841340795.
- **Records:** at `windows/x86-64/msi-payload` (`msiexec /a`) and
  `msi-installer` (winget with the daemon features), both `publishable` and
  `build_equivalent`.

Published Linux and Windows profiles agree on every observed field (peer-wire
`peer_id` and `reserved`, tracker `peer_id` and `user_agent`), read from `data`
at `4221e28`.

⚠ Left OPEN for the same reason as `CLIENT-01`. The two platforms come from
two runs, and this Prove also asks that source expectations be labelled as
non-authoritative notes, which nothing has yet checked.

### ⛔ One refused arrival cost a whole run its barrier. 2026-10-21

capture-client run 39588980862 dispatched Deluge on Linux's `package` and
`source` routes. Both installed by 11:03:36, but the source lane's arrival
came back from GitHub with no `id`. `install-barrier arrive` tried once,
printed *the check run for deluge source was not created*, and gave up.
Both lanes then waited the whole 1500 seconds and recorded `timeout` with
`missing_routes=source`. Both captures afterwards verified, and
`assemble-capture` only warns on a barrier that did not pass, so the run
was not lost. But the order the barrier exists to prove went unproven, and
the job spent 25 minutes on it.

⭐ **An arrival is now posted up to four times, 2, 4 and 8 seconds apart, and
each refusal prints GitHub's own `message`.** The old line printed nothing
GitHub had said, which is why this run cannot say what the refusal was.
`check-step-bodies` gained a case where the stub refuses the first post the
way GitHub refuses one: the barrier passes, and the job log names the
refusal. With the retry reverted, the same case reads `verdict [timeout]`
and fails; 58 of 58 pass with it.

⚠ A retry after a post that landed but lost its answer posts the same name
twice. `wait` takes the first, so a duplicate changes nothing.

## CLIENT-07: Deluge capture adapter

Source: operator scope and upstream project
Priority: P1 | Effort: L | Status: OPEN

Problem: Deluge's daemon and UI separation can change automation and engine
version evidence across packages.

Approach: Drive the daemon through its supported interface, record bundled
engine identity, and compare official packaging with host package routes.

Prove: live Linux and Windows records prove product and embedded-engine
versions plus agreeing observed surfaces.

### ⭐ The adapter is built, and both Linux routes captured on the session host. 2026-10-21

[`../tools/check/adapter_deluge.go`](../tools/check/adapter_deluge.go) and
[`../tools/check/adapter_deluge_windows.go`](../tools/check/adapter_deluge_windows.go)
register `deluge` on each platform. Their pure decisions are one untagged file,
[`../tools/check/deluge.go`](../tools/check/deluge.go), and `bit-check deluge
--selftest` asks them on any host: 80 cases, all held. `check-bitcheck` runs it.

⭐ **What Deluge publishes, read on 2026-10-16** through `AGENTS.md` rule 8's
route and the vendors' own pages:

| source | what it carries |
| --- | --- |
| PyPI `deluge` | 2.2.0, 2025-04-28, the newest stable: `deluge-2.2.0-py3-none-any.whl` (sha256 `5ac4ce73...`) and an sdist. It declares no libtorrent dependency |
| ftp.osuosl.org (download.deluge-torrent.org) | `source/2.2/deluge-2.2.0.tar.xz` with a `.sha256`. In `windows/`: `deluge-2.2.0-win64-setup.exe` and `-win64-lt2.0-setup.exe`, plus the two win32 builds, each with a `.sha256` |
| GitHub `deluge-torrent/deluge` | 85 tags and **no release objects**, so `resolve-release` has nothing to read. `deluge-2.2.0` is commit `e9777eaa...`. `deluge-2.2.1.dev0` is the newest tag |
| the win64 installer, read with 7-Zip | NSIS. A PyInstaller tree on Python 3.9 with **libtorrent 1.2.19** (`-LT12J0-`), Twisted 22.8.0, pyOpenSSL 25.0.0, setuptools 68.2.2. `deluged.exe` is windowed; `deluged-debug.exe` is the same program with a console |

| index | at 2.2.0? |
| --- | --- |
| Ubuntu 24.04 apt | no: `deluged` is `2.1.2~dev0+20240121-1` |
| PyPI | yes |
| winget `DelugeTeam.Deluge` | yes. Its x64 installer is `http://download.deluge-torrent.org/windows/deluge-2.2.0-win64-setup.exe`, InstallerSha256 `B2E414DD...8F4E889`, the vendor's own `.sha256` |
| Scoop extras `deluge` | yes, the same installer, extracted with 7-Zip |
| Chocolatey `deluge` | no: 2.1.1 |

⭐ **The routes, under the operator's rule that the second route is a package
manager at the same version:**

| platform | route | what it does | `package` |
| --- | --- | --- | --- |
| Linux | `package` | `pip install --only-binary :all: deluge` with no version into a venv at `<prefix>/opt/deluge`. PyPI is the resolver; `package-index.txt` names `pypi`, `deluge`, the version pip reported and the wheel's URL | `python-wheel` |
| Linux | `source` | the `deluge-2.2.0` tag from `resolve-source`, cloned and pip-installed into the same venv path | `python-source` |
| Windows | `package` | `winget install --id DelugeTeam.Deluge --location %ProgramFiles%\Deluge`, no version | `nsis-installer` |
| Windows | `source` | the same tag, cloned behind Git for Windows' coreutils `timeout` and pip-installed into a venv on the runner's Python | `python-source` |

Why these:

- **No `release` route anywhere.** The repository has no release objects, and
  a resolver for the vendor's download directory would be a new source kind
  and route kind. So the second route is the tag, as for `aria2-next`.
- **PyPI beside the tag on Linux** differs in resolver (PyPI's index against
  git refs), delivery (a wheel from files.pythonhosted.org against a clone) and
  origin. ⚠ Both take Deluge's dependencies from PyPI, so the pair is
  independent in how Deluge arrived and not in how Twisted did.
  `pip-freeze.txt` beside each install says what each got.
- **winget on Windows** carries the vendor's own build. Scoop names the same
  installer and would be a second `package` route on a host that runs one.
- ⚠ **So the two Windows routes measure one Deluge on two engines**: the
  vendor's installer carries libtorrent 1.2.19 and the source route 2.0.15.
  The two Linux routes carry 2.0.15. A field that differs by platform is
  published as platform-scoped, under the operator's settled rule.

⛔ **Three pins, each measured.** The pip routes name exactly these and no
other version:

- `libtorrent==2.0.15`. Deluge declares no libtorrent, and PyPI's newest is
  2.1.1, a series newer than Deluge 2.2.0. 2.0.15 is the newest 2.0 release and
  the libtorrent the published qBittorrent pair measured.
- `setuptools<81`. PyPI's setuptools 84.0.0 carries no `pkg_resources`, which
  2.2.0's `pluginmanagerbase.py` imports. The first capture's daemon died at
  start with `ModuleNotFoundError` while `--version` still answered. 81.0.0
  carries it and 82.0.0 does not.
- `pyOpenSSL<26.3`. 2.2.0's `rpcserver.py` makes its certificate with
  `OpenSSL.crypto.X509Req`. 26.2.0 carries it and 26.3.0 does not. The second
  capture's daemon logged *Unable to start deluged* with its RPC already bound.

⛔ Both failures passed `--version`, so a pip route now also runs
`python -I -c "import libtorrent, deluge.core.daemon, deluge.ui.console.main;
from OpenSSL.crypto import X509Req"` in the venv and refuses if it fails.

⛔ **The source route's resolver failed closed, and the fix is in the shared
reader.** `deluge-2.0.0b1` and `deluge-2.0b1` are PEP 440 betas. The scheme
could not order them, and their major is the floor's, so they blocked.
`text_marks_prerelease` now reads a lone `a`, `b` or `c` between two digits
as a marker. Over every target's live refs on 2026-10-16, only two sets of
verdicts move: Deluge's two betas, so its route selects `deluge-2.2.0`, and
Transmission's thirteen `1.80b5` to `2.50b1`. Those thirteen move from
`below_scheme_floor` to `prerelease_by_version`, and 4.1.3 is still selected.
qBittorrent, aria2 and aria2-next do not move. `check-source-route` replays
Deluge's 85 refs: 32 of 32. Its Transmission count is now 99 released by the
floor plus 13 betas.

⭐ **How the build is driven.** `deluged` runs in the foreground with `core.conf`
written whole in a fresh config directory per session. Its RPC is on a free
loopback port, and the torrent is added by `deluge-console "connect
127.0.0.1:<port>; add '<torrent>'"`. With no user and a `127.0.0.1` host, the
console reads the daemon's `localclient` credential from the shared config
directory, so the adapter never holds it.

| knob | value | mirrors |
| --- | --- | --- |
| `dht`, `lsd`, `upnp`, `natpmp`, `utpex` | `false`, and read-only to RPC | every adapter's three surfaces plus the port mapping `aria2-next` measured |
| `new_release_check`, `send_info`, `allow_remote` | `false` | the first defaults to true and asks deluge-torrent.org at start |
| `enc_in_policy`, `enc_out_policy`, `enc_level` | 1, 1, 2 | qBittorrent's `Session\Encryption=0`: libtorrent's `pe_enabled` both ways, either handshake |
| uTP | not set: 2.2.0 has no switch | qBittorrent, which leaves it at libtorrent's default |
| `listen_ports`, `random_port` | the peer port twice, `false`; neither for port 0 | the others' "port 0 leaves the port to the build" |

`stop` sends SIGTERM on Linux. Twisted catches it and 2.2.0's `daemon.py`
shuts the core down, so the build announces `stopped`. It sends KILL after
eight seconds. On Windows `stop` sends `halt` through the console, waits ten
seconds, then runs `taskkill /T`. `stop.txt` records which.

⛔ **Rule 12, which the bundle scan alone would miss twice.** The daemon writes:

- `config/auth` with `localclient:<40 hex>:10`, and `auth.bak` beside it;
- `config/ssl/daemon.pkey`, a PEM private key.

`scan-bundle` calls neither name a credential. So `stop` scrubs on every path
out:

- every `auth`, `auth.*` and `hostlist.conf*`;
- everything under `ssl/`;
- any file anywhere in the workdir that holds the auth password.

The key was found by the first green drive, reading its bundle by hand.

⭐ **Driven on the session host on 2026-10-21, as `ubuntu`, each lane with its
own `HOME`, prefix and claim:**

| step | package | source |
| --- | --- | --- |
| `install-step` | 2.2.0, `acquired=yes`, 45 to 70 seconds; `package_origin` the PyPI wheel | `resolve-source` selected `deluge-2.2.0` live; 2.2.0, `acquired=yes`, commit `e9777eaa...`, about 40 seconds |
| `--version` | `deluged 2.2.0`, `libtorrent: 2.0.15.0`, `Python: 3.12.3`; it left no config directory in `HOME` | the same |
| capture, 45 s, 3 sessions, in `bit-check contained` | 6 announces, 6 peer streams, every peer ID `-DE220s-` with a new tail per session, user agent `Deluge/2.2.0 libtorrent/2.0.15.0` | the same |
| `stop` | `sigterm` in every session | `sigterm` in every session |
| bundle | `scan-bundle` clean; no `auth`, no `ssl/` key, no file holding the password | the same |
| `assemble-dispatch`, by hand | both records `publishable`, `classify_across` `build_equivalent`, 4 fields agreeing; refused only for the run manifests a dispatch's barrier and runner claim write | |

⚠ The two routes' Deluge Python files are byte-identical. They differ in the
plugin eggs: the wheel ships `py3.10` and `py3.12` eggs, and the tag builds
`py3.12` only. `installed-files.txt` beside each install digests every file.

⚠ **The executable `binary=` names on Linux is pip's launcher**, about 300
bytes fixed by the venv's path and the entry point. On two runners with one
`HOME`, the two launchers may digest alike whatever Deluge they start, so
`ACQ-03`'s strength over a Deluge pair speaks for the launcher.
`installed-files.txt` is the digest of the install.

⛔ **What needs a runner, in the order to read a dispatch** (`adapters:
["deluge"]`, `routes: ["package","source"]`, `windows_adapters: ["deluge"]`,
`windows_routes: ["package","source"]`):

1. Whether the Linux lanes repeat the session host on `ubuntu-24.04`'s own
   `python3`, 3.12, with `python3 -m venv` working there.
2. Whether `winget install --location` places `deluged-debug.exe` and
   `deluge-console.exe` in `%ProgramFiles%\Deluge`. The package lane's
   `install.log`, `winget.log` and `winget-version.txt` show it.
3. Whether the frozen `deluged-debug.exe --version` prints on the runner and
   names libtorrent 1.2.19. `version.err` beside the install record has it.
4. Whether the Windows source lane finds `usr\bin\timeout.exe` beside Git for
   Windows, clones, and pip-installs on the runner's `python`, whose version
   must have libtorrent 2.0.15 wheels (cp39 to cp313).
5. Whether the console adds the torrent on both Windows routes, and whether
   `halt` ends the daemon in ten seconds. `client/stop.txt` says `halt` or
   `killed`.
6. Whether the firewall list covers the source route's launcher chain:
   `helpers=` names the venv's `python.exe` and the interpreter its
   `pyvenv.cfg` names.
7. The captures themselves, then `assemble-bundle`.

### ⛔ Run 39588980862: both Windows lanes stopped in their install. 2026-10-21

`capture-client` run 39588980862 on `a8016c2` dispatched `deluge` on both
platforms with `["package","source"]`. Both Linux lanes installed 2.2.0, and
the source lane has commit `e9777eaa`. They then waited at the install
barrier for a reason outside this adapter. Both Windows lanes failed in
*Install the client*, each on a defect of this adapter:

1. **Package lane: `No applicable installer found`, winget exit 2316632080
   (`0x8A150010`).** `DelugeTeam.Deluge` 2.2.0's installer manifest, read
   through `AGENTS.md` rule 8's route on 2026-10-21, declares `InstallerType:
   nullsoft` and x86 and x64 installers, and **no `Scope`**. The adapter passed
   `--scope machine`, the convention of the qBittorrent and Transmission
   routes, whose manifests both declare `Scope: machine` (qBittorrent's lt2
   with `ElevationRequirement: elevatesSelf`). winget treats a `--scope` the
   installer does not declare as a requirement nothing meets. The line now
   carries no `--scope`. It names `--architecture x64`, because the manifest
   offers both architectures. `--location %ProgramFiles%\Deluge` is unchanged,
   so the NSIS installer still installs where the adapter looks, for the
   machine, on an administrator's account.
2. **Source lane: the import check refused.** pip installed the tag on the
   runner's Python 3.12.10, and `deluge.ui.console.main` failed at
   `widgets/sidebar.py`'s `import curses` with `No module named '_curses'`.
   CPython on Windows has no curses. The tag's `setup.py` does not declare
   `windows-curses`; its `requirements.txt` carries `windows-curses;
   sys_platform == 'win32'`. So that requirement, with that marker, joins the
   pins every pip route passes, and it is nothing off Windows. It is
   unpinned, as upstream leaves it: 2.4.2 is the newest, and `pip download
   --platform win_amd64 --python-version 3.12` resolved
   `windows_curses-2.4.2-cp312-cp312-win_amd64.whl`.
   ⚠ The same file caps `twisted[tls]<23` on win32, and that cap is not
   taken: Twisted 22.10's classifiers stop at Python 3.10. Whether Twisted 26
   runs the daemon on Windows is something the next dispatch shows.
   ⚠ The lane's `pkg_resources is deprecated` warning shows the setuptools pin
   applied. The warning comes from `pkg_resources`, which 82.0.0 and later do
   not carry, and the Windows pip line passes the same `delugePins`.

The selftest is 86 of 86, with three new cases planted back and each refused:
`--scope machine` restored, `windows-curses` dropped, and `--architecture x64`
dropped. `pip install --dry-run` with the marked requirement on Linux leaves
`windows-curses` out.

⛔ **What the re-dispatch must prove,** in reading order:

1. The Windows package lane's `install.log` shows winget choosing the x64
   installer. `winget-version.txt` names 2.2.0, and `version.err` names
   libtorrent 1.2.19.
2. The Windows source lane's `pip-freeze.txt` carries `windows-curses` and a
   setuptools below 81, and the import check passes.
3. Then the rest of the earlier list from item 5: the console add, `halt`, the
   firewall list over the launcher chain, and the captures.

### ⛔ Runs 39616084687 and 39617383153: both Windows lanes capture, and the pair cannot be filed. 2026-10-21

⭐ **The Linux pair is published.** The `python-wheel` and `python-source`
records are on the `data` branch, from run 39588980862.

⭐ **`capture-client` run 39616084687 on `539ecde` is green on all four lanes.**
Both Windows lanes now install and capture:

| lane | install | capture |
| --- | --- | --- |
| Windows `package` | 2.2.0, `libtorrent=1.2.19.0` in `winget-version.txt` | 3 sessions, 6 announces, 6 peer streams, `-DE220s-`, every `stop` a `halt` |
| Windows `source` | 2.2.0, commit `e9777eaa`, `windows-curses` installed, import check passed | the same, every `stop` a `halt` |

⛔ **`assemble-bundle` run 39617383153 refused the Windows pair:** `origin:
url-scheme: a retrieval location is https`, on both lanes, because each record
carries both routes' acquisitions. The package lane recorded winget's
installer URL, `http://download.deluge-torrent.org/...`, as its origin.
`Url::parse` refuses a plain http retrieval, because it cannot support a claim
about which bytes arrived, and that rule stands. Scoop's extras manifest names
the same http URL. So no package manager hands this route an https origin.

⭐ **The fix: winget resolves, the vendor's https mirror delivers.**

- The route reads three things from `winget show`: the version, the installer
  URL and `Installer SHA256`.
- It fetches the same file from `https://ftp.osuosl.org/pub/deluge/windows/`
  with `install-rootless --fetch`. It holds the file to the index's digest and
  to the vendor's own `.sha256` beside it.
- It runs the installer as winget runs a `nullsoft` one, `/S
  /D=%ProgramFiles%\Deluge`.
- `package-index.txt` names the https URL as the origin, because that is what
  was really fetched. It names `delivery=https-vendor-mirror-nsis`.
  `install-client` carries that as `package_delivery`, and `assemble-capture`
  uses it in place of `host-package-manager`.

It stays a `package` route because winget's index is what resolved it. Only a
URL of the index's measured shape is mapped; any other is refused. The http
allowance and its decision row are reverted.

⚠ Measured through `AGENTS.md` rule 9's route on 2026-10-21, because this
host's proxy cannot verify TLS to either host:

- `ftp.osuosl.org` answered 200 over https. Its `.sha256` reads
  `b2e414dd...8f4e889  deluge-2.2.0-win64-setup.exe`, and the installer it
  serves digests to the same value, which is the index's InstallerSha256.
- `https://download.deluge-torrent.org` answered 526, an invalid certificate at
  the origin.

A runner fetches osuosl directly.

⛔ **And the patched pair is `divergent`, which is the second finding.** A copy
of the run's real artifacts was patched to the https origin and the delivery,
and `assemble-dispatch` then wrote two valid records. Neither is publishable:
`peer_wire/reserved` and `tracker_http/user_agent` disagree. The vendor's
installer runs libtorrent 1.2.19 and the source route ran 2.0.15, and routes
on one host must run one engine. So the Windows source route now pins
`libtorrent==1.2.19`, under a `sys_platform == "win32"` marker, while Linux
keeps 2.0.15. That library's Windows wheels stop at cp310, and the runner's
`python` is 3.12.10. So the route takes the tool cache's newest x64 Python
3.10, else 3.9, from `RUNNER_TOOL_CACHE` or `AGENT_TOOLSDIRECTORY`, and is
could-not-run without one.

Driven on this host with CPython 3.10.20 and the Windows pins:

- the install and the import check passed;
- `--version` answered `libtorrent: 1.2.19.0`;
- the console printed `Torrent added!`;
- the daemon logged *shutdown successfully* on SIGTERM.

⚠ Linux records 2.0.15 and Windows 1.2.19. A field that differs by platform
is platform-scoped under the operator's settled rule.

| held before commit | result |
| --- | --- |
| `bit-check deluge --selftest` | 96 of 96 |
| plants: the index's http URL recorded as the origin, any `download.deluge-torrent.org` file mapped, `/D=` quoted, the digest taken as written, `windows-curses` dropped, Windows back on libtorrent 2.0.15, the runner's Python 3.12 preferred | each refused |
| `assemble-dispatch` over run 39616084687's real Windows artifacts, the package lane's record patched to the https origin and the delivery | the origin refusal is gone, and two valid records are written. They are `divergent` on `reserved` and `user_agent`, which the engine pin above addresses |
| `install-rootless --fetch` of osuosl from this host | could not run: the proxy cannot verify that TLS; a runner proves it |

⛔ **What the re-dispatch must prove** (`adapters: []`, `windows_adapters:
["deluge"]`, `windows_routes: ["package","source"]`), in reading order:

1. The package lane's `install.log` shows both `install-rootless` fetches from
   `ftp.osuosl.org` succeeding. `rootless-install.txt` records the digest, and
   `nsis-install.txt` records `exit=0`.
2. Its install record carries an https `package_origin` and
   `package_delivery=https-vendor-mirror-nsis`.
3. The source lane's `build-deps.txt` names a Python 3.10 (or 3.9) from the
   tool cache, `pip-freeze.txt` carries `libtorrent==1.2.19`, and
   `version.err` reads `libtorrent: 1.2.19.0`.
4. Both lanes capture, then `assemble-bundle` makes the Windows pair
   `publishable` and `build_equivalent`, with `reserved` and `user_agent`
   agreeing.

## CLIENT-08: BitTorrent capture adapter

Source: operator scope and August 2026 priority sample
Priority: P1 | Effort: L | Status: OPEN

Problem: The proprietary Windows product must be distinguished empirically
from uTorrent despite related ownership and code lineage.

Approach: Isolate installations and compare emitted fields only after separate
two-route captures complete.

Prove: the profile contains independent raw evidence and no inherited field
from the uTorrent adapter.

## CLIENT-09: BiglyBT capture adapter

Source: operator scope and upstream project
Priority: P1 | Effort: L | Status: OPEN

Problem: Java runtime and packaging differences can affect observable client
behavior and must be represented, not flattened.

Approach: Record JVM and package metadata, automate the supported interface,
and acquire official and package-manager builds on both host families.

Prove: runs correlate product, JVM, and host facts with agreeing protocol
observations for the same stable release.

## CLIENT-10: Tixati capture adapter

Source: operator scope and August 2026 priority sample
Priority: P1 | Effort: L | Status: OPEN

Problem: The proprietary Linux and Windows client has no open control source
and may require UI-driven setup.

Approach: Prefer documented controls, use bounded UI automation if necessary,
and verify installation and cleanup independently of the UI.

Prove: two-route captures on supported hosts pass connector agreement and
re-run unattended from a clean guest.

## CLIENT-11: KTorrent capture adapter

Source: operator scope and KDE upstream
Priority: P1 | Effort: L | Status: OPEN

Problem: Linux distribution packages may trail or patch the upstream stable
release.

Approach: Compare a verified official build route with a distribution package
and block publication when their upstream build identities diverge.

Prove: two Linux routes resolve to one stable build and produce agreeing live
profiles, or emit a documented blocking divergence.

## CLIENT-12: Free Download Manager capture adapter

Source: operator scope and August 2026 priority sample
Priority: P1 | Effort: L | Status: OPEN

Problem: FDM combines general download behavior with BitTorrent support and
needs a torrent-specific controlled path.

Approach: Exercise only synthetic torrents, compare official and package
routes, and separate HTTP download headers from BitTorrent identity fields.

Prove: Linux and Windows runs produce torrent-specific two-connector evidence
without contacting a public swarm.

## CLIENT-13: Zona capture adapter

Source: operator scope and August 2026 priority sample
Priority: P1 | Effort: L | Status: OPEN

Problem: Zona is proprietary, Windows-oriented, and may not expose unattended
controls or a dependable second package route.

Approach: First measure current availability and terms, then use a disposable
Windows adapter. Keep the entry open if two independent routes cannot be
verified; never substitute an inferred profile.

Prove: a same-version two-route capture passes, or the entry records a precise
blocker and no Zona profile is published.

## CLIENT-14: aria2-next, acquired from its releases and driven over RPC

Source: operator direction on 2026-09-09, after ten dispatches produced no aria2
capture
Priority: P1 | Effort: L | Status: DONE

Problem: `CLIENT-05`'s target cannot be captured. Ten client capture runs have
been dispatched and not one has reached the *Capture* step: every aria2 lane
stops inside *Install the client* and the job ends producing no log and no
artifact. ⛔ Four independent bounds have been measured not to fire on that step
and five readings of it have been refuted, so this entry does not wait on the
diagnosis - it changes the target.

Premise: the operator's direction is to replace the target with
`AnInsomniacy/aria2-next`, acquire it from that project's own releases, and drive
every operation over the product's RPC interface where RPC serves the same
purpose as the command line. ⚠ Nothing about that target is measured yet: this
entry is the record of the direction, not of a capability.

Approach: a new adapter beside the existing ones, declaring its release
repository, tag scheme and asset pattern through `describe` the way `aria2.sh`
already does, so `resolve-release.sh` needs no per-target knowledge. ⛔ The
`version`, `start` and `stop` subcommands go through RPC rather than the command
line **only where RPC answers the same question**; a subcommand whose answer RPC
cannot give keeps the command-line form, and the adapter says which is which.
⚠ The second route stays a question rather than an assumption: two routes must
resolve one version for the same host, and whether this target's releases and any
package index do that is a measurement nobody has taken.

Decision: a new entry rather than an edit to `CLIENT-05`. That entry carries the
measured history of ten dispatches - the four bounds, the five refuted readings,
the transmission control - and rewriting its target would falsify a record this
project needs to keep. ⚠ `CLIENT-05` stays open on the hang; this entry is what
the two-route capture is attempted through.

Prove: `sh scripts/acquisition/check-release-route.sh` selects exactly one asset
for the new adapter from a recorded listing and refuses a newest release that
carries none, `sh scripts/capture/check-capture-client.sh` holds, and a
dispatched capture reaches the *Capture* step and uploads an evidence bundle that
`sha256sum -c` verifies outside the run that wrote it.

⚠ **The Prove said "a dispatched TWO-ROUTE capture" and that is now known to be
unreachable for this target**, because no package index carries it and no second
route has been measured. Amended on 2026-09-09 rather than left standing: a
`Prove` must stay runnable against the current tree, and one that waits on a
route this entry has established does not exist would never close. ⛔ The
two-route record the work order needs is not this entry's to produce; what this
entry can produce is the first capture that reaches the *Capture* step.

Closure evidence: run on 2026-09-09. `sh scripts/acquisition/check-release-route.sh`
is 23 cases, 23 passed, 0 failed, exit 0 - including `aria2-next` selecting
`aria2-next-2.7.4-linux-x86_64` and the newest-release-with-no-assets case
refusing and naming 2.7.5. `sh scripts/common/check-gate.sh` is 31 checks, 30
passed, 0 failed, 1 skipped (`check-remote-items`, the one observed skip on a
session host), exit 0. `cargo test --workspace --locked --all-targets` is 559
passed, 0 failed over 53 binaries; `cargo clippy --workspace --locked
--all-targets -- -D warnings` and `cargo fmt --all -- --check` exit 0.
capture-client runs 11 and 12 both completed with every step green, each in about
two minutes, and each bundle verified with `sha256sum -c` outside the run that
wrote it - three files, all `OK`, exit 0. CI run 94 on `f10021f` is green on all
three jobs.

### ⭐ What has now been measured, on 2026-09-09

⛔ **The paragraph that stood here said "nothing here has been run", and that is
no longer true.** The target has been listed, fetched, verified, installed,
asked its version and driven over RPC on this session's host. Installing a
product and asking it questions is not a capture, so it needed no disposable
host; nothing below was driven against a tracker, and no capture has run.

**The build.** `aria2-next-2.7.4-linux-x86_64`, fetched from the project's own
release and verified against the vendor's published
`aria2-next-2.7.4-checksums.sha256` with `sha256sum -c` **before it was run** -
a reader this project did not write, checking a digest this project did not
compute. It answers `Aria2 Next version 2.7.4` and lists
`libtorrent/2.1.1` among its libraries, so this is an aria2-compatible front end
over libtorrent rather than a fork of aria2's own BitTorrent code.

| question | measured answer |
| --- | --- |
| release tags | `v2.7.4`, so prefix `v` and three components |
| assets per release | eight: seven builds over four platforms - Android arm64 only, the other three in two architectures - plus a checksums file |
| the Linux artifact | `aria2-next-{version}-linux-x86_64`, matching exactly one |
| install cost | **1.2 seconds** - a download and a `chmod`, where `aria2`'s release route is a source build taking minutes |
| `version` over the command line | `Aria2 Next version 2.7.4` |
| `version` over RPC | `{"product":"aria2-next","version":"2.7.4","rpcVersion":"1.1.0",...}` |
| `start` over RPC | `aria2.addTorrent` accepted a base64 torrent and named a download |
| `stop` over RPC | `aria2.shutdown` answered `OK` and the build exited on its own in ~6 seconds |

⛔ **THE VERSION IS THE FOURTH FIELD AND `aria2.sh`'s PARSE WOULD HAVE PUBLISHED
THE WORD `version`.** aria2 prints `aria2 version 1.37.0`; this prints `Aria2
Next version 2.7.4`. `awk '{print $3}'` returns the literal string `version`
here, and it would have passed every check in this repository: it is a non-empty
field. ⭐ The adapter anchors on the `version` token instead of a column, because
the product name is prose whose word count is the vendor's to change.

⛔ **`--enable-dht6` DOES NOT EXIST ON THIS BUILD AND IS ACCEPTED ANYWAY**, with
`Legacy aria2 input from command line: enable-dht6; accepted and skipped;
libtorrent uses one DHT switch for both families`. A flag that is accepted and
skipped is worse than one that is refused: the step succeeds and the switch it
named was never set.

⛔ **AND A FOURTH DISCOVERY SURFACE, FOUND BY READING SOCKETS RATHER THAN HELP
TEXT.** With DHT, LPD and peer exchange all off - the three the adapter contract
names - the running build was still bound to **UDP 1900**, read out of
`/proc/<pid>/fd` and `/proc/net/udp`. `bt-port-mapping`, "Enable UPnP and NAT-PMP
port mapping", defaults to **true**. ⚠ Why a link-local announce escapes a
contained host when a tracker does not is in
[`../scripts/capture/adapters/README.md`](../scripts/capture/adapters/README.md),
which owns the switch list. With `--bt-port-mapping=false` the build binds its
RPC port, its peer port on TCP and UDP, and nothing else.

⭐ **RPC IS WHAT MAKES THAT CHECKABLE, AND IT IS THE CASE THE ENTRY WAS OPENED
FOR.** A command line reports what was *passed*; `aria2.getGlobalOption` reports
what took *effect*. On a product that accepts an option it does not have, those
are different questions, and only one of them is worth recording. The adapter
reads all five switches back from the build and refuses if any is not `false`.

### ⛔ The second route, which is now a measured absence rather than a question

**`aria2-next` is in no package index.** `apt-cache search aria2` on this image
lists `aria2`, `libaria2-0`, `libaria2-0-dev` and `persepolis`, and no fork.

⚠ **The hazard is not the missing package - it is that the obvious fallback
SUCCEEDS.** `apt-get install aria2` exits 0, installs a different product at a
different version, and hands `ACQ-03` a second route that acquired somebody
else's build. The adapter's `package` route therefore refuses by name and says
why, rather than being absent and letting a caller improvise.

⛔ **So this target has ONE route today and `E-ACQ-01` refuses a record with
one.** That is the same wall `CLIENT-05` hit, reached from the other side: aria2
has two routes and cannot be installed, and aria2-next installs in a second and
has one route. ⚠ `candidate_routes` in the catalogue names `source-build` as the
second, which is a research lead and not an availability claim - nothing has
built this target from source.

### ⛔ A release can be newest and empty, and that was caught live

**Measured on 2026-09-09:** `v2.7.5` was published at `12:11:10Z` and carried
**zero** assets when read 47 seconds later, while `v2.7.4` beside it carried
eight. A live resolution against the vendor at that moment refused:
`select-asset: aria2-next 2.7.5 (v2.7.5) offers 0 asset(s)`.

⭐ **The resolver does not fall back, and that is now proved rather than
assumed.** Rule 5 targets the newest stable release, so quietly resolving
`v2.7.4` because `v2.7.5` had nothing to offer would be wrong in a way that looks
like success. `aria2-next-unpublished.json` keeps both releases so the fallback
has something to fall back *to*, and `check-release-route.sh` reads `2.7.5` back
out of the refusal, because "it refused" is satisfied by refusing for any reason
at all.

⚠ **This is a property of every target with a release route, not of this
vendor.** A release exists from the moment it is created; its binaries arrive
when whatever builds them finishes. Any capture dispatched inside that window
resolves nothing.

### ⭐ THE CAPTURE RAN. capture-client run 11, 2026-09-09

⛔ **THE ELEVENTH DISPATCH IS THE FIRST TO REACH THE *Capture* STEP, AND IT PASSED
EVERY STEP.** Ten aria2 dispatches stopped inside *Install the client* and
produced no log and no artifact. This job ran **2 minutes 1 second** end to end
(13:30:55Z to 13:32:56Z) and its install step took **one second**.

| field | value |
| --- | --- |
| `kind` | `client` |
| `stock_client` | `true` |
| `measured_build` | `2.7.5` |
| `acquired` | `yes`, with `preexisting_version` empty |
| `installed_binary` | `/usr/local/bin/aria2-next`, sha256 `5190b4f5…` |
| `egress` | `closed`, from `/proc/net/route` |
| `announces` | 1 |
| `peer_streams` | 1, dialled at `127.0.0.1:51413` |
| `evidence` | 3 files, `evidence_verified_by=sha256sum` |

⭐ **AND THE BUNDLE VERIFIES OUTSIDE THE RUN THAT WROTE IT.** Downloaded here and
checked with `sha256sum -c SHA256SUMS`: three files, all `OK`, exit 0. That is
the last clause of this entry's `Prove`, met by a reader this project did not
write.

### ⛔ The identity: a stock aria2-next announces as qBittorrent

The attestation records peer ID `2d7142353233302d733251627a596a74284c4f51`,
which is twenty bytes reading `-qB5230-s2QbzYjt(LOQ`.

⛔ **Its first eight bytes are qBittorrent 5.2.3.0's Azureus-style prefix**, put
on the wire by a stock `aria2-next` 2.7.5 announcing to this project's own
tracker on a host with no route off it.

⚠ **The hex is written in backticks after the words "peer ID" on purpose.**
`check-no-secrets --public` refuses a bare forty-digit hex identifier and allows
exactly that shape; writing the field as it appears in the attestation turned the
gate red here first. The narrow exclusion is the convention, not a workaround -
`docs/security/secrets.md` says to narrow a pattern rather than switch the rule
off, and this needed no narrowing at all.

⭐ **This was predicted from an RPC read and then OBSERVED, and the difference is
the whole point.** Before the capture, `aria2.getGlobalOption` reported
`bt-peer-id-prefix: -qB5230-`; that is a configured value. The attestation above
is the identity the running build actually emitted. This project publishes the
second and uses the first only to know where to look. ⚠ Nothing in the adapter
overrides the prefix, because the identity a stock build emits IS the
measurement.

⚠ **It is exactly the case the catalogue exists to catch.** A peer-ID table, a
client-ID list or a swarm statistic would file this build under qBittorrent
5.2.3.0. Only a capture separates them - and separating them is what
`docs/AGENTS.md` absolute 1 is for.

### ⭐ The asset-less release was real and it was transient

⚠ **The run resolved and installed `2.7.5`,** the same release that carried zero
assets when this session read it at `12:11:57Z`. The vendor's assets arrived some
time before `13:31:34Z`. ⛔ That does not weaken the measurement - it is the
measurement: the window is real, it is short, and a capture dispatched inside it
resolves nothing. The fixture and the harness case keep the behaviour pinned
whether or not the window is open today.

### ⛔ What run 11 found that no reading had: a secret in the artifact

⛔ **The uploaded evidence bundle contained `client/rpc-token`.** The adapter
generates a per-run JSON-RPC token so `stop` can authenticate its shutdown and
writes it into the workdir - and `capture-client` packs that workdir as the
artifact. ⚠ Each step was reasonable and the composition put a secret inside an
artifact, which is what `docs/AGENTS.md` rule 12 refuses without grading how weak
the secret is.

⭐ **Fixed in the adapter and promoted to a contract rule.** `stop` unlinks the
token before anything packs the directory, `start` writes it under `umask 077`,
and `adapters/README.md` now says that the workdir IS the evidence bundle.
Driven after the fix: the file is mode `600` while the build runs, absent after
`stop`, and `rpc-shutdown.json` still records `"result":"OK"`.

⚠ **No reading found it and the gate could not**, because the bundle is not in
the tree that `check-no-secrets` scans. It was found by downloading the artifact
a real dispatch produced and listing what was in it.

#### ⭐ ADDED 2026-09-10: the fix was a gate on ONE path, and now it is not

⛔ **Unlinking the token in `stop` needs every adapter to remember, `stop` to be
reached, and `stop` to SUCCEED** - and `capture-client` records a non-zero
`adapter_stop_status` rather than refusing on it. That is the single most
recurring hole `docs/methodology/reviews.md` names: a gate on one of several
paths into the same action, with the other paths silent.

⭐ **`capture-client` now refuses at the choke point every capture passes
through**, on two rules, because the two failures look nothing alike:

| what it catches | why a filename rule alone is not enough |
| --- | --- |
| a file whose NAME says it holds a credential - `*token*`, `*secret*`, `*.pem`, `id_rsa*`, `.env*` | this is the run's own state, and `rpc-token` is exactly it |
| a secret-shaped MEMBER inside any text document under the output directory | ⛔ this is the product handing one BACK: an options dump names its own `rpc-secret` and lands in a file called nothing in particular |

⚠ **Whether THIS product's options dump carries its secret is a question the next
dispatch answers**, not one this entry settles. The rule is written for the shape
rather than for the one instance, and `-l` is used so the guard names the file
and never prints the value it found.

⛔ **A WORD BOUNDARY, NOT A SUBSTRING, and that is a case rather than a comment.**
`tokenizer.log` contains `token` and is not one; a near-miss control runs beside
the two refusals, because a rule that refused everything would pass both of them.
Three plants, each verified to apply and to leave the script parseable: the name
rule disabled, the value rule disabled, and the name rule widened to a substring
- refused, refused, and the near-miss control fired.

⚠ **`check-no-secrets` still cannot reach an artifact**, and that is unchanged.
This guard lives in the capture path because that is where the subject is; the
tree scanner's scope is the tree.

#### ⛔ AND A DOOR SWEEP THE SAME DAY FOUND THE SECOND PATH

⛔ **`capture-client.yml` UPLOADS TWO ARTIFACTS PER LANE, AND THE GUARD ABOVE
COVERED ONE.** The second is the INSTALL workdir's logs - `install-<route>/*.log`
and `*.err` - written by an adapter's `install`, which clones, configures and
builds, all of which log. ⚠ So the fix for a gate on one of several paths had
itself been put on one of two paths, which is the shape
`docs/methodology/reviews.md` calls the most recurring hole there is.

⭐ **`install-client` carries the same two rules now, deliberately identical.**
Two spellings of "what a credential looks like" would drift, and the day they did
the one a reader trusts would be whichever they saw first. Two cases, a refusal
and a near miss, because a rule that refused everything would pass the first.

⚠ **No route needs a credential today**, which is exactly the condition that
document names as the easiest place to get a scope wrong: every reading agrees on
every file. The guard is written for the shape rather than for an instance.

### ⭐ ADDED AFTER CLOSURE on 2026-09-09: the second route exists

⛔ **The closure evidence above is a dated measurement and is not rewritten.**
This section records what was built after it, because the sentence below it -
"this target has ONE acquisition route" - stopped being true the same day.

⭐ **`aria2-next` now has a `source` route and both routes resolve 2.7.5.**
Measured here: `git clone --depth 1 --branch v2.7.5`, then the project's own
documented build - `cmake --preset default` and `cmake --build --preset default`
- produced a binary answering `Aria2 Next version 2.7.5`, the same version the
release asset carries.

| | release route | source route |
| --- | --- | --- |
| resolver | the releases API | git refs |
| delivery | one HTTPS asset | a git clone |
| artifact | 14 MB, stripped | **256 MB**, `RelWithDebInfo` |
| cost | 1.2 seconds | **366 seconds**, driven through the adapter |
| version | 2.7.5 | 2.7.5 |

⭐ **One version, two provably different builds** - which is the case `ACQ-03`
exists to classify, and the first time this project has had one.

⚠ **HOW INDEPENDENT THEY ARE IS STATED RATHER THAN CLAIMED.** They differ in
resolver and in delivery, which is what `E-ACQ-07` and `E-ACQ-08` compare.
⛔ They do NOT differ in origin: whoever controls that repository controls both.
That is weaker than a distribution index against a vendor release, and it is
written here so two green checks do not imply otherwise.

⛔ **The bound needed room and only one route got it.** `install-client` bounds
the adapter call at 420 seconds and the build measured 366 - fifty-four seconds
of margin, which a slower runner spends. The workflow now passes
`BIT_IDS_INSTALL_TIMEOUT: 500` for `source` lanes and 420 for every other, which
keeps it under the step's own 540 so an overrun is refused by `install-client`
with a message rather than killed by the outer bound with nothing to read.
⚠ The outer `timeout -k 30 540` literal is deliberately untouched:
`check-step-bodies` plants against that exact string.

### ⭐ THE FIRST TWO-ROUTE CAPTURE. capture-client run 14, 2026-09-09

⛔ **Both lanes green, one version, two builds.** This is the first time this
project has acquired one target through two routes on two hosts and measured
both.

| | `release` lane | `source` lane |
| --- | --- | --- |
| *Install the client* | **6 s** | **496 s** |
| `reported_version` | 2.7.5 | 2.7.5 |
| `installed_binary_sha256` | `5190b4f5…` | `5d67f6c7…` |
| `acquired` | yes | yes |
| `preexisting_version` | empty | empty |
| `measured_build` | 2.7.5 | 2.7.5 |
| `stock_client` | true | true |
| `egress` | closed | closed |
| `announces` | 1 | 1 |
| bundle | 3 files, `sha256sum -c` **OK**, exit 0 | 3 files, `sha256sum -c` **OK**, exit 0 |

⭐ **Two routes, one stable version, two provably different builds**, which is
what absolute 4 asks for and what `ACQ-03` exists to classify. ⚠ Both lanes
resolved through the SAME resolution, which is how the version is made equal
rather than checked afterwards.

### ⭐ And the identity is the same from both builds

| lane | peer ID |
| --- | --- |
| release | `-qB5230-3SGS8~CB*gUf` |
| source | `-qB5230-*eQ2phy)!)RO` |

⭐ **Four captures of this target now agree on the eight-byte prefix and differ
in every twelve-byte tail.** ⭐ Three of the four ran the vendor's stripped 14 MB
release asset and the fourth ran a 256 MB `RelWithDebInfo` build made **on the
runner** from the tag, and all four put `-qB5230-` on the wire. ⚠ So the prefix is
a property of the source rather than of the vendor's build pipeline, which one
route alone could not have separated.

⭐ **The rule 12 fix held on both lanes**: neither bundle contains `rpc-token`,
and both still carry `rpc-add`, `rpc-options`, `rpc-shutdown` and `rpc-version`.

### ⛔ 496 against 500 is not a margin, and the bounds moved

⚠ **The source install took 496 seconds against a bound of 500.** Four seconds.
The same build was measured at **366 seconds** on the session host, so the runner
is about **1.35x slower** - which makes a locally measured build time a lower
bound on what a runner needs and never an estimate of it. ⚠ The lower-bound half
was refuted on 2026-10-14: `capture-client` run 35's runner reached `v3.0.2`'s
compile error in 508 seconds where the session host took 588. The other half
stands in both directions.

⭐ **Raised with the measured number rather than a comfortable one:** the inner
bound is 900 for `source` lanes and stays 420 for every other, and the step's own
bound went from 540 to 1080 because the inner one must stay the smaller of the
two. ⚠ `check-step-bodies` plants against that outer literal, so its plant string
moved in the same change - and `replace_once` refuses a literal it cannot find,
so the two cannot drift apart silently.

### ⛔ THE PAIR WAS ASSEMBLED ON 2026-09-09 AND IT IS NOT TWO ROUTES

⛔ **Both lanes resolved through ONE listing, and `E-ACQ-07` calls that one
route.** Measured by diffing the two lanes' own `release/resolution.txt`: they
differ **only in their timestamps**. One `source_url`, one `listing_sha256`, one
`selected_tag`, one `asset_url`.

⚠ **This entry's own comment in `aria2-next.sh` says otherwise** - *"They differ
in resolver - the releases API against git refs"* - and the artifacts refute it.
The source route is **handed** `BIT_IDS_RELEASE_TAG` and resolves nothing; the
comment describes a design that was never wired. ⭐ It is corrected in the file
rather than argued away here.

⛔ **And the two lanes cannot reach `BuildEquivalent` even so**: they put
different peer-ID tails on the wire, `classify_across` compares every field both
records measured, and a disagreement is `Divergent`. `CI-09` carries the whole
set of refusals and what each one needs.

### ⭐ The peer ID differs between SURFACES inside one run

⛔ **The tail is per-connection, not per-run, and every record here has said
per-run.** Each lane's bundle carries two transcripts and they disagree:

| lane | `tracker_http` announce | `peer_wire` handshake |
| --- | --- | --- |
| release | `-qB5230-3SGS8~CB*gUf` | `-qB5230-O*4PZEf5_2aU` |
| source | `-qB5230-*eQ2phy)!)RO` | `-qB5230-kTAq!FSXbxh2` |

⚠ **Nothing had read the second transcript.** The attestation records the
tracker's value as `measured_build`'s companion `measured_peer_id`, so a reader
working from attestations alone sees one identity per run and concludes the tail
varies per run. ⭐ The schema already expects two: `tracker_http/peer_id` and
`peer_wire/peer_id` are separate paths with separate fixed widths.

### ⛔ The two builds differ in their compiler and in nothing else they report

Both lanes' `version.err` are 1259 bytes and `diff` answers `20c20`:
`gcc 11.4.0` against `gcc 13.3.0`. Enabled Features, Hash Algorithms and
Libraries - `libtorrent/2.1.1` and `OpenSSL/3.5.6` among them - are identical
strings. ⚠ That is a **weaker** difference than `ACQ-03`'s aria2 pair, where the
two builds differed in features, TLS library and compiler, and it is recorded
because "two provably different builds" above is true of the bytes and says
nothing about what they do.

### ⚠ What this entry still does not claim

⭐ **Two captures ARE a `Profile` now, since 2026-09-15.** `capture-client` run 17
assembled into two records and `classify_across` ran over the real pair,
answering `Divergent` on their peer-ID tails - the verdict `check-assemble` had
only ever read off synthetic records. ⚠ Nothing is stored in the tree and nothing
is published: both records are `provisional`. [`ci.md`](ci.md) under `CI-09`
carries what they hold and what makes them provisional.

⭐ **A SECOND CAPTURE ANSWERED THE SAMPLE QUESTION. capture-client run 12** ran
the fixed adapter - `adapter_sha256` differs from run 11's, which is how the
record says so - and measured the same build, `2.7.5`, on a fresh host.

| run | peer ID | prefix |
| --- | --- | --- |
| 11 | `-qB5230-s2QbzYjt(LOQ` | `-qB5230-` |
| 12 | `-qB5230-*MWZBxpa4ngj` | `-qB5230-` |

⭐ **The eight-byte prefix is identical and the twelve bytes after it are not.**
That is the shape a peer ID is specified to have - a client identity followed by
a per-run random tail - and it is now measured for this build rather than assumed
from the specification. ⚠ Two samples of one version on one platform: it says the
prefix is stable across runs, and nothing about across versions.

⭐ **Run 12 also confirms the rule 12 fix on the real path.** Its bundle carries
`rpc-add.json`, `rpc-options.json`, `rpc-shutdown.json` and `rpc-version.json`
and NO `rpc-token`, and `sha256sum -c` verifies its three evidence files outside
the run that wrote it.

⛔ **And this says nothing about `CLIENT-05`'s hang.** A different target
installing cleanly does not diagnose why aria2 does not. `CLIENT-05` stays open
on that, and what run 11 removes is the blockage in front of this entry rather
than the question behind that one.

⚠ **And RPC is a seam, not a guarantee.** Driving a build through its own RPC
interface means the capture measures what that interface causes the build to put
on the wire, which is the same standard every adapter here already meets - and it
also means an adapter failure and a product refusal arrive through one channel,
so the adapter keeps them apart: a JSON-RPC error arrives with HTTP 200 and an
`error` member, so a caller reading only curl's status would take a refusal for
an answer.

### ⛔ 3.0.2 stopped reporting a switch and carried a fifth surface. 2026-10-14

`capture-client` run 35, on `a39d933`, resolved `3.0.2` on both lanes: the
release asset `aria2-next-3.0.2-linux-x86_64`, sha256 `09e34467…afb0d4` and
`OK` against the vendor's checksums, and the tag `v3.0.2` from the repository's
refs. ⚠ The vendor published `3.0.2` at 02:10Z that day, two hours before the
run, and `3.0.0` - a new core on libtorrent 2.1 "with qBittorrent's defaults", by
its release notes - on 2026-10-12. The contained runs of 2026-10-11 measured
`2.8.6`. ⛔ **The release lane was refused at `start`**, by the read-back this
file added for `--enable-dht6`: *aria2-next reports enable-dht6 is not false; a
discovery surface is live*.

⭐ **The bundle came back, which is `CI-09`'s fix of the same day doing its job**,
and its `rpc-options.json` says why:

| option | 3.0.2, read back by run 35 |
| --- | --- |
| `enable-dht6` | not reported; `2.8.6` reported it `false`, or the guard would have refused those runs |
| `enable-dht`, `bt-enable-lpd`, `enable-peer-exchange`, `bt-port-mapping` | `false` |
| `ed2k-port-mapping` | **`true`** |
| `ed2k-listen-port`, `ed2k-udp-listen-port` | `4662`, `4672` |

`rpc-version.json` lists `ED2K` among `enabledFeatures`. ⚠ ED2K is not new:
the `v2.8.6` tree carries `src/Ed2kCommand.cc` and `src/Ed2kKadCommand.cc`.
`--ed2k-port-mapping` is, by `3.0.0`'s release notes - "forwards the ED2K ports
through UPnP IGD" - which also say its ED2K was rewritten against aMule,
"including ... Kad and server search". ⛔ **So the refusal
was right and its reason was the smaller half.** A key the guard required had
gone, and beside it was a UPnP switch for a second protocol, on by default, that
nothing in this file named. Dropping `enable-dht6` from the list would have
passed that build.

| where | what changed |
| --- | --- |
| `aria2-next.sh` `start` | passes `--ed2k-port-mapping=false`; the named list is every discovery switch the file passes, so `enable-dht6` left it and `ed2k-port-mapping` joined it |
| the same | every KEY the build reports in the DHT, port-mapping, local-discovery or peer-exchange families must read exactly `false`, named in the file or not; a value the parse cannot read refuses |
| the same | every option the build reports as `true` must be one of thirteen read off 3.0.2's dump and judged harmless, so a new default reported as `true` refuses whatever it is called; a surface turned on by another value, a port or `auto`, is not caught |
| the same | `--bt-require-crypto=false` is gone: 3.0.2 maps it onto `bt-encryption` and only a value other than `false` sets anything, so it set nothing. The stock `bt-encryption` is `preferred`, under which libtorrent prefers an encrypted handshake on the connections it opens; the observer has an MSE surface, and what the next capture sees is the measurement |

⭐ **Driven against those bytes on the session host**, inside a network namespace
holding loopback alone, `start` given run 35's own torrent. Each plant was
replaced exactly once in a scratch copy and verified to have changed it:

| adapter | `start` |
| --- | --- |
| `a39d933`'s | exit 1, *reports enable-dht6 is not false*: the runner's refusal reproduced |
| this change | exit 0, the torrent accepted; the build held TCP 6800 for RPC and TCP and UDP 51413 for peers, and nothing on 4662 or 4672 |
| the flag not passed | ⭐ exit 1, by the named list |
| the flag not passed and not named | ⭐ exit 1, by the family keys |
| the flag not passed, not named, and the families matching nothing | ⭐ exit 1, by the `true` list: *reports ed2k-port-mapping as true, and this file has not judged it* |
| `follow-torrent` taken off the `true` list | ⭐ exit 1, naming it: the list is read, not decorative |
| the `true` loop fed by a pipe | ⚠ **exit 0 with the refusal printed**: `refuse` ended a subshell, which is why both loops read a here-document |
| this change again, the control | exit 0 |
| this change, over the source build of `v3.0.2` that `CI-09` describes, sha256 `6743a8b2…87cd4bb` | exit 0; the release asset, `09e34467…afb0d4`, driven beside it the same way, also 0 |

⚠ **What the sockets do not show.** A namespace holding loopback alone gives
libtorrent no interface to map a port from, so the absence of UDP 1900 there
says little about the switch; the RPC read-back is the evidence. ⚠ And by the
`v3.0.2` source, read rather than run, the ED2K network is created only by an
ED2K download or call (`src/core/Engine.cc`) and its port mapping starts inside
it, so in a BitTorrent capture the switch is latent rather than live. It is held
to `false` either way.

⭐ **The identity 3.0.2 defaults to is unchanged**: `bt-peer-id-prefix` is
`-qB5230-` and `bt-user-agent` `qBittorrent/5.2.3`, read back from the same
options dump. ⚠ That is the configured default, not a wire observation.

⛔ **The source lane never reached `start`**: `v3.0.2` does not compile on the
runner's GCC 13, because `src/disk/File.cc` calls `std::exchange` without
`<utility>`. The route now configures with `CXXFLAGS='-include utility'` and
changes no file the vendor wrote; `CI-09` carries the measurement and the
bounds that moved with it.

⭐ **Run 36 put both builds on the wire the same day.** Through every guard
above, each announced as `qBittorrent/5.2.3` with `-qB5230-` peer IDs, over
two sessions whose tails differ, and answered one of the lab's two peer
connections. ⚠ The two builds' `--version` answers are byte-identical now, so
the compiler that told 2.7.5's pair apart is no longer in what they report.
`CI-09` carries the records.

### ⭐ aria2-next's Windows half is built, and not yet dispatched. 2026-10-15

[`../tools/check/adapter_aria2-next_windows.go`](../tools/check/adapter_aria2-next_windows.go)
registers `aria2-next` on Windows, so `capture-client.yml` can run with
`windows_adapters: ["aria2-next"]` over its default routes, `release` and
`package`.
[`../tools/check/a2nwindows.go`](../tools/check/a2nwindows.go) holds what it
decides without Windows in it: the command line, the version reading, Scoop's
manifest, where Scoop placed the build, and the JSON-RPC bodies and answers.
`bit-check aria2-next-windows --selftest` asks each of them on any host, 84
cases, all held, and `check-bitcheck` runs it. ⭐ The option read-back moved
into that file as one function, `aria2NextOptionsRefusal`, and the Linux half
now calls it with the same messages, so the two halves judge one list.

| route | what it does | `package` |
| --- | --- | --- |
| `release` | `aria2-next-<version>-windows-x86_64.exe` from GitHub, placed by `install-rootless --install` at `<prefix>\bin\aria2-next.exe` under the resolution's digests. It reaches `both`: the vendor's checksums document and the listing's digest | `pe-binary` |
| `package` | Scoop's own installer, fetched by `install-rootless --fetch` and run with `-ScoopDir %USERPROFILE%\scoop -RunAsAdmin`, then `scoop install main/aria2-next --no-update-scoop` with no version. The manifest and the bucket's git commit are kept in `scoop-index.txt` before the install. The placed file must digest to the manifest's hash and answer the manifest's version | `pe-binary` |

⭐ **The routes were read rather than assumed**, through `AGENTS.md` rule 8's
route on 2026-10-15:

| source | what it carries for Windows x64 |
| --- | --- |
| the vendor's `v3.0.3` release | eight assets, two of them Windows builds: `aria2-next-3.0.3-windows-x86_64.exe`, 18225664 bytes, listing digest `sha256:b2f2f7a9...d2be9465`, and `-windows-arm64.exe`. `aria2-next-3.0.3-checksums.sha256` names the same digest. The file is a console PE built with llvm-mingw against the UCRT, and imports no Visual C++ runtime |
| winget, `AnInsomniacy.Aria2Next` | ⛔ stops at **2.8.3**, a `portable` installer of the vendor's asset. Its 3.0.3 pull request, microsoft/winget-pkgs#452639, was closed unmerged. Its 3.0.4 pull request is open |
| Scoop, `ScoopInstaller/Main` `bucket/aria2-next.json` | ⭐ **3.0.3** from commit `1359d92` (2026-10-14T08:38Z) to `2384d7c` (2026-10-15T08:38Z), with the vendor's own x64 URL and hash `b2f2f7a9...d2be9465`. It names **3.0.4** since then. `Extras`, `Versions` and `Nonportable` carry no `aria2-next` |
| Chocolatey's community feed | no `aria2-next`, `aria2next` or `aria2.next` package. The first page of a search for `aria2` answers versions of `aria2`, `persepolis` and `firedm` |

So Scoop is the one package manager that carries this product at the vendor's
version, which is the operator's condition for a second route. ⛔ **Its pair
shares its origin, as Transmission's winget pair does.** Scoop's x64 URL is the
release route's own GitHub asset. The resolvers differ: a commit of Scoop's
bucket against the vendor's release listing. So do the delivery and its check:
Scoop's downloader and manifest hash against `install-rootless` and the
vendor's document. That is what `E-ACQ-07` and `E-ACQ-08` compare. ⚠ Scoop's
bucket is kept current by Scoop's own bot reading the vendor's releases, so
this index follows the vendor rather than judging it.

⛔ **3.0.3 is no longer what either route resolves.** The vendor published
3.0.4 at 2026-10-15T05:02Z and Scoop's bucket followed at 08:38Z. A dispatch
now would resolve 3.0.4 on the release lane and install 3.0.4 on the package
lane: one version on both, by each route's own resolver. ⚠ The adapter pins
nothing, which is absolute 5. The published Linux pair is 3.0.3, and no Linux
3.0.4 pair exists, so a Windows capture now would not sit at the Linux pair's
version. The resolution was driven offline over the live listing: 3.0.4 selects
`aria2-next-3.0.4-windows-x86_64.exe` alone among eight assets, and a
3.0.3-only listing selects the 3.0.3 file with `digest_source=both`.
`install-rootless --install` placed that 3.0.3 file under the 3.0.3 resolution
as `both`, sha256 `b2f2f7a9...d2be9465`.

⚠ **What differs from the Linux half, each for a reason in the adapter's
header:**

- RPC is the adapter's own HTTP client on loopback with no proxy, not `curl`.
  So `start` and `stop` run no program beside the build and `bit-check`, and
  `describe` names no `helpers=`.
- `stop` deletes the token on every path out. It sends `aria2.shutdown`, waits
  the Linux half's ten seconds, then kills, because Windows has no SIGTERM for
  a console process in another process group. `stop.txt` says which of the two
  ended it.
- There is no PATH lookup, because Scoop puts a shim named `aria2-next.exe` on
  PATH. The build is the one version directory under
  `apps\aria2-next`, never `current` and never the shim.
- The version reading also requires `aria2` on the first line and a
  version-shaped field after `version`.

⭐ **The Windows build's options were read rather than run.** Every option key
the Linux 3.0.3 build reported over RPC on the session host, 104 of them, is a
string in the Windows 3.0.3 executable. Its discovery-family strings are the
same seven: `bt-enable-lpd`, `bt-lpd-interface`, `bt-port-mapping`,
`ed2k-port-mapping`, `enable-dht`, `enable-dht6` and `enable-peer-exchange`.
The Linux build reported exactly the thirteen judged-harmless options as
`true`. ⚠ The Windows build's own defaults are not measured. A default that
differs and is reported `true` refuses at `start` by name.

⛔ **What needs a runner, in the order to read a dispatch:**

1. Whether `resolve-release` verifies the vendor's checksums document on
   Windows, through `sha256sum` or `certutil`. The release lane's
   `release/resolution.txt` should say `digest_source=both`.
2. Whether Scoop's installer runs under `-RunAsAdmin` into
   `%USERPROFILE%\scoop` and clones its bucket with git. The package lane's
   `install.log` and `scoop-installer.txt` show it, and `scoop-index.txt` names
   the bucket commit and the manifest's version.
3. Whether `scoop install main/aria2-next --no-update-scoop` places one version
   directory whose `aria2-next.exe` digests to the manifest's hash.
   `scoop-index.txt` records `placed_sha256` beside `index_sha256`.
4. Whether `aria2-next.exe --version` answers on both routes. `version.err`
   beside each install record carries the whole answer.
5. Whether the Windows build's options pass the shared read-back:
   `client/rpc-options.json`. A refusal names the option.
6. Whether `aria2.shutdown` ends it within ten seconds, so it announces
   `stopped`. `client/stop.txt` says `rpc-shutdown` or `killed`, and no
   `client/rpc-token` may ship.
7. The capture itself: announces from `-qB5230-` with `aria2-next.exe` and
   `bit-check.exe` under the firewall rules.
### ⭐ 3.0.4 is published on Linux and Windows from one run. 2026-10-16

The vendor released 3.0.4 on 2026-10-15, so no route resolves 3.0.3 any more.

- **Capture:** capture-client run 38838967241 on `05d0d21` captured all four
  lanes:
  - Linux release and source;
  - Windows release (the vendor's executable);
  - Windows package (Scoop `main/aria2-next`).
- **Assembly and publish:** assemble-bundle run 38841851826 and publish-data
  run 38841951112.
- **On `data` at `4221e28`:** 14 of 14 records valid and admitted to the
  published view.

