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
  error. The spec's text fills as it loads, with `add`, `mul` and `div`. A file's template fills as
  its unit plans, as an earlier unit may make the account or the file that it reads. It takes
  `envvar FILE NAME` and the facts `mem_mib`, `cpus`, `fs_bytes PATH...` and `uid NAME` too.
- **Load and plan.** Each kind checks its fields as the spec loads, so a bad field fails before any
  unit runs. The resources are made as each unit plans.

## The kinds

The kinds `state` and `guard` follow, with `ws unit`, which plans and applies the units for mise.

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
