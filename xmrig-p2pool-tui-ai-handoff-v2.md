# AI handoff: headless CLI/TUI wrapper for Monero mining with XMRig + P2Pool

**Research snapshot:** 2026-09-10  
**Working project name:** `xmr-stack` / `gupax-tui` (name is intentionally not finalized)  
**Primary target:** Linux headless servers accessed over SSH  
**Primary goal:** provide a small terminal-first **glue/orchestration wrapper** for the core Gupax-style workflow: configure integration, run, monitor, and control **P2Pool + XMRig** without a graphical environment. It is not a new miner and should not grow into one.  
**Primary architectural constraint:** the wrapper itself should require **as close to zero ongoing maintenance as practical** after a stable release.

---

## 0. Read this first — the most important product decision

This should **not** become “Gupax rewritten for the terminal”. It must be based on the **official upstream Monero + P2Pool + XMRig CLI workflow**, with no custom mining integration invented by this project.

It should be a **thin orchestration and monitoring layer around two independent upstream programs**:

1. **P2Pool**
2. **XMRig**

Optionally, it may know how to connect P2Pool to an already-running local or remote Monero node, but **managing `monerod` is not required for the first version**.

The desired long-term maintenance model is:

> The wrapper stays mostly unchanged. XMRig and P2Pool are independently replaceable/updatable binaries. The wrapper talks to them only through stable public interfaces and systemd.

Do **not** fork either XMRig or P2Pool.  
Do **not** vendor their source code into this project.  
Prefer not to bundle their binaries.  
Do **not** build a custom updater unless there is an extremely strong reason.

The user explicitly wants something that can ideally be written once and then left alone.

---


# 0A. Canonical integration source of truth

This project **must be based on the official upstream way of using XMRig together with P2Pool**.

The wrapper must **not invent its own mining topology, protocol, configuration model, or XMRig↔P2Pool integration layer**.

Use these as the canonical references:

1. **Official Monero documentation — XMRig + P2Pool guide**
   - https://docs.getmonero.org/interacting/mining/guides/p2pool/xmrig-p2pool/
   - Defines the expected topology and normal way to connect XMRig to P2Pool.

2. **Official P2Pool command-line documentation**
   - https://github.com/SChernykh/p2pool/blob/master/docs/COMMAND_LINE.MD
   - Source of truth for P2Pool arguments, params-file behavior, Data API, Local API, sidechain selection, ports, wallet configuration, node RPC/ZMQ configuration, etc.

3. **Official XMRig documentation**
   - https://xmrig.com/docs/miner
   - https://xmrig.com/docs/miner/api
   - Source of truth for XMRig configuration and monitoring through its HTTP API.

The canonical topology is:

```text
Monero node
    ↑ RPC + ZMQ
    │
  P2Pool
    ↑ Stratum
    │
  XMRig
```

The wrapper sits **around** this topology, not inside it:

```text
              canonical upstream setup
                       │
                       ▼
          Monero node ← P2Pool ← XMRig

                our wrapper
                     │
       ┌─────────────┼─────────────┐
       ▼             ▼             ▼
   lifecycle       status         TUI
   management    / health      monitoring
```

The most important implementation rule is:

> **Implement and preserve the official Monero XMRig + P2Pool CLI setup as-is. Do not invent another integration layer. The project only adds lifecycle management, observability, diagnostics, and terminal UX around the canonical upstream setup.**

This is a hard architectural constraint.

## What the official guide solves

The official guide is considered a **complete solution for the mining relationship itself**:

- P2Pool connects to a compatible Monero node using the required RPC/ZMQ interfaces.
- XMRig connects to P2Pool over Stratum.
- P2Pool handles P2Pool protocol, shares, payouts, sidechain operation, and network participation.
- XMRig handles RandomX mining and CPU optimization.

Our project does **not** improve or replace any of that.

## What our project adds

The official guides do not primarily solve headless operational UX.

Our wrapper adds only:

```text
start
stop
restart
status
status --json
logs
doctor
optional live TUI
```

plus persistent service lifetime through systemd.

Therefore the project should be understood as:

> **a headless operations wrapper around the official P2Pool + XMRig CLI setup**

not as:

> a new mining stack

and not as:

> a terminal rewrite of Gupax

## Configuration consequence

The wrapper should preserve upstream-native configuration as much as possible.

Preferred model:

```text
p2pool.conf  ───────────────> P2Pool
xmrig.json   ───────────────> XMRig

wrapper.toml
    └─ only contains paths, service names,
       API locations and UI/integration metadata
```

The wrapper should **not** mirror every P2Pool/XMRig setting into its own schema.

The user should always be able to bypass the wrapper and run the upstream programs directly using the same config files.

For example, conceptually:

```bash
p2pool --params-file /etc/xmr-stack/p2pool.conf
xmrig --config /etc/xmr-stack/xmrig.json
```

If those commands work without our wrapper, the architecture is healthy.

This property is deliberate and important for the “write once, forget” maintenance goal.

## Anti-overengineering rule

Before adding any wrapper-side feature, ask:

> Is this already handled by XMRig, P2Pool, Monero, systemd, or their official config/API?

If yes, use the upstream functionality instead of recreating it.

Examples:

- mining algorithm logic → XMRig
- CPU tuning logic → XMRig
- Stratum → XMRig/P2Pool
- P2Pool consensus → P2Pool
- sidechain logic → P2Pool
- payout accounting → P2Pool
- Monero network logic → Monero/P2Pool
- service supervision → systemd
- logs → journald
- XMRig runtime metrics → XMRig HTTP API
- P2Pool runtime metrics → P2Pool Data API / Local API

Our code should remain glue.


# 1. User intent

The user wants a terminal-native alternative to Gupax for a server without a desktop environment.

The essential workflow is:

```text
SSH into server
    ↓
start / stop / restart mining
    ↓
see whether P2Pool and XMRig are healthy
    ↓
see current hashrate / shares / P2Pool state
    ↓
optionally open a continuously updating TUI
    ↓
quit TUI without stopping mining
```

The program should feel more like:

```text
htop
systemctl
docker compose
```

than like a desktop application translated into a terminal.

The user does **not** need a huge mining-management platform, a browser dashboard, a database, a fleet manager, or a wallet manager.

## 1.1 Conceptual classification: this is a glue script, even if implemented as a compiled binary

The project should be mentally modeled as a **smart glue script** rather than a standalone mining application.

Its responsibility is only:

```text
wrapper
  ├─ start/stop/restart P2Pool
  ├─ start/stop/restart XMRig
  ├─ query systemd status
  ├─ read XMRig structured status
  ├─ read P2Pool structured status
  ├─ aggregate both into one small status model
  └─ expose that model through CLI and optional TUI
```

Everything mining-specific remains upstream:

```text
P2Pool
  ├─ P2Pool protocol/consensus
  ├─ shares
  ├─ payout logic
  ├─ Stratum server
  └─ Monero-node interaction

XMRig
  ├─ RandomX
  ├─ CPU feature detection
  ├─ mining threads
  ├─ performance tuning
  └─ share submission
```

This distinction is important because it determines the maintenance strategy: **the less mining logic owned by this repository, the more plausible it is to finish the wrapper once and leave it alone.**

A compiled language is justified here primarily for packaging, structured API parsing, robust error handling, and a clean TUI. It does **not** imply that the project should become a large application.

## 1.2 CLI is the product core; TUI is only a frontend

The project must remain useful with no interactive UI at all:

```text
xmr-stack status
xmr-stack status --json
xmr-stack start
xmr-stack stop
xmr-stack restart
xmr-stack logs
xmr-stack doctor
```

The TUI should be an optional presentation layer over exactly the same core operations and `MiningSnapshot` model:

```text
CLI --------┐
            ├── core/adapters ──> systemd + XMRig API + P2Pool Data API
TUI --------┘
```

Consequences:

- no lifecycle logic inside TUI widgets;
- no statistics parser implemented only for TUI;
- no requirement to launch TUI to keep mining alive;
- if the TUI library ever becomes troublesome, the CLI remains usable;
- `xmr-stack tui` should be removable without redesigning the core.

This is a deliberate longevity feature.

---

# 2. Why a custom project still makes sense

There are already several projects that combine some or all of the same pieces, but none found during research exactly matches the desired combination:

```text
XMRig + P2Pool
+ headless-first
+ simple CLI configuration
+ systemd-backed persistent processes
+ live terminal dashboard
+ useful P2Pool and XMRig statistics
+ minimal dependencies
+ small codebase
+ no browser
+ no desktop environment
```

Gupax now supports daemon/headless operation, but it is still architecturally a GUI application with a daemon mode rather than a terminal-first orchestration tool.

The closest historical project is `monero-bash`, but it was archived in 2023.

There are newer shell installers such as Monerator, but their monitoring UI is much more limited.

There are also modern web dashboards such as Pithead and Monero Farm Panel, but they are substantially larger systems than what is desired here.

---

# 3. Important answer about maintenance

## 3.1 Can this wrapper itself be “written once and forgotten”?

**Mostly yes, if the scope stays narrow.**

A wrapper that:

- runs only on Linux;
- assumes systemd;
- invokes XMRig and P2Pool as separate processes;
- uses native XMRig JSON configuration;
- uses native P2Pool parameter files;
- reads XMRig's official HTTP API;
- reads P2Pool's official Data API files;
- uses `systemctl` / `journalctl`;
- does not download or patch upstream binaries;
- does not maintain a public remote service;
- does not depend on a cloud API;

can plausibly remain unchanged for a long time.

The wrapper is not doing mining itself. It is not implementing RandomX, Stratum, P2Pool consensus, Monero RPC, or wallet logic.

That is exactly why the architecture should stay thin.

## 3.2 What cannot be “forgotten”?

**The upstream mining components themselves still must be updated.**

This distinction is critical:

```text
our wrapper        -> can probably stay unchanged
XMRig              -> must occasionally be upgraded
P2Pool             -> must occasionally be upgraded
Monero node        -> must be kept compatible/up to date by whoever operates it
```

P2Pool is an especially strong example. In June 2026 a critical consensus vulnerability was disclosed. P2Pool v4.16 fixed it, and users were explicitly told to update because an attacker could affect payout calculation. Later v4.17 and v4.17.1 added more network hardening.

Therefore the project must **not create the illusion that a mining node can literally be installed and never touched again**.

The correct “low-maintenance” goal is:

> no scheduled maintenance of the wrapper itself; only normal replacement/update of XMRig/P2Pool when upstream requires it.

## 3.3 Design choices that minimize future maintenance

Do these:

- Keep XMRig and P2Pool **external**.
- Discover them by configurable paths.
- Do not patch them.
- Do not pin the wrapper to one exact upstream version.
- Prefer feature detection over exact version checks.
- Parse API responses tolerantly.
- Treat absent fields as unavailable, not fatal.
- Ignore unknown JSON fields.
- Keep systemd service names configurable.
- Keep ports configurable.
- Do not hardcode public remote nodes.
- Do not ship a list of “recommended” remote nodes that will rot.
- Avoid any online backend controlled by this project.
- Avoid automatic GitHub release downloading in the first release.
- Avoid a package manager for XMRig/P2Pool.
- Avoid Docker unless the user explicitly wants containers.
- Avoid storing historical statistics in a database.
- Avoid wallet creation/seed management.

Avoid these if “write once, forget” is the priority:

- full automatic installer for every Linux distribution;
- built-in auto-updater;
- shipping pinned XMRig/P2Pool binaries;
- remote-node discovery/crawler;
- browser UI;
- authentication system;
- database migrations;
- fleet management;
- cloud telemetry;
- plugin framework;
- support for Windows/macOS;
- support for several init systems;
- copying Gupax internals wholesale.

---

# 4. Current upstream state checked on 2026-09-10

These facts were checked during the research pass and should be treated as a dated snapshot, not eternal truth.

## XMRig

Repository:

- https://github.com/xmrig/xmrig
- https://xmrig.com/docs/miner/api

Latest release found during research:

- **XMRig v6.26.0**
- released **2026-03-28**
- adds, among other changes, RandomX v2 support.

XMRig exposes a built-in HTTP API.

Important endpoint:

```http
GET /2/summary
```

Official API docs:

- https://xmrig.com/docs/miner/api
- https://xmrig.com/docs/miner/api/summary

XMRig strongly prefers its native JSON config file for configuration.

Do not invent an independent wrapper-specific representation of every XMRig option.

---

## P2Pool

Repository:

- https://github.com/SChernykh/p2pool

Command-line documentation:

- https://github.com/SChernykh/p2pool/blob/master/docs/COMMAND_LINE.MD

Latest release found during research:

- **P2Pool v4.17.1**
- released **2026-06-28**

Important recent releases:

- **v4.16 — 2026-06-13**
  - fixed a critical consensus vulnerability;
- **v4.17 — 2026-06-21**
  - further P2P/network security hardening;
- **v4.17.1 — 2026-06-28**
  - additional spam/DoS hardening.

There is also public work/discussion around a future **P2Pool v5** reward-split design as of September 2026. Do not assume that every P2Pool detail will remain frozen forever.

### P2Pool now has a native params/config file

This is important for our architecture.

P2Pool supports:

```text
--params-file <file>
```

This appeared in P2Pool v4.13.

The command-line docs state that `--params-file` cannot be combined with other command-line parameters.

Example from upstream:

```text
p2pool --params-file params.conf
```

Example parameter file:

```ini
host = 127.0.0.1
wallet = YOUR_WALLET_ADDRESS
data-api = "/path/to/api/folder"
loglevel = 4
mini = 1
```

Therefore the wrapper should **not recreate the old `monero-bash` custom `p2pool.conf` abstraction** unless absolutely necessary.

Use the upstream-native P2Pool params file.

### P2Pool Data API

P2Pool supports:

```text
--data-api <path>
--local-api
```

`--data-api` writes JSON files to a directory.

`--local-api` enables additional local statistics for the Stratum server / built-in miner / P2P state.

Useful data surfaces seen in current projects:

```text
network/stats
pool/stats
pool/blocks
local/stratum
local/p2p
```

These are filesystem data written by P2Pool. P2Pool itself does not need to expose them over HTTP for a local TUI.

For our program:

> read the files directly from disk.

Do not add a web server between P2Pool and the TUI.

---

# 5. Existing projects investigated

These projects should be inspected by the coding agent before writing the final architecture. They are references, **not necessarily code to copy**.

---

## 5.1 Gupax

Repository:

https://github.com/gupax-io/gupax

### What it is

Current Gupax describes itself as a GUI uniting P2Pool and XMRig.

It now also contains functionality around:

- local Monero node;
- XMRig Proxy;
- XMRvsBeast;
- updates;
- monitoring;
- payouts;
- node discovery.

This is more scope than the desired project needs.

### Important finding: daemon/headless mode exists

Current Gupax README states that it can run without GUI:

```bash
gupax --daemon
```

This mode is explicitly intended for CLI-only environments.

No OpenGL/Vulkan-capable graphical environment is required for daemon mode.

