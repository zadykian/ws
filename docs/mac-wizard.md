# The Mac's wizard

`mac/wizard.sh` sets up the Mac in stages, around `mac/setup.sh` ([The Mac](mac.md)). It asks for
the values that `~/.config/ws/mac.conf` lacks, makes the Mac's ssh key for ws and adds it to GitHub,
then runs `setup.sh`. It takes ws's host key, gives root on ws the Mac's key, runs ws's playbook
there, and logs gh in on ws. On a new Mac with Apple silicon, one line installs what it needs,
clones the repository and runs the wizard, as yourself:

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
4. Mac key: the ssh key that the Mac offers ws, made where missing, in Secretive by default, below.
5. gh on the Mac: gh's login to GitHub, and the Mac's key among the account's keys, below.
6. setup.sh: `mac/setup.sh`, after sudo's password. Until `mac.conf` has `WS_PUBLIC_KEY`, the
   wizard drops the Mac's key, which `setup.sh` prints at each run, and leaves the tunnel by hand.
7. Host key: ws's host key, matched against ws's console, in `known_hosts` under `ws`, below.
8. Root's key: the Mac's key in root's `authorized_keys` on ws, by the installer's user, below.
9. ws's run: `bootstrap.sh` on ws, in a unit of its own, which the wizard follows, below.
10. gh on ws: root's gh there, logged in to GitHub, and ws's signing key on the account, below.

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
and asks it once more. With no terminal to ask one, the key gets none.

Without `WS_IDENTITY`, and with no key yet, the wizard asks whether Secretive keeps the key, in the
Mac's Secure Enclave, which it never leaves. Yes, the default, installs Secretive's cask, and where
its app is there already, the wizard asks nothing. It opens the app, and waits up to 15 minutes for
a P-256 key made there, at its default protection, which asks for Touch ID at each login. Where git
has a name, a second key follows, for git's signing, at Notify, which signs with no prompt while
the Mac is unlocked ([git](mac-git.md)). Their `.pub` files and Secretive's agent go in `mac.conf`,
as `WS_IDENTITY`, `WS_IDENTITY_AGENT` and `WS_GIT_SIGNING_KEY`. An empty `WS_GIT_SIGNING_KEY`
keeps `setup.sh`'s file key for git. No makes `~/.ssh/id_ed25519` with `ssh-keygen`, as above. With
no terminal to ask, the wizard stops, unless `WS_IDENTITY` names a file.

The stage stops where the `.pub` is there without its key, the key without its `.pub`, or the two
differ, and says how to mend each. A key that an agent holds, at the socket of `WS_IDENTITY_AGENT`
or Secretive's, must be one that the agent lists, or the stage stops, naming the socket.

## GitHub and Ubuntu's install

gh, from Homebrew where missing, logs in to github.com by its device flow, which shows a code to
enter at `github.com/login/device`. Its scopes, `write:public_key` and `write:ssh_signing_key`, let
it add the Mac's keys, and a login without them is refreshed the same way. The wizard adds the
Mac's key for ws to the account's ssh keys, titled with the Mac's name. `setup.sh` then adds git's
signing key ([git](mac-git.md)). With no terminal, the stage stops, naming the command to run.

Ubuntu's installer imports the account's keys for ws's user. So run the line before Ubuntu's
install, or at any time. A run before the install stops at the Host key stage, as ws doesn't answer
yet. Leave ws's address empty there, and run the wizard again once Ubuntu is installed. In the
installer, turn OpenSSH on, import the keys from GitHub, and leave password login off. Where the
installer's user lacks the Mac's key, the Root's key stage runs `ssh-copy-id`, which needs password
login on.

## ws's host key

