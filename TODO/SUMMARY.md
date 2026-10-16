# Work summary

Every count here is derived from the rows in [`INDEX.md`](INDEX.md) and checked
against them by `check-project`. The `prefix` column is what makes that
possible: it says which identifiers the row counts, so the check needs no
mapping of its own and the two twins cannot hold different ones.

| category | prefix | open | in progress | blocked | done | total |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| Foundation | `FOUND` | 0 | 0 | 0 | 5 | 5 |
| Schema | `SCHEMA` | 0 | 0 | 0 | 4 | 4 |
| Observer | `OBS` | 0 | 0 | 0 | 11 | 11 |
| Acquisition | `ACQ` | 0 | 0 | 0 | 6 | 6 |
| Clients | `CLIENT` | 13 | 0 | 0 | 1 | 14 |
| Engines | `ENGINE` | 3 | 0 | 0 | 0 | 3 |
| Corpus | `CORPUS` | 0 | 0 | 0 | 4 | 4 |
| Library | `LIB` | 0 | 0 | 0 | 2 | 2 |
| Publishing | `PUB` | 0 | 0 | 0 | 5 | 5 |
| CI | `CI` | 3 | 0 | 0 | 8 | 11 |
| Documentation | `DOC` | 0 | 0 | 0 | 2 | 2 |
| Total | | 19 | 0 | 0 | 48 | 67 |

Effort inventory: 1 S, 13 M, 51 L, 2 XL. `check-project` derives that line from
the index and refuses it when they disagree. `CI-10` and `CI-11` are the XL entries; the
observer-lab entry was one too and was split on 2026-09-04, because its
acceptance named a client fixture and a Windows run that this repository cannot
produce yet.
[`observer.md`](observer.md) carries what moved where.
