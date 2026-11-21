# tunnel

The `tunnel` role sets up `wg0`, the server's end of a WireGuard tunnel for the Mac:
`10.99.N.1/24` at its slot, N, on UDP port 51820, from networkd's own files. Its key is made on
the server, and each peer's public key is read from `/etc/wireguard`. It keeps Docker's
containers from reaching the Mac through the tunnel or the cable.

The role is mise's packages, `wireguard-tools`, `systemd` and `iptables`, and six units of
[its spec](units.toml.tmpl), whose lines sit in `mise.server.toml` ([The frame](../frame/README.md)). What
Go holds stays Go, as `ws server tunnel check`, `tunnel key` and `tunnel wg0`, which the
units run. The values' check refuses the settings the role cannot use, as the values load.

| Unit | What it does |
| --- | --- |
| `guard/tunnel` | Runs `tunnel check`, which refuses a peer's key file and a networkd that does not run, names each peer left out, and says the public key. |
| `state/tunnel-key` | Runs `tunnel key`, which makes the key, where `tunnel key --check` finds it missing or its mode wrong. |
| `service/tunnel` | Writes wg0's `.netdev` and `.network`, and reloads systemd-networkd where either changes. |
| `state/tunnel-wg0` | Runs `tunnel wg0`, which makes wg0 again on a new key and waits for it, where `--check` finds wg0 without the file's key. |
| `state/tunnel-port` | Keeps ufw's rule of the port, by its comment `WireGuard`. |
| `service/tunnel-drops` | Writes the drops' script and unit, and keeps `wireguard-docker-user.service` enabled and running. |

Without systemd booted, as in CI's container, the files are written, and the key, the reloads,
`wg0`, ufw and the wait skip and say so.

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
a new key goes in force. A machine without systemd booted makes no key, and the guard says so.

`WS_TUNNEL_PEERS` names the peers, each with its address in the tunnel: by default the Mac, `mac`,
at wg0's address plus one, `10.99.0.2` at slot 0. An empty `WS_TUNNEL_PEERS: []`, or a key left
null, lists none. The role reads a peer's public key from `/etc/wireguard/NAME.pub` on the server,
a line as `wg pubkey` prints it, in a directory only root reads:

```sh
printf '%s\n' PUBLIC_KEY >/etc/wireguard/mac.pub
MISE_LOCKED=1 mise -C /opt/ws/roles/server bootstrap
```

The Mac's scripts fix the peer's name, `mac`. A server's slot gives the tunnel's ends: `10.99.N.1`
for the server and `10.99.N.2` for the Mac, slot 0's being `10.99.0.1` and `10.99.0.2`. The pair
goes in the server's values.yml, by hand
([The Mac's tunnel](../../../client/README.md#the-macs-tunnel)). So where `WS_TUNNEL_PEERS`
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

## The firewall

ufw lets in UDP port 51820 on every interface. A run removes any other rule of the comment
`WireGuard` in that shape, as that of a port it let in before. A rule of the comment made by hand
in another shape, as on `wg0` or for TCP, stays. Inside the tunnel ufw takes `wg0` as any other
interface: it lets in SSH and ping. It forwards from `wg0` only ping, as from any interface, and
what the `ikeward` role lets through to its VPN ([Forwarding](../ikeward/README.md#forwarding)).

Docker forwards its containers' packets ahead of ufw, and gives them the server's address on the
way out. Through `wg0` or the cable, the Mac would take them for the server's own, and any container
would reach the Mac's services. So the role drops what Docker's bridges, `docker0` and the `br-` of
each network, send through `wg0` or the cable, `WS_CABLE_PORT`. Replies to a
connection the Mac opened pass, such as one to a port published on wg0's address.

`wireguard-docker-user.service` puts the drops, for IPv4 and IPv6, in the chain `WS-MAC`, which
Docker's chain `DOCKER-USER` jumps to first. It runs before Docker starts, at boot, and at once where
a run changes the drops, which last through Docker's restarts. Where the unit fails, Docker starts
without them, and `systemctl status wireguard-docker-user` shows why.

The drops take a change of `WS_CABLE_PORT` as the next bootstrap runs, which sets up the cable
too. Unless `WS_SECURITY_OUTBOUND` is false,
they are a second layer: the `security` role's table drops what the containers start through `wg0`
or the cable too ([Outbound](../security/README.md#outbound)).
