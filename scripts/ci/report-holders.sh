#!/bin/sh
# report-holders.sh - which processes still hold an open descriptor on a file?
#
# ⛔ A STEP DOES NOT END WHEN ITS COMMAND EXITS. A runner reads a step's output
# through a pipe and the step is over when the command has gone AND that pipe
# has reached end of file, so a process the step left behind holding its output
# keeps the runner waiting on work that finished.
# [`check-step-bodies.sh`](check-step-bodies.sh) is what proves that shape; this
# is what NAMES the process on a host where it happens.
#
# ⚠ IT IS A DIAGNOSTIC AND ITS SUBJECT IS ANOTHER PROCESS, so it reports and
# never refuses: a step that went red because its diagnostic found nothing would
# replace a missing answer with a wrong one. Exit 0 means it looked; exit 2 means
# it could not, which is a different fact and is said out loud.
#
# ⚠ IT NEEDS TO BE ABLE TO READ OTHER PROCESSES' DESCRIPTORS. An install that ran
# under sudo leaves root-owned processes, and an unprivileged reader gets nothing
# from their `/proc` entries - reporting "nobody holds it" over a file three root
# processes are holding. Run it as the same user the work ran as.
#
# Usage:
#   sh scripts/ci/report-holders.sh <path> [<path>...]
#
# Exit codes: 0 it looked and said what it found, 2 it could not look.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

[ $# -ge 1 ] || {
  printf 'report-holders: at least one path is required\n' >&2
  exit 2
}

[ -d /proc/self/fd ] || {
  printf 'report-holders: this host has no /proc, so nothing can be read\n' >&2
  exit 2
}

# ⚠ THE PATH IS RESOLVED ONCE AND COMPARED WHOLE. A descriptor's link is an
# absolute real path, so comparing it against an argument that carries a symlink
# or a relative segment finds nothing and reports it as nobody holding the file.
for _path in "$@"; do
  _real=$(readlink -f -- "$_path" 2>/dev/null) || _real=""
  if [ -z "$_real" ]; then
    printf '## %s does not resolve to a path\n' "$_path"
    continue
  fi
  printf '## processes holding %s\n' "$_real"
  _found=0
  for _proc in /proc/[0-9]*; do
    [ -d "$_proc/fd" ] || continue
    for _fd in "$_proc"/fd/*; do
      _link=$(readlink -- "$_fd" 2>/dev/null) || continue
      [ "$_link" = "$_real" ] || continue
      _pid=${_proc#/proc/}
      _comm=$(cat "$_proc/comm" 2>/dev/null) || _comm='?'
      _args=$(tr '\0' ' ' <"$_proc/cmdline" 2>/dev/null) || _args=''
      printf 'pid=%s comm=%s args=%s\n' "$_pid" "$_comm" "$_args"
      _found=$((_found + 1))
      break
    done
  done
  # ⚠ ZERO IS AN ANSWER AND IT IS WRITTEN DOWN. A section with no rows under it
  # reads the same whether nothing holds the file or the loop never ran, which is
  # the "nothing wrong" against "nothing examined" distinction every guard here
  # owes its reader.
  printf '## %s holder(s)\n' "$_found"
done

# ⚠ AND THE WHOLE PROCESS TABLE BESIDE IT, because a process that holds NO
# descriptor on these files and is still running is the other half of the
# question. `etimes` is how long it has been alive, which separates something
# this step started from something the image did.
printf '## the process table\n'
if ps -ww -eo pid,ppid,pgid,stat,etimes,comm,args 2>/dev/null; then
  :
else
  printf 'ps could not be read on this host\n'
fi
exit 0
