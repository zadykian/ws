package tui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/effect"
)

// LineKind says which call of effect.Out made a Line, as a screen draws each apart.
type LineKind int

// The kinds of Line, by the call of effect.Out.
const (
	Said LineKind = iota
	Warned
	Copied
	Diffed
)

// Line is a line of a run, sent to the screen. Text is what a plain run prints, less the last
// newline, so that a warning reads as one under NO_COLOR and a log can hold the same words.
type Line struct {
	Kind LineKind
	Text string
}

// Bridge puts effect.Out behind a screen. A run's goroutine calls it, and each call reaches the
// model as a message. A Line sent past the screen's end is dropped.
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

// Say sends a line.
func (b *Bridge) Say(format string, a ...any) {
	b.send(Line{Said, fmt.Sprintf(format, a...)})
}

// Raw sends a line to copy.
func (b *Bridge) Raw(format string, a ...any) {
	b.send(Line{Copied, fmt.Sprintf(format, a...)})
}

// Warn sends a line that starts with "warning: ".
func (b *Bridge) Warn(format string, a ...any) {
	b.send(Line{Warned, fmt.Sprintf("warning: "+format, a...)})
}

// Diff sends the unified diff of before and after, and nothing where they are equal.
func (b *Bridge) Diff(name string, before, after []byte) {
	if text := effect.Diff(name, before, after); text != "" {
		b.send(Line{Diffed, strings.TrimSuffix(text, "\n")})
	}
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
