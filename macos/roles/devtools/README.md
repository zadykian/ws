# devtools on the Mac

The `devtools` role puts Go on the Mac, by [`mise.macos.toml`](mise.macos.toml) and
[the units](units.toml.tmpl), as the server's
[devtools](../../../ubuntu/roles/devtools/README.md) does on Linux.

## Go

Go is a tool of mise's, `go = "1.27.3"`, the server's version. The tree's lock,
[`mise.macos.lock`](../../mise/mise.macos.lock), holds the URL and the SHA-256 of Go's tarball for
Apple silicon, which Go's publisher lists, and mise checks the download against it. A bump changes
`[tools]` and the unit's link together, and the lock by `mise lock` for `macos-arm64`.

mise installs Go under `~/.local/share/mise/installs/go/1.27.3`, on no shell's `PATH`. The unit
`file/devtools-go` links `/usr/local/go` to it, root's, so that a program that looks there, as
the server's tools do, finds `go`.
