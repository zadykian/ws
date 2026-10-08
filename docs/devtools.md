# devtools

The `devtools` role installs Node, the .NET SDKs, gh, glow, the libraries headless Chromium needs,
Go and rustup.

Node comes from NodeSource's apt repository, glow from Charm's, and each .NET SDK from Ubuntu's
archive or, for those it lacks, Launchpad's dotnet/backports PPA. Each run fetches their keys again
and checks each against the fingerprint the role pins.

Go goes in `/usr/local/go` from go.dev's tarball, checked against the SHA-256 go.dev lists for it,
and is replaced when `group_vars/all.yml` pins another version. rustup installs Rust's stable
toolchain in the user's home. Once there, it updates itself, so the playbook leaves it alone.
