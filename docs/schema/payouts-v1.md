# Схема `moneroid payouts --json`, schema_version 1

| Поле | Тип |
|---|---|
| `schema_version` | integer (1) |
| `unit` | string — unit P2Pool |
| `payouts[]` | array |
| `payouts[].at` | string RFC3339 UTC (нулевое время, если метка не разобрана) |
| `payouts[].atomic_units` | integer — piconero |
| `payouts[].xmr` | string — как в журнале, 12 знаков |
| `payouts[].block` | integer — высота блока Monero |
| `total_atomic_units` | integer |
| `total_xmr` | string |
| `blocks_without_payout` | integer — блоки пула без вашей доли в PPLNS-окне |
