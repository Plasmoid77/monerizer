# Moneroid v1 — implementation plan

Date: 2026-09-11. Revision 0.2. State on the morning of 2026-09-11: stages 0–5 done (laptop stand + Debian 13 VM, see §10 and the [acceptance report](research/acceptance-report-v1.md)); systemd 249, A10/A20/A28 were still open (closed on 2026-09-16 except A20).
Basis: [spec 0.3](spec.md) and the [source contracts](research/upstream-contracts.md).
This plan does not authorize installing services or starting mining; every stage begins after the owner's explicit confirmation.

## 0. Decisions of the spec review 0.2 → 0.3

The review on 2026-09-11 found no contradictions between spec 0.2 and the verified contracts. Clarifications made:

| Place | Change | Reason |
|---|---|---|
| CLI-07 | Ctrl-C in `logs --follow` and `tui` is a normal exit with code 0; 130 only for interrupted `status`, `doctor` and control | Otherwise a normal exit from the journal view would count as an error |
| SEC-08 | Explicit: XMRig under an unprivileged `User=` cannot apply its own MSR mod; a documented trade-off, not a defect | Without it the user expects "maximum hashrate" and gets `MSR: unknown` |
| DOC-05 (new) | Catalogue of doctor checks with codes, components and results | The doctor implementation and tests need a fixed list |
| §15 | Stage 0 "repository" and the list of owner decisions added | Without a module path, license and stand, stage 1 cannot begin |
| §4, §14 | Stand references: Debian 13 (systemd 257) as the "current" system; a second system with systemd 249 at the owner's discretion | Matches the available test host |
| §2, §5, §6.1, DOC-05, §14, §15 | Node selection added: `node list/select`, `NODE_RPC`, A29–A30, the `params_file`/`nodes_file` fields | Owner decision 2026-09-11 |
| §4, contracts §1 | P2Pool: current release 4.18 (2026-08-05), contracts verified against 4.17.1 — the stand compares with 4.18 | Release check 2026-09-11 |

Handoff v2 remains a historical document; all of its decisions that differ from the spec are considered withdrawn.

## 1. Owner decisions

Taken on 2026-09-11:

| # | Question | Decision |
|---|---|---|
| D1 | Module path and hosting | `github.com/Plasmoid77/moneroid` |
| D2 | License of the own code (INSTALL-05). Upstream GPL-3.0 is not inherited: XMRig/P2Pool code is neither linked nor copied | MIT |
| D3 | Stage-1 stand | The owner's working laptop (Arch Linux, current systemd). Users/units/binaries are installed on it only on the owner's explicit command at every step; mining is limited to the verification time. Acceptance on Debian 13 (stage 5) — a VM or a test VPS |
| D4 | Monero node for the stand | A remote node with RPC+ZMQ; at stage 1 the owner sets the address by hand, from stage 3 `node select` (§1.1) |
| D5 | Payout address for the stand | The owner's test address; replaced by a placeholder in fixtures |
| D6 | Default sidechain of the example | `mini` (spec INSTALL-03) |
| D7 | Second system of the acceptance matrix (systemd 249) | Decide at stage 5 by the Debian 13 results |

### 1.1. Automatic node selection

The owner chose automatic node selection "as in Gupax" while demanding maximum minimalism. The accepted form is spec §6.1 (NODE-01..05): `node list` measures the candidates from an editable `nodes.txt`, `node select` writes three keys into `p2pool.conf` and reminds about a restart. This is the only write Moneroid makes to a native config; there is no built-in list, no background switching and no selection on `start`. The example list is assembled at stage 3 from publicly documented community nodes with ZMQ, with source and date; keeping the list current after the release is the file owner's responsibility.

## 2. Stage 0 — repository

Result: an empty but buildable Go project with pinned versions and rules.

1. `git init`, branch `main`, `.gitignore` (binary, `dist/`, `*.token`, local captures before sanitizing).
2. `go mod init <D1>`, `go 1.27`, toolchain pin in `go.mod`; `CGO_ENABLED=0`.
3. Dependencies only: `charm.land/bubbletea/v2` (and what it pulls transitively), `github.com/BurntSushi/toml`. `golang.org/x/sys` is allowed for `CLOCK_MONOTONIC` and isatty if Bubble Tea already brings it transitively.
4. Directory skeleton from spec §15: `cmd/moneroid`, `internal/{config,status,systemd,xmrig,p2pool,doctor,tui}`, `examples`, `systemd`, `testdata`, `docs`.
5. `Makefile` with targets `build`, `test`, `vet`, `fmt-check`, `release` (see §8). No linter frameworks; `go vet` + `gofmt` mandatory.
6. `LICENSE` (D2); `README.md` is rewritten per INSTALL-04 at the end of stage 3; until then a status stub.
7. Documents: the spec and the plan stay in `docs/`; `docs/research/` — verified facts; `docs/schema/` — JSON schemas after stage 1.

