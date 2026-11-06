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
- `docs/NAME.md` is the page of the role NAME: what it sets up and the values that change it.
  It gives the role's caveats too, such as what a run replaces or what a dry run shows.
- [mac.md](../../docs/mac.md) is the Mac's side, `mac/` and the ws CLI. [mac-git.md](../../docs/mac-git.md) is its
  git, and [mac-wizard.md](../../docs/mac-wizard.md) its wizard. [mac-targets.md](../../docs/mac-targets.md) is the wizard's targets, and
  [mac-ghostty.md](../../docs/mac-ghostty.md) its Ghostty and cld, [mac-zed.md](../../docs/mac-zed.md) its Zed, and
  [mac-secrets.md](../../docs/mac-secrets.md) the secrets file.
  [router.md](../../docs/router.md) is the router's steps, taken once by hand.
- [tui.md](../../docs/tui.md) is the terminal UI: its modes, `--plain`, the layout and the keys.
  [tui-configure.md](../../docs/tui-configure.md) is its Configure screen.
- [releases.md](../../docs/releases.md) says how to install and update ws, and how a release is cut.
- [gates.md](../../docs/gates.md) says what each gate of `make lint` runs.
  [checks.md](../../docs/checks.md) says what CI's other jobs check, and what the hooks of
  `.githooks` and Claude's hooks do.
  [checks-mac-cld.md](../../docs/checks-mac-cld.md) says what the job `mac-cld` checks.
  [checks-rootless.md](../../docs/checks-rootless.md) says what `rootless` checks.
- [zed.md](../../docs/zed.md) is the page of the `zed` role: what runs on ws, Zed's downloads, trust
  and the limits. [zed-projects.md](../../docs/zed-projects.md) is a project's `.zed/settings.json`,
  for SQL Server and Ansible.
- [helix-keymap.md](../../docs/helix-keymap.md) indexes the keymap's pages in
  `docs/helix-keymap/`.
- [zed-keymap.md](../../docs/zed-keymap.md) indexes the pages of Zed's keymap in
  `docs/zed-keymap/`.
- [The dashboards](../../internal/server/signoz/dashboards/README.md) says what each dashboard reads,
  and how to refresh them from SigNoz.
- ws is public, and names no server ([mac.conf](../../docs/mac.md#macconf)). So no doc holds a
  server's address, a secret or a word of the local list (CLAUDE.md).
- A change goes together across the role, its comments, its page, its line in the README, and
  `docs/checks.md` where CI changes.
