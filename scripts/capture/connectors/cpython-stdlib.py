#!/usr/bin/env python3
"""cpython-stdlib - the second connector, reading a capture bundle's raw bytes.

WHAT THIS IS FOR
================

Absolute 2 says every profile uses the Rust active observer plus at least one
INDEPENDENT connector, and their overlapping observations agree or the profile
stays unpublished with the disagreement recorded. Until this file existed the
project had one reader of every surface, so a record's corroboration column
would have been the observer agreeing with itself.

The contract is `CI-09`'s and it is already proved from the other side: a
connector declared in the attestation's `connectors=` writes
`connector/<id>.txt` into the bundle, one `field_path=value` line per field the
observer measured. `assemble-capture` reads it and refuses a declared connector
that is silent about a field.

  The value is `out_of_scope`, or EVERY SAMPLE of the field in document order,
  comma-separated, each lowercase hex or `absent`. A surface the bundle carries
  with no segment showing the build answering is one `absent`, the same line one
  absent sample would write: both say the build put nothing there. ⛔ One sample
  per segment rather than one value per field, because a field resting on four
  announces is four observations and a single value corroborates one of them;
  `assemble-capture` compares the lists position by position.

WHERE THE INDEPENDENCE ACTUALLY IS, STATED RATHER THAN CLAIMED
==============================================================

This reads the same bytes as `bit-ids-probe`, out of the transcript the lab
wrote, and decodes them with implementations this project did not write:

  - the transcript document is read by `json`. The lab SERIALISES it by hand,
    because the field order and the hex case are what a digest names, and it
    parses it back by hand for the same reason. Neither of those had ever been
    read by a conformant JSON reader.
  - the announce target is split by `urllib.parse.urlsplit` and every query
    value is percent-decoded by `urllib.parse.unquote_to_bytes`. That is where
    a peer ID actually lives and it is the decoding most likely to differ:
    `bit-ids-wire` has its own.
  - the request's headers are parsed by `http.client.parse_headers`, which is
    `email.parser` underneath - a header reader with two decades of conformance
    behind it.

  The peer-wire handshake is the weak one and this file will not pretend
  otherwise. It is a fixed layout - a length byte, that many protocol bytes,
  eight reserved, twenty info hash, twenty peer ID - and two readings of it can
  differ only in their bounds checking. The independence there is real and it is
  smaller than on the announce, which is why it is written down here rather than
  left for a reader to assume from the word "connector".

WHAT IS DELIBERATELY NOT SHARED WITH THE OBSERVER
=================================================

The observer takes the FIRST `from_target` segment of each transcript and
requires it to parse. This scans EVERY `from_target` segment and identifies the
surface by its content: an announce is a segment that parses as an HTTP request
carrying a `peer_id` query field, and a handshake is one that begins with the
BitTorrent protocol string. Selecting by position would have made the two
readers share a decision, and a record's corroboration would then rest on both
of them having guessed the same segment.

  Two segments that both parse and DISAGREE are both reported, in order, rather
  than resolved. A connector that took one of two differing announces would be
  manufacturing the constant the record is about to declare, and one that
  refused them would be calling a build that varies a broken bundle.

  ⛔ THIS REFUSED THEM UNTIL 2026-10-14, AND THAT REFUSAL WAS THE DEFECT. Both
  contained `capture-client` runs of 2026-10-11 put `aria2-next` 2.8.6 on the
  wire for two sessions, the build announced a different peer ID in each, and
  this file exited 1 with "two segments disagree about tracker_http/peer_id" -
  so neither capture became a record. A tail that moves between sessions is
  what `SCHEMA-04` calls `patterned`; the samples are the measurement, and
  choosing between them was never this reader's to do.

WHY PYTHON, WHEN THIS REPOSITORY'S RULE IS SHELL AND RUST
=========================================================

bit-ids:python-exception=OBS-07


`docs/AGENTS.md` section 5 permits Python only where a documented constraint
makes both unsuitable, and this is that constraint rather than a preference:

  - Rust in this workspace is the reading that is being corroborated. A second
    connector written against `bit-ids-wire` corroborates nothing, and one
    written beside it in the same tree by the same hand is the self-consistency
    `OBS-07`'s Source names.
  - A third-party Rust crate would be independent and would also be a
    dependency of the crate that publishes records, which `docs/supply-chain.md`
    makes this project argue for. The argument is worse: it puts a parser in the
    publisher's own dependency tree to check the publisher.
  - Shell has no HTTP header parser and no percent-decoder. Writing either in
    `awk` reproduces exactly the shared-reading problem above.

  `scripts/README.md` already records the precedent: `check-staleness` re-derives
  a request identifier with `python3`'s SHA-256, "an implementation this project
  did not write". This is the same argument about a bigger surface.

Usage:
  python3 scripts/capture/connectors/cpython-stdlib.py --describe
  python3 scripts/capture/connectors/cpython-stdlib.py --bundle <dir> [--out <path>]

Exit codes: 0 a report was written, 1 the evidence does not support one,
2 could not run.

Read the exit code from this process, unpiped.
"""

