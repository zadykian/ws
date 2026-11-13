# ws's secrets

The secrets of the roles are one file, `/etc/ws/secrets.age` on ws ([ws's file](#wss-file)): TOML
that age encrypts, in armor. The Mac keeps neither it nor its identity. Git never holds it. `ws
server ubuntu apply --secrets` decrypts it in its process ([The secrets
file](../roles/server/ubuntu/shell/README.md#the-secrets-file)). A value is never in an error, a
plan or a diff: they name kinds, names, keys and paths.

## From the Mac

`ws client macos secrets` runs its own of `ws server ubuntu secrets` on ws, as root, through the
wizard's ssh and its way to root ([ws's file](#wss-file)). So `ws client macos secrets init` makes
the identity on ws. A value goes on ssh's standard input, never in an argument. Where ws's sudo asks
a password, a command that reads its value from standard input asks it at the terminal, `/dev/tty`,
and fails, saying why, with none. With ws out of reach, a command says so and fails. The Configure
screen sets the secrets the same way ([The Configure screen](tui-configure.md)).

## Editing

On ws, each command reads the identity, decrypts the file in memory, edits it, and replaces the file
through a renamed temporary file, mode 600. No plain file is made, and no editor runs. `set` parses
the result again and refuses what a parse refuses, before it writes. Its references stay as written.

```sh
ws client macos secrets set token github                 # also azure_devops
ws client macos secrets set env MY_TOKEN                 # any shell name
ws client macos secrets set password vpn_p12
ws client macos secrets set git https://git.example.com --username bob --proactive-auth
ws client macos secrets set file vpn_p12 vpnclient.p12   # vpn_sswan too: the file whole
ws client macos secrets set file vpn_sswan <ws.sswan     # or the file on standard input
ws client macos secrets unset env MY_TOKEN
ws client macos secrets keys
```

A value comes from a prompt with no echo at a terminal, else from standard input, less one trailing
newline. So `set` never takes a value in an argument. `set file` reads its path whole, or standard
input whole without one, but not a terminal. A git entry takes the token so, and the flags set its
username and proactive auth: one left out keeps the entry's. `unset` fails where the entry is
missing. A name that ws does not know fails, and the error lists the kind's names. `keys` prints a
`KIND NAME` line per entry, sorted, and `git URL` for a server: never a value. `set` without ws's
credential fails and names `ws server ubuntu secrets init`.

## The run

The run stage of `ws client macos wizard` runs `--secrets` on ws's file, and sends nothing from the
Mac ([ws's run](mac-wizard.md#wss-run)). The file's SHA-256 is in the run's stamp, so a changed file
runs the stage again.

## ws's file

`ws server ubuntu secrets` keeps the file on ws, root's and mode 600, and runs as root alone. Its
`set`, `unset` and `keys` are those above. `init` makes the identity in memory and gives it to
`systemd-creds encrypt --with-key=host` on its standard input. ws writes what that prints as the
credential `/etc/credstore.encrypted/ws-secrets`, mode 600 in a directory of 700. The host's key in
`/var/lib/systemd` encrypts it, not the TPM's, which a change of firmware or of Secure Boot can lock
out. Each command decrypts the identity by `systemd-creds decrypt` into memory. `init` keeps a
credential and a file that are there. It refuses a file without the credential, which a new identity
would not open. A unit reads the credential by `LoadCredentialEncrypted=ws-secrets` ([The secrets
file](../roles/server/ubuntu/shell/README.md#the-secrets-file)).

```sh
ws server ubuntu secrets init
ws server ubuntu secrets set file vpn_sswan <ws.sswan
ws server ubuntu secrets keys
```

## The tables

A table per kind. ws refuses a key it does not know, a key twice and a value of the wrong type. A
syntax error names its line alone.

```toml
[token]
github = "MY_TOKEN"                 # also azure_devops

[env]                               # ~/.secrets/env: any shell name
MY_TOKEN = "${token.github}"

[git."https://git.example.com"]     # an https:// URL
token = "${token.azure_devops}"
username = "bob"                    # token by default
proactive_auth = true

[file]                              # whole files, in base64
vpn_p12 = "BASE64"
vpn_sswan = "BASE64"

[password]
vpn_p12 = "PASSWORD"                # left out where the .p12 has none
```

- `[token]` names `github` and `azure_devops`, the tokens ws knows.
- `[env]` is the content of `~/.secrets/env` ([Secrets](../roles/server/ubuntu/shell/README.md#secrets)).
- `[git."URL"]` is a server's token in git's store ([Git credentials](../roles/server/ubuntu/shell/README.md#git-credentials)).
- `[file]` and `[password]` are the ikeward profile's ([The profile](../roles/server/ubuntu/ikeward/README.md#the-profile)). A
  TOML string can't hold the `.p12`'s bytes, so a file is base64. ws decodes it once parsed.

## References

`${kind.name}` in a string of `env`, `git` (token and username) or `password` stands for the value
of a `token`, an `env` or a `password` of that name. A reference may lead to another. `$$` is `$`,
and any other `$` fails. A file takes none.

A name that is missing, or a loop, fails the run, naming the names alone, as `env.A refers to
token.github, which is missing`.

## The whole truth

The file is the truth of what it names. With `--secrets`, a run removes the git tokens for URLs
the file lacks, and `~/.secrets/env` where `[env]` is empty. Without it, as in CI, a run keeps
both. So an entry dropped from the file goes with the next run.
