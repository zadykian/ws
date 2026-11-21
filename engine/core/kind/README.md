# ws's kinds

A kind is a resource type that ws writes once, in Go, for what mise's own parts lack. Such are a
check before a file replaces the live one, a restart owed across a failed run, a pinned SHA-256.
A unit is one resource of a kind, named `KIND/NAME`. A role names its units in its spec, and holds
no Go. A kind names no role's word: a role's facts come in by its spec and its templates.

A unit is a `step.Step`, built on the packages of `engine/core/resource`. A run of units plans,
applies and plans again, in the units' order.

## The specs

A role's spec is `units.toml.tmpl` in the role's own directory, one level below the root of the specs.
Its templates and sources sit beside it, and a path in the spec is relative to that directory.
Each Go template ends in `.tmpl`, the spec's name and the name of each `template`
([A role's layout](../../../roles/server/ubuntu/frame/README.md#a-roles-layout)).

- **TOML, decoded strictly.** A table is a kind, and a key under it a unit of that kind. A field
  that no kind has, or a value of the wrong type, fails the load, naming its key and line.
- **The names.** A unit's name within its kind takes letters, digits, `.`, `_` and `-`. A name in
  two specs fails the load.
- **The order.** Units run by name, each moved behind the unit that its `after` names, ahead of the
  next by name. An `after` that no spec holds, or a loop, fails the load.
- **Two renders.** Both are Go templates over the side's values, and a key the values lack is an
  error. The spec's text fills as it loads, with `add`, `mul`, `div`,
  `has_prefix TEXT PREFIX`, `json VALUE`, which quotes a value as Go's JSON encoder does, and
  `fail MSG`, which stops the render with MSG. It takes `self` too, the path of the ws that runs the
  units, so that a command of ws runs before the pinned binary is installed, or without booted
  systemd. A unit's template fills as it plans, as an earlier unit may make the account or the file
  that it reads. It takes `envvar FILE NAME` and the facts `mem_mib`, `cpus`, `fs_bytes PATH...`,
  `installed_version NAME`, `kernel_release` and `root_file PATH` too.
  `installed_version` gives dpkg's version of an installed package, else "", and `kernel_release`
  the running kernel's, as `uname -r` prints it. `root_file` gives a file's text, read as root and
  trimmed, else "". A spec names no secret's file with it, as a file's diff would show the secret.
  `envvar` gives NAME's value in the env file FILE, which it reads and never runs. A NAME that the
  side hides is an input of mise's, which a run writes to FILE before any unit and a plan writes
  nowhere. In the run of the side's task, which mise names by `MISE_TASK_NAME`, `envvar` takes it
  from the run's env where that sets it, empty or not. So a plan shows what the run changes, and a
  shell's stale value counts nowhere else. Both renders take `trim_suffix S SUFFIX`, S without
  SUFFIX. A unit's template takes the fact `ssh_pub USER PATH` too: the SSH public key at PATH,
  below USER's home, as `TYPE BASE64`, a link on the way refused, and "" for no file.
- **when.** Any unit takes `when = { FACT = "WANT" }`, and runs where every FACT of the machine
  gives its WANT, and `"!WANT"` where it does not. Elsewhere it plans nothing, and says so in a
  line. An empty WANT, negated or not, holds on no machine, so a value left empty turns the unit
  off, and its line says that the `when` gives no FACT. The facts are read as the unit plans, so
  that one can follow an earlier unit. macOS has no sysfs, and gives none. A FACT that the
  table lacks fails the load. So does a WANT of a form that its FACT never gives, which would turn
  the unit off on every machine. A new fact is one entry of `whenFacts`, in `when.go`.

  | FACT | What the machine gives |
  | --- | --- |
  | `board_vendor` | The board's maker, from `/sys/class/dmi/id/board_vendor` |
  | `container` | `yes` where Docker, Podman or systemd marks the machine a container, which shares its host's kernel |
  | `gpu_vendor` | The PCI vendor of a device of class `0x03`, from `/sys/bus/pci/devices`: `0x` and four hex digits in lower case, such as `0x1234` |
  | `interface` | A network interface of that name, of any state, from `/sys/class/net`: a name the kernel takes, with no `/` |
  | `set` | Any WANT, so that a value of the side's turns the unit on where set and off where empty; `"!WANT"` fails the load |

- **Load and plan.** Each kind checks its fields as the spec loads, so a bad field fails before any
  unit runs. The resources are made as each unit plans.

## The kinds

The kinds follow, and then [`ws unit`](#ws-unit), which plans and applies the units for mise.
macOS takes `file`, `state` with `run` alone, and `launchd`, and Linux every kind but `launchd`:
the others read apt, systemd, sysfs or `/proc`, and on macOS would read in sync having done
nothing. Any other kind fails the load, as `kindsOn` in `kind.go` says.

### file

A file is written whole from a template or a source, or holds one line of ws's in a file that is its
owner's. It may be checked before it goes live, restart a service, hold a secret, or be removed.
Its keys and rules are on [its own page](file.md).

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

### binary

```toml
[binary.NAME]
version = "1.2.3"
url = "https://host/v{version}/name-linux-{arch}"
sha256 = { amd64 = "…", arm64 = "…" }   # by dpkg's architecture, or all = "…" for one file
arch_names = { amd64 = "x86_64" }        # the asset's spelling of an architecture, where it differs
path = "/usr/local/sbin/name"            # root's, at mode = 0o755 where unset
```

A release's program, with no package around it. The download is checked against its SHA-256 before
it makes the file, so a wrong sum leaves none, and a file of another content is made again.
ikeward's binary is one.

### packages

```toml
[packages.NAME]
template = "names.tmpl"     # the names, at white space; NAME=VERSION pins one, * as a glob
simulate = false            # check that apt finds each name, and install none
```

- mise's own apt takes a fixed list. This kind's list follows the machine, as its template fills
  when the unit plans, with the facts above: `{{ with installed_version "x" }}y={{ . }}{{ end }}`.
- apt-get updates its lists, then installs what dpkg lacks. Recommends follow apt's default.
- A word that is no package's name, such as an option or a path, fails the plan. So does a word
  that ends in a dash, which apt-get reads as a removal. A template that gives no name plans
  nothing.
- `simulate` runs `apt-get install -s` of the names as the unit plans, and fails with apt's error
  where one is missing. It changes nothing, so it tries a role's names on a machine that installs
  none of them. It reads apt's lists as an earlier `apt-get update` left them.

### dkms

```toml
[dkms.NAME]
module = "name"             # DKMS's name of the source, as its dkms.conf gives it
version = "1.0"             # DKMS's version of the pin
url = "https://host/name.tar.gz"
sha256 = "…"
strip = 1                   # leading parts of each name dropped as the tarball unpacks
conf = "dkms.conf"          # a source written as the tree's dkms.conf, {version} for version
load = "name"               # the module loaded once built
unload = ["other"]          # loaded modules unloaded first; goes with load
hint = "…"                  # follows the line of a refused load; goes with load
```

- The tarball waits in `/var/cache/ws`, checked against its SHA-256, and unpacks whole as
  `/usr/src/MODULE-VERSION`, so a new version is a new tree. `conf` replaces the tree's own
  `dkms.conf`, so that the version is pinned in one place.
- DKMS adds the version, then builds and installs it for the running kernel, `uname -r`. It builds
  it again for each later kernel by itself. Another version of the module in DKMS goes first, with
  its source. A version that `dkms status` gives in no plain word's form is left alone, with a
  warning. Before `/usr/sbin/dkms` exists, as in a dry run before its package, DKMS holds nothing.
- DKMS's packages and the kernel's headers are the role's own, a `packages` unit named by `after`.
- `load` is loaded by `modprobe` once built, after each module of `unload` that the kernel holds is
  unloaded. Without booted systemd it loads nothing, and says so.
- With Secure Boot on, the kernel refuses the module until DKMS's key is enrolled, by hand at the
  console. The run says so, with `hint`, and goes on. The mark `/run/ws-refused-LOAD` keeps the
  runs until the next reboot from trying again. Any other failure of the load fails the unit.

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
  A part that goes, by `absent` or `absent_when_empty`, is removed after the parts written, so
  that a file never names one already gone.
  Two files of one service in two file units would restart it twice.
- `in_force` and `loads` run as root. A load that fails, or nothing in force, counts as a
  difference. Each part must exist first.
- A service whose parts are all `absent` is gone: disabled and stopped before its files go, while
  its unit file still names its `ExecStop`, and the links its `[Install]` names. It takes no
  `enabled`, takes `run = "stopped"` or `""`, and compares nothing. A link made by hand stays.
- Without booted systemd it plans no unit, compares nothing, and says so.

### state

A state is a fact about the machine that no file or package holds: lingering, a firewall's
rules, a command that runs once. Its keys and rules are on [its own page](state.md).

### guard

```toml
[guard.NAME]                # plans nothing; fails where a later unit would lock the machine out
login_key = "USER"          # with home: USER holds a key that sshd takes
home = "/home/USER"
command = ["/abs/cmd"]      # or: the command, as root, exits 0
```

- `login_key` reads `home/.ssh/authorized_keys` through the home's link guard. It fails where the
  file holds no key, or sshd would pass it over for its owner or mode.
- `login_key` fails on every machine where `home` is not the one `getent passwd` gives USER: the
  file kind writes USER's files below that one.
- A container, which has no login to lose, passes `login_key` but for its home.
  A unit that would close a login waits on the guard by its `after`.
- `command` is a role's own check, a command of ws where Go holds it. It runs as root on every
  machine, containers too. Its exit 0 passes, and says each line of its output once a run. Any
  other fails the plan with the command's error, its standard error and its output.

### compose

```toml
[compose.NAME]
dir = "/abs/dir"            # holds compose.yaml and the files it reads
project = "name"            # lowercase letters, digits, '_' and '-'
```

- The project runs on the system's daemon, as root. Compose's up runs the one-shot services
  again, so it ups only where something is to do.
- After a good up, it keeps the SHA-256 of the files below `dir` in `/var/lib/ws/compose/PROJECT`,
  root's 0600. The sum covers each file's path and content, read as root, and each link's target.
  A file that differs, comes or goes, or a missing sum, recreates every container. A failed up
  keeps the old sum, so the next run recreates again.
- With the sum kept, it ups without a recreation where a service whose `restart` is `always` or
  `unless-stopped` is not running. A one-shot that has ended counts for nothing.
- Without booted systemd it plans nothing, reads nothing, and says so.

### launchd

A family of LaunchDaemons on macOS, which launchd runs as root from boot. It loads each daemon, and
restarts it or sends it SIGHUP where a file it reads changes. Its keys and rules are on
[its own page](launchd.md).

## ws unit

`ws unit` plans and applies the units for mise's plugin `ws`, a package manager plugin. Each unit is
a package, `"ws:KIND/NAME" = "in-sync"` under `[bootstrap.packages]`. It sits at ws's root, and on
the server it runs as root.

mise calls it through [`plugins/ws`](../../../plugins/ws), which holds no check of its own. Each
hook runs one `ws unit read --whole` or `ws unit write`, with every unit the hook gets, and finds
ws by `mise which ws`, the pinned binary. The write hook runs `ws unit verify` first.

| Command | What it does |
| --- | --- |
| `ws unit list [--toml]` | Prints each unit in the order units run, or as a line of `[bootstrap.packages]`. |
| `ws unit read [--whole] UNIT...` | Plans the units and those they wait on, and prints `NAME`, a tab, and a state. |
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
  server's is `/opt/ws/roles/server/ubuntu`, in the checkout that `install.sh` makes at `/opt/ws`.
  The values are the side's file, `/etc/ws/values.yml` on the server.
- **verify.** mise drops the units of a plugin that it refuses, with a warning and exit 0. `verify`
  reads `mise bootstrap packages status --json`, and fails where the `ws` manager is missing or not
  available, or lacks a unit of the specs. With every unit there, it prints one line that never
  changes, for a file whose template runs it.
- **--whole.** The read hook gets every line of mise's plan, and passes `--whole`. Where the specs
  hold a unit that the lines lack, and it has a change, each unit reads `fails: mise's plan lacks
  ws:KIND/NAME`, so the hook fails every plan and run. A lacking unit in sync passes: mise reads
  the lines again before it installs, and leaves out those in sync. Where ws runs outside the
  run directory, each unit fails, in sync too. It calls no mise, as a check inside mise's own read
  would call the hook again.
- **The run directory.** The failures of `verify` and `--whole` name the specs' parent, as
  `run mise in /opt/ws/roles/server`
  ([The check](../../../roles/server/ubuntu/frame/README.md#the-check-against-a-silent-skip)).
- **Exit codes.** 0 done. 1 a failure: a spec or values that fail to load, a plan, an apply, a
  refused check, a SHA-256 or a skipped unit. `read` and `write` as a user other than root fail
  too. So do the server's on a system other than Ubuntu 26.04. 2 a usage error, such as a unit
  that no spec holds. ws's own lines go to stderr.
