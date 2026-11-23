# dotnet

The `dotnet` role installs .NET's SDKs 10.0.404 and 11.0.100 on the Mac, as the server's
[devtools](../../../server/ubuntu/devtools/README.md#the-net-sdks) does on Linux.

`tree/dotnet` unpacks Microsoft's tarballs for Apple silicon into `/usr/local/share/dotnet`, root's,
side by side. Each is checked against the SHA-256 that the unit pins, and a new pin replaces the
directory whole ([tree](../../../../engine/core/kind/README.md#tree)). The tarballs wait in
`/Library/Caches/ws` until the install. A tool's own program looks for .NET in that directory on
macOS, with no `DOTNET_ROOT`.

`file/dotnet-path` writes `/etc/paths.d/dotnet`, which puts the directory on a login shell's `PATH`.
A shell open before the first run finds no `dotnet` until it logs in again.

To bump an SDK, take its file's SHA-512 from its channel's `releases.json`, such as
`https://builds.dotnet.microsoft.com/dotnet/release-metadata/10.0/releases.json`. Check the
`osx-arm64` tarball against it with `shasum -a 512`, then pin its SHA-256 and the version in the
unit. The server pins its own.
