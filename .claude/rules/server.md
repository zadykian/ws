---
paths:
  - "internal/server/**"
  - "host/**"
---

# server

The invariants a change to a role keeps, each linking where the repository gives its reason. The
hard rules of [CLAUDE.md](../../CLAUDE.md#hard-rules) hold here too, as do [mac.md](mac.md)'s for
steps, and each role's page in `docs/` says what it sets up.

## The roles

- A role is the Go steps of `internal/server/NAME`, with its page, `docs/NAME.md`, and a line in
  the README ([README](../../README.md)). `roles.List` lists the roles in the order they run
  ([roles.go](../../internal/server/roles/roles.go)).
- The server's values are one struct, `values.Values`, parsed once from the machine's file with
  unknown keys refused. A role's steps read it and add no key without a commit of their own
  ([values.go](../../internal/server/values/values.go)).
- The secrets are one struct, `secrets.Secrets`, beside the values. ws decrypts it from age's armor
  in its process, as TOML with a table per kind. Unknown keys are refused, references expanded,
  and no error or output shows a value. Steps take it as their second argument
  ([secrets.go](../../internal/secrets/secrets.go), [The Mac's secrets](../../docs/mac-secrets.md)).
- The roles set the machine up for `ws_user`, root by default, in `ws_home`. A step run as `ws_user`
  sets `HOME`: root needs no sudo, a container has none, and `HOME` would otherwise be the
  caller's ([claude's install](../../internal/server/claude/install.go)).
- Root's reads and writes under `ws_home` go through `host.Plain` or `host.RefuseLinks`, as the user
  owns the home and could link a file elsewhere ([Links under the home](../../docs/security.md#links-under-the-home)).
- A version is pinned in the package of the role that installs it. SigNoz's images are pinned in
  `internal/server/signoz/casting.yaml`, as its compose files are forged from it.
- A restart or a reload that a watched change calls for is `host.Forced`. `Group.Apply` runs it
  after a failed change too, as the next run finds the file right and plans none.

## Runs

- A second run plans nothing, which CI's `playbook` job checks
  ([The `playbook` job](../../docs/checks.md#the-playbook-job)). A step that only reads plans no change.
- A dry run, `--check`, applies nothing and shows how the machine differs
  ([Dry runs](../../README.md#dry-runs)). A step reads in its Plan, a GET included
  ([devtools](../../internal/server/devtools/golang_plan.go)).
- Where a dry run skips a step, a step beside it shows what the run would change, as Go's move into
  `/usr/local/go` does ([claude's marketplaces](../../internal/server/claude/marketplaces.go)).
- A step that needs a booted machine's systemd checks `host.Booted`: without one, as in CI's
  container, it skips and says so.
- A file a role writes has a `# Written by github.com/zadykian/ws` line at its top, where its format
  takes comments, as the roles' embedded templates and files do. Each run replaces a change made to
  it by hand ([shell](../../docs/shell.md)).
- A tool that updates itself, such as claude or rustup, is installed only where missing
  ([claude's install](../../internal/server/claude/install.go)).
- A role given the secrets of `ws server apply --secrets`, changes nothing where they are
  absent. With them, the file is the whole truth: a run removes what it lacks, such as a git token.
  So a run without them, as CI's, keeps what a run with them made
  ([Git credentials](../../docs/shell.md#git-credentials)).

## Downloads and keys

- A download is checked against its SHA-256 before it writes `dest`, and downloaded again
  where `dest` holds another ([the Go download](../../host/download.go)).
- An apt repository's key goes in only with the primary fingerprint its entry pins
  ([devtools' pins](../../internal/server/devtools/pins.go)). Docker's key goes in only with the
  checksum its code pins, so that a changed key fails the run
  ([docker's repository](../../internal/server/docker/repository.go)).

## signoz

- foundryctl's compose files, in `internal/server/signoz/deployment/`, are never edited by hand: change
  `casting.yaml` and forge again ([The compose files](../../docs/signoz.md#the-compose-files)).
- The dashboards' files are kept as SigNoz's API returns them, so that the step can compare the
  two. After a change of SigNoz's image, refresh them and commit them with the casting
  ([The dashboards](../../internal/server/signoz/dashboards/README.md#refreshing-them)).