Criterion: `make build test vet fmt-check` passes on the empty skeleton; `moneroid version` prints the version from `-ldflags`.

## 3. Stage 1 — stand and contracts

Goal: replace the assumptions of spec §16 with observations. Not a line of UI is written before this stage closes.

### 3.1. Stand preparation (administrator, by hand, per the future instructions)

1. Install the binaries of the current releases (as of 2026-09-11: P2Pool 4.18, XMRig 6.26.0) from the official GitHub Releases, verify checksum/signature. Location `/usr/local/bin`.
2. Create `moneroid-p2pool`, `moneroid-xmrig`, group `moneroid-observers`; the directories from spec §5.2 with SEC-02 permissions.
3. Install the drafts `examples/p2pool.conf`, `examples/xmrig.json`, `systemd/*.service` (profile SYS-02, SYS-03, SYS-11, SEC-08).
4. `daemon-reload`, `start` by hand through `systemctl`, watch the journal.

Every step is recorded in `docs/research/stand-report-1.md`: command, version, result. That report is the basis of `docs/install.md`.

### 3.2. What the stand confirms

| Check | Closes | Expectation from the spec |
|---|---|---|
| Params-file syntax: `key = value`, booleans as `1`, quotes or not | INSTALL-02 | per COMMAND_LINE.MD |
| P2Pool with `StandardInput=null` works and stops cleanly on SIGTERM in < 30 s | SYS-02, A19 | EOF does not stop the pool |
| XMRig with a read-only `--config` does not write the config and does not crash; `watch=false`, `autosave=false` (confirm the `autosave` key and its default) | INSTALL-03 | per the config documentation |
| Actual owner/group/mode of `RuntimeDirectory`, the `local/`,`network/`,`pool/` subdirectories and files | SEC-02, A24, A26 | 02750 / 0750 / 0640 |
| `RuntimeDirectoryMode=02750`: does systemd apply setgid | SEC-02 | if not, an explicit `chmod` is impossible; then the group comes from `Group=` and setgid is not needed |
| Write period of `local/p2p`, frequency of `local/stratum`, `network/stats`, `pool/stats` | DATA-07, contracts §6 | 60 s / on events |
| Presence/values of the fields of all four files and `/2/summary` on the reference versions | contracts §2–5 | the contract tables |
| `hugepages`, `msr` in `/2/summary` and `/2/backends` under an unprivileged user | MET-04, SEC-08 | MSR not applied |
| `ProtectSystem=strict`, `ProtectHome`, `NoNewPrivileges` do not break RandomX/JIT and cache writes | SEC-08 | without `MemoryDenyWriteExecute` |
| `systemctl show` without privileges: the SYS-06 property set is available | SYS-06, A11 | available |
| `journalctl -u` without privileges/in group `systemd-journal` | CLI-06, A11 | the refusal is visible |
| Behaviour with `Restart=on-failure` + `StartLimitBurst` on a wrong node address | A12, A21 | `start-limit-hit` in `Result` |
| Outgoing external P2P port (main 37889 / mini 37888 / nano per the docs) without UPnP | SEC-06 | outgoing only |

### 3.3. Fixtures

- `testdata/xmrig/6.26.0/summary.json` — a real reply, sanitized: `id`→`moneroid-xmrig`, pool→`127.0.0.1:3333`, no token.
- `testdata/p2pool/4.17.1/{local-p2p,local-stratum,network-stats,pool-stats}.json` — real, address and peers replaced by placeholders.
- `testdata/systemd/show-*.txt` — real `systemctl show` output for active/inactive/failed/not-found/masked.
- Synthetic: `*-broken-*.json` (truncated JSON, non-object, null fields, negative counts, a field of the wrong type, 1 MiB+1).
- Every real fixture gets a header comment in the neighbouring `README.md`: version, date, what was replaced.

### 3.4. Schemas

`docs/schema/status-v1.md` and `docs/schema/doctor-v1.md` are fixed: the full list of fields, types, nullability, units; an example JSON from the fixtures. After that `schema_version=1` is frozen: fields may be added, not changed or removed.

Stage criteria: A05–A08, A19, A24 as manual observations in the report; table 3.2 filled without "not verified" rows.

## 4. Stage 2 — read-only CLI

Result: `moneroid status [--json] [--check]`, `version`, `config path` work against the stand and are fully covered by unit tests on fixtures.

### 4.1. Packages and interfaces

