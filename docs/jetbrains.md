# jetbrains

The `jetbrains` role installs the backends of Rider and GoLand, at their latest releases, where
JetBrains Gateway looks for them. Gateway then finds them ready rather than downloading its own. It
installs `libicu78` too, without which Rider's hangs. Each run installs a new release and removes
the builds it replaces, and a weekly timer does the same between runs.

## Installs

The Mac opens Rider and GoLand on the server through JetBrains Gateway, which runs each IDE's
backend here. Left to itself, Gateway downloads a backend on the first connect, 2.4 GB for Rider
and 1.2 GB for GoLand, and never updates it. So each run of the role asks JetBrains' list of
releases, which Gateway reads too, for the latest release of each. It installs that release where
the build is missing, checked against the SHA-256 JetBrains publishes for it.

A build goes where Gateway would put its own download, in `~/.cache/JetBrains/RemoteDev/dist`. Its
directory is named as Gateway names it: the first 13 hex digits of the SHA-256 of the download's
link, then the archive's name, as in `46547c76f522d_goland-2026.2.3`. The role unpacks a build
under that name with `.part` added, and writes `.expandSucceeded` there, as Gateway does once a
build is unpacked. Then it renames the directory. Gateway finds the build whole, lists it among the
installed IDEs, and downloads nothing. Neither the name nor the file is documented: they are
Gateway 2026.2.2's, and a later Gateway may change them.

A run holds a lock on `~/.cache/JetBrains/RemoteDev/dist`, with `flock`, while it unpacks or
removes a build. So the timer's run and one by hand never unpack one build at once, nor remove the
tree the other unpacks. A build's directory without `.expandSucceeded` goes, and the build is
installed again. Where a process runs from it, as one of Gateway's may, the run stops instead and
names the process.

Gateway keeps each recent project on the build it opened it with. It shows "Project IDE is
deleted" once that build is gone, and "Select Different IDE…" picks the new one, with no download.
The JetBrains Client on the Mac follows the backend's version, and Gateway fetches it as needed.
Settings, plugins and caches are kept per major version, as in `~/.config/JetBrains/Rider2026.2`,
so a 2026.2.x update keeps them.

## Disk space

Rider's backend takes 6.6 GB unpacked and GoLand's 3.5 GB. A run of the playbook downloads an
archive into `~/.cache/ws-jetbrains/playbook`, and the timer's into `timer` beside it. It does so
only where the home's disk has 4 times the archive's size free, as JetBrains asks. With less, the
run fails before the download: free some, or leave that code out of `jetbrains_backends`. So a
first run needs about 12 GB free for both, and an update room for the new build beside the old. A
run removes its directory before its downloads and after each install, with what a download cut
short left there.

## Removals

Once the new build is in place, the role removes every other build of that product from the
directory, Gateway's own downloads too; other IDEs' stay. It takes the backends one at a time,
those in place first, so that each install finds the room the older builds of those before it
took. A build a process runs from, such as a backend open in Gateway, stays, and the run names it;
the next run that finds none removes it. `jetbrains_prune: false` keeps every build.

`jetbrains_backends` names the backends by JetBrains' product codes, `[RD, GO]`, and
`-e '{"jetbrains_backends": []}'` installs none. A dry run reads the releases and the directory,
and reports each build it would install and each it would remove; it downloads nothing.

## The timer

Between runs of the playbook, `jetbrains-update.timer` runs the role once a week, in the hour after
Monday's midnight, or at boot where the machine was off then. Its service applies the role as the
last run of the playbook left it. Each run copies the role, and a playbook with that run's
variables, into `/usr/local/lib/ws/jetbrains`, and removes from the copy what the clone has
dropped. So a change in the clone reaches the timer with the next run alone. A run sets the timer
up before the backends, so that a run that fails at one leaves the timer to try again.

The service runs `/usr/bin/ansible-playbook`, from Ubuntu's ansible-core, which the role installs
where bootstrap.sh hasn't. Its output goes to the journal, and so to SigNoz.
`systemctl list-timers jetbrains-update.timer` shows when it runs next,
`journalctl -u jetbrains-update` what it did, and `systemctl start jetbrains-update` runs it now. A
backend open in Gateway as it runs keeps its build until a later run.

A run of the playbook during one of the timer's may still fail, as where the two fill the disk at
once. Where JetBrains publishes a release between the two, each may remove the build the other
installs. A run again installs the latest.
