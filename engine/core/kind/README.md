# ws's kinds

A kind is a resource type that ws writes once, in Go, for what mise's own parts lack. Such are a
check before a file replaces the live one, a restart owed across a failed run, a pinned SHA-256.
A unit is one resource of a kind, named `KIND/NAME`. A role names its units in its spec, and holds
no Go. A kind names no role's word: a role's facts come in by its spec and its templates.

A unit is a `step.Step`, built on the packages of `engine/core/resource`. A run of units plans,
applies and plans again, as a role's steps do.

## The specs

A role's spec is `units.toml` in the role's own directory, one level below the root of the specs.
Its templates and sources sit beside it, and a path in the spec is relative to that directory.

- **TOML, decoded strictly.** A table is a kind, and a key under it a unit of that kind. A field
  that no kind has, or a value of the wrong type, fails the load, and the error names its key and
  line.
- **The names.** A unit's name within its kind takes letters, digits, `.`, `_` and `-`. A name in
  two specs fails the load.
- **The order.** Units run by name, each moved behind the unit that its `after` names, ahead of the
  next by name. An `after` that no spec holds, or a loop, fails the load.
- **Two renders.** Both are Go templates over the side's values, and a key the values lack is an
  error. The spec's text fills as it loads, with `add`, `mul` and `div`. A unit's template fills as
  it plans, as an earlier unit may make the account or the file that it reads. It takes
  `envvar FILE NAME` and the facts `mem_mib`, `cpus`, `fs_bytes PATH...`, `uid NAME` and
  `installed_version NAME` too. The last gives dpkg's version of an installed package, else "".
- **when.** Any unit takes `when = { FACT = "WANT" }`, and runs where every FACT of the machine
  gives its WANT. Elsewhere it plans nothing, and says so in a line. An empty WANT holds on no
  machine, so a value left empty turns the unit off. The facts are read as the unit plans, so that
  one can follow an earlier unit. macOS has no sysfs, and gives neither. A FACT that the table
  lacks fails the load, and a new fact is one entry of `whenFacts`, in `when.go`.

  | FACT | What the machine gives |
  | --- | --- |
  | `board_vendor` | The board's maker, from `/sys/class/dmi/id/board_vendor` |
  | `gpu_vendor` | The PCI vendor of a device of class `0x03`, such as `0x1234`, from `/sys/bus/pci/devices` |

- **Load and plan.** Each kind checks its fields as the spec loads, so a bad field fails before any
  unit runs. The resources are made as each unit plans.

## The kinds

