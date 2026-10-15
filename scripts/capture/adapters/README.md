# Capture adapters

One file per target, and the only thing in this repository that knows how a
particular product is installed, asked its version, started and stopped.

⭐ **The adapters are Go since 2026-10-15, `CI-11`, and this page is still
their contract.** Each is a `bit-check` command, `bit-check adapter <name>
<subcommand> [args...]`, in its own file:

| target | adapter |
| --- | --- |
| `aria2` | [`../../../tools/check/adapter_aria2.go`](../../../tools/check/adapter_aria2.go) |
| `aria2-next` | [`../../../tools/check/adapter_aria2-next.go`](../../../tools/check/adapter_aria2-next.go) |
| `qbittorrent` | [`../../../tools/check/adapter_qbittorrent.go`](../../../tools/check/adapter_qbittorrent.go), with a Windows half registered under the same name |
| `transmission` | [`../../../tools/check/adapter_transmission.go`](../../../tools/check/adapter_transmission.go), with a Windows half registered under the same name |

The four Linux adapters share their mechanics, and none of their decisions, in
[`../../../tools/check/adapterkit.go`](../../../tools/check/adapterkit.go). The
door every caller reaches an adapter through is
[`../../../tools/check/adapter.go`](../../../tools/check/adapter.go): a
registered name runs this binary as a child process, and anything else is run
as `sh <path>`, which is what the harnesses' stub adapters still are. ⛔ This
directory keeps no adapter; it keeps the contract, which did not change when
the language did.

⛔ **An adapter is the seam and never the boundary.** The guards that decide
whether a capture may happen at all live in
`bit-check assert-disposable`, in [`../../../tools/check/assertdisposable.go`](../../../tools/check/assertdisposable.go),
and are run by
`bit-check capture-client` before an adapter is asked to
start anything. An adapter that tried to check the host would be a second
answer to a question one place already answers.

## The contract

Five subcommands. Each reads its arguments positionally, prints to stdout, and
returns 0 for done, 1 for refused and 2 for could-not-run.

| subcommand | when | must |
| --- | --- | --- |
| `describe` | any time | print `target=<slug>` and `kind=stock\|stub`, one per line, plus `binary=<path>` when one is installed and the release-route declarations below when it has one |
| `install <route> <workdir>` | ⛔ **before containment**, with the network in reach | install the target through that route alone |
| `version` | ⛔ **before the route runs, again after it, and again under containment** | ⛔ ask the **installed executable** and print exactly what it answered |
| `start <torrent> <workdir> <peer-port>` | under containment | launch the build on that torrent and return; the build keeps running |
| `stop <workdir>` | under containment | stop whatever `start` launched |

⛔ **`kind` is a declaration the attestation copies rather than assumes.** A
`stub` adapter drives every guard `capture-client` has and is not a product, so
a run it drove writes `stock_client=false`. A field that said otherwise would be
the one sentence in the record nothing backs.

## ⛔ `install` says how it packaged what it installed

**A successful `install <route> <workdir>` writes `<workdir>/package`**, one
slug naming the form the build arrived in: `deb`, `appimage`, `elf-binary`.
`install-client` copies it into the install record and refuses a route that
installed and did not say, or that said something which is not a slug.

⛔ **It is per ROUTE, not per target, because the routes genuinely differ.**
`qbittorrent`'s package route installs a `.deb` and its release route installs
an AppImage - a self-mounting archive carrying its own runtime - and a record
calling both `elf-binary` would say the two routes delivered one form when they
did not. Each adapter maps its own routes in `package_format`.

⛔ **Nothing recorded this until 2026-09-09 and the gap was invisible.**
`Build.package` is part of the identity tuple `store::StoreKey` derives a path
from, so a record filed without it cannot be filed at all, and one filed under a
value this project guessed would file two packagings of a version at one name -
the non-injective layout `store.rs` refuses at length. It was found by
`assemble-capture` refusing both of `capture-client` run 14's lanes.

⚠ **The key is `package` because that is the record's own vocabulary**, and the
route also called `package` is the host's package-manager route. They are
different things, and an adapter is the one file where both appear.

⛔ **`version` asks the build.** A version read from a filename, a package index
or the route that installed it is not the build speaking, which is the rule
`E-ACQ-10` states for a record and `capture-client` restates for a run. An
adapter that cannot ask exits 2, and the capture is *could not run* rather than
a fixture.

⛔ **`version` prints the parsed field on stdout and the build's WHOLE answer on
stderr.** Two routes can install builds that report one version and are not one
build: measured on 2026-09-08, Ubuntu's aria2 1.37.0 and a 1.37.0 compiled from
the vendor's own release tarball answer the same version and enable different
features. ⚠ Features are what a build does on the wire, so a record holding only
the version has dropped the evidence that would have separated them. ⭐ stderr
costs nothing to keep: `install-client` writes it to `version.err` beside the
install record and `capture-client` appends it to `adapter.err` inside the
evidence bundle.

⛔ **So `version` is a pure read and is called when the target may not be there
at all.** `install-client` asks it before the route runs, which is how a route
that installed nothing is detected, and a target that is absent then is the
ordinary case rather than an error: exit 2, and the caller records that the host
had nothing. ⚠ An adapter whose `version` started a daemon, wrote a profile or
assumed containment would be changing the host at the one moment the caller is
trying to observe it unchanged. Every shipped Linux adapter runs `<binary>
--version` and nothing else. ⚠ `qbittorrent`'s Windows half cannot: its build
has no `--version`, so *A Windows half, in Go* below says what it does instead.

⚠ **That is also the limit of "driven over RPC".** `aria2-next` is driven over
its own RPC interface wherever RPC answers the same question, and `version` is
the case where it cannot: `install-client` asks before the route runs, when
nothing is installed and no daemon exists to ask. RPC answers that question
*better* once a build is running - `aria2.getVersion` returns a structured
`version` field rather than a line of prose - and it cannot answer it at all at
the moment the caller needs it.

⛔ **`install` is never called under containment.** By the time a capture runs
there is no route off the host, so an install that needed one would have to
restore egress on the machine that exists to have none.
`bit-check install-client`, in [`../../../tools/check/installclient.go`](../../../tools/check/installclient.go), is
the caller, and it runs in an earlier workflow step.

## Every call is bounded, and the bound is the caller's

⛔ **An adapter shells out to a product this project did not write, and a product
that waits on a question waits forever.** Measured on 2026-09-08, client capture
run 1: two of its three jobs sat in the install step for over half an hour and
reported nothing at all, because a hung install is indistinguishable from a slow
one until the job's own timeout kills the runner and takes the log with it.

⭐ **So the callers bound every subcommand rather than each adapter bounding
itself.** A convention in each adapter is a bound the next adapter forgets; the
callers are the one place every adapter passes through. `124` is coreutils'
verdict for *it never answered*, and it is reported as its own refusal because
the fix differs from a route that said no.

⛔ **And stdin is `/dev/null` on every call.** A product that asks a question
gets end-of-file rather than a wait. ⚠ That is a second control and not the same
one: it catches a prompt, the time limit catches everything else, and
[`../check-capture-client.sh`](../check-capture-client.sh) proves each with its
own case because a stub blocking on `read` would demonstrate only the redirect.

⚠ **An adapter still owns the switches its own product needs.** Ubuntu's
`needrestart` opens a dialog that `DEBIAN_FRONTEND` does not suppress, and
`qbittorrent-nox` asks for its legal notice to be confirmed - on `--version` as
well as on a run, which is a control on one of two paths into the same product.
⭐ And since 2026-10-15 `qbittorrent` runs the GUI build, the only one either
5.2.4 AppImage carries, so the adapter starts its own Xvfb for `version` and for
each session, and asks the build not to open its add-torrent dialog.

## ⭐ A Windows half, in Go

A Windows runner has no `sh`, so an adapter's Windows half is Go, registered
by name and run as `bit-check adapter <name> <subcommand>`, under the same
contract. `qbittorrent` has one,
[`../../../tools/check/adapter_qbittorrent_windows.go`](../../../tools/check/adapter_qbittorrent_windows.go),
and so does `transmission`,
[`../../../tools/check/adapter_transmission_windows.go`](../../../tools/check/adapter_transmission_windows.go).
Three things in the first differ from Linux, and each is a fact of the build:

- ⛔ **`version` starts the build.** The Windows GUI build compiles
  `--version` out, and on Windows the flag opens a modal *Bad command line*
  box. So `version` runs the build in a throwaway profile with every surface
  off and asks `/api/v2/app/version` on loopback. That is still the
  executable answering, and nothing else is read. It is the one exception to
  *a pure read* above, and it ends what it started on every path out.
- **`stop` asks the build to shut down over the same Web UI**, which is
  Windows' graceful exit, and kills it only after a grace period. `stop.txt`
  in the workdir says which of the two happened. The Web UI's password hash,
  of a password nobody keeps, is scrubbed from the profile before the workdir
  ships.
- **Two more `describe` keys.** `routes=` names the routes this half has.
  `helpers=` names, `|`-separated and absolute, every executable other than
  the build that `start` or `stop` runs. ⛔ The second is what the firewall
  blocks beside the build: `bit-check capture-client --programs` reads it, and
  a rule covers only the executable it names.

⭐ **`transmission`'s Windows half differs from Linux less**, because its build
has a `--version` on Windows and is driven by the same `transmission-remote`.
`version` is the pure read the contract describes. `stop` asks the daemon to
exit with `transmission-remote --exit`, the RPC `session-close`, and kills it
only after a grace period; `stop.txt` says which. The scrub of the RPC verifier
is Go rather than `jq`, and `helpers=` names `transmission-remote.exe`. Its
release route lays the vendor's MSI out as an administrative image, `msiexec
/a`, which runs no install; its package route is winget, with the MSI's
`Daemon` and `CmdTools` features named because a default install carries
neither. `CLIENT-06` in [`../../../TODO/clients.md`](../../../TODO/clients.md)
carries the measurements.

## ⛔ The workdir IS the evidence bundle

**Anything an adapter writes into the workdir is uploaded as an artifact.**
`capture-client` calls `stop` and then packs that directory, so a file left
there ships.

⚠ **Which makes rule 12 an adapter rule and not only a repository one.**
Measured on capture-client run 11: `aria2-next` generated a per-run JSON-RPC
token, wrote it to `<workdir>/rpc-token` so `stop` could authenticate its
shutdown, and the token was then inside the uploaded bundle as
`client/rpc-token`. Nothing between those two steps was wrong on its own. ⛔ A
short-lived token, for a loopback port, on a host about to be destroyed, is a
weak secret and rule 12 does not grade them.

⭐ **So an adapter that must persist a secret deletes it in `stop`, before
anything packs the directory** - and writes it at mode `0600` in the
meantime. An adapter that needs no secret should not invent one.

⛔ **AND `capture-client` REFUSES THE RUN IF ONE IS STILL THERE, since
2026-09-10.** The paragraph above is a rule an adapter has to remember, which
needs `stop` to be reached and to succeed - and a non-zero `stop` is recorded in
the attestation rather than refused. The runner scans what it is about to ship
for a credential-shaped **name** and for a secret-shaped **member** inside any
text document, and refuses either. ⚠ The second is the one an adapter is least
likely to think about: it is not a file the adapter chose to write, it is the
product's own answer - an options dump naming its own `rpc-secret` - landing in a
file called nothing in particular.

## What an adapter must switch off

⛔ **Public peer discovery, on every route into it.** A capture host has no
default route, so a DHT bootstrap or a tracker on the internet fails rather than
leaks - but a *link-local* announce does not, and neither does anything the host
can still reach. Every adapter disables DHT, peer exchange and local peer
discovery explicitly, and the settings it wrote are part of what the run
records.

⚠ **Three switches, because they are three surfaces.** `OBS-06` calls them
adjacent for exactly this reason: each carries identity and each names a
destination of its own. An adapter that turned off DHT alone would leave the
other two on, which is the one-gated-door shape
[`../../../docs/methodology/reviews.md`](../../../docs/methodology/reviews.md)
calls the most recurring hole there is.

⛔ **Three is this contract's floor and not every product's ceiling, and that
was measured rather than reasoned.** `aria2-next` has a FOURTH:
`bt-port-mapping`, "Enable UPnP and NAT-PMP port mapping", defaults to **true**.
Driven on 2026-09-09 with all three documented switches off, the build was still
bound to **UDP 1900** - SSDP multicast - read out of `/proc` rather than out of
its own report. ⚠ A contained host has no default route, so a tracker or a DHT
bootstrap fails; a multicast to `239.255.255.250` is link-local and does not.

⭐ **So an adapter enumerates its own product's surfaces, and the way to find
them is to run the build and read its sockets.** The help text names the switch
only if you already suspect it; the socket table names it whether or not you do.
⚠ Whether `aria2`, `transmission` and `qbittorrent` have surfaces of their own
that this list does not cover is an open question and not a claim - nothing has
read their sockets this way.

⛔ **And a product's surfaces move with its version, which a socket table read
once does not see.** `aria2-next` 3.0.0 added `ed2k-port-mapping`, UPnP for its
ED2K ports, which 3.0.2 has on by default, and 3.0.2 no longer reports
`enable-dht6`. The
read-back refused the first 3.0.2 capture over the missing key, and the options
dump that came back named the new switch. ⭐ So `adapter_aria2-next.go` holds every
switch of the four families its build reports to `false`, named in the file or
not, beside the list it passes. `CLIENT-14` in
[`../../../TODO/clients.md`](../../../TODO/clients.md) carries the measurement.

## The routes

⛔ **Two independent routes per target, and `E-ACQ-07`/`E-ACQ-08` are what
refuse a pair that shares a resolver or a delivery mechanism.** `package` is the
host's own package manager, `release` is the vendor's published artifact, and
`source` is a build of the resolved tag. Two package aliases pointing at one
index are one route.

⭐ **`source` was added on 2026-09-09 because a target had no package.** No index
carries `aria2-next`, and the dangerous fallback is that the obvious one
SUCCEEDS: `apt-get install aria2` exits 0 having installed a different product at
a different version. ⚠ The pairing that worked is `release` against `source` -
the vendor's published binary against a build of the same tag - and run 14
measured both at 2.7.5 with different digests.

⛔ **A `source` route resolves its TAG from the same resolution the `release`
route reads its URL from.** Two resolutions per lane would let a vendor whose
newest release moves between two reads hand the two routes different versions,
which is what absolute 4 forbids; sharing one resolution makes them equal rather
than checking it afterwards.

⚠ **AND `source` BESIDE `release` IS WEAKER INDEPENDENCE THAN `package` BESIDE
`release`.** They differ in resolver and in delivery, which is what `E-ACQ-07`
and `E-ACQ-08` compare, and they do NOT differ in origin: whoever controls that
repository controls both. An adapter that has a real package route should prefer
it, and one that does not should say so where the record can see it.

⚠ **A source build is minutes, not seconds, and the caller's bound has to know.**
`aria2-next` 2.7.5's took 366 seconds on a session host and **496** on a runner,
about 1.35x; on 2026-10-14 a runner reached `v3.0.2`'s compile error in 508
seconds where the session host took 588. So a locally measured build time is not
an estimate of what a runner needs in either direction, and the bound carries
the margin. The workflow raises `BIT_IDS_INSTALL_TIMEOUT` for the lanes whose
route the adapter declares in `compiles`.

### The release route's artifact, declared where the target is known

⛔ **A release route needs an artifact chosen and `describe` is where the target
says which.** Six more keys, printed by `describe` and read by
`bit-check resolve-release`, in [`../../../tools/check/resolve.go`](../../../tools/check/resolve.go).
An adapter with no release route prints none of them, which is an answer: the
caller learns this target has no second route rather than watching a fetch fail
with something that names nothing.

| key | is |
| --- | --- |
| `release_repo` | `owner/name`, the repository whose releases the route resolves through |
| `release_tag_prefix` | the literal stripped before a tag is read as a version, or `-` for none |
| `release_min_components` | the fewest dot-separated components a version of this target has |
| `release_max_components` | the most |
| `release_floor_major` | the first major version the scheme speaks for, or `-` for none: a tag the scheme cannot order whose leading component is a plain integer below it is released as `below_scheme_floor` rather than blocking, and only against a selection at or above it |
| `release_asset` | which artifact of a release is the installable one |
| one of `release_digest_asset` / `release_digest_unpublished` | how this vendor publishes a digest, or the measured reason it publishes none |

⛔ **And one more key, about the install's bound rather than its artifact.**
`compiles` names, comma-separated, the routes whose install runs a build, for
example `compiles=release,source`. `capture-client.yml` reads it before the
install and bounds such a lane as a build: 1500 seconds instead of 420. A
route it does not name is bounded as a download. ⚠ It replaced a bound keyed on
the route's name on 2026-10-14, because Transmission's release asset is a
source tarball. `CLIENT-06`.

⛔ **THE DIGEST DISPOSITION IS DECLARED AND NEVER DEFAULTED, AND EXACTLY ONE OF
THE TWO IS REQUIRED.** `ACQ-06`. An adapter that named neither would install
bytes nothing identified, and a default would mean the day a vendor stopped
publishing digests nothing would say so. `bit-check resolve-release` refuses both an
absent declaration and a doubled one.

⭐ **`release_digest_asset` is an asset pattern like `release_asset`**, selected
out of the same listing by the same call, so one retrieval answers the version,
the artifact and the digest and the record's `listing_sha256` covers all three.
⚠ **`release_digest_unpublished` carries the measured REASON**, not a flag: the
route reads it from this adapter rather than spelling it a second time.

⛔ **AND THIS KEY IS ABOUT THE VENDOR'S CHECKSUMS DOCUMENT, NOT ABOUT DIGESTS.**
That is narrower than it sounds and the difference is measured. A release listing
also carries a per-asset `digest`, published by the party SERVING the bytes, and
`bit-check resolve-release` resolves it out of the same response with no extra
retrieval. An adapter cannot know that by reading a release and does not declare
it; what this key decides is only whether the resolver goes looking for a
checksums FILE.

⭐ **So a resolution reaches one of four dispositions**, and the installer's
report carries which:

| `digest_source` | what was checked |
| --- | --- |
| `both` | the vendor's document AND the listing's digest - two parties, two statements about one artifact |
| `vendor-document` | `sha256sum -c` over a file the vendor uploaded |
| `source-listing` | the digest the hosting index publishes of the bytes it serves |
| `unpublished` | neither exists; the artifact is IDENTIFIED and not verified |

⚠ **Measured through `AGENTS.md` rule 8's route on 2026-09-17**, which is why
these are four values rather than two: `aria2-next` reaches `both`; qBittorrent
and Transmission publish no checksums file and their listings state a digest, so
they reach `source-listing`; and `aria2`'s six assets carry `digest: null`, which
makes it the one target that is genuinely `unpublished`.

⛔ **AN IDENTIFIED ARTIFACT IS NOT A VERIFIED ONE, AND A HOST IS NOT A VENDOR.**
`source-listing` establishes that nothing changed between reading the index and
fetching the artifact, and says nothing that would survive that index being
wrong: the party publishing the digest is the party serving the bytes. A record
naming only "verified" would tell a reader more than happened.

⛔ **The route does not choose between them.** It passes
`--from-resolution <file>` and the installer reads it - four adapters choosing
between four dispositions would be four copies of one decision, which is the
one-gated-door shape `docs/methodology/reviews.md` names.

⛔ **They live here because this file is already the only one that knows how
this product is installed**, and they are on `describe` rather than on a sixth
subcommand because both callers already parse `describe` for `binary=`. A second
door into one answer is the shape this repository's reviews call the most
recurring hole there is.

⭐ **`release_asset` is a pattern with two constructs and `{version}` is the one
that matters.** It expands to the version the resolver selected, `*` matches any
run, and everything else is literal. Measured on 2026-09-09 against the live
listings: qBittorrent's release-5.2.3 publishes **fourteen** assets including two
Linux `AppImage` files differing only by an `_lt20`, so
`qbittorrent-*_x86_64.AppImage` matches both, and aria2's release publishes three
source archives differing only in compression, so `aria2-*.tar.*` matches three.
⛔ **Two matches is a refusal, not a first-match answer**: choosing between them
by the order the source listed them is choosing by a property of the source, and
the record would look identical the month that order changed.

⚠ **A declared artifact is not an installable one.** `transmission` declares its
source tarball and its release route still refuses, because nothing here knows
how to build it. That is the honest split: this project can say which file a
route would fetch and cannot say how to compile it, and declaring the first is
what makes the second a named gap rather than a route that is simply unreachable.

⚠ **A route that installs is not yet a route that agreed.** `ACQ-03`'s
same-version gate compares what each installed build *reports*, which is why
`version` asks the executable rather than the installer.

⛔ **And a route that RAN is not yet a route that INSTALLED.** `install-client`
asks `version` and `describe` once before the route runs and once after, and
records `preexisting_version`, both executables, both digests and `acquired`
beside the version.

⛔ **A version is not an identity, which is why `describe` names the
executable.** `aria2` ships on `ubuntu-24.04` at the same version the vendor
publishes, so a release route there installs a genuinely different build - a
2.8-megabyte self-contained program in place of a 14-kilobyte shim over
`libaria2.so.0`, with a different feature list - and both answer `1.37.0`. ⚠ A
verdict on version strings alone calls that no acquisition at all, which is
measured rather than argued: it is what the first two-route capture would have
recorded. The digest is what makes that branch reachable, and it has its own
harness case.

⚠ **An adapter that names no executable is not refused.** The key is omitted when
nothing is installed, the two digest fields come back empty, and the verdict
falls back to the version - a lower bound rather than a wrong answer.

⚠ **Both halves of that are measured rather than imagined.** `aria2` ships on
the `ubuntu-24.04` image, so `apt-get install aria2` there prints `already the
newest version`, installs nothing and exits 0; client capture runs 3 and 4 wrote
`route=package` over exactly that. And every `release` route here fetches its
artifact into the workdir without making it the executable `binary()` finds, so a
release route run after a package route reports the **package** build's version
as its own. ⛔ Two routes in that state declare two independent resolvers, agree
on the version because it is one binary, and reach `ACQ-03` as `byte_identical` -
the strongest verdict the classification has, over an acquisition that never
happened.

⚠ **`acquired=no` is recorded and not refused.** The build on such a host is
real and its identity is worth capturing; what is not real is the claim that this
route acquired it. The refusal belongs to whatever compares two routes, which is
the same reason `install-client` runs one route per call.