import http.client
import io
import json
import os
import sys
import urllib.parse

# The identifier this connector is named by, in the attestation's `connectors=`
# and in the report's own filename. Asked for with `--describe`, never composed
# by a caller: a second spelling goes on writing the old name the day this one
# changes, and `assemble-capture` would then read a file nothing wrote.
#
# It is a `Slug`: a-z0-9 separated by single hyphens.
CONNECTOR_ID = "cpython-stdlib"

# The protocol name every BitTorrent handshake carries, BEP 3.
PROTOCOL = b"BitTorrent protocol"

# A handshake is 1 + len(PROTOCOL) + 8 + 20 + 20.
HANDSHAKE_LEN = 1 + len(PROTOCOL) + 8 + 20 + 20

TRACKER_TRANSCRIPT = "tracker-http.transcript.json"
PEER_TRANSCRIPT = "peer-wire-dialled.transcript.json"

# What a connector may say about a field it did not read as bytes.
ABSENT = "absent"
OUT_OF_SCOPE = "out_of_scope"


class Refused(Exception):
    """The evidence does not support a report. Exit 1."""


class CannotRun(Exception):
    """Something outside the evidence stopped this. Exit 2."""


def hex_of(raw):
    """The lowercase hex this project spells every byte string in."""
    return raw.hex()


def segments(bundle, name):
    """Every `from_target` segment of one transcript, in document order.

    Returns None when the bundle does not carry that transcript at all, which
    is not the same fact as a transcript that will not read.

      - A MISSING file means the run recorded no such surface. A capture whose
        target never accepted a peer connection writes no peer transcript, and
        the attestation says so in `peer_streams`. This connector cannot observe
        a surface the bundle does not contain, and `out_of_scope` is exactly
        that: not a value, not an absence, and the reason its silence proves
        nothing. Reporting `absent` instead would be claiming the condition was
        created and the build produced nothing, which is a corroboratable fact
        this connector is in no position to assert.
      - A file that is PRESENT and will not read is refused. Whatever is wrong
        there is wrong with the evidence, and a connector that turned it into
        `out_of_scope` would answer a broken bundle with a shrug.

    The transcript schema declares LOWERCASE hex, and this refuses anything
    else rather than calling `bytes.fromhex`, which accepts either case. The
    rejected alternative was to be permissive: a connector that read a document
    the lab could not have written would be corroborating bytes whose origin it
    had just declined to check.
    """
    path = os.path.join(bundle, name)
    if not os.path.exists(path):
        return None
    try:
        with open(path, "rb") as handle:
            raw = handle.read()
    except OSError as error:
        raise Refused("%s: %s" % (path, error))

    try:
        document = json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, ValueError) as error:
        raise Refused("%s is not a JSON document: %s" % (path, error))

    if not isinstance(document, dict):
        raise Refused("%s is not a transcript object" % path)
    schema = document.get("schema")
    if schema != "bit-ids/transcript/1":
        raise Refused("%s declares schema %r, not bit-ids/transcript/1" % (path, schema))
    listed = document.get("segments")
    if not isinstance(listed, list):
        raise Refused("%s carries no segment list" % path)

    out = []
    for index, segment in enumerate(listed):
        if not isinstance(segment, dict):
            raise Refused("%s segment %d is not an object" % (path, index))
        if segment.get("direction") != "from_target":
            continue
        text = segment.get("bytes")
        if not isinstance(text, str):
            raise Refused("%s segment %d carries no byte string" % (path, index))
        # Two refusals rather than one, because the fixes differ and a case
        # asserting a message a different guard produced is how two guards over
        # one input come to mask each other. Measured: a plant that made this
        # accept uppercase survived a harness whose "uppercase" case had also
        # changed the length, so the odd-length branch refused it and the case
        # reported a guard it had never reached.
        if len(text) % 2:
            raise Refused(
                "%s segment %d is an odd number of hex digits" % (path, index)
            )
        if any(character not in "0123456789abcdef" for character in text):
            raise Refused(
                "%s segment %d is not lowercase hex" % (path, index)
            )
        out.append(bytes.fromhex(text))
    return out


