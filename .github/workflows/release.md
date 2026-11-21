# Releases

A release of ws is one binary per platform, `ws-darwin-arm64`, `ws-linux-amd64` and
`ws-linux-arm64`, and `ws.sha256`, their checksums as `sha256sum` writes them. Each release
publishes `install.sh` too.

## Install

On Ubuntu Server on x86_64 or aarch64, as root, or on a Mac with Apple silicon, through `sudo`:

```sh
set -o pipefail
curl -fsSL --proto '=https' --tlsv1.2 \
    https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh
```

Without `pipefail`, a curl that fails pipes nothing into sh, which exits 0 having done nothing.
`install.sh` installs ws alone, on Linux and on the Mac. It runs in any POSIX sh, and needs curl and
sha256sum, or `shasum -a 256` where a Mac has no `sha256sum`:

- It picks `ws-linux-amd64`, `ws-linux-arm64` or `ws-darwin-arm64` by `uname`, and refuses another
  system or architecture. It finds the latest release once. A pre-release is never the latest: only
  `WS_VERSION` picks one.
- It downloads the file and the release's `ws.sha256` into a directory in `/usr/local/bin`, checks
  the file against the sum listed for its name, and runs it as `ws version`. Only then it renames it
  to `/usr/local/bin/ws`, uid 0, gid 0 and 0755, the one ws of the machine. A refusal keeps the ws
  there and leaves no file behind.
- It makes `/usr/local` and `/usr/local/bin` where they are missing, root's and 755. It refuses one
  that another user owns or that a group or others may write, and names the chown: launchd would
  refuse ws below it. It changes no owner, as an old Homebrew in `/usr/local` would lose its writes.
- Its last line names the next command, `ws server setup` on Linux and `ws client setup` on a Mac.
  Setup checks the system: it refuses a Linux but Ubuntu 26.04 before its first write.

| Variable | Default | Sets |
| --- | --- | --- |
| `WS_VERSION` | the latest release | the release whose ws installs, `X.Y.Z` or `X.Y.Z-rc.N`, with or without the `v` |
| `WS_RELEASES_URL` | `https://github.com/zadykian/ws/releases` | where the releases are, for tests |

## Update

`ws update` brings a machine's ws to a release, and ends in the setup of its side. On a server it
runs as root, and on a Mac as the user, which asks sudo for the write. Nothing else runs `install.sh`
again.

- It finds the latest release, or takes the one that `--version X.Y.Z` or `X.Y.Z-rc.N` names, with
  or without the `v`. It downloads that release's ws and `ws.sha256`, checks the file against the
  sum listed for its name, and runs it as `ws version`, which must print the tag. Only then it
  renames it over `/usr/local/bin/ws`, uid 0, gid 0 and 0755. A refusal, or a signal before the
  rename, keeps the ws there and leaves no file behind.
- It refuses the latest release where this ws is a release after it, such as a pre-release:
  `--version` names the older one. A ws that is no release, a trial's build, is older than any, so
  an update ends the trial.
- Then it replaces itself with `ws server setup` or `ws client setup` of the ws it wrote. Where this
  ws is the named release already, it downloads nothing, and runs that setup alone.

`WS_RELEASES_URL` names the releases, as for `install.sh`.

## The install job

CI's `install` job runs [install-checks.sh](../install-checks.sh) on each pull request of a stack on `main`. It
runs as a user with sudo on GitHub's runner, as `install.sh` writes `/usr/local/bin/ws`, which each
case removes, and the cases move `/usr/local/bin` aside and change its owner.
[serve-releases](../../.tools/serve-releases/main.go) serves a stub of ws for each target on
loopback, a shell script that prints its release and its target as `ws version`, with the release's
`ws.sha256`. A fake `uname` on the `PATH` of a case stands for aarch64 and for Darwin. The tags are
`v0.2.0`, `v0.3.0` and `v0.4.0-rc.1`, and `v0.5.0` and `v0.6.0` for the refusals. The cases:

- a user is refused, and nothing is installed;
- the latest release installs the stub of each target, root's and 755, and the last line names the
  next command;
- a system with no build is refused: Linux on another architecture, a Mac on x86_64 and another
  system;
- a wrong SHA-256 and a missing file are refused, and the ws that stands and no download is left;
- a newer latest release and a `WS_VERSION` install their own release;
- a `WS_VERSION` of a pre-release installs it, and a pre-release as the latest is refused;
- a `WS_VERSION` that is no version, or a pre-release `rc.0` or `beta.1`, is refused;
- `/usr/local/bin` is made where missing;
- a `/usr/local/bin` that another user owns or others may write, and a `/usr/local` that others
  may write, are refused, and keep their owner.

