"""The network_netplan_problems filter, the network role's check of netplan's settings.

It reads what netplan makes of its files with the direct link's among them, in a scratch root. The
settings may hold secrets, such as Wi-Fi passwords, so no message quotes them.
"""

from fnmatch import fnmatchcase

import yaml

# A match: name that holds one of these may fit more than one interface: networkd reads globs, and
# a leading "!" inverts the test.
GLOB_CHARACTERS = frozenset("*?[!")


def _error(result, root):
    """Return the line of netplan's error, with the scratch root's path cut out.

    The warnings come before it, and the lines below it quote the line of the file at fault, which
    may hold a secret.
    """
    for line in result.get("stderr", "").splitlines():
        if line.startswith(("Command failed: ", "ERROR", root)):
            return line.removeprefix("Command failed: ").replace(root, "").rstrip(":. ")
    return result.get("msg") or f"netplan exited with {result.get('rc')}"


def _globs(value):
    """Return the globs networkd reads from a match: value, which netplan writes as is.

    networkd splits the value at whitespace, and a "!" before the list inverts the test of each.
    """
    text = " ".join(map(str, value)) if isinstance(value, list) else str(value)
    globs = text.removeprefix("!").split()
    return [f"!{glob}" for glob in globs] if text.startswith("!") else globs


def _passes(globs, value):
    """Tell whether value passes networkd's test of globs, which fails where a "!" glob fits."""
    if any(fnmatchcase(value, glob[1:]) for glob in globs if glob.startswith("!")):
        return False
    wanted = [glob for glob in globs if not glob.startswith("!")]
    return not wanted or any(fnmatchcase(value, glob) for glob in wanted)


def _fits(match, port):
    """Tell whether netplan's match: fits the port, as networkd's [Match] tests it.

    A name fits the port's name or any of its alternative names, and macaddress the permanent
    address. A driver udev doesn't name, and a key netplan adds later, count as a fit. The check
    then refuses rather than misses an entry.
    """
    names = [port["name"], *port["altnames"]]
    for key, value in match.items():
        globs = _globs(value)
        if key == "name" and not any(_passes(globs, each) for each in names):
            return False
        if key == "macaddress" and str(value).lower() != port["macaddress"].lower():
            return False
        if key == "driver" and port["driver"] and not _passes(globs, port["driver"]):
            return False
    return True


def _alone(match):
    """Tell whether a match: that fits the port fits no other interface.

    A permanent MAC address is one interface's, and so is a name without a glob, as the kernel
    gives no two interfaces one name.
    """
    literal = "name" in match and not GLOB_CHARACTERS & set(str(match["name"]))
    return "macaddress" in match or literal


def network_netplan_problems(generated, settings, root, port):
    """Return what keeps netplan from setting up the direct link on port alone, as text.

    generated is the result of netplan's configure, run on root, and settings that of
    `netplan get` there. port holds the interface's name, its alternative names, its permanent MAC
    address and its driver. The template's comment says why an entry of another name that fits the
    port is refused. A match: on the port's own entry that may fit other interfaces is refused
    too, as they would take the direct link's settings.
    """
    name = port["name"]
    if generated.get("rc") != 0:
        return [f"netplan rejects its files with the direct link's: {_error(generated, root)}."]
    if settings.get("rc") != 0 or "Command failed" in settings.get("stderr", ""):
        return [f"netplan can't read its settings: {_error(settings, root)}."]
    try:
        network = (yaml.safe_load(settings.get("stdout", "")) or {}).get("network") or {}
    except yaml.YAMLError:
        return ["netplan get printed settings that aren't YAML."]
    ethernets = network.get("ethernets") or {}
    entry = ethernets.get(name) or {}
    problems = []
    renderer = entry.get("renderer", network.get("renderer", "networkd"))
    if renderer != "networkd":
        problems.append(
            f"netplan hands {name} to {renderer}, not networkd, which the role sets up the direct"
            " link with."
        )
    match = entry.get("match") or {}
    if not _fits(match, port):
        problems.append(
            f"netplan's entry {name} has a match: that {name} doesn't fit, so the direct link would"
            " go to another interface."
        )
    elif "match" in entry and not _alone(match):
        problems.append(
            f"netplan's entry {name} has a match: that may fit other interfaces too, whose DHCP the"
            f" direct link would turn off: match {name}'s MAC address, or a name without a glob."
        )
    for other, definition in ethernets.items():
        if other == name or not isinstance(definition, dict):
            continue
        # An entry without a match: matches its own name, which may be the port's alternative one.
        if _fits(definition.get("match", {"name": other}) or {}, port):
            problems.append(
                f"netplan's entry {other} fits {name}, and keeps a networkd file apart from the"
                f" direct link's: move its settings to {name}, or narrow its match: to another"
                " interface."
            )
    return problems


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"network_netplan_problems": network_netplan_problems}
