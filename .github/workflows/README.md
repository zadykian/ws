# Checks

In a stack, CI runs `server`, `client` and `client-cld`, its longest jobs, on
the top pull request alone, as it holds every change below it, and only where the stack rests on
`main`. `install` and `signoz` run on each pull request of such a stack, and `lint` on every pull
request. So a pull request onto another branch, such as an integration branch, takes `lint` alone
of the six: that branch's pull request to `main` runs the rest.
A stack opens as drafts, as a pull request opens before its link, and only a later event
names the stack. So the longest three skip a draft's `opened` run and start on `ready_for_review`,
which `lint` skips. `install` and `signoz` start there too above a stack's bottom, and skip it
where they ran as the pull request opened. A newer push to a pull request cancels the runs of
the push before it.
The checks of `server`, `client-cld` and a Mac's servers have pages: [server](server.md),
[client-cld](client-cld.md) and [client-servers](client-servers.md). So have [the gates](../../.tools/README.md).

## The warnings check

[warnings-check.sh](../warnings-check.sh) fails where a bootstrap or a plan of `server`
prints a warning, the secrets task's runs and plans too. It reads a
run's log, or runs the command and shows its output as it runs. A warning is a line that says
warn, hint, notice, deprecated, error, cannot, unable, could not or ignoring, as a word alone, or
starts as apt's `W:` and `E:` do. mise's colors don't hide one. The check names each line it
finds and exits 1. A plan's exit 2, its answer, turns to 1 as well, but a command that fails
keeps its own exit code.

It allows no line. Its test, [warnings-check-test.sh](../warnings-check-test.sh), runs in
`lint`, on a log with warnings and on one without.

## The install script

CI runs `install.sh` with a stub of ws for each target, served on loopback. It checks that
`install.sh` installs each stub, refuses a wrong sum and a missing file, and makes and checks
`/usr/local/bin`. It runs `ws update` too, with a real build of the tree as the ws that runs. The
update must move a build to a release and run its setup, and the autoinstall file must fill and
pass cloud-init's schema ([The install job](release.md#the-install-job)). The `client` job runs it on macOS with
the tree's build ([16-install.sh](../client/16-install.sh)).

## The release

