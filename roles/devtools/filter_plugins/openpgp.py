"""The openpgp_fingerprint filter, which the devtools role checks the apt keys it fetches with.

Python, as gpg is not on every machine the playbook runs on: an Ubuntu container lacks it.
"""

import base64
import hashlib

from ansible.errors import AnsibleFilterError


def _packet(data):
    """Returns the tag and body of the first OpenPGP packet in data (RFC 9580, section 4.2)."""
    if not data or not data[0] & 0x80:
        raise AnsibleFilterError("openpgp_fingerprint: not an OpenPGP packet")
    if data[0] & 0x40:
        tag = data[0] & 0x3F
        first = data[1]
        if first < 192:
            start, length = 2, first
        elif first < 224:
            start, length = 3, ((first - 192) << 8) + data[2] + 192
        elif first == 255:
            start, length = 6, int.from_bytes(data[2:6], "big")
        else:
            raise AnsibleFilterError("openpgp_fingerprint: a partial length in a key packet")
    else:
        tag = (data[0] >> 2) & 0x0F
        size = {0: 1, 1: 2, 2: 4}.get(data[0] & 0x03)
        if size is None:
            raise AnsibleFilterError("openpgp_fingerprint: an indeterminate packet length")
        start, length = 1 + size, int.from_bytes(data[1 : 1 + size], "big")
    return tag, data[start : start + length]


def openpgp_fingerprint(armored):
    """Returns the fingerprint of the primary key of an armored public key, in upper-case hex."""
    lines = armored.strip().splitlines()
    try:
        begin = lines.index("-----BEGIN PGP PUBLIC KEY BLOCK-----")
        blank = lines.index("", begin)
    except ValueError as err:
        raise AnsibleFilterError("openpgp_fingerprint: not an armored public key") from err
    body = []
    for line in lines[blank + 1 :]:
        if line.startswith(("=", "-----END")):
            break
        body.append(line)
    tag, packet = _packet(base64.b64decode("".join(body)))
    if tag != 6:
        raise AnsibleFilterError("openpgp_fingerprint: the first packet is not a public key")
    # v4 hashes the packet with SHA-1, v6 with SHA-256, each behind its own prefix (section 5.5.4).
    if packet[0] == 4:
        return hashlib.sha1(b"\x99" + len(packet).to_bytes(2, "big") + packet).hexdigest().upper()
    if packet[0] == 6:
        return hashlib.sha256(b"\x9b" + len(packet).to_bytes(4, "big") + packet).hexdigest().upper()
    raise AnsibleFilterError(f"openpgp_fingerprint: a version {packet[0]} key")


class FilterModule:
    """The role's filters."""

    def filters(self):
        return {"openpgp_fingerprint": openpgp_fingerprint}
