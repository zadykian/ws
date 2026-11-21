# Releases

A release of ws is one binary per platform, `ws-darwin-arm64`, `ws-linux-amd64` and
`ws-linux-arm64`, and `ws.sha256`, their checksums as `sha256sum` writes them. Each release
publishes `install.sh` too.

## Install

On Ubuntu 26.04 on x86_64 or aarch64, as root, through `sudo`:

```sh
set -o pipefail
curl -fsSL --proto '=https' --tlsv1.2 \
    https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh
```

Without `pipefail`, a curl that fails pipes nothing into sh, which exits 0 having done nothing.
`install.sh` sets up an Ubuntu server, and refuses a Mac, which follows
[the Mac's runbook](../../roles/client/runbook.md). It installs mise, writes the
machine's miserc and runs `mise bootstrap`:

- It downloads mise's binary at the version it pins. It checks the binary against the SHA-256 it
  holds for the machine's target, and runs it as `mise --version`. Only then it moves it over
  `/usr/local/bin/mise`. A refused download keeps the mise there and leaves no file behind. A mise
  at the pinned version is not downloaded again.
- It writes `/etc/mise/miserc.toml`: the set of envs `ws`, which holds `server` alone
  ([The frame](../../roles/server/ubuntu/frame/README.md#the-machines-env)).
- It finds the latest release once, and checks the release's tag out in `/opt/ws` by git, then
  runs the bootstrap on that tree, in the server's run directory, `/opt/ws/roles/server`. A
  pre-release is never the latest: only `WS_VERSION` picks one. The tree pins ws's binary, and mise
  checks it against `mise.lock`. With `MISE_LOCKED=1`, which `install.sh` sets, mise installs each
  tool only as a lock holds it. Run again, it checks the release's tag out in `/opt/ws`, dropping
  any change and any file git does not track there, such as a trial's `mise.local.toml`. So it
  brings the machine back to the release.
- It refuses a system other than Ubuntu 26.04, by `/etc/os-release`. So does `ws unit`'s read and
  write, which the bootstrap runs.
- It refuses a `/opt/ws` that is a link, or that holds a file or directory root does not own or
  others may write. mise runs the tree's config as root.

| Variable | Default | Sets |
| --- | --- | --- |
| `WS_VERSION` | the latest release | the release whose tree runs, `X.Y.Z` or `X.Y.Z-rc.N`, with or without the `v` |
| `WS_IN_PLACE` | unset | `1` runs the tree in `/opt/ws` as it stands: no lookup, no clone |
| `WS_RELEASES_URL` | `https://github.com/zadykian/ws/releases` | where the releases are, for tests |
| `WS_MISE_URL` | `https://github.com/jdx/mise/releases/download` | where mise's binaries are, for tests |
| `WS_REPO` | `https://github.com/zadykian/ws` | the repository the tag is cloned from, for tests |
| `WS_OS_RELEASE` | `/etc/os-release` | the file that names the system, for tests |

## The install job

CI's `install` job runs [install-checks.sh](../install-checks.sh) on each pull request of a stack on `main`. It
runs as a user with sudo on GitHub's runner, as `install.sh` writes `/etc/mise` and `/opt/ws`,
which each case removes. `WS_OS_RELEASE` names the runner's system Ubuntu 26.04,
whatever its release. It downloads mise's real binary once and serves it on loopback
with [serve-releases](../../.tools/serve-releases/main.go), beside a copy with a byte added. A small
repository stands for the release's tree: a file of mise's config for each env, each writing one
file. Its tags are `v0.2.0`, then `v0.3.0` with one file more, and `v0.4.0-rc.1` with an empty
commit. The cases:

- a user is refused, and nothing is installed;
- Ubuntu 24.04, Debian and a system with no `os-release` are refused, and nothing is installed;
- a wrong SHA-256 of mise is refused, and no mise or download is left;
- the latest release installs mise and runs its bootstrap, so that the files of the envs that the
  miserc names exist;
- the env `secrets` loads by `-E ws,secrets` alone, and a plan without it after keeps its file;
- a second run downloads no mise;
- a newer latest release checks its tag out in `/opt/ws`, and drops a `mise.local.toml` there;
- `WS_VERSION` checks its release's tag out;
- a `WS_VERSION` of a pre-release checks its tag out, and a pre-release as the latest is refused;
- a `WS_VERSION` that is no version, or a pre-release `rc.0` or `beta.1`, is refused;
- `WS_IN_PLACE` runs the tree in `/opt/ws` as it stands;
- a tree with a file that root does not own is refused.

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

A tag holds no checksum of its own binaries, so a pre-release's tree pins an earlier release of
ws. A pull request then pins the pre-release, and the next pre-release, cut once it lands, is the
first whose tree installs that ws. Its own `install.sh` runs it by `WS_VERSION`, as the latest
release's refuses a pre-release:

```sh
set -o pipefail
curl -fsSL --proto '=https' --tlsv1.2 \
    https://github.com/zadykian/ws/releases/download/v0.13.0-rc.2/install.sh |
    sudo env WS_VERSION=0.13.0-rc.2 sh
```

The latest release's `install.sh` may install ws's binary alone, with no mise. So while a
pre-release is tried, this line takes the place of each that runs it. Those are step 3 of
[the server's runbook](../../roles/server/runbook.md), and step 2 of
[ws on ws](../../roles/client/runbook.md#ws-on-ws) on the Mac. The Mac's own binary and
clone take the pre-release's tag in place of the latest release's.

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
