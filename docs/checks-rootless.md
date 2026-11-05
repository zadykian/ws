# The rootless job

A container has no booted systemd, so `playbook` can't run the rootless daemon or the firewall
around it. The job `rootless` runs on GitHub's `ubuntu-26.04` runner, which boots. It sets the
runner up as it would a machine, with a user as Ubuntu's installer leaves it. That user is in the
groups `sudo`, `docker` and `lxd`, and sudo has cached its password. The runner's own Docker is
purged first, as it predates what `--net=host` means on a rootless daemon. A namespace holds a
private address that answers, `10.250.0.2`. The cases are
[rootless-checks.sh](../.github/rootless-checks.sh). Each refusal has a control that holds before
ws runs.

`ws server apply --only security,docker` runs twice, and the second run must change nothing. Then
the user must keep `sudo`, leave `docker` and `lxd`, and have no cached password. The system's
socket must refuse it, and its own Docker must run a container that resolves names through
slirp4netns's resolver and reaches the internet. The daemon and its containers must run in
`docker.slice`, and the user's systemd must have `io`.

The firewall must hold under the rootless daemon:

- A privileged container with the host's root can't flush the ruleset, and can't write the host's
  `/etc` or `/usr`, or read its shadow file.
- A bridged and a host-network container, and the user, can't reach the private address.
- A port published on `0.0.0.0` answers on the host, but not past ufw, and answers again where ufw
  is off.
- Root reaches a system container's address and the user doesn't, though the user reaches a port
  published on loopback.
- A link at `~/.secrets` is refused ([Docker](docker.md)).

The runner has Azure's routes, so only CI proves them. `.github/actionlint.yaml` names the label
`ubuntu-26.04`, as actionlint 1.7.12 doesn't know it yet. It goes once a release lists the label.
