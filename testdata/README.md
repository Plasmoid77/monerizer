# Fixtures

Реальные captures со стенда 2026-09-11 (Arch Linux, запуск от пользователя без systemd), см. `docs/research/stand-report-1.md`.

| Каталог | Источник | Что заменено |
|---|---|---|
| `p2pool/4.18/` | P2Pool v4.18, Data API (`--data-api` + `--local-api`), mini sidechain, нода xmr.support | адрес выплат → `4AAA…`; IP peers → `192.0.2.1`; `local/console`: cookie → `REDACTED`, порт → 0 |
| `xmrig/6.26.0/` | XMRig 6.26.0, `GET /2/summary`, `GET /2/backends`, 4 потока, без hugepages/MSR | ничего (токена не было, pool = 127.0.0.1:3333) |

Имена файлов Data API у upstream без расширения (`local/p2p`); здесь — `local-p2p.json`. Синтетические повреждённые варианты добавляются рядом с суффиксом `-broken-*`.
