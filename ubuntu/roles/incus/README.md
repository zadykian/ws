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
| `state/incus-pool` | The ZFS pool `ws` | `source`, on a create alone: `WS_INCUS_POOL_SOURCE`. Always `volume.zfs.use_refquota` and `volume.zfs.remove_snapshots`. |
| `state/incus-ingress` | The ACL `ws-ingress` | Ingress: each guest's tunnel peers allowed in. |
| `state/incus-bridge` | The bridge `incusbr0` | The subnet, DHCP's resolvers, NAT, no IPv6 and the ACLs. |
| `state/incus-profile` | The profile `ws` | Unprivileged, nesting, isolated IDs, autostart, daily snapshots, an isolated NIC and the disk. |
| `state/incus-image` | The image of the guests | Ubuntu's root tarball, with ws's metadata. |
| `state/incus-ufw` | ufw's rules | DHCP in on the bridge, and the bridge's routes. |
| `service/incus-sources` | The tables `ws_sources` and `ws_guest_out` | What a guest may not claim as its source, and what may leave by a guest's tunnel. |
| `state/incus-owned` | The set `owned4` | Each guest's addresses, with the port of its NIC. |
| `state/incus-guests` | Each guest of `WS_INCUS_GUESTS` | Its instance, launched or edited, and started. |

The first five run in that order: the bridge names the ACLs, and the profile names the bridge and
the pool. The image follows, then the table of sources, the set that it holds, and the guests.

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
is on the bridge, and egress is dropped by default. Ingress is the ingress ACL's.

