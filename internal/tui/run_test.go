package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// leaveBuffer is what puts a terminal back on its normal screen.
const leaveBuffer = "\x1b[?1049l"

// noKeys is a terminal that nobody types on.
func noKeys(out io.Writer) Config {
	return Config{In: strings.NewReader(""), Out: out, Theme: testTheme, Cut: testCut}
}

// waitForCtx is a work that heeds its context.
func waitForCtx(ctx context.Context, _ *Bridge) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestRunGivesTheScreenTheLinesAndReturnsTheError(t *testing.T) {
	failed := errors.New("a step failed")
	var got []tea.Msg
	var out bytes.Buffer
	err := Run(t.Context(), noKeys(&out), recorder{&got}, func(_ context.Context, b *Bridge) error {
		b.Say("a line")
		b.ByHand("restart ws")
		return failed
	})
	if !errors.Is(err, failed) {
		t.Errorf("Run: %v", err)
	}
	if !slices.Contains(got, tea.Msg(Line{Said, "a line"})) {
		t.Errorf("the screen got %v", got)
	}
	left := strings.LastIndex(out.String(), leaveBuffer)
	if left < 0 || !strings.Contains(out.String()[left:], "restart ws") {
		t.Errorf("no line by hand past the screen's end in %q", out.String())
	}
}

func TestRunEndsCtxOnCtrlC(t *testing.T) {
	c := noKeys(io.Discard)
	c.In = strings.NewReader("\x03")
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, waitForCtx)
	if !errors.Is(err, cutError{syscall.SIGINT}) {
		t.Errorf("Run: %v", err)
	}
}

func TestRunLeavesAWorkThatIgnoresCtxOnASecondCtrlC(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	c := noKeys(io.Discard)
	c.In = strings.NewReader("\x03\x03")
	c.Unwind = time.Millisecond
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(context.Context, *Bridge) error {
		<-release
		return nil
	})
	if !errors.Is(err, cutError{syscall.SIGINT}) {
		t.Errorf("Run: %v", err)
	}
}

func TestRunWaitsForAWorkPastItsScreen(t *testing.T) {
	for keys, want := range map[string]error{"\x03\x03": cutError{syscall.SIGINT}, "": syscall.EIO} {
		in, typing := io.Pipe()
		c := noKeys(io.Discard)
		c.In = in
		var cleaned atomic.Bool
		err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(ctx context.Context, b *Bridge) error {
			_, _ = io.WriteString(typing, keys)
			_ = typing.CloseWithError(syscall.EIO)
			<-ctx.Done()
			<-b.gone
			cleaned.Store(true)
			return nil
		})
		if !errors.Is(err, want) || !cleaned.Load() {
			t.Errorf("keys %q: Run = %v, past the cleanup: %v", keys, err, cleaned.Load())
		}
	}
}

func TestUnwoundGivesUpOnAFurtherSignal(t *testing.T) {
	signals := make(chan os.Signal, 1)
	signals <- syscall.SIGINT
	if err := unwound(nil, signals, time.Hour); err != nil {
		t.Errorf("unwound: %v", err)
	}
}

func TestRunEndsCtxOnASignal(t *testing.T) {
	err := Run(t.Context(), noKeys(io.Discard), recorder{&[]tea.Msg{}},
		func(ctx context.Context, b *Bridge) error {
			if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
				return err
			}
			return waitForCtx(ctx, b)
		})
	if !errors.Is(err, cutError{syscall.SIGTERM}) {
		t.Errorf("Run: %v", err)
	}
}

func TestRunTurnsAPanicOfTheWorkIntoAnError(t *testing.T) {
	var out bytes.Buffer
	err := Run(t.Context(), noKeys(&out), recorder{&[]tea.Msg{}},
		func(context.Context, *Bridge) error { panic("a bug") })
	if !errors.Is(err, errPanic) || !strings.Contains(err.Error(), "a bug") ||
		!strings.Contains(err.Error(), "run_test.go") {
		t.Errorf("Run: %v, want the value and the stack", err)
	}
	if !strings.Contains(out.String(), leaveBuffer) {
		t.Errorf("the terminal stayed on the screen: %q", out.String())
	}
}

