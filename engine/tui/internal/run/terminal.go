package run

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
	// away holds while a command has the terminal, and quit once a quit waits for that command.
	away, quit bool
}

// filter takes the moves ahead of Update, which stays pure. A quit waits for the command, as the
// program would paint its last frame over the command's prompt. ctx has ended by then, so the wait
// is the command's grace, and one that no kill can end holds the screen too.
func (h *handover) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	switch m := msg.(type) {
	case terminalMsg:
		return h.move(m)
	case tea.QuitMsg:
		if h.away {
			h.quit = true
			return nil
		}
	}
	return msg
}

// move does what m asks, and returns the quit that waited for it.
func (h *handover) move(m terminalMsg) tea.Msg {
	if m.away {
		err := h.release()
		h.away = err == nil
		m.reply <- err
		return nil
	}
	m.reply <- h.restore()
	h.away = false
	if h.quit {
		return tea.QuitMsg{}
	}
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
