# shell

The `shell` role, Go steps in `roles/server/ubuntu/shell`, writes ws's lines for bash to two
files in `/etc/ws`, root's and 644. It puts one line of each at the top of the user's file:

- `/etc/ws/bashrc`, which `~/.bashrc` sources with `. /etc/ws/bashrc`. It sources `/etc/ws/env`,
  which the role writes from the secrets file ([Secrets](#secrets)), and sets `DOTNET_RUNTIME_ID`.
  Its own guard of a shell that is not interactive follows, then ble.sh and bash-completion. The
  guard returns from that file alone, so `~/.bashrc` goes on to its own.
- `/etc/ws/profile`, which `~/.profile` sources with `. /etc/ws/profile`. It sets `PATH` for the
  tools that ws's roles install, and sources rustup's env file.

`~/.bashrc` and `~/.profile` stay the user's, as `~/.zshrc` does on the Mac. The lines go at the
top, not at the end. Stock's guard returns first in a shell that is not interactive, which needs the
env file too, and `~/.profile` runs `~/.bashrc` before its end. A missing file is made with the line
alone. Before it adds the line, the run keeps the old file beside it as `PATH.TIME~`. A
file that has the line stays untouched, so a second run changes nothing. A user's settings go in
those two files, below the line.

A `~/.profile` that the run makes holds the line alone, so it does not run `~/.bashrc`, as stock's
file does. On an account made without `/etc/skel`, a login shell then reads no `/etc/ws/bashrc`: no
env file, no `DOTNET_RUNTIME_ID`, no ble.sh. To have them, add stock's lines that run `~/.bashrc` to
that `~/.profile`:

```sh
if [ -n "$BASH_VERSION" ] && [ -f "$HOME/.bashrc" ]; then
  . "$HOME/.bashrc"
fi
```

A link at one of the user's files, or at a directory between the home and it, fails the step
alone ([Links under the home](../security/README.md#links-under-the-home)).

## Secrets

The `[env]` table of [the secrets file](#the-secrets-file), empty by default, maps names to values
for `/etc/ws/env`.

With it set, the role writes `/etc/ws/env` 0640, root's with the account's group: `WS_USER` reads it
and cannot change it.
The file opens with ws's header line, then `export NAME=VALUE` for
each name, the value quoted for the shell. Each run replaces the file, so a change goes in
`[env]`. The `otelcol` role reads the file later in the same run:
[claude's metrics elsewhere](../otelcol/README.md#claudes-metrics-elsewhere).

- A name takes letters, digits and underscores, and starts with no digit. The run refuses any
  other, and its message names no value.
- A value is a string, so `"0123"` and `"on"` stay as written.
- An `env` without that line at its top, as one written by hand, fails the run before the role
  touches it, in a dry run too. Move its values into `[env]`, then remove the file.
- The role neither prints nor diffs the file, so a dry run says only that it would change.

Without `--secrets`, the role leaves `/etc/ws/env` alone, even a file it wrote, so that a run
without the secrets file, as CI's, changes nothing. With `--secrets` and an empty `[env]`, the run
removes `/etc/ws/env`, if ws wrote it.

## The secrets file

`ws server ubuntu apply --secrets FILE` takes the secrets of the roles from a TOML file that age
encrypts, in armor. `ws` decrypts it in its process with an X25519 identity. A run without
`--secrets`, as CI's, has no secrets. [ws's secrets](../secrets/README.md) says the file's tables, its
references and its values.

The identity stays in ws's credential, made by `ws server ubuntu secrets init`, or by `ssh -t NAME
sudo ws server ubuntu secrets init` from the Mac ([ws's file](../secrets/README.md#wss-file)). `ws
server ubuntu apply` reads it from the systemd credential `ws-secrets`, in `$CREDENTIALS_DIRECTORY`,
for `--secrets` and `--check` alike. Without the credential it fails and names `LoadCredential`. It
reads no standard input and no terminal. A unit that runs it has
`LoadCredentialEncrypted=ws-secrets`. By hand, on a server, as root, with ws's file:

```sh
systemd-run --pty --wait -p LoadCredentialEncrypted=ws-secrets ws server ubuntu apply --secrets /etc/ws/secrets.age
```

systemd decrypts the credential for the unit alone, and drops it once the run ends.

The roles read these tables, and ws refuses any other key:

- `[env]`: names to values, as [Secrets](#secrets) says;
- `[git."URL"]`: the entries of [Git credentials](../git/README.md#git-credentials);
- `[file]` and `[password]`: the ikeward profile's files and the `.p12`'s password
  ([The profile](../ikeward/README.md#the-profile)).

No message, `--check` included, shows a value. Nothing plain reaches the disk but the
files that the roles write.
