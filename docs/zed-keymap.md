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

## How the keys translate

Zed on macOS, with a US layout:

- Shift with a sign or digit is written as the shifted character without Shift, such as `alt-{`
  and `ctrl-?`. macOS reports it so, and `shift-alt-[` never matches. Shift with a letter stays.
- Zed does not tell numpad keys apart ([zed#11948](https://github.com/zed-industries/zed/issues/11948)).
- A Mac keyboard has no Insert, Pause, Break or menu key, so six F12 keystrokes drop.
- F-keys need Fn, or a setting of System Settings. [The Mac's page](zed-keymap/macos.md) has both.
- Alt with a letter matches by key, so an unbound Alt key still types its character.

## How Zed picks a binding

- The deeper context wins, then the later binding. User bindings load after Zed's.
- A binding with no context counts as the deepest, so the file's global section beats every default.
- A chord whose first key also has a binding waits 1 s, then runs that binding. A chord defined
  before a binding of its first key is ignored (zed#30211). So the file unbinds Zed's Ctrl+K,
  Ctrl+M and Ctrl+R.
- `unbind` works only where the section's context matches, so Zed's Ctrl+R in the Diagnostics
  view still waits.

## The file's sections

The file has 264 bindings and 24 unbindings in 10 sections, in this order:

- F12's global keys, with no context.
- Editing, in every editor: `Editor`.
- Code, in the full editor alone, so that search bars and pickers keep their keys.
- The signature popup, the project panel, the git panel and the commit message.
- A stopped debug session, where the step keys leave the Alt keys to type.
- Esc, which goes back to the editor from a panel. The terminal keeps its own Esc.
- The terminal, last.

## The Terminal's rule

In the terminal, a Ctrl or Alt key with a letter or sign, without Shift, reaches the shell
(zed#26018). The section unbinds 21 global keys, and the chords' first keys go straight
(zed#28423). F-keys, named keys and digits stay Zed's. Ctrl+C, Ctrl+D and Ctrl+R stay the shell's.
F12's Ctrl+B is in the editor only, so the terminal gets it.
