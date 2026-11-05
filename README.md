# ws

Sets up my development server from code. `ws server apply` runs on the machine itself. It sets up
packages, SSH by key and a firewall, and a direct link and a WireGuard tunnel to the Mac. It sets up
Docker, the shell, tmux, the helix editor and Zed's server. It sets up claude and
[cld](https://github.com/zadykian/cld), and SigNoz with a collector for the machine's telemetry.
Running it again is safe, and `--check` shows how a machine differs from the repository.

Each role has a page of its own. A role is Go steps of ws:

- [`base`](docs/base.md): packages, the locale, the timezone, security updates and a swap file,
  as Go steps.
- [`security`](docs/security.md): SSH by key only, ufw and fail2ban, and a table that keeps what
  ws starts to the internet and ikeward, but what the network needs, such as DHCP and DNS. The
  table is off where `security_outbound` is false, as a cloud VM wants.
- [`network`](docs/network.md): the server's end of a cable to the Mac, as Go steps.
- [`wireguard`](docs/wireguard.md): `wg0`, the server's end of a WireGuard tunnel for the Mac.
- [`ikeward`](docs/ikeward.md): ikeward's IKEv2 VPN, where a profile is given, and the WireGuard
  peers' way into it.
- [`fans`](docs/fans.md): fan curves for the board's headers, and the driver that sets them.
- [`shell`](docs/shell.md): `~/.bashrc`, `~/.profile`, `~/.gitconfig`, `~/.secrets/env` from the
  secrets file, a key that signs commits, git's tokens for servers besides GitHub, and clones of
  repositories.
- [`tmux`](docs/tmux.md): tmux from Ubuntu's archive.
- [`devtools`](docs/devtools.md): Node, the .NET SDKs, gh, glow, Chromium's libraries, Go,
  gopls, Delve, roslyn-language-server and rustup.
- [`docker`](docs/docker.md): Docker Engine, its `daemon.json`, a slice for its containers and
  `ws_user`'s rootless daemon.
- [`claude`](docs/claude.md): claude, its user settings and telemetry, its sessions after a reboot,
  and plugin marketplaces from a list.
- [`signoz`](docs/signoz.md): SigNoz in Docker, which keeps the machine's telemetry for 90 days.
- [`otelcol`](docs/otelcol.md): the OpenTelemetry Collector on the host, which feeds SigNoz.
- [`helix`](docs/helix.md): the helix editor, with the IDEs' keys and its config for the Go and C#
  servers.
- [`zed`](docs/zed.md): the settings of Zed's server for Go, C#, SQL, Docker, Ansible and HTML, the npm
  servers, the terminals' tmux wrapper, and the open-file limit that it needs.
  [A project's settings](docs/zed-projects.md) cover SQL Server and Ansible.

`mac/` sets up the Mac that works on the server, and Ghostty and Zed on it: [The Mac](docs/mac.md).

On a new Mac with Apple silicon, one line installs ws's binary. Then `ws mac wizard` installs
Apple's Command Line Tools and Homebrew, and sets up the rest ([The Mac's wizard](docs/mac-wizard.md)):

```sh
curl -fsSL https://github.com/zadykian/ws/releases/latest/download/install.sh | sh
~/.local/bin/ws mac wizard
```

## A new machine

