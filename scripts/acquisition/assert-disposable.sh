#!/bin/sh
# assert-disposable.sh - refuse to install or run a client on a host somebody keeps.
#
# ⛔ THIS RUNS BEFORE THE INSTALL, AND THAT IS THE WHOLE POINT. The run manifest
# already refuses to RECORD a capture whose host was not disposable, per
# `E-MAN-30`. That guard cannot stop one: by the time a manifest exists, an
# untrusted installer has already run somewhere. A record-layer refusal is a
# report, not a boundary.
#
# -- ⭐ TWO INDEPENDENT GUARDS, AND WHY THEY ARE NOT ONE ---------------------
#
#   --claim    the host has never been used for a capture before
#   --egress   the host cannot reach anything but loopback
#
# They fail for different reasons and neither implies the other. A fresh host
# with an open route leaks the capture onto the public network. A firewalled
# host that already ran a capture contaminates this one with the last one's
# state. `docs/AGENTS.md` section 5 requires both to exist before any client is
# installed.
#
# -- ⛔ --claim DETECTS THE FAILURE RATHER THAN TRUSTING THE CLAIM ------------
#
# The obvious design is a token the provisioner writes saying "this host is
# disposable". ⚠ That is a promise, and a promise is exactly what fails
# silently: a runner misconfigured to persist its disk still carries the token,
# still says disposable, and nothing notices until two captures share state.
#
# So this claims the host by WRITING a marker, and refuses if one is already
# there. A second capture on one host means the host survived the first, which
# means it was never disposable, whatever anything claimed. The evidence is the
# marker's existence and it cannot be faked by getting the configuration wrong.
#
# ⚠ The marker therefore lives where a real teardown destroys it and a survived
# host keeps it. ⛔ `/run` and `/tmp` are cleared by a reboot, so a marker there
# would report a host that rebooted rather than being destroyed as fresh, and
# that is the one thing this guard exists to catch.
#
# -- ⭐ IT IS THE USER'S OWN STATE DIRECTORY, NOT `/var/lib`. `ACQ-06`. --------
#
# `/var/lib/bit-ids` needs root to create, so every caller of this guard needed
# `sudo` - including `install-client.sh`, which reads the marker before a route
# runs. `$HOME/.local/state/bit-ids` keeps both properties the marker rests on:
# it is not cleared by a reboot, and it goes when the host does. What it does not
# need is a privilege.
#
# ⛔ AND IT IS PER-USER, WHICH IS A NARROWER CLAIM THAN `/var/lib` MADE. A host
# where two different users each ran a capture carries two markers and neither
# refuses the other. That is a real weakening and it is recorded rather than
# hidden: the runner contract is one capture per host, `docs/capture-host.md`
# carries it, and a shared host was never a host this project would capture on.
#
# ⚠ Override with BIT_IDS_STATE_DIR only to test this script.
#
# Usage:
#   sh scripts/acquisition/assert-disposable.sh --claim <run-id>
#   sh scripts/acquisition/assert-disposable.sh --egress [routing-table]
#   sh scripts/acquisition/assert-disposable.sh --fingerprint
#   sh scripts/acquisition/assert-disposable.sh --marker
#
# --claim prints the host fingerprint it recorded. --fingerprint prints the
# current one without claiming, so the next job can compare and see a different
# host.
#
# ⛔ --marker PRINTS THE MARKER'S PATH AND IS THE ONLY DERIVATION OF IT. A
# caller that wants to know whether this host was claimed, and by which run,
# reads the file this names rather than composing `$STATE_DIR/host-claimed` a
# second time. Two spellings of one path is the shape `docs/publishing.md`
# records twice: the copies agree until the state directory moves, and then the
# reader looks in the old place and reports a host nobody claimed as fresh.
# `capture-run.sh` is the caller this exists for.
#
# ⚠ BOTH GUARDS PRINT THE INPUT THEY READ, and that is not decoration. The
# routing table is an optional argument and the marker directory an environment
# variable, so that the runner test can drive them against fixtures. A seam a
# test can use is a seam a misconfiguration can use, so a passing run says which
# table and which marker it trusted, and an evidence bundle carries the answer.
# A guard that passed over a fixture is then visible rather than indistinguishable
# from one that passed over the machine.
#
# Exit codes: 0 the guard passed, 1 the guard refuses, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

# ⚠ `$HOME` IS REQUIRED TO BE ABSOLUTE RATHER THAN MERELY SET. An empty or
# relative value composes into a path relative to whatever directory the caller
# happened to be in, so `--claim` and the `--marker` a later step reads would
# name two different files and a claimed host would report as fresh.
if [ -n "${BIT_IDS_STATE_DIR:-}" ]; then
  STATE_DIR="$BIT_IDS_STATE_DIR"
