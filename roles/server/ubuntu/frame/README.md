# The frame

The frame is what each role of mise's config leans on. One `mise bootstrap` sets up the server, in
the side's run directory, `roles/server` of the checkout at `/opt/ws`. mise's own parts run first,
then ws's units, which mise hands to `ws unit` through the plugin `ws`. A role moved to mise is its
config, with its spec where it has units, and holds no Go.

## Its files

| File | What it holds |
| --- | --- |
| [`mise.toml`](../../../../mise.toml) | Both sides': mise's settings, among them `locked`, and the pin of ws, a release, for three platforms. |
| [`mise.lock`](../../../../mise.lock) | The release's SHA-256 for each platform, which mise checks as it installs ws. |
| [`roles/server/mise/mise.server.lock`](../../mise/mise.server.lock) | The roles' tools: each download's url and SHA-256 for each platform ([devtools](../devtools/README.md#the-tools)). |
| [`mise.server.toml`](../../../../mise.server.toml) | The server's: dpkg's options, `/etc/ws`, the check against a silent skip and [the check of the values](#the-check-of-the-values). |
| [`plugins/ws`](../../../../plugins/ws) | The plugin: a hook runs one `ws unit read` or `ws unit write` ([ws unit](../../../../engine/core/kind/README.md#ws-unit)), linked as [a system plugin](#the-plugin). |
| `units.toml.tmpl`, `mise.server.toml`, `files/` here | The role `frame`: the unit that writes [the values file](#the-values-file). |

The Mac shares `mise.toml` alone, and loads no file of the server's.

## A role's layout

A role keeps its directory, `roles/server/ubuntu/ROLE/`, and its page there:

- **Its spec,** `units.toml.tmpl`, where it has units, with its templates and sources beside it
  ([The specs](../../../../engine/core/kind/README.md#the-specs)). A path in the spec is relative to
  the directory. Every Go template ends in `.tmpl`: the spec, and each file that a `template`
  names, as `ws.conf.tmpl`. A kind refuses a `template` that lacks the ending, as the spec loads.
  A `.toml` file is TOML as written, which tombi checks.
- **Its mise files,** one an env. `mise.server.toml` holds what every server gets: apt's sources,
  `[bootstrap.packages]`, files, directories, tools, the services and the firewall, with each file
  entry that names a service in `notify`. `mise.secrets.toml` holds its
  [secret inputs](#secret-inputs). A role holds no plain `mise.toml`, which every env would load.
- **The side's link,** `roles/server/mise/conf.d`, to `../ubuntu`. mise loads each role's
  directory through it as a fragment, and a relative `source` resolves from the role's
  directory. A role lands with its directory alone, and edits no shared file.

mise merges `[bootstrap.packages]` across fragments, but an entry of a path or a service is whole.
A later env's entry replaces an earlier one. Where two roles declare one path, mise keeps the later
fragment's by name and drops the other with no error: a path is one role's. mise makes no missing
parent, so a new directory of mise's files is a `[bootstrap.directories]` entry.

## The lines

Each unit of a spec is a line `"ws:KIND/NAME" = "in-sync"` under `[bootstrap.packages]`, as mise
plans the units its files name alone. `ws unit list --toml` prints them.

- A unit for every server goes in the role's `mise.server.toml`.

A value never takes a unit out of a spec. A unit that a value turns off stays in the spec, off by
its `when`. Else its line would name a unit that `ws unit read` refuses.

[`TestSpecsMatchMiseLines`](../../cli/specs_test.go) loads the specs for the defaults. It fails on
a line missing or extra, and prints the line to paste. `TestUnitsFollowNoValue` fails where a bool
or a string turned over changes the units.
`TestTheSideLinksItsRoles` fails where the link leads elsewhere or its directory holds a file, and
on a mise file that no env loads.

## The order

A run of `mise bootstrap` takes its parts in mise's order: the plugins, the files, the packages,
the services, the firewall and the tools. Then it hands every role's units to one `ws unit write`.
ws runs them by `KIND/NAME`, each moved behind the unit that its `after` names, across roles. So:

- A unit that needs another role's names it in `after`. A name is unique across roles: start it
  with the role's name.
- A service of mise's runs before any unit. A service that needs a unit's package or file is a
  `service` unit instead.

## The machine's env

The machine's miserc, `/etc/mise/miserc.toml`, names the envs that load as the set `ws`: `server`
alone. A run by `-E ws,secrets`, the secrets task's, adds the env `secrets` to the set, which no
other run loads. `ws server setup` writes the miserc, as `install.sh` does.

The same command writes `/etc/mise/config.toml`, which trusts the tree: its `trusted_config_paths`,
under `[settings]`, names `/opt/ws`, so that a run by hand needs no `mise trust`. mise reads that
setting of a system config under `[settings]` alone. `mise.toml` sets `locked`, so that a tool
installs only as `mise.lock` holds it. The secrets task checks that setting, so that fnox runs
from the lock alone.
`mise settings get locked` reads the config of the directory it runs in: it prints `true` in the
tree and `false` outside it.

## Secret inputs

A role's secret is an input of mise's, which its `mise.secrets.toml` declares. The env `secrets`
alone loads the file, so the [secrets task](../secrets/README.md#the-task) alone gives the inputs
their values, from fnox. The file holds `[bootstrap.secrets]` and `[bootstrap.files."PATH"]`, and
no unit's line. Each input takes one form, a comment above it:

```toml
WS_GITHUB_TOKEN = { env = "WS_GITHUB_TOKEN", allow_empty = true }
```

- The name is the input's, its variable's and fnox's, and one role declares it. A run with the env
  needs each input set, empty where a server has no value.
- A file of an input's is a `template`, root's, mode `0600`, with `remove_empty`: an empty input
  removes it.
- The file's content opens with a guard that mise's `exec` runs, in a plan and a run alike. Where
  the path holds a file whose first line is not ws's header, both stop and name it, and mise
  replaces nothing. mise's files step runs before ws's units, so no unit could guard it.
- A run without the env, as CI's `server` bootstrap, loads none of these files. It keeps the secret
  files as they are, a file by hand included, and its plan lists none.
- A unit reads a file that the same run wrote: mise's files step comes first. A plan writes none.
  So a template's `envvar` takes an input from the run's env in the secrets task's run alone, and
  the plan shows what the run changes
  ([The specs](../../../../engine/core/kind/README.md#the-specs)). So do the commands of ws that
  read git's tokens and gh's ([Git credentials](../git/README.md#git-credentials),
  [gh](../devtools/README.md#gh)). An input may write no file, as
  ikeward's profile, which ws's own commands read from the run's env
  ([The profile's inputs](../ikeward/README.md#the-profiles-inputs)).
- A run state drops each input from the env of a command whose program is not ws's
  ([run](../../../../engine/core/kind/state.md)), so that an installer never sees a value. The
  commands that ws runs for cld, claude and git start from a cleared env, with `HOME` and `PATH`
  alone.

[`secretfiles_test.go`](../../cli/secretfiles_test.go) holds each file to the form and its guard,
and the inputs to the list that a run drops.

## The plugin

`install.sh` and `ws server setup` link the tree's `plugins/ws` into `/usr/share/mise/plugins`,
before the first run.
mise finds a system plugin there and loads it as it stands, as no entry names it. A
`[bootstrap.plugins]` entry would install it again on every run, with a warning. mise refuses a
system plugin that anyone but root may change, or a directory on its way, `/opt` included. It
drops the units then, with a warning alone. So both refuse `/`, `/usr`, `/usr/share` or
`/opt` where a group or others may write it, and make `/usr/share/mise` and its `plugins` root's
and 755, whatever was there. After that, the [check](#the-check-against-a-silent-skip) fails a run
that drops them. Its other
system directory, `/usr/local/share/mise/plugins`, stays empty here. Root's own,
`/root/.local/share/mise/plugins`, is where mise installs a plugin, no system one.

## The pin

`mise.toml` pins a release of ws, which `mise.lock` checks by SHA-256 for `linux-x64`,
`linux-arm64` and `macos-arm64`. A spec that needs new Go lands after the release that holds it.
[`TestPinLocksEachTarget`](../../../../cmd/pin_test.go) holds the version, the platforms and their
URLs together in both files. `mise.local.toml` and `mise.local.lock`, which git ignores, pin a
build of the tree in their place. [`.tools/serve-build`](../../../../.tools/serve-build) writes
them, beside a build of ws and of serve-releases, which serves it on loopback. CI's `server` job
and a trial of a role run so, as a spec may need what the release lacks.

The pin moves in a pull request of its own, after the release it names
([Cutting a release](../../../../.github/workflows/release.md#cutting-a-release)).

## ws at a fixed path

mise installs ws under its data directory, by version, which neither a unit of systemd nor a
person can name. The unit `state/ws-binary` copies the pinned binary to `/usr/local/bin/ws` where
the two differ, under a temporary name first. A unit that runs `/usr/local/bin/ws` waits on it by
`after`.

## The values file

The unit `file/ws-values` writes `/etc/ws/values.yml` where nothing is at the path, a link
included, as a `seed` file ([file](../../../../engine/core/kind/file.md)). The file, root's at
600, goes in whole under that mode. `Values.YAML` gives its text from the struct, so that a role's
new key needs no second list. It names each key of `values.Values`, a table too, at the value a
missing file gives. A key whose default is derived holds its value as the file is made:
`WS_GIT_SIGNING_PUB_KEY` is under root's home, `WS_TUNNEL_PEERS` the Mac at the tunnel's next
address. A key that a run reads from the machine, such as the swap's size, stays empty
or `null`.

The file is then the owner's: no run reads it as ws's, or changes it. Each call of ws reads the
values as it starts, and the text reads as no file does, so the unit needs no `after`. CI's
`server` job checks that no run changed the file
([The server job](../../../../.github/workflows/server.md)).

## The check against a silent skip

mise drops the units of a plugin that it refuses, with a warning and exit 0. The server's
`/etc/ws/units.verified` is a template that runs `ws unit verify`, so the plan and the run fail
where mise's plan lacks a unit of the specs. It runs `/usr/local/bin/ws`, and fails a run without
it, naming `install.sh`. The plugin's write hook runs `ws unit verify` again before `ws unit
write`.

The check runs in the run's directory, where a nested mise loads the roles as the run does. A run
from `/opt/ws`, or from the Mac's `roles/client`, loads no role of the server's, and a first one
would set up the frame alone with exit 0. So the check looks for the frame's own
`mise.server.toml` through the run's link. It fails a run without it before any phase, and names
`mise -C /opt/ws/roles/server`, as its test proves.

## The check of the values

A bad value in `/etc/ws/values.yml` stops a plan or a run before any write. The server's
`/etc/ws/values.checked` is a template that runs `ws server values check`, which loads the file as
each command does. A missing file passes. A bad value fails with its key, its rule and the file
to edit by hand: `values set` refuses while any value is bad, so the message never names it. A pass
prints one fixed line, so the next plan finds the file unchanged.

The template runs `/usr/local/bin/ws`, and fails a run without it, naming `install.sh`. CI's
`server` job plans once with a bad value. It expects exit 1 from `ws server values check` that
names the key, and puts the file back ([server-plan.sh](../../../../.github/server-plan.sh)).
