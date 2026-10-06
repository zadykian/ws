"""The docker_route_conflicts filter, which the docker role checks the host's routes with.

Python's ipaddress, as ansible.utils' address filters need netaddr on the controller.
"""

import ipaddress

from ansible.errors import AnsibleFilterError


def _network(value):
    """Returns the network value names; ip writes a route to one address without its /32."""
    try:
        return ipaddress.ip_network(value, strict=False)
    except ValueError as err:
        raise AnsibleFilterError(f"docker_route_conflicts: {value!r} is not a network") from err


def docker_route_conflicts(routes, pool, bridge):
    """Returns, as text, the routes Docker's networks would overlap.

    routes is the parsed output of `ip -j -4 route show table all`. Docker leaves out of pool only
    the subnets of the main table's on-link routes, so any other route into pool counts, as does
    any route into bridge, the default bridge's subnet, but docker0's own: bip takes that subnet
    whatever the routes. Routes of another type than unicast (local, broadcast, unreachable) do not
    count, nor do those shorter than /8, a VPN's 0.0.0.0/1 and 128.0.0.0/1, which cover every
    address rather than name a network.
    """
    pool = _network(pool)
    bridge = _network(bridge)
    found = []
    for route in routes:
        dst = route.get("dst", "default")
        if route.get("type", "unicast") != "unicast" or dst == "default":
            continue
        network = _network(dst)
        if network.prefixlen < 8:
            continue
        table = route.get("table", "main")
        dev = route.get("dev", "-")
        on_link = table == "main" and route.get("scope") == "link"
        if (network.overlaps(bridge) and dev != "docker0") or (
            network.overlaps(pool) and not on_link
        ):
            found.append(f"{network} dev {dev} table {table}")
    return found


class FilterModule:
    """The role's filters."""

    def filters(self):
        return {"docker_route_conflicts": docker_route_conflicts}
