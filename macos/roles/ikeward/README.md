# ikeward on the Mac

ikeward is a second way to the networks of ikeward's VPN, from a server's profile: the Mac's tunnel
reaches them through that server alone. This role installs what ikeward needs, strongSwan and
ikeward's own binary. It sets nothing up and starts nothing. `ws macos ikeward install` and `on`
do that, by hand.

## strongSwan

[`mise.macos.toml`](mise.macos.toml) names `"brew:strongswan" = "latest"`, as mise's formula. Homebrew
keeps the current bottle alone, so the pin takes the latest. mise checks the bottle against the
SHA-256 of Homebrew's API at run time, and ws pins none, as for the other
[formulae](../devtools/README.md#the-formulae). The formula is outside the SHA-256 rule.

## ikeward

mise pins `"github:zadykian/ikeward"` by its version, from the release's `ikeward-darwin-arm64`.
[`mise.macos.lock`](../../mise/mise.macos.lock) holds its URL and SHA-256 for macOS on arm64, so the
bootstrap, which runs locked, downloads from that URL alone and checks the file. The pin names mise's
`github` backend, as fnox's does ([the tools](../secrets/README.md#the-tools)), not the `http`
backend, and no kind of ws's. To move it on, change the pin and lock it again, by `MISE_ENV=macos
mise -C macos lock --platform macos-arm64` in a copy of the tree.

The server's ikeward is pinned apart, in [its unit](../../../ubuntu/roles/ikeward/units.toml.tmpl),
and a release for the Mac changes nothing there. A fix of the Mac's side is a new tag, and a new
pin here.

## The unit

mise installs ikeward for the tree alone. [`state.ikeward`](units.toml.tmpl) copies it to
`/usr/local/sbin/ikeward`, root:wheel and 0755:

- It makes `/usr/local/sbin` where a clean Mac lacks it, and keeps it root:wheel and 0755, so that
  the user cannot swap the binary there.
- The copy goes in under a temporary name first, and one there that differs is replaced.
- The check runs as the user, with `cmp` and `stat` of the directory and the file, so a bootstrap
  that changes nothing asks no password. Where mise has no ikeward, the unit alone fails.
- The role reads no value, and no secret.

`/usr/local/sbin` is on no `PATH`, as `/etc/paths` lists `/usr/local/bin` and not it. So a shell
finds no `ikeward`, and the pages and the runbook name `/usr/local/sbin/ikeward` in full. ws runs
it by sudo, by that path.

## Root runs the user's files

ikeward starts Homebrew's charon, and runs swanctl, as root. Their files and libraries are the
user's, under `/opt/homebrew`, as Homebrew refuses root. A process of the user's that replaces one
can become root at the next `install` or `on`. The copy is such a path too: root copies mise's
ikeward, the user's file, with no SHA-256 check at that time, as mise checked it at install. This
breaks [the tunnel's rule](../../README.md#the-macs-tunnel), where root runs no file of the
user's. A person's command runs the binary, by sudo. Setup asks sudo for the copy where it
differs, and a converged run changes nothing.
