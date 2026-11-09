# lazygit

The `lazygit` role is Go steps in `internal/server/lazygit`. It installs lazygit from Ubuntu's
archive, with a wrapper in front of it.

## The install

`Lazygit` installs the package `lazygit`, 0.57.0 in the universe of Ubuntu 26.04, as the
[tmux role](tmux.md) installs tmux. apt checks it, and updates it with the archive.

The role needs no other role and no secret. No step needs a booted systemd, so it runs whole in
CI's container.

## The wrapper

`Lazygit wrapper` writes `/usr/local/bin/lazygit`, mode 0755, from `files/lazygit`. It comes before
the package's `/usr/bin/lazygit` on every `PATH` of Ubuntu's, and runs it with the same arguments.
