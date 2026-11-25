# What the builders left to the real machine

Each line is a claim that a sub-issue's builder could not prove, as its handoff wrote it, with
where it runs, its command and what passes. The [checks](README.md) hold the issue's eleven. A
line that a script covers names it. The rest are by hand, on the host unless the line says
otherwise. Run `ws ubuntu host setup` twice first, with the guests of the checks. `incus` is run
as root on the host. A cloud VM is a guest in the self-contained mode, out of any host.

## A cloud VM

1. A first run of setup leaves `iptables -S FORWARD` at accept with no reboot, and wg0 reaches
   `ikeward0`. Run `ws ubuntu guest setup`, then `iptables -S FORWARD | head -n 1`, which prints
   `-P FORWARD ACCEPT`, and from a peer of wg0 `ping` an address of the VPN's network.
2. With the provider's firewall open for SSH and the tunnel's UDP port alone, a peer reaches the
   VPN, and `nft list ruleset` holds nothing of ws's but fail2ban's table.
3. A full setup on a machine that is no container ran: `ws ubuntu guest setup` ends, and
   `ws ubuntu guest setup --plan` exits 0. `mise bootstrap`'s `units.verified` is its own.
4. CI's `server` job runs the forward-policy case and the ports on loopback on a booted runner.
   Its first run on the landing pull request shows it. [The server job](../../.github/workflows/server.md)

## The host's first boot and its firewall

5. CI's `host` job has never run. Its first run must show the setup twice with plan 0 after each.
   It must show fans off by their fact, ufw with SSH and Docker's rules, sshd taking keys alone,
   and a container resolving a name. It proves `otlp-checks.sh` on alpine and ufw's rule line for
   4317. It proves plan 0 after the socket's units too. [The host job](../../.github/workflows/README.md#the-host-job)
6. The first boot after the installer reaches `ws ubuntu host setup` with no `values.yml` error:
   `ws ubuntu host values check` exits 0, and `cloud-init status --long` ends `done`.
7. With ufw and Docker on the host, a published port on another address is open on every
   network, as Docker's rules come first. Check that no container of ws's publishes one:
   `docker ps --format '{{.Ports}}'` lists none beyond loopback, and `host.sh 8` finds no
   listener on every address but SSH, the DHCP client and the tunnels' UDP ports.
8. The container proof was not run: `units.verified` is made by the bootstrap and a real run.
   `host.sh 10` after two setups shows it.
9. On a real host, a container on `docker0` and one on a compose network (`br-+`) both reach
   `BRIDGE:4317`. `guests.sh 9` covers a guest's. On the host, run
   `docker run --rm -i alpine:3.24.3 ...` as `.github/otlp-checks.sh IMAGE` does, once on the
   default bridge and once with `docker network create n` and `--network n`. SigNoz shows the data
   under the host's `host.name`: in its UI, or by the query of `guests.sh` with the host's name.
10. A moved `WS_DOCKER_BRIDGE_IP` restarts `otelcol-docker.socket` after Docker moves `docker0`:
    change the value, run `ws ubuntu host setup`, then `systemctl is-active otelcol-docker.socket`
    and `ss -ltn | grep 4317` show the new address.
11. ufw's rule for 4317, and a published port on another address, on the real host:
    `ufw show added | grep 4317` holds the rule, and `host.sh 8` finds no stray allow.

## Incus's objects

12. `incus network acl list --format json`, `incus network list --format json` and `incus profile
    list --format json` print an array whose elements have `name`, and an empty list prints `[]`.
    `incus profile list --format json | jq 'map(.name)'` must hold `ws`.
13. `incus ... edit NAME` and `create NAME` take YAML on stdin with no flag. `incus profile show
    ws | incus profile edit ws` exits 0 and changes nothing, and so does the same for a network,
    an ACL and the pool. `ws ubuntu host setup --plan` exits 0 afterwards.
14. A YAML edit of a pool or network from the listed keys is accepted, with no key of `used_by`,
    `status` or `managed`. The plan after an edit by hand, such as `incus network set incusbr0
    ipv4.nat false`, shows the change and the run mends it.
15. The bridge lists `type: bridge` and `managed: true`, and `incus network create incusbr0 --type
    bridge` takes the YAML on stdin: `incus network show incusbr0 | grep -E 'type|managed'`.
16. `incus storage create ws zfs` on stdin YAML with `config.source: /dev/...` makes the pool, and
    `incus storage show ws` holds the partition as `volatile.initial_source`, `source` being the
    pool's name. A second run changes nothing.
17. Zabbly's key fingerprint `4EFC 5906 96CB 15B8 7C73 A3AD 82CC 8797 C838 DCFD` matches the key
    URL: `curl -fsS https://pkgs.zabbly.com/key.asc | gpg --show-keys --with-colons | grep ^fpr`
    prints it without the spaces, among its lines.
