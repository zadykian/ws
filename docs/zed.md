# zed

The `zed` role is Go steps in `internal/server/zed`. Zed itself runs on the Mac, which opens ws over
ssh. The role writes the settings that Zed's server on ws reads, and raises the open-file limit
that the server needs. The Mac's side is `docs/mac-zed.md`.

`~/.config/zed/settings.json` is owned by `ws_user`. A run replaces a change made to it by hand
and keeps the old file beside the new one. The key
`zed_nofile` of `/etc/ws/host_vars.yml` sets the limit, 65536 by default.

## What runs where

- The Mac has the UI, highlighting and unsaved changes.
- ws has the files, git, language servers, debug adapters, terminals and tasks, all as `ws_user`.
- `~/.zed_server` on ws holds the server, one binary per Mac version, which Zed cleans.
- `~/.local/share/zed/` holds `languages`, `remote_extensions`, `server_state` and `logs`.
  `remote_extensions` holds the Mac's extensions that declare a server, with their downloads in
  `work/`.

The settings layers, lowest first:

1. Zed's defaults and its extensions.
2. The global file, which never reaches ws.
3. The user file: the Mac's `settings.json`, as Zed forwards it.
4. ws's `~/.config/zed/settings.json`.
5. A project's `.zed/settings.json`.

