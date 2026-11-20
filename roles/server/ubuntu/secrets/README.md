# secrets

The `secrets` role installs fnox, which decrypts ws's secrets on the server, for the task
`secrets`. It has no spec and no unit: its `mise.server.toml` pins fnox, and `github_known_hosts`
pins GitHub's key for [the fetch](#the-fetch). The secrets sit in a repository of fnox's files
([The repository](#the-repository)).

## fnox

mise installs fnox 1.50.0 from jdx/fnox's release on GitHub, as `[tools]` pins it, before any unit
of ws. [mise.server.lock](../../mise/mise.server.lock) holds the url of the asset for `linux-x64`
and `linux-arm64`, with GitHub's digest of each, which mise checked against the release's
attestations as it wrote the lock. A wrong sum fails the run, and `MISE_LOCKED=1` refuses a fnox
that the lock lacks. `go test` fails where the pin, the lock and the digests that `pins_test.go`
holds disagree.

The pin names mise's `github` backend, `github:jdx/fnox`, which downloads from the lock's url
alone. The registry's name, `fnox`, takes the `packslip` backend, which asks GitHub's API for the
release on each first install, unauthenticated. A runner or a cloud VM behind a shared address
meets the API's limit there, and the install fails.

- To bump fnox, change its version in `[tools]`, and the sums in `pins_test.go` to GitHub's
  digests of the new assets. Then write the lock again in a container with git and mise:
  `MISE_ENV=server mise -C roles/server lock --platform linux-x64,linux-arm64`, which writes
  `roles/server/mise/mise.server.lock`.
- The Mac pins the same version of fnox, as both sides read one repository of secrets.

## The task

The task `secrets`, `roles/server/mise-tasks/secrets`, runs the bootstrap under ws's secrets. Run
it at ws as the [runbook](../../runbook.md) gives it:

```sh
sudo env MISE_DATA_DIR=/usr/local/share/mise MISE_LOCKED=1 mise -C /opt/ws/roles/server run secrets
```

- Its arguments go to `mise bootstrap` whole: none is a run, and `-- plan --detailed-exitcode` a
  plan. A plan fetches too, as mise needs the values to compare the secret files.
- It refuses any user but root, and a run without the runbook's `MISE_DATA_DIR` and
  `MISE_LOCKED=1`, so fnox runs from the lock alone.
- The profile is the hostname, one DNS label that names no recipient. The task decrypts
  `fnox.HOSTNAME.toml` with the shared `fnox.toml` by the host key,
  `/etc/ssh/ssh_host_ed25519_key`, then runs `mise -E ws,secrets bootstrap` under fnox. Each
  value reaches mise as a variable of its name.
- fnox fails closed: `--if-missing error` stops at a value it cannot decrypt, an empty
  `FNOX_CONFIG_DIR` keeps root's config out, and no key but the host's takes part.
- The bootstrap runs with the caller's umask, and with none of the fetch's settings of git and
  ssh, as fnox passes its environment on. A unit run as `WS_USER` starts from a cleared one, so
  the account reads no value from its processes ([run](../../../../engine/core/kind/state.md)).
- Its exit is the bootstrap's, and each refusal of its own exits 1. A plan that finds a change
  exits 2, and mise then prints `[secrets] ERROR task failed`, as it does for any task that exits
  other than 0. That line reports the 2, not a failed plan, and no flag of `mise run` or option of
  a task drops it ([The rootless job](../../../../.github/workflows/rootless.md)).
- `WS_SECRETS_PREFIX`, for the tests alone, stands for `/` in front of the host key and
  `/var/lib/ws`. sudo drops it from the caller's environment.

A run of the bootstrap without the task loads no `secrets` env, and keeps the files that env's
inputs write as they are.

## The fetch

The task clones the repository that `WS_SECRETS_REPO` names, in the root `mise.toml`, into
`/var/lib/ws/secrets`, root's and 700, on `main`. A later run fast-forwards it. The host key is the
repository's read-only deploy key, titled with the hostname.

- ssh reads nothing of root's own. `-F /dev/null` keeps root's `~/.ssh/config` out, since
  `IdentitiesOnly` still takes a config's keys. `IdentityAgent=none` keeps an agent's keys out.
  `BatchMode=yes` asks nothing.
- GitHub's ed25519 key alone proves github.com, as
  [github_known_hosts](github_known_hosts) pins it. `pins_test.go` checks it against GitHub's
  published fingerprint.
- fnox reads every fnox file of the clone, an ignored one too. So the task refuses a clone that
  differs from origin's `main`. That is a link, a change, a file or a commit of its own, another
  branch or origin, or a `main` that origin rewrote. Remove the clone, and the next run clones it
  again.
- A Mac mends three refusals by `ws client macos secrets add NAME`: GitHub refuses the host key,
  the repository holds no profile of the hostname, or the host key opens none of its values.

## The repository

ws keeps no secret of its own. The secrets sit in the private repository that `WS_SECRETS_REPO`
names, in the `[env]` of the root `mise.toml`, as fnox's files, whose values age encrypts. A
server reads it by [the task](#the-task) and [the fetch](#the-fetch) above. The Mac changes it
([The secrets on the Mac](../../../client/macos/secrets/README.md)).

### The files

- `fnox.toml` holds the values that every server shares. `root = true` keeps fnox from a file
  above the clone, and `if_missing = "error"` fails a run at a value that no key opens. Its
  provider `age`, of type age, encrypts each value to every recipient.
- `fnox.HOSTNAME.toml` is one server's profile, its own values, which fnox reads beside
  `fnox.toml` under `-P HOSTNAME` alone. Its provider `host`, of type age, encrypts to the Nano,
  the backup and that server alone, and each of its values names `provider = "host"`. The name
  differs from `age`, as fnox merges providers by name. Under the profile, a second `age` would
  replace the shared one, and a re-encryption would hand the shared values to one server alone.

No provider sets `key_file`, `identity` or `default_identity`, as each would beat the key that the
reader names: the host key that the task gives by `FNOX_AGE_KEY_FILE`, or the Nano's stub on the
Mac.

### The recipients

- `nano`: the YubiKey 5C Nano, by age-plugin-yubikey, with a PIN once a session and a touch each
  time it decrypts.
- `backup`: a plain age key, which opens every value where the Nano is lost. `ws client macos
  secrets init` prints it once, and writes it nowhere.
- each server's hostname: its SSH host key, `/etc/ssh/ssh_host_ed25519_key`. The same key is the
  repository's read-only deploy key, titled with the hostname.

A server's key is a recipient of `fnox.toml` and of its own profile alone, so it opens the shared
values and its own, and no other server's.

### The inputs

Each input of a server role's `mise.secrets.toml` is a secret of that name
([Secret inputs](../frame/README.md#secret-inputs)). init declares each one in `fnox.toml`, empty,
so that mise finds each input set where a server has no value. An empty input writes no file, or
removes the one there.

| Input | Role | What mise or ws does with it |
| --- | --- | --- |
| `OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT` | [shell](../shell/README.md#the-env-file) | an export of `/etc/ws/env` |
| `WS_GITHUB_TOKEN` | [devtools](../devtools/README.md#gh) | `/etc/ws/gh-token`, gh's login |
| `WS_GIT_TOKENS` | [git](../git/README.md#git-credentials) | `/etc/ws/git-tokens.toml`, TOML of each URL's token |
| `WS_IKEWARD_P12` | [ikeward](../ikeward/README.md#the-profiles-inputs) | the `.p12` in base64, read from the run's env |
| `WS_IKEWARD_SSWAN` | [ikeward](../ikeward/README.md#the-profiles-inputs) | the `.sswan`, read from the run's env |
| `WS_IKEWARD_PASSWORD` | [ikeward](../ikeward/README.md#the-profiles-inputs) | the `.p12`'s password, read from the run's env |

A value of one server goes in its fnox profile, and wins there over the shared empty one, as
ikeward's inputs do.

### References

One input takes another's value by fnox's `default`, which names it, with no `value` beside it:

```toml
NAME = { default = "${OTHER}" }
```

- fnox decrypts OTHER first and gives NAME its value, an empty one too. A `value` beside the
  `default` wins, even an empty one, so the line holds the `default` in place of the value.
- In `fnox.HOSTNAME.toml`, the line replaces the empty NAME of `fnox.toml` under that profile
  alone, and so gives one server another input's value.
- OTHER is an input as well, never a secret of its own. fnox gives each secret to the run's env,
  and a run state hides the inputs alone from a command that is not ws's ([run](../../../../engine/core/kind/state.md)).
- A shared value never refers to a profile's: under any other profile, fnox would fail as it
  loads. A name that no file holds fails the run, one of the env too, and so does a loop.
- A reference goes in by hand, in a clone, on a branch, as a value does. The `values` check lets a
  `default` hold `${NAME}` references alone, as any other text would sit there in the clear.

### From the Mac

Every change lands on `main` by a pull request, from a temporary clone
([How a change lands](../../../client/macos/secrets/README.md#how-a-change-lands)):

- `ws client macos secrets init` starts `fnox.toml` with the Nano and the backup, and declares the
  inputs.
- `ws client macos secrets add NAME` makes the server's host key a recipient and its deploy key,
  then runs the task there.
- `ws client macos remove NAME` takes that key out of both.

A value goes in by hand, in a clone, on a branch: `fnox set NAME --provider age` for a shared one,
`fnox -P HOSTNAME set NAME --provider host` for one server's, the value on standard input. The
Mac's runbook gives the steps ([The secrets](../../../client/runbook.md#the-secrets)).

### The checks

The repository's ruleset on `main` takes signed commits and merge commits alone, and refuses a
force push and a deletion. It requires the checks `values` and `gitleaks` on a branch up to date
with `main`, so a push to `main` fails. `values` fails where a value is not an age ciphertext, or a
`default` holds more than `${NAME}` references. `gitleaks` fails where a file holds a credential.
