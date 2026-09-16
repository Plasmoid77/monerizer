#!/bin/sh
# Установка стенда Moneroid (root). Аргументы: путь к распакованным p2pool и xmrig, репозиторий.
# Создаёт: группу moneroid; пользователей moneroid-p2pool, moneroid-xmrig;
# /usr/local/bin/{p2pool,xmrig}; /etc/moneroid/{p2pool.conf,xmrig.json}; два unit-файла.
# Каталоги /var/lib/moneroid/* и /run/moneroid-p2pool-api создаёт systemd (StateDirectory/RuntimeDirectory).
set -eu
P2POOL_BIN=$1; XMRIG_BIN=$2; REPO=$3
groupadd -r moneroid 2>/dev/null || true
useradd -r -g moneroid -s /usr/sbin/nologin -d /var/lib/moneroid/p2pool -M moneroid-p2pool 2>/dev/null || true
useradd -r -g moneroid -s /usr/sbin/nologin -d /var/lib/moneroid/xmrig -M moneroid-xmrig 2>/dev/null || true
install -o root -g root -m 0755 "$P2POOL_BIN" /usr/local/bin/p2pool
install -o root -g root -m 0755 "$XMRIG_BIN" /usr/local/bin/xmrig
install -d -o root -g root -m 0755 /etc/moneroid
install -o root -g moneroid -m 0640 "$REPO/examples/p2pool.conf" /etc/moneroid/p2pool.conf
install -o root -g moneroid -m 0640 "$REPO/examples/xmrig.json" /etc/moneroid/xmrig.json
install -o root -g root -m 0644 "$REPO/systemd/moneroid-p2pool.service" /etc/systemd/system/
install -o root -g root -m 0644 "$REPO/systemd/moneroid-xmrig.service" /etc/systemd/system/
[ -d /etc/polkit-1/rules.d ] && install -o root -g root -m 0644 "$REPO/examples/polkit/50-moneroid.rules" /etc/polkit-1/rules.d/50-moneroid.rules || echo "polkit rules.d absent (polkit < 0.106): use sudo for control"
systemctl daemon-reload
echo "installed; now edit /etc/moneroid/p2pool.conf (wallet, host) and: systemctl start moneroid-p2pool"
