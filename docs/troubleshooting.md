# Диагностика

Первый шаг всегда `monerizer doctor`, второй — `monerizer logs <target>`. Monerizer показывает следствия и коды причин, но не угадывает причину за upstream (ТЗ DOC-04).

| Симптом / код | Что означает | Что делать |
|---|---|---|
| `UNIT_NOT_FOUND` | Unit не установлен или не сделан `daemon-reload` | Установить unit-файл, `systemctl daemon-reload` |
| `UNIT_FAILED`, `Result=start-limit-hit` | Служба падала 5 раз за минуту | `monerizer logs`; чаще всего неверный `wallet` или недоступная нода; после исправления `systemctl reset-failed` не нужен — `monerizer start` |
| `UNIT_MASKED` | Unit замаскирован | `systemctl unmask` |
| `UNIT_NOT_ACTIVE` (health degraded) | Работает только одна служба | `monerizer start <вторая>` |
| `SOURCE_UNAVAILABLE` для `xmrig_summary`, служба active | HTTP API не отвечает | В `xmrig.json`: `http.enabled=true`, `host=127.0.0.1`, порт как в `api_url` |
| `PERMISSION_DENIED` для `xmrig_summary` | 401/403 | Токен в `token_file` не совпадает с `http.access-token`; после смены — `restart xmrig` |
| `SOURCE_ID_MISMATCH` | На порту другой XMRig | `api.id` в `xmrig.json` = `expected_id` в `monerizer.toml` |
| `PERMISSION_DENIED` для `p2pool_*` | Оператор не в группе `monerizer` или не перелогинился | `usermod -aG monerizer`, новый вход |
| `SOURCE_UNAVAILABLE` для `p2pool_p2p`, служба active | Файл ещё не записан (первая запись через ~60 s) или `data-api`/`local-api` не заданы | Подождать минуту; проверить `p2pool.conf` |
| `SOURCE_STALE` (`local/p2p` старше 180 s) | P2Pool завис или не пишет API | `monerizer logs p2pool`, `restart p2pool` |
| `SOURCE_NOT_CURRENT_SESSION` | Файлы/API от другого процесса | Проверить, что нет второго P2Pool/XMRig; для сторонних units — `RuntimeDirectory` |
| `ZMQ_ACTIVITY_OLD` | >600 s без ZMQ-событий | Нода без `--zmq-pub`, неверный `zmq-port` или нестабильный ZMQ у удалённой ноды (в журнале повторяющиеся `ZMQReader disconnected`) → `monerizer node list`, выбрать другую |
| `SIDECHAIN_BEHIND` | Локальная sidechain ниже высоты peers | Первые ~10 минут после старта — норма (P2Pool качает и проверяет окно, в журнале растут `verified block`); если не догоняет — `monerizer logs p2pool`, нода/сеть |
| `P2P_NO_CONNECTIONS` | Нет peers | Исходящие соединения на порт sidechain (mini 37888, main 37889, nano 37890) закрыты |
| `HASHRATE_ZERO`, `XMRIG_DISCONNECTED` | XMRig не получает задания | P2Pool ещё синхронизируется (`SideChain SYNCHRONIZED` в журнале) или `pools[0].url` ≠ `stratum` P2Pool |
| `CLOCK_UNCERTAIN` | mtime файла в будущем | Часы/NTP |
| `FIELD_INVALID` | Upstream изменил тип поля | Показатель null, остальное работает; сообщить в issue с версией upstream |
| XMRig: `FAILED TO APPLY MSR MOD`, hugepages 0% | Ожидаемо под непривилегированным пользователем | См. install.md §6 |
| hugepages зарезервированы, но у XMRig < 100% | Страницы забрал P2Pool (свой RandomX dataset) или их меньше 1040 свободных | `light-mode = 1` в `p2pool.conf`, `nr_hugepages ≥ 1536`, перезагрузка |
| `monerizer start`: код 3 | Нет прав на systemd | `sudo` или polkit-правило |
| `monerizer start`: код 4 | systemctl не ответил за 90 s | Задание могло продолжиться: `monerizer status` |
| `tui`: «needs an interactive terminal» | Нет TTY | Использовать `status` |

Что **не** является ошибкой: `sidechain shares 0 found` (share на mini находится редко), старые `rejected` без роста, `UNIT_DISABLED` (автозапуск просто не включён), `hashrate_15m = null` первые 15 минут.

## Аплинк фильтрует трафик к нодам

Симптом (Zeonux, 2026-09-14): `get_info` проходит, а ответы больше ~15 KB (`get_block_headers_range`, sync sidechain) виснут ко всем нодам независимо от порта и TLS; `node list` показывает `headers: context deadline exceeded`, P2Pool крутится в «Couldn't download block headers» или майнит собственную цепочку (`SIDECHAIN_BEHIND`).

P2Pool умеет ходить через SOCKS5 (P2P, RPC и ZMQ): в `p2pool.conf`

```
socks5 = 127.0.0.1:1080
socks5-proxy-type = plain
```

Источник прокси — любой хост с чистым интернетом, например `ssh -N -D 127.0.0.1:1080 user@host` (ключ на той стороне ограничьте `restrict,port-forwarding`), или VPN-клиент на самом хосте. `monerizer node list/select` и `doctor` при заданном `socks5` пробуют ноды через тот же прокси. Кэш изолированной цепочки после исправления сети лучше удалить: `monerizer stop p2pool; rm /var/lib/monerizer/p2pool/p2pool.cache; monerizer start p2pool`.
