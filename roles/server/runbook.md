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
   installs ws alone, as `/usr/local/bin/ws`, which sudo reaches, and prints the next command.
4. Run `ws server setup` as root.

`ws server setup`, run as root, sets the machine up. Before any write it refuses
a user but root and a system but Ubuntu 26.04. It refuses a tree in `/opt/ws`, or a directory above
it, that another user owns or that a group or others may write, as mise runs the tree as root. It
installs mise at the version ws pins, checked against its SHA-256. It writes `/etc/mise/miserc.toml`
and `/etc/mise/config.toml`, which trusts the tree ([The machine's
env](ubuntu/frame/README.md#the-machines-env)). It checks this ws's release tag out in `/opt/ws`
and cleans the tree. It links the plugin into `/usr/share/mise/plugins`, and runs mise's bootstrap
in `/opt/ws/roles/server` with umask 022, from `/`. A second run changes nothing.

- A ws that names no release tag, `devel` and a pseudo-version included, is refused. `--in-place`
  runs the tree in `/opt/ws` as it stands, after the guard.
- `--plan` plans these steps, and mise's bootstrap, or the task `secrets` where setup would run it,
  where none of them is due, and changes nothing. It exits 0 where nothing would change, and 2
  where something would.
- `--secrets` runs the task `secrets` in place of the bootstrap, as below.

The secrets come once the Mac has run `ws client secrets init`, one run for all servers, and
then `ws client secrets add NAME`, which makes ws's host key a recipient of the secrets
repository ([The secrets](../client/runbook.md#the-secrets)). add then runs
`ws server setup --secrets` at ws, whose task `secrets` clones the secrets on its first run and
writes what they give, such as `/etc/ws/env` and gh's login
([The repository](ubuntu/secrets/README.md#the-repository)). A later run under the secrets takes
the form below.

A run by hand, or its plan, takes `ws server setup`:

```sh
sudo ws server setup
sudo ws server setup --plan
```

The tree's `locked = true` installs a tool only as the lock holds it, by its URL and checksum, as
CI's runs do. The plan changes nothing, and exits 0 where a run would change nothing.

A run under ws's secrets, or its plan, goes through `ws server setup`, whose task `secrets`
fetches them by the host key and passes its arguments to the bootstrap
([The task](ubuntu/secrets/README.md#the-task)):

```sh
ws server setup --secrets
ws server setup --secrets --plan
```

Without `--secrets`, setup runs the task where `/var/lib/ws/secrets` stands, and the bootstrap
where it does not. `--plan` plans through the task there, so the clone fast-forwards first, and
the plan reads the secrets that origin holds.

A run of the bootstrap without the task loads no `secrets` env, and keeps the files that env's
inputs write as they are.
