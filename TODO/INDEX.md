# Work index

Total: 67
Open: 15
In progress: 0
Blocked: 0
Done: 52

| priority | open | in progress | blocked | done | total |
| --- | ---: | ---: | ---: | ---: | ---: |
| P0 | 0 | 0 | 0 | 25 | 25 |
| P1 | 15 | 0 | 0 | 26 | 41 |
| P2 | 0 | 0 | 0 | 1 | 1 |
| Total | 15 | 0 | 0 | 52 | 67 |

| id | priority | effort | status | title |
| --- | --- | --- | --- | --- |
| FOUND-01 | P0 | L | DONE | Repository workspace and policy skeleton |
| FOUND-02 | P0 | L | DONE | Reproducible Rust dependency and action pins |
| FOUND-03 | P0 | L | DONE | Deterministic protocol fixture suite |
| FOUND-04 | P1 | M | DONE | Third-party licence and redistribution register |
| FOUND-05 | P1 | L | DONE | The session host, provisioned by something rather than by memory |
| SCHEMA-01 | P0 | L | DONE | Versioned identity profile schema |
| SCHEMA-02 | P0 | L | DONE | Raw evidence and run manifest schema |
| SCHEMA-03 | P0 | L | DONE | Connector agreement and conflict model |
| SCHEMA-04 | P0 | M | DONE | Variability and repeated-sampling model |
| OBS-01 | P0 | L | DONE | Isolated Rust loopback observation lab |
| OBS-02 | P0 | L | DONE | HTTP tracker observer |
| OBS-03 | P0 | L | DONE | UDP tracker observer |
| OBS-04 | P0 | L | DONE | Peer-wire handshake observer |
| OBS-05 | P0 | L | DONE | BEP 10 and early-message observer |
| OBS-06 | P1 | M | DONE | Adjacent protocol observer suite |
| OBS-07 | P1 | M | DONE | Known-client positive controls |
| OBS-08 | P0 | M | DONE | Synthetic torrent for the observation lab |
| OBS-09 | P0 | M | DONE | Raw evidence journal and bundle writer |
| OBS-10 | P1 | M | DONE | Cross-platform normalized-event equality |
| OBS-11 | P1 | L | DONE | Message stream encryption, DHT and web-seed observers |
| ACQ-01 | P0 | L | DONE | Acquisition route schema |
| ACQ-02 | P0 | L | DONE | Latest stable release resolver |
| ACQ-03 | P0 | L | DONE | Same-version multi-route verifier |
| ACQ-04 | P0 | L | DONE | Disposable-host execution boundary |
| ACQ-05 | P1 | M | DONE | Artifact cache and authenticity evidence |
| ACQ-06 | P0 | L | DONE | Rootless, portable client installation |
| CLIENT-01 | P1 | L | OPEN | qBittorrent capture adapter |
| CLIENT-02 | P1 | L | DONE | qBittorrent Enhanced capture adapter |
| CLIENT-03 | P1 | L | OPEN | uTorrent capture adapter |
| CLIENT-04 | P1 | L | OPEN | BitComet capture adapter |
| CLIENT-05 | P1 | L | OPEN | aria2 capture adapter and connector |
| CLIENT-06 | P1 | L | OPEN | Transmission capture adapter |
| CLIENT-07 | P1 | L | OPEN | Deluge capture adapter |
| CLIENT-08 | P1 | L | OPEN | BitTorrent capture adapter |
| CLIENT-09 | P1 | L | DONE | BiglyBT capture adapter |
| CLIENT-10 | P1 | L | OPEN | Tixati capture adapter |
| CLIENT-11 | P1 | L | OPEN | KTorrent capture adapter |
| CLIENT-12 | P1 | L | OPEN | Free Download Manager capture adapter |
| CLIENT-13 | P1 | L | OPEN | Zona capture adapter |
| CLIENT-14 | P1 | L | DONE | aria2-next, acquired from its releases and driven over RPC |
| ENGINE-01 | P1 | L | OPEN | libtorrent engine matrix |
| ENGINE-02 | P1 | L | DONE | anacrolix/torrent engine matrix |
| ENGINE-03 | P1 | L | DONE | rqbit engine matrix |
| CORPUS-01 | P0 | L | DONE | Append-only canonical store |
| CORPUS-02 | P0 | L | DONE | Semantic corpus validator |
| CORPUS-03 | P0 | L | DONE | Deterministic indexes and latest views |
| CORPUS-04 | P1 | M | DONE | Supersession and correction records |
| LIB-01 | P1 | L | DONE | Rust consumer library |
| LIB-02 | P1 | L | DONE | bit-cli integration adapter |
| PUB-01 | P0 | L | DONE | Deterministic release assembler |
| PUB-02 | P0 | L | DONE | Protected append-only data branch publisher |
| PUB-03 | P1 | L | DONE | Multi-format GitHub release publisher |
| PUB-04 | P1 | M | DONE | Stable raw and index access paths |
| PUB-05 | P1 | M | DONE | SQLite rendering of the published records |
| CI-01 | P0 | L | DONE | Complete cross-platform quality gate |
| CI-02 | P1 | L | DONE | Stable-release staleness monitor |
| CI-03 | P1 | L | DONE | Trusted capture runner matrix |
| CI-04 | P1 | L | OPEN | Build provenance and supply-chain hardening |
| CI-05 | P1 | S | DONE | Acceptance commands that cannot pass over nothing |
| CI-06 | P1 | L | DONE | The first dispatched capture run |
| CI-07 | P1 | L | OPEN | PowerShell halves for the declared gate rows |
| CI-08 | P1 | L | OPEN | Runner-default drift, swept rather than waited for |
| CI-09 | P1 | L | DONE | The capture-to-publisher path, end to end |
| CI-10 | P1 | XL | DONE | Every checking script in Go, and CI across multiple runners |
| CI-11 | P1 | XL | DONE | The capture path in Go, so a client capture runs on Windows |
| DOC-01 | P1 | M | DONE | Public data and library documentation |
| DOC-02 | P2 | M | DONE | Contributor capture-run handbook |

## Ordering argument

Schemas and replayable fixtures come first because every observer, adapter,
validator, and publisher consumes their contracts. Acquisition identity and
isolation follow before any proprietary client runs. The observer surfaces
then establish the shared measurement layer. Three open clients with useful
automation seams prove the vertical path before the remaining product
breadth. Corpus, publishing, and CI close the automation loop; documentation
and refinements follow operating behavior.

Identifiers are allocated in the order entries were authored, so `OBS-08`
through `OBS-10` came out of splitting `OBS-01` and are not the last observer
work to be done. This table is the list; the order is in
[`PROGRESS.md`](PROGRESS.md) and nowhere else.
