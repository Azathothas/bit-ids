# Supply chain

What this repository executes that it did not write, how each of those is
pinned, and what refuses an unpinned one.

The threat is narrow and specific. This project publishes measurements taken by
code running unattended in CI, and a reader trusts a published profile because
they trust the observer that produced it. A dependency that can change without
a reviewed commit changes the observer without changing the record of what the
observer was.

## The three layers

| layer | pinned by | enforced by |
| --- | --- | --- |
| Rust toolchain | [`../rust-toolchain.toml`](../rust-toolchain.toml), an exact channel | `cargo` invoked through `rustup` selects that toolchain and installs it if absent |
| crates | [`../Cargo.lock`](../Cargo.lock), committed | `--locked` in CI, plus the source and checksum test in `check-project` |
| actions | a 40-character commit in [`../.github/workflows/`](../.github/workflows/) and any composite action under `.github/actions/` | the pin test in `check-project`, and `check-remote-items` against GitHub |
| CI tools fetched at run time | a module version, verified against Go's checksum database | the Go toolchain, which refuses a module whose contents no longer match |

⚠ **There is no separate register of any of these.** The lockfile is the crate
inventory and the workflow is the action inventory. A second file listing the
same commits would be a value in two places with nothing checking that they
agree, which is the drift these rules exist to prevent.

### The fourth layer is the loose one

The Linux lane installs `shfmt` with `go install`, at the version
[`ci.yml`](../.github/workflows/ci.yml) names. That is a version tag, not a
commit, and the action-pin rule does not see it: that rule reads `uses:` lines,
not `run:` scripts.

⚠ **The version is not repeated here, and that is the point.** It lives in the
workflow and in [`provision.sh`](../scripts/doctor/provision.sh), which is one
copy more than ideal and exactly two more than nothing - so `check-project`
compares them, over every workflow and script, and refuses a pair that has
drifted. A third copy in this sentence would be a value nothing compares.

⭐ **It is a tag that cannot move underneath us, for a reason specific to Go.**
`go install module@version` verifies the downloaded module against the public
checksum database, so a re-tagged release fails to install rather than
installing different code. That is a real immutability mechanism and not the
same as a git tag, which can be force-pushed and resolves to whatever it points
at today.

⚠ **The reasoning, not the form, is what makes it safe.** Do not copy
`@v1.2.3` to a fetch that has no equivalent verification: a `curl | sh` of a
release asset at a tag has none of this. If the checksum database is disabled
for a runner, through `GONOSUMDB`, `GOFLAGS` or `GOPRIVATE`, this layer loses
its guarantee and the tool needs pinning some other way.

## Crates

Every package in the lockfile either has no `source`, meaning it is a member of
this workspace, or comes from the crates.io registry with a checksum.
`check-project` reads the lockfile and refuses anything else, so a `git` or
`path` dependency is caught in the file cargo generates whatever the manifest
said. A second test names the manifest line as well, because a report that
points at the file somebody edited is easier to act on than one that points at
generated output.

⛔ **A dependency is a decision, not a detail.** `SCHEMA-01` added the first
three and recorded why in its entry: a hand-written JSON reader would have been
a new silent-corruption surface in the layer that must not corrupt. The next
one gets the same argument in its own entry, including what was considered and
rejected.

⭐ `FOUND-03` added a second workspace member, `bit-ids-wire`, and **no** new
third-party crate: its lockfile diff is the member and nothing else. A
`BitTorrent` codec crate off the registry would have been the obvious shortcut
and the wrong one here, because the decode is the measurement. A dependency that
normalises what it reads, by sorting dictionary keys, folding header case or
percent-decoding on arrival, destroys the evidence this project publishes, and
that behaviour is the sensible default for every library written for a client
rather than for an observer.

`OBS-01` and the observers that followed it added two more workspace members,
`bit-ids-lab` and `bit-ids-probe`, and no third-party crate: the lab binds
sockets with `std::net`, one door aside since `OBS-04`, and one thread per
endpoint rather than an async runtime,
which would have been a dependency tree an order of magnitude larger than
everything here, in the component that must stay reviewable.

⛔ **`OBS-04` added `socket2` 0.6.5 on 2026-10-14, for one door.** A build keyed
its peers on their address: `aria2-next` 3.0.2 answered one of two connections
from `127.0.0.1` and both when the second left from `127.0.0.2`; run 36's pair,
the only one, was refused by `E-PUB-03` over one `peer_wire` sample per
capture. Choosing
the address a connection leaves from means binding before connecting, which
`std::net` cannot do. ⭐ **Measured rather than argued: the lockfile diff is three
packages**, `socket2` from the rust-lang organisation and `windows-sys` 0.61.2
and `windows-link` 0.2.1 from Microsoft's `windows-rs`, which `socket2` names
for its Windows half; `libc` was already here. ⛔ The rejected routes: calling
`bind(2)` by hand needs `unsafe`, which this workspace forbids; a tracker answer
offering two peer addresses for the build to dial would have changed which
side speaks first, and `client-capture`'s own documentation argues the side that
dials is the stronger measurement; and a second address family, `::1`, would have changed
two conditions at once. ⚠ The crate is used in `bind::dial_from` alone,
and `tests/lab_supervisor.rs` greps every other module for `Socket::new`.