def announce_fields(raw):
    """The peer ID and User-Agent of one segment, or None if it is not an announce.

    Returns a dict of field path to value, where a value is bytes or ABSENT.
    """
    stream = io.BytesIO(raw)
    line = stream.readline()
    parts = line.split()
    if len(parts) != 3 or not parts[2].upper().startswith(b"HTTP/"):
        return None
    target = parts[1]

    # `urlsplit` takes bytes and gives bytes back, so nothing here round-trips
    # a byte through a text decoding on the way to `unquote_to_bytes`. A value
    # above 0x7f would not survive that trip, and a peer ID's tail is random.
    query = urllib.parse.urlsplit(target).query
    peer_id = None
    for field in query.split(b"&"):
        name, separator, value = field.partition(b"=")
        if not separator:
            continue
        if urllib.parse.unquote_to_bytes(name) != b"peer_id":
            continue
        decoded = urllib.parse.unquote_to_bytes(value)
        if peer_id is not None and peer_id != decoded:
            raise Refused("one announce carries two different peer_id fields")
        peer_id = decoded
    if peer_id is None:
        return None

    try:
        headers = http.client.parse_headers(stream)
    except http.client.HTTPException as error:
        raise Refused("an announce's headers: %s" % error)
    agent = headers.get("User-Agent")

    return {
        "tracker_http/peer_id": peer_id,
        "tracker_http/user_agent": (
            ABSENT if agent is None else agent.encode("iso-8859-1")
        ),
    }


def handshake_fields(raw):
    """The peer ID and reserved bytes of one segment, or None if it is not a handshake."""
    if len(raw) < 1 or raw[0] != len(PROTOCOL):
        return None
    if raw[1 : 1 + len(PROTOCOL)] != PROTOCOL:
        return None
    if len(raw) < HANDSHAKE_LEN:
        raise Refused(
            "a segment begins with the protocol string and is %d bytes, not %d"
            % (len(raw), HANDSHAKE_LEN)
        )
    reserved_at = 1 + len(PROTOCOL)
    peer_id_at = reserved_at + 8 + 20
    return {
        "peer_wire/reserved": raw[reserved_at : reserved_at + 8],
        "peer_wire/peer_id": raw[peer_id_at : peer_id_at + 20],
    }


def collect(found, reader, raw):
    """Append one segment's fields to the report, one sample per segment.

    ⛔ Every sample is kept, in document order, and none is compared with
    another here. Whether two announces differ is the record's question -
    `sampling::field_state` reads the fixed span and the moving one - and a
    reader that answered it by refusing or by picking would be deciding the
    measurement instead of reporting it.
    """
    fields = reader(raw)
    if fields is None:
        return
    for path, value in fields.items():
        found.setdefault(path, []).append(value)


