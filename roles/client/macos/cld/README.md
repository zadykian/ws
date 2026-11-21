# cld on the Mac

`mise -C /opt/ws/roles/client bootstrap` sets up [cld](https://github.com/zadykian/cld) on the Mac, for `cld
restore tabs`, by [cld's unit](units.toml.tmpl). `state.cld` runs `ws client cld` as the user. It
installs cld into `~/.local/bin` where missing, and runs `cld update` where a newer release is out.
Its check, `--check`, reads the latest release as the command does, so a cld at the latest changes
nothing. Both come from GitHub's releases, and a cld built from a clone stays. A failed
install or update, or a latest release that GitHub withholds, warns, and the unit fails while cld
is still due. [Ghostty and cld on the Mac](../ghostty/README.md#cld) says what cld does there, and
how ssh starts its watcher.
