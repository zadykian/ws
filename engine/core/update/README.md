# Releases

A release of ws is one binary per platform, `ws-darwin-arm64`, `ws-linux-amd64` and
`ws-linux-arm64`, and `ws.sha256`, their checksums as `sha256sum` writes them. Each release
publishes `install.sh` too.

## Install

On Linux on x86_64 or aarch64, as root, through `sudo`:

```sh
set -o pipefail
curl -fsSL --proto '=https' --tlsv1.2 \
    https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh
```

Without `pipefail`, a curl that fails pipes nothing into sh, which exits 0 having done nothing.
`install.sh` sets up an Ubuntu server, and refuses a Mac, which follows
[the Mac's runbook](../../../roles/client/macos/runbook.md). It installs mise, writes the
machine's miserc and runs `mise bootstrap`:

- It downloads mise's binary at the version it pins. It checks the binary against the SHA-256 it
  holds for the machine's target, and runs it as `mise --version`. Only then it moves it over
  `/usr/local/bin/mise`. A refused download keeps the mise there and leaves no file behind. A mise
  at the pinned version is not downloaded again.
- It writes `/etc/mise/miserc.toml` from `WS_USER` in `/etc/ws/values.yml`: the envs `server`,
  `booted` where systemd runs, and `account` where `WS_USER` is not root
  ([The frame](../../../roles/server/ubuntu/frame/README.md#the-machines-env)). After a change of
  `WS_USER`, run it again.
- It finds the latest release once, and clones the release's tag into `/opt/ws`, then runs the
  bootstrap on that tree. The tree pins ws's binary, and mise checks it against `mise.lock`. Run
  again, it checks the release's tag out in `/opt/ws`, dropping any change and any file git does
  not track there, such as a trial's `mise.local.toml`. So it brings the machine back to the
  release.
- It refuses a `/opt/ws` that is a link, or that holds a file or directory root does not own or
  others may write. mise runs the tree's config as root.

| Variable | Default | Sets |
| --- | --- | --- |
| `WS_VERSION` | the latest release | the release whose tree runs, `X.Y.Z` or `vX.Y.Z` |
| `WS_IN_PLACE` | unset | `1` runs the tree in `/opt/ws` as it stands: no lookup, no clone |
| `WS_RELEASES_URL` | `https://github.com/zadykian/ws/releases` | where the releases are, for tests |
| `WS_MISE_URL` | `https://github.com/jdx/mise/releases/download` | where mise's binaries are, for tests |
| `WS_REPO` | `https://github.com/zadykian/ws` | the repository the tag is cloned from, for tests |

## The install job

CI's `install` job runs [install-checks.sh](../../../.github/install-checks.sh) on each pull request. It
runs as a user with sudo on GitHub's runner, as `install.sh` writes `/etc/mise`, `/opt/ws` and
`/etc/ws`, which each case removes. It downloads mise's real binary once and serves it on loopback
with [serve-releases](../../../.tools/serve-releases/main.go), beside a copy with a byte added. A small
repository stands for the release's tree: a file of mise's config for each env, each writing one
file, tagged `v0.2.0`, and later `v0.3.0` with one file more. The cases:

- a user is refused, and nothing is installed;
- a wrong SHA-256 of mise is refused, and no mise or download is left;
- the latest release installs mise and runs its bootstrap, so that the files of the envs that the
  miserc names exist;
- with `WS_USER: root`, a second run downloads no mise and leaves the account's part out;
- a newer latest release checks its tag out in `/opt/ws`, and drops a `mise.local.toml` there;
- `WS_VERSION` checks its release's tag out;
- a `WS_VERSION` that is no version is refused;
- `WS_IN_PLACE` runs the tree in `/opt/ws` as it stands;
- a tree with a file that root does not own is refused.

## Cutting a release

A release is a tag on `main`, after a stack has landed. ws is at 0.y.z: a change that breaks a
caller, marked `!`, bumps y, and any other bumps z. Tag the merge commit and push the tag:

```sh
git switch main && git pull --ff-only
git tag -s v0.2.0 && git push origin v0.2.0
```

Nothing else is by hand. `release.yml` runs on the tag, in these jobs:

1. `check`: the tag is `vX.Y.Z`, and `origin/main` holds its commit.
1. `ci` and `mac`: [ci.yml](../../../.github/workflows/ci.yml) and
   [mac.yml](../../../.github/workflows/mac.yml), called on the tag. Their `playbook` and `mac` jobs run,
   as on a push to `main`.
1. `build`: `ws-darwin-arm64`, `ws-linux-amd64` and `ws-linux-arm64` with the Go that
   [.tools/run](../../../.tools/run) pins, without cgo, with `-trimpath` and Go's stamp of the tag, and
   `ws.sha256`. The job fails unless `ws version` of the amd64 binary prints the tag.
1. `publish`: GitHub's generated notes, then a check of the notes, the binaries, `ws.sha256` and
   `install.sh` for the local list's words (CLAUDE.md), from the secret `FORBIDDEN_WORDS`. It fails
   where the secret is empty, and names a file that holds a word, never the word. Then
   `gh release create --verify-tag`. Only this job has `contents: write`.

A tag that fails a job publishes nothing. The ruleset below keeps the tag: fix main, and tag the
next patch version.

## Settings

Three settings are not files, so set them by hand once:

- the secret `FORBIDDEN_WORDS`, the lines of `~/.config/ws/forbidden-words`;
- a ruleset on the tags `v*` that blocks their update and deletion;
- immutable releases, so that a published release's files and tag never change.
