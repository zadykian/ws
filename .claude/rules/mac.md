---
paths:
  - "mac/**"
---

# mac

The rules a change to the ws CLI in `mac/` keeps, each linking its reason
([#273](https://github.com/zadykian/ws/issues/273)). The hard rules of
[CLAUDE.md](../../CLAUDE.md#hard-rules) hold here too.

- Each step is isolated behind `step.Step`: it plans and applies one part of the setup, and reads
  no other step's state. A step that needs another done first names it in `Step.WaitsOn`
  ([step.go](../../mac/internal/step/step.go)).
- A step is pure wherever it can be: it reads, a pure function turns what it read into `Change`
  values, and it applies them. Tests compare the values.
- Effects are passed in: a step holds `effect.Prompt`, `effect.Out` and `effect.Machine` as fields
  and a function takes them as parameters. Tests use the fakes of
  [effecttest](../../mac/internal/effect/effecttest/effecttest.go), never the real ones.
- No cgo: ws builds with `CGO_ENABLED=0`. The race detector's cgo is the tests' alone
  ([Makefile](../../Makefile)).
