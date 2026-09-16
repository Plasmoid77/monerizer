# Moneroid v1 — план реализации

Дата: 2026-09-11. Редакция: 0.2. Состояние на утро 2026-09-11: этапы 0–5 выполнены (стенд-ноутбук + VM Debian 13, см. §10 и [отчёт приёмки](research/acceptance-report-v1.md)); открыты systemd 249, A10/A20/A28.
Основание: [ТЗ 0.3](spec.md) и [контракты источников](research/upstream-contracts.md).
Этот план не разрешает установку служб и запуск майнинга; каждый этап начинается после явного подтверждения владельца.

## 0. Решения ревью ТЗ 0.2 → 0.3

Ревью 2026-09-11 не нашло противоречий между ТЗ 0.2 и проверенными контрактами. Внесены уточнения:

| Место | Изменение | Причина |
|---|---|---|
| CLI-07 | Ctrl-C в `logs --follow` и `tui` — штатный выход с кодом 0; 130 только для прерванных `status`, `doctor` и control | Иначе штатный выход из просмотра журнала считался бы ошибкой |
| SEC-08 | Явно: XMRig под непривилегированным `User=` не может применить собственный MSR-mod; это документируемый trade-off, а не неисправность | Без этого пользователь ждёт «максимальный hashrate» и получает `MSR: unknown` |
| DOC-05 (новый) | Каталог проверок doctor с кодами, компонентами и результатами | Реализация и тесты doctor требуют фиксированного списка |
| §15 | Добавлен этап 0 «репозиторий» и перечень решений владельца | Без module path, лицензии и стенда этап 1 не начинается |
| §4, §14 | Ссылки на стенд: Debian 13 (systemd 257) как «актуальная» система; вторая система с systemd 249 — по решению владельца | Совпадает с имеющимся тестовым хостом |
| §2, §5, §6.1, DOC-05, §14, §15 | Добавлен выбор ноды: `node list/select`, `NODE_RPC`, A29–A30, поля `params_file`/`nodes_file` | Решение владельца 2026-09-11 |
| §4, контракты §1 | P2Pool: актуальный релиз 4.18 (2026-08-05), контракты проверены по 4.17.1 — стенд сверяет с 4.18 | Проверка релизов 2026-09-11 |

Handoff v2 остаётся историческим документом; все его решения, отличающиеся от ТЗ, считаются отменёнными.

## 1. Решения владельца

Приняты 2026-09-11:

| # | Вопрос | Решение |
|---|---|---|
| D1 | Module path и хостинг | `github.com/Plasmoid77/moneroid` |
| D2 | Лицензия собственного кода (INSTALL-05). Upstream GPL-3.0 не наследуется: код XMRig/P2Pool не линкуется и не копируется | MIT |
| D3 | Стенд этапа 1 | Рабочий ноутбук владельца (Arch Linux, systemd актуальной версии). Установка users/units/бинарников на него выполняется только по явной команде владельца на каждом шаге; майнинг ограничен временем проверки. Приёмка на Debian 13 (этап 5) — VM или тестовый VPS |
| D4 | Monero-нода для стенда | Удалённая нода с RPC+ZMQ; на этапе 1 адрес указывает владелец вручную, с этапа 3 — `node select` (§1.1) |
| D5 | Адрес выплат для стенда | Тестовый адрес владельца; в fixtures заменяется на placeholder |
| D6 | Default sidechain примера | `mini` (ТЗ INSTALL-03) |
| D7 | Вторая система матрицы приёмки (systemd 249) | Решить на этапе 5 по результатам Debian 13 |

### 1.1. Автовыбор ноды

Владелец выбрал автовыбор ноды «как в Gupax» при требовании максимальной минималистичности. Принятая форма — ТЗ §6.1 (NODE-01..05): `node list` измеряет кандидатов из редактируемого `nodes.txt`, `node select` записывает три ключа в `p2pool.conf` и напоминает о restart. Это единственная запись Moneroid в нативный конфиг; встроенного списка, фонового переключения и выбора при `start` нет. Список-пример собирается на этапе 3 из публично документированных community-нод с ZMQ с указанием источника и даты; за актуальность списка после релиза отвечает владелец файла.

