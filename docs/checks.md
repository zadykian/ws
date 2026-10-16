# Checks

## make lint

`make lint` is the one gate: CI's lint job and Claude's Stop hook run it, and the maintainer runs
it before a push ([Claude's hooks](#claudes-hooks)). Any finding fails it, warnings included. A
finding is fixed, or justified in place with its linter's own inline mechanism and a reason. No
gate lowers a severity or keeps a baseline, and none leaves out a file written here.
`make -k lint` runs every gate past a failure. Each gate is a target of its own, and those that
check files take `FILES`, as in `make vale FILES=README.md`:

- `shell`: ShellCheck, which follows a script's sources with `-x`, and `shfmt -d -i 4` on every
  shell script, found by extension or shebang;
- `yamllint`: `yamllint --strict`, as `.yamllint` sets it;
- `ansible-lint`: its production profile, strict and offline (`.ansible-lint`);
- `ruff`: `ruff check` and `ruff format --check` on the Python files, as `ruff.toml` sets them;
- `sizecheck`: 300 lines a file, for Markdown, YAML, shell, Python and Jinja templates, and 150
  for the README;
- `vale`: Vale's three rules (`.vale.ini`) on the Markdown files: sentences of 30 words or fewer,
  no wordy phrases and no needless variants;
- `lychee`: the links to files and their headings in the Markdown files, offline;
- `rumdl`: rumdl's rules on the Markdown files, with lines of 100 columns but in code blocks and
  tables (`.rumdl.toml`);
- `actionlint` and `zizmor`: the workflows, zizmor with its online audits where `GH_TOKEN` is set;
- `gitleaks`: every commit of the history, for secrets, or those that `LOG_OPTS` picks with git
  log's options;
- `commits`: the messages from `BASE`, `origin/main` by default, to `HEAD`, as `tools/commits`
  says. On a push to `main`, CI's `BASE` is the commit `main` was at before.

`tools/sizecheck` leaves out the files nobody writes here. Those are foundryctl's compose files and
lock, which yamllint and ansible-lint leave out too, and Vale's vendored styles. Data files, such
as TOML, JSON and conf files, are of no kind it caps. `.editorconfig` sets the files' encoding,
line ends and indents for editors.

`tools/run` downloads the other tools into `.cache/tools`, each pinned by version and SHA-256. make
installs the Python tools of `.github/lint-requirements.txt` into `.cache/venv`, with a Python of
3.12 to 3.14, which ansible-core 2.20 supports. That file pins each package with the SHA-256 of
each of its files. pip takes only the wheels it lists, and builds no package. So the tools install
where each package has a wheel: on Linux, on amd64 or arm64, and on macOS on arm64, as cryptography
has none for macOS on amd64. uv writes the file from `.github/lint-requirements.in`, which says
how, and make stops where the file lacks a pin of the `.in`. make installs the collections of
`requirements.yml` into `.cache/collections`, as ansible-lint runs offline. It installs each again
where a file it comes from is newer.

## The playbook

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`,
which need a booted machine. It runs it twice, and the second run must change nothing. Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM. It installs no
JetBrains backend, `jetbrains_backends: []`, as it has no room for those either.

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

## The Mac

CI also sets up a Mac, on GitHub's macOS runner, in the order of [The Mac](mac.md). `mac/setup.sh`
without ws's key must make the Mac's key and load no daemon, while `ssh -G ws` takes the LAN's way.
With a stand-in's key it must load the daemon. Then a second run and `--check` must print nothing,
and `ssh -G` must give `ws` and `ws-ssh` their addresses. The stand-in for ws is a second
wireguard-go on the runner, with the Mac's key as its peer. It answers on loopback both as the LAN
and as `WS_REMOTE`, and `ws-tunnel status` must show a handshake on each in turn. Last, a network
that `ws_tunnel_routes` adds must be routed into the daemon's utun, and a domain `ws_resolver` adds
must show in `scutil --dns`. Where the runner can make no utun, the steps of the tunnel are skipped,
with a warning.

## The hooks

The hooks in `.githooks` check each commit before git makes it. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath "$PWD/.githooks"` in its checkout, as `bootstrap.sh`
does in its clone. The path is absolute, so that each worktree runs the checkout's hooks, never
those of the branch it holds, such as an outside pull request's. Where
`~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose added
lines, file names or message hold one. Where gitleaks is installed, they refuse a staged secret
too.

## Claude's hooks

`.claude/settings.json` has Claude Code run two hooks, both POSIX sh, with the functions of
`.claude/hooks/lib.sh`. Each runs the work tree's own code, as root on the server: its `Makefile`
and `tools/`, and the Python tools and collections its pins name. ansible-lint loads the tree's
`ansible.cfg`, inventory and Python too, such as the filter plugins. Its syntax check of
`site.yml` runs a lookup in the playbook's YAML, as in an import's name, and `python3 -m venv`
imports modules from the tree's root. So each hook runs that code only on a tree the maintainer
vouches for, where these three hold:

- each commit from `origin/main` to `HEAD` has a good signature, `%G?` of `G`, by the key that
  `gpg.ssh.allowedSignersFile` lists for the commit's author;
- the index matches `HEAD`, with no change staged and no conflict;
- the gates' code, as `lib.sh` lists it, matches `HEAD` in the work tree too, with no file of it
  untracked.

On any other tree, a hook runs no gate and shows the user a line. That holds for an outside pull
request fetched into a worktree, and for one merged, cherry-picked or checked out by path while the
change stays staged. The scripts themselves are the project checkout's, so an outside pull request
goes in a worktree, never there. The check reads what git records, so it lets two cases through:

- A commit made here carries the maintainer's signature, whether Claude, a rebase or a cherry-pick
  made it. A rebased outside commit keeps its author, so the check refuses it, unless its author's
  address is the maintainer's. An outside change committed here as the maintainer's own passes:
  read it first.
- The check sees an unstaged change in the gates' code alone, and not where git ignores, as in
  `.cache/`. So outside code may run where `git apply`, `git restore --source` or `git stash apply`
  writes it, or where `git reset` or `git restore --staged` unstages a merge or checkout of it. The
  maintainer's own edits look the same. Read an outside pull request at its own commits, never as a
  diff applied to a tree.

After each Edit or Write, `.claude/hooks/lint-file.sh` runs the gates of the file's kind on that
file alone:

- Markdown: `vale`, `lychee`, `rumdl` and `sizecheck`;
- YAML: `yamllint`, `ansible-lint` and `sizecheck`, and for a workflow `actionlint` and `zizmor`;
- shell, by extension or shebang: `shell` and `sizecheck`;
- Python: `ruff` and `sizecheck`; Jinja templates: `sizecheck`.

It passes over a file outside the repository, one git ignores, and one of a branch from before the
gates. It passes over a path with a character outside `[A-Za-z0-9._/@+-]` or a part starting with
`-` too, with a line to the user. Its findings go back to Claude, which fixes them. A staged
change, or an edit of the gates' code, stops the gates in its tree until a commit takes it.

It records the file's work tree too, in a file named by the session's id in `claude-lint/` of the
clone's git directory. As Claude ends its turn, `.claude/hooks/lint-branch.sh` runs `make lint` in
each recorded tree that the maintainer vouches for and that differs from `origin/main`. Then it
clears the record, so a turn that wrote no file runs nothing. Nor does a turn that wrote through
Bash alone, as with sed: `make lint` by hand, or CI, checks its changes. A tree whose gates' code
the turn edited and committed is linted then too. That run's gitleaks scans only the branch's own
commits, from `origin/main`.

A failure goes back to Claude, which fixes what the turn broke, and reports rather than fixes one
from before it. Where Claude stops again, `stop_hook_active` lets it. The hook then lints the trees
written since the failure, and shows the user what still fails.
