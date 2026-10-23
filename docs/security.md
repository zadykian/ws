# security

The `security` role sets up an sshd drop-in that lets everyone log in by key only, ufw, and
fail2ban for SSH. It keeps what the server and its containers start to the internet and ikeward's
VPN.

## SSH and the firewall

The playbook turns SSH's passwords off: everyone logs in by key, root included. ufw then lets in
SSH on port 22 and WireGuard on UDP port 51820 ([wireguard](wireguard.md)). It lets in DNS lookups
from Docker's bridges to the host's resolver too ([docker](docker.md)), and nothing else. fail2ban
bans an address for 10 minutes after 5 failed logins in 10 minutes. Over SSH, keep the session that
ran the playbook open until a new one logs in. The playbook stops before it turns passwords off if
root has no key to log in with.

Docker's published ports get past ufw, so Docker publishes a port that names no address on
`127.0.0.1` ([docker](docker.md)): reach it through an SSH tunnel. A port published on another
address, as in `-p 0.0.0.0:8080:80`, is open to every network the host is on.

## Outbound

What the server and Docker's containers start goes to the internet and to `ikeward0`, ikeward's
VPN ([ikeward](ikeward.md)), and nowhere else. An nftables table of ws's own, `inet ws`, drops the
rest. `security-outbound.service` loads it from `/usr/local/lib/ws/security/outbound.nft` at boot,
before the network comes up. `ufw status verbose` still shows outgoing traffic allowed.

These pass:

- replies, to the Mac or any other host, over the LAN, the cable or `wg0`;
- what leaves by loopback, `ikeward0`, `docker0` or a `br-` bridge;
- DHCP and DHCPv6 by networkd's user, `systemd-network`, and IPv6's router and neighbour
  discovery and MLD reports;
- lookups to port 53 by resolved's user, `systemd-resolve`. resolved asks the router for the
  server and its containers ([Names](docker.md#names)).

Any other new packet drops where it goes to a private, shared, link-local, multicast or broadcast
address:

- IPv4's `10.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `172.16.0.0/12`, `192.168.0.0/16`,
  `224.0.0.0/4` and `255.255.255.255`;
- IPv6's `fc00::/7`, `fe80::/10` and `ff00::/8`.

These hold the LAN, the cable and `wg0`, so the server opens no connection to the router's LAN
address, the Mac or another host of the LAN. The rest is the internet, and passes.

The LAN has no IPv6 prefix. Should it gain one, a packet to a global address on the LAN drops
too. Its next hop is the address itself, where a routed packet's is a link-local router. That
holds where the prefix is on-link for the server, as SLAAC from an RA with the on-link flag
gives. A DHCPv6 address alone, or a prefix without that flag, routes through the router, and
passes.

What a container sends out of the server drops alike, but where it leaves by `ikeward0` or a
bridge. What comes in by any other link, as the Mac's packets from `wg0` to `ikeward0`, is ufw's
to judge ([Forwarding](ikeward.md#forwarding)).

The caveats:

- The router answers at its public address too, which the table takes for the internet: the
  packet leaves by the same gateway. A rule on the router drops it ([The router](router.md)).
- The router, the DNS server DHCP names, hears only resolved's lookups. A program that asks it
  for names itself, not resolved at `127.0.0.53`, gets no answer. chrony, which the `base` role
  installs, keeps time from Ubuntu's pools on the internet.
- WireGuard's packets to the Mac are replies: the Mac speaks first, and its tunnel sends a
  keepalive every 25 seconds. Should the Mac fall silent past conntrack's UDP timeout, 2 minutes,
  ws's packets to it drop until its next one.
- An IPv6 route through a global gateway, rather than a link-local one, drops as if on the link.
- A Docker network given a bridge name of its own, not `br-`, is no bridge to the table: the
  server's packets to its containers drop, and theirs go out unchecked.
- Docker's macvlan and ipvlan networks never meet the table, nor does AF_PACKET from a process with
  CAP_NET_RAW, as a host-network container's by default. No role runs one today.
- `nftables.service` stays off. Its `/etc/nftables.conf` flushes every table: ufw's, Docker's,
  fail2ban's and ikeward's.
- `nft list table inet ws` shows the table, and `systemctl status security-outbound` why it failed
  to load. `systemctl stop security-outbound` removes this table alone, until the next boot or run.

A run checks the file with `nft -c` before it lands. Where the file or the unit changed, or the
table is gone, as after `nft flush ruleset`, it reloads the table in one transaction, with no
moment without it. In a container, as in CI, with `--skip-tags systemd`, the run writes both, but
neither checks nor loads the table.
