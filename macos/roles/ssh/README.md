# ssh on the Mac

`mise -C /opt/ws/macos bootstrap` sets up ssh's hosts on the Mac by [ssh's units](units.toml.tmpl): one host
for each server, through the tunnel ([The Mac](../../README.md)). The facts build the text from
`values.yml` and the servers' files, and check it, before any unit runs
([The facts](../../facts/README.md)). A refusal there stops the whole bootstrap, and changes nothing.

## Secretive

[`mise.macos.toml`](mise.macos.toml) pins Secretive, which holds the Mac's keys, by mise's `macos-app`
package: a version, the URL and SHA-256 of its `Secretive.zip`, and the bundle in it. Each bootstrap
installs it into `/Applications`, after mise checks the zip against the SHA-256. `adopt` takes over
the same build where it stands, as a person may have installed it. It refuses another build
and leaves it alone, so remove that copy once. `macos-app` finds no release: a new one changes
`version` and `sha256`, which GitHub's release and Homebrew's cask list.

## seward

[`mise.macos.toml`](mise.macos.toml) pins seward 0.1.0, the Mac's ssh agent and keys' tool, by mise's
`github:` backend. [`mise.macos.lock`](../../mise/mise.macos.lock) holds its release's URL and SHA-256 for
macOS on arm64, and the bootstrap runs locked, so mise refuses a seward that the lock lacks.
mise installs it for the tree alone, on no shell's `PATH`, so `state.seward` copies it to
`/usr/local/bin/seward`, which `/etc/paths` puts on every shell's. Each copy goes in under a
temporary name first, and one there that differs is replaced. Where mise has no seward, that
unit alone fails. To move seward on, change the pin, lock it again and change `WS_SEWARD_VERSION`
in the service below.

### The agent

`[bootstrap.services.seward]` runs `/usr/local/bin/seward agent` as mise's user service: a
LaunchAgent, `~/Library/LaunchAgents/dev.mise.seward.plist`, loaded in the user's login session.
`requires_tools` starts it after ws's units, which copy seward. The agent answers on
`~/.ssh/seward.sock` and logs to the unified log: `log stream --predicate 'subsystem == "seward"'`
reads it. Setup needs the user logged in at the console, as `launchctl bootstrap gui/UID` loads
the agent into that session alone.

mise restarts a user service where its plist changed or its job stopped, and renders no template
there. So the service's `environment` holds `WS_SEWARD_VERSION`, which a new pin changes: the
plist differs, and the run that sees the new pin restarts the agent. With an agent started by hand
on the socket, the service's own exits 1, and the default `restart` starts another every 10 s:
stop the one by hand first.

### The keys

`state.ssh-auth-key` and `state.ssh-sign-key` run `/usr/local/bin/seward keygen ws-auth` and
`ws-sign` as the user, after `state.seward`. They run in the user's own session and never as root,
as a key that root's session made fails to sign in the user's. Each makes `~/.ssh/NAME` and `NAME.pub`.
`creates` names both files, so a lost `.pub` comes back, as keygen writes it again from the key.
With the `.pub` alone, keygen exits 1 and the unit fails: run `seward remove NAME`.

On a real Mac the keys are in the Secure Enclave, and a Mac with none fails the unit. The fact `.VM`
([The facts](../../facts/README.md)) is true on a virtual machine, which has no Secure Enclave, and
both units add `--store file` there: its keys are files, as CI's runner holds them.

## The keys' commands

