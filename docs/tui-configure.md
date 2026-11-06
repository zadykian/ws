# The Configure screen

[The terminal UI](tui.md) holds what every screen shares, such as the modes and the
fields.

`ws mac configure` edits `mac.conf` on a form, with a field for each key: the key, its value and
its help. A key shows where the wizard's Answers stage would ask it, by `WS_TARGET`, and an edit
moves the fields at once, as a `WS_REMOTE` brings its port. A key that the stage never asks, such
as `WS_USER`, stays the file's. The screen draws or stops: with no terminal, or under `--plain`, it
refuses, and the file stays open to an editor. A line that `ws mac apply` would refuse stops it
before the screen, as a form has no field for it. Keys that disagree open, as the form mends them.

Tab, Enter and Down go to the next field, Shift-Tab and Up to the one before, and past either end
to the buttons. A key that needs a value shows the wizard's default in brackets, and a save writes
it. An optional key's default stays in its help as an offer, and an empty field is an empty value,
as `none` is. Esc with edits unsaved asks for a second Esc in the title's place, at any width, and
Ctrl-C drops them with status 130.
Ctrl-N walks the forms where a screen has several.

A save checks the fields that show, and the first it refuses takes the focus. Its changes then go
in as one, through the wizard's writer, which checks the whole file and leaves a link
([The Mac's wizard](mac-wizard.md)). A key that the edits left with no place goes empty, the cable's
with a `WS_TARGET` other than `lan`, as conf would refuse it. The wizard asks no key with a line
again, even an empty one. Under the form, the screen shows the keys a save emptied, or why it
failed, which keeps the edits. A short window keeps the field's input and the end of those words.
At one row it keeps the input alone, and the status still counts the edits.
