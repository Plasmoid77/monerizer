#!/bin/sh
# Moneroid installer: one P2Pool + XMRig pair under systemd from the official
# releases, verified against SHA256 pinned below. POSIX sh, idempotent, Linux
# x86_64 with systemd (Debian/Ubuntu/Arch tested). Re-runs itself under sudo.
#
#   sh install.sh --wallet 4…                       # mini sidechain, node picked from nodes.txt
#   sh install.sh --wallet 4… --node HOST:RPC:ZMQ   # fixed Monero node
#   sh install.sh --wallet 4… --enable --hugepages  # autostart + sysctl/MSR tuning
#   sh install.sh --uninstall [--purge]             # remove services (and configs/state with --purge)
#
# Existing files in /etc/moneroid are never overwritten; delete them to re-seed.
set -eu

MONEROID_VERSION=v0.4.0
MONEROID_SHA256=3b3e147442e3850eda1a8421ba015dc1735dbaad76eff2c2a8209dc040aa4121
EXTRAS_SHA256=495444f5008cc1d1f4e80b26497c02188c027af82e69a310930f08a95f830e13
P2POOL_VERSION=v4.18
P2POOL_SHA256=893691726b0218fe1883a7a326e2c69db4eb228fc72ba00c8adfa6be85b8a415 # from sha256sums.txt.asc, signature checked
XMRIG_VERSION=6.26.0
XMRIG_SHA256=fc6f8ae5f64e4f17481f7e3be29a1c56949f216a998414188003eae1db20c9e5 # from SHA256SUMS

usage() { sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'; exit 2; }
case " $* " in *" -h "* | *" --help "*) usage ;; esac
[ "$(id -u)" -eq 0 ] || exec sudo -- sh "$0" --user "$(id -un)" "$@"

WALLET='' SIDECHAIN=mini NODE='' ENABLE=0 HUGEPAGES=0 OPERATOR=${SUDO_USER:-} UNINSTALL=0 PURGE=0
while [ $# -gt 0 ]; do
	case $1 in
	--wallet) WALLET=$2; shift ;;
	--sidechain) SIDECHAIN=$2; shift ;;
	--node) NODE=$2; shift ;;
	--user) OPERATOR=$2; shift ;;
	--enable) ENABLE=1 ;;
	--hugepages) HUGEPAGES=1 ;;
	--uninstall) UNINSTALL=1 ;;
	--purge) PURGE=1 ;;
	-h | --help) usage ;;
	*) echo "unknown argument: $1" >&2; usage ;;
	esac
	shift
done

die() { echo "install.sh: $*" >&2; exit 1; }
command -v systemctl >/dev/null || die "systemd is required"
[ "$(uname -m)" = x86_64 ] || die "only Linux x86_64 binaries are pinned here"

UNITS="moneroid-p2pool.service moneroid-xmrig.service"
if [ $UNINSTALL = 1 ]; then
	systemctl disable --now $UNITS moneroid-msr.service 2>/dev/null || true
	rm -f /etc/systemd/system/moneroid-p2pool.service /etc/systemd/system/moneroid-xmrig.service \
		/etc/systemd/system/moneroid-msr.service /etc/polkit-1/rules.d/50-moneroid.rules \
		/etc/sysctl.d/90-moneroid-hugepages.conf /usr/local/bin/moneroid /usr/local/bin/p2pool /usr/local/bin/xmrig
	systemctl daemon-reload
	if [ $PURGE = 1 ]; then
		rm -rf /etc/moneroid /var/lib/moneroid
		userdel moneroid-p2pool 2>/dev/null || true
		userdel moneroid-xmrig 2>/dev/null || true
		groupdel moneroid 2>/dev/null || true
	fi
	echo "removed; kept /etc/moneroid and /var/lib/moneroid unless --purge"
	exit 0
fi

case $WALLET in 4*) [ ${#WALLET} -eq 95 ] || die "--wallet must be a 95-character primary address" ;;
*) [ -f /etc/moneroid/p2pool.conf ] || die "--wallet 4… is required (primary address, not a subaddress)" ;; esac
case $SIDECHAIN in mini | nano | main) ;; *) die "--sidechain must be mini, nano or main" ;; esac

fetch() { # url dest
	if command -v curl >/dev/null; then curl -fsSL --retry 3 -o "$2" "$1"
	elif command -v wget >/dev/null; then wget -q -O "$2" "$1"
	else die "curl or wget is required"; fi
}
verify() { echo "$2  $1" | sha256sum -c --quiet - || die "checksum mismatch for $1"; }

TMP=$(mktemp -d) && trap 'rm -rf "$TMP"' EXIT
cd "$TMP"
echo "downloading Moneroid $MONEROID_VERSION, P2Pool $P2POOL_VERSION, XMRig $XMRIG_VERSION…"
fetch "https://github.com/Plasmoid77/moneroid/releases/download/$MONEROID_VERSION/moneroid-$MONEROID_VERSION-linux-amd64" moneroid
fetch "https://github.com/Plasmoid77/moneroid/releases/download/$MONEROID_VERSION/moneroid-$MONEROID_VERSION-extras.tar.gz" extras.tar.gz
fetch "https://github.com/SChernykh/p2pool/releases/download/$P2POOL_VERSION/p2pool-$P2POOL_VERSION-linux-x64.tar.gz" p2pool.tar.gz
fetch "https://github.com/xmrig/xmrig/releases/download/v$XMRIG_VERSION/xmrig-$XMRIG_VERSION-linux-static-x64.tar.gz" xmrig.tar.gz
verify moneroid "$MONEROID_SHA256"; verify extras.tar.gz "$EXTRAS_SHA256"
verify p2pool.tar.gz "$P2POOL_SHA256"; verify xmrig.tar.gz "$XMRIG_SHA256"
tar xzf extras.tar.gz
tar xzf p2pool.tar.gz --strip-components=1 --wildcards '*/p2pool'
tar xzf xmrig.tar.gz --strip-components=1 --wildcards '*/xmrig'

