#!/bin/sh
# check-bitcheck.sh - does the Go checking binary refuse what it must, and does
# it answer exactly what the shell halves answered over the same planted defect?
#
# CI-10 ports the checking layer to one Go binary and DELETES the twin layer
# rather than translating it. ⛔ The entry's own bound on that is why this file
# exists: *a ported check refuses exactly what its shell half refused, over the
# same plants, with the same 0/1/2 vocabulary - and is compared case for case
# against those halves BEFORE they are deleted.*
#
# -- ⭐ ONE CASE LIST, TWO MODES, AND THAT IS THE WHOLE DESIGN ---------------
#
# The case list is the value here and it is written once. What changes is how
# many implementations each case is asked of:
#
#   default     the plants are run against the Go binary alone. Milliseconds.
#               ⭐ THIS IS THE PERMANENT GATE ROW, and it goes on being one after
#               every shell half has been deleted, because a plant does not need
#               a second implementation to be a plant.
#   --compare   each case additionally runs the .sh half and the .ps1 half and
#               refuses any difference in exit code or in what --json prints,
#               stdout and stderr together - one line for every check but
#               check-remote-items, which keeps its report on stderr.
#               ⛔ THIS IS THE PRE-DELETION PROOF and it is deliberately NOT in
#               the gate: check-control-bytes.ps1 alone is 67.9 seconds, so
#               running it once per case would cost more than the twin layer this
#               entry exists to remove.
#
# ⚠ A Go harness compared only against itself is the self-consistency OBS-07 was
# opened for, arriving in the checking layer. --compare is the answer to that,
# and TODO/ci.md records the run rather than leaving it to a session to repeat.
#
# -- ⛔ A PORTED HARNESS'S CASES ARE NOT IN THE DEFAULT MODE ------------------
#
#   --harnesses the default mode plus the cases for ported HARNESSES, which plant
#               into what a harness reads and so have to build its Rust subject
#               out of this scratch tree. --compare runs them too.
#
# ⛔ MEASURED, NOT ASSUMED: the build is cold here, because the scratch tree
# needs a target directory of its own - sharing a caller's would let a planted
# example be uplifted over the binary another gate row is executing. On
# 2026-09-29 it made this row 39 seconds where it had been 23, and the whole
# gate 193 where the same host measured 168 without it. `check-workflow` runs
# the gate about nine times per shard, so that is minutes per shard for plants
# that do not change between pushes, which is `check-defaults`' reason for
# being a step of its own rather than a row. The default mode says what it did
# not run, on stderr, rather than passing a row for it.
#
# -- ⛔ WHY A CLEAN TREE PROVES ALMOST NOTHING HERE --------------------------
#
# check-twins.sh compares two halves' ANSWERS on the tree it runs against, and
# its own header records what that misses. Measured on 2026-08-28 with three
# divergences planted in a .ps1 half: a changed ceiling was caught, a dropped
# `md` scope was caught, and a dropped `py` scope was INVISIBLE, because this
# tree holds no .py file and the smaller scope produced an identical number.
#
# ⭐ So every case below is a PLANT, and the clean tree is a control either side
# rather than the measurement. A rule that differs only on a defect the tree does
# not contain is exactly what this is built to see.
#
# ⚠ AND THAT BLIND SPOT REPRODUCED HERE ON ITS FIRST MUTATION PASS. Adding DEL to
# the Go control class left every case green, because nothing in the tree and
# nothing planted carried the byte. The repair is the fixture below, not a
# reading, and the case says so where it sits.
#
# -- ⚠ A PLANT THE CHECKS ALL ACCEPT IS STILL A CASE ------------------------
#
# Half the cases here plant something the rule must NOT fire on: a specimen
# inside a fenced block, a leading byte-order mark, a control byte in a file that
# is not text. Those are where a port is likeliest to be over-strict, and a
# harness carrying only refusals would never look at them.
#
# Usage:
#   sh scripts/common/check-bitcheck.sh
#   sh scripts/common/check-bitcheck.sh --harnesses
#   sh scripts/common/check-bitcheck.sh --compare
#   sh scripts/common/check-bitcheck.sh --json
#
# Exit codes: 0 every case held, 1 one did not, 2 could not run.
#
# ⛔ Read the exit code from this process, unpiped.

set -u

JSON=0
COMPARE=0
HARNESSES=0
while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON=1 ;;
    --compare)
      COMPARE=1
      HARNESSES=1
      ;;
    --harnesses) HARNESSES=1 ;;
    -h | --help)
      awk 'NR>1 { if (/^#/) { sub(/^# ?/, ""); print } else exit }' "$0"
      exit 0
      ;;
    *)
      printf 'check-bitcheck: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

# ⛔ Resolved from this script's own location, never from the working directory.
HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH='' cd -- "$HERE/../.." && pwd)

# shellcheck disable=SC2034
ME=check-bitcheck
# shellcheck disable=SC1091
# shellcheck source=scripts/corpus/store-lib.sh
. "$ROOT/scripts/corpus/store-lib.sh"

# ⚠ jq IS REQUIRED because the gh stub below answers `--jq` with it and the
# mine-repo cases compare JSON through it. Without it those cases would FAIL
# rather than say they could not run, which is the wrong one of the two.
store_require git go tar jq

WORK=$(store_workdir checkbitcheck) || exit 2
trap 'rm -rf "$WORK"' EXIT INT TERM

TREE="$WORK/tree"
BIN="$WORK/bit-check"

# ⭐ THE BINARY IS BUILT FROM THE TREE ON DISK, not from a release. A harness
# comparing a stale binary against fresh shell halves would report a drift that
# the tree does not have, or miss one it does.
if ! (cd "$ROOT/tools/check" && go build -o "$BIN" .) >"$WORK/build.log" 2>&1; then
  printf 'check-bitcheck: cannot build tools/check\n' >&2
  sed 's/^/  /' "$WORK/build.log" >&2
  exit 2
fi

# ⚠ pwsh IS OPTIONAL AND ITS ABSENCE IS SAID RATHER THAN ASSUMED. On a host
# without it the PowerShell column simply is not compared, and the row says so,
# because a comparison that silently dropped a participant is a comparison
# reporting agreement between one implementation and itself.
HAVE_PWSH=0
command -v pwsh >/dev/null 2>&1 && HAVE_PWSH=1

# -- the scratch tree ---------------------------------------------------------
#
# ⚠ Tracked AND untracked-but-not-ignored, because an uncommitted new file is
# part of the tree the next push carries, and it is in scope for every rule here.
mkdir -p "$TREE" || exit 2
(
  cd "$ROOT" || exit 1
  {
    git ls-files -z
    git ls-files -zo --exclude-standard
  } | tar --null -T - -cf -
) | (cd "$TREE" && tar -xf -) || {
  printf 'check-bitcheck: cannot copy the working tree\n' >&2
  exit 2
}

# ⛔ IT IS A REAL REPOSITORY, because every check here asks git what is in the
# tree. A plain directory would make all three implementations exit 2 and the
# whole run would be three harnesses agreeing that they could not run.
(
  cd "$TREE" || exit 1
  git init -q -b main &&
    git add -A &&
    git -c user.email=gate -c user.name=gate commit -qm scratch
) >/dev/null 2>&1 || {
  printf 'check-bitcheck: cannot make the scratch tree a repository\n' >&2
  exit 2
}

# -- running one implementation ----------------------------------------------
#
# ⛔ EACH EXIT CODE IS READ FROM THE PROCESS THAT PRODUCED IT, UNPIPED. The
# output goes to a file and the status is read on the next line. This whole
# harness is about comparing statuses, so a pipeline's status here would compare
# three zeroes and call it agreement.
OUTF="$WORK/out"

# ⚠ THE MODE FLAGS GO TO THE GO HALF VERBATIM, and that is a property of the
# binary rather than a convenience here: `--public` is spelled the same way the
# `sh` half spells it, so a case states one flag and two of the three
# implementations take it unchanged. Only PowerShell needs a translation, and
# `agree` does that one below rather than deriving all three from a label.
run_go() { # check flags...
  _c=$1
  shift
  (cd "$TREE" && "$BIN" "$_c" --json "$@") >"$OUTF" 2>&1
  printf '%s\t%s\n' "$?" "$(cat "$OUTF")"
}

run_sh() { # relative-path flags...
  _p=$1
  shift
  (cd "$TREE" && sh "$TREE/scripts/$_p.sh" "$@") >"$OUTF" 2>&1
  printf '%s\t%s\n' "$?" "$(cat "$OUTF")"
}

run_ps() { # relative-path flags...
  _p=$1
  shift
  (cd "$TREE" && pwsh -NoProfile -File "$TREE/scripts/$_p.ps1" "$@") >"$OUTF" 2>&1
  printf '%s\t%s\n' "$?" "$(cat "$OUTF")"
}

