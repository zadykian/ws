# The Mac

The Mac reaches the server, ws, over the cable from the 2.5GbE port of the display it docks at. It
reaches it over the LAN, or elsewhere through the router's port forwards. `ws mac apply` sets up its
end of the cable ([The cable](#the-cable)), git ([git on the Mac](mac-git.md)), Zed ([Zed on the
Mac](mac-zed.md)) and one host for ssh, `ws`. It goes through a WireGuard tunnel that ends on ws,
at `10.99.0.1` whichever way the Mac connects. The sessions of `ssh ws` and Zed stay open as the
Mac moves between the cable, the LAN and elsewhere, with a pause of a few seconds
([Zed](zed.md#when-the-connection-drops)). What it gives up: `ssh ws` fails where UDP is blocked and
while the tunnel is down, and bulk copies run at wireguard-go's speed. To reach ws by hand, give ssh
a direct way: `ssh -o HostName=ADDRESS -o Port=PORT ws`, as ufw lets SSH in on every interface.
`WS_REMOTE_PORT` and the router's forward stay for that and for setup.

The Mac's side is Go: one binary, `ws`, of isolated steps, which `go test` checks on each pull
request. It runs as the user, with sudo where it needs root. It sets up Ghostty and cld ([Ghostty
and cld on the Mac](mac-ghostty.md)), and installs Ghostty, Zed and a font from Homebrew,
[installed](https://brew.sh) first. `install.sh` puts `ws` in `~/.local/bin` (put it on `PATH` by
hand) from a release ([Releases](releases.md)). `ws` embeds the files it installs, so it needs no
clone. Then:

```sh
ws mac apply
```

`ws mac wizard`, in its place, asks first for the values below ([The Mac's wizard](mac-wizard.md)),
and `ws mac configure` changes them later on a form ([The Configure
screen](tui-configure.md)). `ws mac apply STEP` runs one step, and `--check` shows each
change and makes none. At a terminal it draws [the run screen](tui.md#the-run-screen), past
[the list of steps](tui.md#the-list-of-steps) where it gets neither, and `--plain` keeps its lines.
`ws mac state` reports what a check reads, and changes nothing ([State](../README.md#state)).
`ws mac` holds the Mac's commands ([the secrets](mac-secrets.md) too), darwin alone. `ws version`
prints its version, and `ws update` replaces it with the latest release
([Update](releases.md#update)).

## mac.conf

The repository names no server, so the Mac's values go in `~/.config/ws/mac.conf`, `KEY=value`
lines that the script reads without running them:

```sh
WS_PUBLIC_KEY=WS_KEY
WS_LAN=10.88.0.0/24
WS_LAN_ADDRESS=10.88.0.10
WS_CABLE_SERVICE="USB 10/100/1G/2.5G LAN"
WS_GIT_NAME="NAME"
WS_GIT_EMAIL=NAME@example.com
WS_REMOTE=ws.example.com
WS_REMOTE_PORT=22022
WS_TUNNEL_NETWORKS="10.20.0.0/16 198.51.100.0/24"
WS_RESOLVERS="example.internal=10.20.0.53,10.20.0.54"
```

`WS_PUBLIC_KEY` is ws's WireGuard public key, which the `wireguard` role prints:
[The Mac's tunnel](#the-macs-tunnel), below. `WS_LAN` is the home LAN, and `WS_LAN_ADDRESS` ws's
address inside it, the router's static lease for it. The script stops on one outside, and on the
LAN's network or broadcast address. `WS_REMOTE` is the router's name on the internet, and
`WS_REMOTE_PORT` the port it forwards to ws's 22. The tunnel takes UDP 51820 every way, which the
router forwards too; `WS_PORT` sets another. A way without its values is left out. The router's
side is in [The router](router.md). `WS_PUBLIC_KEY` goes in last, once ws has the Mac's key, in the
order of [The Mac's tunnel](#the-macs-tunnel). `WS_CABLE_SERVICE` names the network service of the
display's adapter, in quotes where it holds a blank, and `WS_CABLE_PORT` ws's port of the cable:
[The cable](#the-cable). `WS_GIT_NAME` and `WS_GIT_EMAIL`, which go together, are the author of
git's commits on the Mac, the name in quotes where it holds a blank. `WS_GIT_SIGNING_KEY` is the
`.pub` of the key that signs them, by its path from `/` or `~/`: [git on the Mac](mac-git.md).
`WS_IDENTITY` is the `.pub` of the key that ssh offers ws, and `WS_IDENTITY_AGENT` the socket of its
agent, in quotes where it holds a blank: [ssh's hosts](#sshs-hosts).

`WS_USER` is the user ssh logs in as: the server's `ws_user`, `zadykian` by default, or `root` for a
`WS_TARGET` of `cloud`, which may have no other user. The other targets refuse `WS_USER=root`. A key
with an empty value, as `WS_REMOTE=`, takes its default, as a missing line does: that user for
`WS_USER`, 51820 for `WS_PORT` and `~/.ssh/id_ed25519_signing.pub` for `WS_GIT_SIGNING_KEY`. Where
Secretive's agent lists one key for ws, `WS_IDENTITY` and `WS_IDENTITY_AGENT` take ws's copy of that
key's `.pub` and the socket. Several stop the script ([ssh's hosts](#sshs-hosts)). With none, or no
socket, `WS_IDENTITY` takes the file key's `.pub`, `~/.ssh/id_ed25519.pub`, and `WS_IDENTITY_AGENT`
none, as the others do.

`WS_TUNNEL_NETWORKS` sends more IPv4 networks through the tunnel, split by blanks. `WS_RESOLVERS`
sends a domain's names to nameservers behind it, in `DOMAIN=ADDRESS` pairs split by blanks. Each
domain comes once and in lower case, with up to 3 addresses split by commas. Once launchd has the
tunnel's daemon, `ws mac apply` hands both to the code of `ws mac tunnel routes` and
`ws mac resolver`, whose caveats hold
([their section](#routes-and-resolvers-of-another-repository)). The networks go in
`/etc/ws-tunnel/routes.d/mac-conf`, a name no other script may take. Each resolver starts with a
line of `ws mac apply`'s own, so a domain whose file is another script's, or written by hand, stops
the run. To move a domain from such a script, have it remove its file first, with
`ws mac resolver DOMAIN` and no address. A key emptied or removed takes back what earlier runs
wrote, and only that. `--check` shows each change: a diff, or a line where `routes.d` is missing.

## ssh's hosts

The script writes `~/.ssh/config.d/ws`, with one host, `ws`, at the tunnel's `10.99.0.1` from the
first run, whichever network the Mac is on. So `ssh ws` fails until the keys' exchange ends
([The Mac's tunnel](#the-macs-tunnel)), where a direct way serves. It logs in as `WS_USER` with the
key below, and checks ws's host key under the one name `ws` (`HostKeyAlias`). A session ends after
120 s without word from ws, far longer than a switch of the tunnel takes.

ssh offers ws one key alone (`IdentitiesOnly`). Where Secretive's agent has its socket in
`~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data`, the key is the one that the agent
lists, but git's signing key and a certificate ([Secretive's keys](mac-git.md#secretives-keys)). The
script stops on several, and names them. Without one, or elsewhere, the key is
`~/.ssh/id_ed25519`, its passphrase kept in the Keychain.
Where `config.d/ws` gave ws that key, the script warns as Secretive's replaces it. Keep the file key
with `WS_IDENTITY=~/.ssh/id_ed25519.pub` for an installed ws, or put Secretive's key on GitHub
before ws's next install. `WS_IDENTITY` names any `.pub`, as one of Secretive's. Its agent is the
one whose socket `WS_IDENTITY_AGENT` names, else Secretive's where the `.pub` is there. Else ssh
reads the file beside the `.pub`. The script stops where that `.pub`, but the file key's, is missing
or unreadable. It stops too where the `.pub` holds other than one public key's line, as a private
key or a certificate does. So does a key that ssh-keygen cannot read, or one of a type that ws's
OpenSSH lacks, as Secretive's ML-DSA keys on macOS 26.

The script puts `Include config.d/ws` first in `~/.ssh/config`, as ssh keeps the first value it
reads for each key. So neither the file's own settings nor the other files of `config.d` override
ws's. Where the line is missing, the script adds it at the top and keeps the old file as
`~/.ssh/config.TIME~`. Where the line sits below other settings, the script stops. The line also
counts by `~/.ssh/` or the home's full path, in quotes, or as the first of several paths.

Where `~/.ssh/config` is a link, as a dotfiles manager makes it, the script reads the file it points
to. It stops where the line is missing there, rather than replace the link with a file. Where
`~/.ssh/config.d` is a link, the script warns as it writes `config.d/ws` there. It stops where
`config.d/ws` is a file it did not write, rather than replace it. It warns of a `Host` or `Match` of
`~/.ssh/config`'s own for `ws`, whose settings come after those of `config.d/ws`. Remove an older
`Host ws` there, whose `IdentityFile` would add a key to ws's.

ssh keeps the first `IgnoreUnknown` it reads too. `config.d/ws` needs one for the file key's
`UseKeychain`, which only Apple's ssh knows, and keeps it with an agent's key. So it carries the
patterns of every `IgnoreUnknown` in `~/.ssh/config`. Run the script again after a change to those.
A pattern in a file that `~/.ssh/config` includes must be in `~/.ssh/config` too, as the script
reads no other file.

Before it writes, the script has `ssh -G`, for `ws`, read the new `config.d/ws` alone.
Then it reads it first, `~/.ssh/config` next, without its own Include of the old file, and the
system's file last. Where ssh refuses either, the script stops and changes nothing, under `--check`
too. The system's file goes through an Include there, so its relative Includes resolve under
`~/.ssh`. A run changes only what differs, so a second prints nothing but warnings once
`WS_PUBLIC_KEY` is set. Until then, each run prints the Mac's public key again, with the commands of
[The Mac's tunnel](#the-macs-tunnel). `--check` prints each change as a diff and makes none.

Through the forward, ws takes logins from the internet: by key alone, as everywhere, with fail2ban
banning the address of the network the Mac is on after 5 failed ones. Over the tunnel, ws sees
the Mac as `10.99.0.2`. Failed logins over the tunnel ban it, which shuts `ssh ws` out for an
hour. `ssh -o HostName=ADDRESS -o Port=PORT ws` comes from another address, and from there
`fail2ban-client set sshd unbanip 10.99.0.2` lifts the ban.

Unless `security_outbound` is false, ws opens no connection to the Mac over the cable, the LAN or
the tunnel ([The firewall](wireguard.md#the-firewall), [Outbound](security.md#outbound)). That
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

Set the key once ws's end is up ([network](network.md)). Then, docked, check once that the uplink
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
public key. `ws mac apply`, without `WS_PUBLIC_KEY`, makes the Mac's key, which only root reads. It
prints its public key with the two commands that make the Mac ws's peer, over a direct way, as
`ssh ws` needs this done first. ADDRESS and PORT are that way:

```sh
ssh -o HostName=ADDRESS -o Port=PORT ws "printf '%s\n' 'MAC_PUBLIC_KEY' | tee /etc/wireguard/mac.pub >/dev/null"
ssh -o HostName=ADDRESS -o Port=PORT ws ws server apply --only wireguard
```

For a `WS_USER` other than root, both go through `sudo` and `ssh -t`, for the password.

With ws's key as `WS_PUBLIC_KEY`, `ws mac apply` again installs the tunnel's daemon. A new Mac or a
rebuilt server changes one file on each side, and nothing in the repository. The wizard's Tunnel
stage takes these steps, which stay the way by hand ([The tunnel](mac-wizard.md#the-tunnel)).

The daemon, `ws mac tunnel run`, runs as root from launchd, from boot and before any login, through
`/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist`. `ws mac apply` copies
`ws` from `~/.local/bin` to `/usr/local/libexec/ws-tunnel/ws` as
root's. The user owns `~/.local/bin`, so a root daemon that ran that file would let anything that
runs as the user become root. The daemon holds WireGuard's device in its own process, so it
needs no `wg` or `wireguard-go`. `ws mac apply` restarts it where a rebuilt `ws` changed the copy.

The daemon's settings are in `/etc/ws-tunnel`, which only root reads. They are the Mac's key, which
`ws mac tunnel key` makes once, `tunnel.conf` and `routes.d`, below. `ws mac apply` keeps the public
key in `~/.config/ws/tunnel.pub`, which the wizard reads, and copies `tunnel.conf` from `mac.conf`,
so the daemon reads no file the user can write. In System Settings › General › Login Items &
Extensions, its "Allow in the Background" must stay on.

The tunnel's end on the Mac is `10.99.0.2`, on a utun of wireguard-go's, and ws's is `10.99.0.1`.
The daemon fixes both, and `mac.conf` sets neither. So a run of the `wireguard` role stops where its
variables give either end another address ([The tunnel](wireguard.md#the-tunnel)). The daemon
points the Mac's end at the first of these that answers, each on UDP 51820:

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
sudo /usr/local/libexec/ws-tunnel/ws mac tunnel status
```

prints the way in use, its endpoint, the latest handshake and the bytes moved.
`/var/log/ws-tunnel.log` has a line for each switch and failure.

A connection to `ws` outlives a switch. Zed has [its page](mac-zed.md).

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
such as `rsync -e 'ssh -o HostName=ADDRESS -o Port=PORT' -a DIR ws:`. Besides, check `ws mac tunnel
status` docked, on the home Wi-Fi and on a phone's hotspot, and `ethtool CABLE_INTERFACE` on ws, for
2500Mb/s.

## Routes and resolvers of another repository

A private repository's own script for the Mac can send more networks through the tunnel, and a
domain's names to nameservers behind it. It calls `ws`, which changes only what you give it.
`--check` makes each print its diffs and change nothing:

- `ws mac tunnel routes [--check] NAME CIDR...`, which writes the IPv4 networks to
  `/etc/ws-tunnel/routes.d/NAME` and has the daemon read them again. The daemon adds them to the
  peer's AllowedIPs and routes them into its utun, without closing a connection. It leaves out for
  the time being one that holds the endpoint in use, or one that holds a way's endpoint while none
  answers. `NAME` is of `a-z`, `0-9` and `-`, `mac-conf` aside. A network that overlaps the tunnel's
  `10.99.0.0/24` or the cable's `10.77.0.0/30`, as `0.0.0.0/0` does, is refused. As in
  `ws mac tunnel routes example 198.51.100.0/24`. - `ws mac resolver [--check] DOMAIN ADDRESS...`,
  which writes `/etc/resolver/DOMAIN`, with up to 3 nameservers, so that macOS asks them for the
  domain's names (resolver(5)), and takes the file at once. - `ws mac tunnel loaded`, which says
  whether the daemon is loaded, so that the script can stop with "run ws mac apply first".

With no network or address, each removes its file, `ws mac resolver` only a file it wrote. The files
outlive the daemon. While the daemon is stopped, the routes go with its utun, and the domain's
lookups go out by the Mac's own network and fail. ufw on ws forwards from the tunnel only ping, and
what the `ikeward` role lets through to its VPN ([Forwarding](ikeward.md#forwarding)). Any other
network behind ws needs its side set up on ws too. A network the Mac is on, inside one sent into the
tunnel, keeps its part off the tunnel: macOS routes the narrower network to the Mac's own interface.
