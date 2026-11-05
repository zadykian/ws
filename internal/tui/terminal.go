package tui

import (
	"errors"

	tea "charm.land/bubbletea/v2"
)

// errGone refuses a command the terminal of a screen that has ended.
var errGone = errors.New("the screen has ended")

// terminalMsg asks the program's loop to give the terminal to a command, or to take it back. reply
// has room for the answer.
type terminalMsg struct {
	away  bool
	reply chan<- error
}

// handover moves the terminal in the program's loop, as Bubble Tea's Exec does. From a run's
// goroutine, a move could follow the program's end, and nothing would undo its raw mode. Exec itself
// would hold the loop, and with it the signal that stops the command.
type handover struct {
	release, restore func() error
}

// filter takes the moves ahead of Update, which stays pure.
func (h *handover) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	if m, ok := msg.(terminalMsg); ok {
		return h.move(m)
	}
	return msg
}

// move does what m asks.
func (h *handover) move(m terminalMsg) tea.Msg {
	if m.away {
		m.reply <- h.release()
		return nil
	}
	m.reply <- h.restore()
	return nil
}

// pass moves the terminal through the program's loop. A loop that has stopped never answers: the
// terminal is back by then, and pass returns ended.
func (b *Bridge) pass(away bool, ended error) error {
	reply := make(chan error, 1)
	b.send(terminalMsg{away, reply})
	select {
	case err := <-reply:
		return err
	case <-b.gone:
		return ended
	}
}
