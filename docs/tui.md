# The terminal UI

ws draws a full-screen display on a terminal, with Bubble Tea, Bubbles and Lip Gloss. Its code is
`internal/tui`. `ws server apply` and `ws mac apply` draw [the run screen](#the-run-screen). The other
commands print lines for now.

## Modes

A command draws a screen only where each of these holds:

- `--plain` is not given. Every command takes the flag, and `ws --plain mac wizard` is the form.
- Stdin and stderr are terminals.
- `TERM` is set and is not `dumb`.

Anywhere else ws prints lines, as before, byte for byte. That covers CI, a pipe and the unit on
ws, which reads the last line of a run. The wizard's cases in `.github/mac-wizard` that run under
`script -q` get a terminal, so each passes `--plain`. CI's `playbook` job runs `ws server apply
--check` under `script` to draw the screen ([The `playbook` job](checks.md#the-playbook-job)), and
a case of the `mac` job does so with `ws mac apply --check` ([The Mac](checks.md#the-mac)).

The screen draws on stderr, where ws's lines go. Stdout and the exit codes stay what they were.
The mode is chosen in `cmd/ws/screen.go`.

## Colors

The theme uses the 16 ANSI colors alone, so Terminal.app's 256 colors work. It does not use
bright black, as some palettes hide it. `NO_COLOR`, set to any value but nothing, turns the colors off,
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

Ctrl-C is a key in raw mode, and the screen turns it into the signal. On a run that reads
`Bridge.Stop`, the first one asks ([Stopping a run](#stopping-a-run)). On any other, it ends the
run's context, and the screen stays until the work has ended. The next one closes the screen, and
ws waits up to 5 seconds for the work's cleanup, such as the wizard's lock. Another one leaves at
once.

## Questions

A step asks through the same ports as in plain mode, `Prompt` and `Hidden`. The screen shows the
question in a field, and the step's goroutine waits for the answer. While a field is open, the
keys go to it alone. A hidden answer travels on a channel of its own, and enters no log, message
or error. A paste into a hidden field gives its first line, byte for byte, as the plain prompt
reads it, and Enter still answers. A step's lines arrive as messages and show in a pane.

## The run screen

`RunScreen` follows a run. It takes the run's `Group`s, so it lists every step before the first
one starts. `Bridge.Observe` is the run's `step.Observer`, and sends it the run's events.

The title's right side holds the steps ended, a bar of them and the elapsed time. One pane lists
the steps by role, and keeps the current one in sight:

| Mark | Step |
| --- | --- |
| `▸` | The current one. |
| `·` | Waiting. |
| `✓` | Done, with nothing to change. |
| `●` | Done with changes, or with changes to make under `--check`. |
| `○` | Left out by `--only ROLE/STEP`: planned, and neither applied nor shown under `--check`. A plan that fails warns, and the run goes on. |
| `✗` | Failed. |
| `–` | Skipped. |

The other pane is a window on what the run prints, under the current step's name.

- It holds the lines a plain run prints, and what the step's commands print.
- It follows the end of the output. Up, PgUp and Home move it back, and End brings it to the end.
- It keeps the last 10,000 lines of a run.
- It wraps a long line, unlike the other panes. `\r` keeps the last state of a progress bar, and
  escapes and control characters go.
- Under `--check`, each step's changes show in it, a file's diff in color.

The window goes with the screen, and these stay:

- Under `--check`, each step's changes print on the normal screen, a file's diff in full.
- A warning prints there too, on any run, as the step that gave it passes.
- A failed step prints the last 30 lines of its output, then the error.
- The summary of `ws server apply` is the last line. A passing `ws mac apply` prints nothing more.
- What stays by hand prints after the screen.
- `--plain` keeps every line of a run. What `ws mac apply` prints on stdout, the commands that make
  the Mac ws's peer, waits for the screen's end.

What stays by hand holds the lines that the maintainer acts on past a run: ws's WireGuard key, and
what a step leaves to do. A step says one through `effect.SayLasting`, which a plain run prints in
place.

### The log

A run keeps its whole output in `$XDG_STATE_HOME/ws/`, or `~/.local/state/ws/`, in a file named for
the command and the time, as `ws-server-apply-20261105T120000Z.log`. The directory is mode 0700
and the file 0600. The file holds the lines the window shows, cleaned as the window cleans them,
and a failed or skipped step's name. A hidden answer never reaches it.

The newest 20 logs stay. A run whose log can't be made goes on without one. The path prints on the
normal screen after a failure, a stop, a warning or a `--check` with changes.

### Stopping a run

A step cut halfway can leave a machine between two states, so a run stops between its steps. That
holds for a run whose `step.Options.Stop` is `Bridge.Stop`, with `Config.Halts` set.

1. Ctrl-C, or the stop button, opens a box. Esc goes on, and the run has not paused for it.
2. Enter stops the run once its current step ends. The steps left end skipped, and the window stays
   readable, with `Stopping after` in its title.
3. Ctrl-C from there ends the context, as it does from the box.

A question asks nothing first: its step waits on the answer, so Ctrl-C ends the context. A signal
asks nothing either, as nobody may be there to answer. A run stopped past its current step ends
with the status of Ctrl-C, unless that step was the last.

## The list of steps

The list shows a run's roles and their steps, each with a box. A scan ticks the steps with changes.
Past it, `space` turns a box. `a`, `n`, `i` and `p` choose all, none, the inverse and the steps
with changes, among those a filter shows. `/` filters, and `Esc` drops it. `enter` runs the steps
chosen, `c` checks them and `q` leaves.

`ws server apply` and `ws mac apply` open it at a terminal when they get no selection: no `--only`
on the server, no STEP on the Mac, and no `--check` on either. The scan is a check of every step, run
beside the list. `enter` then runs the steps chosen on the run screen, and `c` checks them there. A
list left with `q`, or with nothing chosen, runs nothing and exits 0. A cut that follows a choice
wins: nothing runs, and the status is that of the cut. The server's choice is the `--only` entries,
`ROLE` or `ROLE/STEP`. The Mac's is a set of step names. With `--plain` or no terminal the command
runs at once, as before.

A role's box follows its steps, and `[-]` says that some are on. A chosen step locks the step it
waits on as `[#]`, and the pane beside the list says why. The order of roles is not modeled, so
`!` in the status warns of changes left out in an earlier role.

## Commands

A screen owns the terminal, so a step's commands stay off it.

- A screen sets `Stdin` of `effect.OSMachine` to `effect.NoStdin`, and a command finds the end of
  its input at once.
- `Bridge.Machine` sets `Stderr` to a pipe that sends what the commands print to the screen, a
  whole line at a time. What `Pass` prints then reaches the window, and leaves the terminal alone.
- git and ssh ask at `/dev/tty`, past their input. With `Stdin` set, git gets
  `GIT_TERMINAL_PROMPT=0`, and ssh gets `SSH_ASKPASS_REQUIRE=force` with `SSH_ASKPASS=false`. So a
  clone fails where it would ask, as ssh does of an unknown host, and `--plain` asks. ws's ssh keeps
  `BatchMode` where no prompt is expected.
- A run that ends sends a step's command SIGTERM, and kills it 3 seconds later. No Ctrl-C reaches
  it through a raw terminal, and a kill of `runuser` would leave its child. Past those 3 seconds ws
  no longer waits for what a command left running, under a screen alone.
- A command keeps ws's controlling terminal. sudo keeps its ticket by terminal, so a child's own
  sudo, such as brew's, would ask again without one.
- `effect.Foreground` runs a command that talks to the maintainer. The screen suspends, and the
  command has the terminal until it ends. In plain mode it does what `Pass` does, but for a signal.
- `Bridge.Foreground` runs such a command on a machine. A root command's ticket comes first, as
  only the screen can ask for it. Then the screen releases the terminal, and the command has ws's
  own standard input and error. The screen draws again after it, even after a failure. A screen
  that can't leave runs nothing.
- The wizard's steps hold it as `F` beside `M`. It runs `ssh-keygen` and `ssh-add` for the Mac's
  key, gh's login, and `claude auth login` over `ssh -t`. With no `F`, a step does `M.Pass`.
- A run that ends sends such a command SIGTERM, and kills it 2 seconds later. A kill alone would
  leave the terminal raw, or with no echo.
- A second Ctrl-C waits for such a command's end, and then closes the screen. The program's loop
  moves the terminal, as a move past the screen's end would leave it raw.

## sudo

sudo asks at `/dev/tty`, under a screen. `Bridge.Machine` gives a run's steps a machine whose sudo
can't ask, and takes the password in a hidden field:

- Its root commands run by `sudo -n --`, which fails where it would ask.
- Before each one, `sudo -n -- true` looks for a ticket. A valid one asks nothing, and neither does
  a sudo that wants no password. `sudo -n -v` would ask of a NOPASSWD user in a Mac's admin group.
- With none, the field asks for sudo's password on this machine, and `sudo -S -v` checks it, up to
  three times. Any refusal asks again, as sudo's words differ by version, so a user outside
  sudoers sees them after the third. ws drops the password once sudo has checked it.
- A sudo that keeps no ticket, as with `timestamp_timeout=0`, ends the run past a right password,
  as `sudo -n` can run nothing. The error says to run with `--plain`, where sudo asks each time.
- A ticker runs `sudo -n -v` each minute while a ticket holds, so that a long step, or a child's
  own sudo, finds one.
- A stage that has sudo ask ahead of its lines calls `effect.Validate`. That is `sudo -v` in plain
  mode, and the same field under a screen.

ws runs as root on the server, with no sudo, so there it asks nothing. `ws mac apply` asks at its
first root command, so a run that needs none asks nothing.

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
