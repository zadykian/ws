"""The network_uplink_problems filter, the network role's check that the cable's port is no uplink.

The direct link's file turns DHCP off on the port, which on an uplink cuts the machine off.
"""

import ipaddress


def _devices(route):
    """Return the interfaces a route goes out of, each of its next hops' too."""
    hops = [route, *route.get("nexthops", [])]
    return {hop["dev"] for hop in hops if "dev" in hop}


def network_uplink_problems(link, routes, address):
    """Return why the interface link describes looks like an uplink, as text.

    link is the parsed output of `ip -j address show dev NAME`, and routes that of
    `ip -j route show default table all`, for both families. An uplink carries a default route, in
    any table, or holds an IPv4 address from DHCP, which ip calls dynamic, outside the subnet of
    address, the direct link's. A static address outside it, as an earlier run's address, does not
    count: DHCP's are the ones the file takes away.
    """
    problems = []
    subnet = ipaddress.IPv4Interface(str(address)).network
    for info in link:
        name = info.get("ifname", "")
        tables = sorted({route.get("table", "main") for route in routes if name in _devices(route)})
        if tables:
            where = ", ".join(f"table {table}" for table in tables)
            problems.append(f"{name} carries a default route, in {where}")
        for held in info.get("addr_info", []):
            dhcp = held.get("dynamic") and held.get("scope") == "global"
            if held.get("family") != "inet" or not dhcp:
                continue
            interface = ipaddress.IPv4Interface(f"{held['local']}/{held['prefixlen']}")
            if interface.ip not in subnet:
                problems.append(f"{name} holds {interface} from DHCP, outside {subnet}")
    return problems


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"network_uplink_problems": network_uplink_problems}
