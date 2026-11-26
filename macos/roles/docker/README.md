# docker on the Mac

The `docker` role puts Docker's command line and a machine to run it on the Mac, by
[`mise.macos.toml`](mise.macos.toml). The server's [docker](../../../ubuntu/roles/docker/README.md)
runs the engine itself.

## The formulae

mise pours these by its `brew` manager, which needs no Homebrew, each checked against the SHA-256
in Homebrew's API ([devtools](../devtools/README.md#the-formulae) says what that means):

- `colima`, the machine, with `lima` as its dependency;
- `docker`, the command line alone;
- `docker-buildx` and `docker-compose`, Docker's plugins;
- `docker-credential-helper`, which holds `docker-credential-osxkeychain`.

Homebrew keeps the latest bottle only, so each formula takes `latest`, and the command in
[the runbook](../../runbook.md#what-stays-by-hand) upgrades them. One manager holds the five:
`docker` finds buildx and compose in brew's `cli-plugins` directory, which `config.json` names, and
the other commands are in `/opt/homebrew/bin`, on a login shell's `PATH`. As tools of mise's, each
plugin would take a link of ws's into a `cli-plugins` directory, and each other command one onto
the `PATH`.

`colima start` stays by hand: the machine's size and its start at login are the user's.

## The CLI's config

A `[dotfiles]` entry seeds `~/.docker/config.json`, the user's and 0600. As
`create_if_missing`, mise writes where no file is and leaves the user's changes. The file holds two
keys. `cliPluginsExtraDirs` is `["/opt/homebrew/lib/docker/cli-plugins"]`, where the formulae link
buildx and compose, and `docker` finds them nowhere else. `credsStore` is `osxkeychain`, which
keeps registry logins in the Keychain through `docker-credential-osxkeychain`.
`bootstrap status --missing` shows a missing file, and the plan does not
([Root's paths](../frame/README.md#roots-paths)).
