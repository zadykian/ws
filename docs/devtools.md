# devtools

The `devtools` role installs Node, the .NET SDKs, gh, glow, the libraries headless Chromium needs,
Go and rustup.

Node comes from NodeSource's apt repository, glow from Charm's, and each .NET SDK from Ubuntu's
archive or, for those it lacks, Launchpad's dotnet/backports PPA. Each run fetches their keys again,
and a key goes in only where it holds one certificate, whose fingerprint the role pins. A subkey
added to that certificate passes, and a second certificate fails the run.

Go goes in `/usr/local/go` from go.dev's tarball, checked against the SHA-256 go.dev lists for it,
and is replaced when `group_vars/all.yml` pins another version. rustup installs Rust's stable
toolchain in the user's home. Once there, it updates itself, so the playbook leaves it alone.

## The signing key

The role does not register the key that signs the machine's commits ([Signing](shell.md#signing)).
ws's gh holds a token with repository permissions alone, which cannot add a signing key. The Mac's
wizard adds it with the Mac's gh ([Signing key](mac-wizard.md#signing-key)), or add the `.pub` by
hand in the account's settings on GitHub, as a signing key. Until then, GitHub shows the machine's
commits as unverified, and the ruleset on `main` refuses them.
