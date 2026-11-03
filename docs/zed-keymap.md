# Zed's F12 keymap

F12 is the keymap GoLand and Rider use here: `F12.xml`, 523 actions with keys, over the 112 of
IntelliJ's default keymap, `$default.xml`, that F12 leaves alone. That makes 635 actions with
817 keystrokes. [`mac/zed-keymap.json`](../mac/zed-keymap.json) gives Zed the keys of 271 of them,
and the pages below say what becomes of the other 364. The Mac step writes the file to
`~/.config/zed/keymap.json` ([The keymap](mac-zed.md#the-keymap)).

An action that `F12.xml` names gets exactly F12's keys. One it does not name keeps
`$default.xml`'s. Ctrl stays Ctrl and Cmd stays Cmd.

The file is a snapshot of F12 as of 2026-10-23. It follows GoLand 2026.2's settings over the
defaults of build 2026.2.3 (GO-262.10968.67). `F12.xml` stays in GoLand's and Rider's settings.
No gate notices a drift, and no generator is in `tools/`.

## The pages

- [The Zed keymap on a Mac](zed-keymap/macos.md): the keys a Mac keyboard lacks, the F-keys and
  Fn, and macOS's own shortcuts on F12's keys.
- [The actions the keymap binds](zed-keymap/actions.md), Editing to Folding, with Zed's keys.
- [The same, in the IDE](zed-keymap/actions-ide.md), Problems to Terminal.
- The actions the keymap lacks, in the order of their names, upper and lower case alike:
  [A to J](zed-keymap/lacks-a-j.md) and [L to Z](zed-keymap/lacks-l-z.md).

The statuses are `bind`, the same behaviour, and `approx`, the closest Zed action. Then come `zed`,
where Zed's own default already does it, and `skip`, where Zed has the action and the binding is
left out. `none` means Zed lacks the action, and `n/a` a tool Zed lacks or needs not.

| Source | bind | approx | zed | skip | none | n/a | all |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `F12.xml` | 120 | 63 | 16 | 7 | 34 | 283 | 523 |
| `$default.xml`, untouched | 25 | 17 | 30 | 3 | 11 | 26 | 112 |
