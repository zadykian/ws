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
link, then the archive's name, as in `46547c76f522d_goland-2026.2.3`. The role writes
`.expandSucceeded` there last, as Gateway does once a build is unpacked. Gateway then finds the
build, lists it among the installed IDEs, and downloads nothing. Neither the name nor the file is
documented: they are Gateway 2026.2.2's, and a later Gateway may change them.

Gateway keeps each recent project on the build it opened it with. It shows "Project IDE is
deleted" once that build is gone, and "Select Different IDE…" picks the new one, with no download.
The JetBrains Client on the Mac follows the backend's version, and Gateway fetches it as needed.
Settings, plugins and caches are kept per major version, as in `~/.config/JetBrains/Rider2026.2`,
so a 2026.2.x update keeps them.

## Removals

Once the new build is in place, the role removes every other build of that product from the
directory, Gateway's own downloads too; other IDEs' stay. A build a process runs from, such as a
backend open in Gateway, stays, and the run names it; the next run that finds none removes it.
`jetbrains_prune: false` keeps every build.

Rider's backend takes 6.6 GB unpacked and GoLand's 3.5 GB. A run downloads an archive only where
the disk has 4 times its size free, as JetBrains asks, and removes it once unpacked.
`jetbrains_backends` names the backends by JetBrains' product codes, `[RD, GO]`, and
`-e '{"jetbrains_backends": []}'` installs none. A dry run reads the releases and the directory,
and reports each build it would install and each it would remove; it downloads nothing.

## The timer

Between runs of the playbook, `jetbrains-update.timer` runs the role once a week, in the hour after
Monday's midnight, or at boot where the machine was off then. Its service applies the role as the
last run of the playbook left it. Each run copies the role, and a playbook with that run's
variables, into `/usr/local/lib/ws/jetbrains`, so that a change in the clone reaches the timer with
the next run alone. It runs `/usr/bin/ansible-playbook`, from Ubuntu's ansible-core, which the role
installs where bootstrap.sh hasn't. Its output goes to the journal, and so to SigNoz.

`systemctl list-timers jetbrains-update.timer` shows when it runs next,
`journalctl -u jetbrains-update` what it did, and `systemctl start jetbrains-update` runs it now. A
backend open in Gateway as it runs keeps its build until a later run.
