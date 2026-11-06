# fresh

The `fresh` role is Go steps in `internal/server/fresh`. It installs the Fresh editor on ws from
the `.deb` of a pinned release. Fresh runs in a terminal, and its settings come from the neutral
data in `internal/editor`, which Zed's role also reads.

## The install

- `Fresh download` fetches `fresh-editor_VERSION_ARCH.deb` from the project's GitHub release into
  `/var/cache/`. It checks the file against the SHA-256 in `download.go` before dpkg reads it.
- `Fresh install` runs dpkg on the file, then removes the file.
- Where `dpkg-query` lists the pinned version as installed, the download plans nothing and the
  install only removes a leftover file. A second run changes nothing.
- The pin holds the version, the tag and one sum for amd64 and one for arm64. Any other
  architecture fails the step.

The release's `.deb` needs `libc6` 2.30 and `libgcc-s1` 4.2 alone. So it installs on a physical
machine on the LAN, on a cloud VM and on a VM on the Mac. The testing `.deb` lacks the grammars
for JS, TS and Templ, and needs `libc6` 2.39.

The role needs no other role and no secret. No step needs a booted systemd, so it runs whole in
CI's container.

## The config

`Fresh config` writes `~/.config/fresh/config.json` for `ws_user`, from `internal/editor/fresh`. A
run replaces a change made to it by hand and keeps the old file beside the new one, as
[zed](zed.md) does for its settings.

- Updates, telemetry and the session daemon are off: `check_for_updates`, `self_update` and
  `auto_attach` are false. Fresh reads a missing key as its default, so the render writes each.
- The ruler, visible whitespace and auto-save come from `internal/editor/settings.yaml`. Zed's
  are in `mac/zed/global_settings.json`.
- The file starts with a comment that says ws wrote it. Fresh reads JSONC.
- `~/.config/fresh` stays the user's, as Fresh writes `tsconfig.json` and `types/` there.
- The step fails where `~/.config`, the directory or `config.json` is a link. The user owns
  `~/.config`, and a link would send ws's root to whatever it names.

## The servers

`lsp` in `config.json` names the language servers of `internal/editor/servers.yaml`, each by the
path that `devtools.ServerPaths` gives it for the user's home.

- Fresh takes a list of servers for a language, so dockerfile has two.
- Each entry starts at once (`auto_start`), with the catalogue's arguments and options.
- Fresh has no docker-compose language. ws adds one on the yaml grammar for `compose.yaml`,
  `compose.yml`, `docker-compose.yaml`, `docker-compose.yml` and the catalogue's globs. The server
  sees a compose file by Fresh's key, the extension, mapped to `dockercompose`.
- Ansible has no entry: no pattern tells a playbook from YAML, and `.yml` playbooks stay YAML.
- Fresh's csharp plugin answers Roslyn's restore request for csharp-ls alone. Roslyn starts and
  initialises, but project load and navigation are unverified.

The role needs no server installed to succeed, as it only names paths. The `devtools` role
installs them, so a user wanting both passes `--only devtools,fresh`.

## Trust each project

Fresh opens a folder with a project marker as Restricted, with every server off. Press T, then
Enter, once per repository. Fresh stores the answer in
`~/.local/share/fresh/workspaces/PATH/trust.json`, and a linked worktree inherits it. No key sets
it, so ws cannot. A folder with no marker is trusted at once.

## A new version

1. Change `version`, `tag` and both sums in `download.go`. Check the `.deb`'s `Depends`.
2. From the tag's source, take every name of `define_action_str_mapping!` in
   `crates/fresh-editor/src/input/keybindings.rs`, and the handler names of `registerCommand` at
   the top level of `plugins/`, leaving out `plugins/examples`.
3. Write them to `internal/editor/fresh/testdata/actions.txt`, under its first line with the new
   version. `TestTheActionListNamesThePinnedVersion` fails until then, and
   `TestEveryTargetIsAnActionOfFresh` names any target that is gone.
4. Leave out `unbind`, a config word, and the plugin commands made at run time: 65 of those in
   0.5.7.

A name with no action loads with no error, so the list is the only check of a target.

## The theme

`Fresh config` also writes `~/.config/fresh/themes/jetbrains-dark.json`, and `config.json` names it
by its `name`, `JetBrains Dark`, which comes from `internal/editor/settings.yaml`. The values are
Zed's, from `artemevsevev/zed-theme-jetbrains` at `4385ff2c112e693716badf22e8fb348bede26f93`,
Apache-2.0, by Artem Evsevev. Only the dark theme is used.

Rules a change keeps:

- The file is strict JSON with no header. A comment makes Fresh fall back to its default theme.
- Colours are `[r, g, b]` alone. A hex string renders white with no error.
- No alpha. Two search colours are blended on #1e1f22.

Limits:

- Eleven syntax colours serve fourteen categories, with `syntax.constant` at #2aacb8.
- Property, attribute, doc comment, tag, `type.builtin` and the active line number have no place.
- `rainbow_brackets` is off, as the theme has no place for those colours. `after_eof_bg` is named,
  and the ruler is #393b40.

## The wrapper

`Fresh wrapper` writes `/usr/local/bin/fresh`, mode 0755, from `files/fresh`. It comes before the
`.deb`'s `/usr/bin/fresh` on every `PATH` of Ubuntu's.

Fresh asks for its keys by the kitty protocol, which tmux ignores. With no fix, 0 of 29 Shift+Ctrl
keys arrive through tmux, and 21 mapped rows land on another key: ctrl+shift+z undoes, for one.
Both paths pass `--config "$HOME/.config/fresh/config.json"` ahead of the arguments where that file
exists. Fresh then reads the user's config alone. Without it, a project's `.fresh/config.json` merges
over it, or a legacy `config.json` in the working directory. A cloned repository could name a
language server's command. A `config.json` that does not parse drops the user's config for
Fresh's defaults, with updates and telemetry on. The cost is that a project's own config no longer
applies, and `/usr/bin/fresh` run directly skips the wrapper and this protection.

The wrapper takes one of two paths:

- With `$TMUX` empty, or stdin or stdout not a terminal, it runs `exec /usr/bin/fresh "$@"`.
- Else it runs `tmux set -s extended-keys on` where the option reads `off`, silencing errors. It
  sends `CSI > 4;2m`, runs Fresh, sends `CSI > 4m`, and exits with Fresh's status.

Rules a change keeps:

- The second path never `exec`s, or the reset is never sent.
- A server at `on` or `always` stays so. From `off`, `on` stays after Fresh exits, and new
  panes get it.
- A killed Fresh runs the reset, as the wrapper outlives it. The pane keeps asking only if the
  wrapper dies too.

## The keys

[Fresh's F12 keymap](keymap/fresh.md) holds the adapter's counts, rules and limits. What stays
unproven until the maintainer's trial on the Mac:

- Real Ghostty, with left Option as Alt: its encoder was modelled on a US layout.
- macOS and the ssh leg between Ghostty and tmux, and tmux 3.7c and Ubuntu's own build, as 3.6a
  is what ran.
- The arm64 binary: its hash matched, and it never ran.
- gopls with Fresh, and Roslyn past initialise: diagnostics and navigation.
- Fresh's terminal keys, known by source alone: the 24 `noop` keys reach the shell, and
  ctrl+space, ctrl+], F9 and alt+o leave the terminal. Also `self_update` at run time.
- The wrapper when it dies with Fresh: the pane keeps asking.
- Italic modifiers in the theme.
- Press T per repository, and confirm that a worktree inherits it.
