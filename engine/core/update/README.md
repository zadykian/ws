# Releases

A release of ws is one binary per platform, `ws-darwin-arm64`, `ws-linux-amd64` and
`ws-linux-arm64`, and `ws.sha256`, their checksums as `sha256sum` writes them. Each release
publishes `install.sh` too.

## Install

On a Mac with Apple silicon, or on Linux on x86_64 or aarch64, as the user who runs ws:

```sh
set -o pipefail
curl -fsSL --proto '=https' --tlsv1.2 \
    https://github.com/zadykian/ws/releases/latest/download/install.sh | sh
```

Without `pipefail`, a curl that fails pipes nothing into sh, which exits 0 having installed nothing.
An Intel Mac is refused, as no release is built for it.

`install.sh` finds the latest release once, so that both downloads come from it. It checks the
binary against the release's `ws.sha256` and runs it as `ws version`, which must print the release's
tag. Only then does it move the binary over `~/.local/bin/ws`, or over `/usr/local/bin/ws` as root.
A refused download installs nothing and keeps the ws there. Run again, it updates ws.
Put `~/.local/bin` on `PATH` by hand.

| Variable | Default | Sets |
| --- | --- | --- |
| `WS_VERSION` | the latest release | the release to install, `X.Y.Z` or `vX.Y.Z` |
| `WS_RELEASES_URL` | `https://github.com/zadykian/ws/releases` | where the releases are, for tests |

## Update

`ws update` replaces the ws that runs with the latest release, and changes nothing else. It runs no
role, so `ws server ubuntu apply` follows it by hand. It works as `cld update` does:

- It replaces the file ws runs from, the file a link leads to. `install.sh` goes by the user
  instead, so a ws in another directory stays there.
- It needs no curl or shell, as Go downloads the release.
- A failure leaves ws unchanged. So does SIGINT, SIGTERM or SIGHUP before the replace, which exits
  with 128 plus the signal's number.
- A ws at the latest release, or newer, downloads nothing.
- A build of a clone is refused, as it has no version to compare.

It takes `WS_RELEASES_URL` as the table above says. To pick a release, run `install.sh` with
`WS_VERSION`. A ws before v0.5.1 lacks the command, so `install.sh` updates it once.

`ws client macos wizard --update` stays apart: it runs `install.sh`, then the wizard with the new ws
([The Mac's wizard](../../../roles/client/macos/stages/README.md)).

## The install job

CI's `install` job runs [install-checks.sh](../../../.github/install-checks.sh) on each pull request. It
builds ws for Linux on amd64 at tags it makes and removes again, so that `ws version` prints them. It
lays four releases out in a temporary directory, and serves them on loopback with
[serve-releases](../../../.tools/serve-releases/main.go), which answers `/latest` with GitHub's redirect.
The cases:

- a named release installs, and a second run, for the latest, updates it;
- a wrong `ws.sha256`, and a binary that prints another version, are refused, install nothing and
  keep the installed ws;
- `ws update` moves an older ws to the latest release, and a second run stays;
- `ws update` replaces the file a link leads to, and keeps the link;
- `ws update` refuses a wrong `ws.sha256`, and a binary that prints another version, and keeps ws;
- root installs to `/usr/local/bin`.

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
