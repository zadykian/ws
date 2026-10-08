# ws

Sets up my development server from code. Ansible runs on the machine itself and sets up packages,
SSH by key and a firewall, a direct link and a WireGuard tunnel to the Mac, Docker, the shell,
tmux, the helix editor, claude and [cld](https://github.com/zadykian/cld), SigNoz with a collector
for the machine's telemetry, and the backends of Rider and GoLand that JetBrains Gateway
connects to.
Running it again is safe, and `--check --diff` shows how a machine differs from the repository.

Each role is a tag of `site.yml`:

- `base`: the packages every machine gets, the locale `en_US.UTF-8`, the timezone UTC,
  unattended-upgrades as Ubuntu ships it, which installs Ubuntu's security updates daily, and a
  swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB. `-e base_swap_size_mb=N`
  sets another size in MiB, and 0 none. The file replaces the installer's `/swap.img`. A run
  stops before it changes the swap where the disk lacks room for the file and 2 GiB more, or
  where swapping an area off would bring back more than half the memory available. The kernel's
  swappiness is 10, not its default 60, so that under memory pressure it frees the page cache in
  preference to swapping memory out. `-e base_swappiness=N` sets another, from 0 to 200, in
  `/etc/sysctl.d/99-swappiness.conf`.
- `security`: an sshd drop-in that lets everyone log in by key only, ufw, and fail2ban for SSH:
  [SSH and the firewall](#ssh-and-the-firewall).
- `network`: the server's end of a cable to the Mac, `10.77.0.1/30` on the port
  `network_direct_link_interface` names, from a netplan file: [The direct link](#the-direct-link).
- `wireguard`: `wg0`, the server's end of a WireGuard tunnel for the Mac, `10.99.0.1/24` on UDP
  port 51820, from networkd's own files, with a key made on the server and each peer's public key
  read from `/etc/wireguard`: [The tunnel](#the-tunnel).
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
  images, stays outside the slice's limits. Containers resolve names as the host does:
  `daemon.json`'s `dns` is the default bridge's address, `172.17.0.1` (`docker_bridge_ip`), where
  systemd-resolved listens too, and ufw lets Docker's networks reach port 53 there through their
  bridges, `docker0` and `br-ID`. Docker takes each new network's subnet, a `/20`, from
  `172.16.0.0/12` (`docker_address_pool`), which holds the default bridge too, so that one source
  range covers them all; a network given a subnet outside it, or a bridge name of its own, gets no
  answers. Docker leaves out of the pool only the subnets of the main routing table's on-link
  routes, so the role stops, before it writes `daemon.json`, on any other route into the pool, such
  as one a VPN keeps in a table of its own, and on any route into the default bridge's subnet. It
  reads the routes as they are during the run: a VPN that is down shows none. A running container
  takes the new `dns` when it next starts. A port published without an address, as in `-p 8080:80`
  or compose's `"8080:80"`, goes on `127.0.0.1` (`docker_publish_ip`) rather than on every address:
  `daemon.json`'s `ip` sets that for the default bridge, and `default-network-opts` for each network
  created afterwards. A network created before keeps publishing on every address until it is created
  again, as by `docker compose down` and `up`, and the default bridge takes `ip` when Docker starts
  with no container running, as after a reboot. BuildKit drops build cache that no build has used
  for 90 days (`docker_build_cache_keep_days`). It collects when Docker starts and after each build,
  so idle cache goes at the next build or restart. As Docker's own policy does, it also drops the
  least used cache, down to 10% of the disk, while the disk has less than 20% free or the cache
  takes more than 80% of it. `docker_build_cache_reserved_space`,
  `docker_build_cache_max_used_space` and `docker_build_cache_min_free_space` set other sizes, in
  `daemon.json`'s format (`50GB`). Cache that an image shares frees no space until the image is
  removed. Where `ws_user` isn't root, the role adds it to the `docker` group, so that it runs
  `docker` without sudo. The group is root-equivalent, as a container can mount any of the host's
  files, and the user has it from its next login.
- `claude`: claude, from its native installer, and cld, from its latest release's `install.sh`,
  both in `~/.local/bin`, and cld's completion in bash. claude updates itself in the background,
  so the role installs it only where it is missing. Where cld is installed, the role updates it
  with `cld update`; a dry run compares its version with the latest release instead.
  Next, `cld setup config user` adds to `~/.claude/settings.json` what it lacks: the permission
  set `claude_permissions` names, `read-only` by default, and cld's choices of model, effort,
  theme, editor mode, auto-compact and update channel; with `claude_notifications`, the channel
  claude notifies on. It keeps every value already there. A dry run runs cld on a copy of the
  file and lists the keys it would add. A cld that predates the command gets it from the role's
  `cld update` first; a dry run, which updates nothing, notes that cld lacks it.
  Then `cld setup restore`, which has the user's systemd bring cld's sessions back after a
  reboot. The role turns lingering on, so that the user's systemd starts at boot rather than at
  the first login. The unit keeps the `PATH` of the user's login shell, where `cld restore`
  finds tmux; the next run of the role updates it after a change to that `PATH`.
  claude sends its metrics, logs and traces, with tool details, to the host's collector on
  `127.0.0.1:4317`. The role sets the keys for that in the `env` of `~/.claude/settings.json`
  and removes each signal's own endpoint and protocol. It leaves every other key as it is, and
  writes the file only where `env` changes. A dry run lists the keys it would set or remove,
  not the file's diff, as the file may hold tokens. It replaces an endpoint already in `env`: to
  keep sending claude's metrics to a remote collector, set its endpoint first as
  `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT` in `~/.secrets/env`, which the `otelcol` role reads.
  claude reads the keys as it starts, so a session started before keeps sending where it did.
- `signoz`: [SigNoz](https://signoz.io) in Docker, which keeps the machine's telemetry for 90 days,
  with its UI and OTLP/HTTP intake on loopback, no login, a dashboard of the host and one of
  claude's usage: [below](#signoz).
- `otelcol`: the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) on the host,
  which takes OTLP on loopback, reads the journal, the host's metrics and its temperatures from
  node_exporter, and sends it all to SigNoz, and claude's metrics to another collector where one is
  set: [below](#the-collector).
- `helix`: the helix editor, `hx`, from Ubuntu's archive, with `~/.config/helix/config.toml` set
  close to GoLand's and Rider's settings, and `languages.toml`. Ubuntu's hx ships no tree-sitter
  grammars, so the role fetches and builds those `helix_grammars` lists with `hx --grammar`. Go's
  language server, gopls, comes from `go install` at the version `group_vars/all.yml` pins, with
  the Go of the `devtools` role, and is built again when either version changes. C#'s,
  roslyn-language-server, the server of VS Code's C# extension, comes from nuget.org with
  `dotnet tool` at the version pinned there too, and runs on the .NET 10 of the `devtools` role:
  [C# in helix](#c-in-helix), below. Each run replaces changes made to the two files by hand, and
  keeps the old file beside the new one. The keys follow the IDEs' F12 keymap:
  [helix's keys](#helixs-keys), below.
- `jetbrains`: the backends of Rider and GoLand, at their latest releases, where JetBrains Gateway
  looks for them, so that it finds them ready rather than downloading its own, and `libicu78`,
  without which Rider's hangs. Each run installs a new release and removes the builds it
  replaces, and a weekly timer does the same between runs:
  [JetBrains backends](#jetbrains-backends), below.

`mac/` sets up the Mac that works on the server: [The Mac](#the-mac), below.

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
SSH on port 22, WireGuard on UDP port 51820 (see `wireguard`), and DNS lookups from Docker's
bridges to the host's resolver (see `docker`), and nothing else; fail2ban bans an address for 10
minutes after 5 failed logins in 10 minutes. Over SSH, keep the session that ran the playbook open
until a new one logs in.
The playbook stops before it turns passwords off if root has no key to log in with.

Docker's published ports get past ufw, so Docker publishes a port that names no address on
`127.0.0.1` (see `docker`): reach it through an SSH tunnel. A port published on another address, as
in `-p 0.0.0.0:8080:80`, is open to every network the host is on.

## The direct link

The Mac can plug into the server with a cable of its own, from its dock's 2.5GbE port to a port of
the server's, outside the router. Name the server's port in `group_vars/all.yml`, as `ip link`
names it:

```yaml
network_direct_link_interface: enp5s0
```

The `network` role then writes `/etc/netplan/60-direct-link.yaml`, which gives that interface
`10.77.0.1/30` (`network_direct_link_address`), and no DHCP, gateway or DNS. netplan merges the
file with any other entry for the port, as the installer's: its settings win, and its address
joins theirs. Boot doesn't wait for the link, which is up only while the Mac is docked. The Mac
takes `10.77.0.2`, with the mask `255.255.255.252` and no router, in its adapter's settings, by
hand. ufw lets in SSH over the cable, as over any interface.

The run stops where the interface carries the default route, where netplan hands it to
NetworkManager rather than networkd, as on Ubuntu Desktop, and where systemd-networkd isn't
running. A machine without the interface, as a cloud VM or CI's container, is left as it is, and
the run says so. With the variable empty, as by default, the role removes the file, and the
address goes.

A change takes effect through `netplan generate`, then `networkctl reload`. netplan writes
networkd's files anew for every interface it sets up, so networkd sets up each of those links
again, the uplink too: the journal shows `Reconfiguring with …` for each. The uplink keeps its
address: networkd asks its DHCP server for the same lease again, and leaves the address on the link
meanwhile.

## The tunnel

The Mac reaches the server through a WireGuard tunnel whichever way it is connected: over the
cable, the home LAN, or the internet through the router's forward (see [By hand](#by-hand)).
Inside the tunnel the server keeps one address, and WireGuard knows a peer by its key, not by the
address its packets come from, so the Mac's sessions outlive a move from one way to another. The
`wireguard` role sets up the server's end, `wg0`, at `10.99.0.1/24` (`wireguard_address`) on UDP
port 51820 (`wireguard_port`), from networkd's own files, `/etc/systemd/network/60-wg0.netdev` and
`60-wg0.network`. Boot doesn't wait for `wg0`.

The first run makes the server's private key on the server, in
`/etc/systemd/network/60-wg0.key`, which the playbook never reads. networkd reads it as the user
`systemd-network`, so the file is `root:systemd-network`, mode 0640, as systemd.netdev(5) asks.
The role's last task shows the public key, which the Mac's end of the tunnel needs: last of all in
`--tags wireguard`, among the other roles' output in a full run. `wg show wg0 public-key` shows it
too. A dry run makes no key.

`wireguard_peers` names the peers, each with its address in the tunnel: by default the Mac, `mac`,
at `10.99.0.2`. A peer's public key is read from `/etc/wireguard/NAME.pub` on the server, a line as
`wg pubkey` prints it, in a directory only root reads:

```sh
printf '%s\n' PUBLIC_KEY >/etc/wireguard/mac.pub
/root/repository/ws/bootstrap.sh --tags wireguard
```

The repository holds no peer's key, as access to a machine is granted on the machine, as SSH's
`authorized_keys` is: anyone may run `bootstrap.sh`, and a key in the repository would let its peer
into every machine set up from it. A peer without its file is left out of `wg0`, and the run names
it. The run stops on a peer's name of other than lowercase letters, digits and dashes, on an
address outside the subnet, the server's own or another peer's, and on a file that holds no key.

A change reaches `wg0` through `networkctl reload`, which sets up again only the links whose files
changed: the uplink and the cable are left alone. A change of peers ends every session of the
tunnel, as networkd replaces all of `wg0`'s peers at once, and the Mac handshakes again within 15
seconds of sending. So change the peers over SSH outside the tunnel, or from the console. A key made
anew, after the old one was removed by hand, deletes `wg0` before the reload, as networkd reads the
key only as it creates `wg0`.

ufw lets in UDP port 51820 on every interface. Inside the tunnel it takes `wg0` as any other
interface: it lets in SSH and ping, and forwards nothing, so the tunnel reaches the server alone.
The run stops where systemd-networkd isn't running, as on Ubuntu Desktop, whose NetworkManager
does its work instead.

## Running it again

`bootstrap.sh` leaves the clone in `/root/repository/ws` as it is, so pull first:

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
dashboard, changed a little: `roles/signoz/files/dashboards` says how, and holds its license.

The Claude dashboard shows what claude on the server cost and what it did, from the metrics and
events claude sends to the collector (see `claude`): cost and tokens by model and by day, the share
of tokens read from the prompt cache, sessions, active time, lines of code, commits, pull requests,
edit decisions, prompts, tool calls and API errors, over the range of the UI's time picker. Its
bars are days in UTC, the server's timezone. Cost is claude's estimate at API prices: under a
subscription, it is not what was billed. No panel shows a prompt's text, a tool's input or a
command, and claude sends no prompt text, as `OTEL_LOG_USER_PROMPTS` is unset. It was written for
this repository, and `roles/signoz/files/dashboards` says what it reads.

The role creates each dashboard of `signoz_dashboards` through SigNoz's API and replaces it where it
differs from the repository's, so a change made to it in the UI lasts until the next run, and one
made to a copy of it stays. A name taken off the list leaves its dashboard in SigNoz. SigNoz refuses
to change a dashboard locked in the UI, so the run leaves it and says so. A dry run reports whether
it would create or replace each dashboard.

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

A new SigNoz may return the dashboards with fields added or dropped, and the role would then replace
them on every run. So after changing SigNoz's image, run the role, refresh `host.json` and
`claude.json` from SigNoz as `roles/signoz/files/dashboards/README.md` says, and commit them with
the casting. A second run must then leave both dashboards as they are.

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
- sends it all to SigNoz's intake on `127.0.0.1:14318`, with the host's name added as `host.name`;
- sends claude's metrics to a remote collector too, where `~/.secrets/env` sets
  `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT`: [below](#claudes-metrics-elsewhere).

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

### claude's metrics elsewhere

Where `~/.secrets/env`, the file of secrets the shell reads and this repository never writes, sets
`OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT`, the collector sends claude's metrics there as well, over
OTLP/gRPC. The value is a URL, `http://HOST:PORT` for gRPC without TLS or `https://HOST:PORT`; the
role refuses any other, and one that leads back to the collector's own receiver on `127.0.0.1:4317`,
which would send the metrics around without end. Only the metrics whose resource has `service.name`
`claude-code` go, as claude sends them, without the host's name. claude's traces and logs, with
their tool details, stay in SigNoz.

The role reads that one variable from the last line of the file that sets it, with or without
`export` and quotes, without running the file. It writes it to `/etc/otelcol-contrib/claude.env`,
which only root reads and a drop-in adds to the service's environment. The config names the
variable, never its value, and the run's output leaves the value out. Without the variable, the role
removes that file and the config has no second pipeline. A new value restarts the collector.

## By hand

Secrets are never in this repository. On a new machine, these stay manual:

- `gh auth login` and `gh auth setup-git`;
- the claude login;
- each peer's public key in `/etc/wireguard`, as in [The tunnel](#the-tunnel).

The router, a MikroTik on RouterOS 7, takes these steps once, each in Safe Mode, WinBox's button or
Ctrl+X in its terminal, which undoes the changes if the session drops:

- The LAN on `10.88.0.0/24`, clear of the cable's `10.77.0.0/30`, the tunnel's `10.99.0.0/24` and
  Docker's `172.16.0.0/12`, and off RouterOS's default, `192.168.88.0/24`, which cafés and other
  homes use too. From WinBox connected by MAC address, which the change doesn't drop, once
  `/export` has shown the names these commands find:

  ```
  /ip address set [find address="192.168.88.1/24"] address=10.88.0.1/24
  /ip pool set default-dhcp ranges=10.88.0.100-10.88.0.254
  /ip dhcp-server network set [find address="192.168.88.0/24"] address=10.88.0.0/24 \
      gateway=10.88.0.1 dns-server=10.88.0.1
  /ip dns static set [find name=router.lan] address=10.88.0.1
  ```

  Clients move to the new range as they next renew their lease.
- The server's lease, made static at `10.88.0.10`, HOST being the host name its lease shows:

  ```
  /ip dhcp-server lease make-static [find host-name=HOST]
  /ip dhcp-server lease set [find host-name=HOST] address=10.88.0.10
  ```

- Two forwards from the internet to the server: UDP 51820, the tunnel's, and TCP 22022 to its SSH,
  for networks that block UDP. The default firewall's rule that drops what comes in from the WAN
  without a dst-nat lets both through:

  ```
  /ip firewall nat add chain=dstnat in-interface-list=WAN protocol=udp dst-port=51820 \
      action=dst-nat to-addresses=10.88.0.10 to-ports=51820
  /ip firewall nat add chain=dstnat in-interface-list=WAN protocol=tcp dst-port=22022 \
      action=dst-nat to-addresses=10.88.0.10 to-ports=22
  ```

- A name for the router's public address: `/ip cloud set ddns-enabled=yes`, as `auto`, the default
  since RouterOS 7.17, enables it only with Back To Home. `/ip cloud print` shows its `dns-name`,
  `SERIAL.sn.mynetname.net`, for a CNAME at your domain's DNS; its records live 60 seconds. Its
  `public-address` must be the WAN's own: behind the ISP's NAT, neither forward reaches the router.

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

## helix's keys

`config.toml` binds the keys of F12, the keymap GoLand and Rider use here, in normal and in
insert mode, for each F12 action helix has a command for, and names F12's action beside each.
[docs/helix-keymap.md](docs/helix-keymap.md) lists every F12 action with its keys and helix
command, and helix's own key for it, then the actions helix has no command for. Cmd never reaches
a program in a terminal, so F12's Ctrl keys stand in for it. On a Mac, the Alt keys need Option to
act as Meta or send Esc+.

Where F12 takes a key of helix's, helix's other key for it still works. A few have no other key,
which the keymap's page lists. Alt+D was helix's only key to delete without yanking: Delete now
does the same.

From normal and insert mode, Move line goes through register `m`, so the `"` register keeps the
last yank, and a move is one undo step. A move off or onto a last line that has no final newline
joins two lines, as `p` and `P` do there: `u` undoes it, and a `:w` adds the newline.

A terminal that sends keys as older terminals did loses some: Shift+Ctrl with a letter arrives as
Ctrl alone, so that Shift+Ctrl+W extends the selection and Shift+Ctrl+Z undoes. Ctrl+- and Ctrl+=
don't arrive, and Ctrl+I, Ctrl+M and Ctrl+[ arrive as Tab, Enter and Esc. One that reports
modified keys as `CSI u`, or answers helix's request for the kitty keyboard protocol, sends all of
these. Shift+Ctrl with a letter arrives whole only where the terminal sends the shifted letter, as
`CSI 84;6u` for Shift+Ctrl+T, or answers the kitty request and reports the shifted key too, as
`CSI 116:84;6u`. Where it sends the letter alone, `CSI 116;6u`, helix drops the Shift and runs
Ctrl+T.

tmux 3.6a sends Shift+Ctrl with a letter as Ctrl alone whatever the terminal: it sends such keys
whole only to a program that asks, and helix doesn't. Ctrl+- and Ctrl+= reach helix through tmux
with these server options. Neither tmux's defaults nor cld's servers have them; for a tmux of your
own, they go in `~/.tmux.conf`:

```
set -s extended-keys always
set -s extended-keys-format csi-u
set -as terminal-features 'xterm*:extkeys'
```

The last tells tmux that a terminal whose `TERM` starts with `xterm` reports modified keys; tmux
knows it of iTerm2 already. With `always`, tmux sends those keys as `CSI u` to every program, a
shell too. tmux's prefix reaches helix pressed twice: Ctrl+B, Toggle line breakpoint, by default,
and Ctrl+Q, Go to action, in cld's sessions.

Where an F12 key doesn't arrive, helix's own key, in the keymap's page, does the same.

## C# in helix

hx gives roslyn-language-server the root of the git repository hx started in. There the server
loads the one `.sln` or `.slnx`, or else every project below the root; `dotnet.defaultSolution` in
the root's `.vscode/settings.json` picks one of several solutions. Started outside a git
repository, hx gives the server no root, and the server takes each file as a program of its own.

Ubuntu's hx asks the server for a file's errors when it opens the file and when the file changes,
not when the server has loaded the projects. A file opened while they load shows its errors after
its first edit, even one undone with `u`.

## JetBrains backends

The Mac opens Rider and GoLand on the server through JetBrains Gateway, which runs each IDE's
backend here. Left to itself, Gateway downloads a backend on the first connect, 2.4 GB for Rider
and 1.2 GB for GoLand, and never updates it. So each run of the role asks JetBrains' list of
releases, which Gateway reads too, for the latest release of each, and installs it where it is
missing, checked against the SHA-256 JetBrains publishes for it.

A build goes where Gateway would put its own download, in `~/.cache/JetBrains/RemoteDev/dist`, in
a directory named as Gateway names it: the first 13 hex digits of the SHA-256 of the download's
link, then the archive's name, as in `46547c76f522d_goland-2026.2.3`. The role writes
`.expandSucceeded` there last, as Gateway does once a build is unpacked. Gateway then finds the
build, lists it among the installed IDEs, and downloads nothing. Neither the name nor the file is
documented: they are Gateway 2026.2.2's, and a later Gateway may change them.

Gateway keeps each recent project on the build it was opened with, and shows "Project IDE is
deleted" once that build is gone: "Select Different IDE…" picks the new one, with no download. The
JetBrains Client on the Mac follows the backend's version, and Gateway fetches it as needed.
Settings, plugins and caches are kept per major version, as in `~/.config/JetBrains/Rider2026.2`,
so a 2026.2.x update keeps them.

Once the new build is in place, the role removes every other build of that product from the
directory, Gateway's own downloads too; other IDEs' stay. A build a process runs from, such as a
backend open in Gateway, stays, and the run names it; the next run that finds none removes it.
`jetbrains_prune: false` keeps every build.

Rider's backend takes 6.6 GB unpacked and GoLand's 3.5 GB. A run downloads an archive only where
the disk has 4 times its size free, as JetBrains asks, and removes it once unpacked.
`jetbrains_backends` names the backends by JetBrains' product codes, `[RD, GO]`, and
`-e '{"jetbrains_backends": []}'` installs none. A dry run reads the releases and the directory,
and reports each build it would install and each it would remove; it downloads nothing.

Between runs of the playbook, `jetbrains-update.timer` runs the role once a week, in the hour after
Monday's midnight, or at boot where the machine was off then. Its service applies the role as the
last run of the playbook left it: each run copies the role, and a playbook with that run's
variables, into `/usr/local/lib/ws/jetbrains`, so that a change in the clone reaches the timer
with the next run alone. It runs `/usr/bin/ansible-playbook`, from Ubuntu's ansible-core, which
the role installs where bootstrap.sh hasn't. Its output goes to the journal, and so to SigNoz.
`systemctl list-timers jetbrains-update.timer` shows when it runs next, `journalctl -u
jetbrains-update` what it did, and `systemctl start jetbrains-update` runs it now. A backend open
in Gateway as it runs keeps its build until a later run.

## The Mac

The Mac reaches the server, ws, over the cable from the 2.5GbE port of the monitor it is docked at,
over the home LAN, or from elsewhere through the router's port forwards. `mac/setup.sh`, the one
part of the repository that runs on the Mac, sets up two hosts for ssh:

- `ws`, through a WireGuard tunnel that ends on ws, at `10.99.0.1` whichever way the Mac is
  connected, so that the sessions of `ssh ws` and JetBrains Gateway stay open while the Mac moves
  between the cable, the LAN and elsewhere, with a pause of a few seconds;
- `ws-ssh`, plain SSH over whichever of the three the Mac is on, for networks that block the
  tunnel's UDP.

It is POSIX sh, as the Mac has no Ansible, and runs as the user, from a clone, with sudo where it
needs root. It installs WireGuard's tools from Homebrew, which is [installed](https://brew.sh) by
hand first:

```sh
git clone https://github.com/zadykian/ws ~/repository/ws
~/repository/ws/mac/setup.sh
```

The repository names no server, so the Mac's values go in `~/.config/ws/mac.conf`, `KEY=value`
lines that the script reads without running them:

```sh
WS_PUBLIC_KEY=WS_KEY
WS_LAN=10.88.0.0/24
WS_LAN_ADDRESS=10.88.0.10
WS_REMOTE=ws.example.com
WS_REMOTE_PORT=22022
```

`WS_PUBLIC_KEY` is ws's WireGuard public key, which the `wireguard` role prints:
[The Mac's tunnel](#the-macs-tunnel), below. `WS_LAN` is the home LAN, and `WS_LAN_ADDRESS` ws's
address on it, the router's static lease for it. `WS_REMOTE` is the router's name on the
internet, and `WS_REMOTE_PORT` the port it forwards to ws's 22. The tunnel takes UDP 51820 every
way, which the router forwards too; `WS_PORT` sets another. A way without its values is left out.
The router's side is under [By hand](#by-hand).

The script writes `~/.ssh/config.d/ws`. Its host `ws` goes to `10.99.0.1` from every network;
where UDP is blocked, `ssh ws` times out, and `ssh ws-ssh` serves. `ws-ssh` goes:

- over the cable, to `10.77.0.1`, where one of the Mac's addresses lies in `10.77.0.0/30`;
- over the LAN, to `WS_LAN_ADDRESS`, where one lies in `WS_LAN`;
- elsewhere, to `WS_REMOTE` on `WS_REMOTE_PORT`, over IPv4, as the forward takes IPv4 alone.

ssh picks the way from the Mac's own addresses (`Match localnetwork`, OpenSSH 9.4 and later), and
probes nothing. Both hosts log in as root with `~/.ssh/id_ed25519`, its passphrase kept in the
Keychain, and check ws's host key under the one name `ws` (`HostKeyAlias`). So `known_hosts` has
one entry for ws whichever way ssh goes, and a host on another network that uses the LAN's subnet
meets a host key mismatch, where ssh stops. A session ends after 90 s without word from ws, far
longer than a switch of the tunnel takes.

The script puts `Include config.d/*` first in `~/.ssh/config`, as ssh keeps the first value it reads
for each key. Where the line is missing, it adds it at the top and keeps the old file as
`~/.ssh/config.TIME~`; where the line sits below other settings, the script stops. It warns of a
`Host` or `Match` of the file's own for `ws` or `ws-ssh`. A run changes only what differs, so a
second prints nothing, and `--check` prints each change as a diff and makes none.

The Mac's end of the cable is set by hand, once: in System Settings › Network, the monitor's
adapter › Details › TCP/IP, Configure IPv4 Manually, with the IP address `10.77.0.2`, the subnet
mask `255.255.255.252` and no router.

Through the forward, ws takes logins from the internet: by key alone, as everywhere, with fail2ban
banning the address of the network the Mac is on after 5 failed ones. Over the tunnel, ws sees
the Mac as `10.99.0.2`, which a ban would shut out of `ssh ws` for 10 minutes, while `ws-ssh` comes
from another address.

The first login asks whether to trust ws's host key: compare its fingerprint with the one
`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` prints on ws's console. On the cable and the
LAN, macOS asks once whether iTerm2, Gateway or another app may find devices on the local network,
for `ws-ssh`: allow it. ssh run from Terminal needs no such grant, and neither does `ws`: the
tunnel's utun is no local network, and macOS lets daemons that run as root reach one anyway
(Apple's TN3179). To see which way `ws-ssh` went:

```sh
ssh -v ws-ssh exit 2>&1 | grep 'Connecting to'
```

### The Mac's tunnel

The keys come first, in this order. A run of the `wireguard` role makes ws's key and prints its
public key. `setup.sh`, without `WS_PUBLIC_KEY`, makes the Mac's key, which only root reads, and
prints its public key with the two commands that make the Mac ws's peer, over `ws-ssh`:

```sh
printf '%s\n' 'MAC_PUBLIC_KEY' | ssh ws-ssh 'cat >/etc/wireguard/mac.pub'
ssh ws-ssh /root/repository/ws/bootstrap.sh --tags wireguard
```

With ws's key as `WS_PUBLIC_KEY`, `setup.sh` again installs the tunnel's daemon. A new Mac or a
rebuilt server changes one file on each side, and nothing in the repository.

The daemon, `ws-tunnel`, runs as root from launchd, from boot and before any login, through
`/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist`. It runs root's copies of Homebrew's
`wg` and `wireguard-go`, which sit beside it in `/usr/local/libexec/ws-tunnel`: the user owns
Homebrew's prefix, so a root daemon that ran Homebrew's files would let anything that runs as the
user become root. A `brew upgrade` reaches the copies at the next run of `setup.sh`, which restarts
the daemon where a copy changed. Its settings are in `/etc/ws-tunnel`, which only root reads: the
Mac's key, `tunnel.conf`, which `setup.sh` copies from `mac.conf` so that the daemon reads no file
the user can write, and `routes.d`, below. macOS lists the daemon under System Settings › General
› Login Items & Extensions, where "Allow in the Background" must stay on.

The tunnel's end on the Mac is `10.99.0.2`, on a utun of wireguard-go's. The daemon points it at
the first of these that answers, each on UDP 51820:

- the cable, `10.77.0.1`, where one of the Mac's addresses lies in `10.77.0.0/30`;
- the LAN, `WS_LAN_ADDRESS`, where one lies in `WS_LAN`;
- `WS_REMOTE`, at its first IPv4 address, as the router forwards IPv4 alone.

A switch is proven by a handshake: the daemon removes ws's peer and adds it again with the new
endpoint, so that wireguard-go handshakes at once, and keeps the way where the handshake completes
within 5 s, or tries the next. ws follows the Mac to its new address from that handshake, and TCP
inside the tunnel sends again what the pause lost. The daemon tries the ways again 2 s after macOS
posts a network change, where the Mac's addresses changed, and checks them every 10 s besides, in
case it missed one. It pings `10.99.0.1` every 10 s too, and three pings missed in a row have it
try the ways again, as after the home's address changed. While none answers, it tries them all
every 10 s. A way that comes back while the Mac's addresses stay as they were is taken at the next
change or failure.

```sh
sudo /usr/local/libexec/ws-tunnel/ws-tunnel status
```

prints the way in use, its endpoint, the latest handshake and the bytes moved.
`/var/log/ws-tunnel.log` has a line for each switch and failure.

JetBrains Gateway reads `~/.ssh/config` through the Mac's `ssh -G`, its default, with "Parse
config file ~/.ssh/config" on in the SSH connection's settings. A connection to `ws` outlives a
switch; a second, to `ws-ssh`, serves where UDP is blocked.

The tunnel's end on the Mac runs in userspace, and wireguard-go on macOS moves one packet a read and
a write, so the tunnel may fall short of the cable's 2.5 Gbit/s. To measure it, on ws, install
iperf3 from apt, let its port in on the tunnel and the cable for the test alone, and run the
server:

```sh
ufw allow in on wg0 to any port 5201 proto tcp
ufw allow in on CABLE_INTERFACE to any port 5201 proto tcp
iperf3 -s
```

then, on the Mac, compare `iperf3 -c 10.99.0.1` with `iperf3 -c 10.77.0.1`, each with `-R` too, and
delete the two rules after. Bulk copies can go to `ws-ssh`, outside the tunnel, which is the cable
when the Mac is docked: `rsync -a DIR ws-ssh:`. Besides, check `ws-tunnel status` docked, on the
home Wi-Fi and on a phone's hotspot, and `ethtool CABLE_INTERFACE` on ws, for 2500Mb/s.

### Routes and resolvers of another repository

A private repository's own script for the Mac can send more networks through the tunnel, and a
domain's names to nameservers behind it. It sources this clone's `mac/lib.sh`, which changes
nothing as it is sourced, sets `WS_CHECK` for its own `--check`, and calls:

- `ws_tunnel_routes NAME CIDR...`, which writes the IPv4 networks to
  `/etc/ws-tunnel/routes.d/NAME` and has the daemon read them again. The daemon adds them to the
  peer's AllowedIPs and routes them into its utun, without closing a connection, and leaves out
  for the time being one that holds the endpoint in use. `NAME` is of `a-z`, `0-9` and `-`, and a
  network that overlaps the tunnel's `10.99.0.0/24` or the cable's `10.77.0.0/30`, as `0.0.0.0/0`
  does, is refused. As in `ws_tunnel_routes example 198.51.100.0/24`.
- `ws_resolver DOMAIN ADDRESS...`, which writes `/etc/resolver/DOMAIN`, with up to 3 nameservers,
  so that macOS asks them for the domain's names (resolver(5)), and takes the file at once.
- `ws_tunnel_loaded`, which says whether the daemon is loaded, so that the script can stop with
  "run mac/setup.sh first".

With no network or address, each removes its file, `ws_resolver` only a file it wrote. Under
`WS_CHECK` they print diffs and change nothing. The files outlive the daemon: while it is stopped,
the routes go with its utun, and the domain's lookups go out by the Mac's own network and fail.
ws's firewall forwards nothing, so a network behind ws needs its side set up on ws too.

## Checks

CI runs yamllint, ansible-lint with its production profile, ShellCheck and shfmt on the shell
scripts, and gitleaks over the whole history. Any finding fails it, warnings included.

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`,
which need a booted machine. It runs it twice, and the second run must change nothing. Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM, and it installs no
JetBrains backend, `jetbrains_backends: []`, as it has no room for those either.

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

CI also sets up a Mac, on GitHub's macOS runner, in [The Mac](#the-mac)'s order: `mac/setup.sh`
without ws's key must make the Mac's key and load no daemon, and with a stand-in's key must load
it; then a second run and `--check` must print nothing, and `ssh -G` must give `ws` and `ws-ssh`
their addresses. The stand-in for ws is a second wireguard-go on the runner, with the Mac's key as
its peer, which answers on loopback both as the LAN and as `WS_REMOTE`, and `ws-tunnel status`
must show a handshake on each in turn. Last, a network that `ws_tunnel_routes` adds must be routed
into the daemon's utun, and a domain `ws_resolver` adds must show in `scutil --dns`. Where the
runner can make no utun, the steps of the tunnel are skipped, with a warning.

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
