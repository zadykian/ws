# zsh on the Mac

The `zsh` role sets up the Mac's shell: oh-my-zsh and two plugins, as mise's clones, and ws's
settings, which `~/.zshrc` sources. The hook that resets the terminal's modes has
[its own page](../ghostty/README.md#the-zsh-hook).

## The clones

[`mise.macos.toml`](mise.macos.toml) names three repositories in `[bootstrap.repos]`, each at the
full commit SHA of its default branch's head as of 2026-11-24, with no `depth`:

- `~/.oh-my-zsh`, from `ohmyzsh/ohmyzsh`;
- `~/.oh-my-zsh/custom/plugins/zsh-autosuggestions`, from `zsh-users/zsh-autosuggestions`;
- `~/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting`, from `zsh-users/zsh-syntax-highlighting`.

A clone is checked by its commit, not by a SHA-256 of a download. To move a pin, take a newer head
by `git ls-remote URL HEAD`, and run `mise bootstrap` again: mise fetches the new commit and checks
it out. A test keeps each pin a full commit with no depth
([specsrepos_test.go](../../cli/specsrepos_test.go)).

mise clones in the file's order and refuses a directory with files in it, so oh-my-zsh comes first.
oh-my-zsh's `.gitignore` leaves its `custom/` directory out, so the plugins never make it dirty,
which mise would refuse.

## The settings

[`files/zshrc`](files/zshrc) is `/etc/ws/zshrc`, which a line of `~/.zshrc` sources
([The line](#the-line-in-zshrc)). Above the hook it sets, in this order:

- oh-my-zsh, where `~/.oh-my-zsh/oh-my-zsh.sh` exists, with the plugins `git`, `zsh-autosuggestions`
  and `zsh-syntax-highlighting`. The last must come last, as its install notes say. A Mac without
  the clone skips the block. `zstyle ':omz:update' mode disabled` turns oh-my-zsh's own update
  off, as it would move the checkout off its pin. The theme is oh-my-zsh's default, `robbyrussell`.
- `~/.local/bin` first on `PATH`, once in a nest of shells, and `GPG_TTY`, which gpg-agent's
  pinentry reads.
- `genpass`, an alias of `pwgen -s 24 1`: [devtools](../devtools/README.md) installs pwgen.
- `SSH_AUTH_SOCK` as seward's agent, `~/.ssh/seward.sock`, where the socket exists. Until the agent
  runs there is none, and the shell keeps the agent that macOS starts. ssh and git reach seward's
  agent by their own settings, so the variable serves other programs.

The unit `file/zsh-rc` writes the file whole as root, and an interactive `zsh` reads it as the
user. oh-my-zsh's hooks run before the hook's, so the file moves the hook's `precmd` to the front,
where `$?` still is the command's exit status.

## The line in ~/.zshrc

A `[dotfiles]` line edit in [`mise.macos.toml`](mise.macos.toml) adds `source /etc/ws/zshrc` at the
end of `~/.zshrc`, so that ws's settings and the hook come after the user's. mise makes a missing
`~/.zshrc` with the line alone. Once the line is there, mise leaves it where it stands, and a line
the user adds below it comes after ws's settings.

- mise matches the line exactly. The same line with a space more or less is another, and mise adds
  its own below it.
- mise refuses to edit a `~/.zshrc` that is a link, and the run fails there. It keeps no copy of the
  old file.
- mise applies dotfiles before ws's units, so the first run adds the line before `/etc/ws/zshrc`
  exists. A run that fails between them leaves a zsh that reports the missing file as it starts,
  until a run gets through.
- `bootstrap status --missing` shows a missing line, and `bootstrap plan` does not. Setup's plan
  reads the status ([Root's paths](../frame/README.md#roots-paths)).
- Deleting the entry leaves the line. `mise dot unapply --force` takes it out, as a plain line has
  no marker to prove mise added it.
