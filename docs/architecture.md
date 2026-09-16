# How Moneroid is built

One static Go binary (~3.5k lines without tests), three external modules: `github.com/BurntSushi/toml`, `charm.land/bubbletea/v2` (only for `tui`) and `github.com/charmbracelet/colorprofile` (colour downsampling). No daemon, no state on disk and no network beyond loopback — except node probes on request.

## Data flow

```text
                     ┌──────────────── systemctl show / start / stop / restart / reset-failed
                     │                 journalctl -u …
moneroid ───────────┼──────────────── GET http://127.0.0.1:18088/2/summary        (XMRig HTTP API)
  status/tui/doctor  │
                     └──────────────── read /run/moneroid-p2pool-api/{local/p2p,local/stratum,network/stats,pool/stats}
                                        (P2Pool Data API, files written by P2Pool)

XMRig ──Stratum 127.0.0.1:3333──▶ P2Pool ──RPC+ZMQ──▶ Monero node     (Moneroid takes no part in this chain)
```

One `Collector.Collect` call (`internal/status/collect.go`) polls the three sources **in parallel and independently** with a common 3 s deadline (1 s per HTTP/systemctl call). A failing source does not affect the others; there is always a result — problems land in `sources` and `health.issues`, not in an error.

## Packages

| Package | Responsible for | Not responsible for |
|---|---|---|
| `cmd/moneroid` | argument parsing, text/JSON output, exit codes (0/1/2/3/4/130) | data logic |
| `internal/config` | `moneroid.toml`: strict schema, loopback-only `api_url`, absolute paths | reading the P2Pool/XMRig configs |
| `internal/systemd` | `systemctl show` → properties; `systemctl VERB -- UNIT…`; `journalctl` arguments | parsing `systemctl status`, PID management |
| `internal/xmrig` | `GET /2/summary` → normalized nullable fields | POST/PUT, the miner config |
| `internal/p2pool` | reading the four Data API files (regular files, ≤ 1 MiB, 50 ms retry) | HTTP server, consensus |
| `internal/jsonx` | tolerant JSON: unknown fields are ignored, a wrong type breaks only its own field | — |
| `internal/status` | the `Snapshot` model, collection, freshness, binding data to the process session, health rules | keeping history |
| `internal/doctor` | 31 read-only checks over the same `Snapshot` + journal/token/clock/node | auto-repair |
| `internal/node` | node probes (`get_info`, `get_block_headers_range`, ZMTP handshake, through SOCKS5 when configured, private addresses direct), replacing three keys in `p2pool.conf` | picking a node on the fly |
| `internal/payouts` | the "got a payout of" lines from the journal via `journalctl -g` (the only log reading) | balance, wallet |
| `internal/tui` | the Bubble Tea dashboard over the same `Collector`, full terminal height (`htop` style); control menu; `journalctl -f` via `tea.ExecProcess`; payouts summary once a minute; explicit erase on exit | collecting data of its own |
| `internal/ansi` | a few SGR sequences of the Monero palette (orange/white, red for problems) and `Bar`; colour downsampling and `NO_COLOR` are done by `colorprofile` (in the TUI by Bubble Tea itself) | colour as the only carrier of meaning |

Interfaces exist only at the boundaries with the outside world: `systemd.Runner` (command execution), `http.Client`, the file system, clocks (`Now`, `Monotonic`). That is why all of `internal/status` is tested on fixtures without systemctl and network (`testdata/`).

## Key invariants

1. **The dashboard never owns mining.** `q`, Ctrl-C, Esc or a TUI crash never call stop. The services live in systemd independently of Moneroid (`SYS-01`).
2. **One owner per parameter.** Payout address, node, sidechain — `p2pool.conf`; threads, API — `xmrig.json`; binary paths — the unit files; what to observe — `moneroid.toml`. Moneroid writes to a foreign config in exactly one place: `node select` changes `host/rpc-port/zmq-port`.
3. **`null` ≠ `0`.** Unknown is `null`, a measured zero is `0`; visible in text, JSON and the sparkline.
4. **Freshness and session.** A file has an age (mtime) and a proof of belonging to the current process (`RuntimeDirectory` + uptime comparison); HTTP has a matching `api.id` and uptime versus `ExecMainStartTimestampMonotonic`. Data from a previous run never confirms health.
5. **Consequences are not presented as causes.** "API unavailable" ≠ "process crashed", "service active" ≠ "node synchronized"; `sync_state` is always `unknown`.
6. **Health by rules H-02/H-03** (`internal/status/health.go`): unit not-found/failed → `degraded`; both inactive → `stopped`; activating → `starting`; one of two → `degraded`; both active → `ok` only with a fresh `local/p2p` of the current process, XMRig connected with hashrate > 0, peers > 0, sidechain not behind the peers, ZMQ ≤ 600 s.

The normative document is the [spec](spec.md); where each field and unit comes from — the [source contracts](research/upstream-contracts.md); what was verified and how — the [acceptance report](research/acceptance-report-v1.md).

## Output schemas

`status --json`, `doctor --json` and `payouts --json` are versioned by `schema_version` (currently 1): fields are only added. Description — [docs/schema](schema/).

## Build and check

```sh
make check      # gofmt, go vet, go test -race, static linux/amd64 build
make build      # ./moneroid with the version from git describe
```

Tests never touch the network or call systemctl; the fixtures in `testdata/` are real P2Pool 4.18 and XMRig 6.26.0 replies with addresses replaced.
