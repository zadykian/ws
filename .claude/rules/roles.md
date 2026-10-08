---
paths:
  - "roles/**"
---

# roles

The invariants a change to a role keeps, each linking where the repository gives its reason. The
hard rules of [CLAUDE.md](../../CLAUDE.md#hard-rules) hold here too, and each role's page in
`docs/` says what it sets up.

## The play

- A role is a tag of `site.yml` of its name, with its page, `docs/NAME.md`, and a line in the
  README ([README](../../README.md)).
- The play runs on localhost, where the controller's test of a path, such as `is file`, sees the
  machine's own files ([otelcol's tasks](../../roles/otelcol/tasks/main.yml)).
- The roles set the machine up for `ws_user`, root by default, in `ws_home`
  ([group_vars/all.yml](../../group_vars/all.yml)). A task run as `ws_user` takes
  `become: "{{ ws_user != 'root' }}"` and sets `HOME`. root needs no become, a container has no
  sudo, and without become `HOME` would be the caller's
  ([claude's tasks](../../roles/claude/tasks/main.yml)).
- A version goes in `group_vars/all.yml`, named for the role that installs it. SigNoz's images are
  pinned in `roles/signoz/files/casting.yaml`, as its compose files are forged from it.
- A role's variables carry its name as a prefix, as ansible-lint's `var-naming` rule asks
  ([.ansible-lint](../../.ansible-lint)).
- A failed task still runs the handlers it notified, as the next run would not notify them again
  ([ansible.cfg](../../ansible.cfg)).

## Runs

- A second run changes nothing, which CI's `playbook` job checks
  ([The playbook](../../docs/checks.md#the-playbook)). A task that only reads reports no change,
  `changed_when: false`.
- A dry run, `--check --diff`, changes nothing and shows how the machine differs
  ([Dry runs](../../README.md#dry-runs)). A read, such as a GET, runs in a dry run too,
  `check_mode: false` ([devtools' tasks](../../roles/devtools/tasks/main.yml)).
- Where a dry run skips a task, a task beside it shows what the run would change, as Go's move into
  `/usr/local/go` does ([devtools' tasks](../../roles/devtools/tasks/main.yml)).
- A task that needs a booted machine's systemd is tagged `systemd`, which CI's container run skips.
  An include tagged `systemd` gives its tasks the tag through `apply`, so that `--tags systemd`
  runs them too ([signoz's tasks](../../roles/signoz/tasks/main.yml)).
- A file a role writes has a `# Written by github.com/zadykian/ws` line at its top, where its format
  takes comments, as the roles' templates and files do. Each run replaces a change made to it by
  hand ([shell](../../docs/shell.md)).
- A tool that updates itself, such as claude or rustup, is installed only where missing,
  with `creates:` ([claude's tasks](../../roles/claude/tasks/main.yml)).

## Downloads and keys

- get_url checks a download against its SHA-256, `checksum: sha256:...`, before it writes `dest`,
  and downloads again where `dest` holds another
  ([devtools' tasks](../../roles/devtools/tasks/main.yml)).
- An apt repository's key goes in only with the primary fingerprint its entry pins
  ([devtools' vars](../../roles/devtools/vars/main.yml)). Docker's key goes in only with the
  checksum its task pins, so that a changed key fails the run
  ([docker's tasks](../../roles/docker/tasks/main.yml)).

## signoz and the filter plugins

- foundryctl's compose files, in `roles/signoz/files/deployment/`, are never edited by hand: change
  `casting.yaml` and forge again ([The compose files](../../docs/signoz.md#the-compose-files)).
- The dashboards' files are kept as SigNoz's API returns them, so that the role can compare the
  two. After a change of SigNoz's image, refresh them and commit them with the casting
  ([The dashboards](../../roles/signoz/files/dashboards/README.md#refreshing-them)).
- The filter plugins are Python that runs on the controller, whose ansible-core 2.20 needs Python
  3.12 or newer, under `make ruff` ([ruff.toml](../../ruff.toml)).
