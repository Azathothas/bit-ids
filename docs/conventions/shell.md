# shell.md

Traps in passing text and reading results between shells, and the rules that
avoid them. Every entry here is a defect that actually happened, and most of
them fail silently, which is why they are worth a document.

The shape of almost all of them is the same: **a payload crosses a boundary and
loses its quoting, or a result crosses a boundary and loses its meaning.**

---

## 1. A prose payload goes through a file. Not through a shell.

⛔ **Write the text to a file with a file-writing tool, then pass the path.**
This applies to a commit message, a document, a script, a JSON body, anything
multi-line, and anything containing an apostrophe, a backtick, a dollar sign or
a backslash.

The reason it is a file and not "better quoting" is that quoting is not
sufficient. Measured on 2026-08-25:

| how the payload travelled | result |
| --- | --- |
| written to a file, then read by the shell | 8657 bytes, byte-exact, exit 0 |
| passed inline to `bash -c` inside a **quoted** heredoc `<<'EOF'` | the backticks in the prose were **executed**: `origin: command not found` |

The second row is the surprising one. A quoted heredoc is supposed to be
literal, and when the payload is handed to a shell as an inline string it is
not reliably so. The minimal reproduction is one line of text containing
`` `backticks` `` and it fires with LF endings and with CRLF endings alike.

⚠ **The way this fails is worse than an error.** The file is written, nothing
returns non-zero, and the damage is a substituted or truncated fragment
somewhere in the middle of a long document. The first sign is usually a commit
subject that reads like a fragment of the body.

Related failures with the same cause:

- A PowerShell here-string written inside a `bash` command is parsed by bash
  first. `@'...'@` is an `@`, a single-quoted string, and an `@`, so it ends at
  the first apostrophe in the text. The phrase "the run's own deadline" turns
  the rest of a commit message into shell commands.
- `python -c` and `python - <<'PY'` are fine for code with no apostrophes and
  no backslashes. A Windows path in a Python string literal has both.
- A backslash escape that survives one hop loses a backslash on the next, and
  the receiving language reads what is left as an escape sequence. `\b` becomes
  a backspace byte and `\f` becomes a form feed. The file is written, nothing
  errors, and a regex that was supposed to end in a word boundary now ends in a
  byte no editor shows.