While running, entering:

```text
s
```

prints status for started processes.

Configuration is shared with the normal GUI configuration.

### Why not simply use/clone Gupax?

Gupax daemon mode works, but it is not a terminal-first architecture.

The desired project should avoid inheriting:

- graphical UI framework dependencies;
- Gupax update manager;
- renderer logic;
- tab state;
- XvB logic;
- crawler logic;
- browser/desktop assumptions;
- large application state model.

### What to learn from Gupax

Inspect Gupax for:

- which statistics its authors consider useful;
- how it determines P2Pool synchronization/health;
- how it reports XMRig hashrate;
- process lifecycle edge cases;
- user-friendly labels and failure states.

Do **not** automatically copy implementation code.

Gupax is GPL-3.0.

---

## 5.2 `hinto-janai/monero-bash`

Repository:

https://github.com/hinto-janai/monero-bash

### Status

Archived:

**2023-04-25**

The repository explicitly says its update functionality is broken and it is no longer actively maintained.

### Why it matters

Conceptually this is probably the closest predecessor to what we want.

It is a Linux CLI wrapper for:

- Monero;
- P2Pool;
- XMRig.

Features included:

- package management;
- wallet menu;
- systemd process management;
- mining configuration;
- status;
- live `watch`;
- Tor;
- RPC helpers.

Relevant commands included:

```text
start <all/monero/p2pool/xmrig>
stop <all/monero/p2pool/xmrig>
restart <all/monero/p2pool/xmrig>
enable
disable
watch [monero|p2pool|xmrig]
status
version
```

`monero-bash watch` provided a live terminal status display and allowed switching process output with arrow keys.

### Most useful concepts to borrow

- CLI naming;
- systemd-backed lifetime;
- TUI/watch UX;
- separation of configs;
- `status` vs `watch`;
- `start all` / `stop all`;
- service hardening ideas;
- “mining continues after terminal exits”.

### Important lesson

`monero-bash` tried to also be a package manager/updater.

That part eventually broke.

For the current project, that is evidence **against** owning upstream updates.

### License

MIT.

This is the safest existing project from which to borrow small implementation ideas if its license notice requirements are respected.

Still prefer reimplementation rather than cargo-cult copying old Bash.

---

## 5.3 `Mik-TF/monerominer`

Repository:

https://github.com/Mik-TF/monerominer

A Bash-based installer/manager for:

- Monero node;
- P2Pool;
- XMRig.

It has:

```text
install
build
start
stop
restart
status
stats
```

It creates three systemd services.

Useful as a reference for a minimal installation flow.

Weakness for our use case:

- monitoring is not a proper TUI;
- it owns more of the installation stack;
- small project;
- Bash implementation is not ideal for a polished terminal dashboard.

---

## 5.4 `ts-manuel/monerator`

Repository:

https://github.com/ts-manuel/monerator

Last activity found via GitHub topic search:

- updated **2026-02-11**

It is based on `monerominer`.

Supports:

- Ubuntu/Debian;
- x86-64;
- aarch64;
- systemd;
- component configuration;
- start/stop/status/logs.

README commands:

```text
./monerator install
./monerator uninstall
./monerator start
./monerator stop
./monerator status
./monerator logs
```

It is a useful current reference for installation layout, but it is not a rich live dashboard.

---

## 5.5 `MBrassey/mine-monero`

Repository:

https://github.com/MBrassey/mine-monero

A larger deployment project for:

- monerod;
- P2Pool;
- XMRig;
- systemd;
- host tuning.

Useful ideas:

- service dependency structure;
- explicit local-only API bindings;
- XMRig HTTP API;
- restart ordering;
- health/diagnostic approach.

It is much more opinionated about host tuning and deployment than our wrapper should be.

Do not copy its hardware-specific tuning assumptions into a generic project.

---

## 5.6 `Elli610/P2Pool-frontend`

Repository:

https://github.com/Elli610/P2Pool-frontend

This is a web frontend, not a TUI, but it is useful because it demonstrates which P2Pool Data API files can power a dashboard.

It currently uses:

```text
/api/network/stats
/api/pool/stats
/api/pool/blocks
/api/local/stratum
/api/local/p2p
```

The HTTP layer is only serving files written by `--data-api`.

For our application, read those files directly.

Potential statistics:

- Monero height;
- network difficulty;
- block reward;
- pool hashrate;
- miners;
- blocks found;
- sidechain;
- miner hashrate;
- shares;
- effort;
- workers;
- P2P peers;
- uptime;
- ZMQ/node state.

---

## 5.7 `Ktololp/monero-farm-panel`

Repository:

https://github.com/Ktololp/monero-farm-panel

This is a much larger web/SSH fleet-management product, but as of the research date it is actively developed and contains useful current integration work around:

- XMRig HTTP API;
- P2Pool Data API;
- `local/stratum`;
- `pool/stats`;
- `network/stats`;
- system metrics;
- Huge Pages;
- MSR status;
- safe config editing;
- backups/rollback;
- systemd discovery.

Treat it as a modern integration reference.

Do **not** reproduce its fleet-management, SSH, web UI, history database, authentication, updater, or orchestration complexity.

---

## 5.8 `p2pool-starter-stack/pithead`

Repository:

https://github.com/p2pool-starter-stack/pithead

This is a modern and sophisticated stack with:

- Monero;
- P2Pool;
- XMRig-related routing/proxy;
- Tari merge mining;
- Tor;
- Docker Compose;
- live browser dashboard;
- history;
- alerts;
- configuration editor;
- upgrades;
- backup/restore.

It proves that there is active demand for integrated P2Pool operations, but it is **far beyond the intended scope**.

Useful architectural lesson:

> keep orchestration separate from mining processes.

Do not inherit the container fleet, dashboard, database, alerts, Tor automation, merge mining, XvB engine, or updater.

---

# 6. Recommended scope for v1

The first stable version should be deliberately boring.

## Supported

- Linux only.
- systemd only.
- x86_64 initially.
- XMRig.
- P2Pool.
- local or remote Monero node configured for P2Pool.
- interactive TUI.
- non-interactive CLI.
- static status output.
- JSON output for automation.
- process start/stop/restart.
- logs via journald.
- config validation.
- diagnostics.
- P2Pool main/mini/nano *if supported by the installed P2Pool*.
- configurable ports and paths.

## Explicitly not required for v1

- Windows.
- macOS.
- OpenRC.
- runit.
- Docker.
- Kubernetes.
- browser UI.
- mobile UI.
- XMRig Proxy.
- XvB.
- Tari merge mining.
- wallet generation.
- seed storage.
- wallet RPC.
- payouts database.
- price APIs.
- automatic profitability calculation.
- remote fleet management.
- multi-host SSH.
- Telegram notifications.
- automatic upstream binary downloads.
- automatic upstream binary upgrades.
- source compilation of XMRig or P2Pool.
- built-in Monero node installation.
- persistent chart history.

Every extra subsystem increases maintenance.

---

# 7. Recommended architecture

```text
                    ┌────────────────────────────┐
                    │         xmr-stack          │
                    │                            │
                    │ CLI        TUI             │
                    │ │           │              │
                    │ └────┬──────┘              │
                    │      │                     │
                    │  domain/status model       │
                    │      │                     │
                    │ ┌────┼───────────────┐     │
                    │ │    │               │     │
                    │ ▼    ▼               ▼     │
                    │systemd XMRig API  P2Pool API│
                    └──┬──────┬────────────┬─────┘
                       │      │            │
                       ▼      ▼            ▼
                    systemd  xmrig       p2pool
                              │            │
                              └────Stratum─┘
                                           │
                                           ▼
                                      Monero node
                                  local or remote
```

