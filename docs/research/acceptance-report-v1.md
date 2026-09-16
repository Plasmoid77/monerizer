# Acceptance v1 — report

Date: 2026-09-11 (night). Binary: commit `eef0639` (after the independent audit). Upstream: P2Pool 4.18, XMRig 6.26.0. Node: remote, from `nodes.txt` (picked by `node select`).

## Matrix

| System | systemd | How | Result |
|---|---|---|---|
| Arch Linux (owner's laptop, i7-8650U) | 261 | stand, reports 1–2 | all commands, TUI, polkit, node select, start-limit-hit |
| Debian 13 trixie (libvirt VM, UEFI, 2 vCPU, 6 GB) | 257 | stand install script + the same files | install → doctor → start → SYNCHRONIZED in ~4 min → health ok; polkit restart from the group; stop removes RuntimeDirectory; logs; `status --json`; TUI over ssh -t. Rolled back, VM removed |
| Ubuntu 22.04.5 (libvirt VM, UEFI, 2 vCPU, 6 GB) | 249 | 2026-09-16, binary `82cdb11`, the same stand script | install → doctor → start → SYNCHRONIZED → health ok (25 pass, 0 fail); stop removes RuntimeDirectory; logs; `status --json`. Peculiarity: polkit 0.105 without `rules.d` — the JS rule is not installed, control via sudo; the installer skips the rule when the directory is missing |
| Zeonux (Debian 13, 2×Xeon E5-2683 v4, 62 GB) | 257 | 2026-09-14…16, the owner's production host | 13.8 kH/s on 32 threads, hugepages 100 %; the uplink filters flows > ~15 KB to nodes — works through a SOCKS5 tunnel to the laptop (temporary); see §Zeonux |
| Debian 13 (libvirt VM, fresh cloud image) | 257 | 2026-09-16/17, `install.sh` v0.4.0–v0.4.2 | one-command install in 17 s → doctor → health ok in ~4 min; re-run idempotent; `--i2p` with a LAN-forwarded node: i2p peers within minutes; `--uninstall --purge` leaves nothing |

The systemd ≥ 249 boundary is confirmed (D7 closed).

## Spec §14 scenarios

| ID | Result | Where |
|---|---|---|
| A01 | pass — the services live without Moneroid and without an SSH session | stand, all night |
| A02 | pass — `q`/Ctrl-C/Esc never trigger control (code + unit test `TestQuitNeverControls`) | tui |
| A03 | pass — XMRig active without HTTP → API unavailable, metrics null | unit test `xmrig api down`; stand (stopped xmrig) |
| A04 | pass — old files without a proven link → `SOURCE_SESSION_UNKNOWN`/`NOT_CURRENT_SESSION` | unit tests |
| A05 | pass — 50 ms retry, source independence | unit tests p2pool/status |
| A06 | pass — a field of the wrong type → `FIELD_INVALID`, neighbours intact | unit tests xmrig/status |
| A07 | pass — 0 and null distinguishable in text/JSON/sparkline | unit test tui |
| A08 | pass — a new InvocationID → a gap in the chart | unit test tui |
| A09 | pass — start/stop order in one command | report 2 |
| A10 | pass — SIGINT during `restart`: message about the job handed to systemd, exit 130, final state read; the job completed (2026-09-16, laptop) |
| A11 | pass — `nobody`: read-only works, the refusal is visible (`PERMISSION_DENIED`), doctor warn JOURNAL_ACCESS | stand |
| A12 | pass — not-found, failed/start-limit-hit without parsing human-readable output | stand |
| A13 | pass — one valid JSON with all sources failing | stand (nobody), VM |
| A14 | pass — `--check` 0/1 for ok/degraded/stopped/unknown | stand |
| A15 | pass — 60×15 compact mode, non-TTY refusal; resize not tested separately | pty |
| A16 | pass — redirect/oversized/timeout in unit tests; no external addresses per CFG-04 |
| A17 | pass — 30 min TUI: RSS 15–18 MB, 15 threads, no growth | soak.log |
| A18 | pass — sanitize for pool/id/version/messages; the token only in the header | code, audit |
| A19 | pass — stdin=null, SIGTERM 0.4 s, start-limit after 5 crashes | reports 1–2, stand |
| A20 | not verified — no new upstream version appeared |
| A21 | pass — a wrong config → doctor fail with remedy; the binary path belongs to the unit | stand |
| A22 | pass — both inactive → `stopped`, enabled ≠ active | stand |
| A23 | pass — 0 shares and old rejects raise no errors | code, stand |
| A24 | pass — read-only configs, the group reads the API and does not write | report 2, VM |
| A25 | pass — a future mtime → `CLOCK_UNCERTAIN` | unit test |
| A26 | pass — RuntimeDirectory removed/created with the right permissions | report 2, VM |
| A27 | pass — ID mismatch / uptime mismatch → degraded | unit tests |
| A28 | pass — a drop-in `Requires=` for xmrig → `UNIT_DEPENDENCIES warn DEPENDENCIES_DIFFER`; after removing the drop-in — pass (2026-09-16, laptop) |
| A29 | pass — `node list`: timeouts, ranking, unusable ones at the bottom | stand |
| A30 | pass — `node select`: only three keys, owner/mode preserved, dry-run does not write | stand |

Non-functional: `status` collects in 20–40 ms on the stand; the 3 s deadline is enforced in code; TUI memory is stable (A17).

## Open

- A20 (upstream version change): as of 2026-09-17 the current releases are still P2Pool 4.18 and XMRig 6.26.0 — verify at the next upstream release (replace the binary + `moneroid restart`, bump the pins in `install.sh`).
- XMRig `SHA256SUMS.sig` was not verified by signature (no key imported); the P2Pool `sha256sums.txt.asc` signature was verified when pinning in `install.sh`.

## Zeonux (2026-09-14 … 17)

1. The first run by the instructions revealed `/usr/bin/nologin` → `/usr/sbin/nologin` (Debian) — fixed in the docs and the script.
2. From the Zeonux uplink `get_info` works while `get_block_headers_range` (> ~15 KB) hangs to most nodes; port and TLS make no difference; `deb.debian.org` worked at the time. Conclusion — selective filtering on the path (high confidence, not proven directly). The node probe now repeats that call.
3. `xmr.privacy.cash:18083` accepts TCP but not ZMQ (the ZMTP handshake stays silent) — the probe now does the handshake.
4. P2Pool takes the first DNS answer; Zeonux has an IPv6 route without a way out → `EBADF`. `node select` writes the address that answered (IPv4 first).
5. Workaround: `socks5 = 127.0.0.1:1080` in `p2pool.conf` + `ssh -D` to the laptop behind AmneziaVPN (transient `systemd-run` unit, the key on the laptop restricted with `restrict,port-forwarding`). P2P, RPC and ZMQ go through the proxy. After deleting the cache of the isolated chain P2Pool verified 4 457 blocks in 9 minutes and joined the real mini. `SIDECHAIN_BEHIND`/`SIDECHAIN_SYNC` were added.
6. Incident: `ip link set mtu 1000` disabled IPv6 on eno1 and cut my only path; recovery through the router and a reboot.
7. 2026-09-17: the same uplink also blocks GitHub release downloads, `deb.debian.org` (that day) and the I2P transports (i2pd reseeded through a proxy, 78 routers known, 0 % tunnel success in 13 minutes). The node is now forwarded with `ssh -L` to 127.0.0.1 (P2Pool dials private addresses directly), P2Pool p2p goes through `ssh -D`; i2pd is installed and configured but disabled until the uplink gets a VPN. Migrated from `monerizer` to `moneroid` with the installer and `/var/cache/moneroid`.
