# signoz

The `signoz` role, mise's config and ws's units in `roles/server/ubuntu/signoz`, runs
[SigNoz](https://signoz.io) in Docker, which keeps the machine's telemetry for 90 days. Its UI and
its OTLP/HTTP intake listen on loopback, with no login. It has a dashboard of the host and one of
claude's usage.

## The stack

SigNoz runs on the system's Docker daemon from the compose files in
`roles/server/ubuntu/signoz/deployment`, which mise copies to `/opt/signoz` (`mise.server.toml`).
Docker's published ports get past ufw, so only two are published, on loopback: the UI on
`127.0.0.1:3301` and the OTLP/HTTP intake on `127.0.0.1:14318`. 3301 was SigNoz's UI port before
8080, which dev servers often take.

ClickHouse, SigNoz's database, may use 4 GB of memory. Without a limit of its own, it would size
itself to the host's RAM, as it cannot see the limit of `docker.slice`.

No one logs in: SigNoz serves every request as its root user, and its UI shows a banner that says
so. The role draws the root user's password on the machine, once, in `/opt/signoz/root.env`,
which only root reads; nothing else needs it. Reach the UI through an SSH tunnel, then open
`http://localhost:3301`:

```sh
ssh -N -L 3301:127.0.0.1:3301 root@SERVER
```

A `LocalForward 3301 127.0.0.1:3301` in the client's `~/.ssh/config` does the same.

mise copies the files first, `/etc/ws/env` among them. Then ws's units (`units.toml.tmpl`) run in
this order, after Docker's and the frame's `state/ws-binary`. otelcol's units, which read
`/etc/ws/env`, hang below the sync:

- `state/signoz-password` runs `ws server signoz password` where its `--check` fails: where
  `root.env` is missing, or has another owner or mode.
- `compose/signoz`, of the kind `compose`, ups the project. It keeps the SHA-256 of the files in
  `/opt/signoz`, `root.env` included, in `/var/lib/ws/compose/signoz`. Where they differ from the
  last good up, as after a new password or a forge, it recreates every container. A container
  would not see a bind-mounted file's change otherwise. Otherwise it ups only where a service that
  keeps running is down, as an up runs the schema migrator again.
- `state/signoz-sync` runs `ws server signoz sync` where its `--check` fails: it waits for
  the API, then sets the retention and the dashboards.

Without systemd booted, as in CI's container, the three skip and say so, and `root.env` is not
written. otelcol's units wait on the sync.

## Retention

SigNoz keeps traces, metrics and logs for 90 days, `WS_SIGNOZ_RETENTION_DAYS`; its own default is 15
days, and 30 for metrics. The role sets it through SigNoz's API before anything is sent, as a change
applies only to data that comes in after it. It sets it after SigNoz's schema migrator has finished
too, as the migrator sets its own. Settings › General in the UI shows it; rules set there to keep
some logs for another time stay. `ws server signoz sync --check` reads it and reports
what it would change, and skips it where SigNoz isn't running.

## The dashboards

The Host dashboard shows the CPU, memory, disks, file systems and network from the collector's host
metrics, and the temperatures and fans from node_exporter. This is SigNoz's own host metrics
dashboard, changed a little: `roles/server/ubuntu/signoz/internal/dashboards/dashboards`
says how, and holds its license.

The Claude dashboard shows what claude on the server cost and what it did, from the metrics and
events claude sends to the collector ([claude](../claude/README.md#telemetry)). It shows cost and tokens by
model and by day, and the share of tokens read from the prompt cache. It shows sessions, active
time, lines of code, commits, pull requests, edit decisions, prompts, tool calls and API errors. Its
totals cover the range of the UI's time picker. Its bars are days in UTC, the server's timezone, and
a metric's first bar can count time before the picker's start. Cost is claude's estimate at API
prices: under a subscription, that is not what was billed. No panel shows a prompt's or a response's
text, a tool's input or output, or a command. claude sends them all, and they sit in SigNoz's logs
and traces ([claude](../claude/README.md#telemetry)). The dashboard was written for this repository, and
`roles/server/ubuntu/signoz/internal/dashboards/dashboards` says what it reads.

The role creates each dashboard of `WS_SIGNOZ_DASHBOARDS` through SigNoz's API, from
`roles/server/ubuntu/signoz/internal/dashboards/dashboards/NAME.json`.
That file must hold NAME as its `name`, or a run fails, and a test with it. The role replaces a
dashboard where it differs from its file. So a change made to it in the UI lasts until the next
run, while one made to a copy of it stays. A name
taken off the list leaves its dashboard in SigNoz. SigNoz refuses to change a dashboard locked in
the UI, so a run leaves one that differs and says so. `--check` reports whether it would create or
replace each dashboard.

## The compose files

The compose files are generated. SigNoz's [Foundry](https://github.com/SigNoz/foundry) forges them
from `roles/server/ubuntu/signoz/casting.yaml`, which pins every image, and the checksum of a ClickHouse
function the stack downloads from GitHub. To change them, change the casting, then forge again with
foundryctl 0.3.0 and commit what it writes. A file or directory that a forge adds or drops takes
its entry in `mise.server.toml` too, which `TestEachForgedFileIsMises` checks:

```sh
cd roles/server/ubuntu/signoz
rm -rf deployment casting.yaml.lock
foundryctl forge --no-ledger --no-updater -f casting.yaml -p .
```

Without `--no-ledger` and `--no-updater`, foundryctl reports each command to SigNoz and asks GitHub
for a newer release. It never removes a file, hence the `rm`.

A new SigNoz may return the dashboards with fields added or dropped, and the role would then replace
them on every run. So after changing SigNoz's image, unlock any dashboard locked in the UI, reported
or not, as the refresh stops at one. Then run `ws server signoz sync`, refresh
`host.json` and `claude.json` from SigNoz as
`roles/server/ubuntu/signoz/internal/dashboards/dashboards/README.md` says,
review their diff, and commit them with the casting. A second run must then leave both dashboards as
they are.
