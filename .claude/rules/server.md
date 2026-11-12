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
  ([secrets.go](../../internal/secrets/secrets.go), [ws's secrets](../../docs/mac-secrets.md)).
- The roles set the machine up for `ws_user`, `zadykian` by default, in `ws_home`. A step run as
  `ws_user` sets `HOME` and starts in it: root needs no sudo, a container has none, and `HOME` would
  otherwise be the caller's ([claude's install](../../internal/server/claude/install.go)).
- Root reaches a file under `ws_home` through the account (`Account.File`, `Account.Dir`,
  `Account.Absent`, `User.ReadHome`), or the step acts as `ws_user`, as claude's does. The user owns
  the home and could link a file elsewhere ([Links under the home](../../docs/security.md#links-under-the-home)).
- A line the maintainer acts on past a run, such as a key to copy, goes through
  `effect.SayLasting`, as a screen's window goes with it ([The run screen](../../docs/tui.md#the-run-screen)).
- A version is pinned in the package of the role that installs it. SigNoz's images are pinned in
  `internal/server/signoz/casting.yaml`, as its compose files are forged from it.
- A restart or a reload that a watched change calls for is `host.Forced`. `Group.Apply` runs it
  after a failed change too, as the next run finds the file right and plans none.

## Runs

- A second run plans nothing, which CI's `playbook` job checks
  ([The `playbook` job](../../docs/checks.md#the-playbook-job)). A step that only reads plans no change.
- A dry run, `--check`, applies nothing and shows how the machine differs
  ([Dry runs](../../README.md#dry-runs)). A step reads in its Plan, a GET included
  ([devtools](../../internal/server/devtools/golang.go)).
- A role's test states dpkg's database by effecttest's package table, never by a `dpkg-query` line
  in Results, so host's read can change under it
  ([machine_dpkg.go](../../effect/effecttest/machine_dpkg.go)).
- A package is installed where dpkg lists it `ii`, in plans and reports alike, and a step reads dpkg
  in one run ([dpkg.go](../../host/dpkg.go)). A role reads dpkg through `host.PackageStates`.
- A report reads as Plan does and holds no secret: a `Secret` file shows no diff and no sum
  ([report.go](../../step/report.go)). A `File` that a role builds in `Plan` shows by its change
  alone, so a healthy one has no item.
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
- A fetch tries five times, ten seconds apart, and stops at once on a wrong SHA-256, which
  `effect.ErrSHA256` names ([the fetch](../../host/download.go)).
- A package plans from its pin, a name and a version, and fetches its file only where that version
  is not installed. Before dpkg, it checks the file's own name and version against the pin
  ([package.go](../../host/package.go)).
- A tree is unpacked beside its target and moved in whole, and its marker is its probe
  ([tree.go](../../host/tree.go)).
- An apt repository's key goes in only where its entry pins the primary fingerprint of each
  certificate in it, and it holds one at least
  ([devtools' pins](../../internal/server/devtools/pins.go)). Docker's key goes in only with the
  checksum its code pins, so that a changed key fails the run
  ([docker's repository](../../internal/server/docker/repository.go)).

## signoz

- foundryctl's compose files, in `internal/server/signoz/deployment/`, are never edited by hand: change
  `casting.yaml` and forge again ([The compose files](../../docs/signoz.md#the-compose-files)).
- The dashboards' files are kept as SigNoz's API returns them, so that the step can compare the
  two. After a change of SigNoz's image, refresh them and commit them with the casting
  ([The dashboards](../../internal/server/signoz/dashboards/README.md#refreshing-them)).
