# tmux

The `tmux` role sets up tmux from `mise bootstrap`. Its `mise.guest.toml` installs tmux from
Ubuntu's archive, with no config, as cld runs tmux with `-f /dev/null`, and names the unit
`tmux-snap` of its `units.toml.tmpl`, which removes tmux's snap. The package comes first, so that the
machine never lacks a tmux. The unit runs `snap remove tmux` where snapd is installed and lists
the snap.

Where cld's sessions run the snap's tmux, end them before a real run. cld then finds
`/usr/bin/tmux` first on the `PATH`, while their key bindings call `/snap/bin/tmux`, which the
removal takes away.

Fresh asks for them by [a wrapper](../fresh/README.md#the-wrapper). [lazygit's wrapper](../lazygit/README.md#the-title)
sets a pane's title.

The [zed role](../zed/README.md#terminals-in-tmux) reads a config of its own, with `-f`, on a socket of its
own, so the role's tmux still has none.