Then [update-checks.sh](../update-checks.sh) runs `ws update` as root against stubs as releases,
`v0.2.0`, the latest, and `v0.4.0-rc.1`, and `v0.5.0` whose `ws.sha256` fits no file. A stub prints
its tag as `ws version` and records its call of `server setup`. The ws that runs is a real build of
the tree, which a tag in a copy of it stamps as a release, or none for a trial's. The cases:

- a release's ws moves to the latest and runs its setup;
- a trial's build is replaced;
- `--version` takes a pre-release, and a real pre-release refuses the latest;
- a release older than this ws is refused, unless `--version` names it;
- a wrong sum and a missing release are refused, and ws stays the same;
- a ws of the named release downloads nothing and runs its own setup. The runner's system refuses
  that setup, and the script fails on Ubuntu 26.04, where the setup would be real;
- a user is refused before any download.

The script deletes `/usr/local/bin/ws` as it ends, so it sets that up after its guards. Its test,
[update-checks-test.sh](../update-checks-test.sh), runs in `lint` and checks that a run a guard
refuses deletes nothing.

The `mac` job's case [16-install.sh](../mac/16-install.sh) runs `install.sh` on macOS. It serves the
tree's own build of ws as the release, with its `ws.sha256`, and checks that the file is
`root:wheel` and 755 and that no download is left.

## Cutting a release

A release is a tag on `main`, after a stack has landed. ws is at 0.y.z: a change that breaks a
caller, marked `!`, bumps y, and any other bumps z. Tag the merge commit and push the tag:

```sh
git switch main && git pull --ff-only
git tag -s v0.2.0 && git push origin v0.2.0
```

A pre-release is a tag `vX.Y.Z-rc.N`, N from 1, on a commit of a `feat/` branch, the branch that
holds a change before it lands. The maintainer tries the branch from it before it lands on `main`.
Name the branch in place of BRANCH:

```sh
git fetch origin && git switch --detach origin/feat/BRANCH
git tag -s v0.13.0-rc.1 -m v0.13.0-rc.1 && git push origin v0.13.0-rc.1
```

A pre-release's own `install.sh` installs it by `WS_VERSION`, as the latest release's refuses a
pre-release:

```sh
set -o pipefail
curl -fsSL --proto '=https' --tlsv1.2 \
    https://github.com/zadykian/ws/releases/download/v0.13.0-rc.2/install.sh |
    sudo env WS_VERSION=0.13.0-rc.2 sh
```

So while a pre-release is tried, this line takes the place of each that runs `install.sh`. Those are
step 3 of [the server's runbook](../../roles/server/runbook.md), and step 2 of
[ws on ws](../../roles/client/runbook.md#ws-on-ws) and the Mac's Tools step on the Mac. The side's
setup follows, `ws server setup` or `ws client setup`, which checks out the tree of its own ws's tag.

Nothing else is by hand. `release.yml` runs on the tag, in these jobs:

1. `check`: `origin/main` holds the commit of a tag `vX.Y.Z`, and a branch `origin/feat/*` that of
   a tag `vX.Y.Z-rc.N`. Any other tag fails.
1. `ci` and `mac`: [ci.yml](ci.yml) and
   [mac.yml](mac.yml), called on the tag. Their jobs run,
   as on a push to `main`.
1. `build`: `ws-darwin-arm64`, `ws-linux-amd64` and `ws-linux-arm64` with the Go that
   [.tools/run](../../.tools/run) pins, without cgo, with `-trimpath` and Go's stamp of the tag, and
   `ws.sha256`. The job fails unless `ws version` of the amd64 binary prints the tag.
1. `publish`: GitHub's generated notes, then a check of the notes, the binaries, `ws.sha256` and
   `install.sh` for the local list's words (CLAUDE.md), from the secret `FORBIDDEN_WORDS`. It fails
   where the secret is empty, and names a file that holds a word, never the word. Then
   `gh release create --verify-tag`, with `--prerelease --latest=false` for a tag with `-rc.`, so that
   `releases/latest` keeps the last full release. Only this job has `contents: write`.

A tag that fails a job publishes nothing. The ruleset below keeps the tag: fix the branch, and tag
the next patch version, or the next `rc.N`.

## Settings

Three settings are not files, so set them by hand once:

- the secret `FORBIDDEN_WORDS`, the lines of `~/.config/ws/forbidden-words`;
- a ruleset on the tags `v*` that blocks their update and deletion;
- immutable releases, so that a published release's files and tag never change.
