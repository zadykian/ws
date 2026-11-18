# The frame

The frame is what each role of mise's config leans on. One `mise bootstrap` in the checkout at
`/opt/ws` sets up the server: mise's own parts first, then ws's units, which mise hands to
`ws unit` through the plugin `ws`. A role moved to mise is its config and its spec, and holds no
Go.

## Its files

| File | What it holds |
| --- | --- |
| [`mise.toml`](../../../../mise.toml) | Both sides': mise's settings and the pin of ws, a release, for three platforms. |
| [`mise.lock`](../../../../mise.lock) | The release's SHA-256 for each platform, which mise checks as it installs ws. |
| [`mise/mise.server.lock`](../../../../mise/mise.server.lock) | The roles' tools: each download's url and SHA-256 for each platform ([devtools](../devtools/README.md#the-tools)). |
| [`mise.server.toml`](../../../../mise.server.toml) | The server's: the plugin, dpkg's options, `/etc/ws` and the check against a silent skip. |
| [`plugins/ws`](../../../../plugins/ws) | The plugin: a hook runs one `ws unit read` or `ws unit write` ([ws unit](../../../../engine/core/kind/README.md#ws-unit)). |
| `units.toml`, `mise.server.toml` here | The role `frame`: the unit that keeps `/usr/local/bin/ws`. |

The Mac shares `mise.toml` alone, and loads no file of the server's.

## A role's layout

A role keeps its directory, `roles/server/ubuntu/ROLE/`, and its page there:

- **Its spec,** `units.toml`, with its templates and sources beside it
  ([The specs](../../../../engine/core/kind/README.md#the-specs)). A path in the spec is relative to
  the directory. A file that holds a template's marks takes an ending other than `.toml`, as
  `ws.conf.tmpl`, since tombi checks every other `.toml` as TOML.
- **Its mise files,** one an env. `mise.server.toml` holds what every server gets: apt's sources,
  `[bootstrap.packages]`, files, directories and tools. `mise.booted.toml` holds the services, the
  firewall, and each file entry that names a service in `notify`. `mise.account.toml` holds what
  an account gets where `WS_USER` is not root. A role holds no plain `mise.toml`, which the Mac
  would load too.
- **Its link,** `mise/conf.d/server-ROLE`, to `../../roles/server/ubuntu/ROLE`. mise loads the
  directory through it as a fragment, and a relative `source` resolves from the role's
  directory. A role lands with its directory and its link, and edits no shared file.

mise merges `[bootstrap.packages]` across fragments, but an entry of a path or a service is whole.
A later env's entry replaces an earlier one, so a file's entry in `mise.booted.toml` repeats the
whole entry with its `notify`. Where two roles declare one path, mise keeps the later fragment's
by name and drops the other with no error: a path is one role's. mise makes no missing parent, so
a new directory of mise's files is a `[bootstrap.directories]` entry.

## The lines

Each unit of a spec is a line `"ws:KIND/NAME" = "in-sync"` under `[bootstrap.packages]`, as mise
plans the units its files name alone. `ws unit list --toml` prints them.

- A unit for every server goes in the role's `mise.server.toml`.
- A unit that the spec gives under `{{ if ne .User "root" }}` goes in its `mise.account.toml`.
- No unit goes in `mise.booted.toml`: a unit that needs booted systemd skips itself without it.

The envs follow `WS_USER` alone, so a unit's presence in a spec does too. A unit for root alone
has no env. A unit that another value turns off stays in the spec, off by its `when`. Else its
line would name a unit that `ws unit read` refuses.

[`TestSpecsMatchMiseLines`](../../cli/specs_test.go) loads the specs for root and for an account. It
fails on a line missing or extra, and prints the line to paste. It fails on a unit for root alone.
`TestUnitsFollowTheUserAlone` turns each other value over and fails where the units change.
`TestEachRoleIsLinked` fails on a role without its link, or a mise file that no env loads.

## The order

A run of `mise bootstrap` takes its parts in mise's order: the plugins, the files, the packages,
the services, the firewall and the tools. Then it hands every role's units to one `ws unit write`.
ws runs them by `KIND/NAME`, each moved behind the unit that its `after` names, across roles. So:

- A unit that needs another role's names it in `after`. A name is unique across roles: start it
  with the role's name.
- A service of mise's runs before any unit. A service that needs a unit's package or file is a
  `service` unit instead.

## The machine's env

The machine's miserc, `/etc/mise/miserc.toml`, names the envs that load: `server`, then `booted`
where `/run/systemd/system` is a directory, then `account` where `WS_USER` is not root. `booted`
comes after `server`, so that its entries win. The miserc is a template of the machine's own,
which cannot read `/etc/ws/values.yml`. `install.sh` writes it from there, so a change of `WS_USER`
means running `install.sh` again.

## The pin

`mise.toml` pins a release of ws, which `mise.lock` checks by SHA-256 for `linux-x64`,
`linux-arm64` and `macos-arm64`. A spec that needs new Go lands after the release that holds it.
[`TestPinLocksEachTarget`](../../../../cmd/pin_test.go) holds the version, the platforms and their
URLs together in both files. `mise.local.toml` and `mise.local.lock`, which git ignores, pin a
build of the tree in their place. [`.tools/serve-build`](../../../../.tools/serve-build) writes
them, beside a build of ws and of serve-releases, which serves it on loopback. CI's `playbook` job
and a trial of a role run so, as a spec may need what the release lacks.

The pin moves in a pull request of its own, after the release it names
([Cutting a release](../../../../.github/workflows/release.md#cutting-a-release)).

## ws at a fixed path

mise installs ws under its data directory, by version, which neither a unit of systemd nor a
person can name. Every run as root exports `MISE_DATA_DIR=/usr/local/share/mise`, as `install.sh`
does, so that the account reaches the tools a role pins
([devtools](../devtools/README.md#the-tools)). The unit `state/ws-binary` copies the pinned binary
to `/usr/local/bin/ws` where the two differ, under a temporary name first. As a `run` state, it
skips without booted systemd, as in CI's container. A unit that runs `/usr/local/bin/ws` waits on
it by `after`.

## The check against a silent skip

mise drops the units of a plugin that it refuses, with a warning and exit 0. The server's
`/etc/ws/units.verified` is a template that runs `ws unit verify`, so the plan and the run fail
where mise's plan lacks a unit of the specs. On a first run, and on the run that brings a new pin,
mise renders it before it installs the pin's ws. It prints the line of a pass then, so that the
next plan finds the file unchanged. Each unit reads as missing on that run, so the plugin's write
hook runs, and it runs `ws unit verify` before `ws unit write`.