else
  case "${HOME:-}" in
    /*) STATE_DIR="$HOME/.local/state/bit-ids" ;;
    *)
      printf 'assert-disposable: HOME is [%s], so there is no state directory this user is known to own\n' \
        "${HOME:-}" >&2
      exit 2
      ;;
  esac
fi
MARKER="$STATE_DIR/host-claimed"

usage() {
  awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
}

# A value that differs between two hosts and survives within one. ⚠ Several
# inputs rather than one: a container reusing a machine-id would still differ in
# boot id, and one that shares a boot id with its host still differs in hostname
# and root device. Any single source is a coincidence away from colliding.
fingerprint() {
  {
    cat /etc/machine-id 2>/dev/null
    cat /proc/sys/kernel/random/boot_id 2>/dev/null
    hostname 2>/dev/null
    uname -srm 2>/dev/null
    stat -c '%d' / 2>/dev/null
  } | sha256sum | cut -d' ' -f1
}

# ⛔ Reports whether anything but loopback is reachable, WITHOUT sending a
# packet anywhere. The kernel's own routing table says what the host would do;
# it opens no connection and touches no third party. Probing a real host to see
# whether egress works would be reaching out from a machine this guard exists to
# establish is contained, which is exactly the wrong order.
#
# ⚠ Read from /proc/net/route rather than from ip(8). A capture host is a
# minimal image and `ip` is not on all of them, and a guard that answers "could
# not establish" on the hosts it most needs to check is a guard that does not
# run where it matters. The kernel interface needs no package.
#
# A default route is destination 00000000 with the UP flag (0x1) set.
#
# ⛔ POSIX awk only. The first version tested the flag with `and(strtonum(...))`,
# which are gawk extensions: on this session's host, with a POSIX awk, they are
# undefined functions, awk exits non-zero, and the guard reported "could not
# establish" over a machine that plainly had a default route. It failed closed,
# which is the right direction, but a guard that cannot run on a minimal image
# is a guard that does not run where it matters most. The UP bit is the low bit
# of the flags word, so the last hex digit being odd is the same test with no
# extensions.
has_public_route() {
  [ -r "$ROUTE_TABLE" ] || return 2
  awk 'NR > 1 && $2 == "00000000" {
         last = substr($4, length($4), 1)
         if (index("13579bBdDfF", last) > 0) { found = 1 }
       }
       END { exit(found ? 0 : 1) }' "$ROUTE_TABLE"
}

case "${1:-}" in
  -h | --help)
    usage
    exit 0
    ;;

  --fingerprint)
    [ $# -eq 1 ] || {
      usage >&2
      exit 2
    }
    fingerprint
    exit 0
    ;;

  --marker)
    [ $# -eq 1 ] || {
      usage >&2
      exit 2
    }
    printf '%s\n' "$MARKER"
    exit 0
    ;;

  --egress)
    if [ $# -lt 1 ] || [ $# -gt 2 ]; then
      usage >&2
      exit 2
    fi
    ROUTE_TABLE="${2:-/proc/net/route}"
    has_public_route
    case $? in
      0)
        printf 'assert-disposable: a public route exists; a capture host reaches loopback only\n' >&2
        awk 'NR > 1 && $2 == "00000000" { printf "assert-disposable: default route via %s\n", $1 }' \
          "$ROUTE_TABLE" >&2
        exit 1
        ;;
      2)
        printf 'assert-disposable: %s is unreadable, so egress could not be\n' "$ROUTE_TABLE" >&2
        printf 'assert-disposable: established. That is not a pass.\n' >&2
        exit 2
        ;;
      *)
        printf 'no route off this host (read %s)\n' "$ROUTE_TABLE"
        exit 0
        ;;
    esac
    ;;

  --claim)
    [ $# -eq 2 ] || {
      usage >&2
      exit 2
    }
    RUN_ID="$2"
    case "$RUN_ID" in
      '' | *[!a-z0-9-]*)
        printf 'assert-disposable: run id must be lowercase a-z0-9-: %s\n' "$RUN_ID" >&2
        exit 2
        ;;
    esac

    if [ -e "$MARKER" ]; then
      printf 'assert-disposable: this host already ran a capture, so it was not destroyed\n' >&2
      printf 'assert-disposable: %s\n' "$(head -3 "$MARKER" 2>/dev/null | tr '\n' ' ')" >&2
      exit 1
    fi

    mkdir -p "$STATE_DIR" 2>/dev/null || {
      printf 'assert-disposable: cannot create %s\n' "$STATE_DIR" >&2
      exit 2
    }

    PRINT=$(fingerprint)
    [ -n "$PRINT" ] || {
      printf 'assert-disposable: the host fingerprint is empty\n' >&2
      exit 2
    }

    # ⛔ Created exclusively. Two captures racing on one host must not both
    # believe they claimed it, and `set -C` makes the shell refuse to truncate
    # an existing file rather than overwrite it.
    (
      set -C
      {
        printf 'bit-ids/host-claim/1\n'
        printf 'run=%s\n' "$RUN_ID"
        printf 'claimed_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        printf 'fingerprint=%s\n' "$PRINT"
        # ⭐ THE HOST FACTS A RUN MANIFEST NEEDS, TAKEN AT THE CLAIM. `CI-09`:
        # the image, the operating system and whether the host is GitHub's own
        # exist only in this job's environment, which the capture's namespace
        # does not inherit, and the uptime starts the monotonic span. ⚠ Each is
        # written as read, empty where the host does not say, never filled in.
        printf 'image=%s\n' "${ImageOS:-}${ImageVersion:+/$ImageVersion}"
        printf 'os=%s\n' "$(sed -n 's/^PRETTY_NAME="\{0,1\}\([^"]*\)"\{0,1\}$/\1/p' /etc/os-release 2>/dev/null | head -1)"
        printf 'runner_environment=%s\n' "${RUNNER_ENVIRONMENT:-}"
        printf 'uptime_at_claim=%s\n' "$(cut -d' ' -f1 /proc/uptime 2>/dev/null)"
      } >"$MARKER"
    ) || {
      printf 'assert-disposable: another capture claimed this host first\n' >&2
      exit 1
    }

    printf '%s (claimed %s)\n' "$PRINT" "$MARKER"
    exit 0
    ;;

  *)
    usage >&2
    exit 2
    ;;
esac
