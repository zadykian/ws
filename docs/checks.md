# Checks

In a stack, CI runs `playbook`, `mac` and `mac-wizard`, its longest jobs, on the top pull request
alone, as it holds every change below it. `lint` and `signoz` run on each.

## make lint

`make lint` is the one gate: CI's lint job runs it, the lint plugin through `make lint-tree` as
Claude ends its turn, and the maintainer before a push ([Claude's hooks](#claudes-hooks)). Any
finding fails it, warnings included. A finding is fixed, or justified in place with its linter's
own inline mechanism and a reason. A Vale alert is always fixed, by a rewrite of the sentence, as
[the writing skill][writing] allows no exception to its rules. No gate lowers a severity or keeps
a baseline, and none leaves out a file written here. `make -k lint` runs every gate past a failure.
Each gate is its own target. Those that check files take `FILES`, as `make vale FILES=README.md`:

- `shell`: ShellCheck, which follows a script's sources with `-x`, and `shfmt -d -i 4` on every
  shell script, found by extension or shebang;
- `yamllint`: `--strict` (`.yamllint`) on the YAML files it finds, `host_vars/localhost.yml`
  too, which git ignores, so it starts with `---` ([The direct link](network.md#the-direct-link));
- `ansible-lint`: its production profile, strict and offline (`.ansible-lint`);
- `ruff`: `ruff check` and `ruff format --check` on the Python files, as `ruff.toml` sets them;
- `sizecheck`: 300 lines a file, for Markdown, YAML, shell, Python and Jinja templates;
- `vale`: the three Vale rules of [the writing skill][writing] on the Markdown files (`.vale.ini`);
- `lychee`: the links to files and their headings in the Markdown files, offline;
- `rumdl`: `.rumdl.toml`'s rules on Markdown: lines of 100 columns but in code blocks and tables;
- `actionlint` and `zizmor`: the workflows, zizmor with its online audits where `GH_TOKEN` is set;
- `gitleaks`: every commit, for secrets, or those that `LOG_OPTS` picks with git log's options.

`.editorconfig` sets the files' encoding, line ends and indents for editors.

`tools/run` downloads the other tools into `.cache/tools`, each pinned by version and SHA-256. make
installs the Python tools into `.cache/venv`, with a Python of 3.12 to 3.14, and the collections of
`requirements.yml` into `.cache/collections` ([Makefile](../Makefile)). pip takes only the wheels
that `.github/lint-requirements.txt` lists by SHA-256, which uv writes from its `.in`. So the tools
install where each package has a wheel: on Linux, on amd64 or arm64, and on macOS on arm64, as
cryptography has none for macOS on amd64.

## The playbook

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`, which
need a booted machine. It runs it twice, and the second run must change nothing. Then `nft -c`, with
the container's NET_ADMIN, checks the outbound table the play wrote and the users it names. The run
leaves out its own check under `--skip-tags systemd` ([Outbound](security.md#outbound)). Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM, nor for a JetBrains
backend: `jetbrains_backends: []`. Its board is not MSI's, so the `fans` role leaves its fans alone.
It gives no VPN profile, so the `ikeward` role does nothing.

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

## The Mac's wizard

A workflow of its own, `mac-wizard.yml`, runs `mac/wizard.sh` on GitHub's macOS runner, from a copy
of the checkout at `~/repository/ws`. `standin.sh` gives the wizard what a new Mac and a user would,
such as `mac.conf`'s answers and dash. Among them are the stand-in YubiKey of [The Mac](#the-mac),
and a `gh` logged in to GitHub, first on the wizard's `PATH`. Then
[checks.sh](../.github/mac-wizard/checks.sh) runs the cases beside it. The wizard must refuse an
unknown argument, root, and a `WS_HOOK` of `WS`, as the files script must. A run must stop where
another holds the lock, and leave it. A fresh `--check`, by sh and by dash, must find brew with its
directories taken off `PATH`. Like `setup.sh --check`, it must change nothing, and leave no lock or
temporary file. The job's `mac.conf` names no key for ws, so `--check` must show the YubiKey's
`.pub`s it would write. The first run, with that `PATH` too, must write them, add ws's to that
login, and install what `setup.sh` installs. It must drop the Mac's key, which `setup.sh` prints,
and leave no step of the tunnel by hand, as the Tunnel stage, below, takes it. A second run and
`--check` must print nothing, by sh and by dash. Last, without the login of the stand-in `gh`, a run
with no terminal must stop at gh's stage, naming `gh auth login`. With a login that lacks the
scopes, or the signing one alone, it must name `gh auth refresh`. `--check` must say it would run
each.

At the Mac key stage, with the YubiKey out, `--check` must say that a run would wait, and put
`setup.sh`'s stage after it, and a run must wait. With one from the factory then, a run with no
terminal must stop, naming ykman's commands. At a terminal of `script`'s, a run must make the keys
from ykman's answers typed. `--check` must name their slots, and two in must stop a run. With gh
logged out, a run must pass the stage once ws's agent lists the keys, and stop at gh's, and a second
say nothing of the stage. With a key out of ws's agent, a run must stop, naming its `.pub`, or pass
where launchd's agent, started again, loads
it. Without the agent, a run must stop with `ssh-add`'s own words. A line of `WS_IDENTITY` or
`WS_IDENTITY_AGENT`, empty too, must stop a run and `--check` before `setup.sh`.

`standin.sh` starts an `sshd` of its own as ws, at `127.0.0.1:22022`, where the job's `mac.conf`
leads, with its own host key. A fresh `--check` must show the Host key stage after `setup.sh`. The
first run must warn that ws faces the internet, and add ws's key to `known_hosts`, as `WS_HOST_KEY`
gives its fingerprint. Without that line, a run with no terminal must stop without `WS_HOST_KEY` and
with another fingerprint, and `--check` must show the line, or stop at a link, as a run must. With a
`mac.conf` of no way to ws, a run must stop, naming [The router](router.md). Where `WS_REMOTE`
doesn't answer, ws's address must come from `WS_FIRST_ADDRESS`, and the closing list name `WS_PORT`
and no lease's address. On a cloud VM, it must name `mac.conf`'s way in place of the lease.

Root logs in to that `sshd` by root's own `authorized_keys`. The cases fill it from the stand-in
`gh`'s keys, as Ubuntu's installer and the README's step do for ws. Each session runs `ws.sh`, which
logs the command. The wizard adds no key, so the first run must stop at the Root's key stage, naming
the Mac's key, and add none. The case then gives root the keys, and the wizard runs again. With
another key alone in root's file, the stop must show ssh's error. Then root's line for the key
carries a forced command, as cloud-init writes it. `--check` must say that a run would stop. A run
must stop, with no terminal and at a terminal of `script`'s, and change nothing. With the keys back
in root's file, a run must say nothing of the stage. With another host key under `ws`, a run must
stop, with `StrictHostKeyChecking no` in `~/.ssh/config` too. Neither the `sshd`'s second host key,
an ECDSA one, nor a `--check` under `StrictHostKeyChecking accept-new` may add a line to
`known_hosts`.

The wizard's remote script, `mac/wizard-remote.sh`, runs as root on that `sshd`, sent as the wizard
sends it. `ws.sh` roots its paths in a directory of its own, and puts the stand-ins of
[ws-standin.sh](../.github/mac-wizard/ws-standin.sh) for ws's programs first on its `PATH`. The
cases run its `status`, its full and tagged `run`s, and the files script,
`mac/wizard-remote-files.sh`; [41-remote.sh](../.github/mac-wizard/41-remote.sh) lists what each
must do.

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
run each. Where the key list fails otherwise, both must stop, naming gh's error. With the key off
the account, a run must print nothing where root's git config names no signing key. Else it must run
`bootstrap.sh --tags devtools`, which adds it. Without claude's login, a run must stop at claude on
ws, naming `claude auth login`, and `--check` must say it would run it. The next must print nothing.

The stand-in `bootstrap.sh --tags wireguard` makes ws's `wg0`, a `wireguard-go` on UDP 51820, with
the Mac's key in `mac.pub` as its peer. A fresh `--check` must show the Tunnel stage after ws's run.
The first run must write `mac.pub`, run that, give `WS_PUBLIC_KEY` `wg0`'s key, and load the daemon.
It must log in once through the tunnel, by stand-ins of `ssh` and `nc`, as `wg0` has no address. A
new key for `wg0` must replace `WS_PUBLIC_KEY`'s line, and a missing `mac.pub` come back. Without
`wg0`'s peer alone, a run must run it again. With nothing on UDP 51820, the closing list must name
the port. Where `sudo -n` fails, a run must print nothing, and without `tunnel.pub`, write it first.

With both logged out and the key off the account, a run with no terminal must open nothing. At a
terminal of `script`'s, a run must open `github.com/login/device`, by a stand-in `open`, and run
`gh auth login` in an `ssh -t` session. Then it must add the key, and run `claude auth login`
through `bash -lc` in another session. The stand-ins' logins take a terminal alone, and the next run
must print nothing.

The Hook on ws stage clones `WS_HOOK` by the stand-in `gh`, and runs its `apply.sh` as root over
plain ssh with the secret of a throwaway keychain's item on stdin. It must stop where the item is
missing or unreadable ([47-hook.sh](../.github/mac-wizard/47-hook.sh)). The Hook on the Mac stage
clones it on the Mac, sets its `core.hooksPath`, and copies ws's word list
([48-hook-mac.sh](../.github/mac-wizard/48-hook-mac.sh)). The Cable stage reads ws's ports from a
stand-in sysfs, and runs the roles for `WS_CABLE_PORT`. Off the LAN, a run and `--check` must skip
it, a cloud VM's naming its firewall ([49-cable.sh](../.github/mac-wizard/49-cable.sh)).

The cases give `mac.conf`'s other answers in the environment. With no terminal, a missing answer
must stop a run and `--check` within 60 s, naming its key. A bad one from the environment must stop
`--check`, with the form the key takes. `--check` must show the environment's answers as the lines
it would add to `mac.conf`, and the first run must add them at its end. Typed at a terminal of
`script`'s after bad ones, each asked again, the same answers must show the same lines, and empty
lines the defaults, for each target. After the first run, they must write the same `mac.conf`.

Last, `mac/bootstrap.sh`, run by `sh` with the clone's file on stdin, as `curl | sh` runs it, must
find the runner's Command Line Tools and brew with that `PATH`. It must leave the clone's commit and
files alone, set its `core.hooksPath` once, and run the wizard, which then prints nothing. So a
second run and `--check` must print nothing at all. Where `WS_CONF` names no file, the wizard under
`bootstrap.sh --check` must show the lines it would write there, from the answers in the
environment, and write none. In a home without a clone, `--check` must list the clone and the
wizard's stages, as `wizard.sh` runs them, and make nothing outside macOS's `Library`. Then its
functions must download Homebrew's pinned `.pkg`, pass its check of the SHA-256 and signer, and
refuse the `.pkg` cut short and another signer
([60-homebrew.sh](../.github/mac-wizard/60-homebrew.sh)).

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
`WS_LAN_ADDRESS` outside `WS_LAN`, and on the LAN's network or broadcast address. It must take
`WS_SSH_KEY` and `WS_GIT_KEY` of `a-z`, `0-9` and `-` alone, and stop where both name one key
([mac-conf-checks.sh](../.github/mac-conf-checks.sh)). With `WS_CABLE_SERVICE` empty, `setup.sh`
must print nothing. Set, it must run one `networksetup -setmanual`, a stand-in's, through sudo, and
then none. It must warn of a disabled service. It must stop on one networksetup lacks, or one with a
router outside the cable's `/30`. It must stop too on a `-setmanual` that fails or changes nothing,
and name what differs ([mac-setup-checks.sh](../.github/mac-setup-checks.sh)). The job's runs must
write the keys' `.pub`s in `~/.config/ws/keys`, each the stand-in's key, which a fresh `--check`
must not make. Without git's key, `--check` must name its slot and the commands that make it, and a
run with no terminal stop, naming them. At a terminal of `script`'s, a run must make it from the
management key and PIN typed, and warn that its `.pub` held another key. A key with a PIN or touch
policy, two slots of one name, no YubiKey and two must each stop `setup.sh`. A YubiKey from the
factory must draw a warning of its defaults, and stop `setup.sh` with firmware 5.4.3. Without ykman
and yubico-piv-tool, `--check` must name their install
([mac-yubikey-checks.sh](../.github/mac-yubikey-checks.sh)). `ws-agent`, run at a socket of its
own, must list both keys as it starts, and sign with each. With the stand-in's `ioreg`, it must
take them out at an unplug, and load them again at a plug, a replug, a second YubiKey and an
`ssh-add -D`. Stopped, it must take its `ssh-agent` and socket with it. The job's agent must list
both at `~/.ssh/ws-agent.sock` and sign with each, launchd's from `setup.sh`'s copy, and `setup.sh`
then print nothing. Each signature must run `ws-agent-confirm`, which counts it in its key's stamp
([mac-agent-checks.sh](../.github/mac-agent-checks.sh)). `ssh -G` must give
`ws` and `ws-ssh` the YubiKey's `.pub` and ws's agent, by three lines of `config.d/ws`. It must give
them the file or the agent that `WS_IDENTITY` and `WS_IDENTITY_AGENT` name, with `--check` showing
`config.d/ws`'s diff alone. Back on the YubiKey's key, `setup.sh` must warn once that it replaces
the other. Secretive's agent and its one key must leave ws its key, and a `.pub` of Secretive's in
`WS_IDENTITY` must take its agent. A `.pub` that is missing or unreadable, or holds a private key, a
certificate or a line that is no key, must stop `setup.sh` too. So must a key that ssh-keygen
cannot read or ws's OpenSSH lacks ([mac-identity-checks.sh](../.github/mac-identity-checks.sh)).
Without the stand-in app, `setup.sh` must install Gateway's cask once, through a stand-in brew that
refuses any other call, and then print nothing. With an app in `/Applications`, it must install
nothing. A failed install, or one that leaves no app, must draw a warning, and the run go on to the
tunnel. So must a cask that Homebrew has without its app, which neither a run nor `--check` may
install again ([mac-gateway-checks.sh](../.github/mac-gateway-checks.sh)). With `WS_GIT_NAME` and
`WS_GIT_EMAIL`, git must sign a commit with the YubiKey's key through the wrapper `ws-ssh-keygen`,
whatever `SSH_AUTH_SOCK` names, and verify it. `setup.sh` must stop on a `WS_GIT_SIGNING_KEY` that
is missing or unreadable, or a `.pub` that holds more than a public key. A `~/.gitconfig` must get
the include of `~/.config/ws/gitconfig` at its top, once, keep its own lines, and name its own
settings. An include below them, or a second one, must stop `setup.sh` before it writes a file, as
must a link that lacks the include. One whose file has it must pass. The settings of git's XDG file
must be named. With a key that an agent holds at Secretive's socket, `--check` must show the wrapper
and write none, and git sign whatever `SSH_AUTH_SOCK` names. `ws` must keep the YubiKey's key, as
Secretive's one key is git's. git's credential command for GitHub must be a stand-in gh's, which
must add the key to its login once. Logged out, gh must draw a warning at each run. Logged in,
`setup.sh` must warn of a 404, another error of gh's, and an add that leaves the account without the
key. It must add a key the account lacks. Where `PATH` has no gh, `setup.sh` must install Homebrew's
once, with git's keys alone ([mac-git-checks.sh](../.github/mac-git-checks.sh)). A `.pub` of
`~/.config/ws/keys` in `WS_GIT_SIGNING_KEY` must take the wrapper too. With the stand-in YubiKey
out, git must make no commit, and sign again once it comes back. Nor may a file key sign from an
agent of the case's own ([mac-git-yubikey-checks.sh](../.github/mac-git-yubikey-checks.sh)).
`ws-agent-confirm` must sign through the check's agents, with file keys under `ssh-add -c`
([mac-confirm-checks.sh](../.github/mac-confirm-checks.sh)). 50 signatures in a burst must pass with
one notice, though the notifier lives 5 s. Under a shell that takes descriptors above 9, the
askpass's pipe lies there. Each key must post its own, and 5 s on, the next count 50. A refused or
missing notifier must cost the notice alone, and a missing askpass the signature. While one never
ends, its key must start no second.

The job sets up a stand-in YubiKey before its first run. A SoftHSM2 token holds its keys, `ws`'s in
slot 82 and `git`'s in 83 ([yubikey-standin.sh](../.github/yubikey-standin.sh)). They are Ed25519,
so the mac jobs run on `macos-26`, whose ssh loads them through PKCS#11. The job's ws's agent is
launchd's where launchd has `gui/UID`, else a run of the stand-in's own, as a notice says. Each
change of a YubiKey restarts it ([yubikey-agent.sh](../.github/yubikey-agent.sh)).

## The rules

`.github/workflows/rules.yml` runs the checks of
[zadykian/toolkit](https://github.com/zadykian/toolkit/tree/main/actions) on each pull request.
They check its commits, size and title, its description, and the terms it adds, against the rules
of the toolkit's skills. Each warns and never blocks, so no ruleset requires it.

`.github/workflows/issues.yml` runs the toolkit's issue check as an issue opens or changes. It
warns where the title is no Conventional Commits header, or the body strays from the issue form's
fields. Each workflow's comments say why a newer event cancels an older run.

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
`make lint-file FILES=PATH` runs the gates of the file's kind on that file alone, as the target
lists them. As Claude ends its turn, `make lint-tree` runs `make lint`, but its gitleaks scans only
the branch's own commits, from `origin/main`.

Each target runs the work tree's own code, as root on the server. That is its `Makefile` and
`tools/`, the tools its pins name, and the Python that ansible-lint loads, such as the filter
plugins. ansible-lint's syntax check of `site.yml` runs the lookups in the playbook's YAML too. So
the plugin runs them only on a tree the maintainer vouches for, as [the vouch rule][vouch] says.
The gate code includes what [.claude/lint-gates](../.claude/lint-gates) lists.

[lint]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-hooks
[vouch]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-vouch-rule
[writing]: https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale
