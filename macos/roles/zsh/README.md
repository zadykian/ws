# zsh on the Mac

The `zsh` role sets up the Mac's shell: oh-my-zsh and two plugins, as mise's clones, and ws's
settings, which `~/.zshrc` sources. The hook that resets the terminal's modes has
[its own page](../ghostty/README.md#ghostty).

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
