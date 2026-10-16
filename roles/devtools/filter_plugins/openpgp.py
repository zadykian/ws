"""The openpgp_fingerprints filter, which the devtools role checks the apt keys it fetches with.

Python, as gpg is not on every machine the playbook runs on: an Ubuntu container lacks it.
"""

import base64
import binascii
import hashlib
import re

from ansible.errors import AnsibleFilterError

# The armor's first and last lines, and a header, a key and its value (RFC 9580, section 6.2).
BEGIN = "-----BEGIN PGP PUBLIC KEY BLOCK-----"
END = "-----END PGP PUBLIC KEY BLOCK-----"
HEADER = re.compile(r"[A-Za-z][A-Za-z0-9-]*:(?: .*)?")
# A line of base64, and the checksum's, an = and three octets of base64 (sections 6 and 6.1).
BASE64 = re.compile(r"[A-Za-z0-9+/]*={0,2}")
CHECKSUM = re.compile(r"=[A-Za-z0-9+/]{4}")
# The blanks apt strips from each line of a .asc key, C's isspace.
BLANKS = " \t\n\v\f\r"
# A new-format length's first octet (section 4.2.1): the length itself below 192, the first of two
# octets below 224, and 255 before four octets. The others are partial lengths.
ONE_OCTET_LIMIT = 192
TWO_OCTET_LIMIT = 224
FIVE_OCTET = 255
# The tag of a public key packet (section 5.5.1.1), and the key versions the filter takes.
PUBLIC_KEY_TAG = 6
V4 = 4
V6 = 6
# The packets a certificate holds after its public key (section 10.1): signatures, markers, user
# IDs, subkeys, user attributes and padding. Any other, such as a secret key, fails the filter.
CERTIFICATE_TAGS = frozenset({2, 10, 13, 14, 17, 21})


def _dearmor(armored):
    """Return the data of armored, which must be one armored public key block alone.

    apt dearmors a .asc key itself, and reads on into any block that follows. It decodes each line
    after the headers but those that start with = or -, and skips what is not base64. So each line
    here must be base64, and the file must hold this block alone.
    """
    lines = [line.strip(BLANKS) for line in armored.strip(BLANKS).split("\n")]
    blank = lines.index("") if "" in lines else len(lines)
    headers, body = lines[1:blank], lines[blank + 1 : -1]
    if body and CHECKSUM.fullmatch(body[-1]):
        body.pop()
    if (
        lines[0] != BEGIN
        or lines[-1] != END
        or not all(map(HEADER.fullmatch, headers))
        or not all(map(BASE64.fullmatch, body))
    ):
        raise AnsibleFilterError("openpgp_fingerprints: not one armored public key alone")
    try:
        return base64.b64decode("".join(body), validate=True)
    except binascii.Error as err:
        raise AnsibleFilterError(f"openpgp_fingerprints: the armor's base64: {err}") from err


def _packets(data):
    """Yield the tag and body of each OpenPGP packet in data (RFC 9580, section 4.2)."""
    offset = 0
    while offset < len(data):
        # A header cut short reads as zeros, so that its packet ends past the data and fails below.
        head = data[offset : offset + 6].ljust(6, b"\0")
        if not head[0] & 0x80:
            raise AnsibleFilterError("openpgp_fingerprints: not an OpenPGP packet")
        if head[0] & 0x40:
            tag = head[0] & 0x3F
            first = head[1]
            if first < ONE_OCTET_LIMIT:
                start, length = 2, first
            elif first < TWO_OCTET_LIMIT:
                start, length = 3, ((first - ONE_OCTET_LIMIT) << 8) + head[2] + ONE_OCTET_LIMIT
            elif first == FIVE_OCTET:
                start, length = 6, int.from_bytes(head[2:6], "big")
            else:
                raise AnsibleFilterError("openpgp_fingerprints: a partial length in a key packet")
        else:
            tag = (head[0] >> 2) & 0x0F
            size = {0: 1, 1: 2, 2: 4}.get(head[0] & 0x03)
            if size is None:
                raise AnsibleFilterError("openpgp_fingerprints: an indeterminate packet length")
            start, length = 1 + size, int.from_bytes(head[1 : 1 + size], "big")
        end = offset + start + length
        if end > len(data):
            raise AnsibleFilterError("openpgp_fingerprints: a packet cut short")
        yield tag, data[offset + start : end]
        offset = end


def _fingerprint(packet):
    """Return the fingerprint of a public key packet's body, in upper-case hex."""
    version = packet[0] if packet else None
    # v4 hashes the packet with SHA-1, v6 with SHA-256, each behind its own prefix (section 5.5.4).
    if version == V4:
        data = b"\x99" + len(packet).to_bytes(2, "big") + packet
        digest = hashlib.sha1(data)  # noqa: S324 - OpenPGP v4 fingerprints are SHA-1 (RFC 9580)
    elif version == V6:
        digest = hashlib.sha256(b"\x9b" + len(packet).to_bytes(4, "big") + packet)
    else:
        raise AnsibleFilterError(f"openpgp_fingerprints: a version {version} key")
    return digest.hexdigest().upper()


def openpgp_fingerprints(armored):
    """Return the fingerprint of each certificate's primary key in an armored public key.

    Each public key packet starts a certificate, and every packet must belong to one. So the list
    names every key apt would trust from the file, and a subkey belongs to its certificate.
    """
    fingerprints = []
    for tag, body in _packets(_dearmor(armored)):
        if tag == PUBLIC_KEY_TAG:
            fingerprints.append(_fingerprint(body))
        elif not fingerprints or tag not in CERTIFICATE_TAGS:
            raise AnsibleFilterError(f"openpgp_fingerprints: a tag {tag} packet in no certificate")
    return fingerprints


class FilterModule:
    """The role's filters."""

    def filters(self):
        """Return the filters by name."""
        return {"openpgp_fingerprints": openpgp_fingerprints}
