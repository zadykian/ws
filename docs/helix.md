# helix

The `helix` role installs the helix editor, `hx`, from Ubuntu's archive. It writes
`~/.config/helix/config.toml`, set close to GoLand's and Rider's settings, and `languages.toml`.
Ubuntu's hx ships no tree-sitter grammars, so the role fetches and builds those `helix_grammars`
lists with `hx --grammar`. Each run replaces changes made to the two files by hand, and keeps the
old file beside the new one. The keys follow the IDEs' F12 keymap: [helix's keys](#helixs-keys),
below.

Go's language server, gopls, comes from `go install` at the version `group_vars/all.yml` pins, with
the Go of the `devtools` role, and is built again when either version changes. C#'s,
roslyn-language-server, is the server of VS Code's C# extension. It comes from nuget.org with
`dotnet tool` at the version pinned there too, and runs on the .NET 10 of the `devtools` role:
[C# in helix](#c-in-helix), below.

## helix's keys

`config.toml` binds the keys of F12, the keymap GoLand and Rider use here, in normal and in insert
mode. It binds them for each F12 action helix has a command for, and names F12's action beside
each. [The F12 keymap](helix-keymap.md) lists every F12 action with its keys and helix command,
and helix's own key for it, then the actions helix has no command for. Cmd never reaches a program
in a terminal, so F12's Ctrl keys stand in for it. On a Mac, the Alt keys need Option to act as
Meta or send Esc+.

Where F12 takes a key of helix's, helix's other key for it still works. A few have no other key,
which [helix's keys that F12 takes](helix-keymap/taken.md) lists. Alt+D was helix's only key to
delete without yanking: Delete now does the same.

From normal and insert mode, Move line goes through register `m`, so the `"` register keeps the
last yank, and a move is one undo step. A move off or onto a last line that has no final newline
joins two lines, as `p` and `P` do there: `u` undoes it, and a `:w` adds the newline.

A terminal that sends keys as older terminals did loses some. Shift+Ctrl with a letter arrives as
Ctrl alone, so that Shift+Ctrl+W extends the selection and Shift+Ctrl+Z undoes. Ctrl+- and Ctrl+=
don't arrive, and Ctrl+I, Ctrl+M and Ctrl+[ arrive as Tab, Enter and Esc. One that reports
modified keys as `CSI u`, or answers helix's request for the kitty keyboard protocol, sends every
one of these. Shift+Ctrl with a letter arrives whole only where the terminal sends the shifted
letter, as `CSI 84;6u` for Shift+Ctrl+T. It arrives whole too where the terminal answers the kitty
request and reports the shifted key, as `CSI 116:84;6u`. Where it sends the letter alone,
`CSI 116;6u`, helix drops the Shift and runs Ctrl+T.

tmux 3.6a sends Shift+Ctrl with a letter as Ctrl alone whatever the terminal: it sends such keys
whole only to a program that asks, and helix doesn't. Ctrl+- and Ctrl+= reach helix through tmux
with these server options. Neither tmux's defaults nor cld's servers have them; for a tmux of your
own, they go in `~/.tmux.conf`:

```text
set -s extended-keys always
set -s extended-keys-format csi-u
set -as terminal-features 'xterm*:extkeys'
```

The last tells tmux that a terminal whose `TERM` starts with `xterm` reports modified keys; tmux
knows it of iTerm2 already. With `always`, tmux sends those keys as `CSI u` to every program, a
shell too. tmux's prefix reaches helix pressed twice: Ctrl+B, Toggle line breakpoint, by default,
and Ctrl+Q, Go to action, in cld's sessions.

Where an F12 key doesn't arrive, helix's own key, in [the keymap](helix-keymap.md), does the same.

## C# in helix

hx gives roslyn-language-server the root of the git repository hx started in. There the server
loads the one `.sln` or `.slnx`, or else every project below the root; `dotnet.defaultSolution` in
the root's `.vscode/settings.json` picks one of several solutions. Started outside a git
repository, hx gives the server no root, and the server takes each file as a program of its own.

Ubuntu's hx asks the server for a file's errors when it opens the file and when the file changes,
not when the server has loaded the projects. A file opened while they load shows its errors after
its first edit, even one undone with `u`.
