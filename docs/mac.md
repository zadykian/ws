# The Mac

The Mac reaches the server, ws, over the cable from the 2.5GbE port of the display it docks at. It
reaches it over the home LAN too, or from elsewhere through the router's port forwards.
`mac/setup.sh`, the one part of the repository that runs on the Mac, sets up two hosts for ssh:

- `ws`, through a WireGuard tunnel that ends on ws, at `10.99.0.1` whichever way the Mac connects.
  The sessions of `ssh ws` and JetBrains Gateway then stay open while the Mac moves between the
  cable, the LAN and elsewhere, with a pause of a few seconds;
- `ws-ssh`, plain SSH over whichever of the three the Mac is on, for networks that block the
  tunnel's UDP.

The script is POSIX sh, as the Mac has no Ansible, and runs as the user, from a clone, with sudo
where it needs root. It installs WireGuard's tools from Homebrew, which is
[installed](https://brew.sh) by hand first:

```sh
git clone https://github.com/zadykian/ws ~/repository/ws
~/repository/ws/mac/setup.sh
```

## mac.conf

The repository names no server, so the Mac's values go in `~/.config/ws/mac.conf`, `KEY=value`
lines that the script reads without running them:

```sh
WS_PUBLIC_KEY=WS_KEY
WS_LAN=10.88.0.0/24
WS_LAN_ADDRESS=10.88.0.10
WS_REMOTE=ws.example.com
WS_REMOTE_PORT=22022
```

`WS_PUBLIC_KEY` is ws's WireGuard public key, which the `wireguard` role prints:
[The Mac's tunnel](#the-macs-tunnel), below. `WS_LAN` is the home LAN, and `WS_LAN_ADDRESS` ws's
address on it, the router's static lease for it. `WS_REMOTE` is the router's name on the
internet, and `WS_REMOTE_PORT` the port it forwards to ws's 22. The tunnel takes UDP 51820 every
way, which the router forwards too; `WS_PORT` sets another. A way without its values is left out.
The router's side is in [The router](router.md).

## ssh's hosts

The script writes `~/.ssh/config.d/ws`. Its host `ws` goes to `10.99.0.1` from every network;
where UDP is blocked, `ssh ws` times out, and `ssh ws-ssh` serves. `ws-ssh` goes:

- over the cable, to `10.77.0.1`, where one of the Mac's addresses lies in `10.77.0.0/30`;
- over the LAN, to `WS_LAN_ADDRESS`, where one lies in `WS_LAN`;
- elsewhere, to `WS_REMOTE` on `WS_REMOTE_PORT`, over IPv4, as the forward takes IPv4 alone.

ssh picks the way from the Mac's own addresses (`Match localnetwork`, OpenSSH 9.4 and later), and
probes nothing. Both hosts log in as root with `~/.ssh/id_ed25519`, its passphrase kept in the
Keychain, and check ws's host key under the one name `ws` (`HostKeyAlias`). So `known_hosts` has
one entry for ws whichever way ssh goes. A host on another network that uses the LAN's subnet then
meets a host key mismatch, where ssh stops. A session ends after 90 s without word from ws, far
longer than a switch of the tunnel takes.

The script puts `Include config.d/*` first in `~/.ssh/config`, as ssh keeps the first value it reads
for each key. Where the line is missing, it adds it at the top and keeps the old file as
`~/.ssh/config.TIME~`; where the line sits below other settings, the script stops. It warns of a
`Host` or `Match` of the file's own for `ws` or `ws-ssh`. A run changes only what differs, so a
second prints nothing, and `--check` prints each change as a diff and makes none.

The Mac's end of the cable is set by hand, once. In System Settings › Network, pick the display's
adapter › Details › TCP/IP, and Configure IPv4 Manually. Give it the IP address `10.77.0.2`, the
subnet mask `255.255.255.252` and no router.

Through the forward, ws takes logins from the internet: by key alone, as everywhere, with fail2ban
banning the address of the network the Mac is on after 5 failed ones. Over the tunnel, ws sees
the Mac as `10.99.0.2`, which a ban would shut out of `ssh ws` for 10 minutes, while `ws-ssh` comes
from another address.

The first login asks whether to trust ws's host key: compare its fingerprint with the one
`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` prints on ws's console. On the cable and the
LAN, macOS asks once whether iTerm2, Gateway or another app may find devices on the local network,
for `ws-ssh`: allow it. ssh run from Terminal needs no such grant, and neither does `ws`. The
tunnel's utun is no local network, and macOS lets daemons that run as root reach one anyway
(Apple's TN3179). To see which way `ws-ssh` went:

```sh
ssh -v ws-ssh exit 2>&1 | grep 'Connecting to'
```

## The Mac's tunnel

The keys come first, in this order. A run of the `wireguard` role makes ws's key and prints its
public key. `setup.sh`, without `WS_PUBLIC_KEY`, makes the Mac's key, which only root reads. It
prints its public key with the two commands that make the Mac ws's peer, over `ws-ssh`:

```sh
printf '%s\n' 'MAC_PUBLIC_KEY' | ssh ws-ssh 'cat >/etc/wireguard/mac.pub'
ssh ws-ssh /root/repository/ws/bootstrap.sh --tags wireguard
```

With ws's key as `WS_PUBLIC_KEY`, `setup.sh` again installs the tunnel's daemon. A new Mac or a
rebuilt server changes one file on each side, and nothing in the repository.

The daemon, `ws-tunnel`, runs as root from launchd, from boot and before any login, through
`/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist`. Its functions, `tunnel-conf.sh` and
`tunnel-paths.sh`, sit beside it in `/usr/local/libexec/ws-tunnel`, as root's copies too. It runs
root's copies of Homebrew's `wg` and `wireguard-go`, which sit there as well. The user owns
Homebrew's prefix, so a root daemon that ran Homebrew's files would let anything that runs as the
user become root. A `brew upgrade` reaches the copies at the next run of `setup.sh`, which restarts
the daemon where a copy changed.

The daemon's settings are in `/etc/ws-tunnel`, which only root reads. They are the Mac's key,
`tunnel.conf` and `routes.d`, below. `setup.sh` copies `tunnel.conf` from `mac.conf`, so that the
daemon reads no file the user can write. macOS lists the daemon under System Settings › General ›
Login Items & Extensions, where "Allow in the Background" must stay on.

The tunnel's end on the Mac is `10.99.0.2`, on a utun of wireguard-go's. The daemon points it at
the first of these that answers, each on UDP 51820:

- the cable, `10.77.0.1`, where one of the Mac's addresses lies in `10.77.0.0/30`;
- the LAN, `WS_LAN_ADDRESS`, where one lies in `WS_LAN`;
- `WS_REMOTE`, at its first IPv4 address, as the router forwards IPv4 alone.

A handshake proves a switch. The daemon removes ws's peer and adds it again with the new endpoint,
so that wireguard-go handshakes at once. It keeps the way where the handshake completes within 5 s,
or tries the next. ws follows the Mac to its new address from that handshake, and TCP inside the
tunnel sends again what the pause lost.

The daemon tries the ways again 2 s after macOS posts a network change, where the Mac's addresses
changed. It checks them every 10 s besides, in case it missed one. It pings `10.99.0.1` every 10 s
too, and three pings missed in a row have it try the ways again, as after the home's address
changed. While none answers, it tries them all every 10 s. A way that comes back while the Mac's
addresses stay as they were is taken at the next change or failure.

```sh
sudo /usr/local/libexec/ws-tunnel/ws-tunnel status
```

prints the way in use, its endpoint, the latest handshake and the bytes moved.
`/var/log/ws-tunnel.log` has a line for each switch and failure.

JetBrains Gateway reads `~/.ssh/config` through the Mac's `ssh -G`, its default, with "Parse
config file ~/.ssh/config" on in the SSH connection's settings. A connection to `ws` outlives a
switch; a second, to `ws-ssh`, serves where UDP is blocked.

The tunnel's end on the Mac runs in userspace, and wireguard-go on macOS moves one packet a read and
a write. So the tunnel may fall short of the cable's 2.5 Gbit/s. To measure it, on ws, install
iperf3 from apt, let its port in on the tunnel and the cable for the test alone, and run the
server:

```sh
ufw allow in on wg0 to any port 5201 proto tcp
ufw allow in on CABLE_INTERFACE to any port 5201 proto tcp
iperf3 -s
```

then, on the Mac, compare `iperf3 -c 10.99.0.1` with `iperf3 -c 10.77.0.1`, each with `-R` too, and
delete the two rules after. Bulk copies can go to `ws-ssh`, outside the tunnel, which is the cable
when the Mac is docked: `rsync -a DIR ws-ssh:`. Besides, check `ws-tunnel status` docked, on the
home Wi-Fi and on a phone's hotspot, and `ethtool CABLE_INTERFACE` on ws, for 2500Mb/s.

## Routes and resolvers of another repository

A private repository's own script for the Mac can send more networks through the tunnel, and a
domain's names to nameservers behind it. It sources this clone's `mac/lib.sh`, which changes
nothing as a script sources it. It sets `WS_CHECK` for its own `--check`, and calls:

- `ws_tunnel_routes NAME CIDR...`, which writes the IPv4 networks to
  `/etc/ws-tunnel/routes.d/NAME` and has the daemon read them again. The daemon adds them to the
  peer's AllowedIPs and routes them into its utun, without closing a connection. It leaves out for
  the time being one that holds the endpoint in use. `NAME` is of `a-z`, `0-9` and `-`. A network
  that overlaps the tunnel's `10.99.0.0/24` or the cable's `10.77.0.0/30`, as `0.0.0.0/0` does, is
  refused. As in `ws_tunnel_routes example 198.51.100.0/24`.
- `ws_resolver DOMAIN ADDRESS...`, which writes `/etc/resolver/DOMAIN`, with up to 3 nameservers,
  so that macOS asks them for the domain's names (resolver(5)), and takes the file at once.
- `ws_tunnel_loaded`, which says whether the daemon is loaded, so that the script can stop with
  "run mac/setup.sh first".

With no network or address, each removes its file, `ws_resolver` only a file it wrote. Under
`WS_CHECK` they print diffs and change nothing. The files outlive the daemon. While the daemon is
stopped, the routes go with its utun, and the domain's lookups go out by the Mac's own network and
fail. ws's firewall forwards nothing, so a network behind ws needs its side set up on ws too.
