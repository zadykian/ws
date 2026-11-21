# The file kind

A `file` unit writes one file from a template or a source, keeps one line in a file that is its
owner's, or makes a link. The kinds' page, [ws's kinds](README.md#the-kinds), holds the rest.

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
secret = false              # keep the content out of the diff and out of a refusal's error
absent = false              # remove the file; takes a path, an owner and ours alone
line = ". /etc/x"           # or a line at the top of the owner's file, which keeps its own text
line_at = "top"             # or "end", where the line goes
has = ["cmd"]               # says whether the file, on stdin, holds the line: 0 yes, 1 no
group_of = "USER"           # a file of root's takes this account's primary group
backup = false              # keep the old file beside the new one, as PATH.PID.TIME~
ours = false                # refuse a file there without ws's '# Written by' line at its top
mark = "#"                  # the comment sign before ws's mark, for ours: "//" for JSON
keep_foreign = false        # with ours, leave a file without the mark, and warn
seed = false                # write the file where nothing is there, and never read it after
absent_when_empty = false   # remove the file where its template gives white space alone
link = "/abs/target"        # or a link to the target, root's, in place of a file
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
- On macOS a file of root's is readable by all, a `mode` with `0o004`, or the load fails. The Mac's
  units read root's paths as the user, so that a run that changes nothing asks no password.
- An owner other than root takes the home that the system gives: `getent passwd` on Linux, and
  `dscl` on macOS, which the side names. A path outside it, or a link on the way, is refused, and
  the missing parents are the owner's.
- A file whose `owner` names root takes root's home where its path lies below it, so a link on the
  way is refused there too. An account's file names root where the account is root's.
- Without booted systemd the restarts are skipped, and each says so.
- A `line` goes at the top of its owner's file below the home, root's too, with a blank line
  after it. With `line_at = "end"` it goes last, after a newline that the file lacks, for a format
  whose last setting wins or that reads an include where it sits. A missing file is made with the
  line alone, at `mode`. The old file is kept as `PATH.TIME~` first, and a link at the file or on
  the way is refused. On macOS ws reads as the user, so a link at the file whose target holds the
  line is in sync. A link whose target lacks the line is refused. A `check` runs on the new
  text before that copy, so a refusal leaves the file unchanged.
- Without `has`, a line of the file that equals the line, spaces aside, holds it. A line of more
  than one line needs `has`. Any exit of `has` but 0 and 1 fails the plan with its stderr.
- A `line` takes `owner`, `mode`, `line_at`, `check`, `has`, `after` and `when` alone.
- A `link` takes `after` and `when` alone, beside its path. Root owns it, and its missing parents
  are made as a file's. A link to another target is pointed again, and a file or a link in its
  place is replaced, but a directory there fails the run.
- `group_of` takes a file of root's. With `ours`, a file there whose first line lacks
  `# Written by github.com/zadykian/ws` fails the plan, a dry run's too, before a write or a
  removal. A file by hand is never lost, and the error names its path alone. `mark` sets the sign
  before those words. An account's file is read below its home through no link.
- With `keep_foreign`, such a file is left, as the user's own, and the plan warns and changes
  nothing. Once the user removes it, the next run writes ws's.
- A `seed` file is written where nothing is at its path, a link included, and left to its owner
  after, as a tool writes it from its UI. It takes no `line`, `absent`, `absent_when_empty`,
  `backup`, `ours` or `check`.
