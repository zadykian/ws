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
the build is `devel`. It runs `ws server ubuntu apply` twice from root's home, as `zadykian` and as
root, a cloud VM's user. The steps that need booted systemd skip and say so, and the second run's
last line must be `changed: none`. Then
`nft -c`, with the container's NET_ADMIN, checks the outbound table the run wrote and the users it
names, and a find that the user owns its home. The Zed wrapper's [checks](../zed-terminal-checks.sh)
run on the files it installed, and Fresh's on the config it loaded. The user's empty commit must
sign and verify. Root's leg has no table, and the run leaves out its own check
([Outbound](../../roles/server/ubuntu/security/README.md#outbound)). Its `/etc/ws/values.yml` sets a
64 MiB swap file for the small disk. With no NVIDIA GPU, `nvidia` checks its names alone, and with
no MSI board, `fans` leaves its fans alone. Without a VPN profile, `ikeward` does nothing.

## The install script

CI runs `install.sh` and `ws update` against releases it lays out and serves on loopback, and checks
that they install, update and refuse what they should
([The install job](../../engine/core/update/README.md#the-install-job)).

## The release

`release.yml` runs `ci.yml` and `mac.yml` on a tag before it publishes
([Cutting a release](../../engine/core/update/README.md#cutting-a-release)).

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's, in `roles/server/ubuntu/signoz`.

## The Mac

Case 47 checks [Zed's files](../../roles/client/macos/zed/README.md): `global_settings.json` must be
`roles/client/macos/zed/files/`'s whole, and a `settings.json` there must stay byte for byte. Case 48
checks `keymap.json`: `roles/client/macos/zed/files/`'s whole, a foreign one left with a warning, and
ws's back once it goes. Case 87 checks [the zsh hook](../../roles/client/macos/ghostty/README.md#the-zsh-hook): `zsh -i` must
reset at each prompt, and the alternate screen after a 255 from `ssh` or from `cld --host` alone.

CI also sets up a Mac, by [its cases](../mac), in the order of [The Mac](../../roles/client/macos/README.md). First, `ws
client macos apply --check` must change nothing: it writes no file, keeps no copy of `~/.ssh/config`
or `~/.zshrc`, and installs no formula or cask. It must note the install of Zed's cask. Stand-ins
for Ghostty's and Zed's apps, the font and cld, which `ws client macos apply` leaves alone, spare
the job's runs their downloads. The job's `values.yml` names a file key, made in case 20, as ws's
login key and git's. Without ws's WireGuard key, `ws client macos apply` must make the Mac's key and
load no daemon, and `ssh -G ws` gives the tunnel's end. The public key it prints must be the Mac's,
as must the key in the command that gives it to ws. With a stand-in's key it must load and run the
daemon. A second run and `--check` must print nothing. `ssh -G` must give `ws` the tunnel's end.
The stand-in for ws is a second wireguard-go on the runner, with the printed key as its peer. It
answers on loopback both as the LAN and as `WS_REMOTE`, and `ws client macos tunnel status` must
show a handshake on each in turn. The daemon must take the SIGHUPs for a new `tunnel.conf` and
its `routes.d` in place, with the same pid. Then a network that `ws client macos tunnel routes
--server ws` adds must be routed into the daemon's utun, and a domain `ws client macos resolver`
adds must show in `scutil --dns`. `ws client macos tunnel loaded`, `ws client macos tunnel routes`
and `ws client macos resolver` must do the same, with `--check` showing the diff and writing
nothing ([mac-ws-checks.sh](../mac-ws-checks.sh)). Next, the server's `WS_TUNNEL_NETWORKS` and
`WS_RESOLVERS` must give a route and a resolver, each shown by `--check` first. A second run and
`--check` must print nothing, and a fourth nameserver must be refused. Emptied or removed, the keys
must take both back, and spare a script's own. Last, `ws client macos apply` must read an empty
value in `values.yml` as its key's default, and still stop on a value of the wrong form. It must also
stop on a `WS_LAN_ADDRESS` outside `WS_LAN`, and on the LAN's network or broadcast address
([mac-conf-checks.sh](../mac-conf-checks.sh)). With `WS_CABLE_SERVICE` empty, `ws client
macos apply` must print nothing. Set, it must run one `networksetup -setmanual`, a stand-in's,
through sudo, and then none. It must warn of a disabled service. It must stop on one networksetup
lacks, or one with a router outside the cable's `/30`. It must stop too on a `-setmanual` that fails
or changes nothing, and name what differs ([mac-setup-checks.sh](../mac-setup-checks.sh)).
`ssh -G` must give `ws` the file key that the job's `values.yml` names, by its four lines. It must
give it the file or the agent that `WS_IDENTITY_PUB_KEY` and `WS_IDENTITY_AGENT` name, with
`--check` showing `/etc/ws/ssh_config`'s diff. An empty `WS_IDENTITY_PUB_KEY` must stop `ws client
macos apply`, though an agent at Secretive's socket holds a P-256 key. The stop must give the line
that names the key, by ws's copy of its `.pub`, with its comment and fingerprint. `PublicKeys` is
closed, as macOS 27 denies ws the files of Secretive's container. With that line, `ssh -G` must give
`ws` the agent and the copy, which a run writes where none exists. A `.pub` in `PublicKeys` that
`WS_IDENTITY_PUB_KEY` names must become the copy. With more keys in the agent, the stop must give a
line for each P-256 key alone. A `.pub` that is missing or unreadable, or holds a private key, a
certificate or a line that is no key, must stop `ws client macos apply` too. So must a key that
ssh-keygen cannot read or ws's OpenSSH lacks ([mac-identity-checks.sh](../mac-identity-checks.sh)).
Git's cases start without the job's git files, with a stand-in `gh` on `PATH`. It must stop on an
empty `WS_GIT_SIGNING_PUB_KEY`, or one that is missing or
unreadable, or a `.pub` that holds more than a public key. With a file key there, an ed25519 key
with no passphrase, git must sign a commit and verify it. A `~/.gitconfig` must get the include of
`/etc/ws/gitconfig` at its top, once, keep its own lines, and name its own settings. An include
below them, or a second one, must stop `ws client macos apply` before it writes a file, as must a
link that lacks the include. One whose file has it must pass. The settings of git's XDG file must be
named. With a key that an agent holds at Secretive's socket and `PublicKeys` closed, an empty
`WS_GIT_SIGNING_PUB_KEY` must stop with its line. Named, `--check` must show the wrapper and write
none. A run must write ws's copy of the key's `.pub`, and git sign whatever `SSH_AUTH_SOCK` names.
`ws` must keep the file key that `values.yml` names. git's credential command for GitHub must be a
stand-in gh's, which must add the key to its login once. Logged out, gh must draw a warning at each
run. Logged in, `ws client macos apply` must warn of a 404 or another error of gh's, and of an add
that leaves the account without the key. It must add the key where the account lacks it, and
`--check` name the add. Where `PATH` has no gh, `ws client macos apply` must install Homebrew's
once, and only with git's keys ([mac-git-checks.sh](../mac-git-checks.sh)). Where
wireguard-go makes no utun, the job fails, as the daemon's device makes a utun too.

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
`make lint-file FILES=PATH` runs the gates of the file's kind on that file alone:

- Markdown: `vale`, `lychee`, `rumdl` and `sizecheck`;
- YAML: `yamllint`, `vale` and `sizecheck`, and for a workflow `actionlint` and `zizmor`;
- Go: `gofmt`, `govet`, `golangci-lint`, `govulncheck`, `gotest`, `vale` and `sizecheck`;
- any other file: `shell` on a script, by extension or shebang, `vale` and `sizecheck`.

As Claude ends its turn, `make lint-tree` runs `make lint`, but its gitleaks scans only the
branch's own commits, from `origin/main`.

Each target runs the work tree's own code, as root on the server. That is its `Makefile` and
`.tools/` and the tools its pins name. So the plugin runs them only on a tree the maintainer
vouches for, as [the vouch rule][vouch] says.
The gate code includes what [.claude/lint-gates](../../.claude/lint-gates) lists.

[lint]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-hooks
[vouch]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-vouch-rule