# -- the comparison -----------------------------------------------------------
#
# ⛔ THE EXPECTED CODE IS AN ARGUMENT, AND THAT IS THE HALF THAT MAKES THIS A
# PROOF. Three implementations that all answer 0 over a plant agree perfectly and
# have all missed it. So a case states what the plant is supposed to do, and a
# unanimous wrong answer is a failure rather than agreement.
AGREE_GO_JSON=""
agree() { # label check sh-rel expected-code [sh-flags...]
  _label=$1
  _check=$2
  _rel=$3
  _want=$4
  shift 4

  _g=$(run_go "$_check" "$@")
  _gc=${_g%%	*}
  _gj=${_g#*	}
  # ⚠ Kept for `go_said` below, and named so no case reads agree's own locals.
  AGREE_GO_JSON=$_gj

  if [ "$_gc" != "$_want" ]; then
    fail "$_label: go answered $_gc, expected $_want"
    return
  fi

  # ⭐ THE DEFAULT MODE STOPS HERE, and it is still a proof. The plant landed, the
  # Go binary answered the code the case named, and no shell process was started.
  # ⚠ What it does NOT establish is equivalence to the halves; --compare is the
  # only thing that does, and the row says which of the two it ran.
  if [ "$COMPARE" = "0" ]; then
    pass "$_label: go (exit $_want)"
    return
  fi

  # ⛔ A RULE THAT NEVER HAD A SHELL HALF IS NOT A RULE WHOSE HALF AGREED, and it
  # is not one whose half was deleted either. `-` says so explicitly rather than
  # letting a new Go-only rule borrow the wording of a completed port: there is
  # no predecessor to compare, which is a different fact from having outlived one.
  if [ "$_rel" = "-" ]; then
    pass "$_label: go (exit $_want); a Go-only rule, so there is no half to compare"
    return
  fi

  # ⛔ A HALF THAT IS GONE IS NOT A HALF THAT AGREED. Once a twin is deleted this
  # case has one fewer implementation to compare, and saying so in the row is the
  # difference between a comparison that shrank and one that passed.
  if [ ! -f "$TREE/scripts/$_rel.sh" ]; then
    pass "$_label: go (exit $_want); the sh half is gone, not compared"
    return
  fi

  _s=$(run_sh "$_rel" --json "$@")
  _sc=${_s%%	*}
  _sj=${_s#*	}

  if [ "$_sc" != "$_want" ]; then
    fail "$_label: sh answered $_sc, expected $_want"
    return
  fi
  if [ "$_gj" != "$_sj" ]; then
    fail "$_label: go and sh disagree
$(disagreement go "$_gj" sh "$_sj")"
    return
  fi

  if [ "$HAVE_PWSH" = "0" ] || [ ! -f "$TREE/scripts/$_rel.ps1" ]; then
    pass "$_label: go and sh agree (exit $_want); no PowerShell half compared"
    return
  fi

  # ⚠ The PowerShell flag spelling differs from the sh one, so it is passed
  # rather than derived. A label built from a flag made two runners' row lists
  # differ on a row they both have, which check-gate.sh records.
  _pflags=""
  for _f in "$@"; do
    case "$_f" in
      --public) _pflags="-Public" ;;
    esac
  done
  # shellcheck disable=SC2086
  _p=$(run_ps "$_rel" -Json $_pflags)
  _pc=${_p%%	*}
  _pj=${_p#*	}

  if [ "$_pc" != "$_want" ]; then
    fail "$_label: pwsh answered $_pc, expected $_want"
    return
  fi
  if [ "$_gj" != "$_pj" ]; then
    fail "$_label: go and pwsh disagree
$(disagreement go "$_gj" pwsh "$_pj")"
    return
  fi
  pass "$_label: all three agree (exit $_want)"
}

# ⚠ A DISAGREEMENT IS SHOWN AS A DIFF, INDENTED UNDER ITS ROW. Every check here
# used to answer one JSON line, and printing both lines inline was enough; one
# whose --json mode keeps a report answers many, and an inline copy of two reports
# both buried the difference and put report lines where `store_report` counts
# row starts - so the run with something to say refused to say anything, which
# is the defect that report's own comment records. ⛔ Every line is indented
# deeper than a row start, whatever the reports contain.
disagreement() { # left-name left right-name right
  printf '%s\n' "$2" >"$WORK/left"
  printf '%s\n' "$4" >"$WORK/right"
  diff -u --label "$1" --label "$3" "$WORK/left" "$WORK/right" | sed 's/^/          /'
}

# ⛔ AN EXIT CODE OF 1 SAYS A HARNESS FAILED A ROW AND NOT WHICH. A ported
# harness answers 1 for one failed row and for six, so two plants aimed at
# different rows are indistinguishable by code alone - and a harness whose rows
# had been rewired would pass both. This asserts the whole line the Go binary
# gave for the case just run, in the default mode as well, and costs no second
# run. ⚠ Under --compare, `agree` has already required the halves to print the
# same line, so one expectation here covers all three.
#
# ⚠ THE VERDICT IS THE LAST LINE, NOT THE WHOLE OUTPUT. check-remote-items keeps
# a report for a person on stderr in --json mode, ahead of the document, so the
# combined output is many lines; comparing all of it put report lines into a
# failure message, and `store_report` rightly refused a report whose rows it
# could no longer count. `agree` still compares everything under --compare.
go_said() { # label expected-json
  _said=$(printf '%s\n' "$AGREE_GO_JSON" | tail -n 1)
  if [ "$_said" = "$2" ]; then
    pass "$1: go said $2"
  else
    fail "$1: go said $_said, expected $2"
  fi
}

# -- planting -----------------------------------------------------------------
#
# ⛔ A PLANT IS REMOVED BY THE CASE THAT MADE IT, and the control after it is what
# says the removal worked. A harness whose plants accumulate measures the sum of
# everything it has done so far, and every case after the first is then reporting
# a tree nobody chose.
unplant() { # relative-path
  rm -f "$TREE/$1"
}

# ⛔ SOME PLANTS HAVE TO BE TRACKED, AND .gitignore IS WHY. check-no-secrets'
# first rule fires on a credential FILE, and this repository ignores most of the
# names it knows - `*.pem`, `*.key`, `id_rsa` - so a plant left untracked is
# ignored, out of scope, and the case would report a rule holding over a file it
# never saw. ⭐ `git add -f` is also the truer fixture: the rule's own title is
# *a credential file is TRACKED*, which is what happens when a name is force-added
# or was added before the ignore rule existed.
plant_tracked() { # relative-path
  (cd "$TREE" && git add -f -- "$1")
}

# ⛔ AND IT LEAVES THE INDEX AS IT FOUND IT. Deleting the file alone leaves a
# tracked-but-deleted path, which `git ls-files` still reports - and rule 1 reads
# a PATH rather than a file, so the finding would survive the unplant and every
# case after it would be measuring a tree nobody chose.
unplant_tracked() { # relative-path
  (cd "$TREE" && git rm -q --cached --ignore-unmatch -- "$1")
  rm -f "$TREE/$1"
}

# ============================================================================
# check-control-bytes
# ============================================================================

agree "control-bytes clean tree" check-control-bytes common/check-control-bytes 0

# ⛔ A C0 CONTROL BYTE IN A TEXT FILE. The byte is written by printf rather than
# as a literal, which is this repository's own rule: a literal control byte in
# this harness would make THIS file the thing it exists to refuse.
printf 'plant\001here\n' >"$TREE/tools/check/plant.txt"
agree "control-bytes a C0 byte is refused" check-control-bytes common/check-control-bytes 1
unplant tools/check/plant.txt

# ⛔ NUL IS A SEPARATE TEST, because it cannot live in a shell variable and the
# shell half therefore found it by a different route. A port that folded the two
# would pass every case above and miss the commonest offender there is.
printf 'plant\000here\n' >"$TREE/tools/check/plant.txt"
agree "control-bytes a NUL byte is refused" check-control-bytes common/check-control-bytes 1
unplant tools/check/plant.txt

# ⭐ THE THREE THAT ARE LEGITIMATELY IN TEXT. This is the case a port is likeliest
# to get wrong in the strict direction, and a harness of refusals alone would
# never look at it.
printf 'tab\there\r\nand a return\n' >"$TREE/tools/check/plant.txt"
agree "control-bytes tab, CR and LF are accepted" check-control-bytes common/check-control-bytes 0
unplant tools/check/plant.txt

# ⛔ DEL IS OUTSIDE THE CLASS, AND THIS CASE EXISTS BECAUSE A MUTATION SURVIVED
# WITHOUT IT. Measured on 2026-09-10 while proving this harness: adding 0x7f to
# the Go class left every case above green and the run exited 0, because nothing
# in the tree and nothing planted carried the byte. ⚠ That is check-twins' own
# documented blind spot arriving in its replacement - a rule that differs only on
# a defect the tree does not contain is invisible to any comparison over it - and
# the repair is a fixture rather than a reading.
printf 'plant\177here\n' >"$TREE/tools/check/plant.txt"
agree "control-bytes DEL is outside the class and accepted" check-control-bytes common/check-control-bytes 0
unplant tools/check/plant.txt

# ⚠ SCOPE IS PROVED WITH A FIXTURE RATHER THAN TRUSTED. The same byte in a file
# the extension list does not call text must be accepted, and a scope rule is
# invisible to any comparison over a tree that lacks the shape.
printf 'plant\001here\n' >"$TREE/tools/check/plant.bin"
agree "control-bytes a byte outside the text extensions is out of scope" check-control-bytes common/check-control-bytes 0
unplant tools/check/plant.bin

agree "control-bytes clean tree again" check-control-bytes common/check-control-bytes 0

# ============================================================================
# check-markers
# ============================================================================

agree "markers clean tree" check-markers common/check-markers 0

# An em dash, U+2014, which is outside the five and is the character the rule
# was originally written about.
DASH=$(printf '\342\200\224')
STOP=$(printf '\342\233\224')

printf 'a plant %s here\n' "$DASH" >"$TREE/tools/check/plant.md"
agree "markers a character outside the five is refused" check-markers common/check-markers 1
unplant tools/check/plant.md

# ⭐ THE SPECIMEN EXEMPTION, and it is markdown only. Without it the rule is
# unwritable: a page that bans a character cannot show a reader which character
# it means.
{
  printf 'text\n\n'
  printf '```\n'
  printf 'a specimen %s inside a fence\n' "$DASH"
  printf '```\n'
} >"$TREE/tools/check/plant.md"
agree "markers a specimen inside a fenced block is accepted" check-markers common/check-markers 0
unplant tools/check/plant.md

# shellcheck disable=SC2016
# The backticks are the SUBJECT here rather than a substitution: this case exists
# to put a specimen inside an inline code span. ⚠ The directive is in this file
# rather than in the invocation, because shellcheck answers differently depending
# on how files were grouped on its command line and a contributor checking one
# file does not group them the way CI does.
printf 'an inline `%s` span\n' "$DASH" >"$TREE/tools/check/plant.md"
agree "markers a specimen inside a code span is accepted" check-markers common/check-markers 0
unplant tools/check/plant.md

# ⛔ AND THERE IS NO EXEMPTION OUTSIDE MARKDOWN. A source file has no reader who
# needs a specimen, so the same bytes in a .sh are refused - which is the branch
# that separates the two scopes.
printf '# a plant %s here\n' "$DASH" >"$TREE/tools/check/plant.sh"
agree "markers the same character in a .sh is refused" check-markers common/check-markers 1
unplant tools/check/plant.sh

# ⚠ A LEADING BYTE-ORDER MARK IS EXEMPT AND ONLY A LEADING ONE. Both directions
# are planted, because they are different branches and a port that stripped every
# BOM would pass the first and lose the second.
printf '\357\273\277ascii only\n' >"$TREE/tools/check/plant.md"
agree "markers a leading BOM is accepted" check-markers common/check-markers 0
unplant tools/check/plant.md

printf 'first line\n\357\273\277second line\n' >"$TREE/tools/check/plant.md"
agree "markers a BOM anywhere else is refused" check-markers common/check-markers 1
unplant tools/check/plant.md

# ⛔ THE DENSITY CEILING IS A SEPARATE RULE OVER THE SAME FILES, and a file made
# entirely of permitted characters is how it is reached: every byte here is one
# of the five, so the character rule holds and only the count refuses it.
{
  printf 'a line with markers %s%s%s%s%s%s%s%s%s%s\n' "$STOP" "$STOP" "$STOP" \
    "$STOP" "$STOP" "$STOP" "$STOP" "$STOP" "$STOP" "$STOP"
} >"$TREE/tools/check/plant.md"
agree "markers a file over the density ceiling is refused" check-markers common/check-markers 1
unplant tools/check/plant.md

# ⛔ LICENSES/*.txt IS EXEMPT AND THIS TREE HAS NO SUCH FILE, so the exemption is
# proved with a fixture. The lesson comes from check-markers' own deleted shell
# header: a `py` scope dropped from one half was invisible to check-twins because
# the tree held no .py file at all. TODO/ci.md carries it under CI-10.
mkdir -p "$TREE/LICENSES"
printf 'a canonical text with a dash %s in it\n' "$DASH" >"$TREE/LICENSES/PLANT.txt"
agree "markers LICENSES/*.txt is exempt" check-markers common/check-markers 0
rm -rf "$TREE/LICENSES"

# ⚠ AND THE EXEMPTION IS THE DIRECTORY AND THE EXTENSION TOGETHER. The same bytes
# under LICENSES with a different extension are in scope, which is what says the
# rule is anchored rather than matching the word anywhere in a path.
mkdir -p "$TREE/LICENSES"
printf 'a dash %s in a markdown file\n' "$DASH" >"$TREE/LICENSES/PLANT.md"
agree "markers LICENSES/*.md is NOT exempt" check-markers common/check-markers 1
rm -rf "$TREE/LICENSES"

agree "markers clean tree again" check-markers common/check-markers 0

# ============================================================================
# check-one-home
# ============================================================================

agree "one-home clean tree" check-one-home common/check-one-home 0

# A sentence of twelve or more words, which is the length below which a repeated
# phrase is not a fact.
LONG='the capture host is claimed before anything else writes to it and the route is cut afterwards'
SHORT='the capture host is claimed first'

printf '%s\n' "$LONG" >"$TREE/tools/check/plant-a.md"
printf '%s\n' "$LONG" >"$TREE/tools/check/plant-b.md"
agree "one-home one sentence in two documents is refused" check-one-home common/check-one-home 1
unplant tools/check/plant-a.md
unplant tools/check/plant-b.md

# ⚠ THE SAME SENTENCE TWICE IN ONE DOCUMENT IS NOT A DUPLICATE. The shell half
# ran `sort -u` over (sentence, file) pairs before counting, so this is the case
# that separates "in two documents" from "twice". A port that counted occurrences
# rather than distinct files would refuse every document that repeats a heading.
printf '%s\n\n%s\n' "$LONG" "$LONG" >"$TREE/tools/check/plant-a.md"
agree "one-home the same sentence twice in ONE document is accepted" check-one-home common/check-one-home 0
unplant tools/check/plant-a.md

# ⚠ UNDER THE WORD FLOOR, SHARED. A short phrase in two documents is a phrase,
# not a fact, and refusing it would fire on every correct tree.
printf '%s\n' "$SHORT" >"$TREE/tools/check/plant-a.md"
printf '%s\n' "$SHORT" >"$TREE/tools/check/plant-b.md"
agree "one-home a sentence under the word floor is accepted" check-one-home common/check-one-home 0
unplant tools/check/plant-a.md
unplant tools/check/plant-b.md

# ⛔ ROUTERS ARE EXEMPT AS A SET AND THIS TREE CARRIES NEITHER OF THEM at the
# paths the rule names - it has docs/AGENTS.md, which is NOT in that set - so the
# exemption is proved with a fixture rather than assumed from the tree.
printf '%s\n' "$LONG" >"$TREE/AGENTS.md"
printf '%s\n' "$LONG" >"$TREE/ROUTE.md"
agree "one-home a sentence shared only between routers is accepted" check-one-home common/check-one-home 0
unplant AGENTS.md

# ⛔ AND THE EXEMPTION IS *EVERY* FILE, NOT *ANY*. With one router and one
# ordinary document the same sentence is the defect, which is the branch that
# says the test is over the whole set.
printf '%s\n' "$LONG" >"$TREE/tools/check/plant-a.md"
agree "one-home a router sharing with an ordinary document is refused" check-one-home common/check-one-home 1
unplant tools/check/plant-a.md
unplant ROUTE.md

# ⚠ docs/history/ IS OUT OF SCOPE, because superseded reasoning is where a
# repeated sentence is correct: the history says what was believed then and the
# current document says what is true now.
printf '%s\n' "$LONG" >"$TREE/tools/check/plant-a.md"
mkdir -p "$TREE/docs/history"
printf '%s\n' "$LONG" >"$TREE/docs/history/PLANT.md"
agree "one-home docs/history is out of scope" check-one-home common/check-one-home 0
unplant tools/check/plant-a.md
unplant docs/history/PLANT.md

# ⭐ A SHARED SENTENCE INSIDE A FENCED BLOCK IS NOT A SHARED SENTENCE. Two
# documents quoting one command are quoting it, and the fence is what says so.
for p in plant-a plant-b; do
  {
    printf '```\n'
    printf '%s\n' "$LONG"
    printf '```\n'
  } >"$TREE/tools/check/$p.md"
done
agree "one-home a sentence shared inside fenced blocks is accepted" check-one-home common/check-one-home 0
unplant tools/check/plant-a.md
unplant tools/check/plant-b.md

agree "one-home clean tree again" check-one-home common/check-one-home 0

# ============================================================================
# check-changelog
# ============================================================================

agree "changelog clean tree" check-changelog common/check-changelog 0

CL="$TREE/CHANGELOG.md"
cp "$CL" "$WORK/CHANGELOG.orig" || exit 2
restore_changelog() { cp "$WORK/CHANGELOG.orig" "$CL"; }

# ⛔ THE ORDERING RESETS AT EVERY SECTION, and this is the control for that: an
# Unreleased section above a versioned one is CORRECT even though its date is
# older, so a port comparing across the boundary refuses a right answer.
{
  printf '# Changelog\n\n## Unreleased\n\n'
  printf '### 2026-01-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n\n'
  printf '## 1.0.0\n\n'
  printf '### 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n'
} >"$CL"
agree "changelog an older date in a later section is accepted" check-changelog common/check-changelog 0

{
  printf '# Changelog\n\n## Unreleased\n\n'
  printf '### 2026-01-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n\n'
  printf '### 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n'
} >"$CL"
agree "changelog two entries out of order in ONE section is refused" check-changelog common/check-changelog 1

{
  printf '# Changelog\n\n## Unreleased\n\n'
  printf '### the day it happened\n\n- Record: TODO/ci.md. No deploy.\n'
} >"$CL"
agree "changelog an entry with no date is refused" check-changelog common/check-changelog 1

{
  printf '# Changelog\n\n## Unreleased\n\n'
  printf '### 2026-06-01T00:00:00Z\n\n- Something happened. No deploy.\n'
} >"$CL"
agree "changelog an entry naming no record is refused" check-changelog common/check-changelog 1

{
  printf '# Changelog\n\n## Unreleased\n\n'
  printf '### 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md.\n'
} >"$CL"
agree "changelog an entry silent about deployment is refused" check-changelog common/check-changelog 1

# ⛔ A FILE THAT PARSES TO NO ENTRIES IS A FAILURE, NOT A CLEAN RUN, and this
# repository shipped exactly that shape for as long as it took somebody to
# notice. ⚠ It is the one refusal here that emits no JSON at all - there is no
# `problems` count to report - so it is also the case that says the port carried
# the stderr path rather than inventing an answer.
{
  printf '# Changelog\n\n## 2026-06-01T00:00:00Z\n\n- Record: TODO/ci.md. No deploy.\n'
} >"$CL"
agree "changelog a file with no entries the parser knows is refused" check-changelog common/check-changelog 1

restore_changelog
agree "changelog clean tree again" check-changelog common/check-changelog 0

# ============================================================================
# check-licences
# ============================================================================

agree "licences clean tree" check-licences common/check-licences 0

REG="$TREE/catalogue/licences.toml"
CAT="$TREE/catalogue/clients.toml"
cp "$REG" "$WORK/licences.orig" || exit 2
cp "$CAT" "$WORK/clients.orig" || exit 2
restore_licences() {
  cp "$WORK/licences.orig" "$REG"
  cp "$WORK/clients.orig" "$CAT"
}

# ⛔ A CATALOGUE TARGET WITH NO REGISTER ROW. The comparison runs in both
# directions and this is the direction that matters: a target added and its
# licence forgotten is a build nobody decided the disposition of.
{
  cat "$WORK/clients.orig"
  printf '\n[[targets]]\nid = "zz-plant"\nopen_source = true\n'
} >"$CAT"
agree "licences a catalogue target with no register row is refused" check-licences common/check-licences 1
restore_licences

# ⚠ AND THE OTHER DIRECTION, which is a different branch and would survive a
# check that only walked the catalogue.
{
  cat "$WORK/licences.orig"
  printf '\n[[targets]]\nid = "zz-plant"\nlicence = "MIT"\nlicence_source = "x"\nredistribute = "refused"\n'
} >"$REG"
agree "licences a register row naming no catalogue target is refused" check-licences common/check-licences 1
restore_licences

# ⛔ PERMITTED IS THE EXPENSIVE VALUE AND IT HAS TO BE EARNED. `unverified` plus
# `permitted` is the combination that would publish somebody's bytes on nobody's
# authority, and it is a separate rule from "has a disposition at all".
sed 's/^redistribute = "refused"$/redistribute = "permitted"/' "$WORK/licences.orig" >"$REG"
agree "licences permitted without a verified licence and a notice is refused" check-licences common/check-licences 1
restore_licences

# ⛔ A REGISTER OF NOTHING IS REFUSED - AND THIS CASE DOES NOT ESTABLISH BY WHICH
# RULE, WHICH IS A FINDING RATHER THAN A DETAIL. Measured on 2026-09-10 by
# planting: with the empty-register refusal removed outright, every case here
# still passed. The plant is still refused, because a register with no rows also
# has no row for any catalogue target and rule 1 fires on that.
#
# ⚠ SO THE EMPTY-REGISTER GUARD IS DEFENCE IN DEPTH AND ITS REFUSAL IS NEVER
# ALONE. It is unreachable as a sole cause over a real catalogue: any register
# empty enough to trigger it has already failed the two comparisons. It is kept
# and recorded rather than removed, the way CI-03 keeps a marker's exclusive
# create - a guard nothing can refute is still the guard that matters if a parser
# ever stops matching. ⛔ What this case is NOT is proof that it works, and the
# label says so instead of claiming it.
printf 'schema = "bit-ids/licences/1"\n' >"$REG"
agree "licences an empty register is refused, by rule 1 rather than by the empty guard" check-licences common/check-licences 1
restore_licences

agree "licences clean tree again" check-licences common/check-licences 0

# ============================================================================
# check-cache
# ============================================================================
#
# ⭐ THE FIRST HARNESS TO LEAVE THE TWIN LAYER, and its plants go into what the
# harness READS - the register and the scenario it drives - rather than into the
# harness. A harness's own refusal branches are otherwise exercised by nothing:
# over the real tree every one of its rows passes.
#
# ⛔ CARGO_TARGET_DIR IS THIS RUN'S OWN from here on. Every implementation builds
# `cache-scenario` out of THIS scratch tree, so the build is paid once, by
# whichever asks first - and it never lands in a target directory another check
# of the same gate is executing examples out of, which a variable exported by
# the caller would otherwise arrange.
CARGO_TARGET_DIR="$WORK/target"
export CARGO_TARGET_DIR

if [ "$HARNESSES" = "0" ]; then
  printf 'check-bitcheck: the check-cache cases were NOT run; --harnesses or --compare runs them\n' >&2
else

  agree "cache clean tree" check-cache acquisition/check-cache 0
  go_said "cache clean tree" '{"schema":"check-cache/1","total":13,"passed":13,"failed":0}'

  # ⛔ THE REGISTER PERMITS THE SCENARIO'S OWN TARGET. Three rows must fire and
  # only three: the register row, the E-CAC-01 refusal that no longer appears, and
  # the control whose two runs are now the same run.
  awk '/^id = "aria2"$/ { a = 1 }
    a && /^redistribute = "refused"$/ { print "redistribute = \"permitted\""; a = 0; next }
    { print }' "$WORK/licences.orig" >"$REG"
  agree "cache a register permitting the scenario's target" check-cache acquisition/check-cache 1
  go_said "cache a register permitting the scenario's target" '{"schema":"check-cache/1","total":13,"passed":10,"failed":3}'
  restore_licences

  # ⚠ AND A REGISTER PERMITTING SOMETHING ELSE, which only the register row may
  # notice: the scenario's target is still refused, so a harness that let this
  # plant fail any other row is reading the register twice.
  awk '/^id = "aria2-next"$/ { a = 1 }
    a && /^redistribute = "refused"$/ { print "redistribute = \"permitted\""; a = 0; next }
    { print }' "$WORK/licences.orig" >"$REG"
  agree "cache a register permitting another target" check-cache acquisition/check-cache 1
  go_said "cache a register permitting another target" '{"schema":"check-cache/1","total":13,"passed":12,"failed":1}'
  restore_licences

  # ⛔ A REGISTER THAT CANNOT BE READ IS COULD-NOT-RUN, never an empty permitted
  # list: that would report every target refused, which looks safe and is still a
  # verdict nobody measured.
  rm -f "$REG"
  agree "cache a missing register cannot run" check-cache acquisition/check-cache 2
  restore_licences

  # ⛔ THE SCENARIO'S SOURCE DID NOT MOVE. A retrieval is its location AND its
  # time, so the second one is made an exact repeat of the first: the cache
  # records one retrieval, the scenario refuses itself, and every row that reads
  # its output must say so - six of them. ⚠ Moving the location alone was tried
  # first and changed nothing, because a repeat at a new time is a new retrieval:
  # a plant that did not produce the defect it names, found by its case going
  # green.
  SCEN="$TREE/crates/bit-ids/examples/cache-scenario.rs"
  cp "$SCEN" "$WORK/scenario.orig" || exit 2
  if replace_once "$SCEN" '"https://example.invalid/archive/2026/aria2.tar.gz"' \
    '"https://example.invalid/downloads/v1/aria2.tar.gz"' &&
    replace_once "$SCEN" '"2026-06-18T09:00:00Z"' '"2026-01-04T09:00:00Z"'; then
    agree "cache a retrieval that repeats the first" check-cache acquisition/check-cache 1
    go_said "cache a retrieval that repeats the first" '{"schema":"check-cache/1","total":13,"passed":7,"failed":6}'
  else
    fail "cache a retrieval that repeats the first: NOT PLANTED"
  fi
  cp "$WORK/scenario.orig" "$SCEN"

  agree "cache clean tree again" check-cache acquisition/check-cache 0

fi

# ============================================================================
# check-catalogue
# ============================================================================
#
# ⭐ THE SECOND HARNESS, and its plants go where the harness reads: the crate it
# sweeps for sockets, the manifest it reads for a transport, and the library it
# drives. The publication itself it builds and plants into on its own.
#
# ⛔ THE CASE-VARIANT PLANTS ARE THE ONES A CLEAN TREE CANNOT SEE. The `sh` half
# matches with `grep`, which is case-sensitive, and its twin with `Select-String`
# and `-match`, which are not - so the two agree over this tree, which spells
# every needle one way, and part over a needle spelled another.
if [ "$HARNESSES" = "0" ]; then
  printf 'check-bitcheck: the check-catalogue cases were NOT run; --harnesses or --compare runs them\n' >&2
else

  agree "catalogue clean tree" check-catalogue publishing/check-catalogue 0
  go_said "catalogue clean tree" '{"schema":"check-catalogue/1","total":18,"passed":18,"failed":0}'

  # ⛔ A SOCKET NAMED IN THE CRATE, in a file nothing compiles - the sweep reads
  # the directory, not the module tree, which is the half a build cannot answer.
  PLANTRS="$TREE/crates/bit-ids/src/zz_planted.rs"
  printf '// planted\nuse std::net::TcpStream;\n' >"$PLANTRS"
  agree "catalogue a socket named in the crate" check-catalogue publishing/check-catalogue 1
  go_said "catalogue a socket named in the crate" '{"schema":"check-catalogue/1","total":18,"passed":17,"failed":1}'
  rm -f "$PLANTRS"

  # ⚠ THE SAME NAME IN ANOTHER CASE IS NOT THE TYPE, and the sweep says so: Rust
  # distinguishes the two and the needles are Rust paths.
  printf '// a tcpstream spelled this way names nothing\n' >"$PLANTRS"
  agree "catalogue a needle in another case is not a socket" check-catalogue publishing/check-catalogue 0
  rm -f "$PLANTRS"

  # ⛔ A TRANSPORT IN THE MANIFEST, under a table cargo ignores so the build
  # still succeeds: the rule reads a line, and a plant that broke the build would
  # be refused by the build rather than by the rule.
  CRATEMF="$TREE/crates/bit-ids/Cargo.toml"
  cp "$CRATEMF" "$WORK/crate-manifest.orig" || exit 2
  printf '\n[package.metadata.zz-plant]\ntokio = "planted"\n' >>"$CRATEMF"
  agree "catalogue a transport in the manifest" check-catalogue publishing/check-catalogue 1
  go_said "catalogue a transport in the manifest" '{"schema":"check-catalogue/1","total":18,"passed":17,"failed":1}'
  cp "$WORK/crate-manifest.orig" "$CRATEMF"

  printf '\n[package.metadata.zz-plant]\nTokio = "planted"\n' >>"$CRATEMF"
  agree "catalogue a transport in another case is not one" check-catalogue publishing/check-catalogue 0
  cp "$WORK/crate-manifest.orig" "$CRATEMF"

  # ⛔ AND THE SOCKET CRATE THE LAB TOOK. Until 2026-10-14 a manifest naming
  # `socket2` was refused by the licence register, which had no row for it; the
  # lab's one door gave it a row, so this rule is what refuses it here now.
  printf '\n[package.metadata.zz-plant]\nsocket2 = "planted"\n' >>"$CRATEMF"
  agree "catalogue the lab's socket crate in the manifest" check-catalogue publishing/check-catalogue 1
  go_said "catalogue the lab's socket crate in the manifest" '{"schema":"check-catalogue/1","total":18,"passed":17,"failed":1}'
  cp "$WORK/crate-manifest.orig" "$CRATEMF"

  # ⛔ THE LIBRARY REFUSES A MOVED RECORD UNDER ANOTHER CODE. Both sites that
  # raise E-LIB-02 are disabled - the manifest comparison fires on every per-file
  # mismatch as well, so disabling one would pass through the other - and the
  # record is still refused, as E-LIB-04: the checksum file no longer covers the
  # bytes. The harness must call that the wrong refusal, and one row fires.
  # ⚠ NAMED FOR WHAT IT DOES. It was first called a library that ACCEPTS a moved
  # record; a mutation pass that disabled the harness's code comparison turned
  # it green, which showed the row had only ever failed on the code.
  CATRS="$TREE/crates/bit-ids/src/catalogue.rs"
  cp "$CATRS" "$WORK/catalogue.rs.orig" || exit 2
  if replace_once "$CATRS" 'if release.manifest_json().as_bytes() != manifest {' \
    'if false && release.manifest_json().as_bytes() != manifest {' &&
    replace_once "$CATRS" 'if found != **digest {' 'if false && found != **digest {'; then
    agree "catalogue a moved record refused under another code" check-catalogue publishing/check-catalogue 1
    go_said "catalogue a moved record refused under another code" '{"schema":"check-catalogue/1","total":18,"passed":17,"failed":1}'
  else
    fail "catalogue a moved record refused under another code: NOT PLANTED"
  fi
  cp "$WORK/catalogue.rs.orig" "$CATRS"

  agree "catalogue clean tree again" check-catalogue publishing/check-catalogue 0

fi

# ============================================================================
# check-remote-items
# ============================================================================
#
# ⭐ THE ONE ROW NO SESSION HOST COULD RUN, RUN HERE WITH NO NETWORK AT ALL. It
# asks an authenticated `gh` about the open items and `curl` for an action's own
# `action.yml`, so a stub of each, first on PATH, serves every answer from files
# this section writes: the issues, the pull requests, a diff, the API's view of a
# commit, a tag and a release, and the runtime a pinned commit declares.
#
# ⛔ THE THREE REAL PINS ARE REAL, and their `runs:` lines are copies of what the
# commits carry, read on 2026-09-29: `actions/checkout` at its v4.2.2 commit
# declares `node20`, at v5.0.0 `node24`, and `astral-sh/setup-uv` at v10.2.0
# declares `"node24"` QUOTED - the spelling the check strips quotes for.
# ⚠ That mattered under --compare while the halves lived. The PowerShell half
# fetched with `Invoke-WebRequest`, which no stub on PATH intercepts, so it read
# the real bytes at those commits while the other two read the copies - and all
# three agreed. Both halves were deleted the same day, so every case here now
# runs the binary alone, and a real pin keeps the copies honest.
#
# ⚠ EVERY FORTY-DIGIT NAME IS ASSEMBLED FROM TWO HALVES, because a contiguous one
# in a tracked file is the shape `check-no-secrets --public` refuses.
STUB="$WORK/stub"
STUBBIN="$WORK/stubbin"
mkdir -p "$STUBBIN" || exit 2

cat >"$STUBBIN/gh" <<'STUBGH'
#!/bin/sh
# A stand-in for gh that serves files from $STUB_GH_DIR and reaches nothing.
set -u
D=${STUB_GH_DIR:?}
case "$1 ${2:-}" in
  "auth status")
    [ -f "$D/auth-fails" ] && {
      printf 'You are not logged into any GitHub hosts.\n' >&2
      exit 1
    }
    exit 0
    ;;
  "issue list")
    [ -f "$D/issues.fail" ] && {
      cat "$D/issues.fail" >&2
      exit 1
    }
    cat "$D/issues.json"
    exit 0
    ;;
  "pr list")
    cat "$D/prs.json"
    exit 0
    ;;
  "pr diff")
    shift 2
    while [ $# -gt 1 ]; do shift; done
    [ -f "$D/diff.$1" ] || {
      printf 'could not find a diff for %s\n' "$1" >&2
      exit 1
    }
    cat "$D/diff.$1"
    exit 0
    ;;
esac
# ⚠ AN API PATH IS A FILE NAMED WITH ITS SLASHES AS `%`, leading slash dropped,
# query kept. mine-repo asks for a repository AND for the lists under it, and a
# nested layout cannot hold `repos/o/r` as a file and a directory at once.
if [ "$1" = "api" ]; then
  shift
  path=""
  jqexpr=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --jq)
        jqexpr=$2
        shift 2
        ;;
      -f | -F)
        shift 2
        ;;
      -*) shift ;;
      *)
        [ -n "$path" ] || path=$1
        shift
        ;;
    esac
  done
  key=$(printf '%s' "${path#/}" | tr '/' '%')
  [ -f "$D/api/$key" ] || {
    printf 'gh: Not Found (HTTP 404)\n' >&2
    exit 1
  }
  if [ -n "$jqexpr" ]; then
    jq -r "$jqexpr" "$D/api/$key"
  else
    cat "$D/api/$key"
  fi
  exit 0
