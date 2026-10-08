# base

The `base` role sets up the ground every machine stands on:

- the packages every machine gets;
- the locale `en_US.UTF-8` and the timezone UTC;
- unattended-upgrades as Ubuntu ships it, which installs Ubuntu's security updates daily;
- a swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB.

## The swap file

`-e base_swap_size_mb=N` sets another size in MiB, and 0 none. The file replaces the installer's
`/swap.img`. A run stops before it changes the swap where the disk lacks room for the file and
2 GiB more. It stops too where swapping an area off would bring back more than half the memory
available.

The kernel's swappiness is 10, not its default 60, so that under memory pressure it frees the page
cache in preference to swapping memory out. `-e base_swappiness=N` sets another, from 0 to 200, in
`/etc/sysctl.d/99-swappiness.conf`.