### Fundamental rule

The TUI is **not the process supervisor**.

Closing the TUI must never kill mining.

The mining services live under systemd.

---

# 8. Process management strategy

Use systemd.

Do not keep XMRig/P2Pool alive by:

- `nohup`;
- parent/child process tricks;
- custom daemonization;
- tmux;
- screen;
- hidden background threads in the TUI.

The wrapper should call `systemctl` using an argument array, not shell-concatenated commands.

For example conceptually:

```text
systemctl start xmr-p2pool.service
systemctl start xmr-xmrig.service
systemctl stop xmr-xmrig.service
systemctl stop xmr-p2pool.service
systemctl restart ...
systemctl is-active ...
systemctl is-enabled ...
```

Logs:

```text
journalctl -u xmr-p2pool.service
journalctl -u xmr-xmrig.service
```

### Possible service naming

Use project-specific names rather than generic `xmrig.service`, because a host might already have services:

```text
xmr-stack-p2pool.service
xmr-stack-xmrig.service
```

But allow service names to be overridden in config.

---

# 9. systemd relationship

A reasonable default:

```text
P2Pool starts before XMRig.
XMRig points to P2Pool Stratum.
```

The exact unit dependency should not be over-engineered.

Possible relationship:

```ini
[Unit]
After=xmr-stack-p2pool.service
Wants=xmr-stack-p2pool.service
```

for XMRig.

Avoid a brittle dependency graph that kills every component for every transient failure.

XMRig can naturally reconnect when its pool endpoint temporarily disappears.

Both services should have something similar to:

```ini
Restart=on-failure
RestartSec=5
```

The coding agent should evaluate `Wants=` vs `Requires=` based on desired stop behavior.

---

# 10. Privilege model

Do not require the entire TUI to run as root.

Read-only status should work unprivileged.

Start/stop/restart may:

- rely on normal `sudo systemctl ...`;
- use a narrowly scoped sudoers/polkit rule;
- or be performed by the user with the correct systemd permissions.

Do not implement a homemade privilege daemon.

### XMRig-specific privilege issue

Optimal RandomX performance may involve:

- Huge Pages;
- MSR tweaks;
- privileged system configuration.

Do not mix all of that into the first wrapper release.

Recommended boundary:

> host tuning is an optional setup concern, not the core supervisor.

The `doctor` command can report:

```text
Huge Pages: enabled / unavailable
MSR: active / not active / unknown
```

but the program does not need to own kernel tuning.

---

# 11. Configuration philosophy

Do **not** create one giant abstraction that mirrors every setting from XMRig and P2Pool.

The upstream projects already have configuration formats.

Use those.

Proposed arrangement:

```text
/etc/xmr-stack/
├── wrapper.toml
├── p2pool.conf
└── xmrig.json

/var/lib/xmr-stack/
└── p2pool-api/
```

Alternative user-local XDG layout can be added later if necessary.

## `wrapper.toml`

The wrapper config should contain only integration metadata.

Example concept:

```toml
[p2pool]
binary = "/usr/local/bin/p2pool"
params_file = "/etc/xmr-stack/p2pool.conf"
data_api_dir = "/var/lib/xmr-stack/p2pool-api"
service = "xmr-stack-p2pool.service"

[xmrig]
binary = "/usr/local/bin/xmrig"
config = "/etc/xmr-stack/xmrig.json"
api_url = "http://127.0.0.1:18088"
service = "xmr-stack-xmrig.service"

[ui]
refresh_ms = 1000
```

Do not duplicate wallet, CPU-thread count, node list, sidechain, and every miner option in the wrapper file unless the wrapper genuinely owns that setting.

---

# 12. P2Pool configuration

Prefer the upstream native P2Pool params file.

Concept:

```ini
host = 127.0.0.1
rpc-port = 18081
zmq-port = 18083
wallet = YOUR_PRIMARY_ADDRESS
data-api = "/var/lib/xmr-stack/p2pool-api"
local-api = 1
mini = 1
```

**The coding agent must verify the exact current params-file spelling/representation for each setting before generating this file.**

Reason: current docs show native params-file support, but do not assume every CLI flag maps exactly to the same textual form without checking source/current docs.

Also remember:

> `--params-file` cannot be combined with other CLI parameters.

Therefore if the service uses `--params-file`, all required P2Pool runtime settings need to live in that file.

---

# 13. XMRig configuration

Prefer native `config.json`.

The wrapper should only ensure that:

- XMRig points at the configured P2Pool Stratum endpoint;
- HTTP API is enabled;
- HTTP API is bound to loopback;
- CPU mining remains configured by XMRig itself unless the user changes it.

Do not write CPU-specific tuning tables into this project.

Do not maintain a CPU benchmark database.

Do not implement custom RandomX profile selection.

XMRig already does this work.

---

# 14. XMRig API integration

Use official HTTP API.

Main endpoint:

```http
GET /2/summary
```

Poll locally.

A practical default:

```text
timeout: 500–1000 ms
refresh: ~1 s
```

The agent should inspect the actual current response generated by the targeted XMRig version and map only the fields required by the UI.

Useful likely categories:

- version;
- uptime;
- current pool;
- algorithm;
- hashrate windows;
- accepted/rejected shares;
- huge page status;
- worker identifier;
- connection health.

### Future-proof parsing

Regardless of implementation language, decode only the fields that are actually needed and keep non-essential fields optional. In Go, ordinary `encoding/json` decoding naturally ignores unknown object fields; do not add strict unknown-field rejection for upstream status payloads. Tolerate individual missing metrics and fail the XMRig status section gracefully rather than crashing the whole CLI/TUI.

Desired behavior:

```text
XMRig service: ACTIVE
XMRig API: UNAVAILABLE
Hashrate: —
Last API success: 19s ago
```

rather than terminating.

---

# 15. P2Pool API integration

Prefer direct filesystem reads from the directory given to `--data-api`.

Expected useful surfaces:

```text
network/stats
pool/stats
pool/blocks
local/stratum
local/p2p
```

Do not require nginx, Node.js, Python, or any other HTTP file server.

### Future-proof file parsing

P2Pool may update files while we are reading them.

Handle:

- missing file;
- temporarily empty file;
- partial write;
- malformed JSON during rewrite;
- permissions;
- stale timestamps;
- schema additions;
- missing fields.

Recommended strategy:

1. read file;
2. parse;
3. if parse fails due to likely transient rewrite, retry once after a very short delay;
4. if it still fails, keep the last valid snapshot;
5. mark data as stale/error;
6. do not crash the TUI.

Do not persist the entire history unless necessary.

---

# 16. Unified internal status model

The UI should not directly depend on XMRig/P2Pool JSON structures.

Create an internal model such as:

```text
MiningSnapshot
├── timestamp
├── services
│   ├── p2pool
│   └── xmrig
├── xmrig
│   ├── api_ok
│   ├── uptime
│   ├── hashrate_10s
│   ├── hashrate_60s
│   ├── hashrate_15m
│   ├── accepted
│   ├── rejected
│   ├── pool
│   └── huge_pages
├── p2pool
│   ├── api_ok
│   ├── node_height
│   ├── sidechain
│   ├── local_hashrate
│   ├── shares
│   ├── effort
│   ├── workers
│   ├── peers
│   └── uptime
└── health
    ├── warnings[]
    └── errors[]
```

Everything should be optional if upstream does not provide it.

The TUI renders this model.

The CLI renders this model.

`--json` serializes this model.

