# The server job

A container has no booted systemd, so `playbook` can't run the firewall around Docker. The job
`server` runs on GitHub's `ubuntu-26.04` runner, which boots. It sets the runner up as it would a
server that root alone logs in to: root's `authorized_keys` holds a key, no user is made, and
`/etc/ws/values.yml` says `WS_USER: root`. The runner's own Docker is purged first, as ws installs
its own. Its apt source goes too, as apt refuses a repository that two sources name with different
keys, and mise adds its own. So do the plugins that the image put in Docker's directory past dpkg,
which would keep it and warn. apt's daily timers lose the stamps of their runs at the image's
build, as a new machine's have none before their first start. With them, systemd takes a run as
missed at the boot. Each of the bootstrap's reloads draws the run's random delay again, and one
soon falls in the past: an upgrade then holds dpkg's lock under the bootstrap. The runner's
`/etc/ws/values.yml` sets a 64 MiB swap file, which base makes and swaps to, for the small disk. A
namespace holds a private address that answers, `10.250.0.2`. The cases are
[server-checks.sh](../server-checks.sh). Each refusal has a control that holds before ws runs.

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
exit 0. The bootstrap runs twice, and the plan must exit 0 again. Neither run nor plan may print a
warning ([The warnings check](README.md#the-warnings-check)), and none is allowed here, as the
runner boots. A failure prints the system daemon's log.

The firewall must hold under the system daemon. Docker must pull the image, as a run of an image
it lacks warns, and run a container that reaches the internet. Then:

- A bridged and a host-network container, and root, can't reach the private address.
- A port published with no address answers on loopback, and not on the address of another network.
  Docker's published ports get past ufw, so they go on loopback
  ([Published ports](../../roles/server/ubuntu/docker/README.md#published-ports)).
- Root reaches a container's address and a port published on loopback.
- A link at `~/.bashrc` is refused, by `ws unit write --dry-run` of the shell's line
  ([Shell](../../roles/server/ubuntu/shell/README.md)).

Then [the secrets task](../../roles/server/ubuntu/secrets/README.md#the-task), whose cases are
[secrets-checks.sh](../secrets-checks.sh). A bare repository on disk stands in for GitHub's, as
`WS_SECRETS_REPO` in `/opt/ws/mise.local.toml` names it. Its seed holds a shared value and one of
the profile, the hostname's, each encrypted to the runner's host key. Each input that mise lists
and the shared `fnox.toml` lacks is declared empty there, the profile's among them, whose own value
must win. CI's own `/opt/ws/mise.secrets.toml` writes both values to
`/etc/ws-ci-secrets`, root's 600, under the env `secrets` alone. The task runs three times, and the
plan through it exits 0 after each run. A plain plan after the first run keeps the file. A plan
through the task then fetches a new shared value. It exits 2 and keeps the file, and mise's line of
a failed task ends it, outside the warnings check. The second run writes the new value. A file of
the clone's own, a profile that the host key can't open and a missing profile are refused, and the
file keeps its bytes. The last two name `ws client secrets add NAME`. The third run follows them,
and the file still holds both values. No output of the task's holds a value. Only GitHub proves
the fetch from github.com by a deploy key, against the key that ws pins.

Then the files that the roles' [secret inputs](../../roles/server/ubuntu/frame/README.md#secret-inputs)
write. The seed leaves their inputs empty, so none of gh's token, `/etc/ws/gh-token`, git's,
`/etc/ws/git-tokens.toml`, and the shell's `/etc/ws/env` is there. The repository then gives each
a value, masked in the log, the shell's an endpoint of a `.invalid` host. The task's run writes
the three, root's 640 and group, under ws's header. `/etc/ws/env` exports the endpoint in single
quotes, and sh reads it from there. GitHub refuses a dummy token, so root's gh holds it as a login
with it would, and gh's unit compares that with the file's last line. Root's git gets git's token
through [ws's program](../../roles/server/ubuntu/git/README.md#git-credentials). A plan through
the task exits 0, and a run and a plan without the env `secrets` keep the three files, their
bytes, mode and group. A file at gh's path that ws didn't write stops the run, which keeps it, and
a run without the env too. An empty input removes its file.

The runner has Azure's routes, so only CI proves them. `.github/actionlint.yaml` names the label
`ubuntu-26.04`, as actionlint 1.7.12 doesn't know it yet. It goes once a release lists the label.
