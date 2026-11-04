# The keymap

F12's keys, in no editor's terms. The data is in [`internal/editor`](../internal/editor/), and each
editor has an adapter that renders it as that editor's file. Zed's adapter is
[`internal/editor/zed`](../internal/editor/zed/) and Fresh's
[`internal/editor/fresh`](../internal/editor/fresh/).

## The data

`internal/editor/keymap/TOPIC.yaml` maps a scope to rows. A row has an `action`, its `keys`, and
the `f12` ids it comes from. A topic holds a scope whole, or the rows of a scope that other topics
share. The order of the topics is the order of the rows in each scope.

A key is the physical key with its modifiers, as `ctrl+shift+/`:

- A stroke is modifiers in the order `ctrl`, `alt`, `cmd`, `shift`, joined by `+`, then one key.
- A chord is strokes joined by one space, as `ctrl+k ctrl+c`.
- A key is a letter, a digit, an unshifted sign, `f1` to `f24`, or a named key such as `enter`.
  `numpad*` is the one numpad key.
- A shifted sign, as `ctrl+?`, is refused. An editor spells it, and Zed's `ctrl-?` is a macOS fact.

An action is a group, a dot and what it does, as `edit.copy`. A key is bound once in a scope.

| Scope | Meaning |
| --- | --- |
| `global` | wherever focus is |
| `text` | every text input |
| `code` | the full editor alone |
| `signature` | the signature popup |
| `tree` | the file tree |
| `git` | the git panel |
| `commit` | the commit message |
| `debug` | a stopped debug session |
| `panel` | any panel but the terminal |
| `terminal` | the terminal |

## The contract

Each adapter holds a table of its own: a map of each action to a target in that editor, or to a
reason it has none. An action with neither fails a test, as does a stale row. So no key goes
missing from an editor unseen. For Zed the table is `internal/editor/zed/actions/`. Fresh's is
`internal/editor/fresh/actions/`.

## Changing a key

1. Change the row in `internal/editor/keymap/`.
2. Give a new action a target or a reason in each adapter's table.
3. Change each golden file by hand, in the same commit. For Zed, the file is
   [`keymap.json`](../internal/editor/zed/testdata/keymap.json). `go test` holds the render equal.
   Fresh has no golden file: its tests hold the rules.

## Where F12 comes from

F12 is the keymap GoLand and Rider use here: `F12.xml`, 523 actions with keys, over the 112 of
IntelliJ's default keymap, `$default.xml`, that F12 leaves alone. That makes 635 actions with
817 keystrokes.

An action that `F12.xml` names gets exactly F12's keys. One it does not name keeps
`$default.xml`'s. Ctrl stays Ctrl and Cmd stays Cmd.

The data is a snapshot of F12 as of 2026-10-23. It follows GoLand 2026.2's settings over the
defaults of build 2026.2.3 (GO-262.10968.67). `F12.xml` stays in GoLand's and Rider's settings.
No gate notices a drift, and no generator is in `tools/`.

### Why F12, not F24

GoLand's synced settings also activate a keymap named `F24` on the Mac. `F24.xml` has 642 actions
and `parent` `Visual Studio 2022`. Its 627 keystrokes are all F12's, and it has the same 34 Cmd
keystrokes. The maintainer asked for F12, which helix follows, so F12 stays the source. Nothing
reads F24, and "Ctrl stays Ctrl" holds either way.

## The pages

- [Zed's F12 keymap](keymap/zed.md): the adapter's counts, statuses and decisions.
- [The same for helix](helix-keymap.md).

Plain keys, such as the arrows and Enter, and the mouse stay each editor's own. The data has no
row for them.