NOLOGIN=$(command -v nologin || echo /usr/sbin/nologin)
groupadd -r moneroid 2>/dev/null || true
useradd -r -g moneroid -s "$NOLOGIN" -d /var/lib/moneroid/p2pool -M moneroid-p2pool 2>/dev/null || true
useradd -r -g moneroid -s "$NOLOGIN" -d /var/lib/moneroid/xmrig -M moneroid-xmrig 2>/dev/null || true

install -o root -g root -m 0755 moneroid /usr/local/bin/moneroid
install -o root -g root -m 0755 p2pool /usr/local/bin/p2pool
install -o root -g root -m 0755 xmrig /usr/local/bin/xmrig
install -d -o root -g root -m 0755 /etc/moneroid
install -o root -g root -m 0644 systemd/moneroid-p2pool.service systemd/moneroid-xmrig.service /etc/systemd/system/
if [ -d /etc/polkit-1/rules.d ]; then install -o root -g root -m 0644 examples/polkit/50-moneroid.rules /etc/polkit-1/rules.d/
else echo "note: polkit without rules.d (< 0.106): start/stop need sudo"; fi

seed() { if [ -e "/etc/moneroid/$1" ]; then echo "keeping existing /etc/moneroid/$1"; else install -o root -g "$2" -m "$3" "examples/$1" "/etc/moneroid/$1"; fi; }
if [ ! -e /etc/moneroid/p2pool.conf ]; then
	sed -e "s/^wallet = .*/wallet = $WALLET/" -e '/^mini = 1$/d' examples/p2pool.conf > p2pool.conf
	[ "$SIDECHAIN" = main ] || printf '%s = 1\n' "$SIDECHAIN" >> p2pool.conf
	install -o root -g moneroid -m 0640 p2pool.conf /etc/moneroid/p2pool.conf
else echo "keeping existing /etc/moneroid/p2pool.conf (wallet/sidechain arguments ignored)"; fi
seed xmrig.json moneroid 0640
seed nodes.txt root 0644
if [ ! -e /etc/moneroid/moneroid.toml ]; then
	sed -e 's|^# params_file|params_file|' -e 's|^# nodes_file|nodes_file|' examples/moneroid.toml > moneroid.toml
	install -o root -g root -m 0644 moneroid.toml /etc/moneroid/moneroid.toml
fi
if [ -n "$NODE" ]; then
	h=${NODE%%:*}; r=${NODE#*:}; r=${r%%:*}; z=${NODE##*:}
	sed -i -e "s/^host = .*/host = $h/" -e "s/^rpc-port = .*/rpc-port = $r/" -e "s/^zmq-port = .*/zmq-port = $z/" /etc/moneroid/p2pool.conf
fi

if [ $HUGEPAGES = 1 ]; then
	numa=$(ls -d /sys/devices/system/node/node[0-9]* 2>/dev/null | wc -l); [ "$numa" -ge 1 ] || numa=1
	# RandomX dataset per NUMA node (~1040 pages) + cache + scratchpads + P2Pool light-mode + margin
	nr=$((1100 * numa + 200 + $(nproc) + 300))
	printf 'vm.nr_hugepages = %s\n' "$nr" > /etc/sysctl.d/90-moneroid-hugepages.conf
	sysctl -q -p /etc/sysctl.d/90-moneroid-hugepages.conf || true
	echo "hugepages: vm.nr_hugepages=$nr (fully effective after a reboot)"
	if grep -q GenuineIntel /proc/cpuinfo; then
		if ! command -v wrmsr >/dev/null; then
			if command -v apt-get >/dev/null; then { apt-get update -qq && apt-get install -y -qq msr-tools; } >/dev/null 2>&1 || true
			elif command -v pacman >/dev/null; then pacman -S --noconfirm --needed msr-tools >/dev/null 2>&1 || true; fi
		fi
		if command -v wrmsr >/dev/null; then
			install -o root -g root -m 0644 examples/moneroid-msr.service /etc/systemd/system/
			systemctl enable -q moneroid-msr.service
		else echo "note: msr-tools not found; MSR unit skipped"; fi
	fi
fi

systemctl daemon-reload
if [ -n "$OPERATOR" ] && [ "$OPERATOR" != root ]; then
	usermod -aG moneroid "$OPERATOR" && echo "added $OPERATOR to group moneroid (re-login to read the Data API)"
fi

if [ -z "$NODE" ] && grep -q '^host = 127.0.0.1$' /etc/moneroid/p2pool.conf; then
	echo "probing Monero nodes from /etc/moneroid/nodes.txt…"
	moneroid node select || echo "no usable node found; set host/rpc-port/zmq-port in /etc/moneroid/p2pool.conf and run: moneroid start"
fi
moneroid start
if [ $ENABLE = 1 ]; then systemctl enable -q $UNITS && echo "autostart enabled"; fi
echo; moneroid doctor || true
echo; echo "done: moneroid status | moneroid tui | moneroid logs --follow p2pool  (sidechain sync takes a few minutes)"
