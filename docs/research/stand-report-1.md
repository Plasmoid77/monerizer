# Stand report 1 — the P2Pool + XMRig pair as a user

Date: 2026-09-11. Host: the owner's laptop, Arch Linux, systemd 261, i7-8650U (4C/8T, L3 8 MiB), 15 GiB RAM, no hugepages configured, no MSR applied. Run as a regular user in a scratchpad directory, without systemd and without root. Goal — confirm that the pair works and obtain real fixtures.

## 1. Binaries

| Component | Asset | Verification |
|---|---|---|
| P2Pool v4.18 | `p2pool-v4.18-linux-x64.tar.gz` | `sha256sums.txt.asc` — signature Good, SChernykh's key `1FCA AB4D 3DC3 310D 16CB D508 C47F 82B5 4DA8 7ADF` (imported from monero-project/gitian.sigs); SHA256 `893691…8a415` matched |
| XMRig 6.26.0 | `xmrig-6.26.0-linux-static-x64.tar.gz` | `SHA256SUMS`: `fc6f8a…c9e5` matched; `.sig` not verified (no key imported) |

## 2. Node choice

The clearnet node list was taken from monero.fail (`nodes.json`, 121 http nodes). Probe: JSON-RPC `get_info` + TCP connect to 18083. Candidates with an open 18083 and `synchronized=true` — 7.

| Node | RPC | ZMQ | Result in P2Pool |
|---|---|---|---|
| monerodice.pro | 18089, ping 83 ms | 18083 | ZMQ drops every second ("ZMQ is not running, restarting it", 69 drops in 2.5 min) — unusable |
| xmr.support | 18081, ping 200 ms (P2Pool warns "too high") | 18083 | ZMQ stable (0 drops in 6 min), SideChain SYNCHRONIZED after ~3.5 min |

Conclusion for NODE-02: an open ZMQ TCP port does not prove a working ZMQ; the final check is the P2Pool journal only. Ranking by latency stays, but the documentation must say that the "best" node may turn out unusable and how to see that.

## 3. P2Pool

- `--params-file` with 12 parameters (`host`, `rpc-port`, `zmq-port`, `wallet`, `mini = 1`, `stratum`, `data-dir`, `data-api`, `local-api = 1`, `no-upnp = 1`, `no-color = 1`, `loglevel = 3`) accepted without complaints; format `key = value`, booleans as `1`.
- stdin = `/dev/null`: works. Log: "ConsoleCommands tty or named pipe is not available" — P2Pool opens a **TCP console on a random localhost port** and writes the port and cookie to `local/console`. See §6.
- SIGTERM → full stop in 0.41 s (two runs). `TimeoutStopSec=30s` has margin.
- The Data API creates `local/{p2p,stratum,merge_mining,console}`, `network/stats`, `pool/stats`, `stats_mod`. Files without extension.
- `local/p2p` is updated every ~60 s; `local/stratum` on events; `pool/stats` and `stats_mod` on new sidechain blocks; `network/stats` on a new mainchain block.
- `local/stratum` contains **`wallet`** (the payout address in clear text) and `workers` with `ip:port,…`. Contracts §4 had not accounted for this: the address must be excluded from output and fixtures (SEC-07).
- `p2pool.log` (a duplicate of the console) plus cache/peers are written into `data-dir`. For systemd: `no-log-file = 1`.
- RSS ≈ 2.8 GB (RandomX dataset), 10 outgoing P2P connections, `zmq_last_active` 0–8 s.

## 4. XMRig

- `--config` with `autosave=false`, `watch=false`, `colors=false`, `http` on 127.0.0.1:18088 restricted, `api.id=moneroid-xmrig`, `max-threads-hint=50` (4 threads): connected to 127.0.0.1:3333, first accepted share in < 75 s, ~880 H/s.
- **With stdout redirected to a regular file XMRig writes nothing** (0 bytes even with `--dry-run`). Under systemd (stdout is a journald stream socket) the log flows normally: verified in report 2. `"syslog": true` is not needed (duplicates).
- `/2/summary`: `hashrate.total[2]` = null for the first 15 minutes (confirms nullable); `hugepages = [0, 1172]`; `restricted=true`. `/2/backends[0].msr = false`.
- SIGTERM → stop in 0.41 s.
- Hashrate estimates differ: XMRig 10s ≈ 880 H/s, P2Pool `hashrate_15m` = 432 H/s (from shares) — an illustration of MET-01.
- CPU temperature with 4 threads ≈ 67 °C.

## 5. Fixtures

Saved in `testdata/` (see `testdata/README.md`): payout address, peer IPs and the console cookie replaced.

## 6. Findings that need a decision

1. **Console cookie in the Data API.** With stdin not a tty P2Pool publishes in `local/console` the port and cookie of a TCP console that exposes the P2Pool console commands (presumably including `exit`). An observer group with read access to the API directory gains control of the process. Options: (a) merge observers and operators into one group and document it; (b) check whether any parameter disables the TCP console (`--help` has none); (c) StandardInput through a FIFO — extra complexity. Proposal: (a), minimal.
2. **Duplicate log file**: `no-log-file = 1` in the shipped example.
3. **`local/stratum.wallet`**: add to the contracts as a field that is never printed.

## 7. Not verified at this step

`RuntimeDirectory` permissions, the unit profile (`ProtectSystem=strict` and RandomX), `Restart=on-failure`/start-limit, running as separate users, `systemctl show` without privileges — all of that belongs to the systemd run (the next step).
