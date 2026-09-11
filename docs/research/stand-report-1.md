# Отчёт стенда №1 — пара P2Pool + XMRig от пользователя

Дата: 2026-09-11. Хост: ноутбук владельца, Arch Linux, systemd 261, i7-8650U (4C/8T, L3 8 MiB), 15 GiB RAM, hugepages не настроены, MSR не применялся. Запуск от обычного пользователя в scratchpad-каталоге, без systemd и без root. Цель — убедиться, что связка работает, и получить реальные fixtures.

## 1. Бинарники

| Компонент | Ассет | Проверка |
|---|---|---|
| P2Pool v4.18 | `p2pool-v4.18-linux-x64.tar.gz` | `sha256sums.txt.asc` — подпись Good, ключ SChernykh `1FCA AB4D 3DC3 310D 16CB D508 C47F 82B5 4DA8 7ADF` (импорт из monero-project/gitian.sigs); SHA256 `893691…8a415` совпал |
| XMRig 6.26.0 | `xmrig-6.26.0-linux-static-x64.tar.gz` | `SHA256SUMS`: `fc6f8a…c9e5` совпал; `.sig` не проверялась (ключ не импортировался) |

## 2. Выбор ноды

Список clearnet-нод взят с monero.fail (`nodes.json`, 121 http-нода). Проба: JSON-RPC `get_info` + TCP-connect на 18083. Кандидатов с открытым 18083 и `synchronized=true` — 7.

| Нода | RPC | ZMQ | Результат в P2Pool |
|---|---|---|---|
| monerodice.pro | 18089, ping 83 ms | 18083 | ZMQ рвётся каждую секунду («ZMQ is not running, restarting it», 69 разрывов за 2,5 мин) — непригодна |
| xmr.support | 18081, ping 200 ms (P2Pool предупреждает «too high») | 18083 | ZMQ стабилен (0 разрывов за 6 мин), SideChain SYNCHRONIZED через ~3,5 мин |

Вывод для NODE-02: открытый TCP-порт ZMQ не доказывает работоспособность ZMQ; окончательная проверка — только по журналу P2Pool. Ранжирование по latency остаётся, но документация обязана говорить, что «лучшая» нода может оказаться непригодной, и как это увидеть.

## 3. P2Pool

- `--params-file` с 12 параметрами (`host`, `rpc-port`, `zmq-port`, `wallet`, `mini = 1`, `stratum`, `data-dir`, `data-api`, `local-api = 1`, `no-upnp = 1`, `no-color = 1`, `loglevel = 3`) принят без замечаний; формат `key = value`, boolean как `1`.
- stdin = `/dev/null`: работает. Лог: «ConsoleCommands tty or named pipe is not available» — P2Pool открывает **TCP-консоль на случайном порту localhost** и пишет порт и cookie в `local/console`. См. §6.
- SIGTERM → полная остановка за 0,41 s (два запуска). `TimeoutStopSec=30s` с запасом.
- Data API создаёт `local/{p2p,stratum,merge_mining,console}`, `network/stats`, `pool/stats`, `stats_mod`. Файлы без расширения.
- `local/p2p` обновляется раз в ~60 s; `local/stratum` — по событиям; `pool/stats` и `stats_mod` — при новых sidechain-блоках; `network/stats` — при новом mainchain-блоке.
- `local/stratum` содержит **`wallet`** (адрес выплат открытым текстом) и `workers` с `ip:port,…`. В контрактах §4 это не было учтено: адрес обязателен к исключению из вывода и fixtures (SEC-07).
- В `data-dir` пишется `p2pool.log` (дубликат консоли) и cache/peers. Для systemd: `no-log-file = 1`.
- RSS ≈ 2,8 GB (RandomX dataset), 10 исходящих P2P-соединений, `zmq_last_active` 0–8 s.

## 4. XMRig

- `--config` с `autosave=false`, `watch=false`, `colors=false`, `http` на 127.0.0.1:18088 restricted, `api.id=monerizer-xmrig`, `max-threads-hint=50` (4 потока): подключился к 127.0.0.1:3333, первый accepted share через < 75 s, ~880 H/s.
- **При stdout в обычный файл XMRig не пишет ничего** (0 байт даже с `--dry-run`). Под systemd (stdout — stream socket journald) лог идёт штатно: проверено в отчёте №2. `"syslog": true` не нужен (даёт дубли).
- `/2/summary`: `hashrate.total[2]` = null в первые 15 минут (подтверждает nullable); `hugepages = [0, 1172]`; `restricted=true`. `/2/backends[0].msr = false`.
- SIGTERM → остановка за 0,41 s.
- Оценки hashrate различаются: XMRig 10s ≈ 880 H/s, P2Pool `hashrate_15m` = 432 H/s (по shares) — иллюстрация MET-01.
- Температура CPU при 4 потоках ≈ 67 °C.

## 5. Fixtures

Сохранены в `testdata/` (см. `testdata/README.md`): адрес выплат, IP peers и console-cookie заменены.

## 6. Находки, требующие решения

1. **Console cookie в Data API.** При stdin без tty P2Pool публикует в `local/console` порт и cookie TCP-консоли, через которую доступны консольные команды P2Pool (включая, предположительно, `exit`). Группа наблюдателей с правом чтения каталога API получает управление процессом. Варианты: (a) объединить наблюдателей и операторов в одну группу и задокументировать; (b) проверить, отключает ли какой-либо параметр TCP-консоль (в `--help` такого нет); (c) StandardInput через FIFO — усложнение. Предложение: (a), минимально.
2. **Duplicate log file**: `no-log-file = 1` в поставляемом примере.
4. **`local/stratum.wallet`**: добавить в контракты как поле, которое никогда не выводится.

## 7. Что не проверено на этом шаге

Права `RuntimeDirectory`, unit-профиль (`ProtectSystem=strict` и RandomX), `Restart=on-failure`/start-limit, работа под отдельными пользователями, `systemctl show` без прав — всё это относится к запуску под systemd (следующий шаг).
