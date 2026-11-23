# The server's runbook

A person does these steps at ws, in this order. The Mac's side sets them in its own order
([The Mac's runbook](../client/runbook.md)).

## A new server from the installer

A release publishes `autoinstall.yaml`, the installer's answers for Ubuntu Server 26.04. With it
the installer sets root's keys from the GitHub account, and the console password. The first boot
runs ws's setup of that release, so the steps by hand below are not done. The secrets still come
after it, as [Setup, the secrets and the runs](#setup-the-secrets-and-the-runs) says.

1. Fill the file's host name, root's hash, from `mkpasswd -m sha-512`, and the values, as a `|`
   block of `/etc/ws/values.yml`'s text, or `""` where no value differs.
2. Copy it as `user-data` to a volume labelled `CIDATA`, beside an empty `meta-data`.
3. Boot the installer with `autoinstall` on the kernel's line. It asks for the disk alone.
4. Read the first boot's run in `/var/log/cloud-init-output.log`, or by
   `cloud-init status --wait --long`.

[The autoinstall file](../../.github/workflows/release.md#the-autoinstall-file) says what the
release fills in.

## The steps by hand

A cloud VM has no installer, so it takes these steps. So does any server whose installer ran
without the file.

1. Log in as the installer's user, by key. Run `sudo -H ssh-import-id gh:LOGIN`, which imports the
   GitHub account's keys into `/root/.ssh/authorized_keys`, which root logs in by afterward
   ([SSH and the firewall](ubuntu/security/README.md#ssh-and-the-firewall)). ws copies no key from
   the installer's `authorized_keys`.
2. Where a value differs from its default, write `/etc/ws/values.yml`, root's and mode 600, with
   each such value. Where none is, the bootstrap writes it ([the values](../../README.md#running-it-again)).
3. Run `install.sh` as root, by the three lines of
   [Install](../../.github/workflows/release.md#install), `set -o pipefail` among them. It
   installs ws alone, as `/usr/local/bin/ws`, which root's `PATH` holds, and prints the next command.
4. Run `ws server setup` as root.

## Setup, the secrets and the runs

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
- A ws of another version than `/usr/local/bin/ws`, which the units run, is refused before any
  write, `--in-place` included, and so is a missing file there. `ws update --version X.Y.Z`
  installs that release there.
- `--plan` plans these steps, and mise's bootstrap, or the task `secrets` where setup would run it,
  where none of them is due, and changes nothing. It exits 0 where nothing would change, and 2
  where something would.
- `--secrets` runs the task `secrets` in place of the bootstrap, as below.
- `ws update`, as root, brings ws to the latest release, or to `--version`'s, and runs that ws's
  setup. A new release, or a trial's end, takes it, not `install.sh`
  ([Update](../../.github/workflows/release.md#update)).

The secrets come once the Mac has run `ws client secrets init`, one run for all servers, and
then `ws client secrets add NAME`, which makes ws's host key a recipient of the secrets
repository ([The secrets](../client/runbook.md#the-secrets)). add then runs
`ws server setup --secrets` at ws, whose task `secrets` clones the secrets on its first run and
writes what they give, such as `/etc/ws/env` and gh's login
([The repository](ubuntu/secrets/README.md#the-repository)). A later run under the secrets takes
the form below.

A run by hand, or its plan, takes `ws server setup`:

```sh
ws server setup
ws server setup --plan
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
