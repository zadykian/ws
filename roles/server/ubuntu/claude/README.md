# claude

The `claude` role sets claude and cld up from `mise bootstrap`. Its `mise.server.toml` installs
curl, which fetches the installers, and names the nine units of its `units.toml.tmpl`. They run one
after another, in this order, after git's tokens:

- `state/claude-install` and `state/claude-cld-install`: claude, from its native installer, and
  cld, from its latest release's `install.sh`, both in `~/.local/bin`, as root
  ([Installs and updates](#installs-and-updates)).
- `state/claude-update`, `-completion`, `-settings`, `-telemetry` and `-marketplaces`: the commands
  `ws server claude update`, `completion`, `settings`, `telemetry` and `marketplaces`, as
  root, each with `--check` as its check, which fails where the command would change something.
- `state/claude-linger` and `state/claude-restore`: lingering for root, then
  `ws server claude restore` ([Sessions after a reboot](#sessions-after-a-reboot)).

## Installs and updates

claude updates itself in the background, so its unit installs it only where `~/.local/bin/claude`
is missing. So does cld's, and `ws server claude update` updates cld with `cld update` where
a newer release is out, which it reads over HTTP. Each installer runs as root, with `HOME`
set. It fetches the whole script before it runs it, so a failed fetch runs none of it. Neither is
pinned. claude's installer refuses root where `SUDO_USER` is set, as under `sudo -i`, so its unit
tells it that root is meant. cld's would take the caller's `CLD_INSTALL_DIR` over `~/.local/bin`,
so its unit sets it. `ws server claude completion` writes cld's completion in bash where it
differs from what `cld completion bash` prints.

## User settings

Next, `cld setup config user` adds to `~/.claude/settings.json` what it lacks. It adds the
permission set `WS_CLAUDE_PERMISSIONS` names, `read-only` by default. It adds cld's choices of model,
effort, theme, editor mode, auto-compact and update channel. With `WS_CLAUDE_NOTIFICATIONS`, it adds
the channel claude notifies on. It keeps every value already there. `--check` runs cld on a copy of
the file and lists the keys it would add. A cld that predates the command gets it from the update
first.

Bypass mode, which skips every permission prompt, stays off as root: claude refuses it there, and
ws sets nothing that lifts the refusal.

## Sessions after a reboot

Then `cld setup restore`, which has the user's systemd bring cld's sessions back after a reboot.
`state/claude-linger` turns lingering on, so that the user's systemd starts at boot rather than at
the first login, and `restore` starts it now. The unit keeps the `PATH` of the user's login shell,
where `cld restore sessions` finds tmux. The next run updates it, after a change to that `PATH` or
to the command cld runs there.

## Telemetry

claude sends its metrics, logs and traces to the host's collector on `127.0.0.1:4317`
([otelcol](../otelcol/README.md)), with what it leaves out by default. The role sets the keys for that in the
`env` of `~/.claude/settings.json` and removes each signal's own endpoint and protocol. It removes
`OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE`, so that claude sends its counters as deltas,
its default, which the Claude dashboard counts ([signoz](../signoz/README.md#the-dashboards)). It leaves every
other key untouched, and writes the file only where `env` changes. The role edits the file as an
ordered walk of its JSON, so that every other key keeps its place. It writes the file as claude
does: two spaces, a member a line, a new key at the end. `--check` lists the keys it would set or
remove, not the file's diff, as the file may hold tokens.

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
([Retention](../signoz/README.md#retention)). That includes a secret a command printed or a file held. The
logs and traces stay on the server: the collector sends only claude's metrics elsewhere. SigNoz's
UI listens on loopback alone. A copy of `WS_CLAUDE_TELEMETRY_ENV` without these keys stops the role
setting them, and leaves them in `env` until removed by hand.

The role replaces an endpoint already in `env`. To keep sending claude's metrics to a remote
collector, set its endpoint first as the secret input `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT`
([The env file](../shell/README.md#the-env-file)). mise writes it to `/etc/ws/env`, which the
`otelcol` role reads: [claude's metrics elsewhere](../otelcol/README.md#claudes-metrics-elsewhere).
claude reads the keys as it starts, so a session started before keeps sending where it did. A key
the role removes still applies where the shell exports it, as from `/etc/ws/env`.

## Plugin marketplaces

`WS_CLAUDE_MARKETPLACES`, empty by default, lists the plugin marketplaces the role adds for root,
each a `name` and a `source`. The source is what `claude plugin marketplace add` takes: a GitHub
`owner/repo`, a git URL or an absolute path. The first two take a branch or tag as `#REF`. A git URL
off GitHub ends in `.git`, as claude reads one without it as a hosted `marketplace.json`. The name
is the one the marketplace's own `marketplace.json` gives, as `add` takes none:

```yaml
WS_CLAUDE_MARKETPLACES:
  - name: acme-plugins
    source: https://git.example.internal/team/plugins.git
```

A source that names a private server stays out of this repository: its list goes in the machine's
`values.yml`. A private source needs the VPN of the `ikeward` role
([A run](../ikeward/README.md#a-run)), and a token for its host from the `git` role
([Git credentials](../git/README.md#git-credentials)). claude's units run after git's tokens.

`ws server claude marketplaces` adds each name that `claude plugin marketplace list`
lacks, so the next run finds it and changes nothing. `add` clones a git source and declares the
marketplace in `~/.claude/settings.json`; it needs no claude login. A name claude lists from
another source fails the run: give that source, or remove the marketplace first, with
`claude plugin marketplace remove`.
claude drops a trailing slash, and adds `.git` to a GitHub URL, so those still match. A source it
lists otherwise after `add`, such as a relative path made absolute, fails that run. So does a
marketplace that names itself otherwise, which `add` keeps under its own name. `--check` names the
marketplaces a run would add, and clones none, so it sees neither. An empty list asks claude
nothing. The role removes no marketplace, and installs no plugin.
