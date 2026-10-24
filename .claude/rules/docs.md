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
  with the files they cover, as [roles.md](roles.md) is.
- `docs/NAME.md` is the page of the role NAME: what it sets up and the variables that change it.
  It gives the role's caveats too, such as what a run replaces or what a dry run shows.
- [mac.md](../../docs/mac.md) is the Mac's side, `mac/`, [mac-git.md](../../docs/mac-git.md) its
  git and [mac-wizard.md](../../docs/mac-wizard.md) its wizard. [router.md](../../docs/router.md) is
  the router's steps, taken once by hand.
- [checks.md](../../docs/checks.md) says what each gate of `make lint` runs, what CI's other jobs
  check, and what the hooks of `.githooks` and Claude's hooks do.
- [helix-keymap.md](../../docs/helix-keymap.md) indexes the keymap's pages in
  `docs/helix-keymap/`.
- [The dashboards](../../roles/signoz/files/dashboards/README.md) says what each dashboard reads,
  and how to refresh them from SigNoz.
- ws is public, and names no server ([mac.conf](../../docs/mac.md#macconf)). So no doc holds a
  server's address, a secret or a word of the local list (CLAUDE.md).
- A change goes together across the role, its comments, its page, its line in the README, and
  `docs/checks.md` where CI changes.
