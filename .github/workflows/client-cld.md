# Ghostty and cld on the Mac

A second workflow, `client-cld.yml`, sets up Ghostty, Zed and cld on another macOS runner, as [Ghostty
and cld on the Mac](../../macos/roles/ghostty/README.md) and [Zed on the
Mac](../../macos/roles/zed/README.md) say. It runs `ws macos setup`
as the runbook does, on the tree that
[client-mise.sh](../client-mise.sh) puts in place, on a runner without Ghostty, Zed, its font, Secretive or cld. It
has a stand-in for ws's key, and a file key that `values.yml` names for ws's login and for git's
signing.

- Two runs must install Ghostty, Zed and the font as mise's casks, which `mise bootstrap packages
  status` names as mise's, and cld from its release. The facts read Ghostty's app and cld as a run
  starts, so the second adds ssh's lines. They must install Secretive by `macos-app`, at the pin of
  [ssh's `mise.macos.toml`](../../macos/roles/ssh/mise.macos.toml). A login shell must find `zed`.
- `/etc/ws/ghostty` must be `macos/roles/ghostty/files/ws.ghostty`, and Ghostty's config
  must read it once. Ghostty's `+validate-config` must take the config, and `+list-fonts` find its
  font. Zed's files must be those of `macos/roles/zed/files/`, with the seed of
  `settings.json`.
- A third run must call no sudo, and its plan exit 0.
- `ssh -G ws` must give the watcher's lines where Ghostty's app is there and cld is a release or
  has `restore tabs`. A stub of an older release must be updated by its unit, keep the lines, and
  have its line start the watcher in a Ghostty pane alone.
- With Ghostty's app moved aside, ssh's unit alone, by `ws unit write`, must drop the lines, and
  add them again once the app is back. Under a stub built from a clone without `restore tabs`, a
  run must drop them.

[client-cld-checks.sh](../client-cld-checks.sh) holds the checks.
