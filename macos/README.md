# The Mac

The Mac reaches the server, ws, over the cable from the 2.5GbE port of the display it docks at. It
reaches it over the LAN, or elsewhere through the router's port forwards. `mise bootstrap` sets
up its end of the cable ([The cable](#the-cable)), git ([git on the Mac](roles/git/README.md)), one host
for ssh, `ws` ([ssh on the Mac](roles/ssh/README.md)), and Zed ([Zed on the Mac](roles/zed/README.md)). It goes
through a WireGuard tunnel that ends on ws, at its slot's `10.99.N.1` whichever way the Mac
connects. The sessions of `ssh ws` and
Zed stay open as the Mac moves between the cable, the LAN and elsewhere, with a pause of a few seconds
([Zed](../ubuntu/roles/zed/README.md#when-the-connection-drops)). What it gives up: `ssh ws` fails where UDP is blocked and
while the tunnel is down, and bulk copies run at wireguard-go's speed. To reach ws by hand, give ssh
a direct way: `ssh -o HostName=ADDRESS -o Port=PORT ws`, as sshd takes a login on every interface.
`WS_REMOTE_PORT` and the router's forward stay for that and for setup. A guest has no direct way,
as nothing forwards TCP to it: the Mac reaches its host first, which runs the guest's commands.

The Mac's side is mise's config and ws's units, as the server's is, which `go test` checks on each
pull request. It runs as the user, with sudo where it needs root. `mise bootstrap` sets up Ghostty,
cld and a zsh hook ([Ghostty and cld on the Mac](roles/ghostty/README.md)). It turns the firewall
on and has it block all incoming, as the server reaches the Mac past no router
([the firewall](roles/firewall/README.md)). mise installs Ghostty, Zed
and a font as casks, and gh, fnox and age-plugin-yubikey by its lock, with no Homebrew
([the runbook](runbook.md)). ws's units install .NET's SDKs ([dotnet](roles/dotnet/README.md)).
The runbook runs `install.sh`, which puts `ws` in `/usr/local/bin` from a release, checked against
its `ws.sha256` ([Releases](../.github/workflows/release.md)).
Write each server's values below first, and the Mac's `values.yml`, which stands empty on a new
Mac ([The two files](values/README.md#the-two-files)). Then:

```sh
ws macos setup
```

Setup takes the steps before mise ([Setup](#setup)). Then it runs mise's bootstrap, which sets up
every role of the Mac. It runs on the tree in `/opt/ws`, from the side's run directory,
`macos` ([The Mac's frame](roles/frame/README.md)).
Its units read as the user, and ask sudo for a change alone. `ws macos` holds the Mac's
commands, darwin alone, which the units and the tunnel's daemons run. The user runs
`ws macos secrets init` too, which sets up the secrets repository
([init](roles/secrets/README.md#ws-macos-secrets-init)), and
`ws macos secrets add NAME`, which adds a server to it
([add](roles/secrets/README.md#ws-macos-secrets-add)). `ws version` prints its version.

## Setup

`ws macos setup` runs as the user on macOS on arm64, and refuses root and any other system. The
steps are those of the server's setup, which `internal/setup` holds, with the Mac's paths
([The bootstrap](runbook.md#the-bootstrap)).

- It checks the values before any write, and the guard. A tree in `/opt/ws`, or a directory above
  it or above the plugin, that another user owns or that others may write is refused. mise runs the
  tree's config as the user, and ignores a plugin below such a directory. A directory that a Mac
  lacks, such as `/opt` or `/usr/local/share`, is made root's at 755.
- The command line tools come first, as git needs them: its shim opens Apple's dialog where they
  are missing. Setup runs `xcode-select --install`, not `softwareupdate`, whose marker file and
  labels Apple does not document. A plan with the tools due plans no more.
- mise comes from its release at the version ws pins, and its SHA-256 for `macos-arm64` is in Go
  ([mise.go](../internal/setup/mise.go)). sudo writes the file, once the body matches.
- The tree is root's. Setup reads its tag and status as the user by `git -c
  safe.directory=/opt/ws`, as git refuses a repository that another user owns. It runs git
  through sudo to fetch, check out and clean, and the plugin's link too.
- Where `values.yml` names no `WS_IDENTITY_PUB_KEY`, setup installs Secretive by mise's pin before
  the bootstrap, checks it and waits for a key made in it, which it names in `values.yml`
  ([Secretive](roles/ssh/README.md#secretive)). `--key NAME` takes a key that Secretive held
  before. A plan with the key due plans no more.
- A second run reads as the user, writes nothing and asks no password. `--in-place` and `--plan`
  work as on the server ([The server's runbook](../ubuntu/runbook.md)).

## What the Mac reports

`ws macos setup --plan` shows what setup and the units would change, and
`ws macos servers` each server's tunnel. A cable's service that is disabled, which no run
turns on, draws a warning from `ws macos cable` and its `--check` alone: a bootstrap shows no
check's output. No unit reads a private key: the key's unit looks for the file alone.

## The Mac's values.yml

The repository names no server, so the values go in two YAML files of `KEY: value` lines that the script
reads without running them. `/etc/ws/values.yml` holds the Mac's keys: `WS_GIT_SIGNING_PUB_KEY`,
`WS_IDENTITY_PUB_KEY` and `WS_IDENTITY_AGENT`. `/etc/ws/servers/ws/values.yml` holds every other key,
and a key in the other file is refused. Both are root's, written through sudo. A `WS_VALUES` file
outside `/etc/ws` is the user's. [The two files](values/README.md#the-two-files) shows both.
`--server NAME` picks the server that a `ws macos` command acts on, and may be left out
where there is one ([The server's name](values/README.md#the-servers-name)). `remove`,
`secrets add` and `secrets init` refuse it: remove and add take the server's name as their
argument, and the secrets repository serves every server.

`WS_PUBLIC_KEY` is ws's WireGuard public key, which the `tunnel` role prints: [The Mac's
tunnel](#the-macs-tunnel), below. `WS_LAN` is the home LAN, and `WS_LAN_ADDRESS` ws's address inside
it, the router's static lease for it. The script stops on one outside, and on the LAN's network or
broadcast address. `WS_REMOTE` is the router's name on the internet, and `WS_REMOTE_PORT` the port
it forwards to ws's 22. The tunnel takes UDP 51820 every way, which the router forwards too;
`WS_TUNNEL_PORT` sets another, which wg0 listens on. A way without its values is left out. The
router's side is in [The router](../ubuntu/roles/cable/router.md). `WS_PUBLIC_KEY` goes in last,
once ws has the Mac's key, in the order of [The Mac's tunnel](#the-macs-tunnel). `WS_CABLE_SERVICE`
names the network service of the display's adapter, and `WS_CABLE_PORT` ws's port of the cable: [The
cable](#the-cable). `WS_GIT_SIGNING_PUB_KEY` is the `.pub` of the key that signs git's commits on
the Mac, by its path from `/` or `~/`: [git on the Mac](roles/git/README.md). `WS_IDENTITY_PUB_KEY` is the
`.pub` of the key that ssh offers ws, and `WS_IDENTITY_AGENT` the socket of its agent: [ssh's
hosts](#sshs-hosts). `WS_HOST` makes a server a guest of another, its host, and takes the host's
LAN, address and router for the guest's ([A guest](values/README.md#a-guest)). The bootstrap makes
`/etc/ws` through sudo, at mode 755, for the tunnel's key, git's files and ssh's hosts.

ssh logs in to every server as root, so no key names a user. A key with an empty value, as
`WS_REMOTE: ""`, takes its default, as a missing line does: 51820 for `WS_TUNNEL_PORT`.
`WS_IDENTITY_PUB_KEY` has none: the bootstrap stops without it
([ssh's hosts](#sshs-hosts)). Nor has `WS_GIT_SIGNING_PUB_KEY` ([git on the Mac](roles/git/README.md)).

`WS_TUNNEL_NETWORKS` sends more IPv4 networks through the tunnel, split by blanks. `WS_RESOLVERS`
sends a domain's names to nameservers behind it, in `DOMAIN=ADDRESS` pairs split by blanks. Each
domain comes once and in lower case, with up to 3 addresses split by commas. For each server with
`WS_PUBLIC_KEY`, the networks' unit runs `ws macos networks`, which hands both to the code
of `ws macos tunnel routes` and `ws macos resolver`, whose caveats hold ([their
section](#routes-and-resolvers-of-another-repository)). The networks go in the server's
`/etc/ws/tunnel/NAME/routes.d/values`, a name no other script may take, and launchd's unit has the
daemons read them. Each resolver starts with ws's `Written by` line, which names `ws macos
networks`, so a domain whose file is another script's, or written by hand, stops the run. To move
a domain from such a script, have it remove its file first, with `ws macos resolver DOMAIN`
and no address. A key emptied or removed takes back what earlier runs wrote, and only that, but a
domain another server names stays. Its `--check` names each change, or that the server's
`routes.d` is missing. Root's files go through sudo, by the kinds' code that writes the server's.
Sudo never changes a file in the home, nor one a link names. A file there that the user
may not change stops the run: sudo would leave root's files where the user's go.

## ssh's hosts

`mise -C /opt/ws/macos bootstrap` writes `/etc/ws/ssh_config`, root's and 644, with a host
for each server through its tunnel, and puts its Include first in `~/.ssh/config` ([ssh on the
Mac](roles/ssh/README.md)). ssh offers ws the key of `WS_IDENTITY_PUB_KEY` alone. Failed logins over
the tunnel ban the Mac's `10.99.N.2` for an hour: `ssh -o HostName=ADDRESS -o Port=PORT ws` comes
from another address, and from there `fail2ban-client set sshd unbanip 10.99.N.2` lifts the ban.

## The cable

The Mac's end of the cable is the display's adapter, whose network service `WS_CABLE_SERVICE` names,
as `networksetup -listallnetworkservices` prints it. Where `networksetup -getinfo` shows other
settings, the cable's unit runs `ws macos cable`, which runs `sudo networksetup -setmanual
SERVICE 10.77.0.2 255.255.255.252`: with no router argument, the service gets none. Its `--check`
fails, naming that command. The command stops on a service the list lacks, or one with a router
outside `10.77.0.0/30`, as an uplink has. A disabled service, which the list prints with a `*` first
and the key names without it, plans no change. So no bootstrap shows it: `ws macos cable
--check` warns of it while it stays off. Without the key, it does nothing, and the server's tunnel
does not try the cable: the cable reaches the one server that names it, and that server's guests.
`-setmanual` replaces the service's settings, whatever they were, and emptying the key leaves them.
To give the service DHCP again: `sudo networksetup -setdhcp SERVICE`.

An address, mask or router that `getinfo` omits, as it may undocked, counts as set. Where
`-setmanual` leaves other settings, the command names them and stops. Then set it by hand: in System
Settings › Network, the service › Details › TCP/IP, Configure IPv4 Manually: IP address `10.77.0.2`,
mask `255.255.255.252`, no router. Or empty the key: the tunnel and ssh then skip the cable.

Set the key once ws's end is up ([cable](../ubuntu/roles/cable/README.md)). Then, docked, check once that the uplink
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
public key. `mise bootstrap` makes the Mac's key, which only root reads, through `ws macos
tunnel key`, and puts its public key in `/etc/ws/tunnel.pub`, readable by all. These two commands
make the Mac the server's peer, with that key as MAC_PUBLIC_KEY. They go over a direct way, as
`ssh NAME`, the server's host, needs this done first. ADDRESS and PORT are that way:

```sh
ssh -o HostName=ADDRESS -o Port=PORT NAME "printf '%s\n' 'MAC_PUBLIC_KEY' | tee /etc/wireguard/mac.pub >/dev/null"
ssh -o HostName=ADDRESS -o Port=PORT NAME ws ubuntu guest setup
```

A guest's tunnel ends on its host, which holds the interface's key and its peers. So the same two
commands go to the host, with the guest's slot N: the Mac's key goes in `/etc/wireguard/wgN/mac.pub`
there, and the host's setup makes the interface. Its check prints the interface's public key, which
is the guest's `WS_PUBLIC_KEY`. ADDRESS and PORT reach the host, as the guest has no address:

```sh
ssh -o HostName=ADDRESS -o Port=PORT HOST "install -d -m 700 /etc/wireguard/wgN && printf '%s\n' 'MAC_PUBLIC_KEY' | tee /etc/wireguard/wgN/mac.pub >/dev/null"
ssh -o HostName=ADDRESS -o Port=PORT HOST ws ubuntu host setup
ssh -o HostName=ADDRESS -o Port=PORT HOST ws ubuntu host tunnel guests check
```

With ws's key as `WS_PUBLIC_KEY`, the bootstrap installs the tunnel's daemon. A new
Mac or a rebuilt server changes one file on each side, and nothing in the repository. Do it by hand,
as above.

A daemon per server, `ws macos tunnel run --server NAME`, runs as root from launchd, from
boot and before any login, by `/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.NAME.plist`,
which launchd's unit of the tunnel writes. It runs `/usr/local/bin/ws`, the one ws of the Mac,
which is root's. A root daemon that ran a file of the user's would let anything that
runs as the user become root. The daemon holds WireGuard's device itself, so it needs no `wg` or
`wireguard-go`. launchd's unit restarts each where `/etc/ws/tunnel.pub` changed, and sends each a
SIGHUP where a server's `tunnel.conf` or `routes.d` changed.

The settings are in `/etc/ws/tunnel`, root's and readable by all but the Mac's key, which only root
reads. `ws macos tunnel conf` writes a directory `NAME` per server with `WS_PUBLIC_KEY`, with
its `tunnel.conf`, from the server's file, and its `routes.d` ([the daemons](roles/tunnel/README.md)), so
the daemon reads no file the user can write. In System Settings › General › Login Items &
Extensions, its "Allow in the Background" must stay on.

A server with no directory beside `values.yml` has its daemon removed at the next bootstrap, by
`ws macos tunnel conf`: booted out where loaded, then `/etc/ws/tunnel/NAME`, its state file
and log. launchd's unit removes its plist. Nothing goes where `values.yml`'s directory lists no
server.

The tunnel's ends are `10.99.N.2` on the Mac, on a utun of wireguard-go's, and `10.99.N.1` on ws, N
being `WS_SLOT`. A moved slot restarts the daemon. Name the pair in ws's values.yml ([The
tunnel](../ubuntu/roles/tunnel/README.md#the-tunnel)). The daemon points the Mac's end at the
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
sudo /usr/local/bin/ws macos tunnel status [--server NAME]
```

prints the way in use, its endpoint, the latest handshake and the bytes moved, of the one daemon
installed. `/var/log/ws-tunnel.NAME.log` has a line for each switch and failure.

A connection to `ws` outlives a switch. Zed has [its page](roles/zed/README.md).

The tunnel's end on the Mac runs in userspace, and wireguard-go on macOS moves one packet a read and
a write. So the tunnel may fall short of the cable's 2.5 Gbit/s. To measure it, use a guest,
which holds no filter ([No filter](../ubuntu/roles/security/README.md#no-filter)). Install iperf3
there from apt, and run the server on the tunnel's address alone, so that no other network reaches
it:

```sh
iperf3 -s -B 10.99.N.1
```

then, on the Mac, run `iperf3 -c 10.99.N.1`, and with `-R` too. The cable ends at the host, whose
ufw lets in no TCP but SSH ([The host's firewall](../ubuntu/roles/security/README.md#the-hosts-firewall)).
For the same two runs against `10.77.0.1`, install iperf3 there, let its port in on the cable for
the runs alone, and delete the rule after:

```sh
ufw allow in on CABLE_INTERFACE to 10.77.0.1 port 5201 proto tcp
iperf3 -s -B 10.77.0.1
ufw delete allow in on CABLE_INTERFACE to 10.77.0.1 port 5201 proto tcp
```

Bulk copies run at that speed: for faster ones, give rsync a direct way, such as
`rsync -e 'ssh -o HostName=ADDRESS -o Port=PORT' -a DIR ws:`. Besides, check `ws macos tunnel status`
docked, on the home Wi-Fi and on a phone's hotspot, and `ethtool CABLE_INTERFACE` on the host, for
2500Mb/s.

## Routes and resolvers of another repository

A private repository's own script for the Mac can send more networks through the tunnel, and a
domain's names to nameservers behind it. It calls `ws`, which changes only what you give it.
`--check` makes each print its diffs and change nothing:

- `ws macos tunnel routes [--check] --server SERVER NAME CIDR...`, which writes the IPv4
  networks to `/etc/ws/tunnel/SERVER/routes.d/NAME` and has the daemons read them again. SERVER's
  daemon adds them to the peer's AllowedIPs and routes them into its utun, without closing a
  connection. It leaves out for the time being one that holds the endpoint in use, or one that
  holds a way's endpoint while none answers. `NAME` is of `a-z`, `0-9` and `-`, `values`
  aside. A network that overlaps the tunnels' `10.99.0.0/20` or the cable's `10.77.0.0/30`, as
  `0.0.0.0/0` does, is refused. As in `ws macos tunnel routes --server ws example
  198.51.100.0/24`. - `ws macos resolver [--check] DOMAIN ADDRESS...`, which writes
  `/etc/resolver/DOMAIN`, with up to 3 nameservers, so that macOS asks them for the domain's names
  (resolver(5)), and takes the file at once. - `ws macos tunnel loaded`, which says whether
  the daemon is loaded, so that the script can stop with "run mise -C /opt/ws/macos bootstrap
  first".

With no network or address, each removes its file, `ws macos resolver` only a file it wrote.
`tunnel routes` signals the daemons itself, but launchd's unit keeps the sum of `routes.d` in its
stamp. So the next bootstrap plans a reload and the stamp's write through sudo, which asks a
password.
The files outlive the daemon. While the daemon is stopped, the routes go with its utun, and the
domain's lookups go out by the Mac's own network and fail. ws forwards what the tunnel sends to its
VPN, where the `ikeward` role sets that up ([Forwarding](../ubuntu/roles/ikeward/README.md#forwarding)). Any
other network behind ws needs its side set up on ws too. A network the Mac is on, inside one sent
into the tunnel, keeps its part off the tunnel: macOS routes the narrower network to the Mac's own
interface.
