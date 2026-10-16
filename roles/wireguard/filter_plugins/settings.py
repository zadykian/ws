"""The wireguard_settings_problems filter, which the wireguard role checks its variables with.

Python's ipaddress, as ansible.utils' address filters need netaddr on the controller.
"""

import ipaddress
import re

NAME = re.compile(r"[a-z0-9-]+")
# The longest prefix of a subnet of more than two addresses, which keeps its first and its last
# for itself and its broadcast. A /31 has neither (RFC 3021), nor has a /32, a single address.
LONGEST_BROADCAST_PREFIX = 30
MAX_PORT = 65535
# The Mac's scripts fix the tunnel's ends, as mac/ws-tunnel's server and address, and the peer's
# name, as the mac.pub that mac/setup.sh has the user write on the server.
MAC = "mac"
MAC_SERVER = ipaddress.IPv4Interface("10.99.0.1/24")
MAC_ADDRESS = ipaddress.IPv4Address("10.99.0.2")


def _host(ip, network):
    """Tell whether ip is a host's address in network.

    A network keeps its first and last addresses for itself and its broadcast where it has more
    than two.
    """
    return ip in network and (
        network.prefixlen > LONGEST_BROADCAST_PREFIX
        or ip not in (network.network_address, network.broadcast_address)
    )


def _peer_problems(peer, subnet, names, taken):
    """Return what is wrong with a peer of wireguard_peers, as text.

    names holds the names of the peers before it, and taken the addresses they and wg0 hold, each
    with its holder's name. The peer's name joins names, and its address taken where it is free.
    """
    if not isinstance(peer, dict):
        return [f"A peer, {peer}, is not a mapping with a name and an address."]
    problems = []
    name = str(peer.get("name", ""))
    if not NAME.fullmatch(name):
        problems.append(f"A peer's name, {name!r}, isn't lowercase letters, digits and dashes.")
    elif name in names:
        problems.append(f"Two peers are named {name}.")
    names.add(name)
    value = str(peer.get("address", ""))
    try:
        ip = ipaddress.IPv4Address(value)
    except ValueError:
        problems.append(f"{name}'s address, {value}, is not an IPv4 address, as 10.99.0.2.")
        return problems
    if name == MAC and ip != MAC_ADDRESS:
        problems.append(f"mac's address is {ip}, but the Mac's scripts fix it at {MAC_ADDRESS}.")
    if not _host(ip, subnet):
        problems.append(f"{name}'s address, {ip}, is not a host's address in {subnet}.")
    elif ip in taken:
        problems.append(f"{name}'s address, {ip}, is {taken[ip]}'s too.")
    else:
        taken[ip] = name
    return problems


def wireguard_settings_problems(peers, address, port):
    """Return what is wrong with wireguard_peers, wireguard_address and wireguard_port, as text.

    The address is wg0's: an IPv4 address of a host with the prefix length of the tunnel's subnet.
    Each peer has a name, which names its key's file, of lowercase letters, digits and dashes, and
    an IPv4 address of its own, a host's in that subnet but not wg0's. With the peer mac, wg0 and
    mac take the addresses the Mac's scripts fix. The list is empty where nothing is wrong.
    """
    problems = []
    port = str(port)
    if not port.isdigit() or not 1 <= int(port) <= MAX_PORT:
        problems.append(f"wireguard_port is {port}, not a port from 1 to {MAX_PORT}.")
    address = str(address)
    try:
        if not address.partition("/")[2].isdigit():
            raise ValueError("no prefix length")
        server = ipaddress.IPv4Interface(address)
    except ValueError:
        problems.append(
            f"wireguard_address is {address}, not an IPv4 address with its prefix length, as"
            " 10.99.0.1/24."
        )
        return problems
    subnet = server.network
    if not _host(server.ip, subnet):
        problems.append(f"wireguard_address is {address}, not a host's address in {subnet}.")
    if not isinstance(peers, list):
        problems.append("wireguard_peers is not a list.")
        return problems
    names = set()
    taken = {server.ip: "wg0"}
    for peer in peers:
        problems += _peer_problems(peer, subnet, names, taken)
    if MAC in names and server != MAC_SERVER:
        problems.append(
            f"wireguard_address is {address}, but the Mac's scripts fix wg0 at {MAC_SERVER}."
        )
    return problems


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"wireguard_settings_problems": wireguard_settings_problems}
