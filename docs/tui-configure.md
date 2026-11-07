# The Configure screen

[The terminal UI](tui.md) holds what every screen shares, such as the modes and the
fields.

`ws mac configure` edits `mac.conf` on a form, ws's secrets file on a second, and ws's host_vars
on a third. A form has a field for each key: the key, its value and its help. A key shows where the
wizard's Answers stage would ask it, by `WS_TARGET`, and an edit moves the fields at once, as a
`WS_REMOTE` brings its port. A key that the stage never asks, such as `WS_USER`, stays the file's.
The screen draws or stops: with no terminal, or under `--plain`, it refuses, and the file stays
open to an editor. A line that `ws mac apply` would refuse stops it before the screen, as a form
has no field for it. Keys that disagree open, as the form mends them.

Tab, Enter and Down go to the next field, Shift-Tab and Up to the one before, and past either end
to the buttons. A key that needs a value shows the wizard's default in brackets, and a save writes
it. An optional key's default stays in its help as an offer, and an empty field is an empty value,
as `none` is. Esc with edits unsaved asks for a second Esc in the title's place, at any width, and
Ctrl-C drops them with status 130.
Ctrl-N walks the forms: `mac.conf`, the secrets, then host_vars.

The secrets form has a field for the tokens and the VPN password, and for each env and git entry
the file holds. A field shows no value: its help says whether the entry is set, and a new value
replaces it. A save runs `ws server secrets set` on ws for each field typed in, the value on ssh's
stdin, which keeps the other entries and a git entry's user name ([The secrets](mac-secrets.md)).
A save that failed part way set the first fields. A field can't clear an entry, and the form adds
no env or git entry, nor a profile file: `ws mac secrets unset` and `set` do. With no credential on
ws, the read fails and says to run `ws server secrets init`.

A save checks the fields that show, and the first it refuses takes the focus. Its changes then go
in as one, through the wizard's writer, which checks the whole file and leaves a link
([The Mac's wizard](mac-wizard.md)). A key that the edits left with no place goes empty, the cable's
with a `WS_TARGET` other than `lan`, as conf would refuse it. The wizard asks no key with a line
again, even an empty one. Under the form, the screen shows the keys a save emptied, or why it
failed, which keeps the edits. A short window keeps the field's input and the end of those words.
At one row it keeps the input alone, and the status still counts the edits.

The secrets form reads the entries by `ws server secrets keys`, and the host_vars form
`/etc/ws/host_vars.yml`, over ssh once each shows, so `mac.conf` draws with ws out of reach. A read
that fails shows why, and Ctrl-S tries again. The first read takes the wizard's lock, and asks
sudo's password where ws wants one. Both last until the screen ends, so a save asks nothing, and a
wizard's run shows there as the lock's holder. The host_vars form shows every key with a line, and a
table stays the file's. A list goes as its items with commas, and an emptied
field takes the key's line out, which gives its default back. A save runs `ws server values set` on
ws for each key that changed, in one login, so ws's own check refuses a value. A save that failed
part way wrote the first keys, and a second writes them again as they are.

Only the file changes: the next `ws server apply` takes it. The wizard's next run names `ws_user`
and, on a cloud VM, `security_outbound` again ([The Mac's wizard](mac-wizard.md)).
