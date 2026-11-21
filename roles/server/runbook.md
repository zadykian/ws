# The server's runbook

A person does these steps at ws, in this order. The Mac's side sets them in its own order
([The Mac's runbook](../client/runbook.md)).

1. Log in as the installer's user, by key. Run `sudo -H ssh-import-id gh:LOGIN`, which imports the
   GitHub account's keys into `/root/.ssh/authorized_keys`, which root logs in by afterward
   ([SSH and the firewall](ubuntu/security/README.md#ssh-and-the-firewall)). ws copies no key from
   the installer's `authorized_keys`.
2. Where a value differs from its default, write `/etc/ws/values.yml`, root's and mode 600, with
   each such value. Where none is, the bootstrap writes it ([the values](../../README.md#running-it-again)).
3. Run `install.sh` as root, by the three lines of
   [Install](../../.github/workflows/release.md#install), `set -o pipefail` among them. It
   installs mise and runs its bootstrap, which puts ws in `/usr/local/bin`, which sudo reaches.

The secrets come once the Mac has run `ws client secrets init`, one run for all servers, and
then `ws client secrets add NAME`, which makes ws's host key a recipient of the secrets
repository ([The secrets](../client/runbook.md#the-secrets)). add then runs the task `secrets` at
ws, which writes what the secrets give, such as `/etc/ws/env` and gh's login
([The repository](ubuntu/secrets/README.md#the-repository)). A later run under the secrets takes
the task's form below.

A run of the bootstrap by hand, or of its plan, takes this form:

```sh
sudo env MISE_LOCKED=1 mise -C /opt/ws/roles/server bootstrap
```

`MISE_LOCKED=1` installs a tool only as the lock holds it, by its URL and checksum, as CI's runs
do. With `bootstrap plan --detailed-exitcode` in place
of `bootstrap`, it changes nothing, and exits 0 where a run would change nothing.

A run under ws's secrets, or its plan, goes through the task `secrets`, which fetches them by the
host key and passes its arguments to the bootstrap
([The task](ubuntu/secrets/README.md#the-task)):

```sh
sudo env MISE_LOCKED=1 mise -C /opt/ws/roles/server run secrets
sudo env MISE_LOCKED=1 mise -C /opt/ws/roles/server run secrets -- plan --detailed-exitcode
```

A run of the bootstrap without the task loads no `secrets` env, and keeps the files that env's
inputs write as they are.
