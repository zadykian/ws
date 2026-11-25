# The real machine's checks

The build of the host and its guests rests on Incus 7.0's docs, and CI has no runner of the host's
kind. So the maintainer runs these checks once, on the real machine, in one sitting, once every
other sub-issue is on `feat/incus` and before it lands. A failure of check 1, 3 or 5 reopens the
design. The builders' own open points, each with its check, are in [builders.md](builders.md). The
scripts run nothing in CI, but [checks-test.sh](checks-test.sh) tries their functions in `lint`,
against stand-ins for the tools.

## The scripts

Each prints a line a check, `ok` or `FAIL` with what it saw, and exits 1 where any fails. They
change nothing but where a check says so. Run `host.sh` and `guests.sh` as root.

| Script | Runs | Checks |
| --- | --- | --- |
| [host.sh](host.sh) `[CHECK...]` | On the host. | 2 (the host's part), 6, 7, 8 (the host's part), 10. |
| [guests.sh](guests.sh) `CHECK... [guest=NAME...] [MAC_LAN=ADDRESS]` | On the host. It pushes [guest.sh](guest.sh) into each guest, runs it and removes it. | 1, 2, 3, 5, 9, 11. |
| [guest.sh](guest.sh) `CHECK...` | In a guest, by `guests.sh`. | The guest's part of 1, 2, 3 and 9. |
| [mac.sh](mac.sh) | On the Mac, as its user. | 4, and 5's reach. |
| [lan.sh](lan.sh) `HOST_ADDRESS [GUEST_ADDRESS...]` | On another machine on the LAN. | 8's LAN part. |

Set the host and two guests up first, as the [runbook](../runbook.md) says. Run `ws ubuntu host
setup` twice, then give each guest its setup, `ws ubuntu host guests setup NAME --secrets` for the
one that needs the secrets. Two guests are the least: checks 2, 3 and 11 need a second. Give one
`WS_NVIDIA_GPU: true` and ikeward's values, and each its tunnel slot and peers. Take the
snapshots of each, as check 6 says.

## The checks

**1. Rootful Docker and a compose stack run in an unprivileged, nested guest on ZFS.** This
reopens the design where it fails.

- Where: the host, `guests.sh 1`.
- It checks: the guest is an unprivileged container, and its root is on ZFS. Docker's storage
  driver is overlay2 or overlayfs. A container runs, and a compose stack of two services reaches itself.
- Passes: every line is `ok`. By hand, `docker compose up` of a stack that ws has no part in, such
  as one with a published port, runs in the guest too.

**2. A guest's CUDA runs while the host's `nvidia-smi` feeds ws-fans, and two guests hold the one
GPU.** Again after a driver update and a cold boot.

- Where: the host, `host.sh 2` and `guests.sh 2`.
- It checks: the host's `nvidia-smi` answers, ws-fans runs, and its `nvidia-smi` stream is in its
  cgroup. Each problem that ws-fans logged since it started, such as no NVML or no reading of the
  GPU, is followed by its `back` line. Then `libcuda` initialises in each guest that holds the
  GPU, both at once, with the driver of the host's module.
- Passes: every line is `ok`, `guests.sh 2` with two guests.
- By hand, for the driver update: `apt-get install --only-upgrade` the host's driver, and reboot.
  Then run `apt-cache policy` of the guest's `nvidia-*-server` package, which must name the host's
  new version. Run both scripts again. Do the same after a cold boot: power off, wait, power on.

**3. From a guest, as root, all drop.** This reopens the design where it fails.

- Where: the host, `guests.sh 3 MAC_LAN=ADDRESS`, with `MAC_CABLE=ADDRESS` if the cable's end is
  not `10.77.0.2`.
- It checks, from each guest, that nothing answers at these addresses. They are the router, the
  host's LAN, bridge, cable and tunnel addresses and its resolver, and another guest and its
  slot's address. They are the Mac's LAN, cable and tunnel addresses too. A ping answered, or a
  TCP port that connects or refuses, fails it. Then the internet passes: a ping, a name and
  HTTPS. Then a packet under another guest's address, and a frame under another guest's MAC,
  reach nothing. The guest flushes its neighbour cache first, so that a reply would come back to
  it. The script puts the address and the MAC back.
- Passes: every line is `ok`.

**4. From the Mac, SSH reaches each guest through its tunnel, by each way.**

- Where: the Mac, `mac.sh 4 NAME...`, once at each place: docked at the cable, on the home LAN,
  and on a phone's hotspot, which reaches the router's forward.
- It checks the tunnel's way in use: read the `info` line, cable, LAN or remote. It checks a login
  by key. It checks that sshd in the guest saw the Mac's `10.99.N.2`, not the host's address, and
  that the guest's fail2ban runs its `sshd` jail.
