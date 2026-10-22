# Client matrix

This is the human-readable scope. The machine-readable home is
[`../catalogue/clients.toml`](../catalogue/clients.toml). Candidate routes are
hypotheses for `ACQ-*` work, not claims that the requested version is currently
available there.

## Priority evidence

The August 2026 TorrentAnalytics page was read on 2026-09-04 through the
operator-specified read proxy after the direct fetch route was unavailable. It
reports connection share from DHT discovery followed by TCP handshakes. The
requested families appear as qBittorrent 48.71%, uTorrent 13.41% plus a
separate `uTorrent` row at 3.12%, BitComet 5.35%, libtorrent 3.94% plus
`libTorrent` 2.05%, Transmission 3.19%, Zona 2.66%, Deluge 2.13%, BitTorrent
1.97%, BiglyBT 1.67%, Tixati 1.49%, qBittorrent Enhanced 0.73%, KTorrent 0.62%,
FDM 0.23%, rqbit 0.03% and aria2 0.01%.

Those labels are reproduced as the source presented them; similar-looking rows
are not merged. The page cannot establish which implementation produced an
unknown prefix. Its passive method makes it prioritization evidence only.

Wikipedia's usage-share page, read the same day, has a newest table from March
2020. It corroborates that most named applications have long-standing visible
usage, but it is not current market-share evidence.

Sources:

- <https://torrentanalytics.net/top_client?period=2026-08>
- <https://en.wikipedia.org/wiki/Usage_share_of_BitTorrent_clients>

## Application targets

| id | platforms | licence class | candidate route A | candidate route B | entry |
| --- | --- | --- | --- | --- | --- |
| `qbittorrent` | Linux, Windows | open source | vendor/GitHub-linked release artifact | distro/WinGet/Flatpak package | `CLIENT-01` |
| `qbittorrent-enhanced` | Linux, Windows | open source fork | its own GitHub release ⭐ | source build at the matching tag (Linux), Scoop `extras` (Windows) ⭐ | `CLIENT-02` |
| `utorrent` | Windows | proprietary | vendor installer | WinGet/community package | `CLIENT-03` |
| `bitcomet` | Windows | proprietary | vendor installer | package registry | `CLIENT-04` |
| `aria2` | Linux, Windows | open source | GitHub release | distro/WinGet package | `CLIENT-05` |
| `aria2-next` | Linux, Windows | open source fork | its own GitHub release ⭐ | source build at the matching tag ⭐ | `CLIENT-14` |
| `transmission` | Linux, Windows where supported | open source | upstream release/build | distro/WinGet package | `CLIENT-06` |
| `deluge` | Linux, Windows | open source | upstream/PyPI release | distro/Windows package | `CLIENT-07` |
| `bittorrent` | Windows | proprietary | vendor installer | package registry | `CLIENT-08` |
| `biglybt` | Linux, Windows | open source | its GitHub release: the Linux tarball, the Windows installer with its JRE ⭐ | source build at the matching tag (Linux), winget's NoJRE install with Temurin (Windows) ⭐ | `CLIENT-09` |
| `tixati` | Linux, Windows | proprietary freeware | vendor package | package registry | `CLIENT-10` |
| `ktorrent` | Linux | open source | KDE ships only signed source; Flathub is at 26.04.3 ⛔ | apt needs root or trails; nixpkgs is at 26.08.2 but needs a user-space Nix ⛔ | `CLIENT-11` |
| `fdm` | Linux, Windows | proprietary | vendor package | package registry | `CLIENT-12` |
| `zona` | Windows | proprietary | vendor package | package registry if independently verifiable | `CLIENT-13` |

An entry does not weaken the two-route rule when a candidate disappears. It
records three routes considered, keeps the target open and moves to the next
unblocked client.

⚠ **The `aria2` row names `CLIENT-05` and that is no longer where the first
capture is attempted.** Ten dispatches of that target produced no capture at all,
so by operator direction on 2026-09-09 `CLIENT-14` carries the attempt through a
different upstream, acquired from its own releases and driven over RPC.

