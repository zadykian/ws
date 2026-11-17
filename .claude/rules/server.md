---
paths:
  - "roles/server/**"
  - "engine/core/resource/**"
  - "engine/core/kind/**"
---

# server

The invariants a change to a role keeps, each linking where the repository gives its reason. The
hard rules of [CLAUDE.md](../CLAUDE.md#hard-rules) hold here too, as do [mac.md](mac.md)'s for
steps, and each role's README says what it sets up.

## The roles

- A role is the Go steps of `roles/server/ubuntu/NAME`, with its page, its `README.md`, and a line in
  the root README ([README](../../README.md)). `ubuntu.Roles` lists the roles in the order they run,
  each closed over the machine's values and secrets ([roles.go](../../roles/server/ubuntu/roles.go)).
- The server's values are one struct, `values.Values`, parsed once from the machine's file with
  unknown keys refused. A role's steps read it and add no key without a commit of their own
  ([values.go](../../roles/server/ubuntu/values/values.go)).
- The secrets are one struct, `secrets.Secrets`, beside the values. ws decrypts it from age's armor
  in its process, as TOML with a table per kind. Unknown keys are refused, references expanded,
  and no error or output shows a value. Steps take it as their second argument
  ([secrets.go](../../roles/server/ubuntu/secrets/secrets.go), [ws's secrets](../../roles/server/ubuntu/secrets/README.md)).
- `secrets.Store` alone reads and writes ws's secrets file and makes its credential; cmd keeps
  the commands and the root check ([store.go](../../roles/server/ubuntu/secrets/store.go)).
- The roles set the machine up for `WS_USER`, `zadykian` by default, in `WS_HOME`. A step run as
  `WS_USER` sets `HOME` and starts in it: root needs no sudo, a container has none, and `HOME` would
  otherwise be the caller's ([claude's install](../../roles/server/ubuntu/claude/install.go)).
- Root reaches a file under `WS_HOME` through the account (`Account.File`, `Account.Dir`,
  `Account.Absent`, `User.ReadHome`), or the step acts as `WS_USER`, as claude's does. The user owns
  the home and could link a file elsewhere ([Links under the home](../../roles/server/ubuntu/security/README.md#links-under-the-home)).
- ws's lines in a home file are `Account.Lines`, which refuses any link, at the file or on the way.
- A version is pinned in the package of the role that installs it. SigNoz's images are pinned in
  `roles/server/ubuntu/signoz/casting.yaml`, as its compose files are forged from it.
- A restart or a reload that a watched change calls for is `resource.Forced`. `Group.Apply` runs it
  after a failed change too, as the next run finds the file right and plans none. It waits, owed,
  where a watched change itself did not apply, such as a file that its check refused
  ([unit.go](../../engine/core/resource/systemd/unit.go)).

## Runs

- A kind names no role's word: a role's fact comes in by its spec and its templates
  ([kind](../../engine/core/kind/README.md)).
- A second run plans nothing, which CI's `playbook` job checks
  ([The `playbook` job](../../.github/workflows/README.md#the-playbook-job)). A step that only reads plans no change.
- A dry run, `--check`, applies nothing and shows how the machine differs
  ([Dry runs](../../README.md#dry-runs)). A step reads in its Plan, a GET included
  ([devtools](../../roles/server/ubuntu/devtools/internal/golang/golang.go)).
- A role's test states dpkg's database by effecttest's package table, never by a `dpkg-query` line
  in Results, so apt's read can change under it
  ([machine_dpkg.go](../../engine/core/effect/effecttest/machine_dpkg.go)).
- A package is installed where dpkg lists it `ii`, in plans and reports alike, and a step reads dpkg
  in one run ([dpkg.go](../../engine/core/resource/apt/dpkg.go)). A role reads dpkg through `apt.PackageStates`.
- A report reads as Plan does and holds no secret: a `Secret` file shows no diff and no sum
  ([report.go](../../engine/core/step/report.go)). A `File` that a role builds in `Plan` shows by its change
  alone, so a healthy one has no item.
- Where a dry run skips a step, a step beside it shows what the run would change, as Go's move into
  `/usr/local/go` does ([claude's marketplaces](../../roles/server/ubuntu/claude/marketplaces.go)).
- A step that needs a booted machine's systemd checks `systemd.Booted`: without one, as in CI's
  container, it skips and says so.
- A file ws writes whole, on the server or the Mac, has one line at its top, where its format
  takes comments: `file.Header(sign)`. The roles' templates and files spell it out. Its words,
  `Written by github.com/zadykian/ws`, are `file.Mark`, and `file.Ours` tests a first line for them.
  Each run replaces a change made to it by hand ([shell](../../roles/server/ubuntu/shell/README.md), [git](../../roles/server/ubuntu/git/README.md)).
- A tool that updates itself, such as claude or rustup, is installed only where missing
  ([claude's install](../../roles/server/ubuntu/claude/install.go)).
- A role given the secrets of `ws server ubuntu apply --secrets`, changes nothing where they are
  absent. With them, the file is the whole truth: a run removes what it lacks, such as a git token.
  So a run without them, as CI's, keeps what a run with them made
  ([Git credentials](../../roles/server/ubuntu/git/README.md#git-credentials)).

## Downloads and keys

- A download is checked against its SHA-256 before it writes `dest`, and downloaded again
  where `dest` holds another ([the Go download](../../engine/core/resource/fetch/download.go)).
- A fetch tries five times, ten seconds apart, and stops at once on a wrong SHA-256, which
  `effect.ErrSHA256` names ([the fetch](../../engine/core/resource/fetch/download.go)).
- A package plans from its pin, a name and a version, and fetches its file only where that version
  is not installed. Before dpkg, it checks the file's own name and version against the pin
  ([package.go](../../engine/core/resource/apt/package.go)).
- A tree is unpacked beside its target and moved in whole, and its marker is its probe
  ([tree.go](../../engine/core/resource/fetch/tree.go)).
- An apt repository's key goes in only where its entry pins the primary fingerprint of each
  certificate in it, and it holds one at least
  ([devtools' pins](../../roles/server/ubuntu/devtools/pins.go)). Docker's key goes in only with the
  checksum its code pins, so that a changed key fails the run
  ([docker's repository](../../roles/server/ubuntu/docker/repository.go)).

## signoz

- foundryctl's compose files, in `roles/server/ubuntu/signoz/deployment/`, are never edited by
  hand: change `casting.yaml` and forge again ([The compose files](../../roles/server/ubuntu/signoz/README.md#the-compose-files)).
- The dashboards' files are kept as SigNoz's API returns them, so that the step can compare the
  two. After a change of SigNoz's image, refresh them and commit them with the casting
  ([The dashboards](../../roles/server/ubuntu/signoz/internal/dashboards/dashboards/README.md#refreshing-them)).
