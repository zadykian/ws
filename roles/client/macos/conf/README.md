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
lower-case letter, then up to 11 lower-case letters, digits and `-`. With one server, the flag may
be left out. With none, the wizard asks the name, `ws` by default. With several, it refuses and
lists them. `WS_SLOT`, from 0 to 15, is the slot: the wizard writes `WS_SLOT=0` in a file that
lacks it. A directory of a bad name that holds `server.conf` is refused too.

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
  that `WS_PORT` sets. The closing list says so.
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
ways. A new machine offers another host key, so run `ssh-keygen -R ws` first. From a cloud VM,
remove `security_outbound: false` from ws's `host_vars.yml` too, as the closing list says, then run
the wizard with `--update`.
