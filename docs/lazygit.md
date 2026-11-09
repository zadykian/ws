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

- The title reads `NAME: BRANCH`, such as `ws: main`. A linked worktree shows its main tree's name,
  and a detached HEAD its short hash.
- Each second, the wrapper reads lazygit's directory and its HEAD. So the title follows a checkout,
  and a move to another repository through lazygit's recent repositories.
- Outside tmux, it writes the title as OSC 2, which Ghostty's tab shows.
- In tmux, it sets the pane's title, and no write reaches the terminal that lazygit draws on.
- The terminal's stack keeps the shell's title, and gets it back within a second of lazygit's end.
  Where a terminal lacks the stack, the shell's next prompt sets its own title in an `xterm`
  terminal, as Ubuntu's `~/.bashrc` does.
- Where stdin or stdout is not a terminal, the wrapper runs lazygit and sets nothing.

Where the title shows:

- Ghostty's tab, through `ssh ws` with no tmux on ws.
- Not past tmux, as Zed's tmux and cld's leave `set-titles` off. A status line that names
  `#{pane_title}`, as tmux's default does, shows it.
- Not Zed's terminal tab, which takes no title: Zed shows one in the terminal's breadcrumbs alone,
  and its tmux passes none on.

Outside tmux, the watcher writes to the terminal while lazygit draws. Where a frame of lazygit's
takes more than one write, the title may land inside it and leave stray characters until those
cells change.
