# History

This directory holds superseded explanations, reference sweeps, deep-review
records and dead ends worth preserving. Main documents state current truth and
are amended in place; they do not accumulate a session diary.

[`RESUME.md`](RESUME.md) is the one exception in purpose: it is overwritten at session
start and whenever the in-flight task changes so an interrupted session can be
reconstructed.

[`BOOTSTRAP-REVIEW.md`](BOOTSTRAP-REVIEW.md) records the required structural,
semantic, and release-readiness reviews of the initial repository.

[`SESSION-2026-09-04-ACQUISITION.md`](SESSION-2026-09-04-ACQUISITION.md) is the
saved summary of the session that closed `FOUND-03` and the blocking acquisition
entries, with the defects each review pass found. ⚠ A saved summary, not a
diary: it is one session's measurements, and the current state lives in
[`../../TODO/PROGRESS.md`](../../TODO/PROGRESS.md).

[`SESSION-2026-09-05-CORPUS.md`](SESSION-2026-09-05-CORPUS.md) is the saved
summary of the session that closed the corpus group and the first two publishing
entries, with what each review pass found and the two guards it could not
refute.

⚠ **Three records were missing from this list until 2026-09-08**, which is the
shape a directory index takes when each session adds a file and nobody re-reads
the page that describes them.

⛔ **And a fourth went missing the same day that sentence was written.**
`SESSION-2026-09-08-CLIENTS.md` was held in the tree only by a link from
[`RESUME.md`](RESUME.md), which is overwritten every session, so it was one
rewrite away from being an orphan and was already absent from this list. ⭐ The
observation is a check now rather than a note: `check-docs` refuses a
`SESSION-*.md` this page does not link, which is a stronger rule than the orphan
one it sits beside - a record linked from `RESUME.md` alone passes that and still
cannot be found from the page that exists to list them.

[`SESSION-2026-09-05-EVIDENCE.md`](SESSION-2026-09-05-EVIDENCE.md) and
[`SESSION-2026-09-05-OBSERVERS.md`](SESSION-2026-09-05-OBSERVERS.md) are the
saved summaries of the sessions that built the evidence writer and the observer
surfaces.

[`SESSION-2026-09-06-ADJACENT.md`](SESSION-2026-09-06-ADJACENT.md) is the saved
summary of the session that closed `OBS-11`, the adjacent surfaces, and the
three findings in it that were not about the code being written.

[`SESSION-2026-09-08-DISPATCH.md`](SESSION-2026-09-08-DISPATCH.md) is the saved
summary of the session that pressed the capture workflow's button for the first
time, wrote the queryable rendering, and built the adapter that answers what a
tool asks about its own identity.

[`SESSION-2026-09-08-CLIENTS.md`](SESSION-2026-09-08-CLIENTS.md) is the saved
summary of the session that measured the first two stock clients, over four
dispatches, and the seven defects none of which was found by reading.

[`SESSION-2026-09-08-ROUTES.md`](SESSION-2026-09-08-ROUTES.md) is the saved
summary of the session that read the aria2 hang out of an artifact rather than
dispatching for it, found that no route in this tree had ever been shown to
install anything, acquired one target twice, and turned three rules that existed
only in prose into checks.

[`SESSION-2026-09-09-SECONDROUTE.md`](SESSION-2026-09-09-SECONDROUTE.md) is the
saved summary of the session that wired the second acquisition route into the
capture workflow and answered the question it rested on - which artifact of a
release a route fetches. It also found that the record the work order was waiting
for cannot be written from any capture this project has run, refuted two more
readings of the aria2 hang, and cut the lane a contributor waits for from thirty
minutes to under four.

[`SESSION-2026-09-17-ROOTLESS.md`](SESSION-2026-09-17-ROOTLESS.md) is the saved
summary of the session that took the last privilege off the capture path and
measured that removing it does not end the hang, dissolved a recorded blocker by
noticing that a bound need not be a wrapper, made the assembler ask whether the
store it writes is a corpus at all, and deleted the last `common/` twin - whose
pre-deletion comparison found three defects in the half that was about to go.

[`SESSION-2026-09-29-PORT.md`](SESSION-2026-09-29-PORT.md) is the record of the
session that ported the last four twin pairs into the Go binary and ended
without running its ending. ⚠ It was reconstructed from the log and CI on
2026-10-11, and it names what that session left stale.

[`PROGRESS-2026-09-29.md`](PROGRESS-2026-09-29.md) is the work record as it
stood before it was rewritten to current truth on 2026-10-11, kept verbatim.
