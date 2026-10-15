#!/bin/sh
# store-lib.sh - what the two corpus mutation harnesses both need.
#
# ⛔ IT IS SOURCED, NEVER RUN. It defines functions and sets nothing else, so
# sourcing it twice is harmless and running it does nothing. `check-store.sh`
# plants defects in a disposable tree and reads the refusal, and the machinery
# for that is here: build the example, make a scratch tree, digest a whole
# directory, verify a plant landed, count a row. ⚠ `check-corpus.sh` used it
# too until 2026-10-14, when it became `bit-check check-corpus`.
#
# The alternative was a copy in each, which is the divergent-copies row in
# docs/conventions/forbidden-patterns.md. ⚠ The copies would have diverged in the
# place that matters most: `replace_once` grew a rule about multi-line literals
# only after one of them was written.
#
# A caller sets ME to its own name for diagnostics, then calls store_require,
# store_build and store_workdir before anything else.

# Every tool the harnesses need, checked once and named when absent.
# ⛔ Exit 2, never 1. A machine that cannot run a check has not failed it, and
# the gate runner reads 2 as a skip.
store_require() { # tool...
  for _tool in "$@"; do
    command -v "$_tool" >/dev/null 2>&1 || {
      printf '%s: %s not found\n' "$ME" "$_tool" >&2
      exit 2
    }
  done
}

# Builds one example and reports the path to it.
#
# ⛔ It checks the binary is there afterwards. A build that exits 0 having
# produced nothing is the "step that exits 0 having done nothing" row, and every
# case downstream of it would report a guard that failed to fire.
#
# ⛔ CARGO_TARGET_DIR IS ASKED FOR RATHER THAN ASSUMED AWAY, and that was a real
# hole. This composed the path as root/target and cargo obeys the environment,
# so with that variable set the build succeeded, the binary landed somewhere
# else, and the check above fired: exit 2, which the gate reads as a skip. All
# five provers then reported nothing at all, on a machine whose only oddity was
# a variable a great many Rust developers set. Measured on 2026-09-06 while
# driving CI-01, whose harness sets it.
# ⚠ THE PACKAGE IS AN ARGUMENT AND DEFAULTS TO `bit-ids`, which is what every
# caller before `check-assemble` needed. `assemble-capture` lives in
# `bit-ids-probe`, because it decodes transcripts with `bit-ids-wire` and
# `bit-ids` cannot depend on that crate - the dependency runs the other way. A
# second build path in that one harness is the divergent-copies row this library
# exists to avoid, so the parameter is here rather than a `cargo build` there.
store_build() { # root example [package]
  if ! cargo build --manifest-path "$1/Cargo.toml" -p "${3:-bit-ids}" --locked \
    --example "$2" >/dev/null 2>&1; then
    printf '%s: cannot build the %s example\n' "$ME" "$2" >&2
    exit 2
  fi
  _bin="${CARGO_TARGET_DIR:-$1/target}/debug/examples/$2"
  [ -x "$_bin" ] || {
    printf '%s: %s is not executable after a successful build\n' "$ME" "$_bin" >&2
    exit 2
  }
  printf '%s\n' "$_bin"
}

# Builds `bit-check` into a directory and reports the path to it.
#
# ⭐ FOR A HARNESS WHOSE SUBJECT IS ONE OF ITS HELPERS. `assemble-dispatch` and
# `assemble-bundle` were shell scripts and are `bit-check` helpers since
# 2026-10-15, `CI-07`, and the two harnesses that drive them build it here
# rather than each spelling its own `go build`.
store_bitcheck() { # root dir
  command -v go >/dev/null 2>&1 || {
    printf '%s: go is not on this host, so bit-check cannot be built\n' "$ME" >&2
    exit 2
  }
  if ! (cd "$1/tools/check" && go build -o "$2/bit-check" .) >/dev/null 2>&1; then
    printf '%s: cannot build bit-check\n' "$ME" >&2
    exit 2
  fi
  [ -x "$2/bit-check" ] || {
    printf '%s: %s is not executable after a successful build\n' "$ME" "$2/bit-check" >&2
    exit 2
  }
  printf '%s\n' "$2/bit-check"
}

