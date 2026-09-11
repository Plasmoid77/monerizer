# Установка Monerizer

Все команды — от root, кроме отмеченных. Ничего из этого Monerizer не делает сам: это осознанно ручная установка (ТЗ INSTALL-01).

## 1. Бинарники upstream

```sh
# P2Pool: tar.gz + sha256sums.txt.asc, ключ SChernykh 1FCA AB4D 3DC3 310D 16CB D508 C47F 82B5 4DA8 7ADF
# XMRig:  tar.gz + SHA256SUMS
install -o root -g root -m 0755 p2pool /usr/local/bin/p2pool
install -o root -g root -m 0755 xmrig  /usr/local/bin/xmrig
```

Другой путь → правьте `ExecStart=` в unit-файлах (не конфиг Monerizer).

## 2. Пользователи, группа, файлы

```sh
groupadd -r monerizer
useradd -r -g monerizer -s /usr/bin/nologin -d /var/lib/monerizer/p2pool -M monerizer-p2pool
useradd -r -g monerizer -s /usr/bin/nologin -d /var/lib/monerizer/xmrig  -M monerizer-xmrig
install -d -o root -g root -m 0755 /etc/monerizer
install -o root -g monerizer -m 0640 examples/p2pool.conf  /etc/monerizer/p2pool.conf
install -o root -g monerizer -m 0640 examples/xmrig.json   /etc/monerizer/xmrig.json
install -o root -g root      -m 0644 examples/monerizer.toml /etc/monerizer/monerizer.toml
install -o root -g root      -m 0644 examples/nodes.txt    /etc/monerizer/nodes.txt
install -o root -g root -m 0644 systemd/monerizer-p2pool.service systemd/monerizer-xmrig.service /etc/systemd/system/
install -o root -g root -m 0755 monerizer /usr/local/bin/monerizer
systemctl daemon-reload
usermod -aG monerizer ОПЕРАТОР      # чтение Data API; перелогиниться
```

Каталоги `/var/lib/monerizer/{p2pool,xmrig}` (0700) и `/run/monerizer-p2pool-api` (0750, группа `monerizer`) создаёт systemd при старте служб; вручную их создавать не нужно.

## 3. Нативные конфиги

`/etc/monerizer/p2pool.conf` — params-file P2Pool (`key = value`, boolean `1`):
- `wallet` — основной адрес (`4…`); без него P2Pool не стартует.
- `host`, `rpc-port`, `zmq-port` — Monero-нода. Либо вручную, либо `sudo monerizer node select` (пробует кандидатов из `nodes.txt`).
- `mini = 1` (по умолчанию в примере) / `nano = 1` / ничего для main.
- Остальные строки (`data-api`, `local-api`, `no-upnp`, `no-log-file`, `no-color`) — интеграционные, менять не нужно.

`/etc/monerizer/xmrig.json` — нативный конфиг XMRig. В примере: `autosave=false`, `watch=false` (иначе XMRig пытается перезаписать root-owned файл), `colors=false`, HTTP API на `127.0.0.1:18088` в restricted-режиме, `api.id = monerizer-xmrig`, pool `127.0.0.1:3333`. Потоки — `cpu.max-threads-hint` (100 = все). Донат — upstream-настройка `donate-level`, пример её не трогает.

Токен API (опционально): задайте `http.access-token` в `xmrig.json` и тот же текст в `/etc/monerizer/xmrig-api.token` (`root:monerizer 0640`), раскомментируйте `token_file` в `monerizer.toml`.

## 4. Первый запуск

```sh
monerizer doctor            # ожидаемо: warn UNIT_ENABLED (автозапуск не включён), warn XMRIG_HUGEPAGES
monerizer start             # или sudo monerizer start; P2Pool синхронизирует sidechain 2–5 минут
monerizer status
monerizer logs --follow p2pool   # «SideChain SYNCHRONIZED», без повторяющихся «ZMQReader disconnected»
systemctl enable monerizer-p2pool.service monerizer-xmrig.service   # автозапуск после загрузки
```

`start` возвращается сразу после запуска процессов; синхронизация и первые shares видны в `status`/`logs`, не в коде возврата.

## 5. Управление без sudo (опционально)

```sh
install -o root -g root -m 0644 examples/polkit/50-monerizer.rules /etc/polkit-1/rules.d/
```

Правило разрешает группе `monerizer` только `start/stop/restart` двух units. Другие units и другие действия по-прежнему требуют аутентификации.

## 6. Hugepages и MSR (административно, необязательно)

Службы работают под непривилегированными пользователями, поэтому XMRig не применяет свой MSR-mod и не может выделить hugepages сам. Хостовая настройка — на усмотрение администратора, например `sysctl vm.nr_hugepages=1280` (+ `/etc/sysctl.d/`), затем `monerizer restart xmrig`. Без этого hashrate ниже; Monerizer это только показывает.

## Откат

Обратный список: `systemctl disable --now` двух units, удалить unit-файлы, `daemon-reload`, удалить `/etc/monerizer`, `/var/lib/monerizer`, бинарники, пользователей и группу (см. `docs/research/stand-rollback.sh`).
