package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// QueueRun follows a run of a queue on the lists that chose it, over what the run prints. The
// run's bar and clock count stages, and each step's mark moves in its list. A run of the stages
// names them by their places in the queue, and Bridge.Observe sends this what it sends a RunScreen.
type QueueRun struct {
	run  RunScreen
	list queue
	at   map[string]widget.Cursor
}

// NewQueueRun returns the screen of what choice runs, under the choice's boxes.
func NewQueueRun(title string, theme widget.Theme, choice QueueChoice) QueueRun {
	var stages Group
	for _, s := range choice.left.list.stages {
		stages.Steps = append(stages.Steps, s.row.Name)
	}
	run := NewRunScreen(title, theme, stages)
	run.pane = func(window widget.Size) widget.Size { return inside(stripSize(window)) }
	return QueueRun{run, choice.left.list.filtered(""), choice.left.at}
}

// Init starts the clock.
func (q QueueRun) Init() tea.Cmd {
	return q.run.Init()
}

// Update moves the lists by a run's events, and leaves the rest to the run's own screen, which fits
// the output to the strip.
func (q QueueRun) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if e, ok := msg.(step.Event); ok {
		var side string
		var line int
		if q.list, side, line = q.list.follow(e); line >= 0 {
			q.list.side, q.at = side, withCursor(q.at, side, widget.Cursor(line))
		}
	}
	var cmd tea.Cmd
	q.run, cmd = q.run.update(msg)
	return q, cmd
}

// layout is the run's screen with the lists in place of its panes, and its output in the strip.
func (q QueueRun) layout() widget.Layout {
	l := q.run.layout()
	output := widget.Pane{Title: l.Panes[1].Title, Body: q.run.output.View(), Focused: true}
	l.Panes, l.Below = q.list.panes(), strip(q.run.theme, output, q.run.window)
	return l
}

// View draws the screen.
func (q QueueRun) View() tea.View {
	l := q.list.drawn(q.layout(), q.run.theme, q.run.window, q.at)
	return l.View(q.run.theme, q.run.window.Width, q.run.window.Height)
}
