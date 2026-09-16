# Изменения

## Не выпущено

- TUI: при выходе экран стирается явно (как у htop) — на терминалах без альтернативного экрана (консоль Linux, SOL, `screen` без `altscreen`) панель больше не остаётся на экране.
- `status`/`tui`: при `PERMISSION_DENIED` на Data API сообщение говорит, в какой группе не состоит процесс (после `usermod` нужен новый вход в систему).

## v0.4.2 — 2026-09-17

- `install.sh --i2p`: учитывает уже установленный i2pd (`tunnelsdir`, адреса консоли и SOCKS из `i2pd.conf`); `node select` выполняется только для свежесозданного конфига (существующий `host = 127.0.0.1` — локальная нода, её не трогаем).
- `docs/install.md`: удалённая нода + I2P через локальный проброс портов; reseed i2pd через прокси на фильтруемом аплинке.
- `install.sh`: кэш проверенных загрузок `/var/cache/moneroid` (`MONEROID_CACHE`) — установка без доступа к GitHub; ожидание туннеля i2pd до 120 с.

## v0.4.1 — 2026-09-16

- `install.sh --i2p`: P2Pool p2p через I2P (i2pd, серверный туннель, `.b32.i2p` в `p2pool.conf`); требует ноды в LAN/localhost или `.b32.i2p`.
- `node`/`doctor`: при `socks5` в `p2pool.conf` loopback/LAN-адреса ноды проверяются напрямую, как делает сам P2Pool (нужно для Tor/I2P с локальной нодой).
- `docs/install.md`: смена sidechain (очистка кэша), параметры main/mini/nano по исходникам.
- `install.sh`: по замечаниям Codex — строгая проверка `--wallet`/`--node` (base58, IPv6 `[addr]:RPC:ZMQ`), `restart` при повторном запуске, туннель i2pd по имени и по цепочке из конфига, `--now` для MSR-unit, `--uninstall` останавливает каждую службу отдельно и убирает туннель.

## v0.4.0 — 2026-09-16

- **Проект переименован в Moneroid.** Бинарник, модуль Go, unit'ы `moneroid-p2pool.service`/`moneroid-xmrig.service`, группа и пользователи `moneroid*`, каталоги `/etc/moneroid`, `/var/lib/moneroid`, `/run/moneroid-p2pool-api`, `api.id = moneroid-xmrig`. Переход с Monerizer — `docs/install.md`. Записи ниже про прошлые версии читаются с поправкой на старое имя.
- `install.sh` — установка одной командой от пользователя с sudo: официальные релизы P2Pool 4.18 и XMRig 6.26.0 с закреплёнными SHA256, кошелёк/sidechain/нода аргументами, `--enable`, `--hugepages` (sysctl + MSR-unit на Intel), `--uninstall [--purge]`; идемпотентно.

## v0.3.0 — 2026-09-16

- TUI: экран во всю высоту терминала в духе `htop` — заглавная и клавишная полосы, полосы hashrate/hugepages/синхронизации sidechain, оценка доли в пуле, строка выплат (перечитывается раз в минуту).
- Цвета Monero (оранжевый/белый, красный для проблем) в `tui` и в текстовом `status`; в конвейере и при `NO_COLOR` вывод остаётся чистым текстом; время в `status` — локальная зона.
- `github.com/charmbracelet/colorprofile` стал прямой зависимостью (уже был в графе через Bubble Tea).

## v0.2.0 — 2026-09-16

- `payouts [--json] [--since]` — выплаты из журнала P2Pool (строки «got a payout of» / «didn't get a payout»), сумма и число блоков без доли (PAY-01..03, схема `payouts-v1`).
- TUI: экран выплат по клавише `p` (последние 12, итог, `r` перечитать).
- Время в панели и в `payouts` — локальная зона системы; в JSON по-прежнему RFC 3339 UTC.
- `docs/troubleshooting.md`: раздел про фильтруемый аплинк и `socks5` в `p2pool.conf`.

## v0.1.0 — 2026-09-16

Первый релиз. `status [--json] [--check]`, `tui`, `start|stop|restart` (с авто `reset-failed`), `logs`, `doctor` (31 проверка), `node list|select` (проба `get_info` + заголовки блоков + ZMTP, через SOCKS5 при `socks5` в `p2pool.conf`), `config path`, `version`. Поставка: unit-файлы systemd, примеры конфигов, polkit-правило, MSR-unit (Intel), документация. Проверено на Arch (systemd 261), Debian 13 (257), Ubuntu 22.04 (249) с P2Pool 4.18 и XMRig 6.26.0.
