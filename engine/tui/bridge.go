package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// errHalted ends a run that a confirmed stop ended between two steps.
var errHalted = errors.New("stopped after the current step")

// LineKind says which call of effect.Out made a Line, as a screen draws each apart.
type LineKind int

// The kinds of Line, by where the line came from.
const (
	Said LineKind = iota
	Warned
	Copied
	Diffed
	// Piped is what a step's commands print, which reaches no call of effect.Out.
	Piped
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
	// options make the question a choice among them, and fallback the one it starts on.
	options []string
	valid   func(string) error
	reply   chan<- string
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
	// pipe takes what the machine's commands print.
	pipe *pipe
	// gone closes as the screen ends: no reply comes past it.
	gone chan struct{}
	// last is the id of the last question.
	last atomic.Uint64
	// halted is set once a stop is confirmed, for stop to read on the run's goroutine.
	halted atomic.Bool
	// warned holds the warnings sent so far.
	warned sync.Map
	// suspend and resume take the terminal from the screen and give it back, as Run sets them. Nil
	// ones leave the terminal where it stands.
	suspend, resume func() error
	// said is what a step said last, since it started: a command that talks may need it to read by.
	said atomic.Pointer[string]
	// lasting holds the lines that SayLasting kept for hand.
	lasting sync.Map
	// hand keeps what ByHand gets, to print once the terminal is back.
	lock sync.Mutex
	hand *effect.Terminal
	// stops is the run's stop, which Options hands to its steps. Run sets it before the work starts.
	stops func() error
	// took is set once a work took Options: the first Ctrl-C then asks before it stops the run.
	took atomic.Bool
	// text holds what the work wrote to Effects.Text.
	text *runText
	// effects are what Effects gives the screen's works. Run sets them before the work starts.
	effects Effects
	// rec keeps the run for its log and its end.
	rec record
}

// newBridge returns a bridge that prints what stays by hand on w. Run sets send, as the program's
// model needs the bridge first.
func newBridge(w io.Writer) *Bridge {
	b := &Bridge{gone: make(chan struct{}), hand: effect.NewTerminal(strings.NewReader(""), w)}
	b.pipe = &pipe{bridge: b}
	return b
}

// Ask draws q over the screen and waits for an answer that q.Valid takes, or for ctx's end.
func (b *Bridge) Ask(ctx context.Context, q effect.Question) (string, error) {
	return b.ask(ctx, prompt{text: q.Text, fallback: q.Default, valid: q.Valid})
}

// Choose draws q over the screen as a choice, and waits for the option picked, or for ctx's end.
func (b *Bridge) Choose(ctx context.Context, q effect.ChoiceQuestion) (string, error) {
	if len(q.Options) == 0 {
		return "", fmt.Errorf("choosing %q: no options", q.Text)
	}
	return b.ask(ctx, prompt{text: q.Text, fallback: q.Default, options: slices.Clone(q.Options)})
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
	text := fmt.Sprintf(format, a...)
	b.said.Store(&text)
	b.send(Line{Said, text})
}

// Raw sends a line to copy.
func (b *Bridge) Raw(format string, a ...any) {
	b.send(Line{Copied, fmt.Sprintf(format, a...)})
}

// Warn sends a line that starts with "warning: ", once, as a step plans again after it applies. It
// stays past the screen: a step that warns passes, and the line is all that says what it gave up.
func (b *Bridge) Warn(format string, a ...any) {
	line := fmt.Sprintf("warning: "+format, a...)
	if _, sent := b.warned.LoadOrStore(line, true); !sent {
		l := Line{Warned, line}
		b.rec.keep(l)
		b.send(l)
	}
}

// SayLasting sends a line, and keeps it with what stays by hand, once, as a step plans again after it
// applies. The window goes with the screen, and the maintainer acts on this line past it.
func (b *Bridge) SayLasting(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	b.send(Line{Said, line})
	if _, kept := b.lasting.LoadOrStore(line, true); !kept {
		b.keepByHand("%s", line)
	}
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
	b.pipe.flush()
	name := e.Step
	if e.Parent != "" {
		name = e.Parent + ": " + e.Step
	}
	switch {
	case e.Kind == step.Skipped && e.Err != nil:
		b.Warn("left out: %v", e.Err)
	case e.LeftOut:
	case e.Kind == step.Planned && e.Check:
		for _, c := range e.Changes {
			l := changeLine(name, c)
			b.rec.keep(l)
			b.send(l)
		}
	case e.Kind == step.Started:
		b.said.Store(nil)
		b.send(Line{Said, fmt.Sprintf("%s: applying %d change(s)", name, len(e.Changes))})
	case e.Kind == step.Skipped && e.After != "":
		b.Say("%s: after %s", name, e.After)
	}
	b.send(e)
}

// changeLine is a change as --check shows it, under the name of its step, which a wizard's stage
// writes in the change itself. One of several lines holds a file's diff under its first line. A
// line reaches the log, so it holds the excerpt.
func changeLine(name string, c step.Change) Line {
	text := strings.TrimSuffix(step.ExcerptOf(c), "\n")
	if !strings.HasPrefix(text, name+": ") {
		text = name + ": " + text
	}
	if strings.Contains(text, "\n") {
		return Line{Diffed, text}
	}
	return Line{Said, text}
}

// stop is a run's step.Options.Stop, which Options hands out. It ends the run before its next step, past a confirmed stop or
// ctx's end, as a step that ignores ctx would let the next one start.
func (b *Bridge) stop(ctx context.Context) func() error {
	return func() error {
		if b.halted.Load() {
			return errHalted
		}
		return ctx.Err()
	}
}

// Options returns o with the screen's observer and stop, which a work gives its steps. A caller that
// wraps the observer sets it after. The first call marks the run as one whose Ctrl-C asks first, as
// its steps read the stop between them.
func (b *Bridge) Options(o step.Options) step.Options {
	if b == nil {
		return o
	}
	b.took.Store(true)
	o.Observer, o.Stop = b, b.stops
	return o
}

// ByHand keeps a line for the end of the run, and logs it, as a plain run prints it.
func (b *Bridge) ByHand(format string, a ...any) {
	b.rec.logText(fmt.Sprintf(format, a...))
	b.keepByHand(format, a...)
}

// keepByHand keeps a line for the end of the run, for a line the log holds already.
func (b *Bridge) keepByHand(format string, a ...any) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.hand.ByHand(format, a...)
}

// Log writes text to the run's log alone, as a plain run would print it on stdout: the window never
// shows it. A work calls it at its end, for the text that waits for the screen's end.
func (b *Bridge) Log(text string) {
	b.rec.logText(text)
}

// directions prints what a step said last on the terminal that the screen left, once. A command
// that talks takes over at once, and the pane that shows the line stands behind it.
func (b *Bridge) directions() {
	if text := b.said.Swap(nil); text != nil {
		b.lock.Lock()
		defer b.lock.Unlock()
		b.hand.Say("%s", *text)
	}
}

// flush prints what ByHand kept. A run left behind may still call ByHand, hence the lock.
func (b *Bridge) flush() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.hand.Flush()
}
