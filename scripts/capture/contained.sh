#!/bin/sh
# contained.sh - run a command, as the calling user, in a network namespace of
# its own that holds nothing but loopback.
#
# ⛔ WHY IT EXISTS: THE CAPTURE HANG WAS THE CONTAINMENT. `capture-client` cut
# the HOST's default routes before a capture and restored them after. The host
# is also where the runner agent lives, and the agent reaches GitHub over those
# routes - so from the cut onward the job could not report, and on the runner
# images of 2026-09 onward it never came back: no log, no artifact, and a step
# shown as running that had in fact finished. Measured on 2026-10-11 by a flight
# recorder posting off the host: every operation of *Install the client* ran to
# exit 0, the posts stopped the moment the next step cut the route, and they
# never resumed. Thirteen bounds failed on that step because nothing in it was
# ever hung.
#
# ⭐ SO THE CAPTURE IS CONTAINED, NOT THE HOST. A new network namespace starts
# with one interface, `lo`, and no route at all. The observer and the client
# both run inside it, so they reach each other over loopback and nothing else;
# the runner agent stays in the host's namespace and never loses its link.
# `bit-check assert-disposable --egress` reads `/proc/net/route`, which inside the
# namespace is the namespace's own table, so the guard asks the right question
# of the process it guards.
#
# ⚠ CREATING A NAMESPACE NEEDS `CAP_SYS_ADMIN`, so this asks `sudo -n` for
# exactly that and drops straight back to the caller's uid, gid and groups with
# `setpriv` before the command runs. Nothing the command starts is root.
#
# Usage:
#   sh scripts/capture/contained.sh -- <command> [args...]
#
# Exit codes: the command's own, or 2 when the namespace could not be made.

set -u

if [ "${1:-}" = --inner ]; then
  # ⚠ Root, inside the new namespace, for two commands only.
  if [ $# -lt 5 ] || [ "$4" != -- ]; then
    printf 'contained: --inner takes <uid> <gid> -- <command>\n' >&2
    exit 2
  fi
  _uid=$2
  _gid=$3
  shift 4
  ip link set lo up || {
    printf 'contained: could not bring loopback up in the namespace\n' >&2
    exit 2
  }
  exec setpriv --reuid="$_uid" --regid="$_gid" --init-groups -- "$@"
fi

if [ "${1:-}" != -- ] || [ $# -lt 2 ]; then
  printf 'contained: usage: contained.sh -- <command> [args...]\n' >&2
  exit 2
fi
shift

for _tool in sudo unshare setpriv ip; do
  command -v "$_tool" >/dev/null 2>&1 || {
    printf 'contained: %s is not on this host\n' "$_tool" >&2
    exit 2
  }
done

# ⚠ `sudo -n`, so a host that would prompt says no rather than waiting, and the
# environment is carried explicitly: `env_reset` would otherwise hand the
# command root's PATH and HOME and a capture would write its state somewhere
# the caller does not own.
exec sudo -n env "PATH=$PATH" "HOME=${HOME:-}" \
  unshare --net -- sh "$0" --inner "$(id -u)" "$(id -g)" -- "$@"
