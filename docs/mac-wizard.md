# The Mac's wizard

`mac/wizard.sh` sets up the Mac in stages, around `mac/setup.sh` ([The Mac](mac.md)). It asks for
the values that `~/.config/ws/mac.conf` lacks, makes the Mac's ssh key for ws, then runs `setup.sh`.
On a new Mac with Apple silicon, one line installs what it needs, clones the repository and runs the
wizard, as yourself:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/mac/bootstrap.sh | sh
```

`mac/bootstrap.sh` hands its arguments to the wizard, given as `sh -s -- --check`. From a clone,
`~/repository/ws/mac/wizard.sh` runs the wizard alone.

Each stage reads what it sets up, acts where that isn't done, and reads it again. So a run takes up
where the last one stopped, and a second prints nothing. `--check` prints what each stage would do,
and changes nothing. The wizard holds a lock, `ws-wizard.lock` in `$TMPDIR`. Both it and
`bootstrap.sh` keep the Mac from idle sleep. A run in which a stage acted ends with what stays by
hand. The stages, in order, the first two `bootstrap.sh`'s, before the wizard starts:

1. Tools: the Command Line Tools, through the dialog that `xcode-select --install` opens, waited
   on for up to 60 minutes. Then [Homebrew](https://brew.sh), from the `Homebrew.pkg` of a release
   that `bootstrap.sh` pins, checked by its SHA-256 and signer, through `sudo installer`. The
   `.pkg` puts `/opt/homebrew/bin` on new shells' `PATH`.
2. Clone: the repository's clone in `~/repository/ws`, made where missing and otherwise left as
   found, with git's `core.hooksPath` at its `.githooks` ([The hooks](checks.md#the-hooks)).
   With `--update`, it pulls a clone on `main` with nothing uncommitted, by fast-forward alone,
   and warns of any other. Without a clone, `--check` lists the wizard's stages, each after this
   one.
3. Answers: the values `mac.conf` lacks, below.
4. Mac key: the ssh key that the Mac offers ws, made where missing, below.
5. setup.sh: `mac/setup.sh`, after sudo's password. Until `mac.conf` has `WS_PUBLIC_KEY`, the
   wizard drops the Mac's key, which `setup.sh` prints at each run, and leaves the tunnel by hand.

## The answers

| Question | Kept in | Default | Asked |
|---|---|---|---|
| ws's home LAN | `WS_LAN` | `10.88.0.0/24` | always |
| ws's address on the LAN | `WS_LAN_ADDRESS` | `10.88.0.10`, where the LAN holds it | with a LAN |
| The router's name on the internet | `WS_REMOTE` | none | always |
| The router's port forwarded to ws's 22 | `WS_REMOTE_PORT` | `22022` | with a name |
| git's name on the Mac | `WS_GIT_NAME` | the `shell` role's, `zadykian` | always |
| git's email on the Mac | `WS_GIT_EMAIL` | the `shell` role's, `me@zadykian.dev` | with a name |
| More networks through the tunnel | `WS_TUNNEL_NETWORKS` | none | always |
| Nameservers behind the tunnel | `WS_RESOLVERS` | none | always |

Each answer goes in `mac.conf`, under the key that [mac.conf](mac.md#macconf) explains. The
defaults are [The router](router.md)'s LAN, lease and forward, and ws's own git identity. A key's
answer comes from the first of these that has one:

- `mac.conf`'s line for the key, even an empty one, so the wizard never asks it again;
- the environment, under the key's name, as `WS_LAN=10.88.0.0/24`, or `WS_LAN=` for no value;
- the terminal on the wizard's stdin, where an empty line takes the default in brackets, and `none`
  no value;
- else nothing: the run stops with `no terminal to ask KEY: set KEY`.

A bad answer at the terminal is asked again, with the form the key takes. From the environment, it
stops the run. ws's address and git's email take a value where asked. The wizard adds the answers
at the end of `mac.conf`, and keeps its comments and order. It writes the file once `setup.sh` would
take the whole. It stops before it asks on a `mac.conf` it can't read, or on a line that `setup.sh`
would refuse. Where `mac.conf` is a link, it prints the lines to add to the link's file, and stops.
The other keys, such as `WS_USER` and `WS_PORT`, keep their defaults until set by hand.

Under `--check`, the wizard asks too, and shows the lines it would add as a diff. `setup.sh --check`
then reads the file it would write, in the lock, which `WS_CONF` names. `WS_CONF` names another
`mac.conf` for both scripts.

## The Mac's key

ssh offers ws the key whose `.pub` `mac.conf`'s `WS_IDENTITY` names ([mac.conf](mac.md#macconf)).
Where that is a file, as `~/.ssh/id_ed25519.pub`, and neither the key nor its `.pub` is there,
`ssh-keygen -t ed25519 -a 100` makes it, and asks its passphrase twice.
`ssh-add --apple-use-keychain` then gives the key to the agent and the passphrase to the Keychain,
and asks it once more. Without `WS_IDENTITY`, the wizard makes `~/.ssh/id_ed25519` the same way.
With no terminal to ask one, the wizard stops there, unless `WS_IDENTITY` names that file, which
then gets no passphrase.

The stage stops where the `.pub` is there without its key, the key without its `.pub`, or the two
differ, and says how to mend each.

## What stays by hand

- The Mac's tunnel: the keys' exchange of [The Mac's tunnel](mac.md#the-macs-tunnel), then
  `WS_PUBLIC_KEY` in `mac.conf`, then the wizard again.
- What [The Mac](mac.md) leaves by hand: ws's host key at the first login, macOS's grant of the
  local network to an app, and Gateway's connection to ws.
