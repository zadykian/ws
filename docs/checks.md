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
- `yamllint`: `yamllint --strict`, as `.yamllint` sets it, on the YAML files it finds itself.
  Those include `host_vars/localhost.yml`, which git ignores, so that file starts with `---`
  ([The direct link](network.md#the-direct-link));
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

## The Mac's wizard

A workflow of its own, `mac-wizard.yml`, runs `mac/wizard.sh` on GitHub's macOS runner, from a copy
of the checkout at `~/repository/ws`. `standin.sh` gives the wizard what a new Mac and a user would,
such as `mac.conf`'s answers and dash. Then `checks.sh` runs the cases beside it in order, and stops
at the first that fails ([checks.sh](../.github/mac-wizard/checks.sh)). The wizard must refuse an
unknown argument and root. A run must stop where another holds the lock, and leave it. A fresh
`--check`, by sh and by dash, must find brew with its directories taken off `PATH`. As
`setup.sh --check` must, it must change nothing, and it must leave no lock or temporary file. The
job's `mac.conf` names `~/.ssh/id_ed25519` as the ssh key for ws, which `--check` must show it would
make. With no terminal and a `mac.conf` that names none, a run must stop at the Mac key stage,
naming `WS_IDENTITY`, and make nothing. The first run, with that `PATH` too, must make that key,
with no passphrase, install what `setup.sh` installs, and load no daemon without `WS_PUBLIC_KEY`. It
must drop the Mac's key, which `setup.sh` prints until then, and end with that step as one left by
hand. A second run and `--check` must print nothing, by sh and by dash. Then a stand-in for
Secretive's agent, an `ssh-agent` at its socket, holds a P-256 key, whose `.pub` a `mac.conf` of its
own names in `WS_IDENTITY`. `--check` must print nothing of the Mac key stage while the agent lists
the key. With git's name, it must say that it would wait for git's key made in Secretive, unless
`WS_GIT_SIGNING_KEY` is empty. Once the agent lacks the key, a run and `--check` must stop at the
stage, naming the socket.

The cases give `mac.conf`'s other answers in the environment. With no terminal, a missing answer
must stop a run and `--check` within 60 s, naming its key. A bad one from the environment must stop
`--check`, with the form the key takes. `--check` must show the environment's answers as the lines
it would add to `mac.conf`, and the first run must add them at its end. Typed at a terminal of
`script`'s after bad ones, each asked again, the same answers must show the same lines, and empty
lines the defaults. After the first run, they must write the same `mac.conf`.

Last, `mac/bootstrap.sh`, run by `sh` with the clone's file on stdin, as `curl | sh` runs it, must
find the runner's Command Line Tools and brew with that `PATH`. It must leave the clone's commit and
files alone, set its `core.hooksPath` once, and run the wizard, which then prints nothing. So a
second run and `--check` must print nothing at all. Where `WS_CONF` names no file, the wizard under
`bootstrap.sh --check` must show the lines it would write there, from the answers in the
environment, and write none. In a home without a clone, `--check` must list the clone and the
wizard's stages, as `wizard.sh` runs them, and make nothing. Then its functions, all but its last
line, must download Homebrew's pinned `.pkg` and pass its check of the SHA-256 and signer. A run
makes that check before the install, which the case never runs. The log shows `pkgutil`'s chain. The
check must refuse the `.pkg` cut short, and another signer.

## The Mac

CI also sets up a Mac, on GitHub's macOS runner, in the order of [The Mac](mac.md). First,
`mac/setup.sh --check` must change nothing: it writes no file, keeps no copy of `~/.ssh/config` and
installs no formula or cask. It must note the install of JetBrains Gateway's cask. A stand-in
`JetBrains Gateway.app`, an empty directory in `~/Applications`, then spares the job's runs the
download. Without ws's key, `setup.sh` must make the Mac's key and load no daemon, while
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
and name what differs ([mac-setup-checks.sh](../.github/mac-setup-checks.sh)). `ssh -G` must give
`ws` and `ws-ssh` the file key, by the four lines `config.d/ws` had for it before. It must give
them the file or the agent that `WS_IDENTITY` and `WS_IDENTITY_AGENT` name, with `--check`
showing `config.d/ws`'s diff alone. Then it must give them an agent at Secretive's socket, which
holds a P-256 key whose `.pub` is in `PublicKeys`, where those keys are empty. It must warn that
this replaces the file key. A `-cert.pub`, a name with a blank or a directory there must not count.
A second `.pub` must stop `setup.sh`. Where `WS_GIT_SIGNING_KEY` names one of the two, ws's key
must be the other. The file key's `.pub` in `WS_IDENTITY` must keep the file key. A `.pub` that is
missing or unreadable, or holds a private key, a certificate or a line that is no key, must stop
`setup.sh` too. So must a key that ssh-keygen cannot read or ws's OpenSSH lacks
([mac-identity-checks.sh](../.github/mac-identity-checks.sh)).
Without the stand-in app, `setup.sh` must install Gateway's cask once, through a stand-in brew that
refuses any other call, and then print nothing. With an app in `/Applications`, it must install
nothing. A failed install, or one that leaves no app, must draw a warning, and the run go on to the
tunnel. So must a cask that Homebrew has without its app, which neither a run nor `--check` may
install again ([mac-gateway-checks.sh](../.github/mac-gateway-checks.sh)). With `WS_GIT_NAME`
and `WS_GIT_EMAIL`, it must make git's signing key once, an ed25519 key with no passphrase. It must
stop on a `WS_GIT_SIGNING_KEY` that is missing or unreadable, or a `.pub` that holds more than a
public key. So must the default key or its `.pub` alone. git must then sign a commit and verify it.
A `~/.gitconfig` must get the include of `~/.config/ws/gitconfig` at its top, once, keep its own
lines, and name its own settings. An include below them, or a second one, must stop `setup.sh`
before it writes a file, as must a link that lacks the include. One whose file has it must pass. The
settings of git's XDG file must be named. With a key that an agent holds at Secretive's socket,
`--check` must show the wrapper and write none, and git sign whatever `SSH_AUTH_SOCK` names. `ws`
must keep the file key, as Secretive's one key is git's. git's credential command for GitHub
must be a stand-in gh's, which must add the key to its login once. Logged out, gh must
draw a warning at each run. Logged in, `setup.sh` must warn of a 404 or another error of
gh's, and of an add that leaves the account without the key. It must add the key where the
account lacks it, and `--check` name the add before the key is made. Where `PATH` has no gh,
`setup.sh` must install Homebrew's once, and only with git's keys
([mac-git-checks.sh](../.github/mac-git-checks.sh)). Where wireguard-go makes no utun, the job
fails, as the daemon runs the same wireguard-go.

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
