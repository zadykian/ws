# The terminal UI

ws draws a full-screen display on a terminal, with Bubble Tea, Bubbles and Lip Gloss. Its code is
`internal/tui`. No command draws a screen yet: the package holds the parts, and `--plain` is in
place for the commands that will.

## Modes

A command draws a screen only where each of these holds:

- `--plain` is not given. Every command takes the flag, and `ws --plain mac wizard` is the form.
- Stdin and stderr are terminals.
- `TERM` is set and is not `dumb`.

Anywhere else ws prints lines, as before, byte for byte. That covers CI, a pipe and the unit on
ws, which reads the last line of a run. The wizard's cases in `.github/mac-wizard` that run under
`script -q` get a terminal, so each passes `--plain`.

The screen draws on stderr, where ws's lines go. Stdout and the exit codes stay what they were.
The mode is chosen in `cmd/ws/screen.go`.

## Colors

The theme uses the 16 ANSI colors alone, so Terminal.app's 256 colors work. It does not use
bright black, as some palettes hide it. `NO_COLOR`, set to any true value, turns the colors off,
and the frames and the focus stay visible by their shapes.

## Layout

A screen is a title, panes and a footer of buttons. It resizes with the window.

- Each pane gets its least size first, then the rest by weight.
- A window under 24 columns a pane stacks the panes.
- A window too short to stack shows the focused pane alone.
- A pane's lines are cut, never wrapped.

## Keys

The keyboard does everything, and the mouse is not captured. A button shows its shortcut and takes
focus with Tab.

| Key | Meaning |
| --- | --- |
| Tab, Shift-Tab | Move the focus. |
| Enter | Press the button in focus. |
| Up, k, Down, j | Move by a line. |
| PgUp, Ctrl-U, PgDn, Ctrl-D | Move by a page. |
| Home, g, End, G | Go to the top or the bottom. |
| q | Leave a screen. |
| Ctrl-C | Stop the run. |

Ctrl-C is a key in raw mode, and the screen turns it into the signal. The first one ends the
run's context, and the screen stays until the work has ended. The second one closes the screen,
and ws waits up to 5 seconds for the work's cleanup, such as the wizard's lock. A third one leaves
at once.

## Questions

A step asks through the same ports as in plain mode, `Prompt` and `Hidden`. The screen shows the
question in a field, and the step's goroutine waits for the answer. While a field is open, the
keys go to it alone. A hidden answer travels on a channel of its own, and enters no log, message
or error. A paste into a hidden field gives its first line, byte for byte, as the plain prompt
reads it, and Enter still answers. A step's lines arrive as messages and show in a pane.

## Commands

A screen owns the terminal, so a step's commands stay off it.

- A screen sets `Stdin` of `effect.OSMachine` to `effect.NoStdin`, and a command finds the end of
  its input at once.
- git and ssh ask at `/dev/tty`, past their input. With `Stdin` set, git gets
  `GIT_TERMINAL_PROMPT=0`, and ssh gets `SSH_ASKPASS_REQUIRE=force` with `SSH_ASKPASS=false`. So a
  clone fails where it would ask, as ssh does of an unknown host, and `--plain` asks. ws's ssh keeps
  `BatchMode` where no prompt is expected.
- A command keeps ws's controlling terminal. sudo keeps its ticket by terminal, so a child's own
  sudo, such as brew's, would ask again without one.
- `effect.Foreground` runs a command that talks to the maintainer. The screen suspends, and the
  command has the terminal until it ends. In plain mode it does what `Pass` does, but for a signal.
- A run that ends sends such a command SIGTERM, and kills it 2 seconds later. A kill alone would
  leave the terminal raw, or with no echo.

## sudo

sudo asks at `/dev/tty`, under a screen. `Bridge.Machine` gives a run's steps a machine whose sudo
can't ask, and takes the password in a hidden field:

- Its root commands run by `sudo -n --`, which fails where it would ask.
- Before each one, `sudo -n -v` looks for a ticket. A valid one asks nothing, and neither does a
  sudo that wants no password.
- With none, the field asks for sudo's password on this machine, and `sudo -S -v` checks it, up to
  three times. Any refusal asks again, as sudo's words differ by version, so a user outside
  sudoers sees them after the third. ws drops the password once sudo has checked it.
- A sudo that keeps no ticket, as with `timestamp_timeout=0`, ends the run past a right password,
  as `sudo -n` can run nothing. The error says to run with `--plain`, where sudo asks each time.
- A ticker runs `sudo -n -v` each minute while a ticket holds, so that a long step, or a child's
  own sudo, finds one.
- A stage that has sudo ask ahead of its lines calls `effect.Validate`. That is `sudo -v` in plain
  mode, and the same field under a screen.

ws runs as root on the server, with no sudo, so there it asks nothing.

## The terminal comes back

The terminal is restored after an error, a signal and a panic, a panic in a step's goroutine
included. A panic is an error with its stack and status 1. `TEA_TRACE` logs raw keys, so a run
unsets it, and a typed secret reaches no file. What a run leaves to do by hand prints after the
screen closes.

A signal ends a run with its command's status in plain mode, and a screen ends the steps through
the context in each case:

| Command | Plain mode, on a signal | Status |
| --- | --- | --- |
| `ws mac wizard` | Ends the context, and prints an error line. | 1 |
| `ws update` | Ends the context. | 128 plus the signal |
| `ws server apply`, `ws mac apply` | Dies at once, mid-step. | 128 plus the signal |

So 130 is SIGINT and 143 is SIGTERM wherever the status is 128 plus the signal.
