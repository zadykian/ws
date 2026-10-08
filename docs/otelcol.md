# otelcol

The `otelcol` role runs the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) on
the host. It takes OTLP on loopback, and reads the journal, the host's metrics and its temperatures
from node_exporter. It sends it all to SigNoz, and claude's metrics to another collector where one
is set.

## The collector

The OpenTelemetry Collector runs on the host as the service `otelcol-contrib`, from the `.deb` of
its release, which `group_vars/all.yml` pins by version and SHA-256. Its config,
`roles/otelcol/templates/config.yaml.j2`, starts from SigNoz's for a collector on a VM. It:

- takes OTLP on `127.0.0.1:4317` (gRPC) and `127.0.0.1:4318` (HTTP), and answers its health check
  on `127.0.0.1:13133`;
- reads the journal at priority info and above through `journalctl`, as a member of
  `systemd-journal`, which a drop-in gives the service. rsyslog's files under `/var/log` repeat the
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
line are dropped: as attributes of the resource, they would make each process a resource of its
own in SigNoz. The collector reads the journal from its end each time it starts. So what the
journal gets while the collector is down stays there alone.

node_exporter is Ubuntu's `prometheus-node-exporter`, without its recommends. It runs with only its
`hwmon` and `thermal_zone` collectors and without its own Go and process metrics. Its arguments are
in `/etc/default/prometheus-node-exporter`, which the role writes before it installs the package,
as the package starts node_exporter at once. dpkg keeps that file and puts the package's beside it,
as `.dpkg-dist`. Its metrics keep their Prometheus names, such as `node_hwmon_temp_celsius`, with
`service.name` `node` and the host's name. A VM has no temperatures or fans to read, so there they
are missing.

The collector refuses new data once it holds 384 MiB, as when SigNoz is down and what it would
send piles up. That way it doesn't grow until the kernel kills another process.

The role checks a new config with `otelcol-contrib validate` before the collector restarts with it.
On a first install, the package starts the collector with the package's own config, which listens
on every address; ufw keeps those ports from outside. The role stops and disables it at once, and
enables it again once the role's config is in place.

## claude's metrics elsewhere

`~/.secrets/env` is the file of secrets the shell reads and this repository never writes. Where it
sets `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT`, the collector sends claude's metrics there as well, over
OTLP/gRPC. The value is a URL, `http://HOST:PORT` for gRPC without TLS or `https://HOST:PORT`. The
role refuses any other. It refuses one that leads back to the collector's own receiver on
`127.0.0.1:4317` too, which would send the metrics around without end. Only the metrics whose
resource has `service.name` `claude-code` go, as claude sends them, without the host's name.
claude's traces and logs, with their tool details, stay in SigNoz.

The role reads that one variable from the last line of the file that sets it, with or without
`export` and quotes, without running the file. It writes it to `/etc/otelcol-contrib/claude.env`,
which only root reads and a drop-in adds to the service's environment. The config names the
variable, never its value, and the run's output leaves the value out. Without the variable, the role
removes that file and the config has no second pipeline. A new value restarts the collector.
