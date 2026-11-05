package tui

import (
	"io"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/effect"
)

// Bridge stands between a run's goroutine and a screen.
type Bridge struct {
	send func(tea.Msg)
	// hand keeps what ByHand gets, to print once the terminal is back.
	lock sync.Mutex
	hand *effect.Terminal
}

// newBridge returns a bridge that prints what stays by hand on w. Run sets send, as the program's
// model needs the bridge first.
func newBridge(w io.Writer) *Bridge {
	return &Bridge{hand: effect.NewTerminal(strings.NewReader(""), w)}
}

// ByHand keeps a line for the end of the run.
func (b *Bridge) ByHand(format string, a ...any) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.hand.ByHand(format, a...)
}

// flush prints what ByHand kept. A run left behind may still call ByHand, hence the lock.
func (b *Bridge) flush() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.hand.Flush()
}
