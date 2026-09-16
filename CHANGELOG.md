# Изменения

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
