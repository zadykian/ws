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
  `zed/projects.md`, `network/router.md`, and `.github/workflows/rootless.md`, `mac-cld.md` and
  `mac-servers.md`. CI's page is `.github/workflows/README.md`, never `.github/README.md`, which
  GitHub would show in place of the root's.
- A role's page is the `README.md` in its directory, `roles/server/ubuntu/NAME/`: what it sets up
  and the values that change it. It gives the role's caveats too, such as what a run replaces or
  what a dry run shows.
- [The Mac's README](../../roles/client/macos/README.md) is the Mac's side, `roles/client/macos/`
  and the ws CLI. [Its git](../../roles/client/macos/git/README.md) is its git, and
  [its stages](../../roles/client/macos/stages/README.md) its wizard. [Its conf](../../roles/client/macos/conf/README.md)
  is the wizard's targets, and [ghostty](../../roles/client/macos/ghostty/README.md) its Ghostty
  and cld, [zed](../../roles/client/macos/zed/README.md) its Zed. [The secrets](../../roles/server/ubuntu/secrets/README.md)
  is the secrets file. [The server's stages](../../roles/server/ubuntu/stages/README.md) are
  `ws server ubuntu wizard`, the stages run on ws. [The router](../../roles/server/ubuntu/network/router.md)
  is the router's steps, taken once by hand.
- [The TUI's README](../../engine/tui/README.md) is the terminal UI: its modes, `--plain`, the
  layout and the keys. [Configure](../../engine/tui/internal/configure/README.md) is its Configure
  screen, and [the queue](../../engine/tui/internal/queue/README.md) the wizard's two lists.
- [The update package's README](../../engine/core/update/README.md) says how to install and update
  ws, and how a release is cut.
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
- ws is public, and names no server ([mac.conf](../../roles/client/macos/README.md#macconf)). So no doc holds a
  server's address, a secret or a word of the local list (CLAUDE.md).
- A change goes together across the role, its comments, its page, its line in the README, and
  `.github/workflows/README.md` where CI changes.
