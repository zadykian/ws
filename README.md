# ws

Sets up my development server from code. Ansible runs on the machine itself. It sets up packages,
SSH by key and a firewall, a direct link and a WireGuard tunnel to the Mac, Docker, the shell, tmux
and the helix editor. It sets up claude and [cld](https://github.com/zadykian/cld), SigNoz with a
collector for the machine's telemetry, and the backends of Rider and GoLand that JetBrains Gateway
connects to. Running it again is safe, and `--check --diff` shows how a machine differs from the
repository.

Each role is a tag of `site.yml`, with a page of its own:

- [`base`](docs/base.md): packages, the locale, the timezone, security updates and a swap file.
- [`security`](docs/security.md): SSH by key only, ufw and fail2ban.
- [`network`](docs/network.md): the server's end of a cable to the Mac.
- [`wireguard`](docs/wireguard.md): `wg0`, the server's end of a WireGuard tunnel for the Mac.
- [`shell`](docs/shell.md): `~/.bashrc`, `~/.profile` and `~/.gitconfig`.
- [`tmux`](docs/tmux.md): tmux from Ubuntu's archive.
- [`devtools`](docs/devtools.md): Node, the .NET SDKs, gh, glow, Chromium's libraries, Go and rustup.
- [`docker`](docs/docker.md): Docker Engine, its `daemon.json` and a slice for its containers.
- [`claude`](docs/claude.md): claude, its user settings and telemetry, and its sessions after a
  reboot.
- [`signoz`](docs/signoz.md): SigNoz in Docker, which keeps the machine's telemetry for 90 days.
- [`otelcol`](docs/otelcol.md): the OpenTelemetry Collector on the host, which feeds SigNoz.
- [`helix`](docs/helix.md): the helix editor, with the IDEs' keys and language servers for Go and C#.
- [`jetbrains`](docs/jetbrains.md): the backends of Rider and GoLand that JetBrains Gateway runs.

`mac/` sets up the Mac that works on the server: [The Mac](docs/mac.md).

## A new machine

Install Ubuntu Server 26.04 by hand from its installer, with an SSH key to log in with. The
installer gives the key to its own user, and the playbook lets root log in by key only. So copy the
key to root first, as the installer's user:

```sh
sudo install -d -m 700 /root/.ssh
sudo install -m 600 ~/.ssh/authorized_keys /root/.ssh/authorized_keys
```

Then run `bootstrap.sh` as root, after `sudo -i` if you log in as the installer's user:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh
```

It installs ansible-core and git with apt, clones this repository into `/root/repository/ws`,
installs the Ansible collections of `requirements.yml`, and runs `site.yml`. Then do what stays
by hand, below.

## Running it again

`bootstrap.sh` leaves the clone in `/root/repository/ws` as it finds it, so pull first:

```sh
git -C /root/repository/ws pull --ff-only
/root/repository/ws/bootstrap.sh
```

Its arguments go to `ansible-playbook`: `--tags NAME` runs the role NAME alone.
`-e ws_user=NAME` sets up the user NAME, whose account must exist, instead of root, with
`ws_home` following it as `/home/NAME`; `-e ws_home=DIR` sets another home.

## Dry runs

With `--check --diff`, the playbook shows what it would change and changes nothing:

```sh
/root/repository/ws/bootstrap.sh --check --diff
```

On a new machine, that is:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh -s -- --check --diff
```

`bootstrap.sh` still installs ansible-core and git, clones the repository and installs the
collections, as the playbook cannot run without them.

The playbook's dry run adds no apt repository. So where it would add or change a third-party one,
apt may not know its packages: their install shows that error, and the dry run goes on.

## By hand

Secrets are never in this repository. On a new machine, these stay manual:

- `gh auth login` and `gh auth setup-git`;
- the claude login;
- each peer's public key in `/etc/wireguard`, as in [The tunnel](docs/wireguard.md#the-tunnel).

The router takes a few steps once, by hand: [The router](docs/router.md).

## Checks

CI lints the playbook and the shell scripts, and scans the history for secrets. It runs the
playbook twice in a container, forges SigNoz's compose files again, and sets up a Mac on GitHub's
macOS runner. The hooks in `.githooks` check each commit before git makes it.
[Checks](docs/checks.md) has the details.

## Pull requests

Pull requests land on `main` by fast-forward only, as the commits CI checked. GitHub's merge
methods are all refused. A comment `/ff` on a pull request pushes its head to `main`
(`.github/workflows/fast-forward.yml`). A pull request that changes `.github/workflows` is pushed
by hand, `git push origin SHA:main`.

## License

[MIT](LICENSE), but for the Host dashboard in `roles/signoz/files/dashboards`, which is SigNoz's,
under the [Apache License 2.0](roles/signoz/files/dashboards/LICENSE).