fi
printf 'stub gh: unsupported: %s\n' "$*" >&2
exit 1
STUBGH

cat >"$STUBBIN/curl" <<'STUBCURL'
#!/bin/sh
# A stand-in for curl. It serves raw.githubusercontent.com out of
# $STUB_GH_DIR/raw, and the proxy out of $STUB_GH_DIR/proxy, keyed as the gh
# stub keys a path, and answers what each host answers for a path it lacks.
# ⛔ The proxy refuses any user-agent but curl's own with 420, as the real one
# was measured to on 2026-08-28, so a caller that stops sending it fails here.
set -u
D=${STUB_GH_DIR:?}
url=""
out=""
fmt=""
ua=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o)
      out=$2
      shift 2
      ;;
    -w)
      fmt=$2
      shift 2
      ;;
    -A)
      ua=$2
      shift 2
      ;;
    -m | --max-time)
      shift 2
      ;;
    https://*)
      url=$1
      shift
      ;;
    *) shift ;;
  esac
done
case "$url" in
  https://raw.githubusercontent.com/*)
    rel=${url#https://raw.githubusercontent.com/}
    if [ -f "$D/raw/$rel" ]; then cat "$D/raw/$rel"; else printf '404: Not Found'; fi
    exit 0
    ;;
  https://api.gh.pkgforge.dev/*)
    key=$(printf '%s' "${url#https://api.gh.pkgforge.dev/}" | tr '/' '%')
    src=""
    if [ "$ua" != "curl/8" ]; then
      code=420
    elif [ -f "$D/proxy/$key.code" ]; then
      code=$(cat "$D/proxy/$key.code")
      src="$D/proxy/$key"
    elif [ -f "$D/proxy/$key" ]; then
      code=200
      src="$D/proxy/$key"
    else
      code=404
    fi
    {
      if [ -n "$src" ] && [ -f "$src" ]; then
        cat "$src"
      elif [ "$code" = 404 ]; then
        printf '{"message":"Not Found"}'
      fi
    } >"${out:-/dev/stdout}"
    [ -z "$fmt" ] || printf '%s' "$code"
    exit 0
    ;;
esac
printf 'stub curl: refused %s\n' "$url" >&2
exit 7
STUBCURL
chmod +x "$STUBBIN/gh" "$STUBBIN/curl" || exit 2

V4=11bd71901bbe5b1630ce
V4="${V4}ea73d27597364c9af683"
V5=08c6903cd8c0fde910a3
V5="${V5}7f88322edcfb5dd907a8"
UV=c18668ad3cf93ea998be
UV="${UV}f934396af7bb5c839dc7"
NOSUCH=$(printf '%040d' 0)
TAGOBJ=$(printf '%040d' 7)

stub_reset() {
  rm -rf "$STUB"
  mkdir -p "$STUB/api" "$STUB/raw" "$STUB/proxy" || exit 2
  printf '[]\n' >"$STUB/issues.json"
  printf '[]\n' >"$STUB/prs.json"
}

stub_key() { # path -> the file name both stubs look it up by
  printf '%s' "${1#/}" | tr '/' '%'
}

stub_api() { # path json
  printf '%s\n' "$2" >"$STUB/api/$(stub_key "$1")"
}

stub_runs() { # action sha runs-block
  mkdir -p "$STUB/raw/$1/$2" && printf '%s' "$3" >"$STUB/raw/$1/$2/action.yml"
}

# One open pull request whose diff adds the given line.
stub_pr() { # number diff-line
  printf '[{"number":%s,"title":"a proposal","author":{"login":"a-bot"},"headRefName":"b","files":[{"path":".github/workflows/ci.yml"}]}]\n' \
    "$1" >"$STUB/prs.json"
  printf 'diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml\n+%s\n' "$2" >"$STUB/diff.$1"
}

# What the real repositories answer, as far as these cases ask.
stub_world() {
  stub_api "repos/actions/checkout/commits/$V4" "{\"sha\":\"$V4\"}"
  stub_api "repos/actions/checkout/commits/$V5" "{\"sha\":\"$V5\"}"
  stub_api "repos/actions/checkout/git/ref/tags/v4.2.2" "{\"object\":{\"sha\":\"$V4\",\"type\":\"commit\"}}"
  stub_api "repos/actions/checkout/git/ref/tags/v5.0.0" "{\"object\":{\"sha\":\"$V5\",\"type\":\"commit\"}}"
  stub_api "repos/actions/checkout/releases/latest" '{"tag_name":"v7.0.1"}'
  stub_api "repos/astral-sh/setup-uv/commits/$UV" "{\"sha\":\"$UV\"}"
  stub_api "repos/astral-sh/setup-uv/git/ref/tags/v10.2.0" "{\"object\":{\"sha\":\"$UV\",\"type\":\"commit\"}}"
  stub_api "repos/astral-sh/setup-uv/releases/latest" '{"tag_name":"v10.2.0"}'
  stub_runs actions/checkout "$V4" 'runs:
  using: node20
  main: dist/index.js
'
  stub_runs actions/checkout "$V5" 'runs:
  using: node24
  main: dist/index.js
'
  stub_runs astral-sh/setup-uv "$UV" 'runs:
  using: "node24"
  main: "dist/setup/index.cjs"
'
}

RI=check-remote-items
RIS=common/check-remote-items
RIJ() { printf '{"schema":"check-remote-items/1","problems":%s,"needs_human":%s,"open_prs":%s}' "$1" "$2" "$3"; }

PATH_BEFORE_STUBS=$PATH
PATH="$STUBBIN:$PATH"
STUB_GH_DIR="$STUB"
export PATH STUB_GH_DIR

stub_reset
agree "remote-items nothing open" "$RI" "$RIS" 0
go_said "remote-items nothing open" "$(RIJ 0 0 0)"

stub_reset
printf '[{"number":7,"title":"a report","author":{"login":"someone"},"createdAt":"2026-09-01T00:00:00Z"}]\n' >"$STUB/issues.json"
agree "remote-items an open issue needs a reading and fails nothing" "$RI" "$RIS" 0
go_said "remote-items an open issue needs a reading and fails nothing" "$(RIJ 0 1 0)"

# ⚠ The shape pull request 2 had: a dependency bump with no action pin in it.
stub_reset
stub_pr 2 'rusqlite = { version = "0.40.2" }'
agree "remote-items a pull request with nothing checkable" "$RI" "$RIS" 0
go_said "remote-items a pull request with nothing checkable" "$(RIJ 0 1 1)"

stub_reset
stub_world
stub_pr 3 "      - uses: actions/checkout@$NOSUCH # v4.2.2"
agree "remote-items a pin naming a commit the repository lacks" "$RI" "$RIS" 1
go_said "remote-items a pin naming a commit the repository lacks" "$(RIJ 1 0 1)"

# ⛔ THE CHECK THE RUNTIME DEPRECATION GOT PAST, on the real commit that declares it.
stub_reset
stub_world
stub_pr 4 "      - uses: actions/checkout@$V4 # v4.2.2"
agree "remote-items a pin declaring a deprecated runtime" "$RI" "$RIS" 1
go_said "remote-items a pin declaring a deprecated runtime" "$(RIJ 1 1 1)"

stub_reset
stub_world
stub_pr 5 "      - uses: actions/checkout@$V5 # v4.2.2"
agree "remote-items a label that has drifted from its pin" "$RI" "$RIS" 1
go_said "remote-items a label that has drifted from its pin" "$(RIJ 1 1 1)"

# ⭐ QUOTED, CURRENT AND THE LATEST: the one pin here every question passes.
stub_reset
stub_world
stub_pr 6 "      - uses: astral-sh/setup-uv@$UV # v10.2.0"
agree "remote-items a quoted current runtime at the latest release" "$RI" "$RIS" 0
go_said "remote-items a quoted current runtime at the latest release" "$(RIJ 0 0 1)"

# ⚠ An annotated tag is a tag object, followed to the commit it names.
stub_reset
stub_world
stub_api "repos/astral-sh/setup-uv/git/ref/tags/v9.9.9" "{\"object\":{\"sha\":\"$TAGOBJ\",\"type\":\"tag\"}}"
stub_api "repos/astral-sh/setup-uv/git/tags/$TAGOBJ" "{\"object\":{\"sha\":\"$UV\",\"type\":\"commit\"}}"
stub_pr 7 "      - uses: astral-sh/setup-uv@$UV # v9.9.9"
agree "remote-items an annotated tag followed to its commit" "$RI" "$RIS" 0
go_said "remote-items an annotated tag followed to its commit" "$(RIJ 0 1 1)"

stub_reset
stub_world
stub_pr 8 "      - uses: astral-sh/setup-uv@$UV # v0.0.0-none"
agree "remote-items a label that is not a tag" "$RI" "$RIS" 0
go_said "remote-items a label that is not a tag" "$(RIJ 0 2 1)"

stub_reset
stub_world
stub_pr 9 "      - uses: actions/checkout@$V5"
agree "remote-items a bare pin with no label" "$RI" "$RIS" 0
go_said "remote-items a bare pin with no label" "$(RIJ 0 1 1)"

# ⚠ A PIN THE DIFF REMOVES IS NOT THE PROPOSAL. Only added lines are read; the
# deprecated pin leaving the file must not be charged to the change removing it.
stub_reset
stub_world
stub_pr 11 "      - uses: astral-sh/setup-uv@$UV # v10.2.0"
printf -- '-      - uses: actions/checkout@%s # v4.2.2\n' "$V4" >>"$STUB/diff.11"
agree "remote-items a pin the diff removes is not checked" "$RI" "$RIS" 0
go_said "remote-items a pin the diff removes is not checked" "$(RIJ 0 0 1)"

# ⚠ TWO PINS ON ONE LINE, which YAML's flow style allows. Every match on a line
# is read, so the deprecated second pin is charged even behind a current first.
# ⛔ Added when a mutation reading only the first match per line survived.
stub_reset
stub_world
stub_pr 12 "      steps: [{uses: astral-sh/setup-uv@$UV}, {uses: actions/checkout@$V4}]"
agree "remote-items two pins on one line are both read" "$RI" "$RIS" 1
go_said "remote-items two pins on one line are both read" "$(RIJ 1 2 1)"

stub_reset
stub_pr 10 "      - uses: actions/checkout@$V5 # v5.0.0"
rm -f "$STUB/diff.10"
agree "remote-items a diff that cannot be read" "$RI" "$RIS" 0
go_said "remote-items a diff that cannot be read" "$(RIJ 0 1 1)"

# ⛔ COULD NOT RUN, BOTH WAYS, and never a pass: a CLI with no session, and a
# listing the API refused.
stub_reset
: >"$STUB/auth-fails"
agree "remote-items an unauthenticated gh cannot run" "$RI" "$RIS" 2

stub_reset
printf 'HTTP 502: the listing failed\n' >"$STUB/issues.fail"
agree "remote-items a listing that fails cannot run" "$RI" "$RIS" 2

# ============================================================================
# mine-repo
# ============================================================================
#
# ⭐ A HELPER, NOT A CHECK, AND COMPARED BY WHAT IT LEAVES BEHIND. It writes a
# directory - the API's answers, a clone and PROVENANCE.md - so a case compares
# the exit code, what --json printed, and every file and directory it wrote:
# JSON through `jq -S`, the tree by checksum, and PROVENANCE.md whole but for
# the line naming its writer and the moment it ran.
#
# ⛔ THE STUBS ENFORCE WHAT THE HALVES ONLY SAID. The proxy stub answers 420 to
# any user-agent but curl's own, the git stub refuses a clone without the
# low-speed bound or a depth of one, and it refuses any URL but the target's -
# so an implementation that dropped any of them fails a case, not a reading.
#
# ⚠ THE PROXY ROUTE WAS COMPARED WITHOUT THE POWERSHELL HALF. It fetched with
# `Invoke-WebRequest`, which no stub on PATH intercepts, so it would have asked
# the real proxy about a repository that does not exist; those rows said so, and
# one read-only run against the real proxy stood in for them. The gh route, the
# clone, the trim and the provenance were compared across all three.
#
# ⚠ TWO SPELLINGS WERE THE HOST'S, NOT A DRIFT: the twin named its own flags
# `-Out` and `-NoClone` where the others say `--out` and `--no-clone`, in one
# refusal and one gap, and `mr_exec` maps those two back before comparing.
# ⭐ Both halves were deleted on 2026-09-29, after the comparison ran clean, so
# every case here now runs the port alone and says so; the machinery stays so
# the rows keep saying which question they would have put to a half.
MRS=common/mine-repo
MR_T=stub-owner/stub-repo
MR_OUT="$WORK/mined"
MR_FIX="$WORK/mrfixture"
STUBGIT="$WORK/stubgit"
mkdir -p "$STUBGIT" || exit 2

cat >"$STUBGIT/git" <<'STUBGITEOF'
#!/bin/sh
# A stand-in for git that answers a clone out of a fixture and hands every other
# command to the real git. The commit it makes has a fixed author, committer and
# date, so every implementation cloning one fixture reads one commit.
set -u
REAL=${STUB_GIT_REAL:?}
D=${STUB_GH_DIR:?}
clone=0
bound=0
depth=0
prev=""
src=""
dst=""
for a in "$@"; do
  case "$a" in
    clone) clone=1 ;;
    http.lowSpeedLimit=1024 | http.lowSpeedTime=60) bound=$((bound + 1)) ;;
  esac
  [ "$prev" = "--depth" ] && [ "$a" = 1 ] && depth=1
  prev=$a
  src=$dst
  dst=$a
