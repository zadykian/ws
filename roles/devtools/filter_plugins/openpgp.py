"""The openpgp_fingerprint filter, which the devtools role checks the apt keys it fetches with.

Python, as gpg is not on every machine the playbook runs on: an Ubuntu container lacks it.
"""

import base64
import hashlib

from ansible.errors import AnsibleFilterError

# A new-format length's first octet (RFC 9580, section 4.2.1): the length itself below 192, the
# first of two octets below 224, and 255 before four octets. The others are partial lengths.
ONE_OCTET_LIMIT = 192
TWO_OCTET_LIMIT = 224
FIVE_OCTET = 255
# The tag of a public key packet (section 5.5.1.1), and the key versions the filter takes.
PUBLIC_KEY_TAG = 6
V4 = 4
V6 = 6


def _packet(data):
    """Return the tag and body of the first OpenPGP packet in data (RFC 9580, section 4.2)."""
    if not data or not data[0] & 0x80:
        raise AnsibleFilterError("openpgp_fingerprint: not an OpenPGP packet")
    if data[0] & 0x40:
        tag = data[0] & 0x3F
        first = data[1]
        if first < ONE_OCTET_LIMIT:
            start, length = 2, first
        elif first < TWO_OCTET_LIMIT:
            start, length = 3, ((first - ONE_OCTET_LIMIT) << 8) + data[2] + ONE_OCTET_LIMIT
        elif first == FIVE_OCTET:
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
    """Return the fingerprint of the primary key of an armored public key, in upper-case hex."""
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
    if tag != PUBLIC_KEY_TAG:
        raise AnsibleFilterError("openpgp_fingerprint: the first packet is not a public key")
    # v4 hashes the packet with SHA-1, v6 with SHA-256, each behind its own prefix (section 5.5.4).
    if packet[0] == V4:
        data = b"\x99" + len(packet).to_bytes(2, "big") + packet
        digest = hashlib.sha1(data)  # noqa: S324 - OpenPGP v4 fingerprints are SHA-1 (RFC 9580)
    elif packet[0] == V6:
        digest = hashlib.sha256(b"\x9b" + len(packet).to_bytes(4, "big") + packet)
    else:
        raise AnsibleFilterError(f"openpgp_fingerprint: a version {packet[0]} key")
    return digest.hexdigest().upper()


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"openpgp_fingerprint": openpgp_fingerprint}