This is the main architectural boundary.

---

# 17. Proposed command interface

Working command name:

```text
xmr-stack
```

Suggested v1:

```text
xmr-stack status
xmr-stack status --json

xmr-stack tui

xmr-stack start
xmr-stack stop
xmr-stack restart

xmr-stack start p2pool
xmr-stack stop p2pool
xmr-stack restart p2pool

xmr-stack start xmrig
xmr-stack stop xmrig
xmr-stack restart xmrig

xmr-stack logs
xmr-stack logs p2pool
xmr-stack logs xmrig

xmr-stack doctor
xmr-stack version

xmr-stack config path
```

Possible but not mandatory:

```text
xmr-stack init
xmr-stack service install
xmr-stack service uninstall
```

If an `init` command is implemented, keep it simple:

- detect user-provided XMRig binary;
- detect user-provided P2Pool binary;
- ask for node;
- ask for wallet;
- ask for P2Pool mode;
- write config;
- install units.

It should **not download miners**.

---

# 18. Proposed TUI

Think `htop`, not a desktop application.

One main screen may be enough.

Example:

```text
┌─ Monero Mining ───────────────────────────────────────────────────────┐
│ P2Pool  ● active   synced    Mini        peers 14    uptime 2d 04h  │
│ XMRig   ● active   connected             uptime 2d 04h              │
├─ Hashrate ────────────────────────────────────────────────────────────┤
│ 10s     13.84 kH/s                                                   │
│ 60s     13.77 kH/s      ▁▂▅▇▆▇▅▆▇▇▆▅▇                              │
│ 15m     13.71 kH/s                                                   │
├─ Shares ─────────────────────────┬─ P2Pool ───────────────────────────┤
│ accepted      183                │ local 15m     ...                  │
│ rejected        0                │ effort        ...                  │
│                                  │ workers       ...                  │
├─ Health ─────────────────────────┴────────────────────────────────────┤
│ ✓ XMRig API     ✓ P2Pool Data API     ✓ Stratum                     │
│ Last refresh 0.4s ago                                                 │
├───────────────────────────────────────────────────────────────────────┤
│ q quit   r refresh   l logs   s services   ? help                    │
└───────────────────────────────────────────────────────────────────────┘
```

This is illustrative, not a strict visual requirement.

## TUI rules

- `q` exits only the UI.
- Exiting does **not** stop mining.
- Do not make destructive actions single-key accidental operations.
- Start/stop should either require:
  - a secondary confirmation; or
  - a dedicated services popup.
- Display stale-data state clearly.
- Resize cleanly over SSH.
- Work with 80×24 reasonably.
- Monochrome terminal should remain understandable even if colors are unavailable.
- Do not depend on Unicode graphics for semantic meaning.
- No mouse is required.

---

# 19. History and charts

For v1:

- no SQLite;
- no Prometheus;
- no time-series database;
- no disk-persisted history.

If a graph is desired, maintain an in-memory ring buffer.

Example:

```text
last 5–15 minutes of 1-second/5-second samples
```

When the TUI exits, the graph history disappears.

That is acceptable.

Persistent historical analytics can be a separate future project if ever needed.

This choice drastically reduces long-term maintenance.

---

# 20. `doctor` command

A useful low-complexity feature.

Example:

```text
$ xmr-stack doctor

Wrapper config           OK
P2Pool binary            OK  v4.17.1
XMRig binary             OK  v6.26.0

P2Pool service           active
XMRig service            active

P2Pool data API          OK
P2Pool local API         OK
XMRig HTTP API           OK

P2Pool Stratum           127.0.0.1:3333 reachable
Monero RPC               127.0.0.1:18081 reachable
Monero ZMQ               127.0.0.1:18083 reachable

Huge Pages               available
MSR                      unknown

Warnings: 0
```

This should be read-only.

No magical “fix everything” behavior in v1.

---

# 21. Version handling

Do not couple behavior to a giant matrix such as:

```text
if P2Pool == 4.15
if P2Pool == 4.16
...
```

Prefer:

```text
p2pool --version
xmrig --version
```

for display, and feature detection when needed.

Examples:

- check whether configured P2Pool Data API files actually appear;
- check whether XMRig API responds;
- verify `--params-file` support if supporting older P2Pool installations.

A minimum supported P2Pool version may be reasonable.

Given the 2026 security issue, targeting modern P2Pool rather than supporting ancient versions is preferable.

---

# 22. Compatibility philosophy

The wrapper should be:

```text
strict about configuration syntax we own
loose about optional upstream fields
strict about destructive operations
loose about added upstream API fields
```

This is the correct approach for low-maintenance glue software.

---

# 23. Failure modes that must be handled

At minimum:

## XMRig

- binary missing;
- config missing;
- process fails immediately;
- API disabled;
- API port incorrect;
- API timeout;
- P2Pool unavailable;
- zero hashrate;
- rejected shares;
- permissions/Huge Pages/MSR issue;
- invalid config.

## P2Pool

- binary missing;
- params file invalid;
- wallet missing/invalid;
- Monero node unavailable;
- Monero ZMQ unavailable;
- not yet synchronized;
- Data API path unavailable;
- Data API files temporarily malformed;
- local API not enabled;
- Stratum port conflict;
- P2P port conflict;
- P2Pool process crashes.

## systemd

- unit does not exist;
- service disabled;
- permission denied;
- service start timeout;
- restart loop;
- executable path changed.

## UI

- SSH terminal resized;
- API temporarily disappears;
- one component down and other up;
- stale values;
- non-UTF-8 log data;
- no color;
- extremely narrow terminal.

---

# 24. Startup ordering

Do not blindly start XMRig at exactly the same instant as P2Pool if P2Pool is not usable yet.

Potential flow:

```text
start P2Pool
    ↓
wait until service active
    ↓
optionally wait for P2Pool readiness / Stratum socket
    ↓
start XMRig
```

However do not implement an elaborate state machine in v1.

A simple readiness wait with timeout is enough.

If XMRig starts early, it can reconnect.

The UI should make the distinction clear:

```text
P2Pool process: active
P2Pool ready: no / syncing
XMRig process: active
XMRig pool connection: reconnecting
```

---

# 25. Stopping

For `stop all`:

```text
stop XMRig first
stop P2Pool second
```

This avoids a period where XMRig is needlessly attempting to reconnect.

For restart:

```text
restart P2Pool
wait for readiness
XMRig reconnects automatically
```

It may not be necessary to restart XMRig every time P2Pool restarts.

Keep behavior predictable.

---

# 26. Node handling

P2Pool requires access to a compatible Monero node.

The first version should support both:

```text
local node: 127.0.0.1
remote node: configurable host
```

But the wrapper does not need to install or supervise `monerod`.

This reduces scope dramatically.

Optional later command:

```text
xmr-stack doctor node
```

could verify RPC/ZMQ connectivity.

Do not maintain a remote-node discovery list.

---

# 27. Wallet handling

P2Pool payout address is not a secret like a seed phrase, but it is user-sensitive configuration.

The wrapper should never require:

- wallet seed;
- private spend key;
- private view key;
- wallet password.

Only the public mining payout address should be needed.

P2Pool has evolved: current P2Pool command-line docs include support for a subaddress together with the main address, while some older Monero mining documentation still says subaddresses are unsupported.

Therefore:

> do not write brittle wallet validation based on old assumptions.

If validating addresses, keep validation conservative.

Better:

- require non-empty value;
- optionally perform basic Monero address validation through a small library only if worth the dependency;
- let P2Pool remain the source of truth.

---

# 28. API/network security

## XMRig API

