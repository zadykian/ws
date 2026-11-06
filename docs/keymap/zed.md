# Zed's F12 keymap

F12's 635 actions carry 817 keystrokes ([where F12 comes from](../keymap.md#where-f12-comes-from)).
The adapter gives Zed the keys of 271 of them, and the pages below say what becomes of the other
364. The golden file is [`keymap.json`](../../internal/editor/zed/testdata/keymap.json). The Mac
step writes the file to `~/.config/zed/keymap.json` ([The keymap](../mac-zed.md#the-keymap)).

## The pages

- [The Zed keymap on a Mac](zed-macos.md): the keys a Mac keyboard lacks, the F-keys and
  Fn, and macOS's own shortcuts on F12's keys.
- [The actions the keymap binds](../../internal/editor/keymap/): the data, by topic, with F12's ids.
- The actions the keymap lacks, in the order of their names, upper and lower case alike:
  [A to J](zed-lacks-a-j.md) and [L to Z](zed-lacks-l-z.md).

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
- F-keys need Fn, or a setting of System Settings. [The Mac's page](zed-macos.md) has both.
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

The file has 266 bindings and 24 unbindings in 10 sections, in this order:

- F12's global keys, with no context.
- Editing, in every editor: `Editor`.
- Code, in the full editor alone, so that search bars and pickers keep their keys.
- The signature popup, the project panel, the git panel and the commit message.
- A stopped debug session, where the step keys leave the Alt keys to type.
- Esc, which goes back to the editor from a panel. The terminal keeps its own Esc.
- The terminal, last.

## Two keys F12 lacks

F12 has no action to restart a language server or to reload Zed, and Zed has no default key for
either. They are the remedies after a reconnect (`docs/zed.md`, "When the connection drops").
The file binds both, and both are free in F12's keymap and in Zed's defaults:

- Ctrl+Shift+R is `editor::RestartLanguageServer`, in `Editor && mode == full`.
- Ctrl+Alt+Shift+Y is `workspace::Reload`, in the global section.

Shift keeps the Terminal's rule below, so Ctrl+Alt+Shift+Y reaches Zed from the terminal too.

## The Terminal's rule

In the terminal, a Ctrl or Alt key with a letter or sign, without Shift, reaches the shell
(zed#26018). The section unbinds 21 global keys, and the chords' first keys go straight
(zed#28423). F-keys, named keys and digits stay Zed's. Ctrl+C, Ctrl+D and Ctrl+R stay the shell's.
F12's Ctrl+B is in the editor only, so the terminal gets it.

## Decisions in the file

- **Agent:** F7 and Alt+7 open the agent panel (`agent::ToggleFocus`). Ctrl+/ is the inline
  assistant (`assistant::InlineAssist`) in the full editor, as F12's Ctrl+Enter there is code
  actions.
- **Cmd+Y** leaves F12's Redo for the agent's Allow once and Keep and for `git::StageAndNext`.
- **Ctrl+]**, matching brace, is in the full editor only, so the inline assistant's prompt keeps it.
- **Edit predictions:** Tab, Alt+Tab and Ctrl+Cmd+Right and Down stay Zed's.
- **Left as they are:** F12's global keys win in the agent panel. So its Ctrl+Home and End and
  Ctrl+Alt+Up, Down, PgUp and PgDn are F12's pane, dock and tab keys there.
- Ctrl+Cmd+S is Save all, not the agent's steer of a queued message.
- F7 replaces Zed's F7, step over while stopped, and F10 steps over too.
- **Keys whose action Zed lacks keep Zed's:** F3 tests, F4 NuGet, F10 build and Alt+Delete safe
  delete.
- Ctrl+N in the full editor is New file, so it no longer moves in the completion menu. The arrows
  do, and pickers keep Ctrl+N and Ctrl+P.
- **Code actions:** F12's keys for `Refactor this`, `Change signature`, `Extract method`,
  `Implement methods` and `Generate` all open the full menu, as no binding picks a kind
  (zed#14470).
- Last edit location stays in the file (zed#19731). Next error can skip gopls's unused-variable
  errors (zed#64790).

## The keys left to Zed

The data holds no row for 37 plain F12 keys, and the keymap binds none. Zed's own defaults do it:

- Zed's editor keys: Backspace, Enter, Esc, Tab, Shift+Tab, the arrows, Home, End and Page Up and
  Down, each alone and with Shift.
- Zed's completion and snippet keys: Enter and Tab accept an item, and Ctrl+Up and Ctrl+Down pick
  one. Tab and Shift+Tab move between a snippet's variables and a call's parameters.
- Tab accepts Zed's edit prediction.
- Alt+click adds a cursor, and Shift+Alt+drag selects a column (`multi_cursor_modifier`).
- Ctrl+D in the Terminal reaches the shell.

Seven of the rows share `tab`, so a neutral action for each would collide.

## How the mapping was made

Each action got a hand-made decision: status, Zed action, context and note. The inputs were:

- `F12.xml`, a user keymap in GoLand 2026.2's settings (76,333 bytes, md5
  `b00c6c90bdb6bbf84c3db578d1d93a0a`, identical under Rider 2026.2, with `parent` `$default`);
- `$default.xml` from `lib/intellij.platform.ide.impl.jar` of the GoLand 2026.2.3 backend;
- Zed's `assets/keymaps/default-macos.json`, as of Zed v1.26.2.

Nothing here was run on a Mac.