## 2. Этап 0 — репозиторий

Результат: пустой, но собираемый Go-проект с зафиксированными версиями и правилами.

1. `git init`, ветка `main`, `.gitignore` (бинарник, `dist/`, `*.token`, локальные captures до очистки).
2. `go mod init <D1>`, `go 1.27`, toolchain pin в `go.mod`; `CGO_ENABLED=0`.
3. Зависимости только: `charm.land/bubbletea/v2` (и то, что она тянет транзитивно), `github.com/BurntSushi/toml`. `golang.org/x/sys` допускается для `CLOCK_MONOTONIC` и isatty, если Bubble Tea уже приносит его транзитивно.
4. Каркас каталогов из ТЗ §15: `cmd/moneroid`, `internal/{config,status,systemd,xmrig,p2pool,doctor,tui}`, `examples`, `systemd`, `testdata`, `docs`.
5. `Makefile` с целями `build`, `test`, `vet`, `fmt-check`, `release` (см. §8). Без linters-фреймворков; `go vet` + `gofmt` обязательны.
6. `LICENSE` (D2), `README.md` переписывается по INSTALL-04 в конце этапа 3; до этого остаётся статусной заглушкой.
7. Документы: ТЗ и план остаются в `docs/`; `docs/research/` — проверенные факты; `docs/schema/` — JSON-схемы после этапа 1.

Критерий: `make build test vet fmt-check` проходит на пустом каркасе; `moneroid version` печатает версию из `-ldflags`.

## 3. Этап 1 — стенд и контракты

Цель: заменить предположения ТЗ §16 на наблюдения. Ни одна строка UI не пишется до закрытия этого этапа.

### 3.1. Подготовка стенда (администратор, вручную, по будущей инструкции)

1. Установить бинарники актуальных релизов (на 2026-09-11: P2Pool 4.18, XMRig 6.26.0) из официальных GitHub Releases, проверить checksum/подпись. Размещение `/usr/local/bin`.
2. Создать `moneroid-p2pool`, `moneroid-xmrig`, группу `moneroid-observers`; каталоги из ТЗ §5.2 с правами SEC-02.
3. Установить черновики `examples/p2pool.conf`, `examples/xmrig.json`, `systemd/*.service` (профиль SYS-02, SYS-03, SYS-11, SEC-08).
4. `daemon-reload`, `start` вручную через `systemctl`, наблюдать журнал.

Каждый шаг фиксируется в `docs/research/stand-report-1.md`: команда, версия, результат. Этот отчёт — основа `docs/install.md`.

### 3.2. Что подтверждается на стенде

| Проверка | Закрывает | Ожидание из ТЗ |
|---|---|---|
| Синтаксис params-file: `key = value`, boolean как `1`, отсутствие кавычек/наличие | INSTALL-02 | по COMMAND_LINE.MD |
| P2Pool при `StandardInput=null` работает и корректно останавливается по SIGTERM за < 30 s | SYS-02, A19 | EOF не останавливает pool |
| XMRig с read-only `--config` не пишет конфиг, не падает; `watch=false`, `autosave=false` (наличие ключа `autosave` и его default подтвердить) | INSTALL-03 | по документации config |
| Фактические owner/group/mode `RuntimeDirectory`, подкаталогов `local/`,`network/`,`pool/` и файлов | SEC-02, A24, A26 | 02750 / 0750 / 0640 |
| `RuntimeDirectoryMode=02750`: применяется ли setgid systemd | SEC-02 | если нет — переход на явный `chmod` невозможен; тогда группа задаётся через `Group=` и setgid не нужен |
| Период записи `local/p2p`, частота `local/stratum`, `network/stats`, `pool/stats` | DATA-07, §6 контрактов | 60 s / событийно |
| Наличие/значения полей всех четырёх файлов и `/2/summary` на эталонных версиях | §2–5 контрактов | таблицы контрактов |
| `hugepages`, `msr` в `/2/summary` и `/2/backends` при непривилегированном пользователе | MET-04, SEC-08 | MSR не применяется |
| `ProtectSystem=strict`, `ProtectHome`, `NoNewPrivileges` не ломают RandomX/JIT и запись cache | SEC-08 | без `MemoryDenyWriteExecute` |
| `systemctl show` без прав: набор свойств SYS-06 доступен | SYS-06, A11 | доступен |
| `journalctl -u` без прав/в группе `systemd-journal` | CLI-06, A11 | отказ виден |
| Поведение при `Restart=on-failure` + `StartLimitBurst` при неверном адресе node | A12, A21 | `start-limit-hit` в `Result` |
| Внешний порт P2P исходящий (main 37889 / mini 37888 / nano — по документации) без UPnP | SEC-06 | только исходящие |

