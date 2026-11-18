# The file kind

A `file` unit writes one file from a template or a source, or keeps one line in a file that is its
owner's. The kinds' page, [ws's kinds](README.md#the-kinds), holds the rest.

```toml
[file.NAME]
path = "/abs/path"          # each missing parent is made, root's at 0755
mode = 0o644                # the default
owner = "root"              # the default; another owner's file lies below its home
group = "NAME"              # a file of root's takes this group of /etc/group
template = "name.tmpl"      # or source = "name", copied unchanged
check = ["cmd", "{}"]       # {} is the new file's path; a refusal leaves the live file
check_booted = false        # run check with booted systemd alone
service = "x.service"       # restarted where the file changes
enabled = false             # service starts at boot
reload = false              # reload service, not restart it
user_service = "z.service"  # a unit of the owner's own systemd, restarted where the file changes
secret = false              # keep the content out of the diff and out of a refusal's error
absent = false              # remove the file; takes a path, an owner and ours alone
line = ". /etc/x"           # or a line at the top of the owner's file, which keeps its own text
has = ["cmd"]               # says whether the file, on stdin, holds the line: 0 yes, 1 no
group_of = "USER"           # a file of root's takes this account's primary group
backup = false              # keep the old file beside the new one, as PATH.PID.TIME~
ours = false                # refuse a file there without ws's '# Written by' line at its top
absent_when_empty = false   # remove the file where its template gives white space alone
after = "KIND/NAME"
```

- A checked file is checked as `PATH.ws-check`, and only a good one moves over the live file.
- A restart that a change calls for is noted in `/var/lib/ws/owed` before the change. A run that
  fails before the restart leaves it owed to the next run. A file that its check refused leaves
  the restart owed too, and the service keeps the live file.
- A refusal of a secret file's check names the checker, the path and the exit status, never its
  output or its stderr, which may quote the content.
- `group` is read from `/etc/group` as the unit plans, so a package that mise installs may bring
  it, and a missing group fails the plan. It takes a file of root's, and never sits by `group_of`.
- A file in `/etc/systemd/system` makes systemd read its units again.
- An owner other than root takes the home that the system gives: `getent passwd` on Linux, and
  `dscl` on macOS, which the side names. A path outside it, or a link on the way, is refused, and
  the missing parents are the owner's.
- A file whose `owner` names root takes root's home where its path lies below it, so a link on the
  way is refused there too. An account's file names root where `WS_USER` is root.
- Without booted systemd the restarts are skipped, and each says so.
- A `line` goes at the top of its owner's file below the home, root's too, with a blank line
  after it. A missing file is made with the line alone, at `mode`. The old file is kept as
  `PATH.TIME~` first, and a link at the file or on the way is refused. A `check` runs on the new
  text before that copy, so a refusal leaves the file unchanged.
- Without `has`, a line of the file that equals the line, spaces aside, holds it. A line of more
  than one line needs `has`. Any exit of `has` but 0 and 1 fails the plan with its stderr.
- A `line` takes `owner`, `mode`, `check`, `has`, `after` and `when` alone.
- `group_of` and `ours` take a file of root's. With `ours`, a file there whose first line lacks
  `# Written by github.com/zadykian/ws` fails the plan, a dry run's too, before a write or a
  removal. A file by hand is never lost, and the error names its path alone.
- A `secret` file's template alone may call `secrets`, the side's secrets, as `(secrets).Env`.
  On a machine with no secrets file, as in CI, the unit leaves the file and says so. `exports MAP`
  gives a line `export NAME=VALUE` a name, sorted, each value quoted for the shell. A name that the
  shell does not take fails the plan, and the error names no value.
