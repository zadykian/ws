# ws

Sets up my development server from code. mise's bootstrap runs on the machine itself, with ws's
units. It sets up packages, SSH by key, and a cable and a WireGuard tunnel to the
Mac. It sets up Docker, the shell, git, tmux, lazygit, the Fresh editor and Zed's server. It sets
up claude and [cld](https://github.com/zadykian/cld), and SigNoz with a collector for the
machine's telemetry. A host, the machine that runs the guests, has its own setup and its own
roles: `ws ubuntu host setup` ([A host](#a-host)).
Running it again is safe, and its plan shows how a machine differs from the repository.

Each role has a page of its own. A role is its mise files by env, and its spec of ws's units where
it has any:

- [`frame`](ubuntu/roles/frame): what each role of mise's config leans on, and
  `/usr/local/bin/ws`, the machine's one ws.
- [`base`](ubuntu/roles/base): packages, the locale, the timezone, security updates and a swap file,
  as mise's packages and units.
- [`security`](ubuntu/roles/security): SSH by key only and fail2ban, as mise's config and units. A
  host sets up ufw too, with SSH in and the rest of what comes in denied. A guest sets up no ufw and
  no table of its own.
- [`secrets`](ubuntu/roles/secrets): fnox, which decrypts ws's secrets, as mise's config, and
  the task `secrets`, which fetches them by the host key and runs the bootstrap under them.
- [`cable`](ubuntu/roles/cable): the host's end of a cable to the Mac, as units.
- [`tunnel`](ubuntu/roles/tunnel): `wg0`, the server's end of a WireGuard tunnel for the Mac, and on
  a host each guest's, as units.
- [`ikeward`](ubuntu/roles/ikeward): ikeward's IKEv2 VPN, where a profile is given, and the WireGuard
  peers' way into it, as units.
- [`nvidia`](ubuntu/roles/nvidia): NVIDIA's headless driver from Ubuntu's archive, on a machine with an
  NVIDIA GPU. A container takes the userspace alone, by its switch, and the host gives it the card.
- [`fans`](ubuntu/roles/fans): fan curves for the host's board's headers, and the driver that sets them.
- [`shell`](ubuntu/roles/shell): `/etc/ws/bashrc` and `/etc/ws/profile`, one line of each at the top of
  `~/.bashrc` and `~/.profile`, and `/etc/ws/env` from mise's secret inputs.
- [`git`](ubuntu/roles/git): `/etc/ws/gitconfig` and its include in `~/.gitconfig`, and a key that
  signs commits. A hook adds a co-author trailer to each, and one checks each push. It sets git's
  tokens for servers besides GitHub, and clones of repositories.
- [`tmux`](ubuntu/roles/tmux): tmux from Ubuntu's archive.
- [`lazygit`](ubuntu/roles/lazygit): lazygit from Ubuntu's archive, in JetBrains Dark, with the
  repository and its branch in the terminal's title.
- [`devtools`](ubuntu/roles/devtools): Node, the .NET SDKs, gh, glow, Chromium's libraries, Go,
  gopls, Delve, roslyn-language-server, the SQL, Docker and npm language servers, and rustup.
- [`docker`](ubuntu/roles/docker): Docker Engine, its `daemon.json` and a slice for its containers.
- [`incus`](ubuntu/roles/incus): Incus from Zabbly's repository on the host, with its ID range, ZFS
  pool, bridge, egress ACL and profile. A table of the guests' sources and ufw's rules, as units. The
  guests of the host's list, from an image with a pinned SHA-256, and `ws ubuntu host guests`.
- [`claude`](ubuntu/roles/claude): claude, its user settings and telemetry, its attribution off, its sessions after a
  reboot, and plugin marketplaces from a list.
- [`signoz`](ubuntu/roles/signoz): SigNoz in Docker, which keeps the machine's telemetry for 90 days.
- [`otelcol`](ubuntu/roles/otelcol): the OpenTelemetry Collector on the host, which feeds SigNoz.
- [`fresh`](ubuntu/roles/fresh): the Fresh editor, from the .deb of a pinned release, with the plain keys,
  JetBrains Dark and the language servers.
- [`zed`](ubuntu/roles/zed): the settings of Zed's server for Go, C#, SQL, Docker, Ansible and HTML, the
  terminals' tmux wrapper, and the open-file limit that it needs.
  [A project's settings](ubuntu/roles/zed/projects.md) cover SQL Server and Ansible.

`macos/` sets up the Mac that works on the server, and Ghostty and Zed on it: [The Mac](macos).
Its [`frame`](macos/roles/frame) is what the Mac's roles of mise's config lean on, and
`/usr/local/bin/ws`. Its [`firewall`](macos/roles/firewall) turns the Application Firewall on
and has it block all incoming. Its [`dotnet`](macos/roles/dotnet) installs .NET's SDKs, as `devtools` does on
the server.

A new Mac with Apple silicon follows [the Mac's runbook](macos/runbook.md). It runs
`install.sh` through `sudo`, which installs ws's binary, checked against its SHA-256, and writes the
values ([The Mac's values](macos/values/README.md)). Then `ws macos setup`
installs the command line tools and mise, checks a release's tree out into `/opt/ws`, links its
plugin where mise finds it, and runs mise's bootstrap ([The Mac](macos/README.md#setup)).
[The server's runbook](ubuntu/runbook.md) lists the server's own steps.

## A new machine

A release's `autoinstall.yaml` answers the installer of a new host, and the first boot runs the
host's setup. Fill its host name, root's hash and the values. Copy it as `user-data` to a volume
labelled `CIDATA`, beside an empty `meta-data`, and boot the installer with the volume:
[The server's runbook](ubuntu/runbook.md#a-new-server-from-the-installer) has the steps. A host
holds no secret. A cloud VM has no installer, and any other server takes the steps by hand.

Install Ubuntu Server 26.04 by hand from its installer. In it, turn OpenSSH on, import the GitHub
account's keys, and leave password login off. ws sets the server up for root. Root's keys come from
the GitHub account alone: the installer imports them into its own user's home, and ws copies no key
from another account's file. So log in as the installer's user, by key, and run `sudo -H
ssh-import-id gh:LOGIN` ([the runbook](ubuntu/runbook.md)). A cloud VM has no installer: its
default user holds the provider's keys, which must be the account's. cloud-init has put the same
keys in root's `authorized_keys` behind a forced command that refuses a login, and ssh-import-id
skips a key that is there. So delete those lines first, then import the keys into root the same way.
A VM with root alone logs in as root, where `WS_TARGET=cloud`. An image with Docker holds Docker's
apt source, which apt refuses beside mise's: remove it first
([docker](ubuntu/roles/docker/README.md)).

Then, logged in as that user, write `/etc/ws/values.yml` where a value differs from its default,
root's and mode 600 ([the values](#running-it-again)). Run `install.sh` through `sudo`, by the
three lines of [Install](.github/workflows/release.md#install), which fail where curl does. It
installs ws alone. Then run `ws ubuntu guest setup` as root, which installs mise and runs its bootstrap,
which sets each role up. Once the secrets
hold `WS_GITHUB_TOKEN`, the secrets task logs gh in ([By hand](#by-hand)). The Mac's
`ws macos secrets add NAME` runs it first, as [the server's runbook](ubuntu/runbook.md) says.

Then do what stays by hand, below.

## A host

A host is the machine that runs the guests. `ws ubuntu host setup` sets it up as `ws ubuntu guest
setup` sets up a guest, on its own run directory, `ubuntu/host`. It runs the roles that hold a
`mise.host.toml`: `frame`, `base`, `security`, `docker`, `nvidia`, `signoz`, `otelcol`, `fans`,
`cable`, `incus` and `tunnel`, which ends its own tunnel and each guest's.
Its values are `/etc/ws/values.yml` too, with its own type: `ws ubuntu host values check` and
`set`. The installer's first boot runs `ws update`, then `ws ubuntu host setup`
([the runbook](ubuntu/runbook.md)).

## Running it again

Each release holds the roles it sets up, so run `ws update` as root. It brings ws to the latest
release, then runs that ws's setup of the machine's side: `ws ubuntu guest setup` on a guest,
`ws ubuntu host setup` on a host. The setup checks the release's tree out and runs the bootstrap.
`--version X.Y.Z` picks a release, and `X.Y.Z-rc.N` a pre-release
([Update](.github/workflows/release.md#update)).

The units read their secrets from mise's secret inputs, which the task `secrets` fetches and
decrypts on ws ([The repository](ubuntu/roles/secrets/README.md#the-repository)). The
values of a machine go in `/etc/ws/values.yml`, which only root reads, and every run reads them.
The first run writes it where none is, each key at its default, and no run changes it after
([The values file](ubuntu/roles/frame/README.md#the-values-file)). The server's user is
root, in `/root`: no value names either. `WS_GIT_SIGNING_PUB_KEY` is `/root/.ssh/id_ed25519_signing.pub`
by default, and lies below `/root`, or the run fails. `ws ubuntu guest
values set KEY VALUE` writes one key, and an empty VALUE gives its default back. `ws ubuntu guest
values check` and each plan refuse a bad value before any write
([The check of the values](ubuntu/roles/frame/README.md#the-check-of-the-values)). A second
run changes nothing, so `ws ubuntu guest setup --plan` exits 0.

## Dry runs

Setup's plan shows what a run would change, and nothing changes. A run by hand takes the form
that [the server's runbook](ubuntu/runbook.md) gives:

```sh
ws ubuntu guest setup --plan
```

`ws ubuntu guest setup --secrets --plan` plans through the task `secrets`, which fetches the secrets
first ([The task](ubuntu/roles/secrets/README.md#the-task)).

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
as `wg0`'s peer stay manual. ws's gh logs in as the machine account `@zadykiand`, by its classic
personal access token, scopes `repo`, `workflow` and `read:org`, which the secrets task's run takes
from the secret input `WS_GITHUB_TOKEN` ([gh](ubuntu/roles/devtools/README.md#gh)). The
signing key goes to `@zadykiand`'s settings, not the maintainer's. These stay manual:

- the machine account `@zadykiand`. Verify `su@zadykian.dev` on it, so that GitHub verifies its
  commits, and turn on its 2FA. Accept its invites to ws, cld and ikeward. Give it no access to the
  secrets repository, which the server reads by its deploy key alone. A `repo` token reaches every
  repository the account can, so its invites are its only limit. Root's login keys stay
  `@zadykian`'s: `ssh-import-id` imports those, not the machine account's;
- ws's signing key off `@zadykian`'s settings, where it stands as a signing key. GitHub would show a
  commit that ws signs under the maintainer's email as verified
  ([The signing key](ubuntu/roles/devtools/README.md#the-signing-key));
- other peers' public keys on the server, as in [The tunnel](ubuntu/roles/tunnel/README.md#the-tunnel);
- another device's SSH key on the server: `ssh-import-id gh:LOGIN`, run as root
  ([SSH](ubuntu/roles/security/README.md#ssh));
- ikeward's VPN profile, in the server's fnox profile, `fnox.HOSTNAME.toml`, and the `ikeward`
  role's variables, as in [The profile](ubuntu/roles/ikeward/README.md#the-profile);
- with Secure Boot on, DKMS's key, enrolled at the console: [Secure Boot](ubuntu/roles/fans/README.md#secure-boot);
- Fresh's trust, once per repository: press T, then Enter
  ([Trust each project](ubuntu/roles/fresh/README.md#trust-each-project));
- the ruleset on every branch of ws, cld and ikeward, after the machine account's stack lands. It
  requires signed commits, so that GitHub refuses an unsigned push from ws
  ([The push check](ubuntu/roles/git/README.md#the-push-check));
- the local list of words: set the secret `FORBIDDEN_WORDS` on GitHub, and place
  `~/.config/ws/forbidden-words`, mode 600, on each machine that commits
  ([The hooks](.github/workflows/README.md#the-hooks)).

The router takes a few steps once, by hand: [The router](ubuntu/roles/cable/router.md).

## Checks

CI runs `make lint`, which lints the code, the docs and the workflows, and scans the commits for
secrets. It runs `ws ubuntu guest setup` twice on a booted runner, and setup's plan must exit 0 after
each run. It tries `install.sh` on Linux and on macOS, forges SigNoz's compose files again, and sets
up a Mac on GitHub's macOS runner by the bootstrap. A pushed tag `vX.Y.Z` publishes the
release whose ws `install.sh` installs, and `vX.Y.Z-rc.N` a pre-release of a `feat/` branch:
[Releases](.github/workflows/release.md). The rules workflow warns where a
pull request breaks the rules of zadykian/toolkit's skills, and the issues workflow where an issue
strays from its issue form. The hooks in `.githooks` check each commit before git makes it.
[Checks](.github/workflows) and [The gates](.tools) have the details. What only a real host and its
guests show is [checked by hand](ubuntu/checks/README.md), with scripts.

## Pull requests

Pull requests land on `main` by GitHub's merge commits, as the
[land](https://github.com/zadykian/toolkit/blob/main/skills/land/SKILL.md) skill says. Their signed
commits land as they are, under a merge commit that GitHub signs. The ruleset on `main` allows the
merge method alone, and requires signed commits and branches up to date.

## License

[MIT](LICENSE), but for the Host dashboard in
`ubuntu/roles/signoz/internal/dashboards/dashboards`, which is SigNoz's,
under the [Apache License 2.0](ubuntu/roles/signoz/internal/dashboards/dashboards/LICENSE).
