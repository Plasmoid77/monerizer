# Приёмка v1 — отчёт

Дата: 2026-09-11 (ночь). Бинарник: коммит `eef0639` (после независимого аудита). Upstream: P2Pool 4.18, XMRig 6.26.0. Нода: удалённая из `nodes.txt` (выбрана `node select`).

## Матрица

| Система | systemd | Как | Результат |
|---|---|---|---|
| Arch Linux (ноутбук владельца, i7-8650U) | 261 | стенд, отчёты №1–2 | все команды, TUI, polkit, node select, start-limit-hit |
| Debian 13 trixie (libvirt VM, UEFI, 2 vCPU, 6 GB) | 257 | `stand-install.sh` + те же файлы | install → doctor → start → SYNCHRONIZED за ~4 мин → health ok; polkit restart из группы; stop удаляет RuntimeDirectory; logs; `status --json`; TUI по ssh -t. Откат `stand-rollback.sh`, VM удалена |

Вторая система с systemd 249 (D7) не проверялась: обе системы новее; поддержка 249 остаётся заявленной границей без доказательства.

## Сценарии ТЗ §14

| ID | Результат | Где |
|---|---|---|
| A01 | pass — службы живут без Monerizer и без SSH-сессии | стенд, всю ночь |
| A02 | pass — `q`/Ctrl-C/Esc не вызывают control (код + unit-тест `TestQuitNeverControls`) | tui |
| A03 | pass — XMRig active без HTTP → API unavailable, метрики null | unit-тест `xmrig api down`; стенд (остановленный xmrig) |
| A04 | pass — старые файлы без доказанной связи → `SOURCE_SESSION_UNKNOWN`/`NOT_CURRENT_SESSION` | unit-тесты |
| A05 | pass — retry 50 ms, независимость источников | unit-тесты p2pool/status |
| A06 | pass — поле неверного типа → `FIELD_INVALID`, соседи целы | unit-тесты xmrig/status |
| A07 | pass — 0 и null различимы в тексте/JSON/спарклайне | unit-тест tui |
| A08 | pass — новый InvocationID → разрыв графика | unit-тест tui |
| A09 | pass — порядок start/stop одной командой | отчёт №2 |
| A10 | частично — timeout/Ctrl-C коды 4/130 реализованы; на стенде не воспроизводились |
| A11 | pass — `nobody`: read-only работает, отказ виден (`PERMISSION_DENIED`), doctor warn JOURNAL_ACCESS | стенд |
| A12 | pass — not-found, failed/start-limit-hit без разбора human-readable | стенд |
| A13 | pass — один валидный JSON при полном отказе источников | стенд (nobody), VM |
| A14 | pass — `--check` 0/1 для ok/degraded/stopped/unknown | стенд |
| A15 | pass — 60×15 компактный режим, non-TTY отказ; resize не тестировался отдельно | pty |
| A16 | pass — redirect/oversized/timeout в unit-тестах; внешних адресов нет по CFG-04 |
| A17 | pass — 30 мин TUI: RSS 15–18 MB, 15 потоков, без роста | soak.log |
| A18 | pass — sanitize для pool/id/version/сообщений; токен только в заголовке | код, аудит |
| A19 | pass — stdin=null, SIGTERM 0,4 s, start-limit после 5 падений | отчёты №1–2, стенд |
| A20 | не проверено — новой upstream-версии за ночь не вышло |
| A21 | pass — неверный конфиг → doctor fail с remedy; путь бинарника принадлежит unit | стенд |
| A22 | pass — оба inactive → `stopped`, enabled ≠ active | стенд |
| A23 | pass — shares 0 и старые rejects не дают ошибок | код, стенд |
| A24 | pass — read-only конфиги, группа читает API, не пишет | отчёт №2, VM |
| A25 | pass — будущий mtime → `CLOCK_UNCERTAIN` | unit-тест |
| A26 | pass — RuntimeDirectory удаляется/создаётся с нужными правами | отчёт №2, VM |
| A27 | pass — ID mismatch / uptime mismatch → degraded | unit-тесты |
| A28 | частично — doctor читает effective `After/Wants/Requires/BindsTo/PartOf`; drop-in на стенде не ставился |
| A29 | pass — `node list`: таймауты, ранжирование, непригодные внизу | стенд |
| A30 | pass — `node select`: только три ключа, owner/mode сохранены, dry-run не пишет | стенд |

Нефункциональные: `status` собирает за 20–40 ms на стенде; deadline 3 s соблюдён кодом; память TUI стабильна (A17).

## Открытое

- systemd 249 (Ubuntu 22.04) — не проверено.
- A10 (timeout control), A20 (смена upstream-версии), A28 (drop-in) — не воспроизводились на живой системе.
- XMRig `SHA256SUMS.sig` не проверялась подписью (ключ не импортировался).
