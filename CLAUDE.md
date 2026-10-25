# CLAUDE.md

ws sets up the maintainer's development server from code: `site.yml`, an Ansible playbook that runs
on the machine itself. Its roles set up packages, SSH by key and a firewall, the links to the Mac,
an IKEv2 VPN where a profile is given, and the fans. Then the shell, tmux, helix, Docker, claude,
SigNoz with its collector, and the IDEs' backends. `mac/` sets up the Mac that works on the server.
The rest is the docs, the lint gates' tools and CI.

## Layout

- `site.yml`: the play, on localhost, each role a tag of its name. `bootstrap.sh` installs
  ansible-core and git, clones the repository, installs the collections and runs the play.
- `roles/NAME`: a role per tag, each with its page, `docs/NAME.md`, and a line in the README. The
  filter plugins, in Python, are in `roles/NAME/filter_plugins`; ws-fans, the fans' service, is
  Python too, in `roles/fans/files`.
- `group_vars/all.yml`: what roles share, `ws_user`, `ws_home` and `ws_git_signing_key`, and the
  versions, each named for its role.
  `requirements.yml`: the collections, each pinned. `inventory.ini`: localhost, with the system's
  Python. `ansible.cfg`: the inventory, the roles' path, and handlers that run after a failure.
- `mac/`: the Mac's side, POSIX sh, run on the Mac ([The Mac](docs/mac.md)).
- `docs/`: a page per role; `mac.md`, `router.md` and `checks.md`; `helix-keymap.md`, the index of
  the keymap's pages in `docs/helix-keymap/`.
- `tools/`: the gates' tools. `run` pins the binaries, `files` lists a gate's files and `sizecheck`
  caps them.
- `.githooks/`: the hooks that check each commit for the words of the local list and for secrets.
- `.github/workflows/`: `ci.yml`, with the jobs `lint`, `playbook` and `signoz`; `mac.yml`, with
  `mac`; `rules.yml`, the checks of zadykian/actions, which warn alone; and `fast-forward.yml`,
  which lands a pull request on a `/ff` comment.

## Commands

```sh
make lint                         # every gate, as CI's lint job runs it; -k goes past a failure
make vale FILES=README.md         # one gate alone; the gates that check files take FILES
make ansible-lint FILES=roles/base/tasks/main.yml   # this file alone: no syntax check of site.yml
make shell                        # ShellCheck and shfmt -d -i 4 on every shell script
```

