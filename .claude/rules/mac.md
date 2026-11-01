---
paths:
  - "mac/**"
  - "cmd/**"
  - "internal/**"
  - "step/**"
  - "effect/**"
  - "host/**"
---

# mac

The rules a change to the ws CLI keeps, in `cmd/`, `internal/`, `step/`, `effect/` and `host/` (its
module is at the root), each linking its reason
([#273](https://github.com/zadykian/ws/issues/273)). The hard rules of
[CLAUDE.md](../../CLAUDE.md#hard-rules) hold here too.

- Each step is isolated behind `step.Step`: it plans and applies one part of the setup, and reads
  no other step's state. A step that needs another done first names it in `Step.WaitsOn`
  ([step.go](../../step/step.go)).
- A step is pure wherever it can be: it reads, a pure function turns what it read into `Change`
  values, and it applies them. Tests compare the values.
- Effects are passed in: a step holds `effect.Prompt`, `effect.Out` and `effect.Machine` as fields
  and a function takes them as parameters. Tests use the fakes of
  [effecttest](../../effect/effecttest/effecttest.go), never the real ones.
- No cgo: ws builds with `CGO_ENABLED=0`. The race detector's cgo is the tests' alone
  ([Makefile](../../Makefile)).
- A group of commands exists on the systems where it runs. `ws mac` is `macCommand`, which is built
  everywhere and tested on Linux. Its entry in the table of groups is in a `//go:build darwin` file
  ([groups_darwin.go](../../cmd/ws/groups_darwin.go)).
- A restart or a reload that a watched change calls for is `host.Forced`. `Group.Apply` still runs
  it after a failed change, as `force_handlers` did, as the next run finds the file right and
  plans none ([ansible.cfg](../../ansible.cfg)).
- The server's values are one struct, `values.Values`, parsed once from the machine's file with
  unknown keys refused. A role's steps read it and add no key without a commit of their own
  ([values.go](../../internal/server/values/values.go)).
- The secrets are one struct, `secrets.Secrets`, beside the values: decrypted from age's armor in
  ws's process, unknown keys refused, no error or output showing a value. Steps take it as their
  second argument, and the roles left to the playbook get it on a pipe, `-e @/dev/fd/3`, never a
  file ([secrets.go](../../internal/server/secrets/secrets.go)).
- `roles.List` is the server's roles in `site.yml`'s order. A port sets its role's `Steps` there
  and removes the role from `site.yml`, which a test checks
  ([roles.go](../../internal/server/roles/roles.go)).
