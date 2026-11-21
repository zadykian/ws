# security

The `security` role sets up an sshd drop-in that lets everyone log in by key only, ufw, and
fail2ban for SSH. By default, it keeps what the server and its containers start to the internet and
ikeward's VPN, but what the network needs, such as DHCP and DNS. The role holds mise config and a
spec of ws's units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.server.toml`](mise.server.toml) | The packages, fail2ban's jail and its reload, fail2ban's and ufw's services, the firewall, and the units. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units: the lockout guard, sshd's drop-in, and the outbound table. |
| [`files/`](files) | The files and the template that mise and the units write. |

mise installs the packages and enables ufw with its rules before any unit of
ws's runs. So docker's, tunnel's and ikeward's ufw rules, which are units, find ufw installed and
enabled. Without systemd booted, as in CI's container, the units that need it skip and say so.
They are the services, the table's check and comparison, and the removal of a table loaded by hand.

## SSH and the firewall

ws turns SSH's passwords off: everyone logs in by key, and root by key alone, as on a cloud VM.
ufw then lets in SSH on port 22, by mise's rule `mise:ssh`, and
WireGuard on UDP port 51820 ([tunnel](../tunnel/README.md)). It lets in DNS lookups from Docker's
bridges to the host's resolver too ([docker](../docker/README.md)), and nothing else. fail2ban bans
an address for an hour after 5 failed logins in 10 minutes. sshd's own penalties slow one that keeps
failing. Both hold on every network, with no peer or network exempt. Root logs in by the keys that
[the runbook](../../runbook.md) imports from the GitHub account into `/root/.ssh/authorized_keys`;
ws writes none and copies none from another account's file ([A new
machine](../../../../README.md#a-new-machine)). The drop-in limits no user, so the installer's
user keeps its keys. Over SSH, keep the session that ran ws open until a
new one logs in. The guard `guard/security-keys` stops the run before passwords go off if root
has no key to log in with, or a file of keys that sshd passes over, outside a container. It stops it
on every machine where `/root` is not root's home in `getent passwd`. sshd's drop-in lands
only after `sshd -G` accepts it, and sshd reloads after a change, once, in its unit. ufw's service
is active as well as enabled, as mise keeps it.

Docker's published ports get past ufw, so Docker publishes a port that names no address on
`127.0.0.1` ([docker](../docker/README.md)): reach it through an SSH tunnel. A port published on
another address, as in `-p 0.0.0.0:8080:80`, is open to every network the host is on.

## Outbound

Unless `WS_SECURITY_OUTBOUND` is false, as below, an nftables table of ws's own, `inet ws`, keeps
what the server and Docker's containers start to the internet and to `ikeward0`, ikeward's VPN
([ikeward](../ikeward/README.md)). Elsewhere, only what the list below names passes.
`security-outbound.service` loads the table from `/usr/local/lib/ws/security/outbound.nft` at
boot, before the network comes up. `ufw status verbose` still shows outgoing traffic allowed.

These pass:

- replies, to the Mac or any other host, over the LAN, the cable or `wg0`;
- what leaves by loopback or `ikeward0`, and what root sends to `docker0` or a `br-` bridge. A
  user reaches a container of the system daemon by a port it publishes, and no other way;
- DHCP's renewals, releases and declines, and DHCPv6, to any address, by networkd's user,
  `systemd-network`. networkd sends DHCP's discovers and its other requests by AF_PACKET, which
  the table never sees;
- IPv6's router and neighbour discovery and MLD reports, to any address, whoever sends them;
- lookups to port 53 by resolved's user, `systemd-resolve`, to resolved's own DNS servers. resolved
  asks them for the server and its containers ([Names](../docker/README.md#names)).

Any other new packet drops where it goes to a private, shared, link-local, multicast or broadcast
address:

- IPv4's `10.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `172.16.0.0/12`, `192.168.0.0/16`,
  `224.0.0.0/4` and `255.255.255.255`;
- IPv6's `fc00::/7`, `fe80::/10` and `ff00::/8`.

These hold the LAN, the cable, `wg0`, and a VM host's or a cloud's private network, none of them
named in a rule. By the kind of machine ws runs on, they hold:

- on the LAN, the server's segment, `10.88.1.0/24`, whose router, `10.88.1.1`, is the gateway and
  the DHCP server, and the cable to the Mac, `10.77.0.0/30`;
- on a cloud VM, the cloud's private network, if any, and its metadata service at
  `169.254.169.254`. A gateway at a public address, as some clouds give, is the internet to the
  table, and passes;
- on a VM on the Mac, the Mac's network for its VMs: the Mac is the gateway and the DHCP server,
  and NATs what ws sends on;
- on each, `wg0`, `10.99.N.0/24`, a slot's.

So, but for DHCP and resolved's lookups, the server opens no connection to the Mac or another host
of these, nor to a gateway at a private address. From the server, a UDP send there fails at once
with "Operation not permitted", and ping shows only loss. A TCP connect hangs until the program
gives up or its retries run out, after about 2 minutes, as the drop sends no reset. The rest is
the internet, and passes.

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
bridge. A container on the host's network meets the table as the server's own processes do. What
comes in by any other link, as the Mac's packets from `wg0` to `ikeward0`, is ufw's to judge
([Forwarding](../ikeward/README.md#forwarding)).

The caveats:

- root gets past the table. Its programs meet the table, but it can stop `security-outbound`,
  delete the table, or send by AF_PACKET, which the table never sees. So can any user who can
  become root: the `sudo` group's members, as the installer's user or a cloud image's `ubuntu`,
  and the `docker` group's. The table binds the other users and the bridged containers.
- The router answers at its public address too, which the table takes for the internet: the
  packet leaves by the same gateway. A rule on the router drops it for a machine on the LAN
  ([The router](../cable/router.md)).
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
  `WS_CABLE_ADDRESS` and `WS_TUNNEL_ADDRESS`, lie in the ranges above.
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
- A plan, `mise bootstrap plan`, writes nothing, so it checks neither the table nor sshd's
  drop-in: the checks run as a unit writes.

A run checks the file with `nft -c` before it lands. It then loads the file in a network namespace
of its own, as the unit loads it, and compares that table with the kernel's, as `nft -s list table
inet ws` lists them. `-s` leaves out counters' values, but keeps the counters, and the sets'
elements as a load fills them. Where the tables differ, the run reloads the table and its sets in
one transaction, with no gap. That covers a run cut short before the reload, and a table or set
flushed or deleted by hand. The path and the timer restart where their own files change, and the
fill where its script or its unit changes. In a container, where systemd is not running, the run
writes the files, but skips the check, the comparison, the units and ufw. CI loads the table on a
booted runner, and its refusals hold
([The server job](../../../../.github/workflows/server.md)).

`WS_SECURITY_OUTBOUND`, true by default, turns the table on. A cloud VM wants it false: the table
drops the cloud's metadata service at `169.254.169.254`, which cloud-init, the guest agent and
instance credentials need. On a new machine, name it in `/etc/ws/values.yml` before the first
run ([The cable](../cable/README.md#the-cable)):

```yaml
---
WS_SECURITY_OUTBOUND: false
```

On a cloud VM, `WS_TARGET=cloud`, write the line before the run. A run with `WS_SECURITY_OUTBOUND`
false takes each of the table's units off, the path unit first. Each stops and is disabled while
its file still names its `ExecStop`, and its files go after. Then `/usr/local/lib/ws/security` goes,
and the table, even one loaded by hand. The plan shows each removal. A link made by hand in another
target's `.wants`, outside what the unit's `[Install]` names, stays. Off leaves:

- ufw's default, which allows every outgoing connection;
- the `tunnel` role's chain `WS-MAC`, which drops what Docker's containers send through `wg0`
  or the cable, but for replies to the Mac's own connections
  ([The firewall](../tunnel/README.md#the-firewall)).

So Docker's containers reach a cloud's metadata service, and the VPC or the LAN, too. Only the
table's `containers` chain drops what they forward: Docker forwards it ahead of ufw's rules, and
`WS-MAC` covers `wg0` and the cable alone.

## Links under the home

`WS_USER` owns `WS_HOME`, so a link there could send ws, which runs as root, to any file: its reads,
`chown`, `chmod` and backups would follow it. ws reaches a file there as root through the account
alone: as a home file (`Account.File`, `Account.Dir`), a removal (`Account.Absent`) or the guarded
read (`User.ReadHome`). The file kind's units take that way for each file whose owner is the
account. Each refuses a link at the file, or at a directory between the home and it, and names the
link and its target. A command of ws that acts as `WS_USER`, as claude's do, reaches only what the
user could reach: `settings.json` may link into a clone of dotfiles. Replace the link with the file
itself. The check looks as the unit plans, and a home file or a removal looks again as it applies.
A read of a home file opens each directory below the home, and the file, following no link. It reads
a regular file of 1 MiB at most, with its owner and mode from that open. `rm` and `git` stay outside
the bound.
