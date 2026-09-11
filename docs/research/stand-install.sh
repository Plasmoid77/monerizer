#!/bin/sh
# Установка стенда Monerizer (root). Аргументы: путь к распакованным p2pool и xmrig, репозиторий.
# Создаёт: группу monerizer; пользователей monerizer-p2pool, monerizer-xmrig;
# /usr/local/bin/{p2pool,xmrig}; /etc/monerizer/{p2pool.conf,xmrig.json}; два unit-файла.
# Каталоги /var/lib/monerizer/* и /run/monerizer-p2pool-api создаёт systemd (StateDirectory/RuntimeDirectory).
set -eu
P2POOL_BIN=$1; XMRIG_BIN=$2; REPO=$3
groupadd -r monerizer 2>/dev/null || true
useradd -r -g monerizer -s /usr/bin/nologin -d /var/lib/monerizer/p2pool -M monerizer-p2pool 2>/dev/null || true
useradd -r -g monerizer -s /usr/bin/nologin -d /var/lib/monerizer/xmrig -M monerizer-xmrig 2>/dev/null || true
install -o root -g root -m 0755 "$P2POOL_BIN" /usr/local/bin/p2pool
install -o root -g root -m 0755 "$XMRIG_BIN" /usr/local/bin/xmrig
install -d -o root -g root -m 0755 /etc/monerizer
install -o root -g monerizer -m 0640 "$REPO/examples/p2pool.conf" /etc/monerizer/p2pool.conf
install -o root -g monerizer -m 0640 "$REPO/examples/xmrig.json" /etc/monerizer/xmrig.json
install -o root -g root -m 0644 "$REPO/systemd/monerizer-p2pool.service" /etc/systemd/system/
install -o root -g root -m 0644 "$REPO/systemd/monerizer-xmrig.service" /etc/systemd/system/
install -o root -g root -m 0644 "$REPO/examples/polkit/50-monerizer.rules" /etc/polkit-1/rules.d/50-monerizer.rules
systemctl daemon-reload
echo "installed; now edit /etc/monerizer/p2pool.conf (wallet, host) and: systemctl start monerizer-p2pool"
