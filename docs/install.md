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

## 4a. Смена sidechain (mini ↔ nano ↔ main)

Одна строка в `p2pool.conf` (`mini = 1` / `nano = 1` / ничего для main), затем `sudo moneroid stop p2pool`, удалить `/var/lib/moneroid/p2pool/p2pool_peers.txt` и `p2pool.cache` (там пиры и блоки прежней цепочки — иначе P2Pool часами банит старых пиров и «синхронизирует» пустую собственную цепочку), `sudo moneroid start p2pool`. Параметры цепочек по исходникам 4.18: main и mini — шара каждые 10 с, окно PPLNS 2160 шар ≈ 6 ч; nano — шара каждые 30 с, окно 2160 шар ≈ 18 ч; минимальная сложность шары везде 100 000.

## 4b. P2Pool через I2P (опционально)

`install.sh --i2p --node …` делает то, что описано в `docs/I2P.MD` P2Pool: ставит `i2pd`, создаёт серверный туннель `moneroid-p2pool` на p2p-порт цепочки (37889 main / 37888 mini / 37890 nano; ключи в `/var/lib/i2pd/moneroid-p2pool.dat` — сохраните), берёт адрес `.b32.i2p` из веб-консоли i2pd и дописывает в `p2pool.conf`: `socks5 = 127.0.0.1:4447`, `socks5-proxy-type = i2p`, `no-dns = 1`, `i2p-address = …`, `p2p = 127.0.0.1:PORT`, `no-clearnet-p2p = 1`. Ограничение самого P2Pool: через прокси идут **все** непривычные соединения, а адреса loopback/LAN — напрямую, поэтому Monero-нода должна быть либо локальной/в LAN (`--node 192.168.x.x:RPC:ZMQ`), либо внутри I2P (`.b32.i2p` с RPC и ZMQ). `moneroid node`/`doctor` повторяют это правило. Адрес I2P публично привязывается к кошельку в найденных шарах — не используйте существующий адрес для других сервисов. XMRig I2P не касается (он ходит на `127.0.0.1:3333`). Первые i2p-пиры появляются через несколько минут после старта i2pd (построение туннелей); синхронизация sidechain через I2P заметно медленнее clearnet.

Уже установленный i2pd остаётся как есть: скрипт добавляет только свой файл `moneroid.conf` в каталог туннелей (`tunnelsdir` из `i2pd.conf`, по умолчанию `/etc/i2pd/tunnels.conf.d`), адреса консоли и SOCKS читает из `i2pd.conf`, а `restart i2pd` делает только когда файл туннеля создан или изменился.

На фильтруемом аплинке (см. `troubleshooting.md`) i2pd может не пройти reseed (`NetDbReq: No known routers, reseed seems to be totally failed` в `/var/lib/i2pd/../log`): добавьте в `/etc/i2pd/i2pd.conf` под `[reseed]` строку `proxy = socks://127.0.0.1:1080` (любой рабочий SOCKS/HTTP-прокси) и перезапустите i2pd — прокси нужен только для первого reseed. Сеть готова, когда консоль (`http://127.0.0.1:7070/`) показывает `Network status: OK`, обычно через 3–10 минут. Если же через 10+ минут `Tunnel creation success rate: 0%` — транспорты I2P (NTCP2/SSU2) тоже фильтруются, и I2P на этом аплинке не заработает без VPN; так на стенде Zeonux (2026-09-17).

Установка без доступа к GitHub: положите четыре файла релизов (`moneroid-vX-linux-amd64`, `moneroid-vX-extras.tar.gz`, `p2pool-vY-linux-x64.tar.gz`, `xmrig-Z-linux-static-x64.tar.gz`) в `/var/cache/moneroid/` (или `MONEROID_CACHE=…`) — скрипт возьмёт их оттуда и всё равно сверит SHA256; проверенные загрузки он сам складывает туда же.

Удалённая нода + I2P: пробросьте её RPC и ZMQ на локальный адрес (например, `ssh -N -L 127.0.0.1:18089:НОДА:18089 -L 127.0.0.1:18083:НОДА:18083 …` через доверенный хост, или `systemd-socket-proxyd`) и укажите `--node 127.0.0.1:18089:18083` — для P2Pool это локальный адрес, он пойдёт напрямую, пиры — через I2P. Так сделано на стенде Zeonux.

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
