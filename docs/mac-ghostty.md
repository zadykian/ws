# Ghostty and cld on the Mac

Before the tunnel ([The Mac](mac.md)), `mac/setup.sh` sets up Ghostty, the Mac's terminal: Ghostty
and the font its settings name, from Homebrew's casks, and the settings, `ws.ghostty`. Then it sets
up [cld](https://github.com/zadykian/cld) wherever it runs, for `cld restore tabs`. After Ghostty
quits or a window closes, a restore joins each pane's cld session again, and reopens what Ghostty
did not bring back
([Ghostty's tabs on a Mac](https://github.com/zadykian/cld/blob/main/docs/guide/ghostty-tabs.md)).
Where Ghostty or cld can't be installed or updated, as when Homebrew or GitHub can't be reached, the
script warns and goes on to the tunnel. So it does where Ghostty's config is not its to change, as
below, but it stops where it can't write a file. `--check` notes each install, and shows each file
as a diff.

## Ghostty

The script installs Ghostty from Homebrew's cask `ghostty`, into `/Applications`, where
`Ghostty.app` is in neither `/Applications` nor `~/Applications`. A Ghostty installed another way,
as from its disk image or the `ghostty@tip` cask, is left alone. Where Homebrew has the cask but the
app is gone, each run warns rather than install it again: `brew reinstall --cask ghostty` puts it
back. Without Ghostty's app, as after a failed install, the script writes none of Ghostty's files
and gives ssh none of cld's lines, but still sets up cld.

Ghostty then updates itself, as `ws.ghostty` says: it downloads each stable release and offers to
install it. The script runs no `brew upgrade`. brew skips a cask that updates itself, and the
cask's uninstall quits Ghostty, which may be the terminal the script runs in.

The script installs the font that `ws.ghostty` names, JetBrainsMono Nerd Font Mono, from the cask
`font-jetbrains-mono-nerd-font`. It does so where `JetBrainsMonoNerdFontMono-Regular.ttf` is in
neither `~/Library/Fonts`, where the cask puts it, nor `/Library/Fonts`, and warns as for the app
where Homebrew has the cask without the file. `ghostty +list-fonts` lists monospace families alone,
so it leaves out the cask's other variants. `font-family` takes a family, as `+list-fonts` heads it.
A face's name, such as `JetBrainsMono NFM Regular`, finds no font, and Ghostty falls back to its own
JetBrains Mono, which looks the same.

## Ghostty's settings

The settings are in [`mac/ws.ghostty`](../mac/ws.ghostty), which the script writes, under its
header, to `ws.ghostty`. That file goes beside the config file that Ghostty's menu opens:
`config.ghostty` or `config` in `~/Library/Application Support/com.mitchellh.ghostty`, then in
`~/.config/ghostty`. That is the first that holds anything, else the first that exists. Where none
exists, the script makes the first. It adds `config-file = ?ws.ghostty` at the end of that file,
once. Ghostty reads an included file after the file that names it, so each setting of `ws.ghostty`
wins over the config's own. A later `font-family` adds a fallback rather than replace the list, so
`ws.ghostty` empties it first. Change the settings in `mac/ws.ghostty`, and run the script again.

The script replaces `ws.ghostty` at each run. It warns and changes neither file where `ws.ghostty`
is one it did not write, or where it can't read the config. Where the config is a link, as a
dotfiles manager makes it, and lacks the line, the script warns: add the line to the file the link
points to. Where the config's directory is a link, the files go where it points, and the script says
so.

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

The rest of Ghostty's settings stay the user's. `macos-applescript` stays at its default, on, which
cld needs.

## cld

The script sets cld up wherever it runs, whatever the terminal. It:

- installs cld into `~/.local/bin` where cld is missing, with the `install.sh` of cld's latest
  release, which checks the binary's SHA-256. Each run then has `cld update` replace it where a
  newer release is out, and says so only then. A release before 0.6.0 lacks `cld update`, so
  `install.sh` replaces it. A cld built from a clone is left alone;
- adds the two lines that start cld's watcher to `Host ws ws-ssh` in `config.d/ws`, where Ghostty's
  app is there, once the cld installed has `restore tabs`.

A restore needs Ghostty 1.4 or newer, which cld checks, by Ghostty's AppleScript rather than its
version. The cask and the stable channel of `ws.ghostty` bring Ghostty's stable releases alone, so a
restore needs a stable release of 1.4 or newer.

`--check` can't run a cld it has not installed, so it notes that the two lines may come, rather than
show them.

`install.sh` edits no shell profile, so put `~/.local/bin` on the Mac's `PATH`, as in `~/.zprofile`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

### The watcher

A watcher keeps the record of Ghostty's windows, tabs and panes that a restore reads, and ends as
Ghostty quits. ssh starts it, through these lines of `Host ws ws-ssh`, as each connection opens:

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
the tunnel's address, `10.99.0.2`, after 5 of them, as [ssh's hosts](mac.md#sshs-hosts) says.
`ssh ws` then fails for 10 minutes, while `ssh ws-ssh` comes from another address and serves.

Two open issues meet this. [#56](https://github.com/zadykian/ws/issues/56) plans `mac/ws-join`,
which joins a session again after ssh drops: cld counts a pane that runs it as an ssh pane.
[#59](https://github.com/zadykian/ws/issues/59) plans `~/.config/ws/zshrc`, which is where a Mac
without this ssh config would start the watcher.