# Which surface each field belongs to, so a transcript the bundle does not
# carry makes its own fields out_of_scope and nobody else's.
SURFACES = (
    (TRACKER_TRANSCRIPT, announce_fields, ("tracker_http/peer_id", "tracker_http/user_agent")),
    (PEER_TRANSCRIPT, handshake_fields, ("peer_wire/peer_id", "peer_wire/reserved")),
)


def report_for(bundle):
    """Every field this connector can say something about, as text."""
    found = {}
    unreachable = set()
    for name, reader, paths in SURFACES:
        recorded = segments(bundle, name)
        if recorded is None:
            unreachable.update(paths)
            continue
        for raw in recorded:
            collect(found, reader, raw)

    lines = []
    for _, _, paths in SURFACES:
        for path in paths:
            if path in unreachable:
                value = OUT_OF_SCOPE
            elif path in found:
                # `absent` is what a reader put there for a header the request
                # did not carry; bytes are hex. ⚠ Neither spelling carries a
                # comma, which is what makes the list readable without quoting.
                value = ",".join(
                    hex_of(sample) if isinstance(sample, bytes) else sample
                    for sample in found[path]
                )
            else:
                # The surface is in the bundle and nothing in it showed the
                # build answering. That is an absence two connectors can
                # corroborate, which `out_of_scope` is not.
                value = ABSENT
            lines.append("%s=%s\n" % (path, value))
    lines.sort()
    return "".join(lines)


def describe():
    """What this connector is, for a caller that must not compose either value."""
    version = "%d.%d.%d" % sys.version_info[:3]
    return "id=%s\nversion=%s\nruntime=cpython\n" % (CONNECTOR_ID, version)


def main(argv):
    bundle = None
    out = None
    rest = list(argv)
    while rest:
        flag = rest.pop(0)
        if flag == "--describe":
            sys.stdout.write(describe())
            return 0
        if flag in ("-h", "--help"):
            sys.stdout.write(__doc__)
            return 0
        if flag == "--bundle":
            if not rest:
                raise CannotRun("--bundle takes a value")
            bundle = rest.pop(0)
        elif flag == "--out":
            if not rest:
                raise CannotRun("--out takes a value")
            out = rest.pop(0)
        else:
            raise CannotRun("unknown argument: %s" % flag)

    if bundle is None:
        raise CannotRun("--bundle is required")
    if not os.path.isdir(bundle):
        raise CannotRun("%s is not a directory" % bundle)
    if out is None:
        out = os.path.join(bundle, "connector", "%s.txt" % CONNECTOR_ID)

    text = report_for(bundle)

    directory = os.path.dirname(out)
    if directory:
        try:
            os.makedirs(directory, exist_ok=True)
        except OSError as error:
            raise CannotRun("cannot create %s: %s" % (directory, error))
    try:
        with open(out, "w") as handle:
            handle.write(text)
    except OSError as error:
        raise CannotRun("cannot write %s: %s" % (out, error))

    # Read back rather than assumed, which is this repository's rule for every
    # artifact a script claims to have written.
    try:
        with open(out, "r") as handle:
            written = handle.read()
    except OSError as error:
        raise CannotRun("cannot read back %s: %s" % (out, error))
    if written != text:
        raise CannotRun("%s does not carry what was written" % out)

    sys.stdout.write("%s: %d field(s) -> %s\n" % (CONNECTOR_ID, len(text.splitlines()), out))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except Refused as refusal:
        sys.stderr.write("%s: %s\n" % (CONNECTOR_ID, refusal))
        sys.exit(1)
    except CannotRun as problem:
        sys.stderr.write("%s: %s\n" % (CONNECTOR_ID, problem))
        sys.exit(2)
