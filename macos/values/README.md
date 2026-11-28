# The Mac's values

A server of the Mac's is a host or a cloud VM, one of two targets that the server's file's
`WS_TARGET` names, or a guest of a host. Each target sets which of ws's ways the file names, their
defaults, and what a few units do. A guest, a container under a host, names its host and takes the
host's way ([A guest](#a-guest)).

## The two files

`/etc/ws/values.yml`, root's, holds the Mac's keys, which no server changes. On a new Mac it stands
empty before the first `ws macos setup`, which installs Secretive, waits for a key made in it and
writes the three keys ([Secretive](../roles/ssh/README.md#secretive)):

```sh
sudo mkdir -p /etc/ws/servers/ws
sudo touch /etc/ws/values.yml
```

Setup then writes these, NAME being the name of the key that its wait prints:

```yaml
WS_IDENTITY_PUB_KEY: /etc/ws/secretive/NAME.pub
WS_IDENTITY_AGENT: ~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/socket.ssh
WS_GIT_SIGNING_PUB_KEY: /etc/ws/secretive/NAME.pub
```

Where the file names `WS_IDENTITY_PUB_KEY` already, as `~/.ssh/KEY.pub` for a key in a file,
setup installs no Secretive and writes none of them.

`/etc/ws/servers/ws/values.yml`, beside it, holds every other key:

```yaml
WS_SLOT: 0
WS_PUBLIC_KEY: WS_KEY
WS_LAN: 10.88.0.0/23
WS_LAN_ADDRESS: 10.88.1.10
WS_CABLE_SERVICE: USB 10/100/1G/2.5G LAN
WS_REMOTE: ws.example.com
WS_REMOTE_PORT: 22022
WS_TUNNEL_NETWORKS: 10.20.0.0/16 198.51.100.0/24
WS_RESOLVERS: example.internal=10.20.0.53,10.20.0.54
WS_IKEWARD_DNS: 10.30.0.53
WS_IKEWARD_DOMAINS: corp.example lab.corp.example
WS_IKEWARD_FLOORS: 10.30.0.0/16 192.0.2.0/24
```

Both are YAML files of `KEY: value` lines, one a line. The key is at the line's start with its colon
right after it, and the value fits the line, in quotes where YAML needs them. Comments and blank
lines stay. ws reads that form alone. A quoted key, a space before
the colon, a flow map, a value wrapped over lines and a `---` marker are refused, with their line.
A key in the other file is refused, with the file it goes in. `WS_VALUES` names `values.yml`,
and the server's file is `servers/ws/values.yml` beside it. Each key's meaning is in
[values.yml](../README.md#the-macs-valuesyml).

## The check

`ws macos values check` reads `values.yml` and each server's file beside it, and names each bad or
missing key with its file, then exits 1. Where all pass, it prints one line. The frame's bootstrap
runs it first ([The check of the values](../roles/frame/README.md#the-check-of-the-values)).

The check, the facts, `ws macos servers`, `remove`, `secrets add`, `values print` and setup ask one
read of `values.yml` and every server's file. It works the clashes out once, and gives each server
as accepted or refused, with its refusal. The check, the facts, setup and `remove` refuse on a bad
server's file, `remove NAME` on any but NAME's. `values print` refuses its server for its own file,
for a clash with another's good file, and, for a guest, for its host's bad file. It never refuses
for another's. `secrets add` refuses for the server's own file alone, and never for its host's.

## The server's name

`--server NAME` picks the server, and its file is `servers/NAME/values.yml`. A name is a lower-case
letter, then up to 11 lower-case letters, digits and `-`, other than `direct`, `first` and `key`.
With one server, the flag may be left out. With none, it stops and says to write
`servers/NAME/values.yml`. A name that no file holds sets up a new server, beside the others. With
several and no flag, it refuses and lists them. `WS_SLOT`, from 0 to 15, is the slot, and a host's
own tunnel is slot 0, as each guest's tunnel takes a slot from 1. A file with no
`WS_SLOT` holds slot 0. Give each server the smallest slot that no other holds, as the check refuses
a clash. `WS_TUNNEL_PORT` is 51820, or 51820 plus the slot behind a router that another server
shares, or the next port up that none of those takes. The default of `WS_REMOTE_PORT` follows 22022
likewise. A LAN and `WS_TUNNEL_NETWORKS` stay clear of the slots' `10.99.0.0/20`. A directory of a
bad name that holds A server's `values.yml` is refused too. Where another server's file names a
`WS_CABLE_SERVICE`, leave a server's own empty.

## The VPN's keys

`WS_IKEWARD_DNS`, `WS_IKEWARD_DOMAINS` and `WS_IKEWARD_FLOORS` name the networks that ikeward
takes on the Mac, from the profile of the server whose file holds them. No command sets them.

- `WS_IKEWARD_DNS` is the VPN's resolver, an IPv4 address.
- `WS_IKEWARD_DOMAINS` are the domains whose names go to it, split by blanks, each once and in
  lower case.
- `WS_IKEWARD_FLOORS` are the networks that ikeward holds while on and that the daemons yield,
  split by blanks. Each is clear of the cable's network and the slots', and the resolver lies
  in one. A floor takes every range the VPN grants, as ikeward routes a grant inside a floor alone.

The three come together, each on one line: a YAML list is refused. A domain that the server's own
`WS_RESOLVERS` names too must name `WS_IKEWARD_DNS` there, and so for another server's, as the next
section says. The keys are a server's, never the Mac's.

## What two servers may not share

`mise -C /opt/ws/macos bootstrap` reads every server's file, and refuses a server whose file
takes what another's takes. Each of the two refuses until one changes, in its own file's words:

- `WS_TUNNEL_NETWORKS`: a network that overlaps one of the other's, as a network goes through one
  server's tunnel.
- `WS_RESOLVERS`: a domain the other names, as `/etc/resolver` holds one file a domain.
- `WS_IKEWARD_DNS`: a VPN of its own, as ikeward serves the profile of one server. A domain of
  `WS_IKEWARD_DOMAINS` that the other's `WS_RESOLVERS` names, or the reverse, must name the VPN's
  `WS_IKEWARD_DNS` there. A file that answers a domain both ways is ws's own, and ikeward leaves it.
- `WS_CABLE_SERVICE`: a cable the other names. Leave it empty in all but one server's file: the
  cable reaches that server alone ([the tunnel](../roles/tunnel/README.md#the-cable)).
- `WS_SLOT`: the other's slot, as each server's tunnel is the subnet of its slot.
- `WS_TUNNEL_PORT` and `WS_REMOTE_PORT`: the other's, behind the same `WS_REMOTE`, in any case, as
  the router forwards a port to one server. A guest's `WS_REMOTE` is its host's, so the host and
  each of its guests take a `WS_TUNNEL_PORT` of their own. They clash with no `WS_REMOTE` too, as
  they answer at the host's address. A guest holds no `WS_REMOTE_PORT`, as nothing forwards TCP to
  it, so none of its clashes. A guest's file refuses the clash, and the host's check fails on it.

A file of another server that is refused fails the frame's check of the values first, then each
unit that reads the servers' files, in a plan too. The error names the server and
`ws macos remove NAME`. The units read every
server's file, and the networks' unit would remove the resolver files of one it cannot read.

## The servers

`ws macos servers` lists each server by name, with its target, slot and tunnel, as the
user:

```text
NAME  TARGET  SLOT  TUNNEL
lab   cloud   1     stopped
ws    lan     0     lan 10.88.1.10:51820
```

The tunnel is `absent` before the bootstrap sets its daemon up, and `stopped` where launchd has
not loaded it. Before the daemon writes its state, it reads `starting`, and `unreadable` where ws
can't read the state. Else the column shows the tier and endpoint that the state holds. A server
whose file ws refuses shows `-` for its target and slot. Its errors follow the rows, with exit
code 1. A refusal of `values.yml` comes first and alone, with no row. The command takes no
`--server`.

`ws macos remove NAME` takes a server off the Mac, as the user. It refuses the Mac's last
server, as the bootstrap takes a Mac with none for a stray `WS_VALUES` and prunes no daemon. It
takes no `--server`: the argument is the consent. It first takes NAME's host key out of the
secrets ([The secrets](../roles/secrets/README.md#ws-macos-remove)), and stops with the
server whole where that fails. `--skip-secrets` leaves the secrets, and prints the key to drop by
hand. It then deletes `servers/NAME` and NAME's entry in
`known_hosts`, and runs `mise -C /opt/ws/macos bootstrap`, whose units take
NAME's daemon, hosts and routes away. A server that stays and whose file ws refuses, NAME's clashes
aside, stops it before it deletes anything, as those units would fail. A refusal of `values.yml`
stops it first and alone. Where the bootstrap fails,
run it again to end the removal. Where NAME held the cable, it prints a line to do by hand: the
cable's service keeps `10.77.0.2`. The server itself is not changed.

## A machine on the LAN

`WS_TARGET=lan`, the default, is a machine at home, behind [The router](../../ubuntu/roles/cable/router.md), with a cable to
the Mac's display, if any:

- `WS_LAN` is the home LAN with the server's segment, `10.88.0.0/23` by default, and
  `WS_LAN_ADDRESS` ws's static lease on the segment, `10.88.1.10`.
- `WS_REMOTE` is the router's name on the internet, if any, and `WS_REMOTE_PORT` its forward to
  ws's 22, `22022` by default.
- `WS_CABLE_SERVICE` and `WS_CABLE_PORT` set up the cable ([The cable](../README.md#the-cable)).

## A cloud VM

`WS_TARGET=cloud` is a VM at a provider, reached through the internet alone:

- It has no `WS_LAN`.
- The VM's root takes the Mac's key. An image whose root refuses a login takes `disable_root: false`
  and `users: []` in its cloud-config.
- `WS_REMOTE` is the VM's name or address, and `WS_REMOTE_PORT` its SSH port, `22` by default. The
  target needs both.
- The VM needs a public IPv4 address, as the tunnel takes IPv4 alone.
- The provider's firewall must let in TCP `WS_REMOTE_PORT`, and UDP 51820, the tunnel's, or the port
  that `WS_TUNNEL_PORT` sets, which moves wg0's listen port on ws too.
- It has no cable: `WS_CABLE_SERVICE` and `WS_CABLE_PORT` are refused.

## A guest

`WS_HOST=NAME` makes the server a guest of the host NAME, a server of the Mac's with a file of its
own, which holds the way to it. A guest has no LAN address: the host ends its tunnel, on an
interface and a port of their own, and sends what arrives to it
([The guests' tunnels](../../ubuntu/roles/tunnel/README.md#the-guests-tunnels)).

- The guest takes its host's `WS_LAN`, `WS_LAN_ADDRESS` and `WS_REMOTE`, so its tunnel tries the
  host's addresses with the guest's own `WS_TUNNEL_PORT`. The router forwards one outside UDP port
  a guest and one for the host, all to the host ([The router](../../ubuntu/roles/cable/router.md)).
- Where the host's file holds `WS_CABLE_SERVICE`, the guest tries the cable's address too, on its
  own port. The host's file alone holds the cable's keys.
- The guest holds none of `WS_LAN`, `WS_LAN_ADDRESS`, `WS_REMOTE`, `WS_REMOTE_PORT`,
  `WS_CABLE_SERVICE`, `WS_CABLE_PORT` and `WS_TARGET`, and the check refuses each but a
  `WS_TARGET` of `lan`, which a missing line gives too. It holds no
  `WS_REMOTE_PORT`, its default included, as nothing forwards TCP to it.
- The host is a server of the Mac's too, with a tunnel of its own at slot 0. It serves its SSH,
  and SigNoz's UI on `127.0.0.1:3301` behind that SSH
  ([SigNoz](../../ubuntu/roles/signoz/README.md)). A host is no guest: the check refuses a
  `WS_HOST` that names a guest or a server with no file.
- `ws macos servers` shows a guest's target as `guest`, and `ws macos remove` refuses a host that
  a guest names, until the guest goes.

The guest's first contact goes through its host, which holds the tunnel's key and its peers
([The Mac's tunnel](../README.md#the-macs-tunnel)).

## A move to another target

To move the server's file to another target, set `WS_TARGET`, and remove every key of ws's ways:
`WS_LAN`, `WS_LAN_ADDRESS`, `WS_REMOTE`, `WS_REMOTE_PORT`, `WS_CABLE_SERVICE` and `WS_CABLE_PORT`.
An old way would lead to the old machine still. From the LAN to a cloud VM, the router's `WS_REMOTE`
would. The bootstrap refuses a key of another target's ways: `WS_LAN` on a cloud VM, and
`WS_CABLE_SERVICE` or `WS_CABLE_PORT` off the LAN. Then name the new target's
ways. A new machine offers another host key, so run `ssh-keygen -R NAME` first, NAME the server's
name. Then run the bootstrap on ws, as [its runbook](../../ubuntu/runbook.md) says, and
`ws macos setup` again.
