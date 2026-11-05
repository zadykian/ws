package tui

import (
	"slices"
	"syscall"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Messages of a run.
type (
	// signalMsg is a signal that Run caught.
	signalMsg struct{ signal syscall.Signal }
	// doneMsg is the end of a run's work.
	doneMsg struct{ err error }
)

// session is a run's model: its screen, which takes every message that the session doesn't, with
// the run's questions drawn over it. It leaves when the work ends, so that a run's steps decide the
// end, and a screen never quits by itself.
type session struct {
	screen tea.Model
	theme  Theme
	window Size
	// begin starts the work, and stop ends the run's ctx. cut names a signal's error.
	begin tea.Cmd
	stop  tea.Cmd
	cut   func(syscall.Signal) error
	// asked is the questions waiting, the first of them on the screen in field.
	asked []prompt
	field field
	// cutBy is set once a signal or Ctrl-C ended ctx.
	cutBy error
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
	case doneMsg:
		s.done, s.err = true, msg.err
		return s, tea.Quit
	case tea.KeyPressMsg:
		if key.Matches(msg, Keys.Interrupt) {
			return s.interrupt(syscall.SIGINT)
		}
		if len(s.asked) > 0 {
			return s.typed(msg)
		}
	case tea.PasteMsg:
		if len(s.asked) > 0 {
			return s.typed(msg)
		}
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
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

// drop takes the question id out of the queue, and the next one up where it was on the screen. The
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

// interrupt takes a signal, or Ctrl-C, which raw mode keeps from being one. The first ends ctx, and
// the screen stays until the work has seen that. The next leaves, as a step can ignore ctx.
func (s session) interrupt(signal syscall.Signal) (tea.Model, tea.Cmd) {
	if s.cutBy != nil {
		return s, tea.Quit
	}
	s.cutBy = s.cut(signal)
	return s, s.stop
}

// View draws the screen, and over it the question. It holds to the alternate buffer whatever the
// screen says, as the end of a run restores the terminal by it.
func (s session) View() tea.View {
	view := s.screen.View()
	view.AltScreen = true
	if len(s.asked) > 0 {
		view.Content = over(view.Content, s.field.render(s.theme, s.window), s.window)
	}
	return view
}

// over draws a box in the middle of a window, on what base shows there.
func over(base, drawn string, window Size) string {
	if drawn == "" {
		return base
	}
	x := (window.Width - lipgloss.Width(drawn)) / 2
	y := (window.Height - lipgloss.Height(drawn)) / 2
	layers := lipgloss.NewCompositor(lipgloss.NewLayer(base), lipgloss.NewLayer(drawn).X(x).Y(y).Z(1))
	return lipgloss.NewCanvas(window.Width, window.Height).Compose(layers).Render()
}