# A scratch directory that removes itself. The caller installs the trap, because
# a trap set inside a function is the caller's anyway and hiding that is worse
# than writing it twice.
store_workdir() { # tag
  _dir="${TMPDIR:-/tmp}/.$1.$$"
  mkdir -p "$_dir" || {
    printf '%s: cannot write to %s\n' "$ME" "$_dir" >&2
    exit 2
  }
  printf '%s\n' "$_dir"
}

# ⛔ PREFIXED, BECAUSE A SOURCED LIBRARY SHARES ONE NAMESPACE WITH ITS CALLER.
# These were PASS, FAIL and ROWS, and check-indexes.sh assigned its own ROWS for
# a row count. The accumulator was overwritten, two rows vanished from the
# report, and the run still printed "10 passed" over eight lines: a report that
# lies about itself, produced by nothing more than a variable name.
#
# ⭐ The prefix makes the collision unlikely and store_report makes it visible,
# which is the half that matters: a naming convention is a rule nobody checks.
STORE_PASS=0
STORE_FAIL=0
STORE_ROWS=""

row() {
  STORE_ROWS="$STORE_ROWS  $1
"
}

fail() {
  row "❌ $1"
  STORE_FAIL=$((STORE_FAIL + 1))
}

pass() {
  row "✅ $1"
  STORE_PASS=$((STORE_PASS + 1))
}

# ⛔ THE LAYOUT IS ASKED FOR, NEVER SPELLED IN SHELL. A second copy of the path
# derivation here is the drift check-twins.sh exists to catch, and it would
# drift in the direction that makes every case pass over a tree the store would
# never have written. `check-store --where` is the one derivation.
place() { # locator tree record
  _rel=$("$1" --where "$3")
  [ -n "$_rel" ] || return 1
  mkdir -p "$2/$(dirname "$_rel")" || return 1
  cp "$3" "$2/$_rel" || return 1
  printf '%s\n' "$_rel"
}

# A digest over every path in a tree and what sits at it, so a plant that did not
# land is visible as a digest that did not move.
tree_digest() { # dir
  (
    cd "$1" 2>/dev/null || exit 1
    find . \( -type f -o -type l -o -type p \) | LC_ALL=C sort | while read -r p; do
      if [ -L "$p" ]; then
        printf '%s symlink\n' "$p"
      elif [ -p "$p" ]; then
        printf '%s fifo\n' "$p"
      else
        printf '%s %s\n' "$p" "$(sha256sum "$p" | cut -d' ' -f1)"
      fi
    done
  ) | sha256sum | cut -d' ' -f1
}

tree_files() { # dir
  find "$1" \( -type f -o -type l -o -type p \) | wc -l | tr -d ' '
}

