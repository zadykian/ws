"""The wireguard_settings_problems filter, which the wireguard role checks its variables with.

Python's ipaddress, as ansible.utils' address filters need netaddr on the controller.
"""

import ipaddress
import re

NAME = re.compile(r"[a-z0-9-]+")


def _host(ip, network):
    """Tells whether ip is a host's address in network, which keeps its first and last addresses
    for itself and its broadcast where it has more than two."""
    return ip in network and (
        network.prefixlen > 30 or ip not in (network.network_address, network.broadcast_address)
    )


def wireguard_settings_problems(peers, address, port):
    """Returns what is wrong with wireguard_peers, wireguard_address and wireguard_port, as text.

    The address is wg0's: an IPv4 address of a host with the prefix length of the tunnel's subnet.
    Each peer has a name, which names its key's file, of lowercase letters, digits and dashes, and
    an IPv4 address of its own, a host's in that subnet but not wg0's. The list is empty where
    nothing is wrong.
    """
    problems = []
    port = str(port)
    if not port.isdigit() or not 1 <= int(port) <= 65535:
        problems.append(f"wireguard_port is {port}, not a port from 1 to 65535.")
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
        if not isinstance(peer, dict):
            problems.append(f"A peer, {peer}, is not a mapping with a name and an address.")
            continue
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
            continue
        if not _host(ip, subnet):
            problems.append(f"{name}'s address, {ip}, is not a host's address in {subnet}.")
        elif ip in taken:
            problems.append(f"{name}'s address, {ip}, is {taken[ip]}'s too.")
        else:
            taken[ip] = name
    return problems


class FilterModule:
    """The role's filters."""

    def filters(self):
        return {"wireguard_settings_problems": wireguard_settings_problems}