Bind it to loopback:

```text
127.0.0.1
```

Do not expose it publicly.

If the TUI runs on the same server, there is no reason for LAN exposure.

## P2Pool Data API

Read files directly from local disk.

No HTTP server is required.

## Stratum

If only the local XMRig instance uses P2Pool:

```text
127.0.0.1:3333
```

is the safest simple default.

If external miners will connect later, make bind address configurable.

Do not default to `0.0.0.0` unless the product intentionally supports remote workers.

## Shell execution

Never form commands like:

```text
"systemctl " + user_input
```

Use process argument arrays.

Do not `eval` config values.

---

# 29. Language recommendation

## Recommended default: Go + Bubble Tea

For this specific project, **Go is the preferred default** if the main priority is:

```text
minimum code
+ one easy-to-deploy binary
+ low operational dependency count
+ simple HTTP/JSON/system integration
+ a capable TUI
+ minimal long-term maintenance
```

Suggested stack:

```text
standard library      process execution, JSON, HTTP, filesystem, networking
cobra or urfave/cli   CLI parsing (optional; stdlib flag is also sufficient)
BurntSushi/toml or similar small TOML parser
Bubble Tea            TUI state/update loop
Bubbles                only for widgets that materially help
Lip Gloss              only for presentation; avoid styling complexity
```

Prefer the Go standard library wherever it keeps the code clearer. Every third-party dependency is future maintenance surface.

### Why Go fits this wrapper particularly well

- produces a single executable;
- easy static or near-static Linux builds;
- standard library already covers HTTP, JSON, process execution, signals, files, and TCP checks;
- concurrency model is more than sufficient for a few 1-second polling tasks;
- JSON decoding is tolerant of unknown upstream fields by default;
- Bubble Tea is well suited to a small SSH-friendly TUI;
- the code can remain structurally close to a simple script rather than turning into a framework-heavy application;
- build and release workflow is straightforward.

The important point is not that Go is “better” than Rust generally. It is that **this project is intentionally just glue**, so simplicity has more value than maximum type-level sophistication.

### Keep the Go implementation boring

Do not introduce:

- dependency injection frameworks;
- code generation;
- reflection-heavy configuration systems;
- an internal event bus;
- plugin systems;
- ORM/database libraries;
- a web framework;
- complex reactive abstractions.

Simple structs + interfaces + goroutines + channels/tickers where genuinely needed are enough.

## Strong alternative: Rust + Ratatui

Rust remains an excellent option if the implementing agent/user prefers it.

Typical stack:

```text
clap
serde / serde_json
toml
reqwest
ratatui
crossterm
thiserror
```

Advantages:

- strong compile-time modeling;
- excellent error/type discipline;
- Ratatui is mature;
- still ships as one native binary.

Trade-off for this project:

- more code/ceremony is likely for a wrapper whose domain model is small;
- build/dependency complexity is somewhat higher than necessary for the task.

Therefore Rust is **second choice for minimalism**, not a bad choice.

## Bash + jq: valid only if the full TUI is dropped

If requirements are reduced to:

```text
start
stop
restart
status
logs
watch-like refreshed text
```

then a Bash implementation can be entirely reasonable and may truly be only a few hundred lines.

However a polished interactive TUI introduces awkward terminal control, signal handling, background polling, redraw synchronization, and JSON processing. At that point Bash tends to become harder to maintain than a small Go binary.

So:

```text
full interactive TUI wanted  -> Go preferred
simple CLI/watch only         -> Bash is viable
```

## Python

Easy to prototype, but the interpreter/package-runtime dependency is unnecessary for the long-term “drop one binary on a server” objective.

---
# 30. Repository structure recommendation

Keep it compact. A Go version can remain very small:

```text
xmr-stack/
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── cmd/
│   └── xmr-stack/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── model/
│   │   └── snapshot.go
│   ├── service/
│   │   └── systemd.go
│   ├── xmrig/
│   │   └── api.go
│   ├── p2pool/
│   │   └── api.go
│   ├── doctor/
│   │   └── doctor.go
│   └── tui/
│       └── tui.go
├── systemd/
│   ├── xmr-stack-p2pool.service
│   └── xmr-stack-xmrig.service
├── examples/
│   ├── wrapper.toml
│   ├── p2pool.conf
│   └── xmrig.json
├── testdata/
│   ├── xmrig-summary.json
│   ├── p2pool-network-stats.json
│   ├── p2pool-pool-stats.json
│   ├── p2pool-local-stratum.json
│   └── p2pool-local-p2p.json
└── docs/
    ├── architecture.md
    └── troubleshooting.md
```

This is only a suggested ceiling, not a requirement to create every directory on day one. If the code can stay simpler with fewer packages, prefer fewer packages.

Do not create dozens of tiny modules merely for architectural purity.

The codebase should remain easy for one person or one AI agent to understand in one context window.

---
# 31. Adapter boundaries

Define small interfaces around external systems. In Go these can stay minimal and consumer-owned.

Conceptually:

```text
ServiceManager
    Status()
    Start()
    Stop()
    Restart()

XmrigSource
    Snapshot()

P2PoolSource
    Snapshot()
```

Production adapters:

```text
SystemdServiceManager
HttpXmrigSource
FilesystemP2PoolSource
```

Tests use fakes/fixtures.

This is enough abstraction.

Do not build an extensible plugin system.

---

# 32. Testing strategy

The purpose of tests is future compatibility, not chasing coverage percentage.

## Unit tests

### XMRig parsing

Fixtures:

1. normal current `/2/summary`;
2. missing optional field;
3. extra unknown fields;
4. zero hashrate;
5. rejected shares;
6. malformed JSON.

### P2Pool parsing

Fixtures:

1. current network stats;
2. pool stats;
3. local stratum;
4. local P2P;
5. missing file;
6. partial/malformed write;
7. extra unknown fields;
8. missing optional fields.

## Service manager

Use a mock command runner.

Test:

- active;
- inactive;
- failed;
- unit missing;
- permission denied.

Do not invoke real `systemctl` in normal unit tests.

## Snapshot aggregation

Test mixed states:

```text
P2Pool up, XMRig down
P2Pool down, XMRig reconnecting
both up, one API unavailable
stale P2Pool data
```

## TUI

Do not over-test exact pixel layout.

Test:

- application state transitions;
- key commands;
- no crash on narrow terminal;
- stale/error rendering path.

## Manual integration test

Have one documented test against real:

- modern XMRig;
- modern P2Pool;
- local or test node;
- systemd services.

That is enough.

---

# 33. Tolerant parsing is a core maintenance feature

This deserves emphasis.

Bad:

```text
model every upstream response field as required
reject responses when upstream adds/removes unrelated fields
make one missing metric invalidate the entire snapshot
```

That creates unnecessary coupling and turns routine upstream evolution into wrapper maintenance.

Better:

```text
deserialize only what we display
make non-essential values optional
ignore unknown fields
validate required boundaries separately
```

The wrapper should survive:

```text
upstream adds JSON fields
upstream removes a cosmetic field
one metric temporarily disappears
```

without needing a release.

---

# 34. Do not scrape logs for primary metrics

Primary status should come from structured APIs.

Use logs for:

- troubleshooting;
- recent errors;
- user inspection.

Do not derive current hashrate from regexing console logs if the XMRig API is available.

Do not derive P2Pool statistics from human-readable console lines if Data API is available.

Human-readable log formats are much more likely to change.

---

# 35. Logs UI

Simplest option:

```text
xmr-stack logs xmrig
```

execs or wraps:

```text
journalctl -f -u xmr-stack-xmrig.service
```

