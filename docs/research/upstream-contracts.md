# Контракты источников Monerizer v1

Дата проверки: 2026-09-11. Приложение к [ТЗ](../spec.md).
Проверены публичные исходники; runtime-запусков и реальных captures в этой работе не было.

## 1. Зафиксированные версии

| Компонент | Версия | Commit | Дата релиза |
|---|---|---|---|
| P2Pool | [v4.17.1](https://github.com/SChernykh/p2pool/releases/tag/v4.17.1) | `9b7395b8c97e7705a138dda2f4edef6b91eebe5f` | 2026-06-28 |
| XMRig | [v6.26.0](https://github.com/xmrig/xmrig/releases/tag/v6.26.0) | `b2ca72480c58d197e18c885d9fc1a0c8d517e60a` | 2026-03-28 |

Это исходная проверяемая совместимость, не обещание поддержки всех версий и не рекомендация фиксировать майнеры навсегда.

Повторная проверка 2026-09-11: актуальный релиз P2Pool — [v4.18](https://github.com/SChernykh/p2pool/releases/tag/v4.18) (2026-08-05); XMRig 6.26.0 остаётся актуальным. Ссылки на строки исходников ниже относятся к 4.17.1; на стенде этапа 1 таблицы §2–5 сверяются с фактическим выводом 4.18. Параметры XMRig `watch` (перечитывание конфига с диска) и `autosave` (запись конфига при изменениях; наличие и default подтверждаются на стенде) в поставляемом примере отключаются.

## 2. XMRig: GET /2/summary

| Поле snapshot в `xmrig` | Upstream | Единица / преобразование |
|---|---|---|
| `version` | `version` | строка |
| `id` | `id` | строка; сверяется с expected_id |
| `uptime_seconds` | `uptime` | секунды |
| `hashrate_10s_hs` | `hashrate.total[0]` | H/s |
| `hashrate_60s_hs` | `hashrate.total[1]` | H/s |
| `hashrate_15m_hs` | `hashrate.total[2]` | H/s |
| `hugepages_allocated` | `hugepages[0]` | количество |
| `hugepages_total` | `hugepages[1]` | количество |
| `hugepages_percent` | предыдущие поля | `100 * allocated / total`, при total=0 — null |

Окна hashrate могут быть null. Основные сериализаторы: [Miner.cpp](https://github.com/xmrig/xmrig/blob/b2ca72480c58d197e18c885d9fc1a0c8d517e60a/src/core/Miner.cpp#L145).

| Поле snapshot в `xmrig` | Upstream | Значение |
|---|---|---|
| `accepted` | `results.shares_good` | принятые submissions |
| `rejected` | `results.shares_total - results.shares_good` | отклонённые; при total < good — null и ошибка поля |
| `pool` | `connection.pool` | адрес подключения, очищенный перед выводом |
| `connection_uptime_ms` | `connection.uptime_ms` | миллисекунды |
| `connected` | `connection.uptime_ms > 0` | вычисленный признак текущей сессии; неизвестное поле → null |

При разрыве uptime становится 0, а имя pool может сохраняться. Поэтому непустой pool не доказывает подключение. Общий счётчик работы и submissions не описывает P2Pool payouts. [NetworkState.cpp: ответ](https://github.com/xmrig/xmrig/blob/b2ca72480c58d197e18c885d9fc1a0c8d517e60a/src/base/net/stratum/NetworkState.cpp#L117), [сброс состояния](https://github.com/xmrig/xmrig/blob/b2ca72480c58d197e18c885d9fc1a0c8d517e60a/src/base/net/stratum/NetworkState.cpp#L323).

`msr` есть в CPU-объекте `/2/backends`, но v1 этот endpoint не опрашивает. MSR остаётся unknown и не влияет на health. [CpuBackend.cpp](https://github.com/xmrig/xmrig/blob/b2ca72480c58d197e18c885d9fc1a0c8d517e60a/src/backend/cpu/CpuBackend.cpp#L413)

API требует включения в нативном конфиге; базовые настройки: loopback, `restricted=true`. Настроенный токен передаётся как Bearer. Приложение не использует POST/PUT и не читает endpoint конфигурации. [Официальная документация API](https://xmrig.com/docs/miner/api), [настройки HTTP](https://xmrig.com/docs/miner/config/http)

## 3. P2Pool: периодический local/p2p

| Поле snapshot в `p2pool` | Upstream | Единица |
|---|---|---|
| `p2p_connections` | `connections` | количество |
| `p2p_incoming_connections` | `incoming_connections` | количество |
| `peer_list_size` | `peer_list_size` | количество известных peers, не обязательно подключённых |
| `uptime_seconds` | `uptime` | секунды на момент записи |
| `zmq_age_at_write_seconds` | `zmq_last_active` | секунд с последней ZMQ-активности на момент записи |
| `zmq_age_seconds` | предыдущая величина + возраст файла | оценка текущего возраста, только при корректных часах |

Upstream также выдаёт `peers` как массив строк `dir,ping,?,version,height,ip:port`; v1 берёт из них только максимальную высоту (`peer_max_height`) для `SIDECHAIN_BEHIND`. Запись происходит номинально раз в 60 секунд. Наличие connections не доказывает согласованность sidechain, полную синхронизацию или пригодность каждого peer. [p2p_server.cpp](https://github.com/SChernykh/p2pool/blob/9b7395b8c97e7705a138dda2f4edef6b91eebe5f/src/p2p_server.cpp#L1773)

## 4. P2Pool: событийный local/stratum

| Поле snapshot в `p2pool` | Upstream | Значение |
|---|---|---|
| `hashrate_15m_hs`, `hashrate_1h_hs`, `hashrate_24h_hs` | `hashrate_15m`, `hashrate_1h`, `hashrate_24h` | оценка H/s |
| `stratum_shares` | `total_stratum_shares` | принятые результаты целевой Stratum difficulty |
| `sidechain_shares_found` | `shares_found` | успешно переданные локальные shares sidechain difficulty |
| `sidechain_shares_failed` | `shares_failed` | неуспешная передача соответствующих shares |
| `average_effort_percent`, `current_effort_percent` | `average_effort`, `current_effort` | проценты |
| `last_share_found_at` | `last_share_found_time` | UNIX seconds → RFC3339; 0 → null |
| `stratum_connections` | `connections` | число подключений |

`workers` — массив строк, не структурированные worker-объекты; в v1 не разбирается. Стенд 2026-09-11 (4.18): файл также содержит `wallet` — адрес выплат открытым текстом; поле не читается, не выводится и заменяется в fixtures (SEC-07). Дополнительно есть `total_hashes`, `block_reward_share_percent`. Эти данные записываются по событиям с ограничением частоты: не чаще одного раза в 20 секунд. Гарантии новой записи каждые 20 секунд нет. [Сериализатор и ограничение частоты](https://github.com/SChernykh/p2pool/blob/9b7395b8c97e7705a138dda2f4edef6b91eebe5f/src/stratum_server.cpp#L1574)

## 5. Остальные файлы

`network/stats`: snapshot `network_height ← height`, `network_difficulty ← difficulty`. `timestamp` — время блока, а не время обновления API; для свежести файла его не использовать. Поле reward в atomic units не нужно v1.

`pool/stats`: snapshot `pool_hashrate_hs ← pool_statistics.hashRate`, `sidechain_height ← pool_statistics.sidechainHeight`, `sidechain_difficulty ← pool_statistics.sidechainDifficulty`. `miners` не является точным числом активных майнеров/локальных workers; v1 его не показывает. Название main/mini/nano не экспортируется, `sidechain` в snapshot — null. Оба файла обновляются по событиям. [p2pool.cpp](https://github.com/SChernykh/p2pool/blob/9b7395b8c97e7705a138dda2f4edef6b91eebe5f/src/p2pool.cpp#L1905)

Стенд 2026-09-11 (4.18): дополнительно создаются `stats_mod`, `local/merge_mining` и `local/console`; последний при stdin без tty содержит порт и cookie TCP-консоли P2Pool — v1 его не читает, а право чтения каталога API означает доступ к консоли (см. отчёт стенда №1, §6).

Файлы не имеют расширения. Проверенная реализация пишет временный файл, закрывает и переименовывает его в окончательный. Читаются только окончательные имена; согласованность нескольких файлов в одном атомарном snapshot не гарантируется. [p2pool_api.cpp](https://github.com/SChernykh/p2pool/blob/9b7395b8c97e7705a138dda2f4edef6b91eebe5f/src/p2pool_api.cpp#L153)

## 6. Политика свежести Monerizer

Следующие пороги — решения продукта, не нормативы upstream.

| Источник | Правило |
|---|---|
| `systemd_p2pool`, `systemd_xmrig`, `xmrig_summary` | Успех текущего сбора — ok; после ошибки сохранённое значение не подтверждает health |
| `p2pool_p2p` | mtime не старше 180 s, принадлежность текущему запуску не опровергнута; иначе stale |
| `p2pool_stratum`, `p2pool_network`, `p2pool_pool` | Событийные данные: всегда показывать возраст; возраст сам по себе не повод объявлять источник неисправным |

Значение ZMQ age > 600 s создаёт warning `ZMQ_ACTIVITY_OLD`: долго не наблюдалась активность, причина неизвестна. Это не доказательство разрыва ZMQ или рассинхронизации. Порог учитывает возраст файла.

Возраст mtime более чем на 5 s в будущем → `CLOCK_UNCERTAIN`, возраст null. При перезапуске штатный systemd RuntimeDirectory удаляет прежние API-файлы. Для стороннего каталога без доказанной связи с unit состояние — unknown. Проверка uptime с допуском 5 s выявляет противоречие, но не является точной идентификацией. Изменение wall clock/несовместимость часов делает такую проверку unknown; не обновлять возраст чтением того же файла.

Источник имеет `state`: `ok|stale|unavailable|invalid|permission_denied|unknown`; вместе с ним сохраняются времена из DATA-06. `not_current_session` передаётся как error_code, а не дополнительный boolean готовности. Событийные значения подписываются «оценка на момент записи».

## 7. Структурные ограничения и причины

В snapshot все числовые метрики из таблиц nullable; boolean `connected` nullable. Для v1 не нужны поля MSR, wallet, payout, worker-list или версия P2Pool: их отсутствие является границей функции. `sidechain=null` и `sync_state="unknown"` заданы явно. Версию P2Pool пользователь видит при ручной проверке бинарника/журнала.

Обязательные причины: `SOURCE_UNAVAILABLE`, `SOURCE_STALE`, `SOURCE_INVALID`, `FIELD_INVALID`, `PERMISSION_DENIED`, `UNIT_NOT_FOUND`, `UNIT_FAILED`, `UNIT_MASKED`, `UNIT_NOT_ACTIVE`, `UNIT_DISABLED`, `SOURCE_SESSION_UNKNOWN`, `SOURCE_NOT_CURRENT_SESSION` (доказанное несоответствие uptime), `SOURCE_ID_MISMATCH`, `CLOCK_UNCERTAIN`, `XMRIG_DISCONNECTED`, `HASHRATE_ZERO`, `P2P_NO_CONNECTIONS`, `SIDECHAIN_BEHIND`, `ZMQ_ACTIVITY_OLD`, `NATIVE_CONFIG_NOT_VALIDATED`, `DEPENDENCIES_DIFFER`. Отсутствующее поле отражается null; `FIELD_INVALID` нужен для неверного типа/значения, не для каждого штатно пропущенного поля.

Doctor JSON: `schema_version`, `collected_at`, `checks[]`, `summary`. Каждый check содержит `code`, `component`, `result` (`pass|warn|fail|skip`), `message`, `remedy` (строка либо null). Summary содержит counts `pass`, `warn`, `fail`, `skip`.

## 8. Границы конфигов и сервиса

P2Pool поддерживает `--params-file` без других CLI-параметров; boolean-пример — `mini = 1`. `--no-console-log` отключает console logging, а не обработчик stdin; выдуманного `--no-console` нет. Нельзя отключать console log в примере, рассчитанном на journald. [COMMAND_LINE.MD](https://github.com/SChernykh/p2pool/blob/9b7395b8c97e7705a138dda2f4edef6b91eebe5f/docs/COMMAND_LINE.MD)

Обработчик EOF не вызывает остановку pool; окончательная работа stdin=null и корректная остановка всё равно проверяются на стенде. [console_commands.cpp](https://github.com/SChernykh/p2pool/blob/9b7395b8c97e7705a138dda2f4edef6b91eebe5f/src/console_commands.cpp#L441)

XMRig пишет консольный лог только если stdout — tty, pipe или socket (libuv); при перенаправлении в обычный файл вывод пуст (стенд 2026-09-11: 0 байт). Под systemd (`StandardOutput=journal`, stream socket) лог попадает в journald штатно; `"syslog": true` не нужен и даёт дубли строк. XMRig допускает комментарии и trailing commas в нативном JSON. Проверка его конфига строгим `encoding/json` дала бы ложный отказ, поэтому Monerizer её не выполняет. [Json_unix.cpp](https://github.com/xmrig/xmrig/blob/b2ca72480c58d197e18c885d9fc1a0c8d517e60a/src/base/io/json/Json_unix.cpp#L29)
