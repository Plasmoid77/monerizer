# Изменения

## v0.1.0 — 2026-09-16

Первый релиз. `status [--json] [--check]`, `tui`, `start|stop|restart` (с авто `reset-failed`), `logs`, `doctor` (31 проверка), `node list|select` (проба `get_info` + заголовки блоков + ZMTP, через SOCKS5 при `socks5` в `p2pool.conf`), `config path`, `version`. Поставка: unit-файлы systemd, примеры конфигов, polkit-правило, MSR-unit (Intel), документация. Проверено на Arch (systemd 261), Debian 13 (257), Ubuntu 22.04 (249) с P2Pool 4.18 и XMRig 6.26.0.
