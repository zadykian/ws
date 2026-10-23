# wireguard

The `wireguard` role sets up `wg0`, the server's end of a WireGuard tunnel for the Mac:
`10.99.0.1/24` on UDP port 51820, from networkd's own files. Its key is made on the server, and
each peer's public key is read from `/etc/wireguard`. It keeps Docker's containers from reaching
the Mac through the tunnel or the cable.

## The tunnel

The Mac reaches the server through a WireGuard tunnel whichever way it connects: over the cable,
the home LAN, or the internet through the router's forward ([The router](router.md)). Inside the
tunnel the server keeps one address. WireGuard knows a peer by its key, not by the address its
packets come from, so the Mac's sessions outlive a move from one way to another.

The `wireguard` role sets up the server's end, `wg0`, at `10.99.0.1/24` (`wireguard_address`) on
UDP port 51820 (`wireguard_port`). It does so from networkd's own files,
`/etc/systemd/network/60-wg0.netdev` and `60-wg0.network`. Boot doesn't wait for `wg0`.

The first run makes the server's private key on the server, in
`/etc/systemd/network/60-wg0.key`. The playbook never copies, shows or diffs it: only `wg pubkey`
reads it, on the server. networkd reads it as the user `systemd-network`, so the file is
`root:systemd-network`, mode 0640, as systemd.netdev(5) asks, and each run sets them again. The
role's last task shows the public key, which the Mac's end of the tunnel needs. It comes last of
all in `--tags wireguard`, and among the other roles' output in a full run.
`wg show wg0 public-key` shows it too. A dry run makes no key.

`wireguard_peers` names the peers, each with its address in the tunnel: by default the Mac, `mac`,
at `10.99.0.2`. The role reads a peer's public key from `/etc/wireguard/NAME.pub` on the server, a
line as `wg pubkey` prints it, in a directory only root reads:

```sh
printf '%s\n' PUBLIC_KEY >/etc/wireguard/mac.pub
/root/repository/ws/bootstrap.sh --tags wireguard
```

The Mac's scripts fix the peer's name, `mac`, and the tunnel's addresses: `10.99.0.1` for the
server and `10.99.0.2` for the Mac ([The Mac's tunnel](mac.md#the-macs-tunnel)). So where
`wireguard_peers` holds `mac`, the run stops unless `wireguard_address` is `10.99.0.1/24` and
`mac`'s address `10.99.0.2`.

The repository holds no peer's key, as access to a machine is granted on the machine, as SSH's
`authorized_keys` is. Anyone may run `bootstrap.sh`, and a key in the repository would let its peer
into every machine set up from it. A peer without its file is left out of `wg0`, and the run names
it. The run stops on a peer's name of other than lowercase letters, digits and dashes. It stops on
an address outside the subnet, on the server's own or another peer's, and on a file that holds no
key.

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
its VPN ([Forwarding](ikeward.md#forwarding)).

Docker forwards its containers' packets ahead of ufw, and gives them the server's address on the
way out. Through `wg0` or the cable, the Mac would take them for the server's own, and any container
would reach the Mac's services. So the role drops what Docker's bridges, `docker0` and the `br-` of
each network, send through `wg0` or the cable, `network_direct_link_interface`. Replies to a
connection the Mac opened pass, such as one to a port published on `10.99.0.1`.

`wireguard-docker-user.service` puts the drops, for IPv4 and IPv6, in the chain `WS-MAC`, which
Docker's chain `DOCKER-USER` jumps to first. It runs before Docker starts, at boot, and at once where
a run changes the drops, which last through Docker's restarts. Where the unit fails, Docker starts
without them, and `systemctl status wireguard-docker-user` shows why.

The drops take a change of `network_direct_link_interface` as the `wireguard` role next runs, so
run `--tags network,wireguard` after one. The `security` role's table drops what bridged containers
start to the LAN too, and what the server itself starts to the Mac over the LAN, the cable or
`wg0`. A container on the host's network meets the table as the server's own processes do.
[Outbound](security.md#outbound) says what gets past it.