⚠ **Some agent harnesses collapse `\\` to `\` before the shell sees it.** Verify
it in the environment you are in rather than assuming either way:

```bash
printf 'literal: C:\\Users and regex \\d+\n'
```

If the output shows one backslash where you wrote two, every literal double
backslash has to go through a file-writing tool instead.

### ⭐ The channel that cannot be reached into: base64

When a payload has to cross a shell at all, **base64 is the one encoding no
shell interprets.** It is `[A-Za-z0-9+/=]` and needs no quoting anywhere: not
in bash, not in PowerShell, not in `cmd`. A quote, a backtick, a dollar sign,
a percent, an emoji and an indented terminator all survive it unchanged.

That makes it the right transport for a helper that writes files:

| channel | when |
| --- | --- |
| ⭐ **base64 argument** | anything with quoting hazards. The bulletproof one. |
| **copy from another file** | the payload already exists on disk |
| **stdin** | ⚠ only behind a pipe, and only from a POSIX shell. See below. |

⛔ **PowerShell's stdin to a native command is NOT byte-exact, and Git Bash's
is.** Measured on one 59-byte fixture: piping it through PowerShell wrote **61**
bytes, because PowerShell's native-command pipe appends a trailing CRLF. The
tail `3e 20 3c 0a` arrived as `3c 0a 0d 0a`. The same file piped through Git
Bash was byte-identical, as were the base64 and copy-from-file paths from
**both** shells.

⚠ A receiving tool cannot tell an intended trailing newline from an added one,
so it must not guess. **From PowerShell, use base64 or copy-from-file. Reserve
stdin for pipes in a POSIX shell.**

⭐ Two properties worth building into any such helper, because both turn a
silent failure into a loud one:

- **Write atomically**: a temp file in the *same* directory, then rename. A
  killed process leaves the old file intact rather than a truncated one. Same
  directory matters: a rename across volumes is a copy and loses the guarantee.
- **Require an expected match count on a substitution.** A replace that matches
  a different number of times than you believed is refused, and the file is
  left untouched. ⛔ A silent no-op that reports success is the failure that
  discipline exists to remove, and it is the exact shape that bit this
  repository twice while it was being written.

---

## 2. An exit code is read from the process that produced it, unpiped

⛔ Piping a check into anything reports the **pipeline's** status, not the
check's, so a guard that failed reads as green.

```bash
node scripts/check-thing.mjs
```

```bash
pwsh -NoProfile -File scripts/check-thing.ps1
```

Not `check | grep`, not `check | Select-String`, not `check | tee`. Run it,
read `$?` or `$LASTEXITCODE`, then look at the output separately if you need to.

⛔ **AND TWO CHECKS JOINED BY `&&` ARE ONE CHECK.** `check-a && check-b` runs
`check-b` only when `check-a` says nothing, so a finding in the first means the
second never ran - and the terminal shows one block of output, which reads
exactly like both of them passing.

⚠ **Measured on 2026-09-09, on this repository's own gate work.**
`shellcheck file && shfmt -d file >/dev/null && echo "LINT OK"` was run over a
script that had a real formatting defect. shellcheck reported an *info*-level
finding and exits 1 for that, so shfmt was never invoked, "LINT OK" was never
printed - and the missing line was read as the diff being empty. ⭐ The defect was
caught eleven minutes later by `check-workflow`, whose own control runs the two
as separate commands the way `ci.yml` does.

⛔ **AND AN IN-PLACE `sed` EDITS EVERY LINE THAT MATCHES, NOT THE ONE YOU MEANT.**
Measured the same day, while timing a harness against different values of a
constant: `sed -i 's/^SECS=[0-9]*/SECS=6/' check-capture.sh` was written to change
one assignment and changed two, because `[0-9]*` matches **zero** digits - so
`SECS="$2"`, a line inside an embedded stub script, became `SECS=6"$2"`.

⚠ **It fails the way `replace_once` was built to prevent**: the edit lands, the
command exits 0, and nothing says the file now has a second change in it. ⭐ The
rule is the one that function already states - an edit whose match count you have
not established is an edit somewhere you did not look - and it applies to a
one-off command at a prompt exactly as it does to a harness.

⭐ **So run each one and read each status.** The CI step this repository ships is
two lines for exactly this reason:

```bash
find scripts -name '*.sh' -print0 | xargs -0 shellcheck
find scripts -name '*.sh' -print0 | xargs -0 shfmt -d -i 2 -ci
```

The same rule in PowerShell has a second edge: `-ErrorAction SilentlyContinue`
suppresses the error *output* while the cmdlet failure still sets a failing
status. To make a failure genuinely non-fatal, promote it and swallow it:

```powershell
try { Some-Cmdlet -ErrorAction Stop } catch { }
```

---

## 3. stdout and stderr are different streams and merging them is a decision

⛔ **Anything reading a value reads stdout alone and checks the exit code.**
Merging is correct only when the thing you want is on either stream.

The worked example, from this repository's own probe: `git rev-parse
--abbrev-ref HEAD` in a repository with no commits prints `HEAD` to stdout
**and** a three-line fatal to stderr, exiting 128. A version of the probe that
merged the streams put that fatal into a field called `branch`.

The opposite case is equally real: `java -version` prints the version to
**stderr**, so a probe reading stdout alone finds nothing. Merge on purpose
there, and say why in a comment.

---

## 4. A subshell discards the assignment you made in it

⛔ In POSIX shells, a function called inside `$( )` runs in a subshell. Any
variable it sets is gone when it returns.

```sh
collect() { FOUND="$FOUND $1"; printf 'value'; }   # FOUND is lost
x=$(collect a)                                      # ...because of this
```

Signal through the exit code or the output, and let the caller record it:

```sh
x=$(collect a); rc=$?
[ "$rc" = 3 ] && FOUND="$FOUND a"
```

The same shape appears with `while read` on the right of a pipe: the loop body
runs in a subshell and its assignments vanish. A here-document redirect does
not create one, which is why a lookup table is fed to the loop that way.

---

## 5. Line endings

A carriage return in a file `.gitattributes` says is LF is **invisible to git
and visible to everything else.** The index is normalised either way, so
`git diff` shows nothing and a review cannot see it.

It is not invisible to a regex reading the working tree. In .NET, `(?m)^...$`
matches before the newline and leaves the carriage return inside the capture,
so a status cell reads as `done` plus a byte and matches nothing.

⚠ The drift arrives from your own tooling. `Set-Content` writes CRLF on Windows
by default, and so do most editors and most file-writing tools.

Two things fix it and neither is a manual step anybody has to remember:

- the repository [`.gitattributes`](../../.gitattributes), which states the
  rule per type;
- a check that compares every tracked file's working-tree endings against what
  `.gitattributes` resolves for it, using git's own answer rather than a second
  table:

```bash
git ls-files --eol
```

⛔ **That check is a row in `check-project`, in both halves, and until
2026-09-08 it was only this paragraph.** The document named the rule and nothing
enforced it, which is the shape this repository calls a preference stated as a
rule - and it was found by being broken rather than by being read: a file-writing
tool that emits LF left `check-project.ps1` at `w/lf` under `attr/text eol=crlf`,
a full gate was green over it, and the only thing that said so was an incidental
warning from `git commit`.

⚠ **Both directions are planted, because they are different branches.** A `.ps1`
given LF under `eol=crlf`, and a `.sh` given CRLF under `eol=lf` - the second
being the case this section opens with. ⭐ `git diff` prints **nothing** for
either, which is what makes the working tree the only place they can be read.

⚠ **`w/none` is not a disagreement.** A file with no line ending at all carries
no evidence either way, and refusing it would fire on a correct tree.

⚠ **`.ps1` is the one file type that keeps CRLF.** Windows PowerShell 5.1
mis-parses a here-string whose terminator arrives with a bare LF. The simpler
defence, and the one this template uses, is to write no here-strings in a
`.ps1` at all.

---

## 6. A control byte goes in a file as an escape, never as itself

⛔ A literal control byte makes the file invisible to both review tools. `grep`
calls it binary and skips it, saying so in a line nobody reads, and `git diff`
prints "Binary files differ" so a code review of the file shows no diff at all.

The runtime value is identical either way, so only reviewability is ever at
stake, which is exactly why it survives so long unnoticed.

Write `\0`, `\t`, `\x1f`. Never the byte. Guard it with a check that fails
rather than warns, over every tracked text file.

---

## 7. Windows specifics

- ⭐ **Git Bash rewrites arguments that look like POSIX paths.** Anything with a
  leading slash is converted to a Windows path before the target process sees
  it. When the target is not a Windows program, the rewrite is corruption, it
  is silent, and the error never names the cause. Measured on 2026-08-26:

  ```bash
  gh api /repos/OWNER/NAME/actions/workflows
  ```

  ```text
  invalid API endpoint: "C:/Program Files/Git/repos/OWNER/NAME/actions/workflows".
  ```

  `gh` happens to detect it and say so. Almost nothing else does: a container
  runtime receives the rewritten path as a real argument and acts on it.

  Two variables turn it off, and they cover different things. `MSYS_NO_PATHCONV`
  disables the leading-path heuristic; `MSYS2_ARG_CONV_EXCL` is a per-argument
  exclusion list, and `'*'` excludes everything. ⛔ **Any command whose POSIX
  paths are destined for a non-Windows process carries both:**

  ```bash
  MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*' podman run --rm alpine ls /etc
  ```

  ⚠ This is the root cause behind the reserved-name bullet below, which is why
  the two are next to each other.
- ⛔ **The Windows reserved device names are `CON`, `PRN`, `AUX`, `NUL`, `COM1`
  to `COM9` and `LPT1` to `LPT9`**, in any case and with any extension. Two
  different triggers create one as a real file, and the second is the one nobody
  expects:

  1. **Your own redirect.** `2>/dev/null` under a shell that does not map
     `/dev/null` creates a file called `nul`, which git then tracks, which
     breaks `git stash` outright, and which cannot be deleted by `rm` or by
     Python.
  2. ⚠ **A tool's own argument list.** `podman machine ssh` on Windows passes
     `-o UserKnownHostsFile=NUL` to its own ssh invocation. Under Git Bash that
     is a filename, not the null device, so a 99-byte `NUL` holding an ssh host
     key appears in whatever directory the command ran in. Measured on
     2026-08-27 with `MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'` already set:
     **the prefix above does not prevent this one**, because the argument never
     looked like a path.

  ⚠ The two differ in recoverability, so do not assume the worse case is the
  only case. The `NUL` written by trigger 2 was removed by `rm` on the same
  machine; the lowercase `nul` from trigger 1 was not. Put the whole reserved
  set in `.gitignore` before any of it happens, because the directory it lands
  in is usually a repository.