func TestRunReportsAPanicPastItsScreen(t *testing.T) {
	c := noKeys(io.Discard)
	c.In = strings.NewReader("\x03\x03")
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(_ context.Context, b *Bridge) error {
		<-b.gone
		panic("a bug")
	})
	if !errors.Is(err, errPanic) {
		t.Errorf("Run: %v", err)
	}
}

// fragile is a screen that panics on a Line.
type fragile struct{ recorder }

func (f fragile) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(Line); ok {
		panic("a bug")
	}
	return f, nil
}

// firstWrite is a terminal that says when a frame has reached it.
type firstWrite struct {
	bytes.Buffer
	once  sync.Once
	drawn chan struct{}
}

func (f *firstWrite) Write(p []byte) (int, error) {
	f.once.Do(func() { close(f.drawn) })
	return f.Buffer.Write(p)
}

func TestRunRestoresTheTerminalAfterAPanicOfTheScreen(t *testing.T) {
	out := firstWrite{drawn: make(chan struct{})}
	err := Run(t.Context(), noKeys(&out), fragile{recorder{&[]tea.Msg{}}},
		func(ctx context.Context, b *Bridge) error {
			b.send(tea.WindowSizeMsg{Width: 80, Height: 24})
			<-out.drawn
			b.Say("a line")
			return waitForCtx(ctx, b)
		})
	if !errors.Is(err, tea.ErrProgramPanic) {
		t.Errorf("Run: %v", err)
	}
	if !strings.Contains(out.String(), leaveBuffer) {
		t.Errorf("the terminal stayed on the screen: %q", out.String())
	}
}

// quitter is a screen that quits by itself.
type quitter struct{ recorder }

func (quitter) Init() tea.Cmd { return tea.Quit }

func TestRunRefusesAScreenThatQuitsByItself(t *testing.T) {
	err := Run(t.Context(), noKeys(io.Discard), quitter{recorder{&[]tea.Msg{}}}, waitForCtx)
	if !errors.Is(err, errLeft) {
		t.Errorf("Run: %v", err)
	}
}

func TestRunKeepsAHiddenAnswerOffEveryRecord(t *testing.T) {
	trace := filepath.Join(t.TempDir(), "keys")
	t.Setenv("TEA_TRACE", trace)
	keys, typing := io.Pipe()
	defer func() { _ = typing.Close() }()
	var out bytes.Buffer
	c := noKeys(&out)
	c.In = keys
	var answer string
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(ctx context.Context, b *Bridge) error {
		send := b.send
		b.send = func(msg tea.Msg) {
			send(msg)
			if _, asked := msg.(askMsg); asked {
				go func() { _, _ = io.WriteString(typing, "s3cret\r") }()
			}
		}
		b.send(tea.WindowSizeMsg{Width: 80, Height: 24})
		var err error
		answer, err = b.AskHidden(ctx, "Password")
		return err
	})
	if err != nil || answer != "s3cret" {
		t.Fatalf("Run: %v, with the answer %q", err, answer)
	}
	if strings.Contains(out.String(), "s3cret") {
		t.Errorf("the secret reached the terminal: %q", out.String())
	}
	if _, err := os.Stat(trace); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Bubble Tea kept a log of the keys: %v", err)
	}
}

func TestEnding(t *testing.T) {
	cut, failed := cutError{syscall.SIGINT}, errors.New("a step failed")
	panicked := fmt.Errorf("%w: a bug", errPanic)
	cases := []struct {
		name  string
		s     session
		broke error
		want  error
	}{
		{"a success", session{done: true}, nil, nil},
		{"a failure", session{done: true, err: failed}, nil, failed},
		{"a failure past a cut", session{done: true, err: failed, cutBy: cut}, nil, cut},
		{"a success past a cut", session{done: true, cutBy: cut}, nil, nil},
		{"a panic past a cut", session{done: true, err: panicked, cutBy: cut}, nil, panicked},
		{"a panic past the screen", session{err: panicked, cutBy: cut}, nil, panicked},
		{"a work left behind", session{cutBy: cut}, nil, cut},
		{"a screen that quit", session{}, nil, errLeft},
		{"a broken screen", session{}, syscall.EIO, syscall.EIO},
		{"a broken screen past a cut", session{cutBy: cut}, syscall.EIO, cut},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ending(c.s, c.broke)
			if !errors.Is(got, c.want) || (c.want == nil) != (got == nil) {
				t.Errorf("ending = %v, want %v", got, c.want)
			}
		})
	}
}