⭐ **`aria2-next` has a row now because it has been measured.** The paragraph
here used to say it joins this table "when `CLIENT-14` starts rather than when it
was filed", and on 2026-09-09 it started: the target was listed, fetched,
verified against the vendor's own digests, installed in 1.2 seconds, asked its
version and driven over JSON-RPC. `CLIENT-14` carries every measurement.

⭐ **BOTH OF ITS ROUTES ARE MEASURED, WHICH NO OTHER ROW CAN SAY.** The ⭐ marks
them: `capture-client` run 14 acquired this target through both on two hosts, and
both installed **2.7.5** with different binary digests. ⚠ Every other row's two
columns are candidates - that is what this table's columns are for - and these
two are not.
⛔ **It took a source build because no package index carries this fork**, which
was measured rather than assumed, and because the obvious fallback succeeds:
`apt-get install aria2` exits 0 having installed a different product.
⚠ **That pairing is weaker independence than a package index against a vendor
release** - different resolver and delivery, same origin - and `CLIENT-14` says
so. ⛔ Two routes make a record writable and not publishable: `E-PUB-02` keeps a
measured field provisional while one connector could see it, and both lanes used
one.

⭐ **This table IS pinned to the catalogue in both directions, since
2026-09-10.** ⚠ It was not when the claim was first made, and the correction
that stood here said so: `check-project` carried a **list of seventeen ids** and
asked that each appear in the catalogue *and* here, so a target added to the
catalogue and forgotten here was caught by nothing, in either half. ⛔ The list
is gone rather than extended, because a list is a value in two places and
extending it only resets the clock. Both halves derive the set from
`catalogue/clients.toml`, compare it against the ids in this table's own rows,
and refuse a difference either way - and refuse a set too small to be real,
because two empty sets agree perfectly.

## Library targets

| id | harness | route A | route B | entry |
| --- | --- | --- | --- | --- |
| `libtorrent` | minimal Rust/C++ or Python-binding client pinned to one stable libtorrent | package registry/system package | matching GitHub tag | `ENGINE-01` |
| `anacrolix-torrent` | this tree's `tools/harness/anacrolix-torrent/`, the library's default config, pinned at the stable tag ⭐ | proxy.golang.org, checked by sum.golang.org, Linux and Windows ⭐ | the tag from GitHub with every module from its own origin, `GOPROXY=direct`, Linux and Windows ⭐ | `ENGINE-02` |
| `rqbit` | the upstream `rqbit` command, librqbit's own wrapper, at the stable tag ⭐ | crates.io `rqbit`, `cargo install --locked`, Linux and Windows ⭐ | the vendor's GitHub release binary, Linux and Windows ⭐ | `ENGINE-03` |

Library harnesses are not allowed to choose an identity value for convenience.
The profile records both library version and harness commit, and the observer
measures what the built harness actually emits.

⭐ **`rqbit`'s two routes are measured, and its harness is upstream's.** On
2026-10-21 both Linux routes installed 9.0.1 rootless. Both build the crate's
default features, `default-tls`, `postgres`, `webui` and `prometheus`, which
are what the vendor's own workflows build. Both use the vendor's `panic =
"abort"` profile. So no harness of this project's own is committed. ⚠ **The
record has no field for the library version or the harness commit.**
`Target.kind = library` is the harness label it carries. The library version
the running build reports and cargo's install record are kept as evidence
files beside the capture. Nothing a run writes names the tag's commit.
`ENGINE-03` records both gaps.

⭐ **`anacrolix-torrent`'s harness is this tree's, and its two routes build
one executable.** A module version installs nothing, so the harness is a
module of its own whose `go.mod` and `go.sum` pin v1.61.0, every dependency
and Go 1.27.3. On 2026-10-22 the module proxy and the tag, each with its own
copy of every module, built the same executable byte for byte with the same
build information, and both captures assembled `build_equivalent`. ⚠ The same
gap as rqbit's: the record's version is the library's, and the harness's
digest, the build information and the tag's commit are evidence files beside
the capture. `ENGINE-02` records it.

## Platform rule

Capture every platform a stable upstream release genuinely supports and the
project can acquire twice. Windows-only software stays Windows-only. A missing
Linux row is not filled by Wine unless Wine itself becomes a separately named
platform/packaging dimension.
