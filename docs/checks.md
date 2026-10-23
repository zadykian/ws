# Checks

## make lint

`make lint` is the one gate: CI's lint job runs it, the lint plugin through `make lint-tree` as
Claude ends its turn, and the maintainer before a push ([Claude's hooks](#claudes-hooks)). Any
finding fails it, warnings included. A finding is fixed, or justified in place with its linter's
own inline mechanism and a reason. A Vale alert is always fixed, by a rewrite of the sentence, as
[the writing skill][writing] allows no exception to its rules. No gate lowers a severity or keeps
a baseline, and none leaves out a file written here.
`make -k lint` runs every gate past a failure. Each gate is a target of its own, and those that
check files take `FILES`, as in `make vale FILES=README.md`:

- `shell`: ShellCheck, which follows a script's sources with `-x`, and `shfmt -d -i 4` on every
  shell script, found by extension or shebang;
- `yamllint`: `yamllint --strict`, as `.yamllint` sets it;
- `ansible-lint`: its production profile, strict and offline (`.ansible-lint`);
- `ruff`: `ruff check` and `ruff format --check` on the Python files, as `ruff.toml` sets them;
- `sizecheck`: 300 lines a file, for Markdown, YAML, shell, Python and Jinja templates;
- `vale`: Vale's three rules (`.vale.ini`) on the Markdown files: sentences of 30 words or fewer,
  no wordy phrases and no needless variants;
- `lychee`: the links to files and their headings in the Markdown files, offline;
- `rumdl`: rumdl's rules on the Markdown files, with lines of 100 columns but in code blocks and
  tables (`.rumdl.toml`);
- `actionlint` and `zizmor`: the workflows, zizmor with its online audits where `GH_TOKEN` is set;
- `gitleaks`: every commit of the history, for secrets, or those that `LOG_OPTS` picks with git
  log's options.

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
which need a booted machine. It runs it twice, and the second run must change nothing. Then
`nft -c`, with the container's NET_ADMIN, checks the outbound table the play wrote and the users
it names. The run leaves out its own check under `--skip-tags systemd`
([Outbound](security.md#outbound)). Its swap file is 64 MiB, as the runner's disk has no room for
one the size of its RAM. It installs no JetBrains backend, `jetbrains_backends: []`, as it has no
room for those either. Its board is not MSI's, so the `fans` role leaves its fans alone. It gives
no VPN profile, so the `ikeward` role does nothing.

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

## The Mac

CI also sets up a Mac, on GitHub's macOS runner, in the order of [The Mac](mac.md). First,
`mac/setup.sh --check` must change nothing: it writes no file, keeps no copy of `~/.ssh/config` and
installs no formula. Without ws's key, `setup.sh` must make the Mac's key and load no daemon, while
`ssh -G ws` takes the LAN's way. The public key it prints must be the Mac's, as must the key in the
command that gives it to ws. With a stand-in's key it must load the daemon, which must run. Then a
second run and `--check` must print nothing, and `ssh -G` must give `ws` and `ws-ssh` their
addresses. The stand-in for ws is a second wireguard-go on the runner, with the printed key as its
peer. It answers on loopback both as the LAN and as `WS_REMOTE`, and `ws-tunnel status` must show a
handshake on each in turn. The daemon must take the SIGHUPs for a new `tunnel.conf` and `routes.d`
in place, with the same pid. Then a network that `ws_tunnel_routes` adds must be routed into the
daemon's utun, and a domain `ws_resolver` adds must show in `scutil --dns`
([mac-lib-checks.sh](../.github/mac-lib-checks.sh)). Next, `mac.conf`'s `WS_TUNNEL_NETWORKS` and
`WS_RESOLVERS` must give a route and a resolver, each shown by `--check` first. A second run and
`--check` must print nothing, and a fourth nameserver must be refused. Emptied or removed, the keys
must take both back, and spare a script's own. Last, `ws_conf` must read an empty value in
`mac.conf` as its key's default, and still stop on a value of the wrong form. It must also stop on a
`WS_LAN_ADDRESS` outside `WS_LAN`, and on the LAN's network or broadcast address
([mac-conf-checks.sh](../.github/mac-conf-checks.sh)). With `WS_CABLE_SERVICE` empty, `setup.sh`
must print nothing. Set, it must run one `networksetup -setmanual`, a stand-in's, through sudo, and
then none. It must warn of a disabled service. It must stop on one networksetup lacks, or one with a
router outside the cable's `/30`. It must stop too on a `-setmanual` that fails or changes nothing,
and name what differs ([mac-setup-checks.sh](../.github/mac-setup-checks.sh)). Where wireguard-go
makes no utun, the job fails, as the daemon runs the same wireguard-go.

## The rules

`.github/workflows/rules.yml` runs the checks of
[zadykian/actions](https://github.com/zadykian/actions) on each pull request. They check its
commits, size and title, its description, and the terms it adds, against the rules of
[zadykian/skills](https://github.com/zadykian/skills). Each warns and never blocks, so no ruleset
requires it.

## The hooks

The hooks in `.githooks` check each commit before git makes it. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath "$PWD/.githooks"` in its checkout, as `bootstrap.sh`
does in its clone. The path is absolute, so that each worktree runs the checkout's hooks, never
those of the branch it holds, such as an outside pull request's. Where
`~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose added
lines, file names or message hold one. Case aside, a word matches where no letter or digit touches
it: `_` separates words, as `-` does. Where gitleaks is installed, they refuse a staged secret too.

## Claude's hooks

The lint plugin of the `zadykian` marketplace, which `.claude/settings.json` enables, has Claude
Code run two targets of the `Makefile`, as [its docs][lint] say. After each edit,
`make lint-file FILES=PATH` runs the gates of the file's kind on that file alone:

- Markdown: `vale`, `lychee`, `rumdl` and `sizecheck`;
- YAML: `yamllint`, `ansible-lint` and `sizecheck`, and for a workflow `actionlint` and `zizmor`;
- shell, by extension or shebang: `shell` and `sizecheck`;
- Python: `ruff` and `sizecheck`; Jinja templates: `sizecheck`.

As Claude ends its turn, `make lint-tree` runs `make lint`, but its gitleaks scans only the
branch's own commits, from `origin/main`.

Each target runs the work tree's own code, as root on the server. That is its `Makefile` and
`tools/`, the tools its pins name, and the Python that ansible-lint loads, such as the filter
plugins. ansible-lint's syntax check of `site.yml` runs the lookups in the playbook's YAML too. So
the plugin runs them only on a tree the maintainer vouches for, as [the vouch rule][vouch] says.
The gate code includes what [.claude/lint-gates](../.claude/lint-gates) lists.

[lint]: https://github.com/zadykian/skills/blob/main/docs/lint.md#the-hooks
[vouch]: https://github.com/zadykian/skills/blob/main/docs/lint.md#the-vouch-rule
[writing]: https://github.com/zadykian/skills/blob/main/skills/writing/SKILL.md#the-caps-and-vale
