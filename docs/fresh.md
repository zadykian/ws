# fresh

The `fresh` role is Go steps in `internal/server/fresh`. It installs the Fresh editor on ws from
the `.deb` of a pinned release. Fresh runs in a terminal, and its settings come from the neutral
data in `internal/editor`, which Zed's role also reads.

## The install

- `Fresh download` fetches `fresh-editor_VERSION_ARCH.deb` from the project's GitHub release into
  `/var/cache/`. It checks the file against the SHA-256 in `download.go` before dpkg reads it.
- `Fresh install` runs dpkg on the file, then removes the file.
- Where `dpkg-query` lists the pinned version as installed, the download plans nothing and the
  install only removes a leftover file. A second run changes nothing.
- The pin holds the version, the tag and one sum for amd64 and one for arm64. Any other
  architecture fails the step.

The release's `.deb` needs `libc6` 2.30 and `libgcc-s1` 4.2 alone. So it installs on a physical
machine on the LAN, on a cloud VM and on a VM on the Mac. The testing `.deb` lacks the grammars
for JS, TS and Templ, and needs `libc6` 2.39.

The role needs no other role and no secret. No step needs a booted systemd, so it runs whole in
CI's container.

## The config

`Fresh config` writes `~/.config/fresh/config.json` for `ws_user`, from `internal/editor/fresh`. A
run replaces a change made to it by hand and keeps the old file beside the new one, as
[zed](zed.md) does for its settings.

- Updates, telemetry and the session daemon are off: `check_for_updates`, `self_update` and
  `auto_attach` are false. Fresh reads a missing key as its default, so the render writes each.
- The ruler, visible whitespace and auto-save come from `internal/editor/settings.yaml`, which
  Zed reads too.
- The file starts with a comment that says ws wrote it. Fresh reads JSONC.
- `~/.config/fresh` stays the user's, as Fresh writes `tsconfig.json` and `types/` there.
- The step fails where `~/.config`, the directory or `config.json` is a link. The user owns
  `~/.config`, and a link would send ws's root to whatever it names.
