# secrets

The `secrets` role installs fnox, which decrypts ws's secrets on the server, for the task
`secrets`. It has no spec and no unit: its `mise.server.toml` pins fnox, and `github_known_hosts`
pins GitHub's key for [the fetch](#the-fetch).

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
- Its exit is the bootstrap's. Each refusal of its own exits 1.
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
