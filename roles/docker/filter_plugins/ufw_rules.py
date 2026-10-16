"""The docker_stale_ufw_rules filter, which finds the ufw rules the docker role adds no more."""

# A rule's fields in ufw's user.rules sit between "### tuple ###" and its comment. They are the
# action, the protocol, the destination's port and address, the source's port and address, then the
# direction with the interface, as in_docker0. A rule that names applications has two more.
RULE_FIELDS = 7


def _address(value):
    """Return value as the ufw module takes it: any for ufw's 0.0.0.0/0."""
    return "any" if value == "0.0.0.0/0" else value


def docker_stale_ufw_rules(rules, comment, to_ip, to_port, interfaces):
    """Return, as the ufw module's options, the rules with comment that the role adds no more.

    rules is the text of ufw's user.rules, which holds a rule as `### tuple ### allow any 53
    172.17.0.1 any 0.0.0.0/0 in_docker0 comment=HEX`, the comment in hex. The role allows any
    protocol, from any address, to port to_port of to_ip, in on each of interfaces. Any other rule
    of that form with comment is a former one, as for another to_ip or from a source range. A rule
    of another form, such as one that names a protocol, is left alone.
    """
    tag = "comment=" + comment.encode().hex()
    wanted = {
        ("allow", "any", str(to_port), to_ip, "any", "0.0.0.0/0", f"in_{name}")
        for name in interfaces
    }
    stale = []
    for line in rules.splitlines():
        fields = line.split()
        if fields[:3] != ["###", "tuple", "###"] or fields[-1] != tag:
            continue
        rule = tuple(fields[3:-1])
        if len(rule) != RULE_FIELDS or rule in wanted:
            continue
        action, proto, dport, dst, sport, src, direction = rule
        if (action, proto, sport) != ("allow", "any", "any") or dport == "any":
            continue
        if not direction.startswith("in_"):
            continue
        stale.append(
            {
                "rule": action,
                "interface_in": direction.removeprefix("in_"),
                "from_ip": _address(src),
                "to_ip": _address(dst),
                "to_port": dport,
            }
        )
    return stale


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"docker_stale_ufw_rules": docker_stale_ufw_rules}
