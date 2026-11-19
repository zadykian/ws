# ikeward

The `ikeward` role keeps the server on an IKEv2 VPN with
[ikeward](https://github.com/zadykian/ikeward), from a client profile that hwdsl2's `ikev2setup.sh`
makes. It lets the WireGuard peers it names reach the VPN through the server too. The role is seven
units of [its spec](units.toml.tmpl), whose lines sit in `mise.server.toml`
([The frame](../frame/README.md)). The check, the install and the wait stay Go, as
`ws server ubuntu ikeward check`, `install` and `wait`, which the units run.

| Unit | What it does |
| --- | --- |
| `guard/ikeward` | Runs `ikeward check`, which fails where the secrets hold no profile. |
| `binary/ikeward` | Keeps `/usr/local/sbin/ikeward` the pinned release's binary. |
| `file/ikeward-sysctl` | Writes `/etc/sysctl.d/99-ikeward.conf` for the peers, and removes it where none is listed. |
| `state/ikeward-sysctl` | Puts the file's value in force where IPv4 forwarding is off. |
| `state/ikeward-forward` | Keeps ufw's route rule for each peer. |
| `state/ikeward-install` | Runs `ikeward install` where `ikeward install --check` finds it due. |
| `state/ikeward-status` | Runs `ikeward wait` where `ikeward status` fails. |

Each unit is off without `WS_IKEWARD_PROFILE`, by `when = { set = ... }`
([when](../../../../engine/core/kind/README.md#the-specs)), and says so in a line. As in CI's container,
the VPN, the file and ufw's rules then stay as they are.
Unless `WS_SECURITY_OUTBOUND` is false, the `security` role keeps what the server and its containers
start off the LAN, but what the network needs, such as DHCP and DNS. It leaves `ikeward0` open to
them ([Outbound](../security/README.md#outbound)).

## The profile

ikeward reads two of the profile's files, `NAME.p12` and `NAME.sswan`, as
[its page on the profile](https://github.com/zadykian/ikeward/blob/v0.3.0/docs/profile.md) says.
They hold the client's key, so they stay out of this repository, age-encrypted, on the server. The
role's variables go in the machine's `values.yml` ([Values](../../../../README.md#running-it-again)):

```yaml
WS_IKEWARD_PROFILE: /root/vpn/vpnclient    # vpnclient.p12 and vpnclient.sswan
WS_IKEWARD_DNS: 10.20.0.53                 # the remote network's resolver
WS_IKEWARD_DOMAINS: [example.internal]     # the domains whose names go to it alone
WS_IKEWARD_FLOORS: [10.20.0.0/16]          # the ranges kept off the uplink from boot on
WS_IKEWARD_FORWARD_FROM: [10.99.0.2]       # the WireGuard peers that reach the VPN: the Mac, at .2 of its slot
```

Put the two files in [ws's secrets file](../../secrets/README.md#the-tables), whole. Add the `.p12`'s
password, left out where it has none ([ws's secrets](../../secrets/README.md#editing)). Run the
bootstrap ([The run](../../secrets/README.md#the-run)). Then remove the plain files, which hold the
client's key:

```sh
sudo ws server ubuntu secrets set file vpn_p12 vpnclient.p12
sudo ws server ubuntu secrets set file vpn_sswan vpnclient.sswan
sudo ws server ubuntu secrets set password vpn_p12
```

`WS_IKEWARD_PROFILE` is an absolute path, less the extension, that names the files `install` writes.
The values go to `ikeward install`'s flags: `--dns`, a `--domain` for each domain and a `--floor`
for each range. A profile needs the resolver and a domain, and each peer is an IPv4 address: ws
refuses other values as it reads them, before anything changes. The floors may be left out.
ikeward's [Set up](https://github.com/zadykian/ikeward/blob/v0.3.0/README.md#set-up) says what each
means. `--probe` stays ikeward's default, the first domain. `--mobileconfig` is left out: ikeward
reads it only where neither of the other files holds the CA, which `ikev2setup.sh` always puts in
the `.p12`.

## A run

- `guard/ikeward` runs `ikeward check`, which fails where `WS_IKEWARD_PROFILE` is set and the
  secrets hold no profile for it. The binary and the forwarding's file wait on it, so the run stops
  before anything changes, and a plan shows the guard's failure.
- `binary/ikeward` pins ikeward's version, and the SHA-256 of its binary for each architecture, as
  the release's `ikeward.sha256` gives it. It downloads the binary for the architecture that dpkg
  names into `/usr/local/sbin/ikeward`, where ikeward's `install.sh` would put it, checked against
  the pin, so a wrong sum leaves no file. A new pin replaces it
  ([binary](../../../../engine/core/kind/README.md#binary)).
- `ikeward install` writes the two files, which the secrets file gives, under their names, into
  `/run/ikeward-profile`, which only root may enter. It removes them once `install` has run, whether
  it failed or not. Where the secrets hold no profile, it fails, as the guard does.
- It runs `ikeward install --dry-run` on them, with the password on its standard input, and stops
  where ikeward refuses the profile or a value, with ikeward's error.
- It runs `ikeward install` where the dry run, with `--exit-code`, exits 4: where install would
  install a package, write or remove a file, or take a step. So a new version, a new value or a
  file of ikeward's edited by hand has it run again. So does an install cut short, or a unit of
  ikeward's found stopped. With `--check`, as the run state's check, it fails there instead.
- `ikeward wait` waits about 2 minutes for `ikeward status` to pass, and fails otherwise. A later
  unit may need the VPN, as for a clone from the remote network.

The password goes on `install`'s standard input, into no argument and no output. Where `install`
fails, the run shows ikeward's error, which holds no key and no password. `ikeward status` and
`journalctl -u strongswan.service` show the rest.

The install, the wait, the forwarding's ufw rules and its value in force need a booted machine.
Where systemd is not running, as in CI's container, each of those units says so and plans nothing.
There the role downloads ikeward and writes the forwarding's file, no more.

## Forwarding

`WS_IKEWARD_FORWARD_FROM` names the WireGuard peers that reach the VPN through the server. It names
each by its address in the tunnel, as `WS_TUNNEL_PEERS` gives it. ikeward's NAT rule gives what
they send the address the VPN assigned. For them, the role:

- writes `/etc/sysctl.d/99-ikeward.conf`, `net.ipv4.ip_forward = 1`, and puts it in force. Docker
  turns forwarding on as it starts, but the file does so at boot without Docker;
- adds `ufw route allow in on wg0 out on ikeward0 from ADDRESS` for each peer, with the comment
  `ikeward`. Each names `wg0` too, as ikeward's NAT rule takes whatever leaves by `ikeward0`,
  whatever its source.

A run removes the rule of a peer no longer listed, and the file where none is.
`state/ikeward-forward` owns every rule of the comment `ikeward`. A rule made by hand, without the
comment, stays, unless its peer is listed: the run then takes it as the role's.

ufw's default rules forward pings from any interface. So with forwarding on, any peer's pings reach
the VPN, as ikeward's page on
[NAT](https://github.com/zadykian/ikeward/blob/v0.3.0/docs/nat.md#networks-the-server-forwards)
says. A server's values.yml sends the VPN's networks and domains into the tunnel, as
`WS_TUNNEL_NETWORKS` and `WS_RESOLVERS`
([the two files](../../../client/values/README.md#the-two-files)). A private repository's script
may do it instead
([Routes and resolvers of another repository](../../../client/README.md#routes-and-resolvers-of-another-repository)).

## The plan

`mise bootstrap plan` runs each unit's check: `ikeward install --check` writes the files and runs
`ikeward install --dry-run` all the same, then removes them. Where the binary is still to download,
the install's check fails, as it has no binary to ask, and the install shows as due.

## By hand

The role never removes the VPN. Without `WS_IKEWARD_PROFILE`, it leaves the VPN, the file and ufw's
rules as they are. ikeward's
[Stop and uninstall](https://github.com/zadykian/ikeward/blob/v0.3.0/README.md#stop-and-uninstall)
says how to stop it or remove it. The VPN's server, and the profile `ikev2setup.sh` makes there,
are set up by hand.
