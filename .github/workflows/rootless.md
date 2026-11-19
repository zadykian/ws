# The rootless job

A container has no booted systemd, so `playbook` can't run the rootless daemon or the firewall
around it. The job `rootless` runs on GitHub's `ubuntu-26.04` runner, which boots. It sets the
runner up as it would a machine, with a user as Ubuntu's installer leaves it. That user is in the
groups `sudo`, `docker` and `lxd`, and sudo has cached its password. The runner's own Docker is
purged first, as it predates what `--net=host` means on a rootless daemon. Its apt source goes too,
as apt refuses a repository that two sources name with different keys, and mise adds its own. apt's
daily timers lose the stamps of their runs at the image's build, as a new machine's have none
before their first start. With them, systemd takes a run as missed at the boot. Each of the
bootstrap's reloads draws the run's random delay again, and one soon falls in the past: an upgrade
then holds dpkg's lock under the bootstrap. The runner's `/etc/ws/values.yml` sets a 64 MiB swap
file, which base makes and swaps to, for the small disk. A namespace holds a private address that
answers, `10.250.0.2`. The cases are
[rootless-checks.sh](../rootless-checks.sh). Each refusal has a control that holds before
ws runs.

The tree goes in `/opt/ws`, root's alone, with its pin on the tree's build, as for
[`playbook`](README.md#the-playbook-job). GitHub's runner keeps a default ACL on `/opt` that lets
anyone write what lands there, so the job drops it first. Its image lets others write `/opt` and
`/usr/share` too, where mise would refuse
[the plugin](../../roles/server/ubuntu/frame/README.md#the-plugin), so the cases make both root's
alone. As on a new machine, `install.sh` runs the bootstrap on the tree, which installs ws and sets
up the firewall before Docker's ufw rules.
No apt command in it may wait for dpkg's lock, and neither of apt's daily services may run. The
log shows the timers and their stamps before it and after it, and systemd's reloads, so that a
failure shows what started the upgrade. `mise bootstrap plan --detailed-exitcode` must then
exit 0. The bootstrap runs twice, and the plan must exit 0 again.

Then the user must keep `sudo`, leave `docker` and `lxd`, and have no cached password. The system's
socket must refuse it, and its own Docker must run a container that resolves names through
slirp4netns's resolver and reaches the internet. The daemon and its containers must run in
`docker.slice`, and the user's systemd must have `cpuset` and `io`. The user's systemd must start
the daemon by its link, and `docker` reach it by the `rootless` context, as ws makes both without
Docker's setup tool.

The firewall must hold under the rootless daemon:

- A privileged container with the host's root can't flush the ruleset, and can't write the host's
  `/etc` or `/usr`, or read its shadow file.
- A bridged and a host-network container, and the user, can't reach the private address.
- A port published on `0.0.0.0` answers on the host, but not past ufw, and answers again where ufw
  is off.
- Root reaches a system container's address and the user doesn't, though the user reaches a port
  published on loopback.
- A link at `~/.bashrc` is refused, by `ws unit write --dry-run` of the shell's line
  ([Shell](../../roles/server/ubuntu/shell/README.md)).

Then ws's secrets. `ws server ubuntu secrets init` makes the credential, root's 600 in root's 700,
and the file, root's 600. `set` takes a value on standard input, and `keys` names it but never
prints it. The user can't run them, and the user's `systemd-creds decrypt` of the credential
fails: a container's systemd makes none. `ws unit write --dry-run` of `/etc/ws/env`, as root,
decrypts it, reads the file, and prints no value.
`unset` drops the entry, and a second `init` keeps both files ([ws's
file](../../roles/server/secrets/README.md#wss-file)).

The runner has Azure's routes, so only CI proves them. `.github/actionlint.yaml` names the label
`ubuntu-26.04`, as actionlint 1.7.12 doesn't know it yet. It goes once a release lists the label.
