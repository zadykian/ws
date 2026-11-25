# shell

The `shell` role sets up bash from `mise bootstrap`. Its `mise.guest.toml` installs
bash-completion and ble.sh, and writes ws's lines for bash to two files in `/etc/ws`, root's and
644. Its `units.toml.tmpl` puts one line of each at the top of the user's file. Its
`mise.secrets.toml` writes `/etc/ws/env` ([The env file](#the-env-file)):

- `/etc/ws/bashrc`, from `files/bashrc`, which `~/.bashrc` sources with `. /etc/ws/bashrc`.
  It sources `/etc/ws/env`. Its own guard of a shell that is not interactive follows, then ble.sh
  and bash-completion. The guard returns from that file alone, so `~/.bashrc` goes on to its own.
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
env file and no ble.sh. To have them, add stock's lines that run `~/.bashrc` to that
`~/.profile`:

```sh
if [ -n "$BASH_VERSION" ] && [ -f "$HOME/.bashrc" ]; then
  . "$HOME/.bashrc"
fi
```

A link at one of the user's files, or at a directory between the home and it, fails its unit
alone ([Links under the home](../security/README.md#links-under-the-home)).

## The env file

`/etc/ws/env` exports the shell's [secret inputs](../frame/README.md#secret-inputs), one per
variable. `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT` is the one today, which the `otelcol` role reads:
[claude's metrics elsewhere](../otelcol/README.md#claudes-metrics-elsewhere). The [secrets
task](../secrets/README.md#the-task) gives each input its value, from fnox, by the input's name.

- mise writes the file root's, at 0600.
- It opens with ws's header line, then `export NAME='VALUE'` for each input that is set, the value
  quoted for the shell. An empty input is left out, and where every input is empty the run removes
  the file.
- Each run with the secrets replaces the file, so a change goes in the secrets. A file there
  without ws's header at its top, as one written by hand, fails the plan and the run, and mise
  replaces nothing. Move its values into the secrets, then remove it.
- Another variable is an input of `mise.secrets.toml` and its name in both lists of the file's
  template.
- A run without the task, as CI's `server` bootstrap, keeps the file unchanged, one by hand included.
  No plan or log shows a value.

Zed captures the login shell's environment, so a project's capture holds each value too
([Secrets in ssh's arguments](../zed/README.md#secrets-in-sshs-arguments)).
