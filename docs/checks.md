# Checks

In a stack, CI runs `playbook`, `rootless`, `mac`, `mac-cld` and `mac-wizard`, its longest jobs, on
the top pull request alone, as it holds every change below it. `lint`, `install` and `signoz` run
on each. A stack opens as drafts, as a pull request opens before its link, and only a later event
names the stack. So these five skip a draft's `opened` run and start on `ready_for_review`, which
the other three skip.
The checks of `rootless` and `mac-cld` are on their own pages ([rootless](checks-rootless.md),
[mac-cld](checks-mac-cld.md)), and the gates of `make lint` on [theirs](gates.md).

## The `playbook` job

CI also builds ws from the tree, never from a release, in an Ubuntu 26.04 container with no git, so
that the build is `devel`. It runs `ws server apply` twice from root's home, as `zadykian` and as
root, a cloud VM's user. The steps that need a booted systemd skip and say so, and the second run's
last line must be `changed: none`. Under `script`, `ws server apply --check` must then draw [the run
screen](tui.md#the-run-screen) and end on `would change: none`. Then `nft -c`, with the container's
NET_ADMIN, checks the outbound table the run wrote and the users it names, and a find that the user
owns its home. The Zed wrapper's [checks](../.github/zed-terminal-checks.sh) run on the files it
installed, and Fresh's on the config it loaded. Root's leg has no table. The run leaves out its own
check ([Outbound](security.md#outbound)). Its `/etc/ws/host_vars.yml` gives a swap file of 64 MiB,
as the runner's disk has no room for one of its RAM's size. Its board is not MSI's, so `fans` leaves
its fans alone. It gives no VPN profile, so `ikeward` does nothing.

## The install script

CI runs `install.sh` and `ws update` against releases it lays out and serves on loopback, and checks
that they install, update and refuse what they should
([The install job](releases.md#the-install-job)).

## The release

`release.yml` runs `ci.yml` and `mac.yml` on a tag before it publishes
([Cutting a release](releases.md#cutting-a-release)).

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's, in `internal/server/signoz`.

## The Mac's wizard

A workflow of its own, `mac-wizard.yml`, runs `ws mac wizard` on GitHub's macOS runner, from a copy
of the checkout at `~/repository/ws`. `standin.sh` gives the wizard what a new Mac and user would,
such as `mac.conf`'s answers, dash, and a logged-in `gh`, first on the wizard's `PATH`. Then
`checks.sh` runs the cases beside it in order, stopping at the first failure
([checks.sh](../.github/mac-wizard/checks.sh)). The wizard must refuse an unknown flag, root and
a removed `WS_HOOK` line, as an unknown key, and a removed op of the files script. A run must stop
where another holds the lock, leaving it. A fresh `--check`, by sh and dash, must find brew with its
directories off `PATH`. As `ws mac apply --check` must, it must change nothing, and leave no lock or
temporary file. The job's `mac.conf` names `~/.ssh/id_ed25519` as ws's key, which `--check` must
show it would make. With no terminal and a `mac.conf` that names none, a run must stop at the Mac
key stage, naming `WS_IDENTITY`, making nothing. A file key there with no line must get one, shown
by `--check`. A failed `ssh-add` must stop a run, naming it. The first run, with that `PATH` too,
must make that key with no passphrase, add it to that login, install what `ws mac apply` installs.
It must drop the Mac's key that it prints, and leave no tunnel step by hand, as the Tunnel stage
below takes it. A second run and `--check` must print nothing, by sh and dash. Then a stand-in
`ssh-agent` at Secretive's socket holds a P-256 key, whose `.pub` a `mac.conf` of its own names in
`WS_IDENTITY`. While it lists the key, `--check` must print nothing of the Mac key stage. With git's
name, it must say it would wait for git's key made in Secretive, unless `WS_GIT_SIGNING_KEY` is
empty. With no `WS_IDENTITY` and `PublicKeys` closed, it must take the agent's key, and say it would
write ws's copy. Once the agent lacks it, a run and `--check` must stop there, naming the socket.
Last,
without the stand-in `gh`'s login, a run with no terminal must stop at gh's stage, naming `gh auth
login`. With a login lacking the scopes, or the signing one alone, it must name `gh auth refresh`.
`--check` must say each would run.

`standin.sh` starts an `sshd` of its own as ws, at `127.0.0.1:22022`, where the job's `mac.conf`
leads, with its own host key. A fresh `--check` must show the Host key stage after `ws mac apply`.
The first run must warn that ws faces the internet, and add ws's key to `known_hosts`, as
`WS_HOST_KEY` gives its fingerprint. Without that line, a run with no terminal must stop without
`WS_HOST_KEY` and with another fingerprint, and `--check` must show the line, or stop at a link, as
a run must. With a `mac.conf` of no way to ws, a run must stop, naming [The router](router.md).
Where `WS_REMOTE` doesn't answer, ws's address must come from `WS_FIRST_ADDRESS`, and the closing
list name `WS_PORT` and no lease's address. On a cloud VM, it must name `mac.conf`'s way in place of
the lease.

The runner's user logs in to that `sshd` by its own `authorized_keys`, which `standin.sh` empties
and the cases fill from the stand-in `gh`'s keys. Each session runs `ws.sh`, which logs the command.
The wizard adds no key, so the first run must stop at the Login stage, naming the Mac's key, and add
none. The case then gives the user the keys. With another key alone in the file, the stop must show
ssh's error. With a forced command on the line for the key, as cloud-init writes it, `--check` must
say that a run would stop. A run must stop, with no terminal and at a terminal of `script`'s, and
change nothing. With the keys back, a run must say nothing of the stage. With another host key under
`ws`, a run must stop, with `StrictHostKeyChecking no` in `~/.ssh/config` too. Neither the `sshd`'s
second host key, an ECDSA one, nor `--check` under `accept-new` may add a line to `known_hosts`. A
stand-in `sudo` runs the scripts as root. Where it asks a password, a run with no terminal must stop
at Sudo. At a terminal, it takes the right one after a wrong one, and neither shows in a log. A
cloud target with no `WS_USER` logs in as root, with no `sudo`. A `LocalCommand` that touches a file
must run for a plain ssh to ws, not for the wizard's.

The wizard's remote script, `mac/wizard-remote.sh`, runs as root on that `sshd`, through `sudo`,
sent as the wizard sends it. `ws.sh` roots its paths in a directory of its own, and puts the
stand-ins of [ws-standin.sh](../.github/mac-wizard/ws-standin.sh) for ws's programs first on its
`PATH`. `status` must print the stamps and the stand-ins' states. Those are `wg0`'s key, the Mac's
key for it and as its peer, and ws's signing key or `none`. They are also its hostname, claude's
login, Secure Boot that refuses the fans' driver, and `host_vars.yml`'s `ws_user` and outbound
table. `state` must print ws's report and exit with its status. Like a full `run`, it has root's
`HOME` and a UTF-8 locale. A full `run` must run `ws server apply` with `--json` alone in a unit of
its own. It must print the log, event lines too. It must stamp ws's version, `-` here, 0, `on` and
`-`, the secrets file's sum; after its `outbound` writes the line, which stops on a link, `off`. A
tagged run after it must start its own unit and leave that stamp. A run must follow a full unit or
one of its steps, and refuse another's. A full run cut off must remove the last stamp, so the next
starts a unit. The files script, `mac/wizard-remote-files.sh`: its `link` must list the wired ports,
`user NAME` name `ws_user`, `values` print `host_vars.yml` and `values PAIR...` run `ws server
values set` on each. `link PORT` must name one there, mode 600 in a 755 directory, once, which
`status` names once the roles give `10.77.0.1`. With ws's secrets file and its credential, a run
must start a unit that loads the credential and runs `--secrets` on that file, and stamp its
SHA-256. A second run must start none, a changed file one, and the file without the credential one
with no secrets, then none ([61-secrets.sh](../.github/mac-wizard/61-secrets.sh)).

The stand-in ws holds a `curl` that writes a release's `install.sh`, which sets the version of its
`ws`. The first run must install v0.0.0, which the job's clone is tagged as, once. With v0.0.9,
a newer release, on ws, a run must stop and name `ws mac wizard --update`, with no install or unit.
`standin.sh` serves a release v0.0.0, the job's build, as the latest on 127.0.0.1:22080 with
[serve-releases](../tools/serve-releases/main.go), and the job's `WS_RELEASES_URL` names it. So
`--update` finds v0.0.0 and stays. A tagged clone built as v0.0.1, the latest, must make
`--update` leave v0.0.1 as the Mac's `ws version` and as the stand-in ws's (case 59).
The wizard's stage ws's run must wait for ws on ws under a fresh `--check`. The first run must run
ws's apply through the remote script, and print none of its event lines. Each run since must read
ws's `status` alone, once, and again after a run of apply. A run's calls to ws share one connection,
which its end closes. Without the stamp, a run whose unit is cut off must stop at the stage, and the
next must start a unit of its own. That run must end with what stays by hand: the fans' driver and
the router's forwards. A run while a unit runs, as a dropped ssh's, must follow that unit. With
`WS_TARGET=cloud`, a run must turn the outbound table off, and the next none.

`standin.sh` logs ws's claude in, and the Mac's gh lists ws's signing key, so no run may open a
session on ws for a login. A fresh `--check` must show the Signing key and claude on ws after ws's
run. With the key off the gh login, `--check` must say it would run `gh ssh-key add --type signing`,
titled with ws's hostname. A run must add it to the signing keys alone, and list what stays by hand
without ws's gh login. The next must print nothing, as a run must where root's git config names no
key. Without claude's login, a run must stop at claude on ws, naming `claude auth login`, and
`--check` must say it would run it. The next must print nothing.

The stand-in `ws server apply --only wireguard` makes ws's `wg0`, a `wireguard-go` on UDP 51820,
with the Mac's key in `mac.pub` as its peer. A fresh `--check` must show the Tunnel stage after ws's
run. The first run must write `mac.pub`, run that, give `WS_PUBLIC_KEY` `wg0`'s key, and load the
daemon. It must log in once through the tunnel, by stand-ins of `ssh` and `nc`, as `wg0` has no
address. A new key for `wg0` must replace `WS_PUBLIC_KEY`'s line, and a missing `mac.pub` come back.
Without `wg0`'s peer alone, a run must run it again. With nothing on UDP 51820, the closing list
must name the port. Where `sudo -n` fails, a run must print nothing, and without `tunnel.pub`, write
it first.

With claude logged out, a run at a terminal of `script`'s must run `claude auth login` through
`bash -lc` in an `ssh -t` session. The stand-in takes a terminal alone, and the next run must print
nothing.

For the Cable stage, ws's ports are a stand-in sysfs. Undocked, a run must print nothing, one that
acts name the dock, and `WS_CABLE_PORT` end the wait. Docked, one with no terminal must stop, one
with `WS_CABLE_PORT` name the port, run the roles, read its late address, and stop at the service.
Typed, an empty line must take the `169.254` one, and `ping` answer. The next must read no `link`,
and one after ws's install or a failed roles' run name it again. Off the LAN, a run and `--check`
must skip it, a cloud VM's naming its firewall.

The cases give `mac.conf`'s other answers in the environment. With no terminal, a missing answer
must stop a run and `--check` within 60 s, naming its key. A bad one from the environment must stop
`--check`, with the form the key takes. `--check` must show the environment's answers as the lines
it would add to `mac.conf`, and the first run must add them at its end. Typed at a terminal of
`script`'s after bad ones, each asked again, the same answers must show the same lines, and empty
lines the defaults, for each target. After the first run, they must write the same `mac.conf`.

Last, the wizard with that `PATH` must find the runner's Command Line Tools and brew, so that a
second run and `--check` print nothing at all. Where `WS_CONF` names no file, `--check` must show
the lines it would write there, from the answers in the environment, and write none. The
case that reads [`homebrew.go`](../internal/wizard/homebrew.go)'s pins must download Homebrew's
`.pkg` and match its SHA-256 and signer, the signer as `pkgSigner` reads it. The log shows
`pkgutil`'s chain. The `.pkg` cut short must fail the sum. A run checks it before the install,
which the case never runs.

## The Mac

Case 47 checks [Zed's files](mac-zed.md): `global_settings.json` must be `mac/zed/`'s file whole,
and a `settings.json` that is there must stay byte for byte.
Case 48 checks its `keymap.json`: `mac/zed/`'s keymap whole, a foreign file left with one warning,
and ws's back once the foreign one goes.

CI also sets up a Mac, by [its cases](../.github/mac), in the order of [The Mac](mac.md). First, `ws
mac apply --check` must change nothing: it writes no file, keeps no copy of `~/.ssh/config` and
installs no formula or cask. It must note the install of Zed's cask. Stand-ins for
Ghostty's and Zed's apps, the font and cld, which `ws mac apply` leaves alone, spare the
job's runs their downloads. Without ws's key, `ws mac apply` must make the Mac's key and load no
daemon, while `ssh -G ws` gives the tunnel's end. The public key it prints must be the Mac's, as
must the key in the command that gives it to ws. With a stand-in's key it must load the daemon,
which must run. Then a second run and `--check` must print nothing. Under `script`, `--check` must
draw [the run screen](tui.md#the-run-screen) and leave it. `ssh -G` must give `ws` the
tunnel's end. The stand-in for ws is a second wireguard-go on the runner, with the printed key as
its peer. It answers on loopback both as the LAN and as `WS_REMOTE`, and `ws mac tunnel status` must
show a handshake on each in turn. The daemon must take the SIGHUPs for a new `tunnel.conf` and
`routes.d` in place, with the same pid. Then a network that `ws mac tunnel routes` adds must be
routed into the daemon's utun, and a domain `ws mac resolver` adds must show in `scutil --dns`. `ws
mac tunnel loaded`, `ws mac tunnel routes` and `ws mac resolver` must do the same, with `--check`
showing the diff and writing nothing ([mac-ws-checks.sh](../.github/mac-ws-checks.sh)). Next,
`mac.conf`'s `WS_TUNNEL_NETWORKS` and `WS_RESOLVERS` must give a route and a resolver, each shown by
`--check` first. A second run and `--check` must print nothing, and a fourth nameserver must be
refused. Emptied or removed, the keys must take both back, and spare a script's own. Last, `ws mac
apply` must read an empty value in `mac.conf` as its key's default, and still stop on a value of the
wrong form. It must also stop on a `WS_LAN_ADDRESS` outside `WS_LAN`, and on the LAN's network or
broadcast address ([mac-conf-checks.sh](../.github/mac-conf-checks.sh)). With `WS_CABLE_SERVICE`
empty, `ws mac apply` must print nothing. Set, it must run one `networksetup -setmanual`, a
stand-in's, through sudo, and then none. It must warn of a disabled service. It must stop on one
networksetup lacks, or one with a router outside the cable's `/30`. It must stop too on a
`-setmanual` that fails or changes nothing, and name what differs
([mac-setup-checks.sh](../.github/mac-setup-checks.sh)). `ssh -G` must give `ws` the file key, by
the four lines `config.d/ws` had for it before. It must give it the file or the agent that
`WS_IDENTITY` and `WS_IDENTITY_AGENT` name, with `--check` showing `config.d/ws`'s diff alone. Then
it must give it an agent at Secretive's socket, which holds a P-256 key, and ws's copy of its
`.pub`, where those keys are empty. `PublicKeys` is closed then, as macOS 27 denies ws the files of
Secretive's container. It must warn that this replaces the file key, and write the copy where none
exists. A `.pub` in `PublicKeys` that `WS_IDENTITY` names must become the copy. A second key in the
agent must stop `ws mac apply`. Where `WS_GIT_SIGNING_KEY` names one of the two, ws's key must be
the other. The file key's `.pub` in `WS_IDENTITY` must keep the file key. A `.pub` that is missing
or unreadable, or holds a private key, a certificate or a line that is no key, must stop `ws mac
apply` too. So must a key that ssh-keygen cannot read or ws's OpenSSH lacks
([mac-identity-checks.sh](../.github/mac-identity-checks.sh)). With `WS_GIT_NAME` and `WS_GIT_EMAIL`,
it must make git's signing key once, an ed25519 key with no passphrase. It must stop on a
`WS_GIT_SIGNING_KEY` that is missing or unreadable, or a `.pub` that holds more than a public key.
So must the default key or its `.pub` alone. git must then sign a commit and verify it. A
`~/.gitconfig` must get the include of `~/.config/ws/gitconfig` at its top, once, keep its own
lines, and name its own settings. An include below them, or a second one, must stop `ws mac apply`
before it writes a file, as must a link that lacks the include. One whose file has it must pass. The
settings of git's XDG file must be named. With a key that an agent holds at Secretive's socket and
`PublicKeys` closed, `--check` must show the wrapper and write none. A run must write ws's copy of
the key's `.pub`, and git sign whatever `SSH_AUTH_SOCK` names. `ws` must keep the file key, as
Secretive's one key is git's. git's credential command for GitHub must be a stand-in gh's, which
must add the key to its login once. Logged out, gh must draw a warning at each run. Logged in, `ws
mac apply` must warn of a 404 or another error of gh's, and of an add that leaves the account
without the key. It must add the key where the account lacks it, and `--check` name the add before
the key is made. Where `PATH` has no gh, `ws mac apply` must install Homebrew's once, and only with
git's keys ([mac-git-checks.sh](../.github/mac-git-checks.sh)). Where wireguard-go makes no utun,
the job fails, as the daemon's device makes a utun too.

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
- YAML: `yamllint` and `sizecheck`, and for a workflow `actionlint` and `zizmor`;
- shell, by extension or shebang: `shell` and `sizecheck`;
- Go: `gofmt`, `govet`, `golangci-lint`, `govulncheck`, `gotest` and `sizecheck`.

As Claude ends its turn, `make lint-tree` runs `make lint`, but its gitleaks scans only the
branch's own commits, from `origin/main`.

Each target runs the work tree's own code, as root on the server. That is its `Makefile` and
`tools/` and the tools its pins name. So the plugin runs them only on a tree the maintainer
vouches for, as [the vouch rule][vouch] says.
The gate code includes what [.claude/lint-gates](../.claude/lint-gates) lists.

[lint]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-hooks
[vouch]: https://github.com/zadykian/toolkit/blob/main/docs/lint.md#the-vouch-rule
