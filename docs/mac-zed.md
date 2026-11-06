# Zed on the Mac

`ws mac apply` sets up [Zed](https://zed.dev) on the Mac, the editor that opens projects on ws. The
step is `zed`, so `ws mac apply zed` runs it alone, and `--check` shows its changes and makes none.

## The install

The step installs Zed from Homebrew's cask `zed` where `Zed.app` is in neither `/Applications` nor
`~/Applications`. It installs the font that the settings name, as [Ghostty's step](mac-ghostty.md)
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

- `global_settings.json` is ws's settings, [`mac/zed.json`](../mac/zed.json), under one header
  line. Each run replaces it, so changes made there are lost. A file whose first line is not the
  header is left, with a warning: move it away, and the next run writes ws's. Zed parses it as it
  parses `settings.json`, so the header comment is valid.
- `settings.json` is Zed's own, which its UI writes. The step writes a seed there once, where
  nothing is, and never reads the file after. A file of any content, or a link, counts as there.

Zed merges `settings.json` over `global_settings.json`, so a key in `settings.json` wins over ws's.
This is the reverse of Ghostty's order. A key set only in `global_settings.json` shows Zed's
default in the Settings Editor.

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

- `ws` through the tunnel needs no Local Network grant. Only a direct way does ([The Mac](mac.md)),
  so the seed names `ws` alone.
- The terminal docks right, as the project, git and outline panels dock left. A resize of a terminal
  docked at a side duplicates its lines (zed#65316, open).
- `terminal.shell` stays unset, as Zed runs it on ws too (zed#65548). `option_as_meta` gives helix's
  Alt keys in Zed's terminal, at the cost of Option's special characters. `detect_venv` is `off`:
  with it on, a venv in the directory has Zed type its activation line into every new tab, which
  reaches a reattached tmux session and whatever runs there.
- `terminal.line_height` is Zed's default. Nerd glyphs may look misaligned all the same (zed#4746,
  open), and no setting helps.
- Zed's keymap overlay is [zed-keymap.md](zed-keymap.md), and `docs/zed.md`, its page for the server, says
  what to do when the connection drops.

## The keymap

The step also writes `~/.config/zed/keymap.json`, rendered from [`internal/editor`](../internal/editor/),
F12's keys for Zed. `base_keymap` stays `Zed`. The file carries its own header, so the step adds
none, and takes the start of line 1, `// Written by github.com/zadykian/ws`, as ws's mark.

- Where the file is missing, or its first line starts with the mark, the step writes it whole.
- Where the first line does not, the step leaves the file and warns once: move it away, and the
  next run writes ws's.
- Zed's keymap editor writes this file too (zed#8829). A change made there is replaced at the next
  run while line 1 stands. Delete line 1 to keep the file, and ws leaves it.

[zed-keymap.md](zed-keymap.md) lists every F12 action with its Zed action, or why it has none.

## Changing the settings

Edit [`mac/zed.json`](../mac/zed.json), build `ws` again and run it. `ws` embeds it. For the
keymap, edit the data or the adapter and the golden file: `go test` holds them equal. `ws` renders
the keymap.
