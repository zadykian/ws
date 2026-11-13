# The Mac

The Mac reaches the server, ws, over the cable from the 2.5GbE port of the display it docks at. It
reaches it over the LAN, or elsewhere through the router's port forwards. `ws client macos apply`
sets up its end of the cable ([The cable](#the-cable)), git ([git on the Mac](git/README.md)), Zed
([Zed on the Mac](zed/README.md)) and one host for ssh, `ws`. It goes through a WireGuard tunnel that
ends on ws, at `10.99.0.1` whichever way the Mac connects. The sessions of `ssh ws` and Zed stay
open as the Mac moves between the cable, the LAN and elsewhere, with a pause of a few seconds
([Zed](../../server/ubuntu/zed/README.md#when-the-connection-drops)). What it gives up: `ssh ws` fails where UDP is blocked and
while the tunnel is down, and bulk copies run at wireguard-go's speed. To reach ws by hand, give ssh
a direct way: `ssh -o HostName=ADDRESS -o Port=PORT ws`, as ufw lets SSH in on every interface.
`WS_REMOTE_PORT` and the router's forward stay for that and for setup.

The Mac's side is Go: one binary, `ws`, of isolated steps, which `go test` checks on each pull
request. It runs as the user, with sudo where it needs root. It sets up Ghostty, cld and a zsh
hook ([Ghostty and cld on the Mac](ghostty/README.md)). It installs Ghostty, Zed and a font from
Homebrew, [installed](https://brew.sh) first. `install.sh` puts `ws` in `~/.local/bin` (put it on
`PATH` by hand) from a release ([Releases](../../../engine/core/update/README.md)). `ws` embeds the files it installs, so it
needs no clone. Then:

```sh
ws client macos apply
```

`ws client macos wizard`, in its place, asks first for the values below ([The Mac's
wizard](stages/README.md)), and `ws client macos configure` changes them later on a form, with ws's
secrets and host_vars ([The Configure screen](../../../engine/tui/internal/configure/README.md)). `ws client macos
apply --only STEP` runs one step, and `--check` shows each change and makes none. At a terminal it
draws [the run screen](../../../engine/tui/README.md#the-run-screen), past [the list of
steps](../../../engine/tui/README.md#the-list-of-steps) where it gets neither, and `--plain` keeps
its lines.
`ws client macos state` reports what a check reads, and changes nothing
([State](../../../README.md#state)). `ws client macos` holds the Mac's commands ([the
secrets](../../server/ubuntu/secrets/README.md) too), darwin alone. `ws version` prints its version,
and `ws update` replaces it with the latest release ([Update](../../../engine/core/update/README.md#update)).

## What the Mac reports

`ws client macos state` lists the casks, gh, cld, the cable's service, the tunnel's daemon, and the
files and directories ws writes. A file that differs from ws's, or holds no `Written by` line, is a
conflict. So is a cask that Homebrew holds with its files gone, and a cable's service that is
disabled, which no run turns on. No report reads a private key: `private.key` shows by its owner and
mode, a signing key by its `.pub`. `~/.ssh/config`, `~/.gitconfig` and Ghostty's config show whether
they hold ws's line, and none of their text.

## mac.conf

The repository names no server, so the values go in two files of `KEY=value` lines that the script
reads without running them. `~/.config/ws/mac.conf` holds the Mac's keys, which `WS_GIT_NAME`,
`WS_GIT_EMAIL`, `WS_GIT_SIGNING_PUB_KEY`, `WS_IDENTITY_PUB_KEY` and `WS_IDENTITY_AGENT` are.
`~/.config/ws/servers/ws/server.conf` holds every other key. A key in the other file is refused.
[The two files](conf/README.md#the-two-files) shows both.
`--server NAME`, on every `ws client macos` command, picks the server, which may be left out where
there is one ([The server's name](conf/README.md#the-servers-name)).

`WS_PUBLIC_KEY` is ws's WireGuard public key, which the `wireguard` role prints:
[The Mac's tunnel](#the-macs-tunnel), below. `WS_LAN` is the home LAN, and `WS_LAN_ADDRESS` ws's
address inside it, the router's static lease for it. The script stops on one outside, and on the
LAN's network or broadcast address. `WS_REMOTE` is the router's name on the internet, and
`WS_REMOTE_PORT` the port it forwards to ws's 22. The tunnel takes UDP 51820 every way, which the
router forwards too; `WS_PORT` sets another. A way without its values is left out. The router's
side is in [The router](../../server/ubuntu/network/router.md). `WS_PUBLIC_KEY` goes in last, once ws has the Mac's key, in the
order of [The Mac's tunnel](#the-macs-tunnel). `WS_CABLE_SERVICE` names the network service of the
display's adapter, in quotes where it holds a blank, and `WS_CABLE_PORT` ws's port of the cable:
[The cable](#the-cable). `WS_GIT_NAME` and `WS_GIT_EMAIL`, which go together, are the author of
git's commits on the Mac, the name in quotes where it holds a blank. `WS_GIT_SIGNING_PUB_KEY` is the
`.pub` of the key that signs them, by its path from `/` or `~/`: [git on the Mac](git/README.md).
`WS_IDENTITY_PUB_KEY` is the `.pub` of the key that ssh offers ws, and `WS_IDENTITY_AGENT` the
socket of its agent, in quotes where it holds a blank: [ssh's hosts](#sshs-hosts). A run makes
`~/.config/ws` at mode 700, and sets that mode where it differs.

`WS_USER` is the user ssh logs in as: the server's `ws_user`, `zadykian` by default, or `root` for a
`WS_TARGET` of `cloud`, which may have no other user. The other targets refuse `WS_USER=root`. A key
with an empty value, as `WS_REMOTE=`, takes its default, as a missing line does: that user for
`WS_USER` and 51820 for `WS_PORT`. `WS_IDENTITY_PUB_KEY` has none: the script stops without it
([ssh's hosts](#sshs-hosts)). Nor has `WS_GIT_SIGNING_PUB_KEY`, which git's name needs
([git on the Mac](git/README.md)).

`WS_TUNNEL_NETWORKS` sends more IPv4 networks through the tunnel, split by blanks. `WS_RESOLVERS`
sends a domain's names to nameservers behind it, in `DOMAIN=ADDRESS` pairs split by blanks. Each
domain comes once and in lower case, with up to 3 addresses split by commas. Once launchd has the
tunnel's daemon, `ws client macos apply` hands both to the code of `ws client macos tunnel routes`
and `ws client macos resolver`, whose caveats hold ([their
section](#routes-and-resolvers-of-another-repository)). The networks go in
`/etc/ws-tunnel/routes.d/mac-conf`, a name no other script may take. Each resolver starts with ws's
`Written by` line, which names `ws client macos apply`, so a domain whose file is another script's,
or written by hand, stops the run. To move a domain from such a script, have it remove its file
first, with `ws client macos resolver DOMAIN` and no address. A key emptied or removed takes back
what earlier runs wrote, and only that. `--check` shows each change: a diff, a removal, or a line
where `routes.d` is missing. Root's files go through sudo, by the code that writes `ws server ubuntu
apply`'s. Sudo never changes a file in the home, nor one a link names. A file there that the user
may not change stops the run: sudo would leave root's files where the user's go.

## ssh's hosts

The script writes `~/.ssh/config.d/ws`, with a host for each server of `servers/`, named as the
server is, at its tunnel's `10.99.N.1` from the first run, whichever network the Mac is on. A
server's host is `ws` at `10.99.0.1` for slot 0. So `ssh NAME` fails until the keys' exchange ends
([The Mac's tunnel](#the-macs-tunnel)), where a direct way serves. It logs in as the server's
`WS_USER` with the key below, and checks the server's host key under its name (`HostKeyAlias`). A
server whose `server.conf` fails to load stops the run. A session ends after 120 s without word
from the server, far longer than a switch of the tunnel takes.

ssh offers ws one key alone (`IdentitiesOnly`): the one whose `.pub` `WS_IDENTITY_PUB_KEY` names, as
ws's copy of a key of Secretive's ([Secretive's keys](git/README.md#secretives-keys)) or a file key.
The script picks none by itself, not even the agent's sole key, as a key made for another use would
go to GitHub unasked. Without the key, or with it empty, it stops: run `ws client macos wizard`,
which asks ([The wizard](stages/README.md#the-macs-key)), or add one of the lines it gives. They name
each P-256 key that Secretive's agent lists, by its comment and SHA256 fingerprint. The agent is the
one whose socket `WS_IDENTITY_AGENT` names, else Secretive's where the `.pub` is there. Else ssh
reads the file beside the `.pub`, its passphrase kept in the Keychain. The script stops where that
`.pub` is missing or unreadable. It stops too where the `.pub` holds other than one public key's
line, as a private key or a certificate does. So does a key that ssh-keygen cannot read, or one of a
type that ws's OpenSSH lacks, as Secretive's ML-DSA keys on macOS 26.

The script puts `Include config.d/ws` first in `~/.ssh/config`, as ssh keeps the first value it
reads for each key. So neither the file's own settings nor the other files of `config.d` override
ws's. Where the line is missing, the script adds it at the top and keeps the old file as
`~/.ssh/config.TIME~`. Where the line sits below other settings, the script stops. The line also
counts by `~/.ssh/` or the home's full path, in quotes, or as the first of several paths.

Where `~/.ssh/config` is a link, as a dotfiles manager makes it, the script reads the file it points
to. It stops where the line is missing there, rather than replace the link with a file, and the
report shows the file as a conflict. Where `~/.ssh/config.d` is a link, the script warns as it
writes `config.d/ws` there. It stops where `config.d/ws` is a file it did not write, rather than
replace it. It stops where a `Host` or `Match` of `~/.ssh/config`'s own names a server, whose
settings would come after those of `config.d/ws`: remove it, as its `IdentityFile` would add a key
to ws's. The wizard refuses such a name when it asks the server's name.

ssh keeps the first `IgnoreUnknown` it reads too. `config.d/ws` needs one for the file key's
`UseKeychain`, which only Apple's ssh knows, and keeps it with an agent's key. So it carries the
patterns of every `IgnoreUnknown` in `~/.ssh/config`. Run the script again after a change to those.
A pattern in a file that `~/.ssh/config` includes must be in `~/.ssh/config` too, as the script
reads no other file.

Before it writes, the script has `ssh -G`, for each server, read the new `config.d/ws` alone.
Then it reads it first, `~/.ssh/config` next, without its own Include of the old file, and the
system's file last. Where ssh refuses either, or reads a server at another address or user than
its block sets, the script stops and changes nothing, under `--check` too. The system's file goes
through an Include there, so its relative Includes resolve under `~/.ssh`. A run changes only what
differs, so a second prints nothing but warnings once `WS_PUBLIC_KEY` is set. Until then, each run
prints the Mac's public key again, with the commands of [The Mac's tunnel](#the-macs-tunnel).
`--check` prints each change as a diff and makes none.

Through the forward, ws takes logins from the internet: by key alone, as everywhere, with fail2ban
banning the address of the network the Mac is on after 5 failed ones. Over the tunnel, ws sees
the Mac as `10.99.0.2`. Failed logins over the tunnel ban it, which shuts `ssh ws` out for an
hour. `ssh -o HostName=ADDRESS -o Port=PORT ws` comes from another address, and from there
`fail2ban-client set sshd unbanip 10.99.0.2` lifts the ban.

Unless `security_outbound` is false, ws opens no connection to the Mac over the cable, the LAN or
the tunnel ([The firewall](../../server/ubuntu/wireguard/README.md#the-firewall), [Outbound](../../server/ubuntu/security/README.md#outbound)). That
holds for its bridged containers and its users, but those who can become root. On a VM on the Mac,
DHCP and DNS still go to the Mac, which serves them.

The first login asks whether to trust ws's host key: compare its fingerprint with the one
`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` prints on ws's console. On the cable and the LAN,
macOS asks once whether Ghostty, Zed or another app may find devices on the local
network: allow it. Only a direct way needs it. ssh run from Terminal needs no such grant, and
neither does `ws` through the tunnel. The tunnel's utun is no local network, and macOS lets daemons
that run as root reach one anyway (Apple's TN3179).

## The cable

The Mac's end of the cable is the display's adapter, whose network service `WS_CABLE_SERVICE`
names, as `networksetup -listallnetworkservices` prints it. Where `networksetup -getinfo` shows
other settings, the script runs `sudo networksetup -setmanual SERVICE 10.77.0.2 255.255.255.252`:
with no router argument, the service gets none. `--check` prints that command. The script stops on a
service the list lacks, or one with a router outside `10.77.0.0/30`, as an uplink has. A disabled
service, which the list prints with a `*` first and the key names without it, draws a warning at
each run while it stays off. Without the key, it does nothing. `-setmanual` replaces the service's
settings, whatever they were, and emptying the key leaves them. To give the service DHCP again:
`sudo networksetup -setdhcp SERVICE`.

An address, mask or router that `getinfo` leaves out, as it may undocked, counts as set. Where it
still shows other settings after `-setmanual`, the script names them and stops. Then set the service
by hand. In System Settings › Network, pick it › Details › TCP/IP, and Configure IPv4 Manually: the
IP address `10.77.0.2`, the subnet mask `255.255.255.252`, no router. Or empty the key.

Set the key once ws's end is up ([network](../../server/ubuntu/network/README.md)). Then, docked, check once that the uplink
keeps the default route:

```sh
route -n get default | grep interface
```

It must name the uplink's device, which `networksetup -listnetworkserviceorder` pairs with its
service. Where it names the adapter's, move the service below the uplink's: Set Service Order, in
the menu of System Settings › Network, or `networksetup -ordernetworkservices`, given every service
in order.

## The Mac's tunnel

The keys come first, in this order. A run of the `wireguard` role makes ws's key and prints its
public key. `ws client macos apply`, without `WS_PUBLIC_KEY`, makes the Mac's key, which only root
reads. It prints its public key with the two commands that make the Mac ws's peer, over a direct
way, as `ssh ws` needs this done first. ADDRESS and PORT are that way:

```sh
ssh -o HostName=ADDRESS -o Port=PORT ws "printf '%s\n' 'MAC_PUBLIC_KEY' | tee /etc/wireguard/mac.pub
>/dev/null" ssh -o HostName=ADDRESS -o Port=PORT ws ws server ubuntu apply --only wireguard
```

For a `WS_USER` other than root, both go through `sudo` and `ssh -t`, for the password.

With ws's key as `WS_PUBLIC_KEY`, `ws client macos apply` again installs the tunnel's daemon. A new
Mac or a rebuilt server changes one file on each side, and nothing in the repository. The wizard's
Tunnel stage takes these steps, which stay the way by hand ([The tunnel](stages/README.md#the-tunnel)).

A daemon per server, `ws client macos tunnel run --server NAME`, runs as root from launchd, from
boot and before any login, by `/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.NAME.plist`. `ws
client macos apply` copies `ws` from `~/.local/bin`, which a run makes if missing, to
`/usr/local/libexec/ws-tunnel/ws` as root's. The user owns `~/.local/bin`, so a root daemon that ran
that file would let anything that runs as the user become root. The daemon holds WireGuard's device
itself, so it needs no `wg` or `wireguard-go`. `apply` restarts it where the copy changed.

The settings are in `/etc/ws-tunnel`, which only root reads. It holds the Mac's key, which `ws
client macos tunnel key` makes once, `routes.d`, below, and `NAME/tunnel.conf`. `ws client macos
apply` keeps the public key in `~/.config/ws/tunnel.pub`, which the wizard reads, and copies
`tunnel.conf` from the server's file, so the daemon reads no file the user can write. In System
Settings › General › Login Items & Extensions, its "Allow in the Background" must stay on.

The tunnel's end on the Mac is `10.99.0.2`, on a utun of wireguard-go's, and ws's is `10.99.0.1`, at
slot 0. `WS_SLOT` N gives `10.99.N.2` and `10.99.N.1`, and a moved slot restarts the daemon. So a
run of the `wireguard` role stops where its variables give either end another address ([The tunnel](../../server/ubuntu/wireguard/README.md#the-tunnel)).
The daemon points the Mac's end at the first of these that answers, each on UDP 51820:

- the cable, `10.77.0.1`, where one of the Mac's addresses lies in `10.77.0.0/30`;
- the LAN, `WS_LAN_ADDRESS`, where one lies in `WS_LAN`;
- `WS_REMOTE`, at its first IPv4 address, as the router forwards IPv4 alone.

A handshake proves a switch. The daemon removes ws's peer and adds it again with the new endpoint,
so that wireguard-go handshakes at once. It keeps the way where the handshake completes within 5 s,
or tries the next. ws follows the Mac to its new address from that handshake, and TCP inside the
tunnel sends again what the pause lost.

The daemon tries the ways again 2 s after macOS posts a network change, where the Mac's addresses
changed. Besides, 10 s after each check ends, it checks them again, in case it missed one, and
pings `10.99.0.1`. Three pings missed in a row have it try the ways again, as after the home's
address changed. While none answers, it tries them all again 10 s after each round, which takes up
to 5 s a way. A way that comes back while the Mac's addresses stay as they were is taken at the
next change or failure.

```sh
sudo /usr/local/libexec/ws-tunnel/ws client macos tunnel status [--server NAME]
```

prints the way in use, its endpoint, the latest handshake and the bytes moved, of the one daemon
installed. `/var/log/ws-tunnel.NAME.log` has a line for each switch and failure.

A connection to `ws` outlives a switch. Zed has [its page](zed/README.md).

The tunnel's end on the Mac runs in userspace, and wireguard-go on macOS moves one packet a read and
a write. So the tunnel may fall short of the cable's 2.5 Gbit/s. To measure it, on ws, install
iperf3 from apt, let its port in on the tunnel and the cable for the test alone, and run the server:

```sh
ufw allow in on wg0 to any port 5201 proto tcp
ufw allow in on CABLE_INTERFACE to any port 5201 proto tcp
iperf3 -s
```

then, on the Mac, compare `iperf3 -c 10.99.0.1` with `iperf3 -c 10.77.0.1`, each with `-R` too, and
delete the two rules after. Bulk copies run at that speed: for faster ones, give rsync a direct way,
such as `rsync -e 'ssh -o HostName=ADDRESS -o Port=PORT' -a DIR ws:`. Besides, check `ws client
macos tunnel status` docked, on the home Wi-Fi and on a phone's hotspot, and `ethtool
CABLE_INTERFACE` on ws, for 2500Mb/s.

## Routes and resolvers of another repository

A private repository's own script for the Mac can send more networks through the tunnel, and a
domain's names to nameservers behind it. It calls `ws`, which changes only what you give it.
`--check` makes each print its diffs and change nothing:

- `ws client macos tunnel routes [--check] NAME CIDR...`, which writes the IPv4 networks to
  `/etc/ws-tunnel/routes.d/NAME` and has the daemon read them again. The daemon adds them to the
  peer's AllowedIPs and routes them into its utun, without closing a connection. It leaves out for
  the time being one that holds the endpoint in use, or one that holds a way's endpoint while none
  answers. `NAME` is of `a-z`, `0-9` and `-`, `mac-conf` aside. A network that overlaps the
  tunnels' `10.99.0.0/20` or the cable's `10.77.0.0/30`, as `0.0.0.0/0` does, is refused. As in
  `ws client macos tunnel routes example 198.51.100.0/24`. - `ws client macos resolver [--check]
  DOMAIN ADDRESS...`, which writes `/etc/resolver/DOMAIN`, with up to 3 nameservers, so that macOS
  asks them for the domain's names (resolver(5)), and takes the file at once. - `ws client macos
  tunnel loaded`, which says whether the daemon is loaded, so that the script can stop with "run ws
  client macos apply first".

With no network or address, each removes its file, `ws client macos resolver` only a file it wrote.
The files outlive the daemon. While the daemon is stopped, the routes go with its utun, and the
domain's lookups go out by the Mac's own network and fail. ufw on ws forwards from the tunnel only
ping, and what the `ikeward` role lets through to its VPN ([Forwarding](../../server/ubuntu/ikeward/README.md#forwarding)). Any
other network behind ws needs its side set up on ws too. A network the Mac is on, inside one sent
into the tunnel, keeps its part off the tunnel: macOS routes the narrower network to the Mac's own
interface.
