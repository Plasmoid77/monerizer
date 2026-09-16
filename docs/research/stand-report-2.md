# Stand report 2 — the pair under systemd

Date: 2026-09-11. Same host (Arch, systemd 261). Installed with the stand script (root), rolled back with its counterpart (both superseded by `install.sh` / `install.sh --uninstall --purge` since v0.4.0). Difference from spec 0.3: one group `moneroid` for reading the API, reading the configs and (later) polkit control; the reason is the console cookie in `local/console` (report 1, §6).

## 1. What was created on the system

Group `moneroid`; users `moneroid-p2pool`, `moneroid-xmrig` (system, nologin, primary group `moneroid`); `/usr/local/bin/{p2pool,xmrig}` root 0755; `/etc/moneroid/{p2pool.conf,xmrig.json}` root:moneroid 0640; units `systemd/*.service`; `plasmoid` added to `moneroid`. The directories `/var/lib/moneroid/{p2pool,xmrig}` (0700, StateDirectory) and `/run/moneroid-p2pool-api` (RuntimeDirectory) are created by systemd. Autostart not enabled. Additionally for the overnight run: `sleep/suspend/hibernate/hybrid-sleep.target` masked, a logind drop-in `HandleLidSwitch*=ignore`.

## 2. Results

| Check | Result |
|---|---|
| Unit profile SYS-02/SEC-08 (`ProtectSystem=strict`, `ProtectHome`, `NoNewPrivileges`, `PrivateTmp`) with RandomX/JIT | Both processes run; XMRig `+JIT`, 2336 MB dataset allocated |
| `RuntimeDirectory` without setgid, `Group=moneroid` + `UMask=0027` | Directories 0750, files 0640, owner `moneroid-p2pool:moneroid` — no setgid needed, `RuntimeDirectoryMode=0750` suffices |
| Reading the API as a group member / an outsider | The member reads; the outsider gets `Permission denied` on the directory |
| `RuntimeDirectoryPreserve=no` | After `stop` the directory is gone, after `start` it is recreated empty (A26) |
| `systemctl show` of all SYS-06 properties without privileges | Available, including `InvocationID`, `ExecMainStartTimestampMonotonic`, `RuntimeDirectory*`. `Requires=` contains the implicit `system.slice sysinit.target -.mount` — doctor looks only for the paired unit's name |
| A non-existent unit | `LoadState=not-found`, `ActiveState=inactive`, exit 0 (A12) |
| `stop -- p2pool xmrig` in one command | 0.26 s; XMRig stopped before P2Pool |
| `start -- p2pool xmrig` in one command | 0.11 s; "Started P2Pool" before "Starting XMRig" (A09) |
| XMRig journal | Via `StandardOutput=journal` the lines arrive (`_TRANSPORT=stdout`); `"syslog": true` produced duplicates — removed from the example |
| XMRig as an unprivileged user | `FAILED TO APPLY MSR MOD, HASHRATE WILL BE LOW`, huge pages 0/1168 — the expected consequence of SEC-08; ~1.4 kH/s on 4 threads |
| Journal reading as a user | `plasmoid` is in `wheel` — reads; a separate check without wheel was not done |

## 3. Incident

`systemctl restart systemd-logind` to apply the drop-in crashed the KDE Wayland session (kwin lost DRM access); a hard reboot was needed. Mining was not involved (XMRig had not started yet, no kernel messages). Rule: never restart logind on a live graphical session; the drop-in applies at the next boot.

## 4. Not verified

`Restart=on-failure`/start-limit on a wrong node (A12/A21), journal access without `wheel`, the polkit rule (SEC-03), a second Debian 13 host.
