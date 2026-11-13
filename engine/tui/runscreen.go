package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

const (
	// barLeast is the columns under which a bar says nothing, and barMost is as wide as one needs.
	barLeast = 4
	barMost  = 24
	// statusGaps is the columns around the title and the status's parts.
	statusGaps = 5
)

// The places of the run screen's buttons.
const (
	pageUpPlace = iota
	pageDownPlace
	bottomPlace
	stopButtonPlace
)

// stopNowButton is Ctrl-C's button past a confirmed stop, when the press ends ctx.
var stopNowButton = key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "stop now"))

// tickMsg is the clock, for the elapsed time. It carries the time, so that Update reads no clock.
type tickMsg struct{ at time.Time }

// RunScreen follows a run: its steps by role with the current one marked, a window on what the run
// prints, a bar of the steps ended and the elapsed time. Bridge.Observe and effect.Out send it what
// it shows. The keys scroll the window, as the steps list follows the run by itself.
type RunScreen struct {
	title   string
	theme   widget.Theme
	steps   list.StepList
	output  widget.OutputPane
	footer  widget.Footer
	window  widget.Size
	started time.Time
	now     time.Time
	// halted says that the run ends with its current step. No box covers the window for it, as a
	// step can run for long and its output stays worth reading.
	halted bool
	// pane, where set, sizes the output in a window, for a host that draws the output elsewhere.
	pane func(window widget.Size) widget.Size
}

// NewRunScreen returns the screen of a run of groups, which name every step before the run starts.
func NewRunScreen(title string, theme widget.Theme, groups ...list.Group) RunScreen {
	return RunScreen{
		title:  title,
		theme:  theme,
		steps:  list.NewStepList(groups),
		output: widget.OutputPane{Theme: theme},
		footer: widget.NewFooter(widget.Keys.PageUp, widget.Keys.PageDown, widget.Keys.Bottom, widget.Keys.Interrupt),
	}
}

// Init starts the clock.
func (r RunScreen) Init() tea.Cmd {
	return func() tea.Msg { return tickMsg{time.Now()} }
}

// Update takes the run's events and lines, the clock, the window's size and the keys.
func (r RunScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r.Updated(msg)
}

// Updated is Update by its own type, for a screen that draws a run its way.
func (r RunScreen) Updated(msg tea.Msg) (RunScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case step.Event:
		r.steps = r.steps.Take(msg)
	case widget.Line:
		r.output = r.output.Add(msg)
	case tickMsg:
		if r.started.IsZero() {
			r.started = msg.at
		}
		r.now = msg.at
		return r, tea.Tick(time.Second, func(at time.Time) tea.Msg { return tickMsg{at} })
	case HaltedMsg:
		r.halted = true
		r.footer = widget.NewFooter(widget.Keys.PageUp, widget.Keys.PageDown, widget.Keys.Bottom, stopNowButton)
		return r.fitted(), nil
	case tea.WindowSizeMsg:
		r.window = widget.Size{Width: msg.Width, Height: msg.Height}
		return r.fitted(), nil
	case tea.KeyPressMsg:
		return r.pressed(msg)
	}
	return r, nil
}

// fitted sizes the output to its pane, past a change of the window or of the footer's rows.
func (r RunScreen) fitted() RunScreen {
	if r.pane != nil {
		r.output = r.output.Fit(r.pane(r.window))
		return r
	}
	r.output = r.output.Fit(r.Layout().PaneSizes(r.window.Width, r.window.Height)[1])
	return r
}

// OutputIn returns the screen with its output sized by pane in a window, for a host that draws the
// output elsewhere.
func (r RunScreen) OutputIn(pane func(window widget.Size) widget.Size) RunScreen {
	r.pane = pane
	return r
}

// Output is the window on what the run prints.
func (r RunScreen) Output() widget.OutputPane {
	return r.output
}

// Theme is the theme that styles the screen.
func (r RunScreen) Theme() widget.Theme {
	return r.theme
}

// Window is the size of the terminal, as the screen last heard it.
func (r RunScreen) Window() widget.Size {
	return r.window
}

// pressed moves the focus or the window, or asks the session to stop. Up, Down and Home are no
// buttons, as a row of all the keys would take two lines of an 80-column window.
func (r RunScreen) pressed(msg tea.KeyPressMsg) (RunScreen, tea.Cmd) {
	var place int
	r.footer, place = r.footer.Update(msg)
	page := max(r.output.Size.Height, 1)
	switch {
	case place == pageUpPlace:
		r.output = r.output.Move(page)
	case place == pageDownPlace:
		r.output = r.output.Move(-page)
	case place == bottomPlace:
		r.output = r.output.Move(-len(r.output.Rows))
	case place == stopButtonPlace:
		return r, func() tea.Msg { return StopPressedMsg{} }
	case key.Matches(msg, widget.Keys.Up):
		r.output = r.output.Move(1)
	case key.Matches(msg, widget.Keys.Down):
		r.output = r.output.Move(-1)
	case key.Matches(msg, widget.Keys.Top):
		r.output = r.output.Move(len(r.output.Rows))
	}
	return r, nil
}

// Layout is the screen with its panes empty, as what fills them depends on their sizes. The window
// holds the focus alone, so that a window too short for two panes shows it, under the current
// step's name.
func (r RunScreen) Layout() widget.Layout {
	title := "Output"
	if at := r.steps.At(); at >= 0 {
		title = r.steps[at].Label()
		if r.halted {
			title = "Stopping after " + title
		}
	}
	return widget.Layout{
		Title:  r.title,
		Status: r.status(),
		Panes:  []widget.Pane{{Title: "Steps"}, {Title: title, Weight: 2, Focused: true}},
		Footer: r.footer,
	}
}

// View draws the screen.
func (r RunScreen) View() tea.View {
	l := r.Layout()
	sizes := l.PaneSizes(r.window.Width, r.window.Height)
	l.Panes[0].Body = r.steps.View(r.theme, sizes[0].Height)
	l.Panes[1].Body = r.output.View()
	return l.View(r.theme, r.window.Width, r.window.Height)
}

// status is the title's right side: the steps ended, their bar where it fits, and the elapsed time.
func (r RunScreen) status() string {
	count := fmt.Sprintf("%d/%d", r.steps.Ended(), len(r.steps))
	clock := elapsed(r.now.Sub(r.started))
	room := r.window.Width - lipgloss.Width(r.title) - lipgloss.Width(count) - lipgloss.Width(clock) - statusGaps
	if room < barLeast {
		return count + " " + clock
	}
	return count + " " + bar(r.steps.Ended(), len(r.steps), min(room, barMost)) + " " + clock
}

// bar draws done of total in width columns. Its two shades hold under NO_COLOR.
func bar(done, total, width int) string {
	filled := 0
	if total > 0 {
		filled = width * done / total
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// elapsed writes a time as a clock does: a first run takes 20 to 40 minutes, and a server's first
// download can pass the hour.
func elapsed(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
