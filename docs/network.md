# network

The `network` role sets up the server's end of a cable to the Mac, `10.77.0.1/30` on the port
`network_direct_link_interface` names, from a netplan file.

## The direct link

The Mac can plug into the server with a cable of its own, from its dock's 2.5GbE port to a port of
the server's, outside the router. Name the server's port in `group_vars/all.yml`, as `ip link`
names it:

```yaml
network_direct_link_interface: enp5s0
```

The `network` role then writes `/etc/netplan/60-direct-link.yaml`, which gives that interface
`10.77.0.1/30` (`network_direct_link_address`), and no DHCP, gateway or DNS. netplan merges the
file with the other files' entries of the port's name, as the installer's: its settings win, and
its address joins theirs. Boot doesn't wait for the link, which is up only while the Mac is docked.
The Mac takes `10.77.0.2`, with the mask `255.255.255.252` and no router, in its adapter's
settings, by hand. ufw lets in SSH over the cable, as over any interface. The `wireguard` role
keeps Docker's containers off the cable, so a change of the port needs a run of it too
([The firewall](wireguard.md#the-firewall)).

The run stops where the port looks like an uplink: it carries a default route, in any routing
table, or holds an IPv4 lease outside the direct link's subnet. A static address outside it, as an
earlier run's, doesn't count, so `network_direct_link_address` can change.

Before the file lands, the role checks it with netplan's other files in a scratch copy, in a dry
run too. The run stops where netplan rejects them, and where an entry of another name fits the
port. It fits by a `match:` of the port's name, MAC address or driver, or by its own name. A name
counts where it fits the port's name or any of its alternative names, which `ip link` lists.
netplan keeps such an entry apart, and networkd would take the first of the two.

A `match:` on the port's own entry must hold the port's MAC address, or a name without a glob. One
that may fit other interfaces stops the run, as the direct link's file would turn their DHCP off.
The run stops too where netplan hands the port to NetworkManager rather than networkd, as on Ubuntu
Desktop, and where systemd-networkd isn't running. netplan's settings may hold secrets, such as
Wi-Fi passwords, so the run prints none of them, even with `-v`. A networkd file of another
source, in `/etc/systemd/network`, goes unchecked.

A machine without the interface, as a cloud VM or CI's container, is left alone, and the run says
so. With the variable empty, as by default, the role removes the file, and the address goes.

A change takes effect through `netplan generate`, then `networkctl reload`. netplan writes
networkd's files anew for every interface it sets up, so networkd sets up each of those links
again, the uplink too: the journal shows `Reconfiguring with …` for each. The uplink keeps its
address: networkd asks its DHCP server for the same lease again, and leaves the address on the link
meanwhile.
