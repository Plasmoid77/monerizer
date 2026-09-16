# Установка Moneroid

Быстрый путь — `install.sh` из корня репозитория (см. README): он выполняет ровно шаги 1–4 ниже с закреплёнными версиями upstream и проверкой SHA256, идемпотентно, от пользователя с sudo. Ниже — те же шаги вручную, от root, кроме отмеченных; сам бинарник Moneroid ничего из этого не делает (ТЗ INSTALL-01).

## Переход с Monerizer (до v0.4.0)

Проект переименован; изменились имена unit'ов, группы, пользователей и путей (`monerizer` → `moneroid`). Переход: `moneroid stop` старой установки (`monerizer stop`), затем `sh install.sh --wallet …` (или шаги ниже), перенос своих правок из `/etc/monerizer/*` в `/etc/moneroid/*` (пути внутри тоже переименовать), при желании `/var/lib/monerizer/p2pool` → `/var/lib/moneroid/p2pool` (кэш P2Pool, ускоряет первую синхронизацию; `chown -R moneroid-p2pool:moneroid`), и удаление старого: `systemctl disable --now monerizer-p2pool monerizer-xmrig`, `rm /etc/systemd/system/monerizer-*.service /etc/polkit-1/rules.d/50-monerizer.rules /usr/local/bin/monerizer`, `userdel monerizer-p2pool monerizer-xmrig`, `groupdel monerizer`.

## 1. Бинарники upstream

```sh
# P2Pool: tar.gz + sha256sums.txt.asc, ключ SChernykh 1FCA AB4D 3DC3 310D 16CB D508 C47F 82B5 4DA8 7ADF
# XMRig:  tar.gz + SHA256SUMS
install -o root -g root -m 0755 p2pool /usr/local/bin/p2pool
install -o root -g root -m 0755 xmrig  /usr/local/bin/xmrig
```

Другой путь → правьте `ExecStart=` в unit-файлах (не конфиг Moneroid).

## 2. Пользователи, группа, файлы

```sh
groupadd -r moneroid
useradd -r -g moneroid -s /usr/sbin/nologin -d /var/lib/moneroid/p2pool -M moneroid-p2pool
useradd -r -g moneroid -s /usr/sbin/nologin -d /var/lib/moneroid/xmrig  -M moneroid-xmrig
install -d -o root -g root -m 0755 /etc/moneroid
install -o root -g moneroid -m 0640 examples/p2pool.conf  /etc/moneroid/p2pool.conf
install -o root -g moneroid -m 0640 examples/xmrig.json   /etc/moneroid/xmrig.json
install -o root -g root      -m 0644 examples/moneroid.toml /etc/moneroid/moneroid.toml
install -o root -g root      -m 0644 examples/nodes.txt    /etc/moneroid/nodes.txt
install -o root -g root -m 0644 systemd/moneroid-p2pool.service systemd/moneroid-xmrig.service /etc/systemd/system/
install -o root -g root -m 0755 moneroid /usr/local/bin/moneroid
systemctl daemon-reload
usermod -aG moneroid ОПЕРАТОР      # чтение Data API; перелогиниться
```

Каталоги `/var/lib/moneroid/{p2pool,xmrig}` (0700) и `/run/moneroid-p2pool-api` (0750, группа `moneroid`) создаёт systemd при старте служб; вручную их создавать не нужно.

## 3. Нативные конфиги

`/etc/moneroid/p2pool.conf` — params-file P2Pool (`key = value`, boolean `1`):
- `wallet` — основной адрес (`4…`); без него P2Pool не стартует.
- `host`, `rpc-port`, `zmq-port` — Monero-нода. Либо вручную, либо `sudo moneroid node select` (пробует кандидатов из `nodes.txt`).
- `mini = 1` (по умолчанию в примере) / `nano = 1` / ничего для main.
- Остальные строки (`data-api`, `local-api`, `no-upnp`, `no-log-file`, `no-color`) — интеграционные, менять не нужно.

`/etc/moneroid/xmrig.json` — нативный конфиг XMRig. В примере: `autosave=false`, `watch=false` (иначе XMRig пытается перезаписать root-owned файл), `colors=false`, HTTP API на `127.0.0.1:18088` в restricted-режиме, `api.id = moneroid-xmrig`, pool `127.0.0.1:3333`. Потоки — `cpu.max-threads-hint` (100 = все). Донат — upstream-настройка `donate-level`, пример её не трогает.

Токен API (опционально): задайте `http.access-token` в `xmrig.json` и тот же текст в `/etc/moneroid/xmrig-api.token` (`root:moneroid 0640`), раскомментируйте `token_file` в `moneroid.toml`.

## 4. Первый запуск

```sh
moneroid doctor            # ожидаемо: warn UNIT_ENABLED (автозапуск не включён), warn XMRIG_HUGEPAGES
moneroid start             # или sudo moneroid start; P2Pool синхронизирует sidechain 2–5 минут
moneroid status
moneroid logs --follow p2pool   # «SideChain SYNCHRONIZED», без повторяющихся «ZMQReader disconnected»
systemctl enable moneroid-p2pool.service moneroid-xmrig.service   # автозапуск после загрузки
```

`start` возвращается сразу после запуска процессов; синхронизация и первые shares видны в `status`/`logs`, не в коде возврата.

## 5. Управление без sudo (опционально)

```sh
install -o root -g root -m 0644 examples/polkit/50-moneroid.rules /etc/polkit-1/rules.d/
```

Правило разрешает группе `moneroid` только `start/stop/restart` (и `reset-failed`) двух units. Другие units и другие действия по-прежнему требуют аутентификации. Нужен polkit ≥ 0.106 (JS-правила, каталог `/etc/polkit-1/rules.d`): Debian 12+/Arch — да; Ubuntu 22.04 (polkit 0.105) — нет, там управление через `sudo`.

## 6. Hugepages и MSR (административно, необязательно)

Службы работают под непривилегированными пользователями, поэтому XMRig не применяет свой MSR-mod и не может выделить hugepages сам. Хостовая настройка — на усмотрение администратора:

```sh
printf 'vm.nr_hugepages = 1536\n' > /etc/sysctl.d/90-moneroid-hugepages.conf   # 3 GiB: XMRig ~1172 страниц + P2Pool light ~270
```

**MSR (только Intel, опционально).** Под непривилегированным пользователем XMRig пишет `FAILED TO APPLY MSR MOD`. Тот же эффект даёт oneshot-unit `examples/moneroid-msr.service` (нужен `msr-tools`): `wrmsr -a 0x1a4 0xf` до старта XMRig, откат в `ExecStop`. Измерено на Zeonux (2×Xeon E5-2683 v4): 13,8 → 14,4 kH/s (+4,5 %). Для AMD значения другие — не используйте этот unit.

Hugepages применяются надёжно только при загрузке (на работающей системе память фрагментирована, `sysctl -w` выделит лишь часть). P2Pool сам держит RandomX dataset (2 GB и ~1200 hugepages); в примере `p2pool.conf` включён `light-mode = 1`, чтобы страницы достались XMRig. Проверка: `moneroid doctor` → `XMRIG_HUGEPAGES pass`, в журнале XMRig `huge pages 100%`. Без этого hashrate ниже; Moneroid это только показывает.

## Откат

Обратный список: `systemctl disable --now` двух units, удалить unit-файлы, `daemon-reload`, удалить `/etc/moneroid`, `/var/lib/moneroid`, бинарники, пользователей и группу (см. `docs/research/stand-rollback.sh`).
