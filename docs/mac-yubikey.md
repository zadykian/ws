# The YubiKey

The Mac's key for ws and git's signing key live in the PIV applet of a YubiKey 5C Nano. Its firmware
is 5.7 or later, which makes Ed25519 keys. `mac/setup.sh` reads it by ykman ([The Mac](mac.md)).

## mac.conf's names

`WS_SSH_KEY` names the key that ssh offers ws, `ws` by default, and `WS_GIT_KEY` the one that signs
git's commits, `git` by default. A name is 1 to 32 of `a-z`, `0-9` and `-`, and the two differ, as
each key has one use. A key's name is the subject of its slot's certificate, `CN=NAME`, and its
public key goes in `~/.config/ws/keys/NAME.pub`. The wizard asks neither name.

## The keys

`setup.sh` installs Homebrew's ykman where `PATH` has none, and Homebrew's yubico-piv-tool, whose
libykcs11 gives ssh's agent the keys, where Homebrew lacks it. `--check` names the install. The
script stops unless one YubiKey is in, and names each by its serial where several are, as a second
in the dock would be. Each later call of ykman names the one by `--device SERIAL`.

## The PIV applet's defaults

A YubiKey comes with the PIN `123456`, the PUK `12345678` and a management key that anyone knows.
With them, any program on the Mac could replace or delete the keys. `setup.sh` warns at each run
where `ykman piv info` reports a default, and changes none, as a PUK it chose would be lost to the
user. Set them once, by hand:

```sh
ykman piv access change-pin
ykman piv access change-puk
ykman piv access change-management-key --protect
```

The last keeps a new, random management key on the YubiKey, behind the PIN, so that ykman asks the
PIN alone.
