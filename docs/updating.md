# Обновление

## Upstream (P2Pool, XMRig)

Moneroid менять не нужно, пока не меняются интерфейсы: `systemctl show`, `GET /2/summary`, четыре файла Data API.

```sh
# проверить checksum/подпись нового релиза, затем:
install -o root -g root -m 0755 p2pool /usr/local/bin/p2pool && moneroid restart p2pool
install -o root -g root -m 0755 xmrig  /usr/local/bin/xmrig  && moneroid restart xmrig
moneroid doctor
```

Если путь бинарника изменился — правится `ExecStart=` в unit (drop-in `systemctl edit`), затем `daemon-reload`. Если upstream переименовал/удалил поле — в `status` появится `FIELD_INVALID` или `null` у соответствующей метрики, остальное продолжит работать.

## Нода

`sudo moneroid node select` (или правка `host/rpc-port/zmq-port` вручную), затем `moneroid restart p2pool`. Список кандидатов `/etc/moneroid/nodes.txt` — ваш файл: ноды исчезают, дополняйте его.

## Moneroid

```sh
make build && install -o root -g root -m 0755 moneroid /usr/local/bin/moneroid
```

Службы не зависят от бинарника Moneroid: обновление или удаление не прерывает майнинг. Схемы `status --json` и `doctor --json` версионируются полем `schema_version`; в пределах версии поля только добавляются.
