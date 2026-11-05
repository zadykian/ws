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
	"github.com/zadykian/ws/step"
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

// Messages of a question. They hold no answer: see Bridge.ask.
type (
	askMsg     struct{ prompt prompt }
	retractMsg struct{ id uint64 }
)

// Bridge puts effect.Prompt, effect.Hidden, effect.Out and step.Observer behind a screen. A run's
// goroutine calls it, and each call reaches the model as a message: Ask and AskHidden then wait for
// the reply, and the others go on. A Line sent past the screen's end is dropped.
type Bridge struct {
	send func(tea.Msg)
	// gone closes as the screen ends: no reply comes past it.
	gone chan struct{}
	// last is the id of the last question.
	last atomic.Uint64
	// suspend and resume take the terminal from the screen and give it back, as Run sets them. Nil
	// ones leave it where it is.
	suspend, resume func() error
	// hand keeps what ByHand gets, to print once the terminal is back.
	lock sync.Mutex
	hand *effect.Terminal
}

// newBridge returns a bridge that prints what stays by hand on w. Run sets send, as the program's
// model needs the bridge first.
func newBridge(w io.Writer) *Bridge {
	return &Bridge{gone: make(chan struct{}), hand: effect.NewTerminal(strings.NewReader(""), w)}
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
	if err := ctx.Err(); err != nil {
		return "", err
	}
	reply := make(chan string, 1)
	p.id, p.reply = b.last.Add(1), reply
	b.send(askMsg{p})
	select {
	case answer := <-reply:
		return answer, nil
	case <-ctx.Done():
		b.send(retractMsg{p.id})
		return "", ctx.Err()
	case <-b.gone:
		return "", fmt.Errorf("reading the answer to %q: the screen has ended", p.text)
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

// Observe sends a run's event behind what the step's commands printed, with the lines that a plain
// run prints of it. Their words are made here, on the run's goroutine, as a Change's String can
// take a diff's time, and Update would hold the screen for it.
func (b *Bridge) Observe(e step.Event) {
	name := e.Step
	if e.Parent != "" {
		name = e.Parent + ": " + e.Step
	}
	switch {
	case e.Kind == step.Planned && e.Check:
		for _, c := range e.Changes {
			b.send(changeLine(name, c))
		}
	case e.Kind == step.Started:
		b.Say("%s: applying %d change(s)", name, len(e.Changes))
	case e.Kind == step.Skipped && e.After != "":
		b.Say("%s: after %s", name, e.After)
	}
	b.send(e)
}

// changeLine is a change as --check shows it. One of several lines holds a file's diff under its
// first line.
func changeLine(name string, c step.Change) Line {
	text := name + ": " + strings.TrimSuffix(c.String(), "\n")
	if strings.Contains(text, "\n") {
		return Line{Diffed, text}
	}
	return Line{Said, text}
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
