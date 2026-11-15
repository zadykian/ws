# tunnel

The `tunnel` role sets up `wg0`, the server's end of a WireGuard tunnel for the Mac:
`10.99.N.1/24` at its slot, N, on UDP port 51820, from networkd's own files. Its key is made on
the server, and each peer's public key is read from `/etc/wireguard`. It keeps Docker's
containers from reaching the Mac through the tunnel or the cable.

Its code is Go, in `roles/server/ubuntu/tunnel`, as steps of `ws server ubuntu apply`. They check
the settings, install `wireguard-tools`, `systemd` and `iptables`, and put in the drops that keep
the containers off the Mac. Then they make the tunnel: the key, wg0's files, the firewall and the
reload of networkd. The last step shows the public key and waits for `wg0`. Without systemd booted,
as in CI's container, the unit, networkd, ufw and the wait skip and say so.

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
the step that derives the public key reads it, on the server. networkd reads it as the user
`systemd-network`, so the file is
`root:systemd-network`, mode 0640, as systemd.netdev(5) asks, and each run sets them again. The
role's last step shows the public key, which the Mac's end of the tunnel needs. It comes last of
all in `--only tunnel`, and among the other roles' output in a full run.
`wg show wg0 public-key` shows it too. A dry run makes no key and says so.

`WS_TUNNEL_PEERS` names the peers, each with its address in the tunnel: by default the Mac, `mac`,
at wg0's address plus one, `10.99.0.2` at slot 0. An empty `WS_TUNNEL_PEERS: []`, or a key left
null, lists none. The role reads a peer's public key from `/etc/wireguard/NAME.pub` on the server,
a line as `wg pubkey` prints it, in a directory only root reads:

```sh
printf '%s\n' PUBLIC_KEY >/etc/wireguard/mac.pub
ws server ubuntu apply --only tunnel
```

The Mac's scripts fix the peer's name, `mac`. A server's slot gives the tunnel's ends: `10.99.N.1`
for the server and `10.99.N.2` for the Mac, slot 0's being `10.99.0.1` and `10.99.0.2`. The Mac's
wizard names the pair in the server's host_vars
([The Mac's tunnel](../../../client/macos/README.md#the-macs-tunnel)). So where `WS_TUNNEL_PEERS`
holds `mac`, the run stops unless `WS_TUNNEL_ADDRESS` is a slot's `10.99.N.1/24`, N from 0 to 15,
and `mac`'s address that slot's `10.99.N.2`.

The repository holds no peer's key, as access to a machine is granted on the machine, as SSH's
`authorized_keys` is. Anyone may run `ws server ubuntu apply`, and a key in the repository would let
its peer into every machine set up from it. A peer without its file is left out of `wg0`, and the
run names it. The run stops on a peer's name of other than lowercase letters, digits and dashes. It
stops on an address outside the subnet, on the server's own or another peer's, and on a file that
holds no key.

A change reaches `wg0` through `networkctl reload`, which sets up again only the links whose files
changed: the uplink and the cable are left alone. A change of peers ends every session of the
tunnel, as networkd replaces every peer of `wg0` at once. The Mac then handshakes again within 15
seconds of sending. So change the peers over SSH outside the tunnel, or from the console.

networkd reads the key's file as it creates `wg0`, and as a reload finds the `.netdev` changed, but
never for a change of the key's file alone. So where `wg0` holds another key than the file's, as
after a key put back or made anew by hand, the run deletes `wg0` before the reload. networkd then
creates it with the file's key, and the run checks that `wg0` holds it. The run stops where
systemd-networkd isn't running, as on Ubuntu Desktop, whose NetworkManager does its work instead.

## The firewall

ufw lets in UDP port 51820 on every interface, and a run removes the rule of a port it let in
before. Inside the tunnel ufw takes `wg0` as any other interface: it lets in SSH and ping. It
forwards from `wg0` only ping, as from any interface, and what the `ikeward` role lets through to
its VPN ([Forwarding](../ikeward/README.md#forwarding)).

Docker forwards its containers' packets ahead of ufw, and gives them the server's address on the
way out. Through `wg0` or the cable, the Mac would take them for the server's own, and any container
would reach the Mac's services. So the role drops what Docker's bridges, `docker0` and the `br-` of
each network, send through `wg0` or the cable, `WS_CABLE_PORT`. Replies to a
connection the Mac opened pass, such as one to a port published on wg0's address.

`wireguard-docker-user.service` puts the drops, for IPv4 and IPv6, in the chain `WS-MAC`, which
Docker's chain `DOCKER-USER` jumps to first. It runs before Docker starts, at boot, and at once where
a run changes the drops, which last through Docker's restarts. Where the unit fails, Docker starts
without them, and `systemctl status wireguard-docker-user` shows why.

The drops take a change of `WS_CABLE_PORT` as the `tunnel` role next runs, so run
`ws server ubuntu apply --only cable,tunnel` after one. Unless `WS_SECURITY_OUTBOUND` is false,
they are a second layer: the `security` role's table drops what the containers start through `wg0`
or the cable too ([Outbound](../security/README.md#outbound)).
