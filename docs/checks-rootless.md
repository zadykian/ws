# The rootless job

A container has no booted systemd, so `playbook` can't run the rootless daemon or the firewall
around it. The job `rootless` runs on GitHub's `ubuntu-26.04` runner, which boots. It sets the
runner up as it would a machine, with a user as Ubuntu's installer leaves it. That user is in the
groups `sudo`, `docker` and `lxd`, and sudo has cached its password. The runner's own Docker is
purged first, as it predates what `--net=host` means on a rootless daemon. A namespace holds a
private address that answers, `10.250.0.2`. The cases are
[rootless-checks.sh](../.github/rootless-checks.sh). Each refusal has a control that holds before
ws runs.

`ws server ubuntu apply --only security,docker` runs twice, and the second run must change nothing.
Then the user must keep `sudo`, leave `docker` and `lxd`, and have no cached password. The system's
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
- A link at `~/.bashrc` is refused ([Shell](../roles/server/ubuntu/shell/README.md)).

Then ws's secrets. `ws server ubuntu secrets init` makes the credential, root's 600 in root's 700,
and the file, root's 600. `set` takes a value on standard input, and `keys` names it but never
prints it. The user can't run them. A unit with `LoadCredentialEncrypted=ws-secrets` runs `ws server
ubuntu apply --secrets --check`, and one without it fails: a container's systemd runs neither.
`unset` drops the entry, and a second `init` keeps both files ([ws's
file](../roles/server/ubuntu/secrets/README.md#wss-file)).

Last, `ws server ubuntu wizard`. With no terminal it stops where host_vars lacks a key. Then it runs
`ws server ubuntu apply` in a unit, with the credential, and a second run starts no unit. One after
`ws server ubuntu values set` changes host_vars starts one. The unit's ws is a stand-in that adds
`--only shell` to apply, so that a full run, as its stamp has it, fits the job. The user's claude
is a stand-in too, with a login
([The server's wizard](../roles/server/ubuntu/stages/README.md)).

The runner has Azure's routes, so only CI proves them. `.github/actionlint.yaml` names the label
`ubuntu-26.04`, as actionlint 1.7.12 doesn't know it yet. It goes once a release lists the label.
