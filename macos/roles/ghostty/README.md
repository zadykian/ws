# Ghostty and cld on the Mac

`mise bootstrap` sets up Ghostty, the Mac's terminal: Ghostty and the font its settings name, as
mise's casks, and the settings, `ws.ghostty`, and [cld](https://github.com/zadykian/cld), for `cld
restore tabs` ([cld on the Mac](../cld/README.md)). After Ghostty
quits or a window closes, a restore joins each pane's cld session again, and reopens what Ghostty
did not bring back ([Ghostty's tabs on a
Mac](https://github.com/zadykian/cld/blob/main/docs/guide/ghostty-tabs.md)). Where cld can't be
installed or updated, as when GitHub can't be reached, cld's unit warns and fails, and the next
bootstrap tries again.

## Ghostty

[`mise.macos.toml`](mise.macos.toml) installs Ghostty from the cask `ghostty` by mise's `brew-cask`
backend, which needs no Homebrew. Where `/opt/homebrew` is missing, mise makes it by sudo once. A
cask takes the latest version alone, outside the SHA-256 rule of the downloads. Ghostty then
updates itself, as `ws.ghostty` says: it downloads each stable release and offers to install it.

The same file installs the font that `ws.ghostty` names, and [Zed](../zed/README.md) too,
JetBrainsMono Nerd Font Mono, from the cask `font-jetbrains-mono-nerd-font`. `ghostty
+list-fonts` lists monospace families alone, so it leaves out the cask's other variants.
`font-family` takes a family, as `+list-fonts` heads it. A face's name, such as `JetBrainsMono NFM
Regular`, finds no font, and Ghostty falls back to its own JetBrains Mono, which looks the same.

## Ghostty's settings

The settings are in [`macos/roles/ghostty/files/ws.ghostty`](files/ws.ghostty), which the
unit `file.ghostty-conf` writes whole to `/etc/ws/ghostty`, as root, at each run. The file starts
with ws's mark, and a `/etc/ws/ghostty` without it stops the unit, as a file by hand.

The config file that Ghostty's menu opens is `config.ghostty` or `config` in
`~/Library/Application Support/com.mitchellh.ghostty`, then in `~/.config/ghostty`. That is the
first that holds anything, else the first that exists, else the first of the four. The facts
read it as `.Ghostty.Config` ([The facts](../../facts/README.md)). The unit `file.ghostty-line` adds
`config-file = ?/etc/ws/ghostty` at the end of that file, once, and keeps the old file as
`PATH.TIME~`. A line in the file that equals it, spaces aside, counts: a path quoted, relative
or with `~` does not. Ghostty reads an included file after the file that names it, so each
setting of `/etc/ws/ghostty` wins over the config's own. A later `font-family` adds a fallback
rather than replace the list, so `/etc/ws/ghostty` empties it first. Change the settings in
`macos/roles/ghostty/files/ws.ghostty`, and run `mise bootstrap` again.

Where the config is a link, as a dotfiles manager makes it, the unit reads through it. Where that
file lacks the line, the unit stops: add the line to it, or make the config a plain file. A link
on the way, such as a linked `~/.config`, stops the unit too: ws writes no file of the user's
through one.

Some settings change more than the look:

- `window-save-state = always` has Ghostty bring its windows back after every quit, which a restore
  builds on.
- `shell-integration-features` replaces the config's whole list, as a later line of it does. Its
  `ssh-terminfo` installs Ghostty's terminfo on ws at the first ssh from Ghostty, so ws installs
  none itself.
- `clipboard-read` and `title-report` stay at Ghostty's defaults, `ask` and off. With the first at
  `allow`, a program in a pane, one on ws included, reads the Mac's clipboard without asking, such
  as through `tmux refresh-client -l` and `tmux save-buffer -`. With the second on, Ghostty writes
  the window's title to the pane's input. Text that a pane prints can then sit at the prompt, for
  one Enter to run. cld needs neither.
- `macos-option-as-alt = left` makes left Option act as Alt, for the Alt keys in
  Fresh. Right Option keeps composing characters.

The rest of Ghostty's settings stay the user's. `macos-applescript` stays at its default, on, which
cld needs.

## cld

The bootstrap sets cld up wherever it runs, whatever the terminal. It:

- installs cld into `~/.local/bin` where cld is missing, with the `install.sh` of cld's latest
  release, which checks the binary's SHA-256. Each run then has `cld update` replace it where a
  newer release is out, and says so only then. A release before 0.6.0 lacks `cld update`, so
  `install.sh` replaces it. A cld built from a clone is left alone;
- adds the two lines that start cld's watcher to each server's `Host` in `/etc/ws/ssh_config`, where
  Ghostty's app is there, and the cld installed has `restore tabs` or a release's version. Where
  cld is missing, the lines come ahead, as the release that its unit installs has `restore tabs`.

A restore needs Ghostty 1.4 or newer, which cld checks, by Ghostty's AppleScript rather than its
version. The cask and the stable channel of `ws.ghostty` bring Ghostty's stable releases alone, so a
restore needs a stable release of 1.4 or newer.

The facts read Ghostty's app as the bootstrap starts, so a first bootstrap that installs the cask
adds the lines at the next.

`install.sh` edits no shell profile, so put `~/.local/bin` on the Mac's `PATH`, as in `~/.zprofile`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

### The watcher

A watcher keeps the record of Ghostty's windows, tabs and panes that a restore reads, and ends as
Ghostty quits. ssh starts it, through these lines of each server's `Host`, as each connection opens:

```text
PermitLocalCommand yes
LocalCommand [ "$TERM_PROGRAM" != ghostty ] || [ ! -x %d/.local/bin/cld ] || %d/.local/bin/cld restore tabs --watch
```

ssh runs `LocalCommand` through the user's shell once it has connected, with `%d` as the Mac's
home (ssh_config(5)). In a Ghostty pane, the line starts the watcher, unless one runs. Elsewhere, as
in tmux on the Mac or another terminal, or without cld, it starts nothing and exits 0. scp and sftp
turn it off. The watcher starts inside Ghostty's pane, so macOS charges its Apple Events to
Ghostty, which controls itself with no prompt. A `git push` to ws from a Ghostty pane starts the
watcher too, or finds it running.

A restore opens an ssh connection for each pane it reopens. Where those logins fail, fail2ban bans
the tunnel's address, `10.99.N.2`, which shuts `ssh ws` out for an hour. [ssh's
hosts](../../README.md#sshs-hosts) says how to lift the ban.

## The zsh hook

A dropped ssh leaves Ghostty in the modes that Fresh, tmux and cld's sessions turned on, as their
resets never reach the Mac. Each mouse move then types `^[[<35;42;17M` at the prompt, and some Ctrl
keys arrive as escape codes. `mise bootstrap` writes
[`macos/roles/zsh/files/zshrc`](../zsh/files/zshrc) as `/etc/ws/zshrc`, by
[zsh's units](../zsh/units.toml.tmpl). The file holds [zsh's settings](../zsh/README.md#the-settings)
above the hook. mise adds `source /etc/ws/zshrc` at the end of `~/.zshrc`, keeps no copy of the old
file, and stops where `~/.zshrc` is a link ([The line in ~/.zshrc](../zsh/README.md#the-line-in-zshrc)).

Before each prompt, the hook turns off the reports of the mouse, focus and the colour scheme, and
modifyOtherKeys. It resets kitty's keyboard flags, the cursor keys, the keypad, and the cursor's
style and visibility. It leaves the alternate screen only after `ssh`, or `cld` with `--host`,
which becomes ssh, exits 255, as a drop makes it. ESC 7 comes just before that reset, which then
leaves the main screen's cursor in place. The rest goes before the reset too, as kitty's flags are
each screen's own. Without the hook, Ghostty's menu item Reset Terminal turns the modes off.
