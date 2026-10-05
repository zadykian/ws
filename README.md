# ws

Sets up my development server from code. Ansible runs on the machine itself and sets up packages,
SSH by key and a firewall, Docker, the shell, tmux, the helix editor, claude and
[cld](https://github.com/zadykian/cld), and SigNoz with a collector for the machine's telemetry.
Running it again is safe, and `--check --diff` shows how a machine differs from the repository.

The roles land one pull request at a time. Each is a tag of `site.yml`:

- `base`: the packages every machine gets, the locale `en_US.UTF-8`, the timezone UTC,
  unattended-upgrades as Ubuntu ships it, which installs Ubuntu's security updates daily, and a
  swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB. `-e base_swap_size_mb=N`
  sets another size in MiB, and 0 none. The file replaces the installer's `/swap.img`. A run
  stops before it changes the swap where the disk lacks room for the file and 2 GiB more, or
  where swapping an area off would bring back more than half the memory available.
- `security`: an sshd drop-in that lets everyone log in by key only, ufw, and fail2ban for SSH:
  [SSH and the firewall](#ssh-and-the-firewall).
- `shell`: `~/.bashrc` with ble.sh and bash-completion, `PATH` in `~/.profile`, and
  `~/.gitconfig`; see [The shell](#the-shell).
- `tmux`: tmux from Ubuntu's archive, with no config, as cld runs tmux with `-f /dev/null`. Where
  tmux's snap is installed, the role removes it. Where cld's sessions run the snap's tmux, end them
  before a real run: cld then finds `/usr/bin/tmux` first on the `PATH`, and their key bindings
  call `/snap/bin/tmux`, which the removal takes away.
- `devtools`: Node, the .NET SDKs, gh, glow, the libraries headless Chromium needs, Go and rustup.
  Node comes from NodeSource's apt repository, glow from Charm's, and each .NET SDK from Ubuntu's
  archive or, for those it lacks, Launchpad's dotnet/backports PPA. Each run fetches their keys
  again and checks each against the fingerprint the role pins. Go goes in `/usr/local/go` from
  go.dev's tarball, checked against the SHA-256 go.dev lists for it, and is replaced when
  `group_vars/all.yml` pins another version. rustup installs Rust's stable toolchain in the user's
  home; once there, it updates itself, so the playbook leaves it as it is.
- `docker`: Docker Engine with its buildx and compose plugins, from Docker's apt repository, and
  `/etc/docker/daemon.json`, which rotates each container's log at 10 MB and keeps 3 files. A
  container keeps the log settings it was created with. Containers keep running while Docker
  restarts (`live-restore`). `dockerd` and every container run in `docker.slice`, capped at half
  the RAM and half the CPUs, with no swap; `docker_memory_max`, `docker_cpu_quota` and
  `docker_memory_swap_max` set other limits, in systemd's syntax (`16G`, `400%`). A running
  container moves into the slice when it next starts. `containerd`, which pulls and unpacks
  images, stays outside the slice's limits. Where `ws_user` isn't root, the role adds it to the
  `docker` group, so that it runs `docker` without sudo. The group is root-equivalent, as a
  container can mount any of the host's files, and the user has it from its next login.
- `claude`: claude, from its native installer, and cld, from its latest release's `install.sh`,
  both in `~/.local/bin`, and cld's completion in bash. claude updates itself in the background,
  so the role installs it only where it is missing. Where cld is installed, the role updates it
  with `cld update`; a dry run compares its version with the latest release instead.
  Then `cld setup restore`, which has the user's systemd bring cld's sessions back after a
  reboot. The role turns lingering on, so that the user's systemd starts at boot rather than at
  the first login. The unit keeps the `PATH` of the user's login shell, where `cld restore`
  finds tmux; the next run of the role updates it after a change to that `PATH`.
- `signoz`: [SigNoz](https://signoz.io) in Docker, which keeps the machine's telemetry for 90 days,
  with its UI and OTLP/HTTP intake on loopback, no login, and a dashboard of the host:
  [below](#signoz).
- `otelcol`: the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) on the host,
  which takes OTLP on loopback, reads the journal, the host's metrics and its temperatures from
  node_exporter, and sends it all to SigNoz: [below](#the-collector).

## A new machine

Install Ubuntu Server 26.04 by hand from its installer, with an SSH key to log in with. The
installer gives the key to its own user, and the playbook lets root log in by key only, so copy
the key to root first, as the installer's user:

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

## SSH and the firewall

The playbook turns SSH's passwords off: everyone logs in by key, root included. ufw then lets in
SSH on port 22 and nothing else, and fail2ban bans an address for 10 minutes after 5 failed logins
in 10 minutes. Over SSH, keep the session that ran the playbook open until a new one logs in.
The playbook stops before it turns passwords off if root has no key to log in with.

Docker's published ports get past ufw: publish a container's port on `127.0.0.1` and reach it
through an SSH tunnel.

## Running it again

`bootstrap.sh` leaves the clone in `/root/repository/ws` as it is, so pull first:

```sh
git -C /root/repository/ws pull --ff-only
/root/repository/ws/bootstrap.sh
```

Its arguments go to `ansible-playbook`: `--tags NAME` runs the role NAME alone.

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

The playbook's dry run adds no apt repository, so where it would add or change a third-party one,
apt may not know its packages: their install shows that error, and the dry run goes on.

## SigNoz

SigNoz runs in Docker from the compose files in `roles/signoz/files/deployment`, which the role
copies to `/opt/signoz`. Docker's published ports get past ufw, so only two are published, on
loopback: the UI on `127.0.0.1:3301` and the OTLP/HTTP intake on `127.0.0.1:14318`. 3301 was
SigNoz's UI port before 8080, which dev servers often take.

ClickHouse, SigNoz's database, may use 4 GB of memory. Without a limit of its own, it would size
itself to the host's RAM, as it cannot see `docker.slice`'s.

No one logs in: SigNoz serves every request as its root user, and its UI shows a banner that says
so. The role generates the root user's password on the machine, in `/opt/signoz/root.env`, which
only root reads; nothing else needs it. Reach the UI through an SSH tunnel, then open
`http://localhost:3301`:

```sh
ssh -N -L 3301:127.0.0.1:3301 root@SERVER
```

A `LocalForward 3301 127.0.0.1:3301` in the client's `~/.ssh/config` does the same.

SigNoz keeps traces, metrics and logs for 90 days, `signoz_retention_days`; its own default is 15
days, and 30 for metrics. The role sets it through SigNoz's API before anything is sent, as a change
applies only to data that comes in after it, and after SigNoz's schema migrator has finished, as
the migrator sets its own. Settings › General in the UI shows it; rules set there to keep some logs
for another time stay. A dry run reads it and reports what it would change, and skips it where
SigNoz isn't running.

The Host dashboard shows the CPU, memory, disks, file systems and network from the collector's host
metrics, and the temperatures and fans from node_exporter. It is SigNoz's own host metrics
dashboard, changed a little: `roles/signoz/files/dashboards` says how, and holds its license. The
role creates it through SigNoz's API and replaces it where it differs from the repository's, so a
change made to it in the UI lasts until the next run, and one made to a copy of it stays. SigNoz
refuses to change a dashboard locked in the UI, so the run leaves it and says so. A dry run reports
whether it would create or replace the dashboard.

The compose files are generated. SigNoz's [Foundry](https://github.com/SigNoz/foundry) forges them
from `roles/signoz/files/casting.yaml`, which pins every image, and the checksum of a ClickHouse
function the stack downloads from GitHub. To change them, change the casting, then forge again with
foundryctl 0.3.0 and commit what it writes:

```sh
cd roles/signoz/files
rm -rf deployment casting.yaml.lock
foundryctl forge --no-ledger --no-updater -f casting.yaml -p .
```

Without `--no-ledger` and `--no-updater`, foundryctl reports each command to SigNoz and asks GitHub
for a newer release. It never removes a file, hence the `rm`.

A new SigNoz may return the Host dashboard with fields added or dropped, and the role would then
replace it on every run. So after changing SigNoz's image, run the role, refresh `host.json` from
SigNoz as `roles/signoz/files/dashboards/README.md` says, and commit it with the casting. A second
run must then leave the dashboard as it is.

## The collector

The OpenTelemetry Collector runs on the host as the service `otelcol-contrib`, from the `.deb` of
its release, which `group_vars/all.yml` pins by version and SHA-256. Its config,
`roles/otelcol/templates/config.yaml.j2`, starts from SigNoz's for a collector on a VM. It:

- takes OTLP on `127.0.0.1:4317` (gRPC) and `127.0.0.1:4318` (HTTP), and answers its health check
  on `127.0.0.1:13133`;
- reads the journal at priority info and above through `journalctl`, as a member of
  `systemd-journal`, which a drop-in gives the service; rsyslog's files under `/var/log` repeat the
  journal, so it leaves them out;
- reads the host's metrics every 60 seconds: CPU, load, memory, paging, disks, filesystems,
  network, and the count of processes;
- scrapes temperatures and fans from node_exporter on `127.0.0.1:9100` every 60 seconds, as the
  host's metrics have none;
- sends it all to SigNoz's intake on `127.0.0.1:14318`, with the host's name added as `host.name`.

A journal line's message is its body and its priority its severity; its other fields are
attributes, such as `journald._SYSTEMD_UNIT`, its unit. Its process's ID, executable and command
line are dropped, since as attributes of the resource they would make each process a resource of
its own in SigNoz. The collector reads the journal from its end each time it starts, so what is
logged while it is down stays in the journal alone.

node_exporter is Ubuntu's `prometheus-node-exporter`, without its recommends, with only its
`hwmon` and `thermal_zone` collectors and without its own Go and process metrics. Its arguments are
in `/etc/default/prometheus-node-exporter`, which the role writes before it installs the package,
as the package starts node_exporter at once; dpkg keeps that file and puts the package's beside it,
as `.dpkg-dist`. Its metrics keep their Prometheus names, such as `node_hwmon_temp_celsius`, with
`service.name` `node` and the host's name. A VM has no temperatures or fans to read, so there they
are missing.

The collector refuses new data once it holds 384 MiB, as when SigNoz is down and what it would send
piles up, so that it doesn't grow until the kernel kills another process.

The role checks a new config with `otelcol-contrib validate` before the collector restarts with it.
On a first install, the package starts the collector with the package's own config, which listens
on every address; ufw keeps those ports from outside. The role stops and disables it at once, and
enables it again once the role's config is in place.

## By hand

Secrets are never in this repository. On a new machine, these stay manual:

- `gh auth login` and `gh auth setup-git`;
- the claude login.

## The shell

The `shell` role writes `~/.bashrc`, `~/.profile` and `~/.gitconfig`, and each run replaces any
change made to them by hand. A machine's own settings go in files they read where they exist:

- `~/.secrets/env`, which `~/.bashrc` sources first, above its interactive guard, so that
  non-interactive shells get it too;
- `~/.bash_aliases`, which `~/.bashrc` sources too;
- `~/.config/git/local`, which `~/.gitconfig` includes last: a signing key, say.

An installer that appends to `~/.bashrc` or `~/.profile`, a line that adds its directory to
`PATH`, say, loses that line on the next run: move it to `~/.bash_aliases`.

`~/.gitconfig` has gh's credential helper already, as `gh auth setup-git` writes it, so running
that changes nothing. A dry run shows no diff of `~/.gitconfig`, as one set up by hand may hold a
credential. It names instead the keys a real run would drop, without their values: move them to
`~/.config/git/local` first. A subsection shows as `*`, as in `http.*.extraheader`, since one may
be a URL, which names a host. A real run keeps the old file beside the new one, as
`~/.gitconfig.*~`.

## Checks

CI runs yamllint, ansible-lint with its production profile, ShellCheck and shfmt on the shell
scripts, and gitleaks over the whole history. Any finding fails it, warnings included.

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`,
which need a booted machine. It runs it twice, and the second run must change nothing. Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM.

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

The hooks in `.githooks` check each commit before it is made. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath .githooks`, as `bootstrap.sh` does in its clone.
Where `~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose
added lines, file names or message hold one. Where gitleaks is installed, they refuse a staged
secret too.

## Pull requests

Pull requests land on `main` by fast-forward only, as the commits CI checked. GitHub's merge
methods are all refused. A comment `/ff` on a pull request pushes its head to `main`
(`.github/workflows/fast-forward.yml`). A pull request that changes `.github/workflows` is pushed
by hand, `git push origin SHA:main`.

## License

[MIT](LICENSE), but for the Host dashboard in `roles/signoz/files/dashboards`, which is SigNoz's,
under the [Apache License 2.0](roles/signoz/files/dashboards/LICENSE).
