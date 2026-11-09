# tmux

The `tmux` role, Go steps in `internal/server/tmux`, installs tmux from Ubuntu's archive, with no
config, as cld runs tmux with `-f /dev/null`. Where tmux's snap is installed, the role removes it.
Where cld's sessions run the snap's tmux, end them before a real run. cld then finds
`/usr/bin/tmux` first on the `PATH`, while their key bindings call `/snap/bin/tmux`, which the
removal takes away.

Fresh asks for them by [a wrapper](fresh.md#the-wrapper). [lazygit's wrapper](lazygit.md#the-title)
sets a pane's title.

The [zed role](zed.md#terminals-in-tmux) reads a config of its own, with `-f`, on a socket of its
own, so the role's tmux still has none.