`OBS-08` added `sha1` 0.11.0, the fourth third-party crate and the first since
`SCHEMA-01`. The info hash is SHA-1 by BEP 3 and this project does not get to
choose otherwise. ⭐ **Measured rather than argued: the lockfile diff is one
package.** `sha2` 0.11.0 is already here from the same RustCrypto release train,
so `digest`, `block-buffer`, `crypto-common`, `cpufeatures`, `typenum` and
`hybrid-array` were all present and no new maintainer joined the trust set. The
rejected alternative was implementing SHA-1 in this repository: new unaudited
cryptographic code in the component that decides whether two captures are of the
same torrent, to save one small crate. ⚠ The crate is checked against RFC 3174's
own vectors rather than trusted, in
[`../crates/bit-ids-lab/src/torrent.rs`](../crates/bit-ids-lab/src/torrent.rs).

⭐ `FOUND-04` put every one of these packages in a register with the licence
read out of its own manifest at the version the lockfile pins, beside every
catalogue target. ⚠ A row records who answered, because the answers are not
equally strong, and a licence nobody established is `unverified` rather than
guessed. `catalogue/licences.toml` is the register and `check-licences` is the
pair that holds it to the lockfile.

`PUB-03` added five published renderings and **no** third-party crate. JSON,
JSONL and CSV need no encoder the crate does not already have, and the CBOR one
is written here for the reason the wire codecs are: the published bytes are what
a digest names. ⚠ The subset is small only because the input is a JSON document,
so there are no tags, no indefinite lengths and no floats to get wrong. ⭐ It is
checked against `cbor2` 6.1.4 rather than trusted, and not by a round trip: that
reader's own canonical encoding of what it read is byte-identical to ours, which
is two implementations of RFC 8949 section 4.2.1 agreeing rather than one
agreeing with itself.

⛔ **`PUB-05` took `rusqlite` 0.37.0 with `bundled` and `serialize`, and it is
the largest dependency this repository has taken.** The rejected route was
writing the file format here, which means new unaudited code producing B-tree
pages and varints in the component that publishes evidence - the argument
`OBS-08` rejected for SHA-1, and stronger here because the format is larger.
⚠ `unsafe_code = "forbid"` is a lint on this workspace's own crates and a
dependency compiles under its own, so nothing was relaxed to take this; what was
accepted is a vendored C library and a build script.

⭐ **The version was chosen by measuring rather than by taking the newest.**
`0.40.2` resolves 13 further packages that `0.37.0` does not - `wasm-bindgen`,
`js-sys`, `sqlite-wasm-rs`, `rsqlite-vfs` and what they pull - which is a
WebAssembly stack this project never builds for. Fourteen new packages against
twenty-seven, for the same API and the same bundled library.

⚠ **`bundled` is the point rather than a convenience.** The published bytes are
what a digest names, so a database written against whatever SQLite a host
happens to carry would make the release depend on the builder's machine.
`serialize` is what lets the file be produced in memory, which the crate needs
because it owns no filesystem.

⛔ **`libsqlite3-sys` vendors SQLite itself and the register says so.** The
crate is MIT; the amalgamation it carries states that its author disclaims
copyright, read out of the package's own `sqlite3.c` rather than from a summary
of it. That row is the one place in the register where the crate's licence is
not the whole answer.

`OBS-09` added `serde_json` 1.0.151 to `bit-ids-lab` as a **dev**-dependency, and
no new package: `bit-ids` already depends on it, so the lockfile diff is one line
naming the existing crate against another member. ⭐ **It is a dev-dependency on
purpose.** The acceptance suite needs a JSON reader to splice the bundle's rows
into the golden manifest and profile; the crate itself does not, because a
transcript is serialised by hand. The field order, the hex case and the exact
spacing are the bytes a digest names, and a derive that reordered a field on a
dependency bump would change every artifact's digest with nothing saying why.
The rejected alternative was serialising through the derive and accepting that
the on-disk form of published evidence is a dependency's choice.

⚠ **`check-no-secrets --public` refuses a bare run of long hex, and a published
test vector is one.** Forty lowercase hex digits is exactly what a token looks
like, so the rule is right to fire. It was narrowed rather than switched off, per
[`security/secrets.md`](security/secrets.md): the exclusion is anchored to a
constant named for its RFC and to exactly forty digits, and it was proven with a
credential beside an allowed vector on one line, the same value under another
name, and a 64-digit value under the allowed name, all still reported.

