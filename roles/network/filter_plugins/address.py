"""The network_address_problems filter, the network role's check of the direct link's address.

Python's ipaddress, as ansible.utils' address filters need netaddr on the controller.
"""

import ipaddress

# The longest prefix of a subnet of more than two addresses, which keeps its first and its last
# for itself and its broadcast: a /31 or a /32 has neither (RFC 3021).
LONGEST_BROADCAST_PREFIX = 30


def network_address_problems(value):
    """Return why value is not an IPv4 address of a host with its prefix length, as 10.77.0.1/30.

    The list is empty where it is one. A subnet of more than two addresses keeps its first and its
    last for itself and its broadcast, so neither is a host's.
    """
    value = str(value)
    address, slash, prefix = value.partition("/")
    if not slash or not prefix.isdigit():
        return [f"{value} is not an IPv4 address with its prefix length, as 10.77.0.1/30"]
    try:
        interface = ipaddress.IPv4Interface(value)
    except ValueError as err:
        return [f"{value} is not an IPv4 address with its prefix length: {err}"]
    network = interface.network
    if network.prefixlen <= LONGEST_BROADCAST_PREFIX and interface.ip == network.network_address:
        return [f"{address} is the address of the subnet {network} itself, not of a host in it"]
    if network.prefixlen <= LONGEST_BROADCAST_PREFIX and interface.ip == network.broadcast_address:
        return [f"{address} is the broadcast address of {network}, not a host's"]
    return []


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"network_address_problems": network_address_problems}
