# Checks

CI runs yamllint, ansible-lint with its production profile, ShellCheck and shfmt on the shell
scripts, and gitleaks over the whole history. Any finding fails it, warnings included.

## The playbook

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`,
which need a booted machine. It runs it twice, and the second run must change nothing. Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM. It installs no
JetBrains backend, `jetbrains_backends: []`, as it has no room for those either.

## SigNoz's compose files

CI forges SigNoz's compose files again from their casting, with foundryctl pinned by its checksum,
and fails where they differ from the repository's.

## The Mac

CI also sets up a Mac, on GitHub's macOS runner, in the order of [The Mac](mac.md).
`mac/setup.sh` without ws's key must make the Mac's key and load no daemon, and with a stand-in's
key must load it. Then a second run and `--check` must print nothing, and `ssh -G` must give `ws`
and `ws-ssh` their addresses. The stand-in for ws is a second wireguard-go on the runner, with the
Mac's key as its peer. It answers on loopback both as the LAN and as `WS_REMOTE`, and
`ws-tunnel status` must show a handshake on each in turn. Last, a network that `ws_tunnel_routes`
adds must be routed into the daemon's utun, and a domain `ws_resolver` adds must show in
`scutil --dns`. Where the runner can make no utun, the steps of the tunnel are skipped, with a
warning.

## The hooks

The hooks in `.githooks` check each commit before git makes it. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath .githooks`, as `bootstrap.sh` does in its clone.
Where `~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose
added lines, file names or message hold one. Where gitleaks is installed, they refuse a staged
secret too.
