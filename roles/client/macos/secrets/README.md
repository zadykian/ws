# The secrets on the Mac

The Mac keeps ws's secrets in fnox, in the private secrets repository, encrypted to the YubiKey 5C
Nano, to a backup key and to each server's host key. This role installs the tools that do it.

## The tools

[`mise.mac.toml`](mise.mac.toml) pins fnox 1.50.0 and age-plugin-yubikey 0.5.1, each from its GitHub
release, built for macOS on arm64. Both sides pin one fnox, as they read one secrets repository.
fnox's pin names mise's `github` backend, `github:jdx/fnox`, as the server's
[fnox](../../../server/ubuntu/secrets/README.md#fnox) does. The registry's `packslip` backend asks
GitHub's API for the release on a first install, unauthenticated. A CI runner meets the API's limit
there. age-plugin-yubikey takes the registry's name.

[`mise.mac.lock`](../../mise/mise.mac.lock) holds each one's URL and SHA-256 for macOS on arm64.
The bootstrap runs locked, so mise downloads from the lock's URL alone, and refuses a tool the lock
lacks. Neither publisher lists a checksum, so each SHA-256 is GitHub's digest of its file. mise
checked fnox's against the release's attestations as it wrote the lock. To move either on, change
the pin and lock it again, by `MISE_ENV=mac mise -C roles/client lock --platform macos-arm64` in a
copy of the tree. `TestMacLockHoldsEachTool` checks that the lock holds each tool, and no tool that
no role pins.

fnox encrypts a value to the Nano's `age1yubikey1` recipient by running age-plugin-yubikey from
`PATH`, and needs no YubiKey for it. To decrypt one, the plugin asks for the Nano, its PIN once a
session and a touch each time. Without the plugin on `PATH`, fnox fails either way.

## The units

mise installs both tools for the tree alone, on no shell's `PATH`. fnox must find the plugin on
`PATH`, and the maintainer runs `fnox` by hand in a clone of the secrets repository. So
[the role's units](units.toml.tmpl) copy them where every shell finds them, as `state.gh` does
([gh](../git/README.md#gh)):

- `state.fnox` copies fnox to `/usr/local/bin/fnox`, root:wheel and 755.
- `state.age-plugin-yubikey` copies age-plugin-yubikey to `/usr/local/bin/age-plugin-yubikey`,
  alike.

`/etc/paths` puts `/usr/local/bin` on every shell's `PATH`. Each copy goes in under a temporary name
first, and one there that differs is replaced. The check runs as the user, so a bootstrap that
changes nothing asks no password. Where mise has no such tool, its unit alone fails.

## The Nano's identity

The Nano holds the age identity of the recipient `nano` in `fnox.toml`. The Go package here finds
it by `age-plugin-yubikey --list`:

- Where `fnox.toml` names the `nano` recipient, it takes the slot that holds it, and no other. A
  YubiKey without it stops, and asks for the Nano that made it, as the values open to no other.
- Where `fnox.toml` names none, it takes the lowest slot that holds an age identity. Where none
  does, age-plugin-yubikey makes one, named `ws-secrets`, on the terminal. Its PIN policy is
  `once`, a PIN once a session, and its touch policy `always`, a touch each time it decrypts.

The slot's stub, the identity file that `age-plugin-yubikey --identity` prints, goes to fnox's key
file. So fnox, run by ws or by hand in a clone, asks the Nano to decrypt. The stub names the Nano's
serial and slot, and holds no secret. The key file is the one fnox reads by default:
`FNOX_AGE_KEY_FILE`, else `$FNOX_CONFIG_DIR/age.txt`, else `$XDG_CONFIG_HOME/fnox/age.txt`, else
`~/.config/fnox/age.txt`. Its other lines stay, a missing directory is made 700, and the file is
600. A key file that holds an SSH key stops it, as fnox would read that key alone.

ws runs fnox with `/usr/local/bin` first on `PATH`, where fnox finds the plugin. It drops
`FNOX_AGE_KEY`, which would beat the key file, and `FNOX_PROFILE`, which would take a profile's file
for `fnox.toml`.

## How a change lands

The secrets repository's ruleset on `main` refuses a push, and requires its checks, `values` and
`gitleaks`, on a branch up to date with `main`. So ws changes the repository by a pull request, from
a temporary clone that the Go package here makes:

1. It reads the repository from `WS_SECRETS_REPO` in the `[env]` of `/opt/ws/mise.toml`, and has
   gh read it as private. A missing one stops, and names `gh repo create OWNER/NAME --private`.
2. It clones `main` by HTTPS, which gh's credential command serves, into a temporary directory.
   A clone without `.github/workflows/check.yml` stops, as its checks would never come. fnox runs
   in the clone, as it reads `fnox.toml` from its working directory alone.
3. Where the change's files differ from `main`, it commits them, signed, on a branch
   `ws/TOPIC-YYYYMMDD-HHMMSS`, pushes it, and opens a pull request by gh. Where they don't, it
   opens none.
4. It reads the checks every 10 seconds, for 15 minutes at most. A failed or cancelled check stops
   it with the pull request's link, which stays open. So does a wait that runs out.
5. Once both checks pass, it merges the pull request by a merge commit, of the commit the checks
   ran on, and deletes its branch. It then removes the clone, whatever the outcome.

No value enters a step's arguments: a commit's title and body, a branch and the files' names alone.

## The inputs and fnox.toml

Each server role that needs a secret declares it as an input in its `mise.secrets.toml`, under
mise's `[bootstrap.secrets]`. The input's variable is its secret's name in fnox too, as `fnox exec`
exports each secret under its own name. The Go package here reads the inputs of the roles in
`/opt/ws/roles/server/ubuntu`, each variable once. A variable is of capitals, digits and
underscores, and a role without the file declares none.
