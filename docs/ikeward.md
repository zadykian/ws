# ikeward

The `ikeward` role keeps the server on an IKEv2 VPN with
[ikeward](https://github.com/zadykian/ikeward), from a client profile that hwdsl2's `ikev2setup.sh`
makes. It lets the WireGuard peers it names reach the VPN through the server too. Without a
profile, it does nothing, as in CI's container. The `security` role keeps the server and its
containers off the LAN, but leaves `ikeward0` open to them ([Outbound](security.md#outbound)).

## The profile

ikeward reads two of the profile's files, `NAME.p12` and `NAME.sswan`, as
[its page on the profile](https://github.com/zadykian/ikeward/blob/v0.3.0/docs/profile.md) says.
They hold the client's key, so they stay out of this repository, vault-encrypted, on the server. So
does a file of the role's variables, such as `/root/vpn/ikeward.yml`:

```yaml
ikeward_profile: /root/vpn/vpnclient    # vpnclient.p12 and vpnclient.sswan
ikeward_p12_password: PASSWORD          # empty where the .p12 has none
ikeward_dns: 10.20.0.53                 # the remote network's resolver
ikeward_domains: [example.internal]     # the domains whose names go to it alone
ikeward_floors: [10.20.0.0/16]          # the ranges kept off the uplink from boot on
ikeward_forward_from: [10.99.0.2]       # the WireGuard peers that reach the VPN: the Mac
```

Encrypt the three with one vault password, then run the playbook with them:

```sh
ansible-vault encrypt /root/vpn/vpnclient.p12 /root/vpn/vpnclient.sswan /root/vpn/ikeward.yml
/root/repository/ws/bootstrap.sh -e @/root/vpn/ikeward.yml --ask-vault-pass
```

`--ask-vault-pass` reads the password from the terminal, not stdin, which `bootstrap.sh` makes
`/dev/null`. The variables may go in the vault that holds `shell_secret_env` instead, the two files
encrypted with its password ([Secrets](shell.md#secrets)).

`ikeward_profile` is an absolute path, less the extension. The values go to `ikeward install`'s
flags: `--dns`, a `--domain` for each domain and a `--floor` for each range. A profile needs the
resolver and a domain, or the run stops before it changes anything; the floors may be left out.
ikeward's [Set up](https://github.com/zadykian/ikeward/blob/v0.3.0/README.md#set-up) says what each
means. `--probe` stays ikeward's default, the first domain. `--mobileconfig` is left out: ikeward
reads it only where neither of the other files holds the CA, which `ikev2setup.sh` always puts in
the `.p12`.

## A run

- `group_vars/all.yml` pins ikeward's version, and the SHA-256 of its binary for each
  architecture, as the release's `ikeward.sha256` gives it. The run downloads the binary into
  `/usr/local/sbin/ikeward`, where ikeward's `install.sh` would put it, checked against the pin. A
  new pin replaces it.
- It decrypts the two files, under their names, into `/run/ikeward-profile`, which only root may
  enter. It removes them once `install` has run, whether a task failed or not.
- `ikeward install --dry-run` reads them, with the password on its standard input. The run stops
  where ikeward refuses the profile or a value, and shows ikeward's error.
- `ikeward install` runs where the dry run, with `--exit-code`, exits 4: where install would
  install a package, write or remove a file, or take a step. So a new version, a new value or a
  file of ikeward's edited by hand has it run again. So does an install cut short, or a unit of
  ikeward's found stopped.
- The run then waits about 2 minutes for `ikeward status` to pass, and fails otherwise. A later
  role may need the VPN, as for a clone from the remote network. While the VPN's server is down,
  `--skip-tags ikeward` leaves the role out.

The password goes into no output: the tasks that run `install` hide their arguments and output.
Where `install` fails, the run shows ikeward's error, which holds no key and no password.
`ikeward status` and `journalctl -u strongswan.service` show the rest.

`install`, the wait, and the forwarding's ufw rules and setting in force need a booted machine, so
they carry the tag `systemd`. In a container, with `--skip-tags systemd`, the role downloads
ikeward, writes the forwarding's file and runs the dry run, and no more.

## Forwarding

`ikeward_forward_from` names the WireGuard peers that reach the VPN through the server, each by its
address in the tunnel, as `wireguard_peers` gives it. ikeward's NAT rule gives what they send the
address the VPN assigned. For them, the role:

- writes `/etc/sysctl.d/99-ikeward.conf`, `net.ipv4.ip_forward = 1`, and puts it in force. Docker
  turns forwarding on as it starts, but the file does so at boot without Docker;
- adds `ufw route allow in on wg0 out on ikeward0 from ADDRESS` for each peer, with the comment
  `ikeward`. Each names `wg0` too, as ikeward's NAT rule takes whatever leaves by `ikeward0`,
  whatever its source.

A run removes the rule of a peer no longer listed, and the file where none is. A rule made by hand,
without the comment, stays, unless its peer is listed: the run then takes it as the role's.

ufw's default rules forward pings from any interface. So with forwarding on, any peer's pings reach
the VPN, as ikeward's page on
[NAT](https://github.com/zadykian/ikeward/blob/v0.3.0/docs/nat.md#networks-the-server-forwards)
says. The Mac sends the VPN's networks and domains into the tunnel with `WS_TUNNEL_NETWORKS` and
`WS_RESOLVERS` in [mac.conf](mac.md#macconf), or with a private repository's script
([Routes and resolvers of another repository](mac.md#routes-and-resolvers-of-another-repository)).

## Dry runs

With `--check`, the role decrypts the files and runs `ikeward install --dry-run` all the same, then
removes them. Where `install` would run, it shows the dry run's lines: what the profile gives, and
each file as a diff, but for those only root may read. Where the binary would be downloaded, it
says so alone, as it has no binary to ask. It skips the wait for `ikeward status`.

## By hand

The role never removes the VPN. Without `ikeward_profile`, it leaves the VPN, the file and ufw's
rules as they are. ikeward's
[Stop and uninstall](https://github.com/zadykian/ikeward/blob/v0.3.0/README.md#stop-and-uninstall)
says how to stop it or remove it. The VPN's server, and the profile `ikev2setup.sh` makes there,
are set up by hand.
