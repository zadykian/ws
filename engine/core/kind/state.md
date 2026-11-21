# The state kind

A `state` unit sets one fact of the machine, such as a user's lingering or a firewall's rules. The
kinds' page, [ws's kinds](README.md#the-kinds), holds the rest.

```toml
[state.NAME]                # exactly one of the keys below
linger = "USER"             # USER's systemd starts at boot, and runs: user@UID.service
purged = "package"          # removed with its settings
http = "http://127.0.0.1:PORT/"  # answers 200, waited for 30 times, 2 s apart
ufw = { comment = "ws: NAME", rules = [{ action = "allow", proto = "udp", to_port = "53", dir = "in" }] }
ufw = { comment = "ws: NAME", rules = [{ action = "allow", interface = "IN0", out = "OUT0", from = "CIDR" }] }
routes = { pool = "CIDR", net = "CIDR", dev = "docker0", hint = "…" }  # refuses an overlap
addr = { iface = "docker0", want = "CIDR", hint = "…" }                # refuses a missing address
run = { user = "USER", command = ["cmd"], creates = ["/path"] }        # or check = ["cmd"]
reboot = { module = "name", package = "name", hint = "…" }             # notes a reboot that is due
locale = { name = "en_US.UTF-8", hint = "…" }                          # generated, and LANG
timezone = { name = "Etc/UTC", hint = "…" }                            # /etc/localtime's zone
swap = { size_mib = 1024, hint = "…" }                                 # /swapfile; 0 for none
```

- A new state is one field of `stateSpec`, of a type that makes its resource.
- `ufw` owns the rules of its comment, and with `owns` those whose fields hold its values: those in
  `rules` are added, the others deleted. A rule's address or port left out is any. A rule with `out`
  routes what comes in on `interface` from `from` out by `out`, with `action` and no other field.
- `routes` fails where a route of the moment overlaps `pool`, but for an on-link route of the main
  table, or `net` on a device other than `dev`. `addr` fails where `iface` lacks `want`. Each error
  ends with `hint`, as the run cannot mend either.
- `run` runs its command as `user`, with the system's `PATH`. On Linux `user` is root alone, which
  a spec of the server's names, and the command gets root's `HOME` in ws's env, without the
  caller's `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`. No run reads root's session, and
  dotnet fails on a missing runtime directory. It takes exactly one of `creates`, which runs where
  a path is missing, and `check`, which runs where the check, as `user` too, exits other than 0. A
  procedure that stays a command of ws runs this way.
- A `run` whose program is not ws's own, as `self` names it, gets no variable that the side hides
  (`kind.Env.Hidden`): the server's secret inputs. An installer runs code the run fetched, which
  must not see a token. ws reads its values from the run's env, and keeps them.
- A `run`'s `path`, such as `["/home/USER/.local/bin"]`, puts its directories before the system's
  in the command's and the check's `PATH`. An installer that puts its tool there warns where `PATH`
  lacks it.
- On macOS, ws runs as the user. The `check` runs as that user whatever `user`, so that a read asks
  no password. The command runs as the user bare, as root by sudo, or as another user by root's
  sudo.
- `reboot` plans nothing. Where `package` is installed and `/sys/module/MODULE/version` is not its
  upstream version, without the epoch or the revision, it warns that a reboot is due, with `hint`
  after. It suits a module that loads at boot alone, as a display's driver.
- `locale` generates the locale where `locale -a` lacks it, by its line in `/etc/locale.gen` and
  `locale-gen`, and sets `LANG` in `/etc/locale.conf`. A name that glibc's `SUPPORTED` lacks fails
  the plan, with `hint` after.
- `timezone` links `/etc/localtime` to the zone's file below `/usr/share/zoneinfo`, without
  `timedatectl`, and keeps `/etc/timezone` in step where the machine has it. A name that is no
  path below that directory fails the load, and one that is no zone file the plan, each with `hint`
  after.
- `swap` keeps `/swapfile` of `size_mib` MiB, or the RAM rounded up to a whole GiB where unset, in
  `/etc/fstab` and swapped to, and removes the installer's `/swap.img`. A file of another size is
  made again, under another name until `mkswap` has made it a swap area. The plan reads
  `/proc/swaps`, `/proc/meminfo`, the free space of `/` and `/swapfile`. It fails where the disk
  lacks room for the file and 2 GiB more, with `hint` after. It fails too where swapping an area
  off would bring back more than half the memory available. At 0 it removes the file. Without booted
  systemd it makes the file and skips `swapon` and `swapoff`. The kernel's swap is the host's in a
  container, so a spec turns the unit off there by `when = { container = "!yes" }`.
- Without booted systemd, `linger`, `http`, `ufw`, `addr` and `reboot` skip, and each says so.
