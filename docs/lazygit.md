# lazygit

The `lazygit` role is Go steps in `internal/server/lazygit`. It installs lazygit from Ubuntu's
archive, with a wrapper that puts the repository and its branch in the terminal's title.

## The install

`Lazygit` installs the package `lazygit`, 0.57.0 in the universe of Ubuntu 26.04, as the
[tmux role](tmux.md) installs tmux. apt checks it, and updates it with the archive.

The role needs no other role and no secret. No step needs a booted systemd, so it runs whole in
CI's container.

## The title

lazygit sets no title of its own, so a tab keeps the shell's, `user@host: dir`, while it runs.
`Lazygit wrapper` writes `/usr/local/bin/lazygit`, mode 0755, from `files/lazygit`. It comes before
the package's `/usr/bin/lazygit` on every `PATH` of Ubuntu's.

- The title reads `NAME: BRANCH`, such as `ws: main`.
- The wrapper writes the title as OSC 2, which Ghostty's tab shows.
- The terminal's stack keeps the shell's title, and gets it back when lazygit ends. Where a
  terminal lacks the stack, the shell's next prompt sets its own title in an `xterm` terminal, as
  Ubuntu's `~/.bashrc` does.
- Where stdin or stdout is not a terminal, the wrapper runs lazygit and sets nothing.