- ⚠ **`/tmp` is not one directory.** Git Bash resolves it inside the msys root;
  a native Windows Python or PowerShell resolves it somewhere else entirely, or
  not at all. A file written by one and read by the other is not found. Use a
  repository-relative scratch directory, or an absolute path both agree on.
  This document's author hit it while testing the probe.
- ⚠ **A shim is not an executable.** On Windows the node ecosystem ships shims,
  and scoop's are `.ps1`. `Process.Start` with `UseShellExecute` false throws
  "not a valid application for this OS platform" on a `.ps1` and refuses a
  `.cmd`. Route a `.ps1` to a PowerShell host and a `.cmd` to `cmd.exe`.
- ⚠ **`wsl.exe` writes UTF-16LE**, which a redirected stdout reads as empty or
  as mojibake.
- ⚠ **A machine-wide install is not under the user's home.** Checking only
  `~/scoop` reports a tool as absent on a machine that has it under
  `C:\ProgramData\scoop`. Look in both.
- ⚠ **A release binary left running holds its own executable open**, and the
  next build fails on a locked file with an error naming neither. Kill stray
  processes before rebuilding.
- ⛔ **Python on Windows cannot print this repository's own markers.** stdout
  defaults to cp1252, which has no ⛔, no ⭐ and no ⚠. Measured on 2026-08-27,
  Python 3.13.15:

  ```bash
  python -c "print('⛔')"
  ```

  ```text
  UnicodeEncodeError: 'charmap' codec can't encode character '⛔'
  in position 0: character maps to undefined
  ```

  ⭐ Note what the error itself does: it names the character as a codepoint,
  because it cannot print it either.

  ⚠ The failure is at print time, so it passes every test that captures output
  and fails the moment a person runs it at a console. Any script that echoes a
  marker sets `PYTHONIOENCODING=utf-8` or calls
  `sys.stdout.reconfigure(encoding='utf-8')` before printing. Where the encoding
  is not yours to control, print the codepoint instead of the character.
