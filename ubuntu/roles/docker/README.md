# docker

The `docker` role installs Docker Engine with its buildx and compose plugins, from Docker's apt
repository, and writes `/etc/docker/daemon.json`. The role holds mise config and a spec of ws's
units ([The frame](../frame/README.md)):

| File | What it holds |
| --- | --- |
| [`mise.guest.toml`](mise.guest.toml) | Docker's apt source, the packages, and the units of every server. |
| [`units.toml.tmpl`](units.toml.tmpl) | The units: the route check, the resolver, the slice, Docker's service, the forward policy, and the check of `docker0`. |
| [`files/`](files) | The templates and files that the units write. |

mise takes Docker's apt key only where its primary fingerprint is the one that the source pins,
`9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88`: a changed key fails the run. mise writes the
source as `/etc/apt/sources.list.d/mise-docker.sources`, with the key inline. apt refuses one
repository with two keys, so a machine that holds Docker's source already, as its install guide
writes it with `Signed-By: /etc/apt/keyrings/docker.asc`, fails its first bootstrap at apt. On such
a machine, as a cloud image with Docker, remove each file of `/etc/apt/sources.list.d` but mise's
that names `download.docker.com` before `ws ubuntu guest setup`: `docker.sources` or `docker.list`.

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
address, `172.17.0.1` (`WS_DOCKER_BRIDGE_IP`), where systemd-resolved listens too. A running
container takes the new `dns` when it next starts.

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
running, as after a reboot. A port published on another address, as in `-p 0.0.0.0:8080:80`, is
open to every network the machine is on.

## The forward policy

Docker sets the FORWARD policy to drop where it turns forwarding on itself. It does so on a first
run, as mise installs Docker's package before any unit runs, so it starts with forwarding off. That
cuts `wg0` off `ikeward0` until a reboot, when `99-ikeward.conf` turns forwarding on before Docker
starts ([ikeward](../ikeward/README.md#forwarding)). `daemon.json` sets `ip-forward-no-drop`, so that
no start sets the policy again. That does not undo a policy that the first start set, so the unit
`state/docker-forward-policy` sets it back to accept, once Docker runs, where it finds drop. The
unit is a guest's alone: a host's ufw drops what it routes unless a rule allows it, and the host's
route rules count on that.

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
