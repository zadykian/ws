# The YubiKey

The Mac's key for ws and git's signing key live in the PIV applet of a YubiKey 5C Nano. Its firmware
is 5.7 or later, which makes Ed25519 keys.

## mac.conf's names

`WS_SSH_KEY` names the key that ssh offers ws, `ws` by default, and `WS_GIT_KEY` the one that signs
git's commits, `git` by default. A name is 1 to 32 of `a-z`, `0-9` and `-`, and the two differ, as
each key has one use. A key's name is the subject of its slot's certificate, `CN=NAME`, and its
public key goes in `~/.config/ws/keys/NAME.pub`. The wizard asks neither name.