`ws macos ssh keygen NAME`, `list` and `remove NAME` hand the call to `/usr/local/bin/seward`, which
owns the keys in `~/.ssh` ([seward](#seward)). Each replaces ws with seward, so its lines, its
terminal and its exit status are seward's. Each refuses root, as the keys and the agent are the
user's. Where seward cannot start, as before setup copies it, the command exits 1 and names setup.
ws checks NAME by [the rule of a key's name](../../../internal/sshkey/sshkey.go) first: a bad name
exits 2 and runs nothing.

- `keygen` makes `~/.ssh/NAME` and `NAME.pub`. `--store enclave|file` goes to seward as given: seward
  owns its values and its default, `enclave`, which keeps the key in the Secure Enclave and fails
  where the Mac has none. `file` keeps it in the file, as on a virtual machine.
- `list` prints a line for each key: `NAME`, `TYPE`, `FINGERPRINT` and `STORE`, tab-separated.
- `remove` asks on the terminal before it deletes the key and its `.pub`. `--yes` skips the
  question, which a run with no terminal must pass.

## The units

- `state.seward` copies seward to `/usr/local/bin/seward`, root:wheel and 755. The check runs as
  the user, so a bootstrap that changes nothing asks for no password.
- `state.ssh-auth-key` and `state.ssh-sign-key` make `~/.ssh/ws-auth` and `ws-sign` by seward
  ([The keys](#the-keys)).
- `services.seward` runs seward's agent as mise's own user service, no unit of ws's
  ([The agent](#the-agent)).
- `file.ssh-config` writes `/etc/ws/ssh_config`, root's and 644, from the facts' text. ssh reads
  the new file alone once more before it goes in.
- `state.ssh-dir` makes `~/.ssh`, the user's and 0700, where missing, as ssh makes it.
- `file.ssh-include` puts `Include /etc/ws/ssh_config` first in `~/.ssh/config`, the user's and 600.

## ssh's hosts

`/etc/ws/ssh_config` holds a host for each server of `servers/`, named as the server is, at its
tunnel's `10.99.N.1` from the first run, whichever network the Mac is on. A server's `10.99.N.1`
is `10.99.0.1` at slot 0. So `ssh NAME` fails until the keys' exchange ends ([The Mac's
tunnel](../../README.md#the-macs-tunnel)), where a direct way serves. It logs in as root with
the key below, and checks the server's host key under its name (`HostKeyAlias`). A
server whose `values.yml` fails to load stops the bootstrap. A session ends after 120 s without
word from the server, far longer than a switch of the tunnel takes.

ssh offers ws one key alone (`IdentitiesOnly`): seward's `~/.ssh/ws-auth.pub`, through its agent,
`IdentityAgent ~/.ssh/seward.sock`. ssh asks the agent for the key that the `.pub` names, so no
private key is read. No value names the key, and the facts read none: `state.ssh-auth-key` makes
it ([The keys](#the-keys)), and the file goes in after that unit. ws picks no other key, as a key
made for another use would go to GitHub unasked. Until the agent runs, `ssh NAME` finds no key.

The Include goes first in `~/.ssh/config`, as ssh keeps the first value it reads for each key. So
neither the file's own settings nor those of other files override ws's. Where the line is missing,
the unit adds it at the top and keeps the old file as `~/.ssh/config.TIME~`. Where the line sits
below other settings, the unit stops and leaves the file alone. The line also counts in quotes, or
as the first of several paths. Where `~/.ssh/config` is a link that lacks the line, as a dotfiles
manager makes it, the unit stops: add the line to the file it points to. A `~/.ssh/config.d/ws`
and its `Include` stay as they are. The bootstrap stops where a `Host` or `Match` of
`~/.ssh/config`'s own names a server, whose settings would come after those of
`/etc/ws/ssh_config`: remove it, as its `IdentityFile` would add a key to ws's.

ssh keeps the first `IgnoreUnknown` it reads too. `/etc/ws/ssh_config` gives one for `UseKeychain`,
which only Apple's ssh knows and a user's file may set. So it carries that, and the patterns of
every `IgnoreUnknown` in `~/.ssh/config`. Run the bootstrap again after a change to those. A
pattern in a file that `~/.ssh/config` includes must be in `~/.ssh/config` too, as ws reads no
other file.

The facts have `ssh -G`, for each server, read the new text alone. Then they read it first,
`~/.ssh/config` next, without its own Include of the old file, and the system's file last. Where ssh
refuses either, or reads a server at another address or user than its block sets, the bootstrap
stops. The system's file goes through an Include there, so its relative Includes resolve under
`~/.ssh`. Where cld has `restore tabs`, each host starts cld's watcher ([The
watcher](../ghostty/README.md#the-watcher)).

Through the forward, ws takes logins from the internet: by key alone, as everywhere, with fail2ban
banning the address of the network the Mac is on after 5 failed ones. Over the tunnel, ws sees
the Mac as `10.99.N.2`. Failed logins over the tunnel ban it, which shuts `ssh ws` out for an
hour. `ssh -o HostName=ADDRESS -o Port=PORT ws` comes from another address, and from there
`fail2ban-client set sshd unbanip 10.99.N.2` lifts the ban.

A guest holds no filter, so the Mac's firewall alone keeps ws from opening a connection to the Mac
over the cable, the LAN or the tunnel ([The firewall](../firewall/README.md)).

The first login asks whether to trust ws's host key: compare its fingerprint with the one
`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` prints on ws's console. On the cable and the LAN,
macOS asks once whether Ghostty, Zed or another app may find devices on the local
network: allow it. Only a direct way needs it. ssh run from Terminal needs no such grant, and
neither does `ws` through the tunnel. The tunnel's utun is no local network, and macOS lets daemons
that run as root reach one anyway (Apple's TN3179).
