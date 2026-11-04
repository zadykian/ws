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
