# devtools

The `devtools` role installs Node, the .NET SDKs, gh, glow, the libraries headless Chromium needs,
Go and rustup. It registers the machine's signing key with GitHub.

Node comes from NodeSource's apt repository, glow from Charm's, and each .NET SDK from Ubuntu's
archive or, for those it lacks, Launchpad's dotnet/backports PPA. Each run fetches their keys again,
and a key goes in only where it holds one certificate, whose fingerprint the role pins. A subkey
added to that certificate passes, and a second certificate fails the run.

Go goes in `/usr/local/go` from go.dev's tarball, checked against the SHA-256 go.dev lists for it,
and is replaced when `group_vars/all.yml` pins another version. rustup installs Rust's stable
toolchain in the user's home. Once there, it updates itself, so the playbook leaves it alone.

## The signing key

Right after gh's install, the role registers with GitHub the key that signs the machine's commits,
so that GitHub verifies them ([Signing](shell.md#signing)). Where `ws_git_signing_key`'s `.pub`
exists and the account's signing keys lack it, `gh ssh-key add --type signing` adds it, titled with
the machine's hostname. gh runs as `ws_user`, with that user's login. `shell_git_signing: false`
turns this off too.

The login needs the `write:ssh_signing_key` scope, as in
[A new machine](../README.md#a-new-machine) and [Running it again](../README.md#running-it-again).
Without a login, as in CI's container, without the scope, or on any other error of gh's, the run
says so and goes on. Until a later run adds the key, GitHub shows the machine's commits as
unverified, and the ruleset on `main` refuses them. A dry run reads the account's keys and says
what it would add. An older machine's key stays on GitHub until removed by hand, in the account's
settings.
