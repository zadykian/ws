package tui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

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

// prompt is a question on its way to the screen.
type prompt struct {
	id       uint64
	text     string
	fallback string
	hidden   bool
	valid    func(string) error
	reply    chan<- string
}

// askMsg is a question for the screen. It holds no answer: see Bridge.ask.
type askMsg struct{ prompt prompt }

// Bridge puts effect.Prompt, effect.Hidden and effect.Out behind a screen. A run's goroutine calls
// it, and each call reaches the model as a message: Ask and AskHidden then wait for the reply, and
// the others go on. A Line sent past the screen's end is dropped.
type Bridge struct {
	send func(tea.Msg)
	// last is the id of the last question.
	last atomic.Uint64
	// hand keeps what ByHand gets, to print once the terminal is back.
	lock sync.Mutex
	hand *effect.Terminal
}

// newBridge returns a bridge that prints what stays by hand on w. Run sets send, as the program's
// model needs the bridge first.
func newBridge(w io.Writer) *Bridge {
	return &Bridge{hand: effect.NewTerminal(strings.NewReader(""), w)}
}

// Ask draws q over the screen and waits for an answer that q.Valid takes, or for ctx's end.
func (b *Bridge) Ask(ctx context.Context, q effect.Question) (string, error) {
	return b.ask(ctx, prompt{text: q.Text, fallback: q.Default, valid: q.Valid})
}

// AskHidden draws text over the screen, in a field that shows nothing of what is typed. It returns
// the line as typed, blanks included.
func (b *Bridge) AskHidden(ctx context.Context, text string) (string, error) {
	return b.ask(ctx, prompt{text: text, hidden: true})
}

// ask sends p and waits. The answer comes back on a channel of p's own and never in a message, so
// that no message's printed form holds a secret.
func (b *Bridge) ask(ctx context.Context, p prompt) (string, error) {
	reply := make(chan string, 1)
	p.id, p.reply = b.last.Add(1), reply
	b.send(askMsg{p})
	select {
	case answer := <-reply:
		return answer, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
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
