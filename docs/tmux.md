# tmux

The `tmux` role installs tmux from Ubuntu's archive, with no config, as cld runs tmux with
`-f /dev/null`. Where tmux's snap is installed, the role removes it. Where cld's sessions run the
snap's tmux, end them before a real run. cld then finds `/usr/bin/tmux` first on the `PATH`, while
their key bindings call `/snap/bin/tmux`, which the removal takes away.

The server options that let helix's keys through tmux are in [helix's keys](helix.md#helixs-keys).
