# Checks

In a stack, CI runs `playbook`, `rootless`, `mac` and `mac-cld`, its longest jobs, on
the top pull request alone, as it holds every change below it. `lint`, `install` and `signoz` run
on each. A stack opens as drafts, as a pull request opens before its link, and only a later event
names the stack. So these four skip a draft's `opened` run and start on `ready_for_review`, which
the other three skip. A newer push to a pull request cancels the runs of the push before it.
The checks of `rootless`, `mac-cld` and a Mac's servers have pages: [rootless](rootless.md),
[mac-cld](mac-cld.md) and [mac-servers](mac-servers.md). So have [the gates](../../.tools/README.md).

## The `playbook` job

CI also builds ws from the tree, never from a release, in an Ubuntu 26.04 container with no git, so
the build is `devel`. It copies the checkout to `/opt/ws`, root's alone, and
[`.tools/serve-build`](../../.tools/serve-build) points its pin at the build, served on loopback
in the release's place ([The pin](../../roles/server/ubuntu/frame/README.md#the-pin)). As
`zadykian` and as root, a cloud VM's user, `install.sh` installs the pinned mise and runs the
bootstrap on that tree in place, with `MISE_LOCKED=1` and install.sh's `MISE_DATA_DIR`
([bootstrap.sh](../bootstrap.sh)). The runs by hand alone set the variable, so the first
bootstrap fails at its guard where install.sh drops it. The bootstrap runs twice, the second from
root's home. The parts and units that need booted systemd skip and say so. `mise bootstrap plan
--detailed-exitcode` must exit 0 before the second bootstrap, as that run would mend what it
finds, and again after it. With the frame's line of `state/ws-binary` dropped, it must exit 1
([playbook-plan.sh](../playbook-plan.sh)): the check against a silent skip holds. A plan from
`/opt/ws`, which loads no role, must exit 1 too, and name the run directory. Then `nft -c`, with
the container's NET_ADMIN, checks the outbound table the run wrote and the users it names, and a
find that the user owns its home. The Zed wrapper's [checks](../zed-terminal-checks.sh) run on the
files it installed, and Fresh's on the config it loaded. The user's empty commit must sign and
verify. Root's leg has no table, and the run leaves out its own check
([Outbound](../../roles/server/ubuntu/security/README.md#outbound)). The default user's leg starts
with no values file, so its first run writes one, and root's leg writes its own. After both runs,
the file is root's at 600, and root's keeps its bytes ([The values
file](../../roles/server/ubuntu/frame/README.md#the-values-file)). In the container, base's swap
file is off, as the swap is the host's. With no NVIDIA GPU, `nvidia` checks its names alone, and
with no MSI board, `fans` leaves its fans alone. Without a VPN profile, `ikeward` does nothing.

The container is set up as a machine is, so that no step of CI warns. It carries `container`, as
systemd's container interface does, so procps leaves the kernel's read-only settings alone. dpkg
keeps the man pages, and chrony's log directory is there before its install. The steps run with
root's home, not the job's, which would reach the runner's own Docker client. Each checkout names
the branch of its `git init`, as a machine's git config does.

## The warnings check

[warnings-check.sh](../warnings-check.sh) fails where a bootstrap or a plan of `playbook` or
`rootless` prints a warning. It reads a run's log, or runs the command and shows its output as it
runs. A warning is a line that says warn, hint, notice, deprecated, error, cannot, unable, could
not or ignoring, as a word alone, or starts as apt's `W:` and `E:` do. mise's colors don't hide
one. The check names each line it finds and exits 1, and it keeps a plan's own exit code.

It allows a few lines where systemd has not booted, each named and justified in the script. They
come from what a container lacks: an init that tells a package's script the runlevel, and an
`/etc/resolv.conf` that Docker doesn't bind-mount. On the rootless job's booted runner, it allows
none.

## The install script

CI runs `install.sh` with mise's real binary, served on loopback, and a small repository for the
release's tree. It checks that `install.sh` installs mise, refuses a wrong one and runs the
bootstrap ([The install job](release.md#the-install-job)).

## The release

`release.yml` runs `ci.yml` and `mac.yml` on a tag before it publishes
([Cutting a release](release.md#cutting-a-release)).

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's, in `roles/server/ubuntu/signoz`.

## The Mac

Cases 46 to 87 read what the first `mise bootstrap` wrote. Case 46 checks
[Ghostty's files](../../roles/client/macos/ghostty/README.md#ghosttys-settings): `/etc/ws/ghostty`
must be `ws.ghostty` whole, and Ghostty's config must end with the line that reads it. Case 47
checks [Zed's files](../../roles/client/macos/zed/README.md): `global_settings.json` must be
`roles/client/macos/zed/files/`'s whole, a `settings.json` there must stay byte for byte through
another bootstrap, and a login shell must find the cask's `zed`. Case 48 checks `keymap.json`:
`roles/client/macos/zed/files/`'s whole, a foreign one left with the warning of its unit's read, and
ws's back once it goes. Case 87 checks [the zsh hook](../../roles/client/macos/ghostty/README.md#the-zsh-hook): `zsh -i` must
reset at each prompt, and the alternate screen after a 255 from `ssh` or from `cld --host` alone.

CI also sets up a Mac, by [its cases](../mac), in the order of [The
Mac](../../roles/client/README.md). A stand-in cld, which cld's unit leaves alone, spares the
job's runs its download. Then `mise -C /opt/ws/roles/client bootstrap` with
[the Mac's frame](../../roles/client/macos/frame/README.md) runs as the runbook runs it, on the
tree in `/opt/ws` with ws's build served as the pin ([mac-mise.sh](../mac-mise.sh)). It links the
tree's plugin as the runbook does, and makes `/opt` and `/usr/local/share` root's and 755, as on
a Mac, where the runner's image may differ. It sets up
ssh, git and cld too. Its plan must exit 0 after the first run, a second run must call no sudo,
and `/usr/local/bin/ws` must be the build. A spec whose unit no line names must fail the plan,
and so must a plan from `/opt/ws` with every unit in sync.
The job's `values.yml` names a file key, made in case 20, as ws's
login key and git's. Without ws's WireGuard key, the bootstrap must make the Mac's key, with
`/etc/ws/tunnel.pub` its public half, and load no daemon, and `ssh -G ws` gives the tunnel's end.
With a stand-in's key it must load and run the daemon, which runs `/usr/local/bin/ws`. A second
run must call no sudo, and its plan exit 0. The stand-in for ws is a second wireguard-go on the
runner, with the Mac's key as its peer. It answers on loopback both as the LAN and as
`WS_REMOTE`, and `ws client macos tunnel status` must show a handshake on each in turn. The daemon
must take the SIGHUPs for a new `tunnel.conf` and its `routes.d` in place, with the same pid. Then
a network that `ws client macos tunnel routes --server ws` adds must be routed into the daemon's
utun, and a domain `ws client macos resolver` adds must show in `scutil --dns`. `ws client macos
tunnel loaded`, `ws client macos tunnel routes` and `ws client macos resolver` must do the same,
with `--check` showing the diff and writing nothing ([mac-ws-checks.sh](../mac-ws-checks.sh)).
Next, the server's `WS_TUNNEL_NETWORKS` and `WS_RESOLVERS` must give a route and a resolver
through the bootstrap, each named by `ws client macos networks --check` first. The plan after it
must exit 0, and a fourth nameserver must be refused. Emptied or removed, the keys must take both
back, and spare a script's own ([mac-conf-checks.sh](../mac-conf-checks.sh)). With
`WS_CABLE_SERVICE` empty, `ws client macos cable` must print nothing. Set, it must run one
`networksetup -setmanual`, a stand-in's, through sudo, and then none. It must warn of a disabled
service. It must stop on one networksetup lacks, or one with a router outside the cable's `/30`.
It must stop too on a `-setmanual` that fails or changes nothing, and name what differs
([mac-setup-checks.sh](../mac-setup-checks.sh)).
The cases of the keys run `mise -C /opt/ws/roles/client bootstrap` with an edited `values.yml`, and
read the files it writes ([56-keys.sh](../mac/56-keys.sh)). A refusal must fail the bootstrap and
its plan with its reason, and change no file. `ssh -G` must give `ws` the file key that the job's
`values.yml` names, by its four lines, and the file or the agent that `WS_IDENTITY_PUB_KEY` and
`WS_IDENTITY_AGENT` name. An empty `WS_IDENTITY_PUB_KEY` must stop, though an agent at Secretive's
socket holds a P-256 key. It gives the line that names each P-256 key by ws's copy of its `.pub`.
`PublicKeys` is closed, as macOS 27 denies ws the files of Secretive's container. With that line,
`ssh -G` must give `ws` the agent and the copy, which a run writes where none exists. A `.pub` that
is missing or unreadable must stop too. So must one that holds a private key, a certificate, a line
that is no key, or a key that ws's OpenSSH lacks
([mac-identity-checks.sh](../mac-identity-checks.sh)). `/usr/local/bin/gh` must be mise's gh, root's
and 0755, in place of one there, and a bootstrap that changes nothing must call no sudo. With a file
key, git must sign a commit and verify it, with gh as GitHub's credential command. A `~/.gitconfig`
must get the include of `/etc/ws/gitconfig` at its top, once, and keep its own lines, its mode and a
copy. An include below them, a second one or a link must stop the bootstrap, as must a signing key
that is empty, missing or more than a public key. With a key of Secretive's, git must sign through
ws's wrapper whatever `SSH_AUTH_SOCK` names, and `ws` keep the file key
([mac-git-checks.sh](../mac-git-checks.sh)). Last, after a second server's cases, one more bootstrap
must call no sudo, and its plan and `mise dot status --missing` must exit 0
([96-converged.sh](../mac/96-converged.sh)). Where wireguard-go makes no utun, the job fails, as the
daemon's device makes a utun too.

## The rules

`.github/workflows/rules.yml` runs the checks of
[zadykian/toolkit](https://github.com/zadykian/toolkit/tree/main/actions) on each pull request.
They check its commits, size and title, its description, and the terms it adds, against the rules
of the toolkit's skills. Each warns and never blocks, so no ruleset requires it. A newer event of a
pull request cancels its older run, as the two would race to create each check's comment.

`.github/workflows/issues.yml` runs the toolkit's issue check as an issue opens or changes. It
warns where the title is no Conventional Commits header, or the body strays from the issue form's
fields. It takes a workflow of its own, as a pull request's event names no issue, and runs as
`main` has it. A newer event cancels its older run, as with a pull request.

The workflows pin the toolkit's actions, and `.claude/settings.json` its marketplace, to one
release. `.github/` holds that release's templates: the pull request template, and the issue form,
whose `config.yml` turns blank issues off. The toolkit's `tools/pin` moves both pins and writes the
templates, as its [Pin a release](https://github.com/zadykian/toolkit#pin-a-release) says, so a
change to them goes in the toolkit.

## The hooks

The hooks in `.githooks` check each commit before git makes it. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath "$PWD/.githooks"` in its checkout. The path is
absolute, so that each worktree runs the checkout's hooks, never those of the branch it holds, such
as an outside pull request's. Where
`~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose added
lines, file names or message hold one. Case aside, a word matches where no letter or digit touches
it: `_` separates words, as `-` does. Where gitleaks is installed, they refuse a staged secret too.
The list is a file placed by hand, mode 600, with the lines of the secret `FORBIDDEN_WORDS`.

## The `words` job

`.github/workflows/words.yml` checks the list's words, from the secret `FORBIDDEN_WORDS`, in a
pull request's commits (added lines, file names, messages), its branch's name, title and body.
On an issue it checks the title and body. It fails on an empty secret. `main` requires it.

It runs on a pull request from a branch of this repository and on an issue the owner opened. Other
text skips, as the run's public result would show whether a guessed word is listed. A skipped job
passes a required check. The trigger is never `pull_request_target`. The log names where a word is,
such as `message line N` or `body line N`, never a word, a path or a ref's name. No step puts the
text in `env:` or checks out, as both print it. `words-check-test.sh` tries the script.

## Claude's hooks

The lint plugin of the `zadykian` marketplace, which `.claude/settings.json` enables, has Claude
Code run two targets of the `Makefile`, as [its docs][lint] say. After each edit,
`make lint-file FILES=PATH` runs the gates of the file's kind on that file alone
([The gates of one file](../../.tools/README.md#the-gates-of-one-file)).

As Claude ends its turn, `make lint-tree` runs `make lint`, but its gitleaks scans only the
branch's own commits, from `origin/main`.

Each target runs the work tree's own code, as root on the server. That is its `Makefile` and
`.tools/` and the tools its pins name. So the plugin runs them only on a tree the maintainer
vouches for, as [the vouch rule][vouch] says.
The gate code includes what [.claude/lint-gates](../../.claude/lint-gates) lists.

[lint]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-hooks
[vouch]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-vouch-rule