done
[ "$clone" = 1 ] || exec "$REAL" "$@"
if [ "$bound" != 2 ] || [ "$depth" != 1 ]; then
  printf 'stub git: an unbounded or full clone is refused\n' >&2
  exit 128
fi
if [ -f "$D/clone.fail" ] || [ "$src" != "https://github.com/stub-owner/stub-repo.git" ]; then
  printf 'fatal: repository %s not found\n' "$src" >&2
  exit 128
fi
mkdir -p "$dst" && cp -R "${STUB_GIT_FIXTURE:?}/." "$dst/" || exit 128
if [ -f "$D/fixture.vendor" ]; then
  mkdir -p "$dst/vendor/bundle/gems" && printf 'gem\n' >"$dst/vendor/bundle/gems/g.rb" || exit 128
fi
cd "$dst" || exit 128
GIT_AUTHOR_NAME=stub GIT_AUTHOR_EMAIL=stub GIT_COMMITTER_NAME=stub GIT_COMMITTER_EMAIL=stub
GIT_AUTHOR_DATE=2026-01-01T00:00:00Z GIT_COMMITTER_DATE=2026-01-01T00:00:00Z
export GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL
export GIT_AUTHOR_DATE GIT_COMMITTER_DATE
"$REAL" init -q -b main && "$REAL" add -A &&
  "$REAL" -c commit.gpgsign=false commit -qm fixture || exit 128
"$REAL" rev-parse HEAD >"$D/clone.sha"
STUBGITEOF
chmod +x "$STUBGIT/git" || exit 2

# ⭐ THE FIXTURE CARRIES EVERY SHAPE THE TRIM DECIDES ON: each directory name it
# deletes, one of them nested, a FILE named like one - `find -type d` must keep
# it - and a `vendor/` that stays. Dot-directories are in it on purpose.
mkdir -p "$MR_FIX/src" "$MR_FIX/docs" "$MR_FIX/node_modules/dep" \
  "$MR_FIX/sub/target" "$MR_FIX/build" "$MR_FIX/dist" "$MR_FIX/.next" \
  "$MR_FIX/.venv/bin" "$MR_FIX/lib/__pycache__" "$MR_FIX/vendor" || exit 2
printf 'fixture\n' >"$MR_FIX/README.md"
printf 'int main(void) { return 0; }\n' >"$MR_FIX/src/main.c"
printf 'a file named like a directory the trim deletes\n' >"$MR_FIX/docs/build"
printf 'kept\n' >"$MR_FIX/vendor/keep.rb"
for _d in node_modules/dep sub/target build dist .next .venv/bin lib/__pycache__; do
  printf 'junk\n' >"$MR_FIX/$_d/junk" || exit 2
done

mr_proxy() { # path-and-query json
  printf '%s\n' "$2" >"$STUB/proxy/$(stub_key "$1")"
}

# A proxy page of exactly one hundred records, each carrying one "url", which is
# what the halves count to decide whether another page follows.
mr_full_page() { # path-and-query
  _i=1
  _s=""
  {
    printf '['
    while [ "$_i" -le 100 ]; do
      printf '%s{"url":"https://example.invalid/%s","body":"[unclosed %s"}' "$_s" "$_i" "$_i"
      _s=,
      _i=$((_i + 1))
    done
    printf ']\n'
  } >"$STUB/proxy/$(stub_key "$1")"
}

mr_world_gh() {
  stub_api rate_limit '{"resources":{}}'
  stub_api "repos/pkgforge-dev/reverse-proxies" '{"full_name":"pkgforge-dev/reverse-proxies"}'
  stub_api "repos/$MR_T" '{"full_name":"stub-owner/stub-repo","open_issues_count":2}'
  stub_api "repos/$MR_T/issues?state=all&per_page=100" '[{"url":"https://example.invalid/i/1","body":"see [the report](x"},{"url":"https://example.invalid/i/2","pull_request":{"url":"https://example.invalid/p/2"}}]'
  stub_api "repos/$MR_T/issues/comments?per_page=100" '[{"url":"https://example.invalid/c/1","body":"log: [ERROR] ]["}]'
  stub_api "repos/$MR_T/pulls/comments?per_page=100" '[]'
  stub_api "repos/$MR_T/releases?per_page=100" '[{"url":"https://example.invalid/r/1","tag_name":"v1.0.0"}]'
  stub_api "repos/$MR_T/tags?per_page=100" '[{"name":"v1.0.0"}]'
  stub_api graphql '{"data":{"repository":{"discussions":{"nodes":[]}}}}'
}