ws's file names the language servers by path. Its source is the template
`internal/server/zed/files/settings.json.tmpl`, where `{{.Home}}` is the user's home.
A path makes Zed skip every download and add no arguments. The file sets `terminal.shell.program`
to ws's terminal wrapper. A program set in the Mac's
`settings.json` would run on ws too (zed#65548), and fail there if ws lacks it.

## Zed's downloads, outside the SHA-256 rule

- Zed's server, about 39 MB. ws's own curl fetches it from Zed's release server for each Mac
  version, with no checksum from Zed.
- The extensions, the servers they download other than those below, and `delve-shim-dap`. Zed
  fetches the shim's latest release from GitHub when the server first needs it, so pinning Delve
  does not remove it.
- Zed's own JSON, CSS and YAML servers, and Prettier, which Zed fetches at their latest. The
  pinned `vscode-langservers-extracted` also ships the JSON and CSS servers. Linking their bins
  onto PATH would serve Zed's adapters, which ws does not do.
- Zed's Node is not on the list. It uses ws's Node 24 from PATH ([devtools](devtools.md)), and
  downloads its own only where it finds no Node 22 or newer.

A pin of Zed's server gains nothing, so `upload_binary_over_ssh` stays off. Zed replaces and cleans
the server, and the Mac's Zed updates itself about twice a week. A pin would need the Mac's Zed
pinned too, with auto-update off and a checked download in the Mac's steps, and a commit as often.
ws has the internet ([outbound](security.md#outbound)), so the download works.

The gopls, Delve and Roslyn that ws pins are not on the list ([devtools](devtools.md)). Nor are
postgres-language-server and docker-language-server, which `devtools` checks against a SHA-256.
Nor are the npm servers, which `devtools` checks against the sha512 `integrity` of a lockfile.

## The Go and C# servers

gopls, Delve and Roslyn are the pinned ones of the `devtools` role. The settings name them by path,
with Roslyn's `--telemetryLevel off`. From 5.13.0-1.26514.10 its default is `all` (roslyn#86066).

Roslyn takes the first of these as its solution:

1. `dotnet.defaultSolution` in the root's `.vscode/settings.json` (csharp#100).
2. The one `.sln` or `.slnx` at the root of the folder opened in Zed.
3. Every `.csproj` below it, 500 at most, with `test` and `tests` dropped first.

A `.sln` beside a `.slnx` count as two, which falls to the third. .NET 10's `dotnet new sln`
writes `.slnx`.

The C# extension types `.csproj`, `.props`, `.targets` and `.slnx`, and offers Restore and Build
tasks for projects and solutions. `.sln` opens as plain text.

For Go, the shim and `dlv` run on ws, and the Mac forwards the debug port over ssh. The sshd
drop-in restricts no forwarding, and the outbound table accepts loopback
([security](security.md#outbound)). `ws_user` is `zadykian` by default, so Yama's `ptrace_scope` stops an
attach to a process that is not its child. Launching under `dlv` works.

Zed has no C# debugger (csharp#12), no Razor (csharp#41) and no C# test runnables.

There is no msbuild-lsp yet. Its v0.7.0 of 2026-07-24 is the latest release, and fails on SDKs from
10.0.400 and the 11.0 previews, where MSBuild references `NuGet.Frameworks` itself. The fix
(msbuild-project-tools-server#145) is on master, unreleased. When a release has it, pin its
`language-server.zip` by SHA-256, as otelcol's `.deb` is, name it in
`lsp.msbuild-project-tools.binary`, and add `msbuild-lsp` to the Mac's extensions.

## Secrets in ssh's arguments

`~/.bashrc` sources `~/.secrets/env` above its interactive guard, so that non-interactive shells
get the values too ([shell](shell.md)). Zed captures each project's environment with the login
shell, `bash -l -i -c 'cd DIR; zed --printenv'`. So the capture holds every value of the
`[env]` table of the secrets file ([Secrets](shell.md#secrets)).

A Zed terminal or task on the Mac then goes out as `ssh -t ws "cd DIR && exec env 'NAME=VALUE' ...
PROGRAM"`. The values are arguments of the Mac's `ssh`, visible to `ps` there while the terminal
lives, and a failed command may reach `Zed.log`. Language servers and git stay on ws.

- Keep in `[env]` only what shells need. The page documents this and filters nothing.
- Nothing above the interactive guard of `.bashrc` may print: Zed's proxy speaks over an ssh whose
  bash reads `~/.bashrc` non-interactively. `bash -lic 'echo ok'` must print `ok` alone.
- Not reported upstream yet. Skipping the file where `BASH_EXECUTION_STRING` holds `--printenv`
  would also keep the values from tasks, debug adapters and language servers (untested).

## Open each project from Remote Projects

A project that the agent panel opens starts no extension's server, gopls aside (zed#59871). The
issue carries the `stale` label, so it may close with no fix: check before dropping this advice.

## Trust each project

Zed starts no language server for a worktree until you trust it, whatever a `binary.path` says. It
applies no `.zed/settings.json` of that worktree either. Trust it when Zed asks, once for each
worktree.

- No per-host setting marks ws as trusted (zed#66313). The answer is stored for the ssh user, the
  host and the path, and trusting a folder covers its subfolders.
- `session.trust_all_worktrees` skips the check for every project, local ones too. ws sets none.
  Whether the Mac's value reaches ws's server is untested.
- Check trust first when a server never starts. ws's newest
  `~/.local/share/zed/logs/server-*.log` says `Waiting for worktree ... to be trusted`.

## The open-file limit

`/etc/security/limits.d/ws-zed.conf` gives `ws_user` a soft limit of `zed_nofile` open files. sshd's
logins start at 1024 on Ubuntu 26.04, and past it Zed's server busy-loops at high CPU while its
reconnect proxies never exit (zed#62995). Zed raises the limit in its GUI alone.

- The file names the user, as limits.conf(5) says a `*` line does not apply to root.
- A new ssh login gets the limit through `pam_limits`. A running server keeps 1024 until it
  restarts, which closing the project or quitting Zed does.
- `ssh ws 'ulimit -Sn'` prints `zed_nofile` in a new login.

## Terminals in tmux

Zed's terminals on ws run `/usr/local/lib/ws/zed/terminal`, which opens a tmux session of the
project's directory on a socket of its own, `tmux -L zed`, so that a reconnect leaves the shell
running. A new tab takes a session of that directory that no tab holds, else starts one. Closing a
tab leaves its session for the next tab, and `exit` ends it. `tmux -L zed ls` lists the sessions.

- A new session takes its tab's environment, the project's own included.
- A tab that takes over a session keeps that session's shell, and so its environment.
- The scrollback, and any search of it, are tmux's: the wheel enters tmux's copy mode, not Zed's own
  scrollback.
- tmux has no prefix, so Fresh gets Ctrl+B, and no status line. `/usr/local/lib/ws/zed/tmux.conf`
  has the rest.
- tmux passes on a title that a program sets, for the toolbar of [the Mac's settings](mac-zed.md).
  Where none is set, the title is empty, not tmux's default, the host's name.
- Without tmux on the `PATH` the tab runs a login shell.
- A task ignores the wrapper.
- Keep `terminal.detect_venv` `off`, as [the Mac's settings](mac-zed.md) do.

## When the connection drops

Zed's heartbeat pings 5 s after the last message from ws, waits 5 s, and gives up after 5 misses:
about 50 s after ws's last message. An idle session has no ping, so ssh's `ServerAliveInterval 30`
ends it after 120 s. A reconnect makes 3 attempts back to back, and v1.26 waits up to 30 s in each.
Terminal output is no traffic to the heartbeat.

What survives depends on the kind of reconnect:

- An automatic reconnect gives the old daemon a new proxy. Language servers, buffers and unsaved
  edits stay, and the tabs resync. Tasks, debug sessions and agent threads are lost, and so are
  the terminal tabs, but their tmux sessions stay (zed#20589, zed#65338).
- A manual Reconnect, a quit or an update of Zed starts a new daemon, which kills the old one.
  Unsaved edits stay on the Mac. Tabs are often lost (zed#51697), the project panel collapses
  (zed#56049), and a multi-root project may return with one folder (zed#41708).
- Zed updates itself about twice a week, and each update costs a new daemon.
- A move between networks costs a pause of a few seconds, not a reconnect: `ws` is `10.99.0.1` from
  every network.

### The daemon

- It exits 10 min after its last proxy detaches. No setting changes that.
- It outlives a dropped ssh. sshd gives the proxy a session of its own, so no SIGHUP comes. Never
  set logind's `KillUserProcesses`: a user other than root would lose the daemon at every
  disconnect.
- It serves one proxy at a time. While ws holds the old connection, the old proxy stays attached,
  and the new proxy's attempts all time out. This stale proxy is suspected from the source and from
  zed#59394, and not reproduced. It ends when ws next sends on the old connection and the Mac
  answers with a reset. A manual Reconnect gets through.

### Sleep

ws keeps a connection with nothing unacknowledged for 2 h 11 min (`tcp_keepalive_time` 7200 s and
9 probes 75 s apart). With unacknowledged data it gives up 15.5 to 16.5 min after its first send
that the Mac never acknowledged. Output in a terminal counts as traffic, such as a tmux status
clock.

- A sleep of up to 2 h 11 min while ws is silent loses nothing, a terminal with no output included.
- With traffic, a sleep under 15.5 min usually loses nothing. If proxy traffic armed the heartbeat,
  Zed reconnects first, at 50 s, onto the old daemon.
- From 15.5 to 25.5 min, ws has aborted the connection and the daemon's timer has started. The
  automatic reconnect lands on the old daemon.
- Past 25.5 min, the daemon is gone: a manual Reconnect, with a server restart.
- This is read from the source and the kernel's defaults, and untested.

### What to run

| Symptom | Remedy |
| --- | --- |
| An extension's server never starts for an open buffer (zed#60923) | `editor: restart language server` |
| All servers gone after a manual Reconnect (zed#60328) | `workspace: reload`, or close and reopen the project |
| Tabs lost | Reopen the files |
| An agent's read or grep never ends (zed#65338) | Stop the turn and send it again |
| New tabs start new sessions while old tmux clients stay attached | `tmux -L zed detach-client -s =NAME`, then a new tab |
| The Disconnected overlay | Reconnect, at the cost of a new daemon |
| `ssh ws` fails at once for an hour | fail2ban banned the tunnel address: [lift the ban](mac.md#sshs-hosts) |

Neither command has a default key. Release builds register `remote_debug::SimulateDisconnect`,
`SimulateTimeout` and `SimulateTimeoutExhausted` in a remote project's workspace. They reproduce
each path above except the stale proxy.

### What stays unchanged

- Keep `ServerAliveInterval 30`: a lower value brings forward the reconnect that kills terminals.
- Set no `ConnectTimeout` or Zed `connection_timeout`: each attempt would wait less than 17 s.
- Add no `ControlMaster` for `ws`, and no sshd `ClientAlive*`: every probe is data, and a sleeping
  Mac would lose its session at about 15 min. A drop-in with `ClientAliveInterval 15` and
  `ClientAliveCountMax 60` helps only if the stale proxy is confirmed.
- Raise `MaxSessions` to 32 only if the eleventh terminal or task is refused. Every terminal, task,
  debug adapter, agent and proxy of every project shares the default 10.

## Limits

A project's own `.zed/settings.json` is in [zed-projects](zed-projects.md).

- The project panel may not refresh when files change on ws (zed#29242).
- There is no `zed FILE` from ws's terminal (zed#56057), and files outside the project cannot be
  opened (zed#39695).
- Files added outside the editor may need a restart of the language server (csharp#104).
- gopls stays dead after a workspace load error, such as inconsistent vendoring (zed#54704).
- Delve's "Rerun session" fails with a broken pipe (zed#62389).
- After a reconnect that starts a new server, language servers may never start (zed#60923). Run
  `editor: restart language server`. If all vanish, run `workspace: reload` (zed#60328, untested).
  [When the connection drops](#when-the-connection-drops) says more.
