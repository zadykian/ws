# The server's runbook

A person does these steps at ws, in this order. The Mac's side sets them in its own order
([The Mac's runbook](../../client/macos/runbook.md)).

1. Log in as the installer's user, by key.
2. Write `/etc/ws/values.yml`, root's and mode 600, with each value that differs from its
   default. `install.sh` reads `WS_USER` from it.
3. Run `install.sh` as root, with `curl -fsSL
   https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh`. It installs mise
   and runs its bootstrap, which puts ws in `/usr/local/bin`, which sudo reaches
   ([Releases](../../../.github/workflows/release.md)).
4. Make the secrets: `sudo ws server ubuntu secrets init`, then each secret by `sudo ws server
   ubuntu secrets set KIND NAME`. A value never goes in an argument
   ([The secrets](secrets/README.md)).
5. Run the bootstrap again, by hand as below: it writes what the secrets give, such as
   `/etc/ws/env` and gh's login ([gh](devtools/README.md#gh)). A second run changes nothing.

A run of the bootstrap by hand, or of its plan, names the data directory that `install.sh` gives
mise: `sudo env MISE_DATA_DIR=/usr/local/share/mise mise -C /opt/ws bootstrap`. There the account
reaches the tools. Without it, `guard/devtools-data-dir` fails the run
([devtools](devtools/README.md#the-tools)).
