package tui

import (
	"slices"
	"sync/atomic"
	"syscall"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

const (
	// stopNote is the box of a run that is stopping. A run that heeds ctx never shows it for long. The
	// way out comes first, as a short window cuts the box from the bottom.
	stopNote = "Ctrl-C again leaves it behind.\nThe run ends with its current step."
	// confirmNote asks before a stop, as a press can be a slip and a run can take an hour.
	confirmNote = "Stop the run once its current step ends? Ctrl-C again stops it now."
)

// The places of the buttons that confirm a stop.
const (
	haltPlace = iota
	goOnPlace
)

// The buttons that confirm a stop.
var (
	haltButton = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "stop"))
	goOnButton = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "go on"))
)

// Messages of a run.
type (
	// signalMsg is a signal that Run caught.
	signalMsg struct{ signal syscall.Signal }
	// endedMsg says that the run's ctx ended, whoever ended it.
	endedMsg struct{}
	// doneMsg is the end of a run's work.
	doneMsg struct{ err error }
	// StopPressedMsg is a screen's stop button, pressed by Enter: the key Ctrl-C reaches no screen.
	StopPressedMsg struct{}
	// HaltedMsg tells the screen that the run ends with its current step.
	HaltedMsg struct{}
)

// session is a run's model: its screen, which takes every message that the session doesn't, with
// the run's questions drawn over it. It leaves when the work ends, so that a run's steps decide the
// end, and a screen never quits by itself.
type session struct {
	screen tea.Model
	theme  widget.Theme
	window widget.Size
	// begin starts the work, and stop ends the run's ctx. cut names a signal's error. halt ends the
	// run past its current step, and nil says that it can't.
	begin tea.Cmd
	stop  tea.Cmd
	halt  tea.Cmd
	// took is set once the work took Bridge.Options: only then does it read the stop between its
	// steps, and Ctrl-C ask first. Nil reads as unset.
	took *atomic.Bool
	cut  func(syscall.Signal) error
	// asked is the questions waiting, the first of them on the screen in field.
	asked []prompt
	field field
	// confirming is set while the box in confirm asks before a stop, and halted once the stop is
	// confirmed.
	confirming bool
	confirm    widget.Footer
	halted     bool
	// stopping is set once ctx ended. cutBy is set once a signal or Ctrl-C ended ctx, or halted the
	// run, so that a run stopped short ends with the status of a cut one.
	stopping bool
	cutBy    error
	// done and err are the end of the work.
	done bool
	err  error
}

// Init starts the screen and the work.
func (s session) Init() tea.Cmd {
	return tea.Batch(s.screen.Init(), s.begin)
}

// Update takes what is a run's, and hands the rest to the screen. While a question is up, what the
// keyboard sends goes to its field alone, so that no screen sees a secret.
func (s session) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case askMsg:
		s.asked = append(slices.Clip(s.asked), msg.prompt)
		if len(s.asked) == 1 {
			s.field = newField(msg.prompt, s.window)
		}
		return s, nil
	case retractMsg:
		return s.drop(msg.id), nil
	case signalMsg:
		return s.interrupt(msg.signal)
	case endedMsg:
		s.stopping, s.confirming = true, false
		return s, nil
	case doneMsg:
		s.done, s.err = true, msg.err
		return s, tea.Quit
	case StopPressedMsg:
		return s.pressed()
	case tea.KeyPressMsg:
		if key.Matches(msg, widget.Keys.Interrupt) {
			return s.pressed()
		}
		if len(s.asked) > 0 {
			return s.typed(msg)
		}
		if s.confirming {
			return s.confirmed(msg)
		}
	case tea.PasteMsg:
		if len(s.asked) > 0 {
			return s.typed(msg)
		}
	case tea.WindowSizeMsg:
		s.window = widget.Size{Width: msg.Width, Height: msg.Height}
		s.field = s.field.fit(s.window)
	}
	var cmd tea.Cmd
	s.screen, cmd = s.screen.Update(msg)
	return s, cmd
}

