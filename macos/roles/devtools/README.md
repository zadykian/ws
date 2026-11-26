# devtools on the Mac

The `devtools` role puts Go and the command-line tools on the Mac, by [`mise.macos.toml`](mise.macos.toml) and
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

## The formulae

mise installs these formulae by its `brew` manager, which needs no Homebrew: it pours Homebrew's
bottle into `/opt/homebrew`, each dependency first, and checks its SHA-256 against Homebrew's API.

- `gnupg` and `pinentry-mac`, for signing and the YubiKey's keys;
- `ykman`, the YubiKey's command line;
- `ripgrep`, `tokei`, `tree` and `pwgen`.

A formula takes the latest alone, as Homebrew keeps the current bottle only, and mise never
upgrades it by itself: the command in [the runbook](../../runbook.md#what-stays-by-hand) does. So
the formulae are outside the SHA-256 rule of the pinned downloads
([CLAUDE.md](../../../.claude/CLAUDE.md)). A first run pours about 30 bottles, as `gnupg` has 10
dependencies.

mise could lock `ripgrep` as a tool of its own, by its SHA-256. But such a tool is on no `PATH` on
the Mac, and each would take a link of ws's. `tokei` has no build to lock, as mise's registry builds
it from source by cargo. So brew's latest stands for both, its bottles checked as above.

## The PATH

`file/devtools-path` writes `/etc/paths.d/ws`, root's, with the two directories that the tools
need: `/opt/homebrew/bin`, where mise links the formulae and the casks' commands, such as Zed's,
and `/usr/local/go/bin`. `/usr/libexec/path_helper` puts each on a login shell's `PATH`, where
Homebrew is not set up. A shell open before the first run finds none of them until it logs in
again.

## gpg's agent

`[bootstrap.directories]` makes `~/.gnupg`, the user's and 0700, as gpg makes it, and mise keeps
its mode on each run. A `[dotfiles]` entry seeds `gpg-agent.conf` there, at 0600, with
`pinentry-program /opt/homebrew/bin/pinentry-mac`: gnupg's own pinentry is a curses program, which
finds no terminal where an agent asks. The entry is `create_if_missing`, so mise writes where no
file is and leaves the user's changes. `bootstrap status --missing` shows a missing one, and the
plan does not ([Root's paths](../frame/README.md#roots-paths)). The keys, gpg's and the YubiKey's,
are made by hand.
