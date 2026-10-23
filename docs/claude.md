# claude

The `claude` role installs claude, from its native installer, and cld, from its latest release's
`install.sh`, both in `~/.local/bin`, and cld's completion in bash.

## Installs and updates

claude updates itself in the background, so the role installs it only on a machine without it.
Where cld is installed, the role updates it with `cld update`; a dry run compares its version with
the latest release instead.

## User settings

Next, `cld setup config user` adds to `~/.claude/settings.json` what it lacks. It adds the
permission set `claude_permissions` names, `read-only` by default. It adds cld's choices of model,
effort, theme, editor mode, auto-compact and update channel. With `claude_notifications`, it adds
the channel claude notifies on. It keeps every value already there. A dry run runs cld on a copy of
the file and lists the keys it would add. A cld that predates the command gets it from the role's
`cld update` first; a dry run, which updates nothing, notes that cld lacks it.

## Sessions after a reboot

Then `cld setup restore`, which has the user's systemd bring cld's sessions back after a reboot.
The role turns lingering on, so that the user's systemd starts at boot rather than at the first
login. The unit keeps the `PATH` of the user's login shell, where `cld restore sessions` finds tmux.
The next run of the role updates it, after a change to that `PATH` or to the command cld runs
there.

## Telemetry

claude sends its metrics, logs and traces to the host's collector on `127.0.0.1:4317`
([otelcol](otelcol.md)), with what it leaves out by default. The role sets the keys for that in
the `env` of `~/.claude/settings.json` and removes each signal's own endpoint and protocol. It
removes `OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE`, so that claude sends its counters as
deltas, its default, which the Claude dashboard counts ([signoz](signoz.md#the-dashboards)). It
leaves every other key untouched, and writes the file only where `env` changes. A dry run lists the
keys it would set or remove, not the file's diff, as the file may hold tokens.

Four of the keys add content to the logs and traces:

- `OTEL_LOG_USER_PROMPTS`: each prompt's text, on its `user_prompt` event and its spans.
- `OTEL_LOG_ASSISTANT_RESPONSES`: each response's text, on its `assistant_response` event. claude
  would take it from `OTEL_LOG_USER_PROMPTS` where unset, so the role sets it outright.
- `OTEL_LOG_TOOL_DETAILS`: Bash commands, file paths and each tool's input, and the real names of
  skills, agents, plugins and MCP servers.
- `OTEL_LOG_TOOL_CONTENT`: what a tool returned, on its span. That is a command's output, the text
  Read returned, or an MCP tool's or a web fetch's result. With tool details, an Edit's diff and the
  text a Write wrote go too. claude cuts each at 60 KB.

So SigNoz holds what each session was asked and answered, and what it ran and read, for 90 days
([Retention](signoz.md#retention)). That includes a secret a command printed or a file held. The
logs and traces stay on the server: the collector sends only claude's metrics elsewhere. SigNoz's
UI listens on loopback alone. A copy of `claude_telemetry_env` without these keys stops the role
setting them, and leaves them in `env` until removed by hand.

The role replaces an endpoint already in `env`. To keep sending claude's metrics to a remote
collector, set its endpoint first as `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT` in `shell_secret_env`
([Secrets](shell.md#secrets)). The `shell` role writes it to `~/.secrets/env`, which the `otelcol`
role reads: [claude's metrics elsewhere](otelcol.md#claudes-metrics-elsewhere).
claude reads the keys as it starts, so a session started before keeps sending where it did. A key
the role removes still applies where the shell exports it, as from `~/.secrets/env`.

## Plugin marketplaces

`claude_marketplaces`, empty by default, lists the plugin marketplaces the role adds for `ws_user`,
each a `name` and a `source`. The source is what `claude plugin marketplace add` takes: a GitHub
`owner/repo`, a git URL or an absolute path. The first two take a branch or tag as `#REF`. A git URL
off GitHub ends in `.git`, as claude reads one without it as a hosted `marketplace.json`. The name
is the one the marketplace's own `marketplace.json` gives, as `add` takes none:

```sh
/root/repository/ws/bootstrap.sh -e '{"claude_marketplaces": [{"name": "acme-plugins",
    "source": "https://git.example.internal/team/plugins.git"}]}'
```

A source that names a private server stays out of this repository: a private repository passes the
list in its extra vars, as `-e @FILE`. A private source may sit behind a VPN, which must be up
before the run. It needs a token for its host from the `shell` role, which runs before `claude`
([Git credentials](shell.md#git-credentials)).

The role adds each name that `claude plugin marketplace list` lacks, so the next run finds it and
changes nothing. `add` clones a git source and declares the marketplace in
`~/.claude/settings.json`; it needs no claude login. A dry run names the marketplaces a run would
add, and clones none. The role removes no marketplace, and installs no plugin.
