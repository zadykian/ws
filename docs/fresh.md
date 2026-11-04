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
- The ruler, visible whitespace and auto-save come from `internal/editor/settings.yaml`, which
  Zed reads too.
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
