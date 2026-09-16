# Fixtures

Real captures from the 2026-09-11 stand (Arch Linux, run as a user without systemd), see `docs/research/stand-report-1.md`.

| Directory | Source | What was replaced |
|---|---|---|
| `p2pool/4.18/` | P2Pool v4.18, Data API (`--data-api` + `--local-api`), mini sidechain, node xmr.support | payout address → `4AAA…`; peer IPs → `192.0.2.1`; `local/console`: cookie → `REDACTED`, port → 0 |
| `xmrig/6.26.0/` | XMRig 6.26.0, `GET /2/summary`, `GET /2/backends`, 4 threads, no hugepages/MSR | nothing (no token was set, pool = 127.0.0.1:3333) |

Upstream Data API file names have no extension (`local/p2p`); here they are `local-p2p.json`. Synthetic broken variants live next to them with a `-broken-*` suffix.
