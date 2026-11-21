# cable

The `cable` role sets up the server's end of a cable to the Mac, `10.77.0.1/30` on the port
`WS_CABLE_PORT` names, from a netplan file. The role is three units of [its spec](units.toml.tmpl), whose
lines sit in `mise.server.toml` ([The frame](../frame/README.md)). Its checks stay Go, as
`ws server cable check` and `ws server cable reload`, which the units run.

| Unit | What it does |
| --- | --- |
| `guard/cable` | Runs `cable check`, which refuses the port, before the file changes. |
| `file/cable` | Writes `/etc/netplan/60-cable.yaml`, after `cable check FILE` passes the new file. |
| `state/cable-reload` | Runs `cable reload` where `cable reload --check` finds a reload due. |

## The cable

The Mac can plug into the server with a cable of its own, from its dock's 2.5GbE port to a port of
the server's, outside the router. Name the server's port in `/etc/ws/values.yml` on the
server, as `ip link` names it:

```yaml
---
WS_CABLE_PORT: enp5s0
```

The file lives outside ws, so a new release keeps it. Every run reads it,
so none removes the cable for want of the value.

The file `/etc/netplan/60-cable.yaml` gives that interface `10.77.0.1/30`
(`WS_CABLE_ADDRESS`), and no DHCP, gateway or DNS. netplan merges the file with the other
files' entries of the port's name, as the installer's: its settings win, and its address joins
theirs. Boot doesn't wait for the link, which is up only while the Mac is docked. The Mac takes
`10.77.0.2/30`, with no router, from `mise -C /opt/ws/roles/client bootstrap` ([The cable](../../../client/README.md#the-cable)). ufw
lets in SSH over the cable, as over any interface. Unless `WS_SECURITY_OUTBOUND` is false, the
`security` role's table drops what the server and its containers start over the cable, whatever the
port ([Outbound](../security/README.md#outbound)). The `tunnel` role's drops keep the containers off it by
the port's name: a second layer, or the only one with the table off. So a change of the port needs a
run of that role too ([The firewall](../tunnel/README.md#the-firewall)).
 The guard stops the run where the port looks like an uplink. Such a port carries a default route,
in any routing table, or holds an IPv4 lease outside the cable's subnet. A static address outside
it, as an
earlier run's, doesn't count, so `WS_CABLE_ADDRESS` can change.

ws refuses a name with a slash, a space or a character that JSON escapes as it reads the values, as
no interface has one. It refuses an address other than an IPv4 one with its prefix there too.
 Before the file lands, its check merges it with netplan's other files in a scratch copy. A plan
runs no such check, as the file kind checks a file where it changes. The run stops where netplan
rejects them, and where an entry of another name fits the port. It fits by a `match:` of the port's
name, MAC address or driver, or by its own name. A name counts where it fits the port's name or any
of its alternative names, which `ip link` lists.
netplan keeps such an entry apart, and networkd would take the first of the two.
 A `match:` on the port's own entry must hold the port's MAC address, or a name without a glob. One
that may fit other interfaces stops the run, as the cable's file would turn their DHCP off. The run
stops too where netplan hands the port to NetworkManager rather than networkd, as on Ubuntu Desktop.
So does a booted machine where systemd-networkd isn't running. netplan's settings may hold secrets,
such as Wi-Fi passwords, so the run prints none of them, even with `-v`. A networkd file of another
source, in `/etc/systemd/network`, goes unchecked.

A machine without the interface, as a cloud VM or CI's container, is left alone. There the guard
and the file are off by `when = { interface = ... }`
([when](../../../../engine/core/kind/README.md#the-specs)), and each says so in a line. With the variable empty, as by default,
the file goes, and so does the address.

A change takes effect through `netplan generate`, then `networkctl reload`, on a booted machine.
`/var/lib/ws/cable-in-force` holds the SHA-256 of the cable's file that the last reload put in
force. `cable reload --check` finds a reload due where the file's differs: a new, changed or removed
file. A failed reload leaves the mark, and stays due. Another netplan file's change, such as an
edit of the uplink's not yet applied, never makes it due: ws puts its own file alone in force.
Before a reload, `cable reload` checks the live cable file as the file's check does: a netplan file
added since that takes the port fails the run, and nothing reloads. netplan writes
networkd's files anew for every interface it sets up, so networkd sets up each of those links
again, the uplink too: the journal shows `Reconfiguring with …` for each. The uplink keeps its
address: networkd asks its DHCP server for the same lease again, and leaves the address on the link
meanwhile.
