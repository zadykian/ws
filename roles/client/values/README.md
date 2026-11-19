# The Mac's values

ws is one of three machines, which the server's file's `WS_TARGET` names. Each target sets which of
ws's ways the file names, their defaults, and what a few units do. The server's bootstrap is the
same on each, but for the outbound table on a cloud VM.

## The two files

`/etc/ws/values.yml`, root's, holds the Mac's keys, which no server changes:

```yaml
WS_GIT_SIGNING_PUB_KEY: ~/.ssh/KEY.pub
WS_IDENTITY_PUB_KEY: ~/.ssh/KEY.pub
WS_IDENTITY_AGENT: ~/PATH/socket.ssh
```

`/etc/ws/servers/ws/values.yml`, beside it, holds every other key:

```yaml
WS_SLOT: 0
WS_PUBLIC_KEY: WS_KEY
WS_LAN: 10.88.0.0/24
WS_LAN_ADDRESS: 10.88.0.10
WS_CABLE_SERVICE: USB 10/100/1G/2.5G LAN
WS_REMOTE: ws.example.com
WS_REMOTE_PORT: 22022
WS_TUNNEL_NETWORKS: 10.20.0.0/16 198.51.100.0/24
WS_RESOLVERS: example.internal=10.20.0.53,10.20.0.54
```

