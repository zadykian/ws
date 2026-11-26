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
  `.claude/rules/`, loaded with the files they cover, as [ubuntu.md](ubuntu.md) is.
- `GLOSSARY.md` names the terms of Ubuntu's side and of the Mac's: what the code and the docs
  mean by each.
- A page lives beside the code it describes, as that directory's `README.md`. A second page stands
  beside it, named for its subject, where the cap keeps two apart or no package fits:
  `zed/projects.md`, `cable/router.md`, `runbook.md` of each side, `file.md`, `state.md` and
  `launchd.md` of `engine/core/kind`, and `.github/workflows/server.md`, `client-cld.md`,
  `client-servers.md` and `release.md`. CI's page is
  `.github/workflows/README.md`, never `.github/README.md`, which GitHub would show in place of
  the root's.
- A role's page is the `README.md` in its directory, `ubuntu/roles/NAME/`: what it sets up
  and the values that change it. It gives the role's caveats too, such as what a run replaces or
  what a dry run shows.
- [The frame's README](../../ubuntu/roles/frame/README.md) says how a role sits in mise's
  config: its files by env, the link, its lines, their order, the miserc and ws's path.
- [The Mac's README](../../macos/README.md) is the Mac's side, `macos/`
  and the ws CLI. [Its git](../../macos/roles/git/README.md) is its git and gh.
  [Its ssh](../../macos/roles/ssh/README.md) is its hosts, and [its cld](../../macos/roles/cld/README.md)
  cld's unit. [Its values](../../macos/values/README.md) is
  the Mac's values files, [the tunnel's](../../macos/roles/tunnel/README.md) the daemons' rules, and [ghostty](../../macos/roles/ghostty/README.md) its Ghostty
  and what cld does there, [zed](../../macos/roles/zed/README.md) its Zed,
  [firewall](../../macos/roles/firewall/README.md) its firewall.
  [Its zsh](../../macos/roles/zsh/README.md) is its shell: oh-my-zsh, its plugins and ws's settings.
  [Its power](../../macos/roles/power/README.md) is its display and the App Store's updates.
  [Its devtools](../../macos/roles/devtools/README.md) is its command-line tools, and
  [its docker](../../macos/roles/docker/README.md) its Docker, and
  [its desktop](../../macos/roles/desktop/README.md) its Dock, Finder, sounds and wallpaper.
  [Its keyboard](../../macos/roles/keyboard/README.md) is its Caps Lock, its hotkeys and its layout.
  [Its apps](../../macos/roles/apps/README.md) are its apps, the Dock's apps, its login items and Stats's settings. [Its secrets](../../macos/roles/secrets/README.md)
  is the secrets' tools, fnox and age-plugin-yubikey, `ws macos secrets`, and remove's part.
  [The server's secrets](../../ubuntu/roles/secrets/README.md#the-repository) is the fnox
  repository: its files, its recipients and the inputs of ws's roles. [The router](../../ubuntu/roles/cable/router.md)
  is the router's steps, taken once by hand. [The Mac's runbook](../../macos/runbook.md)
  and [the server's](../../ubuntu/runbook.md) are each side's steps by hand.
- [The real machine's checks](../../ubuntu/checks/README.md) list what a person runs on a host, its
  guests, the Mac and the LAN, with the scripts beside the page. [builders.md](../../ubuntu/checks/builders.md)
  holds the open points that the builds left. [The Mac's checks](../../macos/checks/README.md) are
  the ten trials of a clean macOS 27, with the scripts beside the page.
- [The kind package's README](../../engine/core/kind/README.md) says what a unit and a kind are:
  the spec, each kind's fields, and where the specs sit.
- [release.md](../../.github/workflows/release.md), beside `release.yml`, says how to install ws,
  and how a release is cut.
- [The tools' README](../../.tools/README.md) says what each gate of `make lint` runs.
  [CI's README](../../.github/workflows/README.md) says what CI's other jobs check, and what the
  hooks of `.githooks` and Claude's hooks do.
  [client-cld.md](../../.github/workflows/client-cld.md) says what the job `client-cld` checks.
  [server.md](../../.github/workflows/server.md) says what `server` checks.
- [The zed role's README](../../ubuntu/roles/zed/README.md) is the page of the `zed` role:
  what runs on ws, Zed's downloads, trust and the limits.
  [projects.md](../../ubuntu/roles/zed/projects.md) is a project's `.zed/settings.json`,
  for SQL Server and Ansible.
- [The dashboards](../../ubuntu/roles/signoz/internal/dashboards/dashboards/README.md) says what each dashboard reads,
  and how to refresh them from SigNoz.
- ws is public, and names no server ([values.yml](../../macos/README.md#the-macs-valuesyml)). So no doc holds a
  server's address, a secret or a word of the local list (CLAUDE.md).
- A change goes together across the role, its comments, its page, its line in the README, and
  `.github/workflows/README.md` where CI changes.