# ⛔ EXACTLY ONCE, OR NOT AT ALL. A literal that matches twice edits something
# other than what the case names, and one that matches nothing edits nothing
# while the case still reports a guard that failed to fire.
#
# ⚠ SINGLE-LINE LITERALS ONLY, AND THAT IS CHECKED RATHER THAN ASSUMED. grep -F
# splits a pattern containing a newline into separate alternatives, so a unique
# multi-line literal counts as the sum of its lines and this function would
# report it ambiguous. Measured on 2026-09-05, where exactly that miscounted
# three plants as NOT-PLANTED. Refusing one outright is honest; counting one
# wrongly is the defect this function exists to prevent.
#
# -- ⛔ THE COUNTING AND THE REPLACING SPEAK ONE LANGUAGE, AND FOR A WHILE THEY
# DID NOT ---------------------------------------------------------------------
#
# This counted with `grep -F`, which is LITERAL, and then edited with
# `sed "s/$2/$3/"`, which is a REGULAR EXPRESSION over an unescaped pattern and
# an unescaped replacement. Two consequences, both measured on 2026-09-09 while
# a new caller tripped over the second:
#
#   * ⛔ A literal carrying a regex metacharacter plants somewhere else. Over
#     `axb then a.b`, the literal `a.b` occurs exactly once - at the end - so the
#     count accepted it, and sed then replaced `axb`. The plant applied, so no
#     case reported NOT-PLANTED, and whatever refusal followed was read as proof
#     about a defect that had been planted in the wrong place.
#   * ⚠ A literal carrying a `/` ends the `s` command. sed exits non-zero, this
#     returns 1, and the caller reports a plant that did not apply - which fails
#     safe for a plant and NOT for `store_probe_guards`' no-op case below, where
#     a failure is the expected outcome: that row passed over a sed that never
#     parsed its own expression.
#
# ⭐ So the edit is literal on both sides now. `index` and `substr` are string
# operations with no pattern language behind them, and the two values arrive
# through the environment rather than through `awk -v`, which processes escape
# sequences in what it is given.
replace_once() { # file literal replacement
  case "$2" in
    *"
"*)
      return 1
      ;;
  esac
  _hits=$(grep -o -F -e "$2" "$1" 2>/dev/null | wc -l | tr -d ' ')
  [ "$_hits" = "1" ] || return 1
  _before=$(sha256sum "$1" | cut -d' ' -f1)
  # ⚠ A TEMP FILE IN THE SAME DIRECTORY, THEN A RENAME. A killed run leaves the
  # original intact rather than a half-written file; conventions/shell.md 1.
  _replace_once_old="$2" _replace_once_new="$3" awk '
    BEGIN { old = ENVIRON["_replace_once_old"]; new = ENVIRON["_replace_once_new"] }
    {
      if (!done) {
        at = index($0, old)
        if (at > 0) {
          $0 = substr($0, 1, at - 1) new substr($0, at + length(old))
          done = 1
        }
      }
      print
    }
  ' "$1" >"$1.replace-once" || return 1
  mv "$1.replace-once" "$1" || return 1
  _after=$(sha256sum "$1" | cut -d' ' -f1)
  [ "$_before" != "$_after" ] || return 1
  return 0
}

# The three self-guards every harness that plants defects owes. ⭐ A probe's
# guard is a guard like any other, and this project has been burned by an
# unverified plant three times.
store_probe_guards() { # file present-literal ambiguous-literal
  if replace_once "$1" "a literal this file does not carry" "x"; then
    fail "probe    an absent literal was reported as planted"
  else
    pass "probe    an absent literal is refused"
  fi

  if replace_once "$1" "$3" "x"; then
    fail "probe    an ambiguous literal was reported as planted"
  else
    pass "probe    an ambiguous literal is refused"
  fi

  if replace_once "$1" "$2" "$2"; then
    fail "probe    a no-op edit was reported as planted"
  else
    pass "probe    a no-op edit is refused"
  fi

  if replace_once "$1" "$2
" "x"; then
    fail "probe    a multi-line literal was reported as planted"
  else
    pass "probe    a multi-line literal is refused"
  fi

  # ⛔ AND THE TWO SHAPES THAT MADE THE COUNTING AND THE REPLACING DISAGREE.
  # Each needs a file with specific content, so each writes its own rather than
  # editing the caller's.
  #
  # ⛔ IN A SCRATCH DIRECTORY OF ITS OWN, NEVER BESIDE THE CALLER'S FILE. Two
  # callers hand this a TRACKED path - `check-capture` and `check-capture-client`
  # both pass a runner script in the working tree - so a sibling written next to
  # `$1` is an untracked dropping in the source tree, left behind by any run that
  # is interrupted. ⚠ It is the same rule the harnesses already follow about
  # editing a tracked file, and the first version of these two probes broke it.
  _probe_dir="${TMPDIR:-/tmp}/.storeprobe.$$"
  mkdir -p "$_probe_dir" || {
    fail "probe    a scratch directory for the plant probes could not be made"
    return 0
  }

  # ⚠ The first is the one that cost a case silently: over `axb then a.b` the
  # LITERAL `a.b` occurs once and a REGEX `a.b` matches `axb` first, so the plant
  # applied in a place no case named. It asserts WHERE the edit landed, not that
  # one happened.
  printf 'axb then a.b\n' >"$_probe_dir/metachar"
  if replace_once "$_probe_dir/metachar" 'a.b' 'PLANTED' &&
    [ "$(cat "$_probe_dir/metachar")" = "axb then PLANTED" ]; then
    pass "probe    a literal carrying a regex metacharacter plants where it occurs"
  else
    fail "probe    a metacharacter literal planted [$(cat "$_probe_dir/metachar")]"
  fi

  # ⚠ The second could not plant at all: a `/` ends sed's own `s` command, so
  # every literal naming a path was a case that quietly never ran.
  printf 'keep a/b here\n' >"$_probe_dir/slash"
  if replace_once "$_probe_dir/slash" 'a/b' 'PLANTED' &&
    [ "$(cat "$_probe_dir/slash")" = "keep PLANTED here" ]; then
    pass "probe    a literal carrying a path separator can be planted"
  else
    fail "probe    a literal carrying a slash planted [$(cat "$_probe_dir/slash")]"
  fi

  rm -rf "$_probe_dir"
}

