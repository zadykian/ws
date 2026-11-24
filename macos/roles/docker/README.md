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

Homebrew keeps the latest bottle only, so each formula takes `latest`, and `mise bootstrap packages
upgrade` upgrades them. mise's registry lacks the last two formulae of the list, so brew holds all
five.

`colima start` stays by hand: the machine's size and its start at login are the user's.