18. Incus reads root's range from `/etc/subuid` as it first starts, and needs no `incus admin
    init`: on a new host, `incus info | grep -A3 idmap` shows the range `1000000`, size
    `1000000000`.
19. The ZFS module loads on the host's kernel: `modprobe -n -v zfs`, then `zpool list ws`.
20. The first boot's setup fails at the pool where no `WS_INCUS_POOL_SOURCE` is set. Then the
    value goes in `/etc/ws/values.yml` and a second `ws ubuntu host setup` makes the pool.
21. `security.port_isolation` and `security.mac_filtering` are accepted on a managed network's
    NIC: `incus config device show NAME | grep -E 'isolation|filtering'`.
22. The ACL's `destination` takes a comma list of CIDRs, and the drops apply to guest-to-host
    traffic, so a guest reaches no port of the host's, SSH included. `guests.sh 3` covers it.
23. Each `linux.kernel_modules` entry loads on the host's kernel: `modprobe -n -v` of each name in
    `incus profile get ws linux.kernel_modules`. A missing one fails a guest's start.
24. A guest's resolver is the DHCP's, never the bridge's address: in a guest, `resolvectl dns` or
    `cat /etc/resolv.conf` names `9.9.9.9` and `1.1.1.1`, or the values' resolvers.
25. `incus launch local:ALIAS NAME` and `incus config edit NAME` take the instance YAML on stdin.
    A live NIC `ipv4.address` edit applies: change a guest's `address` in the list, run the
    setup, and `incus list NAME -c 4` shows it. Ubuntu's root tarball imports with ws's metadata,
    and cloud-init takes the seed: `incus image list`, and in the guest `cloud-init status`.
26. `volume.zfs.use_refquota` and `volume.zfs.reserve_space` reach the guest's root volume, and
    the root `size` is the quota. `zfs get refquota,refreservation ws/containers/NAME` prints the
    disk's size twice. Daily snapshots expire:
    `incus config get NAME snapshots.expiry`.
27. The idmap range with `security.idmap.isolated` holds. So does overlayfs for Docker in a
    nested unprivileged guest. `cat /proc/self/uid_map` in each guest shows a range apart from
    the other guests', and check 1 shows Docker's driver.
28. `nft -j list set bridge ws_sources owned4` prints `elem[].concat = [addr, ifname]`, and an
    empty set prints no `elem`. Run it with guests, and again after `nft flush set bridge ws_sources
    owned4`, which `ws ubuntu host setup` mends.
29. `incus exec` passes the exit status through: `incus exec NAME -- false; echo $?` prints 1. The
    guest's setup works under the bridge ACL, with GitHub and apt reachable, and the guest takes
    the pushed ws: `incus exec NAME -- ws ubuntu guest setup --plan` exits 0.

## The table of sources

30. `nft -c -f` accepts the file: `nft -c -f /usr/local/lib/ws/incus/sources.nft` exits 0, with
    `arp saddr ip . iifname @owned4` and `destroy table`. The unit loads it in a namespace of its
    own with `unshare --net`: `systemctl status incus-sources`.
31. The table and Incus's MAC filter hold together on one port. A guest's ARP and IP from its own
    address pass. A forged one in the subnet or the slots drops. A lease from `0.0.0.0` passes.
    `guests.sh 3` covers the forged ones, and a new guest's lease is check 11.
32. ufw's `user.rules` holds `route:allow any any 0.0.0.0/0 any 0.0.0.0/0 in_incusbr0`, so a
    second run changes nothing: `grep in_incusbr0 /etc/ufw/user.rules`, and `host.sh 10`.
33. No `br_netfilter` loads, which `host.sh 7` shows. A guest's Docker network with ICC off
    fails, as ws does not support it. In a guest, `docker network create -o
    com.docker.network.bridge.enable_icc=false n` fails, or a container on it reaches no other.
34. `nft -s -t list table bridge ws_sources` prints the same in force as in the unshared load.
    A frame tagged VID 0 drops: from a guest, `ip link add link eth0 name v0 type vlan id 0` with
    `ping -I v0` reaches nothing. `systemctl show incus.service -p Requires` names
    `incus-sources.service`.

## The tunnels

35. networkd accepts `IPv4Forwarding=yes` and a `[Route]` on an interface with no address, and
    creates `wgN` with the key file: `ip -br link show wg1` and `wg show wg1`.
36. `ufw insert 1 deny in on wgN` goes before the host's SSH rule, and ufw's route rules by way-in
    `wgN` pass what the bridge ACL allows: `host.sh 8` and `ufw status numbered`.
37. rp_filter on `wgN` and `incusbr0` lets a peer's reply from the guest return. `mac.sh 4` logs
    in. If it hangs, `sysctl -w net.ipv4.conf.all.rp_filter=2` is the loose setting to try.
38. A NIC `ipv4.routes` of `10.99.N.1/32` works with a static `ipv4.address`, and the guest holds
    that address: `incus exec NAME -- ip -4 -o addr show dev eth0`.
39. The bridge's ingress ACL applies to the packets that the host forwards from `wgN` to the
    guest. `ws-ingress` allows only as written. `mac.sh 4` passes. From the Mac, a port of the
    guest that the ACL leaves out, such as 8080 with a listener, does not connect.
40. `ip rule add iif wgN lookup T priority T` and the route `default via GUEST dev incusbr0 onlink
    table T`, T being 1000 plus N, forward with no NAT: `host.sh 7`, and `mac.sh 4` shows
    `10.99.N.2`.
41. `tunnel-forward.service` runs after boot once `incusbr0` exists, and retries on failure:
    after a reboot, `systemctl status tunnel-forward` and `host.sh 7`.
42. The Mac handshakes with `wgN`'s public key, which `guests check` shows: `ws ubuntu host tunnel
    guests check` on the host, then `mac.sh 4`, and a peer's reply returns by the same route.
43. `iifname "ws-NAME" meta mark set` in the bridge's input hook keeps the mark to the ip forward
    hook. `oifname wgN meta mark != M drop` at priority raw passes a guest's replies. Read
    `nft list table ip ws_guest_out`. `mac.sh 4` works, and a guest's forged reply, check 3,
    does not.
44. networkd keeps the rule and route of `ip` after `systemctl restart systemd-networkd`: restart
    it, then `host.sh 7`. `service/tunnel-networkd` restarts it once.
45. On the Mac, `ssh -t HOST /usr/local/bin/ws ubuntu host guests setup NAME --secrets`
    works, after `ssh HOST incus exec NAME -- cat /etc/ssh/ssh_host_ed25519_key.pub` and `ssh HOST
    incus exec NAME -- hostname`, which `ws macos secrets add NAME` runs.
46. A guest's tunnel needs its key at `/etc/wireguard/wgN/mac.pub` on the host, and the host's
    `ws ubuntu host setup` before the guest's check prints its `WS_PUBLIC_KEY`.
47. The router forwards UDP `WS_TUNNEL_PORT` of each guest, and the host's, to the host:
    [the router](../roles/cable/router.md). `mac.sh 4` over the remote way, from the hotspot.

## The link to SigNoz

48. The real proxy device makes the socket owned by uid 800 at 0600 in an idmapped guest:
    `guests.sh 9` checks the owner and mode.
49. forkproxy runs as 65534 on the host and still reaches `127.0.0.1:14317`:
    `ps -o user= -p "$(pgrep -f 'forkproxy.*NAME' | head -n 1)"` prints `nobody`, and check 9 passes.
50. Incus makes `/var/lib/ws/otlp` from the image's template file, so a guest's first start does
    not fail the device: launch a new guest and `incus list` shows it `Running`.
51. A collector's `unix://` exporter works over the real device, and SigNoz receives a guest's
    data: check 9.
