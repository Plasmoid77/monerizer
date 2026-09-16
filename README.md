# Moneroid

*Minimal CLI/TUI to run one P2Pool + XMRig pair under systemd on Linux: status, health, doctor, logs, start/stop, node probing. Single static binary, upstream programs stay untouched. Docs are in Russian; see `docs/architecture.md` for a map of the code.*

Маленький CLI/TUI для эксплуатации одной пары **P2Pool + XMRig** под systemd на Linux. Один бинарник без зависимостей; майнеры остаются штатными upstream-программами со своими нативными конфигами и обновляются независимо. Moneroid ничего не майнит, не хранит и не переписывает — только `systemctl`, `journalctl`, HTTP API XMRig и файлы Data API P2Pool.

```text
XMRig ──Stratum──▶ P2Pool ──RPC/ZMQ──▶ Monero-нода
   ▲                  ▲
   └── moneroid ─────┘   (systemd · GET /2/summary · чтение /run/…-api)
```

Состояние: v1 реализована и проверена на Arch Linux (systemd 261) и Debian 13 (systemd 257) с P2Pool 4.18 и XMRig 6.26.0 — [отчёт приёмки](docs/research/acceptance-report-v1.md).

## Что умеет

| Команда | Что делает |
|---|---|
| `moneroid status [--json] [--check]` | Состояние служб, показатели XMRig/P2Pool, свежесть источников, health и причины. `--check` → код 1, если health не `ok` |
| `moneroid tui` | Живая панель во весь экран в духе `htop`: hashrate с полосами и sparkline, P2Pool, выплаты, проблемы; меню start/stop/restart с подтверждением, просмотр журнала, экран выплат (`p`) |
| `moneroid start\|stop\|restart [all\|p2pool\|xmrig]` | Управление через systemd; после операции печатает фактическое состояние |
| `moneroid logs [--follow] [--lines N] [target]` | `journalctl` по точным unit-именам |
| `moneroid doctor [--json]` | 31 проверка: units, зависимости, права, API, свежесть, нода |
| `moneroid payouts [--json] [--since TIME]` | Выплаты из журнала P2Pool: время, сумма, блок, итог; адрес кошелька не выводится |
| `moneroid node list` / `node select [--dry-run]` | Проба нод из `nodes.txt` (RPC latency, sync, ZMQ-порт); `select` переписывает `host/rpc-port/zmq-port` в `p2pool.conf` |
| `moneroid config path`, `version` | Служебные |

`status` и `tui` подсвечены цветами Monero (оранжевый/белый; красный — только проблемы); в конвейере или при `NO_COLOR=1` вывод остаётся чистым текстом, смысл всегда есть в тексте.

Границы (намеренно): нет установщика бинарников, автообновлений, базы данных, web-UI, управления `monerod`, нескольких стеков, настройки ядра/MSR/hugepages. Полный перечень — в [ТЗ](docs/spec.md).

## Требования

- Linux x86_64, systemd ≥ 249 (проверено на 261), polkit — только для опционального управления без sudo.
- Установленные бинарники [P2Pool](https://github.com/SChernykh/p2pool/releases) и [XMRig](https://github.com/xmrig/xmrig/releases) (официальные релизы, проверка checksum/подписи).
- Monero-нода с открытыми RPC и ZMQ (`--zmq-pub`): своя или удалённая. Удалённая нода видит IP хоста и адрес выплат.
- Основной адрес кошелька Monero (начинается с `4`).

## Установка

Одной командой (Linux x86_64 с systemd; Debian 13, Ubuntu 22.04 и Arch проверены), от пользователя с sudo:

```sh
curl -fsSLO https://raw.githubusercontent.com/Plasmoid77/moneroid/main/install.sh
sh install.sh --wallet 4ВАШ_ОСНОВНОЙ_АДРЕС          # + --enable (автозапуск), --hugepages, --node HOST:RPC:ZMQ, --sidechain mini|nano|main
```

`--i2p` (вместе с `--node LAN_IP:RPC:ZMQ` или нодой `.b32.i2p`) ставит `i2pd`, создаёт серверный туннель и переводит p2p-трафик P2Pool в I2P — см. раздел в `docs/install.md`.

Скрипт скачивает официальные релизы P2Pool и XMRig и бинарник Moneroid, сверяет их с SHA256, закреплёнными в скрипте (подпись P2Pool проверена при закреплении), создаёт группу и двух системных пользователей, кладёт конфиги и unit-файлы, выбирает Monero-ноду пробой из `nodes.txt`, запускает службы и показывает `doctor`. Повторный запуск ничего не перезаписывает в `/etc/moneroid`. Удаление: `sh install.sh --uninstall [--purge]`.

Вручную, по шагам (или из исходников: `make build`, Go 1.27) — [docs/install.md](docs/install.md).

## Права

- `status`, `tui`, `doctor`, `logs` — обычный пользователь из группы `moneroid` (чтение Data API и конфигов) и `systemd-journal`/`wheel` для журнала.
- `start/stop/restart` — через `sudo` либо через опциональное polkit-правило `examples/polkit/50-moneroid.rules` (только два unit, только start/stop/restart, только группа `moneroid`).
- `node select` пишет в `p2pool.conf` → `sudo`.
- Каталог Data API содержит `local/console` с cookie TCP-консоли P2Pool: право чтения каталога равнозначно управлению P2Pool, поэтому группа одна.

## Как это устроено и как сопровождать

- [docs/architecture.md](docs/architecture.md) — поток данных, пакеты, инварианты, health-правила.
- [AGENTS.md](AGENTS.md) — правила изменений для людей и ИИ-агентов, известные ловушки.
- [CHANGELOG.md](CHANGELOG.md) · релизы — на GitHub, `sha256sum -c SHA256SUMS`.
- `make check` — gofmt, vet, тесты (без сети и systemctl), статическая сборка; то же делает CI.

## Документы

- [Установка](docs/install.md) · [Диагностика](docs/troubleshooting.md) · [Обновление upstream](docs/updating.md)
- [ТЗ v1](docs/spec.md) · [План реализации и история решений](docs/plan.md)
- [Контракты источников](docs/research/upstream-contracts.md) · [Отчёт стенда №1](docs/research/stand-report-1.md) · [№2](docs/research/stand-report-2.md) · [Приёмка v1](docs/research/acceptance-report-v1.md)
- [Исходный handoff](docs/history/handoff-v2.md) — первоначальное исследование; решения ТЗ имеют приоритет.

Лицензия — MIT. XMRig и P2Pool не входят в поставку и распространяются по своим лицензиям.
