package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// QueueRun follows a run of a queue on the two lists that chose it, over what the run prints. The
// run's bar and clock count stages, and each step's mark moves in its list. A run of the stages
// names them by their places in the queue, and Bridge.Observe sends this what it sends a RunScreen.
type QueueRun struct {
	run  RunScreen
	list pickList
}

// NewQueueRun returns the screen of what choice runs, under the choice's boxes.
func NewQueueRun(title string, theme Theme, choice Choice) QueueRun {
	var stages Group
	for _, r := range choice.list.rows {
		if r.number > 0 {
			stages.Steps = append(stages.Steps, r.name)
		}
	}
	return QueueRun{NewRunScreen(title, theme, stages), choice.list.filtered("")}
}

// Init starts the clock.
func (q QueueRun) Init() tea.Cmd {
	return q.run.Init()
}

// Update moves the lists by a run's events, and leaves the rest to the run's own screen, which
// sized the output to a pane this draws no longer.
func (q QueueRun) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if e, ok := msg.(step.Event); ok {
		q.list = q.list.follow(e)
	}
	var cmd tea.Cmd
	q.run, cmd = q.run.update(msg)
	switch msg.(type) {
	case tea.WindowSizeMsg, haltedMsg:
		q.run.output = q.run.output.fit(inside(stripSize(q.run.window)))
	}
	return q, cmd
}

// layout is the run's screen with the lists in place of its panes, and its output in the strip.
func (q QueueRun) layout() Layout {
	l := q.run.layout()
	output := Pane{Title: l.Panes[1].Title, Body: q.run.output.view(), Focused: true}
	l.Panes, l.Below = q.list.panes(), strip(q.run.theme, output, q.run.window)
	return l
}

// View draws the screen.
func (q QueueRun) View() tea.View {
	l := q.list.drawn(q.layout(), q.run.theme, q.run.window)
	return l.View(q.run.theme, q.run.window.Width, q.run.window.Height)
}
