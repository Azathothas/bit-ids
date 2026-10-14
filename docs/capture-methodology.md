# Capture methodology

## What counts

A measurement comes from network bytes emitted by a running, exact stable
build in an isolated fixture. The subject's source, logs, UI, package metadata
and self-description are supporting evidence only.

The observer must create the condition that exposes the field, preserve the
bytes, parse them, and exit non-zero when an expectation is violated. Every
quoted result must be reproducible from a committed script and fixture.

## Two live connectors

Every run uses:

1. the project-owned Rust active observer; and
2. an independent implementation that overlaps the observation, selected from
   a stock peer's machine-readable report or a raw packet capture.

The two parsers must not share the code that interprets the field. Sharing the
same packet bytes is acceptable for a packet-capture connector; sharing the
same parser is not independent corroboration.

⭐ **The connector that exists is exactly the first of those, and this paragraph
is what permits it.** `scripts/capture/connectors/cpython-stdlib.py` reads the
bytes the lab already recorded and interprets them with implementations this
project did not write. ⛔ **So it corroborates the READING and not the
recording**: a lab that wrote down the wrong bytes would be agreed with, which is
the class only a connector putting its own bytes on the wire can catch.

If connector two cannot observe a field, that field stays provisional until a
second route exists. A profile is not made complete by copying a source-code
constant into the gap.

## Standard run

1. Resolve the newest stable version independently from every configured
   acquisition route.
2. Require at least two routes to agree on the version.
3. Acquire and verify both artifacts. Record URL, timestamp, digest, signature
   status and package metadata.
4. Install route A on a disposable host and ask the running executable for its
   version. Repeat with route B on an equivalent clean host. ⛔ Ask before the
   route runs too: a host that already ships the target makes the route a no-op
   that exits 0, and the version it then reports is not one that route acquired.
5. Require exact normalized version equality, **and require that each route
   acquired the build it reports**. Two no-op routes agree perfectly, for the
   most trivial reason available. Keep distinct executable digests as packaging
   evidence.
6. Generate a tiny deterministic torrent whose payload and metainfo digests are
   recorded. Start only local tracker and peer endpoints.
7. Launch the target with a throwaway configuration. Disable external DHT,
   discovery and trackers unless that surface is the one under a bounded test.
8. Run the Rust observer and independent connector together. Capture multiple
   sessions and torrents to identify refresh lifetime and random fields.
9. Run positive controls through both connectors, then parse and correlate.
10. Scan/redact forbidden environmental data, validate the profile, and upload
    the evidence bundle. A separate job performs publication checks.

## Sample policy

One observation establishes bytes for one connection. It does not establish a
lifetime or randomness rule. Each target needs enough separately initialized
runs to distinguish per-connection, per-session, per-torrent and persistent
values. `SCHEMA-04` and `OBS-07` will lock the minimum after the harness can
measure it; until then no document claims a numeric minimum.

## Controls

- A known fixture client proves every listener and parser sees the fields it is
  meant to see.
- The control is run twice before a causal claim is published.
- Critical guards are mutation-tested against copied evidence.
- Observer configuration is varied to detect perturbation. If terminating a
  protocol changes the client's offer, capture the affected field through a
  raw/passive packet route inside the isolated fixture and record why.
- The two acquisition routes are captured separately at least once. Byte or
  behavior differences are findings, not noise to normalize away.

## Refused evidence

These inputs may seed a hypothesis or parser fixture and may not populate the
catalogue:

- a peer-ID registry or decoder table;
- client source code or a derived profile generator;
- a public DHT or tracker population sample;
- a client UI/API label with no wire evidence;
- an emulator's claimed profile;
- an unversioned binary already present on a runner;
- a capture from a public swarm or a third party's traffic.

## Raw bundle

The bundle contains the generated metainfo and payload digest, observer event
stream, raw tracker requests/datagrams, peer transcript, independent connector
output, packet capture when used, target stdout/stderr after secret scanning,
environment manifest and a checksums file. Timestamps use UTC; relative timing
uses a monotonic clock.

## Failure semantics

- exit 0: measurement ran, controls passed and required correlations agree;
- exit 1: measurement ran and a behavioral/validation expectation failed;
- exit 2: the route could not run, such as an unavailable legitimate package
  or unsupported runner capability.

Exit 2 is not a pass and does not publish. Partial evidence is retained as a
workflow artifact with the reason.

⚠ **A client capture's partial evidence is retained after rule 12 is asked of
it.** `capture-client.yml` uploads a refused capture's bundle once
`scripts/capture/scan-bundle.sh` has passed over it, because the runner's own
scan is on its success path and a refusal is the run whose cleanup may not have
happened. ⛔ `capture.yml`'s fixture captures still upload on success alone, so
this sentence is not yet true of them.