52. A profile whose device lost `security.uid`, `security.gid` or `mode` is edited back by the next
    run: `incus profile device unset ws otlp mode`, then `ws ubuntu host setup` restores it. A
    missing directory fails a guest's start: remove `/var/lib/ws/otlp` from a stopped guest's
    disk and start it, and `incus start` fails on the device.

## The GPU and ikeward's modules

53. `incus config edit` accepts a `gpu` device `{type: gpu, vendorid: "10de"}`, and a second guest
    holds it too: `incus config device show NAME | grep -A2 gpu`, `guests.sh 2`.
54. `/dev/nvidia0` and `/dev/nvidia-uvm` exist after the host's boot with Ubuntu's packages, or
    the wait times out and a guest starts without them: `ls /dev/nvidia*` on a cold boot, and in a
    guest. `nvidia-modprobe -u` creates the uvm node.
55. A container's `nvidia-smi` works with the userspace package alone, whose compute library
    installs at the same version: `guests.sh 2`, and `apt-cache policy` of its
    `nvidia-*-server` package after the host's driver moves.
56. `/sys/module/nvidia/version` reads in a container as the host's, and `apt-get install`
    of that package at `=V-*` takes a glob. A pin at 1001 does not make a plain `apt-get
    upgrade` downgrade it wrongly: `apt-get -s upgrade` in the guest lists no nvidia package.
57. Each of `xfrm_user`, `xfrm_algo`, `esp4`, `xfrm_interface` and `nft_masq` loads on the host's
    kernel, and strongSwan's ESP tunnel works in a guest: `modprobe -n -v` of each, and check 5.
58. `systemctl show incus.service -p Wants,After` names `ws-nvidia-devices.service`, and the unit
    is active with no GPU module loaded. CI's `host` job runs it on a runner with no GPU, where
    the unit is off by `gpu_vendor`.
