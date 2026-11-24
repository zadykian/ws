# security

The `security` role sets up an sshd drop-in that lets everyone log in by key only, and fail2ban for
SSH. A guest holds no filter: it sets up no ufw and no table of its own, as below. The role holds
mise config and a spec of ws's units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.guest.toml`](mise.guest.toml) | The packages, fail2ban's jail and its reload, and its service. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units: the lockout guard and sshd's drop-in. |
| [`files/`](files) | The files and the template that mise and the units write. |

## SSH

ws turns SSH's passwords off: everyone logs in by key, and root by key alone, as on a cloud VM.
fail2ban bans an address for an hour after 5 failed logins in 10 minutes, by its ban action of
nftables, which the package `nftables` gives. sshd's own penalties slow one that keeps failing. Both
hold on every network, with no peer or network exempt. Root logs in by the keys that
[the runbook](../../runbook.md) imports from the GitHub account into `/root/.ssh/authorized_keys`;
ws writes none and copies none from another account's file ([A new
machine](../../../README.md#a-new-machine)). The drop-in limits no user, so the installer's
user keeps its keys. Over SSH, keep the session that ran ws open until a
new one logs in. The guard `guard/security-keys` stops the run before passwords go off in two
cases. One is that root has no key to log in with. A key behind a forced command is none, as
cloud-init writes for root. The other is a file of keys that sshd passes over. A container too, as
it may run sshd. It stops it on every machine where `/root` is not root's home in
`getent passwd`. sshd's drop-in lands only after `sshd -G` accepts it, and sshd reloads after a
change, once, in its unit.

## No filter

ufw and a table of ws's own sit on the machine they guard, where the agent runs as root and gets
past them. So a guest holds none, and no other role sets one up:

- A system container is isolated by its host, which filters what it sends and what reaches it.
- A cloud VM has its provider's firewall, where the runbook has you let in SSH, and the tunnel's UDP
  port where the VM ends the tunnel ([the runbook](../../runbook.md#the-steps-by-hand)).
- Docker's published ports go on `127.0.0.1` ([docker](../docker/README.md#published-ports)): reach
  one through an SSH tunnel. A port published on another address, as in `-p 0.0.0.0:8080:80`, is open
  to every network the machine is on, wherever nothing filters it.
- The Mac's firewall stays, the one guard where a server ends its own tunnel
  ([The Mac's firewall](../../../macos/roles/firewall/README.md)).

`TestGuestHoldsNoFilter` fails where a role's spec or mise file names ufw, a firewall or the old
table.

## Links under the home

ws runs as root, and a link in a home could send its reads, `chown`, `chmod` and backups to any
file. The code states that reason for a home that another user owns, as the installer's user
does. root owns `/root`, and the roles' files there take the same guard: the code gives no other
reason. ws reaches a file under a home through the account alone: as a home file
(`Account.File`, `Account.Dir`), a removal (`Account.Absent`) or the guarded read
(`User.ReadHome`). The file kind's units take that way for each file whose owner is the account.
Each refuses a link at the file, or at a directory between the home and it, and names the
link and its target. The check looks as the unit plans, and a home file or a removal looks again as
it applies. A read of a home file opens each directory below the home, and the file, following no
link. It reads a regular file of 1 MiB at most, with its owner and mode from that open. `rm` and
`git` stay outside the bound.
