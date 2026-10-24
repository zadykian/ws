# security

The `security` role sets up an sshd drop-in that lets everyone log in by key only, ufw, and
fail2ban for SSH. By default, it keeps what the server and its containers start to the internet
and ikeward's VPN, but what the network needs, such as DHCP and DNS.

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

Unless `security_outbound` is false, as below, an nftables table of ws's own, `inet ws`, keeps
what the server and Docker's containers start to the internet and to `ikeward0`, ikeward's VPN
([ikeward](ikeward.md)). Elsewhere, only what the list below names passes.
`security-outbound.service` loads the table from `/usr/local/lib/ws/security/outbound.nft` at
boot, before the network comes up. `ufw status verbose` still shows outgoing traffic allowed.

These pass:

- replies, to the Mac or any other host, over the LAN, the cable or `wg0`;
- what leaves by loopback, `ikeward0`, `docker0` or a `br-` bridge;
- DHCP's renewals, releases and declines, and DHCPv6, to any address, by networkd's user,
  `systemd-network`. networkd sends DHCP's discovers and its other requests by AF_PACKET, which
  the table never sees;
- IPv6's router and neighbour discovery and MLD reports, to any address, whoever sends them;
- lookups to port 53 by resolved's user, `systemd-resolve`, to resolved's own DNS servers. resolved
  asks them for the server and its containers ([Names](docker.md#names)).

Any other new packet drops where it goes to a private, shared, link-local, multicast or broadcast
address:

- IPv4's `10.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `172.16.0.0/12`, `192.168.0.0/16`,
  `224.0.0.0/4` and `255.255.255.255`;
- IPv6's `fc00::/7`, `fe80::/10` and `ff00::/8`.

These hold the LAN, the cable, `wg0`, and a VM host's or a cloud's private network, none of them
named in a rule. By the kind of machine ws runs on, they hold:

- on the LAN, `10.88.0.0/24`, whose router, `10.88.0.1`, is the gateway and the DHCP server, and
  the cable to the Mac, `10.77.0.0/30`;
- on a cloud VM, the cloud's private network, if any, and its metadata service at
  `169.254.169.254`. A gateway at a public address, as some clouds give, is the internet to the
  table, and passes;
- on a VM on the Mac, the Mac's network for its VMs: the Mac is the gateway and the DHCP server,
  and NATs what ws sends on;
- on each, `wg0`, `10.99.0.0/24`.

So, but for DHCP and resolved's lookups, the server opens no connection to the Mac or another host
of these, nor to a gateway at a private address. The rest is the internet, and passes.

resolved's servers are the `nameserver` lines of `/run/systemd/resolve/resolv.conf`, from DHCP,
DHCPv6, router advertisements, `DNS=` or a client such as `resolvectl`. The sets `resolvers4` and
`resolvers6` hold them. A fill follows each change, and another runs each minute while the machine
is up, so a change the first fill missed reaches them within a minute. On a machine on the LAN they
hold the router, and on a VM on the Mac, the Mac. On a cloud VM they hold the cloud's resolver, at a
link-local, private or public address.

`nft list set inet ws resolvers4` shows the IPv4 ones. The table's own load fills them from
`resolv.conf` where it exists, as on a reload, so they are empty at boot alone, until resolved
writes it. Then, and for the moment a fill takes after each change, a lookup to a private server
drops, and one to a public server passes as the internet.

The LAN has no IPv6 prefix. Should it gain one, a packet to a global address on the LAN drops
too. Its next hop is the address itself, where a routed packet's is a link-local router. That
holds where the prefix is on-link for the server, as SLAAC from an RA with the on-link flag
gives. A DHCPv6 address alone, or a prefix without that flag, routes through the router, and
passes.

What a container sends out of the server drops alike, but where it leaves by `ikeward0` or a
bridge. What comes in by any other link, as the Mac's packets from `wg0` to `ikeward0`, is ufw's
to judge ([Forwarding](ikeward.md#forwarding)).

The caveats:

- root gets past the table. Its programs meet the table, but it can stop `security-outbound`,
  delete the table, or send by AF_PACKET, which the table never sees. So can any user who can
  become root: the `sudo` group's members, as the installer's user or a cloud image's `ubuntu`,
  and the `docker` group's ([The docker group](docker.md#the-docker-group)). `ws_user` is root by
  default ([group_vars/all.yml](../group_vars/all.yml)), and in the `docker` group otherwise. The
  table binds the other users and the bridged containers.
- The router answers at its public address too, which the table takes for the internet: the
  packet leaves by the same gateway. A rule on the router drops it for a machine on the LAN
  ([The router](router.md)).
- A private DNS server of resolved's, as the router, hears only resolved's lookups. A program that
  asks it for names itself, not resolved at `127.0.0.53`, gets no answer. chrony, which the `base`
  role installs, keeps time from Ubuntu's pools on the internet.
- `resolv.conf` leaves out the servers of a link that is no default route for lookups. Such a link
  has `DefaultRoute=no`, or, without that setting, a routing domain, as `~corp`, but no `~.` domain.
  It leaves out a DNS delegate's servers too, but where it sets `DefaultRoute=yes`, and any server
  on a port other than 53. Lookups resolved sends to those drop where the server is private.
  `resolvectl status` shows each link's servers and its Default Route, to compare with the sets.
- WireGuard's packets to the Mac are replies: the Mac speaks first, and its tunnel sends a
  keepalive every 25 seconds. Should the Mac fall silent past conntrack's UDP timeout, 2 minutes,
  ws's packets to it drop until its next one.
- An IPv6 route through a global gateway, rather than a link-local one, drops as if on the link.
- The drops name no link: they cover the cable and `wg0` while their subnets, of
  `network_direct_link_address` and `wireguard_address`, lie in the ranges above.
- A Docker network given a bridge name of its own, not `br-`, is no bridge to the table: the
  server's packets to its containers drop, and theirs go out unchecked.
- Docker's macvlan and ipvlan networks never meet the table, nor does AF_PACKET from a
  host-network container, which Docker gives CAP_NET_RAW by default. No role runs either today.
- `nftables.service` stays off. Its `/etc/nftables.conf` flushes every table: ufw's, Docker's,
  fail2ban's and ikeward's.
- A table that fails to load at boot leaves the network up without it.
- `nft list table inet ws` shows the table, and `systemctl status security-outbound` why it failed
  to load. A load says in `journalctl -u security-outbound` which servers it left out, and when it
  loaded the table without the sets' elements, after which the unit still shows active. A fill says
  which it left out in `journalctl -u security-resolvers`. `systemctl stop security-outbound`
  removes this table alone, until the next boot or run.
- A dry run checks no file it would write, neither the table nor sshd's drop-in
  ([Dry runs](../README.md#dry-runs)).

A run checks the file with `nft -c` before it lands. It then loads the file in a network namespace
of its own, as the unit loads it, and compares that table with the kernel's, as
`nft -s list table inet ws` lists them. `-s` leaves out counters' values, but keeps the counters,
and the sets' elements as a load fills them. Where the tables differ, the run reloads the table and
its sets in one transaction, with no gap. So it does where systemd failed to load a unit of the
table's, or hasn't read it since it changed. That covers a run cut short before the reload, and a
table or set flushed or deleted by hand. In a container, with `--skip-tags systemd`, the run writes
the files, but skips the check, the comparison and the reload. CI checks the file after its runs
([The playbook](checks.md#the-playbook)).

`security_outbound`, true by default, turns the table on. A cloud VM wants it false: the table
drops the cloud's metadata service at `169.254.169.254`, which cloud-init, the guest agent and
instance credentials need. `bootstrap.sh` passes it to `ansible-playbook`, on a new machine and on
each run after:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh |
    sh -s -- -e security_outbound=false
/root/repository/ws/bootstrap.sh -e security_outbound=false
```

`-e` holds for that run alone, so the next run without it puts the table back. A run with it false
stops and disables the table's units. It removes the units, `/usr/local/lib/ws/security`
and the table, even one loaded by hand. A dry run shows each removal. Off leaves:

- ufw's default, which allows every outgoing connection;
- the `wireguard` role's chain `WS-MAC`, which drops what Docker's containers send through `wg0`
  or the cable, but for replies to the Mac's own connections
  ([The firewall](wireguard.md#the-firewall)).

So Docker's containers reach a cloud's metadata service, and the VPC or the LAN, too. Only the
table's `containers` chain drops what they forward: Docker forwards it ahead of ufw's rules, and
`WS-MAC` covers `wg0` and the cable alone.
