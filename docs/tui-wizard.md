# The wizard's lists

[The list of steps](tui.md#the-list-of-steps) holds the keys these lists share. `ws client macos wizard`
opens them at a terminal, and `--plain` or no terminal keeps its lines. `--check` goes straight to
its run, on the run screen. `--update` runs ahead of any screen, on lines, as the new ws takes this
one's place and opens them itself. `ws server ubuntu wizard` opens one list, `ws`, where every stage
runs, by the same rules ([The server's wizard](../roles/server/ubuntu/stages/README.md)). Its `configure` button opens `ws
server ubuntu configure`'s forms, and its `s` reads ws on ws.

The wizard's stages are one queue, shown as two lists side by side: `Mac` and `ws`, by the machine
a stage runs on. A narrow window stacks them, and one too short for that keeps the list in focus.
The numbers run across both lists, in the order of the run. Stage 3 can stand in `ws` between two
stages of `Mac`, and the order, the locks and the count hold across the machines. A stage's own
steps stand under it, indented, with no number.

Tab walks the lists, then the buttons, which add `state` and `configure` to the list's, and
Shift-Tab walks back. Each list keeps its cursor. `a`, `n`, `i`, `p` and `/` act on both lists, and
`e` leaves for the values' form at any time, a scan under way included. `q` runs nothing.

Each stage's box follows its steps, and `[-]` says that some are on. A stage run in part stays
pending, and the detail says so. A stage that a chosen one waits on is locked as `[#]`, with its
steps, as that one needs it whole. Cable locks ws client macos apply too, whose steps it applies
itself.

ws's run with every step on and one of them ticked runs once more, whatever its last run left, as a
step can drift after it. A lock alone leaves a good last run as it stands.

Under the lists, a strip of a third of the window holds the detail, between 5 and 12 lines. A window
under 14 lines leaves it out.

## The scan

ws's list shows the roles by name until the stage that reaches ws has reported, and its title says
`ws: roles not scanned`. A check of the Mac can't read ws before ws is there. A report gives each
role its steps, in place of its name. A report that failed keeps the names, and the detail gives its
reason. A run then adds the steps as its events name them.

At the scan's end all that the scan found work in starts chosen, as the wizard left alone runs each
stage it reaches. So does all that it never reached, as a check stops short on a new Mac. A stage
found clean keeps the rows it never named off. A stage with work and no row on, as ws's run can be,
turns on whole.

The scan is `--check`, so it asks what a check asks, over the lists. sudo's password on ws carries
over to the run, and the forms drop it, as they may change the way to ws. The values Answers rehearses
go nowhere, and the run asks them again: `e` is the way to write them first. ws's prepare steps have
no row, as they run whatever the roles do and `--only` refuses their name.

## The state screen

`s` opens [the state screen](tui.md#the-state-screen) of the machine of the list in focus. It waits
for the scan's end, as both read that machine. The screen changes nothing, so `q` comes back to the
lists as they stood, with no new scan. `e` comes back to a new scan, which chooses the boxes again.
The run's screen has no `s`, as ssh and the lock are the run's.

The Mac's read is `ws client macos state`'s, and ws's is `ws server ubuntu state`'s, over ssh under
the wizard's lock. sudo's password carries from the scan. A scan that stopped before it leaves the
state screen to ask, and the run keeps the answer.

A read that fails shows in the header and ends nothing. A Mac with no `mac.conf` fails so, and so
does a ws at another version or with none. A first run's address, which the Host key stage asked,
goes with the scan's lock. ws's read then fails until a run sets the way up.

## The run

The run keeps both lists, each with the boxes it left with. A step's mark moves as it runs, and the
cursor follows it across the lists. The strip holds what the run prints, and a window too short for
the strip shows the lists alone. The bar and the count are of stages. The keys and the stop are
[the run screen's](tui.md#the-run-screen).

A stage run in part leaves the rest out as a run does: ws client macos apply's steps by the run's
filter, and ws's roles by `--only` to `ws server ubuntu apply`. A stop in ws's run ends the Mac's
side alone. The unit goes on on ws, the run says so, and the next wizard follows it. Where that unit
runs part of ws, the next wizard's other steps wait for its end. A signal ends with status 1, as in
[plain mode](tui.md#the-terminal-comes-back).