# ⭐ THE ISSUES SPAN TWO PAGES AND THE SECOND IS SHORT, with brackets in the
# bodies of both - the shape that broke the halves' first joiner.
mr_world_proxy() {
  mr_proxy "repos/pkgforge-dev/reverse-proxies" '{"full_name":"pkgforge-dev/reverse-proxies"}'
  mr_proxy "repos/$MR_T" '{"full_name":"stub-owner/stub-repo"}'
  mr_full_page "repos/$MR_T/issues?state=all&per_page=100&page=1"
  mr_proxy "repos/$MR_T/issues?state=all&per_page=100&page=2" '[{"url":"https://example.invalid/i/101","body":"]["},{"url":"https://example.invalid/i/102","body":"[x](y"}]'
  mr_proxy "repos/$MR_T/issues/comments?per_page=100&page=1" '[{"url":"https://example.invalid/c/1","body":"see [the report](x"},{"url":"https://example.invalid/c/2","body":"log: [ERROR] [WARN]"}]'
  mr_proxy "repos/$MR_T/pulls/comments?per_page=100&page=1" '[]'
  mr_proxy "repos/$MR_T/releases?per_page=100&page=1" '[{"url":"https://example.invalid/r/1","tag_name":"v1.0.0"}]'
  mr_proxy "repos/$MR_T/tags?per_page=100&page=1" '[{"name":"v1.0.0"}]'
}

# What one implementation left in $MR_OUT, in a form the others can be held to.
mr_files() {
  [ -d "$MR_OUT" ] || {
    printf 'no output directory\n'
    return
  }
  (cd "$MR_OUT" && find . -type d | LC_ALL=C sort) | sed 's/^/dir  /'
  (cd "$MR_OUT" && find . -type f | LC_ALL=C sort) | while IFS= read -r _f; do
    case "$_f" in
      *.json)
        printf 'file %s records %s\n' "$_f" \
          "$(jq 'if type == "array" then length else "-" end' "$MR_OUT/$_f" 2>/dev/null | tr '\n' ' ')"
        jq -S -c . "$MR_OUT/$_f" 2>/dev/null | sed 's/^/     /'
        ;;
      */PROVENANCE.md)
        printf 'file %s\n' "$_f"
        # shellcheck disable=SC2016
        # The backticks are the markdown code span PROVENANCE.md writes around
        # its writer's name, matched as text; nothing here is meant to expand.
        sed -e 's/^Fetched .* by `.*`\.$/Fetched <when> by <writer>./' \
          -e 's/tree: -NoClone was passed/tree: --no-clone was passed/' \
          -e 's/^/     /' "$MR_OUT/$_f"
        ;;
      *) printf 'file %s cksum %s\n' "$_f" "$(cksum <"$MR_OUT/$_f" | cut -d' ' -f1-2)" ;;
    esac
  done
}

# Runs one implementation into a fresh $MR_OUT and writes what it did to
# $WORK/mr.<impl>: the exit code, everything it printed, and mr_files.
mr_exec() { # impl flags...
  _mi=$1
  shift
  rm -rf "$MR_OUT"
  case "$_mi" in
    go)
      (cd "$TREE" && "$BIN" mine-repo --json "$@") >"$OUTF" 2>&1
      ;;
    sh)
      (cd "$TREE" && sh "$TREE/scripts/$MRS.sh" --json "$@") >"$OUTF" 2>&1
      ;;
    ps)
      # ⚠ Each flag is rotated to the end in its PowerShell spelling, which keeps
      # every argument one argument whatever it holds.
      _mn=$#
      while [ "$_mn" -gt 0 ]; do
        case "$1" in
          --out) set -- "$@" -Out ;;
          --route) set -- "$@" -Route ;;
          --no-clone) set -- "$@" -NoClone ;;
          --selftest) set -- "$@" -SelfTest ;;
          *) set -- "$@" "$1" ;;
        esac
        shift
        _mn=$((_mn - 1))
      done
      (cd "$TREE" && pwsh -NoProfile -File "$TREE/scripts/$MRS.ps1" -Json "$@") >"$OUTF" 2>&1
      ;;
  esac
  _mc=$?
  {
    printf 'exit %s\n' "$_mc"
    sed 's/choose another -Out,/choose another --out,/' "$OUTF"
    mr_files
  } >"$WORK/mr.$_mi"
}

# ⭐ ONE CASE, UP TO THREE IMPLEMENTATIONS. `all` compares every half there is,
# `go-sh` leaves the twin out for the reason the row names, and `-` is a rule
# the port has and no half ever did.
mr_agree() { # label expected-code all|go-sh|- flags...
  _ml=$1
  _mw=$2
  _mh=$3
  shift 3
  mr_exec go "$@"
  cp "$WORK/mr.go" "$MR_KEPT"
  _mgc=$(sed -n '1s/^exit //p' "$MR_KEPT")
  if [ "$_mgc" != "$_mw" ]; then
    fail "$_ml: go answered $_mgc, expected $_mw"
    return
  fi
  if [ "$COMPARE" = "0" ]; then
    pass "$_ml: go (exit $_mw)"
    return
  fi
  if [ "$_mh" = "-" ]; then
    pass "$_ml: go (exit $_mw); a Go-only rule, so there is no half to compare"
    return
  fi
  if [ ! -f "$TREE/scripts/$MRS.sh" ]; then
    pass "$_ml: go (exit $_mw); the sh half is gone, not compared"
    return
  fi
  mr_exec sh "$@"
  if ! cmp -s "$MR_KEPT" "$WORK/mr.sh"; then
    fail "$_ml: go and sh disagree
$(diff -u --label go --label sh "$MR_KEPT" "$WORK/mr.sh" | sed 's/^/          /')"
    return
  fi
  if [ "$_mh" = "go-sh" ]; then
    pass "$_ml: go and sh agree (exit $_mw); pwsh not compared, its proxy fetch cannot be stubbed"
    return
  fi
  if [ "$HAVE_PWSH" = "0" ] || [ ! -f "$TREE/scripts/$MRS.ps1" ]; then
    pass "$_ml: go and sh agree (exit $_mw); no PowerShell half compared"
    return
  fi
  mr_exec ps "$@"
  if ! cmp -s "$MR_KEPT" "$WORK/mr.ps"; then
    fail "$_ml: go and pwsh disagree
$(diff -u --label go --label pwsh "$MR_KEPT" "$WORK/mr.ps" | sed 's/^/          /')"
    return
  fi
  pass "$_ml: all three agree (exit $_mw)"
}

# ⛔ AN EXIT CODE OF 0 SAYS A FETCH ENDED, NOT WHAT IT GOT. These assert what the
# port's run of the case just made left behind, from the copy `mr_agree` keeps
# of it before any half runs over the same directory.
MR_KEPT="$WORK/mr.kept"
mr_said() { # label expected-line
  _msaid=$(sed -n '2p' "$MR_KEPT")
  if [ "$_msaid" = "$2" ]; then
    pass "$1: go said $2"
  else
    fail "$1: go said $_msaid, expected $2"
  fi
}

mr_has() { # label fixed-string
  if grep -qF -- "$2" "$MR_KEPT"; then
    pass "$1: go left $2"
  else
    fail "$1: go left no $2"
  fi
}

mr_lacks() { # label fixed-string
  if grep -qF -- "$2" "$MR_KEPT"; then
    fail "$1: go left $2"
  else
    pass "$1: go left no $2"
  fi
}

mr_json() { # route commit gaps
  printf '{"schema":"mine-repo/1","target":"%s","route":"%s","commit":"%s","gaps":%s,"dest":"%s"}' \
    "$MR_T" "$1" "$2" "$3" "$MR_OUT/stub-owner__stub-repo"
}

STUB_GIT_REAL=$(command -v git)
STUB_GIT_FIXTURE=$MR_FIX
PATH_BEFORE_GIT=$PATH
PATH="$STUBGIT:$PATH"
export PATH STUB_GIT_REAL STUB_GIT_FIXTURE

MRD="$MR_OUT/stub-owner__stub-repo"
MRF="./stub-owner__stub-repo"

# -- what it refuses to start with ------------------------------------------

stub_reset
mr_agree "mine-repo --selftest joins and guards" 0 all --selftest
mr_said "mine-repo --selftest joins and guards" '{"schema":"mine-repo-selftest/1","cases":4,"failed":0}'

mr_agree "mine-repo no target cannot run" 2 all --out "$MR_OUT"
mr_agree "mine-repo a target with two slashes is refused" 2 all --out "$MR_OUT" a/b/c
mr_agree "mine-repo an unknown argument is refused" 2 all --bogus "$MR_T"
mr_agree "mine-repo a second target is refused, not taken" 2 all "$MR_T" other/x
mr_agree "mine-repo an unknown route is refused, not probed" 2 all --route prxy "$MR_T"
mr_agree "mine-repo a route in another case is refused" 2 all --route GH "$MR_T"
mr_agree "mine-repo --out with no value cannot run" 2 - "$MR_T" --out

# ⛔ THE CORPUS IS THE EVIDENCE, so an --out this repository ignores is refused
# before anything is fetched. The rule is planted in the scratch tree's own
# exclude file rather than in its tracked .gitignore, and taken out after.
stub_reset
mr_world_gh
MR_OUT_SAVED=$MR_OUT
MR_OUT="$TREE/mined"
EXCL="$TREE/.git/info/exclude"
if [ -f "$EXCL" ]; then cp "$EXCL" "$WORK/exclude.saved" || exit 2; else : >"$WORK/exclude.saved"; fi
printf 'mined/\n' >>"$EXCL"
mr_agree "mine-repo an --out the repository ignores is refused" 2 all --route gh --out "$MR_OUT" "$MR_T"
mr_has "mine-repo an --out the repository ignores is refused" "is ignored by this repository"
mr_lacks "mine-repo an --out the repository ignores is refused" "api/repo.json"
cp "$WORK/exclude.saved" "$EXCL" || exit 2
rm -rf "$MR_OUT"
MR_OUT=$MR_OUT_SAVED

# -- the gh route -------------------------------------------------------------

stub_reset
mr_world_gh
mr_agree "mine-repo the gh route fetches every source and keeps the tree" 0 all --route gh --out "$MR_OUT" "$MR_T"
MR_SHA=$(cat "$STUB/clone.sha" 2>/dev/null)
mr_said "mine-repo the gh route fetches every source and keeps the tree" "$(mr_json gh "$MR_SHA" 0)"
mr_has "mine-repo the gh route keeps the source" "file $MRF/tree/src/main.c"
mr_has "mine-repo the gh route keeps a FILE named like junk" "file $MRF/tree/docs/build"
mr_has "mine-repo the gh route keeps vendor/" "file $MRF/tree/vendor/keep.rb"
mr_lacks "mine-repo the gh route drops the git directory" "$MRF/tree/.git"
mr_lacks "mine-repo the gh route trims node_modules" "node_modules"
mr_lacks "mine-repo the gh route trims a nested target" "$MRF/tree/sub/target"
mr_lacks "mine-repo the gh route trims .next" "$MRF/tree/.next"
mr_lacks "mine-repo the gh route trims .venv" "$MRF/tree/.venv"
mr_lacks "mine-repo the gh route trims __pycache__" "__pycache__"
mr_has "mine-repo the gh route fetches discussions" "file $MRF/api/discussions.json"
mr_has "mine-repo the gh route says it missed nothing" "Nothing. Every source above answered."

stub_reset
mr_world_gh
mr_agree "mine-repo auto takes gh when gh answers" 0 all --out "$MR_OUT" "$MR_T"
mr_said "mine-repo auto takes gh when gh answers" "$(mr_json gh "$MR_SHA" 0)"

stub_reset
mr_world_gh
rm -f "$STUB/api/$(stub_key "repos/$MR_T")"
mr_agree "mine-repo a subject gh cannot fetch fails" 1 all --route gh --out "$MR_OUT" "$MR_T"
mr_has "mine-repo a subject gh cannot fetch fails" "mine-repo: could not fetch repos/$MR_T"
mr_has "mine-repo a subject gh cannot fetch fails" "mine-repo: control says: reachable"

stub_reset
mr_world_gh
rm -f "$STUB/api/$(stub_key "repos/pkgforge-dev/reverse-proxies")"
mr_agree "mine-repo a control gh cannot reach is written down" 0 all --route gh --out "$MR_OUT" "$MR_T"
mr_has "mine-repo a control gh cannot reach is written down" "UNREACHABLE (pkgforge-dev/reverse-proxies did not answer)"

stub_reset
mr_world_gh
rm -f "$STUB/api/$(stub_key "repos/$MR_T/tags?per_page=100")"
mr_agree "mine-repo a list gh cannot fetch is a named gap" 0 all --route gh --out "$MR_OUT" "$MR_T"
mr_said "mine-repo a list gh cannot fetch is a named gap" "$(mr_json gh "$MR_SHA" 1)"
mr_has "mine-repo a list gh cannot fetch is a named gap" "tags: gh could not fetch /repos/$MR_T/tags"

stub_reset
mr_world_gh
rm -f "$STUB/api/graphql"
mr_agree "mine-repo failed discussions are a named gap and no file" 0 all --route gh --out "$MR_OUT" "$MR_T"
mr_has "mine-repo failed discussions are a named gap and no file" "discussions: the GraphQL query failed"
mr_lacks "mine-repo failed discussions are a named gap and no file" "api/discussions.json"

stub_reset
mr_world_gh
mr_agree "mine-repo --no-clone keeps no tree and says so" 0 all --route gh --no-clone --out "$MR_OUT" "$MR_T"
mr_said "mine-repo --no-clone keeps no tree and says so" "$(mr_json gh - 1)"
mr_has "mine-repo --no-clone keeps no tree and says so" "tree: --no-clone was passed"

stub_reset
mr_world_gh
: >"$STUB/clone.fail"
mr_agree "mine-repo a failed clone is a named gap" 0 all --route gh --out "$MR_OUT" "$MR_T"
mr_said "mine-repo a failed clone is a named gap" "$(mr_json gh - 1)"
mr_has "mine-repo a failed clone is a named gap" "tree: the clone failed"

# -- the proxy route ----------------------------------------------------------

stub_reset
mr_world_proxy
mr_agree "mine-repo the proxy route pages and joins by hand" 0 go-sh --route proxy --out "$MR_OUT" "$MR_T"
mr_said "mine-repo the proxy route pages and joins by hand" "$(mr_json proxy "$MR_SHA" 1)"
mr_has "mine-repo the proxy route joins both pages" "file $MRF/api/issues.json records 102"
mr_has "mine-repo the proxy route keeps brackets in bodies" "file $MRF/api/comments.json records 2"
mr_lacks "mine-repo the proxy route leaves no page behind" ".page."
mr_has "mine-repo the proxy route names what it cannot reach" "discussions: NOT FETCHED"
mr_has "mine-repo the proxy route reads its control" "reachable (pkgforge-dev/reverse-proxies answered 200)"

stub_reset
mr_world_proxy
: >"$STUB/auth-fails"
mr_agree "mine-repo auto takes the proxy when gh is not authenticated" 0 go-sh --out "$MR_OUT" "$MR_T"
mr_said "mine-repo auto takes the proxy when gh is not authenticated" "$(mr_json proxy "$MR_SHA" 1)"

stub_reset
mr_world_proxy
printf '502\n' >"$STUB/proxy/$(stub_key "repos/$MR_T/issues/comments?per_page=100&page=1").code"
mr_agree "mine-repo a failed page is a named gap and leaves nothing" 0 go-sh --route proxy --out "$MR_OUT" "$MR_T"
mr_has "mine-repo a failed page is a named gap and leaves nothing" "comments: proxy returned 502 on page 1"
mr_lacks "mine-repo a failed page is a named gap and leaves nothing" "api/comments.json"

stub_reset
mr_world_proxy
rm -f "$STUB/proxy/$(stub_key "repos/$MR_T")"
mr_agree "mine-repo a subject the proxy cannot find fails beside its control" 1 go-sh --route proxy --out "$MR_OUT" "$MR_T"
mr_has "mine-repo a subject the proxy cannot find fails beside its control" "mine-repo: proxy returned 404 for repos/$MR_T"

