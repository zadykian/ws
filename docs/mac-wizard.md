# The Mac's wizard

`mac/wizard.sh` sets up the Mac in stages, around `mac/setup.sh` ([The Mac](mac.md)). It asks for
the values that `~/.config/ws/mac.conf` lacks, makes the Mac's ssh key for ws and adds it to GitHub,
then runs `setup.sh`. It takes ws's host key, checks that root on ws takes the Mac's key, runs ws's
playbook there, and logs gh and claude in on ws. Then it exchanges the tunnel's keys, runs another
repository's `apply.sh` on ws, and clones that repository on the Mac, with ws's word list. Last, it
sets up the cable from the Mac's display to ws. On a new Mac with Apple silicon, one line installs
what it needs, clones the repository and runs the wizard, as yourself:

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
6. setup.sh: `mac/setup.sh`, after sudo's password. The wizard drops the Mac's key that it prints.
7. Host key: ws's host key, matched against ws's console, in `known_hosts` under `ws`, below.
8. Root's key: root's login to ws by the Mac's key, checked and never added, below.
9. ws's run: `bootstrap.sh` on ws, in a unit of its own, which the wizard follows, below.
10. gh on ws: root's gh there, logged in to GitHub, and ws's signing key on the account, below.
11. claude on ws: root's claude there, logged in, below.
12. Tunnel: the Mac's WireGuard key as `wg0`'s peer on ws, and `wg0`'s as `WS_PUBLIC_KEY`, below.
13. Hook on ws: another repository's `apply.sh`, as root on ws, where `mac.conf` names one, below.
14. Hook on the Mac: that repository's clone in `~/repository/REPO`, its hooks and its `Brewfile`,
    and ws's word list, below.
15. Cable: on the LAN, ws's port of the cable in its `host_vars`, and the Mac's service, below.

## The answers

| Question | Kept in | Default | Asked |
|---|---|---|---|
| The machine ws runs on: `lan`, `cloud` or `mac-vm` | `WS_TARGET` | `lan` | always |
| ws's home LAN | `WS_LAN` | `10.88.0.0/24` | but on a cloud VM |
| ws's address on the LAN | `WS_LAN_ADDRESS` | `10.88.0.10`, where the LAN holds it | with a LAN |
| The router's name on the internet | `WS_REMOTE` | none | but on a VM on the Mac |
| The router's port forwarded to ws's 22 | `WS_REMOTE_PORT` | `22022` | with a name |
| git's name on the Mac | `WS_GIT_NAME` | the `shell` role's, `zadykian` | always |
| git's email on the Mac | `WS_GIT_EMAIL` | the `shell` role's, `me@zadykian.dev` | with a name |
| More networks through the tunnel | `WS_TUNNEL_NETWORKS` | none | always |
| Nameservers behind the tunnel | `WS_RESOLVERS` | none | always |
| Another repository whose `apply.sh` runs on ws | `WS_HOOK` | none | always |
| The Keychain item of its `apply.sh`'s secret | `WS_HOOK_SECRET` | none | with a repository |

