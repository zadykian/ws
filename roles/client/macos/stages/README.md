# The Mac's wizard

`ws client macos wizard` sets up the Mac in stages, around `ws client macos apply` ([The
Mac](../README.md)). It asks for the values that the Mac's files lack, makes the Mac's ssh
key for ws and adds it to GitHub, then runs `ws client macos apply`. It takes ws's host key and
checks that `WS_USER` on ws takes the Mac's key. It runs `ws server ubuntu apply` there, adds ws's
signing key to GitHub by the Mac's gh, and logs claude in on ws. Then it exchanges the tunnel's
keys. Last, it sets up the cable from the Mac's display to ws. On a new Mac with Apple silicon, one
line installs ws's binary, and the wizard installs what else it needs, as yourself:

```sh
curl -fsSL https://github.com/zadykian/ws/releases/latest/download/install.sh | sh
```

Then `~/.local/bin/ws client macos wizard` opens [the stages' lists](../../../../engine/tui/internal/queue/README.md), with `--check`
to see what it would do. Each stage reads what it sets up, acts where that isn't done, and reads it
again. So a run takes up where the last one stopped, and in a second no stage acts. `--check` prints
what each stage would do, and changes nothing. The wizard holds a lock, `ws-wizard.lock` in
`$TMPDIR`, and keeps the Mac from idle sleep. A run in which a stage acted ends with what stays by
hand. The stages, in order, the first two of which install what the rest needs:

1. Tools, then Homebrew: the Command Line Tools, through the dialog that `xcode-select --install`
   opens, waited on for 60 minutes. Then [Homebrew](https://brew.sh), from the `Homebrew.pkg` that
   `roles/client/macos/stages/tools/homebrew.go` pins, checked by its SHA-256 and signer, through `sudo
   installer`. On a new Mac, `--check` stops there.
2. Answers: the server's name, asked first where none is set up, then the values the files lack.
3. Mac key: the ssh key that the Mac offers ws, and git's, asked among Secretive's keys, below.
4. gh on the Mac: gh's login to GitHub, and the Mac's key among the account's keys, said if there.
5. ws client macos apply: `ws client macos apply`, after sudo's password, asked at each run,
   `--check` too, unless its timestamp is fresh. With no terminal, the stage stops, unless sudo
   needs no password. It shows the warnings of `ws client macos apply` and drops the Mac's key that
   it prints.
6. Host key: ws's host key, matched against ws's console, in `known_hosts` under `ws`, below.
7. Login: `WS_USER`'s login to ws by the Mac's key, checked and never added, below.
8. Sudo: how the remote scripts reach root on ws, with sudo's password asked where it needs one,
   below.
9. ws on ws: this ws's release, installed on ws by its `install.sh`, below.
10. ws's run: `ws server ubuntu apply` on ws, in a unit of its own, which the wizard follows, below.
11. Signing key: ws's public signing key among the signing keys of the Mac's gh login, below.
12. claude on ws: `WS_USER`'s claude there, logged in, below.
13. Tunnel: the Mac's WireGuard key as `wg0`'s peer on ws, and `wg0`'s as `WS_PUBLIC_KEY`, below.
14. Cable: on the LAN, ws's port of the cable in its `host_vars.yml`, and the Mac's service, below.

Each stage states the side it acts on, in its own file by `Side()`. Sudo, ws on ws, ws's run
and claude on ws are ws's, and the rest are the Mac's. A stage that changes both is the Mac's where
its last change lands there, as the Tunnel and the Cable do.

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

Each answer goes in the file of its key, which [mac.conf](../README.md#macconf) explains, but the
first. `WS_TARGET` names ws's machine: one on the LAN, a cloud VM, which needs `WS_REMOTE` and
`WS_REMOTE_PORT`, or a VM on the Mac, which needs `WS_LAN`. The table gives the questions on the
LAN, and [The wizard's targets](../conf/README.md) the others, with the keys that each refuses. The
defaults are [The router](../../../server/ubuntu/network/router.md)'s LAN, lease and forward, and ws's own git identity. A key's
answer comes from the first of these that has one:

- the file's line for the key, even an empty one, so the wizard never asks it again;
- the environment, under the key's name, as `WS_LAN=10.88.0.0/24`, or `WS_LAN=` for no value;
- the terminal on the wizard's stdin, where an empty line takes the default in brackets, and `none`
  no value. A screen asks `WS_TARGET` as a choice of its three values, on the default first;
- else nothing: the run stops with `no terminal to ask KEY: set KEY`.

A bad answer at the terminal is asked again, with the form the key takes. From the environment, it
stops the run. ws's address and git's email take a value where asked. The wizard adds the answers at
the end of each file, keeps their comments and order, and adds `WS_SLOT=0` to a server's file that
lacks it. It writes `mac.conf`, then the server's file, once `ws client macos apply` would take
both. It stops before it asks on a file it can't read, or on a line that `ws client macos apply`
would refuse. Where a file is a link and lacks a key, it prints the lines to add to the link's file,
and stops. The Cable stage asks `WS_CABLE_PORT` and `WS_CABLE_SERVICE` ([The cable](#the-cable)).
The other keys, such as `WS_USER` and `WS_PORT`, keep their defaults until set by hand.

Under `--check`, the wizard asks too, and shows the lines it would add as a diff. `ws client macos
apply --check` then reads the files it would write, in the lock, which `WS_CONF` names. `WS_CONF`
names another `mac.conf` for both scripts.

## The Mac's key

ssh offers ws the key whose `.pub` `mac.conf`'s `WS_IDENTITY_PUB_KEY` names
([mac.conf](../README.md#macconf)). Where that is a file, as `~/.ssh/id_ed25519.pub`, the key and its
`.pub` must be there, as the wizard makes no key in a file. Where one is missing, it stops, naming
the line to drop for the wizard to ask.

Where `mac.conf` names none, or an empty one, the wizard asks for ws's login key. It offers each
P-256 key of Secretive's agent, by comment and SHA256 fingerprint, and a new one. The new one is the
default, so that a keypress never takes a key made for another use. Where git has a name, and
`mac.conf` an empty `WS_GIT_SIGNING_PUB_KEY` or none, it asks for git's key too, the login key
marked ([git](../git/README.md)). One key may serve both. For a new key, the wizard installs Secretive's
cask where its app is missing, and opens the app. It waits up to 15 minutes for a P-256 key made
there, in the Mac's Secure Enclave, which it never leaves. Where the agent wasn't there to ask, a
key it lists once it starts stops the wizard, as no answer named it. ws's key takes Secretive's
default protection, which asks for Touch ID at each login. The wizard takes each key from the agent,
and writes a copy of its `.pub` ([Secretive's keys](../git/README.md#secretives-keys)). The copies and the
agent go in `mac.conf`, as `WS_IDENTITY_PUB_KEY`, `WS_IDENTITY_AGENT` and `WS_GIT_SIGNING_PUB_KEY`.
With no terminal, the wizard stops at the first question, naming the line of each such key. Under
`--check`, the ws client macos apply stage waits on the answer, as `ws client macos apply` stops
without the key.

The stage stops where the `.pub` is there without its key, the key without its `.pub`, or the two
differ, and says how to mend each. A key that an agent holds, at the socket of `WS_IDENTITY_AGENT`
or Secretive's, must be one that the agent lists, or the stage stops, naming the socket.

## GitHub and Ubuntu's install

gh, from Homebrew where missing, logs in to github.com by its device flow, which shows a code to
enter at `github.com/login/device`. Its scopes, `write:public_key` and `write:ssh_signing_key`, let
it add the Mac's keys, and a login without them is refreshed the same way. The wizard adds the Mac's
key for ws to the account's ssh keys, titled with the Mac's name. `ws client macos apply` then adds
git's signing key ([git](../git/README.md)). With no terminal, the stage stops, naming the command to run.

Ubuntu's installer imports the account's keys for ws's user, `zadykian`. So run the line before
Ubuntu's install, as the wizard adds no key to ws afterwards. A run before the install stops at the
Host key stage, as ws doesn't answer yet. Leave ws's address empty there, and run the wizard again
once Ubuntu is installed. In the installer, turn OpenSSH on, import the keys from GitHub, and leave
password login off.

## ws's host key

The stage reads ws's ed25519 key with `ssh-keyscan`, at the host and port that `ssh -G` gives
for a direct way to `ws` ([ssh's hosts](../README.md#sshs-hosts)). Where no way matches, it stops, naming
[The router](../../../server/ubuntu/network/router.md), or off the LAN what the target needs, as a VM on the Mac started. A new ws
has a lease from the router's pool, not its static address. So where ws gives no key within 10 s,
the stage asks ws's address now, as the router's leases show it. Off the LAN, [The wizard's
targets](../conf/README.md) says where it shows. ssh takes that address for the rest of the run, and
`mac.conf` keeps nothing of it. An empty answer stops the run, as before Ubuntu's install.

The stage shows the key's fingerprint. At ws's console, `ssh-keygen -lf
/etc/ssh/ssh_host_ed25519_key.pub` prints ws's own: answer `yes` where they match, or paste what it
prints. Any other answer stops the run. The key goes in `known_hosts` as `ws`, the name that
`HostKeyAlias` gives each way. From then on, the stage reads nothing from ws. Through `WS_REMOTE`,
the stage warns that a new ws faces the internet until its run sets up the `security` role
([security](../../../server/ubuntu/security/README.md)). With no terminal, `WS_FIRST_ADDRESS` and `WS_HOST_KEY` give the
answers. A linked `known_hosts` stops the stage, and a run prints the key's line for its target.

## Login

The stage is done where ssh logs in to `ws` at a direct way as `WS_USER` by the Mac's key. The
wizard adds no key to ws: GitHub's account is the only way to give ws its keys. Where `WS_USER`
doesn't take the key, the stage stops, at a terminal or not, and names the Mac's `.pub`. Ubuntu's
installer imports the account's keys for its user. A cloud VM gives its default user the provider's
keys instead, which must be the account's. Set `WS_USER` to that user, then run the wizard again.
`--check` says that a run would stop. Both show any ssh error.

Where ws offers another host key than `known_hosts` holds, as once ws is installed again, it stops,
naming `ssh-keygen -R ws`. It does so whatever `StrictHostKeyChecking` the user's `ssh_config` sets,
and its logins add nothing to `known_hosts`. They run no `LocalCommand` either, such as the line
that starts cld's watcher ([The watcher](../ghostty/README.md#the-watcher)).

A run shares one ssh connection for its direct calls, and one for the Tunnel stage, so Secretive's
key asks Touch ID once a run. Its end closes them, or 30 idle minutes after a `SIGKILL`.

## Sudo

The stage reads how the remote scripts reach root on ws. A root login needs nothing more. For
another `WS_USER`, it tries `sudo -n true`. Where sudo wants a password, the stage asks for it,
hidden, once a run, and has sudo check it, with nothing behind it on stdin, up to three times. With
no terminal, it stops. A user outside sudoers stops it too, with sudo's words. `--check` asks at a
terminal, as the stages after it read ws through sudo. With no terminal, it says that a run would
ask, and those stages wait. The password stays in the wizard's memory, and goes as the first line
of `sudo -k -S -p ''`'s stdin, ahead of the script. `-k` has sudo read it each time. It goes in no
argument, message or file.

## ws on ws

The stage is done where the remote script's `status` prints this ws's version as `ws`. Else the
script's `install TAG URL` runs the `install.sh` of `WS_RELEASES_URL` or [GitHub's
releases](https://github.com/zadykian/ws/releases), which checks the binary against `ws.sha256`. The
address starts with `https://`, or `http://127.0.0.1:` in CI. A build of a clone is no release: put
a Linux build of the tree at `/usr/local/bin/ws`, then run again. Where ws holds a newer release, as
after a `ws update` there, the stage stops and names `--update`. With it, ws on the Mac comes first,
before the lock: where `/latest` names another release, its `install.sh` runs, then the command
again. A ws that is no release stays, with a warning. The script's `state` prints `ws server ubuntu
state --json`, with a run's `HOME` and locale. The report comes back over ssh, so it reaches no
journal. The wizard takes it from its own version's ws alone: another's names other steps.

## ws's run

The stage sends `roles/server/ubuntu/session/wizard-remote.sh`, the wizard's remote script, to root
on ws over ssh at a direct way, through [sudo](#sudo) where `WS_USER` isn't root. It runs `ws server
ubuntu apply --json` in a unit of its own, `ws-wizard-STAMP`, through `systemd-run`, with root's
`HOME` and a UTF-8 locale. That ws runs its roles ([The `playbook`
job](../../../../.github/workflows/README.md#the-playbook-job)). The wizard prints the unit's log
until it ends, past the [event lines](../../../../README.md#events), and stops where the run fails. A first run
takes 20 to 40 minutes. The unit goes on where ssh drops, as when the Mac sleeps, and the next run
of the wizard follows it to its end, and replays the log. A unit of part of ws stops a run of other
steps, which it would leave unrun, until it ends. Where `WS_USER` isn't the server's default, the
stage first names it as `ws_user` in `/etc/ws/host_vars.yml` on ws, with the files script, so the
roles set up that user. Where the file names another, it changes the line.

Each run writes its unit and status in `/var/lib/ws-wizard/last` on ws. A full run, as the stage's,
with no argument, removes `/var/lib/ws-wizard/run` as it starts. As it ends, it writes these words:
ws's version, or `-` where it has none, and the status there. Then `off` where the values turned the
outbound table off, else `on`, and the SHA-256 of the secrets file and of `host_vars.yml`, each or
`-`. The stage is done where that status is 0, the version is this ws's, both sums are those of the
files now, and no unit runs. So a second run runs no apply, and one after a run cut short starts
again, as does one after either file changed: by a wizard, Configure or by hand. A unit that began
before such a change, which the run followed, has a run of its own follow. With `--update`, the
stage runs `ws server ubuntu apply` once more with the ws installed there.

Where ws has its secrets file, `/etc/ws/secrets.age`, and the identity's credential, which `ws
server ubuntu secrets init` makes ([ws's file](../../../server/ubuntu/secrets/README.md#wss-file)), the unit runs `--secrets`
on that file. It loads the credential by `-p LoadCredentialEncrypted=ws-secrets` and `-p Type=exec`,
whose start returns once systemd has decrypted it ([The secrets file](../../../server/ubuntu/shell/README.md#the-secrets-file)).
The remote script's `status` prints the file's SHA-256 as a run would stamp it, or `-` without
either. The stage sends nothing. Without them the run has no secrets. Then the roles leave the VPN
as they find it ([The profile](../../../server/ubuntu/ikeward/README.md#the-profile)).

On a cloud VM, the outbound table drops the metadata service ([Outbound](../../../server/ubuntu/security/README.md#outbound)). So
with `WS_TARGET=cloud`, the stage names `security_outbound: false` in ws's `/etc/ws/host_vars.yml`
before the run, which reads its values there. It reads the line as YAML does, with blanks or a
carriage return after the value too. The stage is done there once a run that began with the line
passed, and `host_vars.yml` is as that run read it.

## Signing key

The stage is done where `WS_USER`'s `~/.gitconfig` on ws names no signing key file, as
`shell_git_signing: false` leaves it ([Signing](../../../server/ubuntu/shell/README.md#signing)), or where the Mac's gh lists ws's
public signing key. ws's `status` prints that key and its hostname, read as `WS_USER`, who can write
those files, so that root never reads through them. Else it runs `gh ssh-key add --type signing
--title HOSTNAME` on the Mac, from a file in the lock. The Mac's gh login has the
scope, and ws's token has no account permission, so ws adds nothing. `--check` says it would.
The closing list omits ws's gh login: `ws server ubuntu apply` logs it in by the secrets file's
`token.github` ([gh](../../../server/ubuntu/devtools/README.md#gh)).

## claude on ws

The stage is done where `WS_USER`'s claude on ws has a login, as `claude auth status` says, run as
that user. Else the wizard runs `claude auth login` as `WS_USER` on ws, in an `ssh -t` session,
through a login shell, whose `PATH` has `~/.local/bin`, where claude is ([claude](../../../server/ubuntu/claude/README.md)). Take
its steps at the terminal, with the address it shows opened in the Mac's browser. With no terminal,
the stage stops, naming `claude auth login`. Where ws lacks claude, it stops.

## The tunnel

The stage writes the Mac's WireGuard key to `wg0`'s `mac.pub` on ws, where that holds another, and
`ws server ubuntu apply --only wireguard` gives `wg0` that peer. That run goes again where `wg0`
lacks the peer, as after one that failed. Both go over a direct way, outside the tunnel, whose
sessions the change ends ([The tunnel](../../../server/ubuntu/wireguard/README.md#the-tunnel)). Then `WS_PUBLIC_KEY` in `server.conf`
takes `wg0`'s key, in place of its line, and `ws client macos apply` runs again, which loads
ws-tunnel ([The Mac's tunnel](../README.md#the-macs-tunnel)): allow it in the Background. Then a handshake
must come within 30 s, and `ssh ws` log in once. Where none came, or the run asked ws's address, the
run goes on, and the closing list says what to do. The stage asks sudo's password only where it
acts.

## The cable

The stage sets up the cable from the Mac's display to a port of ws's, outside the router ([The
direct link](../../../server/ubuntu/network/README.md#the-direct-link)). A cloud VM and a VM on the
Mac skip it. The wizard's files script lists ws's wired ports but the uplink, each with its carrier
and IPv4 addresses. Where none has carrier, as the Mac isn't docked, the stage waits, and the
closing list says to dock the Mac and run the line again. Docked, the stage asks ws's port of the
cable, the one with carrier by default. It names the port in `/etc/ws/host_vars.yml` on ws, which
both roles read, then runs `ws server ubuntu apply --only network,wireguard` there; the next run of
the wizard runs the whole, as `host_vars.yml` changed. Once the port holds `10.77.0.1`, the stage
asks the Mac's service of the cable. A screen asks it and the port as a choice. The default is the
one service with a `169.254` address, which an adapter gives itself where no DHCP server answers.
The answer goes in `server.conf` as `WS_CABLE_SERVICE`, then `ws client macos apply` gives the service
`10.77.0.2` ([The cable](../README.md#the-cable)), and `ping 10.77.0.1` must get an answer.

`none` as either answer writes an empty `WS_CABLE_SERVICE`, so the wizard asks neither again, as
where ws has no cable. From the environment, at a terminal too, `WS_CABLE_PORT` and
`WS_CABLE_SERVICE` give the answers. `server.conf` keeps the port as `WS_CABLE_PORT`, to change there.
Where `host_vars.yml` lacks it, as after ws's install, the stage names it there and runs the roles,
undocked too, but stops on a port ws lacks. `--check` asks nothing, and says what it would do.

Undocked, an empty `WS_CABLE_SERVICE` in the environment ends the wait, for a Mac without the cable.
So does `WS_CABLE_PORT`, for a port that ws keeps down, as no netplan file names it, so the files
script can't read its carrier. The roles bring it up. The closing list names such ports.

## What stays by hand

After a run in which a stage acted, the wizard lists each of these that applies:

- With Secure Boot on, where the kernel refuses the fans' driver, DKMS's key, enrolled at ws's
  console ([Secure Boot](../../../server/ubuntu/fans/README.md#secure-boot)).
- What [The Mac](../README.md) leaves by hand: macOS's grant of the local network to an app.
- ws's lease, made static, where the Host key stage asked ws's address, and the router's forwards
  where `server.conf` has `WS_REMOTE` ([The router](../../../server/ubuntu/network/router.md)). Off the LAN, `server.conf`'s way to ws
  instead, and a cloud VM's firewall for the tunnel ([The wizard's targets](../conf/README.md)).
- ws's outbound table, which `security_outbound: false` in ws's `host_vars.yml` keeps off, where
  `WS_TARGET` isn't `cloud`, as after a move from a cloud VM: that line removed, then `--update`.
- The tunnel's handshake, where the run asked ws's address or none came. A static lease, or off the
  LAN `server.conf`'s way leading to ws, UDP let through to ws, or ws-tunnel allowed in the Background.
  Then the wizard again, which checks it.
- `gh auth setup-git`, where git's config names no credential command of gh's for github.com, as `ws
  client macos apply`'s does with git's name ([git](../git/README.md)). Through it, pushes from the Mac's
  clones take gh's login.
- The cable, where no port of ws's had carrier: the line again, docked, or with `WS_CABLE_PORT` for
  a port that ws keeps down. Without the cable, the line once with an empty `WS_CABLE_SERVICE`.