`release.yml` runs `ci.yml` and `client.yml` on a tag before it publishes
([Cutting a release](release.md#cutting-a-release)).

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's, in `roles/server/ubuntu/signoz`.

## The Mac

Cases 46 to 87 read what the first `mise bootstrap` wrote. Case 46 checks
[Ghostty's files](../../macos/roles/ghostty/README.md#ghosttys-settings): `/etc/ws/ghostty`
must be `ws.ghostty` whole, and Ghostty's config must end with the line that reads it. Case 47
checks [Zed's files](../../macos/roles/zed/README.md): `global_settings.json` must be
`macos/roles/zed/files/`'s whole, a `settings.json` there must stay byte for byte through
another bootstrap, and a login shell must find the cask's `zed`. Case 48 checks `keymap.json`:
`macos/roles/zed/files/`'s whole, a foreign one left with the warning of its unit's read, and
ws's back once it goes. Case 58 checks [the secrets' tools](../../macos/roles/secrets/README.md):
`/usr/local/bin/fnox` and `age-plugin-yubikey` must be mise's, root's and 0755, at their pins. A
stand-in in place of each must give way to mise's at the next bootstrap, whose plan then changes
nothing. fnox must encrypt a value to the Nano's recipient with no YubiKey, and read it back by an
SSH key that is a recipient too. Case 86 checks [.NET's SDKs](../../macos/roles/dotnet/README.md):
`dotnet` in `/usr/local/share/dotnet` must be root's and list 10.0.404 and 11.0.100 alone, and
`/etc/paths.d/dotnet` must name that directory. Case 87 checks [the zsh hook](../../macos/roles/ghostty/README.md#the-zsh-hook): `zsh -i` must
reset at each prompt, and the alternate screen after a 255 from `ssh` or from `cld --host` alone.

CI also sets up a Mac, by [its cases](../client), in the order of [The
Mac](../../macos/README.md). A stand-in cld, which cld's unit leaves alone, spares the
job's runs its download. Then `ws macos setup --in-place` with
[the Mac's frame](../../macos/roles/frame/README.md) runs as the runbook runs it, on the
tree in `/opt/ws`, with ws's build in `/usr/local/bin` ([client-mise.sh](../client-mise.sh)). Its plan
before the first run must exit 2. Setup installs mise and links the tree's plugin as the runbook
does. The case makes `/opt` and `/usr/local/share` root's and 755 first, as on
a Mac, where the runner's image may differ. It sets up
ssh, git and cld too. The plan must exit 0 after the first run, a second run must call no sudo,
and `/usr/local/bin/ws` must be root:wheel at 755. A spec whose unit no line names must fail the plan,
and so must a plan from `/opt/ws` with every unit in sync. After the first run the firewall must
be on and block all incoming ([the Mac's firewall](../../macos/roles/firewall/README.md)).
The cases after it, whose stand-ins listen on loopback, pass unchanged under it.
The job's `values.yml` names a file key, made in case 20, as ws's
login key and git's. Without ws's WireGuard key, the bootstrap must make the Mac's key, with
`/etc/ws/tunnel.pub` its public half, and load no daemon, and `ssh -G ws` gives the tunnel's end.
With a stand-in's key it must load and run the daemon, which runs `/usr/local/bin/ws`. A second
run must call no sudo, and its plan exit 0. The stand-in for ws is a second wireguard-go on the
runner, with the Mac's key as its peer. It answers on loopback both as the LAN and as
`WS_REMOTE`, and `ws macos tunnel status` must show a handshake on each in turn. The daemon
must take the SIGHUPs for a new `tunnel.conf` and its `routes.d` in place, with the same pid. Then
a network that `ws macos tunnel routes --server ws` adds must be routed into the daemon's
utun, and a domain `ws macos resolver` adds must show in `scutil --dns`. `ws macos
tunnel loaded`, `ws macos tunnel routes` and `ws macos resolver` must do the same,
with `--check` showing the diff and writing nothing ([client-ws-checks.sh](../client-ws-checks.sh)).
Next, the server's `WS_TUNNEL_NETWORKS` and `WS_RESOLVERS` must give a route and a resolver
through the bootstrap, each named by `ws macos networks --check` first. The plan after it
must exit 0, and a fourth nameserver must be refused. Emptied or removed, the keys must take both
back, and spare a script's own ([client-conf-checks.sh](../client-conf-checks.sh)). With
`WS_CABLE_SERVICE` empty, `ws macos cable` must print nothing. Set, it must run one
`networksetup -setmanual`, a stand-in's, through sudo, and then none. It must warn of a disabled
service. It must stop on one networksetup lacks, or one with a router outside the cable's `/30`.
It must stop too on a `-setmanual` that fails or changes nothing, and name what differs
([client-setup-checks.sh](../client-setup-checks.sh)).
The cases of the keys run mise's bootstrap with an edited `values.yml`, and
read the files it writes ([56-keys.sh](../client/56-keys.sh)). A refusal must fail the bootstrap and
its plan with its reason, and change no file. With the server's `WS_LAN_ADDRESS` dropped, the plan
must fail and name it and its file, as the frame's check of the values does
([89-values-check.sh](../client/89-values-check.sh)). `ssh -G` must give `ws` the file key that the job's
`values.yml` names, by its four lines, and the file or the agent that `WS_IDENTITY_PUB_KEY` and
`WS_IDENTITY_AGENT` name. An empty `WS_IDENTITY_PUB_KEY` must stop, though an agent at Secretive's
socket holds a P-256 key. It gives the line that names each P-256 key by ws's copy of its `.pub`.
`PublicKeys` is closed, as macOS 27 denies ws the files of Secretive's container. With that line,
`ssh -G` must give `ws` the agent and the copy, which a run writes where none exists. A `.pub` that
is missing or unreadable must stop too. So must one that holds a private key, a certificate, a line
that is no key, or a key that ws's OpenSSH lacks
([client-identity-checks.sh](../client-identity-checks.sh)). `/usr/local/bin/gh` must be mise's gh, root's
and 0755, in place of one there, and a bootstrap that changes nothing must call no sudo. With a file
key, git must sign a commit and verify it, with gh as GitHub's credential command. A `~/.gitconfig`
must get the include of `/etc/ws/gitconfig` at its top, once, and keep its own lines, its mode and a
copy. An include below them, a second one or a link must stop the bootstrap, as must a signing key
that is empty, missing or more than a public key. With a key of Secretive's, git must sign through
ws's wrapper whatever `SSH_AUTH_SOCK` names, and `ws` keep the file key
([client-git-checks.sh](../client-git-checks.sh)). Last, after a second server's cases, one more bootstrap
must call no sudo, and its plan and `mise dot status --missing` must exit 0
([96-converged.sh](../client/96-converged.sh)). Where wireguard-go makes no utun, the job fails, as the
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