## Actions

A pin is a 40-character commit followed by a comment naming the tag it was:

```text
uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
```

⛔ **Both halves are required.** `check-project` refuses a tag, a branch, an
abbreviated commit and a bare commit with no comment. The comment is not
decoration: `check-remote-items`, in
[`../tools/check/remoteitems.go`](../tools/check/remoteitems.go), resolves it
against the tag it names and refuses a pin whose comment has drifted from the
commit it labels, so a pin without a comment is a pin that check can never
examine.

⚠ **A rule written as a denylist of the floating forms somebody thought of is
not a rule.** This one named `main`, `master` and `vN.N.N`, and a branch called
anything else, or a seven-character commit, passed it. `FOUND-02` inverted it
to an allowlist of the forms that are immutable by construction, so a shape
nobody anticipated fails closed.

A local action, written `./path`, is this repository and is reviewed with
everything else. A container is pinned by digest rather than by tag.

## Updating

Dependabot proposes crate and action updates weekly, grouped, per
[`../.github/dependabot.yml`](../.github/dependabot.yml). A proposal is a
claim, and the procedure is what turns it into a reviewed change.

⛔ **One crate is held rather than proposed.** `rusqlite` stays at 0.37 by an
operator decision recorded in [`../TODO/PROGRESS.md`](../TODO/PROGRESS.md) under
*Settled decisions*, and that file's `ignore` is where the decision is enforced:
without it, a declined bump came back as a new pull request after its
predecessor closed. ⚠ For a `0.y` crate dependabot's cargo rules call any change
of `y` major, so the rule ignores every version from 0.38 on; a patch inside 0.37
is still proposed, and the rule cannot silence a security update, which
dependabot filters by explicit versions alone. Moving past 0.37 is a change somebody
makes on purpose, with its locked packages registered, rather than a proposal.

1. **Read what the item asserts, then check it.** Run
   `go run . check-remote-items` from `tools/check`, and read its report:
   `go run` exits 1 for a refusal and for a check that could not run alike,
   where the binary's own code tells them apart. For every action pin a pull
   request adds, it verifies that the commit exists in the repository the ref
   names, that the tag in the comment resolves to that commit - following an
   annotated tag to the commit it names - and that the runtime the pinned commit
   declares is not deprecated, and it names the latest release when the pin
   proposes another. ⚠ A bump that resolves cleanly can still be two majors
   behind; that has happened here. ⚠ This step used to say it also verified
   that the tag is a published release. The shell half's header claimed that
   and its code never asked it: the one release it reads is the latest, and a
   pin labelled with another tag is reported for a reading, not refused.
2. **Regenerate the lockfile with the pinned toolchain**, never by hand.
   `cargo update -p NAME --precise VERSION` for one crate, `cargo update` for
   the group.
3. **Run the gate.** `sh scripts/common/check-gate.sh`. A crate that changes
   observed behaviour is a finding, not a formality.
4. **Write the changelog entry**, naming the record and saying whether it
   deployed, per [`conventions/docs.md`](conventions/docs.md).
5. **Merge is the operator's.** Nothing here merges, closes, comments on or
   approves an item. [`security/remote-ops.md`](security/remote-ops.md).

⛔ **Never take the item's word for the fact it asserts.** A bot's title says
what it believes it is doing, and by the hundredth bump nobody is looking. That
is the whole reason `check-remote-items` exists.

## What is not covered yet

- Artifact attestations, SBOMs and signed release metadata are `CI-04`. Nothing
  here binds a released asset to the workflow and commit that built it.
- Licence and redistribution disposition per dependency is `FOUND-04`.
- `check-remote-items` verifies pins a pull request proposes. It does not
  re-verify the pins already in the tree, so a tag deleted or moved upstream
  after a merge is not currently detected. `CI-04` owns closing that.
- ⚠ `check-remote-items` needs `gh` **authenticated**: installing the binary is
  not enough, and on 2026-09-04 a session host with `gh` 2.63.2 present had its
  environment token rejected by `gh auth status`, while the other GitHub route
  that host had was scoped to this repository alone and so could not resolve a
  pin in `actions/checkout`. Both CI lanes run it on every push. ⭐ Until
  2026-09-29 this page described what it verifies from its own documentation,
  with no run behind it, and one claim was false; fifteen cases in
  `check-bitcheck` now back each sentence of step 1 above, served from a stub
  `gh` and `curl` on every host.
- The `run:` scripts in a workflow are read by nobody. The pin test covers
  `uses:` lines; a tool fetched by a shell line is only as pinned as the fetch
  it uses, which is why the fourth layer above needed an argument rather than
  a rule.
