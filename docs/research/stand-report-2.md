# Отчёт стенда №2 — пара под systemd

Дата: 2026-09-11. Хост тот же (Arch, systemd 261). Установка — `stand-install.sh` (root), откат — `stand-rollback.sh`. Отличие от ТЗ 0.3: одна группа `monerizer` для чтения API, чтения конфигов и (в будущем) polkit-управления; причина — console cookie в `local/console` (отчёт №1, §6).

## 1. Что создано в системе

Группа `monerizer`; пользователи `monerizer-p2pool`, `monerizer-xmrig` (system, nologin, primary group `monerizer`); `/usr/local/bin/{p2pool,xmrig}` root 0755; `/etc/monerizer/{p2pool.conf,xmrig.json}` root:monerizer 0640; units `systemd/*.service`; `plasmoid` добавлен в `monerizer`. Каталоги `/var/lib/monerizer/{p2pool,xmrig}` (0700, StateDirectory) и `/run/monerizer-p2pool-api` (RuntimeDirectory) создаёт systemd. Автозапуск не включён. Дополнительно для ночной работы: `sleep/suspend/hibernate/hybrid-sleep.target` masked, drop-in logind `HandleLidSwitch*=ignore`.

## 2. Результаты

| Проверка | Результат |
|---|---|
| Unit-профиль SYS-02/SEC-08 (`ProtectSystem=strict`, `ProtectHome`, `NoNewPrivileges`, `PrivateTmp`) с RandomX/JIT | Оба процесса работают; XMRig `+JIT`, dataset 2336 MB выделен |
| `RuntimeDirectory` без setgid, `Group=monerizer` + `UMask=0027` | Каталоги 0750, файлы 0640, owner `monerizer-p2pool:monerizer` — setgid не нужен, `RuntimeDirectoryMode=0750` достаточно |
| Чтение API членом группы / посторонним | Член группы читает; посторонний — `Permission denied` на каталоге |
| `RuntimeDirectoryPreserve=no` | После `stop` каталог удалён, после `start` создан заново пустым (A26) |
| `systemctl show` всех свойств SYS-06 без прав | Доступны, включая `InvocationID`, `ExecMainStartTimestampMonotonic`, `RuntimeDirectory*`. `Requires=` содержит неявные `system.slice sysinit.target -.mount` — doctor ищет только имя парного unit |
| Несуществующий unit | `LoadState=not-found`, `ActiveState=inactive`, exit 0 (A12) |
| `stop -- p2pool xmrig` одной командой | 0,26 s; XMRig остановлен раньше P2Pool |
| `start -- p2pool xmrig` одной командой | 0,11 s; «Started P2Pool» раньше «Starting XMRig» (A09) |
| Журнал XMRig | Через `StandardOutput=journal` строки приходят (`_TRANSPORT=stdout`); `"syslog": true` давал дубли — убран из примера |
| XMRig под непривилегированным пользователем | `FAILED TO APPLY MSR MOD, HASHRATE WILL BE LOW`, huge pages 0/1168 — ожидаемое следствие SEC-08; ~1,4 kH/s на 4 потоках |
| Чтение журнала пользователем | `plasmoid` в `wheel` — читает; отдельная проверка без wheel не делалась |

## 3. Инцидент

`systemctl restart systemd-logind` для применения drop-in обрушил KDE Wayland-сессию (kwin потерял доступ к DRM), потребовалась жёсткая перезагрузка. Майнинг не причастен (XMRig ещё не стартовал, kernel-сообщений нет). Правило: logind на живой графической сессии не перезапускать; drop-in применяется при следующей загрузке.

## 4. Не проверено

`Restart=on-failure`/start-limit на неверной ноде (A12/A21), доступ к журналу без `wheel`, polkit-правило (SEC-03), второй хост Debian 13.
