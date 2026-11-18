# The state kind

A `state` unit sets one fact of the machine, such as a user's lingering or a firewall's rules. The
kinds' page, [ws's kinds](README.md#the-kinds), holds the rest.

```toml
[state.NAME]                # exactly one of the keys below
linger = "USER"             # USER's systemd starts at boot, and runs: user@UID.service
subids = "USER"             # the range 100000-165535 where a file gives USER none of 65536
purged = "package"          # removed with its settings
http = "http://127.0.0.1:PORT/"  # answers 200, waited for 30 times, 2 s apart
ufw = { comment = "ws: NAME", rules = [{ action = "allow", proto = "udp", to_port = "53", dir = "in" }] }
ufw = { comment = "ws: NAME", rules = [{ action = "allow", interface = "IN0", out = "OUT0", from = "CIDR" }] }
routes = { pool = "CIDR", net = "CIDR", dev = "docker0", hint = "…" }  # refuses an overlap
addr = { iface = "docker0", want = "CIDR", hint = "…" }                # refuses a missing address
left = { user = "USER", group = "GROUP" }
run = { user = "USER", command = ["cmd"], creates = ["/path"] }        # or check = ["cmd"]
reboot = { module = "name", package = "name", hint = "…" }             # notes a reboot that is due
```

- A new state is one field of `stateSpec`, of a type that makes its resource.
- `ufw` owns the rules of its comment, and with `owns` those whose fields hold its values: those in
  `rules` are added, the others deleted. A rule's address or port left out is any. A rule with `out`
  routes what comes in on `interface` from `from` out by `out`, with `action` and no other field.
- `routes` fails where a route of the moment overlaps `pool`, but for an on-link route of the main
  table, or `net` on a device other than `dev`. `addr` fails where `iface` lacks `want`. Each error
  ends with `hint`, as the run cannot mend either.
- `run` runs its command as `user`, with the system's `PATH`: on Linux by runuser, in the user's
  systemd session, and on macOS by sudo. It takes exactly one of `creates`, which runs where a path
  is missing, and `check`, which runs where the check, as `user` too, exits other than 0. A
  procedure that stays a command of ws runs this way.
- `reboot` plans nothing. Where `package` is installed and `/sys/module/MODULE/version` is not its
  upstream version, without the epoch or the revision, it warns that a reboot is due, with `hint`
  after. It suits a module that loads at boot alone, as a display's driver.
- Without booted systemd, `linger`, `http`, `ufw`, `addr` and `reboot` skip, and each says so. So
  does `run` on Linux, where it needs the user's session; macOS has no systemd, and runs it. A
  `run` with `unbooted = true` needs no session, and runs there too, without the session's env.
