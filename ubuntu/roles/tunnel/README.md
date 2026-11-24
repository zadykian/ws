# tunnel

The `tunnel` role sets up `wg0`, the server's end of a WireGuard tunnel for the Mac:
`10.99.N.1/24` at its slot, N, on UDP port 51820, from networkd's own files. Its key is made on
the server, and each peer's public key is read from `/etc/wireguard`.

The role is mise's packages, `wireguard-tools` and `systemd`, in `mise.self-contained.toml`, and the
units of [its spec](units.toml.tmpl), four on a guest and twelve on a host, whose lines sit in
`mise.guest.toml` and `mise.host.toml`
([The frame](../frame/README.md)). A guest runs its four in the self-contained mode alone, as
[SigNoz's page](../signoz/README.md#the-self-contained-mode) says: a guest under a host has the
host's tunnel. What Go holds stays Go, as `ws ubuntu SIDE tunnel check`,
`tunnel key` and `tunnel wg0`, which the units run, SIDE being `guest` or `host`. The values'
check refuses the settings the role cannot use, as the values load.

| Unit | What it does |
| --- | --- |
| `guard/tunnel` | Runs `tunnel check`, which refuses a peer's key file and a networkd that does not run, names each peer left out, and says the public key. |
| `state/tunnel-key` | Runs `tunnel key`, which makes the key, where `tunnel key --check` finds it missing or its mode wrong. |
| `service/tunnel` | Writes wg0's `.netdev` and `.network`, and reloads systemd-networkd where either changes. |
| `state/tunnel-wg0` | Runs `tunnel wg0`, which makes wg0 again on a new key and waits for it, where `--check` finds wg0 without the file's key. |

A host holds more units, which end each guest's tunnel ([The guests' tunnels](#the-guests-tunnels)).

## The tunnel

The Mac reaches the server through a WireGuard tunnel whichever way it connects: over the cable,
the home LAN, or the internet through the router's forward ([The router](../cable/router.md)). Inside the
tunnel the server keeps one address. WireGuard knows a peer by its key, not by the address its
packets come from, so the Mac's sessions outlive a move from one way to another.

The `tunnel` role sets up the server's end, `wg0`, at its slot's `10.99.N.1/24`, slot 0's
`10.99.0.1/24` by default (`WS_TUNNEL_ADDRESS`), on UDP port 51820 (`WS_TUNNEL_PORT`). It does so
from networkd's own files, `/etc/systemd/network/60-wg0.netdev` and `60-wg0.network`. Boot doesn't
wait for `wg0`.

The first run makes the server's private key on the server, in
`/etc/systemd/network/60-wg0.key`, by ws's own key code. ws never copies, shows or diffs it: only
the commands that derive the public key read it, on the server. networkd reads it as the user
`systemd-network`, so the file is
`root:systemd-network`, mode 0640, as systemd.netdev(5) asks, and each run sets them again. The
run that makes the key shows its public key, which the Mac's end of the tunnel needs. mise shows a
unit's lines only where the unit changes, so later `ws unit read guard/tunnel` shows it, as the
guard says it where the key exists, and so does `wg show wg0 public-key`. `tunnel wg0` shows it as
a new key goes in force. Where the key is not made yet, the guard says so.

`WS_TUNNEL_PEERS` names the peers, each with its address in the tunnel: by default the Mac, `mac`,
at wg0's address plus one, `10.99.0.2` at slot 0. An empty `WS_TUNNEL_PEERS: []`, or a key left
null, lists none. The role reads a peer's public key from `/etc/wireguard/NAME.pub` on the server,
a line as `wg pubkey` prints it, in a directory only root reads:

```sh
printf '%s\n' PUBLIC_KEY >/etc/wireguard/mac.pub
ws ubuntu guest setup
```

The Mac's scripts fix the peer's name, `mac`. A server's slot gives the tunnel's ends: `10.99.N.1`
for the server and `10.99.N.2` for the Mac, slot 0's being `10.99.0.1` and `10.99.0.2`. The pair
goes in the server's values.yml, by hand
([The Mac's tunnel](../../../macos/README.md#the-macs-tunnel)). So where `WS_TUNNEL_PEERS`
holds `mac`, ws refuses the values unless `WS_TUNNEL_ADDRESS` is a slot's `10.99.N.1/24`, N from 0
to 15, and `mac`'s address that slot's `10.99.N.2`.

The repository holds no peer's key, as access to a machine is granted on the machine, as SSH's
`authorized_keys` is. Anyone may run the bootstrap, and a key in the repository would let its peer
into every machine set up from it. A peer without its file is left out of `wg0`, and the guard
names it. ws refuses the values on a peer's name of other than lowercase letters, digits and
dashes. It refuses an address outside the subnet, and the server's own or another peer's. The
guard stops the run on a file that holds no key.

A change reaches `wg0` through networkd's reload, `networkctl reload`, which sets up again only the
links whose files changed: the uplink and the cable are left alone. A change of peers ends every
session of the tunnel, as networkd replaces every peer of `wg0` at once. The Mac then handshakes
again within 15 seconds of sending. So change the peers over SSH outside the tunnel, or from the console.

networkd reads the key's file as it creates `wg0`, and as a reload finds the `.netdev` changed, but
never for a change of the key's file alone. So where `wg0` holds another key than the file's, as
after a key put back or made anew by hand, `tunnel wg0` deletes `wg0` before a reload of its own.
networkd then creates it with the file's key, and the command waits up to 10 s for `wg0` to hold
it. The guard stops the run where systemd-networkd isn't running, as on Ubuntu Desktop, whose
NetworkManager does its work instead.

## The guests' tunnels

A host ends the tunnel of each guest in `WS_INCUS_GUESTS`, not the guest. The guest's own values,
`/etc/ws/guests/NAME/values.yml`, give its slot, `WS_TUNNEL_ADDRESS`, its port and its peers. The
address is the slot's `10.99.N.1/24`, N from 1: slot 0 is the host's own `wg0`. The values' check
refuses a slot or a port that two tunnels share, the host's own among them, and a guest that sets
no slot. These units run `ws ubuntu host tunnel guests`:

| Unit | What it does |
| --- | --- |
| `state/tunnel-ufw` | Opens the UDP port of the host's own `wg0`. |
| `guard/tunnel-guests` | Runs `check`: the clash, networkd, and each peer's file. |
| `state/tunnel-guest-keys` | Makes each interface's key, `60-wgN.key`, on the host. |
| `state/tunnel-guest-links` | Writes `60-wgN.netdev` and `.network`, and has networkd create `wgN`. |
| `service/tunnel-networkd` | Writes a drop-in of networkd's, so that it keeps the rules and routes that `ip` adds, and restarts it where the drop-in changes. |
| `service/tunnel-forward` | Runs `forward` at boot, as the kernel keeps its rules and routes in memory alone. |
| `state/tunnel-guest-forward` | Runs `forward` again where a rule or a route is lost. |
| `state/tunnel-guest-ufw` | Sets each tunnel's ufw rules. |

Each guest gets one interface, `wgN`, N its slot, as an interface's name holds 15 bytes. It holds
no address: the guest holds `10.99.N.1`, and the host routes it there, the NIC's `ipv4.routes`
([incus](../incus/README.md#the-guests)) sending it to the guest alone. The rest of `10.99.N.0/24`,
the peers', goes out `wgN`. The peers' keys sit in `/etc/wireguard/wgN/NAME.pub`, as the
Mac fixes the peer's name, `mac`, for every tunnel. Each key is made on the host, and only its
public half is shown.

### The forward

A rule on `wgN` selects the table 1000 + N, whose one route leads to the guest's address on the
bridge. So the packets of a peer go to that guest whatever their destination, with no NAT: the
guest sees the peer's tunnel address. Without the rule, the main table would route them to any
guest. The command deletes a rule of a slot's priority that no tunnel wants, one of another
interface or table among them. networkd removes a rule or a route that no `.network` file names,
as it starts or reconfigures a link, so `service/tunnel-networkd` sets `ManageForeignRoutingPolicyRules=no`
and `ManageForeignRoutes=no`.

What leaves by `wgN` must have come in by that guest's NIC. The table of sources marks what comes
in by a guest's port with the guest's mark, 4096 + N. A table of the way out drops what leaves by
`wgN` with another mark
([incus](../incus/README.md#the-table-of-sources)). So no guest sends into another's tunnel, under
a forged source of the VPN's replies.

### The firewall of a host

ufw opens each tunnel's UDP port, the host's own included, and routes each interface to the bridge.
It denies any input on `wgN`, inserted first, so no service of the host's answers a peer, SSH
included. ufw's own rules for ICMP echo and DHCP come ahead of the deny. Each tunnel's rules carry a
comment of their own, `tunnel wgN: ...`. A tunnel that no guest holds loses its port and route
rules, but not its deny: the interface and its peers stay. What a peer reaches in the guest is the
ingress ACL's ([incus](../incus/README.md#the-ingress-acl)).

## The firewall

A guest holds no filter: no ufw and no table of ws's. A system container's host isolates it. A cloud
VM has its provider's firewall, which must let in SSH and the tunnel's UDP port, `WS_TUNNEL_PORT`
([the runbook](../../runbook.md)). Every tunnel peer reaches the VPN there, as no rule names the
peers of `WS_IKEWARD_FORWARD_FROM` ([Forwarding](../ikeward/README.md#forwarding)). On a cloud VM,
a container that sends through `wg0` meets no guard but the Mac's firewall
([The Mac's firewall](../../../macos/roles/firewall/README.md)).
