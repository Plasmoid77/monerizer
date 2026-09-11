# Monerizer

Маленький CLI/TUI для эксплуатации одной пары **P2Pool + XMRig** под systemd на Linux. Один бинарник без зависимостей; майнеры остаются штатными upstream-программами со своими нативными конфигами и обновляются независимо. Monerizer ничего не майнит, не хранит и не переписывает — только `systemctl`, `journalctl`, HTTP API XMRig и файлы Data API P2Pool.

```text
XMRig ──Stratum──▶ P2Pool ──RPC/ZMQ──▶ Monero-нода
   ▲                  ▲
   └── monerizer ─────┘   (systemd · GET /2/summary · чтение /run/…-api)
```

Состояние: v1 реализована и проверена на Arch Linux (systemd 261) и Debian 13 (systemd 257) с P2Pool 4.18 и XMRig 6.26.0 — [отчёт приёмки](docs/research/acceptance-report-v1.md).

## Что умеет

| Команда | Что делает |
|---|---|
| `monerizer status [--json] [--check]` | Состояние служб, показатели XMRig/P2Pool, свежесть источников, health и причины. `--check` → код 1, если health не `ok` |
| `monerizer tui` | Та же панель интерактивно: обновление, меню start/stop/restart с подтверждением, просмотр журнала |
| `monerizer start\|stop\|restart [all\|p2pool\|xmrig]` | Управление через systemd; после операции печатает фактическое состояние |
| `monerizer logs [--follow] [--lines N] [target]` | `journalctl` по точным unit-именам |
| `monerizer doctor [--json]` | 30 проверок: units, зависимости, права, API, свежесть, нода |
| `monerizer node list` / `node select [--dry-run]` | Проба нод из `nodes.txt` (RPC latency, sync, ZMQ-порт); `select` переписывает `host/rpc-port/zmq-port` в `p2pool.conf` |
| `monerizer config path`, `version` | Служебные |

Границы (намеренно): нет установщика бинарников, автообновлений, базы данных, web-UI, управления `monerod`, нескольких стеков, настройки ядра/MSR/hugepages. Полный перечень — в [ТЗ](docs/superpowers/specs/2026-09-11-monerizer-design.md).

## Требования

- Linux x86_64, systemd ≥ 249 (проверено на 261), polkit — только для опционального управления без sudo.
- Установленные бинарники [P2Pool](https://github.com/SChernykh/p2pool/releases) и [XMRig](https://github.com/xmrig/xmrig/releases) (официальные релизы, проверка checksum/подписи).
- Monero-нода с открытыми RPC и ZMQ (`--zmq-pub`): своя или удалённая. Удалённая нода видит IP хоста и адрес выплат.
- Основной адрес кошелька Monero (начинается с `4`).

## Установка

Пошагово — [docs/install.md](docs/install.md). Кратко:

1. Собрать: `make build` (Go 1.27, `CGO_ENABLED=0`) → `./monerizer`.
2. Установить `systemd/*.service`, `examples/{p2pool.conf,xmrig.json,monerizer.toml,nodes.txt}` в `/etc/monerizer`, создать группу `monerizer` и двух системных пользователей — команды в `docs/research/stand-install.sh`.
3. Вписать в `/etc/monerizer/p2pool.conf` адрес выплат и ноду (или `sudo monerizer node select`).
4. `monerizer doctor`, затем `sudo monerizer start`, затем `monerizer status`.

## Права

- `status`, `tui`, `doctor`, `logs` — обычный пользователь из группы `monerizer` (чтение Data API и конфигов) и `systemd-journal`/`wheel` для журнала.
- `start/stop/restart` — через `sudo` либо через опциональное polkit-правило `examples/polkit/50-monerizer.rules` (только два unit, только start/stop/restart, только группа `monerizer`).
- `node select` пишет в `p2pool.conf` → `sudo`.
- Каталог Data API содержит `local/console` с cookie TCP-консоли P2Pool: право чтения каталога равнозначно управлению P2Pool, поэтому группа одна.

## Документы

- [Установка](docs/install.md) · [Диагностика](docs/troubleshooting.md) · [Обновление upstream](docs/updating.md)
- [ТЗ v1](docs/superpowers/specs/2026-09-11-monerizer-design.md) · [План реализации](docs/superpowers/plans/2026-09-11-monerizer-implementation.md)
- [Контракты источников](docs/research/upstream-contracts.md) · [Отчёт стенда №1](docs/research/stand-report-1.md) · [№2](docs/research/stand-report-2.md) · [Приёмка v1](docs/research/acceptance-report-v1.md)
- [Исходный handoff](xmrig-p2pool-tui-ai-handoff-v2.md) — история; решения ТЗ имеют приоритет.

Лицензия — MIT. XMRig и P2Pool не входят в поставку и распространяются по своим лицензиям.
