# ws

Sets up my development server from code. mise's bootstrap runs on the machine itself, with ws's
units. It sets up packages, SSH by key and a firewall, and a cable and a WireGuard tunnel to the
Mac. It sets up Docker, the shell, git, tmux, lazygit, the Fresh editor and Zed's server. It sets
up claude and [cld](https://github.com/zadykian/cld), and SigNoz with a collector for the
machine's telemetry.
Running it again is safe, and its plan shows how a machine differs from the repository.

Each role has a page of its own. A role is its mise files by env, and its spec of ws's units where
it has any:

- [`frame`](roles/server/ubuntu/frame): what each role of mise's config leans on, and
  `/usr/local/bin/ws`, the pinned binary.
- [`base`](roles/server/ubuntu/base): packages, the locale, the timezone, security updates and a swap file,
  as mise's packages and units.
- [`security`](roles/server/ubuntu/security): SSH by key only, ufw and fail2ban, and a table that keeps what
  ws starts to the internet and ikeward, but what the network needs, such as DHCP and DNS. The
  table is off where `WS_SECURITY_OUTBOUND` is false, as a cloud VM wants. As mise's config and
  units.
- [`secrets`](roles/server/ubuntu/secrets): fnox, which decrypts ws's secrets, as mise's config, and
  the task `secrets`, which fetches them by the host key and runs the bootstrap under them.
- [`cable`](roles/server/ubuntu/cable): the server's end of a cable to the Mac, as units.
- [`tunnel`](roles/server/ubuntu/tunnel): `wg0`, the server's end of a WireGuard tunnel for the Mac, as
  units.
- [`ikeward`](roles/server/ubuntu/ikeward): ikeward's IKEv2 VPN, where a profile is given, and the WireGuard
  peers' way into it, as units.
- [`nvidia`](roles/server/ubuntu/nvidia): NVIDIA's headless driver from Ubuntu's archive, on a machine with an
  NVIDIA GPU.
- [`fans`](roles/server/ubuntu/fans): fan curves for the board's headers, and the driver that sets them.
- [`shell`](roles/server/ubuntu/shell): `/etc/ws/bashrc` and `/etc/ws/profile`, one line of each at the top of
  `~/.bashrc` and `~/.profile`, and `/etc/ws/env` from mise's secret inputs.
- [`git`](roles/server/ubuntu/git): `/etc/ws/gitconfig` and its include in `~/.gitconfig`, a key that
  signs commits, git's tokens for servers besides GitHub, and clones of repositories.
- [`tmux`](roles/server/ubuntu/tmux): tmux from Ubuntu's archive.
- [`lazygit`](roles/server/ubuntu/lazygit): lazygit from Ubuntu's archive, in JetBrains Dark, with the
  repository and its branch in the terminal's title.
- [`devtools`](roles/server/ubuntu/devtools): Node, the .NET SDKs, gh, glow, Chromium's libraries, Go,
  gopls, Delve, roslyn-language-server, the SQL, Docker and npm language servers, and rustup.
- [`docker`](roles/server/ubuntu/docker): Docker Engine, its `daemon.json` and a slice for its containers.
- [`claude`](roles/server/ubuntu/claude): claude, its user settings and telemetry, its sessions after a reboot,
  and plugin marketplaces from a list.
- [`signoz`](roles/server/ubuntu/signoz): SigNoz in Docker, which keeps the machine's telemetry for 90 days.
- [`otelcol`](roles/server/ubuntu/otelcol): the OpenTelemetry Collector on the host, which feeds SigNoz.
- [`fresh`](roles/server/ubuntu/fresh): the Fresh editor, from the .deb of a pinned release, with the plain keys,
  JetBrains Dark and the language servers.
- [`zed`](roles/server/ubuntu/zed): the settings of Zed's server for Go, C#, SQL, Docker, Ansible and HTML, the
  terminals' tmux wrapper, and the open-file limit that it needs.
  [A project's settings](roles/server/ubuntu/zed/projects.md) cover SQL Server and Ansible.

`roles/client/` sets up the Mac that works on the server, and Ghostty and Zed on it: [The Mac](roles/client).
Its [`frame`](roles/client/macos/frame) is what the Mac's roles of mise's config lean on, and
`/usr/local/bin/ws`. Its [`firewall`](roles/client/macos/firewall) turns the Application Firewall on
and has it block all incoming.

A new Mac with Apple silicon follows [the Mac's runbook](roles/client/runbook.md), as
`install.sh` is the server's alone. By hand, it installs ws's binary and mise, each checked against
its SHA-256, clones a release's tree into `/opt/ws`, and links its plugin where mise finds it.
Then it writes the values ([The Mac's values](roles/client/values/README.md)) and runs
`MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap` ([The Mac](roles/client/README.md)). [The server's runbook](roles/server/runbook.md) lists
the server's own steps.

## A new machine

Install Ubuntu Server 26.04 by hand from its installer. In it, turn OpenSSH on, import the GitHub
account's keys, and leave password login off. ws sets the server up for root. Root's keys come from
the GitHub account alone: the installer imports them into its own user's home, and ws copies no key
from another account's file. So log in as the installer's user, by key, and run `sudo -H
ssh-import-id gh:LOGIN` ([the runbook](roles/server/runbook.md)). A cloud VM has no installer: its
default user holds the provider's keys, which must be the account's. cloud-init has put the same
keys in root's `authorized_keys` behind a forced command that refuses a login, and ssh-import-id
skips a key that is there. So delete those lines first, then import the keys into root the same way.
A VM with root alone logs in as root, where `WS_TARGET=cloud`. An image with Docker holds Docker's
apt source, which apt refuses beside mise's: remove it first
([docker](roles/server/ubuntu/docker/README.md)).

Then, logged in as that user, write `/etc/ws/values.yml` where a value differs from its default,
root's and mode 600 ([the values](#running-it-again)). Run `install.sh` through `sudo`, by the
three lines of [Install](.github/workflows/release.md#install), which fail where curl does. It
installs mise and runs its bootstrap, which installs ws and sets each role up. Once the secrets
hold `WS_GITHUB_TOKEN`, the secrets task logs gh in ([By hand](#by-hand)). The Mac's
`ws client secrets add NAME` runs it first, as [the server's runbook](roles/server/runbook.md) says.

Then do what stays by hand, below.

## Running it again

Each release holds the roles it sets up, so run `install.sh` again, by the same lines of
[Install](.github/workflows/release.md#install): it checks the release out and runs the bootstrap.
`WS_VERSION=X.Y.Z` before `sh` picks a release.

The units read their secrets from mise's secret inputs, which the task `secrets` fetches and
decrypts on ws ([The repository](roles/server/ubuntu/secrets/README.md#the-repository)). The
values of a machine go in `/etc/ws/values.yml`, which only root reads, and every run reads them.
The first run writes it where none is, each key at its default, and no run changes it after
([The values file](roles/server/ubuntu/frame/README.md#the-values-file)). The server's user is
root, in `/root`: no value names either. `WS_GIT_SIGNING_PUB_KEY` is `/root/.ssh/id_ed25519_signing.pub`
by default, and lies below `/root`, or the run fails. A cloud VM names `WS_SECURITY_OUTBOUND:
false` there before its first run ([Outbound](roles/server/ubuntu/security/README.md#outbound)). `ws
server values set KEY VALUE` writes one key, and an empty VALUE gives its default back. `ws server
values check` and each plan refuse a bad value before any write
([The check of the values](roles/server/ubuntu/frame/README.md#the-check-of-the-values)). A second
run changes nothing, so `ws server setup --plan` exits 0.

## Dry runs

Setup's plan shows what a run would change, and nothing changes. A run by hand takes the form
that [the server's runbook](roles/server/runbook.md) gives:

```sh
sudo ws server setup --plan
```

`ws server setup --secrets --plan` plans through the task `secrets`, which fetches the secrets
first ([The task](roles/server/ubuntu/secrets/README.md#the-task)).

A plan adds no apt source and writes no file. So a unit that waits on another with changes reads
`after` and that unit, and plans nothing more.

A plan checks no file it would write. A unit runs a file's check, such as `nft -c`, `sshd -G` or
`dockerd --validate`, only as it writes the file.

ws's units show more, as root on the server and as the user on the Mac
([ws unit](engine/core/kind/README.md#ws-unit)). `ws unit list` names them in the order they run,
`ws unit read UNIT` prints a unit's state, and `ws unit write --dry-run UNIT` its changes, a file's
diff included, but a secret's. A run by hand of `ws unit write UNIT` sets up that unit alone.

## By hand

Secrets are never in this repository. The claude login, ws's signing key on GitHub and the Mac's key
as `wg0`'s peer stay manual. ws's gh takes a fine-grained personal access token, which the secrets
task's run logs gh in with, from the secret input `WS_GITHUB_TOKEN`
([gh](roles/server/ubuntu/devtools/README.md#gh)). Its permissions are the account's selected
repositories; Metadata and Actions read; Contents, Pull requests, Issues and Workflows read and
write; and no account permission. These stay manual:

- other peers' public keys on the server, as in [The tunnel](roles/server/ubuntu/tunnel/README.md#the-tunnel);
- another device's SSH key on the server: `ssh-import-id gh:LOGIN`, run as root
  ([SSH and the firewall](roles/server/ubuntu/security/README.md#ssh-and-the-firewall));
- ikeward's VPN profile, in the server's fnox profile, `fnox.HOSTNAME.toml`, and the `ikeward`
  role's variables, as in [The profile](roles/server/ubuntu/ikeward/README.md#the-profile);
- with Secure Boot on, DKMS's key, enrolled at the console: [Secure Boot](roles/server/ubuntu/fans/README.md#secure-boot);
- Fresh's trust, once per repository: press T, then Enter
  ([Trust each project](roles/server/ubuntu/fresh/README.md#trust-each-project));
- the local list of words: set the secret `FORBIDDEN_WORDS` on GitHub, and place
  `~/.config/ws/forbidden-words`, mode 600, on each machine that commits
  ([The hooks](.github/workflows/README.md#the-hooks)).

The router takes a few steps once, by hand: [The router](roles/server/ubuntu/cable/router.md).

## Checks

CI runs `make lint`, which lints the code, the docs and the workflows, and scans the commits for
secrets. It runs `ws server setup` twice on a booted runner, and setup's plan must exit 0 after
each run. It tries `install.sh`, forges SigNoz's compose files again, and sets up a Mac on
GitHub's macOS runner by the bootstrap. A pushed tag `vX.Y.Z` publishes the
release whose tree `install.sh` runs, and `vX.Y.Z-rc.N` a pre-release of a `feat/` branch:
[Releases](.github/workflows/release.md). The rules workflow warns where a
pull request breaks the rules of zadykian/toolkit's skills, and the issues workflow where an issue
strays from its issue form. The hooks in `.githooks` check each commit before git makes it.
[Checks](.github/workflows) and [The gates](.tools) have the details.

## Pull requests

Pull requests land on `main` by GitHub's merge commits, as the
[land](https://github.com/zadykian/toolkit/blob/main/skills/land/SKILL.md) skill says. Their signed
commits land as they are, under a merge commit that GitHub signs. The ruleset on `main` allows the
merge method alone, and requires signed commits and branches up to date.

## License

[MIT](LICENSE), but for the Host dashboard in
`roles/server/ubuntu/signoz/internal/dashboards/dashboards`, which is SigNoz's,
under the [Apache License 2.0](roles/server/ubuntu/signoz/internal/dashboards/dashboards/LICENSE).
