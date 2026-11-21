# The Mac

The Mac reaches the server, ws, over the cable from the 2.5GbE port of the display it docks at. It
reaches it over the LAN, or elsewhere through the router's port forwards. `mise bootstrap` sets
up its end of the cable ([The cable](#the-cable)), git ([git on the Mac](macos/git/README.md)), one host
for ssh, `ws` ([ssh on the Mac](macos/ssh/README.md)), and Zed ([Zed on the Mac](macos/zed/README.md)). It goes
through a WireGuard tunnel that ends on ws, at its slot's `10.99.N.1` whichever way the Mac
connects. The sessions of `ssh ws` and
Zed stay open as the Mac moves between the cable, the LAN and elsewhere, with a pause of a few seconds
([Zed](../server/ubuntu/zed/README.md#when-the-connection-drops)). What it gives up: `ssh ws` fails where UDP is blocked and
while the tunnel is down, and bulk copies run at wireguard-go's speed. To reach ws by hand, give ssh
a direct way: `ssh -o HostName=ADDRESS -o Port=PORT ws`, as ufw lets SSH in on every interface.
`WS_REMOTE_PORT` and the router's forward stay for that and for setup.

The Mac's side is mise's config and ws's units, as the server's is, which `go test` checks on each
pull request. It runs as the user, with sudo where it needs root. `mise bootstrap` sets up Ghostty,
cld and a zsh hook ([Ghostty and cld on the Mac](macos/ghostty/README.md)). It turns the firewall
on and has it block all incoming, as the server reaches the Mac past no router
([the firewall](macos/firewall/README.md)). mise installs Ghostty, Zed
and a font as casks, and gh, fnox and age-plugin-yubikey by its lock, with no Homebrew
([the runbook](runbook.md)). The runbook puts `ws` in `~/.local/bin` (put it on `PATH` by hand)
from a release, checked against its `ws.sha256` ([Releases](../../.github/workflows/release.md)),
and clones the release's tree into `/opt/ws`, where the units' specs and files are. It links the
tree's plugin where mise finds it. Write the values below in `values.yml` first ([The
two files](values/README.md#the-two-files)), and trust the tree once
([The bootstrap](runbook.md#the-bootstrap)). Then:

```sh
MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap
```

It sets up every role of the Mac, on the tree in `/opt/ws`, from the side's run directory,
`roles/client` ([The Mac's frame](macos/frame/README.md)).
Its units read as the user, and ask sudo for a change alone. `ws client` holds the Mac's
commands, darwin alone, which the units and the tunnel's daemons run. The user runs
`ws client secrets init` too, which sets up the secrets repository
([init](macos/secrets/README.md#ws-client-secrets-init)), and
`ws client secrets add NAME`, which adds a server to it
([add](macos/secrets/README.md#ws-client-secrets-add)). `ws version` prints its version.

## What the Mac reports

`MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap plan` shows what the units would change, and
`ws client servers` each server's tunnel. A cable's service that is disabled, which no run
turns on, draws a warning from `ws client cable` and its `--check` alone: a bootstrap shows no
check's output. No unit reads a private key: the key's unit looks for the file alone.

## The Mac's values.yml

The repository names no server, so the values go in two YAML files of `KEY: value` lines that the script
reads without running them. `/etc/ws/values.yml` holds the Mac's keys: `WS_GIT_SIGNING_PUB_KEY`,
`WS_IDENTITY_PUB_KEY` and `WS_IDENTITY_AGENT`. `/etc/ws/servers/ws/values.yml` holds every other key,
and a key in the other file is refused. Both are root's, written through sudo. A `WS_VALUES` file
outside `/etc/ws` is the user's. [The two files](values/README.md#the-two-files) shows both.
`--server NAME` picks the server that a `ws client` command acts on, and may be left out
where there is one ([The server's name](values/README.md#the-servers-name)). `remove`,
`secrets add` and `secrets init` refuse it: remove and add take the server's name as their
argument, and the secrets repository serves every server.

`WS_PUBLIC_KEY` is ws's WireGuard public key, which the `tunnel` role prints: [The Mac's
tunnel](#the-macs-tunnel), below. `WS_LAN` is the home LAN, and `WS_LAN_ADDRESS` ws's address inside
it, the router's static lease for it. The script stops on one outside, and on the LAN's network or
broadcast address. `WS_REMOTE` is the router's name on the internet, and `WS_REMOTE_PORT` the port
it forwards to ws's 22. The tunnel takes UDP 51820 every way, which the router forwards too;
`WS_TUNNEL_PORT` sets another, which wg0 listens on. A way without its values is left out. The
router's side is in [The router](../server/ubuntu/cable/router.md). `WS_PUBLIC_KEY` goes in last,
once ws has the Mac's key, in the order of [The Mac's tunnel](#the-macs-tunnel). `WS_CABLE_SERVICE`
names the network service of the display's adapter, and `WS_CABLE_PORT` ws's port of the cable: [The
cable](#the-cable). `WS_GIT_SIGNING_PUB_KEY` is the `.pub` of the key that signs git's commits on
the Mac, by its path from `/` or `~/`: [git on the Mac](macos/git/README.md). `WS_IDENTITY_PUB_KEY` is the
`.pub` of the key that ssh offers ws, and `WS_IDENTITY_AGENT` the socket of its agent: [ssh's
hosts](#sshs-hosts). The bootstrap makes `/etc/ws` through sudo, at mode 755, for the tunnel's key,
git's files and ssh's hosts.

ssh logs in to every server as root, so no key names a user. A key with an empty value, as
`WS_REMOTE: ""`, takes its default, as a missing line does: 51820 for `WS_TUNNEL_PORT`.
`WS_IDENTITY_PUB_KEY` has none: the bootstrap stops without it
([ssh's hosts](#sshs-hosts)). Nor has `WS_GIT_SIGNING_PUB_KEY` ([git on the Mac](macos/git/README.md)).

`WS_TUNNEL_NETWORKS` sends more IPv4 networks through the tunnel, split by blanks. `WS_RESOLVERS`
sends a domain's names to nameservers behind it, in `DOMAIN=ADDRESS` pairs split by blanks. Each
domain comes once and in lower case, with up to 3 addresses split by commas. For each server with
`WS_PUBLIC_KEY`, the networks' unit runs `ws client networks`, which hands both to the code
of `ws client tunnel routes` and `ws client resolver`, whose caveats hold ([their
section](#routes-and-resolvers-of-another-repository)). The networks go in the server's
`/etc/ws/tunnel/NAME/routes.d/values`, a name no other script may take, and launchd's unit has the
daemons read them. Each resolver starts with ws's `Written by` line, which names `ws client
networks`, so a domain whose file is another script's, or written by hand, stops the run. To move
a domain from such a script, have it remove its file first, with `ws client resolver DOMAIN`
and no address. A key emptied or removed takes back what earlier runs wrote, and only that, but a
domain another server names stays. Its `--check` names each change, or that the server's
`routes.d` is missing. Root's files go through sudo, by the kinds' code that writes the server's.
Sudo never changes a file in the home, nor one a link names. A file there that the user
may not change stops the run: sudo would leave root's files where the user's go.

## ssh's hosts

`mise -C /opt/ws/roles/client bootstrap` writes `/etc/ws/ssh_config`, root's and 644, with a host
for each server through its tunnel, and puts its Include first in `~/.ssh/config` ([ssh on the
Mac](macos/ssh/README.md)). ssh offers ws the key of `WS_IDENTITY_PUB_KEY` alone. Failed logins over
the tunnel ban the Mac's `10.99.N.2` for an hour: `ssh -o HostName=ADDRESS -o Port=PORT ws` comes
from another address, and from there `fail2ban-client set sshd unbanip 10.99.N.2` lifts the ban.

## The cable

The Mac's end of the cable is the display's adapter, whose network service `WS_CABLE_SERVICE` names,
as `networksetup -listallnetworkservices` prints it. Where `networksetup -getinfo` shows other
settings, the cable's unit runs `ws client cable`, which runs `sudo networksetup -setmanual
SERVICE 10.77.0.2 255.255.255.252`: with no router argument, the service gets none. Its `--check`
fails, naming that command. The command stops on a service the list lacks, or one with a router
outside `10.77.0.0/30`, as an uplink has. A disabled service, which the list prints with a `*` first
and the key names without it, plans no change. So no bootstrap shows it: `ws client cable
--check` warns of it while it stays off. Without the key, it does nothing, and the server's tunnel
does not try the cable: the cable reaches the one server that names it. `-setmanual` replaces the
service's settings, whatever they were, and emptying the key leaves them. To give the service DHCP
again: `sudo networksetup -setdhcp SERVICE`.

An address, mask or router that `getinfo` omits, as it may undocked, counts as set. Where
`-setmanual` leaves other settings, the command names them and stops. Then set it by hand: in System
Settings › Network, the service › Details › TCP/IP, Configure IPv4 Manually: IP address `10.77.0.2`,
mask `255.255.255.252`, no router. Or empty the key: the tunnel and ssh then skip the cable.

Set the key once ws's end is up ([cable](../server/ubuntu/cable/README.md)). Then, docked, check once that the uplink
keeps the default route:

```sh
route -n get default | grep interface
```

It must name the uplink's device, which `networksetup -listnetworkserviceorder` pairs with its
service. Where it names the adapter's, move the service below the uplink's: Set Service Order, in
the menu of System Settings › Network, or `networksetup -ordernetworkservices`, given every service
in order.

## The Mac's tunnel

The keys come first, in this order. A run of the `tunnel` role makes ws's key and prints its
public key. `mise bootstrap` makes the Mac's key, which only root reads, through `ws client
tunnel key`, and puts its public key in `/etc/ws/tunnel.pub`, readable by all. These two commands
make the Mac the server's peer, with that key as MAC_PUBLIC_KEY. They go over a direct way, as
`ssh NAME`, the server's host, needs this done first. ADDRESS and PORT are that way:

```sh
ssh -o HostName=ADDRESS -o Port=PORT NAME "printf '%s\n' 'MAC_PUBLIC_KEY' | tee /etc/wireguard/mac.pub >/dev/null"
ssh -o HostName=ADDRESS -o Port=PORT NAME env MISE_LOCKED=1 mise -C /opt/ws/roles/server bootstrap
```

With ws's key as `WS_PUBLIC_KEY`, the bootstrap installs the tunnel's daemon. A new
Mac or a rebuilt server changes one file on each side, and nothing in the repository. Do it by hand,
as above.

A daemon per server, `ws client tunnel run --server NAME`, runs as root from launchd, from
boot and before any login, by `/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.NAME.plist`,
which launchd's unit of the tunnel writes. It runs `/usr/local/bin/ws`, the bootstrap's copy of the
pinned ws, which is root's. A root daemon that ran a file of the user's would let anything that
runs as the user become root. The daemon holds WireGuard's device itself, so it needs no `wg` or
`wireguard-go`. launchd's unit restarts each where `/etc/ws/tunnel.pub` changed, and sends each a
SIGHUP where a server's `tunnel.conf` or `routes.d` changed.

The settings are in `/etc/ws/tunnel`, root's and readable by all but the Mac's key, which only root
reads. `ws client tunnel conf` writes a directory `NAME` per server with `WS_PUBLIC_KEY`, with
its `tunnel.conf`, from the server's file, and its `routes.d` ([the daemons](macos/tunnel/README.md)), so
the daemon reads no file the user can write. In System Settings › General › Login Items &
Extensions, its "Allow in the Background" must stay on.

A server with no directory beside `values.yml` has its daemon removed at the next bootstrap, by
`ws client tunnel conf`: booted out where loaded, then `/etc/ws/tunnel/NAME`, its state file
and log. launchd's unit removes its plist. Nothing goes where `values.yml`'s directory lists no
server.

The tunnel's ends are `10.99.N.2` on the Mac, on a utun of wireguard-go's, and `10.99.N.1` on ws, N
being `WS_SLOT`. A moved slot restarts the daemon. Name the pair in ws's values.yml ([The
tunnel](../server/ubuntu/tunnel/README.md#the-tunnel)). The daemon points the Mac's end at the
first of these that answers, each on UDP 51820:

- the cable, `10.77.0.1`, where one of the Mac's addresses lies in `10.77.0.0/30`;
- the LAN, `WS_LAN_ADDRESS`, where one lies in `WS_LAN`;
- `WS_REMOTE`, at its first IPv4 address, as the router forwards IPv4 alone.

A handshake proves a switch. The daemon removes ws's peer and adds it again with the new endpoint,
so that wireguard-go handshakes at once. It keeps the way where the handshake completes within 5 s,
or tries the next. ws follows the Mac to its new address from that handshake, and TCP inside the
tunnel sends again what the pause lost.

The daemon tries the ways again 2 s after macOS posts a network change, where the Mac's addresses
changed. Besides, 10 s after each check ends, it checks them again, in case it missed one, and
pings `10.99.N.1`. Three pings missed in a row have it try the ways again, as after the home's
address changed. While none answers, it tries them all again 10 s after each round, which takes up
to 5 s a way. A way that comes back while the Mac's addresses stay as they were is taken at the
next change or failure.

```sh
sudo /usr/local/bin/ws client tunnel status [--server NAME]
```

prints the way in use, its endpoint, the latest handshake and the bytes moved, of the one daemon
installed. `/var/log/ws-tunnel.NAME.log` has a line for each switch and failure.

A connection to `ws` outlives a switch. Zed has [its page](macos/zed/README.md).

The tunnel's end on the Mac runs in userspace, and wireguard-go on macOS moves one packet a read and
a write. So the tunnel may fall short of the cable's 2.5 Gbit/s. To measure it, on ws, install
iperf3 from apt, let its port in on the tunnel and the cable for the test alone, and run the server:

```sh
ufw allow in on wg0 to any port 5201 proto tcp
ufw allow in on CABLE_INTERFACE to any port 5201 proto tcp
iperf3 -s
```

then, on the Mac, compare `iperf3 -c 10.99.N.1` with `iperf3 -c 10.77.0.1`, each with `-R` too, and
delete the two rules after. Bulk copies run at that speed: for faster ones, give rsync a direct way,
such as `rsync -e 'ssh -o HostName=ADDRESS -o Port=PORT' -a DIR ws:`. Besides, check
`ws client tunnel status` docked, on the home Wi-Fi and on a phone's hotspot, and `ethtool
CABLE_INTERFACE` on ws, for 2500Mb/s.

## Routes and resolvers of another repository

A private repository's own script for the Mac can send more networks through the tunnel, and a
domain's names to nameservers behind it. It calls `ws`, which changes only what you give it.
`--check` makes each print its diffs and change nothing:

- `ws client tunnel routes [--check] --server SERVER NAME CIDR...`, which writes the IPv4
  networks to `/etc/ws/tunnel/SERVER/routes.d/NAME` and has the daemons read them again. SERVER's
  daemon adds them to the peer's AllowedIPs and routes them into its utun, without closing a
  connection. It leaves out for the time being one that holds the endpoint in use, or one that
  holds a way's endpoint while none answers. `NAME` is of `a-z`, `0-9` and `-`, `values`
  aside. A network that overlaps the tunnels' `10.99.0.0/20` or the cable's `10.77.0.0/30`, as
  `0.0.0.0/0` does, is refused. As in `ws client tunnel routes --server ws example
  198.51.100.0/24`. - `ws client resolver [--check] DOMAIN ADDRESS...`, which writes
  `/etc/resolver/DOMAIN`, with up to 3 nameservers, so that macOS asks them for the domain's names
  (resolver(5)), and takes the file at once. - `ws client tunnel loaded`, which says whether
  the daemon is loaded, so that the script can stop with "run mise -C /opt/ws/roles/client bootstrap
  first".

With no network or address, each removes its file, `ws client resolver` only a file it wrote.
`tunnel routes` signals the daemons itself, but launchd's unit keeps the sum of `routes.d` in its
stamp. So the next bootstrap plans a reload and the stamp's write through sudo, which asks a
password.
The files outlive the daemon. While the daemon is stopped, the routes go with its utun, and the
domain's lookups go out by the Mac's own network and fail. ufw on ws forwards from the tunnel only
ping, and what the `ikeward` role lets through to its VPN ([Forwarding](../server/ubuntu/ikeward/README.md#forwarding)). Any
other network behind ws needs its side set up on ws too. A network the Mac is on, inside one sent
into the tunnel, keeps its part off the tunnel: macOS routes the narrower network to the Mac's own
interface.