Likewise for P2Pool.

The TUI can optionally show the last 5–20 error/warning lines, but do not make an embedded terminal/log viewer mandatory for v1.

---

# 36. Installing upstream programs

To minimize responsibility, the wrapper should expect:

```text
p2pool
xmrig
```

to already exist.

The README should show where to get them:

- official XMRig releases;
- official P2Pool releases;
- distro package if trusted/current.

Possible UX:

```text
$ xmr-stack doctor
P2Pool binary: not found

Configure path:
  /etc/xmr-stack/wrapper.toml

Official upstream:
  https://github.com/SChernykh/p2pool/releases
```

Do not silently `curl | bash` upstream releases.

Do not implement a binary downloader in v1.

---

# 37. Updating upstream programs

The wrapper should not own update policy.

Ideal design:

1. user replaces XMRig binary;
2. user replaces P2Pool binary;
3. service restarts;
4. wrapper continues working.

The wrapper config points to stable paths such as:

```text
/usr/local/bin/xmrig
/usr/local/bin/p2pool
```

or paths managed by the OS/package manager.

This keeps the wrapper release independent from upstream release cadence.

---

# 38. Why bundling is a bad fit for the maintenance goal

If this project ships:

```text
xmr-stack + xmrig + p2pool
```

as one release artifact, then every critical P2Pool or XMRig update becomes **our release problem**.

That turns a thin wrapper into a distribution.

Gupax does this and therefore has to maintain update/bundle machinery.

The desired project should avoid that burden.

---

# 39. Licensing considerations

Not legal advice, but important engineering guidance.

Current licenses seen during research:

- Gupax: GPL-3.0
- P2Pool: GPL-3.0
- XMRig: GPL-3.0-or-later / GPL family in source
- `monero-bash`: MIT
- Monerator: Apache-2.0
- `monerominer`: Apache-2.0
- Pithead: MIT for its own code, third-party software retains its licenses
- Monero Farm Panel: MIT

If our wrapper:

- launches upstream executables as separate processes;
- uses their documented API/config interfaces;
- does not copy GPL implementation code;

then it is architecturally and legally cleaner than embedding upstream source.

If GPL code from Gupax/P2Pool/XMRig is copied into this project, licensing obligations change.

Simplest policy:

> study GPL projects for behavior and interfaces; implement our own wrapper logic; do not vendor their code.

---

# 40. Suggested implementation phases

## Phase 0 — API compatibility spike

Before building the UI:

- run modern P2Pool with Data API;
- collect real example files;
- run modern XMRig with HTTP API;
- collect `/2/summary`;
- verify fields;
- save sanitized fixtures.

Deliverable:

```text
fixtures + short notes
```

No TUI yet.

---

## Phase 1 — read-only `status`

Implement:

```text
xmr-stack status
xmr-stack status --json
```

Read:

- systemd;
- XMRig API;
- P2Pool Data API.

This proves the core integrations.

Acceptance:

- no crashes if one component is down;
- useful error messages;
- valid JSON mode.

---

## Phase 2 — process control

Implement:

```text
start
stop
restart
logs
```

Use systemd.

Acceptance:

- TUI/process tool is never parent of mining processes;
- closing shell/UI leaves mining running.

---

## Phase 3 — `doctor`

Implement checks for:

- binaries;
- config paths;
- systemd units;
- API;
- Data API;
- Stratum;
- node connectivity;
- permissions.

---

## Phase 4 — TUI

Build the Bubble Tea dashboard over the already-proven snapshot model. If Rust is chosen instead, use Ratatui with the same architectural boundary.

Do not put business logic in render functions.

---

## Phase 5 — optional setup helper

Only after everything else works:

```text
xmr-stack init
xmr-stack service install
```

Do not let installation complexity delay the core application.

---

# 41. Acceptance criteria for a stable v1

A release is successful if a user can:

1. install XMRig and P2Pool independently;
2. create/configure the required files;
3. install/enable the two systemd units;
4. run:

```text
xmr-stack start
```

5. close the SSH session;
6. reconnect hours later;
7. run:

```text
xmr-stack status
```

and see meaningful health/hashrate statistics;
8. run:

```text
xmr-stack tui
```

and see live statistics;
9. press `q`;
10. verify XMRig and P2Pool continue running;
11. update the XMRig and P2Pool binaries independently;
12. restart services;
13. continue using the same wrapper binary.

That last item is particularly important.

---

# 42. “Write once and forget” compatibility test

Before calling the project finished, perform this thought experiment:

> If P2Pool 4.18 adds three JSON fields and XMRig 6.27 adds two fields to `/2/summary`, does the wrapper still run?

Expected:

```text
yes
```

> If a non-essential field disappears?

Expected:

```text
yes; metric shows unavailable
```

> If service paths move?

Expected:

```text
change wrapper.toml only
```

> If XMRig binary is replaced?

Expected:

```text
restart service; wrapper unchanged
```

> If P2Pool binary is replaced?

Expected:

```text
restart service; wrapper unchanged unless upstream intentionally breaks the Data API/config contract
```

> If P2Pool changes consensus?

Expected:

```text
update P2Pool; wrapper should not care
```

This is the design target.

---

# 43. What would force future wrapper maintenance?

Likely triggers:

1. XMRig removes or fundamentally changes `/2/summary`.
2. P2Pool removes or fundamentally changes Data API file schemas.
3. P2Pool changes `--params-file` semantics.
4. systemd CLI behavior fundamentally changes.
5. the compiled CLI/TUI eventually needs rebuilding for a future OS/toolchain/platform environment.
6. a security bug is found in the wrapper's dependencies.
7. user requests new features.

These are relatively infrequent compared with the upstream mining release cadence.

---

# 44. What should NOT force wrapper maintenance?

These should normally require only upstream binary updates:

- RandomX implementation changes;
- XMRig performance fixes;
- P2Pool consensus fixes;
- P2P hardening;
- Monero protocol changes handled inside upstream components;
- CPU optimization changes;
- new P2Pool peer behavior;
- mining algorithm internals.

If one of these requires our wrapper to change, our abstraction boundary is probably too tight.

---

# 45. Current research conclusion

The optimal project is **not** a “combined miner” and should not be treated as a large standalone application.

It is best understood as a:

> **small glue/orchestration script with a compiled CLI and optional terminal dashboard for existing XMRig + P2Pool services**

Think:

```text
docker compose UX
+
htop-like status
+
systemd
```

for one very specific mining stack.

This is small enough that the wrapper itself can plausibly become “finished software”.

The mining components cannot be frozen forever, but they can be upgraded independently.

---

# 46. Concrete recommendation to the coding agent

Before writing production code:

1. inspect current:
   - XMRig API docs/source;
   - P2Pool command-line docs;
   - actual P2Pool Data API output;
   - `monero-bash` UX;
   - current Gupax daemon/status behavior;
2. build a tiny API spike;
3. save real sanitized fixtures;
4. define the internal `MiningSnapshot`;
5. implement `status`;
6. only then add process control and TUI.

Do not begin by designing the TUI.

The hard part is defining stable adapter boundaries. The implementation itself should stay intentionally small. Do not mistake use of Go/Rust and a TUI library for permission to build a framework.

---

# 47. Recommended agent constraints

Give the implementing AI agent these explicit rules:

