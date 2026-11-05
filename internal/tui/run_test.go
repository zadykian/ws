package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"syscall"
	"testing"

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
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(context.Context, *Bridge) error {
		<-release
		return nil
	})
	if !errors.Is(err, cutError{syscall.SIGINT}) {
		t.Errorf("Run: %v", err)
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

func TestEnding(t *testing.T) {
	cut, failed := cutError{syscall.SIGINT}, errors.New("a step failed")
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