### 3.3. Fixtures

- `testdata/xmrig/6.26.0/summary.json` — реальный ответ, очищенный: `id`→`moneroid-xmrig`, pool→`127.0.0.1:3333`, без токена.
- `testdata/p2pool/4.17.1/{local-p2p,local-stratum,network-stats,pool-stats}.json` — реальные, адрес и peers заменены placeholder.
- `testdata/systemd/show-*.txt` — реальный вывод `systemctl show` для active/inactive/failed/not-found/masked.
- Синтетические: `*-broken-*.json` (обрезанный JSON, не-object, null-поля, отрицательные counts, поле неверного типа, 1 MiB+1).
- К каждому real-fixture — заголовок-комментарий в соседнем `README.md`: версия, дата, что заменено.

### 3.4. Схемы

Фиксируются `docs/schema/status-v1.md` и `docs/schema/doctor-v1.md`: полный перечень полей, типов, nullable, единиц; пример JSON из fixtures. После этого `schema_version=1` заморожена: добавлять поля можно, менять/удалять — нельзя.

Критерии этапа: A05–A08, A19, A24 — как ручные наблюдения в отчёте; таблица 3.2 заполнена без строк «не проверено».

## 4. Этап 2 — read-only CLI

Результат: `moneroid status [--json] [--check]`, `version`, `config path` работают против стенда и полностью покрыты unit-тестами на fixtures.

### 4.1. Пакеты и интерфейсы

```text
cmd/moneroid/main.go
    dispatcher по §6 ТЗ; коды завершения CLI-07; --config до подкоманды.

internal/config
    type Config struct{ Services, P2Pool, XMRig, UI }
    func Load(path string) (Config, error)      // CFG-01..07; ошибка → код 2
    func Default() Config                        // только необязательные поля

internal/status                                  // модель + collector + health
    type Snapshot, ServiceState, Source, Issue, Health
    type Clock interface{ Now() time.Time; Monotonic() time.Duration }
    type Collector struct{ Systemd, XMRig, P2Pool; Clock; Deadline }
    func (c *Collector) Collect(ctx) Snapshot     // DATA-01, параллельно, bounded
    func Evaluate(s *Snapshot)                    // H-02, H-03, H-04; заполняет Health и Issues

internal/systemd
    type Runner interface{ Run(ctx, argv []string, limit int) (stdout, stderr []byte, code int, err error) }
    func Show(ctx, r Runner, units []string) (map[string]Props, error)   // SYS-06
    func Control(ctx, r Runner, verb string, units []string) Result       // SYS-07..09 (этап 3)
    func JournalArgs(units []string, lines int, follow bool) []string    // CLI-06 (этап 3)

internal/xmrig
    type Client struct{ HTTP interface{ Do(*http.Request) (*http.Response, error) }; URL; TokenFile }
    func (c *Client) Summary(ctx) (Summary, SourceMeta)                  // GET /2/summary, 1 MiB, без redirect/proxy

internal/p2pool
    type Reader struct{ FS fs.FS; Dir string }                          // fs.Stat для mtime
    func (r *Reader) Read(ctx) Files                                     // 4 файла, DATA-02..04
```

Правила: `internal/status` не импортирует адаптеры upstream-JSON целиком — адаптеры возвращают уже нормализованные nullable-структуры. Ни один пакет, кроме `internal/tui`, не импортирует Bubble Tea.

### 4.2. Порядок работ