# The shared verdict. ⛔ A run that passed nothing is red whatever else it says:
# zero failures out of zero cases executed is the shape these runners exist to
# refuse.
#
# ⚠ The json flag is an argument rather than a variable this reads out of the
# caller. Ambient state across a source boundary is invisible to shellcheck, and
# silencing the resulting diagnostic teaches the next reader to silence it again.
store_report() { # schema noun json
  _total=$((STORE_PASS + STORE_FAIL))

  # ⛔ THE REPORT CHECKS ITSELF. The row list and the counters are two records of
  # one fact, and a value in two places with nothing comparing them is the copy
  # a reader trusts being the wrong one. Measured: a caller's own variable
  # overwrote the accumulator and the summary went on claiming a count the rows
  # did not support.
  #
  # ⛔ IT COUNTS ROW STARTS AND NOT LINES, and counting lines HID THE FAILURES
  # IT EXISTS TO SURFACE. A row may be several lines - `check-bitcheck` prints a
  # disagreement as the label plus the two JSON lines that differ - so three real
  # failures made the row list six lines longer than the count, this check fired,
  # and the report returned 1 WITHOUT PRINTING A SINGLE ROW. The one run that had
  # something to say was the one run that said nothing.
  #
  # ⚠ A row start is `row`'s own two-space prefix followed by a non-space; a
  # continuation line is indented further by whoever wrote the message.
  _rows=$(printf '%s' "$STORE_ROWS" | grep -c '^  [^ ]')
  if [ "$_rows" != "$_total" ]; then
    printf '%s: %s rows recorded, %s counted; the report does not describe itself\n' \
      "$ME" "$_rows" "$_total" >&2
    return 1
  fi

  if [ "$STORE_PASS" -eq 0 ] || [ "$STORE_FAIL" -gt 0 ]; then
    _rc=1
  else
    _rc=0
  fi

  if [ "${3:-0}" = "1" ]; then
    printf '{"schema":"%s","total":%s,"passed":%s,"failed":%s}\n' \
      "$1" "$_total" "$STORE_PASS" "$STORE_FAIL"
    return "$_rc"
  fi

  printf '\n%s\n' "$STORE_ROWS"

  # ⛔ THE FAILING ROWS ARE REPRINTED AT THE END, AND THAT IS NOT DECORATION.
  # check-gate.sh shows an excerpt of a failed check's log, and a harness like
  # this prints dozens of passing rows before the one that failed. Measured on
  # 2026-09-08: a red Linux lane reported `FAIL check-capture (exit 1)` followed
  # by eleven PASSING rows, so the CI log did not contain the failure at all and
  # the defect had to be reproduced locally to be seen. A report whose failure
  # falls off the end of the excerpt does not say what went wrong.
  if [ "$STORE_FAIL" -gt 0 ]; then
    printf 'the case(s) that failed:\n'
    printf '%s' "$STORE_ROWS" | grep -F -e '❌' || true
  fi

  printf '%s %s: %s passed, %s failed\n' "$_total" "$2" "$STORE_PASS" "$STORE_FAIL"
  if [ "$STORE_PASS" -eq 0 ]; then
    printf -- '❌ NOTHING RAN. Zero cases passed, so this is red whatever else it says.\n'
  elif [ "$STORE_FAIL" -gt 0 ]; then
    printf -- '❌ a guard did not refuse its defect.\n'
  else
    printf -- '✅ every planted defect was refused, and the clean tree was not.\n'
  fi
  return "$_rc"
}
