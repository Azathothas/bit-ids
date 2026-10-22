#!/usr/bin/env python3
"""libtorrent-harness - the ENGINE-01 reference harness, the smallest program
that runs libtorrent-rasterbar's session on one torrent.

bit-ids:python-exception=ENGINE-01

WHY PYTHON
==========

The package route is PyPI's `libtorrent` wheel, and what that wheel delivers is
the library's Python binding: there is no other way to run the build it ships.
The source route builds the same binding from the tag with the vendor's own
setup.py. So one harness runs both builds, and it is this file.
`TODO/engines.md` `ENGINE-01` records the choice.

THE LIBRARY'S DEFAULTS, EXCEPT WHERE A FLAG SAYS OTHERWISE
===========================================================

The session is `lt.session(settings)` with the library's default settings
pack; every flag below names one surface and leaves it at the library's value
unless it is passed. The capture adapter, tools/check/adapter_libtorrent.go,
passes the flags that switch the network surfaces off, and libtorrent.go reads
back what this program reports it configured.

NOTHING ON THE WIRE IS SET HERE. The peer fingerprint, the user agent and the
extended handshake's client version are the library's. They are printed as
self-reports, which are never published values.

The alert mask is the one setting this program changes that no flag names:
the library's default is errors only, and the harness asks for status,
tracker, peer, connect and port-mapping alerts so that `--alerts` can record
what the library says it did. It changes nothing a peer or a tracker sees.

Usage:

    harness.py --version
    harness.py --defaults
    harness.py --torrent F --data D --ready R --stop-file S [--alerts A]
               [--port P] [--ipv4-only] [--no-dht] [--no-lsd] [--no-upnp]
               [--no-natpmp] [--no-pex]

Exit codes: 0 done, 1 the library refused, 2 could not run.
"""

import os
import re
import sys

import libtorrent as lt


def mapped_openssl():
    """The OpenSSL libraries this process has mapped. Read before anything
    else is imported: hashlib maps the interpreter's own libcrypto."""
    found = []
    try:
        with open("/proc/self/maps", encoding="ascii", errors="replace") as maps:
            for line in maps:
                fields = line.split()
                name = os.path.basename(fields[-1]) if len(fields) >= 6 else ""
                if re.match(r"lib(ssl|crypto)[-.]", name) and name not in found:
                    found.append(name)
    except OSError:
        pass
    return sorted(found)


MAPPED_OPENSSL = mapped_openssl()

import argparse  # noqa: E402  (after the read above, on purpose)
import hashlib  # noqa: E402
import time  # noqa: E402

SCHEMA_VERSION = "bit-ids/libtorrent-harness-version/1"
SCHEMA_READY = "bit-ids/libtorrent-harness-ready/1"

# The library's default extensions: session_params' default_plugins().
DEFAULT_EXTENSIONS = ("ut_metadata", "ut_pex", "smart_ban")


def text(value):
    """One setting's value as a line: booleans lower-case, newlines escaped,
    and `-` for a string setting the pack leaves unset or empty."""
    if value is None:
        return "-"
    if isinstance(value, bool):
        return "true" if value else "false"
    return str(value).replace("\\", "\\\\").replace("\n", "\\n")


def defaults_dump():
    """Every default setting of this build, sorted, one `name=value` a line."""
    pack = lt.default_settings()
    return "".join("%s=%s\n" % (k, text(pack[k])) for k in sorted(pack))


def openssl_facts():
    """How the binding carries OpenSSL: linked in, which its providers' names
    inside the module show whatever the release, with the release's own text
    when the module keeps it; or a library the import mapped."""
    with open(lt.__file__, "rb") as module:
        body = module.read()
    linked = b"OpenSSL Base Provider" in body or b"OpenSSL DH Method" in body
    texts = sorted(set(re.findall(rb"OpenSSL [0-9]+\.[0-9]+\.[0-9]+[a-z]? +[0-9]+ [A-Z][a-z]{2} [0-9]{4}", body)))
    if MAPPED_OPENSSL:
        how = "mapped"
    elif linked:
        how = "static"
    else:
        how = "none"
    return how, ",".join(t.decode("ascii") for t in texts) or "-", ",".join(MAPPED_OPENSSL) or "-"


def print_version():
    """The build answering for itself: the library's version, the build
    features the binding exposes, and a digest of every default setting."""
    module = lt.__file__
    with open(module, "rb") as f:
        module_sha = hashlib.sha256(f.read()).hexdigest()
    how, version_text, mapped = openssl_facts()
    defaults = defaults_dump()
    pack = lt.default_settings()
    rows = [
        ("library", "libtorrent"),
        ("library_version", lt.__version__),
        ("python", "%d.%d.%d" % sys.version_info[:3]),
        ("python_implementation", sys.implementation.name),
        ("module", "/".join(module.replace("\\", "/").split("/")[-2:])),
        ("module_sha256", module_sha),
        # Each is a symbol the binding compiles only under one define.
        ("abi_version_1", text(hasattr(lt, "version"))),
        ("dht", text(hasattr(lt.session, "add_dht_node"))),
        ("logging", text(hasattr(lt, "log_alert"))),
        ("i2p", text(hasattr(lt.session, "set_i2p_proxy"))),
        ("encryption", text(hasattr(lt.session, "set_pe_settings"))),
        ("openssl", how),
        ("openssl_version_text", version_text),
        ("openssl_mapped", mapped),
        ("default_settings", str(len(pack))),
        ("default_settings_sha256", hashlib.sha256(defaults.encode("utf-8")).hexdigest()),
    ]
    for name in ("user_agent", "peer_fingerprint", "handshake_client_version", "listen_interfaces",
                 "enable_dht", "enable_lsd", "enable_upnp", "enable_natpmp", "enable_outgoing_utp",
                 "enable_incoming_utp", "out_enc_policy", "in_enc_policy", "alert_mask"):
        rows.append(("default." + name, text(pack.get(name))))
    sys.stdout.write(SCHEMA_VERSION + "\n" + "".join("%s=%s\n" % kv for kv in rows))
    return 0


