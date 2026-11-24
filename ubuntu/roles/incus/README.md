# incus

The `incus` role runs the guests as Incus's system containers, on a host. It installs `incus` and
`zfsutils-linux` from Zabbly's `lts-7.0` repository, not Ubuntu's archive, and makes the objects that
the guests need, from code and with no preseed. The role holds mise config and a spec of ws's
units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.host.toml`](mise.host.toml) | Zabbly's apt source, the packages, root's ID range and the units' lines. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units, which run `ws ubuntu host incus OBJECT`. |
| [`object/`](object) | The package that reads, compares and writes one of Incus's objects. |

mise takes Zabbly's key only where its primary fingerprint is the one that the source pins,
`4EFC 5906 96CB 15B8 7C73 A3AD 82CC 8797 C838 DCFD`: a changed key fails the
run. The package is Incus, not LXD.

## The ID range

`/etc/subuid` and `/etc/subgid` hold `root:1000000:1000000000` and nothing else, written before the
packages, so that Incus reads the range as it first starts. A guest's root maps from it. A
host's other users take no range from these files.

## The objects

Each unit runs a command of ws's. It reads the object from Incus and compares the keys that ws
declares with it. Then it creates or edits the object by YAML on the standard input of `incus`.
Incus fills other keys, such as `volatile.*`, which the comparison never reads and an edit keeps.
A command's `--check` fails where a change is due, and its plan shows the change's YAML or diff.

| Unit | Object | Keys |
| --- | --- | --- |
| `state/incus-acl` | The ACL `ws-egress` | Egress: the private ranges dropped, the rest allowed. |
| `state/incus-pool` | The ZFS pool `ws` | `source`, on a create alone: `WS_INCUS_POOL_SOURCE`. |
| `state/incus-bridge` | The bridge `incusbr0` | The subnet, DHCP's resolvers, NAT, no IPv6 and the ACL. |
| `state/incus-profile` | The profile `ws` | Unprivileged, nesting, isolated IDs, an isolated NIC and the disk. |

The units run in that order: the bridge names the ACL, and the profile names the bridge and the pool.

The pool's source is compared against `volatile.initial_source`, as ZFS's create sets `source` to the
pool's name. A pool made on another partition fails the run, as Incus never moves one and a create
wipes the partition. A host that holds its pool needs no value. Where the pool is missing and the value
empty, the run fails and names the key:

```yaml
---
WS_INCUS_POOL_SOURCE: /dev/disk/by-partuuid/0a1b2c3d-01
```

The runbook leaves the partition unformatted ([The install](../../runbook.md)). Only a booted host
shows that Incus takes the YAML as written: CI's `host` job runs the units.

## The bridge

`incusbr0` is managed and NATed, with no IPv6. Its subnet is `WS_INCUS_BRIDGE_ADDRESS`, `10.100.0.1/24`
by default, in the private ranges below and off the tunnel's slots `10.99.0.0/20`. Its DHCP hands out
`WS_INCUS_BRIDGE_DNS`, as `9.9.9.9` and `1.1.1.1`, so that no guest asks the host's resolver. A
resolver in the private ranges is refused: the ACL drops it, so no guest could resolve by it. The ACL
is on the bridge, and egress is dropped by default. Ingress is the tunnel's.

The ACL drops `10.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `172.16.0.0/12`, `192.168.0.0/16`,
`224.0.0.0/4` and `255.255.255.255`, then allows the rest. The host's own addresses must sit among
them, or a guest would reach the host's services: the values refuse a bridge address or a cable
address outside them. Incus applies each drop before any allow. A network named `incusbr0` that is no
managed bridge fails the run.

## The profile

Each guest takes the profile `ws`: `security.privileged` off, `security.nesting`, and
`security.idmap.isolated`, which gives it a range of root's IDs of its own. `linux.kernel_modules`
names what Docker in a guest needs, which a guest cannot load. `br_netfilter` is not one. Its NIC
joins the bridge with `security.mac_filtering` and `security.port_isolation`, and its disk is on
the pool.
