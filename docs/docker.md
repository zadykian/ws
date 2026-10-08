# docker

The `docker` role installs Docker Engine with its buildx and compose plugins, from Docker's apt
repository, and writes `/etc/docker/daemon.json`.

## Logs and restarts

`daemon.json` rotates each container's log at 10 MB and keeps 3 files. A container keeps the log
settings Docker created it with. Containers keep running while Docker restarts (`live-restore`).

## The slice

`dockerd` and every container run in `docker.slice`, capped at half the RAM and half the CPUs, with
no swap. `docker_memory_max`, `docker_cpu_quota` and `docker_memory_swap_max` set other limits, in
systemd's syntax (`16G`, `400%`). A running container moves into the slice when it next starts.
`containerd`, which pulls and unpacks images, stays outside the slice's limits.

## Names

Containers resolve names as the host does. The `dns` of `daemon.json` is the default bridge's
address, `172.17.0.1` (`docker_bridge_ip`), where systemd-resolved listens too. ufw lets Docker's
networks reach port 53 there through their bridges, `docker0` and `br-ID`. A running container
takes the new `dns` when it next starts.

## The address pool

Docker takes each new network's subnet, a `/20`, from `172.16.0.0/12` (`docker_address_pool`). The
pool holds the default bridge too, so that one source range covers them all. A network given a
subnet outside it, or a bridge name of its own, gets no answers.

Docker leaves out of the pool only the subnets of the main routing table's on-link routes. So the
role stops, before it writes `daemon.json`, on any other route into the pool, such as one a VPN
keeps in a table of its own. It stops on any route into the default bridge's subnet too. It reads
the routes as they stand during the run: a VPN that is down shows none.

## Published ports

A port published without an address, as in `-p 8080:80` or compose's `"8080:80"`, goes on
`127.0.0.1` (`docker_publish_ip`) rather than on every address. The `ip` of `daemon.json` sets that
for the default bridge, and `default-network-opts` for each network created afterwards. A network
created before keeps publishing on every address until created again, as by
`docker compose down` and `up`. The default bridge takes `ip` when Docker starts with no container
running, as after a reboot. Docker's published ports get past ufw:
[SSH and the firewall](security.md#ssh-and-the-firewall).

## Build cache

BuildKit drops build cache that no build has used for 90 days (`docker_build_cache_keep_days`). It
collects when Docker starts and after each build, so idle cache goes at the next build or restart.
As Docker's own policy does, it also drops the least used cache, down to 10% of the disk. It does
so while the disk has less than 20% free or the cache takes more than 80% of it.
`docker_build_cache_reserved_space`, `docker_build_cache_max_used_space` and
`docker_build_cache_min_free_space` set other sizes, in the format of `daemon.json` (`50GB`). Cache
that an image shares frees no space until the image is removed.

## The docker group

Where `ws_user` isn't root, the role adds it to the `docker` group, so that it runs `docker` without
sudo. The group amounts to root, as a container can mount any of the host's files. The user has it
from its next login.
