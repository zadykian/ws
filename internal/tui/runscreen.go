package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/step"
)

const (
	// barLeast is the columns under which a bar says nothing, and barMost is as wide as one needs.
	barLeast = 4
	barMost  = 24
	// statusGaps is the columns around the title and the status's parts.
	statusGaps = 5
)

// tickMsg is the clock, for the elapsed time. It carries the time, so that Update reads no clock.
type tickMsg struct{ at time.Time }

// RunScreen follows a run: its steps by role with the current one marked, a window on what the run
// prints, a bar of the steps ended and the elapsed time. Bridge.Observe and effect.Out send it what
// it shows. The keys scroll the window, as the steps list follows the run by itself.
type RunScreen struct {
	title   string
	theme   Theme
	steps   stepList
	output  outputPane
	footer  Footer
	window  Size
	started time.Time
	now     time.Time
}

// NewRunScreen returns the screen of a run of groups, which name every step before the run starts.
func NewRunScreen(title string, theme Theme, groups ...Group) RunScreen {
	return RunScreen{
		title:  title,
		theme:  theme,
		steps:  newStepList(groups),
		output: outputPane{theme: theme},
		footer: NewFooter(Keys.PageUp, Keys.PageDown, Keys.Bottom),
	}
}

// Init starts the clock.
func (r RunScreen) Init() tea.Cmd {
	return func() tea.Msg { return tickMsg{time.Now()} }
}

// Update takes the run's events and lines, the clock, the window's size and the keys.
func (r RunScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case step.Event:
		r.steps = r.steps.take(msg)
	case Line:
		r.output = r.output.add(msg)
	case tickMsg:
		if r.started.IsZero() {
			r.started = msg.at
		}
		r.now = msg.at
		return r, tea.Tick(time.Second, func(at time.Time) tea.Msg { return tickMsg{at} })
	case tea.WindowSizeMsg:
		r.window = Size{msg.Width, msg.Height}
		r.output = r.output.fit(r.layout().PaneSizes(msg.Width, msg.Height)[1])
	}
	return r, nil
}

// layout is the screen with its panes empty, as what fills them depends on their sizes. The window
// holds the focus alone, so that a window too short for two panes shows it, under the current
// step's name.
func (r RunScreen) layout() Layout {
	title := "Output"
	if at := r.steps.at(); at >= 0 {
		title = r.steps[at].label()
	}
	return Layout{
		Title:  r.title,
		Status: r.status(),
		Panes:  []Pane{{Title: "Steps"}, {Title: title, Weight: 2, Focused: true}},
		Footer: r.footer,
	}
}

// View draws the screen.
func (r RunScreen) View() tea.View {
	l := r.layout()
	sizes := l.PaneSizes(r.window.Width, r.window.Height)
	l.Panes[0].Body = r.steps.view(r.theme, sizes[0].Height)
	l.Panes[1].Body = r.output.view()
	return l.View(r.theme, r.window.Width, r.window.Height)
}

// status is the title's right side: the steps ended, their bar where it fits, and the elapsed time.
func (r RunScreen) status() string {
	count := fmt.Sprintf("%d/%d", r.steps.ended(), len(r.steps))
	clock := elapsed(r.now.Sub(r.started))
	room := r.window.Width - lipgloss.Width(r.title) - lipgloss.Width(count) - lipgloss.Width(clock) - statusGaps
	if room < barLeast {
		return count + " " + clock
	}
	return count + " " + bar(r.steps.ended(), len(r.steps), min(room, barMost)) + " " + clock
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