def kind(socket_type):
    """A listen alert's socket type as its enum name."""
    return str(socket_type).rsplit(".", 1)[-1]


def run(args):
    defaults = lt.default_settings()
    port = args.port if args.port is not None else int(defaults["listen_interfaces"].rsplit(":", 1)[-1])
    settings = {"alert_mask": lt.alert_category.error | lt.alert_category.status | lt.alert_category.tracker
                | lt.alert_category.peer | lt.alert_category.connect | lt.alert_category.port_mapping}
    if args.port is not None or args.ipv4_only:
        listen = ["0.0.0.0:%d" % port]
        if not args.ipv4_only:
            listen.append("[::]:%d" % port)
        settings["listen_interfaces"] = ",".join(listen)
    for flag, name in ((args.no_dht, "enable_dht"), (args.no_lsd, "enable_lsd"),
                       (args.no_upnp, "enable_upnp"), (args.no_natpmp, "enable_natpmp")):
        if flag:
            settings[name] = False

    # The library's default plugins, unless PEX is off: then the session gets
    # none and the other two are added back, so ut_pex alone is missing.
    extensions = [e for e in DEFAULT_EXTENSIONS if not (args.no_pex and e == "ut_pex")]
    if args.no_pex:
        ses = lt.session(settings, 0)
        for e in extensions:
            ses.add_extension(e)
    else:
        ses = lt.session(settings)

    alerts = open(args.alerts, "a", encoding="utf-8") if args.alerts else None
    listening = []

    def drain():
        for a in ses.pop_alerts():
            if isinstance(a, lt.listen_succeeded_alert):
                listening.append("%s %s:%d" % (kind(a.socket_type), a.address, a.port))
            if alerts:
                alerts.write("%.3f %s %s\n" % (time.time(), type(a).__name__, a.message().replace("\n", " ")))
                alerts.flush()

    try:
        ti = lt.torrent_info(args.torrent)
    except RuntimeError as e:
        sys.stderr.write("libtorrent-harness: the torrent would not load: %s\n" % e)
        return 1
    atp = lt.add_torrent_params()
    atp.ti = ti
    atp.save_path = args.data
    handle = ses.add_torrent(atp)

    for _ in range(50):
        drain()
        if any(l.startswith("tcp ") for l in listening):
            break
        time.sleep(0.2)
    live = ses.get_settings()
    rows = [
        ("pid", str(os.getpid())),
        ("infohash", str(handle.info_hashes().v1)),
        ("listen_interfaces", live["listen_interfaces"]),
        ("listen_port", str(ses.listen_port())),
        ("listen", ",".join(sorted(listening))),
        ("enable_dht", text(live["enable_dht"])),
        ("enable_lsd", text(live["enable_lsd"])),
        ("enable_upnp", text(live["enable_upnp"])),
        ("enable_natpmp", text(live["enable_natpmp"])),
        ("extensions", ",".join(sorted(extensions))),
        ("proxy_type", text(live["proxy_type"])),
        ("anonymous_mode", text(live["anonymous_mode"])),
        ("enable_outgoing_utp", text(live["enable_outgoing_utp"])),
        ("enable_incoming_utp", text(live["enable_incoming_utp"])),
        ("alert_mask", text(live["alert_mask"])),
        # Self-reports, printed so the record can set them beside what the
        # observer measured. None is ever a published value.
        ("self_report_user_agent", text(live["user_agent"])),
        ("self_report_peer_fingerprint", text(live["peer_fingerprint"])),
        ("self_report_handshake_client_version", text(live.get("handshake_client_version") or None)),
        ("library_default_user_agent", text(defaults["user_agent"])),
        ("library_default_peer_fingerprint", text(defaults["peer_fingerprint"])),
    ]
    with open(args.ready, "w", encoding="utf-8") as ready:
        ready.write(SCHEMA_READY + "\n" + "".join("%s=%s\n" % kv for kv in rows))
    sys.stdout.write("running %s on %s\n" % (rows[1][1], rows[4][1]))
    sys.stdout.flush()

    # The close is the library's own, on a file appearing: a file because
    # Windows has no SIGTERM to send a console-less process.
    while not os.path.exists(args.stop_file):
        drain()
        time.sleep(0.2)
    started = time.time()
    drain()
    del handle
    del ses
    sys.stdout.write("closed on the stop file in %dms\n" % int((time.time() - started) * 1000))
    if alerts:
        alerts.close()
    return 0


def main():
    if sys.argv[1:] == ["--version"]:
        return print_version()
    if sys.argv[1:] == ["--defaults"]:
        sys.stdout.write(defaults_dump())
        return 0
    p = argparse.ArgumentParser(prog="libtorrent-harness", allow_abbrev=False)
    p.add_argument("--torrent", required=True)
    p.add_argument("--data", required=True)
    p.add_argument("--ready", required=True)
    p.add_argument("--stop-file", required=True)
    p.add_argument("--alerts")
    p.add_argument("--port", type=int)
    for surface in ("ipv4-only", "no-dht", "no-lsd", "no-upnp", "no-natpmp", "no-pex"):
        p.add_argument("--" + surface, action="store_true")
    try:
        args = p.parse_args()
    except SystemExit:
        return 2
    return run(args)


if __name__ == "__main__":
    sys.exit(main())
