# Checks

In a stack, CI runs `playbook`, `mac` and `mac-wizard`, its longest jobs, on the top pull request
alone, as it holds every change below it. `lint` and `signoz` run on each.

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
such as `mac.conf`'s answers, dash, and a `gh` logged in to GitHub, first on the wizard's `PATH`.
Then `checks.sh` runs the cases beside it in order, and stops at the first that fails
([checks.sh](../.github/mac-wizard/checks.sh)). The wizard must refuse an unknown argument and root.
A run must stop where another holds the lock, and leave it. A fresh `--check`, by sh and by dash,
must find brew with its directories taken off `PATH`. As `setup.sh --check` must, it must change
nothing, and it must leave no lock or temporary file. The job's `mac.conf` names `~/.ssh/id_ed25519`
as the ssh key for ws, which `--check` must show it would make. With no terminal and a `mac.conf`
that names none, a run must stop at the Mac key stage, naming `WS_IDENTITY`, and make nothing. The
first run, with that `PATH` too, must make that key, with no passphrase, add it to that login,
install what `setup.sh` installs. It must drop the Mac's key, which `setup.sh` prints, and leave
no step of the tunnel by hand, as the Tunnel stage, below, takes it. A second run
and `--check` must print nothing, by sh and by dash. Then a stand-in for Secretive's agent, an
`ssh-agent` at its socket, holds a P-256 key, whose `.pub` a `mac.conf` of its own names in
`WS_IDENTITY`. `--check` must print nothing of the Mac key stage while the agent lists the key. With
git's name, it must say that it would wait for git's key made in Secretive, unless
`WS_GIT_SIGNING_KEY` is empty. Once the agent lacks the key, a run and `--check` must stop at the
stage, naming the socket. Last, without the login of the stand-in `gh`, a run with no terminal must
stop at gh's stage, naming `gh auth login`. With a login that lacks the scopes, or the signing one
alone, it must name `gh auth refresh`. `--check` must say it would run each.

`standin.sh` starts an `sshd` of its own as ws, at `127.0.0.1:22022`, where the job's `mac.conf`
leads, with a host key made there. A fresh `--check` must show the Host key stage after `setup.sh`.
The first run must warn that ws faces the internet, and add ws's key to `known_hosts`, as
`WS_HOST_KEY` gives its fingerprint. Without that line, a run with no terminal must stop without
`WS_HOST_KEY` and with another fingerprint, and `--check` must show the line. With a `mac.conf`
of no way to ws, a run must stop, naming [The router](router.md). With one whose `WS_REMOTE`
doesn't answer, ws's address must come from `WS_FIRST_ADDRESS`.

Root logs in to that `sshd` by root's own `authorized_keys`. The cases fill it from the stand-in
`gh`'s keys, as Ubuntu's installer and the README's step do for ws. Each session runs `ws.sh`,
which logs the command. The wizard adds no key, so the first run must stop at the Root's key stage,
naming the Mac's key, and add none. The case then gives root the keys, and the wizard runs again.
Then root's line for the key carries a forced command, as cloud-init writes it. `--check` must say
that a run would stop. A run must stop, with no terminal and at a terminal of `script`'s, and change
nothing. With the keys back in root's file, a run must say nothing of the stage. With another host
key under `ws`, a run must stop, with `StrictHostKeyChecking no` in `~/.ssh/config` too. Neither
the `sshd`'s second host key, an ECDSA one, nor a `--check` under
`StrictHostKeyChecking accept-new` may add a line to `known_hosts`.

The wizard's remote script, `mac/wizard-remote.sh`, runs as root on that `sshd`, sent as the wizard
sends it. `ws.sh` roots its paths in a directory of its own, and puts the stand-ins of
[ws-standin.sh](../.github/mac-wizard/ws-standin.sh) for ws's programs first on its `PATH`. `status`
must print the stamps and the stand-ins' states. Those are `wg0`'s key, the Mac's key for it and as
its peer, gh's and claude's logins, Secure Boot that refuses the fans' driver, and `host_vars`'
outbound table. A full `run` must run `bootstrap.sh` with no arguments in a unit of its own, with
root's `HOME` and a UTF-8 locale. It must print the log, and stamp the clone's commit, 0 and `on`;
with `-e security_outbound=false`, `off`, and write the line of its `outbound`, which stops on a
link. A tagged run after it must start its own unit and leave that stamp. A run during another's
unit must follow it. A full run cut off must remove the stamp of the one before, so the next starts
a unit again. The files script, `mac/wizard-remote-files.sh`, must clone another repository by the
stand-in `gh`, whose `apply.sh` must run as root over `ssh`. Its stamp must follow, which `status`
must list, as `OWNER/REPO`. A clone again must leave it alone and remove the stamp, and one of
another owner's repository of that name stop. Its `words` must print git's hash of root's word list,
or `-` without one, and the list for that hash alone. Its `link` must list the wired ports, and
`link PORT` name one in `host_vars` once, which `status` names once the roles give `10.77.0.1`.