1. `internal/config` + тесты (валидные/невалидные TOML, все CFG-правила).
2. `internal/systemd.Show` + парсер `key=value` вывода + fixtures.
3. `internal/xmrig` + тесты на `httptest.Server`: обычный ответ, 401, redirect, медленный ответ, oversized body, ID mismatch.
4. `internal/p2pool` + тесты на `fstest.MapFS`: свежий, stale, future mtime, повреждённый, отсутствующий, oversized, retry 50 ms.
5. `internal/status`: модель, collector с deadline, health-правила — табличные тесты по H-02/H-03.
6. Текстовый вывод `status` и JSON по `docs/schema/status-v1.md`; golden-тесты.
7. Проверка на стенде: A03 (XMRig без HTTP), A04 (старые файлы), A22, A25 (сдвиг часов).

Критерии: A03–A08, A12–A14, A16–A18, A22–A23, A25; `go test -race ./...` без обращений к сети и systemctl.

## 5. Этап 3 — эксплуатация

Результат: `start/stop/restart/logs/doctor`, поставляемые units и полная инструкция установки.

1. `systemd.Control`: одна операция на процесс (SYS-09), deadline 90 s (SYS-08), контрольное `Show` до 3 s, вывод результата CLI-05; коды 1/3/4.
2. `logs`: `journalctl --no-pager -u U [-u U2] -n N [-f] -o short-iso` или эквивалент без цвета; Ctrl-C → 0.
3. `internal/doctor`: проверки по DOC-05 ТЗ; JSON по `docs/schema/doctor-v1.md`; exit 1 при fail.
3a. `internal/node`: парсер `nodes.txt`, параллельный probe (`get_info` + TCP ZMQ), ранжирование, замена трёх ключей в params-file с атомарной записью; `examples/nodes.txt`. Тесты: `httptest` для RPC, `net.Listen` для ZMQ-порта, golden-тесты замены ключей на конфиге с комментариями и без ключей.
4. Финальные `systemd/moneroid-p2pool.service`, `systemd/moneroid-xmrig.service`, `examples/*` — версии, проверенные на этапе 1, плюс правки по результатам.
5. Опциональное polkit-правило `examples/polkit/50-moneroid.rules` строго по SEC-03; отдельный шаг инструкции.
6. Документация: `README.md` (INSTALL-04), `docs/install.md`, `docs/troubleshooting.md`, `docs/updating.md`. Инструкция обязана требовать подстановку адреса и node до запуска (INSTALL-01).

Критерии: A01, A09–A12, A19, A21, A24, A28–A30 на стенде; отчёт `docs/research/stand-report-2.md`.

## 6. Этап 4 — TUI

Результат: `moneroid tui` над тем же `Collector` и `Control`.

1. Модель Bubble Tea v2: состояния `dashboard | services-menu | confirm | help | small-terminal`; таймер по `refresh_ms`; пропуск тика при незавершённом сборе (DATA-01, A17).
2. История hashrate: кольцевой буфер 1200 точек / 10 минут (DATA-09), разрыв при InvocationID/uptime reset (DATA-08).
3. Control из меню: подтверждение с именами units, Cancel по умолчанию (UI-03); один in-flight запрос.
4. Журнал: выход из alt-screen, дочерний `journalctl -f`; **обязательно** — дочерний процесс в собственной foreground process group или временное игнорирование SIGINT родителем, иначе Ctrl-C в просмотре завершит панель. Проверить на стенде.
5. `NO_COLOR`, ASCII fallback, 80×24 и компактный режим (UI-05, UI-06); фильтрация управляющих символов из upstream-строк (A18).
6. Восстановление терминала при panic/сигналах; `q` и Ctrl-C никогда не вызывают control.

Критерии: A02, A07–A11, A15, A17–A18. TUI-тесты: unit-тесты модели (Update на синтетических сообщениях) без реального терминала; ручной чек-лист размеров терминала в отчёте.

## 7. Этап 5 — релизная приёмка

1. Полная матрица A01–A28 на Debian 13; отдельно A20 (замена бинарника на следующую upstream-версию, если вышла) и A26–A28.
2. Вторая система по D7.
3. 30-минутное наблюдение TUI: goroutines и RSS стабильны (нефункциональные критерии §14).
4. Отчёт `docs/research/acceptance-report-v1.md`: ОС, systemd, upstream-версии, команда, результат по каждому A-пункту.
5. Release: `moneroid_<ver>_linux_amd64` + `SHA256SUMS` + `examples/` + `systemd/` + `docs/`. Публикация — решение владельца (GitHub Releases).

## 8. Правила работы во время реализации

- Один writer: правки только в этой сессии; субагенты — исследование, аудит, прогон тестов.
- Майнинг, установка служб и изменения системы — только на стенде (D3, рабочий ноутбук) и только по явной команде владельца на каждый шаг: создание пользователей, установка units, `daemon-reload`, `start`. Перед этапом 1 фиксируется список всего, что будет создано в системе, и команда полного отката; после закрытия этапа стенд убирается тем же списком.
- Никаких токенов, адресов выплат, RPC credentials в репозитории, fixtures, отчётах и промптах субагентов.
- Каждый commit собирается `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=<ver>"`, проходит `go vet`, `gofmt -l`, `go test -race`.
- Максимальная минималистичность — требование владельца. Новый пакет или зависимость — только с записью причины в commit; целевой объём — порядка 3–4 тыс. строк Go без тестов. Рост выше — сигнал переусложнения, а не нормы. Перед добавлением любой функции — вопрос «можно ли это сделать стандартной командой systemd/journalctl/upstream?»; если да, функция не добавляется.
- Порядок фиксирован: контракты → CLI → эксплуатация → TUI. Оформление панели до закрытия этапа 2 не начинается.
- Ограничения handoff §47 (не форкать/не вендорить upstream, не реализовывать Stratum/RPC/updater/БД/web) действуют как есть.

## 9. Риски

| Риск | Влияние | Действие |
|---|---|---|
| Удалённая нода недоступна/не синхронизирована | Этап 1 не закрывается, fixtures синтетические | Владелец выбирает ноду до этапа 1; кандидаты проверяются вручную |
| Стенд на рабочей машине оставляет следы (users, units, каталоги) | Загрязнение системы владельца | Список создаваемых объектов и откат фиксируются до установки |
| Arch на стенде ≠ Debian 13 целевой системы (версии systemd, пути) | Расхождения всплывут поздно | Приёмка этапа 5 на Debian 13; unit-профиль не использует Arch-специфику |
| `RuntimeDirectoryMode` не принимает setgid | Права SEC-02 меняются | Проверка в таблице 3.2; fallback описан там же |
| Bubble Tea v2 API отличается от v1-примеров | Задержка этапа 4 | Только официальные v2-примеры и godoc; TUI последний |
| Upstream-версии обновятся в ходе работ | Fixtures устаревают | Версии в именах fixtures; A20 в этапе 5 |
| Ctrl-C в просмотре журнала убивает TUI | UI-04 не выполняется | Явный пункт этапа 4.4 |
| Список нод в `nodes.txt` устаревает | `node select` не находит кандидатов | Файл редактируемый, дата проверки в нём; код 1 с понятным сообщением, не падение |

## 10. Ход работ (ночь 2026-09-11)

Порядок был изменён владельцем на инкрементальный: сначала голая пара от пользователя (отчёт №1), затем пара под systemd (отчёт №2), затем код по одной команде за коммит.

| Коммит | Что |
|---|---|
| `873ba41` | документы, units, примеры, fixtures, `status/--json/--check`, `version`, `config path` |
| `06755c3` | `start/stop/restart`, `logs`, polkit-правило |
| `32e77c5` | `doctor` (DOC-05) |
| `256c788` | `node list/select`, `NODE_RPC` |
| `93339ed` | `tui` (Bubble Tea v2) |
| `daeab5b` | README, install, troubleshooting, updating |
| `18affd6` | reset-failed перед start/restart (start-limit-hit) |
| `eef0639` | правки по независимому аудиту (deep-auditor, 22 находки) |

Проверено на стенде вручную: A03, A04 (частично), A09, A11, A12 (not-found), A13, A14, A22, A24, A26, A29, A30; TUI — меню, отмена по умолчанию, restart через polkit, журнал с возвратом по Ctrl-C, компактный режим, отказ без TTY. Не проверено: A16 (медленный HTTP — только unit-тестом), A19 start-limit, A20, A25 (только unit-тест), A28, 30-минутное наблюдение памяти, Debian 13.
