# cld and Ghostty

`setup.sh` sets up [cld](https://github.com/zadykian/cld) wherever it runs, for `cld restore tabs`.
After Ghostty quits or a window closes, a restore joins each pane's cld session again, and reopens
what Ghostty did not bring back
([Ghostty's tabs on a Mac](https://github.com/zadykian/cld/blob/main/docs/guide/ghostty-tabs.md)).
A restore needs Ghostty 1.4 or newer, which cld checks, by Ghostty's AppleScript rather than its
version. Ghostty's part needs its app: `Ghostty.app` in `/Applications` or `~/Applications`, which
the script installs from Homebrew's cask `ghostty` where missing. Where that fails, it warns and
writes none of that part. Before the tunnel, it:

- installs cld into `~/.local/bin` where cld is missing, with the `install.sh` of cld's latest
  release, which checks the binary's SHA-256. Each run then has `cld update` replace it where a
  newer release is out, and says so only then. A release before 0.6.0 lacks `cld update`, so
  `install.sh` replaces it. A cld built from a clone is left alone;
- writes `cld.ghostty` beside Ghostty's config file, and adds `config-file = ?cld.ghostty` at the
  end of that file, once, where Ghostty's app is there;
- adds the two lines that start cld's watcher to `Host ws ws-ssh` in `config.d/ws`, where Ghostty's
  app is there, once the cld installed has `restore tabs`.

Where cld can't be installed or updated, as when GitHub can't be reached, the script warns and goes
on to the tunnel. `--check` can't run a cld it has not installed, so it notes that the two lines
may come, rather than show them.

`install.sh` edits no shell profile, so put `~/.local/bin` on the Mac's `PATH`, as in `~/.zprofile`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

## Ghostty's settings

The script writes beside the config file that Ghostty's menu opens: `config.ghostty` or `config`
in `~/Library/Application Support/com.mitchellh.ghostty`, then in `~/.config/ghostty`. That is the
first that holds anything, else the first that exists. Where none exists, the script makes the
first. `cld.ghostty` holds `window-save-state = always`, so that Ghostty brings its windows back
after every quit, which a restore builds on. Ghostty reads an included file after the file that
names it, so this setting wins over the config's own.

The script replaces `cld.ghostty` at each run. It warns and changes neither file where
`cld.ghostty` is one it did not write, or where it can't read the config. Where the config is a
link, as a dotfiles manager makes it, and lacks the line, the script warns: add the line to the file
the link points to. Where the config's directory is a link, the files go where it points, and the
script says so.

The rest of Ghostty's settings stay the user's. `macos-applescript` stays at its default, on, which
cld needs. Put `ssh-terminfo` in your own `shell-integration-features`, as cld's guide asks. It
installs Ghostty's terminfo on ws at the first ssh from Ghostty, so ws installs none itself. ws sets
no `shell-integration-features`, since a later line of it replaces the whole list, yours included.

## The watcher

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