- ⛔ **A byte class is not a character class, and the wrong one is silently
  wrong.** `grep -o '[^\x00-\x7F]'` returns per-byte fragments, so a three-byte
  marker counts as three separate entries and the total is wrong in a way that
  looks like real output. Measured on 2026-08-27 over a file holding exactly one
  ⛔ and one ⚠:

  | tool | answer |
  | --- | --- |
  | `grep -o '[^\x00-\x7F]'` | 6 fragments, none of them a character |
  | `rg -o '[^\x00-\x7F]'` | `1 ⚠`, `1 ⛔` |

  ⚠ **Setting `LC_ALL=C` does not rescue the first row**, and assuming it does
  is the trap. On the measured machine `LANG` was already empty, so `LC_ALL=C`
  changed nothing at all. The fix is choosing the right tool, not the locale.

  ⛔ **The same trap bites a TEST that sets it.** Measured on 2026-09-08 while
  driving the gate under a deliberately hostile environment: `LC_ALL=C` was one
  of four variables changed, the verdict was identical, and the locale half of
  that pass had established nothing, because `locale` reported `POSIX` already.
  ⭐ Read the host's locale before claiming to have varied it, and vary it to
  something it is not - here, `C.utf8`.

  ⛔ **A check states which of the two jobs it is doing**, because the same
  expression is correct for one and quietly wrong for the other. Counting bytes
  is byte-oriented and belongs to a byte tool. Counting characters is
  character-oriented and needs a Unicode-aware one. This matters most to
  [`tools/check/controlbytes.go`](../../tools/check/controlbytes.go),
  whose whole subject is bytes that review tools misreport.

---

## 8. PowerShell specifics

- ⛔ **`[int]` on a double rounds.** `[int](2.65)` is 3, so a 2h39m session
  prints as 3h39m and the number goes straight into a report. Use
  `[math]::Floor`.
