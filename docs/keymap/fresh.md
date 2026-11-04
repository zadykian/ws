# Fresh's F12 keymap

The adapter in [`internal/editor/fresh`](../../internal/editor/fresh/) turns the neutral keymap
into `keybinding_maps.f12` in Fresh's `config.json` ([the config](../fresh.md#the-config)). F12's
keys are described in [where F12 comes from](../keymap.md#where-f12-comes-from). The page holds
counts, rules and limits. The targets and the reasons are in
[`actions/`](../../internal/editor/fresh/actions/), the one place for them.

## The counts

- 186 actions in 14 topics: 104 take a Fresh action, and 82 take a reason that Fresh lacks one.
- The render has 199 bindings: 198 in the map `f12`, which inherits `default`, and ctrl+h outside it.
- They serve 95 distinct actions and 30 chords. By `when`, `normal` has 159, `file_explorer` 13
  and `terminal` 27. By kind, 175 are builtin, 23 plugin and 1 with a char.
- Of 153 keys tested, 147 land on their action through tmux 3.6a, and 151 directly.

## Contexts

- Global, Text and Code render to `normal` alone. A `global` entry loses to the default map's
  `normal`. Fresh's `normal` passes only its UI actions on to the explorer and the panels
  ([Other limits](#other-limits)).
- Tree and Panel render to `file_explorer`, and Terminal to `terminal`.
- Another scope has no context, and a target there fails the render.

## Strokes

A stroke is spelt as Fresh spells it, and a chord that arrives in more than one form gets a binding
for each.

- A stroke with Shift and a sign gets two forms: tmux sends the sign without Shift, and a direct
  terminal sends the key with Shift.
- Terminals rewrite some chords, and a test keeps them apart from the bound keys:
  - ctrl+alt+4 arrives as ctrl+alt+\\, and ctrl+alt+/ as ctrl+alt+shift+-;
  - ctrl+k ctrl+/ arrives as ctrl+k ctrl+shift+-;
  - ctrl+shift+space arrives as ctrl+space, and alt+shift+space as alt+space.
- ctrl+backspace and ctrl+shift+backspace take no binding. Through tmux both arrive as ctrl+h.
- ctrl+h is bound outside the map. The default map binds both ctrl+h and ctrl+backspace, and a
  binding in a map warns at each start.
- With no tmux, Fresh's default for ctrl+backspace deletes a word, as F12's does.

## The shell keys

24 keys render as `noop` in `terminal`, so the shell in Fresh's terminal gets them. A `noop` there
ends Fresh's lookup, so its default map cannot take the key:

- F12 frees nine: ctrl+q, ctrl+comma, ctrl+alt+], ctrl+alt+[, ctrl+alt+s, alt+x, alt+v, alt+left
  and alt+right.
- The default map fires 15 more over the shell:
  - alt+e, alt+f, alt+g, alt+h and alt+s open a menu, and alt+w closes the tab;
  - ctrl+p, ctrl+; and ctrl+' open Quick Open;
  - alt+r, alt+j, alt+/, alt+[, alt+] and alt+` do the editor's work.
- Fresh's own ways out stay: ctrl+space and ctrl+] (`terminal_escape`), F9 (keyboard capture) and
  alt+o (the dock's focus).
- Known by source alone, so the trial on the Mac checks them
  ([the keys](../fresh.md#the-keys)).

## Left Option

Of F12's 266 rows, 105 use Alt, and 59 of those are mapped. On a Mac, left Option sends Alt
([Ghostty's settings](../mac-ghostty.md#ghosttys-settings)). Right Option keeps composing.

## What Fresh lacks

The maintainer accepted these. 82 of 186 actions take a reason that Fresh lacks the action:

| Reason | Actions |
| --- | --- |
| No debugger | 15 |
| One fold action, `toggle_fold` | 11 |
| No syntax-tree selection | 7 |
| The terminal sets the font size | 6 |
| No git panel | 6 |
| No git command | 6 |
| No task runner | 4 |
| Others, one to three each | 27 |

## Six keys that fail through tmux

| Key | Action | What happens |
| --- | --- | --- |
| ctrl+cmd+s | `file.save-all` | Lands on `open_settings` |
| ctrl+alt+2 | `problem.list` | Lands on `lsp_completion` |
| ctrl+backspace | `edit.delete-word-start` | Arrives as ctrl+h, so lands on `replace` |
| ctrl+shift+backspace | `nav.last-change` | Arrives as ctrl+h, so lands on `replace` |
| ctrl+f3 | `search.next-word` | Types `13;~5`, as tmux splits it |
| ctrl+shift+f3 | `search.previous-word` | Types `13;6~`, as tmux splits it |

## Other limits

- 17 keys with no Fresh action keep Fresh's default, such as ctrl+b, `toggle_file_explorer`, and
  alt+f, `menu_open`, in the editor.
- Fresh's `normal` passes on only its UI actions: menus, quick open, splits, tabs, the terminal's
  and the explorer's toggles. With focus in the explorer or a panel, 17 of Global's 37 mapped
  actions do nothing, such as save, save-all, search, replace and revert. F6 there opens Fresh's
  remote menu, not the git log.
- The prefix reaches the editor pressed twice: ctrl+b by default, ctrl+q in cld's sessions. So
  Fresh's palette on ctrl+q needs it twice in cld's sessions, and ctrl+b is lost in tmux.
- A folder with a project marker opens Restricted, with every server off
  ([Trust each project](../fresh.md#trust-each-project)).
- `.yml` playbooks stay YAML, with no Ansible server.