The wizard's stage ws's run must wait for Root's key under a fresh `--check`. The first run must run
the playbook through the remote script, and each run since must read ws's `status` alone, once, and
again after a run of the playbook. Without the stamp, a run whose unit is cut off must stop at the
stage, and the next must start a unit of its own. That run must end with what stays by hand: the
fans' driver and the router's forwards. A run while a unit runs, as a dropped ssh's, must follow
that unit. With `WS_TARGET=cloud`, a run must turn the outbound table off, and the next none.

`standin.sh` logs ws's gh and claude in, and gh's account lists ws's signing key, so no run may open
a session on ws for a login. A fresh `--check` must show gh on ws and claude on ws after ws's run.
Without gh's login, a run with no terminal must stop at gh on ws, naming `gh auth login`, and with a
login that lacks the `write:ssh_signing_key` scope, `gh auth refresh`. `--check` must say it would
run each. With the key off the account, a run must print nothing where root's git config names no
signing key. Otherwise it must run `bootstrap.sh --tags devtools`, which adds it. Without claude's
login, a run must stop at claude on ws, naming `claude auth login`, and `--check` must say it would
run it. The next run must print nothing.

The stand-in `bootstrap.sh --tags wireguard` makes ws's `wg0`, a `wireguard-go` on UDP 51820, with
the Mac's key in `mac.pub` as its peer. A fresh `--check` must show the Tunnel stage after ws's run.
The first run must write `mac.pub`, run that, give `WS_PUBLIC_KEY` `wg0`'s key, and load the daemon.
It must have a handshake, and log in to ws once, by a stand-in `ssh`, as `wg0` has no address.
With a new key for `wg0`, a run must replace `WS_PUBLIC_KEY`'s line, and without `mac.pub`, write
it again. Without `wg0`'s peer alone, as a failed tagged run leaves it, a run must run it again.
With nothing on UDP 51820, a run must stop within a minute, naming it, the forward and Background.

With both logged out and the key off the account, a run with no terminal must open nothing. At a
terminal of `script`'s, a run must open `github.com/login/device`, by a stand-in `open`, and run
`gh auth login` in an `ssh -t` session. Then it must add the key, and run `claude auth login`
through `bash -lc` in another session. The stand-ins' logins take a terminal alone, and the next run
must print nothing.

For the Hook on ws stage, `WS_HOOK` names a repository that the stand-in `gh` clones, whose
`apply.sh` logs its stdin's SHA-256, and `WS_HOOK_SECRET` an item of a throwaway keychain. Without
the item, a run with no terminal must stop, naming `security add-generic-password`. At a terminal of
`script`'s, it must add the item, typed at its prompts, and run `apply.sh` once over plain ssh. Its
stdin must be the secret, no terminal, and the next run print nothing. A failed run leaves no stamp.
For the Hook on the Mac stage, the stand-in `gh` on the Mac clones it, with `.githooks` and an
empty `Brewfile`. A run must clone it once, set its `core.hooksPath`, and name `gh auth setup-git`
by hand, unless git's config names gh's credential command. The next run must print nothing. The
stand-in ws's word list, of stand-in words, must go to the Mac once, mode 600, and none of it be
printed. Once the Mac's differs, `--check` must show no diff. For the Cable stage, ws's ports are a
stand-in sysfs. Undocked, a run must print nothing, one that acts name the dock, and `WS_CABLE_PORT`
end the wait. Docked, one with no terminal must stop, and one with `WS_CABLE_PORT` name the port,
run the roles, and stop at the service. Typed, an empty line must take the `169.254` one, and `ping`
answer. The next run must read no `link`, and one after ws's install or a failed roles' run name the
port again.

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
wizard's stages, as `wizard.sh` runs them, and make nothing outside macOS's `Library`. Then its
functions, all but its last line, must download Homebrew's pinned `.pkg` and pass its check of the
SHA-256 and signer. A run makes that check before the install, which the case never runs. The log
shows `pkgutil`'s chain. The check must refuse the `.pkg` cut short, and another signer.

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
[zadykian/toolkit](https://github.com/zadykian/toolkit/tree/main/actions) on each pull request.
They check its commits, size and title, its description, and the terms it adds, against the rules
of the toolkit's skills. Each warns and never blocks, so no ruleset requires it. A newer event of a
pull request cancels its older run, as the two would race to create each check's comment.

`.github/workflows/issues.yml` runs the toolkit's issue check as an issue opens or changes. It
warns where the title is no Conventional Commits header, or the body strays from the issue form's
fields. It takes a workflow of its own, as a pull request's event names no issue, and runs as
`main` has it. A newer event of an issue cancels its older run, as with a pull request.

The workflows pin the toolkit's actions, and `.claude/settings.json` its marketplace, to one
release. `.github/` holds that release's templates: the pull request template, and the issue form,
whose `config.yml` turns blank issues off. The toolkit's `tools/pin` moves both pins and writes the
templates, as its [Pin a release](https://github.com/zadykian/toolkit#pin-a-release) says. Each pin
writes them again, so a change to them goes in the toolkit.

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

[lint]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-hooks
[vouch]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-vouch-rule
[writing]: https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale
