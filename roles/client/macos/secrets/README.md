# The secrets on the Mac

The Mac keeps ws's secrets in fnox, in the private secrets repository, encrypted to the YubiKey 5C
Nano, to a backup key and to each server's host key. This role installs the tools that do it.

## The tools

[`mise.mac.toml`](mise.mac.toml) pins fnox 1.50.0 and age-plugin-yubikey 0.5.1, each from its GitHub
release, built for macOS on arm64. Both sides pin one fnox, as they read one secrets repository.
fnox's pin names mise's `github` backend, `github:jdx/fnox`, as the server's
[fnox](../../../server/ubuntu/secrets/README.md#fnox) does. The registry's `packslip` backend asks
GitHub's API for the release on a first install, unauthenticated. A CI runner meets the API's limit
there. age-plugin-yubikey takes the registry's name.

[`mise.mac.lock`](../../mise/mise.mac.lock) holds each one's URL and SHA-256 for macOS on arm64.
The bootstrap runs locked, so mise downloads from the lock's URL alone, and refuses a tool the lock
lacks. Neither publisher lists a checksum, so each SHA-256 is GitHub's digest of its file. mise
checked fnox's against the release's attestations as it wrote the lock. To move either on, change
the pin and lock it again, by `MISE_ENV=mac mise -C roles/client lock --platform macos-arm64` in a
copy of the tree. `TestMacLockHoldsEachTool` checks that the lock holds each tool, and no tool that
no role pins.

fnox encrypts a value to the Nano's `age1yubikey1` recipient by running age-plugin-yubikey from
`PATH`, and needs no YubiKey for it. To decrypt one, the plugin asks for the Nano, its PIN once a
session and a touch each time. Without the plugin on `PATH`, fnox fails either way.

## The units

mise installs both tools for the tree alone, on no shell's `PATH`. fnox must find the plugin on
`PATH`, and the maintainer runs `fnox` by hand in a clone of the secrets repository. So
[the role's units](units.toml.tmpl) copy them where every shell finds them, as `state.gh` does
([gh](../git/README.md#gh)):

- `state.fnox` copies fnox to `/usr/local/bin/fnox`, root:wheel and 755.
- `state.age-plugin-yubikey` copies age-plugin-yubikey to `/usr/local/bin/age-plugin-yubikey`,
  alike.

`/etc/paths` puts `/usr/local/bin` on every shell's `PATH`. Each copy goes in under a temporary name
first, and one there that differs is replaced. The check runs as the user, so a bootstrap that
changes nothing asks no password. Where mise has no such tool, its unit alone fails.
