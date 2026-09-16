# Как устроен Moneroid

Один статический бинарник Go (~3,5 тыс. строк без тестов), два внешних модуля: `github.com/BurntSushi/toml` и `charm.land/bubbletea/v2` (только для `tui`). Никакого демона, состояния на диске и сети наружу, кроме loopback и — по запросу — проб нод.

## Поток данных

```text
                     ┌──────────────── systemctl show / start / stop / restart / reset-failed
                     │                 journalctl -u …
moneroid ───────────┼──────────────── GET http://127.0.0.1:18088/2/summary        (XMRig HTTP API)
  status/tui/doctor  │
                     └──────────────── read /run/moneroid-p2pool-api/{local/p2p,local/stratum,network/stats,pool/stats}
                                        (P2Pool Data API, файлы пишет P2Pool)

XMRig ──Stratum 127.0.0.1:3333──▶ P2Pool ──RPC+ZMQ──▶ Monero-нода     (Moneroid в этой цепочке не участвует)
```

Один вызов `Collector.Collect` (`internal/status/collect.go`) опрашивает три источника **параллельно и независимо** с общим deadline 3 s (1 s на HTTP/systemctl). Отказ одного источника не трогает остальные; результат всегда есть — проблемы записываются в `sources` и `health.issues`, а не в ошибку.

## Пакеты

| Пакет | Отвечает за | Не отвечает за |
|---|---|---|
| `cmd/moneroid` | разбор аргументов, вывод текста/JSON, коды завершения (0/1/2/3/4/130) | логику данных |
| `internal/config` | `moneroid.toml`: строгая схема, loopback-only `api_url`, абсолютные пути | чтение конфигов P2Pool/XMRig |
| `internal/systemd` | `systemctl show` → свойства; `systemctl VERB -- UNIT…`; аргументы `journalctl` | разбор `systemctl status`, PID-менеджмент |
| `internal/xmrig` | `GET /2/summary` → нормализованные nullable-поля | POST/PUT, конфиг майнера |
| `internal/p2pool` | чтение четырёх файлов Data API (обычные файлы, ≤ 1 MiB, retry 50 ms) | HTTP-сервер, консенсус |
| `internal/jsonx` | толерантный JSON: неизвестные поля игнорируются, неверный тип портит только своё поле | — |
| `internal/status` | модель `Snapshot`, сбор, свежесть, привязка данных к сессии процесса, правила health | хранение истории |
| `internal/doctor` | 31 read-only проверка над тем же `Snapshot` + journal/token/clock/node | автопочинку |
| `internal/node` | проба нод (`get_info`, `get_block_headers_range`, ZMTP-рукопожатие, при необходимости через SOCKS5), замена трёх ключей в `p2pool.conf` | выбор ноды «на лету» |
| `internal/payouts` | строки «got a payout of» из журнала через `journalctl -g` (единственное чтение логов) | баланс, кошелёк |
| `internal/tui` | Bubble Tea-панель над тем же `Collector` во всю высоту терминала (`htop`-стиль); меню control; `journalctl -f` через `tea.ExecProcess`; сводка выплат раз в минуту | собственный сбор данных |
| `internal/ansi` | несколько SGR-последовательностей палитры Monero (оранжевый/белый, красный для проблем) и `Bar`; понижение цвета и `NO_COLOR` делает `colorprofile` (в TUI — сам Bubble Tea) | цвет как единственный носитель смысла |

Интерфейсы введены только на границах с внешним миром: `systemd.Runner` (запуск команд), `http.Client`, файловая система, часы (`Now`, `Monotonic`). Поэтому весь `internal/status` тестируется на fixtures без systemctl и сети (`testdata/`).

## Ключевые инварианты

1. **Панель никогда не владеет майнингом.** `q`, Ctrl-C, Esc, авария TUI не вызывают stop. Службы живут в systemd независимо от Moneroid (`SYS-01`).
2. **Единственный владелец каждого параметра.** Адрес выплат, нода, sidechain — `p2pool.conf`; потоки, API — `xmrig.json`; пути бинарников — unit-файлы; что наблюдать — `moneroid.toml`. Moneroid пишет в чужой конфиг ровно в одном месте: `node select` меняет `host/rpc-port/zmq-port`.
3. **`null` ≠ `0`.** Отсутствующее знание — `null`, измеренный ноль — `0`; это видно в тексте, JSON и спарклайне.
4. **Свежесть и сессия.** У файла есть возраст (mtime) и доказательство принадлежности текущему процессу (`RuntimeDirectory` + сравнение uptime); у HTTP — совпадение `api.id` и uptime с `ExecMainStartTimestampMonotonic`. Данные прошлого запуска не подтверждают health.
5. **Следствия не выдаются за причины.** «API недоступен» ≠ «процесс упал», «служба active» ≠ «нода синхронизирована»; `sync_state` всегда `unknown`.
6. **Health по правилам H-02/H-03** (`internal/status/health.go`): unit not-found/failed → `degraded`; обе inactive → `stopped`; activating → `starting`; одна из двух → `degraded`; обе active → `ok` только при свежем `local/p2p` текущего процесса, XMRig connected с hashrate > 0, peers > 0, sidechain не отстаёт от peers, ZMQ ≤ 600 s.

Нормативный документ — [ТЗ](spec.md); откуда взято каждое поле и единица — [контракты источников](research/upstream-contracts.md); что и как проверялось — [отчёт приёмки](research/acceptance-report-v1.md).

## Схемы вывода

`status --json` и `doctor --json` версионируются полем `schema_version` (сейчас 1): поля только добавляются. Описание — [docs/schema](schema/).

## Сборка и проверка

```sh
make check      # gofmt, go vet, go test -race, статическая сборка linux/amd64
make build      # ./moneroid с версией из git describe
```

Тесты не ходят в сеть и не вызывают systemctl; fixtures в `testdata/` — реальные ответы P2Pool 4.18 и XMRig 6.26.0 с заменёнными адресами.
