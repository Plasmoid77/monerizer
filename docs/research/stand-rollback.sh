#!/bin/sh
# Полный откат стенда Moneroid на ноутбуке (всё, что создаёт docs/research/stand-install.sh).
set -x
systemctl disable --now moneroid-xmrig.service moneroid-p2pool.service 2>/dev/null
rm -f /etc/systemd/system/moneroid-p2pool.service /etc/systemd/system/moneroid-xmrig.service
systemctl daemon-reload
rm -f /etc/polkit-1/rules.d/50-moneroid.rules
rm -rf /etc/moneroid /var/lib/moneroid /run/moneroid-p2pool-api
rm -f /usr/local/bin/p2pool /usr/local/bin/xmrig
userdel moneroid-p2pool 2>/dev/null; userdel moneroid-xmrig 2>/dev/null
gpasswd -d plasmoid moneroid 2>/dev/null
groupdel moneroid 2>/dev/null
# Возврат режима сна ноутбука
rm -f /etc/systemd/logind.conf.d/moneroid-stand.conf
systemctl unmask sleep.target suspend.target hibernate.target hybrid-sleep.target
# systemd-logind НЕ перезапускать на живой Wayland-сессии (это заморозило экран 2026-09-11); drop-in снимается при следующей загрузке
