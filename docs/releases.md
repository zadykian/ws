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

## The install job

CI's `install` job runs [install-checks.sh](../.github/install-checks.sh) on each pull request. It
builds ws for Linux on amd64 at tags it makes and removes again, so that `ws version` prints them. It
lays four releases out in a temporary directory, and serves them on loopback with
[install-server.py](../.github/install-server.py), which answers `/latest` with GitHub's redirect.
The cases:

- a named release installs, and a second run, for the latest, updates it;
- a wrong `ws.sha256`, and a binary that prints another version, are refused, install nothing and
  keep the installed ws;
- root installs to `/usr/local/bin`.
