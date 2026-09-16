# Schema of `moneroid payouts --json`, schema_version 1

| Field | Type |
|---|---|
| `schema_version` | integer (1) |
| `unit` | string — the P2Pool unit |
| `payouts[]` | array |
| `payouts[].at` | string RFC3339 UTC (zero time if the timestamp could not be parsed) |
| `payouts[].atomic_units` | integer — piconero |
| `payouts[].xmr` | string — as in the journal, 12 decimals |
| `payouts[].block` | integer — Monero block height |
| `total_atomic_units` | integer |
| `total_xmr` | string |
| `blocks_without_payout` | integer — pool blocks without a share of yours in the PPLNS window |
