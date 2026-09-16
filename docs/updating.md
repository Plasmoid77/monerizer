# Updating

## Upstream (P2Pool, XMRig)

Moneroid does not need to change as long as the interfaces stay the same: `systemctl show`, `GET /2/summary`, the four Data API files.

```sh
# verify the checksum/signature of the new release, then:
install -o root -g root -m 0755 p2pool /usr/local/bin/p2pool && moneroid restart p2pool
install -o root -g root -m 0755 xmrig  /usr/local/bin/xmrig  && moneroid restart xmrig
moneroid doctor
```

If the binary path changed, edit `ExecStart=` in the unit (drop-in via `systemctl edit`), then `daemon-reload`. If upstream renamed or removed a field, `status` shows `FIELD_INVALID` or `null` for that metric and everything else keeps working.

`install.sh` pins upstream versions and their SHA256; a new release means updating `P2POOL_VERSION`/`P2POOL_SHA256` (from the signed `sha256sums.txt.asc`) and `XMRIG_VERSION`/`XMRIG_SHA256` (from `SHA256SUMS`) in the script.

## Node

`sudo moneroid node select` (or edit `host/rpc-port/zmq-port` by hand), then `moneroid restart p2pool`. The candidate list `/etc/moneroid/nodes.txt` is your file: nodes disappear, keep it up to date.

## Moneroid

```sh
make build && install -o root -g root -m 0755 moneroid /usr/local/bin/moneroid
```

or re-run `install.sh` (it re-downloads the pinned release, verifies it and restarts the services). The services do not depend on the Moneroid binary: updating or removing it does not interrupt mining. The `status --json` and `doctor --json` schemas are versioned by `schema_version`; within a version fields are only added.

Releasing: builds are reproducible (`-trimpath -buildvcs=false`), so `VERSION=vX make build`, put the binary's SHA256 and the extras tarball's SHA256 into `install.sh`, commit, tag, and the build at the tag yields the same hash.