Both are YAML files of `KEY: value` lines, one a line. The key is at the line's start with its colon
right after it, and the value fits the line, in quotes where YAML needs them. Comments and blank
lines stay. ws reads that form alone. A quoted key, a space before
the colon, a flow map, a value wrapped over lines and a `---` marker are refused, with their line.
A key in the other file is refused, with the file it goes in. `WS_VALUES` names `values.yml`,
and the server's file is `servers/ws/values.yml` beside it. Each key's meaning is in
[values.yml](../README.md#the-macs-valuesyml).

## The server's name

`--server NAME` picks the server, and its file is `servers/NAME/values.yml`. A name is a lower-case
letter, then up to 11 lower-case letters, digits and `-`, other than `direct`, `first` and `key`.
With one server, the flag may be left out. With none, it stops and says to write
`servers/NAME/values.yml`. A name that no file holds sets up a new server, beside the others. With
several and no flag, it refuses and lists them. `WS_SLOT`, from 0 to 15, is the slot. A file with no
`WS_SLOT` holds slot 0. Give each server the smallest slot that no other holds, as the check refuses
a clash. `WS_TUNNEL_PORT` is 51820, or 51820 plus the slot behind a router that another server
shares, or the next port up that none of those takes. The default of `WS_REMOTE_PORT` follows 22022
likewise. A LAN and `WS_TUNNEL_NETWORKS` stay clear of the slots' `10.99.0.0/20`. A directory of a
bad name that holds A server's `values.yml` is refused too. Where another server's file names a
`WS_CABLE_SERVICE`, leave a server's own empty.

## What two servers may not share

`mise -C /opt/ws/roles/client bootstrap` reads every server's file, and refuses a server whose file
takes what another's takes. Each of the two refuses until one changes, in its own file's words:

- `WS_TUNNEL_NETWORKS`: a network that overlaps one of the other's, as a network goes through one
  server's tunnel.
- `WS_RESOLVERS`: a domain the other names, as `/etc/resolver` holds one file a domain.
- `WS_CABLE_SERVICE`: a cable the other names. Leave it empty in all but one server's file: the
  cable reaches that server alone ([the tunnel](../macos/tunnel/README.md#the-cable)).
- `WS_SLOT`: the other's slot, as each server's tunnel is the subnet of its slot.
- `WS_TUNNEL_PORT` and `WS_REMOTE_PORT`: the other's, behind the same `WS_REMOTE`, in any case, as
  the router forwards a port to one server.

A file of another server that is refused fails each unit that reads the servers' files, in a
plan too, and the error names the server and `ws client macos remove NAME`. The units read every
server's file, and the networks' unit would remove the resolver files of one it cannot read.

## The servers

`ws client macos servers` lists each server by name, with its target, slot and tunnel, as the
user:

```text
NAME  TARGET  SLOT  TUNNEL
lab   cloud   1     stopped
ws    lan     0     lan 10.88.0.10:51820
```

The tunnel is `absent` before the bootstrap sets its daemon up, and `stopped` where launchd has
not loaded it. Before the daemon writes its state, it reads `starting`, and `unreadable` where ws
can't read the state. Else the column shows the tier and endpoint that the state holds. A server
whose file ws refuses shows `-` for its target and slot. Its errors follow the rows, with exit
code 1. The command takes no `--server`.

`ws client macos remove NAME` takes a server off the Mac, as the user. It refuses the Mac's last
server, as the bootstrap takes a Mac with none for a stray `WS_VALUES` and prunes no daemon. It
takes no `--server`: the argument is the consent. It first takes NAME's host key out of the
secrets ([The secrets](../macos/secrets/README.md#ws-client-macos-remove)), and stops with the
server whole where that fails. It then deletes `servers/NAME` and NAME's entry in
`known_hosts`, and runs `MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap`, whose units take
NAME's daemon, hosts and routes away. A server that stays and whose file ws refuses, NAME's clashes
aside, stops it before it deletes anything, as those units would fail. Where the bootstrap fails,
run it again to end the removal. Where NAME held the cable, it prints a line to do by hand: the
cable's service keeps `10.77.0.2`. The server itself is not changed.

## A machine on the LAN

`WS_TARGET=lan`, the default, is a machine at home, behind [The router](../../server/ubuntu/cable/router.md), with a cable to
the Mac's display, if any:

- `WS_LAN` is the home LAN, `10.88.0.0/24` by default, and `WS_LAN_ADDRESS` ws's static lease
  there, `10.88.0.10`.
- `WS_REMOTE` is the router's name on the internet, if any, and `WS_REMOTE_PORT` its forward to
  ws's 22, `22022` by default.
- `WS_CABLE_SERVICE` and `WS_CABLE_PORT` set up the cable ([The cable](../README.md#the-cable)).

## A cloud VM

`WS_TARGET=cloud` is a VM at a provider, reached through the internet alone:

- It has no `WS_LAN`.
- `WS_USER` defaults to `root`, as a VM may have no other user. Set it for an image whose root refuses
  a login, such as `ubuntu`. The other targets refuse `WS_USER=root`.
- `WS_REMOTE` is the VM's name or address, and `WS_REMOTE_PORT` its SSH port, `22` by default. The
  target needs both.
- The VM needs a public IPv4 address, as the tunnel takes IPv4 alone.
- The provider's firewall must let in TCP `WS_REMOTE_PORT`, and UDP 51820, the tunnel's, or the port
  that `WS_TUNNEL_PORT` sets, which moves wg0's listen port on ws too.
- The VM's default user takes the provider's keys, which must be the GitHub account's.
- Set `WS_SECURITY_OUTBOUND: false` in ws's values.yml before ws's run, as the outbound table would
  drop the metadata service ([Outbound](../../server/ubuntu/security/README.md#outbound)).
- It has no cable: `WS_CABLE_SERVICE` and `WS_CABLE_PORT` are refused.

## A VM on the Mac

`WS_TARGET=mac-vm` is a VM that the Mac runs. The Mac is its gateway, DHCP server and DNS:

- `WS_LAN` is the Mac's network for its VMs, `192.168.64.0/24` by default, the shared network of
  macOS's Virtualization framework. Parallels uses `10.211.55.0/24`, and VMware Fusion a
  `172.16.N.0/24` of its own. At the VM's console, `ip -4 route` shows the Mac as its gateway there.
- `WS_LAN_ADDRESS` is the VM's address there, which the target needs, as `ip -4 address` prints it.
  Keep it stable, by a static address in the VM or a reservation in its app's DHCP: else ssh goes
  to another host once the lease moves.
- ssh reaches the VM there while the Mac has an address on that network, as while a VM runs
  ([ssh's hosts](../README.md#sshs-hosts)).
- It needs no `WS_REMOTE`, as only the Mac reaches the VM.
- The outbound table stays on: ws's DHCP and DNS go to the Mac, which the table lets through
  ([Outbound](../../server/ubuntu/security/README.md#outbound)).
- It has no cable: `WS_CABLE_SERVICE` and `WS_CABLE_PORT` are refused.

## A move to another target

To move the server's file to another target, set `WS_TARGET`, and remove every key of ws's ways: `WS_LAN`,
`WS_LAN_ADDRESS`, `WS_REMOTE`, `WS_REMOTE_PORT`, `WS_CABLE_SERVICE` and `WS_CABLE_PORT`. An old way
would lead to the old machine still. From the LAN to a cloud VM, the router's `WS_REMOTE` would, and
the cloud VM's run would turn that machine's outbound table off. The bootstrap refuses a
key of another target's ways: `WS_LAN` on a cloud VM, `WS_REMOTE` on a VM on the Mac, and
`WS_CABLE_SERVICE` or `WS_CABLE_PORT` off the LAN. Then name the new target's
ways. A new machine offers another host key, so run `ssh-keygen -R NAME` first, NAME the server's
name. From a cloud VM, remove `WS_SECURITY_OUTBOUND: false` from ws's `values.yml` too, with
`sudo ws server ubuntu values set WS_SECURITY_OUTBOUND ''`. Then run the bootstrap on ws, as
[its runbook](../../server/runbook.md) says, and `MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap` again.