The stage reads ws's ed25519 key with `ssh-keyscan`, at the host and port that `ssh -G ws-ssh`
gives ([ssh's hosts](mac.md#sshs-hosts)). Where no way matches, it stops, naming
[The router](router.md). A new ws has a lease from the router's pool, not its static address. So
where ws gives no key within 10 s, the stage asks ws's address now, as the router's leases show it.
ssh takes that address for the rest of the run, and `mac.conf` keeps nothing of it. An empty answer
stops the run, as before Ubuntu's install.

The stage shows the key's fingerprint. At ws's console, `ssh-keygen -lf
/etc/ssh/ssh_host_ed25519_key.pub` prints ws's own: answer `yes` where they match, or paste what it
prints. Any other answer stops the run. The key goes in `known_hosts` as `ws`, the name that
`HostKeyAlias` gives each way. From then on, the stage reads nothing from ws. Through the router's
forward, the stage warns that a new ws faces the internet until its run sets up the `security` role
([security](security.md)). With no terminal, `WS_FIRST_ADDRESS` and `WS_HOST_KEY` give the
answers.

## Root's key

The stage is done where `ssh ws-ssh` logs in as root by the Mac's key. Else it asks the user that
Ubuntu's installer made, the Mac's user by default, who must log in by the key too. One `ssh -t`
session as that user then asks sudo's password once. It removes root's lines for the key, such as
one with cloud-init's forced command, matched on the key's base64. It adds the bare key, and keeps
root's other keys. The stage stops where neither user takes the key, and where `WS_USER` isn't root:
the [README](../README.md#a-new-machine)'s steps are then by hand. Where ws offers another host key
than `known_hosts` holds, as once ws is installed again, it stops, naming `ssh-keygen -R ws`. It
does so whatever `StrictHostKeyChecking` the user's `ssh_config` sets, and its logins add nothing to
`known_hosts`. With no terminal, `WS_INSTALLER_USER` gives the user.

## ws's run

The stage sends `mac/wizard-remote.sh`, the wizard's remote script, to root on ws over `ssh ws-ssh`.
So the script is the Mac's clone's, whatever ws's clone holds. It runs `bootstrap.sh` from ws's
clone in a unit of its own, `ws-wizard-STAMP`, through `systemd-run`, with root's `HOME` and a UTF-8
locale. Where ws has no clone, the unit runs the README's download of `bootstrap.sh` instead. The
wizard prints the unit's log until it ends, and stops where the run fails. A first run takes 20 to
40 minutes. The unit goes on where ssh drops, as when the Mac sleeps, and the next run of the wizard
follows it to its end.

Each run writes its unit and status in `/var/lib/ws-wizard/last` on ws. A run without arguments, as
the stage's, removes `/var/lib/ws-wizard/run` as it starts, and writes the clone's commit and the
status there as it ends. The stage is done where that status is 0 and no unit runs. So a second run
of the wizard runs no playbook, and one after a run cut short starts again. With `--update`, the
stage runs the playbook again, and the remote script first pulls ws's clone, by fast-forward, where
the clone is on `main` with nothing uncommitted. The run passes no `-e`, so the `ikeward` role
leaves the VPN as it finds it ([The profile](ikeward.md#the-profile)).

## gh on ws

The stage is done where root's gh on ws has a login to github.com, and the account's signing keys
hold ws's ([The signing key](devtools.md#the-signing-key)). Where root's `~/.gitconfig` names no
signing key file, as `shell_git_signing: false` leaves it, the login alone does
([Signing](shell.md#signing)). Where gh lacks its login, or the login the `write:ssh_signing_key`
scope, the wizard opens `github.com/login/device` in the Mac's browser. Then it runs gh's device
flow as root on ws, in an `ssh -t` session. That is `gh auth refresh` for the scope alone, else
`gh auth login -h github.com -p https -w -s write:ssh_signing_key`. Enter the code that gh shows
there at that page. So the login is ws's own, never a copy of the Mac's. With no terminal, the stage
stops, naming the command. Then, or at once where the account lacks the key alone,
`bootstrap.sh --tags devtools` runs in a unit of its own, as ws's run does, and adds the key. That
run needs no terminal, page or session. Where ws lacks gh, which its run installs, the stage stops:
`--update` runs the playbook again.

## What stays by hand

After a run in which a stage acted, the wizard lists each of these that applies:

- The Mac's tunnel: the keys' exchange of [The Mac's tunnel](mac.md#the-macs-tunnel), then
  `WS_PUBLIC_KEY` in `mac.conf`, then the wizard again. Its `setup.sh` loads ws-tunnel: allow it in
  the Background where macOS asks. Where ws lacks the Mac's key, or `wg0`'s key isn't
  `WS_PUBLIC_KEY`, as once ws is installed again, the exchange again.
- claude's login, as root on ws ([By hand](../README.md#by-hand)).
- With Secure Boot on, where the kernel refuses the fans' driver, DKMS's key, enrolled at ws's
  console ([Secure Boot](fans.md#secure-boot)).
- What [The Mac](mac.md) leaves by hand: macOS's grant of the local network to an app, and
  Gateway's connection to ws.
- ws's lease, made static, where the Host key stage asked ws's address, and the router's forwards
  where `mac.conf` has `WS_REMOTE` ([The router](router.md)).
