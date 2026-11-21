# docker

The `docker` role installs Docker Engine with its buildx and compose plugins, from Docker's apt
repository, and writes `/etc/docker/daemon.json`. The role holds mise config and a spec of ws's
units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.server.toml`](mise.server.toml) | Docker's apt source, the packages, and the units of every server. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units: the route check, the resolver, ufw's rules, the slice, Docker's service, and the check of `docker0`. |
| [`files/`](files) | The templates and files that the units write. |

mise takes Docker's apt key only where its primary fingerprint is the one that the source pins,
`9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88`: a changed key fails the run. mise writes the
source as `/etc/apt/sources.list.d/mise-docker.sources`, with the key inline. apt refuses one
repository with two keys, so a machine that holds Docker's source already, as its install guide
writes it with `Signed-By: /etc/apt/keyrings/docker.asc`, fails its first bootstrap at apt. On such
a machine, as a cloud image with Docker, remove each file of `/etc/apt/sources.list.d` but mise's
that names `download.docker.com` before `install.sh`: `docker.sources` or `docker.list`.

## Logs and restarts

`daemon.json` rotates each container's log at 10 MB and keeps 3 files. A container keeps the log
settings Docker created it with. Containers keep running while Docker restarts (`live-restore`).
`daemon.json` and docker.service's drop-in are parts of one unit, so a change to both restarts
Docker once. `dockerd --validate` checks a new `daemon.json`, and a refused one leaves the live file
and the restart owed. `WS_DOCKER_LOG_MAX_SIZE` and `WS_DOCKER_LOG_MAX_FILE` set other sizes.

## The slice

`dockerd` and every container run in `docker.slice`, capped at half the RAM and half the CPUs, with
no swap. `WS_DOCKER_MEMORY_MAX`, `WS_DOCKER_CPU_QUOTA` and `WS_DOCKER_MEMORY_SWAP_MAX` set other limits,
in systemd's syntax (`16G`, `400%`). A running container moves into the slice when it next starts.
`containerd`, which pulls and unpacks images, stays outside the slice's limits.

## Names

Containers resolve names as the host does. The `dns` of `daemon.json` is the default bridge's
address, `172.17.0.1` (`WS_DOCKER_BRIDGE_IP`), where systemd-resolved listens too. ufw lets Docker's
networks reach port 53 there through their bridges, `docker0` and `br-ID`, whatever their subnets.
A network given a bridge name of its own gets no answers. A running container takes the new `dns`
when it next starts. The unit `state/docker-dns` owns every ufw rule with the comment
`DNS from containers`, so it deletes the rules it gave for a former `WS_DOCKER_BRIDGE_IP`.

`docker0` takes a new `WS_DOCKER_BRIDGE_IP` or `WS_DOCKER_BRIDGE_PREFIX_LENGTH` only when Docker starts
with no container running, as after a reboot. Until then lookups to a new address get no answers,
and the run stops after it restarts Docker. Reboot, or stop every container and restart Docker,
then run the bootstrap again. Its plan shows the move. Both run as
[the runbook](../../runbook.md) writes them.

## The address pool

Docker takes each new network's subnet, a `/20`, from `172.16.0.0/12` (`WS_DOCKER_ADDRESS_POOL`). A
network created before the pool moves keeps its subnet until created again, as by
`docker compose down` and `up`.

Docker leaves out of the pool only the subnets of the main routing table's on-link routes. So the
run stops, before it writes `daemon.json`, on any other route into the pool, such as one a VPN
keeps in a table of its own. It stops on any route into the default bridge's subnet too. It reads
the routes as they stand during the run: a VPN that is down shows none.

## Published ports

A port published without an address, as in `-p 8080:80` or compose's `"8080:80"`, goes on
`127.0.0.1` (`WS_DOCKER_PUBLISH_IP`) rather than on every address. The `ip` of `daemon.json` sets that
for the default bridge, and `default-network-opts` for each network created afterwards. A network
created before keeps publishing on every address until created again, as by
`docker compose down` and `up`. The default bridge takes `ip` when Docker starts with no container
running, as after a reboot. Docker's published ports get past ufw:
[SSH and the firewall](../security/README.md#ssh-and-the-firewall).

## Outbound

Unless `WS_SECURITY_OUTBOUND` is false, a container reaches no host of the LAN, the cable or `wg0`,
nor of another private network around the server. The `security` role's table drops what it
starts there, so a TCP connect hangs ([Outbound](../security/README.md#outbound)).

## Build cache

BuildKit drops build cache that no build has used for 90 days (`WS_DOCKER_BUILD_CACHE_KEEP_DAYS`). It
collects when Docker starts and after each build, so idle cache goes at the next build or restart.
As Docker's own policy does, it also drops the least used cache, down to 10% of the disk. It does
so while the disk has less than 20% free or the cache takes more than 80% of it.
`WS_DOCKER_BUILD_CACHE_RESERVED_SPACE`, `WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE` and
`WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE` set other sizes, in the format of `daemon.json`: `50G`, `50GB`
or `50GiB`, but not `50Gi`. Cache that an image shares frees no space until the image is removed.
`dockerd --validate` skips the cache's section, so ws's check of the values refuses a size of
another format, and an age past 99999 days.
