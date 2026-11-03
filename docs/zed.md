# zed

The `zed` role is Go steps in `internal/server/zed`. Zed itself runs on the Mac, which opens ws over
ssh. The role writes the settings that Zed's server on ws reads, and raises the open-file limit
that the server needs. The Mac's side is `docs/mac-zed.md`.

`~/.config/zed/settings.json` is owned by `ws_user`. A run replaces a change made to it by hand
and keeps the old file beside the new one, as [helix](helix.md) does for its files. The key
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

ws's file names the language servers by path. A path makes Zed skip every download and add no
arguments. It sets `terminal.shell` to `system`, Zed's default. A program set in the Mac's
`settings.json` would run on ws too (zed#65548), and fail there if ws lacks it.

## Zed's downloads, outside the SHA-256 rule

- Zed's server, about 39 MB. ws's own curl fetches it from Zed's release server for each Mac
  version, with no checksum from Zed.
- The extensions, the servers they download, and `delve-shim-dap`. Zed fetches the shim's latest
  release from GitHub when the server first needs it, so pinning Delve does not remove it.
- Zed's Node is not on the list. It uses ws's Node 24 from PATH ([devtools](devtools.md)), and
  downloads its own only where it finds no Node 22 or newer.

A pin of Zed's server gains nothing, so `upload_binary_over_ssh` stays off. Zed replaces and cleans
the server, and the Mac's Zed updates itself about twice a week. A pin would need the Mac's Zed
pinned too, with auto-update off and a checked download in the Mac's steps, and a commit as often.
ws has the internet ([outbound](security.md#outbound)), so the download works.

The gopls, Delve and Roslyn that ws pins are not on the list ([devtools](devtools.md)).

## The Go and C# servers

gopls, Delve and Roslyn are the pinned ones of the `devtools` role. The settings name them by path,
with Roslyn's `--telemetryLevel off`. From 5.13.0-1.26514.10 its default is `all` (roslyn#86066).

Roslyn takes the first of these as its solution:

1. `dotnet.defaultSolution` in the root's `.vscode/settings.json` (csharp#100).
2. The one `.sln` or `.slnx` at the root of the folder opened in Zed.
3. Every `.csproj` below it, 500 at most, with `test` and `tests` dropped first.

A `.sln` beside a `.slnx` count as two, which falls to the third. .NET 10's `dotnet new sln`
writes `.slnx`. helix's root is the git root instead ([helix](helix.md#c-in-helix)).

The C# extension types `.csproj`, `.props`, `.targets` and `.slnx`, and offers Restore and Build
tasks for projects and solutions. `.sln` opens as plain text.

For Go, the shim and `dlv` run on ws, and the Mac forwards the debug port over ssh. The sshd
drop-in restricts no forwarding, and the outbound table accepts loopback
([security](security.md#outbound)). `ws_user` is root by default, so Yama's `ptrace_scope` does not
stop an attach.

C# keeps Rider for its gaps: Zed has no C# debugger (csharp#12), no Razor (csharp#41) and no C#
test runnables. Debug C# or edit Razor in Rider through Gateway ([jetbrains](jetbrains.md)).

There is no msbuild-lsp yet. Its v0.7.0 of 2026-07-24 is the latest release, and fails on SDKs from
10.0.400 and the 11.0 previews, where MSBuild references `NuGet.Frameworks` itself. The fix
(msbuild-project-tools-server#145) is on master, unreleased. When a release has it, pin its
`language-server.zip` by SHA-256, as otelcol's `.deb` is, name it in
`lsp.msbuild-project-tools.binary`, and add `msbuild-lsp` to the Mac's extensions.
