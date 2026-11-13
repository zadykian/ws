---
paths:
  - "cmd/**"
  - "main.go"
  - "internal/**"
  - "engine/**"
  - "roles/**"
---

# mac

The rules a change to the ws CLI keeps, in `main.go`, `cmd/`, `internal/`, `engine/` and `roles/`
(its module is at the root), each linking its reason
([#273](https://github.com/zadykian/ws/issues/273)). The hard rules of
[CLAUDE.md](../../CLAUDE.md#hard-rules) hold here too.

- Each step is isolated behind `step.Step`: it plans and applies one part of the setup, and reads
  no other step's state. A step that needs another done first names it in `Step.WaitsOn`
  ([step.go](../../engine/core/step/step.go)).
- A run reports each step to a `step.Observer`, and each step ends with one event of `Done`,
  `Failed` or `Skipped`. A run inside another passes the observer on, and names its role or stage
  as `Options.Parent` ([step.go](../../engine/core/step/step.go)).
- The Mac's steps are `setup.Mac`'s: a caller runs, reads or names them through it, and builds none
  itself. Each step is a role of one step, and `Run` runs them through `role.Run` as one role under
  the run's parent ([mac.go](../../roles/client/macos/setup/mac.go)).
- A step is pure wherever it can be: it reads, a pure function turns what it read into `Change`
  values, and it applies them. Tests compare the values.
- A whole file that ws writes in the user's home is a `userFile`, and a directory it writes under
  a `userDir`: a plan and a report read each alike. Its step chooses to warn or to stop at a file
  that is not ws's ([report.go](../../roles/client/macos/setup/report.go),
  [userdir.go](../../roles/client/macos/setup/userdir.go)). ws's lines in a user's file are `userLines`,
  whose step warns or stops at a link or at misplaced lines
  ([userlines.go](../../roles/client/macos/setup/userlines.go)).
- Effects are passed in: a step holds `effect.Prompt`, `effect.Out` and `effect.Machine` as fields
  and a function takes them as parameters. Tests use the fakes of
  [effecttest](../../engine/core/effect/effecttest/effecttest.go), never the real ones.
- A step that wants sudo's password ahead of its lines calls `effect.Validate`, never `sudo -v`
  by `Pass`, which would ask over a screen ([tui.md](../../docs/tui.md#commands)).
- A command that asks the maintainer, for a passphrase or a login, runs by `effect.Foreground`.
  A step holds it as `F` beside `M`. `Pass` would keep the screen on the terminal ([tui.md](../../docs/tui.md#commands)).
- No cgo: ws builds with `CGO_ENABLED=0`. The race detector's cgo is the tests' alone
  ([Makefile](../../Makefile)).
- ws is cobra's: a command parses no flag or argument by hand and holds no usage text.
  `execute` runs the root, names the command in each error, and returns 2 for a usage error. A
  group of commands exists on the systems where it runs. `newMacCmd` is built everywhere and tested
  on Linux, and `groups` lists it in a `//go:build darwin` file
  ([groups_darwin.go](../../cmd/groups_darwin.go)). Commands take what they need from the
  process through `process`, which tests build from fakes.