```text
cmd/moneroid/main.go
    dispatcher per spec §6; exit codes CLI-07; --config before the subcommand.

internal/config
    type Config struct{ Services, P2Pool, XMRig, UI }
    func Load(path string) (Config, error)      // CFG-01..07; error → exit 2
    func Default() Config                        // optional fields only

internal/status                                  // model + collector + health
    type Snapshot, ServiceState, Source, Issue, Health
    type Clock interface{ Now() time.Time; Monotonic() time.Duration }
    type Collector struct{ Systemd, XMRig, P2Pool; Clock; Deadline }
    func (c *Collector) Collect(ctx) Snapshot     // DATA-01, parallel, bounded
    func Evaluate(s *Snapshot)                    // H-02, H-03, H-04; fills Health and Issues

internal/systemd
    type Runner interface{ Run(ctx, argv []string, limit int) (stdout, stderr []byte, code int, err error) }
    func Show(ctx, r Runner, units []string) (map[string]Props, error)   // SYS-06
    func Control(ctx, r Runner, verb string, units []string) Result       // SYS-07..09 (stage 3)
    func JournalArgs(units []string, lines int, follow bool) []string    // CLI-06 (stage 3)

internal/xmrig
    type Client struct{ HTTP interface{ Do(*http.Request) (*http.Response, error) }; URL; TokenFile }
    func (c *Client) Summary(ctx) (Summary, SourceMeta)                  // GET /2/summary, 1 MiB, no redirect/proxy

internal/p2pool
    type Reader struct{ FS fs.FS; Dir string }                          // fs.Stat for mtime
    func (r *Reader) Read(ctx) Files                                     // 4 files, DATA-02..04
```

Rules: `internal/status` does not import the upstream JSON adapters wholesale — the adapters return already normalized nullable structures. No package except `internal/tui` imports Bubble Tea.

### 4.2. Order of work

1. `internal/config` + tests (valid/invalid TOML, all CFG rules).
2. `internal/systemd.Show` + a `key=value` output parser + fixtures.
3. `internal/xmrig` + tests on `httptest.Server`: normal reply, 401, redirect, slow reply, oversized body, ID mismatch.
4. `internal/p2pool` + tests on `fstest.MapFS`: fresh, stale, future mtime, corrupted, missing, oversized, 50 ms retry.
5. `internal/status`: model, collector with deadline, health rules — table tests per H-02/H-03.
6. Text output of `status` and JSON per `docs/schema/status-v1.md`; golden tests.
7. Stand check: A03 (XMRig without HTTP), A04 (old files), A22, A25 (clock shift).

Criteria: A03–A08, A12–A14, A16–A18, A22–A23, A25; `go test -race ./...` without network or systemctl access.

## 5. Stage 3 — operations

Result: `start/stop/restart/logs/doctor`, the shipped units and the full installation instructions.

1. `systemd.Control`: one operation per process (SYS-09), 90 s deadline (SYS-08), a control `Show` up to 3 s, result output CLI-05; codes 1/3/4.
2. `logs`: `journalctl --no-pager -u U [-u U2] -n N [-f] -o short-iso` or an equivalent without colour; Ctrl-C → 0.
3. `internal/doctor`: checks per spec DOC-05; JSON per `docs/schema/doctor-v1.md`; exit 1 on fail.
3a. `internal/node`: `nodes.txt` parser, parallel probe (`get_info` + TCP ZMQ), ranking, replacing three keys in the params-file with an atomic write; `examples/nodes.txt`. Tests: `httptest` for RPC, `net.Listen` for the ZMQ port, golden tests of the key replacement on a config with comments and without the keys.
4. Final `systemd/moneroid-p2pool.service`, `systemd/moneroid-xmrig.service`, `examples/*` — the versions verified at stage 1 plus corrections from the results.
5. The optional polkit rule `examples/polkit/50-moneroid.rules` strictly per SEC-03; a separate instruction step.
6. Documentation: `README.md` (INSTALL-04), `docs/install.md`, `docs/troubleshooting.md`, `docs/updating.md`. The instructions must require substituting the address and node before start (INSTALL-01).

Criteria: A01, A09–A12, A19, A21, A24, A28–A30 on the stand; report `docs/research/stand-report-2.md`.

## 6. Stage 4 — TUI

Result: `moneroid tui` over the same `Collector` and `Control`.

1. Bubble Tea v2 model: states `dashboard | services-menu | confirm | help | small-terminal`; a timer per `refresh_ms`; skip the tick while a collection is running (DATA-01, A17).
2. Hashrate history: a ring buffer of 1200 points / 10 minutes (DATA-09), a gap on InvocationID/uptime reset (DATA-08).
3. Control from the menu: confirmation with unit names, Cancel by default (UI-03); one in-flight request.
4. Journal: leave the alt screen, child `journalctl -f`; **mandatory** — the child in its own foreground process group or the parent temporarily ignoring SIGINT, otherwise Ctrl-C in the viewer ends the dashboard. Verify on the stand.
5. `NO_COLOR`, ASCII fallback, 80×24 and the compact mode (UI-05, UI-06); filtering control characters from upstream strings (A18).
6. Terminal restoration on panic/signals; `q` and Ctrl-C never trigger control.

