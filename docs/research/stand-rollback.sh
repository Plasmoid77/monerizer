#!/bin/sh
# Полный откат стенда Monerizer на ноутбуке (всё, что создаёт docs/research/stand-install.sh).
set -x
systemctl disable --now monerizer-xmrig.service monerizer-p2pool.service 2>/dev/null
rm -f /etc/systemd/system/monerizer-p2pool.service /etc/systemd/system/monerizer-xmrig.service
systemctl daemon-reload
rm -f /etc/polkit-1/rules.d/50-monerizer.rules
rm -rf /etc/monerizer /var/lib/monerizer /run/monerizer-p2pool-api
rm -f /usr/local/bin/p2pool /usr/local/bin/xmrig
userdel monerizer-p2pool 2>/dev/null; userdel monerizer-xmrig 2>/dev/null
gpasswd -d plasmoid monerizer 2>/dev/null
groupdel monerizer 2>/dev/null
# Возврат режима сна ноутбука
rm -f /etc/systemd/logind.conf.d/monerizer-stand.conf
systemctl unmask sleep.target suspend.target hibernate.target hybrid-sleep.target
# systemd-logind НЕ перезапускать на живой Wayland-сессии (это заморозило экран 2026-09-11); drop-in снимается при следующей загрузке
