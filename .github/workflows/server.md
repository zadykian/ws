# The server job

CI sets up the scenario closest to a server installed from Ubuntu Server 26.04.1's image. The job
`server` runs on GitHub's `ubuntu-26.04` runner, which boots, as root with the defaults. The
runner's image may hold a package that ws forgot to install. A server from
Ubuntu's image lacks it, so the trial on that image finds it. The job sets the runner up as it
would a server that root alone logs in to: root's `authorized_keys` holds a key, and no user is
made. The runner's own Docker is purged first, as ws installs
its own. Its apt source goes too, as apt refuses a repository that two sources name with different
keys, and mise adds its own. So do the plugins that the image put in Docker's directory past dpkg,
which would keep it and warn. The image's source of the git-core PPA goes
too, as ws adds the PPA with its own key. Then apt's lists are fetched anew. The image's git comes
from that PPA, so git and git-man go down to the archive's version. The job fails where git is not
below 2.54 then. apt's daily timers lose the stamps of their runs at the image's
build, as a new machine's have none before their first start. With them, systemd takes a run as
missed at the boot. Each of the bootstrap's reloads draws the run's random delay again, and one
soon falls in the past: an upgrade then holds dpkg's lock under the bootstrap. The runner's
`/etc/ws/values.yml` sets a 64 MiB swap file, which base makes and swaps to, for the small disk. It
sets `WS_SELF_CONTAINED: true` too: the runner is no container and has no host, so the guest runs
SigNoz and its tunnel itself, as setup demands of one ([signoz](../../ubuntu/roles/signoz/README.md#the-self-contained-mode)). A
namespace holds a second network, `10.250.0.2`, which the cases reach published ports from. The
cases are [server-checks.sh](../server-checks.sh). The refusal has a control, which
runs after setup.

CI builds ws from the tree, never from a release. The tree goes in `/opt/ws`, root's alone, and
[`.tools/install-build`](../../.tools/install-build) installs the build as `/usr/local/bin/ws`, the
ws that runs
([ws at a fixed path](../../ubuntu/roles/frame/README.md#ws-at-a-fixed-path)). GitHub's runner keeps a default ACL on `/opt` that lets
anyone write what lands there, so the job drops it first. Its image lets others write `/opt` and
`/usr/share` too, where mise would refuse
[the plugin](../../ubuntu/roles/frame/README.md#the-plugin), so the cases make both root's
alone. As on a new machine, the build's `ws ubuntu guest setup --in-place` installs the pinned
mise and runs the bootstrap on the tree in place ([bootstrap.sh](../bootstrap.sh)). The tree's
`locked = true` has mise install each tool as the lock holds it. Before the first run, forwarding
is off and the FORWARD policy accepts, as on a cloud VM. After it, with no reboot, the policy must
still accept and forwarding be on
([The forward policy](../../ubuntu/roles/docker/README.md#the-forward-policy)).
No apt command in it may wait for dpkg's lock, and neither of apt's daily services may run. The
log shows the timers and their stamps before it and after it, and systemd's reloads, so that a
failure shows what started the upgrade. Setup's `--plan` must exit 2 before the run, which has
mise, the plugin and the bootstrap to do, and exit 0 after it. Setup runs twice, and the plan must
exit 0 again. Neither run nor plan may print a warning, whatever its exit code
([The warnings check](README.md#the-warnings-check)). Git must then be 2.54 or above, with the
PPA's version, so the setup moved it up
([base](../../ubuntu/roles/base/README.md#git)). `dotnet` must list the SDKs 10.0.404 and
11.0.100 alone, and roslyn-language-server must find .NET with no `DOTNET_ROOT`
([dotnet-checks.sh](../dotnet-checks.sh), [The .NET SDKs](../../ubuntu/roles/devtools/README.md#the-net-sdks)).
A failure prints the system daemon's log.

The run leaves checks that need no container. With the frame's line of `file/ws-values` dropped,
the plan must exit 1 ([server-plan.sh](../server-plan.sh)): the check against a silent skip holds.
A plan from `/opt/ws`, which loads no role, must exit 1 too, and name the run directory. A plan
with a bad value in the values file must exit 1 and name its key, and
the file goes back ([The check of the values](../../ubuntu/roles/frame/README.md#the-check-of-the-values)). The
values file is root's at 600, and no run changed its bytes ([The values
file](../../ubuntu/roles/frame/README.md#the-values-file)). A run with no file is
`values_file_test.go`'s. The Zed wrapper's [checks](../zed-terminal-checks.sh) run on the files the
bootstrap installed, and Fresh's on the config it loaded. Root's empty commit must sign and
verify, and hold one `Co-authored-by` trailer, which [git's hook](../../ubuntu/roles/git/README.md#the-co-author-trailer)
adds. With no NVIDIA GPU, `nvidia` checks its names alone, and with no MSI board, `fans` leaves
its fans alone. Without a VPN profile, `ikeward` does nothing.

Docker must pull the image, as a run of an image it lacks warns, and run a container that reaches
the internet. A guest holds no filter, so the cases hold no table. Then:

- A container's empty OTLP/gRPC export to the bridge address, port 4317, gets the collector's
  answer. One to port 4318 is refused ([otlp-checks.sh](../otlp-checks.sh),
  [Docker's containers](../../ubuntu/roles/otelcol/README.md#dockers-containers)).
- A port published with no address answers on loopback, and not on the address of another network.
  A guest holds no filter, so they go on loopback
  ([Published ports](../../ubuntu/roles/docker/README.md#published-ports)).
- Root reaches a container's address and a port published on loopback.
- A link at `~/.bashrc` is refused, by `ws unit write --dry-run` of the shell's line
  ([Shell](../../ubuntu/roles/shell/README.md)).

Then [the secrets task](../../ubuntu/roles/secrets/README.md#the-task), whose cases are
[secrets-checks.sh](../secrets-checks.sh). A bare repository on disk stands in for GitHub's, as
`WS_SECRETS_REPO` in `/opt/ws/mise.local.toml` names it. Its seed holds a shared value and one of
the profile, the hostname's, each encrypted to the runner's host key. Each input that mise lists
and the shared `fnox.toml` lacks is declared empty there, the profile's among them, whose own value
must win. CI's own `/opt/ws/mise.secrets.toml` writes both values to
`/etc/ws-ci-secrets`, root's 600, under the env `secrets` alone. The task runs three times, by
`ws ubuntu guest setup --in-place --secrets`, and its `--plan` exits 0 after each run. A plain plan
after the first run keeps the file. A plan through the task then fetches a new shared value. It
exits 2 and keeps the file, and mise's line of a failed task ends it, outside the warnings check.
The second run writes the new value. A run with
`MISE_LOCKED=0`, where mise's setting locked prints `false`, is refused and names
`ws ubuntu guest setup --secrets`. A file of the clone's own, a profile that the host key can't
open and a missing profile are refused too, and the file keeps its bytes. The last two name
`ws macos secrets add NAME`. The third run follows them, and the file still holds both values. No
output of the task's holds a value. Only GitHub proves
the fetch from github.com by a deploy key, against the key that ws pins.

Then the files that the roles' [secret inputs](../../ubuntu/roles/frame/README.md#secret-inputs)
write. The seed leaves their inputs empty, so none of gh's token, `/etc/ws/gh-token`, git's,
`/etc/ws/git-tokens.toml`, and the shell's `/etc/ws/env` is there. The repository then gives each
a value, masked in the log, the shell's an endpoint of a `.invalid` host. The task's run writes
the three, root's at 600, under ws's header. `/etc/ws/env` exports the endpoint in single
quotes, and sh reads it from there. GitHub refuses a dummy token, so root's gh holds it as a login
with it would, and gh's unit compares that with the file's last line. Root's git gets git's token
through [ws's program](../../ubuntu/roles/git/README.md#git-credentials). A plan through
the task exits 0, and a run and a plan without the env `secrets` keep the three files, their
bytes, mode and group. A file at gh's path that ws didn't write stops the run, which keeps it. A
run without the env keeps it too, as mise loads no guard there. That run fails at gh's unit, as
gh's login by the file's last line fails. An empty input removes its file.

The runner has Azure's routes, so only CI proves them. `.github/actionlint.yaml` names the label
`ubuntu-26.04`, as actionlint 1.7.12 doesn't know it yet. It goes once a release lists the label.
