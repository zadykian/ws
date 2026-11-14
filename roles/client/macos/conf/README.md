# The wizard's targets

The Mac's wizard sets up ws on one of three machines, which the server's file's `WS_TARGET` names ([The
answers](../stages/README.md#the-answers)). Each target sets the questions of ws's ways and their
defaults, and what a few stages do. `ws server ubuntu apply` is the same on each, but for the
outbound table on a cloud VM.

## The two files

`~/.config/ws/mac.conf` holds the Mac's keys, which no server changes:

```sh
WS_GIT_NAME="NAME"
WS_GIT_EMAIL=NAME@example.com
WS_GIT_SIGNING_PUB_KEY=~/.ssh/KEY.pub
WS_IDENTITY_PUB_KEY=~/.ssh/KEY.pub
WS_IDENTITY_AGENT=~/PATH/socket.ssh
```

`~/.config/ws/servers/ws/server.conf`, beside it, holds every other key:

```sh
WS_SLOT=0
WS_PUBLIC_KEY=WS_KEY
WS_LAN=10.88.0.0/24
WS_LAN_ADDRESS=10.88.0.10
WS_CABLE_SERVICE="USB 10/100/1G/2.5G LAN"
WS_REMOTE=ws.example.com
WS_REMOTE_PORT=22022
WS_TUNNEL_NETWORKS="10.20.0.0/16 198.51.100.0/24"
WS_RESOLVERS="example.internal=10.20.0.53,10.20.0.54"
```

A key in the other file is refused, with the file it goes in. `WS_CONF` names `mac.conf`, and the
server's file is `servers/ws/server.conf` beside it. Each key's meaning is in
[mac.conf](../README.md#macconf).

## The server's name

`--server NAME` picks the server, and its file is `servers/NAME/server.conf`. A name is a
lower-case letter, then up to 11 lower-case letters, digits and `-`, other than `direct`,
`first` and `key`. With one server, the flag may be left out. With none, the wizard asks the name,
`ws` by default. A name that no file holds sets up a new server, beside the others. With several
and no flag, it refuses and lists them. `WS_SLOT`, from 0 to 15, is the slot. The
wizard writes the smallest that no other server holds in a file that lacks it, and `WS_PORT` after
it. `WS_PORT` is 51820, or 51820 plus the slot behind a router that another server shares, or the
next port up that none of those takes. The default of `WS_REMOTE_PORT` follows 22022 likewise. A
LAN and `WS_TUNNEL_NETWORKS` stay
clear of the slots' `10.99.0.0/20`. A directory of a bad name that holds `server.conf` is refused
too. The wizard writes an empty `WS_CABLE_SERVICE` in a server's file where another's names one.

## What two servers may not share

`ws client macos apply` reads every server's file, and refuses a server whose file takes what
another's takes. Each of the two refuses until one changes, in its own file's words:

- `WS_TUNNEL_NETWORKS`: a network that overlaps one of the other's, as a network goes through one
  server's tunnel.
- `WS_RESOLVERS`: a domain the other names, as `/etc/resolver` holds one file a domain.
- `WS_CABLE_SERVICE`: a cable the other names. Leave it empty in all but one server's file: the
  cable reaches that server alone ([the tunnel](../tunnel/README.md#the-cable)).
- `WS_SLOT`: the other's slot, as each server's tunnel is the subnet of its slot.
- `WS_PORT` and `WS_REMOTE_PORT`: the other's, behind the same `WS_REMOTE`, as the router forwards
  a port to one server.

A file of another server that is missing or refused is left out: a run for that server refuses it.

## A machine on the LAN

`WS_TARGET=lan`, the default, is a machine at home, behind [The router](../../../server/ubuntu/network/router.md), with a cable to
the Mac's display, if any:

- `WS_LAN` is the home LAN, `10.88.0.0/24` by default, and `WS_LAN_ADDRESS` ws's static lease
  there, `10.88.0.10`.
- `WS_REMOTE` is the router's name on the internet, if any, and `WS_REMOTE_PORT` its forward to
  ws's 22, `22022` by default.
- Where ws doesn't answer at its way, the Host key stage asks its address as the router's leases
  show it.
- The Cable stage sets up the cable ([The cable](../stages/README.md#the-cable)).
- The closing list names ws's lease and the router's forwards.

## A cloud VM

`WS_TARGET=cloud` is a VM at a provider, reached through the internet alone:

- The wizard asks no LAN.
- `WS_USER` defaults to `root`, as a VM may have no other user. Set it for an image whose root refuses
  a login, such as `ubuntu`. The other targets refuse `WS_USER=root`.
- `WS_REMOTE` is the VM's name or address, and `WS_REMOTE_PORT` its SSH port, `22` by default. The
  target needs both.
- The VM needs a public IPv4 address, as the tunnel takes IPv4 alone.
- The provider's firewall must let in TCP `WS_REMOTE_PORT`, and UDP 51820, the tunnel's, or the port
  that `WS_PORT` sets, which moves wg0's listen port on ws too. The closing list says so.
- The VM's default user takes the provider's keys, which must be the GitHub account's
  ([Login](../stages/README.md#login)).
- Where ws doesn't answer at its way, the Host key stage asks the VM's public address, as the
  provider shows it.
- ws's run turns the outbound table off, which would drop the metadata service
  ([ws's run](../stages/README.md#wss-run)).
- The wizard skips the Cable stage.

## A VM on the Mac

`WS_TARGET=mac-vm` is a VM that the Mac runs. The Mac is its gateway, DHCP server and DNS:

- `WS_LAN` is the Mac's network for its VMs, `192.168.64.0/24` by default, the shared network of
  macOS's Virtualization framework. Parallels uses `10.211.55.0/24`, and VMware Fusion a
  `172.16.N.0/24` of its own. At the VM's console, `ip -4 route` shows the Mac as its gateway there.
- `WS_LAN_ADDRESS` is the VM's address there, which the target needs, as `ip -4 address` prints it.
  Keep it stable, by a static address in the VM or a reservation in its app's DHCP: else ssh goes
  to another host once the lease moves.
- ssh reaches the VM there while the Mac has an address on that network, as while a VM runs
  ([ssh's hosts](../README.md#sshs-hosts)). Else the Host key stage stops, and says to start the VM.
- The wizard asks no `WS_REMOTE`, as only the Mac reaches the VM.
- The outbound table stays on: ws's DHCP and DNS go to the Mac, which the table lets through
  ([Outbound](../../../server/ubuntu/security/README.md#outbound)).
- The wizard skips the Cable stage.

## A move to another target

To move the server's file to another target, set `WS_TARGET`, and remove every key of ws's ways: `WS_LAN`,
`WS_LAN_ADDRESS`, `WS_REMOTE`, `WS_REMOTE_PORT`, `WS_CABLE_SERVICE` and `WS_CABLE_PORT`. An old way
would lead to the old machine still. From the LAN to a cloud VM, the router's `WS_REMOTE` would, and
the cloud VM's run would turn that machine's outbound table off. `ws client macos apply` refuses a
key of another target's ways: `WS_LAN` on a cloud VM, `WS_REMOTE` on a VM on the Mac, and
`WS_CABLE_SERVICE` or `WS_CABLE_PORT` off the LAN. Then run the wizard, which asks the new target's
ways. A new machine offers another host key, so run `ssh-keygen -R NAME` first, NAME the server's
name. From a cloud VM, remove `security_outbound: false` from ws's `host_vars.yml` too, as the
closing list says, then run the wizard with `--update`.
