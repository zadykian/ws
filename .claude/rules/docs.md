---
paths:
  - "**/*.md"
---

# Docs

What each document holds. How to write them is in the `writing` skill and [writing.md](writing.md).

- `README.md` is the overview: what ws sets up, with a line per role that links its directory. Then
  a new machine, running it again, and dry runs. Then what stays by hand, the checks, how pull
  requests land, and the license.
- `.claude/CLAUDE.md` holds what Claude needs before a change: the layout, the commands, how
  changes land, and the hard rules, each linking its reason. An area's invariants go in
  `.claude/rules/`, loaded with the files they cover, as [server.md](server.md) is.
- `GLOSSARY.md` names the terms of the server's side and of the Mac's: what the code and the docs
  mean by each.
- A page lives beside the code it describes, as that directory's `README.md`. A second page stands
  beside it, named for its subject, where the cap keeps two apart or no package fits:
  `zed/projects.md`, `cable/router.md`, `runbook.md` of each side, `file.md`, `state.md` and
  `launchd.md` of `engine/core/kind`, and `.github/workflows/rootless.md`, `mac-cld.md`,
  `mac-servers.md` and `release.md`. CI's page is
  `.github/workflows/README.md`, never `.github/README.md`, which GitHub would show in place of
  the root's.
- A role's page is the `README.md` in its directory, `roles/server/ubuntu/NAME/`: what it sets up
  and the values that change it. It gives the role's caveats too, such as what a run replaces or
  what a dry run shows.
- [The frame's README](../../roles/server/ubuntu/frame/README.md) says how a role sits in mise's
  config: its files by env, the link, its lines and their order, the miserc and the pin.
- [The Mac's README](../../roles/client/README.md) is the Mac's side, `roles/client/`
  and the ws CLI. [Its git](../../roles/client/macos/git/README.md) is its git and gh.
  [Its ssh](../../roles/client/macos/ssh/README.md) is its hosts, and [its cld](../../roles/client/macos/cld/README.md)
  cld's unit. [Its values](../../roles/client/values/README.md) is
  the Mac's values files, [the tunnel's](../../roles/client/macos/tunnel/README.md) the daemons' rules, and [ghostty](../../roles/client/macos/ghostty/README.md) its Ghostty
  and what cld does there, [zed](../../roles/client/macos/zed/README.md) its Zed. [Its secrets](../../roles/client/macos/secrets/README.md)
  is the secrets' tools, fnox and age-plugin-yubikey, `ws client macos secrets`, and remove's part.
  [The server's secrets](../../roles/server/ubuntu/secrets/README.md#the-repository) is the fnox
  repository: its files, its recipients and the inputs of ws's roles. [The router](../../roles/server/ubuntu/cable/router.md)
  is the router's steps, taken once by hand. [The Mac's runbook](../../roles/client/runbook.md)
  and [the server's](../../roles/server/runbook.md) are each side's steps by hand.
- [The kind package's README](../../engine/core/kind/README.md) says what a unit and a kind are:
  the spec, each kind's fields, and where the specs sit.
- [release.md](../../.github/workflows/release.md), beside `release.yml`, says how to install ws,
  and how a release is cut.
- [The tools' README](../../.tools/README.md) says what each gate of `make lint` runs.
  [CI's README](../../.github/workflows/README.md) says what CI's other jobs check, and what the
  hooks of `.githooks` and Claude's hooks do.
  [mac-cld.md](../../.github/workflows/mac-cld.md) says what the job `mac-cld` checks.
  [rootless.md](../../.github/workflows/rootless.md) says what `rootless` checks.
- [The zed role's README](../../roles/server/ubuntu/zed/README.md) is the page of the `zed` role:
  what runs on ws, Zed's downloads, trust and the limits.
  [projects.md](../../roles/server/ubuntu/zed/projects.md) is a project's `.zed/settings.json`,
  for SQL Server and Ansible.
- [The dashboards](../../roles/server/ubuntu/signoz/internal/dashboards/dashboards/README.md) says what each dashboard reads,
  and how to refresh them from SigNoz.
- ws is public, and names no server ([values.yml](../../roles/client/README.md#the-macs-valuesyml)). So no doc holds a
  server's address, a secret or a word of the local list (CLAUDE.md).
- A change goes together across the role, its comments, its page, its line in the README, and
  `.github/workflows/README.md` where CI changes.
