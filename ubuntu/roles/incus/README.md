# incus

The `incus` role runs the guests as Incus's system containers, on a host. It installs `incus` and
`zfsutils-linux` from Zabbly's `lts-7.0` repository, not Ubuntu's archive, and makes the objects
that the guests need, from code and with no preseed. The role holds mise config and a spec of ws's
units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.host.toml`](mise.host.toml) | Zabbly's apt source, the packages, root's ID range and the units' lines. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units, which run `ws ubuntu host incus OBJECT`. |
| [`files/`](files) | The table of sources and its service's unit file. |
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
| `state/incus-image` | The image of the guests | Ubuntu's root tarball, with ws's metadata. |
| `state/incus-ufw` | ufw's rules | DHCP in on the bridge, and the bridge's routes. |
| `service/incus-sources` | The table `ws_sources` | What a guest may not claim as its source. |

The first four run in that order: the bridge names the ACL, and the profile names the bridge and
the pool.

The pool's source is compared against `volatile.initial_source`, as ZFS's create sets `source` to
the pool's name. A pool made on another partition fails the run, as Incus never moves one and a
create wipes the partition. A host that holds its pool needs no value. Where the pool is missing and
the value empty, the run fails and names the key:

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
names what Docker in a guest needs, which a guest cannot load. `br_netfilter` is not one, so a
Docker network in a guest with `enable_icc` off, or with the userland proxy off, fails: ws does not
support it. Its NIC joins the bridge with `security.mac_filtering` and `security.port_isolation`,
and its disk is on the pool.

## The table of sources

Incus's MAC filter holds a guest's MAC, and `security.port_isolation` keeps it from the other
guests' ports. Neither holds its IP address. Incus's IPv4 filter would, but it drops the replies of
the VPN, which leave a guest under a source that is not its own. So `ws_sources`, a table in
nftables' bridge family, filters at the bridge, with no `br_netfilter`. It acts on a guest's port.
It drops a frame of any ether type but ARP, IPv4 and IPv6 first: a VLAN tag would hide a source
from the rules, and the host strips the tag. An ARP sender or a source address in the bridge's
subnet or in the tunnel's slots, `10.99.0.0/20`, must be an address that the port owns. The table
lists what a guest may not claim, and passes the rest, so that a guest asks for a lease from
`0.0.0.0` and reaches the internet from its own address.

The set `owned4` holds each guest's address with its port, and starts empty. A guest's unit fills
it. `incus-sources.service` loads the table before Incus starts a guest, and Incus requires it, so
a load that fails keeps Incus down. A reload empties the set, so the unit that fills it runs after
each load. A change to the table reloads it, where the table in force, hooks included, differs from
what the file loads, which `nft -c` checks first.

## ufw

A host's ufw lets UDP 67 in on `incusbr0` from any address, as a guest asks for a lease before it
holds one. It routes what comes in on the bridge out by any interface, and the ACL narrows that. No
rule routes what goes out on the bridge, as the tunnel's rules name the way in. DNS to the host
stays shut, as the bridge hands out other resolvers.

## The guests

The host's `/etc/ws/values.yml` lists its guests in `WS_INCUS_GUESTS`, a table that stays the file's
to edit by hand:

```yaml
---
WS_INCUS_GUESTS:
  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}
```

| Field | What it holds |
| --- | --- |
| `name` | The guest's name and hostname, 1 to 12 lowercase letters, digits and dashes, starting with a letter. The host's name for its NIC is `ws-NAME`, which an interface's 15 bytes bound. |
| `address` | The guest's address in the bridge's subnet: a host's address, not the bridge's own. |
| `cpu`, `memory`, `disk` | The core count, the memory limit and the root disk's quota, as `8GiB`. |

The values refuse a name or an address that two guests share. They refuse a field that Incus could
not take. So a name reaches a command, a path or YAML once checked.

A guest's own values sit on the host, in `/etc/ws/guests/NAME/values.yml`, root's and 600. The host
reads each file as the guest's values, and refuses one that does not parse. A missing file gives
the guest its defaults.

## The image

A guest launches from Ubuntu 26.04's cloud root tarball, `ubuntu-26.04-server-cloudimg-ARCH-root.tar.xz`
of the dated release `release-20261108`. ws pins the SHA-256 that the release's `SHA256SUMS` lists for
`amd64` and `arm64`, and checks the download against it before the import, as a moving release
would break the pin. A new release is a new date and two new sums in [image.go](image.go) and
[metadata.go](metadata.go).

The tarball holds no SSH host key, an empty `/etc/netplan`, the name `ubuntu` and an empty machine-id,
and a container under Incus has `/dev/incus` alone, so cloud-init finds no datasource. So ws writes
the image's metadata, whose templates write cloud-init's NoCloud seed in
`/var/lib/cloud/seed/nocloud-net` as an instance is created:

| File | What it holds |
| --- | --- |
| `meta-data` | The instance's name, as its id and its `local-hostname`. |
| `user-data`, `vendor-data` | The instance's `cloud-init.user-data` and `cloud-init.vendor-data`, or `#cloud-config` and `{}`. |
| `network-config` | The instance's `cloud-init.network-config`, or DHCP on `eth0`. |

cloud-init then makes the host keys, the name and the netplan on the first boot. The seed is
written as an instance is created or copied, so a changed key of it reaches a guest that is made
after, not one that exists.

The image's alias is `ws-ubuntu-26.04-RELEASE-ID`, where ID is the digest of the metadata. A changed
template is another image, and a guest keeps the one that made it. Where Incus holds the
alias, the unit does nothing. Otherwise it downloads the tarball into `/var/cache/ws`, writes the
metadata beside it, imports both and drops them. The metadata is a gzip tarball that is the same
for the same input.
