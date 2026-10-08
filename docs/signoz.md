# signoz

The `signoz` role runs [SigNoz](https://signoz.io) in Docker, which keeps the machine's telemetry
for 90 days. Its UI and its OTLP/HTTP intake listen on loopback, with no login. It has a dashboard
of the host and one of claude's usage.

## The stack

SigNoz runs in Docker from the compose files in `roles/signoz/files/deployment`, which the role
copies to `/opt/signoz`. Docker's published ports get past ufw, so only two are published, on
loopback: the UI on `127.0.0.1:3301` and the OTLP/HTTP intake on `127.0.0.1:14318`. 3301 was
SigNoz's UI port before 8080, which dev servers often take.

ClickHouse, SigNoz's database, may use 4 GB of memory. Without a limit of its own, it would size
itself to the host's RAM, as it cannot see the limit of `docker.slice`.

No one logs in: SigNoz serves every request as its root user, and its UI shows a banner that says
so. The role generates the root user's password on the machine, in `/opt/signoz/root.env`, which
only root reads; nothing else needs it. Reach the UI through an SSH tunnel, then open
`http://localhost:3301`:

```sh
ssh -N -L 3301:127.0.0.1:3301 root@SERVER
```

A `LocalForward 3301 127.0.0.1:3301` in the client's `~/.ssh/config` does the same.

## Retention

SigNoz keeps traces, metrics and logs for 90 days, `signoz_retention_days`; its own default is 15
days, and 30 for metrics. The role sets it through SigNoz's API before anything is sent, as a
change applies only to data that comes in after it. It sets it after SigNoz's schema migrator has
finished too, as the migrator sets its own. Settings › General in the UI shows it; rules set there
to keep some logs for another time stay. A dry run reads it and reports what it would change, and
skips it where SigNoz isn't running.

## The dashboards

The Host dashboard shows the CPU, memory, disks, file systems and network from the collector's host
metrics, and the temperatures and fans from node_exporter. This is SigNoz's own host metrics
dashboard, changed a little: `roles/signoz/files/dashboards` says how, and holds its license.

The Claude dashboard shows what claude on the server cost and what it did, from the metrics and
events claude sends to the collector ([claude](claude.md#telemetry)). It shows cost and tokens by
model and by day, and the share of tokens read from the prompt cache. It shows sessions, active
time, lines of code, commits, pull requests, edit decisions, prompts, tool calls and API errors.
Each covers the range of the UI's time picker. Its bars are days in UTC, the server's timezone.
Cost is claude's estimate at API prices: under a subscription, that is not what was billed. No
panel shows a prompt's text, a tool's input or a command. claude sends no prompt text, as
`OTEL_LOG_USER_PROMPTS` is unset. The dashboard was written for this repository, and
`roles/signoz/files/dashboards` says what it reads.

The role creates each dashboard of `signoz_dashboards` through SigNoz's API. It replaces one where
it differs from the repository's. So a change made to it in the UI lasts until the next run, while
one made to a copy of it stays. A name taken off the list leaves its dashboard in SigNoz. SigNoz
refuses to change a dashboard locked in the UI, so the run leaves it and says so. A dry run reports
whether it would create or replace each dashboard.

## The compose files

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
