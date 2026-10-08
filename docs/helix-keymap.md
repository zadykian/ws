# helix's F12 keymap

F12 is the keymap GoLand and Rider use here: `F12.xml`, 736 actions over IntelliJ's default
keymap, in Visual Studio's style. The `helix` role's `config.toml` binds the keys of each F12
action that helix has a command for, in normal and insert mode. These pages list those 109 actions
by topic, with helix's own key for each, then the 414 that helix has no command for. 107 of the 109
are bound: Parameter info and Open aren't, and [their tables](helix-keymap/actions.md) say why. The
other 213 actions of `F12.xml` have no keys: they take the default keymap's keys away. An action
that `F12.xml` doesn't name keeps the default keymap's keys, which aren't here.

The keys are F12's as the IDEs write them: a comma between the keys of a chord, a semicolon
between an action's keys. F12's Cmd keys are left out, as Cmd never reaches a program in a
terminal. Each has a Ctrl twin, but for those of Cut and Redo in the editor. Its numpad keys are
left out too in the tables of actions helix has, as helix has no names for them.

## The pages

- [Keys a terminal or tmux may not pass on](helix-keymap/terminals.md), and those the JetBrains
  terminal loses.
- [helix's keys that F12 takes](helix-keymap/taken.md), with helix's other key for each.
- [The actions helix has](helix-keymap/actions.md), by topic, with helix's own key for each.
- The actions helix lacks: [A to J](helix-keymap/lacks-a-j.md) and
  [L to Z](helix-keymap/lacks-l-z.md).

## In insert mode

A key does in insert mode what it does in normal mode, but:

- the completion keys work in insert mode alone, as helix drops completions outside it;
- the keys that select leave insert mode first, since text typed in insert mode goes before the
  selection rather than over it;
- Copy and Cut do nothing. Insert mode's selection is the character after the cursor, which Cut
  would take, where the IDEs copy or cut the line when nothing is selected;
- Surround with acts on that character, and Delete stays helix's, which deletes it;
- Paste inserts at the cursor through `insert_register`, rather than before the selection;
- Move line goes back to insert mode.

In insert mode, a Ctrl or Alt key with a character that nothing is bound to types that character,
as helix does with any such key. So does a chord's second key that the chord lacks, after the
first key's character.