Criteria: A02, A07–A11, A15, A17–A18. TUI tests: unit tests of the model (Update on synthetic messages) without a real terminal; a manual checklist of terminal sizes in the report.

## 7. Stage 5 — release acceptance

1. The full matrix A01–A28 on Debian 13; A20 separately (replacing the binary with the next upstream version, if released) and A26–A28.
2. The second system per D7.
3. A 30-minute TUI observation: goroutines and RSS stable (non-functional criteria §14).
4. Report `docs/research/acceptance-report-v1.md`: OS, systemd, upstream versions, command, result per A item.
5. Release: `moneroid_<ver>_linux_amd64` + `SHA256SUMS` + `examples/` + `systemd/` + `docs/`. Publication is the owner's decision (GitHub Releases).

## 8. Working rules during implementation

- One writer: edits only in this session; subagents do research, audits and test runs.
- Mining, service installation and system changes only on the stand (D3, the working laptop) and only on the owner's explicit command for each step: creating users, installing units, `daemon-reload`, `start`. Before stage 1 the list of everything to be created on the system and the full rollback command are recorded; after the stage closes the stand is removed by the same list.
- No tokens, payout addresses or RPC credentials in the repository, fixtures, reports or subagent prompts.
- Every commit builds with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=<ver>"` and passes `go vet`, `gofmt -l`, `go test -race`.
- Maximum minimalism is the owner's requirement. A new package or dependency only with the reason recorded in the commit; the target size is about 3–4k lines of Go without tests. Growth beyond that signals over-engineering, not the norm. Before adding any feature — the question "can a standard systemd/journalctl/upstream command do this?"; if yes, the feature is not added.
- The order is fixed: contracts → CLI → operations → TUI. Dashboard styling does not start before stage 2 closes.
- The handoff §47 restrictions (do not fork/vendor upstream, do not implement Stratum/RPC/updater/DB/web) apply as they are.

## 9. Risks

| Risk | Impact | Action |
|---|---|---|
| The remote node is unreachable/not synced | Stage 1 does not close, fixtures are synthetic | The owner picks the node before stage 1; candidates are checked by hand |
| The stand on the working machine leaves traces (users, units, directories) | Pollution of the owner's system | The list of created objects and the rollback are recorded before installing |
| Arch on the stand ≠ Debian 13 of the target (systemd versions, paths) | Discrepancies surface late | Stage-5 acceptance on Debian 13; the unit profile uses nothing Arch-specific |
| `RuntimeDirectoryMode` does not accept setgid | SEC-02 permissions change | The check in table 3.2; the fallback is described there |
| Bubble Tea v2 API differs from the v1 examples | Stage 4 delay | Only official v2 examples and godoc; TUI last |
| Upstream versions update during the work | Fixtures go stale | Versions in the fixture names; A20 at stage 5 |
| Ctrl-C in the journal view kills the TUI | UI-04 not met | An explicit item 4.4 |
| The node list in `nodes.txt` goes stale | `node select` finds no candidates | The file is editable with the check date inside; exit 1 with a clear message, not a crash |

## 10. Progress (night of 2026-09-11)

The owner changed the order to incremental: first the bare pair as a user (report 1), then the pair under systemd (report 2), then code one command per commit.

| Commit | What |
|---|---|
| `873ba41` | documents, units, examples, fixtures, `status/--json/--check`, `version`, `config path` |
| `06755c3` | `start/stop/restart`, `logs`, polkit rule |
| `32e77c5` | `doctor` (DOC-05) |
| `256c788` | `node list/select`, `NODE_RPC` |
| `93339ed` | `tui` (Bubble Tea v2) |
| `daeab5b` | README, install, troubleshooting, updating |
| `18affd6` | reset-failed before start/restart (start-limit-hit) |
| `eef0639` | fixes after the independent audit (deep-auditor, 22 findings) |

Verified on the stand by hand: A03, A04 (partially), A09, A11, A12 (not-found), A13, A14, A22, A24, A26, A29, A30; TUI — menu, Cancel by default, restart through polkit, journal with return on Ctrl-C, compact mode, refusal without a TTY. Not verified then: A16 (slow HTTP — unit test only), A19 start-limit, A20, A25 (unit test only), A28, the 30-minute memory observation, Debian 13 — all closed later except A20 (see the acceptance report).

Later history (2026-09-16/17): payouts, htop-style TUI with Monero colours, the rename to Moneroid, `install.sh` (one-command install, I2P mode, download cache), reproducible builds — see `CHANGELOG.md`.
