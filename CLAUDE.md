# CLAUDE.md

ws sets up the maintainer's development server from code: `site.yml`, an Ansible playbook that runs
on the machine itself. Its roles set up packages, SSH by key and a firewall, the links to the Mac,
the shell, tmux, helix, Docker, claude, SigNoz with its collector, and the IDEs' backends. `mac/`
sets up the Mac that works on the server. The rest is the docs, the lint gates' tools and CI.

## Layout

- `site.yml`: the play, on localhost, each role a tag of its name. `bootstrap.sh` installs
  ansible-core and git, clones the repository, installs the collections and runs the play.
- `roles/NAME`: a role per tag, each with its page, `docs/NAME.md`, and a line in the README. The
  filter plugins, in Python, are in `roles/NAME/filter_plugins`.
- `group_vars/all.yml`: `ws_user`, `ws_home` and the versions, each named for its role.
  `requirements.yml`: the collections, each pinned. `inventory.ini`: localhost, with the system's
  Python. `ansible.cfg`: the inventory, the roles' path, and handlers that run after a failure.
- `mac/`: the Mac's side, POSIX sh, run on the Mac ([The Mac](docs/mac.md)).
- `docs/`: a page per role; `mac.md`, `router.md` and `checks.md`; `helix-keymap.md`, the index of
  the keymap's pages in `docs/helix-keymap/`.
- `tools/`: the gates' tools. `run` pins the binaries, `files` lists a gate's files, `sizecheck`
  caps them and `commits` checks the messages.
- `.githooks/`: the hooks that check each commit for the words of the local list and for secrets.
- `.github/workflows/`: `ci.yml`, with the jobs `lint`, `playbook` and `signoz`; `mac.yml`, with
  `mac`; and `fast-forward.yml`, which lands a pull request on a `/ff` comment.

## Commands

```sh
make lint                         # every gate, as CI's lint job runs it; -k goes past a failure
make vale FILES=README.md         # one gate alone; the gates that check files take FILES
make ansible-lint FILES=roles/base/tasks/main.yml   # this file alone: no syntax check of site.yml
make shell                        # ShellCheck and shfmt -d -i 4 on every shell script
make commits BASE=origin/BRANCH   # the messages from BRANCH, the one below in a stack, to HEAD
```

The gates are `shell`, `yamllint`, `ansible-lint`, `ruff`, `sizecheck`, `vale`, `lychee`, `rumdl`,
`actionlint`, `zizmor`, `gitleaks` and `commits` ([Checks](docs/checks.md#make-lint)). `make lint`
needs Python 3.12 or newer. It installs the Python tools and the collections into `.cache`, and
`tools/run` downloads the rest there.

The tests are CI's jobs, which run on each pull request ([Checks](docs/checks.md)). `playbook`
runs `bootstrap.sh` twice in an Ubuntu 26.04 container, and `signoz` forges SigNoz's compose files
again. `mac` sets up a Mac on GitHub's macOS runner. A run of the playbook sets up the machine that
runs it, as below.

The hooks check a commit for the local list's words. This checks any other text, such as a pull
request's body, and prints 0 where it holds none:

```sh
grep -ciwF -f <(sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e '/^$/d' -e '/^#/d' \
    ~/.config/ws/forbidden-words) FILE
```

## How changes land

- Pull requests are stacked, each on the one below it, and CI runs on each whatever its base.
- They land on `main` by fast-forward only, as the commits CI checked. GitHub's merge methods are
  all refused: never run `gh pr merge`.
- Rebase onto `main`, then comment `/ff` on the pull request
  (`.github/workflows/fast-forward.yml`). It pushes the head to `main` and deletes the branch.
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
- A pinned download is checked against its SHA-256 before use: otelcol's `.deb`
  ([group_vars/all.yml](group_vars/all.yml)), Docker's apt key
  ([its task](roles/docker/tasks/main.yml)), foundryctl ([ci.yml](.github/workflows/ci.yml)) and
  the gates' tools ([tools/run](tools/run)). Go and the JetBrains backends are checked against the
  SHA-256 their publishers list ([devtools](docs/devtools.md), [jetbrains](docs/jetbrains.md)).
- The collections are pinned in `requirements.yml`, and ansible-lint runs offline on those pins
  ([.ansible-lint](.ansible-lint)).
- foundryctl forges SigNoz's compose files, `roles/signoz/files/deployment/` and
  `casting.yaml.lock`, from `casting.yaml`. Change the casting and forge again, never the files by
  hand: CI's `signoz` job fails where they differ
  ([The compose files](docs/signoz.md#the-compose-files)).
- Secrets are never in this repository ([By hand](README.md#by-hand)).
- ws is public. No word of the local list, `~/.config/ws/forbidden-words`, goes into it. That
  holds for its files, the names of files and branches, the commit messages, and the text of pull
  requests and issues. The hooks in `.githooks` refuse a commit that holds one, and name the line,
  never the word. Never print the list or a word of it, and never pass `--no-verify`
  ([The hooks](docs/checks.md#the-hooks)).
- `make lint` fails on any finding, warnings included, in any line of any file
  ([make lint](docs/checks.md#make-lint); #47).
- Fix a finding, or justify it in place with its linter's own mechanism and a reason: ruff's and
  ansible-lint's `# noqa: CODE`, `# yamllint disable-line rule:NAME` or
  `# shellcheck disable=SCNNNN`, each with the reason beside it. Never lower a severity, add an
  exclusion or keep a baseline: every file passes every gate outright, and `.yamllint`,
  `.ansible-lint`, `ruff.toml` and `.vale.ini` are never weakened.
- The caps: files of 300 lines or fewer (`tools/sizecheck`), the README of 150 or fewer, sentences
  of 30 words or fewer, and prose wrapped at 100 columns ([writing.md](.claude/rules/writing.md)).

## Commits

Messages follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/), as
`git log` on `main` shows them, and `make commits` checks them ([tools/commits](tools/commits)).

- The header is `type(scope): description`, of 72 columns at most. The type is `feat`, `fix`,
  `build`, `chore`, `ci`, `docs`, `perf`, `refactor`, `style` or `test`.
- The scope, optional, is the role changed, such as `docker` or `signoz`, or `mac`; none for a
  change across the repository. The description is a short noun phrase, lower case but for proper
  nouns, with no trailing period: `feat(base): a swappiness of 10`.
- The body is a list of bullets wrapped at 72 columns: what was wrong or missing, what changed, and
  what was checked.
- A pull request is one commit, and its title is the commit's header. Its body starts `Part of #N.`
  or, on the one that completes issue N, `Closes #N.`, then `## What` and what was checked.
- Commits are signed, as the ruleset on `main` requires.

## Where else to look

- `.claude/rules/`: the roles' invariants in [roles.md](.claude/rules/roles.md), what each document
  holds in [docs.md](.claude/rules/docs.md), and the writing policy in
  [writing.md](.claude/rules/writing.md), loaded with the files they cover.
- Skills in `.claude/skills/`: `issue`, and `land`, run by hand only.
- Agents in `.claude/agents/`: `doc-editor`, which edits prose to the writing policy.
- Hooks in `.claude/hooks/`: the file's gates after each Edit or Write, and `make lint` as a turn
  ends ([Claude's hooks](docs/checks.md#claudes-hooks)).