- ⛔ **`-match` is case-insensitive**, so `'FAILED'` matches `"0 failed"` in a
  summary line and a failing test's name is lost exactly when it is needed.
  Use `-cmatch` when case is the signal, and filter on the per-test line rather
  than the summary.
- ⛔ **`$args` inside a function is an automatic variable** and silently
  swallows a parameter of that name. Variable names are case-insensitive, so
  `$Args` collides too. Name locals so they cannot.

  ⚠ **The collision does not need an automatic variable to bite**, and it has
  twice here: a `[switch]` parameter against an ordinary local differing only in
  case is one variable, so the local's value is assigned to the switch and the
  script fails to bind in **every** mode. Both instances were found by running the
  script once rather than by reading it, and `TODO/ci.md` carries them.
- ⛔ **`$PSNativeCommandUseErrorActionPreference` is set explicitly in every
  `.ps1` here, and the reason is that its default CHANGED.** It is `$false` in
  pwsh 7.4 and `$true` from 7.5, where a native command exiting non-zero becomes
  a terminating error under `$ErrorActionPreference = 'Stop'`. ⚠ **A guard that
  REFUSES then stops being a code the caller can read and becomes an exception
  nobody caught**, which is this repository's oldest rule broken by an upgrade
  rather than by an edit.

  Measured on 2026-09-08: `capture-run.ps1` was green on a 7.4 host and turned
  the `ubuntu-24.04` lane red, on the two cases that assert the guard's refusal.
  The Windows lane, on an older pwsh, stayed green - so the two lanes disagreed
  about the same script for a reason neither one printed. ⚠ This note used to
  say only "defaults to false from pwsh 7.4", which is a fact about one version
  being relied on as a guarantee; that sentence is what let the assumption
  spread to sixteen files.

  `check-project` refuses a `.ps1` that sets `$ErrorActionPreference = 'Stop'`
  without also setting this one, **and it reads the `pwsh` blocks of every
  workflow too.** ⚠ That second door was open the whole time: the rule iterated
  `git ls-files '*.ps1'`, and `capture.yml` was carrying three blocks that set
  the one preference and not the other.
- ⛔ **A workflow's `pwsh` step ends in an explicit `exit`, because the residual
  `$LASTEXITCODE` IS the step's verdict.** GitHub's wrapper reads whatever the
  block left behind, so a block whose last command is a guard fails the step with
  that guard's code - even when the code was the outcome the step wanted.

  ⚠ **The shape that bites is an INVERTED guard**, one whose refusal is the
  proof. Measured on 2026-09-08 on the first dispatch of the capture workflow:
  its Windows *Restore the route* step, retired with the route cut on
  2026-10-15, put the default routes back and then ran
  the egress guard, which must now REFUSE, because a host that can reach the
  network again is one that guard refuses. It refused, exit 1, exactly as
  designed - and that 1 failed the step, so the evidence upload was skipped and a
  finished capture was thrown away over a route that had already come back.

  ⭐ The `sh` twin never had it, because `if some_guard; then …; fi` consumes the
  status and an `if` whose condition is false has status 0. A convention held on
  one of the two paths into one mistake, and only the path GitHub reads was
  wrong.

  ```powershell
  ./scripts/acquisition/assert-disposable.ps1 -Egress
  $egress = $LASTEXITCODE
  if ($egress -eq 0) { [Console]::Error.WriteLine('the route did not come back'); exit 1 }
  if ($egress -ne 1) { [Console]::Error.WriteLine("the guard could not run (exit $egress)"); exit 1 }
  exit 0
  ```

  ⚠ **And read all three codes while you are there.** `0`, `1` and `2` are
  *passed*, *refuses* and *could not run*; an `if` folds the last two together,
  so a guard that never got as far as a routing table reads as one that read a
  restored route off it. `check-project` refuses a workflow `pwsh` block that
  reads `$LASTEXITCODE` and does not end in an `exit`. ⚠ The rule is scoped to
  workflows rather than to every `.ps1`, because a script's own exit code is its
  author's to choose and it is only here that something else reads the leftovers.

  ⛔ **The wrapper is `pwsh -command ". '<file>'"`, and a dot-sourced script's
  exit code COLLAPSES through it.** Measured on 2026-09-08, pwsh 7.4.6:

  | the block exits | through `-command ". '<file>'"` | through `-File` |
  | ---: | ---: | ---: |
  | 0 | 0 | 0 |
  | 1 | 1 | 1 |
  | 2 | **1** | 2 |
  | 3 | **1** | 3 |
  | 42 | **1** | 42 |

  ⚠ **So a `pwsh` step's exit code says *failed* and never says WHICH refusal
  fired.** That is the second argument for `[Console]::Error.WriteLine` above:
  the message is the only channel that survives the wrapper, so a block that
  distinguishes its cases says which one it got in words rather than in a code
  nobody downstream will see.