stub_reset
mr_world_proxy
printf '500\n' >"$STUB/proxy/$(stub_key "repos/pkgforge-dev/reverse-proxies").code"
mr_agree "mine-repo a control the proxy cannot reach is written down" 0 go-sh --route proxy --out "$MR_OUT" "$MR_T"
mr_has "mine-repo a control the proxy cannot reach is written down" "UNREACHABLE (pkgforge-dev/reverse-proxies answered 500)"

# -- what only the port does --------------------------------------------------

# ⛔ A LIST CUT OFF AT THE PAGE CAP IS NAMED. Both halves stopped after ten full
# pages and said "ok"; ten full pages and an eleventh behind them is the shape.
stub_reset
mr_world_proxy
_mp=1
while [ "$_mp" -le 11 ]; do
  mr_full_page "repos/$MR_T/tags?per_page=100&page=$_mp"
  _mp=$((_mp + 1))
done
mr_agree "mine-repo a list cut off at the page cap is a named gap" 0 - --route proxy --out "$MR_OUT" "$MR_T"
mr_has "mine-repo a list cut off at the page cap is a named gap" "tags: stopped at 10 pages of 100"
mr_has "mine-repo a list cut off at the page cap keeps what it got" "file $MRF/api/tags.json records 1000"

# ⚠ `vendor/bundle` IS A PATH. The sh half listed it where `find -name` takes a
# name, so it never matched, and its twin left it out.
stub_reset
mr_world_gh
: >"$STUB/fixture.vendor"
mr_agree "mine-repo vendor/bundle is trimmed and vendor/ is not" 0 - --route gh --out "$MR_OUT" "$MR_T"
mr_lacks "mine-repo vendor/bundle is trimmed and vendor/ is not" "vendor/bundle"
mr_has "mine-repo vendor/bundle is trimmed and vendor/ is not" "file $MRF/tree/vendor/keep.rb"

rm -rf "$MR_OUT" "$MRD"
PATH=$PATH_BEFORE_GIT
export PATH

PATH=$PATH_BEFORE_STUBS
export PATH

# ============================================================================
# check-placeholders
# ============================================================================

# ⛔ EVERY NEEDLE BELOW IS ASSEMBLED RATHER THAN WRITTEN, and that is not style.
# A harness that spelled a stand-in value or a double brace as a literal would BE
# the finding: check-placeholders reads every tracked file, this file is one, and
# its first version turned the clean tree red for exactly that reason. ⚠ The three
# implementations of the check are exempt from themselves; a harness is not, and
# exempting one more file is a worse answer than not planting the literal.
# ⭐ It is the same rule the marker harness already followed, by building its
# marker bytes with printf instead of typing them.
BR=$(printf '%s%s' '{' '{')
BRC=$(printf '%s%s' '}' '}')

agree "placeholders clean tree" check-placeholders common/check-placeholders 0

# ⛔ THE FOUR CATEGORIES ARE FOUR RULES and each is planted on its own, because a
# harness that planted them together could not tell which one fired.
printf 'a survived %sPLACEHOLDER%s here\n' "$BR" "$BRC" >"$TREE/tools/check/plant.md"
agree "placeholders a double-brace placeholder is refused" check-placeholders common/check-placeholders 1
unplant tools/check/plant.md

printf 'text\n<!-- %s: fill this in -->\n' 'TEMPLATE' >"$TREE/tools/check/plant.md"
agree "placeholders a template guidance comment is refused" check-placeholders common/check-placeholders 1
unplant tools/check/plant.md

printf 'contact %s%s for details\n' 'CHANGE' 'ME' >"$TREE/tools/check/plant.md"
agree "placeholders a stand-in value is refused" check-placeholders common/check-placeholders 1
unplant tools/check/plant.md

# ⚠ THE OWNER-SLASH-REPO GENERIC IS RECOMMENDED IN A PUBLIC DOCUMENT, so it is a
# defect only where it is configuration rather than prose. Both are planted,
# because they are different branches and a rule that fired on both would fire on
# correct writing.
printf 'clone %s/%s to begin\n' 'OWNER' 'REPO' >"$TREE/tools/check/plant.md"
agree "placeholders the generic in a .md is accepted" check-placeholders common/check-placeholders 0
unplant tools/check/plant.md

printf 'repo: %s/%s\n' 'OWNER' 'REPO' >"$TREE/tools/check/plant.yml"
agree "placeholders the generic in a config file is refused" check-placeholders common/check-placeholders 1
unplant tools/check/plant.yml

# ⭐ THE TWO SHAPES THAT MUST NOT FIRE, and they are why the brace rule is narrow
# rather than absent. GitHub Actions expression syntax and a Go template both
# carry a double brace, and a rule that refused either would refuse every correct
# workflow file and every container format string in the tree.
printf 'run: echo $%s github.sha %s\n' "$BR" "$BRC" >"$TREE/tools/check/plant.yml"
agree "placeholders an Actions expression is accepted" check-placeholders common/check-placeholders 0
unplant tools/check/plant.yml

printf 'info --format %sjson .Host.Arch%s\n' "$BR" "$BRC" >"$TREE/tools/check/plant.yml"
agree "placeholders a Go template is accepted" check-placeholders common/check-placeholders 0
unplant tools/check/plant.yml

agree "placeholders clean tree again" check-placeholders common/check-placeholders 0

# ============================================================================
# check-no-secrets
# ============================================================================
#
# ⛔ EVERY NEEDLE BELOW IS ASSEMBLED AND NONE OF THEM IS WRITTEN, for the reason
# the placeholder needles are: check-no-secrets reads every file in this tree,
# this harness is one of them, and a token shape or a forty-digit run typed here
# as a literal would make THIS file the finding. ⚠ A credential's SHAPE is
# exactly what it looks for, so the rule bites harder here than anywhere else -
# a bare `AKIA` and sixteen characters would turn the clean-tree control red and
# the harness would be refusing itself.
AKID=$(printf '%s%s%s' 'AKI' 'A' 'IOSFODNN7EXAMPLE')
KEYBLOCK=$(printf '%s %s' 'BEGIN' 'PRIVATE KEY')
URLCRED=$(printf '%s//%s:%s@%s' 'https:' 'bob' 'hunter22x' 'example.invalid')
MAIL=$(printf '%s@%s' 'someone' 'example.invalid')
HOMEP=$(printf '%s/%s/work' '/home' 'alice')
HOMEG=$(printf '%s/%s/work' '/home' 'runner')

# ⚠ SIXTEEN HEX DIGITS IS UNDER THE TWENTY-FOUR FLOOR, which is what lets the
# pieces be written and the assembled runs not be.
HX=0123456789abcdef
HEX40=$(printf '%s%s%s' "$HX" "$HX" '01234567')
HEX64=$(printf '%s%s%s%s' "$HX" "$HX" "$HX" "$HX")
HEX46=$(printf '%s%s' "$HEX40" '012345')

agree "no-secrets clean tree" check-no-secrets common/check-no-secrets 0

# ⛔ A CREDENTIAL FILE, TRACKED. `id_ecdsa` and `*.jks` are the two names the
# check knows that .gitignore does not carry, so an untracked plant of either is
# genuinely in scope - but it is force-added anyway, because the rule is about a
# file that reached the index and a fixture that only works while an ignore list
# has a hole is a fixture that breaks when the hole is closed.
printf 'not a real key\n' >"$TREE/tools/check/id_ecdsa"
plant_tracked tools/check/id_ecdsa
agree "no-secrets a tracked credential file is refused" check-no-secrets common/check-no-secrets 1
unplant_tracked tools/check/id_ecdsa

# ⭐ AND THE WAIVER, WHICH IS A DIFFERENT BRANCH. `.env.example` matches the same
# name rule and is then dropped by the `.example` exclusion; without that branch
# the rule fires on the file a project is SUPPOSED to commit. ⚠ .gitignore
# un-ignores this one name explicitly, so it reaches the scan on its own.
printf 'TOKEN=\n' >"$TREE/tools/check/.env.example"
agree "no-secrets an .example credential template is accepted" check-no-secrets common/check-no-secrets 0
unplant tools/check/.env.example

printf 'key %s here\n' "$AKID" >"$TREE/tools/check/plant.md"
agree "no-secrets an aws access key id is refused" check-no-secrets common/check-no-secrets 1
unplant tools/check/plant.md

printf -- '-----%s-----\n' "$KEYBLOCK" >"$TREE/tools/check/plant.md"
agree "no-secrets a private key block is refused" check-no-secrets common/check-no-secrets 1
unplant tools/check/plant.md

printf 'clone %s now\n' "$URLCRED" >"$TREE/tools/check/plant.md"
agree "no-secrets a password in a url is refused" check-no-secrets common/check-no-secrets 1
unplant tools/check/plant.md

# ⛔ AN EMAIL IS NOT A DEFAULT-MODE FINDING, and this is the case that separates
# the two modes rather than treating --public as "the same rules, stricter". In a
# private project an address is legitimate content.
printf 'write to %s\n' "$MAIL" >"$TREE/tools/check/plant.md"
agree "no-secrets an email address is accepted without --public" check-no-secrets common/check-no-secrets 0
unplant tools/check/plant.md

# ⚠ A BINARY FILE IS SKIPPED, which is what `grep -I` does, and the plant carries
# a real key so the case cannot pass by the file being uninteresting. ⛔ The byte
# is written by printf rather than as a literal, this repository's own rule.
printf 'key %s\000here\n' "$AKID" >"$TREE/tools/check/plant.md"
agree "no-secrets a key inside a binary file is out of scope" check-no-secrets common/check-no-secrets 0
unplant tools/check/plant.md

agree "no-secrets clean tree again" check-no-secrets common/check-no-secrets 0

# -- the public rules ---------------------------------------------------------
#
# ⚠ --public IS A SECOND QUESTION AND THE FLAG IS PASSED TO ALL THREE. The Go
# binary and the sh half spell it the same way; `agree` translates it for the
# PowerShell half, because only that one differs.

agree "no-secrets clean tree, public" check-no-secrets common/check-no-secrets 0 --public

printf 'write to %s\n' "$MAIL" >"$TREE/tools/check/plant.md"
agree "no-secrets an email address is refused with --public" check-no-secrets common/check-no-secrets 1 --public
unplant tools/check/plant.md

printf 'built in %s\n' "$HOMEP" >"$TREE/tools/check/plant.md"
agree "no-secrets an absolute home path is refused" check-no-secrets common/check-no-secrets 1 --public
unplant tools/check/plant.md

# ⚠ Narrowed rather than switched off: /home/runner/ is a well-known generic
# path, not a fingerprint of anybody's machine.
printf 'built in %s\n' "$HOMEG" >"$TREE/tools/check/plant.md"
agree "no-secrets a generic runner home path is accepted" check-no-secrets common/check-no-secrets 0 --public
unplant tools/check/plant.md

printf 'a bare %s here\n' "$HEX40" >"$TREE/tools/check/plant.md"
agree "no-secrets a bare long hex run is refused" check-no-secrets common/check-no-secrets 1 --public
unplant tools/check/plant.md

# ⚠ A PINNED ACTION IS THE SAFE PRACTICE and a rule that fired on correct
# hardening is a rule somebody disables.
printf 'uses: owner/repo@%s\n' "$HEX40" >"$TREE/tools/check/plant.yml"
agree "no-secrets a pinned action commit is accepted" check-no-secrets common/check-no-secrets 0 --public
unplant tools/check/plant.yml

# ⛔ THE CASE THAT SAYS THE ALLOW EXPRESSIONS READ THE OUTPUT LINE AND NOT THE
# TEXT. The lockfile allowance is anchored `^(.*Cargo\.lock:[0-9]+:checksum = )`,
# so it can only match with the `path:lineno:` prefix in front of it. A port that
# ran its allowances over the file's own line would refuse every registry digest
# in the tree, and every case above would still be green.
printf '%s "%s"\n' 'checksum =' "$HEX64" >"$TREE/tools/check/plant-Cargo.lock"
agree "no-secrets a registry lockfile digest is accepted" check-no-secrets common/check-no-secrets 0 --public
unplant tools/check/plant-Cargo.lock

# ⛔ AND THE CASE THAT SAYS AN ALLOWED ITEM IS DELETED FROM THE LINE RATHER THAN
# THE LINE BEING DROPPED. `grep -v` drops lines, not characters, so an allowed
# digest sitting beside a real one would take the real one out of the report with
# it. This plants exactly that pair on one line: an allowed `sha256:` digest and
# a bare run. A port that dropped the line answers 0 and every other case here
# still passes.
printf 'sha256:%s and %s\n' "$HEX64" "$HEX40" >"$TREE/tools/check/plant.md"
agree "no-secrets an allowed digest beside a bare one is still refused" check-no-secrets common/check-no-secrets 1 --public
unplant tools/check/plant.md

printf 'Infohash: %s\n' "$HEX40" >"$TREE/tools/check/plant.md"
agree "no-secrets an announce infohash is accepted" check-no-secrets common/check-no-secrets 0 --public
unplant tools/check/plant.md

# ⛔ THE TRAILING CLASS IS THE ANCHOR AND THIS IS WHAT IT IS FOR. Without it
# `{40}` blanks the first forty characters of a LONGER run and leaves a remainder
# too short to reach the threshold, so a forty-six digit value after the field
# name goes unreported. Both shell twins agreed on that and both were wrong.
printf 'Infohash: %s\n' "$HEX46" >"$TREE/tools/check/plant.md"
agree "no-secrets a longer run after the infohash field is refused" check-no-secrets common/check-no-secrets 1 --public
unplant tools/check/plant.md

agree "no-secrets clean tree again, public" check-no-secrets common/check-no-secrets 0 --public

# ============================================================================
# check-ignores
# ============================================================================
#
# ⭐ THE FIRST RULE HERE THAT WAS NEVER A SHELL CHECK, so its cases name `-` for
# the predecessor rather than a path: there is nothing to compare, which is a
# different fact from a half that has been deleted, and the row says which.
#
# ⚠ WHAT THESE CASES CANNOT REACH is the specimen self-check - the branch that
# fires when a shape is narrowed OUT of check-no-secrets while its specimen stays
# behind. That plant is a change to Go source, and this harness builds the binary
# once before the first case, so a case cannot rebuild it. ⛔ It is mutation-proved
# in TODO/ci.md against a plant verified to have changed the file, and recorded as
# a measurement rather than claimed as a case here.

agree "ignores clean tree" check-ignores - 0

# ⛔ A HOLE IN THE IGNORE LIST. This is the defect the rule exists for, and it is
# exactly the shape found by hand on 2026-09-15: a name check-no-secrets refuses
# that `git add -A` would stage anyway.
GI="$TREE/.gitignore"
cp "$GI" "$WORK/gitignore.orig" || exit 2
grep -v '^\*\.pem$' "$WORK/gitignore.orig" >"$GI"
agree "ignores a credential shape missing from .gitignore is refused" check-ignores - 1
cp "$WORK/gitignore.orig" "$GI"

# ⚠ AND MORE THAN ONE HOLE IS STILL ONE REFUSAL, which is worth a case because
# the report counts problems and the verdict is a single code. A port or a rewrite
# that stopped after the first finding would pass the case above and lose the rest
# of the report.
grep -vE '^(\*\.pem|id_rsa|credentials\.json)$' "$WORK/gitignore.orig" >"$GI"
agree "ignores several holes at once are still one refusal" check-ignores - 1
cp "$WORK/gitignore.orig" "$GI"

agree "ignores clean tree again" check-ignores - 0

