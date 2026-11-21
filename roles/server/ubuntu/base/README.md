# base

The `base` role sets up the ground every machine stands on. It installs the packages every machine
gets, and sets the locale `en_US.UTF-8` and the time zone UTC. It turns on unattended upgrades,
and keeps a swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB. The role holds
mise config and a spec of ws's units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.server.toml`](mise.server.toml) | The packages, and the units of every server. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units: the locale, the time zone, unattended upgrades and apt's timers, the swap file and the swappiness. |
| [`files/`](files) | The files that the units write. |

The values are `WS_BASE_LOCALE`, `WS_BASE_TIMEZONE`, `WS_BASE_SWAP_SIZE_MB` and `WS_BASE_SWAPPINESS`
of the values file ([the README](../../../../README.md#running-it-again)). The packages are mise's
lines, so another list is a change to `mise.server.toml`.

## Locale and time zone

`locale -a` and `/etc/locale.gen` decide whether `locale-gen` runs, and `/etc/locale.conf` holds
`LANG`, where Ubuntu 26.04 keeps the system's locale. The time zone is the link `/etc/localtime`,
made without `timedatectl`, which needs a running systemd. `/etc/timezone`, which tzdata no longer
uses, is kept in step where the machine has it, for readers such as older JVMs and Python's
tzlocal. A locale that glibc lacks, or a zone with no file, fails the plan
([The state kind](../../../../engine/core/kind/state.md)).

## Unattended upgrades

The stock `50unattended-upgrades`, left unchanged, installs Ubuntu's security updates daily, and
[devtools](../devtools/README.md#gh) allows gh's releases. `/etc/apt/apt.conf.d/20auto-upgrades`
holds the settings the package writes when its question is answered yes, under the role's header.
ucf, which keeps the file for the package, takes the header for a change made by hand. So an
upgrade of unattended-upgrades that ships other settings asks which file to keep, and keeps the
role's where no one answers. Where the package's is taken, the next run writes the role's back.
apt's daily timers and unattended-upgrades' check at shutdown are enabled and kept running.

## The swap file

`WS_BASE_SWAP_SIZE_MB` sets another size in MiB, and 0 none. The file replaces the installer's
`/swap.img`. A run stops before it changes the swap where the disk lacks room for the file and
2 GiB more. It stops too where swapping an area off would bring back more than half the memory
available. The values refuse a negative size.

A file of another size is made again, not resized. `fallocate` allocates it under another name, or
zeros do where the file system lacks `fallocate`, and on no other error. Then `mkswap` makes it a
swap area, and a rename gives it its name. So a run cut short leaves no file that `swapon` would
refuse.

A container shares its host's kernel, and so its swap and `vm.swappiness`: there the swap file's
unit and the restart of `systemd-sysctl` are off, and say so. The swappiness's file is written
everywhere.

## The swappiness

The kernel's swappiness is 10, not its default 60, so that under memory pressure it frees the page
cache in preference to swapping memory out. `WS_BASE_SWAPPINESS` sets another, from 0 to 200, in
`/etc/sysctl.d/99-swappiness.conf`, and the values refuse one outside that range. Where the
kernel's value differs from the file's, the run restarts `systemd-sysctl.service`. It applies every
file of `/etc/sysctl.d` as boot does, so a key changed by hand there takes effect too.
