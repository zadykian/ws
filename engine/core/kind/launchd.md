# The launchd kind

A `launchd` unit keeps a family of LaunchDaemons on macOS, each a program that launchd runs as root
from boot, before any login. mise has no LaunchDaemon. The kinds' page,
[ws's kinds](README.md#the-kinds), holds the rest.

```toml
[launchd.NAME]
prefix = "com.example.name."      # each daemon's label is the prefix and its name
program = "/usr/local/bin/name"   # each daemon runs it; root's, as each directory above it

[[launchd.NAME.daemon]]           # one table a daemon; none removes them all
name = "one"                      # a-z, 0-9 and '-'
arguments = ["/usr/local/bin/name", "run", "one"]  # the program first
log = "/var/log/name.one.log"     # takes its output and errors, where set
restart_on = ["/etc/name/key"]    # files and directories: a change restarts it
reload_on = ["/etc/name/one"]     # a change sends it SIGHUP
```

- **One unit, many daemons.** A spec loops over a side's values to list them, so that no unit's
  name holds a value's. The unit owns each plist of its prefix in `/Library/LaunchDaemons`: one
  that the list lacks is booted out and removed, with its stamp.
- **The plist.** ws writes it whole, root's at 0644, as launchd refuses another owner or mode. It
  runs the arguments at load, keeps them running, and starts them again 10 s after an exit.
- **The plan.** Each daemon plans the first that holds, then keeps its stamp where it differs:
  1. A plist to write, then a bootout where launchd has the daemon, and a bootstrap.
  2. A bootstrap where launchd has it not.
  3. A bootout and a bootstrap where launchd runs other arguments than the plist's, from a plist
     written before a failed run.
  4. `kickstart -k`, where the program or a path of `restart_on` differs from the stamp, or no
     stamp is kept.
  5. `kill SIGHUP`, where a path of `reload_on` differs. A daemon that runs no process refuses
     it, and a kickstart starts it instead.
- **The stamp.** `/var/db/ws/launchd/LABEL`, root's at 0644, holds the SHA-256 of the program
  and of each watched path, a line a path, as `sha256sum` prints. A directory's sum covers the
  files below it, and a missing path counts as one. launchd hashes neither, so the stamp tells
  their change. ws writes it after the call, so a failed restart is done by the next run.
- **Reads as the user.** `launchctl print` needs no root, and the plist and the stamp are
  readable by all, so a run that changes nothing asks no password. A bootstrap, a bootout, a
  kickstart and a signal go through sudo.
- **The program's guard.** The program and each directory above it must be root's and writable by
  root alone. Another user who could write one could replace what launchd runs as root. Each plan
  checks them before it writes a plist, which launchd runs at the next boot. The program may come
  from an earlier unit, so a plan passes it where missing, and each start checks it again.
- **The tries.** launchctl's bootout returns while the daemon may still be on its way out. So a
  bootout waits until print no longer finds it, for 30 s, as launchd gives a job 20 s to stop. A
  bootstrap tries for 10 s while launchctl exits 5.