# ============================================================================
# check-adapters
# ============================================================================
#
# ⭐ THE SECOND RULE HERE THAT WAS NEVER A SHELL CHECK, so its cases name `-` for
# the predecessor. ⛔ It exists because `capture-client` runs 21 and 22 both hung
# in *Install the client* on the RELEASE lane - six seconds on runs 19 and 20 -
# and an unbounded fetch is the only thing in that route that can wait forever.
# `docs/conventions/shell.md` section 9 had stated the rule for as long as four
# adapters had been breaking it.

agree "adapters clean tree" check-adapters - 0

# ⛔ A RELEASE FETCH WITH NO TIME LIMIT, which is the exact shape found on
# 2026-09-15 in four adapters at once.
#
# ⚠ IT IS PLANTED IN THE INSTALLER NOW, BECAUSE THAT IS WHERE THE FETCH WENT.
# `ACQ-06` moved every release retrieval out of the four adapters and into one
# rootless installer, so this plant found nothing to change there - and the rule
# answered 2 over a scope that no longer held a single `curl -o`. ⭐ The rule's
# own floor is what said so; the scope followed the subject and so does this.
RL="$TREE/scripts/acquisition/install-rootless.sh"
cp "$RL" "$WORK/rootless.orig" || exit 2
# ⚠ THE FLAG ALONE IS DROPPED AND ITS VALUE IS LEFT, which keeps this sed free of
# a `$` the shell would have to be told not to expand. ⛔ And it must drop the
# flag rather than rename it: the rule asks whether the line CONTAINS
# `--max-time`, so a `--no-max-time` would still satisfy it and the plant would
# report a guard that had not been tested.
sed 's/--connect-timeout 20 --max-time/--connect-timeout 20/' \
  "$WORK/rootless.orig" >"$RL"
agree "adapters a curl that writes a file with no --max-time is refused" check-adapters - 1
cp "$WORK/rootless.orig" "$RL"

# ⛔ AND A CLONE THAT NOTHING BOUNDS. `git` has no flag of its own, so the rule
# is a `timeout` wrapper, and a clone is the other way a capture host hangs.
AD="$TREE/scripts/capture/adapters/aria2-next.sh"
cp "$AD" "$WORK/adapter.orig" || exit 2
sed 's/timeout 600 git clone/git clone/' "$WORK/adapter.orig" >"$AD"
agree "adapters a git clone with no timeout wrapper is refused" check-adapters - 1
cp "$WORK/adapter.orig" "$AD"

# ⚠ AND A BOUND THAT BOUNDS SOMETHING ELSE IS NOT A BOUND. A `timeout` after the
# command it is supposed to wrap satisfies a rule that only asks whether the word
# appears on the line.
sed 's/timeout 600 git clone \(.*\)$/git clone \1 \&\& timeout 5 true/' \
  "$WORK/adapter.orig" >"$AD"
agree "adapters a timeout after the command it should wrap is refused" check-adapters - 1
cp "$WORK/adapter.orig" "$AD"

# ⭐ AND THE ACCEPTING HALF THAT MATTERS MOST: a curl that only ASKS is out of
# scope, so a rule that demanded a bound on every invocation would fire on the
# probe this adapter already carries and would be a rule somebody switches off.
agree "adapters clean tree again" check-adapters - 0

# ============================================================================
# check-docs
# ============================================================================
#
# ⛔ EVERY PLANT THAT MUST BE ACCEPTED SITS AT THE REPOSITORY ROOT, and that is
# forced rather than chosen. A new `.md` in a subdirectory is an ORPHAN unless
# something links to it, so a plant placed there would be refused by the orphan
# rule whatever the branch under test did - and a case expecting 0 would fail for
# a reason it was not written to ask about. A root file is an entry point and the
# orphan rule skips it, which leaves exactly one rule looking at the plant.
P=plant.md

agree "docs clean tree" check-docs common/check-docs 0

printf 'see [the thing](nope-does-not-exist.md) here\n' >"$TREE/$P"
agree "docs a broken relative link is refused" check-docs common/check-docs 1
unplant "$P"

# ⭐ THE TWO READERS ARE NOT THE SAME READER, and these two cases are the whole
# of why. The broken-link pass strips inline code spans, because markdown does
# not linkify one; the orphan pass did NOT, in the `sh` half alone. A port that
# unified them passes the clean tree and fails here.
# shellcheck disable=SC2016
# The backticks are the SUBJECT: this case exists to put a link inside an inline
# code span, so nothing here is meant to expand. The directive is in this file
# rather than in the invocation, because shellcheck answers differently depending
# on how files were grouped on its command line.
printf 'see `[the thing](nope-does-not-exist.md)` here\n' >"$TREE/$P"
agree "docs a broken link inside a code span is accepted" check-docs common/check-docs 0
unplant "$P"

{
  printf 'text\n\n'
  printf '```\n'
  printf '[the thing](nope-does-not-exist.md)\n'
  printf '```\n'
} >"$TREE/$P"
agree "docs a broken link inside a fenced block is accepted" check-docs common/check-docs 0
unplant "$P"

# ⚠ AN ABSOLUTE URL IS NOT A PATH and nothing here resolves one. A rule that
# tried would need the network, and a check that needs the network is a check
# that goes red when somebody else's server does.
printf 'see [the thing](https://example.invalid/nope) here\n' >"$TREE/$P"
agree "docs an unresolvable http link is out of scope" check-docs common/check-docs 0
unplant "$P"

{
  printf 'text\n\n'
  printf '```sh\n'
  printf 'if [ 1 = 1 ; then echo broken\n'
  printf '```\n'
} >"$TREE/$P"
agree "docs a shell block that does not parse is refused" check-docs common/check-docs 1
unplant "$P"

# ⛔ THE PLACEHOLDER A HUMAN READS AS *fill this in* AND bash READS AS A
# REDIRECT. The block parses; that is the point, and it is why this is a separate
# rule rather than a consequence of the one above.
#
# ⛔ AND THE NEEDLE IS ASSEMBLED, WHICH THIS FILE HAD TO LEARN TWICE. The first
# spelling used the possessive stand-in word check-placeholders also looks for,
# so the harness became THAT rule's finding and five of its cases went red over a
# clean tree. ⚠ Then the comment explaining the mistake spelled the same literal
# and did it again. ⭐ The angle brackets are check-docs' subject and the word
# inside them is not, so the word is one no other rule looks for and it is built
# rather than typed - and this sentence names the class instead of quoting it.
ANGLE=$(printf '%s%s%s' '<' 'endpoint' '>')
{
  printf 'text\n\n'
  printf '```sh\n'
  printf 'curl -sS %s\n' "$ANGLE"
  printf '```\n'
} >"$TREE/$P"
agree "docs an angle-bracket placeholder in a shell block is refused" check-docs common/check-docs 1
unplant "$P"

# ⭐ AND A BLOCK THAT IS SIMPLY CORRECT, because a rule that refused every fenced
# block would pass both cases above.
{
  printf 'text\n\n'
  printf '```sh\n'
  printf 'set -eu\nprintf "%%s\\n" "ok"\n'
  printf '```\n'
} >"$TREE/$P"
agree "docs a shell block that parses is accepted" check-docs common/check-docs 0
unplant "$P"

# -- the orphan rule ----------------------------------------------------------

mkdir -p "$TREE/docs"
printf 'a page nothing points at\n' >"$TREE/docs/plant-orphan.md"
agree "docs a page nothing links to is refused" check-docs common/check-docs 1

# ⛔ AND THE SAME PAGE, CITED ONLY INSIDE BACKTICKS, IS STILL AN ORPHAN. ⭐ THIS
# CASE IS WHY --compare EXISTS: on its first run the two shell halves ANSWERED
# DIFFERENTLY here, and nothing in this repository could have told anybody.
#
# The `sh` half read links with two awk programs and only the broken-link one
# stripped code spans, so a backticked citation counted as a link; the PowerShell
# twin has a single extractor that strips them and feeds both passes. ⚠ No page
# in this tree is cited only that way, so `check-twins` saw the two agree on
# every run for as long as both existed - its own documented blind spot, a rule
# differing only on a shape the tree does not contain.
#
# ⭐ The twin is correct on the rule's own reasoning: a code span is not a
# hyperlink and a reader following links never arrives. The `sh` half and the
# port were both changed to match, deliberately and in the same change, so this
# case asserts a REFUSAL rather than freezing the defect it found.
# shellcheck disable=SC2016
# The backticks are the subject here too, for the reason given above.
printf 'cited as `[the page](docs/plant-orphan.md)` above\n' >"$TREE/$P"
agree "docs a page cited only inside backticks is still an orphan" check-docs common/check-docs 1
unplant "$P"
unplant docs/plant-orphan.md

# ⚠ A LINK THAT CLIMBS MORE THAN TWO LEVELS, and this case is a GUARD rather than
# a proof - which is worth saying, because it was written believing it was a
# proof. Normalising `.../fixtures/../../../../docs/plant-orphan.md` is where
# both shell halves hand-rolled a `segment/../` collapse, and
# docs/conventions/forbidden-patterns.md records a GLOBAL replace eating a real
# segment and a `../..` pair together, answering `crates/bit-ids/docs/...` and
# leaving the real page an orphan. The Go port wrote that same global spelling
# first.
#
# ⛔ BUT THIS CASE CANNOT REFUTE IT, MEASURED BY PLANTING: with the global
# spelling restored and the plant verified to have changed the file, the case
# still PASSED. `path.Join` normalises before any collapse runs, so the branch
# that differs is unreachable - the port is correct for a reason this case does
# not test. ⭐ The hand-rolled collapse is deleted rather than fixed, and the case
# is kept as a regression guard on the ANSWER with its label saying so, the way
# the empty-register case above names the rule that really refuses it.
#
# ⚠ The README carrying the link is orphan-exempt, so the only thing this case
# can report is the page it points at.
#
# ⛔ AND THIS CASE OVERWRITES A REAL TRACKED FILE, so it is SAVED AND PUT BACK
# rather than unplanted. `unplant` is `rm`, and deleting a tracked file left the
# tree carrying a path git still lists with no bytes behind it - invisible to
# every rule here until `check-project` began reading the WORKING TREE's line
# endings, which then reported `w/` for a file nobody had touched.
FIXREADME=crates/bit-ids/tests/fixtures/README.md
cp "$TREE/$FIXREADME" "$WORK/fixtures-README.orig" || exit 2
printf 'a page nothing else points at\n' >"$TREE/docs/plant-orphan.md"
printf 'see [the page](../../../../docs/plant-orphan.md)\n' \
  >"$TREE/$FIXREADME"
agree "docs a link climbing four levels resolves to the same page" check-docs common/check-docs 0
cp "$WORK/fixtures-README.orig" "$TREE/$FIXREADME"
unplant docs/plant-orphan.md

# ⚠ A README IS AN ENTRY POINT and the rule skips it by name, so a subdirectory
# README nothing links to is accepted. This tree has no such file, so the branch
# is proved with a fixture rather than assumed.
mkdir -p "$TREE/docs/plantdir"
printf 'an entry point\n' >"$TREE/docs/plantdir/README.md"
agree "docs an unlinked subdirectory README is accepted" check-docs common/check-docs 0
rm -rf "$TREE/docs/plantdir"

# -- the history index --------------------------------------------------------
#
# ⛔ A RECORD LINKED FROM ANYWHERE IS NOT AN ORPHAN AND IS STILL MISSING FROM THE
# INDEX, which is the stronger rule. ⚠ The plant is LINKED on purpose, so the
# orphan rule is satisfied and this case can only fail for the index reason.
printf 'an old record\n' >"$TREE/docs/history/SESSION-PLANT.md"
printf 'see [the record](docs/history/SESSION-PLANT.md)\n' >"$TREE/$P"
agree "docs a session record the history index omits is refused" check-docs common/check-docs 1
unplant docs/history/SESSION-PLANT.md
unplant "$P"

# -- the template exemption ---------------------------------------------------
#
# ⛔ docs/templates/ IS EXEMPT FROM LINK RESOLUTION AND THIS TREE HAS NO SUCH
# DIRECTORY, so the branch is invisible to any comparison over it - the `py`
# scope lesson from check-markers, arriving in a different rule. ⚠ A template's
# links are written relative to where the file will live in a PROJECT, not where
# it lives here.
#
# ⚠ The plant is linked from the root file, because the orphan rule has NO
# template exemption and would otherwise refuse it for a different reason.
mkdir -p "$TREE/docs/templates"
printf 'see [gate](methodology/gate.md), which is not here\n' >"$TREE/docs/templates/plant.md"
printf 'see [the template](docs/templates/plant.md)\n' >"$TREE/$P"
agree "docs a template's unresolvable link is exempt" check-docs common/check-docs 0
rm -rf "$TREE/docs/templates"
unplant "$P"

agree "docs clean tree again" check-docs common/check-docs 0

# ============================================================================
# check-project
#
# ⛔ THIRTY REFUSALS IN ONE CHECK, so one case per rule is the floor rather
# than thoroughness. The verdict this harness compares is the whole `--json`
# line, `failures` included, so a plant that trips two rules is still a proof:
# the case states the exit code and the three implementations must agree on the
# COUNT as well. That is why several plants below are not narrowed - an invalid
# status necessarily moves four derived totals, and a port that moved three of
# them would be caught here rather than reasoned about.
# ============================================================================

agree "project clean tree" check-project common/check-project 0

PIDX="$TREE/TODO/INDEX.md"
PPRG="$TREE/TODO/PROGRESS.md"
PSUM="$TREE/TODO/SUMMARY.md"
PMTX="$TREE/docs/client-matrix.md"
PCAT="$TREE/catalogue/clients.toml"
PLCK="$TREE/Cargo.lock"
PPRV="$TREE/scripts/doctor/provision.sh"
for _f in "$PIDX" "$PPRG" "$PSUM" "$PMTX" "$PCAT" "$PLCK" "$PPRV"; do
  cp "$_f" "$WORK/$(basename "$_f").projorig" || exit 2
done
# ⚠ Kept apart from the list above because two cases edit it and then put it
# back on their own, rather than through the blanket restore.
cp "$TREE/TODO/ci.md" "$WORK/ci.md.projsrc" || exit 2
restore_project() {
  for _r in "$PIDX" "$PPRG" "$PSUM" "$PMTX" "$PCAT" "$PLCK" "$PPRV"; do
    cp "$WORK/$(basename "$_r").projorig" "$_r"
  done
}

# -- 1. the files a reader is promised ----------------------------------------
mv "$TREE/docs/reference-sweeps/bit-cli.md" "$WORK/bit-cli.projmoved"
agree "project a promised file that is missing is refused" check-project common/check-project 1
mv "$WORK/bit-cli.projmoved" "$TREE/docs/reference-sweeps/bit-cli.md"

# -- 2. the target set, in both directions ------------------------------------
# shellcheck disable=SC2016
# ⚠ The backticks are the SUBJECT: a matrix row names its target in a code
# span, and expanding them here would hand the file a command's output.
printf '| `plant-client` | x | x | x |\n' >>"$PMTX"
agree "project a matrix row the catalogue does not carry is refused" check-project common/check-project 1
restore_project

printf '\n[[client]]\nid = "plant-client"\n' >>"$PCAT"
agree "project a catalogue target the matrix lacks is refused" check-project common/check-project 1
restore_project

# ⛔ TWO EMPTY SETS AGREE PERFECTLY. Without this floor a parser that stopped
# matching would report a pinned matrix over nothing at all.
printf '# nothing here\n' >"$PMTX"
agree "project a target set too small to be real is refused" check-project common/check-project 1
restore_project

# -- 3-8. the TODO bookkeeping ------------------------------------------------
sed 's/^Total: /Total: 9/' "$WORK/PROGRESS.md.projorig" >"$PPRG"
agree "project a declared total that disagrees is refused" check-project common/check-project 1
restore_project