The kinds follow, and then [`ws unit`](#ws-unit), which plans and applies the units for mise.

### file

```toml
[file.NAME]
path = "/abs/path"          # each missing parent is made, root's at 0755
mode = 0o644                # the default
owner = "root"              # the default; another owner's file lies below its home
template = "name.tmpl"      # or source = "name", copied unchanged
check = ["cmd", "{}"]       # {} is the new file's path; a refusal leaves the live file
check_booted = false        # run check with booted systemd alone
service = "x.service"       # restarted where the file changes
enabled = false             # service starts at boot
reload = false              # reload service, not restart it
user_service = "z.service"  # a unit of the owner's own systemd, restarted where the file changes
secret = false              # keep the content out of the diff; takes no check
absent = false              # remove the file; takes a path and an owner alone
after = "KIND/NAME"
```

- A checked file is written to `PATH.ws-check` and checked there. Only a good file moves over the
  live one.
- A restart that a change calls for is noted in `/var/lib/ws/owed` before the change. A run that
  fails before the restart leaves it owed to the next run. A file that its check refused leaves
  the restart owed too, and the service keeps the live file.
- A secret file takes no check, as a refusal's error shows what the checker prints.
- A file in `/etc/systemd/system` makes systemd read its units again.
- An owner other than root takes the home that the system gives: `getent passwd` on Linux, and
  `dscl` on macOS, which the side names. A path outside it, or a link on the way, is refused, and
  the missing parents are the owner's.
- Without booted systemd the restarts are skipped, and each says so.

### deb

```toml
[deb.NAME]
package = "name"
version = "1.2.3"
url = "https://host/{version}/name_{version}_{arch}.deb"
sha256 = { amd64 = "…", arm64 = "…" }  # by dpkg's architecture
stop = "name.service"                   # kept stopped and disabled where the package installs
```

The file waits in `/var/cache/ws`, checked against its SHA-256. Its own name and version are
checked before apt-get installs it. Where the version that dpkg lists is the pin, it plans nothing.

### packages

```toml
[packages.NAME]
template = "names.tmpl"     # the names, at white space; NAME=VERSION pins one, * as a glob
```

- mise's own apt takes a fixed list. This kind's list follows the machine, as its template fills
  when the unit plans, with the facts above: `{{ with installed_version "x" }}y={{ . }}{{ end }}`.
- apt-get updates its lists, then installs what dpkg lacks. Recommends follow apt's default.
- A word that is no package's name, such as an option or a path, fails the plan. So does a word
  that ends in a dash, which apt-get reads as a removal. A template that gives no name plans
  nothing.

### service

```toml
[service.NAME]
unit = "x.service"
enabled = true
run = "running"             # "stopped", "triggered", or "" to leave the unit as found
reload = false
in_force = ["cmd"]          # with loads: where their outputs differ, the unit restarts
loads = ["cmd"]
[[service.NAME.file]]       # a file's own fields, from path to absent
path = "/abs/path"
source = "name"
```

- The parts are written in order before the unit, and a change to any of them gives one restart.
  Two files of one service in two file units would restart it twice.
- `in_force` and `loads` run as root. A load that fails, or nothing in force, counts as a
  difference. Each part must exist first.
- Without booted systemd it plans no unit, compares nothing, and says so.

### state

```toml
[state.NAME]                # exactly one of the keys below
linger = "USER"             # USER's systemd starts at boot
subids = "USER"             # the range 100000-165535 where a file gives USER none of 65536
purged = "package"          # removed with its settings
http = "http://127.0.0.1:PORT/"  # answers 200, waited for 30 times, 2 s apart
ufw = { comment = "ws: NAME", rules = [{ action = "allow", proto = "udp", to_port = "53", dir = "in" }] }
routes = { pool = "CIDR", net = "CIDR", dev = "docker0", hint = "…" }  # refuses an overlap
addr = { iface = "docker0", want = "CIDR", hint = "…" }                # refuses a missing address
left = { user = "USER", group = "GROUP" }
run = { user = "USER", command = ["cmd"], creates = ["/path"] }        # or check = ["cmd"]
```

- A new state is one field of `stateSpec`, of a type that makes its resource.
- `ufw` owns the rules that carry its comment: those in `rules` are added, the others deleted. A
  rule's address or port left out is any.
- `routes` fails where a route of the moment overlaps `pool`, but for an on-link route of the main
  table, or `net` on a device other than `dev`. `addr` fails where `iface` lacks `want`. Each error
  ends with `hint`, as the run cannot mend either.
- `run` runs its command as `user`, with the system's `PATH`: on Linux by runuser, in the user's
  systemd session, and on macOS by sudo. It takes exactly one of `creates`, which runs where a path
  is missing, and `check`, which runs where the check, as `user` too, exits other than 0. A
  procedure that stays a command of ws runs this way.
- Without booted systemd, `linger`, `http`, `ufw` and `addr` skip, and each says so. So does `run`
  on Linux, where it needs the user's session; macOS has no systemd, and runs it.

### guard

```toml
[guard.NAME]                # plans nothing; fails where a later unit would lock the machine out
login_key = "USER"          # with home: USER holds a key that sshd takes
home = "/home/USER"
sudo_right = "USER"         # or: USER has the right to sudo, and a password for it to ask
```

- `login_key` reads `home/.ssh/authorized_keys` through the home's link guard. It fails where the
  file holds no key, or sshd would pass it over for its owner or mode.
- `sudo_right` passes for root, which needs no sudo.
- A container, which has no login to lose, passes each. A unit that would close a login waits on the
  guard by its `after`.

## ws unit

`ws unit` plans and applies the units for mise's plugin `ws`, a package manager plugin. Each unit is
a package, `"ws:KIND/NAME" = "in-sync"` under `[bootstrap.packages]`. It sits at ws's root, and on
the server it runs as root.

| Command | What it does |
| --- | --- |
| `ws unit list [--toml]` | Prints each unit in the order units run, or as a line of `[bootstrap.packages]`. |
| `ws unit read UNIT...` | Plans the units and those they wait on, and prints `NAME`, a tab, and a state. |
| `ws unit write UNIT... [--dry-run]` | Plans, applies and plans again, in the units' order. |
| `ws unit verify` | Fails where mise's plan lacks a unit of the specs. |

- **The states.** `read` prints a line per unit named, in the order given, on stdout. The state is
  `in-sync`; the first change's headline, with `(+N more)` after it for the rest; `after UNIT`,
  where it waits on a unit with changes; or `fails: WHY`. A failed plan stops no other unit's read.
  mise takes the state for the version, so it plans an update for any but `in-sync`. A unit
  whose `when` does not hold reads `in-sync`, and says on stderr why the unit is off.
- **The write.** A unit that a named one waits on is planned and left out. A named unit behind one
  with changes is skipped, and the write exits 1. `--dry-run` applies nothing and prints each
  change, file diffs included. The last line names the units that changed. A unit whose `when`
  does not hold changes nothing: alone, it ends `changed: none`.
- **Where it reads.** The specs sit below `$WS_UNITS`, else below the side's own directory: the
  server's is `/opt/ws/roles/server/ubuntu`, the checkout of `mise bootstrap --from-dir /opt/ws`.
  The values are the side's file, `/etc/ws/values.yml` on the server, and confine the machine's
  writes under `WS_HOME`.
- **verify.** mise drops the units of a plugin that it refuses, with a warning and exit 0. `verify`
  reads `mise bootstrap packages status --json`, and fails where the `ws` manager is missing or not
  available, or lacks a unit of the specs. With every unit there, it prints one line that never
  changes, for a file whose template runs it.
- **Exit codes.** 0 done. 1 a failure: a spec or values that fail to load, a plan, an apply, a
  refused check, a SHA-256 or a skipped unit. `read` and `write` as a user other than root fail
  too. 2 a usage error, such as a unit that no spec holds. ws's own lines go to stderr.
