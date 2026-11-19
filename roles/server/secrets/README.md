# ws's secrets

The secrets of the roles are one file, `/etc/ws/secrets.age` on ws ([ws's file](#wss-file)): TOML
that age encrypts, in armor. The Mac keeps neither it nor its identity. Git never holds it. `ws
unit` decrypts it in its process for a unit that reads it ([The run](#the-run)). A value is never
in an error, a plan or a diff: they name kinds, names, keys and paths.

## From the Mac

The commands run on ws, as root ([ws's file](#wss-file)). From the Mac, `ssh -t NAME sudo ws server
ubuntu secrets init` makes the identity there. A value goes on standard input, never in an argument.

## Editing

On ws, each command reads the identity, decrypts the file in memory, edits it, and replaces the file
through a renamed temporary file, mode 600. No plain file is made, and no editor runs. `set` parses
the result again and refuses what a parse refuses, before it writes. Its references stay as written.

```sh
sudo ws server ubuntu secrets set token github  # also azure_devops
sudo ws server ubuntu secrets set env MY_TOKEN  # any shell name
sudo ws server ubuntu secrets set password vpn_p12
sudo ws server ubuntu secrets set git https://git.example.com --username bob --proactive-auth
sudo ws server ubuntu secrets set file vpn_p12 vpnclient.p12  # vpn_sswan too: the file whole
sudo ws server ubuntu secrets set file vpn_sswan <ws.sswan  # or the file on standard input
sudo ws server ubuntu secrets unset env MY_TOKEN
sudo ws server ubuntu secrets keys
```

A value comes from a prompt with no echo at a terminal, else from standard input, less one trailing
newline. So `set` never takes a value in an argument. `set file` reads its path whole, or standard
input whole without one, but not a terminal. A git entry takes the token so, and the flags set its
username and proactive auth: one left out keeps the entry's. `unset` fails where the entry is
missing. A name that ws does not know fails, and the error lists the kind's names. `keys` prints a
`KIND NAME` line per entry, sorted, and `git URL` for a server: never a value. `set` without ws's
credential fails and names `ws server ubuntu secrets init`.

## The run

A run of the bootstrap reads ws's file where it stands, and sends nothing from the Mac. A changed
file changes what the units that read it plan, so the next run writes it. `ws unit`, as root,
decrypts the identity by `systemd-creds decrypt`, once a command, and only where a unit reads the
secrets. It reads no standard input and no terminal. With the file and without the credential,
such a unit fails.

## ws's file

`ws server ubuntu secrets` keeps the file on ws, root's and mode 600, and runs as root alone. Its
`set`, `unset` and `keys` are those above. `init` makes the identity in memory and gives it to
`systemd-creds encrypt --with-key=host` on its standard input. ws writes what that prints as the
credential `/etc/credstore.encrypted/ws-secrets`, mode 600 in a directory of 700. The host's key in
`/var/lib/systemd` encrypts it, not the TPM's, which a change of firmware or of Secure Boot can lock
out. Each command decrypts the identity by `systemd-creds decrypt` into memory. `init` keeps a
credential and a file that are there. It refuses a file without the credential, which a new identity
would not open. `ws unit` decrypts the identity the same way ([The run](#the-run)).

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

[env]                               # /etc/ws/env: any shell name
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
- `[env]` is read by no role: `/etc/ws/env` comes from mise's secret inputs ([The env
  file](../ubuntu/shell/README.md#the-env-file)).
- `[git."URL"]` is a server's token in git's store ([Git credentials](../ubuntu/git/README.md#git-credentials)).
- `[file]` and `[password]` are the ikeward profile's ([The profile](../ubuntu/ikeward/README.md#the-profile)). A
  TOML string can't hold the `.p12`'s bytes, so a file is base64. ws decodes it once parsed.

## References

`${kind.name}` in a string of `env`, `git` (token and username) or `password` stands for the value
of a `token`, an `env` or a `password` of that name. A reference may lead to another. `$$` is `$`,
and any other `$` fails. A file takes none.

A name that is missing, or a loop, fails the run, naming the names alone, as `env.A refers to
token.github, which is missing`.

## The whole truth

The file is the truth of what it names. Where the machine has it, a run removes the git tokens
for URLs the file lacks, and `/etc/ws/env` where `[env]` is empty. Without it, as in CI, a run
keeps both. So an entry dropped from the file goes with the next run.
