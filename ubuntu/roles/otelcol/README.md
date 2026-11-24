# otelcol

The `otelcol` role runs the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) on
the machine, from `mise bootstrap`. Its `mise.guest.toml` and `mise.host.toml` install
node_exporter. They name the units of its `units.toml.tmpl`: the `.deb`, the service with its
files, and the check that the collector answers. Two more serve Docker's containers: a socket on
the bridge address with its proxy, and a host's ufw rules for it. The collector takes OTLP on
loopback, and from those containers through that socket. It reads the journal, the
host's metrics and its temperatures from node_exporter. It sends it all to SigNoz, and claude's
metrics to another collector where one is set.

## The collector

The OpenTelemetry Collector runs on the host as the service `otelcol-contrib`, from the `.deb` of
its release, which the unit `deb/otelcol` in `units.toml.tmpl` pins by version and, for each architecture,
SHA-256. The unit fetches the file into `/var/cache/ws/` where dpkg lists another version. It
checks the file against the SHA-256, then its name and version against the pin, before dpkg reads
it, and removes the file after. A plan on a new machine shows the install. Its config,
`ubuntu/roles/otelcol/files/config.yaml.tmpl`, starts from SigNoz's for a collector on a
VM. It:

- takes OTLP on `127.0.0.1:4317` (gRPC) and `127.0.0.1:4318` (HTTP), and answers its health check
  on `127.0.0.1:13133`;
- reads the journal at priority info and above through `journalctl`, as a member of
  `systemd-journal`, which a drop-in gives the service. rsyslog's files under `/var/log` repeat the
  journal, so it leaves them out;
- reads the host's metrics every 60 seconds: CPU, load, memory, paging, disks, filesystems,
  network, and the count of processes;
- scrapes temperatures and fans from node_exporter on `127.0.0.1:9100` every 60 seconds, as the
  host's metrics have none;
- sends it all to SigNoz's intake by gRPC on `127.0.0.1:14317`, with the host's name added as
  `host.name`;
- sends claude's metrics to a remote collector too, where `/etc/ws/env` sets
  `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT`: [below](#claudes-metrics-elsewhere).

A journal line's message is its body and its priority its severity; its other fields are
attributes, such as `journald._SYSTEMD_UNIT`, its unit. Its process's ID, executable and command
line are dropped: as attributes of the resource, they would make each process a resource of its
own in SigNoz. The collector reads the journal from its end each time it starts. So what the
journal gets while the collector is down stays there alone.

node_exporter is Ubuntu's `prometheus-node-exporter`, without its recommends. It runs with only its
`hwmon` and `thermal_zone` collectors and without its own Go and process metrics. Its arguments are
in `/etc/default/prometheus-node-exporter`, which mise writes before it installs the package
(`phase = "pre-packages"`), as the package starts node_exporter at once. A change restarts the
service. dpkg keeps that file and puts the package's beside it,
as `.dpkg-dist`. Its metrics keep their Prometheus names, such as `node_hwmon_temp_celsius`, with
`service.name` `node` and the host's name. A VM has no temperatures or fans to read, so there they
are missing.

The collector refuses new data once it holds 384 MiB, as when SigNoz is down and what it would
send piles up. That way it doesn't grow until the kernel kills another process.

The service owns the collector's four files, so a change of any of them, or of two together,
restarts it once. The service checks a new config with `otelcol-contrib validate` before it
replaces the live file. A config that the check refuses leaves the live file alone. The restart
owed waits for the next good run. On a first install, the package starts the collector
with the package's own config, which listens on every address. A guest holds no filter, so it does
so until the service replaces it. A host's ufw denies what comes in meanwhile.
The `.deb`'s unit stops it at once, and the service enables it again once the config is in place.
The `.deb`'s unit waits on signoz's sync, so that SigNoz's retention is set before the collector
sends ([signoz](../signoz/README.md#retention)).

## Docker's containers

Port 4317 is the machine's own collector, the one destination for every sender, and 14317 is
SigNoz's intake by gRPC, which that collector alone uses. A container sends to `4317` on the bridge
address, `WS_DOCKER_BRIDGE_IP`, never straight to SigNoz, so that a machine's telemetry
all passes its collector, under its `host.name`.

The receiver stays on loopback. A second one on the bridge address could not bind at boot, before
`docker0` has the address, and the collector would stop. So the socket unit `otelcol-docker.socket`
listens there, with `FreeBind=yes`, and `systemd-socket-proxyd`, in `otelcol-docker.service`,
passes each connection to `127.0.0.1:4317`. The service requires the socket, so that a new address
restarts both. A host's ufw denies what comes in, so `state/otelcol-docker-ufw` lets `docker0` and
`br-+` reach the port, as docker's `state/docker-dns` does for the resolver. A guest holds no
filter.

Only gRPC goes through. A container whose SDK keeps its defaults sends OTLP/HTTP to port 4318, where
nothing listens, so it sets `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` and
`OTEL_EXPORTER_OTLP_ENDPOINT=http://172.17.0.1:4317`, with the bridge address in place of
`172.17.0.1` where `WS_DOCKER_BRIDGE_IP` moves it.

## claude's metrics elsewhere

`/etc/ws/env` is the file of secrets the shell reads, which mise writes from the `shell` role's
secret inputs ([The env file](../shell/README.md#the-env-file)). mise's files step writes it before
any unit, so the templates read the run's value. Where it sets
`OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT`, the collector sends claude's metrics there as well, over
OTLP/gRPC. A plan writes no file, so under the secrets task the templates take the input from the
run's env, and the plan shows the change that the run makes. A run without the task, whose env may
hold a stale value, reads the file as it stands. The value is a URL, `http://HOST:PORT` for gRPC
without TLS or `https://HOST:PORT`. The role refuses any other. It refuses one that leads back to
the collector's own receiver on `127.0.0.1:4317` or on the bridge address too, which would send
the metrics around without end. Only the metrics whose resource has
`service.name` `claude-code` go, as claude sends them, without the host's name. claude's traces and
logs, with the prompts' and responses' text and the tools' details and output, stay in SigNoz.

The templates read that one variable from the last line of the file that sets it, with or without
`export` and quotes, without running the file. The value goes to `/etc/otelcol-contrib/claude.env`,
which only root reads and a drop-in adds to the service's environment. The config names the
variable, never its value, and the output leaves the value out, also where the check refuses it.
Without the variable, the config has no second pipeline. `claude.env` is removed after the config,
so a run cut short never leaves a config that names a missing variable. A new value restarts the
collector.

The collector answers its health check a few seconds after it starts, at most 60. The unit
`state/otelcol-health` waits for that: a collector that cannot start, such as one whose OTLP port
another process holds, fails the run there.
