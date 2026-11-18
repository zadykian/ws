# The Mac's frame

The frame is what each of the Mac's roles in mise's config leans on. The user runs one
`mise -C /opt/ws bootstrap`: mise's own parts first, then ws's units, which mise hands to the Mac's
`ws unit` through the plugin `ws`. The Mac shares `mise.toml` alone with the server, and loads no
file of the server's.

## Its files

| File | What it holds |
| --- | --- |
| [`mise.toml`](../../../../mise.toml) | Both sides': mise's settings and the pin of ws, with `macos-arm64` among its platforms. |
| [`mise.mac.toml`](../../../../mise.mac.toml) | The Mac's: the plugin alone. |
| [`plugins/ws`](../../../../plugins/ws) | The plugin: a hook runs one `ws unit read` or `ws unit write` ([ws unit](../../../../engine/core/kind/README.md#ws-unit)). |
| `units.toml`, `mise.mac.toml` here | The role `frame`: the unit that keeps `/usr/local/bin/ws`. |

## Where it sits

- `/opt/ws` is the tree at the release, root's, which no one else may write. A plugin linked by an
  absolute path loads whatever its owner, so the tree's owner is the plugin's.
- `/usr/local/bin/mise` is the pinned mise, checked by its SHA-256.
- `/etc/mise/miserc.toml` says `env = ["mac"]`, which loads `mise.mac.toml` and the fragments.
- mise runs as the user and installs ws in the user's data directory. The Mac's `ws unit` refuses
  root.

## Root's paths

mise reads its own files of root's through sudo on every run, and asks before it writes them on a
terminal. So the Mac's config holds none: no `[bootstrap.files]` and no
`[bootstrap.directories]`. Nor does it hold `[dotfiles]`, which replace the user's own file or
link with no guard. ws's units hold both.

- A read needs no password. The Mac's `ws unit` reads root's paths as the user, and a run state's
  check runs as the user too. A file of root's is readable by all, or its spec fails to load.
- ws asks sudo for a change alone, as `ws client macos apply` does. A run that changes nothing
  asks no password. A change needs a terminal, or a sudo that asks none, as CI's runner has.

## A role's layout

A role keeps its directory, `roles/client/macos/ROLE/`, and its page there. Its spec,
`units.toml`, sits beside its templates and sources. Its one mise file is `mise.mac.toml`, which
holds its lines and mise's own parts, such as casks and tools. Its link,
`mise/conf.d/mac-ROLE`, leads to `../../roles/client/macos/ROLE`.

Each unit of a spec is a line `"ws:KIND/NAME" = "in-sync"` under `[bootstrap.packages]` in the
role's `mise.mac.toml`. The lines follow no server: a spec loops over `.Servers` within one unit,
as a unit a server would come and go with its line ([The facts](../facts/README.md)).
[`TestMacSpecsMatchMiseLines`](../cli/specs_test.go) fails on a line missing or extra.
`TestEachMacRoleIsLinked` fails on a role without its link, and `TestMacConfigHoldsNoRootPath` on
mise's own root path or a dotfile.

## ws at a fixed path

mise installs ws under the user's data directory, by version. A LaunchDaemon runs as root from a
path of root's alone. The unit `state/ws-binary` copies the ws that runs the units to
`/usr/local/bin/ws`, root:wheel and 0755, where the two differ, under a temporary name first, as
root. Its check, `cmp` and `stat`, runs as the user.

## The check against a silent skip

A line gone from a role's `mise.mac.toml` is a unit that no run sets up, with no error. The read
hook gets every line of mise's plan, and runs `ws unit read --whole`. Where the specs hold a unit
that the lines lack, and it has a change, each unit fails and names it, and so does every plan and
run. mise reads the lines again before it installs, without those in sync, so a lacking unit in
sync passes. mise calls the
hook for a line alone, so the frame's own line keeps it called. The server's
check, a file that runs `ws unit verify`, would be one of mise's root files here. Nor can a unit
run it: `verify` calls mise, whose read would call that unit again.