Each answer goes in `mac.conf`, under the key that [mac.conf](mac.md#macconf) explains, but the
first and the last two. `WS_TARGET` names ws's machine: one on the LAN, a cloud VM, which needs
`WS_REMOTE` and `WS_REMOTE_PORT`, or a VM on the Mac, which needs `WS_LAN`. The table gives the
questions on the LAN, and [The wizard's targets](mac-targets.md) the others, with the keys that each
refuses. [Another repository](#another-repository) explains the last two. The defaults are
[The router](router.md)'s LAN, lease and forward, and ws's own git identity. A key's answer comes
from the first of these that has one:

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
The Cable stage asks `WS_CABLE_PORT` and `WS_CABLE_SERVICE` ([The cable](#the-cable)). The other
keys, such as `WS_USER` and `WS_PORT`, keep their defaults until set by hand.

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
install, as the wizard adds no key to ws afterwards. A run before the install stops at the Host key
stage, as ws doesn't answer yet. Leave ws's address empty there, and run the wizard again once
Ubuntu is installed. In the installer, turn OpenSSH on, import the keys from GitHub, and leave
password login off.

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
answers. A linked `known_hosts` stops the stage, and a run prints the key's line for its target.

## Root's key

The stage is done where `ssh ws-ssh` logs in as root by the Mac's key. The wizard adds no key to
ws: GitHub's account is the only way to give ws its keys. Where root doesn't take the key, the
stage stops, at a terminal or not, and names the Mac's `.pub`. Ubuntu's installer imports the
account's keys for its user, and root takes them as the [README](../README.md#a-new-machine) says:
copy that user's `authorized_keys` to root. That replaces root's file, so cloud-init's forced
command goes too. A cloud VM gives its default user the provider's keys instead, which must be the
account's. Then run the wizard again. `--check` says that a run would stop. Both show any ssh error.

The stage stops too where `WS_USER` isn't root: the README's steps are then by hand. Where ws offers
another host key than `known_hosts` holds, as once ws is installed again, it stops, naming
`ssh-keygen -R ws`. It does so whatever `StrictHostKeyChecking` the user's `ssh_config` sets, and
its logins add nothing to `known_hosts`.

## ws's run

The stage sends `mac/wizard-remote.sh`, the wizard's remote script, to root on ws over `ssh ws-ssh`.
So the script is the Mac's clone's, whatever ws's clone holds. It runs `bootstrap.sh` from ws's
clone in a unit of its own, `ws-wizard-STAMP`, through `systemd-run`, with root's `HOME` and a UTF-8
locale. Where ws has no clone, the unit runs the README's download of `bootstrap.sh` instead. The
wizard prints the unit's log until it ends, and stops where the run fails. A first run takes 20 to
40 minutes. The unit goes on where ssh drops, as when the Mac sleeps, and the next run of the wizard
follows it to its end.

Each run writes its unit and status in `/var/lib/ws-wizard/last` on ws. A full run, as the stage's,
with `-e` pairs alone, removes `/var/lib/ws-wizard/run` as it starts. As it ends, it writes the
clone's commit and the status there, and `off` where its `-e` turned the outbound table off. The
stage is done where that status is 0 and no unit runs. So a second run of the wizard runs no
playbook, and one after a run cut short starts again. With `--update`, the stage runs the playbook
again, and the remote script first pulls ws's clone, by fast-forward, where the clone is on `main`
with nothing uncommitted. The run passes no file of the `ikeward` role's variables, so the role
leaves the VPN as it finds it ([The profile](ikeward.md#the-profile)).

On a cloud VM, the outbound table drops the metadata service ([Outbound](security.md#outbound)). So
with `WS_TARGET=cloud`, the run passes `-e security_outbound=false`, and its unit then names that
in ws's `host_vars`, for each run after. The stage is done there once a run with that `-e` passed
and `host_vars` has the line. After a unit without it, which the run followed, a run of its own
goes. Where the line alone is missing, the stage writes it, and runs no playbook.

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
run needs no terminal, page or session. The stage stops where ws lacks gh, which a run with
`--update` installs. It stops with gh's error where gh fails to list the account's keys otherwise.

## claude on ws

The stage is done where root's claude on ws has a login, as `claude auth status` says. Else the
wizard runs `claude auth login` as root on ws, in an `ssh -t` session, through a login shell, whose
`PATH` has `~/.local/bin`, where claude is ([claude](claude.md)). Take its steps at the terminal,
with the address it shows opened in the Mac's browser. With no terminal, the stage stops, naming
`claude auth login`. Where ws lacks claude, it stops, as gh's stage does.

## The tunnel

The stage writes the Mac's WireGuard key to `wg0`'s `mac.pub` on ws, where that holds
another, and `bootstrap.sh --tags wireguard` gives `wg0` that peer. That run goes again where `wg0`
lacks the peer, as after one that failed. Both go over `ws-ssh`, outside the tunnel, whose sessions
the change ends ([The tunnel](wireguard.md#the-tunnel)). Then `WS_PUBLIC_KEY` in `mac.conf` takes
`wg0`'s key, in place of its line, and `setup.sh` runs again, which loads ws-tunnel
([The Mac's tunnel](mac.md#the-macs-tunnel)): allow it in the Background. It then waits up to 30 s
for ws-tunnel's handshake, and logs in once by `ssh ws`. Else it stops, naming the UDP port, the
router's forward and the Background. Where the run asked ws's address, a later run checks it.

## Another repository

`WS_HOOK` names another repository on GitHub, as `OWNER/REPO`, whose `apply.sh` sets up more of ws,
such as values that this public repository leaves out. `WS_HOOK_SECRET` names the Keychain item of a
secret that the script reads. `setup.sh` takes both keys, and reads neither. The stage clones the
repository in `/root/repository/REPO` on ws by root's gh login, where missing, and never pulls it,
as the script updates its own clone. A `REPO` of `ws`, ws's own clone, or another repository's clone
there stops the stage. Where the Keychain lacks the item, it runs `security add-generic-password`,
which asks for its secret twice at the terminal. Where `security find-generic-password -w` can't
read it, the stage stops, naming its error. Then `apply.sh` runs as root over `ssh ws-ssh`, never
`ssh -t`, so it neither prompts nor echoes its stdin: what security prints, then the end of input,
or empty without an item. The wizard never reads the secret, and passes the script no arguments.

That run leaves a stamp, `/var/lib/ws-wizard/hook-REPO` on ws, and the stage is done. So a second
run leaves `apply.sh` alone, and `--update` runs it again. A run that fails stops the wizard, and
the next runs it again. An item whose access list lacks `security` asks at both of the stage's
reads, which "Always Allow" answers. `--check` says what the stage would do, and runs nothing.

The stage Hook on the Mac then clones the repository in `~/repository/REPO` by gh's login on the
Mac, where missing, and never pulls it. Another repository's clone there stops the stage, as on ws.
Where the clone has `.githooks`, git's `core.hooksPath` names it by an absolute path, as in ws's
clone ([The hooks](checks.md#the-hooks)). Where it has a `Brewfile` whose formulae Homebrew lacks,
as `brew bundle check` finds, with Homebrew's auto-update off, `brew bundle` installs them, and
upgrades nothing. `--check` names each command that the stage would run.

The stage copies ws's word list too, root's `~/.config/ws/forbidden-words`, which the hooks read
([The hooks](checks.md#the-hooks)), with `WS_HOOK` or without. Where ws has one, and the Mac's
differs by git's hash or can't be read, the files script, `mac/wizard-remote-files.sh`, sends it
from ws. It replaces the Mac's, mode 600, but a link there stops the stage, `--check` too, naming
its target. The wizard never prints the list: `--check` says it would copy it, with no diff.

## The cable

The stage sets up the cable from the Mac's display to a port of ws's, outside the router
([The direct link](network.md#the-direct-link)). A cloud VM and a VM on the Mac skip it. The
wizard's files script lists ws's wired ports but the uplink, each with its carrier and IPv4
addresses. Where none has carrier, as the Mac isn't
docked, the stage waits, and the closing list says to dock the Mac and run the line again. Docked,
the stage asks ws's port of the cable, the one with carrier by default. It names the port in
`host_vars/localhost.yml` on ws, then runs `bootstrap.sh --tags network,wireguard` there, as both
roles read it. Once the port holds `10.77.0.1`, the stage asks the Mac's service of the cable. The
default is the one service with a `169.254` address, which an adapter gives itself where no DHCP
server answers. The answer goes in `mac.conf` as `WS_CABLE_SERVICE`, then `setup.sh` gives the
service `10.77.0.2` ([The cable](mac.md#the-cable)), and `ping 10.77.0.1` must get an answer.

`none` as either answer writes an empty `WS_CABLE_SERVICE`, so the wizard asks neither again, as
where ws has no cable. From the environment, at a terminal too, `WS_CABLE_PORT` and
`WS_CABLE_SERVICE` give the answers. `mac.conf` keeps the port as `WS_CABLE_PORT`, to change there.
Where `host_vars` lacks it, as after ws's install, the stage names it there and runs the roles,
undocked too, but stops on a port ws lacks. `--check` asks nothing, and says what the stage would
do.

Undocked, an empty `WS_CABLE_SERVICE` in the environment ends the wait, for a Mac without the cable.
So does `WS_CABLE_PORT`, for a port that ws keeps down, as no netplan file names it, so the files
script can't read its carrier. The roles' run brings it up. The closing list names such ports.

## What stays by hand

After a run in which a stage acted, the wizard lists each of these that applies:

- With Secure Boot on, where the kernel refuses the fans' driver, DKMS's key, enrolled at ws's
  console ([Secure Boot](fans.md#secure-boot)).
- What [The Mac](mac.md) leaves by hand: macOS's grant of the local network to an app, and
  Gateway's connection to ws.
- ws's lease, made static, where the Host key stage asked ws's address, and the router's forwards
  where `mac.conf` has `WS_REMOTE` ([The router](router.md)).
- ws's outbound table, which `security_outbound: false` in ws's `host_vars` keeps off, where
  `WS_TARGET` isn't `cloud`, as after a move from a cloud VM: that line removed, then `--update`.
- The tunnel's handshake, where the Host key stage asked ws's address: once ws's lease is static,
  the wizard again, which checks it.
- `gh auth setup-git`, where git's config names no credential command of gh's for github.com, as
  `setup.sh`'s does with git's name ([git](mac-git.md)). Through it, pushes from the Mac's clones
  take gh's login.
- The cable, where no port of ws's had carrier: the line again, docked, or with `WS_CABLE_PORT` for
  a port that ws keeps down. Without the cable, the line once with an empty `WS_CABLE_SERVICE`.
