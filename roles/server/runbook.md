# The server's runbook

A person does these steps at ws, in this order. The Mac's side sets them in its own order
([The Mac's runbook](../client/macos/runbook.md)).

1. Log in as the installer's user, by key.
2. Write `/etc/ws/values.yml`, root's and mode 600, with each value that differs from its
   default. `install.sh` reads `WS_USER` from it.
3. Run `install.sh` as root, by the three lines of
   [Install](../../.github/workflows/release.md#install), `set -o pipefail` among them. It
   installs mise and runs its bootstrap, which puts ws in `/usr/local/bin`, which sudo reaches.
4. Make the secrets: `sudo ws server ubuntu secrets init`, then each secret by `sudo ws server
   ubuntu secrets set KIND NAME`. A value never goes in an argument
   ([The secrets](secrets/README.md)).
5. Run the bootstrap again, by hand as below: it writes what the secrets give, such as
   `/etc/ws/env` and gh's login ([gh](ubuntu/devtools/README.md#gh)). A second run changes nothing.

A run of the bootstrap by hand, or of its plan, takes this form:

```sh
sudo env MISE_DATA_DIR=/usr/local/share/mise MISE_LOCKED=1 mise -C /opt/ws bootstrap
```

`MISE_DATA_DIR` is the data directory that `install.sh` gives mise, where the account reaches the
tools. Without it, `guard/devtools-data-dir` fails the run
([devtools](ubuntu/devtools/README.md#the-tools)). `MISE_LOCKED=1` installs a tool only as the lock holds
it, by its URL and checksum, as CI's runs do. With `bootstrap plan --detailed-exitcode` in place
of `bootstrap`, it changes nothing, and exits 0 where a run would change nothing.