- ⛔ **A message a machine matches on is written with
  `[Console]::Error.WriteLine`, never `Write-Error`.** Called inside a script,
  `Write-Error` renders a source-context block - the file, the line, a caret
  rule - and **wraps the message to the host's width**. Measured on 2026-09-08,
  the same refusal:

  ```text
       | capture-run: the host is claimed by run [capture-0001], not [capture-0002]
  ```

  at width 200, and at width 80:

  ```text
       | capture-run: the host is claimed by run [capture-0001], not
       | [capture-0002]
  ```

  ⚠ **A fixed-string match therefore passes on a developer host and fails on a
  CI runner**, which is what it did. Ten `.ps1` files here already wrote the
  bytes and five did not; `check-project` refuses the rest. ⚠ Its needle is the
  name in command position rather than the word, because the rule fired on its
  own enforcing file first: the failure message contains the name. ⛔ **And the
  rule reads a workflow's `pwsh` blocks as well**, which it did not until
  `CI-06`'s dispatch found `capture.yml` reporting a refusal through exactly this
  cmdlet, one door away from the rule that forbids it.
- ⚠ **`Get-Command` finds cmdlets, functions and aliases too.** Filter to
  `Application` and `ExternalScript` when you mean an executable. A cmdlet
  looked for on PATH reports as missing on every machine that has it.
- ⚠ **Read the child's streams before waiting on it.** Calling `WaitForExit`
  first deadlocks any child that fills the pipe buffer: the child blocks on
  write, the parent blocks on the wait, and neither moves until the timeout.
- ⚠ **`ConvertTo-Json` defaults to depth 2**, and renders anything deeper as
  the literal text `System.Collections.Hashtable`. Pass `-Depth`.
- ⛔ **A file write names its encoding.** `Set-Content`, `Add-Content` and
  `Out-File` otherwise write what the host chooses: UTF-8 with no BOM under
  PowerShell 7 and the ANSI code page under 5.1. Pass `-Encoding utf8NoBOM`,
  which fails loudly on 5.1 rather than writing ANSI. `check-project` refuses
  an unstated write in a `.ps1` and in a workflow block (`CI-08`).
- ⛔ **A `.ps1` containing any non-ASCII byte needs a UTF-8 BOM if Windows
  PowerShell 5.1 has to run it.** 5.1 decodes a BOM-less file as the system ANSI
  code page, so every non-ASCII character is mis-decoded. PowerShell 7 defaults
  to UTF-8 and does not care, which is exactly why this is easy to miss: the
  file works on the machine it was written on and breaks on the one it was
  written for. `PSUseBOMForUnicodeEncodedFile` is the analyzer rule, and it
  caught this repository's own probe.
  ⚠ The alternative is to keep every `.ps1` ASCII-only. That is also defensible;
  what is not defensible is non-ASCII with no BOM and a claim of 5.1 support.

  ⭐ **This is a check now rather than a convention**, in both halves of
  `check-project`. It was a convention for as long as it took a door sweep to
  count: eleven files carried the BOM and four did not, and all four carry this
  project's markers on hundreds of lines. ⚠ The two halves disagreed on their
  first run, because a `.ps1` keeps CRLF and the sh half read the carriage
  return as non-ASCII, so it demanded a BOM for a file that is pure ASCII.
  `check-twins` could not see that: no `.ps1` in this tree is ASCII-only, so the
  branch that differed had nothing to exercise it.
- ⚠ **An empty `catch {}` is refused by PSScriptAnalyzer**, and it should be:
  it is indistinguishable from an accidentally swallowed error. Where swallowing
  is genuinely the design, say so in code rather than by omission. `$null = $_`
  discards the error explicitly and reads as a decision.

---

## 9. A long-running command needs a hard time limit

