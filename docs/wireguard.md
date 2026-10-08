# wireguard

The `wireguard` role sets up `wg0`, the server's end of a WireGuard tunnel for the Mac:
`10.99.0.1/24` on UDP port 51820, from networkd's own files. Its key is made on the server, and
each peer's public key is read from `/etc/wireguard`.

## The tunnel

The Mac reaches the server through a WireGuard tunnel whichever way it connects: over the cable,
the home LAN, or the internet through the router's forward ([The router](router.md)). Inside the
tunnel the server keeps one address. WireGuard knows a peer by its key, not by the address its
packets come from, so the Mac's sessions outlive a move from one way to another.

The `wireguard` role sets up the server's end, `wg0`, at `10.99.0.1/24` (`wireguard_address`) on
UDP port 51820 (`wireguard_port`). It does so from networkd's own files,
`/etc/systemd/network/60-wg0.netdev` and `60-wg0.network`. Boot doesn't wait for `wg0`.

The first run makes the server's private key on the server, in
`/etc/systemd/network/60-wg0.key`, which the playbook never reads. networkd reads it as the user
`systemd-network`, so the file is `root:systemd-network`, mode 0640, as systemd.netdev(5) asks.
The role's last task shows the public key, which the Mac's end of the tunnel needs. It comes last
of all in `--tags wireguard`, and among the other roles' output in a full run.
`wg show wg0 public-key` shows it too. A dry run makes no key.

`wireguard_peers` names the peers, each with its address in the tunnel: by default the Mac, `mac`,
at `10.99.0.2`. The role reads a peer's public key from `/etc/wireguard/NAME.pub` on the server, a
line as `wg pubkey` prints it, in a directory only root reads:

```sh
printf '%s\n' PUBLIC_KEY >/etc/wireguard/mac.pub
/root/repository/ws/bootstrap.sh --tags wireguard
```

The repository holds no peer's key, as access to a machine is granted on the machine, as SSH's
`authorized_keys` is. Anyone may run `bootstrap.sh`, and a key in the repository would let its peer
into every machine set up from it. A peer without its file is left out of `wg0`, and the run names
it. The run stops on a peer's name of other than lowercase letters, digits and dashes. It stops on
an address outside the subnet, on the server's own or another peer's, and on a file that holds no
key.

A change reaches `wg0` through `networkctl reload`, which sets up again only the links whose files
changed: the uplink and the cable are left alone. A change of peers ends every session of the
tunnel, as networkd replaces every peer of `wg0` at once. The Mac then handshakes again within 15
seconds of sending. So change the peers over SSH outside the tunnel, or from the console. A key made
anew, after the old one was removed by hand, deletes `wg0` before the reload, as networkd reads the
key only as it creates `wg0`.

ufw lets in UDP port 51820 on every interface. Inside the tunnel it takes `wg0` as any other
interface: it lets in SSH and ping, and forwards nothing, so the tunnel reaches the server alone.
The run stops where systemd-networkd isn't running, as on Ubuntu Desktop, whose NetworkManager
does its work instead.