- Passes: every line is `ok` at each place, and the way in the `info` line is the place's.
- By hand, for the ban: from the Mac, fail five logins as a user that is not there
  (`ssh -o PreferredAuthentications=password -o NumberOfPasswordPrompts=1 nobody@NAME`, five times).
  In the guest, `fail2ban-client status sshd` lists `10.99.N.2`. Lift it with
  `fail2ban-client set sshd unbanip 10.99.N.2`.

**5. ikeward in a guest: the VPN comes up, and the Mac reaches a VPN address through that guest's
tunnel.** This reopens the design where it fails.

- Where: the host, `guests.sh 5`, then the Mac, `mac.sh 5 ADDRESS`, ADDRESS being a host of the
  VPN's network.
- It checks: `ikeward status` passes in each guest that runs it, and the Mac's ping of ADDRESS is
  answered, over the guest's tunnel (the `info` line names the Mac's utun).
- By hand, for the peer that `WS_IKEWARD_FORWARD_FROM` leaves out: take the Mac's `10.99.N.2` out
  of that list in the guest's values on the host, run `ws ubuntu host setup`, then `mac.sh 5
  ADDRESS refuse` must pass. Put it back and run the setup again.
- Passes: every line is `ok`.

**6. A snapshot older than the latest restores a wrecked guest.**

- Where: the host, `host.sh 6`.
- It checks: the profile takes a daily snapshot with an expiry, and each guest holds two
  snapshots.
- By hand: take `incus snapshot create NAME first`, then in the guest `touch /root/after-first`,
  and take `second`. Then wreck the guest, such as
  `rm -rf /usr/lib/systemd`, and let it fail. On the host, `incus snapshot restore NAME first`,
  which drops `second` as ZFS restores the latest alone, and start the guest. It boots and
  `/root/after-first` is gone.
- Passes: the guest boots, its tunnel and SSH work, and `ws ubuntu host setup --plan` exits 0.

**7. After Docker's restart on the host and after its reboot, the bridge and the tunnels forward
and check 9 holds. Nothing loads `br_netfilter`.**

- Where: the host. Run `host.sh 7` and `guests.sh 3 9 MAC_LAN=ADDRESS`, then `systemctl restart
  docker` and run them again, then `reboot`, wait for the guests, and run them a third time.
- It checks: no `br_netfilter` module and no `/proc/sys/net/bridge`, IPv4 forwarding on, the two
  tables of sources loaded, the bridge up and `tunnel-forward.service` active. For each guest it
  checks the rule of `iif wgN` and the route of table `T`, T being 1000 plus N, which is the
  rule's priority too.
- Passes: every line is `ok` each time.

**8. From the LAN no guest answers, and the host answers SSH and the tunnels' ports alone, and
nothing to a guest's peer.**

- Where: another machine on the LAN, `lan.sh HOST_ADDRESS GUEST_ADDRESS...`. Then the host,
  `host.sh 8`.
- It checks: SSH answers, a list of other TCP ports does not, and no guest answers a ping or
  SSH, even with a route to it through the host. A guest fails where the script can neither add
  that route by sudo nor find it in place. On the host, ufw denies incoming and holds no
  allow that ws's roles do not open. It denies each tunnel's `wgN` ahead of SSH. Nothing listens
  on every address but SSH, the DHCP client and the tunnels' UDP ports, which `wg show` names.
  Those answer a peer with a valid key alone, which `mac.sh 4` shows by the handshake.
- Passes: every line is `ok`. On a peer: from the Mac, with its tunnel up to a guest, SSH to the
  host's LAN address through that tunnel's route times out.

**9. A Docker container in a guest sends OTLP to the bridge address on 4317, and the host's SigNoz
shows it under the guest's `host.name`. Nothing in that guest listens on `14317`.**

- Where: the host, `guests.sh 9`.
- It checks: a container sends one log record by gRPC to the bridge, as the collector's socket
  takes it. Then SigNoz's ClickHouse holds the record, its body a token, under `host.name` of the
  guest. The guest listens on no `14317`, and the proxy's socket is `otelcol-contrib`'s at 600.
- Passes: every line is `ok`. By hand, SigNoz's UI shows the guest's logs and metrics under its
  name, and the host's under its own.

**10. `ws ubuntu host setup` and a guest's setup each plan nothing on a second run.**

- Where: the host. Run `ws ubuntu host setup` twice, then `host.sh 10`.
- It checks: the host's plan, and each guest's plan through `incus exec`, exit 0.
- Passes: every line is `ok`.

**11. A new guest takes a lease, resolves a name and fetches its secrets. Several run at once,
each with its own host key and name.**

- Where: the host, `guests.sh 11`, after a guest is new: `ws ubuntu host setup`, then `ws ubuntu
  host guests setup NAME --secrets`.
- It checks: each guest holds the address its list gives, is named as its list says and resolves
  a name. It has the secrets' clone. No two guests share an SSH host key. It fails with one guest.
- Passes: every line is `ok`, with two or more guests running.
