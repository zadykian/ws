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
