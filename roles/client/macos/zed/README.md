# Zed on the Mac

`ws client macos apply` sets up [Zed](https://zed.dev) on the Mac, the editor that opens projects on
ws. The step is `zed`, so `ws client macos apply --only zed` runs it alone, and `--check` shows its
changes and makes none.

## The install

The step installs Zed from Homebrew's cask `zed` where `Zed.app` is in neither `/Applications` nor
`~/Applications`. It installs the font that the settings name, as [Ghostty's step](../ghostty/README.md)
does, from the cask `font-jetbrains-mono-nerd-font`.

- A failed install draws a warning, and the next run tries again, with brew's output. The step
  writes no file for Zed until the app is there.
- Where Homebrew has the cask but not the app, the step warns and installs nothing:
  `brew reinstall --cask zed` puts the app back.
- Without Homebrew, the step stops and says so.
- The step runs no `brew upgrade`. Zed updates itself, brew skips a cask that does, and the cask's
  uninstall quits the app.

The cask links `zed`. `zed --version` prints `Zed X.Y.Z – /Applications/Zed.app`, with an en dash,
and `zed ssh://ws/PATH` opens a project on ws.

## The two files

The step writes both files in `~/.config/zed`, a path that macOS gives no XDG lookup. It makes
`~/.config` first where missing.

- `global_settings.json` is [the file in `roles/client/macos/zed/files/`](files/global_settings.json), whole, with
  the header as its first line. Each run replaces it, so changes made there are lost. A file whose
  first line is not the header is left, with a warning: move it away, and the next run writes
  ws's. Zed parses it as it parses `settings.json`, so the header comment is valid.
- `settings.json` is Zed's own, which its UI writes. The step writes a seed there once, where
  nothing is, and never reads the file after. A file of any content, or a link, counts as there.

Zed merges `settings.json` over `global_settings.json`, so a key in `settings.json` wins over ws's.
This is the reverse of Ghostty's order. A key set only in `global_settings.json` shows Zed's
default in the Settings Editor.

The file sets a ruler at column 120, shows all whitespace, and saves 3 seconds after an edit, from
`roles/client/macos/zed/files/global_settings.json`. CI parses the file but sees no Zed obey it: check
the three once on a remote project.

The seed holds one connection, `{ "host": "ws" }`. Zed records an opened project by its index in the
list of `ssh_connections` in `settings.json`, and arrays replace rather than merge, so the
connection can sit in no other file. Zed's reader of `~/.ssh/config` has no `Include`, so it
does not list `ws`, which lives in `~/.ssh/config.d/ws`. Zed connects with the system `ssh`, so
`host: "ws"` resolves through that file, with its user, key and keepalive. Nothing else is set
there: no `username`, `port` or `args`.

A linked `~/.config/zed`, as GNU stow's folding makes, draws a warning where the step plans a write:
Zed's files go through the link.

## A settings.json that exists already

Where Zed wrote `settings.json` before the step ran, the step leaves it and does not warn: the
file is there on every later run, and the step never reads it. Zed's first write starts from its
template, whose `theme` (One Light and One Dark, mode `system`) would win over ws's. So:

- delete the `theme` block of `settings.json`, or the theme stays Zed's;
- add `ws` in Remote Projects by hand, as the seed would have.

## Extensions

`auto_install_extensions` in ws's settings names the extensions that Zed installs at its next start,
from its own registry. They update unpinned, outside the SHA-256 rule of the downloads. The key
merges with Zed's default one by one. `docker-compose` is false: its latest release attaches a Compose
server to every YAML file, Ansible's too.

## Other facts

- `ws` through the tunnel needs no Local Network grant. Only a direct way does ([The Mac](../README.md)),
  so the seed names `ws` alone.
- The terminal docks right, as the project, git and outline panels dock left. A resize of a terminal
  docked at a side duplicates its lines (zed#65316, open).
- `terminal.shell` stays unset, as Zed runs it on ws too (zed#65548). `option_as_meta` gives the
  Alt keys in Zed's terminal, at the cost of Option's special characters. `detect_venv` is `off`:
  with it on, a venv in the directory has Zed type its activation line into every new tab, which
  reaches a reattached tmux session and whatever runs there.
- `terminal.toolbar.breadcrumbs` shows a title that a program in the terminal sets, such as
  [lazygit's](../../../server/ubuntu/lazygit/README.md#the-title), above it. Zed's tab takes none.
- `terminal.line_height` is Zed's default. Nerd glyphs may look misaligned all the same (zed#4746,
  open), and no setting helps.
- [Zed's page for the server](../../../server/ubuntu/zed/README.md#when-the-connection-drops) says what to do when the
  connection drops.

## The keymap

The step also writes `~/.config/zed/keymap.json`, which is
[`roles/client/macos/zed/files/keymap.json`](files/keymap.json) whole. That plain file is the source of the keys,
and a `//` comment beside a binding gives its reason. `base_keymap` stays `Zed`.
The file carries its own header, and the step takes the start of line 1,
`// Written by github.com/zadykian/ws`, as ws's mark.

- Where the file is missing, or its first line starts with the mark, the step writes it whole.
- Where the first line does not, the step leaves the file and warns once: move it away, and the
  next run writes ws's.
- Zed's keymap editor writes this file too (zed#8829). A change made there is replaced at the next
  run while line 1 stands. Delete line 1 to keep the file, and ws leaves it.

The block at the file's end lists the keys left out, each with its reason. The rest of this
section holds what the file's comments do not.

### How the keys translate

Zed on macOS, with a US layout:

- Shift with a sign or digit is written as the shifted character without Shift, such as `alt-{`
  and `ctrl-?`. macOS reports it so, and `shift-alt-[` never matches. Shift with a letter stays.
- Zed does not tell numpad keys apart ([zed#11948](https://github.com/zed-industries/zed/issues/11948)).
- Alt with a letter matches by key, so an unbound Alt key still types its character.

### How Zed picks a binding

- The deeper context wins, then the later binding. User bindings load after Zed's.
- A binding with no context counts as the deepest, so the file's global section beats every default.
- A chord whose first key also has a binding waits 1 s, then runs that binding. A chord defined
  before a binding of its first key is ignored (zed#30211). So the file unbinds Zed's Ctrl+K,
  Ctrl+M and Ctrl+R.
- `unbind` works only where the section's context matches, so Zed's Ctrl+R in the Diagnostics
  view still waits.

### Two keys of the file's own

Zed has no default key to restart a language server or to reload Zed. They are the remedies after
a reconnect ([When the connection drops](../../../server/ubuntu/zed/README.md#when-the-connection-drops)). Both keys are free
in Zed's defaults:

- Ctrl+Shift+R is `editor::RestartLanguageServer`, in `Editor && mode == full`.
- Ctrl+Alt+Shift+Y is `workspace::Reload`, in the global section.

Shift keeps the terminal's rule below, so Ctrl+Alt+Shift+Y reaches Zed from the terminal too.

### The terminal's rule

In the terminal, a Ctrl or Alt key with a letter or sign, without Shift, reaches the shell
(zed#26018). The section unbinds 21 global keys, and the chords' first keys go straight
(zed#28423). F-keys, named keys and digits stay Zed's. Ctrl+C, Ctrl+D and Ctrl+R stay the shell's.
Ctrl+B is in the editor only, so the terminal gets it.

### Decisions in the file

- **Agent:** F7 and Alt+7 open the agent panel (`agent::ToggleFocus`). Ctrl+/ is the inline
  assistant (`assistant::InlineAssist`) in the full editor.
- **Cmd+Y** is the agent's Allow once and Keep, and `git::StageAndNext`.
- **Edit predictions:** Tab, Alt+Tab and Ctrl+Cmd+Right and Down stay Zed's.
- Ctrl+Cmd+S is Save all, not the agent's steer of a queued message.
- F7 replaces Zed's F7, step over while stopped, and F10 steps over too.
- The global keys win in the agent panel. So its Ctrl+Home and End and Ctrl+Alt+Up, Down, PgUp and
  PgDn are the pane, dock and tab keys there.
- **Keys whose action Zed lacks keep Zed's:** F3 tests, F4 NuGet, F10 build and Alt+Delete safe
  delete.
- Ctrl+N in the full editor is New file, so it no longer moves in the completion menu. The arrows
  do, and pickers keep Ctrl+N and Ctrl+P.
- **Code actions:** the keys for refactor, change signature, extract method, add methods and
  generate all open the full menu, as no binding picks a kind (zed#14470).
- Last edit location stays in the file (zed#19731). Next error can skip gopls's unused-variable
  errors (zed#64790).

### The keys left to Zed

The file binds no plain key: the arrows, Enter, Tab, Home, End and Page Up and Down, nor the
completion and snippet keys. Alt+click adds a cursor, and Shift+Alt+drag selects a column
(`multi_cursor_modifier`). Ctrl+D in the terminal reaches the shell.

### Keys a Mac keyboard lacks

A Mac keyboard has no Insert, Pause, Break or menu key, so the file leaves out Alt+Insert,
Shift+Insert, the menu key, Ctrl+Pause and Ctrl+Alt+Break.

F-keys need Fn, or the setting "Use F1, F2 keys as standard function keys" in System
Settings. The name of that setting is unchecked: it comes from Linux. `defaults write -g
com.apple.keyboard.fnState -bool true` should do the same, and is unchecked too.

### macOS's own shortcuts

Apple lists its shortcuts at <https://support.apple.com/en-us/102650>. Spaces' keys are in System
Settings, under the Mission Control shortcuts, not on that page.

| Key | macOS | The file's action |
| --- | --- | --- |
| Ctrl+Space | previous input source | Completion (Zed's own default too) |
| Shift+Ctrl+Space | input source, by Zed's comment | Parameter info |
| Ctrl+Alt+Space | next input source | Type-matching completion |
| Ctrl+Up, Ctrl+Down | Mission Control, app windows | Scroll line, overloads |
| Ctrl+Left, Ctrl+Right | move a space (on by default; unchecked) | Word left, right |
| Ctrl+F3, Ctrl+F4, Ctrl+F5 | focus Dock, window, toolbar | Find word, Close tab, Debug |

They go off by hand, in System Settings.

`defaults write com.apple.symbolichotkeys` can do it. The ids are from memory and unchecked:

- 60 and 61 are the input sources.
- 32 and 33 are Mission Control.
- 79 and 81 are Spaces.
- 8, 9 and 10 are the focus keys.

A change needs a re-login, or `activateSettings -u`. That is why ws does not do it.

## Changing the settings

Edit the two files in `roles/client/macos/zed/files/` by hand. They are plain files, and
`roles/client/macos/zed/files_test.go` checks that each is valid JSON under its comment lines.