⛔ Several tools block for as long as you let them. `kubectl version` without
`--client` contacts a cluster, `gradle --version` starts a daemon, a cloud CLI
sits on an update check. A script that shells out to unknown tools without a
limit is a script that hangs, and a script that hangs is one nobody runs twice.

```bash
timeout 6 some-tool --version
```

`timeout` is absent on stock macOS. `gtimeout` is there when coreutils is
installed, and the fallback is to run in the background and kill on a counter.
Exit 124 is the coreutils verdict and 137 is a kill; **both mean "it never
answered", which is a different fact from "it is not installed"** and belongs in
a different field.

### ⛔ A bound inside `$( )` is not a bound

⛔ **A command substitution ends when the PIPE reaches end of file, not when the
command exits.** So `V=$(timeout 5 thing)` waits for as long as *anything* holds
that pipe's write end - and the one process `timeout` killed is usually not the
one holding it. A product that leaves a single child behind blocks the
substitution forever, with the bound sitting there having fired correctly.

```sh
V=$(timeout 5 sh "$ADAPTER" version)         # ⛔ blocks past the 5s, silently

timeout 5 sh "$ADAPTER" version >"$out" 2>"$err"   # ⭐ the product gets a FILE
V=$(cat "$out")                                     # and a file has an end
```

⚠ **Measured on 2026-09-16**: an adapter whose `version` left one `sleep` behind
was still blocking its substitution at **25 seconds** under a **4-second** bound.
That is how `capture-client` spent thirty minutes in a step that takes six
seconds, on three dispatches that produced no log and no artifact.

⭐ The same rule is why a CI step does not end when its command does: the runner
is reading a pipe too. Redirect the untrusted process to a file and read the
file.

### ⛔ `grep` exits 1 on no match, and `set -e` reads that as a failure

⛔ **A status captured on the NEXT line is never captured under `set -e`.** The
shell exits at the failing command, so the assignment, the report and any
trailing `true` are all unreachable.

```sh
timeout 60 grep -lE "$pat" -- *   # ⛔ exits 1 when nothing matches, which is
_rc=$?                            #    the ordinary case; -e ends the script here
true

timeout 60 grep -lE "$pat" -- * && _rc=0 || _rc=$?   # ⭐ captured where produced
```

⚠ This bites hardest in a diagnostic, where **no match is the expected result**.
Measured on 2026-09-16: two probe steps written to be harmless would have failed
their job instead, because GitHub runs a `run:` block as `bash -e` and both ended
in a `true` that could not be reached. ⭐ **A trailing `true` is not what makes a
block harmless - reaching it is.** Drive the block before trusting it.

### ⛔ An unprivileged bound cannot end a root process

⛔ **`timeout`, and every `kill` it sends, is refused by the kernel when the
target runs as root and the sender does not.** A bound placed *outside* `sudo`
fires on schedule, reports 124, and leaves the work running: measured on
2026-09-16, `timeout -k 2 5 sh -c 'sudo … sleep 120'` from a uid-1001 shell
exited 124 and left the root `sleep` alive with **PPID 1**, orphaned and past its
KILL grace.

```sh
timeout -k 30 900 sudo -E sh install.sh     # ⛔ the bound cannot signal root
sudo -E timeout -k 30 900 sh install.sh     # ⭐ sudo execs timeout, so it can
```

⚠ **Put the bound inside the `sudo`, not around it.** And note that nested
`timeout`s each create their **own** process group, so an outer bound does not
reach an inner one's children; a survivor is named by
[`../../scripts/ci/report-holders.sh`](../../scripts/ci/report-holders.sh)
rather than assumed absent.

---

## 10. Waiting inside an agent session

⚠ **Do not end a turn to wait for something.** The conversation idles, the
harness times out, and the session dies mid-operation with state half-changed.

⛔ **And do not reach for the harness's own scheduler, monitor or wake-up tool
to do the waiting.** They end the turn by design: they work by giving control
back and being re-invoked later. Substituting one is not holding, it is the
thing this section forbids, wearing a different name.

⚠ **This is the rule agents break most, and the reason is specific.** Many
harnesses block a foreground `sleep`. A session that has learned "hold with a
sleep loop", finds `sleep` refused, and concludes the rule cannot be followed
here, goes looking for a built-in that can wait. The conclusion is wrong: the
rule was never about `sleep`.

