# shell

The `shell` role sets up bash from `mise bootstrap`. Its `mise.server.toml` installs
bash-completion and ble.sh, and writes ws's lines for bash to two files in `/etc/ws`, root's and
644. Its `units.toml` puts one line of each at the top of the user's file, and writes `/etc/ws/env`:

- `/etc/ws/bashrc`, from `files/bashrc.tera`, which `~/.bashrc` sources with `. /etc/ws/bashrc`.
  It sources `/etc/ws/env` ([Secrets](#secrets)), and sets `DOTNET_RUNTIME_ID` from mise's
  `arch()`. Its own guard of a shell that is not interactive follows, then ble.sh and
  bash-completion. The guard returns from that file alone, so `~/.bashrc` goes on to its own.
- `/etc/ws/profile`, which `~/.profile` sources with `. /etc/ws/profile`. It sets `PATH` for the
  tools that ws's roles install, and sources rustup's env file.

`~/.bashrc` and `~/.profile` stay the user's, as `~/.zshrc` does on the Mac. The units
`file/shell-bashrc-line` and `file/shell-profile-line` put the lines at the top, not at the end.
Stock's guard returns first in a shell that is not interactive, which needs the env file too, and
`~/.profile` runs `~/.bashrc` before its end. A missing file is made with the line alone. Before it
adds the line, the run keeps the old file beside it as `PATH.TIME~`. A file that has the line stays
untouched, so a second run changes nothing. A user's settings go in those two files, below the
line.

A `~/.profile` that the run makes holds the line alone, so it does not run `~/.bashrc`, as stock's
file does. On an account made without `/etc/skel`, a login shell then reads no `/etc/ws/bashrc`: no
env file, no `DOTNET_RUNTIME_ID`, no ble.sh. To have them, add stock's lines that run `~/.bashrc` to
that `~/.profile`:

```sh
if [ -n "$BASH_VERSION" ] && [ -f "$HOME/.bashrc" ]; then
  . "$HOME/.bashrc"
fi
```

A link at one of the user's files, or at a directory between the home and it, fails its unit
alone ([Links under the home](../security/README.md#links-under-the-home)).

## Secrets

The `[env]` table of [the secrets file](#the-secrets-file), empty by default, maps names to values
for `/etc/ws/env`.

The unit `file/shell-env` reads ws's file, `/etc/ws/secrets.age`, with the identity of ws's
credential. With `[env]` set, it writes `/etc/ws/env` 0640, root's with the account's group:
`WS_USER` reads it and cannot change it. The file opens with ws's header line, then
`export NAME=VALUE` for each name, sorted, the value quoted for the shell. Each run replaces the
file, so a change goes in `[env]`. The `otelcol` role reads the file:
[claude's metrics elsewhere](../otelcol/README.md#claudes-metrics-elsewhere). The unit waits on the
frame's `state/ws-binary`, and signoz's and otelcol's units on it, so that they hang on one chain.

- A name takes letters, digits and underscores, and starts with no digit. The run refuses any
  other, and its message names no value.
- A value is a string, so `"0123"` and `"on"` stay as written.
- An `env` without that line at its top, as one written by hand, fails the run before the unit
  touches it, in a dry run too. Move its values into `[env]`, then remove the file.
- The unit neither prints nor diffs the file, so a dry run says only that it would change.

A machine with no secrets file, as CI's, leaves `/etc/ws/env` alone, even a file ws wrote, and the
unit says so. With the file and an empty `[env]`, the run removes `/etc/ws/env`, if ws wrote it.

## The secrets file

The secrets of the roles are a TOML file that age encrypts, in armor: `/etc/ws/secrets.age`, which
`ws server ubuntu secrets` edits. [ws's secrets](../secrets/README.md) says the file's tables, its
references and its values. A machine without the file, as CI's, has no secrets.

The identity stays in ws's credential, made by `ws server ubuntu secrets init`, or by `ssh -t NAME
sudo ws server ubuntu secrets init` from the Mac ([ws's file](../secrets/README.md#wss-file)). `ws
unit`, as root, decrypts it by `systemd-creds decrypt`, once a command, and only where a unit reads
the secrets. It reads no standard input and no terminal. With the file and without the credential,
such a unit fails.

The roles read these tables, and ws refuses any other key:

- `[env]`: names to values, as [Secrets](#secrets) says;
- `[git."URL"]`: the entries of [Git credentials](../git/README.md#git-credentials);
- `[file]` and `[password]`: the ikeward profile's files and the `.p12`'s password
  ([The profile](../ikeward/README.md#the-profile)).

No message, a plan's or a dry run's included, shows a value. Nothing plain reaches the disk but
the files that the roles write.