```text
- Treat the official Monero XMRig + P2Pool guide as the canonical mining topology.
- Treat official P2Pool CLI docs as the source of truth for P2Pool configuration.
- Treat official XMRig docs/API as the source of truth for XMRig configuration and metrics.
- Do not invent a wrapper-specific XMRig↔P2Pool integration layer.
- Keep upstream-native config files usable without this wrapper.
- Do not fork Gupax.
- Do not modify XMRig.
- Do not modify P2Pool.
- Do not vendor either source tree.
- Do not implement mining logic.
- Do not implement Stratum.
- Do not implement Monero RPC beyond optional health checks.
- Do not implement an updater in v1.
- Do not implement a browser UI.
- Do not introduce a database in v1.
- Do not manage wallet secrets.
- Do not make TUI lifetime equal mining lifetime.
- Treat CLI/core as the product; TUI is only an optional frontend.
- Keep the project conceptually equivalent to a small glue script even if compiled.
- Prefer Go + Bubble Tea for the default implementation if there is no stronger language constraint.
- If requirements shrink to CLI + refreshed text only, consider Bash + jq instead of overbuilding.
- Keep upstream configs canonical.
- Parse upstream statistics tolerantly.
- Every optional upstream metric must be allowed to disappear without crashing.
- Prefer stable OS/upstream interfaces over log scraping.
- Keep Linux/systemd-only until the core project is stable.
```

---

# 48. Potential MVP command transcript

```text
$ xmr-stack status

P2Pool  ACTIVE
  version        4.17.1
  sidechain      mini
  node           127.0.0.1:18081
  peers          16
  local hashrate 13.7 kH/s
  shares         4
  effort         83 %
  data age       1.2 s

XMRig   ACTIVE
  version        6.26.0
  pool           127.0.0.1:3333
  hashrate 10s   13.8 kH/s
  hashrate 60s   13.7 kH/s
  hashrate 15m   13.7 kH/s
  shares         183 accepted / 0 rejected
  uptime         2d 04h

Health  OK
```

JSON:

```text
$ xmr-stack status --json
{
  "health": "ok",
  "p2pool": { ... },
  "xmrig": { ... }
}
```

Control:

```text
$ sudo xmr-stack restart p2pool
Restarted xmr-stack-p2pool.service
Waiting for Stratum...
P2Pool ready.
XMRig reconnected.
```

---

# 49. Possible config ownership model

To reduce accidental complexity:

```text
wrapper.toml
    owns only integration

p2pool.conf
    owned by P2Pool/user

xmrig.json
    owned by XMRig/user
```

`xmr-stack init` may create initial versions, but later must not aggressively rewrite them.

Avoid a config-sync engine.

If the user manually edits `xmrig.json`, respect it.

If the user manually edits `p2pool.conf`, respect it.

The wrapper is an observer/operator, not the single source of truth for every upstream option.

---

# 50. Configuration changes from TUI

Not required for v1.

This is intentional.

A settings editor looks convenient but creates:

- schema mirroring;
- validation burden;
- migration burden;
- write/rollback logic;
- privilege problems;
- upstream-version coupling.

For low maintenance, the TUI should initially be:

```text
monitor + lifecycle control
```

Configuration remains files + `init`.

---

# 51. Remote miners

Not required for v1, but do not architecturally block them.

If later external XMRig workers mine to the same P2Pool instance:

```text
remote XMRig ──Stratum──> P2Pool
local XMRig  ──Stratum──> P2Pool
```

P2Pool `local/stratum` may provide worker statistics.

The TUI can eventually display connected workers.

But v1 does not need fleet orchestration.

---

# 52. Why no monerod management in MVP

Adding monerod means adding:

- blockchain data directory;
- pruning;
- disk sizing;
- sync progress;
- RPC security;
- ZMQ config;
- network peers;
- bootstrap nodes;
- upgrades;
- long initial sync;
- much more disk/error handling.

The user originally asked for XMRig + P2Pool.

P2Pool may connect to:

- local separately managed monerod;
- remote compatible node.

Therefore monerod should remain an external dependency for MVP.

A later optional adapter can monitor it.

---

# 53. Documentation that should ship

Small set:

## README

- what project does;
- architecture;
- prerequisites;
- quick start;
- commands;
- update philosophy.

## `docs/troubleshooting.md`

Common failures:

- no XMRig API;
- P2Pool data directory empty;
- node/ZMQ unreachable;
- Stratum connection refused;
- permission denied on systemd;
- Huge Pages/MSR warning.

## `docs/updating.md`

Very short:

```text
The wrapper does not update XMRig/P2Pool.
Update those components from their official source/package manager.
Restart services afterward.
```

Make this distinction explicit so users know the wrapper being “stable” does not mean mining software can be abandoned indefinitely.

---

# 54. Sources checked during research

## Primary upstream sources

### Gupax

- https://github.com/gupax-io/gupax
- https://github.com/gupax-io/gupax/blob/main/README.md
- https://github.com/gupax-io/gupax/releases

Key finding:
- official daemon mode for CLI-only environments;
- current Gupax still actively maintained in 2026;
- Gupax had to publish a critical P2Pool update notice.

### XMRig

- https://github.com/xmrig/xmrig
- https://github.com/xmrig/xmrig/releases
- https://xmrig.com/docs/miner/api
- https://xmrig.com/docs/miner/api/summary

Key finding:
- official local HTTP API;
- `/2/summary`;
- current v6.26.0 as found on 2026-09-10.

### P2Pool

- https://github.com/SChernykh/p2pool
- https://github.com/SChernykh/p2pool/releases
- https://github.com/SChernykh/p2pool/blob/master/docs/COMMAND_LINE.MD

Key findings:
- native `--params-file`;
- `--data-api`;
- `--local-api`;
- v4.16 critical consensus fix;
- v4.17/v4.17.1 security hardening.

### Monero mining guide

- https://github.com/monero-project/monero-docs/blob/master/docs/en/interacting/mining/guides/p2pool/xmrig-p2pool.md

Key finding:
- canonical XMRig → P2Pool relationship;
- reminder that P2Pool needs a properly configured Monero node;
- upstream guide explicitly advises keeping P2Pool updated.

---

## Existing integration projects

### monero-bash

- https://github.com/hinto-janai/monero-bash

### monerominer

- https://github.com/Mik-TF/monerominer

### Monerator

- https://github.com/ts-manuel/monerator

### mine-monero

- https://github.com/MBrassey/mine-monero

### P2Pool frontend

- https://github.com/Elli610/P2Pool-frontend

### Monero Farm Panel

- https://github.com/Ktololp/monero-farm-panel

### Pithead / P2Pool Starter Stack

- https://github.com/p2pool-starter-stack/pithead
- https://github.com/p2pool-starter-stack

---

# 55. Final decision summary

If the objective is:

> build this once and ideally never maintain the wrapper again,

then the recommended design is:

```text
small Go CLI core
    +
optional Bubble Tea TUI
    ↓
thin internal snapshot model
    ↓
systemctl/journalctl
XMRig official HTTP API
P2Pool official Data API files
    ↓
external XMRig + P2Pool binaries
```

Conceptually, this should still feel like **a robust script that glues two programs together**, not like a new mining application. If the project can be made smaller without losing the required CLI/TUI workflow, smaller is preferable.

with:

```text
NO bundling
NO upstream source modifications
NO auto-updater
NO database
NO web UI
NO wallet secrets
NO monerod ownership in MVP
NO multi-distro installer logic beyond systemd/Linux
NO framework/plugin architecture
NO duplicated mining logic
NO TUI-only business logic
```

The wrapper may still eventually need an occasional compatibility fix, but this architecture makes that the exception rather than routine maintenance.

**XMRig and P2Pool themselves must still be updated when upstream publishes security, consensus, compatibility, or mining-algorithm updates.**

That separation is the key design principle for the entire project.