Install Ubuntu Server 26.04 by hand from its installer. In it, turn OpenSSH on, import the GitHub
account's keys, and leave password login off. Neither the Mac's wizard nor `ws server apply` adds a
key later. So run the Mac's line above first, which puts the Mac's key on the account ([GitHub and
Ubuntu's install](docs/mac-wizard.md#github-and-ubuntus-install)). Name the installer's user
`zadykian`, ws's default user. A cloud VM has no installer: it gives its default user the provider's
keys, which must be the account's. A VM with root alone logs in as root, where `WS_TARGET=cloud`.

Then, logged in as that user, install ws and apply it through `sudo`. `ws server apply` logs nothing
in before it: the `devtools` role installs gh, and its login comes by hand ([By hand](#by-hand)).

```sh
curl -fsSL https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh
sudo ws server apply
```

It runs the roles in order. Then do what stays by hand, below.

## Running it again

Each ws holds the roles of its release, so update ws first, then apply:

```sh
sudo ws update
sudo ws server apply
```

A ws before v0.5.1 lacks `ws update`: [Update](docs/releases.md#update).

`ws server apply --only NAME,...` runs the roles named, in the order they run in. A name is a role,
or a step of it by the name a run shows, as in `--only 'docker/Docker group',tmux`. That role's
other steps are planned and left out. A step that waits on one of them with changes is skipped,
and says so, and `--check` shows what the same run would change. A left-out step whose plan fails
warns, and the run goes on. `--secrets FILE` gives the roles their secrets
([The secrets file](docs/shell.md#the-secrets-file)).
The values of a machine go in `/etc/ws/host_vars.yml`, which only root reads, and every run reads
them. `ws_user: NAME` sets up the user NAME instead of `zadykian`, the default, whose account must
exist. `ws_user: root` sets up a cloud VM with no other user. `ws_home` follows the user as
`/home/NAME`; `ws_home: DIR` sets another home. A cloud VM names
`security_outbound: false` there before its first run ([Outbound](docs/security.md#outbound)).
ws's last line names the roles that changed, `changed: none` on a second run.

## Dry runs

With `--check`, ws plans its steps and shows what each would change, and nothing changes:

```sh
sudo ws server apply --check
```

A dry run adds no apt repository. So where a step needs a repository still to come, it says "after"
and the repository's step, and plans nothing more.

A dry run checks no file it would write. ws runs a file's check, such as `nft -c`, `sshd -G` or
`dockerd --validate`, only as it writes the file.

## State

`ws server state` reads and applies nothing. It needs what `--check` needs, root on Ubuntu 26.04,
and takes `--only` as `apply` does. It reports what ws manages, not the rest of the machine: each
step's packages, services and files, what stands against ws, and the changes that `apply` would
make. A step that fails stands in the report as a check, and the exit code stays 0. It takes no
secrets, so a report holds none and says nothing of what they write.

```sh
sudo ws server state
sudo ws server state --json
```

The report is on stdout and ws's own lines on stderr. `--json` prints the version of ws, then
each role with its steps, items and changes.

## By hand

Secrets are never in this repository. On a new machine, [The Mac's wizard](docs/mac-wizard.md)
checks that `WS_USER` logs in and installs its own ws on ws and runs `ws server apply`, with the
Mac's secrets file. Then it adds ws's signing key to GitHub, logs claude in there, and gives `wg0`
the Mac's key.
Without the wizard, `gh auth setup-git`, the claude login and the Mac's key stay manual. ws's gh
takes a fine-grained personal access token, which `ws server apply` logs gh in with, from the
secrets file's `token.github` ([gh](docs/devtools.md#gh)). Its permissions are the
account's selected repositories; Metadata and Actions read; Contents, Pull requests, Issues and
Workflows read and write; and no account permission. These stay manual:

- other peers' public keys on the server, as in [The tunnel](docs/wireguard.md#the-tunnel);
- the VPN's profile, age-encrypted on the server, and the `ikeward` role's variables, as in
  [The profile](docs/ikeward.md#the-profile);
- with Secure Boot on, DKMS's key, enrolled at the console: [Secure Boot](docs/fans.md#secure-boot);
- the local list of words: set the secret `FORBIDDEN_WORDS` on GitHub, and place
  `~/.config/ws/forbidden-words`, mode 600, on each machine that commits
  ([The hooks](docs/checks.md#the-hooks)).

The router takes a few steps once, by hand: [The router](docs/router.md).

## Checks

CI runs `make lint`, which lints the code, the docs and the workflows, and scans the commits for
secrets. It runs `ws server apply` twice in a container, tries `install.sh`, forges SigNoz's compose
files again, and sets up a Mac on GitHub's macOS runner. A pushed tag `vX.Y.Z` publishes the release
that `install.sh` installs: [Releases](docs/releases.md). The rules workflow warns where a pull
request breaks the rules of zadykian/toolkit's skills, and the issues workflow where an issue strays
from its issue form. The hooks in `.githooks` check each commit before git makes it.
[Checks](docs/checks.md) and [The gates](docs/gates.md) have the details.

On a terminal, ws draws a full-screen display, and `--plain` keeps its lines: [The terminal UI](docs/tui.md).

## Pull requests

Pull requests land on `main` by GitHub's merge commits, as the
[land](https://github.com/zadykian/toolkit/blob/main/skills/land/SKILL.md) skill says. Their signed
commits land as they are, under a merge commit that GitHub signs. The ruleset on `main` allows the
merge method alone, and requires signed commits and branches up to date.

## License

[MIT](LICENSE), but for the Host dashboard in `internal/server/signoz/dashboards`, which is SigNoz's,
under the [Apache License 2.0](internal/server/signoz/dashboards/LICENSE).