sed 's/| OPEN |/| WIP |/' "$WORK/INDEX.md.projorig" >"$PIDX"
agree "project an invalid status in the index is refused" check-project common/check-project 1
restore_project

# ⚠ THE OPEN COUNT IN THE TOTAL ROW, by field rather than by a number typed
# here, so the plant follows the file. A first version prepended a digit to the
# row's PREFIX cell instead, which is a different rule's business - and the three
# implementations disagreed about it, which is how the twin's whole-line
# comparison was found.
awk -F'|' 'BEGIN { OFS = "|" } $2 == " Total " { $4 = " 99 " } { print }' \
  "$WORK/SUMMARY.md.projorig" >"$PSUM"
agree "project a summary total that disagrees is refused" check-project common/check-project 1
restore_project

# ⛔ THE CATEGORY ROWS ARE ELEVEN OF TWELVE, and the total row agreeing says
# nothing about them: setting one category to a wrong number used to pass.
awk -F'|' 'BEGIN { OFS = "|" } $2 == " Foundation " { $4 = " 99 " } { print }' \
  "$WORK/SUMMARY.md.projorig" >"$PSUM"
agree "project a summary category row that disagrees is refused" check-project common/check-project 1
restore_project

# ⚠ THE EFFORT SENTENCE IS PROSE AND IT DRIFTED AS PROSE: it said 49 L and 0 XL
# while the index carried 51 and 1. The plant rewrites the XL count whatever it
# is, so it follows the file rather than a count typed here.
awk '/^Effort inventory: / { sub(/, [0-9]+ XL\./, ", 7 XL.") } { print }' \
  "$WORK/SUMMARY.md.projorig" >"$PSUM"
agree "project a summary effort inventory that disagrees is refused" check-project common/check-project 1
restore_project

awk 'BEGIN { done = 0 }
     /^\| P1 \|/ && !done { sub(/\| [0-9]+ \|/, "| 98 |"); done = 1 }
     { print }' "$WORK/INDEX.md.projorig" >"$PIDX"
agree "project a priority table row that disagrees is refused" check-project common/check-project 1
restore_project

# ⚠ A DUPLICATED ROW IS TWO FINDINGS, not one: the duplicate itself and every
# total it moves. All three implementations have to count the same way.
awk 'BEGIN { done = 0 }
     { print }
     /^\| CI-0/ && !done { print; done = 1 }' "$WORK/INDEX.md.projorig" >"$PIDX"
agree "project a duplicated index row is refused" check-project common/check-project 1
restore_project

# -- 9. Python exists only with an approved exception -------------------------
printf 'print("plant")\n' >"$TREE/tools/check/plant.py"
agree "project a .py declaring no exception is refused" check-project common/check-project 1

printf '# bit-ids:python-exception=ZZZ-99\nprint("plant")\n' >"$TREE/tools/check/plant.py"
agree "project a .py naming an entry the index lacks is refused" check-project common/check-project 1

# ⛔ THE OWNING ENTRY'S OWN SECTION, NOT ITS FILE. A plant naming a real but
# unrelated entry survived this rule twice before it was narrowed this far.
printf '# bit-ids:python-exception=CI-10\nprint("plant")\n' >"$TREE/tools/check/plant.py"
agree "project a .py whose entry never mentions it is refused" check-project common/check-project 1

# ⭐ AND THE ACCEPT CASE, which is the half a harness of refusals never looks at.
# The argument is written into the entry's section, so the rule is satisfiable.
awk -v want="## CI-10:" '
  index($0, want) == 1 { print; print "tools/check/plant.py is approved here."; next }
  { print }' "$WORK/ci.md.projsrc" >"$TREE/TODO/ci.md"
agree "project a .py its entry argues for is accepted" check-project common/check-project 0
unplant tools/check/plant.py
cp "$WORK/ci.md.projsrc" "$TREE/TODO/ci.md"

# -- 10. every action is pinned to an immutable form --------------------------
PWF="$TREE/.github/workflows/plant.yml"
# ⚠ ASSEMBLED RATHER THAN WRITTEN WHOLE. A bare forty-character hex run in a
# tracked file is what `check-no-secrets --public` refuses, and it is right to:
# the allowance it carries is anchored to a `uses:` line, which this is not.
HEX40=$(printf '%s%s%s' 0123456789abcdef 0123456789abcdef 01234567)

printf 'on: push\njobs:\n  j:\n    steps:\n      - uses: actions/checkout@v4\n' >"$PWF"
agree "project an action pinned to a tag is refused" check-project common/check-project 1

printf 'on: push\njobs:\n  j:\n    steps:\n      - uses: actions/checkout@%s\n' "$HEX40" >"$PWF"
agree "project a pin with no version comment is refused" check-project common/check-project 1

printf 'on: push\njobs:\n  j:\n    steps:\n      - uses: actions/checkout@%s # v4.2.2\n' "$HEX40" >"$PWF"
agree "project a pin with a version comment is accepted" check-project common/check-project 0

printf 'on: push\njobs:\n  j:\n    steps:\n      - uses: actions/checkout\n' >"$PWF"
agree "project a uses: with no ref at all is refused" check-project common/check-project 1

printf 'on: push\njobs:\n  j:\n    steps:\n      - uses: docker://alpine:3\n' >"$PWF"
agree "project a container not pinned to a digest is refused" check-project common/check-project 1

# ⭐ A LOCAL ACTION IS THIS REPOSITORY, reviewed with everything else, so it
# needs no pin - the branch a denylist of floating forms would never reach.
printf 'on: push\njobs:\n  j:\n    steps:\n      - uses: ./.github/actions/plant\n' >"$PWF"
agree "project a local action needs no pin" check-project common/check-project 0

# -- 11. an artifact a workflow downloads is one some workflow uploads --------
{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - uses: actions/download-artifact@%s # v8.0.0\n' "$HEX40"
  printf '        with:\n          name: plant-nobody-uploads\n'
} >"$PWF"
agree "project a download nothing uploads is refused" check-project common/check-project 1

{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - uses: actions/upload-artifact@%s # v7.0.0\n' "$HEX40"
  printf '        with:\n          name: plant-pair\n'
  printf '      - uses: actions/download-artifact@%s # v8.0.0\n' "$HEX40"
  printf '        with:\n          name: plant-pair\n'
} >"$PWF"
agree "project a download its own workflow uploads is accepted" check-project common/check-project 0

# ⛔ A MARKER THAT OUTLIVED ITS REASON IS REFUSED TOO, which is what stops this
# declaration becoming a permanent exemption.
{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - uses: actions/upload-artifact@%s # v7.0.0\n' "$HEX40"
  printf '        with:\n          name: plant-pair\n'
  printf '      - uses: actions/download-artifact@%s # v8.0.0\n' "$HEX40"
  # ⚠ INSIDE THE STEP IT DECLARES, which is where the tree spells it: a marker
  # above the step belongs to the step before it, and the parser is right to say
  # so rather than to guess.
  printf '        # bit-ids:no-producer=CI-09\n'
  printf '        with:\n          name: plant-pair\n'
} >"$PWF"
agree "project a no-producer marker over a produced name is refused" check-project common/check-project 1

# ⚠ `pattern:` COUNTS TOO: `download-artifact` accepts either key, so a rule
# reading only `name:` is a gate on one of two doors.
{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - uses: actions/download-artifact@%s # v8.0.0\n' "$HEX40"
  printf '        with:\n          pattern: plant-nobody-uploads-*\n'
} >"$PWF"
agree "project a download written as pattern: is read too" check-project common/check-project 1

# -- 22-24. the same language behind a second door ----------------------------
{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - name: plant\n        shell: pwsh\n        run: |\n'
  printf "          \$ErrorActionPreference = 'Stop'\n"
  printf '          Write-Output ok\n'
} >"$PWF"
agree "project a workflow pwsh block that stops without the native rule is refused" \
  check-project common/check-project 1

{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - name: plant\n        shell: pwsh\n        run: |\n'
  printf "          \$ErrorActionPreference = 'Stop'\n"
  # shellcheck disable=SC2016
  printf '          $PSNativeCommandUseErrorActionPreference = $false\n'
  printf '          Write-Output ok\n'
} >"$PWF"
agree "project a workflow pwsh block that says both is accepted" check-project common/check-project 0

{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - name: plant\n        shell: pwsh\n        run: |\n'
  printf "          Write-Error 'plant'\n"
} >"$PWF"
agree "project a workflow pwsh block reporting through Write-Error is refused" \
  check-project common/check-project 1

# ⛔ AN INVERTED GUARD FAILS THE STEP BY SUCCEEDING AT ITS JOB, because GitHub
# reads the block's residual exit code as the verdict.
{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - name: plant\n        shell: pwsh\n        run: |\n'
  printf '          git status\n'
  # shellcheck disable=SC2016
  printf '          if ($LASTEXITCODE -ne 0) { Write-Output no }\n'
} >"$PWF"
agree "project a pwsh block leaving LASTEXITCODE as the verdict is refused" \
  check-project common/check-project 1

{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - name: plant\n        shell: pwsh\n        run: |\n'
  printf '          git status\n'
  # shellcheck disable=SC2016
  printf '          if ($LASTEXITCODE -ne 0) { Write-Output no }\n'
  printf '          exit 0\n'
} >"$PWF"
agree "project a pwsh block that ends in an explicit exit is accepted" check-project common/check-project 0
unplant .github/workflows/plant.yml

# -- 12. the working tree agrees with .gitattributes --------------------------
#
# ⚠ A `.bat` RATHER THAN A `.ps1`, so this plant trips the line-ending rule and
# nothing else: the three `.ps1` rules below have their own fixtures.
printf 'echo plant\n' >"$TREE/plant.bat"
plant_tracked plant.bat
agree "project a working tree ending that contradicts .gitattributes is refused" \
  check-project common/check-project 1
unplant_tracked plant.bat

# -- 13-15, 17. the tracked-only scopes ---------------------------------------
# ⚠ THE NEEDLE IS ASSEMBLED, because this file is under `scripts/` and the
# rule reads every tracked file there: written whole, the harness would be the
# thing it refuses. That is this repository's own recurring shape - a harness
# that plants a pattern cannot spell it.
printf 'set -u\n# mvdan.cc/sh/v3/cmd/%s@v0.0.1\n' shfmt >"$TREE/scripts/plant.sh"
plant_tracked scripts/plant.sh
agree "project an shfmt version that disagrees with the pin is refused" \
  check-project common/check-project 1

# ⛔ RE-STAGED AFTER EVERY REWRITE. `git rm --cached` refuses a path whose
# staged content matches neither HEAD nor the working tree, so a plant rewritten
# three times could not be unplanted and survived into every case after it.
printf 'echo plant\n' >"$TREE/scripts/plant.sh"
plant_tracked scripts/plant.sh
agree "project a script that does not state set -u first is refused" \
  check-project common/check-project 1

# ⚠ Assembled for the reason above: spelled whole, this line is itself a cargo
# output path composed without `CARGO_TARGET_DIR`.
printf 'set -u\nBIN=target/%s/examples/thing\n' debug >"$TREE/scripts/plant.sh"
plant_tracked scripts/plant.sh
agree "project a cargo output path composed by hand is refused" \
  check-project common/check-project 1

printf 'set -u\n# see: cargo test some_name\n' >"$TREE/scripts/plant.sh"
plant_tracked scripts/plant.sh
agree "project a cargo test in a script is out of this rule's scope" \
  check-project common/check-project 0
unplant_tracked scripts/plant.sh

sed 's/^SHFMT_VERSION=/SHFMT_PIN_REMOVED=/' "$WORK/provision.sh.projorig" >"$PPRV"
agree "project a provision script with no shfmt pin is refused" check-project common/check-project 1
restore_project

# -- 16. every dependency comes from the registry, with a checksum ------------
sed 's|^source = "registry+https://github.com/rust-lang/crates.io-index"$|source = "git+https://example.invalid/x"|' \
  "$WORK/Cargo.lock.projorig" >"$PLCK"
agree "project a dependency from somewhere other than the registry is refused" \
  check-project common/check-project 1
restore_project

grep -v '^checksum = "' "$WORK/Cargo.lock.projorig" >"$PLCK"
agree "project a dependency with no checksum is refused" check-project common/check-project 1
restore_project

# -- 17. an acceptance that can pass over nothing -----------------------------
#
# ⛔ `cargo test` WITH A BARE WORD SELECTS BY NAME, and a filter matching none
# prints `running 0 tests` for every binary and exits 0.
# shellcheck disable=SC2016
printf '\nProve: run `cargo test plant_name` and read it\n' >>"$TREE/TODO/ci.md"
agree "project a Prove selecting tests by name is refused" check-project common/check-project 1
cp "$WORK/ci.md.projsrc" "$TREE/TODO/ci.md"

# shellcheck disable=SC2016
printf '\nProve: run `cargo test --workspace --locked` and read it\n' >>"$TREE/TODO/ci.md"
agree "project a Prove selecting by package rather than name is accepted" \
  check-project common/check-project 0
cp "$WORK/ci.md.projsrc" "$TREE/TODO/ci.md"

# ⚠ THE WORKFLOW IS THE SECOND DOOR into the same mistake, and it is read by a
# different extractor with the same judgement.
{
  printf 'on: push\njobs:\n  j:\n    steps:\n'
  printf '      - run: cargo test plant_name\n'
} >"$PWF"
# ⚠ TRACKED, because this rule's scope is `git ls-files` rather than a glob -
# unlike the pin and artifact rules above, which read the directory. A plant left
# untracked would have all three implementations agreeing that they saw nothing.
plant_tracked .github/workflows/plant.yml
agree "project a workflow cargo test selecting by name is refused" check-project common/check-project 1
unplant_tracked .github/workflows/plant.yml

# -- 18. a git dependency in a manifest ---------------------------------------
mkdir -p "$TREE/plantcrate"
printf '[dependencies]\nthing = { git = "https://example.invalid/x" }\n' >"$TREE/plantcrate/Cargo.toml"
agree "project a git dependency in a manifest is refused" check-project common/check-project 1
rm -rf "$TREE/plantcrate"

# -- 19-21. the three .ps1 rules ----------------------------------------------
printf 'Write-Output "a plant \342\233\224 here"\n' >"$TREE/plant.ps1"
agree "project a .ps1 with non-ASCII and no BOM is refused" check-project common/check-project 1

printf '\357\273\277Write-Output "a plant \342\233\224 here"\n' >"$TREE/plant.ps1"
agree "project a .ps1 with non-ASCII and a BOM is accepted" check-project common/check-project 0

printf "\$ErrorActionPreference = 'Stop'\nWrite-Output ok\n" >"$TREE/plant.ps1"
agree "project a .ps1 that stops without the native rule is refused" check-project common/check-project 1

printf "\$ErrorActionPreference = 'Stop'\n\$PSNativeCommandUseErrorActionPreference = \$false\n" >"$TREE/plant.ps1"
agree "project a .ps1 that says both is accepted" check-project common/check-project 0

printf "Write-Error 'plant'\n" >"$TREE/plant.ps1"
agree "project a .ps1 reporting through Write-Error is refused" check-project common/check-project 1

# ⚠ THE NEEDLE IS AN INVOCATION, NOT THE WORD, and this is the case that proves
# it: a rule matching the name anywhere fires on the file that describes it.
printf "# Write-Error is what this comment is about\nWrite-Output ok\n" >"$TREE/plant.ps1"
agree "project a .ps1 that only mentions Write-Error is accepted" check-project common/check-project 0
unplant plant.ps1

agree "project clean tree again" check-project common/check-project 0

# ============================================================================

store_report check-bitcheck/1 cases \
  "$([ "$JSON" = "1" ] && printf 1 || printf 0)"
