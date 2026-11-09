# base

The `base` role sets up the ground every machine stands on. It runs as Go steps of ws, in
`internal/server/base`, first of the roles:

- the packages every machine gets;
- the locale `en_US.UTF-8` and the timezone UTC;
- unattended-upgrades as Ubuntu ships it, which installs Ubuntu's security updates daily, and gh's
  releases, which [devtools](devtools.md#gh) allows;
- a swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB.

Its steps are Packages, Locale, Timezone, Upgrades, Swap and Swappiness, in that order. Each plans
from reads alone, so that `--check` prints what a run would change. The values are `base_packages`,
`base_locale`, `base_timezone`, `base_swap_size_mb` and `base_swappiness` of the values file ([the README](../README.md#running-it-again)).

A step that needs a booted systemd skips without one, as in CI's container, and says so: the
units of Upgrades, `swapon` and `swapoff` of Swap, and `systemd-sysctl` of Swappiness.

## Locale and timezone

`locale -a` and `/etc/locale.gen` decide whether `locale-gen` runs, and `/etc/locale.conf` holds
`LANG`, where Ubuntu 26.04 keeps the system's locale. The timezone is the link `/etc/localtime`,
made without `timedatectl`, which needs a running systemd. `/etc/timezone`, which tzdata no longer
uses, is kept in step where the machine has it, for readers such as older JVMs and Python's
tzlocal.

## Unattended upgrades

`/etc/apt/apt.conf.d/20auto-upgrades` holds the settings the package writes, under the role's
header. ucf, which keeps the file for the package, takes the header for a change made by hand. So
an upgrade of unattended-upgrades that ships other settings asks which file to keep, and keeps the
role's where no one answers. Where the package's is taken, the next run writes the role's back.

## The swap file

`base_swap_size_mb` sets another size in MiB, and 0 none. The file replaces the installer's
`/swap.img`. The swap's plan is a function of `/proc/swaps`, `/proc/meminfo`, the free space of `/`
from `statfs` and the size, mode and owner of `/swapfile`, with table tests. A run stops before it
changes the swap where the disk lacks room for the file and 2 GiB more. It stops too where swapping
an area off would bring back more than half the memory available. A negative size is refused.

A file of another size is made again, not resized. `fallocate` allocates it under another name, or
zeros do where the file system lacks `fallocate`, and on no other error. Then `mkswap` makes it a
swap area, and a rename gives it its name. So a run cut short leaves no file that `swapon` would
refuse.

The kernel's swappiness is 10, not its default 60, so that under memory pressure it frees the page
cache in preference to swapping memory out. `base_swappiness` sets another, from 0 to 200, in
`/etc/sysctl.d/99-swappiness.conf`, and a value outside that range stops the Swap step too.