The gates are `shell`, `yamllint`, `ansible-lint`, `ruff`, `sizecheck`, `vale`, `lychee`, `rumdl`,
`actionlint`, `zizmor` and `gitleaks` ([Checks](docs/checks.md#make-lint)). `make lint`
needs Python 3.12 to 3.14, on Linux or on macOS on arm64. It installs the Python tools and the
collections into `.cache`, and `tools/run` downloads the rest there.

The tests are CI's jobs, which run on each pull request ([Checks](docs/checks.md)). `playbook`
runs `bootstrap.sh` twice in an Ubuntu 26.04 container, and `signoz` forges SigNoz's compose files
again. `mac` sets up a Mac on GitHub's macOS runner. A run of the playbook sets up the machine that
runs it, as below.

The hooks check a commit for the local list's words. This checks any other text, such as a pull
request's body, as they do, and prints 0 where it holds none:

```sh
LC_ALL=C tr _ ' ' <FILE | grep -ciwF -f <(sed -e 'y/_/ /' -e 's/^[[:space:]]*//' \
    -e 's/[[:space:]]*$//' -e '/^$/d' -e '/^#/d' ~/.config/ws/forbidden-words)
```

## How changes land

- Pull requests are stacked, each on the one below it, and CI runs on each whatever its base.
- They land on `main` by fast-forward only, as the commits CI checked. GitHub's merge methods are
  all refused: never run `gh pr merge`.
- A comment `/ff` on the pull request lands it, as
  [`/flow:land`](https://github.com/zadykian/skills/blob/main/skills/land/SKILL.md) does
  (`.github/workflows/fast-forward.yml`). It pushes the head to `main`, closes the pull request's
  linked issues and deletes the branch.
- The maintainer pushes a pull request that changes `.github/workflows` by hand,
  `git push origin SHA:main`, as the workflow's token may not push such a change.
- `.claude/settings.json` denies `gh pr merge` and the usual forms of a push to `main`. A rule
  matches the command as written, so the ruleset on `main` stays the guard.

## Hard rules

Each rule links where the repository gives its reason: a doc, a comment or a CI job.

- The playbook sets up the machine that runs it: `site.yml` runs on localhost, over a local
  connection, with the system's Python ([inventory.ini](inventory.ini)). CI's `playbook` job runs
  it in a container ([The playbook](docs/checks.md#the-playbook)).
- A second run changes nothing: CI's `playbook` job runs it twice and fails where the second run
  reports a change ([ci.yml](.github/workflows/ci.yml)).
- A task that needs a booted machine's systemd carries the tag `systemd`, which CI's container run
  skips ([The playbook](docs/checks.md#the-playbook)).
- A pinned download is checked against its SHA-256 before use: otelcol's `.deb` and ikeward's
  binary ([group_vars/all.yml](group_vars/all.yml)), Docker's apt key
  ([its task](roles/docker/tasks/main.yml)), foundryctl ([ci.yml](.github/workflows/ci.yml)) and
  the gates' tools ([tools/run](tools/run)). So are the Python tools
  ([lint-requirements.in](.github/lint-requirements.in)). Go, the JetBrains backends and the
  collections are checked against the SHA-256 their publishers list ([devtools](docs/devtools.md),
  [jetbrains](docs/jetbrains.md), [requirements.yml](requirements.yml)). roslyn-language-server is
  pinned by version alone ([group_vars/all.yml](group_vars/all.yml)).
- The collections are pinned in `requirements.yml`, and ansible-lint runs offline on those pins
  ([.ansible-lint](.ansible-lint)).
- foundryctl forges SigNoz's compose files, `roles/signoz/files/deployment/` and
  `casting.yaml.lock`, from `casting.yaml`. Change the casting and forge again, never the files by
  hand: CI's `signoz` job fails where they differ
  ([The compose files](docs/signoz.md#the-compose-files)).
- Secrets are never in this repository ([By hand](README.md#by-hand)).
- ws is public. No word of the local list, `~/.config/ws/forbidden-words`, goes into it. That
  holds for its files, the names of files and branches, the commit messages, and the text of pull
  requests and issues. The hooks in `.githooks` refuse a commit that holds one. They name an added
  line by its file and number, never its text, but print a file name that holds one whole. Never
  print the list or a word of it, and never pass `--no-verify`
  ([The hooks](docs/checks.md#the-hooks)).
- `make lint` fails on any finding, warnings included, in any line of any file
  ([make lint](docs/checks.md#make-lint); #47).
- Fix a finding, or justify it in place with its linter's own mechanism and a reason: ruff's and
  ansible-lint's `# noqa: CODE`, `# yamllint disable-line rule:NAME` or
  `# shellcheck disable=SCNNNN`, each with the reason beside it. Never lower a severity, add an
  exclusion or keep a baseline: every file passes every gate outright, and `.yamllint`,
  `.ansible-lint`, `ruff.toml` and `.vale.ini` are never weakened.
- The caps are those of the
  [writing](https://github.com/zadykian/skills/blob/main/skills/writing/SKILL.md#the-caps-and-vale)
  skill, which `make sizecheck`, `make vale` and `make rumdl` check.

## Commits

Commits and pull requests follow the
[atomic-commits](https://github.com/zadykian/skills/blob/main/skills/atomic-commits/SKILL.md)
skill. Commits are signed, as the ruleset on `main` requires.

## Where else to look

- `.claude/rules/`: the roles' invariants in [roles.md](.claude/rules/roles.md), what each document
  holds in [docs.md](.claude/rules/docs.md), and ws's facts beside the `writing` skill in
  [writing.md](.claude/rules/writing.md), loaded with the files they cover.
- Plugins of the `zadykian` marketplace, [zadykian/skills](https://github.com/zadykian/skills),
  which `.claude/settings.json` enables: `flow`, with `/flow:issue`, and `/flow:land`, run by
  hand only; `docs`, with `/docs:trim`; and `lint`, with `/lint:split-file`.
- Hooks: the lint plugin's, which run `make lint-file` on each file Claude edits and
  `make lint-tree` as a turn ends, on a tree the maintainer vouches for. The gate code includes
  what [.claude/lint-gates](.claude/lint-gates) lists
  ([Claude's hooks](docs/checks.md#claudes-hooks)).
