# Checks

## make lint

`make lint` is the one gate: CI's lint job runs it, and the maintainer runs it before a push. Any
finding fails it, warnings included. A finding is fixed, or justified in place with its linter's
own inline mechanism and a reason. No gate lowers a severity or keeps a baseline, and none leaves
out a file written here. `make -k lint` runs every gate past a failure. Each gate is a target of
its own, and those that check files take `FILES`, as in `make vale FILES=README.md`:

- `shell`: ShellCheck and `shfmt -d -i 4` on every shell script, found by extension or shebang;
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
- `gitleaks`: every commit of the history, for secrets;
- `commits`: the messages from `origin/main` to `HEAD`, as `tools/commits` says.

`tools/sizecheck` leaves out the files nobody writes here. Those are foundryctl's compose files and
lock, which yamllint and ansible-lint leave out too, and Vale's vendored styles. Data files, such
as TOML, JSON and conf files, are of no kind it caps. `.editorconfig` sets the files' encoding,
line ends and indents for editors.

`tools/run` downloads the other tools into `.cache/tools`, each pinned by version and SHA-256. make
installs the Python tools of `.github/lint-requirements.txt` into `.cache/venv`, with a Python of
3.12 or newer. It installs the collections of `requirements.yml` into `.cache/collections`, as
ansible-lint runs offline. It installs each again where the file it comes from is newer.

## The playbook

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`,
which need a booted machine. It runs it twice, and the second run must change nothing. Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM. It installs no
JetBrains backend, `jetbrains_backends: []`, as it has no room for those either.

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

## The Mac

CI also sets up a Mac, on GitHub's macOS runner, in the order of [The Mac](mac.md).
`mac/setup.sh` without ws's key must make the Mac's key and load no daemon, and with a stand-in's
key must load it. Then a second run and `--check` must print nothing, and `ssh -G` must give `ws`
and `ws-ssh` their addresses. The stand-in for ws is a second wireguard-go on the runner, with the
Mac's key as its peer. It answers on loopback both as the LAN and as `WS_REMOTE`, and
`ws-tunnel status` must show a handshake on each in turn. Last, a network that `ws_tunnel_routes`
adds must be routed into the daemon's utun, and a domain `ws_resolver` adds must show in
`scutil --dns`. Where the runner can make no utun, the steps of the tunnel are skipped, with a
warning.

## The hooks

The hooks in `.githooks` check each commit before git makes it. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath .githooks`, as `bootstrap.sh` does in its clone.
Where `~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose
added lines, file names or message hold one. Where gitleaks is installed, they refuse a staged
secret too.
