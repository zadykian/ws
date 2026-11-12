package tui

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

// unwindBound bounds the wait for a work's cleanup, short as a second Ctrl-C asked to leave.
const unwindBound = 5 * time.Second

var (
	// errLeft is the error of a screen that quit by itself, with the work unfinished.
	errLeft = errors.New("the screen ended before the run")
	// errPanic marks a panic of the work, which a cut must not hide.
	errPanic = errors.New("panic")
)

// Config is what Run takes of the process.
type Config struct {
	// In gives the keys, and Out takes the screen and what stays by hand.
	In  io.Reader
	Out io.Writer
	// Theme is what the questions draw with.
	Theme Theme
	// Cut returns the error of a run that a signal ended. Ctrl-C counts as SIGINT. A command gives
	// the error that its plain run ends with, so that its exit status stays.
	Cut func(syscall.Signal) error
	// Unwind bounds the wait for a work past the screen's end, and defaults to unwindBound.
	Unwind time.Duration
	// Log takes every line of the run, and LogPath names it for the report past the screen. A nil
	// Log keeps none.
	Log     io.Writer
	LogPath string
	// Text is where a run's stdout goes past the screen, once a run that passed has ended. Nil drops
	// it. A run logs it either way.
	Text io.Writer
	// Machine is what the screen's works run on, as Bridge.Effects gives it. Nil leaves them none.
	Machine effect.Machine
	// quiet leaves nothing past the screen, for a screen that a run follows and says the same again.
	// It keeps Ctrl-C from asking too, as that screen's scan stops with ctx.
	quiet bool
}

// Work is what a run does. It takes b as its effects, and sends the screen its messages by them.
type Work func(ctx context.Context, b *Bridge) error

// Run draws screen on c.Out, runs work in a goroutine beside it, and returns once the terminal is
// back. That is after the work, or once Ctrl-C or a signal ended ctx and then came again, past a
// command that holds the terminal.
// A screen that ends first ends ctx, and Run gives the work c.Unwind. It returns the work's error,
// or Cut's where a cut run failed, for execute to report. What stays by hand prints on the normal
// screen before Run returns. A panic of the work is an error too.
func Run(ctx context.Context, c Config, screen tea.Model, work Work) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Bubble Tea logs each key read to the file that this names, a hidden field's too.
	_ = os.Unsetenv("TEA_TRACE")
	bridge := newBridge(c.Out)
	bridge.text = &runText{}
	bridge.effects = bridge.effectsOn(ctx, c.Machine)
	bridge.stops = bridge.stop(ctx)
	result := make(chan error, 1)
	begin := tea.Batch(
		func() tea.Msg {
			err := guarded(ctx, bridge, work)
			bridge.Log(bridge.text.String()) // A failed run prints it too.
			result <- err
			return doneMsg{err}
		},
		func() tea.Msg { <-ctx.Done(); return endedMsg{} },
	)
	stop := func() tea.Msg { cancel(); return nil }
	model := session{screen: screen, theme: c.Theme, begin: begin, stop: stop, cut: c.Cut, took: &bridge.took}
	if !c.quiet {
		model.halt = func() tea.Msg { bridge.halted.Store(true); return nil }
	}
	terminal := &handover{}
	// Signals stay ws's: Bubble Tea's handler would end the screen before the work had stopped.
	program := tea.NewProgram(model, tea.WithInput(c.In), tea.WithOutput(c.Out),
		tea.WithoutSignalHandler(), tea.WithFilter(terminal.filter))
	terminal.release, terminal.restore = program.ReleaseTerminal, program.RestoreTerminal
	bridge.rec.log = c.Log
	bridge.send = func(m tea.Msg) { bridge.rec.note(m); program.Send(m) }
	bridge.suspend = func() error { return bridge.pass(true, errGone) }
	bridge.resume = func() error { return bridge.pass(false, nil) }
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go forward(signals, program, bridge.gone)
	final, broke := program.Run()
	close(bridge.gone)
	cancel()
	late := unwound(result, signals, cmp.Or(c.Unwind, unwindBound))
	bridge.rec.close()
	s, _ := final.(session) // A panic in Update or View leaves no model.
	s.err = cmp.Or(late, s.err)
	if c.quiet { // Nothing ran, so a cut past the work's end is no run stopped at its end.
		return cmp.Or(ending(s, broke), s.cutBy)
	}
	bridge.flush()
	bridge.rec.report(c.Out, c.LogPath, s.cutBy != nil || !s.done)
	err := ending(s, broke)
	if err == nil { // A work left behind may still write to the text.
		bridge.text.writeTo(c.Text)
	}
	return err
}

// unwound waits for a work that ctx has ended, and returns its error. The work's defers hold a
// run's cleanup, and execute's exit would skip them. A work past the bound stays behind, and so
// does one that a further signal leaves, which is how the normal screen's Ctrl-C ends the wait.
func unwound(result <-chan error, signals <-chan os.Signal, bound time.Duration) error {
	select {
	case err := <-result:
		return err
	case <-signals:
	case <-time.After(bound):
	}
	return nil
}

// forward sends each signal to the model until gone closes.
func forward(signals <-chan os.Signal, program *tea.Program, gone <-chan struct{}) {
	for {
		select {
		case got := <-signals:
			//nolint:forcetypeassert // Notify's signals are syscall.Signal
			program.Send(signalMsg{got.(syscall.Signal)})
		case <-gone:
			return
		}
	}
}

// ending is the error a run ends with, broke being that of the screen itself. Past a cut, a step's
// failure is the cut's work, and so is a screen that broke as it closed.
func ending(s session, broke error) error {
	switch {
	case errors.Is(s.err, errPanic):
		return s.err
	case !s.done:
		return cmp.Or(s.cutBy, broke, errLeft)
	case s.err != nil && s.cutBy != nil:
		return s.cutBy
	}
	return s.err
}

// guarded runs work, and turns its panic into an error, so that the terminal comes back and the
// stack prints on the normal screen. A panic left alone in this goroutine would leave the terminal
// raw.
func guarded(ctx context.Context, bridge *Bridge, work Work) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v\n%s", errPanic, r, debug.Stack())
		}
	}()
	return work(ctx, bridge)
}
