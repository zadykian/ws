---
paths:
  - "**/*.md"
---

# Docs

What each document holds. How to write them is in the `writing` skill and [writing.md](writing.md).

- `README.md` is the overview: what ws sets up, with a line per role that links its page. Then a
  new machine, running it again, and dry runs. Then what stays by hand, the checks, how pull
  requests land, and the license.
- `CLAUDE.md` holds what Claude needs before a change: the layout, the commands, how changes land,
  and the hard rules, each linking its reason. An area's invariants go in `.claude/rules/`, loaded
  with the files they cover, as [server.md](server.md) is.
- `GLOSSARY.md` names the terms of the server's side and of the Mac's: what the code and the docs
  mean by each.
- `docs/NAME.md` is the page of the role NAME: what it sets up and the values that change it.
  It gives the role's caveats too, such as what a run replaces or what a dry run shows.
- [mac.md](../../roles/client/macos/README.md) is the Mac's side, `roles/client/macos/` and the ws CLI.
  [mac-git.md](../../roles/client/macos/git/README.md) is its git, and [mac-wizard.md](../../roles/client/macos/stages/README.md) its
  wizard. [mac-targets.md](../../roles/client/macos/conf/README.md) is the wizard's targets, and
  [mac-ghostty.md](../../roles/client/macos/ghostty/README.md) its Ghostty and cld,
  [mac-zed.md](../../roles/client/macos/zed/README.md) its Zed, and [mac-secrets.md](../../roles/server/ubuntu/secrets/README.md) the
  secrets file. [server-wizard.md](../../roles/server/ubuntu/stages/README.md) is `ws server ubuntu wizard`, the
  stages run on ws. [router.md](../../roles/server/ubuntu/network/router.md) is the router's steps, taken once by hand.
- [tui.md](../../engine/tui/README.md) is the terminal UI: its modes, `--plain`, the layout and the keys.
  [tui-configure.md](../../engine/tui/internal/configure/README.md) is its Configure screen, and
  [tui-wizard.md](../../engine/tui/internal/queue/README.md) the wizard's two lists.
- [releases.md](../../engine/core/update/README.md) says how to install and update ws, and how a release is cut.
- [gates.md](../../tools/README.md) says what each gate of `make lint` runs.
  [checks.md](../../.github/workflows/README.md) says what CI's other jobs check, and what the hooks of
  `.githooks` and Claude's hooks do.
  [checks-mac-cld.md](../../.github/workflows/mac-cld.md) says what the job `mac-cld` checks.
  [checks-rootless.md](../../.github/workflows/rootless.md) says what `rootless` checks.
- [zed.md](../../roles/server/ubuntu/zed/README.md) is the page of the `zed` role: what runs on ws, Zed's downloads, trust
  and the limits. [zed-projects.md](../../roles/server/ubuntu/zed/projects.md) is a project's `.zed/settings.json`,
  for SQL Server and Ansible.
- [The dashboards](../../roles/server/ubuntu/signoz/internal/dashboards/dashboards/README.md) says what each dashboard reads,
  and how to refresh them from SigNoz.
- ws is public, and names no server ([mac.conf](../../roles/client/macos/README.md#macconf)). So no doc holds a
  server's address, a secret or a word of the local list (CLAUDE.md).
- A change goes together across the role, its comments, its page, its line in the README, and
  `docs/checks.md` where CI changes.