The egress ACL drops `10.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `172.16.0.0/12`, `192.168.0.0/16`,
`224.0.0.0/4` and `255.255.255.255`, then allows the rest. The host's own addresses must sit among
them, or a guest would reach the host's services: the values refuse a bridge address or a cable
address outside them. Incus applies each drop before any allow. A network named `incusbr0` that is no
managed bridge fails the run.

## The ingress ACL

`ws-ingress` holds the allows of what reaches the guests. The bridge's default for ingress is
drop, so the ACL has allows alone. It holds one for each peer of each guest's tunnel, and none for
a guest with no peers. A peer of `WS_IKEWARD_FORWARD_FROM` in that guest's values passes to any
destination, as ikeward keeps no rule of its own. Any other peer passes to the guest's own
addresses: its address on the bridge and the slot's `10.99.N.1`. The rules name each peer by its
address, which its slot keeps apart from the other tunnels' peers. The unit reads the guests' own
values. It fails where two tunnels share a slot or a port, or a guest sets no slot
([tunnel](../tunnel/README.md#the-guests-tunnels)). Only a booted host shows that Incus applies
the ACL to the packets that the host forwards: CI's `host` job runs it.

## The profile

Each guest takes the profile `ws`: `security.privileged` off, `security.nesting`, and
`security.idmap.isolated`, which gives it a range of root's IDs of its own. `linux.kernel_modules`
names what Docker in a guest needs, and what ikeward needs: the xfrm modules `xfrm_user`, `xfrm_algo`,
`esp4` and `xfrm_interface` for strongSwan's ESP tunnel and its link, and `nft_masq` for its
masquerade. A guest cannot load them. `br_netfilter` is not
one, so a Docker network in a guest with `enable_icc` off, or with the userland proxy off, fails: ws
does not support it. Its NIC joins the bridge with `security.mac_filtering` and `security.port_isolation`,
and its disk is on the pool. Its proxy device `otlp` is [the link to SigNoz](#the-link-to-signoz).

## The link to SigNoz

A guest under a host runs no SigNoz, and the host's gives it no address on the LAN to reach. The
profile's device `otlp` carries the guest's collector to the host's SigNoz:

| Key | Value |
| --- | --- |
| `type`, `bind` | `proxy`, `instance`: the device listens in the guest and connects from the host. |
| `listen` | `unix:/var/lib/ws/otlp/signoz.sock`, a socket in the guest. |
| `connect` | `tcp:127.0.0.1:14317`, SigNoz's intake on the host's loopback. |
| `uid`, `gid`, `mode` | `800`, `800` and `0600`: the socket is `otelcol-contrib`'s alone. |
| `security.uid`, `security.gid` | `65534` for both: the proxy never runs as the host's root. |

The device needs no ACL rule and no ufw rule. The guest's connection ends in its own namespace.
The host's side is a new connection on `lo`, which the bridge's ACL and ufw's input chain never
filter. The ACL does not deny the host whole, though: Incus's own table accepts DNS, DHCP and some
ICMP to the host ahead of it, and [ufw](#ufw) decides those.

**Why a socket.** SigNoz on a guest's loopback port would let any process there send to `14317` and
pass the guest's collector. No sender's default names a socket file, so none sends to it by mistake.
Root can open the file, on purpose alone. The host keeps `127.0.0.1:14317` open to any local
process, as it does `14318`: Docker publishes a port and no socket. Closing it would take a filter
by user, which a guest does not hold.

**The socket's owner.** The device's own defaults, 0, 0 and 0644, would let any process open it, so
the profile sets them by number. They are the IDs of `values.OtelcolID`, and the profile is shared,
so they are fixed: the guest's role [otelcol](../otelcol/README.md) makes
the account with them before the `.deb`.

**The socket's directory.** The device makes none, and a guest whose directory is missing at its
first start fails the device, so Incus stops the guest. The directory is not in `/run` or `/tmp`,
which a guest mounts after its devices start. The image holds it, as the template of a file
`README` in it: Incus makes the directory of a template's path as it creates the instance.

A self-contained guest, which runs its own SigNoz, exports to its loopback and uses no socket. The
host refuses a guest whose values turn that mode on, as its tunnel would need a LAN address.

## The table of sources

Incus's MAC filter holds a guest's MAC, and `security.port_isolation` keeps it from the other
guests' ports. Neither holds its IP address. Incus's IPv4 filter would, but it drops the replies of
the VPN, which leave a guest under a source that is not its own. So `ws_sources`, a table in
nftables' bridge family, filters at the bridge, with no `br_netfilter`. It acts on a guest's port.
It drops a frame of any ether type but ARP, IPv4 and IPv6 first: a VLAN tag would hide a source
from the rules, and the host strips the tag. An ARP sender or a source address in the bridge's
subnet or in the tunnel's slots, `10.99.0.0/20`, must be an address that the port owns. The table
lists what a guest may not claim, and passes the rest, so that a guest asks for a lease from
`0.0.0.0` and reaches the internet from its own address. A guest's own addresses are its address on
the bridge and its slot's `10.99.N.1`, which it holds on `eth0`, so that a peer's replies leave from
it.

What comes in by a guest's port takes the guest's mark, 4096 + N, before any accept. The table
`ws_guest_out`, in the `ip` family at the same priority, drops what leaves by `wgN` with another mark
([tunnel](../tunnel/README.md#the-forward)). So a guest cannot send into another guest's tunnel
under a source that the first table passes, as a forged reply of the VPN.

The set `owned4` holds each guest's addresses with its port. The file lists them from
`WS_INCUS_GUESTS` and the guests' own values, so each load fills the set. A boot gives a guest its
address before Incus starts it, and a reload leaves no moment without it. `incus-sources.service`
loads the table before Incus starts a guest, and Incus requires it, so a load that fails keeps Incus
down. A change to the file or to the table in force, hooks included, reloads it, which `nft -c`
checks first. The set's elements are no part of that comparison, so `state/incus-owned` mends one
that a hand edit changed.

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
reads each file as the guest's values, and refuses one that does not parse, or one that sets
`WS_SELF_CONTAINED`. A missing file gives the guest its defaults.

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

The image holds one more file, `/var/lib/ws/otlp/README`, so that the directory of
[the SigNoz link's socket](#the-link-to-signoz) exists at a guest's first start.

cloud-init then makes the host keys, the name and the netplan on the first boot. The seed is
written as an instance is created or copied, so a changed key of it reaches a guest that is made
after, not one that exists.

The image's alias is `ws-ubuntu-26.04-RELEASE-ID`, where ID is the digest of the metadata. A changed
template is another image, and a guest keeps the one that made it. Where Incus holds the
alias, the unit does nothing. Otherwise it downloads the tarball into `/var/cache/ws`, writes the
metadata beside it, imports both and drops them. The metadata is a gzip tarball that is the same
for the same input.

## A guest's instance

`state/incus-guests` makes each guest of the list from the image, with `incus launch` and the
instance's YAML on its standard input. An edit reads the instance from `incus list`, compares the
keys and devices that ws declares, and patches those into the writable keys that Incus holds, so the
`volatile.*` and `image.*` keys survive. A whole replace would drop them. An instance that Incus
holds with no profile `ws` fails the run: ws never turns another instance into a guest.

| Key | Value |
| --- | --- |
| `profiles` | `ws` alone. |
| `config` | `limits.cpu` and `limits.memory`, from the list. |
| `devices.eth0` | The profile's NIC, whole, with `host_name` `ws-NAME`, `ipv4.address` from the list, and `ipv4.routes`, the slot's `10.99.N.1/32`, where the guest's values name a slot but slot 0. |
| `devices.root` | The profile's disk, whole, with `size`, the quota, from the list. |
| `devices.gpu` | `type: gpu` and `vendorid: 10de`, where the guest's own values set `WS_NVIDIA_GPU`. Removed where they do not. |

The card is NVIDIA's, which several guests may hold at once, time-sliced and with no memory
isolation between them ([nvidia](../nvidia/README.md#a-guests-card)). A device that ws declares
absent is removed by the edit, and the others stay.

A device of an instance replaces the profile's device of its name, so each is whole. The NIC's
address is the lease that the bridge's DHCP gives, and the port's name is the one that the set
`owned4` holds with it. A guest that is stopped is started, and `boot.autostart` starts it with the
host. A guest of ws's that the list lacks gets a warning, and no deletion: remove it with
`ws ubuntu host guests remove NAME`.

The create alone sets `cloud-init.network-config`, which the image's template writes into the seed.
It holds DHCP on `eth0`. It adds the slot's address as a host route where the guest's own values
name a slot but slot 0, which is the host's. A later change of the slot reaches no existing guest.

## Storage

The pool counts each volume's quota by `refquota`, and a restore of an older snapshot drops the newer
ones, as ZFS restores the latest alone: `volume.zfs.use_refquota` and `volume.zfs.remove_snapshots`.
So a guest's snapshots stay out of its disk quota. The profile takes a daily snapshot of each guest,
which expires after two weeks, as a schedule with no expiry keeps every snapshot. Docker's data
stays on the guest's root dataset.

## A guest's setup

After its start, `state/incus-guests` sets each guest up. It pushes `/usr/local/bin/ws`, and the
guest's own values to `/etc/ws/values.yml` where the host holds a file, root's and mode 600. Each
goes where the SHA-256 of the guest's copy differs. It then waits for the boot and for cloud-init,
and asks the guest's own `ws ubuntu guest setup --plan`. Its exit 2, or no ws in the guest, makes
the setup due, and `incus exec` runs it. A second run finds nothing due. A guest that cloud-init
failed in is not set up, and the run fails naming it.

A guest's root controls what the host reads from it. So each exec stops after a limit: a minute for
a checksum, 15 for the boot and the plan, and an hour for the setup. What the guest prints, and
what its errors quote, loses its control characters, so that an escape sequence cannot write the
operator's clipboard. The guest checks out the tag of the pushed ws, so it needs a release's ws: a
build that names no tag fails in the guest's setup.

The unit passes no `--secrets`, as a host holds no secret. The secrets come by hand
([The guests of a host](../../runbook.md#the-guests-of-a-host)).

## By hand

| Command | What it does |
| --- | --- |
| `ws ubuntu host guests list` | Each guest of the list with its address and its state, then each guest of ws's that the list lacks. |
| `ws ubuntu host guests setup NAME [--secrets]` | The push and the setup above, whatever the guest's plan says. `--secrets` runs the task `secrets`. |
| `ws ubuntu host guests remove NAME` | Asks, then deletes the guest with its disk and snapshots. It refuses a guest that the list names, and prints `ws macos remove NAME`. |
