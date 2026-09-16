# Schema of `moneroid doctor --json`, schema_version 1

`checks[].result`: `pass|warn|fail|skip`; `remedy` is a string or `null`. Exit code 1 when `summary.fail > 0`. The code catalogue is spec DOC-05.

| Field | Type in the example |
|---|---|
| `schema_version` | integer |
| `collected_at` | string |
| `checks[]` | array |
| `checks[].code` | string |
| `checks[].component` | string |
| `checks[].result` | string |
| `checks[].message` | string |
| `checks[].remedy` | null |
| `summary.fail` | integer |
| `summary.pass` | integer |
| `summary.skip` | integer |
| `summary.warn` | integer |

Check codes v1 (plus `SIDECHAIN_SYNC` added later): `CLOCK`, `CONTROL_ACCESS`, `DATA_API_DIR`, `DATA_API_EVENT_FILES`, `DATA_API_P2P`, `JOURNAL_ACCESS`, `NODE_RPC`, `P2P_CONNECTIONS`, `SYSTEMD_AVAILABLE`, `TOKEN_FILE`, `UNIT_ACTIVE`, `UNIT_DEPENDENCIES`, `UNIT_ENABLED`, `UNIT_LOADED`, `UNIT_MASKED`, `UNIT_ORDERING`, `UNIT_RUNTIME_DIR`, `XMRIG_API`, `XMRIG_CONNECTED`, `XMRIG_HASHRATE`, `XMRIG_HUGEPAGES`, `XMRIG_ID`, `XMRIG_SESSION`, `ZMQ_ACTIVITY`.