// typed hands the field a key or a paste, and acts on the button it pressed.
func (s session) typed(msg tea.Msg) (tea.Model, tea.Cmd) {
	var place int
	s.field, place = s.field.update(msg)
	switch place {
	case stopPlace:
		return s.interrupt(syscall.SIGINT)
	case answerPlace:
		var answer string
		var ok bool
		if s.field, answer, ok = s.field.answer(); !ok {
			return s, nil
		}
		reply := s.field.prompt.reply
		return s.drop(s.field.prompt.id), func() tea.Msg {
			reply <- answer
			return nil
		}
	}
	return s, nil
}

// drop takes the question id out of the queue, and the next one up where id was on the screen. The
// field goes with its question, and what was typed in it.
func (s session) drop(id uint64) session {
	first := len(s.asked) > 0 && s.asked[0].id == id
	s.asked = slices.DeleteFunc(slices.Clone(s.asked), func(p prompt) bool { return p.id == id })
	switch {
	case first && len(s.asked) > 0:
		s.field = newField(s.asked[0], s.window)
	case first:
		s.field = field{}
	}
	return s
}

// pressed takes Ctrl-C, which raw mode keeps from being a signal, or a screen's stop button. A run
// whose work took Bridge.Options asks first. Under a question it can't: a step waits on the answer,
// so that ctx alone ends it.
func (s session) pressed() (tea.Model, tea.Cmd) {
	if s.halt == nil || s.took == nil || !s.took.Load() || s.confirming || s.halted || s.stopping || len(s.asked) > 0 {
		return s.interrupt(syscall.SIGINT)
	}
	s.confirming, s.confirm = true, widget.NewFooter(haltButton, goOnButton)
	return s, nil
}

// confirmed takes a key under the box that asks before a stop. The screen hears of a stop, as the
// run goes on under it and the maintainer can still read its output.
func (s session) confirmed(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var place int
	s.confirm, place = s.confirm.Update(msg)
	switch place {
	case goOnPlace:
		s.confirming = false
	case haltPlace:
		s.confirming, s.halted = false, true
		s.cutBy = s.cut(syscall.SIGINT)
		var cmd tea.Cmd
		s.screen, cmd = s.screen.Update(HaltedMsg{})
		return s, tea.Batch(s.halt, cmd)
	}
	return s, nil
}

// interrupt takes a signal, which nobody confirms, or a press past the confirmation. The first ends
// ctx, and the screen stays until the work has seen that. The next leaves, as a step can ignore ctx.
func (s session) interrupt(signal syscall.Signal) (tea.Model, tea.Cmd) {
	if s.stopping && s.cutBy != nil {
		return s, tea.Quit
	}
	s.cutBy = s.cut(signal)
	s.stopping, s.confirming = true, false
	return s, s.stop
}

// View draws the screen, and over it the question, the box that asks before a stop or the note that
// the run is stopping. It holds to the alternate buffer whatever the screen says, as the end of a run restores the terminal by it.
func (s session) View() tea.View {
	view := s.screen.View()
	view.AltScreen = true
	switch {
	case len(s.asked) > 0:
		view.Content = over(view.Content, s.field.render(s.theme, s.window), s.window)
	case s.confirming:
		body := wrapped(confirmNote, s.window) + "\n" + s.confirm.Render(s.theme, min(s.window.Width, widget.BoxWidth)-2)
		view.Content = over(view.Content, widget.Box(s.theme, "Stop", body, s.window), s.window)
	case s.stopping:
		view.Content = over(view.Content, widget.Box(s.theme, "Stopping", wrapped(stopNote, s.window), s.window), s.window)
	}
	return view
}

// wrapped fits words to the inside of a box, which would cut them.
func wrapped(words string, window widget.Size) string {
	return lipgloss.Wrap(words, max(min(window.Width, widget.BoxWidth)-2, 1), "")
}

// over draws a box in the middle of a window, on what base shows there.
func over(base, drawn string, window widget.Size) string {
	if drawn == "" {
		return base
	}
	x := (window.Width - lipgloss.Width(drawn)) / 2
	y := (window.Height - lipgloss.Height(drawn)) / 2
	layers := lipgloss.NewCompositor(lipgloss.NewLayer(base), lipgloss.NewLayer(drawn).X(x).Y(y).Z(1))
	return lipgloss.NewCanvas(window.Width, window.Height).Compose(layers).Render()
}
