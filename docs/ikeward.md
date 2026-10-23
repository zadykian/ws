# ikeward

The `ikeward` role sets the server up for an IKEv2 VPN with
[ikeward](https://github.com/zadykian/ikeward), from a client profile that hwdsl2's `ikev2setup.sh`
makes. Without a profile, it does nothing, as in CI's container.

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
  enter. It removes them once ikeward has read them, whether a task failed or not.
- `ikeward install --dry-run` reads them, with the password on its standard input. The run stops
  where ikeward refuses the profile or a value, and shows ikeward's error.

The password goes into no output: the task that runs the dry run hides its arguments and output.
ikeward's error holds no key and no password.

## Dry runs

With `--check`, the role decrypts the files and runs `ikeward install --dry-run` all the same, then
removes them. Where the binary would be downloaded, it skips both, as it has no binary to ask.

## By hand

The role never removes ikeward. Without `ikeward_profile`, it leaves the binary alone. ikeward's
[Stop and uninstall](https://github.com/zadykian/ikeward/blob/v0.3.0/README.md#stop-and-uninstall)
says how to remove it. The VPN's server, and the profile `ikev2setup.sh` makes there, are set up by
hand.
