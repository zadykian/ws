# The server job

CI sets up one scenario, the closest to a server installed from Ubuntu Server 26.04.1's image. The
job `server` runs on GitHub's `ubuntu-26.04` runner, which boots, as root with the defaults and the
outbound table on. The runner's image may hold a package that ws forgot to install. A server from
Ubuntu's image lacks it, so the trial on that image finds it. The job sets the runner up as it
would a server that root alone logs in to: root's `authorized_keys` holds a key, and no user is
made. The runner's own Docker is purged first, as ws installs
its own. Its apt source goes too, as apt refuses a repository that two sources name with different
keys, and mise adds its own. So do the plugins that the image put in Docker's directory past dpkg,
which would keep it and warn. apt's daily timers lose the stamps of their runs at the image's
build, as a new machine's have none before their first start. With them, systemd takes a run as
missed at the boot. Each of the bootstrap's reloads draws the run's random delay again, and one
soon falls in the past: an upgrade then holds dpkg's lock under the bootstrap. The runner's
`/etc/ws/values.yml` sets a 64 MiB swap file, which base makes and swaps to, for the small disk. A
namespace holds a private address that answers, `10.250.0.2`. The cases are
[server-checks.sh](../server-checks.sh). Each refusal has a control that holds before ws runs.

CI builds ws from the tree, never from a release. The tree goes in `/opt/ws`, root's alone, and
[`.tools/serve-build`](../../.tools/serve-build) serves the build on loopback in the release's
place. The ws that runs is `/usr/local/bin/ws`
([ws at a fixed path](../../roles/server/ubuntu/frame/README.md#ws-at-a-fixed-path)). GitHub's runner keeps a default ACL on `/opt` that lets
anyone write what lands there, so the job drops it first. Its image lets others write `/opt` and
`/usr/share` too, where mise would refuse
[the plugin](../../roles/server/ubuntu/frame/README.md#the-plugin), so the cases make both root's
alone. As on a new machine, the build's `ws server setup --in-place` installs the pinned mise and
runs the bootstrap on the tree in place ([bootstrap.sh](../bootstrap.sh)). The tree's
`locked = true` has mise install each tool as the lock holds it. The bootstrap installs ws and
sets up the firewall before Docker's ufw rules.
No apt command in it may wait for dpkg's lock, and neither of apt's daily services may run. The
log shows the timers and their stamps before it and after it, and systemd's reloads, so that a
failure shows what started the upgrade. Setup's `--plan` must exit 2 before the run, which has
mise, the plugin and the bootstrap to do, and exit 0 after it. Setup runs twice, and the plan must
exit 0 again. Neither run nor plan may print a
warning ([The warnings check](README.md#the-warnings-check)). A failure prints the system
daemon's log.

The run leaves checks that need no container. With the frame's line of `file/ws-values` dropped,
the plan must exit 1 ([server-plan.sh](../server-plan.sh)): the check against a silent skip holds.
A plan from `/opt/ws`, which loads no role, must exit 1 too, and name the run directory. A plan
with a bad value in the values file must exit 1 by `ws server values check` and name its key, and
the file goes back ([The check of the values](../../roles/server/ubuntu/frame/README.md#the-check-of-the-values)). The
values file is root's at 600, and no run changed its bytes ([The values
file](../../roles/server/ubuntu/frame/README.md#the-values-file)). A run with no file is
`values_file_test.go`'s. The Zed wrapper's [checks](../zed-terminal-checks.sh) run on the files the
bootstrap installed, and Fresh's on the config it loaded. Root's empty commit must sign and
verify. With no NVIDIA GPU, `nvidia` checks its names alone, and with no MSI board, `fans` leaves
its fans alone. Without a VPN profile, `ikeward` does nothing.

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
`/etc/ws-ci-secrets`, root's 600, under the env `secrets` alone. The task runs three times, by
`ws server setup --in-place --secrets`, and its `--plan` exits 0 after each run. A plain plan
after the first run keeps the file. A plan through the task then fetches a new shared value. It
exits 2 and keeps the file, and mise's line of a failed task ends it, outside the warnings check.
The second run writes the new value. A run with
`MISE_LOCKED=0`, where mise's setting locked prints `false`, is refused and names
`ws server setup --secrets`. A file of the clone's own, a profile that the host key can't open and
a missing profile are refused too, and the file keeps its bytes. The last two name
`ws client secrets add NAME`. The third run follows them, and the file still holds both values. No
output of the task's holds a value. Only GitHub proves
the fetch from github.com by a deploy key, against the key that ws pins.

Then the files that the roles' [secret inputs](../../roles/server/ubuntu/frame/README.md#secret-inputs)
write. The seed leaves their inputs empty, so none of gh's token, `/etc/ws/gh-token`, git's,
`/etc/ws/git-tokens.toml`, and the shell's `/etc/ws/env` is there. The repository then gives each
a value, masked in the log, the shell's an endpoint of a `.invalid` host. The task's run writes
the three, root's at 600, under ws's header. `/etc/ws/env` exports the endpoint in single
quotes, and sh reads it from there. GitHub refuses a dummy token, so root's gh holds it as a login
with it would, and gh's unit compares that with the file's last line. Root's git gets git's token
through [ws's program](../../roles/server/ubuntu/git/README.md#git-credentials). A plan through
the task exits 0, and a run and a plan without the env `secrets` keep the three files, their
bytes, mode and group. A file at gh's path that ws didn't write stops the run, which keeps it. A
run without the env keeps it too, as mise loads no guard there. An empty input removes its file.

The runner has Azure's routes, so only CI proves them. `.github/actionlint.yaml` names the label
`ubuntu-26.04`, as actionlint 1.7.12 doesn't know it yet. It goes once a release lists the label.
