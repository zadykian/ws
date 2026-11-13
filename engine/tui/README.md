# The terminal UI

ws draws a full-screen display on a terminal, with Bubble Tea, Bubbles and Lip Gloss. Its code is
`engine/tui`. `ws server ubuntu apply` and `ws client macos apply` draw [the run
screen](#the-run-screen), `ws server ubuntu state` and `ws client macos state` draw [the state
screen](#the-state-screen), and `ws client macos configure` and `ws server ubuntu configure` draw
[the Configure screen](internal/configure/README.md). `ws client macos wizard` draws [two lists](internal/queue/README.md),
which lead to the other two, and `ws server ubuntu wizard` one. The rest, such as `ws update` and
`ws version`, print lines.

## Modes

A command draws a screen only where each of these holds:

- `--plain` is not given. Every command takes the flag, and `ws --plain client macos wizard` is the
  form.
- `--json` is not given, where a command takes it.
- Stdin and stderr are terminals. So is stdout for `ws server ubuntu state` and `ws client macos
  state`, as their report prints there.
- `TERM` is set and is not `dumb`.

Anywhere else ws prints lines, as before, byte for byte. That covers CI, a pipe and the unit on ws,
which reads the last line of a run. The wizard's cases in `.github/mac-wizard` that run under
`script -q` get a terminal, so each passes `--plain`. CI's `playbook` job runs `ws server ubuntu
apply --check` under `script` to draw the screen ([The `playbook` job](../../.github/workflows/README.md#the-playbook-job)),
and a case of the `mac` job does so with `ws client macos apply --check` ([The
Mac](../../.github/workflows/README.md#the-mac)).

The screen draws on stderr, where ws's lines go. Stdout and the exit codes stay what they were.
The mode is chosen in `engine/screen.go`.

## Colors

The theme uses the 16 ANSI colors alone, so Terminal.app's 256 colors work. It does not use
bright black, as some palettes hide it. `NO_COLOR`, set to any value but nothing, turns the colors off,
and the frames and the focus stay visible by their shapes.

## Layout

A screen is a title, panes and a footer of buttons, and a header may stand above the panes. It
resizes with the window.

- Each pane gets its least size first, then the rest by weight.
- A window under 24 columns a pane stacks the panes.
- A window too short to stack shows the focused pane alone.
- A pane's lines are cut, but for a run's window and the state screen's detail, which wrap.

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
| Ctrl-S | Save the form in sight. |
| Esc | Leave the Configure screen. |
| Ctrl-N | Show the next form. |
| Ctrl-C | Stop the run. |

Ctrl-C is a key in raw mode, and the screen turns it into the signal. On a run whose work took
`Bridge.Options`, the first one asks ([Stopping a run](#stopping-a-run)). On any other, it ends the
run's context, and the screen stays until the work has ended. The next one closes the screen, and
ws waits up to 5 seconds for the work's cleanup, such as the wizard's lock. Another one leaves at
once.

## Questions

A step asks through the same ports as in plain mode, `Prompt` and `Hidden`, and `effect.Choosing`
for a few answers to pick from. The screen shows the question in a field, and the step's goroutine
waits for the answer. While a field is open, the keys go to it alone. A hidden answer travels on a
channel of its own, and enters no log, message or error. A step's lines arrive as messages and show
in a pane.

A short window gives up the buttons, whose keys still work, then the frame, then the question's
words. The input stays, down to a window of one line.

## Fields

A question draws a text field, a password field or a choice. `effect.Choose` asks a choice of a
`Prompt`. The bridge draws it as a choice field. Any other `Prompt` gets a line that must name an
option, with the first, or the default, as its answer when empty.

- **Text:** takes a line. `Valid` runs on each edit, and its refusal shows under the input until an
  edit mends it. A paste's line ends become blanks.
- **Password:** shows nothing of what is typed. A paste gives its first line, byte for byte, as the
  plain prompt reads it, and Enter still answers.
- **Choice:** shows its options in a row, and `←` and `→`, `h` and `l`, or Space move round them.
  A window too narrow for the row shows the chosen option alone.

## The run screen

`run.Screen` follows a run. It takes the run's `Group`s, so it lists every step before the first
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
- It wraps a long line. `\r` keeps the last state of a progress bar, and escapes and control
  characters go.
- Under `--check`, each step's changes show in it, a file's diff in color.

The window goes with the screen, and these stay:

- Under `--check`, each step's changes print on the normal screen, a file's diff in full. Of a file
  of the user's, as `~/.gitconfig`, the screen, the log and a report hold ws's lines alone, as the
  lines beside them may hold a secret. `--plain` prints the whole diff.
- A warning prints there too, on any run, as the step that gave it passes.
- A failed step prints the last 30 lines of its output, then the error.
- The summary of `ws server ubuntu apply` is the last line. A passing `ws client macos apply` prints
  nothing more.
- What stays by hand prints after the screen.
- `--plain` keeps every line of a run. What `ws client macos apply` prints on stdout, the commands
  that make the Mac ws's peer, waits for the screen's end, and prints after a run that passed.

What stays by hand holds the lines that the maintainer acts on past a run: ws's WireGuard key, and
what a step leaves to do. A step says one through `effect.SayLasting`, which a plain run prints in
place.

### The log

A run keeps its whole output in `$XDG_STATE_HOME/ws/`, or `~/.local/state/ws/`, in a file named for
the command and the time, as `ws-server-ubuntu-apply-20261105T120000Z.log`. The directory is mode
0700 and the file 0600. The file holds the lines the window shows, cleaned as the window cleans
them, and a failed or skipped step's name. It holds too what a plain run prints apart from them: the
summary, the stdout text of a run, and what stays by hand. A hidden answer never reaches it, and
neither does a foreground command's own output, which holds the terminal.

The newest 20 logs stay. A run whose log can't be made goes on without one. The path prints on the
normal screen after a failure, a stop, a warning or a `--check` with changes.

### Stopping a run

A step cut halfway can leave a machine between two states, so a run stops between its steps. That
holds for a run whose steps took their options from `Bridge.Options`.

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
chosen, `c` checks them and `q` leaves. The filter is the state screen's: it takes the same keys
and a paste.

`ws server ubuntu apply` and `ws client macos apply` open it at a terminal when they get no
selection: no `--only` and no `--check`. The scan is a check of every step, run beside the list.
`enter` then runs the steps chosen on the run screen, and `c` checks them there. Both are off with
nothing chosen. A list left with `q` runs nothing and exits 0. A cut that follows a choice wins:
nothing runs, and the status is that of the cut. The server's choice is the `--only` entries, `ROLE`
or `ROLE/STEP`. The Mac's is a set of step names. `ssh again` has no row, as a check has no second
plan: it runs, unless `ssh` is left out with changes. With `--plain` or no terminal the command runs
at once, as before, and so does the server's with `--json`.

A role's box follows its steps, and `[-]` says that some are on. A chosen step locks the step it
waits on as `[#]`, and the pane beside the list says why. The order of roles is not modeled, so
`!` in the status warns of changes left out in an earlier role.

[The wizard's lists](internal/queue/README.md) set two such lists side by side.

## The state screen

`ws server ubuntu state`, `ws client macos state` and [the wizard's
`s`](internal/queue/README.md#the-state-screen) draw what a machine holds of what ws manages. The
screen reads and changes nothing.

The header holds the tabs, each with its count: Packages, Services, Files, Conflicts and Pending.
The tabs wrap to a new line in a narrow window. Under a filter a count reads `shown/all`. A check
that failed stands in Conflicts, and a step with an unread report stands in Pending. The list
shows a row for each item, and the detail pane beside it shows the selected row's wanted and found
state, and a file's diff in color. The diff's marks say the same under `NO_COLOR`. A long line of
the detail wraps. A failed read keeps the old rows and shows its first line in the header.

Tab walks the tabs, then the buttons, and wraps to the first tab. The selection stays on its row,
by step and name, across a read and a filter.

| Key | Meaning |
| --- | --- |
| Tab, Shift-Tab | Move to the next or the previous tab, then the buttons. |
| Up, k, Down, j | Move the selection, or the detail's lines. |
| PgUp, PgDn, Home, End | Move by a page, or to either end. |
| Enter, Right, l | Move the keys to the detail pane. |
| Esc, Left, h | Move them back to the list. |
| `/` | Filter the rows by name. |
| `r` | Read the machine again. |
| q | Leave. |

The filter keeps the rows whose name holds its words, in any case. An open filter takes the keys
alone. Enter keeps the words and closes it, and Esc drops them. Esc on the list drops a
filter that stands.

The q key ends a read that runs, through its context, and Ctrl-C acts as above.

A read takes the machine a run takes ([Commands](#commands)). What its commands print shows nowhere,
as this screen has no window. sudo's password comes through a question over the screen
([sudo](#sudo)), and a step that gets no ticket stands in Conflicts with sudo's words.

## Commands

A screen owns the terminal, so a step's commands stay off it.

- A screen sets `Stdin` of `local.OSMachine` to `local.NoStdin`, and a command finds the end of
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
  own standard input and error. The line a step said last prints above it, once, as the pane is
  out of sight. The screen draws again after it, even after a failure. A screen that can't leave
  runs nothing.
- The wizard's steps hold it as `F` beside `M`. It runs gh's login, and `claude auth login` over
  `ssh -t`. With no `F`, a step does `M.Pass`.
- A run that ends sends such a command SIGTERM, and kills it 2 seconds later. A kill alone would
  leave the terminal raw, or with no echo.
- A second Ctrl-C waits for such a command's end, and then closes the screen. The program's loop
  moves the terminal, as a move past the screen's end would leave it raw.

## sudo

sudo asks at `/dev/tty`, under a screen. `Bridge.Machine` gives the steps of a run or a read a
machine whose sudo can't ask, and takes the password in a hidden field. `Run` wraps the screen's
`Config.Machine` once, as each wrapping would hold sudo's ticket until the screen ends. Every work
takes it from `Bridge.Effects`, with `Bridge.Foreground` and the run's stop:

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
- A stage that has sudo ask ahead of its lines calls `local.Validate`. That is `sudo -v` in plain
  mode, and the same field under a screen.

ws runs as root on the server, with no sudo, so there it asks nothing. `ws client macos apply` and
`ws client macos state` ask at their first root command, so one that needs none asks nothing.

## The terminal comes back

The terminal is restored after an error, a signal and a panic, a panic in a step's goroutine
included. A panic is an error with its stack and status 1. `TEA_TRACE` logs raw keys, so a run
unsets it, and a typed secret reaches no file. What a run leaves to do by hand prints after the
screen closes.

A signal ends a run with its command's status in plain mode, and a screen ends the steps through
the context in each case:

| Command | Plain mode, on a signal | Status |
| --- | --- | --- |
| `ws client macos wizard` | Ends the context, and prints an error line. | 1 |
| `ws update` | Ends the context. | 128 plus the signal |
| `ws server ubuntu apply`, `ws client macos apply` | Dies at once, mid-step. | 128 plus the signal |

So 130 is SIGINT and 143 is SIGTERM wherever the status is 128 plus the signal.
