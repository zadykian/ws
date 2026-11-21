# The secrets on the Mac

The Mac keeps ws's secrets in fnox, in the private secrets repository, encrypted to the YubiKey 5C
Nano, to a backup key and to each server's host key. This role installs the tools that do it.
`ws client secrets init` sets the repository up, `ws client secrets add NAME` adds a
server to it, and `ws client remove NAME` takes a server's key out.

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

## ws client secrets init

Run it as yourself, with the Nano plugged in, once the secrets repository exists
([the runbook](../../runbook.md#the-secrets)). It takes no `--server`, as the repository serves
every server. It needs fnox and age-plugin-yubikey in `/usr/local/bin`, which the bootstrap puts
there. In a temporary clone of the repository, it:

1. Finds the Nano's age identity, or makes one, and keeps its stub in fnox's key file.
2. Where `fnox.toml` is missing, makes the backup key, and starts `fnox.toml` with the Nano's and the
   backup's recipients.
3. Declares each input that the server's roles in `/opt/ws` declare and `fnox.toml` lacks, by
   `fnox set NAME --provider age` with an empty value. A server's run then finds each input, and
   fails on a key that opens none.
4. Prints the backup's secret line, where it made one, and waits for Enter.
5. Lands `fnox.toml` by a pull request, as below.

## The Nano and the backup

The Nano holds the age identity of the recipient `nano` in `fnox.toml`. The Go package here finds
it by `age-plugin-yubikey --list`:

- Where `fnox.toml` names the `nano` recipient, it takes the slot that holds it, and no other. A
  YubiKey without it stops, and asks for the Nano that made it, as the values open to no other.
- Where `fnox.toml` names none, it takes the lowest slot that holds an age identity. Where none
  does, age-plugin-yubikey makes one, named `ws-secrets`, on the terminal. Its PIN policy is
  `once`, a PIN once a session, and its touch policy `always`, a touch each time it decrypts.

The list prints a blank line for each YubiKey it reads, and nothing where none is plugged in. So a
list of nothing stops init with `no YubiKey found`, before it would read the Nano as empty.

The slot's stub, the identity file that `age-plugin-yubikey --identity` prints, goes to fnox's key
file. So fnox, run by ws or by hand in a clone, asks the Nano to decrypt. The stub names the Nano's
serial and slot, and holds no secret. The key file is the one fnox reads by default:
`FNOX_AGE_KEY_FILE`, else `$FNOX_CONFIG_DIR/age.txt`, else `$XDG_CONFIG_HOME/fnox/age.txt`, else
`~/.config/fnox/age.txt`. Its other lines stay, a missing directory is made 700, and the file is
600. The stub goes in by a new file renamed over the old one, so a failed write keeps the old file
whole, with any key of the user's. A key file that holds an SSH key stops it, as fnox would read
that key alone.

ws runs fnox with `/usr/local/bin` first on `PATH`, where fnox finds the plugin. It drops
`FNOX_AGE_KEY`, which would beat the key file, and `FNOX_PROFILE`, which would take a profile's file
for `fnox.toml`.

The backup is a plain age key that opens every value where the Nano is lost. The maintainer owns one
YubiKey alone, so a second one can't serve. ws makes it in Go, as macOS has no `age-keygen`, where
`fnox.toml` is missing. It prints its secret line once, before the push, to keep on paper or in a
password manager. It writes the line nowhere, and puts it in no argument, error or log. Enter goes
on; an end of input stops before the push, so nothing lands, and the next run makes a new backup.
A merged `fnox.toml` whose backup line was never shown would have no backup.

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

A Ctrl-C or a SIGTERM stops init, add and remove as a failure does, so that each removes its clone
before it ends.

No value enters a step's arguments: a commit's title and body, a branch and the files' names alone.

## The inputs and fnox.toml

Each server role that needs a secret declares it as an input in its `mise.secrets.toml`, under
mise's `[bootstrap.secrets]`. The input's variable is its secret's name in fnox too, as `fnox exec`
exports each secret under its own name. The Go package here reads the inputs of the roles in
`/opt/ws/roles/server/ubuntu`, each variable once. A variable is of capitals, digits and
underscores, and a role without the file declares none.

`fnox.toml` and its recipients, a table of labels, take the layout that the server's page gives
([The files](../../../server/ubuntu/secrets/README.md#the-files)). ws reads the labels and the
secrets' names, and no value. A list of recipients, or a provider of another type, stops it, as ws
changes a recipient by its label. A new `fnox.toml` holds the two first recipients and no secret
yet.

## Again

A later run keeps the Nano's identity, both recipients and every value. A `fnox.toml` without the
recipient `nano` or `backup` stops it: a new recipient needs every value encrypted again, with the
Nano, which init doesn't do. It declares the inputs that a newer release of ws adds, and lands them
by a pull request titled `chore: declare the inputs of a newer release`. Where none is new, it says
so and opens no pull request. Where `fnox set` leaves `fnox.toml` unchanged, it fails, as the
inputs would stay undeclared.

A pull request of init's that is still open stops a run before it reads the Nano, as after a check
that failed or a wait that ran out. A second one would make another backup, and both would add
`fnox.toml`. Merge it once its checks pass, or close it and run init again, which makes a new
backup.

A release may declare a new input. A server's run of it fails at mise, which finds no value of
the input, until init runs again on a Mac whose `/opt/ws` holds that release.

## A server's key

A server opens the values by its SSH host key, `/etc/ssh/ssh_host_ed25519_key`. Its public half,
`ssh-ed25519` and the key, is a recipient twice, under the label of the server's hostname: in
`fnox.toml`'s table and in its profile file's, `fnox.HOSTNAME.toml`
([The recipients](../../../server/ubuntu/secrets/README.md#the-recipients)).

The Go package here edits a table by its lines, `nano` first, then `backup`, then the servers by
name, and keeps every other byte of the file. A table with a line of another form, a comment
included, stops it. A new profile file holds the Nano's and the backup's recipients of `fnox.toml`,
and no value. A rebuilt server's key replaces the older one under its label. A key under another
label, of a server whose hostname changed, leaves that label first. A removed server's key leaves
every table, while its profile file stays with its values for a rebuild. Where the tables already
hold what they should, nothing changes, and nothing is encrypted again.

The hostname serves unchanged as the profile, the file's name, the label and the deploy key's
title. So it must be one DNS label: lowercase letters, digits and `-`, 63 at most, neither starting
nor ending with `-`. `nano` and `backup` are labels, `local` names `fnox.local.toml`, fnox's local
override, and `default` is fnox's profile without `-P`, so none of them serves. The server's task
refuses the same names.

## ws client secrets add

Run `ws client secrets add NAME` as yourself, with the Nano plugged in, once init has run
and ws's `install.sh` has ended ([the runbook](../../runbook.md#ws-on-ws)). NAME is a server of
the Mac's. Before the tunnel is up, `--address` and `--port` reach the server as ssh's `HostName`
and `Port`. The command:

1. Takes NAME's ed25519 key from `~/.ssh/known_hosts`. Where it holds none, it scans the server,
   prints the key's fingerprint, and asks whether the server's console prints the same. A `yes`
   records the key under NAME, and any other answer, or a Ctrl-C, stops it with no change.
2. Reads the server's hostname by `ssh NAME hostname`, which must serve as a profile, as above.
3. Lists the repository's deploy keys by gh, so that gh's login fails before a touch.
4. In a temporary clone, sets the key as above. It encrypts again the values of each changed file
   that holds one, by `fnox reencrypt`, which asks for the Nano's PIN and a touch.
5. Keeps one read-only deploy key titled with the hostname that holds the key, and deletes an older
   one first, as GitHub changes none in place.
6. Lands the changed files by a pull request, and opens none where the key was in place.
7. Runs the server's task `secrets` over `ssh -t`, as root by sudo, even where nothing landed.

Last, add prints a line for each host key that the secrets hold, as a recipient or a deploy key,
and that `~/.ssh/known_hosts` gives no host. A server that came back with another hostname and
another key leaves its older ones so. Its values hold no hostname, and `ssh-keygen -R NAME` dropped
its older key, so add cannot tell them from another server's. Drop such a key by hand where no
server holds it now.

A step that is done changes nothing, so a run that fails runs again whole. gh's login adds the
deploy keys, and needs admin on the repository. Where that login is revoked, GitHub deletes every
deploy key it added. Each server's fetch then fails until add runs again for it.

## ws client remove

`ws client remove NAME` ([The Mac's values](../../values/README.md)) takes the server's key
out of the secrets first. A removed server then opens no value. The key is NAME's ed25519 line in
`~/.ssh/known_hosts`. Where it holds none, remove cannot tell which key is NAME's: add may never
have run, or `ssh-keygen -R` emptied the line while the key stays a recipient. The secrets then
stay as they are, and remove prints a line to drop the hostname's recipients and deploy key by
hand. Else, in a temporary clone, remove:

1. Lists the repository's deploy keys by gh, so that gh's login fails before a touch.
2. Takes the key from every table of recipients, as above, and encrypts again the values of each
   changed file that holds one, with the Nano's PIN and a touch.
3. Lands the changed files by a pull request, and opens none where the key was no recipient.
4. Deletes each deploy key that holds the key, whatever its title.

A failure in any step stops remove with the server whole, so it runs again. The steps need the
network, gh's login and the Nano. Where they cannot work, `--skip-secrets` leaves the secrets as
they are. So does a Mac where init never ran, which has no fnox key file to encrypt with. remove
then goes on, and prints a line that names the key, to drop it by hand from a Mac that can. That
line holds the key's last copy on the Mac, as remove drops it from `~/.ssh/known_hosts`. A key that
remove leaves in the secrets so, add lists at its next run.