---

### ⭐ Wait on the work, not on the clock

**The best hold has no timer in it at all.** A blocking read costs no CPU,
needs no `sleep`, and ticks exactly when there is something to say.

**If the job prints anything, run it in the foreground and let its own output
be the tick.** There is nothing else to write:

```bash
long_running_thing 2>&1
```

The turn cannot idle while a process is writing to it, and every line is
progress a reader can see.

**If the job is already in the background, block on its log.** Measured on one
Windows 11 machine, 2026-08-28, Git Bash:

```bash
: > run.log
long_running_thing > run.log 2>&1 &
JOB=$!
tail -n +1 -f --pid="$JOB" run.log
wait "$JOB"
printf 'exit=%s\n' "$?"
```

⭐ `--pid` is what makes this terminate: `tail` stops when that process does,
so the hold ends by itself when the work does. Without it the pipeline outlives
the job and the session hangs on a command that will never return.

⚠ **The heartbeat belongs to the JOB, not to the waiter.** A job that can go
quiet for a long time prints its own progress line; the foreground just relays
it. That puts the interval where the knowledge is.

### ⛔ The trap that costs a whole turn

```bash
producer | while IFS= read -r line; do
  printf 'tick: %s\n' "$line"
  case "$line" in done) break ;; esac      # ⛔ does not stop `producer`
done
```

**Each side of a pipeline runs in its own subshell, so `break` leaves the loop
and nothing tells the producer to stop.** The producer keeps writing into a
pipe nobody is reading, and the command never returns.

Measured here: this exact shape ran to a two-minute tool timeout while its
output showed every tick arriving correctly. ⭐ **It looks like it worked right
up until it does not finish**, which is why it is worth a box rather than a
sentence.

The fix is to make the producer's own end the loop's end, which is what
`--pid` above does.

---

### When there genuinely is no signal to block on

Something outside this machine, with nothing local to read, is the only case
that needs a timer. `sleep` is one spelling of a timer and not the only one.
All of these were measured on the same machine and day, asked for 3 seconds:

| | elapsed | needs |
| --- | --- | --- |
| `sleep 3` | 3s | ⚠ blocked outright by some harnesses |
| `timeout 3 tail -f /dev/null` | 3s | coreutils `timeout` |
| `timeout 3 cat` | 3s | coreutils `timeout`, and a stdin nothing writes to |
| `perl -e 'select(undef,undef,undef,3)'` | 4s | perl |
| `[System.Threading.Thread]::Sleep(3000)` | 5s | ⚠ pwsh, and about 2s of that is pwsh starting |

⚠ **The last row is why a PowerShell tick is not free.** Starting `pwsh` per
tick costs roughly two seconds on this machine, so a loop of short ticks spends
most of its time launching a shell. Hold inside one `pwsh` process instead of
starting one per tick.

⚠ **`read -t` is not a portable answer.** It is a bash builtin; `/bin/sh` here
is dash and dash's `read` has no `-t`. And reading from `/dev/null` returns at
once on end-of-file rather than waiting, so it is not a timer even where the
flag exists.

⚠ **`ping` is not a portable timer either.** `-c` counts on POSIX and `-n`
counts on Windows, and against localhost the replies return instantly: `ping -c
3 127.0.0.1` finished in **0s** here.

The shape, when a timer really is needed:

```bash
i=0
while [ $i -lt 8 ]; do
  timeout 45 tail -f /dev/null
  i=$((i + 1))
  printf 'tick %s  %s\n' "$i" "$(date -u +%H:%M:%SZ)"
  if some_done_condition; then printf 'complete\n'; break; fi
done
```

⚠ **Keep a tick under four minutes** so progress is visible and nothing looks
hung, and note that the ceiling is per tick rather than per wait. A
forty-five-minute operation is ten holds that each print progress, never one
forty-five-minute wait.

---

### The summary

| the situation | hold with |
| --- | --- |
| ⭐ the job prints | run it in the foreground. Nothing else. |
| the job is backgrounded | `tail -f --pid=$JOB` on its log, then `wait` |
| the job is silent | make the job print, then as above |
| ⚠ waiting on something off this machine | a bounded timer loop, from the table |
| ⛔ any of the above | never a harness scheduler, monitor or wake-up |
